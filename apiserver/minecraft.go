package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ammario/mcping"
	"github.com/golang-commonmark/markdown"
	"github.com/shirou/gopsutil/process"
	"golang.org/x/net/context"
)

type mcModZipFilesystem struct {
	ModZipFilesystem
	lock       sync.Mutex
	serverList []string
}

var minecraftModFS = &mcModZipFilesystem{
	ModZipFilesystem: ModZipFilesystem{
		MatchRegex: regexp.MustCompile(`\A/([a-zA-Z0-9-]+)/mods\.zip\z`),
	},
}

func (fs *mcModZipFilesystem) Open(name string) (http.File, error) {
	match := fs.MatchRegex.FindStringSubmatch(name)
	if match == nil {
		return nil, os.ErrNotExist
	}
	valid := false
	fs.lock.Lock()
	for _, v := range fs.serverList {
		m := &mcserverdata{CWD: v}
		if match[1] == m.Name() {
			valid = true
			break
		}
	}
	fs.lock.Unlock()

	if !valid {
		return nil, os.ErrPermission
	}
	return os.Open(fmt.Sprintf("%s%s", fs.BaseDir, name))
}

type propertiesFile map[string]string

func LoadServerPropsFile(file io.Reader) (propertiesFile, error) {
	s := bufio.NewScanner(file)
	result := make(propertiesFile)
	for s.Scan() {
		line := s.Text()
		if line[0:1] == "#" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		result[parts[0]] = parts[1]
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

var ErrProcessExited = errors.New("Process exited while reading the data")
var ErrNotAMinecraftServer = errors.New("not a Minecraft server")
var ErrServerStarting = errors.New("Server starting up... (stage 1)")
var ErrServerStarting2 = errors.New("Server starting up... (stage 2)")

type ErrAsString struct {
	Inner error
}

func (e ErrAsString) MarshalJSON() ([]byte, error) {
	if e.Inner == nil {
		return []byte(`""`), nil
	}
	return json.Marshal(e.Inner.Error())
}

func (e ErrAsString) Error() string {
	return e.Inner.Error()
}

type mcserverdata struct {
	Err   error  `json:"-"`
	Error string `json:"Err"`

	PID      int32
	CWD      string
	MOTD     string
	Port     string
	NewsFile template.HTML
	MapName  string
	HasPack  bool

	PingData  mcping.PingResponse
	PingError ErrAsString
}

func (m *mcserverdata) IsAServer() bool {
	return m.Err != ErrNotAMinecraftServer
}

// IsError returns whether the Err field is filled.
func (m *mcserverdata) IsError() bool {
	return m.Err != nil
}

func (m *mcserverdata) HasPingError() bool {
	return m.PingError.Inner != nil
}

// Name is the name of the server, which is the name of the directory it is run from.
func (m *mcserverdata) Name() string {
	lastSlash := strings.LastIndex(m.CWD, "/")
	if lastSlash == -1 {
		return ""
	}
	return m.CWD[lastSlash+1:]
}

func (m *mcserverdata) IncludeMapName() bool {
	return m.MapName != "world"
}

func (m *mcserverdata) FaviconURL() string {
	return "" //m.PingData.Favicon
}

func (m *mcserverdata) ModsPath() string {
	return fmt.Sprintf("https://home.riking.org/api/minecraftmods/%s/mods.zip", m.Name())
}

func (m *mcserverdata) ServerType() string {
	if m.PingData.Server == "Unknown" {
		return fmt.Sprintf("Minecraft %s", m.PingData.Version)
	}
	return fmt.Sprintf("%s %s", m.PingData.Server, m.PingData.Version)
}

var markdownRenderer = markdown.New(markdown.Breaks(true))

func (m *mcserverdata) readData(ctx context.Context, pid int32, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if err := recover(); err != nil {
			m.Err = err.(error)
		}
	}()
	failOnError := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	m.PID = pid

	javaProc, err := process.NewProcess(m.PID)
	if err != nil {
		failOnError(ErrProcessExited)
	}

	cwd, err := javaProc.Cwd()
	failOnError(err)
	// XXX totally hacky
	if cwd == "/tank/crashplan" {
		failOnError(ErrNotAMinecraftServer)
	}
	m.CWD = cwd

	file, err := os.Open(fmt.Sprintf("%s/server.properties", cwd))
	failOnError(err)
	props, err := LoadServerPropsFile(file)
	failOnError(err)
	_, err = os.Stat(fmt.Sprintf("%s/mods.zip", cwd))
	if err != nil && !os.IsNotExist(err) {
		failOnError(err)
	} else if os.IsNotExist(err) {
		m.HasPack = false
	} else {
		m.HasPack = true
	}

	m.Port = props["server-port"]
	m.MOTD = props["motd"]
	m.MapName = props["level-name"]

	newsFile, err := ioutil.ReadFile(fmt.Sprintf("%s/NEWS.md", cwd))
	if err != nil && !os.IsNotExist(err) {
		failOnError(err)
	} else if err == nil {
		m.NewsFile = template.HTML(markdownRenderer.RenderToString(newsFile))
	}

	pingResponse, err := mcping.PingContext(ctx, fmt.Sprintf("localhost:%s", m.Port))
	if netErr, ok := err.(*net.OpError); ok {
		if _, ok := netErr.Err.(*os.SyscallError); ok {
			m.PingError = ErrAsString{ErrServerStarting}
		} else {
			m.PingError = ErrAsString{netErr}
		}
	} else if _, ok := err.(mcping.ErrSmallPacket); ok {
		m.PingError = ErrAsString{ErrServerStarting2}
	} else if err != nil {
		fmt.Printf("%#v\n", err)
		m.PingError = ErrAsString{err}
	} else {
		m.PingData = pingResponse
	}

	/*
		// Send /who command
		firstBashPid, err := javaProc.Parent()
		failOnError(err)
		firstBash, err := process.NewProcess(firstBashPid)
		failOnError(err)
		firstBashCmdline, err := firstBash.Cmdline()
		failOnError(err)
		if firstBashCmdline != "/bin/bash" {
			failOnError(fmt.Errorf("error: first parent's [pid %d] cmdline is %s, not /bin/bash", firstBashPid, firstBashCmdline))
		}

		secondBashPid, err := firstBash.Parent()
		failOnError(err)
		secondBash, err := process.NewProcess(secondBashPid)
		failOnError(err)
		secondBashCmdline, err := secondBash.Cmdline()
		failOnError(err)
		if secondBashCmdline != "/bin/bash" {
			failOnError(fmt.Errorf("error: second parent's [pid %d] cmdline is %s, not /bin/bash", secondBashPid, secondBashCmdline))
		}

		screenProcPid, err := secondBash.Parent()
		failOnError(err)
		screenProc, err := process.NewProcess(screenProcPid)
		failOnError(err)
		screenCmd, err := screenProc.CmdlineSlice()
	*/

}

func loadMCServersData(ctx context.Context) ([]mcserverdata, error) {
	pids, err := pgrep("java")
	if err != nil {
		return nil, err
	}
	data := make([]mcserverdata, len(pids))

	ctx, cancel := context.WithTimeout(ctx, 2250*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(len(pids))
	for i, pid := range pids {
		go data[i].readData(ctx, pid, &wg)
	}
	wg.Wait()

	// TODO

	return data, nil
}

var minecraftStatusTemplate = template.Must(template.New("minecraftStatus").Parse(`
<table class="table table-bordered table-striped"><thead>
    <th>Server</th>
    <th>Port</th>
    <th>About</th>
    <th>Online</th>
</thead>
{{- range . -}}
{{- if .IsAServer -}}
<tr>
{{- if .IsError -}}
	<td colspan="4"><b>Error</b>: {{ .Err.Error }}
{{- else -}}
	<td class="name">
        {{- if .PingData.Favicon -}}
            <img src="{{.FaviconURL}}" width="64" height="64">
        {{ end -}}
        {{- .Name -}}
    </td>
	<td class="port">
		<span class="connect-hostname">home.riking.org:</span><span class="connect-port">{{ .Port }}</span>
	</td>
    <td class="motd">
	{{- if .NewsFile }}{{ .NewsFile }}{{ end -}}
	{{- if .IncludeMapName }}<p><strong>Map: </strong><em>{{ .MapName }}</em></p>{{ end -}}
	{{- if true }}<p><strong>MOTD: </strong><em>{{.MOTD}}</em></p>{{ end -}}
	{{- if .HasPack }}<p><a href="{{ .ModsPath }}">Download Modpack</a></p>{{ end -}}
    </td>
    <td class="online">
        {{- if .HasPingError -}}
            <p class="has-warning"><span class="control-label">{{ .PingError.Error }}</span></p>
        {{- else -}}
            <p><strong>{{ .PingData.Online }}</strong> players online</p>
            <ul>{{ range .PingData.Sample }}<li>{{ .Name }}</li>{{ end }}</ul>
            <p>{{ .ServerType }}</p>
        {{- end -}}
    </td>
{{- end -}}
</tr>
{{- end}}{{end -}}
</table>
`))

var jsonTemplate = template.Must(template.New("showJson").Parse(`
<details><summary>JSON source</summary><pre><code>{{.}}</code></pre></details>
`))

const includeJsonDump = true

func HTTPMCServers(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var serverInfo []mcserverdata
	if false {
		// TODO load cached info - 1sec max
	} else {
		var err error
		serverInfo, err = loadMCServersData(ctx)
		if err != nil {
			// write info failed to load
			w.(stringWriter).WriteString("<p>ERROR: failed to load server information<br>")
			w.(stringWriter).WriteString(err.Error())
			return
		} else {
			cwds := make([]string, len(serverInfo))
			for i := range serverInfo {
				cwds[i] = serverInfo[i].CWD
			}
			minecraftModFS.lock.Lock()
			minecraftModFS.serverList = cwds
			minecraftModFS.lock.Unlock()
		}
	}

	// Print the table
	err := minecraftStatusTemplate.Execute(w, serverInfo)
	if err != nil {
		w.(stringWriter).WriteString(fmt.Sprintf("<p>ERROR: %s", err))
	}

	if includeJsonDump {
		// Include raw data as a JSON dump
		for i := range serverInfo {
			if serverInfo[i].Err != nil {
				serverInfo[i].Error = serverInfo[i].Err.Error()
			}
		}
		bytes, err := json.MarshalIndent(serverInfo, "", "\t")
		if err != nil {
			w.(stringWriter).WriteString(fmt.Sprintf("<p>ERROR: failed to marshal json: %s", err))
			return
		}
		err = jsonTemplate.Execute(w, string(bytes))
		if err != nil {
			w.(stringWriter).WriteString(fmt.Sprintf("<p>ERROR: %s", err))
		}
	}
}

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/ammario/mcping"
	"github.com/shirou/gopsutil/process"
	"github.com/golang-commonmark/markdown"
	"io/ioutil"
)

func main() {
	fmt.Println("starting")
	mux := http.NewServeMux()
	mux.HandleFunc("/healthcheck", HTTPHealthCheck)
	mux.HandleFunc("/minecraftstatus.html", HTTPMCServers)

	err := http.ListenAndServe("127.0.0.1:2201", http.StripPrefix("/api", mux))
	if err != nil {
		log.Fatalln(err)
	}
}

// ---

type stringWriter interface {
	WriteString(s string) (n int, err error)
}

func HTTPHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.(stringWriter).WriteString("ok\n")
}

// ---

var ErrProcessExited = errors.New("Process exited while reading the data")
var ErrNotAMinecraftServer = errors.New("not a Minecraft server")
var ErrServerStarting = errors.New("Server starting up... (stage 1)")
var ErrServerStarting2 = errors.New("Server starting up... (stage 2)")

type mcserverdata struct {
	Err       error `json:"-"`
	Error     string `json:"Err"`

	PID       int32
	CWD       string
	MOTD      string
	Port      string
	NewsFile  template.HTML
	MapName   string

	PingData  mcping.PingResponse
	PingError error
}

func (m *mcserverdata) IsAServer() bool {
	return m.Err != ErrNotAMinecraftServer
}

// IsError returns whether the Err field is filled.
func (m *mcserverdata) IsError() bool {
	return m.Err != nil
}

func (m *mcserverdata) HasPingError() bool {
	return m.PingError != nil
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

func (m *mcserverdata) ServerType() string {
	if m.PingData.Server == "Unknown" {
		return fmt.Sprintf("Minecraft %s", m.PingData.Version)
	}
	return fmt.Sprintf("%s %s", m.PingData.Server, m.PingData.Version)
}

var markdownRenderer = markdown.New(markdown.Breaks(true))

func (m *mcserverdata) readData(strPid string, wg *sync.WaitGroup) {
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

	pid, err := strconv.Atoi(strPid)
	failOnError(err)
	m.PID = int32(pid)

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

	m.Port = props["server-port"]
	m.MOTD = props["motd"]
	m.MapName = props["level-name"]

	newsFile, err := ioutil.ReadFile(fmt.Sprintf("%s/NEWS.md", cwd))
	if err != nil && !os.IsNotExist(err) {
		failOnError(err)
	} else if err != nil {
		m.NewsFile = template.HTML(markdownRenderer.RenderToString(newsFile))
	}

	pingResponse, err := mcping.Ping(fmt.Sprintf("localhost:%s", m.Port))
	if netErr, ok := err.(*net.OpError); ok {
		if _, ok := netErr.Err.(*os.SyscallError); ok {
			m.PingError = ErrServerStarting
		} else {
			m.PingError = netErr
		}
	} else if err == mcping.ErrSmallPacket {
		m.PingError = ErrServerStarting2
	} else if err != nil {
		fmt.Printf("%#v\n", err)
		m.PingError = err
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

func loadMCServersData() ([]mcserverdata, error) {
	bytes, err := exec.Command("pgrep", "java").Output()
	if err != nil {
		return nil, err
	}
	pids := strings.Split(strings.TrimSpace(string(bytes)), "\n")
	data := make([]mcserverdata, len(pids))
	var wg sync.WaitGroup
	for i, pid := range pids {
		wg.Add(1)
		go data[i].readData(pid, &wg)
	}
	wg.Wait()

	// TODO

	return data, nil
}

var serverStatusTemplate = template.Must(template.New("serverStatus").Parse(`
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
	{{- if .NewsFile }}<p class="props-comment">{{ .NewsFile }}</p>{{ end -}}
	{{- if .IncludeMapName }}<p><strong>Map: </strong><em>{{ .MapName }}</em></p>{{ end -}}
	{{- if true }}<p><strong>MOTD: </strong><em>{{.MOTD}}</em></p>{{ end -}}
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
	serverInfo, err := loadMCServersData()
	if err != nil {
		// write info failed to load
		w.(stringWriter).WriteString("<p>ERROR: failed to load server information")
		return
	}

	// Print the table
	err = serverStatusTemplate.Execute(w, serverInfo)
	if err != nil {
		w.(stringWriter).WriteString(fmt.Sprintf("<p>ERROR: %s", err))
	}

	if includeJsonDump {
		// Include raw data as a JSON dump
		for i, _ := range serverInfo {
			if serverInfo[i].Err != nil {
				serverInfo[i].Error = serverInfo[i].Err.Error()
			}
		}
		bytes, err := json.MarshalIndent(serverInfo, "", "\t")
		if err != nil {
			w.(stringWriter).WriteString("<p>ERROR: failed to marshal json")
			return
		}
		err = jsonTemplate.Execute(w, string(bytes))
		if err != nil {
			w.(stringWriter).WriteString(fmt.Sprintf("<p>ERROR: %s", err))
		}
	}
}

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/riking/homeapi/rcon"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/process"

	stderrors "errors"
)

type factorioModZipFilesystem struct {
	BaseDir string
}

var mustMatchRegex = regexp.MustCompile(`\A/factorio-\d+-\d+-\d+/mods\.zip\z`)
var errBadFilename = stderrors.New("Unacceptable filename for modpack download")

func (fs *factorioModZipFilesystem) Open(name string) (http.File, error) {
	if !mustMatchRegex.Match([]byte(name)) {
		return nil, errBadFilename
	}
	return os.Open(fmt.Sprintf("%s%s", fs.BaseDir, name))
}

var _rcon_password string = "__X"

func RconPassword() string {
	if _rcon_password != "__x" {
		return _rcon_password
	}
	content, err := ioutil.ReadFile(fmt.Sprintf("%s/Factorio/rcon", os.Getenv("HOME")))
	if err != nil {
		panic(errors.Wrap(err, "fetching rcon password"))
	}
	_rcon_password = string(content)
	return _rcon_password
}

type factoriodata struct {
	PID   int32
	Err   error
	Stack string

	CWD      string
	Cmdline  []string
	Port     string
	NewsFile template.HTML

	PingError error
	PingData  struct {
		Online  int
		Players []string
	}
	RconDebug string

	ModpackErr error
}

func (m *factoriodata) IsError() bool {
	return m.Err != nil
}

func (m *factoriodata) HasPingError() bool {
	return m.PingError != nil
}

func (m *factoriodata) DefaultPort() bool {
	return m.Port == "34197"
}

func (m *factoriodata) PortNumber() int {
	i, _ := strconv.Atoi(m.Port)
	return i
}

func (m *factoriodata) Name() string {
	lastSlash := strings.LastIndex(m.CWD, "/")
	if lastSlash == -1 {
		return ""
	}
	return m.CWD[lastSlash+1:]
}

func (m *factoriodata) ModsPath() string {
	return fmt.Sprintf("https://home.riking.org/api/factoriomods/%s/mods.zip", m.Name())
}

var mapNameRgx = regexp.MustCompile(`saves/([a-zA-z0-9_ \.\-]+)\.zip`)

func (m *factoriodata) MapName() string {
	// rely on stable format of start.sh
	if len(m.Cmdline) >= 3 {
		match := mapNameRgx.FindStringSubmatch(m.Cmdline[2])
		if match != nil {
			return match[1]
		}
	}
	return "(UNKNOWN - map file must be argument 3, in format saves/xxx.zip)"
}

func (m *factoriodata) loadConfigFile(r io.Reader) error {
	s := bufio.NewScanner(r)
	for s.Scan() {
		t := s.Text()
		split := strings.SplitN(t, "=", 2)
		if len(split) == 1 {
			continue // don't care about ini headings
		}
		k := split[0]
		v := split[1]
		switch k {
		case "port":
			m.Port = v
		default:
			// pass
		}
	}
	if s.Err() != nil {
		return s.Err()
	}
	return nil
}

func (m *factoriodata) checkModpackFile() error {
	// exit status 12 = nothing to freshen
	err := exec.Command("zip", "-r", "-u", "mods.zip", "mods/").Wait()
	if exErr, ok := err.(*exec.ExitError); ok {
		if exErr.ProcessState == nil {
			return err
		}
		dat := exErr.ProcessState.Sys()
		if ws, ok := dat.(syscall.WaitStatus); ok {
			if ws.ExitStatus() == 12 {
				return nil
			}
		}
	}
	return err
}

func (m *factoriodata) readData(pid int32, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if err := recover(); err != nil {
			trace := make([]byte, 1024)
			count := runtime.Stack(trace, false)
			m.Err = err.(error)
			m.Stack = string(trace[:count])
		}
	}()
	failOnError := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	m.PID = pid

	proc, err := process.NewProcess(m.PID)
	failOnError(err)

	cwd, err := proc.Cwd()
	failOnError(err)

	m.CWD = cwd

	cmdline, err := proc.CmdlineSlice()
	failOnError(err)
	m.Cmdline = cmdline

	file, err := os.Open(fmt.Sprintf("%s/config/config.ini", m.CWD))
	failOnError(err)
	err = m.loadConfigFile(file)
	failOnError(err)

	newsFile, err := ioutil.ReadFile(fmt.Sprintf("%s/NEWS.md", cwd))
	if err != nil && !os.IsNotExist(err) {
		failOnError(err)
	} else if err == nil {
		m.NewsFile = template.HTML(markdownRenderer.RenderToString(newsFile))
	}

	m.PingError = m.pingServer()

	m.ModpackErr = m.checkModpackFile()
}

const RCON_PORT_OFFSET = -1000

func (m *factoriodata) pingServer() error {
	c, err := rcon.DialTimeout("127.0.0.1", m.PortNumber()+RCON_PORT_OFFSET, RconPassword(), 1*time.Second)
	if err != nil {
		return errors.Wrap(err, "connecting to rcon")
	}
	resp, err := c.Command("print 'hello'")
	if err != nil {
		return errors.Wrap(err, "executing command")
	}
	fmt.Println(resp)
	m.RconDebug = resp
	return nil
}

func loadFactorioData() ([]factoriodata, error) {
	pids, err := pgrep("factorio")
	if err != nil {
		return nil, errors.Wrap(err, "checking for factorio processes")
	}

	data := make([]factoriodata, len(pids))
	var wg sync.WaitGroup
	wg.Add(len(pids))
	for i, pid := range pids {
		go data[i].readData(pid, &wg)
	}
	wg.Wait()

	return data, nil
}

var factorioStatusTemplate = template.Must(template.New("factorioStatus").Parse(`
<table class="table table-bordered table-striped"><thead>
    <th>Server</th>
    <th>Port</th>
    <th>About</th>
</thead>
{{- range . -}}
<tr>
{{- if .IsError -}}
    <td colspan="4"><b>Error</b>: {{ .Err.Error }}<br>{{.Stack}}
{{- else -}}
    <td class="name">
        {{- .Name -}}
    </td>
    <td class="port">
        {{- if .DefaultPort -}}
        <span class="connect-hostname connect-port">home.riking.org</span>
        {{- else -}}
        <span class="connect-hostname">home.riking.org:</span><span class="connect-port">{{ .Port }}</span>
        {{- end -}}
    </td>
    <td class="motd">
        {{- if .NewsFile }}{{ .NewsFile }}{{ end -}}
        {{- if .MapName }}<p><strong>Map: </strong><em>{{ .MapName }}</em></p>{{ end -}}
        <p><a href="{{.ModsPath}}">Download Modpack</a></p>
    </td>
    <td class="online">
	{{- if .HasPingError -}}
            <p class="has-warning"><span class="control-label">{{ .PingError.Error }}</span></p>
        {{- else -}}
            <p><strong>{{ .PingData.Online }}</strong> players online</p>
            <ul>{{ range .PingData.Players }}<li>{{ . }}</li>{{ end }}</ul>
        {{- end -}}
        {{ .RconDebug }}
    </td>
{{- end -}}
</tr>
{{- end -}}
</table>
`))

func HTTPFactorio(w http.ResponseWriter, r *http.Request) {
	serverInfo, err := loadFactorioData()
	if err != nil {
		// write info failed to load
		w.(stringWriter).WriteString("<p>ERROR: failed to load server information<br>")
		w.(stringWriter).WriteString(err.Error())
		return
	}

	// Print the table
	err = factorioStatusTemplate.Execute(w, serverInfo)
	if err != nil {
		w.(stringWriter).WriteString("<p>ERROR: failed to print server information<br>")
		w.(stringWriter).WriteString(err.Error())
		return
	}

	if false {
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

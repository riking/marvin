package main

import (
	"bufio"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/shirou/gopsutil/process"
)

type factorioModZipFilesystem struct {
	BaseDir string
}

var mustMatchRegex = regexp.MustCompile(`\Afactorio-\d+-\d+-\d+/mods\.zip\z`)
var errBadFilename = errors.New("Unacceptable filename")

func (fs *factorioModZipFilesystem) Open(name string) (http.File, error) {
	if !mustMatchRegex.Match([]byte(name)) {
		return nil, errBadFilename
	}
	return os.Open(fmt.Sprintf("%s/%s", fs.BaseDir, name))
}

type factoriodata struct {
	PID   int32
	Err   error
	Stack string

	CWD      string
	Cmdline  []string
	Port     string
	NewsFile template.HTML

	ModpackErr error
}

func (m *factoriodata) IsError() bool {
	return m.Err != nil
}

func (m *factoriodata) DefaultPort() bool {
	return m.Port == "34197"
}

func (m *factoriodata) Name() string {
	lastSlash := strings.LastIndex(m.CWD, "/")
	if lastSlash == -1 {
		return ""
	}
	return m.CWD[lastSlash+1:]
}

func (m *factoriodata) ModsPath() string {
	return fmt.Sprintf("https://home.riking.org/factoriomods/%s/mods.zip", m.Name())
}

func (m *factoriodata) MapName() string {
	// rely on stable format of start.sh
	if len(m.Cmdline) == 3 {
		return m.Cmdline[2]
	}
	return "(UNKNOWN - TELL OPERATOR TO CHECK start.sh)"
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

func (m *factoriodata) checkModpackFile() {
	// exit status 12 = nothing to freshen
	err := exec.Command("zip", "-r", "-u", "mods.zip", "mods/").Wait()
	if exErr, ok := err.(*exec.ExitError); ok {
		if exErr.ProcessState == nil {
			m.ModpackErr = err
			return
		}
		dat := exErr.ProcessState.Sys()
		if ws, ok := dat.(syscall.WaitStatus); ok {
			if ws.ExitStatus() == 12 {
				m.ModpackErr = nil
				return
			}
		}
	}
	if err != nil {
		m.ModpackErr = err
		return
	}
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

	m.checkModpackFile()
}

func loadFactorioData() ([]factoriodata, error) {
	pids, err := pgrep("factorio")
	if err != nil {
		return nil, err
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
        <span class="connect-hostname">home.riking.org</span>
        {{- else -}}
        <span class="connect-hostname">home.riking.org:</span><span class="connect-port">{{ .Port }}</span>
        {{- end -}}
    </td>
    <td class="motd">
        {{- if .NewsFile }}{{ .NewsFile }}{{ end -}}
        {{- if .MapName }}<p><strong>Map: </strong><em>{{ .MapName }}</em></p>{{ end -}}
        <p><a href="{{.ModsPath}}">Download Modpack</a></p>
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
}

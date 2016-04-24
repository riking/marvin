package main

import (
	"net/http"
	"log"
	"os/exec"
	"strings"
	"fmt"
	"os"
	"sync"
	"errors"
	"strconv"
	"encoding/json"
	"html/template"
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

type stringWriter interface{
	WriteString(s string) (n int, err error)
}

func HTTPHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.(stringWriter).WriteString("ok\n")
}

// ---

var ErrProcessExited = errors.New("Process exited while reading the data")
var ErrNotAMinecraftServer = errors.New("not a Minecraft server")

type mcserverdata struct {
	Err error

	PID int
	CWD string
	MOTD string
	Port string
	PropsComment string
}

func (m *mcserverdata) IsAServer() bool {
	return m.Err != ErrNotAMinecraftServer
}

// IsError returns whether the Err field is filled.
func (m *mcserverdata) IsError() bool {
	return m.Err != nil
}

// Name is the name of the server, which is the name of the directory it is run from.
func (m *mcserverdata) Name() string {
	// TODO
	return m.CWD
}

func (m *mcserverdata) readData(strPid string, wg *sync.WaitGroup) {
	defer wg.Done()
	pid, err := strconv.Atoi(strPid)
	if err != nil {
		m.Err = err
	}
	m.PID = pid

	cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", m.PID))
	if err != nil {
		m.Err = err
		return
	}
	// XXX totally hacky
	if cwd == "/tank/crashplan" {
		m.Err = ErrNotAMinecraftServer
		return
	}
	m.CWD = cwd

	file, err := os.Open(fmt.Sprintf("%s/server.properties", cwd))
	if err != nil {
		m.Err = err
		return
	}
	props, err := LoadServerPropsFile(file)
	if err != nil {
		m.Err = err
		return
	}

	m.Port = props["server-port"]
	m.MOTD = props["motd"]
	m.PropsComment = props["homepage-comment"]
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
    <th>MOTD</th>
</thead>
{{- range . -}}
{{- if .IsAServer -}}
<tr>
    {{- if .IsError -}}
        <td colspan="4"><b>Error</b>: {{.Err.Error}}
    {{- else -}}
        <td class="name">{{.Name}}</td><td class="port">{{.Port}}</td><td class="motd"><blockquote>{{.MOTD}}</blockquote></td>
    {{- end -}}
</tr>
{{- end}}{{end -}}
</table>
`))

var jsonTemplate = template.Must(template.New("showJson").Parse(`<pre><code>{{.}}</code></pre>`))

func HTTPMCServers(w http.ResponseWriter, r *http.Request) {
	serverInfo, err := loadMCServersData()
	if err != nil {
		// write info failed to load
		w.(stringWriter).WriteString("<p>ERROR: failed to load server information")
		return
	}
	_ = serverInfo

	bytes, err := json.MarshalIndent(serverInfo, "", "\t")
	if err != nil {
		w.(stringWriter).WriteString("<p>ERROR: failed to marshal json")
		return
	}
	err = jsonTemplate.Execute(w, string(bytes))
	if err != nil {
		w.(stringWriter).WriteString(fmt.Sprintf("<p>ERROR: %s", err))
	}

	err = serverStatusTemplate.Execute(w, serverInfo)
	if err != nil {
		w.(stringWriter).WriteString(fmt.Sprintf("<p>ERROR: %s", err))
	}
}

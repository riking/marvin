package main

import (
	"net/http"
	"log"
	"os/exec"
	"strings"
	"io/ioutil"
	"fmt"
	"os"
	"sync"
	"errors"
	"strconv"
)

func main() {

	http.HandleFunc("/healthcheck", HTTPHealthCheck)

	err := http.ListenAndServe("127.0.0.1:2201", nil)
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

type mcserverdata struct {
	Err error

	PID int
	CWD string
	MOTD string
	Port string
}

var ErrProcessExited = errors.New("Process exited while reading the data")

func (m *mcserverdata) readData(strPid string, wg *sync.WaitGroup) {
	defer wg.Done()
	pid, err := strconv.Atoi(strPid)
	if err != nil {
		m.Err = err
	}
	m.PID = pid

	bytes, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cwd", m.PID))
	if os.IsNotExist(err) {
		m.Err = ErrProcessExited
		return
	} else if err != nil {
		m.Err = err
		return
	}
	cwd := string(bytes)
	m.CWD = cwd

	bytes, err = ioutil.ReadFile(fmt.Sprintf("%s/server.properties", cwd))
	if err != nil {
		m.Err = err
		return
	}
	serverProps := string(bytes)
	_ = serverProps
}

func loadMCServersData() ([]mcserverdata, error) {
	bytes, err := exec.Command("pgrep", "java").Output()
	if err != nil {
		return nil, err
	}
	pids := strings.Split(string(bytes), "\n")
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

func HTTPMCServers(w http.ResponseWriter, r *http.Request) {

	serverInfo, err := loadMCServersData()
	if err != nil {
		// write info failed to load
		// writeFailure(w, "Failed to load Minecraft server information")
		return
	}
	_ = serverInfo
}

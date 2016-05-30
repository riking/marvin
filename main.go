package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	fmt.Println("starting")
	mux := http.NewServeMux()
	mux.HandleFunc("/healthcheck", HTTPHealthCheck)
	mux.HandleFunc("/minecraftstatus.html", HTTPMCServers)
	mux.HandleFunc("/factoriostatus.html", HTTPFactorio)

	mux.Handle("/factoriomods/", http.StripPrefix("/factoriomods/", http.FileServer(factorioModZipFilesystem{BaseDir: "/tank/home/mcserver/Factorio"})))

	err := http.ListenAndServe("127.0.0.1:2201", http.StripPrefix("/api", mux))
	if err != nil {
		log.Fatalln(err)
	}
}

func pgrep(search string) ([]int32, error) {
	bytes, err := exec.Command("pgrep", search).Output()
	if exErr, ok := err.(*exec.ExitError); ok {
		if exErr.ProcessState != nil && exErr.ProcessState.Success() == false {
			// no processes
			return nil, nil
		}
	} else if err != nil {
		return nil, err
	}
	strPids := strings.Split(strings.TrimSpace(string(bytes)), "\n")
	pids := make([]int32, len(strPids))
	for i := range pids {
		p, err := strconv.Atoi(strPids[i])
		if err != nil {
			return nil, err
		}
		pids[i] = int32(p)
	}
	return pids, nil
}

// ---

type stringWriter interface {
	WriteString(s string) (n int, err error)
}

func HTTPHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.(stringWriter).WriteString("ok\n")
}

// ---

package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"regexp"
)

func main() {
	fmt.Println("starting")
	rootMux := http.NewServeMux()
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/healthcheck", HTTPHealthCheck)
	apiMux.HandleFunc("/minecraftstatus.html", HTTPMCServers)
	apiMux.HandleFunc("/factoriostatus.html", HTTPFactorio)

	apiMux.Handle("/factoriomods/", http.StripPrefix("/factoriomods/", http.FileServer(&ModZipFilesystem{
		BaseDir: "/tank/home/mcserver/Factorio",
		MatchRegex: regexp.MustCompile(`\A/factorio-\d+-\d+-\d+/mods\.zip\z`),
	})))
	minecraftModFS.BaseDir = "/tank/home/mcserver"
	apiMux.Handle("/minecraftmods/", http.StripPrefix("/minecraftmods/", http.FileServer(minecraftModFS)))

	api := http.StripPrefix("/api", apiMux)
	rootMux.Handle("/api/", api)

	rootMux.HandleFunc("/42/", curlKiller(http.StripPrefix("/42/", http.FileServer(http.Dir("/tank/www/home.riking.org/42")))))

	err := http.ListenAndServe("127.0.0.1:2201", rootMux)
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

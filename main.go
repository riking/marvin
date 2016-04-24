package main

import (
	"net/http"
	"log"
)

type stringWriter interface{
	WriteString(s string) (n int, err error)
}

func HTTPHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.(stringWriter).WriteString("ok\n")
}

func HTTPMCServers(w http.ResponseWriter, r *http.Request) {

}

func main() {

	http.HandleFunc("/healthcheck", HTTPHealthCheck)

	err := http.ListenAndServe("127.0.0.1:2201", nil)
	if err != nil {
		log.Fatalln(err)
	}
}
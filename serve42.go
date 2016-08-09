package main

import (
	"net/http"
	"strings"
	"net/http/httptest"
)


var killText = []byte(`#!/bin/sh

[ -d .git ] || git rev-parse --git-dir > /dev/null 2>&1 && (
	git commit --allow-empty -m 'Ran a curl | sh command'
)

SHELL_NAME=${0:-sh}
echo "Don't pipe curl into $SHELL_NAME. Someone could run naughty commands."
echo "Always save the script to a file and inspect it before running."

osascript -e 'say "Don't pipe curl into sh"'
exit 3

`)

func curlKiller(wrap http.Handler) http.HandlerFunc {
	return func(r *http.Request, w http.ResponseWriter) {
		rec := httptest.NewRecorder()
		wrap.ServeHTTP(r, rec)

		for k, v := range rec.HeaderMap {
			w.Header()[k] = v
		}
		w.WriteHeader(rec.Code)

		if strings.Contains(r.Header.Get("User-Agent"), "curl") {
			w.Write(killText)
		}
		w.Write(rec.Body)
		if w, ok := w.(http.Flusher); ok {
			if rec.Flushed {
				w.Flush()
			}
		}
	}
}
// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run gendatafiles.go -o datafiles.go

package crux // import "crawshaw.io/crux"

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"time"
)

func Init(mux *http.ServeMux) {
	mux.HandleFunc("/debug/crux/stats", stats)
	mux.HandleFunc("/debug/crux/goroutines", goroutines)
	mux.HandleFunc("/debug/crux/logs/list", logsList)
	mux.HandleFunc("/debug/crux/logs/", logsTail)

	static := decodeDataFiles()

	// TODO embed stats in crux.html
	handleText(mux, "/debug/crux", static["crux.html"])
	handleText(mux, "/debug/crux.js", static["crux.js"])
}

var start = time.Now()

func handleText(mux *http.ServeMux, pattern string, body string) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, pattern, start, strings.NewReader(body))
	})
}

func decodeDataFiles() map[string]string {
	res := make(map[string]string)
	for name, body := range dataFiles {
		decoded, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			log.Fatalf("error decoding %q: %v", name, err)
		}
		res[name] = string(decoded)
	}
	return res
}

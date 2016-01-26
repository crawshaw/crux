// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate tsc crux.ts
//go:generate go run gendatafiles.go -o datafiles.go

package crux

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"crawshaw.io/crux/internal/prg"
)

func Init(mux *http.ServeMux) {
	mux.HandleFunc("/debug/crux/stats", stats)
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

var stackBuf []byte

/*
goroutine 1 [running]:
crawshaw.io/crux.Load()
	/Users/crawshaw/src/crawshaw.io/crux/crux.go:11 +0x4c
main.main()
	/Users/crawshaw/src/crawshaw.io/crux/main.go:17 +0x53
*/

func Load() (string, error) {
	if stackBuf == nil {
		stackBuf = make([]byte, 1<<22)
	}
	n := runtime.Stack(stackBuf, true)
	fmt.Printf("%s\n", stackBuf[:n])
	s, err := prg.Load(stackBuf[:n])
	if err != nil {
		fmt.Printf("Load failed; %v\n", err)
		return "", fmt.Errorf("crux: %v", err)
	}
	res, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("crux: %v", err)
	}

	// debug
	{
		buf := new(bytes.Buffer)
		if err := json.Indent(buf, res, "", "\t"); err != nil {
			panic(err)
		}
		fmt.Printf("%s\n", buf.String())
	}

	return string(res), nil
}

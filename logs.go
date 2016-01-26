// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crux

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

func TeeStdio() {
	tee("stdout", &os.Stdout)
	tee("stderr", &os.Stderr)
}

func RegisterLogs(paths ...string) {
	logsMu.Lock()
	defer logsMu.Unlock()
	panic("TODO")
}

var (
	logsMu   sync.Mutex
	logFiles = make(map[string]*os.File)
)

func tee(name string, orig **os.File) {
	r, w, err := os.Pipe()
	if err != nil {
		log.Fatalf("crux: cannot create IO pipe: %v", err)
	}

	storageW, err := ioutil.TempFile("", "crux-stdio-")
	if err != nil {
		log.Fatalf("crux: cannot create IO temp file: %v", err)
	}
	storageR, err := os.Open(storageW.Name())
	if err != nil {
		log.Fatalf("crux: cannot reopen IO temp file: %v", err)
	}

	go io.Copy(*orig, io.TeeReader(r, storageW))
	*orig = w

	logsMu.Lock()
	defer logsMu.Unlock()
	logFiles[name] = storageR
}

func logsList(w http.ResponseWriter, r *http.Request) {
	logsMu.Lock()
	var res struct {
		Files []string
	}
	for name := range logFiles {
		res.Files = append(res.Files, name)
	}
	logsMu.Unlock()

	b, err := json.Marshal(res)
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot marshal: %v", err), 500)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func logsTail(w http.ResponseWriter, r *http.Request) {
	var limit, offset int
	var err error

	if str := r.Header.Get("offset"); str != "" {
		offset, err = strconv.Atoi(str)
		if err != nil {
			http.Error(w, fmt.Sprintf("bad offset %q: %v", r.Header["offset"], err), 400)
			return
		}
		if offset < 0 {
			http.Error(w, fmt.Sprintf("bad offset: %d", offset, err), 400)
			return
		}
	}

	if str := r.Header.Get("limit"); str != "" {
		limit, err = strconv.Atoi(str)
		if err != nil {
			http.Error(w, fmt.Sprintf("bad limit %q: %v", str, err), 400)
			return
		}
		if limit < 0 {
			http.Error(w, fmt.Sprintf("bad limit: %d", limit, err), 400)
			return
		}
	}

	logName := strings.TrimPrefix(r.URL.Path, "/debug/crux/logs/")

	logsMu.Lock()
	defer logsMu.Unlock()
	src := logFiles[logName]
	if src == nil {
		http.Error(w, fmt.Sprintf("unknown log: %q", logName), 400)
		return
	}
	if _, err := src.Seek(int64(offset), 0); err != nil {
		http.Error(w, fmt.Sprintf("cannot seek to %s:%d: %v", logName, offset, err), 500)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if limit > 0 {
		_, err = io.CopyN(w, src, int64(limit))
	} else {
		_, err = io.Copy(w, src)
	}
	if err != nil && err != io.EOF {
		http.Error(w, fmt.Sprintf("cannot read %s:%d: %v", logName, offset, err), 500)
		return
	}
}

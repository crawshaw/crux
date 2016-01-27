// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crux

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

func stats(w http.ResponseWriter, r *http.Request) {
	// The stats handler is not designed to be visited directly. It is
	// an HTML fragment inserted into the main page.
	buf := new(bytes.Buffer)

	dataMu.Lock()
	updateData()
	err := statsTmpl.Execute(buf, data)
	dataMu.Unlock()

	if err != nil {
		err := fmt.Errorf("cannot generate stats: %v", err)
		fmt.Fprintf(os.Stderr, "crux: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	buf.WriteTo(w)
}

var (
	dataMu sync.Mutex
	data   statsData
)

func updateData() {
	if data.MemChart == nil {
		go updateMemChartLoop()
	}
	data = statsData{
		CommandLine:  strings.Join(os.Args, " "),
		Uptime:       time.Now().Sub(start).String(),
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		Environ:      make(map[string]string),
		MemChart:     data.MemChart,
	}
	for _, v := range os.Environ() {
		i := strings.Index(v, "=")
		if i == -1 {
			continue
		}
		data.Environ[v[:i]] = v[i+1:]
	}
}

type statsData struct {
	CommandLine  string
	Uptime       string
	GoVersion    string
	NumCPU       int
	NumGoroutine int
	Environ      map[string]string
	MemChart     *memChart
}

var statsTmpl = template.Must(memchartTmpl.New("stats").Parse(`
<table>
<tr><th>Command Line</th><td>{{.CommandLine}}</td></tr>
<tr><th>Uptime</th><td>{{.Uptime}}</td></tr>
<tr><th>Go Version</th><td>{{.GoVersion}}</td></tr>
<tr><th></th><td>&nbsp;</td></tr>
<tr><th>CPUs</th><td>{{.NumCPU}}</td></tr>
<tr><th>Goroutines</th><td>{{.NumGoroutine}}</td></tr>
</table>
{{if .MemChart}}{{template "memchart" .MemChart}}{{end}}

<h3>Environment Variables</h3>
<table id="env">
{{range $k, $v := .Environ}}
<tr><th>{{$k}}</th><td>{{$v}}</td></tr>
{{end}}
</table>
`))

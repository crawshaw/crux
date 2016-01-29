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
	"sync"

	"crawshaw.io/crux/internal/prg"
)

var (
	stackBufMu sync.Mutex
	stackBuf   []byte
)

func goroutines(w http.ResponseWriter, r *http.Request) {
	stackBufMu.Lock()
	if stackBuf == nil {
		stackBuf = make([]byte, 1<<22)
	}
	n := runtime.Stack(stackBuf, true)
	s, err := prg.Load(stackBuf[:n])
	stackBufMu.Unlock()

	if err != nil {
		err := fmt.Errorf("goroutine load failed: %v", err)
		fmt.Fprintf(os.Stderr, "crux: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}

	buf := new(bytes.Buffer)
	if err := goroutinesTmpl.Execute(buf, s); err != nil {
		err := fmt.Errorf("cannot generate goroutines: %v", err)
		fmt.Fprintf(os.Stderr, "crux: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	buf.WriteTo(w)
}

var goroutinesTmpl = template.Must(template.New("goroutines").Parse(`
<ul>
{{range .Goroutines}}
<li>{{.Num}}:
	{{range $i, $element := .Func}}
		{{if eq $i 0}}
		{{.Name}}<ul>
		{{else}}
		<li>{{.Name}}</li>
		{{end}}
	{{end}}
	</ul>
</li>
{{end}}
</ul>
`))

// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// Generates datafiles.go
package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var outfile = flag.String("o", "", "result will be written file")

func main() {
	flag.Parse()

	files := []string{
		"crux.js",
		"crux.html",
	}

	buf := new(bytes.Buffer)
	fmt.Fprint(buf, `// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Auto-generated. Do not edit.

package crux
			        
var dataFiles = map[string]string{
`)

	for _, file := range files {
		src, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}
		data := base64.StdEncoding.EncodeToString(src)

		fmt.Fprintf(buf, "\t%q: `` +\n", filepath.Base(file))

		chunk := ""
		for len(data) > 0 {
			l := len(data)
			if l > 70 {
				l = 70
			}
			chunk, data = data[:l], data[l:]
			fmt.Fprintf(buf, "\t\t`%s` + \n", chunk)
		}
		fmt.Fprintf(buf, "\t``,\n")
	}

	fmt.Fprint(buf, "}\n")

	out, err := format.Source(buf.Bytes())
	if err != nil {
		buf.WriteTo(os.Stderr)
		log.Fatal(err)
	}

	w, err := os.Create(*outfile)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := w.Write(out); err != nil {
		log.Fatal(err)
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
}

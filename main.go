// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

// Small crux manual test program.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"crawshaw.io/crux"
)

func main() {
	crux.TeeStdio()
	fmt.Printf("test stdout\n")
	fmt.Fprintf(os.Stderr, "test stderr\n")
	crux.Init(http.DefaultServeMux)

	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	for i := 0; i < 50; i++ {
		fmt.Printf("%0d: initial noise\n", i)
	}

	go func() {
		i := 0
		for {
			time.Sleep(1 * time.Second)
			fmt.Printf("sleep, print %d\n", i)
			i++
		}
	}()

	// Test stack
	ch := make(chan int)
	go func() {
		<-ch
	}()
	select {}
}

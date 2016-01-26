// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prg

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type State struct {
	Snapshot   time.Time
	Goroutines []Stack
}

type Stack struct {
	Num       int
	State     string // "running", "idle", etc
	Func      []Func
	CreatedBy *Func `json:",omitempty"`
}

type Func struct {
	Name    string
	Pkg     string
	PkgPath string
	File    string
	Line    int
	Offset  int `json:",omitempty"`
}

var (
	headerRE = regexp.MustCompile(`^goroutine (\d+) \[([a-zA-Z ]+)\]:\n$`)
	funcRE   = regexp.MustCompile(`^(.*/)?([a-zA-Z0-9\.]+)\.([a-zA-Z0-9]+)(\(.*\))?\n$`)
	posRE    = regexp.MustCompile(`\t(.*):(\d+) ?(\+0x.*)?\n$`)
)

func (s *Stack) readFuncs(buf *bytes.Buffer) error {
	for {
		line, err := buf.ReadString('\n')
		if err == io.EOF {
			return io.EOF // TODO: error?
		}
		if err != nil {
			return err
		}
		if line == "\n" {
			break
		}
		createdBy := false
		if strings.HasPrefix(line, "created by ") {
			createdBy = true
			line = line[len("created by "):]
		}
		res := funcRE.FindStringSubmatch(line)
		if res == nil {
			return fmt.Errorf("expected function name, got: %q", line)
		}
		var (
			pkgPrefix = res[1]
			pkg       = res[2]
			name      = res[3]
		)
		// TODO: func arguments: %v\n", res[4]

		line, err = buf.ReadString('\n')
		if err == io.EOF {
			return fmt.Errorf("unexpected EOF reading function position")
		}
		res = posRE.FindStringSubmatch(line)
		if res == nil {
			return fmt.Errorf("expected function position, got: %q", line)
		}
		file := res[1]
		lineNum, err := strconv.Atoi(res[2])
		if err != nil {
			return fmt.Errorf("invalid function line %q: %v", line, err)
		}
		offsetNum := 0
		if res[3] != "" {
			if _, err := fmt.Sscanf(res[3][1:], "%x", &offsetNum); err != nil {
				return fmt.Errorf("invalid offset: %q: %v", line, err)
			}
		}

		f := Func{
			Name:    name,
			Pkg:     pkg,
			PkgPath: pkgPrefix + pkg,
			File:    file,
			Line:    lineNum,
			Offset:  offsetNum,
		}

		if createdBy {
			s.CreatedBy = &f
		} else {
			s.Func = append(s.Func, f)
		}
	}

	return nil
}

func Load(src []byte) (State, error) {
	buf := bytes.NewBuffer(src)

	state := State{
		Snapshot: time.Now(),
	}

	for {
		line, err := buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return State{}, err
		}
		res := headerRE.FindStringSubmatch(line)
		if res == nil {
			return State{}, fmt.Errorf("expected goroutine stack, got: %q", line)
		}
		num, err := strconv.Atoi(res[1])
		if err != nil {
			return State{}, fmt.Errorf("invalid goroutine number in %q: %v", line, err)
		}
		s := Stack{
			Num:   num,
			State: res[2],
		}
		err = s.readFuncs(buf)
		state.Goroutines = append(state.Goroutines, s)
		if err == io.EOF {
			break
		}
		if err != nil {
			return State{}, err
		}
	}

	return state, nil
}

// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crux

import (
	"fmt"
	"html/template"
	"math"
	"runtime"
	"time"
)

var memchartTmpl = template.Must(template.New("memchart").Parse(`
<div id="memchart">
<h3>Recent memory use</h3>
<svg id="memchartsvg" viewBox="0 0 {{.Width}} {{.Height}}" role="img">
	{{$heightMinusPad := .HeightMinusPad}}
	{{range .Pauses}}
	<g class="gcpause"><rect width="{{.Width}}" y="0" x="{{.X}}" height="{{$heightMinusPad}}"></rect></g>
	{{end}}

	<g class="grid"><line x1="{{.WidthMinusPad}}" x2="{{.WidthMinusPad}}" y1="0" y2="{{.HeightMinusPad}}"></line></g>
	<g class="grid"><line x1="0" x2="{{.WidthMinusPad}}" y1="{{.HeightMinusPad}}" y2="{{.HeightMinusPad}}"></line></g>

	<g class="labels x-labels">{{range .LabelsX}}
		<text x="{{.X}}" y="{{.Y}}">{{.Label}}</text>{{end}}
		<text x="300" y="340" class="label-title">Time</text>
	</g>
	<g class="labels y-labels">{{range .LabelsY}}
		<text x="{{.X}}" y="{{.Y}}">{{.Label}}</text>{{end}}
		<text x="{{.Width}}" y="150" class="label-title">{{.MemoryUnit}}</text>
	</g>

	<polyline fill="none" stroke="blue" strokwidth="1" points="{{range .RSS}}
		{{.X}}, {{.Yscaled}}{{end}}
	"/>
	<polyline fill="none" stroke="red" strokwidth="1" points="{{range .HeapInuse}}
		{{.X}}, {{.Yscaled}}{{end}}
	"/>


</svg>
</div>
`))

const (
	chartHorizon   = 1 * time.Minute
	chartPad       = 50
	chartWidth     = 480 + chartPad
	chartHeight    = 300 + chartPad
	chartTimePerPx = chartHorizon / time.Duration(chartWidth-chartPad)
)

type memChart struct {
	MemTimes  []time.Time
	RSS       []point
	HeapInuse []point

	numGC  int
	Pauses []pause

	Width          int
	Height         int
	Pad            int
	WidthMinusPad  int
	HeightMinusPad int
	LabelsX        []label
	LabelsY        []label
	MemoryUnit     string
}

type pause struct {
	Time  time.Time
	End   time.Time
	Width int
	X     int
}

type label struct {
	X, Y  int
	Label string
}

type point struct{ X, Y, Yscaled uint64 }

func updateMemChart(s *runtime.MemStats) {
	now := time.Now()

	dataMu.Lock()
	defer dataMu.Unlock()

	if data.MemChart == nil {
		data.MemChart = &memChart{
			Width:          chartWidth,
			Height:         chartHeight,
			Pad:            chartPad,
			WidthMinusPad:  chartWidth - chartPad,
			HeightMinusPad: chartHeight - chartPad,
		}
	}
	c := data.MemChart

	// Drop outdated memtimes data.
	dropI := 0
	for dropI < len(c.MemTimes) {
		if now.Sub(c.MemTimes[dropI]) < chartHorizon {
			break
		}
		dropI++
	}
	c.MemTimes = c.MemTimes[dropI:]
	c.RSS = c.RSS[dropI:]
	c.HeapInuse = c.HeapInuse[dropI:]

	// Add new memtimes data.
	c.MemTimes = append(c.MemTimes, now)
	c.RSS = append(c.RSS, point{Y: s.Sys - s.HeapReleased})
	c.HeapInuse = append(c.HeapInuse, point{Y: s.HeapInuse})

	// Add new GCs.
	numNewGCs := int(s.NumGC) - c.numGC
	if numNewGCs >= 256 {
		numNewGCs = 256
	}
	for numNewGCs > 0 {
		numNewGCs--
		i := int((s.NumGC - uint32(numNewGCs) + 255) % 256)
		end := time.Unix(0, int64(s.PauseEnd[i]))
		p := pause{
			Time:  end.Add(-time.Duration(s.PauseNs[i])),
			End:   end,
			Width: int(time.Duration(s.PauseNs[i]) / chartTimePerPx),
		}
		if p.Width == 0 {
			p.Width = 1
		}
		c.Pauses = append(c.Pauses, p)
	}
	c.numGC = int(s.NumGC)

	// Drop outdated GCs.
	dropI = 0
	for dropI < len(c.Pauses) {
		if now.Sub(c.Pauses[dropI].Time) < chartHorizon {
			break
		}
		dropI++
	}
	if dropI > 0 {
		fmt.Printf("dropping %d pauses\n", dropI)
	}
	c.Pauses = c.Pauses[dropI:]

	// Scale y-axis.
	ymax := uint64(0)
	for _, rss := range c.RSS {
		if rss.Y > ymax {
			ymax = rss.Y
		}
	}
	ymaxn := math.Pow(10, math.Floor(math.Log10(float64(ymax))))
	ymax = uint64(math.Ceil(float64(ymax)/ymaxn)) * uint64(ymaxn)
	ydiv := ymax / (chartHeight - chartPad)
	for i := range c.MemTimes {
		c.RSS[i].Yscaled = (chartHeight - chartPad) - c.RSS[i].Y/ydiv
		c.HeapInuse[i].Yscaled = (chartHeight - chartPad) - c.HeapInuse[i].Y/ydiv
	}
	var memdiv float64
	switch {
	case ymax >= 500e6:
		c.MemoryUnit = "GB"
		memdiv = 1e9
	case ymax >= 500e3:
		c.MemoryUnit = "MB"
		memdiv = 1e6
	default:
		c.MemoryUnit = "KB"
		memdiv = 1e3
	}
	textSize := 10
	ylabx := chartWidth - chartPad + 20
	ylaby := chartHeight - chartPad
	c.LabelsY = []label{
		{Label: fmt.Sprintf("%0.1f", float64(ymax)/1/memdiv), X: ylabx, Y: textSize},
		{Label: fmt.Sprintf("%0.1f", float64(ymax)/2/memdiv), X: ylabx, Y: (ylaby / 2) + textSize},
		{Label: "0", X: ylabx, Y: ylaby + textSize},
	}
	c.LabelsX = []label{
		{Label: "now", X: chartWidth - chartPad - 10, Y: chartHeight - chartPad + textSize},
		{
			Label: "-" + (chartHorizon / 2 / time.Second * time.Second).String(),
			X:     (chartWidth - chartPad) / 2,
			Y:     chartHeight - chartPad + textSize,
		},
		{
			Label: "-" + (chartHorizon / time.Second * time.Second).String(),
			X:     0,
			Y:     chartHeight - chartPad + textSize,
		},
	}

	// Position x-axis of memory time points.
	for i := len(c.MemTimes) - 1; i >= 0; i-- {
		pxOff := uint64(now.Sub(c.MemTimes[i]) / chartTimePerPx)
		x := chartWidth - pxOff - chartPad
		c.RSS[i].X = x
		c.HeapInuse[i].X = x
	}
	for i := len(c.Pauses) - 1; i >= 0; i-- {
		pxOff := uint64(now.Sub(c.Pauses[i].Time) / chartTimePerPx)
		x := chartWidth - pxOff - chartPad
		c.Pauses[i].X = int(x)
	}
}

func updateMemChartLoop() {
	ticker := time.Tick(100 * time.Millisecond)
	var s runtime.MemStats
	for {
		<-ticker
		runtime.ReadMemStats(&s)
		updateMemChart(&s)
	}
}

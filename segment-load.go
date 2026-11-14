package main

import (
	"fmt"
	"runtime"

	pwl "github.com/justjanne/powerline-go/powerline"
	"github.com/shirou/gopsutil/v3/load"
)

func segmentLoad(p *powerline) []pwl.Segment {
	avg, err := load.Avg()
	if err != nil {
		return []pwl.Segment{}
	}
	return loadSegment(p, avg, runtime.NumCPU())
}

func loadSegment(p *powerline, avg *load.AvgStat, cpus int) []pwl.Segment {
	value := avg.Load5
	switch p.theme.LoadAvgValue {
	case 1:
		value = avg.Load1
	case 15:
		value = avg.Load15
	}

	bg := p.theme.LoadBg
	if value > float64(cpus)*p.theme.LoadThresholdBad {
		bg = p.theme.LoadHighBg
	}

	return []pwl.Segment{{
		Name:       "load",
		Content:    fmt.Sprintf("%.2f", value),
		Foreground: p.theme.LoadFg,
		Background: bg,
	}}
}

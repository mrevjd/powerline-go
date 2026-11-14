package main

import (
	pwl "github.com/mrevjd/powerline-go/powerline"
	"strings"
	"time"
)

func segmentTime(p *powerline) []pwl.Segment {
	return []pwl.Segment{{
		Name:       "time",
		Content:    time.Now().Format(strings.TrimSpace(p.cfg.Time)),
		Foreground: p.theme.TimeFg,
		Background: p.theme.TimeBg,
	}}
}

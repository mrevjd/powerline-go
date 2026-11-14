package main

import (
	"os"

	pwl "github.com/mrevjd/powerline-go/powerline"
)

func segmentVirtualGo(p *powerline) []pwl.Segment {
	env, _ := os.LookupEnv("VIRTUALGO")
	if env == "" {
		return []pwl.Segment{}
	}

	return []pwl.Segment{{
		Name:       "vgo",
		Content:    env,
		Foreground: p.theme.VirtualGoFg,
		Background: p.theme.VirtualGoBg,
	}}
}

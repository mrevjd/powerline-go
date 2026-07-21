package main

import (
	"os"
	"strconv"
	"strings"

	pwl "github.com/justjanne/powerline-go/powerline"
)

// segmentShLvl shows the shell nesting depth from $SHLVL. It renders only when
// the depth is at least MinShlvl (default 2), so the base shell stays quiet and
// the segment appears once you are in a sub-shell. It reuses the jobs palette so
// it is themed consistently in every theme without introducing new colours.
// See #290.
func segmentShLvl(p *powerline) []pwl.Segment {
	shlvl, err := strconv.Atoi(strings.TrimSpace(os.Getenv("SHLVL")))
	if err != nil || shlvl < p.cfg.MinShlvl {
		return []pwl.Segment{}
	}
	return []pwl.Segment{{
		Name:       "shlvl",
		Content:    strconv.Itoa(shlvl),
		Foreground: p.theme.JobsFg,
		Background: p.theme.JobsBg,
	}}
}

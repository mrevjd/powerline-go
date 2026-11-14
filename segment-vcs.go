package main

import (
	"strings"

	pwl "github.com/mrevjd/powerline-go/powerline"
)

type vcsChange int

const (
	vcsModified vcsChange = iota
	vcsUntracked
	vcsMissing
)

// vcsTool describes a version control system whose prompt segment is just the
// branch name, coloured by whether the working copy is dirty.
type vcsTool struct {
	name      string
	branchCmd []string
	statusCmd []string
	// classify reports what kind of change a non-empty status line describes.
	classify func(line string) vcsChange
}

func (t vcsTool) segment(p *powerline) []pwl.Segment {
	out, _ := runCommand(t.branchCmd[0], t.branchCmd[1:]...)
	branch := firstLine(out)
	if branch == "" {
		return []pwl.Segment{}
	}

	changes := map[vcsChange]bool{}
	if out, err := runCommand(t.statusCmd[0], t.statusCmd[1:]...); err == nil {
		for _, line := range strings.Split(out, "\n") {
			if line != "" {
				changes[t.classify(line)] = true
			}
		}
	}

	if len(changes) == 0 {
		return []pwl.Segment{{
			Name:       t.name,
			Content:    branch,
			Foreground: p.theme.RepoCleanFg,
			Background: p.theme.RepoCleanBg,
		}}
	}
	extra := ""
	if changes[vcsUntracked] {
		extra += "+"
	}
	if changes[vcsMissing] {
		extra += "!"
	}
	return []pwl.Segment{{
		Name:       t.name,
		Content:    branch + " " + extra,
		Foreground: p.theme.RepoDirtyFg,
		Background: p.theme.RepoDirtyBg,
	}}
}

var (
	hgTool = vcsTool{
		name:      "hg",
		branchCmd: []string{"hg", "branch"},
		statusCmd: []string{"hg", "status"},
		classify: func(line string) vcsChange {
			switch line[0] {
			case '?':
				return vcsUntracked
			case '!':
				return vcsMissing
			}
			return vcsModified
		},
	}
	bzrTool = vcsTool{
		name:      "bzr",
		branchCmd: []string{"bzr", "nick"},
		statusCmd: []string{"bzr", "status"},
		classify: func(line string) vcsChange {
			switch line {
			case "unknown:":
				return vcsUntracked
			case "missing:":
				return vcsMissing
			}
			return vcsModified
		},
	}
	fossilTool = vcsTool{
		name:      "fossil",
		branchCmd: []string{"fossil", "branch", "current"},
		statusCmd: []string{"fossil", "changes", "--differ"},
		classify: func(line string) vcsChange {
			switch {
			case strings.HasPrefix(line, "EXTRA"):
				return vcsUntracked
			case strings.HasPrefix(line, "MISSING"):
				return vcsMissing
			}
			return vcsModified
		},
	}
)

func segmentHg(p *powerline) []pwl.Segment     { return hgTool.segment(p) }
func segmentBzr(p *powerline) []pwl.Segment    { return bzrTool.segment(p) }
func segmentFossil(p *powerline) []pwl.Segment { return fossilTool.segment(p) }

package main

import (
	"os"
	"path/filepath"
	"strings"

	pwl "github.com/mrevjd/powerline-go/powerline"
)

func goenvVersionOutput() (string, bool) {
	out, err := runCommand("goenv", "version")
	if err != nil {
		return "", false
	}
	if items := strings.Split(out, " "); len(items) > 1 {
		return items[0], true
	}
	return "", false
}

// segmentGoenv shows the selected Go version, taking the first of
// GOENV_VERSION, a .go-version file and `goenv version` that is set, and
// nothing while that is just the global version.
func segmentGoenv(p *powerline) []pwl.Segment {
	home, _ := os.UserHomeDir()
	global, _ := readTrimmed(filepath.Join(home, ".goenv", "version"))

	lookups := []func() (string, bool){
		func() (string, bool) {
			v := os.Getenv("GOENV_VERSION")
			return v, v != ""
		},
		func() (string, bool) { return readFileUpwards(p.cwd, ".go-version") },
		goenvVersionOutput,
	}
	for _, lookup := range lookups {
		if version, ok := lookup(); ok && version != global {
			return []pwl.Segment{{
				Name:       "goenv",
				Content:    version,
				Foreground: p.theme.GoenvFg,
				Background: p.theme.GoenvBg,
			}}
		}
	}
	return []pwl.Segment{}
}

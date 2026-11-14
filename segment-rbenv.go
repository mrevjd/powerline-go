package main

import (
	"os"
	"path/filepath"
	"strings"

	pwl "github.com/mrevjd/powerline-go/powerline"
)

// firstWordOf runs a version manager's "which version" command and returns the
// first space-separated word it prints.
func firstWordOf(cmd string, args ...string) (string, bool) {
	out, err := runCommand(cmd, args...)
	if err != nil {
		return "", false
	}
	word, _, _ := strings.Cut(out, " ")
	return word, true
}

func envVersion(name string) (string, bool) {
	v := os.Getenv(name)
	return v, v != ""
}

// segmentRbenv shows the first of RBENV_VERSION, a .ruby-version file, the
// global rbenv version and `rbenv version` that is set.
func segmentRbenv(p *powerline) []pwl.Segment {
	version, ok := envVersion("RBENV_VERSION")
	if !ok {
		version, ok = readFileUpwards(p.cwd, ".ruby-version")
	}
	if !ok {
		home, _ := os.UserHomeDir()
		version, ok = readTrimmed(filepath.Join(home, ".rbenv", "version"))
	}
	if !ok {
		version, ok = firstWordOf("rbenv", "version")
	}
	if !ok {
		return []pwl.Segment{}
	}
	return []pwl.Segment{{
		Name:       "rbenv",
		Content:    version,
		Foreground: p.theme.TimeFg,
		Background: p.theme.TimeBg,
	}}
}

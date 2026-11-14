package main

import (
	"os"
	"strings"

	pwl "github.com/justjanne/powerline-go/powerline"
)

func segmentRvm(p *powerline) []pwl.Segment {
	version, ok := envVersion("RUBY_VERSION")
	if !ok {
		version, ok = readFileUpwards(p.cwd, ".ruby-version")
	}
	if !ok {
		version, ok = firstWordOf("rvm", "current")
	}
	if !ok {
		return []pwl.Segment{}
	}

	// Drop the superfluous "ruby-" prefix.
	if parts := strings.Split(version, "-"); len(parts) > 1 {
		version = parts[1]
	}
	// Without a gemset in the version, take it from GEM_HOME.
	if !strings.Contains(version, "@") {
		if _, gemset, ok := strings.Cut(os.Getenv("GEM_HOME"), "@"); ok && gemset != "" {
			version += "@" + strings.Split(gemset, "@")[0]
		}
	}

	return []pwl.Segment{{
		Name:       "rvm",
		Content:    p.symbols.RvmIndicator + " " + version,
		Foreground: p.theme.RvmFg,
		Background: p.theme.RvmBg,
	}}
}

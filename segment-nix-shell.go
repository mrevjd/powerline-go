package main

import (
	pwl "github.com/justjanne/powerline-go/powerline"
	"os"
)

func segmentNixShell(p *powerline) []pwl.Segment {
	nixShell, _ := os.LookupEnv("IN_NIX_SHELL")
	if nixShell == "" {
		return []pwl.Segment{}
	}
	content := p.symbols.NixShellIndicator
	// The zsh-nix-shell plugin exposes the shell's packages in
	// NIX_SHELL_PACKAGES; append them when present. See #375.
	if packages := os.Getenv("NIX_SHELL_PACKAGES"); packages != "" {
		content += " " + packages
	}
	return []pwl.Segment{{
		Name:       "nix-shell",
		Content:    content,
		Foreground: p.theme.NixShellFg,
		Background: p.theme.NixShellBg,
	}}
}

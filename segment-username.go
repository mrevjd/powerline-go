package main

import (
	pwl "github.com/justjanne/powerline-go/powerline"
)

func segmentUser(p *powerline) []pwl.Segment {
	var userPrompt string
	// bash and zsh expand the username themselves, so those two get a prompt
	// template; every other shell gets the name as plain text.
	shellTemplate := true
	switch p.cfg.Shell {
	case "bash":
		userPrompt = "\\u"
	case "zsh":
		userPrompt = "%n"
	default:
		userPrompt = p.username
		shellTemplate = false
	}

	var background uint8
	if p.userIsAdmin {
		background = p.theme.UsernameRootBg
	} else {
		background = p.theme.UsernameBg
	}

	return []pwl.Segment{{
		Name:          "user",
		Content:       userPrompt,
		Foreground:    p.theme.UsernameFg,
		Background:    background,
		ShellTemplate: shellTemplate,
	}}
}

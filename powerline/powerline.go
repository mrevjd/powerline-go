package powerline

import (
	runewidth "github.com/mattn/go-runewidth"
)

// Segment describes an information to display on the command line prompt
type Segment struct {
	Name string
	// Content is the text to be displayed on the command line prompt
	Content string
	// Foreground is the text color (see https://misc.flogisoft.com/bash/tip_colors_and_formatting#background1)
	Foreground uint8
	// Background is the color of the filling background (see https://misc.flogisoft.com/bash/tip_colors_and_formatting#background1)
	Background uint8
	// Separator is the character to be used when generating multiple segments to override the default separator
	Separator string
	// SeparatorForeground is the character to be used when generating multiple segments to override the default foreground separator
	SeparatorForeground uint8
	// Priority is the priority of the segment. The higher, the less probable the segment will be dropped if the total length is too long
	Priority int
	// HideSeparators indicated not to display any separator with next segment.
	HideSeparators bool
	// ShellTemplate marks Content as a prompt template written for the current
	// shell (an escape sequence, or a `\u`/`%n`-style expansion) that has to
	// reach the prompt verbatim. Content is otherwise treated as data and is
	// shell-escaped when the prompt is rendered, so anything derived from a
	// repository, an environment variable or a plugin cannot be evaluated by
	// the shell. Leave this false unless the segment builds the template
	// itself.
	ShellTemplate bool
	Width         int
	// NewLine defines a newline segment to break the powerline in multi lines
	NewLine bool
}

func (s Segment) ComputeWidth(condensed bool) int {
	if condensed {
		return runewidth.StringWidth(s.Content) + runewidth.StringWidth(s.Separator)
	}
	return runewidth.StringWidth(s.Content) + runewidth.StringWidth(s.Separator) + 2
}

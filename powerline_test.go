package main

import (
	"strings"
	"testing"
	"time"

	pwl "github.com/justjanne/powerline-go/powerline"
)

func Test_detectShell(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "",
			want: "bare",
		},
		{
			name: "/bin/sh",
			want: "bare",
		}, {
			name: "/bin/bash",
			want: "bash",
		}, {
			name: "/usr/local/bin/bash5",
			want: "bash",
		}, {
			name: "/usr/bin/zsh",
			want: "zsh",
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectShell(tt.name); got != tt.want {
				t.Errorf("detectShell(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// A row that is too wide but holds nothing truncateRow may drop (termtitle's
// MaxInteger priority) used to spin forever looking for a segment to remove.
func TestTruncateRowTerminatesWithUndroppableSegments(t *testing.T) {
	t.Setenv("COLUMNS", "20")
	p := &powerline{
		cfg:      Config{MaxWidthPercentage: 100},
		Segments: [][]pwl.Segment{{{Content: "wide", Width: 1 << 20, Priority: MaxInteger}}},
	}
	done := make(chan struct{})
	go func() {
		p.truncateRow(0)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("truncateRow did not return")
	}
	if len(p.Segments[0]) != 1 {
		t.Errorf("undroppable segment was removed: %v", p.Segments[0])
	}
}

// A -truncate-segment-width too small to fit even the ellipsis left the
// shortened segment still over the limit, so it was picked and shortened again
// forever.
func TestTruncateRowTerminatesWithTinySegmentWidth(t *testing.T) {
	t.Setenv("COLUMNS", "20")
	p := &powerline{
		cfg: Config{MaxWidthPercentage: 100, TruncateSegmentWidth: 2},
	}
	// Enough segments that the row is still too wide once each is down to the
	// ellipsis, which is what kept the shortening loop going.
	row := make([]pwl.Segment, 200)
	for i := range row {
		row[i] = pwl.Segment{Content: strings.Repeat("x", 10), Priority: 1}
		row[i].Width = row[i].ComputeWidth(false)
	}
	p.Segments = [][]pwl.Segment{row}

	done := make(chan struct{})
	go func() {
		p.truncateRow(0)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("truncateRow did not return")
	}
	width := 0
	for _, s := range p.Segments[0] {
		width += s.Width
	}
	if width > 20 {
		t.Errorf("row is %d wide after truncation, want at most 20", width)
	}
}

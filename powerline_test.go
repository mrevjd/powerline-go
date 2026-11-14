package main

import (
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

package main

import (
	"strings"
	"testing"
)

// Covers #84: BoldForeground (exposed via -bold) makes fgColor emit the bold
// SGR prefix.
func Test_fgColorBold(t *testing.T) {
	plain := &powerline{shell: ShellInfo{ColorTemplate: "%s"}, theme: Theme{Reset: 0xFF}}
	bold := &powerline{shell: ShellInfo{ColorTemplate: "%s"}, theme: Theme{Reset: 0xFF, BoldForeground: true}}

	if got := plain.fgColor(42); strings.Contains(got, "1;38") {
		t.Errorf("non-bold fgColor should not use the bold prefix: %q", got)
	}
	if got := bold.fgColor(42); !strings.Contains(got, "1;38") {
		t.Errorf("bold fgColor should use the bold prefix 1;38: %q", got)
	}
}

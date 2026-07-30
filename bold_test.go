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

// -bold overrides the theme's own BoldForeground in both directions. An absent
// flag must leave a theme that asked for bold alone, and -bold=false must be able
// to turn bold off, which a plain bool config field could not express.
func Test_boldOverridesTheme(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	tests := []struct {
		name      string
		cfgBold   *bool
		themeBold bool
		want      bool
	}{
		{name: "unset leaves a non-bold theme alone", cfgBold: nil, themeBold: false, want: false},
		{name: "unset leaves a bold theme alone", cfgBold: nil, themeBold: true, want: true},
		{name: "-bold turns a non-bold theme bold", cfgBold: boolPtr(true), themeBold: false, want: true},
		{name: "-bold=false turns a bold theme off", cfgBold: boolPtr(false), themeBold: true, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newPowerline(Config{
				Shell:  "bare",
				Shells: ShellMap{"bare": {ColorTemplate: "%s"}},
				Theme:  "test",
				Themes: ThemeMap{"test": {Reset: 0xFF, BoldForeground: tt.themeBold}},
				Mode:   "test",
				Modes:  SymbolMap{"test": {}},
				Bold:   tt.cfgBold,
			}, "/", alignLeft)

			if p.theme.BoldForeground != tt.want {
				t.Errorf("BoldForeground = %v, want %v", p.theme.BoldForeground, tt.want)
			}
			if got := strings.Contains(p.fgColor(42), "1;38"); got != tt.want {
				t.Errorf("fgColor bold = %v, want %v", got, tt.want)
			}
		})
	}
}

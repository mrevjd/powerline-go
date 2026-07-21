package main

import (
	"reflect"
	"testing"
)

// Covers #217: -path-aliases pairs are comma-separated, and a comma inside a
// path can be escaped as "\,".
func Test_parsePathAliases(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want map[string]string
	}{
		{"simple pairs", "~/work=@w,~/src=@s", map[string]string{"~/work": "@w", "~/src": "@s"}},
		{"escaped comma in key", `/mnt/c/OneDrive - Co\, Inc/src=@one`, map[string]string{"/mnt/c/OneDrive - Co, Inc/src": "@one"}},
		{"escaped comma in value", `~/x=a\,b`, map[string]string{"~/x": "a,b"}},
		{"pair without equals is skipped", "~/x=@a,junk,~/y=@b", map[string]string{"~/x": "@a", "~/y": "@b"}},
		{"empty input", "", map[string]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePathAliases(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePathAliases(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

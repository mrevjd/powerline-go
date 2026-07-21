package main

import "strings"

// parsePathAliases parses the -path-aliases value: comma-separated key=value
// pairs. A comma inside a path can be escaped as "\," so aliases can target
// paths that contain commas. Pairs without an "=" are ignored. See #217.
func parsePathAliases(s string) map[string]string {
	aliases := map[string]string{}
	for _, pair := range splitEscaped(s, ',') {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		aliases[kv[0]] = kv[1]
	}
	return aliases
}

// splitEscaped splits s on sep, treating "\<sep>" as a literal separator (which
// is unescaped in the output). Backslashes before any other byte are left as-is.
func splitEscaped(s string, sep byte) []string {
	var parts []string
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == sep {
			buf.WriteByte(sep)
			i++
			continue
		}
		if s[i] == sep {
			parts = append(parts, buf.String())
			buf.Reset()
			continue
		}
		buf.WriteByte(s[i])
	}
	return append(parts, buf.String())
}

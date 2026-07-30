package main

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	pwl "github.com/justjanne/powerline-go/powerline"
)

const ellipsis = "\u2026"

type pathSegment struct {
	path     string
	home     bool
	root     bool
	ellipsis bool
	alias    bool
}

// segEqual compares two path segments, honouring the case-insensitive option.
func segEqual(a, b string, caseInsensitive bool) bool {
	if caseInsensitive {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func maybeAliasPathSegments(p *powerline, pathSegments []pathSegment) []pathSegment {
	pathSeparator := string(os.PathSeparator)

	if len(p.cfg.PathAliases) == 0 {
		return pathSegments
	}

	// Capacity, not length: appending to a len(n) slice would leave n empty keys
	// in front, which then match empty path segments and alias them to "".
	keys := make([]string, 0, len(p.cfg.PathAliases))
	for k := range p.cfg.PathAliases {
		keys = append(keys, k)
	}
	// Longest key first, so the most specific alias wins, then lexicographically
	// so the order is total. Sorting on length alone left keys of equal length in
	// Go's randomised map iteration order, and sort.Sort is not stable, so two
	// aliases of the same length competing for the same path rendered a different
	// prompt from one invocation to the next.
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) {
			return len(keys[i]) > len(keys[j])
		}
		return keys[i] < keys[j]
	})

Aliases:
	for _, k := range keys {
		// This turns a string like "foo/bar/baz" into an array of strings.
		path := strings.Split(strings.Trim(k, pathSeparator), pathSeparator)

		// If the path has 3 elements, we know we should look at pathSegments
		// in 3-element chunks.
		size := len(path)
		// If there aren't that many segments in our path we can skip to the
		// next alias.
		if size > len(pathSegments) {
			continue Aliases
		}

		alias := p.cfg.PathAliases[k]

	Segments:
		// We want to see if that array of strings exists in pathSegments.
		for i := range pathSegments {
			// This is the upper index that we would look at. So if i is 0,
			// then we'd look at pathSegments[0,1,2], then [1,2,3], etc.. If i
			// is 2, we'd look at pathSegments[2,3,4] and so on.
			max := (i + size) - 1

			// But if the upper index is out of bounds we can short-circuit
			// and move on to the next alias. The bound is the last valid index:
			// comparing against len(pathSegments)-i-1 instead made the limit
			// shrink as i advanced, so a run was only ever found in the first
			// half of the path and an alias covering the final segment never
			// matched unless it started at index 0.
			if max > len(pathSegments)-1 {
				continue Aliases
			}

			// Then we loop over the indices in path and compare the
			// elements. If any element doesn't match we can move on to the
			// next index in pathSegments.
			for j := range path {
				if !segEqual(path[j], pathSegments[i+j].path, p.cfg.PathAliasesCaseInsensitive) {
					continue Segments
				}
			}

			// They all matched! That means we can replace this slice with our
			// alias and skip to the next alias.
			pathSegments = append(
				pathSegments[:i],
				append(
					[]pathSegment{{
						path:  alias,
						alias: true,
					}},
					pathSegments[max+1:]...,
				)...,
			)
			continue Aliases
		}
	}

	return pathSegments
}

// homeDirs returns the paths that should be abbreviated to "~": the shell's
// $HOME and the account's home directory from the user database, plus their
// symlink-resolved forms. On some systems (notably WSL) $HOME and the passwd
// home diverge, or os.Getwd resolves a symlink that the home path does not, so
// comparing cwd against a single home path misses. Resolving is cheap because
// home paths are shallow. See #418.
func homeDirs(p *powerline) []string {
	seen := map[string]bool{}
	homes := make([]string, 0, 4)
	add := func(h string) {
		if h == "" {
			return
		}
		// Clean before comparing: filepath.Rel cleans its arguments, so "//" and
		// "/./" are the root as far as it is concerned, and a trailing separator
		// would otherwise defeat the dedup.
		h = filepath.Clean(h)
		// A home that is its own parent is a root, and every path is relative to a
		// root, so filepath.Rel would report the whole filesystem as home. That
		// covers "/" (containers with no home directory set HOME=/), "." (the
		// relative-path equivalent) and a bare Windows drive root. The old prefix
		// comparison never matched any of them, so keep those paths absolute.
		if h == filepath.Dir(h) || seen[h] {
			return
		}
		seen[h] = true
		homes = append(homes, h)
	}
	for _, h := range []string{os.Getenv("HOME"), p.userInfo.HomeDir} {
		if h == "" {
			continue
		}
		add(h)
		if resolved, err := filepath.EvalSymlinks(h); err == nil {
			add(resolved)
		}
	}
	return homes
}

// homeRelativePath reports whether cwd lies within one of the home directories
// and, if so, returns the path relative to that home ("" when cwd is exactly
// home). The result has no leading separator.
func homeRelativePath(p *powerline, cwd string) (string, bool) {
	parentDir := ".." + string(os.PathSeparator)
	for _, home := range homeDirs(p) {
		// filepath.Rel also rejects a relative cwd against an absolute home (and
		// vice versa) via its error, which a prefix comparison would miss.
		rel, err := filepath.Rel(home, cwd)
		if err != nil {
			continue
		}
		if rel == "." {
			return "", true
		}
		// Rel happily walks upwards, so ".." means cwd is outside this home.
		if rel == ".." || strings.HasPrefix(rel, parentDir) {
			continue
		}
		return rel, true
	}
	return "", false
}

// plainPath renders path segments back into the single string -cwd-mode plain
// emits. cwdToPathSegments strips the leading separators of an absolute path (the
// segmented modes imply them), so they are restored here unless the first segment
// already stands in for the root: "~" and an alias both replace it, and the root
// segment is the separator itself.
func plainPath(cwd string, pathSegments []pathSegment) string {
	pathSeparator := string(os.PathSeparator)

	names := make([]string, 0, len(pathSegments))
	for _, segment := range pathSegments {
		names = append(names, segment.path)
	}
	joined := strings.Join(names, pathSeparator)

	if len(pathSegments) == 0 {
		return joined
	}
	first := pathSegments[0]
	if first.root || first.home || first.alias {
		return joined
	}
	// Restore the whole leading separator run rather than a single separator, so
	// a Windows UNC path keeps its "\\" prefix. Taking it from the normalised
	// path keeps POSIX behaviour intact: path.Clean collapses "//" to "/" but
	// leaves backslashes alone, since it only understands "/".
	//
	// A Windows drive root renders as "C:" rather than "C:\", because the trailing
	// separator is not part of any segment. That matches what the segmented modes
	// have always shown there, so plain mode is no longer the odd one out.
	cleaned := path.Clean(cwd)
	return cleaned[:len(cleaned)-len(strings.TrimLeft(cleaned, pathSeparator))] + joined
}

func cwdToPathSegments(p *powerline, cwd string) []pathSegment {
	pathSeparator := string(os.PathSeparator)
	pathSegments := make([]pathSegment, 0)
	// Normalise the path so duplicate or trailing separators (e.g. bash
	// exporting PWD="//" after `cd //`) don't produce empty segments, which
	// previously panicked in dironly mode. See #424.
	cwd = path.Clean(cwd)

	if rel, ok := homeRelativePath(p, cwd); ok {
		pathSegments = append(pathSegments, pathSegment{
			path: "~",
			home: true,
		})
		cwd = rel
	} else if cwd == pathSeparator {
		pathSegments = append(pathSegments, pathSegment{
			path: pathSeparator,
			root: true,
		})
	}

	cwd = strings.Trim(cwd, pathSeparator)
	names := strings.Split(cwd, pathSeparator)
	if names[0] == "" {
		names = names[1:]
	}

	for _, name := range names {
		pathSegments = append(pathSegments, pathSegment{
			path: name,
		})
	}

	return maybeAliasPathSegments(p, pathSegments)
}

func maybeShortenName(p *powerline, pathSegment string) string {
	if p.cfg.CwdMaxDirSize > 0 && len(pathSegment) > p.cfg.CwdMaxDirSize {
		return pathSegment[:p.cfg.CwdMaxDirSize]
	}
	return pathSegment
}

func getColor(p *powerline, pathSegment pathSegment, isLastDir bool) (uint8, uint8, bool) {
	if pathSegment.home && p.theme.HomeSpecialDisplay {
		return p.theme.HomeFg, p.theme.HomeBg, true
	} else if pathSegment.alias {
		return p.theme.AliasFg, p.theme.AliasBg, true
	} else if isLastDir {
		return p.theme.CwdFg, p.theme.PathBg, false
	}
	return p.theme.PathFg, p.theme.PathBg, false
}

func segmentCwd(p *powerline) (segments []pwl.Segment) {
	cwd := p.cwd

	switch p.cfg.CwdMode {
	case "plain":
		// Plain mode goes through the same segmentation and aliasing as the other
		// modes and is rejoined afterwards, so "~" abbreviation, path
		// normalisation and -path-aliases cannot drift between modes. See #406.
		segments = append(segments, pwl.Segment{
			Name:       "cwd",
			Content:    plainPath(cwd, cwdToPathSegments(p, cwd)),
			Foreground: p.theme.CwdFg,
			Background: p.theme.PathBg,
		})
	default:
		pathSegments := cwdToPathSegments(p, cwd)

		if p.cfg.CwdMode == "dironly" {
			pathSegments = pathSegments[len(pathSegments)-1:]
		} else {
			maxDepth := p.cfg.CwdMaxDepth
			if maxDepth <= 0 {
				warn("Ignoring -cwd-max-depth argument since it's smaller than or equal to 0")
			} else if len(pathSegments) > maxDepth {
				var nBefore int
				if maxDepth > 2 {
					nBefore = 2
				} else {
					nBefore = maxDepth - 1
				}
				firstPart := pathSegments[:nBefore]
				secondPart := pathSegments[len(pathSegments)+nBefore-maxDepth:]

				pathSegments = make([]pathSegment, 0)
				pathSegments = append(pathSegments, firstPart...)
				pathSegments = append(pathSegments, pathSegment{
					path:     ellipsis,
					ellipsis: true,
				})
				pathSegments = append(pathSegments, secondPart...)
			}

			if p.cfg.CwdMode == "semifancy" && len(pathSegments) > 1 {
				var path string
				for idx, pathSegment := range pathSegments {
					if pathSegment.home || pathSegment.alias {
						continue
					}
					path += pathSegment.path
					if idx != len(pathSegments)-1 {
						path += string(os.PathSeparator)
					}
				}
				first := pathSegments[0]
				pathSegments = make([]pathSegment, 0)
				if first.home || first.alias {
					pathSegments = append(pathSegments, first)
				}
				pathSegments = append(pathSegments, pathSegment{
					path: path,
				})
			}
		}

		for idx, pathSegment := range pathSegments {
			isLastDir := idx == len(pathSegments)-1
			foreground, background, special := getColor(p, pathSegment, isLastDir)

			segment := pwl.Segment{
				Content:    maybeShortenName(p, pathSegment.path),
				Foreground: foreground,
				Background: background,
			}

			if !special {
				if p.align == alignRight && p.supportsRightModules() && idx != 0 {
					segment.Separator = p.symbols.SeparatorReverseThin
					segment.SeparatorForeground = p.theme.SeparatorFg
				} else if (p.align == alignLeft || !p.supportsRightModules()) && !isLastDir {
					segment.Separator = p.symbols.SeparatorThin
					segment.SeparatorForeground = p.theme.SeparatorFg
				}
			}

			segment.Name = "cwd-path"
			if isLastDir {
				segment.Name = "cwd"
			}

			segments = append(segments, segment)
		}
	}
	return segments
}

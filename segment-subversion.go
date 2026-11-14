package main

import (
	"strings"

	pwl "github.com/mrevjd/powerline-go/powerline"
)

func parseSvnInfo() (map[string]string, error) {
	info, err := runCommand("svn", "info")
	if err != nil {
		return nil, err
	}
	svnInfo := make(map[string]string)
	for _, line := range strings.Split(info, "\n") {
		if key, value, ok := strings.Cut(line, ": "); ok {
			svnInfo[key] = value
		}
	}
	return svnInfo, nil
}

// parseSvnStatus counts the entries of `svn status -u`. Its first nine columns
// are one status code each; other reports any change those counts do not
// cover (property changes, locks, switched paths and so on), which still makes
// the working copy dirty.
func parseSvnStatus() (stats repoStats, other int) {
	out, err := runCommand("svn", "status", "-u")
	if err != nil {
		return stats, 0
	}
	for _, line := range strings.Split(out, "\n") {
		// -u appends a "Status against revision" footer, which is not an entry.
		if len(line) < 9 || strings.HasPrefix(line, "Status against revision") {
			continue
		}
		switch line[0] {
		case '?':
			stats.untracked++
		case 'C':
			stats.conflicted++
		case 'A', 'D', 'M':
			stats.notStaged++
		case ' ':
		default:
			other++
		}
		switch line[1] {
		case 'C':
			stats.conflicted++
		case 'M':
			stats.notStaged++
		case ' ':
		default:
			other++
		}
		for _, c := range line[2:8] {
			if c != ' ' {
				other++
			}
		}
		switch line[8] {
		case '*':
			stats.behind++
		case ' ':
		default:
			other++
		}
	}
	return stats, other
}

func segmentSubversion(p *powerline) []pwl.Segment {
	svnInfo, err := parseSvnInfo()
	if err != nil || p.ignoreRepos[svnInfo["URL"]] || p.ignoreRepos[svnInfo["Relative URL"]] {
		return []pwl.Segment{}
	}

	stats, other := parseSvnStatus()
	foreground, background := p.theme.RepoCleanFg, p.theme.RepoCleanBg
	if stats.dirty() || other > 0 {
		foreground, background = p.theme.RepoDirtyFg, p.theme.RepoDirtyBg
	}

	return append([]pwl.Segment{{
		Name:       "svn-branch",
		Content:    svnInfo["Relative URL"],
		Foreground: foreground,
		Background: background,
	}}, stats.segments(p, "svn-status")...)
}

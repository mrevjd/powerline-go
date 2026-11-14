package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	pwl "github.com/mrevjd/powerline-go/powerline"
)

type repoStats struct {
	ahead      int
	behind     int
	untracked  int
	notStaged  int
	staged     int
	conflicted int
	stashed    int
}

func (r repoStats) dirty() bool {
	return r.untracked+r.notStaged+r.staged+r.conflicted > 0
}

func (r repoStats) any() bool {
	return r.ahead+r.behind+r.untracked+r.notStaged+r.staged+r.conflicted+r.stashed > 0
}

type repoStat struct {
	count  int
	symbol string
	fg, bg uint8
}

// list returns every stat with its symbol and colours, in display order.
func (r repoStats) list(p *powerline) []repoStat {
	s, t := p.symbols, p.theme
	return []repoStat{
		{r.ahead, s.RepoAhead, t.GitAheadFg, t.GitAheadBg},
		{r.behind, s.RepoBehind, t.GitBehindFg, t.GitBehindBg},
		{r.staged, s.RepoStaged, t.GitStagedFg, t.GitStagedBg},
		{r.notStaged, s.RepoNotStaged, t.GitNotStagedFg, t.GitNotStagedBg},
		{r.untracked, s.RepoUntracked, t.GitUntrackedFg, t.GitUntrackedBg},
		{r.conflicted, s.RepoConflicted, t.GitConflictedFg, t.GitConflictedBg},
		{r.stashed, s.RepoStashed, t.GitStashedFg, t.GitStashedBg},
	}
}

// segments renders each non-zero stat as its own segment, for the fancy mode.
func (r repoStats) segments(p *powerline, name string) []pwl.Segment {
	var segments []pwl.Segment
	for _, stat := range r.list(p) {
		if stat.count > 0 {
			segments = append(segments, pwl.Segment{
				Name:       name,
				Content:    fmt.Sprintf("%d%s", stat.count, stat.symbol),
				Foreground: stat.fg,
				Background: stat.bg,
			})
		}
	}
	return segments
}

// summary renders the non-zero stats inline: with their counts in compact mode,
// as bare symbols otherwise.
func (r repoStats) summary(p *powerline) string {
	var b strings.Builder
	for _, stat := range r.list(p) {
		if stat.count == 0 {
			continue
		}
		if p.cfg.GitMode == "compact" {
			fmt.Fprintf(&b, " %d%s", stat.count, stat.symbol)
		} else {
			b.WriteString(stat.symbol)
		}
	}
	return b.String()
}

var branchRegex = regexp.MustCompile(`^## (?P<local>\S+?)(\.{3}(?P<remote>\S+?)( \[(ahead (?P<ahead>\d+)(, )?)?(behind (?P<behind>\d+))?])?)?$`)

func groupDict(pattern *regexp.Regexp, haystack string) map[string]string {
	result := make(map[string]string)
	if match := pattern.FindStringSubmatch(haystack); match != nil {
		for i, name := range pattern.SubexpNames()[1:] {
			result[name] = match[i+1]
		}
	}
	return result
}

// gitProcessEnv runs git with a minimal environment, so user configuration such
// as LANG cannot change the output being parsed.
var gitProcessEnv = []string{
	"LANG=C",
	homeEnvName() + "=" + os.Getenv(homeEnvName()),
	"PATH=" + os.Getenv("PATH"),
}

func runGitCommand(cmd string, args ...string) (string, error) {
	command := exec.Command(cmd, args...)
	command.Env = gitProcessEnv
	out, err := command.Output()
	return string(out), err
}

// startBackgroundFetch refreshes the remote-tracking ref that ahead/behind is
// counted against, without making the prompt wait on the network: the fetch is
// detached and outlives this process, so its result shows on a later prompt.
func startBackgroundFetch(interval, timeout time.Duration) {
	out, err := runGitCommand("git", "--no-optional-locks", "rev-parse", "--git-common-dir")
	if err != nil {
		return
	}
	// Stamped before fetching, not on success, so a remote you are not
	// authenticated to is retried once per interval rather than every prompt.
	stamp := filepath.Join(strings.TrimSpace(out), "powerline-go-fetch")
	if info, err := os.Stat(stamp); err == nil && time.Since(info.ModTime()) < interval {
		return
	}
	if err := os.WriteFile(stamp, nil, 0o644); err != nil {
		return
	}
	now := time.Now()
	_ = os.Chtimes(stamp, now, now)

	// The fetch is supervised by a detached copy of powerline-go, since nothing
	// else survives this process to enforce a deadline on it.
	self, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(self, backgroundFetchArg, timeout.String())
	// Full environment, unlike gitProcessEnv, so the SSH agent and credential
	// helpers are reachable. Nothing may prompt: with no terminal and every
	// askpass blanked (an IDE terminal exports GIT_ASKPASS), a fetch needing
	// credentials you have not already provided just fails.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=", "SSH_ASKPASS=", "GCM_INTERACTIVE=never", "SSH_ASKPASS_REQUIRE=never")
	detachProcess(cmd)
	if cmd.Start() == nil {
		_ = cmd.Process.Release()
	}
}

// backgroundFetchArg, as the first argument, makes powerline-go run
// runBackgroundFetch instead of drawing a prompt.
const backgroundFetchArg = "__powerline-go-background-fetch"

// runBackgroundFetch fetches, killing the fetch if it outlives timeout, which
// bounds a fetch stalled on a dead network.
func runBackgroundFetch(timeout string) {
	d, err := time.ParseDuration(timeout)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	// --no-write-fetch-head leaves FETCH_HEAD to the user's own fetches.
	cmd := exec.CommandContext(ctx, "git", "-c", "credential.interactive=false", "fetch", "--quiet", "--no-tags", "--no-write-fetch-head")
	cmd.Cancel = func() error { return killFetch(cmd) }
	_ = cmd.Run()
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

func getGitDetachedBranch(p *powerline) string {
	out, err := runGitCommand("git", "--no-optional-locks", "rev-parse", "--short", "HEAD")
	if err == nil {
		return p.symbols.RepoDetached + " " + firstLine(out)
	}
	out, err = runGitCommand("git", "--no-optional-locks", "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "Error"
	}
	return firstLine(out)
}

// parseGitStats counts the file entries of `git status --porcelain -b` output,
// whose first line is the branch header.
func parseGitStats(status []string) repoStats {
	stats := repoStats{}
	if len(status) < 2 {
		return stats
	}
	for _, line := range status[1:] {
		if len(line) <= 2 {
			continue
		}
		switch code := line[:2]; code {
		case "??":
			stats.untracked++
		case "DD", "AU", "UD", "UA", "DU", "AA", "UU":
			stats.conflicted++
		default:
			if code[0] != ' ' {
				stats.staged++
			}
			if code[1] != ' ' {
				stats.notStaged++
			}
		}
	}
	return stats
}

func repoRoot() (string, error) {
	out, err := runGitCommand("git", "--no-optional-locks", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func indexSize(root string) (int64, error) {
	fileInfo, err := os.Stat(filepath.Join(root, ".git", "index"))
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), nil
}

func segmentGit(p *powerline) []pwl.Segment {
	root, err := repoRoot()
	if err != nil || p.ignoreRepos[root] {
		return []pwl.Segment{}
	}

	args := []string{"--no-optional-locks", "status", "--porcelain", "-b", "--ignore-submodules"}
	if p.cfg.GitAssumeUnchangedSize > 0 {
		if size, _ := indexSize(root); size > p.cfg.GitAssumeUnchangedSize*1024 {
			args = append(args, "-uno")
		}
	}

	out, err := runGitCommand("git", args...)
	if err != nil {
		return []pwl.Segment{}
	}

	status := strings.Split(out, "\n")
	stats := parseGitStats(status)
	branchInfo := groupDict(branchRegex, status[0])

	if p.cfg.GitFetchInterval > 0 && branchInfo["remote"] != "" {
		interval := time.Duration(p.cfg.GitFetchInterval) * time.Minute
		// At least 10 minutes, so a slow but working fetch still lands when
		// the interval is short.
		startBackgroundFetch(interval, max(interval, 10*time.Minute))
	}

	var branch string
	if branchInfo["local"] != "" {
		stats.ahead, _ = strconv.Atoi(branchInfo["ahead"])
		stats.behind, _ = strconv.Atoi(branchInfo["behind"])
		branch = branchInfo["local"]
	} else {
		branch = getGitDetachedBranch(p)
	}
	if p.symbols.RepoBranch != "" {
		branch = p.symbols.RepoBranch + " " + branch
	}

	foreground, background := p.theme.RepoCleanFg, p.theme.RepoCleanBg
	if stats.dirty() {
		foreground, background = p.theme.RepoDirtyFg, p.theme.RepoDirtyBg
	}

	for _, stat := range p.cfg.GitDisableStats {
		switch stat {
		case "ahead":
			stats.ahead = 0
		case "behind":
			stats.behind = 0
		case "staged":
			stats.staged = 0
		case "notStaged":
			stats.notStaged = 0
		case "untracked":
			stats.untracked = 0
		case "conflicted":
			stats.conflicted = 0
		}
	}
	if !slices.Contains(p.cfg.GitDisableStats, "stashed") {
		if out, err := runGitCommand("git", "--no-optional-locks", "rev-list", "-g", "refs/stash"); err == nil {
			stats.stashed = strings.Count(out, "\n")
		}
	}

	segment := pwl.Segment{
		Name:       "git-branch",
		Content:    branch,
		Foreground: foreground,
		Background: background,
	}
	switch p.cfg.GitMode {
	case "simple":
		if stats.any() {
			segment.Content += " " + stats.summary(p)
		}
	case "compact":
		segment.Content += stats.summary(p)
	default: // fancy
		return append([]pwl.Segment{segment}, stats.segments(p, "git-status")...)
	}
	return []pwl.Segment{segment}
}

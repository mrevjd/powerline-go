package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	pwl "github.com/justjanne/powerline-go/powerline"
)

// injection is a payload bash and zsh both evaluate if it reaches PS1/PROMPT
// unescaped. It contains no space or other character git rejects, so the same
// string also serves as a branch name.
const injection = "$(id)`whoami`"

// escapedInjection is what each shell's prompt must carry instead. "bare" is
// never expanded by a shell, so its escapes are the identity and the payload
// is expected to pass through as literal text.
var escapedInjection = map[string]string{
	"bash": `\$(id)\` + "`" + `whoami\` + "`",
	"zsh":  `\$(id)\` + "`" + `whoami\` + "`",
	"bare": injection,
}

func shells() []string { return []string{"bash", "zsh", "bare"} }

// renderPrompt builds the prompt the way main() does, so the assertions cover
// the whole path from segment to rendered prompt rather than one segment.
func renderPrompt(t *testing.T, shell string, modules ...string) string {
	t.Helper()
	cfg := defaults
	cfg.Shell = shell
	cfg.Modules = modules
	cfg.MaxWidthPercentage = 0 // never truncate: the payload must survive whole
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return newPowerline(cfg, cwd, alignLeft).draw()
}

func assertEscaped(t *testing.T, shell, prompt string) {
	t.Helper()
	want := escapedInjection[shell]
	if !strings.Contains(prompt, want) {
		t.Errorf("prompt is missing the escaped payload %q:\n%q", want, prompt)
	}
	if want != injection && strings.Contains(prompt, injection) {
		t.Errorf("prompt carries the payload %q unescaped:\n%q", injection, prompt)
	}
}

func requireBinary(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s is not available: %v", name, err)
	}
}

// A version string in a package.json belongs to whoever wrote the repository.
func Test_promptEscapesPackageVersion(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"version":` + strconv.Quote(injection) + `}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	for _, shell := range shells() {
		t.Run(shell, func(t *testing.T) {
			assertEscaped(t, shell, renderPrompt(t, shell, "node"))
		})
	}
}

// So does a branch name, which arrives with a `git clone` and needs no local
// checkout of the hostile branch to be displayed.
func Test_promptEscapesGitBranch(t *testing.T) {
	requireBinary(t, "git")

	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
			"GIT_AUTHOR_NAME=powerline-go",
			"GIT_AUTHOR_EMAIL=test@example.invalid",
			"GIT_COMMITTER_NAME=powerline-go",
			"GIT_COMMITTER_EMAIL=test@example.invalid",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	git("init", "--quiet", ".")
	git("commit", "--quiet", "--allow-empty", "-m", "init")
	git("branch", "--move", injection)
	t.Chdir(dir)

	for _, shell := range shells() {
		t.Run(shell, func(t *testing.T) {
			assertEscaped(t, shell, renderPrompt(t, shell, "git"))
		})
	}
}

// A plugin's output is JSON that unmarshals straight into Segment, so it must
// not be able to opt out of escaping by claiming to be a shell template.
func Test_promptEscapesPluginOutput(t *testing.T) {
	requireBinary(t, "sh")

	segments, err := json.Marshal([]pwl.Segment{{
		Name:          "inject-test",
		Content:       injection,
		ShellTemplate: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(segments), "'") {
		t.Fatalf("payload would break out of the script's single quotes: %s", segments)
	}

	dir := t.TempDir()
	plugin := filepath.Join(dir, "powerline-go-inject-test")
	script := "#!/bin/sh\nprintf '%s' '" + string(segments) + "'\n"
	if err := os.WriteFile(plugin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	for _, shell := range shells() {
		t.Run(shell, func(t *testing.T) {
			assertEscaped(t, shell, renderPrompt(t, shell, "inject-test"))
		})
	}
}

// The escaping must not reach the segments that are prompt templates. Only
// bash can regress here: its templates are backslash escapes, whereas zsh's
// are `%` sequences that carry nothing the escaping touches.
func Test_promptKeepsShellTemplates(t *testing.T) {
	t.Setenv("TERM", "xterm")
	prompt := renderPrompt(t, "bash", "user", "host", "root", "termtitle")

	for _, want := range []string{`\u`, `\h`, `\$`, `\[\e]0;`} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt is missing the bash template %q:\n%q", want, prompt)
		}
	}
	// An escaped template would show up as a doubled backslash; nothing bash
	// legitimately emits here contains one.
	if strings.Contains(prompt, `\\`) {
		t.Errorf("a bash prompt template was escaped:\n%q", prompt)
	}
}

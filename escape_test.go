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

// assertEscaped checks that payload reached the prompt in its escaped form and
// that the raw payload is not also there. A shell whose escaped form is the
// payload itself expands nothing, so for it only the first check applies.
func assertEscaped(t *testing.T, prompt, payload, want string) {
	t.Helper()
	if !strings.Contains(prompt, want) {
		t.Errorf("prompt is missing the escaped payload %q:\n%q", want, prompt)
	}
	if want != payload && strings.Contains(prompt, payload) {
		t.Errorf("prompt carries the payload %q unescaped:\n%q", payload, prompt)
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
			assertEscaped(t, renderPrompt(t, shell, "node"), injection, escapedInjection[shell])
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
			assertEscaped(t, renderPrompt(t, shell, "git"), injection, escapedInjection[shell])
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
			assertEscaped(t, renderPrompt(t, shell, "inject-test"), injection, escapedInjection[shell])
		})
	}
}

// zsh expands `%` sequences whenever it draws the prompt, with or without
// PROMPT_SUBST, so a directory, branch or venv name holding them is prompt
// markup rather than text. This is display corruption and prompt spoofing
// (%n and %m are the real username and host, %F{red} a colour, %(x.a.b) a
// conditional) rather than command execution, but it is the same class of bug
// as $ and a backtick, and the same fix. Confirmed against zsh 5.9: in a
// directory named `%n@%m` an unescaped prompt renders the real user and host.
const percentInjection = "%n@%m"

// bash's prompt expansion has no `%` sequences and "bare" expands nothing, so
// for both the payload is expected to pass through as literal text.
var escapedPercentInjection = map[string]string{
	"bash": percentInjection,
	"zsh":  `%%n@%%m`,
	"bare": percentInjection,
}

func Test_promptEscapesZshPromptSequences(t *testing.T) {
	dir := filepath.Join(t.TempDir(), percentInjection)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	for _, shell := range shells() {
		t.Run(shell, func(t *testing.T) {
			assertEscaped(t, renderPrompt(t, shell, "cwd"), percentInjection, escapedPercentInjection[shell])
		})
	}
}

// The escaping must not reach the segments that are prompt templates. Both
// shells can regress here, because the escaping rewrites the lead character of
// bash's backslash templates and of zsh's `%` templates alike.
func Test_promptKeepsShellTemplates(t *testing.T) {
	t.Setenv("TERM", "xterm")

	for _, tc := range []struct {
		shell     string
		templates []string
		// doubled is what an escaped template would show up as. Nothing the
		// shell legitimately emits in these segments contains one: zsh's
		// colour template renders as a single `%{`, bash's as `\[`.
		doubled string
	}{
		{"bash", []string{`\u`, `\h`, `\$`, `\[\e]0;`}, `\\`},
		{"zsh", []string{`%n`, `%m`, `%#`, "%{\033]0;%n@%m: %~"}, `%%`},
	} {
		t.Run(tc.shell, func(t *testing.T) {
			prompt := renderPrompt(t, tc.shell, "user", "host", "root", "termtitle")
			for _, want := range tc.templates {
				if !strings.Contains(prompt, want) {
					t.Errorf("prompt is missing the %s template %q:\n%q", tc.shell, want, prompt)
				}
			}
			if strings.Contains(prompt, tc.doubled) {
				t.Errorf("a %s prompt template was escaped (%q):\n%q", tc.shell, tc.doubled, prompt)
			}
		})
	}
}

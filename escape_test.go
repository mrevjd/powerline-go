package main

import (
	"fmt"
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
// not be able to opt out of escaping by claiming to be a shell template. The
// payload is a literal string rather than a marshalled Segment: the point is
// that ShellTemplate cannot be set over the wire, which a marshalled struct
// would stop expressing the moment the field is excluded from JSON.
func Test_promptEscapesPluginOutput(t *testing.T) {
	requireBinary(t, "sh")

	payload := `[{"Name":"inject-test","Content":"` + injection + `","ShellTemplate":true}]`
	if strings.Contains(payload, "'") {
		t.Fatalf("payload would break out of the script's single quotes: %s", payload)
	}

	dir := t.TempDir()
	plugin := filepath.Join(dir, "powerline-go-inject-test")
	script := "#!/bin/sh\nprintf '%s' '" + payload + "'\n"
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

// Every other test compares against an expected string, which only stands in
// for "bash will not run this". This one asks bash. Without it, replacing
// EscapedDollar with something that does not hold (`\044`, say, which bash
// decodes and then expands) would fail the table test on a string mismatch,
// and updating the expected string would turn the suite green again over a
// table that executes.
func Test_bashDoesNotExecuteSegmentContent(t *testing.T) {
	requireBinary(t, "bash")

	dir := t.TempDir()
	pkg := `{"version":` + strconv.Quote(injection) + `}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	// ${PS1@P} is bash's own prompt expansion: the backslash decoding, and the
	// word expansion that runs a substitution when promptvars is on.
	prompt := renderPrompt(t, "bash", "node")
	out, err := exec.Command("bash", "--norc", "--noprofile", "-c",
		`PS1="$1"; printf '%s' "${PS1@P}"`, "bash", prompt).Output()
	if err != nil {
		t.Fatalf("bash: %v", err)
	}
	// Assert the payload survives rather than that some marker is absent: the
	// marker a substitution would print also appears inside the literal.
	if rendered := string(out); !strings.Contains(rendered, injection) {
		t.Errorf("bash did not render %q literally: %q", injection, rendered)
	}
}

// A plugin can set Separator as well as Content, so it is the second way into
// the prompt and has to be escaped the same way.
func Test_promptEscapesPluginSeparator(t *testing.T) {
	requireBinary(t, "sh")

	payload := `[{"Name":"sep-test","Content":"hello","Separator":"` + injection + `"}]`
	dir := t.TempDir()
	plugin := filepath.Join(dir, "powerline-go-sep-test")
	script := "#!/bin/sh\nprintf '%s' '" + payload + "'\n"
	if err := os.WriteFile(plugin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	for _, shell := range shells() {
		t.Run(shell, func(t *testing.T) {
			assertEscaped(t, renderPrompt(t, shell, "sep-test"), injection, escapedInjection[shell])
		})
	}
}

// The user and host segments hand bash and zsh a template and everything else
// plain text. The plain-text branches carry data that is not entirely ours:
// a Windows/AD username contains a backslash, and a hostname is attacker
// influenced in a container or over DHCP.
func Test_userAndHostShellTemplateDefaults(t *testing.T) {
	segments := map[string]func(*powerline) []pwl.Segment{
		"user": segmentUser,
		"host": segmentHost,
	}

	for name, segment := range segments {
		for _, shell := range []string{"bash", "zsh"} {
			t.Run(name+"/"+shell, func(t *testing.T) {
				p := &powerline{cfg: Config{Shell: shell}, username: "u", hostname: "h"}
				segs := segment(p)
				if len(segs) != 1 || !segs[0].ShellTemplate {
					t.Errorf("%s under %s hands the shell a template, which must reach it verbatim", name, shell)
				}
			})
		}
		for _, shell := range []string{"autodetect", "bare", "fish"} {
			t.Run(name+"/"+shell, func(t *testing.T) {
				p := &powerline{cfg: Config{Shell: shell}, username: "u", hostname: "h"}
				segs := segment(p)
				if len(segs) != 1 {
					t.Fatalf("%s returned %d segments, want 1", name, len(segs))
				}
				if segs[0].ShellTemplate {
					t.Errorf("%s under %s reports plain text, so it must not be exempt from escaping", name, shell)
				}
			})
		}
	}

	// -colorize-hostname reports the hostname as text whatever the shell is.
	t.Run("host/colorized", func(t *testing.T) {
		p := &powerline{cfg: Config{Shell: "bash", ColorizeHostname: true}, hostname: "h"}
		segs := segmentHost(p)
		if len(segs) != 1 || segs[0].ShellTemplate {
			t.Error("a colorized hostname is text, so it must not be exempt from escaping")
		}
	})
}

// The termtitle fallback interpolates the cwd instead of handing the shell a
// template, so it must not claim the ShellTemplate exemption. It is the branch
// bash and zsh users actually reach, because p.cfg.Shell stays "autodetect"
// when the shell was detected rather than passed with -shell.
func Test_termTitleFallbackIsNotAShellTemplate(t *testing.T) {
	t.Setenv("TERM", "xterm")

	for _, shell := range []string{"autodetect", "bare", "fish"} {
		t.Run(shell, func(t *testing.T) {
			p := &powerline{cfg: Config{Shell: shell}, cwd: "/tmp/" + injection}
			segs := segmentTermTitle(p)
			if len(segs) != 1 {
				t.Fatalf("segmentTermTitle returned %d segments, want 1", len(segs))
			}
			if !strings.Contains(segs[0].Content, injection) {
				t.Fatalf("expected the fallback to interpolate the cwd: %q", segs[0].Content)
			}
			if segs[0].ShellTemplate {
				t.Error("the fallback interpolates the cwd, so it must not be exempt from escaping")
			}
		})
	}

	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			p := &powerline{cfg: Config{Shell: shell}, cwd: "/tmp/" + injection}
			segs := segmentTermTitle(p)
			if len(segs) != 1 || !segs[0].ShellTemplate {
				t.Errorf("%s hands the shell a title template, which must reach it verbatim", shell)
			}
		})
	}
}

// The same thing end to end, against the mismatch that made it exploitable:
// p.cfg.Shell says "autodetect" while the prompt is really being written for
// bash. Rendering the fallback title must still escape the cwd.
func Test_termTitleFallbackEscapesCwdUnderBash(t *testing.T) {
	t.Setenv("TERM", "xterm")

	p := &powerline{
		cfg:      Config{Shell: "autodetect"},
		shell:    defaults.Shells["bash"],
		theme:    defaults.Themes[defaults.Theme],
		symbols:  defaults.Modes[defaults.Mode],
		cwd:      "/tmp/" + injection,
		Segments: make([][]pwl.Segment, 1),
	}
	p.reset = fmt.Sprintf(p.shell.ColorTemplate, "[0m")
	for _, s := range segmentTermTitle(p) {
		p.appendSegment(s.Name, s)
	}
	assertEscaped(t, p.draw(), injection, escapedInjection["bash"])
}

// The render tests cannot carry a backslash: git rejects one in a ref name, so
// the shared payload leaves EscapedBackslash untested and hides the one place
// the bash and zsh tables genuinely differ.
func Test_escapeVariables(t *testing.T) {
	tests := []struct {
		shell, text, want string
	}{
		{"bash", `a\b`, `a\\\\b`},
		{"zsh", `a\b`, `a\\b`},
		{"bare", `a\b`, `a\b`},
		{"bash", "a`b", "a\\`b"},
		{"zsh", "a`b", "a\\`b"},
		{"bare", "a`b", "a`b"},
		{"bash", `a$b`, `a\$b`},
		{"zsh", `a$b`, `a\$b`},
		{"bare", `a$b`, `a$b`},
		{"bash", `a%b`, `a%b`},
		{"zsh", `a%b`, `a%%b`},
		{"bare", `a%b`, `a%b`},
		// The backslash is replaced first, so the backslashes the later two
		// introduce are not escaped again. An already-escaped payload must come
		// out escaped exactly once more, not twice.
		{"bash", `\$`, `\\\\\$`},
		{"zsh", `\$`, `\\\$`},
	}

	for _, tt := range tests {
		t.Run(tt.shell+"/"+tt.text, func(t *testing.T) {
			p := &powerline{shell: defaults.Shells[tt.shell]}
			if got := p.escapeVariables(tt.text); got != tt.want {
				t.Errorf("escapeVariables(%q) for %s = %q, want %q", tt.text, tt.shell, got, tt.want)
			}
		})
	}
}

// A shell whose Shells entry carries no escape table drops the characters
// rather than passing them through. A config that overrides one field of a
// built-in shell produces exactly that entry, and letting the characters
// through would switch escaping off without saying so. newPowerline warns
// about the same entry, so the loss is not silent either.
func Test_escapeVariablesWithoutAnEscapeTable(t *testing.T) {
	p := &powerline{shell: ShellInfo{RootIndicator: "#"}}
	if got := p.escapeVariables(injection); got != "(id)whoami" {
		t.Errorf("escapeVariables(%q) = %q, want the metacharacters dropped", injection, got)
	}
}

// Every built-in shell must carry a complete escape table, or escapeVariables
// silently drops what it is meant to escape.
func Test_builtinShellsAllDefineAnEscapeTable(t *testing.T) {
	for name, shell := range defaults.Shells {
		if shell.EscapedBackslash == "" || shell.EscapedBacktick == "" ||
			shell.EscapedDollar == "" || shell.EscapedPercent == "" {
			t.Errorf("built-in shell %q has an incomplete escape table: %+v", name, shell)
		}
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

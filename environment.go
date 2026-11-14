package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func homeEnvName() string {
	if runtime.GOOS == "windows" {
		return "USERPROFILE"
	}
	return "HOME"
}

// runCommand runs a tool with the inherited environment and returns its stdout.
func runCommand(cmd string, args ...string) (string, error) {
	out, err := exec.Command(cmd, args...).Output()
	return string(out), err
}

// readTrimmed returns the contents of path without surrounding whitespace.
func readTrimmed(path string) (string, bool) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(contents)), true
}

// readFileUpwards looks for name in dir and then each of its parents, and
// returns the trimmed contents of the first one it can read.
func readFileUpwards(dir, name string) (string, bool) {
	for {
		if contents, ok := readTrimmed(filepath.Join(dir, name)); ok {
			return contents, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

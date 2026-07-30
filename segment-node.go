package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	pwl "github.com/justjanne/powerline-go/powerline"
)

const pkgfile = "./package.json"

type packageJSON struct {
	Version string `json:"version"`
}

func getNodeVersion() string {
	out, err := exec.Command("node", "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(out), "\n")
}

// readPackageJSON returns the version declared in ./package.json, and whether the
// file exists at all. Both answers come from one stat: the caller needs the
// existence check to decide whether this is a node project (#356) and the version
// to render, and this runs on every prompt.
func readPackageJSON() (string, bool) {
	stat, err := os.Stat(pkgfile)
	if err != nil || stat.IsDir() {
		return "", false
	}
	raw, err := os.ReadFile(pkgfile)
	if err != nil {
		return "", true
	}
	pkg := packageJSON{}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return "", true
	}

	return strings.TrimSpace(pkg.Version), true
}

func segmentNode(p *powerline) []pwl.Segment {
	segments := []pwl.Segment{}

	// Only surface node info inside a node project (package.json present).
	// Otherwise the segment showed on every prompt merely because `node` was
	// on PATH. See #356.
	packageVersion, inNodeProject := readPackageJSON()
	if !inNodeProject {
		return segments
	}

	nodeVersion := getNodeVersion()

	if nodeVersion != "" {
		segments = append(segments, pwl.Segment{
			Name:       "node",
			Content:    p.symbols.NodeIndicator + " " + nodeVersion,
			Foreground: p.theme.NodeVersionFg,
			Background: p.theme.NodeVersionBg,
		})
	}

	if packageVersion != "" {
		segments = append(segments, pwl.Segment{
			Name:       "node-segment",
			Content:    packageVersion + " " + p.symbols.NodeIndicator,
			Foreground: p.theme.NodeFg,
			Background: p.theme.NodeBg,
		})
	}

	return segments
}

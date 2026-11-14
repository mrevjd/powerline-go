package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	pwl "github.com/justjanne/powerline-go/powerline"
)

func getCloudConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "gcloud"), nil
	}
	return filepath.Join(home, ".config", "gcloud"), nil
}

// getActiveGCloudConfig names the active configuration. gcloud only writes
// active_config once a second configuration exists, so without a readable one
// the active configuration is "default".
func getActiveGCloudConfig(configDir string) string {
	if config, ok := readTrimmed(filepath.Join(configDir, "active_config")); ok && config != "" {
		return config
	}
	return "default"
}

func getGCPProjectFromGCloud() (string, error) {
	out, err := exec.Command("gcloud", "config", "list", "project", "--format", "value(core.project)").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

func getGCPProjectFromFile() (string, error) {
	configDir, err := getCloudConfigDir()
	if err != nil {
		return "", err
	}
	configPath := filepath.Join(configDir, "configurations", "config_"+getActiveGCloudConfig(configDir))
	stat, err := os.Stat(configPath)
	if err != nil {
		return "", err
	}
	if stat.IsDir() {
		return "", fmt.Errorf("%s is a directory", configPath)
	}
	file, err := os.Open(configPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	foundCore, inCore := false, false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			inCore = line == "[core]"
			foundCore = foundCore || inCore
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if inCore && ok && strings.TrimSpace(key) == "project" {
			return strings.TrimSpace(value), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if !foundCore {
		return "", fmt.Errorf("could not find [core] section in %s", configPath)
	}
	return "", nil
}

func getGCPProject() (string, error) {
	if project, err := getGCPProjectFromFile(); err == nil {
		return project, nil
	}
	return getGCPProjectFromGCloud()
}

func segmentGCP(p *powerline) []pwl.Segment {
	// No readable config and no gcloud on PATH just means there is no project to
	// show. This used to log.Fatal, which took the whole prompt down with it.
	project, err := getGCPProject()
	if err != nil || project == "" {
		return []pwl.Segment{}
	}
	return []pwl.Segment{{
		Name:       "gcp",
		Content:    project,
		Foreground: p.theme.GCPFg,
		Background: p.theme.GCPBg,
	}}
}

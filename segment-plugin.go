package main

import (
	"encoding/json"
	"os/exec"

	pwl "github.com/justjanne/powerline-go/powerline"
)

func segmentPlugin(p *powerline, plugin string) ([]pwl.Segment, bool) {
	output, err := exec.Command("powerline-go-" + plugin).Output()
	if err != nil {
		return nil, false
	}
	segments := []pwl.Segment{}
	err = json.Unmarshal(output, &segments)
	if err != nil {
		// The plugin was found but no valid data was returned. Ignore it
		return []pwl.Segment{}, true
	}
	// A plugin reports data, and whatever it reports on - a ticket title, a
	// cluster name, a branch - can come from somewhere the plugin does not
	// control. Clear ShellTemplate so its content is always escaped: writing a
	// prompt template is the prompt's job, not a plugin's.
	for i := range segments {
		segments[i].ShellTemplate = false
	}
	return segments, true
}

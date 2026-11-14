package main

import (
	"testing"

	"github.com/shirou/gopsutil/v3/load"
)

// The theme's LoadAvgValue picks which average is compared against the
// threshold, and it must be the one displayed too.
func TestLoadSegmentShowsSelectedAverage(t *testing.T) {
	avg := &load.AvgStat{Load1: 1, Load5: 5, Load15: 15}
	for which, want := range map[byte]string{1: "1.00", 5: "5.00", 15: "15.00", 0: "5.00"} {
		p := &powerline{theme: Theme{LoadAvgValue: which, LoadThresholdBad: 100}}
		segs := loadSegment(p, avg, 1)
		if len(segs) != 1 || segs[0].Content != want {
			t.Errorf("LoadAvgValue %d: got %v, want content %q", which, segs, want)
		}
	}
}

package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Caption is one line of a captions file: a view-time stamp and its text.
// View time is the raw take's time with its idle_time_limit applied, which is
// what the annotate viewer's clock shows.
type Caption struct {
	At   float64
	Text string
}

var stampRe = regexp.MustCompile(`^~?(\d+):(\d+(?:\.\d+)?)\s+(.+)$`)

// parseCaptions reads `~m:ss.s  text` lines, skipping blank ones, and returns
// them sorted by time (then text, so the order never depends on the file's).
func parseCaptions(path string, data []byte) ([]Caption, error) {
	var caps []Caption
	for i, line := range splitLines(data) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := stampRe.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("%s:%d: expected `~m:ss.s  text`, got %q", path, i+1, line)
		}
		// Bounded at the door, like every duration castcut reads (maxSeconds).
		min, err := strconv.Atoi(m[1])
		sec, _ := strconv.ParseFloat(m[2], 64)
		if err != nil || min > maxSeconds/60 || !inRange(float64(min*60)+sec) {
			return nil, fmt.Errorf("%s:%d: stamp %s:%s is past %d hours", path, i+1, m[1], m[2], maxSeconds/3600)
		}
		caps = append(caps, Caption{At: float64(min*60) + sec, Text: strings.TrimSpace(m[3])})
	}
	sort.SliceStable(caps, func(i, j int) bool {
		if caps[i].At != caps[j].At {
			return caps[i].At < caps[j].At
		}
		return caps[i].Text < caps[j].Text
	})
	return caps, nil
}

// sidecarPath is where annotate keeps a take's notes and where cut looks for
// captions by default: take.cast → take.captions.txt.
func sidecarPath(cast string) string {
	return strings.TrimSuffix(cast, filepath.Ext(cast)) + ".captions.txt"
}

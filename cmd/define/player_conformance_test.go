//go:build darwin && conformance

package main

// Live conformance for the afplay seam (ARCH-MOCK).
//
// The assumption everything else rests on is that afplay BLOCKS until playback
// finishes. If it ever returned immediately, three overlapping sounds would
// still satisfy fakePlayer's count and every other test in this repo — the "play
// it three times" criterion would be met on paper and wrong in the room.
//
//	go test -tags conformance -run AfplayBlocks ./cmd/define/

import (
	"os"
	"testing"
	"time"
)

func TestAfplayBlocksUntilPlaybackCompletes(t *testing.T) {
	data, from, err := newHTTPAudioSource().Fetch(t.Context(), AudioCandidates("sycophantic", "us"))
	if err != nil {
		skipOrFail(t, "network unavailable", err)
	}
	dir := t.TempDir()
	path := dir + "/pronunciation.mp3"
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	if err := (afplayPlayer{}).Play(t.Context(), path); err != nil {
		skipOrFail(t, "afplay unavailable", err)
	}
	single := time.Since(start)

	// A recording is a few hundred milliseconds at minimum; returning faster than
	// that means afplay is not blocking and playN's repeat loop is a fiction.
	if single < 200*time.Millisecond {
		t.Fatalf("afplay returned after %v for %s — it is not blocking until playback completes", single, from)
	}

	// Three plays must take materially longer than one. This is the property the
	// acceptance criterion actually depends on.
	start = time.Now()
	if err := playN(t.Context(), afplayPlayer{}, path, 3); err != nil {
		t.Fatal(err)
	}
	if triple := time.Since(start); triple < 2*single {
		t.Errorf("3 plays took %v, 1 took %v — playback is overlapping, not repeating", triple, single)
	}
}

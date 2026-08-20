package main

import (
	"context"
	"fmt"
	"sync"
)

// fakePlayer records every play across calls, which is what makes "play it three
// times" an assertion instead of something you check by ear. FailOn arms it to
// fail on the Nth play so partial-playback handling is testable.
type fakePlayer struct {
	mu     sync.Mutex
	Played []string
	FailOn int // 1-based; 0 disables
}

func (f *fakePlayer) Play(_ context.Context, path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Played = append(f.Played, path)
	if f.FailOn > 0 && len(f.Played) == f.FailOn {
		return fmt.Errorf("fake player failed on play %d", f.FailOn)
	}
	return nil
}

func (f *fakePlayer) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.Played)
}

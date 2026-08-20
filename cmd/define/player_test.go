package main

import (
	"context"
	"testing"
	"time"
)

// The acceptance criterion from the issue: it plays three times.
func TestPlayNPlaysRequestedNumberOfTimes(t *testing.T) {
	p := &fakePlayer{}
	if err := playN(t.Context(), p, "/tmp/x.mp3", 3); err != nil {
		t.Fatalf("playN: %v", err)
	}
	if got := p.count(); got != 3 {
		t.Errorf("played %d times, want 3", got)
	}
	for i, path := range p.Played {
		if path != "/tmp/x.mp3" {
			t.Errorf("play %d used %q", i, path)
		}
	}
}

func TestPlayNStopsOnError(t *testing.T) {
	p := &fakePlayer{FailOn: 2}
	if err := playN(t.Context(), p, "/tmp/x.mp3", 3); err == nil {
		t.Fatal("want an error from the failing player")
	}
	if got := p.count(); got != 2 {
		t.Errorf("played %d times, want 2 (attempted 1 and 2, stopped before 3)", got)
	}
}

func TestPlayNZeroTimes(t *testing.T) {
	p := &fakePlayer{}
	if err := playN(t.Context(), p, "/tmp/x.mp3", 0); err != nil {
		t.Fatal(err)
	}
	if got := p.count(); got != 0 {
		t.Errorf("played %d times, want 0", got)
	}
}

// The gap must not be paid after the final play — that would add latency to
// every invocation for no benefit.
func TestPlayNDoesNotPauseAfterLastPlay(t *testing.T) {
	start := time.Now()
	if err := playN(t.Context(), &fakePlayer{}, "/tmp/x.mp3", 1); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed >= gapBetweenPlays {
		t.Errorf("single play took %v — a trailing gap was paid", elapsed)
	}
}

func TestPlayNHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	p := &fakePlayer{}
	cancel()
	if err := playN(ctx, p, "/tmp/x.mp3", 3); err == nil {
		t.Error("want a context error")
	}
	if got := p.count(); got > 1 {
		t.Errorf("played %d times after cancellation", got)
	}
}

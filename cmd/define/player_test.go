package main

import (
	"bytes"
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

// playAnnounced applies the audio-off guard ITSELF (#38 T0).
//
// The predicate was hand-copied at four sites in two spellings, always ABOVE
// this call — so no test could reach here with audio off, and a fifth caller
// written below the guard would have fetched audio a `-no-audio` session asked
// not to have. playAnnounced's own doc comment already named that divergence as
// one it existed to end; it had ended the other two.
//
// This is the row none of the existing audio tests could be, because they all
// go through callers that stop first.
func TestPlayAnnouncedFetchesNothingWithAudioOff(t *testing.T) {
	for _, tc := range []struct {
		name string
		opt  options
	}{
		{"-no-audio", options{noAudio: true, times: 3, locale: "us"}},
		{"--sound 0", options{times: 0, locale: "us"}},
		{"a negative count is not a request to play", options{times: -1, locale: "us"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rig := newAudioRig(t, "sycophantic", true)
			var out, errb bytes.Buffer
			got := playAnnounced(t.Context(), rig.deps, tc.opt,
				utteranceFor("sycophantic", "", "", tc.opt),
				indicator{show: true, erase: eraseLine}, &out, &errb)

			if got {
				t.Error("reported that playback happened")
			}
			if n := len(rig.cdn.Requested()); n != 0 {
				t.Errorf("made %d CDN request(s) with audio off: %v", n, rig.cdn.Requested())
			}
			if rig.player.count() != 0 {
				t.Errorf("played %d times with audio off", rig.player.count())
			}
			if out.Len() != 0 {
				t.Errorf("announced something with audio off: %q", out.String())
			}
		})
	}
}

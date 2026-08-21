package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// decideCapture is a truth table over (found, noCapture, raw). The two rows that
// are DECISIONS rather than mechanics carry the reasoning.
func TestDecideCapture(t *testing.T) {
	tests := []struct {
		name  string
		found bool
		opt   options
		want  captureDecision
	}{
		{"found", true, options{}, captureEventAndWord},
		// A failed lookup is still history — #14 recalls the typo, #15 filters it —
		// but it is not vocabulary, so it never reaches the deck.
		{"not found", false, options{}, captureEventOnly},
		{"opted out", true, options{noCapture: true}, captureNothing},
		{"opted out and not found", false, options{noCapture: true}, captureNothing},
		// Scripting a dictionary must not mutate the deck it stands in.
		{"raw", true, options{raw: true}, captureNothing},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := decideCapture(tc.found, tc.opt); got != tc.want {
				t.Errorf("decideCapture(%v, %+v) = %v, want %v", tc.found, tc.opt, got, tc.want)
			}
		})
	}
}

func TestStoreCapturerWritesWhatWasDecided(t *testing.T) {
	st := store.NewMem()
	c := newStoreCapturer(st, store.FixedClock(time.Date(2026, 8, 21, 9, 0, 0, 0, time.UTC)), nil)

	c.Capture("sycophantic", true, options{})
	c.Capture("sykophantic", false, options{})

	deck, _ := st.Deck()
	if len(deck) != 1 || deck[0].Text != "sycophantic" {
		t.Errorf("deck = %+v, want only the word that exists", deck)
	}
	ev, _ := st.Events(time.Time{})
	if len(ev) != 2 {
		t.Errorf("got %d events, want 2 — a failed lookup is still history", len(ev))
	}
}

// Moved from #3's storeHistory tests with the writes they assert.
func TestStoreCapturerDegradesOnWriteFailure(t *testing.T) {
	var warn strings.Builder
	c := newStoreCapturer(failingStore{}, store.FixedClock(time.Now()), &warn)

	for i := 0; i < 5; i++ {
		c.Capture("sycophantic", true, options{})
	}
	if n := strings.Count(warn.String(), "define:"); n != 1 {
		t.Errorf("warned %d times over 5 failures, want exactly 1 — a directory that cannot be written is a standing condition, not news", n)
	}
}

// countingCapturer records every Capture call, so arity is assertable.
type countingCapturer struct {
	calls []string
	found []bool
}

func (c *countingCapturer) Capture(word string, found bool, _ options) {
	c.calls = append(c.calls, word)
	c.found = append(c.found, found)
}

// ONE lookup, ONE capture — on every entry path.
//
// Silent double-counting is this refactor's failure mode: the raw path used to
// record via storeHistory.Add and would now also record at the capture site. A
// deck that counts every lookup twice is wrong in a way nobody notices until #5
// orders by it.
func TestCaptureArityIsOnePerLookup(t *testing.T) {
	t.Run("one-shot", func(t *testing.T) {
		rig := newAudioRig(t, "sycophantic", true)
		c := &countingCapturer{}
		rig.deps.capture = c
		rig.deps.stdinIsTerminal = func() bool { return false }
		var out, errb bytes.Buffer
		run(t.Context(), []string{"-no-audio", "sycophantic"}, rig.deps, strings.NewReader(""), &out, &errb)
		assertCaptured(t, c, []string{"sycophantic"}, []bool{true})
	})

	t.Run("piped line loop", func(t *testing.T) {
		rig := newAudioRig(t, "sycophantic", true)
		c := &countingCapturer{}
		rig.deps.capture = c
		rig.deps.stdinIsTerminal = func() bool { return false }
		var out, errb bytes.Buffer
		run(t.Context(), []string{"-no-audio"}, rig.deps, strings.NewReader("sycophantic\n"), &out, &errb)
		assertCaptured(t, c, []string{"sycophantic"}, []bool{true})
	})

	t.Run("raw editor", func(t *testing.T) {
		rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
		c := &countingCapturer{}
		rig.deps.capture = c
		var out, errb bytes.Buffer
		runEditor(t.Context(), scriptKeys("sycophantic\r"), rig.deps, opt, cooked, finish, &out, &errb)
		assertCaptured(t, c, []string{"sycophantic"}, []bool{true})
	})

	// A bare Enter replays; it is not a new lookup and must capture nothing.
	t.Run("replay captures nothing", func(t *testing.T) {
		rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
		c := &countingCapturer{}
		rig.deps.capture = c
		var out, errb bytes.Buffer
		runEditor(t.Context(), scriptKeys("sycophantic\r\r\r"), rig.deps, opt, cooked, finish, &out, &errb)
		assertCaptured(t, c, []string{"sycophantic"}, []bool{true})
	})

	// A failed lookup captures exactly once, as not-found.
	t.Run("failed lookup", func(t *testing.T) {
		rig := newAudioRig(t, "sycophantic", true)
		c := &countingCapturer{}
		rig.deps.capture = c
		rig.deps.stdinIsTerminal = func() bool { return false }
		var out, errb bytes.Buffer
		run(t.Context(), []string{"-no-audio", "rizz"}, rig.deps, strings.NewReader(""), &out, &errb)
		assertCaptured(t, c, []string{"rizz"}, []bool{false})
	})
}

func assertCaptured(t *testing.T, c *countingCapturer, words []string, found []bool) {
	t.Helper()
	if len(c.calls) != len(words) {
		t.Fatalf("captured %d time(s) %v, want %d %v", len(c.calls), c.calls, len(words), words)
	}
	for i := range words {
		if c.calls[i] != words[i] || c.found[i] != found[i] {
			t.Errorf("capture %d = (%q, %v), want (%q, %v)", i, c.calls[i], c.found[i], words[i], found[i])
		}
	}
}

// --- --forget ---------------------------------------------------------------

func forgetRig(t *testing.T) (deps, *store.Mem) {
	t.Helper()
	st := store.NewMem()
	d := testDeps(t)
	d.deck = st
	return d, st
}

func TestForgetRemovesTheWordAndKeepsEvents(t *testing.T) {
	d, st := forgetRig(t)
	_ = st.Upsert(store.Word{Text: "sycophantic", LastSeen: time.Now()})
	_ = st.AppendEvent(store.ReviewEvent{Word: "sycophantic", Kind: store.EventLookedUp, Found: true, At: time.Now()})

	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-forget", "sycophantic"}, d, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}
	if deck, _ := st.Deck(); len(deck) != 0 {
		t.Errorf("deck = %+v, want empty", deck)
	}
	// The log is history. #8's statistics are a fold over it, so rewriting the
	// past to remove a word would corrupt every one of them.
	if ev, _ := st.Events(time.Time{}); len(ev) != 1 {
		t.Errorf("--forget deleted events; the log must be untouched")
	}
	if !strings.Contains(out.String(), "sycophantic") {
		t.Errorf("removal was not reported: %q", out.String())
	}
}

// Succeeding silently would hide a typo in the very command meant to fix one.
func TestForgetAbsentWordExitsNonZero(t *testing.T) {
	d, _ := forgetRig(t)
	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-forget", "never-seen"}, d, strings.NewReader(""), &out, &errb); code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "not in the deck") {
		t.Errorf("stderr = %q", errb.String())
	}
}

// -forget plus a word is two commands on one line. Honouring one silently is
// how -raw came to mean two different things in #2.
func TestForgetWithAWordIsAUsageError(t *testing.T) {
	d, _ := forgetRig(t)
	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-forget", "a", "b"}, d, strings.NewReader(""), &out, &errb); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

// --- the opt-out ------------------------------------------------------------

// DEFINE_NO_CAPTURE means "write nothing in this directory", so it must reach
// the policy as an input rather than as a second mechanism beside it.
func TestNoCaptureSuppressesEverything(t *testing.T) {
	st := store.NewMem()
	c := newStoreCapturer(st, store.FixedClock(time.Now()), nil)

	c.Capture("sycophantic", true, options{noCapture: true})

	if deck, _ := st.Deck(); len(deck) != 0 {
		t.Errorf("deck = %+v, want empty", deck)
	}
	// Events too — persisted history IS the event log, so opting out drops
	// history to session-only. That cost is documented beside the flag.
	if ev, _ := st.Events(time.Time{}); len(ev) != 0 {
		t.Errorf("got %d events, want 0", len(ev))
	}
}

package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
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

	// -raw: the policy is consulted and says "nothing". This is the row the truth
	// table asserts and production can actually produce — the previous round's
	// fix (calling Capture inside the raw branch) could be deleted with the whole
	// suite staying green.
	t.Run("raw captures nothing but still asks", func(t *testing.T) {
		rig := newAudioRig(t, "sycophantic", true)
		c := &countingCapturer{}
		rig.deps.capture = c
		rig.deps.stdinIsTerminal = func() bool { return false }
		var out, errb bytes.Buffer
		run(t.Context(), []string{"-raw", "-no-audio", "sycophantic"}, rig.deps, strings.NewReader(""), &out, &errb)
		// Asked exactly once — the policy, not an early return, decides.
		assertCaptured(t, c, []string{"sycophantic"}, []bool{true})
	})

	// ...and nothing reaches the store under -raw, through the real capturer.
	t.Run("raw writes nothing", func(t *testing.T) {
		st := store.NewMem()
		rig := newAudioRig(t, "sycophantic", true)
		rig.deps.capture = newStoreCapturer(st, store.FixedClock(time.Now()), nil)
		rig.deps.stdinIsTerminal = func() bool { return false }
		var out, errb bytes.Buffer
		run(t.Context(), []string{"-raw", "-no-audio", "sycophantic"}, rig.deps, strings.NewReader(""), &out, &errb)
		if deck, _ := st.Deck(); len(deck) != 0 {
			t.Errorf("-raw wrote to the deck: %+v", deck)
		}
		if ev, _ := st.Events(time.Time{}); len(ev) != 0 {
			t.Errorf("-raw wrote %d events", len(ev))
		}
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

// The arity test that can actually SEE a double write.
//
// The per-path subtests above inject a countingCapturer AT the seam, so they
// count Capture calls — which is a different question, and one they answer well.
// They cannot see the bug this issue's refactor risks: writes happening BELOW
// the seam, once via storeHistory and once via the capturer. Verified by
// restoring the old storeHistory.Add writes: this goes red, those stay green.
func TestNoDoubleWriteThroughTheRealWiring(t *testing.T) {
	dir := t.TempDir()
	st := store.NewYAML(dir, nil)

	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	rig.deps.history = newStoreHistory(st, nil)
	rig.deps.capture = newStoreCapturer(st, fixedClock(1), nil)

	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r"), rig.deps, opt, cooked, finish, &out, &errb)

	deck, err := st.Deck()
	if err != nil {
		t.Fatal(err)
	}
	if len(deck) != 1 {
		t.Fatalf("deck has %d entries, want 1: %+v", len(deck), deck)
	}
	if deck[0].Lookups != 1 {
		t.Errorf("Lookups = %d after ONE lookup, want 1 — the word was recorded twice", deck[0].Lookups)
	}
	ev, _ := st.Events(time.Time{})
	if len(ev) != 1 {
		t.Errorf("got %d events for one lookup, want 1", len(ev))
	}
}

// --- openStore -------------------------------------------------------------

// Half of a Done-when lived here untested: DEFINE_NO_CAPTURE must not merely
// suppress writes, it must leave history session-only rather than half-persisting.
func TestOpenStoreUnderOptOut(t *testing.T) {
	sd := openStore(options{noCapture: true}, nil)
	h, c, deck := sd.history, sd.capture, sd.deck
	if _, ok := h.(*memHistory); !ok {
		t.Errorf("history = %T, want *memHistory — session-only", h)
	}
	if _, ok := c.(noopCapturer); !ok {
		t.Errorf("capturer = %T, want noopCapturer", c)
	}
	if deck != nil {
		t.Errorf("deck = %v, want nil — nothing was opened", deck)
	}
}

// The env → option wiring, end to end: nothing may reach the disk.
func TestNoCaptureWritesNothingToDisk(t *testing.T) {
	// Build the rig BEFORE chdir: the fixture corpus is loaded from a relative
	// testdata path.
	rig := newAudioRig(t, "sycophantic", true)
	rig.deps.stdinIsTerminal = func() bool { return false }
	rig.deps.capture = nil // force the real wiring
	rig.deps.newStore = openStore

	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("DEFINE_NO_CAPTURE", "1")

	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-no-audio", "sycophantic"}, rig.deps, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("DEFINE_NO_CAPTURE=1 still wrote %v", names)
	}
}

// openStore's live path: without the opt-out it must hand back a real deck, or
// --forget has nothing to act on. Previously verified only by running the binary.
func TestOpenStoreWithoutOptOut(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	sd := openStore(options{}, nil)
	h, c, deck := sd.history, sd.capture, sd.deck
	if _, ok := h.(*storeHistory); !ok {
		t.Errorf("history = %T, want *storeHistory", h)
	}
	if _, ok := c.(*storeCapturer); !ok {
		t.Errorf("capturer = %T, want *storeCapturer", c)
	}
	if deck == nil {
		t.Fatal("deck is nil — --forget would report no deck in this directory")
	}
	// And it is rooted at the working directory, not somewhere else.
	if err := deck.Upsert(store.Word{Text: "sycophantic", LastSeen: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "words", "sycophantic.yaml")); err != nil {
		t.Errorf("deck did not write to the working directory: %v", err)
	}
}

// Done-when: "Repeat lookups increment the count rather than duplicating."
//
// The conformance suite pins the merge at the Store level, but nothing pinned it
// THROUGH capture — and capture is what supplies Lookups:1 on every call, so the
// accumulation depends on both halves agreeing.
func TestRepeatLookupsIncrementThroughCapture(t *testing.T) {
	st := store.NewMem()
	c := newStoreCapturer(st, store.FixedClock(time.Now()), nil)

	c.Capture("sycophantic", true, options{})
	c.Capture("Sycophantic", true, options{}) // same word, different case
	c.Capture("sycophantic", true, options{})

	deck, err := st.Deck()
	if err != nil {
		t.Fatal(err)
	}
	if len(deck) != 1 {
		t.Fatalf("three lookups produced %d entries, want 1: %+v", len(deck), deck)
	}
	if deck[0].Lookups != 3 {
		t.Errorf("Lookups = %d after three lookups, want 3", deck[0].Lookups)
	}
}

// Done-when: "A failing store degrades to a warning, never a failed lookup."
//
// The warn-once half was pinned at the capturer; the "never a failed lookup"
// half — the part a user actually feels — was not pinned anywhere.
func TestFailingStoreStillDefinesAndExitsZero(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	rig.deps.capture = newStoreCapturer(failingStore{}, store.FixedClock(time.Now()), nil)
	rig.deps.stdinIsTerminal = func() bool { return false }

	var out, errb bytes.Buffer
	code := run(t.Context(), []string{"-no-audio", "sycophantic"}, rig.deps, strings.NewReader(""), &out, &errb)

	if code != 0 {
		t.Errorf("exit = %d, want 0 — a store that cannot be written is not a failed lookup", code)
	}
	if !strings.Contains(out.String(), "/ˌsikəˈfan(t)ik/") {
		t.Error("the definition was not printed")
	}
}

// A mistyped command must not read the event log. Usage validation therefore
// runs before withStore opens anything.
//
// The obvious assertion — "the directory is still empty" — CANNOT FAIL, and I
// wrote it that way first: NewYAML is a pure constructor, so nothing is created
// on open either way. What does distinguish the two orderings is that
// newStoreHistory READS the log at construction, so with the store opened first
// a corrupt log reports itself in the middle of a usage error. That is the pin.
//
// Verified failable: moving `d = d.withStore(...)` back above the usage switch
// makes every row here fail with
// "define: 2020-01-01.yaml: recovered 0 event(s), dropped 1 torn record(s)".
func TestUsageErrorsDoNotOpenTheLog(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"empty -forget", []string{"-no-audio", "-forget="}},
		{"-forget plus a word", []string{"-no-audio", "-forget=a", "b"}},
		{"two words", []string{"-no-audio", "a", "b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rig := newAudioRig(t, "sycophantic", true)
			rig.deps.stdinIsTerminal = func() bool { return false }
			rig.deps.history, rig.deps.capture = nil, nil // force the real wiring
			rig.deps.newStore = openStore

			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "events"), 0o755); err != nil {
				t.Fatal(err)
			}
			// A record cut mid-write: reading it warns, which is what makes the
			// difference between the two orderings observable at all.
			torn := "word: sycophantic\nkind: looked_up\nat: 2020-01-01T10:00:00-07:00\nfound: true\n---\nword: torn\nkind: looked"
			if err := os.WriteFile(filepath.Join(dir, "events", "2020-01-01.yaml"), []byte(torn), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Chdir(dir)

			var out, errb bytes.Buffer
			if code := run(t.Context(), tc.args, rig.deps, strings.NewReader(""), &out, &errb); code != 2 {
				t.Errorf("exit = %d, want 2 (usage error)", code)
			}
			if strings.Contains(errb.String(), "torn record") || strings.Contains(errb.String(), "recovered") {
				t.Errorf("a usage error read the event log: %q", errb.String())
			}
		})
	}
}

// One source for "what time is it". The clock was constructed inline where the
// capturer was built, so nothing else could reach it; #15's /history needs the
// same clock to compute a local-day window, and two clocks would be two answers.
//
// The noCapture row is the one worth having: that path writes nothing, but
// /history still READS, so a nil clock there is a panic waiting for the first
// opt-out user to run a command.
func TestOpenStoreSuppliesOneClock(t *testing.T) {
	for _, tc := range []struct {
		name     string
		opt      options
		wantDeck bool
	}{
		{"normal", options{}, true},
		{"DEFINE_NO_CAPTURE", options{noCapture: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			sd := openStore(tc.opt, io.Discard)
			if sd.clock == nil {
				t.Error("clock is nil; a command that reads the log would panic")
			}
			if got := sd.deck != nil; got != tc.wantDeck {
				t.Errorf("deck non-nil = %v, want %v", got, tc.wantDeck)
			}
			if sd.clock != nil && sd.clock.Now().IsZero() {
				t.Error("the clock reports the zero time")
			}
		})
	}
}

// The clock has to REACH a command, not merely exist on storeDeps.
func TestWithStoreCarriesTheClock(t *testing.T) {
	t.Run("a test-supplied clock wins", func(t *testing.T) {
		want := fixedClock(1)
		d := deps{clock: want, history: &memHistory{}, capture: noopCapturer{}}
		if got := d.withStore(options{}, io.Discard).clock; got.Now() != want.Now() {
			t.Errorf("clock = %v, want the supplied one (%v)", got.Now(), want.Now())
		}
	})
	t.Run("nothing supplied still yields a usable clock", func(t *testing.T) {
		d := deps{history: &memHistory{}, capture: noopCapturer{}}
		if got := d.withStore(options{}, io.Discard).clock; got == nil || got.Now().IsZero() {
			t.Error("withStore left a nil or zero clock; a command would panic")
		}
	})
}

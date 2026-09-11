package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/schedule"
)

// THE FOUR INPUTS, AND THE ORDER IS THE DESIGN.
//
// Each row asserts BOTH whether the question was put and what came back —
// because "it allowed" and "it allowed without asking" are different behaviours
// and only one of them is right per row.
func TestDeckAskerPolicy(t *testing.T) {
	for _, tc := range []struct {
		name        string
		alreadyDeck bool
		here        bool
		tty         bool
		typed       string
		wantAsked   bool
		wantAllow   bool
	}{
		{"an existing deck is never asked about", true, false, true, "", false, true},
		{"an existing deck is not asked about without a terminal either", true, false, false, "", false, true},
		{"--here creates without asking", false, true, true, "", false, true},
		{"--here works with no terminal, which is its whole point", false, true, false, "", false, true},
		{"no terminal means no deck and no question", false, false, false, "", false, false},
		{"y creates", false, false, true, "y\n", true, true},
		{"yes creates", false, false, true, "yes\n", true, true},
		{"Y creates", false, false, true, "Y\n", true, true},
		{"n declines", false, false, true, "n\n", true, false},
		{"a bare Enter declines — the default is the safe one", false, false, true, "\n", true, false},
		{"anything else declines", false, false, true, "maybe\n", true, false},
		{"EOF mid-question declines", false, false, true, "", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.alreadyDeck {
				if err := os.MkdirAll(filepath.Join(dir, "words"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			got := deckAsker(dir, options{here: tc.here}, strings.NewReader(tc.typed), &out,
				func() bool { return tc.tty })()

			if got != tc.wantAllow {
				t.Errorf("allowed = %v, want %v (said %q)", got, tc.wantAllow, out.String())
			}
			// "Did it ask?" is visible in what it printed: the question is the only
			// thing that ends in a prompt.
			didAsk := strings.Contains(out.String(), "Create one here?")
			if didAsk != tc.wantAsked {
				t.Errorf("asked = %v, want %v; output was %q", didAsk, tc.wantAsked, out.String())
			}
		})
	}
}

// A PIPE IS TOLD WHY, because otherwise someone scripting define in a fresh
// directory gets a working lookup, no deck, and no explanation.
func TestDeckAskerSaysWhyWhenItCannotAsk(t *testing.T) {
	var out bytes.Buffer
	deckAsker(t.TempDir(), options{}, strings.NewReader(""), &out, func() bool { return false })()
	if !strings.Contains(out.String(), "--here") {
		t.Errorf("the no-terminal path did not name the way out; said %q", out.String())
	}
}

// A nil terminal predicate is the same as "not a terminal": absent information
// about the terminal must not create a deck.
func TestDeckAskerTreatsANilPredicateAsNoTerminal(t *testing.T) {
	var out bytes.Buffer
	if deckAsker(t.TempDir(), options{}, strings.NewReader("y\n"), &out, nil)() {
		t.Error("a nil terminal predicate allowed creation; absent information about " +
			"the terminal must not be read as permission")
	}
}

// BOTH LOOP SHELLS SETTLE THE QUESTION BEFORE THEY READ A KEY (#50 PQ-3).
//
// One test per shell, not one test for "the loop": a single test covers only the
// branch it happened to take, and replRaw additionally FALLS BACK to replLines
// twice. The claim is about the choice point, so both sides of it are driven.
func TestBothLoopShellsResolveBeforeReading(t *testing.T) {
	for _, tc := range []struct {
		name string
		tty  bool // opt.tty picks replRaw vs replLines
	}{
		{"line shell", false},
		{"raw shell (falls back to lines on a non-file stdin)", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			asked := 0
			perm := newDeckPermission(func() bool { asked++; return false })

			// testDeps loads the fixture corpus by a path relative to the package
			// directory, so it is built BEFORE chdir — the same ordering
			// lang_cmd_test.go documents.
			d := testDeps(t)
			t.Chdir(dir)
			d.deckPermission = perm
			d.stdinIsTerminal = func() bool { return true }

			var out, errb bytes.Buffer
			// An empty stdin: the loop reads EOF and exits immediately. If the
			// permission were not resolved above the choice, it would still be
			// undecided here.
			repl(t.Context(), d, options{tty: tc.tty}, strings.NewReader(""), &out, &errb)

			if _, decided := perm.saving(); !decided {
				t.Errorf("the loop started without settling whether this directory may " +
					"become a deck. A question put mid-session races the shell's own " +
					"stdin reader for the answer.")
			}
			if asked != 1 {
				t.Errorf("asked %d times, want exactly 1", asked)
			}
		})
	}
}

// THE EMPTY SCREEN MUST NOT PROMISE A DECK THAT WILL NOT EXIST (#50 PQ-6).
//
// "Nothing yet — look a word up and it joins your deck" is true of a new deck and
// FALSE in a directory define was told not to write to. This screen is precisely
// where the person who ran define in the wrong place ends up, so it is the worst
// place in the program to make that promise.
func TestStatsDoesNotPromiseToSaveWhenItCannot(t *testing.T) {
	notSaving := strings.Join(renderStats(schedule.Stats{}, time.Now(), false), "\n")
	if strings.Contains(notSaving, "joins your deck") {
		t.Errorf("the empty screen promises a word will join the deck, in a directory "+
			"nothing is written to:\n%s", notSaving)
	}
	if !strings.Contains(notSaving, "--here") {
		t.Errorf("the empty screen does not say how to fix it:\n%s", notSaving)
	}

	// And the ordinary empty screen is UNCHANGED — an undecided directory is one
	// where "look a word up and it joins your deck" is still true.
	saving := strings.Join(renderStats(schedule.Stats{}, time.Now(), true), "\n")
	if !strings.Contains(saving, "joins your deck") {
		t.Errorf("the ordinary empty screen lost its call to action:\n%s", saving)
	}
}

// --stats NEVER PUTS THE QUESTION. Reading your figures must not become a
// request to create a deck — that is the false alarm the lazy design exists to
// prevent, and false alarms train people to answer yes.
func TestStatsNeverAsks(t *testing.T) {
	dir := t.TempDir()
	d := testDeps(t)
	t.Chdir(dir)

	asked := 0
	d.deckPermission = newDeckPermission(func() bool { asked++; return true })
	d.newStore = openStore

	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-stats"}, d, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit = %d: %s", code, errb.String())
	}
	if asked != 0 {
		t.Errorf("--stats put the deck question %d time(s). It creates nothing, so "+
			"asking permission to create is a false alarm.", asked)
	}
	if names := lsNames(t, dir); len(names) != 0 {
		t.Errorf("--stats created %v in a directory it only read", names)
	}
}

// --stats IS HONEST WITHOUT BEING INTRUSIVE, which is the whole point of
// splitting deckPolicy out of deckAsker.
//
// FOUND BY SMOKE TESTING, not by a test: `define --stats` piped in a non-deck
// directory printed "look a word up and it joins your deck" — the exact promise
// this feature exists to stop making — because saving() refused to resolve and so
// reported "undecided". But the answer there needs NOBODY: no terminal means no
// deck. The free cases settle; the one that would prompt does not.
func TestStatsIsHonestWhereTheAnswerNeedsNobody(t *testing.T) {
	for _, tc := range []struct {
		name      string
		deck      bool
		tty       bool
		wantSaved bool
	}{
		{"piped, not a deck: says so", false, false, false},
		{"piped, already a deck: ordinary screen", true, false, true},
		{"a terminal, not a deck: stays undecided rather than prompting", false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.deck {
				if err := os.MkdirAll(filepath.Join(dir, "words"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			asked := 0
			perm := newDeckPermission(func() bool { asked++; return true }).
				withQuiet(func() deckDecision {
					return deckPolicy(dir, options{}, func() bool { return tc.tty })
				})

			allowed, decided := perm.saving()
			saving := !decided || allowed
			if saving != tc.wantSaved {
				t.Errorf("saving = %v, want %v", saving, tc.wantSaved)
			}
			if asked != 0 {
				t.Errorf("settling for --stats put the question %d time(s); the free "+
					"answers are free precisely because nobody is asked", asked)
			}
		})
	}
}

package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestParsePronArgs(t *testing.T) {
	// A language is REQUIRED, which is where this differs from /lang and /sound.
	// Those split "set" from "report" because they name a SETTING with a current
	// value worth printing. /pron names an ACTION and leaves nothing behind, so a
	// bare one is a half-typed command rather than a question.
	if _, err := parsePronArgs(nil); err == nil {
		t.Error("a bare /pron was accepted; it has nothing to do and nothing to report")
	}
	if _, err := parsePronArgs([]string{"fr", "es"}); err == nil {
		t.Error("/pron took two languages")
	}
	if _, err := parsePronArgs([]string{"french"}); err == nil {
		t.Error("/pron took something that is not a language tag")
	}
	// Complete exactly, accept loosely — the rule dispatch already follows.
	if got, err := parsePronArgs([]string{"FR"}); err != nil || got != "fr" {
		t.Errorf("parsePronArgs(FR) = %q, %v; want fr, nil", got, err)
	}
}

// D2, and the assertion that would catch someone "simplifying" /pron into a
// session field later: it replays THIS word and leaves nothing switched on.
func TestPronReplaysOnceAndLeavesNoMode(t *testing.T) {
	en := voice{Lang: "en", Locale: "us"}
	es := voice{Lang: "es", Locale: "es"}
	first := AudioCandidates("jalapeno", en)[0]
	spanish := AudioCandidates("jalapeño", es)[0]
	third := AudioCandidates("sycophantic", en)[0]

	rig := newAudioRigServing(t, first, spanish, third)
	rig.deps.stdinIsTerminal = func() bool { return false }
	var out, errb bytes.Buffer

	code := run(t.Context(), nil, rig.deps,
		strings.NewReader("jalapeno\n/pron es\nsycophantic\n"), &out, &errb)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}

	want := []string{
		stripHost(t, first, audioBase),   // the lookup, in the session's English
		stripHost(t, spanish, audioBase), // /pron es — Spanish, and the SOURCE spelling
		stripHost(t, third, audioBase),   // English again, with nothing to undo
	}
	if got := rig.cdn.Requested(); !slices.Equal(got, want) {
		t.Errorf("the CDN was asked for:\n  %q\nwant:\n  %q", got, want)
	}
}

// A command that cannot do its one thing says so rather than playing silence —
// the call /sound already makes for a nil setTimes.
func TestPronWithNothingLookedUpSaysSo(t *testing.T) {
	rig := newAudioRigServing(t)
	rig.deps.stdinIsTerminal = func() bool { return false }
	var out, errb bytes.Buffer

	code := run(t.Context(), nil, rig.deps, strings.NewReader("/pron fr\n"), &out, &errb)

	// EXIT 2, the usage code the README documents and the one `fail()` exists to
	// propagate out of the loop. Discarding it left that path unpinned.
	if code != 2 {
		t.Errorf("exit = %d, want 2 — a command that cannot do its one thing is a usage error", code)
	}
	if !strings.Contains(errb.String(), "/pron") {
		t.Errorf("stderr should explain what /pron could not do, got %q", errb.String())
	}
	if n := rig.player.count(); n != 0 {
		t.Errorf("played %d times with nothing looked up, want 0", n)
	}
}

package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestParsePronArgs(t *testing.T) {
	// NO argument is a request to INFER (#35), not a usage error. #29 shipped it
	// as an error because it had rejected inference — but that rejection was
	// about inferring on EVERY lookup, where a wrong guess is silent and
	// unasked-for. Here the user typed the gesture and the choice is reported.
	if lang, err := parsePronArgs(nil); err != nil || lang != "" {
		t.Errorf("parsePronArgs(nil) = %q, %v; want \"\", nil — an absent language means "+
			"read it off the entry, and runPron owns that reading", lang, err)
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

// Bare /pron infers the language from ORIGIN and says which it chose (#35).
//
// On committed fixtures, so the cases are the ones origin_test.go already
// reasons about: `concrete` names French, `read` names Dutch and German only as
// COGNATES, and `gaslighting` names no language at all.
func TestPronInfersTheOriginLanguage(t *testing.T) {
	en := voice{Lang: "en", Locale: "us"}
	fr := voice{Lang: "fr", Locale: "fr"}
	rig := newAudioRigServing(t,
		AudioCandidates("concrete", en)[0],
		AudioCandidates("concrete", fr)[0])
	rig.deps.stdinIsTerminal = func() bool { return false }
	var out, errb bytes.Buffer

	code := run(t.Context(), nil, rig.deps, strings.NewReader("concrete\n/pron\n"), &out, &errb)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}
	want := []string{
		stripHost(t, AudioCandidates("concrete", en)[0], audioBase),
		stripHost(t, AudioCandidates("concrete", fr)[0], audioBase),
	}
	if got := rig.cdn.Requested(); !slices.Equal(got, want) {
		t.Errorf("the CDN was asked for:\n  %q\nwant:\n  %q", got, want)
	}
	// It SAYS what it read. A silent inference cannot be audited, and it is what
	// makes a contested ORIGIN visible rather than decided behind your back.
	if !strings.Contains(out.String()+errb.String(), "ORIGIN says French") {
		t.Errorf("it did not report the language it inferred:\n%s%s", out.String(), errb.String())
	}
}

// And it declines with the reason, on the two shapes that differ.
func TestPronReportsWhyItCannotInfer(t *testing.T) {
	for _, tc := range []struct{ word, because string }{
		// Dutch and German appear only after "related to".
		{"read", "cognate"},
		// "1960s: see gaslight (verb)" — an ORIGIN that names no language at
		// all, which is a DIFFERENT reason and must say so. The `because` field
		// went unasserted in the first version of this test, and that is what let
		// this case ship with the cognates-and-stages message.
		{"gaslighting", "names no language"},
	} {
		t.Run(tc.word, func(t *testing.T) {
			rig := newAudioRigServing(t, AudioCandidates(tc.word, voice{Lang: "en", Locale: "us"})[0])
			rig.deps.stdinIsTerminal = func() bool { return false }
			var out, errb bytes.Buffer

			run(t.Context(), nil, rig.deps, strings.NewReader(tc.word+"\n/pron\n"), &out, &errb)

			if !strings.Contains(errb.String(), "/pron") {
				t.Errorf("the refusal does not name the command: %q", errb.String())
			}
			// The REASON, asserted. Both shapes decline, and telling a user their
			// entry names only stages when it names no language at all is a
			// record that is not true.
			if !strings.Contains(errb.String(), tc.because) {
				t.Errorf("the refusal does not say WHY (want %q): %q", tc.because, errb.String())
			}
			// Exactly ONE request: the lookup's own. No inferred replay happened.
			if got := rig.cdn.Requested(); len(got) != 1 {
				t.Errorf("made %d CDN requests, want 1 — it replayed despite declining: %q", len(got), got)
			}
		})
	}
}

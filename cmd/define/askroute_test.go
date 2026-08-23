package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

const aQuestion = "what's the difference to obsequious?"

// "Every entry mode reaches it" is the invariant BR-13 cost us once already, so
// it gets a test per mode rather than one test and an assumption.
//
// And per mode is not enough: a question arrives by TWO routes — forced ("?…",
// decided by the parser without a dictionary call) and unforced (a miss that
// reads as one) — so the enumeration this file has to cover is
// {replLines, runEditor} x {forced, unforced}, four cells. Three were empty on
// the forced side at the M1 boundary review (I-2): the whole `case cmdAsk:` could
// be deleted from BOTH loops with the suite green, because every loop-level test
// took the unforced route and routeFor answers "question" for cmdAsk without
// entering a loop at all. That is lessons.md define #15 — a wiring only a loop
// shell supplies must be pinned by a test that drives that loop shell — applied
// to the branch this milestone is named after.
func TestLineLoopRoutesAQuestion(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	replLines(t.Context(), rig.deps, options{times: 3, locale: "us"},
		strings.NewReader(aQuestion+"\n"), &out, &errb, true, false)

	assertAskedAndUnanswered(t, errb.String(), out.String())
}

func TestEditorLoopRoutesAQuestion(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys(aQuestion+"\r"), rig.deps, opt, cooked, finish, &out, &errb)

	assertAskedAndUnanswered(t, errb.String(), out.String())
}

// The forced half of the enumeration. Both pass against today's code — which is
// the point: they exist to go RED when the branch is touched.
func TestLineLoopRoutesAForcedQuestion(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	// "why" IS a headword, so only the hatch can make this a question — which is
	// what makes it a test of the forced branch rather than of the classifier.
	replLines(t.Context(), rig.deps, options{times: 3, locale: "us"},
		strings.NewReader("?why\n"), &out, &errb, true, false)

	assertAskedAndUnanswered(t, errb.String(), out.String())
	if strings.Contains(out.String(), "adverb") {
		t.Error("the forced question was defined instead of asked")
	}
}

func TestEditorLoopRoutesAForcedQuestion(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("?why\r"), rig.deps, opt, cooked, finish, &out, &errb)

	assertAskedAndUnanswered(t, errb.String(), out.String())
	if rig.player.count() != 0 {
		t.Errorf("played %d times — a forced question fell through to replay", rig.player.count())
	}
}

func TestOneShotRoutesAQuestion(t *testing.T) {
	var out, errb bytes.Buffer
	// Quoted: one argument, unforced. The dictionary misses and the classifier
	// answers, exactly as it does at the prompt.
	if code := run(t.Context(), []string{aQuestion}, testDeps(t),
		strings.NewReader(""), &out, &errb); code == 0 {
		t.Errorf("exit = 0, want non-zero for an unanswerable question")
	}
	assertAskedAndUnanswered(t, errb.String(), out.String())
}

func TestOneShotForcedQuestionIsNotAUsageError(t *testing.T) {
	var out, errb bytes.Buffer
	// Unquoted and multi-word: a question is multi-word by nature, exactly as a
	// command is, so the argument-count guard must exempt it the same way.
	run(t.Context(), []string{"?what", "is", "the", "difference"},
		testDeps(t), strings.NewReader(""), &out, &errb)

	if strings.Contains(errb.String(), "usage") || strings.Contains(errb.String(), "Usage") {
		t.Errorf("a forced question was rejected as a usage error: %q", errb.String())
	}
	assertAskedAndUnanswered(t, errb.String(), out.String())
}

func assertAskedAndUnanswered(t *testing.T, stderr, stdout string) {
	t.Helper()
	if !strings.Contains(stderr, "no model configured") {
		t.Errorf("stderr = %q, want the no-model message", stderr)
	}
	if strings.Contains(stderr, "not found") {
		t.Errorf("the question reached the dictionary: %q", stderr)
	}
	if strings.Contains(stdout, "|") {
		t.Errorf("a definition was rendered for a question: %q", stdout)
	}
}

// The ask outcome's contract, one test per row. Each fails if its guard is
// deleted — which is the only reason to write them separately.

func TestAQuestionDoesNotBecomeTheCurrentWord(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	// Look up a word, ask a question, then press Enter: the REPLAY is the word.
	runEditor(t.Context(), scriptKeys("sycophantic\r"+aQuestion+"\r\r"), rig.deps, opt, cooked, finish, &out, &errb)

	// 3 plays for the lookup, 3 more for the replay. If the question had become
	// the current word the replay would have found no audio for it.
	if got := rig.player.count(); got != 6 {
		t.Errorf("played %d times, want 6 — the question became the current word", got)
	}
}

func TestAQuestionIsNotCaptured(t *testing.T) {
	cap := &countingCapturer{}
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	rig.deps.capture = cap
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r"+aQuestion+"\r"), rig.deps, opt, cooked, finish, &out, &errb)

	if len(cap.calls) != 1 || cap.calls[0] != "sycophantic" {
		t.Errorf("captured %v, want just the lookup — a question is not a lookup", cap.calls)
	}
}

// Asserted against the injected History, NOT against stdout.
//
// The first version of this test checked stdout for the question and could not
// fail: the raw editor re-renders the whole line on every keystroke, so the
// question is in stdout from typing alone — before Enter, before Up, before
// history is consulted. Removing hist.Add outright left it green (I-1). The
// general shape is worth remembering: in a loop that echoes, an assertion on
// stdout is satisfied by the echo, so it has to be made against the thing the
// behaviour actually writes to.
func TestAQuestionIsRecalledByUpArrow(t *testing.T) {
	for _, tc := range []struct{ name, keys, want string }{
		{"forced", "?why\r", "?why"},
		{"unforced", aQuestion + "\r", aQuestion},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
			hist := &memHistory{}
			rig.deps.history = hist
			var out, errb bytes.Buffer
			runEditor(t.Context(), scriptKeys(tc.keys), rig.deps, opt, cooked, finish, &out, &errb)

			if len(hist.lines) != 1 || hist.lines[0] != tc.want {
				t.Fatalf("history = %q, want [%q] — the question was not recorded for recall", hist.lines, tc.want)
			}
			// And that what was recorded is what Up actually offers back.
			if got := hist.Prefix(""); len(got) == 0 || got[0] != tc.want {
				t.Errorf("Prefix(\"\") = %q, want %q first", got, tc.want)
			}
		})
	}
}

// BR-4: the one-shot's dispatch must be exhaustive over what parseREPLLine can
// return, not "handle the kinds I added and fall through". #16 gave the parser a
// kind that branch had never seen — cmdNothing, from a bare "?" or "\" — and the
// fall-through handed lookupAndRender an EMPTY word, which then appended an event
// with no word that the log discards at read time as if it were torn.
func TestOneShotRejectsAHatchWithNothingAfterIt(t *testing.T) {
	for _, line := range []string{"?", `\`} {
		t.Run(line, func(t *testing.T) {
			cap := &countingCapturer{}
			d := testDeps(t)
			d.newStore = func(options, io.Writer) storeDeps {
				return storeDeps{history: &memHistory{}, capture: cap, clock: store.SystemClock()}
			}
			var out, errb bytes.Buffer
			code := run(t.Context(), []string{line}, d, strings.NewReader(""), &out, &errb)

			if code != 2 {
				t.Errorf("exit = %d, want 2 (a usage error)", code)
			}
			if strings.Contains(errb.String(), "no dictionary entry") {
				t.Errorf("an empty word reached the dictionary: %q", errb.String())
			}
			for _, w := range cap.calls {
				if w == "" {
					t.Error("an empty word was captured — the log would hold a record it discards as torn")
				}
			}
		})
	}
}

// -raw is the scripting form; README documents it as recording nothing "because
// it is for scripts". It must not reach the model either — in M1 that costs a
// different message, in M2 a network call for a line a script piped in.
func TestRawNeverAsks(t *testing.T) {
	var out, errb bytes.Buffer
	code := run(t.Context(), []string{"-raw", aQuestion}, testDeps(t), strings.NewReader(""), &out, &errb)

	if code != 1 {
		t.Errorf("exit = %d, want 1 (a miss stays a miss under -raw)", code)
	}
	if strings.Contains(errb.String(), "no model configured") {
		t.Errorf("-raw routed to the model: %q", errb.String())
	}
}

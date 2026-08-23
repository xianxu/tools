package main

import (
	"bytes"
	"strings"
	"testing"
)

const aQuestion = "what's the difference to obsequious?"

// "Every entry mode reaches it" is the invariant BR-13 cost us once already,
// so it gets a test per mode rather than one test and an assumption.
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

func TestAQuestionIsRecalledByUpArrow(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	keys := make(chan Key, 64)
	for _, k := range keysFor(aQuestion + "\r") {
		keys <- k
	}
	keys <- Key{Kind: KeyUp}
	close(keys)
	runEditor(t.Context(), keys, rig.deps, opt, cooked, finish, &out, &errb)

	if !strings.Contains(out.String(), aQuestion) {
		t.Errorf("Up did not recall the question; stdout = %q", out.String())
	}
}

func keysFor(s string) []Key {
	var ks []Key
	for _, r := range s {
		switch r {
		case '\r', '\n':
			ks = append(ks, Key{Kind: KeyEnter})
		default:
			ks = append(ks, Key{Kind: KeyRune, Rune: r})
		}
	}
	return ks
}

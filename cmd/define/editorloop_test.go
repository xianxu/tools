package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// scriptKeys turns a string into keypresses, with "\r" meaning Enter — the
// scripted key source the plan called for, so the editor loop is testable with
// no terminal anywhere.
func scriptKeys(s string) <-chan Key {
	ch := make(chan Key, len(s)+1)
	for _, r := range s {
		switch r {
		case '\r', '\n':
			ch <- Key{Kind: KeyEnter}
		case '\x03':
			ch <- Key{Kind: KeyInterrupt}
		default:
			ch <- Key{Kind: KeyRune, Rune: r}
		}
	}
	close(ch) // channel close == end of input
	return ch
}

func editorRig(t *testing.T, word string, audioPresent bool) (*audioRig, options, func(func()), func()) {
	t.Helper()
	rig := newAudioRig(t, word, audioPresent)
	rig.deps.stdinIsTerminal = func() bool { return true }
	// cooked and finish are no-ops: there is no real terminal in a test.
	return rig, options{times: 3, locale: "us", tty: true, color: true}, func(run func()) { run() }, func() {}
}

func TestEditorLoopDefinesTypedWord(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	code := runEditor(t.Context(), scriptKeys("sycophantic\r"), rig.deps, opt, cooked, finish, &out, &errb)

	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "/ˌsikəˈfan(t)ik/") {
		t.Error("the typed word was not defined")
	}
	if rig.player.count() != 3 {
		t.Errorf("played %d times, want 3", rig.player.count())
	}
}

// A bare Enter replays, and costs no second fetch.
func TestEditorLoopBareEnterReplays(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r\r"), rig.deps, opt, cooked, finish, &out, &errb)

	if got := rig.player.count(); got != 6 {
		t.Errorf("played %d times, want 6", got)
	}
	if got := rig.cdn.Requested(); len(got) != 1 {
		t.Errorf("made %d CDN requests, want 1 — the replay refetched: %v", len(got), got)
	}
	if n := strings.Count(out.String(), "/ˌsikəˈfan(t)ik/"); n != 1 {
		t.Errorf("definition printed %d times, want 1 — replay reprinted it", n)
	}
}

// History is populated from submitted lines, so the next frame can suggest.
func TestEditorLoopFeedsHistory(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\rsyc"), rig.deps, opt, cooked, finish, &out, &errb)

	// The final frame should carry the grey remainder of the earlier word.
	if !strings.Contains(out.String(), greyOn+"ophantic"+greyOff) {
		t.Errorf("no grey suggestion in the final frame: %q", tailOf(out.String()))
	}
}

func TestEditorLoopCtrlCExitsZero(t *testing.T) {
	rig, opt, cooked, _ := editorRig(t, "sycophantic", true)
	restored := false
	var out, errb bytes.Buffer
	code := runEditor(t.Context(), scriptKeys("syc\x03"), rig.deps, opt, cooked,
		func() { restored = true }, &out, &errb)

	if code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
	if !restored {
		t.Error("the terminal was not restored — the shell would be left raw")
	}
}

// Cancellation must restore the terminal too. This is the worst failure this
// tool can produce: no later output can fix a shell left in raw mode.
func TestEditorLoopCancellationRestoresTerminal(t *testing.T) {
	rig, opt, cooked, _ := editorRig(t, "sycophantic", true)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	restored := false
	var out, errb bytes.Buffer

	runEditor(ctx, make(chan Key), rig.deps, opt, cooked, func() { restored = true }, &out, &errb)
	if !restored {
		t.Error("cancellation exited without restoring the terminal")
	}
}

func tailOf(s string) string {
	if len(s) > 120 {
		return s[len(s)-120:]
	}
	return s
}

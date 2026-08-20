package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestParseREPLLine(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		hasCurrent bool
		want       replCommand
	}{
		{"a word", "sycophantic", false, replCommand{cmdDefine, "sycophantic"}},
		{"surrounding space trimmed", "  ephemeral  ", false, replCommand{cmdDefine, "ephemeral"}},
		{"multi-word headword", "hot  dog", false, replCommand{cmdDefine, "hot dog"}},
		{"blank replays when there is a current word", "", true, replCommand{kind: cmdReplay}},
		{"blank with nothing current", "", false, replCommand{kind: cmdNothing}},
		{"whitespace only is blank", "   \t ", true, replCommand{kind: cmdReplay}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseREPLLine(tc.line, tc.hasCurrent); got != tc.want {
				t.Errorf("parseREPLLine(%q, %v) = %+v, want %+v", tc.line, tc.hasCurrent, got, tc.want)
			}
		})
	}
}

// replRig drives the loop from a string with no terminal anywhere.
func replRig(t *testing.T, word string, audioPresent, interactive bool) (*audioRig, options) {
	return replRigStreams(t, word, audioPresent, interactive, interactive)
}

// replRigStreams keeps the two terminal questions INDEPENDENT. An earlier
// version hard-coupled them, commented as "the real-world pairing" — and that
// assumption is exactly what hid escape sequences leaking into a redirected
// stdout while stdin was still a terminal.
func replRigStreams(t *testing.T, word string, audioPresent, stdinTTY, stdoutTTY bool) (*audioRig, options) {
	t.Helper()
	rig := newAudioRig(t, word, audioPresent)
	rig.deps.stdinIsTerminal = func() bool { return stdinTTY }
	return rig, options{times: 3, locale: "us", tty: stdoutTTY}
}

// NOTE: #2's interactive tests drove repl() with a strings.Reader. That path is
// now the LINE loop — raw mode needs a real terminal handle — so the editor's
// behaviour is exercised through runEditor with a scripted key channel instead,
// in editorloop_test.go. What remains here covers the line loop, which is still
// what piped input and a redirected stdout use.

func TestREPLEndOfInputExitsZero(t *testing.T) {
	rig, opt := replRig(t, "sycophantic", true, false)
	var out, errb bytes.Buffer
	if code := repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader(""), &out, &errb); code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
}

// Cancellation must not be blocked behind a pending read — this is what makes
// Ctrl-C work while the loop is waiting for input.
func TestREPLCancelledContextReturnsPromptly(t *testing.T) {
	rig, opt := replRig(t, "sycophantic", true, false)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var out, errb bytes.Buffer
	done := make(chan int, 1)
	go func() {
		// A reader that never yields a line and never ends: only ctx can end this.
		done <- repl(ctx, cancel, rig.deps, opt, blockingReader{}, &out, &errb)
	}()
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit = %d, want 0", code)
		}
	case <-t.Context().Done():
		t.Fatal("repl did not return on a cancelled context")
	}
}

// blockingReader blocks forever, modelling a terminal awaiting input.
type blockingReader struct{}

func (blockingReader) Read([]byte) (int, error) { select {} }

func TestREPLPromptOnlyWhenInteractive(t *testing.T) {
	for _, interactive := range []bool{true, false} {
		rig, opt := replRig(t, "sycophantic", true, interactive)
		var out, errb bytes.Buffer
		repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader("sycophantic\n"), &out, &errb)
		if got := strings.Contains(out.String(), prompt); got != interactive {
			t.Errorf("interactive=%v: prompt present=%v", interactive, got)
		}
	}
}

// The strongest statement of the replay contract: stdout is byte-identical
// before and after a bare return.
func TestREPLReplayWritesNothingToStdout(t *testing.T) {
	rig, opt := replRig(t, "sycophantic", true, false)

	var once bytes.Buffer
	repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader("sycophantic\n"), &once, &bytes.Buffer{})

	rig2, opt2 := replRig(t, "sycophantic", true, false)
	var twice bytes.Buffer
	repl(t.Context(), func() {}, rig2.deps, opt2, strings.NewReader("sycophantic\n\n\n"), &twice, &bytes.Buffer{})

	if once.String() != twice.String() {
		t.Errorf("two replays changed stdout:\n one: %q\n two: %q", once.String(), twice.String())
	}
	if got := rig2.player.count(); got != 9 {
		t.Errorf("played %d times, want 9 (3 definitions worth: 1 typed + 2 replays)", got)
	}
}

// The >64 KB guard exists so a large paste cannot look like EOF and silently end
// the session. It shipped untested; this pins the branch.
func TestREPLOverlongLineIsReportedNotSilentEOF(t *testing.T) {
	rig, opt := replRig(t, "sycophantic", true, false)
	var out, errb bytes.Buffer
	huge := strings.Repeat("a", maxLineBytes+1) + "\n"

	if code := repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader(huge), &out, &errb); code != 1 {
		t.Errorf("exit = %d, want 1 — an unreadable line must not be mistaken for EOF", code)
	}
	if !strings.Contains(errb.String(), "reading input") {
		t.Errorf("want a diagnostic naming the read failure, got %q", errb.String())
	}
}

// A line just under the cap is ordinary input, not an error — the guard must not
// be so eager that it rejects a long paste it can actually handle.
func TestREPLLongButReadableLineIsJustAWord(t *testing.T) {
	rig, opt := replRig(t, "sycophantic", true, false)
	var out, errb bytes.Buffer
	long := strings.Repeat("a", 100_000) + "\n"

	repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader(long), &out, &errb)

	// It is looked up (and fails, since it is not a word) rather than being
	// reported as an unreadable line — that distinction is the point of the test.
	if strings.Contains(errb.String(), "reading input") {
		t.Errorf("a readable line was reported as a read failure: %q", errb.String())
	}
	if !strings.Contains(errb.String(), "no dictionary entry") {
		t.Errorf("want an ordinary failed lookup, got %q", errb.String())
	}
}

// Interactively, a replay flashes the indicator and then erases it, ending with
// #2's flash-and-step-back test is deleted with the behaviour it covered: raw
// mode does not echo Enter, so there is no echo to step back over. The replay
// contract now lives in TestEditorLoopBareEnterReplays.
func TestREPLNonInteractiveReplayEmitsNoEscapes(t *testing.T) {
	rig, opt := replRig(t, "sycophantic", true, false)
	var out, errb bytes.Buffer
	repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader("sycophantic\n\n"), &out, &errb)

	if strings.Contains(out.String(), "\x1b[") {
		t.Error("ANSI escapes leaked into non-interactive output")
	}
}

// The indicator is ephemeral on the DEFINE path too, not only on replay: once
// the sound has finished it is noise, so the settled screen shows the definition
// and nothing else.
func TestREPLDefineIndicatorIsErasedAfterPlayback(t *testing.T) {
	rig, opt := replRig(t, "sycophantic", true, true)
	var out, errb bytes.Buffer
	repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader("sycophantic\n"), &out, &errb)

	s := out.String()
	i := strings.Index(s, "♫")
	if i < 0 {
		t.Fatal("no indicator shown")
	}
	if !strings.Contains(s[i:], eraseLine) {
		t.Error("the define-path indicator was never erased")
	}
}

// I-1: cursor control goes to STDOUT, so a terminal stdin with a redirected
// stdout must emit no escapes. `define > out.txt` is ordinary usage.
func TestREPLMismatchedStreamsEmitNoEscapes(t *testing.T) {
	rig, opt := replRigStreams(t, "sycophantic", true, true /*stdin tty*/, false /*stdout redirected*/)
	var out, errb bytes.Buffer
	repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader("sycophantic\n\n"), &out, &errb)

	if strings.Contains(out.String(), "\x1b[") {
		t.Errorf("ANSI escapes leaked into a redirected stdout: %q", out.String())
	}
}

// I-2: a failed replay must not write its diagnostic onto a redrawn prompt, and
// must not consume the next prompt.
func TestREPLFailedReplayDoesNotEatThePrompt(t *testing.T) {
	rig, opt := replRig(t, "sycophantic", true, true)
	opt.noAudio = true
	var out, errb bytes.Buffer
	repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader("sycophantic\n\n"), &out, &errb)

	if !strings.Contains(errb.String(), "audio is off") {
		t.Errorf("want the hint on stderr, got %q", errb.String())
	}
	// Three prompts for two reads plus the iteration that discovers EOF: the
	// failure path claims no line, so the loop draws every one of them.
	if n := strings.Count(out.String(), prompt); n != 3 {
		t.Errorf("prompt written %d times, want 3", n)
	}
	// With audio off there is nothing to announce and no cursor to move.
	if strings.Contains(out.String(), "♫") {
		t.Error("announced playback with -no-audio")
	}
	if strings.Contains(out.String(), "\x1b[") {
		t.Error("moved the cursor for a replay that never played")
	}
}

// The likelier I-2 trigger in real use: the replay plays nothing because the CDN
// has no recording.
//
// stdout and stderr are teed into ONE buffer here, because that is what a
// terminal is — and the defect is only visible in the interleaving. With the
// streams captured separately both the fixed and the broken version produce
// identical bytes, which is why the first version of this test could not fail.
func TestREPLFailedReplayDoesNotWriteOntoThePrompt(t *testing.T) {
	rig, opt := replRig(t, "sycophantic", false /* no recording */, true)
	var screen bytes.Buffer
	repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader("sycophantic\n\n"), &screen, &screen)

	s := screen.String()
	i := strings.Index(s, "define: sycophantic: no recorded pronunciation")
	if i < 0 {
		t.Fatalf("failure was not reported: %q", s)
	}
	// The diagnostic must not land on a prompt the replay had already redrawn.
	if strings.HasSuffix(s[:i], prompt) {
		t.Error("the diagnostic was written onto a redrawn prompt")
	}
	// And the loop must still draw a prompt afterwards, so the user is not left
	// typing onto a bare line.
	if !strings.Contains(s[i:], prompt) {
		t.Error("no prompt after the failure — the next read has no prompt")
	}
}

// I-4: the Ctrl-C suppression guard was dead to the suite — re-deleting it was
// green. A pre-cancelled context drives it end to end.
func TestCancellationPrintsNoDiagnostic(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var out, errb bytes.Buffer
	if code := defineOnce(ctx, rig.deps, options{times: 3, locale: "us"}, "sycophantic", &out, &errb); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if errb.Len() != 0 {
		t.Errorf("Ctrl-C printed a diagnostic: %q", errb.String())
	}
}

// I-2: README presents `echo word | define` as interchangeable with `define
// word` and documents exit 1 for an unknown word. The loop returned 0 at EOF
// regardless, so a script doing `echo "$w" | define || …` got no signal.
func TestREPLPipedFailedLookupExitsNonZero(t *testing.T) {
	rig, opt := replRig(t, "sycophantic", true, false /* piped */)
	var out, errb bytes.Buffer
	if code := repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader("rizz\n"), &out, &errb); code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
}

// ...but a typo at an interactive prompt is not a failed session.
func TestREPLInteractiveFailedLookupExitsZero(t *testing.T) {
	rig, opt := replRig(t, "sycophantic", true, true)
	var out, errb bytes.Buffer
	if code := repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader("rizz\nsycophantic\n"), &out, &errb); code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
}

// I-3: -no-color must suppress cursor control too, not merely colour.
func TestNoColorSuppressesAllANSI(t *testing.T) {
	rig, opt := replRig(t, "sycophantic", true, true)
	opt.color, opt.tty = false, false // what -no-color produces on a terminal
	var out, errb bytes.Buffer
	repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader("sycophantic\n\n"), &out, &errb)

	if strings.Contains(out.String(), "\x1b") {
		t.Errorf("-no-color emitted ANSI: %q", out.String())
	}
}

// The third instance of one family: UI written to stdout but gated on stdin.
// `define > out.txt` from a terminal must leave the file clean AND is why the
// human sees nothing — so the prompt is gated on the same predicate as the
// indicator and the cursor control.
func TestREPLPromptRequiresBothStreams(t *testing.T) {
	rig, opt := replRigStreams(t, "sycophantic", true, true /*stdin tty*/, false /*stdout redirected*/)
	var out, errb bytes.Buffer
	repl(t.Context(), func() {}, rig.deps, opt, strings.NewReader("sycophantic\n"), &out, &errb)

	if strings.Contains(out.String(), prompt) {
		t.Errorf("prompt polluted a redirected stdout: %q", out.String())
	}
	if strings.Contains(out.String(), "\x1b") {
		t.Errorf("ANSI leaked into a redirected stdout: %q", out.String())
	}
}

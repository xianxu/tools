package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

func editorRig(t *testing.T, word string, audioPresent bool) (*audioRig, options, func()) {
	t.Helper()
	rig := newAudioRig(t, word, audioPresent)
	rig.deps.stdinIsTerminal = func() bool { return true }
	// finish is a no-op: there is no real terminal in a test.
	return rig, options{times: 3, locale: "us", tty: true, color: true}, func() {}
}

// paintInto is the paint double: it writes the frame where the loop's own output
// goes, so a test still reads one session as one stream.
//
// In production paint is liveScreen.Draw, which puts the live edge — the prompt
// and the command menu — on a terminal rather than in the buffer (#30 D5). A
// test has no terminal and no screen, so the two halves are folded back into one
// writer, which is exactly what stdout was before the screen existed. What these
// tests pin is the LOOP's decisions: what it writes, when it looks up, what it
// replays. The screen's own arithmetic is pinned in screen_test.go, with no
// terminal either.
func paintInto(w io.Writer) func(prompt string, menu []string) {
	return func(prompt string, menu []string) {
		// Menu first, prompt last, in the order Paint puts them on a screen: the
		// dropdown hangs below the line you are typing.
		for _, m := range menu {
			fmt.Fprint(w, "\r\n"+m)
		}
		fmt.Fprint(w, prompt)
	}
}

func TestEditorLoopDefinesTypedWord(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	code := runEditor(t.Context(), scriptKeys("sycophantic\r"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

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
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r\r"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

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
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\rsyc"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

	// The final frame should carry the grey remainder of the earlier word.
	if !strings.Contains(out.String(), greyOn+"ophantic"+greyOff) {
		t.Errorf("no grey suggestion in the final frame: %q", tailOf(out.String()))
	}
}

func TestEditorLoopCtrlCExitsZero(t *testing.T) {
	rig, opt, _ := editorRig(t, "sycophantic", true)
	restored := false
	var out, errb bytes.Buffer
	code := runEditor(t.Context(), scriptKeys("syc\x03"), nil, rig.deps, opt, paintInto(&out),
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
	rig, opt, _ := editorRig(t, "sycophantic", true)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	restored := false
	var out, errb bytes.Buffer

	runEditor(ctx, make(chan Key), nil, rig.deps, opt, paintInto(&out), func() { restored = true }, &out, &errb)
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

// A bare Enter must not advance the prompt: it replays, and the screen ends
// where it started. Operator-reported — each empty return was adding a line.
func TestEditorLoopBareEnterDoesNotAdvance(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)
	baseline := strings.Count(out.String(), "\n")

	rig2, opt2, finish2 := editorRig(t, "sycophantic", true)
	var out2 bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r\r\r\r"), nil, rig2.deps, opt2, paintInto(&out2), finish2, &out2, &bytes.Buffer{})

	// Three extra replays, zero extra lines.
	if got := strings.Count(out2.String(), "\n"); got != baseline {
		t.Errorf("three replays added %d newlines, want 0", got-baseline)
	}
	if rig2.player.count() != 12 {
		t.Errorf("played %d times, want 12 (4 × 3)", rig2.player.count())
	}
}

// In RAW mode "\n" is a line feed with no carriage return, so anything written
// before dropping back to cooked mode must use "\r\n" — otherwise the next line
// starts at the current column and the definition renders indented.
func TestEditorLoopUsesCarriageReturnsInRawMode(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

	s := out.String()
	// Find the newline the loop writes to commit the input line.
	i := strings.Index(s, "\n")
	if i < 1 {
		t.Fatalf("no newline written: %q", s)
	}
	if s[i-1] != '\r' {
		t.Errorf("bare \\n written in raw mode at %d — the next line would start at the current column: %q", i, s[max(0, i-20):i+1])
	}
}

// The input line must be findable in a screenful of definition text: the prompt
// carries an accent colour and the typed word is bold, so the one line you can
// act on reads differently from everything you cannot.
func TestRenderLineMakesTheInputLineDistinct(t *testing.T) {
	e := NewEditor()
	e.Line = []rune("fold")
	e.Cursor = 4

	got := RenderLine(e, "able", nil, true)
	if !strings.Contains(got, promptOn+prompt) {
		t.Error("the prompt is not accented")
	}
	if !strings.Contains(got, inputOn+"fold") {
		t.Error("typed text is not emphasised")
	}
	if !strings.Contains(got, greyOn+"able") {
		t.Error("the suggestion lost its grey")
	}
	// -no-color must still produce no ANSI beyond the frame control.
	plain := RenderLine(e, "able", nil, false)
	for _, sgr := range []string{promptOn, inputOn, greyOn} {
		if strings.Contains(plain, sgr) {
			t.Errorf("colour leaked into a no-colour render: %q", plain)
		}
	}
	if !strings.Contains(plain, "fold"+"able") {
		t.Errorf("plain render lost content: %q", plain)
	}
}

// The grey tail was never accepted, so committing it to scrollback would claim
// the user typed something they did not. Operator-reported.
func TestEditorLoopCommitsWithoutTheSuggestion(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	// Define the long word, then type a prefix of it and submit.
	runEditor(t.Context(), scriptKeys("sycophantic\rsyc\r"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

	s := out.String()
	// The last frame written before the second submit must carry no grey.
	idx := strings.LastIndex(s, "\r\n")
	if idx < 0 {
		t.Fatal("no committed line")
	}
	head := s[:idx]
	lastFrame := head[strings.LastIndex(head, eraseLine):]
	if strings.Contains(lastFrame, greyOn) {
		t.Errorf("the committed line still showed the suggestion: %q", lastFrame)
	}
	if !strings.Contains(lastFrame, "syc") {
		t.Errorf("committed frame lost the typed text: %q", lastFrame)
	}
}

// C-2: the interactive path must mean the same thing as the line path. It
// bypassed parseREPLLine, so it trimmed nothing and did not collapse interior
// whitespace — "hot  dog" never matched the multi-word headword.
func TestEditorLoopNormalisesTheSubmittedLine(t *testing.T) {
	rig, opt, finish := editorRig(t, "hot dog", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("  hot   dog  \r"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

	if strings.Contains(errb.String(), "no dictionary entry") {
		t.Errorf("the line was not normalised before lookup: %q", errb.String())
	}
	if !strings.Contains(out.String(), "hot dog") {
		t.Errorf("the multi-word headword was not defined: %q", tailOf(out.String()))
	}
}

// I-2: a bare Enter with nothing defined yet is a hint, not a crash or a lookup.
func TestEditorLoopBareEnterWithNoCurrentWord(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("\r"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

	if !strings.Contains(errb.String(), "press return to replay") {
		t.Errorf("want the hint, got %q", errb.String())
	}
	if rig.player.count() != 0 {
		t.Errorf("played %d times with nothing defined", rig.player.count())
	}
}

// I-2: a failed lookup must not become the current word, so a following bare
// Enter replays the last GOOD word rather than retrying the typo.
func TestEditorLoopFailedLookupKeepsPreviousWord(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\rrizz\r\r"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

	if !strings.Contains(errb.String(), "rizz") {
		t.Error("the failed lookup was not reported")
	}
	if got := rig.player.count(); got != 6 {
		t.Errorf("played %d times, want 6 — the replay did not use the last good word", got)
	}
}

// The loop through a real SCREEN, which is what production hands it (#30 D5).
//
// The rest of this file drives the loop with a plain buffer, which is the right
// instrument for what the loop DECIDES. This one pins where its output LANDS:
// the definition and the line the user committed are buffer lines — they are
// scrollback, and D3 prints them back into the normal screen on exit — while the
// `♫ playing 3×` indicator is drawn and taken back, so the transcript never
// files a claim that playback happened.
func TestEditorLoopWritesThroughAScreen(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	sc := &screen{}
	runEditor(t.Context(), scriptKeys("sycophantic\r"), nil, rig.deps, opt,
		func(string, []string) {}, finish, sc, sc)

	got := sc.Transcript()
	if !strings.Contains(got, "/ˌsikəˈfan(t)ik/") {
		t.Errorf("the definition never reached the buffer: %q", got)
	}
	// The committed line, colours and all, is the FIRST thing in the buffer: the
	// word the user typed stays above the entry it produced.
	first := sc.Lines()[0]
	if !strings.Contains(first, prompt) || !strings.Contains(first, "sycophantic") {
		t.Errorf("the committed line is not the head of the transcript: %q", first)
	}
	if strings.Contains(got, "♫") {
		t.Errorf("the ephemeral indicator survived into the record: %q", got)
	}
	// The prompt is the LIVE EDGE and goes to paint, so a session that typed
	// eleven characters must not hold eleven copies of the prompt.
	if n := strings.Count(got, prompt); n != 1 {
		t.Errorf("the buffer holds %d prompts, want 1 — the live edge was buffered", n)
	}
	if strings.Contains(got, "\x1b[K") {
		t.Errorf("a terminal control sequence was buffered as text: %q", got)
	}
}

// writerFunc adapts a function to io.Writer, so a test can observe WHEN a write
// happens rather than only what it said.
type writerFunc func(p []byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

// The prompt belongs to a loop that is WAITING for a keystroke.
//
// Every write repaints the frame around the live edge last recorded, so a prompt
// left standing through a lookup is redrawn under the definition it produced:
// `arrondissement` appeared twice, once as the entry's headword and once as a
// prompt still holding the line just submitted, while the recording played
// (operator-reported). What is pinned here is the general property rather than
// that instance — nothing is written while a prompt is on the frame, because a
// prompt drawn when nothing is reading keys invites typing at a line that does
// not exist.
func TestNothingIsWrittenWhileAPromptIsShown(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var live string
	var whenWritten []string
	paint := func(p string, _ []string) { live = p }
	w := writerFunc(func(p []byte) (int, error) {
		whenWritten = append(whenWritten, live)
		return len(p), nil
	})
	runEditor(t.Context(), scriptKeys("sycophantic\r"), nil, rig.deps, opt, paint, finish, w, w)

	if len(whenWritten) == 0 {
		t.Fatal("the session wrote nothing at all, so nothing is asserted")
	}
	for i, p := range whenWritten {
		if p != "" {
			t.Fatalf("write %d of %d landed with a prompt on the frame: %q",
				i+1, len(whenWritten), p)
		}
	}
	// And it comes BACK: blanking the live edge for the whole session would
	// satisfy the loop above and leave a session with no prompt at all.
	if live == "" {
		t.Error("the prompt never returned after the work finished")
	}
}

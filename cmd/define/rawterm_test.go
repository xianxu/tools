package main

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

// readKeys is the boundary between bytes and the editor. It had no test in the
// default build: everything below ran only under the conformance tag, against a
// real pty, which is not where a decoding bug should first be found.

func collect(t *testing.T, in string, n int) []Key {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	keys := readKeys(ctx, strings.NewReader(in), &interrupter{fn: cancel})
	var got []Key
	for i := 0; i < n; i++ {
		select {
		case k, ok := <-keys:
			if !ok {
				return got
			}
			got = append(got, k)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out after %d keys", len(got))
		}
	}
	return got
}

func TestReadKeysDecodesAStream(t *testing.T) {
	got := collect(t, "ab\x1b[A\r", 4)
	want := []KeyKind{KeyRune, KeyRune, KeyUp, KeyEnter}
	if len(got) != len(want) {
		t.Fatalf("got %d keys, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Kind != w {
			t.Errorf("key %d = %v, want %v", i, got[i].Kind, w)
		}
	}
}

// An escape sequence split across reads must not decode as Escape-then-junk.
// This is the case that breaks only when the terminal is slow, which is exactly
// when a user is least able to explain it.
func TestReadKeysHandlesSequencesSplitAcrossReads(t *testing.T) {
	pr, pw := io.Pipe()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	keys := readKeys(ctx, pr, &interrupter{fn: cancel})

	go func() {
		pw.Write([]byte("\x1b"))
		time.Sleep(50 * time.Millisecond)
		pw.Write([]byte("["))
		time.Sleep(50 * time.Millisecond)
		pw.Write([]byte("A"))
	}()

	select {
	case k := <-keys:
		if k.Kind != KeyUp {
			t.Errorf("split sequence decoded as %v, want KeyUp", k.Kind)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a split escape sequence never produced a key")
	}
}

// Ctrl-C fires from the READER, because in raw mode it is a byte and the loop
// may be blocked in playback and unable to act on it.
//
// It fires the interrupt SINK rather than a context directly: #16 D5 made the
// sink the one answer to what an interrupt means, so that a question can scope
// it without the key reader knowing anything about questions.
func TestReadKeysFiresTheSinkOnInterrupt(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	fired := make(chan struct{}, 1)
	keys := readKeys(ctx, strings.NewReader("a\x03"), &interrupter{fn: func() { fired <- struct{}{} }})
	<-keys // the rune

	select {
	case <-fired:
	case <-time.After(2 * time.Second):
		t.Fatal("Ctrl-C did not reach the sink — cancellation would not reach a blocked loop")
	}
}

// restore must be safe to call more than once: it runs from a defer AND on the
// cancellation path, and a terminal left raw is unrecoverable for the user.
func TestRawSessionRestoreIsIdempotent(t *testing.T) {
	var r *rawSession
	r.restore() // nil receiver: the interrupt path may fire before entry
	r = &rawSession{fd: -1}
	r.restore()
	r.restore()
}

// restore hands back EVERY terminal state this program takes, in the order that
// makes each one safe (#30).
//
// The predecessor of this test could not fail. It built a session with a nil
// file, so every enter and every leave returned at the same nil guard and the
// assertion checked a field nothing had set: deleting the teardown from
// restore() left the whole suite green. That is why
// rawSession now writes its mode sequences to an io.Writer — the seam exists so
// this protocol is assertable without a terminal.
func TestRestoreHandsBackEveryTerminalState(t *testing.T) {
	var b strings.Builder
	// state is nil, so term.Restore is a no-op and what is under test is the
	// escape sequences and their ORDER.
	r := &rawSession{control: &b}

	r.enterModes()
	// DERIVED: entering runs enabledModes in reverse teardown order, so the alt
	// screen is taken first and the input modes apply to it. A literal here would
	// be a second statement of the order.
	var want string
	for i := len(enabledModes) - 1; i >= 0; i-- {
		want += enabledModes[i].on
	}
	if got := b.String(); got != want {
		t.Fatalf("entering wrote %q, want %q", got, want)
	}
	b.Reset()

	r.restore()
	got := b.String()
	if !strings.Contains(got, altScreenOff) {
		t.Error("restore left the terminal on the alternate screen: everything the user had scrolled back to stays hidden")
	}
	if !strings.Contains(got, mouseOff) {
		t.Error("restore left mouse reporting on: the next program run in this terminal gets escape sequences typed into it")
	}
	if !strings.Contains(got, pasteOff) {
		t.Error("restore left bracketed paste on: the next program gets ESC[200~ typed into it, which is the same class of mess as mouse reporting")
	}
	// ORDER, and it is not cosmetic. Mouse reporting goes first because it is
	// the state with no `reset` reflex behind it — the shell looks fine while
	// every click types garbage. The alternate screen goes before raw mode ends,
	// so a cooked terminal is never briefly drawing into a buffer about to be
	// discarded.
	if strings.Index(got, mouseOff) > strings.Index(got, altScreenOff) {
		t.Errorf("restore gave the terminal back in the wrong order: %q", got)
	}
	// Paste reporting is in the same class as the mouse — no `reset` reflex
	// behind it — so it goes back before the screen does, for the same reason.
	if strings.Index(got, pasteOff) > strings.Index(got, altScreenOff) {
		t.Errorf("restore gave paste mode back after the screen: %q", got)
	}
	if anyModeClaimed(r) {
		t.Error("restore returned with state still claimed")
	}
}

// A session that never took a state must not hand one back: a stray
// ESC[?1049l on a terminal that was never switched clears the user's screen.
func TestRestoreSendsNothingItDidNotTake(t *testing.T) {
	var b strings.Builder
	r := &rawSession{control: &b}
	r.restore()
	if got := b.String(); got != "" {
		t.Errorf("restore wrote %q for states it never entered", got)
	}
}

// A control stream that fails must not leave the session CLAIMING the state: a
// leave would then be sent for a screen the terminal never showed.
func TestEnterDoesNotClaimAStateItCouldNotWrite(t *testing.T) {
	r := &rawSession{control: failingWriter{}}
	r.enterModes()
	if anyModeClaimed(r) {
		t.Error("a failed write still claimed the terminal state")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// Idempotence, for the same reason restore has it: both run from more than one
// path, and making a second call an error would make the paths care about each
// other.
func TestLeaveModesIsIdempotent(t *testing.T) {
	var b strings.Builder
	r := &rawSession{control: &b}
	r.enterModes()
	b.Reset()
	r.leaveModes()
	first := b.String()
	r.leaveModes()
	if anyModeClaimed(r) {
		t.Error("leaveModes left a mode claimed")
	}
	if got := b.String(); got != first {
		t.Errorf("two leaves wrote %q, want one round of %q", got, first)
	}
	// And it gives back everything the list holds, in the list's own order.
	var want string
	for _, m := range enabledModes {
		want += m.off
	}
	if first != want {
		t.Errorf("leaveModes wrote %q, want %q — the order is the teardown order", first, want)
	}
}

// EVERY mode the program enables is written by NEWCONSOLE — the place it is
// actually enabled — derived from enabledModes rather than hand-written per mode.
//
// #67's BR-16, and the distinction is the whole finding: the test that existed
// called the mode enable itself, so it pinned the METHOD and never the call
// site. Deleting the call from newConsole left the suite green while the entire
// milestone silently reverted to a pasted newline submitting mid-paste. This
// drives newConsole, so removing any enter reddens.
func TestNewConsoleEnablesEveryMode(t *testing.T) {
	var control strings.Builder
	sess := &rawSession{control: &control}
	con := newConsole(t.Context(), testDeps(t), sess, io.Discard,
		func(tty io.Writer, rows, cols int) *liveScreen { return newLiveScreen(tty, rows, cols) }, false)
	if con.finish != nil {
		defer con.finish()
	}
	for _, m := range enabledModes {
		if !strings.Contains(control.String(), m.on) {
			t.Errorf("newConsole never enabled %s, so the terminal is never asked for it: %q",
				m.name, control.String())
		}
	}
}

// ...and every one is given back, in the ordered teardown, before raw mode ends.
func TestRestoreGivesBackEveryMode(t *testing.T) {
	var control strings.Builder
	sess := &rawSession{control: &control}
	sess.enterModes()
	control.Reset()
	sess.restore()
	for _, m := range enabledModes {
		if !strings.Contains(control.String(), m.off) {
			t.Errorf("%s was left ON: the next program run in this terminal inherits it (%q)",
				m.name, control.String())
		}
	}
}

// anyModeClaimed reports whether the session still believes it holds any mode.
//
// Derived from the session's own map rather than checking three named fields, so
// a mode added to enabledModes is covered without this helper changing.
func anyModeClaimed(r *rawSession) bool {
	for _, held := range r.modes {
		if held {
			return true
		}
	}
	return false
}

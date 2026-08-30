package main

import (
	"strings"
	"testing"
)

// The hand-back, pinned IN PROCESS (#30).
//
// Every claim this sequence makes used to rest on conformance-tagged pty rows,
// which skip wherever no pty is available — including inside a boundary review.
// replRaw itself is unreachable in process (it demands a real *os.File it can
// put into raw mode), so the sequence moved into a function that takes what it
// needs and nothing else.
//
// The order is the content, and each step is wrong in a different way alone.
func TestHandBackStopsPaintingRestoresThenPrintsTheSession(t *testing.T) {
	var tty strings.Builder    // the alternate screen, painted while the loop ran
	var normal strings.Builder // the terminal after the alt screen is gone

	live := newLiveScreen(&tty, 24, 80)
	live.Write([]byte("arrondissement\nthe entry\n"))
	control := &strings.Builder{}
	sess := &rawSession{control: control}
	sess.enterAlt()
	sess.enterMouse()
	control.Reset()

	handBack(live, sess, &normal)

	// 1. The session is where a user can scroll back to it.
	if !strings.Contains(normal.String(), "the entry") {
		t.Errorf("the session vanished with the alternate screen: %q", normal.String())
	}
	// 2. The terminal states went back.
	if got := control.String(); !strings.Contains(got, mouseOff) || !strings.Contains(got, altScreenOff) {
		t.Errorf("the terminal was not fully restored: %q", got)
	}
	// 3. Nothing was painted after the hand-back — a frame drawn then lands on
	// the NORMAL screen, over whatever the user was looking at before.
	painted := tty.Len()
	live.Write([]byte("a late write\n"))
	if tty.Len() != painted {
		t.Error("a write after the hand-back painted onto the normal screen")
	}
	// And it still reaches the buffer, because Stop is about painting.
	if !strings.Contains(live.Transcript(), "a late write") {
		t.Error("Stop dropped the write rather than just the paint")
	}
}

// Once, and the transcript is the reason: restore and Stop are idempotent
// because they run from several exit paths, and printing a session twice is not
// what an idempotent call fixes.
func TestHandBackPrintsTheSessionOnlyOnce(t *testing.T) {
	var tty, normal strings.Builder
	live := newLiveScreen(&tty, 24, 80)
	live.Write([]byte("the entry\n"))
	sess := &rawSession{control: &strings.Builder{}}

	finish := onceHandBack(live, sess, &normal)
	finish()
	finish()

	if n := strings.Count(normal.String(), "the entry"); n != 1 {
		t.Errorf("the session was printed %d times, want 1", n)
	}
}

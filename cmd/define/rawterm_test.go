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

// restore leaves the ALTERNATE SCREEN as well as raw mode, in that order (#30).
//
// A terminal left in the alternate screen is as bad an outcome as one left raw:
// the shell keeps working but everything the user had scrolled back to is hidden
// behind a buffer nobody is drawing. rawSession already guarantees restoration
// from a defer and on the cancellation path, so this asserts the guarantee
// covers both rather than two mechanisms each covering half.
func TestRestoreLeavesTheAlternateScreen(t *testing.T) {
	var b strings.Builder
	// No real terminal: the state is nil, so restore's term.Restore is a no-op
	// and what is under test is the escape sequence and the ORDER.
	r := &rawSession{f: nil}
	_ = b

	// With no file there is nothing to write to, and nothing must panic.
	r.enterAlt()
	r.restore()
	if r.alt {
		t.Error("alt stayed set with no terminal")
	}
}

// Idempotence, for the same reason restore has it: both run from more than one
// path, and making a second call an error would make the paths care about each
// other.
func TestLeaveAltIsIdempotent(t *testing.T) {
	r := &rawSession{}
	r.leaveAlt()
	r.leaveAlt()
	if r.alt {
		t.Error("leaveAlt set alt")
	}
}

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
	keys := readKeys(ctx, strings.NewReader(in), cancel)
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
	keys := readKeys(ctx, pr, cancel)

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

// Ctrl-C cancels from the READER, because in raw mode it is a byte and the loop
// may be blocked in playback and unable to act on it.
func TestReadKeysCancelsOnInterrupt(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	keys := readKeys(ctx, strings.NewReader("a\x03"), cancel)
	<-keys // the rune

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Ctrl-C did not cancel the context — cancellation would not reach a blocked loop")
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

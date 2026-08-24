package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// What Ctrl-C means right now is one value, and swapping it is scoped.
func TestInterrupterFiresWhatIsInstalled(t *testing.T) {
	var session, question int
	i := &interrupter{fn: func() { session++ }}

	i.Fire()
	if session != 1 || question != 0 {
		t.Fatalf("default sink: session=%d question=%d, want 1/0", session, question)
	}
	restore := i.Set(func() { question++ })
	i.Fire()
	if session != 1 || question != 1 {
		t.Fatalf("scoped sink: session=%d question=%d, want 1/1", session, question)
	}
	restore()
	i.Fire()
	if session != 2 || question != 1 {
		t.Fatalf("after restore: session=%d question=%d, want 2/1", session, question)
	}
}

// A nil sink must not panic: Fire races with whatever the loop is doing.
func TestInterrupterWithNothingInstalled(t *testing.T) {
	(&interrupter{}).Fire()
}

// PQ-1 and PQ-6, as one enumeration: {byte, signal} x {replRaw, replLines}.
// Every cell must reach the SAME sink, because a design fed by only one
// transport is cancelled out from under by the other, and a detach that serves
// both loops while only one is watched leaves the other uninterruptible.
func TestBothInterruptTransportsReachTheSink(t *testing.T) {
	t.Run("the byte transport fires the sink", func(t *testing.T) {
		fired := make(chan struct{}, 1)
		i := &interrupter{fn: func() { fired <- struct{}{} }}
		keys := readKeys(t.Context(), strings.NewReader("\x03"), i)
		<-keys // the key still reaches the loop
		select {
		case <-fired:
		default:
			t.Error("a decoded KeyInterrupt did not fire the sink")
		}
	})

	t.Run("the signal transport fires the same sink", func(t *testing.T) {
		sigs := make(chan os.Signal, 1)
		d := testDeps(t)
		d.notifySignals = func(...os.Signal) <-chan os.Signal { return sigs }
		assertSignalEndsTheLoop(t, d, sigs)
	})
}

// PQ-6's cell: replLines has no key reader, so the sink's default is its ONLY
// transport. A detach in run() with a watcher only in replRaw strands exactly
// this path — SIGINT diverted by NotifyContext and delivered to no one.
func TestThePipedLoopStillExitsOnASignal(t *testing.T) {
	sigs := make(chan os.Signal, 1)
	d := testDeps(t)
	d.stdinIsTerminal = func() bool { return false }
	d.notifySignals = func(...os.Signal) <-chan os.Signal { return sigs }
	assertSignalEndsTheLoop(t, d, sigs)
}

// assertSignalEndsTheLoop drives the injected signal transport and requires the
// loop to return — which is what the sink's DEFAULT does, so this is the same
// observable both loops must produce and neither may lose.
func assertSignalEndsTheLoop(t *testing.T, d deps, sigs chan os.Signal) {
	t.Helper()
	done := make(chan int, 1)
	pr, pw := io.Pipe() // never closed: only the signal may end this loop
	t.Cleanup(func() { pw.Close() })
	go func() {
		var out, errb bytes.Buffer
		done <- repl(t.Context(), d, options{times: 3, locale: "us"}, pr, &out, &errb)
	}()
	// Let the loop reach its select before the signal arrives.
	sigs <- os.Interrupt
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit = %d, want 0", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the loop did not return: this transport reaches no sink")
	}
}

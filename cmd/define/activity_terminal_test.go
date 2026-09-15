package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestPlainActivityHostReplacement(t *testing.T) {
	var out bytes.Buffer
	h := &plainActivityHost{out: &out}
	first, err := h.begin("⠋")
	if err != nil {
		t.Fatal(err)
	}
	if err := first.set("⠙"); err != nil {
		t.Fatal(err)
	}
	second, err := h.begin("⠋")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-first.revoked:
	default:
		t.Fatal("old lease still active")
	}
	first.clear()
	if err := first.set("STALE"); err != nil {
		t.Fatal(err)
	}
	if err := second.set("⠹"); err != nil {
		t.Fatal(err)
	}
	second.clear()
	second.clear()
	if got, want := out.String(), "⠋\r⠙\r⠋\r⠹"+eraseLine; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

type activityFailWriter struct{}

func (activityFailWriter) Write([]byte) (int, error) { return 0, errors.New("broken terminal") }
func TestPlainActivityHostBeginFailure(t *testing.T) {
	h := &plainActivityHost{out: activityFailWriter{}}
	if _, err := h.begin("⠋"); err == nil {
		t.Fatal("missing write error")
	}
	if h.current != nil {
		t.Fatal("failed begin retained lease")
	}
}

// The terminal accepts only part of one paint and violates Writer's usual
// error convention; a display worker must still treat that as a failed write.
type activityShortWriter struct{ writes, shortAt int }

func (w *activityShortWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == w.shortAt {
		return len(p) - 1, nil
	}
	return len(p), nil
}
func TestPlainActivityHostShortWrite(t *testing.T) {
	for _, shortAt := range []int{1, 2} {
		out := &activityShortWriter{shortAt: shortAt}
		h := &plainActivityHost{out: out}
		lease, err := h.begin("⠋")
		if shortAt == 2 {
			if err != nil {
				t.Fatal(err)
			}
			err = lease.set("⠙")
			lease.clear()
		}
		if !errors.Is(err, io.ErrShortWrite) {
			t.Errorf("write %d: got %v want short write", shortAt, err)
		}
		if h.current != nil {
			t.Fatal("failed activity retained lease")
		}
	}
}
func TestPlainActivityShortWriteStopsWorker(t *testing.T) {
	out := &activityShortWriter{shortAt: 2}
	h := &plainActivityHost{out: out}
	ticks := make(chan time.Time)
	ended := make(chan struct{})
	stop := startActivityTicks(context.Background(), h, func() (<-chan time.Time, func()) {
		return ticks, func() { close(ended) }
	})
	defer stop()
	ticks <- time.Now()
	select {
	case <-ended:
	case <-time.After(time.Second):
		t.Fatal("short write did not stop ticker")
	}
	stop()
	if h.current != nil {
		t.Fatal("worker ended without clearing lease")
	}
	select {
	case ticks <- time.Now():
		t.Fatal("worker still accepting ticks")
	default:
	}
}

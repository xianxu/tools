package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// memoryClipboard models a clipboard whose writes may finish, fail, or be cancelled.
type memoryClipboard struct {
	started      chan string
	release      chan error
	mu           sync.Mutex
	text         string
	writes       []string
	active, peak int
}

func newMemoryClipboard() *memoryClipboard {
	return &memoryClipboard{started: make(chan string, 16), release: make(chan error)}
}
func (m *memoryClipboard) Write(ctx context.Context, text string) error {
	m.mu.Lock()
	m.active++
	if m.active > m.peak {
		m.peak = m.active
	}
	m.writes = append(m.writes, text)
	m.mu.Unlock()
	defer func() { m.mu.Lock(); m.active--; m.mu.Unlock() }()
	m.started <- text
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-m.release:
		if err == nil {
			m.mu.Lock()
			m.text = text
			m.mu.Unlock()
		}
		return err
	}
}
func clipboardAwait[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("clipboard operation stalled")
		var z T
		return z
	}
}
func TestClipboardQueueOrdersAndBoundsAcceptedWrites(t *testing.T) {
	m := newMemoryClipboard()
	q := newClipboardQueue(m)
	defer q.Stop()
	done := make(chan error, 9)
	if err := q.Submit("first", func(e error) { done <- e }); err != nil {
		t.Fatal(err)
	}
	if got := clipboardAwait(t, m.started); got != "first" {
		t.Fatal(got)
	}
	for i := 0; i < 8; i++ {
		if err := q.Submit(fmt.Sprint(i), func(e error) { done <- e }); err != nil {
			t.Fatal(err)
		}
	}
	if err := q.Submit("overflow", nil); err == nil {
		t.Fatal("accepted ninth pending copy")
	}
	for i := 0; i < 9; i++ {
		m.release <- nil
		if err := clipboardAwait(t, done); err != nil {
			t.Fatal(err)
		}
		if i < 8 {
			if got := clipboardAwait(t, m.started); got != fmt.Sprint(i) {
				t.Fatalf("write %q out of order", got)
			}
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.peak != 1 || m.text != "7" {
		t.Fatalf("peak=%d clipboard=%q", m.peak, m.text)
	}
}
func TestClipboardQueueStopCancelsAndDropsPending(t *testing.T) {
	m := newMemoryClipboard()
	q := newClipboardQueue(m)
	done := make(chan error, 1)
	if err := q.Submit("active", func(e error) { done <- e }); err != nil {
		t.Fatal(err)
	}
	clipboardAwait(t, m.started)
	for i := 0; i < 8; i++ {
		if err := q.Submit("pending", func(error) { t.Error("dropped request completed") }); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Go(q.Stop)
	}
	wg.Wait()
	if err := clipboardAwait(t, done); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := q.Submit("late", nil); err == nil {
		t.Fatal("accepted after Stop")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.writes) != 1 || m.active != 0 {
		t.Fatalf("writes=%v active=%d", m.writes, m.active)
	}
}
func TestClipboardQueueFailureDoesNotBlockLaterCopy(t *testing.T) {
	m := newMemoryClipboard()
	q := newClipboardQueue(m)
	defer q.Stop()
	done := make(chan error, 2)
	for _, s := range []string{"bad", "good"} {
		if err := q.Submit(s, func(e error) { done <- e }); err != nil {
			t.Fatal(err)
		}
	}
	clipboardAwait(t, m.started)
	want := errors.New("busy")
	m.release <- want
	if got := clipboardAwait(t, done); got != want {
		t.Fatal(got)
	}
	clipboardAwait(t, m.started)
	m.release <- nil
	if got := clipboardAwait(t, done); got != nil {
		t.Fatal(got)
	}
}
func TestClipboardQueueRejectsInvalidPayloadBeforeAdmission(t *testing.T) {
	q := newClipboardQueue(newMemoryClipboard())
	defer q.Stop()
	for _, s := range []string{strings.Repeat("x", clipboardTextLimit+1), string([]byte{0xff})} {
		if err := q.Submit(s, nil); err == nil {
			t.Fatal("invalid text accepted")
		}
	}
}

func TestClipboardQueueCallbackCanSubmit(t *testing.T) {
	m := newMemoryClipboard()
	q := newClipboardQueue(m)
	defer q.Stop()
	submitted := make(chan error, 1)
	if err := q.Submit("first", func(error) { submitted <- q.Submit("second", nil) }); err != nil {
		t.Fatal(err)
	}
	clipboardAwait(t, m.started)
	m.release <- nil
	if err := clipboardAwait(t, submitted); err != nil {
		t.Fatal(err)
	}
	if got := clipboardAwait(t, m.started); got != "second" {
		t.Fatal(got)
	}
}

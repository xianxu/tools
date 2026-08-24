package main

import (
	"context"
	"sync"
)

// interrupter is what Ctrl-C means RIGHT NOW.
//
// Two transports deliver it and neither may own the meaning. In raw mode
// term.MakeRaw clears ISIG, so \x03 is a byte the key reader decodes — but the
// pty suite measured a \x03 arriving as a SIGINT instead
// (pty_conformance_test.go:13-25), which leaves through main's NotifyContext.
// A scoped cancel fed by only one of the two is cancelled out from under by the
// other, ending a session mid-answer whatever this holds (#16 D5).
//
// So both transports feed this, and whoever owns the foreground decides what it
// does: the loop installs its own cancel by default, and the ask path points it
// at one question for the duration of a stream.
type interrupter struct {
	mu sync.Mutex
	fn context.CancelFunc
}

// Set installs fn until the returned func puts back what was there.
func (i *interrupter) Set(fn context.CancelFunc) (restore func()) {
	i.mu.Lock()
	defer i.mu.Unlock()
	prev := i.fn
	i.fn = fn
	return func() {
		i.mu.Lock()
		defer i.mu.Unlock()
		i.fn = prev
	}
}

// Fire calls whatever is installed. Safe with nothing installed: it races with
// whatever the loop is doing, and a panic here would be a panic in the key
// reader's goroutine.
func (i *interrupter) Fire() {
	i.mu.Lock()
	fn := i.fn
	i.mu.Unlock()
	if fn != nil {
		fn()
	}
}

package main

import (
	"context"
	"os"
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
	// scoped is set while something narrower than the session owns the
	// interrupt — a streaming answer. It is what lets Fire tell its caller that
	// the interrupt has been CONSUMED, which the key reader needs: an interrupt
	// that cancelled a stream must not also be delivered to the loop, where it
	// would be applied as "quit" the moment the answer ended.
	scoped bool
}

// Set installs fn until the returned func puts back what was there, and marks
// the interrupt as owned by something narrower than the session.
func (i *interrupter) Set(fn context.CancelFunc) (restore func()) {
	i.mu.Lock()
	defer i.mu.Unlock()
	prevFn, prevScoped := i.fn, i.scoped
	i.fn, i.scoped = fn, true
	return func() {
		i.mu.Lock()
		defer i.mu.Unlock()
		i.fn, i.scoped = prevFn, prevScoped
	}
}

// Fire calls whatever is installed and reports whether a SCOPE consumed it.
//
// Safe with nothing installed: it races with whatever the loop is doing, and a
// panic here would be a panic in the key reader's goroutine.
func (i *interrupter) Fire() (consumed bool) {
	i.mu.Lock()
	fn, consumed := i.fn, i.scoped
	i.mu.Unlock()
	if fn != nil {
		fn()
	}
	return consumed
}

// detachedInterrupts detaches ctx from signal cancellation and routes SIGINT to
// a sink the caller owns.
//
// ONE body, because both interactive loops need exactly this and the second copy
// was verbatim (BR-47). A third is coming with #7's form, and the reasoning below
// is what each copy would have to re-derive.
//
// The loop owns what Ctrl-C MEANS — stop the answer, not the session — so it must
// not also be cancelled behind the sink's back by main's NotifyContext, or a
// SIGINT would end the session while an answer streams, whatever the sink points
// at.
//
// The detach happens BEFORE the choice of loop, not inside one, because every
// transport needs it: detaching in run() and watching only in the raw loop leaves
// every piped, redirected or raw-mode-fallback run with a ctx.Done() nothing can
// reach — SIGINT diverted from default termination by NotifyContext, and then
// delivered to no one (PQ-6).
//
// The caller defers the returned cancel.
func detachedInterrupts(ctx context.Context, d deps) (context.Context, *interrupter, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	interrupts := &interrupter{fn: cancel}
	if d.notifySignals != nil {
		go func() {
			for range d.notifySignals(os.Interrupt) {
				interrupts.Fire()
			}
		}()
	}
	return ctx, interrupts, cancel
}

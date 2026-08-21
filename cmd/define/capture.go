package main

import (
	"fmt"
	"io"
	"sync"

	"github.com/xianxu/tools/cmd/define/store"
)

// captureDecision is how far a lookup gets recorded.
type captureDecision int

const (
	captureNothing captureDecision = iota
	captureEventOnly
	captureEventAndWord
)

// decideCapture is the ONE place that answers "does this lookup get recorded".
//
// The opt-out arrives as opt.noCapture, set once at flag parse, so the
// environment is an input to this policy rather than a second mechanism beside
// it — an earlier design had both this and a null object, and "is capture off?"
// would have drifted the moment either grew a case.
func decideCapture(found bool, opt options) captureDecision {
	switch {
	case opt.noCapture:
		return captureNothing
	case opt.raw:
		// The scripting form. Piping a dictionary through a script must not
		// mutate the deck it happens to be standing in.
		return captureNothing
	case !found:
		// A failed lookup is still history: #14's Up-arrow must recall the typo
		// you just made, and #15's /history filters on Found. It is not
		// vocabulary, so it never reaches the deck.
		return captureEventOnly
	default:
		return captureEventAndWord
	}
}

// Capturer records a lookup. It returns NO error on purpose: capture must never
// change the outcome of a lookup, so a store failure warns and the definition
// still prints.
type Capturer interface {
	Capture(word string, found bool, opt options)
}

// storeCapturer is the only thing in the process that writes to the store.
type storeCapturer struct {
	mu     sync.Mutex
	st     store.Store
	clock  store.Clock
	warn   io.Writer
	warned bool // once per process, not once per lookup
}

func newStoreCapturer(st store.Store, clock store.Clock, warn io.Writer) *storeCapturer {
	return &storeCapturer{st: st, clock: clock, warn: warn}
}

func (c *storeCapturer) Capture(word string, found bool, opt options) {
	d := decideCapture(found, opt)
	if d == captureNothing {
		return
	}
	now := c.clock.Now()
	if err := c.st.AppendEvent(store.ReviewEvent{
		Word: word, Kind: store.EventLookedUp, Found: found, At: now,
	}); err != nil {
		c.warnf("could not record %q: %v", word, err)
		return
	}
	if d != captureEventAndWord {
		return
	}
	if err := c.st.Upsert(store.Word{Text: word, FirstSeen: now, LastSeen: now, Lookups: 1}); err != nil {
		c.warnf("could not record %q: %v", word, err)
	}
}

// warnf reports at most once per process. A directory that cannot be written is
// a standing condition, not news on every lookup.
func (c *storeCapturer) warnf(format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.warn == nil || c.warned {
		return
	}
	c.warned = true
	fmt.Fprintf(c.warn, "define: "+format+"\n", args...)
}

// noopCapturer is what a store-less run uses. Not a policy: the policy is
// decideCapture. This is only "there is nowhere to write".
type noopCapturer struct{}

func (noopCapturer) Capture(string, bool, options) {}

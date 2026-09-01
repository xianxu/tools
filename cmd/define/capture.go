package main

import (
	"io"
	"sync"

	"github.com/xianxu/tools/cmd/define/play"
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
	// CaptureAsk records a question (#16). Here rather than through the store
	// directly, so the event log keeps ONE write path: main.go's deps comment
	// says capture is the only thing that records, and a second appender beside
	// it is how that stops being true without anyone noticing.
	//
	// word is the word the question followed, and may be empty. The QUESTION is
	// recorded and the answer is not — #17 wants what the learner asked about,
	// and every consumer of this log is a fold.
	CaptureAsk(word, question string, opt options)
	// CaptureReview records the outcome of one review (#6). The third verb, here
	// for the same reason CaptureAsk is: capture is the ONLY thing that records,
	// and a second appender beside it is how that stops being true without
	// anyone noticing.
	//
	// A bool rather than a verdict, because a SKIP never reaches this: play's
	// Apply emits no record outcome for one, so the type refusing to express it
	// is the design rather than a gap. A recorded skip would read as a miss in
	// schedule.Fold and demote a word the learner was honest about.
	//
	// TAKES THE WHOLE OUTCOME rather than a widening list of positional
	// arguments. It was `(word, correct bool, axis, opt)`, and #39 needed to add
	// `unaided bool` — which would have put two adjacent swappable bools at the
	// call site, in the same change that introduced schedule.Grade to remove
	// one. The Outcome already carries every field, so the swap becomes
	// unexpressible.
	CaptureReview(out play.Outcome, opt options)
}

// storeCapturer is the only thing that RECORDS a lookup. It is not the only
// thing that writes: --forget deletes a word file through store.Forget. That
// one is deliberate, user-initiated and loud, where capture is automatic and
// silent — which is why they are separate seams.
type storeCapturer struct {
	mu     sync.Mutex
	st     store.Store
	clock  store.Clock
	warn   io.Writer
	warned bool // once per process, not once per lookup
	vocab  Vocabulary
}

// vocab is the highlight set this capturer grows. Explicit in the constructor
// rather than a settable field: a set that silently stayed empty because a
// caller forgot to wire it is the kind of quiet nothing this package has been
// bitten by before.
func newStoreCapturer(st store.Store, clock store.Clock, warn io.Writer, vocab Vocabulary) *storeCapturer {
	return &storeCapturer{st: st, clock: clock, warn: warn, vocab: vocab}
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
		return
	}
	// Only after the deck actually took it: the highlight set says "this is in
	// your deck", so it must not claim a word the deck rejected.
	if c.vocab != nil {
		c.vocab.Add(word)
	}
}

// CaptureReview appends one EventReviewed, immediately.
//
// Called the instant a verdict is graded, before the next question is drawn —
// which is what makes Ctrl-C mid-session lossless by construction rather than by
// a flush at the end. That property is free from the append-only log (#3) and
// would be lost by any batching.
func (c *storeCapturer) CaptureReview(out play.Outcome, opt options) {
	// decideCapture, not a second policy: -raw and DEFINE_NO_CAPTURE mean "write
	// nothing into this directory", and a review is a write. captureEventOnly is
	// the right floor — a review is an EVENT, and it must never touch the deck,
	// which is #4's job.
	if decideCapture(true, opt) == captureNothing {
		return
	}
	key := store.Key(out.Word)
	if key == "" {
		return
	}
	if err := c.st.AppendEvent(store.ReviewEvent{
		Word: key, Kind: store.EventReviewed, Found: true, Correct: out.Verdict == play.Correct,
		// An observation the session made, not a claim the learner asserted —
		// see play.SelfRated. It is what earns the ladder's two-rung promotion.
		Unaided: out.Unaided,
		// AxisNone stringifies to "", which omitempty drops — so D8 ("a correct
		// answer records no axis") is enforced by the type, not by a branch here
		// that a later caller could forget to write.
		Missed: out.Axis.String(), At: c.clock.Now(),
	}); err != nil {
		c.warnf("could not record the review of %q: %v", out.Word, err)
	}
}

func (c *storeCapturer) CaptureAsk(word, question string, opt options) {
	// The same opt-out governs both: DEFINE_NO_CAPTURE and -raw mean "write
	// nothing in this directory", and a question is a write. decideCapture is
	// asked with found=true because an answered question is not a failed
	// lookup — the distinction it draws is about the DECK, which an ask never
	// reaches.
	if decideCapture(true, opt) == captureNothing {
		return
	}
	if err := c.st.AppendEvent(store.ReviewEvent{
		Word: word, Kind: store.EventAsked, Question: question, At: c.clock.Now(),
	}); err != nil {
		c.warnf("could not record the question: %v", err)
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
	warnTo(c.warn, format, args...)
}

// noopCapturer is what a store-less run uses. Not a policy: the policy is
// decideCapture. This is only "there is nowhere to write".
type noopCapturer struct{}

func (noopCapturer) Capture(string, bool, options)       {}
func (noopCapturer) CaptureAsk(string, string, options)  {}
func (noopCapturer) CaptureReview(play.Outcome, options) {}

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"strconv"
	"strings"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
)

// Background preparation (#54): the interactive session keeps practice material
// current without a command. This file is its pure core — when a job runs, and
// what counts as work — and the session wires it to a goroutine.

// bgThreshold is how many deck words must need work before a job runs, and how
// many successful lookups pass between checks. The operator's number.
const bgThreshold = 10

// bgBudget is the most model calls one job may make: about bgThreshold words at
// roughly six calls each (band, author, entail, a veto per wrong answer). A larger
// backlog drains over several jobs.
const bgBudget = 60

// noBackgroundEnv turns background preparation off for a session when set to
// anything: background calls cost money for someone paying per call.
const noBackgroundEnv = "DEFINE_NO_BACKGROUND"

// bgPhase is where the session's background work stands.
type bgPhase int

const (
	bgIdle    bgPhase = iota // nothing running; lookups count toward the next check
	bgRunning                // one job in flight; lookups still count
	bgOff                    // the model or the deck failed: nothing more this session
)

// bgState is everything the session carries between events for background work.
// Two fields and a written table (stepBackground) rather than a set of flags,
// because the legal combinations are few and the table names every one.
type bgState struct {
	phase bgPhase
	since int // successful lookups since the last job started
}

// The events the session feeds stepBackground.
const (
	bgSessionStart = iota // the session began
	bgLookedUp            // a lookup succeeded
	bgJobDone             // a job finished; bgEvent.result says what it did
)

// bgEvent is one thing that happened to the session.
type bgEvent struct {
	kind   int
	result bgJobResult // bgJobDone only
}

// bgEffect is what the session must do after a step: start a job, or print a
// notice between prompts. Never both in one value.
type bgEffect struct {
	runJob bool
	notice string
}

// bgJobResult is what one job did.
type bgJobResult struct {
	authored      int      // practice items it wrote
	failed        []string // words the job ran for and could not finish
	reflected     bool     // the learner model was written or refreshed
	noModel       bool     // the model did not answer; the session stops trying
	deckErr       error    // the deck's files could not be read or written; the session stops trying
	reflectFailed bool     // a reflect ran and wrote nothing; not asked for again this session
}

// stepBackground is the transition table. Pure: the session applies its effects.
//
//	state    event          next                                   effects
//	idle     session start  running, since 0                       start a job
//	idle     looked up      since+1; at bgThreshold: running, 0    start a job at the threshold
//	running  looked up      since+1                                none
//	running  job done       off when the model or deck failed      its notices, then the stop's
//	running  job done       idle, or running again past threshold  a notice per thing it did
//	off      anything       off                                    none
//	idle     job done       idle                                   none (a job starts only from idle)
func stepBackground(s bgState, ev bgEvent) (bgState, []bgEffect) {
	switch s.phase {
	case bgIdle:
		switch ev.kind {
		case bgSessionStart:
			return bgState{phase: bgRunning}, []bgEffect{{runJob: true}}
		case bgLookedUp:
			s.since++
			if s.since >= bgThreshold {
				return bgState{phase: bgRunning}, []bgEffect{{runJob: true}}
			}
			return s, nil
		}
	case bgRunning:
		switch ev.kind {
		case bgLookedUp:
			s.since++
			return s, nil
		case bgJobDone:
			var effects []bgEffect
			for _, n := range bgNoticeFor(ev.result) {
				effects = append(effects, bgEffect{notice: n})
			}
			if ev.result.noModel || ev.result.deckErr != nil {
				return bgState{phase: bgOff, since: s.since}, effects
			}
			if s.since >= bgThreshold {
				return bgState{phase: bgRunning}, append(effects, bgEffect{runJob: true})
			}
			return bgState{phase: bgIdle, since: s.since}, effects
		}
	}
	return s, nil
}

// bgNoticeFor is what the session says about one job: a line per thing it did, in
// the order it did them (the learner model is written before the harvest that
// reads it), then the stop, when one turned background work off. A fold with no
// early return, so a stop never hides what the job had already written.
func bgNoticeFor(r bgJobResult) []string {
	var out []string
	if r.reflected {
		out = append(out, "learner model updated")
	}
	if r.authored > 0 {
		s := "s"
		if r.authored == 1 {
			s = ""
		}
		out = append(out, fmt.Sprintf("%d new practice question%s ready for /play", r.authored, s))
	}
	switch {
	case r.noModel:
		out = append(out, "practice questions are not being prepared: the model did not answer (see define --llm-check)")
	case r.deckErr != nil:
		out = append(out, "practice questions are not being prepared: "+r.deckErr.Error())
	}
	return out
}

// pendingWords is the deck words --harvest would still do work for, the Spec's
// definition: no band yet, or no practice item. Newest first, because Deck() is
// ordered by last seen and the words being learned now matter most, and minus
// skip, the words a job already could not finish this session. deck is the deck as
// its caller read it, so a job reads it once.
//
// The store is the only count. Words looked up from the command line, harvested
// by hand, or forgotten are counted the same way, which a counter kept by the
// session could not do.
func pendingWords(st store.Store, deck []store.Word, skip map[string]bool) ([]string, error) {
	var out []string
	for _, w := range deck {
		if skip[w.Text] {
			continue
		}
		f, err := st.WordFacts(w.Text)
		if err != nil {
			return nil, err
		}
		if !f.Harvested() {
			out = append(out, w.Text)
			continue
		}
		items, err := st.Items(w.Text)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			out = append(out, w.Text)
		}
	}
	return out, nil
}

// runBackgroundJob is one job: write the learner model when it is due, then list
// what needs work and, when there is enough of it, band and author the newest
// bgThreshold words within bgBudget calls. The model comes first because
// authoring reads it (readLearner). Its prose goes to io.Discard and it reads the
// store through quietStore, so only its result reaches the screen, through the
// session. skip is what a job already could not finish this session.
func runBackgroundJob(ctx context.Context, d deps, skip bgMemory) bgJobResult {
	if !hasModelSeam(d) {
		return bgJobResult{} // tests call the job directly; a session checked already
	}
	d.deck = quietStore(d.deck)
	cfg, err := llm.Resolve(d.getenv)
	if err != nil {
		// No usable configuration, the case --llm-check reports loudly. The session
		// stops asking rather than fail the same way at every check.
		return bgJobResult{noModel: true}
	}
	deck, err := d.deck.Deck()
	if err != nil {
		return bgJobResult{deckErr: deckIO(err)}
	}
	var res bgJobResult
	// Only a deck at the reflect floor can be due, so a smaller one reads nothing
	// more; a reflect that already wrote nothing this session is not asked again.
	if !skip.reflectFailed && len(deck) >= minDeckForReflection {
		ran, written, err := reflectIfDue(ctx, d, cfg, deck)
		res.reflected, res.reflectFailed = written, ran && !written
		if res.noModel, res.deckErr = stopMeans(err); res.noModel || res.deckErr != nil {
			return res
		}
	}
	pending, err := pendingWords(d.deck, deck, skip.words)
	if err != nil {
		res.deckErr = deckIO(err)
		return res
	}
	if len(pending) < bgThreshold {
		return res
	}
	o := harvestDeck(ctx, d, d.newLLM(cfg), deck, &budget{left: bgBudget}, bgBudget, pending[:bgThreshold], io.Discard, io.Discard)
	res.authored, res.failed = o.authored, o.failed
	res.noModel, res.deckErr = stopMeans(o.stopped)
	return res
}

// bgMemory is what a session remembers between jobs, so that what a job tried and
// could not finish is not tried again this session: the words, and a learner
// model it could not write. A job gets a copy; received merges each result back.
type bgMemory struct {
	words         map[string]bool // words a job could not finish
	reflectFailed bool            // a reflect ran and wrote nothing
}

// bgRunner owns the session's one background goroutine. The session owns the
// state (stepBackground); the runner only starts a job and hands its result back
// on results, which the editor loop selects on beside resizes.
//
// Its context is a child of the session's, so quitting cancels the job, while a
// scoped Ctrl-C that stops a streamed answer does not. Every method is called
// from the loop's goroutine; the job's goroutine touches only its own copies and
// the results channel.
type bgRunner struct {
	ctx     context.Context
	cancel  context.CancelFunc
	job     func(context.Context, deps, bgMemory) bgJobResult
	results chan bgJobResult // capacity 1: one job at a time, so a send never waits on the loop
	done    chan struct{}    // closed when the latest job's goroutine has returned
	tried   bgMemory         // what a job could not finish this session
}

// newBgRunner is a runner under the session's context, running job.
func newBgRunner(parent context.Context, job func(context.Context, deps, bgMemory) bgJobResult) *bgRunner {
	ctx, cancel := context.WithCancel(parent)
	return &bgRunner{ctx: ctx, cancel: cancel, job: job, results: make(chan bgJobResult, 1), tried: bgMemory{words: map[string]bool{}}}
}

// start runs one job with the session's deps as they are now, so a job finishes
// the language it started in even if /lang switches mid-job. The job's skip is a
// copy of tried, so the two goroutines never share a map.
func (r *bgRunner) start(d deps) {
	skip := bgMemory{words: maps.Clone(r.tried.words), reflectFailed: r.tried.reflectFailed}
	done := make(chan struct{})
	r.done = done
	go func() {
		defer close(done)
		res := r.job(r.ctx, d, skip)
		select {
		case r.results <- res:
		case <-r.ctx.Done(): // the session is gone, and nobody will read it
		}
	}()
}

// received is the loop's half of a result: the words a job could not finish join
// tried, and so does a learner model it could not write, so no later job this
// session spends calls on them again.
func (r *bgRunner) received(res bgJobResult) {
	for _, w := range res.failed {
		r.tried.words[w] = true
	}
	r.tried.reflectFailed = r.tried.reflectFailed || res.reflectFailed
}

// stop cancels the job and waits up to wait for its goroutine to return. Every
// store write is atomic, so any stop is safe; the wait lets a write in flight
// finish rather than leave a temp file behind.
func (r *bgRunner) stop(wait time.Duration) {
	r.cancel()
	if r.done == nil {
		return
	}
	select {
	case <-r.done:
	case <-time.After(wait):
	}
}

// backgroundEnabled is whether a session may prepare practice in the background: a
// deck is open, the directory already agreed to be one (repl settles that before
// the loop starts, and this only reads the answer, never asks), a model seam
// exists, and DEFINE_NO_BACKGROUND is unset.
func backgroundEnabled(d deps) bool {
	if !hasModelSeam(d) {
		return false
	}
	if allowed, decided := d.deckPermission.saving(); !allowed || !decided {
		return false
	}
	return d.getenv(noBackgroundEnv) == ""
}

// hasModelSeam is whether deps has a deck to read and a model to ask: the
// session's gate checks it, and so does the job, which tests call directly.
func hasModelSeam(d deps) bool {
	return d.deck != nil && d.getenv != nil && d.newLLM != nil
}

// quietStore is the job's view of the deck: the same files, with the store's
// warnings dropped (#54). The session's stores warn to the process stderr, and the
// job reads off the loop, so a warning about a bad file would land in the frame at
// an arbitrary moment. A gated store keeps its gate and gets a quiet disk; the
// in-memory store never warns.
func quietStore(st store.Store) store.Store {
	switch s := st.(type) {
	case *gatedStore:
		q := *s
		q.disk = quietStore(s.disk)
		return &q
	case *store.YAML:
		return s.Quiet()
	}
	return st
}

// bgRefreshFactor is how far the deck's lookups must grow past the count a learner
// model records before the session rewrites the model. A level is stable, so the
// model is refreshed rarely: at double, then at double again.
const bgRefreshFactor = 2

// reflectIfDue writes the learner model when it is due (#54). It reads the model on
// disk first, and the lookup log, the largest thing a check reads, only when that
// model could be due. ran says a reflect was attempted and written that it wrote
// the file; err is what stopped it, a read error marked errDeckIO, or nil.
func reflectIfDue(ctx context.Context, d deps, cfg llm.Config, deck []store.Word) (ran, written bool, err error) {
	md, err := d.deck.UserModel()
	if err != nil {
		return false, false, deckIO(err)
	}
	if !reflectCouldBeDue(md) {
		return false, false, nil
	}
	events, err := d.deck.Events(time.Time{})
	if err != nil {
		return false, false, deckIO(err)
	}
	ev := foldLookups(deck, events, d.clock.Now())
	if !reflectDue(md, ev.DeckLookups(), len(ev.Words)) {
		return false, false, nil
	}
	o := reflectDeck(ctx, d, d.newLLM(cfg), cfg.Model, ev, io.Discard, io.Discard)
	return true, o.written, o.stopped
}

// reflectCouldBeDue is whether any deck could make the learner model due: there is
// none yet, or the count it records can be read. A model someone edited by hand is
// left alone whatever the deck holds, so for it the job reads no lookup log.
func reflectCouldBeDue(md string) bool {
	if strings.TrimSpace(md) == "" {
		return true
	}
	_, ok := modelLookups(md)
	return ok
}

// reflectDue is whether the session should write the learner model now: there is
// none yet and the deck has reached minDeckForReflection words, or the session can
// read the one there and the deck's lookups have grown to bgRefreshFactor times
// what it records. Nothing is due below the floor, where --reflect itself refuses;
// unknown is not due, so a model someone edited by hand is left alone; and a model
// written from no lookups waits for one, because doubling nothing is no growth.
func reflectDue(md string, lookups, words int) bool {
	if words < minDeckForReflection || !reflectCouldBeDue(md) {
		return false
	}
	if strings.TrimSpace(md) == "" {
		return true
	}
	recorded, _ := modelLookups(md)
	return lookups > recorded && lookups >= bgRefreshFactor*recorded
}

// modelLookups is the lookup count a learner model was written from, read off its
// frontmatter's window: line (renderUserModel writes "# N lookups"). The
// frontmatter only, as parseLearnerBand reads it, and only a terminated one: this
// count decides a paid call, so a number in the Corrections a person writes, or in
// a truncated file, reads as unknown.
func modelLookups(md string) (int, bool) {
	lines := strings.Split(md, "\n")
	if strings.TrimSpace(lines[0]) != "---" {
		return 0, false
	}
	n, found := 0, false
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			return n, found
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok || found || strings.TrimSpace(key) != "window" {
			continue
		}
		_, comment, ok := strings.Cut(val, "#")
		fields := strings.Fields(comment)
		if !ok || len(fields) < 2 || strings.TrimSuffix(fields[1], ",") != "lookups" {
			continue
		}
		if v, err := strconv.Atoi(fields[0]); err == nil && v >= 0 {
			n, found = v, true
		}
	}
	return 0, false // no closing fence: a truncated file, not a model
}

// errDeckIO marks a stop that came from the deck's own files rather than the
// model: a read or a write the store refused. It repeats at every check, so the
// session says it once and stops, as it does for a model that will not answer.
var errDeckIO = errors.New("the deck could not be read or written")

// deckIO marks err as the deck's, where the store returned it.
func deckIO(err error) error { return fmt.Errorf("%w: %w", errDeckIO, err) }

// stopMeans is what a pass's stop means for the session, decided by the error's
// kind in this one place. A model that is not answering, or that refuses every
// request (the two stops internal/llm/errors.go says repeat on every call), and a
// deck whose files fail both turn background work off. Any other stop, a
// malformed answer or a cancel, leaves the session trying at the next check; a
// word it stopped on is already reported unfinished.
func stopMeans(err error) (noModel bool, deckErr error) {
	switch {
	case errors.Is(err, llm.ErrUnavailable), errors.Is(err, llm.ErrRequest):
		return true, nil
	case errors.Is(err, errDeckIO):
		return false, err
	}
	return false, nil
}

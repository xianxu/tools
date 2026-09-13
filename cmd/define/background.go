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
	bgOff                    // the model did not answer: nothing more this session
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
	authored  int      // practice items it wrote
	failed    []string // words the job ran for and could not finish
	reflected bool     // the learner model was written or refreshed
	noModel   bool     // the model did not answer; the session stops trying
}

// stepBackground is the transition table. Pure: the session applies its effects.
//
//	state    event          next                                   effects
//	idle     session start  running, since 0                       start a job
//	idle     looked up      since+1; at bgThreshold: running, 0    start a job at the threshold
//	running  looked up      since+1                                none
//	running  job done       off when the model did not answer      one notice
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
			if ev.result.noModel {
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

// bgNoticeFor is what the session says about one job, in the order the job did
// it: the learner model is written before the harvest that reads it.
func bgNoticeFor(r bgJobResult) []string {
	if r.noModel {
		return []string{"practice questions are not being prepared: the model did not answer (see define --llm-check)"}
	}
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

// runBackgroundJob is one job: list what needs work and, when there is enough of
// it, band and author the newest bgThreshold words within bgBudget calls. Its
// prose goes to io.Discard and it reads the store through quietStore, so only its
// result reaches the screen, through the session. skip is the words a job already
// could not finish this session.
func runBackgroundJob(ctx context.Context, d deps, skip map[string]bool) bgJobResult {
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
		return bgJobResult{}
	}
	pending, err := pendingWords(d.deck, deck, skip)
	if err != nil || len(pending) < bgThreshold {
		return bgJobResult{}
	}
	o := harvestDeck(ctx, d, d.newLLM(cfg), deck, &budget{left: bgBudget}, bgBudget, pending[:bgThreshold], io.Discard, io.Discard)
	return bgJobResult{
		authored: o.authored,
		failed:   o.failed,
		// The two stops that repeat on every call (internal/llm/errors.go): no model
		// answering, or a request it will always refuse. Any other stop, a malformed
		// answer or a cancel, leaves the session trying at the next check.
		noModel: errors.Is(o.stopped, llm.ErrUnavailable) || errors.Is(o.stopped, llm.ErrRequest),
	}
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
	job     func(context.Context, deps, map[string]bool) bgJobResult
	results chan bgJobResult // capacity 1: one job at a time, so a send never waits on the loop
	done    chan struct{}    // closed when the latest job's goroutine has returned
	tried   map[string]bool  // words a job could not finish this session
}

// newBgRunner is a runner under the session's context, running job.
func newBgRunner(parent context.Context, job func(context.Context, deps, map[string]bool) bgJobResult) *bgRunner {
	ctx, cancel := context.WithCancel(parent)
	return &bgRunner{ctx: ctx, cancel: cancel, job: job, results: make(chan bgJobResult, 1), tried: map[string]bool{}}
}

// start runs one job with the session's deps as they are now, so a job finishes
// the language it started in even if /lang switches mid-job. The job's skip is a
// snapshot of tried, so the two goroutines never share a map.
func (r *bgRunner) start(d deps) {
	skip := maps.Clone(r.tried)
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
// tried, so no later job this session spends calls on them again.
func (r *bgRunner) received(res bgJobResult) {
	for _, w := range res.failed {
		r.tried[w] = true
	}
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

// reflectDue is whether the session should write the learner model now: there is
// none yet and the deck has reached minDeckForReflection words, or the session can
// read the one there and the deck's lookups have grown to bgRefreshFactor times
// what it records. Nothing is due below the floor, where --reflect itself refuses;
// unknown is not due, so a model someone edited by hand is left alone; and a model
// written from no lookups waits for one, because doubling nothing is no growth.
func reflectDue(md string, lookups, words int) bool {
	if words < minDeckForReflection {
		return false
	}
	if strings.TrimSpace(md) == "" {
		return true
	}
	recorded, ok := modelLookups(md)
	return ok && lookups > recorded && lookups >= bgRefreshFactor*recorded
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

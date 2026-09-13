package main

import (
	"fmt"

	"github.com/xianxu/tools/cmd/define/store"
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

// bgPhase is where the session's background work stands.
type bgPhase int

const (
	bgIdle    bgPhase = iota // nothing running; lookups count toward the next check
	bgRunning                // one job in flight; lookups still count
	bgOff                    // no model answered: nothing more this session
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
	failed    []string // words whose authoring ran and kept nothing
	reflected bool     // the learner model was written or refreshed
	noModel   bool     // no model answered; the session stops trying
}

// stepBackground is the transition table. Pure: the session applies its effects.
//
//	state    event          next                                   effects
//	idle     session start  running, since 0                       start a job
//	idle     looked up      since+1; at bgThreshold: running, 0    start a job at the threshold
//	running  looked up      since+1                                none
//	running  job done       off when no model answered             one notice
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
		return []string{"practice questions are not being prepared: no model answered (see define --llm-check)"}
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
// skip, the words whose authoring already failed this session.
//
// The store is the only count. Words looked up from the command line, harvested
// by hand, or forgotten are counted the same way, which a counter kept by the
// session could not do.
func pendingWords(st store.Store, skip map[string]bool) ([]string, error) {
	deck, err := st.Deck()
	if err != nil {
		return nil, err
	}
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

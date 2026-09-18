package main

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/xianxu/tools/cmd/define/store"
)

// schemeSource says where the scheme in effect came from, so /scheme can say
// it truthfully. sourceDefault means nothing was chosen and nothing detected.
type schemeSource int

const (
	sourceDefault schemeSource = iota
	sourceDetected
	sourceFlag
	sourceSaved
	sourceSession
)

// schemeState is the whole of what the process knows about its scheme (#70).
//
// IMMUTABLE: every transition returns a new value, and schemeHolder swaps it in.
// Two independent facts, so their product is the legal state space: an explicit
// choice (chosenBy is flag, saved or session; sourceDefault means none), and
// what the terminal last reported (heard). The choice outranks the report; the
// report is kept underneath, so clearing the choice reveals it.
type schemeState struct {
	choice   store.Scheme
	chosenBy schemeSource
	detected store.Scheme
	heard    bool
}

func (s schemeState) effective() (store.Scheme, schemeSource) {
	if s.chosenBy != sourceDefault {
		return s.choice, s.chosenBy
	}
	if s.heard {
		return s.detected, sourceDetected
	}
	return store.SchemeDark, sourceDefault
}

// withChoice records an explicit choice. by is sourceFlag, sourceSaved or
// sourceSession; it replaces any earlier choice whatever its source.
func (s schemeState) withChoice(v store.Scheme, by schemeSource) schemeState {
	s.choice, s.chosenBy = v, by
	return s
}

func (s schemeState) withoutChoice() schemeState {
	s.choice, s.chosenBy = "", sourceDefault
	return s
}

func (s schemeState) withDetected(v store.Scheme) schemeState {
	s.detected, s.heard = v, true
	return s
}

// schemeHolder is the ONE per-process home of schemeState, shared by the editor,
// a sitting it starts, and every screen and writer that paints a tint.
//
// An atomic pointer to an immutable value rather than a mutex: five goroutines
// read it while painting under the screen lock (the loop, the throttle timer,
// the activity ticker, readInput's pointer routing, clipboard completion), and a
// load takes no lock, so there is no ordering to get wrong against l.mu or the
// pointer router's. There is exactly ONE writer at a time — the loop goroutine in
// force (a /play sitting runs on the editor loop's own goroutine) — so a
// load-transition-store needs no compare-and-swap.
//
// A nil holder paints dark and cannot change: that is a test's bare deps, and
// run() always builds one.
type schemeHolder struct{ p atomic.Pointer[schemeState] }

func newSchemeHolder(s schemeState) *schemeHolder {
	h := &schemeHolder{}
	h.p.Store(&s)
	return h
}

func (h *schemeHolder) Load() schemeState {
	if h == nil {
		return schemeState{}
	}
	if s := h.p.Load(); s != nil {
		return *s
	}
	return schemeState{}
}

// The holder's TRANSITIONS are its only writers (ARCH-ORDER structural
// enforcement): choose, forget and detect, each applying one pure schemeState
// transition and reporting whether the painted shade changed. set is their
// shared step; nothing outside this file calls it. Callers run on the loop in
// force, and a nil holder changes nothing.
func (h *schemeHolder) set(next schemeState) bool {
	a, _ := h.Load().effective()
	h.p.Store(&next)
	b, _ := next.effective()
	return a != b
}

func (h *schemeHolder) choose(v store.Scheme, by schemeSource) bool {
	return h != nil && h.set(h.Load().withChoice(v, by))
}

func (h *schemeHolder) forget() bool {
	return h != nil && h.set(h.Load().withoutChoice())
}

// Scheme is the value to paint with now.
func (h *schemeHolder) Scheme() store.Scheme {
	v, _ := h.Load().effective()
	return v
}

// detect applies a terminal report and says whether what is painted changed —
// the only case worth a repaint.
func (h *schemeHolder) detect(v store.Scheme) bool {
	return h != nil && h.set(h.Load().withDetected(v))
}

// schemeArg is what -scheme and /scheme accept: a scheme, or auto (no choice).
// ONE parser for both (ARCH-DRY), so the flag and the command cannot disagree.
type schemeArg struct {
	auto  bool
	value store.Scheme
}

func parseSchemeArg(s string) (schemeArg, error) {
	if strings.EqualFold(strings.TrimSpace(s), "auto") {
		return schemeArg{auto: true}, nil
	}
	v, err := store.ParseScheme(s)
	if err != nil {
		return schemeArg{}, fmt.Errorf("%q is not a colour scheme; use light, dark or auto", s)
	}
	return schemeArg{value: v}, nil
}

// parseTintFlag reads -language-tint, which #70 narrowed to on|off. Its old
// values name the shade, which -scheme owns now, so they are refused by name
// rather than guessed at (operator decision: narrowed, not aliased).
func parseTintFlag(s string) (bool, error) {
	switch v := strings.ToLower(strings.TrimSpace(s)); v {
	case "on":
		return true, nil
	case "off":
		return false, nil
	case "dark", "light":
		return false, fmt.Errorf("-language-tint is on or off now; the shade follows the colour scheme: use -scheme %s", v)
	}
	return false, fmt.Errorf("invalid -language-tint %q: use on or off", s)
}

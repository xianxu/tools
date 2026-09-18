package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
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

// choiceSource is WHO made an explicit choice: a strict subset of
// schemeSource, as its own type so a choice cannot claim to be detected or
// default — /scheme prints the source, and the report must be true.
type choiceSource int

const (
	choiceFlag choiceSource = iota
	choiceSaved
	choiceSession
)

func (c choiceSource) source() schemeSource {
	switch c {
	case choiceSaved:
		return sourceSaved
	case choiceSession:
		return sourceSession
	}
	return sourceFlag
}

// schemeChoice is an explicit choice. Never mutated once made.
type schemeChoice struct {
	value store.Scheme
	by    choiceSource
}

// schemeState is the whole of what the process knows about its scheme (#70).
//
// IMMUTABLE: every transition returns a new value, and schemeHolder swaps it in.
// Two independent facts, so their product is the legal state space: an explicit
// choice (nil for none), and what the terminal last reported (empty for
// nothing heard — one field, so "heard" and "what" cannot disagree). The choice
// outranks the report; the report is kept underneath, so clearing the choice
// reveals it.
type schemeState struct {
	choice   *schemeChoice
	detected store.Scheme
}

func (s schemeState) effective() (store.Scheme, schemeSource) {
	if s.choice != nil {
		return s.choice.value, s.choice.by.source()
	}
	if s.detected != "" {
		return s.detected, sourceDetected
	}
	return store.SchemeDark, sourceDefault
}

// withChoice records an explicit choice, replacing any earlier one whatever
// its source.
func (s schemeState) withChoice(v store.Scheme, by choiceSource) schemeState {
	s.choice = &schemeChoice{value: v, by: by}
	return s
}

func (s schemeState) withoutChoice() schemeState {
	s.choice = nil
	return s
}

func (s schemeState) withDetected(v store.Scheme) schemeState {
	s.detected = v
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

func (h *schemeHolder) choose(v store.Scheme, by choiceSource) bool {
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

// schemeArg is what -scheme and /scheme accept: a scheme, or auto — no choice.
// ONE field, not an auto flag beside a value: the empty value IS auto, so the
// two cannot contradict each other and the zero value is the harmless one (it
// forgets rather than saving a blank). ONE parser for the flag and the command
// (ARCH-DRY), so they cannot disagree either.
type schemeArg struct{ value store.Scheme }

func (a schemeArg) auto() bool { return a.value == "" }

func parseSchemeArg(s string) (schemeArg, error) {
	if strings.EqualFold(strings.TrimSpace(s), "auto") {
		return schemeArg{}, nil
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

// configDirFrom resolves define's user config directory (#70). Only ABSOLUTE
// bases count: a relative XDG_CONFIG_HOME or HOME would put the file wherever
// the process happens to stand — the deck's directory, the one place this
// setting must not live.
func configDirFrom(getenv func(string) string) (string, bool) {
	if x := getenv("XDG_CONFIG_HOME"); filepath.IsAbs(x) {
		return filepath.Join(x, "define"), true
	}
	if h := getenv("HOME"); filepath.IsAbs(h) {
		return filepath.Join(h, ".config", "define"), true
	}
	return "", false
}

var (
	errNoScheme      = errors.New("there is no colour scheme to change here")
	errNowhereToSave = errors.New("nowhere to save it: $XDG_CONFIG_HOME and $HOME are unset or not absolute")
)

// schemePersister is the durable half of the scheme: the startup read and
// /scheme's save and clear, through ONE seam. nil means there is no config
// directory.
type schemePersister interface {
	load() (store.Scheme, bool, error)
	save(store.Scheme) error
	clear() error
}

type dirSchemePersister string

func (d dirSchemePersister) load() (store.Scheme, bool, error) { return store.ReadScheme(string(d)) }
func (d dirSchemePersister) save(s store.Scheme) error         { return store.WriteScheme(string(d), s) }
func (d dirSchemePersister) clear() error                      { return store.ClearScheme(string(d)) }

// initialSchemeState is the scheme a process starts with: the -scheme flag,
// else the saved choice, else nothing (the dark default, or a later report).
// A saved file that cannot be read is untrusted input gone wrong, so it warns
// once and counts as unset. Pure apart from the persister and warn it is given.
func initialSchemeState(flag schemeArg, p schemePersister, warn io.Writer) schemeState {
	var st schemeState
	if !flag.auto() {
		return st.withChoice(flag.value, choiceFlag)
	}
	if p == nil {
		return st
	}
	v, found, err := p.load()
	if err != nil {
		fmt.Fprintf(warn, "define: ignoring saved scheme: %v\n", err)
		return st
	}
	if found {
		st = st.withChoice(v, choiceSaved)
	}
	return st
}

// schemePersister is this process's durable half of /scheme, or nil when there
// is no config directory to write to.
func (d deps) schemePersister() schemePersister {
	if d.configDir == nil {
		return nil
	}
	if dir, ok := d.configDir(); ok {
		return dirSchemePersister(dir)
	}
	return nil
}

// applyScheme is /scheme's transition. PERSIST, THEN SWITCH — /bilingual's rule
// (bilingual_cmd.go), not a second one: a failed write changes nothing, so the
// message can never claim a switch that did not persist. With nowhere to save, a
// session switches for itself alone and says so; a one-shot has nothing else to
// change, so it refuses.
func applyScheme(h *schemeHolder, arg schemeArg, p schemePersister, session bool) (schemeState, error) {
	if h == nil {
		return schemeState{}, errNoScheme
	}
	switch {
	case p == nil && !session:
		return h.Load(), errNowhereToSave
	case p == nil && arg.auto():
		h.forget()
	case p == nil:
		h.choose(arg.value, choiceSession)
	case arg.auto():
		if err := p.clear(); err != nil {
			return h.Load(), err
		}
		h.forget()
	default:
		if err := p.save(arg.value); err != nil {
			return h.Load(), err
		}
		h.choose(arg.value, choiceSaved)
	}
	return h.Load(), nil
}

// describeScheme is /scheme's report, and every wording is TRUE of its state:
// "has not reported" holds whether the reply is pending, unsupported or never
// asked for; outside a full-screen session nothing asks.
func describeScheme(s schemeState, fullScreen bool) string {
	v, src := s.effective()
	switch src {
	case sourceDetected:
		return string(v) + " (detected)"
	case sourceFlag:
		return string(v) + " (-scheme flag)"
	case sourceSaved:
		return string(v) + " (saved)"
	case sourceSession:
		return string(v) + " (session only; not saved)"
	}
	if fullScreen {
		return string(v) + " (default: the terminal has not reported its background)"
	}
	return string(v) + " (default: detected only in a full-screen session)"
}

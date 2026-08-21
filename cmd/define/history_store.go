package main

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// storeHistory makes a store.Store satisfy the editor's History.
//
// Two things are deliberately NOT the same here:
//
//   - Add always appends an EVENT, but upserts a Word only when the lookup
//     found something. Recall must include the typo you just made — that is when
//     you most want to edit and retry — while the deck must not fill with
//     misspellings.
//   - Prefix therefore reads the EVENT log, not the deck. Reading the deck would
//     silently drop every failed lookup from Up-arrow recall.
//
// History.Prefix runs on every keystroke and cannot return an error, so the log
// is loaded once at construction and held in memory. The store is the durable
// copy; the slice is what the editor talks to.
type storeHistory struct {
	mu     sync.Mutex
	lines  []string // oldest first, every submitted line
	st     store.Store
	clock  store.Clock
	warn   io.Writer
	warned bool // a write failure is reported once, not once per keystroke
}

func newStoreHistory(st store.Store, clock store.Clock, warn io.Writer) *storeHistory {
	h := &storeHistory{st: st, clock: clock, warn: warn}
	events, err := st.Events(time.Time{})
	if err != nil {
		h.warnf("could not read history: %v", err)
		return h
	}
	for _, e := range events {
		if e.Kind == store.EventLookedUp {
			h.lines = append(h.lines, e.Word)
		}
	}
	return h
}

func (h *storeHistory) Add(line string, found bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	now := h.clock.Now()

	h.mu.Lock()
	h.lines = append(h.lines, line)
	h.mu.Unlock()

	// The event records what happened, whether or not the word exists.
	if err := h.st.AppendEvent(store.ReviewEvent{
		Word: line, Kind: store.EventLookedUp, Found: found, At: now,
	}); err != nil {
		h.warnf("could not save history: %v", err)
	}
	if !found {
		return // a typo is history, not vocabulary
	}
	if err := h.st.Upsert(store.Word{Text: line, FirstSeen: now, LastSeen: now, Lookups: 1}); err != nil {
		h.warnf("could not save word: %v", err)
	}
}

func (h *storeHistory) Prefix(p string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []string
	seen := map[string]bool{}
	for i := len(h.lines) - 1; i >= 0; i-- { // newest first
		l := h.lines[i]
		if seen[l] || !strings.HasPrefix(l, p) {
			continue
		}
		seen[l] = true
		out = append(out, l)
	}
	return out
}

// warnf reports at most once. Losing durability is not a reason to interrupt
// someone mid-word on every keystroke.
func (h *storeHistory) warnf(format string, args ...any) {
	if h.warn == nil || h.warned {
		return
	}
	h.warned = true
	fmt.Fprintf(h.warn, "define: "+format+" (history is session-only)\n", args...)
}

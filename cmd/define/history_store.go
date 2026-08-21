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
// is loaded once at construction and held in memory. Since #4 this type only
// READS the store — at construction — and storeCapturer owns every write.
type storeHistory struct {
	mu    sync.Mutex
	lines []string // oldest first, every submitted line
	warn  io.Writer
}

func newStoreHistory(st store.Store, _ store.Clock, warn io.Writer) *storeHistory {
	h := &storeHistory{warn: warn}
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

// Add records the line for RECALL only.
//
// It used to write to the store as well. Those writes moved to storeCapturer
// (#4) so the process has exactly one writer — otherwise the raw path would
// record every lookup twice, once here and once at the capture site, and a deck
// that counts double is wrong in a way nobody notices until #5 orders by it.
func (h *storeHistory) Add(line string, _ bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	h.mu.Lock()
	h.lines = append(h.lines, line)
	h.mu.Unlock()
}

func (h *storeHistory) Prefix(p string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return prefixMatch(h.lines, p)
}

// warnf reports a store that could not be READ at startup — a different message,
// with a different home, from storeCapturer's failed-write warning. Conflating
// the two is what stranded the warn-once rule when the writes moved.
func (h *storeHistory) warnf(format string, args ...any) {
	if h.warn == nil {
		return
	}
	fmt.Fprintf(h.warn, "define: "+format+" (history is session-only)\n", args...)
}

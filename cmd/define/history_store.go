package main

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// storeHistory is the editor's History: it RECALLS, and never writes.
//
// Deliberately not restating the capture rules here. Every design fact in this
// package gets one normative home, because the same sentence in two places is
// the drift this file has already caused twice — it described itself as the
// writer for two review rounds after storeCapturer took that over. What writes,
// and when, is decided in capture.go and described in atlas/define.md.
//
// The one fact that IS local: Prefix runs on every keystroke and returns no
// error, so the log is read once at construction and everything after is memory.
type storeHistory struct {
	mu    sync.Mutex
	lines []string // oldest first, every submitted line
	warn  io.Writer
}

func newStoreHistory(st store.Store, warn io.Writer) *storeHistory {
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
// (#4) so lookups have exactly one recorder — otherwise the raw path would
// record every lookup twice, once here and once at the capture site, and a deck
// that counts double is wrong in a way nobody notices until #5 orders by it.
func (h *storeHistory) Add(line string) {
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

package main

import "strings"

// History is the editor's view of what has been typed before.
//
// An interface because the persistent implementation is #3's store: #15 requires
// /history to read that store rather than a private history file, so building
// one here would be building the thing #15 forbids.
//
// Every SUBMITTED line is recorded, with whether the lookup found anything.
// Up-arrow recall must include a word you typed and got wrong — that is exactly
// when you want to edit and retry — while #15's /history is about words
// *queried* and filters to found. One record, two readers.
type History interface {
	Add(line string, found bool)
	// Prefix returns entries beginning with p, newest first, deduped. An empty
	// p returns everything.
	Prefix(p string) []string
}

// memHistory is the session-scoped implementation. #3's store replaces it
// through this seam without the editor changing.
type memHistory struct {
	lines []string // oldest first
}

func (h *memHistory) Add(line string, _ bool) {
	if line = strings.TrimSpace(line); line != "" {
		h.lines = append(h.lines, line)
	}
}

func (h *memHistory) Prefix(p string) []string { return prefixMatch(h.lines, p) }

// prefixMatch is the ONE definition of History.Prefix's contract: newest first,
// deduped, prefix-filtered.
//
// Both implementations call it. They previously carried byte-identical copies,
// and the coverage was lopsided — the editor's tests all built memHistory, so
// fifteen recall and dedup assertions exercised the fallback while the durable
// one had a single test. A refinement to either would silently not apply to the
// other.
func prefixMatch(lines []string, p string) []string {
	var out []string
	seen := map[string]bool{}
	for i := len(lines) - 1; i >= 0; i-- { // newest first
		l := lines[i]
		if seen[l] || !strings.HasPrefix(l, p) {
			continue
		}
		seen[l] = true
		out = append(out, l)
	}
	return out
}

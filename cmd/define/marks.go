package main

import "slices"

// passageCell is a point in the passage's own coordinate space: a line and a
// display column within it. No wrap correction: the passage wraps at
// construction, so one of its lines IS one buffer line.
//
// Distinct from selectionPoint, which is a physical terminal cell. The conversion
// between them belongs to the screen, and keeping the types apart is what stops a
// frame row being used as a passage line by accident.
type passageCell struct {
	line, col int
}

// markSet is what is currently marked, ordered by position.
//
// ONE definition of "what is marked", read by the renderer, the prompt builder
// and the deck admission. Three readers is exactly the count at which a second
// definition starts to drift.
//
// Ordered by POSITION rather than insertion because the prompt brackets marks in
// place: a reader marking right-to-left must produce the same request as one
// marking left-to-right, or the same two words asked about twice are two
// different questions.
type markSet struct {
	spans []passageSpan
}

// toggle adds a span, or removes it if it is already marked.
//
// A toggle rather than an append, because clicking a marked word unmarks it
// (#67) — and because a click-mark and a drag-mark are the same kind of thing, so
// either gesture must be able to cancel the other.
func (m markSet) toggle(s passageSpan) markSet {
	// OVERLAP counts as already marked, not just an exact match.
	//
	// A drag produces one span over several words; clicking a word inside it used
	// to add a SECOND, overlapping span. Nothing showed it — paintMarks draws the
	// phrase's cells and markedPassageText skips a mark that starts before the
	// last one ended — but admitMarkedWords walks the set, so the word was looked
	// up and silently entered the deck. An invisible mark with a durable effect.
	//
	// Removing every overlap is also the reading a reader expects: clicking
	// inside a marked phrase clears that phrase, exactly as clicking a marked word
	// clears the word.
	var kept []passageSpan
	overlapped := false
	for _, x := range m.spans {
		if spansOverlap(x, s) {
			overlapped = true
			continue
		}
		kept = append(kept, x)
	}
	if overlapped {
		return markSet{spans: kept}
	}
	out := append(slices.Clone(m.spans), s)
	slices.SortFunc(out, compareSpans)
	return markSet{spans: out}
}

// spansOverlap reports whether two spans of the same line share any byte.
func spansOverlap(a, b passageSpan) bool {
	return a.line == b.line && a.start < b.end && b.start < a.end
}

// ordered is the marks in reading order.
func (m markSet) ordered() []passageSpan { return slices.Clone(m.spans) }

func (m markSet) empty() bool { return len(m.spans) == 0 }

// clear drops every mark. The whole set goes at once, after an ask: the
// transient state converts into deck membership (#67), and a bare Enter
// afterwards must find nothing to re-ask.
func (m markSet) clear() markSet { return markSet{} }

func compareSpans(a, b passageSpan) int {
	if a.line != b.line {
		return a.line - b.line
	}
	return a.start - b.start
}

// marksForDrag turns a drag into marks — one per LINE, because the prompt
// brackets a mark in place and a bracket cannot span a line break.
//
// Both ends snap OUTWARD to whole words: the unit everywhere else in this feature
// is a word, so a drag that starts mid-word marks that whole word rather than
// half of it. A drag that covers no word at all marks nothing, because an empty
// bracket would be a question about nothing.
func marksForDrag(p *passage, from, to passageCell) []passageSpan {
	if p == nil {
		return nil
	}
	// A backwards drag is the same selection: the reader dragged right-to-left,
	// which says nothing about what they meant.
	if to.line < from.line || (to.line == from.line && to.col < from.col) {
		from, to = to, from
	}
	var out []passageSpan
	for line := from.line; line <= to.line && line < p.lineCount(); line++ {
		if line < 0 {
			continue
		}
		lo, hi := 0, len(p.line(line))
		if line == from.line {
			if s, ok := firstWordFrom(p, line, from.col); ok {
				lo = s.start
			} else {
				continue
			}
		}
		if line == to.line {
			if s, ok := lastWordTo(p, line, to.col); ok {
				hi = s.end
			} else if line == from.line {
				continue
			}
		}
		if lo < hi {
			out = append(out, passageSpan{line: line, start: lo, end: hi})
		}
	}
	return out
}

// firstWordFrom is the first word at or after a column — the outward snap for a
// drag's left edge.
func firstWordFrom(p *passage, line, col int) (passageSpan, bool) {
	if s, ok := wordAtCell(p, line, col); ok {
		return s, true
	}
	at, _ := byteAtCell(p.line(line), col)
	for _, s := range p.spans(line) {
		if s.start >= at {
			return s, true
		}
	}
	return passageSpan{}, false
}

// lastWordTo is the last word at or before a column — the outward snap for a
// drag's right edge.
func lastWordTo(p *passage, line, col int) (passageSpan, bool) {
	if s, ok := wordAtCell(p, line, col); ok {
		return s, true
	}
	at, ok := byteAtCell(p.line(line), col)
	if !ok {
		at = len(p.line(line))
	}
	var best passageSpan
	var found bool
	for _, s := range p.spans(line) {
		if s.end <= at {
			best, found = s, true
		}
	}
	return best, found
}

package main

// session is what one REPL session is holding.
//
// It replaced three separate declarations of the same idea — replLines' own
// `current`, runEditor's, and the `current *string` threaded through submitLine
// — because #16 added a rule that would otherwise have had to be written three
// times and believed once: an ask touches none of this.
type session struct {
	// current is the word a bare Enter replays. Only a SUCCESSFUL lookup sets
	// it: not a typo, and not a question. A question that became the current
	// word would be "replayed" by the next bare Enter, and would be the word
	// #16's ask context claims the next question is about.
	current string
	// entry is the raw dictionary text for current, kept so a question that
	// follows a lookup can carry it as context without a second dictionary
	// call (#16 D1).
	entry string
	// words is every word looked up in THIS session, oldest first. Distinct from
	// the deck, which survives the process: "the last few words of this session"
	// is a different signal from "what this learner has been studying", and the
	// prompt carries both.
	words []string
	// passage is the text being read, written to the buffer (#67). Session-scoped
	// like the rest of this struct: it is transient reading material, and what
	// deserves to persist is the residue — a marked word on the deck — not the
	// passage. A second paste replaces it.
	passage *passage
	// marks is what is currently marked in the passage. Cleared after an ask: the
	// transient state converts into deck membership (#67).
	marks markSet
	// passageBase is the buffer line the passage was written at, which is what
	// ties a mark's passage coordinates to the screen's.
	passageBase int
	// turns is the Q&A transcript, so a follow-up ("give me three more
	// examples") resolves against the answer it follows. Session-scoped by
	// design: a fresh process answers just as well, with the directory rather
	// than a chat history as its context.
	turns []exchange
}

// hasCurrent is what parseREPLLine needs to know to read a blank line.
func (s *session) hasCurrent() bool { return s.current != "" }

// lineState is everything the session holds that changes what a line MEANS,
// handed over as one value so no caller has to remember the precedence.
// lineState takes whether the live passage is ON SCREEN, because "a passage was
// once pasted" is not authority over Enter for the rest of the session.
//
// It was `s.passage != nil` when the passage was pinned footer chrome, where it
// could not scroll away. The footer-to-buffer reversal changed the surface's
// LIFETIME and this predicate was not re-derived: a bare Enter never returned to
// replaying the current word again, and the nudge pointed at something long
// off-screen (BR-28).
func (s *session) lineState(visible bool) lineState {
	return lineState{
		hasCurrent: s.hasCurrent(),
		hasPassage: visible,
		hasMarks:   !s.marks.empty(),
	}
}

// sawLookup records a successful lookup. The ask outcome deliberately does not
// reach here — see lookupOutcome.
func (s *session) sawLookup(word string, out lookupOutcome) {
	s.current, s.entry = word, out.entry
	s.words = append(s.words, word)
}

// recordExchange remembers one question and its answer for the next follow-up.
// An empty answer is not recorded: a question that produced nothing gives a
// follow-up nothing to resolve against, and "Q: … A: " in the prompt reads as a
// refusal the model then imitates.
func (s *session) recordExchange(q, answer string) {
	if answer == "" {
		return
	}
	s.turns = append(s.turns, exchange{Question: q, Answer: answer})
}

// ownsBufferLine reports whether an absolute buffer line belongs to the passage
// currently on screen.
//
// The identity check a Region cannot carry: regions are addressed relative to
// the render they came from, and a superseded passage's stay in the screen's map
// forever, so "is this region mine" can only be answered by line number.
func (s *session) ownsBufferLine(line int) bool {
	return s.passage != nil && line >= s.passageBase && line < s.passageBase+s.passage.lineCount()
}

// passageCell converts an absolute buffer point into the passage's own
// coordinates, REFUSING a row the live passage does not own.
//
// Refusing rather than clamping. Clamping was the first version and it is what
// let a drag in a SUPERSEDED passage land inside the current one: every row
// outside got pulled to the nearest line that existed. A point that is not in
// this passage is not a point in this passage (BR-28).
func (s *session) passageCell(p selectionPoint) (passageCell, bool) {
	if !s.ownsBufferLine(p.row) {
		return passageCell{}, false
	}
	return passageCell{line: p.row - s.passageBase, col: p.col}, true
}

// passageSpanAt resolves a clicked region to a span of the CURRENT passage, or
// refuses when the region belongs to a superseded one.
func (s *session) passageSpanAt(line int, r Region) (passageSpan, bool) {
	if !s.ownsBufferLine(line) {
		return passageSpan{}, false
	}
	return wordAtCell(s.passage, line-s.passageBase, r.Col)
}

// passageVisible reports whether any line of the live passage is in the viewport.
func (s *session) passageVisible(top, bottom int) bool {
	if s.passage == nil || s.passage.empty() {
		return false
	}
	return s.passageBase < bottom && s.passageBase+s.passage.lineCount() > top
}

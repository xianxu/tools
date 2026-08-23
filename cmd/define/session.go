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
}

// hasCurrent is what parseREPLLine needs to know to read a blank line.
func (s *session) hasCurrent() bool { return s.current != "" }

// sawLookup records a successful lookup. The ask outcome deliberately does not
// reach here — see lookupOutcome.
func (s *session) sawLookup(word string, out lookupOutcome) {
	s.current, s.entry = word, out.entry
}

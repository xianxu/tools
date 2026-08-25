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
	// turns is the Q&A transcript, so a follow-up ("give me three more
	// examples") resolves against the answer it follows. Session-scoped by
	// design: a fresh process answers just as well, with the directory rather
	// than a chat history as its context.
	turns []exchange
}

// hasCurrent is what parseREPLLine needs to know to read a blank line.
func (s *session) hasCurrent() bool { return s.current != "" }

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

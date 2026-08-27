package play

// InputKind separates the two things a keystroke can be: a CONTROL intent the
// session understands, or a graded key only the form understands.
//
// This is where main.Key stops. main owns the terminal and knows that Ctrl-C is
// 0x03 and that Enter may arrive as \r or \n; play must not, or it could not be
// tested without one. The loop decodes once, at the boundary where it already
// decodes escape sequences, and hands over an intent.
type InputKind int

const (
	// InputRune is a key the FORM grades. The session never learns what it means.
	InputRune InputKind = iota
	// InputReveal asks to see the answer.
	InputReveal
	// InputQuit ends the session now, keeping everything already recorded.
	InputQuit
	// InputDrop removes the current word from the deck and moves on.
	//
	// A SESSION-level action rather than a verdict, which is why it is an Input
	// kind and not something a form grades: "this word does not belong in my
	// deck" is true whatever form is asking about it, so every future form gets
	// it for free. It is also not an assessment — dropping records no review.
	InputDrop
)

// Input is one decoded keystroke.
type Input struct {
	Kind InputKind
	Rune rune // set when Kind is InputRune
}

// OutcomeKind is what the LOOP must do next. The session performs no effects
// itself — the same split #14's editor uses, so the two interactive surfaces in
// this binary work the same way.
type OutcomeKind int

const (
	// OutcomeNone: redraw and wait. Also what a skip produces, which is how the
	// skip rule is enforced in ONE place.
	OutcomeNone OutcomeKind = iota
	// OutcomeRecord: write this verdict to the event log, NOW, before drawing the
	// next question. Recording as it happens is what makes Ctrl-C lossless by
	// construction rather than by a flush.
	OutcomeRecord
	// OutcomeReveal: the answer is now showing; play the pronunciation.
	OutcomeReveal
	// OutcomeDone: the session is over.
	OutcomeDone
	// OutcomeDrop: remove this word from the deck. The EVENTS stay, matching
	// --forget's contract: history is what happened and cannot be untrue, while
	// the deck is the working set and is the learner's to curate.
	OutcomeDrop
)

// Outcome is what the loop must do, and about which word.
//
// Word and Verdict are set only for OutcomeRecord. The loop calls CaptureReview
// whenever it sees one and NEVER inspects the verdict — because a skip never
// produces this outcome, so there is nothing to filter downstream. One rule, one
// place.
type Outcome struct {
	Kind    OutcomeKind
	Word    string
	Verdict Verdict
}

// Session is where the learner is in today's queue.
type Session struct {
	Questions []Question
	Index     int
	Revealed  bool
	Right     int
	Wrong     int
	Done      bool
}

// NewSession starts at the first question. An empty queue is immediately Done,
// which is the honest state for "nothing due today" — the expected state most
// days, and not an error.
func NewSession(qs []Question) Session {
	return Session{Questions: qs, Done: len(qs) == 0}
}

// Current is the question on screen, or nil when the session is over.
func (s Session) Current() Question {
	if s.Done || s.Index >= len(s.Questions) {
		return nil
	}
	return s.Questions[s.Index]
}

// Apply is the state machine: one input, the next state and what the loop owes.
func Apply(s Session, in Input) (Session, Outcome) {
	if s.Done {
		return s, Outcome{Kind: OutcomeDone}
	}
	q := s.Current()
	if q == nil {
		s.Done = true
		return s, Outcome{Kind: OutcomeDone}
	}

	switch in.Kind {
	case InputDrop:
		// Dropping is allowed before OR after reveal: you may recognise a word as
		// not-yours without needing to see the definition again.
		word := q.Word()
		next, _ := advance(s, q, Skipped) // advances, records nothing
		return next, Outcome{Kind: OutcomeDrop, Word: word}

	case InputQuit:
		// Everything already recorded stays recorded — that is a property of
		// recording as it happens, not of anything done here.
		s.Done = true
		return s, Outcome{Kind: OutcomeDone}

	case InputReveal:
		if s.Revealed {
			return s, Outcome{Kind: OutcomeNone}
		}
		s.Revealed = true
		return s, Outcome{Kind: OutcomeReveal}

	case InputRune:
		if !s.Revealed {
			// A learner cannot rate what they have not seen. Grading before
			// reveal is a mis-keystroke, and treating it as an answer would
			// record a verdict about a word still hidden.
			return s, Outcome{Kind: OutcomeNone}
		}
		verdict, ok := q.Grade(in.Rune)
		if !ok {
			return s, Outcome{Kind: OutcomeNone} // a key this form does not use
		}
		return advance(s, q, verdict)
	}
	return s, Outcome{Kind: OutcomeNone}
}

// advance moves to the next question and says what to record.
//
// THE SKIP FILTER LIVES HERE, and only here. Only the session knows a verdict;
// the loop just performs outcomes. A loop that inspected verdicts would be
// re-deciding what this already decided, which is how one rule ends up
// implemented in two places or neither.
func advance(s Session, q Question, v Verdict) (Session, Outcome) {
	switch v {
	case Correct:
		s.Right++
	case Wrong:
		s.Wrong++
	}
	s.Index++
	s.Revealed = false
	if s.Index >= len(s.Questions) {
		s.Done = true
	}

	if v == Skipped {
		// Not an assessment. schedule.Fold would read a recorded skip as a miss
		// and demote the word.
		return s, Outcome{Kind: OutcomeNone}
	}
	return s, Outcome{Kind: OutcomeRecord, Word: q.Word(), Verdict: v}
}

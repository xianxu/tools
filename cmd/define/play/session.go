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
// Word is set on every outcome that is ABOUT a word — Record, Reveal and Drop.
// Verdict is set only on Record; the loop calls CaptureReview whenever it sees
// one and NEVER inspects the verdict, because a skip never produces this outcome,
// so there is nothing to filter downstream. One rule, one place.
//
// Reveal carries its word for the same reason the other two do: the loop needs a
// word to play the pronunciation for, and reading it back off the session
// (`s.Current().Word()`) is only correct while no input both advances AND
// reveals. That held, but it held by accident — the first input that did both
// would not have played the wrong word, it would have NIL-PANICKED at the end of
// the queue, where Current() returns nil (BR-4). An outcome that names its own
// subject cannot go stale that way, and #7's multiple-choice form is about to
// add inputs to this machine.
//
// SessionDone is set on the outcome that ENDED the session, whatever its kind.
// The last answer produces OutcomeRecord and finishes the queue, so a caller
// reading Kind alone cannot tell that was the end — it would have to consult the
// returned Session, which makes "did we finish" two facts in two places. The
// loop here does look at the Session, and that is fine; the flag is for callers
// that receive only an Outcome, which is every consumer this interface is meant
// to allow.
type Outcome struct {
	Kind        OutcomeKind
	Word        string
	Verdict     Verdict
	SessionDone bool
}

// Session is where the learner is in today's queue.
type Session struct {
	Questions []Question
	Index     int
	Revealed  bool
	// Graded means this question's verdict is already recorded and its answer is
	// on screen: the next keystroke moves on rather than grading again.
	//
	// Independent of Revealed — a learner can reveal without grading (space) and
	// now grade without revealing (y).
	Graded bool
	Right  int
	Wrong  int
	Done   bool
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

// Apply is the state machine: one input, the next state, and EVERY effect the
// loop owes.
//
// A SLICE because one input can owe more than one. `n` on a word the learner has
// not seen records the miss AND reveals the definition, which is two things the
// loop must do. Encoding the second as a flag on the first would make "reveal"
// expressible two ways, and this machine has already paid once for a rule living
// in two places (#6 BR-5).
//
// Order is significant: the record is emitted FIRST, so a caller performing them
// in order writes the event before anything that can block on the terminal.
func Apply(s Session, in Input) (Session, []Outcome) {
	if s.Done {
		return s, []Outcome{{Kind: OutcomeDone, SessionDone: true}}
	}
	q := s.Current()
	if q == nil {
		s.Done = true
		return s, []Outcome{{Kind: OutcomeDone, SessionDone: true}}
	}

	// "Any key = next word" — ONE rule, one home.
	//
	// Both kinds arrive here meaning the same thing once the answer is up and
	// recorded: InputRune for any letter, InputReveal for Enter and space, which
	// toInput maps to the same kind. A second assessment is not on offer — Fold
	// would read a duplicate as another review — so the input is spent moving on.
	//
	// Hoisted out of the two arms (BR-12): they were four identical lines with
	// different comments, implementing the single rule the prompt states as
	// "any key = next word". This machine's own doc comment cites #6 BR-5 against
	// a rule living in two places, and then grew one.
	//
	// InputDrop and InputQuit stay OUTSIDE deliberately: "this word is not mine"
	// and "stop" are still true after a verdict, and routing them here would
	// silently turn a drop into a plain advance.
	if s.Graded && (in.Kind == InputRune || in.Kind == InputReveal) {
		next, out := advance(s, q, Skipped)
		return next, []Outcome{out}
	}

	switch in.Kind {
	case InputDrop:
		// Dropping is allowed in every state — before a reveal, after a peek, and
		// after a miss. "This word is not mine" is true whatever is on screen,
		// and someone who just missed a word is exactly who wants to drop it.
		word := q.Word()
		next, _ := advance(s, q, Skipped) // advances, records nothing
		return next, []Outcome{{Kind: OutcomeDrop, Word: word, SessionDone: next.Done}}

	case InputQuit:
		// Everything already recorded stays recorded — that is a property of
		// recording as it happens, not of anything done here.
		s.Done = true
		return s, []Outcome{{Kind: OutcomeDone, SessionDone: true}}

	case InputReveal:
		// The graded case is handled above: Enter and space both land on this
		// kind, and once graded they mean "next".
		if s.Revealed {
			return s, []Outcome{{Kind: OutcomeNone}}
		}
		s.Revealed = true
		return s, []Outcome{{Kind: OutcomeReveal, Word: q.Word()}}

	case InputRune:
		// The graded case is handled above.
		verdict, ok := q.Grade(in.Rune)
		if !ok {
			return s, []Outcome{{Kind: OutcomeNone}} // a key this form does not use
		}
		if s.Revealed || verdict != Wrong {
			// Nothing left to show: the answer is already on screen, or the
			// learner had it and does not need it.
			//
			// GRADING BEFORE A REVEAL IS THE NORMAL PATH, and it used to be
			// refused here on the grounds that "a learner cannot rate what they
			// have not seen". That is true of a recognition test and false of a
			// RECALL test, which is what form 2.1 is: the learner rates their own
			// recall, which they know before they check, and the definition is
			// FEEDBACK rather than stimulus. Getting it backwards put a mandatory
			// keystroke in front of every correct answer (#24).
			next, out := advance(s, q, verdict)
			return next, []Outcome{out}
		}
		// A MISS on a hidden word earns the definition, and earns it WITHOUT
		// advancing — moving on would scroll the answer past unread, which is the
		// entire reason for showing it.
		//
		// Scored here rather than by advance, because we are not advancing. The
		// first draft of this branch recorded the event and left the tally alone,
		// so a session of three misses ended "0 right, 0 wrong" (PQ-1).
		s.Revealed, s.Graded = true, true
		s = score(s, verdict)
		return s, []Outcome{
			{Kind: OutcomeRecord, Word: q.Word(), Verdict: verdict},
			{Kind: OutcomeReveal, Word: q.Word()},
		}
	}
	return s, []Outcome{{Kind: OutcomeNone}}
}

// score is what a verdict does to the tally, and the only place that decides it.
//
// Split from advance because a miss on a hidden word must score without moving
// on. Skipped scores nothing, so advancing off an already-scored miss with
// Skipped cannot double-count.
func score(s Session, v Verdict) Session {
	switch v {
	case Correct:
		s.Right++
	case Wrong:
		s.Wrong++
	}
	return s
}

// advance moves to the next question and says what to record.
//
// THE SKIP FILTER LIVES HERE, and only here. Only the session knows a verdict;
// the loop just performs outcomes. A loop that inspected verdicts would be
// re-deciding what this already decided, which is how one rule ends up
// implemented in two places or neither.
func advance(s Session, q Question, v Verdict) (Session, Outcome) {
	s = score(s, v)
	s.Index++
	s.Revealed, s.Graded = false, false
	if s.Index >= len(s.Questions) {
		s.Done = true
	}

	if v == Skipped {
		// Not an assessment. schedule.Fold would read a recorded skip as a miss
		// and demote the word.
		return s, Outcome{Kind: OutcomeNone, SessionDone: s.Done}
	}
	return s, Outcome{Kind: OutcomeRecord, Word: q.Word(), Verdict: v, SessionDone: s.Done}
}

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
	// InputReveal asks to see the answer. SPACE, and Enter for every form that
	// holds one word — see InputFinish.
	InputReveal
	// InputFinish is ENTER, split from space because a form holding many words
	// spends itself on it.
	//
	// The two were one kind, deliberately: "Enter and space both land on this
	// kind". That held while every form had one word and one answer. A board
	// takes every unmarked word as Wrong when it is finished, and leaving the
	// pair merged would fire that on SPACE — the one expensive-to-undo action on
	// that surface, on the most careless key there is.
	//
	// For every form holding one word the two remain EQUIVALENT, which is what
	// keeps 2.1 and 2.3 from noticing the split.
	InputFinish
	// InputToggle switches the MODE of a form that has one. Tab.
	//
	// It belongs in play rather than being intercepted by the loop, and the line
	// between the two is what the keystroke is ABOUT: the paging keys change what
	// you are looking at, which is the terminal's business, while this changes
	// what your next answer will mean, which is the session's. An earlier draft
	// routed the paging keys through here and was reversed for exactly that
	// reason (#41 D6); this one goes the other way for the same test.
	InputToggle
	// InputMark answers ONE CELL of a form drawn as a grid. A click, and the
	// only input that carries a coordinate.
	//
	// #38 shipped "a click ACTS and never answers", pinned by a row that is
	// still green: a click on a headword plays the word and stops before
	// toInput, so Apply never sees it. A board reverses that for itself, and the
	// invariant survives restated honestly — A CLICK NEVER ANSWERS A FORM THAT
	// DID NOT ASK FOR IT. Every form that is not a Grid declines this kind, which
	// is what makes the seam a widening rather than a branch.
	//
	// It is not a forged keystroke. The loop could have looked up the cell's
	// printed label and sent InputRune, which would need no new kind at all —
	// and it would mean this machine could no longer tell a key from a pointer,
	// on the one surface where the difference is the whole design.
	InputMark
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
	Cell int  // set when Kind is InputMark: which cell of a grid form
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
	Kind    OutcomeKind
	Word    string
	Verdict Verdict
	// Axis is WHY the option they picked was in the set, on a Record outcome
	// for a form that can say (see Missed). AxisNone otherwise — including on
	// every correct answer, which is D8: a right answer writes no finding.
	//
	// On Outcome rather than returned from Grade because only SOME forms have
	// it. Widening Grade would make form 2.1 answer a question it cannot.
	Axis Axis
	// Unaided marks a right answer given COLD — the form checked it against an
	// answer it already knew, and no reveal preceded it. False on every wrong
	// answer, on every self-rated form, and after any reveal.
	//
	// It is the objective half of the ladder's two-rung promotion: an
	// observation the session makes rather than a confidence the learner
	// asserts.
	Unaided     bool
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
	if s.Graded && (in.Kind == InputRune || in.Kind == InputReveal || in.Kind == InputFinish) {
		next, out := advance(s, q, Skipped, false)
		return next, []Outcome{out}
	}

	switch in.Kind {
	case InputDrop:
		// REFUSED by a form holding many words, because `d` names no word there:
		// advance would drop q.Word(), which on a grid is whichever cell happens
		// to be next. Dropping the wrong word is silent and takes a word out of
		// the deck, so the form does nothing rather than guessing.
		if batchOf(q) != nil {
			return s, []Outcome{{Kind: OutcomeNone}}
		}
		// Dropping is allowed in every state — before a reveal, after a peek, and
		// after a miss. "This word is not mine" is true whatever is on screen,
		// and someone who just missed a word is exactly who wants to drop it.
		word := q.Word()
		next, _ := advance(s, q, Skipped, false) // advances, records nothing
		return next, []Outcome{{Kind: OutcomeDrop, Word: word, SessionDone: next.Done}}

	case InputQuit:
		// Everything already recorded stays recorded — that is a property of
		// recording as it happens, not of anything done here.
		s.Done = true
		return s, []Outcome{{Kind: OutcomeDone, SessionDone: true}}

	case InputMark:
		// The cell was resolved by the FORM before this — the loop asked
		// Grid.CellAt where the click landed, because the form is the only thing
		// that knows where it drew its words. This lands the mark and records it
		// exactly as the printed key would: one act, two ways in.
		g, ok := q.(Grid)
		if !ok {
			return s, []Outcome{{Kind: OutcomeNone}}
		}
		v, ok := g.Mark(in.Cell)
		if !ok {
			// A cell that is already answered, or no cell at all. Nothing
			// happens and nothing is said, which is what a click on ordinary
			// text has always done here.
			//
			// LOAD-BEARING FOR A GRID THAT IS NOT A BATCH, which is the only
			// reason it is not `advance(Skipped)` — that would be the same thing
			// for a board, because an unspent form does not advance. Grid and
			// Batch are separate capabilities, and a one-word grid form is spent
			// by definition: without this, a click on nothing would move it on.
			// TestARefusedClickDoesNotAdvanceAGridThatIsNotABatch is the pin.
			return s, []Outcome{{Kind: OutcomeNone}}
		}
		next, out := advance(s, q, v, unaidedNow(s, q, v))
		return next, []Outcome{out}

	case InputToggle:
		// Tab, and it reaches only a form that HAS a mode.
		//
		// A NO-OP everywhere else, rather than a member of the "any key = next
		// word" rule above. A board is never Graded, so the only forms Tab could
		// advance there are 2.1 and 2.3 — where it would be an accident-prone
		// extra way to scroll a definition away mid-read. Doing nothing is the
		// safer failure of the two, and the only one that cannot lose something
		// the learner was still reading.
		if m, ok := q.(Moded); ok {
			m.Toggle()
		}
		return s, []Outcome{{Kind: OutcomeNone}}

	case InputFinish:
		// A form holding many words SPENDS itself: every word still unmarked is
		// answered Wrong, one record each, and the session moves on. That is
		// "I am out of time, ask me all of these again", and it is the reason
		// Enter needed its own kind.
		if b := batchOf(q); b != nil {
			rest := b.Rest(Wrong)
			outs := make([]Outcome, 0, len(rest)+1)
			for _, w := range rest {
				s = score(s, Wrong)
				outs = append(outs, Outcome{Kind: OutcomeRecord, Word: w, Verdict: Wrong})
			}
			next, out := advance(s, q, Skipped, false)
			outs = append(outs, out)
			for i := range outs {
				outs[i].SessionDone = next.Done
			}
			return next, outs
		}
		// Every other form: Enter means what space means.
		fallthrough

	case InputReveal:
		// The graded case is handled above: once graded they mean "next".
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
			// COMPUTED HERE, while s.Revealed still holds its real value.
			// advance() zeroes it before it builds the outcome, so reading it
			// there would mark EVERY correct answer unaided — the feature would
			// look like it worked while running the ladder at double speed.
			// TestARevealDisqualifiesUnaided is the pin.
			next, out := advance(s, q, verdict, unaidedNow(s, q, verdict))
			return next, []Outcome{out}
		}
		// A form holding many words has no hidden word to reveal — its marks are
		// self-report over a grid — so a Wrong mark is an ordinary graded answer.
		// Falling through to the branch below would set Graded, and the next
		// keystroke would then mean "any key = next word": the form would freeze
		// after its first No.
		if batchOf(q) != nil {
			next, out := advance(s, q, verdict, unaidedNow(s, q, verdict))
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
			{Kind: OutcomeRecord, Word: q.Word(), Verdict: verdict, Axis: missedAxis(q)},
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
func advance(s Session, q Question, v Verdict, unaided bool) (Session, Outcome) {
	s = score(s, v)
	// A form holding many words keeps the slot until every one is answered
	// (Batch). Everything else is spent by definition.
	if spent(q) {
		s.Index++
		s.Revealed, s.Graded = false, false
		if s.Index >= len(s.Questions) {
			s.Done = true
		}
	}

	if v == Skipped {
		// Not an assessment. schedule.Fold would read a recorded skip as a miss
		// and demote the word.
		return s, Outcome{Kind: OutcomeNone, SessionDone: s.Done}
	}
	return s, Outcome{Kind: OutcomeRecord, Word: q.Word(), Verdict: v, Axis: missedAxis(q), Unaided: unaided, SessionDone: s.Done}
}

// Missed is implemented by forms whose WRONG answers carry a kind.
//
// Optional deliberately. Form 2.1 cannot say why a recall failed — the learner
// simply did not remember — so requiring every form to answer would make the
// interface lie for the one form that has no answer. Apply asks and takes
// AxisNone when nobody answers, which is also what a correct answer reports.
//
// This is what keeps the session form-AGNOSTIC while still carrying #17 M2's
// finding out: Apply names a CAPABILITY here, never a form. A type switch on
// *Choice would be the thing Done-when 7 forbids.
type Missed interface {
	MissedAxis() Axis
}

// missedAxis asks a question for the axis it was missed on, if it can answer.
func missedAxis(q Question) Axis {
	if m, ok := q.(Missed); ok {
		return m.MissedAxis()
	}
	return AxisNone
}

// Batch is implemented by forms that hold MORE THAN ONE word.
//
// Third of its kind beside Missed and SelfRated, and for the same reason both of
// those exist: the session must not learn which form is asking (#6's Done-when),
// so it names a CAPABILITY and asks. A type switch on *Board here would be the
// thing that Done-when forbids.
//
// It is consulted at FOUR points rather than one, which is the honest cost of a
// form holding many words. `advance` is the obvious one; the other three were
// found by measurement rather than by reading:
//
//	the miss branch — a Wrong mark reaches "a MISS on a hidden word earns the
//	                  definition", which sets Graded, and the next key would then
//	                  mean "next word". The form would freeze after one mark.
//	InputDrop      — advance(Skipped) drops q.Word(), and a form holding many has
//	                 no single current word to name. Refused rather than guessed.
//	InputFinish    — Enter spends the form; every other form has nothing to spend.
type Batch interface {
	// Spent reports whether every word this form holds has been answered.
	Spent() bool
	// Rest answers every word still unmarked with v and returns them, which is
	// what Enter means to a form holding many.
	Rest(v Verdict) []string
}

// spent asks a form whether it is finished, and takes YES from anything that
// cannot answer — which is every form holding one word, and the right default.
func spent(q Question) bool {
	b, ok := q.(Batch)
	return !ok || b.Spent()
}

// batchOf is the form as a Batch, or nil. Named so the four call sites read as
// one question rather than four type assertions.
func batchOf(q Question) Batch {
	b, _ := q.(Batch)
	return b
}

// Grid is implemented by forms drawn as a GRID on the live edge, whose cells are
// answered by clicking them.
//
// Fifth of its kind, and the first one the LOOP asks rather than Apply — which
// is why all three methods are on one interface instead of split by consumer.
// They are one capability: a form that decides where its own words are printed
// is the only thing that can say which one was clicked, and a form that can say
// that is the only thing whose clicks mean anything. Splitting the geometry from
// the answering would put the two halves of that sentence in different places.
type Grid interface {
	// Rows is how many lines Prompt() produces, so the caller can budget the
	// live edge and know how far the grid extends.
	Rows() int
	// CellAt is which cell is at a position INSIDE the grid: row 0 is the grid's
	// first line, column 0 its first column. The caller subtracts wherever it
	// drew the block, which is the only part of this it is qualified to know.
	CellAt(row, col int) (int, bool)
	// Mark lands the form's active mode on cell i, and reports what that meant.
	// False for a cell that is not there or is already answered.
	Mark(i int) (Verdict, bool)
}

// Moded is implemented by forms that hold a MODE: a setting the learner
// switches, which changes what their next answer MEANS rather than what it is
// about.
//
// Fourth of its kind beside Missed, SelfRated and Batch, and asked rather than
// switched on for the same reason all three exist. Separate from Batch
// deliberately, even though the board is today the only implementer of either:
// "I hold many words" and "my answer key has two meanings" are different facts,
// and folding them together would force a mode onto the next batch form that
// does not want one.
type Moded interface {
	// Toggle switches the mode. This is the whole of what Tab means, and the
	// whole of this interface: the form DRAWS its own toggle, so nothing outside
	// it ever needs to read the mode. An accessor here would be a second way to
	// learn a fact the form already puts on screen.
	Toggle()
}

// SelfRated is implemented by forms whose verdict is the learner's CLAIM rather
// than something the form checked.
//
// Form 2.1 is the whole population today: `y` means "I knew it" and nobody
// verified it. `#40`'s board joins it — a `firm` mark is triage, not retrieval.
//
// It is `SelfRated` and not `Observed` deliberately, even though the DEFAULT is
// then the permissive one. A marker that a new form must remember to add would
// silently deny every future form the promotion it has earned; a marker it must
// remember to add to CLAIM strictness fails the other way, loudly, the first
// time someone checks. TestSelfRatedFormsNeverEarnUnaided is that check, and it
// ranges over the forms rather than naming a rule.
type SelfRated interface {
	IsSelfRated() bool
}

// unaidedNow is the observation, made while the session still holds the state
// that proves it.
func unaidedNow(s Session, q Question, v Verdict) bool {
	if v != Correct || s.Revealed {
		return false
	}
	sr, ok := q.(SelfRated)
	return !ok || !sr.IsSelfRated()
}

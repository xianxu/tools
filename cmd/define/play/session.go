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
	// numInputKinds is the sentinel a test derives the SET from, never a list.
	//
	// It exists because D12 wrote down four Apply paths that must consult Batch
	// and the true matrix is InputKind x Batch — `InputReveal` was the fifth
	// cell and nothing noticed, because the enumeration lived in prose. A form
	// holding many words has no single hidden word to reveal, so space set
	// `Revealed` and handed the loop an arbitrary cell's word to pronounce and a
	// blank reveal to file in the append-only buffer.
	//
	// TestEveryInputKindIsAnsweredForABatchForm ranges over this, so the NEXT
	// kind added cannot skip the question. Same move as choice.go's `numAxes`.
	numInputKinds
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
	// OutcomeFlag: the learner called this QUESTION broken. Record it, with the
	// options that made it broken, and move on.
	//
	// A KIND OF ITS OWN rather than a Record with a verdict, and the reason is
	// mechanical: CaptureReview writes EventReviewed, Fold folds every reviewed
	// event, and GradeOf(correct=false) is GradeWrong — so a flag routed through
	// Record would DEMOTE the word on an append-only log. The loop's arm for this
	// calls a different store verb, exactly as OutcomeDrop's calls Forget.
	OutcomeFlag
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
	// it. Widening Grade would have made form 2.1 give an answer it did not have.
	Axis Axis
	// Unaided marks a right answer given COLD — the form checked it against an
	// answer it already knew, and no reveal preceded it. False on every wrong
	// answer, on every self-rated form, and after any reveal.
	//
	// It is the objective half of the ladder's two-rung promotion: an
	// observation the session makes rather than a confidence the learner
	// asserts.
	Unaided bool
	// Form is which form asked (#40 D4a). Set on every Record outcome, and on
	// OutcomeFlag too, because a flag the log cannot attribute to a form is not
	// diagnosable — which is the whole reason a flag is recorded at all.
	//
	// It was once set in one place, and that sentence stood here after #12 added
	// the two sites that set it directly on a non-Record kind. The reason behind
	// it still holds and is worth stating as the rule rather than the count: a
	// call site building an Outcome without a Form ships a record nobody can
	// attribute, so every site that builds one sets it.
	Form string
	// Options is the option set of a FLAGGED question, and is set on no other
	// kind. See Flagging for why a flag carries what Missed deliberately does
	// not: the flag exists to diagnose this question, so the options are the
	// evidence.
	Options     []string
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
	q := s.Current()
	next, outs := apply(s, q, in)
	// EVERY RECORD NAMES THE FORM THAT ASKED, and it is stamped HERE rather than
	// at the three places that build one (#40 D4a).
	//
	// The alternative is `Form: q.Form()` written out at the advance, the
	// miss-on-a-hidden-word branch and the Enter-spends-a-board loop — three
	// chances to ship a promotion the log cannot attribute, and the failure would
	// be silent: an event with an empty form looks like data. What this field
	// exists to watch is already silent and delayed enough.
	for i := range outs {
		if outs[i].Kind == OutcomeRecord && q != nil {
			outs[i].Form = q.Form()
		}
	}
	return next, outs
}

// apply is the machine itself, with the current question already in hand.
func apply(s Session, q Question, in Input) (Session, []Outcome) {
	if s.Done {
		return s, []Outcome{{Kind: OutcomeDone, SessionDone: true}}
	}
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
	// A FLAG IS ASKED BEFORE the graded branch below, for the reason InputDrop
	// and InputQuit sit outside it: "this question is broken" is still true after
	// a verdict, and falling through would silently turn a flag into a plain
	// advance. That is the state it matters MOST in — a learner discovers a
	// question is broken by reading the reveal.
	//
	// The keystroke is offered to Flag DIRECTLY, never to Grade. Routing it
	// through Grade was the close review's I-2: a graded question handed any rune
	// to Grade to discover a flag, and a stray digit then re-picked the answer.
	if s.Graded && (in.Kind == InputRune || in.Kind == InputReveal || in.Kind == InputFinish) {
		// ASKED BEFORE "any key advances" swallows it. After a verdict is the
		// state a flag matters MOST in, because a learner discovers a question is
		// broken by reading the reveal — InputDrop and InputQuit sit outside this
		// branch for the same reason.
		//
		// The GESTURE is asked, not Grade: handing every rune to Grade to find
		// out would re-pick the answer on a question already graded.
		if in.Kind == InputRune {
			if opts, ok := flaggedBy(q, in.Rune); ok {
				next, _ := advance(s, q, Skipped, false)
				return next, []Outcome{{
					Kind: OutcomeFlag, Word: q.Word(), Options: opts,
					Form: q.Form(), SessionDone: next.Done,
				}}
			}
		}
		next, out := advance(s, q, Skipped, false)
		return next, []Outcome{out}
	}

	switch in.Kind {
	case InputDrop:
		// `d` IS THE FORM'S when the form holds many words.
		//
		// It was REFUSED here, because `d` names no word on a grid: advance would
		// drop q.Word(), whichever cell that happened to be, and dropping the
		// wrong word is silent. That reasoning is intact — what changed is what
		// happens instead of the drop. The key did nothing on that screen, and
		// the board's labels carried a hole at `d` to protect it, which an
		// operator reading a real grid found confusing rather than safe.
		//
		// So the session reserves `d` for forms that HAVE a current word to
		// remove, and hands it to any other form as an ordinary graded key. A
		// form that does not grade it gets the old behaviour exactly — false,
		// then OutcomeNone.
		if batchOf(q) != nil {
			verdict, ok := q.Grade(in.Rune)
			if !ok {
				return s, []Outcome{{Kind: OutcomeNone}}
			}
			next, out := advance(s, q, verdict, unaidedNow(s, q, verdict))
			return next, []Outcome{out}
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
		// A FORM HOLDING MANY WORDS HAS NOTHING TO REVEAL, and this is the fifth
		// Apply path that has to ask (D12 enumerated four).
		//
		// Its words are all on screen from the first frame and its marks are
		// self-report over a grid; there is no hidden answer to earn. Without
		// this, space set `Revealed` and returned OutcomeReveal carrying
		// `q.Word()` — which on a grid is whichever cell was marked last, or the
		// first cell before any mark — so the loop filed a blank reveal in the
		// append-only buffer and played the pronunciation of a word nobody
		// asked about. Space is the natural key to press: it reveals on both
		// other forms, and a board's prompt does not mention it.
		if batchOf(q) != nil {
			return s, []Outcome{{Kind: OutcomeNone}}
		}
		// The graded case is handled above: once graded they mean "next".
		if s.Revealed {
			return s, []Outcome{{Kind: OutcomeNone}}
		}
		s.Revealed = true
		return s, []Outcome{{Kind: OutcomeReveal, Word: q.Word()}}

	case InputRune:
		// The graded case is handled above.
		// THE GESTURE FIRST, before the key is offered as an answer: a flag is a
		// statement about the QUESTION, and Grade is only asked about answers.
		if opts, ok := flaggedBy(q, in.Rune); ok {
			next, _ := advance(s, q, Skipped, false)
			return next, []Outcome{{
				Kind: OutcomeFlag, Word: q.Word(), Options: opts,
				Form: q.Form(), SessionDone: next.Done,
			}}
		}
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
			// RECALL test, which is what form 2.1 was: the learner rated their own
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

	// A REMOVAL, asked HERE because this is the one place both mark paths meet —
	// the printed key and the click — so the question is put once rather than at
	// two call sites that could drift (#42). It is asked only after a mark
	// actually landed, which is what keeps it from re-firing on a Tab or a
	// refused click.
	if word, ok := droppedBy(q); ok {
		return s, Outcome{Kind: OutcomeDrop, Word: word, SessionDone: s.Done}
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
// Optional deliberately. Form 2.1 could not say why a recall failed — the learner
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

// Flagging is implemented by a form whose last keystroke said the QUESTION is
// broken, rather than that the answer was wrong.
//
// Optional, exactly as Missed and Dropping are, and for the same reason: most
// forms have no such gesture and widening Question would make every one of them
// answer a question it cannot.
//
// NOT A VERDICT. Verdict is what an ANSWER meant, and schedule.Fold reads
// verdicts to move boxes — a fourth one meaning "this question is broken" would
// put material curation in front of the ladder. A flagged question scores
// NOTHING: the word neither promotes nor demotes, because a broken question is
// not evidence about the learner.
//
// It returns the OPTION SET, which is the deliberate opposite of what Missed
// does. Missed records the axis rather than the distractor's word because
// "picked larceny is a fact about one question whose option set no longer
// exists" — but a flag's whole purpose is diagnosing THAT question, so the
// options are the evidence and a flag naming none is "something was wrong once".
//
// ASKED WITH THE RUNE, not read off the form afterwards. The first shape had
// Grade intercept the key and set a one-shot field, which meant the session had
// to hand EVERY rune to Grade to find out — and on a graded question that
// RE-PICKED the answer. "Is this a flag" and "is this an answer" are different
// questions about a keystroke, and keeping them different is what stops one
// corrupting the other. It also leaves the form stateless.
type Flagging interface {
	// Flag reports whether this keystroke is the form's bad-question gesture,
	// and if so the options that made the question bad.
	Flag(k rune) ([]string, bool)
}

// CanFlag reports whether a form has the gesture at all, so the KEYS LINE can
// name it. A prompt promising `?` on a form that ignores it is the bug
// gradePrompt was created to fix, one form later.
func CanFlag(q Question) bool {
	_, ok := q.(Flagging)
	return ok
}

// flaggedBy asks a question whether a keystroke called it broken.
func flaggedBy(q Question, k rune) ([]string, bool) {
	if f, ok := q.(Flagging); ok {
		return f.Flag(k)
	}
	return nil, false
}

// Dropping is implemented by a form whose last mark asked for a word to be
// REMOVED from the deck rather than rated (#42).
//
// Optional, exactly as Missed is: most forms have no such gesture, and widening
// Question would make every one of them answer a question it cannot. A type
// switch on *Board here is the thing #6's Done-when forbids.
//
// NOT A VERDICT, and that is the design decision it encodes. Verdict is what an
// ANSWER meant, and `schedule.Fold` reads verdicts to move boxes — a fourth one
// meaning "remove this word" would put deck curation in front of the ladder and
// make every consumer branch on it.
//
// ONE-SHOT BY CONTRACT: `advance` asks after every mark, so an implementation
// that kept answering would perform one removal repeatedly, against a word
// already gone.
type Dropping interface {
	Dropped() (string, bool)
}

// droppedBy asks a question whether its last mark was a removal.
func droppedBy(q Question) (string, bool) {
	if d, ok := q.(Dropping); ok {
		return d.Dropped()
	}
	return "", false
}

// Batch is implemented by forms that hold MORE THAN ONE word.
//
// Third of its kind beside Missed and SelfRated, and for the same reason both of
// those exist: the session must not learn which form is asking (#6's Done-when),
// so it names a CAPABILITY and asks. A type switch on *Board here would be the
// thing that Done-when forbids.
//
// IT IS CONSULTED ON MORE PATHS THAN ONE, and this comment used to say how many.
//
// It said FOUR and listed three, and the number was wrong within the same commit
// that declared prose enumerations the problem — `InputReveal` was a fifth and
// nobody recounted. That is the fault, not the number: a set the code owns,
// restated in prose, is a second owner and drifts silently.
//
// So the set is `InputKind × Batch` and it is DERIVED, from `numInputKinds`, by
// TestEveryInputKindIsAnsweredForABatchForm. Read that table for what each kind
// means to a form holding many words; a kind added later arrives in it with no
// expectation and fails. `advance` is the obvious consumer besides.
type Batch interface {
	// Words is how many words this form holds, for the bar (D8).
	//
	// The bar's total was `len(s.Questions)`, which is the SLOT count — a
	// twenty-word sitting with a board in it read "0 of 2". The budget is
	// unaffected either way, because schedule.Queue returns that many KEYS
	// whatever they are packed into; what was wrong was the number on screen.
	Words() int
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
	// Resize tells the form the width it will now be DRAWN at, so its rows keep
	// fitting the terminal (R9).
	//
	// On the interface because it is the same fact as the other three: a form
	// that decides where its own words are printed has to be told when the space
	// they are printed in changes. A layout fixed at selection time survives
	// exactly until the window does not — and then a footer entry wraps, stops
	// being one physical row, and a click on the continuation carries a column
	// that means something else.
	Resize(cols int)
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
// Form 2.1 and form 2.5's board: `y` means "I knew it" and nobody verified it,
// and a mark on a grid is triage rather than retrieval. The board's marks are
// `yes` and `no`; an earlier draft called the positive one `firm` and had a
// third, and both went when the operator cut the mark set to two (#40 D7).
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

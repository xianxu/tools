// Package play runs a review session: what to ask, how to read the answer, and
// what the caller must do about it.
//
// ENTIRELY PURE, the same discipline #5's schedule package earned and for the
// same reason: every question here is about state and keystrokes, and a package
// that could reach a terminal or a disk would be untestable at exactly the point
// where correctness lives. TWO guards enforce it — an import allowlist and a
// wall-clock grep — because a comment cannot.
//
// Not three: `schedule` also takes puretest's store-SYMBOL guard, and `play`
// does not, because it names no store symbol at all. A guard with an empty
// allowlist would fatal on finding nothing to check, correctly — it would be
// asserting nothing. If `play` ever imports `store`, that guard has to be added
// deliberately, which is the point.
//
// The division of labour: #5's schedule decides WHICH words are worth asking
// about today; this decides HOW to ask and what the answer means; and the loop
// in package main owns the terminal, the store and the audio.
package play

// Verdict is what an answer meant.
//
// THREE, not two. A learner who skips has not got it wrong, and recording a skip
// as a miss would demote the word through schedule.Answer — punishing honesty
// about a word you half-know is exactly the wrong incentive for a tool whose
// only user is the person it is teaching.
type Verdict int

const (
	// A FORM DOES PRODUCE Skipped now, and this comment used to say none did.
	// `#42`'s `Dropped` mark maps to it, so `Board.Mark` returns (Skipped, true)
	// for a cell being removed — which is the point: a removal is not an
	// assessment, and Skipped is the verdict the session declines to record.
	//
	// It is also still returned as (Skipped, FALSE) by every form for a key it
	// does not grade, which the session ignores entirely. The two are different
	// claims and the bool is what separates them.
	//
	// Skipped is deliberately the ZERO value: Grade returns (Skipped, false) for
	// a key it does not use, so a form that forgets to name a verdict on its
	// ignore path cannot accidentally return Correct or Wrong. The safest
	// outcome is the default one.
	Skipped Verdict = iota
	Correct
	Wrong
)

// Question is what a form must provide, and it is the whole of what the session
// knows about a form.
//
// The Done-when says adding a second form must require no change to the loop.
// That is a property of THIS interface rather than a promise, which is why it
// exists before there is a second form — and why Grade lives here: form 2.1's
// y/n and form 2.3's 1/2/3/4 are the same shape to the session, because the
// session never learns what either key means.
type Question interface {
	// Word is the deck key the answer is recorded against, already normalised.
	Word() string
	// Prompt is what the learner sees BEFORE answering. For a recall form that
	// is the word alone — showing the definition here would defeat the form.
	Prompt() string
	// Reveal is the answer: shown when they ask (space or Enter), and shown
	// unasked when they say they MISSED it, which is the case it mainly serves.
	Reveal() string
	// Grade interprets a graded keystroke. The bool is "this key meant something
	// to me": false lets the session ignore a stray key rather than the form
	// inventing a meaning for it, and the Verdict is then ignored — return the
	// zero value.
	//
	// Grade is asked BEFORE the answer is on screen, not only after — a recall
	// form is rated by the learner, not by the screen. A form whose question is
	// unanswerable unseen (2.3's options, 2.2's cloze) puts that content in
	// Prompt, which is what Prompt is for.
	//
	// The session RESERVES some keys before a form ever sees them: Enter and
	// space reveal (and, once a verdict is in, move on), `d` drops the word, and
	// Ctrl-C quits (see toInput in play_loop.go). A form must not build its
	// answer set from those.
	Grade(r rune) (Verdict, bool)
	// Keys describes the ANSWER keys this form grades, for the line the loop
	// prints under the question — "y = got it, n = missed it".
	//
	// On the interface rather than a constant in the loop because the loop's
	// prompt was a const naming form 2.1's keys, and form 2.3 grades digits: a
	// learner would have been told to press `y` on a screen where only 1-4 did
	// anything. A form that knows which keys it grades is the only thing that
	// can describe them.
	//
	// The form's keys ONLY. The session's reserved keys — `d` and Ctrl-C — are
	// appended by the loop, because they are true whatever form is asking and a
	// form repeating them would be two owners of one fact.
	Keys() string
	// Form is this form's name for the LOG — "recall", "meaning", "board"
	// (#40 D4a).
	//
	// ON THE INTERFACE rather than an optional capability, and that is the whole
	// point: a form that could forget to name itself would write an empty string
	// beside a real promotion, and an event that says "some form" is worse than
	// no field at all — it looks like data. Here a new form cannot compile
	// without answering, which is the same reason Keys() lives here.
	//
	// It is telemetry. Nothing in the scheduler reads it, and schedule.Fold
	// ignoring it is pinned.
	Form() string
}

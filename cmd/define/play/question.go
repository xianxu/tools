// Package play runs a review session: what to ask, how to read the answer, and
// what the caller must do about it.
//
// ENTIRELY PURE, the same discipline #5's schedule package earned and for the
// same reason: every question here is about state and keystrokes, and a package
// that could reach a terminal or a disk would be untestable at exactly the point
// where correctness lives. Three guards enforce it — an import allowlist, a
// wall-clock grep, and a store-SYMBOL allowlist — because a comment cannot.
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
	// Reveal is what they see after asking to see it.
	Reveal() string
	// Grade interprets a graded keystroke. The bool is "this key meant something
	// to me": false lets the session ignore a stray key rather than the form
	// inventing a meaning for it.
	Grade(r rune) (Verdict, bool)
}

package store

import "time"

// EventKind is what happened.
type EventKind string

const (
	EventLookedUp EventKind = "looked-up"
	EventReviewed EventKind = "reviewed"
	// EventAsked is a free-form question put to the model (#16). The QUESTION is
	// recorded and the answer is not: #17 wants to know what the learner asked
	// about — a strong signal of what they are working on — and every consumer of
	// this log is a fold, which answers would bloat for nothing.
	EventAsked EventKind = "asked"
)

// ReviewEvent is one thing that happened, at a time.
//
// Append-only, and deliberately the ONLY record of activity: every statistic #8
// lists — words per day, streak, active days, accuracy — is a fold over these.
// Storing counters alongside would create a second source of truth that drifts.
type ReviewEvent struct {
	Word    string    `yaml:"word,omitempty"`
	Kind    EventKind `yaml:"kind"`
	Found   bool      `yaml:"found"`
	Correct bool      `yaml:"correct,omitempty"`
	// Question is set on EventAsked. Word may be empty beside it — a question
	// asked before any lookup has no word — which is what generalised complete()
	// below from "has a word" to "has a subject".
	Question string `yaml:"question,omitempty"`
	// Missed is WHY a wrong answer was wrong, when the form can say: the axis of
	// the option the learner picked ("domain", "register", "general"). #7's
	// form 2.3 sets it; the board does not, because a mark carries no error kind
	// (nor could form 2.1, which #42 retired — a failed recall had no kind either).
	//
	// Empty on every correct answer and omitted from the file (D8) — writing
	// something on a right answer would put a word in the log that #17 M2's
	// weakness taxonomy then has to filter back out.
	//
	// The AXIS rather than the distractor's word (D7): "picked the Law one" is a
	// finding a later reader can interpret, while "picked larceny" is a fact
	// about one question whose option set no longer exists. It also keeps the
	// event small and stable while the deck churns underneath it.
	Missed string `yaml:"missed,omitempty"`
	// Unaided marks a right answer given COLD — the form compared the learner's
	// answer to one it already knew, and no reveal preceded it.
	//
	// An OBSERVATION, never a claim. Form 2.1's `y` means "I knew it" with
	// nobody checking, so it never sets this; form 2.3's correct pick does.
	// The distinction is what keeps the ladder's two-rung promotion honest —
	// see schedule.GradeUnaided and play.SelfRated.
	//
	// Absent on every event written before this existed, which folds to
	// GradeCorrect: one rung rather than two, the conservative reading.
	Unaided bool `yaml:"unaided,omitempty"`
	// Form is WHICH FORM ASKED — "recall", "meaning", "board" (#40 D4a).
	//
	// TELEMETRY, not assessment. Fold does not read it and must not: nothing
	// about the ladder depends on which form produced an answer, and a scheduler
	// that started branching on it would make this field load-bearing when it is
	// meant to be observational.
	//
	// It exists because form 2.5 promotes a word on SELF-REPORT, and the two
	// remedies for that — promote more slowly, or offer the board less often —
	// are both deferred to be chosen from evidence. The query is "did a
	// board-promoted word fail beyond reasonable in a real recall test later",
	// which joins a promotion to the word's NEXT test — so the form has to be on
	// the event that promoted it, not on a summary somewhere else.
	//
	// The damage it watches for is silent and delayed: a wrongly promoted word
	// vanishes for weeks, and when it is eventually forgotten that is
	// indistinguishable from ordinary forgetting. Without this the log cannot
	// tell the two apart, and neither remedy can be chosen from anything but
	// argument.
	//
	// NAMES rather than the project's form numbers, which are 2.1, 2.3 and 2.5.
	// A log is read years later by a script or a person, and "board" needs no
	// atlas to decode while "2.5" does.
	//
	// Absent on every event written before this existed, which reads as "some
	// earlier form" and is the truth.
	Form string `yaml:"form,omitempty"`
	// At stays LAST, and a field added after it would break the torn-record rule
	// silently. See complete(): the rule is termination PLUS completeness, and
	// completeness leans on a cut record losing its timestamp. A field written
	// after `at:` would survive the cut that drops `at`, and a fragment would
	// then look whole.
	At time.Time `yaml:"at"`
}

// complete reports whether an event carries every field a real one does.
//
// Half of the torn-record test: a fragment that happens to parse is missing
// something. The other half is termination — see parseDay, which needs both,
// because a cut inside the timestamp leaves a shorter date that IS a valid time.
func (e ReviewEvent) complete() bool {
	// A record identifies its SUBJECT — a word for a lookup, a question for an
	// asked event. That generalisation is what the asked event forced: requiring
	// a word would have dropped every question asked before a lookup, which is
	// exactly the history #17 reads.
	return (e.Word != "" || e.Question != "") && e.Kind != "" && !e.At.IsZero()
}

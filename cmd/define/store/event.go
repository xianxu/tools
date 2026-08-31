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
	// form 2.3 sets it; form 2.1 cannot, because a failed recall has no kind.
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

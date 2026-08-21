package store

import "time"

// EventKind is what happened.
type EventKind string

const (
	EventLookedUp EventKind = "looked-up"
	EventReviewed EventKind = "reviewed"
)

// ReviewEvent is one thing that happened, at a time.
//
// Append-only, and deliberately the ONLY record of activity: every statistic #8
// lists — words per day, streak, active days, accuracy — is a fold over these.
// Storing counters alongside would create a second source of truth that drifts.
type ReviewEvent struct {
	Word    string    `yaml:"word"`
	Kind    EventKind `yaml:"kind"`
	Found   bool      `yaml:"found"`
	Correct bool      `yaml:"correct,omitempty"`
	At      time.Time `yaml:"at"`
}

// complete reports whether an event carries every field a real one does.
//
// Half of the torn-record test: a fragment that happens to parse is missing
// something. The other half is termination — see parseDay, which needs both,
// because a cut inside the timestamp leaves a shorter date that IS a valid time.
func (e ReviewEvent) complete() bool {
	return e.Word != "" && e.Kind != "" && !e.At.IsZero()
}

package store

import "time"

// Store persists the deck and the activity log.
//
// Two implementations ship — in-memory and YAML-on-disk — and both are held to
// one shared conformance suite (storetest.Suite). "The fake behaves like the
// real thing" is otherwise a claim rather than a test.
type Store interface {
	// Upsert records a word, merging with any existing entry: FirstSeen is kept,
	// LastSeen advances, Lookups accumulates.
	Upsert(w Word) error
	// Deck returns every word, ordered by LastSeen descending (most recent first).
	Deck() ([]Word, error)
	// AppendEvent records one thing that happened. Append-only.
	AppendEvent(e ReviewEvent) error
	// Events returns events at or after since, in chronological order.
	Events(since time.Time) ([]ReviewEvent, error)
	// UserModel returns the learner model this directory holds, or "" when there
	// is none. Absent is NOT an error: it is the normal first-run state, and the
	// file is #17's to write.
	//
	// It lives on the Store because the store already owns which directory this
	// session is — words/, events/ and user-model.md are the three artifacts in
	// it, and reading this one anywhere else would be a second answer to that
	// question.
	UserModel() (string, error)
	// Forget removes a word from the deck. It does NOT remove events: the deck is
	// a working set, the log is history, and rewriting the past would corrupt
	// every statistic derived from it. Reports whether anything was removed;
	// absence is not an error.
	Forget(key string) (removed bool, err error)
}

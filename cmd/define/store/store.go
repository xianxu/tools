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
}

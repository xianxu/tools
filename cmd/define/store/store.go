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
	// SetUserModel writes it. Here because a getter the reference implementation
	// cannot hold state for makes the conformance row "empty before anything
	// writes one" assert the only value Mem can produce — unfalsifiable, and a
	// fake that cannot model the real one's state is the gap storetest exists to
	// close (#16 M2, BR-45). #17 is the consumer that writes it for real.
	SetUserModel(text string) error
	// NewsItems returns the cached feed items for a word and WHEN they were
	// fetched. A zero time means never fetched, which is deliberately distinct
	// from "fetched and found nothing" — some words are simply not in the news,
	// that is a real answer, and a caller that cannot tell the two apart either
	// re-fetches those words forever or freezes them empty.
	NewsItems(key string) ([]NewsItem, time.Time, error)
	// SetNewsItems replaces the cache for a word, recording the fetch time.
	// Only successful fetches are written here — see cachingFeed for why a
	// FAILED fetch must not be cached.
	SetNewsItems(key string, items []NewsItem, at time.Time) error
	// Forget removes a word from the deck. It does NOT remove events: the deck is
	// a working set, the log is history, and rewriting the past would corrupt
	// every statistic derived from it. Reports whether anything was removed;
	// absence is not an error.
	Forget(key string) (removed bool, err error)
}

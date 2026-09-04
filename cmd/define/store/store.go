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
	// session is — words/, events/ and the learner model are the three artifacts in
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
	// WordFacts returns what --harvest cached about a word: its CEFR band and its
	// domain. An unharvested word reads as the zero value, which is NOT an error
	// — it is the state of every word until the first harvest. WordFacts.At is
	// what tells the two apart, so this needs no second return value the way
	// NewsItems does.
	//
	// A stored record too damaged to parse also reads as unharvested, on purpose:
	// both are worth one re-ask, and a half-trusted band would reach the
	// comparison every distractor rule depends on.
	WordFacts(key string) (WordFacts, error)
	// SetWordFacts replaces a word's facts. REPLACES, not merges: a band and a
	// domain are one judgement made in one call, and a merge would leave a word
	// holding half of one run and half of another.
	SetWordFacts(key string, f WordFacts) error
	// Items returns the practice items authored for a word, or none. A word may
	// hold several — of different Forms — and #12 picks among the ones it renders.
	//
	// READ-SIDE RULE, which this surface and WordFacts both obey: a record read
	// back out of a RuntimeDirs directory is UNTRUSTED INPUT and goes through the
	// same canonicalisation its write applies. Not because the writer is
	// suspect — because these files are documented as inspectable, so a
	// hand-edited one is an invited workflow and an older build's output is a
	// certainty.
	//
	// NewsItems is deliberately OUT of the class, recorded here rather than left
	// ambiguous: its fields are a feed's text, not a model's, and nothing renders
	// them into a structure a newline could forge. If that changes — if a
	// headline ever reaches the board — it joins the rule.
	Items(key string) ([]Item, error)
	// SetItems replaces a word's authored items, for the same reason
	// SetNewsItems replaces rather than appends: a re-harvest must not silently
	// double a word's material every run.
	SetItems(key string, items []Item) error
	// Forget removes a word from the deck. It does NOT remove events: the deck is
	// a working set, the log is history, and rewriting the past would corrupt
	// every statistic derived from it. Reports whether anything was removed;
	// absence is not an error.
	Forget(key string) (removed bool, err error)
}

package store

import "time"

// NewsItem is one raw item from a news feed, exactly as the feed reported it.
//
// It is the ONLY thing cached, and it deliberately knows nothing about usage,
// filtering or provenance: the store's job is "what did the feed say", not "what
// is worth teaching". Everything the learner sees is derived from these at read
// time, so improving that derivation improves every word already on disk without
// re-fetching anything.
type NewsItem struct {
	Title  string    `yaml:"title"`
	URL    string    `yaml:"url"`
	Source string    `yaml:"source"` // the publisher, from the feed's <source>
	At     time.Time `yaml:"at"`
}

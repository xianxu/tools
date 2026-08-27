package main

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// Where a usage came from. A field rather than two types because the consumer
// (#10) wants "sentences for this word, tagged by provenance" — splitting them
// into parallel slices would push the join into every caller.
const (
	usageNews = "news"
	usageNOAD = "noad"
)

// Usage is one real sentence containing the word, ready to show or to author
// from.
//
// Derived at READ time, never cached: only store.NewsItem goes to disk. That is
// what lets containsWord improve and every word already cached improve with it,
// without re-fetching anything.
type Usage struct {
	Text      string
	Source    string // usageNews | usageNOAD
	Publisher string // empty for NOAD
	URL       string
	At        time.Time
}

// UsageSource answers "what real sentences use this word".
//
// The Entry is a parameter because NOAD's own examples are one of the two
// sources and the caller already has the parsed entry in hand — refetching or
// reparsing it here would be the seam doing work its caller already did.
//
// NO ERROR RETURN, and that is a contract rather than an oversight: the
// dictionary half is computed from an Entry the caller already holds, so there
// is always an answer. A feed failure DEGRADES to it and is reported on the warn
// writer. The first version returned an error that was nil on every path — a
// dead branch for #10 to write code against.
type UsageSource interface {
	Usages(ctx context.Context, word string, e Entry) []Usage
}

// containsWord reports whether text genuinely contains the word.
//
// The feed is queried with the word quoted and STILL returns items that do not
// contain it — measured, 12 to 99 matching out of 41 to 100 — so this filter is
// load-bearing, not defensive.
//
// It delegates to highlightSpans rather than matching here. That is the whole
// matcher: store.Key normalisation for case, phraseGap/phraseRunsJoin for
// multi-word entries like `hot dog`, joiner trimming so `hot-dog` and
// `'ephemeral'` behave. Reimplementing any of it would re-commit #21's
// consolidation finding one level up, and the divergence would be VISIBLE to the
// learner: a headline whose word renders green while this filter rejects the
// same headline. TestContainsWordAgreesWithTheHighlighter is the pin.
func containsWord(text, word string) bool {
	if text == "" || strings.TrimSpace(word) == "" {
		return false
	}
	v := &memVocabulary{}
	v.Add(word)
	for _, s := range highlightSpans(text, v) {
		if s.known {
			return true
		}
	}
	return false
}

// usagesFrom filters raw items down to the ones that actually use the word, and
// shapes them.
func usagesFrom(items []store.NewsItem, word string) []Usage {
	var out []Usage
	for _, it := range items {
		text := withoutAttribution(it.Title, it.Source)
		if !containsWord(text, word) {
			continue
		}
		out = append(out, Usage{
			Text:      text,
			Source:    usageNews,
			Publisher: it.Source,
			URL:       it.URL,
			At:        it.At,
		})
	}
	return out
}

// withoutAttribution drops the " - Publisher" a news feed appends to a headline.
//
// Found by capturing a real feed rather than by reasoning about one: every
// Google News title ends that way, and keeping it would end every sentence the
// learner reads with an unrelated proper noun.
//
// It strips only when the suffix IS the publisher the feed named, so a headline
// that legitimately contains a dash keeps it. "Remove everything after the last
// dash" would have eaten `Ephemeral - a study in impermanence`.
func withoutAttribution(title, publisher string) string {
	if publisher == "" {
		return title
	}
	if trimmed, ok := strings.CutSuffix(title, " - "+publisher); ok {
		return trimmed
	}
	return title
}

// entryUsages lifts NOAD's own example sentences into the same shape.
//
// The free source, and the one that keeps a word from being taught only through
// this week's news cycle — the feed's thematic collapse is measured and real (10
// of 14 `sycophantic` headlines were about AI chatbots), and a dictionary's
// examples are not current, which is exactly the point.
//
// Filtered by the same containsWord: an example under sense 3 of `bank` may
// illustrate a phrase without using the headword.
func entryUsages(e Entry, word string) []Usage {
	var out []Usage
	for _, blk := range e.Blocks {
		for _, s := range blk.Senses {
			for _, ex := range s.Examples {
				if !containsWord(ex.Text, word) {
					continue
				}
				out = append(out, Usage{Text: ex.Text, Source: usageNOAD})
			}
		}
	}
	return out
}

// bothSources is the seam's real implementation: the feed and the dictionary,
// merged.
//
// News first, because it is the current half and the reason this issue exists;
// NOAD always, because it is free, offline, and not current — which is exactly
// the complement. The feed's thematic collapse is measured (ten of fourteen
// `sycophantic` headlines were about AI chatbots), so a word sourced only from
// this week is taught narrowly.
//
// A news failure DEGRADES rather than propagating: the dictionary half still
// runs, and half the sentences beat none. A nil news source is the same case —
// "no feed configured" is not an error here, the same shape every other seam in
// this package uses for absent.
type bothSources struct {
	news *cachingFeed
	warn io.Writer

	mu     sync.Mutex
	warned bool
}

func (b *bothSources) Usages(ctx context.Context, word string, e Entry) []Usage {
	var out []Usage
	if b.news != nil {
		items, err := b.news.items(ctx, word)
		if err != nil {
			// Degrading is right; degrading SILENTLY is not this package's
			// shape. A permanently broken feed — wrong URL, TLS failure, Google
			// blocking us — would otherwise be indistinguishable from "this word
			// is not in the news", both for the reader and for #10.
			//
			// Once per session, like storeCapturer's write warning: a failing
			// feed fails for every word, and one line per lookup is noise.
			b.warnOnce("could not read the news feed (%v); using dictionary examples only", err)
		}
		out = append(out, usagesFrom(items, word)...)
	}
	return append(out, entryUsages(e, word)...)
}

func (b *bothSources) warnOnce(format string, args ...any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.warned {
		return
	}
	b.warned = true
	warnTo(b.warn, format, args...)
}

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// cacheTTL is how long cached feed items are considered fresh.
//
// A week, because the thing being cached is "how is this word being used lately"
// and that does not turn over hourly. It is not a correctness knob: a stale entry
// is refreshed when the network allows and SERVED when it does not, so the only
// thing shortening it buys is more requests.
const cacheTTL = 7 * 24 * time.Hour

// maxFeedBytes caps a response, mirroring maxAudioBytes in fetch.go. The
// measured feeds are ~140 KB; anything far larger is not a feed.
const maxFeedBytes = 8 << 20

// feed is the raw transport: bytes in, bytes out. Deliberately not parsed here,
// so the cache wrapper and the fake both sit on the same narrow surface and
// parseRSS stays in the path for both.
type feed interface {
	Fetch(ctx context.Context, word string) ([]byte, error)
}

// httpFeed fetches Google News RSS.
//
// The SERP is not an option and that was measured, not assumed: an earlier probe
// of google.com/search returned a 91 KB JS shell with zero usable content.
// That measurement is why this is RSS-shaped.
type httpFeed struct{ client *http.Client }

func newHTTPFeed() *httpFeed {
	return &httpFeed{client: &http.Client{Timeout: 20 * time.Second}}
}

func (h *httpFeed) Fetch(ctx context.Context, word string) ([]byte, error) {
	// The word is quoted in the query so the feed prefers items containing it.
	// It still returns items that do not — measured, 12 to 99 matching out of 41
	// to 100 — which is why containsWord exists downstream.
	u := "https://news.google.com/rss/search?q=" + url.QueryEscape(`"`+word+`"`) +
		"&hl=en-US&gl=US&ceid=US:en"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("news feed: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes))
}

// cachingFeed serves feed items from the store, fetching only when it must.
//
// Three outcomes, and the first draft of the plan modelled two:
//
//  1. Fetch succeeds WITH items -> cache, serve.
//  2. Fetch succeeds with ZERO items -> cache it, with its timestamp. A real
//     answer: some words are simply not in the news. Not caching it would
//     re-fetch forever for exactly the words the feed is worst at.
//  3. Fetch FAILS -> do NOT cache. A network blip must not become a permanent
//     empty answer for that word.
//
// Because (2) is cached, entries carry a fetch time and go stale — and a stale
// entry whose refresh fails is SERVED anyway. That is what keeps "works offline"
// true while letting a word that had no news last week pick some up this week.
type cachingFeed struct {
	inner feed
	st    store.Store
	clock store.Clock
}

func newCachingFeed(inner feed, st store.Store, clock store.Clock) *cachingFeed {
	return &cachingFeed{inner: inner, st: st, clock: clock}
}

// items returns the cached-or-fetched raw items for a word.
//
// Raw, not filtered: only the feed's own words go to disk, and everything the
// learner sees is derived at read time. That is what lets containsWord improve
// and every word already cached improve with it, with no re-fetch.
func (c *cachingFeed) items(ctx context.Context, word string) ([]store.NewsItem, error) {
	key := store.Key(word)
	cached, at, err := c.st.NewsItems(key)
	if err != nil {
		// An unreadable cache is a miss, not a failure: re-fetching costs what a
		// miss costs anyway.
		cached, at = nil, time.Time{}
	}
	if !at.IsZero() && c.clock.Now().Sub(at) < cacheTTL {
		return cached, nil
	}

	body, err := c.inner.Fetch(ctx, word)
	if err != nil {
		if !at.IsZero() {
			return cached, nil // stale beats nothing
		}
		return nil, err // outcome 3: nothing to fall back on, and NOT cached
	}
	items, err := parseRSS(body)
	if err != nil {
		if !at.IsZero() {
			return cached, nil
		}
		return nil, err
	}
	// Outcomes 1 and 2 are the same write: a successful fetch is recorded with
	// its time, whether or not it found anything.
	if err := c.st.SetNewsItems(key, items, c.clock.Now()); err != nil {
		// The answer is good even if we could not keep it. Failing here would
		// trade a usable result for a bookkeeping problem.
		return items, nil
	}
	return items, nil
}

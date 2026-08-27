package main

import (
	"io"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

func newsRig(t *testing.T, body []byte, now time.Time) (*cachingFeed, *fakeFeed, store.Store) {
	t.Helper()
	st := store.NewMem()
	f := newFakeFeed(body)
	return newCachingFeed(f, st, store.FixedClock(now)), f, st
}

// Outcome 1: a successful fetch with items is cached and served from cache.
//
// Asserted on the fake's COUNTER. Asserting on the returned items instead would
// pass for a cache that re-fetches every time and returns the same answer.
func TestFetchSucceedsAndIsCached(t *testing.T) {
	body := rssFixture(t, "ephemeral.rss")
	c, f, _ := newsRig(t, body, aDay)

	first, err := c.items(t.Context(), "ephemeral")
	if err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	second, err := c.items(t.Context(), "ephemeral")
	if err != nil {
		t.Fatalf("second fetch: %v", err)
	}

	if len(first) != 13 || len(second) != 13 {
		t.Errorf("got %d then %d items, want the fixture's 13 both times", len(first), len(second))
	}
	if got := f.fetches("ephemeral"); got != 1 {
		t.Errorf("reached the network %d times, want 1 — the second call must come from cache", got)
	}
}

// Outcome 2: a fetch that succeeds with ZERO items is a real answer and is
// cached. Some words are simply not in the news; re-fetching them forever is the
// cache failing at exactly the words the feed is worst at.
func TestSuccessfulEmptyFetchIsCached(t *testing.T) {
	empty := []byte(`<?xml version="1.0"?><rss version="2.0"><channel></channel></rss>`)
	c, f, _ := newsRig(t, empty, aDay)

	if _, err := c.items(t.Context(), "quokka"); err != nil {
		t.Fatalf("first fetch: %v", err)
	}
	if _, err := c.items(t.Context(), "quokka"); err != nil {
		t.Fatalf("second fetch: %v", err)
	}

	if got := f.fetches("quokka"); got != 1 {
		t.Errorf("reached the network %d times, want 1 — an empty ANSWER is still an answer", got)
	}
}

// Outcome 3: a FAILED fetch is not cached. A network blip must not become a
// permanent empty answer for that word.
func TestFailedFetchIsNotCached(t *testing.T) {
	c, f, _ := newsRig(t, rssFixture(t, "ephemeral.rss"), aDay)
	f.failWith(errFeedDown)

	if _, err := c.items(t.Context(), "ephemeral"); err == nil {
		t.Fatal("want the failure reported")
	}
	f.succeed()
	got, err := c.items(t.Context(), "ephemeral")
	if err != nil {
		t.Fatalf("retry after recovery: %v", err)
	}

	if len(got) != 13 {
		t.Errorf("got %d items after recovery, want 13 — the failure was cached", len(got))
	}
	if n := f.fetches("ephemeral"); n != 2 {
		t.Errorf("reached the network %d times, want 2 — a failure must not be cached", n)
	}
}

// A stale entry is re-fetched, so a word that had no news last week can pick some
// up this week.
func TestStaleCacheIsRefetched(t *testing.T) {
	st := store.NewMem()
	f := newFakeFeed(rssFixture(t, "ephemeral.rss"))
	clk := &movingClock{now: aDay}
	c := newCachingFeed(f, st, clk)

	if _, err := c.items(t.Context(), "ephemeral"); err != nil {
		t.Fatal(err)
	}
	clk.now = aDay.Add(cacheTTL + time.Hour)
	if _, err := c.items(t.Context(), "ephemeral"); err != nil {
		t.Fatal(err)
	}

	if got := f.fetches("ephemeral"); got != 2 {
		t.Errorf("reached the network %d times, want 2 — a stale entry must be refreshed", got)
	}
}

// ...and a stale entry whose re-fetch FAILS serves the stale copy. This is what
// keeps "works offline" true: yesterday's sentences beat none.
func TestStaleCacheFallsBackWhenRefetchFails(t *testing.T) {
	st := store.NewMem()
	f := newFakeFeed(rssFixture(t, "ephemeral.rss"))
	clk := &movingClock{now: aDay}
	c := newCachingFeed(f, st, clk)

	if _, err := c.items(t.Context(), "ephemeral"); err != nil {
		t.Fatal(err)
	}
	clk.now = aDay.Add(cacheTTL + time.Hour)
	f.failWith(errFeedDown)

	got, err := c.items(t.Context(), "ephemeral")

	if err != nil {
		t.Fatalf("a failed refresh must fall back, not fail: %v", err)
	}
	if len(got) != 13 {
		t.Errorf("got %d items, want the 13 stale ones — offline must still work", len(got))
	}
}

// movingClock lets a test stand at a chosen instant and then move.
type movingClock struct{ now time.Time }

func (c *movingClock) Now() time.Time { return c.now }

// The seam's whole point: news failing degrades to the dictionary rather than to
// nothing. The Done-when says NOAD's examples are available "through the same
// seam", and this is what that buys.
func TestBothSourcesDegradesToNOADWhenNewsFails(t *testing.T) {
	c, f, _ := newsRig(t, rssFixture(t, "ephemeral.rss"), aDay)
	f.failWith(errFeedDown)
	src := &bothSources{news: c}
	e := ParseEntry(fixture(t, "bank"))

	got, err := src.Usages(t.Context(), "bank", e)

	if err != nil {
		t.Fatalf("a failing feed must degrade, not fail: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("no usages at all — the dictionary half did not run")
	}
	for _, u := range got {
		if u.Source != usageNOAD {
			t.Errorf("usage %q tagged %q, want only dictionary usages", u.Text, u.Source)
		}
	}
}

// With both working, both arrive, news first.
func TestBothSourcesMergesNewsAndNOAD(t *testing.T) {
	c, _, _ := newsRig(t, rssFixture(t, "ephemeral.rss"), aDay)
	src := &bothSources{news: c}
	e := ParseEntry(fixture(t, "ephemeral"))

	got, err := src.Usages(t.Context(), "ephemeral", e)
	if err != nil {
		t.Fatal(err)
	}

	var news, noad int
	for _, u := range got {
		switch u.Source {
		case usageNews:
			news++
		case usageNOAD:
			noad++
		}
	}
	if news == 0 {
		t.Error("no news usages")
	}
	if noad == 0 {
		t.Error("no dictionary usages — the free source is what keeps a word from being taught only through this week")
	}
	if got[0].Source != usageNews {
		t.Errorf("first usage is %q, want news first", got[0].Source)
	}
}

// A nil news source is "no feed configured", not a crash — the same shape every
// other seam in this package uses for absent.
func TestBothSourcesWorksWithNoFeedAtAll(t *testing.T) {
	src := &bothSources{}
	e := ParseEntry(fixture(t, "bank"))

	got, err := src.Usages(t.Context(), "bank", e)

	if err != nil {
		t.Fatalf("no feed must not be an error: %v", err)
	}
	if len(got) == 0 {
		t.Error("the dictionary half must still run")
	}
}

// #21's lesson applied BEFORE the fact rather than after: the enumeration is
// every process entry path that can reach the seam, driven through production
// wiring with the dependency in its REAL initial state.
//
// #21 shipped two dead entry paths because its tests injected a pre-filled
// dependency and so began after the hop that fills it. There is no user-facing
// consumer here yet, so what this pins is the wiring itself — that openStore
// builds a source and withStore carries it through, which is the hop that was
// missing in #21 and would be missing here for exactly the same reason.
func TestWithStoreCarriesTheUsageSourceThrough(t *testing.T) {
	// Parsed BEFORE the chdir: fixture() reads relative to the working
	// directory, and this test moves it.
	e := ParseEntry(fixture(t, "bank"))
	t.Chdir(t.TempDir())

	d := deps{newStore: openStore}.withStore(options{}, io.Discard)

	if d.usage == nil {
		t.Fatal("withStore left deps.usage nil — #10 would have no sentences")
	}
	// And it reaches the dictionary half without a network, which is the
	// degradation the seam promises.
	got, err := d.usage.Usages(t.Context(), "bank", e)
	if err != nil {
		t.Fatalf("Usages: %v", err)
	}
	if len(got) == 0 {
		t.Error("no usages offline — the dictionary half is not wired")
	}
}

// The opt-out path still yields a usable process: DEFINE_NO_CAPTURE means
// "write nothing here", not "the seam does not exist".
func TestNoCaptureStillLeavesAUsageSource(t *testing.T) {
	t.Chdir(t.TempDir())

	d := deps{newStore: openStore}.withStore(options{noCapture: true}, io.Discard)

	if d.usage == nil {
		t.Error("no-capture left deps.usage nil")
	}
}

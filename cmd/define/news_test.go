package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
	var warn bytes.Buffer
	src := &bothSources{news: c, warn: &warn}
	e := ParseEntry(fixture(t, "bank"))

	got := src.Usages(t.Context(), "bank", e)

	if len(got) == 0 {
		t.Fatal("no usages at all — the dictionary half did not run")
	}
	// Degrading is right; degrading SILENTLY would make a permanently broken
	// feed indistinguishable from "this word is not in the news".
	if !strings.Contains(warn.String(), "news feed") {
		t.Errorf("the feed failure was swallowed: warn = %q", warn.String())
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

	got := src.Usages(t.Context(), "ephemeral", e)

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

	got := src.Usages(t.Context(), "bank", e)

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
	t.Chdir(t.TempDir())

	d := deps{newStore: openStore}.withStore(options{}, io.Discard)

	// The WIRING HOP, and only that. The first version of this test also called
	// Usages on production deps — which builds a real httpFeed, so every plain
	// `go test` fetched news.google.com and wrote ~48 KB of live headlines into
	// a temp dir. It passed either way, because bothSources degrades to the
	// dictionary, so the assertion could not tell network from no-network while
	// its own comment claimed "without a network".
	//
	// The offline claim now lives in TestUsagesOfflineWithNoFeed, on a source
	// built with no feed at all. Two claims, two tests, neither lying.
	if d.usage == nil {
		t.Fatal("withStore left deps.usage nil — #10 would have no sentences")
	}
	if _, ok := d.usage.(*bothSources); !ok {
		t.Errorf("deps.usage is %T, want the merged source", d.usage)
	}
}

// The offline half, with no feed in the picture at all — so "offline" is a
// property of the code rather than of whether the test machine has a network.
func TestUsagesOfflineWithNoFeed(t *testing.T) {
	e := ParseEntry(fixture(t, "bank"))
	src := &bothSources{} // no feed, no network, no cache

	got := src.Usages(t.Context(), "bank", e)

	if len(got) == 0 {
		t.Fatal("no usages with no feed — the dictionary half is not reachable")
	}
	for _, u := range got {
		if u.Source != usageNOAD {
			t.Errorf("usage %q tagged %q with no feed configured", u.Text, u.Source)
		}
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

// BR-14's enumeration, swept: every branch in news.go whose comment states "on
// failure X we do Y instead" gets one test that reddens when Y is removed.
//
// BR-11 swept that rule over output FIELDS; the degradation PATHS were never
// enumerated, and the coverage profile named them all at once. The sibling rule:
// a comment describing a fallback is a claim, and an uncovered branch is a claim
// nothing checks.
func TestUnreadableCacheIsAMissNotAFailure(t *testing.T) {
	// failingStore returns errFail for NewsItems — a fixture that was already in
	// the tree and unused for this.
	f := newFakeFeed(rssFixture(t, "ephemeral.rss"))
	c := newCachingFeed(f, failingStore{}, store.FixedClock(aDay))

	got, err := c.items(t.Context(), "ephemeral")

	if err != nil {
		t.Fatalf("an unreadable cache must be a miss, not a failure: %v", err)
	}
	if len(got) != 13 {
		t.Errorf("got %d items, want the 13 fetched despite the cache being unreadable", len(got))
	}
}

// A cache WRITE that fails must not lose the answer we already have: failing
// there would trade a usable result for a bookkeeping problem.
func TestCacheWriteFailureStillReturnsTheAnswer(t *testing.T) {
	f := newFakeFeed(rssFixture(t, "ephemeral.rss"))
	c := newCachingFeed(f, writeFailsStore{Store: store.NewMem()}, store.FixedClock(aDay))

	got, err := c.items(t.Context(), "ephemeral")

	if err != nil {
		t.Fatalf("a failed cache write must not fail the fetch: %v", err)
	}
	if len(got) != 13 {
		t.Errorf("got %d items, want 13", len(got))
	}
}

// A body that does not parse behaves like a failed fetch: stale beats nothing,
// and with nothing stale it is an error rather than a silent empty answer.
func TestUnparseableBodyFallsBackOrErrors(t *testing.T) {
	t.Run("with nothing cached it is an error", func(t *testing.T) {
		c := newCachingFeed(newFakeFeed([]byte("not xml")), store.NewMem(), store.FixedClock(aDay))
		if _, err := c.items(t.Context(), "ephemeral"); err == nil {
			t.Error("want an error — a silent empty answer would read as 'not in the news'")
		}
	})
	t.Run("with something stale it falls back", func(t *testing.T) {
		f := newFakeFeed(rssFixture(t, "ephemeral.rss"))
		clk := &movingClock{now: aDay}
		c := newCachingFeed(f, store.NewMem(), clk)
		if _, err := c.items(t.Context(), "ephemeral"); err != nil {
			t.Fatal(err)
		}
		clk.now = aDay.Add(cacheTTL + time.Hour)
		f.body = []byte("not xml")

		got, err := c.items(t.Context(), "ephemeral")

		if err != nil {
			t.Fatalf("a stale entry must survive an unparseable refresh: %v", err)
		}
		if len(got) != 13 {
			t.Errorf("got %d items, want the 13 stale ones", len(got))
		}
	})
}

// httpFeed's own error branches, against a local server rather than the live
// feed — the whole point of BR-13's fix.
func TestHTTPFeedErrors(t *testing.T) {
	t.Run("a non-200 is an error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer srv.Close()
		h := &httpFeed{client: srv.Client(), base: srv.URL}

		if _, err := h.Fetch(t.Context(), "ephemeral"); err == nil {
			t.Error("want an error for a non-200")
		}
	})
	t.Run("a cancelled context does not reach the network", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Error("the server was reached despite a cancelled context")
		}))
		defer srv.Close()
		h := &httpFeed{client: srv.Client(), base: srv.URL}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		if _, err := h.Fetch(ctx, "ephemeral"); err == nil {
			t.Error("want an error for a cancelled context")
		}
	})
	// The word is QUOTED in the query so the feed prefers items containing it.
	// Dropping the quotes changed no test before this one — the fixture is
	// served regardless of what was asked for, which is exactly what a fake
	// cannot tell you and a captured request can.
	t.Run("the word is quoted in the query", func(t *testing.T) {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.Query().Get("q")
		}))
		defer srv.Close()
		h := &httpFeed{client: srv.Client(), base: srv.URL}

		if _, err := h.Fetch(t.Context(), "hot dog"); err != nil {
			t.Fatalf("Fetch: %v", err)
		}

		if gotQuery != `"hot dog"` {
			t.Errorf("q = %q, want the word quoted — unquoted, the feed returns items about either word", gotQuery)
		}
	})
	t.Run("a body is capped", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			for i := 0; i < maxFeedBytes/1024+16; i++ {
				w.Write(make([]byte, 1024))
			}
		}))
		defer srv.Close()
		h := &httpFeed{client: srv.Client(), base: srv.URL}

		body, err := h.Fetch(t.Context(), "ephemeral")

		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		if len(body) > maxFeedBytes {
			t.Errorf("read %d bytes, want at most %d", len(body), maxFeedBytes)
		}
	})
}

// writeFailsStore accepts reads and refuses cache writes.
type writeFailsStore struct{ store.Store }

func (writeFailsStore) SetNewsItems(string, []store.NewsItem, time.Time) error {
	return errFeedDown
}

// BR-15's real lesson: I added `warn`, tested it by passing a buffer into a
// hand-built bothSources, and never wired it at either production site — so
// production degraded exactly as silently as before while the test was green.
//
// That is the wiring-hop class for the third time in two issues (#21's vocab
// Load, #9's no-capture seam, now this). A test that constructs the struct
// begins AFTER the hop that fills its fields. This one builds deps the way a
// process does and asserts the field arrived.
func TestProductionWiringGivesTheUsageSourceItsWarnWriter(t *testing.T) {
	for _, tc := range []struct {
		name string
		opt  options
	}{
		{"with a durable store", options{}},
		{"no-capture", options{noCapture: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			var warn bytes.Buffer

			d := deps{newStore: openStore}.withStore(tc.opt, &warn)

			src, ok := d.usage.(*bothSources)
			if !ok {
				t.Fatalf("deps.usage is %T", d.usage)
			}
			if src.warn == nil {
				t.Fatal("the usage source has no warn writer — a broken feed would degrade silently")
			}
			// And it is the writer the caller handed in, not some other sink.
			src.warnOnce("probe %d", 1)
			if !strings.Contains(warn.String(), "probe 1") {
				t.Errorf("the warning went somewhere else: warn = %q", warn.String())
			}
		})
	}
}

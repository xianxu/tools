package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func rssFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "news", name))
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	return b
}

// The captured feed. Not invented: you cannot fake the shape of something you
// have not looked at, and this one turned up a detail the plan did not
// anticipate — every title carries a " - Publisher" suffix.
func TestParseRSSOverACapturedFeed(t *testing.T) {
	items, err := parseRSS(rssFixture(t, "ephemeral.rss"))
	if err != nil {
		t.Fatalf("parseRSS = %v", err)
	}
	if len(items) != 13 {
		t.Fatalf("got %d items, want the fixture's 13", len(items))
	}

	first := items[0]
	if first.Title != "Ephemeral Architecture Prototypes: The Future of Public Spaces - ArchDaily" {
		t.Errorf("title = %q", first.Title)
	}
	if first.Source != "ArchDaily" {
		t.Errorf("source = %q, want the publisher from <source>", first.Source)
	}
	if !strings.HasPrefix(first.URL, "https://news.google.com/rss/articles/") {
		t.Errorf("url = %q", first.URL)
	}
	if first.At.IsZero() {
		t.Error("pubDate did not parse")
	}
	if first.At.Year() != 2026 {
		t.Errorf("pubDate year = %d", first.At.Year())
	}
}

// One row per DECISION the parser makes about imperfect input. The class of
// arbitrary malformed bytes belongs to FuzzParseRSS, not here.
func TestParseRSSDecisions(t *testing.T) {
	wrap := func(items string) []byte {
		return []byte(`<?xml version="1.0"?><rss version="2.0"><channel>` + items + `</channel></rss>`)
	}
	for _, tc := range []struct {
		name  string
		in    []byte
		check func(t *testing.T, items []store.NewsItem, err error)
	}{
		{
			"an item with no pubDate parses with a zero time, not an error",
			wrap(`<item><title>T</title><link>u</link></item>`),
			func(t *testing.T, items []store.NewsItem, err error) {
				if err != nil || len(items) != 1 {
					t.Fatalf("got %d items, err %v", len(items), err)
				}
				if !items[0].At.IsZero() {
					t.Errorf("At = %v, want zero", items[0].At)
				}
			},
		},
		{
			"an unparseable pubDate does not fail the whole feed",
			wrap(`<item><title>A</title><pubDate>not a date</pubDate></item><item><title>B</title></item>`),
			func(t *testing.T, items []store.NewsItem, err error) {
				if err != nil || len(items) != 2 {
					t.Fatalf("got %d items, err %v — one bad date must not lose the feed", len(items), err)
				}
			},
		},
		{
			"CDATA in a title is unwrapped",
			wrap(`<item><title><![CDATA[Ephemeral & odd]]></title></item>`),
			func(t *testing.T, items []store.NewsItem, err error) {
				if err != nil || len(items) != 1 {
					t.Fatalf("got %d items, err %v", len(items), err)
				}
				if items[0].Title != "Ephemeral & odd" {
					t.Errorf("title = %q, want the unwrapped text", items[0].Title)
				}
			},
		},
		{
			"a feed with no items is empty, not an error",
			wrap(``),
			func(t *testing.T, items []store.NewsItem, err error) {
				if err != nil || len(items) != 0 {
					t.Fatalf("got %d items, err %v", len(items), err)
				}
			},
		},
		{
			"bytes that are not XML at all are an error",
			[]byte("not xml"),
			func(t *testing.T, items []store.NewsItem, err error) {
				if err == nil {
					t.Error("want an error")
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items, err := parseRSS(tc.in)
			tc.check(t, items, err)
		})
	}
}

// The malformed class, which a table is blind to by construction.
//
// The second property is the one that matters: a parser that INVENTS content is
// worse than one that finds none, because the invented sentence reaches the
// learner as if the feed had said it.
func FuzzParseRSS(f *testing.F) {
	if b, err := os.ReadFile(filepath.Join("testdata", "news", "ephemeral.rss")); err == nil {
		f.Add(string(b))
	}
	for _, s := range []string{
		"", "<rss>", "<?xml version=\"1.0\"?><rss><channel><item><title>x</title></item></channel></rss>",
		"<rss><channel><item><title><![CDATA[y]]></title></item></channel></rss>",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data string) {
		items, err := parseRSS([]byte(data))
		if err != nil {
			return
		}
		for i, it := range items {
			if it.Title == "" {
				continue
			}
			// SUBSEQUENCE, not substring — and the fuzzer taught me that in two
			// seconds. `<title>0<![CDATA[0]]></title>` is legitimate mixed
			// content that XML concatenates to "00", which appears nowhere in the
			// input contiguously. A substring property calls correct parsing an
			// invention; entity decoding (`&amp;` -> `&`) breaks it too.
			//
			// Subsequence still catches what this is for: a parser that emits a
			// character the input never contained, or emits them out of order.
			// It is the same claim Render's no-data-loss invariant makes about
			// rendered text, so it reuses that helper rather than growing a
			// second one (invariant_test.go).
			if gap := subsequenceGap([]rune(it.Title), []rune(data)); gap >= 0 {
				t.Fatalf("item %d title %q is not an ordered subsequence of the input (first stray rune at %d) — the parser invented content",
					i, it.Title, gap)
			}
		}
	})
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
			"an unreadable pubDate is the ZERO time, never a guess",
			wrap(`<item><title>A</title><pubDate>not a date</pubDate></item>`),
			func(t *testing.T, items []store.NewsItem, err error) {
				if err != nil || len(items) != 1 {
					t.Fatalf("got %d items, err %v", len(items), err)
				}
				// The guard the first version of this suite had — skipping when
				// At is zero — skipped exactly the case it claimed to pin, and a
				// parsePubDate that returned a fixed date survived the whole
				// suite because of it.
				if !items[0].At.IsZero() {
					t.Errorf("At = %v, want the zero time — a guessed date on a real sentence is worse than none", items[0].At)
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
// THE RULE, arrived at after three wrong properties on this one target: a fuzz
// property may assert only what THIS CODE's contract guarantees over arbitrary
// input. It may not assert a property of the input's textual form, because the
// decoder is entitled to transform it.
//
// The three that broke, each against entirely correct parsing:
//
//  1. "a title is a SUBSTRING of the input" — mixed content concatenates:
//     `0<![CDATA[0]]>` is legitimately `00`.
//  2. "a title is a SUBSEQUENCE of the input" — entities decode: `&#65;` is `A`,
//     `&#39;` is `'` (which is what Google News emits for apostrophes), and a
//     lone CR becomes LF under XML line-ending normalisation.
//  3. "there are no more items than `<item` tags" — encoding/xml matches by
//     LOCAL name, so `<x:item>` is an item that the textual count cannot see.
//     The committed fixture already declares a namespace prefix, so a two-byte
//     insertion reaches it.
//
// Every one of those was a claim about the bytes rather than about us. What IS
// ours: that an error comes with no items, and that parsePubDate never guesses —
// the latter now has its own target below, on the string alone, where no XML
// decoding stands between the property and the code it describes.
func FuzzParseRSS(f *testing.F) {
	if b, err := os.ReadFile(filepath.Join("testdata", "news", "ephemeral.rss")); err == nil {
		f.Add(string(b))
	}
	for _, s := range []string{
		"", "<rss>", "<?xml version=\"1.0\"?><rss><channel><item><title>x</title></item></channel></rss>",
		"<rss><channel><item><title><![CDATA[y]]></title></item></channel></rss>",
		// The shapes that refuted the three earlier properties, seeded so a
		// future attempt to reinstate any of them fails here rather than in the
		// wild.
		"<rss><channel><item><title>&#65;</title></item></channel></rss>",
		"<rss><channel><item><title>&#39;q&#39;</title></item></channel></rss>",
		"<rss><channel><item><title>a\rb</title></item></channel></rss>",
		"<rss><channel><x:item><title>t</title></x:item></channel></rss>",
		"<rss><channel><item><pubDate>Mon, 24 Aug 1026 08:00:00 GMT</pubDate></item></channel></rss>",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data string) {
		items, err := parseRSS([]byte(data))
		if err != nil && items != nil {
			// Our contract: a failed parse yields nothing, so a caller cannot
			// half-use a broken feed.
			t.Fatalf("returned %d items alongside an error %v", len(items), err)
		}
	})
}

// parsePubDate is OUR logic, so it gets a property with no decoder in between.
//
// The contract in rss.go is exact: an unreadable date is the zero time, never a
// guess. A guess is worse than nothing here — it would put a made-up date on a
// real sentence, and #10 sorts by recency.
func FuzzParsePubDate(f *testing.F) {
	for _, s := range []string{
		"", "not a date", "Mon, 24 Aug 2026 08:00:00 GMT", "Mon, 24 Aug 1026 08:00:00 GMT",
		"2026-08-24T08:00:00Z", "24 Aug 26 08:00 GMT", "Mon 24 Aug",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := parsePubDate(s)
		if got.IsZero() {
			return
		}
		// A non-zero result must be one this string genuinely spells: re-parsing
		// its own canonical form must land on the same instant. That catches a
		// guess without asserting anything about the string's shape.
		for _, layout := range pubDateFormats {
			if again, err := time.Parse(layout, s); err == nil {
				if !again.Equal(got) {
					t.Fatalf("parsePubDate(%q) = %v, but %s parses it as %v", s, got, layout, again)
				}
				return
			}
		}
		t.Fatalf("parsePubDate(%q) returned %v, but no known layout parses that string — the date was invented", s, got)
	})
}

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
// The property is the Done-when's own words — "never returns an item it did not
// find in the input" — expressed as a COUNT. There cannot be more items than
// there are item tags, and that bound holds no matter what the decoder does to
// the characters inside them.
//
// It is the third property this target has carried, and the first two were both
// wrong in the same direction: too strong, failing on correct parsing.
// "Substring" died to mixed content (`0<![CDATA[0]]>` legitimately concatenates
// to `00`). "Subsequence" survived that but dies to entity decoding — `&#65;`
// yields "A", `&#39;` yields "'", and a lone `\r` yields "\n" under XML
// line-ending normalisation, none of which are subsequences of their input. That
// last one matters practically: `&#39;` is exactly what Google News emits for
// apostrophes, so re-capturing the fixture would have turned this red against
// entirely correct code.
//
// The lesson recorded, because the hazard is what happens NEXT: a property that
// fails on correct input invites weakening the thing it was defending. Character
// provenance inside an item is `encoding/xml`'s contract, not ours; what is ours
// is how many items we report and what we do with a date we cannot read, and
// both are pinned here.
func FuzzParseRSS(f *testing.F) {
	if b, err := os.ReadFile(filepath.Join("testdata", "news", "ephemeral.rss")); err == nil {
		f.Add(string(b))
	}
	for _, s := range []string{
		"", "<rss>", "<?xml version=\"1.0\"?><rss><channel><item><title>x</title></item></channel></rss>",
		"<rss><channel><item><title><![CDATA[y]]></title></item></channel></rss>",
		// The three shapes that refuted the previous property. Seeded so a future
		// weakening of it fails here rather than in the wild.
		"<rss><channel><item><title>&#65;</title></item></channel></rss>",
		"<rss><channel><item><title>&#39;q&#39;</title></item></channel></rss>",
		"<rss><channel><item><title>a\rb</title></item></channel></rss>",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data string) {
		items, err := parseRSS([]byte(data))
		if err != nil {
			if items != nil {
				t.Fatalf("returned %d items alongside an error", len(items))
			}
			return
		}
		// The Done-when, as a bound: an item requires an item tag.
		if tags := strings.Count(data, "<item"); len(items) > tags {
			t.Fatalf("returned %d items from input holding %d item tags — the parser invented items",
				len(items), tags)
		}
		for i, it := range items {
			// A date we could not read is the zero time, never a guess. This is
			// OUR logic rather than the decoder's, which is why it is pinned.
			if !it.At.IsZero() && it.At.Year() < 1900 {
				t.Fatalf("item %d has an implausible parsed date %v", i, it.At)
			}
		}
	})
}

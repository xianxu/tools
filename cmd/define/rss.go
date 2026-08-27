package main

import (
	"encoding/xml"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// parseRSS turns an RSS 2.0 body into items. The only XML knowledge in the
// package, and pure over bytes — so the malformed-feed requirement is a table
// and a fuzz target with no network anywhere near it.
//
// It returns OUR type rather than encoding/xml's, which is what would let a
// second feed format land later without every caller learning about it.
func parseRSS(data []byte) ([]store.NewsItem, error) {
	var doc struct {
		Items []struct {
			Title   string `xml:"title"`
			Link    string `xml:"link"`
			PubDate string `xml:"pubDate"`
			Source  string `xml:"source"`
		} `xml:"channel>item"`
	}
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	out := make([]store.NewsItem, 0, len(doc.Items))
	for _, it := range doc.Items {
		out = append(out, store.NewsItem{
			Title:  it.Title,
			URL:    it.Link,
			Source: it.Source,
			At:     parsePubDate(it.PubDate),
		})
	}
	return out, nil
}

// pubDateFormats are the spellings observed in the wild. RFC1123Z first: it is
// what Google News emits.
var pubDateFormats = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC822,
	time.RFC3339,
}

// parsePubDate returns the zero time for a date it cannot read, rather than an
// error.
//
// A feed with one bad date is still a feed, and failing the whole fetch over it
// would trade ninety-nine usable sentences for strictness nobody asked for. The
// zero value is honest — "we do not know when" — and a consumer that cares can
// test IsZero.
func parsePubDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, f := range pubDateFormats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

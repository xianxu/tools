package main

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// containsWord delegates to highlightSpans, so these rows are the properties
// that delegation is supposed to INHERIT. Each would need its own code if the
// matcher were reimplemented here, and each is a place the two could drift.
func TestContainsWord(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		word string
		want bool
	}{
		{"exact", "Springtime is ephemeral today", "ephemeral", true},
		{"case-insensitive", "Ephemeral Architecture Prototypes", "ephemeral", true},
		{"quoted", "Snapchat Hooks Kids With 'Ephemeral' Posts", "ephemeral", true},
		{"parenthesised", "a note (ephemeral) here", "ephemeral", true},
		{"trailing punctuation", "it was ephemeral.", "ephemeral", true},
		// The operator's call on #21, inherited: exact only. A false positive
		// puts a sentence about a DIFFERENT word in front of the learner.
		{"inflection does not match", "it faded ephemerally", "ephemeral", false},
		{"longer word does not match", "the preephemeral phase", "ephemeral", false},
		{"absent", "The Hidden Litigation Risk of Disappearing Messages", "ephemeral", false},
		// Multi-word headwords are real lookups, so they must match as phrases.
		{"phrase", "I ate a hot dog", "hot dog", true},
		{"phrase not across punctuation", "too hot, dog days", "hot dog", false},
		{"hyphenated word", "a hot-dog stand", "hot-dog", true},
		{"empty text", "", "ephemeral", false},
		{"empty word", "anything", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := containsWord(tc.text, tc.word); got != tc.want {
				t.Errorf("containsWord(%q, %q) = %v, want %v", tc.text, tc.word, got, tc.want)
			}
		})
	}
}

// The property that makes reusing the matcher worth doing: the filter and the
// highlighter must never disagree about the same text.
//
// If they did, the learner would see a headline whose word renders green while
// the filter rejected that headline — the same word treated as known and unknown
// on one screen. This is the pin on that, and it is why containsWord delegates
// rather than reimplementing.
func TestContainsWordAgreesWithTheHighlighter(t *testing.T) {
	texts := []string{
		"Springtime is ephemeral", "it faded ephemerally", "the preephemeral phase",
		"Snapchat Hooks Kids With 'Ephemeral' Posts", "I ate a hot dog", "too hot, dog days",
		"a hot-dog stand", "nothing here at all", "Ephemeral, briefly",
	}
	for _, word := range []string{"ephemeral", "hot dog", "hot-dog"} {
		v := vocab(word)
		for _, text := range texts {
			filtered := containsWord(text, word)
			highlighted := strings.Contains(
				string(renderSpans(highlightSpans(text, v))), "[")
			if filtered != highlighted {
				t.Errorf("disagreement on %q for %q: containsWord=%v, highlighter=%v",
					text, word, filtered, highlighted)
			}
		}
	}
}

// renderSpans marks known spans so the highlighter's answer is comparable.
func renderSpans(spans []span) []byte {
	var b strings.Builder
	for _, s := range spans {
		if s.known {
			b.WriteString("[" + s.text + "]")
			continue
		}
		b.WriteString(s.text)
	}
	return []byte(b.String())
}

func TestUsagesFromTheCapturedFeed(t *testing.T) {
	items, err := parseRSS(rssFixture(t, "ephemeral.rss"))
	if err != nil {
		t.Fatal(err)
	}

	got := usagesFrom(items, "ephemeral")

	// The fixture holds 13 items, 10 of which contain the word. Asserting the
	// COUNT rather than a range here is right because this fixture is committed
	// and its composition is known — the range in the Spec (12-99) describes the
	// live feed, not a frozen artifact.
	if len(got) != 10 {
		t.Errorf("got %d usages from 13 items, want the 10 that contain the word", len(got))
	}
	for _, u := range got {
		if u.Source != usageNews {
			t.Errorf("usage %q tagged %q, want %q", u.Text, u.Source, usageNews)
		}
		if !containsWord(u.Text, "ephemeral") {
			t.Errorf("a usage that does not contain the word survived the filter: %q", u.Text)
		}
	}
}

// The publisher suffix is attribution, not part of the sentence. Discovered by
// capturing a real feed rather than by reasoning about one: every Google News
// title ends " - Publisher", and leaving it on would end every sentence the
// learner reads with an unrelated proper noun.
func TestUsageTextDropsThePublisherSuffix(t *testing.T) {
	items := []store.NewsItem{{
		Title:  "Springtime is ephemeral - The University of Chicago Magazine",
		Source: "The University of Chicago Magazine",
	}}

	got := usagesFrom(items, "ephemeral")

	if len(got) != 1 {
		t.Fatalf("got %d usages", len(got))
	}
	if got[0].Text != "Springtime is ephemeral" {
		t.Errorf("Text = %q, want the headline without its attribution", got[0].Text)
	}
	// The publisher is not lost — it moves to the field that means publisher.
	if got[0].Publisher != "The University of Chicago Magazine" {
		t.Errorf("Publisher = %q", got[0].Publisher)
	}
}

// A feed item with no publisher keeps its whole title. Zero coverage before
// this: every fixture named a publisher, so the guard that handles an absent one
// was never entered.
func TestNoPublisherLeavesTheTitleAlone(t *testing.T) {
	items := []store.NewsItem{{Title: "Springtime is ephemeral - Somewhere", Source: ""}}

	got := usagesFrom(items, "ephemeral")

	if len(got) != 1 {
		t.Fatalf("got %d usages", len(got))
	}
	if got[0].Text != "Springtime is ephemeral - Somewhere" {
		t.Errorf("Text = %q, want the title intact — nothing names an attribution to strip", got[0].Text)
	}
}

// A dash that is NOT attribution stays put: the rule is "strip the publisher the
// feed named", not "strip anything after the last dash".
//
// The fixture has to tell those two apart, and the first version did not — with
// the title ending in the publisher, both rules cut at the same place and a
// last-dash mutant survived. Here the title carries NO attribution suffix, so
// the correct rule keeps the whole headline and the last-dash rule truncates it.
func TestUsageKeepsADashThatIsNotAttribution(t *testing.T) {
	items := []store.NewsItem{{
		Title:  "Ephemeral - a study in impermanence",
		Source: "ArchDaily", // named, but NOT appended to this title
	}}

	got := usagesFrom(items, "ephemeral")

	if len(got) != 1 {
		t.Fatalf("got %d usages", len(got))
	}
	if got[0].Text != "Ephemeral - a study in impermanence" {
		t.Errorf("Text = %q, want the headline intact — nothing here is attribution", got[0].Text)
	}
}

// And the ordinary case, where the suffix IS the publisher, still strips.
func TestUsageStripsAttributionAfterAnInternalDash(t *testing.T) {
	items := []store.NewsItem{{
		Title:  "Ephemeral - a study in impermanence - ArchDaily",
		Source: "ArchDaily",
	}}

	got := usagesFrom(items, "ephemeral")

	if len(got) != 1 || got[0].Text != "Ephemeral - a study in impermanence" {
		t.Errorf("Text = %q, want only the trailing attribution removed", got[0].Text)
	}
}

func TestEntryUsagesLiftsNOADExamples(t *testing.T) {
	e := ParseEntry(fixture(t, "bank"))

	got := entryUsages(e, "bank")

	if len(got) == 0 {
		t.Fatal("no usages from an entry with examples — this test would assert nothing")
	}
	for _, u := range got {
		if u.Source != usageNOAD {
			t.Errorf("usage %q tagged %q, want %q", u.Text, u.Source, usageNOAD)
		}
		if !containsWord(u.Text, "bank") {
			t.Errorf("an example that does not contain the word survived: %q", u.Text)
		}
	}
}

// BR-1: the ORDER of strip-then-filter is load-bearing and nothing pinned it.
//
// A headline whose only occurrence of the word is inside the PUBLISHER name is
// the discriminating case. Strip first and it is correctly dropped; filter first
// and it survives with text that does not contain the word — a sentence handed to
// the learner as an example of a word it does not use.
//
// TestUsagesFromTheCapturedFeed already asserts every returned usage contains the
// word, but the captured feed holds no such headline, so that assertion never ran
// over the state where the two orders differ. The assertion was right; the input
// never reached it.
func TestAWordOnlyInThePublisherIsNotAUsage(t *testing.T) {
	items := []store.NewsItem{{
		Title:  "Weekly roundup of local business - Ephemeral Times",
		Source: "Ephemeral Times",
	}}

	got := usagesFrom(items, "ephemeral")

	if len(got) != 0 {
		t.Errorf("got %d usages, want none — the word appears only in the publisher: %q",
			len(got), got[0].Text)
	}
}

// The invariant behind it, asserted directly rather than only as a side effect:
// every usage's text contains the word it was collected for.
func TestEveryUsageContainsItsWord(t *testing.T) {
	items := []store.NewsItem{
		{Title: "Springtime is ephemeral - UChicago", Source: "UChicago"},
		{Title: "Weekly roundup - Ephemeral Times", Source: "Ephemeral Times"},
		{Title: "Unrelated headline - ArchDaily", Source: "ArchDaily"},
	}

	got := usagesFrom(items, "ephemeral")

	if len(got) == 0 {
		t.Fatal("no usages at all — this test would assert nothing")
	}
	for _, u := range got {
		if !containsWord(u.Text, "ephemeral") {
			t.Errorf("usage %q does not contain the word it was collected for", u.Text)
		}
	}
}

// BR-11's enumeration, swept: every field that crosses from a NewsItem into a
// Usage needs one assertion that goes red when it is blanked or invented.
//
// Three were unpinned — Usage.URL, Usage.At, and NewsItem.At on an unreadable
// date — and "the struct is obviously copied" is exactly the reasoning that lets
// a field quietly stop being copied. A table over the fields makes a NEW field
// conspicuous by its absence.
func TestEveryUsageFieldCarriesThrough(t *testing.T) {
	at := aDay.Add(3 * time.Hour)
	item := store.NewsItem{
		Title:  "Springtime is ephemeral - UChicago",
		URL:    "https://news.google.com/rss/articles/abc",
		Source: "UChicago",
		At:     at,
	}

	got := usagesFrom([]store.NewsItem{item}, "ephemeral")
	if len(got) != 1 {
		t.Fatalf("got %d usages", len(got))
	}
	u := got[0]

	for _, f := range []struct {
		field string
		got   any
		want  any
	}{
		{"Text", u.Text, "Springtime is ephemeral"},
		{"Source", u.Source, usageNews},
		{"Publisher", u.Publisher, "UChicago"},
		{"URL", u.URL, "https://news.google.com/rss/articles/abc"},
		{"At", u.At, at},
	} {
		if f.got != f.want {
			t.Errorf("Usage.%s = %v, want %v", f.field, f.got, f.want)
		}
	}
}

// A NOAD usage has no publisher, URL or date, and that is a CONTRACT rather than
// an accident: inventing any of them would attribute a dictionary example to a
// news outlet.
func TestNOADUsageCarriesNoNewsMetadata(t *testing.T) {
	e := ParseEntry(fixture(t, "bank"))

	got := entryUsages(e, "bank")
	if len(got) == 0 {
		t.Fatal("no usages — this test would assert nothing")
	}

	for _, u := range got {
		if u.Publisher != "" || u.URL != "" || !u.At.IsZero() {
			t.Errorf("a dictionary usage carries news metadata: %+v", u)
		}
	}
}

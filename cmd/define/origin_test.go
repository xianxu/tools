package main

import (
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// OriginLanguage over the WHOLE committed corpus, one expected outcome per
// fixture (#35).
//
// A curated dozen pinned three of the eight declining entries whose ORIGIN
// carries a language token the rule must suppress — and missed the one nobody
// would pick. `run` reads "Old English rinnan, irnan (verb), of Germanic origin,
// probably reinforced in Middle English by Old Norse rinna, renna": it has NO
// cognate marker, so the D0 cut never fires, and it declines only because
// `Germanic` is masked before any modern name is searched for. Unpinned, its
// failure mode is `run` inferring German — #29's D1 by another road.
//
// Ranging over the corpus also forces a decision when a fixture is added, which
// a hand-picked list cannot.
func TestOriginLanguageOverTheCorpus(t *testing.T) {
	// Every English fixture, and why it lands where it does. The comment is the
	// point for the declines: several name a language that is NOT a source.
	want := map[string]store.Lang{
		"jalapeño": "es", // from Mexican Spanish — a qualified name still matches
		"mesa":     "es", // Spanish … from Latin: a chain, first-named wins
		"concrete": "fr", // from French concret or Latin concretus
		"parrot":   "fr", // probably from dialect French perrot — hedge + modifier

		// Declines, and the reason differs:
		"a priori":     "", // Latin only
		"alewife":      "", // no language named
		"amazon":       "", // prose, no source language
		"bank":         "", // cut at "related to bench" — NOT the Germanic mask
		"bargainer":    "", // Old French, masked
		"bases":        "",
		"complete":     "", // Old French or Latin
		"content":      "", // via Old French from Latin — search-before-mask says French
		"defenestrate": "", // "see defenestration"
		"desert":       "", // Old French, late Latin
		"ephemeral":    "", // Greek — ancient by NOAD's convention (D3)
		"even":         "", // cognate: "related to Dutch even" (D0)
		"gaslighting":  "", // "see gaslight (verb)" — no language at all
		"hot dog":      "", // US college slang
		"ipad":         "",
		"iphone":       "",
		"macbook":      "",
		"man":          "", // cognate clause
		"minute":       "", // via Old French from late Latin
		"present":      "", // via Old French from Latin
		"pulp":         "", // Latin
		"quokka":       "", // Nyungar — a real language, absent from the map: decline
		"read":         "", // cognate: "related to Dutch raden and German raten" (D0)
		"record":       "", // Old French
		"run":          "", // NO cognate marker; declines via the Germanic mask alone
		"set":          "",
		"subject":      "", // Old French, Latin
		"sycophantic":  "",
		"thing":        "",
		"use":          "",
	}

	d := testDict(t)
	if len(d.entries) == 0 {
		t.Fatal("empty corpus; this test would be vacuous")
	}
	for word, raw := range d.entries {
		exp, listed := want[word]
		if !listed {
			t.Errorf("%s is in the corpus but has no expected outcome here — a new fixture "+
				"must be decided, not defaulted. Add a row with its ORIGIN as the comment.", word)
			continue
		}
		t.Run(word, func(t *testing.T) {
			got, named, err := OriginLanguage(ParseEntry(raw))
			if exp == "" {
				if err == nil {
					t.Errorf("inferred %q (%s) — this entry names no SOURCE language; "+
						"the token it carries is a cognate, a historical stage, or a family",
						got, named)
				}
				return
			}
			if err != nil {
				t.Errorf("declined (%v), want %q", err, exp)
				return
			}
			if got != exp {
				t.Errorf("= %q (from %q), want %q", got, named, exp)
			}
			if named == "" {
				t.Error("returned no NAME — the report says what was READ, not what was derived")
			}
		})
	}
}

// The shapes that decide the rule, stated as text rather than fixtures so the
// reasoning is readable without opening the corpus.
func TestOriginLanguageRules(t *testing.T) {
	for _, tc := range []struct {
		name, origin string
		want         store.Lang
	}{
		{"a bare language", "French, from arrondir 'make round'.", "fr"},
		{"a date prefix", "1970s: from Japanese, literally 'empty orchestra'.", "ja"},
		{"a qualified name", "from Mexican Spanish (chile) jalapeño.", "es"},
		{"a chain: first-named is the immediate source", "from French, from Italian balletto.", "fr"},
		{"via X from Y is still a chain", "mid 17th century: via French from Russian knut.", "fr"},

		{"a cognate clause names no source", "Old English bringan, of Germanic origin; related to Dutch brengen and German bringen.", ""},
		{"compare with is also a cognate marker", "from French, ok. Compare with Spanish casco.", "fr"},
		{"a stage is masked before the search", "via Old French from Latin natio(n-).", ""},
		{"bare Greek is ancient", "from Greek ephemeros.", ""},
		{"Germanic is a family, not a language", "Old English rinnan, of Germanic origin, reinforced by Old Norse.", ""},
		// THE rule that actually protects the case above, pinned on its own so a
		// change to the matching rule reddens here rather than silently making
		// `run` infer German. The mask list is redundancy; this is the guard.
		{"German does not match inside Germanic", "of Germanic origin.", ""},
		{"and the boundary is not doing it by accident", "from German Schadenfreude.", "de"},
		{"no ORIGIN content at all", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := Entry{Sections: []Section{{Name: "ORIGIN", Text: tc.origin}}}
			got, _, err := OriginLanguage(e)
			if tc.want == "" {
				if err == nil {
					t.Errorf("inferred %q, want a decline", got)
				}
				return
			}
			if err != nil {
				t.Errorf("declined (%v), want %q", err, tc.want)
			} else if got != tc.want {
				t.Errorf("= %q, want %q", got, tc.want)
			}
		})
	}
}

// It reads dictionary prose, so it must survive whatever prose is.
func FuzzOriginLanguageDoesNotPanic(f *testing.F) {
	f.Add("from French, from Latin")
	f.Add("")
	f.Add(strings.Repeat("related to Dutch ", 700))
	f.Add("\xff\xfe not utf8")
	f.Fuzz(func(t *testing.T, origin string) {
		_, _, _ = OriginLanguage(Entry{Sections: []Section{{Name: "ORIGIN", Text: origin}}})
	})
}

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
		// THE word-boundary guard, on a case the MASK CANNOT SAVE.
		//
		// "of Germanic origin" was the first attempt and it pinned nothing: the
		// `Germanic` mask holds it green, so the two mechanisms were mutually
		// redundant and each hid the other's removal. The mutation that "proved"
		// the boundary changed BOTH at once and the redness was attributed to
		// one — an invalid result of exactly the kind workshop/lessons.md records.
		//
		// "Germany" is in no mask list, so only \b stops it matching German.
		{"a country is not a language", "named after a town in Germany.", ""},
		{"nor is a person", "named for the Frenchman Germaine.", ""},
		{"and the boundary is not refusing everything", "from German Schadenfreude.", "de"},

		// The derived stage mask (D4 as a CATEGORY): every mapped language
		// generates its own stages, so these four leaked while Old French did not.
		{"Old Italian is a stage", "from Old Italian mezzo.", ""},
		{"Middle French is a stage", "from Middle French bureau.", ""},
		{"Old Spanish is a stage", "from Old Spanish casco.", ""},
		{"Low German is a stage", "from Low German bugseren.", ""},
		{"Old High German is masked whole", "from Old High German hus.", ""},
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

// Offsets index the SOURCE text, which is what makes a mention addressable on
// screen (#30 M2.1).
//
// It was nearly wrong in a way no existing test could see. The stage mask used
// to be `strings.ReplaceAll(text, stage, " ")` — a length CHANGE — so every
// offset after a masked stage pointed at the wrong column.
//
// MEASURED, because the plan named the wrong entry for it: `concrete` does not
// shift at all. "Middle English" is not a masked stage — there is no `English`
// row to derive one from — and its only masked stage, "Latin", comes AFTER the
// language. `ballet` is the real shape: "from Old French ballet, from Italian
// balletto", where masking the 10-character "Old French" to a single space moves
// "Italian" 9 columns left, onto "et, fro". A region drawn there lands on the
// wrong word, and clicking it plays something the user did not point at.
func TestOriginMentionOffsetsIndexTheSourceText(t *testing.T) {
	for _, tc := range []struct {
		name, origin string
		want         []string // the languages, in source order
	}{
		{
			// The measured case: a masked stage BEFORE the language, which is the
			// only arrangement the shift can show up in.
			"a stage before the language", "early 17th century: from Old French ballet, from Italian balletto.",
			[]string{"Italian"},
		},
		{
			// Two masked stages, so a length change compounds.
			"two stages before the language", "mid 16th century: from Old Norse and Old French, from Italian arsenale.",
			[]string{"Italian"},
		},
		{
			// And the arrangement that does NOT shift, so each case says which half
			// of the rule it exercises.
			"a stage after the language", "late Middle English: from French concret or Latin concretus.",
			[]string{"French"},
		},
		{
			// The reason this producer exists: a click has nothing to pick, so
			// both must be addressable.
			"two languages, both clickable", "either from French, or an abbreviation of pianoforte; Italian piano is not attested until later.",
			[]string{"French", "Italian"},
		},
		{"cognates are cut", "Old English bringan, of Germanic origin; related to Dutch brengen and German bringen.", nil},
		{"a stage is not its modern language", "from Old French concret.", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := Entry{Sections: []Section{{Name: "ORIGIN", Text: tc.origin}}}
			got, _ := OriginLanguageMentions(e)

			if len(got) != len(tc.want) {
				t.Fatalf("found %v, want %v", names(got), tc.want)
			}
			for i, m := range got {
				if m.Name != tc.want[i] {
					t.Errorf("mention %d is %q, want %q — source order is what a reader clicks", i, m.Name, tc.want[i])
				}
				// THE ASSERTION THAT MATTERS: the offset points at the language
				// in the text a region will be drawn over.
				if end := m.Offset + len(m.Name); m.Offset < 0 || end > len(tc.origin) || tc.origin[m.Offset:end] != m.Name {
					at := ""
					if m.Offset >= 0 && m.Offset < len(tc.origin) {
						at = tc.origin[m.Offset:min(m.Offset+20, len(tc.origin))]
					}
					t.Errorf("%q is reported at offset %d, where the source reads %q — a region drawn there lands on the wrong word",
						m.Name, m.Offset, at)
				}
			}
		})
	}
}

func names(ms []Mention) []string {
	var out []string
	for _, m := range ms {
		out = append(out, m.Name)
	}
	return out
}

// The producer and its first-named consumer cannot disagree: OriginLanguage IS
// the first mention, so a rule that changes one changes both.
func TestOriginLanguageIsTheFirstMention(t *testing.T) {
	for _, origin := range []string{
		"either from French, or an abbreviation of pianoforte; Italian piano is not attested until later.",
		"early 17th century: from French, from Italian balletto.",
		"late Middle English: from French concret or Latin concretus.",
		"mid 17th century: from Italian, from Latin opera.",
	} {
		e := Entry{Sections: []Section{{Name: "ORIGIN", Text: origin}}}
		mentions, _ := OriginLanguageMentions(e)
		lang, named, err := OriginLanguage(e)
		if err != nil {
			t.Fatalf("%q: %v", origin, err)
		}
		if len(mentions) == 0 {
			t.Fatalf("%q: the producer found nothing while the consumer found %q", origin, named)
		}
		if named != mentions[0].Name || lang != mentions[0].Lang {
			t.Errorf("%q: OriginLanguage says %q/%q, the first mention is %q/%q",
				origin, named, lang, mentions[0].Name, mentions[0].Lang)
		}
	}
}

package main

import (
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// installedOnThisMachine is the metadata actually measured on 2026-08-28, not an
// invented fixture. The five that matter out of 87.
//
// It is the argument for the whole design: requiring every language entry to be
// L->L gives exactly ONE candidate for es and SIX for en — and two of the six are
// THESAURUSES while a third is an accessibility dictionary. Nothing in the
// metadata says "general-purpose dictionary", so no rule over it can prefer NOAD
// to a thesaurus. That is why a curated list decides and metadata only narrows.
func installedOnThisMachine() []dictMeta {
	// NOAD is LAST on purpose. With it first, a rule that ignored the curated
	// list would pick it by accident and the mutation that drops curation could
	// not redden anything — the test would be asserting fixture order.
	return []dictMeta{
		{ID: "com.apple.dictionary.OTE", Langs: []langPair{{Index: "en", Description: "en"}}},  // thesaurus
		{ID: "com.apple.dictionary.OAWT", Langs: []langPair{{Index: "en", Description: "en"}}}, // thesaurus
		{ID: "com.apple.accessibility.dictionary.TTY", Langs: []langPair{{Index: "en", Description: "en"}}},
		{ID: "com.apple.dictionary.NOAD", Langs: []langPair{{Index: "en", Description: "en"}}},
		{ID: "com.apple.dictionary.es.DGLEV", Langs: []langPair{{Index: "es", Description: "es"}}},
		{ID: "com.apple.dictionary.OxfordSpanish", Langs: []langPair{
			{Index: "es", Description: "es"},
			{Index: "en", Description: "es"}, // bilingual: this pair is what disqualifies it
		}},
	}
}

func TestChooseDictionary(t *testing.T) {
	for _, tc := range []struct {
		name string
		lang store.Lang
		want string
		ok   bool
	}{
		{
			name: "English prefers the curated general dictionary over two thesauruses",
			lang: "en", want: "com.apple.dictionary.NOAD", ok: true,
		},
		{
			name: "Spanish is the monolingual Larousse, not the bilingual Oxford",
			lang: "es", want: "com.apple.dictionary.es.DGLEV", ok: true,
		},
		{
			// Honest degradation, and the Done-when row it serves: a word absent
			// from the current language reports NO ENTRY rather than silently
			// answering from English.
			name: "a language nothing indexes is not found, never substituted",
			lang: "de", want: "", ok: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := chooseDictionary(installedOnThisMachine(), tc.lang)
			if ok != tc.ok || got.ID != tc.want {
				t.Errorf("chooseDictionary(%q) = (%q, %v), want (%q, %v)", tc.lang, got.ID, ok, tc.want, tc.ok)
			}
		})
	}
}

// The curated list is honest about BEING curated, and there are TWO ways to fall
// off it — an uncurated identifier, and an uncurated language. Both must defer
// to today's NULL behaviour rather than confidently picking wrong.
//
// The second row is the one that matters and the one a first draft omitted: a
// perfectly good monolingual French dictionary, installed, indexing exactly the
// language asked for, and still not chosen — because "indexes fr monolingually"
// does not distinguish a dictionary from a thesaurus, which is the whole reason
// curation exists. Without this row, a rule that guessed for uncurated languages
// survived every test.
func TestChooseDictionaryWithNoCuratedMatch(t *testing.T) {
	for _, tc := range []struct {
		name      string
		installed []dictMeta
		lang      store.Lang
	}{
		{
			name: "an uncurated identifier for a curated language",
			installed: []dictMeta{
				{ID: "org.example.SomeOtherEnglishDictionary", Langs: []langPair{{Index: "en", Description: "en"}}},
			},
			lang: "en",
		},
		{
			name: "an uncurated LANGUAGE, with a good candidate installed",
			installed: []dictMeta{
				{ID: "com.apple.dictionary.fr.Robert", Langs: []langPair{{Index: "fr", Description: "fr"}}},
			},
			lang: "fr",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := chooseDictionary(tc.installed, tc.lang); ok {
				t.Errorf("chooseDictionary(%q) picked %q; an uncurated choice must defer to NULL",
					tc.lang, got.ID)
			}
		})
	}
}

// The metadata check is not decoration behind the curated list: it must reject
// even the CURATED identifier if that dictionary stops being monolingual.
//
// The first version of this test used OxfordSpanish, which is not the curated id
// for Spanish — so it passed on the ID check and never reached the monolingual
// one. It asserted nothing and would have survived deleting the very rule it
// names. The fixture below is the curated id carrying a Spanish->ENGLISH pair,
// which is what an OS update could plausibly ship and what would put English
// glosses back into a Spanish session.
func TestChooseDictionaryRequiresEVERYPairToBeMonolingual(t *testing.T) {
	curatedButBilingual := []dictMeta{
		{ID: curated["es"], Langs: []langPair{
			{Index: "es", Description: "es"},
			{Index: "es", Description: "en"}, // headwords Spanish, definitions English
		}},
	}
	if got, ok := chooseDictionary(curatedButBilingual, "es"); ok {
		t.Errorf("accepted %q as monolingual Spanish though it defines in English", got.ID)
	}
}

// Order-independence, which is a REQUIREMENT rather than a nicety:
// DCSCopyAvailableDictionaries returns a CFSet, whose iteration order is
// unspecified, so the same machine can hand us these records in any sequence. A
// selection that depended on order would be right on Tuesday and wrong on
// Wednesday with nothing changed.
func TestChooseDictionaryDoesNotDependOnOrder(t *testing.T) {
	base := installedOnThisMachine()
	for _, lang := range []store.Lang{"en", "es"} {
		want, ok := chooseDictionary(base, lang)
		if !ok {
			t.Fatalf("%s: no choice from the measured set", lang)
		}
		for i := range base {
			shuffled := append([]dictMeta(nil), base...)
			// Every rotation, so no single ordering is privileged.
			shuffled = append(shuffled[i:], shuffled[:i]...)
			got, ok := chooseDictionary(shuffled, lang)
			if !ok || got.ID != want.ID {
				t.Errorf("%s: rotation %d = (%q, %v), want %q — the API returns a SET",
					lang, i, got.ID, ok, want.ID)
			}
		}
	}
}

// A dictionary that does not index the language at all is never a candidate,
// however curated it is — NOAD must not answer for Spanish.
func TestChooseDictionaryIgnoresDictionariesThatDoNotIndexTheLanguage(t *testing.T) {
	if got, ok := chooseDictionary(installedOnThisMachine(), "fr"); ok {
		t.Errorf("chooseDictionary(fr) = %q; nothing installed indexes French", got.ID)
	}
}

package main

import (
	"errors"
	"slices"
	"strings"
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
		{ID: "com.apple.dictionary.AppleDictionary", Langs: []langPair{{Index: "en", Description: "en"}}},
		{ID: "com.apple.dictionary.es.DGLEV", Langs: []langPair{{Index: "es", Description: "es"}}},
		{ID: "com.apple.dictionary.OxfordSpanish", Langs: []langPair{
			{Index: "es", Description: "es"},
			{Index: "en", Description: "es"}, // bilingual: this pair is what disqualifies it
		}},
	}
}

// ids is what the caller actually consumes: the chosen identifiers, in order.
func ids(ms []dictMeta) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.ID
	}
	return out
}

func TestChooseDictionary(t *testing.T) {
	for _, tc := range []struct {
		name string
		lang store.Lang
		want []string
		ok   bool
	}{
		{
			// In CURATED order, and both of them: NOAD answers ordinary words,
			// Apple Dictionary answers iPhone. Neither can leak another language
			// because both index en->en, which is what separates this from the
			// NULL search over every ACTIVE dictionary.
			name: "English takes both curated books, NOAD first, thesauruses never",
			lang: "en",
			want: []string{"com.apple.dictionary.NOAD", "com.apple.dictionary.AppleDictionary"},
			ok:   true,
		},
		{
			name: "Spanish is the monolingual Larousse, not the bilingual Oxford",
			lang: "es", want: []string{"com.apple.dictionary.es.DGLEV"}, ok: true,
		},
		{
			// Honest degradation, and the Done-when row it serves: a word absent
			// from the current language reports NO ENTRY rather than silently
			// answering from English.
			name: "a language nothing indexes is not found, never substituted",
			lang: "de", want: nil, ok: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := chooseDictionary(installedOnThisMachine(), tc.lang)
			if ok != tc.ok || !slices.Equal(ids(got), tc.want) {
				t.Errorf("chooseDictionary(%q) = (%v, %v), want (%v, %v)",
					tc.lang, ids(got), ok, tc.want, tc.ok)
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
				t.Errorf("chooseDictionary(%q) picked %v; an uncurated choice must defer to NULL",
					tc.lang, ids(got))
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
		{ID: curated["es"][0], Langs: []langPair{
			{Index: "es", Description: "es"},
			{Index: "es", Description: "en"}, // headwords Spanish, definitions English
		}},
	}
	if got, ok := chooseDictionary(curatedButBilingual, "es"); ok {
		t.Errorf("accepted %v as monolingual Spanish though it defines in English", ids(got))
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
			if !ok || !slices.Equal(ids(got), ids(want)) {
				t.Errorf("%s: rotation %d = (%v, %v), want %v — the API returns a SET, and "+
					"preference must come from the curated list rather than from position",
					lang, i, ids(got), ok, ids(want))
			}
		}
	}
}

// A dictionary that does not index the language at all is never a candidate,
// however curated it is — NOAD must not answer for Spanish.
func TestChooseDictionaryIgnoresDictionariesThatDoNotIndexTheLanguage(t *testing.T) {
	if got, ok := chooseDictionary(installedOnThisMachine(), "fr"); ok {
		t.Errorf("chooseDictionary(fr) = %v; nothing installed indexes French", ids(got))
	}
}

// The three outcomes of dictionary selection, including BOTH fallbacks — which
// had no automated test on any platform until the policy was extracted from the
// cgo shell.
//
// The Done-when row "the seam FALLS BACK to today's NULL behaviour if any symbol
// is missing" was ticked on a manual experiment: misspell a symbol, rebuild, run
// the binary. That proved it once, on one machine, and pinned nothing. These
// branches are not exotic — a shell context that reports only one installed
// dictionary takes the second one on EVERY run.
func TestDictionaryFor(t *testing.T) {
	for _, tc := range []struct {
		name          string
		installed     []dictMeta
		lang          store.Lang
		wantIDs       []string
		wantName      string
		wantComplaint bool
	}{
		{
			name: "the private surface did not resolve at all",
			// nil, NOT empty: a surface that is gone is a different thing from a
			// machine with no dictionaries, and the warnings differ.
			installed: nil, lang: "en",
			wantIDs: nil, wantName: everyActiveDictionary, wantComplaint: true,
		},
		{
			name:      "the surface works but nothing curated is installed",
			installed: []dictMeta{{ID: "org.example.Whatever", Langs: []langPair{{Index: "en", Description: "en"}}}},
			lang:      "en",
			wantIDs:   nil, wantName: everyActiveDictionary, wantComplaint: true,
		},
		{
			name:      "an empty set is the same degradation, not a panic",
			installed: []dictMeta{}, lang: "en",
			wantIDs: nil, wantName: everyActiveDictionary, wantComplaint: true,
		},
		{
			name:      "the happy path is SILENT, and /lang reports the name",
			installed: installedOnThisMachine(), lang: "es",
			wantIDs:  []string{"com.apple.dictionary.es.DGLEV"},
			wantName: "com.apple.dictionary.es.DGLEV",
		},
		{
			name:      "English names both books in curated order",
			installed: installedOnThisMachine(), lang: "en",
			wantIDs:  []string{"com.apple.dictionary.NOAD", "com.apple.dictionary.AppleDictionary"},
			wantName: "com.apple.dictionary.NOAD, com.apple.dictionary.AppleDictionary",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ids, name, complaint := dictionaryFor(tc.installed, tc.lang)
			if !slices.Equal(ids, tc.wantIDs) {
				t.Errorf("ids = %v, want %v", ids, tc.wantIDs)
			}
			if name != tc.wantName {
				t.Errorf("name = %q, want %q", name, tc.wantName)
			}
			if (complaint != "") != tc.wantComplaint {
				t.Errorf("complaint = %q, want said=%v — a fallback must be LOUD and the "+
					"happy path silent", complaint, tc.wantComplaint)
			}
		})
	}
}

// The two fallbacks must be distinguishable in what they SAY, because they mean
// different things to whoever reads the warning: one is "your OS changed", the
// other is "install a dictionary".
func TestTheTwoFallbacksSayDifferentThings(t *testing.T) {
	_, _, gone := dictionaryFor(nil, "en")
	_, _, uncurated := dictionaryFor([]dictMeta{}, "en")
	if gone == uncurated {
		t.Errorf("both fallbacks say %q; a vanished API and an uninstalled dictionary are "+
			"different problems with different remedies", gone)
	}
}

// The rule whose first fix shipped INOPERATIVE, which is the reason this logic
// was extracted from the cgo shell: an ABSENCE never overwrites a real failure.
//
// "This word is not Spanish" is a correct answer. "The Spanish dictionary is
// gone" means the caller cannot vouch for that absence. The C side keeps the two
// statuses distinct precisely so this fold does not collapse them — and the
// first attempt at this rule set ErrNoEntry unconditionally on status 1, so a
// vanished primary followed by an ordinary miss still reported "no entry", with
// no test to notice.
func TestFoldLookupError(t *testing.T) {
	gone := foldLookupError(nil, lookupNoSuchDict, "com.apple.dictionary.NOAD")

	for _, tc := range []struct {
		name        string
		prev        error
		status      int
		wantNoEntry bool
	}{
		{name: "a plain miss is an absence", prev: nil, status: lookupNoEntry, wantNoEntry: true},
		{name: "a miss after a miss is still an absence", prev: ErrNoEntry, status: lookupNoEntry, wantNoEntry: true},
		{
			// THE regression. Status 3 then status 1 must not read as "this word
			// does not exist in English".
			name: "a miss must NOT overwrite a vanished dictionary",
			prev: gone, status: lookupNoEntry, wantNoEntry: false,
		},
		{
			name: "nor may it overwrite an internal failure",
			prev: ErrLookupFailed, status: lookupNoEntry, wantNoEntry: false,
		},
		{name: "a vanished dictionary after a miss IS reported", prev: ErrNoEntry, status: lookupNoSuchDict, wantNoEntry: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := foldLookupError(tc.prev, tc.status, "com.apple.dictionary.NOAD")
			if errors.Is(got, ErrNoEntry) != tc.wantNoEntry {
				t.Errorf("foldLookupError(%v, %d) = %v; ErrNoEntry=%v, want %v",
					tc.prev, tc.status, got, !tc.wantNoEntry, tc.wantNoEntry)
			}
		})
	}

	// And a vanished dictionary names WHICH one, so the warning is actionable.
	if !strings.Contains(gone.Error(), "com.apple.dictionary.NOAD") {
		t.Errorf("a missing dictionary error does not name it: %v", gone)
	}
}

// The flat cgo encoding, parsed on any platform — which is the point: these
// parsers claimed to be "testable without CoreServices" while sitting behind
// //go:build darwin, and GOOS=linux go vet failed on their own test.
func TestParseDictRecords(t *testing.T) {
	got := parseDictRecords("com.apple.dictionary.NOAD\ten_US>en_US,\n" +
		"com.apple.dictionary.OxfordSpanish\tes>es,en>es,\n" +
		"\n" + // a blank line is not a dictionary
		"no-tab-here\n")
	want := []dictMeta{
		{ID: "com.apple.dictionary.NOAD", Langs: []langPair{{Index: "en", Description: "en"}}},
		{ID: "com.apple.dictionary.OxfordSpanish", Langs: []langPair{
			{Index: "es", Description: "es"}, {Index: "en", Description: "es"},
		}},
	}
	if len(got) != len(want) {
		t.Fatalf("parseDictRecords gave %d records, want %d: %+v", len(got), len(want), got)
	}
	for i := range got {
		if got[i].ID != want[i].ID || !slices.Equal(got[i].Langs, want[i].Langs) {
			t.Errorf("record %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	// The bilingual one must survive parsing WITH both pairs, or chooseDictionary
	// would accept it as monolingual Spanish.
	if _, ok := chooseDictionary(got, "es"); ok {
		t.Error("a bilingual dictionary parsed as an acceptable Spanish choice")
	}
}

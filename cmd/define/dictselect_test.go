package main

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
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

// capture.sh must capture through the SAME dictionaries production selects.
//
// This already diverged once: English fixtures were captured through the NULL
// search while production selected curated identifiers, and they agreed only
// because this host's active set happened to match — until it did not, and
// TestFixturesMatchLiveDictionary went red. The fix pointed capture.sh at the
// curated ids, but by HAND-RESTATING them, which is the same divergence waiting
// to happen again (ARCH-MOCK: the corpus must model what the seam actually does).
//
// A shell script cannot import a Go map, so the two copies are COMPARED rather
// than unified — the same move as the C-preamble symbol guard.
func TestCaptureScriptUsesTheCuratedDictionaries(t *testing.T) {
	b, err := os.ReadFile("testdata/capture.sh")
	if err != nil {
		t.Fatalf("reading capture.sh: %v", err)
	}
	script := string(b)

	inScript := map[string]bool{}
	// The HYPHEN is in the class because Apple's identifiers use it —
	// it.Devoto-Oli, nl-en.oup, zh_TW-en.DrEye. Without it this regex truncated
	// "com.apple.dictionary.it.Devoto-Oli" to "…it.Devoto" and then reported the
	// pair as mismatched in BOTH directions: the curated id looked absent from
	// the script, and the script's id looked uncurated. Every curated identifier
	// happened to be hyphen-free until #31, so the defect was latent rather than
	// absent.
	for _, m := range regexp.MustCompile(`com\.apple\.[A-Za-z0-9._-]+`).FindAllString(script, -1) {
		inScript[m] = true
	}
	if len(inScript) == 0 {
		t.Fatal("capture.sh names no dictionary identifiers; this test would pass vacuously")
	}

	for lang, want := range curated {
		for _, id := range want {
			if !inScript[id] {
				t.Errorf("production selects %q for %s, but capture.sh does not capture through "+
					"it — the corpus would model a dictionary the tool never asks", id, lang)
			}
		}
	}
	for id := range inScript {
		var known bool
		for _, ids := range curated {
			if slices.Contains(ids, id) {
				known = true
			}
		}
		if !known {
			t.Errorf("capture.sh captures through %q, which no curated list names — fixtures "+
				"from a dictionary production never consults cannot be conformance-checked", id)
		}
	}
}

// Italian resolves to the Devoto-Oli, and the BILINGUAL Oxford is rejected (#31).
//
// The second half is the one worth having. `OxfordItalian` is installed on this
// host and indexes `it>it` AND `en>it`, so a rule that accepted "indexes Italian"
// would put English glosses in front of an Italian learner — the outcome #23's
// mode exists to prevent, and the reason "just prefer the other Italian book" is
// not available to #34 either.
func TestChooseDictionaryPicksTheCuratedItalian(t *testing.T) {
	devoto := dictMeta{ID: "com.apple.dictionary.it.Devoto-Oli",
		Langs: []langPair{{Index: "it", Description: "it"}}}
	bilingual := dictMeta{ID: "com.apple.dictionary.OxfordItalian",
		Langs: []langPair{{Index: "it", Description: "it"}, {Index: "en", Description: "it"}}}

	got, ok := chooseDictionary([]dictMeta{bilingual, devoto}, "it")
	if !ok || len(got) != 1 || got[0].ID != devoto.ID {
		t.Errorf("chooseDictionary(it) = %v, %v; want just the Devoto-Oli", got, ok)
	}
	// The bilingual book ALONE is not a fallback: no curated monolingual book
	// means the NULL search, never a different-language dictionary.
	if got, ok := chooseDictionary([]dictMeta{bilingual}, "it"); ok {
		t.Errorf("chooseDictionary(it) accepted the bilingual Oxford: %v", got)
	}
}

// Every curated language has a corpus, which is the direction nothing checked.
//
// `capturedLanguages` derives from the DIRECTORY and guards only `len(out) >= 2`
// (dict_fake_test.go), so it answers "what did we capture" and can never answer
// "did we capture what production selects". Concretely: deleting
// testdata/entries/it/ outright leaves en+es, satisfies that guard, and reddens
// nothing — while `curated` still points every Italian session at a book whose
// fixtures are gone, and every conformance check that would have caught the
// drift silently has nothing to read.
//
// This is the same row-vs-tree asymmetry #29 closed for plan tables: a claim in
// one artifact is only checked in the direction someone thought to look.
func TestEveryCuratedLanguageHasACorpus(t *testing.T) {
	for lang := range curated {
		t.Run(string(lang), func(t *testing.T) {
			paths, err := filepath.Glob(filepath.Join("testdata", "entries", string(lang), "*.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if len(paths) == 0 {
				t.Errorf("production curates %v for %s, but testdata/entries/%s holds no "+
					"fixtures — the seam has nothing to conformance-check, so a change in "+
					"that dictionary would surface as a user complaint rather than a red test. "+
					"Run testdata/capture.sh (unsandboxed).", curated[lang], lang, lang)
			}
		})
	}
}

// The README names every curated language, or it goes stale the way the atlas
// command table did — three of five commands listed, the two missing being the
// two most recently added (#29).
//
// NOT a generated span, and that is a decision. `TestDocsQuoteTheCommandList`
// can generate because `commands` owns the strings the doc prints; `curated`
// owns bundle IDENTIFIERS while the README names book TITLES ("Larousse
// Diccionario General"), and nothing maps one to the other. The three ways out
// were: add a title field to `curated`, replace the prose with an identifier
// table, or pin the LANGUAGES. The first puts a display string into production
// data to serve a doc test and changes the shape every consumer of `curated`
// reads; the second makes a friendly paragraph unfriendly. The drift actually
// worth catching is a language curated but undocumented, and that is this.
//
// The code→name map lives HERE rather than in production because it is a fact
// about English prose, not about the dictionaries.
//
// SCOPED TO A MARKED SPAN, not to the whole file. Free-text containment passed
// the moment it was written, and for the wrong reason: #29 had left the sentence
// "Italian and Japanese have no recordings in this CDN generation" in the -pron
// section, so the README "named Italian" while its dictionary paragraph still
// listed two languages. A check satisfied by an unrelated sentence certifies
// nothing — the class internal/conformance/guard_test.go records four rounds of.
func TestDocsNameEveryCuratedLanguage(t *testing.T) {
	names := map[store.Lang]string{"en": "English", "es": "Spanish", "it": "Italian"}
	b, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("README.md unreadable: %v", err)
	}
	_, rest, ok := strings.Cut(string(b), "<!-- curated-languages -->")
	if !ok {
		t.Fatal("README.md has no <!-- curated-languages --> span; the dictionary paragraph " +
			"is what must name every curated language, and an unmarked one cannot be checked")
	}
	span, _, ok := strings.Cut(rest, "<!-- /curated-languages -->")
	if !ok {
		t.Fatal("README.md opens <!-- curated-languages --> and never closes it")
	}
	for lang := range curated {
		name, ok := names[lang]
		if !ok {
			t.Errorf("curated has %s but this test has no English name for it — add the row "+
				"here and the language to the README, which is the pair this test keeps together", lang)
			continue
		}
		if !strings.Contains(span, name) {
			t.Errorf("the README's dictionary paragraph never names %s (%s), which production "+
				"curates %v for — a reader cannot discover a language the docs omit",
				name, lang, curated[lang])
		}
	}
}

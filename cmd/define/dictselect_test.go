package main

import (
	"errors"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// installedOnThisMachine is the metadata actually measured on this host — 2026-08-28
// for English and Spanish, 2026-08-29 for Italian — not an invented fixture. The
// ones that matter out of 87.
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
		// Italian, measured 2026-08-29 (#31). Same shape as the Spanish pair and
		// present for the same reason: the monolingual book plus the bilingual
		// one that must NOT be chosen. Adding a language to `curated` without
		// adding it here leaves the fixture modelling a machine the tool no
		// longer targets, which is what forced the Italian case to re-declare
		// these two records locally.
		{ID: "com.apple.dictionary.it.Devoto-Oli", Langs: []langPair{{Index: "it", Description: "it"}}},
		{ID: "com.apple.dictionary.OxfordItalian", Langs: []langPair{
			{Index: "it", Description: "it"},
			{Index: "en", Description: "it"},
		}},
	}
}

// EVERY curated language is modelled in the fixture above, in BOTH directions.
//
// This is the rule the close review named after four instances: no site states a
// per-language fact by hand — every per-language enumeration ranges over
// `curated`, and every language-keyed fixture is cross-checked against it. The
// fixture is what the order test and the selection tests range over, so a
// language curated but unmodelled silently drops out of all of them.
func TestTheMeasuredSetModelsEveryCuratedLanguage(t *testing.T) {
	installed := installedOnThisMachine()
	for lang, ids := range curated {
		for _, want := range ids {
			if !slices.ContainsFunc(installed, func(m dictMeta) bool { return m.ID == want }) {
				t.Errorf("production curates %q for %s, but installedOnThisMachine() does not "+
					"model it — every test that ranges over the fixture silently skips %s",
					want, lang, lang)
			}
		}
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
	// EVERY curated language, not a hand-written pair. It said {"en", "es"} and
	// stayed that way when Italian was curated, so the order-independence
	// requirement — a real one, since DCSCopyAvailableDictionaries returns a SET —
	// went unchecked for the newest language.
	for lang := range curated {
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
	// From the MEASURED set, not a local re-declaration: both books are modelled
	// in installedOnThisMachine(), and declaring them twice is two statements of
	// one fixture fact that can drift apart.
	got, ok := chooseDictionary(installedOnThisMachine(), "it")
	if !ok || len(got) != 1 || got[0].ID != "com.apple.dictionary.it.Devoto-Oli" {
		t.Errorf("chooseDictionary(it) = %v, %v; want just the Devoto-Oli", ids(got), ok)
	}
	// The bilingual book ALONE is not a fallback: no curated monolingual book
	// means the NULL search, never a different-language dictionary.
	bilingual := dictMeta{ID: "com.apple.dictionary.OxfordItalian",
		Langs: []langPair{{Index: "it", Description: "it"}, {Index: "en", Description: "it"}}}
	if got, ok := chooseDictionary([]dictMeta{bilingual}, "it"); ok {
		t.Errorf("chooseDictionary(it) accepted the bilingual Oxford: %v", ids(got))
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
			// Through loadFakeDictionary, not a raw glob: it ALSO rejects a
			// zero-byte fixture (dict_fake_test.go), which a glob counts as
			// present. A short capture is exactly the failure capture.sh's
			// MIN_BYTES floor exists for, and this guard would have called it
			// coverage.
			d, err := loadFakeDictionary("testdata/entries", lang)
			if err != nil || len(d.entries) == 0 {
				t.Errorf("production curates %v for %s, but testdata/entries/%s holds no "+
					"usable fixtures (%v) — the seam has nothing to conformance-check, so a "+
					"change in that dictionary would surface as a user complaint rather than "+
					"a red test. Run testdata/capture.sh (unsandboxed).",
					curated[lang], lang, lang, err)
			}
		})
	}
}

// derivedDocs is the doc set that consumes code-owned strings, named ONCE.
//
// It was spelled three times — here and twice in doc_sync_test.go — so a third
// document is covered only by whichever test its author happened to remember.
var derivedDocs = []string{"../../README.md", "../../atlas/define.md"}

// curatedSurface is one place obliged to name every curated language, plus how
// that place SPELLS a language: the docs write "Italian", the flag help writes
// "it".
type curatedSurface struct {
	what  string
	text  func(t *testing.T) string
	token func(store.Lang) string
}

// langCode and langName are the two spellings a surface can use.
//
// The name map lives in the TEST because it is a fact about English prose, not
// about the dictionaries — `curated` holds identifiers and has no business
// carrying display strings.
func langCode(l store.Lang) string { return string(l) }

func langName(l store.Lang) string {
	switch l {
	case "en":
		return "English"
	case "es":
		return "Spanish"
	case "it":
		return "Italian"
	}
	return "" // unknown: reported as a missing name rather than silently passing
}

// curatedSurfaces is THE registry, and the only hand-written list left in this
// family: "which surfaces exist". Everything else ranges over it.
//
// Five rounds of `curated-consumer-unpinned` findings got here. Each earlier
// round fixed the sites it could see, and the next round found more — because
// the enforcement was per-site, so a new surface was pinned only if its author
// remembered to write a test. With a registry, a new surface is a row and the
// assertion already exists.
func curatedSurfaces() []curatedSurface {
	out := []curatedSurface{
		{
			// This row is what pins langHelp's DERIVATION. The flagset test pins
			// its DELIVERY — that run() passes it rather than a literal — and the
			// two are different: replacing langHelp's body with a hardcoded
			// "en, es" keeps delivery green and fails here.
			what:  "the -lang flag help (langHelp)",
			text:  func(*testing.T) string { return langHelp },
			token: langCode,
		},
	}
	// The derived docs are ROWS, generated from the one list — not two literals
	// beside a var claiming to be that list. `derivedDocs` was introduced with a
	// comment saying it existed so the doc set was "named ONCE" and then had zero
	// callers: package-level vars are exempt from Go's unused check, so it passed
	// every suite while consolidating nothing.
	for _, doc := range derivedDocs {
		out = append(out, curatedSurface{
			what:  doc + "'s curated-languages span",
			text:  func(t *testing.T) string { return markedSpan(t, doc, "curated-languages") },
			token: langName,
		})
	}
	return out
}

// markedSpan returns the text between <!-- name --> and <!-- /name -->.
//
// SCOPED, not whole-file: free-text containment over a README passes for the
// wrong reason — #29 had left "Italian and Japanese have no recordings" in an
// unrelated section, which satisfied an earlier version of this check while the
// dictionary paragraph still listed two languages.
func markedSpan(t *testing.T, doc, name string) string {
	t.Helper()
	b, err := os.ReadFile(doc)
	if err != nil {
		t.Fatalf("%s unreadable: %v", doc, err)
	}
	_, rest, ok := strings.Cut(string(b), "<!-- "+name+" -->")
	if !ok {
		t.Fatalf("%s has no <!-- %s --> span; an unmarked paragraph cannot be checked", doc, name)
	}
	span, _, ok := strings.Cut(rest, "<!-- /"+name+" -->")
	if !ok {
		t.Fatalf("%s opens <!-- %s --> and never closes it", doc, name)
	}
	return span
}

// Every registered surface names every curated language.
//
// ONE assertion over the registry, replacing three per-site tests. Word-boundary
// rather than Contains, because "Italiano" contains "Italian" and a renamed row
// passed the substring version — a check a near-miss satisfies is the same
// defect as one an unrelated sentence satisfies.
func TestEverySurfaceNamesEveryCuratedLanguage(t *testing.T) {
	for _, surface := range curatedSurfaces() {
		t.Run(surface.what, func(t *testing.T) {
			text := surface.text(t)
			for lang := range curated {
				token := surface.token(lang)
				if token == "" {
					t.Errorf("curated has %s but this test has no spelling for it — add it to "+
						"langName, which is the pair that keeps the docs and the map together", lang)
					continue
				}
				if !regexp.MustCompile(`\b` + regexp.QuoteMeta(token) + `\b`).MatchString(text) {
					t.Errorf("%s never names %s (%s), which production curates %v for — a "+
						"reader cannot discover a language the surface omits",
						surface.what, token, lang, curated[lang])
				}
			}
		})
	}
}

// ownLanguageRow is one language's live own-language check, and ownLanguageRows
// is the table.
//
// UNTAGGED on purpose, the same move rawnotation_test.go documents: the
// conformance test that USES these rows is `//go:build darwin && conformance`,
// so anything declared beside it is invisible to the default suite — and the
// cross-check "every curated language has a row" is PURE, a fact about two Go
// values. One producer, two consumers.
//
// Each row carries both halves, and the second is what #23 was built for:
//
//	shared   a word that exists in this language AND in English with an
//	         unrelated meaning. If the two lookups return the same text,
//	         dictionary selection has silently stopped working.
//	marker   a string only a real entry in that language carries, so "different
//	         from English" cannot be satisfied by an error page or an empty read.
//	absent   an English word that must NOT resolve here.
type ownLanguageRow struct{ lang, shared, marker, absent string }

var ownLanguageRows = []ownLanguageRow{
	// mesa: an isolated flat-topped hill in English, furniture in Spanish.
	{"es", "mesa", "nombre femenino", "sycophantic"},
	// pizza: NOAD has it as a loanword; Devoto-Oli has it as ordinary
	// vocabulary. `s.f.` is sostantivo femminile, which NOAD never writes.
	{"it", "pizza", "s.f.", "sycophantic"},
}

// Every curated language has a live own-language row — checked in the DEFAULT
// gate, not behind a build tag.
//
// The cross-check first landed inside TestSelectedDictionaryAnswersInItsOwnLanguage,
// which is `//go:build darwin && conformance`. That is right for the half that
// talks to DictionaryServices and wrong for this half, which is pure: whether a
// curated language has a row is a fact about two Go values. Behind the tag it
// ran nowhere in CI and, as this issue's own reviews showed, nowhere in the
// review environment either — DictionaryServices is unreachable there, so it
// skipped at all four gates.
//
// ownLanguageRows is the shared source; the conformance test ranges over the
// same slice.
func TestEveryCuratedLanguageHasAnOwnLanguageRow(t *testing.T) {
	for lang := range curated {
		// English is the BASELINE every row is measured against ("this entry
		// differs from the English one"), so a row for it would compare it with
		// itself and assert nothing.
		if lang == store.DefaultLang {
			continue
		}
		if !slices.ContainsFunc(ownLanguageRows, func(r ownLanguageRow) bool {
			return r.lang == string(lang)
		}) {
			t.Errorf("production curates %v for %s, but there is no own-language row for it — "+
				"the language ships with nothing asserting it answers from its own dictionary",
				curated[lang], lang)
		}
	}
}

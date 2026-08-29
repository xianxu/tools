package main

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// The fake lives in a _test.go file so it never links into the shipped binary.
// fakeDictionary serves the committed corpus in testdata/entries/<lang>, captured
// from the real API by testdata/capture.sh. Fixtures are real captured output
// rather than hand-written approximations, so the parser is developed against
// text the system actually produces (ARCH-MOCK).
//
// PER-LANGUAGE since #23 M2, and that is what makes the fake model the thing
// that actually changed: the real seam now selects a dictionary, so a fake
// holding one flat pile of entries could not represent "this word exists in
// Spanish but not in English", which is the behaviour the mode buys.
type fakeDictionary struct {
	entries map[string]string
}

// loadFakeDictionary reads one language's fixtures. It fails on an empty corpus:
// a silently-empty fake would make the no-data-loss invariant vacuously green,
// which is the exact failure testdata/capture.sh's byte floor guards against —
// and after #23 it would ALSO make "no entry in this language" pass for a
// language whose fixtures simply were not captured.
func loadFakeDictionary(dir string, lang store.Lang) (*fakeDictionary, error) {
	dir = filepath.Join(dir, string(lang))
	paths, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no fixtures in %s — run testdata/capture.sh (unsandboxed)", dir)
	}
	d := &fakeDictionary{entries: make(map[string]string, len(paths))}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			return nil, fmt.Errorf("empty fixture %s — re-run testdata/capture.sh", p)
		}
		// Lowercase at LOAD, matching what Lookup does to the query. The real
		// dependency is case-insensitive (verified: capture.py Amazon and
		// capture.py amazon both return the entry), so a fake keyed by the raw
		// filename stem diverges from it — iPhone, iPad, MacBook and Amazon were
		// all present on disk yet unreachable through the seam.
		d.entries[strings.ToLower(strings.TrimSuffix(filepath.Base(p), ".txt"))] = string(b)
	}
	return d, nil
}

func (d *fakeDictionary) Lookup(word string) (string, error) {
	if s, ok := d.entries[strings.ToLower(word)]; ok {
		return s, nil
	}
	// ACCENT-INSENSITIVE on a miss, because the real dependency is: `define
	// jalapeno` returns the `jalapeño` entry, and so do pinata, senor, cliche and
	// fiance (measured 2026-08-29; TestLiveDictionaryResolvesAnUnaccentedQuery
	// pins it).
	//
	// #29 is what made this divergence matter rather than merely exist. Its whole
	// mechanism starts from "the typed form is not the source orthography", so a
	// fake that can only be reached by the accented spelling cannot represent the
	// case the feature is FOR — the end-to-end test would have had to type
	// `jalapeño`, where typed and headword agree and nothing is exercised.
	//
	// differsOnlyByDiacritics is the production predicate, so the fake and the
	// feature agree on what "the same word in another dress" means by
	// construction rather than by two similar loops (ARCH-DRY).
	// SORTED, because a Go map iterates in random order: two entries differing
	// from the query only by diacritics would otherwise answer differently run to
	// run, and a fake that is not deterministic makes every test above it flaky
	// for reasons that look like the code.
	for _, key := range slices.Sorted(maps.Keys(d.entries)) {
		if differsOnlyByDiacritics(key, word) {
			return d.entries[key], nil
		}
	}
	return "", ErrNoEntry
}

// testDict is the ENGLISH corpus, which is what almost every test wants — the
// parser suite, the renderer goldens, the invariants. testDictFor is the one to
// reach for when the language is the subject.
func testDict(t *testing.T) *fakeDictionary { return testDictFor(t, store.DefaultLang) }

func testDictFor(t *testing.T, lang store.Lang) *fakeDictionary {
	t.Helper()
	d, err := loadFakeDictionary("testdata/entries", lang)
	if err != nil {
		t.Fatalf("loadFakeDictionary: %v", err)
	}
	return d
}

func TestFakeDictionaryLoadsRealFixtures(t *testing.T) {
	d := testDict(t)
	if len(d.entries) == 0 {
		t.Fatal("corpus is empty — later invariant tests would be vacuous")
	}
	got, err := d.Lookup("sycophantic")
	if err != nil {
		t.Fatalf("Lookup(sycophantic): %v", err)
	}
	if !strings.Contains(got, "ˌsikəˈfan(t)ik") {
		t.Errorf("fixture missing the NOAD IPA, got %q", got)
	}
}

func TestFakeDictionaryMissIsErrNoEntry(t *testing.T) {
	if _, err := testDict(t).Lookup("rizz"); !errors.Is(err, ErrNoEntry) {
		t.Errorf("want ErrNoEntry, got %v", err)
	}
}

func TestLoadFakeDictionaryRejectsEmptyCorpus(t *testing.T) {
	if _, err := loadFakeDictionary(t.TempDir(), store.DefaultLang); err == nil {
		t.Error("want an error for an empty corpus, got nil")
	}
	// And a language whose fixtures were never captured is the same failure, not
	// a silently empty dictionary that would make "no entry in this language"
	// pass for every word.
	if _, err := loadFakeDictionary("testdata/entries", store.Lang("zz")); err == nil {
		t.Error("want an error for an uncaptured language, got nil")
	}
}

// Every fixture on disk must be reachable through the seam, not merely present
// in the map. Four were not, which silently disabled the end-to-end coverage of
// the no-pronunciation path they had been added for.
func TestEveryFixtureIsReachableViaLookup(t *testing.T) {
	d := testDict(t)
	for key := range d.entries {
		if _, err := d.Lookup(key); err != nil {
			t.Errorf("fixture %q unreachable via Lookup: %v", key, err)
		}
	}
	// And the capitalized spellings a user would actually type.
	for _, w := range []string{"iPhone", "MacBook", "Amazon", "iPad"} {
		if _, err := d.Lookup(w); err != nil {
			t.Errorf("Lookup(%q) failed: %v", w, err)
		}
	}
}

// The fake models a SET of dictionaries, which is what the real seam became.
//
// Before #23 M2 there was one flat corpus, and it could not represent the thing
// the mode buys: that a word's presence is a fact about a LANGUAGE, not about
// the machine. These are real captures — mesa through the Larousse, not a
// hand-written approximation — so the parser meets text the system produces.
func TestFakeDictionaryIsPerLanguage(t *testing.T) {
	en, es := testDict(t), testDictFor(t, "es")

	// The headline case: an English homograph, correct in both directions.
	enEntry, err := en.Lookup("mesa")
	if err != nil {
		t.Fatalf("mesa in English: %v", err)
	}
	esEntry, err := es.Lookup("mesa")
	if err != nil {
		t.Fatalf("mesa in Spanish: %v", err)
	}
	if !strings.Contains(enEntry, "flat-topped") {
		t.Errorf("the English mesa is not the landform: %.80s", enEntry)
	}
	if !strings.Contains(esEntry, "nombre femenino") {
		t.Errorf("the Spanish mesa is not the Spanish entry: %.80s", esEntry)
	}
	if enEntry == esEntry {
		t.Error("both languages returned the same entry; the fake is not per-language")
	}
}

// "Not a word in this language" — which the tool could not say about anything
// before #23, and which is a Done-when row rather than a nicety.
//
// It needs no fixture: the ABSENCE is the assertion. sycophantic really has no
// Larousse entry (verified live during capture), so the fake models the real
// dependency by not holding one.
func TestAWordAbsentFromTheLanguageReportsNoEntry(t *testing.T) {
	if _, err := testDictFor(t, "es").Lookup("sycophantic"); !errors.Is(err, ErrNoEntry) {
		t.Errorf("sycophantic in Spanish = %v, want ErrNoEntry — answering from English is the bug", err)
	}
	// And the converse, so this is not just "the Spanish corpus is small":
	// madrugar is absent from English.
	if _, err := testDict(t).Lookup("madrugar"); !errors.Is(err, ErrNoEntry) {
		t.Errorf("madrugar in English = %v, want ErrNoEntry", err)
	}
}

// The flattened language-pair encoding the cgo boundary uses. Pure, so the one
// place a malformed pair could make a bilingual dictionary look monolingual is
// testable without CoreServices.
func TestParseLangPairs(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want []langPair
	}{
		{name: "monolingual", in: "es>es,", want: []langPair{{Index: "es", Description: "es"}}},
		{
			name: "bilingual keeps BOTH pairs, which is what disqualifies it",
			in:   "es>es,en>es,",
			want: []langPair{{Index: "es", Description: "es"}, {Index: "en", Description: "es"}},
		},
		{
			// Apple writes both "en" and "en_US"; a region is not a language, and
			// keeping it would make ParseLang reject the pair and silently drop a
			// dictionary from consideration.
			name: "regions are stripped", in: "en_US>en_US,", want: []langPair{{Index: "en", Description: "en"}},
		},
		{name: "empty", in: "", want: nil},
		{name: "junk is skipped, not guessed at", in: "nonsense,es>es,", want: []langPair{{Index: "es", Description: "es"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := parseLangPairs(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("parseLangPairs(%q) = %+v, want %+v", tc.in, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("pair %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// capturedLanguages reads the SET of languages the corpus actually holds.
//
// Every check over the corpus takes its language dimension from HERE rather than
// spelling store.DefaultLang, which is the rule this exists for: when a corpus
// gains a dimension, the checks over it gain the same dimension — or the new
// half is unchecked while two documents claim otherwise. M2 added five real
// Larousse captures and nothing byte-compared them to anything.
//
// Derived from the directory rather than listed, so capturing a third language
// brings it under every check without anyone remembering to widen one.
func capturedLanguages(t *testing.T) []store.Lang {
	t.Helper()
	entries, err := os.ReadDir("testdata/entries")
	if err != nil {
		t.Fatalf("reading the corpus: %v", err)
	}
	var out []store.Lang
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		l, err := store.ParseLang(e.Name())
		if err != nil {
			t.Errorf("testdata/entries/%s is not a language directory", e.Name())
			continue
		}
		out = append(out, l)
	}
	if len(out) < 2 {
		t.Fatalf("found %d captured language(s); #23 M2 committed at least en and es, so this "+
			"check would be blind to the very dimension it exists for", len(out))
	}
	return out
}

// Every captured language loads, and none is empty — the generic form of
// TestFakeDictionaryLoadsRealFixtures, which named English.
func TestEveryCapturedLanguageLoads(t *testing.T) {
	for _, lang := range capturedLanguages(t) {
		d, err := loadFakeDictionary("testdata/entries", lang)
		if err != nil {
			t.Errorf("%s: %v", lang, err)
			continue
		}
		if len(d.entries) == 0 {
			t.Errorf("%s: corpus is empty — later invariant tests would be vacuous", lang)
		}
	}
}

// TestRenderLosesNothingInEveryCapturedLanguage lived here and was DELETED by
// #31, not lost.
//
// It asserted only that a captured entry renders to something non-empty, and it
// existed because it was the one sweep that ran every captured language through
// the parser and renderer. #31 widened TestRenderLosesNothing (invariant_test.go)
// to range over capturedLanguages, and that one asserts EXACT alnum counts —
// which is strictly stronger: an entry with any alnum content cannot render to
// "" without the counts differing.
//
// Proven rather than argued, per this repo's rule that deleting a test needs the
// evidence writing one does: making Render return "" for the Italian `pizza`
// fixture reddens TestRenderLosesNothing at two subtests. The weaker assertion
// has no case left of its own.

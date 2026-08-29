//go:build darwin && conformance

package main

// Live conformance check for the NOAD seam (ARCH-MOCK).
//
// Cadence is deliberately on-demand rather than CI-scheduled: it depends on the
// host having the New Oxford American Dictionary installed, which a CI runner
// does not. Run it after a macOS upgrade, or whenever a parse bug is reported:
//
//	go test -tags conformance ./cmd/define/
//
// It must run UNSANDBOXED — DCSCopyTextDefinition returns silence, not an
// error, without real access to /System/Library/AssetsV2, which would make this
// test report drift that is really just a sandbox.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/conformance"
)

func TestFixturesMatchLiveDictionary(t *testing.T) {
	// EVERY captured language, not just English. M2 added five real Larousse
	// captures and nothing byte-compared them to the live dictionary, while the
	// atlas and the plan both described the corpus as conformance-checked. When a
	// corpus gains a dimension, the checks over it take that dimension.
	for _, lang := range capturedLanguages(t) {
		t.Run(string(lang), func(t *testing.T) {
			fake, err := loadFakeDictionary("testdata/entries", lang)
			if err != nil {
				t.Fatalf("loadFakeDictionary: %v", err)
			}
			live, name := systemDictionary(lang, nil)
			if name == everyActiveDictionary {
				// The fixtures were captured through a CURATED dictionary. With
				// none installed, comparing them against the whole active set
				// would report drift that is really a different machine.
				conformance.SkipOrFail(t, "no curated "+string(lang)+" dictionary is installed",
					errors.New("selection fell back to the NULL search"))
				return
			}
			// PROBE FIRST, because a sandboxed run and a drifted fixture look
			// identical from inside the loop: DCSCopyTextDefinition returns
			// silence without real access to /System/Library/AssetsV2, so every
			// word fails the same way. One lookup up front separates the two.
			var probe string
			for w := range fake.entries {
				probe = w
				break
			}
			if _, err := live.Lookup(probe); err != nil {
				conformance.SkipOrFail(t, "system dictionary unreachable", err)
				return
			}
			// Read through Lookup, not fake.entries: a conformance check that
			// bypasses the seam cannot see the fake diverging at that seam.
			for word := range fake.entries {
				want, err := fake.Lookup(word)
				if err != nil {
					t.Errorf("%s: unreachable through the fake seam: %v", word, err)
					continue
				}
				got, err := live.Lookup(word)
				if err != nil {
					t.Errorf("%s: live lookup failed: %v (sandboxed?)", word, err)
					continue
				}
				if got != want {
					t.Errorf("%s: the %s dictionary drifted from the fixture — re-run "+
						"testdata/capture.sh\n live: %.120q\n fixt: %.120q", word, lang, got, want)
				}
			}
		})
	}
}

// Undocumented symbols, absent from the SDK header, free to disappear on any OS
// update. The list is dcsPrivateSymbols and each member is checked by name, so
// there is no count in prose to go stale. The seam degrades to the pre-#23 NULL search when they do, which is
// the right failure — but a SILENT degradation means the tool quietly stops
// speaking Spanish and nothing says why. This is what says why.
//
// On demand, like the rest of the conformance suite, and routed through
// SkipOrFail so an unreachable dictionary SKIPS by default and FAILS under
// CONFORMANCE_STRICT. It must run UNSANDBOXED: DCSCopyTextDefinition returns
// silence rather than an error without real access to /System/Library/AssetsV2,
// which would make a sandbox indistinguishable from genuine drift.
func TestPrivateDictionarySurfaceStillResolves(t *testing.T) {
	// Member by member, so a report names WHICH symbol moved rather than
	// "something did".
	var missing []string
	for _, name := range dcsPrivateSymbols {
		if !hasPrivateSymbol(name) {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		conformance.SkipOrFail(t, "the private DictionaryServices surface moved",
			fmt.Errorf("dlsym found no %s", strings.Join(missing, ", ")))
		return
	}

	installed := installedDictionaries()
	if installed == nil {
		conformance.SkipOrFail(t, "the private surface resolved but returned nothing",
			errors.New("DCSCopyAvailableDictionaries gave no set"))
		return
	}
	if len(installed) == 0 {
		conformance.SkipOrFail(t, "no dictionaries are installed", errors.New("empty set"))
		return
	}

	// The metadata must still be rich enough to CHOOSE with. A surface that
	// resolves but reports no language pairs would silently make every language
	// uncurated, and the tool would fall back to NULL for everything.
	var withLangs int
	for _, m := range installed {
		if len(m.Langs) > 0 {
			withLangs++
		}
	}
	if withLangs == 0 {
		t.Error("no installed dictionary reports language pairs — DCSDictionaryGetLanguages " +
			"resolved but its shape has changed; chooseDictionary can no longer narrow")
	}

	// The curated policy this repo actually ships, checked against what is
	// really installed. A missing book is a LOG, not a failure: the designed
	// degradation is that the tool falls back, and a machine without the
	// Larousse is not a broken machine.
	for lang, want := range curated {
		chosen, ok := chooseDictionary(installed, lang)
		if !ok {
			t.Logf("%s: none of %v is installed here; the tool falls back to the NULL "+
				"search, which is the designed degradation rather than a defect", lang, want)
			continue
		}
		got := make([]string, len(chosen))
		for i, m := range chosen {
			got[i] = m.ID
		}
		// A SUBSEQUENCE of the curated list, in order: some books may be absent,
		// but a chosen one must be curated and preference must be preserved.
		var wi int
		for _, id := range got {
			for wi < len(want) && want[wi] != id {
				wi++
			}
			if wi == len(want) {
				t.Errorf("%s: chose %v, which is not the curated list %v in order", lang, got, want)
				break
			}
			wi++
		}
	}
}

// The behaviour the whole milestone exists for, checked against the real
// dictionaries rather than the fixtures they were captured from.
//
// A TABLE, not one language with a copy per addition. #23 wrote this for Spanish
// alone; #31 added Italian, and a second copy is how the two drift — the family
// this repo has closed twice, most recently when one symbol locator was written
// twice and the copies had already diverged. A fourth language is a row.
//
// Each row carries BOTH halves, and the second is the one #23 was built for:
//
//	shared   a word that exists in this language AND in English with an
//	         unrelated meaning. If the two lookups return the same text,
//	         dictionary selection has silently stopped working and every session
//	         in this language is answering from English.
//	marker   a string only a real entry in that language carries, so "different
//	         from English" cannot be satisfied by an error page or an empty read.
//	absent   an English word that must NOT resolve here. Answering it from
//	         English is exactly the bug the mode removes.
type ownLanguageRow struct{ lang, shared, marker, absent string }

func TestSelectedDictionaryAnswersInItsOwnLanguage(t *testing.T) {
	en, _ := systemDictionary(store.DefaultLang, nil)

	rows := []ownLanguageRow{
		// mesa: an isolated flat-topped hill in English, furniture in Spanish.
		{"es", "mesa", "nombre femenino", "sycophantic"},
		// pizza: NOAD has it as a loanword; Devoto-Oli has it as ordinary
		// vocabulary. `s.f.` is sostantivo femminile, which NOAD never writes.
		{"it", "pizza", "s.f.", "sycophantic"},
	}
	// EVERY curated language has a row, checked in both directions. Without this,
	// #34 can add fr/de to `curated` and acquire no live own-language check at
	// all — the language would ship with nothing asserting it answers in its own
	// book. Fourth instance of the family; the rule is that every language-keyed
	// table is cross-checked against `curated`, not that this table gets a row.
	for lang := range curated {
		// English is excluded, and not as an oversight: it is the BASELINE every
		// row is measured against ("this entry differs from the English one"), so
		// a row for English would compare it with itself and assert nothing.
		if lang == store.DefaultLang {
			continue
		}
		if !slices.ContainsFunc(rows, func(r ownLanguageRow) bool { return r.lang == string(lang) }) {
			t.Errorf("production curates %v for %s, but this table has no row for it — the "+
				"language ships with no live check that it answers in its own dictionary",
				curated[lang], lang)
		}
	}
	for _, tc := range rows {
		t.Run(tc.lang, func(t *testing.T) {
			d, name := systemDictionary(store.Lang(tc.lang), nil)
			if name == everyActiveDictionary {
				conformance.SkipOrFail(t, "no "+tc.lang+" dictionary is installed",
					errors.New("chooseDictionary fell back to the NULL search"))
				return
			}
			own, err := d.Lookup(tc.shared)
			if err != nil {
				conformance.SkipOrFail(t, tc.lang+" dictionary unreachable", err)
				return
			}
			english, err := en.Lookup(tc.shared)
			if err != nil {
				t.Fatalf("%s in English: %v", tc.shared, err)
			}
			if own == english {
				t.Errorf("%s returned the same entry in both languages — dictionary selection "+
					"is not taking effect, and every %s session is answering from English",
					tc.shared, tc.lang)
			}
			if !strings.Contains(own, tc.marker) {
				t.Errorf("the %s %s is not a %s entry (no %q): %.100q",
					tc.lang, tc.shared, tc.lang, tc.marker, own)
			}
			if _, err := d.Lookup(tc.absent); !errors.Is(err, ErrNoEntry) {
				t.Errorf("%s through the %s dictionary = %v, want ErrNoEntry — answering it "+
					"from English is exactly the bug the mode removes", tc.absent, tc.lang, err)
			}
		})
	}
}

// The raw-notation exemplars must still BE exemplars against the live
// dictionary (ARCH-MOCK).
//
// They are a fake — four captured entries standing in for shapes the renderer
// mishandles — and a fake with no conformance check becomes a fossil. A macOS
// dictionary update could fix any of these upstream, and then the classifier
// would keep passing against text the tool no longer produces, while the live
// ratchet quietly reported a lower count that nobody attributed.
//
// Same standard the language corpus is held to by TestFixturesMatchLiveDictionary:
// read through the seam, byte-compare, and skip rather than fail when the
// dictionary is unreachable.
func TestRawNotationExemplarsMatchLiveDictionary(t *testing.T) {
	live, name := systemDictionary(store.DefaultLang, nil)
	if name == everyActiveDictionary {
		conformance.SkipOrFail(t, "no curated English dictionary is installed",
			errors.New("selection fell back to the NULL search"))
		return
	}
	paths, err := filepath.Glob(filepath.Join("testdata", "rawnotation", "*.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no raw-notation exemplars — run testdata/capture.sh (unsandboxed)")
	}
	for _, path := range paths {
		word := strings.TrimSuffix(filepath.Base(path), ".txt")
		t.Run(word, func(t *testing.T) {
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			got, err := live.Lookup(word)
			if err != nil {
				conformance.SkipOrFail(t, "dictionary unreachable for "+word, err)
				return
			}
			if got != string(want) {
				t.Errorf("%s drifted from its captured exemplar — re-run testdata/capture.sh, "+
					"and check whether the SHAPE it exemplifies still exists\n live: %.120q\n fixt: %.120q",
					word, got, string(want))
			}
			// And it must still trip the oracle, or it has stopped being an
			// exemplar and the offline test passes vacuously.
			if _, tripped := rawNotationNear(Render(ParseEntry(got), RenderOpts{Width: 0})); !tripped {
				t.Errorf("%s no longer renders raw notation — the shape may be fixed upstream; "+
					"retire the exemplar and lower knownRawByCause", word)
			}
		})
	}
}

// The real dictionary resolves an UNACCENTED query to its accented headword, and
// fakeDictionary now models that (#29).
//
// This is the live half of that model. It is not decoration: #29's whole
// mechanism begins with "what a person types is not the source orthography", and
// the evidence for it is that NOAD hands back `jalapeño` when asked for
// `jalapeno`. If the dictionary ever stops doing that, SourceSpellings' first
// and best source dries up — every borrowing would fall back to the session's
// recording, quietly, with the unit suite still green because the fake would go
// on modelling behaviour the dependency no longer has.
func TestLiveDictionaryResolvesAnUnaccentedQuery(t *testing.T) {
	for _, tc := range []struct{ typed, headword string }{
		{"jalapeno", "jalapeño"},
		{"pinata", "piñata"},
		{"senor", "Señor"},
		{"cliche", "cliché"},
		{"fiance", "fiancé"},
	} {
		t.Run(tc.typed, func(t *testing.T) {
			raw, err := noadDictionary{}.Lookup(tc.typed)
			if err != nil {
				conformance.SkipOrFail(t, "NOAD unavailable", err)
				return
			}
			if got := ParseEntry(raw).Headword(); got != tc.headword {
				t.Errorf("Lookup(%q) headword = %q, want %q — an unaccented query no "+
					"longer reaches the accented entry, so SourceSpellings' best source "+
					"is gone and every borrowing silently falls back", tc.typed, got, tc.headword)
			}
		})
	}
}

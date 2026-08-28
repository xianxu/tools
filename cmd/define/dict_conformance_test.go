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
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/conformance"
)

func TestFixturesMatchLiveDictionary(t *testing.T) {
	fake, err := loadFakeDictionary("testdata/entries", store.DefaultLang)
	if err != nil {
		t.Fatalf("loadFakeDictionary: %v", err)
	}
	live, _ := systemDictionary(store.DefaultLang, nil)
	// PROBE FIRST, because a sandboxed run and a drifted fixture look identical
	// from inside the loop: DCSCopyTextDefinition returns silence without real
	// access to /System/Library/AssetsV2, so EVERY word fails the same way. One
	// lookup up front separates the two — past this point a failed lookup means
	// the fixture and the live dictionary genuinely disagree, which is the thing
	// this test exists to report.
	//
	// It routes through skipOrFail for the same reason the Skipf sites do (BR-9),
	// and it is the half of that rule a `grep 't\.Skipf('` cannot see: written as
	// an unconditional t.Errorf, an absent dependency was a hard FAILURE, so the
	// non-strict `go test -tags conformance ./...` could never be green offline.
	// One helper now owns both directions.
	if _, err := live.Lookup("sycophantic"); err != nil {
		conformance.SkipOrFail(t, "system dictionary unreachable", err)
	}
	// Read through Lookup, not fake.entries: a conformance check that bypasses
	// the seam cannot see the fake diverging from the dependency at that seam.
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
			t.Errorf("%s: NOAD drifted from the fixture — re-run testdata/capture.sh\n live: %.120q\n fixt: %.120q",
				word, got, want)
		}
	}
}

// The PRIVATE surface, which is the whole risk M2 took on.
//
// Nine undocumented symbols, absent from the SDK header, free to disappear on any
// OS update. The seam degrades to the pre-#23 NULL search when they do, which is
// the right failure — but a SILENT degradation means the tool quietly stops
// speaking Spanish and nothing says why. This is what says why.
//
// On demand, like the rest of the conformance suite, and routed through
// SkipOrFail so an unreachable dictionary SKIPS by default and FAILS under
// CONFORMANCE_STRICT. It must run UNSANDBOXED: DCSCopyTextDefinition returns
// silence rather than an error without real access to /System/Library/AssetsV2,
// which would make a sandbox indistinguishable from genuine drift.
func TestPrivateDictionarySurfaceStillResolves(t *testing.T) {
	installed := installedDictionaries()
	if installed == nil {
		conformance.SkipOrFail(t, "the private DictionaryServices surface did not resolve",
			errors.New("dlsym found no DCSCopyAvailableDictionaries/GetIdentifier/GetLanguages"))
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
// mesa is the case: a word that exists in BOTH languages with unrelated
// meanings. If this ever returns the same text twice, dictionary selection has
// silently stopped working and every Spanish session is answering from English.
func TestSelectedDictionaryAnswersInItsOwnLanguage(t *testing.T) {
	es, esName := systemDictionary(store.Lang("es"), nil)
	if esName == everyActiveDictionary {
		conformance.SkipOrFail(t, "no Spanish dictionary is installed",
			errors.New("chooseDictionary fell back to the NULL search"))
		return
	}
	en, _ := systemDictionary(store.DefaultLang, nil)

	esEntry, err := es.Lookup("mesa")
	if err != nil {
		conformance.SkipOrFail(t, "Spanish dictionary unreachable", err)
		return
	}
	enEntry, err := en.Lookup("mesa")
	if err != nil {
		t.Fatalf("mesa in English: %v", err)
	}
	if esEntry == enEntry {
		t.Error("mesa returned the same entry in both languages — dictionary selection is not " +
			"taking effect, and every Spanish session is answering from English")
	}
	if !strings.Contains(esEntry, "nombre femenino") {
		t.Errorf("the Spanish mesa is not a Spanish entry: %.100q", esEntry)
	}

	// And the absence, which is the answer the tool could not give before #23.
	if _, err := es.Lookup("sycophantic"); !errors.Is(err, ErrNoEntry) {
		t.Errorf("sycophantic through the Spanish dictionary = %v, want ErrNoEntry — "+
			"answering it from English is exactly the bug the mode removes", err)
	}
}

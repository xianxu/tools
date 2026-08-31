//go:build conformance

package main

// Live conformance for the audio CDN (ARCH-MOCK).
//
// The candidate ORDER in AudioCandidates rests on two measured facts about
// Google's CDN. This asserts they still hold; the fake models them.
//
// Cadence is on-demand with the rest of the conformance suite — it needs network
// access, which does not belong in merge-check.yml.
//
//	go test -tags conformance ./cmd/define/
//
// UNFILTERED, deliberately. This line used to say `-run CDN`, and a filter is
// exactly the mechanism that lets a row decay unnoticed: #29 wrote a row named
// TestFrenchCoverageIsStillPartial, which that filter would have skipped while
// the verification step reported success. workshop/lessons.md already has the
// class — "A live conformance check that is never run is not a check", where
// TestPTYSuggestionAndAcceptance sat red through two merges. Every row here is
// TestCDN*-prefixed anyway, so the filter bought nothing and cost that.

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/conformance"
)

// missesEntirely asserts that NO candidate answers — through the production
// fetch, not by probing one URL.
//
// The negative rows here pin claims about a WALK ("Italian is absent", "French
// coverage is partial", "jalapeno_es_es is a 404"), and they were checking
// AudioCandidates(...)[0] alone. A recording appearing only at the _2 suffix
// would have left every row green while the fallback quietly stopped firing —
// the row would be pinning a smaller claim than the one it is named for.
// TestCDNReturnsRealAudio already uses this shape for the positive case.
func missesEntirely(t *testing.T, word string, v voice) {
	t.Helper()
	_, from, err := newHTTPAudioSource().Fetch(t.Context(), AudioCandidates(word, v))
	if errors.Is(err, ErrNoAudio) {
		return
	}
	if err != nil {
		conformance.SkipOrFail(t, "network unavailable", err)
		return
	}
	t.Errorf("%s now has a %s_%s recording (%s) — a claim this file pins has changed",
		word, v.Lang, v.Locale, from)
}

func head(t *testing.T, url string) int {
	t.Helper()
	c := &http.Client{Timeout: 15 * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		conformance.SkipOrFail(t, "network unavailable", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestCDNStillServesTheExpectedPaths(t *testing.T) {
	// Fact 1: the primary path serves a known word.
	if got := head(t, AudioCandidates("sycophantic", voice{Lang: "en", Locale: "us"})[0]); got != http.StatusOK {
		t.Errorf("primary path for sycophantic = %d, want 200 — the CDN generation may have moved", got)
	}
	// Fact 2: the 2022 generation strictly dominates the legacy one. gaslighting
	// exists ONLY on the newer path; if that stops being true the ordering in
	// AudioCandidates is no longer justified by measurement.
	cands := AudioCandidates("gaslighting", voice{Lang: "en", Locale: "us"})
	var modern, legacy string
	for _, u := range cands {
		if strings.Contains(u, "/pronunciation/") && modern == "" {
			modern = u
		}
		if strings.Contains(u, "/sounds/oxford/") && legacy == "" {
			legacy = u
		}
	}
	if got := head(t, modern); got != http.StatusOK {
		t.Errorf("gaslighting on the 2022 path = %d, want 200", got)
	}
	if got := head(t, legacy); got == http.StatusOK {
		t.Errorf("gaslighting now exists on the legacy path too — re-run the survey; the ordering rationale has changed")
	}
}

// The SPANISH facts, which #23 added and which had no live row until this one.
//
// The English-only gate on the legacy path (audiourl.go) rests on measurement,
// and measurement that lives only in a comment and in a fake written to agree
// with it is not checked by anything. Without this, a CDN move would surface as
// silence in Spanish sessions — a language whose recordings simply stopped
// arriving, with every test still green.
func TestCDNStillServesSpanishOnTheExpectedPaths(t *testing.T) {
	// BOTH locales, because #27 makes both selectable and es_us was the one
	// nothing checked. They are not two accents of one recording: es_es is
	// Castilian (cazar /θ/ ≠ casar /s/), es_us is Latin American seseo (both
	// /s/), so a learner acquires a different sound system from each. If one
	// disappears, that half of the flag silently stops working.
	for _, locale := range []string{"es", "us"} {
		cands := AudioCandidates("madrugar", voice{Lang: "es", Locale: locale})
		if got := head(t, cands[0]); got != http.StatusOK {
			t.Errorf("madrugar on es_%s = %d, want 200 — that locale's recordings may have "+
				"moved, and -locale %s would go silent for Spanish", locale, got, locale)
		}
	}
	// And the pair the distinction is ABOUT, so a drift in coverage shows up on
	// the words where the phonemic split is audible rather than only on one verb.
	for _, w := range []string{"cazar", "casar"} {
		if got := head(t, AudioCandidates(w, voice{Lang: "es", Locale: "es"})[0]); got != http.StatusOK {
			t.Errorf("%s on es_es = %d, want 200 — the Castilian /θ/ evidence", w, got)
		}
	}
	cands := AudioCandidates("madrugar", voice{Lang: "es", Locale: "es"})
	// Fact 2: the legacy generation does NOT, which is why AudioCandidates emits
	// it for English only. If this starts returning 200 the gate is no longer
	// justified by measurement and should be re-surveyed rather than kept.
	legacy := audioBase + "/sounds/oxford/madrugar--_es_1.mp3"
	if got := head(t, legacy); got == http.StatusOK {
		t.Errorf("the legacy path now serves Spanish (%s) — re-run the survey; "+
			"AudioCandidates gates it to English on the measurement that it does not", legacy)
	}
	for _, u := range cands {
		if strings.Contains(u, "/sounds/oxford/") {
			t.Errorf("a Spanish candidate reached the English-only legacy path: %s", u)
		}
	}
}

func TestCDNReturnsRealAudio(t *testing.T) {
	data, from, err := newHTTPAudioSource().Fetch(t.Context(), AudioCandidates("sycophantic", voice{Lang: "en", Locale: "us"}))
	if err != nil {
		conformance.SkipOrFail(t, "network unavailable", err)
	}
	if len(data) < 1000 {
		t.Errorf("%s returned %d bytes — too small to be a recording", from, len(data))
	}
	// MPEG audio starts with an ID3 tag or a frame sync.
	if !(strings.HasPrefix(string(data), "ID3") || (data[0] == 0xFF && data[1]&0xE0 == 0xE0)) {
		t.Errorf("%s does not look like MPEG audio: % x", from, data[:4])
	}
}

// The source-orthography constraint #29's SourceSpellings exists for.
//
// If jalapeno_es_es ever starts answering, that function is carrying weight it
// no longer needs and its second and third sources could go.
func TestCDNStillKeysSourceRecordingsOnTheSourceSpelling(t *testing.T) {
	es := voice{Lang: "es", Locale: "es"}
	if got := head(t, AudioCandidates("jalapeño", es)[0]); got != http.StatusOK {
		t.Errorf("jalapeño on es_es = %d, want 200 — the Spanish recording this issue fetches", got)
	}
	// The whole walk, not one URL: if the unaccented spelling answered at ANY
	// suffix, SourceSpellings' extra sources would be unnecessary.
	missesEntirely(t, "jalapeno", es)
}

// D1's evidence, and the row that would justify REOPENING the decision.
//
// #29 rejected inferring the language from ORIGIN because neither NOAD nor the
// CDN can tell a live loanword from a naturalised one: the dictionary writes
// "from French" for police exactly as it does for arrondissement, and the CDN
// serves French recordings for words nobody wants said in French. If that stops
// being true the argument weakens and the decision deserves another look.
func TestCDNStillCannotTellALoanwordFromANaturalisedOne(t *testing.T) {
	fr := voice{Lang: "fr", Locale: "fr"}
	for _, w := range []string{"police", "restaurant", "machine"} {
		if got := head(t, AudioCandidates(w, fr)[0]); got != http.StatusOK {
			t.Errorf("%s on fr_fr = %d, want 200 — #29 D1 rejected ORIGIN inference BECAUSE "+
				"the CDN serves French for fully anglicised words. Re-run the survey; the "+
				"decision may deserve reopening", w, got)
		}
	}
}

// Italian's absence is a Done-when: it must be REPORTED, not heard as silence.
//
// Nine probes across three locale forms found no Italian audio in this CDN
// generation. If it arrives, reportVoice stops firing for Italian and the
// atlas's limitation note goes stale — so this failing is good news that still
// needs acting on.
func TestCDNItalianIsStillAbsentFromThisGeneration(t *testing.T) {
	it := voice{Lang: "it", Locale: "it"}
	for _, w := range []string{"ciao", "pizza", "espresso"} {
		missesEntirely(t, w, it)
	}
}

// Why the fallback exists at all: French coverage is PARTIAL.
//
// Named TestCDN* like every row here, because the file's cadence used to filter
// on that prefix and a row that does not match is a row that never runs.
func TestCDNFrenchCoverageIsStillPartial(t *testing.T) {
	fr := voice{Lang: "fr", Locale: "fr"}
	en := voice{Lang: "en", Locale: "us"}
	// NOT déjeuner, which #29's Done-when originally named: it has no NOAD entry,
	// so the lookup fails before audio is reached and it can never exercise this.
	for _, w := range []string{"hotel", "debut"} {
		missesEntirely(t, w, fr)
		if got := head(t, AudioCandidates(w, en)[0]); got != http.StatusOK {
			t.Errorf("%s on en_us = %d, want 200 — this word is the fallback's TARGET, "+
				"so without it the pair proves nothing", w, got)
		}
	}
}

// Multi-word headwords are keyed with the spaces REMOVED, and the separators that
// look plausible are 404s.
//
// This row exists because the unit test could not have caught the bug it pins.
// AudioCandidates spelled "hot dog" as `hot_dog` from the day it was written —
// an assumption that never met the server — and the unit test asserted the same
// assumption back, so every multi-word headword in the corpus missed silently for
// the life of the feature. A miss is a SUPPORTED outcome on this path, which is
// what kept it quiet: nothing distinguishes "the CDN has no recording" from "we
// asked for the wrong file".
//
// So the negative half is the point. Asserting only that `hotdog` answers would
// stay green if the CDN started accepting both spellings, and the day it stopped
// would be the day phrases broke again with no test to say why.
func TestCDNKeysPhrasesWithoutSeparators(t *testing.T) {
	// Both measured 200 on 2026-08-30. Two words rather than one: `hot dog` is in
	// the committed corpus, and `de facto` is not, so the rule is pinned as a rule
	// and not as one fixture's accident.
	for _, phrase := range []string{"hot dog", "de facto"} {
		t.Run(phrase, func(t *testing.T) {
			// Through the production fetch, like the other positive rows here — the
			// recording may sit at the _2 suffix, and a row that probes [0] alone
			// pins a smaller claim than its name.
			_, from, err := newHTTPAudioSource().Fetch(t.Context(), AudioCandidates(phrase, voice{Lang: "en", Locale: "us"}))
			if err != nil {
				if errors.Is(err, ErrNoAudio) {
					t.Errorf("%q has no recording under any candidate — either the CDN dropped it "+
						"or the key rule changed again; re-measure before editing AudioCandidates", phrase)
					return
				}
				conformance.SkipOrFail(t, "network unavailable", err)
				return
			}
			if !strings.Contains(from, strings.ReplaceAll(phrase, " ", "")) {
				t.Errorf("%q answered from %s, which is not the spaces-removed key", phrase, from)
			}
		})
	}

	// The separators the code used to send, and the one that reads as its obvious
	// alternative. Probed directly rather than through AudioCandidates, since the
	// whole claim is about a spelling the builder must NOT produce.
	base := "https://ssl.gstatic.com/dictionary/static/pronunciation/2022-03-02/audio/ho/"
	for _, bad := range []string{"hot_dog", "hot-dog"} {
		if got := head(t, base+bad+"_en_us_1.mp3"); got == http.StatusOK {
			t.Errorf("%s now resolves — the CDN accepts more than one spelling, so the "+
				"unit test's negative assertion has stopped discriminating", bad)
		}
	}
}

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
//	go test -tags conformance -run CDN ./cmd/define/

import (
	"github.com/xianxu/tools/internal/conformance"
	"net/http"
	"strings"
	"testing"
	"time"
)

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

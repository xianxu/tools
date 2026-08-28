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

package main

import (
	"math"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

// The ARITHMETIC, on synthetic input.
//
// Deliberately NOT driven through a fake seeded to vary: that would report how
// the fake was seeded, which is a fact about the test. The number only says
// something about the model when it is measured against the real service, which
// is what the conformance row does — this owns the maths, that owns the floor.
func TestAgreement(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []store.Band
		want float64
	}{
		{"empty is zero, not a divide by zero", nil, 0},
		{"one assignment is trivially stable", []store.Band{store.C1}, 1},
		{"unanimous", []store.Band{store.B2, store.B2, store.B2, store.B2}, 1},
		{"the mode, not the first answer", []store.Band{store.C1, store.B2, store.B2, store.B2}, 0.75},
		{"an even split scores half", []store.Band{store.B1, store.B1, store.C1, store.C1}, 0.5},
		{"pure noise scores 1/N", []store.Band{store.A1, store.A2, store.B1, store.B2}, 0.25},
		// A refusal is INSTABILITY, not a missing sample: the denominator is
		// every assignment asked for, so a model answering off-scale half the
		// time cannot score 1.0 on the half that parsed.
		{"off-scale answers count against stability", []store.Band{store.C1, store.C1, "B2+", "intermediate"}, 0.5},
		{"all off-scale is zero", []store.Band{"B2+", "advanced"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := agreement(tc.in); math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("agreement(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// The prompt is pinned, both shapes, because the domain half is the thing this
// task conditionally drops and a golden is what makes that visible.
func TestBandPromptGolden(t *testing.T) {
	llmtest.AssertGolden(t, "testdata", "band-prompt",
		renderBandPrompt("sycophantic", "behaving or done in an obsequious way in order to gain advantage", ""))
}

func TestBandPromptWithAKnownDomainGolden(t *testing.T) {
	law, ok := store.ParseDomain("Law")
	if !ok {
		t.Fatal(`ParseDomain("Law") refused`)
	}
	llmtest.AssertGolden(t, "testdata", "band-prompt-known-domain",
		renderBandPrompt("certiorari", "Law a writ by which a higher court reviews a lower court's decision", law))
}

// The closed set must actually REACH the model. A prompt that asks for "the
// subject field" without listing the values is how an open vocabulary comes back
// despite a parse that refuses one — the parse would then silently answer
// `general` for everything and topicSpread would read 1.0 forever.
func TestBandPromptCarriesTheClosedDomainSet(t *testing.T) {
	got := renderBandPrompt("sycophantic", "", "").Prompt
	for _, d := range store.Domains() {
		if !contains(got, string(d)) {
			t.Errorf("the prompt never names the domain %q", d)
		}
	}
	if !contains(got, "general") {
		t.Error("the prompt never offers `general`, which is the common and correct answer")
	}
}

// The dictionary's answer SUPPRESSES the ask rather than merely being ignored
// afterwards — the whole saving is not sending it.
func TestBandPromptDoesNotAskForAKnownDomain(t *testing.T) {
	law, ok := store.ParseDomain("Law")
	if !ok {
		t.Fatal(`ParseDomain("Law") refused`)
	}
	got := renderBandPrompt("certiorari", "", law).Prompt
	if contains(got, "EXACTLY this list") {
		t.Error("the prompt still enumerates the domain set although the dictionary already answered")
	}
	if !contains(got, "already known from the dictionary") {
		t.Error("the prompt does not tell the model the domain is settled")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}

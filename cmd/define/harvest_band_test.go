package main

import (
	"math"
	"strings"
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
		// CASING IS NOT DISAGREEMENT. ParseBand calls case and space transcription
		// noise; counting the raw answer made a perfectly stable model score 0.67
		// and fail a floor whose prescribed remedy is the expensive hand-labelled
		// sample this issue defers.
		{"casing is not disagreement", []store.Band{store.C1, "c1", store.C1}, 1},
		{"trailing space is not disagreement", []store.Band{store.C1, "C1 "}, 1},
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
		renderBandPrompt(store.DefaultLang, "sycophantic", "behaving or done in an obsequious way in order to gain advantage", ""))
}

func TestBandPromptWithAKnownDomainGolden(t *testing.T) {
	law, ok := store.ParseDomain("Law")
	if !ok {
		t.Fatal(`ParseDomain("Law") refused`)
	}
	llmtest.AssertGolden(t, "testdata", "band-prompt-known-domain",
		renderBandPrompt(store.DefaultLang, "certiorari", "Law a writ by which a higher court reviews a lower court's decision", law))
}

// The closed set must actually REACH the model. A prompt that asks for "the
// subject field" without listing the values is how an open vocabulary comes back
// despite a parse that refuses one — the parse would then silently answer
// `general` for everything and topicSpread would read 1.0 forever.
func TestBandPromptCarriesTheClosedDomainSet(t *testing.T) {
	got := renderBandPrompt(store.DefaultLang, "sycophantic", "", "").Prompt
	for _, d := range store.Domains() {
		if !strings.Contains(got, string(d)) {
			t.Errorf("the prompt never names the domain %q", d)
		}
	}
	if !strings.Contains(got, "general") {
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
	got := renderBandPrompt(store.DefaultLang, "certiorari", "", law).Prompt
	if strings.Contains(got, "EXACTLY this list") {
		t.Error("the prompt still enumerates the domain set although the dictionary already answered")
	}
	if !strings.Contains(got, "already known from the dictionary") {
		t.Error("the prompt does not tell the model the domain is settled")
	}
}

// The language reaches the model, because facts are stored per-language and a
// Spanish deck is a shipped path. A prompt asserting English while writing
// facts/es/ would be wrong FOREVER — the cache is never re-examined.
func TestBandPromptCarriesTheLanguage(t *testing.T) {
	en := renderBandPrompt(store.DefaultLang, "red", "", "").Prompt
	es := renderBandPrompt(store.Lang("es"), "red", "", "").Prompt
	if en == es {
		t.Fatal("the prompt is identical in two languages; `red` is a different word in each")
	}
	if !strings.Contains(es, "`es`") {
		t.Errorf("the Spanish prompt never names its language: %q", es)
	}
	// And no language is asserted anywhere a caller cannot override.
	if strings.Contains(renderBandPrompt(store.Lang("es"), "red", "", "").System, "English") {
		t.Error("the system prompt still hardcodes English")
	}
}

// An empty Lang is DefaultLang, not an empty code in the prompt: every other
// store constructor makes the same substitution, and a caller predating
// languages must not produce a request naming no language at all.
func TestBandPromptDefaultsTheLanguage(t *testing.T) {
	if got, want := renderBandPrompt("", "red", "", "").Prompt,
		renderBandPrompt(store.DefaultLang, "red", "", "").Prompt; got != want {
		t.Error("an empty Lang did not default the way NewYAML does")
	}
}

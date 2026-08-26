package main

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/llm/llmtest"
)

func sampleEvidence() deckEvidence {
	d := func(n int) time.Time { return time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -n) }
	return deckEvidence{
		Words: []historyRow{
			{Word: "certiorari", FirstAt: d(20), LastAt: d(2), Lookups: 3},
			{Word: "estoppel", FirstAt: d(18), LastAt: d(9), Lookups: 1},
			{Word: "sycophantic", FirstAt: d(4), LastAt: d(1), Lookups: 2},
		},
		Lookups: 84, Questions: 6, From: d(20), To: d(1),
	}
}

func TestRenderReflectPrompt(t *testing.T) {
	llmtest.AssertGolden(t, "testdata", "reflect-prompt", renderReflectPrompt(sampleEvidence()))
}

// The evidence list is the ONLY thing a claim may cite, so it has to reach the
// prompt in full — checkEvidence drops what is not in the deck, and a word the
// model never saw is a claim it cannot make rather than one we discard.
func TestRenderReflectPromptCarriesEveryWord(t *testing.T) {
	got := renderReflectPrompt(sampleEvidence())

	for _, want := range []string{"certiorari", "estoppel", "sycophantic"} {
		if !strings.Contains(got.Prompt, want) {
			t.Errorf("the prompt does not carry %q:\n%s", want, got.Prompt)
		}
	}
	// Counts and recency travel with the word: "looked up three times over three
	// weeks" and "looked up once" support different claims about a learner.
	if !strings.Contains(got.Prompt, "3") || !strings.Contains(got.Prompt, "2026-08-05") {
		t.Errorf("counts or dates did not reach the prompt:\n%s", got.Prompt)
	}
}

// The two rules checkEvidence then ENFORCES are stated in the prompt, because a
// model told them produces fewer dropped claims. Saying is not guaranteeing —
// that is why the check exists — but not saying is a guaranteed cost.
func TestRenderReflectPromptStatesTheRulesTheCheckerEnforces(t *testing.T) {
	got := renderReflectPrompt(sampleEvidence())
	sys := strings.ToLower(got.System)

	// Asserted on the FIELD NAME and the words-not-phrases rule, because those
	// are what steer the answer: the first version matched the keyword "only"
	// and went red when the rule was reworded to be clearer, which is a test of
	// the wording rather than of the claim.
	//
	// The rewording was itself the fix for a measured failure: the model put
	// whole sentences in the evidence array and lost the level claim to the
	// deck check every run.
	for _, want := range []string{"evidence_words", "exactly", "list"} {
		if !strings.Contains(sys, want) {
			t.Errorf("the system prompt does not state the citation rule (%q missing): %q", want, got.System)
		}
	}
	if !strings.Contains(sys, "rationale") {
		t.Errorf("the prompt does not say where explanation goes instead: %q", got.System)
	}
	if !strings.Contains(sys, "authoring") && !strings.Contains(sys, "practice") {
		t.Errorf("the system prompt does not ask what to DO about a domain: %q", got.System)
	}
}

func TestReflectRequestNamesItsTask(t *testing.T) {
	if got := renderReflectPrompt(sampleEvidence()); got.Task != reflectTaskName {
		t.Errorf("Task = %q, want %q", got.Task, reflectTaskName)
	}
}

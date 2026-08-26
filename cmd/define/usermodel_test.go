package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/llm/llmtest"
)

func sampleLearnerModel() learnerModel {
	return learnerModel{
		Level: levelClaim{
			Band:      "C1",
			Rationale: "Reaches for precise low-frequency words rather than looking up common ones.",
			Evidence:  []string{"certiorari", "estoppel", "sycophantic"},
		},
		Domains: []domainClaim{
			{
				Name: "law", Share: 0.42,
				Evidence:  []string{"certiorari", "estoppel", "dicta"},
				Directive: "Draw comparables from judicial prose; a legal register is familiar ground.",
			},
			{
				Name: "business news", Share: 0.28,
				Evidence:  []string{"sycophantic", "ephemeral"},
				Directive: "Prefer corporate-governance usages when a word has one.",
			},
		},
	}
}

func sampleMeta() modelMeta {
	return modelMeta{
		Updated:   time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
		From:      time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		To:        time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC),
		Lookups:   84,
		Questions: 6,
		Model:     "claude-opus-5",
	}
}

func TestRenderUserModel(t *testing.T) {
	assertGoldenFile(t, "testdata/golden/user-model.md", renderUserModel(sampleLearnerModel(), sampleMeta()))
}

// The issue's rule, asserted over the OUTPUT because that is where a reader
// checks it: "a claim that cannot name the events behind it does not go in the
// file". checkEvidence guarantees the evidence EXISTS; this guarantees it is
// SHOWN, which is what makes the claim checkable by a person.
func TestRenderUserModelShowsEvidenceForEveryClaim(t *testing.T) {
	got := renderUserModel(sampleLearnerModel(), sampleMeta())

	for _, want := range []string{"certiorari", "estoppel", "dicta", "sycophantic", "ephemeral"} {
		if !strings.Contains(got, want) {
			t.Errorf("the rendered file does not name %q", want)
		}
	}
	// And the directive, which is the only reason a domain claim is worth
	// generating: a share without a directive tells authoring nothing.
	if !strings.Contains(got, "judicial prose") {
		t.Errorf("a domain rendered without its authoring directive:\n%s", got)
	}
}

// The human-owned section exists from the FIRST run, so nobody has to know the
// marker's spelling to use it.
func TestRenderUserModelEndsWithTheCorrectionsMarker(t *testing.T) {
	got := renderUserModel(sampleLearnerModel(), sampleMeta())

	if !strings.Contains(got, correctionsMarker) {
		t.Fatalf("no corrections marker:\n%s", got)
	}
	if i := strings.Index(got, correctionsMarker); strings.Contains(got[:i], "## Corrections") {
		t.Error("the marker appears before its own section")
	}
}

// A dropped claim leaves no empty scaffolding behind: an empty "## Level" says
// there IS a level and it is blank, which is a different claim from "we do not
// know yet" — the same distinction #16's prompt sections draw.
func TestRenderUserModelOmitsClaimsThatWereDropped(t *testing.T) {
	got := renderUserModel(learnerModel{}, sampleMeta())

	for _, unwanted := range []string{"## Level", "## Domains"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("rendered an empty %q section:\n%s", unwanted, got)
		}
	}
	// The frontmatter and the human-owned section still stand: the file is
	// valid and says, truthfully, that nothing was inferred.
	if !strings.Contains(got, "type: user-model") || !strings.Contains(got, correctionsMarker) {
		t.Errorf("an empty model must still be a well-formed file:\n%s", got)
	}
}

// assertGoldenFile compares a rendered FILE with its golden.
//
// llmtest.AssertGolden takes an llm.Request and renders it through
// llm.RenderRequest, so it cannot compare a markdown document. This reads
// llmtest.Updating() rather than registering a second -update flag: two flags of
// that name in one test binary is a panic at init, and one of them refreshing
// half the artifacts is worse than either.
func assertGoldenFile(t *testing.T, path, got string) {
	t.Helper()
	if llmtest.Updating() {
		if err := os.MkdirAll("testdata/golden", 0o755); err != nil {
			t.Fatalf("golden: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("golden: %v", err)
		}
		t.Logf("golden: wrote %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden: %v\n\nit renders as:\n%s\nrun with -update to record it", err, got)
	}
	if string(want) != got {
		t.Errorf("golden %s differs:\n--- want ---\n%s\n--- got ---\n%s", path, want, got)
	}
}

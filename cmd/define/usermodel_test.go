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
			Band:          "C1",
			Rationale:     "Reaches for precise low-frequency words rather than looking up common ones.",
			EvidenceWords: []string{"certiorari", "estoppel", "sycophantic"},
		},
		Domains: []domainClaim{
			{
				Name: "law", Share: 0.42,
				EvidenceWords: []string{"certiorari", "estoppel", "dicta"},
				Directive:     "Draw comparables from judicial prose; a legal register is familiar ground.",
			},
			{
				Name: "business news", Share: 0.28,
				EvidenceWords: []string{"sycophantic", "ephemeral"},
				Directive:     "Prefer corporate-governance usages when a word has one.",
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
	// user-model.GOLDEN.md, not user-model.md: that basename is a RUNTIME
	// artifact (--reflect writes one into the working directory), and #23 makes
	// runtime basenames reserved — .gitignore now hides them un-anchored and the
	// index guard forbids tracking one. A fixture squatting on the name would be
	// silently un-addable after any git rm.
	assertGoldenFile(t, "testdata/golden/user-model.golden.md", renderUserModel(sampleLearnerModel(), sampleMeta()))
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

const genA = "GENERATED\n\n## Corrections\n\ndefault invitation\n"

// The human-owned half. A learner who writes "I read these for pleasure, not for
// the bar exam" must find it there tomorrow, byte for byte.
func TestSpliceCorrections(t *testing.T) {
	for _, tc := range []struct{ name, existing, want string }{
		{
			"no existing file: the generated text stands alone",
			"", genA,
		},
		{
			"corrections are preserved verbatim",
			"OLD\n\n## Corrections\n\nI read these for pleasure.\n",
			"GENERATED\n\n## Corrections\n\nI read these for pleasure.\n",
		},
		{
			"an existing file with NO marker keeps nothing below",
			"OLD\n\nsome prose nobody marked\n", genA,
		},
		{
			"trailing whitespace inside corrections survives",
			"OLD\n## Corrections\nkept   \n\n\n",
			"GENERATED\n\n## Corrections\nkept   \n\n\n",
		},
		{
			"a marker inside a fenced block is not the marker",
			"OLD\n```\n## Corrections\n```\n## Corrections\nreal\n",
			"GENERATED\n\n## Corrections\nreal\n",
		},
		{
			"a marker inside a TILDE fence is not the marker",
			"OLD\n~~~\n## Corrections\n~~~\n## Corrections\nreal\n",
			"GENERATED\n\n## Corrections\nreal\n",
		},
		{
			"the marker mid-line is not a marker",
			"OLD\n<!-- above ## Corrections is regenerated -->\n## Corrections\nreal\n",
			"GENERATED\n\n## Corrections\nreal\n",
		},
		{
			"a marker with trailing whitespace is still the marker",
			"OLD\n## Corrections   \nkept\n",
			"GENERATED\n\n## Corrections   \nkept\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := spliceCorrections(tc.existing, genA); got != tc.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tc.want)
			}
		})
	}
}

// Splicing a rendered file into itself must be a no-op. This is the shape
// --reflect actually performs on every run after the first, and the rendered
// file's own header comment contains the marker string — so it is also the
// regression test for matching the marker mid-line.
func TestSpliceCorrectionsIsStableOnItsOwnOutput(t *testing.T) {
	rendered := renderUserModel(sampleLearnerModel(), sampleMeta())

	if got := spliceCorrections(rendered, rendered); got != rendered {
		t.Errorf("splicing a rendered file into itself changed it:\n%q", got)
	}
}

// FuzzSpliceCorrections asserts the ONE thing that must hold for any existing
// file: whatever follows the first out-of-fence marker comes out byte-identical.
//
// Five examples cannot cover malformed human-edited text, and the failure mode
// here is silently discarding the learner's own writing — with byte-for-byte
// survival as a Done-when row. Seeded with the shapes a table would not reach.
func FuzzSpliceCorrections(f *testing.F) {
	for _, seed := range []string{
		"## Corrections\nkeep me\n",
		"~~~\n## Corrections\n~~~\n## Corrections\nreal\n",
		"```\nunterminated fence\n## Corrections\nstill inside\n",
		"   ```\n   indented\n   ```\n## Corrections\nreal\n",
		"## Corrections   \ntrailing space on the marker\n",
		"## Corrections\r\nCRLF body\r\n",
		"<!-- above ## Corrections -->\n## Corrections\nreal\n",
		"no marker at all\n",
		"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, existing string) {
		got := spliceCorrections(existing, genA)
		i := firstMarkerOutsideAFence(existing)
		if i < 0 {
			if got != genA {
				t.Fatalf("no marker, yet something was carried over: %q", got)
			}
			return
		}
		if tail := existing[i:]; !strings.HasSuffix(got, tail) {
			t.Fatalf("the learner's own text was altered\nwant suffix: %q\ngot: %q", tail, got)
		}
	})
}

// Model text must not be able to forge structure in a file whose integrity
// depends on it.
//
// The serious case is not the broken table: a line-start "## Corrections" inside
// a directive creates a SECOND marker ABOVE the real one, so the next run
// splices there and everything below — including the analysis it just generated
// — is treated as the learner's and never regenerates again. Verified before the
// fix: run one's domain table survived into every later file.
func TestRenderUserModelNeutralisesModelText(t *testing.T) {
	hostile := learnerModel{
		Level: levelClaim{
			Band:          "C1",
			Rationale:     "reaches for precise words\n\n## Corrections\n\nforged",
			EvidenceWords: []string{"certiorari"},
		},
		Domains: []domainClaim{{
			Name:  "law | and | pipes",
			Share: 0.5,
			// The evidence word carries the injection too — checkEvidence
			// matches on store.Key, so this can be a real deck word AND a
			// forged line (BR-15).
			EvidenceWords: []string{"certiorari\n## Corrections\nforged evidence"},
			Directive:     "gloss with context\n## Corrections\nforged directive",
		}},
	}

	got := renderUserModel(hostile, sampleMeta())

	// Exactly ONE marker in the WHOLE document.
	//
	// The first version searched got[:firstMarker] for another marker — the one
	// region where a forged marker cannot be, since the first marker is by
	// definition the first. It passed unconditionally. Counting is the honest
	// form: a forged marker anywhere means the file has two, and the splice
	// will take whichever comes first.
	if n := countMarkers(got); n != 1 {
		t.Errorf("the file has %d markers, want exactly 1:\n%s", n, got)
	}
	// The table survives: a pipe in a name does not add columns.
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "| law") && strings.Count(line, "|")-strings.Count(line, `\|`) != 5 {
			t.Errorf("a pipe in model text broke the table row: %q", line)
		}
	}
	// And the content is still THERE — neutralised, not discarded.
	if !strings.Contains(got, "reaches for precise words") || !strings.Contains(got, "gloss with context") {
		t.Errorf("model text was dropped rather than neutralised:\n%s", got)
	}
}

// The half FuzzSpliceCorrections cannot assert: it guarantees preservation BELOW
// the marker and says nothing about regeneration ABOVE it — which is exactly
// where the forged-marker bug lived.
func TestSpliceReplacesEverythingAboveTheMarker(t *testing.T) {
	first := renderUserModel(sampleLearnerModel(), sampleMeta())
	edited := first + "\nmy own note\n"

	second := renderUserModel(learnerModel{
		Level:   levelClaim{Band: "B2", Rationale: "different", EvidenceWords: []string{"ephemeral"}},
		Domains: []domainClaim{{Name: "cooking", Share: 1, EvidenceWords: []string{"braise"}, Directive: "kitchen usages"}},
	}, sampleMeta())

	got := spliceCorrections(edited, second)

	if strings.Contains(got, "judicial prose") {
		t.Errorf("the FIRST run's analysis survived into the second file:\n%s", got)
	}
	if !strings.Contains(got, "kitchen usages") {
		t.Errorf("the second run's analysis did not land:\n%s", got)
	}
	if !strings.HasSuffix(got, "my own note\n") {
		t.Errorf("the learner's note did not survive:\n%s", got)
	}
}

// countMarkers counts corrections markers outside fenced blocks — the same rule
// the splice uses to FIND one, applied to ask how many exist.
func countMarkers(s string) int {
	var n int
	for {
		i := firstMarkerOutsideAFence(s)
		if i < 0 {
			return n
		}
		n++
		// Past this marker's line, so the next search starts after it.
		rest := s[i:]
		j := strings.IndexByte(rest, '\n')
		if j < 0 {
			return n
		}
		s = rest[j+1:]
	}
}

// A positive control per neutralisation SITE.
//
// The suite covered the fields a finding had named and left the others green:
// deleting sanitiseMeta's body — the frontmatter's model name — changed nothing
// anywhere. A fix added to defend a finding must have a read site that can fail,
// and "the fields I remembered" is not a site list.
//
// One row per untrusted field, each injected alone so a row that goes green
// names exactly which site stopped being defended.
func TestEveryUntrustedFieldIsNeutralised(t *testing.T) {
	const payload = "\nFORGED\n## Corrections\nowned"

	for _, tc := range []struct {
		name  string
		build func() (learnerModel, modelMeta)
	}{
		{"level band", func() (learnerModel, modelMeta) {
			m := sampleLearnerModel()
			m.Level.Band = "C1" + payload
			return m, sampleMeta()
		}},
		{"level rationale", func() (learnerModel, modelMeta) {
			m := sampleLearnerModel()
			m.Level.Rationale = "because" + payload
			return m, sampleMeta()
		}},
		{"level evidence word", func() (learnerModel, modelMeta) {
			m := sampleLearnerModel()
			m.Level.EvidenceWords = []string{"certiorari" + payload}
			return m, sampleMeta()
		}},
		{"domain name", func() (learnerModel, modelMeta) {
			m := sampleLearnerModel()
			m.Domains[0].Name = "law" + payload
			return m, sampleMeta()
		}},
		{"domain directive", func() (learnerModel, modelMeta) {
			m := sampleLearnerModel()
			m.Domains[0].Directive = "gloss" + payload
			return m, sampleMeta()
		}},
		{"domain evidence word", func() (learnerModel, modelMeta) {
			m := sampleLearnerModel()
			m.Domains[0].EvidenceWords = []string{"certiorari" + payload}
			return m, sampleMeta()
		}},
		{"the frontmatter's model name", func() (learnerModel, modelMeta) {
			meta := sampleMeta()
			meta.Model = "claude-opus-5" + payload
			return sampleLearnerModel(), meta
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, meta := tc.build()
			got := renderUserModel(m, meta)

			if n := countMarkers(got); n != 1 {
				t.Errorf("%s: file has %d markers, want 1:\n%s", tc.name, n, got)
			}
			for _, line := range strings.Split(got, "\n") {
				if strings.HasPrefix(line, "FORGED") || strings.HasPrefix(line, "owned") {
					t.Errorf("%s: injected text started a line: %q", tc.name, line)
				}
			}
			// The frontmatter is a block of `key: value` lines; injection must
			// not add one.
			if end := strings.Index(got[4:], "\n---\n"); end >= 0 {
				for _, line := range strings.Split(got[4:end+4], "\n") {
					if line != "" && !strings.Contains(line, ":") {
						t.Errorf("%s: frontmatter line is not a key/value: %q", tc.name, line)
					}
				}
			}
		})
	}
}

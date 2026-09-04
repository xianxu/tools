package main

import (
	"math"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

func mustDomain(t *testing.T, s string) store.Domain {
	t.Helper()
	d, ok := store.ParseDomain(s)
	if !ok {
		t.Fatalf("ParseDomain(%q) refused", s)
	}
	return d
}

func banded(word string, b store.Band, d store.Domain) bandedWord {
	return bandedWord{Word: word, Facts: store.WordFacts{Band: b, Domain: d, At: harvestClock}}
}

// The learner-side fold: #17's free model text onto the closed set.
func TestParseLearnerDomains(t *testing.T) {
	const md = `---
type: user-model
level: C1
---

## Domains they read in

| domain | share | read off | what to do about it |
|---|---|---|---|
| law | 42% | ` + "`certiorari`" + ` | Draw comparables from judicial prose. |
| business news | 28% | ` + "`sycophantic`" + ` | Prefer corporate-governance usages. |
| MEDICINE | 10% | ` + "`sepsis`" + ` | Clinical register. |

## Corrections

| domain | share | read off | what to do about it |
| Astrology | 99% | none | this is a person writing their own table |
`
	got := parseLearnerDomains(md)

	// `law` folds case-insensitively onto NOAD's `Law`.
	// `business news` is a real thing this learner reads and NOAD prints no such
	// field label, so it is IGNORED rather than becoming a new domain — the
	// narrowing the function's comment records.
	// `MEDICINE` folds too, which is the same rule in a different casing.
	want := []store.Domain{mustDomain(t, "Law"), mustDomain(t, "Medicine")}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("domain %d = %q, want %q", i, got[i], want[i])
		}
	}

	// The human-owned section must not reach selection. #17 promises never to
	// rewrite ## Corrections, so reading a table there would let the one
	// un-generated region change what the generated region means.
	for _, d := range got {
		if strings.EqualFold(string(d), "Astrology") {
			t.Error("a table in ## Corrections reached the domain fold")
		}
	}
}

func TestParseLearnerDomainsOnAnAbsentOrThinModel(t *testing.T) {
	for _, tc := range []struct{ name, in string }{
		{"no model at all", ""},
		{"a model with no domains section", "---\ntype: user-model\n---\n\n## Level\n\n**C1** — prose.\n"},
		{"an empty table", "## Domains they read in\n\n| domain | share |\n|---|---|\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseLearnerDomains(tc.in); len(got) != 0 {
				t.Errorf("got %v, want none — an absent model means generic authoring", got)
			}
		})
	}
}

// THE SELECTION RULE, which is the whole of "distractors are selected, never
// invented": same domain, at the learner's band or one below, never the answer.
func TestPickDistractorsHoldsTheRule(t *testing.T) {
	law, med := mustDomain(t, "Law"), mustDomain(t, "Medicine")
	target := store.WordFacts{Band: store.C1, Domain: law, At: harvestClock}
	pool := []bandedWord{
		banded("certiorari", store.C1, law),  // same domain, at band
		banded("estoppel", store.B2, law),    // same domain, one below
		banded("laches", store.A1, law),      // same domain, FAR below
		banded("sepsis", store.C1, med),      // wrong domain
		banded("sycophantic", store.C1, law), // the answer itself
	}

	got, tier := pickDistractors("sycophantic", target, store.C1, sortedBanded(pool), 2, 7, nil)

	if tier != tierSameDomain {
		t.Errorf("tier = %v, want same-domain — the pool can supply it", tier)
	}
	if len(got) != 2 {
		t.Fatalf("got %v, want exactly the two admissible words", got)
	}
	for _, w := range got {
		switch w {
		case "certiorari", "estoppel": // at band, one below
		case "sycophantic":
			t.Error("THE ANSWER was offered as its own distractor")
		case "laches":
			t.Error("an A1 word was offered to a C1 learner — unrejectable by ignorance, not by knowledge")
		case "sepsis":
			t.Error("a Medicine word was offered beside a Law answer; three obviously-medical " +
				"options beside one legal one give the answer away")
		}
	}
}

// One band BELOW, never above. A distractor the learner does not know is
// eliminated by ignorance rather than by knowing it does not fit, which teaches
// nothing — so a word ABOVE their band is worse than useless.
func TestPickDistractorsNeverReachesAboveTheLearner(t *testing.T) {
	law := mustDomain(t, "Law")
	target := store.WordFacts{Band: store.B1, Domain: law, At: harvestClock}
	pool := []bandedWord{
		banded("above-one", store.B2, law),
		banded("above-two", store.C1, law),
		banded("above-three", store.C2, law),
	}
	got, tier := pickDistractors("x", target, store.B1, sortedBanded(pool), 3, 1, nil)
	// Nothing at or below B1 exists, so selection must WIDEN and say so rather
	// than reaching up.
	if tier != tierAboveBand {
		t.Errorf("tier = %v, want the above-band tier — the deck has nothing at or below level", tier)
	}
	if len(got) == 0 {
		t.Error("widening produced nothing; a question with no wrong answers is not a question")
	}
}

// Widening is TIERED and REPORTED. A selector that silently falls back to "any
// word at all" is indistinguishable from one that is working.
func TestPickDistractorsWidensThroughGeneralAndSaysSo(t *testing.T) {
	law := mustDomain(t, "Law")
	target := store.WordFacts{Band: store.C1, Domain: law, At: harvestClock}
	pool := []bandedWord{
		banded("ordinary-one", store.C1, store.DomainGeneral),
		banded("ordinary-two", store.B2, store.DomainGeneral),
	}
	got, tier := pickDistractors("x", target, store.C1, sortedBanded(pool), 3, 1, nil)
	if tier != tierGeneral {
		t.Errorf("tier = %v, want general — no same-domain word exists but level-matched general ones do", tier)
	}
	if len(got) != 2 {
		t.Errorf("got %v, want both general words", got)
	}
	if !strings.Contains(tierAboveBand.String(), "could not supply") {
		t.Error("the widest tier does not explain itself; a person reading a batch must be able to tell")
	}
}

// With no learner model, selection falls back to the ANSWER's own band. Recorded
// because it is a real decision: generic authoring has no "the learner's band"
// to be one below of, and pitching at the whole scale would be worse than
// pitching at the word.
func TestPickDistractorsWithoutALearnerBandUsesTheWords(t *testing.T) {
	law := mustDomain(t, "Law")
	target := store.WordFacts{Band: store.B1, Domain: law, At: harvestClock}
	pool := []bandedWord{
		banded("at-band", store.B1, law),
		banded("one-below", store.A2, law),
		banded("far-above", store.C2, law),
	}
	got, tier := pickDistractors("x", target, "", sortedBanded(pool), 2, 3, nil)
	if tier != tierSameDomain {
		t.Errorf("tier = %v, want same-domain", tier)
	}
	if len(got) != 2 {
		t.Fatalf("got %v, want the two at or one below the WORD's band", got)
	}
	for _, w := range got {
		if w == "far-above" {
			t.Error("a C2 word was selected against a B1 answer with no learner model")
		}
	}
}

// Deterministic under a seed, because authoring runs offline and a batch that
// cannot be reproduced cannot be debugged.
func TestPickDistractorsIsDeterministic(t *testing.T) {
	law := mustDomain(t, "Law")
	target := store.WordFacts{Band: store.C1, Domain: law, At: harvestClock}
	var pool []bandedWord
	for _, w := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		pool = append(pool, banded(w, store.C1, law))
	}
	pool = sortedBanded(pool)

	first, _ := pickDistractors("x", target, store.C1, pool, 3, 42, nil)
	second, _ := pickDistractors("x", target, store.C1, pool, 3, 42, nil)
	if strings.Join(first, ",") != strings.Join(second, ",") {
		t.Errorf("same seed gave %v then %v", first, second)
	}
	// And a DIFFERENT seed varies, or the shuffle is not doing its job — a pool
	// walked in fixed order hands every question the same options and the learner
	// answers by elimination.
	varied := false
	for s := uint64(1); s < 40 && !varied; s++ {
		other, _ := pickDistractors("x", target, store.C1, pool, 3, s, nil)
		if strings.Join(other, ",") != strings.Join(first, ",") {
			varied = true
		}
	}
	if !varied {
		t.Error("every seed selected the same three of eight equally-eligible words")
	}
}

// The prompt carries both REQUIREMENTS, and the learner model verbatim.
func TestAuthorPromptGolden(t *testing.T) {
	law := mustDomain(t, "Law")
	llmtest.AssertGolden(t, "testdata", "author-prompt", renderAuthorPrompt(
		store.DefaultLang, "certiorari",
		"Law a writ by which a higher court reviews a lower court's decision",
		store.WordFacts{Band: store.C2, Domain: law, At: harvestClock},
		learnerFacts{
			Band:    store.C1,
			Domains: []store.Domain{law},
			Model:   "## Level\n\n**C1** — Reaches for precise low-frequency words.\n\n## Domains they read in\n\n| domain | share | read off | what to do about it |\n|---|---|---|---|\n| law | 42% | `certiorari` | Draw comparables from judicial prose. |\n",
		}))
}

func TestAuthorPromptWithNoLearnerModelGolden(t *testing.T) {
	llmtest.AssertGolden(t, "testdata", "author-prompt-generic", renderAuthorPrompt(
		store.DefaultLang, "ephemeral", "lasting for a very short time",
		store.WordFacts{Band: store.C1, Domain: store.DomainGeneral, At: harvestClock},
		learnerFacts{}))
}

// All THREE requirements are REQUIREMENTS in the prompt, not preferences, and
// each carries its counter-example. Measured twice: asked for a natural sentence
// the model drifts to the neutral and unnamed, and asked to make a sentence
// entail a word it defines the word (the M2 checkpoint's finding).
func TestAuthorPromptStatesEveryRequirement(t *testing.T) {
	got := renderAuthorPrompt(store.DefaultLang, "x", "", store.WordFacts{}, learnerFacts{}).Prompt
	for _, want := range []string{"POINT AT the word", "Never gloss the word", "Name real people", "was noted by all"} {
		if !strings.Contains(got, want) {
			t.Errorf("the prompt never says %q", want)
		}
	}
	// An absent learner model means GENERIC authoring, not an empty section
	// header the model has to interpret.
	if strings.Contains(got, "## The learner") {
		t.Error("an absent learner model still rendered its section")
	}
}

// topicSpread, and the one thing that could make it lie.
func TestTopicSpread(t *testing.T) {
	law, med := mustDomain(t, "Law"), mustDomain(t, "Medicine")
	for _, tc := range []struct {
		name string
		in   []store.Domain
		want float64
	}{
		{"empty is zero, not perfect variety", nil, 0},
		{"all distinct", []store.Domain{law, med, store.DomainGeneral}, 1},
		{"all one domain", []store.Domain{law, law, law}, 1.0 / 3.0},
		{"half and half", []store.Domain{law, law, med, med}, 0.5},
		// THE FAILURE THIS MEASURE HAS. Counting raw values would report three
		// distinct domains here where there is one, and topicSpread is the one
		// measure taken with no model — nothing else can catch it being wrong.
		{"casing variants collapse", []store.Domain{"Medicine", "medicine", "MEDICINE"}, 1.0 / 3.0},
		{"whitespace variants collapse", []store.Domain{"Law", " Law ", "law"}, 1.0 / 3.0},
		// A value outside the closed set is `general`, not a fourth domain.
		{"unknown values are general, not new", []store.Domain{"Astrology", "Crypto", store.DomainGeneral}, 1.0 / 3.0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := topicSpread(tc.in); math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("topicSpread(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// blankOut hides the answer from the VETO, or every candidate looks wrong beside
// a sentence that already contains the right one.
func TestBlankOut(t *testing.T) {
	for _, tc := range []struct{ stem, answer, want string }{
		{"The aide was sycophantic to a fault.", "sycophantic", "The aide was ___ to a fault."},
		{"Sycophantic aides surrounded him.", "sycophantic", "___ aides surrounded him."},
		// Not present: return the stem rather than mangling it. #12 owns real
		// blanking — stem, plural, hyphenation — where the LEARNER must not see
		// the answer; here a miss only costs the judge some context.
		{"No such word here.", "sycophantic", "No such word here."},
	} {
		if got := blankOut(tc.stem, tc.answer); got != tc.want {
			t.Errorf("blankOut(%q, %q) = %q, want %q", tc.stem, tc.answer, got, tc.want)
		}
	}
}

// DIVERSITY PRESSURE, measured into existence by the first real batch: over 20
// words `ephemeral` served as a wrong answer in 8 items, and the four A1 words
// selected each other in all four of theirs. A learner who meets one word as a
// wrong answer eight times learns it is never the answer.
func TestPickDistractorsSpreadsAcrossABatch(t *testing.T) {
	law := mustDomain(t, "Law")
	var pool []bandedWord
	for _, w := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		pool = append(pool, banded(w, store.C1, law))
	}
	pool = sortedBanded(pool)
	target := store.WordFacts{Band: store.C1, Domain: law, At: harvestClock}

	served := map[string]int{}
	for _, answer := range []string{"a", "b", "c", "d", "e", "f"} {
		got, _ := pickDistractors(answer, target, store.C1, pool, 3, seedFor("t", answer), served)
		for _, w := range got {
			served[store.Key(w)]++
		}
	}

	worst := 0
	for _, n := range served {
		if n > worst {
			worst = n
		}
	}
	// Six items x three options = 18 slots over 10 words. Perfectly even is 1.8,
	// so 3 leaves real slack for the eligibility constraints; without pressure the
	// same word took 6 of 6.
	if worst > 3 {
		t.Errorf("one word served as a distractor %d times across 6 items (counts %v); "+
			"the batch is leaning on whichever words happen to be eligible", worst, served)
	}
	// And the pressure must not COST coverage: every item still got its options.
	if len(served) < 6 {
		t.Errorf("only %d distinct words were used across the batch: %v", len(served), served)
	}
}

// A cap would refuse to fill an option set on a small deck, and fewer options is
// a worse question than a repeated one. Pressure ORDERS, it does not exclude.
func TestDiversityPressureNeverStarvesAnItem(t *testing.T) {
	law := mustDomain(t, "Law")
	pool := sortedBanded([]bandedWord{
		banded("only-one", store.C1, law),
		banded("only-two", store.C1, law),
	})
	target := store.WordFacts{Band: store.C1, Domain: law, At: harvestClock}
	// Both candidates already heavily used; the item must still be filled.
	served := map[string]int{"only-one": 99, "only-two": 99}
	got, _ := pickDistractors("x", target, store.C1, pool, 3, 1, served)
	if len(got) != 2 {
		t.Errorf("got %v, want both candidates — pressure orders, it must not exclude", got)
	}
}

// A nil map is no pressure, which is what a single-item run and every other
// unit test wants — and it must not change the seeded order.
func TestDiversityPressureIsOptional(t *testing.T) {
	law := mustDomain(t, "Law")
	var pool []bandedWord
	for _, w := range []string{"a", "b", "c", "d", "e"} {
		pool = append(pool, banded(w, store.C1, law))
	}
	pool = sortedBanded(pool)
	target := store.WordFacts{Band: store.C1, Domain: law, At: harvestClock}

	withNil, _ := pickDistractors("x", target, store.C1, pool, 3, 9, nil)
	withEmpty, _ := pickDistractors("x", target, store.C1, pool, 3, 9, map[string]int{})
	if strings.Join(withNil, ",") != strings.Join(withEmpty, ",") {
		t.Errorf("nil gave %v and an empty map gave %v; an unused map must not reorder", withNil, withEmpty)
	}
}

// The free check, added by the second checkpoint batch: it produced
// "...has stood atop the narrow ___ of First Mesa" for the word `mesa` — the
// model blanked the word itself against an explicit instruction, and both judges
// passed the item because neither was asked.
func TestStemUsesTheWord(t *testing.T) {
	for _, tc := range []struct {
		name, stem, word string
		want             bool
	}{
		{"present", "Shipwrights laid the keel of the frigate.", "keel", true},
		{"inflected", "Domtar's mills shipped tonnes of pulp.", "mill", true},
		{"capitalised", "Mesa country begins north of Flagstaff.", "mesa", true},
		// THE CASE THAT SHIPPED, and it is subtler than it looks: this stem DOES
		// contain `mesa`, in the place name "First Mesa" — so a containment check
		// alone passes it. What is wrong is the blank the model inserted against
		// an explicit instruction, which #12 would then blank again or not at all.
		{"the model blanked it itself", "The village stood atop the narrow ___ of First Mesa.", "mesa", false},
		{"blanked and absent", "A sentence with a ___ in it.", "quokka", false},
		{"absent entirely", "A sentence about nothing in particular.", "quokka", false},
		// A substring is not the word. `set` must not be satisfied by `sunset`,
		// or a stem the learner cannot answer would pass the only free check.
		{"substring at the end", "The sun dipped below the horizon at sunset.", "set", false},
		{"substring inside", "They met for brunch in Tromso.", "run", false},
		{"empty word", "Anything at all.", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := stemUsesTheWord(tc.stem, tc.word); got != tc.want {
				t.Errorf("stemUsesTheWord(%q, %q) = %v, want %v", tc.stem, tc.word, got, tc.want)
			}
		})
	}
}

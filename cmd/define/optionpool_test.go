package main

import (
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/play"
)

// countingDict wraps a dictionary and counts lookups.
//
// The ARCH-CONSTRAINTS budget was declared and implemented and enforced by
// nothing: the quadratic regression play_loop.go's own comment warns about —
// one lookup per deck word per DUE word — would have been caught by no test,
// and neither would poolCap being bypassed. A counting seam is the only way a
// cost claim can go red.
type countingDict struct {
	inner   Dictionary
	lookups int
	words   []string
}

func (c *countingDict) Lookup(w string) (string, error) {
	c.lookups++
	c.words = append(c.words, w)
	return c.inner.Lookup(w)
}

// THE COST ENVELOPE. One lookup per POOL word, capped, plus one per DUE word.
func TestSittingCostIsBoundedByTheCap(t *testing.T) {
	// A deck far larger than the cap. Most of these are not in the corpus, and
	// that is fine — a failed lookup still COSTS a lookup, which is the thing
	// being bounded.
	var words []string
	for _, real := range []string{"sycophantic", "ephemeral", "quokka", "mesa", "parrot", "concrete", "pulp", "minute"} {
		words = append(words, real)
	}
	for i := 0; i < 120; i++ {
		words = append(words, "filler"+string(rune('a'+i%26))+string(rune('a'+i/26)))
	}
	d, opt, _ := playRig(t, words...)
	counting := &countingDict{inner: d.dict}
	d.dict = counting
	opt.count = 5

	qs, code := todaysQuestions(d, opt, &strings.Builder{}, &strings.Builder{})
	if code != 0 {
		t.Fatalf("todaysQuestions exit %d", code)
	}
	// The budget: the pool (capped) plus one per due word.
	if max := poolCap + opt.count; counting.lookups > max {
		t.Errorf("a sitting over a %d-word deck cost %d dictionary lookups, want at most %d "+
			"(poolCap %d + %d due) — the pool is being walked whole, or built per question",
			len(words), counting.lookups, max, poolCap, opt.count)
	}
	// A BOUND ALONE IS SATISFIED BY ZERO. Counting lookups proves the cost is
	// capped and nothing else — a build that skipped the pool entirely would
	// pass it while silently disabling form 2.3 for every learner. So the
	// OUTPUT is asserted too: the sitting must actually contain a form 2.3
	// question, which is only possible if the pool was populated.
	if counting.lookups < poolCap {
		t.Errorf("only %d lookups — the pool is not being built at all", counting.lookups)
	}
	if len(qs) == 0 {
		t.Fatal("no questions")
	}
	choices := 0
	for _, q := range qs {
		if _, ok := q.(*play.Choice); ok {
			choices++
		}
	}
	if choices == 0 {
		t.Errorf("%d questions and not one is form 2.3 — the pool was capped to nothing, "+
			"which the lookup bound above cannot distinguish from working", len(qs))
	}
}

// D4a: at most one sense per axis, and the general one is the FIRST usable
// sense of the first block.
func TestOptionCandidatesPicksOneSensePerAxis(t *testing.T) {
	d := testDict(t)
	raw, err := d.Lookup("record")
	if err != nil {
		// NOT a skip: the corpus is COMMITTED, so a missing entry is a broken
		// fixture rather than an absent dependency. Skipping would make this
		// test disappear the day someone trims testdata.
		t.Fatalf("record is not in the committed corpus: %v", err)
	}
	got := optionCandidates("record", ParseEntry(raw))
	if len(got) == 0 {
		t.Fatal("no candidates for a rich entry")
	}
	seen := map[play.Axis]int{}
	for _, c := range got {
		seen[c.Axis]++
		if c.Word != "record" {
			t.Errorf("candidate carries word %q", c.Word)
		}
		if c.Gloss == "" || strings.HasPrefix(c.Gloss, "[") || strings.HasPrefix(c.Gloss, "(") {
			t.Errorf("candidate gloss %q still carries the apparatus, or is empty", c.Gloss)
		}
	}
	for axis, n := range seen {
		if n > 1 {
			t.Errorf("%d candidates for axis %v; one word may occupy one slot", n, axis)
		}
	}
	// `record` carries `Law` and `Computing` senses, so a domain candidate must
	// be among them — the axis machinery working on real NOAD text.
	if seen[play.AxisDomain] == 0 {
		t.Errorf("no domain candidate from `record`, which NOAD labels Law and Computing: %+v", got)
	}
}

// An entry whose senses are ALL cross-references cannot be an answer, and the
// caller must fall back rather than ask an unanswerable question.
func TestTargetCandidateRefusesAnEntryWithNoDefinition(t *testing.T) {
	d := testDict(t)
	for _, tc := range []struct {
		word string
		want bool
	}{
		// `bases` is "plural form of base1" / "/ˈbāsēz/ plural form of basis" —
		// every sense a cross-reference.
		{"bases", false},
		{"sycophantic", true},
		{"mesa", true},
	} {
		raw, err := d.Lookup(tc.word)
		if err != nil {
			t.Fatalf("%s is not in the committed corpus: %v", tc.word, err)
		}
		_, ok := targetCandidate(tc.word, ParseEntry(raw))
		if ok != tc.want {
			t.Errorf("targetCandidate(%q) ok = %v, want %v", tc.word, ok, tc.want)
		}
	}
}

// choiceFor's two refusals, both ordinary states rather than errors.
func TestChoiceForRefusesWhenItMust(t *testing.T) {
	d := testDict(t)
	raw, _ := d.Lookup("sycophantic")
	entry := ParseEntry(raw)
	rich := []play.Candidate{
		{Word: "mesa", Gloss: "an isolated flat-topped hill", Axis: play.AxisGeneral},
		{Word: "quokka", Gloss: "a small wallaby", Axis: play.AxisGeneral},
	}
	if choiceFor("sycophantic", "", entry, rich, 1) == nil {
		t.Error("refused a word with a usable definition and two distractors")
	}
	if choiceFor("sycophantic", "", entry, nil, 1) != nil {
		t.Error("built a question with an empty pool")
	}
	// A pool holding only the target itself: a word is never its own distractor.
	self := []play.Candidate{{Word: "sycophantic", Gloss: "behaving obsequiously", Axis: play.AxisGeneral}}
	if choiceFor("sycophantic", "", entry, self, 1) != nil {
		t.Error("the target was used as its own distractor")
	}
	// D3a at the seam: a near-synonym is excluded, and with nothing else in the
	// pool that leaves no question.
	near := []play.Candidate{{Word: "obsequious", Gloss: "obedient to excess", Axis: play.AxisGeneral}}
	if choiceFor("sycophantic", "", entry, near, 1) != nil {
		t.Error("a candidate NOAD names inside the target's own gloss became an option")
	}
}

// seedFor is GOLDEN for the same reason the PRNG is: the argument for writing
// it out rather than using hash/fnv is that this repo pins it, and until this
// test existed, changing the offset basis left the whole suite green.
func TestSeedForIsPinned(t *testing.T) {
	// A different word or a different DAY must give a
	// different seed, or a sitting repeats itself.
	a := seedFor("bank", "2026-08-30")
	b := seedFor("bank", "2026-08-31")
	c := seedFor("mesa", "2026-08-30")
	if a == b || a == c || b == c {
		t.Errorf("seedFor collided: bank/30=%d bank/31=%d mesa/30=%d", a, b, c)
	}
	if seedFor("bank", "2026-08-30") != a {
		t.Error("seedFor is not deterministic")
	}
	// The parts must not be concatenable into each other: ("ab","c") and
	// ("a","bc") are different questions and must not share a seed.
	if seedFor("ab", "c") == seedFor("a", "bc") {
		t.Error("seedFor ignores the part boundary, so different inputs collide")
	}
	if a != goldenBankSeed {
		t.Errorf("seedFor(\"bank\",\"2026-08-30\") = %d, want %d — the hash constants moved, "+
			"so every question ever recorded now reproduces differently", a, goldenBankSeed)
	}
}

const goldenBankSeed uint64 = 16298003680678406606

// NOAD redirects derived forms to their base headword, and form 2.3 must not
// present the base's definition as the derived word's meaning.
//
// Measured: `bargainer` returns the `bargain` entry, so without this gate the
// question "what does bargainer mean?" offers "an agreement between two or more
// parties…" as the correct answer, records Correct, and promotes the word.
//
// The multi-word rows are the other half. Gating on Headword() alone would send
// every multi-word headword to the fallback, because the head is built from
// fields[0] — "hot" for `hot dog`, "a" for `a priori`.
func TestEntryDefinesTheWordItWasLookedUpFor(t *testing.T) {
	d := testDict(t)
	for _, tc := range []struct {
		word string
		want bool
	}{
		{"bargainer", false}, // returns the `bargain` entry
		{"sycophantic", true},
		{"hot dog", true},  // head is [hot][dog]
		{"a priori", true}, // head is [a][priori][a][pri·o·ri]
		{"jalapeño", true}, // diacritics
		{"bases", true},    // defines itself; it fails LATER, on no usable gloss
		{"gaslighting", true},
		{"mesa", true},
	} {
		raw, err := d.Lookup(tc.word)
		if err != nil {
			t.Fatalf("%s is not in the committed corpus: %v", tc.word, err)
		}
		if got := entryDefines(tc.word, ParseEntry(raw)); got != tc.want {
			t.Errorf("entryDefines(%q) = %v, want %v (headword %q)",
				tc.word, got, tc.want, ParseEntry(raw).Headword())
		}
	}
}

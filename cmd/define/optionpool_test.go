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

	qs, _, code := todaysQuestions(d, opt, &strings.Builder{}, &strings.Builder{})
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

// A redirect must not supply DISTRACTORS either, which is the half round 3
// missed: gating only the target left `bargain`'s glosses in every question,
// labelled `bargainer`, so picking one recorded a choice whose word and meaning
// came from different entries.
func TestARedirectSuppliesNoOptionMaterialAtAll(t *testing.T) {
	d := testDict(t)
	raw, err := d.Lookup("bargainer")
	if err != nil {
		t.Fatalf("bargainer is not in the committed corpus: %v", err)
	}
	e := ParseEntry(raw)

	if got := optionCandidates("bargainer", e); len(got) != 0 {
		t.Errorf("a redirect produced %d distractor candidates: %+v — every one carries "+
			"`bargain`'s meaning under the word `bargainer`", len(got), got)
	}
	if _, ok := targetCandidate("bargainer", e); ok {
		t.Error("a redirect produced a target candidate")
	}
	// And THE SAME ENTRY is perfectly good material for the word it does define
	// — the over-breadth half, which is the whole reason this is a gate and not
	// a ban. One entry, two words, opposite answers: that is the distinction the
	// gate makes, asserted on the one object that can show it.
	//
	// NOT nested under `if err == nil`, which is how the first version wrote it:
	// the corpus is committed, so a lookup failure is a broken fixture, and
	// hiding the only assertion that can detect over-breadth behind a condition
	// no fixture guarantees is how it silently stops running.
	if got := optionCandidates("bargain", e); len(got) == 0 {
		t.Error("the entry supplies no candidates for `bargain`, the word it DOES define; " +
			"the guard is too broad and would send every word to form 2.1")
	}
	if _, ok := targetCandidate("bargain", e); !ok {
		t.Error("no target candidate for `bargain` from its own entry")
	}
}

// The rule stated as a property over the WHOLE corpus, so a fixture added later
// cannot reintroduce the class: no candidate may carry a gloss from an entry
// that does not define its word.
func TestNoCandidateEverCarriesAnotherWordsGloss(t *testing.T) {
	d := testDict(t)
	if len(d.entries) == 0 {
		t.Fatal("empty corpus; this test would be vacuous")
	}
	checked := 0
	for word, raw := range d.entries {
		e := ParseEntry(raw)
		defines := entryDefines(word, e)
		cands := optionCandidates(word, e)
		_, targetOK := targetCandidate(word, e)
		if !defines && (len(cands) > 0 || targetOK) {
			t.Errorf("%q: the entry defines %q, yet it produced %d candidates and target=%v",
				word, e.Headword(), len(cands), targetOK)
		}
		if !defines {
			checked++
		}
	}
	// The corpus must actually CONTAIN the shape, or this test is ranging over
	// nothing. `bargainer` is the measured instance, 1 of 34.
	if checked == 0 {
		t.Error("no redirect entry in the corpus — this test asserted nothing")
	}
}

// The same defect at the SITTING level, on the deck shape that produces it:
// two keys the dictionary answers with one entry.
//
// `define jalapeno` and `define jalapeño` are two Upserts under two keys
// (capture.go), and the accent-insensitive lookup is documented production
// behaviour with its own live conformance check — so this deck is ordinary, not
// contrived.
func TestNoQuestionDrawsTwoOptionsFromOneEntry(t *testing.T) {
	// THE FIXTURE IS THE TEST, and the first one could not produce the defect:
	// jalapeño/jalapeno share an entry with ONE usable sense, which Gloss-dedup
	// already covered, so this test passed with the Source fix removed entirely.
	// `concrete` is the shape that matters — one entry, TWO differently-glossed
	// usable senses ("existing in a material or physical form" and the archaic
	// "form (something) into a mass") — under two deck keys.
	d, opt, _ := playRig(t, "concrete", "cóncrete", "quokka", "mesa", "parrot")
	qs, _ := questionsFor(t, d, opt)
	if len(qs) == 0 {
		t.Fatal("no questions")
	}
	// Map every gloss the deck can produce back to the entry it came from.
	glossSource := map[string]string{}
	for _, w := range []string{"concrete", "cóncrete", "quokka", "mesa", "parrot"} {
		raw, err := d.dict.Lookup(w)
		if err != nil {
			continue
		}
		e := ParseEntry(raw)
		for _, c := range optionCandidates(w, e) {
			glossSource[c.Gloss] = e.Headword()
		}
	}
	choices := 0
	for _, q := range qs {
		c, ok := q.(*play.Choice)
		if !ok {
			continue
		}
		choices++
		seen := map[string]int{}
		for _, o := range c.Options() {
			if src, known := glossSource[o.Gloss]; known {
				seen[src]++
			}
		}
		for src, n := range seen {
			if n > 1 {
				t.Errorf("the question for %q draws %d options from the entry %q — "+
					"two senses of one entry, both defining the word, one marked wrong:\n%s",
					q.Word(), n, src, c.Prompt())
			}
		}
	}
	if choices == 0 {
		t.Fatal("no form 2.3 questions; this test asserted nothing")
	}
}

// Entry identity is the HEAD RUN, not Headword().
//
// Headword() is fields[0] (parse.go), so it is "hot" for `hot dog` and "a" for
// `a priori` — two genuinely different entries can share it, and using it as a
// dedup key silently drops one of their options. A learner with both `hot dog`
// and `hot` in the deck would lose a distractor, and on a small deck lose the
// form entirely to Recall.
func TestEntryIdentityDistinguishesEntriesHeadwordConflates(t *testing.T) {
	d := testDict(t)
	ids := map[string]string{}
	for _, w := range []string{"hot dog", "a priori", "sycophantic", "mesa", "concrete"} {
		raw, err := d.Lookup(w)
		if err != nil {
			t.Fatalf("%s is not in the committed corpus: %v", w, err)
		}
		e := ParseEntry(raw)
		id := entryIdentity(e)
		if id == "" {
			t.Errorf("%q has no entry identity", w)
		}
		if prev, dup := ids[id]; dup {
			t.Errorf("%q and %q share the identity %q — one of their options would be "+
				"silently dropped", w, prev, id)
		}
		ids[id] = w
		// The specific conflation this replaces.
		if w == "hot dog" && e.Headword() == id {
			t.Errorf("identity for %q is %q, the same as Headword() — the head run was not used", w, id)
		}
	}
}

// A LONG GLOSS WRAPS, and every line fits the terminal (#41).
//
// The operator's screenshot is the case: `ligament`'s option line ran off the
// right edge and was CUT, not wrapped. Before #41 the terminal wrapped it, at
// the column and with no indent; a frame clips instead, because a line that
// wraps makes the frame a row too tall and the terminal then scrolls every row
// the sitting placed.
//
// Two claims, and the second is the one a fix that merely truncated would drop:
// it fits, AND nothing was lost.
func TestALongOptionGlossWrapsRatherThanBeingCut(t *testing.T) {
	const width = 50
	d := testDict(t)
	raw, err := d.Lookup("quokka")
	if err != nil {
		t.Fatalf("quokka is not in the committed corpus: %v", err)
	}
	entry := ParseEntry(raw)
	target, ok := targetCandidate("quokka", entry)
	if !ok {
		t.Fatal("quokka yields no target gloss")
	}
	if visibleCells(target.Gloss)+play.OptionIndent <= width {
		t.Fatalf("the corpus gloss is only %d columns, so nothing would wrap and this test "+
			"would assert nothing: %q", visibleCells(target.Gloss), target.Gloss)
	}
	q := choiceFor("quokka", "", entry, []play.Candidate{
		{Word: "mesa", Gloss: "an isolated flat-topped hill with steep sides", Axis: play.AxisGeneral},
		{Word: "parrot", Gloss: "a bird with a short hooked bill", Axis: play.AxisGeneral},
	}, 1)
	if q == nil {
		t.Fatal("no choice built")
	}

	prompt := wrapWritten(q.Prompt(), width)
	for _, line := range strings.Split(prompt, "\n") {
		if n := visibleCells(line); n > width {
			t.Errorf("a prompt line is %d columns wide in a %d-column terminal, so the frame "+
				"clips it: %q", n, width, line)
		}
	}
	// ...and every word survived. Truncation also "fits".
	if !strings.Contains(collapseSpace(prompt), collapseSpace(target.Gloss)) {
		t.Errorf("the gloss did not survive wrapping — it fits because it was cut:\n%s", prompt)
	}
	// The continuations are INDENTED under the gloss rather than starting at
	// column 0, where the eye expects the next option. That is the half #7 left
	// as a known rough edge and #41 had to close anyway.
	indented := false
	for _, line := range strings.Split(prompt, "\n") {
		if strings.TrimSpace(line) != "" && strings.HasPrefix(line, strings.Repeat(" ", play.OptionIndent)) {
			indented = true
		}
	}
	if !indented {
		t.Error("no continuation line was indented, so either nothing wrapped or the hanging indent is gone")
	}
}

// collapseSpace flattens a wrap so the assertion is about the WORDS surviving
// rather than about where the breaks landed.
func collapseSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

// A width of 0 means "do not wrap", which is terminalWidth's sentinel for a
// terminal too narrow to break a definition in. The gloss must arrive whole.
func TestUnwrappedWidthLeavesTheGlossAlone(t *testing.T) {
	d := testDict(t)
	raw, _ := d.Lookup("quokka")
	entry := ParseEntry(raw)
	target, _ := targetCandidate("quokka", entry)

	q := choiceFor("quokka", "", entry, []play.Candidate{
		{Word: "mesa", Gloss: "an isolated flat-topped hill", Axis: play.AxisGeneral},
		{Word: "parrot", Gloss: "a bird with a short hooked bill", Axis: play.AxisGeneral},
	}, 1)
	if q == nil {
		t.Fatal("no choice built")
	}
	if got := wrapWritten(q.Prompt(), 0); !strings.Contains(got, target.Gloss) {
		t.Errorf("width 0 did not leave the gloss intact:\n%s", got)
	}
}

// wrapWritten leaves alone anything that already FITS, whatever kind of line it
// is — so text Render has already wrapped passes through untouched and only what
// is too wide is broken.
func TestWrapWrittenLeavesFittingLinesAlone(t *testing.T) {
	// WIDE ENOUGH for every row below to fit — the claim is "a line that fits is
	// returned untouched", so a row that does not fit would be testing the other
	// half and passing for the wrong reason.
	const width = 60
	for _, tc := range []struct{ name, in string }{
		{"a headword", "internationalization"},
		{"a blank line", ""},
		{"a rendered definition line, already wrapped", "  a definition line"},
		{"a numbered SENSE, which Render writes with a dot", "    1. cover an area with concrete and then some"},
		{"a digit with one space is not an option line", "1 not an option"},
		{"a definition body line Render already wrapped", "      a definition line"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := wrapWritten(tc.in, width); got != tc.in {
				t.Errorf("wrapWritten rewrote a line that already fits:\n in  %q\n out %q", tc.in, got)
			}
		})
	}
	// ...and it DOES break what is too wide, at a width narrow enough to force
	// it, or the table above passes for a function that does nothing. Both kinds, each under its own indent: an
	// option line hangs under the gloss, a body line under its own indentation.
	for _, tc := range []struct{ name, in, wantIndent string }{
		{"an option line hangs under the gloss", "1  " + strings.Repeat("word ", 10), strings.Repeat(" ", play.OptionIndent)},
		{"a body line hangs under its own indent", "      " + strings.Repeat("word ", 10), "      "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const narrow = 20
			got := wrapWritten(tc.in, narrow)
			lines := strings.Split(got, "\n")
			if len(lines) < 2 {
				t.Fatalf("an over-wide line was not wrapped: %q", got)
			}
			for _, l := range lines {
				if visibleCells(l) > narrow {
					t.Errorf("a wrapped line is still %d columns: %q", visibleCells(l), l)
				}
			}
			if !strings.HasPrefix(lines[1], tc.wantIndent) {
				t.Errorf("continuation %q does not hang under %q", lines[1], tc.wantIndent)
			}
		})
	}
}

// STYLED text wraps, and only the ERASE gesture is exempt (BR-23).
//
// The Critical this pins was mine and it shipped for one round: protecting the
// `♫ playing 3×` indicator, whose `\r\x1b[K` marker `wrapText` would scatter
// across a break, I skipped every line carrying an escape. `--play` refuses to
// run with `-no-color` (BR-3), so every rendered definition line carries colour
// — the skip therefore exempted exactly the lines the wrap exists for.
//
// A UNIT test rather than a sitting, because the sitting's version of this
// depends on which word comes second and how long its entry is. A contract is
// the thing to state.
func TestWrapWrittenWrapsStyledTextButNotTheEraseGesture(t *testing.T) {
	const width = 40

	t.Run("a coloured line wraps", func(t *testing.T) {
		line := "      \x1b[3;32m“" + strings.Repeat("word ", 20) + "”\x1b[0m"
		if visibleCells(line) <= width {
			t.Fatalf("the fixture is only %d cells, so nothing would wrap", visibleCells(line))
		}
		got := wrapWritten(line, width)
		for _, l := range strings.Split(got, "\n") {
			if n := visibleCells(l); n > width {
				t.Errorf("a styled line is still %d columns after wrapping: %q", n, l)
			}
		}
		// ...and the style survived: the escapes are still in there, attached to
		// the words they opened on.
		if !strings.Contains(got, "\x1b[3;32m") || !strings.Contains(got, "\x1b[0m") {
			t.Errorf("wrapping dropped the styling: %q", got)
		}
	})

	t.Run("the erase gesture is left whole", func(t *testing.T) {
		line := eraseLine + "♫ playing 3× " + strings.Repeat("and again ", 8)
		if got := wrapWritten(line, width); got != line {
			t.Errorf("the take-that-line-back marker was scattered across a break:\n in  %q\n out %q", line, got)
		}
	})
}

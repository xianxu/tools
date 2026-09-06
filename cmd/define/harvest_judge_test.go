package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

// THE COMMITTED KNOWN-BAD PAIR, which Done-when 4 names precisely: "the veto is
// exercised by a committed known-bad case".
//
// `obsequious` is a near-synonym of `sycophantic`. Under the selection rule it
// is MORE likely to be chosen than a random word, not less — same domain, same
// band — so the plausibility that makes a good distractor is exactly what makes
// this the failure mode. A veto that has never rejected anything is a veto
// nobody has seen work.
//
// The case moved here from `#12` when selection moved (see #12's Revisions): the
// same assertion, made at authoring time instead of at review time.
const (
	knownBadCandidate = "obsequious"
	knownBadAnswer    = "sycophantic"
	knownBadStem      = "The aide was sycophantic, praising a policy he had privately called unworkable."
)

// The veto REJECTS, and the rejection reaches the item on disk.
func TestVetoRejectsTheKnownBadDistractor(t *testing.T) {
	d, fake, st := harvestRig(t, 0)
	if err := d.deck.SetWordFacts(knownBadAnswer, store.WordFacts{
		Band: store.C1, Domain: store.DomainGeneral, At: harvestClock,
	}); err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{knownBadCandidate, "laconic", "punctilious"} {
		if err := d.deck.Upsert(store.Word{Text: w, FirstSeen: harvestClock, LastSeen: harvestClock, Lookups: 1}); err != nil {
			t.Fatal(err)
		}
		if err := d.deck.SetWordFacts(w, store.WordFacts{
			Band: store.C1, Domain: store.DomainGeneral, At: harvestClock,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.deck.Upsert(store.Word{
		Text: knownBadAnswer, FirstSeen: harvestClock, LastSeen: harvestClock, Lookups: 1,
	}); err != nil {
		t.Fatal(err)
	}

	// REGISTERED BEFORE scriptAll: matchers are ordered and first match wins, and
	// two Script calls with one key extend a single queue — so a specific reply
	// added afterwards sits behind the catch-all and is never served.
	//
	// The veto's answer depends on the CANDIDATE, which is the whole shape of the
	// task: a yes/no on a concrete pair rather than a ranking.
	// Matched on the veto prompt's OWN candidate line, not on the bare word: the
	// deck holds `obsequious` too, so a substring match would also catch the
	// author prompt written for it and answer that with a veto verdict.
	fake.Script("considered: **"+knownBadCandidate+"**",
		llmtest.Reply{Text: `{"fits":true,"reason":"a near-synonym; it would also make the sentence true"}`})
	fake.Script(authorKey(knownBadAnswer), llmtest.Reply{Text: `{"stem":"` + knownBadStem + `"}`})
	// This test's deck is its own — the near-synonym cluster, not deckWord's
	// range — so its author replies are registered here rather than relying on
	// scriptAll's coverage.
	for _, w := range []string{knownBadCandidate, "laconic", "punctilious"} {
		for range 8 {
			fake.Script(authorKey(w), llmtest.Reply{
				Text: `{"stem":"The Times of London called the aide ` + w + ` in its Monday leader."}`,
			})
		}
	}
	scriptAll(fake, 24)

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}

	items, err := st.Items(knownBadAnswer)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatalf("nothing was authored; stderr: %s", errOut.String())
	}
	for _, dstr := range items[0].Distractors {
		if strings.EqualFold(dstr, knownBadCandidate) {
			t.Errorf("%q survived the veto beside %q — a question with two correct options teaches nothing",
				knownBadCandidate, knownBadAnswer)
		}
	}
	// SAID OUT LOUD. A veto that rejects silently cannot be audited from a batch.
	if !strings.Contains(errOut.String(), "vetoed") {
		t.Errorf("the veto fired but said nothing: %q", errOut.String())
	}
}

// The other half of Done-when 4: an item whose every candidate is vetoed is NOT
// written with two options. It stays unauthored and re-authorable.
func TestEveryCandidateVetoedLeavesTheWordUnauthored(t *testing.T) {
	d, fake, st := harvestRig(t, 3)
	for range 40 {
		fake.Script(markVeto, llmtest.Reply{Text: `{"fits":true,"reason":"all of these also fit"}`})
	}
	scriptAll(fake, 40)

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	for i := range 3 {
		items, err := st.Items(deckWord(i))
		if err != nil {
			t.Fatal(err)
		}
		if len(items) > 0 {
			t.Errorf("%q was authored with %d option(s) after every candidate was vetoed",
				deckWord(i), len(items[0].Distractors))
		}
	}
	if !strings.Contains(out.String(), "rejected") {
		t.Errorf("the run did not report the rejections: %q", out.String())
	}
}

// THE COMMITTED KNOWN-BAD STEM, which Done-when 5 asks of the entailment judge.
//
// "His ___ behaviour was noted by all" is the measured failure the Spec opens
// with: a sentence its own context does not entail, so almost any adjective
// fits and the item teaches nothing. A judge that has never rejected anything
// is worth exactly as much as no judge.
const knownBadStemUnentailed = "His sycophantic behaviour was noted by all."

func TestAnUnentailedStemIsRejected(t *testing.T) {
	d, fake, st := harvestRig(t, 2)
	// Keyed per word for the same reason scriptAll is: the stem must contain the
	// word or the free check rejects it before the judge is consulted.
	for _, w := range allDeckWords() {
		fake.Script(authorKey(w), llmtest.Reply{
			Text: `{"stem":"His ` + w + ` behaviour was noted by all at the Chatham Dockyard."}`,
		})
	}
	for range 24 {
		fake.Script(markEntail, llmtest.Reply{
			Text: `{"entails":false,"glosses":false,"named":false,"reason":"almost any adjective fits the blank"}`,
		})
	}
	scriptAll(fake, 24)

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	for i := range 2 {
		items, err := st.Items(deckWord(i))
		if err != nil {
			t.Fatal(err)
		}
		if len(items) > 0 {
			t.Errorf("%q kept a stem the judge rejected: %q", deckWord(i), items[0].Stem)
		}
	}
	if !strings.Contains(errOut.String(), "stem rejected") {
		t.Errorf("the rejection was silent: %q", errOut.String())
	}
}

// A stem that entails but names nobody is REJECTED TOO, and the two conditions
// are separate fields so a reader can tell which failed. The Spec makes naming a
// requirement, not a preference: an unnamed subject gives the learner no
// referent to attach the word to.
func TestAStemThatNamesNobodyIsRejected(t *testing.T) {
	d, fake, st := harvestRig(t, 3)
	for range 12 {
		fake.Script(markEntail, llmtest.Reply{
			Text: `{"entails":true,"glosses":false,"named":false,"reason":"the subject is \"a manager\""}`,
		})
	}
	scriptAll(fake, 12)

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	// The pass must have RUN — a rig too small to author would satisfy the
	// assertion below without the requirement doing anything.
	if strings.Contains(out.String(), "skipped authoring") {
		t.Fatalf("authoring never ran, so this pin cannot fail: %q", out.String())
	}
	if got := countTask(fake, markEntail); got == 0 {
		t.Fatal("the judge was never consulted, so the named requirement was never exercised")
	}
	for i := range 3 {
		items, err := st.Items(deckWord(i))
		if err != nil {
			t.Fatal(err)
		}
		if len(items) > 0 {
			t.Errorf("%q: an unnamed stem was authored; naming is a requirement, not a preference",
				deckWord(i))
		}
	}
	if !strings.Contains(errOut.String(), "stem rejected") {
		t.Errorf("the rejection was silent: %q", errOut.String())
	}
}

// The judge runs BEFORE any distractor is selected, so a doomed item does not
// spend the veto's calls. Asserted on the wire: a rejected stem must produce no
// veto request at all.
func TestARejectedStemNeverReachesTheVeto(t *testing.T) {
	d, fake, _ := harvestRig(t, 2)
	for range 24 {
		fake.Script(markEntail, llmtest.Reply{
			Text: `{"entails":false,"glosses":false,"named":true,"reason":"the sentence does not pin the word down"}`,
		})
	}
	scriptAll(fake, 24)

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	if got := countTask(fake, markVeto); got != 0 {
		t.Errorf("%d veto call(s) were spent on a stem already rejected; a stem that does not "+
			"entail its answer cannot be rescued by better wrong answers", got)
	}
}

// The batch's topic spread is REPORTED, and it is the measure taken with no
// model — the reason it exists rather than a judge scoring its own variety.
func TestTheBatchReportsItsTopicSpread(t *testing.T) {
	d, fake, _ := harvestRig(t, 3)
	scriptAll(fake, 40)

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "topic spread") {
		t.Errorf("the batch did not report its spread: %q", out.String())
	}
}

// The prompts are pinned. The veto's especially: it must show the model the
// BLANKED sentence, or every candidate looks wrong beside one that already
// contains the right answer.
func TestJudgePromptGoldens(t *testing.T) {
	llmtest.AssertGolden(t, "testdata", "entail-prompt", renderEntailPrompt(store.DefaultLang, knownBadAnswer, knownBadStem))
	llmtest.AssertGolden(t, "testdata", "veto-prompt",
		renderVetoPrompt(store.DefaultLang, knownBadAnswer, knownBadStem, knownBadCandidate))
}

func TestVetoPromptHidesTheAnswer(t *testing.T) {
	got := renderVetoPrompt(store.DefaultLang, knownBadAnswer, knownBadStem, knownBadCandidate).Prompt
	// The STEM is blanked. Asserted on the sentence rather than by counting the
	// word across the whole prompt: the instructions legitimately name it twice
	// more — once as the intended answer, once in the near-synonym example — and
	// a global count would make those look like leaks.
	if strings.Contains(got, knownBadStem) {
		t.Errorf("the sentence went in with its answer still in it:\n%s", got)
	}
	if !strings.Contains(got, blankOut(knownBadStem, knownBadAnswer)) {
		t.Errorf("the blanked sentence is not in the prompt:\n%s", got)
	}
	// The near-synonym trap is stated, because it is the case the veto exists for
	// and the one a model is most likely to wave through.
	if !strings.Contains(got, "near-synonym") {
		t.Error("the prompt does not name the near-synonym case")
	}
}

// THE THIRD COMMITTED KNOWN-BAD STEM, added by the M2 checkpoint.
//
// This one ENTAILS its answer perfectly — and is still rejected, which is the
// whole point. Half of the first real batch came back like this: the cheapest
// way to make a sentence entail a word is to define the word in it, so the
// entailment requirement produced a reading test. A judge scoring only
// entailment passes every one of them.
const knownBadStemGlossed = "Each spring, biologists at the Holyoke Dam count the alewife, " +
	"the small silver herring that leaves the Atlantic to spawn upstream in fresh water."

func TestAGlossedStemIsRejectedEvenThoughItEntails(t *testing.T) {
	d, fake, st := harvestRig(t, 2)
	for _, w := range allDeckWords() {
		fake.Script(authorKey(w), llmtest.Reply{
			Text: `{"stem":"Biologists at the Holyoke Dam count the ` + w + `, the small silver herring that spawns upstream."}`,
		})
	}
	for range 24 {
		// Entails AND names — and still rejected, on the gloss alone.
		fake.Script(markEntail, llmtest.Reply{
			Text: `{"entails":true,"glosses":true,"named":true,"reason":"an appositive defines the word"}`,
		})
	}
	scriptAll(fake, 24)

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	for i := range 2 {
		items, err := st.Items(deckWord(i))
		if err != nil {
			t.Fatal(err)
		}
		if len(items) > 0 {
			t.Errorf("%q kept a stem that carries its own definition: %q", deckWord(i), items[0].Stem)
		}
	}
	if !strings.Contains(errOut.String(), "stem rejected") {
		t.Errorf("the rejection was silent: %q", errOut.String())
	}
}

// The three conditions are separate FIELDS, so a batch can be read for which one
// is failing — and the gloss field is what the checkpoint added.
func TestEntailPromptAsksAllThreeSeparately(t *testing.T) {
	got := renderEntailPrompt(store.DefaultLang, knownBadAnswer, knownBadStem).Prompt
	for _, want := range []string{"**entails**", "**glosses**", "**named**"} {
		if !strings.Contains(got, want) {
			t.Errorf("the judge is never asked %s", want)
		}
	}
	// SHOWN, not described. "Do not define it" is what the author's system prompt
	// already said, and the model honoured it by writing an appositive instead.
	if !strings.Contains(got, "appositive") {
		t.Error("the gloss question does not name the shape it is looking for")
	}
}

// And the author prompt shows the same shape it forbids, for the same reason.
func TestAuthorPromptForbidsTheGlossByExample(t *testing.T) {
	got := renderAuthorPrompt(store.DefaultLang, "x", "", store.WordFacts{}, learnerFacts{}).Prompt
	if !strings.Contains(got, "Never gloss the word") {
		t.Error("the author prompt does not forbid glossing")
	}
	// A WRONG example and a RIGHT one. The first batch showed that stating the
	// rule is not enough: the system prompt already said "never write a
	// definition" and got 10 appositives out of 20.
	if !strings.Contains(got, "small silver herring") {
		t.Error("the prompt does not show a wrong example")
	}
	if !strings.Contains(got, "climbing the fish lift") {
		t.Error("the prompt does not show a right example")
	}
}

// The free check runs BEFORE the judge is paid to have an opinion. Asserted on
// the wire: a stem that does not contain its word must cost zero judge calls.
func TestAStemWithoutItsWordNeverReachesAJudge(t *testing.T) {
	d, fake, st := harvestRig(t, 2)
	for _, w := range allDeckWords() {
		fake.Script(authorKey(w), llmtest.Reply{
			Text: `{"stem":"The village stood atop the narrow ___ of First ` + w + `."}`,
		})
	}
	scriptAll(fake, 24)

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	if got := countTask(fake, markEntail); got != 0 {
		t.Errorf("%d judge call(s) were spent on a stem that cannot be rendered at all", got)
	}
	for i := range 2 {
		items, err := st.Items(deckWord(i))
		if err != nil {
			t.Fatal(err)
		}
		if len(items) > 0 {
			t.Errorf("%q was authored from a stem that does not contain it", deckWord(i))
		}
	}
	if !strings.Contains(errOut.String(), "does not use the word") {
		t.Errorf("the rejection was silent: %q", errOut.String())
	}
}

// The entailment judge asks about MEANING DOING WORK, not about unique
// recoverability — the second checkpoint rejected 9 of 20 items with reasons
// citing "no definition is supplied", which is the gloss rule's own forbidden
// thing offered as grounds for rejection.
func TestEntailPromptDoesNotDemandUniqueRecoverability(t *testing.T) {
	got := renderEntailPrompt(store.DefaultLang, knownBadAnswer, knownBadStem).Prompt
	if !strings.Contains(got, "beside three other options") {
		t.Error("the judge is not told this is a multiple-choice form")
	}
	// The contradiction, named explicitly, because the model found it on its own
	// and resolved it the wrong way.
	if !strings.Contains(got, "near-synonym also fitting is NOT a reason to answer no") {
		t.Error("the judge is not told the veto owns the per-option question")
	}
	if strings.Contains(got, "from the rest of the sentence alone") {
		t.Error("the judge still asks for recoverability from bare context, which forces a gloss")
	}
}

// The budget charges the VETO's inner loop too, not only the outer passes.
//
// The veto runs up to three calls per item, so a budget that skipped it could be
// exceeded threefold by a deck whose items all pass judging — which is the good
// case, not an edge case.
func TestTheBudgetChargesTheVeto(t *testing.T) {
	d, fake, _ := harvestRig(t, 8)
	preBand(t, d) // so the budget reaches authoring rather than being spent on banding
	scriptAll(fake, 80)

	const limit = 7
	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{limit: limit}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	if got := countTask(fake, markVeto); got == 0 {
		t.Fatal("no veto calls at all, so this assertion cannot fail")
	}
	if got := len(fake.Requests()); got > limit {
		t.Errorf("made %d model calls with -limit %d (veto alone: %d); the veto's inner loop "+
			"must charge the budget like every other call site", got, limit, countTask(fake, markVeto))
	}
}

// A budget that runs out MID-ITEM abandons the item rather than writing it
// short. An item is cached forever and a later run skips a word that already has
// material, so a truncated write would make a budget limit permanently a quality
// limit for that word.
func TestABudgetExhaustedMidItemWritesNothing(t *testing.T) {
	d, fake, st := harvestRig(t, 8)
	preBand(t, d)
	scriptAll(fake, 80)

	// Enough for one author + one entail + one veto, then nothing: the second
	// veto call of the first item is where it runs out.
	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{limit: 3}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}

	for _, w := range allDeckWords() {
		items, err := st.Items(w)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) == 0 {
			continue
		}
		if len(items[0].Distractors) >= optionsPerItem {
			continue // a complete item is fine
		}
		t.Errorf("%q was written with only %d of %d options after the budget ran out; a later "+
			"run will skip it, so the shortfall is permanent",
			w, len(items[0].Distractors), optionsPerItem)
	}
	if !strings.Contains(out.String(), "mid-item") {
		t.Errorf("the abandoned item was not reported: %q", out.String())
	}
}

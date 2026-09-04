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
	fake.Script(markAuthor, llmtest.Reply{Text: `{"stem":"` + knownBadStem + `"}`})
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
	fake.Script(markAuthor, llmtest.Reply{Text: `{"stem":"` + knownBadStemUnentailed + `"}`})
	for range 24 {
		fake.Script(markEntail, llmtest.Reply{
			Text: `{"entails":false,"named":false,"reason":"almost any adjective fits the blank"}`,
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
	d, fake, st := harvestRig(t, 1)
	for range 12 {
		fake.Script(markEntail, llmtest.Reply{
			Text: `{"entails":true,"named":false,"reason":"the subject is \"a manager\""}`,
		})
	}
	scriptAll(fake, 12)

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	items, err := st.Items(deckWord(0))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) > 0 {
		t.Error("an unnamed stem was authored; naming is a requirement, not a preference")
	}
}

// The judge runs BEFORE any distractor is selected, so a doomed item does not
// spend the veto's calls. Asserted on the wire: a rejected stem must produce no
// veto request at all.
func TestARejectedStemNeverReachesTheVeto(t *testing.T) {
	d, fake, _ := harvestRig(t, 2)
	for range 24 {
		fake.Script(markEntail, llmtest.Reply{
			Text: `{"entails":false,"named":true,"reason":"the sentence does not pin the word down"}`,
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
	llmtest.AssertGolden(t, "testdata", "entail-prompt", renderEntailPrompt(knownBadAnswer, knownBadStem))
	llmtest.AssertGolden(t, "testdata", "veto-prompt",
		renderVetoPrompt(knownBadAnswer, knownBadStem, knownBadCandidate))
}

func TestVetoPromptHidesTheAnswer(t *testing.T) {
	got := renderVetoPrompt(knownBadAnswer, knownBadStem, knownBadCandidate).Prompt
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

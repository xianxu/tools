package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

var harvestClock = time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

// harvestRig wires runHarvest against the WIRE-LEVEL fake and a real YAML store
// in a temp dir, so facts/ is written and read through the production path
// rather than seeded into a field.
func harvestRig(t *testing.T, words int) (deps, *llmtest.Fake, *store.YAML) {
	t.Helper()
	fake := llmtest.NewFake(t)
	dir := t.TempDir()
	st := store.NewYAML(dir, store.DefaultLang, nil)
	for i := range words {
		if err := st.Upsert(store.Word{
			Text:      deckWord(i),
			FirstSeen: harvestClock.AddDate(0, 0, -20),
			LastSeen:  harvestClock.AddDate(0, 0, -i),
			Lookups:   1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	d := testDeps(t)
	d.deck = st
	d.clock = store.FixedClock(harvestClock)
	d.newLLM = llm.New
	d.getenv = envFor(fake.URL)
	return d, fake, st
}

const bandReply = `{"band":"C1","domain":"Law"}`

// Done-when 2, and it is a claim about CALLS rather than about a file existing.
//
// The second run is driven against a fake that has served everything it was
// scripted; asserting the request COUNT is what makes "cached forever" mean the
// network is not touched, rather than meaning a file happens to be on disk while
// the loop asks anyway and overwrites it with the same answer.
func TestHarvestAsksOncePerWordAndNeverAgain(t *testing.T) {
	d, fake, st := harvestRig(t, 3)
	fake.Script("", llmtest.Reply{Text: bandReply}, llmtest.Reply{Text: bandReply}, llmtest.Reply{Text: bandReply})

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("first run = %d, stderr: %s", code, errOut.String())
	}
	first := len(fake.Requests())
	if first == 0 {
		t.Fatal("the first run made no calls at all")
	}

	deck, err := st.Deck()
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range deck {
		f, err := st.WordFacts(w.Text)
		if err != nil {
			t.Fatal(err)
		}
		if !f.Harvested() {
			t.Errorf("%q was not banded by the first run", w.Text)
		}
	}

	out.Reset()
	errOut.Reset()
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("second run = %d, stderr: %s", code, errOut.String())
	}
	if got := len(fake.Requests()); got != first {
		t.Errorf("the second run made %d more call(s); a banded word must be re-READ, not re-ASKED", got-first)
	}
}

// Done-when 6: an outage leaves the existing store usable and untouched, and
// harvesting is the only thing that stops.
//
// The fake fails partway, and the assertion is about what SURVIVED: words banded
// before the failure are intact and complete. Each word is written atomically as
// it is answered, so this is a property rather than an accident — but only a test
// makes it one.
func TestHarvestOutageKeepsWhatWasAlreadyBought(t *testing.T) {
	d, fake, st := harvestRig(t, 4)
	fake.Script("", llmtest.Reply{Text: bandReply}, llmtest.Reply{Text: bandReply},
		llmtest.Reply{Status: 500, Text: "upstream is having a day"})

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code == 0 {
		t.Fatal("a mid-batch outage reported success")
	}

	deck, err := st.Deck()
	if err != nil {
		t.Fatal(err)
	}
	var banded int
	for _, w := range deck {
		f, err := st.WordFacts(w.Text)
		if err != nil {
			t.Fatalf("a word's facts became unreadable after the outage: %v", err)
		}
		if !f.Harvested() {
			continue
		}
		banded++
		// Not merely present — WHOLE. A torn write would show up as a record with
		// a timestamp and no band.
		if f.Band == "" {
			t.Errorf("%q kept a timestamp but lost its band; the write was not atomic", w.Text)
		}
	}
	if banded == 0 {
		t.Error("the outage discarded work that had already been paid for")
	}
	if banded == len(deck) {
		t.Error("the fake failed but every word was banded; the outage never happened")
	}
	if !strings.Contains(out.String(), "saved") {
		t.Errorf("the run did not say what survived: %q", out.String())
	}
}

// --limit is a real cap, not documentation (the Minor the estimate-quality round
// priced). Asserted on a deck larger than the limit, which is the only shape
// where an unbounded loop and a bounded one differ.
func TestHarvestStopsAtTheLimit(t *testing.T) {
	d, fake, _ := harvestRig(t, 6)
	for range 6 {
		fake.Script("", llmtest.Reply{Text: bandReply})
	}

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{limit: 2}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	if got := len(fake.Requests()); got != 2 {
		t.Errorf("made %d calls with -limit 2; the cap is not enforced", got)
	}
	// A capped run is a PARTIAL run, not a failure, and it has to say so or the
	// operator cannot tell it from a finished one.
	if !strings.Contains(out.String(), "run again") {
		t.Errorf("a capped run did not say it was partial: %q", out.String())
	}
}

// A band that does not parse leaves the word UNBANDED and re-askable, rather
// than writing a value the comparison every distractor rule depends on cannot
// order.
func TestHarvestRefusesABandOffTheScale(t *testing.T) {
	d, fake, st := harvestRig(t, 1)
	fake.Script("", llmtest.Reply{Text: `{"band":"B2+","domain":"Law"}`})

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	f, err := st.WordFacts(deckWord(0))
	if err != nil {
		t.Fatal(err)
	}
	if f.Harvested() {
		t.Errorf("an off-scale band was written: %+v", f)
	}
	if !strings.Contains(errOut.String(), "not a CEFR band") {
		t.Errorf("the refusal was silent: %q", errOut.String())
	}
}

// The measurement mode WRITES NOTHING. That is what lets it re-ask a word the
// cache already holds without destroying the thing it is measuring.
func TestAgreementModeWritesNothing(t *testing.T) {
	d, fake, st := harvestRig(t, 2)
	if err := st.SetWordFacts(deckWord(0), store.WordFacts{
		Band: store.B1, Domain: store.DomainGeneral, At: harvestClock,
	}); err != nil {
		t.Fatal(err)
	}
	for range 6 {
		fake.Script("", llmtest.Reply{Text: bandReply}) // C1, disagreeing with the stored B1
	}

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{agreement: 3}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}

	f, err := st.WordFacts(deckWord(0))
	if err != nil {
		t.Fatal(err)
	}
	if f.Band != store.B1 {
		t.Errorf("band = %q, want the stored B1 untouched — measuring must not overwrite", f.Band)
	}
	// The unbanded word must not have been banded as a side effect either.
	if f2, err := st.WordFacts(deckWord(1)); err != nil {
		t.Fatal(err)
	} else if f2.Harvested() {
		t.Error("the measurement mode banded a word that was not banded before")
	}
	if !strings.Contains(out.String(), "agreement") {
		t.Errorf("the measurement reported no number: %q", out.String())
	}
	// The caveat travels with the number, every time, because the number invites
	// exactly the reading it does not support.
	if !strings.Contains(out.String(), "STABILITY, not correctness") {
		t.Errorf("the agreement was reported without its caveat: %q", out.String())
	}
}

// The measurement samples ALREADY-BANDED words: an unbanded word has no cached
// answer to be stable about, so measuring one reports nothing about the cache.
func TestAgreementModeRefusesWhenNothingIsBanded(t *testing.T) {
	d, _, _ := harvestRig(t, 2)
	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{agreement: 3}, &out, &errOut); code == 0 {
		t.Fatal("measuring an unbanded deck reported success")
	}
	if !strings.Contains(errOut.String(), "--harvest before measuring") {
		t.Errorf("the refusal did not say what to do: %q", errOut.String())
	}
}

// DONE-WHEN 1: a review sitting NEVER waits on harvesting.
//
// The seam is made to PANIC rather than left nil, and the difference is the whole
// point of the test: nil passes on a loop that reaches for the model behind a
// `!= nil` guard, which is exactly how a network dependency creeps into a path
// that promises to be offline. A panic fails whatever touches it.
//
// The deck here is entirely UNHARVESTED, which is the state that would tempt a
// sitting into harvesting on demand. It must simply play.
func TestASittingNeverWaitsOnHarvesting(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic", "ephemeral")
	d.getenv = func(string) string {
		panic("a review sitting resolved the model configuration; it must stay offline")
	}
	d.newLLM = func(llm.Config) llm.Client {
		panic("a review sitting constructed a model client; harvesting is a batch path, never a sitting's")
	}

	qs, held := questionsFor(t, d, opt)
	if len(qs) == 0 {
		t.Fatal("the rig produced no questions, so this would pass vacuously")
	}

	var out, errb bytes.Buffer
	keys := "\r" + gradeKey(t, qs[0], play.Correct)
	playSession(t.Context(), d, opt, play.NewSession(qs[:1]), held, keysFor(keys), playbackConsole(&out, &errb))

	if len(reviewEvents(t, st)) != 1 {
		t.Fatal("the sitting did not complete")
	}
	// And it did not quietly harvest on the way past: a sitting that banded words
	// would be doing the batch path's work on the learner's time even if it
	// happened to be fast today.
	for _, w := range []string{"sycophantic", "ephemeral"} {
		f, err := st.WordFacts(w)
		if err != nil {
			t.Fatal(err)
		}
		if f.Harvested() {
			t.Errorf("the sitting banded %q; harvesting must happen ahead of time, not during a review", w)
		}
	}
}

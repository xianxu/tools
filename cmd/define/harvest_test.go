package main

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
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
// harvestRig builds a deck of `words` deck words with the wire-level fake.
//
// A rig with fewer than two words cannot exercise authoring at all — runAuthoring
// needs a pool to select wrong answers from — and a test that asserts "nothing
// was authored" against such a rig passes for the wrong reason. Three M2 pins
// were unfalsifiable exactly this way, so the constraint is enforced here rather
// than left to each test to remember. Pass 0 deliberately when seeding the deck
// by hand.
func harvestRig(t *testing.T, words int) (deps, *llmtest.Fake, *store.YAML) {
	t.Helper()
	if words == 1 {
		t.Fatal("harvestRig(1) cannot exercise authoring: runAuthoring needs a pool of at least 2, " +
			"so every assertion about authoring would pass without the pass running")
	}
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

// The four tasks --harvest runs, matched by a marker unique to each prompt.
// Scripting by TASK rather than by call order is what keeps these tests readable
// once one invocation drives banding, authoring, entailment and the veto.
const (
	markBand   = "## What to answer"
	markAuthor = "## What to write"
	markEntail = "**entails**"
	markVeto   = "**fits**"
)

// scriptAll answers every task, so a test that cares about one pass is not
// written in terms of the others. n is generous: the fake serves its fallback
// once a queue is drained, and these replies are all well-formed.
//
// The author reply is PER WORD, keyed on the prompt's "## The word" line,
// because a stem must actually contain its answer — stemUsesTheWord checks that
// for free before any judge is paid, and one canned sentence would be rejected
// for every word but the one it happens to name.
func scriptAll(f *llmtest.Fake, n int) {
	for _, w := range allDeckWords() {
		for range n {
			f.Script(authorKey(w),
				llmtest.Reply{Text: `{"stem":"Senator Murkowski raised the question of ` + w + ` at the Commerce Committee hearing in Anchorage."}`})
		}
	}
	for range n {
		f.Script(markBand, llmtest.Reply{Text: bandReply})
		f.Script(markEntail, llmtest.Reply{Text: `{"entails":true,"glosses":false,"named":true,"reason":"names the committee"}`})
		f.Script(markVeto, llmtest.Reply{Text: `{"fits":false,"reason":"unrelated meaning"}`})
	}
}

// authorKey matches the AUTHOR prompt for one word and nothing else.
//
// "## The word\n\n<word>" is not enough: the band prompt opens identically, so
// that key serves author replies to banding calls and drains the queue. The
// parenthesised facts are what only authoring carries — its pool is banded by
// construction, so the band is always present there and never in the other.
func authorKey(word string) string { return "\n\n" + word + " (CEFR" }

// preBandExcept marks every deck word harvested except one, so a test can drive
// BOTH passes in one invocation: the unbanded word makes banding run, and the
// rest are the pool authoring needs.
func preBandExcept(t *testing.T, d deps, skip string) {
	t.Helper()
	for _, w := range allDeckWords() {
		if w == skip {
			continue
		}
		if err := d.deck.SetWordFacts(w, store.WordFacts{
			Band: store.C1, Domain: store.DomainGeneral, At: harvestClock,
		}); err != nil {
			t.Fatal(err)
		}
	}
}

// preBand marks every deck word harvested, so a test's budget reaches the
// authoring pass instead of being spent on banding.
func preBand(t *testing.T, d deps) {
	t.Helper()
	for _, w := range allDeckWords() {
		if err := d.deck.SetWordFacts(w, store.WordFacts{
			Band: store.C1, Domain: store.DomainGeneral, At: harvestClock,
		}); err != nil {
			t.Fatal(err)
		}
	}
}

// allDeckWords is deckWord's full range, so scriptAll covers whatever a rig
// asked for without each test restating its deck.
func allDeckWords() []string {
	seen := map[string]bool{}
	var out []string
	for i := range 14 {
		if w := deckWord(i); !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	return out
}

// countTask reports how many requests carried a task's marker.
func countTask(f *llmtest.Fake, mark string) int {
	n := 0
	for _, r := range f.Requests() {
		if strings.Contains(r.Prompt(), mark) {
			n++
		}
	}
	return n
}

// Done-when 2, and it is a claim about CALLS rather than about a file existing.
//
// The second run is driven against a fake that has served everything it was
// scripted; asserting the request COUNT is what makes "cached forever" mean the
// network is not touched, rather than meaning a file happens to be on disk while
// the loop asks anyway and overwrites it with the same answer.
func TestHarvestAsksOncePerWordAndNeverAgain(t *testing.T) {
	d, fake, st := harvestRig(t, 3)
	scriptAll(fake, 12)

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("first run = %d, stderr: %s", code, errOut.String())
	}
	first := countTask(fake, markBand)
	if first == 0 {
		t.Fatal("the first run asked about no words at all")
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

	before := len(fake.Requests())
	out.Reset()
	errOut.Reset()
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("second run = %d, stderr: %s", code, errOut.String())
	}
	if got := countTask(fake, markBand); got != first {
		t.Errorf("the second run made %d more banding call(s); a banded word must be re-READ, not re-ASKED", got-first)
	}
	// And nothing is re-authored either: an item on disk is finished material.
	if got := len(fake.Requests()); got != before {
		t.Errorf("the second run made %d call(s) of any kind; a harvested deck must cost nothing", got-before)
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
	fake.Script(markBand, llmtest.Reply{Text: bandReply}, llmtest.Reply{Text: bandReply},
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
// THE FLAG x PASS TABLE, which is what BR-16 asked for and BR-30 found missing.
//
// `-limit` bounds model calls, and this run makes four kinds of them: banding,
// authoring, entailment and the veto. A cell is pinned only by a test that
// PROVABLY ENTERS the pass — the first version of these tests used un-banded
// rigs, so the whole budget went on banding and `author=0 entail=0 veto=0`. The
// property in each name was untested and the scripted replies were never served.
//
// Each subtest therefore asserts the pass was reached before asserting the bound.
func TestTheLimitBoundsEveryPass(t *testing.T) {
	for _, tc := range []struct {
		name     string
		preBand  bool
		limit    int
		mustSend []string // markers that must appear, or the cell is untested
	}{
		{"banding", false, 4, []string{markBand}},
		{"authoring and the judges", true, 6, []string{markAuthor, markEntail, markVeto}},
		{"both passes in one run", false, 12, []string{markBand, markAuthor}},
		{"a limit of one", false, 1, []string{markBand}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, fake, _ := harvestRig(t, 8)
			if tc.preBand {
				preBand(t, d)
			}
			scriptAll(fake, 120)

			var out, errOut bytes.Buffer
			if code := runHarvest(context.Background(), d, options{}, harvestOptions{limit: tc.limit}, &out, &errOut); code != 0 {
				t.Fatalf("run = %d, stderr: %s", code, errOut.String())
			}

			for _, mark := range tc.mustSend {
				if countTask(fake, mark) == 0 {
					t.Fatalf("no request of that kind was sent, so this cell is untested "+
						"(band %d, author %d, entail %d, veto %d)",
						countTask(fake, markBand), countTask(fake, markAuthor),
						countTask(fake, markEntail), countTask(fake, markVeto))
				}
			}
			// EXACTLY the budget, not the budget plus one: a spend whose refusal
			// is discarded charges the call and then makes it anyway.
			if got := len(fake.Requests()); got > tc.limit {
				t.Errorf("made %d model calls with -limit %d (band %d, author %d, entail %d, veto %d)",
					got, tc.limit, countTask(fake, markBand), countTask(fake, markAuthor),
					countTask(fake, markEntail), countTask(fake, markVeto))
			}
		})
	}
}

// A capped run is a PARTIAL run and has to say so, or it cannot be told from a
// finished one.
func TestACappedRunSaysItIsPartial(t *testing.T) {
	d, fake, _ := harvestRig(t, 8)
	scriptAll(fake, 60)
	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{limit: 3}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "run again") {
		t.Errorf("a capped run did not say it was partial: %q", out.String())
	}
}

// The budget survives a pass that rejects everything — the shape that made the
// original counter wrong, since a rejected word charged nothing.
func TestTheLimitHoldsWhenEveryStemIsRejected(t *testing.T) {
	d, fake, _ := harvestRig(t, 8)
	preBand(t, d) // or the budget is spent on banding and the rejections never happen
	for range 60 {
		fake.Script(markEntail, llmtest.Reply{
			Text: `{"entails":false,"glosses":false,"named":false,"reason":"rejected"}`,
		})
	}
	scriptAll(fake, 60)

	const limit = 6
	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{limit: limit}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	if countTask(fake, markEntail) == 0 {
		t.Fatal("no stem was ever judged, so the rejection path is untested")
	}
	if got := len(fake.Requests()); got > limit {
		t.Errorf("made %d model calls with -limit %d although every stem was rejected; "+
			"rejections must charge the budget too", got, limit)
	}
}

// A band that does not parse leaves the word UNBANDED and re-askable, rather
// than writing a value the comparison every distractor rule depends on cannot
// order.
func TestHarvestRefusesABandOffTheScale(t *testing.T) {
	d, fake, st := harvestRig(t, 3)
	fake.Script(markBand, llmtest.Reply{Text: `{"band":"B2+","domain":"Law"}`})

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
	scriptAll(fake, 12) // the band replies say C1, disagreeing with the stored B1

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

// I-2's first green mutation: the axis filter in senseFacts.
//
// readGloss reports REGISTER (`informal`, `archaic`) on the same Label field as
// DOMAIN, so a filter that accepts any label stores `informal` as a word's
// domain. ParseDomain then flattens it to `general` — silently losing the
// specialist label a later sense actually carried, into a forever cache.
//
// Table-tested with no dictionary fake, which is why senseFacts was split out of
// wordSense: the derivation is the part worth pinning and it needs no IO.
func TestSenseFactsTakesTheDomainAxisOnly(t *testing.T) {
	d := testDict(t)
	for _, tc := range []struct {
		word       string
		wantDomain store.Domain
	}{
		// Entries the committed corpus labels with a subject field. Chosen from
		// the corpus rather than invented, so the fixture cannot drift from what
		// NOAD actually prints.
		{"record", "Law"},
		{"subject", "Music"},
		{"desert", "Military"},
		// Ordinary vocabulary: no field label anywhere, so no domain.
		{"ephemeral", ""},
	} {
		t.Run(tc.word, func(t *testing.T) {
			raw, err := d.Lookup(tc.word)
			if err != nil {
				t.Fatalf("%s is not in the committed corpus: %v", tc.word, err)
			}
			gloss, domain := senseFacts(tc.word, ParseEntry(raw))
			if gloss == "" {
				t.Error("no leading gloss extracted")
			}
			if domain != tc.wantDomain {
				t.Errorf("domain = %q, want %q", domain, tc.wantDomain)
			}
		})
	}

	// The property directly: a register label must never become a domain.
	// Every label readGloss can report on a non-domain axis has to be refused
	// here, or ParseDomain quietly turns it into `general`.
	// Whatever any corpus entry yields must be a real subject field or nothing.
	// `general` reaching this return is the tell that a register label was
	// accepted and then flattened by ParseDomain, losing a specialist label a
	// later sense carried.
	for _, w := range []string{"sycophantic", "run", "pulp", "minute", "present", "bank"} {
		raw, err := d.Lookup(w)
		if err != nil {
			continue
		}
		_, domain := senseFacts(w, ParseEntry(raw))
		if domain == store.DomainGeneral {
			t.Errorf("%q yielded the general fallback as a DOMAIN; a register label was accepted", w)
		}
		if domain == "" {
			continue
		}
		if _, ok := store.ParseDomain(string(domain)); !ok {
			t.Errorf("%q yielded %q, which is not in the closed set", w, domain)
		}
	}
}

// I-2's second green mutation: the dictionary's label WINS over the model's.
//
// The atlas states this as "Where the dictionary spoke, its label wins
// outright", and the plan calls it the milestone's payoff. Nothing checked it:
// replacing the precedence with the model's answer left the suite green, and the
// failure it allows is a model paraphrase overwriting an editorial fact into a
// cache that is never re-examined.
func TestTheDictionaryDomainBeatsTheModel(t *testing.T) {
	d, fake, st := harvestRig(t, 0)
	d.dict = testDict(t)
	// `record` rather than the rig's default deck: it is a corpus entry NOAD
	// labels `Law`, which is what makes the precedence observable at all.
	if err := d.deck.Upsert(store.Word{
		Text: "record", FirstSeen: harvestClock, LastSeen: harvestClock, Lookups: 1,
	}); err != nil {
		t.Fatal(err)
	}
	// The model is told the answer and asked to repeat it; here it paraphrases.
	fake.Script(markBand, llmtest.Reply{Text: `{"band":"C2","domain":"Politics"}`})
	scriptAll(fake, 8)

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}

	_, known := wordSense(d, "record")
	if known == "" {
		t.Fatal("`record` carries no NOAD subject label, so this test cannot see the " +
			"precedence it exists to pin — pick a corpus word that does")
	}
	f, err := st.WordFacts("record")
	if err != nil {
		t.Fatal(err)
	}
	if f.Domain != known {
		t.Errorf("domain = %q, want the dictionary's %q — a model paraphrase overwrote an editorial label",
			f.Domain, known)
	}
}

// I-2's third green mutation: the LANGUAGE reaches the request.
//
// TestBandPromptCarriesTheLanguage pins the renderer, but BR-3's operative
// sentence was "d.lang is already in scope at the call site; thread it" — and
// the threading is what nothing checked. Asserted on the wire, which is the only
// place the distinction between the two is visible.
func TestHarvestSendsTheDecksLanguage(t *testing.T) {
	d, fake, _ := harvestRig(t, 3)
	d.lang = store.Lang("es")
	// All but one banded: the unbanded word makes the BANDING pass run, and the
	// rest make a pool for authoring. Both passes in one invocation is the only
	// shape that can check every prompt's language at once.
	preBandExcept(t, d, deckWord(0))
	scriptAll(fake, 40)

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}

	// EVERY request, not reqs[0]. Asserting the first one covered the band
	// prompt and nothing else — so the author prompt and both judges could stop
	// receiving d.lang with this test green, which is exactly the gap M1's BR-3
	// named one layer down: the renderer was pinned and the THREADING was not.
	reqs := fake.Requests()
	if len(reqs) < 4 {
		t.Fatalf("only %d requests; this run must reach banding, authoring and both judges "+
			"or the assertion below covers one prompt", len(reqs))
	}
	seen := map[string]bool{}
	for i, r := range reqs {
		p := r.Prompt()
		for mark, name := range map[string]string{
			markBand: "band", markAuthor: "author", markEntail: "entail", markVeto: "veto",
		} {
			if strings.Contains(p, mark) {
				seen[name] = true
			}
		}
		if !strings.Contains(p, "`es`") {
			t.Errorf("request %d never names the deck's language; a Spanish deck would be "+
				"banded, authored and judged by English-assuming prompts. Prompt: %s", i, p)
		}
	}
	// The coverage claim itself, so a run that silently stopped reaching a task
	// cannot make this test pass by having nothing to check.
	for _, name := range []string{"band", "author", "entail", "veto"} {
		if !seen[name] {
			t.Errorf("no %s request was sent, so its language threading is unchecked", name)
		}
	}
}

// The mode rule, pinned on the RULE rather than on the pairs.
//
// modeCollision is what run() calls, so this covers every pair including the ones
// nobody has typed.
//
// AND THE LIST IS DERIVED, which it was not (#8). This comment used to claim "a
// sixth mode added to run()'s slice is covered by construction rather than by
// someone remembering", and main.go said "modeCollision's table test derives from
// this" — both false while the names were hand-written here. The PAIRS derived;
// the SET did not, so the sixth mode would have been the first one no pair test
// ever saw, in a place two comments called safe. `#8` is that sixth mode.
//
// Same move TestEveryFormIsEnrolled makes for forms: read the extent out of the
// code that owns it, and fail closed when the read finds less than the code
// declares.
func TestModeCollision(t *testing.T) {
	all := declaredModes(t)
	if _, _, clash := modeCollision(all); clash {
		t.Error("no mode requested reported a collision")
	}
	for i := range all {
		one := append([]mode(nil), all...)
		one[i].on = true
		if _, _, clash := modeCollision(one); clash {
			t.Errorf("%s alone reported a collision", all[i].name)
		}
		// EVERY pair, derived from the list rather than enumerated by hand: the
		// bug this replaces was -harvest refused beside -play and -reflect while
		// -forget and -llm-check silently swallowed it, because those two were
		// never written down.
		for j := range all {
			if i == j {
				continue
			}
			two := append([]mode(nil), all...)
			two[i].on, two[j].on = true, true
			a, b, clash := modeCollision(two)
			if !clash {
				t.Errorf("%s with %s was accepted; two modes is two commands on one line",
					all[i].name, all[j].name)
			}
			if a == "" || b == "" || a == b {
				t.Errorf("collision of %s and %s named %q and %q", all[i].name, all[j].name, a, b)
			}
		}
	}
}

// And the guard reached through run(), which is where the repo pins this class
// (play_loop_test.go does the same for "-forget takes the word to remove").
// modeCollision being right is not the same claim as run() calling it.
//
// DERIVED, LIKE EVERY OTHER MODE-SET ENUMERATION (#49 IV, 4th in family
// `two-commands-one-line`). This was a hand-typed table, and the commit series
// that argued a remembered enumeration IS the defect added three rows to it by
// hand — so the through-run() pair claim was the last mode set carried by memory,
// and an eighth mode would get no row. Every pair now comes from declaredModes,
// which is what makes the claim survive a mode nobody has typed yet.
//
// EVERY unordered pair, not a sample: modeCollision returns on the first two it
// finds, so a pair that dispatches before the check would be invisible to any
// subset that happened to miss it.
func TestRunRefusesTwoModes(t *testing.T) {
	strFlags := stringFlagNames(t)
	modes := declaredModesOrFail(t)
	for i, a := range modes {
		for _, b := range modes[i+1:] {
			args := append(argvForMode(strFlags, a.name), argvForMode(strFlags, b.name)...)
			t.Run(a.name+" "+b.name, func(t *testing.T) {
				var out, errb bytes.Buffer
				code := run(t.Context(), args, testDeps(t), strings.NewReader(""), &out, &errb)
				if code != 2 {
					t.Errorf("`define %s` exit = %d, want 2 — a dropped mode is a silently "+
						"different command", strings.Join(args, " "), code)
				}
				if !strings.Contains(errb.String(), "modes") {
					t.Errorf("stderr = %q, want it to name the collision", errb.String())
				}
			})
		}
	}
}

// flagsDeclaredWith maps `x := fs.<Kind>("name", …)` to x -> "-name".
//
// ONE WALKER, PARAMETERISED. There were two, differing only in the selector they
// matched and the map they built (#49 minor, ARCH-DRY) — and two copies of an AST
// walk is two things to widen the next time a flag is declared a new way.
func flagsDeclaredWith(t *testing.T, kind string) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing main.go: %v", err)
	}
	out := map[string]string{}
	ast.Inspect(file, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		id, ok := as.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}
		call, ok := as.Rhs[0].(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != kind {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if name, err := strconv.Unquote(lit.Value); err == nil {
			out[id.Name] = "-" + name
		}
		return true
	})
	return out
}

// declaredModesOrFail is declaredModes plus the floor both set-enumerating tests
// need, in one place rather than two byte-identical copies (#49 minor, ARCH-DRY).
//
// The floor is the #12 BR-17 rule: an extraction finding FEWER members than the
// code declares is under-deriving, and every check built on it is then certifying
// a set nobody chose.
func declaredModesOrFail(t *testing.T) []mode {
	t.Helper()
	modes := declaredModes(t)
	if len(modes) < 7 {
		t.Fatalf("derived %d modes; run() declares at least seven, so this check is "+
			"under-deriving and would certify a set nobody chose", len(modes))
	}
	return modes
}

// argvForMode is one mode as a user would type it: a string flag needs a value,
// a bool flag is the name alone. Derived, so `-forget` is not special-cased by
// memory in the two tests that build argv.
func argvForMode(strFlags map[string]bool, name string) []string {
	if strFlags[name] {
		return []string{name, "x"}
	}
	return []string{name}
}

// stringFlagNames is the SET of flag names declared with fs.String.
func stringFlagNames(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, name := range flagsDeclaredWith(t, "String") {
		out[name] = true
	}
	return out
}

// EVERY MODE REFUSES A TRAILING WORD, and the list is DERIVED rather than
// remembered — which is the whole finding (#49 I-2, 3rd in family
// `two-commands-one-line`).
//
// The mode x mode half already closed this way: TestModeCollision builds every
// pair from declaredModes, so a seventh mode gets pair coverage for free. The
// mode x WORD half did not — run() carried six hand-written
// `case *X && fs.NArg() != 0:` arms and the tests hand-listed which ones they
// checked. So a mode added tomorrow joined the pair matrix automatically and got
// NO word coverage, which is exactly how -version reached production swallowing
// one (and -llm-check before it).
//
// Iterating declaredModes closes it: an eighth mode is covered the day its row is
// added, with nobody remembering to extend a table.
func TestEveryModeRefusesATrailingWord(t *testing.T) {
	// A REMOTE provider with no key, so a REGRESSION cannot reach the network:
	// if -llm-check ever stops refusing, it dispatches for real, and an empty
	// environment resolves against the local proxy and spends tokens
	// (llmcheck_test.go:135 writes the rule down; the review measured this test
	// making a live 1.2s call before these lines existed).
	t.Setenv("DEFINE_LLM_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("DEFINE_LLM_BASE_URL", "https://api.anthropic.com")

	strFlags := stringFlagNames(t)
	modes := declaredModesOrFail(t)
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			args := append(argvForMode(strFlags, m.name), "cat")
			var out, errb bytes.Buffer
			code := run(t.Context(), args, testDeps(t), strings.NewReader(""), &out, &errb)
			if code != 2 {
				t.Errorf("`define %s` exit = %d, want 2 — a mode plus a word is two "+
					"commands on one line, and swallowing one is a silently different "+
					"command. Every mode refuses this; add the arm beside its siblings "+
					"in run()'s argument-count switch.",
					strings.Join(args, " "), code)
			}
			if out.Len() != 0 {
				t.Errorf("`define %s` wrote %q to stdout — it should refuse, not act",
					strings.Join(args, " "), out.String())
			}
		})
	}
}

// I-3's class: a guard that only exists on the run() path is pinned THROUGH
// run().
//
// Round 3 named the enumeration and one member of it was swept. The rest are
// here, derived from the switch arms rather than from what anyone remembered:
// every arm that refuses, plus the bare-flag default, plus the wiring hop.
//
// These are regression pins, not bug reports — each behaves correctly today. The
// exposure is that runHarvest's own tests construct deps directly, so they begin
// AFTER the hop that fills them, which is the class news_test.go names.
func TestRunHarvestUsageErrors(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		args       []string
	}{
		{"a word beside -harvest", "takes no word", []string{"-harvest", "sycophantic"}},
		{"-limit without -harvest", "only mean anything with", []string{"-limit", "5", "sycophantic"}},
		{"-agreement without -harvest", "only mean anything with", []string{"-agreement", "3", "sycophantic"}},
		{"a negative -limit", "cannot be negative", []string{"-harvest", "-limit", "-1"}},
		{"a negative -agreement", "cannot be negative", []string{"-harvest", "-agreement", "-1"}},
		{"-agreement past its cap", "is capped at", []string{"-harvest", "-agreement", "9999"}},
		{"-limit with -agreement", "does not apply to -agreement", []string{"-harvest", "-agreement", "3", "-limit", "5"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			code := run(t.Context(), tc.args, testDeps(t), strings.NewReader(""), &out, &errb)
			if code != 2 {
				t.Errorf("exit = %d, want 2 (stderr %q)", code, errb.String())
			}
			if !strings.Contains(errb.String(), tc.want) {
				t.Errorf("stderr = %q, want it to mention %q", errb.String(), tc.want)
			}
		})
	}
}

// The bare -agreement default, and the WIRING HOP.
//
// Both are things runHarvest's own tests structurally cannot see: they build
// deps by hand, so they start after `d = d.withStore(...)` has filled d.deck,
// d.dict and d.lang. This drives the whole program, so the hop is exercised —
// the class news_test.go calls "a test that constructs the struct begins AFTER
// the hop that fills its fields", three issues and counting.
func TestRunHarvestThroughTheWiringHop(t *testing.T) {
	// testDeps loads the committed dictionary corpus from a RELATIVE path, so it
	// is built before the chdir — openStore is what must see the temp directory,
	// not the fixture loader.
	d := testDeps(t)
	fake := llmtest.NewFake(t)
	scriptAll(fake, 8)

	dir := t.TempDir()
	st := store.NewYAML(dir, store.DefaultLang, nil)
	if err := st.Upsert(store.Word{
		Text: "sycophantic", FirstSeen: harvestClock, LastSeen: harvestClock, Lookups: 1,
	}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	d.newStore = openStore // the real wiring, not a hand-filled struct
	d.clock = store.FixedClock(harvestClock)
	d.newLLM = llm.New
	d.getenv = envFor(fake.URL)

	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-harvest"}, d, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errb.String())
	}
	// The deck the hop opened is the deck that was banded.
	f, err := st.WordFacts("sycophantic")
	if err != nil {
		t.Fatal(err)
	}
	if !f.Harvested() {
		t.Error("run() reached runHarvest but nothing was banded; the store wiring did not arrive")
	}

	// And a bare -agreement takes the documented default rather than erroring.
	var out2, errb2 bytes.Buffer
	scriptAll(fake, agreementRounds+4)
	if code := run(t.Context(), []string{"-harvest", "-agreement", "0"}, d,
		strings.NewReader(""), &out2, &errb2); code != 0 {
		t.Fatalf("-agreement=0 exit = %d, stderr %q", code, errb2.String())
	}
	if !strings.Contains(out2.String(), "x "+itoa(agreementRounds)+" assignment") {
		t.Errorf("-agreement=0 did not take the default of %d rounds: %q", agreementRounds, out2.String())
	}
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

// declaredModes reads run()'s mode list out of main.go.
//
// PARSED, NOT LISTED. The names live in one place — the `modes := []mode{...}`
// literal that run() hands to modeCollision — and this reads them there, so a
// mode added to run() joins every pair check without anyone remembering.
//
// It fails closed on the count for the reason #12 BR-17 established: an
// extraction that finds FEWER members than the code declares is under-deriving,
// and every guard built on it is then checking a set nobody chose.
func declaredModes(t *testing.T) []mode {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing main.go: %v", err)
	}
	var names []string
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		at, ok := lit.Type.(*ast.ArrayType)
		if !ok {
			return true
		}
		if id, ok := at.Elt.(*ast.Ident); !ok || id.Name != "mode" {
			return true
		}
		for _, el := range lit.Elts {
			row, ok := el.(*ast.CompositeLit)
			if !ok || len(row.Elts) == 0 {
				continue
			}
			s, ok := row.Elts[0].(*ast.BasicLit)
			if !ok || s.Kind != token.STRING {
				t.Errorf("%s: a mode's name is not a string literal, so the set of "+
					"modes cannot be known statically", fset.Position(row.Pos()))
				continue
			}
			name, err := strconv.Unquote(s.Value)
			if err != nil {
				t.Fatalf("%s: %v", fset.Position(row.Pos()), err)
			}
			names = append(names, name)
		}
		return true
	})
	if len(names) < 5 {
		t.Fatalf("found %d modes in main.go %v; run() declares at least five, so this "+
			"derivation is under-deriving and every check built on it would be "+
			"certifying a set nobody chose", len(names), names)
	}
	out := make([]mode, 0, len(names))
	for _, n := range names {
		out = append(out, mode{name: n})
	}
	return out
}

// EVERY MODE run() DISPATCHES IS IN THE COLLISION LIST (#8 BR-1).
//
// declaredModes reads the LIST, so it derives whatever is there — and removing a
// row leaves it deriving one fewer, silently. The boundary review measured
// exactly that: deleting `{"-stats", *statsFlag}` left the whole package green,
// so the mode would still dispatch while colliding with nothing.
//
// The extent that matters is therefore not the list but the DISPATCH: a flag
// run() returns on is a mode, whether or not anyone remembered to write it down.
// This reads both out of main.go and requires them to agree, which is the same
// both-directions closure #46 BR-9 arrived at — one side cannot hide what the
// other declares.
func TestEveryDispatchedModeIsInTheCollisionList(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing main.go: %v", err)
	}

	// 1. Every `x := fs.Bool("name", …)` — the variable a flag is read through,
	//    and the name a user types. Shared with stringFlagNames' walker.
	flagName := flagsDeclaredWith(t, "Bool")
	// A mode can also be a bool LOCAL that names its flag directly —
	// `forgetting := isSet(fs, "forget")`. That is one of the six, and the first
	// version of this guard missed it (#8 BR-11): it found five, and its
	// hand-typed floor of five certified exactly the gap it existed to catch.
	//
	// Read from the CALL rather than from the variable's name: `isSet(fs, "x")`
	// says which flag it is, so nothing here has to guess from spelling.
	ast.Inspect(file, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		id, ok := as.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}
		call, ok := as.Rhs[0].(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			return true
		}
		fn, ok := call.Fun.(*ast.Ident)
		if !ok || fn.Name != "isSet" {
			return true
		}
		lit, ok := call.Args[1].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if name, err := strconv.Unquote(lit.Value); err == nil {
			flagName[id.Name] = "-" + name
		}
		return true
	})

	// 2. Every `if *x { return runY(…) }` — a flag run() RETURNS on is a mode.
	dispatched := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		is, ok := n.(*ast.IfStmt)
		if !ok || is.Cond == nil || is.Else != nil {
			return true
		}
		// TWO SHAPES, because run() has two. A bool flag is read through a
		// pointer (`if *playFlag {`); a bool LOCAL derived from a string flag is
		// read directly (`if forgetting {`). Recognising only the first is what
		// left -forget uncovered (#8 BR-11).
		var id *ast.Ident
		switch cond := is.Cond.(type) {
		case *ast.StarExpr:
			id, _ = cond.X.(*ast.Ident)
		case *ast.Ident:
			id = cond
		}
		if id == nil {
			return true
		}
		name, ok := flagName[id.Name]
		if !ok {
			return true
		}
		// The body must RETURN — ANY return, whatever the expression. What makes a
		// flag a mode is that run() ENDS on it; the shape of what it hands back is
		// incidental.
		//
		// This required a `return <call>` and so failed OPEN on #49's `-version`,
		// whose body was `fmt.Fprintln(...); return 0`. A BasicLit is not a
		// CallExpr, so the parse found 6 modes, the list declared 6, the floor
		// agreed with itself, and `define --version --play` silently swallowed
		// --play. That is the THIRD instance of the collision-list bug this guard
		// exists to end, and the second time the guard itself was the reason it
		// went unseen (#8 BR-11 was the first: it recognised only `if *x`, so
		// -forget's `if forgetting` slipped through).
		//
		// Widening to any return makes the guard shape-INDEPENDENT, so an eighth
		// mode cannot repeat this by being written a new way. It cannot over-match:
		// the condition must already be a bare flag ident, so a validation branch
		// like `if *harvest && fs.NArg() != 0 { return 2 }` is a BinaryExpr and
		// never reaches here.
		for _, stmt := range is.Body.List {
			if ret, ok := stmt.(*ast.ReturnStmt); ok && len(ret.Results) == 1 {
				dispatched[name] = true
			}
		}
		return true
	})
	listed := map[string]bool{}
	for _, m := range declaredModes(t) {
		listed[m.name] = true
	}
	// THE FLOORS DERIVE FROM EACH OTHER, not from typed numbers. The first
	// version calibrated against three hand-written constants (#8 BR-11), and
	// one of them — "at least five dispatched modes" — was exactly the count a
	// derivation missing `-forget` produced, so the floor certified the gap it
	// was meant to catch.
	//
	// The two sides are independent readings of one fact, so each is the other's
	// floor: fewer dispatches than the list declares means this parse missed a
	// shape, and that is a failure OF THIS GUARD rather than of the code.
	if len(dispatched) < len(listed) {
		t.Errorf("the dispatch parse found %d modes %v but the collision list declares "+
			"%d %v — this guard is missing a dispatch SHAPE (a mode wired some way it "+
			"does not recognise), so it would certify a set nobody chose.",
			len(dispatched), keysOfBool(dispatched), len(listed), keysOfBool(listed))
	}
	for name := range dispatched {
		if !listed[name] {
			t.Errorf("run() returns on %s, so it is a MODE, but it is not in the "+
				"collision list — it would coexist silently with every other mode, "+
				"and `define %s -play` would honour one of them without saying so.",
				name, name)
		}
	}
}

// keysOfBool is a stable listing for a diagnostic.
func keysOfBool(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

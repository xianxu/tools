package main

import (
	"bytes"
	"context"
	"fmt"
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
func TestHarvestStopsAtTheLimit(t *testing.T) {
	// The deck must EXCEED the limit for the cap to be observable at all. The
	// first version banded 2 of 6 and then asserted authoring made at most 2
	// calls — but banding had capped the pool AT 2, so the assertion held with
	// the check deleted entirely.
	d, fake, _ := harvestRig(t, 8)
	scriptAll(fake, 60)

	const limit = 5
	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{limit: limit}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}

	// -limit names MODEL CALLS, so that is what it must bound — across every
	// pass, not per pass. Counting successes let a run whose judge rejected
	// everything make one author call and one entail call per deck word.
	total := len(fake.Requests())
	if total > limit {
		t.Errorf("made %d model calls with -limit %d (band %d, author %d, entail %d, veto %d); "+
			"the flag names calls and must bound every pass",
			total, limit, countTask(fake, markBand), countTask(fake, markAuthor),
			countTask(fake, markEntail), countTask(fake, markVeto))
	}
	if total == 0 {
		t.Fatal("no calls at all, so this assertion cannot fail")
	}
	// A capped run is a PARTIAL run and has to say so.
	if !strings.Contains(out.String(), "run again") {
		t.Errorf("a capped run did not say it was partial: %q", out.String())
	}
}

// The budget survives a pass that rejects everything — the shape that made the
// original counter wrong, since a rejected word charged nothing.
func TestTheLimitHoldsWhenEveryStemIsRejected(t *testing.T) {
	d, fake, _ := harvestRig(t, 8)
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
	scriptAll(fake, 8)

	var out, errOut bytes.Buffer
	if code := runHarvest(context.Background(), d, options{}, harvestOptions{}, &out, &errOut); code != 0 {
		t.Fatalf("run = %d, stderr: %s", code, errOut.String())
	}
	reqs := fake.Requests()
	if len(reqs) == 0 {
		t.Fatal("no request was sent")
	}
	if !strings.Contains(reqs[0].Prompt(), "`es`") {
		t.Errorf("the request never names the deck's language; a Spanish deck would be "+
			"banded from an English prompt, forever. Prompt: %s", reqs[0].Prompt())
	}
}

// The mode rule, pinned on the RULE rather than on the pairs.
//
// modeCollision is what run() calls, so this covers every pair including the
// ones nobody has typed — and a sixth mode added to run()'s slice is covered by
// construction rather than by someone remembering to add a case here.
func TestModeCollision(t *testing.T) {
	all := []mode{
		{"-llm-check", false}, {"-forget", false}, {"-play", false},
		{"-reflect", false}, {"-harvest", false},
	}
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
func TestRunRefusesTwoModes(t *testing.T) {
	for _, args := range [][]string{
		{"-forget", "x", "-harvest"},
		{"-llm-check", "-harvest"},
		{"-play", "-harvest"},
		{"-reflect", "-harvest"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var out, errb bytes.Buffer
			code := run(t.Context(), args, testDeps(t), strings.NewReader(""), &out, &errb)
			if code != 2 {
				t.Errorf("exit = %d, want 2 — a dropped mode is a silently different command", code)
			}
			if !strings.Contains(errb.String(), "modes") {
				t.Errorf("stderr = %q, want it to name the collision", errb.String())
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

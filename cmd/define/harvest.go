package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
)

// harvestLimit bounds the words one --harvest run will ask the model about.
//
// ARCH-CONSTRAINTS. "Bounded in practice by the deck" is not a bound: a deck
// reaches thousands, this is the first path in the program that makes many calls
// in a row, and the first run against a large deck is exactly where an unbounded
// loop is discovered. poolCap has the same shape for the same reason.
//
// 200 rather than 40, because unlike a sitting's option pool this work is
// PERMANENT — every call buys a fact cached forever — and a capped run is a
// partial run rather than a failed one: the cache is its own progress marker, so
// running again picks up where this stopped. The number is a cost ceiling per
// invocation, not a judgement about how many words deserve banding.
const harvestLimit = 200

// agreementSample and agreementRounds are the measurement mode's shape.
//
// K words, N times each, is 100 calls — deliberately its own invocation rather
// than something --harvest does on the way past. See runHarvestAgreement.
//
// agreementRounds is the DEFAULT a bare -agreement takes, which is what the
// issue's Done-when and the plan both describe. It was declared and never read
// in the first cut, so `-agreement` without a number was a flag error and the
// documented shape did not exist.
//
// agreementMaxRounds bounds the other end: K x N calls with no ceiling is the
// same unbounded-batch shape --limit exists to prevent, one flag over.
const (
	agreementSample    = 20
	agreementRounds    = 5
	agreementMaxRounds = 25
)

// optionsPerItem is three wrong answers beside one right one — four options,
// which is what play.maxOptions can express and what the 1-4 key range grades.
const optionsPerItem = 3

// harvestOptions is what the flags decide.
type harvestOptions struct {
	limit int
	// agreement > 0 selects the MEASUREMENT mode, which writes nothing.
	agreement int
}

// runHarvest assigns a band and a domain to every unbanded word in the deck.
//
// A BATCH path, and the only one in this program that may block: a review
// sitting must never wait on it, which is why nothing here is reachable from
// --play. The work is per NEW word, so a second run over an unchanged deck makes
// ZERO model calls — that is what keeps the cost from growing with time, and it
// is a claim about CALLS rather than about a file existing.
func runHarvest(ctx context.Context, d deps, opt options, ho harvestOptions, out, errOut io.Writer) int {
	if d.deck == nil {
		fmt.Fprintln(errOut, noDeckMessage(opt.noCapture))
		return 1
	}
	deck, err := d.deck.Deck()
	if err != nil {
		fmt.Fprintf(errOut, "define: could not read the deck: %v\n", err)
		return 1
	}
	if len(deck) == 0 {
		fmt.Fprintln(errOut, "define: the deck is empty; look some words up first")
		return 1
	}
	if d.getenv == nil || d.newLLM == nil {
		return unavailableToHarvest(errOut)
	}
	cfg, err := llm.Resolve(d.getenv)
	if err != nil {
		return unavailableToHarvest(errOut)
	}
	client := d.newLLM(cfg)

	if ho.agreement > 0 {
		return runHarvestAgreement(ctx, d, client, deck, ho.agreement, out, errOut)
	}

	limit := ho.limit
	if limit <= 0 {
		limit = harvestLimit
	}

	var asked, skipped, refused int
	for _, w := range deck {
		if asked >= limit {
			fmt.Fprintf(out, "define: stopped at the --limit of %d; run again to continue\n", limit)
			break
		}
		// The cache check comes FIRST, before anything reaches for the network.
		// "Cached forever" is this branch.
		facts, err := d.deck.WordFacts(w.Text)
		if err != nil {
			fmt.Fprintf(errOut, "define: could not read facts for %q: %v\n", w.Text, err)
			return 1
		}
		if facts.Harvested() {
			skipped++
			continue
		}

		gloss, known := wordSense(d, w.Text)
		claim, err := llm.Run(ctx, client, bandTask(d.lang, w.Text, gloss, known))
		if err != nil {
			// STOP, and leave the store as it is. Everything banded before this
			// point is already durable — each word is written atomically as it is
			// answered — so an outage costs the rest of the run and nothing that
			// was already bought. Continuing past an error would burn a
			// rate-limited budget on calls that will each fail the same way.
			fmt.Fprintf(errOut, "define: harvesting stopped: %v\n", err)
			fmt.Fprintf(out, "define: banded %d word(s) before stopping; they are saved\n", asked)
			return 1
		}
		asked++

		band, ok := store.ParseBand(claim.Band)
		if !ok {
			// Counted and skipped, not coerced. The word stays unbanded and the
			// next run asks again, which is the cheap failure; a nonsense band
			// written to a forever cache is the expensive one.
			refused++
			fmt.Fprintf(errOut, "define: %q: %q is not a CEFR band; leaving it unbanded\n", w.Text, claim.Band)
			continue
		}
		// The DICTIONARY wins where it spoke. The model was told the answer and
		// asked to repeat it, so this is belt-and-braces rather than a
		// disagreement to resolve — but a model that paraphrased anyway must not
		// be able to overwrite an editorial label with a guess.
		domain := known
		if domain == "" {
			domain, _ = store.ParseDomain(claim.Domain)
		}
		if err := d.deck.SetWordFacts(w.Text, store.WordFacts{
			Band: band, Domain: domain, At: now(d),
		}); err != nil {
			fmt.Fprintf(errOut, "define: could not save facts for %q: %v\n", w.Text, err)
			return 1
		}
	}

	fmt.Fprintf(out, "define: %d word(s) banded, %d already known", asked-refused, skipped)
	if refused > 0 {
		fmt.Fprintf(out, ", %d refused", refused)
	}
	fmt.Fprintln(out, ".")

	return runAuthoring(ctx, d, client, limit, out, errOut)
}

// runAuthoring writes practice items for banded words that have none.
//
// SECOND PASS, after banding, and it depends on the first having run: selection
// draws from the deck's BANDED words, so a word cannot supply a distractor until
// it has a level. That ordering is why a single --harvest does both rather than
// authoring being its own flag — the two are one job with a dependency, not two
// jobs a person chooses between.
func runAuthoring(ctx context.Context, d deps, client llm.Client, limit int, out, errOut io.Writer) int {
	deck, err := d.deck.Deck()
	if err != nil {
		fmt.Fprintf(errOut, "define: could not read the deck: %v\n", err)
		return 1
	}

	// THE POOL is every banded word, built once. Selection is pure over it, so
	// this is the only place the store is read for candidates.
	var pool []bandedWord
	for _, w := range deck {
		f, err := d.deck.WordFacts(w.Text)
		if err != nil {
			fmt.Fprintf(errOut, "define: could not read facts for %q: %v\n", w.Text, err)
			return 1
		}
		if f.Harvested() {
			pool = append(pool, bandedWord{Word: w.Text, Facts: f})
		}
	}
	pool = sortedBanded(pool)
	if len(pool) < 2 {
		fmt.Fprintln(out, "define: too few banded words to select wrong answers from; harvest more first.")
		return 0
	}

	learner := readLearner(d.deck)
	var authored, skipped, rejected int
	var domains []store.Domain
	widened := map[selectionTier]int{}
	// How often each word has already served as a wrong answer in THIS batch.
	// Threaded through selection so a batch spreads its distractors rather than
	// leaning on whichever words happen to be eligible — see pickDistractors.
	served := map[string]int{}

	for _, c := range pool {
		if authored >= limit {
			fmt.Fprintf(out, "define: stopped authoring at the --limit of %d; run again to continue\n", limit)
			break
		}
		existing, err := d.deck.Items(c.Word)
		if err != nil {
			fmt.Fprintf(errOut, "define: could not read items for %q: %v\n", c.Word, err)
			return 1
		}
		if len(existing) > 0 {
			skipped++
			continue
		}

		gloss, _ := wordSense(d, c.Word)
		stem, err := llm.Run(ctx, client, authorTask(d.lang, c.Word, gloss, c.Facts, learner))
		if err != nil {
			fmt.Fprintf(errOut, "define: authoring stopped: %v\n", err)
			fmt.Fprintf(out, "define: authored %d item(s) before stopping; they are saved\n", authored)
			return 1
		}

		// THE FREE CHECK FIRST. A stem that does not contain its answer cannot be
		// rendered as a question, and finding that out costs nothing — so it runs
		// before the judge is paid to have an opinion about it.
		if !stemUsesTheWord(stem.Stem, c.Word) {
			rejected++
			fmt.Fprintf(errOut, "define: %q: the stem does not use the word; leaving it unauthored\n", c.Word)
			continue
		}

		// THE ENTAILMENT JUDGE, before any distractor is selected. A stem that
		// does not entail its answer cannot be rescued by better wrong answers,
		// so judging first is what stops the veto being spent on a doomed item.
		verdict, err := llm.Run(ctx, client, entailTask(c.Word, stem.Stem))
		if err != nil {
			fmt.Fprintf(errOut, "define: judging stopped: %v\n", err)
			return 1
		}
		if !verdict.Entails || verdict.Glosses || !verdict.Named {
			rejected++
			fmt.Fprintf(errOut, "define: %q: stem rejected (%s); leaving it unauthored\n", c.Word, verdict.Reason)
			continue
		}

		// SELECTED, never invented, and then vetoed one pair at a time.
		candidates, tier := pickDistractors(c.Word, c.Facts, learner.Band, pool, optionsPerItem,
			// optionpool's seedFor, reused rather than reimplemented (ARCH-DRY):
			// it is already variadic, already pinned by TestSeedForIsPinned, and
			// the property wanted here is the same one — a stable sequence this
			// repo owns rather than a stdlib version's.
			//
			// NO DAY PART, unlike a sitting's seed. #7 varies options per day so a
			// learner cannot learn a position; an authored item is written ONCE and
			// cached forever, so varying it by day would make two runs of the same
			// deck produce different material and a bad batch undebuggable.
			seedFor("harvest-options", c.Word), served)
		widened[tier]++
		var kept []string
		for _, cand := range candidates {
			v, err := llm.Run(ctx, client, vetoTask(c.Word, stem.Stem, cand))
			if err != nil {
				fmt.Fprintf(errOut, "define: the veto stopped: %v\n", err)
				return 1
			}
			if v.Fits {
				// The veto EXERCISED. Said out loud because a veto nobody sees
				// work is a veto nobody can trust.
				fmt.Fprintf(errOut, "define: %q: vetoed %q (%s)\n", c.Word, cand, v.Reason)
				continue
			}
			kept = append(kept, cand)
		}
		for _, k := range kept {
			// Counted only for options that SURVIVED the veto: a vetoed candidate
			// was never shown, so charging it would push the next item away from a
			// word this batch has not actually used.
			served[store.Key(k)]++
		}
		if len(kept) == 0 {
			rejected++
			fmt.Fprintf(errOut, "define: %q: every candidate was vetoed; leaving it unauthored\n", c.Word)
			continue
		}

		if err := d.deck.SetItems(c.Word, []store.Item{{
			Word: c.Word, Form: store.FormCloze, Stem: stem.Stem,
			Answer: c.Word, Distractors: kept, At: now(d),
		}}); err != nil {
			fmt.Fprintf(errOut, "define: could not save items for %q: %v\n", c.Word, err)
			return 1
		}
		authored++
		domains = append(domains, c.Facts.Domain)
	}

	fmt.Fprintf(out, "define: %d item(s) authored, %d already had material", authored, skipped)
	if rejected > 0 {
		fmt.Fprintf(out, ", %d rejected", rejected)
	}
	fmt.Fprintln(out, ".")
	if authored > 0 {
		// REPORTED, not buried, and measured with NO MODEL — a judge scoring its
		// own batch's variety is the self-oracle problem the atlas records.
		fmt.Fprintf(out, "define: topic spread %.2f across the batch.\n", topicSpread(domains))
	}
	// How far selection had to widen is a fact about the DECK, and the difference
	// between "these wrong answers are pitched" and "these were what was lying
	// around".
	for _, t := range []selectionTier{tierGeneral, tierAnyDomain, tierAboveBand} {
		if widened[t] > 0 {
			fmt.Fprintf(out, "define: %d item(s) drew options from %s.\n", widened[t], t)
		}
	}
	return 0
}

// authorTask, entailTask and vetoTask are the one place each request is built,
// so no caller can drift into asking a different question — the rule bandTask's
// comment states and the conformance row once broke from outside.
func authorTask(lang store.Lang, word, gloss string, facts store.WordFacts, learner learnerFacts) llm.Task[authoredStem] {
	req := renderAuthorPrompt(lang, word, gloss, facts, learner)
	return llm.Task[authoredStem]{Name: req.Task, System: req.System, Prompt: req.Prompt}
}

func entailTask(word, stem string) llm.Task[entailVerdict] {
	req := renderEntailPrompt(word, stem)
	return llm.Task[entailVerdict]{Name: req.Task, System: req.System, Prompt: req.Prompt}
}

func vetoTask(answer, stem, candidate string) llm.Task[vetoVerdict] {
	req := renderVetoPrompt(answer, stem, candidate)
	return llm.Task[vetoVerdict]{Name: req.Task, System: req.System, Prompt: req.Prompt}
}

// runHarvestAgreement measures how stable the model's banding is, and WRITES
// NOTHING.
//
// Its own mode rather than a number --harvest prints on the way past, because
// the two cannot coexist: --harvest promises one call per unbanded word and zero
// calls on a second run, and measuring N assignments of the same word breaks
// both. Splitting them keeps those properties true of the path that runs every
// day and puts the cost of measurement where somebody is choosing to pay it.
//
// Reported as AGREEMENT, with no claim about correctness — see agreement's own
// comment for what that does and does not buy.
func runHarvestAgreement(ctx context.Context, d deps, client llm.Client, deck []store.Word, rounds int, out, errOut io.Writer) int {
	sample := make([]store.Word, 0, agreementSample)
	for _, w := range deck {
		if len(sample) == agreementSample {
			break
		}
		facts, err := d.deck.WordFacts(w.Text)
		if err != nil {
			fmt.Fprintf(errOut, "define: could not read facts for %q: %v\n", w.Text, err)
			return 1
		}
		// ALREADY-BANDED words only: the question is whether re-asking gives the
		// same answer the cache is holding, and an unbanded word has no answer to
		// be stable about.
		if facts.Harvested() {
			sample = append(sample, w)
		}
	}
	if len(sample) == 0 {
		fmt.Fprintln(errOut, "define: nothing is banded yet; run --harvest before measuring it")
		return 1
	}

	total := 0.0
	for _, w := range sample {
		// Hoisted: the dictionary's answer does not change between rounds, and
		// asking it N times was N ParseEntry walks per word for one result.
		gloss, known := wordSense(d, w.Text)
		bands := make([]store.Band, 0, rounds)
		for i := 0; i < rounds; i++ {
			claim, err := llm.Run(ctx, client, bandTask(d.lang, w.Text, gloss, known))
			if err != nil {
				fmt.Fprintf(errOut, "define: measurement stopped: %v\n", err)
				return 1
			}
			// Unparsed answers go in as themselves, so refusals count against
			// stability rather than being quietly dropped from the denominator.
			bands = append(bands, store.Band(claim.Band))
		}
		total += agreement(bands)
	}

	mean := total / float64(len(sample))
	fmt.Fprintf(out, "define: banding agreement %.2f over %d word(s) x %d assignment(s).\n",
		mean, len(sample), rounds)
	// Said out loud every time, because the number invites exactly the reading it
	// does not support.
	fmt.Fprintln(out, "define: this is STABILITY, not correctness — a consistently wrong scale scores 1.00.")
	return 0
}

// bandTask is the one place a word becomes a request, so --harvest and the
// measurement mode cannot drift into asking different questions — which would
// make the measurement a report about a prompt nobody runs.
func bandTask(lang store.Lang, word, gloss string, known store.Domain) llm.Task[bandClaim] {
	req := renderBandPrompt(lang, word, gloss, known)
	return llm.Task[bandClaim]{Name: req.Task, System: req.System, Prompt: req.Prompt}
}

// wordSense asks the dictionary what it already knows: the word's leading sense,
// and the subject field NOAD printed on it if any.
//
// This is the call --harvest makes INSTEAD of a model call, not before one. A
// domain read off a dictionary's editorial prose is free, offline, and more
// reliable than a model re-deriving it; the model is the fallback for words NOAD
// leaves unlabelled, which is most of them.
//
// A word the dictionary cannot find yields nothing and is not an error: the
// model can still band a word from its name alone, and refusing to would make
// the deck's coverage depend on the local dictionary's.
func wordSense(d deps, word string) (gloss string, domain store.Domain) {
	if d.dict == nil {
		return "", ""
	}
	text, err := d.dict.Lookup(word)
	if err != nil {
		return "", ""
	}
	return senseFacts(word, ParseEntry(text))
}

// senseFacts is the PURE half: an entry in, the leading gloss and the first
// subject label out.
//
// Split from wordSense because the IO is one line and the judgement is the rest,
// and the judgement is what needs pinning — the axis filter below was mutable
// with the whole package green until this had a table test (ARCH-PURE: if a test
// needs a dict fake to exercise a derivation, the derivation is in the wrong
// function).
func senseFacts(word string, e Entry) (gloss string, domain store.Domain) {
	if !entryDefines(word, e) {
		return "", ""
	}
	for _, b := range e.Blocks {
		for _, s := range b.Senses {
			f := readGloss(s.Gloss)
			if !f.Usable {
				continue
			}
			if gloss == "" {
				gloss = f.Text
			}
			// The first DOMAIN label in document order, not the first label of
			// any kind: readGloss also reports REGISTER (`informal`, `archaic`),
			// which is a different axis and not a subject field. Without this
			// filter an `informal` sense would be stored as a word's domain, and
			// ParseDomain would then flatten it to `general` — silently losing
			// the specialist label a later sense actually carried.
			if domain == "" && f.Axis == play.AxisDomain {
				if parsed, ok := store.ParseDomain(f.Label); ok {
					domain = parsed
				}
			}
		}
	}
	return gloss, domain
}

// now reads the injected clock, falling back to the system one — the same
// nil-merge rule the store-backed deps use, so a test that does not care about
// time need not wire a clock.
func now(d deps) time.Time {
	if d.clock != nil {
		return d.clock.Now()
	}
	return time.Now()
}

func unavailableToHarvest(errOut io.Writer) int {
	fmt.Fprintln(errOut, "define: --harvest needs the model seam; set ANTHROPIC_API_KEY (see --llm-check)")
	return 1
}

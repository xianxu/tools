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
	return 0
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

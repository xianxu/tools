package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
)

// authorTaskName names this request: the golden file, the cassette key, the
// usage label.
const authorTaskName = "authored-item"

// authoredStem is the typed answer: the sentence the model writes.
//
// The DISTRACTORS are deliberately not a field. They are selected from the deck
// here, never asked for — the whole point of the Spec's "selected, never
// invented" is that a model asked to write four options writes four it can
// justify, and one of them is often also correct.
type authoredStem struct {
	Stem string `json:"stem"`
}

// learnerFacts is what authoring reads off #17's learner model.
//
// Two halves, and the split is deliberate. The RAW markdown goes into the prompt
// because its `directive` column ("Draw comparables from judicial prose") is
// prose written for a model to read, and folding it into an enum would throw
// away the thing #17 exists to produce. The TYPED halves — a band and closed
// domains — are what selection does arithmetic on, and arithmetic cannot run on
// prose.
type learnerFacts struct {
	Band    store.Band
	Domains []store.Domain
	Model   string
}

// readLearner loads the learner model, or returns the zero value.
//
// AN ABSENT MODEL IS NOT AN ERROR — it is the normal first-run state and the
// Spec says so. Everything downstream degrades to generic authoring: the prompt
// carries no learner section, and selection falls back to the target word's own
// band (see pickDistractors).
func readLearner(s store.Store) learnerFacts {
	if s == nil {
		return learnerFacts{}
	}
	md, err := s.UserModel()
	if err != nil || strings.TrimSpace(md) == "" {
		return learnerFacts{}
	}
	band, _ := parseLearnerBand(md)
	return learnerFacts{Band: band, Domains: parseLearnerDomains(md), Model: md}
}

// parseLearnerDomains folds #17's free-text domain claims onto the closed set.
//
// #17 writes what the model said — `law`, `business news` — into a markdown
// table. This reads the first column and keeps only what ParseDomain recognises.
//
// AN UNMAPPED CLAIM IS IGNORED, not an error and NOT a new domain. That is a
// real narrowing and it is recorded rather than discovered: `business news` is a
// genuine thing this learner reads, NOAD prints no such field label, and so it
// contributes nothing to selection until someone decides it deserves a row. The
// alternative — letting the learner model widen the set — is exactly what makes
// topicSpread uncountable.
func parseLearnerDomains(md string) []store.Domain {
	var out []store.Domain
	seen := map[store.Domain]bool{}
	inTable := false
	for _, line := range strings.Split(md, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "## ") {
			// Only the domains table. A `|` in the Corrections section is a person
			// writing a table of their own, and #17 promises never to touch that
			// half — so reading it would let the human-owned region change what
			// the generated region means.
			inTable = strings.EqualFold(t, "## Domains they read in")
			continue
		}
		if !inTable || !strings.HasPrefix(t, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(t, "|"), "|")
		if len(cells) == 0 {
			continue
		}
		name := strings.TrimSpace(cells[0])
		if name == "" || name == "domain" || strings.HasPrefix(name, "---") {
			continue // the header and the rule beneath it
		}
		if d, ok := store.ParseDomain(name); ok && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}

// bandedWord is a deck word with the facts --harvest cached for it: the unit
// selection works over.
type bandedWord struct {
	Word  string
	Facts store.WordFacts
}

// THE ARCH-DRY QUESTION, ANSWERED (plan Task 5 Step 3).
//
// `#7` already selects options at review time (`play.PickOptions`), and the plan
// required establishing whether these are one function with two callers or two
// genuinely different rules — and writing the answer down either way.
//
// **They are two rules, and the reasons are structural rather than stylistic.**
//
//  1. **Different unit.** PickOptions selects GLOSSES — form 2.3 asks which
//     definition matches a word. This selects WORDS — a cloze asks which word
//     fills a blank. The heart of PickOptions is deduplicating on entry identity
//     because two deck keys can share one dictionary entry (`jalapeño` and
//     `jalapeno`), which cost three review rounds to get right. That problem does
//     not exist here: when the option IS the word, two options can only collide
//     by being the same word.
//  2. **Different constraint.** PickOptions picks one candidate per play.Axis —
//     domain, register, general — for VARIETY OF REASON, so a miss says what kind
//     of mistake it was. This filters by band and domain for LEVEL, and register
//     is not an axis here at all.
//  3. **Different time, and different failure.** Review-time selection runs on
//     the learner's clock against a live pool and must never block; this runs
//     offline against banded facts and may take as long as it likes. A bad choice
//     there costs one question; a bad choice here is cached forever.
//  4. **A structural barrier that is itself the evidence.** `play` imports
//     NOTHING by design (D5a), so it cannot see store.Band. Merging the two would
//     either drag the store's vocabulary across that seam or lift the band
//     comparison out of the selector — and the band comparison is the selector's
//     entire job.
//
// What they DO share is one line each ("never the answer") and the argument for
// walking a seeded permutation rather than the slice, which is recorded in
// PickOptions at length and cited here rather than restated.

// pickDistractors selects wrong answers from the deck. Pure.
//
// THE RULE, from the Spec: same domain (or general vocabulary), at the learner's
// band or one below, never the answer.
//
// **One band BELOW rather than above.** A distractor the learner does not know is
// unrejectable — they eliminate it by ignorance rather than by knowing it does
// not fit, and known-and-wrong is the whole point of a distractor.
//
// **Same domain, because mixing registers gives the answer away.** Three
// obviously-medical options beside one legal one is answerable without reading
// them. That also makes distractors MORE plausible, which is the point and also
// the risk — it is why the veto exists (see vetoTask).
//
// **Widening is TIERED and REPORTED, never silent.** A small deck will not have
// three same-domain words at the right level, and returning two options rather
// than four is a worse question than one drawn from general vocabulary. The tier
// reached is returned so the caller can say so and a test can see it, because a
// selector that silently falls back to "any word at all" is indistinguishable
// from one that is working.
// used counts how many items across the batch each word has already served as a
// distractor for. Optional — a nil map means no diversity pressure, which is what
// every unit test wants and what a single-item run gets.
func pickDistractors(answer string, target store.WordFacts, learner store.Band, pool []bandedWord, n int, seed uint64, servedSoFar map[string]int) ([]string, selectionTier) {
	// The learner's band when there is one; otherwise the ANSWER's own band, so
	// generic authoring still pitches distractors at the word rather than at the
	// whole scale. Recorded because it is a real decision: with no learner model
	// there is no "the learner's band" to be one below of.
	at := learner
	if at.Rank() < 0 {
		at = target.Band
	}
	below, hasBelow := at.Below()

	inBand := func(b store.Band) bool {
		if at.Rank() < 0 {
			return true // nothing to compare against; every band is admissible
		}
		return b == at || (hasBelow && b == below)
	}

	key := store.Key(answer)
	// Walked in a seeded permutation rather than in deck order, for the reason
	// play.PickOptions gives at length: a pool scanned in fixed order hands every
	// question the same first-matching candidates, and a learner answers the rest
	// by elimination without knowing a word.
	order := make([]int, len(pool))
	for i := range order {
		order[i] = i
	}
	shuffleInts(newHarvestPRNG(seed), order)

	// atOrBelow is the CEILING, and it is the constraint widening must never
	// relax before it has relaxed everything else. A word above the learner is
	// unrejectable — they eliminate it by ignorance rather than by knowing it
	// does not fit — so the tiers below widen the DOMAIN first and the level
	// downward second, and only reach upward as a last resort that says so.
	atOrBelow := func(b store.Band) bool {
		if at.Rank() < 0 {
			return true
		}
		return b.Rank() >= 0 && b.Rank() <= at.Rank()
	}

	tiers := []struct {
		tier selectionTier
		ok   func(bandedWord) bool
	}{
		{tierSameDomain, func(c bandedWord) bool {
			return c.Facts.Domain == target.Domain && inBand(c.Facts.Band)
		}},
		{tierGeneral, func(c bandedWord) bool {
			return c.Facts.Domain == store.DomainGeneral && inBand(c.Facts.Band)
		}},
		{tierAnyDomain, func(c bandedWord) bool { return atOrBelow(c.Facts.Band) }},
		{tierAboveBand, func(c bandedWord) bool { return true }},
	}

	// DIVERSITY PRESSURE, measured into existence by the first real batch: over
	// 20 words, `ephemeral` was a wrong answer in 8 items and the four A1 words
	// selected each other in all four of theirs. Every constraint above is
	// per-ITEM, so nothing stopped one eligible word from serving the whole
	// batch — and a learner who meets `ephemeral` as a wrong answer eight times
	// learns that it is never the answer, which is the opposite of the point.
	//
	// A SORT rather than a cap: a cap would refuse to fill an option set on a
	// small deck, and fewer options is a worse question than a repeated one. The
	// least-used eligible candidates simply come first, so repetition is what
	// happens when the deck has nothing else, not the default.
	//
	// Stable, and keyed on the shuffled position so equally-used candidates keep
	// their seeded order — the tie-break must not become a second ordering that
	// undoes the shuffle.
	if servedSoFar != nil {
		place := make(map[int]int, len(order))
		for pos, i := range order {
			place[i] = pos
		}
		sort.SliceStable(order, func(a, b int) bool {
			ua := servedSoFar[store.Key(pool[order[a]].Word)]
			ub := servedSoFar[store.Key(pool[order[b]].Word)]
			if ua != ub {
				return ua < ub
			}
			return place[order[a]] < place[order[b]]
		})
	}

	var out []string
	used := map[string]bool{key: true}
	reached := tierSameDomain
	for _, t := range tiers {
		if len(out) == n {
			break
		}
		for _, i := range order {
			if len(out) == n {
				break
			}
			c := pool[i]
			k := store.Key(c.Word)
			if used[k] || !c.Facts.Harvested() || !t.ok(c) {
				continue
			}
			used[k] = true
			out = append(out, c.Word)
			reached = t.tier
		}
	}
	return out, reached
}

// selectionTier is how far pickDistractors had to widen to fill the options.
//
// Reported rather than swallowed: tierAnyBand means the deck could not supply
// level-matched wrong answers, which is a fact about the DECK and not about the
// item — and it is the difference between "these distractors are pitched" and
// "these are whatever was lying around".
type selectionTier int

const (
	tierSameDomain selectionTier = iota
	tierGeneral
	tierAnyDomain
	tierAboveBand
)

func (t selectionTier) String() string {
	switch t {
	case tierSameDomain:
		return "same domain, at band"
	case tierGeneral:
		return "general vocabulary, at band"
	case tierAnyDomain:
		return "any domain, at or below band"
	default:
		return "above the learner's band — the deck could not supply level-matched options"
	}
}

const authorSystem = "You write one sentence that teaches a word by using it. " +
	"You never explain the word, and you never write a definition."

// renderAuthorPrompt builds the request for one item. Pure.
func renderAuthorPrompt(lang store.Lang, word, gloss string, facts store.WordFacts, learner learnerFacts) llm.Request {
	var b strings.Builder
	if lang == "" {
		lang = store.DefaultLang
	}

	fmt.Fprintf(&b, "## The word\n\n%s", word)
	if facts.Band != "" {
		fmt.Fprintf(&b, " (CEFR %s", facts.Band)
		if facts.Domain != "" && facts.Domain != store.DomainGeneral {
			fmt.Fprintf(&b, ", %s", facts.Domain)
		}
		b.WriteString(")")
	}
	fmt.Fprintf(&b, "\n\nLanguage: `%s`.\n\n", lang)
	if strings.TrimSpace(gloss) != "" {
		fmt.Fprintf(&b, "Its dictionary sense:\n\n%s\n\n", strings.TrimSpace(gloss))
	}

	if strings.TrimSpace(learner.Model) != "" {
		// The RAW model, not a summary of it. Its `directive` column is prose
		// written for a model to read — "Draw comparables from judicial prose" —
		// and it is the thing #17 exists to produce. Summarising it here would
		// throw that away and leave #17 delivering an enum.
		b.WriteString("## The learner\n\n")
		b.WriteString(strings.TrimSpace(learner.Model))
		b.WriteString("\n\n")
	}

	b.WriteString("## What to write\n\n")
	b.WriteString("ONE sentence using the word, which will be shown with the word blanked out.\n\n")
	// REQUIREMENTS, not preferences, and the reason is measured: asked for a
	// natural sentence the model drifts to the neutral and unnamed, and an
	// unnamed subject gives the learner no referent to attach the word to.
	b.WriteString("Three requirements:\n\n")
	b.WriteString("1. **The sentence must POINT AT the word without defining it.** A reader who knows " +
		"the word must find it the obvious fit; a reader who does not must be left guessing. " +
		"\"His ___ behaviour was noted by all\" is too loose — almost any adjective fits.\n")
	// THE APPOSITIVE BAN, and it is stated as its own requirement because the
	// first batch showed requirement 1 CAUSES this: the cheapest way to make a
	// sentence point at a word is to define the word in it. Half of 20 items came
	// back as "the alewife, the small silver herring Alosa pseudoharengus" — a
	// reading test rather than a vocabulary test, since the learner need only read
	// the gloss beside the blank.
	//
	// Shown rather than described: "do not define it" is what the system prompt
	// already said, and the model honoured it by writing an appositive instead.
	b.WriteString("2. **Never gloss the word.** The sentence must not contain a definition of it — " +
		"not as an appositive, not as a relative clause, not as a contrast. These are all WRONG:\n\n")
	b.WriteString("   - \"...the run of ___, the small silver herring that spawns upstream.\"\n")
	b.WriteString("   - \"...classified the landform as a ___, since it is too broad to be a butte.\"\n")
	b.WriteString("   - \"...the ship's ___, the massive timber spine running the length of her hull.\"\n\n")
	b.WriteString("   Write instead a sentence in which the word simply DOES ITS WORK, the way a " +
		"newspaper would use it in front of readers assumed to know it:\n\n")
	b.WriteString("   - \"Biologists at the Holyoke Dam counted 400,000 ___ climbing the fish lift this spring.\"\n")
	b.WriteString("   - \"The road climbs 300 metres from the valley floor to the ___ above Monument Valley.\"\n\n")
	b.WriteString("3. **Name real people, places or institutions.** Not \"a manager\" or \"the company\" — " +
		"a named subject the reader can picture. This is what the word attaches to in memory.\n\n")
	b.WriteString("Write the sentence with the word itself present, spelled exactly as given. " +
		"Do not blank it out; do not quote it; do not explain it afterwards.\n\n")

	schema, err := llm.SchemaFor[authoredStem]()
	if err != nil {
		schema = nil
	}
	return llm.Request{Task: authorTaskName, System: authorSystem, Prompt: b.String(), Schema: schema}
}

// newHarvestPRNG and shuffleInts are the authoring-side deterministic shuffle.
//
// Not play's: that package imports NOTHING by design (D5a) and hand-rolls its
// own, and reaching into it would either export an internal or drag the store's
// vocabulary across that seam. Same algorithm, stated once here — see the
// recorded ARCH-DRY finding on pickDistractors vs play.PickOptions.
type harvestPRNG struct{ s uint64 }

func newHarvestPRNG(seed uint64) *harvestPRNG { return &harvestPRNG{s: seed*6364136223846793005 + 1} }

func (p *harvestPRNG) next() uint64 {
	p.s ^= p.s << 13
	p.s ^= p.s >> 7
	p.s ^= p.s << 17
	return p.s
}

func shuffleInts(p *harvestPRNG, xs []int) {
	for i := len(xs) - 1; i > 0; i-- {
		j := int(p.next() % uint64(i+1))
		xs[i], xs[j] = xs[j], xs[i]
	}
}

// sortedBanded gives a run a stable candidate order before the seeded shuffle,
// so two runs over the same deck select the same options.
func sortedBanded(in []bandedWord) []bandedWord {
	out := append([]bandedWord(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Word < out[j].Word })
	return out
}

// stemUsesTheWord reports whether a stem actually contains its answer. Pure, and
// checked BEFORE any judge is paid.
//
// The second checkpoint batch produced "The Hopi village of Walpi has stood atop
// the narrow ___ of First Mesa" for the word `mesa`: the model blanked the word
// itself, against an explicit instruction not to, and both judges passed the
// item because neither was asked. An item whose stem does not contain its answer
// cannot be rendered as a question at all.
//
// Deterministic and free, which is why it runs first: a model call to check
// whether a string contains a substring would be the same mistake as asking one
// to derive a domain the dictionary printed.
func stemUsesTheWord(stem, word string) bool {
	lower, target := strings.ToLower(stem), strings.ToLower(strings.TrimSpace(word))
	if target == "" {
		return false
	}
	// ALREADY BLANKED. The shipped case was subtler than "the word is missing":
	// "...has stood atop the narrow ___ of First Mesa" DOES contain `mesa`, in the
	// place name, so a containment check alone passes it. The defect is the blank
	// the model inserted against an explicit instruction — #12 owns blanking, and
	// a stem that arrives pre-blanked would be blanked twice or not at all.
	if strings.Contains(stem, "___") {
		return false
	}
	i := strings.Index(lower, target)
	if i < 0 {
		return false
	}
	// A WHOLE WORD, not a substring: `set` must not be satisfied by `sunset`, and
	// `run` not by `brunch`. Inflections are allowed to follow (`ran` will not
	// match, but `runs` and `keels` will), because a stem using the word
	// naturally often inflects it and rejecting that would push the model back
	// toward the stilted constructions the gloss rule already fought.
	if i > 0 && isWordByte(lower[i-1]) {
		return false
	}
	return true
}

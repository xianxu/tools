package main

import (
	"strings"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
)

// Assembling form 2.3's option material from the learner's own deck.
//
// All of it happens HERE, in main, because it reads the dictionary: play
// receives finished Candidates and never parses prose (D5, D5a). The division is
// the same one the board uses — finished text in, no rendering inside.

// poolCap bounds the dictionary work one sitting costs.
//
// ARCH-CONSTRAINTS: the new cost is one local DCSCopyTextDefinition per POOL
// word, on top of the one per DUE word --play already pays. Both are local
// framework calls with no network. Forty keeps the worst case in the same order
// as today's sitting rather than growing with a deck that may reach thousands —
// and a pool of forty already offers far more distractors than the three any one
// question needs, so raising it would buy variety nobody can perceive.
const poolCap = 40

// buildPool looks up a SAMPLE of the deck and returns every candidate distractor
// it can supply.
//
// Sampled under the seed rather than truncated, so a large deck does not always
// contribute its alphabetically-first forty words — the same forty would make
// every sitting draw from one corner of the deck forever.
//
// A word the dictionary can no longer find is skipped in silence here, unlike in
// todaysQuestions where the same failure is reported: a DUE word that will not
// look up costs the learner a review and they should know, while a pool word is
// one of forty interchangeable distractor sources and saying so would be noise.
func buildPool(d deps, deck []store.Word, seed uint64) []play.Candidate {
	keys := make([]string, 0, len(deck))
	for _, w := range deck {
		if k := store.Key(w.Text); k != "" {
			keys = append(keys, k)
		}
	}
	play.SampleStrings(keys, poolCap, seed)
	if len(keys) > poolCap {
		keys = keys[:poolCap]
	}

	var pool []play.Candidate
	for _, k := range keys {
		text, err := d.dict.Lookup(k)
		if err != nil {
			continue
		}
		pool = append(pool, optionCandidates(k, ParseEntry(text))...)
	}
	return pool
}

// optionCandidates is D4a: WHICH sense of an entry becomes an option.
//
// At most three per word — one per axis — because a question may use a word once
// (two options from one headword would be two answers with a single source).
// Selection then picks whichever of them the slot it is filling needs.
//
//   - each axis takes the FIRST usable sense carrying it, in document order
//     across all blocks. For general that means the first UNLABELLED usable
//     sense — which is usually the first sense of the first block, since NOAD
//     orders by centrality, but is a later one whenever the earlier senses are
//     all labelled. Measured: `defenestrate`'s general candidate is not its
//     first sense, because that one is `rare`.
//
// The plan's D4a said "the first sense of the first block" flatly. That was
// written before the labelled/unlabelled split existed and is wrong for any
// entry whose opening sense carries a label; the code is right and D4a is
// corrected in the plan's revisions. `#12` reuses this rule, so the precise
// statement matters beyond this form.
//
// "Usable" is doing real work: the corpus contains senses whose whole gloss is
// `[with object]` or `(plural men /men/)` or `another term for menhaden`, and
// any of those as an option would make the form look broken. See readGloss.
func optionCandidates(word string, e Entry) []play.Candidate {
	// GUARDED HERE, not in the caller. Round 3 put this check in choiceFor and
	// caught only the TARGET; buildPool kept feeding `bargain`'s glosses into
	// every question labelled `bargainer`, so a learner could pick a distractor
	// whose word and meaning belonged to different entries. Both producers of a
	// (word, gloss) pair now guard themselves, which is what makes the rule
	// "a gloss is only ever attributed to the word its entry defines" true of
	// every path rather than of the one path a finding named.
	if !entryDefines(word, e) {
		return nil
	}
	var out []play.Candidate
	seen := map[play.Axis]bool{}
	for _, b := range e.Blocks {
		for _, s := range b.Senses {
			f := readGloss(s.Gloss)
			if !f.Usable || seen[f.Axis] {
				continue
			}
			seen[f.Axis] = true
			out = append(out, play.Candidate{Word: word, Source: entryIdentity(e), Gloss: f.Text, Axis: f.Axis})
		}
	}
	return out
}

// targetCandidate is the sense the question is ASKING about: the first usable
// sense in document order — the first block's, whenever that block has one at
// all. Not ok when the entry offers no definition anywhere: an entry that is
// nothing but cross-references (`bases` — "plural form of base1") cannot be the
// answer to a recognition question, so the caller triages the word on a board.
//
// Unlike optionCandidates this takes the first usable sense of ANY axis, not
// the first unlabelled one. The target is what the learner looked up; a `rare`
// or `Law` sense is still what NOAD leads with, and substituting a later
// unlabelled one would ask about a meaning the entry does not put first.
func targetCandidate(word string, e Entry) (play.Candidate, bool) {
	if !entryDefines(word, e) {
		return play.Candidate{}, false
	}
	for _, b := range e.Blocks {
		for _, s := range b.Senses {
			if f := readGloss(s.Gloss); f.Usable {
				return play.Candidate{Word: word, Source: entryIdentity(e), Gloss: f.Text, Axis: play.AxisGeneral}, true
			}
		}
	}
	return play.Candidate{}, false
}

// entryDefines reports whether the entry actually defines the prompted word.
//
// NOAD REDIRECTS DERIVED FORMS to their base headword: looking up `bargainer`
// returns the `bargain` entry, whose first gloss is "an agreement between two or
// more parties…". Form 2.1 survived that because it shows the whole rendered
// entry, DERIVATIVES line included, and the learner reads it as the answer to
// "did you know this word". Form 2.3 cannot: it asserts that ONE gloss IS the
// meaning of the word on screen, marks it Correct, and promotes the word in the
// schedule on the strength of it. So the entry has to be checked, and a redirect
// is triaged on a board instead — the same route `bases` takes.
//
// `Headword()` alone is NOT the check, which is why this walks the token run:
// the head is built from `fields[0]` (parse.go:436), so it returns "hot" for
// `hot dog` and "a" for `a priori`, and gating on it would send every multi-word
// headword to the fallback. Accumulating `HeadWord` plus the `HeadOther` tokens
// that follow it reconstructs "hot dog" and "a priori", and stops at the
// syllable token — which is exactly where `bargainer` fails to match.
func entryDefines(word string, e Entry) bool {
	want := store.Key(word)
	if want == "" {
		return false
	}
	// Compared at every PREFIX rather than at the end: the run keeps growing
	// past the headword into NOAD's own repetitions ("a priori a pri·o·ri"), so
	// only a prefix can be expected to match.
	var run string
	for _, tok := range headRun(e) {
		if run != "" {
			run += " "
		}
		run += tok
		if got := store.Key(run); got == want || differsOnlyByDiacritics(got, want) {
			return true
		}
	}
	return false
}

// headRun is the entry's leading word tokens — the ONE place that decides what
// the head of an entry is.
//
// `HeadWord` plus the `HeadOther` tokens that follow it, stopping at the first
// token of any other kind (the syllable form, the part of speech). That
// reconstructs "hot dog" and "a priori" from a head that `parse.go` builds out
// of `fields[0]`, and it stops exactly where `bargainer` diverges from the
// `bargain` entry it redirects to.
func headRun(e Entry) []string {
	var out []string
	for _, h := range e.Head {
		if h.Kind != HeadWord && h.Kind != HeadOther {
			break
		}
		out = append(out, h.Text)
	}
	return out
}

// entryIdentity is which ENTRY this is, for deduplication.
//
// Derived from headRun, NOT from Headword(): `Headword()` is `fields[0]`
// (parse.go:436), which is "hot" for `hot dog` and "a" for `a priori`, so two
// genuinely different entries can share it. Using it as a dedup key silently
// drops one of their options — a learner with both `hot dog` and `hot` in the
// deck loses a distractor, and on a small deck loses the form entirely.
//
// The direction of that failure is over-dedup, never a wrong answer, which is
// why it was Minor. It is fixed here rather than lived with because the file had
// grown TWO answers to "which entry is this" — this one and Headword() — and
// entryDefines already walked the right one.
func entryIdentity(e Entry) string {
	return store.Key(strings.Join(headRun(e), " "))
}

// fallbackReasons is every reason choiceFor refuses, as prose a doc must carry.
//
// DECLARED rather than described, because the same enumeration was being
// hand-maintained in three places — this function, the README and the atlas —
// and had already drifted: both docs listed two of the reasons and the atlas
// listed one, while the code branched on three. A sentence that enumerates
// ("X, or Y") is a closed claim about the code, so it derives from here or it
// goes stale. Same move doc_sync_test.go already made for the prompt lines.
var fallbackReasons = []string{
	"the deck has no other word to draw on",
	"the entry is nothing but cross-references",
	"the entry defines a different word",
}

// choiceFor builds form 2.3 for one word, or reports that it cannot.
//
// Cannot happens for three ordinary reasons, none an error: the entry does not
// define the prompted word (a NOAD derivative redirect — see entryDefines), the
// entry offers no usable definition at all (`bases`, every sense a
// cross-reference), or the deck has not yet grown enough distractors (D9 — a
// learner three lookups in). The caller TRIAGES the word on a board (#42), which
// is invisible to the learner and keeps the sitting the length the schedule asked
// for.
func choiceFor(word, rendered string, e Entry, pool []play.Candidate, seed uint64) *play.Choice {
	opts := optionsFor(word, e, pool, seed)
	if opts == nil {
		return nil
	}
	return play.NewChoice(word, rendered, opts)
}

// optionsFor is whether this entry can be form 2.3 AT ALL, and with which
// options. Nil means it cannot.
//
// SPLIT FROM choiceFor so SELECTION can ask without paying a Render (#42). The
// form is chosen after the lookup now — "can this word have distractors" is only
// knowable once the entry is parsed — and a word that turns out to be triaged
// would otherwise have been wrapped, coloured and click-mapped for a form it
// never takes.
//
// It is also the whole of the DECISION, so the three fallback reasons all live on
// this side of the split and `fallbackReasons` still describes one function.
func optionsFor(word string, e Entry, pool []play.Candidate, seed uint64) []play.Option {
	// No entryDefines call here: targetCandidate guards itself, and so does
	// optionCandidates. A check at this level is what let the distractor path
	// through unguarded once already.
	target, ok := targetCandidate(word, e)
	if !ok {
		return nil
	}
	// D3a, applied per question rather than to the pool: whether a candidate is
	// a near-synonym is a fact about THIS target, so the same pool word may be
	// excluded here and perfectly good for the next word in the sitting.
	usable := make([]play.Candidate, 0, len(pool))
	for _, c := range pool {
		if c.Word == target.Word || crossReferenced(target.Word, target.Gloss, c.Word, c.Gloss) {
			continue
		}
		usable = append(usable, c)
	}
	opts := play.PickOptions(target, usable, seed)
	if len(opts) < 2 {
		return nil
	}
	return opts
}

// seedFor is a question's seed: FNV-1a over the parts.
//
// Written out rather than taken from hash/fnv, and the reason is narrower than
// the first version claimed. It said "reproducible from a log indefinitely",
// which is not true and cannot be: the option set also depends on the POOL, and
// the pool is the deck at that moment — its membership and its `LastSeen`
// ordering — none of which the event log records. A recorded question is not
// re-derivable from the log alone whatever hash is used.
//
// What the seed does guarantee is that the same deck on the same day yields the
// same sitting: a mid-sitting restart re-asks the same questions rather than
// reshuffling, and tests are stable across runs and machines. Writing the hash
// out keeps that sequence this repo's to pin (TestSeedForIsPinned) instead of a
// property of a stdlib version — the same standard pick.go's PRNG is held to.
//
// The DAY is one of the parts, so a word asked again next week gets a different
// option set. Without it the learner would meet the same options in the same
// order every time and could learn the position instead of the meaning.
func seedFor(parts ...string) uint64 {
	const offset64, prime64 = 14695981039346656037, 1099511628211
	h := uint64(offset64)
	for _, p := range parts {
		for i := 0; i < len(p); i++ {
			h ^= uint64(p[i])
			h *= prime64
		}
		h ^= '\n'
		h *= prime64
	}
	return h
}

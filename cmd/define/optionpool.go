package main

import (
	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
)

// Assembling form 2.3's option material from the learner's own deck.
//
// All of it happens HERE, in main, because it reads the dictionary: play
// receives finished Candidates and never parses prose (D5, D5a). The division is
// the same one Recall uses — already-rendered text in, no rendering inside.

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
//   - general: the FIRST usable sense of the first block. NOAD orders senses by
//     centrality, so this is the meaning the learner most likely met, and it is
//     the sense Recall's reveal leads with.
//   - domain / register: the first usable sense whose gloss carries that label.
//
// "Usable" is doing real work: the corpus contains senses whose whole gloss is
// `[with object]` or `(plural men /men/)` or `another term for menhaden`, and
// any of those as an option would make the form look broken. See readGloss.
func optionCandidates(word string, e Entry) []play.Candidate {
	var out []play.Candidate
	seen := map[play.Axis]bool{}
	for _, b := range e.Blocks {
		for _, s := range b.Senses {
			f := readGloss(s.Gloss)
			if !f.Usable || seen[f.Axis] {
				continue
			}
			seen[f.Axis] = true
			out = append(out, play.Candidate{Word: word, Gloss: f.Text, Axis: f.Axis})
		}
	}
	return out
}

// targetCandidate is the sense the question is ASKING about: the first usable
// one, first block. Not ok when the entry offers no definition at all — an
// entry that is nothing but cross-references cannot be the answer to a
// recognition question, so the caller falls back to form 2.1.
func targetCandidate(word string, e Entry) (play.Candidate, bool) {
	for _, b := range e.Blocks {
		for _, s := range b.Senses {
			if f := readGloss(s.Gloss); f.Usable {
				return play.Candidate{Word: word, Gloss: f.Text, Axis: play.AxisGeneral}, true
			}
		}
	}
	return play.Candidate{}, false
}

// choiceFor builds form 2.3 for one word, or reports that it cannot.
//
// Cannot happens for two ordinary reasons, neither an error: the entry offers no
// usable definition, or the deck has not yet grown enough distractors (D9 — a
// learner three lookups in). The caller falls back to form 2.1, which is
// invisible to the learner and keeps the sitting the length the schedule asked
// for.
func choiceFor(word, rendered string, e Entry, pool []play.Candidate, seed uint64) *play.Choice {
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
	return play.NewChoice(word, rendered, opts)
}

// seedFor is a question's seed: FNV-1a over the parts.
//
// Written out rather than taken from hash/fnv because Done-when 3 wants a
// question reproducible from a log indefinitely, and a hash defined here is
// pinned by this repo's tests — the same argument the shuffle makes about
// math/rand, and the reason the two live at the same standard.
//
// The DAY is one of the parts, so a word asked again next week gets a different
// option set. Without it the learner would meet the same four definitions in the
// same order every time and could learn the position instead of the meaning.
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

package main

import (
	"strings"
	"unicode"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
)

// Blank is what replaces the answer in a stem.
//
// Three underscores rather than a variable-width run: a blank whose width
// tracked the answer would leak its LENGTH, which on a four-option question is
// often the whole answer.
const Blank = "___"

// blankStem hides the answer in a stem, everywhere it occurs.
//
// THE LEAK IS THE FAILURE MODE, and it is silent: a question that gives away its
// answer still renders perfectly and still grades. So every rule here is about a
// way the answer survives.
//
//   - EVERY occurrence, not the first. A stem using the word twice hands it over
//     in full, and this is the row most easily missed because the common case
//     has one.
//   - The whole WORD RUN, not the answer's own length. #10's author prompt
//     permits inflection to follow, so `keels` for `keel` must not leave `___s`.
//     wordRunEnd uses highlight.go's isWordRune, which counts apostrophes and
//     hyphens as inside a word — so a possessive goes too.
//   - Never a SUBSTRING. `set` must not blank inside `sunset`; wordIndexIn
//     already refuses that and is mutation-tested for it.
//   - Case-insensitively, because a sentence-initial answer is capitalised.
//
// Returns the stem unchanged when the answer is not in it. That is not silent
// failure — usableItem refuses such an item before a question is ever built, and
// this stays total so the caller has one thing to check rather than two.
func blankStem(stem, answer string) string {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return stem
	}
	var b strings.Builder
	for rest, from := stem, 0; ; {
		i, n := wordIndexIn(rest[from:], answer)
		if i < 0 {
			b.WriteString(rest[from:])
			break
		}
		at := from + i
		end := wordRunEnd(rest, at)
		// THE LOOP MUST ADVANCE, whatever the match looked like.
		//
		// Found by the fuzz in eleven seconds, and it HUNG rather than failing:
		// invalid UTF-8 decodes to RuneError, which matches itself, and
		// RuneError is not a word rune — so the run ended where it began, `from`
		// never moved, and the loop spun forever. workshop/lessons.md carries
		// this exact rule from one issue ago: "bound every search whose
		// termination depends on the property under test". Here the property was
		// "a match has a word run", which garbage input does not have.
		//
		// The match length is the floor and is at least one byte, because the
		// answer is non-empty by the time we are here.
		if end <= at {
			end = at + n
		}
		b.WriteString(rest[from:at])
		b.WriteString(Blank)
		from = end
	}
	return b.String()
}

// usableItem reports whether a stored item can be rendered as a question.
//
// THE CLASS, not one instance. sanitiseItem neutralises on read and does NOT
// re-validate, the README documents items/ as inspectable, and runAuthoring can
// legitimately write an item whose veto left only one distractor. So every way a
// file on disk can fail to be a question is enumerated here:
//
//   - no answer, or a stem that does not contain it — nothing to blank, so the
//     question would render with no hole in it;
//   - no distractors — a one-option question is not a question;
//   - a distractor EQUAL to the answer, which is the nastiest: it renders as two
//     correct options with one marked wrong, and grading it marks the learner
//     wrong for being right. That is precisely what #10's veto exists to
//     prevent, arriving by the one path the veto never sees.
//
// Case-insensitive on the last, because `Sycophantic` and `sycophantic` are one
// word to a learner reading four options.
func usableItem(it store.Item) bool {
	// AN ANSWER MUST BE A WORD. Found by the fuzz: an answer of "_" is matchable
	// (nothing before it, so the boundary check passes) and blanking it produces
	// "___" — a question whose blank IS its answer. A deck entry with no letter
	// and no digit is not a word, and an item carrying one is not a question.
	if !hasLetterOrDigit(it.Answer) {
		return false
	}
	if strings.TrimSpace(it.Answer) == "" || !stemUsesTheWord(it.Stem, it.Answer) {
		return false
	}
	if len(it.Distractors) == 0 {
		return false
	}
	for _, d := range it.Distractors {
		if strings.EqualFold(strings.TrimSpace(d), strings.TrimSpace(it.Answer)) {
			return false
		}
	}
	return true
}

// clozeFor turns a word's stored items into one question, or reports that it
// cannot.
//
// THREE DECISIONS, all of which the "deterministic under a fixed seed" row
// depends on and none of which is arbitrary:
//
//   - WHICH ITEM: the newest FormCloze one. Items() returns newest-first (a
//     Store promise held by storetest) and prune keeps the newest, so "the first
//     of the right form" is already the newest and needs no second rule.
//   - THE SEED: the caller's, which is seedFor(key, day) — the same word gets
//     the same question all day and a different one tomorrow, exactly as #7
//     seeds its options.
//   - THE ORDER: shuffled. Without it the answer sits at position 1 in every
//     question and a learner answers the whole sitting by pressing 1 — a
//     question that still LOOKS right, which is why #7 pinned the same property.
//
// definition is the rendered entry for the reveal; empty is acceptable and the
// reveal simply carries less.
func clozeFor(key string, items []store.Item, definition string, seed uint64) play.Question {
	for _, it := range items {
		if it.Form != store.FormCloze || !usableItem(it) {
			continue
		}
		opts := make([]play.Option, 0, len(it.Distractors)+1)
		opts = append(opts, play.Option{Word: it.Answer, Correct: true})
		for _, d := range it.Distractors {
			opts = append(opts, play.Option{Word: d})
		}
		order := make([]int, len(opts))
		for i := range order {
			order[i] = i
		}
		play.ShuffleInts(seed, order)
		shuffled := make([]play.Option, len(opts))
		for i, j := range order {
			shuffled[i] = opts[j]
		}
		return play.NewCloze(key, blankStem(it.Stem, it.Answer), it.Stem, definition, shuffled)
	}
	return nil
}

// hasLetterOrDigit reports whether a string is a word at all.
//
// NOT isWordRune, and the distinction is the point. isWordRune counts hyphens
// and apostrophes as inside a word, which is right for TOKENISING — `hot-dog` is
// one token and splitting it would make a hyphenated deck entry unmatchable. It
// is wrong for "is this a word": `---` is three joiners and nothing to join.
// The fuzz found the general case as `_`; `---` is the same hole one predicate
// over, and it passes an isWordRune test.
//
// A word needs something a word is MADE of. Hyphens and apostrophes only join.
func hasLetterOrDigit(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

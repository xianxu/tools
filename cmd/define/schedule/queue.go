package schedule

import (
	"sort"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// Queue is today's words, in the order they should be offered, as normalised
// store.Keys — not the deck's original spelling. A caller rendering them to the
// learner should look the word up in the deck for display text; a caller
// recording an event should use the key, which is what Fold reads back.
//
// TWO TIERS, and the Done-when forces it: "never starves an overdue word in
// favour of a fresh one". Words with review history that are due come FIRST,
// most overdue first; words never reviewed follow, most looked-up first.
//
// Ranking everything on one "how long since we saw it" axis looks simpler and is
// wrong: a word first seen months ago and never reviewed has a larger age than a
// word reviewed last week and three days overdue, so the newcomer would go first
// and the word actually being learned would wait. That is the exact starvation
// the row forbids, and it is why the tiers are separate rather than a single
// sort key with a clever weight.
//
// The DECK is the roster and the log is the history. A word with progress but no
// deck entry is not queued: `--forget` deliberately keeps a word's events after
// removing it, and resurrecting it here would make forgetting not work.
func Queue(deck []store.Word, prog map[string]Progress, now time.Time, budget int) []string {
	if budget <= 0 {
		// "No budget" is not "unlimited". The opposite reading is a way to
		// accidentally sit down to four hundred words.
		return nil
	}

	type candidate struct {
		key     string
		overdue int // local calendar days past due; fresh words are not ranked by this
		lookups int
	}
	var reviewed, fresh []candidate
	// Deduped by KEY, because store.Key collapses "Define" and "define" into one
	// word while the deck can legitimately hold both — a hand-edited file, or one
	// written before Key existed. Without this the queue offered the same word
	// twice and spent two budget slots on it.
	//
	// First occurrence wins: Lookups may differ between the two rows and the
	// deck is ordered by LastSeen, so the more recent row is the better record.
	seen := make(map[string]bool, len(deck))

	for _, w := range deck {
		key := store.Key(w.Text)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		p, seen := prog[key]
		// Mastered words are NOT excluded, and the first version of this excluded
		// them. Two things were wrong with that. It contradicted progress.go's own
		// comment — "#6's --play decides what to stop offering" — by deciding here
		// instead. And it made mastery ABSORBING: a word never offered can never be
		// answered wrong, so it could never be demoted, and the learner's mastered
		// count could only ever grow while their actual recall decayed.
		//
		// Nothing is needed to make mastered words rare: they sit at the 90-day
		// interval, which is the ladder doing its job. Mastery is a status for
		// reporting (#8) and for presentation (#6), not a removal.
		if !Due(p, now) {
			continue
		}
		c := candidate{key: key, lookups: w.Lookups}
		if !seen || p.LastReviewed.IsZero() {
			fresh = append(fresh, c)
			continue
		}
		c.overdue = store.DaysBetween(p.LastReviewed, now) - IntervalDays(p.Box)
		reviewed = append(reviewed, c)
	}

	// Deterministic on every axis: a queue that reorders between runs is
	// untestable and looks broken to the learner.
	// The tie-break tail — most looked-up, then alphabetical — is shared rather
	// than written twice. Two copies would be two places to keep in agreement,
	// and a queue whose two halves broke ties differently would be a puzzle to
	// diagnose from the outside.
	byLookupsThenKey := func(a, b candidate) bool {
		if a.lookups != b.lookups {
			return a.lookups > b.lookups
		}
		return a.key < b.key
	}
	sort.Slice(reviewed, func(i, j int) bool {
		if reviewed[i].overdue != reviewed[j].overdue {
			return reviewed[i].overdue > reviewed[j].overdue
		}
		return byLookupsThenKey(reviewed[i], reviewed[j])
	})
	sort.Slice(fresh, func(i, j int) bool {
		return byLookupsThenKey(fresh[i], fresh[j])
	})

	// Capacity is bounded by what can actually be returned, not by the budget:
	// make([]string, 0, 1<<62) panics, and a budget is caller input.
	capacity := budget
	if n := len(reviewed) + len(fresh); n < capacity {
		capacity = n
	}
	out := make([]string, 0, capacity)
	for _, c := range append(reviewed, fresh...) {
		if len(out) == budget {
			break
		}
		out = append(out, c.key)
	}
	return out
}

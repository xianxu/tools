package schedule

import (
	"sort"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// Queue is today's words, in the order they should be offered.
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

	for _, w := range deck {
		key := store.Key(w.Text)
		if key == "" {
			continue
		}
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
	sort.Slice(reviewed, func(i, j int) bool {
		if reviewed[i].overdue != reviewed[j].overdue {
			return reviewed[i].overdue > reviewed[j].overdue
		}
		if reviewed[i].lookups != reviewed[j].lookups {
			return reviewed[i].lookups > reviewed[j].lookups
		}
		return reviewed[i].key < reviewed[j].key
	})
	sort.Slice(fresh, func(i, j int) bool {
		if fresh[i].lookups != fresh[j].lookups {
			return fresh[i].lookups > fresh[j].lookups
		}
		return fresh[i].key < fresh[j].key
	})

	out := make([]string, 0, budget)
	for _, c := range append(reviewed, fresh...) {
		if len(out) == budget {
			break
		}
		out = append(out, c.key)
	}
	return out
}

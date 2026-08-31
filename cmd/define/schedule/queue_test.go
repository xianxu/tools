package schedule

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

func word(text string, lookups int, firstSeen int) store.Word {
	return store.Word{Text: text, Lookups: lookups, FirstSeen: at(firstSeen), LastSeen: at(firstSeen)}
}

func keys(q []string) string { return strings.Join(q, ",") }

// The Done-when's starvation row, and the fixture has to contain BOTH kinds or
// it cannot tell the two orderings apart.
//
// `fresh` was first seen on day 0 and never reviewed. `overdue` was reviewed on
// day 40 and is at box 1 (3 days), so on day 60 it is 17 days overdue. Ranking
// everything on one "how long since we saw it" axis would put `fresh` first —
// its age is 60 days — which is exactly the starvation the row forbids.
func TestOverdueReviewedWordBeatsAFreshOne(t *testing.T) {
	deck := []store.Word{word("fresh", 9, 0), word("overdue", 1, 40)}
	prog := Fold([]store.ReviewEvent{reviewed("overdue", true, at(40))})

	got := Queue(deck, prog, at(60), 10)

	if keys(got) != "overdue,fresh" {
		t.Errorf("queue = %v, want overdue first — a word already being learned must not be starved by a new one", got)
	}
}

func TestQueueRespectsTheBudget(t *testing.T) {
	deck := []store.Word{word("a", 1, 0), word("b", 1, 0), word("c", 1, 0), word("d", 1, 0)}

	got := Queue(deck, nil, at(1), 2)

	if len(got) != 2 {
		t.Errorf("queue has %d words with a budget of 2: %v", len(got), got)
	}
}

// "No budget" is not "unlimited". The opposite reading is a way to accidentally
// sit down to 400 words.
func TestZeroBudgetReturnsNothing(t *testing.T) {
	deck := []store.Word{word("a", 1, 0), word("b", 1, 0)}

	for _, budget := range []int{0, -1} {
		if got := Queue(deck, nil, at(1), budget); len(got) != 0 {
			t.Errorf("budget %d returned %v, want nothing", budget, got)
		}
	}
}

// Within the overdue tier, most overdue first. Both are due; only the ordering
// distinguishes them.
func TestOverdueTierOrdersByHowOverdue(t *testing.T) {
	deck := []store.Word{word("recent", 5, 0), word("ancient", 5, 0)}
	prog := Fold([]store.ReviewEvent{
		reviewed("recent", true, at(50)),
		reviewed("ancient", true, at(10)),
	})

	got := Queue(deck, prog, at(60), 10)

	if keys(got) != "ancient,recent" {
		t.Errorf("queue = %v, want the more overdue word first", got)
	}
}

// Within the fresh tier, most looked-up first: how often you reached for a word
// is the best signal available for a word never reviewed.
func TestFreshTierOrdersByLookups(t *testing.T) {
	deck := []store.Word{word("once", 1, 0), word("often", 12, 0)}

	got := Queue(deck, nil, at(1), 10)

	if keys(got) != "often,once" {
		t.Errorf("queue = %v, want the more looked-up word first", got)
	}
}

// A queue that reorders between runs is untestable and looks broken.
func TestQueueIsDeterministicOnTies(t *testing.T) {
	deck := []store.Word{word("beta", 3, 0), word("alpha", 3, 0), word("gamma", 3, 0)}

	first := keys(Queue(deck, nil, at(1), 10))
	for i := 0; i < 20; i++ {
		if got := keys(Queue(deck, nil, at(1), 10)); got != first {
			t.Fatalf("run %d gave %q, first run gave %q", i, got, first)
		}
	}
	if first != "alpha,beta,gamma" {
		t.Errorf("tie-break order = %q, want alphabetical by key", first)
	}
}

// A word not yet due is not in today's queue — that is the whole point of a
// schedule.
func TestNotDueWordsAreExcluded(t *testing.T) {
	deck := []store.Word{word("waiting", 1, 0), word("due", 1, 0)}
	// Three corrects reach box 3, whose interval is 4 days — long enough that
	// "reviewed yesterday" is unambiguously not due.
	prog := Fold([]store.ReviewEvent{
		reviewed("waiting", true, at(57)), reviewed("waiting", true, at(58)), reviewed("waiting", true, at(59)),
		reviewed("due", true, at(48)), reviewed("due", true, at(49)), reviewed("due", true, at(50)),
	})

	got := Queue(deck, prog, at(60), 10)

	if keys(got) != "due" {
		t.Errorf("queue = %v, want only the due word", got)
	}
}

// The DECK is the roster and the log is the history. --forget deliberately keeps
// a word's events after removing it from the deck, so progress without a deck
// entry must not resurrect it.
func TestForgottenWordDoesNotReturn(t *testing.T) {
	prog := Fold([]store.ReviewEvent{reviewed("forgotten", true, at(1))})

	got := Queue(nil, prog, at(60), 10)

	if len(got) != 0 {
		t.Errorf("queue = %v, want nothing — the word is not in the deck", got)
	}
}

// Mastery is a STATUS, not an exit from the rotation.
//
// The first version of Queue excluded mastered words and this test asserted that.
// Two things were wrong with it. It contradicted progress.go's own comment —
// "#6's --play decides what to stop offering" — by deciding in the queue instead.
// And it made mastery ABSORBING: a word never offered can never be answered
// wrong, so it could never be demoted, and the learner's mastered count could
// only grow while their actual recall decayed.
//
// Nothing is needed to make mastered words rare: they sit at box 9's 68-day
// interval and climb from there, which is the ladder doing its job.
func TestMasteredWordsStillComeRoundAtTheLongInterval(t *testing.T) {
	var events []store.ReviewEvent
	for i := 0; i < MasteredBox; i++ {
		events = append(events, reviewed("known", true, at(i)))
	}
	prog := Fold(events)
	if !Mastered(prog[store.Key("known")]) {
		t.Fatal("fixture is wrong: the word is not mastered, so this test asserts nothing")
	}
	last := prog[store.Key("known")].LastReviewed

	iv := IntervalDays(prog[store.Key("known")].Box)
	// Not due the day after mastery — the long interval is what keeps it rare.
	if got := Queue([]store.Word{word("known", 3, 0)}, prog, last.AddDate(0, 0, 1), 10); len(got) != 0 {
		t.Errorf("queue = %v one day after mastery, want nothing", got)
	}
	// But it DOES come back, or mastery would be permanent and unverifiable.
	if got := Queue([]store.Word{word("known", 3, 0)}, prog, last.AddDate(0, 0, iv), 10); keys(got) != "known" {
		t.Errorf("queue = %v %d days later, want the mastered word back — mastery must not be absorbing", got, iv)
	}
}

// The consequence that makes it matter: a mastered word CAN be demoted, so the
// status reflects current recall rather than a high-water mark.
func TestMasteryCanBeLost(t *testing.T) {
	var events []store.ReviewEvent
	for i := 0; i < MasteredBox; i++ {
		events = append(events, reviewed("known", true, at(i)))
	}
	before := Fold(events)[store.Key("known")].Box
	events = append(events, reviewed("known", false, at(200)))

	p := Fold(events)[store.Key("known")]

	if Mastered(p) {
		t.Error("still mastered after forgetting it — the status is a high-water mark, not recall")
	}
	if p.Box != before/2 {
		t.Errorf("box = %d, want %d (half of %d)", p.Box, before/2, before)
	}
}

// The budget must not be spent on fresh words while overdue ones wait: the
// starvation row again, at the boundary where it actually bites.
func TestBudgetGoesToOverdueBeforeFresh(t *testing.T) {
	deck := []store.Word{
		word("fresh1", 9, 0), word("fresh2", 9, 0),
		word("overdue1", 1, 40), word("overdue2", 1, 40),
	}
	prog := Fold([]store.ReviewEvent{
		reviewed("overdue1", true, at(40)),
		reviewed("overdue2", true, at(40)),
	})

	got := Queue(deck, prog, at(60), 2)

	if keys(got) != "overdue1,overdue2" {
		t.Errorf("queue = %v, want both overdue words — the budget went to fresh ones", got)
	}
}

// A deck can legitimately hold both spellings — hand-edited, or written before
// store.Key existed — and they are ONE word. Offering it twice also spends two
// budget slots on one review.
func TestQueueDedupesByKey(t *testing.T) {
	deck := []store.Word{word("Define", 5, 0), word("define", 2, 0)}

	got := Queue(deck, nil, at(1), 5)

	if len(got) != 1 {
		t.Errorf("queue = %v, want one entry — Define and define are one word", got)
	}
}

// budget is caller input, and make([]string, 0, 1<<62) panics.
func TestHugeBudgetDoesNotPanic(t *testing.T) {
	deck := []store.Word{word("a", 1, 0)}

	got := Queue(deck, nil, at(1), 1<<62)

	if len(got) != 1 {
		t.Errorf("queue = %v, want the one deck word", got)
	}
}

// The zero time is what an uninitialised clock hands over. It must not panic or
// silently offer everything.
func TestZeroNowIsHandled(t *testing.T) {
	deck := []store.Word{word("a", 1, 0)}
	prog := Fold([]store.ReviewEvent{reviewed("a", true, at(0))})

	_ = Queue(deck, prog, time.Time{}, 5) // must not panic
}

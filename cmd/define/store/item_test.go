package store_test

import (
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// Harvested is the whole absence signal for word facts — there is no second
// return value and no bool on the interface, so this predicate is what every
// caller branches on.
func TestWordFactsHarvestedIsTheTimestamp(t *testing.T) {
	if (store.WordFacts{}).Harvested() {
		t.Error("the zero WordFacts reports harvested; every unharvested word would be skipped")
	}
	// A band without a timestamp is NOT harvested. The zero value must be the
	// only thing that decides, or a partially-populated record read off disk
	// would claim to be a cache hit and suppress the call that would fix it.
	if (store.WordFacts{Band: store.C1}).Harvested() {
		t.Error("facts with a band but no At report harvested")
	}
	f := store.WordFacts{Band: store.C1, Domain: store.DomainGeneral, At: time.Now()}
	if !f.Harvested() {
		t.Error("facts with a timestamp report unharvested")
	}
}

// Done-when 7: growth is bounded and pruning is DETERMINISTIC — proved by
// pruning TWICE, not by inspecting one run.
func TestPruneIsDeterministic(t *testing.T) {
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	// Every item shares a timestamp, which is the case that separates
	// deterministic from usually-stable: two items authored in one run are
	// written at the same instant, so the tie-break is what decides.
	var items []store.Item
	for _, stem := range []string{"e", "c", "a", "d", "b", "f"} {
		items = append(items, store.Item{Word: "w", Stem: stem, At: at})
	}

	first := store.PruneForTest(items, 3)
	second := store.PruneForTest(items, 3)
	if len(first) != 3 {
		t.Fatalf("pruned to %d, want the cap of 3", len(first))
	}
	for i := range first {
		if first[i].Stem != second[i].Stem {
			t.Errorf("item %d: %q then %q — pruning twice gave different survivors",
				i, first[i].Stem, second[i].Stem)
		}
	}

	// And PRUNING A PRUNED LIST is a fixed point: running it again cannot drop
	// more. A cap that shrinks on every write would empty a word silently.
	third := store.PruneForTest(first, 3)
	if len(third) != len(first) {
		t.Errorf("pruning an already-pruned list went from %d to %d", len(first), len(third))
	}
}

// Newest survives, because recency is the only ranking available — nothing here
// can judge quality, and pretending to would be the self-oracle problem again.
func TestPruneKeepsTheNewest(t *testing.T) {
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	items := []store.Item{
		{Word: "w", Stem: "oldest", At: at.AddDate(0, 0, -2)},
		{Word: "w", Stem: "newest", At: at},
		{Word: "w", Stem: "middle", At: at.AddDate(0, 0, -1)},
	}
	got := store.PruneForTest(items, 1)
	if len(got) != 1 || got[0].Stem != "newest" {
		t.Errorf("kept %+v, want the newest item", got)
	}
}

// The cap is the STORE's guarantee, not a caller's discipline: a cap enforced by
// callers is one every future caller has to remember.
func TestTheStoreCapsAWordsItems(t *testing.T) {
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	var many []store.Item
	for i := range store.ItemCap * 3 {
		many = append(many, store.Item{
			Word: "w", Form: store.FormCloze, Stem: string(rune('a' + i)), At: at,
		})
	}
	s := store.NewMem()
	if err := s.SetItems("w", many); err != nil {
		t.Fatal(err)
	}
	got, err := s.Items("w")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != store.ItemCap {
		t.Errorf("stored %d items, want the cap of %d", len(got), store.ItemCap)
	}
}

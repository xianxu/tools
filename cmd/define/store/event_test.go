package store_test

import (
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// AddsAWord is TOTAL over the extent, so a kind added without an answer reddens
// rather than defaulting to "does not add a word".
//
// #67's rule, from the second finding in that family: a member added to a declared
// extent is only added when every PARTITIONING consumer is total or fails closed.
// EventMarked was appended to EventKinds() while schedule.Stats named
// EventLookedUp literally, so a deck built by marking reported Added == 0.
func TestAddsAWordIsTotalOverTheExtent(t *testing.T) {
	for _, k := range store.EventKinds() {
		if !store.AddsAWordDeclared(k) {
			t.Errorf("EventKind %q has no row in addsAWord — a kind nobody decided about is how "+
				"a deck built by marking came to report Added == 0", k)
		}
	}
}

// And the behaviour the declaration exists for.
func TestAMarkedWordCountsAsAdded(t *testing.T) {
	if !store.AddsAWord(store.EventMarked) {
		t.Error("a marked word does not count as added, so a learner who builds their deck by " +
			"marking passages sees Added == 0 and loses the words/day line")
	}
	if store.AddsAWord(store.EventAsked) {
		t.Error("a question was counted as adding a word; it reaches no deck")
	}
}

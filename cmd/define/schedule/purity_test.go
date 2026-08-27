package schedule_test

import (
	"testing"

	"github.com/xianxu/tools/cmd/define/puretest"
)

const schedulePkg = "github.com/xianxu/tools/cmd/define/schedule"

// The purity claim, ENFORCED — three guards, because each sees a hazard the
// others structurally cannot.
//
// Extracted into puretest at #6: schedule and play make the same claims, and
// #7/#12/#13 each add a form package, so copying the ~150 lines would have meant
// five copies to keep in agreement.
func TestSchedulePurity(t *testing.T) {
	t.Run("imports", func(t *testing.T) {
		// The claim is "no IO and no hidden clock", not "exactly two imports" —
		// the first version said the latter and fired on `sort`, which is as pure
		// as arithmetic. A guard that reddens on correct code invites deleting the
		// guard.
		puretest.ImportsOnly(t, schedulePkg, []string{
			"time", "sort", "slices", "cmp",
			"github.com/xianxu/tools/cmd/define/store",
		})
	})
	t.Run("no wall clock", func(t *testing.T) {
		puretest.NoWallClock(t, schedulePkg)
	})
	t.Run("only pure store symbols", func(t *testing.T) {
		puretest.StoreSymbolsOnly(t, schedulePkg, []string{
			"Key", "Word", "ReviewEvent",
			"EventReviewed", "EventLookedUp", "EventAsked",
			"StartOfDay", "DaysBetween",
		})
	})
}

package play_test

import (
	"testing"

	"github.com/xianxu/tools/cmd/define/puretest"
)

const playPkg = "github.com/xianxu/tools/cmd/define/play"

// TWO of schedule's three guards, from the same shared body.
//
// play is stricter: it needs no store symbols at all today, because the session
// works in deck KEYS the caller supplies rather than in store types. Declaring
// an empty allowlist would make the guard vacuous (it fatals when it finds
// nothing), so the store guard is deliberately absent and the import guard
// carries the weight — if play ever imports store, that line has to be added
// consciously, which is the point.
func TestPlayPurity(t *testing.T) {
	t.Run("imports", func(t *testing.T) {
		puretest.ImportsOnly(t, playPkg, []string{})
	})
	t.Run("no wall clock", func(t *testing.T) {
		puretest.NoWallClock(t, playPkg)
	})
}

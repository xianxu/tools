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

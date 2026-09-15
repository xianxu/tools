package main

import (
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// overlapDict records how many lookups were ever inside it at once.
type overlapDict struct{ in, max atomic.Int32 }

func (d *overlapDict) Lookup(string) (string, error) {
	n := d.in.Add(1)
	for m := d.max.Load(); n > m && !d.max.CompareAndSwap(m, n); m = d.max.Load() {
	}
	time.Sleep(time.Millisecond)
	d.in.Add(-1)
	return "", nil
}

// Two INSTANCES share the lock: the loop's dictionary and a background job's copy
// after /lang are different values over the same DictionaryServices (#54).
func TestLockedDictionarySerializesAcrossInstances(t *testing.T) {
	inner := &overlapDict{}
	a, b := lockedDictionary{inner: inner}, lockedDictionary{inner: inner}
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				a.Lookup("x")
			} else {
				b.Lookup("x")
			}
		}()
	}
	wg.Wait()
	if got := inner.max.Load(); got != 1 {
		t.Errorf("%d lookups overlapped; every dictionary call must hold one lock", got)
	}
}

// Production builds every dictionary through realDeps' newDict: run() calls it for
// the startup language and sessionSetLang for every /lang. Wrapping that one
// builder is what covers both, so this reads what it actually hands out.
func TestProductionDictionariesAreLocked(t *testing.T) {
	d := realDeps()
	for _, l := range []store.Lang{store.DefaultLang, "es"} {
		dict, _ := d.newDict(l, io.Discard)
		if _, ok := dict.(lockedDictionary); !ok {
			t.Errorf("realDeps().newDict(%s) built %T, not a lockedDictionary", l, dict)
		}
	}
}

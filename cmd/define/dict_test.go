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
		switch dict.(type) {
		case lockedDictionary, lockedSupplementalDictionary:
		default:
			t.Errorf("realDeps().newDict(%s) built unlocked %T", l, dict)
		}
	}
}

func TestLockedDictionaryPreservesSupplementalCapability(t *testing.T) {
	primary := "red nombre femenino tejido"
	inner := spanishDefinitions{Dictionary: &definitionFake{primary: primary}, english: &fakeRecordSource{installed: true, entries: map[string][]bilingualRecord{"red": bilingualFixture(t, "red")}}}
	build := lockedDictionaries(func(store.Lang, io.Writer) (Dictionary, string) { return inner, "Spanish" })
	dict, _ := build("es", io.Discard)
	text, err := dict.Lookup("red")
	set := definitionsFor(dict, "red", text, err, true)
	out, _ := renderDefinitions(set, RenderOpts{Color: true, Language: "es", Tint: tintPolicy{lang: "es", background: languageDark}})
	if !set.labeled || len(set.sections) != 2 {
		t.Fatal("locking erased the optional supplemental capability")
	}
	assertDictionaryTint(t, out, "subir a la red", false)
	assertDictionaryTint(t, out, "to go up to", false)
	english, _ := renderDefinitions(set, RenderOpts{Color: true, Language: "es", Tint: tintPolicy{lang: "en", background: languageDark}})
	assertDictionaryTint(t, english, "subir a la red", true)
	assertDictionaryTint(t, english, "to go up to", true)
	monoBuild := lockedDictionaries(func(store.Lang, io.Writer) (Dictionary, string) { return &overlapDict{}, "English" })
	mono, _ := monoBuild("en", io.Discard)
	if _, ok := mono.(supplementalDictionary); ok {
		t.Fatal("locking invented supplemental support for monolingual source")
	}
}

type overlapSupplementalDictionary struct{ *overlapDict }

func (d overlapSupplementalDictionary) primaryLabel() string { return "Spanish" }
func (d overlapSupplementalDictionary) supplement(word, primary string) definitionSection {
	d.Lookup(word)
	return definitionSection{entries: []string{"red noun net"}}
}

func TestLockedDictionarySerializesSupplementAndPrimaryTogether(t *testing.T) {
	inner := &overlapDict{}
	build := lockedDictionaries(func(store.Lang, io.Writer) (Dictionary, string) {
		return overlapSupplementalDictionary{inner}, "Spanish"
	})
	a, _ := build("es", io.Discard)
	b, _ := build("es", io.Discard)
	provider, ok := b.(supplementalDictionary)
	if !ok {
		t.Fatal("wrapper dropped supplement")
	}
	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				a.Lookup("red")
			} else {
				provider.supplement("red", "")
			}
		}()
	}
	wg.Wait()
	if got := inner.max.Load(); got != 1 {
		t.Fatalf("%d primary/supplement lookups overlapped", got)
	}
}

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/cmd/define/store/storetest"
)

// The classification, checked against the INTERFACE by the tests below rather
// than trusted. Named for what each method does to the disk — creates, or does
// not — because "write-shaped" is the wrong axis: Forget writes in the sense that
// it changes the deck, and creates nothing.
var createsOnDisk = map[string]bool{
	"Upsert": true, "AppendEvent": true, "SetUserModel": true,
	"SetNewsItems": true, "SetWordFacts": true, "SetItems": true, "SetAudio": true,
}

var doesNotCreate = map[string]bool{
	"Deck": true, "Events": true, "UserModel": true, "NewsItems": true,
	"WordFacts": true, "Items": true, "Audio": true, "Forget": true,
}

// storeInterfaceMethods reads the method set off `type Store interface` rather
// than restating it — the declaredModes shape (harvest_test.go), applied to an
// interface instead of a slice literal.
func storeInterfaceMethods(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join("store", "store.go"), nil, 0)
	if err != nil {
		t.Fatalf("parsing store/store.go: %v", err)
	}
	var names []string
	ast.Inspect(file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "Store" {
			return true
		}
		it, ok := ts.Type.(*ast.InterfaceType)
		if !ok {
			return true
		}
		for _, m := range it.Methods.List {
			for _, id := range m.Names {
				names = append(names, id.Name)
			}
		}
		return false
	})
	sort.Strings(names)
	return names
}

// EVERY Store METHOD IS CLASSIFIED, and the classification is DERIVED from the
// interface rather than typed beside it.
//
// This is #49's lesson applied BEFORE the bug instead of after it. A hand-listed
// "these are the writes" is a list a sixteenth method joins silently — and that
// method would then reach the disk of a directory the learner declined. An
// unknown method FAILS rather than defaulting, because defaulting is how a set
// nobody chose gets certified.
func TestEveryStoreMethodIsClassified(t *testing.T) {
	methods := storeInterfaceMethods(t)
	if len(methods) < 15 {
		t.Fatalf("derived %d Store methods %v; the interface has at least fifteen, so "+
			"this derivation is under-deriving and every check built on it would be "+
			"certifying a set nobody chose", len(methods), methods)
	}
	for _, m := range methods {
		switch {
		case createsOnDisk[m] && doesNotCreate[m]:
			t.Errorf("store.Store.%s is classified both ways", m)
		case !createsOnDisk[m] && !doesNotCreate[m]:
			t.Errorf("store.Store.%s is in neither createsOnDisk nor doesNotCreate. "+
				"Decide what it does to the DISK: if it can create a directory it must "+
				"be gated, or it writes into a directory nobody confirmed; if it only "+
				"reads or removes it must not be, or `define --forget` starts asking to "+
				"CREATE a deck in order to delete nothing.", m)
		}
	}
}

// And the classification has to MATCH the code, not merely exist.
func TestEveryCreatingMethodConsultsThePermission(t *testing.T) {
	for name := range createsOnDisk {
		t.Run(name, func(t *testing.T) {
			c := &counter{allow: true}
			s := newGatedStore(store.NewMem(), newDeckPermission(c.ask))
			callStoreMethod(t, s, name)
			if c.calls == 0 {
				t.Errorf("%s did not consult the permission — it would create in a "+
					"directory nobody confirmed", name)
			}
		})
	}
}

func TestNoNonCreatingMethodConsultsThePermission(t *testing.T) {
	for name := range doesNotCreate {
		t.Run(name, func(t *testing.T) {
			c := &counter{allow: true}
			s := newGatedStore(store.NewMem(), newDeckPermission(c.ask))
			callStoreMethod(t, s, name)
			if c.calls != 0 {
				t.Errorf("%s put the question to the learner. It creates nothing, so "+
					"asking permission to CREATE is a false alarm — and false alarms "+
					"train people to answer yes.", name)
			}
		})
	}
}

// callStoreMethod calls one method by name with zero values, which is enough:
// what is under test is whether the permission was consulted, not what came back.
func callStoreMethod(t *testing.T, s store.Store, name string) {
	t.Helper()
	m := reflect.ValueOf(s).MethodByName(name)
	if !m.IsValid() {
		t.Fatalf("%s is on the Store interface but not on gatedStore", name)
	}
	args := make([]reflect.Value, m.Type().NumIn())
	for i := range args {
		args[i] = reflect.Zero(m.Type().In(i))
	}
	m.Call(args)
}

// ONE DIRECTORY, ONE QUESTION, across a whole lookup's worth of writes.
func TestGatedStoreAsksOnceAcrossManyWrites(t *testing.T) {
	c := &counter{allow: true}
	s := newGatedStore(store.NewMem(), newDeckPermission(c.ask))
	_ = s.Upsert(store.Word{Text: "alpha"})
	_ = s.AppendEvent(store.ReviewEvent{Word: "alpha", At: time.Now()})
	_ = s.SetItems("alpha", nil)
	if c.calls != 1 {
		t.Errorf("asked %d times for one lookup, want 1", c.calls)
	}
}

// DENIAL REACHES THE BACKING STORE NOT AT ALL — asserted against the BACKING
// store, not through the wrapper, because the wrapper is the thing under test.
func TestDeniedWritesNeverReachTheDisk(t *testing.T) {
	backing := store.NewMem()
	s := newGatedStore(backing, newDeckPermission(func() bool { return false }))

	if err := s.Upsert(store.Word{Text: "alpha"}); err != nil {
		t.Errorf("Upsert returned %v — a declined write is an ANSWER, not a failure; "+
			"an error would put 'could not save' in front of someone who said not to", err)
	}
	if deck, _ := backing.Deck(); len(deck) != 0 {
		t.Errorf("the backing store holds %d word(s); nothing may reach a directory "+
			"the learner declined", len(deck))
	}
}

// A DENIED SESSION STILL REMEMBERS ITSELF, which is what memHistory already does
// on the no-capture path — the two degraded modes agree.
func TestDeniedSessionKeepsItselfInMemory(t *testing.T) {
	s := newGatedStore(store.NewMem(), newDeckPermission(func() bool { return false }))
	if err := s.Upsert(store.Word{Text: "alpha"}); err != nil {
		t.Fatal(err)
	}
	deck, err := s.Deck()
	if err != nil {
		t.Fatal(err)
	}
	if len(deck) != 1 {
		t.Errorf("deck = %d, want 1 — a denied session writes nothing to disk but "+
			"still recalls what it did", len(deck))
	}
}

// AN ALLOWING PERMISSION MUST BE INVISIBLE. Whatever storetest.Suite asserts
// about a Store it asserts about this one — the strongest available check that a
// wrapper delegates faithfully, since delegating wrong is a wrapper's only
// failure mode.
func TestGatedStoreConformsWhenAllowed(t *testing.T) {
	storetest.Suite(t, func(t *testing.T) store.Store {
		return newGatedStore(store.NewMem(), newDeckPermission(func() bool { return true }))
	})
}

// AND WHEN DENIED, because a declined store is not a broken one — it is an empty
// one that keeps the session. Every read-after-write invariant the suite states
// still holds; what differs is only WHERE the bytes go, which the suite never
// asserts.
func TestGatedStoreConformsWhenDenied(t *testing.T) {
	storetest.Suite(t, func(t *testing.T) store.Store {
		disk := store.NewYAML(t.TempDir(), store.DefaultLang, io.Discard)
		return newGatedStore(disk, newDeckPermission(func() bool { return false }))
	})
}

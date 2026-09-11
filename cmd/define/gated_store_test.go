package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
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

// --- the wiring, which is the feature -----------------------------------------
//
// gatedStore being correct is not the claim that openStore USES it. That gap is
// already a recorded lesson in this repo (#72): a pure helper that is correct and
// never called, with a green suite either way.

func TestOpenStoreGatesTheLanguageDeck(t *testing.T) {
	t.Chdir(t.TempDir())
	sd := openStore(options{}, io.Discard, newDeckPermission(func() bool { return true }))
	if _, ok := sd.deck.(*gatedStore); !ok {
		t.Errorf("deck is %T, want *gatedStore — an ungated deck creates in a "+
			"directory nobody confirmed", sd.deck)
	}
}

// THE FLAT STORE TOO, not just the language one (#50 PQ-2).
//
// `flat` backs BOTH newStoreHistory and the news cache, and
// cachingFeed.SetNewsItems reaches MkdirAll(usageDir) — so wrapping only the
// language store leaves a live write path that creates a deck having asked
// nothing.
//
// ASSERTED ON THE WIRING, not through history.Add, and that correction is worth
// recording: a first version called sd.history.Add("alpha") and then listed the
// directory. It passed under the ungated mutation, because storeHistory.Add only
// appends to an in-memory slice (history_store.go:77) and never writes at all —
// so the test exercised no write path and proved nothing. The write that matters
// goes through the news cache, which needs a feed and a network; the honest
// cheap assertion is that the store handed to these consumers IS the gated one.
func TestOpenStoreGatesTheFlatStoreToo(t *testing.T) {
	t.Chdir(t.TempDir())
	sd := openStore(options{}, io.Discard, newDeckPermission(func() bool { return false }))

	h, ok := sd.history.(*storeHistory)
	if !ok {
		t.Fatalf("history is %T, want *storeHistory", sd.history)
	}
	if _, gated := h.st.(*gatedStore); !gated {
		t.Errorf("the history store is backed by %T, want *gatedStore — `flat` also "+
			"backs the news cache, whose SetNewsItems reaches MkdirAll(usageDir), so "+
			"an ungated `flat` creates a deck having asked nothing", h.st)
	}
}

// And the write path `flat` actually owns: the news cache, driven directly so no
// network is involved. This is the behavioural half of the pin above.
func TestTheNewsCacheCannotCreateWhenDenied(t *testing.T) {
	dir := t.TempDir()
	perm := newDeckPermission(func() bool { return false })
	gated := newGatedStore(store.NewYAML(dir, store.DefaultLang, io.Discard), perm)

	if err := gated.SetNewsItems("k", []store.NewsItem{{Title: "x"}}, time.Now()); err != nil {
		t.Fatalf("SetNewsItems: %v", err)
	}
	if names := lsNames(t, dir); len(names) != 0 {
		t.Errorf("a denied news-cache write created %v; usage/ is a deck directory "+
			"like any other", names)
	}
}

// persistLang IS THE ONE WRITE PATH A Store WRAPPER CANNOT REACH (#50 PQ-1).
//
// store.WriteLang is a free function, so no amount of wrapping the Store
// interface touches it. Ungated, `define /lang es` writes lang.txt into a
// directory nobody confirmed — and so does /lang AFTER a decline, which a
// one-shot lookup test would never see.
func TestPersistLangIsGated(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	sd := openStore(options{}, io.Discard, newDeckPermission(func() bool { return false }))

	if err := sd.persistLang(store.Lang("es")); err != nil {
		t.Errorf("persistLang returned %v — declining is an answer, not a failure", err)
	}
	if names := lsNames(t, dir); len(names) != 0 {
		t.Errorf("declining still wrote %v; store.WriteLang is a free function and a "+
			"Store wrapper cannot reach it, so it needs the permission directly", names)
	}
}

func TestPersistLangWritesWhenAllowed(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	sd := openStore(options{}, io.Discard, newDeckPermission(func() bool { return true }))
	if err := sd.persistLang(store.Lang("es")); err != nil {
		t.Fatalf("persistLang: %v", err)
	}
	if names := lsNames(t, dir); len(names) == 0 {
		t.Error("an allowed persistLang wrote nothing")
	}
}

// The Integration-points section claims MigrateToLanguages creates nothing in a
// non-deck directory, which is why it is not gated. A claim about code is pinned
// or it is folklore.
func TestMigrateCreatesNothingInANonDeckDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := store.MigrateToLanguages(dir, io.Discard); err != nil {
		t.Fatalf("MigrateToLanguages: %v", err)
	}
	if names := lsNames(t, dir); len(names) != 0 {
		t.Errorf("migration created %v in a directory with no deck; it is UNGATED on "+
			"the claim that it cannot, so this is the claim failing", names)
	}
}

func lsNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

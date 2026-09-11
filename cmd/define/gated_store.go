package main

import (
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// gatedStore is a store.Store that asks before the first write REACHES DISK.
//
// IT WRAPS RATHER THAN MODIFIES store.YAML, and the split is ARCH-PURE: whether
// to create a deck is a policy about a person's intent; YAML is about bytes. The
// store package's only UI is a `warn io.Writer` — it must not prompt — so the
// decision lives in the command and travels down as a deckPermission.
//
// ON DENIAL IT SERVES store.Mem RATHER THAN ERRORING, and that is what makes this
// feature cheap. A nil deck is REFUSED at eight sites (stats.go:37,
// play_loop.go:29, harvest.go:110, …) and quietly degrades at two more, so
// teaching all of them a new "empty but absent" meaning would be the expensive
// version. An empty Store means them no changes at all: --stats folds nothing,
// --play finds nothing due, /history recalls only this session.
//
// A DECLINED WRITE IS NOT AN ERROR. The learner was asked and answered; returning
// an error would put "could not save" in front of someone who said not to.
type gatedStore struct {
	disk store.Store
	mem  store.Store
	perm *deckPermission
}

// newGatedStore wraps `disk`, falling back to `mem` when the learner declines.
//
// THE FALLBACK IS PASSED IN, NOT ALLOCATED HERE, and that is #50 BR-2: an earlier
// version called store.NewMem() per wrapper, and newLangDeps rebuilds the wrapper
// on every /lang. Measured by the boundary review: allowed kept a word across a
// switch, denied dropped to zero — so "a denied session still recalls itself"
// held until the learner changed language, then silently stopped. That is PQ-2's
// rule (one decision for the process) applied to the decision but not to the
// store it swaps in. The caller owns the fallback's lifetime and keys it by
// language, exactly as it keys the real store.
func newGatedStore(disk, mem store.Store, perm *deckPermission) *gatedStore {
	return &gatedStore{disk: disk, mem: mem, perm: perm}
}

// creating is where a write that CREATES goes, resolving the decision first.
func (g *gatedStore) creating() store.Store {
	if g.perm.allowed() {
		return g.disk
	}
	return g.mem
}

// reading is where reads go, and it NEVER resolves the decision.
//
// Before the question is settled the disk is the honest source: a directory
// nobody has been asked about is by construction not a deck, so it reads empty
// anyway. Resolving in order to answer a READ is the false alarm this design
// exists to prevent.
func (g *gatedStore) reading() store.Store {
	if allowed, decided := g.perm.saving(); decided && !allowed {
		return g.mem
	}
	return g.disk
}

// --- does not create: never gated -------------------------------------------

func (g *gatedStore) Deck() ([]store.Word, error) { return g.reading().Deck() }
func (g *gatedStore) Events(since time.Time) ([]store.ReviewEvent, error) {
	return g.reading().Events(since)
}
func (g *gatedStore) UserModel() (string, error) { return g.reading().UserModel() }
func (g *gatedStore) NewsItems(key string) ([]store.NewsItem, time.Time, error) {
	return g.reading().NewsItems(key)
}
func (g *gatedStore) WordFacts(key string) (store.WordFacts, error) {
	return g.reading().WordFacts(key)
}
func (g *gatedStore) Items(key string) ([]store.Item, error) { return g.reading().Items(key) }
func (g *gatedStore) Audio(k store.AudioKey) ([]byte, store.AudioRecord, error) {
	return g.reading().Audio(k)
}

// Forget REMOVES; it never creates a directory (YAML.Forget is os.Remove and
// os.RemoveAll only). Gating it would make `define --forget x` in a non-deck
// directory ask permission to CREATE a deck in order to delete nothing — the
// exact false alarm the lazy rule exists to prevent. It is classified by what it
// does to the disk, not by being write-shaped.
func (g *gatedStore) Forget(key string) (bool, error) { return g.reading().Forget(key) }

// --- creates on disk: gated --------------------------------------------------

func (g *gatedStore) Upsert(w store.Word) error { return g.creating().Upsert(w) }
func (g *gatedStore) AppendEvent(e store.ReviewEvent) error {
	return g.creating().AppendEvent(e)
}
func (g *gatedStore) SetUserModel(text string) error { return g.creating().SetUserModel(text) }
func (g *gatedStore) SetNewsItems(key string, items []store.NewsItem, at time.Time) error {
	return g.creating().SetNewsItems(key, items, at)
}
func (g *gatedStore) SetWordFacts(key string, f store.WordFacts) error {
	return g.creating().SetWordFacts(key, f)
}
func (g *gatedStore) SetItems(key string, items []store.Item) error {
	return g.creating().SetItems(key, items)
}
func (g *gatedStore) SetAudio(k store.AudioKey, data []byte, rec store.AudioRecord) error {
	return g.creating().SetAudio(k, data, rec)
}

var _ store.Store = (*gatedStore)(nil)

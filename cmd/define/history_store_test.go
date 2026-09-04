package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// loaded restores a storeHistory's recall state. Reading the log moved out of
// the constructor: opening a store and READING it are different costs, and only
// the raw editor recalls — a one-shot lookup or a command was paying for a log
// it never consulted, twice in /history's case.
func loaded(h *storeHistory) *storeHistory {
	h.Load()
	return h
}

func fixedClock(day int) store.Clock {
	return store.FixedClock(time.Date(2026, 8, day, 12, 0, 0, 0, time.UTC))
}

func TestStoreHistoryPrefixIsNewestFirstAndDeduped(t *testing.T) {
	h := newStoreHistory(store.NewMem(), nil)
	h.Add("sycophantic")
	h.Add("ephemeral")
	h.Add("sycophantic") // looked up again

	got := h.Prefix("")
	if len(got) != 2 || got[0] != "sycophantic" {
		t.Errorf("Prefix = %v, want [sycophantic ephemeral]", got)
	}
}

// Prefix runs on every keystroke and returns no error, so it must never touch
// the disk. Asserted by removing the store's ability to answer.
func TestStoreHistoryPrefixDoesNotQueryTheStore(t *testing.T) {
	counting := &countingStore{Store: store.NewMem()}
	h := newStoreHistory(counting, nil)
	h.Add("sycophantic")

	before := counting.reads
	for i := 0; i < 50; i++ {
		h.Prefix("sy")
	}
	if counting.reads != before {
		t.Errorf("Prefix hit the store %d times over 50 keystrokes", counting.reads-before)
	}
}

// countingStore counts the store's two READS, separately and together.
//
// `reads` is the total, which is what a caller asking "did this path touch the
// disk at all" wants. `decks` and `events` are the halves, because #41 D7 claims
// something narrower — that a whole sitting calls Deck() exactly once and
// Events() exactly once — and a sum cannot tell one extra deck read from one
// fewer log read.
type countingStore struct {
	store.Store
	reads  int
	decks  int
	events int
}

func (c *countingStore) Events(t time.Time) ([]store.ReviewEvent, error) {
	c.reads++
	c.events++
	return c.Store.Events(t)
}
func (c *countingStore) Deck() ([]store.Word, error) {
	c.reads++
	c.decks++
	return c.Store.Deck()
}

type failingStore struct{}

func (failingStore) Upsert(store.Word) error             { return errFail }
func (failingStore) Deck() ([]store.Word, error)         { return nil, errFail }
func (failingStore) AppendEvent(store.ReviewEvent) error { return errFail }
func (failingStore) Events(time.Time) ([]store.ReviewEvent, error) {
	return nil, errFail
}
func (failingStore) UserModel() (string, error)  { return "", errFail }
func (failingStore) SetUserModel(string) error   { return errFail }
func (failingStore) Forget(string) (bool, error) { return false, errFail }
func (failingStore) NewsItems(string) ([]store.NewsItem, time.Time, error) {
	return nil, time.Time{}, errFail
}
func (failingStore) SetNewsItems(string, []store.NewsItem, time.Time) error { return errFail }
func (failingStore) WordFacts(string) (store.WordFacts, error) {
	return store.WordFacts{}, errFail
}
func (failingStore) SetWordFacts(string, store.WordFacts) error { return errFail }
func (failingStore) Items(string) ([]store.Item, error)         { return nil, errFail }
func (failingStore) SetItems(string, []store.Item) error        { return errFail }

var errFail = &failErr{}

type failErr struct{}

func (*failErr) Error() string { return "store unavailable" }

// I-4: the wiring IS the issue. Nothing set deps.history, so runEditor's
// `hist := d.history` was only ever reached through the nil fallback — deleting
// the field left the whole suite green while persistence silently stopped.
func TestEditorPersistsThroughDeps(t *testing.T) {
	dir := t.TempDir()

	// Both halves are wired, because they are now different objects: the capturer
	// writes, storeHistory reads. Setting only history would persist nothing —
	// which is exactly the behaviour #4 moved.
	first, opt, finish := editorRig(t, "sycophantic", true)
	st1 := store.NewYAML(dir, store.DefaultLang, nil)
	first.deps.history = newStoreHistory(st1, nil)
	first.deps.capture = newStoreCapturer(st1, fixedClock(1), nil, nil)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r"), nil, first.deps, opt, recordingConsole(&out, &errb, finish))

	// A second editor over the same directory: the restart case.
	second, opt2, finish2 := editorRig(t, "sycophantic", true)
	st2 := store.NewYAML(dir, store.DefaultLang, nil)
	second.deps.history = newStoreHistory(st2, nil)
	second.deps.capture = newStoreCapturer(st2, fixedClock(2), nil, nil)
	var out2 bytes.Buffer
	runEditor(t.Context(), scriptKeys("syc"), nil, second.deps, opt2, recordingConsole(&out2, &bytes.Buffer{}, finish2))

	if !strings.Contains(out2.String(), greyOn+"ophantic") {
		t.Errorf("the previous session's word was not suggested: %q", tailOf(out2.String()))
	}
}

// Restoring history must include NOT-FOUND lines. Adding `&& e.Found` to the
// restore loop breaks Up-arrow recall of typos after a restart, and every other
// test still passes — so this pins it directly.
func TestStoreHistoryRestoresTyposAcrossSessions(t *testing.T) {
	dir := t.TempDir()
	st := store.NewYAML(dir, store.DefaultLang, nil)

	// Writes go through the CAPTURER now (#4); storeHistory only recalls.
	c := newStoreCapturer(st, fixedClock(1), nil, nil)
	c.Capture("sykophantic", false, options{}) // a typo, never in the deck
	c.Capture("ephemeral", true, options{})

	restored := loaded(newStoreHistory(store.NewYAML(dir, store.DefaultLang, nil), nil)).Prefix("sy")
	if len(restored) != 1 || restored[0] != "sykophantic" {
		t.Errorf("Prefix(sy) after restart = %v — the typo was dropped from recall", restored)
	}
}

// Moved from #3 with the writes it asserts: persistence is now the capturer's
// job, and storeHistory's job is to read it back.
func TestCapturedWordsPersistAcrossSessions(t *testing.T) {
	dir := t.TempDir()
	st := store.NewYAML(dir, store.DefaultLang, nil)

	c := newStoreCapturer(st, fixedClock(1), nil, nil)
	c.Capture("sycophantic", true, options{})
	c.Capture("ephemeral", true, options{})

	got := loaded(newStoreHistory(store.NewYAML(dir, store.DefaultLang, nil), nil)).Prefix("")
	if len(got) != 2 {
		t.Fatalf("restored %d entries, want 2: %v", len(got), got)
	}
	if got[0] != "ephemeral" {
		t.Errorf("newest-first ordering lost across the restart: %v", got)
	}
}

// Moved from #3: the deck/recall split is the capturer's decision now.
func TestCapturerRecallsTyposButDoesNotDeckThem(t *testing.T) {
	dir := t.TempDir()
	st := store.NewYAML(dir, store.DefaultLang, nil)
	c := newStoreCapturer(st, fixedClock(1), nil, nil)

	c.Capture("sycophantic", true, options{})
	c.Capture("sykophantic", false, options{})

	if got := loaded(newStoreHistory(store.NewYAML(dir, store.DefaultLang, nil), nil)).Prefix("sy"); len(got) != 2 {
		t.Errorf("Prefix returned %v — a failed lookup must still be recallable", got)
	}
	deck, err := st.Deck()
	if err != nil {
		t.Fatal(err)
	}
	if len(deck) != 1 || deck[0].Text != "sycophantic" {
		t.Errorf("deck = %+v, want only the word that exists", deck)
	}
}

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

func fixedClock(day int) store.Clock {
	return store.FixedClock(time.Date(2026, 8, day, 12, 0, 0, 0, time.UTC))
}

// The whole point of this issue: history outlives the process.
func TestStoreHistoryPersistsAcrossSessions(t *testing.T) {
	dir := t.TempDir()

	first := newStoreHistory(store.NewYAML(dir, nil), fixedClock(1), nil)
	first.Add("sycophantic", true)
	first.Add("ephemeral", true)

	// A second storeHistory over the same directory is what a restart looks like.
	second := newStoreHistory(store.NewYAML(dir, nil), fixedClock(2), nil)
	got := second.Prefix("")
	if len(got) != 2 {
		t.Fatalf("restored %d entries, want 2: %v", len(got), got)
	}
	if got[0] != "ephemeral" {
		t.Errorf("newest-first ordering lost across the restart: %v", got)
	}
}

// Add and Prefix answer DIFFERENT questions. A typo must be recallable — that is
// when you most want to edit and retry — but must not become a deck entry.
func TestStoreHistoryRecallsTyposButDoesNotDeckThem(t *testing.T) {
	dir := t.TempDir()
	st := store.NewYAML(dir, nil)
	h := newStoreHistory(st, fixedClock(1), nil)

	h.Add("sycophantic", true)
	h.Add("sykophantic", false) // a typo

	if got := h.Prefix("sy"); len(got) != 2 {
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

func TestStoreHistoryPrefixIsNewestFirstAndDeduped(t *testing.T) {
	h := newStoreHistory(store.NewMem(), fixedClock(1), nil)
	h.Add("sycophantic", true)
	h.Add("ephemeral", true)
	h.Add("sycophantic", true) // looked up again

	got := h.Prefix("")
	if len(got) != 2 || got[0] != "sycophantic" {
		t.Errorf("Prefix = %v, want [sycophantic ephemeral]", got)
	}
}

// Prefix runs on every keystroke and returns no error, so it must never touch
// the disk. Asserted by removing the store's ability to answer.
func TestStoreHistoryPrefixDoesNotQueryTheStore(t *testing.T) {
	counting := &countingStore{Store: store.NewMem()}
	h := newStoreHistory(counting, fixedClock(1), nil)
	h.Add("sycophantic", true)

	before := counting.reads
	for i := 0; i < 50; i++ {
		h.Prefix("sy")
	}
	if counting.reads != before {
		t.Errorf("Prefix hit the store %d times over 50 keystrokes", counting.reads-before)
	}
}

// A store that cannot be written must not break the editor: warn once, keep going.
func TestStoreHistoryDegradesOnWriteFailure(t *testing.T) {
	var warn strings.Builder
	h := newStoreHistory(failingStore{}, fixedClock(1), &warn)

	h.Add("sycophantic", false)
	h.Add("ephemeral", false)

	if got := h.Prefix(""); len(got) != 2 {
		t.Errorf("session history broke when the store failed: %v", got)
	}
	if n := strings.Count(warn.String(), "define:"); n != 1 {
		t.Errorf("warned %d times, want exactly 1 — not once per keystroke", n)
	}
}

type countingStore struct {
	store.Store
	reads int
}

func (c *countingStore) Events(t time.Time) ([]store.ReviewEvent, error) {
	c.reads++
	return c.Store.Events(t)
}
func (c *countingStore) Deck() ([]store.Word, error) {
	c.reads++
	return c.Store.Deck()
}

type failingStore struct{}

func (failingStore) Upsert(store.Word) error             { return errFail }
func (failingStore) Deck() ([]store.Word, error)         { return nil, errFail }
func (failingStore) AppendEvent(store.ReviewEvent) error { return errFail }
func (failingStore) Events(time.Time) ([]store.ReviewEvent, error) {
	return nil, errFail
}

var errFail = &failErr{}

type failErr struct{}

func (*failErr) Error() string { return "store unavailable" }

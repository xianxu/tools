package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func TestMemVocabularyNormalisesOnTheWayIn(t *testing.T) {
	v := &memVocabulary{}
	v.Add("Hot  Dog")

	if !v.Has(store.Key("hot dog")) {
		t.Error("Add must normalise through store.Key, the same identity the deck uses")
	}
}

func TestMaxPhraseWordsTracksTheLongestEntry(t *testing.T) {
	v := &memVocabulary{}
	if got := v.MaxPhraseWords(); got != 0 {
		t.Errorf("empty: MaxPhraseWords = %d, want 0 — nothing to look ahead for", got)
	}
	v.Add("obsequious")
	if got := v.MaxPhraseWords(); got != 1 {
		t.Errorf("one word: MaxPhraseWords = %d, want 1", got)
	}
	v.Add("a priori")
	if got := v.MaxPhraseWords(); got != 2 {
		t.Errorf("a phrase: MaxPhraseWords = %d, want 2", got)
	}
	v.Add("hot dog") // no longer than the last: the max must not drift up
	if got := v.MaxPhraseWords(); got != 2 {
		t.Errorf("MaxPhraseWords = %d, want 2", got)
	}
}

func TestEmptyVocabularyHasNothing(t *testing.T) {
	v := &memVocabulary{}
	if v.Has("") || v.Has("obsequious") {
		t.Error("an empty vocabulary must answer no to everything")
	}
}

// Blank input must not become a member — an empty key would then match the
// empty candidate a caller builds at a boundary.
func TestMemVocabularyIgnoresBlankWords(t *testing.T) {
	v := &memVocabulary{}
	v.Add("")
	v.Add("   ")

	if v.Has("") || v.MaxPhraseWords() != 0 {
		t.Error("blank words must not enter the set")
	}
}

func TestStoreVocabularyLoadsTheDeckOnce(t *testing.T) {
	st := store.NewMem()
	if err := st.Upsert(store.Word{Text: "obsequious"}); err != nil {
		t.Fatal(err)
	}
	if err := st.Upsert(store.Word{Text: "hot dog"}); err != nil {
		t.Fatal(err)
	}
	var warn bytes.Buffer
	v := newStoreVocabulary(st, &warn)

	if v.Has(store.Key("obsequious")) {
		t.Error("Has answered before Load — the deck read must be explicit, as History's is")
	}
	v.Load()
	v.Load() // second Load is free and must not double anything

	if !v.Has(store.Key("obsequious")) || !v.Has(store.Key("hot dog")) {
		t.Error("Load did not fill the set from the deck")
	}
	if got := v.MaxPhraseWords(); got != 2 {
		t.Errorf("MaxPhraseWords = %d, want 2", got)
	}
	if warn.Len() != 0 {
		t.Errorf("unexpected warning: %q", warn.String())
	}
}

// A deck that cannot be READ must degrade to "nothing highlighted". Highlighting
// is decoration; it may never be the reason a lookup fails.
func TestStoreVocabularyDegradesWhenTheDeckCannotBeRead(t *testing.T) {
	var warn bytes.Buffer
	v := newStoreVocabulary(errDeck{err: errors.New("disk gone")}, &warn)

	v.Load()

	if v.Has(store.Key("obsequious")) {
		t.Error("a failed deck read must leave the set empty")
	}
	if !strings.Contains(warn.String(), "disk gone") {
		t.Errorf("warning = %q, want it to name the underlying error", warn.String())
	}
}

// errDeck is a Store whose Deck() fails. The embedded nil Store is deliberate:
// anything else this double is asked for panics, so a Vocabulary that reaches
// past Deck() fails loudly instead of silently getting a zero value.
type errDeck struct {
	store.Store
	err error
}

func (e errDeck) Deck() ([]store.Word, error) { return nil, e.err }

// A word looked up NOW must highlight on the very next frame, without a restart.
// Capture is the one place that already knows a lookup both succeeded and earned
// a deck entry, so it is the one place that grows the set.
func TestCaptureGrowsTheHighlightSet(t *testing.T) {
	st := store.NewMem()
	v := &memVocabulary{}
	c := newStoreCapturer(st, store.FixedClock(aDay), nil, v)

	c.Capture("obsequious", true, options{})

	if !v.Has(store.Key("obsequious")) {
		t.Error("a successful lookup did not enter the highlight set")
	}
}

// A failed lookup is history, not vocabulary — decideCapture already says so for
// the deck, and the highlight set must agree. Highlighting a word that is not in
// the deck would highlight typos.
func TestCaptureDoesNotAddAFailedLookup(t *testing.T) {
	st := store.NewMem()
	v := &memVocabulary{}
	c := newStoreCapturer(st, store.FixedClock(aDay), nil, v)

	c.Capture("obseqious", false, options{})

	if v.Has(store.Key("obseqious")) {
		t.Error("a failed lookup entered the highlight set")
	}
}

// The two seams must be the SAME object, or Adds land in a set nothing renders
// from. This is the wiring assertion; the two above only prove the mechanism.
func TestOpenStoreSharesOneHighlightSet(t *testing.T) {
	t.Chdir(t.TempDir())
	var warn bytes.Buffer

	sd := openStore(options{}, &warn)
	sd.capture.Capture("obsequious", true, options{})

	if !sd.vocab.Has(store.Key("obsequious")) {
		t.Error("openStore handed the capturer a different set than it handed the renderer")
	}
}

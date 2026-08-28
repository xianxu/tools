package main

import (
	"fmt"
	"io"
	"sync"

	"github.com/xianxu/tools/cmd/define/store"
)

// Vocabulary is the set of words the learner knows, and the ONE predicate every
// highlight decision goes through.
//
// Today it is filled from the deck. #22 narrows it to the words still being
// learned — a word that has become theirs stops being highlighted — and that is
// the whole reason this is a predicate rather than a deck read at each call
// site: the swap is one constructor, and nothing above the seam moves.
//
// Deliberately NOT History, though the shapes rhyme. Different source (words/
// vs the event log), different query (set membership vs ordered prefix search),
// and — decisive — different contents: history carries typos on purpose so they
// stay recallable (#20), and highlighting a misspelling as a word you know is
// the opposite of reinforcement.
type Vocabulary interface {
	// Load reads whatever durable state the set needs, once. Separate from
	// construction for the same reason History.Load is: opening a store and
	// READING the deck are different costs, and a one-shot `define /help` should
	// pay neither.
	Load()
	// Add puts a word in the set, normalising it the way the deck does.
	Add(word string)
	// Has answers for an ALREADY-NORMALISED key — callers build candidate keys as
	// they scan text, so normalising here would mean doing it twice per token.
	Has(key string) bool
	// MaxPhraseWords is how many tokens the longest entry spans.
	//
	// Not a leak of the implementation: a streaming renderer has to know how far
	// to look ahead before it can rule out a phrase match, and for a deck of
	// single words the answer is 1 and it holds back almost nothing.
	MaxPhraseWords() int
}

// memVocabulary is the in-memory set, and the only implementation of the set
// itself — storeVocabulary embeds it rather than growing a second one.
//
// Mutex-guarded, and the honest reason is narrower than the one this comment
// used to give. It claimed the capture path and the render run concurrently;
// measurement says otherwise — the loop's goroutines carry values over channels
// and none of them touch a vocabulary, so every Add/Has runs on the loop
// goroutine today. The lock is here so the type stays safe if that changes,
// which is a defensible reason to keep it but not a description of what happens.
//
// storeVocabulary's `loaded` is guarded by the SAME mutex, not left bare beside
// it: vocabularyFor calls Load on every render, so an unguarded flag was a real
// data race the moment any second goroutine rendered — TestVocabularyIsSafeUnder
// Concurrency drives it.
//
// What that does NOT buy, stated so the comment stops over-licensing: Load is
// not a barrier. It releases the mutex before reading the deck, so a concurrent
// caller's Load can return while the set is still filling — measured, 97 of 200
// two-goroutine trials. That is safe (no torn state, no race) and it is the
// right trade, since holding the lock across a file read would serialise every
// render behind IO. It just means "Load returned" does not mean "the deck is
// visible". Nothing needs it to: the render that follows shows one frame without
// the highlight, and the next keystroke redraws with it.
type memVocabulary struct {
	mu       sync.RWMutex
	words    map[string]bool
	maxWords int
}

// Load is a no-op: there is nothing durable behind an in-memory set.
func (v *memVocabulary) Load() {}

func (v *memVocabulary) Add(word string) {
	key := store.Key(word)
	if key == "" {
		// A blank key would match the empty candidate a scanner builds at a
		// boundary, which would highlight nothing-shaped runs of text.
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.words == nil {
		v.words = map[string]bool{}
	}
	v.words[key] = true
	// Counted with wordRuns, so the set agrees with the tokenizer that will look
	// it up — and counted ONLY for keys whose tokens can actually rejoin.
	//
	// A key holding other punctuation (`e.g.`, `9/11`) is permanently
	// unmatchable, because phraseGap allows only spaces and tabs between a
	// phrase's tokens. Letting it raise maxWords anyway made every stream hold a
	// wider window for a match that can never happen: measured on "the quick
	// brown fox jumps", a single-word deck holds 5 bytes and adding the
	// unmatchable `e.g.` holds 9 — the same cost as a real three-token phrase.
	// A bound derived from an input set must come from the subset that can
	// exercise it. TestAPunctuatedKeyIsNotMatchable pins the matching half.
	runs := wordRuns(key)
	if n := len(runs); n > v.maxWords && phraseRunsJoin(key, runs) {
		v.maxWords = n
	}
}

func (v *memVocabulary) Has(key string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.words[key]
}

func (v *memVocabulary) MaxPhraseWords() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.maxWords
}

// storeVocabulary fills the set from the deck. It adds exactly one behaviour —
// Load — and inherits the set from memVocabulary, so "what is in the set" has
// one implementation rather than two that have to agree (ARCH-DRY).
type storeVocabulary struct {
	memVocabulary
	st     store.Store
	warn   io.Writer
	loaded bool
}

func newStoreVocabulary(st store.Store, warn io.Writer) *storeVocabulary {
	return &storeVocabulary{st: st, warn: warn}
}

// Load reads the deck once, when a loop is about to render from it.
//
// A deck that cannot be read leaves the set EMPTY and warns. Highlighting is
// decoration; it must never be the reason a lookup fails or the program stops.
func (v *storeVocabulary) Load() {
	if v.st == nil {
		return
	}
	v.mu.Lock()
	if v.loaded {
		v.mu.Unlock()
		return
	}
	v.loaded = true
	v.mu.Unlock()
	deck, err := v.st.Deck()
	if err != nil {
		v.warnf("could not read the deck for highlighting: %v", err)
		return
	}
	for _, w := range deck {
		v.Add(w.Text)
	}
}

func (v *storeVocabulary) warnf(format string, args ...any) {
	warnTo(v.warn, format, args...)
}

// warnTo is the one place package main writes the "define: " prefix and the
// trailing newline. Three seams warn — history, capture and this one — and each
// keeps its OWN policy (capture warns once behind a mutex, history appends
// "history is session-only"); what they share is the shape of the line, and that
// was written out three times.
//
// Package store keeps its own copy (YAML.warnf), deliberately: it cannot import
// package main, and a shared formatter would be a dependency in the wrong
// direction for one format string. "The one place" is scoped to this package,
// which is what the claim can honestly cover.
func warnTo(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "define: "+format+"\n", args...)
}

// vocabularyFor answers "what should be highlighted right now" for every caller
// that renders, and it is the ONE place that answers it.
//
// It exists because the same two questions were being asked in different places
// and one of them was missed. Highlighting needs the set LOADED, and Load lived
// in runEditor — so `define <word>` and piped stdin, which never enter the raw
// editor, rendered definitions against an empty set and silently highlighted
// nothing. Two of three entry paths were dead while every test passed, because
// the tests injected a pre-filled set and so began one hop after the gap.
//
// Load is idempotent, so calling this per render costs one map read after the
// first. Colour off returns nil rather than a loaded set: with no style to
// inject there is nothing to show, and reading the whole deck for it would be IO
// for a disabled feature.
func vocabularyFor(d deps, opt options) Vocabulary {
	if d.vocab == nil || !opt.color {
		return nil
	}
	d.vocab.Load()
	return d.vocab
}

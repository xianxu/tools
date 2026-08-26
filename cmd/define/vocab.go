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
// Mutex-guarded because the two accesses are genuinely concurrent: the capture
// path Adds a word as a lookup completes while the editor's render reads.
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
	// Counted with wordRuns, not strings.Fields, so the set agrees with the
	// tokenizer that will look it up. They diverge for any key holding other
	// punctuation — `e.g.`, `9/11` — and such a key is currently UNMATCHABLE
	// whichever way it is counted, because phraseGap allows only spaces and tabs
	// between a phrase's tokens. TestAPunctuatedKeyIsNotMatchable pins that as
	// known behaviour rather than leaving M2 to rediscover it.
	if n := len(wordRuns(key)); n > v.maxWords {
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
	if v.loaded || v.st == nil {
		return
	}
	v.loaded = true
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
	if v.warn == nil {
		return
	}
	fmt.Fprintf(v.warn, "define: "+format+"\n", args...)
}

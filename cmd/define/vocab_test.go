package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
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

// BR-2: "must not claim a word the deck rejected" was a comment with nothing
// behind it — moving the Add above the Upsert error check survived the suite.
//
// The double has to fail ONLY the deck write. failingStore fails AppendEvent
// too, so Capture returns before it ever reaches the deck, and the first version
// of this test passed while the mutant lived — it was asserting "the event log
// failed", not "the deck rejected the word".
func TestCaptureDoesNotAddAWordTheDeckRejected(t *testing.T) {
	v := &memVocabulary{}
	c := newStoreCapturer(deckRejects{Store: store.NewMem()}, store.FixedClock(aDay), nil, v)

	c.Capture("sycophantic", true, options{})

	if v.Has(store.Key("sycophantic")) {
		t.Error("a word the deck rejected entered the highlight set")
	}
}

// deckRejects accepts events and refuses deck writes — the state that isolates
// "the deck said no" from "the store is broken".
type deckRejects struct{ store.Store }

func (deckRejects) Upsert(store.Word) error { return errors.New("deck full") }

// The `loaded` guard survived removal, because Add is idempotent and maxWords is
// a max — the old assertion could not fail. Count the reads instead.
func TestStoreVocabularyReadsTheDeckOnlyOnce(t *testing.T) {
	st := &countingDeck{Store: store.NewMem()}
	if err := st.Upsert(store.Word{Text: "obsequious"}); err != nil {
		t.Fatal(err)
	}
	v := newStoreVocabulary(st, nil)

	v.Load()
	v.Load()

	if st.reads != 1 {
		t.Errorf("Deck() read %d times, want exactly 1 — Prefix-rate work must not reach the disk", st.reads)
	}
}

type countingDeck struct {
	store.Store
	reads int
}

func (c *countingDeck) Deck() ([]store.Word, error) {
	c.reads++
	return c.Store.Deck()
}

// Hop 2 of the Vocabulary production chain: withStore merging sd.vocab into
// deps.vocab. Deleting that merge kills the feature outright — the capturer
// would hold openStore's real set while the renderer read a fresh empty one — and
// the entire suite stayed green, because both loop tests set rig.deps.vocab
// directly and so begin AFTER this hop.
//
// The pattern is the one BR-31 left behind at command_test.go:137: build the
// deps a production entry point actually builds, then assert across the seam.
func TestWithStoreCarriesTheHighlightSetThrough(t *testing.T) {
	t.Chdir(t.TempDir())

	d := deps{newStore: openStore}.withStore(options{}, io.Discard)
	d.capture.Capture("obsequious", true, options{})

	if d.vocab == nil {
		t.Fatal("withStore left deps.vocab nil")
	}
	if !d.vocab.Has(store.Key("obsequious")) {
		t.Error("withStore handed the renderer a different set than the capturer got")
	}
}

// EVERY process entry path that can render a definition, each driven through
// production wiring with an UNLOADED store vocabulary.
//
// The enumeration is the point. M2 shipped highlighting into lookupAndRender
// while Load() lived in runEditor, so `define <word>` and piped stdin rendered
// against an empty set — two of three paths dead, with the whole suite green,
// because every test injected a pre-filled memVocabulary and so began one hop
// after the gap. A test that starts at the dependency it injects can never see
// the thing that fills it.
//
// A new entry path that renders belongs in this table.
func TestEveryEntryPathHighlightsDefinitions(t *testing.T) {
	// sycophantic's gloss contains "obsequious" — the issue's motivating case.
	deck := func(t *testing.T) Vocabulary {
		t.Helper()
		st := store.NewMem()
		if err := st.Upsert(store.Word{Text: "obsequious"}); err != nil {
			t.Fatal(err)
		}
		return newStoreVocabulary(st, nil) // deliberately NOT loaded
	}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T, d deps, opt options, out, errb *bytes.Buffer)
	}{
		{"one-shot: define <word>", func(t *testing.T, d deps, opt options, out, errb *bytes.Buffer) {
			defineOnce(t.Context(), d, opt, replCommand{kind: cmdDefine, word: "sycophantic"}, out, errb)
		}},
		{"piped stdin", func(t *testing.T, d deps, opt options, out, errb *bytes.Buffer) {
			replLines(t.Context(), nil, d, opt, strings.NewReader("sycophantic\n"), out, errb, true, false)
		}},
		{"raw editor", func(t *testing.T, d deps, opt options, out, errb *bytes.Buffer) {
			runEditor(t.Context(), scriptKeys("sycophantic\r"), nil, d, opt,
				paintInto(out), nil, func() {}, out, errb)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rig, opt, _ := editorRig(t, "sycophantic", true)
			rig.deps.vocab = deck(t)
			opt.noAudio = true
			var out, errb bytes.Buffer

			tc.run(t, rig.deps, opt, &out, &errb)

			if !strings.Contains(out.String(), "\x1b[1;32mobsequious") {
				t.Errorf("no highlight on this entry path: %q", out.String())
			}
		})
	}
}

// BR-24(3): the colour gate's only effect is the ABSENCE of work, so no output
// assertion can see it — deleting `!opt.color` passed the whole suite while
// reading the entire deck under -no-color. That was M1 round 3's finding,
// unpinned again by M2's refactor moving the check.
//
// A guard whose effect is absence needs a counting double, not an output check.
func TestNoColourReadsNoDeck(t *testing.T) {
	st := &countingDeck{Store: store.NewMem()}
	d := deps{langDeps: langDeps{vocab: newStoreVocabulary(st, nil)}}

	if got := vocabularyFor(d, options{color: false}); got != nil {
		t.Error("colour off returned a vocabulary; nothing can render it")
	}
	if st.reads != 0 {
		t.Errorf("read the deck %d times with colour off — IO for a disabled feature", st.reads)
	}

	if got := vocabularyFor(d, options{color: true}); got == nil {
		t.Fatal("colour on returned no vocabulary")
	}
	if st.reads != 1 {
		t.Errorf("read the deck %d times with colour on, want exactly 1", st.reads)
	}
}

// BR-39: `loaded` was unguarded while vocabularyFor calls Load on every render,
// so a second rendering goroutine raced it. Nothing in the package could have
// caught that — every existing -race run drives one goroutine.
func TestVocabularyIsSafeUnderConcurrency(t *testing.T) {
	st := store.NewMem()
	if err := st.Upsert(store.Word{Text: "obsequious"}); err != nil {
		t.Fatal(err)
	}
	v := newStoreVocabulary(st, nil)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v.Load()
			v.Add(fmt.Sprintf("word%d", i))
			_ = v.Has(store.Key("obsequious"))
			_ = v.MaxPhraseWords()
		}(i)
	}
	wg.Wait()

	if !v.Has(store.Key("obsequious")) {
		t.Error("the deck word did not survive concurrent access")
	}
}

// BR-38: an unmatchable key must not widen the window every stream pays for.
//
// maxWords is the only input to the writer's hold arithmetic, so a key that can
// never match still costs every stream a wider held tail. Measured on "the quick
// brown fox jumps": a single-word deck holds 5 bytes, and adding the unmatchable
// `e.g.` held 9 — the same as a real three-token phrase.
func TestAnUnmatchableKeyDoesNotWidenTheHoldWindow(t *testing.T) {
	v := &memVocabulary{}
	v.Add("obsequious")

	// Two tokens by wordRuns, but the gap is a period, so they can never rejoin.
	v.Add("e.g.")
	if got := v.MaxPhraseWords(); got != 1 {
		t.Errorf("after an unmatchable key, MaxPhraseWords = %d, want 1", got)
	}

	// Subtler, and the reason the check is phraseRunsJoin rather than a
	// punctuation test: apostrophes are word characters, so wordRuns sees THREE
	// tokens here — but they are trimmed off the token edges, leaving gaps that
	// carry an apostrophe, which phraseGap refuses. Unmatchable too.
	v.Add("rock 'n' roll")
	if got := v.MaxPhraseWords(); got != 1 {
		t.Errorf("after an apostrophe-gapped key, MaxPhraseWords = %d, want 1", got)
	}

	// A real three-token phrase: plain spaces, so it can match and must count.
	v.Add("in spite of")
	if got := v.MaxPhraseWords(); got != 3 {
		t.Errorf("MaxPhraseWords = %d, want 3 from the phrase that CAN match", got)
	}
}

// The class BR-39 named, swept: storeHistory has the identical shape — a bare
// `loaded` beside a mutex that Add and Prefix both take — and races under the
// same driver. Fixing only the vocabulary would have been the instance.
func TestHistoryIsSafeUnderConcurrency(t *testing.T) {
	st := store.NewMem()
	if err := st.AppendEvent(store.ReviewEvent{Word: "obsequious", Kind: store.EventLookedUp, Found: true, At: aDay}); err != nil {
		t.Fatal(err)
	}
	h := newStoreHistory(st, nil)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			h.Load()
			h.Add(fmt.Sprintf("word%d", i))
			_ = h.Prefix("")
		}(i)
	}
	wg.Wait()

	if got := h.Prefix("obsequious"); len(got) == 0 {
		t.Error("the loaded entry did not survive concurrent access")
	}
}

// The same wiring assertion as above, AFTER a mid-session /lang — plus the half
// that a deps swap cannot reach.
//
// runEditor resolves the highlight set into a LOCAL before its loop starts, so
// reassigning d alone would leave the editor highlighting the previous
// language's words while every other path had moved on. sessionSetLang takes
// &voc for exactly that reason, and the `notWant` assertions below are what
// redden if it stops.
func TestLangSwitchKeepsOneHighlightSetAndItIsTheNewLanguages(t *testing.T) {
	dir := t.TempDir()
	for _, seed := range []struct {
		lang store.Lang
		word string
	}{{"en", "sycophantic"}, {"es", "madrugar"}} {
		if err := store.NewYAML(dir, seed.lang, nil).Upsert(store.Word{Text: seed.word}); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)

	var warn bytes.Buffer
	opt := options{color: true}
	d := deps{newStore: openStore}.withStore(opt, &warn)
	voc := vocabularyFor(d, opt)

	if d.lang != "en" {
		t.Fatalf("started in %q, want the default", d.lang)
	}
	if !voc.Has(store.Key("sycophantic")) || voc.Has(store.Key("madrugar")) {
		t.Fatal("the English session does not start from the English deck")
	}

	beforeHistory := d.history
	setLang := sessionSetLang(&d, &opt, d.persistLang, &voc, &warn)
	if setLang == nil {
		t.Fatal("no setLang in a directory that has a store")
	}
	if err := setLang("es"); err != nil {
		t.Fatal(err)
	}

	if d.lang != "es" {
		t.Errorf("d.lang = %q after the switch", d.lang)
	}
	// The editor's cached set followed the switch.
	if !voc.Has(store.Key("madrugar")) {
		t.Error("the editor's highlight set did not follow /lang: it cannot see the Spanish deck")
	}
	if voc.Has(store.Key("sycophantic")) {
		t.Error("the editor's highlight set is still the English one after /lang es")
	}
	// And the capturer and the renderer are still ONE set — the original
	// invariant, which a rebuild is exactly the thing that could break.
	d.capture.Capture("bonito", true, opt)
	if !d.vocab.Has(store.Key("bonito")) {
		t.Error("after /lang, the capturer and the renderer hold different sets")
	}
	if !voc.Has(store.Key("bonito")) {
		t.Error("after /lang, the EDITOR holds a third set that captures do not reach")
	}
	// D6's news gate must follow the switch too. It did not: newsFeedFor was
	// applied only at the boundary, so a session that started in English kept the
	// English feed after /lang es, and one that started in Spanish kept news==nil
	// forever after /lang en. That is why langDeps is a STRUCT taken whole from
	// one builder rather than a list of members in a comment.
	if bs, ok := d.usage.(*bothSources); !ok || bs.news != nil {
		t.Errorf("after /lang es the session still holds the English news feed (%T); the feed "+
			"is English by construction and must not be consulted", d.usage)
	}

	// history keeps its IDENTITY across the switch, which applyLang argues for in
	// a comment and this makes structural: events/ is not language-scoped, and
	// rebuilding it would leave the raw editor holding an orphaned, already-
	// Load()ed History while everything else read a fresh empty one.
	if d.history != beforeHistory {
		t.Error("the switch rebuilt history; events/ is not language-scoped and the editor " +
			"already holds the loaded one")
	}
	// The switch is durable, not just live.
	if got := store.ReadLang(dir); got != "es" {
		t.Errorf("the directory says %q; /lang must persist", got)
	}

	// And back, because the reverse loses the feed silently rather than gaining
	// a wrong one — the harder direction to notice.
	if err := setLang(store.DefaultLang); err != nil {
		t.Fatal(err)
	}
	if bs, ok := d.usage.(*bothSources); !ok || bs.news == nil {
		t.Errorf("switching back to English did not restore the news feed (%T)", d.usage)
	}
}

// No directory, no setLang — which is what makes /lang report honestly instead
// of accepting a switch it cannot keep.
func TestSessionSetLangIsNilWithNowhereToPersist(t *testing.T) {
	d := deps{}
	if got := sessionSetLang(&d, &options{}, nil, nil, nil); got != nil {
		t.Error("built a session switch with no directory to persist to")
	}
}

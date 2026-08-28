package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// seedTwoLanguageDeck files one English and one Spanish word, both long enough
// ago to be due for review.
func seedTwoLanguageDeck(t *testing.T, dir string) {
	t.Helper()
	long := time.Now().AddDate(0, 0, -60)
	for _, seed := range []struct {
		lang store.Lang
		word string
	}{{"en", "sycophantic"}, {"es", "madrugar"}} {
		st := store.NewYAML(dir, seed.lang, nil)
		if err := st.Upsert(store.Word{
			Text: seed.word, FirstSeen: long, LastSeen: long, Lookups: 1,
		}); err != nil {
			t.Fatal(err)
		}
	}
}

// --forget acts on the CURRENT language's deck, and — the assertion that
// matters — cannot reach the other one. A delete that lands in the wrong deck is
// the failure mode worth pinning: the deck is the one artifact here that cannot
// be regenerated, and #forget deliberately fails loudly rather than silently.
func TestForgetActsOnTheCurrentLanguageOnly(t *testing.T) {
	dir := t.TempDir()
	d := testDeps(t)
	d.newStore = openStore
	seedTwoLanguageDeck(t, dir)
	t.Chdir(dir)

	stillThere := func(lang store.Lang, word string) bool {
		t.Helper()
		deck, err := store.NewYAML(dir, lang, nil).Deck()
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range deck {
			if w.Text == word {
				return true
			}
		}
		return false
	}

	// An English word is NOT reachable from a Spanish session, even by name.
	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-lang", "es", "-forget", "sycophantic"}, d,
		strings.NewReader(""), &out, &errb); code == 0 {
		t.Error("-forget reached across languages and reported success")
	}
	if !stillThere("en", "sycophantic") {
		t.Fatal("a Spanish -forget deleted an English word")
	}

	// And the word that IS in this language goes.
	out.Reset()
	errb.Reset()
	if code := run(t.Context(), []string{"-lang", "es", "-forget", "madrugar"}, d,
		strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("-forget failed on a word in the current language: %s", errb.String())
	}
	if stillThere("es", "madrugar") {
		t.Error("madrugar survived -forget in its own language")
	}
	if !stillThere("en", "sycophantic") {
		t.Error("forgetting a Spanish word removed an English one")
	}
}

// A review session offers one language's words. #5's schedule interleaves the
// deck by due-date, so a shared namespace would not merely ALLOW a mixed sitting
// — it would produce one, which is what the operator asked to stop.
//
// Asserted at todaysQuestions rather than through run(): --play refuses without
// a terminal, so a run()-level test would pass while asserting nothing about
// which words were offered. This is the function that decides.
func TestPlayReviewsTheCurrentLanguageOnly(t *testing.T) {
	for _, tc := range []struct {
		name    string
		lang    store.Lang
		want    string
		notWant string
	}{
		{name: "Spanish", lang: "es", want: "madrugar", notWant: "sycophantic"},
		{name: "the default is English", lang: "", want: "sycophantic", notWant: "madrugar"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			dict := testDict(t) // relative fixture path: build before chdir
			seedTwoLanguageDeck(t, dir)
			t.Chdir(dir)

			var warn bytes.Buffer
			opt := options{count: 20, lang: tc.lang}
			d := deps{newStore: openStore, dict: dict}.withStore(opt, &warn)

			// Everything this session CONSIDERED: the questions it built, plus the
			// words it named as skipped. Both halves are needed because the deck
			// is scoped from M1 while the dictionary is not scoped until M2 — so
			// today a Spanish word is correctly SELECTED and then skipped for
			// having no entry, and asserting only on questions would read that as
			// "the session offered nothing" rather than "the scoping worked".
			qs, _ := todaysQuestions(d, opt, &warn, &warn)
			considered := warn.String()
			for _, q := range qs {
				considered += " " + q.Word()
			}
			if !strings.Contains(considered, tc.want) {
				t.Errorf("a %s session never considered %q:\n%s", tc.name, tc.want, considered)
			}
			if strings.Contains(considered, tc.notWant) {
				t.Errorf("a %s session considered %q, a word from the other deck:\n%s",
					tc.name, tc.notWant, considered)
			}
		})
	}
}

// The highlight set follows the mode too: a Spanish session must not paint
// English words in a definition, and vice versa. Same seam as the deck, because
// #21 made the vocabulary the ONE predicate every highlight decision goes
// through — so this is proof that the seam was the right place, not a new rule.
func TestTheHighlightSetFollowsTheMode(t *testing.T) {
	dir := t.TempDir()
	seedTwoLanguageDeck(t, dir)
	t.Chdir(dir)

	for _, tc := range []struct {
		lang    store.Lang
		want    string
		notWant string
	}{
		{lang: "es", want: "madrugar", notWant: "sycophantic"},
		{lang: "en", want: "sycophantic", notWant: "madrugar"},
	} {
		var warn bytes.Buffer
		opt := options{color: true, lang: tc.lang}
		d := deps{newStore: openStore}.withStore(opt, &warn)
		voc := vocabularyFor(d, opt)
		if voc == nil {
			t.Fatalf("%s: no highlight set", tc.lang)
		}
		if !voc.Has(store.Key(tc.want)) {
			t.Errorf("%s session does not highlight %q", tc.lang, tc.want)
		}
		if voc.Has(store.Key(tc.notWant)) {
			t.Errorf("%s session highlights %q, a word from the other deck", tc.lang, tc.notWant)
		}
	}
}

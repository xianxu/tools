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

// C1, the boundary review's Critical: /lang must re-derive the VOICE, not only
// the deck.
//
// This is asserted at the level the bug lived at — drive `/lang es` through
// run() and look at what the CDN was actually asked for. A unit test on voiceFor
// cannot see it, because voiceFor was always right; what was wrong is that
// nothing called it again after the switch. The session's deck went Spanish
// while its pronunciation stayed English, including the two legacy
// /sounds/oxford/ URLs that had just been gated to English for costing ~450ms
// per guaranteed miss.
//
// The general rule this pins: anything derived from the language BEFORE a switch
// must be re-derived BY it. applyLang owns that enumeration.
func TestLangSwitchReDerivesEverythingDownstreamOfTheLanguage(t *testing.T) {
	dir := t.TempDir()
	cdn := newFakeCDN(t, nil) // every URL 404s: we are watching what is ASKED for
	d := deps{
		dict:     testDict(t),
		audio:    &rebasedSource{cdn: cdn},
		player:   &fakePlayer{},
		newStore: openStore,
	}
	t.Chdir(dir)

	var out, errb bytes.Buffer
	if code := run(t.Context(), nil, d,
		strings.NewReader("/lang es\nsycophantic\n"), &out, &errb); code != 0 {
		t.Logf("exit %d (a missing recording is not a failed lookup): %s", code, errb.String())
	}

	for _, path := range cdn.Requested() {
		if strings.Contains(path, "_en_") {
			t.Errorf("after /lang es the session still asked for an ENGLISH recording: %s", path)
		}
		if strings.Contains(path, "/sounds/oxford/") {
			t.Errorf("after /lang es the session paid for the English-only legacy path: %s", path)
		}
	}
	// And it did ask for the Spanish one, so the assertions above are not
	// vacuously true of a session that fetched nothing at all.
	var sawSpanish bool
	for _, path := range cdn.Requested() {
		if strings.Contains(path, "_es_es_") {
			sawSpanish = true
		}
	}
	if !sawSpanish {
		t.Errorf("the session requested no Spanish recording at all: %v", cdn.Requested())
	}
}

// I1: the persisted setting is READ by a one-shot run — the Done-when row
// "a one-shot `define madrugar` uses the persisted language, with no session to
// inherit from" was ticked while every test either wrote the setting or passed
// -lang, so replacing ReadLang with DefaultLang left the suite green.
func TestAOneShotLookupReadsThePersistedLanguage(t *testing.T) {
	for _, tc := range []struct {
		name      string
		args      []string
		wantLang  store.Lang
		emptyLang store.Lang
	}{
		{name: "the persisted setting is used", args: []string{"sycophantic"},
			wantLang: "es", emptyLang: "en"},
		// The precedence, not just the read: the flag still wins for one run.
		{name: "-lang overrides it", args: []string{"-lang", "en", "sycophantic"},
			wantLang: "en", emptyLang: "es"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			// NOT testDeps: it pins a noopCapturer, and withStore only fills nil
			// fields — so the real capturer never runs and the deck stays empty
			// whatever the language is. That would make this test pass for the
			// wrong reason in the opposite direction.
			d := deps{
				dict: testDict(t), audio: noAudioSource{}, player: &fakePlayer{},
				newStore: openStore,
			}
			if err := store.WriteLang(dir, "es"); err != nil {
				t.Fatal(err)
			}
			t.Chdir(dir)

			var out, errb bytes.Buffer
			run(t.Context(), tc.args, d, strings.NewReader(""), &out, &errb)

			deck, err := store.NewYAML(dir, tc.wantLang, nil).Deck()
			if err != nil {
				t.Fatal(err)
			}
			if len(deck) != 1 || deck[0].Text != "sycophantic" {
				t.Errorf("the word did not land in words/%s/: %+v", tc.wantLang, deck)
			}
			// The negative half: the positive one alone survives a DOUBLE write,
			// which is exactly what a precedence bug looks like.
			other, err := store.NewYAML(dir, tc.emptyLang, nil).Deck()
			if err != nil {
				t.Fatal(err)
			}
			if len(other) != 0 {
				t.Errorf("the word also landed in words/%s/: %+v", tc.emptyLang, other)
			}
		})
	}
}

// D6: the news feed is English BY CONSTRUCTION (httpFeed hardcodes
// hl=en-US&gl=US&ceid=US:en), so it is consulted only for English.
//
// Decided in M2's plan rather than discovered at M2's boundary, which is the
// lesson the learner model taught at M1's. A Spanish session asking it about
// `mesa` gets English news about a landform or a city in Arizona — the wrong
// language AND the wrong sense — cached under a key an English session shares.
// Same rule as the dictionary: no data beats the wrong language's data.
func TestTheNewsFeedIsConsultedOnlyForTheLanguageItServes(t *testing.T) {
	inner := newCachingFeed(newHTTPFeed(), store.NewMem(), store.SystemClock())
	if got := newsFeedFor(store.DefaultLang, inner); got != inner {
		t.Error("English lost its news feed; the feed serves exactly this language")
	}
	for _, lang := range []store.Lang{"es", "fr", "de"} {
		if got := newsFeedFor(lang, inner); got != nil {
			t.Errorf("%s consults the English news feed; its examples must come from its "+
				"own dictionary entry instead", lang)
		}
	}
}

// And the seam above it tolerates that, which is what makes the gate safe: a
// session with no feed still gets the dictionary's own usage examples, which is
// exactly what M2 makes correct per language.
func TestUsagesSurviveWithoutANewsFeed(t *testing.T) {
	b := &bothSources{news: nil}
	got := b.Usages(t.Context(), "madrugar", Entry{})
	if got == nil && len(got) != 0 {
		t.Error("a nil feed broke the usage seam")
	}
}

// Package storetest holds the conformance suite every Store implementation must
// pass.
//
// It exists so "the in-memory store behaves like the YAML one" is a test rather
// than an assumption — the gap that a fake diverging from its real counterpart
// silently creates.
package storetest

import (
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/xianxu/tools/cmd/define/store"
)

// Suite runs every Store obligation against a freshly constructed store.
func Suite(t *testing.T, newStore func(t *testing.T) store.Store) {
	t.Helper()
	day := func(n int) time.Time {
		return time.Date(2026, 8, n, 12, 0, 0, 0, time.UTC)
	}
	// Through the parse rather than a cast: there are deliberately no per-label
	// constants — the domain set is a data table read off NOAD's prose, and
	// ParseDomain is the only way in. Using it here means these rows also fail if
	// the label ever leaves the closed set.
	domain := func(t *testing.T, name string) store.Domain {
		t.Helper()
		d, ok := store.ParseDomain(name)
		if !ok {
			t.Fatalf("ParseDomain(%q) refused; it must be in the closed set", name)
		}
		return d
	}

	t.Run("empty store is empty, not an error", func(t *testing.T) {
		s := newStore(t)
		deck, err := s.Deck()
		if err != nil {
			t.Fatalf("Deck: %v", err)
		}
		if len(deck) != 0 {
			t.Errorf("deck = %v, want empty", deck)
		}
		ev, err := s.Events(time.Time{})
		if err != nil || len(ev) != 0 {
			t.Errorf("Events = %v, %v, want empty", ev, err)
		}
	})

	t.Run("the user model round-trips, and is empty before anything writes one", func(t *testing.T) {
		// Absent reads as "" and NOT an error: the normal first-run state for the
		// whole of #16's life, since #17 is what writes it.
		//
		// The write half is here because a row that only reads asserts the only
		// value a store with no setter can produce — unfalsifiable for the
		// reference implementation, and a fake that cannot hold the real one's
		// state is the gap this suite exists to close.
		s := newStore(t)
		got, err := s.UserModel()
		if err != nil {
			t.Fatalf("UserModel: %v", err)
		}
		if got != "" {
			t.Errorf("UserModel = %q, want empty", got)
		}

		const model = "## Level\nC1, reads judicial opinions.\n\n## Corrections\nHuman-owned.\n"
		if err := s.SetUserModel(model); err != nil {
			t.Fatalf("SetUserModel: %v", err)
		}
		got, err = s.UserModel()
		if err != nil {
			t.Fatalf("UserModel after write: %v", err)
		}
		if got != model {
			t.Errorf("UserModel = %q, want %q", got, model)
		}
	})

	t.Run("an asked event round-trips with its question", func(t *testing.T) {
		s := newStore(t)
		// Word is the word the question FOLLOWED, and may be empty — a question
		// asked cold has no word. Question is what #17 reads.
		e := store.ReviewEvent{
			Word: "sycophantic", Kind: store.EventAsked,
			Question: "what's the difference to obsequious?", At: day(2),
		}
		if err := s.AppendEvent(e); err != nil {
			t.Fatalf("AppendEvent: %v", err)
		}
		got, err := s.Events(time.Time{})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("Events = %v, want 1", got)
		}
		if got[0].Question != e.Question || got[0].Kind != store.EventAsked || got[0].Word != e.Word {
			t.Errorf("round-tripped %+v, want %+v", got[0], e)
		}
	})

	t.Run("a question with no word is still a whole record", func(t *testing.T) {
		// Completeness generalised from "has a word" to "has a SUBJECT" when the
		// asked event arrived. A question asked before any lookup has no word,
		// and dropping it at read time would lose exactly the events #17 wants.
		s := newStore(t)
		e := store.ReviewEvent{Kind: store.EventAsked, Question: "why is it pejorative?", At: day(3)}
		if err := s.AppendEvent(e); err != nil {
			t.Fatalf("AppendEvent: %v", err)
		}
		got, err := s.Events(time.Time{})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if len(got) != 1 || got[0].Question != e.Question {
			t.Errorf("Events = %v, want the wordless question to survive", got)
		}
	})

	t.Run("a question cannot break the record boundary", func(t *testing.T) {
		// `question:` is the FIRST free-form user text this log has ever held —
		// every value before it was a single dictionary headword. The reader's
		// record boundary is a literal top-level "- ", and the only thing keeping
		// user text off column 0 is the writer's quoting. Nothing pinned that.
		//
		// A regression corrupts an append-only log #17 folds over, irreversibly,
		// and the precedent for "a fragment that looks whole" is already in this
		// file's torn-record rule.
		hostile := []string{
			"what about\na newline?",
			"- word: injected\n  kind: looked-up\n  found: true\n  at: 2026-08-20T09:00:00Z\n",
			"why does it say at: here?",
			"- not a record, but it starts like one",
			"trailing colon: and a #comment, plus \"quotes\" and 'apostrophes'",
		}
		s := newStore(t)
		for i, q := range hostile {
			if err := s.AppendEvent(store.ReviewEvent{
				Kind: store.EventAsked, Question: q, At: day(1).Add(time.Duration(i) * time.Minute),
			}); err != nil {
				t.Fatalf("AppendEvent(%q): %v", q, err)
			}
		}
		got, err := s.Events(time.Time{})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if len(got) != len(hostile) {
			t.Fatalf("got %d events, want %d — a question forged or broke a record boundary: %+v",
				len(got), len(hostile), got)
		}
		for i, q := range hostile {
			if got[i].Question != q {
				t.Errorf("event %d question = %q, want %q", i, got[i].Question, q)
			}
			if got[i].Word != "" || got[i].Kind != store.EventAsked {
				t.Errorf("event %d = %+v — a forged record leaked through", i, got[i])
			}
		}
	})

	t.Run("upsert round-trips", func(t *testing.T) {
		s := newStore(t)
		w := store.Word{Text: "sycophantic", FirstSeen: day(1), LastSeen: day(1), Lookups: 1}
		if err := s.Upsert(w); err != nil {
			t.Fatal(err)
		}
		deck, err := s.Deck()
		if err != nil {
			t.Fatal(err)
		}
		if len(deck) != 1 || deck[0].Text != "sycophantic" {
			t.Fatalf("deck = %+v", deck)
		}
		if !deck[0].FirstSeen.Equal(day(1)) || !deck[0].LastSeen.Equal(day(1)) {
			t.Errorf("times did not survive: %+v", deck[0])
		}
	})

	t.Run("second upsert merges, does not duplicate", func(t *testing.T) {
		s := newStore(t)
		_ = s.Upsert(store.Word{Text: "sycophantic", FirstSeen: day(1), LastSeen: day(1)})
		_ = s.Upsert(store.Word{Text: "Sycophantic", FirstSeen: day(3), LastSeen: day(3)})

		deck, _ := s.Deck()
		if len(deck) != 1 {
			t.Fatalf("case variants produced %d entries, want 1: %+v", len(deck), deck)
		}
		if !deck[0].FirstSeen.Equal(day(1)) {
			t.Errorf("FirstSeen = %v, want the earlier sighting kept", deck[0].FirstSeen)
		}
		if !deck[0].LastSeen.Equal(day(3)) {
			t.Errorf("LastSeen = %v, want the later sighting", deck[0].LastSeen)
		}
		if deck[0].Lookups != 2 {
			t.Errorf("Lookups = %d, want 2", deck[0].Lookups)
		}
	})

	t.Run("deck is newest first", func(t *testing.T) {
		s := newStore(t)
		_ = s.Upsert(store.Word{Text: "older", LastSeen: day(1)})
		_ = s.Upsert(store.Word{Text: "newer", LastSeen: day(5)})
		deck, _ := s.Deck()
		if len(deck) != 2 || deck[0].Text != "newer" {
			t.Errorf("deck = %+v, want newer first", deck)
		}
	})

	t.Run("events filter by time and stay chronological", func(t *testing.T) {
		s := newStore(t)
		for _, n := range []int{3, 1, 5} {
			if err := s.AppendEvent(store.ReviewEvent{
				Word: "w", Kind: store.EventLookedUp, Found: true, At: day(n),
			}); err != nil {
				t.Fatal(err)
			}
		}
		all, err := s.Events(time.Time{})
		if err != nil {
			t.Fatal(err)
		}
		if len(all) != 3 {
			t.Fatalf("got %d events, want 3", len(all))
		}
		for i := 1; i < len(all); i++ {
			if all[i].At.Before(all[i-1].At) {
				t.Errorf("events out of order: %v", all)
			}
		}
		since, _ := s.Events(day(3))
		if len(since) != 2 {
			t.Errorf("Events(day 3) returned %d, want 2", len(since))
		}
	})

	t.Run("forget removes a word and is a no-op when absent", func(t *testing.T) {
		s := newStore(t)
		_ = s.Upsert(store.Word{Text: "sycophantic", LastSeen: day(1)})
		_ = s.AppendEvent(store.ReviewEvent{Word: "sycophantic", Kind: store.EventLookedUp, Found: true, At: day(1)})

		removed, err := s.Forget("Sycophantic") // case-insensitive, like every other key
		if err != nil || !removed {
			t.Fatalf("Forget = %v, %v; want true, nil", removed, err)
		}
		if deck, _ := s.Deck(); len(deck) != 0 {
			t.Errorf("deck = %+v, want empty", deck)
		}
		// The log is history and must survive: #8's statistics are a fold over it.
		if ev, _ := s.Events(time.Time{}); len(ev) != 1 {
			t.Errorf("Forget deleted %d event(s); the log must be untouched", 1-len(ev))
		}
		// Absence is not an error.
		if removed, err := s.Forget("never-seen"); err != nil || removed {
			t.Errorf("Forget(absent) = %v, %v; want false, nil", removed, err)
		}
		if removed, err := s.Forget(""); err != nil || removed {
			t.Errorf("Forget(empty) = %v, %v; want false, nil — never a wildcard", removed, err)
		}
	})

	t.Run("a flagged event round-trips with its options, and is NOT a review", func(t *testing.T) {
		// In the SUITE because it is a Store-surface promise, and because the
		// property that matters is negative: a flag must not be readable as a
		// review. Fold folds every EventReviewed and GradeOf(false) is
		// GradeWrong, so a flag that arrived as one would demote the word on an
		// append-only log.
		s := newStore(t)
		if err := s.AppendEvent(store.ReviewEvent{
			Word: "sycophantic", Kind: store.EventFlagged, Found: true,
			Options: []string{"sycophantic", "ephemeral", "keel"}, At: day(1),
		}); err != nil {
			t.Fatalf("AppendEvent: %v", err)
		}
		got, err := s.Events(time.Time{})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d events, want 1", len(got))
		}
		if got[0].Kind != store.EventFlagged {
			t.Errorf("kind = %q, want %q — a flag read back as a review would demote the word",
				got[0].Kind, store.EventFlagged)
		}
		// THE OPTIONS ARE THE EVIDENCE. A flag naming none is "something was
		// wrong once" — the deliberate opposite of Missed, which records only the
		// axis because the option set will not exist later.
		if len(got[0].Options) != 3 {
			t.Errorf("options = %v, want the whole set the learner saw", got[0].Options)
		}
		// And it carries NO verdict: Correct is the zero value and stays there.
		if got[0].Correct {
			t.Error("a flagged event carries a verdict; the ladder would move on a broken question")
		}
	})

	t.Run("forget removes everything the word OWNS, and no events", func(t *testing.T) {
		// The bug this pins: Forget removed the deck entry only, so a forgotten
		// word kept its cached band, domain and authored items. Looking it up
		// again re-added it to the deck while --harvest, seeing facts already
		// harvested and items already present, SKIPPED it — so "forget this word,
		// its material is bad" was the one thing forgetting could not do.
		s := newStore(t)
		if err := s.Upsert(store.Word{Text: "sycophantic", FirstSeen: day(1), LastSeen: day(1), Lookups: 1}); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
		if err := s.SetWordFacts("sycophantic", store.WordFacts{
			Band: store.C2, Domain: store.DomainGeneral, At: day(1),
		}); err != nil {
			t.Fatalf("SetWordFacts: %v", err)
		}
		if err := s.SetItems("sycophantic", []store.Item{
			{Word: "sycophantic", Form: store.FormCloze, Stem: "a bad stem", Answer: "sycophantic", At: day(1)},
		}); err != nil {
			t.Fatalf("SetItems: %v", err)
		}
		if err := s.SetNewsItems("sycophantic", []store.NewsItem{{Title: "x"}}, day(1)); err != nil {
			t.Fatalf("SetNewsItems: %v", err)
		}
		if err := s.AppendEvent(store.ReviewEvent{
			Word: "sycophantic", Kind: store.EventLookedUp, Found: true, At: day(1),
		}); err != nil {
			t.Fatalf("AppendEvent: %v", err)
		}

		removed, err := s.Forget("sycophantic")
		if err != nil {
			t.Fatalf("Forget: %v", err)
		}
		if !removed {
			t.Error("Forget reported nothing removed for a word in the deck")
		}

		if f, err := s.WordFacts("sycophantic"); err != nil {
			t.Fatalf("WordFacts: %v", err)
		} else if f.Harvested() {
			t.Error("a forgotten word kept its band and domain; --harvest will skip it as done")
		}
		if items, err := s.Items("sycophantic"); err != nil {
			t.Fatalf("Items: %v", err)
		} else if len(items) != 0 {
			t.Errorf("a forgotten word kept %d authored item(s); its bad material is unregenerable", len(items))
		}
		if _, at, err := s.NewsItems("sycophantic"); err != nil {
			t.Fatalf("NewsItems: %v", err)
		} else if !at.IsZero() {
			t.Error("a forgotten word kept its news cache")
		}

		// EVENTS STAY. The deck is a working set, the log is history, and
		// rewriting the past would corrupt every statistic derived from it.
		ev, err := s.Events(time.Time{})
		if err != nil {
			t.Fatalf("Events: %v", err)
		}
		if len(ev) != 1 {
			t.Errorf("got %d events after Forget, want the 1 that was there — history is not "+
				"the deck's to rewrite", len(ev))
		}
	})

	t.Run("forget cannot escape the words directory", func(t *testing.T) {
		s := newStore(t)
		_ = s.Upsert(store.Word{Text: "sycophantic", LastSeen: day(1)})
		// An end-to-end net, NOT the assertion of the traversal guard: Slug
		// sanitises first, so this passes whether or not wordFileName's guard
		// exists (measured). The guard is pinned by its own unit test; this is
		// here so a Store implementation that derived filenames some other way
		// would be caught.
		for _, key := range []string{
			"../../../etc/passwd", "/etc/passwd", "..", ".", "../sycophantic",
		} {
			removed, err := s.Forget(key)
			if removed {
				t.Errorf("Forget(%q) reported a removal", key)
			}
			_ = err // an error is fine; a deletion is not
		}
		if deck, _ := s.Deck(); len(deck) != 1 {
			t.Errorf("a traversal key removed a real word: deck = %+v", deck)
		}
	})

	t.Run("event fields survive the round trip", func(t *testing.T) {
		s := newStore(t)
		want := store.ReviewEvent{Word: "hot dog", Kind: store.EventReviewed, Found: true, Correct: true, At: day(2)}
		_ = s.AppendEvent(want)
		got, _ := s.Events(time.Time{})
		if len(got) != 1 {
			t.Fatalf("got %d events", len(got))
		}
		if got[0].Word != want.Word || got[0].Kind != want.Kind || got[0].Correct != want.Correct {
			t.Errorf("got %+v, want %+v", got[0], want)
		}
		if !got[0].At.Equal(want.At) {
			t.Errorf("At = %v, want %v", got[0].At, want.At)
		}
	})

	t.Run("news items round-trip, and never-fetched is distinct from fetched-empty", func(t *testing.T) {
		// The distinction is the whole reason NewsItems returns a timestamp
		// alongside the slice. A fetch that succeeds with ZERO items is a real
		// answer — some words are simply not in the news — and a []NewsItem alone
		// cannot tell it from "we have never looked", so the cache would either
		// re-fetch those words forever or freeze them empty.
		s := newStore(t)

		items, at, err := s.NewsItems("ephemeral")
		if err != nil {
			t.Fatalf("NewsItems: %v", err)
		}
		if len(items) != 0 || !at.IsZero() {
			t.Errorf("never fetched: got %d items at %v, want none at the zero time", len(items), at)
		}

		want := []store.NewsItem{
			{Title: "Springtime is ephemeral", URL: "u1", Source: "UChicago", At: day(1)},
			{Title: "Ephemeral Architecture", URL: "u2", Source: "ArchDaily", At: day(2)},
		}
		if err := s.SetNewsItems("ephemeral", want, day(3)); err != nil {
			t.Fatalf("SetNewsItems: %v", err)
		}
		got, gotAt, err := s.NewsItems("ephemeral")
		if err != nil {
			t.Fatalf("NewsItems after set: %v", err)
		}
		if len(got) != len(want) {
			t.Fatalf("got %d items, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i].Title != want[i].Title || got[i].URL != want[i].URL || got[i].Source != want[i].Source {
				t.Errorf("item %d = %+v, want %+v", i, got[i], want[i])
			}
			if !got[i].At.Equal(want[i].At) {
				t.Errorf("item %d At = %v, want %v", i, got[i].At, want[i].At)
			}
		}
		if !gotAt.Equal(day(3)) {
			t.Errorf("fetched-at = %v, want %v", gotAt, day(3))
		}

		// Fetched-empty: a real answer, and distinguishable from never-fetched
		// only by the timestamp.
		if err := s.SetNewsItems("quokka", nil, day(4)); err != nil {
			t.Fatalf("SetNewsItems empty: %v", err)
		}
		empty, emptyAt, err := s.NewsItems("quokka")
		if err != nil {
			t.Fatalf("NewsItems empty: %v", err)
		}
		if len(empty) != 0 {
			t.Errorf("got %d items, want none", len(empty))
		}
		if emptyAt.IsZero() {
			t.Error("fetched-empty reads as never-fetched — the cache cannot tell them apart")
		}
	})

	t.Run("a second SetNewsItems replaces rather than appends", func(t *testing.T) {
		s := newStore(t)
		if err := s.SetNewsItems("ephemeral", []store.NewsItem{{Title: "old"}}, day(1)); err != nil {
			t.Fatalf("SetNewsItems: %v", err)
		}
		if err := s.SetNewsItems("ephemeral", []store.NewsItem{{Title: "new"}}, day(2)); err != nil {
			t.Fatalf("SetNewsItems: %v", err)
		}
		got, at, err := s.NewsItems("ephemeral")
		if err != nil {
			t.Fatalf("NewsItems: %v", err)
		}
		if len(got) != 1 || got[0].Title != "new" {
			t.Errorf("got %v, want only the newer item", got)
		}
		if !at.Equal(day(2)) {
			t.Errorf("fetched-at = %v, want the newer %v", at, day(2))
		}
	})

	t.Run("news item keys are normalised the way word keys are", func(t *testing.T) {
		s := newStore(t)
		if err := s.SetNewsItems("Hot  Dog", []store.NewsItem{{Title: "x"}}, day(1)); err != nil {
			t.Fatalf("SetNewsItems: %v", err)
		}
		got, _, err := s.NewsItems("hot dog")
		if err != nil {
			t.Fatalf("NewsItems: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("got %d items — the cache key must be store.Key, as the deck's is", len(got))
		}
	})

	t.Run("word facts are absent before the first harvest, and absence is not an error", func(t *testing.T) {
		// The normal state of every word until --harvest runs, exactly as an
		// absent user model is (#16 M2). A zero At is what says so: WordFacts
		// carries its own timestamp, so it needs no second return value to
		// distinguish "never assigned" from "assigned", the way NewsItems needs
		// one for items it does not timestamp itself.
		s := newStore(t)
		f, err := s.WordFacts("sycophantic")
		if err != nil {
			t.Fatalf("WordFacts: %v", err)
		}
		if f.Harvested() {
			t.Errorf("WordFacts = %+v, want unharvested", f)
		}
	})

	t.Run("word facts round-trip", func(t *testing.T) {
		s := newStore(t)
		want := store.WordFacts{Band: store.C1, Domain: domain(t, "Law"), At: day(3)}
		if err := s.SetWordFacts("certiorari", want); err != nil {
			t.Fatalf("SetWordFacts: %v", err)
		}
		got, err := s.WordFacts("certiorari")
		if err != nil {
			t.Fatalf("WordFacts: %v", err)
		}
		if !got.Harvested() {
			t.Fatal("facts read back unharvested")
		}
		if got.Band != want.Band || got.Domain != want.Domain || !got.At.Equal(want.At) {
			t.Errorf("WordFacts = %+v, want %+v", got, want)
		}
	})

	t.Run("a second SetWordFacts REPLACES rather than merging", func(t *testing.T) {
		// A band is assigned once and reused forever. A write that merged would
		// leave a word holding a band from one run and a domain from another,
		// and the cache's whole premise is that these are one judgement.
		s := newStore(t)
		if err := s.SetWordFacts("estoppel", store.WordFacts{Band: store.B2, Domain: store.DomainGeneral, At: day(1)}); err != nil {
			t.Fatalf("SetWordFacts: %v", err)
		}
		if err := s.SetWordFacts("estoppel", store.WordFacts{Band: store.C2, Domain: domain(t, "Law"), At: day(2)}); err != nil {
			t.Fatalf("SetWordFacts again: %v", err)
		}
		got, err := s.WordFacts("estoppel")
		if err != nil {
			t.Fatalf("WordFacts: %v", err)
		}
		if got.Band != store.C2 || got.Domain != domain(t, "Law") || !got.At.Equal(day(2)) {
			t.Errorf("WordFacts = %+v, want the second write whole", got)
		}
	})

	t.Run("a damaged record reads as unharvested from EVERY implementation", func(t *testing.T) {
		// The Store interface promises this, so the suite is where it belongs —
		// it landed in yaml_test.go only, and Mem quietly disagreed: it handed
		// back an off-scale band as harvested while YAML refused it. A fake that
		// holds a state the real store cannot is the gap this suite exists to
		// close, and here the fake was the PERMISSIVE one, which is the direction
		// that hides a bug rather than inventing one.
		s := newStore(t)
		if err := s.SetWordFacts("word", store.WordFacts{
			Band: "B2+", Domain: "Astrology", At: day(1),
		}); err != nil {
			t.Fatalf("SetWordFacts: %v", err)
		}
		got, err := s.WordFacts("word")
		if err != nil {
			t.Fatalf("WordFacts: %v", err)
		}
		if got.Harvested() {
			t.Errorf("WordFacts = %+v, want unharvested: an off-scale band must never "+
				"reach Rank, which answers -1 and sorts below A1", got)
		}
	})

	t.Run("a band is stored CANONICAL, so casing is not two facts", func(t *testing.T) {
		s := newStore(t)
		if err := s.SetWordFacts("word", store.WordFacts{
			Band: "c1", Domain: "law", At: day(1),
		}); err != nil {
			t.Fatalf("SetWordFacts: %v", err)
		}
		got, err := s.WordFacts("word")
		if err != nil {
			t.Fatalf("WordFacts: %v", err)
		}
		if got.Band != store.C1 {
			t.Errorf("band = %q, want the canonical C1", got.Band)
		}
		if got.Domain != domain(t, "Law") {
			t.Errorf("domain = %q, want the canonical Law", got.Domain)
		}
	})

	t.Run("model text in an item is neutralised on the way in", func(t *testing.T) {
		// Stem, Answer and Distractors are the first free-text model fields this
		// store persists, and the board renders them one per line. A distractor
		// carrying a newline forges a row there — the same class as the band that
		// forged a "define: ..." diagnostic in #17. Neutralised at the WRITE so
		// every render site is safe without remembering to be.
		//
		// THE CLASS IS "A RUNE THE OUTPUT OBEYS", not "a line break" (#12 BR-15).
		// A cloze prompt paints these fields into a raw alternate screen, where
		// ESC and BEL are the dangerous ones and neither is whitespace:
		// "\x1b[2J\x1b[H" clears the screen mid-sitting. This row asserts over
		// unicode.IsControl rather than over "\r\n" so the next control
		// character nobody thought of is covered by the assertion that is already
		// here.
		s := newStore(t)
		if err := s.SetItems("word", []store.Item{{
			Word:        "word",
			Stem:        "a stem\nwith a forged second line\x1b[2J\x1b[H",
			Answer:      "word\r\nand another\a",
			Distractors: []string{"one\ntwo", "three\x1b[31m"},
		}}); err != nil {
			t.Fatalf("SetItems: %v", err)
		}
		got, err := s.Items("word")
		if err != nil {
			t.Fatalf("Items: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d items, want 1", len(got))
		}
		fields := append([]string{got[0].Stem, got[0].Answer}, got[0].Distractors...)
		for _, field := range fields {
			for _, r := range field {
				if unicode.IsControl(r) {
					t.Errorf("%q kept the control rune %q — a terminal obeys it", field, r)
					break
				}
			}
		}
		// AND THE WORDS SURVIVE. Dropping every control rune including the
		// whitespace ones would join "a\nb" into "ab", which is a different
		// sentence rather than a safe one.
		if !strings.Contains(got[0].Stem, "stem with a forged") {
			t.Errorf("the newline was dropped rather than collapsed to a space: %q", got[0].Stem)
		}
	})

	t.Run("word fact keys are normalised the way word keys are", func(t *testing.T) {
		s := newStore(t)
		if err := s.SetWordFacts("Hot  Dog", store.WordFacts{Band: store.A2, Domain: store.DomainGeneral, At: day(1)}); err != nil {
			t.Fatalf("SetWordFacts: %v", err)
		}
		got, err := s.WordFacts("hot dog")
		if err != nil {
			t.Fatalf("WordFacts: %v", err)
		}
		if !got.Harvested() {
			t.Error("the facts key must be store.Key, as the deck's is")
		}
	})

	t.Run("items are absent before authoring, and round-trip once written", func(t *testing.T) {
		s := newStore(t)
		got, err := s.Items("sycophantic")
		if err != nil {
			t.Fatalf("Items: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("Items = %v, want none", got)
		}

		want := []store.Item{{
			Word:        "sycophantic",
			Form:        store.FormCloze,
			Stem:        "Reporters described the aide as ___, agreeing with the minister before he finished speaking.",
			Answer:      "sycophantic",
			Distractors: []string{"laconic", "punctilious", "querulous"},
			At:          day(3),
		}}
		if err := s.SetItems("sycophantic", want); err != nil {
			t.Fatalf("SetItems: %v", err)
		}
		got, err = s.Items("sycophantic")
		if err != nil {
			t.Fatalf("Items after write: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d items, want 1", len(got))
		}
		if got[0].Stem != want[0].Stem || got[0].Answer != want[0].Answer || got[0].Form != want[0].Form {
			t.Errorf("Item = %+v, want %+v", got[0], want[0])
		}
		if len(got[0].Distractors) != 3 {
			t.Errorf("distractors = %v, want 3", got[0].Distractors)
		}
		if !got[0].At.Equal(day(3)) {
			t.Errorf("At = %v, want %v", got[0].At, day(3))
		}
	})

	t.Run("a second SetItems replaces rather than appends", func(t *testing.T) {
		// Same rule as the news cache: the authored set for a word is REPLACED,
		// so a re-harvest cannot silently double a word's items every run.
		s := newStore(t)
		if err := s.SetItems("ephemeral", []store.Item{{Word: "ephemeral", Stem: "old", At: day(1)}}); err != nil {
			t.Fatalf("SetItems: %v", err)
		}
		if err := s.SetItems("ephemeral", []store.Item{{Word: "ephemeral", Stem: "new", At: day(2)}}); err != nil {
			t.Fatalf("SetItems again: %v", err)
		}
		got, err := s.Items("ephemeral")
		if err != nil {
			t.Fatalf("Items: %v", err)
		}
		if len(got) != 1 || got[0].Stem != "new" {
			t.Errorf("got %v, want only the newer item", got)
		}
	})

	t.Run("items come back NEWEST FIRST, from every implementation", func(t *testing.T) {
		// Promised on the Store interface, so it is asserted here — it was
		// pinned only through the pure prune tests, which cannot see whether a
		// real store's read path preserves the order its write path produced.
		//
		// #12 picks among a word's items and will take the first; insertion
		// order would hand it the oldest.
		s := newStore(t)
		if err := s.SetItems("w", []store.Item{
			{Word: "w", Form: store.FormCloze, Stem: "oldest", At: day(1)},
			{Word: "w", Form: store.FormCloze, Stem: "newest", At: day(3)},
			{Word: "w", Form: store.FormCloze, Stem: "middle", At: day(2)},
		}); err != nil {
			t.Fatalf("SetItems: %v", err)
		}
		got, err := s.Items("w")
		if err != nil {
			t.Fatalf("Items: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d items, want 3", len(got))
		}
		for i, want := range []string{"newest", "middle", "oldest"} {
			if got[i].Stem != want {
				t.Errorf("item %d = %q, want %q — items must read back newest first",
					i, got[i].Stem, want)
			}
		}
	})

	t.Run("a word's items are CAPPED, by every implementation", func(t *testing.T) {
		// "Growth is bounded" is stated on the Store interface, so it belongs
		// here. It landed in a Mem-only test, and held for YAML by the
		// coincidence that both call the same helper — which is the assumption
		// this suite exists to stop relying on.
		s := newStore(t)
		var many []store.Item
		for i := range store.ItemCap * 3 {
			many = append(many, store.Item{
				Word: "w", Form: store.FormCloze, Stem: string(rune('a' + i)), At: day(1),
			})
		}
		if err := s.SetItems("w", many); err != nil {
			t.Fatalf("SetItems: %v", err)
		}
		got, err := s.Items("w")
		if err != nil {
			t.Fatalf("Items: %v", err)
		}
		if len(got) != store.ItemCap {
			t.Errorf("stored %d items, want the cap of %d", len(got), store.ItemCap)
		}
	})

	t.Run("an unknown Form is refused on the way in", func(t *testing.T) {
		s := newStore(t)
		if err := s.SetItems("w", []store.Item{{Word: "w", Form: "telepathy", Stem: "x", At: day(1)}}); err != nil {
			t.Fatalf("SetItems: %v", err)
		}
		got, err := s.Items("w")
		if err != nil {
			t.Fatalf("Items: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d items, want 1", len(got))
		}
		if got[0].Form != "" {
			t.Errorf("Form = %q, want it refused — a renderer switching on it knows no such form", got[0].Form)
		}
	})

	t.Run("a word may hold several items", func(t *testing.T) {
		// 1:N with a word, which is what lets #12 pick among them and what the
		// Form field exists to let #13 share.
		s := newStore(t)
		want := []store.Item{
			{Word: "obdurate", Form: store.FormCloze, Stem: "a", Answer: "obdurate", At: day(1)},
			{Word: "obdurate", Form: store.FormSentence, Stem: "b", Answer: "obdurate", At: day(2)},
		}
		if err := s.SetItems("obdurate", want); err != nil {
			t.Fatalf("SetItems: %v", err)
		}
		got, err := s.Items("obdurate")
		if err != nil {
			t.Fatalf("Items: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d items, want 2", len(got))
		}
		if got[0].Form == got[1].Form {
			t.Errorf("both items have Form %q — the discriminator did not survive", got[0].Form)
		}
	})

}

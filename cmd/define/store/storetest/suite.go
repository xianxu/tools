// Package storetest holds the conformance suite every Store implementation must
// pass.
//
// It exists so "the in-memory store behaves like the YAML one" is a test rather
// than an assumption — the gap that a fake diverging from its real counterpart
// silently creates.
package storetest

import (
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
)

// Suite runs every Store obligation against a freshly constructed store.
func Suite(t *testing.T, newStore func(t *testing.T) store.Store) {
	t.Helper()
	day := func(n int) time.Time {
		return time.Date(2026, 8, n, 12, 0, 0, 0, time.UTC)
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

}

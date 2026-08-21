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
}

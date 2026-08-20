package main

import (
	"errors"
	"slices"
	"testing"
)

// The walk order is the assertion, not just the returned bytes.
func TestFetchWalksCandidatesInOrderAndStopsAtFirstHit(t *testing.T) {
	cdn := newFakeCDN(t, map[string][]byte{"/c.mp3": []byte("ID3audio")})
	data, from, err := cdn.source().Fetch(t.Context(), cdn.urls("/a.mp3", "/b.mp3", "/c.mp3", "/d.mp3"))
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(data) != "ID3audio" {
		t.Errorf("data = %q", data)
	}
	if got := stripHost(t, from, cdn.URL); got != "/c.mp3" {
		t.Errorf("from = %q, want /c.mp3", got)
	}
	// /d.mp3 must NOT appear — the walk stops at the first hit.
	if want := []string{"/a.mp3", "/b.mp3", "/c.mp3"}; !slices.Equal(cdn.Requested(), want) {
		t.Errorf("walk order = %v, want %v", cdn.Requested(), want)
	}
}

func TestFetchAllMissingTriesEveryCandidate(t *testing.T) {
	cdn := newFakeCDN(t, nil)
	urls := cdn.urls("/a.mp3", "/b.mp3", "/c.mp3")
	_, _, err := cdn.source().Fetch(t.Context(), urls)
	if !errors.Is(err, ErrNoAudio) {
		t.Errorf("err = %v, want ErrNoAudio", err)
	}
	if got := len(cdn.Requested()); got != len(urls) {
		t.Errorf("tried %d candidates, want all %d", got, len(urls))
	}
}

func TestFetchNoCandidates(t *testing.T) {
	cdn := newFakeCDN(t, nil)
	if _, _, err := cdn.source().Fetch(t.Context(), nil); !errors.Is(err, ErrNoAudio) {
		t.Errorf("err = %v, want ErrNoAudio", err)
	}
}

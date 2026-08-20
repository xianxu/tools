package main

import (
	"context"
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

// A transport failure must NOT be reported as "no recorded pronunciation": one
// is a normal outcome for a word, the other means the network is down. The
// %w:%w chain is the whole point of the distinction, so it is pinned here — a
// regression to %v, or a swap back to ErrNoAudio, passes nothing.
func TestFetchTransportFailureIsNotErrNoAudio(t *testing.T) {
	cdn := newFakeCDN(t, nil)
	urls := cdn.urls("/a.mp3")
	cdn.Close() // server gone: every request is a transport error

	_, _, err := cdn.source().Fetch(t.Context(), urls)
	if err == nil {
		t.Fatal("want an error")
	}
	if !errors.Is(err, ErrFetchFailed) {
		t.Errorf("err = %v, want ErrFetchFailed", err)
	}
	if errors.Is(err, ErrNoAudio) {
		t.Error("a transport failure must not report as ErrNoAudio")
	}
}

func TestFetchContextCancellationReachesTheCaller(t *testing.T) {
	cdn := newFakeCDN(t, map[string][]byte{"/a.mp3": []byte("ID3")})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, _, err := cdn.source().Fetch(ctx, cdn.urls("/a.mp3"))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("errors.Is(err, context.Canceled) = false for %v — the cause was flattened out of the chain", err)
	}
}

// --- cachingAudioSource ------------------------------------------------------

func TestCachingAudioSourceServesRepeatsFromMemory(t *testing.T) {
	cdn := newFakeCDN(t, map[string][]byte{"/a.mp3": []byte("ID3audio")})
	src := newCachingAudioSource(cdn.source())
	urls := cdn.urls("/a.mp3")

	first, _, err := src.Fetch(t.Context(), urls)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := src.Fetch(t.Context(), urls)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Errorf("cached bytes differ: %q vs %q", first, second)
	}
	if got := cdn.Requested(); len(got) != 1 {
		t.Errorf("made %d requests, want 1: %v", len(got), got)
	}
}

func TestCachingAudioSourceDistinguishesWords(t *testing.T) {
	cdn := newFakeCDN(t, map[string][]byte{"/a.mp3": []byte("A"), "/b.mp3": []byte("B")})
	src := newCachingAudioSource(cdn.source())

	src.Fetch(t.Context(), cdn.urls("/a.mp3"))
	src.Fetch(t.Context(), cdn.urls("/b.mp3"))
	if got := cdn.Requested(); len(got) != 2 {
		t.Errorf("made %d requests, want 2 — different words shared a cache entry: %v", len(got), got)
	}
}

// A TRANSPORT failure is transient and must stay retryable — unlike ErrNoAudio,
// which is permanent and is cached (see the test below). A 404 is not a
// transport failure, so this closes the server to produce a real one.
func TestCachingAudioSourceDoesNotCacheTransportFailures(t *testing.T) {
	cdn := newFakeCDN(t, nil)
	urls := cdn.urls("/a.mp3")
	src := newCachingAudioSource(cdn.source())
	cdn.Close()

	for i := 0; i < 2; i++ {
		if _, _, err := src.Fetch(t.Context(), urls); !errors.Is(err, ErrFetchFailed) {
			t.Fatalf("fetch %d: %v, want ErrFetchFailed", i, err)
		}
	}
	// The server is closed, so nothing is recorded server-side; what matters is
	// that the second call still ATTEMPTED rather than being served a cached
	// error — a closed cache would return instantly with no attempt.
	if _, _, err := src.Fetch(t.Context(), urls); !errors.Is(err, ErrFetchFailed) {
		t.Errorf("third fetch: %v — a transient failure was cached", err)
	}
}

// "No recording exists" is permanent, unlike a transport failure. Replaying a
// word with no audio must not re-issue all four candidate requests every time.
func TestCachingAudioSourceCachesErrNoAudio(t *testing.T) {
	cdn := newFakeCDN(t, nil) // every candidate 404s → ErrNoAudio
	src := newCachingAudioSource(cdn.source())
	urls := cdn.urls("/a.mp3", "/b.mp3")

	if _, _, err := src.Fetch(t.Context(), urls); !errors.Is(err, ErrNoAudio) {
		t.Fatalf("first fetch: %v", err)
	}
	before := len(cdn.Requested())
	if _, _, err := src.Fetch(t.Context(), urls); !errors.Is(err, ErrNoAudio) {
		t.Fatalf("second fetch: %v", err)
	}
	if got := len(cdn.Requested()); got != before {
		t.Errorf("made %d more requests for a word with no recording, want 0", got-before)
	}
}

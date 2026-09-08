package main

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
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

// --- audioSeam ---------------------------------------------------------------

func TestAudioSeamServesRepeatsFromMemory(t *testing.T) {
	cdn := newFakeCDN(t, map[string][]byte{"/a.mp3": []byte("ID3audio")})
	src := newAudioSeam(cdn.source())
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

func TestAudioSeamDistinguishesWords(t *testing.T) {
	cdn := newFakeCDN(t, map[string][]byte{"/a.mp3": []byte("A"), "/b.mp3": []byte("B")})
	src := newAudioSeam(cdn.source())

	src.Fetch(t.Context(), cdn.urls("/a.mp3"))
	src.Fetch(t.Context(), cdn.urls("/b.mp3"))
	if got := cdn.Requested(); len(got) != 2 {
		t.Errorf("made %d requests, want 2 — different words shared a cache entry: %v", len(got), got)
	}
}

// A TRANSPORT failure is transient and must stay retryable — unlike ErrNoAudio,
// which is permanent and is cached (see the test below). A 404 is not a
// transport failure, so this closes the server to produce a real one.
func TestAudioSeamDoesNotCacheTransportFailures(t *testing.T) {
	cdn := newFakeCDN(t, nil)
	urls := cdn.urls("/a.mp3")
	src := newAudioSeam(cdn.source())
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
func TestAudioSeamCachesErrNoAudio(t *testing.T) {
	cdn := newFakeCDN(t, nil) // every candidate 404s → ErrNoAudio
	src := newAudioSeam(cdn.source())
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

// Italian's absence must be REPORTED, not heard as an English recording nobody
// said was English (#29). Nine probes across three locale forms found no Italian
// audio in this CDN generation, so this is the permanent case, not a transient.
func TestPlayAnnouncedReportsTheVoiceThatAnswered(t *testing.T) {
	en := voice{Lang: "en", Locale: "us"}
	it := voice{Lang: "it", Locale: "it"}
	asked := utterance{Word: "ciao", Spellings: []string{"ciao"}, Source: it, Session: en}
	plain := utterance{Word: "ciao", Session: en}

	for _, tc := range []struct {
		name    string
		u       utterance
		serve   []string // which URLs the CDN has
		wantErr []string // substrings stderr must carry; empty means stderr must be EMPTY
	}{
		{
			name:    "a source was asked for and is missing: the session's plays, and it says so",
			u:       asked,
			serve:   AudioCandidates("ciao", en),
			wantErr: []string{"no it recording for ciao", "played the en one"},
		},
		{
			// The counterpart, so the report cannot quietly become a line on
			// every lookup.
			name:  "a source was asked for and answered: nothing to report",
			u:     asked,
			serve: AudioCandidates("ciao", it),
		},
		{
			// The third cell: the session's recording played, but nobody asked
			// for anything else, so there is no surprise to report.
			name:  "no source was asked for: nothing to report",
			u:     plain,
			serve: AudioCandidates("ciao", en),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rig := newAudioRigServing(t, tc.serve...)
			opt := options{times: 1}
			var out, errb bytes.Buffer

			playAnnounced(t.Context(), rig.deps, opt, tc.u, indicator{}, &out, &errb)

			if n := rig.player.count(); n != opt.times {
				t.Errorf("played %d times, want %d — nothing reached the player", n, opt.times)
			}
			if len(tc.wantErr) == 0 {
				if errb.Len() != 0 {
					t.Errorf("stderr should be empty, got %q", errb.String())
				}
				return
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(errb.String(), want) {
					t.Errorf("stderr missing %q, got %q", want, errb.String())
				}
			}
		})
	}
}

// A record with a hole in it is the one failure this design cannot afford.
//
// reportVoice used to print voice.Lang raw, while AudioCandidates defaults an
// empty Lang to English — so a zero session voice produced "played the  one".
// Latent in production (applyVoice always runs) and exactly the kind of latent
// the close review found by scratch-running it.
func TestTheVoiceReportNamesALanguageEvenWithAZeroVoice(t *testing.T) {
	u := utterance{
		Word:      "ciao",
		Spellings: []string{"ciao"},
		Source:    voice{Lang: "it", Locale: "it"},
		Session:   voice{}, // never through applyVoice
	}
	var b bytes.Buffer
	reportVoice(&b, u, "https://example.invalid/not-a-source.mp3")
	if got := b.String(); !strings.Contains(got, "played the en one") {
		t.Errorf("report = %q, want it to name a language rather than a blank", got)
	}
}

// A ZERO-BYTE 200 IS NOT A RECORDING, AT EVERY LAYER (#46 BR-22).
//
// The store learned this in round 4 and the memo did not, which is the same
// finding one layer up: httpAudioSource returns an empty body as SUCCESS, so the
// seam cached silence for the whole sitting and handed back a `from` URL that
// reportVoice prints as the voice that answered — a fabricated record, which is
// the half that makes it more than a nuisance.
func TestAZeroByteResponseIsNotAHit(t *testing.T) {
	cdn := newFakeCDN(t, map[string][]byte{"/a.mp3": {}})
	seam := newAudioSeam(cdn.source())
	urls := cdn.urls("/a.mp3")

	data, from, err := seam.Fetch(t.Context(), urls)
	if err == nil {
		t.Errorf("an empty body was served as a hit: %d bytes, from %q — it plays as "+
			"silence and reportVoice prints that URL as the voice that answered", len(data), from)
	}
	if !errors.Is(err, ErrNoAudio) {
		t.Errorf("err = %v, want ErrNoAudio", err)
	}
	// AND IT IS NOT REMEMBERED AS A MISS. A miss suppresses the re-ask; an empty
	// body is a server hiccup, not "this word has no recording".
	before := len(cdn.Requested())
	seam.Fetch(t.Context(), urls)
	if after := len(cdn.Requested()); after <= before {
		t.Error("the empty response was cached as a permanent miss; the next play must re-ask")
	}
}

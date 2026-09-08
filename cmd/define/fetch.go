package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ErrNoAudio means no candidate URL carried a recording. Like ErrNoEntry this is
// a normal outcome: NOAD's gaps and the CDN's gaps correlate, since both trace
// back to Oxford.
var ErrNoAudio = errors.New("no recorded pronunciation")

// ErrFetchFailed means the CDN could not be reached or read. Distinguishing it
// from ErrNoAudio matters: "this word has no recording" is a normal outcome,
// "the network is down" is not. dict_darwin.go draws the same line between "no
// entry" and a CoreFoundation failure.
var ErrFetchFailed = errors.New("audio fetch failed")

// AudioSource fetches a recording, given candidate URLs in preference order.
//
// The seam exists so AudioCandidates can stay pure and offline, and so the walk
// order can be asserted against a fake rather than against Google.
type AudioSource interface {
	Fetch(ctx context.Context, urls []string) (data []byte, from string, err error)
}

// maxAudioBytes caps a response. The real files are a few KB; anything far
// larger is not a pronunciation and should not be buffered.
const maxAudioBytes = 4 << 20

type httpAudioSource struct {
	client *http.Client
}

func newHTTPAudioSource() *httpAudioSource {
	return &httpAudioSource{client: &http.Client{Timeout: 10 * time.Second}}
}

// Fetch walks the candidates in order and returns the first that answers 200.
func (s *httpAudioSource) Fetch(ctx context.Context, urls []string) ([]byte, string, error) {
	var firstErr error
	for _, u := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, "", err
		}
		resp, err := s.client.Do(req)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxAudioBytes))
		resp.Body.Close()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		return data, u, nil
	}
	if firstErr != nil {
		// %w on both, so errors.Is reaches the transport cause (context.Canceled,
		// a DNS failure) rather than flattening it into the text.
		return nil, "", fmt.Errorf("%w: %w", ErrFetchFailed, firstErr)
	}
	return nil, "", ErrNoAudio
}

// audioSeam is the audio source AND its memo, as one value.
//
// A POINTER, so every by-value copy of deps shares one memo — which a decorator
// on the field could never guarantee, because deps is copied at every call.
//
// THIS SHAPE IS WHAT FOUR PLAN-GATE ROUNDS CONVERGED ON, and the reason is that
// the previous shape asked a question with no derivable answer. "Which functions
// must remember to wrap the source" was answered wrongly four times: realDeps
// (no test calls it), run()'s callees (seven one-shot commands), replRaw (the
// wrap is in runEditor), and "~8 functions" (measured: three times that). Every
// answer was a statement about the code that the code did not support.
//
// A field of this type does not ask the question. There is no unwrapped source
// to hold, so there is no wrap to forget, no predicate to derive and no guard to
// keep honest — the COMPILER enumerates the construction sites, which is the
// only enumeration in this program that cannot drift. #2's I-1 lesson is
// satisfied absolutely rather than by convention: a test cannot build a deps
// whose audio differs in KIND from production's, only in what it wraps.
type audioSeam struct {
	inner AudioSource
	mu    sync.Mutex
	hits  map[string]cachedAudio
	// misses records candidate lists the CDN has no recording for. That is
	// PERMANENT — unlike a transport failure — so replaying such a word must not
	// re-issue all four candidate requests every time. The error taxonomy above
	// is the single source of that distinction; this derives from it rather than
	// re-deciding what "failed" means.
	misses map[string]struct{}
}

type cachedAudio struct {
	data []byte
	from string
}

// newAudioSeam is the ONE door. Production and every test reach the source
// through it, which is what makes "the line the tests exercise is the line
// production runs" (#2 I-1) a property of the type rather than a habit.
//
// A nil inner is legal and means "no audio": Fetch reports ErrNoAudio without
// reaching for anything. That is what the tests that used to write
// noAudioSource{} want, and it removes their need to define a source at all.
func newAudioSeam(inner AudioSource) *audioSeam {
	return &audioSeam{inner: inner, hits: map[string]cachedAudio{}, misses: map[string]struct{}{}}
}

// wordFiler is an AudioSource that can file what it caches under a WORD.
//
// An optional capability, asked for rather than assumed — the same shape `play`
// uses for Missed and Flagging, and for the same reason: most sources have no
// use for a word, and a mandatory parameter would put one in the interface that
// AudioCandidates and the fake both have to carry for nothing.
//
// It exists because the seam's key is the CANDIDATE LIST while a durable cache
// must also be reachable by the word Forget names. Neither can be derived from
// the other — parsing a word out of a URL is the driftable second statement of
// identity AudioKey refuses — so the caller that has both says so.
type wordFiler interface {
	AudioSource
	forWord(word string) AudioSource
}

// FetchFor is Fetch, told which word the recording belongs to.
//
// The memo does not use the word: its key is the candidate list, and two words
// that somehow produced one candidate list would BE one recording. The word is
// carried for the layer below, which files what it stores so Forget can find it.
func (c *audioSeam) FetchFor(ctx context.Context, word string, urls []string) ([]byte, string, error) {
	if c == nil {
		return nil, "", ErrNoAudio
	}
	if wf, ok := c.inner.(wordFiler); ok {
		return c.fetch(ctx, wf.forWord(word), urls)
	}
	return c.Fetch(ctx, urls)
}

// Fetch answers from the memo, or from the source once.
//
// The KEY is the whole candidate list, because that is what identifies a
// recording: utterance.Candidates() builds it from a source voice, a session
// voice and the spellings, so `-locale gb` and `-locale us` are different keys
// for one word — and anything narrower would serve the wrong recording.
func (c *audioSeam) Fetch(ctx context.Context, urls []string) ([]byte, string, error) {
	if c == nil {
		return nil, "", ErrNoAudio
	}
	return c.fetch(ctx, c.inner, urls)
}

// fetch is the memo itself, over whichever source the caller resolved.
//
// The source is a PARAMETER so FetchFor can hand down a word-filed view without
// a second memo — one map, one lock, one set of hits, however the inner was
// resolved. Two memos would be two answers to "have we fetched this".
func (c *audioSeam) fetch(ctx context.Context, inner AudioSource, urls []string) ([]byte, string, error) {
	if inner == nil {
		return nil, "", ErrNoAudio
	}
	key := strings.Join(urls, "\n")

	c.mu.Lock()
	hit, ok := c.hits[key]
	_, missed := c.misses[key]
	c.mu.Unlock()
	if ok {
		return hit.data, hit.from, nil
	}
	if missed {
		return nil, "", ErrNoAudio
	}

	data, from, err := inner.Fetch(ctx, urls)
	if err != nil {
		if errors.Is(err, ErrNoAudio) {
			c.mu.Lock()
			c.misses[key] = struct{}{}
			c.mu.Unlock()
		}
		// ErrFetchFailed stays retryable: a transient outage must not poison the
		// rest of the session.
		return nil, "", err
	}
	if len(data) == 0 {
		// A ZERO-BYTE 200 IS NOT A RECORDING, and the memo has to say so too.
		//
		// The store already refuses this at both ends, and the round-4 finding
		// that produced those guards named the class — "the payload cannot be a
		// recording" — while I fixed only the layer in front of me. Here it is
		// one layer up: httpAudioSource returns an empty body as SUCCESS, so
		// without this the sitting serves silence for every replay, with a `from`
		// URL that reportVoice prints as the voice that answered. A memo entry
		// never expires, so it lasts the whole session.
		//
		// Not recorded as a miss either: a miss suppresses the re-ask, and an
		// empty body is a server hiccup rather than "this word has no recording".
		return nil, "", ErrNoAudio
	}
	c.mu.Lock()
	c.hits[key] = cachedAudio{data, from}
	c.mu.Unlock()
	return data, from, nil
}

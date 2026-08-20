package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrNoAudio means no candidate URL carried a recording. Like ErrNoEntry this is
// a normal outcome: NOAD's gaps and the CDN's gaps correlate, since both trace
// back to Oxford.
var ErrNoAudio = errors.New("no recorded pronunciation")

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
		return nil, "", fmt.Errorf("%w: %v", ErrNoAudio, firstErr)
	}
	return nil, "", ErrNoAudio
}

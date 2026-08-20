package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeCDN models Google's pronunciation CDN: a set of paths that exist, and a
// record of every path requested IN ORDER.
//
// The ordering record is the point. A function-call mock can say "Fetch was
// called"; only a stateful fake can show that the walk tried the 2022 path
// before the legacy one and stopped at the first hit — which is the entire
// behaviour AudioCandidates' ordering exists to produce.
type fakeCDN struct {
	*httptest.Server
	mu        sync.Mutex
	present   map[string][]byte
	requested []string
}

func newFakeCDN(t *testing.T, present map[string][]byte) *fakeCDN {
	t.Helper()
	c := &fakeCDN{present: present}
	c.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.mu.Lock()
		c.requested = append(c.requested, r.URL.Path)
		body, ok := c.present[r.URL.Path]
		c.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write(body)
	}))
	t.Cleanup(c.Close)
	return c
}

func (c *fakeCDN) Requested() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.requested...)
}

func (c *fakeCDN) urls(paths ...string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = c.URL + p
	}
	return out
}

func (c *fakeCDN) source() *httpAudioSource {
	return &httpAudioSource{client: c.Client()}
}

func stripHost(t *testing.T, url, base string) string {
	t.Helper()
	return strings.TrimPrefix(url, base)
}

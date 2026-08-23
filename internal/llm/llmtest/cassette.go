package llmtest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// Cassette is a real exchange, frozen at the WIRE.
//
// You cannot fake judgment. You can freeze a real answer and pin our handling of
// it — which is what a consumer's tests need: that a well-formed veto verdict
// decodes, that a refusal is skipped, that a truncation is caught. What a
// cassette does NOT establish is that the model reliably produces that answer; it
// is one sample of a stochastic process, which is what the conformance run is for.
//
// It sits BENEATH the seam, as an http.RoundTripper, for the same reason the Fake
// does: replacing llm.Client would mean a replayed test never serialises a
// request, never runs a retry and never parses an SSE frame — so it could not see
// a dropped header or a mis-serialized output_config, which are exactly the bugs
// this harness can have. Below the transport, all of that still runs.
type Cassette struct {
	dir string
	t   *testing.T
}

// Cassettes opens the store under a consumer's testdata directory. Recordings
// live with the consumer that owns the prompt, as goldens do.
func Cassettes(t *testing.T, dir string) *Cassette {
	t.Helper()
	return &Cassette{dir: filepath.Join(dir, "cassettes"), t: t}
}

// exchange is what lands on disk: the question AND the answer, both as sent.
//
// The request is stored, not just hashed, so the artifact is self-describing: a
// reviewer reading a diff can see what was asked without recomputing a hash.
type exchange struct {
	Request  json.RawMessage `json:"request"`
	Status   int             `json:"status"`
	Response json.RawMessage `json:"response"`
}

// key hashes the request body with volatile fields removed.
//
// max_tokens is dropped deliberately, matching llm.RequestHash: it changes how
// much room the answer had, not what was asked, so a default moving must not
// invalidate every recording.
func key(body []byte) string {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err == nil {
		delete(m, "max_tokens")
		if norm, err := json.Marshal(m); err == nil {
			body = norm
		}
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])[:12]
}

// Transport returns a RoundTripper that replays recordings — or, with -update,
// calls through to next and records what comes back.
//
// A MISS without -update is a loud failure naming the path and quoting the
// request. Never a fallback: falling back is how an edited prompt comes to pass
// against a recording of the question it no longer asks.
func (c *Cassette) Transport(next http.RoundTripper) http.RoundTripper {
	return &cassetteTransport{store: c, next: next}
}

type cassetteTransport struct {
	store *Cassette
	next  http.RoundTripper
}

func (c *cassetteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := readBody(req)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(c.store.dir, key(body)+".json")

	if !Updating() {
		raw, err := os.ReadFile(path)
		if err != nil {
			// A miss is a test-authoring problem, so it is reported as a 400 the
			// caller surfaces as ErrRequest — loud, and not retried. t.Fatalf is
			// wrong here: RoundTrip runs on whatever goroutine the SDK is using.
			return jsonResponse(http.StatusBadRequest, []byte(fmt.Sprintf(
				`{"type":"error","error":{"type":"invalid_request_error","message":`+
					`"llmtest: no cassette at %s — re-run with -update to record it. request was: %s"}}`,
				path, jsonEscape(body)))), nil
		}
		var ex exchange
		if err := json.Unmarshal(raw, &ex); err != nil {
			return nil, fmt.Errorf("llmtest: cassette %s is unreadable: %w", path, err)
		}
		return jsonResponse(ex.Status, ex.Response), nil
	}

	resp, err := c.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.store.dir, 0o755); err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(exchange{
		Request: json.RawMessage(body), Status: resp.StatusCode, Response: json.RawMessage(respBody),
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return nil, err
	}
	c.store.t.Logf("cassette: recorded %s", path)
	resp.Body = io.NopCloser(bytes.NewReader(respBody))
	return resp, nil
}

// readBody consumes the request body and restores it, so the SDK's retry — which
// re-sends the same request — still has one to send.
func readBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func jsonResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func jsonEscape(b []byte) string {
	q, err := json.Marshal(string(b))
	if err != nil {
		return `"<unquotable>"`
	}
	return string(q[1 : len(q)-1])
}

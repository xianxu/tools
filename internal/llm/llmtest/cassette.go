package llmtest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/tools/internal/llm"
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
//
// Response is a STRING, not json.RawMessage: an SSE body is not JSON, so the
// RawMessage version could not marshal a streamed exchange at all — half the
// Client interface was unrecordable, and the marshalling failure surfaced as
// ErrUnavailable, the class every consumer absorbs silently. ContentType is
// recorded with it so replay answers as the service did.
type exchange struct {
	Request     json.RawMessage `json:"request"`
	Status      int             `json:"status"`
	ContentType string          `json:"content_type"`
	Response    string          `json:"response"`
}

// key identifies a recording.
//
// Derived from llm.RequestHash over the Request carried in the context — the SAME
// canonical form AssertGolden prints — so a prompt edit moves the golden and the
// cassette together. Hashing the wire body instead is what an earlier version did,
// and it collided: Task is never sent, so two Requests differing only in Task
// hashed identically and the second recording overwrote the first.
//
// The body hash remains as a fallback for a request that reaches the transport
// without a Request in context, which should not happen through llm.Client.
func key(ctx context.Context, body []byte) string {
	if r, ok := llm.RequestFromContext(ctx); ok {
		return llm.RequestHash(r)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err == nil {
		delete(m, "max_tokens")
		if norm, err := json.Marshal(m); err == nil {
			body = norm
		}
	}
	sum := sha256.Sum256(body)
	return "body-" + hex.EncodeToString(sum[:])[:12]
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
	path := filepath.Join(c.store.dir, key(req.Context(), body)+".json")

	if !Updating() {
		raw, err := os.ReadFile(path)
		if err != nil {
			// A miss is a test-authoring problem, so it is reported as a 400 the
			// caller surfaces as ErrRequest — loud, and not retried. t.Fatalf is
			// wrong here: RoundTrip runs on whatever goroutine the SDK is using.
			// body is passed RAW: harnessError marshals the envelope, which escapes
			// it once. Escaping here too produced \\\" chains in the very message
			// whose purpose is to be readable without recomputing a hash.
			return harnessError(path, fmt.Sprintf(
				"no cassette — re-run with -update to record it. request was: %s", body)), nil
		}
		var ex exchange
		if err := json.Unmarshal(raw, &ex); err != nil {
			return harnessError(path, fmt.Sprintf("cassette is unreadable: %v", err)), nil
		}
		return replay(ex), nil
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
		Request:     json.RawMessage(body),
		Status:      resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Response:    string(respBody),
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

func replay(ex exchange) *http.Response {
	ct := ex.ContentType
	if ct == "" {
		ct = "application/json"
	}
	return &http.Response{
		StatusCode: ex.Status,
		Header:     http.Header{"Content-Type": []string{ct}},
		Body:       io.NopCloser(strings.NewReader(ex.Response)),
	}
}

// harnessError reports a problem with the HARNESS, not the dependency.
//
// Deliberately a 400: it reaches the caller as ErrRequest, the class that must
// stay loud. An earlier version let a marshalling failure escape as a transport
// error, which classifies as ErrUnavailable — the class every consumer is built
// to absorb silently, so a broken cassette would have looked like flight mode.
func harnessError(path, msg string) *http.Response {
	body, _ := json.Marshal(map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    "invalid_request_error",
			"message": "llmtest [" + path + "]: " + msg,
		},
	})
	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

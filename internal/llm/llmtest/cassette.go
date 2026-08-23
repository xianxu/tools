package llmtest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/tools/internal/llm"
)

// Cassette is a real response, frozen.
//
// You cannot fake judgment. You can freeze a real answer and pin OUR HANDLING of
// it — which is what a consumer's tests actually need: that a well-formed veto
// verdict decodes, that a refusal is skipped, that a truncation is caught. What a
// cassette does NOT establish is that the model reliably produces that answer. It
// is one sample of a stochastic process, and the live conformance run is what
// detects drift.
//
// Keyed by llm.RequestHash over the SAME canonical form llmtest.Golden prints, so
// a prompt edit misses loudly rather than passing against a stale recording.
type Cassette struct {
	dir string
	t   *testing.T
}

// Cassettes opens the store under a consumer's testdata directory. Recordings
// live with the consumer that owns the prompt, exactly as goldens do.
func Cassettes(t *testing.T, dir string) *Cassette {
	t.Helper()
	return &Cassette{dir: filepath.Join(dir, "cassettes"), t: t}
}

// Path is where a request's recording lives. Exposed so a failure message can
// name the file rather than describe it.
func (c *Cassette) Path(r llm.Request) string {
	return filepath.Join(c.dir, r.Task+"-"+llm.RequestHash(r)+".json")
}

// Client returns a Client that replays recordings, or — with -update — calls
// through to live and records what comes back.
//
// A MISS without -update is a loud failure naming the task and the path. Never a
// fallback: falling back is how a prompt edit comes to pass against a recording
// of the question it no longer asks.
func (c *Cassette) Client(live llm.Client) llm.Client {
	return &cassetteClient{store: c, live: live}
}

type cassetteClient struct {
	store *Cassette
	live  llm.Client
}

func (c *cassetteClient) Complete(ctx context.Context, r llm.Request) (llm.Response, error) {
	path := c.store.Path(r)
	if !Updating() {
		raw, err := os.ReadFile(path)
		if err != nil {
			c.store.t.Fatalf("cassette: no recording for task %q at %s\n\n"+
				"the request renders as:\n%s\n"+
				"re-run with -update to record it against the live service",
				r.Task, path, llm.RenderRequest(r))
			return llm.Response{}, err
		}
		var rec recorded
		if err := json.Unmarshal(raw, &rec); err != nil {
			c.store.t.Fatalf("cassette: %s is unreadable: %v", path, err)
			return llm.Response{}, err
		}
		return rec.Response, rec.err()
	}

	if c.live == nil {
		c.store.t.Fatalf("cassette: -update needs a live client; none was supplied")
		return llm.Response{}, nil
	}
	resp, callErr := c.live.Complete(ctx, r)
	rec := recorded{Response: resp}
	if callErr != nil {
		rec.Err = callErr.Error()
	}
	if err := os.MkdirAll(c.store.dir, 0o755); err != nil {
		c.store.t.Fatalf("cassette: %v", err)
	}
	out, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		c.store.t.Fatalf("cassette: %v", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		c.store.t.Fatalf("cassette: %v", err)
	}
	c.store.t.Logf("cassette: recorded %s", path)
	return resp, callErr
}

// Stream is not recorded. A cassette exists so a consumer can assert on a real
// ANSWER; the streaming path's obligation is that deltas concatenate to that
// answer, which llmtest.Suite already holds against both backends. Recording SSE
// frame timing would model something no consumer asserts.
func (c *cassetteClient) Stream(ctx context.Context, r llm.Request, onDelta func(string)) (llm.Response, error) {
	resp, err := c.Complete(ctx, r)
	if err == nil && onDelta != nil {
		onDelta(resp.Text)
	}
	return resp, err
}

// recorded is what lands on disk: the response, plus the error text when the
// recorded call failed. Errors are recorded too — "this prompt gets refused" is
// exactly the kind of thing a consumer needs to handle and would otherwise have
// to invent.
type recorded struct {
	Response llm.Response `json:"response"`
	Err      string       `json:"error,omitempty"`
}

func (r recorded) err() error {
	if r.Err == "" {
		return nil
	}
	// The taxonomy is reconstructed from the stop reason rather than the text, so
	// a replayed refusal is still errors.Is(err, ErrRefused).
	if e := llm.ErrorForStop(r.Response.Stop); e != nil {
		return e
	}
	return fmt.Errorf("%w: %s", llm.ErrUnavailable, r.Err)
}

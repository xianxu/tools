package llmtest

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/llm"
)

// recordingClient drives a cassette in -update mode against the wire Fake, so
// "the live service" in these tests is itself a real HTTP exchange. Nothing here
// replaces llm.Client.
func recordingClient(t *testing.T, dir, base string) llm.Client {
	t.Helper()
	return llm.New(llm.Config{
		BaseURL: base, APIKey: "sk-test-1234567890", Model: "claude-opus-5",
		Effort: "high", MaxTokens: 8192, Timeout: 30 * time.Second,
		Transport: Cassettes(t, dir).Transport(http.DefaultTransport),
	})
}

func withUpdate(t *testing.T, fn func()) {
	t.Helper()
	*update = true
	defer func() { *update = false }() // defer, so a Fatal cannot leak -update
	fn()
}

func TestCassetteRecordsThenReplaysWithoutTheService(t *testing.T) {
	dir := t.TempDir()
	f := NewFake(t)
	f.Script("near-synonym", Reply{Text: `{"fits":true,"reason":"both describe servile flattery"}`})
	r := llm.Request{Task: "veto", Prompt: "Is obsequious a near-synonym?"}

	withUpdate(t, func() {
		if _, err := recordingClient(t, dir, f.URL).Complete(t.Context(), r); err != nil {
			t.Fatalf("record: %v", err)
		}
	})
	if len(f.Requests()) != 1 {
		t.Fatalf("recording made %d service calls, want 1", len(f.Requests()))
	}

	// Replay points at an address nothing listens on: if the replay reached the
	// network at all, this fails. That is the assertion.
	got, err := llm.New(llm.Config{
		BaseURL: "http://127.0.0.1:1", APIKey: "sk-test-1234567890", Model: "claude-opus-5",
		Timeout: 10 * time.Second, Transport: Cassettes(t, dir).Transport(nil),
	}).Complete(t.Context(), r)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if !strings.Contains(got.Text, "servile flattery") {
		t.Errorf("replayed text = %q", got.Text)
	}
	if len(f.Requests()) != 1 {
		t.Errorf("replay reached the service; calls = %d", len(f.Requests()))
	}
}

// A prompt edit moves the key, so the recording for the old question no longer
// answers the new one — loudly, and as ErrRequest, which a consumer must not
// absorb the way it absorbs an outage.
func TestAPromptEditMissesItsCassetteLoudly(t *testing.T) {
	dir := t.TempDir()
	f := NewFake(t)
	f.Script("obsequious", Reply{Text: `{"fits":true,"reason":"x"}`})

	withUpdate(t, func() {
		_, _ = recordingClient(t, dir, f.URL).Complete(t.Context(),
			llm.Request{Task: "veto", Prompt: "Is obsequious a near-synonym?"})
	})

	_, err := llm.New(llm.Config{
		BaseURL: "http://127.0.0.1:1", APIKey: "sk-test-1234567890", Model: "claude-opus-5",
		Timeout: 10 * time.Second, Transport: Cassettes(t, dir).Transport(nil),
	}).Complete(t.Context(), llm.Request{Task: "veto", Prompt: "Is obsequious a near-synonym, precisely?"})

	if err == nil {
		t.Fatal("an edited prompt silently replayed a recording of a different question")
	}
	if !errors.Is(err, llm.ErrRequest) {
		t.Errorf("err = %v, want ErrRequest — a missing recording is a test-authoring bug, not an outage", err)
	}
	if !strings.Contains(err.Error(), "-update") {
		t.Errorf("the miss does not say how to fix it: %v", err)
	}
}

// The taxonomy survives replay BECAUSE the status is replayed. A 400 recorded is
// a 400 replayed, so it reaches classifyStatus as ErrRequest — the one collapse
// the taxonomy exists to prevent, now structurally impossible rather than
// re-derived from a stored string.
func TestCassetteReplaysTheTaxonomyFromTheStatus(t *testing.T) {
	for _, c := range []struct {
		name   string
		status int
		want   error
	}{
		{"a bad request stays loud", 400, llm.ErrRequest},
		{"an outage stays absorbable", 503, llm.ErrUnavailable},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			f := NewFake(t)
			f.Script("x", Reply{Status: c.status})
			r := llm.Request{Task: "t", Prompt: "x"}

			withUpdate(t, func() {
				_, _ = recordingClient(t, dir, f.URL).Complete(t.Context(), r)
			})
			_, err := llm.New(llm.Config{
				BaseURL: "http://127.0.0.1:1", APIKey: "sk-test-1234567890", Model: "claude-opus-5",
				Timeout: 10 * time.Second, Transport: Cassettes(t, dir).Transport(nil),
			}).Complete(t.Context(), r)
			if !errors.Is(err, c.want) {
				t.Errorf("replayed err = %v, want %v", err, c.want)
			}
		})
	}
}

// The artifact records the QUESTION as well as the answer, so a reviewer reading
// a diff can see what was asked without recomputing a hash.
func TestCassetteOnDiskIsSelfDescribing(t *testing.T) {
	dir := t.TempDir()
	f := NewFake(t)
	f.Script("sycophantic", Reply{Text: `{"fits":true,"reason":"x"}`})
	withUpdate(t, func() {
		_, _ = recordingClient(t, dir, f.URL).Complete(t.Context(),
			llm.Request{Task: "veto", Prompt: "Is obsequious close to sycophantic?"})
	})

	files, _ := filepath.Glob(filepath.Join(dir, "cassettes", "*.json"))
	if len(files) != 1 {
		t.Fatalf("%d cassettes written, want 1", len(files))
	}
	raw, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var ex struct {
		Request  json.RawMessage `json:"request"`
		Status   int             `json:"status"`
		Response json.RawMessage `json:"response"`
	}
	if err := json.Unmarshal(raw, &ex); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, raw)
	}
	if !strings.Contains(string(ex.Request), "sycophantic") {
		t.Errorf("the recording does not show the question:\n%s", ex.Request)
	}
	if !strings.Contains(string(ex.Response), "fits") {
		t.Errorf("the recording does not show the answer:\n%s", ex.Response)
	}
	if ex.Status != 200 {
		t.Errorf("status = %d", ex.Status)
	}
}

// max_tokens is excluded from the key, matching llm.RequestHash: it changes how
// much room the answer had, not what was asked, so a default moving must not
// invalidate every recording in the repo.
func TestMaxTokensDoesNotMoveTheKey(t *testing.T) {
	a := []byte(`{"model":"m","max_tokens":1024,"messages":[{"role":"user","content":"x"}]}`)
	b := []byte(`{"model":"m","max_tokens":8192,"messages":[{"role":"user","content":"x"}]}`)
	if key(a) != key(b) {
		t.Error("max_tokens moves the cassette key")
	}
	c := []byte(`{"model":"m","max_tokens":1024,"messages":[{"role":"user","content":"y"}]}`)
	if key(a) == key(c) {
		t.Error("the prompt does NOT move the cassette key")
	}
}

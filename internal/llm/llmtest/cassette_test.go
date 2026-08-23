package llmtest

import (
	"context"
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
// The cassette key derives from llm.RequestHash — the SAME renderer AssertGolden
// prints — so a prompt edit moves both artifacts. Keying on the wire body instead
// collided: Task is never sent, so two Requests differing only in Task hashed
// identically and the second recording overwrote the first.
func TestTheKeyDerivesFromTheRequestNotTheBody(t *testing.T) {
	body := []byte(`{"model":"m","messages":[{"role":"user","content":"x"}]}`)
	a := llm.Request{Task: "veto", Model: "m", Prompt: "x"}
	b := llm.Request{Task: "author", Model: "m", Prompt: "x"} // same wire body, different task

	ctxA, ctxB := withReq(a), withReq(b)
	ka := key(ctxA, body)
	kb := key(ctxB, body)
	if ka == kb {
		t.Error("two tasks with the same wire body share a cassette; the second would overwrite the first")
	}
	// One renderer: the key is RequestHash of the request actually carried down,
	// which is the EFFECTIVE one — zero fields resolved against the Config, so
	// what is hashed is what was sent. Comparing against the caller's raw Request
	// would pass while two different default models shared a recording.
	effA, ok := llm.RequestFromContext(ctxA)
	if !ok {
		t.Fatal("no Request reached the transport")
	}
	if effA.Model == "" {
		t.Error("the context carried an unresolved model; the key would not distinguish two configs")
	}
	if ka != llm.RequestHash(effA) {
		t.Errorf("key = %s, want llm.RequestHash(effective) = %s — the golden and the cassette must share one renderer",
			ka, llm.RequestHash(effA))
	}
}

// max_tokens is excluded, matching llm.RequestHash: it changes how much room the
// answer had, not what was asked, so a default moving must not invalidate every
// recording in the repo.
func TestMaxTokensDoesNotMoveTheKey(t *testing.T) {
	a := llm.Request{Task: "veto", Model: "m", Prompt: "x", MaxTokens: 1024}
	b := a
	b.MaxTokens = 8192
	if key(withReq(a), nil) != key(withReq(b), nil) {
		t.Error("max_tokens moves the cassette key")
	}
	c := a
	c.Prompt = "y"
	if key(withReq(a), nil) == key(withReq(c), nil) {
		t.Error("the prompt does NOT move the cassette key")
	}
}

// A transport reached without a Request in context still keys deterministically,
// but says so in the filename — that path should not happen through llm.Client.
func TestKeyFallsBackToTheBodyWhenTheContextIsBare(t *testing.T) {
	body := []byte(`{"model":"m","messages":[{"role":"user","content":"x"}]}`)
	k := key(context.Background(), body)
	if !strings.HasPrefix(k, "body-") {
		t.Errorf("key = %s, want a body- prefix marking the fallback", k)
	}
}

func withReq(r llm.Request) context.Context {
	c := llm.New(llm.Config{BaseURL: "http://example", APIKey: "sk-test-1234567890"})
	var got context.Context
	// Complete attaches the Request to the context it passes down; capture it via
	// a transport rather than reimplementing the attachment here.
	_, _ = llm.New(llm.Config{
		BaseURL: "http://example", APIKey: "sk-test-1234567890", Timeout: time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			got = req.Context()
			return nil, errStop
		}),
	}).Complete(context.Background(), r)
	_ = c
	return got
}

var errStop = errors.New("stop")

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// BR-52: an SSE body is not JSON, so storing the response as json.RawMessage made
// half the Client interface unrecordable — and the marshalling failure surfaced as
// ErrUnavailable, the class every consumer absorbs silently.
func TestCassetteRecordsAndReplaysAStream(t *testing.T) {
	dir := t.TempDir()
	f := NewFake(t)
	r := llm.Request{Task: "qa", Prompt: "x"}

	var recorded strings.Builder
	withUpdate(t, func() {
		_, err := recordingClient(t, dir, f.URL).Stream(t.Context(), r,
			func(s string) { recorded.WriteString(s) })
		if err != nil {
			t.Fatalf("record stream: %v", err)
		}
	})
	if recorded.Len() == 0 {
		t.Fatal("recording produced no deltas")
	}

	var replayed strings.Builder
	got, err := llm.New(llm.Config{
		BaseURL: "http://127.0.0.1:1", APIKey: "sk-test-1234567890", Model: "claude-opus-5",
		Timeout: 10 * time.Second, Transport: Cassettes(t, dir).Transport(nil),
	}).Stream(t.Context(), r, func(s string) { replayed.WriteString(s) })
	if err != nil {
		t.Fatalf("replay stream: %v", err)
	}
	if replayed.String() != recorded.String() {
		t.Errorf("replayed %q, recorded %q", replayed.String(), recorded.String())
	}
	// The SDK's SSE parser ran on replay — that is the reason the cassette sits
	// beneath the seam at all.
	if got.Text != recorded.String() {
		t.Errorf("Text = %q, deltas = %q", got.Text, recorded.String())
	}
	var signed bool
	for _, b := range got.Blocks {
		if b.Type == "thinking" && b.Signature != "" {
			signed = true
		}
	}
	if !signed {
		t.Error("replay lost the thinking signature; the frames were not parsed as SSE")
	}
}

// A harness problem must not wear the dependency's absorbable class. A broken
// cassette that reads as ErrUnavailable looks exactly like flight mode.
func TestAHarnessFailureIsLoudNotAbsorbable(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "cassettes"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := llm.Request{Task: "veto", Model: "claude-opus-5", Prompt: "x"}
	// A corrupt recording at exactly the key this request will look for.
	path := filepath.Join(dir, "cassettes", llm.RequestHash(r)+".json")
	if err := os.WriteFile(path, []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := llm.New(llm.Config{
		BaseURL: "http://127.0.0.1:1", APIKey: "sk-test-1234567890", Model: "claude-opus-5",
		Timeout: 10 * time.Second, Transport: Cassettes(t, dir).Transport(nil),
	}).Complete(t.Context(), r)

	if !errors.Is(err, llm.ErrRequest) {
		t.Fatalf("err = %v, want ErrRequest — a broken harness must not read as an outage", err)
	}
	if errors.Is(err, llm.ErrUnavailable) {
		t.Error("a corrupt cassette was reported as the service being unavailable")
	}
}

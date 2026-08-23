package llmtest

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/tools/internal/llm"
)

// stubLive stands in for the live service during a -update recording. It is NOT
// a general test double — nothing replays through it — it exists so the record
// path can be exercised without a key.
type stubLive struct {
	resp  llm.Response
	err   error
	calls int
}

func (s *stubLive) Complete(context.Context, llm.Request) (llm.Response, error) {
	s.calls++
	return s.resp, s.err
}
func (s *stubLive) Stream(_ context.Context, _ llm.Request, onDelta func(string)) (llm.Response, error) {
	s.calls++
	if onDelta != nil {
		onDelta(s.resp.Text)
	}
	return s.resp, s.err
}

func recordThenReplay(t *testing.T, dir string, r llm.Request, live *stubLive) llm.Response {
	t.Helper()
	*update = true
	defer func() { *update = false }()
	c := Cassettes(t, dir).Client(live)
	got, err := c.Complete(t.Context(), r)
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	return got
}

func TestCassetteRecordsThenReplaysWithoutCallingLive(t *testing.T) {
	dir := t.TempDir()
	r := llm.Request{Task: "veto", Model: "claude-opus-5", Prompt: "Is obsequious a near-synonym?"}
	live := &stubLive{resp: llm.Response{
		Text: `{"fits":true,"reason":"both describe servile flattery"}`, Stop: "end_turn",
	}}

	recordThenReplay(t, dir, r, live)
	if live.calls != 1 {
		t.Fatalf("recording made %d live calls, want 1", live.calls)
	}

	// Replay: the live client is nil, so any call through would panic — which is
	// the assertion. A replay that reaches the service is not a replay.
	got, err := Cassettes(t, dir).Client(nil).Complete(t.Context(), r)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if !strings.Contains(got.Text, "servile flattery") {
		t.Errorf("replayed text = %q", got.Text)
	}
	if live.calls != 1 {
		t.Errorf("replay made a live call; calls = %d", live.calls)
	}
}

// THE property. A prompt edit moves the hash, so the recording for the old
// question no longer answers the new one — loudly, naming the file.
func TestAPromptEditMissesItsCassetteLoudly(t *testing.T) {
	dir := t.TempDir()
	original := llm.Request{Task: "veto", Model: "claude-opus-5", Prompt: "Is obsequious a near-synonym?"}
	recordThenReplay(t, dir, original, &stubLive{resp: llm.Response{Text: `{"fits":true}`, Stop: "end_turn"}})

	edited := original
	edited.Prompt = "Is obsequious a near-synonym of sycophantic, precisely?"

	fake := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Fatalf on a miss, so this runs on its own goroutine.
		_, _ = Cassettes(fake, dir).Client(nil).Complete(context.Background(), edited)
	}()
	<-done
	if !fake.Failed() {
		t.Error("an edited prompt silently replayed the recording of a different question")
	}
}

// A recorded refusal must replay as ErrRefused, not as a generic failure —
// otherwise a consumer's degradation path is never exercised by the recording
// that exists precisely to exercise it.
func TestCassetteReplaysTheTaxonomy(t *testing.T) {
	dir := t.TempDir()
	r := llm.Request{Task: "grade", Model: "claude-opus-5", Prompt: "grade this"}
	live := &stubLive{
		resp: llm.Response{Stop: "refusal", StopDetails: &llm.StopDetails{Type: "refusal", Category: "cyber"}},
		err:  llm.ErrRefused,
	}
	*update = true
	c := Cassettes(t, dir).Client(live)
	_, _ = c.Complete(t.Context(), r)
	*update = false

	_, err := Cassettes(t, dir).Client(nil).Complete(t.Context(), r)
	if !errors.Is(err, llm.ErrRefused) {
		t.Errorf("replayed err = %v, want ErrRefused", err)
	}
}

// The recording is a readable artifact, not an opaque blob: a human reviewing a
// diff must be able to see what the model said.
func TestCassetteOnDiskIsReadable(t *testing.T) {
	dir := t.TempDir()
	r := llm.Request{Task: "veto", Model: "claude-opus-5", Prompt: "p"}
	recordThenReplay(t, dir, r, &stubLive{resp: llm.Response{Text: `{"fits":true}`, Stop: "end_turn"}})

	raw, err := os.ReadFile(Cassettes(t, dir).Path(r))
	if err != nil {
		t.Fatal(err)
	}
	// The answer is a JSON string INSIDE the recording, so it appears escaped —
	// `"text": "{\"fits\":true}"`. Still legible in a diff, which is the point,
	// but it means the property to assert is the round trip rather than a literal
	// substring: what the model said survives the artifact unchanged.
	var back struct {
		Response struct {
			Text string `json:"Text"`
			Stop string `json:"Stop"`
		} `json:"response"`
	}
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("recording is not valid JSON: %v\n%s", err, raw)
	}
	if back.Response.Text != `{"fits":true}` {
		t.Errorf("recorded text = %q, want the model's answer verbatim", back.Response.Text)
	}
	if back.Response.Stop != "end_turn" {
		t.Errorf("recorded stop = %q", back.Response.Stop)
	}
	// And a human reading the diff must be able to see the answer at all.
	if !strings.Contains(string(raw), "fits") {
		t.Errorf("the answer is not visible in the artifact:\n%s", raw)
	}
}

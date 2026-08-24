package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

// askRig wires runAsk against the WIRE-LEVEL fake — an httptest server speaking
// the Anthropic protocol, not a stubbed Client. Placement decides what a test
// can see: a stubbed Client sits above a mis-serialised body, a dropped header,
// a retry that re-sends a consumed body, and an SSE frame the parser mishandles.
// The store is a real YAML one in a temp dir, so user-model.md is read through
// the production path rather than seeded into a field.
func askRig(t *testing.T) (deps, *llmtest.Fake, *store.YAML, string) {
	t.Helper()
	fake := llmtest.NewFake(t)
	dir := t.TempDir()
	st := store.NewYAML(dir, nil)
	d := testDeps(t)
	d.deck = st
	d.capture = noopCapturer{}
	d.clock = store.FixedClock(aDay)
	d.newLLM = llm.New
	d.getenv = func(k string) string {
		switch k {
		case "DEFINE_LLM_BASE_URL":
			return fake.URL
		case "DEFINE_LLM_API_KEY":
			return "test-key"
		}
		return ""
	}
	return d, fake, st, dir
}

var aDay = time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)

// streamCapture is the one committed streaming capture. Its content is a real
// answer to a real vocabulary question, which is what makes it usable here.
const streamCapture = "stream-sample.sse"

func TestAskStreamsAnAnswerWithTheDirectoryAsContext(t *testing.T) {
	d, fake, st, dir := askRig(t)
	// A committed capture, not invented text: the fake refuses a literal on a
	// streaming request, and the rule behind that refusal is the point — you
	// cannot fake judgment, but you can freeze a real answer and pin our
	// handling of it. This capture happens to BE an answer about "obsequious".
	fake.Script("", llmtest.Reply{Capture: streamCapture})
	// The directory holds a deck and a learner model — the two things that make
	// the answer adapted rather than merely generated.
	if err := st.Upsert(store.Word{Text: "ephemeral", FirstSeen: aDay, LastSeen: aDay, Lookups: 1}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "user-model.md"),
		[]byte("## Level\nC1, reads judicial opinions.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	sess := &session{current: "sycophantic", entry: "sycophantic | adjective | fawning"}
	var out, errb bytes.Buffer
	code := runAsk(t.Context(), d, options{}, sess, question{text: "difference to obsequious?"}, &out, &errb)

	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, errb.String())
	}
	if !strings.Contains(out.String(), "Obsequious") {
		t.Errorf("the answer did not reach stdout: %q", out.String())
	}
	// The RECORDED REQUEST is the assertion: the context reached the model.
	reqs := fake.Requests()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want 1", len(reqs))
	}
	body := reqs[0].System() + "\n" + reqs[0].Prompt()
	for _, want := range []string{"sycophantic", "ephemeral", "reads judicial opinions", "difference to obsequious?"} {
		if !strings.Contains(body, want) {
			t.Errorf("the prompt is missing %q:\n%s", want, body)
		}
	}
}

func TestAskRecordsAnAskedEvent(t *testing.T) {
	d, fake, st, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})

	sess := &session{current: "sycophantic"}
	var out, errb bytes.Buffer
	runAsk(t.Context(), d, options{}, sess, question{text: "is it pejorative?"}, &out, &errb)

	ev, err := st.Events(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 1 {
		t.Fatalf("events = %+v, want exactly the asked event", ev)
	}
	if ev[0].Kind != store.EventAsked || ev[0].Question != "is it pejorative?" || ev[0].Word != "sycophantic" {
		t.Errorf("event = %+v, want an asked event carrying the question and the word it followed", ev[0])
	}
}

func TestAskFollowUpCarriesThePreviousExchange(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture}, llmtest.Reply{Capture: streamCapture})

	sess := &session{current: "sycophantic"}
	var out, errb bytes.Buffer
	runAsk(t.Context(), d, options{}, sess, question{text: "is it pejorative?"}, &out, &errb)
	runAsk(t.Context(), d, options{}, sess, question{text: "give me three more examples"}, &out, &errb)

	reqs := fake.Requests()
	if len(reqs) != 2 {
		t.Fatalf("requests = %d, want 2", len(reqs))
	}
	second := reqs[1].Prompt()
	// The first QUESTION and a distinctive fragment of the first ANSWER: a
	// follow-up resolves against what was just said, so both halves must travel.
	for _, want := range []string{"is it pejorative?", "servile deference"} {
		if !strings.Contains(second, want) {
			t.Errorf("the follow-up prompt is missing %q:\n%s", want, second)
		}
	}
}

// The user's own Ctrl-C is not a failure to report.
//
// Asked of the CONTEXT, not the error: a cancelled request has no HTTP status,
// and mapError classifies statusless failures as ErrUnavailable — so a
// decode-the-error version prints "no model configured" as the answer to a
// keypress.
func TestAskSaysNothingWhenTheUserCancels(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture, Stall: true})

	ctx, cancel := context.WithCancel(t.Context())
	var out, errb bytes.Buffer
	go func() {
		// Cancel once the first delta has landed, so this is mid-stream rather
		// than before the request.
		for out.Len() == 0 {
		}
		cancel()
	}()
	code := runAsk(ctx, d, options{}, &session{}, question{text: "why?"}, &out, &errb)

	if errb.Len() != 0 {
		t.Errorf("a cancelled answer printed a diagnostic: %q", errb.String())
	}
	if code != 0 {
		t.Errorf("exit = %d, want 0 — the user's own keypress is not a failure", code)
	}
}

func TestAskDegradesWhenTheSeamIsUnavailable(t *testing.T) {
	d := testDeps(t)
	d.newLLM = llm.New
	d.getenv = func(string) string { return "" } // no key at all

	var out, errb bytes.Buffer
	code := runAsk(t.Context(), d, options{}, &session{}, question{text: "how so"}, &out, &errb)

	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if want := "define: no model configured; `how so` is not a word\n"; errb.String() != want {
		t.Errorf("stderr = %q, want %q", errb.String(), want)
	}
	// And a LOOKUP still works in the same session afterwards.
	var out2, errb2 bytes.Buffer
	if got := lookupAndRender(d, options{}, replCommand{kind: cmdDefine, word: "sycophantic"}, &out2, &errb2); got.code != 0 {
		t.Errorf("a lookup after an unavailable ask failed: %q", errb2.String())
	}
}

// ErrRequest is OUR bug — a bad prompt, a bad model — and must stay loud where
// ErrUnavailable degrades quietly. The taxonomy is the whole point of the seam.
func TestAskStaysLoudOnOurOwnBug(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Status: 400})

	var out, errb bytes.Buffer
	code := runAsk(t.Context(), d, options{}, &session{}, question{text: "why is it so"}, &out, &errb)

	if code == 0 {
		t.Error("a 400 exited 0")
	}
	if strings.Contains(errb.String(), "no model configured") {
		t.Errorf("our own bug was reported as an unavailable seam: %q", errb.String())
	}
}

// A nil seam is "no model wired", not a crash. llm.Resolve dereferences the
// getenv it is given, so an unwired deps used to panic inside the ask path —
// and every test that does not care about the model leaves it unset.
func TestAskWithNoSeamWiredDegrades(t *testing.T) {
	for _, tc := range []struct {
		name string
		d    deps
	}{
		{"nothing wired", testDeps(t)},
		{"getenv without a client", func() deps { d := testDeps(t); d.getenv = func(string) string { return "" }; return d }()},
		{"a client without getenv", func() deps { d := testDeps(t); d.newLLM = llm.New; return d }()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			if code := runAsk(t.Context(), tc.d, options{}, &session{}, question{text: "how so"}, &out, &errb); code != 1 {
				t.Errorf("exit = %d, want 1", code)
			}
			if !strings.Contains(errb.String(), "no model configured") {
				t.Errorf("stderr = %q", errb.String())
			}
		})
	}
}

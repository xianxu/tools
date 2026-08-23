//go:build conformance

package llm_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

// The typed-task layer against the real model.
//
// Everything else live goes through Complete/Stream with a hand-built schema;
// this is the only place Run[T] — what #10, #12, #13, #16 and #17 will actually
// call — is exercised end to end: a Go result type in, a populated struct out,
// through a reflected schema and a defensive decode.
//
// It asserts SHAPE, never the model's judgment: that the fields are populated,
// that the schema constrained the answer, that a nested result decodes. Whether
// the verdict is *correct* is #12's problem and is not testable here.
//
//	go test -tags conformance -run TypedTask ./internal/llm/
func TestTypedTaskAgainstTheLiveService(t *testing.T) {
	cfg, err := llm.Resolve(os.Getenv)
	if err != nil {
		t.Skipf("llm seam not configured: %v", err)
	}
	llmtest.SkipIfUnreachable(t, cfg.BaseURL)
	c := llm.New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Run("a flat result type round-trips", func(t *testing.T) {
		type verdict struct {
			Fits   bool   `json:"fits"`
			Reason string `json:"reason"`
		}
		got, err := llm.Run(ctx, c, llm.Task[verdict]{
			Name:   "veto-distractor",
			System: "You judge whether a candidate word could also correctly fill a cloze blank.",
			Prompt: `Sentence: "The board produced nothing but ______ agreement — every ` +
				`executive praised a plan they had privately called unworkable."` + "\n" +
				`Intended answer: sycophantic. Candidate distractor: obsequious.` + "\n" +
				`Could the candidate ALSO correctly fill the blank?`,
		})
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if strings.TrimSpace(got.Reason) == "" {
			t.Error("reason is empty; the required-field check should have rejected this")
		}
		// obsequious IS a near-synonym here, so a working pipeline says yes — but
		// that is the model's judgment, so it is LOGGED, not asserted. #12 is
		// where the quality of that judgment becomes a requirement.
		t.Logf("veto verdict: fits=%v reason=%q", got.Fits, got.Reason)
	})

	t.Run("a nested result type round-trips", func(t *testing.T) {
		// The shape #10 will actually author: an item with its options. Nested and
		// list-valued, which is where the required-field walk earned four rounds
		// of review — so it is the shape worth proving against a real model.
		type option struct {
			Word string `json:"word"`
			Why  string `json:"why"`
		}
		type item struct {
			Stem    string   `json:"stem"`
			Answer  string   `json:"answer"`
			Options []option `json:"options"`
		}
		got, err := llm.Run(ctx, c, llm.Task[item]{
			Name:   "author-cloze",
			System: "You write vocabulary practice items. The stem must entail its answer.",
			Prompt: "Write one cloze item for the word `sycophantic`, with exactly three " +
				"options: the answer plus two words that clearly do NOT fit. For each " +
				"option say why in a few words.",
		})
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		// Shape only, as the doc comment above promises. An earlier version
		// asserted the stem contained "___" — but how a model renders a blank is
		// its own choice (three underscores, five, an ellipsis, "[blank]"), so
		// that assertion failed on ordinary variation and made the suite a
		// judgment test the comment disclaimed. What the LAYER guarantees is that
		// the fields arrive populated.
		if strings.TrimSpace(got.Stem) == "" {
			t.Error("stem is empty")
		}
		if strings.TrimSpace(got.Answer) == "" {
			t.Error("answer is empty")
		}
		if len(got.Options) == 0 {
			t.Fatal("no options returned")
		}
		for i, o := range got.Options {
			// Every required field at every depth — the property the walk exists
			// for, now asserted against real output rather than a fixture.
			if strings.TrimSpace(o.Word) == "" || strings.TrimSpace(o.Why) == "" {
				t.Errorf("option[%d] = %+v: a required field came back empty", i, o)
			}
		}
		t.Logf("authored stem: %q", got.Stem)
		t.Logf("answer: %q, %d options", got.Answer, len(got.Options))
		for _, o := range got.Options {
			t.Logf("  %-16s %s", o.Word, o.Why)
		}
	})
}

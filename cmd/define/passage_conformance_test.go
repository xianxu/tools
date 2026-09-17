//go:build conformance

package main

// Live conformance for the passage question (#67, ARCH-MOCK).
//
// The fake replays a committed capture, so it cannot say whether the persona
// actually holds. The one that matters is the level default: "a curious reader
// going to college who does not have the background yet" is TWO settings in
// opposite directions, and a model that reads it as one simplifies — which in a
// vocabulary tool means paraphrasing away the very word being explained. A prompt
// line does not defend that; only a real answer does.
//
// Cadence: before a release that changes askSystem or passageSystem, and after
// the proxy's model changes.
//
//	go test -tags conformance -run Passage ./cmd/define/
//
// It SKIPS when the seam is unreachable, unless CONFORMANCE_STRICT is set; a
// reply that arrives and fails the checks is a failure, not a skip.

import (
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/tools/internal/conformance"
	"github.com/xianxu/tools/internal/llm"
)

func TestPassageAnswerKeepsTheHardWordAgainstTheLiveService(t *testing.T) {
	cfg, err := llm.Resolve(realGetenv)
	if err != nil {
		conformance.SkipOrFail(t, "no model configured", err)
	}
	client := llm.New(cfg)

	p := newPassage("The slow precession of the equinox points westward along the ecliptic, "+
		"so the pole star is only temporarily the pole star.", 0)
	m := markSet{}.toggle(p.spans(0)[2]).toggle(p.spans(0)[10])
	req := renderPassagePrompt(passageAsk{
		Context: askContext{Question: "What does this mean?"},
		Passage: p,
		Marks:   m,
	})

	// Streamed, because that is the production path: ask() streams so the reader
	// sees the answer arrive, and a conformance row that used a different call
	// would be checking a request nothing sends.
	var answer strings.Builder
	res, err := client.Stream(t.Context(), req, func(s string) { answer.WriteString(s) })
	if errors.Is(err, llm.ErrUnavailable) {
		conformance.SkipOrFail(t, "the model is unreachable", err)
	}
	if err != nil {
		t.Fatalf("the passage question failed against the live service: %v", err)
	}
	reply := answer.String()
	if reply == "" {
		reply = res.Text
	}
	got := strings.ToLower(reply)

	// THE FAILURE THE PERSONA INVITES. "Going to college but without the
	// background" read as one setting means "simplify", and the cheapest way to
	// simplify is to swap the hard word for an easy one — leaving a learner who
	// never sees the word they marked.
	for _, word := range []string{"precession", "ecliptic"} {
		if !strings.Contains(got, word) {
			t.Errorf("the answer never contains %q, the word that was marked — "+
				"the level default was read as 'simplify' and paraphrased it away:\n%s", word, reply)
		}
	}
	// And the other dial: the background is NOT assumed. An answer that explains
	// precession without saying what it is has stopped at the question's edge.
	if !strings.Contains(got, "wobble") && !strings.Contains(got, "axis") && !strings.Contains(got, "26,000") &&
		!strings.Contains(got, "26000") && !strings.Contains(got, "slowly") {
		t.Errorf("the answer names the marked words but explains nothing about them:\n%s", reply)
	}
	// The marks are answered TOGETHER with the passage, not glossed one by one.
	if len(reply) < 120 {
		t.Errorf("the answer is too short to have covered the passage and both marks:\n%s", reply)
	}
}

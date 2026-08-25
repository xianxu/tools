package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

func sampleAskContext() askContext {
	return askContext{
		Question:     "what's the difference to obsequious?",
		CurrentWord:  "sycophantic",
		CurrentEntry: "sycophantic | adjective | behaving or done in an obsequious way in order to gain advantage",
		SessionWords: []string{"sycophantic", "gaslighting"},
		DeckWords:    []string{"ephemeral", "defenestrate", "quokka"},
		UserModel:    "## Level\nC1, reads judicial opinions.\n",
		Turns:        []exchange{{Question: "is it pejorative?", Answer: "Yes — strongly."}},
	}
}

// The golden snapshots the REQUEST, not a string this test concatenates: it
// renders through the same llm.RenderRequest the transport hashes, so a field
// added to the prompt without thought shows up in its diff.
func TestRenderAskPrompt(t *testing.T) {
	llmtest.AssertGolden(t, "testdata", "ask-prompt", renderAskPrompt(sampleAskContext()))
}

// An absent section is OMITTED, never rendered empty. An empty "## The learner"
// header tells the model there IS a learner model and it is blank — a different
// claim from "we do not know this learner yet", and the one that produces a
// generic answer confidently.
func TestRenderAskPromptOmitsAbsentContext(t *testing.T) {
	full := renderAskPrompt(sampleAskContext())
	bare := renderAskPrompt(askContext{Question: "what does defenestrate mean?"})

	for _, tc := range []struct{ name, header, content string }{
		{"the current word", headerCurrent, "sycophantic"},
		{"the learner model", headerLearner, "reads judicial opinions"},
		{"the session", headerSession, "gaslighting"},
		{"the deck", headerDeck, "quokka"},
		{"the earlier exchange", headerEarlier, "is it pejorative?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(full.Prompt, tc.header) || !strings.Contains(full.Prompt, tc.content) {
				t.Errorf("the full prompt is missing %q / %q:\n%s", tc.header, tc.content, full.Prompt)
			}
			if strings.Contains(bare.Prompt, tc.header) {
				t.Errorf("the bare prompt rendered an empty %q section:\n%s", tc.header, bare.Prompt)
			}
		})
	}
	// The question itself is never optional.
	if !strings.Contains(bare.Prompt, "what does defenestrate mean?") {
		t.Errorf("the question is missing from the bare prompt:\n%s", bare.Prompt)
	}
}

func TestRecentTurnsBoundsTheTranscript(t *testing.T) {
	var many []exchange
	for i := range maxTurns + 3 {
		many = append(many, exchange{Question: string(rune('a' + i)), Answer: "x"})
	}
	got := recentTurns(many)

	if len(got) != maxTurns {
		t.Fatalf("kept %d turns, want %d", len(got), maxTurns)
	}
	// The OLDEST are dropped and order is preserved: a follow-up resolves
	// against what was just said, so the tail is the part that must survive.
	if got[0].Question != "d" || got[len(got)-1].Question != string(rune('a'+maxTurns+2)) {
		t.Errorf("kept %v — want the newest %d in order", got, maxTurns)
	}
	if short := recentTurns(many[:2]); len(short) != 2 {
		t.Errorf("a short transcript was truncated: %v", short)
	}
	if recentTurns(nil) != nil {
		t.Error("no turns must render as no section, not an empty one")
	}
}

// The task name keys the cassette and names the golden, so it is part of the
// contract rather than a label.
func TestAskRequestNamesItsTask(t *testing.T) {
	if got := renderAskPrompt(sampleAskContext()); got.Task != askTask {
		t.Errorf("Task = %q, want %q", got.Task, askTask)
	}
	var _ llm.Request = renderAskPrompt(askContext{})
}

// A pure ordering policy gets a table test, not a store and a wire fake.
//
// This is the shape BR-22's defect hid in: inlined in the IO gatherer, taking
// the wrong end of a newest-first slice was only observable through a real
// store, a real prompt and a recorded request.
func TestRecentDeckTakesTheNewestAndReadsForwards(t *testing.T) {
	deck := func(n int) []store.Word { // newest first, as Store.Deck() promises
		var out []store.Word
		for i := n; i >= 1; i-- {
			out = append(out, store.Word{Text: fmt.Sprintf("w%d", i)})
		}
		return out
	}
	for _, tc := range []struct {
		name string
		deck []store.Word
		n    int
		want []string
	}{
		{"shorter than the bound, reversed to reading order", deck(3), 5, []string{"w1", "w2", "w3"}},
		{"longer: the NEWEST survive, oldest-first", deck(5), 3, []string{"w3", "w4", "w5"}},
		{"exactly the bound", deck(3), 3, []string{"w1", "w2", "w3"}},
		{"empty deck renders no section", nil, 3, []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := recentDeck(tc.deck, tc.n)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

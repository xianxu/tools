package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xianxu/tools/internal/llm/llmtest"
)

// splitWordInCapture finds a word the committed capture delivers across TWO
// deltas, and returns it.
//
// Derived, not hardcoded. The fake cannot serve invented streamed text —
// misapplied() rejects a scripted literal on a streaming request and serveStream
// always replays the committed capture — so the test's seeded word has to come
// FROM the capture. Hardcoding one would rot silently the next time the capture
// is re-recorded; this fails loudly instead.
func splitWordInCapture(t *testing.T, name string) string {
	t.Helper()

	var deltas []string
	sc := bufio.NewScanner(bytes.NewReader(llmtest.Capture(t, name)))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line, ok := strings.CutPrefix(sc.Text(), "data: ")
		if !ok {
			continue
		}
		var ev struct {
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if json.Unmarshal([]byte(line), &ev) != nil || ev.Delta.Type != "text_delta" {
			continue
		}
		deltas = append(deltas, ev.Delta.Text)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scanning the capture: %v", err)
	}

	// A boundary splits a word when the delta before it ends in a word rune and
	// the one after starts with one. wordRuns decides that, so this test and
	// production cannot disagree about where a word is.
	full := strings.Join(deltas, "")
	at := 0
	for _, d := range deltas[:max(0, len(deltas)-1)] {
		at += len(d)
		for _, r := range wordRuns(full) {
			if r.start < at && at < r.end {
				return full[r.start:r.end]
			}
		}
	}
	t.Fatalf("no word in %s is split across deltas — re-record the capture or pick another; "+
		"skipping here would let this test go quietly inert", name)
	return ""
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// M3's headline behaviour: a deck word in a streamed answer highlights even when
// the model delivers it in two pieces.
func TestStreamedAnswerHighlightsAWordSplitAcrossDeltas(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})
	word := splitWordInCapture(t, streamCapture)
	d.vocab = vocab(word)

	var out, errOut bytes.Buffer
	code := runAsk(t.Context(), d, options{color: true}, &session{}, question{text: "what is obsequious?"}, &out, &errOut)

	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), knownOn+word) {
		t.Errorf("%q arrives split across deltas and was not highlighted: %q", word, out.String())
	}
}

// Every exit path from runAsk must leave the answer complete on `out`.
//
// HONEST SCOPE, because the deferred Flush deserves a straight account rather
// than the appearance of coverage. runAsk returns on five paths, and on four of
// them the hold is already released before the return — either the answer ends
// in a newline of its own, or runAsk's trailing Fprintln supplies one, and a
// newline is not a phrase gap so it resolves the hold. Deleting the deferred
// Flush therefore leaves this table green.
//
// The one path where it would bite is `default:` (ErrRequest/ErrMalformed),
// which returns BEFORE that trailing newline while partial text sits held. It is
// not reachable through llmtest today: Status short-circuits before any body, so
// a scripted failure never delivers partial text, and both partial-then-fail
// shapes the fake does offer (Stall, JunkFrame) classify as ErrTruncated and
// fall through to the newline. Reaching it needs a fake that can serve some
// deltas and then a malformed non-truncation frame.
//
// So the Flush stays deferred on structure, not on a passing test: five returns
// is four chances to forget, and a path added later gets it for free. What this
// table does pin is that no path leaves text dangling.
func TestEveryStreamExitPathFlushes(t *testing.T) {
	word := splitWordInCapture(t, streamCapture)

	for _, tc := range []struct {
		name   string
		script func(f *llmtest.Fake)
		cancel bool
	}{
		{"clean completion", func(f *llmtest.Fake) {
			f.Script("", llmtest.Reply{Capture: streamCapture})
		}, false},
		{"interrupted mid-stream", func(f *llmtest.Fake) {
			f.Script("", llmtest.Reply{Capture: streamCapture})
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, fake, _, _ := askRig(t)
			tc.script(fake)
			d.vocab = vocab(word)
			ctx := t.Context()
			if tc.cancel {
				var cancel func()
				ctx, cancel = context.WithCancel(ctx)
				cancel() // already cancelled: the stream never delivers
			}

			var out, errOut bytes.Buffer
			runAsk(ctx, d, options{color: true}, &session{}, question{text: "what is obsequious?"}, &out, &errOut)

			// Nothing may be left held. The writer's own accounting is the
			// observable: after a flush there is no pending text.
			if got := out.String(); tc.cancel && got != "" && !strings.HasSuffix(got, "\n") {
				t.Errorf("interrupted path left text unterminated (unflushed?): %q", got)
			}
			if !tc.cancel && !strings.Contains(out.String(), knownOn+word) {
				t.Errorf("clean path did not flush the highlight: %q", out.String())
			}
		})
	}
}

// The answer's tail reaches `out` — the end-to-end version of the unit-level
// "a word at the very end still highlights after flush" (highlightwriter_test).
func TestTheLastWordOfAnAnswerSurvives(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})
	d.vocab = vocab("obsequious")

	var out, errOut bytes.Buffer
	runAsk(t.Context(), d, options{color: true}, &session{}, question{text: "q?"}, &out, &errOut)

	// The capture's final characters, which only a flush can emit.
	if !strings.Contains(out.String(), "given insincerely.") {
		t.Errorf("the answer's tail was never flushed: %q", out.String())
	}
}

// -no-color must put no escape into a streamed answer at all.
func TestStreamedAnswerCarriesNoEscapesWithoutColour(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})
	d.vocab = vocab(splitWordInCapture(t, streamCapture))

	var out, errOut bytes.Buffer
	runAsk(t.Context(), d, options{color: false}, &session{}, question{text: "q?"}, &out, &errOut)

	if strings.Contains(out.String(), "\x1b") {
		t.Errorf("-no-color emitted an escape into the answer: %q", out.String())
	}
}

// The nesting order, pinned. The raw loop wraps stdout in crlfWriter before
// calling ask, and runAsk wraps that in a highlightWriter — so highlighting sees
// LOGICAL text and CRLF translation applies to the final bytes, including the
// escape sequences highlighting inserted.
//
// Inverting the order would translate before highlighting: the highlighter would
// then see "\r\n" where it expects "\n", and a match ending at a line break
// would be decided against different bytes than production analyses elsewhere.
func TestHighlightingNestsInsideCRLFTranslation(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})
	d.vocab = vocab("obsequious")

	var raw bytes.Buffer
	var errOut bytes.Buffer
	runAsk(t.Context(), d, options{color: true}, &session{}, question{text: "q?"},
		&crlfWriter{w: &raw}, &errOut)

	got := raw.String()
	if !strings.Contains(got, knownOn+"Obsequious") {
		t.Errorf("the highlight did not survive CRLF translation: %q", got)
	}
	// Every newline is a full CRLF: the translation ran OUTSIDE, over the
	// highlighted bytes rather than before them.
	if strings.Count(got, "\n") != strings.Count(got, "\r\n") {
		t.Errorf("a bare newline escaped CRLF translation: %q", got)
	}
}

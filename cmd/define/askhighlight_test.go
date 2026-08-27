package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
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
		name      string
		script    func(f *llmtest.Fake)
		cancel    bool
		wantEmpty bool
	}{
		{"clean completion", func(f *llmtest.Fake) {
			f.Script("", llmtest.Reply{Capture: streamCapture})
		}, false, false},
		{"cancelled before any delta", func(f *llmtest.Fake) {
			f.Script("", llmtest.Reply{Capture: streamCapture})
		}, true, true},
		// JunkFrame rather than Stall: both classify as ErrTruncated, but Stall
		// goes silent WITHOUT closing and the row then waits out the client's
		// 30s timeout. Same path, two orders of magnitude cheaper.
		{"truncated mid-answer", func(f *llmtest.Fake) {
			f.Script("", llmtest.Reply{Capture: streamCapture, JunkFrame: true})
		}, false, false},
		// NOT a distinct exit path, and the first version of this row claimed it
		// was: instrumenting every return showed `stalled upstream` arrives with
		// err=nil and ctxErr=nil, i.e. the same `case err == nil:` branch as
		// clean completion. Kept as a transport-shape row, labelled for what it
		// is — a table whose rows do not say which branch they reach is how a
		// missing path hides behind a full-looking list.
		{"stalled upstream (same branch as clean)", func(f *llmtest.Fake) {
			f.Script("", llmtest.Reply{Capture: streamCapture, Stall: true})
		}, false, false},
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

			got := out.String()
			if tc.wantEmpty {
				// The row's whole claim. An already-cancelled context returns
				// before any delta, so guarding an assertion on `got != ""`
				// would make this row assert NOTHING — which is what the first
				// version did while its comment claimed otherwise.
				if got != "" {
					t.Errorf("a stream that never delivered wrote %q", got)
				}
				return
			}
			if got == "" {
				t.Fatal("no output at all — this row would assert nothing")
			}
			if !strings.HasSuffix(got, "\n") {
				t.Errorf("the answer was left unterminated: %q", got)
			}
			// The split word only arrives on a stream that ran to completion; a
			// truncated one stops before it. Asserting it unconditionally would
			// be asserting the fixture, not the behaviour.
			if strings.Contains(stripANSI(got), word) && !strings.Contains(got, knownOn+word) {
				t.Errorf("the word arrived but was not highlighted: %q", got)
			}
		})
	}
}

// The genuinely distinct fifth branch: interrupted WITH text already delivered,
// at color=true so the highlight path is live. The exit-path table's cancel row
// cancels before any delta and so returns with nothing held; this one returns
// through `if ctx.Err() != nil` with a partial answer, which is the branch that
// writes the closing newline and records the exchange.
func TestInterruptedMidStreamKeepsWhatArrivedHighlighted(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture, Stall: true})
	d.vocab = vocab("Obsequious")

	ctx, cancel := context.WithCancel(t.Context())
	var out, errOut bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- runAsk(ctx, d, options{color: true}, &session{}, question{text: "q?"}, &out, &errOut)
	}()
	// The stall delivers its first text block, then goes silent; cancelling then
	// exercises the interrupt branch with text already held/emitted.
	<-time.After(200 * time.Millisecond)
	cancel()
	<-done

	if got := out.String(); got != "" && !strings.Contains(got, knownOn+"Obsequious") {
		t.Errorf("an interrupted answer lost its highlight: %q", got)
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

	// The capture's final characters. NOT "which only a flush can emit" — the
	// first version of this comment said that and mutation refutes it: deleting
	// the deferred Flush leaves this green, because runAsk's trailing Fprintln
	// supplies a newline that releases the hold. What this pins is that the tail
	// arrives, which is worth pinning on its own.
	if !strings.Contains(out.String(), "given insincerely.") {
		t.Errorf("the answer's tail never arrived: %q", out.String())
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

// The enumeration's axis is entry path x RENDER SURFACE, not entry path alone.
//
// TestEveryEntryPathHighlightsDefinitions covers the DEFINITION surface. M3 added
// a second one — the answer stream — and did not widen that table, which is how a
// mutant that keeps the nil/colour gate but drops d.vocab.Load() passed the whole
// suite: in production it means a streamed answer never highlights on piped stdin
// or one-shot, since only runEditor loads. That is M2's shipped Critical, one
// surface over, and the guard written to prevent it did not reach.
//
// Every row drives an UNLOADED store vocabulary, because a pre-filled one begins
// after the hop that fills it.
func TestEveryEntryPathHighlightsAnswers(t *testing.T) {
	// The capture answers a question about "obsequious" and says the word.
	const inAnswer = "Obsequious"

	newDeck := func(t *testing.T) Vocabulary {
		t.Helper()
		st := store.NewMem()
		if err := st.Upsert(store.Word{Text: inAnswer}); err != nil {
			t.Fatal(err)
		}
		return newStoreVocabulary(st, nil) // deliberately NOT loaded
	}

	for _, tc := range []struct {
		name string
		run  func(t *testing.T, d deps, opt options, out, errOut *bytes.Buffer)
	}{
		{"one-shot: define '?question'", func(t *testing.T, d deps, opt options, out, errOut *bytes.Buffer) {
			ask(t.Context(), d, opt, &session{}, out, errOut, question{text: "what is obsequious?", forced: true})
		}},
		{"piped stdin", func(t *testing.T, d deps, opt options, out, errOut *bytes.Buffer) {
			replLines(t.Context(), nil, d, opt, strings.NewReader("?what is obsequious\n"), out, errOut, true, false)
		}},
		{"raw editor", func(t *testing.T, d deps, opt options, out, errOut *bytes.Buffer) {
			runEditor(t.Context(), scriptKeys("?what is obsequious\r"), nil, d, opt,
				func(run func()) error { run(); return nil }, func() {}, out, errOut)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, fake, _, _ := askRig(t)
			fake.Script("", llmtest.Reply{Capture: streamCapture})
			d.dict = testDict(t)
			d.audio = noAudioSource{}
			d.player = &fakePlayer{}
			d.stdinIsTerminal = func() bool { return true }
			d.vocab = newDeck(t)

			var out, errOut bytes.Buffer
			tc.run(t, d, options{color: true, times: 1, locale: "us", tty: true, noAudio: true}, &out, &errOut)

			if !strings.Contains(out.String(), knownOn+inAnswer) {
				t.Errorf("no highlight in the answer on this entry path: %q", out.String())
			}
		})
	}
}

// BR-33: the poison report itself. runAsk now checks Flush's error and says so
// on stderr, because the writer poisons on first failure and one failed write
// otherwise drops the REST of an answer in silence. Untested, that decision was
// just a comment.
func TestAPoisonedWriterIsReported(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})
	d.vocab = vocab("obsequious")

	var errOut bytes.Buffer
	runAsk(t.Context(), d, options{color: true}, &session{},
		question{text: "q?"}, &shortWriter{limit: 2}, &errOut)

	if !strings.Contains(errOut.String(), "could not be fully written") {
		t.Errorf("a failed write was swallowed: stderr = %q", errOut.String())
	}
}

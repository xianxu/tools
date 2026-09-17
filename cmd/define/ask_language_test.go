package main

import (
	"bytes"
	"context"
	"github.com/xianxu/tools/internal/llm"
	"strings"
	"testing"

	"github.com/xianxu/tools/internal/llm/llmtest"
)

func TestAskAnnotatedCaptureReachesCleanHistoryAndTint(t *testing.T) {
	for _, color := range []bool{true, false} {
		d, fake, _, _ := askRig(t)
		d.lang = "es"
		fake.Script("", llmtest.Reply{Capture: "stream-language.sse"})
		var out, errOut bytes.Buffer
		sess := &session{}
		code := runAsk(t.Context(), d, options{color: color, tintBackground: languageDark}, sess, question{text: "Explain buenos días"}, &out, &errOut)
		if code != 0 {
			t.Fatalf("code %d: %s", code, errOut.String())
		}
		if len(sess.turns) != 1 {
			t.Fatalf("history %+v", sess.turns)
		}
		stored := sess.turns[0].Answer
		if !strings.Contains(stored, "buenos días") || !strings.Contains(stored, "good morning") || strings.Contains(stored, "[lang=") || strings.Contains(stored, "[/lang]") {
			t.Fatalf("dirty/incomplete history %q", stored)
		}
		if strings.TrimRight(stripEscapes(out.String()), "\n") != strings.TrimRight(stored, "\n") {
			t.Fatal("display and stored answer diverged")
		}
		if strings.Contains(out.String(), languageDark) {
			t.Fatalf("width-zero plain stream tinted: %q", out.String())
		}
		if !color && strings.Contains(out.String(), "\x1b") {
			t.Fatal("no-color leaked styling")
		}
		req := fake.Requests()[0]
		if !strings.Contains(req.Prompt(), "## Study language\nes") || !strings.Contains(req.System(), "[lang=es]") {
			t.Fatal("language/annotation contract absent from actual request")
		}
	}
}

// The wrapper only cancels at a real captured delta boundary; all request,
// protocol and semantic content still pass through the stateful wire fake.
type cancelLanguageStream struct {
	llm.Client
	cancel context.CancelFunc
}

func (c cancelLanguageStream) Stream(ctx context.Context, r llm.Request, delta func(string)) (llm.Response, error) {
	var seen strings.Builder
	return c.Client.Stream(ctx, r, func(s string) {
		if ctx.Err() != nil {
			return
		}
		seen.WriteString(s)
		delta(s)
		if strings.Contains(seen.String(), "[lang=en]The") {
			c.cancel()
		}
	})
}
func TestAskAnnotatedCancellationFlushesBeforeHistory(t *testing.T) {
	d, fake, _, _ := askRig(t)
	d.lang = "en"
	fake.Script("", llmtest.Reply{Capture: "stream-language.sse"})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	d.newLLM = func(cfg llm.Config) llm.Client { return cancelLanguageStream{Client: llm.New(cfg), cancel: cancel} }
	var out, errOut bytes.Buffer
	sess := &session{}
	runAsk(ctx, d, options{color: true, width: 20, tintBackground: languageDark}, sess, question{text: "Explain buenos días"}, &out, &errOut)
	if len(sess.turns) != 1 || !strings.HasPrefix(sess.turns[0].Answer, "The") {
		t.Fatalf("missing partial history: %+v / %s", sess.turns, errOut.String())
	}
	if strings.Contains(out.String(), "[lang") {
		t.Fatalf("annotation markers leaked into the answer: %q", out.String())
	}
	// An unterminated passage KEEPS the language its opening marker announced
	// (#72). It was painted the instant it arrived, which is the whole of the
	// change, and painted text cannot be taken back — least of all here, where
	// the close marker a cancellation never delivers is what used to release it.
	// This assertion is the accepted consequence, pinned where it happens.
	if !strings.Contains(out.String(), languageDark) {
		t.Fatalf("unterminated segment lost the ownership it announced: %q", out.String())
	}
	// A tinted row is padded to the terminal width, because that is how a
	// background covers a row (#65) — so the comparison is against the row's
	// TEXT, not its cells. The claim is unchanged: what was displayed is what was
	// recorded.
	if shown := trimRowPadding(stripEscapes(out.String())); shown != sess.turns[0].Answer {
		t.Fatalf("partial display/history differ: %q vs %q", shown, sess.turns[0].Answer)
	}
}

// trimRowPadding drops the trailing cells a background fill writes, per row.
func trimRowPadding(s string) string {
	rows := strings.Split(s, "\n")
	for i, row := range rows {
		rows[i] = strings.TrimRight(row, " ")
	}
	return strings.TrimRight(strings.Join(rows, "\n"), "\n")
}

const longPassageCapture = "stream-long-passage.sse"

// countingSink records how many separate writes reached the screen, which is the
// difference between an answer that streams and one that lands.
//
// The buffer is a FIELD, not embedded, and that is load-bearing rather than
// style. Embedded, bytes.Buffer promotes WriteString — and io.WriteString, which
// is what the wrap writer uses, prefers it. Every byte then bypassed the
// counting Write, and this sink reported one write for a 1262-byte answer that
// had in fact arrived in pieces: a double that hid precisely the thing it was
// built to measure.
type countingSink struct {
	writes  int
	largest int
	buf     bytes.Buffer
}

func (c *countingSink) Write(p []byte) (int, error) {
	c.writes++
	if len(p) > c.largest {
		c.largest = len(p)
	}
	return c.buf.Write(p)
}

func (c *countingSink) Len() int       { return c.buf.Len() }
func (c *countingSink) String() string { return c.buf.String() }

// deltaObserver watches the production stream without altering it: it reports
// the index of each answer delta AFTER the real writers have handled it.
//
// The plan named Reply{AfterText, FinishRelease} for this, and that barrier
// cannot carry the assertion: it holds the stream after the FIRST text delta,
// which in this capture is exactly "[lang=es]" — a marker with no prose. The
// sink is legitimately empty there whether or not the bug is present, so the
// barrier would have asserted nothing. Observing deltas as they are processed
// does assert it, and needs no clock.
type deltaObserver struct {
	llm.Client
	seen func(n int)
}

func (o deltaObserver) Stream(ctx context.Context, r llm.Request, onDelta func(string)) (llm.Response, error) {
	n := 0
	return o.Client.Stream(ctx, r, func(s string) {
		onDelta(s)
		n++
		o.seen(n)
	})
}

// TestALongPassageReachesTheScreenInPieces is this issue at the level the
// operator reported it: not "is the text right" but "when does it arrive".
//
// Every earlier test asserts the answer's final content, and a decoder that
// buffers a whole passage produces byte-identical final content — which is
// exactly how a ten-second blank screen passed a green suite.
//
// TWO properties, because one of them alone was not enough. ORDERING — text
// reached the screen while deltas were still arriving — is the defect itself; a
// granularity check alone passes an implementation that buffers the passage and
// releases it rune-by-rune at the close marker, which is still a blank screen.
// GRANULARITY — no single write carries the passage — is what rules out the
// original shape, 818 bytes in one go. Both run at width 0 and at a real
// terminal width, since the wrap writer's row-commit path is a second hold and
// only the second case puts it on the tested path.
func TestALongPassageReachesTheScreenInPieces(t *testing.T) {
	assertDominantPassage(t, longPassageCapture)

	for _, tc := range []struct {
		name    string
		width   int
		largest int // a row at width 100 carries its own cells plus styling
	}{
		{"piped", 0, 64},
		{"terminal", 100, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, fake, _, _ := askRig(t)
			d.lang = "es"
			fake.Script("", llmtest.Reply{Capture: longPassageCapture})
			var sink countingSink
			firstVisible, deltas := 0, 0
			d.newLLM = func(cfg llm.Config) llm.Client {
				return deltaObserver{Client: llm.New(cfg), seen: func(n int) {
					deltas = n
					if firstVisible == 0 && sink.Len() > 0 {
						firstVisible = n
					}
				}}
			}
			var errOut bytes.Buffer
			code := runAsk(t.Context(), d, options{color: true, width: tc.width, tintBackground: languageDark}, &session{}, question{text: "sicofante vs obsequioso"}, &sink, &errOut)

			if code != 0 {
				t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
			}
			if firstVisible == 0 || firstVisible*4 > deltas {
				t.Fatalf("nothing reached the screen until delta %d of %d; the answer is still being withheld until the stream is over", firstVisible, deltas)
			}
			if sink.largest > tc.largest {
				t.Fatalf("a %d-byte answer arrived in %d writes, the largest %d; a buffered passage lands in one",
					sink.Len(), sink.writes, sink.largest)
			}
			t.Logf("%d bytes in %d writes, largest %d, first visible at delta %d of %d",
				sink.Len(), sink.writes, sink.largest, firstVisible, deltas)
		})
	}
}

// assertDominantPassage keeps the test above from going quietly inert.
//
// It is only a test of streaming while the fixture is a SINGLE LONG passage —
// the shape stream-language.sse does not have, which is why that capture could
// never have caught this. Re-record with the command in
// internal/llm/llmtest/testdata/README.md.
func assertDominantPassage(t *testing.T, name string) {
	t.Helper()
	// The SAME predicate the conformance recorder applies before promoting a
	// capture, over the same decoded denominator: one rule, one statement, so the
	// guard that admits a capture and the guard that depends on it cannot
	// disagree about the same file.
	decoded := decodedChunks(strings.Join(captureDeltas(t, name), ""), 0)
	if !dominantPassage(decoded) {
		longest, total := longestPassage(decoded)
		t.Fatalf("%s: longest passage %d of %d decoded bytes — no longer a single-passage answer, so it cannot exhibit the buffering this test exists for", name, longest, total)
	}
}

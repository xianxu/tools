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

// TestALongPassageReachesTheScreenInPieces is this issue at the level the
// operator reported it: not "is the text right" but "when does it arrive".
//
// Every earlier test asserts the answer's final content, and a decoder that
// buffers a whole passage produces byte-identical final content — which is
// exactly how a ten-second blank screen passed a green suite. Write COUNT is the
// observable that separates them, and it needs no clock: a buffered passage
// reaches the sink in one write whenever it arrives, a streamed one in hundreds.
func TestALongPassageReachesTheScreenInPieces(t *testing.T) {
	assertDominantPassage(t, longPassageCapture)

	d, fake, _, _ := askRig(t)
	d.lang = "es"
	fake.Script("", llmtest.Reply{Capture: longPassageCapture})
	var sink countingSink
	var errOut bytes.Buffer
	code := runAsk(t.Context(), d, options{color: true}, &session{}, question{text: "sicofante vs obsequioso"}, &sink, &errOut)

	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	// THE LARGEST SINGLE WRITE, not the write count, because it is the direct
	// expression of the defect: the buffer handed the passage over in one piece.
	// Measured against the buffer restored, this capture arrives in 11 writes,
	// the largest 818 bytes — so a count threshold would catch it HERE, but only
	// by accident of how much untagged prose this particular answer carries.
	// Neutral text streamed per rune before this issue too, so any count is a
	// number about the answer rather than about the mechanism.
	if sink.largest > 64 {
		t.Fatalf("a %d-byte answer arrived in %d writes, the largest %d bytes; a buffered passage lands in one",
			sink.Len(), sink.writes, sink.largest)
	}
	t.Logf("%d bytes in %d writes, largest %d", sink.Len(), sink.writes, sink.largest)
}

// assertDominantPassage keeps the test above from going quietly inert.
//
// It is only a test of streaming while the fixture is a SINGLE LONG passage —
// the shape stream-language.sse does not have, which is why that capture could
// never have caught this. Re-record with the command in
// internal/llm/llmtest/testdata/README.md.
func assertDominantPassage(t *testing.T, name string) {
	t.Helper()
	full := strings.Join(captureDeltas(t, name), "")
	longest := 0
	for _, r := range annotatedRegions(full) {
		if n := r[1] - r[0]; n > longest {
			longest = n
		}
	}
	// Half the RAW capture, which is a stricter bar than it looks: full still
	// carries the marker bytes the decoder strips, so this ratio understates the
	// share of visible text the passage covers (818 of 1261 decoded bytes, 65%,
	// when this capture was recorded).
	if longest*2 < len(full) {
		t.Fatalf("%s: longest passage %d of %d raw bytes — no longer a single-passage answer, so it cannot exhibit the buffering this test exists for", name, longest, len(full))
	}
}

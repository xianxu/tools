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
	if strings.Contains(out.String(), languageDark) || strings.Contains(out.String(), "[lang") {
		t.Fatalf("unfinished segment should be clean and neutral: %q", out.String())
	}
	if strings.TrimRight(stripEscapes(out.String()), "\n") != sess.turns[0].Answer {
		t.Fatal("partial display/history differ")
	}
}

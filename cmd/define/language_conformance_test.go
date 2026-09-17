//go:build conformance

package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/conformance"
	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

type languageCaptureTransport struct{ wire bytes.Buffer }

func (c *languageCaptureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		resp.Body = &languageCaptureBody{Reader: io.TeeReader(resp.Body, &c.wire), Closer: resp.Body}
	}
	return resp, nil
}

type languageCaptureBody struct {
	io.Reader
	io.Closer
}

// Production prompt and decoder, with real semantic assertions. Optional
// DEFINE_LANGUAGE_CAPTURE writes only response SSE, never request credentials.
func TestLanguageAnnotationsAgainstLiveService(t *testing.T) {
	cfg, err := llm.Resolve(realGetenv)
	if err != nil {
		conformance.SkipOrFail(t, "model not configured", err)
	}
	llmtest.SkipIfUnreachable(t, cfg.BaseURL)
	capture := &languageCaptureTransport{}
	cfg.Transport = capture
	client := llm.New(cfg)
	req := renderAskPrompt(askContext{Language: "es", Question: "Explain the Spanish phrase buenos días in English. Include buenos días as an inline Spanish phrase inside your English explanation, and give its English translation good morning."})
	var got languageText
	decoder := newLanguageDecoder(func(v languageText) {
		off := len(got.text)
		got.text += v.text
		for _, sp := range v.spans {
			sp.start += off
			sp.end += off
			got.spans = append(got.spans, sp)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	_, err = client.Stream(ctx, req, decoder.Write)
	decoder.Finish()
	if err != nil {
		t.Fatal(err)
	}
	var es, en strings.Builder
	for _, sp := range got.spans {
		switch sp.lang {
		case "es":
			es.WriteString(got.text[sp.start:sp.end])
		case "en":
			en.WriteString(got.text[sp.start:sp.end])
		}
	}
	if !strings.Contains(strings.ToLower(es.String()), "buenos días") || !strings.Contains(strings.ToLower(en.String()), "good morning") {
		t.Fatalf("ownership failed: Spanish=%q English=%q full=%q", es.String(), en.String(), got.text)
	}
	if strings.Contains(got.text, "[lang=") || strings.ContainsAny(got.text, "\x1b") {
		t.Fatalf("metadata/control leak: %q", got.text)
	}
	if path := os.Getenv("DEFINE_LANGUAGE_CAPTURE"); path != "" {
		if err := os.WriteFile(path, capture.wire.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("Spanish: %q; English: %q", es.String(), en.String())
}

// TestLongPassageStreamsAgainstLiveService records the shape stream-language.sse
// cannot have (#72).
//
// That capture was recorded from a question asking for an inline foreign phrase,
// so its passages close every few deltas. A capture is evidence only for the
// shape its recording conditions elicit — testdata/README.md's own rule — and the
// shape this issue is about is the ORDINARY one: a monolingual answer the model
// writes as a single passage, whose close marker arrives only when generation
// ends. Against the buffer this replaces, that was ten seconds of blank screen,
// and no committed capture could show it.
//
// So this records a long single passage and refuses to promote one that is not.
func TestLongPassageStreamsAgainstLiveService(t *testing.T) {
	cfg, err := llm.Resolve(realGetenv)
	if err != nil {
		conformance.SkipOrFail(t, "model not configured", err)
	}
	llmtest.SkipIfUnreachable(t, cfg.BaseURL)
	capture := &languageCaptureTransport{}
	cfg.Transport = capture
	client := llm.New(cfg)
	// A STUDY-LANGUAGE session, and that is the point rather than a detail.
	// Measured while recording this: with English selected the model often leaves
	// its English prose untagged and annotates only a foreign fragment — one run
	// produced a longest span of 3 bytes — so the hold this issue removes was
	// STOCHASTIC in an English session. Where the prose is the annotated
	// language, it is reliable, which is both the honest reproduction and the
	// stable fixture.
	req := renderAskPrompt(askContext{
		Language: "es",
		Question: "¿Cuál es la diferencia entre «sicofante» y «obsequioso»? Responde en tres párrafos cortos, en español.",
	})
	// MERGING accumulator, unlike the sibling above, which only concatenates by
	// language and so does not care. Since #72 a passage is emitted as it
	// arrives — one span per rune — so "the longest span" is 3 bytes for every
	// answer ever written unless adjacent runs of one language are joined back
	// up. The first version of this test measured the unmerged spans and
	// declared a perfectly good Spanish answer fragmented.
	var got languageText
	decoder := newLanguageDecoder(func(v languageText) {
		off := len(got.text)
		got.text += v.text
		for _, sp := range v.spans {
			sp.start += off
			sp.end += off
			if n := len(got.spans); n > 0 && got.spans[n-1].end == sp.start && got.spans[n-1].lang == sp.lang {
				got.spans[n-1].end = sp.end
				continue
			}
			got.spans = append(got.spans, sp)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	if _, err = client.Stream(ctx, req, decoder.Write); err != nil {
		t.Fatal(err)
	}
	decoder.Finish()

	// The shape this capture exists to demonstrate, asserted before it is
	// written: one passage carrying most of the answer. A fragmented answer is a
	// legitimate model output and a useless fixture here.
	longest := 0
	for _, sp := range got.spans {
		if n := sp.end - sp.start; n > longest {
			longest = n
		}
	}
	if len(got.text) < 500 || longest*10 < len(got.text)*6 {
		t.Fatalf("not a long-passage answer: longest span %d of %d bytes — rerun, or pick a question that elicits one passage", longest, len(got.text))
	}
	if strings.Contains(got.text, "[lang=") || strings.ContainsAny(got.text, "\x1b") {
		t.Fatalf("metadata/control leak: %q", got.text)
	}
	if path := os.Getenv("DEFINE_LONG_PASSAGE_CAPTURE"); path != "" {
		if err := os.WriteFile(path, capture.wire.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("longest passage %d of %d bytes", longest, len(got.text))
}

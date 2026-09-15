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

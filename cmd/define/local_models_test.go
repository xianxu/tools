package main

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

func discoveredClient(f *llmtest.Fake) func(llm.Config) llm.Client {
	return func(c llm.Config) llm.Client { c.BaseURL = f.URL; c.AutoModel = true; return llm.New(c) }
}

func TestLLMCheckReportsDiscoveredProviderAndModel(t *testing.T) {
	for _, status := range []int{0, 400} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			f := llmtest.NewFake(t)
			f.SetCatalog([]llm.ModelInfo{{ID: "gemini-3-flash", OwnedBy: "antigravity"}, {ID: "gemini-3.1-pro-high", OwnedBy: "antigravity"}})
			f.Script("PONG", llmtest.Reply{Text: "PONG", Status: status})
			var out, errs bytes.Buffer
			code := runLLMCheck(t.Context(), envOf(nil), discoveredClient(f), &out, &errs, options{})
			if (code == 0) != (status == 0) {
				t.Fatalf("code %d errors %s", code, errs.String())
			}
			if !strings.Contains(out.String(), "gemini-3-flash") || !strings.Contains(out.String(), "antigravity") {
				t.Fatalf("selected model absent: %s", out.String())
			}
			if strings.Contains(out.String(), "claude-opus-5") {
				t.Fatalf("reported old default: %s", out.String())
			}
		})
	}
}

func TestReflectRecordsDiscoveredModel(t *testing.T) {
	d, f, st, _ := reflectRig(t, 14)
	f.SetCatalog([]llm.ModelInfo{{ID: "gpt-5.6", OwnedBy: "openai"}})
	f.Script("", llmtest.Reply{Text: reflectReply})
	d.newLLM = discoveredClient(f)
	var out, errs bytes.Buffer
	if code := runReflect(t.Context(), d, options{tty: true}, &out, &errs); code != 0 {
		t.Fatalf("code %d: %s", code, errs.String())
	}
	got, err := st.UserModel()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "gpt-5.6") || strings.Contains(got, "claude-opus-5") {
		t.Fatalf("incorrect provenance: %s", got)
	}
	if len(f.CatalogRequests()) != 1 || len(f.Requests()) != 1 {
		t.Fatal("unexpected discovery/inference calls")
	}
}

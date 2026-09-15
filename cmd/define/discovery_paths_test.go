package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

func TestAskUsesDiscoveredModel(t *testing.T) {
	for _, model := range []llm.ModelInfo{{ID: "claude-opus-5", OwnedBy: "anthropic"}, {ID: "gpt-5.6", OwnedBy: "openai"}, {ID: "gemini-3-flash", OwnedBy: "antigravity"}} {
		t.Run(model.OwnedBy, func(t *testing.T) {
			d, f, _, _ := askRig(t)
			f.SetCatalog([]llm.ModelInfo{model})
			f.Script("", llmtest.Reply{Capture: streamCapture})
			d.newLLM = discoveredClient(f)
			var out, errs bytes.Buffer
			if code := runAsk(t.Context(), d, options{}, &session{current: "sycophantic"}, question{text: "difference to obsequious?"}, &out, &errs); code != 0 {
				t.Fatalf("exit %d: %s", code, errs.String())
			}
			requests := f.Requests()
			if len(f.CatalogRequests()) != 1 || len(requests) != 1 {
				t.Fatalf("catalog=%d inference=%d", len(f.CatalogRequests()), len(requests))
			}
			r := requests[0]
			if r.Body["model"] != model.ID || !r.Streaming() {
				t.Fatalf("incorrect selected streaming request: %+v", r.Body)
			}
			if !strings.Contains(r.Prompt(), "difference to obsequious?") || !strings.Contains(r.Prompt(), "sycophantic") {
				t.Fatalf("question context lost: %s", r.Prompt())
			}
			if r.Schema() != nil {
				t.Fatal("freeform ask unexpectedly requires JSON")
			}
			if !strings.Contains(out.String(), "Obsequious") {
				t.Fatalf("streamed answer absent: %s", out.String())
			}
		})
	}
}

func TestHarvestUsesDiscoveredModelAndCacheRemainsOffline(t *testing.T) {
	for _, model := range []llm.ModelInfo{{ID: "claude-opus-5", OwnedBy: "anthropic"}, {ID: "gpt-5.6", OwnedBy: "openai"}, {ID: "gemini-3-flash", OwnedBy: "antigravity"}} {
		t.Run(model.OwnedBy, func(t *testing.T) {
			d, f, st := harvestRig(t, 3)
			f.SetCatalog([]llm.ModelInfo{model})
			scriptAll(f, 12)
			d.newLLM = discoveredClient(f)
			var out, errs bytes.Buffer
			if code := runHarvest(t.Context(), d, options{tty: true}, harvestOptions{}, &out, &errs); code != 0 {
				t.Fatalf("exit %d: %s", code, errs.String())
			}
			requests := f.Requests()
			if len(f.CatalogRequests()) != 1 || len(requests) == 0 {
				t.Fatalf("catalog=%d inference=%d", len(f.CatalogRequests()), len(requests))
			}
			for _, r := range requests {
				if r.Body["model"] != model.ID {
					t.Fatalf("selected ID lost: %v", r.Body["model"])
				}
				schema := r.Schema()
				if schema == nil {
					t.Fatal("harvest schema missing from wire")
				}
				if model.OwnedBy != "anthropic" {
					encoded, err := json.MarshalIndent(schema, "", "  ")
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(r.System(), string(encoded)) || !strings.Contains(r.System(), "Return only valid JSON") {
						t.Fatalf("schema instructions missing: %s", r.System())
					}
				}
			}
			deck, err := st.Deck()
			if err != nil {
				t.Fatal(err)
			}
			for _, word := range deck {
				facts, err := st.WordFacts(word.Text)
				if err != nil || !facts.Harvested() {
					t.Fatalf("word %s was not persisted: %+v %v", word.Text, facts, err)
				}
			}
			// A fresh client and endpoint distinguish lazy discovery from reuse of a
			// cached client: reading already-bought material must make no network call.
			offline := llmtest.NewFake(t)
			offline.ConfigureCatalog(llmtest.CatalogOptions{Status: 401})
			d.newLLM = discoveredClient(offline)
			out.Reset()
			errs.Reset()
			if code := runHarvest(t.Context(), d, options{tty: true}, harvestOptions{}, &out, &errs); code != 0 {
				t.Fatalf("cached exit %d: %s", code, errs.String())
			}
			if strings.ContainsAny(out.String(), "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏") {
				t.Fatal("cached harvest drew activity")
			}
			if len(offline.CatalogRequests()) != 0 || len(offline.Requests()) != 0 {
				t.Fatalf("cached harvest used network: discovery=%d inference=%d", len(offline.CatalogRequests()), len(offline.Requests()))
			}
		})
	}
}

func TestDictionaryLookupDoesNoDiscovery(t *testing.T) {
	d := testDeps(t)
	f := llmtest.NewFake(t)
	f.ConfigureCatalog(llmtest.CatalogOptions{Status: 401})
	d.getenv = envFor(f.URL)
	d.newLLM = discoveredClient(f)
	var out, errs bytes.Buffer
	if code := run(t.Context(), []string{"-no-audio", "sycophantic"}, d, strings.NewReader(""), &out, &errs); code != 0 {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	if !strings.Contains(out.String(), "adjective") || !strings.Contains(out.String(), "obsequious") {
		t.Fatalf("dictionary definition absent: %s", out.String())
	}
	if len(f.CatalogRequests()) != 0 || len(f.Requests()) != 0 {
		t.Fatalf("dictionary lookup used network: discovery=%d inference=%d", len(f.CatalogRequests()), len(f.Requests()))
	}
}

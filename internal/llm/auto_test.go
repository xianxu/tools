package llm_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

func autoConfig(f *llmtest.Fake) llm.Config {
	return llm.Config{BaseURL: f.URL, APIKey: "test-key", AutoModel: true, Timeout: time.Second}
}

func TestAutoClientDiscoversOnlyOnUseAndPinsSelection(t *testing.T) {
	f := llmtest.NewFake(t)
	f.SetCatalog([]llm.ModelInfo{{ID: "gpt-5.6", OwnedBy: "openai"}})
	c := llm.New(autoConfig(f))
	if len(f.CatalogRequests()) != 0 || llm.SelectionOf(c).ID != "" {
		t.Fatal("construction discovered a model")
	}
	if _, err := c.Complete(t.Context(), llm.Request{Prompt: "hello"}); err != nil {
		t.Fatal(err)
	}
	f.SetCatalog([]llm.ModelInfo{{ID: "gpt-5.6", OwnedBy: "openai"}, {ID: "claude-opus-5", OwnedBy: "anthropic"}})
	if _, err := c.Stream(t.Context(), llm.Request{Prompt: "hello"}, nil); err != nil {
		t.Fatal(err)
	}
	if len(f.CatalogRequests()) != 1 {
		t.Fatalf("catalog calls=%d", len(f.CatalogRequests()))
	}
	if got := llm.SelectionOf(c); got.ID != "gpt-5.6" || got.Provider != "openai" {
		t.Fatalf("selection=%+v", got)
	}
	for _, r := range f.Requests() {
		if r.Body["model"] != "gpt-5.6" {
			t.Fatalf("wire model=%v", r.Body["model"])
		}
	}
}

func TestAutoClientExplicitRequestBypassesDiscovery(t *testing.T) {
	f := llmtest.NewFake(t)
	f.ConfigureCatalog(llmtest.CatalogOptions{Status: 401})
	c := llm.New(autoConfig(f))
	if _, err := c.Complete(t.Context(), llm.Request{Model: "claude-opus-5", Prompt: "hi"}); err != nil {
		t.Fatal(err)
	}
	if len(f.CatalogRequests()) != 0 || llm.SelectionOf(c).ID != "" {
		t.Fatal("override mutated automatic selection")
	}
}

func TestAutoClientOwnerCancellationAllowsLaterRetry(t *testing.T) {
	f := llmtest.NewFake(t)
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	f.ConfigureCatalog(llmtest.CatalogOptions{Started: started, Release: release})
	c := llm.New(autoConfig(f))
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { _, err := c.Complete(ctx, llm.Request{}); done <- err }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("owner=%v", err)
	}
	f.ConfigureCatalog(llmtest.CatalogOptions{})
	if _, err := c.Complete(t.Context(), llm.Request{}); err != nil {
		t.Fatal(err)
	}
	close(release)
	if len(f.CatalogRequests()) != 2 {
		t.Fatalf("catalog calls=%d", len(f.CatalogRequests()))
	}
}

func TestAutoClientFailureDoesNotPinOrSwitchModel(t *testing.T) {
	f := llmtest.NewFake(t)
	f.SetCatalog(nil)
	c := llm.New(autoConfig(f))
	if _, err := c.Complete(t.Context(), llm.Request{}); !errors.Is(err, llm.ErrUnavailable) {
		t.Fatalf("empty=%v", err)
	}
	if len(f.Requests()) != 0 {
		t.Fatal("empty catalog fell back to old model")
	}
	f.SetCatalog([]llm.ModelInfo{{ID: "gpt-5.6", OwnedBy: "openai"}, {ID: "claude-opus-5", OwnedBy: "anthropic"}})
	f.Script("", llmtest.Reply{Status: 400})
	if _, err := c.Complete(t.Context(), llm.Request{}); !errors.Is(err, llm.ErrRequest) {
		t.Fatalf("inference=%v", err)
	}
	if len(f.Requests()) != 1 {
		t.Fatal("inference failure tried another model")
	}
	if llm.SelectionOf(c).ID != "claude-opus-5" {
		t.Fatal("inference failure changed selection")
	}
}

// deadlineTransport observes actual HTTP context budgets while retaining the
// shared fake's routing, model validation, and completion/streaming behavior.
type deadlineTransport struct {
	mu        sync.Mutex
	deadlines map[string]time.Time
}

func (d *deadlineTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	deadline, ok := r.Context().Deadline()
	if !ok {
		return nil, errors.New("request has no deadline")
	}
	d.mu.Lock()
	d.deadlines[r.URL.Path] = deadline
	d.mu.Unlock()
	return http.DefaultTransport.RoundTrip(r)
}

func TestAutoClientDiscoveryAndInferenceShareDeadline(t *testing.T) {
	for _, stream := range []bool{false, true} {
		name := "complete"
		if stream {
			name = "stream"
		}
		t.Run(name, func(t *testing.T) {
			f := llmtest.NewFake(t)
			f.SetCatalog([]llm.ModelInfo{{ID: "gpt-5.6", OwnedBy: "openai"}})
			transport := &deadlineTransport{deadlines: make(map[string]time.Time)}
			cfg := autoConfig(f)
			cfg.Transport = transport
			c := llm.New(cfg)
			var err error
			if stream {
				_, err = c.Stream(t.Context(), llm.Request{Prompt: "hello"}, nil)
			} else {
				_, err = c.Complete(t.Context(), llm.Request{Prompt: "hello"})
			}
			if err != nil {
				t.Fatal(err)
			}
			transport.mu.Lock()
			defer transport.mu.Unlock()
			discovery, ok := transport.deadlines["/v1/models"]
			if !ok {
				t.Fatal("missing discovery")
			}
			inference, ok := transport.deadlines["/v1/messages"]
			if !ok {
				t.Fatal("missing inference")
			}
			if !inference.Equal(discovery) {
				t.Fatalf("inference reset total budget: discovery=%v inference=%v", discovery, inference)
			}
		})
	}
}

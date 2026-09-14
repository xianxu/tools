package llmtest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/llm"
)

func TestCatalogState(t *testing.T) {
	f := NewFake(t)
	catalog := []llm.ModelInfo{{ID: "gpt-5.6", OwnedBy: "openai"}}
	f.SetCatalog(catalog)
	catalog[0].ID = "mutated"
	f.ConfigureCatalog(CatalogOptions{APIKey: "secret"})
	resp, err := http.Get(f.URL + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Fatal(resp.Status)
	}
	req, _ := http.NewRequest("GET", f.URL+"/v1/models", nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var got struct {
		Data []llm.ModelInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got.Data) != 1 || got.Data[0].ID != "gpt-5.6" {
		t.Fatalf("%+v", got)
	}
	if len(f.CatalogRequests()) != 2 || len(f.Requests()) != 0 {
		t.Fatal("histories mixed")
	}
	if !f.acceptsModel("gpt-5.6") || f.acceptsModel("claude-opus-5") {
		t.Fatal("catalog not governing models")
	}
	f.SetCatalog(nil)
	if f.acceptsModel("gpt-5.6") {
		t.Fatal("empty catalog retained model")
	}
}

func TestCatalogControlsAndCancellation(t *testing.T) {
	f := NewFake(t)
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	f.ConfigureCatalog(CatalogOptions{Started: started, Release: release, Status: 503, Body: []byte("scripted catalog")})
	result := make(chan *http.Response, 1)
	errs := make(chan error, 1)
	go func() {
		resp, err := http.Get(f.URL + "/v1/models")
		if err != nil {
			errs <- err
			return
		}
		result <- resp
	}()
	select {
	case <-started:
	case err := <-errs:
		t.Fatal(err)
	case <-time.After(time.Second):
		t.Fatal("catalog never started")
	}
	if len(f.CatalogRequests()) != 1 {
		t.Fatal("request not recorded before blocking")
	}
	select {
	case resp := <-result:
		resp.Body.Close()
		t.Fatal("release ignored")
	default:
	}
	close(release)
	select {
	case resp := <-result:
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != 503 || string(body) != "scripted catalog" {
			t.Fatalf("%s %s %v", resp.Status, body, err)
		}
	case err := <-errs:
		t.Fatal(err)
	case <-time.After(time.Second):
		t.Fatal("release did not unblock")
	}
	started = make(chan struct{}, 1)
	f.ConfigureCatalog(CatalogOptions{Started: started, Release: make(chan struct{})})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", f.URL+"/v1/models", nil)
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		errs <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("cancel test never started")
	}
	cancel()
	select {
	case err := <-errs:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not unblock")
	}
}

func TestCatalogAdvertisedModelOnWire(t *testing.T) {
	f := NewFake(t)
	f.SetCatalog([]llm.ModelInfo{{ID: "gpt-5.6", OwnedBy: "openai"}})
	for _, tc := range []struct {
		model  string
		status int
	}{{"gpt-5.6", 200}, {"claude-opus-5", 502}} {
		resp, err := http.Post(f.URL+"/v1/messages", "application/json", strings.NewReader(`{"model":"`+tc.model+`","messages":[]}`))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.status {
			t.Fatalf("%s: %s", tc.model, resp.Status)
		}
	}
	if len(f.Requests()) != 2 || len(f.CatalogRequests()) != 0 {
		t.Fatal("histories mixed")
	}
}

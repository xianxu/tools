package llmtest

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/xianxu/tools/internal/llm"
)

// CatalogOptions controls catalog transport independently from inference.
// Started receives a notification before Release is awaited. Use a buffered
// channel or an active receiver. Context cancellation and cleanup unblock both.
// A nil Body renders the configured catalog; a nonnil Body is served verbatim.
type CatalogOptions struct {
	APIKey  string
	Status  int
	Body    []byte
	Started chan<- struct{}
	Release <-chan struct{}
}
type catalogState struct {
	models   []llm.ModelInfo
	requests []Recorded
	options  CatalogOptions
}

func defaultCatalog() catalogState {
	state := catalogState{}
	for id := range knownModels {
		state.models = append(state.models, llm.ModelInfo{ID: id, OwnedBy: "anthropic"})
	}
	sort.Slice(state.models, func(i, j int) bool { return state.models[i].ID < state.models[j].ID })
	return state
}

// SetCatalog replaces the advertised catalog and accepted inference model IDs.
func (f *Fake) SetCatalog(models []llm.ModelInfo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.catalog.models = append([]llm.ModelInfo{}, models...)
}

// ConfigureCatalog atomically replaces the catalog transport controls.
func (f *Fake) ConfigureCatalog(options CatalogOptions) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if options.Body != nil {
		options.Body = append([]byte{}, options.Body...)
	}
	f.catalog.options = options
}

// CatalogRequests returns GET history independently of inference Requests.
func (f *Fake) CatalogRequests() []Recorded {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := append([]Recorded(nil), f.catalog.requests...)
	for i := range out {
		out[i].Headers = out[i].Headers.Clone()
	}
	return out
}
func (f *Fake) acceptsModel(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, model := range f.catalog.models {
		if model.ID == id {
			return true
		}
	}
	return false
}
func (f *Fake) serveCatalog(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.catalog.requests = append(f.catalog.requests, Recorded{Path: r.URL.Path, Headers: r.Header.Clone()})
	options := f.catalog.options
	models := append([]llm.ModelInfo{}, f.catalog.models...)
	f.mu.Unlock()
	if options.Started != nil {
		select {
		case options.Started <- struct{}{}:
		case <-r.Context().Done():
			return
		case <-f.closing:
			return
		}
	}
	if options.Release != nil {
		select {
		case <-options.Release:
		case <-r.Context().Done():
			return
		case <-f.closing:
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if options.APIKey != "" && r.Header.Get("Authorization") != "Bearer "+options.APIKey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	status := options.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if options.Body != nil {
		_, _ = w.Write(options.Body)
		return
	}
	_ = json.NewEncoder(w).Encode(struct {
		Data []llm.ModelInfo `json:"data"`
	}{models})
}

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const catalogByteLimit = 1 << 20
const catalogEntryLimit = 4096

// discoverModels performs one bounded authenticated catalog read. Network error
// strings and response bodies are deliberately excluded from public errors.
func discoverModels(ctx context.Context, c Config) (ModelSelection, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/v1/models", nil)
	if err != nil {
		return ModelSelection{}, fmt.Errorf("%w: model discovery URL is invalid", ErrRequest)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	client := http.Client{Transport: c.Transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return ModelSelection{}, discoveryNetworkError(ctx)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return ModelSelection{}, fmt.Errorf("%w: model discovery authentication failed; check proxy and DEFINE_LLM_API_KEY", ErrUnavailable)
		}
		return ModelSelection{}, classifyStatus(resp.StatusCode, fmt.Errorf("model discovery HTTP status %d", resp.StatusCode))
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, catalogByteLimit+1))
	if err != nil {
		return ModelSelection{}, discoveryNetworkError(ctx)
	}
	models, err := parseModels(b)
	if err != nil {
		return ModelSelection{}, err
	}
	return SelectModel(models)
}
func discoveryNetworkError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: model discovery: %w", ErrUnavailable, err)
	}
	return fmt.Errorf("%w: model discovery network failure", ErrUnavailable)
}

// parseModels validates the complete untrusted envelope before ranking any entry.
// Unknown fields are intentionally ignored for forward-compatible catalogs.
func parseModels(b []byte) ([]ModelInfo, error) {
	malformed := func() ([]ModelInfo, error) {
		return nil, fmt.Errorf("%w: invalid or oversized model discovery catalog", ErrMalformed)
	}
	if len(b) > catalogByteLimit {
		return malformed()
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(b, &envelope); err != nil {
		return malformed()
	}
	data := bytes.TrimSpace(envelope.Data)
	if len(data) == 0 || data[0] != '[' {
		return malformed()
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(data, &entries); err != nil || len(entries) > catalogEntryLimit {
		return malformed()
	}
	models := make([]ModelInfo, 0, len(entries))
	for _, entry := range entries {
		var model ModelInfo
		if err := json.Unmarshal(entry, &model); err != nil || !catalogText(model.ID, 256) || !catalogText(model.OwnedBy, 64) {
			return malformed()
		}
		models = append(models, model)
	}
	return models, nil
}
func catalogText(s string, limit int) bool {
	if len(s) == 0 || len(s) > limit {
		return false
	}
	for i := range len(s) {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

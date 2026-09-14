package llm

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type discoveryTransport func(*http.Request) (*http.Response, error)

func (f discoveryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDiscoverBoundaries(t *testing.T) {
	tests := []struct {
		name, body string
		status     int
		want       error
	}{
		{"valid", `{"data":[{"id":"claude-opus-5","owned_by":"anthropic"}],"other":true}`, 200, nil},
		{"missing", `{}`, 200, ErrMalformed}, {"null", `{"data":null}`, 200, ErrMalformed},
		{"top array", `[]`, 200, ErrMalformed}, {"entry null", `{"data":[null]}`, 200, ErrMalformed},
		{"missing owner", `{"data":[{"id":"claude-opus-5"}]}`, 200, ErrMalformed},
		{"control", `{"data":[{"id":"bad\n","owned_by":"anthropic"}]}`, 200, ErrMalformed},
		{"trailing", `{"data":[]} {}`, 200, ErrMalformed},
		{"empty", `{"data":[]}`, 200, ErrUnavailable},
		{"oversize", strings.Repeat(" ", 1<<20) + `{"data":[]}`, 200, ErrMalformed},
		{"auth", "SECRET", 401, ErrUnavailable}, {"forbidden", "SECRET", 403, ErrUnavailable},
		{"server", "SECRET", 503, ErrUnavailable}, {"redirect", "SECRET", 302, ErrRequest},
		{"request", "SECRET", 400, ErrRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			cfg := Config{BaseURL: "http://localhost:8317", APIKey: "SECRET", Transport: discoveryTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/v1/models" || r.Method != "GET" || r.Header.Get("Authorization") != "Bearer SECRET" {
					t.Errorf("incorrect request: %s %s", r.Method, r.URL.Path)
				}
				deadline, ok := r.Context().Deadline()
				if !ok || time.Until(deadline) > 5*time.Second {
					t.Error("missing discovery cap")
				}
				return &http.Response{StatusCode: tt.status, Header: http.Header{"Location": []string{"http://elsewhere.invalid/"}}, Body: io.NopCloser(strings.NewReader(tt.body)), Request: r}, nil
			})}
			got, err := discoverModels(context.Background(), cfg)
			if tt.want == nil {
				if err != nil || got.ID != "claude-opus-5" {
					t.Fatalf("%+v %v", got, err)
				}
			} else if !errors.Is(err, tt.want) {
				t.Fatalf("got %v want %v", err, tt.want)
			}
			if err != nil && strings.Contains(err.Error(), "SECRET") {
				t.Fatal("secret leak")
			}
			if calls != 1 {
				t.Fatalf("requests=%d", calls)
			}
		})
	}
}
func TestDiscoverCancellationAndSanitization(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := discoverModels(ctx, Config{BaseURL: "http://localhost", Transport: discoveryTransport(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, errors.New("SECRET") })})
	if !errors.Is(err, context.Canceled) || !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("%v", err)
	}
	_, err = discoverModels(context.Background(), Config{BaseURL: "http://localhost", Transport: discoveryTransport(func(r *http.Request) (*http.Response, error) { return nil, errors.New("SECRET") })})
	if !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("%v", err)
	}
}
func TestDiscoverParserLimits(t *testing.T) {
	for _, entry := range []string{`{"id":"` + strings.Repeat("a", 257) + `","owned_by":"anthropic"}`, `{"id":"x","owned_by":"` + strings.Repeat("a", 65) + `"}`, `{"id":"é","owned_by":"anthropic"}`, `{"id":7,"owned_by":"anthropic"}`} {
		if _, err := parseModels([]byte(`{"data":[` + entry + `]}`)); !errors.Is(err, ErrMalformed) {
			t.Fatalf("accepted invalid entry: %v", err)
		}
	}
	entry := `{"id":"x","owned_by":"unknown"}`
	for _, n := range []int{4096, 4097} {
		_, err := parseModels([]byte(`{"data":[` + strings.TrimSuffix(strings.Repeat(entry+",", n), ",") + `]}`))
		if (err == nil) != (n == 4096) {
			t.Fatalf("count %d: %v", n, err)
		}
	}
}
func FuzzParseModels(f *testing.F) {
	for _, s := range []string{`{"data":[]}`, `{"data":[{"id":"claude-opus-5","owned_by":"anthropic"}]}`, `null`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		models, err := parseModels(b)
		if err != nil {
			return
		}
		if len(b) > 1<<20 || len(models) > 4096 {
			t.Fatal("unbounded parser")
		}
		for _, m := range models {
			if !catalogText(m.ID, 256) || !catalogText(m.OwnedBy, 64) {
				t.Fatal("invalid metadata accepted")
			}
		}
	})
}

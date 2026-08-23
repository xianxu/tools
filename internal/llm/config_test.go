package llm

import (
	"errors"
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestResolveDefaultsToTheLocalProxy(t *testing.T) {
	c, err := Resolve(env(map[string]string{"ANTHROPIC_API_KEY": "sk-test-key-1234"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL != defaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, defaultBaseURL)
	}
	if c.Model != defaultModel {
		t.Errorf("Model = %q, want %q", c.Model, defaultModel)
	}
	if c.Effort != defaultEffort || c.MaxTokens != defaultMaxTokens {
		t.Errorf("effort/maxtokens = %q/%d", c.Effort, c.MaxTokens)
	}
	if c.StallAfter == 0 {
		t.Error("StallAfter unset — a stalled stream would hang until the total deadline")
	}
}

func TestResolvePrecedence(t *testing.T) {
	c, err := Resolve(env(map[string]string{
		"ANTHROPIC_API_KEY":   "sk-generic",
		"DEFINE_LLM_API_KEY":  "sk-specific",
		"DEFINE_LLM_BASE_URL": "http://elsewhere:9000",
		"DEFINE_LLM_MODEL":    "claude-sonnet-5",
		"DEFINE_LLM_EFFORT":   "max",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if c.APIKey != "sk-specific" {
		t.Errorf("APIKey = %q, want the DEFINE_-scoped one to win", c.APIKey)
	}
	if c.BaseURL != "http://elsewhere:9000" || c.Model != "claude-sonnet-5" || c.Effort != "max" {
		t.Errorf("got %+v", c)
	}
}

// No key is not an error to shout about — it is the ordinary offline state, and
// it must be recognisable with errors.Is so callers degrade uniformly.
func TestResolveWithNoKeyIsUnavailable(t *testing.T) {
	_, err := Resolve(env(nil))
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
	// A diagnostic that says "unavailable" without saying what to do is half a
	// diagnostic. It must name both env vars and where parley keeps the key.
	for _, want := range []string{"DEFINE_LLM_API_KEY", "ANTHROPIC_API_KEY", "parley"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %q: %v", want, err)
		}
	}
}

func TestRedactNeverLeaksTheKey(t *testing.T) {
	const key = "sk-ant-api03-SUPERSECRETVALUE"
	got := Redact(key)
	if strings.Contains(got, "SUPERSECRET") {
		t.Fatalf("Redact leaked the key: %q", got)
	}
	if got == "" {
		t.Fatal("Redact returned nothing; an empty diagnostic is useless")
	}
}

func TestRedactShortAndEmpty(t *testing.T) {
	if Redact("") != "(unset)" {
		t.Errorf("Redact(\"\") = %q", Redact(""))
	}
	// The parley proxy's key is 4 characters. A "short" key is the REAL case
	// here, not an edge case, so it must not be echoed whole.
	if strings.Contains(Redact("kknd"), "kknd") {
		t.Error("a short key must not be echoed whole")
	}
}

// Resolve must not consult the real environment. If it did, this test would
// depend on the developer's shell.
func TestResolveIsPureOverTheLookup(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-from-the-real-environment")
	_, err := Resolve(env(nil))
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Resolve read the process environment instead of the lookup: %v", err)
	}
}

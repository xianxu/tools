package llm

import (
	"errors"
	"strings"
	"testing"
)

type answer struct {
	Fits   bool   `json:"fits"`
	Reason string `json:"reason"`
}

func TestDecodeAcceptsWhatModelsActuallySend(t *testing.T) {
	cases := []struct{ name, body string }{
		{"bare object", `{"fits":true,"reason":"near-synonym"}`},
		{"fenced", "```json\n{\"fits\":true,\"reason\":\"near-synonym\"}\n```"},
		{"fenced, no language tag", "```\n{\"fits\":true,\"reason\":\"x\"}\n```"},
		{"leading and trailing whitespace", "  \n {\"fits\":true,\"reason\":\"x\"}  \n "},
		{"unknown extra field", `{"fits":true,"reason":"x","confidence":0.9}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := decode[answer](c.body)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !got.Fits || got.Reason == "" {
				t.Errorf("got %+v, want a fully populated value", got)
			}
		})
	}
}

// The invariant: zero value AND an ErrMalformed, never a partial value with a
// nil error, never a panic.
func TestDecodeRejectsWithoutPartialResults(t *testing.T) {
	cases := []struct{ name, body string }{
		{"empty", ""},
		{"whitespace only", "   \n  "},
		{"prose", "I think obsequious fits here."},
		{"trailing prose after the object", `{"fits":true,"reason":"x"} — hope that helps!`},
		{"two objects", `{"fits":true,"reason":"x"}{"fits":false,"reason":"y"}`},
		{"truncated object", `{"fits":true,"reason":"x`},
		{"array where an object was expected", `[{"fits":true,"reason":"x"}]`},
		// C1: encoding/json zero-fills a missing field, so these decoded to a
		// half-populated value with a NIL error — #12 could not tell {} from a
		// genuine "no", and would drop a distractor rather than skip a question.
		{"missing a required field", `{"fits":true}`},
		{"empty object", `{}`},
		{"json null", `null`},
		{"fence with prose inside", "```json\nSure! {\"fits\":true}\n```"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := decode[answer](c.body)
			if err == nil {
				t.Fatalf("accepted %q as %+v", c.body, got)
			}
			if !errors.Is(err, ErrMalformed) {
				t.Errorf("err = %v, want ErrMalformed", err)
			}
			if got != (answer{}) {
				t.Errorf("returned a partial value %+v alongside an error", got)
			}
		})
	}
}

// A malformed-response error that does not show the response is undiagnosable.
func TestDecodeErrorQuotesTheBody(t *testing.T) {
	_, err := decode[answer]("I think obsequious fits here.")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "obsequious") {
		t.Errorf("error does not quote the body: %v", err)
	}
}

// …and one that quotes all of a runaway response is unreadable. Multibyte, so a
// byte-based excerpt would cut mid-rune.
func TestDecodeErrorBoundsTheExcerpt(t *testing.T) {
	body := strings.Repeat("señor ", 500)
	_, err := decode[answer](body)
	if err == nil {
		t.Fatal("expected an error")
	}
	if len(err.Error()) > 1000 {
		t.Errorf("error is %d bytes; the excerpt is unbounded", len(err.Error()))
	}
	if strings.ContainsRune(err.Error(), '�') {
		t.Error("excerpt cut a multibyte character")
	}
}

// The fuzz corpus is where enumerated cases belong — a prose list rots because
// nobody adds the eighth entry. The property is the invariant above.
func FuzzDecode(f *testing.F) {
	for _, seed := range []string{
		`{"fits":true,"reason":"x"}`,
		"```json\n{\"fits\":true}\n```",
		`{"fits":true,"reason":"x"} trailing`,
		`[{"fits":true}]`,
		`{"fits":true,"reason":"x`,
		`{"fits":true}{"fits":false}`,
		"", "   ", "not json at all", "{}", `{"reason":"señor — dash"}`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body string) {
		got, err := decode[answer](body)
		if err == nil {
			// The success half of the invariant, which this used to skip — and
			// that omission is why C1 survived: every required field must be
			// populated, or decode has returned a partial value with a nil error.
			if got.Reason == "" {
				t.Fatalf("decode(%q) succeeded with an empty required field: %+v", body, got)
			}
			return
		}
		if !errors.Is(err, ErrMalformed) {
			t.Fatalf("error is not ErrMalformed: %v", err)
		}
		if got != (answer{}) {
			t.Fatalf("partial value %+v returned with an error", got)
		}
	})
}

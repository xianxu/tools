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

type innerResult struct {
	Score  int    `json:"score"`
	Detail string `json:"detail"`
}

type outerResult struct {
	Fits  bool        `json:"fits"`
	Inner innerResult `json:"inner"`
}

// BR-51: the required check walked only the top level, so the C1 bug survived one
// level down. #10's authoring result — an item with its distractors — is the
// first consumer likely to be nested, so this is not a hypothetical depth.
func TestDecodeRequiresNestedFieldsToo(t *testing.T) {
	for _, c := range []struct{ name, body string }{
		{"nested object empty", `{"fits":true,"inner":{}}`},
		{"nested object partial", `{"fits":true,"inner":{"score":3}}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := decode[outerResult](c.body)
			if err == nil {
				t.Fatalf("accepted %q as %+v", c.body, got)
			}
			if got != (outerResult{}) {
				t.Errorf("returned a partial value %+v with an error", got)
			}
			// The error must name WHERE, or a nested miss is undiagnosable.
			if !strings.Contains(err.Error(), "inner.") {
				t.Errorf("error does not name the nested path: %v", err)
			}
		})
	}

	full := `{"fits":true,"inner":{"score":3,"detail":"x"}}`
	got, err := decode[outerResult](full)
	if err != nil {
		t.Fatalf("rejected a complete nested payload: %v", err)
	}
	if got.Inner.Detail != "x" {
		t.Errorf("got %+v", got)
	}
}

type arrInner struct {
	Score  int    `json:"score"`
	Detail string `json:"detail"`
}

type arrOuter struct {
	Fits bool       `json:"fits"`
	List []arrInner `json:"list"`
}

type nullableOuter struct {
	Fits  bool     `json:"fits"`
	Inner arrInner `json:"inner"`
}

// The invariant quantifies over the whole schema TREE, so the check must cover
// every shape a schema can nest — not just the example a finding used. Written
// against nested objects alone, it left objects inside ARRAYS unchecked, and
// #10's authoring result is exactly a list of objects.
func TestDecodeRequiresFieldsAtEveryShape(t *testing.T) {
	t.Run("objects inside arrays", func(t *testing.T) {
		for _, body := range []string{
			`{"fits":true,"list":[{}]}`,
			`{"fits":true,"list":[{"score":1}]}`,
			`{"fits":true,"list":[{"score":1,"detail":"ok"},{"score":2}]}`,
		} {
			got, err := decode[arrOuter](body)
			if err == nil {
				t.Errorf("accepted %s as %+v", body, got)
				continue
			}
			if got.Fits || got.List != nil {
				t.Errorf("returned a partial value %+v with an error", got)
			}
			// The path names the position, or a bad item in a long list is
			// undiagnosable.
			if !strings.Contains(err.Error(), "list[") {
				t.Errorf("error does not name the array position: %v", err)
			}
		}
	})

	t.Run("a complete list is accepted", func(t *testing.T) {
		got, err := decode[arrOuter](`{"fits":true,"list":[{"score":1,"detail":"a"},{"score":2,"detail":"b"}]}`)
		if err != nil {
			t.Fatalf("rejected a complete payload: %v", err)
		}
		if len(got.List) != 2 || got.List[1].Detail != "b" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("an explicit null for a required object", func(t *testing.T) {
		// Present but carrying nothing: without the nil check this zero-fills
		// silently, which is the original Critical wearing a different hat.
		got, err := decode[nullableOuter](`{"fits":true,"inner":null}`)
		if err == nil {
			t.Errorf("accepted an explicit null as %+v", got)
		}
	})
}

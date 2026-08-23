package llm

import (
	"encoding/json"
	"errors"
	"reflect"
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
// nobody adds the eighth entry. The property is the invariant decode advertises.
//
// It ranges over SEVERAL result shapes, not one. The first version asserted
// `got.Reason != ""` — a named field of a single fixture type — so it exercised
// exactly the shape that already had cases, and every other vehicle (nested
// object, array item, map value) had to be found by a reviewer instead.
func FuzzDecode(f *testing.F) {
	for _, seed := range []string{
		`{"fits":true,"reason":"x"}`,
		"```json\n{\"fits\":true}\n```",
		`{"fits":true,"reason":"x"} trailing`,
		`[{"fits":true}]`,
		`{"fits":true,"reason":"x`,
		`{"fits":true}{"fits":false}`,
		`{"fits":true,"inner":{}}`,
		`{"fits":true,"list":[{}]}`,
		`{"fits":true,"by":{"a":{}}}`,
		`{"fits":true,"inner":null}`,
		"", "   ", "not json at all", "{}", "null", `{"reason":"señor — dash"}`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body string) {
		// One arm per nesting keyword, so a hole in the walk is reachable from
		// the fuzzer rather than only from a hand-written case.
		checkInvariant(t, body, func(s string) (any, error) { return decode[answer](s) })
		checkInvariant(t, body, func(s string) (any, error) { return decode[kwNested](s) })
		checkInvariant(t, body, func(s string) (any, error) { return decode[kwSlice](s) })
		checkInvariant(t, body, func(s string) (any, error) { return decode[kwMap](s) })
	})
}

// checkInvariant asserts decode's advertised contract without naming any field:
// on success every schema-required path must have been PRESENT in the input, and
// on failure the error is ErrMalformed and the value is zero.
//
// The presence check is written independently of missingRequired — it is an
// oracle, so sharing that traversal would make it agree with the code by
// construction rather than by correctness.
func checkInvariant(t *testing.T, body string, dec func(string) (any, error)) {
	t.Helper()
	got, err := dec(body)
	if err != nil {
		if !errors.Is(err, ErrMalformed) && !errors.Is(err, ErrTruncated) {
			t.Fatalf("error is neither ErrMalformed nor ErrTruncated: %v", err)
		}
		if !reflect.ValueOf(got).IsZero() {
			t.Fatalf("partial value %+v returned with an error", got)
		}
		return
	}
	// Success: every field the decoded value carries must have come from the
	// input, so re-marshalling and comparing key sets catches a zero-filled
	// required field that decode should have rejected.
	out, mErr := json.Marshal(got)
	if mErr != nil {
		return
	}
	var fromInput, fromResult any
	if json.Unmarshal([]byte(strings.TrimSpace(stripFence(body))), &fromInput) != nil {
		return
	}
	if json.Unmarshal(out, &fromResult) != nil {
		return
	}
	if missing := keysAbsentFromInput(fromResult, fromInput, ""); len(missing) > 0 {
		t.Fatalf("decode(%q) succeeded but %v were zero-filled rather than supplied", body, missing)
	}
}

// keysAbsentFromInput reports keys present in the decoded result but absent from
// the input — i.e. fields encoding/json zero-filled. Deliberately a separate,
// simpler traversal from missingRequired: an oracle that shares its subject's
// logic cannot disagree with it.
func keysAbsentFromInput(result, input any, path string) []string {
	rm, ok := result.(map[string]any)
	if !ok {
		return nil
	}
	im, ok := input.(map[string]any)
	if !ok {
		return nil
	}
	var missing []string
	for k, rv := range rm {
		iv, found := im[k]
		if !found {
			missing = append(missing, join(path, k))
			continue
		}
		missing = append(missing, keysAbsentFromInput(rv, iv, join(path, k))...)
	}
	return missing
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

type kwItem struct {
	Score  int    `json:"score"`
	Detail string `json:"detail"`
}

type kwStruct struct {
	Fits bool   `json:"fits"`
	Note string `json:"note"`
}
type kwNested struct {
	Fits  bool   `json:"fits"`
	Inner kwItem `json:"inner"`
}
type kwSlice struct {
	Fits bool     `json:"fits"`
	List []kwItem `json:"list"`
}
type kwMap struct {
	Fits bool              `json:"fits"`
	By   map[string]kwItem `json:"by"`
}
type kwPtr struct {
	Fits  bool    `json:"fits"`
	Inner *kwItem `json:"inner"`
}

// One case per subschema-bearing keyword the generator can emit, because four
// instance-fixes in a row each covered the example a finding used and missed the
// next. The traversal is driven by that vocabulary, so the tests are too.
func TestRequiredWalkCoversEveryNestingKeyword(t *testing.T) {
	cases := []struct {
		keyword string
		bad     string
		good    string
		decode  func(string) (any, error)
	}{
		{"properties (struct)", `{"fits":true}`, `{"fits":true,"note":"x"}`,
			func(s string) (any, error) { return decode[kwStruct](s) }},
		{"properties (nested struct)", `{"fits":true,"inner":{}}`, `{"fits":true,"inner":{"score":1,"detail":"d"}}`,
			func(s string) (any, error) { return decode[kwNested](s) }},
		{"items (slice of struct)", `{"fits":true,"list":[{}]}`, `{"fits":true,"list":[{"score":1,"detail":"d"}]}`,
			func(s string) (any, error) { return decode[kwSlice](s) }},
		{"additionalProperties (map of struct)", `{"fits":true,"by":{"a":{}}}`, `{"fits":true,"by":{"a":{"score":1,"detail":"d"}}}`,
			func(s string) (any, error) { return decode[kwMap](s) }},
		{"properties (pointer to struct)", `{"fits":true,"inner":{"score":1}}`, `{"fits":true,"inner":{"score":1,"detail":"d"}}`,
			func(s string) (any, error) { return decode[kwPtr](s) }},
	}
	for _, c := range cases {
		t.Run(c.keyword, func(t *testing.T) {
			if got, err := c.decode(c.bad); err == nil {
				t.Errorf("accepted an incomplete payload %s as %+v", c.bad, got)
			}
			if _, err := c.decode(c.good); err != nil {
				t.Errorf("rejected a complete payload %s: %v", c.good, err)
			}
		})
	}
}

// The latent half of the enumeration: if reflectSchema's configuration ever
// changes, a schema could arrive carrying a keyword missingRequired does not
// traverse, and a required field beneath it would be silently unchecked. This
// fails when that day comes, rather than letting a consumer find it.
func TestSchemaKeywordsAreCovered(t *testing.T) {
	// Keyword -> whether missingRequired traverses it. A keyword absent from this
	// map is one nobody has considered, which is the interesting case.
	traversed := map[string]bool{
		"properties": true, "items": true, "additionalProperties": true,
		// Structural; carry no subschema that needs traversing.
		"type": true, "required": true, "description": true, "title": true,
		"format": true, "enum": true, "default": true,
		// Would need traversal if this configuration ever emitted them.
		"patternProperties": false, "$ref": false, "$defs": false,
		"oneOf": false, "anyOf": false, "allOf": false,
	}

	var walkSchema func(map[string]any, string)
	walkSchema = func(s map[string]any, path string) {
		for k, v := range s {
			covered, listed := traversed[k]
			if !listed {
				t.Errorf("%s: schema keyword %q is not in the enumeration missingRequired is written against", path, k)
				continue
			}
			if !covered {
				t.Errorf("%s: keyword %q is emitted but NOT traversed; a required field beneath it is unchecked", path, k)
				continue
			}
			// Each traversed keyword has its own child shape, and conflating them
			// is what made the first version read property NAMES as keywords.
			switch k {
			case "properties", "patternProperties", "$defs":
				named, _ := v.(map[string]any)
				for name, sub := range named {
					if child, ok := sub.(map[string]any); ok {
						walkSchema(child, path+"."+k+"."+name)
					}
				}
			case "items", "additionalProperties":
				if child, ok := v.(map[string]any); ok {
					walkSchema(child, path+"."+k)
				}
			}
		}
	}

	for name, get := range map[string]func() (map[string]any, error){
		"struct": SchemaFor[kwStruct], "nested": SchemaFor[kwNested],
		"slice": SchemaFor[kwSlice], "map": SchemaFor[kwMap], "pointer": SchemaFor[kwPtr],
	} {
		s, err := get()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		walkSchema(s, name)
	}
}

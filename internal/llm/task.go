package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Task is one prompt with one typed answer.
//
// A package-level Run rather than a method on Client, because Go forbids type
// parameters on methods — and that is the better shape here anyway: the
// transport interface stays two methods wide and un-generic, so the wire fake
// and the live client implement the same tiny surface, while the per-task unit
// worth testing lives with the consumer that owns the prompt.
type Task[T any] struct {
	// Name is stable and identifies the task everywhere it is observed: the
	// golden file, the cassette key, the usage label, the error message.
	Name   string
	System string
	Prompt string
	// Zero means "use the Config default".
	Model     string
	Effort    string
	MaxTokens int64
}

// Run renders a Task to a Request, calls the model, and decodes the answer into T.
func Run[T any](ctx context.Context, c Client, t Task[T]) (T, error) {
	var zero T
	schema, err := SchemaFor[T]()
	if err != nil {
		return zero, fmt.Errorf("%s: %w", t.Name, err)
	}
	resp, err := c.Complete(ctx, Request{
		Task: t.Name, System: t.System, Prompt: t.Prompt,
		Model: t.Model, Effort: t.Effort, MaxTokens: t.MaxTokens,
		Schema: schema,
	})
	// The stop reason is checked FIRST, by Complete, and its error is returned
	// before anything is decoded. Not an ordering nicety: a truncated structured
	// answer commonly parses, so a decode-first implementation returns success on
	// garbage and no consumer downstream can tell.
	if err != nil {
		return zero, fmt.Errorf("%s: %w", t.Name, err)
	}
	out, err := decode[T](resp.Text)
	if err != nil {
		return zero, fmt.Errorf("%s: %w", t.Name, err)
	}
	return out, nil
}

// decode parses a model's answer into T.
//
// The strategy, stated once: strip one optional markdown fence, decode exactly
// one JSON value, require the whole payload consumed to EOF, require every field
// the schema marks REQUIRED to be present, and allow unknown fields. Unknown
// fields are allowed deliberately — a provider adding one must not break us —
// while a missing required field is not, because a half-populated result is worse
// than no result.
//
// The required check is not decoration. encoding/json silently zero-fills a
// missing field, so `{}` decoded into a veto verdict yields Fits:false — which
// #12 cannot distinguish from a real "no", and would drop a distractor rather
// than skip a question. The schema already declares which fields are required;
// this derives the check from that single source rather than restating it.
//
// The invariant, asserted by the fuzz target on BOTH branches: decode returns
// either a fully populated T and nil, or the zero T and an error matching
// ErrMalformed. It never panics, and it never returns a partially populated value
// with a nil error.
func decode[T any](raw string) (T, error) {
	var zero T
	body := strings.TrimSpace(stripFence(raw))
	if body == "" {
		return zero, fmt.Errorf("%w: empty response", ErrMalformed)
	}
	if err := requireSchemaFields[T](body); err != nil {
		return zero, err
	}

	var out T
	dec := json.NewDecoder(strings.NewReader(body))
	if err := dec.Decode(&out); err != nil {
		return zero, fmt.Errorf("%w: %w (body: %s)", ErrMalformed, err, excerpt(body))
	}
	// The whole payload must be consumed. Testing for "another token" is not
	// enough: dec.Token() returns an ERROR for trailing prose, so `err == nil`
	// caught a second object and let `{...} — hope that helps!` through with a
	// populated value. Clean EOF is the only acceptable outcome — anything else,
	// token or parse error, means the model kept talking and what we decoded may
	// be a fragment of a larger intent.
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return zero, fmt.Errorf("%w: trailing content after the JSON value (body: %s)",
			ErrMalformed, excerpt(body))
	}
	return out, nil
}

// requireSchemaFields rejects a payload missing any field the schema marks
// required, deriving the set from SchemaFor[T] rather than restating it.
//
// Only object payloads are checked: a T that is not a struct has no required
// set, and a non-object body is rejected by the decode below anyway.
func requireSchemaFields[T any](body string) error {
	var present map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &present); err != nil {
		return nil // not an object; decode reports the real problem
	}
	schema, err := SchemaFor[T]()
	if err != nil {
		return nil // a type we cannot reflect declares nothing to require
	}
	required, _ := schema["required"].([]any)
	var missing []string
	for _, k := range required {
		name, ok := k.(string)
		if !ok {
			continue
		}
		if _, found := present[name]; !found {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: missing required field(s) %s (body: %s)",
			ErrMalformed, strings.Join(missing, ", "), excerpt(body))
	}
	return nil
}

// stripFence removes one ```json … ``` wrapper. Models add them despite a schema,
// and a fence is framing rather than content.
func stripFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[i+1:] // drop the ``` line, language tag and all
	}
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// excerpt bounds what a malformed-response error quotes. An error that does not
// show the body is undiagnosable; one that shows all of it is unreadable.
func excerpt(s string) string {
	const max = 200
	r := []rune(s)
	if len(r) <= max {
		return strconv.Quote(s)
	}
	return strconv.Quote(string(r[:max])) + "…"
}

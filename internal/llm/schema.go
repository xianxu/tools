package llm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/invopop/jsonschema"
)

// SchemaFor derives a JSON Schema from the caller's result type.
//
// Reflected rather than hand-written beside each result struct, because two
// sources of truth for one shape drift the first time a field is added
// (ARCH-DRY). The generated schema is still a visible artifact: llmtest.Golden
// snapshots it, so a struct field added without thought shows up in a diff —
// which is the only property the hand-written version actually had.
//
// Memoised per type: reflection is not free and a task runs the same type on
// every call.
func SchemaFor[T any]() (map[string]any, error) {
	var zero T
	t := reflect.TypeOf(&zero).Elem()
	if cached, ok := schemaCache.Load(t); ok {
		c := cached.(cachedSchema)
		return c.schema, c.err
	}
	s, err := reflectSchema(zero)
	schemaCache.Store(t, cachedSchema{s, err})
	return s, err
}

type cachedSchema struct {
	schema map[string]any
	err    error
}

var schemaCache sync.Map

func reflectSchema(v any) (map[string]any, error) {
	r := &jsonschema.Reflector{
		// Inline everything. A schema full of $ref/$defs is harder for a model to
		// follow and harder for a human to read in a golden file, and the result
		// types here are small by construction.
		DoNotReference: true,
		ExpandedStruct: true,
		// Every field required unless it says otherwise. A model given optional
		// fields omits them, and a half-populated result is worse than no result —
		// the same reason decode rejects a missing required field.
		RequiredFromJSONSchemaTags: false,
	}
	raw, err := json.Marshal(r.Reflect(v))
	if err != nil {
		return nil, fmt.Errorf("%w: reflecting schema: %w", ErrRequest, err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%w: reflecting schema: %w", ErrRequest, err)
	}
	// The provider rejects a schema carrying JSON Schema metadata it does not
	// use, and $schema/$id say nothing about the shape.
	delete(out, "$schema")
	delete(out, "$id")
	// additionalProperties:false is what makes the constraint meaningful — without
	// it a model may return the required fields plus anything else, which is not
	// the shape the caller declared.
	out["additionalProperties"] = false
	return out, nil
}

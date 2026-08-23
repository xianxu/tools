package llm

import (
	"encoding/json"
	"testing"
)

type vetoVerdict struct {
	Fits   bool   `json:"fits"`
	Reason string `json:"reason"`
}

func TestSchemaForDerivesTheShape(t *testing.T) {
	s, err := SchemaFor[vetoVerdict]()
	if err != nil {
		t.Fatal(err)
	}
	if s["type"] != "object" {
		t.Errorf("type = %v", s["type"])
	}
	props, ok := s["properties"].(map[string]any)
	if !ok {
		t.Fatalf("no properties: %+v", s)
	}
	for _, want := range []string{"fits", "reason"} {
		if _, ok := props[want]; !ok {
			t.Errorf("property %q missing — the json tag is the field name the model must return", want)
		}
	}
	if s["additionalProperties"] != false {
		t.Error("additionalProperties must be false, or the declared shape is not a constraint")
	}
	// $schema/$id describe the schema, not the answer, and the provider rejects
	// metadata it does not use.
	for _, k := range []string{"$schema", "$id"} {
		if _, present := s[k]; present {
			t.Errorf("%s should be stripped", k)
		}
	}
}

// Adding a field to the result type must change the schema — that is the DRY
// property the reflection exists for, and the reason a golden snapshot is worth
// keeping.
func TestSchemaTracksTheStruct(t *testing.T) {
	type v1 struct {
		Fits bool `json:"fits"`
	}
	type v2 struct {
		Fits   bool   `json:"fits"`
		Reason string `json:"reason"`
	}
	a, _ := SchemaFor[v1]()
	b, _ := SchemaFor[v2]()
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) == string(jb) {
		t.Error("adding a field did not change the schema; the derivation is not tracking the type")
	}
}

// Memoised per type, and the cache must not confuse two types.
func TestSchemaCacheIsPerType(t *testing.T) {
	type alpha struct {
		A string `json:"a"`
	}
	type beta struct {
		B string `json:"b"`
	}
	sa, _ := SchemaFor[alpha]()
	sb, _ := SchemaFor[beta]()
	pa := sa["properties"].(map[string]any)
	pb := sb["properties"].(map[string]any)
	if _, ok := pa["a"]; !ok {
		t.Error("alpha lost its field")
	}
	if _, ok := pb["b"]; !ok {
		t.Error("beta got alpha's schema from the cache")
	}
	// Second call comes from the cache and must be identical.
	again, _ := SchemaFor[alpha]()
	if again["properties"].(map[string]any)["a"] == nil {
		t.Error("cached schema differs from the first derivation")
	}
}

// The memoised schema is handed out by value, not by reference.
//
// Returning the cached map itself means one consumer writing to it permanently
// changes what every later Run[T] sends on the wire and what requireSchemaFields
// reads — a process-wide poisoning that only shows up in the SECOND caller, in a
// package five issues will consume.
func TestSchemaIsolationAcrossCallers(t *testing.T) {
	type isolated struct {
		A string `json:"a"`
	}
	first, err := SchemaFor[isolated]()
	if err != nil {
		t.Fatal(err)
	}
	first["description"] = "poisoned"
	delete(first, "required")
	props, _ := first["properties"].(map[string]any)
	props["injected"] = map[string]any{"type": "string"}

	second, err := SchemaFor[isolated]()
	if err != nil {
		t.Fatal(err)
	}
	if _, poisoned := second["description"]; poisoned {
		t.Error("a top-level write by one caller reached the next")
	}
	if _, ok := second["required"]; !ok {
		t.Error("a delete by one caller reached the next")
	}
	if p, _ := second["properties"].(map[string]any); p != nil {
		if _, injected := p["injected"]; injected {
			t.Error("a NESTED write by one caller reached the next — the clone is shallow")
		}
	}
}

package llm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// renderRequest is THE canonical text form of a Request: model, effort, system,
// prompt and schema, in a fixed order with fixed separators.
//
// Exactly one renderer, consumed by two things that must never disagree —
// llmtest.Golden prints it for humans to diff, and llmtest.Cassette hashes it to
// key a recording. Rendered separately, a prompt edit could move the golden while
// the cassette kept matching, so the change would be visible in one artifact and
// invisible in the other (ARCH-DRY).
//
// Deliberately excludes MaxTokens: it does not change what was asked, only how
// much room the answer had, and including it would invalidate every cassette the
// moment a default moved.
func renderRequest(r Request) string {
	var b strings.Builder
	fmt.Fprintf(&b, "task:   %s\n", r.Task)
	fmt.Fprintf(&b, "model:  %s\n", r.Model)
	fmt.Fprintf(&b, "effort: %s\n", r.Effort)
	b.WriteString("--- system ---\n")
	b.WriteString(r.System)
	b.WriteString("\n--- prompt ---\n")
	b.WriteString(r.Prompt)
	b.WriteString("\n--- schema ---\n")
	b.WriteString(renderSchema(r.Schema))
	b.WriteString("\n")
	return b.String()
}

// renderSchema serialises a schema deterministically.
//
// encoding/json sorts map keys at every level, which is the property the whole
// hash rests on — a Go map has no iteration order, so an unsorted render would
// give a different hash run to run and every cassette would miss at random.
// Asserted in render_test.go rather than trusted.
func renderSchema(s map[string]any) string {
	if s == nil {
		return "(none)"
	}
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		// A schema that will not marshal cannot be sent either; render the error
		// so the golden shows it rather than silently rendering "(none)".
		return fmt.Sprintf("(unmarshalable schema: %v)", err)
	}
	return string(out)
}

// RequestHash keys a cassette. Short by design — it appears in filenames and in
// the failure message when a recording is missing.
func RequestHash(r Request) string {
	sum := sha256.Sum256([]byte(renderRequest(r)))
	return hex.EncodeToString(sum[:])[:12]
}

// RenderRequest exposes the canonical form to llmtest, which must not render its
// own. Not part of the Client contract — a consumer never needs it.
func RenderRequest(r Request) string { return renderRequest(r) }

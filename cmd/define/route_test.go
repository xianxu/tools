package main

import (
	"bytes"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// TestConsoleDecisionTable is the issue's decision table, asserted through the
// REAL route: parseREPLLine, then the dictionary, then readsAsQuestion.
//
// Driving the two halves separately proves each is correct and leaves the table
// — which is the thing the spec promises — unasserted. This is the test that
// fails if either half stops agreeing with the other.
func TestConsoleDecisionTable(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
		want string // "command" | "lookup" | "question" | "not-found"
	}{
		{"a slash command is untouched", "/history 7", "command"},
		{"a one-word headword", "sycophantic", "lookup"},
		{"a two-word headword", "hot dog", "lookup"},
		{"a question NOAD cannot answer", "what's the difference to obsequious?", "question"},
		{"a typo stays a miss", "sycophanti", "not-found"},
		{"the ? hatch skips the dictionary", "?hot dog", "question"},
		{`the \ hatch suppresses the fallback`, `\how so`, "not-found"},
		{"a follow-up request", "use it in a sentence", "question"},
		{"a long phrase with no interrogative reading", "difference between sycophantic and obsequious", "not-found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := routeFor(t, tc.line); got != tc.want {
				t.Errorf("%q routed to %s, want %s", tc.line, got, tc.want)
			}
		})
	}
}

// routeFor runs one line through the production route and names where it landed.
// A harness, deliberately not a second router: everything it decides, it decides
// by asking parseREPLLine and lookupAndRender.
func routeFor(t *testing.T, line string) string {
	t.Helper()
	dict, err := loadFakeDictionary("testdata/entries", store.DefaultLang)
	if err != nil {
		t.Fatalf("fake dictionary: %v", err)
	}
	d := deps{dict: dict, capture: noopCapturer{}}
	cmd := parseREPLLine(line, false)
	switch cmd.kind {
	case cmdCommand:
		return "command"
	case cmdAsk:
		return "question"
	case cmdDefine:
		var out, errb bytes.Buffer
		switch got := lookupAndRender(d, options{}, cmd, &out, &errb); {
		case got.ask != "":
			return "question"
		case got.code == 0:
			return "lookup"
		default:
			return "not-found"
		}
	}
	return "nothing"
}

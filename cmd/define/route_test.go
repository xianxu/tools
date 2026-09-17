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
		{"a long phrase with no interrogative reading", "difference between sycophantic and obsequious", "question"},
		{"reported statement", "so lickspittle is similar to sycophantic", "question"},
		{"four words", "same meaning then actually", "question"},
		{"three words", "same meaning then", "not-found"},
		{`literal long sentence`, `\so lickspittle is similar to sycophantic`, "not-found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := routeFor(t, tc.line); got != tc.want {
				t.Errorf("%q routed to %s, want %s", tc.line, got, tc.want)
			}
		})
	}
}

func TestSentenceFallbackRespectsDictionaryAndRawMode(t *testing.T) {
	const phrase = "a phrase with four words"
	for _, hit := range []bool{false, true} {
		for _, raw := range []bool{false, true} {
			dict, err := loadFakeDictionary("testdata/entries", store.DefaultLang)
			if err != nil {
				t.Fatal(err)
			}
			if hit {
				// Alias a real entry: this test concerns lookup precedence, not
				// inventing dictionary content or a second rendering fixture.
				dict.entries[phrase] = dict.entries["sycophantic"]
			}
			d := deps{dict: dict, langDeps: langDeps{capture: noopCapturer{}}}
			var out, errOut bytes.Buffer
			got := lookupAndRender(d, options{raw: raw}, parseREPLLine(phrase, lineState{}), &out, &errOut)
			wantAsk := !hit && !raw
			if (got.ask != "") != wantAsk || (hit && (got.code != 0 || out.Len() == 0)) || (!hit && raw && got.code != 1) {
				t.Fatalf("hit=%v raw=%v: outcome=%+v output=%q error=%q", hit, raw, got, out.String(), errOut.String())
			}
		}
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
	d := deps{dict: dict, langDeps: langDeps{capture: noopCapturer{}}}
	cmd := parseREPLLine(line, lineState{})
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

// EVERY repl kind is decided for BOTH loops, derived from numReplKinds — and the
// declaration is CHECKED against the loops, not merely present.
//
// The first version asserted only that a row existed, so inverting every row left
// the suite green: a payload nothing reads is a comment with a type. #67 added
// cmdAskPassage and the piped loop had no case for it; a map nobody consults
// would not have caught the next one either.
func TestEveryReplKindIsDecidedForBothLoops(t *testing.T) {
	for kind := replKind(0); kind < numReplKinds; kind++ {
		row, declared := replKindHandling[kind]
		if !declared {
			t.Errorf("replKind %d has no row in replKindHandling — a kind nobody decided about "+
				"is how the piped loop came to silently ignore cmdAskPassage", kind)
			continue
		}
		// The claim each row makes is that the loop can REACH this kind, which is
		// checkable: parseREPLLine is the only producer, and lineState is the only
		// thing that varies between the two loops.
		if got := kindReachable(kind, editorStates()); got != row.editor {
			t.Errorf("replKind %d: reachable in the editor = %v, declared %v", kind, got, row.editor)
		}
		if got := kindReachable(kind, pipedStates()); got != row.piped {
			t.Errorf("replKind %d: reachable in the piped loop = %v, declared %v", kind, got, row.piped)
		}
	}
}

// The lines each loop can present, paired with the states it can be in. The
// piped loop has no screen, so it passes hasPassage/hasMarks false — that is the
// whole difference, and it is what makes cmdAskPassage unreachable there.
func replLines2() []string {
	return []string{"", "sycophantic", "?what is this", `\word`, "/lang es", "  "}
}

func editorStates() []lineState {
	return []lineState{{}, {hasCurrent: true}, {hasPassage: true}, {hasPassage: true, hasMarks: true},
		{hasCurrent: true, hasPassage: true, hasMarks: true}}
}

func pipedStates() []lineState { return []lineState{{}, {hasCurrent: true}} }

func kindReachable(kind replKind, states []lineState) bool {
	for _, st := range states {
		for _, line := range replLines2() {
			if parseREPLLine(line, st).kind == kind {
				return true
			}
		}
	}
	return false
}

// And the claim that the piped loop cannot see cmdAskPassage is CHECKED, not
// asserted: it passes hasMarks=false, which is the only way to reach that kind.
func TestThePipedLoopCannotProduceAPassageAsk(t *testing.T) {
	for _, line := range []string{"", "sycophantic", "?what is this", "/lang es", `\word`} {
		if got := parseREPLLine(line, lineState{hasCurrent: true}); got.kind == cmdAskPassage {
			t.Errorf("parseREPLLine(%q) produced cmdAskPassage without marks", line)
		}
	}
	// With marks it DOES, which is what makes the row above a real distinction.
	if got := parseREPLLine("", lineState{hasPassage: true, hasMarks: true}); got.kind != cmdAskPassage {
		t.Errorf("with marks, bare Enter = %v, want cmdAskPassage", got.kind)
	}
}

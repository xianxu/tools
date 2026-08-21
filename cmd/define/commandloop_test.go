package main

import (
	"bytes"
	"strings"
	"testing"
)

// A dictionary that fails the test if it is consulted. Dispatch's whole job is
// to keep a command away from the dictionary, so the assertion has to be that
// the dictionary was never reached — not merely that the output looks right.
type refusingDict struct{ t *testing.T }

func (d refusingDict) Lookup(word string) (string, error) {
	d.t.Errorf("the dictionary was consulted for %q; a command must never reach it", word)
	return "", nil
}

var dispatchCmds = []command{
	{name: "help", summary: "list the commands", run: runHelp},
	{name: "history", summary: "words looked up recently",
		run: func(c commandCtx, args []string) int {
			c.stdout.Write([]byte("HISTORY RAN args=" + strings.Join(args, ",") + "\n"))
			return 0
		}},
}

func TestParseREPLLineClassifiesCommands(t *testing.T) {
	for _, tc := range []struct {
		in   string
		kind replKind
		name string
		args []string
	}{
		{"/history", cmdCommand, "history", nil},
		{"/history 7", cmdCommand, "history", []string{"7"}},
		{"/", cmdCommand, "", nil},
		{"hot dog", cmdDefine, "", nil},
		{"", cmdNothing, "", nil},
	} {
		t.Run(tc.in, func(t *testing.T) {
			got := parseREPLLine(tc.in, false)
			if got.kind != tc.kind || got.name != tc.name {
				t.Errorf("parseREPLLine(%q) = kind %v name %q, want kind %v name %q",
					tc.in, got.kind, got.name, tc.kind, tc.name)
			}
			if !sameSet(got.args, tc.args) {
				t.Errorf("args = %v, want %v", got.args, tc.args)
			}
		})
	}
}

// PQ-2: the first draft dispatched in submitLine, which ONLY the raw editor
// reaches. `echo /history | define` would have gone to the dictionary. Both
// loops get their own test for exactly that reason.
func TestLineLoopDispatchesCommands(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}
	rig.deps.stdinIsTerminal = func() bool { return false }

	var out, errb bytes.Buffer
	code := replLines(t.Context(), rig.deps, options{times: 3, locale: "us"},
		strings.NewReader("/help\n"), &out, &errb, true, false)

	if code != 0 {
		t.Errorf("exit = %d, stderr = %s", code, errb.String())
	}
	// Asserted on the command's OUTPUT, never on "/help": the raw editor echoes
	// the submitted line, so matching the name matches the echo and passes with
	// dispatch deleted (BR-3, measured). "list the commands" is a summary only
	// runHelp can emit.
	if !strings.Contains(out.String(), "list the commands") {
		t.Errorf("the piped loop did not run the command: %q", out.String())
	}
}

func TestRawEditorDispatchesCommands(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}

	var out, errb bytes.Buffer
	code := runEditor(t.Context(), scriptKeys("/help\r"), rig.deps, opt, cooked, finish, &out, &errb)

	if code != 0 {
		t.Errorf("exit = %d, stderr = %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "list the commands") {
		t.Errorf("the raw editor did not run the command: %q", out.String())
	}
}

func TestUnknownCommandSuggestsWithoutDefining(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}

	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("/histry\r"), rig.deps, opt, cooked, finish, &out, &errb)

	if !strings.Contains(errb.String(), "/histry") {
		t.Errorf("the unknown command was not named: %q", errb.String())
	}
	// Against the LIVE registry, which at M1 holds only /help — so the menu is
	// what a near-miss can offer. TestDispatchCommand covers the "did you mean"
	// path against a fixture set with more rows; this test's job is that the
	// unknown command reached dispatch at all instead of the dictionary.
	if !strings.Contains(errb.String(), "/help") {
		t.Errorf("no suggestion offered: %q", errb.String())
	}
}

func TestDispatchCommand(t *testing.T) {
	for _, tc := range []struct {
		name     string
		line     string
		wantCode int
		wantOut  string
		wantErr  string
	}{
		{"runs the command", "/history", 0, "HISTORY RAN args=", ""},
		{"passes arguments", "/history 7", 0, "HISTORY RAN args=7", ""},
		{"a bare slash lists the menu", "/", 0, "/history", ""},
		{"unknown suggests the near match", "/histry", 2, "", "did you mean /history"},
		{"nothing close lists everything", "/qqqqqq", 2, "", "/help"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			cc := commandCtx{stdout: &out, stderr: &errb}
			code := dispatchCommand(parseREPLLine(tc.line, false), dispatchCmds, cc)

			if code != tc.wantCode {
				t.Errorf("exit = %d, want %d", code, tc.wantCode)
			}
			if tc.wantOut != "" && !strings.Contains(out.String(), tc.wantOut) {
				t.Errorf("stdout = %q, want it to contain %q", out.String(), tc.wantOut)
			}
			if tc.wantErr != "" && !strings.Contains(errb.String(), tc.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", errb.String(), tc.wantErr)
			}
		})
	}
}

// BR-2: completionsFor having the right answer is not the same as the editor
// asking it. This pins the wiring — type "/his" and the grey tail must be
// offered from the COMMAND set, which history could never produce.
func TestEditorSuggestsFromCommands(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}
	// History that would suggest something else entirely if it were consulted.
	h := &memHistory{}
	h.Add("hibernate")
	rig.deps.history = h

	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("/hel\x03"), rig.deps, opt, cooked, finish, &out, &errb)

	if !strings.Contains(out.String(), greyOn+"p") {
		t.Errorf("no grey completion from the command set: %q", out.String())
	}
	if strings.Contains(out.String(), "ibernate") {
		t.Errorf("the line starts with / but history was consulted: %q", out.String())
	}
}

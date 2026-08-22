package main

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
)

// A fixture table rather than the live registry: these are pure functions over a
// command set, and coupling their tests to whatever commands happen to be
// registered would make adding a command break unrelated tests.
var testCmds = []command{
	{name: "help", summary: "list the commands"},
	{name: "history", summary: "words looked up recently"},
	{name: "stats", summary: "deck statistics"},
}

func TestParseCommandLine(t *testing.T) {
	for _, tc := range []struct {
		in   string
		name string
		args []string
		ok   bool
	}{
		{"/history", "history", nil, true},
		{"/history 7", "history", []string{"7"}, true},
		{"/history --days 7", "history", []string{"--days", "7"}, true},
		{"  /history  ", "history", nil, true},
		{"/history   7", "history", []string{"7"}, true},
		// A bare slash is command mode with nothing typed yet — the UI wants to
		// offer every command, so it is ok=true with an empty name.
		{"/", "", nil, true},
		{"  /  ", "", nil, true},
		// Not commands. "hot dog" is a real headword, and the empty line is the
		// replay gesture #2 defined.
		{"hot dog", "", nil, false},
		{"", "", nil, false},
		{"   ", "", nil, false},
		// A slash that is not in the first column is part of the word.
		{"and/or", "", nil, false},
	} {
		t.Run(tc.in, func(t *testing.T) {
			name, args, ok := parseCommandLine(tc.in)
			if ok != tc.ok || name != tc.name || !reflect.DeepEqual(args, tc.args) {
				t.Errorf("parseCommandLine(%q) = (%q, %v, %v), want (%q, %v, %v)",
					tc.in, name, args, ok, tc.name, tc.args, tc.ok)
			}
		})
	}
}

func TestCommandCompletions(t *testing.T) {
	for _, tc := range []struct {
		prefix string
		want   []string
	}{
		{"", []string{"/help", "/history", "/stats"}}, // bare slash offers everything
		{"h", []string{"/help", "/history"}},
		{"his", []string{"/history"}},
		{"history", []string{"/history"}},
		{"zzz", nil},
		// Case-SENSITIVE by policy: Suggestion byte-prefix-matches the typed
		// line, so a completion has to literally extend it. Dispatch is the
		// forgiving half — see TestDispatchCommand's mixed-case row.
		{"HIS", nil},
	} {
		t.Run(tc.prefix, func(t *testing.T) {
			if got := commandCompletions(tc.prefix, testCmds); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("commandCompletions(%q) = %v, want %v", tc.prefix, got, tc.want)
			}
		})
	}
}

func TestNearestCommands(t *testing.T) {
	for _, tc := range []struct {
		name      string
		want      []string
		wantClose bool
	}{
		{"histry", []string{"/history"}, true},  // one deletion
		{"hisotry", []string{"/history"}, true}, // one transposition
		{"h", []string{"/help", "/history"}, true},
		// Nothing close: offer the whole menu rather than a confident wrong
		// guess, and SAY it is not close so the caller need not infer it.
		{"qqqqqq", []string{"/help", "/history", "/stats"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, close := nearestCommands(tc.name, testCmds)
			if !reflect.DeepEqual(got, tc.want) || close != tc.wantClose {
				t.Errorf("nearestCommands(%q) = %v, %v; want %v, %v", tc.name, got, close, tc.want, tc.wantClose)
			}
		})
	}
}

// BR-9 measured that with ONE command registered, len(near) == len(cmds) is true
// for every near-miss — so a caller inferring "nothing was close" from that got
// the wrong answer on a genuine near-miss. This is the regression pin.
func TestNearMissWithOneRegisteredCommand(t *testing.T) {
	one := []command{{name: "help", summary: "list the commands", run: runHelp}}
	if got, close := nearestCommands("hel", one); !close || !reflect.DeepEqual(got, []string{"/help"}) {
		t.Errorf("nearestCommands(\"hel\", one) = %v, %v; want [/help], true", got, close)
	}
	var out, errb bytes.Buffer
	dispatchCommand(parseREPLLine("/hel", false), one, commandCtx{stdout: &out, stderr: &errb})
	if !strings.Contains(errb.String(), "did you mean /help") {
		t.Errorf("a near miss against a one-command registry did not suggest: %q", errb.String())
	}
}

// BR-10: a row with a nil run panics on dispatch. The registry is data, so the
// check is a test rather than a runtime branch.
func TestEveryRegisteredCommandIsRunnable(t *testing.T) {
	if len(commands) == 0 {
		t.Fatal("the registry is empty; this test would pass vacuously")
	}
	for _, c := range commands {
		if c.run == nil {
			t.Errorf("command /%s has a nil run", c.name)
		}
		if c.summary == "" {
			t.Errorf("command /%s has no summary; /help would print a blank line", c.name)
		}
	}
}

// BR-31: a fix that exempted commands from withStore dropped an invariant it was
// not thinking about — deps.clock is supplied there, so the exemption stranded
// it as nil and any row that read the clock would have panicked. The exemption
// is gone, but the guard is the durable part: assert the ctx a one-shot ACTUALLY
// builds carries every field a row may read.
func TestOneShotCommandCtxIsFullyPopulated(t *testing.T) {
	t.Chdir(t.TempDir())
	d := deps{newStore: openStore}.withStore(options{}, io.Discard)
	cc := newCommandCtx(d, options{times: 3, width: 80}, io.Discard, io.Discard)

	if cc.clock == nil {
		t.Error("clock is nil; a command that reads it would panic")
	}
	if cc.stdout == nil || cc.stderr == nil {
		t.Error("a command has nowhere to write")
	}
	if cc.deck == nil {
		t.Error("deck is nil in a writable directory")
	}
	if cc.times == 0 {
		t.Error("times did not reach the ctx")
	}
}

// The namespace switch. One decision point for "which set is this line drawing
// from", so the call sites in replraw.go cannot drift apart.
func TestCompletionsFor(t *testing.T) {
	h := &memHistory{}
	h.Add("sycophantic")
	h.Add("sybarite")
	h.Add("ephemeral")

	for _, tc := range []struct {
		name string
		base string
		want []string
	}{
		{"command prefix draws from commands", "/his", []string{"/history"}},
		{"bare slash offers every command", "/", []string{"/help", "/history", "/stats"}},
		{"a word draws from history", "syc", []string{"sycophantic"}},
		{"a shared word prefix draws several", "sy", []string{"sybarite", "sycophantic"}},
		{"an empty line draws all history", "", []string{"ephemeral", "sybarite", "sycophantic"}},
		{"a slash mid-word is not a command", "and/or", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := completionsFor(tc.base, h, testCmds)
			// History order is newest-first; compare as sets so this test is about
			// WHICH namespace was consulted, not about either one's ordering.
			if !sameSet(got, tc.want) {
				t.Errorf("completionsFor(%q) = %v, want %v (as a set)", tc.base, got, tc.want)
			}
		})
	}
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		seen[s]--
		if seen[s] < 0 {
			return false
		}
	}
	return true
}

// Typing "/" must SHOW what there is. The inline grey suggestion completes one
// candidate but never reveals the set, so command mode was undiscoverable
// (operator, 2026-08-21: "hard to use").
func TestMenuLines(t *testing.T) {
	for _, tc := range []struct {
		name  string
		base  string
		width int
		want  []string
	}{
		{"a bare slash shows everything", "/", 0, []string{
			"  /help     list the commands",
			"  /history  words looked up recently",
			"  /stats    deck statistics",
		}},
		{"a prefix filters", "/h", 0, []string{
			"  /help     list the commands",
			"  /history  words looked up recently",
		}},
		{"a longer prefix narrows further", "/his", 0, []string{
			"  /history  words looked up recently",
		}},
		// No menu at all outside command mode: a word being typed must not have
		// the screen jump under it.
		{"a word shows nothing", "syc", 0, nil},
		{"an empty line shows nothing", "", 0, nil},
		// Nothing matches: the menu disappears rather than showing a stale set.
		{"no match shows nothing", "/zzz", 0, nil},
		// Once an argument is typed the command is settled.
		{"an argument settles it", "/history 7", 0, nil},
		{"narrow terminals truncate", "/his", 14, []string{"  /history  wo"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := menuLines(tc.base, testCmds, tc.width); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("menuLines(%q, w=%d) =\n  %q\nwant\n  %q", tc.base, tc.width, got, tc.want)
			}
		})
	}
}

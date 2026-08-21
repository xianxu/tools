package main

import (
	"reflect"
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
		// Completion is over the name only; an argument already typed means the
		// command is settled and there is nothing left to narrow.
		{"HIS", []string{"/history"}}, // case-insensitive: typing is not a spelling test
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
		name string
		want []string
	}{
		{"histry", []string{"/history"}},  // one deletion
		{"hisotry", []string{"/history"}}, // one transposition
		{"h", []string{"/help", "/history"}},
		// Nothing close: offer the whole menu rather than a confident wrong guess.
		{"qqqqqq", []string{"/help", "/history", "/stats"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := nearestCommands(tc.name, testCmds); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("nearestCommands(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

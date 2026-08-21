package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// command is one thing the REPL can do that is not a lookup.
//
// A `run` field arrives with dispatch; at this point the table is data, which is
// what lets the functions below be pure and table-tested with a fixture set.
type command struct {
	name    string
	summary string
	run     func(commandCtx, []string) int
}

// commands is the registry. Adding a command is a row here plus its run
// function — the dispatch loop never changes, which is a Done-when.
var commands = []command{
	{name: "help", summary: "list the commands", run: runHelp},
}

// completionsFor is the ONE place that decides which namespace a line is drawing
// from, and it is why command-mode type-ahead needed no change to the pure
// editor: Apply already takes its candidate list from the caller, so command
// mode is a different match SOURCE rather than a different editor.
//
// The candidates come back "/"-prefixed because Suggestion matches against the
// whole typed line — with "/his" typed, "/history" is what completes it.
//
// Once an argument has been typed ("/history 7") the command is settled and the
// completion is shorter than the line, so Suggestion offers nothing. That falls
// out rather than being special-cased.
func completionsFor(base string, hist History, cmds []command) []string {
	if name, _, ok := parseCommandLine(base); ok {
		return commandCompletions(name, cmds)
	}
	return hist.Prefix(base)
}

// parseCommandLine reports whether a submitted line is a command, and splits it.
//
// `/` in the FIRST column is the marker. No English headword starts with one, so
// the namespace cannot collide with a lookup — which matters because `define`
// takes multi-word headwords ("hot dog"), so the namespace had to be a character
// rather than a reserved word. A slash anywhere else is part of the word:
// "and/or" is a lookup.
//
// A bare "/" is command mode with nothing typed yet, not an error: the UI offers
// the whole menu, so it returns ok with an empty name.
func parseCommandLine(line string) (name string, args []string, ok bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "/") {
		return "", nil, false
	}
	fields := strings.Fields(line[1:])
	if len(fields) == 0 {
		return "", nil, true
	}
	if len(fields) == 1 {
		return fields[0], nil, true
	}
	return fields[0], fields[1:], true
}

// commandCompletions returns the commands whose names begin with prefix, as the
// user would type them. Sorted, so the suggestion a keystroke produces does not
// depend on registry order.
//
// Case-insensitive: typing a command is not a spelling test.
func commandCompletions(prefix string, cmds []command) []string {
	prefix = strings.ToLower(prefix)
	var out []string
	for _, c := range cmds {
		if strings.HasPrefix(c.name, prefix) {
			out = append(out, "/"+c.name)
		}
	}
	sort.Strings(out)
	return out
}

// nearestCommands answers "you typed something that is not a command" — by
// prefix if anything matches, then by edit distance, and failing both by showing
// the whole menu. Offering everything is deliberately better than a confident
// wrong guess: the Done-when asks for an unknown command to SUGGEST rather than
// silently define something.
func nearestCommands(name string, cmds []command) []string {
	if hits := commandCompletions(name, cmds); len(hits) > 0 {
		return hits
	}
	name = strings.ToLower(name)
	var out []string
	for _, c := range cmds {
		if editDistance(name, c.name) <= 2 {
			out = append(out, "/"+c.name)
		}
	}
	if len(out) == 0 {
		return commandCompletions("", cmds)
	}
	sort.Strings(out)
	return out
}

// editDistance is Levenshtein, two rows rather than a full matrix — the inputs
// are command names, so this is about clarity, not speed.
func editDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, min(curr[j-1]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[len(br)]
}

// commandCtx is what a command may touch. Deliberately NOT the whole deps: a
// command has no business reaching the dictionary or the player, and a narrow
// struct makes that structural rather than a convention.
//
// cmds is here so /help can list the table it was dispatched from, which keeps
// the fixture set in tests honest — help lists what dispatch would actually run.
type commandCtx struct {
	cmds   []command
	stdout io.Writer
	stderr io.Writer
	width  int
}

// dispatchCommand runs a parsed command, or explains why it cannot.
//
// The loop never grows a case: adding a command is a row in `commands`. That is
// a Done-when, so it is worth stating that the switch below is on OUTCOME
// (found / not found), never on which command it is.
func dispatchCommand(c replCommand, cmds []command, cc commandCtx) int {
	cc.cmds = cmds
	if c.name == "" { // a bare "/" was submitted: show what there is
		return runHelp(cc, nil)
	}
	for _, cmd := range cmds {
		if strings.EqualFold(cmd.name, c.name) {
			return cmd.run(cc, c.args)
		}
	}
	near := nearestCommands(c.name, cmds)
	if len(near) == len(cmds) {
		// Nothing was close, so "did you mean" would be a lie about all of them.
		fmt.Fprintf(cc.stderr, "define: unknown command /%s. Commands: %s\n", c.name, strings.Join(near, " "))
	} else {
		fmt.Fprintf(cc.stderr, "define: unknown command /%s; did you mean %s?\n", c.name, strings.Join(near, " or "))
	}
	return 2
}

func runHelp(c commandCtx, _ []string) int {
	for _, cmd := range c.cmds {
		fmt.Fprintf(c.stdout, "  /%-10s %s\n", cmd.name, cmd.summary)
	}
	return 0
}

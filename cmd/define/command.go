package main

import (
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

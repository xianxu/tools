package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
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
	{name: "history", summary: "words looked up recently", run: runHistory},
	{name: "sound", summary: "how many times to play a pronunciation", run: runSound},
	{name: "lang", summary: "the language this deck is in", run: runLang},
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
	return historyCompletions(base, hist)
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
// Case-SENSITIVE, deliberately, and the asymmetry with dispatch is the point:
// Suggestion does a byte-prefix match against the typed line, so a completion
// must literally extend what was typed — "/HIS" cannot be completed by
// "/history" without rewriting the user's keystrokes. Dispatch stays forgiving
// (EqualFold), so a submitted "/HELP" still runs. Complete exactly, accept
// loosely.
func commandCompletions(prefix string, cmds []command) []string {
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
// The second return says whether anything was actually CLOSE, rather than
// leaving the caller to infer it. Inferring it from len(near) == len(cmds) is
// true for every near-miss while only one command is registered, so the
// "did you mean" branch had zero production reachability (BR-9, measured:
// `/hel` printed the whole menu, not a suggestion).
func nearestCommands(name string, cmds []command) (matches []string, close bool) {
	if hits := commandCompletions(name, cmds); len(hits) > 0 {
		return hits, true
	}
	lower := strings.ToLower(name)
	var out []string
	for _, c := range cmds {
		if editDistance(lower, c.name) <= 2 {
			out = append(out, "/"+c.name)
		}
	}
	if len(out) == 0 {
		return commandCompletions("", cmds), false
	}
	sort.Strings(out)
	return out, true
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
	deck   store.Store // nil when there is nowhere to read
	clock  store.Clock
	stdout io.Writer
	stderr io.Writer
	width  int
	// times is the current playback count, and setTimes changes it for the rest
	// of the session. A func rather than a *options: a command has no business
	// reaching the rest of the options, and nil is the honest representation of
	// "there is no session here" for the one-shot and piped paths.
	times    int
	setTimes func(int)
	// noCapture only so a nil deck can say WHY. DEFINE_NO_CAPTURE means the
	// deck was never opened; without it, nil means this directory has none.
	noCapture bool
	// lang is the language in effect, and setLang changes it — the /sound pairing
	// above, with one deliberate difference. /sound is explicitly "for the rest of
	// this session", so a nil setTimes means /sound must REFUSE. Language
	// persists, so setLang's durable half works without a session: a one-shot
	// `define /lang es` has no loop to change but a directory to write. nil here
	// means there is no directory either, which is the only case /lang refuses.
	lang    store.Lang
	setLang func(store.Lang) error
}

// newCommandCtx is the single construction point. Built at two call sites (both
// loops) and M2 adds a deck and a clock, so a literal in each loop is two places
// to forget a field (ARCH-DRY).
// width comes from opt, which run() computes ONCE. An earlier version re-derived
// it per dispatch via terminalWidth(stdout) — a second source that can disagree
// with opt.width across a resize.
func newCommandCtx(d deps, opt options, stdout, stderr io.Writer) commandCtx {
	return commandCtx{
		deck: d.deck, clock: d.clock,
		stdout: stdout, stderr: stderr,
		width: opt.width, noCapture: opt.noCapture,
		times: opt.times,
		lang:  d.lang,
		// The DURABLE half only. Both loops override this with a version that
		// also re-derives the session; a one-shot keeps this one, which is why
		// `define /lang es` still sets the directory's language.
		setLang: d.persistLang,
	}
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
	near, close := nearestCommands(c.name, cmds)
	if close {
		fmt.Fprintf(cc.stderr, "define: unknown command /%s; did you mean %s?\n", c.name, strings.Join(near, " or "))
	} else {
		// Nothing was close, so "did you mean" would be a lie about all of them.
		fmt.Fprintf(cc.stderr, "define: unknown command /%s. Commands: %s\n", c.name, strings.Join(near, " "))
	}
	return 2
}

func runHelp(c commandCtx, _ []string) int {
	for _, cmd := range c.cmds {
		fmt.Fprintf(c.stdout, "  /%-10s %s\n", cmd.name, cmd.summary)
	}
	// The two hatches are one keystroke each and otherwise invisible: nothing on
	// screen suggests a line can be forced either way. This is the only place
	// that lists what the console understands, so it is where they go.
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "  a word is defined, a question is asked — no mode to switch")
	fmt.Fprintln(c.stdout, `  ?  ask, even if it is a word     \  define, even if it reads as a question`)
	return 0
}

// menuLines is the list shown UNDER the prompt in command mode: every command
// matching what has been typed so far, narrowing as you type.
//
// Separate from commandCompletions because they answer different questions.
// Completion answers "what single string extends this line" and feeds the grey
// inline suggestion; the menu answers "what are my choices", which is the one
// the user actually needs first — the inline suggestion completes a command you
// already know the name of, and reveals nothing to someone who does not.
//
// Returns nil outside command mode, and nil when nothing matches: the menu
// vanishing is the right answer to "/zzz", not a stale set left on screen.
func menuLines(base string, cmds []command, width int) []string {
	name, args, ok := parseCommandLine(base)
	if !ok || len(args) > 0 { // an argument means the command is settled
		return nil
	}
	var out []string
	for _, c := range cmds {
		if !strings.HasPrefix(c.name, name) {
			continue
		}
		line := fmt.Sprintf("  /%-*s%s", menuNameWidth(cmds), c.name, c.summary)
		line = truncate(line, width)
		out = append(out, line)
	}
	sort.Strings(out)
	return out
}

// truncate cuts a line to width, measuring in RUNES.
//
// Shared because the two renderers disagreed: the menu cut bytes while
// /history cut runes, so one of them would have split a multi-byte character in
// half on a narrow terminal. A width is a column count, and a column is a rune.
func truncate(s string, width int) string {
	if width <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width])
}

// menuNameWidth is the name field's width: the longest name plus a two-space
// gutter. Computed from ALL commands, not the filtered set, so the summary
// column does not shuffle sideways as the list narrows under your typing.
func menuNameWidth(cmds []command) int {
	w := 0
	for _, c := range cmds {
		if len(c.name) > w {
			w = len(c.name)
		}
	}
	return w + 2
}

// candidatesFor resolves both candidate lists for one keystroke.
//
// Two lists rather than one because they answer different questions, and #20 is
// where the answers diverged: recall is what you SUBMITTED, complete is what the
// line could BECOME. Until now every completion was itself a past line, so one
// slice served both and nobody noticed — except in command mode, where feeding
// the menu to walk made Up put "/help" on a line that had never been submitted.
func candidatesFor(base string, hist History, cmds []command) candidates {
	return candidates{
		recall:   hist.Prefix(base),
		complete: completionsFor(base, hist, cmds),
	}
}

// sessionSetLang lifts /lang's durable half into a full session switch.
//
// The durable half (persist) is what newCommandCtx supplies and what a one-shot
// run keeps. A LOOP can do more: it re-derives the language-scoped dependencies
// so the rest of the session reads the new deck. Both halves, in that order —
// persisting first means a failure to write is reported before anything visible
// changes, rather than leaving the session and the directory disagreeing.
//
// d is taken by POINTER on purpose. Both loops hold their deps by value, and the
// switch has to outlive one dispatch: commandCtx is rebuilt per command, so
// writing through a copy would be forgotten by the next line the learner types.
//
// vocPtr is the raw editor's cached highlight set, or nil for the loop that has
// none. That parameter exists because the editor resolves the set ONCE before
// its loop — a d swap cannot reach that local, and without this the editor would
// keep highlighting the old language's words.
func sessionSetLang(d *deps, opt options, persist func(store.Lang) error, vocPtr *Vocabulary) func(store.Lang) error {
	if persist == nil {
		// No directory: /lang has nothing durable to do, so there is no session
		// switch worth making either. nil is what makes the command say so.
		return nil
	}
	return func(l store.Lang) error {
		if err := persist(l); err != nil {
			return err
		}
		d.lang = l
		if d.newDeck != nil {
			d.deck, d.capture, d.vocab = d.newDeck(l)
			if vocPtr != nil {
				// The REAL options, not a fabricated one: vocabularyFor owns
				// "loaded, and only with colour", and forcing colour on here
				// would resurrect highlighting under -no-color.
				*vocPtr = vocabularyFor(*d, opt)
			}
		}
		return nil
	}
}

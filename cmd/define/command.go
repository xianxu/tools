package main

import (
	"errors"
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
	// args is the argument synopsis that follows the name — "[N | --days N |
	// --days=N]" — and empty for a command that takes none.
	args string
	// usage says how to use the command: what each form of its arguments does,
	// and what it does bare. `/help <name>` and `/<name> --help` print it, and
	// the docs quote it through the command-usage span (#53), so a command's
	// argument rule is stated once, beside the parser that implements it.
	usage string
	run   func(commandCtx, []string) int
}

// synopsis is the command as it would be typed: its name and, only when it
// takes any, its arguments. The ONE builder, so the screen and the docs' span
// cannot disagree about a trailing space on a command that takes nothing.
func (c command) synopsis() string {
	if c.args == "" {
		return "/" + c.name
	}
	return "/" + c.name + " " + c.args
}

// commands is the registry. Adding a command is a row here plus its run
// function — the dispatch loop never changes, which is a Done-when. The row's
// usage is required (TestEveryRegisteredCommandIsRunnable): /help <name> prints
// it and the docs quote it.
var commands = []command{
	{name: "help", summary: "list the commands, or explain one", args: "[command]", usage: helpUsage, run: runHelp},
	{name: "history", summary: "words looked up recently", args: "[N | --days N | --days=N]", usage: historyUsage, run: runHistory},
	{name: "stats", summary: "deck, streak and accuracy figures", usage: statsUsage, run: runStatsCommand},
	{name: "play", summary: "review the words due today", usage: playUsage, run: runPlayCommand},
	{name: "sound", summary: "how many times to play a pronunciation", args: "[N]", usage: soundUsage, run: runSound},
	{name: "lang", summary: "the language this deck is in", args: "[language]", usage: langUsage, run: runLang},
	{name: "pron", summary: "replay this word in its source language, once", args: "[language]", usage: pronUsage, run: runPron},
}

// helpUsage is /help's own row. Its argument is a command's NAME, with or
// without the slash, resolved the way dispatch resolves one.
const helpUsage = "With nothing, list the commands. With a command's name, say how to use it, which --help after any command also does."

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
	cmds []command
	deck store.Store // nil when there is nowhere to read
	// deckPermission answers "is anything written here going to survive", WITHOUT
	// asking. /stats needs to say so and must not turn reading into a question.
	deckPermission *deckPermission
	clock          store.Clock
	stdout         io.Writer
	stderr         io.Writer
	width          int
	// times is the current playback count, and setTimes changes it for the rest
	// of the session. A func rather than a *options: a command has no business
	// reaching the rest of the options, and nil is the honest representation of
	// "there is no session here" for the one-shot and piped paths.
	times    int
	setTimes func(int)
	// startSitting asks the LOOP to run today's review, and is nil wherever one
	// cannot run: the one-shot path, a pipe, and the line-mode REPL, which has no
	// raw terminal. Nil IS the refusal — the rule setTimes states above.
	//
	// A func rather than a bool, and the loop supplies it only when it can honour
	// it, so "can I" and "do it" are one fact rather than two that can disagree.
	// That is cc.replay's shape (#48).
	startSitting func()
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
	// dictName is which dictionary is answering, so /lang can say. Empty when
	// nothing resolved one — a test's fake, or a run that never opened a
	// dictionary at all.
	dictName string
	// entry is the RAW dictionary text of the current word, so /pron can read
	// its ORIGIN (#35).
	//
	// DATA, not a capability. commandCtx is deliberately narrower than deps — a
	// command may not reach the dictionary or the player — and this is text the
	// session already holds, the same kind of thing as lang and dictName. The
	// inference lives in the COMMAND because the command owns the message that
	// explains it; putting it in the loop would split the decision from its
	// explanation.
	//
	// Empty when nothing has been looked up, which /pron reports rather than
	// inferring from nothing.
	entry string
	// replay asks the LOOP to play the current word once in another language
	// (#29). A closure, like setTimes and setLang, rather than the Player: a
	// command still cannot reach the dictionary or the player, it can only ask
	// for the one thing /pron means.
	//
	// It RECORDS rather than plays — see runPron for why the raw editor cannot
	// have a command play in place. nil where there is no current word to
	// replay, which is what lets /pron say so instead of playing silence.
	replay func(store.Lang)
}

// newCommandCtx is the single construction point. Built at two call sites (both
// loops) and M2 adds a deck and a clock, so a literal in each loop is two places
// to forget a field (ARCH-DRY).
// width comes from opt, which run() computes ONCE. An earlier version re-derived
// it per dispatch via terminalWidth(stdout) — a second source that can disagree
// with opt.width across a resize.
func newCommandCtx(d deps, opt options, stdout, stderr io.Writer) commandCtx {
	return commandCtx{
		deck: d.deck, clock: d.clock, deckPermission: d.deckPermission,
		stdout: stdout, stderr: stderr,
		width: opt.width, noCapture: opt.noCapture,
		times:    opt.times,
		lang:     d.lang,
		dictName: d.dictName,
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
	cmd, ok := findCommand(c.name, cmds)
	if !ok {
		return unknownCommand(cc.stderr, c.name, cmds)
	}
	return cmd.run(cc, c.args)
}

// findCommand resolves a submitted name to its row. Case-INSENSITIVE: dispatch's
// half of the case policy commandCompletions describes (complete exactly, accept
// loosely). /help resolves through here too, so it accepts what dispatch does.
func findCommand(name string, cmds []command) (command, bool) {
	for _, c := range cmds {
		if strings.EqualFold(c.name, name) {
			return c, true
		}
	}
	return command{}, false
}

// unknownCommand explains a name that is not a command and returns dispatch's
// exit code for it. One wording for two routes to the same mistake — a submitted
// `/histry` and `/help histry` — so they cannot drift apart.
func unknownCommand(stderr io.Writer, name string, cmds []command) int {
	near, close := nearestCommands(name, cmds)
	if close {
		fmt.Fprintf(stderr, "define: unknown command /%s; did you mean %s?\n", name, strings.Join(near, " or "))
	} else {
		// Nothing was close, so "did you mean" would be a lie about all of them.
		fmt.Fprintf(stderr, "define: unknown command /%s. Commands: %s\n", name, strings.Join(near, " "))
	}
	return 2
}

// commandUsage is what `/help <name>` and `/<name> --help` print for one
// command: the synopsis, then the usage wrapped to width under the command
// list's two-space indent. Pure — a row and a width in, the text out.
func commandUsage(c command, width int) string {
	return "  " + c.synopsis() + "\n  " + wrapText(c.usage, width, 2) + "\n"
}

// runHelp is /help: bare, the command list; with a command's name, that
// command's usage (#53).
func runHelp(c commandCtx, args []string) int {
	switch {
	case len(args) > 1:
		fmt.Fprintf(c.stderr, "define: /help takes one command, not %q\n", strings.Join(args, " "))
		return 2
	case len(args) == 1 && strings.TrimPrefix(args[0], "/") != "":
		// ONE command, resolved exactly as dispatch resolves a submitted one, so
		// `/help HISTORY` works because `/HISTORY` does, and a name dispatch
		// would refuse is refused here in dispatch's own words. `/help /` falls
		// through to the list, as a bare `/` does.
		name := strings.TrimPrefix(args[0], "/")
		cmd, ok := findCommand(name, c.cmds)
		if !ok {
			return unknownCommand(c.stderr, name, c.cmds)
		}
		fmt.Fprint(c.stdout, commandUsage(cmd, c.width))
		return 0
	}
	for _, cmd := range c.cmds {
		fmt.Fprintf(c.stdout, "  /%-10s %s\n", cmd.name, cmd.summary)
	}
	// The way past the summary, stated where the summary is: a command's
	// arguments live in its usage, and nothing else on screen says so.
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "  /help <command>, or --help after one, says how to use it")
	// The two hatches are one keystroke each and otherwise invisible: nothing on
	// screen suggests a line can be forced either way. This is the only place
	// that lists what the console understands, so it is where they go.
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "  a word is defined, a question is asked — no mode to switch")
	fmt.Fprintln(c.stdout, `  ?  ask, even if it is a word     \  define, even if it reads as a question`)
	// The screen and its one cost (#30). Mouse reporting is what makes the wheel
	// a scroll rather than a history walk, and it takes drag-select away from the
	// terminal — a real regression for anyone who copies definitions, so the
	// escape is stated in the one place that lists what the console understands
	// rather than left for a user to discover by failing to select a word.
	fmt.Fprintln(c.stdout)
	fmt.Fprintln(c.stdout, "  wheel or PageUp/PageDown scrolls   hold Option to select text")
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

// truncate cuts a line to width, measuring in terminal COLUMNS.
//
// Shared because the two renderers disagreed: the menu cut bytes while /history
// cut runes, so one of them would have split a multi-byte character in half on a
// narrow terminal. A width is a column count — and a column is not a rune, which
// is why this now delegates to the screen's one owner of that arithmetic: a
// combining mark takes no column and a CJK rune takes two.
func truncate(s string, width int) string {
	return clipVisible(s, width)
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
// run keeps. A LOOP can do more: it re-derives everything downstream of the
// language. Both halves, in that order — persisting first means a failure to
// write is reported before anything visible changes, rather than leaving the
// session and the directory disagreeing.
//
// d and opt are taken by POINTER on purpose. Both loops hold theirs by value,
// and the switch has to outlive one dispatch: commandCtx is rebuilt per command,
// so writing through a copy would be forgotten by the next line typed.
//
// vocPtr is the raw editor's cached highlight set, or nil for the loop that has
// none. See applyLang for why that parameter exists.
func sessionSetLang(d *deps, opt *options, persist func(store.Lang) error, vocPtr *Vocabulary, warn io.Writer) func(store.Lang) error {
	if persist == nil {
		// No directory: /lang has nothing durable to do, so there is no session
		// switch worth making either. nil is what makes the command say so.
		return nil
	}
	return func(l store.Lang) error {
		err := persist(l)
		// A DECLINE IS NOT A FAILURE, and the two halves part company here
		// (#50 BR-18). Persisting is the DURABLE half; applyLang is the SESSION
		// half, and a directory the learner declined stops only the first. An
		// earlier version returned on any error, so a declined /lang left the
		// session still in English while the command printed a switch — the same
		// lie in the other direction, caught by
		// TestLangSwitchReDerivesEverythingDownstreamOfTheLanguage.
		if err != nil && !errors.Is(err, errDeckDeclined) {
			return err
		}
		applyLang(d, opt, l, vocPtr, warn)
		return err // nil, or the decline for the caller to report honestly
	}
}

// applyLang re-derives everything that is a function of the language.
//
// THE ENUMERATION IS THE POINT, and the rule that generates it is: anything
// derived from the language BEFORE a switch must be re-derived BY the switch.
// Listing the members here, in one function, is what stops the next one from
// being missed — an earlier version enumerated four of the five by hand at the
// call site and shipped a session whose deck was Spanish while its pronunciation
// stayed English.
//
// The members, and why each is one:
//
//   - d.lang        — the answer everything else reads.
//   - langDeps      — the deck, capturer, vocabulary and usage source, taken
//     WHOLE from the one builder openStore used. It is a struct
//     rather than a list precisely because a list in this comment
//     already failed once: M2 made the news feed language-dependent
//     and this enumeration still called usage "not language-scoped".
//   - opt.voice     — the CDN asks per language; derived via applyVoice, the
//     same function the boundary uses.
//   - d.dict /
//     d.dictName    — #23 M2: the dictionary follows the mode. Registered in
//     this list while it was still M2's to build, which is the
//     point of writing an enumeration down rather than sweeping
//     by hand.
//   - *vocPtr       — the raw editor resolves the highlight set into a LOCAL
//     before its loop and reads it on every redraw. A deps
//     reassignment structurally cannot reach that local; without
//     this the editor paints the old language's words. nil for
//     the piped loop, which has no such local.
//
// Deliberately NOT here: d.history. events/ is not language-scoped, and
// rebuilding it would orphan the one runEditor has already Load()ed while
// everything else read a fresh empty one.
func applyLang(d *deps, opt *options, l store.Lang, vocPtr *Vocabulary, warn io.Writer) {
	d.lang = l
	if d.newDict != nil {
		d.dict, d.dictName = d.newDict(l, warn)
	}
	// NOT opt.lang: that field is the -lang FLAG, documented as "empty when it
	// was not given", and a switch does not retroactively make the flag present.
	// d.lang is the language in effect and the only thing that should answer it.
	applyVoice(opt, l)
	if d.newLangDeps == nil {
		return // no store here; the language still applies to everything else
	}
	// The WHOLE set, in one assignment. Copying members individually is what let
	// d.usage be forgotten, and it stayed forgettable even after the set became a
	// struct, because both sites still spelled the fields out. langDeps is
	// embedded in deps, so this adopts every member including ones added later.
	d.langDeps = d.newLangDeps(l)
	if vocPtr != nil {
		// The REAL options, not a fabricated one: vocabularyFor owns "loaded,
		// and only with colour", and forcing colour on here would resurrect
		// highlighting under -no-color.
		*vocPtr = vocabularyFor(*d, *opt)
	}
}

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
	"golang.org/x/term"
)

// deps is the injected IO surface. Keeping it explicit is what lets run() be
// driven end-to-end by fakes (ARCH-PURE): main() supplies the real ones, tests
// supply recorders.
type deps struct {
	dict   Dictionary
	audio  AudioSource
	player Player
	// history is the durable word history. Constructed at the boundary so the
	// loop takes a seam rather than deciding where state lives.
	history History
	// capture is the only thing that RECORDS lookups. deck below is the other
	// way the store is mutated: --forget deletes through it.
	capture Capturer
	// deck is the store --forget acts on. Separate from capture because capture
	// deliberately cannot fail loudly and --forget deliberately must.
	deck store.Store
	// newStore builds the three store-backed dependencies AFTER flags are parsed —
	// it cannot happen in realDeps, because DEFINE_NO_CAPTURE is read at flag
	// parse and decides whether anything is opened at all. Tests leave it nil and
	// get in-memory defaults. Tests that DO want the real wiring set it to
	// openStore and t.Chdir into a t.TempDir first — several do, so this seam is
	// what keeps the real filesystem opt-in, not unreachable.
	newStore func(options, io.Writer) storeDeps
	// clock is what a command reads to answer "now". Injected for the same
	// reason storeCapturer's is: /history's window is a local-DAY computation,
	// so a test has to be able to stand at a chosen instant in a chosen zone.
	clock store.Clock
	// stdinIsTerminal decides whether the loop prints a prompt. Injected rather
	// than probed directly because a test harness's stdin is never a terminal,
	// which would make the interactive path unwritable. Note this is a different
	// question from the stdout check that drives colour.
	stdinIsTerminal func() bool
}

func realDeps() deps {
	return deps{
		dict:            systemDictionary(),
		audio:           newHTTPAudioSource(),
		player:          afplayPlayer{},
		newStore:        openStore,
		stdinIsTerminal: func() bool { return isTerminal(os.Stdin) },
	}
}

// storeDeps is the trio openStore produces. One value rather than three returns
// and three nil-merges at the call site: they are always built together, always
// consumed together, and the merge was three chances to forget one.
type storeDeps struct {
	history History
	capture Capturer
	deck    store.Store
	// clock is the process's ONE answer to "what time is it". It used to be
	// constructed inline where the capturer was built, so nothing else could
	// reach it — and #15's /history needs the same clock to compute a local-day
	// window. Two clocks would be two answers, and a test could only move one.
	//
	// Supplied on every path, including the opt-out: DEFINE_NO_CAPTURE means
	// "write nothing here", not "time does not exist", and a command that reads
	// the log still needs one.
	clock store.Clock
}

// withStore fills any store-backed dependency a caller did not supply, leaving
// whatever a test supplied alone.
//
// It used to short-circuit when history and capture were both supplied. No test
// ever entered that branch (probe-verified: a panic there left the suite green)
// and it left deck nil, so --forget had nothing to act on. The nil-merge below
// reaches the same result without a second path through the function.
func (d deps) withStore(opt options, warn io.Writer) deps {
	var sd storeDeps
	if d.newStore != nil {
		sd = d.newStore(opt, warn)
	}
	if d.history == nil {
		d.history = orElse[History](sd.history, &memHistory{})
	}
	if d.capture == nil {
		d.capture = orElse[Capturer](sd.capture, noopCapturer{})
	}
	if d.deck == nil {
		d.deck = sd.deck
	}
	// Same shape as the memHistory/noopCapturer fallbacks above: a test that
	// supplies no newStore still gets a usable process. A test that wants to
	// control time sets d.clock and it survives.
	d.clock = orElse[store.Clock](d.clock, orElse[store.Clock](sd.clock, store.SystemClock()))
	return d
}

func orElse[T comparable](v, fallback T) T {
	var zero T
	if v == zero {
		return fallback
	}
	return v
}

// openStore builds the store-backed dependencies over the WORKING DIRECTORY.
//
// DEFINE_NO_CAPTURE means "write nothing in this directory", and that has a real
// cost: persisted history IS the event log (#3), so opting out also drops
// history to session-only. Stated here, in --help, and in the README, rather
// than discovered.
//
// A store that cannot be opened must not break define: warn and fall back,
// exactly as a missing recording degrades rather than fails. Someone in a
// read-only directory still gets a dictionary.
func openStore(opt options, warn io.Writer) storeDeps {
	// NOT a second copy of the capture policy: this decides whether there is
	// anywhere to write at all. decideCapture stays the only thing that decides
	// whether a given lookup counts.
	clk := store.SystemClock()
	if opt.noCapture {
		return storeDeps{history: &memHistory{}, capture: noopCapturer{}, clock: clk}
	}
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(warn, "define: no working directory (%v); history is session-only\n", err)
		return storeDeps{history: &memHistory{}, capture: noopCapturer{}, clock: clk}
	}
	st := store.NewYAML(dir, warn)
	return storeDeps{
		history: newStoreHistory(st, warn),
		capture: newStoreCapturer(st, clk, warn),
		deck:    st,
		clock:   clk,
	}
}

func main() {
	// NotifyContext rather than the default SIGINT handling: Ctrl-C now cancels
	// the context, which stops afplay through exec.CommandContext and lets
	// deferred cleanup run, instead of killing the process mid-playback and
	// leaving a temp file behind. This changes the one-shot path too, deliberately.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], realDeps(), os.Stdin, os.Stdout, os.Stderr))
}

// options are the session settings: parsed once, applied to every word, whether
// that is one word from argv or many from the loop.
type options struct {
	raw     bool
	color   bool
	noAudio bool
	times   int
	locale  string
	// noCapture means "write nothing in this directory". Read ONCE here, at flag
	// parse, so the environment is an input to decideCapture rather than a second
	// mechanism beside it. Note it also drops history to session-only, because
	// persisted history IS the event log (#3) — documented beside the flag.
	noCapture bool
	// width is the terminal width for wrapping; 0 on a pipe, where a consumer
	// re-wraps for itself and baked-in breaks cannot be undone.
	width int
	// tty reports whether stdout is a terminal, which decides whether transient
	// UI can be erased. Distinct from color (same probe, different question) and
	// from stdinIsTerminal (different stream entirely).
	tty bool
}

// run is the thin IO shell: parse flags, look up, render, print, play. All of
// the reasoning lives in ParseEntry, Render and AudioCandidates, none of which
// see an io.Writer or a socket.
func run(ctx context.Context, args []string, d deps, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("define", flag.ContinueOnError)
	fs.SetOutput(stderr)
	raw := fs.Bool("raw", false, "print the unparsed dictionary entry")
	noColor := fs.Bool("no-color", false, "disable ANSI colour")
	noAudio := fs.Bool("no-audio", false, "do not fetch or play the pronunciation")
	sound := fs.Int("sound", 3, "how many times to play the pronunciation")
	// The older name for -sound. Kept working rather than removed: it is
	// documented and in people's shell history. One of them wins, and asking for
	// both is a mistyped command, not a preference to guess at.
	times := fs.Int("times", 3, "how many times to play the pronunciation (older name for -sound)")
	locale := fs.String("locale", "us", "pronunciation locale: us or gb")
	forget := fs.String("forget", "", "remove a word from the deck (events are kept)")
	llmCheck := fs.Bool("llm-check", false, "check the model configuration and exit")
	fs.Usage = func() {
		fmt.Fprint(stderr, "usage: define [flags] [word]\n\n"+
			"Looks the word up in macOS's active dictionaries — normally the New\n"+
			"Oxford American Dictionary, the one Google licenses, hence the matching\n"+
			"notation — and plays its recorded pronunciation.\n\n"+
			"With no word, reads words from stdin; on a terminal that is an\n"+
			"interactive loop — return replays the pronunciation, Ctrl-C quits.\n"+
			"A line starting with / is a command rather than a word. Type / to\n"+
			"see the list, keep typing to narrow it, Tab to complete. /history\n"+
			"shows what you looked up in the last two days (/history 7, or\n"+
			"--days 7, for a wider window); /sound sets how many times a\n"+
			"pronunciation plays for the rest of the session.\n\n"+
			"define records what you look up under words/ and events/ in the\n"+
			"CURRENT DIRECTORY, so your deck follows whichever directory you run\n"+
			"it in. A word that was found is added to the deck; a word that was\n"+
			"not is kept as history only, so typos never become vocabulary. -raw\n"+
			"records nothing, because it is for scripts.\n"+
			"DEFINE_NO_CAPTURE=1 disables that entirely; with it set, history is\n"+
			"session-only, because the event log is what persists it.\n\n"+
			"--llm-check reports whether the model seam is configured and reachable.\n"+
			"Model features degrade silently by design, so this is where they are loud.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) { // -h is a successful request for help
			return 0
		}
		return 2
	}
	if isSet(fs, "sound") && isSet(fs, "times") {
		fmt.Fprintln(stderr, "define: -sound and -times are the same setting; pass one")
		return 2
	}
	// The flag and /sound are the same setting, so they get the SAME limits —
	// -sound 1000 used to be accepted while /sound 1000 was refused at 20. And
	// the message names the flag the user actually typed, rather than the one
	// the code happens to read.
	flagName := "-times"
	if isSet(fs, "sound") {
		*times = *sound
		flagName = "-sound"
	}
	if *times < 0 {
		fmt.Fprintf(stderr, "define: %s must not be negative\n", flagName)
		return 2
	}
	if *times > maxSoundTimes {
		fmt.Fprintf(stderr, "define: %s %d would take a while to sit through; the limit is %d\n",
			flagName, *times, maxSoundTimes)
		return 2
	}
	opt := options{
		raw:   *raw,
		color: !*noColor && isTerminal(stdout),
		// -no-color means "emit no ANSI", so it disables cursor control too — the
		// flag exists for terminals that mangle escapes, and splitting its meaning
		// would leave those users with erase sequences they cannot render.
		tty:   !*noColor && isTerminal(stdout),
		width: terminalWidth(stdout),
		// -raw is the scripting form: the unparsed entry and nothing else. The
		// one-shot path already returned before playing, but the loop's replay
		// branch never consulted the flag — so a bare return under -raw fetched
		// and played. Deciding it once here makes the flag mean the same thing on
		// both paths instead of depending on which line you are on.
		noAudio:   *noAudio || *raw,
		noCapture: os.Getenv("DEFINE_NO_CAPTURE") != "",
		times:     *times,
		locale:    *locale,
	}

	// Usage errors are settled BEFORE a store is opened. A mistyped command must
	// not be the thing that creates words/ and events/ in the current directory.
	//
	// --forget is a mode, not a lookup, so it is validated and dispatched apart
	// from the argument count. Combining it with a word is two commands on one
	// line; silently honouring one of them is how -raw came to mean two different
	// things in #2.
	// --llm-check is a mode, like --forget: it answers a question about the
	// configuration rather than looking a word up, so it is dispatched before the
	// argument count is judged.
	if *llmCheck {
		return runLLMCheck(ctx, os.Getenv, llm.New, stdout, stderr)
	}
	forgetting := isSet(fs, "forget")
	// A command may take arguments, so the WHOLE argument list is one line:
	// `define /history 7` has to mean what `/history 7` means at the prompt.
	// Classifying only fs.Arg(0) made the argument count reject it as "too many
	// words" while the piped loop ran it happily (BR-20).
	oneShot := parseREPLLine(strings.Join(fs.Args(), " "), false)
	switch {
	case forgetting && *forget == "":
		fmt.Fprintln(stderr, "define: -forget needs a word")
		return 2
	case forgetting && fs.NArg() != 0:
		fmt.Fprintln(stderr, "define: -forget takes the word to remove; do not also pass one")
		return 2
	// cmdAsk is exempted for the same reason cmdCommand is: a question is
	// multi-word by nature, so counting words would reject the thing the flag
	// exists to accept (BR-20's shape).
	case !forgetting && oneShot.kind != cmdCommand && oneShot.kind != cmdAsk && fs.NArg() > 1:
		fs.Usage()
		return 2
	}

	// Store-backed dependencies are built HERE, not in realDeps: the opt-out is a
	// flag-parse-time input and decides whether anything is opened at all.
	//
	// Nothing is exempted from this. An earlier fix skipped it for commands that
	// read nothing, which dropped an invariant it was not thinking about —
	// deps.clock is supplied here, so the exemption stranded it as nil. The cost
	// it was avoiding is gone at the source instead: opening a store no longer
	// reads the log (History.Load does, when a loop is about to recall).
	d = d.withStore(opt, stderr)

	if forgetting {
		return forgetWord(d, opt, *forget, stdout, stderr)
	}

	switch fs.NArg() {
	case 0:
		// The loop needs a cancel it can call itself: in raw mode Ctrl-C arrives
		// as a byte, so signal.NotifyContext cannot deliver it and the key reader
		// must cancel instead. NotifyContext stays for the one-shot and piped
		// paths, which still receive it as a signal.
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		return repl(ctx, cancel, d, opt, stdin, stdout, stderr)
	default:
		// A command is a command from every entry mode, arguments and all.
		if oneShot.kind == cmdCommand {
			return dispatchCommand(oneShot, commands, newCommandCtx(d, opt, stdout, stderr))
		}
		// So is a question. The forced route ("?…") skips the dictionary here
		// exactly as it does at the prompt.
		if oneShot.kind == cmdAsk {
			return askUnavailable(stderr, oneShot.question)
		}
		// oneShot, not fs.Arg(0): the parsed line is what carries #16's hatches,
		// and a one-shot that re-derived the word from argv would send `define
		// "?what is X"` to the dictionary — BR-13's shape, in a new place.
		out := defineOnce(ctx, d, opt, oneShot, stdout, stderr)
		if out.ask != "" {
			// The unforced route: the dictionary missed and the line reads as a
			// question. Returning out.code here would exit 0 having printed
			// nothing, since an ask outcome carries no failure.
			return askUnavailable(stderr, out.ask)
		}
		return out.code
	}
}

// defineOnce is the whole define path for a single word: look up, render, print,
// speak. Extracted so the loop calls exactly this rather than growing a parallel
// copy (ARCH-DRY).
func defineOnce(ctx context.Context, d deps, opt options, cmd replCommand, stdout, stderr io.Writer) lookupOutcome {
	out := lookupAndRender(d, opt, cmd, stdout, stderr)
	if out.ask != "" {
		// Not this function's to answer: the caller decides where an answer is
		// rendered, because the raw loop streams it in a terminal mode this path
		// knows nothing about (#16 D6).
		return out
	}
	if out.play {
		// A missing recording is not a failed lookup: the definition is the
		// deliverable and has already been printed, so audio problems warn on
		// stderr and leave the exit code at 0.
		playAnnounced(ctx, d, opt, cmd.word, defaultIndicator(opt), stdout, stderr)
	}
	return out
}

// lookupOutcome is what one line turned out to be, once the dictionary has
// answered. It carries a third possibility the define path did not used to have:
// the line was a question. That has to travel as DATA rather than be acted on
// here, because the raw loop renders a definition cooked and streams an answer
// raw — one function cannot do both (#16 D6).
type lookupOutcome struct {
	code  int    // exit semantics, unchanged
	play  bool   // audio should follow
	ask   string // non-empty: this line is a question for the model
	entry string // the raw dictionary text, kept for the ask context
}

// lookupAndRender is also the ONE capture site. Verified against the call graph
// rather than assumed: one-shot and the line loop reach it through defineOnce,
// while the raw editor's submitLine calls it directly — #14 extracted it exactly
// so the raw path could render cooked and play raw. Capturing in defineOnce
// would leave the interactive path, the only one capturing today, silent.
//
// lookupAndRender is the part of the define path that only WRITES — look up,
// render, print. Split out because the raw-mode loop must run it in cooked mode
// (so newlines translate) while playing in RAW mode (so Ctrl-C arrives as a byte
// the key reader can see). Returns whether audio should follow.
func lookupAndRender(d deps, opt options, cmd replCommand, stdout, stderr io.Writer) lookupOutcome {
	word := cmd.word
	text, err := d.dict.Lookup(word)
	if err != nil {
		// The route decision comes BEFORE capture, and that order is the point:
		// a question recorded as a not-found lookup lands in the event log that
		// #8's statistics and #17's learner model both fold over — data that is
		// not a lookup at all. The dictionary is asked once and its miss is the
		// free, offline signal the classifier runs on (#16 D1).
		if !cmd.literal && readsAsQuestion(word) {
			return lookupOutcome{ask: word}
		}
		fmt.Fprintf(stderr, "define: %s: %v\n", word, err)
		d.capture.Capture(word, false, opt)
		return lookupOutcome{code: 1}
	}
	if opt.raw {
		fmt.Fprintln(stdout, text)
		// Ask the policy even here. decideCapture answers "nothing" for -raw, and
		// it must be the thing that says so — returning early made that branch
		// unreachable and gave "capture is off" a second home.
		d.capture.Capture(word, true, opt)
		return lookupOutcome{entry: text}
	}
	fmt.Fprint(stdout, Render(ParseEntry(text), RenderOpts{Color: opt.color, Width: opt.width}))
	d.capture.Capture(word, true, opt)
	return lookupOutcome{play: !opt.noAudio && opt.times > 0, entry: text}
}

// defaultIndicator is the ephemeral form on a terminal, the record form on a pipe.
func defaultIndicator(opt options) indicator {
	ind := indicator{show: true, before: "\n", erase: eraseLine}
	if !opt.tty {
		ind.erase, ind.trail = "", "\n"
	}
	return ind
}

// indicator describes the ephemeral "♫ playing N×" line: what to write before it
// (cursor positioning), and what to write after playback to remove it.
type indicator struct {
	show   bool
	before string
	erase  string
	trail  string // written instead of erase when there is nothing to erase
}

// playAnnounced is the single owner of the announce → play → erase → report
// sequence. Both entry paths ran their own copy and had diverged three ways —
// which terminal they gated on, whether the audio-off guard applied, and the
// duplicated literal — so this exists to make the erase style the only
// difference between them (ARCH-DRY).
//
// Returns true when playback finished with nothing reported, so the caller can
// decide whether its redrawn UI is still intact.
func playAnnounced(ctx context.Context, d deps, opt options, word string, ind indicator, stdout, stderr io.Writer) bool {
	// An erasable indicator is ephemeral UI and may be optimistic — if playback
	// fails it is taken back and never seen. A non-erasable one (a pipe, or
	// -no-color) is a RECORD, and a record has to be true: announced only after
	// something actually played. Otherwise `define <word-with-no-recording> >
	// out.txt` files a claim that it played three times when it played none.
	erasable := ind.show && ind.erase != ""
	if erasable {
		fmt.Fprint(stdout, ind.before)
		fmt.Fprintf(stdout, "  ♫ playing %d×", opt.times)
	}
	err := speak(ctx, d, word, opt.locale, opt.times)
	if erasable {
		fmt.Fprint(stdout, ind.erase)
	}
	// A cancelled context is the user pressing Ctrl-C, not a failure. Without
	// this guard SIGINT during playback prints "define: afplay: signal: killed" —
	// killing afplay is how cancellation is *implemented*, so reporting it as an
	// error tells the user their own keypress went wrong.
	if err != nil {
		if ctx.Err() == nil {
			fmt.Fprintf(stderr, "define: %s\n", err)
		}
		return false
	}
	if ind.show && !erasable {
		fmt.Fprint(stdout, ind.before)
		fmt.Fprintf(stdout, "  ♫ playing %d×%s", opt.times, ind.trail)
	}
	return true
}

// speak fetches the recording and plays it n times. It prints NOTHING — the
// announcement belongs to the caller, because a replay in the loop must leave
// the screen exactly as it was.
func speak(ctx context.Context, d deps, word, locale string, n int) error {
	data, _, err := d.audio.Fetch(ctx, AudioCandidates(word, locale))
	if err != nil {
		return fmt.Errorf("%s: %w", word, err)
	}
	dir, err := os.MkdirTemp("", "define-audio-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "pronunciation.mp3")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return playN(ctx, d.player, path, n)
}

// isSet reports whether a flag was given at all, which is different from being
// given an empty value: `-forget=""` is a mistake, not a request to start a REPL.
func isSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// forgetWord removes a word from the deck.
//
// An absent word exits NON-ZERO: succeeding silently would hide a typo in the
// very command meant to correct one.
func forgetWord(d deps, opt options, word string, stdout, stderr io.Writer) int {
	if d.deck == nil {
		// Under DEFINE_NO_CAPTURE there may well BE a deck on disk — we simply
		// did not open one. Saying "no deck" would be a lie about their data.
		fmt.Fprintln(stderr, noDeckMessage(opt.noCapture))
		return 1
	}
	removed, err := d.deck.Forget(word)
	if err != nil {
		fmt.Fprintf(stderr, "define: %s: %v\n", word, err)
		return 1
	}
	if !removed {
		fmt.Fprintf(stderr, "define: %s is not in the deck\n", word)
		return 1
	}
	fmt.Fprintf(stdout, "removed %s\n", word)
	return 0
}

// noDeckMessage explains a nil deck. Shared by --forget and /history: the same
// fact stated in two places is how the atlas contradictions in #4 started.
func noDeckMessage(noCapture bool) string {
	if noCapture {
		return "define: DEFINE_NO_CAPTURE is set, so no deck was opened"
	}
	return "define: no deck in this directory"
}

// terminalWidth reports the usable width of stdout, or 0 when it is not a
// terminal. Wrapping is a presentation decision, so it stays at the boundary and
// Render receives a number.
func terminalWidth(w io.Writer) int {
	f, ok := w.(*os.File)
	if !ok {
		return 0
	}
	cols, _, err := term.GetSize(int(f.Fd()))
	if err != nil || cols < 20 { // an implausibly narrow terminal: do not wrap
		return 0
	}
	return cols
}

// isTerminal keeps the TTY probe out of Render, so rendering stays pure and
// piping `define x | less` yields clean text.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

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
	// loop takes a seam rather than deciding where state lives. NOT in langDeps:
	// events/ is not language-scoped.
	history History
	// langDeps is EMBEDDED, so d.deck, d.vocab and the rest read exactly as they
	// did — and so adopting a switch is one whole-struct assignment rather than a
	// list of fields. The struct alone was not enough: while both openStore and
	// applyLang copied its members by hand, a new member stayed forgettable at
	// two sites, which is the failure the struct was introduced to prevent.
	langDeps
	// dictName is which dictionary answered, for /lang to report. A learner on a
	// machine with a different installed set asks that question once, so the
	// answer belongs in a command rather than in a line per lookup.
	dictName string
	// newDict builds the dictionary for a language, and is the M2 member of
	// applyLang's enumeration — registered in that comment before it existed.
	// nil when a test supplied its own dict, exactly like newLangDeps.
	newDict func(store.Lang, io.Writer) (Dictionary, string)
	// lang is the language this process is operating in — the deck it reads and
	// writes, the recording it asks for, and the dictionary it consults. Resolved
	// ONCE at the boundary (openStore, which is the only thing that knows the
	// directory) and passed down; nothing below here re-reads the setting.
	lang store.Lang
	// newLangDeps rebuilds every language-derived dependency for another language.
	//
	// The one builder, called twice: openStore constructs the session with it,
	// and /lang switches with it. That is what makes the set structural rather
	// than a list in a comment — a comment did not survive one milestone.
	//
	// nil means there is no store here (a --llm-check run, an unopenable
	// directory, a test that supplied its own deps): /lang can still write the
	// setting, it just has no session to re-derive.
	newLangDeps func(store.Lang) langDeps
	// persistLang writes the directory's language setting. Separate from newLangDeps
	// because the two halves of /lang have different preconditions: persisting
	// needs a DIRECTORY, re-deriving needs a SESSION, and a one-shot
	// `define /lang es` has the first without the second. nil means there is no
	// directory to write to.
	persistLang func(store.Lang) error
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
	// getenv and newLLM are the model seam, split the way llmcheck already splits
	// it: configuration is resolved from the environment, then a client is built
	// from it. Injected together because a test that redirects one and not the
	// other builds a real client pointed at a real proxy.
	getenv func(string) string
	newLLM func(llm.Config) llm.Client
	// notifySignals is the SIGNAL half of the interrupt story — the other half is
	// the raw key reader's byte. Injected so a test can drive it without raising
	// a real signal in the test binary, which `go test` would treat as a failure.
	// nil means "no signal transport", which is what every test that does not
	// care about interrupts gets.
	notifySignals func(...os.Signal) <-chan os.Signal
	// stdinIsTerminal decides whether the loop prints a prompt. Injected rather
	// than probed directly because a test harness's stdin is never a terminal,
	// which would make the interactive path unwritable. Note this is a different
	// question from the stdout check that drives colour.
	stdinIsTerminal func() bool
}

func realDeps() deps {
	return deps{
		newDict:         systemDictionary, // dict itself is language-dependent, built in run()
		audio:           newHTTPAudioSource(),
		player:          afplayPlayer{},
		newStore:        openStore,
		stdinIsTerminal: func() bool { return isTerminal(os.Stdin) },
		notifySignals:   notifySignals,
		getenv:          os.Getenv,
		newLLM:          llm.New,
	}
}

// notifySignals is the production signal transport: a channel signal.Notify
// feeds. Buffered by one, per os/signal's contract — an unbuffered channel a
// receiver is not sitting on drops the signal.
func notifySignals(sigs ...os.Signal) <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sigs...)
	return ch
}

// storeDeps is the trio openStore produces. One value rather than three returns
// and three nil-merges at the call site: they are always built together, always
// consumed together, and the merge was three chances to forget one.
// langDeps is every dependency that is a FUNCTION OF THE LANGUAGE.
//
// A named type rather than four return values, so adding a member is a field —
// visible at both the construction site and the switch — instead of a positional
// change someone can absorb at one of the two.
type langDeps struct {
	deck    store.Store
	capture Capturer
	vocab   Vocabulary
	usage   UsageSource
	// dict is a member too, and has been since M1 registered it — but it is
	// built by a different seam (newDict, which tests replace independently of
	// the store), so applyTo takes it as an argument rather than a field.
}

type storeDeps struct {
	history History
	langDeps
	// lang is the language openStore RESOLVED — the flag if one was given, else
	// the directory's setting, else English. The flag half lives in options; this
	// is the answer, and it is what everything downstream reads.
	lang        store.Lang
	newLangDeps func(store.Lang) langDeps
	persistLang func(store.Lang) error
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
	// nil is the ONE representation of "nothing to highlight" — no second
	// empty-set stand-in. highlightSpans is where that nil is interpreted; the
	// only other guards are where a nil would panic (Load, Add).
	if d.vocab == nil {
		d.vocab = sd.vocab
	}
	if d.usage == nil {
		d.usage = sd.usage
	}
	if d.newLangDeps == nil {
		d.newLangDeps = sd.newLangDeps
	}
	if d.persistLang == nil {
		d.persistLang = sd.persistLang
	}
	// The flag is the fallback here, not the winner: openStore has already
	// applied the precedence when there was a directory to apply it against. This
	// branch is what a test with no newStore gets, where the flag is all there is.
	d.lang = orElse(d.lang, orElse(sd.lang, orElse(opt.lang, store.DefaultLang)))
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
// newsFeedFor gates the news feed on the language it actually serves.
//
// httpFeed hardcodes hl=en-US&gl=US&ceid=US:en — it is English BY CONSTRUCTION.
// Asking it about a Spanish word returns English news about a different sense
// entirely (mesa is a landform, and a city in Arizona), and caches it under a key
// an English session shares.
//
// So it is consulted only for the language it serves, and in any other language
// the examples come from that language's own dictionary entry — which M2 makes
// correct, and which is where "Levantarse muy temprano, especialmente al
// amanecer" comes from. Same rule as the dictionary: no data beats the wrong
// language's data.
//
// This is also why usage/ needs no language dimension — nothing writes it outside
// English. When #10 or #18 makes the feed language-aware, scoping the cache
// becomes REQUIRED, and that is the moment to add it (D6).
func newsFeedFor(lang store.Lang, f *cachingFeed) *cachingFeed {
	if lang != store.DefaultLang {
		return nil
	}
	return f
}

// sessionUsage is the no-durable-store form: the same seam, cached in memory.
//
// DEFINE_NO_CAPTURE means "write nothing into this directory", not "the feed does
// not exist" — the same reading that gives this path a memHistory rather than no
// history at all. Nothing reaches disk, and a session still does not hit the
// network per question.
func sessionUsage(lang store.Lang, clk store.Clock, warn io.Writer) UsageSource {
	return &bothSources{news: newsFeedFor(lang, newCachingFeed(newHTTPFeed(), store.NewMem(), clk)), warn: warn}
}

func openStore(opt options, warn io.Writer) storeDeps {
	// NOT a second copy of the capture policy: this decides whether there is
	// anywhere to write at all. decideCapture stays the only thing that decides
	// whether a given lookup counts.
	clk := store.SystemClock()
	// With no directory there is no persisted setting to consult, so the flag is
	// the whole of the precedence. newLangDeps stays nil on both of these paths: /lang
	// can still validate and report, it just has nothing to re-derive.
	if opt.noCapture {
		lang := orElse(opt.lang, store.DefaultLang)
		return storeDeps{
			history:  &memHistory{},
			langDeps: langDeps{capture: noopCapturer{}, usage: sessionUsage(lang, clk, warn)},
			clock:    clk,
			lang:     lang,
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(warn, "define: no working directory (%v); history is session-only\n", err)
		lang := orElse(opt.lang, store.DefaultLang)
		return storeDeps{
			history:  &memHistory{},
			langDeps: langDeps{capture: noopCapturer{}, usage: sessionUsage(lang, clk, warn)},
			clock:    clk,
			lang:     lang,
		}
	}
	// Before anything reads the deck: a deck written before #23 lives flat in
	// words/ and Deck() now reads words/<lang>/, so without this it is orphaned.
	// Idempotent and language-blind by design — see MigrateToLanguages.
	if err := store.MigrateToLanguages(dir, warn); err != nil {
		fmt.Fprintf(warn, "define: could not migrate the existing deck (%v); it stays where it is\n", err)
	}
	// #23's precedence, applied ONCE and only here, because this is the only
	// thing that knows the directory: the flag wins for THIS invocation and does
	// not persist; otherwise the directory's setting; otherwise English.
	lang := opt.lang
	if lang == "" {
		lang = store.ReadLang(dir)
	}

	// EVERY dependency that is a function of the language, built in ONE place.
	//
	// The set used to be enumerated in a doc comment on applyLang, and that did
	// not survive a single milestone: M2 made the news feed's presence
	// language-dependent (D6) and the comment still said usage was "not
	// language-scoped". A member added at the boundary and forgotten at the
	// switch is silent — the session simply keeps the old language's copy.
	//
	// Now it is structural: /lang calls THIS function, so anything constructed
	// here is necessarily re-derived there. Adding a member cannot be half done.
	// A store whose language is irrelevant to it: history reads events/ and the
	// usage cache reads usage/, neither of which is language-scoped.
	flat := store.NewYAML(dir, store.DefaultLang, warn)

	newLangDeps := func(l store.Lang) langDeps {
		st := store.NewYAML(dir, l, warn)
		// ONE highlight set, handed to both the capturer that grows it and the
		// renderers that read it. Two instances would mean lookups landing in a
		// set nothing draws from — TestOpenStoreSharesOneHighlightSet is the
		// pin, and TestLangSwitchKeepsOneHighlightSet is the same pin after a
		// switch.
		voc := newStoreVocabulary(st, warn)
		return langDeps{
			deck:    st,
			capture: newStoreCapturer(st, clk, warn, voc),
			vocab:   voc,
			// The news feed is English by construction, so it is gated rather
			// than scoped — see newsFeedFor. It is a member of THIS set because
			// its presence depends on the language, even though usage/ does not.
			usage: &bothSources{news: newsFeedFor(l, newCachingFeed(newHTTPFeed(), flat, clk)), warn: warn},
		}
	}
	ld := newLangDeps(lang)

	sd := storeDeps{
		history:     newStoreHistory(flat, warn),
		clock:       clk,
		lang:        lang,
		newLangDeps: newLangDeps,
		persistLang: func(l store.Lang) error { return store.WriteLang(dir, l) },
	}
	sd.langDeps = ld
	return sd
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
	// count bounds a review session (#6). A flag rather than a constant because
	// twenty is a guess about one learner's attention span — exactly the kind of
	// guess that should be changeable without a rebuild. 0 means "no budget" and
	// returns nothing, matching schedule.Queue's contract rather than inventing a
	// second meaning for it.
	count int
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
	// lang is the -lang FLAG, empty when it was not given — not the language in
	// effect, which is deps.lang. The distinction matters because the flag is
	// one-third of a precedence openStore applies (flag, then the directory's
	// setting, then English), and only openStore knows the directory.
	lang store.Lang
	// voice is the recording to ask for: the language in effect plus its regional
	// variant. DERIVED FROM THE LANGUAGE, so /lang has to re-derive it — see
	// applyLang, which owns the whole enumeration. Built once at the boundary
	// rather than per play because it is a session-level fact, not a per-lookup
	// one. (It also used to carry a -locale complaint that would otherwise
	// re-print on every replay; #27 removed the refusal that produced it.)
	voice voice
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
	locale := fs.String("locale", "", localeHelp)
	// -lang exists so a script can ask a question without mutating state: it
	// applies to THIS invocation and does not persist. /lang is the other half —
	// it persists and does not need re-typing.
	langFlag := fs.String("lang", "", langHelp)
	// -pron is NOT -lang's sibling despite the shape. -lang moves the mode: the
	// deck, the dictionary, the highlight set and the recording. -pron moves only
	// the recording, for one lookup, which is the whole of #29.
	pronFlag := fs.String("pron", "", pronHelp)
	forget := fs.String("forget", "", "remove a word from the deck (events are kept)")
	llmCheck := fs.Bool("llm-check", false, "check the model configuration and exit")
	// Names the artifact, not the file: the filename is per-language and this
	// help text is printed before any language is resolved.
	reflect := fs.Bool("reflect", false, "read the deck and write the learner model")
	playFlag := fs.Bool("play", false, "review the words due today")
	count := fs.Int("count", 20, "how many words a review session offers")
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
			"pronunciation plays for the rest of the session; /lang says which\n"+
			"language this deck is in, and /lang es switches it; /pron fr\n"+
			"replays the word you just looked up in French, once.\n\n"+
			"define works in ONE language at a time. The setting lives in the\n"+
			"directory, so a one-shot lookup inherits it with no session to ask;\n"+
			"-lang es applies to one run without changing it. Each language has\n"+
			"its own deck under words/<lang>/, so a review session never mixes\n"+
			"them.\n\n"+
			"-pron is the exception, and it is not a mode: -pron fr arrondissement\n"+
			"plays the French recording and changes nothing else — same entry,\n"+
			"same deck, and the next lookup is English again. You name the\n"+
			"language; the entry's ORIGIN says which. A source with no recording\n"+
			"falls back to the session's and says so.\n\n"+
			"A line that is not a word and reads as a question is answered by\n"+
			"the model rather than looked up — there is no mode to switch. The\n"+
			"dictionary is asked first, so multi-word headwords (hot dog) are\n"+
			"still definitions. Force either way: ? asks, \\ defines.\n\n"+
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
	// -count is validated HERE, beside its siblings, because it is the same
	// class: a negative budget is a typo, not an instruction. It used to be
	// accepted silently, and schedule.Queue then returned nil for it, which
	// --play reported as "nothing due today" — blaming the schedule for what the
	// flag did (BR-46).
	if *count < 0 {
		fmt.Fprintln(stderr, "define: -count must not be negative")
		return 2
	}
	if *times > maxSoundTimes {
		fmt.Fprintf(stderr, "define: %s %d would take a while to sit through; the limit is %d\n",
			flagName, *times, maxSoundTimes)
		return 2
	}
	// Validated HERE, before anything opens a directory: this value becomes a
	// path segment (words/<lang>/), and a usage error must not be the thing that
	// creates a directory — the same rule the --forget checks below follow.
	var lang store.Lang
	if *langFlag != "" {
		parsed, err := store.ParseLang(*langFlag)
		if err != nil {
			fmt.Fprintf(stderr, "define: %v\n", err)
			return 2
		}
		lang = parsed
	}
	// Validated HERE with -lang, before anything opens a directory, and for the
	// same reason: this value becomes a path segment on the CDN. store.ParseLang
	// is the validator, so the complaint is its one message rather than a second
	// spelling of "that is not a language".
	var pron store.Lang
	if *pronFlag != "" {
		parsed, err := store.ParseLang(*pronFlag)
		if err != nil {
			fmt.Fprintf(stderr, "define: %v\n", err)
			return 2
		}
		pron = parsed
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
		count:     *count,
		lang:      lang,
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
	case *reflect && fs.NArg() != 0:
		// A mode plus a word is two commands on one line, and silently honouring
		// one of them is how -raw came to mean two things in #2.
		fmt.Fprintln(stderr, "define: --reflect reads the deck; do not also pass a word")
		return 2
	case *playFlag && fs.NArg() != 0:
		// Same rule, and --play needed it MORE than --reflect does: it writes
		// events, so `define --play sycophantic` would change state under a
		// misread intent. The first version dispatched above this switch and so
		// could never reach the guard — the comment three lines up states the
		// rule it was breaking.
		fmt.Fprintln(stderr, "define: --play reviews the deck; do not also pass a word")
		return 2
	case !forgetting && oneShot.kind != cmdCommand && oneShot.kind != cmdAsk && fs.NArg() > 1:
		fs.Usage()
		return 2
	// -pron applies to ONE lookup, so it needs one. Refused rather than quietly
	// made session-wide: a pronunciation language living in opt.voice would
	// survive a /lang switch and ask for fr_fr recordings in a Spanish session,
	// which is the drift applyLang's enumeration exists to stop (#29 D3). The
	// message names /pron rather than leaving the in-session form to be found.
	case pron != "" && oneShot.kind != cmdDefine:
		fmt.Fprintln(stderr, "define: -pron applies to one lookup; at the prompt use /pron fr")
		return 2
	}
	// The flag rides on the LINE, beside `literal`, because that is what it is:
	// a per-line modifier. parseREPLLine never sets it, so nothing either loop
	// parses carries a language.
	oneShot.pron = pron

	// Store-backed dependencies are built HERE, not in realDeps: the opt-out is a
	// flag-parse-time input and decides whether anything is opened at all.
	//
	// Nothing is exempted from this. An earlier fix skipped it for commands that
	// read nothing, which dropped an invariant it was not thinking about —
	// deps.clock is supplied here, so the exemption stranded it as nil. The cost
	// it was avoiding is gone at the source instead: opening a store no longer
	// reads the log (History.Load does, when a loop is about to recall).
	d = d.withStore(opt, stderr)
	// The language is known only after the store has resolved it, so the voice is
	// derived here rather than at flag parse. Through applyVoice, the same
	// function /lang re-derives it with — one derivation, two callers, which is
	// what stops the two from disagreeing after a switch.
	applyVoice(&opt, d.lang)
	// The dictionary follows the mode too (#23 M2). Only when a test did not
	// supply one — same nil-merge rule withStore uses for the store-backed deps.
	if d.dict == nil && d.newDict != nil {
		d.dict, d.dictName = d.newDict(d.lang, stderr)
	}

	if forgetting {
		return forgetWord(d, opt, *forget, stdout, stderr)
	}
	// Dispatched HERE and not beside --llm-check, which runs before withStore
	// precisely because it needs no directory. --reflect needs both the deck and
	// the clock, so it belongs after them, where --forget is (#17 D5).
	if *playFlag {
		// No second withStore: it is called unconditionally nine lines above.
		// Calling it twice is harmless only because it fills nils — which is
		// exactly the kind of "harmless" that stops being true when someone adds
		// a field that is not nil-merged.
		return runPlay(ctx, d, opt, stdin, stdout, stderr)
	}
	if *reflect {
		return runReflect(ctx, d, opt, stdout, stderr)
	}

	switch fs.NArg() {
	case 0:
		// The loop derives its OWN context and owns what an interrupt means —
		// see repl, where the detach sits above the choice of loop so both are
		// served. This branch used to derive the cancel itself, which put the
		// decision one level above the thing that makes it (#16 D5).
		//
		// NotifyContext stays for the one-shot, -forget and --llm-check paths,
		// where "SIGINT ends the program" is the right contract.
		return repl(ctx, d, opt, stdin, stdout, stderr)
	default:
		// A command is a command from every entry mode, arguments and all.
		if oneShot.kind == cmdCommand {
			return dispatchCommand(oneShot, commands, newCommandCtx(d, opt, stdout, stderr))
		}
		// So is a question. The forced route ("?…") skips the dictionary here
		// exactly as it does at the prompt.
		if oneShot.kind == cmdAsk {
			// A one-shot session holds nothing but this question: the DIRECTORY
			// is the context, which is what makes a fresh process answer as well
			// as a long-running one.
			return ask(ctx, d, opt, &session{}, stdout, stderr, question{text: oneShot.question, forced: true})
		}
		// EXHAUSTIVE over what parseREPLLine can return, not "handle the two I
		// added and let the rest fall through". #16 gave the parser a kind this
		// branch had never seen — cmdNothing, from a bare "?" or "\" — and
		// falling through handed lookupAndRender an EMPTY word: `define "?"`
		// printed `define: : no dictionary entry` and appended a ReviewEvent with
		// no word, which the log then discards at read time as indistinguishable
		// from a torn record (BR-4).
		if oneShot.kind != cmdDefine {
			// inSession is false: a one-shot has no loop to press return in.
			fmt.Fprintf(stderr, "define: %s\n", nothingSays(oneShot, false))
			return 2
		}
		// oneShot, not fs.Arg(0): the parsed line is what carries #16's hatches,
		// and a one-shot that re-derived the word from argv would send `define
		// "?what is X"` to the dictionary — BR-13's shape, in a new place.
		out := defineOnce(ctx, d, opt, oneShot, stdout, stderr)
		if out.ask != "" {
			// The unforced route: the dictionary missed and the line reads as a
			// question. Returning out.code here would exit 0 having printed
			// nothing, since an ask outcome carries no failure.
			// NO current word: the dictionary missed, so nothing was defined and
			// the line IS the question. Passing oneShot.word here put the
			// question text into session.current — and from there into "## The
			// word on screen" and the event log's word field, which is the exact
			// pollution routing-before-capture exists to prevent (BR-23).
			return ask(ctx, d, opt, &session{}, stdout, stderr, question{text: out.ask})
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
		playAnnounced(ctx, d, opt, utteranceFor(cmd.word, out.entry, cmd.pron, opt),
			defaultIndicator(opt), stdout, stderr)
	}
	return out
}

// lookupOutcome is what one line turned out to be, once the dictionary has
// answered. It carries a third possibility the define path did not used to have:
// the line was a question. That has to travel as DATA rather than be acted on
// here, because a definition and a streamed answer are different jobs with
// different cancellation — one function cannot do both (#16 D6). The reason used
// to be sharper still: they ran in different terminal modes, until #30 D4 left
// only one mode.
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
// render, print. The split outlives its original reason: #14 needed the render
// to happen in cooked mode and the playback in raw, and #30 D4 removed the modes
// altogether. It stays because the two halves still differ in kind — this one is
// pure output, and the caller owns the playback that can be interrupted.
// Returns whether audio should follow.
func lookupAndRender(d deps, opt options, cmd replCommand, stdout, stderr io.Writer) lookupOutcome {
	word := cmd.word
	text, err := d.dict.Lookup(word)
	if err != nil {
		// The route decision comes BEFORE capture, and that order is the point:
		// a question recorded as a not-found lookup lands in the event log that
		// #8's statistics and #17's learner model both fold over — data that is
		// not a lookup at all. The dictionary is asked once and its miss is the
		// free, offline signal the classifier runs on (#16 D1).
		// Two conditions, one question: may this miss fall back to a question?
		// cmd.literal is the user's per-line answer ("\\"), mayAsk is the
		// session's (-raw, the scripting form). mayAsk lives in ask.go because
		// the forced route needs the same predicate and guarding only this one
		// left three of six cells open (BR-9).
		//
		// NOT redundant with ask()'s own mayAsk check, however much it looks it:
		// this one decides the OBSERVABLE. Without it a -raw miss reaches ask(),
		// which refuses with advice to "drop the ?" for a line that contains no
		// "?" — instead of the `no dictionary entry` the scripting contract
		// promises. Deleting either changes what a script sees (BR-14).
		if !cmd.literal && mayAsk(opt) && readsAsQuestion(word) {
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
	// Word is the LOOKUP KEY, and passing it is what makes a click a shortcut
	// rather than a second opinion: it is exactly what `sess.current` becomes,
	// so a click on the headword and the bare Enter beside it ask for the same
	// recording by construction.
	rendered, regions := Render(ParseEntry(text), RenderOpts{
		Color: opt.color, Width: opt.width, Vocab: vocabularyFor(d, opt), Word: word,
	})
	writeRendered(stdout, rendered, regions)
	d.capture.Capture(word, true, opt)
	return lookupOutcome{play: !opt.noAudio && opt.times > 0, entry: text}
}

// regionWriter is a writer that can also hold a CLICK MAP for what it is given.
// The interactive screen is the only one; everything else takes bytes.
type regionWriter interface {
	io.Writer
	WriteRegions(text string, rs []Region)
}

// writeRendered gives an entry to a writer, with its regions if the writer has
// somewhere to put them.
//
// The seam fills itself rather than the caller branching: a one-shot, a pipe or
// `> out.txt` has nowhere to click and gets exactly the bytes it always did
// (#30 D6), while the interactive screen gets the map. One call either way, so
// the text and the regions cannot be written at different moments and disagree
// about which line they landed on.
func writeRendered(w io.Writer, text string, rs []Region) {
	if rw, ok := w.(regionWriter); ok {
		rw.WriteRegions(text, rs)
		return
	}
	fmt.Fprint(w, text)
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
func playAnnounced(ctx context.Context, d deps, opt options, u utterance, ind indicator, stdout, stderr io.Writer) bool {
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
	from, err := speak(ctx, d, u, opt.times)
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
	// AFTER the erase and only on success, because this is a record rather than
	// the ephemeral indicator above it: on a pipe it cannot be taken back, so it
	// is written once the answer is known instead of predicted from the request.
	reportVoice(stderr, u, from)
	if ind.show && !erasable {
		fmt.Fprint(stdout, ind.before)
		fmt.Fprintf(stdout, "  ♫ playing %d×%s", opt.times, ind.trail)
	}
	return true
}

// reportVoice says what actually played, when it was not what was asked for.
//
// ONLY in that case (#29). A line on every lookup would be noise, and a line
// when no source was asked for would answer a question nobody put. The Italian
// case is why it exists at all: nine probes found no Italian audio in this CDN
// generation, so `/pron it ciao` will always fall back — and falling back
// silently would present the English recording as the Italian one.
//
// It reads `from`, the URL that ANSWERED, through utterance.spokeSource, which
// tests membership in the list actually built. Predicting from the request
// instead would report a fallback that did not happen the moment coverage
// changes.
func reportVoice(w io.Writer, u utterance, from string) {
	if !u.askedForSource() || u.spokeSource(from) {
		return
	}
	// NORMALISED, not the raw fields: a zero session voice would otherwise print
	// "played the  one". AudioCandidates defaults the same way, so the record
	// names the language actually asked for.
	fmt.Fprintf(w, "define: no %s recording for %s; played the %s one\n",
		u.Source.langOrDefault(), u.Word, u.Session.langOrDefault())
}

// utteranceFor builds one request: the session's voice always, and a source
// voice only when a language was asked for.
//
// ONE builder for every call site — the one-shot, both loops' replay, the raw
// loop's post-lookup play and the review session — because the source spellings
// have to be derived identically at each. Two paths that each decide what to ask
// the CDN is #14's "two loops, one decision table" in a new costume, and that
// divergence is silent because each path is individually tested.
//
// `entry` may be empty: a caller with no raw text still gets a working
// session-voice request, and SourceSpellings falls back to the typed word.
//
// The source voice comes from voiceFor — #27's function, UNCHANGED — so the
// locale is #27's policy rather than a second one invented here, and -locale
// still qualifies whatever language is in effect: `-pron es` builds es_es,
// `-pron es -locale us` builds es_us, Latin American seseo.
func utteranceFor(word, entry string, pron store.Lang, opt options) utterance {
	u := utterance{Word: word, Session: opt.voice}
	if pron == "" {
		return u
	}
	u.Source = voiceFor(pron, opt.locale)
	u.Spellings = SourceSpellings(word, ParseEntry(entry))
	return u
}

// speak fetches the recording and plays it n times. It prints NOTHING — the
// announcement belongs to the caller, because a replay in the loop must leave
// the screen exactly as it was.
//
// Returns the URL that ANSWERED, which it used to fetch and discard. That is
// what lets the caller report which voice was heard from what actually happened
// rather than from what was requested (#29).
func speak(ctx context.Context, d deps, u utterance, n int) (from string, err error) {
	data, from, err := d.audio.Fetch(ctx, u.Candidates())
	if err != nil {
		return "", fmt.Errorf("%s: %w", u.Word, err)
	}
	dir, err := os.MkdirTemp("", "define-audio-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "pronunciation.mp3")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return from, playN(ctx, d.player, path, n)
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
	sz, ok := terminalSize(w)
	if !ok || sz.cols < minWrapWidth { // not a terminal, or too narrow to break: do not wrap
		return 0
	}
	return sz.cols
}

// terminalSize is the terminal's true SHAPE, and the distinction from
// terminalWidth is the point: this one cannot return a sentinel.
//
// terminalWidth answers a POLICY question — "how wide should text be wrapped",
// where 0 means "do not wrap" and a 15-column terminal gets that answer. The
// screen asks a different question — "how many columns does this terminal have"
// — and an answer of 0 there means a frame budgeted with no width at all, which
// is the sentinel leaking into arithmetic that has no use for it.
//
// ok is false when w is not a terminal. The caller decides what to do about it;
// the full-screen loop runs only when it is one.
func terminalSize(w io.Writer) (winSize, bool) {
	f, ok := w.(*os.File)
	if !ok {
		return winSize{}, false
	}
	cols, rows, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return winSize{}, false
	}
	return winSize{rows: rows, cols: cols}, true
}

// terminalRows reports the height of w, or a conventional 24 when it cannot be
// measured.
//
// A height is only needed by the full-screen loop (#30), which runs solely when
// stdout is a terminal — so the fallback covers a probe that fails rather than a
// pipe. It has to be a plausible number rather than 0: a screen told it has no
// rows shows the prompt and nothing else, which would hide the very definition
// the user asked for.
func terminalRows(w io.Writer) int {
	sz, ok := terminalSize(w)
	if !ok || sz.rows < 2 { // one row cannot hold both a definition and a prompt
		return defaultRows
	}
	return sz.rows
}

// terminalCols is the width the SCREEN paints to — never 0, because a frame
// budgeted with no width is a frame with no budget.
func terminalCols(w io.Writer) int {
	sz, ok := terminalSize(w)
	if !ok || sz.cols < 1 {
		return defaultCols
	}
	return sz.cols
}

const (
	defaultRows = 24
	defaultCols = 80
)

// isTerminal keeps the TTY probe out of Render, so rendering stays pure and
// piping `define x | less` yields clean text.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

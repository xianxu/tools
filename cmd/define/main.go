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

	"golang.org/x/term"
)

// deps is the injected IO surface. Keeping it explicit is what lets run() be
// driven end-to-end by fakes (ARCH-PURE): main() supplies the real ones, tests
// supply recorders.
type deps struct {
	dict   Dictionary
	audio  AudioSource
	player Player
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
		stdinIsTerminal: func() bool { return isTerminal(os.Stdin) },
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
	times := fs.Int("times", 3, "how many times to play the pronunciation")
	locale := fs.String("locale", "us", "pronunciation locale: us or gb")
	fs.Usage = func() {
		fmt.Fprint(stderr, "usage: define [flags] [word]\n\n"+
			"Looks the word up in macOS's active dictionaries — normally the New\n"+
			"Oxford American Dictionary, the one Google licenses, hence the matching\n"+
			"notation — and plays its recorded pronunciation.\n\n"+
			"With no word, reads words from stdin; on a terminal that is an\n"+
			"interactive loop — return replays the pronunciation, Ctrl-C quits.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) { // -h is a successful request for help
			return 0
		}
		return 2
	}
	if *times < 0 {
		fmt.Fprintf(stderr, "define: -times must not be negative\n")
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
		noAudio: *noAudio || *raw,
		times:   *times,
		locale:  *locale,
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
	case 1:
		return defineOnce(ctx, d, opt, fs.Arg(0), stdout, stderr)
	default:
		fs.Usage()
		return 2
	}
}

// defineOnce is the whole define path for a single word: look up, render, print,
// speak. Extracted so the loop calls exactly this rather than growing a parallel
// copy (ARCH-DRY).
func defineOnce(ctx context.Context, d deps, opt options, word string, stdout, stderr io.Writer) int {
	code, play := lookupAndRender(d, opt, word, stdout, stderr)
	if play {
		// A missing recording is not a failed lookup: the definition is the
		// deliverable and has already been printed, so audio problems warn on
		// stderr and leave the exit code at 0.
		playAnnounced(ctx, d, opt, word, defaultIndicator(opt), stdout, stderr)
	}
	return code
}

// lookupAndRender is the part of the define path that only WRITES — look up,
// render, print. Split out because the raw-mode loop must run it in cooked mode
// (so newlines translate) while playing in RAW mode (so Ctrl-C arrives as a byte
// the key reader can see). Returns whether audio should follow.
func lookupAndRender(d deps, opt options, word string, stdout, stderr io.Writer) (code int, play bool) {
	text, err := d.dict.Lookup(word)
	if err != nil {
		fmt.Fprintf(stderr, "define: %s: %v\n", word, err)
		return 1, false
	}
	if opt.raw {
		fmt.Fprintln(stdout, text)
		return 0, false
	}
	fmt.Fprint(stdout, Render(ParseEntry(text), RenderOpts{Color: opt.color, Width: opt.width}))
	return 0, !opt.noAudio && opt.times > 0
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

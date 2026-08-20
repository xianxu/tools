package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
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
}

func realDeps() deps {
	return deps{
		dict:   systemDictionary(),
		audio:  newHTTPAudioSource(),
		player: afplayPlayer{},
	}
}

func main() {
	os.Exit(run(os.Args[1:], realDeps(), os.Stdout, os.Stderr))
}

// run is the thin IO shell: parse flags, look up, render, print, play. All of
// the reasoning lives in ParseEntry, Render and AudioCandidates, none of which
// see an io.Writer or a socket.
func run(args []string, d deps, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("define", flag.ContinueOnError)
	fs.SetOutput(stderr)
	raw := fs.Bool("raw", false, "print the unparsed dictionary entry")
	noColor := fs.Bool("no-color", false, "disable ANSI colour")
	noAudio := fs.Bool("no-audio", false, "do not fetch or play the pronunciation")
	times := fs.Int("times", 3, "how many times to play the pronunciation")
	locale := fs.String("locale", "us", "pronunciation locale: us or gb")
	fs.Usage = func() {
		fmt.Fprint(stderr, "usage: define [flags] <word>\n\n"+
			"Looks the word up in macOS's active dictionaries — normally the New\n"+
			"Oxford American Dictionary, the one Google licenses, hence the matching\n"+
			"notation — and plays its recorded pronunciation.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) { // -h is a successful request for help
			return 0
		}
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	word := fs.Arg(0)
	if *times < 0 {
		fmt.Fprintf(stderr, "define: -times must not be negative\n")
		return 2
	}

	text, err := d.dict.Lookup(word)
	if err != nil {
		fmt.Fprintf(stderr, "define: %s: %v\n", word, err)
		return 1
	}
	if *raw {
		fmt.Fprintln(stdout, text)
		return 0
	}

	color := !*noColor && isTerminal(stdout)
	fmt.Fprint(stdout, Render(ParseEntry(text), RenderOpts{Color: color}))

	if !*noAudio && *times > 0 {
		// A missing recording is not a failed lookup: the definition is the
		// deliverable and has already been printed, so audio problems warn on
		// stderr and leave the exit code at 0.
		if err := speak(context.Background(), d, word, *locale, *times, stdout); err != nil {
			fmt.Fprintf(stderr, "define: %s\n", err)
		}
	}
	return 0
}

// speak fetches the recording and plays it n times.
func speak(ctx context.Context, d deps, word, locale string, n int, stdout io.Writer) error {
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
	fmt.Fprintf(stdout, "\n  ♫ playing %d×\n", n)
	return playN(ctx, d.player, path, n)
}

// isTerminal keeps the TTY probe out of Render, so rendering stays pure and
// piping `define x | less` yields clean text.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// deps is the injected IO surface. Keeping it explicit is what lets run() be
// driven end-to-end by fakes (ARCH-PURE): main() supplies the real ones, tests
// supply recorders.
type deps struct {
	dict Dictionary
}

func realDeps() deps { return deps{dict: systemDictionary()} }

func main() {
	os.Exit(run(os.Args[1:], realDeps(), os.Stdout, os.Stderr))
}

// run is the thin IO shell: parse flags, look up, render, print. All of the
// reasoning lives in ParseEntry and Render, which never see an io.Writer.
func run(args []string, d deps, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("define", flag.ContinueOnError)
	fs.SetOutput(stderr)
	raw := fs.Bool("raw", false, "print the unparsed dictionary entry")
	noColor := fs.Bool("no-color", false, "disable ANSI colour")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: define [flags] <word>\n\nLooks the word up in the New Oxford American Dictionary\nbundled with macOS — the same dictionary Google licenses.\n\nFlags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	word := fs.Arg(0)

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
	return 0
}

// isTerminal keeps the TTY probe out of Render, so rendering stays pure and
// piping `define x | less` yields clean text.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

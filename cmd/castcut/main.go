// Command castcut records, annotates and cuts captioned terminal demos.
// `castcut --help` is the manual.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

// version is stamped by the release build (-ldflags "-X main.version=vX.Y.Z").
var version = "built from source"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// errUsage reports a usage mistake whose message the flag set already printed.
var errUsage = errors.New("usage")

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, helpText)
		return 0
	}
	var err error
	switch args[0] {
	case "help", "-h", "-help", "--help":
		fmt.Fprint(stdout, helpText)
		return 0
	case "version", "-version", "--version":
		fmt.Fprintf(stdout, "castcut %s\n", version)
		return 0
	case "cut":
		err = runCut(args[1:], stdout, stderr)
	default:
		err = fmt.Errorf("unknown command %q (castcut --help lists them)", args[0])
	}
	switch {
	case err == nil:
		return 0
	case errors.Is(err, flag.ErrHelp):
		return 0
	case errors.Is(err, errUsage):
		return 2
	default:
		fmt.Fprintf(stderr, "castcut: %v\n", err)
		return 1
	}
}

// parseInterspersed lets flags follow positional arguments, so
// `castcut cut take.cast -o out.cast` works as typed.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, err
			}
			return nil, errUsage
		}
		args = fs.Args()
		if len(args) == 0 {
			return pos, nil
		}
		if args[0] == "--" {
			return append(pos, args[1:]...), nil
		}
		pos, args = append(pos, args[0]), args[1:]
	}
}

func runCut(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("castcut cut", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, "usage: castcut cut <take.cast> [captions.txt] -o <cut.cast> [flags]\n\n"+
			"captions default to <take>.captions.txt, the file castcut annotate saves.\n\n")
		fs.PrintDefaults()
	}
	var out string
	t := defaultTiming
	fs.StringVar(&out, "o", "", "output cast (required)")
	fs.StringVar(&out, "out", "", "same as -o")
	fs.Float64Var(&t.Speed, "speed", t.Speed, "playback speed between captions")
	fs.Float64Var(&t.Idle, "idle", t.Idle, "max idle seconds per gap before speeding up")
	fs.Float64Var(&t.Lead, "lead", t.Lead, "start real time this long before a stamp")
	fs.Float64Var(&t.MinHold, "min-hold", t.MinHold, "minimum real-time seconds per caption")
	fs.Float64Var(&t.WPS, "wps", t.WPS, "caption reading speed, words per second")
	fs.Float64Var(&t.Beat, "beat", t.Beat, "extra seconds added to reading time")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	if len(pos) < 1 || len(pos) > 2 || out == "" {
		fs.Usage()
		return errUsage
	}
	castPath, capsPath := pos[0], sidecarPath(pos[0])
	if len(pos) == 2 {
		capsPath = pos[1]
	}

	data, err := os.ReadFile(castPath)
	if err != nil {
		return err
	}
	c, err := parseCast(castPath, data)
	if err != nil {
		return err
	}
	if data, err = os.ReadFile(capsPath); err != nil {
		if len(pos) == 1 && errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("no captions at %s: stamp them with `castcut annotate %s`, or name a captions file", capsPath, castPath)
		}
		return err
	}
	caps, err := parseCaptions(capsPath, data)
	if err != nil {
		return err
	}
	cut, sum, err := Cut(c, caps, t)
	if err != nil {
		return err
	}
	enc, err := encodeCast(cut)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, enc, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%s: %.1fs (view %.1fs), %d captions\n", out, sum.Total, sum.View, len(sum.Captions))
	for _, w := range sum.Captions {
		fmt.Fprintf(stdout, "  %6.1f-%6.1f  %s\n", w.Start, w.End, w.Text)
	}
	return nil
}

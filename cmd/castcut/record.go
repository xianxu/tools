package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// recordOpts are the knobs castcut passes through to asciinema.
type recordOpts struct {
	CaptureInput bool
	IdleLimit    float64 // 0 = none
	Headless     bool    // no terminal: CI and the live conformance check
}

// recordArgs is the asciinema argv for one take. --return makes asciinema exit
// with the recorded command's status; without it asciinema 3 always exits 0.
// There is no size: the take is the terminal's own, so the operator sizes the
// window before recording.
func recordArgs(out string, o recordOpts, cmd []string) []string {
	args := []string{"rec", "--return", "--output-format", "asciicast-v3"}
	if o.CaptureInput {
		args = append(args, "--capture-input")
	}
	if o.IdleLimit > 0 {
		args = append(args, "--idle-time-limit", strconv.FormatFloat(o.IdleLimit, 'f', -1, 64))
	}
	if o.Headless {
		args = append(args, "--headless")
	}
	return append(args, "--command", shellJoin(cmd), out)
}

// shellJoin quotes each word for the shell asciinema runs --command through.
func shellJoin(words []string) string {
	q := make([]string, len(words))
	for i, w := range words {
		if w != "" && strings.Trim(w, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_./=:@%+,") == "" {
			q[i] = w
		} else {
			q[i] = "'" + strings.ReplaceAll(w, "'", `'\''`) + "'"
		}
	}
	return strings.Join(q, " ")
}

var takeRe = regexp.MustCompile(`^take-(\d+)\.cast$`)

// nextTake is the first unused take-NN.cast in dir, after the highest one
// there; NN is at least two digits and keeps counting past 99.
func nextTake(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	high := 0
	for _, e := range entries {
		if m := takeRe.FindStringSubmatch(e.Name()); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil && n > high {
				high = n
			}
		}
	}
	return filepath.Join(dir, fmt.Sprintf("take-%02d.cast", high+1)), nil
}

// exitCode carries the recorded command's status out through run.
type exitCode int

func (e exitCode) Error() string { return fmt.Sprintf("recorded command exited %d", int(e)) }

func runRecord(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("castcut record", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, "usage: castcut record [-o take.cast] [flags] -- <command> [args...]\n\n"+
			"without -o the take goes to recordings/take-NN.cast (next free number).\n\n")
		fs.PrintDefaults()
	}
	o := recordOpts{}
	var out string
	fs.StringVar(&out, "o", "", "output cast (default recordings/take-NN.cast)")
	fs.StringVar(&out, "out", "", "same as -o")
	fs.BoolVar(&o.CaptureInput, "capture-input", false, "also record keystrokes as i events (passwords too)")
	fs.Float64Var(&o.IdleLimit, "idle-time-limit", 0, "cap idle gaps in the take itself (seconds; 0 keeps them; cut squeezes idle anyway)")
	fs.BoolVar(&o.Headless, "headless", false, "record without a terminal (CI, tests)")
	// Everything after `--` is the command, untouched by flag parsing.
	var cmd []string
	for i, a := range args {
		if a == "--" {
			args, cmd = args[:i], args[i+1:]
			break
		}
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return errUsage
	}
	if fs.NArg() > 0 || len(cmd) == 0 || !inRange(o.IdleLimit) {
		fs.Usage()
		return errUsage
	}
	if out == "" {
		next, err := nextTake("recordings")
		if err != nil {
			return err
		}
		out = next
	}
	if _, err := os.Stat(out); err == nil {
		return fmt.Errorf("%s exists; castcut never overwrites a take (choose another -o)", out)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	bin, err := exec.LookPath("asciinema")
	if err != nil {
		return fmt.Errorf("asciinema not found on PATH (brew install asciinema)")
	}
	fmt.Fprintf(stderr, "castcut: recording %s to %s with asciinema at this terminal's size; exit the command to stop\n",
		shellJoin(cmd), out)
	c := exec.Command(bin, recordArgs(out, o, cmd)...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, stdout, stderr
	runErr := c.Run()
	var ee *exec.ExitError
	if runErr != nil && !errors.As(runErr, &ee) {
		return runErr // asciinema did not start
	}
	// Only the take says how the command ended: asciinema writes an `x` event
	// with its status. Without one, asciinema itself failed and the take is
	// not a recording of the command, whatever exit code it left.
	status, ok := takeStatus(out)
	if !ok {
		why := "exited 0"
		if ee != nil {
			why = ee.String()
		}
		return fmt.Errorf("asciinema %s without finishing the take; %s is incomplete", why, out)
	}
	fmt.Fprintf(stderr, "castcut: take saved to %s\nnext: castcut annotate %s\n", out, out)
	if status != 0 {
		return exitCode(status)
	}
	return nil
}

// takeStatus reads the recorded command's exit status from a finished take:
// its last event is `x` with the status as data.
func takeStatus(path string) (int, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	c, err := parseCast(path, data)
	if err != nil {
		return 0, false
	}
	last := c.Events[len(c.Events)-1]
	var s string
	if last.Kind != "x" || json.Unmarshal(last.Data, &s) != nil {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

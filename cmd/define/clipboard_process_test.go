package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClipboardHelperValidatesBeforeNativeWrite(t *testing.T) {
	for _, tc := range []struct {
		args []string
		text string
	}{
		{nil, "text"}, {[]string{clipboardHelperFlag}, "text"}, {[]string{clipboardHelperFlag, "", "extra"}, "text"},
		{[]string{clipboardHelperFlag, ""}, "text"}, {[]string{clipboardHelperFlag, "a\x00b"}, "text"},
		{[]string{clipboardHelperFlag, "isolated"}, string([]byte{255})}, {[]string{clipboardHelperFlag, "isolated"}, strings.Repeat("x", clipboardTextLimit+1)},
	} {
		if err := runClipboardHelper(tc.args, strings.NewReader(tc.text), func(string, string) error { t.Fatal("invalid request touched native clipboard"); return nil }); err == nil {
			t.Fatal("invalid request accepted")
		}
	}
	literal := "{\\rtf1 literal}\x00世界\né"
	var gotTarget, gotText string
	err := runClipboardHelper([]string{clipboardHelperFlag, "isolated"}, strings.NewReader(literal), func(target, text string) error { gotTarget, gotText = target, text; return nil })
	if err != nil || gotTarget != "isolated" || gotText != literal {
		t.Fatalf("target=%q text=%q error=%v", gotTarget, gotText, err)
	}
}
func clipboardTestProcess(t *testing.T, target string) processClipboardWriter {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return processClipboardWriter{executable: exe, prefix: []string{"-test.run=^TestClipboardProcessChild$", "--"}, target: target, timeout: 2 * time.Second}
}
func TestClipboardProcessLiteralAndBoundedFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "text")
	t.Setenv("DEFINE_CLIPBOARD_CHILD_RECORD", path)
	w := clipboardTestProcess(t, "record")
	literal := "%!PS literal\x00世界\né"
	if err := w.Write(context.Background(), literal); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != literal {
		t.Fatalf("text=%q err=%v", got, err)
	}
	w.target = "flood"
	err = w.Write(context.Background(), "x")
	if err == nil || len(err.Error()) > clipboardDiagnosticLimit+200 {
		t.Fatalf("unbounded or missing error: %v", err)
	}
	w.target = "record"
	if err := w.Write(context.Background(), strings.Repeat("x", clipboardTextLimit+1)); err == nil {
		t.Fatal("oversize accepted")
	}
}
func TestClipboardProcessCancellationReapsChild(t *testing.T) {
	w := clipboardTestProcess(t, "blocked")
	w.timeout = 100 * time.Millisecond
	start := time.Now()
	err := w.Write(context.Background(), "x")
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 2*time.Second {
		t.Fatalf("cancellation: %v elapsed=%v", err, time.Since(start))
	}
}
func TestClipboardHelperOrphanWatchdog(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestClipboardProcessChild$", "--", "parent")
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer pipe.Close()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	reader := bufio.NewReader(pipe)
	ready := make(chan string, 1)
	go func() { line, _ := reader.ReadString('\n'); ready <- line }()
	if got := clipboardAwait(t, ready); got != "ready\n" {
		t.Fatalf("orphan helper not ready: %q %s", got, &stderr)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() { _, err := io.Copy(io.Discard, reader); finished <- err }()
	if err := clipboardAwait(t, finished); err != nil {
		t.Fatal(err)
	}
	// The killed parent cannot close the child's inherited stdout. EOF proves
	// the orphan's own watchdog terminated it without a live Go parent to cancel.
}
func TestClipboardProcessChild(t *testing.T) {
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		return
	}
	args = args[1:]
	if len(args) == 1 && args[0] == "parent" {
		exe, _ := os.Executable()
		cmd := exec.Command(exe, "-test.run=^TestClipboardProcessChild$", "--", clipboardHelperFlag, "orphan")
		cmd.Stdin = strings.NewReader("text")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			os.Exit(3)
		}
		for {
			time.Sleep(time.Hour)
		}
	}
	err := runClipboardHelper(args, os.Stdin, func(target, text string) error {
		switch target {
		case "record":
			return os.WriteFile(os.Getenv("DEFINE_CLIPBOARD_CHILD_RECORD"), []byte(text), 0600)
		case "flood":
			fmt.Fprint(os.Stderr, strings.Repeat("!", clipboardDiagnosticLimit*8))
			return errors.New("failed")
		case "orphan":
			fmt.Fprintln(os.Stdout, "ready")
			fallthrough
		case "blocked":
			for {
				time.Sleep(time.Hour)
			}
		default:
			return errors.New("unknown child target")
		}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

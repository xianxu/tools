package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	clipboardHelperFlag      = "--internal-clipboard-write"
	clipboardDiagnosticLimit = 4096
	clipboardName            = "com.apple.pasteboard.clipboard"
)

// processClipboardWriter isolates synchronous native IO in a killable helper.
type processClipboardWriter struct {
	executable string
	prefix     []string
	target     string
	timeout    time.Duration
}

func newProcessClipboardWriter() (clipboardWriter, error) {
	target, err := clipboardTarget()
	if err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return processClipboardWriter{executable: executable, target: target, timeout: 2 * time.Second}, nil
}
func (w processClipboardWriter) Write(ctx context.Context, text string) error {
	if err := validateClipboardText(text); err != nil {
		return err
	}
	timeout := w.timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := append(append([]string(nil), w.prefix...), clipboardHelperFlag, w.target)
	cmd := exec.CommandContext(ctx, w.executable, args...)
	cmd.Stdin = strings.NewReader(text)
	diagnostics := &clipboardDiagnostics{}
	cmd.Stderr = diagnostics
	cmd.WaitDelay = 100 * time.Millisecond
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("clipboard write: %w", ctx.Err())
		}
		return fmt.Errorf("clipboard write: %w: %s", err, strings.TrimSpace(diagnostics.text))
	}
	return nil
}

// clipboardDiagnostics drains all child diagnostics while retaining a fixed cap.
type clipboardDiagnostics struct{ text string }

func (d *clipboardDiagnostics) Write(p []byte) (int, error) {
	n := len(p)
	left := clipboardDiagnosticLimit - len(d.text)
	if len(p) > left {
		p = p[:left]
	}
	d.text += string(p)
	return n, nil
}

// runClipboardHelper owns the protocol and a watchdog independent of its parent.
// The native function receives only validated, bounded literal UTF-8.
func runClipboardHelper(args []string, in io.Reader, write func(string, string) error) error {
	watchdog := time.AfterFunc(3*time.Second, func() { os.Exit(124) })
	defer watchdog.Stop()
	if len(args) != 2 || args[0] != clipboardHelperFlag || args[1] == "" || len(args[1]) > 4096 || strings.ContainsRune(args[1], 0) || !utf8.ValidString(args[1]) {
		return errors.New("invalid clipboard helper arguments")
	}
	data, err := io.ReadAll(io.LimitReader(in, clipboardTextLimit+1))
	if err != nil {
		return err
	}
	text := string(data)
	if err := validateClipboardText(text); err != nil {
		return err
	}
	return write(args[1], text)
}

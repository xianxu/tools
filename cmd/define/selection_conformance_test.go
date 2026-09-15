//go:build darwin && cgo && conformance

package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/internal/conformance"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

func selectionBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "define")
	if out, err := exec.Command("go", "build", "-tags", "define_clipboard_conformance", "-o", path, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return path
}

func TestPTYSelectionCopiesDuringModelWait(t *testing.T) {
	name, read, closeBoard, err := newIsolatedClipboard()
	if err != nil {
		conformance.SkipOrFail(t, "isolated pasteboard unavailable", err)
		return
	}
	defer closeBoard()
	fake := llmtest.NewFake(t)
	started, release := make(chan struct{}, 1), make(chan struct{})
	fake.Script("", llmtest.Reply{Capture: streamCapture, Started: started, Release: release})
	cmd, f := startDefineBinary(t, selectionBinary(t), t.TempDir(), []string{
		"DEFINE_NO_CAPTURE=1", "DEFINE_CLIPBOARD_CONFORMANCE_TARGET=" + name,
		"DEFINE_LLM_BASE_URL=" + fake.URL, "DEFINE_LLM_API_KEY=pty-conformance",
		"DEFINE_LLM_MODEL=claude-opus-5", "DEFINE_LLM_PROVIDER=anthropic",
	}, "--no-audio")
	out := watch(f)
	awaitActivityPTY(t, out, func(s string) bool { return strings.Contains(s, "› ") })
	f.WriteString("/help\r")
	awaitActivityPTY(t, out, func(s string) bool { return strings.Contains(s, "PageUp") })
	f.WriteString("?why\r")
	awaitActivity(t, started)
	painted := awaitActivityPTY(t, out, func(s string) bool { return strings.Contains(lastFrame(s), "⠋") })
	frame := readFrame(t, lastFrame(painted), 80)
	a := selectionPoint{-1, 0}
	for row, text := range frame.text {
		if col := strings.Index(text, "PageUp"); col >= 0 {
			a = selectionPoint{row, col}
			break
		}
	}
	if a.row < 0 {
		t.Fatalf("source absent from waiting screen: %q", frame.text)
	}
	b := a
	b.col += 5
	if _, err := f.WriteString(selectionWire(a, b)); err != nil {
		t.Fatal(err)
	}
	awaitActivityPTY(t, out, func(s string) bool { return strings.Contains(s, "\x1b[7m") })
	deadline := time.Now().Add(5 * time.Second)
	copied := ""
	for time.Now().Before(deadline) {
		copied, _, err = read()
		if err == nil && copied == "PageUp" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if copied != "PageUp" {
		t.Fatalf("clipboard=%q err=%v", copied, err)
	}
	close(release)
	awaitActivityPTY(t, out, func(s string) bool {
		return strings.Contains(s, "Obsequious") && strings.Contains(lastFrame(s), "› ")
	})
	f.WriteString("\x04")
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	if err := clipboardAwait(t, exited); err != nil {
		t.Fatal(err)
	}
	final := out.take(100 * time.Millisecond)
	if !strings.Contains(final, mouseOff) || !strings.Contains(final, altScreenOff) {
		t.Fatalf("terminal modes not restored: %q", final)
	}
}

func TestSelectionConformanceTargetFailsClosed(t *testing.T) {
	bin := selectionBinary(t)
	for _, target := range []string{"", "com.apple.pasteboard.clipboard"} {
		// The tagged factory reports unavailable; helper dispatch must never run.
		cmd, f := startDefineBinary(t, bin, t.TempDir(), []string{"DEFINE_NO_CAPTURE=1", "DEFINE_CLIPBOARD_CONFORMANCE_TARGET=" + target}, "--no-audio")
		out := watch(f)
		got := awaitActivityPTY(t, out, func(s string) bool { return strings.Contains(s, "copy unavailable:") })
		if !strings.Contains(got, "copy unavailable:") {
			t.Fatal(fmt.Sprint(got))
		}
		f.WriteString("\x04")
		exited := make(chan error, 1)
		go func() { exited <- cmd.Wait() }()
		clipboardAwait(t, exited)
	}
}

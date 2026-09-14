//go:build define_clipboard_conformance

package main

import "testing"

func TestClipboardTargetRequiresIsolation(t *testing.T) {
	for _, name := range []string{"", clipboardName} {
		t.Setenv("DEFINE_CLIPBOARD_CONFORMANCE_TARGET", name)
		if _, err := clipboardTarget(); err == nil {
			t.Fatalf("unsafe target %q accepted", name)
		}
		if _, err := newProcessClipboardWriter(); err == nil {
			t.Fatal("factory bypassed target refusal")
		}
	}
	t.Setenv("DEFINE_CLIPBOARD_CONFORMANCE_TARGET", "isolated")
	got, err := clipboardTarget()
	if err != nil || got != "isolated" {
		t.Fatalf("target=%q err=%v", got, err)
	}
}

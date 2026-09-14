//go:build !define_clipboard_conformance

package main

import "testing"

func TestClipboardTargetIgnoresTestEnvironment(t *testing.T) {
	t.Setenv("DEFINE_CLIPBOARD_CONFORMANCE_TARGET", "isolated")
	got, err := clipboardTarget()
	if err != nil || got != clipboardName {
		t.Fatalf("target=%q err=%v", got, err)
	}
}

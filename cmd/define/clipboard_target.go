//go:build !define_clipboard_conformance

package main

// clipboardTarget selects the user's clipboard; test environment has no effect.
func clipboardTarget() (string, error) { return clipboardName, nil }

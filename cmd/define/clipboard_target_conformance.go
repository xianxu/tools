//go:build define_clipboard_conformance

package main

import (
	"errors"
	"os"
	"strings"
	"unicode/utf8"
)

// clipboardTarget makes a conformance binary fail closed without an isolated board.
func clipboardTarget() (string, error) {
	name := os.Getenv("DEFINE_CLIPBOARD_CONFORMANCE_TARGET")
	if name == "" || name == clipboardName || len(name) > 4096 || strings.ContainsRune(name, 0) || !utf8.ValidString(name) {
		return "", errors.New("conformance clipboard requires an isolated pasteboard")
	}
	return name, nil
}

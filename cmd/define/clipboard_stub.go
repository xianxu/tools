//go:build !darwin || !cgo

package main

import "errors"

// writeNativeClipboard reports that this build has no native clipboard transport.
func writeNativeClipboard(target, text string) error {
	return errors.New("clipboard requires macOS with cgo")
}

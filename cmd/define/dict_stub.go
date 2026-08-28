//go:build !darwin

package main

import (
	"fmt"
	"io"
	"runtime"

	"github.com/xianxu/tools/cmd/define/store"
)

// unsupportedDictionary keeps `go build ./...` and `go vet ./...` green off
// darwin. The tool still compiles everywhere; only the lookup is unavailable.
type unsupportedDictionary struct{}

func (unsupportedDictionary) Lookup(string) (string, error) {
	return "", fmt.Errorf("the system dictionary is only available on macOS (GOOS=%s)", runtime.GOOS)
}

// Same signature as the darwin one, language and warn included: a seam that
// changes shape per platform is two seams.
func systemDictionary(store.Lang, io.Writer) (Dictionary, string) {
	return unsupportedDictionary{}, "none (macOS only)"
}

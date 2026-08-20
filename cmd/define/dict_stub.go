//go:build !darwin

package main

import (
	"fmt"
	"runtime"
)

// unsupportedDictionary keeps `go build ./...` and `go vet ./...` green off
// darwin. The tool still compiles everywhere; only the lookup is unavailable.
type unsupportedDictionary struct{}

func (unsupportedDictionary) Lookup(string) (string, error) {
	return "", fmt.Errorf("the system dictionary is only available on macOS (GOOS=%s)", runtime.GOOS)
}

func systemDictionary() Dictionary { return unsupportedDictionary{} }

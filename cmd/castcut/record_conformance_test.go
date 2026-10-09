//go:build conformance

package main

import (
	"os/exec"
	"testing"

	"github.com/xianxu/tools/internal/conformance"
)

// The fake in record_test.go stands in for asciinema; this runs the same
// contract against the real binary, headless, so the fake cannot drift.
func TestRecordThroughRealAsciinema(t *testing.T) {
	if _, err := exec.LookPath("asciinema"); err != nil {
		conformance.SkipOrFail(t, "asciinema not installed", err)
		return
	}
	recordContract(t, t.TempDir(), "--headless")
}

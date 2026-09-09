//go:build conformance

package main

// Live conformance for the RELEASE STAMP (ARCH-MOCK, applied to the linker).
//
// THE PRODUCTION FLOW AND THE TEST FLOW DID NOT SHARE A BOUNDARY. The unit test
// (TestVersionIsHonestAboutUnstampedBuilds) assigns the Go variable directly,
// which exercises versionLine but NOT the mechanism the formula depends on:
// `-ldflags -X main.version=...`. Rename `version` to anything else and the
// linker silently writes nothing — `go build` exits 0, no error anywhere — so a
// release ships reporting "built from source" while the whole package stays
// green. That is the failure the issue exists to prevent, and it was pinned by
// nothing (#49 BR-2).
//
// So this shells the REAL toolchain with the REAL flag and reads the answer off
// the REAL binary, which is the only place the `-X main.version` symbol path is
// a fact rather than an assumption. It is the same seam Formula/define.rb uses,
// which is what makes it conformance rather than a unit test.
//
//	go test -tags conformance -run Ldflags ./cmd/define/
//
// It SKIPS when there is no Go toolchain to shell, on the suite's rule that
// "not running" is not "wrong".

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xianxu/tools/internal/conformance"
)

// stampedVersion is deliberately NOT the released number: pinning v0.1.0 here
// would make this test a second place the version lives, which is the third copy
// the whole ldflags design exists to avoid.
const stampedVersion = "v9.9.9-conformance"

func TestLdflagsStampReachesTheBinary(t *testing.T) {
	// Routed, not bare: the Go toolchain is an EXTERNAL dependency, so its
	// absence is "did not run" on a normal box and a FAILURE under
	// CONFORMANCE_STRICT=1 — where a silent skip would report green for the one
	// check that guards the release stamp.
	goBin, err := exec.LookPath("go")
	if err != nil {
		conformance.SkipOrFail(t, "no go toolchain to shell", err)
		return
	}

	out := filepath.Join(t.TempDir(), "define")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	// The SAME flag string the formula passes. If this drifts from
	// Formula/define.rb the test stops meaning what it claims.
	//
	// It does NOT replicate the formula's GOFLAGS ("-trimpath -mod=readonly"):
	// those change the build's reproducibility and module resolution, neither of
	// which the `-X` symbol path depends on. What is under test is whether the
	// linker can still find main.version, and that is orthogonal.
	build := exec.CommandContext(t.Context(), goBin, "build",
		"-ldflags", "-X main.version="+stampedVersion, "-o", out, ".")
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build with the formula's ldflags failed: %v\n%s", err, b)
	}

	// RUN FROM A TEMP DIR, not from the package dir `go test` inherits. Harmless
	// today because --version returns above withStore — but this repo's own
	// history is "three deck files reached commits because go test runs with cwd
	// set to cmd/define/", and the directory IS the deck.
	runCmd := exec.CommandContext(t.Context(), out, "--version")
	runCmd.Dir = t.TempDir()
	b, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running the stamped binary failed: %v\n%s", err, b)
	}
	got := strings.TrimSpace(string(b))

	want := "define " + stampedVersion
	if got != want {
		t.Errorf("--version on a stamped build = %q, want %q.\n"+
			"The linker path is broken: `-X main.version` wrote nothing, which "+
			"means the symbol was renamed or moved package. A release built this "+
			"way reports 'built from source' and go build still exits 0, so this "+
			"test is the only thing that says so.", got, want)
	}
	if strings.Contains(got, "built from source") {
		t.Errorf("--version = %q — the stamp did not reach the binary at all", got)
	}
}

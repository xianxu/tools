//go:build darwin && conformance

package main

// Live conformance check for the NOAD seam (ARCH-MOCK).
//
// Cadence is deliberately on-demand rather than CI-scheduled: it depends on the
// host having the New Oxford American Dictionary installed, which a CI runner
// does not. Run it after a macOS upgrade, or whenever a parse bug is reported:
//
//	go test -tags conformance ./cmd/define/
//
// It must run UNSANDBOXED — DCSCopyTextDefinition returns silence, not an
// error, without real access to /System/Library/AssetsV2, which would make this
// test report drift that is really just a sandbox.

import "testing"

func TestFixturesMatchLiveDictionary(t *testing.T) {
	fake, err := loadFakeDictionary("testdata/entries")
	if err != nil {
		t.Fatalf("loadFakeDictionary: %v", err)
	}
	live := systemDictionary()
	// PROBE FIRST, because a sandboxed run and a drifted fixture look identical
	// from inside the loop: DCSCopyTextDefinition returns silence without real
	// access to /System/Library/AssetsV2, so EVERY word fails the same way. One
	// lookup up front separates the two — past this point a failed lookup means
	// the fixture and the live dictionary genuinely disagree, which is the thing
	// this test exists to report.
	//
	// It routes through skipOrFail for the same reason the Skipf sites do (BR-9),
	// and it is the half of that rule a `grep 't\.Skipf('` cannot see: written as
	// an unconditional t.Errorf, an absent dependency was a hard FAILURE, so the
	// non-strict `go test -tags conformance ./...` could never be green offline.
	// One helper now owns both directions.
	if _, err := live.Lookup("sycophantic"); err != nil {
		skipOrFail(t, "system dictionary unreachable", err)
	}
	// Read through Lookup, not fake.entries: a conformance check that bypasses
	// the seam cannot see the fake diverging from the dependency at that seam.
	for word := range fake.entries {
		want, err := fake.Lookup(word)
		if err != nil {
			t.Errorf("%s: unreachable through the fake seam: %v", word, err)
			continue
		}
		got, err := live.Lookup(word)
		if err != nil {
			t.Errorf("%s: live lookup failed: %v (sandboxed?)", word, err)
			continue
		}
		if got != want {
			t.Errorf("%s: NOAD drifted from the fixture — re-run testdata/capture.sh\n live: %.120q\n fixt: %.120q",
				word, got, want)
		}
	}
}

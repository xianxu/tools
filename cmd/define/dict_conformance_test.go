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
	for word, want := range fake.entries {
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

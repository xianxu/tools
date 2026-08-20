//go:build darwin && conformance

package main

// The live property check: run the no-data-loss predicate over thousands of
// REAL dictionary entries, not just the captured corpus and not just fuzz-shaped
// strings.
//
// This is the check that caught M1's four rendering bugs. The corpus test proves
// the property for shapes we sampled; the fuzz target explores arbitrary strings;
// only this one asks "does it hold for the actual dictionary?".
//
// Cadence is on-demand with the rest of the conformance suite (see
// dict_conformance_test.go). Must run UNSANDBOXED.
//
//	go test -tags conformance -run LiveEntries ./cmd/define/

import (
	"bufio"
	"os"
	"testing"
)

const liveSampleSize = 9000

func TestRenderLosesNothingOverLiveEntries(t *testing.T) {
	f, err := os.Open("/usr/share/dict/words")
	if err != nil {
		t.Skipf("no system word list: %v", err)
	}
	defer f.Close()

	dict := systemDictionary()
	var checked, missing, failed, rawPipes int
	// Deterministic stride sample across the whole list, so the words are spread
	// over the alphabet rather than clustered in the a's.
	sc := bufio.NewScanner(f)
	var words []string
	for sc.Scan() {
		words = append(words, sc.Text())
	}
	stride := max(1, len(words)/liveSampleSize)

	for i := 0; i < len(words); i += stride {
		w := words[i]
		raw, err := dict.Lookup(w)
		if err != nil {
			missing++
			continue
		}
		checked++
		out := Render(ParseEntry(raw), RenderOpts{Color: false})
		// Independent oracle — see strayStress. The previous formulation asked
		// isPronunciation to grade its own output and therefore reported 0%
		// while 2.2% of these same entries were showing raw notation.
		if near := strayStress(out); near != "" {
			rawPipes++
			if rawPipes <= 3 {
				t.Errorf("%s: unconverted NOAD notation survived, near %q", w, near)
			}
		}
		if gap := subsequenceGap(alnum(raw), alnum(out)); gap >= 0 {
			failed++
			if failed <= 5 { // report a handful, not thousands
				want := alnum(raw)
				lo, hi := max(0, gap-40), min(len(want), gap+40)
				t.Errorf("%s: alnum loss at rune %d/%d near %q", w, gap, len(want), string(want[lo:hi]))
			}
		}
	}
	if checked < 500 {
		t.Fatalf("only %d live entries checked (%d missing) — sandboxed?", checked, missing)
	}
	t.Logf("checked %d live entries: %d lost content, %d kept raw notation, %d absent from NOAD",
		checked, failed, rawPipes, missing)
	if rawPipes > 0 {
		t.Errorf("%d/%d live entries rendered unconverted NOAD notation (%.1f%%)",
			rawPipes, checked, 100*float64(rawPipes)/float64(checked))
	}
	if failed > 0 {
		t.Errorf("%d/%d live entries lost content (%.1f%%)", failed, checked, 100*float64(failed)/float64(checked))
	}
}

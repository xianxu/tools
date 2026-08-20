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
	"strings"
	"testing"
	"unicode"
)

const liveSampleSize = 9000

func TestRenderLosesNothingOverLiveEntries(t *testing.T) {
	f, err := os.Open("/usr/share/dict/words")
	if err != nil {
		t.Skipf("no system word list: %v", err)
	}
	defer f.Close()

	dict := systemDictionary()
	var checked, missing, failed, rawPipes, nonLatin int
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
		// DCSCopyTextDefinition searches every ACTIVE dictionary, not NOAD (there
		// is no public API to select one), so some English words resolve to a
		// Chinese-dictionary entry with an entirely different structure. Those
		// are outside what this parser targets; counted and reported, not
		// silently dropped. See "Limits" in atlas/define.md.
		if hasHan(raw) {
			nonLatin++
			continue
		}
		checked++
		out := Render(ParseEntry(raw), RenderOpts{Color: false})
		// Independent oracle — see strayStress. The previous formulation asked
		// isPronunciation to grade its own output and therefore reported 0%
		// while 2.2% of these same entries were showing raw notation.
		// Two oracles: strayStress (stress marks outside /…/) and a bare "|"
		// check that consults nothing at all. The second exists because the
		// first cannot see example-separator pipes, which carry no stress mark —
		// the atlas published "0%" on the strength of the narrower one while 2%
		// of entries still showed raw delimiters.
		if near, bar := strayStress(out), strings.IndexByte(out, '|'); near != "" || bar >= 0 {
			if near == "" {
				lo, hi := max(0, bar-50), min(len(out), bar+50)
				near = out[lo:hi]
			}
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
	t.Logf("checked %d live entries: %d lost content, %d kept raw notation; %d non-Latin (other active dictionaries), %d absent",
		checked, failed, rawPipes, nonLatin, missing)
	if rawPipes > 0 {
		t.Errorf("%d/%d live entries rendered unconverted NOAD notation (%.1f%%)",
			rawPipes, checked, 100*float64(rawPipes)/float64(checked))
	}
	if failed > 0 {
		t.Errorf("%d/%d live entries lost content (%.1f%%)", failed, checked, 100*float64(failed)/float64(checked))
	}
}

// hasHan reports whether the entry is predominantly Han script — i.e. it came
// from a Chinese dictionary rather than an English one.
func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

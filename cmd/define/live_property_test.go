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
	"fmt"
	"github.com/xianxu/tools/internal/conformance"
	"os"
	"strings"
	"testing"
	"unicode"

	"github.com/xianxu/tools/cmd/define/store"
)

// Sweep every word. The full pass runs in well under a minute, and sampling is
// how the atlas came to publish "0 raw notation" three times while the next
// order of magnitude still had failures.
const liveSampleSize = 1 << 30

// knownRawNotationEntries and classifyRawNotation live in the UNTAGGED
// rawnotation_test.go, so the normal suite can consume them too.

func TestRenderLosesNothingOverLiveEntries(t *testing.T) {
	f, err := os.Open("/usr/share/dict/words")
	if err != nil {
		conformance.SkipOrFail(t, "no system word list", err)
	}
	defer f.Close()

	// English: this walks /usr/share/dict/words, which is an English word list.
	dict, _ := systemDictionary(store.DefaultLang, nil)
	var checked, missing, failed, rawPipes, nonLatin int
	byCause := map[rawCause]int{}
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
		// Non-Latin entries are counted and reported, never silently dropped.
		//
		// This branch used to catch English words resolving to a Chinese
		// dictionary, because DCSCopyTextDefinition searched every ACTIVE
		// dictionary and the comment here said there was no public API to select
		// one. That was true of the SDK header and false of the framework, and
		// #23 M2 made it false of this code: the sweep now sees the curated
		// English books only. Measured 2026-08-28: 0 non-Latin, where there used
		// to be a class of them.
		//
		// Kept rather than deleted — a curated list is a short honest list, and
		// adding a book to it can bring the shape back. See "Limits" in
		// atlas/define.md.
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
			// CLASSIFY, don't just count. The ratchet used to print one number
			// and three samples, so a movement said nothing about WHICH shape
			// moved and the other survivors had to be dug out by raising this
			// cap by hand — a cost #26 paid once before mechanising it.
			cause := classifyRawNotation(w, out)
			byCause[cause]++
			if byCause[cause] <= 2 { // two exemplars per cause, not three overall
				t.Logf("[%s] %s: unconverted NOAD notation survived, near %q", cause, w, near)
			}
		}
		// Count first: the subsequence check detects loss only, so an insertion
		// passes it silently. Both widths need both directions.
		if len(alnum(raw)) != len(alnum(out)) {
			failed++
			if failed <= 5 {
				t.Errorf("%s: alnum count raw=%d rendered=%d — content inserted or dropped",
					w, len(alnum(raw)), len(alnum(out)))
			}
		} else if gap := subsequenceGap(alnum(raw), alnum(out)); gap >= 0 {
			failed++
			if failed <= 5 { // report a handful, not thousands
				want := alnum(raw)
				lo, hi := max(0, gap-40), min(len(want), gap+40)
				t.Errorf("%s: alnum loss at rune %d/%d near %q", w, gap, len(want), string(want[lo:hi]))
			}
		}
	}
	if checked < 500 {
		// The same missing dependency as dict_conformance_test.go's probe, caught
		// after the fact because this test discovers reachability by sweeping.
		// It is a SKIP, not a failure, unless strict mode says a run that did not
		// execute may not report green (BR-9) — an unconditional Fatalf here is
		// why the sandboxed conformance suite was red before the helper owned
		// both directions.
		conformance.SkipOrFail(t, fmt.Sprintf("only %d live entries reachable (%d missing)", checked, missing), nil)
	}
	t.Logf("checked %d live entries: %d lost content, %d kept raw notation; %d non-Latin, %d absent",
		checked, failed, rawPipes, nonLatin, missing)
	for _, c := range rawCauses {
		t.Logf("  raw notation by cause: %-24s %d", c, byCause[c])
	}
	// The taxonomy must be TOTAL. An unclassified survivor is a shape nobody has
	// described, and the whole point of classifying is that such a shape surfaces
	// instead of being absorbed into whichever cause it happens to resemble.
	if n := byCause[causeUnclassified]; n > 0 {
		t.Errorf("%d survivor(s) matched no known cause — classifyRawNotation is not total over "+
			"the live population; name the shape rather than widening an existing cause", n)
	}
	// The pinned count and its four causes live in rawnotation_test.go. A
	// movement here should be read together with the per-cause lines above: the
	// number alone says something changed, the breakdown says what.
	if rawPipes > knownRawNotationEntries {
		t.Errorf("%d/%d live entries rendered unconverted NOAD notation, up from the known %d — a regression",
			rawPipes, checked, knownRawNotationEntries)
	}
	if rawPipes < knownRawNotationEntries {
		t.Errorf("only %d/%d entries render raw notation, below the pinned %d — lower knownRawByCause to lock the improvement in",
			rawPipes, checked, knownRawNotationEntries)
	}
	// PER CAUSE, because a total is a weak ratchet: two causes can move in
	// opposite directions and leave it unchanged, and a misclassification
	// between two known causes is invisible in a sum. This is what makes a
	// movement attributable rather than absorbed.
	for _, c := range rawCauses {
		if c == causeUnclassified {
			continue // asserted zero above
		}
		if got, want := byCause[c], knownRawByCause[c]; got != want {
			t.Errorf("%s: %d entries, pinned at %d — update knownRawByCause with the reason, "+
				"since this names WHICH shape moved", c, got, want)
		}
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

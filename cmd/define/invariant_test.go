package main

import (
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/xianxu/tools/cmd/define/store"
)

// The no-data-loss invariant.
//
// Every letter and digit of the raw entry must appear, in order, in the
// rendered output. Punctuation may be restructured (":" becomes a line break,
// examples gain quotes); words may never vanish and never reorder.
//
// This is what makes a best-effort parser over schema-less text safe: an
// unrecognized construct degrades to a paragraph instead of disappearing.
//
// It guarantees FIDELITY, NOT COMPLETENESS. It asserts that nothing NOAD
// returned is lost on the way to the screen. It says nothing about what NOAD
// declined to return — DCSCopyTextDefinition("bank") yields only homograph 1,
// and no test here would notice. See the Non-goals in the plan.

var slashSpan = regexp.MustCompile(`/[^/\n]*/`)

// strayStress is an INDEPENDENT oracle for "did any raw NOAD notation survive
// rendering?" — it does not call isPronunciation.
//
// The earlier check did: it scanned the output for |…| spans and asked
// isPronunciation whether each was a pronunciation. A test that asks the
// function under test to grade its own output can only ever detect false
// positives. Every span isPronunciation wrongly REJECTED was, by construction,
// reported as "not a pronunciation" and passed — so the check read 0% while
// 2.2% of live entries displayed raw pipes, and that false 0% was published in
// the atlas.
//
// This oracle rests on a fact about the notation instead: a NOAD stress mark
// (ˈ or ˌ) may appear only inside a /…/ span in rendered output. Anything else
// is unconverted source.
func strayStress(out string) string {
	if i := strayStressAt(out); i >= 0 {
		rest := slashSpan.ReplaceAllString(out, "")
		lo, hi := max(0, i-50), min(len(rest), i+50)
		return rest[lo:hi]
	}
	return ""
}

// strayStressAt is the same oracle returning the POSITION, in the coordinate
// space of the stripped text.
//
// Split out because two callers need different things from one fact: this test
// wants a window to print, and classifyRawNotation wants the index to decide
// whether the leak sits in the headword block. A copy of these two lines lived
// in the classifier until the boundary review pointed out that an oracle
// restated is an oracle that can drift from itself.
//
// The stripped coordinate space is load-bearing, not incidental: the window
// strayStress returns does NOT exist verbatim in `out`, so a caller that
// searched for it there got -1 every time. That bug is why this returns an
// index rather than leaving each caller to find one.
func strayStressAt(out string) int {
	return strings.IndexAny(slashSpan.ReplaceAllString(out, ""), "ˈˌ")
}

// rawNotationNear reports whether ANY unconverted NOAD notation survived, and a
// window to show. THE one trip predicate — it was written in three spellings
// across three files, which is three chances for the ratchet, the corpus test
// and the classifier to disagree about what they are counting.
//
// A disjunction of two independent oracles, and both halves are needed: the
// stress-mark oracle cannot see example-separator pipes (they carry no stress
// mark), and the pipe oracle cannot see a stress mark that leaked without one.
func rawNotationNear(out string) (string, bool) {
	if near := strayStress(out); near != "" {
		return near, true
	}
	if i := strings.IndexByte(out, '|'); i >= 0 {
		lo, hi := max(0, i-50), min(len(out), i+50)
		return out[lo:hi], true
	}
	return "", false
}

func alnum(s string) []rune {
	var out []rune
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out = append(out, r)
		}
	}
	return out
}

// subsequenceGap returns the index in want of the first rune that could not be
// matched in got, or -1 if want is a subsequence of got. Reporting the position
// (with context at the call site) is the difference between a usable failure
// and an unreadable one.
func subsequenceGap(want, got []rune) int {
	j := 0
	for i, r := range want {
		for j < len(got) && got[j] != r {
			j++
		}
		if j == len(got) {
			return i
		}
		j++
	}
	return -1
}

func TestRenderLosesNothing(t *testing.T) {
	d := testDict(t)
	if len(d.entries) == 0 {
		t.Fatal("empty corpus — this test would be vacuous")
	}
	for word, raw := range d.entries {
		t.Run(word, func(t *testing.T) {
			out := Render(ParseEntry(raw), RenderOpts{Color: false})
			want, got := alnum(raw), alnum(out)
			// The subsequence check is one-directional: it detects LOSS only.
			// Render inserting content passes it silently — which is how %q's
			// escape sequences ("\u00ad" → five alphanumeric runes) survived five
			// review rounds. Render draws every letter from the raw entry, so the
			// counts must match exactly.
			if len(want) != len(got) {
				t.Errorf("alnum count %d rendered vs %d raw — Render inserted or dropped content", len(got), len(want))
			}
			if i := subsequenceGap(want, got); i >= 0 {
				lo := max(0, i-40)
				hi := min(len(want), i+40)
				t.Errorf("renderer dropped content at rune %d of %d\n  near: %q\n  (raw %d alnum runes, rendered %d)",
					i, len(want), string(want[lo:hi]), len(want), len(got))
			}
		})
	}
}

func TestSubsequenceGapDetectsALoss(t *testing.T) {
	if subsequenceGap([]rune("abc"), []rune("axbxc")) != -1 {
		t.Error("want a subsequence to be accepted")
	}
	if subsequenceGap([]rune("abc"), []rune("ac")) != 1 {
		t.Error("want the gap reported at the dropped rune")
	}
}

// FuzzRenderLosesNothing is the property form of the invariant.
//
// TestRenderLosesNothing above only proves the property for the shapes someone
// already sampled, which is the opposite of what the invariant is for. This
// target explores the open input space, seeded from the corpus. It is how the
// M1 boundary review found the head-reordering, first-token, and head-overwrite
// bugs; minimized crashers land in testdata/fuzz/ as permanent regressions.
func FuzzRenderLosesNothing(f *testing.F) {
	d, err := loadFakeDictionary("testdata/entries", store.DefaultLang)
	if err != nil {
		f.Fatal(err)
	}
	for _, raw := range d.entries {
		f.Add(raw)
	}
	f.Add("in in | ˈin | noun a thing.")
	f.Add("wug | wʌg | noun a thing. verb | wʌg | 1 to wug.")
	f.Add("iPhone\nA combination mobile phone and media player.")

	f.Fuzz(func(t *testing.T, raw string) {
		out := Render(ParseEntry(raw), RenderOpts{Color: false})
		want, got := alnum(raw), alnum(out)
		if len(want) != len(got) {
			t.Fatalf("alnum count raw=%d rendered=%d for %q -> %q", len(want), len(got), raw, out)
		}
		if i := subsequenceGap(want, got); i >= 0 {
			t.Fatalf("alnum loss at rune %d of %d for %q", i, len(want), raw)
		}
	})
}

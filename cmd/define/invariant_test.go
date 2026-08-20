package main

import (
	"regexp"
	"strings"
	"testing"
	"unicode"
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
	rest := slashSpan.ReplaceAllString(out, "")
	if i := strings.IndexAny(rest, "ˈˌ"); i >= 0 {
		lo, hi := max(0, i-50), min(len(rest), i+50)
		return rest[lo:hi]
	}
	return ""
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
	d, err := loadFakeDictionary("testdata/entries")
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
		if i := subsequenceGap(alnum(raw), alnum(out)); i >= 0 {
			t.Fatalf("alnum loss at rune %d of %d for %q", i, len([]rune(alnum(raw))), raw)
		}
	})
}

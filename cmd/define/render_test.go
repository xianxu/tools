package main

import (
	"fmt"
	"strings"
	"testing"
)

// Render had no direct test at the M1 boundary review: the colour path,
// prettyPronunciations, homograph rendering, and the FromHead header
// suppression were all covered only indirectly by the invariant, which is blind
// to anything that preserves letter order.

func TestRenderHomographIsVisible(t *testing.T) {
	// Only one homograph is reachable through this API (see Non-goals), so the
	// number must be shown — it is the user's only signal that "bank" here means
	// the riverbank and the financial sense was never returned.
	out := Render(ParseEntry(fixture(t, "bank")), RenderOpts{Color: false})
	first := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(first, "bank") || !strings.Contains(first, "1") {
		t.Errorf("header %q should show the homograph number", first)
	}
}

func TestRenderHeadKeepsSourceOrder(t *testing.T) {
	// present is "present 1 pres·ent" — homograph BEFORE syllabification, the
	// opposite of record. A renderer with a fixed field order reorders one of them.
	out := Render(ParseEntry(fixture(t, "present")), RenderOpts{Color: false})
	first := strings.SplitN(out, "\n", 2)[0]
	iHomo, iSyl := strings.Index(first, "1"), strings.Index(first, "pres·ent")
	if iHomo < 0 || iSyl < 0 || iHomo > iSyl {
		t.Errorf("header %q must keep NOAD's order: headword, homograph, syllabification", first)
	}
}

func TestRenderGluedPOSNotPrintedTwice(t *testing.T) {
	out := Render(ParseEntry(fixture(t, "record")), RenderOpts{Color: false})
	lines := strings.Split(out, "\n")

	// The head line carries the glued POS...
	if !strings.Contains(lines[0], "noun") {
		t.Errorf("head line %q lost the glued part-of-speech", lines[0])
	}
	// ...and no block heading repeats it. A block heading is a line indented by
	// exactly two spaces; the earlier version of this test looked for an empty
	// line that also had a "  " prefix, which is unsatisfiable.
	for _, line := range lines[1:] {
		if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "   ") {
			continue
		}
		if strings.TrimSpace(line) == "" {
			t.Error("empty block heading — FromHead suppression left a blank line")
		}
		if strings.TrimSpace(line) == "noun" {
			t.Error("block heading repeated the head's part-of-speech")
		}
	}
}

func TestRenderColorOnlyWhenAsked(t *testing.T) {
	e := ParseEntry(fixture(t, "sycophantic"))
	if plain := Render(e, RenderOpts{Color: false}); strings.Contains(plain, "\x1b[") {
		t.Error("ANSI escapes present with Color:false")
	}
	colored := Render(e, RenderOpts{Color: true})
	if !strings.Contains(colored, "\x1b[") {
		t.Error("no ANSI escapes with Color:true")
	}
	// Colour must be presentation-only: stripping the escapes reproduces the
	// plain rendering exactly.
	if stripANSI(colored) != Render(e, RenderOpts{Color: false}) {
		t.Error("colour changed more than presentation")
	}
}

func TestPrettyPronunciations(t *testing.T) {
	p := newPalette(false)
	got := prettyPronunciations("sycophantically | ˌsikəˈfan(t)ək(ə)lē | adverb", p)
	if want := "sycophantically /ˌsikəˈfan(t)ək(ə)lē/ adverb"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// Prose between pipes is an example separator, not a pronunciation.
	prose := "a | b c | d"
	if got := prettyPronunciations(prose, p); got != prose {
		t.Errorf("prose span rewritten: %q", got)
	}
}

// stripANSI removes every escape sequence, leaving what a reader sees.
//
// It defers to scanEscape rather than scanning for 'm': a CSI with any other
// final byte (erase-line, cursor moves) used to run this loop past the sequence
// and eat real text with it, which mattered once #21 put arbitrary styled
// streams through the same assertions.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			n := scanEscape(s[i:])
			if n <= 0 {
				n = len(s) - i // an unterminated escape: the rest is the sequence
			}
			i += n
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// --- structural goldens over the corpus ------------------------------------

// Sense numbering is the one structure the alnum invariant is blind to: a bare
// numeral in an example ("she ran in the 200 meters") preserves letter order
// while producing sense "200". Requiring a consecutive run per block catches it.
//
// The run does not always start at 1: when the head swallows sense 1's number
// ("use verb 1 [with object]"), the body opens at 2. So this asserts
// consecutiveness from whatever the block starts at, and separately bounds the
// start — which is what distinguishes "opens at 2" from "opens at 200".
func TestCorpusSenseNumbersAreSequential(t *testing.T) {
	d := testDict(t)
	for word, raw := range d.entries {
		t.Run(word, func(t *testing.T) {
			for bi, blk := range ParseEntry(raw).Blocks {
				want := -1
				for _, s := range blk.Senses {
					if s.Number == "" {
						continue
					}
					if want < 0 {
						if s.Number != "1" && s.Number != "2" {
							t.Errorf("block %d (%s): sequence starts at %q — a prose numeral, not a sense",
								bi, blk.POS, s.Number)
							break
						}
						want, _ = atoi(s.Number)
					}
					if s.Number != fmt.Sprint(want) {
						t.Errorf("block %d (%s): sense number %q, want %d", bi, blk.POS, s.Number, want)
					}
					want++
				}
			}
		})
	}
}

// The complement of the test above: it validates numbers that WERE assigned,
// and is blind to numbering that was never assigned at all. When the "want"
// anchor was wrong, use's verb senses 2-5 silently merged into the preceding
// sense text and this is the shape that catches it.
func TestCorpusNumberedSensesAreNotSwallowed(t *testing.T) {
	d := testDict(t)
	for word, raw := range d.entries {
		t.Run(word, func(t *testing.T) {
			for bi, blk := range ParseEntry(raw).Blocks {
				var numbered int
				for _, s := range blk.Senses {
					if s.Number != "" {
						numbered++
					}
					// A sense whose own text still contains a later split
					// candidate means the numbering anchor rejected it.
					for n := 2; n <= 9; n++ {
						if strings.Contains(s.Gloss, fmt.Sprintf(" %d ", n)) && numbered == 0 {
							t.Errorf("block %d (%s): gloss carries an unconsumed sense number %d: %.80q",
								bi, blk.POS, n, s.Gloss)
						}
					}
				}
			}
		})
	}
}

// Block structure golden. bank and run previously grew a phantom "adjective"
// block from "(banked as adjective)" — order-preserving, so invisible to the
// invariant.
func TestCorpusBlockStructure(t *testing.T) {
	want := map[string][]string{
		"sycophantic":  {"adjective"},
		"quokka":       {"noun"},
		"ephemeral":    {"adjective", "noun"},
		"defenestrate": {"verb"},
		"bank":         {"noun", "verb"},
		"record":       {"noun", "verb"},
		"run":          {"verb", "noun"},
		"gaslighting":  {"noun"},
		// The phantom-block family: a POS word inside a bracket (man, thing),
		// in plain prose ("a noun phrase", subject), and a real opener after a
		// paren ("(subject to) adjective", subject). Each of these shipped a bug.
		"man":   {"noun", "verb", "exclamation"},
		"thing": {"noun"},
		// A part of speech opening directly after an editorial note's "]".
		"complete": {"adjective", "verb"},
		"subject":  {"noun", "adjective", "adverb", "verb"},
	}
	d := testDict(t)
	for word, expect := range want {
		t.Run(word, func(t *testing.T) {
			raw, err := d.Lookup(word)
			if err != nil {
				// FAIL, not skip. The fixtures are COMMITTED, so an absent one is
				// a deleted or renamed file, never a dependency this machine
				// happens to lack — and each exists to pin a specific parse shape
				// (here: a part of speech opening straight after an editorial
				// note's "]"). Skipping retired that coverage silently while the
				// package still reported ok (BR-16).
				t.Fatalf("fixture absent for %q: %v — testdata/entries is committed, "+
					"so this is a missing file, not a missing dependency", word, err)
			}
			var got []string
			for _, b := range ParseEntry(raw).Blocks {
				got = append(got, b.POS)
			}
			if strings.Join(got, ",") != strings.Join(expect, ",") {
				t.Errorf("blocks = %v, want %v", got, expect)
			}
		})
	}
}

// No raw NOAD notation may survive into rendered output: the tool exists to show
// Google-style /…/, and a screen mixing both notations misses the point.
//
// Checked with strayStress, which does NOT consult isPronunciation — see the
// oracle note there for why the obvious formulation of this test is circular.
func TestNoRawPronunciationNotationSurvives(t *testing.T) {
	d := testDict(t)
	for word, raw := range d.entries {
		t.Run(word, func(t *testing.T) {
			out := Render(ParseEntry(raw), RenderOpts{Color: false})
			if near := strayStress(out); near != "" {
				t.Errorf("unconverted NOAD notation survived rendering, near %q", near)
			}
			// Second, blunter oracle: a raw "|" is NOAD's delimiter and has no
			// place in rendered output. strayStress cannot see example-separator
			// pipes, because those carry no stress mark.
			if i := strings.IndexByte(out, '|'); i >= 0 {
				lo, hi := max(0, i-50), min(len(out), i+50)
				t.Errorf("raw NOAD delimiter survived rendering, near %q", out[lo:hi])
			}
		})
	}
}

// Wrapping must break at spaces. The terminal will wrap for us otherwise, but it
// wraps at the column — splitting words mid-syllable ("fing/ers"), which is
// exactly what a dictionary entry must not do.
func TestWrapTextBreaksAtSpaces(t *testing.T) {
	got := wrapText("mid 16th century from French sycophante or via Latin", 24, 4)
	for _, line := range strings.Split(got, "\n") {
		if visibleLen(strings.TrimSpace(line)) == 0 {
			t.Error("blank line produced")
		}
	}
	// No line may exceed the width once its indent is counted.
	for i, line := range strings.Split(got, "\n") {
		w := visibleLen(line)
		if i == 0 {
			w += 4
		}
		if w > 24 && len(strings.Fields(line)) > 1 {
			t.Errorf("line %d is %d wide, want <= 24: %q", i, w, line)
		}
	}
	// And nothing is lost.
	if strings.Join(strings.Fields(got), " ") != "mid 16th century from French sycophante or via Latin" {
		t.Errorf("wrapping changed the words: %q", got)
	}
}

// Width is measured in visible columns: wrapping on byte length would break
// early on every coloured line, and these lines are coloured.
func TestWrapTextIgnoresANSI(t *testing.T) {
	plain := wrapText("alpha beta gamma delta", 20, 0)
	colored := wrapText("alpha \x1b[35mbeta\x1b[0m gamma delta", 20, 0)
	if strings.Count(plain, "\n") != strings.Count(colored, "\n") {
		t.Errorf("colour changed the wrap points:\n plain: %q\n color: %q", plain, colored)
	}
}

// A word longer than the line goes on its own line rather than being cut.
func TestWrapTextDoesNotSplitAWord(t *testing.T) {
	long := "supercalifragilisticexpialidocious"
	got := wrapText("a "+long+" b", 10, 0)
	if !strings.Contains(got, long) {
		t.Errorf("a long word was split: %q", got)
	}
}

// Width 0 means a pipe: leave the text alone, since a consumer re-wraps for
// itself and baked-in breaks cannot be undone.
func TestWrapTextDisabledAtZeroWidth(t *testing.T) {
	in := "some text that would otherwise wrap at a narrow width"
	if got := wrapText(in, 0, 4); got != in {
		t.Errorf("wrapped at width 0: %q", got)
	}
}

// The no-data-loss property must survive wrapping — it only inserts whitespace.
func TestWrappedRenderStillLosesNothing(t *testing.T) {
	d := testDict(t)
	for word, raw := range d.entries {
		t.Run(word, func(t *testing.T) {
			out := Render(ParseEntry(raw), RenderOpts{Color: false, Width: 60})
			want, got := alnum(raw), alnum(out)
			if len(want) != len(got) {
				t.Errorf("alnum count raw=%d wrapped=%d", len(want), len(got))
			}
			if i := subsequenceGap(want, got); i >= 0 {
				t.Errorf("wrapping lost content at rune %d", i)
			}
		})
	}
}

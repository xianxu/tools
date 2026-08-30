package main

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// Render had no direct test at the M1 boundary review: the colour path,
// prettyPronunciations, homograph rendering, and the FromHead header
// suppression were all covered only indirectly by the invariant, which is blind
// to anything that preserves letter order.

func TestRenderHomographIsVisible(t *testing.T) {
	// Only one homograph is reachable through this API (see Non-goals), so the
	// number must be shown — it is the user's only signal that "bank" here means
	// the riverbank and the financial sense was never returned.
	out, _ := Render(ParseEntry(fixture(t, "bank")), RenderOpts{Color: false})
	first := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(first, "bank") || !strings.Contains(first, "1") {
		t.Errorf("header %q should show the homograph number", first)
	}
}

func TestRenderHeadKeepsSourceOrder(t *testing.T) {
	// present is "present 1 pres·ent" — homograph BEFORE syllabification, the
	// opposite of record. A renderer with a fixed field order reorders one of them.
	out, _ := Render(ParseEntry(fixture(t, "present")), RenderOpts{Color: false})
	first := strings.SplitN(out, "\n", 2)[0]
	iHomo, iSyl := strings.Index(first, "1"), strings.Index(first, "pres·ent")
	if iHomo < 0 || iSyl < 0 || iHomo > iSyl {
		t.Errorf("header %q must keep NOAD's order: headword, homograph, syllabification", first)
	}
}

func TestRenderGluedPOSNotPrintedTwice(t *testing.T) {
	out, _ := Render(ParseEntry(fixture(t, "record")), RenderOpts{Color: false})
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
	if plain, _ := Render(e, RenderOpts{Color: false}); strings.Contains(plain, "\x1b[") {
		t.Error("ANSI escapes present with Color:false")
	}
	colored, _ := Render(e, RenderOpts{Color: true})
	if !strings.Contains(colored, "\x1b[") {
		t.Error("no ANSI escapes with Color:true")
	}
	// Colour must be presentation-only: stripping the escapes reproduces the
	// plain rendering exactly.
	plain, _ := Render(e, RenderOpts{Color: false})
	if stripANSI(colored) != plain {
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
// EVERY captured language, not just English (#31). It took testDict(t) while two
// committed comments — in capture.sh and rawnotation_test.go — described it as a
// hard zero "over the committed corpus". That was harmless while the corpus was
// English plus a Spanish corpus nobody claimed this covered; capturing Italian
// is what would have made the phrasing actively misleading, so the check is
// widened to match the claim rather than the claim narrowed to match the check.
//
// Measured before widening: 15 Devoto-Oli entries render 0 raw pipes and parse 0
// IPA, so Italian passes the hard zero rather than needing an exemption.
func TestNoRawPronunciationNotationSurvives(t *testing.T) {
	for _, lang := range capturedLanguages(t) {
		t.Run(string(lang), func(t *testing.T) {
			noRawNotationIn(t, testDictFor(t, lang))
		})
	}
}

func noRawNotationIn(t *testing.T, d *fakeDictionary) {
	t.Helper()
	for word, raw := range d.entries {
		t.Run(word, func(t *testing.T) {
			out, _ := Render(ParseEntry(raw), RenderOpts{Color: false})
			// THE trip predicate, shared with the live ratchet and the
			// classifier — it was written in three spellings across three files,
			// which is three chances for them to disagree about what they count.
			// It is a disjunction: the stress oracle cannot see
			// example-separator pipes, which carry no stress mark.
			if near, tripped := rawNotationNear(out); tripped {
				t.Errorf("unconverted NOAD notation survived rendering, near %q", near)
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
		if visibleCells(strings.TrimSpace(line)) == 0 {
			t.Error("blank line produced")
		}
	}
	// No line may exceed the width once its indent is counted.
	for i, line := range strings.Split(got, "\n") {
		w := visibleCells(line)
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
//
// EVERY captured language (#31), for the same reason its unwrapped sibling was
// widened: wrapping is not English-specific, and a corpus checked by neither
// form of the invariant is a corpus nothing holds to it.
func TestWrappedRenderStillLosesNothing(t *testing.T) {
	for _, lang := range capturedLanguages(t) {
		t.Run(string(lang), func(t *testing.T) {
			wrappedLosesNothingIn(t, testDictFor(t, lang))
		})
	}
}

func wrappedLosesNothingIn(t *testing.T, d *fakeDictionary) {
	t.Helper()
	for word, raw := range d.entries {
		t.Run(word, func(t *testing.T) {
			out, _ := Render(ParseEntry(raw), RenderOpts{Color: false, Width: 60})
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

// A non-English entry carries NO pronunciation notation, and that is correct
// rather than a gap — but the REASON differs per language, so the reason is the
// row rather than a shared sentence.
//
// ONE table, not one function per language. #31 first added Italian as a second
// function whose doc comment was appended to this one's, leaving BOTH
// misdocumented: the Spanish explanation ran into the Italian one and the
// Spanish test lost its header. That is the one-predicate-two-spellings family
// this repo keeps closing, and the boundary review caught a new instance being
// created rather than an old one surviving.
//
// SCOPED DELIBERATELY, because the unscoped claim is false. This is about a
// non-English word in ITS OWN dictionary. The same word in an ENGLISH one is a
// different case with real notation — NOAD gives `jalapeño` four anglicised
// pronunciations — and conflating the two is how "Spanish has no notation"
// becomes wrong. The English half is asserted below.
// notationExempt names languages whose dictionary DOES write pronunciation, so a
// missing row below is distinguishable from a deliberate absence.
//
// Without it, "this language has no row" and "this language must not have a row"
// look identical, and #34 would add German — whose Duden field is REAL
// (`Wạsser`, `ˈkatsə, Kạtze`) — to a table asserting the opposite. The value is
// the reason, because a bare set would answer "which" and not "why".
var notationExempt = map[store.Lang]string{
	"en": "NOAD writes IPA for every entry; English is the baseline the others are contrasted with",
}

func TestNonEnglishEntriesCarryNoPronunciationNotation(t *testing.T) {
	rows := []struct {
		lang store.Lang
		// why says what a failure MEANS for this language. The two languages
		// reach the same zero for different reasons, and a shared message would
		// blur exactly the distinction #30 reads this pair for.
		why string
	}{
		{
			"es",
			"Spanish orthography is phonemic — spelling plus the written accent determines " +
				"the pronunciation exactly — so the Larousse writes none, unlike NOAD's " +
				"`lig·a·ment | ˈliɡəmənt |`. If this starts passing, the Larousse changed, " +
				"not the parser. It also makes the RECORDING load-bearing in a way it is not " +
				"for English: for a Spanish word the audio is the only place pronunciation " +
				"information exists, which is the argument behind #27's locale work",
		},
		{
			"it",
			"Devoto-Oli DOES write something — `(cià·o)`, `(pìz·za)`, `(e·sprès·so)` — and it " +
				"is syllabification with stress, not a phonetic transcription. isPronunciation " +
				"declines it, correctly. If the parser ever reads those parens as an IPA span, " +
				"#30's click target and the renderer's `/…/` would both start showing syllable " +
				"breaks as pronunciation. Measured 0 of 15 against the live dictionary before " +
				"the corpus was captured",
		},
	}
	// EVERY curated language is either a row or exempt, with its reason. Fifth
	// instance of the family: a language-keyed table that does not range over
	// `curated` silently gains no row when a language is curated.
	for lang := range curated {
		if _, exempt := notationExempt[lang]; exempt {
			continue
		}
		if !slices.ContainsFunc(rows, func(r struct {
			lang store.Lang
			why  string
		}) bool {
			return r.lang == lang
		}) {
			t.Errorf("production curates %v for %s, and it is neither a row here nor in "+
				"notationExempt — so nothing states whether that dictionary writes "+
				"pronunciation, which is the fact #30 reads", curated[lang], lang)
		}
	}
	for _, tc := range rows {
		t.Run(string(tc.lang), func(t *testing.T) {
			d := testDictFor(t, tc.lang)
			if len(d.entries) == 0 {
				t.Fatalf("the %s corpus is empty; this test would pass vacuously", tc.lang)
			}
			for word := range d.entries {
				raw, err := d.Lookup(word)
				if err != nil {
					t.Errorf("%s: %v", word, err)
					continue
				}
				if got := ParseEntry(raw).IPA; got != "" {
					t.Errorf("%s: parsed a pronunciation %q from a %s entry — %s",
						word, got, tc.lang, tc.why)
				}
			}
		})
	}
}

// The other half of the scope, so the claim above cannot be over-read: a Spanish
// word in an ENGLISH dictionary does carry notation, and it is anglicised.
func TestASpanishWordInAnEnglishEntryDoesCarryNotation(t *testing.T) {
	// NOT a skip: jalapeño is a committed fixture (capture.sh captures it for
	// exactly this test), so its absence is a broken corpus, never a missing
	// dependency. A skip here would let the test assert nothing and still read
	// green.
	raw, err := testDict(t).Lookup("jalapeño")
	if err != nil {
		t.Fatalf("jalapeño is committed under testdata/entries/en; %v", err)
	}
	if got := ParseEntry(raw).IPA; got == "" {
		t.Error("no pronunciation parsed for jalapeño in the English dictionary — the " +
			"'Spanish has no notation' rule is about the Spanish DICTIONARY, not about " +
			"Spanish words, and this is the case that distinguishes them")
	}
}

// Width is measured in COLUMNS, not runes, and both exceptions are this
// program's daily traffic.
//
// NOAD writes `bänˈZHo͝or` — the o͝o carries a combining double breve, a rune
// that occupies no column of its own — so counting runes reports more width than
// the terminal uses and cuts text that fits. A Japanese entry is full-width: ten
// runes are twenty columns, so counting runes builds a frame twice as tall as it
// measured, the terminal scrolls, and every row the screen placed has moved.
func TestVisibleCellsCountsColumns(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want int
	}{
		{"plain ascii", "hello", 5},
		{"a colour costs nothing", "\x1b[1;36mhello\x1b[0m", 5},
		{"a cursor move costs nothing either", "hello\x1b[3D", 5},
		// NOAD's own anglicisation of `bonjour`, combining breve and all.
		{"a combining mark rides on the rune before it", "bänˈZHo͡or", 9},
		{"CJK is two columns a rune", "日本語", 6},
		{"fullwidth latin is two as well", "ＡＢ", 4},
		{"mixed", "a日b", 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := visibleCells(tc.in); got != tc.want {
				t.Errorf("visibleCells(%q) = %d columns, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// The rendered bytes ARE the product, and this golden is the only thing that
// says so (#30 M2.1, Done-when 3).
//
// M2.1 changed Render's signature to also return regions, and the risk is that
// collecting them alters a single byte — which would change what `define <word>`,
// `-raw` and `> out.txt` print, paths D6 promises are untouched. The other tests
// here assert STRUCTURE (a line exists, a colour appears, wrapping holds); none
// of them would notice a stray space.
//
// GENERATED FROM THE COMMIT BEFORE the signature change, which is what makes it
// evidence rather than a restatement: comparing Render to itself proves nothing,
// and a golden regenerated by the change it exists to catch is worse. Three
// option sets, because colour and width take different paths through the walk.
//
// When a rendering change is INTENDED, regenerate this file deliberately and say
// so in the commit — the diff is then the description of what a reader will see.
func TestRenderOutputMatchesTheCorpusGolden(t *testing.T) {
	want, err := os.ReadFile("testdata/golden/render-corpus.golden")
	if err != nil {
		// NOT a skip: the golden is committed, so an unreadable one is a broken
		// checkout, never an absent dependency.
		t.Fatalf("golden unreadable: %v", err)
	}
	d := testDict(t)
	if len(d.entries) == 0 {
		t.Fatal("empty corpus; this test would be vacuous")
	}
	var words []string
	for w := range d.entries {
		words = append(words, w)
	}
	sort.Strings(words)

	var b strings.Builder
	for _, w := range words {
		e := ParseEntry(d.entries[w])
		for _, opt := range []RenderOpts{{Color: true, Width: 80}, {Color: false, Width: 0}, {Color: true, Width: 40}} {
			b.WriteString("\x00" + w + "\x00")
			rendered, _ := Render(e, opt)
			b.WriteString(rendered)
		}
	}
	if got := b.String(); got != string(want) {
		t.Errorf("the rendered corpus changed. %s", firstDifference(got, string(want)))
	}
}

// firstDifference reports WHERE two renderings diverge, since the corpus is
// 300KB and a diff of the whole thing says nothing.
func firstDifference(got, want string) string {
	n := min(len(got), len(want))
	for i := 0; i < n; i++ {
		if got[i] != want[i] {
			lo := max(0, i-40)
			return fmt.Sprintf("first difference at byte %d:\n got %q\nwant %q", i,
				got[lo:min(i+40, len(got))], want[lo:min(i+40, len(want))])
		}
	}
	return fmt.Sprintf("one is a prefix of the other: %d bytes vs %d", len(got), len(want))
}

// Regions ADDRESS the output they were collected from: every span is where it
// says it is, in the line and the display column it claims.
//
// A region that is merely present is worthless — the whole point is that a click
// at (row, col) finds the thing under the pointer. Checked over the whole corpus
// so an entry shape nobody thought of is checked too.
func TestRegionsAddressTheRenderedOutput(t *testing.T) {
	d := testDict(t)
	found := 0
	for word, raw := range d.entries {
		e := ParseEntry(raw)
		for _, opt := range []RenderOpts{{Color: true, Width: 80}, {Color: false, Width: 0}} {
			rendered, regions := Render(e, opt)
			lines := strings.Split(rendered, "\n")
			for _, r := range regions {
				found++
				if r.Line < 0 || r.Line >= len(lines) {
					t.Errorf("%s: region %q claims line %d of %d", word, r.Text, r.Line, len(lines))
					continue
				}
				// The span is at the column it claims, measured in the cells a
				// terminal will use — escapes stepped over, so a highlighted
				// deck word inside the line does not shift it.
				plain, cols := visibleIndex(lines[r.Line])
				at := -1
				for i := range cols {
					if cols[i] == r.Col {
						at = i
						break
					}
				}
				if at < 0 || !strings.HasPrefix(plain[at:], r.Text) {
					t.Errorf("%s: region %q claims line %d column %d, where the screen shows %q",
						word, r.Text, r.Line, r.Col, plainAround(plain, r.Col, cols))
					continue
				}
				if r.Width != visibleCells(r.Text) {
					t.Errorf("%s: region %q is %d cells wide, want %d", word, r.Text, r.Width, visibleCells(r.Text))
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("no regions over the whole corpus — the walk found nothing, so nothing is asserted")
	}
}

func plainAround(plain string, col int, cols []int) string {
	for i := range cols {
		if cols[i] >= col {
			return plain[i:min(i+20, len(plain))]
		}
	}
	return ""
}

// The underline NEVER reaches Render's output (#30 D6, Done-when 4).
//
// The mark is spliced by the SCREEN, and that placement is a promise rather than
// an implementation detail: `define <word>`, `echo w | define`, `-raw` and
// `> out.txt` keep today's bytes exactly. An underline emitted here would leak
// into all four — and a file or a pipe cannot be clicked, so the mark would be
// decoration claiming an affordance that does not exist there.
//
// The corpus golden already fails if any byte moves; this says WHICH byte and
// why, so the next reader meets the reason rather than a diff.
func TestRenderNeverMarksSpansItself(t *testing.T) {
	d := testDict(t)
	marked := 0
	for word, raw := range d.entries {
		e := ParseEntry(raw)
		for _, opt := range []RenderOpts{{Color: true, Width: 80}, {Color: false, Width: 0}} {
			rendered, regions := Render(e, opt)
			if strings.Contains(rendered, underlineOn) || strings.Contains(rendered, underlineOff) {
				t.Errorf("%s: Render emitted the clickable underline, which would reach a pipe and a file", word)
			}
			marked += len(regions)
		}
	}
	if marked == 0 {
		t.Fatal("no regions over the corpus, so the absence of marks proves nothing")
	}
}

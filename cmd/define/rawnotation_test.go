package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The raw-notation ratchet's shared surface, UNTAGGED on purpose.
//
// live_property_test.go is `//go:build darwin && conformance`, so anything
// declared there is invisible to the normal suite — which is why the atlas could
// not consume the pinned count and why a live-only taxonomy would be untestable
// off-conformance. An untagged file compiles into every build, so one producer
// serves three consumers: the live ratchet, the classifier's unit test, and the
// doc-sync test that keeps the atlas honest.

// knownRawByCause pins the live population PER CAUSE, not just its total.
//
// A single number is a weak ratchet: two causes can move in opposite directions
// and leave it unchanged, and a misclassification between two known causes is
// invisible in a total. Pinning the breakdown is what "attributable rather than
// absorbed" actually asks for — a movement names its group.
//
// Measured against the live dictionary; see the live sweep for the date and
// width. It is FOUR causes, not one: this ratchet's comment used to say "all of
// one known cause" and name the prose numeral, which was true of a minority of
// them. The rest were never looked at, because the run sampled only the first
// three survivors.
var knownRawByCause = map[rawCause]int{
	causeProseNumeral:          7,
	causeHeadwordPronunciation: 7,
	causePhrasePronunciation:   8,
	causeLiteralPipe:           4,
}

// knownRawNotationEntries is the pinned TOTAL, derived from the breakdown rather
// than restated beside it — one producer, so the two cannot disagree.
//
// A RATCHET, not a clean zero, and bidirectional: a rise fails as a regression,
// a fall fails demanding the gain be locked in, so the number cannot drift in
// either direction without someone deciding it should.
var knownRawNotationEntries = func() int {
	n := 0
	for _, c := range knownRawByCause {
		n += c
	}
	return n
}()

// rawCause is why one entry still shows raw notation. A CLOSED set, and
// causeUnclassified is a real member rather than an error value: the live run
// asserts it is zero, so a shape nobody has seen surfaces as unclassified
// instead of being absorbed into whichever cause it happens to resemble.
type rawCause string

const (
	causeProseNumeral          rawCause = "prose-numeral"
	causeHeadwordPronunciation rawCause = "headword-pronunciation"
	causePhrasePronunciation   rawCause = "phrase-pronunciation"
	causeLiteralPipe           rawCause = "literal-pipe"
	causeUnclassified          rawCause = "unclassified"
)

// rawCauses is the reporting order, so a run's per-cause line is stable.
var rawCauses = []rawCause{
	causeProseNumeral, causeHeadwordPronunciation,
	causePhrasePronunciation, causeLiteralPipe, causeUnclassified,
}

// classifyRawNotation says WHICH of the four shapes an entry's rendered output
// hit, so a future movement in the count names the group that moved rather than
// printing one number and three samples.
//
// TOTAL: every input gets exactly one cause. The precedence is
// stress-mark-before-pipe, and it is a decision rather than an ordering
// accident — a stress mark surviving outside a /…/ span is unambiguously leaked
// notation, while a pipe can be legitimate content (`pipe` defines the
// character). An entry firing both is therefore a pronunciation leak that
// happens to contain a pipe, not the reverse.
//
// It must consult BOTH oracles, because the live check is a disjunction
// (live_property_test.go: `strayStress(out) != "" || IndexByte(out,'|') >= 0`)
// and the stress-mark half carries no pipe at all. A pipe-keyed classifier would
// mis-handle most of the population.
func classifyRawNotation(word, rendered string) rawCause {
	// The SAME strip the oracle performs (strayStress, invariant_test.go), so
	// the index below is measured in the same coordinate space. strayStress
	// returns a window cut from the STRIPPED text, which does not exist verbatim
	// in the input — a first version searched for it in `rendered` and silently
	// got -1 every time.
	rest := slashSpan.ReplaceAllString(rendered, "")
	if i := strings.IndexAny(rest, "ˈˌ"); i >= 0 {
		// Glued to the headword, or deep among the idioms? Measured over the
		// captured exemplars: `hundred` puts it at byte 27 of 1828, immediately
		// after the headword line; `shape` at 2436 of 4116, in a phrase block
		// two thirds of the way down. The gap is two orders of magnitude, so the
		// cut is nowhere near either population.
		if i < headwordBlockBytes {
			return causeHeadwordPronunciation
		}
		return causePhrasePronunciation
	}
	if strings.IndexByte(rendered, '|') >= 0 {
		// The entry's SUBJECT is the character: "• the symbol |."
		if strings.Contains(rendered, "the symbol |") {
			return causeLiteralPipe
		}
		return causeProseNumeral
	}
	return causeUnclassified
}

// headwordBlockBytes is how far into an entry the headword's own block reaches.
// Measured, not guessed — see classifyRawNotation.
const headwordBlockBytes = 128

// The classifier, pinned against REAL captured entries rather than hand-written
// snippets — one exemplar per cause, from testdata/rawnotation/.
//
// They live outside testdata/entries/<lang>/ deliberately: capturedLanguages
// walks that directory expecting language names, and
// TestNoRawPronunciationNotationSurvives asserts a hard ZERO raw pipes over the
// committed corpus. These four are exactly the entries that would break that
// assertion, so they are a separate corpus with a different contract — here the
// raw notation is the point, not a defect in the fixture.
//
// This is what makes the taxonomy testable off-conformance. Without it the
// classifier would only ever run inside a darwin+conformance sweep against a
// host-dependent dictionary, which is not a pin.
func TestClassifyRawNotationOverRealEntries(t *testing.T) {
	for _, tc := range []struct {
		word string
		want rawCause
	}{
		// A prose numeral continuing a sense sequence: "…2. euros for the
		// postcard | the restaurant charged $15…" — the documented cause.
		{word: "charge", want: causeProseNumeral},
		// A pronunciation glued to the headword: "(aˈhəndrədzˈhəndrəd/)".
		{word: "hundred", want: causeHeadwordPronunciation},
		// A phrase block's pronunciation run into prose, deep among the idioms:
		// "lick someoneˌoud əv ˈSHāp/".
		{word: "shape", want: causePhrasePronunciation},
		// The entry whose SUBJECT is the character: "• the symbol |."
		{word: "pipe", want: causeLiteralPipe},
	} {
		t.Run(tc.word, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "rawnotation", tc.word+".txt"))
			if err != nil {
				t.Fatalf("reading exemplar: %v", err)
			}
			out := Render(ParseEntry(string(raw)), RenderOpts{Width: 0})
			// The exemplar must actually still trip the live oracle, or it has
			// stopped being an exemplar and this test passes vacuously.
			if strayStress(out) == "" && !strings.ContainsRune(out, '|') {
				t.Fatalf("%s no longer renders raw notation — re-capture or retire the exemplar", tc.word)
			}
			if got := classifyRawNotation(tc.word, out); got != tc.want {
				t.Errorf("classifyRawNotation(%s) = %q, want %q", tc.word, got, tc.want)
			}
		})
	}
}

// The taxonomy is TOTAL: anything that trips the oracle gets a cause, and only
// input that trips NEITHER oracle is unclassified.
func TestClassifyRawNotationIsTotal(t *testing.T) {
	if got := classifyRawNotation("clean", "an ordinary rendered entry with no notation"); got != causeUnclassified {
		t.Errorf("clean text = %q, want %q", got, causeUnclassified)
	}
	// Precedence, stated in classifyRawNotation and asserted here: a stress mark
	// outside a slash span is unambiguously leaked notation, while a pipe can be
	// legitimate content — so an entry firing BOTH is a pronunciation leak that
	// happens to contain a pipe, not a pipe case.
	both := "someˌstress here and the symbol | too"
	if got := classifyRawNotation("both", both); got == causeLiteralPipe {
		t.Errorf("an entry firing both oracles classified as %q; stress-mark causes outrank pipe causes", got)
	}
}

package main

import (
	"os"
	"path/filepath"
	"regexp"
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

// classifyRawNotation says WHICH of the known shapes an entry hit, so a movement
// in the count names the group that moved rather than printing one number.
//
// EVERY cause is a POSITIVE test, and causeUnclassified is the residue. That is
// the whole design, and a first version got it backwards: it had two catch-all
// branches — any stress mark that was not headword-shaped fell into "phrase",
// any pipe that was not the literal case fell into "prose numeral" — so every
// oracle survivor landed in a named bucket BY CONSTRUCTION. The live run's
// assertion that causeUnclassified is zero could not fail, and a genuinely new
// shape would have been absorbed into whichever bucket it resembled. That is the
// exact failure this classifier exists to prevent, implemented as its opposite.
//
// Probed after the fix: an invented shape with a stress mark in no recognised
// position, and this issue's own dormant "| AmE brɛnt, BrE brɛnt |", both now
// reach causeUnclassified.
//
// Precedence is stress-before-pipe where both match, because a stress mark
// outside a /…/ span is unambiguously leaked notation while a pipe can be
// legitimate content (`pipe` defines the character).
func classifyRawNotation(rendered string) rawCause {
	if i := strayStressAt(rendered); i >= 0 {
		switch {
		// Glued to the headword: NOAD collapses the pronunciation into a
		// parenthesised group in the opening block — "(aˈhəndrədzˈhəndrəd/)".
		//
		// The PARENTHESIS is the signature; position alone is not. A first
		// version tested only `i < headwordBlockBytes`, which claimed any stray
		// stress in the first 128 bytes and therefore absorbed the stress-marked
		// form of #26's own dormant shape — "| AmE ˈhəndrəd, BrE ˈhʌndrəd |"
		// classified as headword-pronunciation. The captured `brent` reached the
		// residue only because it is a monosyllable carrying no stress mark,
		// which is luck rather than design. Position is kept as a cheap
		// conjunct, not as the test.
		case i < headwordBlockBytes && stressIsParenthesised(rendered, i):
			return causeHeadwordPronunciation
		// A phrase block's pronunciation run into prose: NOAD writes these as an
		// idiom followed immediately by its pronunciation, so the stress mark is
		// preceded by lowercase prose with no sentence break — "lick
		// someoneˌoud əv ˈSHāp/". The trailing slash is the tell: the closing
		// delimiter survived while the opening one was consumed.
		case strings.Contains(strayStressWindow(rendered, i, 60), "/"):
			return causePhrasePronunciation
		}
		return causeUnclassified
	}
	if strings.IndexByte(rendered, '|') >= 0 {
		switch {
		// The entry's SUBJECT is the character: "• the symbol |."
		case strings.Contains(rendered, "the symbol |"):
			return causeLiteralPipe
		// A prose numeral taken as a sense number leaves the pipe adjacent to a
		// digit-and-period run — "2. euros for the postcard | the restaurant".
		// POSITIVE, not an else: that adjacency is what the defect produces.
		case proseNumeralPipe.MatchString(rendered):
			return causeProseNumeral
		}
		return causeUnclassified
	}
	return causeUnclassified
}

// proseNumeralPipe matches a sense-number run followed by a pipe on the same
// line — the signature of a prose numeral accepted as a sense opener.
var proseNumeralPipe = regexp.MustCompile(`(?m)^\s*\d+\.[^|\n]*\|`)

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
			if _, tripped := rawNotationNear(out); !tripped {
				t.Fatalf("%s no longer renders raw notation — re-capture or retire the exemplar", tc.word)
			}
			if got := classifyRawNotation(out); got != tc.want {
				t.Errorf("classifyRawNotation(%s) = %q, want %q", tc.word, got, tc.want)
			}
		})
	}
}

// causeUnclassified must be REACHABLE for input that TRIPS the oracle.
//
// This is the guard on the design, and it exists because the first version got
// it exactly backwards. Both branches were catch-alls, so every oracle survivor
// landed in a named bucket by construction, the live run's
// `unclassified == 0` could not fail, and a new shape would have been absorbed
// into whichever bucket it resembled — the failure this classifier exists to
// prevent, shipped as its opposite. A guard that cannot fail is not a guard, and
// its passing was reported as evidence.
//
// So: novel shapes, each tripping the oracle, each landing in the residue.
func TestUnclassifiedIsReachableForOracleTrippingInput(t *testing.T) {
	for _, tc := range []struct{ name, in string }{
		{"a stress mark in no recognised position", strings.Repeat("x", 3000) + "ˈnovel-shape"},
		{"a pipe with no sense-number run", strings.Repeat("y", 3000) + " a | in a new context"},
		// The shape #26 was filed for. It is dormant — a British dictionary left
		// curated["en"] — and if one is ever added back, this is what makes it
		// surface as unclassified rather than be silently counted as something
		// it is not.
		//
		// THREE forms, because the first version of this test used only the
		// monosyllable and passed for the wrong reason: `brent` carries no stress
		// mark at all, so it reached the residue via the pipe branch while a
		// POSITION-ONLY headword branch was still absorbing every polysyllabic
		// form. The two below are the cases that were silently mis-attributed —
		// they sit inside the first 128 bytes and do carry stress marks.
		{"the dormant AmE/BrE block", "brent\n\n    | AmE brɛnt, BrE brɛnt | noun (British English) "},
		{"dual-locale, polysyllabic", "hundred  hun·dred\n\n    | AmE ˈhəndrəd, BrE ˈhʌndrəd | numeral\n"},
		{"dual-locale, longer word", "laboratory\n\n    | AmE ˈlabrəˌtôrē, BrE ləˈbɒrət(ə)ri | noun\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, tripped := rawNotationNear(tc.in); !tripped {
				t.Fatalf("the input does not trip the oracle, so this proves nothing")
			}
			if got := classifyRawNotation(tc.in); got != causeUnclassified {
				t.Errorf("a novel shape classified as %q — the residue is unreachable and new "+
					"shapes are being absorbed into named causes", got)
			}
		})
	}
}

// Input that trips NEITHER oracle is unclassified too, and the precedence holds.
func TestClassifyRawNotationIsTotal(t *testing.T) {
	if got := classifyRawNotation("an ordinary rendered entry with no notation"); got != causeUnclassified {
		t.Errorf("clean text = %q, want %q", got, causeUnclassified)
	}
	// Precedence, stated in classifyRawNotation and asserted here: a stress mark
	// outside a slash span is unambiguously leaked notation, while a pipe can be
	// legitimate content — so an entry firing BOTH is a pronunciation leak that
	// happens to contain a pipe, not a pipe case.
	// The EXACT cause, not "anything but literal-pipe". An assertion by
	// exclusion passes for the residue too, so it cannot tell "the precedence
	// held" from "neither branch matched" — and the second is a real outcome
	// here, since every branch is now a positive signature.
	both := "hundred\n\n    (ˈhəndrəd/) and the symbol | too"
	if _, tripped := rawNotationNear(both); !tripped {
		t.Fatal("the fixture does not trip the oracle, so this proves nothing")
	}
	if got := classifyRawNotation(both); got != causeHeadwordPronunciation {
		t.Errorf("an entry firing BOTH oracles classified as %q, want %q — a parenthesised "+
			"stress mark outranks a pipe, because a pipe can be legitimate content",
			got, causeHeadwordPronunciation)
	}
}

package main

import (
	"testing"

	"github.com/xianxu/tools/cmd/define/play"
)

// Every string here is a REAL gloss from the committed corpus. D2 claimed labels
// simply lead the gloss; they do not always — a grammar bracket or a
// parenthetical can come first — and that is why this table exists rather than a
// prefix match written from the decision.
func TestReadGloss(t *testing.T) {
	for _, tc := range []struct {
		name, gloss string
		axis        play.Axis
		label       string
		usable      bool
		text        string
	}{
		// --- the simple case D2 described
		{"a bare register label", "archaic form (something) into a mass; solidify",
			play.AxisRegister, "archaic", true, "form (something) into a mass; solidify"},
		{"rare", "rare throw (someone) out of a window",
			play.AxisRegister, "rare", true, "throw (someone) out of a window"},
		{"dated", "dated a manservant or valet.",
			play.AxisRegister, "dated", true, "a manservant or valet."},
		{"historical", "historical a vassal.",
			play.AxisRegister, "historical", true, "a vassal."},

		// --- domain labels
		{"a domain label", "Grammar (of a tense or participle) expressing an action now",
			play.AxisDomain, "Grammar", true, "expressing an action now"},
		{"Law", "Law (also court record) an official report of the proceedings",
			play.AxisDomain, "Law", true, "an official report of the proceedings"},
		{"Computing", "Computing a number of related items of information",
			play.AxisDomain, "Computing", true, "a number of related items of information"},
		{"Nautical", "Nautical the after part of a ship's bottom",
			play.AxisDomain, "Nautical", true, "the after part of a ship's bottom"},
		{"a two-word domain", "American football (of a quarterback) successfully throw a pass",
			play.AxisDomain, "American football", true, "successfully throw a pass"},

		// --- THE SHAPE D2 MISSED: a grammar bracket leads, the label follows.
		{"a bracket before the label", "[no object] Military (of a soldier) illegally run away",
			play.AxisDomain, "Military", true, "illegally run away"},
		{"a bracket before a register label", "[with adjective] informal a book considered in terms of its readability",
			play.AxisRegister, "informal", true, "a book considered in terms of its readability"},
		{"a parenthetical before the label", "(the runs) informal diarrhea.",
			play.AxisRegister, "informal", true, "diarrhea."},

		// --- regional labels are NOT an axis, but must be scanned PAST
		{"regional alone is not an axis", "British English (of a locomotive) provide additional power",
			play.AxisGeneral, "", true, "provide additional power"},
		{"regional then register", "North American English informal a person who shows off",
			play.AxisRegister, "informal", true, "a person who shows off"},
		{"register with a trailing region", "informal, mainly North American English used, irrespective of sex",
			play.AxisRegister, "informal", true, "used, irrespective of sex"},
		{"a bracket then a region", "[no object] British English conclude the sale of a property",
			play.AxisGeneral, "", true, "conclude the sale of a property"},

		// --- unlabelled, the common case
		{"no label at all", "the land alongside or sloping down to a river or lake",
			play.AxisGeneral, "", true, "the land alongside or sloping down to a river or lake"},
		{"a parenthetical and no label", "(especially in sports) the best performance or most remarkable",
			play.AxisGeneral, "", true, "the best performance or most remarkable"},

		// --- THE WORD BOUNDARY. These are SYNTHETIC, deliberately: the corpus
		// contains no gloss beginning "Lawrence" or "rarefied", which is exactly
		// why the guard needs rows written for it. Mutation-checked — deleting
		// the boundary check in hasLabelPrefix left every corpus row green, the
		// same shape #35 hit when "German" matched inside "Germany"
		// (origin_test.go:127) and the mask, not the boundary, was holding the
		// test up.
		{"a domain label is not a prefix of a word", "Lawrence's own account of the events",
			play.AxisGeneral, "", true, "Lawrence's own account of the events"},
		{"nor is a register label", "rarefied air at high altitude",
			play.AxisGeneral, "", true, "rarefied air at high altitude"},
		{"nor a longer one", "informally arranged among the participants",
			play.AxisGeneral, "", true, "informally arranged among the participants"},
		{"Musical is not Music", "Musical instruments of the baroque period",
			play.AxisGeneral, "", true, "Musical instruments of the baroque period"},
		{"and the boundary is not refusing everything", "Music a set of notes",
			play.AxisDomain, "Music", true, "a set of notes"},

		// --- NOT DEFINITIONS. D4 asserted Sense.Gloss is always "a single clean
		// definition line"; the corpus says otherwise, and an option reading
		// "[with object]" would make the form look broken.
		{"a bare grammar bracket", "[with object]", play.AxisGeneral, "", false, ""},
		{"an inflection note", "(plural men /men/)", play.AxisGeneral, "", false, ""},
		{"comparative forms", "(evener, evenest)", play.AxisGeneral, "", false, ""},
		{"stacked inflections", "(mans) (, manning /maniNG/) [with object]", play.AxisGeneral, "", false, ""},
		{"a variant note", "(also jalapeño pepper)", play.AxisGeneral, "", false, ""},
		{"empty", "", play.AxisGeneral, "", false, ""},

		// --- CROSS-REFERENCES. "another term for menhaden" tests nothing about
		// meaning; as a distractor it is noise, as an answer it is unanswerable.
		{"another term for", "another term for menhaden", play.AxisGeneral, "", false, ""},
		{"short for", "short for criminal record", play.AxisGeneral, "", false, ""},
		{"plural form of", "plural form of base1", play.AxisGeneral, "", false, ""},
		{"a pronunciation then a cross-reference", "/ˈbāsēz/ plural form of basis", play.AxisGeneral, "", false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := readGloss(tc.gloss)
			if got.Axis != tc.axis {
				t.Errorf("axis = %v, want %v", got.Axis, tc.axis)
			}
			if got.Label != tc.label {
				t.Errorf("label = %q, want %q", got.Label, tc.label)
			}
			if got.Usable != tc.usable {
				t.Errorf("usable = %v, want %v (text %q)", got.Usable, tc.usable, got.Text)
			}
			if tc.usable && got.Text != tc.text {
				t.Errorf("text = %q,\n want %q", got.Text, tc.text)
			}
		})
	}
}

// D3a: NOAD defines near-synonyms through each other, so the cross-reference IS
// the signal. A reduction, not a proof — the test says so by naming what it
// cannot catch.
func TestNearSynonymExclusion(t *testing.T) {
	for _, tc := range []struct {
		name          string
		aWord, aGloss string
		bWord, bGloss string
		wantExcluded  bool
	}{
		{"the target's gloss names the candidate", "sycophantic", "behaving or done in an obsequious way", "obsequious", "obedient to excess", true},
		{"the candidate's gloss names the target", "obsequious", "obedient to excess", "sycophantic", "behaving in an obsequious way", true},
		{"unrelated words", "mesa", "an isolated flat-topped hill", "quokka", "a small short-tailed wallaby", false},
		// The measured false-positive guard: three corpus glosses contain "thing".
		{"a short common word is not a cross-reference", "record", "a thing constituting a piece of evidence", "thing", "an object one need not name", false},
		{"word boundaries, not substrings", "run", "move at a speed faster than a walk", "u", "the letter", false},
		{"a longer word inside another is not a match", "man", "an adult male human being", "manservant", "a male servant", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := crossReferenced(tc.aWord, tc.aGloss, tc.bWord, tc.bGloss)
			if got != tc.wantExcluded {
				t.Errorf("crossReferenced = %v, want %v", got, tc.wantExcluded)
			}
		})
	}
}

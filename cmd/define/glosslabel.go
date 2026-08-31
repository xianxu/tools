package main

import (
	"strings"

	"github.com/xianxu/tools/cmd/define/play"
)

// Reading NOAD's own editorial apparatus off the head of a gloss: the usage
// labels that give a distractor its axis, and the shapes that mean "this is not
// a definition at all".
//
// This lives in main, not in play, for two reasons that agree. It reads PROSE,
// and play imports nothing (D5a). And it is dictionary knowledge, which D5 keeps
// on this side of the seam — play receives finished options and never parses.

// glossFacts is everything one scan of a gloss's head yields.
//
// ONE function rather than senseLabel + isDefinition + definitionText, because
// all three answers come from the same left-to-right walk. Three entry points
// would walk it three times and, worse, could disagree — a gloss judged usable
// by one and labelled by another is a bug with no obvious home.
type glossFacts struct {
	// Axis is what this sense can supply as a distractor.
	Axis play.Axis
	// Label is the text that decided the Axis, for the Log and for tests. Empty
	// when the gloss carries no axis-bearing label.
	Label string
	// Text is the definition with the leading apparatus removed — the option
	// line a learner reads.
	Text string
	// Usable reports whether Text is a definition at all.
	Usable bool
}

// noadDomainLabels — NOAD's subject-field labels, capitalized in its own style.
//
// A CLOSED TABLE this repo owns, and #35 already argued the case for the shape:
// originLanguages faced the same objection and answered that a table restating a
// fact the CDN owns goes stale, while one reading a dictionary's EDITORIAL
// PROSE does not — the set of field labels a dictionary prints is stable, small,
// and ours to read. A missing row costs a distractor its axis and it degrades to
// general; it never produces a wrong question, which is what makes an incomplete
// table an acceptable state rather than a latent bug.
//
// Longest first: "American football" must win over any prefix of it.
var noadDomainLabels = []string{
	"American football", "Psychoanalysis", "Archaeology", "Mathematics",
	"Architecture", "Linguistics", "Philosophy", "Psychology", "Statistics",
	"Astronomy", "Chemistry", "Computing", "Geometry", "Heraldry", "Medicine",
	"Military", "Nautical", "Politics", "Theology", "Anatomy", "Baseball",
	"Biology", "Ecology", "Finance", "Geology", "Grammar", "Cricket", "Physics",
	"Printing", "Botany", "Mining", "Music", "Zoology", "Bridge", "Chess",
	"Sports", "Law",
}

// noadRegisterLabels — the usage labels that mark a sense as non-neutral in TONE
// or CURRENCY. Lowercase in NOAD's style, which is also what keeps them from
// colliding with the capitalized domain set.
var noadRegisterLabels = []string{
	"vulgar slang", "poetic/literary", "derogatory", "euphemistic", "historical",
	"humorous", "informal", "offensive", "technical", "archaic", "dialect",
	"literary", "formal", "dated", "slang", "rare",
}

// noadRegionalLabels are scanned PAST but are not an axis, and that is a
// deliberate narrowing rather than an oversight.
//
// The settled taxonomy has exactly three values (issue Revisions, 2026-08-30).
// Regional marks WHERE a sense is used, which is neither a subject field nor a
// tone — folding it into register would make "register confusion" mean two
// different things and blunt the one finding #17 M2 exists to read. They must
// still be RECOGNIZED, because NOAD stacks them in front of real labels:
// "North American English informal a person who shows off" carries a register
// label that a scanner stopping at the first unknown word would never reach.
//
// A regional-only sense is therefore a general distractor. If that loses
// something worth having, the fix is a fourth axis, decided deliberately.
var noadRegionalLabels = []string{
	"Australian and New Zealand English", "South African English",
	"North American English", "New Zealand English", "Australian English",
	"Canadian English", "Scottish English", "British English", "Indian English",
	"Irish English", "US English",
}

// crossRefLeads open a gloss that points at another entry instead of defining
// anything. As an option they are unanswerable ("another term for menhaden"
// tests nothing about meaning); as the ANSWER they are worse.
var crossRefLeads = []string{
	"another term for", "variant spelling of", "past participle of",
	"abbreviation for", "plural form of", "short for", "see also", "past of",
	"singular of",
	// A BARE "see " is deliberately NOT here. It would catch NOAD's shortest
	// cross-references and would also reject any real definition beginning
	// "see the light" or "see something through", losing a usable candidate for
	// a gain the whole-phrase rows above already mostly cover. Excluding a good
	// candidate is cheap — it degrades to one fewer distractor — so the rows
	// here are only the ones specific enough to earn the risk.
}

// minDefinitionLen: below this there is no definition left after the apparatus
// is stripped. The bare-bracket cases reduce to exactly "", so this mostly
// guards fragments; 4 keeps real short glosses like "a vassal." and "diarrhea."
const minDefinitionLen = 4

// readGloss walks the head of a gloss once.
//
// The walk exists because D2's claim — "labels LEAD the gloss, so extracting one
// is a PREFIX match" — is true of most senses and false of enough to matter.
// Measured over the committed corpus: "[no object] Military (of a soldier)…" and
// "[with adjective] informal a book…" put a GRAMMAR BRACKET first, and
// "(the runs) informal diarrhea." puts a parenthetical there. A prefix match
// written from the decision would have missed every one of them and silently
// called those senses unlabelled.
func readGloss(gloss string) glossFacts {
	f := glossFacts{Axis: play.AxisGeneral}
	rest := strings.TrimSpace(gloss)

	for {
		rest = strings.TrimLeft(rest, " \t,;")
		if rest == "" {
			break
		}
		switch rest[0] {
		case '[':
			if i := strings.IndexByte(rest, ']'); i >= 0 {
				rest = rest[i+1:]
				continue
			}
		case '(':
			// Depth-counted: "(mans) (, manning /maniNG/)" is two groups, but
			// "(a (nested) note)" would end early on a plain IndexByte.
			if i := closeParen(rest); i >= 0 {
				rest = rest[i+1:]
				continue
			}
		case '/':
			// A leading pronunciation, as in "/ˈbāsēz/ plural form of basis".
			// Bounded so an unmatched slash in prose cannot eat the definition.
			if i := strings.IndexByte(rest[1:], '/'); i >= 0 && i < 40 {
				rest = rest[i+2:]
				continue
			}
		}
		if label, kind, ok := leadingLabel(rest); ok {
			rest = rest[len(label):]
			// Domain outranks register: it is the scarcer axis and the more
			// specific fact. Neither overwrites a label already found.
			if kind == play.AxisDomain && f.Axis != play.AxisDomain {
				f.Axis, f.Label = play.AxisDomain, label
			} else if kind == play.AxisRegister && f.Axis == play.AxisGeneral {
				f.Axis, f.Label = play.AxisRegister, label
			}
			continue
		}
		break
	}

	// "mainly"/"chiefly" only ever qualify a following regional label, which is
	// not an axis — so whatever survives here is the definition.
	rest = strings.TrimSpace(rest)
	f.Text = rest
	f.Usable = len(rest) >= minDefinitionLen && !hasCrossRefLead(rest)
	if !f.Usable {
		f.Text = ""
	}
	return f
}

// leadingLabel matches the longest known label at the head, on a word boundary.
//
// The boundary is what keeps "Law" from matching "Lawrence" and "rare" from
// matching "rarefied" — the same defect #35's origin table was bitten by, where
// "German" matched inside "Germany" (origin_test.go:127).
func leadingLabel(s string) (string, play.Axis, bool) {
	for _, set := range []struct {
		labels []string
		axis   play.Axis
	}{
		{noadDomainLabels, play.AxisDomain},
		{noadRegisterLabels, play.AxisRegister},
		{noadRegionalLabels, play.AxisGeneral}, // recognized, not an axis
	} {
		for _, l := range set.labels {
			if hasLabelPrefix(s, l) {
				return l, set.axis, true
			}
		}
	}
	// A qualifier only ever introduces a regional label, so consume the pair.
	for _, q := range []string{"mainly ", "chiefly "} {
		if strings.HasPrefix(s, q) {
			for _, l := range noadRegionalLabels {
				if hasLabelPrefix(s[len(q):], l) {
					return q + l, play.AxisGeneral, true
				}
			}
		}
	}
	return "", play.AxisGeneral, false
}

func hasLabelPrefix(s, label string) bool {
	if !strings.HasPrefix(s, label) {
		return false
	}
	rest := s[len(label):]
	return rest == "" || !isWordByte(rest[0])
}

func isWordByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b >= 0x80
}

func closeParen(s string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return -1
}

func hasCrossRefLead(s string) bool {
	low := strings.ToLower(s)
	for _, p := range crossRefLeads {
		if strings.HasPrefix(low, p) {
			return true
		}
	}
	return false
}

// minCrossRefWord is why "thing" does not exclude half the deck.
//
// Measured over the corpus: three first-glosses (record, subject, use) contain
// the word "thing", and every one of those is a coincidence rather than a
// cross-reference. Six characters clears them while keeping the case this
// mechanism exists for — NOAD glosses `sycophantic` as "behaving or done in an
// obsequious way", and "obsequious" is ten.
const minCrossRefWord = 6

// crossReferenced is D3a: NOAD defines near-synonyms THROUGH each other, so a
// headword appearing in the other's gloss is the dictionary telling us the two
// are close.
//
// A REDUCTION, not a proof, and worth being plain about: two deck words can be
// near-synonyms NOAD never cross-references, and nothing here would know. That
// residual is accepted because the options are DEFINITIONS — two different words
// rarely share one — and because this form has no model veto by design. What it
// removes is the case that is both likeliest and most visible: the pair the
// learner met through each other's entry.
func crossReferenced(aWord, aGloss, bWord, bGloss string) bool {
	return mentions(aGloss, bWord) || mentions(bGloss, aWord)
}

func mentions(gloss, word string) bool {
	if len(word) < minCrossRefWord {
		return false
	}
	low, w := strings.ToLower(gloss), strings.ToLower(word)
	for i := 0; ; {
		j := strings.Index(low[i:], w)
		if j < 0 {
			return false
		}
		j += i
		beforeOK := j == 0 || !isWordByte(low[j-1])
		end := j + len(w)
		afterOK := end == len(low) || !isWordByte(low[end])
		if beforeOK && afterOK {
			return true
		}
		i = j + 1
	}
}

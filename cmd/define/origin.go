package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
)

// originLanguages maps the modern language names NOAD writes in an etymology to
// their codes.
//
// A CLOSED TABLE, where ParseLang and localeFor both refuse one — and the
// difference is whose fact it is. Those refuse because the CDN and the installed
// dictionaries own what exists, so a table here would restate someone else's
// fact and go stale when they add a row. This table reads NOAD's EDITORIAL
// PROSE: the set of language names a dictionary writes in its etymologies, which
// is stable, small, and ours to read. It makes no claim about what the CDN
// serves — an inferred language with no recording degrades through #29's
// fallback exactly as a typed one does.
//
// Drawn from a survey of 300 entries (2026-08-29) plus the languages the CDN is
// known to serve. Adding a row is cheap and safe; the cost of a missing one is a
// decline, which is the honest outcome.
var originLanguages = map[string]store.Lang{
	"French": "fr", "Italian": "it", "German": "de", "Spanish": "es",
	"Japanese": "ja", "Dutch": "nl", "Portuguese": "pt", "Russian": "ru",
	"Swedish": "sv", "Norwegian": "no", "Danish": "da", "Polish": "pl",
	"Turkish": "tr", "Arabic": "ar", "Hebrew": "he", "Hindi": "hi",
	"Chinese": "zh", "Korean": "ko", "Persian": "fa", "Czech": "cs",
	"Finnish": "fi", "Hungarian": "hu",
}

// historicalStages are masked out of an ORIGIN before any modern language is
// looked for.
//
// Excluded BY CATEGORY, not because the CDN 404s them. `Latin` has the code
// `la`, `Old English` has `ang`, `Sanskrit` has `sa` — so "no recording exists"
// would be right by accident and would silently start playing something the day
// Google added Latin. They are excluded because a superseded stage of a language
// is not something a speaker says today: the thing /pron offers does not apply.
//
// THE ORDER MATTERS, and it is the most common case rather than an edge one.
// Measured over 300 entries: `Old French` occurs 11 times against `French`'s 8,
// so searching for modern names first would call the majority case French.
//
// `Greek` is here despite `el` existing, because NOAD's bare "Greek" means
// ANCIENT Greek — it is the second most frequent language word in the survey,
// behind Latin. NOAD writes "modern Greek" for the living language; that is left
// unhandled rather than special-cased, because it did not occur in the survey
// and a rule written for a case nobody has seen is a guess.
//
// `Germanic` is a FAMILY, not a language, and it is masked to say so — but NOT
// because the search would otherwise match it. That claim was written here and
// measured false: removing `Germanic` from this list leaves every fixture green,
// including `run` ("of Germanic origin, probably reinforced in Middle English by
// Old Norse"), which carries no cognate marker and would be the case to fail.
// What actually protects `run` is the WORD BOUNDARY in the search below —
// `\bGerman\b` does not match inside "Germanic" — and that is pinned by its own
// case in TestOriginLanguageRules rather than left to this comment.
//
// The mask stays because a family name has no business being read as a source
// language even if the matching rule later changes, and because it documents the
// distinction. It is redundancy, and it is labelled as redundancy.
var historicalStages = []string{
	"Old English", "Middle English", "Old French", "Anglo-Norman French",
	"Old Norse", "Middle Dutch", "Middle Low German", "Old High German",
	"Old Saxon", "Latin", "Greek", "Sanskrit", "Germanic", "Scots",
	"Old Irish", "Old Provençal", "Frankish",
}

// cognateMarkers open a clause that names related words rather than sources.
//
// This is the distinction the whole function turns on. NOAD names languages in
// two roles and only one is a source:
//
//	bring   Old English bringan, of Germanic origin; related to Dutch brengen
//	        and German bringen.
//
// `bring` is an Old English word with NO source language; Dutch and German are
// COGNATES, words sharing an ancestor. Searching the whole section infers German
// for it — which is exactly #29's D1 failure, arriving by a different road.
var cognateMarkers = []string{
	"related to", "Related to", "compare with", "Compare with",
	"cognate with", "shared by", "compare ", "Compare ",
}

// ErrNoOriginLanguage means the entry names no modern source language. A normal
// outcome, not a malfunction: most English words are English.
var ErrNoOriginLanguage = fmt.Errorf("no source language named")

// OriginLanguage reads an entry's ORIGIN and reports the language a borrowing
// came from, plus the NAME as NOAD wrote it.
//
// The name is returned as well as the code because /pron reports what it READ —
// "ORIGIN says French" — not what it derived. A silent inference cannot be
// audited, and this repo's rule is that a record has to be true.
//
// FIRST-NAMED WINS, and that is not a tiebreak. NOAD's convention is that the
// first source named is the immediate one: `casque` is "from French, from
// Spanish casco" — English took it from French, which took it from Spanish — and
// French is what a speaker says. Same for `ballet` and `mesa`. So there is no
// ambiguity case to resolve once cognates are cut, and the one genuinely
// contested entry (`piano`: "either from French, or …") is answered with the
// first-named and REPORTED, which is what makes the hedge visible.
//
// Order: cut cognates, mask stages, then search. Each step is measured; see the
// table comments above.
func OriginLanguage(e Entry) (store.Lang, string, error) {
	text := ""
	for _, s := range e.Sections {
		if s.Name == "ORIGIN" {
			text = s.Text
			break
		}
	}
	if strings.TrimSpace(text) == "" {
		return "", "", fmt.Errorf("%w: this entry has no ORIGIN", ErrNoOriginLanguage)
	}
	for _, marker := range cognateMarkers {
		if i := strings.Index(text, marker); i >= 0 {
			text = text[:i]
		}
	}
	for _, stage := range historicalStages {
		text = strings.ReplaceAll(text, stage, " ")
	}

	best, bestName := -1, ""
	for name := range originLanguages {
		// Word-boundaried: "German" must not match inside a longer word, and a
		// language name embedded in a proper noun is not a source.
		loc := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).FindStringIndex(text)
		if loc == nil {
			continue
		}
		if best < 0 || loc[0] < best {
			best, bestName = loc[0], name
		}
	}
	if bestName == "" {
		return "", "", fmt.Errorf("%w: its ORIGIN names only historical stages or cognates", ErrNoOriginLanguage)
	}
	return originLanguages[bestName], bestName, nil
}

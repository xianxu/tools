package main

import (
	"fmt"
	"regexp"
	"slices"
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

// Historical stages are masked out of an ORIGIN before any modern language is
// looked for.
//
// Excluded BY CATEGORY, not because the CDN 404s them. `Latin` has the code
// `la`, `Old English` has `ang`, `Sanskrit` has `sa` — so "no recording exists"
// would be right by accident and would silently start playing something the day
// Google added Latin. They are excluded because a superseded stage of a language
// is not something a speaker says today: the thing /pron offers does not apply.
//
// THE ORDER MATTERS, and it is the common case rather than an edge one. Measured
// over 300 entries: `Old French` occurs 11 times against `French`'s 8, so
// searching for modern names first would call the majority case French.
//
// `Greek` is masked despite `el` existing, because NOAD's bare "Greek" means
// ANCIENT Greek — the second most frequent language word in the survey, behind
// Latin. NOAD writes "modern Greek" for the living language; that is left
// unhandled rather than special-cased, because it did not occur in the survey and
// a rule written for a case nobody has seen is a guess.
//
// `Germanic` is a FAMILY, not a language, and is masked to say so — but NOT
// because the search would otherwise match it. That claim was written here once
// and measured false: `\bGerman\b` does not match inside "Germanic", and the WORD
// BOUNDARY is what protects `run`. It is redundancy, labelled as such, and
// TestOriginLanguageRules pins the boundary with a case this mask cannot save.
var stagePrefixes = []string{
	"Old", "Middle", "Old High", "Middle High", "Middle Low", "Low", "Early",
	"Anglo-Norman", "Anglo-",
}

// stagesWithNoModernMember carries the stages whose language has no row in
// originLanguages, so no prefix rule can generate them.
var stagesWithNoModernMember = []string{
	"Latin", "Greek", "Sanskrit", "Old Norse", "Frankish", "Germanic", "Scots",
	"Old Irish", "Old Provençal", "Old Saxon", "Old Church Slavonic",
}

// historicalStages is DERIVED from originLanguages rather than enumerated.
//
// Enumerating instances was the first shape and it leaked, measured at the close
// review: `Old French`, `Old English` and `Middle Dutch` were masked while `Old
// Italian`, `Middle French`, `Old Spanish` and `Low German` were not — each
// inferring a modern recording for an explicitly superseded stage, and each
// printing "ORIGIN says Italian" when ORIGIN said *Old* Italian, so the record
// was untrue as well. D4 promises a CATEGORY; a hand list is instances of one.
//
// So every mapped language generates its own stages, and the hand list carries
// only those whose language has no modern member to generate from.
var historicalStages = func() []string {
	out := append([]string(nil), stagesWithNoModernMember...)
	for name := range originLanguages {
		for _, p := range stagePrefixes {
			out = append(out, p+" "+name)
		}
	}
	// LONGEST FIRST, so "Old High German" is masked whole rather than "Old
	// German" eating its head and leaving "High German" behind.
	slices.SortFunc(out, func(a, b string) int { return len(b) - len(a) })
	return out
}()

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
	// Whether the RAW section mentions any mapped language at all decides which
	// of two declines the user gets, and they are genuinely different facts:
	// "gaslighting" reads "1960s: see gaslight (verb)" and names no language,
	// while "read" names Dutch and German as cognates. Telling the first user
	// their entry names only stages or cognates is a record that is not true.
	mentionsAny := anyLanguageIn(text)
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
		if !mentionsAny {
			return "", "", fmt.Errorf("%w: its ORIGIN names no language", ErrNoOriginLanguage)
		}
		return "", "", fmt.Errorf("%w: its ORIGIN names languages only as historical stages or cognates", ErrNoOriginLanguage)
	}
	return originLanguages[bestName], bestName, nil
}

// anyLanguageIn reports whether the text mentions a mapped language ANYWHERE,
// before cognate clauses are cut or stages masked.
//
// It answers "was there a language to reject", which is what separates the two
// declines. Same word-boundary rule as the search proper, so the two cannot
// disagree about what counts as a mention.
func anyLanguageIn(text string) bool {
	for name := range originLanguages {
		if regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).MatchString(text) {
			return true
		}
	}
	return false
}

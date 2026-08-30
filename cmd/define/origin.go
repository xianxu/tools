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
	mentions, mentionsAny := OriginLanguageMentions(e)
	if mentions == nil && !mentionsAny {
		if originText(e) == "" {
			return "", "", fmt.Errorf("%w: this entry has no ORIGIN", ErrNoOriginLanguage)
		}
		return "", "", fmt.Errorf("%w: its ORIGIN names no language", ErrNoOriginLanguage)
	}
	if len(mentions) == 0 {
		return "", "", fmt.Errorf("%w: its ORIGIN names languages only as historical stages or cognates", ErrNoOriginLanguage)
	}
	// THE FIRST of them, which is the whole of the difference between this and
	// its producer — see the comment above.
	return mentions[0].Lang, mentions[0].Name, nil
}

// Mention is one modern language named as a SOURCE in an ORIGIN section, with
// where it sits in that section's text.
//
// Offset indexes `sec.Text` — the raw section, exactly as `Render` receives it —
// which is what makes it addressable on screen. That is load-bearing and was
// nearly wrong: the stage mask used to be `strings.ReplaceAll(text, stage, " ")`,
// which CHANGES LENGTH, so every offset after a masked stage pointed at the
// wrong column. Measured on the `concrete` fixture: a 13-character shift before
// "French". The mask preserves length now, and `TestOriginMentionOffsetsIndexTheSourceText`
// is what keeps it that way.
type Mention struct {
	Name   string
	Lang   store.Lang
	Offset int // bytes into the ORIGIN section's text
}

// OriginLanguageMentions reports EVERY modern language an ORIGIN names as a
// source, in the order they appear, plus whether the raw section mentioned any
// mapped language at all.
//
// Every one, not the first, and that is the whole reason this exists beside
// OriginLanguage. `/pron` with no argument wants the first — NOAD's convention
// is that the first source named is the immediate one. A CLICK wants them all:
// `piano` is "either from French, or … Italian", and the insight #30 is founded
// on is that a click has nothing to disambiguate, because the user points at the
// one they meant. Two consumers, one cut-and-mask — rather than a second
// spelling of a rule #35's close review spent four findings getting right.
//
// The second return value answers "was there a language to reject", which is
// what separates OriginLanguage's two declines: "gaslighting" reads "1960s: see
// gaslight (verb)" and names none, while "read" names Dutch and German as
// cognates. Telling the first user their entry names only cognates is a record
// that is not true.
func OriginLanguageMentions(e Entry) ([]Mention, bool) {
	text := originText(e)
	if text == "" {
		return nil, false
	}
	mentionsAny := anyLanguageIn(text)

	// Cognate clauses are CUT, which keeps every surviving offset valid because
	// what survives is a prefix. Stages are MASKED IN PLACE for the same reason.
	searchable := text
	for _, marker := range cognateMarkers {
		if i := strings.Index(searchable, marker); i >= 0 {
			searchable = searchable[:i]
		}
	}
	for _, stage := range historicalStages {
		searchable = maskOut(searchable, stage)
	}

	var out []Mention
	for name, lang := range originLanguages {
		// Word-boundaried: "German" must not match inside a longer word, and a
		// language name embedded in a proper noun is not a source.
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
		for _, loc := range re.FindAllStringIndex(searchable, -1) {
			out = append(out, Mention{Name: name, Lang: lang, Offset: loc[0]})
		}
	}
	// SOURCE ORDER. The map is iterated in Go's random order, and both consumers
	// depend on position: OriginLanguage takes the first, and a reader clicks the
	// one they can see.
	slices.SortFunc(out, func(a, b Mention) int { return a.Offset - b.Offset })
	return out, mentionsAny
}

// originText is the ORIGIN section's text, or "" when the entry has none.
func originText(e Entry) string {
	for _, s := range e.Sections {
		if s.Name == "ORIGIN" && strings.TrimSpace(s.Text) != "" {
			return s.Text
		}
	}
	return ""
}

// maskOut blanks every occurrence of a stage name, PRESERVING LENGTH so the
// offsets of everything after it stay true.
//
// Spaces rather than deletion, and it is the difference between a region drawn
// on the right word and one drawn 13 columns to its left.
func maskOut(text, stage string) string {
	var b strings.Builder
	for {
		i := strings.Index(text, stage)
		if i < 0 {
			b.WriteString(text)
			return b.String()
		}
		b.WriteString(text[:i])
		b.WriteString(strings.Repeat(" ", len(stage)))
		text = text[i+len(stage):]
	}
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

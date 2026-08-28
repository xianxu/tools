package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
)

// langPair is one entry from DCSDictionaryGetLanguages: what the HEADWORDS are
// (index) and what the DEFINITIONS are (description).
//
// The pair is the whole point. A bilingual dictionary is not a dictionary with
// two languages, it is one whose index and description differ — Gran Diccionario
// Oxford indexes es->es AND en->es, and it is that second pair which makes it
// bilingual rather than a Spanish dictionary that also knows English.
type langPair struct {
	Index       store.Lang
	Description store.Lang
}

// dictMeta is one installed dictionary as the private DictionaryServices surface
// describes it: a stable reverse-DNS identifier plus its language pairs.
//
// A plain struct with no CoreServices in it, so chooseDictionary below is a pure
// function over data and is unit-testable on any machine — including one with no
// dictionaries installed at all (ARCH-PURE).
type dictMeta struct {
	ID    string
	Langs []langPair
}

// monolingualIn reports whether EVERY language pair is L->L.
//
// Every, not any: Gran Diccionario Oxford has an es->es pair, so "any" would
// accept it as monolingual Spanish and the learner would get English glosses in
// a Spanish session — the exact outcome the mode exists to prevent.
func (m dictMeta) monolingualIn(l store.Lang) bool {
	if len(m.Langs) == 0 {
		return false
	}
	for _, p := range m.Langs {
		if p.Index != l || p.Description != l {
			return false
		}
	}
	return true
}

// curated names the dictionary to prefer for a language, and it is honestly a
// CURATED LIST rather than a rule.
//
// Measured 2026-08-28: requiring L->L leaves exactly one candidate for `es` but
// SIX for `en` — NOAD, ODE, Apple Dictionary, two THESAURUSES (OAWT, OTE) and an
// accessibility dictionary. Nothing in the metadata distinguishes a
// general-purpose dictionary from a thesaurus, so no rule over the metadata can
// prefer NOAD. Proof that it matters: a deterministic smallest-identifier
// tiebreak picks the accessibility dictionary for English and the BILINGUAL
// Oxford for Spanish. Deterministic and wrong is still wrong.
//
// A short list, easy to extend, and deliberately not a fallback ordering: on a
// machine whose installed set nobody has curated, chooseDictionary reports "not
// found" and the caller degrades to today's behaviour rather than guessing.
// A LIST per language, in preference order, because "the English dictionary" is
// not one book. NOAD answers ordinary words and is the one whose notation matches
// Google's; Apple Dictionary answers iPhone, iPad and MacBook, which NOAD simply
// does not have. Both index en->en, so neither can leak another language in.
//
// Selecting only NOAD was the first shape and it was a real regression: `define
// iPhone` went from an entry to "no dictionary entry". Selecting nothing — the
// pre-#23 NULL search over every ACTIVE dictionary — is the other failure, and a
// worse one: with Spanish dictionaries enabled, `madrugar` answers in ENGLISH
// mode, which is precisely the Done-when row this milestone exists to satisfy.
// An ordered list of same-language dictionaries is what satisfies both.
var curated = map[store.Lang][]string{
	"en": {"com.apple.dictionary.NOAD", "com.apple.dictionary.AppleDictionary"},
	"es": {"com.apple.dictionary.es.DGLEV"},
}

// chooseDictionary answers "which installed dictionaries serve language L".
//
// Metadata NARROWS, a curated default DECIDES. The two steps are separate
// because only the first is derivable: "indexes L, monolingually" is a fact the
// system reports, while "is a general dictionary rather than a thesaurus" is a
// judgement nothing in the metadata supports.
//
// The false return is load-bearing, not an error case. It means "nothing here is
// known-good for this language", and the caller's answer to that is today's NULL
// behaviour — never a different language's dictionary, which would put the
// flat-topped hill back in a Spanish session.
// Returns them in CURATED order, not installed order — the installed set is a
// CFSet with no order at all, so preference has to come from the list.
func chooseDictionary(installed []dictMeta, l store.Lang) ([]dictMeta, bool) {
	byID := make(map[string]dictMeta, len(installed))
	for _, m := range installed {
		byID[m.ID] = m
	}
	var out []dictMeta
	for _, want := range curated[l] {
		if m, ok := byID[want]; ok && m.monolingualIn(l) {
			out = append(out, m)
		}
	}
	return out, len(out) > 0
}

// everyActiveDictionary is what the NULL search is called when /lang reports it.
// Not an identifier, because it is not one dictionary — it is the host's whole
// active set, which is why results depend on Dictionary.app's configuration.
const everyActiveDictionary = "every active dictionary"

// dictionaryFor is the whole SELECTION POLICY, pure over metadata.
//
// Extracted from the cgo shell deliberately (ARCH-PURE). While it lived beside
// installedDictionaries() the two fallback branches — the private surface gone,
// and nothing curated for this language — could not be tested on any platform,
// and the Done-when row about degrading was ticked on a manual experiment
// instead. They are the branches that MATTER: a shell context that reports only
// one dictionary takes the second one on every single run.
//
// installed == nil means the private surface did not resolve. That is a
// different thing from an empty set, and both degrade the same way but for
// reasons worth telling apart in the warning.
//
// Returns the identifiers to search, a name for /lang to report, and whether a
// curated choice was made at all. No ids means "use the NULL search".
func dictionaryFor(installed []dictMeta, lang store.Lang) (ids []string, name string, complaint string) {
	if installed == nil {
		// LOUD, because it is the surprising one: the private surface moved
		// under us and the tool has silently become its pre-#23 self.
		return nil, everyActiveDictionary,
			"the dictionary-selection API is unavailable; searching every active dictionary"
	}
	chosen, ok := chooseDictionary(installed, lang)
	if !ok {
		return nil, everyActiveDictionary,
			fmt.Sprintf("no known %s dictionary is installed; searching every active dictionary", lang)
	}
	ids = make([]string, len(chosen))
	for i, m := range chosen {
		ids[i] = m.ID
	}
	// SILENT on the happy path, and the name is reported by /lang instead. A
	// line per lookup saying the expected thing happened is noise; a learner on
	// a machine with a different set installed asks the question once.
	return ids, strings.Join(ids, ", "), ""
}

// The statuses dcs_lookup_in reports. Mirrored in Go so the FOLD below is a
// platform-neutral decision rather than a branch inside cgo.
const (
	lookupFound      = 0
	lookupNoEntry    = 1
	lookupFailed     = 2
	lookupNoSuchDict = 3
)

// foldLookupError decides what a walk over several dictionaries reports.
//
// PURE, and extracted for a reason worth stating: while this lived inside the
// darwin-only file it had no test, and a fix for exactly this bug shipped
// INOPERATIVE — `case lookupNoEntry` overwrote the error unconditionally, so a
// vanished primary dictionary followed by an ordinary miss still reported
// "no entry". Untestable code is where a fix can look right and do nothing.
//
// The rule: an ABSENCE never overwrites a real failure. "This word is not
// Spanish" is a correct answer; "the Spanish dictionary is gone" means the
// caller cannot vouch for that absence, and the C side keeps the two statuses
// distinct precisely so this layer does not collapse them.
func foldLookupError(prev error, status int, id string) error {
	// Already carrying a real failure: nothing weaker replaces it.
	if prev != nil && !errors.Is(prev, ErrNoEntry) {
		return prev
	}
	switch status {
	case lookupNoEntry:
		return ErrNoEntry
	case lookupNoSuchDict:
		return fmt.Errorf("%w: dictionary %s is unavailable", ErrLookupFailed, id)
	default:
		return ErrLookupFailed
	}
}

// parseDictRecords reads the flat "id\tindex>desc,index>desc,\n" encoding the
// cgo boundary emits.
//
// Platform-neutral on purpose: these three parsers claimed to be "testable
// without CoreServices" while sitting behind //go:build darwin, which made
// GOOS=linux go vet fail on their own test. A pure function that only compiles
// on one platform is not pure enough to be worth the claim.
func parseDictRecords(s string) []dictMeta {
	out := []dictMeta{}
	for _, line := range strings.Split(s, "\n") {
		id, langs, ok := strings.Cut(line, "\t")
		if !ok || id == "" {
			continue
		}
		out = append(out, dictMeta{ID: id, Langs: parseLangPairs(langs)})
	}
	return out
}

// parseLangPairs reads the flat "index>description," encoding.
//
// This is the one place a malformed pair could silently make a bilingual
// dictionary look monolingual, which is why it is separate and tested.
func parseLangPairs(s string) []langPair {
	var out []langPair
	for _, field := range strings.Split(s, ",") {
		idx, desc, ok := strings.Cut(field, ">")
		if !ok {
			continue
		}
		// Apple writes both "en" and "en_US"; the region is not a language.
		i, err := store.ParseLang(baseLang(idx))
		if err != nil {
			continue
		}
		d, err := store.ParseLang(baseLang(desc))
		if err != nil {
			continue
		}
		out = append(out, langPair{Index: i, Description: d})
	}
	return out
}

func baseLang(s string) string {
	if i := strings.IndexAny(s, "_-"); i >= 0 {
		return s[:i]
	}
	return s
}

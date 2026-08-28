package main

import "github.com/xianxu/tools/cmd/define/store"

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

package main

import (
	"strings"
	"unicode"
)

// questionOpeners are the words that make a line read as a question when they
// lead it. Contractions are matched on the stem ("what's" → what), and negated
// auxiliaries on the stem before "n't" ("isn't" → is), so the set stays the
// vocabulary rather than its inflections.
//
// A package-level slice rather than an inline switch because #18 widens it with
// Spanish (qué, cómo, por qué), and a switch buried in a function is a thing
// that has to be found first.
var questionOpeners = []string{
	"what", "why", "how", "when", "where", "which", "who", "whom", "whose",
	"is", "are", "was", "were", "am", "do", "does", "did", "have", "has", "had",
	"can", "could", "should", "would", "will", "shall", "may", "might", "must",
}

// requestVerbs open an imperative that is still a question to this tool: a
// follow-up ("give me three more examples") carries no wh-word and no question
// mark. They need a second word — bare "use", "give" and "list" are headwords,
// and NOAD answers them before this function is ever consulted.
var requestVerbs = []string{
	"give", "show", "tell", "explain", "compare", "contrast", "use", "make",
	"write", "list", "translate", "rewrite", "define",
}

// readsAsQuestion is the semantic half of the console's one decision table. The
// syntactic half is parseREPLLine; between them they answer "what did the user
// mean by this line", and neither loop is allowed a third opinion.
//
// It is only ever asked about a line NOAD has already missed. That containment
// is the safety argument: "hot dog", "a priori" and "use" are lookups because
// the dictionary said so, not because this function was careful.
//
// There is deliberately NO length arm. Word count is the signal the spec
// rejects — "hot dog" is two words and "defenestrate" is one — and a line being
// long is not the same fact as it reading as a question. The cost is named
// rather than hidden: "difference between sycophantic and obsequious" reads as
// neither interrogative nor imperative and answers "not found", with "?" as its
// recovery.
func readsAsQuestion(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	// A mark with no word in front of it is punctuation, not a question.
	if strings.HasSuffix(line, "?") && strings.IndexFunc(line, unicode.IsLetter) >= 0 {
		return true
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		// One word is a headword shape. NOAD missing it means a typo, which is
		// what "not found" is for.
		return false
	}
	first := openerStem(fields[0])
	return contains(questionOpeners, first) || contains(requestVerbs, first)
}

// openerStem lowercases and strips the inflection a question opener carries:
// "What's" → what, "isn't" → is, "how," → how.
//
// The negation is cut as the whole unit "n't", before the apostrophe split and
// never as a bare trailing "n" — "when" is an opener whose last letter is the
// one a lazier rule would eat.
func openerStem(w string) string {
	w = strings.ToLower(strings.TrimFunc(w, func(r rune) bool {
		return !unicode.IsLetter(r) && r != '\''
	}))
	if s, ok := strings.CutSuffix(w, "n't"); ok {
		return s
	}
	if i := strings.Index(w, "'"); i > 0 {
		return w[:i]
	}
	return w
}

func contains(set []string, s string) bool {
	for _, v := range set {
		if v == s {
			return true
		}
	}
	return false
}

// maxQuestionRunes bounds a question quoted back in a diagnostic.
const maxQuestionRunes = 40

// truncateQuestion elides a question for a one-line message. Rune-based, not
// byte-based: cutting mid-rune prints a replacement character.
func truncateQuestion(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= maxQuestionRunes {
		return string(r)
	}
	return strings.TrimRight(string(r[:maxQuestionRunes]), " ") + "…"
}

package main

import (
	"io"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
)

// languageAnswer owns decoded prose, vocabulary highlighting and language
// boundaries. Changing ownership flushes the delayed vocabulary tail first.
//
// Owned text used to be rendered by highlightRegion — a one-shot over a complete
// string — because a passage arrived as one buffered lump. Since #72 it arrives
// a rune at a time, and a one-shot highlighter over a rune matches nothing: the
// deck word `obsequious` inside a Spanish passage simply stopped being green.
// So both paths now share the one streaming highlightWriter, which is what the
// atlas already claimed was true of definitions and answers.
type languageAnswer struct {
	plain     strings.Builder
	decoder   *languageDecoder
	highlight *highlightWriter
	wrap      *ownedAnswerWrapWriter
	policy    tintPolicy
	vocab     Vocabulary
	ownership store.Lang // the run the highlighter is writing into; "" is neutral
	err       error
}

func newLanguageAnswer(out io.Writer, width int, v Vocabulary, policy tintPolicy) *languageAnswer {
	a := &languageAnswer{wrap: newOwnedAnswerWrapWriter(out, width, policy), policy: policy, vocab: v}
	// runVocabulary(""), not the raw v: own() is the sole place the (ownership,
	// highlighter) pair is decided, and a constructor that restates the answer
	// bypasses it. They agree today only because the neutral run happens to keep
	// the whole vocabulary — change that rule (#64 plausibly will) and an answer
	// opening in neutral prose would silently keep the old one while an answer
	// opening inside a passage would not.
	a.highlight = newHighlightWriter(a, a.runVocabulary(""), knownOn)
	a.decoder = newLanguageDecoder(a.accept)
	return a
}
func (a *languageAnswer) accept(v languageText) {
	a.plain.WriteString(v.text)
	at := 0
	for _, sp := range v.spans {
		a.write(v.text[at:sp.start], "")
		a.write(v.text[sp.start:sp.end], sp.lang)
		at = sp.end
	}
	a.write(v.text[at:], "")
}

// write puts text into the run that owns it, switching runs first.
func (a *languageAnswer) write(s string, lang store.Lang) {
	if s == "" {
		return
	}
	a.own(lang)
	_, err := io.WriteString(a.highlight, s)
	a.remember(err)
}

// own switches the ownership run, and both halves of flush-then-replace carry
// weight.
//
// The FLUSH releases text the old run was still holding against a possible
// phrase, while a.ownership still names that run — held text belongs to the run
// it arrived in, not the one that displaced it. The REPLACE swaps the
// vocabulary, which is both how a Spanish `red` and an English `red` stay
// different words (#61) and why a phrase cannot span the boundary: a phrase half
// in another language is not a phrase. That is the invariant
// TestLanguageAnswerForeignHomographDoesNotUseTargetVocabulary exists for, and
// it is preserved here rather than re-derived.
func (a *languageAnswer) own(lang store.Lang) {
	if lang == a.ownership {
		return
	}
	a.remember(a.highlight.Flush())
	a.ownership = lang
	a.highlight = newHighlightWriter(a, a.runVocabulary(lang), knownOn)
}

// runVocabulary withholds the deck from text that is not in the session's
// language, so an ambiguous spelling cannot acquire target-language styling by
// coincidence. Neutral prose keeps the session's vocabulary: it makes no claim
// about being foreign.
//
// NOT named vocabularyFor: that is the package function ask.go:164 calls to
// decide what the whole answer highlights against, and two unrelated things of
// one name in one package is a reader's trap.
func (a *languageAnswer) runVocabulary(lang store.Lang) Vocabulary {
	if lang == "" {
		return a.vocab
	}
	target := a.policy.lang
	if target == "" {
		target = store.DefaultLang
	}
	if normalized, err := store.ParseLang(string(target)); err != nil || normalized != lang {
		return nil
	}
	return a.vocab
}

func (a *languageAnswer) remember(err error) {
	if a.err == nil {
		a.err = err
	}
}

// Write is the vocabulary writer's downstream sink, not the model input. It
// tags what the highlighter emits with the run that is currently open.
func (a *languageAnswer) Write(p []byte) (int, error) {
	if a.err != nil {
		return 0, a.err
	}
	err := a.wrap.WriteOwned(string(p), string(a.ownership))
	a.remember(err)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
func (a *languageAnswer) Finish() error {
	a.decoder.Finish()
	a.remember(a.highlight.Flush())
	a.remember(a.wrap.Flush())
	return a.err
}

package main

import (
	"io"
	"strings"

	"github.com/xianxu/tools/cmd/define/store"
)

// languageAnswer owns decoded prose, vocabulary highlighting and language
// boundaries. Changing ownership flushes the delayed vocabulary tail first.
type languageAnswer struct {
	plain     strings.Builder
	decoder   *languageDecoder
	highlight *highlightWriter
	wrap      *ownedAnswerWrapWriter
	policy    tintPolicy
	vocab     Vocabulary
	err       error
}

func newLanguageAnswer(out io.Writer, width int, v Vocabulary, policy tintPolicy) *languageAnswer {
	a := &languageAnswer{wrap: newOwnedAnswerWrapWriter(out, width, policy), policy: policy, vocab: v}
	a.highlight = newHighlightWriter(a, v, knownOn)
	a.decoder = newLanguageDecoder(a.accept)
	return a
}
func (a *languageAnswer) accept(v languageText) {
	a.plain.WriteString(v.text)
	at := 0
	for _, sp := range v.spans {
		a.neutral(v.text[at:sp.start])
		a.remember(a.highlight.Flush())
		vocabulary := a.vocab
		target := a.policy.lang
		if target == "" {
			target = store.DefaultLang
		}
		if normalized, err := store.ParseLang(string(target)); err != nil || normalized != sp.lang {
			vocabulary = nil
		}
		rendered := highlightRegion(v.text[sp.start:sp.end], vocabulary, knownOn, "")
		if a.err == nil {
			a.remember(a.wrap.WriteOwned(rendered, string(sp.lang)))
		}
		at = sp.end
	}
	a.neutral(v.text[at:])
}
func (a *languageAnswer) neutral(s string) {
	if s != "" {
		_, err := io.WriteString(a.highlight, s)
		a.remember(err)
	}
}

func (a *languageAnswer) remember(err error) {
	if a.err == nil {
		a.err = err
	}
}

// Write is the vocabulary writer's downstream sink, not the model input.
func (a *languageAnswer) Write(p []byte) (int, error) {
	if a.err != nil {
		return 0, a.err
	}
	err := a.wrap.WriteOwned(string(p), "")
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

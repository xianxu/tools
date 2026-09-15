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
	wrap      *answerWrapWriter
	policy    tintPolicy
	vocab     Vocabulary
	err       error
}

func newLanguageAnswer(out io.Writer, width int, v Vocabulary, policy tintPolicy) *languageAnswer {
	a := &languageAnswer{wrap: newAnswerWrapWriter(out, width), policy: policy, vocab: v}
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
		owned := languageText{text: rendered, spans: []languageSpan{{start: 0, end: len(rendered), lang: sp.lang}}}
		if a.err == nil {
			_, err := io.WriteString(a.wrap, styleLanguageText(owned, a.policy))
			a.remember(err)
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
	_, err := a.wrap.Write(p)
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

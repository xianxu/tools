package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
)

type practicePresenter interface {
	PromptPresentation() play.Presentation
	RevealPresentation() play.Presentation
}

// Vocabulary and language styling share the form's emitted boundaries. Neutral
// fragments include pre-rendered dictionary entries, which must stay untouched.
func renderPracticePresentation(p play.Presentation, lang, source store.Lang, policy tintPolicy, vocab Vocabulary, sf surface, subject string) string {
	var out strings.Builder
	at := 0
	for _, s := range p.Spans {
		out.WriteString(p.Text[at:s.Start])
		text := p.Text[s.Start:s.End]
		owned := store.Lang("")
		switch s.Role {
		case play.Target:
			owned = lang
		case play.English:
			owned = "en"
		case play.DictionarySource:
			owned = source
		}
		if (s.Role == play.Target || s.Role == play.DictionarySource) && !s.AnswerStyled && vocab != nil && sf.admitsColour() {
			text = highlightRegion(text, withoutWord(vocab, subject), knownOn, "")
		}
		if !s.AnswerStyled && owned != "" {
			text = styleLanguageText(languageText{text: text, spans: []languageSpan{{start: 0, end: len(text), lang: owned}}}, policy)
		}
		out.WriteString(text)
		at = s.End
	}
	out.WriteString(p.Text[at:])
	return out.String()
}

func writePracticePresentation(w io.Writer, p play.Presentation, rs []Region, d deps, opt options, sf surface, subject, already string) {
	text := "\n" + p.Text + "\n"
	v := deckVocabulary(d)
	// English help keeps its existing exclusion from target-deck actions.
	englishLines := map[int]bool{}
	for _, s := range p.Spans {
		if s.Role == play.English {
			line := 1 + strings.Count(p.Text[:s.Start], "\n")
			for n := 0; n <= strings.Count(p.Text[s.Start:s.End], "\n"); n++ {
				englishLines[line+n] = true
			}
		}
	}
	var own []Region
	for _, r := range wordRegionsOutside(text, already, v) {
		if !englishLines[r.Line] {
			own = append(own, r)
		}
	}
	if !opt.color {
		v = nil
	}
	styled := renderPracticePresentation(p, d.lang, dictionarySourceLanguage(d.dict), opt.tintFor(d.lang), v, sf, subject)
	writeRendered(w, "\n"+styled+"\n", mergeRegions(rs, own))
}

// practiceBuilder records generated chrome beside its numbers and key glyphs.
type practiceBuilder struct{ play.Presentation }

func (b *practiceBuilder) neutral(s string) { b.Text += s }
func (b *practiceBuilder) english(s string) {
	start := len(b.Text)
	b.Text += s
	b.Spans = append(b.Spans, play.LanguageSpan{Start: start, End: len(b.Text), Role: play.English})
}
func (b *practiceBuilder) append(p play.Presentation) {
	off := len(b.Text)
	b.Text += p.Text
	for _, s := range p.Spans {
		s.Start += off
		s.End += off
		b.Spans = append(b.Spans, s)
	}
}
func reservedPresentation(q play.Question) play.Presentation {
	var p practiceBuilder
	if _, ok := q.(play.Batch); !ok {
		p.neutral("d = ")
		p.english("remove from deck")
		p.neutral(", ")
	}
	p.neutral("Ctrl-C ")
	p.english("to stop")
	return p.Presentation
}
func gradePromptPresentation(q play.Question) play.Presentation {
	var p practiceBuilder
	if k, ok := q.(interface{ KeysPresentation() play.Presentation }); ok {
		p.append(k.KeysPresentation())
	} else {
		p.neutral(q.Keys())
	}
	p.neutral(", ")
	p.append(reservedPresentation(q))
	return p.Presentation
}
func livePromptPresentation(s play.Session) play.Presentation {
	q := s.Current()
	if q == nil {
		return play.Presentation{}
	}
	if !s.Graded {
		return gradePromptPresentation(q)
	}
	var p practiceBuilder
	p.english("any key")
	p.neutral(" = ")
	p.english("next word")
	p.neutral(", ")
	if play.CanFlag(q) {
		p.neutral(string(play.FlagKey) + " = ")
		p.english("bad question")
		p.neutral(", ")
	}
	p.append(reservedPresentation(q))
	return p.Presentation
}
func practiceChrome(p play.Presentation, d deps, opt options) string {
	return renderPracticePresentation(p, d.lang, dictionarySourceLanguage(d.dict), opt.tintFor(d.lang), nil, surfaceProse, "")
}

func styledBoardPrompt(q play.Question, whole bool, pal palette, d deps, opt options) string {
	p := gradePromptPresentation(q)
	if !whole {
		var b practiceBuilder
		b.english("window too short for the whole board — mark what you see")
		b.neutral(", Ctrl-C ")
		b.english("to stop")
		p = b.Presentation
	}
	text := renderPracticePresentation(p, d.lang, dictionarySourceLanguage(d.dict), opt.tintFor(d.lang), nil, surfaceProse, "")
	return highlightBoardPrompt(text, true, pal)
}
func finishStyled(w io.Writer, s play.Session, fig sittingFigures, d deps, opt options) int {
	var p practiceBuilder
	p.neutral("\n" + fmt.Sprint(s.Right) + " ")
	p.english("right")
	p.neutral(", " + fmt.Sprint(s.Wrong) + " ")
	p.english("wrong")
	p.neutral("\n")
	p.append(costPresentation(fig))
	p.neutral("\n")
	io.WriteString(w, renderPracticePresentation(p.Presentation, d.lang, dictionarySourceLanguage(d.dict), opt.tintFor(d.lang), nil, surfaceProse, ""))
	return 0
}

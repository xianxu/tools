package main

import (
	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
	"strings"
	"unicode/utf8"
)

// Practice owns its prose and answer exclusions; dictionary rows arrive with
// their frozen section paint and bypass mixed-language row classification.
func renderPracticeOutput(p play.Presentation, lang, source store.Lang, policy tintPolicy, vocab Vocabulary, sf surface, subject string, width int) renderedOutput {
	var text strings.Builder
	var spans []languageSpan
	var exclusions []languageSpan
	at := 0
	for _, sp := range p.Spans {
		if sp.Start < at || sp.End < sp.Start || sp.End > len(p.Text) {
			return renderedOutput{text: p.Text}
		}
		text.WriteString(p.Text[at:sp.Start])
		fragment := p.Text[sp.Start:sp.End]
		owner := store.Lang("")
		switch sp.Role {
		case play.Target:
			owner = lang
		case play.DictionarySource:
			owner = source
		case play.English:
			owner = "en"
		case play.Decoration:
			owner = "decoration"
		}
		if (sp.Role == play.Target || sp.Role == play.DictionarySource) && !sp.AnswerStyled && vocab != nil && sf.admitsColour() {
			fragment = highlightRegion(fragment, withoutWord(vocab, subject), knownOn, "")
		}
		start := text.Len()
		text.WriteString(fragment)
		if owner != "" {
			spans = append(spans, languageSpan{start, text.Len(), owner})
		}
		if sp.AnswerStyled {
			exclusions = append(exclusions, languageSpan{start, text.Len(), "excluded"})
		}
		at = sp.End
	}
	text.WriteString(p.Text[at:])
	styled := text.String()
	wrapped := strings.Join(outputWrappedRows(styled, width), "\n")
	owned := projectDisplayText(languageText{text: styled, spans: spans}, wrapped)
	protected := projectDisplayText(languageText{text: styled, spans: exclusions}, wrapped)
	o := renderedOutput{text: wrapped}
	pos := 0
	ownerIndex, exclusionIndex := 0, 0
	for _, line := range strings.Split(wrapped, "\n") {
		var state rowOwnership
		paint := rowPaint{}
		col := 0
		for i := 0; i < len(line); {
			if n := escapeLen(line[i:]); n > 0 {
				i += n
				continue
			}
			n, w := nextDisplayUnit(line[i:])
			r, _ := utf8.DecodeRuneInString(line[i:])
			owner := ""
			for ownerIndex < len(owned.spans) && owned.spans[ownerIndex].end <= pos+i {
				ownerIndex++
			}
			if ownerIndex < len(owned.spans) {
				sp := owned.spans[ownerIndex]
				if sp.start <= pos+i && sp.end >= pos+i+n {
					owner = string(sp.lang)
				}
			}

			// Whitespace, numbering and punctuation are structural decorations.
			state = advanceRowOwnership(state, rowOwnershipEvent{lang: owner, substantive: owner != "decoration" && isLanguageProse(r, owner != "")})
			for exclusionIndex < len(protected.spans) && protected.spans[exclusionIndex].end <= pos+i {
				exclusionIndex++
			}
			if exclusionIndex < len(protected.spans) {
				sp := protected.spans[exclusionIndex]
				if sp.start <= pos+i && sp.end >= pos+i+n {
					for c := col; c < col+w; c++ {
						paint.exclusions = appendCellRange(paint.exclusions, c)
					}
				}
			}

			i += n
			col += w
		}
		if !state.mixed && state.lang != "" && normalizedLang(store.Lang(state.lang)) == normalizedLang(policy.lang) {
			paint.background = policy.background
		}
		o.rows = append(o.rows, paint)
		pos += len(line) + 1
	}
	// Section offsets are measured on the producer's unpadded source. Highlighting
	// changes only ANSI, so newline row positions remain stable before wrapping.
	sectionBase := renderedOutput{text: styled}
	sectionBase.rows = make([]rowPaint, strings.Count(styled, "\n")+1)
	sectionLines := make(map[int]bool)
	for _, region := range p.Regions {
		if region.Start < 0 || region.End < region.Start || region.End > len(p.Text) {
			continue
		}
		first := strings.Count(p.Text[:region.Start], "\n")
		last := first + strings.Count(p.Text[region.Start:region.End], "\n")
		if region.End > region.Start && p.Text[region.End-1] == '\n' {
			last--
		}
		for row := first; row <= last && row < len(sectionBase.rows); row++ {
			sectionBase.rows[row].background = region.Background
			sectionLines[row] = true
		}
	}
	if len(sectionLines) > 0 {
		mapped := layoutOutput(sectionBase, width)
		// A neutral section needs an explicit bypass as well as tinted sections.
		row := 0
		for i, line := range strings.Split(styled, "\n") {
			count := len(outputWrappedRows(line, width))
			for j := 0; j < count; j++ {
				if sectionLines[i] && row+j < len(o.rows) {
					o.rows[row+j] = paintAt(mapped.rows, row+j)
				}
			}
			row += count
		}
	}
	return o
}

func definitionPresentationRegions(o renderedOutput) []play.PresentationRegion {
	var rs []play.PresentationRegion
	at := 0
	for i, line := range strings.SplitAfter(o.text, "\n") {
		if line != "" {
			rs = append(rs, play.PresentationRegion{Start: at, End: at + len(line), Background: paintAt(o.rows, i).background})
		}
		at += len(line)
	}
	return rs
}

func practiceChromeOutput(p play.Presentation, d deps, opt options, pal palette) renderedOutput {
	o := renderPracticeOutput(p, d.lang, dictionarySourceLanguage(d.dict), opt.tintFor(d.lang), nil, surfaceProse, "", 0)
	o.text = asChrome(o.text, pal)
	return o
}
func boardFooterOutput(q play.Question, fig sittingFigures, pal palette, d deps, opt options) renderedOutput {
	o := renderedOutput{text: q.Prompt()}
	if p, ok := q.(practicePresenter); ok {
		o = renderPracticeOutput(p.PromptPresentation(), d.lang, dictionarySourceLanguage(d.dict), opt.tintFor(d.lang), nil, surfaceOf(q.Form()), q.Word(), 0)
	}
	rows := strings.Count(o.text, "\n") + 1
	for len(o.rows) < rows {
		o.rows = append(o.rows, rowPaint{})
	}
	bar := practiceChromeOutput(sittingBarPresentation(fig), d, opt, pal)
	o.text += "\n" + bar.text
	o.rows = append(o.rows, bar.rows...)
	return o
}
func boardPromptOutput(q play.Question, whole bool, pal palette, d deps, opt options) renderedOutput {
	p := gradePromptPresentation(q)
	if !whole {
		var b practiceBuilder
		b.english("window too short for the whole board — mark what you see")
		b.neutral(", Ctrl-C ")
		b.english("to stop")
		p = b.Presentation
	}
	o := practiceChromeOutput(p, d, opt, pal)
	o.text = highlightBoardPrompt(o.text, true, pal)
	return o
}

func drawPracticeOutput(view interface{ Draw(string, []string) }, prompt, footer renderedOutput) {
	if sink, ok := view.(interface {
		DrawOutput(renderedOutput, renderedOutput)
	}); ok {
		sink.DrawOutput(prompt, footer)
		return
	}
	view.Draw(prompt.text, strings.Split(footer.text, "\n"))
}

package main

import (
	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
	"strings"
	"unicode/utf8"
)

// Practice owns its prose and answer exclusions; dictionary rows arrive with
// their frozen section paint and bypass mixed-language row classification.
func renderPracticeOutput(p play.Presentation, lang, source store.Lang, policy tintPolicy, vocab Vocabulary, sf surface, subject string, width int, actions ...Region) renderedOutput {
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
	mappedActions := layoutOutput(renderedOutput{text: styled, regions: actions}, width)
	wrapped := mappedActions.text
	owned := projectDisplayText(languageText{text: styled, spans: spans}, wrapped)
	protected := projectDisplayText(languageText{text: styled, spans: exclusions}, wrapped)
	o := renderedOutput{text: wrapped, regions: mappedActions.regions}
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
			paint.tinted = policy.on
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
			sectionBase.rows[row].tinted = region.Tinted
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
			rs = append(rs, play.PresentationRegion{Start: at, End: at + len(line), Tinted: paintAt(o.rows, i).tinted})
		}
		at += len(line)
	}
	return rs
}

func practiceChromeOutput(p play.Presentation, d deps, opt options, pal palette) renderedOutput {
	o := renderPracticeOutput(p, d.lang, dictionarySourceLanguage(d.dict), tintFor(d, opt), nil, surfaceProse, "", 0)
	o.text = asChrome(o.text, pal)
	return o
}

// boardFooterOutput is the live edge for a board: everything the FORM draws, then the
// bar.
//
// One line of assembly, and that is the point. The grid and the panel are the
// board's own rendering — a form owns how it looks, and a loop composing it out
// of accessors would make the board's appearance a thing two files agree about,
// on the surface where disagreeing marks the wrong word. All this adds is the
// bar, which belongs to the sitting rather than to the question.
//
// THE FORM IS FIRST, which is load-bearing rather than aesthetic: formCell reads
// a footer entry index straight back as a grid row, so anything above it would
// silently shift every cell. It is also the order of value that fitFooter drops
// from: the bar goes first, then the panel, then grid rows.
//
// A BOARD CAN END UP IN A FOOTER THAT DROPS ROWS, and D15's "never" was measured
// wrong (R11). It holds at SELECTION — `packBoards` refuses a board the terminal
// cannot draw whole — and a resize afterwards is a shape nobody chose.
//
// What the order buys is that the losses are SURVIVABLE in sequence: the bar (a
// figure), the panel (cosmetic), then grid rows. An earlier version of this
// comment called them "harmless", which was checked against the CLICK map —
// FooterRowAt answers nothing for a row that was never painted — and was false
// of the SWEEP, which does not go through that map at all: Enter took every
// unmarked word including ones the window never drew. That is why Enter is now
// held while the board is not whole (R17), and why a safety word has to name the
// path it was checked on.
//
// The one thing that must not go is the statement of what a click will MEAN, and
// that is why the mode moved to the prompt row, which Paint clips last.
//
// THE PALETTE is threaded in rather than reached for, on the same seam
// `boardPalette` sits on: `main` owns the terminal's colours and the form takes
// finished sequences. It styles only the BAR — the grid above it is the board's
// own rendering, already painted through `play.Palette` (#44).
func boardFooterOutput(q play.Question, fig sittingFigures, pal palette, d deps, opt options) renderedOutput {
	o := renderedOutput{text: q.Prompt()}
	if p, ok := q.(practicePresenter); ok {
		o = renderPracticeOutput(p.PromptPresentation(), d.lang, dictionarySourceLanguage(d.dict), tintFor(d, opt), nil, surfaceOf(q.Form()), q.Word(), 0)
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

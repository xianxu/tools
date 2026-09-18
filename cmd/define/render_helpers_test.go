package main

import (
	"strings"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
)

// The string renderers below were production code no production path called
// (#70): every writer paints at its terminal boundary instead (serializeOutput,
// the screen). They live on as the tests' string view of that same path.

// Compatibility string renderers have no viewport. Paint their source width,
// in the scheme sc; production writers retain metadata until their actual
// terminal boundary.
func renderOutputText(o renderedOutput, sc store.Scheme) string {
	lines := outputStyledRows(o.text)
	for i, line := range lines {
		lines[i] = paintLanguageRow(line, paintAt(o.rows, i), visibleCells(line), sc)
	}
	return strings.Join(lines, "\n")
}

// holderFor is a holder already set to one scheme, for tests that paint a shade.
func holderFor(s store.Scheme) *schemeHolder {
	return newSchemeHolder(schemeState{}.withChoice(s, sourceFlag))
}

// renderDefinitions preserves per-section language ownership of word actions.
// Regions are complete here, so callers must not run an unscoped vocabulary
// pass over the composed bilingual string afterward.
func renderDefinitions(set definitionSet, opt RenderOpts) (string, []Region) {
	o := renderDefinitionOutput(set, opt)
	return renderOutputText(o, opt.Tint.scheme.Scheme()), o.regions
}

// Vocabulary and language styling share the form's emitted boundaries. Neutral
// fragments include pre-rendered dictionary entries, which must stay untouched.
func renderPracticePresentation(p play.Presentation, lang, source store.Lang, policy tintPolicy, vocab Vocabulary, sf surface, subject string) string {
	return renderOutputText(renderPracticeOutput(p, lang, source, policy, vocab, sf, subject, 0), policy.scheme.Scheme())
}

func practiceChrome(p play.Presentation, d deps, opt options) string {
	return renderPracticePresentation(p, d.lang, dictionarySourceLanguage(d.dict), tintFor(d, opt), nil, surfaceProse, "")
}

// boardFooter is the live edge for a board: everything the FORM draws, then the
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
func boardFooter(q play.Question, fig sittingFigures, pal palette, d deps, opt options) []string {
	text := q.Prompt()
	if p, ok := q.(practicePresenter); ok {
		text = renderPracticePresentation(p.PromptPresentation(), d.lang, dictionarySourceLanguage(d.dict), tintFor(d, opt), nil, surfaceOf(q.Form()), q.Word())
	}
	return append(strings.Split(text, "\n"), asChrome(practiceChrome(sittingBarPresentation(fig), d, opt), pal))
}

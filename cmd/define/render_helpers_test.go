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
	return newSchemeHolder(schemeState{}.withChoice(s, choiceFlag))
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

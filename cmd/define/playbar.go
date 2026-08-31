package main

import (
	"fmt"
	"strings"

	"github.com/xianxu/tools/cmd/define/play"
)

// What a sitting tells the learner about its own cost.
//
// ONE formatter for two surfaces — the pinned bar during the sitting and the
// summary after it — because they describe the same deck and the same `-count`
// assumption, and two spellings of one assumption is how they come to disagree.
// The same move doc_sync_test.go already made for the prompt lines.

// sittingFigures is everything both lines are built from.
//
// PLAIN NUMBERS, never a deck or a store: this file formats and does not
// compute, so it stays testable without any of it and the arithmetic stays in
// `schedule`, which is mechanically pure.
type sittingFigures struct {
	// load is reviews per day the deck costs in steady state.
	load float64
	// fresh is how many new words a day that leaves room for. Never negative —
	// schedule.SustainableNewWords floors at zero.
	fresh float64
	// budget is the sitting's -count, and it is NAMED in the output rather than
	// assumed: it is the daily budget for someone who sits down once a day, and
	// someone who sits twice needs to know to double it.
	budget int
	// done and total are the sitting's progress, for the live bar only.
	done, total int
}

// sittingBar is the pinned footer: progress on the left, cost on the right.
//
// ONE ROW, always. Paint charges a footer its real height and takes it out of
// the buffer's share, so a bar that wrapped to two rows would silently steal a
// line of the question — which is why the test asserts there is no newline in it
// rather than trusting the format string.
func sittingBar(f sittingFigures) string {
	return fmt.Sprintf("%d of %d · %s", f.done, f.total, costPhrase(f))
}

// sittingSummary is the line finish() prints after the last answer. It carries
// no progress counter — the score line above it already says how the sitting
// went, and repeating it here would be two answers to one question.
func sittingSummary(f sittingFigures) string {
	return costPhrase(f)
}

// costPhrase is the half both share, which is what makes them agree by
// construction rather than by anyone remembering to update the other.
//
// "no room for new words" rather than "0.0 new words/day": a floored zero reads
// like a rounding artifact, and the learner needs to know it is a wall rather
// than a small number.
func costPhrase(f sittingFigures) string {
	if f.fresh <= 0 {
		return fmt.Sprintf("~%.0f reviews/day · no room for new words at %d a sitting", f.load, f.budget)
	}
	return fmt.Sprintf("~%.0f reviews/day · %.1f new words/day at %d a sitting", f.load, f.fresh, f.budget)
}

// wrapOptionLines wraps a form's OPTION lines to the terminal, and leaves
// everything else exactly as it arrived.
//
// It exists because a frame CLIPS: `Paint` cuts a buffer line at the terminal's
// width, since letting it wrap would make the frame a row too tall and the
// terminal would then scroll every row the sitting believes it placed. The
// terminal used to do the wrapping, at the column and with no indent, which `#7`
// recorded as a known rough edge — so what was ugly became missing when `#41`
// took the screen.
//
// AT WRITE TIME rather than when the queue is built, and that is the whole of
// BR-4. The first fix wrapped inside `choiceFor`, which bakes the sitting's
// STARTUP width into every question — so narrowing the window mid-sitting clipped
// every option from there on, the same defect one resize later. Here the width is
// whatever `opt.width` says at the moment the question is written, and the resize
// case keeps that current.
//
// ONLY option lines, matched on the shape `optionLine` writes (a digit and
// `play.OptionIndent-1` spaces). The rest of a prompt is a headword, a blank, or
// a definition `Render` has already wrapped — and re-wrapping an already-wrapped
// line would re-indent it.
func wrapOptionLines(text string, width int) string {
	if width <= 0 {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if !isOptionLine(line) {
			continue
		}
		lines[i] = line[:play.OptionIndent] +
			wrapText(line[play.OptionIndent:], width, play.OptionIndent)
	}
	return strings.Join(lines, "\n")
}

// isOptionLine reports whether a line is one `optionLine` wrote: a digit, then
// the rest of `play.OptionIndent` in spaces, then the gloss.
//
// A shape test rather than a parser. The alternative — teaching `play` to hand
// back its options separately for wrapping — would put line-breaking in the
// package whose whole point is that the caller owns formatting.
func isOptionLine(line string) bool {
	if len(line) <= play.OptionIndent || line[0] < '1' || line[0] > '9' {
		return false
	}
	return strings.TrimLeft(line[1:play.OptionIndent], " ") == ""
}

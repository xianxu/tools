package main

import (
	"fmt"
	"io"
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

// wrapWritten wraps ANYTHING the sitting writes into the buffer, to the width in
// force at the moment of writing.
//
// THE RULE, and it is the rule rather than the three instances that reached it:
// `--play` RENDERS its text when the queue is built and WRITES it much later, so
// every pre-rendered artifact carries a width that may already be wrong by the
// time the frame clips it. `Paint` cuts a buffer line at the terminal's width —
// letting it wrap would make the frame a row too tall and the terminal would
// scroll every row the sitting believes it placed — so a line that arrives too
// wide loses its tail.
//
// The class took three findings to state. The operator saw it at the STARTUP
// width on an option gloss; the boundary review measured it at a NARROWING
// RESIZE on the same lines, then a third time on the rendered DEFINITION a
// reveal writes. Wrapping one line-kind at a time is what produced three
// findings, so this wraps every line the loop writes and the loop routes all of
// them through here: the question, the reveal, the drop notice, the summary and
// the diagnostics.
//
// A line that already fits is returned untouched, so text `Render` has already
// wrapped passes through — only what is too wide is broken, and it is broken at
// spaces rather than at the column.
//
// APPLIED AT THE SEAM, by the pinned screen's own Write, and that is the fourth
// version of this fix. The first three enforced it at call sites: the queue
// build, then the loop's writes. Both leave a helper the loop calls writing
// around it — `playAnnounced`'s network warning goes to the screen and was
// measured at 156 cells in a 40-column terminal. A rule that every caller must
// remember is a rule with a caller who will not.
func wrapWritten(text string, width int) string {
	// THE SUB-20 POLICY, in one place. `terminalWidth` answers 0 below twenty
	// columns because a definition cannot be broken that narrowly and stay
	// readable; the screen's own `cols` is never 0, so the rule has to live with
	// the wrap rather than with whoever happens to be supplying the number.
	if width < minWrapWidth {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		// THE ERASE GESTURE is left alone, and ONLY it. `wrapText` rebuilds a
		// line out of its FIELDS, which would scatter the screen's own
		// `\r\x1b[K` take-that-line-back marker across a break and leave the
		// `♫ playing 3×` indicator in the exit transcript it exists to stay out
		// of. Those lines are short by construction, so nothing is lost.
		//
		// SGR is NOT skipped, and the first version of this skipped every line
		// carrying an escape — which disabled the wrap for exactly the lines it
		// exists for. `--play` refuses to run with `-no-color` (BR-3), so every
		// rendered definition line carries colour, so every one of them was
		// exempt: a Critical, and one no in-process test could see while the rig
		// ran colourless. `visibleCells` measures styled text correctly, and a
		// span broken across a wrap still renders — the attribute persists to
		// its reset, wherever the line break falls.
		if strings.Contains(line, eraseLine) {
			continue
		}
		// The columns before the content, which is what a continuation has to
		// line up under: an option's number-and-gap, or the indentation Render
		// gave a definition body, a quotation or a bullet.
		prefix := len(line) - len(strings.TrimLeft(line, " "))
		if isOptionLine(line) {
			prefix = play.OptionIndent
		}
		lines[i] = line[:prefix] + wrapText(line[prefix:], width, prefix)
	}
	return strings.Join(lines, "\n")
}

// isOptionLine reports whether a line is one `optionLine` wrote: a digit, then
// the rest of `play.OptionIndent` in spaces, then the gloss.
//
// A shape test rather than a parser, and it exists because an option line is the
// one kind whose hanging indent is NOT its own leading whitespace — it has none,
// and its continuation belongs under the gloss rather than under the number. The
// alternative, teaching `play` to hand back its options separately, would put
// line-breaking in the package whose whole point is that the caller owns
// formatting.
// minWrapWidth is the narrowest terminal worth breaking lines for: below it a
// dictionary entry cannot be broken and stay readable.
//
// THE ONLY spelling of that judgement, which it became one round after a comment
// here claimed it already was. `terminalWidth` and the editor's resize case both
// carried their own `< 20`, so declaring a constant and leaving them was half a
// fix — the same half BR-1 left when a shared helper landed beside the copy it
// was meant to replace. Extracting an owner is one move; deleting what it
// replaces is the other, and the second is the one that makes the claim true.
const minWrapWidth = 20

func isOptionLine(line string) bool {
	if len(line) <= play.OptionIndent || line[0] < '1' || line[0] > '9' {
		return false
	}
	return strings.TrimLeft(line[1:play.OptionIndent], " ") == ""
}

// writeClickable writes text and its click map, but ONLY while the two still
// agree about where things are.
//
// A region's Line and Col are relative to the text they were computed from, and
// `#41` put a WRAP between the caller and the buffer: a pinned screen breaks an
// over-wide line before it lands, which moves every span below and to the right
// of the break. `#41` BR-25 wrote the obligation down for the first consumer to
// write regions into a pinned screen, and this is it.
//
// So the wrap is applied HERE, and the regions ride along only if it changed
// nothing. At a sitting's own width it changes nothing — the entry was rendered
// at that width — so this is the ordinary case. After the window NARROWS, the
// text still arrives whole and readable and the underlines stop appearing until
// the next question is written at the new width.
//
// THAT TRADE IS THE POINT, not a shortcut: an underline that plays the word
// beside the one you pointed at is worse than no underline, because the reader
// cannot tell it happened. Losing an affordance is visible; a wrong click is
// not.
func writeClickable(w io.Writer, text string, rs []Region, width int) {
	if wrapWritten(text, width) != text {
		fmt.Fprint(w, text)
		return
	}
	writeRendered(w, text, rs)
}

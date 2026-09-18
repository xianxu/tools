package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/xianxu/tools/cmd/define/store"
)

// screen is the interactive loop's line buffer and viewport (#30).
//
// It exists to make a click's coordinates exact. `define` takes the ALTERNATE
// SCREEN, which has no scrollback — there is nothing above the viewport for the
// terminal to show — so nothing can move the view except this type, and a click
// at viewport row R is buffer line `R + offset` by construction rather than by
// tracking something the app never observes.
//
// It is an io.Writer, and that is what keeps this from being a rewrite: Render
// returns a string, the ask path streams, commands and the indicator print, and
// every one of them feeds the buffer unchanged. It REPLACED the translating
// writer both raw loops used to wrap stdout in — both translate for a raw
// terminal, and two owners of line endings is how they drift. With `#41`
// converting `--play`, that writer had no caller left and is gone.
//
// Write and Frame do no terminal IO. Paint is the only part that touches a
// terminal, so the arithmetic here is unit-testable with no pty.
type screen struct {
	// lines is everything the session has shown, oldest first. The LAST line may
	// be partial: deltas arrive chunked and a reply split as "one" then " two\n"
	// is one line, not two.
	lines  []string
	paints map[int]rowPaint
	// scheme is the process's scheme holder (#70), read ONCE per frame at
	// paint: a row keeps only WHETHER it is tinted, so a /scheme switch
	// recolours everything already here. Nil paints dark. Set by attachScheme,
	// before the screen is shared.
	scheme      *schemeHolder
	promptPaint []rowPaint
	footerPaint []rowPaint
	// partial reports whether the final element is still being written to, so a
	// later Write continues it rather than starting a line.
	partial bool
	// offset is how far back the viewport sits, in lines from the tail. 0 is the
	// bottom — where a session lives — so a fresh screen needs no initialisation.
	offset int
	rows   int
	cols   int
	// regions is what each buffer line OFFERS, keyed by line. Sparse: most lines
	// have none, and a session's worth of empty slices would be the bulk of it.
	regions map[int][]Region
	// passageLo and passageHi bound the LIVE passage in buffer lines, half-open.
	//
	// The screen needs them because the copy-vs-mark decision is made here, and
	// `regions` is never pruned: a superseded passage's rows still carry
	// RegionPassageWord forever, so "does this row have a passage word" answered
	// yes for a passage that scrolled away and a drag there marked the CURRENT
	// one (#67, BR-28).
	passageLo, passageHi int
	// marks is the paint-time mark overlay, keyed by buffer line (#67).
	//
	// PAINT TIME, because the buffer's bytes are immutable once written and a
	// mark is transient — it exists between marking a word and asking about it.
	// The same mechanism markClickable uses for the underline, and for the same
	// reason: decoration that changes belongs to the frame, not to the record.
	//
	// The session owns which SPANS are marked; this is those spans in the
	// screen's own coordinates, handed down on every draw. One fact, one owner,
	// two coordinate spaces.
	marks map[int][]cellRange
	// pinned makes the buffer region occupy its FULL height, so the footer sits
	// at the terminal's bottom edge rather than directly under the content.
	//
	// A property of the screen rather than of a paint call, because it is a
	// standing fact about what this surface is: `--play`'s status bar belongs at
	// the bottom, and the editor's dropdown belongs under the line you are
	// typing (D3a). A REPL prompt stranded at the screen's edge with thirty
	// blank rows above it would be a regression in a loop people already use,
	// which is why the two are named constructors rather than a boolean at a
	// call site that already takes two integers.
	//
	// The padding is blank rows emitted at PAINT time, never lines appended to
	// the buffer: the transcript and the click map must not gain rows that exist
	// only because the terminal is tall.
	pinned bool
	// gap is the rows held EMPTY between the record and the live edge, honoured
	// by Paint. Zero for the editor, chromeGap for a sitting — set at the
	// constructor, where `pinned` already makes that difference visible.
	gap int
	// footer and footerTop are WHERE THE LIVE EDGE ENDED UP, recorded by the
	// last Paint so a click can be resolved against it (#40 D10).
	//
	// Paint already computes both — the footer it actually drew, after fitFooter
	// dropped what would not fit, and the viewport row it began at — and simply
	// did not report them. This is that report, and it is what makes the live
	// edge clickable at all: the buffer is append-only, which is what makes a
	// click's coordinates exact and also what stops anything written there from
	// ever changing, so a surface with marks that change colour has to live in
	// the footer instead.
	footer    []string
	footerTop int
}

// Write appends bytes to the buffer, splitting on newlines.
//
// A bare "\r" is dropped rather than kept: it is the carriage half of a CRLF
// that arrived in a different chunk, which is the case the writer this replaced
// documented
// ("a reply split as \"one\\r\" then \"\\ntwo\" must not become \"one\\r\\r\\ntwo\"").
// Here there is no terminal to position, so the CR carries no information at
// all — a line's placement is Paint's business.
//
// eraseLine ("\r\x1b[K") is the ONE exception, and M1.1's "a CR carries no
// information" was too broad: this pair is the ephemeral indicator's own
// gesture, "take that line back". It is how `♫ playing 3×` disappears once it
// has served its purpose, and how a `define: …` note replaces the line it is
// written over. A buffer that ignored it would keep the indicator, and the exit
// transcript (D3) would then file a claim that playback happened — which is the
// "ephemeral UI vs record" doctrine failing in the one direction it exists to
// prevent. So the erase is HONOURED here rather than stripped: the open line is
// taken back whole.
func (s *screen) Write(p []byte) (int, error) {
	text := strings.ReplaceAll(string(p), "\r\n", "\n")
	// Split on the erase gesture BEFORE bare CRs are dropped — dropping first
	// would leave a lone "\x1b[K" that means nothing to a line buffer.
	if segments := strings.Split(text, eraseLine); len(segments) > 1 {
		for i, seg := range segments {
			if i > 0 {
				s.eraseOpenLine()
			}
			s.write(seg)
		}
		return len(p), nil
	}
	s.write(text)
	return len(p), nil
}

// eraseOpenLine takes back the line currently being written.
//
// Only an OPEN line: a completed line is scrollback, and eraseLine addresses
// the row the cursor sits on, which after a newline is a row nothing has been
// written to yet. Dropping the line rather than blanking it is what keeps an
// erased indicator out of the transcript entirely instead of leaving a blank
// row where it used to be.
func (s *screen) eraseOpenLine() {
	if !s.partial || len(s.lines) == 0 {
		return
	}
	delete(s.paints, len(s.lines)-1)
	s.lines = s.lines[:len(s.lines)-1]
	s.partial = false
}

func (s *screen) write(text string) {
	text = strings.ReplaceAll(text, "\r", "")
	if text == "" {
		return
	}
	// New output SNAPS the viewport back to the tail. Everything this program
	// writes is an answer to something the user just typed, so the thing they
	// asked for has to be the thing they see — and an offset held across an
	// append does not even hold the view still: it is measured from the tail, so
	// the text under the reader's eye slides up by a line per line written.
	s.offset = 0
	parts := strings.Split(text, "\n")
	for i, part := range parts {
		if i == 0 && s.partial && len(s.lines) > 0 {
			s.lines[len(s.lines)-1] += part
			continue
		}
		s.lines = append(s.lines, part)
	}
	// A trailing "\n" ends the last line; anything else leaves it open.
	s.partial = !strings.HasSuffix(text, "\n")
	if !s.partial && len(s.lines) > 0 && s.lines[len(s.lines)-1] == "" {
		// Split leaves an empty tail after a terminating newline. Drop it, or
		// every completed write would add a blank line.
		s.lines = s.lines[:len(s.lines)-1]
	}
}

// addRegions records the regions belonging to a run of buffer lines, so a click
// can be resolved back to what was rendered there (#30 M2.3).
//
// A per-LINE map rather than a list scanned linearly: a session is thousands of
// lines and a click has to answer at once, but more importantly the line is the
// only thing that stays true. Regions are collected against a Render's own
// coordinates; the screen knows where that render landed in the buffer, and
// nothing afterwards can move a line that is already written.
func (s *screen) addRegions(rs []Region) {
	if len(rs) == 0 {
		return
	}
	if s.regions == nil {
		s.regions = map[int][]Region{}
	}
	// base is the buffer line the render STARTED at. Render's line 0 is the line
	// the writer was on when it began, which is the last line written — the
	// caller ends the prompt line before writing an entry, so that line is
	// complete and the entry begins on the next one.
	base := len(s.lines)
	if s.partial {
		base--
	}
	for _, r := range rs {
		ln := base + r.Line
		s.regions[ln] = append(s.regions[ln], r)
	}
}

// RegionAt is the hit test: what, if anything, is offered at this BUFFER line
// and display column. PURE.
//
// The column is the region's own — display cells, as everything else in this
// program measures width — so a caller passes what the terminal reported and
// nothing has to agree about a second unit.
func (s *screen) RegionAt(line, col int) (Region, bool) {
	for _, r := range s.regions[line] {
		if col >= r.Col && col < r.Col+r.Width {
			return r, true
		}
	}
	return Region{}, false
}

// LineAt maps a VIEWPORT row to the buffer line showing there, which is the
// mapping the alternate screen exists to make exact: nothing but this type can
// move the view, so it is arithmetic rather than a guess (#30 D1).
//
// NOT pure, and the plan's table used to call it so: it reaches `visible`, which
// re-establishes the clamp and WRITES BACK `s.offset`. That write is the fix for
// BR-42 rather than an accident — the invariant has to be re-established on
// every path that reads it — but a comment claiming purity about a function that
// mutates is the kind of false label a reader plans around.
//
// The second return value is false for a row below the buffer's tail — the
// prompt, the footer, or blank space — where there is nothing to click.
func (s *screen) LineAt(row int) (int, bool) {
	frame, top := s.visible()
	if row < 0 || row >= len(frame) {
		return 0, false
	}
	return top + row, true
}

// FooterRowAt resolves a click on the LIVE EDGE: a viewport row to the index of
// the footer entry drawn there, and to WHICH of that entry's physical rows was
// hit — 0 for its first, 1 for the first continuation, and so on.
//
// The INDEX, because that is what a caller wants to know: "which of the things I
// handed over was clicked". The screen wrapped them, so the screen owns the
// mapping; making the caller work out which of its entries had wrapped would be
// two owners of one measurement.
//
// AND THE OFFSET, because the index alone is not enough to place a COLUMN
// (R9). An entry too wide for the terminal is drawn across several rows, and a
// click on the second of them carries a column that means nothing in the
// entry's own coordinate space — column 4 of a continuation is column
// cols+4 of the entry. A caller that acts on a column has to be able to refuse
// that, and it can only refuse what it is told about.
//
// False for the buffer, for the prompt, and for a footer row fitFooter dropped:
// a click on a row that was not drawn is a click on nothing, and inventing an
// entry for it would mark a word that is not on screen.
//
// Answered from the LAST PAINT rather than from the current state, because that
// is what the person clicking was looking at.
func (s *screen) FooterRowAt(row int) (entry, offset int, ok bool) {
	if row < s.footerTop {
		return 0, 0, false
	}
	off := row - s.footerTop
	for i, m := range s.footer {
		h := displayRows(m, s.cols)
		if off < h {
			return i, off, true
		}
		off -= h
	}
	return 0, 0, false
}

// Lines is the whole buffer. Present for tests and for the exit transcript
// (D3), which is a loop over exactly this.
func (s *screen) Lines() []string { return s.lines }

// Frame is the rows to paint, oldest first. PURE.
func (s *screen) Frame() []string {
	frame, _ := s.visible()
	return frame
}

// visible is the frame AND the buffer line it starts at, as ONE answer.
//
// Together, because they are one fact about the viewport and computing them
// separately is how they came apart: `Frame` used to return early — for an empty
// viewport, and for a buffer that fits — WITHOUT clamping, so a stale offset
// survived. Growing the viewport while scrolled back (a resize taller, the
// footer closing, a wrapped prompt cleared with Ctrl-U) then left `offset`
// pointing past the end, `topLine` negative, and the click map detached from the
// text: the underline painted on one row while the region answered on another.
//
// The rule it leaves: a DERIVED invariant is re-established on every path that
// reads it, not only on the path that calls its owner. So the clamp happens
// once, here, above every return.
func (s *screen) visible() ([]string, int) {
	s.clamp()
	if s.rows <= 0 {
		return nil, len(s.lines)
	}
	if len(s.lines) <= s.rows {
		return s.lines, 0
	}
	end := len(s.lines) - s.offset
	return s.lines[end-s.rows : end], end - s.rows
}

// Scroll moves the viewport by n lines — positive is BACKWARD, toward older
// text, which is the direction "scroll up" means to a reader.
func (s *screen) Scroll(n int) {
	s.offset += n
	s.clamp()
}

// clamp is the ONE place the viewport's limits are spelled. A wheel event
// arrives per notch and a held PageUp repeats, so both overshoot routinely.
func (s *screen) clamp() {
	max := len(s.lines) - s.rows
	if max < 0 {
		max = 0
	}
	if s.offset > max {
		s.offset = max
	}
	if s.offset < 0 {
		s.offset = 0
	}
}

// Page moves the viewport by whole screenfuls, keeping ONE line of overlap so
// the eye has an anchor across the jump — the convention every pager follows.
//
// The step is computed here rather than passed in because the screen is the only
// thing that knows how tall the viewport is: the loop knows a key was pressed,
// not how much of the buffer that key is worth.
func (s *screen) Page(n int) {
	step := s.rows - 1
	if step < 1 {
		step = 1 // a viewport too short for overlap still moves
	}
	s.Scroll(n * step)
}

// underlineOn/Off mark a span as CLICKABLE (#30 M2.5).
//
// An ATTRIBUTE, not a seventh colour. `newPalette` already spends six on MEANING
// — head, ipa, pos, num, ex, sect — plus bold-green for deck words, so another
// colour would compete with a scheme that is already saying something. Underline
// composes with whatever colour a span already carries, which is exactly what
// "this text is also clickable" should do.
//
// STATIC, not on hover, and the tracking mode is the reason: hover needs mode
// 1003, which streams an event for every cell the pointer crosses, so the loop
// would wake constantly to redraw. Mode 1000 reports presses only, and with it
// the app never learns where the pointer is.
const (
	underlineOn  = "\x1b[4m"
	underlineOff = "\x1b[24m"
)

// markClickable splices the underline attribute into the spans a line offers.
//
// SPLICED BY THE SCREEN, never by Render, and that placement is a promise: D6
// says `define <word>`, `-raw` and `> out.txt` keep today's bytes exactly, and
// an underline emitted by Render would leak into all three — and redden the
// corpus golden that says so.
//
// Attributes only. 24 turns the underline off without touching colour, which is
// why the span's existing style survives untouched and no state has to be
// remembered across the splice. Applying `0` here would be the bug this comment
// exists to prevent: it would end the colour the palette opened, and the rest of
// the line would go plain.
func markClickable(line string, rs []Region) string {
	if len(rs) == 0 {
		return line
	}
	// By column, so the splices are applied left to right and the offsets stay
	// meaningful as we walk.
	//
	// Kinds that do not want an underline are dropped first. The mark means "this
	// particular span offers something the text around it does not" — in a
	// passage EVERY word is clickable, so underlining them all says nothing and
	// only makes the passage hard to read (operator-reported, with a screenshot).
	spans := make([]Region, 0, len(rs))
	for _, r := range rs {
		if regionUnderlines(r.Kind) {
			spans = append(spans, r)
		}
	}
	if len(spans) == 0 {
		return line
	}
	slices.SortFunc(spans, func(a, b Region) int { return a.Col - b.Col })

	var b strings.Builder
	col, next, open := 0, 0, false
	for i := 0; i < len(line); {
		// Escapes are stepped over FIRST, so the mark lands immediately before
		// the span's first visible character rather than before whatever colour
		// the palette opens there. The predecessor tested the column on every
		// iteration — escape steps included — so it emitted the attribute once
		// per byte at that column, with the palette's escape spliced between:
		// harmless to a terminal, and it made `underlineOn+text` stop being a
		// thing a reader (or a test) could look for.
		if skip := escapeLen(line[i:]); skip > 0 {
			b.WriteString(line[i : i+skip])
			i += skip
			continue
		}
		size, w := nextDisplayUnit(line[i:])
		// A region can start in either cell of a wide unit. Mark that whole
		// unit, and discard exhausted spans so they cannot hide later regions.
		for !open && next < len(spans) && spans[next].Col+spans[next].Width <= col {
			next++
		}
		if !open && next < len(spans) && spans[next].Col < col+w {
			b.WriteString(underlineOn)
			open = true
		}
		b.WriteString(line[i : i+size])
		col += w
		i += size
		// Closed the moment the span's last cell is written, so the attribute
		// covers the span and nothing after it.
		if open && col >= spans[next].Col+spans[next].Width {
			b.WriteString(underlineOff)
			open, next = false, next+1
		}
	}
	// A span reaching the end of the line still closes: an unterminated
	// underline runs on through everything painted after it.
	if open {
		b.WriteString(underlineOff)
	}
	return b.String()
}

// The two sequences a whole-frame redraw needs, beside eraseLine (repl.go) which
// is the partial-redraw model this replaces.
const (
	cursorHome = "\x1b[H" // row 1, column 1
	eraseDown  = "\x1b[J" // clear from the cursor to the end of the screen
)

// chromeGap is the rows a frame holds EMPTY between the record and the live edge.
//
// The live edge is a legend of what you can press; the buffer is what you are
// reading. With nothing between them the action row butts the last line of the
// definition it belongs under and the two read as one block — which is what the
// operator saw in a real sitting (#44).
//
// A ROW THE FRAME RESERVES, never a "\n" inside the prompt. `displayRows`
// measures the prompt in visible CELLS and knows nothing about an embedded
// newline, so a two-line prompt would be charged one row and the frame would come
// out one row too tall — the terminal scrolls and every placed row moves. Paint
// also writes the prompt with a bare WriteString, where raw mode needs "\r\n".
const chromeGap = 1

// grantedGap is whether a frame of this shape gets its gap.
//
// ONE PRODUCTION CONSUMER — `Paint` — and that is a CORRECTION to what this
// comment first said. It claimed two, `fitsABoard` being the second, which was
// true of the design and never of the code: charging the gap there turned out to
// be provably a no-op, so the term was never written (see the plan's
// `## Revisions`, and `fitsABoard`'s own doc, which now says the opposite). A
// comment advertising a consumer that does not exist reads as an instruction to
// add it back (#44 I4).
//
// What the two DO share is the property rather than a call: they agree at every
// shape because this hands out a row only where there was already slack for it,
// and TestTheChromeGapNeverChangesWhetherABoardFits pins that by exhaustion.
//
// DECORATION, so it is the first component given up — before the buffer, before
// the footer, before the prompt. A frame that scrolls has lost every coordinate
// on it, and a border is not worth that. Granted only when a buffer row survives
// beside it: at that size the reader needs the content more than the border.
func grantedGap(want, termRows, promptRows, footerRows int) int {
	if termRows-promptRows-footerRows >= want+1 {
		return want
	}
	return 0
}

// Paint draws one whole frame: the buffer's visible tail, then the prompt, then
// the FOOTER under it.
//
// "Footer" and not "menu", because the concept is *rows below the prompt that
// give up whole rows before the prompt does*, and there are two consumers of it:
// the editor's command menu and `--play`'s status bar. Naming it for one of them
// would make the other's call site read as something it is not, and would invite
// a third consumer to add a third parameter for its own bottom rows.
//
// The frame is redrawn WHOLE — home, clear, everything — rather than patched.
// That is the change of model this milestone buys: the editor currently tracks
// how many footer rows it drew so it can erase exactly that many, and carries a
// documented known limit for when the count is wrong ("if the footer does not fit
// below the cursor the terminal scrolls, and the cursor-up count then lands a
// row off"). A whole-frame redraw cannot be off by a row, because it never
// counts rows it drew earlier.
//
// The prompt and the footer are NOT buffer lines. They are the live edge of the
// session and change on every keystroke; putting them in `lines` would append a
// copy of the prompt per character typed.
//
// A frame is budgeted in DISPLAY ROWS, not in lines, and that distinction is the
// whole guarantee. A line wider than the terminal wraps onto a second row; a
// frame that counted it as one is a frame one row too tall, and the terminal
// then SCROLLS to fit it — which moves every row the app believes it placed, and
// a click at viewport row R stops meaning buffer line R+offset. Two routine ways
// in: narrow the window (buffer lines keep the wrapping they were rendered with,
// by decision) or type a line longer than the terminal is wide.
//
// So the prompt and footer are charged their REAL height, and buffer lines are
// clipped to the width — clipped at PAINT time, so the transcript and the click
// map keep the whole text.
//
// termRows and termCols are passed in rather than stored, so a resize is one
// call site's business (the loop's SIGWINCH case) and not fields that go stale.
func (s *screen) Paint(w io.Writer, termRows, termCols int, prompt string, footer []string) {
	s.paintActivity(w, termRows, termCols, prompt, footer, "")
}

// selectionLayout retains the painter's original chunks and their physical rows.
// Placement is computed once; selection never reconstructs a second viewport.
type selectionLayout struct {
	frame  selectionFrame
	chunks []selectionPaintChunk
	// scheme is read once when the frame is laid out, so a transition landing
	// mid-frame cannot paint two shades in one frame.
	scheme store.Scheme
}

type selectionPaintChunk struct {
	text         string
	paint        rowPaint
	first, count int
}

func (layout selectionLayout) paint(w io.Writer, gesture selectionGesture) {
	var b strings.Builder
	for _, chunk := range layout.chunks {
		if (gesture.selected || gesture.active && gesture.dragging) && gesture.frame != 0 && layout.frame.err == nil && chunk.count > 0 {
			for row := chunk.first; row < chunk.first+chunk.count; row++ {
				b.WriteString(paintLanguageRow(layout.frame.highlightRow(row, gesture.anchor, gesture.end), layout.frame.rows[row].paint, layout.frame.width, layout.scheme))
			}
		} else {
			b.WriteString(paintOutputChunk(chunk.text, chunk.paint, layout.frame.width, layout.scheme))
		}
	}
	fmt.Fprint(w, b.String())
}

func (s *screen) paintActivity(w io.Writer, termRows, termCols int, prompt string, footer []string, glyph string) {
	s.layoutSelectionFrame(termRows, termCols, prompt, footer, glyph, "", false).paint(w, selectionGesture{})
}

func (s *screen) layoutSelectionFrame(termRows, termCols int, prompt string, footer []string, glyph, notice string, retry bool) selectionLayout {
	s.cols = termCols
	// Retain RenderLine's erase/cursor controls, except its redundant leading CR.
	prompt = strings.TrimPrefix(prompt, "\r")
	prompt = clipVisible(prompt, selectionPromptBudget(termRows, termCols))
	prompt = clipSelectionRows(prompt, max(1, termRows), termCols)
	promptRows := displayRows(prompt, s.cols)
	glyph = activityRow(prompt, glyph, termRows, termCols)
	activityRows := 0
	if glyph != "" {
		activityRows = 1
		if prompt == "" {
			promptRows = 0
		}
	}
	// Feedback is chrome, never a record. On the smallest terminal it borrows
	// the prompt display, leaving the liveScreen's stored editor prompt intact.
	noticePrompt := notice != "" && termRows-promptRows-activityRows < 1
	if noticePrompt {
		prompt = clipVisible(notice, max(termCols, 1))
		promptRows, glyph, activityRows = 1, "", 0
		footer = nil
	} else if notice != "" {
		footer = append([]string{clipVisible(notice, max(termCols, 1))}, footer...)
	}
	footer, footerRows := fitFooter(footer, termRows-promptRows-activityRows, s.cols)
	gap := grantedGap(s.gap, termRows, promptRows+activityRows, footerRows)
	s.rows = max(0, termRows-promptRows-activityRows-footerRows-gap)
	layout := selectionLayout{scheme: s.scheme.Scheme()}
	var rows []selectionRow
	snapshotBytes, snapshotRefused := 0, false
	// Refuse oversized selection snapshots before allocating rows/cells. The
	// ordinary painter keeps working and the attempted gesture explains refusal.
	bounded := termRows > 0 && termCols > 0 && termRows <= maxSelectionCells/termCols
	control := func(text string) { layout.chunks = append(layout.chunks, selectionPaintChunk{text: text}) }
	place := func(text string, role selectionRow) {
		chunk := selectionPaintChunk{text: text, paint: role.paint, first: len(rows)}
		if bounded && len(text) > maxSelectionSource-snapshotBytes {
			bounded, snapshotRefused, rows = false, true, nil
		}
		if bounded {
			physical := selectionPhysicalRows(text, termCols)
			if physical == nil {
				bounded, snapshotRefused, rows = false, true, nil
			}
			paintColumn := 0
			for offset, line := range physical {
				if len(line) > maxSelectionSource-snapshotBytes {
					bounded, snapshotRefused, rows = false, true, nil
					break
				}
				snapshotBytes += len(line)
				r := role
				r.styled = line
				r.paint = sliceRowPaint(role.paint, paintColumn, visibleCells(line))
				paintColumn += visibleCells(line)
				if r.footer {
					r.footerOffset = offset
				}
				rows = append(rows, r)
			}
			chunk.count = len(physical)
		}
		layout.chunks = append(layout.chunks, chunk)
	}
	control(cursorHome + eraseDown)
	frame, top := s.visible()
	for i, line := range frame {
		painted := markClickable(line, s.regions[top+i])
		if m := s.marks[top+i]; len(m) > 0 {
			painted = paintMarks(painted, m)
		}
		place(clipVisible(painted, s.cols), selectionRow{selectable: true, regions: s.regions[top+i], paint: s.paints[top+i]})
		control("\r\n")
	}
	bufRows := len(frame)
	if s.pinned {
		for range s.rows - len(frame) {
			place("", selectionRow{})
			control("\r\n")
		}
		bufRows = s.rows
	}
	for range gap {
		place("", selectionRow{})
		control("\r\n")
	}
	s.footer, s.footerTop = footer, bufRows+gap+activityRows+promptRows
	if glyph != "" {
		place(glyph, selectionRow{})
		if promptRows > 0 {
			control("\r\n")
		}
	}
	promptStart := len(rows)
	if promptRows > 0 {
		place(prompt, selectionRow{selectable: !noticePrompt, retry: noticePrompt && retry, paint: paintAt(s.promptPaint, 0)})
	}
	for i, line := range footer {
		control("\r\n")
		role := selectionRow{selectable: true, footer: true, footerEntry: i, paint: paintAt(s.footerPaint, i)}
		if notice != "" && !noticePrompt {
			role.footerEntry--
			if i == 0 {
				role.selectable, role.footer, role.retry = false, false, retry
			}
		}
		role.paint = paintAt(s.footerPaint, role.footerEntry)
		if role.retry {
			role.paint = rowPaint{}
		}
		place(line, role)
	}
	if len(footer) > 0 && promptRows > 0 {
		control(fmt.Sprintf("\x1b[%dA\r", footerRows+promptRows-1))
		// Reprinting parks the original editor cursor, including suggestion-back
		// controls. Reuse the same row spans so selection remains highlighted.
		count := 0
		if bounded {
			count = len(selectionPhysicalRows(prompt, termCols))
		}
		layout.chunks = append(layout.chunks, selectionPaintChunk{text: prompt, paint: paintAt(s.promptPaint, 0), first: promptStart, count: count})
	}
	layout.frame = newSelectionFrame(termCols, termRows, rows)
	if snapshotRefused {
		layout.frame = selectionFrame{width: termCols, height: termRows, err: errSelectionBounds}
	}
	return layout
}

// selectionPromptBudget avoids overflowing dimension multiplication on a hostile
// resize. This limits clipping arithmetic; the frame constructor bounds snapshots.
func selectionPromptBudget(rows, cols int) int {
	cols = max(cols, 1)
	if rows > 0 && rows > int(^uint(0)>>1)/cols {
		return int(^uint(0) >> 1)
	}
	return rows * cols
}

// selectionPhysicalRows splits only at terminal soft wraps, retaining controls
// in their original positions and resuming SGR on a continuation. Joining these
// rows emits the same glyphs and cursor controls as the original stream.
func selectionPhysicalRows(text string, width int) []string {
	var rows []string
	var style sgrState
	sourceBytes := 0
	ok := walkSelectionRows(text, width, func(start, end int) bool {
		prefix := style.resume()
		if len(prefix) > maxSelectionSource-sourceBytes || end-start > maxSelectionSource-sourceBytes-len(prefix) {
			return false
		}
		line := text[start:end]
		rows = append(rows, prefix+line)
		sourceBytes += len(prefix) + len(line)
		for i := 0; i < len(line); {
			if n := escapeLen(line[i:]); n > 0 {
				style.observe(line[i : i+n])
				i += n
			} else {
				_, n := utf8.DecodeRuneInString(line[i:])
				i += n
			}
		}
		return true
	})
	if !ok {
		return nil
	}
	return rows
}

// walkSelectionRows owns the terminal soft-wrap boundary for budgeting, clipping,
// painting spans and hit snapshots. Wide glyphs can leave an unused last column;
// dividing the total cell count by width loses those rows.
func walkSelectionRows(text string, width int, visit func(start, end int) bool) bool {
	start, col := 0, 0
	for i := 0; i < len(text); {
		if n := escapeLen(text[i:]); n > 0 {
			i += n
			continue
		}
		n, w := nextDisplayUnit(text[i:])
		if width > 0 && w > 0 && col > 0 && col+w > width {
			if !visit(start, i) {
				return false
			}
			start, col = i, 0
		}
		col += w
		i += n
	}
	return visit(start, len(text))
}

func clipSelectionRows(text string, rows, cols int) string {
	n, end := 0, len(text)
	walkSelectionRows(text, cols, func(_, stop int) bool {
		n++
		if n == rows {
			end = stop
			return false
		}
		return true
	})
	if end < len(text) {
		return text[:end] + sgrOff
	}
	return text
}

// liveScreen is the screen wired to a terminal: an io.Writer that SHOWS what is
// written to it, which is what stdout was before the alternate screen.
//
// It exists because the buffer alone is invisible. A definition, a streamed
// answer arriving token by token, a `♫ playing 3×` that must appear while
// playback blocks for seconds — every one of them reached the terminal today by
// being written to it, and under a buffer they would appear only at the loop's
// next redraw. So a write repaints, and the callers D5 promised would not change
// genuinely do not.
//
// The split is the ARCH-MOCK line: `screen` is pure and its arithmetic is
// unit-tested with no terminal; this type is the only part that does IO, and it
// holds the live edge (the prompt and menu) that Paint needs and the buffer does
// not own.
type liveScreen struct {
	s   *screen
	tty io.Writer
	// rows and cols are the terminal's SHAPE. Owned here rather than in `screen`
	// because they are facts about the terminal, not about the text — and
	// M1.4's SIGWINCH has exactly one place to update.
	rows                       int
	cols                       int
	prompt                     string
	footer                     []string
	activity                   *activityLease
	activityGlyph              string
	paintErr                   error
	frame                      selectionFrame
	frameID                    uint64
	framePublished             bool
	gesture                    selectionGesture
	selectionNotice            string
	failedCopy                 string
	copySeq                    uint64
	observedRows, observedCols int
	sizeObserved               bool
	// stopped is set when the terminal has been handed back. Writes still reach
	// the buffer — the exit transcript needs them — but painting must stop dead,
	// or a farewell newline written after restore would draw a frame onto the
	// NORMAL screen, over whatever the user was looking at before define ran.
	stopped bool
	// suspended is set while ANOTHER screen owns the terminal — see suspend.
	// Separate from stopped because it is reversible and stopped is not.
	suspended bool
	// painted is when the last frame went out, and pending says a write has
	// happened since. Together they THROTTLE the repaint (#30 M1.4/BR-16): the
	// ask path writes once per streamed delta, and a frame per delta is a
	// full-screen erase-and-redraw per token — hundreds of them, megabytes to
	// the tty, for one answer.
	//
	// The bound this buys is on STALENESS, not on correctness: at any moment the
	// unpainted text is only what arrived since the last frame, at most one
	// interval's worth, and every gesture that ends a burst — Draw, Page,
	// Scroll, Resize, Stop — paints unconditionally. A stalled stream therefore
	// shows everything up to the stall.
	painted time.Time
	// interval is the throttle's window. Zero means paintInterval; a test sets
	// it long to make the pending state deterministic.
	interval time.Duration
	pending  bool
	// timer is the TRAILING half, and it is what makes the throttle safe rather
	// than merely cheap. A held frame must go out whether or not another write
	// follows: the `♫ playing 3×` indicator is written and then playback blocks
	// for seconds, so a throttle that waited for the next write would hide it for
	// the whole recording — a worse bug than the one being fixed.
	//
	// mu guards everything above, because this timer paints from its own
	// goroutine while the loop is blocked inside speak.
	mu    sync.Mutex
	timer *time.Timer
}

// paintInterval is the shortest gap between frames driven by writes. 60fps: fast
// enough that a stream reads as continuous, slow enough that a 300-delta answer
// costs tens of frames instead of hundreds.
//
// A FIELD on liveScreen rather than a bare constant, so a test can hold the
// window open and observe "pending" deterministically instead of racing the
// clock — which is the only way to assert the trailing flush without reaching
// into fields the timer's goroutine writes.
const paintInterval = 16 * time.Millisecond

// newLiveScreen is the EDITOR's screen: the footer follows the content, because
// a REPL prompt belongs directly under the last output.
func newLiveScreen(tty io.Writer, rows, cols int) *liveScreen {
	return &liveScreen{s: &screen{}, tty: tty, rows: rows, cols: cols, interval: paintInterval}
}

// newPinnedScreen is `--play`'s: the buffer region fills, so the footer sits at
// the terminal's bottom edge (D3a).
//
// A named constructor rather than a bool at a call site that already takes two
// integers — and a second constructor rather than a parameter on the first,
// because the two surfaces want opposite things and the difference should be
// visible where the screen is BUILT rather than at every paint.
func newPinnedScreen(tty io.Writer, rows, cols int) *liveScreen {
	l := newLiveScreen(tty, rows, cols)
	l.s.pinned = true
	l.s.gap = chromeGap
	return l
}

// attachScheme gives the screen the process's scheme holder (#70). Call it
// BEFORE the screen is shared with another goroutine — the pointer router, the
// resize watcher, the throttle timer — because this field is not atomic; the
// holder it points at is.
func (l *liveScreen) attachScheme(h *schemeHolder) { l.s.scheme = h }

// window is the throttle's gap. Zero means the default; NEGATIVE means none at
// all, which is how a test asks for the unthrottled behaviour rather than
// waiting out a real interval.
func (l *liveScreen) window() time.Duration {
	if l.interval == 0 {
		return paintInterval
	}
	return l.interval
}

// Write feeds the buffer and shows the result, at most paintInterval apart.
//
// On a PINNED screen it wraps first, to the terminal's current width. That is
// the seam the wrap belongs at rather than at the loop's call sites: `Paint`
// clips a line too wide for the terminal, and a sitting writes not only its own
// text but whatever the helpers it calls write — a playback warning reached the
// buffer at 156 cells in a 40-column terminal while every site the loop owns was
// wrapped. Here nothing can write around it.
//
// The editor's screen does NOT wrap: definitions arrive pre-wrapped by Render,
// and runAsk's answerWrapWriter carries word boundaries across streamed chunks.
// Both use the policy width. A sitting writes whole messages.
func (l *liveScreen) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.writeBuffer(string(p)); err != nil {
		return 0, err
	}
	l.throttledPaint()
	// The CALLER's units, which is what io.Writer means by n — the wrap changes
	// how many bytes the buffer received, and reporting that would tell a caller
	// it had written more than it handed over.
	return len(p), nil
}

// writeBuffer is the ONE way text reaches the buffer, and the wrap lives here so
// that is true of every path rather than of the one anybody thought about.
//
// `Write` was the first, and its comment claimed "nothing can write around it"
// while `WriteRegions` did exactly that — the sixth finding in this family, and
// the axis the fifth did not enumerate: that one closed which LINES are wrapped
// and left which PATHS. Callers hold mu.
func (l *liveScreen) writeBuffer(text string) error {
	l.s.invalidatePartialPaint(text)
	if text != "" {
		l.invalidateSelectionLocked()
	}
	if l.s.pinned {
		text = wrapWritten(text, l.cols)
	}
	_, err := l.s.Write([]byte(text))
	return err
}

// throttledPaint paints unless a frame went out too recently, in which case it
// arms the trailing timer. Callers hold mu.
func (l *liveScreen) throttledPaint() {
	if since := time.Since(l.painted); since < l.window() {
		l.pending = true
		if l.timer == nil {
			l.timer = time.AfterFunc(l.window()-since, l.flush)
		}
		return
	}
	l.repaint()
}

// flush paints what the throttle held. Runs on the timer's goroutine, which is
// why every field is behind mu.
func (l *liveScreen) flush() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timer = nil
	if l.pending {
		l.repaint()
	}
}

// Draw records the live edge and repaints. It is the editor loop's draw(), and
// it paints UNCONDITIONALLY: the loop draws when it has stopped writing, so this
// is the frame that settles whatever a burst left pending.
func (l *liveScreen) Draw(prompt string, footer []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.s.promptPaint, l.s.footerPaint = nil, nil
	l.prompt, l.footer = prompt, footer
	l.repaint()
}

// WriteRegions is Write plus the click map for what is being written. The two
// are one call because they must not be able to disagree about which buffer line
// the render landed on — the regions are relative to the render, and only the
// screen knows where that render begins.
func (l *liveScreen) WriteRegions(text string, rs []Region) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// THE REGIONS ARE MOVED HERE, against THIS screen's own cols, and that is
	// the whole of the fix.
	//
	// A region's Line and Col are relative to the text they were computed from,
	// and a pinned screen wraps between the caller and the buffer — so a line
	// the wrap breaks moves everything below it. The caller cannot do this
	// arithmetic: it does not know the width. `--play` tried, using the
	// `opt.width` it was given at startup, and after a resize the two widths
	// disagreed and every region landed on a line that did not contain its text
	// — a headword region on a blank line, an ORIGIN region on a quotation. That
	// is the WRONG-CLICK failure this whole path exists to make impossible, and
	// it was possible because the wrap and the map were measured by two
	// different rulers.
	//
	// One ruler. The screen owns the wrap, so the screen owns the map.
	if l.s.pinned {
		rs = wrapMovedRegions(text, rs, l.cols)
	}
	l.s.addRegions(rs)
	l.writeBuffer(text)
	l.throttledPaint()
}

// BufferLines is how many lines the record holds right now.
//
// Read BEFORE a write, so a caller learns where that write will land. addRegions
// makes the same adjustment for a partial last line, and for the same reason: a
// render begins on the line the writer is already on.
func (l *liveScreen) BufferLines() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(l.s.lines)
	if l.s.partial {
		n--
	}
	return n
}

// SetPassage tells the screen where the LIVE passage is and what is marked in it.
//
// One call for both, because they answer the same question — "is the live passage
// reachable at this point?" — and a screen that knew the marks but not the range
// would gate the paint and not the gesture. Replaces rather than merges: the
// session's state is the whole truth, handed down entire on every draw, so
// nothing stale can survive a clear.
func (l *liveScreen) SetPassage(lo, hi int, m map[int][]cellRange) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.s.passageLo, l.s.passageHi, l.s.marks = lo, hi, m
}

// VisibleRange is the buffer lines the viewport is showing, half-open.
//
// The caller needs it to answer whether the live passage is still on screen —
// "a passage was once pasted" is not authority over Enter for the rest of the
// session.
func (l *liveScreen) VisibleRange() (int, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	frame, top := l.s.visible()
	return top, top + len(frame)
}

// RegionAtRow resolves a click: a VIEWPORT row and display column to whatever is
// offered there.
func (l *liveScreen) RegionAtRow(row, col int) (Region, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	line, ok := l.s.LineAt(row)
	if !ok {
		return Region{}, false
	}
	return l.s.RegionAt(line, col)
}

// FooterRowAt resolves a click on the live edge, under the lock like every other
// read of the screen: Paint runs from the throttle's goroutine too, and this
// reads what Paint wrote.
func (l *liveScreen) FooterRowAt(row int) (int, int, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s.FooterRowAt(row)
}

// Page and Scroll move the viewport and show the result. The paint is the point:
// a scroll nobody can see is not a scroll.
func (l *liveScreen) Page(n int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.invalidateSelectionLocked()
	l.s.Page(n)
	l.repaint()
}

func (l *liveScreen) Scroll(lines int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.invalidateSelectionLocked()
	l.s.Scroll(lines)
	l.repaint()
}

// Resize takes SIGWINCH's whole answer (M1.4). It does NOT paint: the caller
// redraws, because the live edge is rendered against the new width too and a
// frame drawn before that is a frame drawn twice.
func (l *liveScreen) Resize(rows, cols int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.rows != rows || l.cols != cols {
		l.invalidateSelectionLocked()
	}
	l.rows, l.cols = rows, cols
}

// Size is the terminal's shape AS IT IS NOW, for a caller that has to lay
// something out before drawing it (#40 R17).
//
// The screen is the authority: it is given the shape by the resize watcher and
// it is what Paint budgets against. A loop keeping its own copy would be a
// second owner of a number the terminal owns, and the copy would be right until
// the first SIGWINCH it happened not to see.
func (l *liveScreen) Size() (rows, cols int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.rows, l.cols
}

// Stop ends painting. Called as the terminal is handed back, and idempotent for
// the same reason restore is: it runs from more than one exit path.
//
// It FLUSHES first: a throttled write may be waiting, and the last thing a
// session showed must not be the thing the throttle held back.
func (l *liveScreen) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	hadActivity := l.activity != nil
	l.revokeActivity()
	if l.pending || hadActivity {
		l.repaint() // the last thing shown must not be what the throttle held
	}
	if l.timer != nil {
		l.timer.Stop()
		l.timer = nil
	}
	l.stopped = true
	l.cancelSelectionLocked(true)
	l.invalidateSelectionLocked()
	l.frame = selectionFrame{}
}

// suspend stops this screen PAINTING without ending its life, so another screen
// can own the terminal for a while (#48's /play).
//
// Stop is not this. Stop is terminal — it flushes what the throttle held, kills
// the timer and refuses forever — which is right for a session that is over and
// wrong for one that is waiting. A sitting entered from the REPL needs the
// editor's screen to go quiet and then come back with its buffer and viewport
// intact.
//
// IT DOES NOT FLUSH. The pending frame belongs to THIS screen, and painting it
// now would put the editor's last line on top of the sitting's first. `pending`
// is kept so resume can paint it.
//
// The timer is disarmed as HYGIENE, not as the guard — and the distinction is
// worth writing down because the first version of this comment claimed the
// opposite. A timer that fires while suspended calls flush, which calls repaint,
// which returns at the gate above; the mutation sweep proved it, by removing
// this block and watching nothing redden. What it buys is a wakeup not taken and
// a timer not left pointing at a screen nobody is painting — real, small, and
// not the reason the frame stays off the terminal.
//
// Idempotent, so a caller cannot suspend twice and resume once.
func (l *liveScreen) suspend() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.suspended {
		return
	}
	if l.activity != nil {
		l.revokeActivity()
		l.repaint()
	}
	l.suspended = true
	l.cancelSelectionLocked(true)
	l.invalidateSelectionLocked()
	l.frame = selectionFrame{}
	if l.timer != nil {
		l.timer.Stop()
		l.timer = nil
	}
}

// resume gives the terminal back to this screen and paints immediately.
//
// UNCONDITIONALLY, not `if pending`: whatever ran while this screen was quiet has
// overwritten every cell, so the last frame is no longer on screen whether or not
// this screen thinks it has changes. That is the clause that distinguishes resume
// from clearing a flag, and a test without it passes on a Stop in disguise.
//
// A resize that arrived meanwhile is already in rows/cols — the loop sets them —
// so this paints at the new shape rather than the old one.
func (l *liveScreen) resume() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.suspended {
		return
	}
	l.suspended = false
	l.repaint()
}

// Transcript is the buffer, for printing back into the normal buffer on exit.
func (l *liveScreen) Transcript() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s.Transcript()
}

func (l *liveScreen) repaint() {
	// ONE PLACE decides whether a frame may go out, and both flags live here.
	// repaint is the only thing that writes to the tty, so a second gate
	// elsewhere would be a second answer to "may I paint" (#48).
	if l.stopped || l.suspended || l.tty == nil {
		return
	}
	w := &activityPaintWriter{writer: l.tty}
	layout := l.s.layoutSelectionFrame(l.rows, l.cols, l.prompt, l.footer, l.activityGlyph, l.selectionNotice, l.failedCopy != "")
	if !l.frame.same(layout.frame) {
		l.invalidateSelectionLocked()
	}
	layout.paint(w, l.gesture)
	if l.paintErr == nil {
		l.paintErr = w.err
	}
	if w.err != nil {
		l.revokeActivity()
		l.invalidateSelectionLocked()
	} else if !l.sizeObserved || l.rows == l.observedRows && l.cols == l.observedCols {
		l.frame, l.framePublished = layout.frame, true
		l.sizeObserved = false
	}
	l.painted, l.pending = time.Now(), false
}

// fitFooter drops whole menu rows from the END until the dropdown fits the space
// the prompt left, and reports what it costs.
//
// Whole rows, because half a row is worse than no row — half a command name is
// not a menu entry, and half a status bar is a number with no label. From the
// END, because the editor's list is sorted and the first matches are the likely
// ones; --play passes one row, for which the distinction does not arise.
func fitFooter(footer []string, avail, cols int) ([]string, int) {
	used := 0
	for i, m := range footer {
		h := displayRows(m, cols)
		if used+h > avail {
			return footer[:i], used
		}
		used += h
	}
	return footer, used
}

// displayRows is how many terminal rows a line occupies once the terminal has
// wrapped it. Always at least one: an empty line is still a row.
func displayRows(line string, cols int) int {
	n := 0
	walkSelectionRows(line, cols, func(_, _ int) bool { n++; return true })
	return n
}

// clipVisible cuts a line to width VISIBLE columns, keeping the escape sequences
// that fall inside the cut and closing any style left open.
//
// Cutting rather than wrapping, because a frame row is a row: the alternative is
// letting the terminal wrap and losing the placement the whole screen exists to
// guarantee. The buffer keeps the full text, so nothing is lost from the
// transcript — only from the view, which is what a viewport is.
func clipVisible(s string, width int) string {
	if width <= 0 || visibleCells(s) <= width {
		return s
	}
	var b strings.Builder
	n, styled := 0, false
	for i := 0; i < len(s); {
		// Sequences are SKIPPED through the one owner of the escape grammar
		// (escapeLen, render.go) and kept: they cost no columns, and dropping
		// them would strip the colour from the text that survives the cut.
		if skip := escapeLen(s[i:]); skip > 0 {
			b.WriteString(s[i : i+skip])
			styled = true
			i += skip
			continue
		}
		size, w := nextDisplayUnit(s[i:])
		if n+w > width {
			// Cut here — BEFORE the unit, so a wide rune or flag is never half
			// drawn — and hand the style back, or the terminal keeps whatever
			// colour was open when the cut landed.
			if styled {
				b.WriteString(sgrOff)
			}
			return b.String()
		}
		b.WriteString(s[i : i+size])
		n += w
		i += size
	}
	return b.String()
}

// Transcript is every line the session showed, for printing back into the normal
// buffer on exit (#30 D3).
//
// The alternate screen is discarded when the tool quits, so without this a
// session's entries vanish from the terminal's history — and today
// `define arrondissement` leaves the entry where you can scroll back to it
// tomorrow or copy from it. Losing that silently is a regression a user meets
// immediately.
func (s *screen) Transcript() string {
	if len(s.lines) == 0 {
		return ""
	}
	return strings.Join(s.lines, "\n") + "\n"
}

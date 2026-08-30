package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
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
// every one of them feeds the buffer unchanged. It REPLACES crlfWriter on this
// path — both translate for a raw terminal, and two owners of line endings is
// how they drift.
//
// Write and Frame do no terminal IO. Paint is the only part that touches a
// terminal, so the arithmetic here is unit-testable with no pty.
type screen struct {
	// lines is everything the session has shown, oldest first. The LAST line may
	// be partial: deltas arrive chunked and a reply split as "one" then " two\n"
	// is one line, not two.
	lines []string
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
}

// Write appends bytes to the buffer, splitting on newlines.
//
// A bare "\r" is dropped rather than kept: it is the carriage half of a CRLF
// that arrived in a different chunk, which is the case crlfWriter documents
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

// regionsAt records the regions belonging to a run of buffer lines, so a click
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
// The second return value is false for a row below the buffer's tail — the
// prompt, the menu, or blank space — where there is nothing to click.
func (s *screen) LineAt(row int) (int, bool) {
	frame, top := s.visible()
	if row < 0 || row >= len(frame) {
		return 0, false
	}
	return top + row, true
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
// command menu closing, a wrapped prompt cleared with Ctrl-U) then left `offset`
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
	spans := append([]Region(nil), rs...)
	slices.SortFunc(spans, func(a, b Region) int { return a.Col - b.Col })

	var b strings.Builder
	col, next := 0, 0
	for i := 0; i < len(line); {
		if next < len(spans) && col == spans[next].Col {
			b.WriteString(underlineOn)
		}
		if next < len(spans) && col == spans[next].Col+spans[next].Width {
			b.WriteString(underlineOff)
			next++
			continue // re-test this column: two spans can meet
		}
		if skip := escapeLen(line[i:]); skip > 0 {
			b.WriteString(line[i : i+skip])
			i += skip
			continue
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		b.WriteString(line[i : i+size])
		col += cellWidth(r)
		i += size
	}
	// A span reaching the end of the line still closes: an unterminated
	// underline runs on through everything painted after it.
	if next < len(spans) {
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

// Paint draws one whole frame: the buffer's visible tail, then the prompt, then
// the command menu under it.
//
// The frame is redrawn WHOLE — home, clear, everything — rather than patched.
// That is the change of model this milestone buys: the editor currently tracks
// how many menu rows it drew so it can erase exactly that many, and carries a
// documented known limit for when the count is wrong ("if the menu does not fit
// below the cursor the terminal scrolls, and the cursor-up count then lands a
// row off"). A whole-frame redraw cannot be off by a row, because it never
// counts rows it drew earlier.
//
// The prompt and the menu are NOT buffer lines. They are the live edge of the
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
// So the prompt and menu are charged their REAL height, and buffer lines are
// clipped to the width — clipped at PAINT time, so the transcript and the click
// map keep the whole text.
//
// termRows and termCols are passed in rather than stored, so a resize is one
// call site's business (the loop's SIGWINCH case) and not fields that go stale.
func (s *screen) Paint(w io.Writer, termRows, termCols int, prompt string, menu []string) {
	var menuRows int
	s.cols = termCols
	// ONE row accounting, used twice: it budgets the buffer's share of the frame
	// AND says where the cursor has to walk back to. Two summations of the same
	// quantity is how the second one came to omit the prompt's own height, which
	// is the off-by-a-row limit this whole-frame redraw exists to have deleted.
	//
	// The buffer gets whatever the live edge does not need. A terminal too short
	// for even the prompt still gets the prompt: losing the line you are typing
	// is worse than losing history you can scroll to.
	// EVERY component is budgeted, not just the buffer's share. Charging the live
	// edge its height and then writing it unclipped is not a budget: a prompt or
	// a menu taller than the terminal overflows exactly as a wide buffer line
	// did, and the terminal scrolls, and every placed row moves.
	//
	// The order of sacrifice is the order of value. The prompt is the line you
	// are typing and survives first — clipped only if it alone is taller than the
	// terminal, where the alternative is a frame nobody owns. The menu is a
	// dropdown and gives up whole rows next. The buffer is scrollable, so it
	// takes what is left.
	prompt = clipVisible(prompt, termRows*max(s.cols, 1))
	promptRows := displayRows(prompt, s.cols)
	menu, menuRows = fitMenu(menu, termRows-promptRows, s.cols)
	s.rows = termRows - promptRows - menuRows
	if s.rows < 0 {
		s.rows = 0
	}
	var b strings.Builder
	b.WriteString(cursorHome + eraseDown)
	// Each painted row carries the marks for the BUFFER line it is showing, found
	// through the same mapping a click uses to go the other way — the same call,
	// so the paint and the hit test cannot be answering from different states.
	frame, top := s.visible()
	for i, line := range frame {
		b.WriteString(clipVisible(markClickable(line, s.regions[top+i]), s.cols) + "\r\n")
	}
	b.WriteString(prompt)
	for _, m := range menu {
		b.WriteString("\r\n" + m)
	}
	if len(menu) > 0 {
		// Back to the prompt's FIRST row, in the rows the terminal actually
		// moved: the menu's own height, plus the prompt's beyond its first row.
		// Counting menu ENTRIES leaves the cursor low when a row wraps; omitting
		// the prompt's height reprints it over the menu.
		fmt.Fprintf(&b, "\x1b[%dA\r", menuRows+promptRows-1)
		// And forward to the prompt's own cursor column, which the caller
		// encoded into `prompt` — reprinting it is cheaper than tracking a
		// column here and cannot disagree with what was drawn.
		b.WriteString(prompt)
	}
	// The error is DISCARDED, and deliberately: a terminal that cannot be written
	// to is a session that is already over, and the key reader's EOF is what ends
	// it. There is no recovery to attempt here and nowhere to report to — the
	// report would go to the same terminal.
	fmt.Fprint(w, b.String())
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
	rows   int
	cols   int
	prompt string
	menu   []string
	// stopped is set when the terminal has been handed back. Writes still reach
	// the buffer — the exit transcript needs them — but painting must stop dead,
	// or a farewell newline written after restore would draw a frame onto the
	// NORMAL screen, over whatever the user was looking at before define ran.
	stopped bool
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

func newLiveScreen(tty io.Writer, rows, cols int) *liveScreen {
	return &liveScreen{s: &screen{}, tty: tty, rows: rows, cols: cols, interval: paintInterval}
}

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
func (l *liveScreen) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	n, err := l.s.Write(p)
	l.throttledPaint()
	return n, err
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
func (l *liveScreen) Draw(prompt string, menu []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prompt, l.menu = prompt, menu
	l.repaint()
}

// WriteRegions is Write plus the click map for what is being written. The two
// are one call because they must not be able to disagree about which buffer line
// the render landed on — the regions are relative to the render, and only the
// screen knows where that render begins.
func (l *liveScreen) WriteRegions(text string, rs []Region) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.s.addRegions(rs)
	l.s.Write([]byte(text))
	l.throttledPaint()
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

// Page and Scroll move the viewport and show the result. The paint is the point:
// a scroll nobody can see is not a scroll.
func (l *liveScreen) Page(n int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.s.Page(n)
	l.repaint()
}

func (l *liveScreen) Scroll(lines int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.s.Scroll(lines)
	l.repaint()
}

// Resize takes SIGWINCH's whole answer (M1.4). It does NOT paint: the caller
// redraws, because the live edge is rendered against the new width too and a
// frame drawn before that is a frame drawn twice.
func (l *liveScreen) Resize(rows, cols int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rows, l.cols = rows, cols
}

// Stop ends painting. Called as the terminal is handed back, and idempotent for
// the same reason restore is: it runs from more than one exit path.
//
// It FLUSHES first: a throttled write may be waiting, and the last thing a
// session showed must not be the thing the throttle held back.
func (l *liveScreen) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.pending {
		l.repaint() // the last thing shown must not be what the throttle held
	}
	if l.timer != nil {
		l.timer.Stop()
		l.timer = nil
	}
	l.stopped = true
}

// Transcript is the buffer, for printing back into the normal buffer on exit.
func (l *liveScreen) Transcript() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s.Transcript()
}

func (l *liveScreen) repaint() {
	if l.stopped || l.tty == nil {
		return
	}
	l.s.Paint(l.tty, l.rows, l.cols, l.prompt, l.menu)
	l.painted, l.pending = time.Now(), false
}

// fitMenu drops whole menu rows from the END until the dropdown fits the space
// the prompt left, and reports what it costs.
//
// Whole rows, because half a command name is not a menu entry — and from the
// end, because the list is sorted and the first matches are the likely ones.
func fitMenu(menu []string, avail, cols int) ([]string, int) {
	used := 0
	for i, m := range menu {
		h := displayRows(m, cols)
		if used+h > avail {
			return menu[:i], used
		}
		used += h
	}
	return menu, used
}

// displayRows is how many terminal rows a line occupies once the terminal has
// wrapped it. Always at least one: an empty line is still a row.
func displayRows(line string, cols int) int {
	if cols <= 0 {
		return 1 // an unmeasurable terminal: charge one row and let it wrap
	}
	w := visibleCells(line)
	if w <= cols {
		return 1
	}
	return (w + cols - 1) / cols
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
		r, size := utf8.DecodeRuneInString(s[i:])
		w := cellWidth(r)
		if n+w > width {
			// Cut here — BEFORE the rune, so a two-cell rune is never half
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

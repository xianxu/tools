package play

// Board is form 2.5: up to sixteen words on a grid, one mark each.
//
// It is TRIAGE rather than retrieval — the learner sweeps a screenful of mature
// words saying "yes I still have that one" — which is why it implements
// SelfRated and can therefore never earn the ladder's two-rung promotion. That
// is structural rather than a rule anyone must remember (D1).
//
// It is the first form to hold MORE THAN ONE word, which is what Batch exists
// for. The session learns nothing new: it asks the capability and this answers.
//
// THE GRID IS THE LIVE EDGE, NOT A BUFFER LINE (D10). Prompt() renders it, but
// unlike every other form's prompt it is not written into the buffer once — the
// buffer is append-only, which is what makes a click's coordinates exact, and a
// grid filed there could never change as marks land. The loop calls this per
// frame and puts it in the FOOTER, which redraws. That is the whole shape of
// #40, and it is why this form owns its own geometry (see CellAt): the thing
// that decides where a word is printed is the only thing that can say which word
// was clicked.
//
// NO IMPORTS. The package's purity guard declares an empty allowlist, so the
// layout arithmetic here is hand-rolled — columnsIn counts runes by ranging,
// which is what a formatter would have been imported for.

// Mark is what a cell says, and what the board's MODE is set to.
//
// TWO marks and an ABSENCE, which is not a third mark. D7 deleted `unsure`
// ("I guess unsure means no") along with the whole EventUnsure mechanism the
// first draft had drawn around it.
//
// Unmarked is the zero value deliberately, and it is the reason this is not just
// a Verdict: a cell has three states and a verdict has no way to say "nobody has
// answered this one". Verdict's own zero is Skipped, which means something else
// entirely — a word the learner declined — and spending it on "untouched" would
// have made Rest's job unstateable.
type Mark int

const (
	Unmarked Mark = iota
	Yes
	No
	// Dropped is a cell REMOVED from the deck rather than rated (#42).
	//
	// Its Verdict is Skipped, which is what keeps it out of the schedule: a
	// removal is not an assessment, and recording one as a miss would demote a
	// word on its way out. `Rest` already skips anything not Unmarked, so Enter's
	// sweep leaves a dropped cell alone for free.
	//
	// It is a MARK rather than a fourth Verdict because Verdict is what an ANSWER
	// meant, and `schedule.Fold` reads verdicts to move boxes — "remove this
	// word" in front of that would be a category error the ladder would have to
	// branch on.
	Dropped
)

// Verdict is what a mark means to the schedule: Yes climbs the ladder, No falls
// back down it. Unmarked answers Skipped, which records nothing — that is what
// makes Ctrl-C on a half-swept board free (D3).
func (m Mark) Verdict() Verdict {
	switch m {
	case Yes:
		return Correct
	case No:
		return Wrong
	}
	return Skipped
}

// markOf is the inverse, for Rest — which takes a Verdict because that is what
// the Batch interface speaks.
//
// Everything that is not Correct becomes No, and the default direction is the
// safe one: an unmarked word asked again tomorrow costs a review, while an
// unmarked word promoted costs a word the learner has quietly stopped knowing.
// Apply's only call is Rest(Wrong); the mapping is written to be right anyway
// rather than to be reached.
func markOf(v Verdict) Mark {
	if v == Correct {
		return Yes
	}
	return No
}

// boardLabels is the printed key beside each word, in cell order.
//
// `d` IS HERE, and it took an operator sitting to see why it should be. The
// first cut skipped it — toInput takes `d` and `D` as DROP FROM DECK before any
// form sees a key — and printed `0`–`9` then `a b c e f g`. Operator,
// 2026-09-01: *"I think [d] should be used? jumping from [c] to [e] is a bit
// confusing. and I don't think in this screen we are using d in keyboard
// shortcut?"*
//
// They are right, and D12 is the reason: a form holding many words has no single
// current word, so `Apply` already REFUSED the drop here. The key did nothing on
// this screen, and the sequence carried a hole to protect a key that was not in
// use. Sixteen labels, no gap — and the session's rule survives in the sharper
// form it always had: **`d` is reserved for forms that HAVE a current word to
// remove.** See Apply's InputDrop case, which hands the key to the form instead.
const boardLabels = BoardLabels

// BoardLabels is the printed sequence, exported so a caller types a key by
// DERIVING it rather than by restating the string.
//
// The restatement is what broke when `d` was added back: two tests and a pty row
// each carried their own copy of "0123456789abcefg", and they typed a key the
// board no longer had. Same failure the README prompt lines are guarded against,
// one package over.
const BoardLabels = "0123456789abcdef"

// MaxBoardWords is the capacity, and it is the label alphabet's length rather
// than a number chosen beside it — a seventeenth cell would have no key, so a
// mouse-less terminal could not reach it.
const MaxBoardWords = len(boardLabels)

const (
	// labelWidth is `[x] ` — the bracketed key and one space.
	labelWidth = 4
	// boardGutter separates one column from the next. The LAST column pays none,
	// which is the +boardGutter in the column-count arithmetic.
	boardGutter = 2
	// boardCols is the widest grid offered however wide the terminal is.
	//
	// Four because the operator's sketch is a 4x4, and because sixteen cells on
	// one line stop being a grid — the eye scans a block, not a ribbon. A wide
	// terminal spends its extra columns on longer words instead.
	boardCols = 4
)

// Cell is one word on the board and the ONE-LINE gloss shown when it is marked.
//
// The gloss arrives finished, exactly as Choice takes rendered Options and
// Choice takes rendered Options: extracting a sense, reading NOAD's labels
// and rejecting an entry that defines a different word all need the dictionary,
// and this package imports nothing.
//
// A one-line gloss rather than the whole entry, and that is the live edge's
// constraint rather than a preference — a board is offered only when the
// terminal can draw it WHOLE, so a panel that could be twenty rows tall would
// mean a board nobody's terminal fits.
type Cell struct {
	Word  string
	Gloss string
}

// Palette is how a mark is PAINTED, supplied by the caller.
//
// The form draws its own live edge (R1) and `main` owns the terminal's colours
// (`newPalette`), so the sequences arrive finished — the same seam `Choice` sits
// on, where the caller does the rendering and the form takes the result. The
// zero value styles nothing, which is what every test not about colour wants.
//
// Escape sequences cost NO COLUMNS, and this file's arithmetic must not count
// them: the padding is computed from the plain word and applied OUTSIDE the
// style, so a styled cell occupies exactly the columns an unstyled one does.
type Palette struct {
	Yes  string // starts the style for a cell marked yes
	No   string // ...and for one marked no
	Drop string // ...and for one being removed from the deck
	Off  string // ends any of them
}

// Board is one grid. Pointer receivers: it REMEMBERS every mark, which is the
// whole of what makes it a batch.
type Board struct {
	cells []Cell
	marks []Mark
	// mode is the mark a click or a labelled key lands, flipped by Tab.
	//
	// Yes by default: on a board of mature words most cells are a yes, so the
	// default is the common case and the toggle is for the exceptions.
	mode Mark
	// pal paints the marks. COLOUR rather than a changed label, which is the
	// operator's correction after a real sitting: *"selected words on the board
	// should change color instead of use [y]/[n]"*, and *"don't change the
	// [0]...[f] as they are needed for keyboard operation"*. The first cut put
	// the mark WHERE THE KEY WAS — saying "answered" by taking away the thing
	// the keyboard needs, on the one path a mouse-less terminal has.
	pal Palette
	// dropping is set by a mark landed in Dropped mode and cleared by the reader,
	// which is what makes Dropped() one-shot.
	dropping bool
	// last is the cell most recently marked, and it is what Word() reports.
	// advance builds Outcome{Word: q.Word()} at a call site that knows nothing
	// about grids, so the form has to answer "which word did that just mean".
	last int

	// The layout, derived from the width the board is DRAWN at — which changes,
	// and the first draft said it could not (R9).
	//
	// D15 read "a resize below the board's height leaves the current board drawn
	// as it was" as safe. It is not: a board laid out for eighty columns has
	// 74-column rows, and at forty the terminal wraps each into two — so a footer
	// entry stops being one physical row, and a click on the continuation row
	// arrives with a column that means something else entirely. The mark is
	// permanent. Resize is what keeps this honest.
	width     int
	cols      int
	cell      int // columns from one cell's start to the next cell's start
	wordCells int // columns a word is given inside a cell
}

// NewBoard takes the words and the WIDTH they must fit in.
//
// The width is a parameter rather than something this package could ask for,
// which is the same seam Choice sits on: prose and terminals belong to the
// caller, and a form takes the finished dimensions. It is used to choose the
// column count, so that Prompt() can guarantee no line wider than width — and
// that guarantee is what D15's whole argument rests on. A footer row that wraps
// is a frame one row too tall, the terminal scrolls to fit it, and every row the
// app believes it placed moves; a click at viewport row R stops meaning what the
// board drew there.
//
// More than MaxBoardWords is CAPPED rather than rejected, and nothing is lost by
// it: a word this board does not ask about gets no event, so its box does not
// move and it is due again tomorrow. The caller (boardsFor) packs in
// MaxBoardWords chunks and is pinned there; this is the belt.
func NewBoard(cells []Cell, width int, pal Palette) *Board {
	if len(cells) > MaxBoardWords {
		cells = cells[:MaxBoardWords]
	}
	b := &Board{cells: cells, marks: make([]Mark, len(cells)), mode: Yes, width: width, pal: pal}
	b.layout()
	return b
}

// layout fixes the column count and the room a word gets.
//
// The word column is as wide as the LONGEST word on this board, so the grid is
// aligned and a click lands where the eye says it should. Ragged columns pack
// more words per line and would make CellAt a search rather than arithmetic;
// with at most sixteen short cells the packing is not worth the seam.
func (b *Board) layout() {
	longest := 0
	for _, c := range b.cells {
		if n := columnsIn(c.Word); n > longest {
			longest = n
		}
	}
	// A single column must fit whatever the terminal is, so the word's room is
	// clamped and Prompt truncates into it. This only bites on a terminal too
	// narrow to be offered a board at all (D15) — it is the floor that keeps the
	// no-line-wider-than-width guarantee true rather than a case worth designing
	// for.
	room := b.width - labelWidth
	if room < 1 {
		room = 1
	}
	b.wordCells = longest
	if b.wordCells > room {
		b.wordCells = room
	}
	b.cell = labelWidth + b.wordCells + boardGutter
	// c columns occupy c*cell - gutter, because the last one pays no gutter.
	b.cols = (b.width + boardGutter) / b.cell
	if b.cols > boardCols {
		b.cols = boardCols
	}
	if b.cols > len(b.cells) {
		b.cols = len(b.cells)
	}
	if b.cols < 1 {
		b.cols = 1
	}
}

// Resize lays the board out again for the width it will now be DRAWN at.
//
// THE INVARIANT THIS DEFENDS: every line Prompt() produces fits the terminal, so
// every footer entry is exactly one physical row, so the entry index the screen
// reports for a click IS a grid row and the column it carries is in this board's
// own coordinate space. Let the width drift and all three of those stop being
// true at once — silently, and the symptom is a permanent mark on the wrong word.
//
// The MARKS SURVIVE, which is the whole reason this is a relayout rather than a
// new board: those answers are already in the event log and cannot be retracted.
// Only the geometry moves — the columns, and therefore how many rows the same
// cells occupy.
func (b *Board) Resize(cols int) {
	if cols == b.width {
		return
	}
	b.width = cols
	b.layout()
}

// Rows is how many lines Prompt() produces — the WHOLE live edge, the grid and
// the chrome under it together.
//
// Exported because it is the number D15's fit test is about: a board is offered
// only when the terminal can hold it whole, and the board is the only thing that
// knows how tall it is. That it counts the CHROME too is the point — a fit
// computed from the grid alone would put the last row off the bottom — and
// `chromeRows` below is the one place that says what the chrome IS. Naming the
// rows here was a second owner, and it went stale the day R11 deleted one.
func (b *Board) Rows() int {
	return b.gridRows() + chromeRows
}

// gridRows is the grid's own share: the rows CellAt can find a word on.
func (b *Board) gridRows() int {
	return (len(b.cells) + b.cols - 1) / b.cols
}

// chromeRows is what Prompt draws under the grid: a blank and the panel.
//
// The panel row is drawn EVEN WHEN EMPTY, which is not tidiness. A row that
// appeared with the first mark would shift the grid up by one, and every word
// would move under a pointer already resting on it.
//
// The TOGGLE used to be a third, and it moved to the prompt row (R11): the
// footer drops rows from the end, so the one statement of what a click will mean
// was the first thing a short terminal lost. What is left below the grid is the
// panel, whose loss is cosmetic — and then grid rows, which are visible when
// missing and are not clickable when undrawn.
const chromeRows = 2

// Word is the word the LAST mark landed on.
//
// This is what lets advance keep its existing `Outcome{Word: q.Word()}` with no
// change at the call site: the session records a verdict against a word, and on
// a grid the form is the only thing that knows which one that was.
func (b *Board) Word() string {
	if len(b.cells) == 0 {
		return ""
	}
	return b.cells[b.last].Word
}

// Prompt is the labelled grid, rebuilt every frame.
//
// A marked cell is PAINTED and keeps its key. The first cut put the mark where
// the key was — `[y]` in place of `[3]` — reasoning that it said "answered" and
// "this key is spent" at once. An operator sitting corrected it: the key is how
// a mouse-less terminal reaches the cell, and it is also how a learner reads the
// grid back, so taking it away is the wrong half to spend. Colour says answered;
// the key stays put.
//
// COLOUR IS WHY THE GRID IS THE LIVE EDGE (D10). A grid filed in the append-only
// buffer could never change, so a mark that repaints an existing cell would be
// impossible there — the footer is what makes it expressible at all, and this is
// the feature that spends that.
//
// The sequences come from `main` through Palette: escape codes cost no columns,
// and the padding is applied outside them, so nothing here has to measure a
// styled string. ASCII text throughout otherwise — `✓`/`✗` are East Asian
// Ambiguous, so some terminals give them two columns, and a cell one column
// wider than the board believes is exactly the failure D15 is written against.
func (b *Board) Prompt() string {
	// BUILT AS A SLICE, so len(lines) IS Rows() rather than merely equalling it.
	//
	// Concatenation got this wrong for an empty board: with no grid rows the
	// separator's "\n\n" produced two blanks instead of one, so Prompt yielded
	// four lines where Rows() said three. Unreachable — boardsFor never builds an
	// empty board — but Word() and panelLine() both defend the empty case, and an
	// invariant held in three places and dropped in a fourth is worse than one
	// held nowhere.
	lines := make([]string, 0, b.Rows())
	for r, rows := 0, b.gridRows(); r < rows; r++ {
		line := ""
		for c := 0; c < b.cols; c++ {
			i := r*b.cols + c
			if i >= len(b.cells) {
				break
			}
			if c > 0 {
				line += spaces(boardGutter)
			}
			line += b.cellText(i)
		}
		// The last cell on a row is padded to the column width like every other,
		// and trailing blanks on a footer row are columns the screen has to
		// erase for nothing.
		lines = append(lines, trimRight(line))
	}
	// THE CHROME IS THE FORM'S TOO, and that is why it is here rather than
	// assembled by the loop out of Mode() and a gloss. A form owns how it looks —
	// Choice owns its option layout for the same reason — and the loop assembling
	// it would make the board's appearance a thing two files agree about, on the
	// surface where disagreeing marks the wrong word. All the loop adds is the
	// bar, which belongs to the sitting rather than to this question.
	lines = append(lines, "", b.panelLine())
	return joinLines(lines)
}

// panelLine is the last-marked word and its gloss: the feedback moment, in the
// place a definition would be on any other form.
//
// EMPTY UNTIL SOMETHING IS MARKED, and empty is still a row — see chromeRows.
// Truncated to the width like everything else here, because the whole live edge
// has to fit the terminal the board was offered for.
func (b *Board) panelLine() string {
	if len(b.cells) == 0 || b.marks[b.last] == Unmarked {
		return ""
	}
	c := b.cells[b.last]
	if c.Gloss == "" {
		return c.Word
	}
	return truncate(c.Word+"  "+c.Gloss, b.width)
}

// cellText is one cell: its key, then its word — PAINTED once it is marked.
//
// The key never changes, because the key is how a mouse-less terminal reaches
// the cell, and taking it away at the moment of marking is the wrong half to
// give up. The padding sits OUTSIDE the style, so a styled cell occupies exactly
// the columns an unstyled one does and the layout arithmetic never sees an
// escape sequence.
func (b *Board) cellText(i int) string {
	word := truncate(b.cells[i].Word, b.wordCells)
	text := "[" + string(boardLabels[i]) + "] " + word
	if on := b.paint(i); on != "" {
		text = on + text + b.pal.Off
	}
	return text + spaces(b.wordCells-columnsIn(word))
}

// paint is the sequence that starts this cell's style: empty for an unmarked
// cell, and for a board built with no palette.
func (b *Board) paint(i int) string {
	// THROUGH Palette.For, which is the one owner of mark → sequence. Spelling the
	// mapping again here would be a second owner, and the failure is silent: a
	// mark painted one way in the form and another in the test that checks it.
	return b.pal.For(b.marks[i])
}

// Marked reports how cell i is marked, for a caller that must see the state
// without reading colour back out of a string.
func (b *Board) Marked(i int) Mark {
	if i < 0 || i >= len(b.marks) {
		return Unmarked
	}
	return b.marks[i]
}

// Reveal is EMPTY, and that is the honest answer rather than a stub.
//
// A board hides nothing — every word is on screen from the first frame, and the
// mark is the learner's own report about a word they are looking at. There is no
// definition to earn by missing one, which is also why Apply must not route a No
// through the miss-on-a-hidden-word branch (D12): that branch sets Graded, and
// the board would freeze after its first No.
func (b *Board) Reveal() string { return "" }

// Keys names the board's keys AND which mark is live, and it is the one owner of
// both (R11).
//
// THE MODE LIVES HERE BECAUSE THIS ROW SURVIVES. It had its own footer row, on
// the reasoning that the live edge is where things that change belong — and a
// resize measured what that costs: `fitFooter` drops footer rows from the END,
// so a terminal too short after a narrowing dropped the panel and then the
// TOGGLE, leaving the board on screen with no statement of what the next click
// would mean while every mark is irreversible. `Paint` clips the PROMPT last and
// only when it alone exceeds the terminal, so the mode is knowable for as long
// as anything is.
//
// One owner either way — the mode MOVED here, it was not copied here. What
// changed is which row states it, and the reason is which row survives.
//
// Tab and Enter ARE named here even though both are session Input kinds, and
// that is deliberate: sessionKeys in the loop is the set that is true WHATEVER
// form is asking, and neither of these is. Enter finishes a form only when the
// form holds many words, and Tab reaches nothing at all on 2.1 or 2.3. The form
// is the only thing that can describe them truthfully.
//
// The label set is NOT enumerated. It is printed beside every word, so spelling
// it out here would be a second owner of the sequence and a harder thing to read
// than the grid itself. (This used to say the set "has a hole at `d`" — it had
// one until #40's last round filled it, and the sentence outlived the hole.)
//
// Both spellings are the SAME WIDTH, so the line does not jump under a key
// pressed to be pressed again — and short enough that this plus the session's
// reserved keys fits eighty columns.
func (b *Board) Keys() string {
	// RE-CUT TO FIT, not appended to (#42). `boardFitsIn` charges
	// `displayRows(gradePrompt(q), termCols)` into the board's fit, so a row that
	// wraps at eighty columns raises the minimum terminal height for EVERY board.
	// The old row was 62 columns and left two of headroom; adding a third state
	// naively would have wrapped, so "Tab switches" became "Tab cycles" and
	// "click or key marks" lost its verb — with three modes a click no longer
	// only marks. 59 columns, 75 with the reserved key.
	switch b.mode {
	case No:
		return "marking yes [no] drop, Tab cycles, click or key, Enter ends"
	case Dropped:
		return "marking yes no [drop], Tab cycles, click or key, Enter ends"
	}
	return "marking [yes] no drop, Tab cycles, click or key, Enter ends"
}

// Mode is the mark a click will land. Not on any interface — the form states its
// own mode, on its prompt row — but the tests and the panel's story both read
// better for it.
func (b *Board) Mode() Mark { return b.mode }

// Form names this form in the log (#40 D4a), and this is the name the whole
// field exists for: a board PROMOTES on self-report, and the query that will
// eventually decide whether that is too generous joins these events to the
// word's next real test.
func (b *Board) Form() string { return "board" }

// Toggle cycles the mode. This is what Tab means on a board.
//
// THREE now, and the ORDER is a UX decision rather than arithmetic on the iota
// (#42): Yes → No → Dropped, so the destructive mode is never one press from the
// default. A learner reaching for `no` cannot overshoot into a removal, and the
// mode they most often want is the one they start in.
//
// Written as a switch for that reason — `(b.mode % 3) + 1` would be shorter and
// would hide the decision, and the day a fourth mode arrives the order question
// has to be asked again rather than answered by an increment.
func (b *Board) Toggle() {
	switch b.mode {
	case Yes:
		b.mode = No
	case No:
		b.mode = Dropped
	default:
		b.mode = Yes
	}
}

// Grade marks the cell whose printed key is k.
//
// It resolves the key to a cell and hands it to Mark, because a key and a click
// are THE SAME ACT reached two ways. Two implementations would be two chances to
// disagree, and the one that would rot is the keyboard path — it is the one a
// mouse-owning developer never presses, and it is the one a mouse-less terminal
// has (Done-when 6).
//
// Case-insensitive, as every key path here is. AND `d` DOES ARRIVE: the session
// reserves that key only for forms with a single current word to remove, and
// hands it to a grid as an ordinary cell key (#40 D12) — this comment used to
// claim the opposite ("`d` and `D` never arrive… boardLabels has no cell for them
// either way"), which #40's own final round falsified when it put `d` back in the
// alphabet. Corrected in #42, the issue about that key.
func (b *Board) Grade(k rune) (Verdict, bool) {
	if k >= 'A' && k <= 'Z' {
		k += 'a' - 'A'
	}
	for i := 0; i < len(boardLabels); i++ {
		if rune(boardLabels[i]) == k {
			return b.Mark(i)
		}
	}
	return Skipped, false
}

// Mark lands the active mode on cell i, and it is what a click becomes.
//
// It takes no verdict: the MODE decides, and the mode is the board's own state —
// it is drawn on the prompt row (`Keys`) and flipped by Tab. A caller passing a
// verdict in would be a second owner of a fact this form already renders, and
// the two would disagree the first time a frame was drawn between the prompt and
// the click.
//
// A CELL IS MARKED ONCE. A second mark returns false and changes nothing,
// because the first one is already in the event log — every mark emits its
// OutcomeRecord as it lands, which is what makes Ctrl-C lossless (D3), and the
// price of writing immediately is that nothing can be taken back. schedule.Fold
// would read the pair as two reviews of one word on one day. Apply states the
// same rule for single-word forms: "A second assessment is not on offer".
func (b *Board) Mark(i int) (Verdict, bool) {
	if i < 0 || i >= len(b.cells) || b.marks[i] != Unmarked {
		return Skipped, false
	}
	b.marks[i] = b.mode
	b.last = i
	// The drop is ARMED here and disarmed by the reader, so it is one-shot: the
	// loop asks `Dropped()` on every mark, and a sticky answer would remove the
	// same word again on the next keystroke — against a word already gone.
	b.dropping = b.mode == Dropped
	return b.mode.Verdict(), true
}

// Dropped is the word the last mark asked to REMOVE, once (#42).
//
// It implements the session's `Dropping` capability rather than being reached by
// a type switch on *Board — the same shape as `Missed` and `SelfRated`, and the
// reason is `#6`'s Done-when: a session that knew what a board IS would have to
// change for every future form.
//
// ONE-SHOT, and that is the whole of its correctness. `Apply` asks after each
// mark lands; an answer that persisted would re-emit the removal on the next Tab
// or refused click, performing one act twice.
func (b *Board) Dropped() (string, bool) {
	if !b.dropping {
		return "", false
	}
	b.dropping = false
	return b.cells[b.last].Word, true
}

// CellAt is which cell a click landed on, given a position INSIDE the grid
// block: row 0 is the grid's first line, column 0 its first column. The caller
// subtracts wherever it drew the block.
//
// The board answers this because the board decided the layout. The alternative —
// exporting the column count and the cell width so the loop could do the
// arithmetic — is two owners of one measurement, and the failure mode is a click
// that marks the word next to the one under the pointer.
//
// THE GUTTER IS NOT A TARGET. A forgiving hit box is the usual kindness, and it
// is wrong here: a mark cannot be taken back (see Mark), so a click that is not
// clearly on a word must do nothing rather than mark its neighbour.
//
// THE CELL'S OWN PADDING IS, and that is the deliberate other half. A short word
// in a column sized for a long one leaves blanks after it, and a click there
// marks that word — never a neighbour, because the gutter still separates them.
// It reads as clicking blank space, so it is written down: the alternative is a
// target that changes width with whatever else happens to be on the board.
func (b *Board) CellAt(row, col int) (int, bool) {
	if row < 0 || col < 0 {
		return 0, false
	}
	c := col / b.cell
	if c >= b.cols || col-c*b.cell >= labelWidth+b.wordCells {
		return 0, false
	}
	i := row*b.cols + c
	// NO SEPARATE `row >= gridRows` BOUND, and its absence is the honest kind.
	// A row below the grid — the chrome, whatever `chromeRows` currently draws —
	// indexes past the last cell by construction, because the grid has exactly as
	// many rows as it takes to hold them all. The guard was written, and a mutation showed it
	// could not be made to fail: it was dead code, and dead code here would hide
	// the day this line stopped being the one that answers.
	// TestTheChromeRowsAreNotCells is the property, pinned separately from the
	// mechanism.
	if i >= len(b.cells) {
		return 0, false
	}
	return i, true
}

// IsSelfRated: a mark is the learner's CLAIM about a word they are looking at,
// and nothing checked it. That is what keeps a board from ever producing the
// two-rung promotion, and it is the form's structure rather than a rule (D1).
func (b *Board) IsSelfRated() bool { return true }

// Words is how many this board holds, which is what the bar counts (D8).
func (b *Board) Words() int { return len(b.cells) }

// Spent reports that every word here has been answered — the Batch question the
// session asks before it advances.
func (b *Board) Spent() bool {
	for _, m := range b.marks {
		if m == Unmarked {
			return false
		}
	}
	return true
}

// Rest answers every word still unmarked and names them, which is what Enter
// means to a form holding many: "I am out of time, ask me all of these again".
//
// Sixteen demotions from one keystroke, which is why Enter had to be split from
// space (D14) — it is the only expensive-to-undo action on this surface, and it
// was on the most careless key there is.
func (b *Board) Rest(v Verdict) []string {
	m := markOf(v)
	var rest []string
	for i := range b.marks {
		if b.marks[i] == Unmarked {
			b.marks[i] = m
			b.last = i
			rest = append(rest, b.cells[i].Word)
		}
	}
	return rest
}

// visibleColumns is columnsIn ignoring ANSI escape sequences, for reading a
// DRAWN line back — a styled cell's colour costs no columns, and anything that
// measures the grid has to agree with the terminal about that.
//
// Nothing in the layout needs it: the padding is computed from plain words and
// applied outside the style, so this file never measures a styled string. It is
// here for callers and tests that hold a finished line.
//
// A minimal recogniser — ESC, then anything up to a letter — because that is the
// whole grammar this package emits: it emits none at all, and only ever passes
// through what Palette was handed. main.escapeLen is the real one, and it is on
// the other side of a purity guard.
func visibleColumns(s string) int {
	n, inEsc := 0, false
	for _, r := range s {
		switch {
		case inEsc:
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
		case r == 0x1b:
			inEsc = true
		default:
			n++
		}
	}
	return n
}

// columnsIn counts a string's runes, which is its width for the Latin-script
// words a deck holds. Ranging a string decodes UTF-8 natively, so this needs no
// import — which is what the purity guard's empty allowlist requires.
func columnsIn(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// truncate cuts a word to n columns, marking the cut so a clipped word does not
// read as a shorter real one.
func truncate(s string, n int) string {
	if columnsIn(s) <= n {
		return s
	}
	if n <= 1 {
		return "~"
	}
	out, i := "", 0
	for _, r := range s {
		if i == n-1 {
			break
		}
		out += string(r)
		i++
	}
	return out + "~"
}

func pad(s string, n int) string { return s + spaces(n-columnsIn(s)) }

func spaces(n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += " "
	}
	return out
}

// joinLines is strings.Join with "\n". Hand-rolled: this package imports nothing.
func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}

func trimRight(s string) string {
	end := len(s)
	for end > 0 && s[end-1] == ' ' {
		end--
	}
	return s[:end]
}

// Marks is every mark a cell can carry, in cycle order, and it is the EXTENT of
// the set (#42).
//
// Exported so a caller enumerates them rather than restating them — the same move
// `BoardLabels` made for the key sequence and `numRegionKinds` made for the click
// registry. A palette that a test checked by listing three fields would say
// nothing about a fourth mark; deriving the loop from here means a new mark
// arrives already covered, or fails loudly.
func Marks() []Mark { return []Mark{Unmarked, Yes, No, Dropped} }

// For is the sequence that paints this mark, and the Palette is the one owner of
// that mapping — so a caller asking "how is a drop drawn" cannot answer it from a
// field it picked itself.
func (p Palette) For(m Mark) string {
	switch m {
	case Yes:
		return p.Yes
	case No:
		return p.No
	case Dropped:
		return p.Drop
	}
	return ""
}

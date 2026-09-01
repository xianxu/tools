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
// `d` IS ABSENT, and its absence is load-bearing. toInput takes `d` and `D` as
// DROP FROM DECK before any form sees the key, and Question.Grade's doc states
// the rule: "The session RESERVES some keys before a form ever sees them… A form
// must not build its answer set from those." A board labelled with `d` would
// have a cell whose key silently removed a word from the deck instead.
//
// The gap is visible rather than surprising, because the labels are PRINTED
// beside the words — nobody has to know the sequence, they read it (D5).
const boardLabels = "0123456789abcefg"

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

// Board is one grid. Pointer receivers: it REMEMBERS every mark, which is the
// whole of what makes it a batch.
type Board struct {
	words []string
	marks []Mark
	// mode is the mark a click or a labelled key lands, flipped by Tab.
	//
	// Yes by default: on a board of mature words most cells are a yes, so the
	// default is the common case and the toggle is for the exceptions.
	mode Mark
	// last is the cell most recently marked, and it is what Word() reports.
	// advance builds Outcome{Word: q.Word()} at a call site that knows nothing
	// about grids, so the form has to answer "which word did that just mean".
	last int

	// The layout, computed once in NewBoard because the width it is derived from
	// cannot change: a resize below the board's height leaves the current board
	// drawn as it was and chooses the NEXT question at the new size (D15).
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
func NewBoard(words []string, width int) *Board {
	if len(words) > MaxBoardWords {
		words = words[:MaxBoardWords]
	}
	b := &Board{words: words, marks: make([]Mark, len(words)), mode: Yes, width: width}
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
	for _, w := range b.words {
		if n := columnsIn(w); n > longest {
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
	if b.cols > len(b.words) {
		b.cols = len(b.words)
	}
	if b.cols < 1 {
		b.cols = 1
	}
}

// Rows is how many grid lines Prompt() produces.
//
// Exported because it is the number D15's fit test is about: a board is offered
// only when the terminal can hold its grid whole, and the board is the only
// thing that knows how tall it is.
func (b *Board) Rows() int {
	return (len(b.words) + b.cols - 1) / b.cols
}

// Word is the word the LAST mark landed on.
//
// This is what lets advance keep its existing `Outcome{Word: q.Word()}` with no
// change at the call site: the session records a verdict against a word, and on
// a grid the form is the only thing that knows which one that was.
func (b *Board) Word() string {
	if len(b.words) == 0 {
		return ""
	}
	return b.words[b.last]
}

// Prompt is the labelled grid, rebuilt every frame.
//
// A marked cell shows its MARK WHERE ITS KEY WAS — `[y]` or `[n]` in place of
// `[3]` — which says two things at once: this word is answered, and that key no
// longer does anything. The refusal in Mark is otherwise silent, and a key that
// stops working without saying so is the kind of thing a learner blames
// themselves for.
//
// ASCII rather than ✓/✗, and the reason is geometry rather than taste: those
// runes are East Asian Ambiguous, so some terminals give them two columns. A
// cell one column wider than the board believes is exactly the failure D15 is
// written against — a click that lands on the wrong word. Colour is what makes
// the marks pop, and the loop adds it: keeping the grid in the live edge is what
// bought colour in the first place (D10).
func (b *Board) Prompt() string {
	var s string
	for r, rows := 0, b.Rows(); r < rows; r++ {
		if r > 0 {
			s += "\n"
		}
		line := ""
		for c := 0; c < b.cols; c++ {
			i := r*b.cols + c
			if i >= len(b.words) {
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
		s += trimRight(line)
	}
	return s
}

// cellText is one cell: the key or the mark, then the word padded to the column.
func (b *Board) cellText(i int) string {
	return "[" + string(b.glyph(i)) + "] " + pad(truncate(b.words[i], b.wordCells), b.wordCells)
}

// glyph is what stands in the brackets: the mark once there is one, the key
// until then.
func (b *Board) glyph(i int) rune {
	switch b.marks[i] {
	case Yes:
		return 'y'
	case No:
		return 'n'
	}
	return rune(boardLabels[i])
}

// Reveal is EMPTY, and that is the honest answer rather than a stub.
//
// A board hides nothing — every word is on screen from the first frame, and the
// mark is the learner's own report about a word they are looking at. There is no
// definition to earn by missing one, which is also why Apply must not route a No
// through the miss-on-a-hidden-word branch (D12): that branch sets Graded, and
// the board would freeze after its first No.
func (b *Board) Reveal() string { return "" }

// Keys names the board's own keys and the MODE they will land, live.
//
// Tab and Enter are named here even though both are session Input kinds, and
// that is deliberate: sessionKeys in the loop is the set that is true WHATEVER
// form is asking, and neither of these is. Enter finishes a form only when the
// form holds many words, and Tab reaches nothing at all on 2.1 or 2.3. The form
// is the only thing that can describe them truthfully.
//
// The label set is NOT enumerated. It has a hole at `d` and it is printed beside
// every word, so spelling "0-9 a-c e-g" here would be a second owner of the
// sequence and a harder thing to read than the grid itself.
func (b *Board) Keys() string {
	if b.mode == Yes {
		return "key beside a word = yes, Tab = switch to no, Enter = finish"
	}
	return "key beside a word = no, Tab = switch to yes, Enter = finish"
}

// Mode is the mark a click will land, for the footer's toggle line.
func (b *Board) Mode() Mark { return b.mode }

// Toggle flips the mode. This is what Tab means on a board.
func (b *Board) Toggle() {
	if b.mode == Yes {
		b.mode = No
		return
	}
	b.mode = Yes
}

// Grade marks the cell whose printed key is k.
//
// It resolves the key to a cell and hands it to Mark, because a key and a click
// are THE SAME ACT reached two ways. Two implementations would be two chances to
// disagree, and the one that would rot is the keyboard path — it is the one a
// mouse-owning developer never presses, and it is the one a mouse-less terminal
// has (Done-when 6).
//
// Case-insensitive, as form 2.1 is. `d` and `D` never arrive: toInput takes them
// first, and boardLabels has no cell for them either way.
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
// it is drawn in the footer's toggle and flipped by Tab. A caller passing a
// verdict in would be a second owner of a fact this form already renders, and
// the two would disagree the first time a frame was drawn between the toggle and
// the click.
//
// A CELL IS MARKED ONCE. A second mark returns false and changes nothing,
// because the first one is already in the event log — every mark emits its
// OutcomeRecord as it lands, which is what makes Ctrl-C lossless (D3), and the
// price of writing immediately is that nothing can be taken back. schedule.Fold
// would read the pair as two reviews of one word on one day. Apply states the
// same rule for single-word forms: "A second assessment is not on offer".
func (b *Board) Mark(i int) (Verdict, bool) {
	if i < 0 || i >= len(b.words) || b.marks[i] != Unmarked {
		return Skipped, false
	}
	b.marks[i] = b.mode
	b.last = i
	return b.mode.Verdict(), true
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
func (b *Board) CellAt(row, col int) (int, bool) {
	if row < 0 || row >= b.Rows() || col < 0 {
		return 0, false
	}
	c := col / b.cell
	if c >= b.cols || col-c*b.cell >= labelWidth+b.wordCells {
		return 0, false
	}
	i := row*b.cols + c
	if i >= len(b.words) {
		return 0, false
	}
	return i, true
}

// IsSelfRated: a mark is the learner's CLAIM about a word they are looking at,
// and nothing checked it. That is what keeps a board from ever producing the
// two-rung promotion, and it is the form's structure rather than a rule (D1).
func (b *Board) IsSelfRated() bool { return true }

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
			rest = append(rest, b.words[i])
		}
	}
	return rest
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

func trimRight(s string) string {
	end := len(s)
	for end > 0 && s[end-1] == ' ' {
		end--
	}
	return s[:end]
}

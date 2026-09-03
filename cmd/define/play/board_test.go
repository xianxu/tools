package play

import (
	// strings in a TEST is not a purity violation: puretest reads .Imports, not
	// .TestImports (choice_test.go says the same).
	"strings"
	"testing"
)

// The three the session asks about. A compile-time assertion rather than a test
// body, because "does *Board satisfy Batch" is a question the compiler answers
// better than an assertion can.
var (
	_ Question  = (*Board)(nil)
	_ Batch     = (*Board)(nil)
	_ SelfRated = (*Board)(nil)
	_ Moded     = (*Board)(nil)
)

// cellsOf is a board's worth of words with no glosses. The panel is pinned on
// its own; everywhere else the gloss is not what is under test.
func cellsOf(words ...string) []Cell {
	cs := make([]Cell, len(words))
	for i, w := range words {
		cs[i] = Cell{Word: w}
	}
	return cs
}

// sixteen is a full board, and the words vary in length so the layout has
// something to align.
var sixteen = []string{
	"arrondissement", "bailiwick", "keel", "mesa",
	"ephemeral", "quokka", "potassium", "ligament",
	"sycophantic", "concrete", "parrot", "run",
	"light", "bank", "set", "obsequious",
}

// `d` IS A LABEL, and the sequence has no hole in it.
//
// It was skipped, because `toInput` takes `d` as drop-from-deck before any form
// sees a key. An operator reading a real grid found the jump from `[c]` to `[e]`
// confusing, and they were right that nothing on this screen uses the key: D12
// had already REFUSED the drop for a form holding many words, because a grid has
// no single current word to remove. So the hole protected a key that was not in
// use, and the session's rule is the sharper one it always was — `d` is reserved
// for forms that HAVE a current word.
func TestBoardLabelsRunWithoutAGap(t *testing.T) {
	b := NewBoard(cellsOf(sixteen...), 100, Palette{})

	if want := "0123456789abcdef"; boardLabels != want {
		t.Errorf("boardLabels = %q, want %q — sixteen in sequence", boardLabels, want)
	}
	if MaxBoardWords != 16 {
		t.Errorf("MaxBoardWords = %d, want 16 — the label alphabet is the capacity", MaxBoardWords)
	}
	// EVERY printed label reaches its cell, which is the other half of
	// Done-when 6: a mouse-less terminal must be able to mark every word.
	for i := range sixteen {
		fresh := NewBoard(cellsOf(sixteen...), 100, Palette{})
		if _, ok := fresh.Grade(rune(boardLabels[i])); !ok {
			t.Errorf("Grade(%q) did not reach cell %d", boardLabels[i], i)
		}
	}
	// ...including `d`, which is cell 13.
	if v, ok := b.Grade('d'); !ok || v != Correct {
		t.Errorf("Grade('d') = (%v, %v), want it to mark cell 13", v, ok)
	}
	if got := b.Word(); got != sixteen[13] {
		t.Errorf("`d` marked %q, want %q", got, sixteen[13])
	}
}

// The keys this form grades, and nothing the session reserved
// (question.go:74-77).
func TestBoardGradesOnlyItsOwnKeys(t *testing.T) {
	for _, tc := range []struct {
		key    rune
		wantOK bool
	}{
		{'0', true},
		{'9', true},
		{'a', true},
		{'A', true}, // case-insensitive, as form 2.1 is
		{'f', true},
		{'d', true},  // cell 13 — the session reserves `d` only where a form has
		{'D', true},  // a current word to drop, which a grid does not
		{'g', false}, // past the alphabet
		{'h', false},
		{'y', false}, // form 2.1's keys are not this form's
		{'\r', false},
		{' ', false},
		{0x03, false},
	} {
		b := NewBoard(cellsOf(sixteen...), 100, Palette{})
		v, ok := b.Grade(tc.key)
		if ok != tc.wantOK {
			t.Errorf("Grade(%q) = (%v, %v), want ok=%v", tc.key, v, ok, tc.wantOK)
		}
		if !ok && v != Skipped {
			t.Errorf("Grade(%q) returned verdict %v on a key it does not use, want Skipped", tc.key, v)
		}
	}
}

// A key and a click are THE SAME ACT, and the pin is that they produce the same
// state — not that they call the same function, which a refactor could undo.
//
// This is the row that keeps the keyboard path from rotting: it is the path a
// mouse-owning developer never presses and the only path a mouse-less terminal
// has.
func TestAKeyAndAClickAreTheSameAct(t *testing.T) {
	for i := range sixteen {
		byKey, byClick := NewBoard(cellsOf(sixteen...), 100, Palette{}), NewBoard(cellsOf(sixteen...), 100, Palette{})
		kv, kok := byKey.Grade(rune(boardLabels[i]))
		cv, cok := byClick.Mark(i)
		if kv != cv || kok != cok {
			t.Errorf("cell %d: key = (%v,%v), click = (%v,%v)", i, kv, kok, cv, cok)
		}
		if byKey.Prompt() != byClick.Prompt() {
			t.Errorf("cell %d: the two paths drew different grids\nkey:\n%s\nclick:\n%s", i, byKey.Prompt(), byClick.Prompt())
		}
		if byKey.Word() != byClick.Word() {
			t.Errorf("cell %d: Word() = %q by key, %q by click", i, byKey.Word(), byClick.Word())
		}
	}
}

// THE MODE DECIDES, which is why Mark takes no verdict.
func TestAMarkLandsTheActiveMode(t *testing.T) {
	b := NewBoard(cellsOf(sixteen...), 100, Palette{})
	if b.Mode() != Yes {
		t.Errorf("a new board's mode is %v, want Yes — most cells on a mature board are a yes", b.Mode())
	}
	if v, _ := b.Mark(0); v != Correct {
		t.Errorf("a Yes-mode mark graded %v, want Correct", v)
	}
	b.Toggle()
	if b.Mode() != No {
		t.Fatalf("Tab left the mode at %v", b.Mode())
	}
	if v, _ := b.Mark(1); v != Wrong {
		t.Errorf("a No-mode mark graded %v, want Wrong", v)
	}
	// TWO more Tabs to come back round: the cycle is three since #42 added drop.
	b.Toggle()
	if b.Mode() != Dropped {
		t.Fatalf("the second Tab gave %v, want Dropped", b.Mode())
	}
	b.Toggle()
	if b.Mode() != Yes {
		t.Errorf("Tab did not come back round: mode is %v", b.Mode())
	}
	// AND Keys() IS WHERE THE MODE IS SHOWN (R11). It had a footer row of its
	// own, and a resize measured what that costs: fitFooter drops from the END,
	// so a short terminal lost the one statement of what a click would mean
	// while every mark stayed irreversible. Paint clips the PROMPT last, so the
	// mode now lives on the row that survives longest.
	//
	// STILL ONE OWNER — the toggle row is gone, not duplicated. The mode is Yes
	// here: the two Toggles above returned it.
	yes := b.Keys()
	if !strings.Contains(yes, "[yes]") || strings.Contains(yes, "[no]") {
		t.Errorf("Keys() in Yes mode = %q, want the live mark bracketed", yes)
	}
	b.Toggle()
	no := b.Keys()
	if !strings.Contains(no, "[no]") || strings.Contains(no, "[yes]") {
		t.Errorf("Keys() in No mode = %q, want the live mark bracketed", no)
	}
	b.Toggle()
	drop := b.Keys()
	if !strings.Contains(drop, "[drop]") || strings.Contains(drop, "[no]") {
		t.Errorf("Keys() in Dropped mode = %q, want the live mark bracketed", drop)
	}
	// Same width, so the line does not jump under a key pressed repeatedly.
	if columnsIn(yes) != columnsIn(no) || columnsIn(no) != columnsIn(drop) {
		t.Errorf("the three spellings are %d, %d and %d columns:\n%q\n%q\n%q",
			columnsIn(yes), columnsIn(no), columnsIn(drop), yes, no, drop)
	}
	// And nothing BELOW the grid says it a second time.
	for _, line := range strings.Split(b.Prompt(), "\n") {
		if strings.Contains(line, "marking") {
			t.Errorf("the mode is drawn twice — Prompt has %q as well as the keys line", line)
		}
	}
}

// A CELL IS MARKED ONCE, because the first mark is already in the event log —
// every mark records as it lands, which is what makes Ctrl-C lossless, and the
// price of writing immediately is that nothing can be taken back. Fold would
// read the pair as two reviews of one word on one day.
func TestACellIsMarkedOnce(t *testing.T) {
	b := NewBoard(cellsOf(sixteen...), 100, Palette{})
	if _, ok := b.Mark(3); !ok {
		t.Fatal("the first mark was refused")
	}
	// The GRID, not the whole live edge: the toggle below it legitimately moves
	// when the mode flips, and that is not the cell changing.
	grid := func() string {
		return strings.Join(strings.Split(b.Prompt(), "\n")[:b.gridRows()], "\n")
	}
	before := grid()

	if v, ok := b.Mark(3); ok {
		t.Errorf("a second click on cell 3 graded (%v, true) — that is a duplicate review in the log", v)
	}
	b.Toggle() // and the other mode must not overwrite it either
	if v, ok := b.Mark(3); ok {
		t.Errorf("cell 3 was re-marked as %v after a mode flip", v)
	}
	if v, ok := b.Grade(rune(boardLabels[3])); ok {
		t.Errorf("cell 3 was re-marked as %v by its key", v)
	}
	if grid() != before {
		t.Errorf("a refused mark still changed the grid:\nbefore:\n%s\nafter:\n%s", before, grid())
	}
}

// Off the end is not a cell.
func TestAMarkOffTheBoardIsRefused(t *testing.T) {
	b := NewBoard(cellsOf("keel", "mesa", "run"), 80, Palette{})
	for _, i := range []int{-1, 3, 16, 99} {
		if v, ok := b.Mark(i); ok {
			t.Errorf("Mark(%d) = (%v, true) on a board of three", i, v)
		}
	}
}

// Spent is the Batch question the session asks before it advances, and the
// SIXTEENTH mark is the one that matters — a board that reported itself spent
// early would advance with words unanswered and no event for them.
func TestABoardIsSpentOnlyWhenEveryCellIsMarked(t *testing.T) {
	b := NewBoard(cellsOf(sixteen...), 100, Palette{})
	for i := range sixteen {
		if b.Spent() {
			t.Fatalf("spent after %d of %d marks", i, len(sixteen))
		}
		if _, ok := b.Mark(i); !ok {
			t.Fatalf("mark %d was refused", i)
		}
	}
	if !b.Spent() {
		t.Error("not spent after the sixteenth mark")
	}
}

// A BOARD OF THREE (D5). The board is what the caller could fill, not always
// sixteen, and every part of it has to work at that size.
func TestABoardOfThreeIsAWholeBoard(t *testing.T) {
	words := []string{"keel", "mesa", "run"}
	b := NewBoard(cellsOf(words...), 80, Palette{})

	if got := b.gridRows(); got != 1 {
		t.Errorf("gridRows() = %d, want 1 — three short words fit on one line", got)
	}
	if _, ok := b.Grade('3'); ok {
		t.Error("Grade('3') found a fourth cell on a board of three")
	}
	for i := range words {
		if _, ok := b.Grade(rune(boardLabels[i])); !ok {
			t.Errorf("cell %d unreachable on a board of three", i)
		}
	}
	if !b.Spent() {
		t.Error("three marks did not spend a board of three")
	}
}

// Rest is what Enter means to a form holding many: the unmarked are answered and
// NAMED, because the session records one event per word it returns.
func TestRestTakesTheUnmarkedAndNamesThem(t *testing.T) {
	b := NewBoard(cellsOf(sixteen...), 100, Palette{})
	b.Mark(0)
	b.Toggle()
	b.Mark(1)

	rest := b.Rest(Wrong)
	if len(rest) != len(sixteen)-2 {
		t.Fatalf("Rest returned %d words, want the %d still unmarked", len(rest), len(sixteen)-2)
	}
	for _, w := range rest {
		if w == sixteen[0] || w == sixteen[1] {
			t.Errorf("Rest named %q, which was already marked — that is a second event for one word", w)
		}
	}
	if !b.Spent() {
		t.Error("the board is not spent after Rest, so the session would never advance")
	}
	if again := b.Rest(Wrong); len(again) != 0 {
		t.Errorf("a second Rest named %v, want nothing left", again)
	}
	// The safe direction: anything that is not Correct falls back down the
	// ladder rather than climbing it.
	fresh := NewBoard(cellsOf("keel", "mesa"), 80, Palette{})
	fresh.Rest(Wrong)
	if fresh.marks[0] != No {
		t.Errorf("Rest(Wrong) left mark %v, want No", fresh.marks[0])
	}
	fresh2 := NewBoard(cellsOf("keel", "mesa"), 80, Palette{})
	fresh2.Rest(Skipped)
	if fresh2.marks[0] != No {
		t.Errorf("Rest(Skipped) left mark %v, want No — an unmarked word is asked again, not promoted", fresh2.marks[0])
	}
}

// advance builds Outcome{Word: q.Word()} at a call site that knows nothing about
// grids, so the form has to answer "which word did that just mean".
func TestWordIsTheCellTheMarkLandedOn(t *testing.T) {
	b := NewBoard(cellsOf(sixteen...), 100, Palette{})
	for _, i := range []int{5, 0, 15, 9} {
		if _, ok := b.Mark(i); !ok {
			t.Fatalf("mark %d refused", i)
		}
		if got := b.Word(); got != sixteen[i] {
			t.Errorf("after marking cell %d, Word() = %q, want %q", i, got, sixteen[i])
		}
	}
}

// NO LINE WIDER THAN THE WIDTH IT WAS BUILT FOR. This is the guarantee D15's
// whole argument rests on: a footer row that wraps is a frame one row too tall,
// the terminal scrolls to fit it, and a click at viewport row R stops meaning
// what the board drew there.
func TestNoGridLineExceedsTheWidth(t *testing.T) {
	long := []string{"antidisestablishmentarianism", "keel"}
	for _, words := range [][]string{sixteen, {"keel", "mesa", "run"}, long, {"a"}} {
		for _, w := range []int{20, 24, 37, 40, 60, 79, 80, 120, 200} {
			b := NewBoard(cellsOf(words...), w, Palette{})
			for i, line := range strings.Split(b.Prompt(), "\n") {
				if n := columnsIn(line); n > w {
					t.Errorf("width %d, %d words: line %d is %d columns:\n%s", w, len(words), i, n, line)
				}
			}
			if got := len(strings.Split(b.Prompt(), "\n")); got != b.Rows() {
				t.Errorf("width %d, %d words: Prompt drew %d lines, Rows() says %d", w, len(words), got, b.Rows())
			}
		}
	}
}

// Every word is on screen from the first frame, each with its printed key.
func TestPromptLabelsEveryWord(t *testing.T) {
	b := NewBoard(cellsOf(sixteen...), 100, Palette{})
	p := b.Prompt()
	for i, w := range sixteen {
		if !strings.Contains(p, "["+string(boardLabels[i])+"] ") {
			t.Errorf("cell %d has no printed key in:\n%s", i, p)
		}
		if !strings.Contains(p, w) {
			t.Errorf("%q is not on the grid:\n%s", w, p)
		}
	}
}

// A MARKED CELL IS PAINTED AND KEEPS ITS KEY.
//
// The first cut put the mark where the key was — `[y]` in place of `[3]` — and
// an operator sitting corrected it: *"selected words on the board should change
// color instead of use [y]/[n]"*, *"don't change the [0]...[f] as they are
// needed for keyboard operation"*. The key is how a mouse-less terminal reaches
// the cell and how a learner reads the grid back, so it is the wrong half to
// spend on saying "answered".
func TestAMarkedCellIsPaintedAndKeepsItsKey(t *testing.T) {
	// EVERY MARK'S SEQUENCE, so no arm of the paint can be deleted unnoticed. The
	// drop was added to the Palette and to `paint` with neither pinned, and both
	// could be removed with the whole suite green — a dropped cell would then have
	// painted identically to an untouched one while the README promised otherwise
	// (#42 BR-1).
	pal := Palette{Yes: "\x1b[1;32m", No: "\x1b[1;31m", Drop: "\x1b[2;9m", Off: "\x1b[0m"}
	b := NewBoard(cellsOf(sixteen...), 100, pal)
	b.Mark(2)
	b.Toggle()
	b.Mark(7)
	b.Toggle()
	b.Mark(9)
	p := b.Prompt()

	// THE KEYS ARE ALL STILL THERE, marked or not.
	for i := range sixteen {
		if !strings.Contains(p, "["+string(boardLabels[i])+"] ") {
			t.Errorf("cell %d lost its key:\n%s", i, p)
		}
	}
	// ...and the marked ones are painted, in the caller's own sequences.
	if !strings.Contains(p, pal.Yes+"[2] "+sixteen[2]+pal.Off) {
		t.Errorf("cell 2 is not painted as a yes:\n%s", p)
	}
	if !strings.Contains(p, pal.No+"[7] "+sixteen[7]+pal.Off) {
		t.Errorf("cell 7 is not painted as a no:\n%s", p)
	}
	if !strings.Contains(p, pal.Drop+"[9] "+sixteen[9]+pal.Off) {
		t.Errorf("cell 9 is not painted as a drop:\n%s", p)
	}
	// THREE DISTINCT sequences, or two marks read as one on screen.
	if pal.Yes == pal.No || pal.No == pal.Drop || pal.Yes == pal.Drop {
		t.Error("two marks share a sequence, so the test cannot tell them apart either")
	}
	// An unmarked cell carries no sequence at all.
	if strings.Contains(p, pal.Yes+"[0] ") || strings.Contains(p, pal.No+"[0] ") {
		t.Errorf("an unmarked cell is painted:\n%s", p)
	}
	// A board with no palette draws plain text, which is what -no-color would
	// get if --play did not refuse to run under it.
	plain := NewBoard(cellsOf(sixteen...), 100, Palette{})
	plain.Mark(2)
	if strings.ContainsRune(plain.Prompt(), 0x1b) {
		t.Errorf("an unpalletted board emitted an escape:\n%q", plain.Prompt())
	}
	// AND Marked() reports the state without anyone parsing colour back out.
	if b.Marked(2) != Yes || b.Marked(7) != No || b.Marked(9) != Dropped || b.Marked(0) != Unmarked {
		t.Errorf("Marked() = %v/%v/%v/%v, want Yes/No/Dropped/Unmarked",
			b.Marked(2), b.Marked(7), b.Marked(9), b.Marked(0))
	}
}

// PAINT COSTS NO COLUMNS, which is what keeps the click map honest: the padding
// is applied outside the style, so a styled cell occupies exactly the columns an
// unstyled one does.
func TestPaintDoesNotChangeTheLayout(t *testing.T) {
	pal := Palette{Yes: "\x1b[1;32m", No: "\x1b[1;31m", Off: "\x1b[0m"}
	for _, w := range []int{24, 40, 80} {
		plain := NewBoard(cellsOf(sixteen...), w, Palette{})
		painted := NewBoard(cellsOf(sixteen...), w, pal)
		for i := range sixteen {
			if i%3 == 0 {
				plain.Mark(i)
				painted.Mark(i)
			}
		}
		if plain.Rows() != painted.Rows() {
			t.Errorf("width %d: %d rows plain, %d painted", w, plain.Rows(), painted.Rows())
		}
		for i := range sixteen {
			pr, pc := findCell(t, plain, i)
			ar, ac := findCell(t, painted, i)
			if pr != ar || pc != ac {
				t.Errorf("width %d: cell %d is at (%d,%d) plain and (%d,%d) painted", w, i, pr, pc, ar, ac)
			}
			if got, ok := painted.CellAt(ar, ac); !ok || got != i {
				t.Errorf("width %d: CellAt(%d,%d) = (%d,%v) on a painted board, want cell %d", w, ar, ac, got, ok, i)
			}
		}
	}
}

// findCell is where a cell's key is drawn, in grid rows and visible columns.
func findCell(t *testing.T, b *Board, i int) (int, int) {
	t.Helper()
	key := "[" + string(boardLabels[i]) + "] "
	for r, line := range strings.Split(b.Prompt(), "\n")[:b.gridRows()] {
		if c := strings.Index(line, key); c >= 0 {
			return r, visibleColumns(line[:c])
		}
	}
	t.Fatalf("cell %d is not on the grid:\n%s", i, b.Prompt())
	return 0, 0
}

// CellAt is derived from what Prompt DREW, not from the layout fields, because
// the invariant is that the two agree.
func TestCellAtFindsWhatPromptDrew(t *testing.T) {
	for _, width := range []int{40, 80, 120} {
		b := NewBoard(cellsOf(sixteen...), width, Palette{})
		lines := strings.Split(b.Prompt(), "\n")
		for i := range sixteen {
			key := "[" + string(boardLabels[i]) + "] "
			row, start := -1, -1
			for r, line := range lines {
				if c := strings.Index(line, key); c >= 0 {
					row, start = r, c
					break
				}
			}
			if row < 0 {
				t.Fatalf("width %d: cell %d is not on the grid:\n%s", width, i, b.Prompt())
			}
			// The whole cell is a target: its key and its word.
			last := start + labelWidth + columnsIn(sixteen[i]) - 1
			for _, col := range []int{start, start + 1, last} {
				got, ok := b.CellAt(row, col)
				if !ok || got != i {
					t.Errorf("width %d: CellAt(%d,%d) = (%d,%v), want cell %d — the click is on its own text", width, row, col, got, ok, i)
				}
			}
			// THE GUTTER IS NOT A TARGET. A mark cannot be taken back, so a
			// click that is not clearly on a word must do nothing rather than
			// mark its neighbour.
			if start > 0 {
				if got, ok := b.CellAt(row, start-1); ok {
					t.Errorf("width %d: a click in the gutter before cell %d marked cell %d", width, i, got)
				}
			}
		}
		if _, ok := b.CellAt(-1, 0); ok {
			t.Error("a click above the grid found a cell")
		}
		if _, ok := b.CellAt(b.Rows(), 0); ok {
			t.Error("a click below the grid found a cell")
		}
		if _, ok := b.CellAt(0, 10_000); ok {
			t.Error("a click far to the right found a cell")
		}
	}
}

// A short final row has no cell past its last word.
func TestAClickPastTheLastCellFindsNothing(t *testing.T) {
	// Five words at a width that holds four columns: the second row has one.
	b := NewBoard(cellsOf("keel", "mesa", "run", "bank", "set"), 80, Palette{})
	if b.gridRows() != 2 || b.cols != 4 {
		t.Fatalf("expected a 4-wide grid of 2 rows, got %d cols and %d rows", b.cols, b.gridRows())
	}
	if got, ok := b.CellAt(1, 0); !ok || got != 4 {
		t.Fatalf("CellAt(1,0) = (%d,%v), want cell 4", got, ok)
	}
	if got, ok := b.CellAt(1, b.cell); ok {
		t.Errorf("a click in the empty part of the last row marked cell %d", got)
	}
}

// A board hides nothing, so there is no definition to earn by missing one.
func TestABoardRevealsNothingAndRatesItself(t *testing.T) {
	b := NewBoard(cellsOf(sixteen...), 100, Palette{})
	if b.Reveal() != "" {
		t.Errorf("Reveal() = %q, want empty — every word is already on screen", b.Reveal())
	}
	if !b.IsSelfRated() {
		t.Error("a board is self-rated: a mark is a claim about a word the learner is looking at")
	}
}

// The label alphabet is the capacity, and a word this board does not ask about
// gets no event — so its box does not move and it is due again tomorrow.
func TestNewBoardCapsAtItsLabelAlphabet(t *testing.T) {
	words := append(append([]string{}, sixteen...), "seventeenth")
	b := NewBoard(cellsOf(words...), 100, Palette{})
	if len(b.cells) != MaxBoardWords {
		t.Errorf("a board of %d words kept %d, want %d", len(words), len(b.cells), MaxBoardWords)
	}
	if strings.Contains(b.Prompt(), "seventeenth") {
		t.Error("the seventeenth word is on a grid with no key for it")
	}
}

// THE FORM THROUGH THE REAL STATE MACHINE. session_test.go asserts what the
// SESSION does, with a double; this asserts that *Board is the thing the session
// was widened for.
func TestABoardRunsThroughTheSession(t *testing.T) {
	b := NewBoard(cellsOf("keel", "mesa", "run"), 80, Palette{})
	s := NewSession([]Question{b})

	// A yes records against the word the mark landed on, and does NOT advance.
	s, outs := Apply(s, Input{Kind: InputRune, Rune: '1'})
	if len(outs) != 1 || outs[0].Kind != OutcomeRecord || outs[0].Word != "mesa" || outs[0].Verdict != Correct {
		t.Fatalf("a yes on cell 1 produced %+v", outs)
	}
	if s.Index != 0 || s.Done {
		t.Fatalf("the session advanced off a board with two cells unmarked: index %d done %v", s.Index, s.Done)
	}
	if outs[0].Unaided {
		t.Error("a board is self-rated, so it can never earn the two-rung promotion")
	}

	// A NO MUST NOT FREEZE THE BOARD (D12): the miss-on-a-hidden-word branch
	// would set Graded, and the next key would then mean "any key = next word".
	b.Toggle()
	s, outs = Apply(s, Input{Kind: InputRune, Rune: '0'})
	if len(outs) != 1 || outs[0].Kind != OutcomeRecord || outs[0].Word != "keel" || outs[0].Verdict != Wrong {
		t.Fatalf("a no on cell 0 produced %+v", outs)
	}
	if s.Graded {
		t.Fatal("a no set Graded — the board would end after one mark")
	}

	// `d` names no word on a grid, so it is refused rather than guessed.
	s, outs = Apply(s, Input{Kind: InputDrop})
	if len(outs) != 1 || outs[0].Kind != OutcomeNone {
		t.Errorf("InputDrop on a board produced %+v, want nothing", outs)
	}

	// Space reveals nothing and must not spend the board (D14).
	s, _ = Apply(s, Input{Kind: InputReveal})
	if s.Done || s.Index != 0 {
		t.Fatal("space spent the board")
	}
	if b.Spent() {
		t.Fatal("space marked the remaining cell")
	}

	// Enter takes the rest as No and finishes the sitting.
	s, outs = Apply(s, Input{Kind: InputFinish})
	var recorded []string
	for _, o := range outs {
		if o.Kind == OutcomeRecord {
			if o.Verdict != Wrong {
				t.Errorf("%q was committed as %v, want Wrong", o.Word, o.Verdict)
			}
			recorded = append(recorded, o.Word)
		}
	}
	if len(recorded) != 1 || recorded[0] != "run" {
		t.Errorf("Enter recorded %v, want just the unmarked word", recorded)
	}
	if !s.Done {
		t.Error("the session did not end after the board was spent")
	}
	if s.Right != 1 || s.Wrong != 2 {
		t.Errorf("tally is %d right %d wrong, want 1 and 2", s.Right, s.Wrong)
	}
}

// Tab reaches the board through the state machine, and the grid says so.
func TestTabFlipsTheBoardsModeThroughApply(t *testing.T) {
	b := NewBoard(cellsOf("keel", "mesa", "run"), 80, Palette{})
	s := NewSession([]Question{b})

	s, _ = Apply(s, Input{Kind: InputToggle})
	if b.Mode() != No {
		t.Fatalf("Tab left the mode at %v", b.Mode())
	}
	s, outs := Apply(s, Input{Kind: InputRune, Rune: '0'})
	if len(outs) != 1 || outs[0].Verdict != Wrong {
		t.Errorf("a mark after Tab produced %+v, want Wrong", outs)
	}
	if _, ok := Apply(s, Input{Kind: InputToggle}); b.Mode() != Dropped {
		t.Errorf("a second Tab left the mode at %v, want Dropped; outs %+v", b.Mode(), ok)
	}
	if _, ok := Apply(s, Input{Kind: InputToggle}); b.Mode() != Yes {
		t.Errorf("a third Tab left the mode at %v, want Yes; outs %+v", b.Mode(), ok)
	}
}

// A CLICK ANSWERS A BOARD, AND ONLY A BOARD (D11).
//
// The loop resolves the cell — it asks the screen which footer row and the form
// which cell — and hands Apply an InputMark. From here on it is the same act the
// printed key performs, which is the property that keeps the two paths from
// drifting.
func TestAClickMarksAGridFormThroughApply(t *testing.T) {
	b := NewBoard(cellsOf("keel", "mesa", "run"), 80, Palette{})
	s := NewSession([]Question{b})

	s, outs := Apply(s, Input{Kind: InputMark, Cell: 2})
	if len(outs) != 1 || outs[0].Kind != OutcomeRecord || outs[0].Word != "run" || outs[0].Verdict != Correct {
		t.Fatalf("a click on cell 2 produced %+v", outs)
	}
	if s.Index != 0 || s.Done {
		t.Errorf("the session advanced off a board with two cells unmarked")
	}
	// A second click on the same cell is not a second review.
	if _, outs = Apply(s, Input{Kind: InputMark, Cell: 2}); len(outs) != 1 || outs[0].Kind != OutcomeNone {
		t.Errorf("a second click on cell 2 produced %+v, want nothing", outs)
	}
	// Off the grid is nothing.
	if _, outs = Apply(s, Input{Kind: InputMark, Cell: 9}); len(outs) != 1 || outs[0].Kind != OutcomeNone {
		t.Errorf("a click on cell 9 of a board of three produced %+v", outs)
	}
	// And the mode is what lands, exactly as it does for a key.
	b.Toggle()
	if _, outs = Apply(s, Input{Kind: InputMark, Cell: 0}); outs[0].Verdict != Wrong {
		t.Errorf("a click in No mode recorded %v", outs[0].Verdict)
	}
}

// THE PANEL IS THE FEEDBACK MOMENT: mark a word and its gloss appears where a
// definition would be on any other form.
func TestThePanelShowsTheLastMarkedWordsGloss(t *testing.T) {
	cells := []Cell{
		{Word: "quokka", Gloss: "a short-tailed wallaby of SW Australia"},
		{Word: "keel", Gloss: "the lengthwise timber along a ship's base"},
		{Word: "mesa"}, // no gloss: an entry that is only cross-references
	}
	b := NewBoard(cells, 80, Palette{})
	lines := func() []string { return strings.Split(b.Prompt(), "\n") }

	// EMPTY IS STILL A ROW. A panel that appeared with the first mark would
	// shift the grid up by one, and every word would move under a pointer
	// already resting on it.
	before := len(lines())
	if got := lines()[before-1]; got != "" {
		t.Errorf("the panel starts as %q, want empty — nothing is marked yet", got)
	}

	b.Mark(0)
	if got, after := lines()[len(lines())-1], len(lines()); after != before {
		t.Errorf("the live edge grew from %d rows to %d when a mark landed — the grid moved under the pointer (panel %q)", before, after, got)
	}
	if got := lines()[len(lines())-1]; !strings.Contains(got, "quokka") || !strings.Contains(got, "wallaby") {
		t.Errorf("the panel is %q, want the word it just marked and its gloss", got)
	}

	// It follows the LAST mark, whichever that is.
	b.Toggle()
	b.Mark(1)
	if got := lines()[len(lines())-1]; !strings.Contains(got, "keel") {
		t.Errorf("the panel is %q after marking keel", got)
	}
	// A word with no gloss names itself rather than showing a bare separator.
	b.Mark(2)
	if got := lines()[len(lines())-1]; got != "mesa" {
		t.Errorf("the panel is %q for a word with no gloss, want just the word", got)
	}
	// A refused mark leaves it where it was.
	b.Mark(2)
	if got := lines()[len(lines())-1]; got != "mesa" {
		t.Errorf("a refused mark moved the panel to %q", got)
	}
}

// The panel obeys the width like every other row: a long gloss is the most
// likely thing here to wrap, and a footer row that wraps scrolls the terminal.
func TestALongGlossDoesNotWidenTheLiveEdge(t *testing.T) {
	long := "a very long definition indeed, going on well past any reasonable terminal width and then continuing for a while after that"
	for _, w := range []int{20, 40, 80, 120} {
		b := NewBoard([]Cell{{Word: "quokka", Gloss: long}}, w, Palette{})
		b.Mark(0)
		for i, line := range strings.Split(b.Prompt(), "\n") {
			if n := columnsIn(line); n > w {
				t.Errorf("width %d: line %d is %d columns:\n%s", w, i, n, line)
			}
		}
	}
}

// THE ROWS UNDER THE GRID ARE NOT WORDS. A click on the toggle must not mark
// something — a mark cannot be taken back, so the wrong one is permanent.
//
// The property, pinned apart from whichever line in CellAt happens to deliver
// it: this is the thing that must stay true, and a rewrite of the arithmetic
// should have to keep it rather than keep a particular guard.
func TestTheChromeRowsAreNotCells(t *testing.T) {
	// Boards of every shape a full grid can take, because the row that stops
	// being a cell moves with the word count.
	for _, n := range []int{1, 2, 3, 4, 5, 8, 13, 16} {
		b := NewBoard(cellsOf(sixteen[:n]...), 80, Palette{})
		lines := strings.Split(b.Prompt(), "\n")
		for row := b.gridRows(); row < len(lines); row++ {
			for col := 0; col < 80; col++ {
				if i, ok := b.CellAt(row, col); ok {
					t.Fatalf("%d words: a click at row %d (%q), col %d marked cell %d",
						n, row, lines[row], col, i)
				}
			}
		}
		// ...and the premise: the grid rows above them DO answer.
		if _, ok := b.CellAt(b.gridRows()-1, 0); !ok {
			t.Errorf("%d words: the grid's last row offers no cell at column 0", n)
		}
	}
}

// Rows() IS len(Prompt()'s lines), at every size including none.
//
// It was arithmetic that happened to match rather than a construction that had
// to: with no grid rows the separator produced two blanks instead of one, so an
// empty board's Prompt yielded four lines while Rows() said three. Unreachable
// through the packing caller, and that is exactly why it needs a test — the invariant is
// defended in Word() and panelLine() and was dropped here.
func TestRowsIsWhatPromptDraws(t *testing.T) {
	for n := 0; n <= MaxBoardWords; n++ {
		for _, w := range []int{20, 40, 80} {
			b := NewBoard(cellsOf(sixteen[:n]...), w, Palette{})
			if got, want := len(strings.Split(b.Prompt(), "\n")), b.Rows(); got != want {
				t.Errorf("%d words at %d columns: Prompt drew %d lines, Rows() says %d:\n%q",
					n, w, got, want, b.Prompt())
			}
		}
	}
}

// A CLICK IN A CELL'S OWN PADDING MARKS THAT CELL, never a neighbour.
//
// The other half of "the gutter is not a target": a short word in a column sized
// for a long one leaves blanks after it, and those blanks belong to it. Written
// down as a test because it reads as clicking empty space.
func TestACellsPaddingBelongsToIt(t *testing.T) {
	// "keel" is short and "arrondissement" sets the column width.
	b := NewBoard(cellsOf("arrondissement", "keel", "mesa", "run"), 80, Palette{})
	line := strings.Split(b.Prompt(), "\n")[0]
	start := strings.Index(line, "[1] ")
	if start < 0 {
		t.Fatalf("no second cell in %q", line)
	}
	// One column past the end of "keel", which is still inside its cell.
	col := start + len("[1] ") + len("keel")
	got, ok := b.CellAt(0, col)
	if !ok || got != 1 {
		t.Errorf("CellAt(0,%d) = (%d,%v) in cell 1's padding, want cell 1", col, got, ok)
	}
	// ...and the gutter after the padding is still nobody's.
	gutter := strings.Index(line, "[2] ") - 1
	if gutter <= col {
		t.Fatalf("could not find a gutter column after %d in %q", col, line)
	}
	if got, ok := b.CellAt(0, gutter); ok {
		t.Errorf("a click in the gutter at column %d marked cell %d", gutter, got)
	}
}

// A BOARD IS LAID OUT FOR THE WIDTH IT IS DRAWN AT, not the one it was chosen at
// (R9).
//
// The first version fixed the layout in NewBoard and the comment said the width
// "cannot change". It can: the terminal is resized under a live board, and a
// board built for eighty columns has 74-column rows that the terminal then wraps
// into two. A footer entry stops being one physical row, and a click on the
// continuation carries a column that means a different word — permanently,
// because a mark cannot be taken back.
//
// The marks SURVIVE the relayout, which is why this is a relayout and not a new
// board: those answers are already in the log.
func TestABoardRelaysOutForTheWidthItIsDrawnAt(t *testing.T) {
	pal := Palette{Yes: "\x1b[1;32m", No: "\x1b[1;31m", Off: "\x1b[0m"}
	b := NewBoard(cellsOf(sixteen...), 100, pal)
	b.Mark(2)
	b.Toggle()
	b.Mark(7)
	markedYes, markedNo := sixteen[2], sixteen[7]

	for _, w := range []int{80, 60, 40, 24, 100} {
		b.Resize(w)

		// EVERY LINE FITS, which is the invariant the click map rests on.
		lines := strings.Split(b.Prompt(), "\n")
		for i, line := range lines {
			// VISIBLE columns: this board is painted, and colour costs none.
			if n := visibleColumns(line); n > w {
				t.Errorf("after Resize(%d) line %d is %d columns:\n%s", w, i, n, line)
			}
		}
		if len(lines) != b.Rows() {
			t.Errorf("after Resize(%d): Prompt drew %d lines, Rows() says %d", w, len(lines), b.Rows())
		}
		// AND THE CLICK MAP AGREES WITH WHAT WAS DRAWN, at the new shape.
		for i := range sixteen {
			// The label, always — a marked cell keeps its key and changes colour
			// instead, which is what the operator asked for after a real sitting.
			key := "[" + string(boardLabels[i]) + "] "
			row, col := -1, -1
			for r := 0; r < b.gridRows(); r++ {
				if c := strings.Index(lines[r], key+b.cells[i].Word); c >= 0 {
					row, col = r, visibleColumns(lines[r][:c])
					break
				}
			}
			if row < 0 {
				t.Fatalf("after Resize(%d), cell %d is not on the grid:\n%s", w, i, b.Prompt())
			}
			if got, ok := b.CellAt(row, col); !ok || got != i {
				t.Errorf("after Resize(%d), CellAt(%d,%d) = (%d,%v), want cell %d", w, row, col, got, ok, i)
			}
		}
		// THE MARKS SURVIVE: they are already in the event log.
		if !strings.Contains(b.Prompt(), pal.Yes+"[2] "+markedYes) {
			t.Errorf("after Resize(%d) the yes mark on %q is gone:\n%s", w, markedYes, b.Prompt())
		}
		if !strings.Contains(b.Prompt(), pal.No+"[7] "+markedNo) {
			t.Errorf("after Resize(%d) the no mark on %q is gone:\n%s", w, markedNo, b.Prompt())
		}
		if b.Spent() {
			t.Errorf("after Resize(%d) the board reports itself spent after two marks", w)
		}
	}
}

// TAB CYCLES THREE MODES, and the order is a UX decision rather than arithmetic
// on the iota (#42).
//
// Yes → No → Dropped, so the DESTRUCTIVE mode is never one Tab from the default:
// a learner reaching for `no` cannot overshoot into a removal, and the mode they
// are most likely to want is the one they start in.
func TestTabCyclesThreeModes(t *testing.T) {
	b := NewBoard([]Cell{{Word: "keel"}, {Word: "mesa"}, {Word: "run"}}, 80, Palette{})
	for i, want := range []Mark{No, Dropped, Yes, No} {
		b.Toggle()
		if got := b.Mode(); got != want {
			t.Fatalf("Tab %d gave mode %v, want %v — the cycle is Yes → No → Dropped, "+
				"so the destructive mode is never one press from the default", i+1, got, want)
		}
	}
}

// A DROPPED CELL RECORDS NO REVIEW. Its verdict is Skipped, which the session
// never writes — a drop is not an assessment, and recording one as a miss would
// demote a word on its way out of the deck.
func TestADroppedCellRecordsNothing(t *testing.T) {
	if got := Dropped.Verdict(); got != Skipped {
		t.Errorf("Dropped.Verdict() = %v, want Skipped — a removal is not an answer, and "+
			"schedule.Fold reads verdicts to move boxes", got)
	}
	b := NewBoard([]Cell{{Word: "keel"}, {Word: "mesa"}}, 80, Palette{})
	b.Toggle()
	b.Toggle() // into drop mode
	if v, ok := b.Mark(0); !ok || v != Skipped {
		t.Errorf("marking a cell in drop mode gave (%v, %v), want (Skipped, true)", v, ok)
	}
	// ...and it IS marked, so Enter's sweep leaves it alone and a second press
	// cannot drop it twice.
	if got := b.Marked(0); got != Dropped {
		t.Errorf("cell 0 is %v after a drop, want Dropped", got)
	}
	if rest := b.Rest(Wrong); len(rest) != 1 || rest[0] != "mesa" {
		t.Errorf("Enter swept %v, want only the unmarked word — a dropped cell is not "+
			"unmarked and must not be answered on the way out", rest)
	}
}

// THE FORM SAYS WHICH WORD WAS DROPPED, through a capability rather than a type
// switch — one-shot, so the same removal cannot be performed twice.
func TestABoardNamesTheWordItDropped(t *testing.T) {
	b := NewBoard([]Cell{{Word: "keel"}, {Word: "mesa"}}, 80, Palette{})
	if _, ok := b.Dropped(); ok {
		t.Error("a board with no marks claims a drop")
	}
	b.Toggle()
	b.Toggle()
	b.Mark(1)
	word, ok := b.Dropped()
	if !ok || word != "mesa" {
		t.Fatalf("Dropped() = (%q, %v), want (\"mesa\", true)", word, ok)
	}
	// ONE-SHOT. Asked again — which the loop does on the next frame, and after a
	// refused click — it must not name the word a second time, or the removal is
	// performed twice against a word already gone.
	if _, ok := b.Dropped(); ok {
		t.Error("Dropped() answered twice for one drop — the loop asks it per mark, so a " +
			"sticky answer removes the same word on every following keystroke")
	}
	// A yes after a drop is a MARK, not a drop.
	b.Toggle()
	b.Mark(0)
	if _, ok := b.Dropped(); ok {
		t.Error("a yes mark claimed to be a drop")
	}
}

// ALL THREE MODE SPELLINGS ARE THE SAME WIDTH, and the row fits eighty columns
// with the session's reserved key beside it.
//
// NOT COSMETIC: `boardFitsIn` charges `displayRows(gradePrompt(q), termCols)` into
// the board's fit, so a row that wraps at 80 raises the minimum terminal height
// for EVERY board — on the path #42 routes untestable young words onto, where
// there is no form 2.1 left to fall back to. The existing refusal-row pin cannot
// catch this: it compares the refusal against the keys row, and both grow
// together.
func TestEveryModeSpellingIsTheSameWidthAndFitsEighty(t *testing.T) {
	b := NewBoard([]Cell{{Word: "keel"}, {Word: "mesa"}}, 80, Palette{})
	want := visibleColumns(b.Keys())
	for range 3 {
		got := visibleColumns(b.Keys())
		if got != want {
			t.Errorf("mode %v spells a %d-column row; the first was %d — the line must not "+
				"jump under a key pressed to be pressed again", b.Mode(), got, want)
		}
		// The reserved half a board gets is `quitKey` — `d` is a cell key here.
		if total := got + len(", ") + len("Ctrl-C to stop"); total > 80 {
			t.Errorf("mode %v: the prompt row is %d columns with the reserved keys, over 80 — "+
				"it wraps, and boardFitsIn charges the wrapped height to every board",
				b.Mode(), total)
		}
		b.Toggle()
	}
}

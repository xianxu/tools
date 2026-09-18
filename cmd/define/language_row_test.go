package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

type rowTestCell struct {
	glyph   rune
	bg, fg  int
	inverse bool
}

// Independent terminal-state oracle: consumes SGR, writes wide glyph cells,
// and leaves unwritten cells at the terminal default background (-1).
func rowTestCells(t *testing.T, text string, width int) ([]rowTestCell, rowTestCell) {
	t.Helper()
	cells := make([]rowTestCell, width)
	for i := range cells {
		cells[i].bg = -1
		cells[i].fg = -1
	}
	state := rowTestCell{bg: -1, fg: -1}
	col := 0
	for len(text) > 0 {
		if strings.HasPrefix(text, "\x1b[") {
			end := strings.IndexByte(text, 'm')
			if end < 0 {
				t.Fatalf("non-SGR sequence: %q", text)
			}
			params := strings.Split(text[2:end], ";")
			for i := 0; i < len(params); i++ {
				n, _ := strconv.Atoi(params[i])
				switch {
				case n == 0:
					state.bg = -1
					state.fg = -1
					state.inverse = false
				case n == 7:
					state.inverse = true
				case n == 27:
					state.inverse = false
				case n == 49:
					state.bg = -1
				case n == 39:
					state.fg = -1
				case n >= 40 && n <= 47:
					state.bg = n
				case n >= 30 && n <= 37:
					state.fg = n
				case (n == 48 || n == 38) && i+2 < len(params) && params[i+1] == "5":
					v, _ := strconv.Atoi(params[i+2])
					if n == 48 {
						state.bg = v
					} else {
						state.fg = v
					}
					i += 2
				}
			}
			text = text[end+1:]
			continue
		}
		rs := []rune(text)
		r := rs[0]
		text = strings.TrimPrefix(text, string(r))
		w := 1
		if r >= 0x4e00 && r <= 0x9fff {
			w = 2
		}
		if r >= 0x300 && r <= 0x36f {
			w = 0
		}
		for j := 0; j < w; j++ {
			if col >= width {
				t.Fatalf("row overflow at column %d", col)
			}
			state.glyph = r
			cells[col] = state
			col++
		}
	}
	return cells, state
}

func TestLanguageRowFillsWhitespace(t *testing.T) {
	text := "  hola"
	got := paintLanguageRow(text, rowPaint{tinted: true}, 10, store.SchemeDark)
	cells, _ := rowTestCells(t, got, 10)
	for col, c := range cells {
		if c.bg != 236 {
			t.Fatalf("column %d background=%d, want 236 (including indentation and padding)", col, c.bg)
		}
	}
}

func TestLanguageRowStylesAndExclusions(t *testing.T) {
	text := "\x1b[31m a\x1b[0mb\x1b[42mc\x1b[49m\x1b[7md\x1b[27me"
	got := paintLanguageRow(text, rowPaint{tinted: true, exclusions: []cellRange{{1, 2}}}, 9, store.SchemeLight)
	cells, end := rowTestCells(t, got, 9)
	want := []int{254, -1, 254, 42, 254, 254, 254, 254, 254}
	for col, c := range cells {
		if c.bg != want[col] {
			t.Errorf("column %d background %d, want %d", col, c.bg, want[col])
		}
	}
	if cells[0].fg != 31 || cells[1].fg != 31 || cells[2].fg != -1 {
		t.Errorf("foreground changed: %+v", cells)
	}
	if !cells[4].inverse || cells[5].inverse || cells[6].inverse {
		t.Errorf("selection inverse changed: %+v", cells)
	}
	if end.bg != -1 || end.inverse {
		t.Errorf("style leaks past row: %+v", end)
	}
}

func TestLanguageRowBlankWideAndExactEdge(t *testing.T) {
	for _, text := range []string{"", "  ", "界ab", "abc界", "abcdef"} {
		got := paintLanguageRow(text, rowPaint{tinted: true}, 4, store.SchemeDark)
		cells, end := rowTestCells(t, got, 4)
		for col, c := range cells {
			if c.bg != 236 {
				t.Errorf("%q column %d not filled: %+v", text, col, c)
			}
		}
		if end.bg != -1 {
			t.Errorf("%q leaks background", text)
		}
		if strings.ContainsAny(got, "\r\n") {
			t.Errorf("paint introduces wrap for %q", text)
		}
	}
}

func TestLanguageRowNeutralAndInvalid(t *testing.T) {
	control := "\x1b[H\x1b[2Jhello\r\n"
	// An untrusted background STRING was once a case here; since #70 a paint is
	// only a bit, so that input is unrepresentable rather than filtered.
	for _, p := range []rowPaint{{}, {tinted: true, exclusions: []cellRange{{3, 1}}}} {
		if got := paintLanguageRow(control, p, 4, store.SchemeDark); got != control {
			t.Errorf("neutral/control write changed: %q", got)
		}
	}
	if got := paintLanguageRow("hola", rowPaint{tinted: true}, 0, store.SchemeDark); got != "hola" {
		t.Errorf("zero width adds paint: %q", got)
	}
}

func TestLanguageRowResetsBeforeCursorControl(t *testing.T) {
	got := paintLanguageRow("a\x1b[Kb", rowPaint{tinted: true}, 4, store.SchemeDark)
	if !strings.Contains(got, "\x1b[0m\x1b[K") {
		t.Fatalf("background reaches erase control: %q", got)
	}
}

// The role is frozen at production; the shade resolves at paint (#70).
func TestATintedRowTakesTheShadeOfTheSchemeItIsPaintedIn(t *testing.T) {
	p := rowPaint{tinted: true}
	if got := paintLanguageRow("hola", p, 6, store.SchemeDark); !strings.Contains(got, languageDark) || strings.Contains(got, languageLight) {
		t.Errorf("dark: %q", got)
	}
	if got := paintLanguageRow("hola", p, 6, store.SchemeLight); !strings.Contains(got, languageLight) || strings.Contains(got, languageDark) {
		t.Errorf("light: %q", got)
	}
	if got := paintLanguageRow("hola", rowPaint{}, 6, store.SchemeLight); got != "hola" {
		t.Errorf("an untinted row is untouched: %q", got)
	}
}

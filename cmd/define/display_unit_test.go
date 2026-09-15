package main

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNextDisplayUnitPolicy(t *testing.T) {
	for _, tc := range []struct {
		text         string
		bytes, cells int
	}{
		{"", 0, 0}, {"a", 1, 1}, {"界", 3, 2}, {"\u0301", 2, 0},
		{"🇪", 4, 1}, {"🇪x", 4, 1}, {"🇪🇸🇮🇹", 8, 2},
		{"\xff", 1, 1}, {"\xf0\x9f", 1, 1}, {"🇪\x1b[32m🇸", 4, 1},
	} {
		n, w := nextDisplayUnit(tc.text)
		if n != tc.bytes || w != tc.cells {
			t.Errorf("unit(%q) = (%d,%d), want (%d,%d)", tc.text, n, w, tc.bytes, tc.cells)
		}
	}
	if got := visibleCells("🇪🇸🇮🇹🇺"); got != 5 {
		t.Fatalf("RI run width = %d", got)
	}
	if got := selectionPhysicalRows("🇪🇸🇮🇹🇺", 2); !reflect.DeepEqual(got, []string{"🇪🇸", "🇮🇹", "🇺"}) {
		t.Fatalf("RI run rows = %q", got)
	}
}

func TestFlagClickableSecondCellDoesNotBlockLaterRegion(t *testing.T) {
	got := markClickable("a🇪🇸b", []Region{{Col: 2, Width: 1}, {Col: 3, Width: 1}})
	want := "a" + underlineOn + "🇪🇸" + underlineOff + underlineOn + "b" + underlineOff
	if got != want {
		t.Fatalf("mark second flag cell then neighbor = %q, want %q", got, want)
	}
}

func TestFlagDisplayGeometry(t *testing.T) {
	t.Run("width and byte columns", func(t *testing.T) {
		if got := visibleCells("\x1b[32m🇪🇸 › \x1b[0m"); got != 5 {
			t.Fatalf("width = %d", got)
		}
		if got := visibleCells("[es] › "); got != 7 {
			t.Fatalf("fallback width = %d", got)
		}
		plain, cols := visibleIndex("a🇪🇸b")
		if plain != "a🇪🇸b" || !reflect.DeepEqual(cols, []int{0, 1, 1, 1, 1, 1, 1, 1, 1, 3}) {
			t.Fatalf("index = %q %v", plain, cols)
		}
	})
	t.Run("clip", func(t *testing.T) {
		for _, tc := range []struct {
			text  string
			width int
			want  string
		}{
			{"🇪🇸x", 1, ""}, {"🇪🇸x", 2, "🇪🇸"}, {"a🇪🇸x", 2, "a"},
			{"\x1b[32ma🇪🇸x", 2, "\x1b[32ma\x1b[0m"},
		} {
			if got := clipVisible(tc.text, tc.width); got != tc.want {
				t.Errorf("clip(%q,%d) = %q, want %q", tc.text, tc.width, got, tc.want)
			}
		}
	})
	t.Run("wrap", func(t *testing.T) {
		if got := selectionPhysicalRows("a🇪🇸b", 2); !reflect.DeepEqual(got, []string{"a", "🇪🇸", "b"}) {
			t.Fatalf("rows = %q", got)
		}
		if got := displayRows("a🇪🇸b", 2); got != 3 {
			t.Fatalf("row count = %d", got)
		}
		if got := clipSelectionRows("a🇪🇸b", 1, 2); got != "a\x1b[0m" {
			t.Fatalf("row clip = %q", got)
		}
		if got := selectionPhysicalRows("\x1b[32ma🇪🇸b", 2); !reflect.DeepEqual(got, []string{"\x1b[32ma", "\x1b[32m🇪🇸", "\x1b[32mb"}) {
			t.Fatalf("styled rows = %q", got)
		}
	})
	t.Run("selection", func(t *testing.T) {
		f := newSelectionFrame(4, 1, []selectionRow{{styled: "a🇪🇸b", selectable: true}})
		for _, col := range []int{1, 2} {
			p := selectionPoint{0, col}
			if got, err := selectedText(f, p, p); err != nil || got != "🇪🇸" {
				t.Errorf("col %d copied %q: %v", col, got, err)
			}
			if got := f.highlightRow(0, p, p); got != "a\x1b[7m🇪🇸\x1b[0mb" {
				t.Errorf("col %d highlight = %q", col, got)
			}
		}
	})
	t.Run("clickable and slice", func(t *testing.T) {
		if got := markClickable("a🇪🇸b", []Region{{Col: 1, Width: 1}}); got != "a"+underlineOn+"🇪🇸"+underlineOff+"b" {
			t.Fatalf("mark = %q", got)
		}
		if got := cellSlice("a🇪🇸b", 1, 1); got != "🇪🇸" {
			t.Fatalf("slice = %q", got)
		}
		if got := markClickable("a🇪🇸b", []Region{{Col: 3, Width: 1}}); got != "a🇪🇸"+underlineOn+"b"+underlineOff {
			t.Fatalf("neighbor mark = %q", got)
		}
	})
}

func FuzzFlagDisplayBoundaries(f *testing.F) {
	f.Add(uint8(4), uint8(2), uint8(0), uint8(1), "界e\u0301🇪🇸")
	f.Add(uint8(0), uint8(1), uint8(25), uint8(25), "\xff\xf0\x9f")
	f.Fuzz(func(t *testing.T, padding, width, first, second uint8, raw string) {
		if len(raw) > 4096 {
			return
		}
		for rest := raw; rest != ""; {
			n, w := nextDisplayUnit(rest)
			if n <= 0 || n > len(rest) || w < 0 || w > 2 {
				t.Fatalf("invalid progress (%d,%d) in %q", n, w, rest)
			}
			if utf8.ValidString(rest) && !utf8.ValidString(rest[:n]) {
				t.Fatalf("split UTF-8: %q", rest[:n])
			}
			rest = rest[n:]
		}
		flag := string([]rune{0x1f1e6 + rune(first%26), 0x1f1e6 + rune(second%26)})
		text := strings.Repeat("x", int(padding%32)) + flag + "z"
		cols := 1 + int(width%32)
		check := func(s string) {
			t.Helper()
			for _, r := range flag {
				if strings.ContainsRune(s, r) && !strings.Contains(s, flag) {
					t.Fatalf("split flag %q in %q", flag, s)
				}
			}
		}
		check(clipVisible(text, cols))
		for _, row := range selectionPhysicalRows(text, cols) {
			check(row)
		}
		frame := newSelectionFrame(len(text), 1, []selectionRow{{styled: text, selectable: true}})
		for col := int(padding % 32); col < int(padding%32)+2; col++ {
			p := selectionPoint{0, col}
			got, err := selectedText(frame, p, p)
			if err != nil || got != flag {
				t.Fatalf("copy = %q, want %q: %v", got, flag, err)
			}
		}
	})
}

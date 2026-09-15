package main

import "unicode/utf8"

// nextDisplayUnit reads one indivisible terminal unit after ANSI escapes have
// been skipped. Adjacent regional indicators form a two-cell flag; other runes
// retain cellWidth's policy. Producers keep each flag pair free of ANSI escapes.
func nextDisplayUnit(s string) (size, cells int) {
	if s == "" {
		return 0, 0
	}
	r, size := utf8.DecodeRuneInString(s)
	if r >= 0x1f1e6 && r <= 0x1f1ff {
		next, n := utf8.DecodeRuneInString(s[size:])
		if next >= 0x1f1e6 && next <= 0x1f1ff {
			return size + n, 2
		}
	}
	return size, cellWidth(r)
}

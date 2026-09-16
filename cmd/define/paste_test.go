package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// wholePaste frames a body the way a terminal in mode 2004 does.
func wholePaste(body string) []byte { return []byte(pasteStart + body + pasteEnd) }

func TestPasteScannerTakesAWholePaste(t *testing.T) {
	var s pasteScanner
	in := append(wholePaste("hello\nworld"), []byte("rest")...)
	k, used := s.scan(in)
	if k.Kind != KeyPaste {
		t.Fatalf("kind = %v, want KeyPaste", k.Kind)
	}
	if string(k.Raw) != "hello\nworld" {
		t.Errorf("Raw = %q, want %q", k.Raw, "hello\nworld")
	}
	if want := len(wholePaste("hello\nworld")); used != want {
		t.Errorf("used = %d, want %d — the paste and both markers, not the trailing text", used, want)
	}
}

// THE REGRESSION AN EARLIER DESIGN SHIPPED. readInput re-presents its WHOLE
// buffer after a short read and advances only on consumption
// (selection_input.go:174), so a scanner that also accumulated internally saw
// every byte twice. Driven the way the caller actually drives it: one scanner, a
// growing buffer, nothing consumed in between.
func TestPasteScannerSurvivesAReadBoundary(t *testing.T) {
	var s pasteScanner
	buf := []byte(pasteStart + "hello")
	if _, used := s.scan(buf); used != 0 {
		t.Fatalf("an unterminated paste consumed %d bytes; it must consume none", used)
	}
	buf = append(buf, []byte(" world"+pasteEnd)...) // the caller APPENDS; it does not advance
	k, used := s.scan(buf)
	if string(k.Raw) != "hello world" {
		t.Errorf("Raw = %q, want %q — the scanner double-counted the re-presented bytes", k.Raw, "hello world")
	}
	if used != len(buf) {
		t.Errorf("used = %d, want %d", used, len(buf))
	}
}

// A realistic paste in readInput's real 256-byte chunks (selection_input.go:161).
// An earlier design refused this at 900 bytes, well inside the cap, because its
// internal buffer grew quadratically.
func TestPasteScannerTakesAPasteDeliveredInChunks(t *testing.T) {
	body := strings.Repeat("a", 900)
	whole := wholePaste(body)
	var s pasteScanner
	var buf []byte
	for i := 0; i < len(whole); i += 256 {
		buf = append(buf, whole[i:min(i+256, len(whole))]...)
		k, used := s.scan(buf)
		if used == 0 {
			continue
		}
		if k.Kind != KeyPaste {
			t.Fatalf("kind = %v, want KeyPaste", k.Kind)
		}
		if string(k.Raw) != body {
			t.Fatalf("Raw length = %d, want %d", len(k.Raw), len(body))
		}
		return
	}
	t.Fatal("a 900-byte paste delivered in 256-byte chunks never completed")
}

// The cap is in RUNES. A byte cap would refuse a CJK paragraph at roughly a third
// of its length — on exactly the decks /lang exists for.
func TestPasteScannerCapsInRunesNotBytes(t *testing.T) {
	body := strings.Repeat("漢", maxPasteRunes-1) // three bytes each: far over any byte cap
	var s pasteScanner
	k, _ := s.scan(wholePaste(body))
	if k.Kind != KeyPaste {
		t.Fatalf("a %d-rune CJK paste was refused; the cap is counting bytes", maxPasteRunes-1)
	}
	if utf8.RuneCount(k.Raw) != maxPasteRunes-1 {
		t.Errorf("Raw = %d runes, want %d", utf8.RuneCount(k.Raw), maxPasteRunes-1)
	}
}

func TestPasteScannerRefusesAnOversizePasteItCanSeeWhole(t *testing.T) {
	var s pasteScanner
	k, used := s.scan(wholePaste(strings.Repeat("x", maxPasteRunes+1)))
	if k.Kind != KeyPasteRefused {
		t.Fatalf("kind = %v, want KeyPasteRefused", k.Kind)
	}
	if used == 0 {
		t.Error("a refused paste consumed nothing; its bytes would arrive as keystrokes")
	}
	if len(k.Raw) != 0 {
		t.Errorf("Raw = %q, want empty — a refusal carries no text", k.Raw)
	}
}

// An oversize paste is CONSUMED as it arrives, not held: otherwise readInput's
// buffer grows with the input, and the ARCH-CONSTRAINTS memory bound is a
// comment rather than a property.
func TestPasteScannerDrainsAnOversizePasteBoundedly(t *testing.T) {
	var s pasteScanner
	k, used := s.scan([]byte(pasteStart + strings.Repeat("x", maxPasteRunes*3)))
	if used == 0 {
		t.Fatal("an over-cap paste consumed nothing; the caller's buffer grows without bound")
	}
	if k.Kind != KeyUnknown {
		t.Errorf("kind = %v, want KeyUnknown while draining — it must be inert", k.Kind)
	}
	if !s.draining {
		t.Fatal("the scanner did not enter the drain")
	}
	k, used = s.scan([]byte("more body" + pasteEnd))
	if k.Kind != KeyPasteRefused || used == 0 {
		t.Errorf("the closer did not end the drain: kind=%v used=%d", k.Kind, used)
	}
	if s.draining {
		t.Error("the drain did not clear; the next real paste would be swallowed")
	}
}

// The drain must not cut a closer that straddles two reads. Same whole-or-nothing
// care decodeX10Mouse takes (key.go:330).
func TestTheDrainDoesNotCutAStraddlingCloser(t *testing.T) {
	var s pasteScanner
	if _, used := s.scan([]byte(pasteStart + strings.Repeat("x", maxPasteRunes*2))); used == 0 {
		t.Fatal("the over-cap scan consumed nothing")
	}
	half := pasteEnd[:3]
	if k, used := s.scan([]byte("tail" + half)); used >= len("tail"+half) {
		t.Errorf("the drain consumed into a partial closer (used=%d, kind=%v)", used, k.Kind)
	}
	k, _ := s.scan([]byte("tail" + pasteEnd))
	if k.Kind != KeyPasteRefused {
		t.Errorf("the straddling closer was lost: kind = %v", k.Kind)
	}
}

// ARCH-SECURE. The body is untrusted text on its way to the footer, which passes
// producer SGR through by construction (selection_frame.go:236-245). A pasted
// escape would recolour the passage and defeat the mark painting, which
// re-asserts over KNOWN producer SGR rather than arbitrary injected state. The
// scanner is where the bytes become a typed value.
func TestAPastedEscapeNeverLeavesTheBoundary(t *testing.T) {
	var s pasteScanner
	k, _ := s.scan(wholePaste("the \x1b[31mslow\x1b[0m precession"))
	if strings.ContainsRune(string(k.Raw), 0x1b) {
		t.Errorf("an escape survived the boundary: %q", k.Raw)
	}
	if got := string(k.Raw); got != "the slow precession" {
		t.Errorf("Raw = %q, want %q — stripping must remove only the escapes", got, "the slow precession")
	}
}

// Control characters go the same way, EXCEPT newline: a passage has lines.
func TestTheBoundaryKeepsNewlinesAndDropsOtherControls(t *testing.T) {
	var s pasteScanner
	k, _ := s.scan(wholePaste("a\nb\tc\x00d\x07e"))
	if got := string(k.Raw); got != "a\nb\tc de" && got != "a\nb\tcde" {
		t.Errorf("Raw = %q; want the newline and tab kept and NUL/BEL dropped", got)
	}
	if !strings.Contains(string(k.Raw), "\n") {
		t.Error("the newline was dropped; a passage has lines")
	}
}

// An embedded closer ends the paste — that IS the protocol, and the remainder is
// ordinary input. Decided rather than inherited: the remainder can contain \r,
// which submits.
func TestAnEmbeddedCloserEndsThePaste(t *testing.T) {
	var s pasteScanner
	k, used := s.scan([]byte(pasteStart + "first" + pasteEnd + "second" + pasteEnd))
	if string(k.Raw) != "first" {
		t.Errorf("Raw = %q, want %q — the first closer wins", k.Raw, "first")
	}
	if want := len(pasteStart + "first" + pasteEnd); used != want {
		t.Errorf("used = %d, want %d — the remainder is ordinary input", used, want)
	}
}

func TestTextThatIsNotAPasteIsNotOurs(t *testing.T) {
	var s pasteScanner
	if k, used := s.scan([]byte("\x1b[A")); used != 0 || k.Kind != KeyUnknown {
		t.Errorf("scan claimed a non-paste escape: kind=%v used=%d", k.Kind, used)
	}
	if k, used := s.scan([]byte("plain")); used != 0 || k.Kind != KeyUnknown {
		t.Errorf("scan claimed ordinary text: kind=%v used=%d", k.Kind, used)
	}
}

// ONE scanner across MANY calls — the stateful half that a fresh-scanner-per-
// iteration fuzz rule cannot reach. Invariants: the scanner never reports more
// bytes than it was shown, and no escape byte ever leaves it as passage text.
func FuzzPasteScannerAcrossCalls(f *testing.F) {
	f.Add([]byte(pasteStart+"hi"+pasteEnd), 3)
	f.Add([]byte(pasteStart+strings.Repeat("x", maxPasteRunes+5)+pasteEnd), 7)
	f.Add([]byte("\x1b[200~\x1b[201~"), 1)
	f.Fuzz(func(t *testing.T, in []byte, split int) {
		if split <= 0 {
			split = 1
		}
		var s pasteScanner
		var buf []byte
		for i := 0; i < len(in); i += split {
			buf = append(buf, in[i:min(i+split, len(in))]...)
			for {
				k, used := s.scan(buf)
				if used == 0 {
					break
				}
				if used > len(buf) {
					t.Fatalf("scan reported %d bytes consumed from a %d-byte buffer", used, len(buf))
				}
				if strings.ContainsRune(string(k.Raw), 0x1b) {
					t.Fatalf("an escape left the boundary as text: %q", k.Raw)
				}
				buf = buf[used:]
			}
		}
	})
}

// --- the decoder hook -------------------------------------------------------

func TestDecodeKeyTakesAPasteAsOneKey(t *testing.T) {
	in := append(wholePaste("hot dog"), 'x')
	k, used := decodeKey(in)
	if k.Kind != KeyPaste {
		t.Fatalf("kind = %v, want KeyPaste", k.Kind)
	}
	if string(k.Raw) != "hot dog" {
		t.Errorf("Raw = %q, want %q", k.Raw, "hot dog")
	}
	if used != len(in)-1 {
		t.Errorf("used = %d, want everything but the trailing x", used)
	}
}

// The standing bug this milestone fixes, asserted at its CAUSE (key.go:87)
// rather than where it was observed (the editor's ActSubmit).
func TestAPastedNewlineIsNotEnter(t *testing.T) {
	k, _ := decodeKey(wholePaste("a\nb"))
	if k.Kind == KeyEnter {
		t.Fatal("a pasted newline decoded as Enter; the paste would submit mid-text")
	}
	if string(k.Raw) != "a\nb" {
		t.Errorf("Raw = %q, want the newline preserved inside the paste", k.Raw)
	}
}

// PQ-1. The drain must survive a read boundary that lands on ORDINARY TEXT.
// A hook that consults the scanner only on 0x1b never sees it again — the head
// of the buffer is body bytes by then — and delivers the rest of an oversize
// paste as keystrokes, which is the exact failure the drain exists to prevent.
func TestAnOversizePasteKeepsDrainingAcrossReads(t *testing.T) {
	var d keyDecoder
	first := []byte(pasteStart + strings.Repeat("x", maxPasteRunes+10))
	if _, used := d.decode(first); used == 0 {
		t.Fatal("the over-cap decode consumed nothing")
	}
	k, used := d.decode([]byte("still body text, no closer yet"))
	if used == 0 {
		t.Fatal("draining stopped at a read boundary; the rest arrives as keystrokes")
	}
	if k.Kind == KeyRune {
		t.Fatalf("a drained body byte decoded as the rune %q", k.Rune)
	}
}

// decodeKey allocates a fresh decoder per call, which is what keeps its 38
// existing call sites — and both fuzz targets — free of paste state.
func TestAFreshDecodeKeyIsNotMidPaste(t *testing.T) {
	if _, used := decodeKey([]byte(pasteStart + strings.Repeat("x", maxPasteRunes+10))); used == 0 {
		t.Fatal("the over-cap decode consumed nothing")
	}
	if k, _ := decodeKey([]byte("a")); k.Kind != KeyRune || k.Rune != 'a' {
		t.Errorf("paste state leaked into a fresh decodeKey: %v %q", k.Kind, k.Rune)
	}
}

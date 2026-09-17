package main

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// wholePaste frames a body the way a terminal in mode 2004 does.
func wholePaste(body string) []byte { return []byte(pasteStart + body + pasteEnd) }

func TestPasteScannerTakesAWholePaste(t *testing.T) {
	var s pasteScanner
	in := append(wholePaste("hello\nworld"), []byte("rest")...)
	k, used, _ := s.scan(in)
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
	if _, used, _ := s.scan(buf); used != 0 {
		t.Fatalf("an unterminated paste consumed %d bytes; it must consume none", used)
	}
	buf = append(buf, []byte(" world"+pasteEnd)...) // the caller APPENDS; it does not advance
	k, used, _ := s.scan(buf)
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
		k, used, _ := s.scan(buf)
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
	k, _, _ := s.scan(wholePaste(body))
	if k.Kind != KeyPaste {
		t.Fatalf("a %d-rune CJK paste was refused; the cap is counting bytes", maxPasteRunes-1)
	}
	if utf8.RuneCount(k.Raw) != maxPasteRunes-1 {
		t.Errorf("Raw = %d runes, want %d", utf8.RuneCount(k.Raw), maxPasteRunes-1)
	}
}

func TestPasteScannerRefusesAnOversizePasteItCanSeeWhole(t *testing.T) {
	var s pasteScanner
	k, used, _ := s.scan(wholePaste(strings.Repeat("x", maxPasteRunes+1)))
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
//
// The trigger is the BYTE bound, not the rune cap — those became separate
// predicates in round 2 (BR-12), because the rune cap cannot be evaluated on a
// buffer that may end mid-rune.
func TestPasteScannerDrainsAnOversizePasteBoundedly(t *testing.T) {
	var s pasteScanner
	k, used, _ := s.scan([]byte(pasteStart + strings.Repeat("x", maxPasteBytes+50)))
	if used == 0 {
		t.Fatal("an over-cap paste consumed nothing; the caller's buffer grows without bound")
	}
	if k.Kind != KeyUnknown {
		t.Errorf("kind = %v, want KeyUnknown while draining — it must be inert", k.Kind)
	}
	if !s.draining {
		t.Fatal("the scanner did not enter the drain")
	}
	k, used, _ = s.scan([]byte("more body" + pasteEnd))
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
	if _, used, _ := s.scan([]byte(pasteStart + strings.Repeat("x", maxPasteBytes+50))); used == 0 {
		t.Fatal("the over-cap scan consumed nothing")
	}
	half := pasteEnd[:3]
	if k, used, _ := s.scan([]byte("tail" + half)); used >= len("tail"+half) {
		t.Errorf("the drain consumed into a partial closer (used=%d, kind=%v)", used, k.Kind)
	}
	k, _, _ := s.scan([]byte("tail" + pasteEnd))
	if k.Kind != KeyPasteRefused {
		t.Errorf("the straddling closer was lost: kind = %v", k.Kind)
	}
}

// ARCH-SECURE. The body is untrusted text on its way to the screen — today the
// line editor, which opens its own styles around what it draws, so a pasted
// escape would leak out of the line and repaint the frame. The scanner is where
// the bytes become a typed value, which keeps the rule true for any later surface
// without it having to be restated there.
func TestAPastedEscapeNeverLeavesTheBoundary(t *testing.T) {
	var s pasteScanner
	k, _, _ := s.scan(wholePaste("the \x1b[31mslow\x1b[0m precession"))
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
	k, _, _ := s.scan(wholePaste("a\nb\tc\x00d\x07e"))
	// ONE outcome, not either: a test that accepts both pins neither, and NUL/BEL
	// handling is exactly the thing a later refactor would change silently.
	//
	// A TAB BECOMES A SPACE. It is the one character whose width depends on where
	// it sits, and nextDisplayUnit counts it as one cell — so a surviving tab put
	// every later word's column out and clicks landed on the wrong word.
	if got, want := string(k.Raw), "a\nb cde"; got != want {
		t.Errorf("Raw = %q, want %q — newline kept, tab expanded, NUL and BEL dropped", got, want)
	}
	if !strings.Contains(string(k.Raw), "\n") {
		t.Error("the newline was dropped; a passage has lines")
	}
	if strings.ContainsRune(string(k.Raw), '\t') {
		t.Error("a tab survived; its display width depends on the column, so every word after it lands wrong")
	}
}

// An embedded closer ends the paste — that IS the protocol, and the remainder is
// ordinary input. Decided rather than inherited: the remainder can contain \r,
// which submits.
func TestAnEmbeddedCloserEndsThePaste(t *testing.T) {
	var s pasteScanner
	k, used, _ := s.scan([]byte(pasteStart + "first" + pasteEnd + "second" + pasteEnd))
	if string(k.Raw) != "first" {
		t.Errorf("Raw = %q, want %q — the first closer wins", k.Raw, "first")
	}
	if want := len(pasteStart + "first" + pasteEnd); used != want {
		t.Errorf("used = %d, want %d — the remainder is ordinary input", used, want)
	}
}

func TestTextThatIsNotAPasteIsNotOurs(t *testing.T) {
	var s pasteScanner
	if k, used, _ := s.scan([]byte("\x1b[A")); used != 0 || k.Kind != KeyUnknown {
		t.Errorf("scan claimed a non-paste escape: kind=%v used=%d", k.Kind, used)
	}
	if k, used, _ := s.scan([]byte("plain")); used != 0 || k.Kind != KeyUnknown {
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
				k, used, _ := s.scan(buf)
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
	first := []byte(pasteStart + strings.Repeat("x", maxPasteBytes+50))
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
	if _, used := decodeKey([]byte(pasteStart + strings.Repeat("x", maxPasteBytes+50))); used == 0 {
		t.Fatal("the over-cap decode consumed nothing")
	}
	if k, _ := decodeKey([]byte("a")); k.Kind != KeyRune || k.Rune != 'a' {
		t.Errorf("paste state leaked into a fresh decodeKey: %v %q", k.Kind, k.Rune)
	}
}

// --- the editor ------------------------------------------------------------

// A paste is text typed at once. It goes in at the cursor, ATOMICALLY: one key,
// one insertion, one history-walk reset — not N of each.
func TestApplyInsertsAPasteAtTheCursor(t *testing.T) {
	e := NewEditor()
	for _, k := range runes("ab") {
		e, _ = Apply(e, k, candidates{})
	}
	e.Cursor = 1
	e, act := Apply(e, Key{Kind: KeyPaste, Raw: []byte("XY")}, candidates{})
	if act != ActNone {
		t.Errorf("act = %v, want ActNone — a paste is not a submission", act)
	}
	if got := e.String(); got != "aXYb" {
		t.Errorf("line = %q, want %q", got, "aXYb")
	}
	if e.Cursor != 3 {
		t.Errorf("cursor = %d, want 3 — it follows the inserted text", e.Cursor)
	}
}

// THE BUG THIS MILESTONE FIXES, at the editor rather than the decoder: a pasted
// newline must not submit. The line editor holds ONE line, so an interior
// newline becomes a space — which is what parseREPLLine would do to it anyway.
func TestAPastedNewlineDoesNotSubmitAndBecomesASpace(t *testing.T) {
	e, act := Apply(NewEditor(), Key{Kind: KeyPaste, Raw: []byte("hot\ndog")}, candidates{})
	if act != ActNone {
		t.Fatalf("act = %v, want ActNone — the paste submitted mid-text", act)
	}
	if got := e.String(); got != "hot dog" {
		t.Errorf("line = %q, want %q", got, "hot dog")
	}
}

// A refusal leaves the line exactly as it was: it carries no text, and a paste
// the user was told was too long must not half-arrive.
func TestARefusedPasteLeavesTheLineAlone(t *testing.T) {
	e := NewEditor()
	for _, k := range runes("keep") {
		e, _ = Apply(e, k, candidates{})
	}
	e, act := Apply(e, Key{Kind: KeyPasteRefused}, candidates{})
	if act != ActNone || e.String() != "keep" {
		t.Errorf("line = %q act = %v, want %q ActNone", e.String(), act, "keep")
	}
}

// End to end through the loop: a HEADWORD-SHAPED paste reaches the line, and
// the session does not submit or look anything up on the way.
//
// A paste goes into the line and does not submit. Apply's own contract — one
// atomic insertion, interior newlines flattened — is asserted directly in
// TestAPastedNewlineDoesNotSubmitAndBecomesASpace; this is the loop-level case.
func TestEditorLoopTakesAPasteWithoutSubmitting(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	view := paintInto(&out)
	ks := keySeq(Key{Kind: KeyPaste, Raw: []byte("hot dog")})
	runEditor(t.Context(), ks, nil, rig.deps, opt,
		console{view: view, finish: finish, stdout: &out, stderr: &errb})
	if !strings.Contains(out.String(), "hot dog") {
		t.Errorf("the paste never reached the line:\n%s", out.String())
	}
	if errb.Len() != 0 {
		t.Errorf("a well-formed paste wrote to stderr: %q", errb.String())
	}
}

// A refusal is REPORTED. A paste that simply vanished would read as a broken
// terminal, and there is no other way to learn the limit.
func TestEditorLoopReportsARefusedPaste(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	view := paintInto(&out)
	ks := keySeq(Key{Kind: KeyPasteRefused})
	runEditor(t.Context(), ks, nil, rig.deps, opt,
		console{view: view, finish: finish, stdout: &out, stderr: &errb})
	if !strings.Contains(errb.String(), "longer than") {
		t.Errorf("the refusal was silent; stderr = %q", errb.String())
	}
}

// C1 from the M1 boundary review. An unterminated ESC[200~ used to deafen the
// input path FOREVER. Raw mode disables ISIG, so Ctrl-C is reachable only as a
// decoded KeyInterrupt — the program could not be quit from the keyboard.
//
// BOTH halves, and the second one is the point: round 4 caught that the earlier
// "draining" row never drained, because the trailing \x03 fired the abandon rule
// before the byte bound was reached. Each row now ASSERTS which exit it took, so
// a case cannot silently go unexercised again.
func TestAnUnterminatedPasteDoesNotSwallowEnterOrInterrupt(t *testing.T) {
	t.Run("under the byte bound", func(t *testing.T) {
		assertEscapesReach(t, pasteStart+"hello\r\x03", pasteAbandoned)
	})
	t.Run("after the drain has started", func(t *testing.T) {
		var s pasteScanner
		// No control byte here, so the byte bound — not the abandon rule —
		// decides. Pinned by the exit, which is what the old row lacked.
		_, used, exit := s.scan([]byte(pasteStart + strings.Repeat("x", maxPasteBytes+50)))
		if exit != pasteDrainStarted || used == 0 {
			t.Fatalf("exit = %v used = %d; this row must reach the drain", exit, used)
		}
		if !s.draining {
			t.Fatal("the scanner is not draining; the rest of this row is vacuous")
		}
		// Now the keyboard must still work.
		var d keyDecoder
		d.paste = s
		assertKeysEmerge(t, &d, "more body\r\x03")
	})
}

// assertEscapesReach drives the whole input through one decoder and requires that
// Enter and the interrupt both come out.
func assertEscapesReach(t *testing.T, in string, want pasteExit) {
	t.Helper()
	var s pasteScanner
	if _, _, exit := s.scan([]byte(in)); exit != want {
		t.Fatalf("exit = %v, want %v — this row is not exercising the case it names", exit, want)
	}
	var d keyDecoder
	assertKeysEmerge(t, &d, in)
}

func assertKeysEmerge(t *testing.T, d *keyDecoder, in string) {
	t.Helper()
	buf := []byte(in)
	var got []KeyKind
	for len(buf) > 0 {
		k, used := d.decode(buf)
		if used == 0 {
			break
		}
		got = append(got, k.Kind)
		buf = buf[used:]
	}
	if !slices.Contains(got, KeyEnter) {
		t.Errorf("Enter never emerged: %v", got)
	}
	if !slices.Contains(got, KeyInterrupt) {
		t.Errorf("Ctrl-C never emerged — the program cannot be quit from the keyboard: %v", got)
	}
}

// Every exit scan can take is reached by a named row. DERIVED from the
// registry's extent, so an exit added without a row reddens here — the same move
// TestEveryRegionKindIsActionable makes, and the guard that would have caught the
// drain shipping untested.
func TestEveryPasteExitIsExercised(t *testing.T) {
	seen := map[pasteExit]string{}
	for _, tc := range []struct {
		name  string
		drive func(*pasteScanner) pasteExit
	}{
		{"ordinary text", func(s *pasteScanner) pasteExit {
			_, _, e := s.scan([]byte("plain"))
			return e
		}},
		{"a prefix", func(s *pasteScanner) pasteExit {
			_, _, e := s.scan([]byte(pasteStart + "partial"))
			return e
		}},
		{"a control byte inside an open paste", func(s *pasteScanner) pasteExit {
			_, _, e := s.scan([]byte(pasteStart + "oops\x03"))
			return e
		}},
		{"a complete paste", func(s *pasteScanner) pasteExit {
			_, _, e := s.scan(wholePaste("fine"))
			return e
		}},
		{"a complete paste over the rune cap", func(s *pasteScanner) pasteExit {
			_, _, e := s.scan(wholePaste(strings.Repeat("x", maxPasteRunes+1)))
			return e
		}},
		{"over the byte bound", func(s *pasteScanner) pasteExit {
			_, _, e := s.scan([]byte(pasteStart + strings.Repeat("x", maxPasteBytes+50)))
			return e
		}},
		{"the drain reaching its closer", func(s *pasteScanner) pasteExit {
			s.scan([]byte(pasteStart + strings.Repeat("x", maxPasteBytes+50)))
			_, _, e := s.scan([]byte("tail" + pasteEnd))
			return e
		}},
	} {
		var s pasteScanner
		seen[tc.drive(&s)] = tc.name
	}
	for e := pasteExit(0); e < numPasteExits; e++ {
		if _, ok := seen[e]; !ok {
			t.Errorf("exit %d is reachable in scan and no row here exercises it — "+
				"that is how the drain came to ship with a green test that never drained", e)
		}
	}
}

// A well-formed paste still wins: the abandon rule must not fire on a closer
// that is already present.
func TestAClosedPasteIsUnaffectedByTheAbandonRule(t *testing.T) {
	k, used := decodeKey(append(wholePaste("fine"), 0x03))
	if k.Kind != KeyPaste || string(k.Raw) != "fine" {
		t.Errorf("a closed paste followed by Ctrl-C decoded as %v %q", k.Kind, k.Raw)
	}
	if used != len(wholePaste("fine")) {
		t.Errorf("used = %d, want the paste only", used)
	}
}

// BR-3 from the M1 boundary review: readInput's long-lived keyDecoder was the
// one piece of production state this milestone introduced, and reverting it to
// the stateless decodeKey left the whole suite green.
//
// The FIRST version of this test did not fix that — it split a small paste
// across reads and passed under the mutation. The reason is the scanner's own
// contract: it accumulates nothing and the caller re-presents its whole buffer,
// so a fresh decoder handles a split paste perfectly well. The buffer was never
// advanced, so nothing was lost.
//
// What genuinely needs continuity is the DRAIN, which is the only state the
// buffer cannot carry. An over-cap paste CONSUMES as it goes, so the caller does
// advance — and a per-call decoder then starts the next read with no paste open
// and hands the discarded body to the line as keystrokes. Verified by mutation:
// this test goes red when `dec.decode` is replaced by `decodeKey`.
func TestReadKeysCarriesTheDrainAcrossReads(t *testing.T) {
	pr, pw := io.Pipe()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	keys := readKeys(ctx, pr, &interrupter{fn: cancel})

	go func() {
		pw.Write([]byte(pasteStart + strings.Repeat("x", maxPasteBytes+50)))
		time.Sleep(50 * time.Millisecond)
		pw.Write([]byte("tail body" + pasteEnd))
	}()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case k := <-keys:
			switch k.Kind {
			case KeyPasteRefused:
				return // the drain survived and ended at its closer
			case KeyRune:
				t.Fatalf("a drained body byte reached the line as the rune %q — the drain did not survive the read", k.Rune)
			}
		case <-deadline:
			t.Fatal("the drain never ended")
		}
	}
}

// BR-6: the plan required this to be a DECISION rather than an inherited side
// effect. A KeyPaste is a non-pointer, non-KeyUnknown key, so pointerRouter.route
// cancels any gesture in flight (selection_input.go:47). That is right — a paste
// replaces the passage, so a drag over the old one means nothing — but it is only
// right on purpose.
//
// Asserted through route rather than cancelPointerInput. The first version called
// cancelPointerInput directly, which cancels for every kind but KeyUnknown, so it
// could not tell whether route had exempted KeyPaste: adding "&& k.Kind !=
// KeyPaste" to route's condition left it green. Mutation-verified now.
func TestAPasteCancelsALiveDrag(t *testing.T) {
	live := newLiveScreen(&bytes.Buffer{}, 24, 80)
	defer live.Stop()
	live.Draw("\u203a ", []string{"the slow precession of the equinox"})
	router := newPointerRouter(live, nil)

	live.pointerLocked(selectionPress, selectionPoint{row: 0, col: 0})
	live.pointerLocked(selectionMotion, selectionPoint{row: 0, col: 8})
	live.mu.Lock()
	active := live.gesture.active
	live.mu.Unlock()
	if !active {
		t.Fatal("the drag never started; the rest of this test would be vacuous")
	}

	// THROUGH route, not cancelPointerInput. route is where the exemption would
	// live, and cancelPointerInput cancels for every kind but KeyUnknown — so a
	// test calling it directly cannot tell whether route let KeyPaste past.
	router.route(Key{Kind: KeyPaste, Raw: []byte("a new passage entirely")})
	live.mu.Lock()
	still := live.gesture.active
	live.mu.Unlock()
	if still {
		t.Error("a paste left a drag in flight; its release would select across replaced text")
	}
}

// BR-12 from the M1 boundary review round 2. The rune cap used to be evaluated
// on a buffer that could end mid-rune, and utf8.RuneCount counts each orphan
// byte as a RuneError — so a 1000-rune CJK paste split at the wrong byte counted
// 1001, latched the drain, and was refused. A false refusal on exactly the decks
// the rune cap exists for.
//
// Driven at every split point, because the bug only appears at some of them.
func TestALegalCJKPasteIsNotRefusedAtAnySplit(t *testing.T) {
	body := strings.Repeat("漢", maxPasteRunes)
	whole := wholePaste(body)
	for _, split := range []int{256, 512, 1000, 2999, 3001, len(whole) - 1} {
		if split <= 0 || split >= len(whole) {
			continue
		}
		var s pasteScanner
		if k, used, _ := s.scan(whole[:split]); used != 0 {
			t.Fatalf("split %d: the partial buffer consumed %d as %v; it must wait", split, used, k.Kind)
		}
		k, _, _ := s.scan(whole)
		if k.Kind != KeyPaste {
			t.Errorf("split %d: a legal %d-rune paste decoded as %v", split, maxPasteRunes, k.Kind)
		}
	}
}

// BR-16: the paste mode enable was production wiring nothing exercised. newConsole
// is where the modes are taken; this asserts the paste enable is among them,
// beside the alt screen and the mouse.
func TestNewConsoleEnablesBracketedPaste(t *testing.T) {
	var control strings.Builder
	sess := &rawSession{control: &control}
	sess.enterModes()
	if got := control.String(); !strings.Contains(got, pasteOn) {
		t.Errorf("bracketed paste was never enabled: %q", got)
	}
}

// BR-15: a paste during a review sitting is IGNORED, and that is a decision.
// Mode 2004 is on for this surface too, so a paste really does arrive; a sitting
// takes keystrokes and has no field for prose, so inserting the body would answer
// the question with whatever was on the clipboard.
func TestAPasteDuringASittingIsIgnored(t *testing.T) {
	for _, k := range []Key{{Kind: KeyPaste, Raw: []byte("some pasted prose")}, {Kind: KeyPasteRefused}} {
		if in, ok := toInput(k); ok {
			t.Errorf("%v produced the sitting input %v; a sitting takes keystrokes, not text", k.Kind, in.Kind)
		}
	}
}

// BR-17: the refusal notice is a write between prompts, so it clears the frame
// first. A write with the prompt on screen lands inside it — the invariant
// TestNothingIsWrittenWhileAPromptIsShown pins for every other write this loop
// makes, and the first version of this notice was the exception.
func TestTheRefusalNoticeDoesNotLandInsideThePrompt(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	view := paintInto(&out)
	runEditor(t.Context(), keySeq(Key{Kind: KeyPasteRefused}), nil, rig.deps, opt,
		console{view: view, finish: finish, stdout: &out, stderr: &errb})

	drawn := view.prompts
	if len(drawn) < 2 {
		t.Fatalf("the loop drew %d prompts; the notice did not clear the frame", len(drawn))
	}
	if drawn[len(drawn)-2] != "" {
		t.Errorf("the prompt was still on screen when the notice was written: %q", drawn[len(drawn)-2])
	}
}

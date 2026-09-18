package main

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
)

// Every consumer of the new key kind treats it as a REPORT (#70). The loops are
// fed the reply's BYTES through readInput, so the decoder, the loop and the
// consumer are one path — a hand-built Key could not see the decoder go wrong.

const (
	lightReplyST  = "\x1b]11;rgb:ffff/ffff/ffff\x1b\\"
	lightReplyBEL = "\x1b]11;rgb:ffff/ffff/ffff\x07"
	rgbaReply     = "\x1b]11;rgba:ffff/ffff/ffff/ffff\x1b\\"
)

// replyEditor drives runEditor over input BYTES, on a live screen already
// showing one tinted row, and returns the looked-up words and the last frame.
func replyEditor(t *testing.T, st schemeState, input string) ([]string, string) {
	t.Helper()
	d := testDeps(t)
	d.scheme = newSchemeHolder(st)
	counted := &countingDict{inner: d.dict}
	d.dict = counted
	var terminal syncBuf
	var errout bytes.Buffer
	live := newLiveScreen(&terminal, 24, 80)
	live.interval = -1
	live.attachScheme(d.scheme)
	if err := live.WriteOutput(renderedOutput{text: "hola\n", rows: []rowPaint{{tinted: true}}}); err != nil {
		t.Fatal(err)
	}
	keys := readInput(t.Context(), strings.NewReader(input), nil, nil)
	opt := options{noAudio: true, width: 80, rows: 24, tty: true, color: true, tintOn: true}
	runEditor(t.Context(), keys, &interrupter{}, d, opt, console{view: live, stdout: live, stderr: &errout, finish: live.Stop})
	return counted.words, lastFrame(terminal.String())
}

func TestRawEditorBackgroundReplyRepaints(t *testing.T) {
	for _, tc := range []struct {
		name  string
		st    schemeState
		reply string
		shade string
	}{
		{"light reply, ST", schemeState{}, lightReplyST, languageLight},
		{"light reply, BEL", schemeState{}, lightReplyBEL, languageLight},
		{"rgba is swallowed, nothing detected", schemeState{}, rgbaReply, languageDark},
		{"#hex is swallowed, nothing detected", schemeState{}, "\x1b]11;#ffffff\x07", languageDark},
		{"Alt-] then typing is unchanged", schemeState{}, "\x1b]", languageDark},
		{"a flag choice outranks the reply", schemeState{}.withChoice(store.SchemeDark, choiceFlag), lightReplyST, languageDark},
	} {
		t.Run(tc.name, func(t *testing.T) {
			words, _ := replyEditor(t, tc.st, tc.reply+"parrot\r")
			if !reflect.DeepEqual(words, []string{"parrot"}) {
				t.Fatalf("looked up %q; a reply byte reached the line", words)
			}
			// The frame after the reply but before the lookup scrolls the row away,
			// so ask the screen's own record of it: the repaint of history.
			_, frame := replyEditor(t, tc.st, tc.reply)
			row := holaRow(frame)
			other := languageLight
			if tc.shade == languageLight {
				other = languageDark
			}
			if !strings.Contains(row, tc.shade) || strings.Contains(row, other) {
				t.Fatalf("the row on screen is %q, want the %q shade", row, tc.shade)
			}
		})
	}
}

// A sitting takes the reply as a report: exactly the intended answer is
// recorded, never a digit out of "11;rgb:…".
func TestASittingIgnoresABackgroundReply(t *testing.T) {
	for _, tc := range []struct {
		name   string
		reply  string
		scheme store.Scheme
	}{
		{"rgb", lightReplyST, store.SchemeLight},
		{"rgba", rgbaReply, store.SchemeDark},
		{"#hex", "\x1b]11;#ffffff\x07", store.SchemeDark},
		// Alt-] then the answer key: ESC ] 1 waits for "1;", the next byte aborts
		// the swallow, and the key still answers.
		{"Alt-] then the answer", "\x1b]", store.SchemeDark},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, opt, st := playRig(t, "sycophantic", "ephemeral")
			d.scheme = newSchemeHolder(schemeState{})
			qs, held := questionsFor(t, d, opt)
			var out, errb bytes.Buffer
			input := "\r" + tc.reply + gradeKey(t, qs[0], play.Correct) + "\x03"
			playSession(t.Context(), d, opt, play.NewSession(qs), held,
				readInput(t.Context(), strings.NewReader(input), nil, nil), playbackConsole(&out, &errb))
			events := reviewEvents(t, st)
			if len(events) != 1 || !events[0].Correct {
				t.Fatalf("recorded %+v; want exactly the one correct answer", events)
			}
			if got := d.scheme.Scheme(); got != tc.scheme {
				t.Fatalf("scheme after the reply = %q, want %q", got, tc.scheme)
			}
		})
	}
}

// The SITTING's own repaint (the M3 review's BR-10): a consumer's test checks
// what that consumer paints, not only the shared state it writes — the shared
// state is also what PaintedTranscript re-reads, so it cannot show a repaint.
// A tinted row on the sitting's screen, then the reply: the frame the sitting
// PAINTS must carry the new shade.
func TestASittingRepaintsOnABackgroundReply(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	d.scheme = newSchemeHolder(schemeState{})
	var tty syncBuf
	parent := newLiveScreen(&tty, 24, 80)
	parent.attachScheme(d.scheme)
	defer parent.Stop()
	router := newPointerRouter(parent, nil)
	defer router.Stop()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	keys := make(chan Key, 2)
	done := make(chan struct{})
	go func() {
		defer close(done)
		sittingInPlace(ctx, d, opt, keys, &interrupter{}, parent, nil, &tty, io.Discard, router)
	}()
	var nested *liveScreen
	waitFor(t, func() bool {
		router.mu.Lock()
		defer router.mu.Unlock()
		nested = router.active
		return nested != nil && nested != parent
	})
	nested.interval = -1
	if err := nested.WriteOutput(renderedOutput{text: "hola\n", rows: []rowPaint{{tinted: true}}}); err != nil {
		t.Fatal(err)
	}
	keys <- Key{Kind: KeyBackground, Background: store.SchemeLight}
	waitFor(t, func() bool { return strings.Contains(holaRow(lastFrame(tty.String())), languageLight) })
	keys <- Key{Kind: KeyInterrupt}
	<-done
}

// A reply heard during a /play sitting is in force in the editor after it.
func TestAReplyDuringPlayReachesTheEditor(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	d.scheme = newSchemeHolder(schemeState{})
	var tty syncBuf
	var editor *liveScreen
	con := newConsole(t.Context(), d, &rawSession{control: io.Discard}, &tty,
		func(w io.Writer, rows, cols int) *liveScreen {
			editor = newLiveScreen(w, rows, cols)
			return editor
		}, false)
	if err := writeOutput(con.stdout, renderedOutput{text: "hola\n", rows: []rowPaint{{tinted: true}}}, 20, store.SchemeDark); err != nil {
		t.Fatal(err)
	}
	keys := keySeq(Key{Kind: KeyBackground, Background: store.SchemeLight}, Key{Kind: KeyInterrupt})
	con.newSitting(t.Context(), d, opt, keys, &interrupter{}, io.Discard)
	if con.finish != nil {
		defer con.finish()
	}
	if v, src := d.scheme.Load().effective(); v != store.SchemeLight || src != sourceDetected {
		t.Fatalf("after the sitting the scheme is %s/%v, want light, detected", v, src)
	}
	// The frame the editor PAINTS on resume, not PaintedTranscript (which
	// re-reads the holder and so would pass without any repaint).
	if row := holaRow(lastFrame(tty.String())); !strings.Contains(row, languageLight) || strings.Contains(row, languageDark) {
		t.Fatalf("the editor did not repaint in the sitting's reply: %q", row)
	}
}

// A reply arriving mid-drag is not input: the selection survives it.
func TestAReplyMidDragKeepsTheSelection(t *testing.T) {
	live := newLiveScreen(io.Discard, 5, 40)
	live.Draw("select me", nil)
	router := newPointerRouter(live, newMemoryClipboard())
	defer router.Stop()
	router.route(Key{Kind: KeyPointerPress})
	router.route(Key{Kind: KeyPointerMotion, Col: 3})
	router.route(Key{Kind: KeyBackground, Background: store.SchemeLight})
	live.mu.Lock()
	dragging := live.gesture.dragging
	live.mu.Unlock()
	if !dragging {
		t.Fatal("a background report cancelled the drag")
	}
}

// A report dropped on a full input channel posts no notice and does not set
// the latch that suppresses the NEXT real key's notice.
func TestADroppedReplyIsSilent(t *testing.T) {
	live := newLiveScreen(io.Discard, 5, 40)
	router := newPointerRouter(live, nil)
	defer router.Stop()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	pr, pw := io.Pipe()
	defer pw.Close()
	_ = readInput(ctx, pr, nil, router) // never drained: the channel fills
	notice := func() string { live.mu.Lock(); defer live.mu.Unlock(); return live.selectionNotice }
	io.WriteString(pw, strings.Repeat("x", 256)) // exactly the channel's capacity
	io.WriteString(pw, lightReplyBEL)
	// A zero-length write is delivered as a Read, so its return proves readInput
	// finished the reply chunk and came back for more.
	pw.Write(nil)
	if n := notice(); n != "" {
		t.Fatalf("a dropped report posted %q", n)
	}
	io.WriteString(pw, "y")
	waitFor(t, func() bool { return notice() != "" })
}

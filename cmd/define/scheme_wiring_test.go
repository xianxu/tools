package main

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

// The scheme holder reaches a screen only where production ATTACHES it, so these
// drive the two places that do — newConsole and sittingInPlace — rather than a
// screen a test attached itself, which could not see the attach go missing
// (TestNewConsoleEnablesEveryMode records the same failure for modes).

func TestNewConsoleAttachesTheSchemeHolder(t *testing.T) {
	var control strings.Builder
	sess := &rawSession{control: &control}
	d := testDeps(t)
	d.scheme = holderFor(store.SchemeLight)
	var screen *liveScreen
	con := newConsole(t.Context(), d, sess, io.Discard,
		func(tty io.Writer, rows, cols int) *liveScreen {
			screen = newLiveScreen(tty, rows, cols)
			return screen
		}, false)
	if con.finish != nil {
		defer con.finish()
	}
	// The scheme argument is for non-screen sinks; a screen paints its own.
	if err := writeOutput(con.stdout, renderedOutput{text: "hola\n", rows: []rowPaint{{tinted: true}}}, 20, store.SchemeDark); err != nil {
		t.Fatal(err)
	}
	// Through the screen's lock, not a buffer a throttled flush may be writing.
	if tr := screen.PaintedTranscript(); !strings.Contains(tr, languageLight) || strings.Contains(tr, languageDark) {
		t.Fatalf("a console built by newConsole did not paint in the process's scheme: %q", tr)
	}
}

func TestASittingPaintsInTheEditorsScheme(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	d.scheme = holderFor(store.SchemeLight)
	var tty syncBuf
	parent := newLiveScreen(&tty, 24, 80)
	parent.attachScheme(d.scheme)
	defer parent.Stop()
	router := newPointerRouter(parent, nil)
	defer router.Stop()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	keys := make(chan Key, 1)
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
	if err := nested.WriteOutput(renderedOutput{text: "hola\n", rows: []rowPaint{{tinted: true}}}); err != nil {
		t.Fatal(err)
	}
	if tr := nested.PaintedTranscript(); !strings.Contains(tr, languageLight) || strings.Contains(tr, languageDark) {
		t.Fatalf("the sitting's screen did not paint in the editor's scheme: %q", tr)
	}
	keys <- Key{Kind: KeyInterrupt} // end the sitting
	<-done
}

// newConsole asks EXACTLY the terminal's queries, after the modes, when asked
// to — one recorder for the control stream and the screen, as production writes
// both to the same tty. Derived from terminalQueries, so the sends and the list
// cannot drift (#67's lesson with enabledModes).
func TestNewConsoleAsksEveryQuery(t *testing.T) {
	var queries strings.Builder
	for _, q := range terminalQueries {
		queries.WriteString(q.query)
	}
	for _, ask := range []bool{true, false} {
		var tty syncBuf
		sess := &rawSession{control: &tty}
		con := newConsole(t.Context(), testDeps(t), sess, &tty,
			func(w io.Writer, rows, cols int) *liveScreen { return newLiveScreen(w, rows, cols) }, ask)
		var modes strings.Builder
		for i := len(enabledModes) - 1; i >= 0; i-- {
			modes.WriteString(enabledModes[i].on)
		}
		got := strings.TrimPrefix(tty.String(), modes.String())
		if !strings.HasPrefix(tty.String(), modes.String()) {
			t.Fatalf("the modes were not written first: %q", tty.String())
		}
		if ask && !strings.HasPrefix(got, queries.String()) {
			t.Errorf("asked, but after the modes came %q, want %q", got, queries.String())
		}
		if !ask && strings.Contains(tty.String(), backgroundQuery) {
			t.Errorf("not asked, yet the query was sent: %q", tty.String())
		}
		if con.finish != nil {
			con.finish()
		}
	}
}

func TestWantsBackground(t *testing.T) {
	on := options{tty: true, color: true, tintOn: true}
	if !wantsBackground(on) {
		t.Fatal("a colour terminal with the tint on asks")
	}
	for name, o := range map[string]options{
		"tint off": {tty: true, color: true},
		"-raw":     {tty: true, color: true, tintOn: true, raw: true},
		"no tty":   {color: true, tintOn: true},
		"no color": {tty: true, tintOn: true},
	} {
		if wantsBackground(o) {
			t.Errorf("%s: asked where no tint can appear", name)
		}
	}
}

// A /play sitting borrows the editor's raw session and must not ask again.
func TestASittingDoesNotAskAgain(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	var tty syncBuf
	sess := &rawSession{control: &tty}
	con := newConsole(t.Context(), d, sess, &tty,
		func(w io.Writer, rows, cols int) *liveScreen { return newLiveScreen(w, rows, cols) }, true)
	keys := make(chan Key, 1)
	keys <- Key{Kind: KeyInterrupt}
	con.newSitting(t.Context(), d, opt, keys, &interrupter{}, io.Discard)
	if con.finish != nil {
		con.finish()
	}
	if n := strings.Count(tty.String(), backgroundQuery); n != 1 {
		t.Fatalf("the query went out %d times; the sitting must borrow, not ask again", n)
	}
}

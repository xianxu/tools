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
		})
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

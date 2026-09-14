package main

import (
	"context"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

func TestActivityPathsWaitDuringInference(t *testing.T) {
	for _, path := range []string{"ask", "reflect", "harvest", "agreement"} {
		t.Run(path, func(t *testing.T) {
			var d deps
			var f *llmtest.Fake
			switch path {
			case "ask":
				d, f, _, _ = askRig(t)
			case "reflect":
				d, f, _, _ = reflectRig(t, 14)
			default:
				var st *store.YAML
				d, f, st = harvestRig(t, 2)
				if path == "agreement" {
					if err := st.SetWordFacts(deckWord(0), store.WordFacts{Band: store.B1, Domain: store.DomainGeneral, At: harvestClock}); err != nil {
						t.Fatal(err)
					}
				}
			}
			started, release := make(chan struct{}), make(chan struct{})
			f.Script("", llmtest.Reply{Capture: streamCapture, Started: started, Release: release})
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var out, errs syncBuf
			done := make(chan struct{})
			go func() {
				defer close(done)
				switch path {
				case "ask":
					runAsk(ctx, d, options{tty: true}, &session{}, question{text: "why?", forced: true}, &out, &errs)
				case "reflect":
					runReflect(ctx, d, options{tty: true}, &out, &errs)
				case "harvest":
					runHarvest(ctx, d, options{tty: true}, harvestOptions{limit: 1}, &out, &errs)
				case "agreement":
					runHarvest(ctx, d, options{tty: true}, harvestOptions{agreement: 1}, &out, &errs)
				}
			}()
			awaitActivity(t, started)
			waiting := out.String()
			cancel()
			awaitActivity(t, done)
			if !strings.Contains(waiting, "⠋") {
				t.Fatalf("%s has no spinner while waiting: %q", path, waiting)
			}
			if !strings.Contains(out.String(), eraseLine) {
				t.Fatalf("%s did not clear spinner", path)
			}
		})
	}
}

func TestActivityPathsScannerKeepsLineBoundaries(t *testing.T) {
	d, f, _, _ := askRig(t)
	f.Script("", llmtest.Reply{Capture: streamCapture})
	var out, errs syncBuf
	code := replLines(t.Context(), &interrupter{}, d, options{tty: true, noAudio: true}, strings.NewReader("?why\n?and why\n"), &out, &errs, true, false)
	if code != 0 {
		t.Fatalf("scanner exit %d: %s", code, errs.String())
	}
	got := out.String()
	starts := 0
	for i, r := range got {
		if r == '⠋' {
			starts++
			if i != 0 && got[i-1] != '\n' {
				t.Fatalf("spinner began on a content line: %q", got)
			}
		}
	}
	if starts != 2 || strings.Count(got, "**Obsequious.**") != 2 {
		t.Fatalf("two complete answers required: %q", got)
	}
}

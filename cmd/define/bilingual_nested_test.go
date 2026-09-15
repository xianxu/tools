package main

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/play"
)

func TestBilingualNestedPracticeUsesCurrentSetting(t *testing.T) {
	d, opt, _ := playRig(t, "madrugar", "mesa", "bonito", "real", "once")
	d.lang = "es"
	source := &fixtureSupplement{Dictionary: testDictFor(t, "es")}
	d.dict = source
	// Use real command dispatch and editor mutation, then construct actual
	// questions at the console's sitting seam using the dependencies it receives.
	var terminal syncBuf
	var errout bytes.Buffer
	live := newPinnedScreen(&terminal, 40, 80)
	defer live.Stop()
	var states []bool
	con := console{view: live, stdout: live, stderr: &errout, finish: func() {}}
	con.newSitting = func(_ context.Context, sitting deps, sittingOpt options, _ <-chan Key, _ *interrupter, _ io.Writer) (int, winSize) {
		states = append(states, sitting.bilingualEnabled())
		before := source.calls
		qs, _ := questionsFor(t, sitting, sittingOpt)
		choices := 0
		for _, q := range qs {
			if _, ok := q.(*play.Choice); !ok {
				continue
			}
			choices++
			if strings.Contains(q.Reveal(), "English") != sitting.bilingualEnabled() {
				t.Fatalf("stale reveal in sitting %d: %q", len(states), q.Reveal())
			}
			if strings.Contains(q.Prompt(), "English") {
				t.Fatal("English leaked before answer")
			}
		}
		if choices == 0 {
			t.Fatal("nested sitting produced no choices")
		}
		wantCalls := 0
		if sitting.bilingualEnabled() {
			wantCalls = choices
		}
		if got := source.calls - before; got != wantCalls {
			t.Fatalf("sitting supplemental calls=%d, want %d", got, wantCalls)
		}
		return 0, winSize{rows: 40, cols: 80}
	}
	runEditor(t.Context(), keysFor("/play\r/bilingual off\r/play\r/bilingual on\r/play\r"), &interrupter{}, d, opt, con)
	if len(states) != 3 || !states[0] || states[1] || !states[2] {
		t.Fatalf("nested states=%v, want [true false true]; errors=%s", states, &errout)
	}
}

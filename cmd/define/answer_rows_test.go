package main

import (
	"bytes"
	"context"
	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
	"io"
	"strings"
	"testing"
)

func TestLanguageAnswerPhysicalRowsAreFinal(t *testing.T) {
	raw := "[lang=es]primero segundo tercero cuarto quinto sexto [/lang][lang=en]last[/lang]"
	for split := 0; split <= len(raw); split++ {
		var out bytes.Buffer
		a := newLanguageAnswer(&out, 20, nil, tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)})
		a.decoder.Write(raw[:split])
		a.decoder.Write(raw[split:])
		if err := a.Finish(); err != nil {
			t.Fatal(err)
		}
		rows := strings.Split(out.String(), "\n")
		if len(rows) != 3 {
			t.Fatalf("rows %q", rows)
		}
		for i := 0; i < 2; i++ {
			if !strings.Contains(rows[i], languageDark) {
				t.Fatalf("pure row %d unfilled: %q", i, rows)
			}
		}
		if strings.Contains(rows[2], languageDark) {
			t.Fatalf("mixed final row tinted at split %d: %q", split, rows[2])
		}
		if a.plain.String() != "primero segundo tercero cuarto quinto sexto last" {
			t.Fatalf("history %q", a.plain.String())
		}
	}
}

func TestLanguageAnswerBuffersPendingRow(t *testing.T) {
	var out bytes.Buffer
	a := newLanguageAnswer(&out, 20, nil, tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)})
	a.decoder.Write("[lang=es]hola [/lang]")
	if out.Len() != 0 {
		t.Fatalf("unfinished row escaped: %q", out.String())
	}
	a.decoder.Write("[lang=en]hello[/lang]")
	if err := a.Finish(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), languageDark) {
		t.Fatalf("mixed row tinted: %q", out.String())
	}
}

func TestAdvanceRowOwnership(t *testing.T) {
	states := []rowOwnership{{}, {lang: "es"}, {lang: "en"}, {mixed: true}}
	for i, s := range states {
		for _, decoration := range []rowOwnershipEvent{{}, {lang: "es"}} {
			if got := advanceRowOwnership(s, decoration); got != s {
				t.Fatalf("state %d decoration = %+v", i, got)
			}
		}
		if got := advanceRowOwnership(s, rowOwnershipEvent{finalize: true}); got != (rowOwnership{}) {
			t.Fatalf("state %d finalize = %+v", i, got)
		}
		for _, lang := range []string{"es", "en", ""} {
			want := rowOwnership{lang: lang}
			if s.mixed || lang == "" || s.lang != "" && s.lang != lang {
				want = rowOwnership{mixed: true}
			}
			if got := advanceRowOwnership(s, rowOwnershipEvent{lang: lang, substantive: true}); got != want {
				t.Fatalf("state %+v + %q = %+v want %+v", s, lang, got, want)
			}
		}
	}
}

type answerRowsSink struct {
	bytes.Buffer
	width int
	rows  []renderedOutput
}

func (s *answerRowsSink) OutputWidth() int { return s.width }
func (s *answerRowsSink) WriteOutput(o renderedOutput) error {
	s.rows = append(s.rows, o)
	return nil
}
func TestLanguageAnswerPendingResizeAndAtomicSinks(t *testing.T) {
	sink := &answerRowsSink{width: 30}
	a := newLanguageAnswer(sink, 30, nil, tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)})
	a.decoder.Write("[lang=es]primero segundo tercero cuarto [/lang]")
	if len(sink.rows) != 0 {
		t.Fatalf("provisional rows %+v", sink.rows)
	}
	sink.width = 20
	a.decoder.Write("[lang=en]last[/lang]")
	if err := a.Finish(); err != nil {
		t.Fatal(err)
	}
	if len(sink.rows) != 2 {
		t.Fatalf("rows %+v", sink.rows)
	}
	if sink.rows[0].text != "primero segundo\n" || !sink.rows[0].rows[0].tinted {
		t.Fatalf("first row %+v", sink.rows[0])
	}
	if sink.rows[1].text != "tercero cuarto last" || sink.rows[1].rows[0].tinted {
		t.Fatalf("mixed row %+v", sink.rows[1])
	}
	if a.plain.String() != "primero segundo tercero cuarto last" {
		t.Fatal(a.plain.String())
	}
}
func TestLanguageAnswerRowLimitAndFailureKeepHistory(t *testing.T) {
	for _, input := range []string{strings.Repeat(" ", maxAnswerWrapPending+1), strings.Repeat("x", maxAnswerWrapPending+1)} {
		var out bytes.Buffer
		a := newLanguageAnswer(&out, 20, nil, tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)})
		a.decoder.Write(input)
		if err := a.Finish(); err == nil {
			t.Fatal("missing pending limit")
		}
		if a.plain.String() != input {
			t.Fatal("history lost after row limit")
		}
		if out.Len() != 0 {
			t.Fatal("oversized unfinished row emitted")
		}
	}
	w := &languageFailWriter{}
	a := newLanguageAnswer(w, 20, nil, tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)})
	a.decoder.Write("[lang=es]primero segundo tercero cuarto quinto sexto [/lang]tail")
	if err := a.Finish(); err == nil {
		t.Fatal("missing write failure")
	}
	if a.plain.String() != "primero segundo tercero cuarto quinto sexto tail" || w.calls != 1 {
		t.Fatalf("poison failed: %q calls %d", a.plain.String(), w.calls)
	}
}

func TestLanguageAnswerCapturedRowsAllChunkBoundaries(t *testing.T) {
	// The wire fake and production parser provide the actual captured deltas;
	// split testing below exercises the unchanged decoder with that exact prose.
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: "stream-language.sse"})
	var captured strings.Builder
	d.newLLM = func(cfg llm.Config) llm.Client { return captureAnswerDeltas{Client: llm.New(cfg), raw: &captured} }
	d.scheme = holderFor(store.SchemeDark) // the shade both renderings are compared in
	var ordinary, stderr bytes.Buffer
	sess := &session{}
	if code := runAsk(t.Context(), d, options{color: true, width: 20, tintOn: true}, sess, question{text: "Explain buenos días"}, &ordinary, &stderr); code != 0 {
		t.Fatalf("capture replay: %d %s", code, &stderr)
	}
	raw := captured.String()
	for split := 0; split <= len(raw); split++ {
		sink := &answerRowsSink{width: 20}
		a := newLanguageAnswer(sink, 20, nil, tintPolicy{lang: "en", on: true, scheme: holderFor(store.SchemeDark)})
		a.decoder.Write(raw[:split])
		a.decoder.Write(raw[split:])
		if err := a.Finish(); err != nil {
			t.Fatal(err)
		}
		var rendered strings.Builder
		for _, row := range sink.rows {
			rendered.WriteString(serializeOutput(row, 20, store.SchemeDark))
		}
		if rendered.String()+"\n" != ordinary.String() {
			t.Fatalf("split %d live/append differ\nlive %q\nappend %q", split, rendered.String(), ordinary.String())
		}
		if a.plain.String() != sess.turns[0].Answer {
			t.Fatalf("split %d logical history differs", split)
		}
	}
}

type captureAnswerDeltas struct {
	llm.Client
	raw *strings.Builder
}

func (c captureAnswerDeltas) Stream(ctx context.Context, r llm.Request, delta func(string)) (llm.Response, error) {
	return c.Client.Stream(ctx, r, func(s string) { c.raw.WriteString(s); delta(s) })
}

func TestLanguageAnswerLiveScreenRowsMatchAppendSink(t *testing.T) {
	raw := "[lang=es]primero segundo tercero cuarto quinto sexto [/lang][lang=en]last[/lang]"
	var ordinary bytes.Buffer
	a := newLanguageAnswer(&ordinary, 20, nil, tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)})
	a.decoder.Write(raw)
	if err := a.Finish(); err != nil {
		t.Fatal(err)
	}
	live := newLiveScreen(io.Discard, 10, 20)
	b := newLanguageAnswer(live, 20, nil, tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)})
	for _, r := range raw {
		b.decoder.Write(string(r))
	}
	if err := b.Finish(); err != nil {
		t.Fatal(err)
	}
	if got := live.s.paintedTranscript(20); got != ordinary.String()+"\n" {
		t.Fatalf("live %q append %q", got, ordinary.String())
	}
	for _, line := range live.s.lines {
		if strings.Contains(line, languageDark) || strings.HasSuffix(line, " ") {
			t.Fatalf("paint padding entered source: %q", line)
		}
	}
}

func TestLanguageAnswerTrailingSpacesRemainPhysicalRows(t *testing.T) {
	sink := &answerRowsSink{width: 20}
	a := newLanguageAnswer(sink, 20, nil, tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)})
	a.decoder.Write("[lang=es]hola" + strings.Repeat(" ", 38) + "\n[/lang]")
	if err := a.Finish(); err != nil {
		t.Fatal(err)
	}
	for _, row := range sink.rows {
		if visibleCells(strings.TrimSuffix(row.text, "\n")) > 20 {
			t.Fatalf("row wider than physical width: %q", row.text)
		}
	}
	if a.plain.String() != "hola"+strings.Repeat(" ", 38)+"\n" {
		t.Fatal("lost history spaces")
	}
}

func TestLanguageAnswerDisplayUnitSurvivesChunkBoundary(t *testing.T) {
	// A flag may arrive as separate regional indicators. A forced word wrap
	// must never split that one display unit between physical rows.
	raw := strings.Repeat("x", 19) + "🇪🇸tail"
	var expected string
	for split := 0; split <= len(raw); split++ {
		sink := &answerRowsSink{width: 20}
		a := newLanguageAnswer(sink, 20, nil, tintPolicy{})
		a.decoder.Write(raw[:split])
		a.decoder.Write(raw[split:])
		if err := a.Finish(); err != nil {
			t.Fatal(err)
		}
		var text strings.Builder
		for _, row := range sink.rows {
			text.WriteString(row.text)
		}
		if split == 0 {
			expected = text.String()
		}
		if text.String() != expected || !strings.Contains(text.String(), "🇪🇸") {
			t.Fatalf("split %d broke display unit: %q", split, text.String())
		}
	}
}

func TestLanguageAnswerUnknownDecorationsDoNotDisqualifyOwnedRows(t *testing.T) {
	for _, raw := range []string{
		"1. [lang=es]hola[/lang]!",
		"  • [lang=es]hola[/lang] — 2026",
		"[lang=es]hola[/lang] / [lang=es]mundo[/lang]",
	} {
		sink := &answerRowsSink{width: 40}
		a := newLanguageAnswer(sink, 40, nil, tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)})
		a.decoder.Write(raw)
		if err := a.Finish(); err != nil {
			t.Fatal(err)
		}
		if len(sink.rows) != 1 || !sink.rows[0].rows[0].tinted {
			t.Fatalf("decorations disqualified %q: %+v", raw, sink.rows)
		}
	}
}

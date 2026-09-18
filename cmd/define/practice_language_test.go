package main

import (
	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
	"strings"
	"testing"
)

func TestPracticePresentationTintAndEnglishNeutral(t *testing.T) {
	q := play.NewChoice("rojo", "\x1b[48;5;254mentry\x1b[49m", []play.Option{{Gloss: "color", Correct: true}})
	q.SetHelp([]string{"red color"})
	policy := tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)}
	got := renderPracticePresentation(q.PromptPresentation(), "es", "es", policy, nil, surfaceProse, "")
	if !strings.Contains(got, languageDark+inkOnDark+"rojo") { // the tint and its paired ink (#70)
		t.Fatalf("missing target tint: %q", got)
	}
	if !strings.Contains(got, "   red color") || strings.Contains(got, "\x1b[2m") {
		t.Fatalf("English help changed: %q", got)
	}
	if stripANSI(got) != q.Prompt() || visibleCells(got) != visibleCells(q.Prompt()) {
		t.Fatal("copy/geometry changed")
	}
	reveal := renderPracticePresentation(q.RevealPresentation(), "es", "es", policy, nil, surfaceProse, "")
	if !strings.Contains(reveal, "\x1b[48;5;254mentry\x1b[49m") {
		t.Fatal("embedded definition tint changed")
	}
	b := play.NewBoard([]play.Cell{{Word: "rojo", Gloss: "color"}}, 30, play.Palette{Yes: "\x1b[32m", Off: "\x1b[0m"})
	b.Grade('0')
	got = renderPracticePresentation(b.PromptPresentation(), "es", "es", policy, nil, surfaceProse, "")
	assertDictionaryTint(t, got, "rojo", false)
}

func TestPracticePresentationRealPromptAndFooter(t *testing.T) {
	opt := options{color: true, width: 80, tintOn: true}
	d := deps{lang: "es", dict: practiceSourceDictionary{source: "es"}, scheme: holderFor(store.SchemeDark)}
	c := play.NewCloze("rojo", "Es ___", "Es rojo", "", []play.Option{{Word: "rojo", Correct: true}})
	c.SetHelp("It is ___")
	var out strings.Builder
	writePrompt(&out, c, d, opt)
	assertDictionaryTint(t, out.String(), "Es", true)
	assertDictionaryTint(t, out.String(), "It", false)
	if trimPaintPadding(out.String()) != "\n"+c.Prompt()+"\n" {
		t.Fatal("prompt text changed")
	}
	b := play.NewBoardPanel([]play.Cell{{Word: "rojo", Gloss: "color", Help: "red color"}, {Word: "azul", Gloss: "otro"}}, 40, play.Palette{Yes: "\x1b[32m", Off: "\x1b[0m"}, 2)
	b.Grade('0')
	rows := paintedBoardFooterForTest(b, sittingFigures{}, palette{}, d, opt)
	assertDictionaryTint(t, rows[0], "azul", true)
	assertDictionaryTint(t, rows[0], "rojo", false)
	if strings.Contains(rows[len(rows)-2], languageDark) {
		t.Fatalf("English footer tinted %q", rows)
	}
	frame := newSelectionFrame(40, len(rows), func() []selectionRow {
		var r []selectionRow
		source := boardFooterOutput(b, sittingFigures{}, palette{}, d, opt)
		for i, line := range strings.Split(source.text, "\n") {
			r = append(r, selectionRow{styled: line, selectable: true, paint: paintAt(source.rows, i)})
		}
		return r
	}())
	copied, err := selectedText(frame, selectionPoint{0, 0}, selectionPoint{0, 39})
	if err != nil || copied != strings.TrimRight(stripANSI(rows[0]), " ") {
		t.Fatalf("board copy %q %v", copied, err)
	}
	selected := frame.highlightRow(0, selectionPoint{0, 4}, selectionPoint{0, 7})
	if !strings.Contains(selected, "\x1b[7m") || stripANSI(selected) != strings.TrimRight(stripANSI(rows[0]), " ") {
		t.Fatalf("selection %q", selected)
	}
	d.lang = "en"
	d.dict = practiceSourceDictionary{source: "en"}
	rows = paintedBoardFooterForTest(b, sittingFigures{}, palette{}, d, opt)
	assertDictionaryTint(t, rows[len(rows)-2], "red color", true)
}

func TestPracticePresentationVocabularyRetainsForeground(t *testing.T) {
	v := &memVocabulary{}
	v.Add("color")
	q := play.NewChoice("rojo", "", []play.Option{{Gloss: "color", Correct: true}})
	q.SetHelp([]string{"color"})
	got := renderPracticePresentation(q.PromptPresentation(), "es", "es", tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)}, v, surfaceProse, "rojo")
	if !strings.Contains(got, knownOn) {
		t.Fatalf("vocabulary foreground lost %q", got)
	}
	if strings.Count(got, knownOn) != 1 {
		t.Fatalf("English acquired deck foreground %q", got)
	}
}

func TestPracticeChromeOwnership(t *testing.T) {
	q := play.NewChoice("red", "", []play.Option{{Gloss: "color", Correct: true}})
	p := gradePromptPresentation(q)
	if p.Text != gradePrompt(q) {
		t.Fatal("keys layout differs")
	}
	got := renderPracticePresentation(p, "en", "en", tintPolicy{lang: "en", on: true, scheme: holderFor(store.SchemeDark)}, nil, surfaceProse, "")
	assertDictionaryTint(t, got, "1-1", true)
	assertDictionaryTint(t, got, "pick the definition", true)
	assertDictionaryTint(t, got, "Ctrl-C", true)
	assertDictionaryTint(t, got, "to stop", true)
	output := renderPracticeOutput(p, "en", "en", tintPolicy{lang: "en", on: true, scheme: holderFor(store.SchemeDark)}, nil, surfaceProse, "", 80)
	cells, _ := rowTestCells(t, serializeOutput(output, 80, store.SchemeDark), 80)
	for col, c := range cells {
		if c.bg != 236 {
			t.Fatalf("chrome column %d not filled", col)
		}
	}
	bar := sittingBarPresentation(sittingFigures{done: 2, total: 4, load: 8, budget: 10})
	got = renderPracticePresentation(bar, "en", "en", tintPolicy{lang: "en", on: true, scheme: holderFor(store.SchemeDark)}, nil, surfaceProse, "")
	assertDictionaryTint(t, got, "2", true)
	assertDictionaryTint(t, got, "reviews/day", true)
}

func TestPracticeSubsequentEnglishOutputAndAllAnswerMarks(t *testing.T) {
	opt := options{color: true, width: 80, tintOn: true}
	d := deps{lang: "es", dict: practiceSourceDictionary{source: "es"}}
	spanish := play.NewChoice("rojo", "", []play.Option{{Gloss: "color vivo", Correct: true}})
	spanish.SetHelp([]string{"bright color"})
	var transcript strings.Builder
	writePrompt(&transcript, spanish, d, opt)
	before := transcript.String()
	assertDictionaryTint(t, before, "bright color", false)
	d.lang = "en"
	d.dict = practiceSourceDictionary{source: "en"}
	english := play.NewChoice("red", "", []play.Option{{Gloss: "bright color", Correct: true}})
	writePrompt(&transcript, english, d, opt)
	if !strings.HasPrefix(transcript.String(), before) {
		t.Fatal("language switch rewrote previous output")
	}
	after := transcript.String()[len(before):]
	assertDictionaryTint(t, after, "red", true)
	assertDictionaryTint(t, after, "bright color", true)
	for _, lang := range []store.Lang{"es", "en"} {
		for toggles := 0; toggles < 3; toggles++ {
			b := play.NewBoard([]play.Cell{{Word: "red", Gloss: "color"}, {Word: "blue", Gloss: "other"}}, 40, boardPalette(opt))
			for i := 0; i < toggles; i++ {
				b.Toggle()
			}
			b.Grade('0')
			rows := paintedBoardFooterForTest(b, sittingFigures{}, newPalette(true), deps{lang: lang}, opt)
			markedPrefix, _, ok := strings.Cut(rows[0], "[1]")
			if !ok {
				t.Fatalf("lang=%s mark=%v language tint over answer: %q", lang, b.Marked(0), rows[0])
			}
			want := boardPalette(opt).For(b.Marked(0)) + "[0] red" + boardPalette(opt).Off
			expectedCells, _ := rowTestCells(t, want, 7)
			actualCells, _ := rowTestCells(t, rows[0], opt.width)
			for col, wantCell := range expectedCells {
				if actualCells[col].fg != wantCell.fg {
					t.Fatalf("answer cell %d foreground=%d want %d", col, actualCells[col].fg, wantCell.fg)
				}
				if actualCells[col].bg != wantCell.bg {
					t.Fatalf("answer key/word cell %d background=%d want %d", col, actualCells[col].bg, wantCell.bg)
				}
			}
			if !strings.Contains(markedPrefix, boardPalette(opt).For(b.Marked(0))) {
				t.Fatalf("answer emphasis lost: %q", markedPrefix)
			}
			assertDictionaryTint(t, rows[0], "red", false)
			assertDictionaryTint(t, rows[0], "blue", true)
		}
	}
}

// practiceSourceDictionary supplies provenance independently from the requested
// language, just as selected native dictionary metadata does.
type practiceSourceDictionary struct{ source store.Lang }

func (d practiceSourceDictionary) Lookup(string) (string, error)     { return "", ErrNoEntry }
func (d practiceSourceDictionary) primarySourceLanguage() store.Lang { return d.source }

func TestPracticeDictionaryGlossOwnership(t *testing.T) {
	for _, tc := range []struct {
		name     string
		source   store.Lang
		wantTint bool
	}{
		{"unknown fallback", "", false}, {"known foreign source", "en", false}, {"verified target source", "it", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := deps{lang: "it", dict: practiceSourceDictionary{source: tc.source}}
			opt := options{color: true, width: 80, tintOn: true}
			q := play.NewChoice("rosso", "", []play.Option{{Gloss: "a bright color", Correct: true}})
			var out strings.Builder
			writePrompt(&out, q, d, opt)
			assertDictionaryTint(t, out.String(), "a bright color", tc.wantTint)
			assertDictionaryTint(t, out.String(), "rosso", true)
			out.Reset()
			writePracticePresentation(&out, q.RevealPresentation(), nil, d, opt, surfaceProse, "", "")
			assertDictionaryTint(t, out.String(), "a bright color", tc.wantTint)
			b := play.NewBoard([]play.Cell{{Word: "rosso", Gloss: "a bright color"}}, 60, boardPalette(opt))
			b.Grade('0')
			rows := paintedBoardFooterForTest(b, sittingFigures{}, palette{}, d, opt)
			panel := rows[len(rows)-2]
			assertDictionaryTint(t, panel, "a bright color", tc.wantTint)
		})
	}
}

func TestPracticeDictionaryRoleResolvesToActualSource(t *testing.T) {
	q := play.NewChoice("rosso", "", []play.Option{{Gloss: "a bright color", Correct: true}})
	got := renderPracticePresentation(q.PromptPresentation(), "it", "en", tintPolicy{lang: "en", on: true, scheme: holderFor(store.SchemeDark)}, nil, surfaceProse, "")
	assertDictionaryTint(t, got, "rosso", false)
	assertDictionaryTint(t, got, "a bright color", true)
}

// Paint-only padding is absent from the producer text and clipboard model.
func trimPaintPadding(s string) string {
	rows := strings.Split(stripANSI(s), "\n")
	for i := range rows {
		rows[i] = strings.TrimRight(rows[i], " ")
	}
	return strings.Join(rows, "\n")
}

func paintedBoardFooterForTest(q play.Question, fig sittingFigures, pal palette, d deps, opt options) []string {
	return strings.Split(serializeOutput(boardFooterOutput(q, fig, pal, d, opt), opt.width, d.scheme.Scheme()), "\n")
}

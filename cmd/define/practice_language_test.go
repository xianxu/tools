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
	policy := tintPolicy{lang: "es", background: "\x1b[48;5;236m"}
	got := renderPracticePresentation(q.PromptPresentation(), "es", policy, nil, surfaceProse, "")
	if !strings.Contains(got, "\x1b[48;5;236mrojo") {
		t.Fatalf("missing target tint: %q", got)
	}
	if !strings.Contains(got, "   red color") || strings.Contains(got, "\x1b[2m") {
		t.Fatalf("English help changed: %q", got)
	}
	if stripANSI(got) != q.Prompt() || visibleCells(got) != visibleCells(q.Prompt()) {
		t.Fatal("copy/geometry changed")
	}
	reveal := renderPracticePresentation(q.RevealPresentation(), "es", policy, nil, surfaceProse, "")
	if !strings.Contains(reveal, "\x1b[48;5;254mentry\x1b[49m") {
		t.Fatal("embedded definition tint changed")
	}
	b := play.NewBoard([]play.Cell{{Word: "rojo", Gloss: "color"}}, 30, play.Palette{Yes: "\x1b[32m", Off: "\x1b[0m"})
	b.Grade('0')
	got = renderPracticePresentation(b.PromptPresentation(), "es", policy, nil, surfaceProse, "")
	if strings.Contains(strings.Split(got, "\n")[0], policy.background) {
		t.Fatalf("answer mark tinted: %q", got)
	}
}

func TestPracticePresentationRealPromptAndFooter(t *testing.T) {
	opt := options{color: true, tintBackground: languageDark}
	d := deps{lang: "es"}
	c := play.NewCloze("rojo", "Es ___", "Es rojo", "", []play.Option{{Word: "rojo", Correct: true}})
	c.SetHelp("It is ___")
	var out strings.Builder
	writePrompt(&out, c, d, opt)
	if !strings.Contains(out.String(), languageDark+"Es") || strings.Contains(out.String(), languageDark+"It") {
		t.Fatalf("prompt ownership %q", out.String())
	}
	if stripANSI(out.String()) != "\n"+c.Prompt()+"\n" {
		t.Fatal("prompt text changed")
	}
	b := play.NewBoardPanel([]play.Cell{{Word: "rojo", Gloss: "color", Help: "red color"}, {Word: "azul", Gloss: "otro"}}, 40, play.Palette{Yes: "\x1b[32m", Off: "\x1b[0m"}, 2)
	b.Grade('0')
	rows := boardFooter(b, sittingFigures{}, palette{}, d, opt)
	if !strings.Contains(rows[0], languageDark+"azul") || strings.Contains(rows[0], languageDark+"rojo") {
		t.Fatalf("board mark ownership %q", rows[0])
	}
	if strings.Contains(rows[len(rows)-2], languageDark) {
		t.Fatalf("English footer tinted %q", rows)
	}
	frame := newSelectionFrame(40, len(rows), func() []selectionRow {
		var r []selectionRow
		for _, s := range rows {
			r = append(r, selectionRow{styled: s, selectable: true})
		}
		return r
	}())
	copied, err := selectedText(frame, selectionPoint{0, 0}, selectionPoint{0, 39})
	if err != nil || copied != strings.TrimRight(stripANSI(rows[0]), " ") {
		t.Fatalf("board copy %q %v", copied, err)
	}
	selected := frame.highlightRow(0, selectionPoint{0, 4}, selectionPoint{0, 7})
	if !strings.Contains(selected, "\x1b[7m") || stripANSI(selected) != stripANSI(rows[0]) {
		t.Fatalf("selection %q", selected)
	}
	d.lang = "en"
	rows = boardFooter(b, sittingFigures{}, palette{}, d, opt)
	if !strings.Contains(rows[len(rows)-2], languageDark+"red color") {
		t.Fatalf("English-selected help lacks tint %q", rows)
	}
}

func TestPracticePresentationVocabularyRetainsForeground(t *testing.T) {
	v := &memVocabulary{}
	v.Add("color")
	q := play.NewChoice("rojo", "", []play.Option{{Gloss: "color", Correct: true}})
	q.SetHelp([]string{"color"})
	got := renderPracticePresentation(q.PromptPresentation(), "es", tintPolicy{"es", languageDark}, v, surfaceProse, "rojo")
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
	got := renderPracticePresentation(p, "en", tintPolicy{"en", languageDark}, nil, surfaceProse, "")
	if strings.Contains(got, languageDark+"1-") || !strings.Contains(got, languageDark+"pick the definition") || !strings.Contains(got, languageDark+"to stop") {
		t.Fatalf("keys ownership %q", got)
	}
	bar := sittingBarPresentation(sittingFigures{done: 2, total: 4, load: 8, budget: 10})
	got = renderPracticePresentation(bar, "en", tintPolicy{"en", languageDark}, nil, surfaceProse, "")
	if strings.Contains(got, languageDark+"2") || !strings.Contains(got, languageDark+"reviews/day") {
		t.Fatalf("counter ownership %q", got)
	}
}

func TestPracticeSubsequentEnglishOutputAndAllAnswerMarks(t *testing.T) {
	opt := options{color: true, tintBackground: languageDark}
	d := deps{lang: "es"}
	spanish := play.NewChoice("rojo", "", []play.Option{{Gloss: "color vivo", Correct: true}})
	spanish.SetHelp([]string{"bright color"})
	var transcript strings.Builder
	writePrompt(&transcript, spanish, d, opt)
	before := transcript.String()
	if strings.Contains(before, languageDark+"bright color") {
		t.Fatalf("Spanish sitting tinted English help %q", before)
	}
	d.lang = "en"
	english := play.NewChoice("red", "", []play.Option{{Gloss: "bright color", Correct: true}})
	writePrompt(&transcript, english, d, opt)
	if !strings.HasPrefix(transcript.String(), before) {
		t.Fatal("language switch rewrote previous output")
	}
	after := transcript.String()[len(before):]
	if !strings.Contains(after, languageDark+"red") || !strings.Contains(after, languageDark+"bright color") {
		t.Fatalf("English subsequent output %q", after)
	}
	for _, lang := range []store.Lang{"es", "en"} {
		for toggles := 0; toggles < 3; toggles++ {
			b := play.NewBoard([]play.Cell{{Word: "red", Gloss: "color"}, {Word: "blue", Gloss: "other"}}, 40, boardPalette(opt))
			for i := 0; i < toggles; i++ {
				b.Toggle()
			}
			b.Grade('0')
			rows := boardFooter(b, sittingFigures{}, newPalette(true), deps{lang: lang}, opt)
			markedPrefix, _, ok := strings.Cut(rows[0], "[1]")
			if !ok || strings.Contains(markedPrefix, languageDark) {
				t.Fatalf("lang=%s mark=%v language tint over answer: %q", lang, b.Marked(0), rows[0])
			}
			want := boardPalette(opt).For(b.Marked(0)) + "[0] red" + boardPalette(opt).Off
			if !strings.Contains(markedPrefix, want) {
				t.Fatalf("answer style lost: got %q want %q", markedPrefix, want)
			}
			if !strings.Contains(rows[0], languageDark+"blue") {
				t.Fatalf("neighbor lost tint: %q", rows[0])
			}
		}
	}
}

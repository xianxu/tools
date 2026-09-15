package play

import (
	"reflect"
	"testing"
)

func TestPresentationOwnsLiteralPromptAndReveal(t *testing.T) {
	c := NewChoice("rojo", "entry", []Option{{Gloss: "color vivo", Correct: true}, {Gloss: "otro", Word: "otro"}})
	c.SetHelp([]string{"bright color", "other"})
	p := c.PromptPresentation()
	if p.Text != c.Prompt() {
		t.Fatal("prompt differs")
	}
	var got []string
	var roles []LanguageRole
	for _, s := range p.Spans {
		if s.Role != Neutral {
			got = append(got, p.Text[s.Start:s.End])
			roles = append(roles, s.Role)
		}
	}
	if want := []string{"rojo", "color vivo", "   bright color", "otro", "   other"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("owned text %q want %q", got, want)
	}
	if !reflect.DeepEqual(roles, []LanguageRole{Target, DictionarySource, English, DictionarySource, English}) {
		t.Fatalf("roles %v", roles)
	}
	c.Grade('2')
	r := c.RevealPresentation()
	if r.Text != "1  color vivo\n\nyou chose\n2  otro\n\nentry" {
		t.Fatalf("reveal %q", r.Text)
	}
	found := false
	for _, s := range r.Spans {
		if s.Role == English && r.Text[s.Start:s.End] == "you chose" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing English reveal label")
	}
}

func TestClozePresentationBlankNeutral(t *testing.T) {
	c := NewCloze("rojo", "Es ___ hoy", "Es rojo hoy", "", []Option{{Word: "rojo", Correct: true}})
	p := c.PromptPresentation()
	for _, s := range p.Spans {
		if s.Role == Target && p.Text[s.Start:s.End] == "Es ___ hoy" {
			t.Fatal("blank must be neutral")
		}
	}
	var owned string
	for _, s := range p.Spans {
		if s.Role == Target {
			owned += p.Text[s.Start:s.End]
		}
	}
	if owned != "Es  hoyrojo" {
		t.Fatalf("target %q", owned)
	}
}

func TestBoardPresentationTruncatedAndAnswerOwned(t *testing.T) {
	b := NewBoardPanel([]Cell{{Word: "larguísimo", Gloss: "color vivo", Help: "bright color"}}, 9, Palette{Yes: "\x1b[32m", Off: "\x1b[0m"}, 2)
	p := b.PromptPresentation()
	if p.Text != b.Prompt() {
		t.Fatal("layout drift")
	}
	if len(p.Spans) == 0 || p.Text[p.Spans[0].Start:p.Spans[0].End] != "larg~" {
		t.Fatalf("truncated ownership %#v %q", p.Spans, p.Text)
	}
	b.Grade('0')
	p = b.PromptPresentation()
	if !p.Spans[0].AnswerStyled {
		t.Fatal("marked word must suppress tint")
	}
	found := false
	for _, s := range p.Spans {
		if s.Role == English {
			found = true
		}
	}
	if !found {
		t.Fatal("board help lacks English ownership")
	}
}

func TestDictionaryGlossesDoNotClaimDeckOwnership(t *testing.T) {
	c := NewChoice("rosso", "", []Option{{Gloss: "a bright color", Correct: true}})
	b := NewBoard([]Cell{{Word: "rosso", Gloss: "a bright color"}}, 60, Palette{})
	b.Grade('0')
	for _, p := range []Presentation{c.PromptPresentation(), c.RevealPresentation(), b.PromptPresentation()} {
		found := false
		for _, s := range p.Spans {
			if p.Text[s.Start:s.End] == "a bright color" {
				found = true
				if s.Role != DictionarySource {
					t.Fatalf("dictionary gloss lacks source ownership: %q", p.Text)
				}
			}
		}
		if !found {
			t.Fatalf("missing gloss ownership: %q", p.Text)
		}
	}
}

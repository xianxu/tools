package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpanishPracticeUsesDefinitionsInsteadOfGrammar(t *testing.T) {
	for _, tc := range []struct{ word, pos, prefix string }{
		{"bonito", "adjetivo", "Que tiene belleza o atractivo"},
		{"madrugar", "verbo intransitivo", "Levantarse muy temprano"},
		{"mesa", "nombre femenino", "Mueble formado por un tablero"},
		{"once", "numeral cardinal", "Indica que el nombre"},
		{"real", "adjetivo", "Que tiene existencia verdadera"},
	} {
		t.Run(tc.word, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "entries", "es", tc.word+".txt"))
			if err != nil {
				t.Fatal(err)
			}
			e := ParseEntry(string(raw))
			if len(e.Blocks) == 0 || e.Blocks[0].POS != tc.pos {
				t.Errorf("first block must identify %q, got %+v", tc.pos, e.Blocks)
			}
			c, ok := targetCandidate(tc.word, e)
			if !ok || !strings.HasPrefix(c.Gloss, tc.prefix) {
				t.Errorf("target = %q, usable=%v; want semantic prefix %q", c.Gloss, ok, tc.prefix)
			}
			pool := optionCandidates(tc.word, e)
			if len(pool) == 0 || !strings.HasPrefix(pool[0].Gloss, tc.prefix) {
				t.Errorf("distractor candidates = %+v; want semantic prefix %q", pool, tc.prefix)
			}
			gloss, _ := senseFacts(tc.word, e)
			if !strings.HasPrefix(gloss, tc.prefix) {
				t.Errorf("harvest sense = %q; want semantic prefix %q", gloss, tc.prefix)
			}
			for _, width := range []int{0, 24, 80} {
				rendered, _ := Render(e, RenderOpts{Width: width})
				if string(alnum(rendered)) != string(alnum(string(raw))) {
					t.Errorf("render at width %d changed dictionary content", width)
				}
			}
			if tc.word == "bonito" {
				if len(e.Blocks) != 2 || e.Blocks[1].POS != "nombre masculino" ||
					len(e.Blocks[1].Senses) == 0 || !strings.HasPrefix(e.Blocks[1].Senses[0].Gloss, "Pez marino") {
					t.Error("bonito must retain its distinct fish noun definition")
				}
			}
			if tc.word == "once" {
				if len(e.Blocks) != 4 || e.Blocks[1].POS != "numeral ordinal" ||
					e.Blocks[2].POS != "nombre masculino" || e.Blocks[3].POS != "nombre femenino" {
					t.Error("once must retain its ordinal and noun blocks")
				}
			}
		})
	}
}

func TestSpanishGrammarLabelsDoNotBecomePracticeAnswers(t *testing.T) {
	for _, tc := range []struct{ word, pos, meaning string }{
		{"comprar", "verbo transitivo", "Adquirir una cosa a cambio de dinero"},
		{"arrepentirse", "verbo pronominal", "Sentir pesar por haber hecho algo"},
		{"ayer", "adverbio", "En el día anterior al de hoy"},
		{"ella", "pronombre", "Designa a la persona de la que se habla"},
		{"desde", "preposición", "Indica el punto de origen"},
		{"aunque", "conjunción", "Introduce una dificultad que no impide algo"},
		{"ay", "interjección", "Expresa dolor o sorpresa"},
	} {
		t.Run(tc.word, func(t *testing.T) {
			raw := tc.word + " " + tc.pos + " " + tc.meaning
			e := ParseEntry(raw)
			c, ok := targetCandidate(tc.word, e)
			if !ok || c.Gloss != tc.meaning {
				t.Errorf("target = %q, usable=%v; want %q", c.Gloss, ok, tc.meaning)
			}
			rendered, _ := Render(e, RenderOpts{})
			if string(alnum(rendered)) != string(alnum(raw)) {
				t.Error("grammar classification changed source content")
			}
		})
	}
}

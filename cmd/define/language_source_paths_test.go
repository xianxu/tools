package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
)

// This fixture declares its captured source, independently of the study language.
// The unwrapped fixture models an unknown every-active-dictionary result.
type tintSourceFixture struct {
	Dictionary
	language store.Lang
}

func (d tintSourceFixture) primarySourceLanguage() store.Lang { return d.language }

func TestLanguageTintSourceAcrossLookupAndReveals(t *testing.T) {
	for _, source := range []store.Lang{"", "en"} {
		for _, target := range []store.Lang{"it", "en"} {
			for _, path := range []string{"lookup", "choice", "cloze"} {
				t.Run(string(source)+"-"+string(target)+"-"+path, func(t *testing.T) {
					d, opt, st := playRig(t, "sycophantic", "ephemeral", "sanguine", "gaslighting", "keel", "mesa", "quokka")
					if source != "" {
						d.dict = tintSourceFixture{d.dict, source}
					}
					// Both the lock and the non-English supplement wrapper are production
					// composition seams. Neither may turn a requested language into evidence.
					d.dict = untranslatedDefinitions{Dictionary: d.dict, language: "Italian"}
					build := lockedDictionaries(func(store.Lang, io.Writer) (Dictionary, string) { return d.dict, everyActiveDictionary })
					d.dict, _ = build(target, io.Discard)
					d.lang = target
					off := false
					d.bilingual = &off
					opt.tintBackground = languageDark
					want := source != "" && source == target
					assert := func(text string) {
						t.Helper()
						if !strings.Contains(stripANSI(text), "sycophantic") {
							t.Fatalf("missing definition: %q", text)
						}
						assertDictionaryTint(t, text, "sycophantically", want)
					}
					if path == "lookup" {
						var out bytes.Buffer
						r := lookupAndRender(d, opt, replCommand{word: "sycophantic", literal: true}, &out, io.Discard)
						if r.code != 0 {
							t.Fatal("lookup failed")
						}
						assert(out.String())
						return
					}
					if path == "cloze" {
						if err := st.SetItems("sycophantic", []store.Item{clozeItem()}); err != nil {
							t.Fatal(err)
						}
					}
					qs, _ := questionsFor(t, d, opt)
					for _, q := range qs {
						if q.Word() != "sycophantic" {
							continue
						}
						if path == "choice" {
							if _, ok := q.(*play.Choice); !ok {
								t.Fatalf("got %T, want Choice", q)
							}
						}
						if path == "cloze" {
							if _, ok := q.(*play.Cloze); !ok {
								t.Fatalf("got %T, want Cloze", q)
							}
						}
						p, ok := q.(practicePresenter)
						if !ok {
							t.Fatalf("question %T lacks presentation", q)
						}
						output := renderPracticeOutput(p.RevealPresentation(), target, source, opt.tintFor(target), nil, surfaceProse, q.Word(), 80)
						assert(serializeOutput(output, 80))
						return
					}
					t.Fatal("sycophantic question not built")
				})
			}
		}
	}
}

package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
)

func TestBilingualSessionSwitch(t *testing.T) {
	for _, raw := range []bool{false, true} {
		t.Run(bilingualState(raw), func(t *testing.T) {
			fake := &definitionFake{primary: "mesa nombre femenino mueble", entries: []string{"mesa feminine noun table"}}
			d := testDeps(t)
			d.dict = fake
			d.lang = "es"
			d.persistBilingual = nil
			d.newDict = func(l store.Lang, _ io.Writer) (Dictionary, string) {
				if l == "es" {
					return fake, "Spanish sources"
				}
				return testDict(t), "English"
			}
			d.persistLang = func(store.Lang) error { return nil }
			input := "mesa\n/bilingual off\nmesa\n/lang en\n/lang es\nmesa\n/bilingual on\nmesa\n"
			var out, errout bytes.Buffer
			opt := options{noAudio: true, width: 80, rows: 80, tty: raw, color: raw}
			if raw {
				var terminal syncBuf
				live := newPinnedScreen(&terminal, 80, 80)
				runEditor(t.Context(), keysFor(strings.ReplaceAll(input, "\n", "\r")), &interrupter{}, d, opt, console{view: live, stdout: live, stderr: &errout, finish: live.Stop})
			} else {
				replLines(t.Context(), &interrupter{}, d, opt, strings.NewReader(input), &out, &errout, false, true)
			}
			if fake.primaryCalls != 4 || fake.supplementCalls != 2 {
				t.Fatalf("raw=%v primary=%d supplemental=%d errors=%s", raw, fake.primaryCalls, fake.supplementCalls, &errout)
			}
		})
	}
}

func TestBilingualPracticeReveal(t *testing.T) {
	var firstPrompts []string
	for _, on := range []bool{false, true} {
		d, opt, _ := playRig(t, "madrugar", "mesa", "bonito", "real", "once")
		primary := testDictFor(t, "es")
		supplement := &fixtureSupplement{Dictionary: primary}
		d.dict = supplement
		d.lang = "es"
		d.bilingual = &on
		qs, held := questionsFor(t, d, opt)
		choices := 0
		var prompts []string
		for _, q := range qs {
			if _, ok := q.(*play.Choice); !ok {
				continue
			}
			choices++
			prompts = append(prompts, q.Prompt())
			if strings.Contains(q.Reveal(), "English") != on {
				t.Fatalf("on=%v reveal=%q", on, q.Reveal())
			}
			if strings.Contains(q.Prompt(), "English") {
				t.Fatal("English leaked into question")
			}
			if len(held.marks[q.Word()].regions) == 0 {
				t.Fatal("missing reveal regions")
			}
		}
		if choices == 0 {
			t.Fatal("no actual choice constructed")
		}
		if !on {
			firstPrompts = prompts
		} else if strings.Join(prompts, "\n") != strings.Join(firstPrompts, "\n") {
			t.Fatal("toggle changed questions")
		}
		if on && supplement.calls != choices || !on && supplement.calls != 0 {
			t.Fatalf("calls=%d choices=%d on=%v", supplement.calls, choices, on)
		}
	}
}

type fixtureSupplement struct {
	Dictionary
	calls int
}

func (d *fixtureSupplement) primaryLabel() string { return "Spanish — Larousse" }
func (d *fixtureSupplement) supplement(word, _ string) definitionSection {
	d.calls++
	return definitionSection{label: "English — Oxford Spanish–English", entries: []string{word + " noun an English explanation"}}
}

func TestBilingualScreenSelection(t *testing.T) {
	d := testDeps(t)
	d.dict = &definitionFake{primary: "mesa nombre femenino mueble", entries: []string{"mesa feminine noun table | a red table"}}
	v := &memVocabulary{}
	v.Add("red")
	v.Add("mueble")
	d.vocab = v
	var terminal syncBuf
	live := newPinnedScreen(&terminal, 30, 80)
	defer live.Stop()
	result := lookupAndRender(d, options{color: true, width: 80}, replCommand{word: "mesa", literal: true}, live, io.Discard)
	if result.code != 0 {
		t.Fatal("lookup failed")
	}
	live.Draw("› ", nil)
	a, ok := selectionFind(live, "red table")
	if !ok {
		t.Fatal("English absent from screen")
	}
	if r, found := live.RegionAtRow(a.row, a.col); found && r.Kind == RegionWord {
		t.Fatalf("English red became a Spanish deck action: %+v", r)
	}
	b := a
	b.col += 8
	clipboard := newMemoryClipboard()
	router := newPointerRouter(live, clipboard)
	defer router.Stop()
	for wire := []byte(selectionWire(a, b)); len(wire) > 0; {
		k, n := decodeKey(wire)
		if n == 0 {
			t.Fatal("incomplete mouse sequence")
		}
		router.route(k)
		wire = wire[n:]
	}
	if got := clipboardAwait(t, clipboard.started); got != "red table" {
		t.Fatalf("copied %q", got)
	}
	clipboard.release <- nil
}

func TestBilingualClozeReveal(t *testing.T) {
	for _, on := range []bool{false, true} {
		d, opt, st := playRig(t, "madrugar")
		d.lang = "es"
		d.bilingual = &on
		d.dict = &fixtureSupplement{Dictionary: testDictFor(t, "es")}
		it := clozeItem()
		it.Answer = "madrugar"
		it.Stem = "Para llegar a la estación, Elena tiene que madrugar cada lunes."
		it.Distractors = []string{"dormir", "cantar", "correr"}
		if err := st.SetItems("madrugar", []store.Item{it}); err != nil {
			t.Fatal(err)
		}
		qs, _ := questionsFor(t, d, opt)
		if len(qs) != 1 || qs[0].Form() != "cloze" {
			t.Fatalf("no cloze: %+v", qs)
		}
		if strings.Contains(qs[0].Reveal(), "English") != on {
			t.Fatalf("on=%v reveal=%q", on, qs[0].Reveal())
		}
	}
}

func TestBilingualInflectedAudioAndCapture(t *testing.T) {
	v := voice{Lang: "es", Locale: "es"}
	url := AudioCandidates("madrugaste", v)[0]
	rig := newAudioRigServing(t, url)
	rig.deps.dict = &definitionFake{primary: "madrugar verbo intransitivo levantarse temprano", entries: []string{"madrugar verb to get up early"}}
	captured := &countingCapturer{}
	rig.deps.capture = captured
	opt := options{times: 1, voice: v}
	var out, errout bytes.Buffer
	result := defineOnce(t.Context(), rig.deps, opt, replCommand{word: "madrugaste", literal: true}, &out, &errout)
	if result.code != 0 || len(captured.calls) != 1 {
		t.Fatalf("result=%+v captures=%v err=%s", result, captured.calls, &errout)
	}
	if got := rig.cdn.Requested(); len(got) != 1 || got[0] != stripHost(t, url, audioBase) {
		t.Fatalf("wrong initial voice: %v", got)
	}
	replayInPlace(t.Context(), rig.deps, opt, session{current: "madrugaste", entry: result.entry}, "", &out, &errout)
	if len(rig.player.Played) != 2 || len(captured.calls) != 1 {
		t.Fatalf("replay changed capture or did not play: audio=%v capture=%v", rig.player.Played, captured.calls)
	}
}

package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
)

func TestLanguageAnswerFlushesOwnedAndPartialText(t *testing.T) {
	for _, tt := range []struct{ raw, plain, owned string }{
		{"English [lang=es]buenos días[/lang] good morning", "English buenos días good morning", "buenos días"},
		{"before [lang=es]unfinished", "before unfinished", ""},
		{"[lang=es]red[/lang] [lang=en]red[/lang]", "red red", "red"},
	} {
		var out bytes.Buffer
		a := newLanguageAnswer(&out, 0, nil, tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)})
		for _, r := range tt.raw {
			a.decoder.Write(string(r))
		}
		if err := a.Finish(); err != nil {
			t.Fatal(err)
		}
		if a.plain.String() != tt.plain || stripEscapes(out.String()) != tt.plain {
			t.Fatalf("text divergence: stored %q visible %q", a.plain.String(), out.String())
		}
		if strings.Contains(out.String(), languageDark) {
			t.Fatalf("width-zero plain output tinted: %q", out.String())
		}
	}
}

type languageFailWriter struct{ calls int }

func (w *languageFailWriter) Write(p []byte) (int, error) {
	w.calls++
	return 0, errors.New("write failed")
}
func TestLanguageAnswerWriteFailureStillFinishesCleanTranscript(t *testing.T) {
	w := &languageFailWriter{}
	a := newLanguageAnswer(w, 0, nil, tintPolicy{})
	a.decoder.Write("before [lang=es]unfinished")
	if a.Finish() == nil {
		t.Fatal("lost write error")
	}
	if a.plain.String() != "before unfinished" {
		t.Fatalf("partial transcript %q", a.plain.String())
	}
	if w.calls != 1 {
		t.Fatalf("poisoned writer called %d times", w.calls)
	}
}

func TestLanguageAnswerWrapClosesBackgroundBeforePhysicalNewline(t *testing.T) {
	var out bytes.Buffer
	a := newLanguageAnswer(&out, 20, nil, tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)})
	a.decoder.Write("[lang=es]primero segundo tercero cuarto quinto sexto[/lang]")
	if err := a.Finish(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\n") {
		t.Fatal("fixture did not wrap")
	}
	for _, line := range strings.Split(out.String(), "\n")[:strings.Count(out.String(), "\n")] {
		if !strings.HasSuffix(line, languageOff) && !strings.HasSuffix(line, sgrOff) {
			t.Fatalf("physical newline inherits background: %q", out.String())
		}
	}
}

func TestLanguageAnswerForeignHomographDoesNotUseTargetVocabulary(t *testing.T) {
	var out bytes.Buffer
	a := newLanguageAnswer(&out, 0, vocab("red"), tintPolicy{lang: "es", on: true, scheme: holderFor(store.SchemeDark)})
	a.decoder.Write("[lang=en]red[/lang] [lang=es]red[/lang]")
	if err := a.Finish(); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), knownOn) != 1 {
		t.Fatalf("only Spanish red is target vocabulary: %q", out.String())
	}
	if !strings.HasPrefix(out.String(), "red ") {
		t.Fatalf("English red acquired target styling: %q", out.String())
	}
}

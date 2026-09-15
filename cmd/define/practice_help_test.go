package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

const (
	testStem    = "Para llegar a la estación, Elena tiene que ___ cada lunes."
	testStemRaw = "Para llegar a la estación, Elena tiene que madrugar cada lunes."
)

func glossNeed(text string) helpNeed { return helpNeed{key: helpKey{"es", helpGloss, text}} }
func clozeNeed(text, answer string) helpNeed {
	return helpNeed{key: helpKey{"es", helpCloze, text}, answer: answer}
}

// Every row's expectation is a literal: the check is not asked what it thinks.
func TestPracticeHelpCheck(t *testing.T) {
	for _, tc := range []struct {
		name    string
		need    helpNeed
		english string
		want    string
		ok      bool
	}{
		{"folds whitespace", glossNeed("mueble"), "  a piece of\n\tfurniture  ", "a piece of furniture", true},
		{"empty", glossNeed("mueble"), "", "", false},
		{"blank only", glossNeed("mueble"), " \n\t ", "", false},
		{"escape sequence", glossNeed("mueble"), "table\x1b[31m", "", false},
		{"C1 control", glossNeed("mueble"), "table\u0085x\u009b", "", false},
		{"bidi override", glossNeed("mueble"), "table \u202edcba", "", false},
		{"invalid utf-8", glossNeed("mueble"), "table \xff", "", false},
		{"oversize", glossNeed("mueble"), strings.Repeat("a", maxHelpEnglish+1), "", false},
		{"gloss may carry underscores", glossNeed("mueble"), "a ___ thing", "a ___ thing", true},
		{"cloze keeps its blank", clozeNeed(testStem, "madrugar"), "To get to the station, Elena has to ___ every Monday.",
			"To get to the station, Elena has to ___ every Monday.", true},
		{"cloze fills the blank", clozeNeed(testStem, "madrugar"), "To get to the station, Elena has to get up early every Monday.", "", false},
		{"cloze adds a blank", clozeNeed(testStem, "madrugar"), "Elena has to ___ ___ every Monday.", "", false},
		{"cloze names the answer", clozeNeed(testStem, "madrugar"), "Elena has to ___ (madrugar) every Monday.", "", false},
		{"cloze names the answer capitalised", clozeNeed(testStem, "madrugar"), "Madrugar: Elena has to ___ every Monday.", "", false},
		{"answer inside another word is not a leak", clozeNeed("Vio el ___ del sol.", "set"), "She saw the ___ of the sunset.", "She saw the ___ of the sunset.", true},
		{"two blanks kept", clozeNeed("___ y ___", "sal"), "___ and ___", "___ and ___", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := checkHelp(tc.need, tc.english)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("checkHelp = %q, %v; want %q, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func FuzzPracticeHelpCheck(f *testing.F) {
	for _, s := range []string{"a table", "Elena has to ___ every Monday.", "___", "\x1b[2J", "\u202e", "madrugar ___", "\xff\xfe", strings.Repeat("_", 40)} {
		f.Add(s, testStem, "madrugar", true)
		f.Add(s, "mueble", "", false)
	}
	f.Fuzz(func(t *testing.T, english, source, answer string, cloze bool) {
		mode := helpGloss
		if cloze {
			mode = helpCloze
		}
		got, ok := checkHelp(helpNeed{key: helpKey{"es", mode, source}, answer: answer}, english)
		if !ok {
			if got != "" {
				t.Fatalf("a refusal returned text %q", got)
			}
			return
		}
		if got == "" || len(got) > maxHelpEnglish || !utf8.ValidString(got) {
			t.Fatalf("accepted text out of bounds: %q", got)
		}
		for _, r := range got {
			if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) {
				t.Fatalf("accepted text carries %U", r)
			}
		}
		if strings.Contains(got, "  ") || strings.TrimSpace(got) != got {
			t.Fatalf("accepted text is not folded: %q", got)
		}
		if cloze && strings.Count(got, "___") != strings.Count(source, "___") {
			t.Fatalf("accepted cloze help changed the blank count: %q for %q", got, source)
		}
	})
}

func testChoice() *play.Choice {
	return play.NewChoice("mesa", "mesa nombre femenino", []play.Option{
		{Gloss: "mueble con una tabla horizontal", Correct: true},
		{Gloss: "asiento con respaldo"},
		{Gloss: "recipiente para beber"},
		{Gloss: "lugar donde se duerme"},
	})
}

func testCloze() *play.Cloze {
	return play.NewCloze("madrugar", testStem, testStemRaw, "madrugar verbo", []play.Option{
		{Word: "dormir"}, {Word: "madrugar", Correct: true}, {Word: "cantar"}, {Word: "correr"},
	})
}

func testBoard() *play.Board {
	return play.NewBoardPanel([]play.Cell{
		{Word: "real", Gloss: "que existe de verdad"},
		{Word: "once", Gloss: ""},
		{Word: "bonito", Gloss: "que es agradable de ver"},
	}, 80, play.Palette{}, 2)
}

func TestPracticeHelpSources(t *testing.T) {
	needs := helpNeedsOf([]play.Question{testChoice(), testCloze(), testBoard()}, "es")
	want := [][]helpNeed{
		{
			{key: helpKey{"es", helpGloss, "mueble con una tabla horizontal"}, slot: 0},
			{key: helpKey{"es", helpGloss, "asiento con respaldo"}, slot: 1},
			{key: helpKey{"es", helpGloss, "recipiente para beber"}, slot: 2},
			{key: helpKey{"es", helpGloss, "lugar donde se duerme"}, slot: 3},
		},
		{{key: helpKey{"es", helpCloze, testStem}, answer: "madrugar"}},
		{
			{key: helpKey{"es", helpGloss, "que existe de verdad"}, slot: 0},
			{key: helpKey{"es", helpGloss, "que es agradable de ver"}, slot: 2},
		},
	}
	if len(needs) != len(want) {
		t.Fatalf("got %d question needs, want %d", len(needs), len(want))
	}
	for i := range want {
		if len(needs[i]) != len(want[i]) {
			t.Fatalf("question %d: got %+v, want %+v", i, needs[i], want[i])
		}
		for j := range want[i] {
			if needs[i][j] != want[i][j] {
				t.Fatalf("question %d need %d: got %+v, want %+v", i, j, needs[i][j], want[i][j])
			}
		}
	}
	// What leaves for the model is the keys' text: never the restored stem, never
	// the hidden word, never the Choice's headword.
	for _, k := range helpKeysOf(needs) {
		for _, secret := range []string{testStemRaw, "madrugar", "mesa"} {
			if strings.Contains(k.text, secret) {
				t.Fatalf("source %q carries %q", k.text, secret)
			}
		}
	}
}

func TestPracticeHelpApply(t *testing.T) {
	full := map[helpKey]string{
		{"es", helpGloss, "mueble con una tabla horizontal"}: "furniture with a flat top",
		{"es", helpGloss, "asiento con respaldo"}:            "a seat with a back",
		{"es", helpGloss, "recipiente para beber"}:           "a container to drink from",
		{"es", helpGloss, "lugar donde se duerme"}:           "a place where you sleep",
		{"es", helpCloze, testStem}:                          "To get to the station, Elena has to ___ every Monday.",
		{"es", helpGloss, "que existe de verdad"}:            "that truly exists",
	}
	t.Run("complete", func(t *testing.T) {
		c, z, b := testChoice(), testCloze(), testBoard()
		qs := []play.Question{c, z, b}
		if missed := applyHelp(qs, helpNeedsOf(qs, "es"), full); missed != 0 {
			t.Fatalf("missed = %d, want 0", missed)
		}
		for _, e := range []string{"furniture with a flat top", "a seat with a back", "a container to drink from", "a place where you sleep"} {
			if !strings.Contains(c.Prompt(), "\n   "+e) {
				t.Fatalf("choice prompt lacks %q:\n%s", e, c.Prompt())
			}
		}
		if !strings.Contains(z.Prompt(), testStem+"\nTo get to the station, Elena has to ___ every Monday.") {
			t.Fatalf("cloze prompt lacks its help:\n%s", z.Prompt())
		}
		if cells := b.Cells(); cells[0].Help != "that truly exists" || cells[1].Help != "" || cells[2].Help != "" {
			t.Fatalf("board help = %+v", cells)
		}
	})
	t.Run("a choice takes all or none", func(t *testing.T) {
		partial := map[helpKey]string{}
		for k, v := range full {
			if k.text != "lugar donde se duerme" {
				partial[k] = v
			}
		}
		c, control := testChoice(), testChoice()
		if missed := applyHelp([]play.Question{c}, helpNeedsOf([]play.Question{c}, "es"), partial); missed != 1 {
			t.Fatalf("missed = %d, want 1", missed)
		}
		if c.Prompt() != control.Prompt() {
			t.Fatalf("a partial set changed the prompt:\n%s", c.Prompt())
		}
	})
	t.Run("nothing accepted leaves every prompt as it was", func(t *testing.T) {
		c, z := testChoice(), testCloze()
		qs := []play.Question{c, z}
		if missed := applyHelp(qs, helpNeedsOf(qs, "es"), nil); missed != 2 {
			t.Fatalf("missed = %d, want 2", missed)
		}
		if c.Prompt() != testChoice().Prompt() || z.Prompt() != testCloze().Prompt() {
			t.Fatal("a prompt changed with no help")
		}
	})
}

func TestPracticeHelpAcceptsPerQuestion(t *testing.T) {
	// One text, two clozes with different answers: safe beside one, a leak
	// beside the other, so it is refused for both.
	stem := "Hoy ___ temprano."
	needs := [][]helpNeed{{clozeNeed(stem, "salgo")}, {clozeNeed(stem, "early")}}
	got := acceptHelp(needs, map[helpKey]string{{"es", helpCloze, stem}: "Today I ___ early."})
	if len(got) != 0 {
		t.Fatalf("accepted %v; a translation that names one question's answer serves neither", got)
	}
}

func TestPracticeHelpBatches(t *testing.T) {
	k := func(s string) helpKey { return helpKey{"es", helpGloss, s} }
	t.Run("duplicates and empties", func(t *testing.T) {
		batches, excluded := planHelpBatches([]helpKey{k("uno"), k(""), k("uno"), k("dos"), k("  ")})
		if len(excluded) != 0 || len(batches) != 1 || len(batches[0]) != 2 || batches[0][0] != k("uno") || batches[0][1] != k("dos") {
			t.Fatalf("batches %v excluded %v", batches, excluded)
		}
	})
	t.Run("seventeen texts make two calls", func(t *testing.T) {
		var keys []helpKey
		for i := 0; i < 17; i++ {
			keys = append(keys, k(strings.Repeat("a", i+1)))
		}
		batches, _ := planHelpBatches(keys)
		if len(batches) != 2 || len(batches[0]) != 16 || len(batches[1]) != 1 {
			t.Fatalf("batch sizes %d", len(batches))
		}
	})
	t.Run("an oversize text is left out whole", func(t *testing.T) {
		big := k(strings.Repeat("x", 4097))
		batches, excluded := planHelpBatches([]helpKey{k("uno"), big})
		if len(excluded) != 1 || excluded[0] != big || len(batches) != 1 || len(batches[0]) != 1 {
			t.Fatalf("batches %v excluded %d", batches, len(excluded))
		}
	})
	t.Run("bytes bound a call", func(t *testing.T) {
		var keys []helpKey
		for i := 0; i < 9; i++ {
			keys = append(keys, k(strings.Repeat(string(rune('a'+i)), 4000)))
		}
		batches, _ := planHelpBatches(keys)
		total := 0
		for _, b := range batches {
			size := 0
			for _, key := range b {
				size += len(key.text)
			}
			if size > 32<<10 {
				t.Fatalf("a batch carries %d bytes", size)
			}
			total += len(b)
		}
		if len(batches) != 2 || total != 9 {
			t.Fatalf("%d batches carrying %d texts", len(batches), total)
		}
	})
}

func TestHelpPromptGolden(t *testing.T) {
	llmtest.AssertGolden(t, "testdata", "help-prompt", renderHelpPrompt("es", []helpKey{
		{"es", helpGloss, "mueble con una tabla horizontal"},
		{"es", helpCloze, testStem},
	}))
}

func TestHelpPromptCarriesTheLanguage(t *testing.T) {
	batch := []helpKey{{"es", helpGloss, "asiento\ncon respaldo"}}
	es, it := renderHelpPrompt("es", batch), renderHelpPrompt("it", batch)
	if es.Prompt == it.Prompt || !strings.Contains(es.Prompt, "`es`") {
		t.Fatalf("the language did not reach the prompt:\n%s", es.Prompt)
	}
	if strings.Contains(es.System, "Spanish") {
		t.Fatal("the system prompt asserts a language")
	}
	if !strings.Contains(es.Prompt, "1. (a dictionary definition) asiento con respaldo\n") {
		t.Fatalf("a text is not one numbered line:\n%s", es.Prompt)
	}
	if es.Schema == nil {
		t.Fatal("no schema")
	}
}

func TestHelpReplyNumbers(t *testing.T) {
	batch := []helpKey{{"es", helpGloss, "uno"}, {"es", helpGloss, "dos"}, {"es", helpGloss, "tres"}}
	got := readHelpReply(batch, helpReply{Translations: []helpTranslation{
		{N: 1, English: "one"}, {N: 2, English: "two"}, {N: 2, English: "deux"}, {N: 4, English: "four"}, {N: 0, English: "zero"},
	}})
	if len(got) != 1 || got[batch[0]] != "one" {
		t.Fatalf("got %v; only the unambiguous in-range number maps", got)
	}
}

func TestHelpCause(t *testing.T) {
	if s := helpCause(context.DeadlineExceeded); !strings.Contains(s, "longer than 30s") {
		t.Fatalf("deadline: %q", s)
	}
	if s := helpCause(errors.Join(llm.ErrUnavailable, errors.New("dial"))); s != "the model is unavailable" {
		t.Fatalf("unavailable: %q", s)
	}
	if s := helpCause(errHelpRefused); s != errHelpRefused.Error() {
		t.Fatalf("refused: %q", s)
	}
}

func TestPracticeHelpCacheSeam(t *testing.T) {
	var reads, writes int
	var written []store.HelpEntry
	disk := []store.HelpEntry{{Lang: "es", Mode: "gloss", Source: "uno", English: "one"}}
	c := newPracticeHelpCache(
		func() []store.HelpEntry { reads++; return disk },
		func(e []store.HelpEntry) error { writes++; written = e; return nil })
	uno, dos := helpKey{"es", helpGloss, "uno"}, helpKey{"es", helpGloss, "dos"}
	if got := c.lookup([]helpKey{uno, dos}); len(got) != 1 || got[uno] != "one" {
		t.Fatalf("lookup = %v", got)
	}
	if err := c.remember(map[helpKey]string{dos: "two", uno: "one, again"}, aDay); err != nil {
		t.Fatal(err)
	}
	if got := c.lookup([]helpKey{uno, dos}); got[uno] != "one, again" || got[dos] != "two" {
		t.Fatalf("after remember: %v", got)
	}
	if reads != 1 || writes != 1 || len(written) != 2 {
		t.Fatalf("reads %d writes %d written %v", reads, writes, written)
	}
	var nilCache *practiceHelpCache
	if got := nilCache.lookup([]helpKey{uno}); len(got) != 0 || nilCache.remember(map[helpKey]string{uno: "x"}, aDay) != nil {
		t.Fatal("a nil cache must hold nothing and accept nothing quietly")
	}
	declined := newPracticeHelpCache(nil, func([]store.HelpEntry) error { return errDeckDeclined })
	if err := declined.remember(map[helpKey]string{uno: "one"}, aDay); !errors.Is(err, errDeckDeclined) {
		t.Fatalf("declined write = %v", err)
	}
	if got := declined.lookup([]helpKey{uno}); got[uno] != "one" {
		t.Fatal("a declined deck must still keep the translation for the session")
	}
}

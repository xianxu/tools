package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

var spanishDeck = []string{"madrugar", "mesa", "bonito", "real", "once"}

// helpReplyText answers any batch of up to 16 texts with numbered English that
// passes the gloss check. It cannot pass a cloze check: it has no blank.
func helpReplyText(prefix string) string {
	parts := make([]string, 0, 16)
	for i := 1; i <= 16; i++ {
		parts = append(parts, fmt.Sprintf(`{"n":%d,"english":"%s %d"}`, i, prefix, i))
	}
	return `{"translations":[` + strings.Join(parts, ",") + `]}`
}

// helpEnv points the model seam at the fake and turns #54's background job off,
// so every request the fake records was made by the sitting under test.
func helpEnv(url string) func(string) string {
	base := envFor(url)
	return func(k string) string {
		if k == noBackgroundEnv {
			return "1"
		}
		return base(k)
	}
}

// spanishSitting is a Spanish deck with bilingual display at its default (on)
// and the model seam pointed at a stateful fake.
func spanishSitting(t *testing.T, words ...string) (deps, options, *store.Mem, *llmtest.Fake) {
	t.Helper()
	d, opt, st := playRig(t, words...)
	d.dict = testDictFor(t, "es")
	d.lang = "es"
	fake := llmtest.NewFake(t)
	d.getenv = helpEnv(fake.URL)
	d.newLLM = llm.New
	d.practiceHelp = newPracticeHelpCache(nil, nil)
	return d, opt, st, fake
}

func helpRequests(f *llmtest.Fake) []llmtest.Recorded {
	var out []llmtest.Recorded
	for _, r := range f.Requests() {
		if strings.Contains(r.Prompt(), "## The texts") {
			out = append(out, r)
		}
	}
	return out
}

func refusingModel(d *deps, who string) {
	d.getenv = func(string) string { panic(who + " resolved the model configuration") }
	d.newLLM = func(llm.Config) llm.Client { panic(who + " constructed a model client") }
}

// shownTexts is every distinct text a sitting would translate, read off the
// questions directly rather than through helpNeedsOf.
func shownTexts(qs []play.Question) map[string]bool {
	texts := map[string]bool{}
	for _, q := range qs {
		switch q := q.(type) {
		case *play.Choice:
			for _, o := range q.Options() {
				texts[o.Gloss] = true
			}
		case *play.Board:
			for _, c := range q.Cells() {
				if strings.TrimSpace(c.Gloss) != "" {
					texts[c.Gloss] = true
				}
			}
		case *play.Cloze:
			texts[q.Blanked()] = true
		}
	}
	return texts
}

// PQ-4, clause 3: a cold cache reaches the model once per batch, inside the
// queue build, before the first question; the session itself never does.
func TestColdCachePreparesOnceBeforeTheFirstQuestion(t *testing.T) {
	d, opt, st, fake := spanishSitting(t, spanishDeck...)
	fake.Script("## The texts", llmtest.Reply{Text: helpReplyText("english")})
	built := 0
	d.newLLM = func(c llm.Config) llm.Client { built++; return llm.New(c) }

	var out, errb bytes.Buffer
	qs, held, code := todaysQuestions(t.Context(), d, opt, &out, &errb)
	if code != 0 || len(qs) == 0 {
		t.Fatalf("todaysQuestions = %d with %d questions; stderr %s", code, len(qs), &errb)
	}
	if built != 1 {
		t.Fatalf("built %d clients, want exactly 1", built)
	}
	choices := 0
	for _, q := range qs {
		c, ok := q.(*play.Choice)
		if !ok {
			continue
		}
		choices++
		for _, o := range c.Options() {
			if o.Help == "" {
				t.Fatalf("option %q has no English", o.Gloss)
			}
		}
		if !strings.Contains(c.Prompt(), "\n   english ") {
			t.Fatalf("the prompt does not show its English:\n%s", c.Prompt())
		}
	}
	if choices == 0 {
		t.Fatal("no multiple choice was built, so this pin cannot fail")
	}
	if want, got := (len(shownTexts(qs))+15)/16, len(helpRequests(fake)); got != want {
		t.Fatalf("%d model calls for %d texts, want %d", got, len(shownTexts(qs)), want)
	}
	if strings.Contains(errb.String(), "no English help") {
		t.Fatalf("a complete preparation warned: %s", &errb)
	}

	refusingModel(&d, "the session")
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("\r"+gradeKey(t, qs[0], play.Correct)), playbackConsole(&out, &errb))
	if len(reviewEvents(t, st)) == 0 {
		t.Fatal("the sitting recorded nothing")
	}
}

// PQ-4, clause 2, across a restart: a later process over the same deck finds
// every translation on disk and builds no client.
func TestWarmCacheSittingConstructsNoClient(t *testing.T) {
	dir := t.TempDir()
	onDisk := func() *practiceHelpCache {
		return newPracticeHelpCache(
			func() []store.HelpEntry { return store.ReadPracticeHelp(dir) },
			func(e []store.HelpEntry) error { return store.WritePracticeHelp(dir, e) })
	}
	d, opt, _, fake := spanishSitting(t, spanishDeck...)
	fake.Script("## The texts", llmtest.Reply{Text: helpReplyText("english")})
	d.practiceHelp = onDisk()
	first, _, _ := todaysQuestions(t.Context(), d, opt, io.Discard, io.Discard)
	if len(helpRequests(fake)) == 0 {
		t.Fatal("the cold sitting asked nothing, so this pin cannot fail")
	}

	d.practiceHelp = onDisk()
	refusingModel(&d, "a warm sitting")
	var errb bytes.Buffer
	second, _, code := todaysQuestions(t.Context(), d, opt, io.Discard, &errb)
	if code != 0 || len(second) != len(first) {
		t.Fatalf("warm sitting: code %d, %d questions (cold had %d); stderr %s", code, len(second), len(first), &errb)
	}
	for i := range first {
		if second[i].Prompt() != first[i].Prompt() {
			t.Fatalf("question %d differs after a restart:\ncold:\n%s\nwarm:\n%s", i, first[i].Prompt(), second[i].Prompt())
		}
	}
}

// PQ-4, clause 1: off never reaches, whatever the deck's language.
func TestBilingualOffSittingNeverReachesForTheModel(t *testing.T) {
	d, opt, st, _ := spanishSitting(t, spanishDeck...)
	off := false
	d.bilingual = &off
	refusingModel(&d, "an off sitting")
	var out, errb bytes.Buffer
	qs, held, code := todaysQuestions(t.Context(), d, opt, &out, &errb)
	if code != 0 || len(qs) == 0 {
		t.Fatalf("todaysQuestions = %d with %d questions", code, len(qs))
	}
	for _, q := range qs {
		if h, ok := q.(interface{ HelpLines() []int }); ok && len(h.HelpLines()) != 0 {
			t.Fatalf("an off sitting shows English:\n%s", q.Prompt())
		}
		if b, ok := q.(*play.Board); ok && b.PanelRows() != 1 {
			t.Fatal("an off board reserved an English row")
		}
	}
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("\r"+gradeKey(t, qs[0], play.Correct)), playbackConsole(&out, &errb))
	if len(reviewEvents(t, st)) == 0 {
		t.Fatal("the sitting recorded nothing")
	}
}

// PQ-4, clause 4: nested /play prepares through the same boundary with the
// setting in force, and a second nested sitting reuses the first one's work.
func TestBilingualNestedPracticePreparesHelp(t *testing.T) {
	d, opt, _, fake := spanishSitting(t, spanishDeck...)
	fake.Script("## The texts", llmtest.Reply{Text: helpReplyText("english")})
	var terminal syncBuf
	var errout bytes.Buffer
	live := newPinnedScreen(&terminal, 40, 80)
	defer live.Stop()
	var helped []bool
	var calls []int
	con := console{view: live, stdout: live, stderr: &errout, finish: func() {}}
	con.newSitting = func(ctx context.Context, sitting deps, sittingOpt options, _ <-chan Key, _ *interrupter, _ io.Writer) (int, winSize) {
		qs, _, _ := todaysQuestions(ctx, sitting, sittingOpt, io.Discard, io.Discard)
		shown := false
		for _, q := range qs {
			if h, ok := q.(interface{ HelpLines() []int }); ok && len(h.HelpLines()) != 0 {
				shown = true
			}
		}
		helped = append(helped, shown)
		calls = append(calls, len(helpRequests(fake)))
		return 0, winSize{rows: 40, cols: 80}
	}
	runEditor(t.Context(), keysFor("/play\r/bilingual off\r/play\r/bilingual on\r/play\r"), &interrupter{}, d, opt, con)
	if len(helped) != 3 || !helped[0] || helped[1] || !helped[2] {
		t.Fatalf("help shown per sitting = %v, want [true false true]; errors %s", helped, &errout)
	}
	if calls[0] == 0 || calls[1] != calls[0] || calls[2] != calls[0] {
		t.Fatalf("model calls after each sitting = %v; the first asks, the rest reuse it", calls)
	}
}

// Hold the real HTTP request before its reply: the interrupt must cancel
// preparation, leave the editor alive, and restore its interrupt afterward.
func TestBilingualPreparationScopesTheInterrupt(t *testing.T) {
	d, opt, st, fake := spanishSitting(t, spanishDeck...)
	started, release := make(chan struct{}), make(chan struct{})
	fake.Script("## The texts", llmtest.Reply{Text: helpReplyText("english"), Started: started, Release: release})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	loopCtx, loopCancel := context.WithCancel(t.Context())
	defer loopCancel()
	interrupts := &interrupter{fn: loopCancel}
	var tty, errb syncBuf
	repl := newPinnedScreen(&tty, 40, 80)
	defer repl.Stop()
	done := make(chan struct{})
	go func() {
		defer close(done)
		sittingInPlace(ctx, d, opt, make(chan Key), interrupts, repl, nil, &tty, &errb)
	}()
	awaitActivity(t, started)
	consumed := interrupts.Fire()
	if !consumed {
		cancel() // Release the worker even when the scope regression is present.
	}
	awaitActivity(t, done)
	if !consumed || loopCtx.Err() != nil {
		t.Fatal("Ctrl-C during preparation reached the editor instead of the sitting")
	}
	if len(reviewEvents(t, st)) != 0 || errb.String() != "" {
		t.Fatalf("cancelled preparation recorded answers or warned: %q", errb.String())
	}
	if len(d.practiceHelp.entries) != 0 {
		t.Fatal("cancelled response was cached")
	}
	if interrupts.Fire() || loopCtx.Err() == nil {
		t.Fatal("the editor's interrupt was not restored after preparation")
	}
}

type countingItems struct {
	store.Store
	calls map[string]int
}

func (c *countingItems) Items(key string) ([]store.Item, error) {
	c.calls[key]++
	return c.Store.Items(key)
}

// The blanked sentence a cloze prints is the text English help translates,
// selected once, and the hidden word never leaves for the model.
func TestClozeSelectsItsItemOnce(t *testing.T) {
	d, opt, st, fake := spanishSitting(t, "madrugar")
	it := clozeItem()
	it.Answer, it.Stem, it.Distractors = "madrugar", testStemRaw, []string{"dormir", "cantar", "correr"}
	if err := st.SetItems("madrugar", []store.Item{it}); err != nil {
		t.Fatal(err)
	}
	counter := &countingItems{Store: st, calls: map[string]int{}}
	d.deck = counter
	fake.Script("## The texts", llmtest.Reply{Text: `{"translations":[{"n":1,"english":"To get to the station, Elena has to ___ every Monday."}]}`})

	qs, _ := questionsFor(t, d, opt)
	if len(qs) != 1 || qs[0].Form() != "cloze" {
		t.Fatalf("got %d questions, first form %q; want one cloze", len(qs), qs[0].Form())
	}
	if counter.calls["madrugar"] != 1 {
		t.Fatalf("items read %d times, want once", counter.calls["madrugar"])
	}
	lines := strings.Split(qs[0].Prompt(), "\n")
	if lines[0] != testStem || lines[1] != "To get to the station, Elena has to ___ every Monday." {
		t.Fatalf("prompt:\n%s", qs[0].Prompt())
	}
	reqs := helpRequests(fake)
	if len(reqs) != 1 || !strings.Contains(reqs[0].Prompt(), testStem) {
		t.Fatalf("requests %d; the blanked sentence must be what is sent", len(reqs))
	}
	for _, secret := range []string{testStemRaw, "madrugar"} {
		if strings.Contains(reqs[0].Prompt(), secret) || strings.Contains(reqs[0].System(), secret) {
			t.Fatalf("the request carries %q", secret)
		}
	}
}

// A bilingual board reserves its English row at selection and at draw alike.
func TestBilingualBoardRowsFit(t *testing.T) {
	words := []string{"real", "once", "bonito", "mesa"}
	d, opt, st, fake := spanishSitting(t, words...)
	for _, w := range words {
		atBox(t, st, w, boardBox+1)
	}
	fake.Script("## The texts", llmtest.Reply{Text: helpReplyText("english")})
	qs, _ := questionsFor(t, d, opt)
	var b *play.Board
	for _, q := range qs {
		if g, ok := q.(*play.Board); ok {
			b = g
		}
	}
	if b == nil {
		t.Fatal("no board was built, so this pin cannot fail")
	}
	if b.PanelRows() != 2 {
		t.Fatalf("a bilingual board has %d panel rows, want 2", b.PanelRows())
	}
	if control := play.NewBoard(b.Cells(), opt.width, play.Palette{}); b.Rows() != control.Rows()+1 {
		t.Fatalf("rows %d, want %d", b.Rows(), control.Rows()+1)
	}
	if !boardFitsIn(b, opt.rows, opt.width) {
		t.Fatal("the board drawn does not fit where it was selected")
	}
	helped := -1
	for i, c := range b.Cells() {
		if c.Help != "" {
			helped = i
		}
	}
	if helped < 0 {
		t.Fatal("no board cell got English")
	}
	b.Mark(helped)
	if lines := strings.Split(b.Prompt(), "\n"); !strings.Contains(lines[len(lines)-1], "english ") {
		t.Fatalf("the English row does not show the marked cell's help:\n%s", b.Prompt())
	}

	// Selection answers with the same budget: at the height where a one-row
	// panel just fits, a two-row panel does not.
	tight := 0
	for r := 1; r < 80 && tight == 0; r++ {
		if boardFits(words, options{width: 80, rows: r}, 1) {
			tight = r
		}
	}
	if tight == 0 || boardFits(words, options{width: 80, rows: tight}, 2) || !boardFits(words, options{width: 80, rows: tight + 1}, 2) {
		t.Fatalf("at %d rows the two budgets do not differ by exactly one row", tight)
	}
}

// English help is selectable text with no Spanish deck actions; the Spanish
// line beside it keeps them, which is what proves the check can fail.
func TestBilingualPromptHelpIsSelectable(t *testing.T) {
	d := testDeps(t)
	v := &memVocabulary{}
	v.Add("red")
	v.Add("mueble")
	d.vocab = v
	c := play.NewChoice("mesa", "", []play.Option{{Gloss: "mueble alto", Correct: true}, {Gloss: "silla baja"}})
	if !c.SetHelp([]string{"a tall red table", "a low chair"}) {
		t.Fatal("SetHelp refused a complete set")
	}
	var terminal syncBuf
	live := newPinnedScreen(&terminal, 30, 80)
	defer live.Stop()
	writePrompt(live, c, d, options{color: true, tty: true, width: 80})
	live.Draw("› ", nil)

	m, ok := selectionFind(live, "mueble")
	if !ok {
		t.Fatal("the Spanish option is not on screen")
	}
	if r, found := live.RegionAtRow(m.row, m.col); !found || r.Kind != RegionWord {
		t.Fatal("the Spanish deck word lost its action, so the English check below cannot fail")
	}
	a, ok := selectionFind(live, "red table")
	if !ok {
		t.Fatal("the English help is not on screen")
	}
	if r, found := live.RegionAtRow(a.row, a.col); found && r.Kind == RegionWord {
		t.Fatalf("English red became a Spanish deck action: %+v", r)
	}
	b := a
	b.col += len("red table") - 1
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

// helpDeps is a Spanish deps with no questions of its own, for driving
// prepareHelp directly against the fake.
func helpDeps(t *testing.T) (deps, *llmtest.Fake) {
	t.Helper()
	d := testDeps(t)
	d.deck = store.NewMem()
	d.lang = "es"
	d.clock = store.FixedClock(aDay)
	fake := llmtest.NewFake(t)
	d.getenv = helpEnv(fake.URL)
	d.newLLM = llm.New
	d.practiceHelp = newPracticeHelpCache(nil, nil)
	return d, fake
}

func TestPracticeHelpPreparation(t *testing.T) {
	warnings := func(s string) int { return strings.Count(s, "define: no English help") }

	t.Run("cold then warm", func(t *testing.T) {
		d, fake := helpDeps(t)
		fake.Script("## The texts", llmtest.Reply{Text: helpReplyText("english")})
		c := testChoice()
		var errb bytes.Buffer
		prepareHelp(t.Context(), d, options{}, []play.Question{c}, io.Discard, &errb)
		reqs := helpRequests(fake)
		if len(reqs) != 1 || len(c.HelpLines()) != 4 || errb.Len() != 0 {
			t.Fatalf("requests %d, help lines %v, stderr %q", len(reqs), c.HelpLines(), &errb)
		}
		for _, o := range c.Options() {
			if !strings.Contains(reqs[0].Prompt(), o.Gloss) {
				t.Fatalf("the request lacks %q", o.Gloss)
			}
		}
		if strings.Contains(reqs[0].Prompt(), "mesa") || strings.Contains(strings.ToLower(reqs[0].Prompt()), "correct") {
			t.Fatalf("the request carries the headword or the answer:\n%s", reqs[0].Prompt())
		}
		again := testChoice()
		prepareHelp(t.Context(), d, options{}, []play.Question{again}, io.Discard, &errb)
		if len(helpRequests(fake)) != 1 || len(again.HelpLines()) != 4 {
			t.Fatal("a cached translation was asked for again")
		}
	})

	for _, tc := range []struct {
		name   string
		reply  llmtest.Reply
		reason string
		cached int
	}{
		{"malformed", llmtest.Reply{Text: "not json"}, "", 0},
		{"partial", llmtest.Reply{Text: `{"translations":[{"n":1,"english":"furniture with a flat top"}]}`}, "failed their checks", 1},
		{"unavailable", llmtest.Reply{Status: 503}, "the model is unavailable", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, fake := helpDeps(t)
			fake.Script("## The texts", tc.reply)
			c := testChoice()
			var errb bytes.Buffer
			prepareHelp(t.Context(), d, options{}, []play.Question{c}, io.Discard, &errb)
			if len(c.HelpLines()) != 0 {
				t.Fatal("a choice got partial or refused English")
			}
			if warnings(errb.String()) != 1 || !strings.Contains(errb.String(), tc.reason) {
				t.Fatalf("stderr %q; want one warning naming %q", &errb, tc.reason)
			}
			if got := d.practiceHelp.lookup(helpKeysOf(helpNeedsOf([]play.Question{testChoice()}, "es"))); len(got) != tc.cached {
				t.Fatalf("cached %d translations, want %d", len(got), tc.cached)
			}
		})
	}

	t.Run("a cloze reply naming the answer is refused and not cached", func(t *testing.T) {
		d, fake := helpDeps(t)
		fake.Script("## The texts", llmtest.Reply{Text: `{"translations":[{"n":1,"english":"Elena has to ___ (madrugar) every Monday."}]}`})
		z := testCloze()
		var errb bytes.Buffer
		prepareHelp(t.Context(), d, options{}, []play.Question{z}, io.Discard, &errb)
		if len(z.HelpLines()) != 0 || warnings(errb.String()) != 1 {
			t.Fatalf("help %v, stderr %q", z.HelpLines(), &errb)
		}
		if got := d.practiceHelp.lookup([]helpKey{{"es", helpCloze, testStem}}); len(got) != 0 {
			t.Fatal("a refused translation was cached")
		}
	})

	t.Run("interrupted before it could ask", func(t *testing.T) {
		d, fake := helpDeps(t)
		fake.Script("## The texts", llmtest.Reply{Text: helpReplyText("english")})
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		c := testChoice()
		var errb bytes.Buffer
		prepareHelp(ctx, d, options{}, []play.Question{c}, io.Discard, &errb)
		if len(c.HelpLines()) != 0 || errb.Len() != 0 {
			t.Fatalf("an interrupted preparation showed help %v or warned %q", c.HelpLines(), &errb)
		}
	})

	for _, tc := range []struct {
		name  string
		write error
		warn  bool
	}{
		{"declined deck keeps it for the session", errDeckDeclined, false},
		{"a failed save is said once", errors.New("disk full"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, fake := helpDeps(t)
			fake.Script("## The texts", llmtest.Reply{Text: helpReplyText("english")})
			d.practiceHelp = newPracticeHelpCache(nil, func([]store.HelpEntry) error { return tc.write })
			c := testChoice()
			var errb bytes.Buffer
			prepareHelp(t.Context(), d, options{}, []play.Question{c}, io.Discard, &errb)
			if len(c.HelpLines()) != 4 {
				t.Fatal("a save failure cost the live translation")
			}
			if got := strings.Count(errb.String(), "could not save the English help"); got != map[bool]int{true: 1, false: 0}[tc.warn] {
				t.Fatalf("stderr %q", &errb)
			}
		})
	}

	t.Run("an English deck never asks", func(t *testing.T) {
		d, _ := helpDeps(t)
		d.lang = store.DefaultLang
		refusingModel(&d, "an English deck")
		c := testChoice()
		prepareHelp(t.Context(), d, options{}, []play.Question{c}, io.Discard, io.Discard)
		if len(c.HelpLines()) != 0 {
			t.Fatal("an English deck got English help")
		}
	})
}

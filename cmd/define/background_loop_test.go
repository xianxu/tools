package main

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

// bgLookupWords are ten deck words the dictionary fixtures define and scriptAll
// scripts the model for, so a session can look them up and a job can prepare them.
var bgLookupWords = []string{"sycophantic", "ephemeral", "quokka", "defenestrate", "gaslighting",
	"alewife", "bargainer", "parrot", "pulp", "concrete"}

// bgLoopRig is the editor rig with a real store, a real capturer and the model
// behind the harvest seam, so a session's lookups add the deck words a background
// job then prepares (#54). seeded words start in the deck unbanded.
func bgLoopRig(t *testing.T, seeded int) (deps, options, *llmtest.Fake, *store.YAML, func()) {
	t.Helper()
	rig, opt, finish := editorRig(t, "sycophantic", true)
	hd, fake, st := harvestRig(t, seeded)
	d := rig.deps
	d.deck, d.clock, d.newLLM, d.getenv = hd.deck, hd.clock, hd.newLLM, hd.getenv
	d.capture = newStoreCapturer(st, d.clock, nil, nil)
	return d, opt, fake, st, finish
}

// bgRunSession starts the editor on an open key channel and returns it with a
// function that ends the session (EOF) and waits for it.
func bgRunSession(t *testing.T, ctx context.Context, d deps, opt options, con console) (chan<- Key, func() int) {
	t.Helper()
	keys := make(chan Key, 512)
	done := make(chan int, 1)
	go func() { done <- runEditor(ctx, keys, nil, d, opt, con) }()
	return keys, func() int {
		close(keys)
		select {
		case code := <-done:
			return code
		case <-time.After(10 * time.Second):
			t.Fatal("the session did not end")
			return -1
		}
	}
}

// typeLine types s and presses Enter, as a person at the prompt would.
func typeLine(keys chan<- Key, s string) {
	for _, r := range s {
		keys <- Key{Kind: KeyRune, Rune: r}
	}
	keys <- Key{Kind: KeyEnter}
}

// bgDeckHolds reports whether the deck has every word. It is how a test knows a
// lookup finished: capture is the last thing a lookup does.
func bgDeckHolds(st store.Store, words ...string) bool {
	deck, err := st.Deck()
	if err != nil {
		return false
	}
	var have []string
	for _, w := range deck {
		have = append(have, w.Text)
	}
	for _, w := range words {
		if !slices.Contains(have, w) {
			return false
		}
	}
	return true
}

// bgStub is a model client the test controls: it answers the first answer calls
// with a band, then blocks each call until released or cancelled, saying so on
// waiting. The llmtest fake cannot hold a non-streamed reply, which the
// never-waits and quitting tests need.
type bgStub struct {
	answer  int
	calls   atomic.Int32
	waiting chan struct{}
	release chan struct{}
}

func newBgStub(answer int) *bgStub {
	return &bgStub{answer: answer, waiting: make(chan struct{}, 64), release: make(chan struct{})}
}

func (s *bgStub) Complete(ctx context.Context, _ llm.Request) (llm.Response, error) {
	if int(s.calls.Add(1)) <= s.answer {
		return llm.Response{Text: bandReply}, nil
	}
	s.waiting <- struct{}{}
	select {
	case <-s.release:
		return llm.Response{}, errors.New("released by the test")
	case <-ctx.Done():
		return llm.Response{}, ctx.Err()
	}
}

func (s *bgStub) Stream(ctx context.Context, r llm.Request, _ func(string)) (llm.Response, error) {
	return s.Complete(ctx, r)
}

// The Done-when, end to end: ten new words looked up in a session become cloze
// questions without a command.
func TestTheSessionPreparesPracticeAfterTenNewWords(t *testing.T) {
	d, opt, fake, st, finish := bgLoopRig(t, 0)
	scriptAll(fake, 4)
	out := &syncBuf{}
	keys, end := bgRunSession(t, t.Context(), d, opt, recordingConsole(out, out, finish))
	for _, w := range bgLookupWords {
		typeLine(keys, w)
	}
	waitFor(t, func() bool { return strings.Contains(out.String(), "new practice question") })
	end()
	var authored []string
	for _, w := range bgLookupWords {
		if items, _ := st.Items(w); len(items) > 0 {
			authored = append(authored, w)
		}
	}
	if len(authored) == 0 {
		t.Fatalf("the notice showed but no word holds items:\n%s", out.String())
	}
	dq := d
	dq.clock = store.FixedClock(harvestClock.AddDate(0, 0, 2))
	opt.count, opt.rows, opt.width = 20, 40, 100
	qs, _ := questionsFor(t, dq, opt)
	cloze := map[string]bool{}
	for _, q := range qs {
		if _, ok := q.(*play.Cloze); ok {
			cloze[q.Word()] = true
		}
	}
	for _, w := range authored {
		if !cloze[w] {
			t.Errorf("%s holds items, but today's sitting does not ask it as a cloze", w)
		}
	}
}

// While a job waits on the model, the session still answers: two more lookups
// finish with the job blocked mid-call.
func TestALookupNeverWaitsForTheBackgroundJob(t *testing.T) {
	d, opt, _, st, finish := bgLoopRig(t, 0)
	stub := newBgStub(0)
	d.newLLM = func(llm.Config) llm.Client { return stub }
	out := &syncBuf{}
	keys, end := bgRunSession(t, t.Context(), d, opt, recordingConsole(out, out, finish))
	for _, w := range bgLookupWords {
		typeLine(keys, w)
	}
	select {
	case <-stub.waiting:
	case <-time.After(10 * time.Second):
		t.Fatal("the job never reached the model")
	}
	typeLine(keys, "minute")
	typeLine(keys, "mesa")
	waitFor(t, func() bool { return bgDeckHolds(st, "minute", "mesa") })
	close(stub.release)
	end()
}

// Quitting cancels the job: the session ends promptly, and the word banded before
// the quit is on disk.
func TestQuittingCancelsTheJobAndKeepsWhatItWrote(t *testing.T) {
	d, opt, _, st, finish := bgLoopRig(t, 0)
	stub := newBgStub(1)
	d.newLLM = func(llm.Config) llm.Client { return stub }
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	out := &syncBuf{}
	keys := make(chan Key, 512)
	done := make(chan int, 1)
	go func() { done <- runEditor(ctx, keys, nil, d, opt, recordingConsole(out, out, finish)) }()
	for _, w := range bgLookupWords {
		typeLine(keys, w)
	}
	select {
	case <-stub.waiting:
	case <-time.After(10 * time.Second):
		t.Fatal("the job never reached its second model call")
	}
	began := time.Now()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the session did not end after its context was cancelled")
	}
	if took := time.Since(began); took > 3*time.Second {
		t.Errorf("quitting took %v; the job should end on cancel, well inside the stop's wait", took)
	}
	banded := 0
	for _, w := range bgLookupWords {
		if f, _ := st.WordFacts(w); f.Harvested() {
			banded++
		}
	}
	if banded != 1 {
		t.Errorf("%d word(s) banded; the one answered before the quit should be on disk, and only it", banded)
	}
}

// No job in a directory nobody agreed to make a deck, whether the question is
// still open or was answered no.
func TestNoJobWhereTheDeckWasNotAgreedTo(t *testing.T) {
	declined := newDeckPermission(func() bool { return false })
	declined.resolve()
	for name, perm := range map[string]*deckPermission{
		"undecided": newDeckPermission(func() bool { return true }),
		"declined":  declined,
	} {
		t.Run(name, func(t *testing.T) {
			d, opt, fake, _, finish := bgLoopRig(t, bgThreshold)
			scriptAll(fake, 4)
			d.deckPermission = perm
			assertNoJob(t, d, opt, fake, finish)
		})
	}
}

func TestTheOffSwitchStopsIt(t *testing.T) {
	d, opt, fake, _, finish := bgLoopRig(t, bgThreshold)
	scriptAll(fake, 4)
	base := d.getenv
	d.getenv = func(k string) string {
		if k == noBackgroundEnv {
			return "1"
		}
		return base(k)
	}
	assertNoJob(t, d, opt, fake, finish)
}

// assertNoJob runs a session over a deck that already needs work, so an enabled
// runner would start a job at once, and checks that no model client was built.
// end waits for the session, whose deferred stop waits for any job it started, and
// a job over this deck builds its client before its first call; so a zero here is
// an observed absence, not a timing guess.
func assertNoJob(t *testing.T, d deps, opt options, fake *llmtest.Fake, finish func()) {
	t.Helper()
	var built atomic.Int32
	base := d.newLLM
	d.newLLM = func(c llm.Config) llm.Client {
		built.Add(1)
		return base(c)
	}
	out := &syncBuf{}
	keys, end := bgRunSession(t, t.Context(), d, opt, recordingConsole(out, out, finish))
	typeLine(keys, "sycophantic")
	end()
	if n := built.Load(); n != 0 || len(fake.Requests()) != 0 {
		t.Errorf("a model client was built %d time(s) and %d call(s) made; the session must not prepare anything here",
			n, len(fake.Requests()))
	}
}

// No model is said once, and then the session stops asking.
func TestNoModelIsOneNoticeThenQuiet(t *testing.T) {
	d, opt, fake, st, finish := bgLoopRig(t, bgThreshold)
	fake.Script(markBand, llmtest.Reply{Status: 500, Text: "upstream is having a day"})
	out := &syncBuf{}
	keys, end := bgRunSession(t, t.Context(), d, opt, recordingConsole(out, out, finish))
	waitFor(t, func() bool { return strings.Contains(out.String(), "the model did not answer") })
	before := len(fake.Requests())
	for _, w := range bgLookupWords {
		typeLine(keys, w)
	}
	waitFor(t, func() bool { return bgDeckHolds(st, "concrete") }) // the last word typed, and not seeded
	end()
	if n := strings.Count(out.String(), "the model did not answer"); n != 1 {
		t.Errorf("the no-model notice showed %d times, want once", n)
	}
	if after := len(fake.Requests()); after != before {
		t.Errorf("%d more model call(s) after the session learned there is no model", after-before)
	}
}

// TestNothingIsWrittenWhileAPromptIsShown's rule, applied to a job's notice: it is
// written between prompts, never into one.
func TestABackgroundNoticeIsWrittenBetweenPrompts(t *testing.T) {
	d, opt, fake, _, finish := bgLoopRig(t, bgThreshold)
	scriptAll(fake, 4)
	view := paintInto(io.Discard)
	seen := &syncBuf{}
	var mu sync.Mutex
	var whenWritten []string
	w := writerFunc(func(p []byte) (int, error) {
		mu.Lock()
		whenWritten = append(whenWritten, view.livePrompt())
		mu.Unlock()
		return seen.Write(p)
	})
	_, end := bgRunSession(t, t.Context(), d, opt, console{view: view, finish: finish, stdout: w, stderr: w})
	waitFor(t, func() bool { return strings.Contains(seen.String(), "new practice question") })
	end()
	mu.Lock()
	defer mu.Unlock()
	for i, p := range whenWritten {
		if p != "" {
			t.Fatalf("write %d of %d landed with a prompt on the frame: %q", i+1, len(whenWritten), p)
		}
	}
}

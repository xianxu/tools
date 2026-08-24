package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

// askRig wires runAsk against the WIRE-LEVEL fake — an httptest server speaking
// the Anthropic protocol, not a stubbed Client. Placement decides what a test
// can see: a stubbed Client sits above a mis-serialised body, a dropped header,
// a retry that re-sends a consumed body, and an SSE frame the parser mishandles.
// The store is a real YAML one in a temp dir, so user-model.md is read through
// the production path rather than seeded into a field.
func askRig(t *testing.T) (deps, *llmtest.Fake, *store.YAML, string) {
	t.Helper()
	fake := llmtest.NewFake(t)
	dir := t.TempDir()
	st := store.NewYAML(dir, nil)
	d := testDeps(t)
	d.deck = st
	// The REAL capturer over the same store: since #16 the event log has one
	// write path and questions go through it, so a rig with a no-op capturer
	// would be testing a wiring production does not have.
	d.capture = newStoreCapturer(st, store.FixedClock(aDay), nil)
	d.clock = store.FixedClock(aDay)
	d.newLLM = llm.New
	d.getenv = envFor(fake.URL)
	return d, fake, st, dir
}

var aDay = time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)

// streamCapture is the one committed streaming capture. Its content is a real
// answer to a real vocabulary question, which is what makes it usable here.
const streamCapture = "stream-sample.sse"

func TestAskStreamsAnAnswerWithTheDirectoryAsContext(t *testing.T) {
	d, fake, st, dir := askRig(t)
	// A committed capture, not invented text: the fake refuses a literal on a
	// streaming request, and the rule behind that refusal is the point — you
	// cannot fake judgment, but you can freeze a real answer and pin our
	// handling of it. This capture happens to BE an answer about "obsequious".
	fake.Script("", llmtest.Reply{Capture: streamCapture})
	// The directory holds a deck and a learner model — the two things that make
	// the answer adapted rather than merely generated.
	if err := st.Upsert(store.Word{Text: "ephemeral", FirstSeen: aDay, LastSeen: aDay, Lookups: 1}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "user-model.md"),
		[]byte("## Level\nC1, reads judicial opinions.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Each context source gets a fixture ONLY IT can supply. The first version
	// asserted "sycophantic" for the session list — a string CurrentWord also
	// supplies — so deleting SessionValues entirely left it green (I7).
	sess := &session{
		current: "sycophantic",
		entry:   "sycophantic | adjective | fawning",
		words:   []string{"gaslighting"},
	}
	var out, errb bytes.Buffer
	code := runAsk(t.Context(), d, options{}, sess, question{text: "difference to obsequious?"}, &out, &errb)

	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, errb.String())
	}
	if !strings.Contains(out.String(), "Obsequious") {
		t.Errorf("the answer did not reach stdout: %q", out.String())
	}
	// The RECORDED REQUEST is the assertion: the context reached the model.
	reqs := fake.Requests()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want 1", len(reqs))
	}
	body := reqs[0].System() + "\n" + reqs[0].Prompt()
	// sycophantic → the current word; gaslighting → the session list; ephemeral
	// → the deck; the sentence → user-model.md. No two share a source.
	for _, want := range []string{"sycophantic", "gaslighting", "ephemeral", "reads judicial opinions", "difference to obsequious?"} {
		if !strings.Contains(body, want) {
			t.Errorf("the prompt is missing %q:\n%s", want, body)
		}
	}
}

// BR-22: "Recently in the deck" must carry the words the learner touched most
// RECENTLY. Deck() is newest-first, so taking the tail took the twelve OLDEST
// and labelled them recent.
//
// Asserted with words only the DECK can supply — no fixture shared with the
// current word or the session list, because a shared one makes the assertion
// pass for the wrong reason (I7).
func TestTheDeckSectionCarriesTheNewestWords(t *testing.T) {
	d, fake, st, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})
	// LastSeen ascending, so "deckword00" is the oldest and "deckword19" the
	// newest. maxContextWords is 12, so 00..07 must NOT appear.
	for i := range 20 {
		w := store.Word{
			Text:      fmt.Sprintf("deckword%02d", i),
			FirstSeen: aDay, LastSeen: aDay.Add(time.Duration(i) * time.Hour), Lookups: 1,
		}
		if err := st.Upsert(w); err != nil {
			t.Fatal(err)
		}
	}

	var out, errb bytes.Buffer
	runAsk(t.Context(), d, options{}, &session{}, question{text: "why"}, &out, &errb)

	prompt := fake.Requests()[0].Prompt()
	if !strings.Contains(prompt, "deckword19") {
		t.Errorf("the newest deck word is missing from the prompt:\n%s", prompt)
	}
	if strings.Contains(prompt, "deckword00") {
		t.Errorf("the OLDEST deck word reached the prompt — the cut took the wrong end:\n%s", prompt)
	}
	// And oldest-first within what was kept, so the section reads forwards.
	if i8, i19 := strings.Index(prompt, "deckword08"), strings.Index(prompt, "deckword19"); i8 < 0 || i19 < i8 {
		t.Errorf("the kept words are not in reading order (08 at %d, 19 at %d)", i8, i19)
	}
}

// BR-23: the one-shot unforced route has NO current word — the dictionary
// missed, so nothing was defined and the line itself is the question. Passing
// the line as the current word put question text into "## The word on screen"
// and into the event log's word field.
func TestOneShotQuestionHasNoCurrentWord(t *testing.T) {
	fake := llmtest.NewFake(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})
	dir := t.TempDir()
	st := store.NewYAML(dir, nil)
	d := testDeps(t)
	cap := newStoreCapturer(st, store.FixedClock(aDay), nil)
	d.newStore = func(options, io.Writer) storeDeps {
		return storeDeps{history: &memHistory{}, capture: cap, deck: st, clock: store.FixedClock(aDay)}
	}
	d.capture, d.history = nil, nil // force the real wiring, as capture_test.go does
	d.newLLM = llm.New
	d.getenv = envFor(fake.URL)

	var out, errb bytes.Buffer
	// Unforced and quoted: NOAD misses, the line reads as a question.
	run(t.Context(), []string{"-no-audio", "what is the difference to obsequious?"},
		d, strings.NewReader(""), &out, &errb)

	if len(fake.Requests()) != 1 {
		t.Fatalf("requests = %d, want 1: %q", len(fake.Requests()), errb.String())
	}
	if got := fake.Requests()[0].Prompt(); strings.Contains(got, headerCurrent) {
		t.Errorf("a question with no lookup behind it rendered a current-word section:\n%s", got)
	}
	ev, err := st.Events(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 1 {
		t.Fatalf("events = %+v, want exactly the asked event", ev)
	}
	if ev[0].Word != "" {
		t.Errorf("event word = %q, want empty — the question text was recorded as a word", ev[0].Word)
	}
	if ev[0].Question == "" {
		t.Error("the question was not recorded")
	}
}

func envFor(url string) func(string) string {
	return func(k string) string {
		switch k {
		case "DEFINE_LLM_BASE_URL":
			return url
		case "DEFINE_LLM_API_KEY":
			return "test-key"
		}
		return ""
	}
}

func TestAskRecordsAnAskedEvent(t *testing.T) {
	d, fake, st, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})

	sess := &session{current: "sycophantic"}
	var out, errb bytes.Buffer
	runAsk(t.Context(), d, options{}, sess, question{text: "is it pejorative?"}, &out, &errb)

	ev, err := st.Events(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 1 {
		t.Fatalf("events = %+v, want exactly the asked event", ev)
	}
	if ev[0].Kind != store.EventAsked || ev[0].Question != "is it pejorative?" || ev[0].Word != "sycophantic" {
		t.Errorf("event = %+v, want an asked event carrying the question and the word it followed", ev[0])
	}
}

func TestAskFollowUpCarriesThePreviousExchange(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture}, llmtest.Reply{Capture: streamCapture})

	sess := &session{current: "sycophantic"}
	var out, errb bytes.Buffer
	runAsk(t.Context(), d, options{}, sess, question{text: "is it pejorative?"}, &out, &errb)
	runAsk(t.Context(), d, options{}, sess, question{text: "give me three more examples"}, &out, &errb)

	reqs := fake.Requests()
	if len(reqs) != 2 {
		t.Fatalf("requests = %d, want 2", len(reqs))
	}
	second := reqs[1].Prompt()
	// The first QUESTION and a distinctive fragment of the first ANSWER: a
	// follow-up resolves against what was just said, so both halves must travel.
	for _, want := range []string{"is it pejorative?", "servile deference"} {
		if !strings.Contains(second, want) {
			t.Errorf("the follow-up prompt is missing %q:\n%s", want, second)
		}
	}
}

// The user's own Ctrl-C is not a failure to report.
//
// Asked of the CONTEXT, not the error: a cancelled request has no HTTP status,
// and mapError classifies statusless failures as ErrUnavailable — so a
// decode-the-error version prints "no model configured" as the answer to a
// keypress.
func TestAskSaysNothingWhenTheUserCancels(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture, Stall: true})

	ctx, cancel := context.WithCancel(t.Context())
	var out syncBuf
	var errb bytes.Buffer
	go func() {
		// Cancel once the first delta has landed, so this is mid-stream rather
		// than before the request. Polled through syncBuf, because reading a
		// bytes.Buffer another goroutine is writing is a data race — which is
		// what this test did until `go test -race` said so.
		for out.Len() == 0 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()
	code := runAsk(ctx, d, options{}, &session{}, question{text: "why?"}, &out, &errb)

	if errb.Len() != 0 {
		t.Errorf("a cancelled answer printed a diagnostic: %q", errb.String())
	}
	if code != 0 {
		t.Errorf("exit = %d, want 0 — the user's own keypress is not a failure", code)
	}
}

func TestAskDegradesWhenTheSeamIsUnavailable(t *testing.T) {
	d := testDeps(t)
	d.newLLM = llm.New
	d.getenv = func(string) string { return "" } // no key at all

	var out, errb bytes.Buffer
	code := runAsk(t.Context(), d, options{}, &session{}, question{text: "how so"}, &out, &errb)

	if code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
	if want := "define: no model configured; `how so` is not a word\n"; errb.String() != want {
		t.Errorf("stderr = %q, want %q", errb.String(), want)
	}
	// And a LOOKUP still works in the same session afterwards.
	var out2, errb2 bytes.Buffer
	if got := lookupAndRender(d, options{}, replCommand{kind: cmdDefine, word: "sycophantic"}, &out2, &errb2); got.code != 0 {
		t.Errorf("a lookup after an unavailable ask failed: %q", errb2.String())
	}
}

// ErrRequest is OUR bug — a bad prompt, a bad model — and must stay loud where
// ErrUnavailable degrades quietly. The taxonomy is the whole point of the seam.
func TestAskStaysLoudOnOurOwnBug(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Status: 400})

	var out, errb bytes.Buffer
	code := runAsk(t.Context(), d, options{}, &session{}, question{text: "why is it so"}, &out, &errb)

	if code == 0 {
		t.Error("a 400 exited 0")
	}
	if strings.Contains(errb.String(), "no model configured") {
		t.Errorf("our own bug was reported as an unavailable seam: %q", errb.String())
	}
}

// A nil seam is "no model wired", not a crash. llm.Resolve dereferences the
// getenv it is given, so an unwired deps used to panic inside the ask path —
// and every test that does not care about the model leaves it unset.
func TestAskWithNoSeamWiredDegrades(t *testing.T) {
	for _, tc := range []struct {
		name string
		d    deps
	}{
		{"nothing wired", testDeps(t)},
		{"getenv without a client", func() deps { d := testDeps(t); d.getenv = func(string) string { return "" }; return d }()},
		{"a client without getenv", func() deps { d := testDeps(t); d.newLLM = llm.New; return d }()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			if code := runAsk(t.Context(), tc.d, options{}, &session{}, question{text: "how so"}, &out, &errb); code != 1 {
				t.Errorf("exit = %d, want 1", code)
			}
			if !strings.Contains(errb.String(), "no model configured") {
				t.Errorf("stderr = %q", errb.String())
			}
		})
	}
}

// Ctrl-C mid-stream returns to the prompt with the session INTACT — the one
// place in this program where an interrupt means something narrower than quit.
//
// Two tests, not one, because the two transports are exactly what a design can
// silently serve only half of: the byte the raw reader decodes, and a SIGINT.
// Both must stop the answer and leave the loop running, and the observable that
// proves it is a LOOKUP AFTER the interrupt — a session that quit cannot answer.
func TestEditorCtrlCMidStreamReturnsToThePrompt(t *testing.T) {
	for _, tc := range []struct {
		name    string
		byteKey bool
	}{
		{"the byte transport", true},
		{"the signal transport", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, fake, _, _ := askRig(t)
			// Stall: frames up to the first text delta, then silence — the shape
			// a hung upstream has, and the case an interrupt exists for.
			fake.Script("", llmtest.Reply{Capture: streamCapture, Stall: true})
			d.stdinIsTerminal = func() bool { return true }
			opt := options{noAudio: true, locale: "us", tty: true, color: true}

			// Driven through the REAL reader, not a scripted channel: the byte
			// transport's swallow happens inside readKeys, so a test that feeds
			// the key channel directly cannot exercise it at all.
			interrupts := &interrupter{fn: func() {
				t.Error("the SESSION cancel fired: the interrupt was not scoped to the answer")
			}}
			// The signal row drives the real seam, so the watcher repl installs
			// is part of what this pins.
			sigs := make(chan os.Signal, 1)
			d.notifySignals = func(...os.Signal) <-chan os.Signal { return sigs }
			go func() {
				for range sigs {
					interrupts.Fire()
				}
			}()
			pr, pw := io.Pipe()
			t.Cleanup(func() { pw.Close() })
			keys := readKeys(t.Context(), pr, interrupts)

			var out syncBuf
			var errb syncBuf
			done := make(chan int, 1)
			go func() {
				done <- runEditor(t.Context(), keys, interrupts, d, opt,
					func(run func()) error { run(); return nil }, func() {}, &out, &errb)
			}()

			io.WriteString(pw, "?why\r")
			waitFor(t, func() bool { return strings.Contains(out.String(), "Obsequious") })

			// Interrupt it the way this transport ACTUALLY does. The earlier
			// version called interrupts.Fire() for the signal row, which is the
			// sink itself — so the row named a transport it never touched, and
			// deleting repl's signal watcher left it green (BR-30).
			if tc.byteKey {
				io.WriteString(pw, "\x03")
			} else {
				sigs <- os.Interrupt
			}

			// THE observable: a lookup after the interrupt still answers. A
			// session that quit cannot, and "exited cleanly" is produced by the
			// byte path, the signal path AND a crash alike.
			io.WriteString(pw, "sycophantic\r")
			waitFor(t, func() bool { return strings.Contains(out.String(), "sikəˈfan(t)ik") })
			pw.Close()

			select {
			case code := <-done:
				if code != 0 {
					t.Errorf("exit = %d, want 0", code)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("the loop never returned")
			}
			if strings.Contains(errb.String(), "define: llm") || strings.Contains(errb.String(), "no model") {
				t.Errorf("the interrupt was reported as a failure: %q", errb.String())
			}
		})
	}
}

// Both routes into the ask reach the SAME wiring, so the interrupt scoping and
// the CRLF writer exist once. A second copy would pass the tests above while
// diverging on everything they do not assert.
func TestForcedAndUnforcedAsksShareOneWiring(t *testing.T) {
	// The wiring the two routes share is the interrupt SCOPING as much as the
	// CRLF writer, so this asserts both — and with a real sink, because passing
	// nil could only ever pin the writer half (I7).
	for _, tc := range []struct{ name, keys string }{
		{"forced", "?why\r"},
		{"unforced", "what's the difference to obsequious?\r"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, fake, _, _ := askRig(t)
			fake.Script("", llmtest.Reply{Capture: streamCapture, Stall: true})
			d.stdinIsTerminal = func() bool { return true }
			opt := options{noAudio: true, locale: "us", tty: true}

			interrupts := &interrupter{fn: func() {
				t.Errorf("%s: the SESSION cancel fired — this route does not scope the interrupt", tc.name)
			}}
			pr, pw := io.Pipe()
			t.Cleanup(func() { pw.Close() })
			keys := readKeys(t.Context(), pr, interrupts)

			var out syncBuf
			var errb syncBuf
			done := make(chan int, 1)
			go func() {
				done <- runEditor(t.Context(), keys, interrupts, d, opt,
					func(run func()) error { run(); return nil }, func() {}, &out, &errb)
			}()

			io.WriteString(pw, tc.keys)
			waitFor(t, func() bool { return strings.Contains(out.String(), "Obsequious") })
			io.WriteString(pw, "\x03")

			// Both routes must survive it, and both must have written through
			// crlfWriter.
			io.WriteString(pw, "sycophantic\r")
			waitFor(t, func() bool { return strings.Contains(out.String(), "sikəˈfan(t)ik") })
			pw.Close()
			<-done

			assertCRLFTerminated(t, streamedAnswer(out.String()), tc.name+" answer")
		})
	}
}

// streamedAnswer isolates what the ASK wrote from the rest of the session.
//
// Scoping matters: a definition is rendered under cooked(), where a bare "\n" is
// correct and the terminal translates it. Only the streamed answer is written
// while the terminal is raw, so only it must carry its own carriage returns —
// asserting over the whole session would fail on output that is already right.
func streamedAnswer(s string) string {
	i := strings.Index(s, "**Obsequious.**")
	if i < 0 {
		return ""
	}
	rest := s[i:]
	if j := strings.Index(rest, eraseLine); j >= 0 {
		rest = rest[:j] // stops at the next prompt redraw
	}
	return rest
}

// assertCRLFTerminated asserts the POSITIVE observable: every line break in
// raw-mode output is "\r\n".
//
// Its predecessor asserted the ABSENCE of a bare "\n" while excusing a trailing
// one, which made it unfalsifiable for any single-line message — vacuous for 3
// of the 3 rows that used it (I8). Absence of the wrong thing is not evidence of
// the right thing; this counts the terminator itself.
func assertCRLFTerminated(t *testing.T, s, where string) {
	t.Helper()
	breaks := strings.Count(s, "\n")
	crlf := strings.Count(s, "\r\n")
	if breaks == 0 {
		t.Errorf("%s: no line breaks at all — nothing was placed, so nothing is asserted: %q", where, s)
		return
	}
	if crlf != breaks {
		t.Errorf("%s: %d of %d line breaks carry their carriage return; the rest start the next line at the current column: %q",
			where, crlf, breaks, s)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the stream")
		}
		time.Sleep(time.Millisecond)
	}
}

// session.recordExchange drops an empty answer, and the guard was deletable
// green. A turn with no answer renders as "Q: … A: " in the next prompt, which
// reads as a refusal the model then imitates.
func TestAnEmptyAnswerIsNotATurn(t *testing.T) {
	sess := &session{}
	sess.recordExchange("why?", "")
	if len(sess.turns) != 0 {
		t.Errorf("turns = %+v, want none — an unanswered question is not an exchange", sess.turns)
	}
	sess.recordExchange("why?", "Because.")
	if len(sess.turns) != 1 {
		t.Fatalf("turns = %+v, want the answered one", sess.turns)
	}
	// And it reaches the prompt, which is the only reason to keep it.
	if got := renderAskPrompt(askContext{Question: "more", Turns: sess.turns}); !strings.Contains(got.Prompt, "Because.") {
		t.Errorf("the recorded turn did not reach the prompt:\n%s", got.Prompt)
	}
}

// rawterm.go claims the key channel's buffering is load-bearing: the loop stops
// reading while an answer streams, so on an unbuffered channel the reader blocks
// on the first key typed during it and never decodes the Ctrl-C BEHIND it. The
// claim was deletable green (make(chan Key, 256) → make(chan Key)).
//
// The observable that distinguishes it: type a key FIRST, then Ctrl-C, both
// while the answer is streaming. With buffering the interrupt is decoded and the
// session survives; without it the reader is still blocked on the first key.
func TestAKeyTypedBeforeCtrlCDoesNotBlockTheReader(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture, Stall: true})
	d.stdinIsTerminal = func() bool { return true }

	interrupts := &interrupter{fn: func() { t.Error("the session cancel fired") }}
	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close() })
	keys := readKeys(t.Context(), pr, interrupts)

	var out, errb syncBuf
	done := make(chan int, 1)
	go func() {
		done <- runEditor(t.Context(), keys, interrupts, d, options{noAudio: true, locale: "us", tty: true},
			func(run func()) error { run(); return nil }, func() {}, &out, &errb)
	}()

	io.WriteString(pw, "?why\r")
	waitFor(t, func() bool { return strings.Contains(out.String(), "Obsequious") })

	// A keystroke AHEAD of the interrupt: this is the byte the reader would be
	// blocked on if the channel could not hold it.
	io.WriteString(pw, "x")
	io.WriteString(pw, "\x03")

	// The session survived, which means the interrupt was decoded behind the
	// buffered "x" — and the "x" itself is still there as type-ahead.
	io.WriteString(pw, "yz\r")
	// The miss is reported on STDERR; what matters is that the line was "xyz"
	// and not "yz" — the "x" survived as type-ahead.
	waitFor(t, func() bool { return strings.Contains(errb.String(), "define: xyz:") })
	pw.Close()
	<-done
}

// README's rule for this block: every record type or field the tool writes into
// the working directory is named there. The enumeration is
// {words/, events/, user-model.md} x {looked-up, asked} — and the claim that
// answers are NOT stored is the one a reader most needs to be able to trust.
// README: "every question that reaches the model is recorded, whatever became of
// the answer". An absolute, so it names its enumeration and asserts every cell —
// three rounds of doc-overstates-code findings say a claim written without one
// is a claim measured in exactly one cell (BR-39 measured this one true in 1 of 4).
func TestAQuestionIsRecordedWhateverBecameOfTheAnswer(t *testing.T) {
	for _, tc := range []struct {
		name   string
		reply  llmtest.Reply
		cancel bool
	}{
		{"answered", llmtest.Reply{Capture: streamCapture}, false},
		{"refused by the service", llmtest.Reply{Status: 400}, false},
		{"cut off mid-answer", llmtest.Reply{Capture: streamCapture, Stall: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, fake, st, _ := askRig(t)
			fake.Script("", tc.reply)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var out syncBuf
			var errb bytes.Buffer
			if tc.cancel {
				go func() {
					for out.Len() == 0 {
						time.Sleep(time.Millisecond)
					}
					cancel()
				}()
			}
			runAsk(ctx, d, options{}, &session{}, question{text: "is it pejorative?"}, &out, &errb)

			ev, err := st.Events(time.Time{})
			if err != nil {
				t.Fatal(err)
			}
			if len(ev) != 1 || ev[0].Question != "is it pejorative?" {
				t.Fatalf("events = %+v, want the question recorded", ev)
			}
		})
	}

	t.Run("but not when no model was ever reached", func(t *testing.T) {
		_, _, st, _ := askRig(t)
		d := testDeps(t)
		d.capture = newStoreCapturer(st, store.FixedClock(aDay), nil)
		d.getenv = func(string) string { return "" } // no seam at all

		var out, errb bytes.Buffer
		runAsk(t.Context(), d, options{}, &session{}, question{text: "why"}, &out, &errb)

		ev, _ := st.Events(time.Time{})
		if len(ev) != 0 {
			t.Errorf("events = %+v, want none — nothing was asked of anything", ev)
		}
	})
}

func TestTheEventLogHoldsQuestionsAndNotAnswers(t *testing.T) {
	d, fake, st, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})

	var out, errb bytes.Buffer
	runAsk(t.Context(), d, options{}, &session{current: "sycophantic"},
		question{text: "is it pejorative?"}, &out, &errb)

	ev, err := st.Events(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 1 || ev[0].Question != "is it pejorative?" {
		t.Fatalf("events = %+v, want the question recorded", ev)
	}
	// The answer streamed to the user; none of it may be on disk.
	if !strings.Contains(out.String(), "Obsequious") {
		t.Fatal("the answer never streamed, so this proves nothing")
	}
	for _, e := range ev {
		if strings.Contains(e.Question, "Obsequious") || strings.Contains(e.Word, "Obsequious") {
			t.Errorf("answer text reached the log: %+v", e)
		}
	}
}

// THE ASK-WIRING TABLE. Its rows ARE the cells.
//
// M2 adds three facts to the ask path — which session it carries, where the
// answer is written, and whether the interrupt is scoped — across three entry
// modes. Two rounds answered that enumeration with targeted one-off tests, and
// three cells kept surviving their mutation with the full suite green (BR-38).
// One fixture-driven test, the way TestRawNeverAsks does its six.
type askMode struct {
	name string
	// carriesSession is false for the one-shot: one line, one process, so there
	// is no session to carry and no interrupt to return to.
	carriesSession bool
	drive          func(t *testing.T, d deps, lines []string, ints *interrupter, out, errOut io.Writer)
}

func askModes() []askMode {
	return []askMode{
		{
			name: "one-shot",
			drive: func(t *testing.T, d deps, lines []string, _ *interrupter, out, errOut io.Writer) {
				run(t.Context(), append([]string{"-no-audio"}, lines[0]), d, strings.NewReader(""), out, errOut)
			},
		},
		{
			name: "piped", carriesSession: true,
			drive: func(t *testing.T, d deps, lines []string, ints *interrupter, out, errOut io.Writer) {
				replLines(t.Context(), ints, d, options{noAudio: true, locale: "us"},
					strings.NewReader(strings.Join(lines, "\n")+"\n"), out, errOut, true, false)
			},
		},
		{
			name: "editor", carriesSession: true,
			drive: func(t *testing.T, d deps, lines []string, ints *interrupter, out, errOut io.Writer) {
				d.stdinIsTerminal = func() bool { return true }
				runEditor(t.Context(), scriptKeys(strings.Join(lines, "\r")+"\r"), ints, d,
					options{noAudio: true, locale: "us", tty: true},
					func(r func()) error { r(); return nil }, func() {}, out, errOut)
			},
		},
	}
}

func TestTheAskWiringTable(t *testing.T) {
	for _, mode := range askModes() {
		t.Run(mode.name+"/the answer reaches stdout", func(t *testing.T) {
			d, fake, _, _ := askRig(t)
			fake.Script("", llmtest.Reply{Capture: streamCapture})
			var out, errb bytes.Buffer
			mode.drive(t, d, []string{"?why"}, &interrupter{}, &out, &errb)

			if !strings.Contains(out.String(), "Obsequious") {
				t.Errorf("the answer did not reach stdout: %q / %q", out.String(), errb.String())
			}
		})

		t.Run(mode.name+"/the session carries across lines", func(t *testing.T) {
			if !mode.carriesSession {
				t.Skip("one line, one process: nothing to carry")
			}
			d, fake, _, _ := askRig(t)
			fake.Script("", llmtest.Reply{Capture: streamCapture}, llmtest.Reply{Capture: streamCapture})
			var out, errb bytes.Buffer
			// A lookup, then two questions: the second must carry the word the
			// lookup made current AND the first exchange.
			mode.drive(t, d, []string{"sycophantic", "?is it pejorative", "?give me two more"},
				&interrupter{}, &out, &errb)

			reqs := fake.Requests()
			if len(reqs) != 2 {
				t.Fatalf("requests = %d, want 2: %q", len(reqs), errb.String())
			}
			second := reqs[1].Prompt()
			for _, want := range []string{"sycophantic", "is it pejorative", "servile deference"} {
				if !strings.Contains(second, want) {
					t.Errorf("%s lost its session — %q missing from the second prompt:\n%s", mode.name, want, second)
				}
			}
		})

		t.Run(mode.name+"/the interrupt is scoped to the answer", func(t *testing.T) {
			if !mode.carriesSession {
				t.Skip("a one-shot has no session to return to: ending it IS correct")
			}
			d, fake, _, _ := askRig(t)
			fake.Script("", llmtest.Reply{Capture: streamCapture, Stall: true})
			// The sink must be pointed at the QUESTION while the answer streams,
			// so firing it must not reach this.
			fired := false
			ints := &interrupter{fn: func() { fired = true }}

			var out, errb syncBuf
			done := make(chan struct{})
			go func() {
				defer close(done)
				mode.drive(t, d, []string{"?why"}, ints, &out, &errb)
			}()
			waitFor(t, func() bool { return strings.Contains(out.String(), "Obsequious") })
			ints.Fire()
			<-done

			if fired {
				t.Errorf("%s: the SESSION cancel fired — the interrupt was not scoped to the answer", mode.name)
			}
		})
	}
}

// A user-model.md that exists but cannot be READ must say so.
//
// store/yaml.go returns an error for exactly this case, and its comment gives
// the reason: answering "" for a model that exists pitches every answer at the
// wrong level with no way to tell. The warn was correct and nothing failed
// without it — and failingStore.UserModel, added for it, had zero call sites
// (BR-24).
func TestAnUnreadableUserModelIsReported(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})
	d.deck = failingStore{}

	var out, errb bytes.Buffer
	code := runAsk(t.Context(), d, options{}, &session{}, question{text: "why"}, &out, &errb)

	if code != 0 {
		t.Errorf("exit = %d — an unreadable model must not fail the answer", code)
	}
	if !strings.Contains(errb.String(), "user-model.md") {
		t.Errorf("stderr = %q, want it to name user-model.md", errb.String())
	}
	// And the answer still came: a missing model degrades the answer's aim, not
	// the answer itself.
	if !strings.Contains(out.String(), "Obsequious") {
		t.Errorf("the answer did not arrive: %q", out.String())
	}
}

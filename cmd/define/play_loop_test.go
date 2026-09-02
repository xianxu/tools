package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/schedule"
	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
)

// playRig builds a session over a real Mem store, a fake dictionary and a fixed
// clock — everything a session touches except the terminal.
func playRig(t *testing.T, words ...string) (deps, options, *store.Mem) {
	t.Helper()
	st := store.NewMem()
	for _, w := range words {
		if err := st.Upsert(store.Word{Text: w, FirstSeen: aDay, LastSeen: aDay, Lookups: 1}); err != nil {
			t.Fatal(err)
		}
	}
	d := testDeps(t)
	d.deck = st
	d.clock = store.FixedClock(aDay.AddDate(0, 0, 30))
	d.capture = newStoreCapturer(st, d.clock, nil, nil)
	// fakePlayer, because playAnnounced shells out to afplay(1) and a real player
	// in a test is a real process.
	d.player = &fakePlayer{}
	d.audio = noAudioSource{}
	// COLOUR AND TTY ON, which is the only configuration a sitting can be in:
	// `--play` refuses unless `opt.tty`, and `tty` and `color` are the identical
	// expression at the flag parse (main.go). A rig whose defaults are
	// unreachable from the command under test is a rig that tests a state
	// production cannot produce — which is how BR-23's Critical shipped, with
	// both of Done-when 0b's pins green over uncoloured text.
	// WIDTH 80, which is what a sitting actually has: `--play` refuses unless
	// stdout is a terminal, so opt.width is terminalWidth's real answer and never
	// the 0 sentinel. A rig that left it 0 turned the wrap OFF, and the wrap is
	// what every region has to survive — that default hid a dropped click map on
	// every multiple-choice question until the operator found it.
	// ROWS 24, for the same reason width is 80 and stated again because it is the
	// same trap: a rig whose default is a state production cannot produce hides
	// the feature that reads it. `--play` refuses unless stdout is a terminal, so
	// opt.rows is terminalRows' real answer and never zero — and zero would make
	// fitsABoard false for every board, so #40's whole form would never be
	// offered in any test while looking perfectly healthy.
	return d, options{color: true, tty: true, width: defaultCols, rows: defaultRows, count: 20, times: 1, noAudio: true}, st
}

// audible makes the playback branch REACHABLE and returns the player recording it.
//
// playRig deliberately installs noAudioSource AND noAudio:true, so every test
// that says anything about playback — that it happens, or that it does not —
// has to undo both. Inline, that was five copies of the same three lines (BR-5).
//
// The copy that matters is the one that forgets `d.audio`: a negative assertion
// ("a correct answer plays nothing") against a source that can never produce a
// recording passes for the wrong reason, whatever the session does. That is the
// vacuous check PQ-6 caught, and one helper is how it stops being possible to
// write again by hand.
func audible(d *deps, opt *options) *fakePlayer {
	fp := &fakePlayer{}
	d.player = fp
	d.audio = okAudio{}
	opt.noAudio = false
	return fp
}

func reviewEvents(t *testing.T, st *store.Mem) []store.ReviewEvent {
	t.Helper()
	all, err := st.Events(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	var out []store.ReviewEvent
	for _, e := range all {
		if e.Kind == store.EventReviewed {
			out = append(out, e)
		}
	}
	return out
}

// keysFor turns a script into the channel runPlay reads. Enter reveals, y/n
// grade, and ^C interrupts.
func keysFor(script string) <-chan Key {
	ch := make(chan Key, len(script)+1)
	for _, r := range script {
		switch r {
		case '\r':
			ch <- Key{Kind: KeyEnter}
		case '^':
			ch <- Key{Kind: KeyInterrupt}
		default:
			ch <- Key{Kind: KeyRune, Rune: r}
		}
	}
	close(ch)
	return ch
}

// playbackConsole is the console a sitting's test drives.
//
// recordingConsole's shape unchanged — one recorder for the frame AND stdout,
// which is production's wiring (#30 D5b) — with a hand-back that does nothing,
// because a test has no terminal to give back. Named here so a `--play` test
// says which loop it is driving; #41 made the console the shared seam, and the
// helper being shared is the point rather than an accident.
func playbackConsole(out, errb io.Writer) console {
	return recordingConsole(out, errb, func() {})
}

// questionsFor is the queue AND the deck the sitting holds in memory — both,
// because todaysQuestions computes both and a test that dropped the second would
// drive every sitting against an empty deck, where the cost figures are zero and
// the bar cannot be wrong (D7).
func questionsFor(t *testing.T, d deps, opt options) ([]play.Question, *sittingDeck) {
	t.Helper()
	var out, errb bytes.Buffer
	qs, held, code := todaysQuestions(d, opt, &out, &errb)
	if code != 0 {
		t.Fatalf("todaysQuestions = %d, stderr %s", code, errb.String())
	}
	return qs, held
}

// gradeKey is the keystroke that grades q with the wanted verdict, WHICHEVER
// form q is.
//
// Necessary because a session's forms are chosen by the deck, not by the test:
// a rig with two words now produces form 2.3, whose answer sits in a shuffled
// position, so "y" and "n" stopped being answers at all. Tests that assert
// something about the SESSION — that it records once per answer, that an
// interrupt preserves what was recorded — should not have to care which form
// asked, and before this they silently did.
//
// Naming *play.Choice here is fine where Done-when 7 forbids it in Apply: the
// prohibition is on the session learning about forms, and this is a test
// deciding what to type.
func gradeKey(t *testing.T, q play.Question, want play.Verdict) string {
	t.Helper()
	if c, ok := q.(*play.Choice); ok {
		for i, o := range c.Options() {
			if o.Correct == (want == play.Correct) {
				return string(rune('1' + i))
			}
		}
		t.Fatalf("form 2.3 for %q has no option giving %v: %+v", q.Word(), want, c.Options())
	}
	if want == play.Correct {
		return "y"
	}
	return "n"
}

// THE DONE-WHEN: a full session against a fake store and fake clock records one
// event per answer.
func TestFullSessionRecordsOneEventPerAnswer(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic", "ephemeral")
	qs, held := questionsFor(t, d, opt)
	if len(qs) != 2 {
		t.Fatalf("got %d questions, want 2", len(qs))
	}

	var out, errb bytes.Buffer
	keys := "\r" + gradeKey(t, qs[0], play.Correct) + "\r" + gradeKey(t, qs[1], play.Wrong)
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor(keys), playbackConsole(&out, &errb))

	events := reviewEvents(t, st)
	if len(events) != 2 {
		t.Fatalf("got %d review events, want 2: %+v", len(events), events)
	}
	if !events[0].Correct {
		t.Error("first answer recorded as wrong")
	}
	if events[1].Correct {
		t.Error("second answer recorded as correct")
	}
}

// THE DONE-WHEN: interrupting mid-session preserves already-recorded events.
//
// A separate test because "preserves what was recorded" is a different claim
// from "records", and it is the one that would break if anything batched.
func TestInterruptPreservesRecordedEvents(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic", "ephemeral")
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	// Answer the first, then Ctrl-C before the second.
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("\r"+gradeKey(t, qs[0], play.Correct)+"^"), playbackConsole(&out, &errb))

	events := reviewEvents(t, st)
	if len(events) != 1 {
		t.Fatalf("got %d events after an interrupt, want the 1 already answered", len(events))
	}
	if !events[0].Correct {
		t.Error("the preserved event has the wrong verdict")
	}
}

// THE DONE-WHEN: a full session that NEVER REACHES for the model.
//
// This used to point the seam at a dead address and assert the session finished
// — which a session using a WORKING model would also do, since d.newLLM is read
// only by ask.go and reflect.go and the play path never touches it. The seam was
// unreachable from the code under test, so the test was byte-identical in
// meaning to TestFullSessionRecordsOneEventPerAnswer and the Done-when row it
// stood for could not fail (BR-48).
//
// The discriminating double is this repo's refusingDict shape: a seam that fails
// the test WHEN USED. "Degrades when the model is unavailable" and "never asks"
// are different claims, and only the second one is form 2.1's.
//
// Scope, measured: this drives playSession, so it pins the SESSION. Adding the
// same construction to runPlay leaves it green — the first mutation of this test
// did exactly that and passed. runPlay's own reach is pinned by its coverage,
// not here.
func TestSessionRunsWithTheModelUnavailable(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic")
	d.getenv = envFor("http://127.0.0.1:1")
	d.newLLM = func(llm.Config) llm.Client {
		t.Error("the review loop constructed a model client; form 2.1 needs no model at all")
		return llm.New(llm.Config{})
	}
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("\ry"), playbackConsole(&out, &errb))

	if len(reviewEvents(t, st)) != 1 {
		t.Error("the session did not complete")
	}
}

// An empty sitting is the expected state most days, not an error.
//
// This test asserted "nothing due" for a deck with NO WORDS IN IT — it was
// written against the single conflated message and so encoded the conflation
// BR-46 names. The exit code is what its name is about and that has not changed;
// what it says now is the cause, which for an empty deck is not the schedule.
func TestEmptyQueueExitsZero(t *testing.T) {
	d, opt, _ := playRig(t) // no words at all
	var out, errb bytes.Buffer

	// Driven at todaysQuestions rather than at runPlay, and the move is BR-3's
	// doing: `--play` now settles its terminal preconditions before it reads
	// anything, so runPlay in a test — which has no terminal — refuses before
	// the queue is ever built. The CLAIM is unchanged and this is where it
	// lives; that runPlay reaches this code at all is pinned separately, by the
	// dispatch row in TestClaimsWithoutTestsUntilNow.
	qs, _, code := todaysQuestions(d, opt, &out, &errb)

	if code != 0 {
		t.Errorf("exit = %d, want 0 — an empty sitting is not a failure", code)
	}
	if len(qs) != 0 {
		t.Errorf("got %d questions from a deck with no words", len(qs))
	}
	if !strings.Contains(out.String(), "the deck is empty") {
		t.Errorf("stdout = %q, want it to name the empty deck rather than the schedule", out.String())
	}
}

// BR-3: the FULL-SCREEN surface is gated on stdout, not on stdin alone.
//
// `repl` computes `terminalUI := interactive && opt.tty` and falls back to the
// line loop, with a comment recording that this family — gating cursor control
// on the wrong stream — has already shipped three times. `--play` checked stdin
// only, which was harmless while it appended lines and emitted no escapes at
// all. Since #41 a sitting takes the alternate screen, reports the mouse and
// paints `ESC[H ESC[J` frames, so `define --play > file` would write them into
// the file at a fabricated 80 columns.
//
// The assertion is on the ESCAPES, not on the message: a guard that returned 1
// after already sending the alt-screen sequence would pass a message check.
func TestPlayRefusesWhenStdoutIsNotATerminal(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	tty, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer tty.Close()
	d.stdinIsTerminal = func() bool { return true }

	var out, errb bytes.Buffer
	code := runPlay(t.Context(), d, opt, tty, &out, &errb)

	if code != 1 {
		t.Errorf("exit = %d, want 1 — a sitting the learner cannot see is not a sitting", code)
	}
	if strings.Contains(out.String(), "\x1b") {
		t.Errorf("escape sequences reached a non-terminal stdout: %q", out.String())
	}
	if !strings.Contains(errb.String(), "must be a terminal") {
		t.Errorf("stderr = %q, want it to say why", errb.String())
	}
}

// No deck, no session — guarded on the PRECONDITION, since DEFINE_NO_CAPTURE and
// a Getwd failure both produce it and keying on the flag would panic on the other
// path.
func TestNoDeckExitsZeroWithAMessage(t *testing.T) {
	d := testDeps(t)
	d.deck = nil
	var out, errb bytes.Buffer

	code := runPlay(t.Context(), d, options{}, strings.NewReader(""), &out, &errb)

	if code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "no deck") {
		t.Errorf("stdout = %q, want an explanation", out.String())
	}
}

// A skip records NOTHING — asserted on the CAPTURER, not on the event log.
//
// The log cannot tell the two implementations apart: an OutcomeNone carries an
// empty Word, and CaptureReview drops those anyway, so a loop that recorded
// every outcome would still write nothing and this test would pass. The
// discriminating observable is whether the capturer was CALLED at all.
func TestUngradedKeyNeverReachesTheCapturer(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	spy := &countingCapturer{}
	d.capture = spy
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	// Reveal, then a key form 2.1 does not grade, then interrupt.
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("\rz^"), playbackConsole(&out, &errb))

	if spy.reviews != 0 {
		t.Errorf("CaptureReview called %d times for an ungraded key — the loop is recording outcomes it should not",
			spy.reviews)
	}
}

// And a graded key DOES reach it, or the test above would pass on a loop that
// never records anything at all.
func TestGradedKeyReachesTheCapturerExactlyOnce(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	spy := &countingCapturer{}
	d.capture = spy
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("\ry"), playbackConsole(&out, &errb))

	if spy.reviews != 1 {
		t.Errorf("CaptureReview called %d times for one graded answer, want 1", spy.reviews)
	}
}

// A cancelled context ends the session, and what was recorded stays.
//
// PROBABILISTIC, and stated as such — the previous version of this comment said
// "DETERMINISTIC by construction", which was false. A reviewer measured the
// claim: with the pre-select guard removed, 119 of 400 runs went red, matching
// the theoretical 0.25 for winning two consecutive selects. Go's `select` picks
// uniformly among ready cases, so no single run can be made to catch this.
//
// What CAN be done is repeat until a surviving mutant is not a plausible event.
// Each round pre-reveals the question so ONE select win produces one graded
// answer — the highest per-round catch probability available, ~0.5 — and 40
// rounds put a survivor at ~1e-12.
func TestCancelledContextEndsTheSession(t *testing.T) {
	const rounds = 40

	for i := 0; i < rounds; i++ {
		d, opt, st := playRig(t, "sycophantic", "ephemeral")
		qs, held := questionsFor(t, d, opt)
		spy := &countingCapturer{}
		d.capture = spy

		// Pre-revealed: the very first input is a GRADE, so a single select win
		// is immediately observable.
		s := play.NewSession(qs)
		s, _ = play.Apply(s, play.Input{Kind: play.InputReveal})

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		var out, errb bytes.Buffer
		code := playSession(ctx, d, opt, s, held, keysFor("yyyy"), playbackConsole(&out, &errb))

		if code != 0 {
			t.Fatalf("round %d: exit = %d, want 0 — an interrupted session is not a failure", i, code)
		}
		if spy.reviews != 0 {
			t.Fatalf("round %d: graded %d answers after the context was already cancelled", i, spy.reviews)
		}
		if len(reviewEvents(t, st)) != 0 {
			t.Fatalf("round %d: recorded an event after cancellation", i)
		}
	}
}

// The queue comes from the SCHEDULE: a word not due is not offered.
func TestOnlyDueWordsAreOffered(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic", "ephemeral")
	// Review ephemeral today, so it moves to box 1 and is not due again for 3 days.
	if err := st.AppendEvent(store.ReviewEvent{
		Word: "ephemeral", Kind: store.EventReviewed, Correct: true, At: d.clock.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	qs, _ := questionsFor(t, d, opt)

	if len(qs) != 1 || qs[0].Word() != "sycophantic" {
		var got []string
		for _, q := range qs {
			got = append(got, q.Word())
		}
		t.Errorf("queue = %v, want only sycophantic — ephemeral was just reviewed", got)
	}
}

// EVERY newline a session PAINTS must be a full CRLF, and #41 changed who owes
// that guarantee.
//
// This is the defect the first version of --play shipped, and it is worth
// recording WHY the pty smoke test missed it: that test captured the bytes and
// printed them through Python, where a bare \n renders at column 0 and the
// output looked perfect. A real terminal in raw mode does not — it moves down
// and stays put — so the definition cascaded diagonally across the screen, each
// line starting where the last one ended.
//
// The owner used to be a translating writer wrapped around stdout. It is now Paint, which
// places every row itself — so the test drives a REAL liveScreen over a buffer
// standing in for the tty, rather than a recorder that writes what it is given.
// A double could not fail this: it has no rows to place.
func TestSessionOutputIsAllCRLF(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic", "ephemeral")
	qs, held := questionsFor(t, d, opt)

	var raw, errb bytes.Buffer
	live := newLiveScreen(&raw, 24, 80)
	live.interval = -1 // every write paints; nothing waits on a real clock
	playSession(t.Context(), d, opt, play.NewSession(qs), held,
		keysFor("\r"+gradeKey(t, qs[0], play.Correct)+"\r"+gradeKey(t, qs[1], play.Wrong)),
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	got := raw.String()
	if strings.Count(got, "\n") == 0 {
		t.Fatal("no newlines at all — this test would assert nothing")
	}
	if bare := strings.Count(got, "\n") - strings.Count(got, "\r\n"); bare != 0 {
		t.Errorf("%d bare newlines in painted output — in raw mode each one starts the next line "+
			"where the last ended, which is the diagonal cascade", bare)
	}
	// And the definition itself must be in there, or the assertion above is
	// ranging over prompts alone.
	if !strings.Contains(got, "adjective") {
		t.Error("the rendered definition never reached the terminal")
	}
}

// paintedSitting is a sitting drawn on a REAL screen: the buffer, the live edge
// and the terminal bytes all separable, which a recorder cannot give.
//
// Three of #41's Done-when rows are about WHICH of those three a thing lands in,
// so the double that folds them into one writer cannot see any of them.
func paintedSitting(t *testing.T, d deps, opt options, qs []play.Question, held *sittingDeck,
	script string, normal io.Writer) (*liveScreen, *bytes.Buffer) {
	t.Helper()
	tty := &bytes.Buffer{}
	// PINNED, which is what production builds for a sitting (D3a) — a test on
	// the editor's screen would paint a frame --play never draws.
	live := newPinnedScreen(tty, 24, 80)
	live.interval = -1 // every write paints; nothing here waits on a real clock
	var errb bytes.Buffer
	finish := func() {}
	if normal != nil {
		// A rawSession with a Builder for its control stream: enough for
		// handBack to restore something, with no terminal anywhere.
		finish = onceHandBack(live, &rawSession{control: &strings.Builder{}}, normal)
	}
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor(script),
		console{view: live, finish: finish, stdout: live, stderr: &errb})
	return live, tty
}

// DONE-WHEN 1: --play paints whole frames, and the live edge is never filed.
//
// The keys are rewritten on every keystroke, so a path that APPENDS them files a
// copy per key — and every one of those copies would be in the exit transcript,
// claiming the learner was offered the keys a hundred times. That is what
// "no path appends bare lines" is worth asserting about: not the absence of a
// call, but the difference between the transcript and the frame.
func TestPlayDrawsThroughTheDisplay(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	qs, held := questionsFor(t, d, opt)

	live, tty := paintedSitting(t, d, opt, qs, held, "\r^", nil)

	if got := live.Transcript(); strings.Contains(got, sessionKeys) {
		t.Errorf("the grading keys reached the BUFFER:\n%s\nThey are the live edge — appending them "+
			"files a copy per keystroke and puts every one of them in the exit transcript", got)
	}
	if !strings.Contains(tty.String(), sessionKeys) {
		t.Error("the grading keys were never painted — a learner who cannot see what to press cannot answer")
	}
	// ...and the question IS the transcript, or the assertion above passes for
	// a session that drew nothing at all.
	if !strings.Contains(live.Transcript(), "sycophantic") {
		t.Error("the word never reached the buffer")
	}
}

// DONE-WHEN 2: one question leaves ONE copy in the buffer, however many keys are
// pressed. This is the assertion the naive port fails (D4).
func TestRepeatedKeystrokesDoNotDuplicateTheQuestion(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	qs, held := questionsFor(t, d, opt)

	// Keys this form does not use: every one is OutcomeNone, so the session
	// does not move and the only thing that happens is a redraw.
	live, _ := paintedSitting(t, d, opt, qs, held, "qqqqqqq^", nil)

	if n := strings.Count(live.Transcript(), qs[0].Prompt()); n != 1 {
		t.Errorf("the question is in the buffer %d times after 7 keystrokes, want 1 — "+
			"a draw that writes the question every frame files a copy per key", n)
	}
}

// DONE-WHEN 5: the transcript survives exit, and the summary is part of it.
//
// The same shape #30 used for the editor (TestHandBackStopsPaintingRestoresThen-
// PrintsTheSession), driven through a whole sitting — and it pins the ORDER the
// loop's exit owes as well: the summary is written into the buffer BEFORE the
// hand-back prints it, so reversing them loses the last line a learner sees.
func TestPlayTranscriptSurvivesExit(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	qs, held := questionsFor(t, d, opt)

	var normal bytes.Buffer
	paintedSitting(t, d, opt, qs, held, "\ry", &normal)

	if !strings.Contains(normal.String(), "sycophantic") {
		t.Errorf("the sitting vanished with the alternate screen: %q", normal.String())
	}
	if !strings.Contains(normal.String(), "1 right") {
		t.Errorf("the summary is not in the transcript: %q\nIt is written into the buffer, so a "+
			"hand-back that runs first prints a buffer that does not have it yet", normal.String())
	}
}

// DONE-WHEN 3: the bar is pinned across question, reveal and resize.
//
// Drawn in EVERY state, which is what "pinned" means for a footer: a state that
// forgets to pass it is a bar that blinks out exactly when the learner is
// reading. Asserted on the frames, not on the buffer — the bar is the live edge
// and filing it would be the other bug (Done-when 1).
func TestTheBarSurvivesEveryState(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	view := paintInto(&out)
	// Question, then reveal, then graded, then a resize, then the summary.
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("\rn^"),
		console{view: view, finish: func() {}, stdout: view, stderr: &errb})

	if len(view.menus) == 0 {
		t.Fatal("no frames drawn at all")
	}
	for i, footer := range view.menus {
		if len(footer) != 1 {
			t.Fatalf("frame %d drew %d footer rows, want the one bar: %q", i, len(footer), footer)
		}
		if !strings.Contains(footer[0], "reviews/day") {
			t.Errorf("frame %d's footer is not the bar: %q", i, footer[0])
		}
	}
}

// The bar's PROGRESS moves as answers land, which is the half a static footer
// would pass. Two questions, two answers, three distinct counters.
func TestTheBarCountsAnswersAsTheyLand(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic", "ephemeral")
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	view := paintInto(&out)
	playSession(t.Context(), d, opt, play.NewSession(qs), held,
		keysFor(gradeKey(t, qs[0], play.Correct)+gradeKey(t, qs[1], play.Correct)),
		console{view: view, finish: func() {}, stdout: view, stderr: &errb})

	var counters []string
	for _, footer := range view.drawnMenus() {
		// UNSTYLED, because the bar is drawn as chrome now (#44) and this test is
		// about the COUNTER — a dim escape on the front of the row would otherwise
		// read as part of the number.
		bar := unstyled(footer[0])
		// Cut at the separator, and FAIL rather than panic if it moves: a bar
		// format change should read as a broken assertion, not as a slice
		// bounds error in a test about counting.
		i := strings.Index(bar, " ·")
		if i < 0 {
			t.Fatalf("no ` ·` separator in the bar, so its shape changed: %q", bar)
		}
		if n := bar[:i]; len(counters) == 0 || counters[len(counters)-1] != n {
			counters = append(counters, n)
		}
	}
	want := []string{"0 of 2", "1 of 2", "2 of 2"}
	if !slices.Equal(counters, want) {
		t.Errorf("the bar counted %v, want %v — a bar that does not move is a bar nobody reads", counters, want)
	}
}

// DONE-WHEN 6: a whole sitting reads the deck ONCE and the log ONCE, whatever
// its length (D7).
//
// The cost claim is the reason this is a test and not a comment: Deck() reads a
// file per WORD and Events() a file per DAY of history, so a per-answer refresh
// is thousands of file reads per question with a person waiting. A counting
// store is the only thing that can see the difference.
func TestASittingReadsTheDeckOnce(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic", "ephemeral", "quokka", "mesa")
	counter := &countingStore{Store: st}
	d.deck = counter
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	var script string
	for _, q := range qs {
		script += gradeKey(t, q, play.Correct)
	}
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor(script), playbackConsole(&out, &errb))

	if counter.decks != 1 || counter.events != 1 {
		t.Errorf("a %d-answer sitting called Deck() %d times and Events() %d times, want 1 and 1 — "+
			"every extra one is a file per deck word and a file per day of log, on a path a person is waiting on",
			len(qs), counter.decks, counter.events)
	}
}

// A word dropped mid-sitting leaves the bar's deck too, or the figures charge
// for a word the learner just curated away and disagree with the deck on disk.
//
// ASSERTED ON THE DRAWN FOOTER, which is the thing the claim is about (BR-5).
// The first version of this read the caller's own `sittingDeck` after the loop
// returned, and saw the drop only because `slices.DeleteFunc` compacts the
// shared backing array in place — so rewriting `dropped` to build a new slice,
// behaviour-identical for the loop, turned it red. A test that pins an
// implementation detail of a helper is not pinning the behaviour it names.
func TestDroppingAWordLowersTheCostTheBarShows(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic", "ephemeral", "quokka", "mesa")
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	view := paintInto(&out)
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("d^"),
		console{view: view, finish: func() {}, stdout: view, stderr: &errb})

	bars := view.drawnMenus()
	if len(bars) < 2 {
		t.Fatalf("%d frames drawn, want the one before the drop and the one after", len(bars))
	}
	before, after := loadIn(t, bars[0][0]), loadIn(t, bars[len(bars)-1][0])
	if after >= before {
		t.Errorf("the bar said %.3f reviews/day before the drop and %.3f after — dropping a word "+
			"must lower what the deck costs, or the bar is charging for a word that is gone",
			before, after)
	}
}

// loadIn reads the reviews/day figure out of a drawn bar, so the assertion is on
// what the learner sees rather than on a field behind it.
func loadIn(t *testing.T, bar string) float64 {
	t.Helper()
	i := strings.Index(bar, "~")
	j := strings.Index(bar, " reviews/day")
	if i < 0 || j < i {
		t.Fatalf("no reviews/day figure in the bar: %q", bar)
	}
	n, err := strconv.ParseFloat(bar[i+1:j], 64)
	if err != nil {
		t.Fatalf("bar %q: %v", bar, err)
	}
	return n
}

// DONE-WHEN 4: a reveal taller than the terminal PAGES, and the word comes back.
//
// This is the user-visible thing #41 was filed for. Form 2.3's reveal shows the
// whole rendered entry; on a word like `run` or `bank` that is several
// screenfuls, and before frames the word being asked about scrolled off the top
// with no way back.
//
// Driven live rather than through a scripted channel, because what is asserted
// is the frame ON SCREEN at one moment — and the sitting's last acts (the
// summary, the hand-back) write to the buffer, which resets the viewport. So the
// keys are sent one at a time and each frame is read where it is drawn.
func TestALongRevealPagesRatherThanScrollingTheWordAway(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	qs, held := questionsFor(t, d, opt)

	// A SHORT terminal, which is what makes one entry several screenfuls.
	//
	// NINE rows, not eight: a sitting's frame reserves `chromeGap` between the
	// record and the live edge (#44), so eight rows now leave five for the buffer
	// and the entry's tail no longer reaches DERIVATIVES. The height is a proxy
	// for "several screenfuls" and nine is still that — the premise assertion
	// below is what actually holds the test honest.
	tty := &syncBuf{}
	live := newPinnedScreen(tty, 9, 80)
	live.interval = -1
	var errb bytes.Buffer
	keys := make(chan Key)
	done := make(chan int, 1)
	go func() {
		done <- playSession(t.Context(), d, opt, play.NewSession(qs), held, keys,
			console{view: live, finish: func() {}, stdout: live, stderr: &errb})
	}()

	keys <- Key{Kind: KeyEnter} // reveal: the entry is taller than the viewport
	waitFor(t, func() bool { return strings.Contains(unstyled(lastFrame(tty.String())), "DERIVATIVES") })
	if got := unstyled(lastFrame(tty.String())); strings.Contains(got, "sycophantic  syc·o·phan·tic") {
		t.Fatalf("the reveal fits the terminal, so this test asserts nothing:\n%s", got)
	}

	keys <- Key{Kind: KeyPageUp}
	waitFor(t, func() bool {
		return strings.Contains(unstyled(lastFrame(tty.String())), "sycophantic  syc·o·phan·tic")
	})

	keys <- Key{Kind: KeyInterrupt}
	<-done
}

// A viewport gesture is NOT an answer, and `play` never learns a viewport exists
// (D6).
//
// The other half of paging, and the one a naive wiring gets wrong: routing these
// keys through toInput would have put a display concept inside the pure package.
func TestPagingIsNotAnAnswer(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic")
	qs, held := questionsFor(t, d, opt)
	spy := &countingCapturer{}
	d.capture = spy

	var out, errb bytes.Buffer
	view := paintInto(&out)
	keys := make(chan Key, 6)
	for _, k := range []Key{
		{Kind: KeyPageUp}, {Kind: KeyPageDown}, {Kind: KeyWheelUp}, {Kind: KeyWheelDown},
		{Kind: KeyInterrupt},
	} {
		keys <- k
	}
	close(keys)
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keys,
		console{view: view, finish: func() {}, stdout: view, stderr: &errb})

	if !slices.Equal(view.pages, []int{1, -1}) {
		t.Errorf("the loop paged %v, want [1 -1] — PageUp is BACKWARD, toward older text", view.pages)
	}
	if !slices.Equal(view.lines, []int{wheelLines, -wheelLines}) {
		t.Errorf("the wheel scrolled %v, want [%d %d]", view.lines, wheelLines, -wheelLines)
	}
	if spy.reviews != 0 || len(reviewEvents(t, st)) != 0 {
		t.Error("a viewport gesture graded an answer — looking at something is not answering it")
	}
}

// BR-4: narrowing the window mid-sitting wraps the REST of the sitting to the
// new width.
//
// The operator's finding was that a frame CLIPS an over-wide option line. The
// first fix wrapped inside `choiceFor`, which bakes the sitting's STARTUP width
// into every question — so the same defect was one resize away, and the review
// measured it. The wrap is now applied where the question is WRITTEN, and the
// resize case keeps opt.width current, which is the only place that can be true
// for a question that has not been asked yet.
//
// The question already on screen keeps the wrapping it was written with, exactly
// as the editor's scrollback does (#30) — and nothing is lost by that: clipping
// happens at PAINT, so the whole text is in the buffer and returns if the window
// widens again.
func TestANarrowedSittingWrapsTheRestOfItself(t *testing.T) {
	const wide, narrow = 100, 40
	d, opt, _ := playRig(t, "sycophantic", "ephemeral", "quokka", "mesa")
	opt.width = wide
	qs, held := questionsFor(t, d, opt)
	if len(qs) < 2 {
		t.Fatalf("got %d questions, need at least 2 to advance across a resize", len(qs))
	}

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 24, wide)
	live.interval = -1
	var errb bytes.Buffer
	resizes := make(chan winSize, 1)
	keys := make(chan Key)
	done := make(chan int, 1)
	go func() {
		done <- playSession(t.Context(), d, opt, play.NewSession(qs), held, keys,
			console{view: live, resizes: resizes, finish: func() {}, stdout: live, stderr: &errb})
	}()

	waitFor(t, func() bool { return strings.Contains(live.Transcript(), qs[0].Word()) })
	written := len(live.Transcript())
	frames := strings.Count(tty.String(), cursorHome)

	// The resize FIRST, and its frame observed — otherwise the select has two
	// ready cases and picks between them at random.
	resizes <- winSize{rows: 24, cols: narrow}
	waitFor(t, func() bool { return strings.Count(tty.String(), cursorHome) > frames })

	keys <- Key{Kind: KeyRune, Rune: []rune(gradeKey(t, qs[0], play.Correct))[0]}
	waitFor(t, func() bool { return strings.Contains(live.Transcript()[written:], qs[1].Word()) })
	// ...and REVEAL it, so the rendered definition — the third instance of this
	// class, and the one wrapped by Render at queue-build time — is in the window
	// the assertion below reads.
	asked := len(live.Transcript())
	keys <- Key{Kind: KeyEnter}
	waitFor(t, func() bool { return len(live.Transcript()) > asked })
	keys <- Key{Kind: KeyInterrupt}
	<-done

	// EVERY line, not just the option lines — which is the difference between
	// fixing an instance and fixing the class. `--play` renders when the queue is
	// built and writes much later, so the question, the reveal's option line, the
	// reveal's rendered DEFINITION, the drop notice and the summary all carry a
	// width that may already be wrong. Three findings reached that rule one
	// line-kind at a time; this predicate covers a sixth site the day someone
	// adds one.
	after := live.Transcript()[written:]
	wrapped := false
	for _, line := range strings.Split(after, "\n") {
		if n := visibleCells(line); n > narrow {
			t.Errorf("a line written AFTER the window narrowed is %d columns wide in a "+
				"%d-column terminal, so the frame clips it: %q", n, narrow, line)
		}
		if strings.HasPrefix(line, strings.Repeat(" ", play.OptionIndent)) && strings.TrimSpace(line) != "" {
			wrapped = true
		}
	}
	if !wrapped {
		t.Fatalf("nothing wrapped after the resize, so this test asserts nothing:\n%s", after)
	}
}

// DONE-WHEN 7: a failed LOG read degrades to empty progress and the sitting
// still runs — with a test, not with "the existing behaviour, unchanged".
//
// The Done-when preamble requires every pin be a predicate over behaviour, and
// this was the one row that broke its own rule (BR-2). The blast radius also
// grew this window: the degraded progress now feeds the BAR and the summary,
// where before `finish` re-read the log and would have shown the true figure.
//
// The reviews are recorded as they happen, so a sitting must not end because a
// summary could not be computed — but it must also not lie about the deck.
func TestAFailedLogReadStillRunsTheSitting(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic", "ephemeral")
	d.deck = &logRefusingStore{Store: st}

	var out, errb bytes.Buffer
	qs, held, code := todaysQuestions(d, opt, &out, &errb)
	if code != 0 {
		t.Fatalf("todaysQuestions = %d — an unreadable log must not end a sitting: %s", code, errb.String())
	}
	if len(qs) == 0 {
		t.Fatal("no questions: every word should look new when there is no history")
	}
	if !strings.Contains(errb.String(), "could not read the review log") {
		t.Errorf("stderr = %q, want it to say the log could not be read", errb.String())
	}
	if held.prog == nil {
		t.Fatal("progress is nil, so the loop's first schedule.Answer would write into a nil map")
	}
	if len(held.prog) != 0 {
		t.Errorf("progress has %d entries from an unreadable log", len(held.prog))
	}

	// ...and the sitting runs to its end, drawing a bar the whole way.
	view := paintInto(&out)
	playSession(t.Context(), d, opt, play.NewSession(qs), held,
		keysFor(gradeKey(t, qs[0], play.Correct)+"^"),
		console{view: view, finish: func() {}, stdout: view, stderr: &errb})
	bars := view.drawnMenus()
	if len(bars) == 0 || !strings.Contains(bars[len(bars)-1][0], "reviews/day") {
		t.Errorf("no bar drawn on the degraded path: %v", bars)
	}
}

// logRefusingStore reads its deck and refuses its log — the partial failure
// Done-when 7 is about. failingStore refuses everything, which takes the
// earlier `return 1` and never reaches this branch.
type logRefusingStore struct {
	store.Store
}

func (logRefusingStore) Events(time.Time) ([]store.ReviewEvent, error) {
	return nil, errFail
}

// DONE-WHEN 1: the word being asked about is CLICKABLE (T4).
//
// The whole issue in one row. `Prompt()`'s first line is the headword for both
// forms, so the region is line 1 of the write — the leading blank is line 0 —
// at column 0, and a headword is never wide enough to wrap.
func TestPlayClickOnThePromptWordPlaysIt(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	player := audible(&d, &opt)
	qs, held := questionsFor(t, d, opt)

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 24, 80)
	live.interval = -1
	var errb bytes.Buffer
	keys := make(chan Key)
	done := make(chan int, 1)
	go func() {
		done <- playSession(t.Context(), d, opt, play.NewSession(qs), held, keys,
			console{view: live, finish: func() {}, stdout: live, stderr: &errb})
	}()

	waitFor(t, func() bool { return strings.Contains(live.Transcript(), qs[0].Word()) })
	// The cell the WORD is on, found the way a reader's eye would: the buffer
	// line carrying it, at column 0. No coordinate invented by the test.
	row := -1
	for i, line := range strings.Split(live.Transcript(), "\n") {
		if unstyled(line) == qs[0].Word() {
			row = i
		}
	}
	if row < 0 {
		t.Fatalf("the prompt word is not on a line of its own:\n%s", live.Transcript())
	}

	keys <- Key{Kind: KeyClick, Row: row, Col: 0}
	waitFor(t, func() bool { return player.count() > 0 })
	keys <- Key{Kind: KeyInterrupt}
	<-done

	if player.count() == 0 {
		t.Error("clicking the word the sitting is asking about played nothing")
	}
}

// DONE-WHEN 2: a click ACTS and never answers (D8, T7).
//
// The first version of this row did not discriminate, and the boundary review
// measured it: disabling the whole `KeyClick` branch left it green, and so did
// routing a click into `play.Apply` as an `InputReveal`. The "never answers"
// half is delivered by `toInput`'s default — it returns false for `KeyClick` —
// not by T7's guard, so a test asserting only that nothing was recorded pins a
// property the guard does not provide.
//
// **A `red when` cell is a mutation, and it has to be RUN.** The discriminating
// state is GRADED: there "any key = next word", so a click reaching Apply
// advances the question. So this drives a real pinned screen to the graded
// state, clicks, and asserts BOTH halves — the click played, and the sitting did
// not move on.
func TestPlayClickActsAndIsNotAnAnswer(t *testing.T) {
	// TWO words, and that is load-bearing. With one, a click that reached Apply
	// would ADVANCE off the last question and simply end the sitting — which
	// looks identical to not advancing, and the review's own `red when` mutation
	// passed against exactly that. A second question is what makes the advance
	// observable.
	d, opt, st := playRig(t, "sycophantic", "ephemeral")
	player := audible(&d, &opt)
	qs, held := questionsFor(t, d, opt)
	if len(qs) < 2 {
		t.Fatalf("got %d questions, need 2 so an advance is visible", len(qs))
	}

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 200, opt.width)
	live.interval = -1
	var errb bytes.Buffer

	// SCRIPTED and buffered, so the loop consumes them in order and nothing is
	// observed mid-flight. A WRONG answer on a hidden word records the miss and
	// reveals WITHOUT advancing — the graded state, where "any key = next word"
	// is live and a click reaching Apply would move the sitting on.
	//
	// Row 1 is the word, and it is not a guess: show() writes "\n"+Prompt()+"\n"
	// as the sitting's first write, so line 0 is the leading blank.
	keys := make(chan Key, 3)
	keys <- Key{Kind: KeyRune, Rune: []rune(gradeKey(t, qs[0], play.Wrong))[0]}
	keys <- Key{Kind: KeyClick, Row: 1, Col: 0}
	keys <- Key{Kind: KeyInterrupt}
	close(keys)

	playSession(t.Context(), d, opt, play.NewSession(qs), held, keys,
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	// The premise: the click landed on something. Without this the assertions
	// below pass for a sitting where the region was never written.
	if _, ok := live.RegionAtRow(1, 0); !ok {
		t.Fatalf("row 1 offers no region, so no click was possible:\n%s", live.Transcript())
	}
	// It ACTED: the miss plays once, the click plays again.
	if got := player.count(); got < 2 {
		t.Errorf("played %d times, want the miss AND the click — the click did nothing", got)
	}
	// ...and it did not ANSWER. A click reaching Apply in the graded state
	// advances past the word, and the sitting would end "0 right, 0 wrong" on a
	// second question instead.
	if n := len(reviewEvents(t, st)); n != 1 {
		t.Errorf("%d review events, want the one miss — a click graded", n)
	}
	if !strings.Contains(live.Transcript(), "0 right, 1 wrong") {
		t.Errorf("the sitting did not end on the miss alone:\n%s", live.Transcript())
	}
	// THE DISCRIMINATOR: the sitting never moved on. A click that reached Apply
	// in the graded state advances, and the next question would be written.
	if strings.Contains(unstyled(live.Transcript()), qs[1].Word()) {
		t.Errorf("the sitting advanced to %q — the click was consumed as an answer:\n%s",
			qs[1].Word(), live.Transcript())
	}
}

// DONE-WHEN 3: a revealed definition is clickable like anywhere else (T5), and
// its regions are moved into the coordinates of what is actually WRITTEN.
//
// `Choice.Reveal()` names the right option and the learner's pick above the
// entry, so the render's own line numbers are wrong by however many lines the
// form prepended. Located rather than counted from a formula: the form owns its
// layout, and a formula here would be a second copy of it.
func TestPlayARevealedDefinitionCarriesItsRegions(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic", "ephemeral", "quokka", "mesa")
	qs, held := questionsFor(t, d, opt)

	// A REAL screen, and the assertion is a CLICK — the joint, not the pieces.
	// editorloop_test.go records the rule: every place two separately-pinned
	// layers exchange a value across a coordinate boundary earns a row, and a row
	// is earned only when a test drives both real objects. Here those layers are
	// the loop's offset arithmetic and the screen's buffer rebasing.
	tty := &syncBuf{}
	live := newPinnedScreen(tty, 200, 80) // taller than the sitting, so no scroll
	live.interval = -1
	var errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("\r^"),
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	// The definition's own header line, which is the LAST line beginning with
	// the headword — the prompt wrote the first one. Found by reading the
	// transcript, not by a coordinate the test invented.
	word := qs[0].Word()
	row := -1
	for i, line := range strings.Split(live.Transcript(), "\n") {
		if strings.HasPrefix(unstyled(line), word) {
			row = i
		}
	}
	if row < 0 {
		t.Fatalf("the revealed definition never named %q:\n%s", word, live.Transcript())
	}

	r, ok := live.RegionAtRow(row, 0)
	if !ok {
		t.Fatalf("clicking the headword INSIDE the revealed definition (row %d) resolves to "+
			"nothing — the reveal reached the buffer without its click map, or with one "+
			"whose lines were never shifted past the option lines Choice.Reveal puts above it",
			row)
	}
	if r.Word != word {
		t.Errorf("the click at row %d answers for %q, want %q", row, r.Word, word)
	}
}

// ...and a region on a line the wrap does NOT break survives, moved to where
// that line lands (#41 BR-25, and the operator's report).
//
// The first version of this rule was all-or-nothing: drop the whole map if the
// wrap changed anything. Form 2.3's prompt is the headword, a blank, then four
// glosses — and a gloss routinely wraps — so the word the sitting is ASKING
// ABOUT lost its region on every multiple-choice question. The feature was
// inert in exactly the case it exists for, and green, because the rig ran at
// width 0 where the wrap does nothing.
func TestAWrapMovesTheClickMapRatherThanDroppingIt(t *testing.T) {
	const width = 20
	// Line 0 is a short headword; line 1 is a gloss that must wrap.
	text := "word\n1  " + strings.Repeat("gloss ", 8) + "\ntail\n"
	head := Region{Kind: RegionHeadword, Text: "word", Word: "word", Line: 0, Col: 0, Width: 4}
	inGloss := Region{Kind: RegionHeadword, Text: "gloss", Word: "gloss", Line: 1, Col: 3, Width: 5}
	onTail := Region{Kind: RegionHeadword, Text: "tail", Word: "tail", Line: 2, Col: 0, Width: 4}

	got := wrapMovedRegions(text, []Region{head, inGloss, onTail}, width)

	if len(got) != 2 {
		t.Fatalf("kept %d regions, want the two on unbroken lines: %+v", len(got), got)
	}
	if got[0].Line != 0 || got[0].Word != "word" {
		t.Errorf("the headword moved to line %d, want 0 — nothing above it wrapped", got[0].Line)
	}
	// The tail moved DOWN by however many rows the gloss became.
	rows := strings.Count(wrapWritten("1  "+strings.Repeat("gloss ", 8), width), "\n") + 1
	if want := 1 + rows; got[1].Line != want {
		t.Errorf("the tail is on line %d, want %d — a region below a wrap moves by the rows it added",
			got[1].Line, want)
	}
	// And the one INSIDE the wrapped line is gone: its column belongs to a
	// continuation now, and guessing which is the wrong-click bug.
	for _, r := range got {
		if r.Word == "gloss" {
			t.Error("a region on a line the wrap BROKE survived — its column is no longer where it says")
		}
	}
}

// THE OPERATOR'S CASE, end to end: a form-2.3 sitting at a real terminal width,
// where the prompt's glosses wrap and the headword above them must not.
func TestTheAskedWordIsClickableOnAMultipleChoiceQuestion(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic", "ephemeral", "quokka", "mesa")
	qs, held := questionsFor(t, d, opt)
	if _, ok := qs[0].(*play.Choice); !ok {
		t.Fatalf("first question is %T, want form 2.3 — this test is about a prompt that wraps", qs[0])
	}

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 200, opt.width)
	live.interval = -1
	var errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("^"),
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	word := qs[0].Word()
	row := -1
	for i, line := range strings.Split(live.Transcript(), "\n") {
		if unstyled(line) == word {
			row = i
			break
		}
	}
	if row < 0 {
		t.Fatalf("the prompt word is not on a line of its own:\n%s", live.Transcript())
	}
	r, ok := live.RegionAtRow(row, 0)
	if !ok {
		t.Fatalf("the word the sitting is ASKING ABOUT is not clickable on a multiple-choice " +
			"question — its region was dropped because the glosses below it wrapped")
	}
	if r.Word != word {
		t.Errorf("the click answers for %q, want %q", r.Word, word)
	}
}

// THE CRITICAL: a reveal written AFTER a resize still underlines its own words.
//
// `writeClickable` used to take the loop's `opt.width`, fixed at startup, while
// the screen wraps at its own `cols`, which `Resize` updates. After a narrowing
// resize the two rulers disagreed and every region landed on a line that did not
// contain its text — a headword region on a blank line, an ORIGIN region on a
// quotation. That is the wrong-click failure the whole path exists to make
// impossible, and it was possible because the wrap and the map were measured
// separately.
//
// The assertion is the invariant itself rather than a coordinate: EVERY region
// in the buffer underlines the text it names.
func TestAResizeDoesNotMisplaceTheClickMap(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic", "ephemeral", "quokka", "mesa")
	qs, held := questionsFor(t, d, opt)

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 200, opt.width)
	live.interval = -1
	var errb bytes.Buffer
	resizes := make(chan winSize, 1)
	keys := make(chan Key)
	done := make(chan int, 1)
	go func() {
		done <- playSession(t.Context(), d, opt, play.NewSession(qs), held, keys,
			console{view: live, resizes: resizes, finish: func() {}, stdout: live, stderr: &errb})
	}()

	waitFor(t, func() bool { return strings.Contains(live.Transcript(), qs[0].Word()) })
	frames := strings.Count(tty.String(), cursorHome)
	// NARROW ENOUGH that a line ABOVE the definition's headword must break —
	// Choice.Reveal puts the correct option's gloss there. At 40 that gloss
	// happens to fit for some words, and then nothing below it shifts and the
	// test discriminates nothing.
	resizes <- winSize{rows: 200, cols: 28}
	waitFor(t, func() bool { return strings.Count(tty.String(), cursorHome) > frames })

	written := len(live.Transcript())
	keys <- Key{Kind: KeyEnter} // reveal, written at the new width
	waitFor(t, func() bool { return len(live.Transcript()) > written })
	keys <- Key{Kind: KeyInterrupt}
	<-done

	lines := strings.Split(live.Transcript(), "\n")
	checked := 0
	for i := range lines {
		r, ok := live.RegionAtRow(i, 0)
		if !ok {
			continue
		}
		checked++
		if !strings.Contains(unstyled(lines[i]), r.Text) {
			t.Errorf("a region for %q is on buffer line %d, which reads %q — the map was moved "+
				"by a different width than the one the screen wrapped at", r.Text, i, unstyled(lines[i]))
		}
	}
	if checked == 0 {
		t.Fatal("no region resolved anywhere in the buffer, so this test asserts nothing")
	}
}

// A REGION ANSWERS FOR THE DECK'S KEY, not the entry's headword.
//
// `RenderOpts.Word` is identity rather than presentation, and its own doc says
// so: empty means "no click map wanted" and `regionsIn` falls back to
// `Entry.Headword()`. `--play` passed nothing until this issue, and the deck
// holds normalised keys — `jalapeno` where NOAD's head line reads `jalapeño`.
// The CDN answers different URLs for the two, so a click would have fetched a
// recording for a word the learner never looked up.
//
// The fix was one field; this is the row that makes it falsifiable. Deleting
// `Word: key` left the whole suite green.
func TestARegionAnswersForTheDeckKeyNotTheHeadword(t *testing.T) {
	d, opt, _ := playRig(t, "jalapeno")
	qs, held := questionsFor(t, d, opt)
	if len(qs) == 0 {
		t.Fatal("no questions: the fake dictionary should resolve jalapeno to the jalapeño entry")
	}

	c, ok := held.marks[store.Key("jalapeno")]
	if !ok || len(c.regions) == 0 {
		t.Fatalf("no regions kept for the deck key: %+v", held.marks)
	}
	// The premise, or this test asserts nothing: the entry's head line must
	// spell the word DIFFERENTLY from the key.
	if !strings.Contains(unstyled(c.text), "jalapeño") {
		t.Fatalf("the rendered entry does not carry the accented spelling, so key and headword "+
			"do not diverge here:\n%s", unstyled(c.text))
	}
	for _, r := range c.regions {
		if r.Word != "jalapeno" {
			t.Errorf("a region answers for %q, want the deck's key %q — the CDN has different "+
				"URLs for the two, so a click would fetch a recording for a word the learner "+
				"never looked up", r.Word, "jalapeno")
		}
	}
}

// DONE-WHEN 9: SIGWINCH repaints mid-sitting.
func TestPlayRepaintsOnResize(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	qs, held := questionsFor(t, d, opt)

	out := &syncBuf{}
	var errb bytes.Buffer
	view := paintInto(out)
	resizes := make(chan winSize, 1)
	// Sent BEFORE any key, and the keys channel is unbuffered — otherwise both
	// cases are ready at once and select picks uniformly at random, so the
	// interrupt would sometimes end the sitting before the resize was read.
	resizes <- winSize{rows: 12, cols: 40}
	keys := make(chan Key)
	done := make(chan int, 1)
	go func() {
		done <- playSession(t.Context(), d, opt, play.NewSession(qs), held, keys,
			console{view: view, resizes: resizes, finish: func() {}, stdout: view, stderr: &errb})
	}()

	waitFor(t, func() bool { _, _, rows := view.scrolls(); return len(rows) == 1 })
	keys <- Key{Kind: KeyInterrupt}
	<-done

	if _, _, rows := view.scrolls(); !slices.Equal(rows, []int{12}) {
		t.Errorf("the loop resized to %v rows, want [12] — a frame drawn for the old shape "+
			"scrolls the terminal, and every row the sitting placed moves with it", rows)
	}
	// And it REDREW. Resize deliberately does not paint (the live edge is
	// rendered against the new width too), so a loop that resizes without
	// drawing leaves the old frame on a differently shaped screen.
	if n := len(view.drawnMenus()); n < 2 {
		t.Errorf("%d frames drawn, want the initial one and the one after the resize", n)
	}
}

// Audio plays before reveal by DEFAULT, which is the Spec and which no test
// exercised: playRig sets noAudio, so the whole playback branch was at zero
// coverage. fakePlayer is the seam — playAnnounced shells out to afplay(1), so a
// real player in a test is a real process.
func TestRevealPlaysThePronunciationByDefault(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	player := audible(&d, &opt) // the default; playRig turns audio off for every other test
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("\r^"), playbackConsole(&out, &errb))

	if len(player.Played) == 0 {
		t.Error("revealing did not play the pronunciation, and audio is on by default")
	}
}

// ...and -no-audio silences it.
func TestNoAudioSilencesTheSession(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	player := audible(&d, &opt)
	opt.noAudio = true // ...and THEN silence it: the flag, not an unreachable source
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("\r^"), playbackConsole(&out, &errb))

	if len(player.Played) != 0 {
		t.Errorf("-no-audio played %v", player.Played)
	}
}

// -count bounds the sitting, and nothing exercised it either.
func TestCountBoundsTheSession(t *testing.T) {
	// All three must be real fixtures: a word the fake dictionary lacks is
	// SKIPPED by todaysQuestions, so the count would be bounded by the corpus
	// rather than by the flag — which is what the first version of this test
	// measured, and it read as the flag working when it was not exercised.
	d, opt, _ := playRig(t, "sycophantic", "ephemeral", "defenestrate")
	opt.count = 2

	qs, _ := questionsFor(t, d, opt)

	if len(qs) != 2 {
		t.Errorf("got %d questions with -count 2, want 2", len(qs))
	}
}

// The drop key, end to end: the deck loses the word and the EVENTS keep it —
// --forget's contract, because history is what happened and cannot be untrue
// while the deck is the working set the learner curates.
func TestDropRemovesFromDeckButKeepsEvents(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic", "ephemeral")
	// The lookup that CREATED the deck entry, which is the history the drop must
	// not erase. playRig upserts words without events, so asserting preservation
	// without this would range over an empty log and pass vacuously — which is
	// how the first version of this test failed for the wrong reason.
	if err := st.AppendEvent(store.ReviewEvent{
		Word: "ephemeral", Kind: store.EventLookedUp, Found: true, At: aDay,
	}); err != nil {
		t.Fatal(err)
	}
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	// Drop the first word, then quit.
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("d^"), playbackConsole(&out, &errb))

	deck, err := st.Deck()
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range deck {
		if store.Key(w.Text) == "ephemeral" {
			t.Error("the dropped word is still in the deck")
		}
	}
	if len(deck) != 1 {
		t.Errorf("deck has %d words, want 1 remaining", len(deck))
	}
	// The lookup that created it is still history.
	all, _ := st.Events(time.Time{})
	found := false
	for _, e := range all {
		if e.Word == "ephemeral" {
			found = true
		}
	}
	if !found {
		t.Error("dropping erased the word's history — that is --forget's contract broken")
	}
	if !strings.Contains(out.String(), "removed") {
		t.Errorf("stdout = %q, want it to say what was removed", out.String())
	}
}

// Dropping is not an assessment.
func TestDropRecordsNoReview(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	spy := &countingCapturer{}
	d.capture = spy
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("d"), playbackConsole(&out, &errb))

	if spy.reviews != 0 {
		t.Errorf("dropping recorded %d reviews", spy.reviews)
	}
}

// A mode plus a word is two commands on one line. --play needed this guard MORE
// than --reflect does: it writes events, so honouring one of the two would
// change state under a misread intent.
//
// The first version dispatched --play ABOVE the switch carrying that rule, so it
// could never reach it — with the comment stating the rule three lines above the
// dispatch that broke it.
func TestPlayWithAWordIsAUsageError(t *testing.T) {
	var out, errb bytes.Buffer

	code := run(t.Context(), []string{"--play", "sycophantic"}, testDeps(t),
		strings.NewReader(""), &out, &errb)

	if code != 2 {
		t.Errorf("exit = %d, want 2 — a mode plus a word is a usage error", code)
	}
	if !strings.Contains(errb.String(), "do not also pass a word") {
		t.Errorf("stderr = %q, want it to say why", errb.String())
	}
}

// okAudio always has a recording, so the playback branch is reachable. The
// package's other double, noAudioSource, always returns ErrNoAudio — which is
// why the branch was at zero coverage.
type okAudio struct{}

func (okAudio) Fetch(context.Context, []string) ([]byte, string, error) {
	return []byte("mp3"), "https://example.invalid/word.mp3", nil
}

// A deck whose words the dictionary no longer knows is NOT "nothing due today".
//
// Words WERE due; every one failed to look up. Saying nothing is due would send
// the learner away believing their deck is clear when the dictionary is the
// problem — and that branch was at coverage 0 when the fix landed.
func TestAllLookupsFailingIsNotNothingDue(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	// NOT refusingDict — that double FAILS the test when consulted, because it
	// exists to prove a command never reaches the dictionary. Here the dictionary
	// is supposed to be consulted and supposed to say no.
	d.dict = missingDict{}

	var out, errb bytes.Buffer
	qs, _, code := todaysQuestions(d, opt, &out, &errb)

	if len(qs) != 0 {
		t.Fatalf("got %d questions from a dictionary that refuses everything", len(qs))
	}
	if code == 0 {
		t.Error("exit 0 — a deck that cannot be looked up is not a clear deck")
	}
	if strings.Contains(out.String(), "nothing due") {
		t.Errorf("stdout claims nothing is due: %q", out.String())
	}
	if !strings.Contains(errb.String(), "due but none could be looked up") {
		t.Errorf("stderr = %q, want it to say what actually happened", errb.String())
	}
}

// A MISS IS RECORDED BEFORE IT PLAYS, and the pin had to be rebuilt.
//
// Apply emits {Record, Reveal} and session.go calls that order load-bearing: the
// record is written before anything that can block on the terminal. The old pin
// was TestLosingTheTerminalAfterPlaybackExitsOne, which drove a miss into a
// terminal that could not be re-entered — reversing the loop's iteration lost
// the verdict AND exited 1.
//
// #41 D5a DELETES that branch, and #38's plan (PQ-11, round 4) had already
// worked out what the replacement has to be. It must OBSERVE THE ORDER, not a
// consequence of it: a draft that asserted "the record survives a failed
// playback" measured GREEN under a reversed iteration, correctly — once the
// early return is gone, both orders write the record, so the consequence stops
// discriminating. The old test worked only because the reveal arm could abort.
//
// So both sides append to ONE ordered log, and the assertion is on the sequence.
// Falsifiable by reversing the outs iteration and by nothing else.
func TestAMissRecordsBeforeItPlays(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	audible(&d, &opt)
	seq := &orderLog{}
	d.capture = &loggingCapturer{seq: seq}
	d.player = &loggingPlayer{seq: seq}
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	// A MISS, not a peek — only a miss owes two outcomes, so only a miss has an
	// order to get wrong.
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("n"), playbackConsole(&out, &errb))

	if got := seq.first(2); len(got) < 2 || got[0] != "record" || got[1] != "play" {
		t.Errorf("the sitting did %v, want the record first — a verdict written after the "+
			"reveal is a verdict lost to anything that goes wrong while the answer is up", got)
	}
}

// orderLog is one ordered record of things that must happen in a fixed order,
// written by every seam that participates.
//
// ONE log rather than a timestamp on each double: two clocks can tie, and an
// assertion over two independent recordings is an assertion about how they were
// read as much as about what happened.
type orderLog struct {
	mu   sync.Mutex
	seen []string
}

func (l *orderLog) add(what string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.seen = append(l.seen, what)
}

func (l *orderLog) first(n int) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.seen[:min(n, len(l.seen))]
}

type loggingCapturer struct {
	countingCapturer
	seq *orderLog
}

func (c *loggingCapturer) CaptureReview(out play.Outcome, opt options) {
	c.seq.add("record")
	c.countingCapturer.CaptureReview(out, opt)
}

type loggingPlayer struct {
	fakePlayer
	seq *orderLog
}

func (p *loggingPlayer) Play(ctx context.Context, path string) error {
	p.seq.add("play")
	return p.fakePlayer.Play(ctx, path)
}

// missingDict answers every lookup with "no entry" — a deck whose words the
// dictionary no longer knows.
//
// Not fakeDictionary with an out-of-corpus word, which returns ErrNoEntry too
// (BR-43). That would make "every lookup fails" depend on a chosen word being
// ABSENT from testdata/entries, so adding that word to the corpus later would
// silently turn this into a test of something else. #6's -count test had exactly
// that fault — it used obsequious, and was bounded by the corpus rather than by
// the flag it named. A double that states the property outright cannot drift
// with a fixture.
type missingDict struct{}

func (missingDict) Lookup(word string) (string, error) {
	return "", ErrNoEntry
}

// Each empty sitting names its OWN cause (BR-46).
//
// Table-driven because the defect was one message serving three situations; the
// table is what makes "three situations, three messages" checkable rather than
// asserted.
func TestEmptyQueueNamesItsCause(t *testing.T) {
	for _, tc := range []struct {
		name     string
		deckSize int
		budget   int
		want     string
	}{
		{"an empty deck is not a clear schedule", 0, 20, "the deck is empty"},
		{"a zero budget is not a clear schedule", 5, 0, "-count 0"},
		{"and the schedule speaks for itself", 5, 20, "nothing due today"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := emptyQueueReason(tc.deckSize, tc.budget)
			if !strings.Contains(got, tc.want) {
				t.Errorf("emptyQueueReason(%d, %d) = %q, want it to mention %q",
					tc.deckSize, tc.budget, got, tc.want)
			}
			// The reserved sentence belongs to exactly one row.
			if said := strings.Contains(got, "nothing due today"); said != (tc.want == "nothing due today") {
				t.Errorf("emptyQueueReason(%d, %d) = %q — \"nothing due today\" is reserved "+
					"for the schedule genuinely having nothing", tc.deckSize, tc.budget, got)
			}
		})
	}
}

// Every claim in this file's neighbourhood that had NO named test, pinned
// together (BR-48).
//
// The enumeration these came from was built by running `go tool cover` over the
// close window and reading the zero-count blocks, not by remembering what had
// been written — which is how it stayed at "five sites" for four rounds while
// the true count grew.
func TestClaimsWithoutTestsUntilNow(t *testing.T) {
	// toInput — space reveals. The README's key table and draw() both promise
	// this to the learner, and nothing asserted it: Enter was covered, space was
	// not.
	//
	// Cited by NAME, not by line. The original comment said "play_loop.go:182"
	// and "README:54"; the reversal that this issue shipped moved both, so the
	// citations pointed at a `case` that had moved and at a table header (BR-1).
	// A line number in a comment is a restatement of a fact the file owns, and
	// it drifts exactly like the doc prose in the same family — with no build to
	// catch it, since a comment cannot be wrong enough to fail.
	t.Run("space reveals, like Enter", func(t *testing.T) {
		got, ok := toInput(Key{Kind: KeyRune, Rune: ' '})
		if !ok || got.Kind != play.InputReveal {
			t.Errorf("space produced (%+v, %v), want an InputReveal — the README key table promises it", got, ok)
		}
	})

	// The -count guard in main.go, added for BR-46 and itself shipped unpinned, which is
	// what made BR-48 the sixth in its family rather than the fifth.
	t.Run("-count rejects a negative", func(t *testing.T) {
		d, _, _ := playRig(t)
		var out, errb bytes.Buffer
		code := run(t.Context(), []string{"-no-audio", "-count", "-1", "--play"},
			d, strings.NewReader(""), &out, &errb)
		if code != 2 {
			t.Errorf("exit = %d, want 2 — a negative budget is a typo, like -times and -sound", code)
		}
		if !strings.Contains(errb.String(), "-count must not be negative") {
			t.Errorf("stderr = %q, want it to name the flag", errb.String())
		}
	})

	// main.go:425 — the --play dispatch itself. BR-14 fixed the argument guard
	// here and the dispatch stayed uncovered, so "define --play" reaching the
	// review loop at all rested on nothing.
	t.Run("--play reaches the review loop", func(t *testing.T) {
		d, _, _ := playRig(t, "sycophantic")
		var out, errb bytes.Buffer
		code := run(t.Context(), []string{"-no-audio", "--play"},
			d, strings.NewReader(""), &out, &errb)
		// The TERMINAL REFUSAL is the observable, and it is a better one than
		// the empty-deck message this used to read: that message is
		// todaysQuestions', while only runPlay writes this. A test has no
		// terminal, so a sitting that dispatched correctly says so and stops
		// (BR-3) — and one that never dispatched says nothing at all.
		if code != 1 {
			t.Fatalf("exit = %d, want 1; stderr=%q", code, errb.String())
		}
		if !strings.Contains(errb.String(), "--play needs a terminal") {
			t.Errorf("stderr = %q — --play did not reach runPlay", errb.String())
		}
	})
}

// A correct answer costs one keystroke and plays NOTHING.
//
// The pronunciation is what a miss earns; playing it on a hit is the step #24
// removes. Audio is enabled here, so this is about the FLOW and not the flag.
func TestCorrectAnswerPlaysNoAudio(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic")
	// audible() installs a source that HAS a recording as well as the player,
	// so this negative assertion is about the FLOW: without a reachable source
	// "played nothing" would be true whatever the session did (PQ-6).
	fp := audible(&d, &opt)
	opt.times = 1

	qs, held := questionsFor(t, d, opt)
	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("y"), playbackConsole(&out, &errb))

	if len(fp.Played) != 0 {
		t.Errorf("played %v for a word the learner got right", fp.Played)
	}
	if len(reviewEvents(t, st)) != 1 {
		t.Error("the answer was not recorded")
	}
}

// A miss plays the pronunciation AND records the miss — both halves of the
// slice, pinned by one test.
//
// The record half was unpinned until BR-8: mutating the loop to
// `outs[len(outs)-1:]` drops every OutcomeRecord, so no miss reaches events/ or
// the schedule, and the WHOLE suite stayed green. The pty test cannot see it
// either — its "0 right, 1 wrong" comes from the session tally that score() sets
// inside Apply, which the loop never touches. My own mutation table ran the
// mirror (`outs[:1]`, dropping the reveal) and not this one; a slice has two
// ends and only one was tested.
func TestAMissPlaysThePronunciationAndRecordsIt(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic")
	fp := audible(&d, &opt)
	opt.times = 1

	qs, held := questionsFor(t, d, opt)
	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("n^"), playbackConsole(&out, &errb))

	if len(fp.Played) == 0 {
		t.Error("a miss played nothing; the definition it earns includes hearing it")
	}
	evs := reviewEvents(t, st)
	if len(evs) != 1 {
		t.Fatalf("got %d events, want the one miss — a verdict that never reaches the log "+
			"never reaches the schedule either", len(evs))
	}
	if evs[0].Correct {
		t.Error("the miss was recorded as correct")
	}
}

// Three states, three prompts.
func TestThePromptSaysWhatTheKeysDo(t *testing.T) {
	q := play.NewRecall("sycophantic", "a definition")
	for _, tc := range []struct {
		name, want, absent string
		s                  play.Session
	}{
		{"unrevealed: the grading keys, straight away", "y = got it, n = missed it", "to reveal",
			play.Session{Questions: []play.Question{q}}},
		{"peeked: still grading", "y = got it, n = missed it", "any key",
			play.Session{Questions: []play.Question{q}, Revealed: true}},
		{"missed: the answer is up, move on", "any key = next word", "y = got it",
			play.Session{Questions: []play.Question{q}, Revealed: true, Graded: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := livePrompt(tc.s)
			if !strings.Contains(got, tc.want) {
				t.Errorf("prompt = %q, want it to contain %q", got, tc.want)
			}
			if strings.Contains(got, tc.absent) {
				t.Errorf("prompt = %q, must NOT offer %q in this state", got, tc.absent)
			}
		})
	}
}

// Done-when 2: the CHOSEN option reaches the log, not just right/wrong.
//
// Driven through playSession rather than by calling CaptureReview, because the
// claim is about a wiring only the loop supplies (lessons.md, #15): deleting
// `out.Axis` from the loop's call leaves every play package test green, since
// Apply would still be putting the axis on the Outcome nobody read.
func TestAMissRecordsTheAxisItChose(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	opts := []play.Option{
		{Gloss: "behaving in an obsequious way", Correct: true},
		{Gloss: "an official report of proceedings", Word: "record", Axis: play.AxisDomain},
		{Gloss: "a manservant or valet", Word: "man", Axis: play.AxisRegister},
	}
	for _, tc := range []struct {
		name string
		key  string
		want play.Axis
	}{
		{"picking the domain distractor", "2", play.AxisDomain},
		{"picking the register distractor", "3", play.AxisRegister},
		// D8: a right answer writes no axis, so the taxonomy has nothing to
		// filter back out later.
		{"answering correctly", "1", play.AxisNone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spy := &countingCapturer{}
			d.capture = spy
			q := play.NewChoice("sycophantic", "", opts)

			var out, errb bytes.Buffer
			playSession(t.Context(), d, opt, play.NewSession([]play.Question{q}), &sittingDeck{}, keysFor(tc.key), playbackConsole(&out, &errb))

			if spy.reviews != 1 {
				t.Fatalf("CaptureReview called %d times, want 1", spy.reviews)
			}
			if got := spy.axes[0]; got != tc.want {
				t.Errorf("recorded axis %v (%q), want %v — the axis never left the form",
					got, got.String(), tc.want)
			}
		})
	}
}

// D9: a young deck falls back to form 2.1, invisibly.
//
// The learner three lookups in is the NORMAL early state of this tool, not an
// edge case — and it is the state every new user is in, so a session that broke
// here would break on first use.
func TestASittingFallsBackToRecall(t *testing.T) {
	for _, tc := range []struct {
		name     string
		deck     []string
		wantKind string
	}{
		// One word: its own entry is the only thing in the pool, and a word is
		// never its own distractor, so there is nothing to choose between.
		{"a one-word deck", []string{"sycophantic"}, "*play.Recall"},
		{"two words", []string{"sycophantic", "ephemeral"}, "*play.Choice"},
		{"a fuller deck", []string{"sycophantic", "ephemeral", "quokka", "mesa", "parrot"}, "*play.Choice"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, opt, _ := playRig(t, tc.deck...)
			qs, held := questionsFor(t, d, opt)
			if len(qs) == 0 {
				t.Fatal("no questions")
			}
			if got := typeName(qs[0]); got != tc.wantKind {
				t.Errorf("first question is %s, want %s", got, tc.wantKind)
			}
			// Whichever form it is, the sitting must be answerable — the
			// fallback is only invisible if it actually works.
			var out, errb bytes.Buffer
			playSession(t.Context(), d, opt, play.NewSession(qs[:1]), held,
				keysFor("\r"+gradeKey(t, qs[0], play.Correct)), playbackConsole(&out, &errb))
			if out.Len() == 0 {
				t.Error("the session drew nothing")
			}
		})
	}
}

// A four-option question needs four sources, and the option count must GROW
// with the deck rather than sitting at two forever.
func TestOptionCountGrowsWithTheDeck(t *testing.T) {
	for _, tc := range []struct{ deck, wantOptions int }{
		{2, 2}, {3, 3}, {4, 4}, {8, 4},
	} {
		words := []string{"sycophantic", "ephemeral", "quokka", "mesa", "parrot", "concrete", "pulp", "minute"}[:tc.deck]
		d, opt, _ := playRig(t, words...)
		qs, _ := questionsFor(t, d, opt)
		c, ok := qs[0].(*play.Choice)
		if !ok {
			t.Fatalf("a deck of %d gave %s, want form 2.3", tc.deck, typeName(qs[0]))
		}
		if got := len(c.Options()); got != tc.wantOptions {
			t.Errorf("a deck of %d gave %d options, want %d", tc.deck, got, tc.wantOptions)
		}
	}
}

// Done-when 5: the whole sitting runs with no model and no network.
//
// The model seam is made to PANIC rather than left nil: nil would make this pass
// on a loop that reaches for the model behind a `!= nil` guard, which is exactly
// how a network dependency creeps into an offline path unnoticed.
func TestSittingWithNoModelAndNoNetwork(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic", "ephemeral", "quokka", "mesa")
	d.newLLM = func(llm.Config) llm.Client {
		panic("form 2.3 reached the model seam; this form is offline by design")
	}
	d.getenv = func(string) string {
		panic("form 2.3 read the environment for a credential")
	}
	// And no audio source at all, so nothing can reach the CDN either.
	d.audio = noAudioSource{}

	qs, held := questionsFor(t, d, opt)
	if len(qs) == 0 {
		t.Fatal("no questions")
	}
	var keys string
	for _, q := range qs {
		keys += "\r" + gradeKey(t, q, play.Correct)
	}
	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor(keys), playbackConsole(&out, &errb))

	if got := len(reviewEvents(t, st)); got != len(qs) {
		t.Errorf("%d events for %d questions — an offline sitting must still record", got, len(qs))
	}
}

func typeName(v any) string {
	switch v.(type) {
	case *play.Choice:
		return "*play.Choice"
	case *play.Recall:
		return "*play.Recall"
	}
	return "unknown"
}

// The two ways an ENTRY (rather than the deck) sends a word to form 2.1.
//
// Both are properties of what the dictionary returned, so the deck here is large
// enough that every other word gets form 2.3 — which is what isolates the entry
// as the cause rather than the deck size.
func TestASittingFallsBackForAnEntryThatCannotBeAsked(t *testing.T) {
	for _, tc := range []struct{ word, why string }{
		// Every sense is a cross-reference: "plural form of base1",
		// "/ˈbāsēz/ plural form of basis". No definition to be the answer.
		{"bases", "every sense is a cross-reference"},
		// NOAD redirects the derived form: looking this up returns the `bargain`
		// entry, so the "correct" option would be a different word's meaning
		// offered as this one's, recorded Correct, and promoted in the schedule.
		{"bargainer", "the entry defines the base word, not this one"},
	} {
		t.Run(tc.word, func(t *testing.T) {
			d, opt, _ := playRig(t, tc.word, "sycophantic", "quokka", "mesa", "parrot", "concrete")
			qs, _ := questionsFor(t, d, opt)

			var form, otherForms string
			for _, q := range qs {
				if q.Word() == tc.word {
					form = typeName(q)
				} else if otherForms == "" {
					otherForms = typeName(q)
				}
			}
			if form == "" {
				t.Fatalf("%q was dropped from the sitting entirely; %d questions", tc.word, len(qs))
			}
			if form != "*play.Recall" {
				t.Errorf("%q got %s — %s, so it cannot be a recognition question", tc.word, form, tc.why)
			}
			if otherForms != "*play.Choice" {
				t.Errorf("the rest of the deck got %s, so this test is not distinguishing "+
					"the entry from the deck size", otherForms)
			}
		})
	}
}

// Done-when 10: an unaided answer reaches the EVENT LOG, not just the Outcome.
//
// Driven through playSession rather than by calling CaptureReview, because the
// wiring is the claim: deleting `out.Unaided` from the capturer leaves every
// `play` test green, since Apply would still be setting a field nobody read.
func TestUnaidedAnswerReachesTheLog(t *testing.T) {
	opts := []play.Option{
		{Gloss: "behaving in an obsequious way", Correct: true},
		{Gloss: "an isolated flat-topped hill", Word: "mesa", Axis: play.AxisGeneral},
	}
	for _, tc := range []struct {
		name        string
		keys        string
		wantUnaided bool
	}{
		// Answered cold: the form checked it and no reveal preceded it.
		{"answered cold", "1", true},
		// Revealed FIRST, then answered correctly. Still Correct, never unaided
		// — and this is the case that catches reading the flag after advance()
		// has zeroed s.Revealed.
		{"revealed, then answered", "\r1", false},
		// A wrong answer is never unaided whatever preceded it.
		{"answered wrongly", "2", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, opt, st := playRig(t, "sycophantic")
			var out, errb bytes.Buffer
			q := play.NewChoice("sycophantic", "", opts)
			playSession(t.Context(), d, opt, play.NewSession([]play.Question{q}), &sittingDeck{}, keysFor(tc.keys), playbackConsole(&out, &errb))

			events := reviewEvents(t, st)
			if len(events) != 1 {
				t.Fatalf("%d review events, want 1", len(events))
			}
			if events[0].Unaided != tc.wantUnaided {
				t.Errorf("event.Unaided = %v, want %v", events[0].Unaided, tc.wantUnaided)
			}
		})
	}
}

// And form 2.1's `y` never earns it, however fast — one form to the left of the
// board, and the same overconfidence.
func TestRecallNeverRecordsUnaided(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic")
	var out, errb bytes.Buffer
	q := play.NewRecall("sycophantic", "the definition")
	playSession(t.Context(), d, opt, play.NewSession([]play.Question{q}), &sittingDeck{}, keysFor("y"), playbackConsole(&out, &errb))

	events := reviewEvents(t, st)
	if len(events) != 1 {
		t.Fatalf("%d review events, want 1", len(events))
	}
	if !events[0].Correct {
		t.Fatal("the answer was not recorded as correct")
	}
	if events[0].Unaided {
		t.Error("form 2.1's `y` recorded as unaided — it is the learner's claim that " +
			"they knew it, and nothing checked")
	}
}

// Done-when 12: the sitting reports what the deck costs, and names the budget
// it assumed.
//
// Without a reader, DailyLoad would be a function nobody calls — the same "no
// reader" smell that got Progress.Streak deleted in this very issue.
func TestFinishReportsTheLoad(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic", "ephemeral", "quokka", "mesa")
	qs, held := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs[:1]), held,
		keysFor("\r"+gradeKey(t, qs[0], play.Correct)), playbackConsole(&out, &errb))

	got := out.String()
	if !strings.Contains(got, "reviews/day") {
		t.Errorf("the summary does not report the deck's daily cost:\n%s", got)
	}
	if !strings.Contains(got, "new words/day") && !strings.Contains(got, "no room for new words") {
		t.Errorf("the summary does not report the sustainable new-word rate:\n%s", got)
	}
	// The budget it assumed must be NAMED, or a learner who sits twice a day has
	// no way to know the number is per-sitting.
	if !strings.Contains(got, "a sitting") {
		t.Errorf("the summary does not say which budget it assumed:\n%s", got)
	}
	// And the number must be real: a four-word deck of unreviewed words costs
	// about four reviews a day, not zero.
	if strings.Contains(got, "~0 reviews/day") {
		t.Errorf("the load reports zero for a four-word deck — the deck is not being read:\n%s", got)
	}
}

// ENTER IS ITS OWN KIND NOW, AND TAB REACHES play AT ALL (#40 D13, D14).
//
// Both rows are about the same seam: toInput is where main.Key stops and a
// session intent begins, and both of these were wrong there rather than deeper
// in. Enter was merged with space, which would have fired a board's commit on
// the most careless key there is; Tab was decoded by key.go and dropped on the
// floor because nothing here had a case for it.
func TestToInputSplitsEnterFromSpaceAndCarriesTab(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  Key
		want play.InputKind
	}{
		{"Enter finishes", Key{Kind: KeyEnter}, play.InputFinish},
		{"space reveals", Key{Kind: KeyRune, Rune: ' '}, play.InputReveal},
		{"Tab toggles", Key{Kind: KeyTab}, play.InputToggle},
		{"d drops, before any form sees it", Key{Kind: KeyRune, Rune: 'd'}, play.InputDrop},
		{"Ctrl-C quits", Key{Kind: KeyInterrupt}, play.InputQuit},
		{"anything else is the form's", Key{Kind: KeyRune, Rune: 'y'}, play.InputRune},
	} {
		got, ok := toInput(tc.key)
		if !ok || got.Kind != tc.want {
			t.Errorf("%s: toInput(%+v) = (%+v, %v), want kind %v", tc.name, tc.key, got, ok, tc.want)
		}
	}
}

// AND THE SPLIT IS INVISIBLE TO EVERY FORM THAT HOLDS ONE WORD, which is the
// property that let it ship without touching 2.1 or 2.3. Apply treats
// InputFinish exactly as InputReveal for them, so Enter still reveals.
func TestEnterStillRevealsOnASingleWordForm(t *testing.T) {
	for _, q := range []play.Question{
		play.NewRecall("keel", "the bottom of a ship"),
		play.NewChoice("keel", "", []play.Option{{Gloss: "the bottom of a ship", Correct: true}, {Gloss: "a flat-topped hill"}}),
	} {
		s := play.NewSession([]play.Question{q})
		in, _ := toInput(Key{Kind: KeyEnter})
		s, outs := play.Apply(s, in)
		if !s.Revealed {
			t.Errorf("%T: Enter did not reveal", q)
		}
		if len(outs) != 1 || outs[0].Kind != play.OutcomeReveal {
			t.Errorf("%T: Enter produced %+v, want an OutcomeReveal", q, outs)
		}
	}
}

// THE LOOP'S HALF OF THE CLICK (#40 D11, T5): the subtraction between two
// answers it is not qualified to give itself.
//
// The screen says which footer entry the pointer was on; the form says which
// cell is at that spot. The loop only knows that the grid is drawn as the FIRST
// footer entries, so a row at or past Rows() is the toggle or the bar rather
// than a word. The end-to-end join — that the grid really is drawn there — is
// TestAClickOnABoardMarksIt on the real screen.
func boardCells(words ...string) []play.Cell {
	cs := make([]play.Cell, len(words))
	for i, w := range words {
		cs[i] = play.Cell{Word: w}
	}
	return cs
}

func TestFormCellAsksTheScreenAndTheForm(t *testing.T) {
	board := play.NewBoard(boardCells("keel", "mesa", "run", "bank", "set"), 80, play.Palette{})
	// Five words at four columns: two grid rows, then the blank and the panel.
	// (R11 moved the toggle to the prompt row.)
	if board.Rows() != 4 {
		t.Fatalf("expected a four-row live edge, got %d:\n%s", board.Rows(), board.Prompt())
	}

	// The gutter column, derived from what Prompt DREW rather than from the
	// board's arithmetic: one column left of where the second cell starts.
	firstLine := strings.Split(board.Prompt(), "\n")[0]
	gutter := strings.Index(firstLine, "[1] ") - 1
	if gutter < 1 {
		t.Fatalf("could not find the second cell in %q", firstLine)
	}

	view := paintInto(io.Discard)
	view.footerAt(7, 0) // the grid's first row
	view.footerAt(8, 1) // its second
	view.footerAt(9, 2) // the toggle, which is not a cell
	// A CONTINUATION ROW: the same entry, drawn a second time because it was too
	// wide for the terminal. Its columns are not in the entry's own space (R9).
	view.footerAtOffset(10, 0, 1)

	for _, tc := range []struct {
		name     string
		q        play.Question
		row, col int
		want     int
		wantOK   bool
	}{
		{"the first grid row", board, 7, 0, 0, true},
		{"the second grid row", board, 8, 0, 4, true},
		{"a gutter is not a cell", board, 7, gutter, 0, false},
		{"the toggle row is not a cell", board, 9, 0, 0, false},
		{"a row the screen does not place", board, 3, 0, 0, false},
		// The board keeps its rows fitting by relaying out, so this should never
		// arise — which is why the loop refuses it rather than trusting that.
		// Column 0 of a continuation is column `cols` of the line, and acting on
		// it lands a permanent mark on whatever word sits at column 0.
		{"a wrapped entry's continuation row", board, 10, 0, 0, false},
		{"a form that is not a grid", play.NewRecall("keel", "d"), 7, 0, 0, false},
		{"no form at all", nil, 7, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := formCell(view, tc.q, Key{Kind: KeyClick, Row: tc.row, Col: tc.col})
			if ok != tc.wantOK || (ok && got != tc.want) {
				t.Errorf("formCell = (%d, %v), want (%d, %v)", got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

// DONE-WHEN 7, THE END TO END: A CLICK MARKS ON A BOARD (#40 D11, T5+T6).
//
// The two halves are pinned separately — formCell's subtraction against a
// scripted screen, and FooterRowAt's arithmetic on a real one — and the object
// that JOINS them is this loop. #30's rule applies: a double may not stand in
// for the joining object, so this drives a real pinned screen with a real board
// and asserts the mark reached the log.
//
// The discriminator is the EVENT. A loop that did not offer the click to the
// form would send it to playRegion instead, which records nothing at all.
func TestAClickOnABoardMarksIt(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic", "ephemeral")
	_, held := questionsFor(t, d, opt)
	board := play.NewBoard(boardCells("sycophantic", "ephemeral"), opt.width, play.Palette{})
	// One grid row, then the blank and the panel (R11 moved the toggle out).
	if board.Rows() != 3 {
		t.Fatalf("expected a three-row live edge, got %d rows:\n%s", board.Rows(), board.Prompt())
	}

	// TEN ROWS, so the geometry is arithmetic rather than a guess. Pinned, the
	// footer sits at the bottom edge: four entries (the board's own three, plus
	// the bar) means the grid's only row is viewport row 10-4 = 6.
	//
	// The board also writes ONE blank line into the buffer as it opens, which
	// separates it from the previous question — the operator's fourth note from
	// a real sitting. It is a buffer line, so it does not move the footer.
	const termRows, gridRow = 10, 6
	// The second cell's column, read off what Prompt DREW rather than computed.
	col := strings.Index(board.Prompt(), "[1] ") + len("[1] ")
	if col < 1 {
		t.Fatalf("no second cell in %q", board.Prompt())
	}

	tty := &syncBuf{}
	live := newPinnedScreen(tty, termRows, opt.width)
	live.interval = -1
	var errb bytes.Buffer

	keys := make(chan Key, 2)
	keys <- Key{Kind: KeyClick, Row: gridRow, Col: col}
	keys <- Key{Kind: KeyInterrupt}
	close(keys)

	playSession(t.Context(), d, opt, play.NewSession([]play.Question{board}), held, keys,
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	// THE PREMISE, checked against the PAINT rather than assumed: the grid really
	// was drawn on the row that was clicked, and the click's column really was
	// inside the second cell. Without this the assertions below would pass for a
	// sitting whose footer was somewhere else entirely.
	//
	// Read off the frame, not off FooterRowAt — the click map and the paint are
	// the two things that have to agree, so a premise taken from one of them
	// could not catch the two disagreeing.
	rows := paintedRowsShowing(t, tty.String(), "[0] sycophantic")
	if len(rows) <= gridRow {
		t.Fatalf("the first frame is %d rows, want the grid at row %d:\n%s", len(rows), gridRow, strings.Join(rows, "\n"))
	}
	if !strings.HasPrefix(rows[gridRow], "[0] sycophantic") {
		t.Fatalf("viewport row %d is %q, want the grid's first row", gridRow, rows[gridRow])
	}
	if !strings.HasPrefix(rows[gridRow][col:], "ephemeral") {
		t.Fatalf("column %d of the grid row is %q, want the second cell's word", col, rows[gridRow][col:])
	}

	// AND THE MARK LANDED, on the word that was actually under the pointer.
	evs := reviewEvents(t, st)
	if len(evs) != 1 {
		t.Fatalf("%d review events, want the one click:\n%s", len(evs), unstyled(tty.String()))
	}
	if evs[0].Word != "ephemeral" {
		t.Errorf("the click recorded %q, want the word it landed on", evs[0].Word)
	}
	if board.Spent() {
		t.Error("one click spent a board of two cells")
	}
}

// paintedRowsShowing is the first frame that DREW want, split into rows.
//
// By content rather than by index, because a board writes a blank buffer line as
// it opens — separating it from the previous question — and that write paints a
// frame of its own before the grid is drawn. A test that indexed frames counted
// that one and read the wrong screen.
func paintedRowsShowing(t *testing.T, out, want string) []string {
	t.Helper()
	for i, f := range strings.Split(out, cursorHome+eraseDown) {
		if i == 0 {
			continue // whatever preceded the first frame
		}
		if rows := strings.Split(unstyled(f), "\r\n"); strings.Contains(unstyled(f), want) {
			return rows
		}
	}
	t.Fatalf("no frame drew %q:\n%s", want, out)
	return nil
}

// paintedRows is frame n of a session's output, split into the rows the terminal
// placed.
//
// A frame begins at cursorHome+eraseDown, and its rows are separated by the
// CRLFs Paint writes. The last row carries the cursor walk-back and the prompt
// reprint appended to it, which is why callers match on a PREFIX.
func paintedRows(t *testing.T, out string, n int) []string {
	t.Helper()
	frames := strings.Split(out, cursorHome+eraseDown)[1:] // [0] is whatever preceded the first
	if len(frames) <= n {
		t.Fatalf("output holds %d frames, want at least %d:\n%s", len(frames), n+1, out)
	}
	return strings.Split(unstyled(frames[n]), "\r\n")
}

// A BOARD IS THE LIVE EDGE, so it is NOT in the transcript (#41 D4, #40 D10).
//
// The buffer is what survives the sitting, and a grid whose marks changed in
// place could never have been written there. The relearn line is what the
// transcript gets instead (T9).
func TestABoardIsDrawnInTheFooterAndNotTheBuffer(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic", "ephemeral")
	_, held := questionsFor(t, d, opt)
	board := play.NewBoard(boardCells("sycophantic", "ephemeral"), opt.width, play.Palette{})

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 10, opt.width)
	live.interval = -1
	var errb bytes.Buffer

	keys := make(chan Key, 1)
	keys <- Key{Kind: KeyInterrupt}
	close(keys)
	playSession(t.Context(), d, opt, play.NewSession([]play.Question{board}), held, keys,
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	if strings.Contains(unstyled(live.Transcript()), "[0] ") {
		t.Errorf("the grid reached the transcript, where nothing can change:\n%s", live.Transcript())
	}
	if !strings.Contains(unstyled(tty.String()), "[0] sycophantic") {
		t.Errorf("the grid was never drawn:\n%s", unstyled(tty.String()))
	}
	// The toggle is drawn, and the FORM draws it — the loop only appends the bar,
	// so every row of the board on screen came out of one Prompt().
	frame := unstyled(tty.String())
	for _, row := range strings.Split(board.Prompt(), "\n") {
		if !strings.Contains(frame, row) {
			t.Errorf("the frame is missing the board's row %q:\n%s", row, frame)
		}
	}
	// The mode is on the PROMPT row now (R11) rather than a footer row, because
	// fitFooter drops from the end and that made the one statement of what a
	// click will mean the first thing a short terminal lost.
	if !strings.Contains(frame, "marking [yes] no") {
		t.Errorf("the frame does not say which mark is live:\n%s", frame)
	}
}

// THE PROMPT MUST NOT OFFER `d` ON A BOARD (#40 D12).
//
// Apply refuses the drop for a form holding many words, because `d` names no
// word on a grid. A prompt line offering it anyway is the exact bug gradePrompt
// was created to fix.
func TestABoardsPromptDoesNotOfferTheDropKey(t *testing.T) {
	board := play.NewBoard(boardCells("keel", "mesa"), 80, play.Palette{})
	line := gradePrompt(board)
	if strings.Contains(line, "remove from deck") {
		t.Errorf("a board's prompt offers a key Apply refuses:\n\t%q", line)
	}
	if !strings.Contains(line, quitKey) {
		t.Errorf("a board's prompt does not offer Ctrl-C:\n\t%q", line)
	}
	if n := visibleCells(line); n > defaultCols {
		t.Errorf("the prompt is %d columns wide, which wraps at %d and makes the frame a row taller than the board was offered for:\n\t%q", n, defaultCols, line)
	}
	// ...and every single-word form still gets the full set.
	for _, q := range []play.Question{
		play.NewRecall("keel", "d"),
		play.NewChoice("keel", "", []play.Option{{Gloss: "a", Correct: true}, {Gloss: "b"}}),
	} {
		if !strings.Contains(gradePrompt(q), sessionKeys) {
			t.Errorf("%T lost the reserved keys: %q", q, gradePrompt(q))
		}
	}
}

// A BOARD THAT DOES NOT FIT IS NOT OFFERED (#40 D15), which is what leaves
// fitFooter's budget invariant true instead of negotiating with it.
func TestFitsABoardCountsTheWholeLiveEdge(t *testing.T) {
	for _, tc := range []struct {
		termRows, boardRows, promptRows int
		want                            bool
	}{
		{10, 6, 1, true}, // a 4x4 board: 4 grid rows plus 2 of its own chrome
		{8, 6, 1, true},  // exactly: the board, a one-row keys prompt, the bar
		{7, 6, 1, false}, // one short, and half a board is unusable
		{24, 6, 1, true}, // an ordinary terminal
		{5, 3, 1, true},  // a board of one grid row
		{4, 3, 1, false}, //
		{0, 3, 1, false}, //
		{100, 28, 1, true},
		// AND THE PROMPT'S REAL HEIGHT, which is what a constant got wrong: the
		// keys line is 76 columns wide and a board is offered from 20.
		{8, 6, 2, false},
		{9, 6, 2, true},
		{11, 6, 4, true},
		{10, 6, 4, false},
	} {
		if got := fitsABoard(tc.termRows, tc.boardRows, tc.promptRows); got != tc.want {
			t.Errorf("fitsABoard(%d rows, a %d-row board, a %d-row prompt) = %v, want %v",
				tc.termRows, tc.boardRows, tc.promptRows, got, tc.want)
		}
	}
	// The rows it counts below the board are the rows boardFooter actually
	// DRAWS. Two owners of that number would put half a board on screen.
	board := play.NewBoard(boardCells("keel", "mesa", "run", "bank", "set"), 80, play.Palette{})
	footer := boardFooter(board, sittingFigures{}, palette{})
	if got, want := len(footer)-board.Rows(), barRows; got != want {
		t.Errorf("boardFooter adds %d rows below the board's own, but fitsABoard budgets %d", got, want)
	}
	// AND THE MEASUREMENT REACHES boardFits: at a width where the keys line
	// wraps, a board that would fit a one-row prompt must be refused.
	//
	// Read off the real prompt rather than assumed, so the numbers here cannot
	// drift from the wording.
	probe := play.NewBoard(boardCells("keel", "mesa", "run", "bank"), 40, play.Palette{})
	pr := displayRows(gradePrompt(probe), 40)
	if pr < 2 {
		t.Fatalf("the keys prompt is %d row(s) at 40 columns; this case is vacuous", pr)
	}
	tight := probe.Rows() + 1 + barRows // enough for a ONE-row prompt, and no more
	if boardFits([]string{"keel", "mesa", "run", "bank"}, options{width: 40, rows: tight}) {
		t.Errorf("a %d-row terminal was offered a board whose prompt needs %d rows", tight, pr)
	}
	if !boardFits([]string{"keel", "mesa", "run", "bank"}, options{width: 40, rows: tight + pr - 1}) {
		t.Errorf("a terminal with exactly enough room refused the board")
	}
}

// DONE-WHEN 9, THROUGH THE STORE: a review event on disk names the form that
// asked it (#40 D4a).
//
// play_test pins that Apply stamps every record; this pins that the stamp
// survives CaptureReview and reaches the log, which is the only place the query
// this field exists for can read it.
func TestAReviewEventNamesItsFormOnDisk(t *testing.T) {
	for _, tc := range []struct {
		name string
		q    func(opt options) play.Question
		key  Key
		want string
	}{
		{"form 2.1, the recall", func(options) play.Question {
			return play.NewRecall("sycophantic", "behaving obsequiously")
		}, Key{Kind: KeyRune, Rune: 'y'}, "recall"},
		{"form 2.3, the meaning", func(options) play.Question {
			return play.NewChoice("sycophantic", "", []play.Option{
				{Gloss: "behaving obsequiously", Correct: true}, {Gloss: "a flat-topped hill"},
			})
		}, Key{Kind: KeyRune, Rune: '1'}, "meaning"},
		{"form 2.5, the board", func(opt options) play.Question {
			return play.NewBoard(boardCells("sycophantic", "ephemeral"), opt.width, play.Palette{})
		}, Key{Kind: KeyRune, Rune: '0'}, "board"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, opt, st := playRig(t, "sycophantic", "ephemeral")
			_, held := questionsFor(t, d, opt)

			tty := &syncBuf{}
			live := newPinnedScreen(tty, 24, opt.width)
			live.interval = -1
			var errb bytes.Buffer

			keys := make(chan Key, 2)
			keys <- tc.key
			keys <- Key{Kind: KeyInterrupt}
			close(keys)

			playSession(t.Context(), d, opt, play.NewSession([]play.Question{tc.q(opt)}), held, keys,
				console{view: live, finish: func() {}, stdout: live, stderr: &errb})

			evs := reviewEvents(t, st)
			if len(evs) != 1 {
				t.Fatalf("%d review events, want the one answer:\n%s", len(evs), unstyled(tty.String()))
			}
			if evs[0].Form != tc.want {
				t.Errorf("the log names the form %q, want %q — the field is what lets a later query ask whether this form promotes too generously", evs[0].Form, tc.want)
			}
		})
	}
}

// seedBox drives a word to a box by logging n correct reviews, so a test can
// build a deck that spans the board's threshold.
//
// Through the LOG rather than by writing a Progress: box is a fold over events,
// and a test that set the number directly would be asserting against a state the
// program cannot reach.
func seedBox(t *testing.T, st *store.Mem, word string, n int) {
	t.Helper()
	for i := range n {
		if err := st.AppendEvent(store.ReviewEvent{
			Word: word, Kind: store.EventReviewed, Found: true, Correct: true,
			At: aDay.Add(time.Duration(i) * time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
	}
}

// DONE-WHEN 3 (the issue's): THE SCHEDULER CHOOSES THE FORM, and both sides are
// asserted — 2.5 at box >= 3, one at a time below (D4).
//
// This is the first thing in the program to consult a box when choosing HOW to
// ask. Everything before it was a capability question: form 2.3 when the deck
// could supply distractors, 2.1 when it could not.
func TestTheBoxPicksTheForm(t *testing.T) {
	opt := options{width: defaultCols, rows: defaultRows}
	prog := map[string]schedule.Progress{
		"keel": {Box: 0}, "mesa": {Box: 1}, "run": {Box: 2},
		"bank": {Box: 3}, "set": {Box: 4}, "quokka": {Box: 9},
	}
	keys := []string{"keel", "bank", "mesa", "set", "run", "quokka"}
	single, boards := boardsFor(keys, prog, opt)

	if want := []string{"keel", "mesa", "run"}; !slices.Equal(single, want) {
		t.Errorf("asked one at a time: %v, want %v — the boxes under %d, in queue order", single, want, boardBox)
	}
	if len(boards) != 1 {
		t.Fatalf("%d boards, want one holding the three mature words", len(boards))
	}
	if want := []string{"bank", "set", "quokka"}; !slices.Equal(boards[0], want) {
		t.Errorf("the board holds %v, want %v — the boxes at or above %d", boards[0], want, boardBox)
	}
	// A word with no history at all is box 0, which is the common case on a young
	// deck and must not reach the board.
	single, boards = boardsFor([]string{"unheard-of"}, map[string]schedule.Progress{}, opt)
	if len(boards) != 0 || len(single) != 1 {
		t.Errorf("a word with no history produced %d boards and %d singles, want 0 and 1", len(boards), len(single))
	}
}

// PACKED SIXTEEN AT A TIME, which is the load argument made concrete.
func TestBoardsArePackedToTheLabelAlphabet(t *testing.T) {
	opt := options{width: defaultCols, rows: 60}
	var keys []string
	prog := map[string]schedule.Progress{}
	for i := range 40 {
		k := fmt.Sprintf("word%02d", i)
		keys = append(keys, k)
		prog[k] = schedule.Progress{Box: 5}
	}
	single, boards := boardsFor(keys, prog, opt)
	if len(single) != 0 {
		t.Errorf("%d words were asked one at a time, want none — every one is eligible", len(single))
	}
	var sizes []int
	for _, b := range boards {
		sizes = append(sizes, len(b))
	}
	if want := []int{16, 16, 8}; !slices.Equal(sizes, want) {
		t.Errorf("boards of %v, want %v", sizes, want)
	}
	// And every word is on exactly one of them.
	seen := map[string]bool{}
	for _, b := range boards {
		for _, k := range b {
			if seen[k] {
				t.Errorf("%q is on two boards", k)
			}
			seen[k] = true
		}
	}
	if len(seen) != len(keys) {
		t.Errorf("%d of %d words reached a board", len(seen), len(keys))
	}
}

// DONE-WHEN 10: A BOARD IS NEVER DRAWN CLIPPED (D15).
//
// A terminal too short for the whole board sends those words to form 2.3 for
// that sitting, which is a complete answer rather than a degraded one. The
// alternative — a floor in fitFooter — would have broken the budget Paint rests
// on, and the symptom would have been a click landing on the wrong word.
func TestAShortTerminalGetsMeaningChoiceNotAClippedBoard(t *testing.T) {
	prog := map[string]schedule.Progress{}
	var keys []string
	for i := range 16 {
		k := fmt.Sprintf("word%02d", i)
		keys = append(keys, k)
		prog[k] = schedule.Progress{Box: 5}
	}
	// Tall enough: a 16-word board at 80 columns is four grid rows plus its own
	// two, and the prompt and bar make eight.
	if _, boards := boardsFor(keys, prog, options{width: defaultCols, rows: 8}); len(boards) != 1 {
		t.Errorf("an 8-row terminal offered %d boards, want 1", len(boards))
	}
	for _, tc := range []struct {
		name string
		opt  options
	}{
		{"one row too short", options{width: defaultCols, rows: 7}},
		{"a small window", options{width: defaultCols, rows: 5}},
		{"too narrow to lay out at all", options{width: 12, rows: 60}},
		// THE WIDTH AXIS. The keys line is 76 columns, so below that it wraps
		// and takes rows off the top that the board was counting on — which a
		// constant chrome budget missed entirely, and `fitFooter` then dropped
		// the bar, the panel and eventually the TOGGLE off a board that had
		// been offered anyway.
		{"narrow enough that the prompt wraps past the height", options{width: 40, rows: 10}},
		{"very narrow, where the prompt takes four rows", options{width: 24, rows: 13}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			single, boards := boardsFor(keys, prog, tc.opt)
			if len(boards) != 0 {
				t.Errorf("%d boards offered on a terminal that cannot draw one whole", len(boards))
			}
			if len(single) != len(keys) {
				t.Errorf("%d of %d words fell back to being asked one at a time", len(single), len(keys))
			}
		})
	}
}

// DONE-WHEN 1: A SITTING OF ELIGIBLE WORDS PRESENTS THEM AS A GRID — end to end
// through todaysQuestions, over a deck that spans the threshold.
func TestASittingOfDueWordsIsABoard(t *testing.T) {
	mature := []string{"quokka", "mesa", "parrot", "bank"}
	young := []string{"sycophantic", "ephemeral"}
	d, opt, st := playRig(t, append(append([]string{}, mature...), young...)...)
	for _, w := range mature {
		seedBox(t, st, w, boardBox)
	}

	var out, errb bytes.Buffer
	qs, _, code := todaysQuestions(d, opt, &out, &errb)
	if code != 0 {
		t.Fatalf("todaysQuestions = %d: %s", code, errb.String())
	}

	var board *play.Board
	var singles []play.Question
	for _, q := range qs {
		if b, ok := q.(*play.Board); ok {
			if board != nil {
				t.Fatal("two boards for four mature words, want one")
			}
			board = b
			continue
		}
		singles = append(singles, q)
	}
	if board == nil {
		t.Fatalf("no board in a sitting of %d questions over a deck with %d mature words", len(qs), len(mature))
	}
	if len(singles) != len(young) {
		t.Errorf("%d questions asked one at a time, want the %d young words", len(singles), len(young))
	}
	// BOTH SIDES: every mature word is on the board, every young one is not.
	grid := board.Prompt()
	for _, w := range mature {
		if !strings.Contains(grid, w) {
			t.Errorf("%q is mature and not on the board:\n%s", w, grid)
		}
	}
	for _, w := range young {
		if strings.Contains(grid, w) {
			t.Errorf("%q is young and reached the board:\n%s", w, grid)
		}
	}
	// The panel has something to show: the gloss came from the same sense form
	// 2.3 asks about.
	if _, ok := board.Mark(0); !ok {
		t.Fatal("the first cell refused a mark")
	}
	panel := strings.Split(board.Prompt(), "\n")
	if last := panel[len(panel)-1]; !strings.Contains(last, " ") {
		t.Errorf("the panel is %q — the board was built with no glosses", last)
	}
	// The board comes LAST: retrieval gets the freshest attention.
	if _, ok := qs[len(qs)-1].(*play.Board); !ok {
		t.Errorf("the board is not the last question; the queue is %T...%T", qs[0], qs[len(qs)-1])
	}
}

// DONE-WHEN 11: THE OUTCOME SURVIVES THE SITTING (#40 D10, T9).
//
// The board is live edge and vanishes whole — that is what bought marks that
// change as they land. What must not vanish is which words the learner said no
// to, because those are the ones the sitting was actually about.
func TestABoardLeavesItsRelearnListInTheTranscript(t *testing.T) {
	d, opt, _ := playRig(t, "quokka", "mesa", "parrot")
	_, held := questionsFor(t, d, opt)
	board := play.NewBoard(boardCells("quokka", "mesa", "parrot"), opt.width, play.Palette{})

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 24, opt.width)
	live.interval = -1
	var errb bytes.Buffer

	// yes on the first, then no on the other two — Tab switches the mark.
	keys := make(chan Key, 5)
	keys <- Key{Kind: KeyRune, Rune: '0'}
	keys <- Key{Kind: KeyTab}
	keys <- Key{Kind: KeyRune, Rune: '1'}
	keys <- Key{Kind: KeyRune, Rune: '2'}
	keys <- Key{Kind: KeyInterrupt}
	close(keys)

	playSession(t.Context(), d, opt, play.NewSession([]play.Question{board}), held, keys,
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	script := unstyled(live.Transcript())
	if !strings.Contains(script, "relearn: mesa, parrot") {
		t.Errorf("the transcript does not name the words marked no:\n%s", script)
	}
	if strings.Contains(script, "quokka") {
		t.Errorf("a word marked YES is in the relearn list:\n%s", script)
	}
	// ...and the grid itself is still not in the transcript.
	if strings.Contains(script, "[0] ") {
		t.Errorf("the grid reached the transcript:\n%s", script)
	}
}

// A board swept entirely `yes` leaves NOTHING, because an empty relearn list is
// not news — and a bare "relearn:" would read as a list that failed to render.
func TestABoardWithNothingToRelearnWritesNoLine(t *testing.T) {
	d, opt, _ := playRig(t, "quokka", "mesa")
	_, held := questionsFor(t, d, opt)
	board := play.NewBoard(boardCells("quokka", "mesa"), opt.width, play.Palette{})

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 24, opt.width)
	live.interval = -1
	var errb bytes.Buffer
	keys := make(chan Key, 3)
	keys <- Key{Kind: KeyRune, Rune: '0'}
	keys <- Key{Kind: KeyRune, Rune: '1'}
	keys <- Key{Kind: KeyInterrupt}
	close(keys)

	playSession(t.Context(), d, opt, play.NewSession([]play.Question{board}), held, keys,
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	if strings.Contains(unstyled(live.Transcript()), "relearn") {
		t.Errorf("an all-yes board wrote a relearn line:\n%s", live.Transcript())
	}
}

// DONE-WHEN 12: THE BAR COUNTS WORDS, NOT SLOTS (#40 D8).
//
// `done` was always a word count — every mark scores — so a board made the bar
// compare words against slots, and a twenty-word sitting with one board in it
// read "0 of 2".
func TestTheBarCountsWordsNotSlots(t *testing.T) {
	qs := []play.Question{
		play.NewRecall("keel", "d"),
		play.NewBoard(boardCells("quokka", "mesa", "parrot", "bank"), 80, play.Palette{}),
		play.NewChoice("run", "", []play.Option{{Gloss: "a", Correct: true}, {Gloss: "b"}}),
	}
	if got, want := sittingWords(qs), 6; got != want {
		t.Errorf("sittingWords = %d, want %d — one board of four plus two single questions", got, want)
	}
	if got := sittingWords(nil); got != 0 {
		t.Errorf("an empty sitting counts %d words", got)
	}
	// The number reaches the bar.
	d, opt, _ := playRig(t, "quokka", "mesa", "parrot", "bank")
	_, held := questionsFor(t, d, opt)
	board := play.NewBoard(boardCells("quokka", "mesa", "parrot", "bank"), opt.width, play.Palette{})

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 24, opt.width)
	live.interval = -1
	var errb bytes.Buffer
	keys := make(chan Key, 2)
	keys <- Key{Kind: KeyRune, Rune: '0'}
	keys <- Key{Kind: KeyInterrupt}
	close(keys)

	playSession(t.Context(), d, opt, play.NewSession([]play.Question{board}), held, keys,
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	if frame := unstyled(tty.String()); !strings.Contains(frame, "of 4") {
		t.Errorf("the bar does not count the board's four words:\n%s", frame)
	}
}

// DONE-WHEN 13, MEASURED — and the measurement moved the claim (#40 R5).
//
// The plan asked for "materially fewer KEYSTROKES than form 2.3", and that is
// not what the numbers say. Both forms cost one keystroke per word in the good
// case; 2.3 costs a second on every miss (the definition goes up and any key
// moves on) and a board costs one per mode switch. Over eight words that is 1.00
// against 1.00, or 1.12 against 1.25. Marginal either way.
//
// What IS material is how much the learner has to READ. Form 2.3 writes a whole
// rendered entry per word into the transcript — the four options, then the right
// answer, then the entry — and a board writes one line for the entire sweep.
// Measured at 8.8 lines per word against 0.4, and 23.1 against 0.6 once misses
// are involved. That is the Spec's own claim ("a hundred mature words swept in a
// grid cost what ten fragile ones cost") and it is a reading cost, not a typing
// one — which is the right reading of "cost" for a form whose whole argument is
// that a large deck becomes unaffordable.
//
// So this pins BOTH: the keystroke floor the plan's red-when names, and the
// reading ratio that carries the claim.
func TestABoardCostsFarLessPerWordThanMeaningChoice(t *testing.T) {
	words := []string{"quokka", "mesa", "parrot", "bank", "concrete", "ephemeral", "run", "set"}

	// THE BOARD: one keystroke per word, one mode throughout.
	d, opt, _ := playRig(t, words...)
	_, held := questionsFor(t, d, opt)
	board := play.NewBoard(boardCells(words...), opt.width, play.Palette{})
	var boardKeys []Key
	for i := range words {
		boardKeys = append(boardKeys, Key{Kind: KeyRune, Rune: rune(play.BoardLabels[i])})
	}
	boardLines := sittingCost(t, d, opt, []play.Question{board}, held, boardKeys)

	// FORM 2.3 over the same words, every one answered right — which is the
	// cheapest that form can possibly be, so the comparison is against its best
	// case rather than a convenient one.
	d2, opt2, _ := playRig(t, words...)
	qs, held2, code := todaysQuestions(d2, opt2, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("todaysQuestions = %d", code)
	}
	var singles []play.Question
	for _, q := range qs {
		if _, ok := q.(*play.Board); !ok {
			singles = append(singles, q)
		}
	}
	if len(singles) != len(words) {
		t.Fatalf("%d single questions for %d words; the comparison is not like for like", len(singles), len(words))
	}
	var choiceKeys []Key
	for _, q := range singles {
		choiceKeys = append(choiceKeys, Key{Kind: KeyRune, Rune: []rune(gradeKey(t, q, play.Correct))[0]})
	}
	choiceLines := sittingCost(t, d2, opt2, singles, held2, choiceKeys)

	// ONE KEYSTROKE PER WORD, which is the plan's red-when: "the grid asks for
	// more than one keystroke per word".
	//
	// Asserted against the BOARD, not against the script: `len(boardKeys) >
	// len(words)` was the first version and it cannot fail, because boardKeys is
	// built by ranging over words. What has to be true is that those keystrokes
	// SPENT the board — one per word, and nothing left owing.
	if len(boardKeys) != len(words) {
		t.Fatalf("the script is %d keystrokes for %d words; this row asserts nothing", len(boardKeys), len(words))
	}
	if !board.Spent() {
		t.Errorf("%d keystrokes did not finish a board of %d words — it asks for more than one per word", len(boardKeys), len(words))
	}
	// AND THE READING COST, which is where the load argument actually lives.
	perWordBoard := float64(boardLines) / float64(len(words))
	perWordChoice := float64(choiceLines) / float64(len(words))
	if perWordBoard <= 0 {
		t.Fatal("the board's sitting produced no transcript at all; the ratio below would be meaningless")
	}
	if ratio := perWordChoice / perWordBoard; ratio < 10 {
		t.Errorf("form 2.3 costs %.1f transcript lines per word and the board costs %.1f — a ratio of %.1f, "+
			"and the Spec's claim is that a grid makes a large deck affordable (ten to one)",
			perWordChoice, perWordBoard, ratio)
	}
	// The MECHANISM, stated so a future change that quietly starts writing the
	// grid to the buffer fails here too rather than only in the ratio.
	if boardLines > len(words) {
		t.Errorf("the board wrote %d transcript lines for %d words — it is meant to write one line for the whole sweep", boardLines, len(words))
	}
}

// sittingCost runs a scripted sitting and returns how many lines it left in the
// transcript — what the learner has to read.
//
// The summary the sitting ends with is counted for BOTH forms, which makes the
// board look slightly worse than it is. That is the conservative direction.
func sittingCost(t *testing.T, d deps, opt options, qs []play.Question, held *sittingDeck, script []Key) int {
	t.Helper()
	tty := &syncBuf{}
	live := newPinnedScreen(tty, 40, opt.width)
	live.interval = -1
	var errb bytes.Buffer
	keys := make(chan Key, len(script)+1)
	for _, k := range script {
		keys <- k
	}
	close(keys)
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keys,
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})
	return strings.Count(unstyled(live.Transcript()), "\n")
}

// DONE-WHEN 2 AND 4: EVERY MARK REACHES THE LOG AS IT HAPPENS, so Ctrl-C leaves
// the marked words recorded and the unmarked ones untouched (#40 D3).
//
// These are one observable. A board that wrote its marks at the END would look
// identical in a completed sitting and lose everything on an interrupt — and the
// interrupt is the case the operator asked for by name: *"ctrl-C means nothing is
// changed from that form. already clicked words can still be recorded, but
// unmarked words are just unmarked, no state change for them."*
//
// Asserted through the FOLD rather than the event count alone, because "no state
// change" is a claim about boxes, and boxes are what the log folds to.
func TestCtrlCCancelsABoardWithoutMovingUnmarkedWords(t *testing.T) {
	words := []string{"quokka", "mesa", "parrot", "bank"}
	d, opt, st := playRig(t, words...)
	_, held := questionsFor(t, d, opt)
	board := play.NewBoard(boardCells(words...), opt.width, play.Palette{})

	before := schedule.Fold(eventsOf(t, st))

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 24, opt.width)
	live.interval = -1
	var errb bytes.Buffer
	keys := make(chan Key, 3)
	keys <- Key{Kind: KeyRune, Rune: '0'} // quokka: yes
	keys <- Key{Kind: KeyRune, Rune: '1'} // mesa: yes
	keys <- Key{Kind: KeyInterrupt}       // ...and stop, two words unmarked
	close(keys)

	playSession(t.Context(), d, opt, play.NewSession([]play.Question{board}), held, keys,
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	// TWO events, not four and not zero. Zero is what batching at the end would
	// leave; four would mean the interrupt committed the rest.
	evs := reviewEvents(t, st)
	if len(evs) != 2 {
		t.Fatalf("%d review events after two marks and Ctrl-C, want 2", len(evs))
	}
	got := map[string]bool{evs[0].Word: true, evs[1].Word: true}
	if !got["quokka"] || !got["mesa"] {
		t.Errorf("the log holds %v, want the two words that were marked", got)
	}

	// AND THE UNMARKED WORDS DID NOT MOVE.
	after := schedule.Fold(eventsOf(t, st))
	for _, w := range []string{"parrot", "bank"} {
		if after[w] != before[w] {
			t.Errorf("%q was never marked and its progress changed: %+v -> %+v", w, before[w], after[w])
		}
	}
	if after["quokka"] == before["quokka"] {
		t.Errorf("quokka WAS marked and its progress did not change — the premise above is vacuous")
	}
	// Both marks were `yes`, so there is nothing to relearn and no line — the
	// interrupted board that DOES have one is the sibling test below.
	if strings.Contains(unstyled(live.Transcript()), "relearn") {
		t.Errorf("an all-yes board wrote a relearn line:\n%s", live.Transcript())
	}
}

// Ctrl-C mid-board still writes the relearn line for what WAS marked no.
func TestCtrlCOnABoardStillLeavesItsRelearnList(t *testing.T) {
	words := []string{"quokka", "mesa", "parrot", "bank"}
	d, opt, _ := playRig(t, words...)
	_, held := questionsFor(t, d, opt)
	board := play.NewBoard(boardCells(words...), opt.width, play.Palette{})

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 24, opt.width)
	live.interval = -1
	var errb bytes.Buffer
	keys := make(chan Key, 4)
	keys <- Key{Kind: KeyTab}             // switch to marking no
	keys <- Key{Kind: KeyRune, Rune: '2'} // parrot: no
	keys <- Key{Kind: KeyInterrupt}       // ...and stop, three unmarked
	close(keys)

	playSession(t.Context(), d, opt, play.NewSession([]play.Question{board}), held, keys,
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	script := unstyled(live.Transcript())
	if !strings.Contains(script, "relearn: parrot") {
		t.Errorf("an interrupted board lost the outcome it had:\n%s", script)
	}
	for _, w := range []string{"quokka", "mesa", "bank"} {
		if strings.Contains(script, w) {
			t.Errorf("%q was never marked and is in the relearn list:\n%s", w, script)
		}
	}
}

// THE BOARD'S OWN ROWS COME FIRST IN THE FOOTER, IN ORDER, and formCell reads a
// footer entry index straight back as a grid row — so anything inserted above
// the grid silently shifts every cell.
//
// `boardFooter`'s comment calls that load-bearing and nothing tested it. A second
// live-edge form, or anything wanting a row above the grid, is where it breaks.
func TestBoardFooterPutsTheFormsOwnRowsFirst(t *testing.T) {
	board := play.NewBoard(boardCells("quokka", "mesa", "parrot", "bank", "set"), 80, play.Palette{})
	footer := boardFooter(board, sittingFigures{}, palette{})
	own := strings.Split(board.Prompt(), "\n")
	if len(footer) < len(own) {
		t.Fatalf("the footer is %d rows and the board draws %d", len(footer), len(own))
	}
	for i, row := range own {
		if footer[i] != row {
			t.Errorf("footer entry %d is %q, but the board's own row %d is %q — formCell reads that index back as a grid row", i, footer[i], i, row)
		}
	}
	if len(footer) != len(own)+barRows {
		t.Errorf("the footer is %d rows, want the board's %d plus %d for the bar", len(footer), len(own), barRows)
	}
}

// DONE-WHEN 2, the completed sweep: N marks, N events.
func TestEveryMarkOnABoardIsRecordedImmediately(t *testing.T) {
	words := []string{"quokka", "mesa", "parrot", "bank"}
	d, opt, st := playRig(t, words...)
	_, held := questionsFor(t, d, opt)
	board := play.NewBoard(boardCells(words...), opt.width, play.Palette{})

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 24, opt.width)
	live.interval = -1
	var errb bytes.Buffer
	keys := make(chan Key, len(words)+1)
	for i := range words {
		keys <- Key{Kind: KeyRune, Rune: rune(play.BoardLabels[i])}
	}
	keys <- Key{Kind: KeyInterrupt}
	close(keys)

	playSession(t.Context(), d, opt, play.NewSession([]play.Question{board}), held, keys,
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	evs := reviewEvents(t, st)
	if len(evs) != len(words) {
		t.Fatalf("%d review events for %d marks:\n%s", len(evs), len(words), unstyled(live.Transcript()))
	}
	seen := map[string]bool{}
	for _, e := range evs {
		if seen[e.Word] {
			t.Errorf("%q was recorded twice — Fold would read that as two reviews on one day", e.Word)
		}
		seen[e.Word] = true
	}
}

// DONE-WHEN 6: THE MOUSE-LESS PATH WORKS, and `d` is not a cell label.
//
// #38's pty rows exist because a terminal reporting no mouse must keep working.
// Without the printed keys a board would silently degrade to "everything is no",
// which is wrong rather than merely limited.
func TestABoardIsMarkableByKeyAlone(t *testing.T) {
	// SIXTEEN REAL WORDS, so the whole label alphabet is exercised including the
	// `d` gap — and real ones because the rig's deck is looked up for its glosses.
	words := []string{
		"bank", "concrete", "content", "defenestrate",
		"desert", "ephemeral", "even", "man",
		"mesa", "minute", "parrot", "present",
		"pulp", "quokka", "read", "run",
	}
	if len(words) != play.MaxBoardWords {
		t.Fatalf("%d words, want a full board of %d", len(words), play.MaxBoardWords)
	}
	d, opt, st := playRig(t, words...)
	_, held := questionsFor(t, d, opt)
	board := play.NewBoard(boardCells(words...), opt.width, play.Palette{})

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 30, opt.width)
	live.interval = -1
	var errb bytes.Buffer
	keys := make(chan Key, len(words)+1)
	for i := range words {
		keys <- Key{Kind: KeyRune, Rune: rune(play.BoardLabels[i])}
	}
	keys <- Key{Kind: KeyInterrupt}
	close(keys)

	playSession(t.Context(), d, opt, play.NewSession([]play.Question{board}), held, keys,
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	if n := len(reviewEvents(t, st)); n != len(words) {
		t.Errorf("%d events for %d keys — a mouse-less terminal cannot finish a board", n, len(words))
	}
}

// ...AND `d` IS NOT A CELL LABEL. It is the session's drop key, taken by toInput
// before any form sees it — and on a board Apply refuses it too, because a grid
// has no single current word to remove.
func TestDOnABoardIsNotACellLabel(t *testing.T) {
	words := []string{"quokka", "mesa", "parrot", "bank"}
	d, opt, st := playRig(t, words...)
	_, held := questionsFor(t, d, opt)
	board := play.NewBoard(boardCells(words...), opt.width, play.Palette{})

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 24, opt.width)
	live.interval = -1
	var errb bytes.Buffer
	keys := make(chan Key, 3)
	keys <- Key{Kind: KeyRune, Rune: 'd'}
	keys <- Key{Kind: KeyRune, Rune: 'D'}
	keys <- Key{Kind: KeyInterrupt}
	close(keys)

	playSession(t.Context(), d, opt, play.NewSession([]play.Question{board}), held, keys,
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	if n := len(reviewEvents(t, st)); n != 0 {
		t.Errorf("%d review events after pressing d twice — it graded a cell", n)
	}
	if strings.Contains(unstyled(live.Transcript()), "removed") {
		t.Errorf("d removed a word from the deck on a board, where it names none:\n%s", live.Transcript())
	}
	deck, err := st.Deck()
	if err != nil {
		t.Fatal(err)
	}
	if len(deck) != len(words) {
		t.Errorf("the deck holds %d words, want %d — d took one", len(deck), len(words))
	}
	// ...and it still drops on a form that HAS a current word, which is the other
	// half: the key was refused for a reason, not disabled.
	d2, opt2, st2 := playRig(t, "quokka", "mesa")
	_, held2 := questionsFor(t, d2, opt2)
	tty2 := &syncBuf{}
	live2 := newPinnedScreen(tty2, 24, opt2.width)
	live2.interval = -1
	keys2 := make(chan Key, 2)
	keys2 <- Key{Kind: KeyRune, Rune: 'd'}
	keys2 <- Key{Kind: KeyInterrupt}
	close(keys2)
	playSession(t.Context(), d2, opt2, play.NewSession([]play.Question{play.NewRecall("quokka", "d")}), held2, keys2,
		console{view: live2, finish: func() {}, stdout: live2, stderr: &errb})
	deck2, err := st2.Deck()
	if err != nil {
		t.Fatal(err)
	}
	if len(deck2) != 1 {
		t.Errorf("the deck holds %d words after a drop on form 2.1, want 1 — d stopped working everywhere", len(deck2))
	}
}

// eventsOf is every event in the store, for folding.
func eventsOf(t *testing.T, st *store.Mem) []store.ReviewEvent {
	t.Helper()
	all, err := st.Events(time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	return all
}

// R9, THROUGH THE LOOP: a narrowing resize under a live board must not leave a
// click marking the wrong word (R12).
//
// The first version of this test asserted over an EVENT SET IT NEVER PRODUCED —
// `for _, e := range reviewEvents(...)` with no count check, and the click never
// landed, so it passed with `formCell` stubbed to return false. The rule it
// broke is one this issue's own Log already records for T13: **read the click's
// row and column off the PAINT, never compute them** — the goroutine derived a
// row from logical writes while `FooterRowAt` works in physical rows.
//
// So the click is placed from the frame, and the premise is checked before the
// assertion: a test whose subject is an event must assert the event happened.
func TestANarrowingResizeKeepsTheBoardsClickMapHonest(t *testing.T) {
	// Long words, so the board's rows are wide at 80 and MUST be relaid out at
	// 40 or they wrap — which is the whole failure.
	words := []string{"arrondissement", "sycophantic", "defenestrate", "ephemeral"}
	d, opt, st := playRig(t, words...)
	_, held := questionsFor(t, d, opt)
	board := play.NewBoard(boardCells(words...), 80, play.Palette{})
	if board.Rows() != 3 {
		t.Fatalf("expected a one-row grid plus chrome at 80 columns, got %d rows:\n%s", board.Rows(), board.Prompt())
	}

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 24, 80)
	live.interval = -1
	var errb bytes.Buffer

	resizes := make(chan winSize, 1)
	keys := make(chan Key, 2)

	// The resize lands, the frame settles, and only THEN is the click placed —
	// from the painted frame, at the physical row and column the terminal is
	// actually showing the second cell at.
	// The driver ALWAYS closes `keys`, whatever it fails to find. A helper
	// goroutine that gives up without closing leaves playSession blocked on the
	// channel forever, so the test HANGS instead of failing — which is how the
	// first version of this behaved under the very mutation it exists to catch.
	// A test that hangs on the defect certifies about as much as one that passes
	// on it, and takes longer to say so. (`waitFor`'s own t.Fatal is worse than
	// useless here: FailNow on a non-test goroutine is a Goexit, so it skips
	// every remaining line including the close.)
	go func() {
		defer close(keys)
		settled := func(cond func() bool) bool {
			for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
				if cond() {
					return true
				}
				time.Sleep(time.Millisecond)
			}
			return false
		}
		if !settled(func() bool { return strings.Contains(unstyled(tty.String()), "[0] ") }) {
			return
		}
		resizes <- winSize{rows: 24, cols: 40}
		// THE DRIVER NEVER TOUCHES THE BOARD. `board` belongs to the loop
		// goroutine, which is about to relayout it — reading `board.Rows()` here
		// is a data race, and -race says so. Production has one goroutine on a
		// form for exactly this reason.
		//
		// A PROBE instead: the same words at the same width lay out the same way,
		// which is the property boardFits already relies on. Everything the
		// driver needs to place a click comes from the probe (a column) and the
		// screen (a row), and the screen is behind a mutex.
		probe := play.NewBoard(boardCells(words...), 40, play.Palette{})
		col := strings.Index(strings.Split(probe.Prompt(), "\n")[0], "[1] ")
		if col < 0 {
			return
		}

		// EVERYTHING COMES FROM THE SCREEN, and the frame text is not consulted
		// at all. Two earlier spellings scraped it and both were wrong in the
		// same way: `FooterRowAt` answers in the TERMINAL's rows, and a frame
		// split on "\r\n" gives LOGICAL lines. The keys prompt is one logical
		// line and — at 76 columns in a 40-column window — two physical rows, so
		// the two indices differ by one from that point down and a click placed
		// by frame index lands a row high. (The first spelling also matched the
		// pre-resize paint outright, because the 40-column grid row is a PREFIX
		// of the 80-column one.)
		//
		// The relayout is observable through the map alone: at 80 the board is
		// three rows and the footer holds four entries; at 40 it is four and the
		// footer holds five. So wait until some row reports the LAST entry index
		// a relaid-out board produces — that count cannot be reached by the old
		// layout — and then take the row that is entry 0.
		lastEntry := probe.Rows() // entries are the board's rows, then the bar
		row := -1
		if !settled(func() bool {
			seenLast, first := false, -1
			for r := 0; r < 24; r++ {
				e, off, ok := live.FooterRowAt(r)
				if !ok {
					continue
				}
				if e == lastEntry {
					seenLast = true
				}
				if e == 0 && off == 0 && first < 0 {
					first = r
				}
			}
			row = first
			return seenLast && first >= 0
		}) {
			return
		}
		keys <- Key{Kind: KeyClick, Row: row, Col: col + len("[1] ")}
		keys <- Key{Kind: KeyInterrupt}
	}()

	playSession(t.Context(), d, opt, play.NewSession([]play.Question{board}), held, keys,
		console{view: live, resizes: resizes, finish: func() {}, stdout: live, stderr: &errb})

	// THE PREMISE FIRST: the click landed at all. Without this the assertion
	// below is a loop over nothing, which is what shipped and passed.
	evs := reviewEvents(t, st)
	if len(evs) != 1 {
		t.Fatalf("%d review events, want the one click — the click never reached the form:\n%s",
			len(evs), lastPaintedFrame(tty.String()))
	}
	// ...and the frame really did show the grid, so the row above was a grid row
	// rather than an empty footer entry that happened to answer.
	if !strings.Contains(unstyled(tty.String()), "[1] "+words[1]) {
		t.Fatalf("the second cell was never painted:\n%s", lastPaintedFrame(tty.String()))
	}
	// AND IT MARKED THE WORD DRAWN THERE. Without the relayout the board's rows
	// are 74 columns wide at a 40-column terminal, each wraps into two physical
	// rows, and the column carries a different word.
	if evs[0].Word != words[1] {
		t.Errorf("the click marked %q, want %q — the column meant a different word after the resize", evs[0].Word, words[1])
	}

	// THE BOARD'S OWN ROWS FIT, which is the invariant the click map rests on:
	// one footer entry, one physical row, so an entry index IS a grid row. The
	// keys prompt and the bar may wrap — Paint budgets the first with
	// displayRows and fitFooter drops the second — which is exactly why the
	// board's rows are the ones that have to fit.
	//
	// Read here, AFTER playSession returned: the loop goroutine is done, so the
	// board is this goroutine's to look at.
	frame := unstyled(tty.String())
	for i, row := range strings.Split(board.Prompt(), "\n") {
		if n := visibleCells(row); n > 40 {
			t.Errorf("the board's row %d is %d columns after a resize to 40 — it wraps, and a click on the continuation means another word:\n%q", i, n, row)
		}
		if row != "" && !strings.Contains(frame, row) {
			t.Errorf("the board's row %d was never drawn after the resize:\n%q", i, row)
		}
	}
	if board.Rows() <= 3 {
		t.Errorf("the board is %d rows at 40 columns and was 3 at 80 — it did not relayout", board.Rows())
	}
}

// lastPaintedFrame is the most recent whole frame in a session's output.
func lastPaintedFrame(out string) string {
	frames := strings.Split(unstyled(out), cursorHome+eraseDown)
	return frames[len(frames)-1]
}

// R17: A BOARD THAT BECOMES CURRENT AFTER A RESIZE IS LAID OUT FOR THE TERMINAL
// AS IT IS, not the one it was built for.
//
// The resize case told `s.Current()` its new width, which fixed the board on
// screen at that moment and no other. The NEXT board was built by
// `todaysQuestions` at the old width and painted with rows too wide — so its
// rows wrapped, a footer entry stopped being one physical row, and BR-8's
// wrong-word click came back through a door the first fix could not see.
//
// `show` is the one place that draws, so it is the only place that can promise
// this for every board.
func TestABoardBuiltBeforeAResizeIsStillLaidOutForTheTerminal(t *testing.T) {
	first := []string{"keel", "mesa", "run", "bank"}
	// Long words, so an 80-column layout is far too wide for a 40-column window.
	second := []string{"arrondissement", "sycophantic", "defenestrate", "ephemeral"}
	d, opt, _ := playRig(t, "quokka", "concrete")
	_, held := questionsFor(t, d, opt)

	// BOTH boards built at 80, as todaysQuestions would before any resize.
	a := play.NewBoard(boardCells(first...), 80, play.Palette{})
	b := play.NewBoard(boardCells(second...), 80, play.Palette{})

	tty := &syncBuf{}
	live := newPinnedScreen(tty, 24, 80)
	live.interval = -1
	var errb bytes.Buffer
	resizes := make(chan winSize, 1)
	keys := make(chan Key, 8)

	go func() {
		defer close(keys)
		settled := func(cond func() bool) bool {
			for dl := time.Now().Add(2 * time.Second); time.Now().Before(dl); {
				if cond() {
					return true
				}
				time.Sleep(time.Millisecond)
			}
			return false
		}
		if !settled(func() bool { return strings.Contains(unstyled(tty.String()), "[0] keel") }) {
			return
		}
		// Narrow the window while the FIRST board is up...
		resizes <- winSize{rows: 24, cols: 40}
		if !settled(func() bool { return strings.Contains(unstyled(tty.String()), "[3] bank") }) {
			return
		}
		// ...then sweep it, so the SECOND board becomes current after the resize.
		for i := range first {
			keys <- Key{Kind: KeyRune, Rune: rune(play.BoardLabels[i])}
		}
		if !settled(func() bool { return strings.Contains(unstyled(tty.String()), "arrondissement") }) {
			return
		}
		keys <- Key{Kind: KeyInterrupt}
	}()

	playSession(t.Context(), d, opt, play.NewSession([]play.Question{a, b}), held, keys,
		console{view: live, resizes: resizes, finish: func() {}, stdout: live, stderr: &errb})

	// THE PREMISE: the second board was actually reached and drawn.
	frame := unstyled(tty.String())
	if !strings.Contains(frame, "arrondissement") {
		t.Fatalf("the second board was never drawn:\n%s", frame)
	}
	// AND IT FITS. Built at 80, four long words are one row of 74 columns; at 40
	// that wraps into two physical rows and every click below it means another
	// word.
	for i, row := range strings.Split(b.Prompt(), "\n") {
		if n := visibleCells(row); n > 40 {
			t.Errorf("the second board's row %d is %d columns in a 40-column window — it was never told the terminal changed:\n%q", i, n, row)
		}
	}
	if b.Rows() <= 3 {
		t.Errorf("the second board is %d rows; at 40 columns four long words need more, so it did not relayout", b.Rows())
	}
}

// R17: ENTER IS HELD WHILE THE BOARD IS NOT DRAWN IN FULL.
//
// Enter takes every unmarked word as Wrong. On a terminal too short to draw the
// whole board, `fitFooter` drops trailing grid rows — so one keystroke would
// halve the box of words that were never on screen. The loop refuses, the prompt
// says why, and marking what IS visible still works.
func TestEnterIsHeldWhileTheBoardIsNotDrawnInFull(t *testing.T) {
	var words []string
	for _, w := range []string{
		"bank", "concrete", "content", "defenestrate", "desert", "ephemeral",
		"even", "man", "mesa", "minute", "parrot", "present", "pulp", "quokka",
		"read", "run",
	} {
		words = append(words, w)
	}
	d, opt, st := playRig(t, words...)
	_, held := questionsFor(t, d, opt)
	board := play.NewBoard(boardCells(words...), 80, play.Palette{})

	// A window that cannot hold it: the board is six rows plus a prompt and bar.
	const short = 6
	tty := &syncBuf{}
	live := newPinnedScreen(tty, short, 80)
	live.interval = -1
	var errb bytes.Buffer

	keys := make(chan Key, 4)
	keys <- Key{Kind: KeyRune, Rune: rune(play.BoardLabels[0])} // one mark, by hand
	keys <- Key{Kind: KeyEnter}                                 // ...and the sweep, which must not land
	keys <- Key{Kind: KeyInterrupt}
	close(keys)

	playSession(t.Context(), d, opt, play.NewSession([]play.Question{board}), held, keys,
		console{view: live, finish: func() {}, stdout: live, stderr: &errb})

	// THE PREMISE: the terminal really is too short for this board.
	if fitsABoard(short, board.Rows(), displayRows(gradePrompt(board), 80)) {
		t.Fatalf("a %d-row window fits a %d-row board; this test asserts nothing", short, board.Rows())
	}
	// ONE event — the mark. Not sixteen.
	evs := reviewEvents(t, st)
	if len(evs) != 1 {
		t.Fatalf("%d review events, want the one mark — Enter swept words the window never drew:\n%s",
			len(evs), unstyled(tty.String()))
	}
	if evs[0].Word != words[0] {
		t.Errorf("recorded %q, want the word that was marked by hand", evs[0].Word)
	}
	// ...and the learner is told why, on the row Paint clips last.
	if !strings.Contains(unstyled(tty.String()), "window too short") {
		t.Errorf("Enter was refused silently:\n%s", unstyled(tty.String()))
	}

	// AND THE SAME BOARD IN A WINDOW THAT FITS DOES commit, so the refusal is
	// about the window and not about boards.
	d2, opt2, st2 := playRig(t, words...)
	_, held2 := questionsFor(t, d2, opt2)
	board2 := play.NewBoard(boardCells(words...), 80, play.Palette{})
	tty2 := &syncBuf{}
	live2 := newPinnedScreen(tty2, 24, 80)
	live2.interval = -1
	keys2 := make(chan Key, 3)
	keys2 <- Key{Kind: KeyRune, Rune: rune(play.BoardLabels[0])}
	keys2 <- Key{Kind: KeyEnter}
	keys2 <- Key{Kind: KeyInterrupt}
	close(keys2)
	playSession(t.Context(), d2, opt2, play.NewSession([]play.Question{board2}), held2, keys2,
		console{view: live2, finish: func() {}, stdout: live2, stderr: &errb})
	if n := len(reviewEvents(t, st2)); n != len(words) {
		t.Errorf("%d events in a window that fits, want all %d — Enter must still commit there", n, len(words))
	}
}

// readmeBoardWords are the words the README's board example shows, and the
// README DERIVES its grid from them (BR-18).
//
// The block used to be hand-drawn, and it went stale the moment the operator's
// sitting changed the design: it still showed `[y] keel` for a mark standing
// where the key was, and a label row with the old hole at `d` — both contradicted
// by the README's own prose eight lines below. doc_sync pins the prompt LINE, so
// the grid was a restatement with no consumer.
var readmeBoardWords = []string{
	"arrondissement", "bailiwick", "keel", "mesa",
	"ephemeral", "quokka", "potassium", "ligament",
	"sycophantic", "concrete", "parrot", "run",
	"light", "bank", "set", "obsequious",
}

// THE REFUSAL ROW IS NO WIDER THAN THE KEYS ROW IT REPLACES.
//
// `Paint` charges the frame for the prompt it is given, and `boardFitsIn`
// computes the fit from the KEYS row — so a taller replacement would drop one
// more footer row than the fit was computed against. Measured before the fix:
// the keys row is 78 columns and the refusal was 79, which disagreed at 78, 39
// and 26 columns.
//
// Bounded to cosmetics either way — Enter is already held in that state and an
// unpainted row is unclickable — but "these two strings are the same width" is
// not a fact anyone re-checks by eye.
func TestTheRefusalRowIsNoWiderThanTheKeysRow(t *testing.T) {
	board := play.NewBoard(boardCells("keel", "mesa", "run", "bank"), 80, play.Palette{})
	keys := boardPrompt(board, true)
	refusal := boardPrompt(board, false)
	if refusal == keys {
		t.Fatal("the two prompts are identical; this test asserts nothing")
	}
	if visibleCells(refusal) > visibleCells(keys) {
		t.Errorf("the refusal is %d columns and the keys row is %d — the frame is budgeted "+
			"for the keys row, so a wider refusal drops a footer row the fit did not account for:\n\t%q\n\t%q",
			visibleCells(refusal), visibleCells(keys), refusal, keys)
	}
	// ...and at every width the board is offered at, the refusal costs no MORE
	// rows than the budget was computed for. Fewer is fine and is the safe
	// direction — the frame then has a row it did not spend.
	for _, w := range []int{minWrapWidth, 24, 26, 39, 40, 78, defaultCols, 120} {
		if a, b := displayRows(keys, w), displayRows(refusal, w); b > a {
			t.Errorf("at %d columns the keys row is %d rows and the refusal is %d — the frame is "+
				"budgeted for the first and would draw the second", w, a, b)
		}
	}
}

// ONE FIT FORMULA, asked at selection and at draw (BR-19).
//
// It was spelled twice and had already diverged: the selection copy refused a
// terminal under minWrapWidth and the draw-time copy did not, so a board
// narrowed below that by a resize still reported itself whole. Not reachable as
// harm — the row arithmetic turns the answer false well before the words become
// unreadable — which is the reason to consolidate rather than a reason not to.
func TestSelectionAndDrawAskTheSameFitQuestion(t *testing.T) {
	words := []string{"arrondissement", "sycophantic", "defenestrate", "ephemeral"}
	for _, tc := range []struct{ rows, cols int }{
		{24, 80}, {10, 80}, {8, 80}, {24, 40}, {10, 40}, {60, 19}, {60, 12}, {5, 80},
	} {
		opt := options{width: tc.cols, rows: tc.rows}
		atSelection := boardFits(words, opt)
		probe := play.NewBoard(boardCells(words...), tc.cols, play.Palette{})
		atDraw := boardFitsIn(probe, tc.rows, tc.cols)
		if atSelection != atDraw {
			t.Errorf("%dx%d: selection says %v and the frame says %v — two spellings of one formula",
				tc.rows, tc.cols, atSelection, atDraw)
		}
	}
	// THE WIDTH RULE REACHES THE DRAW, which is the divergence that existed.
	narrow := play.NewBoard(boardCells(words...), minWrapWidth-1, play.Palette{})
	if boardFitsIn(narrow, 100, minWrapWidth-1) {
		t.Errorf("a %d-column terminal reports a whole board; below minWrapWidth this program "+
			"treats the terminal as too narrow to lay text out at all", minWrapWidth-1)
	}
}

// PLAYBACK IN A SITTING MUST COMMIT NOTHING TO THE BUFFER (#44).
//
// The indicator is ephemeral UI and takes its own line back — `screen.Write`
// splits on the erase gesture and `eraseOpenLine` drops the line it was writing.
// But `defaultIndicator` also writes a newline BEFORE it, and inside an
// append-only buffer that newline is CONTENT the erase cannot reach: it is a
// completed line by the time the erase arrives, and a completed line is
// scrollback by definition.
//
// Operator, 2026-09-02: "after clicking on pronunciation in the daily play, one
// additional line's inserted".
//
// DRIVEN THROUGH THE REVEAL, not the click. The click is where it was SEEN and
// the reveal is where it lives — every answered question that plays audio pays
// one row, so a sitting drifts up the screen on its own with nobody clicking
// anything.
//
// MEASURED AS A DIFFERENCE, which is what makes the assertion sharp: the same
// sitting is run twice against the same deck, audible and silent, so the content
// is identical and playback is the only variable. Counting blank lines instead
// would be counting the dictionary's own — a rendered entry is full of them.
func TestSittingPlaybackCommitsNothingToTheBuffer(t *testing.T) {
	// TWO questions at least, because one leftover row is indistinguishable from
	// ordinary spacing — which is how this shipped.
	const words = 6
	run := func(t *testing.T, wantAudio bool) int {
		t.Helper()
		d, opt, _ := playRig(t, "quokka", "mesa", "parrot", "bank", "keel", "run")
		var fp *fakePlayer
		if wantAudio {
			fp = audible(&d, &opt)
		}
		qs, held := questionsFor(t, d, opt)
		if len(qs) < 2 {
			t.Fatalf("need two questions to see a per-playback leak, got %d", len(qs))
		}
		tty := &syncBuf{}
		live := newPinnedScreen(tty, 24, opt.width)
		live.interval = -1
		var errb bytes.Buffer

		script := ""
		for _, q := range qs {
			script += "\r" + gradeKey(t, q, play.Correct)
		}
		playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor(script),
			console{view: live, finish: func() {}, stdout: live, stderr: &errb})

		if wantAudio {
			if len(fp.Played) < 2 {
				t.Fatalf("played %d times; this test cannot see the leak it is about", len(fp.Played))
			}
			for _, l := range live.s.Lines() {
				if strings.Contains(unstyled(l), "playing") {
					t.Errorf("the indicator itself survived in the buffer: %q", l)
				}
			}
		}
		return len(live.s.Lines())
	}

	silent := run(t, false)
	audible := run(t, true)
	if audible != silent {
		t.Errorf("the buffer is %d lines with playback and %d without, over a deck of %d — "+
			"playback writes a newline its erase cannot take back, so a sitting drifts "+
			"up the screen by one row per answered question",
			audible, silent, words)
	}
}

// THE GAP NEVER CHANGES WHETHER A BOARD FITS (#44 PQ-8).
//
// `boardFitsIn` is asked at SELECTION and at every DRAW, where its answer decides
// whether Enter may spend the board (R17). `Paint` decides the gap separately, so
// the two could disagree — a board drawn whole and refused in the same breath.
// They cannot, and this is the proof by exhaustion rather than by argument:
// charging the gap and not charging it are the same predicate at every shape,
// because `grantedGap` only hands out a row there was already slack for.
//
// If this ever fails, `fitsABoard` and `grantedGap` have drifted and the visible
// symptom is `boardRefusal` printed over a board with every cell on screen.
func TestTheChromeGapNeverChangesWhetherABoardFits(t *testing.T) {
	for termRows := 0; termRows <= 40; termRows++ {
		for boardRows := 1; boardRows <= 30; boardRows++ {
			for promptRows := 1; promptRows <= 5; promptRows++ {
				footerRows := boardRows + barRows
				charged := footerRows+promptRows+
					grantedGap(chromeGap, termRows, promptRows, footerRows) <= termRows
				if got := fitsABoard(termRows, boardRows, promptRows); got != charged {
					t.Fatalf("termRows=%d boardRows=%d promptRows=%d: fitsABoard=%v but "+
						"charging the gap gives %v — the draw and the fit disagree, so a board "+
						"is drawn whole and refused at once",
						termRows, boardRows, promptRows, got, charged)
				}
			}
		}
	}
}

// THE CHROME BAND IS DIMMED, AND BOTH ROWS OF IT ARE (#44).
//
// The action row and the bar are one band: dimming only the first would leave the
// figure line brighter than the controls above it, which inverts what they are
// worth. Operator, 2026-09-02: *"should be colorized to make the border clear"*.
//
// Driven through the FRAME rather than by calling asChrome, because the claim is
// the wiring — asChrome could be perfect and unreferenced at either Draw site,
// which is exactly the half an earlier draft of this work missed (PQ-6).
func TestTheChromeBandIsDimmedTogether(t *testing.T) {
	for _, tc := range []struct {
		name  string
		color bool
		board bool
	}{
		// `--play` refuses -no-color (BR-3), so the colourless row is belt — but a
		// rig running colourless is how #40's wrap Critical stayed invisible, so
		// both are driven.
		{"a coloured sitting", true, false},
		{"no palette at all", false, false},
		// THE BOARD'S BAR IS A SECOND SITE. It reaches the frame through
		// `boardFooter` rather than the inline `[]string{sittingBar(fig)}`, and
		// an earlier draft of this work styled one and not the other (PQ-6) — so
		// covering only the common sitting would leave exactly the half that was
		// missed before.
		{"a board's bar", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, opt, _ := playRig(t, "quokka", "mesa", "parrot", "bank")
			opt.color = tc.color
			qs, held := questionsFor(t, d, opt)
			if tc.board {
				qs = []play.Question{play.NewBoard(boardCells("quokka", "mesa"), opt.width, boardPalette(opt))}
			}

			tty := &syncBuf{}
			live := newPinnedScreen(tty, 24, opt.width)
			live.interval = -1
			var errb bytes.Buffer
			playSession(t.Context(), d, opt, play.NewSession(qs[:1]), held,
				keysFor("\r"+gradeKey(t, qs[0], play.Correct)),
				console{view: live, finish: func() {}, stdout: live, stderr: &errb})

			// EVERY frame the sitting painted, not just the last: the live edge
			// changes between reveal, graded and done, and a rule that held on one
			// of them and not the others is exactly the gap this is about. The
			// last frame is the summary, where there is no action row at all.
			out := tty.String()
			// The bar is matched by its PROGRESS PREFIX, not by "reviews/day": the
			// summary `finish` writes carries that phrase too, and the summary is
			// the record rather than the live edge — it is correctly undimmed, so
			// a looser matcher fails on content that is behaving.
			barRe := regexp.MustCompile(`\d+ of \d+ · .*?reviews/day`)
			// THE ESCAPE IS ANCHORED TO THE TEXT, not merely present on the row.
			// `Paint` walks the cursor back and REPRINTS the prompt with no
			// newline between, so one "\n"-split line carries the bar AND the
			// prompt — and "this line contains a dim" then passed on the prompt's
			// dim while the bar had none. That made the board's row of this table
			// vacuous, which is the failure a premise check exists to catch.
			plainRow := func(line string, find func(string) string) (string, bool) {
				got := find(unstyled(line))
				return got, got != ""
			}
			for _, row := range []struct {
				what string
				find func(string) string
			}{
				{"the action row", func(l string) string {
					if p := gradePrompt(qs[0]); strings.Contains(l, p) {
						return p
					}
					return ""
				}},
				{"the bar", barRe.FindString},
			} {
				// THE ESCAPE COMES FROM THE PALETTE, not spelled here: newPalette
				// owns the sequence, and a second speller is how the two come to
				// disagree (ARCH-DRY).
				dim := newPalette(true).dim
				var seen bool
				for _, line := range strings.Split(out, "\n") {
					text, ok := plainRow(line, row.find)
					if !ok {
						continue
					}
					seen = true
					if got := strings.Contains(line, dim+text); got != tc.color {
						t.Errorf("%s dimmed = %v, want %v — the band must read as chrome in a "+
							"sitting and carry no escape without a palette:\n\t%q",
							row.what, got, tc.color, line)
						break
					}
				}
				if !seen {
					t.Fatalf("%s was never painted:\n%s", row.what, unstyled(out))
				}
			}
		})
	}
}

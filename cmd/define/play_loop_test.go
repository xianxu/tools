package main

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/play"
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
	return d, options{color: false, width: 0, count: 20, times: 1, noAudio: true}, st
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
func questionsFor(t *testing.T, d deps, opt options) ([]play.Question, sittingDeck) {
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

	code := runPlay(t.Context(), d, opt, strings.NewReader(""), &out, &errb)

	if code != 0 {
		t.Errorf("exit = %d, want 0 — an empty sitting is not a failure", code)
	}
	if !strings.Contains(out.String(), "the deck is empty") {
		t.Errorf("stdout = %q, want it to name the empty deck rather than the schedule", out.String())
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
// The owner used to be crlfWriter wrapped around stdout. It is now Paint, which
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
func paintedSitting(t *testing.T, d deps, opt options, qs []play.Question, held sittingDeck,
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
	for _, footer := range view.menus {
		if n := footer[0][:strings.Index(footer[0], " ·")]; len(counters) == 0 || counters[len(counters)-1] != n {
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
func TestDroppingAWordLowersTheCostTheBarShows(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic", "ephemeral", "quokka", "mesa")
	qs, held := questionsFor(t, d, opt)
	before := held.figures(opt.count).load

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), held, keysFor("d^"), playbackConsole(&out, &errb))

	if after := held.figures(opt.count).load; after >= before {
		t.Errorf("the deck cost %.3f before the drop and %.3f after — dropping a word must "+
			"lower what the deck costs, or the bar is charging for a word that is gone", before, after)
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
		d, _, _ := playRig(t) // no words: the empty-deck message proves runPlay ran
		var out, errb bytes.Buffer
		code := run(t.Context(), []string{"-no-audio", "--play"},
			d, strings.NewReader(""), &out, &errb)
		if code != 0 {
			t.Fatalf("exit = %d, want 0; stderr=%q", code, errb.String())
		}
		if !strings.Contains(out.String(), "the deck is empty") {
			t.Errorf("stdout = %q — --play did not reach runPlay", out.String())
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
			playSession(t.Context(), d, opt, play.NewSession([]play.Question{q}), sittingDeck{}, keysFor(tc.key), playbackConsole(&out, &errb))

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
			playSession(t.Context(), d, opt, play.NewSession([]play.Question{q}), sittingDeck{}, keysFor(tc.keys), playbackConsole(&out, &errb))

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
	playSession(t.Context(), d, opt, play.NewSession([]play.Question{q}), sittingDeck{}, keysFor("y"), playbackConsole(&out, &errb))

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

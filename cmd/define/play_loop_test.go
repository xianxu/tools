package main

import (
	"bytes"
	"context"
	"os"
	"strings"
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

func questionsFor(t *testing.T, d deps, opt options) []play.Question {
	t.Helper()
	var out, errb bytes.Buffer
	qs, code := todaysQuestions(d, opt, &out, &errb)
	if code != 0 {
		t.Fatalf("todaysQuestions = %d, stderr %s", code, errb.String())
	}
	return qs
}

// THE DONE-WHEN: a full session against a fake store and fake clock records one
// event per answer.
func TestFullSessionRecordsOneEventPerAnswer(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic", "ephemeral")
	qs := questionsFor(t, d, opt)
	if len(qs) != 2 {
		t.Fatalf("got %d questions, want 2", len(qs))
	}

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\ry\rn"), rawTerm{}, &out, &errb)

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
	qs := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	// Answer the first, then Ctrl-C before the second.
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\ry^"), rawTerm{}, &out, &errb)

	events := reviewEvents(t, st)
	if len(events) != 1 {
		t.Fatalf("got %d events after an interrupt, want the 1 already answered", len(events))
	}
	if !events[0].Correct {
		t.Error("the preserved event has the wrong verdict")
	}
}

// THE DONE-WHEN: a full session with the LLM seam unavailable.
//
// Asserted with a client that fails, not by unsetting an env var — the point is
// that the loop degrades, and form 2.1 needs no model at all, so it should never
// reach for one.
func TestSessionRunsWithTheModelUnavailable(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic")
	// A seam that RESOLVES but fails on every call — the Done-when is explicit
	// that this is asserted with a failing client, not by unsetting an env var:
	// the point is that the loop degrades, not that config resolution does.
	d.getenv = envFor("http://127.0.0.1:1") // nothing listens there
	d.newLLM = llm.New
	qs := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\ry"), rawTerm{}, &out, &errb)

	if len(reviewEvents(t, st)) != 1 {
		t.Error("the session did not complete with no model configured")
	}
}

// "Nothing due today" is the expected state most days, not an error.
func TestEmptyQueueExitsZero(t *testing.T) {
	d, opt, _ := playRig(t) // no words at all
	var out, errb bytes.Buffer

	code := runPlay(t.Context(), d, opt, strings.NewReader(""), &out, &errb)

	if code != 0 {
		t.Errorf("exit = %d, want 0 — nothing due is not a failure", code)
	}
	if !strings.Contains(out.String(), "nothing due") {
		t.Errorf("stdout = %q, want a line saying nothing is due", out.String())
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
	qs := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	// Reveal, then a key form 2.1 does not grade, then interrupt.
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\rz^"), rawTerm{}, &out, &errb)

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
	qs := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\ry"), rawTerm{}, &out, &errb)

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
		qs := questionsFor(t, d, opt)
		spy := &countingCapturer{}
		d.capture = spy

		// Pre-revealed: the very first input is a GRADE, so a single select win
		// is immediately observable.
		s := play.NewSession(qs)
		s, _ = play.Apply(s, play.Input{Kind: play.InputReveal})

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		var out, errb bytes.Buffer
		code := playSession(ctx, d, opt, s, keysFor("yyyy"), rawTerm{}, &out, &errb)

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

	qs := questionsFor(t, d, opt)

	if len(qs) != 1 || qs[0].Word() != "sycophantic" {
		var got []string
		for _, q := range qs {
			got = append(got, q.Word())
		}
		t.Errorf("queue = %v, want only sycophantic — ephemeral was just reviewed", got)
	}
}

// EVERY newline a session writes must be a full CRLF.
//
// This is the defect the first version of --play shipped, and it is worth
// recording WHY the pty smoke test missed it: that test captured the bytes and
// printed them through Python, where a bare \n renders at column 0 and the
// output looked perfect. A real terminal in raw mode does not — it moves down
// and stays put — so the definition cascaded diagonally across the screen, each
// line starting where the last one ended.
//
// A byte capture is not a screenshot. What can be asserted about bytes is this:
// in raw mode there is no such thing as a bare newline.
func TestSessionOutputIsAllCRLF(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic", "ephemeral")
	qs := questionsFor(t, d, opt)

	var raw, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs),
		keysFor("\ry\rn"), rawTerm{}, &crlfWriter{w: &raw}, &errb)

	got := raw.String()
	if strings.Count(got, "\n") == 0 {
		t.Fatal("no newlines at all — this test would assert nothing")
	}
	if bare := strings.Count(got, "\n") - strings.Count(got, "\r\n"); bare != 0 {
		t.Errorf("%d bare newlines in session output — in raw mode each one starts the next line "+
			"where the last ended, which is the diagonal cascade", bare)
	}
	// And the definition itself must be in there, or the assertion above is
	// ranging over prompts alone.
	if !strings.Contains(got, "adjective") {
		t.Error("the rendered definition never reached the writer")
	}
}

// Audio plays before reveal by DEFAULT, which is the Spec and which no test
// exercised: playRig sets noAudio, so the whole playback branch was at zero
// coverage. fakePlayer is the seam — playAnnounced shells out to afplay(1), so a
// real player in a test is a real process.
func TestRevealPlaysThePronunciationByDefault(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	opt.noAudio = false // the default; playRig turns it off for every other test
	player := &fakePlayer{}
	d.player = player
	d.audio = okAudio{}
	qs := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\r^"), rawTerm{}, &out, &errb)

	if len(player.Played) == 0 {
		t.Error("revealing did not play the pronunciation, and audio is on by default")
	}
}

// ...and -no-audio silences it.
func TestNoAudioSilencesTheSession(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	opt.noAudio = true
	player := &fakePlayer{}
	d.player = player
	d.audio = okAudio{}
	qs := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\r^"), rawTerm{}, &out, &errb)

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

	qs := questionsFor(t, d, opt)

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
	qs := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	// Drop the first word, then quit.
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("d^"), rawTerm{}, &out, &errb)

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
	qs := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("d"), rawTerm{}, &out, &errb)

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
	qs, code := todaysQuestions(d, opt, &out, &errb)

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

// Losing the terminal after playback exits 1, like failing to enter raw mode in
// the first place — the same failure class, and returning 0 from one of them
// tells a script the session ended normally when it did not.
//
// Driven by handing the session a rawTerm whose file is NOT a terminal, so the
// re-entry after playback genuinely fails.
func TestLosingTheTerminalAfterPlaybackExitsOne(t *testing.T) {
	d, opt, _ := playRig(t, "sycophantic")
	opt.noAudio = false
	d.player = &fakePlayer{}
	d.audio = okAudio{}
	qs := questionsFor(t, d, opt)

	notATerminal, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer notATerminal.Close()

	var out, errb bytes.Buffer
	code := playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\r"),
		rawTerm{sess: &rawSession{}, f: notATerminal}, &out, &errb)

	if code != 1 {
		t.Errorf("exit = %d, want 1 — the same code as failing to enter raw mode at all", code)
	}
	if !strings.Contains(errb.String(), "lost the terminal") {
		t.Errorf("stderr = %q, want it to say the terminal was lost", errb.String())
	}
}

// missingDict answers every lookup with "no entry" — a deck whose words the
// dictionary no longer knows.
type missingDict struct{}

func (missingDict) Lookup(word string) (string, error) {
	return "", ErrNoEntry
}

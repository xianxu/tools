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
	qs := questionsFor(t, d, opt)

	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\ry"), rawTerm{}, &out, &errb)

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
	player := audible(&d, &opt) // the default; playRig turns audio off for every other test
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
	player := audible(&d, &opt)
	opt.noAudio = true // ...and THEN silence it: the flag, not an unreachable source
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
	audible(&d, &opt)
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

	qs := questionsFor(t, d, opt)
	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("y"), rawTerm{}, &out, &errb)

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

	qs := questionsFor(t, d, opt)
	var out, errb bytes.Buffer
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("n^"), rawTerm{}, &out, &errb)

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
			var b bytes.Buffer
			draw(&b, tc.s)
			if !strings.Contains(b.String(), tc.want) {
				t.Errorf("prompt = %q, want it to contain %q", b.String(), tc.want)
			}
			if strings.Contains(b.String(), tc.absent) {
				t.Errorf("prompt = %q, must NOT offer %q in this state", b.String(), tc.absent)
			}
		})
	}
}

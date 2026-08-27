package main

import (
	"bytes"
	"context"
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
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\ry\rn"), nil, &out, &errb)

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
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\ry^"), nil, &out, &errb)

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
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\ry"), nil, &out, &errb)

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
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\rz^"), nil, &out, &errb)

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
	playSession(t.Context(), d, opt, play.NewSession(qs), keysFor("\ry"), nil, &out, &errb)

	if spy.reviews != 1 {
		t.Errorf("CaptureReview called %d times for one graded answer, want 1", spy.reviews)
	}
}

// A cancelled context ends the session, and what was recorded stays.
func TestCancelledContextEndsTheSession(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic", "ephemeral")
	qs := questionsFor(t, d, opt)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var out, errb bytes.Buffer
	code := playSession(ctx, d, opt, play.NewSession(qs), keysFor("\ry"), nil, &out, &errb)

	if code != 0 {
		t.Errorf("exit = %d, want 0 — an interrupted session is not a failure", code)
	}
	if len(reviewEvents(t, st)) != 0 {
		t.Error("recorded an event after the context was already cancelled")
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

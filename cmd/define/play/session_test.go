package play

import "testing"

func rune_(r rune) Input { return Input{Kind: InputRune, Rune: r} }

var reveal = Input{Kind: InputReveal}
var quit = Input{Kind: InputQuit}

// drive runs a whole input sequence and collects what the loop was told to do.
func drive(s Session, ins ...Input) (Session, []Outcome) {
	var out []Outcome
	for _, in := range ins {
		var os []Outcome
		s, os = Apply(s, in)
		out = append(out, os...)
	}
	return s, out
}

// only asserts Apply owed the loop exactly ONE thing and returns it.
//
// Most inputs do owe exactly one; the miss-before-reveal branch is the exception
// and its tests read the slice directly. Asserting the count here means a site
// that silently grew a second outcome fails rather than ignoring it.
func only(t *testing.T, outs []Outcome) Outcome {
	t.Helper()
	if len(outs) != 1 {
		t.Fatalf("outcomes = %+v, want exactly one", outs)
	}
	return outs[0]
}

func records(outs []Outcome) []Outcome {
	var rec []Outcome
	for _, o := range outs {
		if o.Kind == OutcomeRecord {
			rec = append(rec, o)
		}
	}
	return rec
}

func twoQuestions() Session {
	return NewSession([]Question{
		NewRecall("obsequious", "fawning"),
		NewRecall("ephemeral", "short-lived"),
	})
}

func TestRevealThenGradeAdvancesAndRecords(t *testing.T) {
	s, outs := drive(twoQuestions(), reveal, rune_('y'))

	rec := records(outs)
	if len(rec) != 1 {
		t.Fatalf("got %d record outcomes, want 1: %+v", len(rec), outs)
	}
	if rec[0].Word != "obsequious" || rec[0].Verdict != Correct {
		t.Errorf("recorded %+v, want obsequious/Correct", rec[0])
	}
	if s.Index != 1 {
		t.Errorf("index = %d, want 1 — the session did not advance", s.Index)
	}
	if s.Revealed {
		t.Error("still revealed on the next question — the answer is showing before it is asked")
	}
	if s.Right != 1 {
		t.Errorf("Right = %d, want 1", s.Right)
	}
}

// A RECALL test is rated by the learner, not by the screen.
//
// This replaces TestGradingBeforeRevealIsIgnored, whose name WAS the claim #24
// reverses: "a learner cannot rate what they have not seen". True of a
// recognition test, false of this one — the learner knows their own recall
// before they check, and the definition is feedback rather than stimulus.
func TestCorrectBeforeRevealAdvancesWithNoReveal(t *testing.T) {
	next, outs := Apply(twoQuestions(), rune_('y'))

	if len(outs) != 1 || outs[0].Kind != OutcomeRecord {
		t.Fatalf("outcomes = %+v, want exactly one OutcomeRecord — a correct answer earns no reveal", outs)
	}
	if outs[0].Verdict != Correct || outs[0].Word != "obsequious" {
		t.Errorf("recorded %v for %q, want Correct for \"obsequious\"", outs[0].Verdict, outs[0].Word)
	}
	if next.Index != 1 {
		t.Errorf("Index = %d, want 1 — y advances", next.Index)
	}
	if next.Revealed || next.Graded {
		t.Errorf("Revealed=%v Graded=%v, want both false on the NEXT question", next.Revealed, next.Graded)
	}
	if next.Right != 1 || next.Wrong != 0 {
		t.Errorf("tally = %d right %d wrong, want 1/0", next.Right, next.Wrong)
	}
}

// A miss earns the definition, and earns it WITHOUT advancing.
func TestWrongBeforeRevealRecordsAndReveals(t *testing.T) {
	next, outs := Apply(twoQuestions(), rune_('n'))

	if len(outs) != 2 {
		t.Fatalf("outcomes = %+v, want two: the record and the reveal", outs)
	}
	// ORDER matters: recorded before anything that can block on the terminal.
	if outs[0].Kind != OutcomeRecord || outs[0].Verdict != Wrong {
		t.Errorf("outs[0] = %+v, want the Wrong record first", outs[0])
	}
	if outs[1].Kind != OutcomeReveal {
		t.Errorf("outs[1] = %+v, want the reveal second", outs[1])
	}
	if next.Index != 0 {
		t.Errorf("Index = %d, want 0 — a miss stays on the word so the answer can be read", next.Index)
	}
	if !next.Revealed || !next.Graded {
		t.Errorf("Revealed=%v Graded=%v, want both true", next.Revealed, next.Graded)
	}
	// PQ-1: scored WITHOUT advancing. The first draft recorded the event and left
	// the tally alone, so three misses ended "0 right, 0 wrong".
	if next.Wrong != 1 {
		t.Errorf("Wrong = %d, want 1 — a miss counts even though it does not advance", next.Wrong)
	}
}

// The verdict is recorded ONCE. Answering again is the learner moving on, not a
// second assessment — Fold would read a duplicate as another review.
func TestAKeyAfterAMissAdvancesWithoutRecordingAgain(t *testing.T) {
	s, _ := Apply(twoQuestions(), rune_('n'))

	next, outs := Apply(s, rune_('n'))

	for _, o := range outs {
		if o.Kind == OutcomeRecord {
			t.Errorf("recorded %+v a second time for the same question", o)
		}
	}
	if next.Index != 1 {
		t.Errorf("Index = %d, want 1", next.Index)
	}
	if next.Wrong != 1 {
		t.Errorf("Wrong = %d, want 1 — moving on must not re-score", next.Wrong)
	}
}

// PQ-2: Enter and space are the keys a learner already reaches for, and toInput
// maps BOTH to InputReveal — so without an arm for the graded state they would
// be dead exactly where the prompt says "any key = next word".
func TestEnterAndSpaceMoveOnAfterAMiss(t *testing.T) {
	s, _ := Apply(twoQuestions(), rune_('n'))

	next, outs := Apply(s, reveal)

	if next.Index != 1 {
		t.Errorf("Index = %d, want 1 — Enter/space must move on once the answer is up", next.Index)
	}
	for _, o := range outs {
		if o.Kind == OutcomeRecord {
			t.Errorf("moving on recorded %+v a second time", o)
		}
	}
	if next.Wrong != 1 {
		t.Errorf("Wrong = %d, want 1 — advancing must not re-score", next.Wrong)
	}
}

// Space still reveals without grading, for a learner who wants to check before
// rating. The OLD flow, still available — just no longer mandatory.
func TestRevealWithoutGradingThenGrade(t *testing.T) {
	s, outs := Apply(twoQuestions(), reveal)
	if len(outs) != 1 || outs[0].Kind != OutcomeReveal {
		t.Fatalf("outcomes = %+v, want one OutcomeReveal", outs)
	}
	if s.Graded {
		t.Error("a reveal graded the question; revealing is not answering")
	}

	next, outs := Apply(s, rune_('y'))
	if len(outs) != 1 || outs[0].Kind != OutcomeRecord || outs[0].Verdict != Correct {
		t.Errorf("outcomes = %+v, want one Correct record", outs)
	}
	if next.Index != 1 {
		t.Errorf("Index = %d, want 1 — grading after a peek advances", next.Index)
	}
}

// A key the form does not grade is ignored ENTIRELY — it does not advance, and
// it is not a skip. Form 2.1 grades only y and n.
//
// (The comment here described the skip rule for two rounds after the test was
// renamed away from it: renaming a test does not move the comment above it.)
func TestAnUngradedKeyDoesNotAdvance(t *testing.T) {
	s, outs := drive(twoQuestions(), reveal, rune_('s'))

	// 's' is not a key Recall grades, so it is ignored entirely rather than
	// skipping — this asserts the CURRENT contract: only y/n advance form 2.1.
	if s.Index != 0 {
		t.Errorf("index = %d, want 0 — an ungraded key advanced the session", s.Index)
	}
	if rec := records(outs); len(rec) != 0 {
		t.Errorf("recorded %+v for a key the form does not use", rec)
	}
}

// The verdict-level rule, driven through a form that DOES produce Skipped.
func TestSkippedVerdictRecordsNothing(t *testing.T) {
	// fakeForm's '3' is Skipped — a second fixture existing only to produce that
	// verdict was one double too many.
	s := NewSession([]Question{&fakeForm{word: "alpha"}})

	next, outs := drive(s, reveal, rune_('3'))

	if rec := records(outs); len(rec) != 0 {
		t.Errorf("a Skipped verdict was recorded: %+v", rec)
	}
	if next.Index != 1 {
		t.Errorf("index = %d, want 1 — a skip must still advance", next.Index)
	}
	if next.Right != 0 || next.Wrong != 0 {
		t.Errorf("a skip counted in the tally: right %d wrong %d", next.Right, next.Wrong)
	}
}

func TestAnsweringTheLastQuestionEndsTheSession(t *testing.T) {
	s, outs := drive(twoQuestions(), reveal, rune_('y'), reveal, rune_('n'))

	if !s.Done {
		t.Error("session is not done after the last answer")
	}
	if len(records(outs)) != 2 {
		t.Errorf("got %d records, want 2", len(records(outs)))
	}
	if s.Right != 1 || s.Wrong != 1 {
		t.Errorf("tally right %d wrong %d, want 1/1", s.Right, s.Wrong)
	}
	// And nothing after Done does anything.
	after, outs := Apply(s, rune_('y'))
	o := only(t, outs)
	if o.Kind != OutcomeDone || after.Index != s.Index {
		t.Errorf("input after Done changed things: %+v", o)
	}
}

func TestQuitEndsTheSessionAtAnyPoint(t *testing.T) {
	for _, name := range []string{"before reveal", "after reveal"} {
		t.Run(name, func(t *testing.T) {
			s := twoQuestions()
			if name == "after reveal" {
				s, _ = drive(s, reveal)
			}

			next, outs := Apply(s, quit)
			o := only(t, outs)

			if !next.Done {
				t.Error("quit did not end the session")
			}
			if o.Kind != OutcomeDone {
				t.Errorf("outcome = %v, want OutcomeDone", o.Kind)
			}
		})
	}
}

// Revealing is what triggers the pronunciation, and revealing twice must not
// play it twice.
func TestRevealIsIdempotent(t *testing.T) {
	_, outs := drive(twoQuestions(), reveal, reveal)

	if outs[0].Kind != OutcomeReveal {
		t.Errorf("first reveal = %v, want OutcomeReveal", outs[0].Kind)
	}
	if outs[1].Kind != OutcomeNone {
		t.Errorf("second reveal = %v, want OutcomeNone — the word would play twice", outs[1].Kind)
	}
}

// Every outcome that is ABOUT a word NAMES that word, from both paths that can
// produce a reveal.
//
// The loop plays the pronunciation for whatever OutcomeReveal names. It used to
// read the word back off the session instead — correct only for as long as no
// input both advanced and revealed, and the cost of the first one that did was
// not the wrong word but a nil-interface panic at the end of the queue, where
// Current() returns nil (BR-4). #7's multiple-choice form adds inputs to this
// machine, so the invariant is pinned before it is relied on rather than after.
func TestEveryWordOutcomeNamesItsWord(t *testing.T) {
	// A peek: reveal without grading.
	_, outs := Apply(twoQuestions(), reveal)
	if len(outs) != 1 || outs[0].Kind != OutcomeReveal {
		t.Fatalf("outcomes = %+v, want one OutcomeReveal", outs)
	}
	if outs[0].Word != "obsequious" {
		t.Errorf("a peek revealed %q, want %q — the loop plays what the outcome names",
			outs[0].Word, "obsequious")
	}

	// A miss on a hidden word: records AND reveals, and both name the word.
	_, outs = Apply(twoQuestions(), rune_('n'))
	if len(outs) != 2 {
		t.Fatalf("outcomes = %+v, want a record and a reveal", outs)
	}
	for _, o := range outs {
		if o.Word != "obsequious" {
			t.Errorf("%v names %q, want %q", o.Kind, o.Word, "obsequious")
		}
	}
}

// "Nothing due today" is the expected state most days, not an error.
func TestEmptyQueueIsImmediatelyDone(t *testing.T) {
	s := NewSession(nil)

	if !s.Done {
		t.Error("an empty session is not done")
	}
	if s.Current() != nil {
		t.Error("an empty session has a current question")
	}
}

// THE DONE-WHEN: "adding a second form requires no change to the loop".
//
// This is the only honest way to test that before a second form exists. fakeForm
// grades entirely different keys — digits, not y/n — and produces all three
// verdicts. The same session behaviour must fall out. If any key semantics had
// leaked into Apply, this fails.
func TestSessionIsFormAgnostic(t *testing.T) {
	s := NewSession([]Question{&fakeForm{word: "alpha"}, &fakeForm{word: "beta"}})

	next, outs := drive(s, reveal, rune_('1'), reveal, rune_('2'))

	rec := records(outs)
	if len(rec) != 2 {
		t.Fatalf("got %d records, want 2: %+v", len(rec), outs)
	}
	if rec[0].Word != "alpha" || rec[0].Verdict != Correct {
		t.Errorf("first record %+v, want alpha/Correct", rec[0])
	}
	if rec[1].Word != "beta" || rec[1].Verdict != Wrong {
		t.Errorf("second record %+v, want beta/Wrong", rec[1])
	}
	if !next.Done || next.Right != 1 || next.Wrong != 1 {
		t.Errorf("done %v right %d wrong %d", next.Done, next.Right, next.Wrong)
	}
	// And y/n — form 2.1's keys — mean NOTHING here, which is the proof that the
	// session never learned them.
	ignored := NewSession([]Question{&fakeForm{word: "alpha"}})
	after, outs2 := drive(ignored, reveal, rune_('y'))
	if len(records(outs2)) != 0 || after.Index != 0 {
		t.Error("'y' graded a form that does not use it — key semantics leaked into the session")
	}
}

// fakeForm uses digits and produces every verdict, sharing no key with Recall.
type fakeForm struct{ word string }

func (f *fakeForm) Word() string   { return f.word }
func (f *fakeForm) Prompt() string { return "which one?" }
func (f *fakeForm) Reveal() string { return "it was the first" }
func (f *fakeForm) Keys() string   { return "1 = right, 2 = wrong" }
func (f *fakeForm) Grade(r rune) (Verdict, bool) {
	switch r {
	case '1':
		return Correct, true
	case '2':
		return Wrong, true
	case '3':
		return Skipped, true
	}
	return Skipped, false
}

// Dropping is a SESSION action, not a verdict — so every form gets it, and it
// records no review.
// The third state is new in #24: a learner who just MISSED a word is exactly who
// wants to drop it, and the graded state must not swallow the key.
func TestDropAdvancesRecordsNothingAndNamesTheWord(t *testing.T) {
	for _, when := range []string{"before reveal", "after reveal", "after a miss"} {
		t.Run(when, func(t *testing.T) {
			s := twoQuestions()
			if when == "after reveal" {
				s, _ = drive(s, reveal)
			}
			if when == "after a miss" {
				// 'n', not a digit: twoQuestions() is Recall. Against fakeForm
				// 'n' grades nothing, Graded would never be set, and this row
				// would silently re-run "before reveal" under another name.
				s, _ = drive(s, rune_('n'))
			}

			next, outs := Apply(s, Input{Kind: InputDrop})
			o := only(t, outs)

			if o.Kind != OutcomeDrop {
				t.Fatalf("outcome = %v, want OutcomeDrop", o.Kind)
			}
			if o.Word != "obsequious" {
				t.Errorf("dropped %q, want the CURRENT word", o.Word)
			}
			if next.Index != 1 {
				t.Errorf("index = %d, want 1 — dropping must move on", next.Index)
			}
			// Dropping ITSELF scores nothing. After a miss the tally already
			// holds that miss, and dropping must neither erase it nor add to it:
			// the word leaves the deck, its events stay, which is --forget's
			// contract and the reason OutcomeDrop is not a verdict.
			wantWrong := 0
			if when == "after a miss" {
				wantWrong = 1
			}
			if next.Right != 0 || next.Wrong != wantWrong {
				t.Errorf("tally = %d/%d, want 0/%d — dropping scores nothing of its own",
					next.Right, next.Wrong, wantWrong)
			}
		})
	}
}

// It works for a form that shares no keys with Recall, because it is the
// session's action rather than the form's.
func TestDropWorksForAnyForm(t *testing.T) {
	s := NewSession([]Question{&fakeForm{word: "alpha"}})

	next, outs := Apply(s, Input{Kind: InputDrop})
	o := only(t, outs)

	if o.Kind != OutcomeDrop || o.Word != "alpha" {
		t.Errorf("outcome %+v, want a drop of alpha", o)
	}
	if !next.Done {
		t.Error("dropping the last question did not end the session")
	}
}

// A caller holding only an Outcome must be able to tell the session ended.
//
// The last answer produces OutcomeRecord and finishes the queue, so Kind alone
// says "record this" and nothing about the end — a consumer would have to
// consult the Session too, making "did we finish" two facts in two places.
func TestTheOutcomeThatEndsTheSessionSaysSo(t *testing.T) {
	// One question: revealing then grading it is both a record AND the end.
	s := NewSession([]Question{NewRecall("obsequious", "fawning")})
	s, _ = drive(s, reveal)

	_, outs := Apply(s, rune_('y'))
	o := only(t, outs)

	if o.Kind != OutcomeRecord {
		t.Fatalf("kind = %v, want OutcomeRecord", o.Kind)
	}
	if !o.SessionDone {
		t.Error("the outcome that ended the session does not say so")
	}
}

func TestAnOutcomeMidSessionDoesNotClaimTheEnd(t *testing.T) {
	s, _ := drive(twoQuestions(), reveal)

	_, outs := Apply(s, rune_('y'))
	o := only(t, outs)

	if o.SessionDone {
		t.Error("an outcome mid-session claims the session ended")
	}
}

func TestQuitAndDropAlsoReportTheEnd(t *testing.T) {
	if _, outs := Apply(twoQuestions(), quit); !only(t, outs).SessionDone {
		t.Error("quit did not report the end")
	}
	last := NewSession([]Question{NewRecall("w", "d")})
	if _, outs := Apply(last, Input{Kind: InputDrop}); !only(t, outs).SessionDone {
		t.Error("dropping the last question did not report the end")
	}
}

// A FORM THAT HOLDS MANY WORDS STAYS CURRENT UNTIL IT IS SPENT.
//
// `advance` moves on after every graded answer, which is right for every form
// that holds one word and wrong for a board. The session must not learn what a
// board IS (#6's Done-when), so it asks — the third instance of the pattern
// `Missed` and `SelfRated` already establish.
//
// Four points need it, not one, and three of them are why this test drives Apply
// rather than advance directly: a `Wrong` mark reaches the miss-on-a-hidden-word
// branch, which sets Graded and would freeze the form; a drop has no single word
// to name; and Enter must spend the form rather than reveal.
func TestABatchFormStaysCurrentUntilSpent(t *testing.T) {
	s := NewSession([]Question{newFakeBatch("alpha", "beta", "gamma")})

	// Two marks, and the session must not have moved.
	for i, r := range []rune{'y', 'n'} {
		var outs []Outcome
		s, outs = Apply(s, Input{Kind: InputRune, Rune: r})
		if s.Index != 0 {
			t.Fatalf("mark %d advanced the session to index %d — the form is not spent", i, s.Index)
		}
		if s.Done {
			t.Fatalf("mark %d ended the session", i)
		}
		if s.Graded {
			t.Fatalf("mark %d set Graded, so the next key means \"next word\" and the form is frozen", i)
		}
		if len(outs) == 0 || outs[0].Kind != OutcomeRecord {
			t.Fatalf("mark %d recorded nothing: %+v", i, outs)
		}
	}

	// The third mark spends it, and only then does the session move.
	s, _ = Apply(s, Input{Kind: InputRune, Rune: 'y'})
	if !s.Done {
		t.Errorf("the form is spent and the session did not finish: index %d done %v", s.Index, s.Done)
	}
}

// A single-word form is spent after one answer, which is every form that exists
// and is the right default for one that does not implement the capability.
func TestASingleWordFormIsSpentAfterOneAnswer(t *testing.T) {
	s := NewSession([]Question{NewRecall("alpha", "a"), NewRecall("beta", "b")})
	s, _ = Apply(s, Input{Kind: InputRune, Rune: 'y'})
	if s.Index != 1 {
		t.Errorf("index = %d after one answer, want 1 — a form that holds one word advances", s.Index)
	}
}

// fakeBatch is the double: N words, one verdict each, spent when they are all
// answered. A test double rather than *Board, so this file keeps asserting what
// the SESSION does rather than what a form does.
type fakeBatch struct {
	words  []string
	marked int
}

func newFakeBatch(words ...string) *fakeBatch { return &fakeBatch{words: words} }

func (b *fakeBatch) Word() string   { return b.words[min(b.marked, len(b.words)-1)] }
func (b *fakeBatch) Prompt() string { return "grid" }
func (b *fakeBatch) Reveal() string { return "" }
func (b *fakeBatch) Keys() string   { return "y = yes, n = no" }
func (b *fakeBatch) Grade(r rune) (Verdict, bool) {
	switch r {
	case 'y':
		b.marked++
		return Correct, true
	case 'n':
		b.marked++
		return Wrong, true
	}
	return Skipped, false
}
func (b *fakeBatch) Spent() bool { return b.marked >= len(b.words) }
func (b *fakeBatch) Rest(v Verdict) []string {
	rest := b.words[b.marked:]
	b.marked = len(b.words)
	return rest
}

// ENTER SPENDS A BATCH FORM AND SPACE MUST NOT.
//
// toInput maps BOTH Enter and space to InputReveal, deliberately — this
// machine's own comment says so. Spending a board on that kind means a casual
// space takes every unmarked word as No, which is the one action on that surface
// that is expensive to undo, on the most careless key there is.
//
// So Enter carries its own kind. For every form that holds ONE word the two are
// equivalent, which is what keeps 2.1 and 2.3 from noticing the split.
func TestEnterSpendsABatchFormAndSpaceDoesNot(t *testing.T) {
	t.Run("space leaves it alone", func(t *testing.T) {
		s := NewSession([]Question{newFakeBatch("alpha", "beta", "gamma")})
		s, outs := Apply(s, Input{Kind: InputReveal})
		if s.Done || s.Index != 0 {
			t.Errorf("space spent the form: index %d done %v", s.Index, s.Done)
		}
		for _, o := range outs {
			if o.Kind == OutcomeRecord {
				t.Errorf("space recorded %q — it marked a word nobody marked", o.Word)
			}
		}
	})

	t.Run("Enter takes the rest as Wrong", func(t *testing.T) {
		s := NewSession([]Question{newFakeBatch("alpha", "beta", "gamma")})
		s, _ = Apply(s, Input{Kind: InputRune, Rune: 'y'}) // one marked by hand
		s, outs := Apply(s, Input{Kind: InputFinish})

		var recorded []string
		for _, o := range outs {
			if o.Kind == OutcomeRecord {
				if o.Verdict != Wrong {
					t.Errorf("%q was committed as %v, want Wrong", o.Word, o.Verdict)
				}
				recorded = append(recorded, o.Word)
			}
		}
		if len(recorded) != 2 {
			t.Errorf("Enter recorded %v, want the two words left unmarked", recorded)
		}
		if !s.Done {
			t.Error("Enter did not spend the form")
		}
	})

	t.Run("a single-word form treats Enter exactly as space", func(t *testing.T) {
		// The equivalence that keeps 2.1 and 2.3 from noticing the split.
		reveal := NewSession([]Question{NewRecall("alpha", "a")})
		finish := NewSession([]Question{NewRecall("alpha", "a")})
		reveal, ro := Apply(reveal, Input{Kind: InputReveal})
		finish, fo := Apply(finish, Input{Kind: InputFinish})
		if reveal.Revealed != finish.Revealed || len(ro) != len(fo) || ro[0].Kind != fo[0].Kind {
			t.Errorf("Enter and space diverged on a one-word form: %+v vs %+v", ro, fo)
		}
	})
}

// `d` IS REFUSED BY A FORM HOLDING MANY WORDS, rather than dropping a guess.
//
// InputDrop advances with Skipped and drops q.Word(). On a grid there is no
// single current word, so q.Word() is whichever cell happens to be next — and
// dropping the wrong word is SILENT and takes it out of the deck. The form does
// nothing instead.
func TestDropIsRefusedByABatchForm(t *testing.T) {
	s := NewSession([]Question{newFakeBatch("alpha", "beta", "gamma")})
	s, outs := Apply(s, Input{Kind: InputDrop})

	for _, o := range outs {
		if o.Kind == OutcomeDrop {
			t.Errorf("a board dropped %q — on a grid `d` names no word", o.Word)
		}
	}
	if s.Done || s.Index != 0 {
		t.Errorf("`d` moved the session on: index %d done %v", s.Index, s.Done)
	}
}

// TAB REACHES A FORM THAT HAS A MODE, AND NOTHING ELSE (D13).
//
// The session must not learn what a board is, so Tab asks a CAPABILITY like the
// three before it. The half that matters for 2.1 and 2.3 is the second subtest:
// a new input kind that quietly advanced them would be a way to scroll a
// definition away mid-read.
func TestTabSwitchesTheModeOfAFormThatHasOne(t *testing.T) {
	t.Run("a moded form is toggled", func(t *testing.T) {
		m := &fakeModed{}
		s := NewSession([]Question{m})
		s, outs := Apply(s, Input{Kind: InputToggle})
		if m.toggles != 1 {
			t.Errorf("Tab toggled %d times, want 1", m.toggles)
		}
		if len(outs) != 1 || outs[0].Kind != OutcomeNone {
			t.Errorf("Tab produced %+v, want one OutcomeNone — the frame redraws and nothing is recorded", outs)
		}
		if s.Index != 0 || s.Done {
			t.Errorf("Tab advanced the session: index %d done %v", s.Index, s.Done)
		}
	})

	t.Run("a form without a mode is untouched", func(t *testing.T) {
		s := NewSession([]Question{NewRecall("keel", "the bottom of a ship")})
		s, outs := Apply(s, Input{Kind: InputToggle})
		if len(outs) != 1 || outs[0].Kind != OutcomeNone {
			t.Errorf("Tab on form 2.1 produced %+v, want nothing", outs)
		}
		if s.Index != 0 || s.Done || s.Revealed || s.Graded {
			t.Errorf("Tab moved form 2.1: %+v", s)
		}
	})

	t.Run("Tab does not mean next word once a verdict is in", func(t *testing.T) {
		// "any key = next word" deliberately excludes it: a board is never
		// Graded, so the only thing Tab could advance is a definition the
		// learner is still reading.
		s := NewSession([]Question{NewRecall("keel", "the bottom of a ship"), NewRecall("mesa", "a flat-topped hill")})
		s, _ = Apply(s, Input{Kind: InputRune, Rune: 'n'})
		if !s.Graded {
			t.Fatal("a miss on a hidden word did not set Graded")
		}
		s, _ = Apply(s, Input{Kind: InputToggle})
		if s.Index != 0 {
			t.Error("Tab advanced past a definition that was still on screen")
		}
	})
}

// fakeModed is the double: a form with a mode and nothing else. A double rather
// than *Board so this file keeps asserting what the SESSION does.
type fakeModed struct {
	toggles int
	mode    Mark
}

func (m *fakeModed) Word() string   { return "keel" }
func (m *fakeModed) Prompt() string { return "keel" }
func (m *fakeModed) Reveal() string { return "" }
func (m *fakeModed) Keys() string   { return "y = yes, n = no" }
func (m *fakeModed) Grade(rune) (Verdict, bool) {
	return Skipped, false
}
func (m *fakeModed) Mode() Mark { return m.mode }
func (m *fakeModed) Toggle() {
	m.toggles++
	if m.mode == Yes {
		m.mode = No
		return
	}
	m.mode = Yes
}

// AND IT NEVER ANSWERS A FORM THAT DID NOT ASK FOR IT. #38's invariant, in the
// form it takes once one form does ask: every other form declines the kind, so
// its row stays green untouched.
func TestAClickDoesNotAnswerANonGridForm(t *testing.T) {
	for _, q := range []Question{
		NewRecall("keel", "the bottom of a ship"),
		NewChoice("keel", "", []Option{{Gloss: "the bottom of a ship", Correct: true}, {Gloss: "a flat-topped hill"}}),
		newFakeBatch("alpha", "beta"), // holds many words, but draws no cells
	} {
		s := NewSession([]Question{q, NewRecall("mesa", "a flat-topped hill")})
		s, outs := Apply(s, Input{Kind: InputMark, Cell: 0})
		if len(outs) != 1 || outs[0].Kind != OutcomeNone {
			t.Errorf("%T: a click produced %+v, want nothing", q, outs)
		}
		if s.Index != 0 || s.Revealed || s.Graded || s.Right != 0 || s.Wrong != 0 {
			t.Errorf("%T: a click moved the session: %+v", q, s)
		}
	}
}

// fakeGrid is a Grid that is NOT a Batch: one word, drawn as one cell. It exists
// because Grid and Batch are separate capabilities and the board happens to be
// both — so the board cannot show what the machine does with a form that is only
// one of them.
type fakeGrid struct{ marked bool }

func (g *fakeGrid) Word() string                { return "keel" }
func (g *fakeGrid) Prompt() string              { return "[0] keel" }
func (g *fakeGrid) Reveal() string              { return "" }
func (g *fakeGrid) Keys() string                { return "0 = mark" }
func (g *fakeGrid) Grade(rune) (Verdict, bool)  { return Skipped, false }
func (g *fakeGrid) Rows() int                   { return 1 }
func (g *fakeGrid) CellAt(int, int) (int, bool) { return 0, true }
func (g *fakeGrid) Mark(i int) (Verdict, bool) {
	if i != 0 || g.marked {
		return Skipped, false
	}
	g.marked = true
	return Correct, true
}

// A REFUSED CLICK MOVES NOTHING, and a form holding one word is where that
// matters: it is spent by definition, so routing a refusal through advance would
// step past the question on a click that hit nothing.
func TestARefusedClickDoesNotAdvanceAGridThatIsNotABatch(t *testing.T) {
	s := NewSession([]Question{&fakeGrid{}, NewRecall("mesa", "a flat-topped hill")})

	// The cell this form does not have.
	s, outs := Apply(s, Input{Kind: InputMark, Cell: 3})
	if len(outs) != 1 || outs[0].Kind != OutcomeNone {
		t.Fatalf("a click on a cell that is not there produced %+v", outs)
	}
	if s.Index != 0 {
		t.Fatal("a refused click advanced past the question")
	}
	// The real one lands and DOES advance, because this form holds one word.
	s, outs = Apply(s, Input{Kind: InputMark, Cell: 0})
	if len(outs) != 1 || outs[0].Kind != OutcomeRecord || outs[0].Word != "keel" {
		t.Fatalf("the mark produced %+v", outs)
	}
	if s.Index != 1 {
		t.Errorf("a spent one-word grid did not advance: index %d", s.Index)
	}
	// And now the SECOND click on a spent cell must not step off the queue.
	s, outs = Apply(s, Input{Kind: InputMark, Cell: 0})
	if s.Index != 1 || s.Done {
		t.Errorf("a click on the next question's non-existent grid moved the session: %+v %+v", s, outs)
	}
}

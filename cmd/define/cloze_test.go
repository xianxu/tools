package main

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/xianxu/tools/internal/llm"

	"github.com/xianxu/tools/cmd/define/play"
	"github.com/xianxu/tools/cmd/define/store"
)

// THE LEAK TABLE, which is this function's whole specification.
//
// Done-when: "the blanked sentence never leaks the answer (stem, plural,
// hyphenation)". Every row is a way it could — and the failure is SILENT: a
// question that gives away its answer still renders perfectly and still grades.
func TestBlankStem(t *testing.T) {
	for _, tc := range []struct{ name, stem, answer, want string }{
		{
			"the ordinary case",
			"The aide was sycophantic to a fault.", "sycophantic",
			"The aide was ___ to a fault.",
		},
		{
			"CAPITALISED, because a sentence-initial answer is",
			"Sycophantic aides surrounded him.", "sycophantic",
			"___ aides surrounded him.",
		},
		{
			"INFLECTED — the tail goes too, or `___s` gives it away",
			"Shipwrights laid the keels of three frigates.", "keel",
			"Shipwrights laid the ___ of three frigates.",
		},
		{
			"EVERY occurrence — a stem using it twice hands it over",
			"The keel cracked; the keel was replaced.", "keel",
			"The ___ cracked; the ___ was replaced.",
		},
		{
			"a POSSESSIVE goes with the run",
			"The keel's timbers rotted.", "keel",
			"The ___ timbers rotted.",
		},
		{
			"a SUBSTRING is not the word",
			"Sunset over the set of the play.", "set",
			"Sunset over the ___ of the play.",
		},
		{
			"a compound goes whole — `___-dog` would narrow it to one word",
			"They sold a hot-dog at the stand.", "hot",
			"They sold a ___ at the stand.",
		},
		{
			"absent: unchanged, and usableItem is what refuses such an item",
			"A sentence about nothing.", "quokka",
			"A sentence about nothing.",
		},
		{"an empty answer changes nothing", "Anything at all.", "", "Anything at all."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := blankStem(tc.stem, tc.answer); got != tc.want {
				t.Errorf("blankStem(%q, %q)\n = %q\nwant %q", tc.stem, tc.answer, got, tc.want)
			}
		})
	}
}

// The PROPERTY behind every row, asserted separately so a stem nobody thought of
// still cannot leak.
func TestBlankStemNeverLeavesTheAnswer(t *testing.T) {
	for _, tc := range []struct{ stem, answer string }{
		{"The aide was sycophantic to a fault.", "sycophantic"},
		{"Sycophantic and sycophantic again, sycophantically.", "sycophantic"},
		{"Shipwrights laid the keels; the keel's timbers rotted.", "keel"},
		{"İstanbul shipwrights laid the keel.", "keel"},
		{"The Ⱥ institute laid the keel.", "keel"},
	} {
		got := blankStem(tc.stem, tc.answer)
		if i, _ := wordIndexIn(got, tc.answer); i >= 0 {
			t.Errorf("blankStem(%q, %q) = %q — the answer survives at %d",
				tc.stem, tc.answer, got, i)
		}
		if !utf8.ValidString(got) {
			t.Errorf("blankStem(%q, %q) produced invalid UTF-8: %q", tc.stem, tc.answer, got)
		}
	}
}

// usableItem enumerates every way a stored item fails to be a question, because
// sanitiseItem neutralises on read without re-validating and the README invites
// hand-editing.
func TestUsableItem(t *testing.T) {
	ok := store.Item{
		Word: "sycophantic", Form: store.FormCloze,
		Stem: "The aide was sycophantic.", Answer: "sycophantic",
		Distractors: []string{"ephemeral", "keel"},
	}
	if !usableItem(ok) {
		t.Fatal("a well-formed item was refused")
	}
	for _, tc := range []struct {
		name string
		mut  func(store.Item) store.Item
	}{
		{"no answer", func(i store.Item) store.Item { i.Answer = ""; return i }},
		{"the stem does not contain the answer", func(i store.Item) store.Item {
			i.Stem = "A sentence about nothing."
			return i
		}},
		{"the stem arrives pre-blanked", func(i store.Item) store.Item {
			i.Stem = "The aide was ___."
			return i
		}},
		{"no distractors", func(i store.Item) store.Item { i.Distractors = nil; return i }},
		// THE NASTIEST: it renders as two correct options with one marked wrong,
		// so grading marks the learner wrong for being right. Exactly what #10's
		// veto prevents, arriving by the one path the veto never sees.
		{"a distractor EQUAL to the answer", func(i store.Item) store.Item {
			i.Distractors = []string{"ephemeral", "sycophantic"}
			return i
		}},
		{"equal but for case", func(i store.Item) store.Item {
			i.Distractors = []string{"Sycophantic"}
			return i
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if usableItem(tc.mut(ok)) {
				t.Error("an unusable item was accepted; it would render as a broken question")
			}
		})
	}
}

func clozeItem() store.Item {
	return store.Item{
		Word: "sycophantic", Form: store.FormCloze,
		Stem:        "The Times dismissed the interviews as sycophantic.",
		Answer:      "sycophantic",
		Distractors: []string{"ephemeral", "keel", "mesa"},
	}
}

// THE ANSWER MOVES. Without a shuffle it sits at position 1 in every question
// and a learner answers the whole sitting by pressing 1 — a question that still
// looks right, which is why #7 pinned the same property.
func TestClozeForMovesTheAnswerAround(t *testing.T) {
	seen := map[int]bool{}
	for seed := uint64(1); seed < 60; seed++ {
		q := clozeFor("sycophantic", []store.Item{clozeItem()}, "", seed)
		if q == nil {
			t.Fatal("a usable item produced no question")
		}
		for i, o := range q.(*play.Cloze).Options() {
			if o.Correct {
				seen[i] = true
			}
		}
	}
	if len(seen) < 2 {
		t.Errorf("the answer only ever appeared at position(s) %v; the option order is not shuffled", seen)
	}
}

// Deterministic under a fixed seed, which is the Done-when row.
func TestClozeForIsDeterministic(t *testing.T) {
	words := func(q play.Question) string {
		var b []string
		for _, o := range q.(*play.Cloze).Options() {
			b = append(b, o.Word)
		}
		return strings.Join(b, ",")
	}
	a := clozeFor("sycophantic", []store.Item{clozeItem()}, "", 42)
	b := clozeFor("sycophantic", []store.Item{clozeItem()}, "", 42)
	if words(a) != words(b) {
		t.Errorf("the same seed gave %s then %s", words(a), words(b))
	}
}

// The NEWEST item of the right form, and no other. Items() returns newest-first,
// so "the first of the right form" is already the newest.
func TestClozeForTakesTheFirstUsableClozeItem(t *testing.T) {
	sentence := clozeItem()
	sentence.Form = store.FormSentence
	sentence.Stem = "WRONG FORM"

	broken := clozeItem()
	broken.Distractors = nil // unusable

	newest := clozeItem()
	newest.Stem = "The reviewer found the biography sycophantic."

	q := clozeFor("sycophantic", []store.Item{sentence, broken, newest}, "", 7)
	if q == nil {
		t.Fatal("no question built although a usable cloze item was present")
	}
	if !strings.Contains(q.Reveal(), "biography") {
		t.Errorf("took the wrong item; reveal was %q", q.Reveal())
	}
}

// No usable item is not an error — it is the ordinary state of a word #10 has
// not authored yet, and the caller falls back to form 2.3.
func TestClozeForReportsWhenItCannot(t *testing.T) {
	for _, name := range []string{"no items", "only a sentence item", "only a broken item"} {
		var items []store.Item
		switch name {
		case "only a sentence item":
			it := clozeItem()
			it.Form = store.FormSentence
			items = []store.Item{it}
		case "only a broken item":
			it := clozeItem()
			it.Distractors = nil
			items = []store.Item{it}
		}
		if q := clozeFor("sycophantic", items, "", 1); q != nil {
			t.Errorf("%s: built a question anyway", name)
		}
	}
}

// The question the learner actually sees: the stem blanked, the answer among the
// options, and the reveal restoring it.
func TestClozeForBuildsAnAnswerableQuestion(t *testing.T) {
	q := clozeFor("sycophantic", []store.Item{clozeItem()}, "flattering", 3)
	if q == nil {
		t.Fatal("no question")
	}
	stem, _, _ := strings.Cut(q.Prompt(), "\n")
	if strings.Contains(strings.ToLower(stem), "sycophantic") {
		t.Errorf("the prompt's sentence leaks the answer: %q", stem)
	}
	if !strings.Contains(stem, Blank) {
		t.Errorf("the prompt's sentence has no blank: %q", stem)
	}
	if !strings.Contains(q.Reveal(), "sycophantic") {
		t.Errorf("the reveal does not restore the word: %q", q.Reveal())
	}
	if q.Form() != "cloze" {
		t.Errorf("Form() = %q", q.Form())
	}
}

// An answer with no word characters is not a word, and an item carrying one is
// not a question — found by the fuzz, which blanked "_" into "___" and produced
// a question whose blank IS its answer.
func TestUsableItemRefusesAnAnswerThatIsNotAWord(t *testing.T) {
	for _, answer := range []string{"_", "...", "---", "  ", "!?"} {
		it := store.Item{
			Word: answer, Form: store.FormCloze,
			Stem: "A sentence with " + answer + " in it.", Answer: answer,
			Distractors: []string{"ephemeral"},
		}
		if usableItem(it) {
			t.Errorf("answer %q was accepted; it contains nothing a word is made of", answer)
		}
	}
	// `-` and `'` ARE inside words, so a hyphenated answer is still a word.
	it := store.Item{
		Word: "hot-dog", Form: store.FormCloze,
		Stem: "They sold a hot-dog.", Answer: "hot-dog",
		Distractors: []string{"ephemeral"},
	}
	if !usableItem(it) {
		t.Error("a hyphenated word was refused")
	}
}

// THE SELECTION RULE, as a rule rather than a branch: "an authored item beats a
// definition match", and everything else unchanged.
func TestTheFormSelectionRule(t *testing.T) {
	for _, tc := range []struct {
		name     string
		box      int
		items    []store.Item
		wantForm string
	}{
		{"a young word WITH an item takes cloze", 0, []store.Item{clozeItem()}, "cloze"},
		// The row that catches a rule written as "always cloze": #7's behaviour
		// must be untouched for a word #10 has not authored.
		{"a young word WITHOUT one still takes 2.3", 0, nil, "meaning"},
		// #42's rule, unchanged: a mature word is triaged, not tested, even
		// holding perfectly good material.
		{"a MATURE word still goes to the board", boardBox, []store.Item{clozeItem()}, "board"},
		// The store neutralises on read but does not re-validate, and the README
		// documents items/ as inspectable.
		{"an UNUSABLE item falls back to 2.3", 0, []store.Item{brokenItem()}, "meaning"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A REAL DECK, not one word: form 2.3 draws its distractors from the
			// sitting's pool, so a single-word rig cannot build one and every
			// word falls to the board — which would make the 2.3 rows below pass
			// for #42's reason rather than for the rule under test.
			d, opt, st := playRig(t, "sycophantic", "ephemeral", "keel", "mesa", "parrot", "concrete")
			for _, it := range tc.items {
				if err := st.SetItems("sycophantic", []store.Item{it}); err != nil {
					t.Fatal(err)
				}
			}
			for range tc.box {
				if err := st.AppendEvent(store.ReviewEvent{
					Word: "sycophantic", Kind: store.EventReviewed, Found: true,
					Correct: true, Unaided: true, At: aDay,
				}); err != nil {
					t.Fatal(err)
				}
			}

			var out, errb bytes.Buffer
			qs, _, code := todaysQuestions(d, opt, &out, &errb)
			if code != 0 {
				t.Fatalf("todaysQuestions = %d, stderr %q", code, errb.String())
			}
			var got string
			for _, q := range qs {
				if q.Word() == "sycophantic" {
					got = q.Form()
				}
			}
			if got == "" {
				// The board holds many words, so a triaged word is not its own
				// question — which IS the board answer for the mature row.
				got = "board"
			}
			if got != tc.wantForm {
				t.Errorf("form = %q, want %q (stdout %q)", got, tc.wantForm, out.String())
			}
		})
	}
}

func brokenItem() store.Item {
	it := clozeItem()
	// A distractor equal to the answer: two correct options, one marked wrong.
	it.Distractors = []string{"sycophantic"}
	return it
}

// A cloze sitting makes NO model call. The seam is made to PANIC rather than
// left nil, because nil passes on a loop that reaches for the model behind a
// `!= nil` guard — #10's Done-when 1 test is the precedent and the reason.
func TestAClozeSittingNeverReachesForTheModel(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic", "ephemeral", "keel")
	if err := st.SetItems("sycophantic", []store.Item{clozeItem()}); err != nil {
		t.Fatal(err)
	}
	d.getenv = func(string) string { panic("a cloze sitting resolved the model configuration") }
	d.newLLM = func(llm.Config) llm.Client { panic("a cloze sitting constructed a model client") }

	var out, errb bytes.Buffer
	qs, held, code := todaysQuestions(d, opt, &out, &errb)
	if code != 0 || len(qs) == 0 {
		t.Fatalf("todaysQuestions = %d with %d questions", code, len(qs))
	}
	var cloze play.Question
	for _, q := range qs {
		if q.Form() == "cloze" {
			cloze = q
		}
	}
	if cloze == nil {
		t.Fatal("no cloze question was built, so this pin cannot fail")
	}
	// And through a whole sitting, not only the build.
	keys := "\r" + gradeKey(t, cloze, play.Correct)
	playSession(t.Context(), d, opt, play.NewSession([]play.Question{cloze}), held, keysFor(keys), playbackConsole(&out, &errb))
}

// THE FLAG, end to end: keystroke → capability → outcome → capture verb → log.
//
// Driven through a real sitting rather than by calling the verb, because the
// gate's finding was about the PATH: the obvious wiring writes an EventReviewed
// and demotes the word.
func TestFlaggingAQuestionRecordsItWithoutScoringIt(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic", "ephemeral", "keel")
	if err := st.SetItems("sycophantic", []store.Item{clozeItem()}); err != nil {
		t.Fatal(err)
	}
	cap := &countingCapturer{}
	d.capture = cap

	var out, errb bytes.Buffer
	qs, held, code := todaysQuestions(d, opt, &out, &errb)
	if code != 0 {
		t.Fatalf("todaysQuestions = %d", code)
	}
	var cloze play.Question
	for _, q := range qs {
		if q.Form() == "cloze" {
			cloze = q
		}
	}
	if cloze == nil {
		t.Fatal("no cloze question, so this pin cannot fail")
	}

	playSession(t.Context(), d, opt, play.NewSession([]play.Question{cloze}), held,
		keysFor(string(play.FlagKey)), playbackConsole(&out, &errb))

	if len(cap.flags) != 1 {
		t.Fatalf("recorded %d flags, want 1", len(cap.flags))
	}
	// THE EVIDENCE TRAVELLED: the option set is what makes a flag diagnosable.
	if len(cap.flags[0]) != 4 {
		t.Errorf("the flag carried %v, want the whole option set", cap.flags[0])
	}
	// AND IT SCORED NOTHING. This is the finding the gate caught: a flag written
	// as a review would have marked the word wrong, on an append-only log.
	if cap.reviews != 0 {
		t.Errorf("a flag produced %d review(s); the ladder must not move on a broken question", cap.reviews)
	}
	if !strings.Contains(out.String(), "flagged") {
		t.Errorf("the flag was silent: %q", out.String())
	}
}

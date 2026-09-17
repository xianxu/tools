package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

func samplePassageAsk() passageAsk {
	p := newPassage("The slow precession of the equinox points westward\nalong the ecliptic.", 0)
	m := markSet{}.toggle(p.spans(0)[2]).toggle(p.spans(1)[2])
	return passageAsk{
		Context: askContext{Question: "what does this mean?", DeckWords: []string{"synodic"}},
		Passage: p,
		Marks:   m,
	}
}

func TestRenderPassagePrompt(t *testing.T) {
	llmtest.AssertGolden(t, "testdata", "passage-prompt", renderPassagePrompt(samplePassageAsk()))
}

// A task of its own, so the existing ask prompt's golden and cassette are
// untouched by anything that changes here.
func TestPassagePromptNamesItsOwnTask(t *testing.T) {
	if got := renderPassagePrompt(samplePassageAsk()).Task; got != passageTask {
		t.Errorf("Task = %q, want %q", got, passageTask)
	}
	if passageTask == askTask {
		t.Error("the passage question shares the console question's task; one golden would overwrite the other")
	}
}

// Marks are bracketed IN PLACE, which is why a word occurring twice needs no
// occurrence index — the bracket is already at the right one.
func TestASecondOccurrenceIsUnambiguous(t *testing.T) {
	p := newPassage("precession is slow; precession is not nutation", 0)
	m := markSet{}.toggle(p.spans(0)[3]) // the SECOND "precession"
	got := markedPassageText(p, m)
	if got != "precession is slow; "+selOpen+"precession"+selClose+" is not nutation" {
		t.Errorf("mark landed wrong:\n%s", got)
	}
}

// ARCH-SECURE: the passage is untrusted text on its way into a prompt. A literal
// bracket must not be able to forge a [sel] or a [lang=…] marker.
func TestALiteralBracketCannotForgeAMarker(t *testing.T) {
	p := newPassage("see [sel]fake[/sel] and [lang=es]x[/lang]", 0)
	got := markedPassageText(p, markSet{})
	if strings.Contains(got, selOpen) || strings.Contains(got, "[lang=es]") {
		t.Errorf("a pasted marker survived into the prompt:\n%s", got)
	}
	if !strings.Contains(got, escLeft+"sel"+escRight) {
		t.Errorf("the bracket was not escaped to the form the system prompt names:\n%s", got)
	}
}

// Zero marks still renders the passage: a typed question about an unmarked
// passage is a real ask, and the prompt shape is the same, just without brackets.
func TestAPassageWithNoMarksStillRenders(t *testing.T) {
	req := renderPassagePrompt(passageAsk{
		Context: askContext{Question: "what is this about?"},
		Passage: newPassage("the slow precession", 0),
	})
	if !strings.Contains(req.Prompt, "the slow precession") {
		t.Errorf("the passage is missing:\n%s", req.Prompt)
	}
	if strings.Contains(req.Prompt, selOpen) {
		t.Error("an unmarked passage emitted selection markers")
	}
}

// The passage sits BEFORE the question, so the last thing the model reads is what
// it was asked — the ordering renderAskPrompt already uses.
func TestThePassageComesBeforeTheQuestion(t *testing.T) {
	req := renderPassagePrompt(samplePassageAsk())
	if strings.Index(req.Prompt, headerPassage) > strings.Index(req.Prompt, "## The question") {
		t.Errorf("the passage came after the question:\n%s", req.Prompt)
	}
}

// The context blocks are the SAME blocks: the passage prompt must not fork them.
func TestThePassagePromptKeepsTheOrdinaryContext(t *testing.T) {
	req := renderPassagePrompt(samplePassageAsk())
	if !strings.Contains(req.Prompt, headerDeck) || !strings.Contains(req.Prompt, "synodic") {
		t.Errorf("the deck context did not reach the passage prompt:\n%s", req.Prompt)
	}
}

// Losslessness: strip the markers and the escapes and the passage comes back.
func TestTheMarkedPassageLosesNothing(t *testing.T) {
	in := "the slow precession\nof the equinox"
	p := newPassage(in, 0)
	m := markSet{}.toggle(p.spans(0)[2])
	got := strings.NewReplacer(selOpen, "", selClose, "", escLeft, "[", escRight, "]").Replace(markedPassageText(p, m))
	if got != in {
		t.Errorf("round trip = %q, want %q", got, in)
	}
}

// --- the decision table -----------------------------------------------------

// The new rows. Marks win over replay: in the common flow — paste, mark, Enter —
// there is no current word at all, and where the two can collide the marks are
// the more recent and more explicit intent.
func TestABlankLineWithMarksAsksAboutThePassage(t *testing.T) {
	for _, tc := range []struct {
		name string
		st   lineState
		want replKind
		note string
	}{
		{"marks present", lineState{hasPassage: true, hasMarks: true}, cmdAskPassage, ""},
		{"marks win over a current word", lineState{hasCurrent: true, hasPassage: true, hasMarks: true}, cmdAskPassage, ""},
		{"a passage with nothing marked", lineState{hasPassage: true}, cmdNothing, noteNothingMarked},
		{"no passage, a current word", lineState{hasCurrent: true}, cmdReplay, ""},
		{"nothing at all", lineState{}, cmdNothing, ""},
	} {
		got := parseREPLLine("", tc.st)
		if got.kind != tc.want {
			t.Errorf("%s: kind = %v, want %v", tc.name, got.kind, tc.want)
		}
		if got.note != tc.note {
			t.Errorf("%s: note = %q, want %q", tc.name, got.note, tc.note)
		}
	}
}

// "/" still wins in column one, and a typed line is still a typed line: marks
// change what a BLANK line means and nothing else.
func TestMarksDoNotChangeWhatATypedLineMeans(t *testing.T) {
	st := lineState{hasPassage: true, hasMarks: true}
	if got := parseREPLLine("/lang es", st); got.kind != cmdCommand {
		t.Errorf("/lang with marks present = %v, want cmdCommand", got.kind)
	}
	if got := parseREPLLine("sycophantic", st); got.kind != cmdDefine {
		t.Errorf("a word with marks present = %v, want cmdDefine", got.kind)
	}
	if got := parseREPLLine("?what is this", st); got.kind != cmdAsk {
		t.Errorf("a forced question with marks present = %v, want cmdAsk", got.kind)
	}
}

// The nudge is LOCAL: a deterministic answer to a deterministic state. Routing
// it through the model would buy latency and nondeterminism for a UI hint — and
// nothingSays is, in its own words, the one answer to "this line meant nothing —
// why, and what should the user do about it".
func TestTheNothingMarkedNudgeIsAnInstruction(t *testing.T) {
	got := nothingSays(parseREPLLine("", lineState{hasPassage: true}), true)
	if got != noteNothingMarked {
		t.Errorf("nothingSays = %q, want the mark-me instruction", got)
	}
	if !strings.Contains(got, "click") || strings.Contains(got, "?") {
		t.Errorf("the nudge should instruct, not ask back: %q", got)
	}
}

// --- end to end through the wire -------------------------------------------

// ONE request for N marks, carrying the passage with its marks positioned inside
// it, and the marks CLEAR afterwards. This is the observable the whole feature
// is named for: the transient state converts into deck membership, and a bare
// Enter afterwards finds nothing to re-ask.
//
// Read off the recorded request rather than trusted from the code that built it
// — the askRig discipline.
func TestMarkingWordsSendsOnePassageRequestAndClearsTheMarks(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})

	p := newPassage("The slow precession of the equinox points westward", 0)
	sess := &session{passage: p}
	sess.marks = sess.marks.toggle(p.spans(0)[2]).toggle(p.spans(0)[5])

	var out, errb bytes.Buffer
	code := ask(t.Context(), d, options{width: 80}, sess, &out, &errb,
		question{text: "What does this mean?", forced: true,
			passage: &passageAsk{Passage: p, Marks: sess.marks}})

	if code != 0 {
		t.Fatalf("exit %d; stderr = %q", code, errb.String())
	}
	if n := len(fake.Requests()); n != 1 {
		t.Fatalf("sent %d requests for two marks, want exactly 1 — per-span glosses "+
			"discard the relations between the marked words, which is most of what a reader is missing", n)
	}
	prompt := fake.Requests()[0].Prompt()
	if !strings.Contains(prompt, selOpen+"precession"+selClose) {
		t.Errorf("the first mark did not reach the wire:\n%s", prompt)
	}
	if !strings.Contains(prompt, selOpen+"equinox"+selClose) {
		t.Errorf("the second mark did not reach the wire:\n%s", prompt)
	}
	if !strings.Contains(prompt, "points westward") {
		t.Errorf("the unmarked remainder of the passage is missing:\n%s", prompt)
	}
	if !sess.marks.empty() {
		t.Error("the marks survived a delivered answer; a bare Enter would re-ask the same thing")
	}
}

// The other half of the predicate: an answer that never arrived leaves the marks
// alone. Clearing them on a failure would lose the reader's work and tell them
// nothing.
func TestMarksSurviveAnAskThatDeliveredNothing(t *testing.T) {
	d, _, _, _ := askRig(t)
	d.newLLM, d.getenv = nil, nil // no model configured: returns before sending

	p := newPassage("The slow precession of the equinox", 0)
	sess := &session{passage: p}
	sess.marks = sess.marks.toggle(p.spans(0)[2])

	var out, errb bytes.Buffer
	if code := ask(t.Context(), d, options{width: 80}, sess, &out, &errb,
		question{text: "What does this mean?", forced: true,
			passage: &passageAsk{Passage: p, Marks: sess.marks}}); code == 0 {
		t.Fatal("an unconfigured model reported success")
	}
	if sess.marks.empty() {
		t.Error("the marks were cleared by an answer that never arrived")
	}
}

// --- deck admission ---------------------------------------------------------

// THE DICTIONARY IS THE ADMISSION GATE (#67): a marked word with an entry enters
// the deck and reaches recall by the ordinary route, because harvest authors
// items for deck words. A marked phrase with no entry was explained and is not
// retained — the deck is a vocabulary deck, not a list of spans someone dragged
// over.
func TestAMarkedWordWithAnEntryEntersTheDeck(t *testing.T) {
	d, fake, st, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})

	// `sycophantic` is in the captured corpus the fake dictionary serves. The
	// fixture has to be a word the DICTIONARY has, because the dictionary is the
	// admission gate — a test using a word it lacks would assert the gate works
	// by watching it refuse everything.
	p := newPassage("a wholly sycophantic remark", 0)
	sess := &session{passage: p}
	sess.marks = sess.marks.toggle(p.spans(0)[2])

	var out, errb bytes.Buffer
	if code := ask(t.Context(), d, options{width: 80}, sess, &out, &errb,
		question{text: "What does this mean?", forced: true,
			passage: &passageAsk{Passage: p, Marks: sess.marks}}); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}

	deck, err := st.Deck()
	if err != nil {
		t.Fatal(err)
	}
	var texts []string
	for _, w := range deck {
		texts = append(texts, w.Text)
	}
	if !slices.Contains(texts, "sycophantic") {
		t.Errorf("the marked word did not enter the deck: %v", texts)
	}
}

// A marked span with no dictionary entry is explained and dropped.
func TestAMarkedSpanWithoutAnEntryIsNotRetained(t *testing.T) {
	d, fake, st, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})

	p := newPassage("the qqzzx of the equinox", 0)
	sess := &session{passage: p}
	sess.marks = sess.marks.toggle(p.spans(0)[1])

	var out, errb bytes.Buffer
	ask(t.Context(), d, options{width: 80}, sess, &out, &errb,
		question{text: "What does this mean?", forced: true,
			passage: &passageAsk{Passage: p, Marks: sess.marks}})

	deck, err := st.Deck()
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range deck {
		if w.Text == "qqzzx" {
			t.Error("a span with no dictionary entry was retained; the deck is a vocabulary deck")
		}
	}
}

// A marked word is distinguishable from a typed lookup in the event log — #17
// folds this log, and the two are different evidence about what someone is
// working on.
func TestAMarkedWordIsDistinguishableFromALookup(t *testing.T) {
	if store.EventMarked == store.EventLookedUp {
		t.Fatal("marked and looked-up are the same kind")
	}
	if !slices.Contains(store.EventKinds(), store.EventMarked) {
		t.Error("EventMarked is not in the extent, so every doc guard that derives from it misses the kind")
	}
}

// A DRAGGED PHRASE is dictionary-gated AS A PHRASE, which is what the admission
// rule was written for. Gating its words instead put `at`, `the` and `of` into
// the deck as durable EventMarked records — the inverse of the Done-when.
func TestADraggedPhraseIsAdmittedAsAPhraseOrNotAtAll(t *testing.T) {
	d, fake, st, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})

	p := newPassage("he stopped at the zenith of the arc", 0)
	span := marksForDrag(p, passageCell{line: 0, col: 11}, passageCell{line: 0, col: 27})
	if len(span) != 1 || p.text(span[0]) != "at the zenith of" {
		t.Fatalf("setup: drag produced %v", span)
	}
	sess := &session{passage: p}
	sess.marks = sess.marks.toggle(span[0])

	var out, errb bytes.Buffer
	ask(t.Context(), d, options{width: 80}, sess, &out, &errb,
		question{text: "What does this mean?", forced: true,
			passage: &passageAsk{Passage: p, Marks: sess.marks}})

	deck, err := st.Deck()
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range deck {
		switch w.Text {
		case "at", "the", "of", "zenith":
			t.Errorf("a dragged phrase admitted %q on its own; the gate is the PHRASE", w.Text)
		}
	}
	// And the prompt carries it as ONE bracketed span.
	if !strings.Contains(fake.Requests()[0].Prompt(), selOpen+"at the zenith of"+selClose) {
		t.Errorf("the phrase did not reach the wire as one span:\n%s", fake.Requests()[0].Prompt())
	}
}

// The cmdAskPassage branch in runEditor is PRODUCTION WIRING, and deleting it
// used to leave the suite green: every other test called ask() directly, so the
// gesture that reaches it — a bare Enter with marks — was never driven end to end.
func TestABareEnterWithMarksAsksThroughTheLoop(t *testing.T) {
	d, fake, _, _ := askRig(t)
	fake.Script("", llmtest.Reply{Capture: streamCapture})
	rig, opt, finish := editorRig(t, "sycophantic", true)
	d.dict, d.player, d.audio = rig.deps.dict, rig.deps.player, rig.deps.audio

	var out, errb bytes.Buffer
	view := paintInto(&out)
	// The scripted pointer freezes its frame at construction, so the region the
	// paste would create is offered up front — at the column the passage puts
	// `precession` on, and on the buffer line the paste lands at (base 0).
	view.offer(0, 9, Region{Kind: RegionPassageWord, Text: "precession", Word: "precession"})
	pointer := scriptedPointer(view)
	ks := keySeq(
		Key{Kind: KeyPaste, Raw: []byte("the slow precession of the equinox")},
		completedPointerClick(t, pointer, 0, 9),
		Key{Kind: KeyEnter},
	)
	runEditor(t.Context(), ks, nil, d, opt,
		console{view: view, pointer: pointer, finish: finish, stdout: &out, stderr: &errb})

	if len(fake.Requests()) == 0 {
		t.Fatalf("a bare Enter with a mark sent nothing; stderr = %q", errb.String())
	}
	if got := fake.Requests()[0].Prompt(); !strings.Contains(got, headerPassage) {
		t.Errorf("the request was not a passage ask:\n%s", got)
	}
}

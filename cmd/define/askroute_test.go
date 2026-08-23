package main

import (
	"bytes"
	"strings"
	"testing"
)

const aQuestion = "what's the difference to obsequious?"

// "Every entry mode reaches it" is the invariant BR-13 cost us once already, so
// it gets a test per mode rather than one test and an assumption.
//
// And per mode is not enough: a question arrives by TWO routes — forced ("?…",
// decided by the parser without a dictionary call) and unforced (a miss that
// reads as one) — so the enumeration this file has to cover is
// {replLines, runEditor} x {forced, unforced}, four cells. Three were empty on
// the forced side at the M1 boundary review (I-2): the whole `case cmdAsk:` could
// be deleted from BOTH loops with the suite green, because every loop-level test
// took the unforced route and routeFor answers "question" for cmdAsk without
// entering a loop at all. That is lessons.md define #15 — a wiring only a loop
// shell supplies must be pinned by a test that drives that loop shell — applied
// to the branch this milestone is named after.
func TestLineLoopRoutesAQuestion(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	replLines(t.Context(), rig.deps, options{times: 3, locale: "us"},
		strings.NewReader(aQuestion+"\n"), &out, &errb, true, false)

	assertAskedAndUnanswered(t, errb.String(), out.String())
}

func TestEditorLoopRoutesAQuestion(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys(aQuestion+"\r"), rig.deps, opt, cooked, finish, &out, &errb)

	assertAskedAndUnanswered(t, errb.String(), out.String())
}

// The forced half of the enumeration. Both pass against today's code — which is
// the point: they exist to go RED when the branch is touched.
func TestLineLoopRoutesAForcedQuestion(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	// "why" IS a headword, so only the hatch can make this a question — which is
	// what makes it a test of the forced branch rather than of the classifier.
	replLines(t.Context(), rig.deps, options{times: 3, locale: "us"},
		strings.NewReader("?why\n"), &out, &errb, true, false)

	assertAskedAndUnanswered(t, errb.String(), out.String())
	if strings.Contains(out.String(), "adverb") {
		t.Error("the forced question was defined instead of asked")
	}
}

func TestEditorLoopRoutesAForcedQuestion(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("?why\r"), rig.deps, opt, cooked, finish, &out, &errb)

	assertAskedAndUnanswered(t, errb.String(), out.String())
	if rig.player.count() != 0 {
		t.Errorf("played %d times — a forced question fell through to replay", rig.player.count())
	}
}

func TestOneShotRoutesAQuestion(t *testing.T) {
	var out, errb bytes.Buffer
	// Quoted: one argument, unforced. The dictionary misses and the classifier
	// answers, exactly as it does at the prompt.
	if code := run(t.Context(), []string{aQuestion}, testDeps(t),
		strings.NewReader(""), &out, &errb); code == 0 {
		t.Errorf("exit = 0, want non-zero for an unanswerable question")
	}
	assertAskedAndUnanswered(t, errb.String(), out.String())
}

func TestOneShotForcedQuestionIsNotAUsageError(t *testing.T) {
	var out, errb bytes.Buffer
	// Unquoted and multi-word: a question is multi-word by nature, exactly as a
	// command is, so the argument-count guard must exempt it the same way.
	run(t.Context(), []string{"?what", "is", "the", "difference"},
		testDeps(t), strings.NewReader(""), &out, &errb)

	if strings.Contains(errb.String(), "usage") || strings.Contains(errb.String(), "Usage") {
		t.Errorf("a forced question was rejected as a usage error: %q", errb.String())
	}
	assertAskedAndUnanswered(t, errb.String(), out.String())
}

func assertAskedAndUnanswered(t *testing.T, stderr, stdout string) {
	t.Helper()
	if !strings.Contains(stderr, "no model configured") {
		t.Errorf("stderr = %q, want the no-model message", stderr)
	}
	if strings.Contains(stderr, "not found") {
		t.Errorf("the question reached the dictionary: %q", stderr)
	}
	if strings.Contains(stdout, "|") {
		t.Errorf("a definition was rendered for a question: %q", stdout)
	}
}

// The ask outcome's contract, one test per row. Each fails if its guard is
// deleted — which is the only reason to write them separately.

func TestAQuestionDoesNotBecomeTheCurrentWord(t *testing.T) {
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer
	// Look up a word, ask a question, then press Enter: the REPLAY is the word.
	runEditor(t.Context(), scriptKeys("sycophantic\r"+aQuestion+"\r\r"), rig.deps, opt, cooked, finish, &out, &errb)

	// 3 plays for the lookup, 3 more for the replay. If the question had become
	// the current word the replay would have found no audio for it.
	if got := rig.player.count(); got != 6 {
		t.Errorf("played %d times, want 6 — the question became the current word", got)
	}
}

func TestAQuestionIsNotCaptured(t *testing.T) {
	cap := &countingCapturer{}
	rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
	rig.deps.capture = cap
	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("sycophantic\r"+aQuestion+"\r"), rig.deps, opt, cooked, finish, &out, &errb)

	if len(cap.calls) != 1 || cap.calls[0] != "sycophantic" {
		t.Errorf("captured %v, want just the lookup — a question is not a lookup", cap.calls)
	}
}

// Asserted against the injected History, NOT against stdout.
//
// The first version of this test checked stdout for the question and could not
// fail: the raw editor re-renders the whole line on every keystroke, so the
// question is in stdout from typing alone — before Enter, before Up, before
// history is consulted. Removing hist.Add outright left it green (I-1). The
// general shape is worth remembering: in a loop that echoes, an assertion on
// stdout is satisfied by the echo, so it has to be made against the thing the
// behaviour actually writes to.
func TestAQuestionIsRecalledByUpArrow(t *testing.T) {
	for _, tc := range []struct{ name, keys, want string }{
		{"forced", "?why\r", "?why"},
		{"unforced", aQuestion + "\r", aQuestion},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
			hist := &memHistory{}
			rig.deps.history = hist
			var out, errb bytes.Buffer
			runEditor(t.Context(), scriptKeys(tc.keys), rig.deps, opt, cooked, finish, &out, &errb)

			if len(hist.lines) != 1 || hist.lines[0] != tc.want {
				t.Fatalf("history = %q, want [%q] — the question was not recorded for recall", hist.lines, tc.want)
			}
			// And that what was recorded is what Up actually offers back.
			if got := hist.Prefix(""); len(got) == 0 || got[0] != tc.want {
				t.Errorf("Prefix(\"\") = %q, want %q first", got, tc.want)
			}
		})
	}
}

// BR-4: the one-shot's dispatch must be exhaustive over what parseREPLLine can
// return, not "handle the kinds I added and fall through". #16 gave the parser a
// kind that branch had never seen — cmdNothing, from a bare "?" or "\" — and the
// fall-through handed lookupAndRender an EMPTY word, which then appended an event
// with no word that the log discards at read time as if it were torn.
func TestOneShotRejectsAHatchWithNothingAfterIt(t *testing.T) {
	for _, line := range []string{"?", `\`} {
		t.Run(line, func(t *testing.T) {
			cap := &countingCapturer{}
			d := testDeps(t)
			// Set at the seam withStore actually reads. Injecting through
			// d.newStore looked equivalent and was DEAD: withStore only fills
			// nils (main.go:96), and testDeps has already supplied a capturer,
			// so the double was discarded and every assertion over cap.calls
			// ranged over an empty slice (BR-10). The rule: a test that injects
			// a double must inject where production reads, or assert the
			// injection took effect. The control subtest below does the second
			// half, so a future re-break is a failure rather than a silence.
			d.capture = cap
			var out, errb bytes.Buffer
			code := run(t.Context(), []string{line}, d, strings.NewReader(""), &out, &errb)

			if code != 2 {
				t.Errorf("exit = %d, want 2 (a usage error)", code)
			}
			if strings.Contains(errb.String(), "no dictionary entry") {
				t.Errorf("an empty word reached the dictionary: %q", errb.String())
			}
			for _, w := range cap.calls {
				if w == "" {
					t.Error("an empty word was captured — the log would hold a record it discards as torn")
				}
			}
		})
	}
}

// The control half of the rule above: the same injection, in a case that MUST
// capture. If the double is ever discarded again, this goes red immediately
// instead of leaving the assertions above ranging over an empty slice.
func TestTheCapturerInjectionIsLive(t *testing.T) {
	cap := &countingCapturer{}
	d := testDeps(t)
	d.capture = cap
	var out, errb bytes.Buffer
	run(t.Context(), []string{"-no-audio", "sycophantic"}, d, strings.NewReader(""), &out, &errb)

	if len(cap.calls) == 0 {
		t.Fatal("the injected capturer saw nothing — every assertion over cap.calls is dead")
	}
}

// -raw is the scripting form; README documents it as recording nothing "because
// it is for scripts". It must not reach the model either — in M1 that costs a
// different message, in M2 a network call for a line a script piped in.
//
// "Never" is an absolute, so it names its enumeration and asserts every cell:
// {forced, unforced} x {one-shot, piped loop, raw editor}. The first version of
// this test named the class and pinned ONE cell — the unforced one-shot — while
// all three forced cells still asked (BR-9). A claim stated as an absolute must
// name what it quantifies over.
func TestRawNeverAsks(t *testing.T) {
	rawOpt := options{times: 3, locale: "us", raw: true}

	// The unforced cells must produce the miss the scripting contract promises;
	// the forced ones must produce the refusal. Both are POSITIVE observables —
	// see assertDidNotAsk.
	const wantMiss, wantRefusal = "no dictionary entry", "-raw does not ask"

	t.Run("one-shot/unforced", func(t *testing.T) {
		var out, errb bytes.Buffer
		code := run(t.Context(), []string{"-raw", aQuestion}, testDeps(t), strings.NewReader(""), &out, &errb)
		assertDidNotAsk(t, code, 1, errb.String(), wantMiss)
	})
	t.Run("one-shot/forced", func(t *testing.T) {
		var out, errb bytes.Buffer
		code := run(t.Context(), []string{"-raw", "?why"}, testDeps(t), strings.NewReader(""), &out, &errb)
		assertDidNotAsk(t, code, 2, errb.String(), wantRefusal)
	})
	t.Run("piped/unforced", func(t *testing.T) {
		rig := newAudioRig(t, "sycophantic", true)
		var out, errb bytes.Buffer
		code := replLines(t.Context(), rig.deps, rawOpt, strings.NewReader(aQuestion+"\n"), &out, &errb, true, false)
		assertDidNotAsk(t, code, 1, errb.String(), wantMiss)
	})
	t.Run("piped/forced", func(t *testing.T) {
		rig := newAudioRig(t, "sycophantic", true)
		var out, errb bytes.Buffer
		// BR-15: ask() computes 2 and the loop must not collapse it to 1.
		code := replLines(t.Context(), rig.deps, rawOpt, strings.NewReader("?why\n"), &out, &errb, true, false)
		assertDidNotAsk(t, code, 2, errb.String(), wantRefusal)
	})
	t.Run("editor/unforced", func(t *testing.T) {
		rig, _, cooked, finish := editorRig(t, "sycophantic", true)
		opt := rawOpt
		opt.tty = true
		var out, errb bytes.Buffer
		runEditor(t.Context(), scriptKeys(aQuestion+"\r"), rig.deps, opt, cooked, finish, &out, &errb)
		assertDidNotAsk(t, 0, 0, errb.String(), wantMiss)
	})
	t.Run("editor/forced", func(t *testing.T) {
		rig, _, cooked, finish := editorRig(t, "sycophantic", true)
		opt := rawOpt
		opt.tty = true
		var out, errb bytes.Buffer
		runEditor(t.Context(), scriptKeys("?why\r"), rig.deps, opt, cooked, finish, &out, &errb)
		assertDidNotAsk(t, 0, 0, errb.String(), wantRefusal)
	})
}

// assertDidNotAsk pins "it did not ask" by asserting what DID happen.
//
// The first version checked only that stderr lacked "no model configured", and
// two of the six cells then passed for the very failure the test exists to
// catch: with the miss-branch guard removed, a -raw miss reaches ask(), which
// refuses with advice to drop a "?" the line never contained — no ask message,
// assertion satisfied, scripting contract broken (BR-14). The rule: an assertion
// that pins "X did not happen" must assert the positive observable that
// distinguishes X from every other outcome, not the absence of one string.
//
// wantCode of 0 means the route has no exit code of its own (the raw editor).
func assertDidNotAsk(t *testing.T, code, wantCode int, stderr, want string) {
	t.Helper()
	if strings.Contains(stderr, "no model configured") {
		t.Errorf("-raw routed to the model: %q", stderr)
	}
	if !strings.Contains(stderr, want) {
		t.Errorf("stderr = %q, want it to contain %q — the ABSENCE of the ask message is not evidence the right thing happened", stderr, want)
	}
	if wantCode != 0 && code != wantCode {
		t.Errorf("exit = %d, want %d", code, wantCode)
	}
}

// BR-12: what recall stores must RE-SUBMIT TO THE SAME MEANING. A forcing prefix
// is part of the meaning, so stripping it inverts the line: `\how so` recalled as
// `how so` re-submits as a question, which is what the hatch was typed to
// prevent.
func TestRecallPreservesWhatALineMeant(t *testing.T) {
	for _, tc := range []struct{ name, typed, want string }{
		{"a forced lookup keeps its backslash", `\how so`, `\how so`},
		{"a forced question keeps its mark", "?why", "?why"},
		{"an ordinary word is stored collapsed", "hot  dog", "hot dog"},
		{"a command is stored as typed", "/history 7", "/history 7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseREPLLine(tc.typed, false).recallLine(); got != tc.want {
				t.Errorf("recallLine(%q) = %q, want %q", tc.typed, got, tc.want)
			}
			// And the round trip that makes it matter: what comes back must
			// parse to the same kind it was.
			first, again := parseREPLLine(tc.typed, false), parseREPLLine(tc.want, false)
			if first.kind != again.kind || first.literal != again.literal {
				t.Errorf("re-submitting %q changed the meaning: kind %v/%v literal %v/%v",
					tc.want, first.kind, again.kind, first.literal, again.literal)
			}
		})
	}
}

// The raw loop writes three classes of message of its own, and a bare "\n" in
// raw mode starts the next line at the current column. Every class gets an
// assertion on the EMITTED BYTES, because two placement fixes shipped unpinned
// and the family recurred three times.
func TestRawLoopMessagePlacement(t *testing.T) {
	for _, tc := range []struct {
		name, keys string
		wantErase  bool
	}{
		{"the bare-? note", "?\r", true},
		{"a forced ask", "?why\r", false},
		{"an unforced ask", aQuestion + "\r", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rig, opt, _, finish := editorRig(t, "sycophantic", true)
			var out, errb bytes.Buffer
			// A RECORDING cooked. The rig's is a no-op closure, so the two ask
			// rows previously asserted only on stdout while `ask` writes to
			// stderr — and deleting `cooked(...)` from askInSession, which in
			// production is the thing that makes the message's "\n" translate
			// at all, left the suite green (BR-13).
			var duringCooked strings.Builder
			cooked := func(run func()) error {
				before := errb.Len()
				run()
				duringCooked.WriteString(errb.String()[before:])
				return nil
			}
			runEditor(t.Context(), scriptKeys(tc.keys), rig.deps, opt, cooked, finish, &out, &errb)

			assertNoBareNewline(t, out.String(), "stdout")
			if tc.wantErase {
				// Written in RAW mode, so it carries its own escapes.
				if !strings.Contains(errb.String(), eraseLine) {
					t.Errorf("no eraseLine: the message is appended to the line the user typed: %q", errb.String())
				}
				assertNoBareNewline(t, errb.String(), "stderr")
				return
			}
			// Written in COOKED mode, which is what lets it use a bare "\n".
			if !strings.Contains(duringCooked.String(), "define:") {
				t.Errorf("the message was not written inside cooked mode; cooked saw %q, stderr = %q",
					duringCooked.String(), errb.String())
			}
		})
	}
}

// The two ask routes differ only in whether the dictionary was consulted, so
// they must land at the same height. They did not: the shared closure emitted a
// second "\r\n" that only the unforced route had already written.
func TestForcedAndUnforcedAsksRenderAtTheSameHeight(t *testing.T) {
	framing := func(keys string) int {
		rig, opt, cooked, finish := editorRig(t, "sycophantic", true)
		var out, errb bytes.Buffer
		runEditor(t.Context(), scriptKeys(keys), rig.deps, opt, cooked, finish, &out, &errb)
		return strings.Count(out.String(), "\r\n")
	}
	if forced, unforced := framing("?why\r"), framing(aQuestion+"\r"); forced != unforced {
		t.Errorf("forced ask wrote %d newlines, unforced wrote %d — the two routes render at different heights",
			forced, unforced)
	}
}

func assertNoBareNewline(t *testing.T, s, where string) {
	t.Helper()
	// The loop's LAST newline is written after finish() has restored cooked
	// mode, so a bare "\n" there is correct — the terminal translates it again.
	// Everything before it is emitted in raw mode and must carry its own \r.
	s = strings.TrimSuffix(s, "\n")
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' && (i == 0 || s[i-1] != '\r') {
			t.Errorf("bare \\n in %s at %d — the next line starts at the current column: %q",
				where, i, s[max(0, i-24):min(len(s), i+1)])
			return
		}
	}
}

// What #16's messages SAY, asserted against literals written here.
//
// Every earlier round pinned a message's PLACEMENT and none pinned its text, so
// the family recurred four times and a wrong message shipped: `define '\'`
// printed `type a word after "\\"`, two backslashes, because noteEmptyLiteral is
// a Go RAW string and the escape survived into the output. repl_test.go asserted
// `note: noteEmptyLiteral` — the constant compared to itself, which proves a
// branch was selected and nothing about what the user reads (BR-19).
//
// The rule: an assertion on a message compares the bytes the user receives
// against a literal expectation written in the test. Never the constant.
func TestWhatTheMessagesSay(t *testing.T) {
	for _, tc := range []struct {
		name, line string
		wantCode   int
		wantErr    string
	}{
		{"a bare question mark", "?", 2, `define: type a question after "?"` + "\n"},
		{"a bare backslash", `\`, 2, `define: type a word after "\"` + "\n"},
	} {
		// {?, \} x {one-shot, piped}: the same bytes and the same code from both,
		// which is what README's "exit 2" absolute quantifies over.
		t.Run(tc.name+"/one-shot", func(t *testing.T) {
			var out, errb bytes.Buffer
			code := run(t.Context(), []string{tc.line}, testDeps(t), strings.NewReader(""), &out, &errb)
			assertMessage(t, code, tc.wantCode, errb.String(), tc.wantErr)
		})
		t.Run(tc.name+"/piped", func(t *testing.T) {
			rig := newAudioRig(t, "sycophantic", true)
			var out, errb bytes.Buffer
			code := replLines(t.Context(), rig.deps, options{times: 3, locale: "us"},
				strings.NewReader(tc.line+"\n"), &out, &errb, true, false)
			assertMessage(t, code, tc.wantCode, errb.String(), tc.wantErr)
		})
	}
}

// The ask messages, both routes — including that the question is ELIDED, which
// nothing asserted: replacing truncateQuestion(q.text) with q.text was green.
func TestWhatTheAskMessagesSay(t *testing.T) {
	long := "what is the difference between sycophantic and obsequious in formal writing"
	for _, tc := range []struct {
		name    string
		q       question
		wantErr string
	}{
		{"unforced names the word it could not find", question{text: "how so"},
			"define: no model configured; `how so` is not a word\n"},
		{"forced claims nothing about the text", question{text: "why", forced: true},
			"define: no model configured; cannot answer `why`\n"},
		{"a long question is elided", question{text: long},
			"define: no model configured; `what is the difference between sycophant…` is not a word\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var errb bytes.Buffer
			code := ask(options{}, &errb, tc.q)
			assertMessage(t, code, 1, errb.String(), tc.wantErr)
		})
	}
}

func assertMessage(t *testing.T, code, wantCode int, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
	if code != wantCode {
		t.Errorf("exit = %d, want %d", code, wantCode)
	}
}

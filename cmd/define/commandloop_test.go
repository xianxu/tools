package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// A dictionary that fails the test if it is consulted. Dispatch's whole job is
// to keep a command away from the dictionary, so the assertion has to be that
// the dictionary was never reached — not merely that the output looks right.
type refusingDict struct{ t *testing.T }

func (d refusingDict) Lookup(word string) (string, error) {
	d.t.Errorf("the dictionary was consulted for %q; a command must never reach it", word)
	return "", nil
}

var dispatchCmds = []command{
	{name: "help", summary: "list the commands", run: runHelp},
	{name: "history", summary: "words looked up recently",
		run: func(c commandCtx, args []string) int {
			c.stdout.Write([]byte("HISTORY RAN args=" + strings.Join(args, ",") + "\n"))
			return 0
		}},
}

func TestParseREPLLineClassifiesCommands(t *testing.T) {
	for _, tc := range []struct {
		in   string
		kind replKind
		name string
		args []string
	}{
		{"/history", cmdCommand, "history", nil},
		{"/history 7", cmdCommand, "history", []string{"7"}},
		{"/", cmdCommand, "", nil},
		{"hot dog", cmdDefine, "", nil},
		{"", cmdNothing, "", nil},
	} {
		t.Run(tc.in, func(t *testing.T) {
			got := parseREPLLine(tc.in, false)
			if got.kind != tc.kind || got.name != tc.name {
				t.Errorf("parseREPLLine(%q) = kind %v name %q, want kind %v name %q",
					tc.in, got.kind, got.name, tc.kind, tc.name)
			}
			if !reflect.DeepEqual(got.args, tc.args) {
				t.Errorf("args = %v, want %v", got.args, tc.args)
			}
		})
	}
}

// PQ-2: the first draft dispatched in submitLine, which ONLY the raw editor
// reaches. `echo /history | define` would have gone to the dictionary. Both
// loops get their own test for exactly that reason.
func TestLineLoopDispatchesCommands(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}
	rig.deps.stdinIsTerminal = func() bool { return false }

	var out, errb bytes.Buffer
	code := replLines(t.Context(), nil, rig.deps, options{times: 3, locale: "us"},
		strings.NewReader("/help\n"), &out, &errb, true, false)

	if code != 0 {
		t.Errorf("exit = %d, stderr = %s", code, errb.String())
	}
	// Asserted on the command's OUTPUT, never on "/help": the raw editor echoes
	// the submitted line, so matching the name matches the echo and passes with
	// dispatch deleted (BR-3, measured). "list the commands" is a summary only
	// runHelp can emit.
	if !strings.Contains(out.String(), "list the commands") {
		t.Errorf("the piped loop did not run the command: %q", out.String())
	}
}

func TestRawEditorDispatchesCommands(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}

	var out, errb bytes.Buffer
	code := runEditor(t.Context(), scriptKeys("/help\r"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

	if code != 0 {
		t.Errorf("exit = %d, stderr = %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "list the commands") {
		t.Errorf("the raw editor did not run the command: %q", out.String())
	}
}

func TestUnknownCommandSuggestsWithoutDefining(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}

	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("/histry\r"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

	if !strings.Contains(errb.String(), "/histry") {
		t.Errorf("the unknown command was not named: %q", errb.String())
	}
	// Against the LIVE registry. At M1 this asserted /help, because one command
	// was registered and every near-miss got the whole menu; with /history
	// registered it can assert the suggestion it was always meant to.
	if !strings.Contains(errb.String(), "did you mean /history") {
		t.Errorf("no suggestion offered: %q", errb.String())
	}
}

func TestDispatchCommand(t *testing.T) {
	for _, tc := range []struct {
		name     string
		line     string
		wantCode int
		wantOut  string
		wantErr  string
	}{
		{"runs the command", "/history", 0, "HISTORY RAN args=", ""},
		{"passes arguments", "/history 7", 0, "HISTORY RAN args=7", ""},
		{"a bare slash lists the menu", "/", 0, "/history", ""},
		{"unknown suggests the near match", "/histry", 2, "", "did you mean /history"},
		{"nothing close lists everything", "/qqqqqq", 2, "", "/help"},
		// The forgiving half of the case policy: completion is exact, dispatch
		// accepts what was submitted.
		{"dispatch accepts mixed case", "/HISTORY", 0, "HISTORY RAN", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			cc := commandCtx{stdout: &out, stderr: &errb}
			code := dispatchCommand(parseREPLLine(tc.line, false), dispatchCmds, cc)

			if code != tc.wantCode {
				t.Errorf("exit = %d, want %d", code, tc.wantCode)
			}
			if tc.wantOut != "" && !strings.Contains(out.String(), tc.wantOut) {
				t.Errorf("stdout = %q, want it to contain %q", out.String(), tc.wantOut)
			}
			if tc.wantErr != "" && !strings.Contains(errb.String(), tc.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", errb.String(), tc.wantErr)
			}
		})
	}
}

// BR-2: completionsFor having the right answer is not the same as the editor
// asking it. This pins the wiring — type "/his" and the grey tail must be
// offered from the COMMAND set, which history could never produce.
func TestEditorSuggestsFromCommands(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}
	// History that would suggest something else entirely if it were consulted.
	h := &memHistory{}
	h.Add("hibernate")
	rig.deps.history = h

	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("/hel\x03"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

	if !strings.Contains(out.String(), greyOn+"p") {
		t.Errorf("no grey completion from the command set: %q", out.String())
	}
	if strings.Contains(out.String(), "ibernate") {
		t.Errorf("the line starts with / but history was consulted: %q", out.String())
	}
}

// The menu has to actually reach the screen, not merely be computable.
// menuLines being right is a different claim from the editor painting it.
func TestEditorShowsTheCommandMenu(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}

	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("/\x03"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

	if !strings.Contains(out.String(), "list the commands") {
		t.Errorf("typing / did not show the menu: %q", out.String())
	}
}

func TestTypingNarrowsTheMenuAndAWordHidesIt(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}

	var out, errb bytes.Buffer
	// A word, not a command: the screen must not sprout a menu under it.
	runEditor(t.Context(), scriptKeys("syc\x03"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)
	if strings.Contains(out.String(), "list the commands") {
		t.Errorf("a word drew the command menu: %q", out.String())
	}

	// A prefix that matches nothing: the menu must disappear rather than
	// leaving a stale set on screen.
	out.Reset()
	runEditor(t.Context(), scriptKeys("/zzz\x03"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)
	last := out.String()[strings.LastIndex(out.String(), "/zzz"):]
	if strings.Contains(last, "list the commands") {
		t.Errorf("a non-matching prefix left the menu on screen: %q", last)
	}
}

// What the grey tail SHOWS must be what Tab ACCEPTS.
//
// The loop computes `matches` from the line BEFORE the keystroke (Apply needs
// that list to anchor a history walk) and then drew the suggestion against the
// line AFTER it. Both lists were history until #15, so a stale superset usually
// had the same first match and nothing showed. Command mode made the stale list
// come from a DIFFERENT NAMESPACE: typing "/" rendered a suggestion out of
// history — which contains "/history", because submitted commands are recalled —
// while the next keystroke resolved Tab against the command set and accepted
// "/help". Reported from the terminal: grey said history, Tab gave help.
func TestSuggestionMatchesWhatTabAccepts(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}
	// A history that has seen commands, which is what any real session has.
	h := &memHistory{}
	h.Add("/history")
	rig.deps.history = h

	var out, errb bytes.Buffer
	runEditor(t.Context(), scriptKeys("/\x03"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

	if strings.Contains(out.String(), greyOn+"history") {
		t.Errorf("typing / suggested from HISTORY; the menu below it lists commands: %q", out.String())
	}
	if !strings.Contains(out.String(), greyOn+"help") {
		t.Errorf("no command suggestion after /: %q", out.String())
	}

	// And the acceptance agrees: Tab commits the tail that was shown.
	out.Reset()
	runEditor(t.Context(), scriptKeys("/\t\x03"), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)
	if !strings.Contains(out.String(), "/help") {
		t.Errorf("Tab did not accept the suggestion that was displayed: %q", out.String())
	}
}

// keySeq scripts an exact key sequence, for tests that need keys scriptKeys
// does not spell (arrows, Tab).
func keySeq(ks ...Key) <-chan Key {
	ch := make(chan Key, len(ks)+1)
	for _, k := range ks {
		ch <- k
	}
	close(ch)
	return ch
}

// BR-19's rule: a value or effect that only a LOOP SHELL supplies must be
// pinned by a test that drives that loop shell. Every one of these was green
// with its wiring deleted, because the existing tests either built the callee's
// context by hand or drove the OTHER loop.

// runEditor's cc.setTimes. The M3 test drove replLines — the piped loop — so
// deleting the raw loop's wiring left /sound silently broken at the actual TUI
// prompt, which is the only place the operator asked for it.
func TestRawEditorSoundChangesTheSession(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	var out, errb bytes.Buffer

	ks := append(runes("/sound 1"), Key{Kind: KeyEnter})
	ks = append(ks, runes("sycophantic")...)
	ks = append(ks, Key{Kind: KeyEnter}, Key{Kind: KeyInterrupt})
	runEditor(t.Context(), keySeq(ks...), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

	if got := rig.player.count(); got != 1 {
		t.Errorf("played %d times after /sound 1 at the raw prompt, want 1", got)
	}
}

// runEditor's hist.Add for commands: a submitted command must be recallable.
func TestRawEditorRecallsSubmittedCommands(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	rig.deps.history = &memHistory{}
	var out, errb bytes.Buffer

	ks := append(runes("/sound"), Key{Kind: KeyEnter}, Key{Kind: KeyUp}, Key{Kind: KeyInterrupt})
	runEditor(t.Context(), keySeq(ks...), nil, rig.deps, opt, paintInto(&out), finish, &out, &errb)

	// Asserted only on what came AFTER the command ran. The submitted line is
	// ECHOED through RenderLine, which uses the same inputOn sequence, so
	// searching the whole stream matches the echo and passes with hist.Add
	// deleted — measured. That is BR-3's mistake, repeated inside the fix for
	// BR-19; the marker is the one thing only a completed dispatch emits.
	after := out.String()
	i := strings.Index(after, "playing")
	if i < 0 {
		t.Fatalf("the command never ran, so this test cannot see a recall: %q", after)
	}
	if !strings.Contains(after[i:], inputOn+"/sound") {
		t.Errorf("Up did not recall the submitted command: %q", after[i:])
	}
}

// run()'s one-shot dispatch, driven through run rather than dispatchCommand.
func TestOneShotCommandGoesThroughRun(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}
	rig.deps.stdinIsTerminal = func() bool { return false }

	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-no-audio", "/help"}, rig.deps, strings.NewReader(""), &out, &errb); code != 0 {
		t.Errorf("exit = %d, stderr = %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "list the commands") {
		t.Errorf("define /help did not run the command: %q", out.String())
	}
}

// replLines' command exit code, which used to be collapsed into a generic 1.
func TestPipedLoopReturnsTheCommandsExitCode(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}
	rig.deps.stdinIsTerminal = func() bool { return false }

	var out, errb bytes.Buffer
	code := replLines(t.Context(), nil, rig.deps, options{times: 3, locale: "us"},
		strings.NewReader("/histry\n"), &out, &errb, true, false)
	if code != 2 {
		t.Errorf("exit = %d, want 2 — a usage error, not the generic failure 1", code)
	}
}

// BR-20: `define /history 7` printed a usage dump while `echo '/history 7' |
// define` ran it. The behaviour was fixed without a pin, which is the same
// omission BR-19 names — so it gets one that drives run().
func TestOneShotCommandTakesArguments(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}
	rig.deps.stdinIsTerminal = func() bool { return false }
	rig.deps.history, rig.deps.capture, rig.deps.deck = nil, nil, nil
	rig.deps.newStore = openStore // /history needs a deck to report an empty one
	t.Chdir(t.TempDir())

	var out, errb bytes.Buffer
	code := run(t.Context(), []string{"-no-audio", "/history", "7"}, rig.deps, strings.NewReader(""), &out, &errb)

	if code != 0 {
		t.Errorf("exit = %d, stderr = %s", code, errb.String())
	}
	if strings.Contains(errb.String(), "usage:") {
		t.Errorf("a command with an argument was rejected by the word count: %q", errb.String())
	}
	// The window has to REACH the command, not just get past the arity guard.
	if !strings.Contains(out.String(), "7 days") {
		t.Errorf("the argument did not reach the command: %q", out.String())
	}
	// Two plain words are still a usage error: the exemption is for commands.
	errb.Reset()
	if code := run(t.Context(), []string{"-no-audio", "hot", "dog"}, rig.deps, strings.NewReader(""), &out, &errb); code != 2 {
		t.Errorf("two words exit = %d, want 2", code)
	}
}

// BR-26: opening the store constructs storeHistory, which READS the whole event
// log. A command that reads nothing must not pay for it — the same invariant #4
// established for usage errors, which the command path was skipping.
func TestCommandThatReadsNothingDoesNotOpenTheLog(t *testing.T) {
	rig := newAudioRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}
	rig.deps.stdinIsTerminal = func() bool { return false }
	rig.deps.history, rig.deps.capture, rig.deps.deck = nil, nil, nil
	rig.deps.newStore = openStore

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "events"), 0o755); err != nil {
		t.Fatal(err)
	}
	torn := "- word: sycophantic\n  kind: looked-up\n  found: true\n  at: 2020-01-01T10:00:00-07:00\n- word: torn\n  kind: looked"
	if err := os.WriteFile(filepath.Join(dir, "events", "2020-01-01.yaml"), []byte(torn), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	for _, args := range [][]string{{"-no-audio", "/help"}, {"-no-audio", "/histry"}} {
		var out, errb bytes.Buffer
		run(t.Context(), args, rig.deps, strings.NewReader(""), &out, &errb)
		if strings.Contains(errb.String(), "torn record") {
			t.Errorf("%v read the event log: %q", args, errb.String())
		}
	}

	// …and /history, which does read it, still does.
	var out, errb bytes.Buffer
	run(t.Context(), []string{"-no-audio", "/history"}, rig.deps, strings.NewReader(""), &out, &errb)
	if !strings.Contains(errb.String(), "torn record") {
		t.Errorf("/history did NOT read the log, so this test cannot tell the two apart: %q", errb.String())
	}
}

// BR-19's successor, and the one claim D4 leaves to pin.
//
// The dropdown used to be erased before a command's output was written under it,
// by counting the rows drawn and walking the cursor back over exactly that many
// — arithmetic that carried a documented known limit for when the count was
// wrong, and whose clearMenu shipped in the same commit as the rule, itself
// unpinned. #30 deletes it: the menu is an ARGUMENT to a whole-frame redraw,
// recomputed from the line as it stands, so a stale row cannot survive a submit.
// There is nothing left to erase and nothing left to miscount.
func TestSubmitLeavesNoMenuInTheFrame(t *testing.T) {
	rig, opt, finish := editorRig(t, "sycophantic", true)
	rig.deps.dict = refusingDict{t}

	var out, errb bytes.Buffer
	view := paintInto(&out)
	// /sound, deliberately: its OUTPUT ("playing 3×") shares no text with its
	// menu row, so a marker cannot match the menu instead of the output.
	ks := append(runes("/sound"), Key{Kind: KeyEnter}, Key{Kind: KeyInterrupt})
	runEditor(t.Context(), keySeq(ks...), nil, rig.deps, opt, view, finish, &out, &errb)

	if !strings.Contains(out.String(), "playing") {
		t.Fatalf("the command never ran: %q", out.String())
	}
	// The dropdown really was on screen while the command was being typed —
	// without this the assertion below would hold for a menu that never drew.
	drawn := false
	for _, m := range view.menus {
		if len(m) > 0 {
			drawn = true
		}
	}
	if !drawn {
		t.Fatal("the dropdown never appeared, so its absence afterwards proves nothing")
	}
	if last := view.menus[len(view.menus)-1]; len(last) != 0 {
		t.Errorf("the frame drawn after the command still lists the dropdown: %v", last)
	}
}

// The raw editor's /pron branch: a command RECORDS a language and the loop
// replays it (#29).
//
// The hazard that shape was built for is gone. Commands used to be dispatched
// inside cooked(), and playing there handed Ctrl-C to the line discipline, which
// swallowed the byte — the key reader saw nothing and the session looked frozen
// for the length of the recording. #30 D4 deletes the cooked block: raw mode is
// continuous now, so there is no mode for playback to happen in the wrong one
// of, and workshop/lessons.md's "render cooked, play raw" is answered by there
// being nothing left to flap. With the hazard went the instrument this test used
// to carry — CDN requests counted inside the cooked callback.
//
// What is left is still worth pinning, and is the reason the shape survives D4:
// /pron and a bare Enter are ONE replay path, so the SOURCE spelling is fetched
// and played by the loop rather than by a second implementation inside the
// command. The repo pairs TestLineLoopDispatchesCommands with
// TestRawEditorDispatchesCommands for the two-loops reason; this is /pron's half.
func TestRawEditorPronReplaysThroughTheLoop(t *testing.T) {
	en := voice{Lang: "en", Locale: "us"}
	es := voice{Lang: "es", Locale: "es"}
	english := AudioCandidates("jalapeno", en)[0]
	spanish := AudioCandidates("jalapeño", es)[0]

	rig := newAudioRigServing(t, english, spanish)
	rig.deps.stdinIsTerminal = func() bool { return true }
	// locale EMPTY, unlike editorRig's "us", and the difference is not cosmetic:
	// opt.locale is the -locale FLAG, and #29 D4 keeps it qualifying whatever
	// language is in effect. With "us" this same script correctly asks for
	// jalapeño_es_us — Latin American seseo — which is the flag working, not a
	// bug. An unqualified session is what a reader of this test should picture.
	opt := options{times: 1, tty: true, color: true}

	var out, errb bytes.Buffer
	code := runEditor(t.Context(), scriptKeys("jalapeno\r/pron es\r"), nil,
		rig.deps, opt, paintInto(&out), func() {}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}

	// It replayed, in Spanish, on the SOURCE spelling.
	want := []string{stripHost(t, english, audioBase), stripHost(t, spanish, audioBase)}
	if got := rig.cdn.Requested(); !slices.Equal(got, want) {
		t.Errorf("the CDN was asked for:\n  %q\nwant:\n  %q", got, want)
	}
	if got := rig.player.count(); got != 2 {
		t.Errorf("played %d times, want 2 — the lookup and the /pron replay", got)
	}
}

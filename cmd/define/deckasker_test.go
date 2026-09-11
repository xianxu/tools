package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/schedule"
)

// THE FOUR INPUTS, AND THE ORDER IS THE DESIGN.
//
// Each row asserts BOTH whether the question was put and what came back —
// because "it allowed" and "it allowed without asking" are different behaviours
// and only one of them is right per row.
func TestDeckAskerPolicy(t *testing.T) {
	for _, tc := range []struct {
		name        string
		alreadyDeck bool
		here        bool
		tty         bool
		typed       string
		wantAsked   bool
		wantAllow   bool
	}{
		{"an existing deck is never asked about", true, false, true, "", false, true},
		{"an existing deck is not asked about without a terminal either", true, false, false, "", false, true},
		{"--here creates without asking", false, true, true, "", false, true},
		{"--here works with no terminal, which is its whole point", false, true, false, "", false, true},
		{"no terminal means no deck and no question", false, false, false, "", false, false},
		{"y creates", false, false, true, "y\n", true, true},
		{"yes creates", false, false, true, "yes\n", true, true},
		{"Y creates", false, false, true, "Y\n", true, true},
		{"n declines", false, false, true, "n\n", true, false},
		{"a bare Enter declines — the default is the safe one", false, false, true, "\n", true, false},
		{"anything else declines", false, false, true, "maybe\n", true, false},
		{"EOF mid-question declines", false, false, true, "", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.alreadyDeck {
				if err := os.MkdirAll(filepath.Join(dir, "words"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			var out bytes.Buffer
			got := deckAsker(t.Context(), dir, options{here: tc.here}, strings.NewReader(tc.typed), &out,
				func() bool { return tc.tty })()

			if got != tc.wantAllow {
				t.Errorf("allowed = %v, want %v (said %q)", got, tc.wantAllow, out.String())
			}
			// "Did it ask?" is visible in what it printed: the question is the only
			// thing that ends in a prompt.
			didAsk := strings.Contains(out.String(), "Create one here?")
			if didAsk != tc.wantAsked {
				t.Errorf("asked = %v, want %v; output was %q", didAsk, tc.wantAsked, out.String())
			}
		})
	}
}

// A PIPE IS TOLD WHY, and it is told by the PERMISSION rather than by the asker.
//
// The explanation used to live in deckAsker, so it was printed only when a WRITE
// resolved the question. A READ settling the state quietly first meant the same
// invocation said nothing — measured on the real binary, `echo word | define`
// explained itself and `define word` with stdin redirected did not (#50 BR-12).
func TestADeniedSessionIsToldWhyOnEitherPath(t *testing.T) {
	for _, tc := range []struct {
		name  string
		drive func(*deckPermission)
	}{
		{"a read settles it first", func(p *deckPermission) { p.saving() }},
		{"a write settles it first", func(p *deckPermission) { p.allowed() }},
		{"resolve settles it first", func(p *deckPermission) { p.resolve() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			var out bytes.Buffer
			perm := newDeckPermission(func() bool { return true }).
				withQuiet(func() (deckDecision, deckReason) {
					return deckPolicy(dir, options{}, func() bool { return false })
				}, func(r deckReason) { explainDenial(r, dir, &out) })

			tc.drive(perm)

			if !strings.Contains(out.String(), "--here") {
				t.Errorf("the learner was not told why nothing is being saved; said %q", out.String())
			}
		})
	}
}

// AND ONLY ONCE. A lookup reads and then writes; saying it twice is noise.
func TestTheExplanationIsGivenOnce(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	perm := newDeckPermission(func() bool { return true }).
		withQuiet(func() (deckDecision, deckReason) {
			return deckPolicy(dir, options{}, func() bool { return false })
		}, func(r deckReason) { explainDenial(r, dir, &out) })

	perm.saving()
	perm.allowed()
	perm.resolve()

	if n := strings.Count(out.String(), "--here"); n != 1 {
		t.Errorf("explained %d times, want exactly 1:\n%s", n, out.String())
	}
}

// -raw NEVER ASKS AND NEVER EXPLAINS, because -raw RECORDS NOTHING (#50 BR-14).
//
// The usage text has said "-raw records nothing, because it is for scripts" since
// #2. Asking permission to create a deck this invocation will not write to is
// exactly the false alarm the lazy design exists to prevent — and it was reaching
// users: `define -raw word` in a fresh directory put a question on screen, and
// piped it advised scripts to pass --here for a deck it would never have made.
func TestRawNeitherAsksNorAdvises(t *testing.T) {
	for _, tty := range []bool{true, false} {
		t.Run(map[bool]string{true: "on a terminal", false: "piped"}[tty], func(t *testing.T) {
			dir := t.TempDir()
			d, why := deckPolicy(dir, options{raw: true}, func() bool { return tty })
			if d != deckDeny {
				t.Errorf("-raw decision = %v, want deckDeny; it records nothing", d)
			}
			if why != reasonRecordsNothing {
				t.Errorf("-raw reason = %v, want reasonRecordsNothing", why)
			}

			var out bytes.Buffer
			explainDenial(why, dir, &out)
			if out.Len() != 0 {
				t.Errorf("-raw was explained to the user (%q). Nothing is being lost — "+
					"-raw records nothing by design — so there is nothing to act on and "+
					"advising --here points at a deck it would never have written to.",
					out.String())
			}
		})
	}
}

// A nil terminal predicate is the same as "not a terminal": absent information
// about the terminal must not create a deck.
func TestDeckAskerTreatsANilPredicateAsNoTerminal(t *testing.T) {
	var out bytes.Buffer
	if deckAsker(t.Context(), t.TempDir(), options{}, strings.NewReader("y\n"), &out, nil)() {
		t.Error("a nil terminal predicate allowed creation; absent information about " +
			"the terminal must not be read as permission")
	}
}

// observingReader records whether the deck question was already settled at the
// moment the shell first READ from stdin.
//
// THE ORDERING CLAIM NEEDS AN OBSERVATION DURING THE LOOP, NOT AFTER IT
// (#50 BR-10). A previous version asserted saving() once repl had returned — by
// which time the permission is resolved whether it was settled before the shell
// started or during it. The reviewer proved the hole: moving resolve() BELOW both
// shells left that test green. The claim is about ORDER, so the control has to
// sample the order.
type observingReader struct {
	perm           *deckPermission
	decidedAtFirst bool
	sawRead        bool
}

func (r *observingReader) Read(p []byte) (int, error) {
	if !r.sawRead {
		r.sawRead = true
		_, r.decidedAtFirst = r.perm.saving()
	}
	return 0, io.EOF
}

// THE LINE SHELL SETTLES THE QUESTION BEFORE IT READS A KEY (#50 PQ-3).
//
// NAMED FOR WHAT IT COVERS, which is the correction (#50 BR-10). This used to be
// called "both loop shells" and ran two cases — but BOTH land in replLines: the
// second reaches it through replRaw's non-file-stdin fallback, because nothing
// short of a terminal satisfies replRaw's two requirements (stdin must be an
// *os.File AND enterRaw must succeed on it). The raw shell was pinned zero times
// while the test's name claimed otherwise; its subtest name even said so in
// parentheses, which is a thing to notice rather than to write down.
//
// The raw shell is covered by TestPTYDeckQuestionArrivesBeforeTheEditor
// (pty_conformance_test.go), which drives the real binary through a pty. Both
// cases are kept here because the two ENTRY POINTS still differ — one is chosen
// directly, one via the fallback — and both must settle before reading.
func TestTheLineShellResolvesBeforeReading(t *testing.T) {
	for _, tc := range []struct {
		name string
		tty  bool // opt.tty picks the entry point; both land in replLines here
	}{
		{"chosen directly", false},
		{"reached through replRaw's non-file-stdin fallback", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			asked := 0
			perm := newDeckPermission(func() bool { asked++; return false })

			// testDeps loads the fixture corpus by a path relative to the package
			// directory, so it is built BEFORE chdir.
			d := testDeps(t)
			t.Chdir(dir)
			d.deckPermission = perm
			d.stdinIsTerminal = func() bool { return true }

			in := &observingReader{perm: perm}
			var out, errb bytes.Buffer
			repl(t.Context(), d, options{tty: tc.tty}, in, &out, &errb)

			if !in.sawRead {
				t.Fatal("the shell never read stdin, so this test observed no ordering " +
					"at all — it would pass with the resolution removed entirely")
			}
			if !in.decidedAtFirst {
				t.Errorf("the shell read its first byte of stdin with the deck question " +
					"still unsettled. A question put mid-session races the shell's own " +
					"reader for the answer.")
			}
			if asked != 1 {
				t.Errorf("asked %d times, want exactly 1", asked)
			}
		})
	}
}

// --stats IS HONEST WITHOUT BEING INTRUSIVE, which is the whole point of
// splitting deckPolicy out of deckAsker.
//
// FOUND BY SMOKE TESTING, not by a test: `define --stats` piped in a non-deck
// directory printed "look a word up and it joins your deck" — the exact promise
// this feature exists to stop making — because saving() refused to resolve and so
// reported "undecided". But the answer there needs NOBODY: no terminal means no
// deck. The free cases settle; the one that would prompt does not.
func TestStatsIsHonestWhereTheAnswerNeedsNobody(t *testing.T) {
	for _, tc := range []struct {
		name      string
		deck      bool
		tty       bool
		wantSaved bool
	}{
		{"piped, not a deck: says so", false, false, false},
		{"piped, already a deck: ordinary screen", true, false, true},
		{"a terminal, not a deck: stays undecided rather than prompting", false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.deck {
				if err := os.MkdirAll(filepath.Join(dir, "words"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			asked := 0
			perm := newDeckPermission(func() bool { asked++; return true }).
				withQuiet(func() (deckDecision, deckReason) {
					return deckPolicy(dir, options{}, func() bool { return tc.tty })
				}, nil)

			allowed, decided := perm.saving()
			saving := !decided || allowed
			if saving != tc.wantSaved {
				t.Errorf("saving = %v, want %v", saving, tc.wantSaved)
			}
			if asked != 0 {
				t.Errorf("settling for --stats put the question %d time(s); the free "+
					"answers are free precisely because nobody is asked", asked)
			}
		})
	}
}

// THE EMPTY SCREEN MUST NOT PROMISE A DECK THAT WILL NOT EXIST (#50 PQ-6).
//
// "Nothing yet — look a word up and it joins your deck" is true of a new deck and
// FALSE in a directory define was told not to write to. This screen is precisely
// where the person who ran define in the wrong place ends up.
func TestStatsDoesNotPromiseToSaveWhenItCannot(t *testing.T) {
	notSaving := strings.Join(renderStats(schedule.Stats{}, time.Now(), false), "\n")
	if strings.Contains(notSaving, "joins your deck") {
		t.Errorf("the empty screen promises a word will join the deck, in a directory "+
			"nothing is written to:\n%s", notSaving)
	}
	if !strings.Contains(notSaving, "--here") {
		t.Errorf("the empty screen does not say how to fix it:\n%s", notSaving)
	}

	// And the ordinary empty screen is UNCHANGED — a directory that may still
	// become a deck is one where "it joins your deck" is still true.
	saving := strings.Join(renderStats(schedule.Stats{}, time.Now(), true), "\n")
	if !strings.Contains(saving, "joins your deck") {
		t.Errorf("the ordinary empty screen lost its call to action:\n%s", saving)
	}
}

// --stats NEVER PUTS THE QUESTION. Reading your figures must not become a request
// to create a deck — that is the false alarm the lazy design exists to prevent,
// and false alarms train people to answer yes.
func TestStatsNeverAsks(t *testing.T) {
	dir := t.TempDir()
	d := testDeps(t)
	t.Chdir(dir)

	asked := 0
	d.deckPermission = newDeckPermission(func() bool { asked++; return true })
	d.newStore = openStore

	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"-stats"}, d, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("exit = %d: %s", code, errb.String())
	}
	if asked != 0 {
		t.Errorf("--stats put the deck question %d time(s). It creates nothing, so "+
			"asking permission to create is a false alarm.", asked)
	}
	if names := lsNames(t, dir); len(names) != 0 {
		t.Errorf("--stats created %v in a directory it only read", names)
	}
}

// EVERY SURFACE THAT DESCRIBES CAPTURE MENTIONS THE QUESTION (#50 BR-11).
//
// The rule the review stated, taken as a rule rather than an instance: when
// behaviour changes, enumerate every place the OLD behaviour is asserted and
// sweep them in one round. --help is the first surface a user types, and it was
// still promising unconditional recording after the READMEs and the atlas had
// been updated — the gap was that nothing enumerated the surfaces.
func TestEverySurfaceDescribingCaptureMentionsTheQuestion(t *testing.T) {
	surfaces := map[string]string{
		"--help (fs.Usage)":    usageText(t),
		"README.md":            readFileForTest(t, filepath.Join("..", "..", "README.md")),
		"cmd/define/README.md": readFileForTest(t, "README.md"),
		"atlas/define.md":      readFileForTest(t, filepath.Join("..", "..", "atlas", "define.md")),
	}
	// TWO HALVES, because a document can mention -here in one section and assert
	// the old unconditional behaviour in another — which is exactly what
	// cmd/define/README.md did after the first pass at this family. Presence is
	// not enough; the superseded claim has to be ABSENT.
	superseded := []string{
		"*Every* successful lookup",      // true only in a directory already a deck
		"records what you look up under", // the usage prose's old promise
	}
	for name, text := range surfaces {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(text, "--here") && !strings.Contains(text, "-here") {
				t.Errorf("%s describes what define writes but never mentions -here, so a "+
					"reader learns the old unconditional behaviour", name)
			}
			for _, claim := range superseded {
				if !strings.Contains(text, claim) {
					continue
				}
				// Present is only a failure when it stands UNQUALIFIED — the
				// sentence has to carry the condition with it.
				idx := strings.Index(text, claim)
				window := text[idx:min(len(text), idx+400)]
				if !strings.Contains(window, "not one yet") && !strings.Contains(window, "ASKS before") {
					t.Errorf("%s still asserts %q without the condition. Mentioning -here "+
						"elsewhere in the file does not repair a sentence that promises "+
						"unconditional recording where a reader will meet it.", name, claim)
				}
			}
		})
	}
}

// THE READMEs QUOTE THE PROMPT, so the quote cannot drift from the program.
func TestTheREADMEQuotesTheRealPrompt(t *testing.T) {
	// The prompt is a format string; the stable half is what a reader recognises.
	stable := "Create one here? [y/N]"
	if !strings.Contains(deckPrompt, stable) {
		t.Fatalf("deckPrompt = %q, which no longer contains %q — update this test AND "+
			"the READMEs together", deckPrompt, stable)
	}
	doc := readFileForTest(t, "README.md")
	if !strings.Contains(doc, stable) {
		t.Errorf("cmd/define/README.md shows a prompt that is not the one the program "+
			"prints; it must quote %q", stable)
	}
}

func readFileForTest(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}

// usageText captures what `define --help` actually prints.
func usageText(t *testing.T) string {
	t.Helper()
	var out, errb bytes.Buffer
	run(t.Context(), []string{"-h"}, testDeps(t), strings.NewReader(""), &out, &errb)
	return out.String() + errb.String()
}

// THE ANSWER IS READ WITHOUT BUFFERING AHEAD (#50 BR-6).
//
// bufio.Reader is the obvious choice and the wrong one: `in` is the program's
// shared stdin, and a buffered reader pulls as much as it can get. Bytes it
// swallowed past the newline are lost to whoever reads stdin next — which, since
// the loop shells settle this question before they start, is the loop itself. The
// learner would answer "y" and watch their next line vanish.
//
// Asserted on the REMAINDER, because that is the observable difference; a test on
// the answer alone passes under either implementation.
func TestTheAnswerDoesNotSwallowTheRestOfStdin(t *testing.T) {
	in := strings.NewReader("y\nsycophantic\narrondissement\n")
	var out bytes.Buffer

	if !deckAsker(t.Context(), t.TempDir(), options{}, in, &out, func() bool { return true })() {
		t.Fatal("y did not allow")
	}

	rest, err := io.ReadAll(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(rest) != "sycophantic\narrondissement\n" {
		t.Errorf("after reading the answer, stdin holds %q — the question consumed "+
			"input meant for the loop. Everything past the newline belongs to whoever "+
			"reads stdin next.", string(rest))
	}
}

// CTRL-C AT THE QUESTION DECLINES, RATHER THAN BLOCKING FOREVER (#50 BR-15).
//
// The question is put BEFORE the loop's own stdin reader exists, so nothing else
// is watching the terminal — an unarmed read simply sat there and ^C did nothing.
// An interrupt is the clearest "no" a person can give.
func TestInterruptAtTheQuestionDeclines(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	var out bytes.Buffer

	// blockingReader (repl_test.go) never returns — the learner has been asked and
	// is not typing. Reused rather than redeclared: it models exactly this.
	done := make(chan bool, 1)
	go func() {
		done <- deckAsker(ctx, t.TempDir(), options{}, blockingReader{}, &out,
			func() bool { return true })()
	}()

	cancel()

	select {
	case allowed := <-done:
		if allowed {
			t.Error("an interrupt allowed the deck to be created; ^C is a no")
		}
		if !strings.Contains(out.String(), "not saving") {
			t.Errorf("the interrupt was silent; said %q", out.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the question did not return after the context was cancelled — it " +
			"blocks forever, and since the question is put before the loop's reader " +
			"exists there is nothing else watching the terminal")
	}
}

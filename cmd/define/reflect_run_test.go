package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/tools/cmd/define/store"
	"github.com/xianxu/tools/internal/llm"
	"github.com/xianxu/tools/internal/llm/llmtest"
)

var reflectClock = time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)

// reflectRig wires runReflect against the WIRE-LEVEL fake and a real YAML store
// in a temp dir, so user-model.md is written and read through the production
// path rather than seeded into a field.
func reflectRig(t *testing.T, words int) (deps, *llmtest.Fake, *store.YAML, string) {
	t.Helper()
	fake := llmtest.NewFake(t)
	dir := t.TempDir()
	st := store.NewYAML(dir, nil)
	for i := range words {
		w := store.Word{
			Text:      deckWord(i),
			FirstSeen: reflectClock.AddDate(0, 0, -20+i),
			LastSeen:  reflectClock.AddDate(0, 0, -i),
			Lookups:   1,
		}
		if err := st.Upsert(w); err != nil {
			t.Fatal(err)
		}
		if err := st.AppendEvent(store.ReviewEvent{
			Word: w.Text, Kind: store.EventLookedUp, Found: true, At: w.LastSeen,
		}); err != nil {
			t.Fatal(err)
		}
	}
	d := testDeps(t)
	d.deck = st
	d.clock = store.FixedClock(reflectClock)
	d.newLLM = llm.New
	d.getenv = envFor(fake.URL)
	return d, fake, st, dir
}

func deckWord(i int) string {
	return []string{
		"certiorari", "estoppel", "dicta", "sycophantic", "ephemeral", "quokka",
		"defenestrate", "gaslighting", "alewife", "bargainer", "parrot", "pulp",
		"concrete", "minute",
	}[i%14]
}

// The answer the fake returns: a real shape, with one fabricated citation so the
// check has something to drop.
const reflectReply = `{"level":{"band":"C1","rationale":"Reaches for precise low-frequency words.",` +
	`"evidence_words":["certiorari","estoppel"]},` +
	`"domains":[{"name":"law","share":0.5,"evidence_words":["certiorari","dicta"],"directive":"Draw comparables from judicial prose."},` +
	`{"name":"sailing","share":0.2,"evidence_words":["luffing"],"directive":"Use nautical usages."}]}`

func TestReflectWritesAModelFromTheDeck(t *testing.T) {
	d, fake, st, _ := reflectRig(t, 14)
	fake.Script("", llmtest.Reply{Text: reflectReply})

	var out, errb bytes.Buffer
	if code := runReflect(t.Context(), d, options{}, &out, &errb); code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, errb.String())
	}

	got, err := st.UserModel()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"type: user-model", "C1", "certiorari", "judicial prose", correctionsMarker} {
		if !strings.Contains(got, want) {
			t.Errorf("the written model is missing %q:\n%s", want, got)
		}
	}
	// D1 end to end: the fabricated domain never reaches the file.
	if strings.Contains(got, "sailing") || strings.Contains(got, "luffing") {
		t.Errorf("an unsupported claim was written:\n%s", got)
	}
	if !strings.Contains(errb.String(), "sailing") {
		t.Errorf("a dropped claim must be said out loud; stderr = %q", errb.String())
	}
}

// D4: same clock, same store, same answer → the same bytes. Without it there is
// no way to tell a real change in the learner from noise in the generator.
func TestReflectIsIdempotent(t *testing.T) {
	d, fake, st, _ := reflectRig(t, 14)
	fake.Script("", llmtest.Reply{Text: reflectReply}, llmtest.Reply{Text: reflectReply})

	var out, errb bytes.Buffer
	mustReflect(t, d, &out, &errb)
	first, _ := st.UserModel()
	mustReflect(t, d, &out, &errb)
	second, _ := st.UserModel()

	if first == "" {
		t.Fatal("nothing was written, so 'unchanged' proves nothing")
	}
	if first != second {
		t.Errorf("regeneration changed the file:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

// The Done-when row: a hand-written corrections section survives byte-for-byte.
func TestReflectPreservesCorrections(t *testing.T) {
	d, fake, st, dir := reflectRig(t, 14)
	fake.Script("", llmtest.Reply{Text: reflectReply}, llmtest.Reply{Text: reflectReply})

	var out, errb bytes.Buffer
	mustReflect(t, d, &out, &errb)

	// The learner argues with it, in their own words and their own spacing.
	const mine = "\n\nI read these for pleasure, not for the bar exam.\n\n   — me, tersely\n"
	before, _ := st.UserModel()
	// The production scanner, not strings.Index: the first draft of this test
	// used Index and cut the file inside its own header comment, which is the
	// very confusion firstMarkerOutsideAFence exists to prevent.
	i := firstMarkerOutsideAFence(before)
	if i < 0 {
		t.Fatalf("no marker in the generated file:\n%s", before)
	}
	edited := before[:i] + correctionsMarker + mine
	if err := os.WriteFile(filepath.Join(dir, "user-model.md"), []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	mustReflect(t, d, &out, &errb)

	after, _ := st.UserModel()
	if !strings.HasSuffix(after, correctionsMarker+mine) {
		t.Errorf("the learner's own text did not survive regeneration:\n%q", after)
	}
	// Note deliberately NOT asserted here: that the analysis CHANGED. Under a
	// fixed clock and a scripted reply it is byte-identical by design — that is
	// what TestReflectIsIdempotent asserts. The hole BR-2 named is closed by
	// mustReflect, which fails when the run did not write at all; asserting a
	// change on top of it would contradict idempotency.
}

// D2: below the floor there is nothing worth reflecting on, and a confidently
// generic file would then steer authoring. Absence already degrades cleanly.
func TestReflectRefusesATinyDeck(t *testing.T) {
	d, fake, st, _ := reflectRig(t, 3)
	fake.Script("", llmtest.Reply{Text: reflectReply})

	var out, errb bytes.Buffer
	code := runReflect(t.Context(), d, options{}, &out, &errb)

	if code == 0 {
		t.Error("exit = 0, want non-zero: nothing was written")
	}
	if !strings.Contains(errb.String(), "3") || !strings.Contains(errb.String(), "12") {
		t.Errorf("stderr = %q, want it to say how many words there are and how many it needs", errb.String())
	}
	if len(fake.Requests()) != 0 {
		t.Error("a request was sent for a deck too small to reflect on")
	}
	if got, _ := st.UserModel(); got != "" {
		t.Errorf("a file was written anyway:\n%s", got)
	}
}

func TestReflectDegradesWithNoModel(t *testing.T) {
	d, _, st, _ := reflectRig(t, 14)
	d.getenv = func(string) string { return "" }

	var out, errb bytes.Buffer
	code := runReflect(t.Context(), d, options{}, &out, &errb)

	if code == 0 {
		t.Error("exit = 0, want non-zero")
	}
	if !strings.Contains(errb.String(), "no model configured") {
		t.Errorf("stderr = %q", errb.String())
	}
	if got, _ := st.UserModel(); got != "" {
		t.Error("a file was written with no model to write it from")
	}
}

// A nil deck is reachable under DEFINE_NO_CAPTURE, and refuses through the same
// sentence --forget and /history use.
func TestReflectRefusesWithNoDeck(t *testing.T) {
	d := testDeps(t)
	d.deck = nil

	var out, errb bytes.Buffer
	if code := runReflect(t.Context(), d, options{noCapture: true}, &out, &errb); code == 0 {
		t.Error("exit = 0, want non-zero")
	}
	if !strings.Contains(errb.String(), "DEFINE_NO_CAPTURE") {
		t.Errorf("stderr = %q, want noDeckMessage's wording", errb.String())
	}
}

// The mode is a mode from every entry point: combining it with a word is two
// commands on one line, and silently honouring one is how -raw came to mean two
// things in #2.
func TestReflectWithAWordIsAUsageError(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(t.Context(), []string{"--reflect", "sycophantic"}, testDeps(t),
		strings.NewReader(""), &out, &errb); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

// A model with NO claims left must not be written.
//
// Found by smoke-testing against the live seam: a run came back with an empty
// band and an empty domain, every claim was dropped, and --reflect wrote a file
// containing nothing but frontmatter and the corrections stub. That file LOOKS
// like an answer — it says "here is what we know about you" and knows nothing —
// which is the same failure D2's floor exists to prevent, arriving through a
// different door. The renderer already omits empty sections; nothing stopped the
// caller from writing the result.
func TestReflectWritesNothingWhenEveryClaimIsDropped(t *testing.T) {
	d, fake, st, _ := reflectRig(t, 14)
	// Every citation is a word the deck does not hold.
	fake.Script("", llmtest.Reply{Text: `{"level":{"band":"C2","rationale":"x","evidence_words":["luffing"]},` +
		`"domains":[{"name":"sailing","share":1,"evidence_words":["clew"],"directive":"y"}]}`})

	var out, errb bytes.Buffer
	code := runReflect(t.Context(), d, options{}, &out, &errb)

	if code == 0 {
		t.Error("exit = 0, want non-zero: nothing was learned")
	}
	if got, _ := st.UserModel(); got != "" {
		t.Errorf("a file was written with no claims in it:\n%s", got)
	}
	if !strings.Contains(errb.String(), "nothing") {
		t.Errorf("stderr = %q, want it to say nothing survived", errb.String())
	}
}

// The dropped-claim message names the words it REJECTED. "No evidence in the
// deck" is the same unactionable shape as a claim that names none, and this
// message is the only place a person sees why a section went missing.
func TestReflectNamesTheRejectedCitations(t *testing.T) {
	d, fake, _, _ := reflectRig(t, 14)
	fake.Script("", llmtest.Reply{Text: `{"level":{"band":"C1","rationale":"x","evidence_words":["certiorari"]},` +
		`"domains":[{"name":"sailing","share":0.2,"evidence_words":["luffing","clew"],"directive":"y"}]}`})

	var out, errb bytes.Buffer
	runReflect(t.Context(), d, options{}, &out, &errb)

	for _, want := range []string{"sailing", "luffing", "clew"} {
		if !strings.Contains(errb.String(), want) {
			t.Errorf("stderr = %q, want it to name the rejected citation %q", errb.String(), want)
		}
	}
}

// mustReflect fails the test when the run did not succeed.
//
// Both the idempotency and the corrections tests compared a file before and
// after — and a run that FAILS writes nothing, so "unchanged" and "suffix
// preserved" are satisfied by a file nobody touched. I caught exactly this in
// the live smoke test ("the second run had failed and written nothing") and did
// not sweep it back into the unit tests, which is the whole finding: a class
// found in one place is not fixed until it is looked for in the others.
func mustReflect(t *testing.T, d deps, out, errOut *bytes.Buffer) {
	t.Helper()
	if code := runReflect(t.Context(), d, options{}, out, errOut); code != 0 {
		t.Fatalf("runReflect failed (exit %d), so the comparison below would prove nothing: %s",
			code, errOut.String())
	}
}

// A diagnostic is structured output too: one line each, to a terminal.
//
// The file's table was the instance a finding named first; the class is
// untrusted text reaching ANY structured output. A band or domain name carrying
// a newline forges an extra line the reader cannot tell from a real one — and
// these messages exist precisely so a person can see WHY a section went missing,
// so a forged one is worse than none.
//
// Asserted by COUNTING lines, because the obvious assertions are both vacuous
// and I wrote both before noticing: "every line starts with define: " is
// satisfied when the injected text itself starts with "define: ", and
// `Contains(text+"\n")` misses when the forged line has the rest of the message
// appended after it. Injection adds LINES; that is the observable.
func TestDroppedClaimDiagnosticsCannotForgeALine(t *testing.T) {
	d, fake, _, _ := reflectRig(t, 14)
	// Both claims are dropped (their citations are not in the deck), so exactly
	// three messages are expected: the level, the domain, and nothing-survived.
	fake.Script("", llmtest.Reply{Text: `{"level":{"band":"C1\nFORGED-LEVEL","rationale":"r",` +
		`"evidence_words":["luffing"]},` +
		`"domains":[{"name":"law\nFORGED-DOMAIN","share":0.5,` +
		`"evidence_words":["clew"],"directive":"d"}]}`})

	var out, errb bytes.Buffer
	runReflect(t.Context(), d, options{}, &out, &errb)

	lines := strings.Split(strings.TrimRight(errb.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("stderr has %d lines, want 3 — model text started a line of its own:\n%s",
			len(lines), errb.String())
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "define: ") {
			t.Errorf("a diagnostic line does not start with the program name: %q", line)
		}
	}
	// Neutralised, not discarded: the message must still name what it rejected.
	if !strings.Contains(errb.String(), "FORGED-LEVEL") {
		t.Errorf("the rejected text was dropped rather than neutralised:\n%s", errb.String())
	}
}

// EVERY arm that renders untrusted text is exercised with text that tries to
// forge a line — the positive control the neutralisation never had.
//
// TestDroppedClaimDiagnosticsCannotForgeALine reaches two of checkEvidence's
// five arms, so three sanitisers were unpinned: deleting them left the whole
// suite green (BR-17). A neutralising call with no test that constructs the
// violation is a check that cannot fail, and this is the third finding in that
// family on this file.
//
// Driven through dropClaim.String rather than the whole run, because the point
// is the ONE formatter every arm now goes through: a new arm gets this coverage
// by construction, where the old inline strings each needed remembering.
func TestEveryDropDiagnosticNeutralisesItsSubject(t *testing.T) {
	const forged = "\nFORGED"

	for _, tc := range []struct {
		name string
		drop dropClaim
	}{
		{"level, no band or rationale", dropClaim{
			Kind: "level", Subject: "C1" + forged,
			Reason: "no band or no rationale — nothing a reader could check"}},
		{"level, evidence not in deck", dropClaim{
			Kind: "level", Subject: "C1" + forged,
			Reason: "cites", Cited: []string{"luffing" + forged}}},
		{"domain, no name or directive", dropClaim{
			Kind: "domain", Subject: "law" + forged,
			Reason: "no name or no directive — nothing authoring could act on"}},
		{"domain, share out of range", dropClaim{
			Kind: "domain", Subject: "law" + forged,
			Reason: "share out of range"}},
		{"domain, evidence not in deck", dropClaim{
			Kind: "domain", Subject: "law" + forged,
			Reason: "cites", Cited: []string{"clew" + forged}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.drop.String()

			if strings.Contains(got, "\n") {
				t.Errorf("the diagnostic spans lines, so the model can forge one: %q", got)
			}
			// Neutralised, not discarded: a message that hides what it rejected
			// cannot be acted on.
			if !strings.Contains(got, "FORGED") {
				t.Errorf("the rejected text was dropped rather than neutralised: %q", got)
			}
		})
	}
}

// The frontmatter's provenance line is the same class, and was the site the
// original sweep found AFTER the table (lessons.md: "a class found in one place
// is not fixed until you look for it in the others").
//
// It had no positive control either: sampleMeta().Model carries no injection, so
// deleting sanitiseMeta's body left the suite green (BR-17). A newline here
// breaks the YAML frontmatter it sits inside.
func TestModelNameCannotBreakTheFrontmatter(t *testing.T) {
	out := renderUserModel(
		learnerModel{Level: levelClaim{Band: "C1", Rationale: "r", EvidenceWords: []string{"w"}}},
		modelMeta{Model: "claude\ntype: forged"},
	)

	head, _, ok := strings.Cut(strings.TrimPrefix(out, "---\n"), "\n---\n")
	if !ok {
		t.Fatalf("no frontmatter block:\n%s", out)
	}
	// A LINE of its own is the violation, not the substring. The first version
	// asserted strings.Contains(head, "type: forged") and failed against a
	// correctly neutralised file, because the collapsed text still contains that
	// substring inline — "generated_by: define --reflect (claude type: forged)".
	// That is this file's own lesson: choose injection text that does not satisfy
	// your own assertion.
	for _, line := range strings.Split(head, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "type: forged") {
			t.Errorf("the model name forged a frontmatter key:\n%s", head)
		}
	}
	if got := strings.Count(head, "\n") + 1; got != 4 {
		t.Errorf("frontmatter has %d lines, want 4 — the model name started one of its own:\n%s", got, head)
	}
	if !strings.Contains(head, "forged") {
		t.Errorf("the model name was discarded rather than neutralised:\n%s", head)
	}
}

# Grade Before Reveal Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** In `define --play`, grade the word straight from the prompt — `y` advances with no reveal, `n` records the miss and shows the definition.

**Architecture:** The session's refusal to grade an unrevealed word is reversed, because form 2.1 is a RECALL test and the learner knows their own recall before they check. `n` then owes the loop two effects — record the miss, and play the pronunciation — so `Apply` returns `[]Outcome` instead of one. A new `Graded` flag keeps the session on the missed word so the definition can be read before advancing.

**Tech Stack:** Go 1.26, `cmd/define/play` (pure, no imports), `cmd/define` (the loop), `creack/pty` for the live check.

---

## Core concepts

This is one atomic change to one state machine. **No `Mx` tags** — it closes in a
single `sdlc close` (AGENTS.md §3: an `Mx` row commits to its own boundary, and
tagging one-shot work forces a redundant double-log).

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `Session.Graded` | `cmd/define/play/session.go` | new |
| `score` | `cmd/define/play/session.go` | new |
| `Apply` | `cmd/define/play/session.go` | modified |
| `draw` | `cmd/define/play_loop.go` | modified |

- **`Session.Graded`** — this question's verdict is already recorded; the next
  input advances rather than grading again.
  - **Relationships:** 1:1 with the current question, cleared by `advance`. It is
    independent of `Revealed`: a learner can reveal without grading (space) and
    now grade without revealing (`y`).
  - **DRY rationale:** First occurrence. It exists because `n`-before-reveal must
    NOT advance — the definition it just earned would scroll past unread.
  - **Future extensions:** #7's multiple choice reveals the correct option on a
    wrong answer, which is this same state.

- **`Apply`** — one input, the next state, and **every** effect the loop owes.
  - **Why `[]Outcome`:** one input can owe the loop more than one effect, and
    `n`-before-reveal is the first that does — it records a verdict AND plays the
    pronunciation. The alternative, a `Reveal bool` on `Outcome`, would make
    "reveal" expressible two ways; this state machine has already paid for one
    rule living in two places (#6 BR-5).
  - **Rejected alternative:** defer the record to the advance and emit only
    `OutcomeReveal`. That breaks the invariant `OutcomeRecord`'s own comment
    states — *"Recording as it happens is what makes Ctrl-C lossless by
    construction rather than by a flush"* — because a Ctrl-C between the `n` and
    the advance would drop the answer.
  - **Future extensions:** any form needing "record and also do X" is now
    expressible without touching the contract again.

- **`score`** — what a verdict does to the running tally, in ONE place.
  - **Why it must exist (PQ-1):** `advance` currently both scores and moves, and
    the miss-before-reveal branch moves nowhere — so it would have recorded the
    event and left `s.Wrong` at zero, and the end-of-session line would have read
    "3 right, 0 wrong" after three misses. Splitting the tally from the move is
    what lets a verdict be scored without advancing.
  - **Relationships:** called by `advance` and by the miss branch. `Skipped`
    scores nothing, so `advance(s, q, Skipped)` after a miss double-counts
    nothing.

- **`draw`** — renders the question in one of three states.
  - **Relationships:** reads `Revealed` and `Graded`; owns no state.

**Test surface.** `cmd/define/play` is PURE and takes two of `puretest`'s guards;
its tests use no fakes. `session_test.go` covers `Apply` directly.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `playSession` outcome loop | `cmd/define/play_loop.go` | modified | terminal + store |
| `TestPTYPlayGradeFirst` | `cmd/define/pty_conformance_test.go` | new | a real pty |

- **`playSession` outcome loop** — iterates the returned slice instead of
  switching on one value.
  - **Injected into:** nothing; it is the performer. The existing `switch` body
    is unchanged, wrapped in `for _, out := range outs`.

- **`TestPTYPlayGradeFirst`** — drives the new flow on a real terminal.
  - **Why:** #6 BR-45. `--play`'s only shipped defect was invisible to every
    in-process test and to a byte-capture smoke run; the operator found it. The
    harness (`startDefineInDir`, `bareNewlines`, `unstyled`) already exists.
  - **Cadence:** on demand with the rest of the conformance suite, and RUN at
    this issue's close — #6 found this suite had been red since #21 because two
    closes ran only `go test ./...`.

---

## Chunk 1: the state machine

### Task 1: `Apply` returns every effect it owes

**Files:**
- Modify: `cmd/define/play/session.go`
- Test: `cmd/define/play/session_test.go`

- [ ] **Step 1: Change the signature, mechanically, with no behaviour change**

Every existing `return s, Outcome{...}` becomes `return s, []Outcome{{...}}`.
`advance` keeps returning a single `Outcome`; its callers wrap it.

```go
// Apply is the state machine: one input, the next state, and EVERY effect the
// loop owes.
//
// A slice because one input can owe more than one: `n` on an unrevealed word
// records the miss AND plays the pronunciation. Encoding the second as a flag on
// the first would make "reveal" expressible two ways, and this machine has
// already paid once for a rule living in two places (#6 BR-5).
//
// Order is significant: the record is emitted FIRST, so a caller that performs
// them in order records before it can block on audio.
func Apply(s Session, in Input) (Session, []Outcome) {
```

- [ ] **Step 2: Update the NINE `Apply` call sites in `session_test.go`**

Most go through the `drive` helper (`session_test.go:11-19`), which collects one
`Outcome` per input and must now append all of them:

```go
func drive(s Session, ins ...Input) (Session, []Outcome) {
	var out []Outcome
	for _, in := range ins {
		var os []Outcome
		s, os = Apply(s, in)
		out = append(out, os...)
	}
	return s, out
}
```

`records(outs)` needs no change — it already filters a slice.

Run: `go test ./cmd/define/play/`
Expected: PASS — pure refactor, behaviour identical.

- [ ] **Step 3: Commit**

```bash
git add cmd/define/play/session.go cmd/define/play/session_test.go
git commit -m "#24: Apply returns every effect it owes, not one"
```

### Task 2: `y` on an unrevealed word records and advances

**Files:**
- Modify: `cmd/define/play/session.go:133-146` (the `InputRune` arm)
- Test: `cmd/define/play/session_test.go`

- [ ] **Step 1: Write the failing test**

```go
// A RECALL test is rated by the learner, not by the screen. The definition is
// FEEDBACK, not stimulus — which is why grading before a reveal is now the
// normal path and not a mis-keystroke.
//
// twoQuestions() is Recall, which grades y/n — the file's own idiom, and the
// right one here because this is about form 2.1's actual keys. fakeForm's digits
// belong to the form-AGNOSTIC tests, where sharing no key with Recall is the
// point.
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
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/play/ -run TestCorrectBeforeReveal`
Expected: FAIL — `outcomes = [{OutcomeNone ...}]`, because the `!s.Revealed`
guard returns `OutcomeNone`.

- [ ] **Step 2b: DELETE the test that asserts the old premise**

`TestGradingBeforeRevealIsIgnored` (`cmd/define/play/session_test.go:59-70`) is
the one test this issue reverses — its comment states the position being dropped:
*"A learner cannot rate what they have not seen."* It is replaced by the two
tests above, not merely edited, because its NAME is the claim.

This is the test the plan first failed to name (PQ-3): the ones it did name
(`TestUngradedKeyNeverReachesTheCapturer`, anything pressing Enter before `y`)
all stay green, because the peek path is unchanged.

- [ ] **Step 3: Replace the guard with the grade-then-branch**

```go
	case InputReveal:
		if s.Graded {
			// PQ-2: Enter and space reach here, not the InputRune arm — toInput
			// maps them to InputReveal (play_loop.go:174-190). Without this they
			// would be DEAD in the graded state, which is precisely the two keys
			// every existing user reaches for, while the prompt says "any key".
			next, out := advance(s, q, Skipped)
			return next, []Outcome{out}
		}
		if s.Revealed {
			return s, []Outcome{{Kind: OutcomeNone}}
		}
		s.Revealed = true
		return s, []Outcome{{Kind: OutcomeReveal}}

	case InputRune:
		if s.Graded {
			// Already answered; this keystroke is the learner moving on.
			next, out := advance(s, q, Skipped)
			return next, []Outcome{out}
		}
		verdict, ok := q.Grade(in.Rune)
		if !ok {
			return s, []Outcome{{Kind: OutcomeNone}} // a key this form does not use
		}
		if s.Revealed || verdict != Wrong {
			// Nothing left to show: either it is already on screen, or the
			// learner got it and does not need it.
			next, out := advance(s, q, verdict)
			return next, []Outcome{out}
		}
		// A MISS on a hidden word earns the definition. Stay on it — advancing
		// would scroll the answer past unread, which is the whole point of
		// showing it. Recorded NOW, in the same breath, so Ctrl-C before the
		// next keystroke still keeps this answer.
		s.Revealed, s.Graded = true, true
		s = score(s, verdict) // PQ-1: the tally, since we are not advancing
		return s, []Outcome{
			{Kind: OutcomeRecord, Word: q.Word(), Verdict: verdict},
			{Kind: OutcomeReveal},
		}
	}
```

`advance` delegates the tally and clears both flags:

```go
// score is what a verdict does to the tally, and the only place that decides it.
// Split from advance because a miss on a hidden word scores WITHOUT advancing.
func score(s Session, v Verdict) Session {
	switch v {
	case Correct:
		s.Right++
	case Wrong:
		s.Wrong++
	}
	return s
}

func advance(s Session, q Question, v Verdict) (Session, Outcome) {
	s = score(s, v)
	s.Index++
	s.Revealed, s.Graded = false, false
	// ... unchanged ...
}
```

- [ ] **Step 4: Run it**

Run: `go test ./cmd/define/play/ -run TestCorrectBeforeReveal`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/define/play/
git commit -m "#24: y grades a hidden word and advances"
```

### Task 3: `n` on an unrevealed word records the miss and reveals

**Files:**
- Test: `cmd/define/play/session_test.go`

- [ ] **Step 1: Write the failing test**

```go
// A miss earns the definition, and earns it WITHOUT advancing.
func TestWrongBeforeRevealRecordsAndReveals(t *testing.T) {
	next, outs := Apply(twoQuestions(), rune_('n'))

	if len(outs) != 2 {
		t.Fatalf("outcomes = %+v, want two: the record and the reveal", outs)
	}
	// ORDER matters: recorded before anything that can block.
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
	// the tally at zero, so a session of three misses ended "0 right, 0 wrong".
	if next.Wrong != 1 {
		t.Errorf("Wrong = %d, want 1 — a miss counts even though it does not advance", next.Wrong)
	}
}

// PQ-2: Enter and space are the keys a user already reaches for, and toInput
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

// The verdict is recorded ONCE. Answering again is the learner moving on, not a
// second assessment — Fold would read a duplicate as a second review.
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
}
```

- [ ] **Step 2: Run both**

Run: `go test ./cmd/define/play/ -run "BeforeReveal|AfterAMiss"`
Expected: PASS (Task 2's implementation already covers these)

- [ ] **Step 3: Verify the SKIP rule survived**

Run: `go test ./cmd/define/play/ -run Skip -v`
Expected: PASS. `advance(s, q, Skipped)` still emits no record — the skip filter
stays in `advance` and only in `advance`.

- [ ] **Step 4: Confirm peeking still works**

```go
// Space still reveals without grading, for a learner who wants to check before
// rating. This is the OLD flow, still available, just no longer mandatory.
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
```

Run: `go test ./cmd/define/play/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/define/play/
git commit -m "#24: a miss earns the definition without advancing"
```

### Task 4: dropping still works in every state

**Files:**
- Test: `cmd/define/play/session_test.go`

- [ ] **Step 1: Extend the drop test to the graded state**

```go
// d works before a reveal, after a peek, and after a miss — "this word is not
// mine" is true whatever is on screen, and a learner who just missed a word is
// exactly who wants to drop it.
func TestDropWorksInEveryState(t *testing.T) {
	for _, tc := range []struct {
		name string
		pre  []Input
	}{
		{"before anything", nil},
		{"after a peek", []Input{{Kind: InputReveal}}},
		{"after a miss", []Input{{Kind: InputRune, Rune: 'n'}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := drive(twoQuestions(), tc.pre...)
			next, outs := Apply(s, Input{Kind: InputDrop})
			if len(outs) != 1 || outs[0].Kind != OutcomeDrop || outs[0].Word != "obsequious" {
				t.Fatalf("outcomes = %+v, want one OutcomeDrop for \"obsequious\"", outs)
			}
			if next.Index != 1 {
				t.Errorf("Index = %d, want 1", next.Index)
			}
		})
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test ./cmd/define/play/ -run TestDropWorksInEveryState -v`
Expected: PASS all three subtests

- [ ] **Step 3: Run the purity guards**

Run: `go test ./cmd/define/play/ -run Purity`
Expected: PASS — no new imports.

- [ ] **Step 4: Commit**

```bash
git add cmd/define/play/
git commit -m "#24: d works before a reveal, after a peek, after a miss"
```

---

## Chunk 2: the loop, the prompt, the docs

### Task 5: the loop performs every outcome

**Files:**
- Modify: `cmd/define/play_loop.go:114-116` (the `Apply` call and `switch`)

- [ ] **Step 1: Iterate the slice**

```go
		var outs []play.Outcome
		s, outs = play.Apply(s, in)

		for _, out := range outs {
			switch out.Kind {
			// ... existing body verbatim ...
			}
		}
```

The `OutcomeReveal` arm calls `s.Current().Word()`. After a miss the session is
still on that word, so this stays correct — **verify it, do not assume**: the
arm must not run after `advance` has moved on.

- [ ] **Step 2: Build and run the existing loop tests**

Run: `go build ./... && go test ./cmd/define/ -run "Play|Session|Drop|Reveal"`
Expected: PASS, all of them. The loop tests drive the peek path (`"\ry"`), which
is unchanged, so none of them encode the reversed premise — that one lives in
`cmd/define/play/session_test.go` and is handled in Task 2 Step 2b (PQ-3).

- [ ] **Step 3: Commit**

```bash
git add cmd/define/play_loop.go cmd/define/play_loop_test.go
git commit -m "#24: the loop performs every outcome an input owes"
```

### Task 6: `y` plays no audio

**Files:**
- Test: `cmd/define/play_loop_test.go`

- [ ] **Step 1: Write the failing test using the EXISTING player fake**

`fakePlayer` already records what it played in `Played []string`, so a new
refusing double is not justified — #6 BR-43's own rule: *when adding a double
next to one that nearly fits, the comment says why the near-fit was rejected; if
it cannot, use the existing one.* It cannot here. `playRig` already installs one.

```go
// A correct answer costs one keystroke and plays nothing. The pronunciation is
// what a MISS earns; playing it on a hit is the step this issue removes.
func TestCorrectAnswerPlaysNoAudio(t *testing.T) {
	d, opt, st := playRig(t, "sycophantic")
	// Audio ENABLED, so this is about the FLOW and not about the flag —
	// playRig's own options set noAudio.
	opt.noAudio, opt.times = false, 1
	fp := &fakePlayer{}
	d.player = fp

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
```

- [ ] **Step 2: Run it**

Run: `go test ./cmd/define/ -run TestCorrectAnswerPlaysNoAudio`
Expected: PASS with the new flow (would FAIL on the old one, which required a
reveal — and a reveal plays audio).

- [ ] **Step 3: Measure that it bites**

```bash
# MUTANT: make y reveal before advancing, i.e. the old flow.
# In session.go's InputRune arm, change `verdict != Wrong` to `false` — i.e.
# every answer reveals first, which is the old flow.
go test ./cmd/define/ -run TestCorrectAnswerPlaysNoAudio
# Expected: FAIL "played audio for a word the learner got right"
git checkout -- cmd/define/play/session.go
```

- [ ] **Step 4: Commit**

```bash
git add cmd/define/play_loop_test.go
git commit -m "#24: a correct answer plays nothing"
```

### Task 7: three prompts for three states

**Files:**
- Modify: `cmd/define/play_loop.go:243-258` (`draw`)

- [ ] **Step 1: Write the failing test**

```go
func TestThePromptSaysWhatTheKeysDo(t *testing.T) {
	q := play.NewRecall("sycophantic", "a definition")
	for _, tc := range []struct{ name string; s play.Session; want, absent string }{
		{"unrevealed: the grading keys, straight away",
			play.Session{Questions: []play.Question{q}},
			"y = got it, n = missed it", "to reveal"},
		{"peeked: still grading",
			play.Session{Questions: []play.Question{q}, Revealed: true},
			"y = got it, n = missed it", "any key"},
		{"missed: the answer is up, move on",
			play.Session{Questions: []play.Question{q}, Revealed: true, Graded: true},
			"any key = next word", "y = got it"},
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
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define/ -run TestThePromptSaysWhatTheKeysDo`
Expected: FAIL on the first subtest — the unrevealed prompt says "to reveal".

- [ ] **Step 3: Rewrite `draw`'s tail**

```go
	fmt.Fprintf(w, "\n%s\n", q.Prompt())
	if s.Revealed {
		fmt.Fprintf(w, "\n%s\n", q.Reveal())
	}
	if s.Graded {
		// Already answered and the answer is on screen: the only thing left is
		// to read it and move on.
		fmt.Fprint(w, "\nany key = next word, d = remove from deck, Ctrl-C to stop\n")
		return
	}
	// The GRADING keys, whether or not the definition is showing. A learner who
	// wants to check first still can (space or Enter); they simply no longer
	// have to, which is the point of this issue.
	fmt.Fprint(w, "\ny = got it, n = missed it, d = remove from deck, Ctrl-C to stop\n")
```

- [ ] **Step 4: Run it**

Run: `go test ./cmd/define/ -run TestThePromptSaysWhatTheKeysDo -v`
Expected: PASS all three

- [ ] **Step 5: Commit**

```bash
git add cmd/define/play_loop.go cmd/define/play_loop_test.go
git commit -m "#24: the prompt offers grading first"
```

### Task 8: the live check

**Files:**
- Modify: `cmd/define/pty_conformance_test.go`

- [ ] **Step 1: Write the pty test**

```go
// The new flow on a real terminal: one keystroke for a hit, and a miss puts the
// definition on screen without moving on.
//
// #6 BR-45 is why this exists at all — --play's one shipped defect was invisible
// to every in-process test and to a byte-capture smoke run.
func TestPTYPlayGradeFirst(t *testing.T) {
	deck := t.TempDir()
	_, seed := startDefineInDir(t, deck, nil, "--no-audio", "sycophantic")
	if !strings.Contains(watch(seed).take(3*time.Second), "sikəˈfan(t)ik") {
		t.Fatal("the seeding lookup did not resolve")
	}
	seed.WriteString("\x04")

	_, f := startDefineInDir(t, deck, nil, "--play", "--no-audio")
	out := watch(f)
	first := out.take(3 * time.Second)
	if !strings.Contains(unstyled(first), "y = got it") {
		t.Fatalf("the grading keys were not offered up front:\n%q", first)
	}
	if strings.Contains(unstyled(first), "sikəˈfan(t)ik") {
		t.Errorf("the definition was showing before the learner answered:\n%q", first)
	}

	f.WriteString("n")
	missed := out.take(2 * time.Second)
	if !strings.Contains(unstyled(missed), "sikəˈfan(t)ik") {
		t.Errorf("n did not put the definition on screen:\n%q", missed)
	}
	if !strings.Contains(unstyled(missed), "any key = next word") {
		t.Errorf("the graded prompt did not appear:\n%q", missed)
	}

	// PQ-2, on a real terminal: space is what a user presses, and it travels as
	// InputReveal. This is the assertion that would have caught it dead.
	f.WriteString(" ")
	moved := out.take(2 * time.Second)
	if !strings.Contains(unstyled(moved), "0 right, 1 wrong") {
		t.Errorf("space did not move on from the missed word:\n%q", moved)
	}
	if bad := bareNewlines(first + missed + moved); bad != 0 {
		t.Errorf("%d bare newline(s) — the CRLF cascade is back", bad)
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test -tags conformance -run TestPTYPlayGradeFirst ./cmd/define/`
Expected: PASS. Needs a real pty — if it reports `no pty available: operation
not permitted`, the sandbox is blocking `pty.Start`; re-run outside it.

- [ ] **Step 3: Run the WHOLE conformance suite**

Run: `go test -tags conformance -run TestPTY ./cmd/define/`
Expected: PASS. #6 found this suite had been red since #21 because two closes ran
only `go test ./...`. Running it here is the point.

- [ ] **Step 4: Commit**

```bash
git add cmd/define/pty_conformance_test.go
git commit -m "#24: the new flow, checked on a real terminal"
```

### Task 9: the docs say what the keys do

**Files:**
- Modify: `README.md:41-58`
- Modify: `atlas/define.md:1284-1287`

- [ ] **Step 1: Grep for every site that describes the flow**

```bash
grep -rn "to reveal\|Enter or space\|cannot rate what they have not seen\|before reveal" \
  README.md atlas/ cmd/define/ workshop/issues/ workshop/plans/
```

The second and third patterns are what reach `atlas/define.md:1284` and the
session's own comments; the first alone misses them (PQ-3).

Fix every hit this returns. **Build the list from what the grep returns, not from
memory** — #6 BR-44 and BR-48 were both "the correction reached two artifacts of
three", and BR-48 found four of fourteen names typed from memory were wrong.

- [ ] **Step 2: Update the README sample and key table**

```
$ define --play
ephemeral

y = got it, n = missed it, d = remove from deck, Ctrl-C to stop
```

| key | does |
|---|---|
| `y` | you had it — next word, no definition shown |
| `n` | you missed it — the definition appears; any key moves on |
| space or Enter | see the definition before answering, if you want to check |
| `d` | remove this word from the deck — its history is kept |
| Ctrl-C | stop; everything you answered is already saved |

- [ ] **Step 3: Reverse the atlas paragraph at :1284**

It currently reads *"Grading before reveal is ignored, because a learner cannot
rate what they have not seen."* Replace with the reasoning that is actually true
of a recall test, and say what changed and why.

- [ ] **Step 4: Re-run the grep from Step 1**

Expected: no stale hits.

- [ ] **Step 5: Commit**

```bash
git add README.md atlas/define.md
git commit -m "#24: the docs describe the flow that exists"
```

### Task 10: close

- [ ] **Step 1: Full suite, both halves**

```bash
gofmt -l cmd/ && go vet ./... && go test ./...
go test -tags conformance -run TestPTY ./cmd/define/
```

- [ ] **Step 2: Smoke it on the real deck**

```bash
cd ~/workspace/brain/data/life/vocab && define --play
```

Answer one `y` (no definition, straight to the next word), one `n` (definition
appears, any key moves on), and one `d`. Confirm nothing drifts rightward.

- [ ] **Step 3: `sdlc close --issue 24 --verified '<evidence>'`**

Single boundary, no `Mx` — one `sdlc close`, and the mandatory fresh-eyes review
runs there.

---

## Risks

**A duplicate record on the missed word.** The `Graded` flag is the only thing
preventing the second `n` from grading again, and `Fold` would read a duplicate
as a second review and demote the word twice. Pinned by
`TestAKeyAfterAMissAdvancesWithoutRecordingAgain`.

**`OutcomeReveal` reading the wrong word.** The loop's reveal arm calls
`s.Current().Word()` on the session AFTER `Apply` returned. That is correct only
because a miss does not advance. If a future change makes any advancing input
also emit `OutcomeReveal`, the audio plays for the NEXT word. Task 5 Step 1 says
verify rather than assume; the honest fix if it ever changes is to carry the word
on the outcome, as `OutcomeRecord` and `OutcomeDrop` already do.

**Muscle memory.** Anyone used to pressing space first will now see the
definition without having answered — which still works, and still grades
afterwards. The prompt change is what tells them.

**The conformance suite is opt-in.** It decays silently; #21 left it red through
two closes. Task 8 Step 3 and Task 10 Step 1 both run it deliberately.

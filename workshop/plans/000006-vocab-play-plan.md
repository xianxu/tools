# `define --play` Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Sit down and review. `define --play` runs today's queue: show the word, recall it, reveal the definition, self-rate — recording one event per answer as it happens, with no key and no network.

**Architecture:** A pure `Question` interface with one implementation (form 2.1), a pure session state machine over keystrokes, and a thin loop that owns the terminal and the store. The schedule (`#5`) decides what to ask; this decides how to ask it and what to record.

**Tech Stack:** Go 1.26, the existing `schedule`, `store`, `Capturer`, `Dictionary` and raw-terminal seams in `cmd/define`.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `Question` | `cmd/define/play/question.go` | new |
| `Verdict` | `cmd/define/play/question.go` | new |
| `Recall` | `cmd/define/play/recall.go` | new |
| `Input` / `InputKind` | `cmd/define/play/session.go` | new |
| `Outcome` / `OutcomeKind` | `cmd/define/play/session.go` | new |
| `Session` | `cmd/define/play/session.go` | new |
| `Apply` | `cmd/define/play/session.go` | new |
| `puretest` guards | `cmd/define/puretest/puretest.go` | new |

**Its own package, for the reason `schedule` was: the purity boundary becomes checkable.** `play` imports NOTHING at all — not `store`, not `schedule`. The first draft said it imports both; it does not, because the session works in deck keys and rendered strings the caller supplies, which is the strongest form of the claim being made. `#5` proved the shape works and left guards to copy; this reuses them rather than re-arguing the case.

- **`Question`** — what a form must provide: `Prompt() string` (what the learner sees before answering), `Reveal() string` (what they see after), `Word() string` (the deck key the answer is recorded against), and `Grade(r rune) (Verdict, bool)` (interpret a keystroke; the bool is "this key meant something to me").

  **It takes a `rune`, not `Key`.** The first draft said `Key`, which cannot compile: `Key` is declared in `package main` (`cmd/define/key.go:29`) and a subpackage cannot name it. That is not a technicality to route around — it is the import direction telling the truth. `main` owns the terminal and its key decoding; `play` is pure and must not know that Ctrl-C is `0x03`. So the loop decodes, and hands `play` the two things a form can act on: a rune for graded keys, and the CONTROL intents as explicit method calls (`Reveal()`, `Quit()`) rather than as key constants.
  - **The Done-when's "adding a second form requires no change to the loop" is a property of THIS interface**, and the whole reason it exists before there is a second form. `#7`'s form 2.3 is the test of that claim and cannot be written here — so the plan's obligation is to make the interface answer every question the loop asks, not to guess `#7`'s internals. `Grade` returning a bool rather than the loop switching on keys is what keeps key semantics inside the form: 2.1's `y/n` and 2.3's `1/2/3/4` are the same shape to the loop.
  - **Future extensions:** a form needing multiple screens (a cloze with a hint) would add `Next() bool`; nothing in the loop's structure forbids it.

- **`Recall`** — form 2.1: show the word, reveal the definition, self-rate.
  - Holds the word and the rendered definition. **Rendering happens in the caller**, not here — `Render` needs `RenderOpts` and the dictionary, which are IO-shaped; `Recall` receives the finished string. That is what keeps this package pure and testable without a fake dictionary.

- **The Spec's bad-question keypress is DEFERRED, explicitly.** The Spec asks for a key that flags a question as bad, and calls it "groundwork for the generated forms, which need this feedback loop". Form 2.1's question is the learner's own deck word plus NOAD's definition — there is nothing generated to be bad, so the feature would ship with no way to exercise it and no consumer. It belongs with `#12`/`#13`, the first forms that author anything. Recorded here rather than silently dropped, and named in the issue's Log.

- **`Verdict`** — `Correct`, `Wrong`, or `Skipped`. Three, not two: a learner who skips has not got it wrong, and recording a skip as a miss would demote the word and corrupt the schedule.

- **`Session`** — where the learner is: the queue, the index, whether the current answer is revealed, and the tally.

- **`Apply(s Session, in Input) (Session, Outcome)`** — the state machine, the same shape as `#14`'s editor `Apply`, and for the same reason: input in, next state out, and the LOOP does the effects.

  `Input` is `play`'s own type — `struct{ Rune rune; Kind InputKind }` with `InputRune`, `InputReveal`, `InputQuit` — decoded by the loop from `main`'s `Key`. **Not `main.Key`**, which a subpackage cannot name (`cmd/define/key.go:29`), and the compile error is the import direction telling the truth: `main` owns the terminal and knows Ctrl-C is `0x03`; `play` is pure and must not. The loop translates once, at the boundary where it already decodes escape sequences. The `Outcome` says what the caller must do — record an event, play audio, redraw, quit — so the pure core never touches the store or the terminal.
  - **DRY rationale:** `#14`'s editor already proved this split. Copying it means the two interactive surfaces in this binary work the same way, and a reader who has understood one has understood both.

**Test surface.** Table tests over key sequences, no fakes. `#5`'s purity guards are EXTRACTED into `cmd/define/puretest` rather than copied (see M1 Task 2 Step 6), and `play` takes the import and wall-clock guards. It does NOT take the store-symbol guard: it names no store symbol at all, and a guard with an empty allowlist would fatal on finding nothing to check — correctly, since it would be asserting nothing.

`puretest` needs its OWN tests, and that is not bookkeeping: a guard nothing has ever seen fail is indistinguishable from a guard that cannot fail. `#5` verified its guards by mutating the tree in a scratch copy and deleting it, and this plan's first draft repeated that. The tests run each guard against committed known-bad packages under `puretest/testdata/`.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `runPlay` | `cmd/define/play_loop.go` | new | terminal + store |
| `Capturer.CaptureReview` | `cmd/define/capture.go` | modified | the event log |
| `--play` flag | `cmd/define/main.go` | modified | flag surface |

- **`runPlay`** — reads the deck and the log, folds progress, builds the queue, drives `Apply` over the key channel, and performs each `Outcome`.
  - **The daily budget is a flag, `-count`, defaulting to 20.** The first draft never said, and `schedule.Queue` requires one. Twenty because it is a sitting rather than a chore — long enough to be worth opening, short enough to finish — and a flag because that number is a guess about one learner's attention, exactly the kind of guess that should be changeable without a rebuild. `0` means "no budget" and returns nothing, matching `Queue`'s contract rather than inventing a second meaning for it.
  - Reuses `readKeys`, `enterRaw`/`(*rawSession).restore` (`rawterm.go:21,29` — the first draft called this `withRawTerminal`, which does not exist) and `playAnnounced` rather than growing new ones. The interrupt path is `#16`'s `interrupter`.
  - **Events are written AS THEY HAPPEN**, which the Spec calls out and which comes free from the append-only log: `CaptureReview` is called the instant a verdict is graded, before the next question is drawn. Ctrl-C then loses nothing by construction rather than by a flush.

- **`Capturer.CaptureReview(word string, correct bool, opt options)`** — the third verb on the seam that already owns the log.
  - A `bool` cannot represent `Skipped`, and it does not need to: **a skip never reaches it.**

    **ONE filter, in `Apply`.** The two previous drafts of this bullet named two different places — "the loop drops skips before recording" and "`Apply` emits no record outcome" — and never chose, which is how a rule ends up implemented in both places or neither. `Apply` is the right one: only the session knows a verdict, the loop just performs outcomes, and a loop that inspected verdicts would be re-deciding something the state machine already decided. So `Apply` emits `OutcomeRecord` for `Correct` and `Wrong` and `OutcomeNone` for `Skipped`; the loop calls `CaptureReview` whenever it sees `OutcomeRecord` and never looks at the verdict.

    The signature's inability to express a skip is then a FEATURE — the type refuses a call that would be wrong — rather than a gap.
  - **Why the seam and not `store.AppendEvent` directly:** `capture.go` states the rule — capture is the ONLY thing that records, and "a second appender beside it is how that stops being true without anyone noticing". `#16` added `CaptureAsk` for exactly this reason; this follows it.
  - **`decideCapture` must be consulted, not bypassed.** `-raw` and `DEFINE_NO_CAPTURE` mean "write nothing here", and a review is a write.

  - **`DEFINE_NO_CAPTURE` leaves `deck` NIL**, which the first draft's "a session still runs, recording nothing" ignored — verified at `main.go`'s `openStore`: the opt-out branch returns a `storeDeps` with no `deck` at all. With no deck there is no queue and nothing to review, so that Done-when row was unsatisfiable as written.

    **Guard on `deck == nil`, not on the flag.** The env var is only ONE cause: `openStore`'s `Getwd` failure path also returns a `storeDeps` with no deck, and keying the check on the flag would hand a nil `store.Store` to the queue builder and panic. The precondition is "there is no deck here", whatever produced it.

    **Decision: `--play` with no deck prints one line and exits 0** — "not recording in this directory, so there is no deck to review" — rather than pretending to run. Reviewing a deck it cannot read would be theatre, and inventing a read-only deck for the opt-out path would give that flag a second meaning. The Done-when row is rewritten to say that.

**The LLM seam is not an integration point here, and that is the Done-when.** Form 2.1 needs no model. The row asks that a session run with the seam *unavailable*, which for this issue means the loop never reaches for it at all — asserted with a client returning `llm.ErrUnavailable` wired into `deps`, so the assertion is "the loop degrades" rather than "config resolution fails".

---

## Chunk 1: M1 — the pure core

### Task 1: `Question`, `Verdict`, `Recall`

**Files:**
- Create: `cmd/define/play/question.go`, `cmd/define/play/recall.go`, and their tests

- [ ] **Step 1: Write the failing tests.** Strategy: a table over `Recall.Grade` — the keys that mean correct, wrong and skip, and that an unrelated key returns `false` so the loop can ignore it rather than the form inventing a meaning. Plus that `Prompt` shows the word and NOT the definition, which is the entire point of the form and the one thing a careless implementation gets wrong.

- [ ] **Step 2: Verify red. Step 3: Implement. Step 4: Verify green. Step 5: Commit.**

### Task 2: `Session` and `Apply`

**Files:**
- Create: `cmd/define/play/session.go`, `cmd/define/play/session_test.go`

- [ ] **Step 1: Write the failing tests.** Strategy: table over key SEQUENCES, asserting `(state, outcome)` pairs — reveal then grade advances and emits a record outcome; grading before reveal is ignored (a learner cannot rate what they have not seen); the last answer emits quit; Ctrl-C emits quit at any point; a skip emits a record outcome with `Skipped` and does NOT demote.

- [ ] **Step 2: The Done-when's second-form property, tested HERE rather than asserted.** A `fakeForm` in the test package implements `Question` with different keys entirely, and the same table drives it. If the loop's behaviour is a function of the interface, that passes; if any key semantics leaked into `Apply`, it fails. This is the only honest way to test "adding a form requires no loop change" before a second form exists.

- [ ] **Step 3: Verify red. Step 4: Implement. Step 5: Verify green.**

- [ ] **Step 6: EXTRACT `#5`'s purity guards into a shared, parameterised helper** rather than copying ~150 lines. `#7`, `#12` and `#13` each add a form package, so "copy two" becomes "copy five" — and the `storetest.Suite` precedent is exactly this shape: one conformance body, many callers. The helper takes `(importPath, allowedImports, allowedStoreSymbols)`; `schedule`'s tests become a two-line call to it, and `play`'s another. Doing it now while there are two callers is cheaper than after there are five, and it is the same ARCH-DRY argument `#5` made about the day-boundary helper.

- [ ] **Step 7: Mutation-check** that grading-before-reveal, the skip verdict and the record-before-advance ordering each redden a named test.

- [ ] **Step 8: Atlas** — `atlas/define.md` gains the play model. **Per milestone, not deferred**: `#21` and `#5` both scheduled atlas work at the end and both were refused at their first milestone close.

- [ ] **Step 9: `sdlc milestone-close --issue 6 --milestone M1`.**

## Chunk 2: M2 — the loop, the flag, the recording

### Task 3: `CaptureReview`

**Files:**
- Modify: `cmd/define/capture.go`, `cmd/define/capture_test.go`
- Modify: every `Capturer` implementation and test double the compiler names

- [ ] **Step 1: Write the failing tests, at the RIGHT layer.** For `CaptureReview`: a correct answer appends one `EventReviewed` with `Correct: true`, a wrong one with `false`, `DEFINE_NO_CAPTURE` records nothing, and the word is `store.Key`-normalised.

      The skip rule is NOT tested here — the first draft put it in this list, which is testing the wrong thing. `CaptureReview`'s `bool` cannot express a skip *because a skip never reaches it*; the loop drops it. That rule belongs to the session, and its test (Task 2 Step 1) is that `Apply` emits no record outcome for a skip. Asserting it against the capturer would pass trivially while proving nothing about the behaviour.

- [ ] **Step 2: Verify red. Step 3: Implement. Step 4: Verify green.**

### Task 4: `runPlay` and `--play`

**Files:**
- Create: `cmd/define/play_loop.go`, `cmd/define/play_loop_test.go`
- Modify: `cmd/define/main.go`

- [ ] **Step 1: Write the failing session test.** Drive a whole session through `runPlay` with a scripted key channel, a fake dictionary, `fakePlayer` (`player_fake_test.go` — `playAnnounced` shells out to `afplay(1)`, so a real player in a test is a real process), a `store.Mem` and a `FixedClock`, and assert one event per answer — the Done-when's first row.

- [ ] **Step 1b: Audio is on by default, and that needs a row.** The Spec makes playback default-on during review and the first draft discussed it only in Risks. Assert `fakePlayer` was asked to play the current word before reveal, and that `--no-audio` leaves it untouched.

- [ ] **Step 2: The interrupt row.** Script two answers then Ctrl-C, and assert BOTH events are on disk. Written as a separate test because "preserves already-recorded events" is a different claim from "records events".

- [ ] **Step 3: The LLM row.** Same session with `deps.newLLM` returning a client whose every call is `llm.ErrUnavailable`, asserting the session completes and records normally — the loop must never reach for it.

- [ ] **Step 4: An empty queue is not an error.** "Nothing due today" is the expected state most days; it prints a line and exits 0.

- [ ] **Step 5: Verify red. Step 6: Implement `runPlay`. Step 7: Verify green.**

- [ ] **Step 8: Wire `--play`** in `main.go`, reusing the raw-terminal path. Assert the flag reaches the loop through the same entry-path enumeration `#21` needed — one row per process entry that can start a session.

- [ ] **Step 9: Mutation-check** the record-on-interrupt path and the no-capture gate.

- [ ] **Step 9b: Atlas for M2's surface.** The M2 window adds a user-visible flag and a new loop, so the close gate's atlas guard fires — and this plan already records that `#21` and `#5` were both refused for exactly this deferral. `atlas/define.md` carries the Player seam and the render-cooked/play-raw narrative; extend it with the play loop, the `Input` translation at the boundary, and the record-as-you-go guarantee.

- [ ] **Step 10: README** — `--play` in the flag list and one line of what a session looks like. This is the first user-visible surface since `#20`, so it is the first thing in a while a reader could look for and not find.

- [ ] **Step 11: `sdlc close --issue 6 --verified '<evidence>'`.**

---

## Risks

- **The interface is a guess until `#7` lands.** Four methods is my best reading of what 2.3 will need, and the `fakeForm` test proves the LOOP is form-agnostic, not that the interface is sufficient. If `#7` has to change it, that is the plan working — better a signature change with one implementation than three.
- **A skip that records nothing is a deliberate hole in the data.** `#8` will not be able to distinguish "skipped" from "never reviewed". The alternative is a fourth event kind, which `#5`'s `Fold` would have to learn to ignore; recording nothing keeps the log meaning one thing. Worth revisiting if `#8` wants the distinction.
- **Audio during review is on by default** and every word plays before reveal. That is the Spec, but it is also the kind of thing that is delightful twice and irritating on the twentieth word — a real session is the only way to find out, and `--no-audio` is the escape hatch until then.

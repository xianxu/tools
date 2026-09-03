# Retire Form 2.1 Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete form 2.1, so the form set reads as one sentence — **2.3 tests you, the board triages you** — and give the board the drop it now owes, since it is about to be the only form some words ever see.

**Architecture:** Three separable changes behind one sentence. Selection stops being a box question answered before the lookup and becomes a *preference order* resolved after it, because "can this word have distractors" is only knowable once the entry is parsed. `play.Recall` is deleted along with the thirty-odd tests that used it as a convenient one-word `Question` — those want a double, and one already exists. And `Mark` gains a third value so a cell can be removed rather than rated, reusing the `OutcomeDrop` the loop already handles.

**Tech Stack:** Go, `cmd/define` (package `main`) and `cmd/define/play` (pure, import-allowlisted). No new dependencies.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Dropped` (a `Mark`) | `cmd/define/play/board.go` | new |
| `Board.Toggle` | `cmd/define/play/board.go` | modified |
| `Board.Dropped` | `cmd/define/play/board.go` | new |
| `Dropping` | `cmd/define/play/session.go` | new |
| `Recall` | `cmd/define/play/recall.go` | deleted |
| `optionsFor` | `cmd/define/optionpool.go` | new |
| `formFor` | `cmd/define/play_loop.go` | new |
| `packBoards` | `cmd/define/play_loop.go` | new |
| `boardsFor` | `cmd/define/play_loop.go` | deleted |

(Three rows corrected against the tree at close, because the table has to
describe what the diff DID: `Board.Mode` is unchanged — it returns a field whose
TYPE gained a value, which is not a change to the accessor; `Board.Mark` gains one
line arming the drop, with the answer read by a new sibling rather than by
widening `Mark`; and the entity that is genuinely new is the `Dropped` constant,
not the `Mark` type.)

- **`Dropped`** — a third `Mark`, beside `Yes` and `No`. `Unmarked` stays the zero value; `Dropped.Verdict()` is `Skipped`, so a drop records no review and the schedule never sees it.
  - **Relationships:** 1:1 with a cell. `Rest` (Enter's sweep) already skips anything not `Unmarked`, so a dropped cell is untouched by it for free.
  - **DRY rationale:** The mode already exists, is already drawn on the prompt row, and is already landed by both a key and a click. A third value costs no new gesture, no new key and no second input grammar.
  - **Future extensions:** A fourth mode would want a mode *list* rather than an `if` chain in `Toggle`; two values did not earn one and three is the point at which to look again.

- **`Dropping`** — the optional capability a form implements when its last act asked for a REMOVAL rather than a grade: `Dropped() (string, bool)`.
  - **Relationships:** asked by `Apply` after `Grade`/`Mark`, exactly as `missedAxis(q)` asks `Missed`.
  - **DRY rationale:** This is the codebase's own doctrine — "everything a session needs to know about a form that isn't on `Question` is an optional capability it ASKS for", and there are five already. A type switch on `*Board` in `Apply` is the thing `#6`'s Done-when forbids.
  - **Why not a `Verdict`:** `Verdict` is what an ANSWER meant, and a drop is not an answer. Adding a fourth verdict would put "remove this word" in front of `schedule.Fold`, which reads verdicts to move boxes.

- **`optionsFor`** — the pure half of today's `choiceFor`: `(word, Entry, pool, seed) → []play.Option`. `choiceFor` keeps building the `Choice` from a rendered string.
  - **DRY rationale:** Splits the DECISION from the CONSTRUCTION so selection can ask "could this be a 2.3?" without paying a `Render` for a word that turns out to be triaged.

- **`formFor`** — the preference order, as one function over one already-parsed entry.
  - **DRY rationale:** Selection is currently split between `boardsFor` (box) and an `if q := choiceFor(...)` in the render loop (capability). One rule in one place is the whole readability claim of this issue.

- **`packBoards`** — triage words → boards the terminal can draw whole, plus any it cannot.
  - **Relationships:** replaces `boardsFor`'s chunking half.
  - **DRY rationale:** reuses `boardFits`, so selection and draw keep answering through the one helper `#40` consolidated them into.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `todaysQuestions` | `cmd/define/play_loop.go` | modified | the dictionary, once per due word |
| `boardPalette` | `cmd/define/play_loop.go` | modified | the terminal's colours |

- **`todaysQuestions`** — one lookup and one parse per due word, then a pure decision over the result.
  - **Injected into:** nothing new; it already receives `deps`.
  - **ARCH-CONSTRAINTS:** lookups are UNCHANGED at one `DCSCopyTextDefinition` per due word — today singles are fetched in the render loop and board words in the board loop; after this one loop does both. What must not appear is a SECOND lookup for a word that changes hands, which is why the parsed entry is carried rather than re-fetched.
    **Both directions, because the first draft named only the saving (PQ-8):** saved is a `Render` on a young word that turns out to be triaged. **Newly paid** is a `Render` plus a click-region map entry on every MATURE board word — today's board loop does `Lookup` + `ParseEntry` + `targetCandidate` and no `Render` at all. Bounded by `opt.count` (20 by default) and pure string work, so it is small; it is written down because a cost table that lists only savings is an argument, not a measurement. **If the render turns out to matter, the fix is to render lazily at the point a `Choice` is built** — the split in Step 3 is what makes that possible without restructuring again.

- **`boardPalette`** — a third sequence for a dropped cell.
  - **Injected into:** `play.Palette`, which already takes finished escapes from `main`.

**Test surface.** `play` is mechanically guarded pure (import allowlist + wall-clock grep), so `Mark`, `Toggle`, `Dropping` and the board's layout are unit-tested with no terminal. `formFor`/`packBoards` are pure over an already-parsed `Entry`. The end-to-end claims — a drop reaching `store.Forget`, a young deck's sitting length — run through `playSession` against the store fake.

---

## Chunk 1: selection, deletion, and the drop

### Task 1: one rule decides the form, and it decides AFTER the lookup

**Files:**
- Modify: `cmd/define/optionpool.go` (split `optionsFor` out of `choiceFor`)
- Modify: `cmd/define/play_loop.go` (`formFor`, `packBoards`; delete `boardsFor`; rewrite `todaysQuestions`'s loop)
- Test: `cmd/define/play_loop_test.go`

- [x] **Step 1: Write the failing selection test**

The table IS the spec, and the third row is the change:

```go
// ONE RULE: the board unless a real test can be built (#42).
//
// | box >= 3 | the board            |
// | box <= 2 | 2.3, or the board    |
//
// The third row is what this issue is: a word that cannot be tested is TRIAGED
// rather than self-rated on a screen of its own.
func TestUntestableWordsReachABoardAtEveryBox(t *testing.T) {
	// A deck spanning the threshold, with a word whose entry cannot be asked
	// (`bases` — every sense a cross-reference) at a LOW box, and assert both
	// sides: it reaches a board, and a testable young word still reaches 2.3.
}
```

Drive box state through the event log the way the existing rig does; check `questionsFor`'s output by type.

- [x] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define -run TestUntestableWordsReachABoard -v`
Expected: FAIL — the word arrives as `*play.Recall`.

- [x] **Step 3: Split the decision from the construction**

```go
// optionsFor is whether this entry can be a 2.3 AT ALL, and with which options.
//
// Split from choiceFor so SELECTION can ask without paying a Render: a young word
// with no usable distractors is triaged, and rendering it first would be a
// wrapped, coloured, click-mapped string built for a form it will never take.
func optionsFor(word string, e Entry, pool []play.Candidate, seed uint64) []play.Option
```

`choiceFor` becomes `optionsFor` + `play.NewChoice`, so no caller loses a step.

- [x] **Step 4: Write the preference order down once**

```go
// formFor is THE selection rule, and it is one sentence: 2.3 tests you, the board
// triages you.
//
// It runs AFTER the lookup, which is the structural change. `boardsFor` ran on
// keys before anything was fetched, because the box was all it consulted — and
// "can this word have distractors" is not knowable until the entry is parsed.
// A word is TESTED when it is young AND the deck can build a real question;
// everything else is triaged.
```

Mature words skip `optionsFor` entirely — the box alone decides, and that keeps the pool work off words that will never use it.

**WHERE A TRIAGED YOUNG WORD LANDS IN THE QUEUE, decided rather than left to fall
out (PQ-6).** `boardsFor` records singles-first-then-boards as a deliberate choice:
*"retrieval gets the learner's freshest attention and the maintenance sweep comes
after"*. Under the new rule an untestable young word moves from the retrieval half
to the sweep tail, and may share a board with box ≥ 3 words.

**Both are accepted, and neither is an accident.** It is not a retrieval question —
nothing can be retrieved from it, which is why it is triaged — so putting it in
the retrieval half would buy the learner's freshest attention for a question that
does not use it. And a mixed board is the point of the form: sixteen words for
sixteen keystrokes is what makes the sweep cheap, and splitting young from mature
would mean two half-empty boards where one full one packs.

- [x] **Step 5: Pack the triage words into boards the terminal can draw**

```go
// packBoards splits triage words into the largest boards this terminal can draw
// WHOLE, and returns the ones it cannot draw at all.
//
// A CHUNK THAT DOES NOT FIT SHRINKS rather than going back to singles (#40 D15's
// widening). Shrinking is monotone: fewer words never need more rows, because
// `cols` is capped at four and a subset's longest word is no longer than the
// whole's.
//
// SO THE LEFTOVER CASE IS ALL-OR-NOTHING, and that is worth knowing rather than
// discovering: a one-word board's height does not depend on the word (the grid is
// one row, the chrome two), so either this terminal can draw a board or it cannot.
```

- [x] **Step 6: A LEFTOVER WORD TRIES 2.3 BEFORE IT IS SKIPPED — D15 survives**

**This is the step the plan-quality gate caught as a Critical (PQ-1), and the
mistake is worth naming: a rule true of the words this issue ADDS was applied to
the words that were already there.** "2.3 is unavailable for exactly the words
newly arriving here" is true of box ≤ 2 words with no distractors — and false of
mature words, which reach the board because their BOX sent them, not because no
test could be built. Many of them can still take a 2.3.

`boardFitsIn` refuses every board below `minWrapWidth` (20 columns) or ~5 rows. So
shrink-then-skip applied to all triage words would, on a 19-column terminal, skip
every due word on a mature deck — and `todaysQuestions` would then print *"N words
are due but none could be looked up"*, which is false and is the one message that
path exists to avoid. Today that same terminal runs a full sitting on singles
(`TestAShortTerminalGetsMeaningChoiceNotAClippedBoard`).

So the order is: **board, else 2.3, else skip.** A word is skipped only when it can
be neither drawn nor tested — which is the Done-when's actual wording, and the
plan had drifted off it.

The entry is already parsed and in hand, so asking `optionsFor` for the leftovers
costs nothing extra and happens only on a terminal too small for any board.

- [x] **Step 7: Pin BOTH leftover paths**

```go
// A terminal too short for any board sends triage words to 2.3 where one can be
// built — D15's fallback, unchanged for the words it always covered — and skips
// only the ones that can be neither drawn nor tested.
//
// TWO ROWS, because one of them is the regression: a mature deck on a narrow
// terminal must still get a full sitting, not a screen of skips.
func TestANarrowTerminalFallsBackToChoiceBeforeSkipping(t *testing.T) { /* … */ }

// The word is skipped with its cause named, and the sitting CONTINUES — like a
// word the dictionary cannot find, which is the shape this reuses rather than
// inventing. Assert the other words are still asked: a skip that ended the sitting
// would pass a test that only checked the message.
func TestAWordNeitherTestableNorDrawableIsSkippedWithItsCause(t *testing.T) { /* … */ }
```

- [x] **Step 8: Measure the sitting length (Done-when 5)**

```go
// The board PACKS, so retiring 2.1 must not add screens to a young deck.
// MEASURED, because "the board packs" is an argument and this is the number.
//
// THE BASELINE IS NAMED, not read from the tree (PQ-9): form 2.1 gave ONE SCREEN
// PER WORD, so that is what "no longer than it was" means, and it stays checkable
// after the form it describes is deleted. A test comparing against whatever the
// tree does today would assert nothing the day the tree changes.
func TestRetiringRecallDoesNotLengthenAYoungSitting(t *testing.T) {
	// For decks of 1..5, count QUESTIONS (screens), not words, and assert
	// len(qs) <= len(deck) — one screen per word being what 2.1 cost.
}
```

- [x] **Step 8a: RE-POINT `boardsFor`'s PINS, do not just delete them (PQ-5)**

`boardsFor` is called from five places across three of `#40`'s Done-when tests, and
deleting the function silently deletes what they proved:

| test | what it pins | goes to |
|---|---|---|
| `TestTheBoxPicksTheForm` | the box threshold | `formFor` (Step 1's new test covers this half) |
| `TestBoardsArePackedToTheLabelAlphabet` | sixteen words per board | **`packBoards`** — must be re-pointed, not dropped |
| `TestAShortTerminalGetsMeaningChoiceNotAClippedBoard` | `#40` D15's fallback | **`packBoards` + the Step 6 order** — this is the test PQ-1's regression would have broken |

The third is the one to write FIRST, because it is the existing pin on the
behaviour the Critical was about.

- [x] **Step 9: Run, then commit**

```bash
go test ./cmd/define/... && go vet ./...
git commit -m "#42: one rule picks the form, and it picks after the lookup"
```

---

### Task 2: `play.Recall` is deleted, and the tests that used it take a double

**Files:**
- Delete: `cmd/define/play/recall.go`, `cmd/define/play/recall_test.go`
- Modify: `cmd/define/play/session_test.go`, `cmd/define/play/missed_test.go`, `cmd/define/play_loop_test.go` (plus `typeName`), `cmd/define/doc_sync_test.go`
- Modify (PRODUCTION COMMENTS naming Recall, found by the tree-wide grep and in no other task's list): `cmd/define/play/choice.go` (cites `recall.go:19` by file:line), `cmd/define/play/board.go`, `cmd/define/play_loop.go`, `cmd/define/optionpool.go`

**No per-file counts here, deliberately (PQ-7).** The first draft carried them and
they were already wrong in three of four places. The compiler enumerates the test
uses; `git grep -n 'Recall'` over the TREE enumerates the comments. A hand-copied
count is a third owner of a fact two tools already own.

**TWO REFERENCES THE COMPILER WILL NOT CATCH**, so they are named individually —
this is the exception the rule above needs, not a lapse from it:

- `cmd/define/play/purity_test.go:56` — `[]string{"Board", "Choice", "Recall"}`, a
  STRING literal. The purity guard iterates form names; a deleted form leaves it
  scanning for a type that no longer exists, which passes silently forever.
- `cmd/define/optionpool_test.go:362` — a doc comment ending *"on a small deck lose
  the form entirely to Recall"*, which after this issue is "to the board".

Both go in the tree-wide grep of Task 4 Step 3, and neither is found by
`go build`.
- Test: the suites above

- [x] **Step 1: Delete the form, then follow the compiler**

`go build ./...` first: production has exactly one reference (`todaysQuestions`), and Task 1 already removed it. Everything else is tests.

- [x] **Step 2: In `play`'s own tests, use the double that already exists**

`fakeForm` (session_test.go) is the right substitute and its comment already says why: it "uses digits and produces every verdict, sharing no key with Recall". Where a test types `y`/`n` it must move to `1`/`2` — and if that reads oddly, that is the test telling you it was asserting the FORM's key semantics inside a session test, which is what `TestSessionIsFormAgnostic` exists to forbid.

**A test that genuinely needs a SELF-RATED form takes one**, not `fakeForm`: check each use against `SelfRated` before substituting, because the ladder's two-rung promotion turns on it.

- [x] **Step 3: In `main`'s tests, add a local double**

`play`'s `fakeForm` is unexported and in another package. A `fakeQuestion` in `play_loop_test.go` is the right answer — a test double belongs to the test — and `typeName` loses its `*play.Recall` arm.

- [x] **Step 4: `doc_sync_test.go` drops a row**

The forms slice becomes 2.3 and the board. Its own comment names the residual honestly ("a THIRD form added to play and not added to this slice is not checked here") and that stays true with two.

- [x] **Step 5: Run, then commit**

```bash
go test ./... && go vet ./...
git commit -m "#42: delete form 2.1; the tests that borrowed it take a double"
```

---

### Task 3: the board learns to drop

**Files:**
- Modify: `cmd/define/play/board.go` (`Mark`, `Toggle`, `Mode`, `Prompt`, `Palette`)
- Modify: `cmd/define/play/session.go` (`Dropping`, `Apply`)
- Modify: `cmd/define/play_loop.go` (`boardPalette`, the dropped line)
- Test: `cmd/define/play/board_test.go`, `session_test.go`, `cmd/define/play_loop_test.go`

- [x] **Step 1: Write the failing end-to-end test**

END TO END, because the form is not what removes anything:

```go
// A DROP ON A BOARD REACHES store.Forget (#42).
//
// Driven through playSession rather than by calling Mark, because the wiring is
// the claim: a board that records a drop the loop never acts on leaves every
// `play` test green while the word stays in the deck.
func TestDroppingAWordOnABoardRemovesItFromTheDeck(t *testing.T) {
	// Tab twice into drop mode, press a cell's key, assert the deck no longer
	// holds that word AND that the OTHER cells are untouched.
}
```

And the same act by CLICK, since a key and a click are one act reached two ways.

- [x] **Step 2: Add the third mark**

```go
// Dropped is a cell REMOVED rather than rated.
//
// Its Verdict is Skipped, which is what keeps it out of the schedule: a drop is
// not an assessment, and recording one as a miss would demote a word on its way
// out of the deck. `Rest` already skips anything not Unmarked, so Enter's sweep
// leaves a dropped cell alone for free.
```

`Toggle` cycles three. Keep the cycle written as a `switch` on the current value rather than arithmetic on the iota — the order is a UX decision (`Yes → No → Dropped`, so the destructive mode is never one Tab from the default) and arithmetic hides it.

- [x] **Step 3: The capability, not a type switch**

```go
// Dropping is implemented by a form that can ask for a word to be REMOVED rather
// than graded. Optional, exactly as Missed is: most forms have no such gesture,
// and widening Question would make every one of them answer a question it cannot.
type Dropping interface {
	Dropped() (string, bool)
}
```

`Apply` asks it **on the successful `Grade`/`Mark` path only**, and emits `Outcome{Kind: OutcomeDrop, Word: …}` — the kind the loop **already** handles, so no new outcome and no new loop branch.

**WHERE it is asked is the whole of its correctness.** Asked from the outer `Apply`
instead, a board whose last mark was a drop would re-emit `OutcomeDrop` on the next
Tab or on a refused click — performing one act twice, against a word already gone,
and breaking the "no outcome is performed twice" obligation the loop enumerates. So
either it is consulted only where a mark just landed, or `Dropped()` is one-shot.
**Pin it:** Tab, then a refused click, after a drop — exactly one `OutcomeDrop`.

- [x] **Step 4: Say what mode is live, in all three states — INSIDE THE WIDTH BUDGET**

`Keys()` owns the mode line (R11) and is pinned by the README. Three states, and the destructive one must be unmistakable — this is the row a short window keeps longest, and the only statement of what the next click will mean.

**THE ROW HAS TWO COLUMNS OF HEADROOM, and the width is not cosmetic (PQ-3).**
`gradePrompt(board)` is `Keys()` + `", "` + `quitKey` = 62 + 2 + 14 = **78 at 80
columns**. `boardFitsIn` charges `displayRows(gradePrompt(q), termCols)` into
`fitsABoard`, so a row that wraps at 80 costs every board a row of terminal height
— on exactly the path this issue routes untestable young words onto, where the
fallback is now narrower. Naively adding `drop` costs 5 columns and wraps.

`TestTheRefusalRowIsNoWiderThanTheKeysRow` will NOT catch this: it compares the
refusal against the keys row, and both grow together.

So the row is re-cut to fit, e.g.:

```
marking [yes] no drop, Tab cycles, click or key, Enter ends
```

— 59 columns, 75 with the reserved keys, and `"click or key marks"` loses `marks`
because with three modes a click no longer only marks. **Two new pins:** all three
spellings are the same visible width (the existing invariant, extended), and
`gradePrompt` of a board fits 80 columns in one row.

- [x] **Step 4a: while here — the same doc block is stale about `d`**

`Keys()`'s comment says *"The label set is NOT enumerated. It has a hole at `d`"*.
`#40` removed that hole; the sentence describes the alphabet before its own final
round. Same family as `Grade`'s comment in Step 7.

- [x] **Step 5: DO NOT add a `dropped:` line — the loop already writes one**

The first draft of this plan wanted `relearnLine`'s sibling. It would have been a
SECOND OWNER (PQ-4): `play_loop.go` already writes `removed %q from the deck` into
the buffer on every `OutcomeDrop`, and that line is already in the right place —
it lands as the drop happens, which is where `relearnLine`'s own reasoning
("written as the board closes, so the line sits in the transcript where the board
was") wants it.

`relearn:` is batched because a board's `no` marks are only knowable once it
closes; a drop is knowable immediately and is already reported. Nothing to add,
and the step survives as the reason not to.

**Verify rather than assume**: pin that a board drop produces exactly ONE
transcript line.

- [x] **Step 6: A third colour**

Through `boardPalette` from `main`, on the same seam. Distinct from green/red at a glance.

- [x] **Step 7: Fix the doc comment that is about this key**

`Board.Grade` says *"`d` and `D` never arrive: toInput takes them first, and boardLabels has no cell for them either way."* Both halves are false since `#40` put `d` in the alphabet. Correct it here, because this is the issue about that key.

- [x] **Step 8: Run, mutation-check, commit**

Revert each of: the `Dropped` verdict mapping, `Apply`'s `Dropping` call, the palette entry. Each must redden a named test — see `workshop/lessons.md`, "A pin that cannot fail is not a pin".

```bash
git commit -m "#42: the board drops as well as triages"
```

---

### Task 4: the docs follow

**Files:**
- Modify: `cmd/define/optionpool.go` (`fallbackReasons`'s prose), `cmd/define/README.md`, `atlas/define.md`
- Modify: `cmd/define/schedule`/`play` doc comments naming 2.1 as the fallback that always works

- [x] **Step 1: `fallbackReasons` is re-aimed, not re-listed**

The three reasons are unchanged; what they SELECT changes. It stops being "why a word goes to form 2.1" and becomes "why a word is triaged rather than tested". `TestREADMENamesEveryFallbackReason` derives from the list, so the README follows or the build fails.

- [x] **Step 2: README**

The "Recall, on a young deck" section goes. The board section absorbs it: **one form, two doors** — a settled word, or a word no test can be built for. The key table loses `y`/`n` and gains the drop mode. The board block is DERIVED (`#40` BR-18) — regenerate, don't hand-edit.

- [x] **Step 3: Atlas**

`atlas/define.md` names form 2.1 at ~10 sites (`grep -n 'form 2\.1\|Recall'`). Sweep the list, not the first hit — and per `workshop/lessons.md` ("A retraction is not done until `git grep` over the TREE is clean") run the grep over the tree, not `cmd/`.

Three passages are history rather than deletions and should be kept as such: the `#24` grading-before-reveal argument (it explains a mechanism 2.3 still uses), the `SelfRated` paragraph (the board is now its only implementor), and reason 3's "form 2.1 shows the whole rendered entry, so a redirect is harmless under it" — which is the argument this issue's trade-off ACCEPTS, so deleting it deletes the reason the trade-off is a trade-off.

- [x] **Step 4: Run the doc pins and commit**

```bash
go test ./cmd/define -run 'README|Atlas|Doc' -v
git commit -m "#42: docs — 2.3 tests you, the board triages you"
```

---

## Verification

- [x] `go test ./...` green.
- [x] `go test -tags conformance ./cmd/define` green — the pty suite drives the board's clicks, and this issue changes what a click can mean.
- [x] `go vet ./...` and `gofmt -l` clean.
- [x] **A real sitting**, on a deck small enough to have no distractors: confirm the untestable words arrive on a board, that the drop mode is unmistakable, and that dropping removes the word.
- [x] Every Done-when row ticked with the mutation that proved it.


## Revisions

### 2026-09-02 — three deltas found while implementing

**1. No `dropped:` transcript line.** Task 3 Step 5 planned `relearnLine`'s
sibling; the plan-quality gate (PQ-4) showed the loop already writes
`removed %q from the deck` per drop, so the sibling would have been a second owner
of one fact. The step survives as the reason NOT to, and
`TestADropOnABoardIsReportedOnce` pins that there is exactly one.

**2. The Core-concepts table named the wrong entities**, corrected against the
tree at close and caught by `TestPlanTableStatusMatchesTheChangeWindow`:
`Board.Mode` is unchanged (it returns a field whose TYPE gained a value),
`Board.Mark` gains one line arming the drop rather than being widened, and the
genuinely new entity is the `Dropped` constant rather than the `Mark` type. A
table row has to describe what the diff did, not what the design felt like.

**3. `gradeKey` had to become what its doc already claimed.** Not in any task's
file list, and unavoidable: its comment promised "the keystroke that grades q with
the wanted verdict, WHICHEVER form q is" while ending in form 2.1's hardcoded
`y`/`n`. Deleting that form left it silently answering nothing, and about a dozen
tests stopped grading while still passing their earlier assertions. It probes the
form now, with an explicit arm for a grid (whose key means whatever the MODE is,
and whose marks are irreversible so probing would spend one).

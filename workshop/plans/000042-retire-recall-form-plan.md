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
| `Mark` | `cmd/define/play/board.go` | modified |
| `Board.Mode` / `Board.Toggle` | `cmd/define/play/board.go` | modified |
| `Dropping` | `cmd/define/play/session.go` | new |
| `Recall` | `cmd/define/play/recall.go` | deleted |
| `optionsFor` | `cmd/define/optionpool.go` | new |
| `formFor` | `cmd/define/play_loop.go` | new |
| `packBoards` | `cmd/define/play_loop.go` | new |
| `boardsFor` | `cmd/define/play_loop.go` | deleted |

- **`Mark`** — gains `Dropped` beside `Yes` and `No`. `Unmarked` stays the zero value; `Dropped.Verdict()` is `Skipped`, so a drop records no review and the schedule never sees it.
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
  - **ARCH-CONSTRAINTS:** cost is UNCHANGED at one `DCSCopyTextDefinition` per due word. Today singles are looked up in the render loop and board words in the board loop; after this there is one loop that does both. What is saved is a `Render` on a young word that turns out to be triaged; what must not appear is a second lookup for a word that changes hands, which is why the parsed entry is carried rather than re-fetched.

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

- [ ] **Step 1: Write the failing selection test**

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

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define -run TestUntestableWordsReachABoard -v`
Expected: FAIL — the word arrives as `*play.Recall`.

- [ ] **Step 3: Split the decision from the construction**

```go
// optionsFor is whether this entry can be a 2.3 AT ALL, and with which options.
//
// Split from choiceFor so SELECTION can ask without paying a Render: a young word
// with no usable distractors is triaged, and rendering it first would be a
// wrapped, coloured, click-mapped string built for a form it will never take.
func optionsFor(word string, e Entry, pool []play.Candidate, seed uint64) []play.Option
```

`choiceFor` becomes `optionsFor` + `play.NewChoice`, so no caller loses a step.

- [ ] **Step 4: Write the preference order down once**

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

- [ ] **Step 5: Pack the triage words into boards the terminal can draw**

```go
// packBoards splits triage words into the largest boards this terminal can draw
// WHOLE, and returns the ones it cannot draw at all.
//
// A CHUNK THAT DOES NOT FIT SHRINKS rather than going back to singles (#40 D15's
// widening). D15 sent it to form 2.3 — the form that is unavailable for exactly
// the words newly arriving here. Shrinking is monotone: fewer words never need
// more rows, because `cols` is capped at four and a subset's longest word is no
// longer than the whole's.
//
// SO THE LEFTOVER CASE IS ALL-OR-NOTHING, and that is worth knowing rather than
// discovering: a one-word board's height does not depend on the word (the grid is
// one row, the chrome two), so either this terminal can draw a board or it cannot.
```

- [ ] **Step 6: A word that can be neither tested nor drawn is skipped, by cause**

Reuse the existing shape — `define: skipping %q: %v` already exists for a word the dictionary cannot find, and this is the same event with a different cause. Name BOTH conditions, because both must hold.

- [ ] **Step 7: Pin the leftover path**

```go
// A terminal too short for even a ONE-WORD board. The word is skipped for the
// sitting with its cause named, and the sitting continues — like a word the
// dictionary cannot find, which is the shape this reuses rather than inventing.
func TestAWordNeitherTestableNorDrawableIsSkippedWithItsCause(t *testing.T) { /* … */ }
```

Assert the sitting CONTINUES (the other words are still asked) — a skip that ended the sitting would pass a test that only checked the message.

- [ ] **Step 8: Measure the sitting length (Done-when 5)**

```go
// The board PACKS, so retiring 2.1 must not add screens to a young deck.
// MEASURED, because "the board packs" is an argument and this is the number.
func TestRetiringRecallDoesNotLengthenAYoungSitting(t *testing.T) {
	// questions (screens), not words, for decks of 1..5 — strictly fewer or equal
}
```

- [ ] **Step 9: Run, then commit**

```bash
go test ./cmd/define/... && go vet ./...
git commit -m "#42: one rule picks the form, and it picks after the lookup"
```

---

### Task 2: `play.Recall` is deleted, and the tests that used it take a double

**Files:**
- Delete: `cmd/define/play/recall.go`, `cmd/define/play/recall_test.go`
- Modify: `cmd/define/play/session_test.go` (16 uses), `cmd/define/play/missed_test.go` (2), `cmd/define/play_loop_test.go` (8 + `typeName`), `cmd/define/doc_sync_test.go` (1)
- Test: the suites above

- [ ] **Step 1: Delete the form, then follow the compiler**

`go build ./...` first: production has exactly one reference (`todaysQuestions`), and Task 1 already removed it. Everything else is tests.

- [ ] **Step 2: In `play`'s own tests, use the double that already exists**

`fakeForm` (session_test.go) is the right substitute and its comment already says why: it "uses digits and produces every verdict, sharing no key with Recall". Where a test types `y`/`n` it must move to `1`/`2` — and if that reads oddly, that is the test telling you it was asserting the FORM's key semantics inside a session test, which is what `TestSessionIsFormAgnostic` exists to forbid.

**A test that genuinely needs a SELF-RATED form takes one**, not `fakeForm`: check each use against `SelfRated` before substituting, because the ladder's two-rung promotion turns on it.

- [ ] **Step 3: In `main`'s tests, add a local double**

`play`'s `fakeForm` is unexported and in another package. A `fakeQuestion` in `play_loop_test.go` is the right answer — a test double belongs to the test — and `typeName` loses its `*play.Recall` arm.

- [ ] **Step 4: `doc_sync_test.go` drops a row**

The forms slice becomes 2.3 and the board. Its own comment names the residual honestly ("a THIRD form added to play and not added to this slice is not checked here") and that stays true with two.

- [ ] **Step 5: Run, then commit**

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

- [ ] **Step 1: Write the failing end-to-end test**

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

- [ ] **Step 2: Add the third mark**

```go
// Dropped is a cell REMOVED rather than rated.
//
// Its Verdict is Skipped, which is what keeps it out of the schedule: a drop is
// not an assessment, and recording one as a miss would demote a word on its way
// out of the deck. `Rest` already skips anything not Unmarked, so Enter's sweep
// leaves a dropped cell alone for free.
```

`Toggle` cycles three. Keep the cycle written as a `switch` on the current value rather than arithmetic on the iota — the order is a UX decision (`Yes → No → Dropped`, so the destructive mode is never one Tab from the default) and arithmetic hides it.

- [ ] **Step 3: The capability, not a type switch**

```go
// Dropping is implemented by a form that can ask for a word to be REMOVED rather
// than graded. Optional, exactly as Missed is: most forms have no such gesture,
// and widening Question would make every one of them answer a question it cannot.
type Dropping interface {
	Dropped() (string, bool)
}
```

`Apply` asks it after the mark lands and emits `Outcome{Kind: OutcomeDrop, Word: …}` — the kind the loop **already** handles, so no new outcome and no new loop branch.

- [ ] **Step 4: Say what mode is live, in all three states**

`Keys()` owns the mode line (R11) and is pinned by the README. Three states, and the destructive one must be unmistakable — this is the row a short window keeps longest, and it is the only statement of what the next click will mean.

- [ ] **Step 5: A board names what it dropped as it closes**

`relearnLine`'s sibling. The board is the live edge and vanishes whole, so what it did has to reach the transcript or it did not visibly happen — and a dropped word is recoverable by looking it up again (`Forget` keeps events), which the line is what makes discoverable.

- [ ] **Step 6: A third colour**

Through `boardPalette` from `main`, on the same seam. Distinct from green/red at a glance.

- [ ] **Step 7: Fix the doc comment that is about this key**

`Board.Grade` says *"`d` and `D` never arrive: toInput takes them first, and boardLabels has no cell for them either way."* Both halves are false since `#40` put `d` in the alphabet. Correct it here, because this is the issue about that key.

- [ ] **Step 8: Run, mutation-check, commit**

Revert each of: the `Dropped` verdict mapping, `Apply`'s `Dropping` call, the palette entry. Each must redden a named test — see `workshop/lessons.md`, "A pin that cannot fail is not a pin".

```bash
git commit -m "#42: the board drops as well as triages"
```

---

### Task 4: the docs follow

**Files:**
- Modify: `cmd/define/optionpool.go` (`fallbackReasons`'s prose), `cmd/define/README.md`, `atlas/define.md`
- Modify: `cmd/define/schedule`/`play` doc comments naming 2.1 as the fallback that always works

- [ ] **Step 1: `fallbackReasons` is re-aimed, not re-listed**

The three reasons are unchanged; what they SELECT changes. It stops being "why a word goes to form 2.1" and becomes "why a word is triaged rather than tested". `TestREADMENamesEveryFallbackReason` derives from the list, so the README follows or the build fails.

- [ ] **Step 2: README**

The "Recall, on a young deck" section goes. The board section absorbs it: **one form, two doors** — a settled word, or a word no test can be built for. The key table loses `y`/`n` and gains the drop mode. The board block is DERIVED (`#40` BR-18) — regenerate, don't hand-edit.

- [ ] **Step 3: Atlas**

`atlas/define.md` names form 2.1 at ~10 sites (`grep -n 'form 2\.1\|Recall'`). Sweep the list, not the first hit — and per `workshop/lessons.md` ("A retraction is not done until `git grep` over the TREE is clean") run the grep over the tree, not `cmd/`.

Three passages are history rather than deletions and should be kept as such: the `#24` grading-before-reveal argument (it explains a mechanism 2.3 still uses), the `SelfRated` paragraph (the board is now its only implementor), and reason 3's "form 2.1 shows the whole rendered entry, so a redirect is harmless under it" — which is the argument this issue's trade-off ACCEPTS, so deleting it deletes the reason the trade-off is a trade-off.

- [ ] **Step 4: Run the doc pins and commit**

```bash
go test ./cmd/define -run 'README|Atlas|Doc' -v
git commit -m "#42: docs — 2.3 tests you, the board triages you"
```

---

## Verification

- [ ] `go test ./...` green.
- [ ] `go test -tags conformance ./cmd/define` green — the pty suite drives the board's clicks, and this issue changes what a click can mean.
- [ ] `go vet ./...` and `gofmt -l` clean.
- [ ] **A real sitting**, on a deck small enough to have no distractors: confirm the untestable words arrive on a board, that the drop mode is unmistakable, and that dropping removes the word.
- [ ] Every Done-when row ticked with the mutation that proved it.

# Scheduling Engine Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decide what is worth the learner's attention today — Leitner boxes, fixed intervals, entirely pure, and derived from the event log rather than from counters kept beside it.

**Architecture:** One package of pure functions over `[]store.ReviewEvent` and `[]store.Word`. No IO, no clock of its own, no new store fields. `#6` supplies the deck, the events and the time; this decides the queue and how a box moves.

**Tech Stack:** Go 1.26, `cmd/define/store` types only.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `Box` | `cmd/define/schedule/box.go` | new |
| `Progress` | `cmd/define/schedule/progress.go` | new |
| `Fold` | `cmd/define/schedule/progress.go` | new |
| `Due` | `cmd/define/schedule/progress.go` | new |
| `Answer` | `cmd/define/schedule/progress.go` | new |
| `Mastered` | `cmd/define/schedule/progress.go` | new |
| `Queue` | `cmd/define/schedule/queue.go` | new |

**Its own package, and that is the one structural decision here.** `cmd/define` is `package main` and this is the first thing in the project with no reason to touch IO at all — the whole issue says "no IO whatsoever". A subpackage makes the purity boundary *checkable from outside*: `schedule` imports only `store` and `time`, and a test that needed a fake would not compile. `store` is already the precedent for a subpackage that earns its own conformance discipline. (AGENTS.local.md's "no `internal/` until a second consumer" is about premature extraction from `main`; here the second consumer — `#8 --stats` — is already specified to share `Mastered`.)

- **`Box`** — which interval a word is on, `0` to `len(intervals)-1`.
  - Intervals are `1, 3, 7, 14, 30, 90` days, fixed. Leitner over SM-2 deliberately: the Spec's reason is that *"why is this due?"* must be answerable in one sentence, and an ease factor cannot be.
  - **A day is a LOCAL CALENDAR day, not 24h**, and this repo already settled that question once: `#15`'s `/history` computes its window from local midnight, with the comment *"It is a local-CALENDAR question, so the boundary is local midnight, not a Duration at all."* The same reasoning applies harder here — a learner who reviews at 9am Monday and sits down at 8am Tuesday must find the word due, and under `N × 24h` they would not. `Due` therefore compares `startOfDay(LastReviewed)` plus N days against `startOfDay(now)`, in the clock's location. **No such helper exists yet** — the first draft said one did, and it does not. `#15` computes the boundary inline twice, in two different shapes: `history_cmd.go:41-42` builds a local midnight and subtracts days, and `relativeDay` at `:195-198` uses a `dayIndex` closure that normalises to UTC before dividing. Two encodings of one idea, already.

**Ownership: `store.StartOfDay(t time.Time) time.Time`,** added by Task 0 below. It goes in `store` rather than `schedule` because `store` already owns `Clock` — the day boundary is a property of the same time model — and because putting it in `schedule` would make `cmd/define` import `schedule` to render `/history`, which is backwards. Both callers then depend on `store`, and `schedule`'s import set stays exactly `store` + `time`.
  - **Future extensions:** the interval table is one slice; a per-learner variant would replace it, not the logic around it.

- **`Progress`** — one word's schedule state: `Box`, `Streak` (consecutive correct), `LastReviewed`. No `Reviews` count: the first draft declared one with no reader in this issue, and `#8` can fold the log for it directly.
  - **Relationships:** 1:1 with a deck word, but only for words that have review events. A word with none has the zero `Progress`, which is meaningfully "new" rather than "missing".
  - **DRY rationale:** `Due`, `Mastered` and `Queue` all read this one shape. Without it each would re-derive box state from raw events differently.

- **`Fold(events []store.ReviewEvent) map[string]Progress`** — the derivation, and the load-bearing decision of this issue.
  - **`Fold` APPLIES `Answer`; it does not reimplement the transition.** Two encodings of "correct promotes, wrong demotes and resets the streak" would be two things to keep in agreement, and the disagreement would be invisible — a deck that folds one way and answers another. `Fold` is a loop over events calling `Answer`, and a test asserts that folding a one-event log equals `Answer` on the zero `Progress`.
  - **Derived, never stored.** `event.go` states the rule for the whole store: the log is *"deliberately the ONLY record of activity ... storing counters alongside would create a second source of truth that drifts."* A `Box` field on `store.Word` would be exactly that second source — and it would go wrong in a way nobody notices, because a hand-edited or partially-written deck file would disagree with the events that produced it. Folding costs one pass over an append-only log the session already reads.
  - Only `EventReviewed` participates. `EventLookedUp` and `EventAsked` are activity, not assessment.
  - **Future extensions:** if the fold ever costs real time, the answer is a cache keyed by the log's length, not a stored field.

- **`Due(p Progress, now time.Time) bool`** — has this word's interval elapsed. The zero `Progress` is due: a word you looked up and never reviewed is exactly what should come first.

- **`Answer(p Progress, correct bool, at time.Time) Progress`** — the transition. Correct promotes one box and increments the streak; wrong demotes and **resets the streak to zero**. Demotion is one box rather than back to zero: a word at box 5 that slips once is not a word you have never seen, and Leitner's whole claim is that the interval carries the information.

- **`Mastered(p Progress) bool`** — final box reached with `masteryStreak` consecutive correct, and **`masteryStreak = 7`**.

  The Spec left N unstated and the number needs a reason rather than a taste. Reaching the last box from box 0 takes 5 consecutive correct answers, so any N of 5 or less makes `Mastered` mean nothing beyond "arrived". 7 is "the five promotions that reach the 90-day interval, plus two confirmations at it" — which is the one-sentence explanation the Spec demands of every scheduling answer.
  - **ONE definition, used by both `--play` and `--stats`** — a Done-when row, and the reason this is an exported function rather than a condition written twice.

- **`Queue(deck []store.Word, prog map[string]Progress, now time.Time, budget int) []string`** — today's words, in order.
  - **Two tiers, and the Done-when forces it.** *"never starves an overdue word in favour of a fresh one"*: words with review history that are due sort FIRST, most overdue first; new words follow, most-looked-up first. Ranking everything on one "overdue-ness" axis would put a word first seen months ago and never reviewed ahead of a genuinely overdue one, because its age is larger — the exact starvation the row forbids.
  - Ties broken by `Lookups` descending, then by key, so the queue is deterministic. A queue that reorders between runs is untestable and looks broken.
  - `budget <= 0` returns nothing rather than everything — "no budget" is not "unlimited", and the opposite reading is a way to accidentally review 400 words.

**Test surface.** Everything is a table test over explicit times, colocated in `schedule/`. No fakes anywhere.

**The purity claim needs a GUARD, because nothing about Go enforces it.** The first draft said "a test needing a fake would not compile", which is simply false — a Go test file may import anything. `TestScheduleImportsOnlyStoreAndTime` reads the package's imports with `go list -f '{{join .Imports "\n"}}'` and asserts the set is exactly `store` + `time` for the non-test files. (`repo_guard_test.go` is the precedent for *shelling out in a guard test*, but it runs `git` over the index and history — `exec.Command("git", ...)` at `:46` and `:82` — not `go list`; the first draft of this plan said otherwise.) An unenforced purity claim is a comment, and this repo has now been bitten four times by facts that lived only in comments.

### Integration points (where pure meets the world)

**None, and that is the deliverable.** The issue says "No IO whatsoever". `#6` reads the deck and the log through the seams that already exist (`store.Store.Deck`, `store.Events`), gets `now` from the injected `store.Clock`, calls `Fold` and `Queue`, and writes `EventReviewed` through the capture path. This issue adds no seam, no dependency and no field.

---

## Chunk 1: M1 — boxes, progress and the fold

### Task 0: `store.StartOfDay`, and collapse the two existing encodings

**Files:**
- Modify: `cmd/define/store/clock.go`, `cmd/define/store/clock_test.go`
- Modify: `cmd/define/history_cmd.go` — both sites (`:41-42`, and `relativeDay`'s `dayIndex` closure at `:195-198`)

- [ ] **Step 1: Write the failing table test** for `StartOfDay`: it returns local midnight of its argument's date in its argument's location; it is idempotent; it is stable across a DST boundary in a zone that has one (the reason `relativeDay`'s closure normalises to UTC before dividing — that comment names the hazard, and a shared helper has to keep the property, not just the shape).

- [ ] **Step 2: Verify red. Step 3: Implement. Step 4: Verify green.**

- [ ] **Step 5: Rewrite both `history_cmd.go` sites to call it,** and confirm `/history`'s existing tests still pass unchanged — they are the regression net for a refactor whose whole risk is changing behaviour while tidying.

- [ ] **Step 6: Commit.** This is a genuine ARCH-DRY fix that `#5` merely forced into the open: two encodings of "which local day is this" already existed before this issue.

### Task 1: `Box` and the interval table

**Files:**
- Create: `cmd/define/schedule/box.go`, `cmd/define/schedule/box_test.go`

- [ ] **Step 1: Write the failing table test.** Strategy: rows are the DECISIONS — box 0's interval, the last box's interval, that promotion past the last box stays there, that demotion below zero stays there, and that the table is strictly increasing (a non-increasing table would make a "promotion" shorten the interval).

- [ ] **Step 2: Verify red. Step 3: Implement. Step 4: Verify green.**

- [ ] **Step 5: Commit.**

### Task 2: `Progress`, `Fold`, `Due`, `Answer`, `Mastered`

**Files:**
- Create: `cmd/define/schedule/progress.go`, `cmd/define/schedule/progress_test.go`

- [ ] **Step 1: Write the failing tests.** Strategy: one table for `Answer`'s transitions (promote, demote, streak reset on wrong, clamping at both ends); one for `Due` across an explicit multi-week timeline; one for `Fold` (only `EventReviewed` counts; events apply in order; a word with no events folds to the zero value; `store.Key` normalisation so `Define` and `define` are one word).

- [ ] **Step 2: The Done-when's own test —** a simulated multi-week schedule with a fixed clock, walking a word from box 0 to mastery and back down after a miss, asserting the due date at each step. This is the row that says "verified across a simulated multi-week schedule".

- [ ] **Step 3: Verify red. Step 4: Implement. Step 5: Verify green.**

- [ ] **Step 6: `FuzzFold`** — with the property stated correctly, because the first draft's was FALSE.

      Permutation-independence does not hold and cannot: two `EventReviewed` sharing an `At` with different `Correct` fold differently depending on order, and `ReviewEvent` has no tiebreaker. The rationale was wrong too — `store.Store.Events` documents chronological order and both implementations sort on `At`, so events do not arrive "in file order".

      **`Fold`'s ordering contract, stated:** it consumes events in the order `Events` returns them and is not order-independent. What the fuzz target asserts instead: the box is always inside the table whatever the input, `Streak` is never negative, and `Fold` is IDEMPOTENT over a re-fold of the same slice. Those hold over the whole domain.

- [ ] **Step 7: Mutation-check** that streak-reset-on-wrong, the box clamps, and the `EventReviewed`-only filter each redden a named test.

- [ ] **Step 7b: Update `atlas/define.md` with M1's surface.** The first draft of
      this plan scheduled ALL atlas work at M2 Task 3 Step 6, and the close gate
      refused M1 for it — correctly. AGENTS.md §8 requires the atlas at EACH
      milestone close, and M1 introduces a whole new package with real
      architectural surface: the derived-not-stored decision, the ordering
      contract, the calendar-day model, `Mastered`'s single definition. Deferring
      is the end-of-project sweep §8 exists to prevent. **This is the second time
      the same deferral has been made and refused — #21 M1 scheduled its atlas at
      M3 and was refused identically.** M2 extends the section rather than writing
      it.

- [ ] **Step 8: `sdlc milestone-close --issue 5 --milestone M1`.**

## Chunk 2: M2 — the queue

### Task 3: `Queue`

**Files:**
- Create: `cmd/define/schedule/queue.go`, `cmd/define/schedule/queue_test.go`

- [ ] **Step 1: Write the failing tests, one per Done-when clause.** Strategy: the budget is never exceeded; an overdue reviewed word outranks a fresh one (the starvation row — the fixture must contain BOTH, or it cannot tell the two orderings apart); ordering within each tier; determinism given equal keys; `budget <= 0` returns nothing; a word in the deck with no events appears; a word with events but not in the deck does NOT (the deck is the roster, the log is the history, and `--forget` deliberately keeps events for a word it removed).

- [ ] **Step 2: Verify red. Step 3: Implement. Step 4: Verify green.**

- [ ] **Step 5: Mutation-check** the tier order, the budget cap and the tie-break; each must redden a named test. The tier-order mutant is the one that matters: the fixture has to make the two orderings differ.

- [ ] **Step 6: Atlas.** EXTEND the scheduling section M1 wrote — the two tiers and the starvation rule they prevent. Link a new `atlas/` file from `atlas/index.md` only if this outgrows a section.

- [ ] **Step 7: `sdlc close --issue 5 --verified '<evidence>'`.**

---

## Risks

- **Derived-not-stored costs a fold per session.** At a few thousand events this is microseconds and the log is already read for `/history`. Worth stating so the first person to see it in a profile knows it was chosen, not overlooked.
- **The intervals are a guess, and honest to call one.** 1/3/7/14/30/90 is conventional Leitner; nothing here measures whether it suits this learner. `#8`'s stats are what would eventually say, and the table is one slice when that day comes.
- **`Mastered` is defined here and consumed by `#8`, which is filed and open** (`workshop/issues/000008-vocab-stats.md`) but not built. The Done-when says one definition used by both. This delivers the definition; the row cannot be fully closed until `--stats` actually calls it, and that is worth recording rather than ticking optimistically. (The first draft said `#8` "does not exist" — it does, as an unstarted issue.)

## Revisions

### 2026-08-26 — plan-quality rounds 1 and 2

Round 1, four blocking findings, all accepted:

- **PQ-1 `time-model-unstated`** — the plan never said whether an interval is
  `N × 24h` or N local calendar days. It is calendar days, and the repo had
  already settled the question once for `/history`: a learner who reviews at 9am
  Monday and returns at 8am Tuesday must find the word due.
- **PQ-2 `duplicate-transition`** — `Fold` and `Answer` would each have encoded
  "correct promotes, wrong demotes and resets". `Fold` now applies `Answer`, with
  a test that a one-event fold equals one `Answer`.
- **PQ-3 `unenforced-claim`** — I wrote that "a test needing a fake would not
  compile", which is false; a Go test may import anything. The purity claim now
  has a guard that reads the package's import set.
- **PQ-4 `invariant-total-over-domain`** — `FuzzFold`'s permutation invariant was
  FALSE: two `EventReviewed` sharing an `At` with different `Correct` fold
  differently by order, and `ReviewEvent` has no tiebreaker. The rationale was
  wrong too — `Events` documents chronological order and both implementations
  sort on `At`. Replaced with box-in-range, non-negative streak and idempotence,
  which hold over the whole domain, and `Fold`'s ordering contract is now stated.

Round 2, one blocking finding, and it is the sharper one:

- **PQ-6 `claim-about-existing-code`** — I asserted three things about the tree
  and all three were wrong: that a `startOfDay` helper exists (it does not —
  `#15` computes the boundary inline in two DIFFERENT shapes), that
  `repo_guard_test.go` uses `go list` (it runs `git`), and that `#8` does not
  exist (it is a filed, open issue). Each is now verified with a file:line, and
  the helper's ownership is decided rather than assumed: `store.StartOfDay`,
  because `store` already owns `Clock` and putting it in `schedule` would make
  `cmd/define` import `schedule` to render `/history`.

  **The rule the finding states, and it is the one worth keeping: every claim
  about existing code carries a `file:line` or is rewritten as the intent it
  stood in for.** Writing "the helper exists" from memory cost a round; grepping
  for it costs seconds. This is the same family as the plan-record staleness that
  ran six rounds on `#21`.

  **And the sweep for it must enumerate ARTIFACTS, not just the file the finding
  named.** Round 3 caught the residue: I fixed three instances in this plan and
  left the identical `#8 does not exist` claim standing in the issue's Done-when.
  The enumeration for a claim-about-code sweep in this repo is: the issue file,
  the plan file, `atlas/`, and code comments — four places, checked by grepping
  the claim rather than by remembering where it was written.

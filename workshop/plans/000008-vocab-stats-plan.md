# `define --stats` Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** One screen that answers "is any of this working", every figure folded
out of the append-only event log.

**Architecture:** A pure fold from `[]store.ReviewEvent` + a clock to a `Stats`
struct, and a separate pure renderer from `Stats` to lines. Nothing is stored;
nothing is counted twice. The command is the thin shell that reads the log and
prints the lines — the same shape `--history` already has, which is the sibling
to copy rather than a pattern to invent.

**Tech Stack:** Go 1.24. No new dependencies. `cmd/define/schedule` (pure fold),
`cmd/define` (flag + render).

---

## Core concepts

### What this issue is NOT

Half the figures the Spec names already exist as pure functions, and the failure
mode of a stats screen is a second implementation that disagrees with the first.
So, explicitly (ARCH-DRY):

| figure | already exists | this issue |
|---|---|---|
| a word's box, max box, lapses | `schedule.Fold` | folds it, adds nothing |
| whether a word is mastered | `schedule.Mastered` | calls it, does not re-decide |
| a local calendar day | `store.StartOfDay` | uses it |
| days between two instants | `store.DaysBetween` | uses it — this is the DST-correct one, and re-deriving it is how a streak breaks on the 25-hour day |
| a window of "the last N days" | `historyWindow` | reuses it if `--stats` grows a window; MVP has none |

**A second definition of "mastered" is the specific thing this plan refuses.**
`MasteredBox` is 9 and `Mastered` reads `MaxBox`, not `Box`, so a lapsed word
stays mastered. A stats screen that counted `Box >= 9` would disagree with the
sitting's own display for exactly the words a learner would ask about.

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Stats` | `cmd/define/schedule/stats.go` | new |
| `Summarise` | `cmd/define/schedule/stats.go` | new |
| `FormAccuracy` | `cmd/define/schedule/stats.go` | new |
| `activeDays` | `cmd/define/schedule/stats.go` | new |
| `renderStats` | `cmd/define/stats.go` | new |

- **`Stats`** — every figure the screen shows, and nothing else.
  - **Relationships:** 1:1 with a fold over the whole log. Holds
    `map[string]FormAccuracy` keyed by the form NAME (`meaning`, `cloze`,
    `board`), which is what `Outcome.Form` already records and what the log
    already speaks in — no enum to keep in step with `play`.
  - **DRY rationale:** the alternative is the command computing seven numbers
    inline, where each is untestable without parsing a screen. The Spec asks for
    exactly this split.
  - **Future extensions:** a window (`--stats --days 30`) is one parameter on
    `Summarise`, because a fold over a filtered slice is the same fold.

- **`Summarise(events []store.ReviewEvent, deck []store.Word, now time.Time) Stats`**
  — the fold.
  - **Why it takes the DECK as well as the log.** "Words known" is a property of
    the deck, and the log alone cannot answer it: `--forget` removes a word while
    its events remain (deliberately — the deck is a working set, the log is
    history). Folding only the log would count words the learner has deleted.
    That asymmetry is the single most likely wrong answer on this screen and it
    is why the signature takes both.
  - **PURE, including the clock.** `now` is a parameter, so "current streak"
    is testable at any instant and across any timezone without a fake clock
    interface. ARCH-PURE.

- **`FormAccuracy`** — attempts and correct, per form.
  - **What counts as an attempt** is the one judgement here, and it is `#12`'s
    to inherit: an `EventFlagged` is NOT an attempt. A broken question is not
    evidence about the learner — the same reason `Fold` ignores the kind — so
    counting it would let bad material lower a learner's accuracy.

- **`activeDays`** — the set of local calendar days carrying at least one event,
  which both `ActiveDays` and the two streaks are read off.
  - **ONE traversal, three figures.** Current streak, longest streak and the
    active-day count are three questions about one set; computing them
    separately is three chances to disagree about what a day is.

**Test surface.** `schedule/stats_test.go`, colocated, no IO and no fake — the
package's `purity_test.go` already guards that `schedule` imports nothing but
`store` and the standard library, so the boundary is mechanical rather than
promised.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `runStats` | `cmd/define/stats.go` | new | the store and stdout |

- **`runStats(d deps, opt options, out io.Writer) int`** — read the log, read the
  deck, fold, render, print.
  - **Injected into:** nothing; it is the shell. It mirrors `runReflect` and the
    `--history` path, which is what makes it reviewable at a glance.
  - **No store, no problem.** `d.deck == nil` (no directory, or
    `DEFINE_NO_CAPTURE`) prints the same sentence `--play` prints for the same
    cause, rather than a screen of zeros that reads as "you have done nothing".

**Test surface for integration points.** `store.Mem` through the existing
`testDeps` rig — no new fake. The store is already behind a conformance-tested
seam; this adds no external dependency, so ARCH-MOCK has nothing new to answer.

### ARCH-CONSTRAINTS — the operating envelope

`Summarise` is O(events + deck) with a map of days and a map of forms. The log is
append-only and one file per day; a heavy year is a few thousand events and a
deck is a few thousand words, so this is milliseconds and there is no paging,
no incremental cache, and deliberately no stored counter. **If the log ever grows
past what a single fold can carry, the fix is a window, not a counter** — a
counter reintroduces the drift `#3` chose this shape to avoid.

### ARCH-ORDER — state and events

`Summarise` holds no state between calls and reads a snapshot, so most of the
lens is `N/A` — but written out rather than marked: there is no cancellation
path (a fold that returns is done), no concurrency (one process, one read), and
no ordering dependency between events beyond the timestamp each carries. The one
real ordering question is that **the log is not guaranteed sorted** — it is one
file per UTC day, appended within each — so anything reading "first" or "last"
must not assume position. The fold takes min/max rather than `events[0]`.

---

## Chunk 1: the fold

### Task 1: `Stats`, and the deck-vs-log asymmetry

**Files:**
- Create: `cmd/define/schedule/stats.go`, `cmd/define/schedule/stats_test.go`

- [ ] **Step 1: Write the failing test for the asymmetry that defines the shape**

```go
// WORDS KNOWN COMES FROM THE DECK, NOT THE LOG, and the two genuinely differ:
// --forget removes a word and deliberately leaves its events, because the deck
// is a working set and the log is history. A fold over the log alone counts
// words the learner has deleted, which is the most likely wrong number on this
// screen.
func TestKnownCountsTheDeckNotTheLog(t *testing.T)
```

- [ ] **Step 2: Run it, watch it fail** (`Summarise` undefined).
- [ ] **Step 3: Implement `Stats` + `Summarise` for `Known` and `Mastered` only.**
      `Mastered` calls `schedule.Mastered(prog[key])` — never `Box >= MasteredBox`.
- [ ] **Step 4: Run it, watch it pass.**
- [ ] **Step 5: Pin the mastery agreement.** A lapsed word (high `MaxBox`, low
      `Box`) counts as mastered here exactly as the sitting displays it; reverting
      to `Box >= MasteredBox` must redden a named row.
- [ ] **Step 6: Commit.**

### Task 2: active days and the two streaks

**Files:**
- Modify: `cmd/define/schedule/stats.go`, `cmd/define/schedule/stats_test.go`

- [ ] **Step 1: Write the failing tests — the calendar rows first**

```go
// A STREAK IS A LOCAL CALENDAR QUESTION. StartOfDay and DaysBetween already
// carry that reasoning (and the DST correction: a day is 23 or 25 hours, so
// "now minus 24h" lands on the wrong date twice a year). This must USE them
// rather than re-derive.
func TestStreaksCountLocalCalendarDays(t *testing.T)

// TODAY IS NOT REQUIRED. A learner who reviewed yesterday and has not yet sat
// down today still has their streak — breaking it at midnight would punish them
// for the hour they read the screen.
func TestAStreakSurvivesUntilADayIsMissed(t *testing.T)

// A gap ends the current streak and preserves the longest.
func TestTheLongestStreakSurvivesAGap(t *testing.T)

// The DST days this repo's other date code names: 2026-03-08 (23h) and
// 2026-11-01 (25h). Events an hour apart across those must land on the days a
// reader would say they did.
func TestStreaksAcrossDSTBoundaries(t *testing.T)

// A timezone whose offset is not a whole hour, because a fold using Duration
// arithmetic passes every whole-hour test and fails here.
func TestStreaksInAHalfHourOffsetZone(t *testing.T)
```

- [ ] **Step 2: Run them, watch them fail.**
- [ ] **Step 3: Implement `activeDays` + the two streaks** over `store.StartOfDay`
      and `store.DaysBetween`.
- [ ] **Step 4: Run them, watch them pass.**
- [ ] **Step 5: Mutation sweep.** Replace `DaysBetween` with a
      `Sub()/24h` computation and confirm the DST row reddens BY NAME. *A pin
      that cannot fail is not a pin*, and this is the row the Done-when names.
- [ ] **Step 6: Commit.**

### Task 3: added-per-day and accuracy by form

**Files:**
- Modify: `cmd/define/schedule/stats.go`, `cmd/define/schedule/stats_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// A FLAGGED QUESTION IS NOT AN ATTEMPT. #12 records the option set and moves on
// without marking the learner wrong, and Fold ignores the kind for the same
// reason: a broken question is not evidence about the learner. Counting it here
// would let bad material lower an accuracy score.
func TestAFlaggedQuestionIsNotAnAttempt(t *testing.T)

// Accuracy is keyed by the form NAME the log already records, so a form added to
// play appears here with no edit.
func TestAccuracyIsKeyedByTheFormTheLogRecords(t *testing.T)

// Added-per-day is over the window the learner has actually been using this,
// not since the epoch — an average diluted by dormant months answers a question
// nobody asked.
func TestAddedPerDayIsOverTheActiveWindow(t *testing.T)
```

- [ ] **Step 2-4: Red, implement, green.**
- [ ] **Step 5: Commit.**

## Chunk 2: the screen

### Task 4: `renderStats`

**Files:**
- Create: `cmd/define/stats.go`, `cmd/define/stats_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// AN EMPTY DECK RENDERS SENSIBLY — a Done-when row, and the state every new
// learner is in. Not a screen of zeros that reads as failure: it says there is
// nothing yet and what to do about it.
func TestStatsOnAnEmptyDeckSaysSoRatherThanPrintingZeros(t *testing.T)

// Every figure the Spec names reaches the screen. Derived from the Stats struct
// so a field added without a line reddens, rather than being silently invisible.
func TestEveryStatsFieldIsRendered(t *testing.T)
```

The second uses reflection over `Stats` — the same "derive the extent" move
`TestEveryFormIsEnrolled` and `numRegionKinds` make, and the correction `#12`
BR-17 forced: a hand-listed set of fields is half a guard.

- [ ] **Step 2-4: Red, implement, green.**
- [ ] **Step 5: Commit.**

### Task 5: the flag and the shell

**Files:**
- Modify: `cmd/define/main.go` (flag + dispatch)
- Modify: `cmd/define/README.md`, `atlas/define.md`
- Test: `cmd/define/stats_test.go`

- [ ] **Step 1: Write the failing end-to-end test** through `run()` with a
      `store.Mem` holding a deck and a log.
- [ ] **Step 2: Register `-stats`** beside `-reflect` and `-play`, and dispatch
      it the same way.
- [ ] **Step 3: The no-deck path** prints what `--play` prints for the same
      cause, not zeros.
- [ ] **Step 4: `/stats` in the REPL too**, if the command table makes it a row
      rather than a feature — check `command.go` and do it only if it is a row.
- [ ] **Step 5: README and atlas.** The README's key table and command list are
      both guarded by derived tests; adding a command must satisfy them.
- [ ] **Step 6: Run the whole suite, `go vet` under all three tag sets, gofmt.**
- [ ] **Step 7: Commit, then `sdlc close --issue 8`.**

---

## Verification

1. `go test ./...`, `go vet` under default, `pty` and `conformance` tags, `gofmt -l` clean.
2. `TestKnownCountsTheDeckNotTheLog` — the asymmetry that shapes the signature.
3. `TestStreaksAcrossDSTBoundaries` + `TestStreaksInAHalfHourOffsetZone` — the
   Done-when's timezone row, mutation-swept against a Duration implementation.
4. `TestAFlaggedQuestionIsNotAnAttempt` — bad material cannot lower a score.
5. `TestStatsOnAnEmptyDeckSaysSoRatherThanPrintingZeros` — the Done-when's empty row.
6. `TestEveryStatsFieldIsRendered` — a field with no line reddens.
7. Manual, once: `--stats` in the smoke deck after a sitting, and in an empty
   directory. A screen is a thing a person reads.

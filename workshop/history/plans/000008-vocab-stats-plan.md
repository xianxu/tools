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

### Every claim about existing code carries a file:line

**The rule this plan is under, written down because it was broken four times in
two rounds.** PQ-1: "`Mastered` reads `MaxBox`" — it reads `Box`. PQ-6: "the log
is not guaranteed sorted" — `store.go:18` promises chronological order and both
implementations sort. PQ-7: "`Fold` gives box, max box, lapses" — `Progress` has
no lapse count. PQ-9: "`modeCollision`'s table test derives from the slice" — it
hand-lists.

Every one was written from recollection while ARGUING for reuse, and three of
them would have produced code or tests asserting behaviour that does not exist.
So: **a declarative sentence about what existing code does carries a `file:line`
and is verified before it is written.** The citations below are not decoration;
they are the evidence that the sentence was checked.

**The fourth is not just a plan error — it is a live bug this issue trips.**
`main.go:596-597` says *"modeCollision's table test derives from this"* and
`harvest_test.go:622-626` says *"a sixth mode added to run()'s slice is covered
by construction"*. Both are false: `TestModeCollision` hand-lists five modes at
`harvest_test.go:628-631`. `#8` adds the sixth, which would be the first mode
the guard does not cover — the "a hand-maintained extent is half a guard" family
(`#12` BR-17, `#46` BR-21) arriving in a place two comments already claim is
safe. Task 5 fixes it.

### What this issue is NOT

Half the figures the Spec names already exist as pure functions, and the failure
mode of a stats screen is a second implementation that disagrees with the first.
So, explicitly (ARCH-DRY):

| figure | already exists | this issue |
|---|---|---|
| a word's box and high-water mark | `schedule.Fold` → `Progress{Box, MaxBox, LastReviewed}` (`progress.go:52-63`) | folds it, adds nothing. There is no lapse COUNT to reuse; an earlier draft credited `Fold` with one |
| whether a word is mastered | `schedule.Mastered` (`progress.go:130`) | calls it, does not re-decide — its doc already names this issue as the second consumer |
| a local calendar day | `store.StartOfDay` (`clock.go:41`) | **NOT used in the end** — see the 2026-09-08 revision: it returns an INSTANT, and where DST moves at local midnight that instant does not exist. `civilDay` keys by date instead |
| days between two instants | `store.DaysBetween` (`clock.go:46`) | **its RULE is used, not the function** — "b's location defines the calendar" is applied where the keys are built; the walk itself is `civilDay.add` |
| a window of "the last N days" | `historyWindow` (`history_cmd.go:37`) | reuses it if `--stats` grows a window; MVP has none |

**A second definition of "mastered" is the specific thing this plan refuses**,
and `Mastered`'s own doc comment says so in advance: *"Exported and defined once
because two consumers need the same answer: `#6`'s `--play` decides what to stop
highlighting and `#8`'s `--stats` reports how many words are known. Two
conditions written separately would drift, and the drift would show as a stats
screen disagreeing with the review queue."* This issue is that second consumer,
and calling the function is the whole of its obligation.

**What it actually says** (`p.Box >= MasteredBox`, box 9): a LAPSED word is not
mastered, because `Box` falls on a wrong answer while `MaxBox` does not. The
first draft of this plan asserted the opposite — that `Mastered` reads `MaxBox`,
so a lapsed word stays mastered — and built a pin on it. Plan-quality caught it.
Recorded rather than silently corrected, because the mistake is instructive: the
DRY argument for calling the function is exactly that a second author's
*recollection* of the rule is not the rule.

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Stats` | `cmd/define/schedule/stats.go` | new |
| `Summarise` | `cmd/define/schedule/stats.go` | new |
| `FormAccuracy` | `cmd/define/schedule/stats.go` | new |
| `civilDay` | `cmd/define/schedule/stats.go` | new |
| `countable` / `countableEvents` | `cmd/define/schedule/stats.go` | new |
| `streaks` | `cmd/define/schedule/stats.go` | new |
| `printStats` | `cmd/define/stats.go` | new |
| `runStatsCommand` | `cmd/define/stats.go` | new |
| `renderStats` | `cmd/define/stats.go` | new |

- **`Stats`** — every figure the screen shows, and nothing else.
  - **Relationships:** 1:1 with a fold over the whole log. Holds
    `map[string]FormAccuracy` keyed by the form NAME, which is what
    `ReviewEvent.Form` already stores (`store/event.go:108`, a `string`) — no
    enum to keep in step with `play`.
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

- **the day set** — the local calendar days carrying at least one event, which
  `ActiveDays` and both streaks are read off.
  - **ONE traversal, three figures.** Current streak, longest streak and the
    active-day count are three questions about one set; computing them separately
    is three chances to disagree about what a day is.
  - **It did NOT become a named function.** The first draft of this table listed
    `activeDays` as an entity; it shipped as a local `days` map inside
    `Summarise` plus `streaks(days, now)`, because a function returning a set
    that only one caller builds and only one caller reads is a seam with nothing
    on either side of it. `TestPlanTablesNameEntitiesThatExist` caught the row —
    a plan naming a function the code never declares reads as evidence that it
    exists.

**Test surface.** `schedule/stats_test.go`, colocated, no IO and no fake —
`TestSchedulePurity` (`schedule/purity_test.go:17`) already enforces "no IO and
no hidden clock" for this package, so the boundary is mechanical rather than
promised.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `runStats` | `cmd/define/stats.go` | new | the store and stdout |

- **`runStats(ctx context.Context, d deps, opt options, out, errOut io.Writer) int`**
  — read the log, read the deck, fold, render, print.
  - **The signature MIRRORS `runReflect` (`reflect.go:338`)**, which is the
    closest sibling: same package, same shape, also a mode. Two writers because a
    diagnostic is not output — a `--stats` piped to a file must not have "could
    not read the log" in the middle of it.
  - **EXIT CODES:** 0 when the screen printed, including on an EMPTY deck (there
    is nothing wrong with having done nothing yet); 1 when the deck or the log
    could not be read, because the figures would then be silently low rather than
    absent, and a wrong number is worse than a refusal.

    **A NIL DECK IS ALSO 1, on stderr** — a correction. This plan first said 0
    to stdout, "a statement about the directory, not a failure". Every sibling
    disagrees: `--forget` (`main.go:1193`), `--harvest` (`harvest.go:111`),
    `--reflect` (`reflect.go:340`) and `/history` (`history_cmd.go:220`) all
    print `noDeckMessage` to stderr and return 1, unanimously. The case the
    original argument was protecting is the EMPTY deck, which still exits 0.
  - **Injected into:** nothing; it is the shell. It mirrors `runReflect` and the
    `--history` path, which is what makes it reviewable at a glance.
  - **No store, no problem — through `noDeckMessage`, not a fourth string.**
    `d.deck == nil` has TWO causes and they need different sentences:
    `DEFINE_NO_CAPTURE` is set (nothing was opened on purpose) or there is no
    deck in this directory. `noDeckMessage(opt.noCapture)` already distinguishes
    them and is already shared by `--forget` and `/history`; its own comment
    gives the reason — *"the same fact stated in two places is how the atlas
    contradictions in `#4` started"*. `--play`'s hardcoded string does NOT make
    that distinction, so copying it would tell a `DEFINE_NO_CAPTURE` user their
    directory is empty. ARCH-DRY.

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

### ARCH-SECURE — the log is hand-editable input

`events/` is plain YAML in a directory the README explicitly invites editing, and
every figure here is folded from it. **A bad timestamp is the attack surface, and
it is silent**: a zero `At` puts an event on year 1, which makes the day set
span two thousand years and the "longest streak" arithmetic meaningless; a
far-future `At` breaks the current streak by leaving a gap nobody can close.

So the fold VALIDATES rather than trusts:

- an event whose `At` is zero is skipped — it is not a day;
- an event dated after `now` is skipped, because a fold cannot be evidence about
  the future, and this is the shape a clock-skewed or hand-edited row takes;
- the figures degrade to "fewer events" rather than to a wrong number, which is
  the same choice `sanitiseItem` and `readCapped` make one package over.

**And the FORM NAME is text from that file reaching a terminal.**
`ReviewEvent.Form` is a bare `string` (`store/event.go:108`) written by
`Outcome.Form`, and this screen prints it as a row label — the first path that
puts it on screen. A hand-edited `form: "meaning\x1b[2J"` would clear the
display, which is `#12` BR-15 exactly, one field over. The renderer neutralises
it the way `oneLine` does (`store/item.go`): control runes that are not
whitespace are dropped. An unknown form name is still SHOWN rather than filtered
— a row labelled with a name nobody recognises is how a learner discovers a
stale or hand-edited log, where silently dropping it hides the fact.

**Every one of these is pinned**, not merely declared: a zero `At`, a future
`At`, and an escape in `Form` each get a row asserting the figure the fold
produces, and each must redden when its skip is removed.

The alternative — validating in the store — is wrong here: the store's job is to
return what is written, and `#3`'s whole design is that the log is the record.
The consumer decides what it can count.

### ARCH-ORDER — state and events

`Summarise` holds no state between calls and reads a snapshot, so most of the
lens is `N/A` — but written out rather than marked: there is no cancellation
path (a fold that returns is done), no concurrency (one process, one read), and
no ordering dependency between events beyond the timestamp each carries. **The log IS chronological** — `store/store.go:18` promises it ("Events returns
events at or after since, in chronological order") and both implementations sort
to keep it (`mem.go:105`, `yaml.go:353`). An earlier draft of this plan claimed
the opposite and would have defended against a hazard the seam already excludes.

The fold still takes min/max over timestamps rather than reading `events[0]`,
but for a DIFFERENT and smaller reason: with the ARCH-SECURE skips below, the
first element may be one of the skipped ones, so position is not the same
question as order.

---

## Chunk 1: the fold

### Task 1: `Stats`, and the deck-vs-log asymmetry

**Files:**
- Create: `cmd/define/schedule/stats.go`, `cmd/define/schedule/stats_test.go`

- [x] **Step 1: Write the failing test for the asymmetry that defines the shape**

```go
// WORDS KNOWN COMES FROM THE DECK, NOT THE LOG, and the two genuinely differ:
// --forget removes a word and deliberately leaves its events, because the deck
// is a working set and the log is history. A fold over the log alone counts
// words the learner has deleted, which is the most likely wrong number on this
// screen.
func TestKnownCountsTheDeckNotTheLog(t *testing.T)
```

- [x] **Step 2: Run it, watch it fail** (`Summarise` undefined).
- [x] **Step 3: Implement `Stats` + `Summarise` for `Known` and `Mastered` only.**
      `Mastered` calls `schedule.Mastered(prog[key])` — never `Box >= MasteredBox`.
- [x] **Step 4: Run it, watch it pass.**
- [x] **Step 5: Pin the mastery AGREEMENT, which is what can actually drift.**
      A lapsed word (high `MaxBox`, `Box` back below 9) is NOT mastered, here and
      in the sitting alike. The pin is that both answers come from one function:
      assert `Stats.Mastered` equals the count of `schedule.Mastered` over the
      same progress map, so an inlined predicate — of any spelling, including a
      correct one that later drifts — reddens.
- [x] **Step 6: Commit.**

### Task 2: active days and the two streaks

**Files:**
- Modify: `cmd/define/schedule/stats.go`, `cmd/define/schedule/stats_test.go`

- [x] **Step 1: Write the failing tests — the calendar rows first**

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
func TestStreaksInFractionalOffsetZones(t *testing.T)
```

- [x] **Step 2: Run them, watch them fail.**
- [x] **Step 3: Implement the day set + the two streaks.** Over `civilDay` in
      the end, not `store.StartOfDay` — the revision below records why.
- [x] **Step 4: Run them, watch them pass.**
- [x] **Step 5: Mutation sweep.** Restore the `StartOfDay`-derived key and
      confirm the midnight-DST row reddens BY NAME. *A pin
      that cannot fail is not a pin*, and this is the row the Done-when names.
- [x] **Step 6: Commit.**

### Task 3: added-per-day and accuracy by form

**Files:**
- Modify: `cmd/define/schedule/stats.go`, `cmd/define/schedule/stats_test.go`

- [x] **Step 1: Write the failing tests**

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

// AND IT COUNTS FROM THE LOG, unlike Known — the same deck-vs-log choice, going
// the other way, because the two questions differ. "How many words do I know" is
// about the deck as it stands; "how fast am I adding them" is about what
// HAPPENED, and a forgotten word was still a word added that day. Folding the
// deck for this would rewrite the past every time --forget ran.
func TestAddedPerDayCountsForgottenWordsToo(t *testing.T)
```

- [x] **Step 2-4: Red, implement, green.**
- [x] **Step 5: Commit.**

## Chunk 2: the screen

### Task 4: `renderStats`

**Files:**
- Create: `cmd/define/stats.go`, `cmd/define/stats_test.go`

- [x] **Step 1: Write the failing tests**

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

- [x] **Step 2-4: Red, implement, green.**
- [x] **Step 5: Commit.**

### Task 5: the flag and the shell

**Files:**
- Modify: `cmd/define/main.go` (flag + dispatch)
- Modify: `cmd/define/README.md`, `atlas/define.md`
- Test: `cmd/define/stats_test.go`

- [x] **Step 1: Write the failing end-to-end test** through `run()` with a
      `store.Mem` holding a deck and a log.
- [x] **Step 2: Register `-stats` as a MODE, which is three edits and a guard.**
      `main.go:598`'s `modes` slice is the list AND the collision check —
      `modeCollision`'s table test derives from it, so a mode added there is
      covered by construction. Then the `fs.NArg() != 0` refusal beside
      `-reflect`'s and `-play`'s, because `define -stats sycophantic` is two
      commands on one line, which `#2` shipped the wrong way once.

      **Pin the registration, not just the behaviour:** a test that `-stats` with
      a word is refused, and one that `-stats -play` collides. Without them,
      omitting the slice entry leaves a mode that silently coexists with every
      other.

- [x] **Step 2b: Make `TestModeCollision` actually derive, because it does not.**
      `main.go:596-597` claims *"modeCollision's table test derives from this"*
      and `harvest_test.go:622-626` claims *"a sixth mode added to run()'s slice
      is covered by construction"*. Both are false: the test hand-lists five
      modes at `harvest_test.go:628-631`, and `#8` is the sixth — the first one
      the guard would not have covered, in a place two comments say is safe.

      Extract the slice so `run()` and the test read ONE source (the move
      `docSyncForms` makes for forms, `#12` BR-10), and fail closed on the count
      so a derivation that finds fewer than the code declares is loud rather than
      silent (`#12` BR-17). Then mutate: remove `-stats` from the source and
      confirm the guard names it.
- [x] **Step 3: The no-deck path** goes through `noDeckMessage(opt.noCapture)`,
      so the `DEFINE_NO_CAPTURE` cause and the no-directory cause say different
      things. Pin both, and assert the sentence comes from the helper rather than
      matching a literal — a test asserting the literal is a fourth statement of
      the fact.
- [x] **Step 4: `/stats` in the REPL too**, if the command table makes it a row
      rather than a feature — check `command.go` and do it only if it is a row.
- [x] **Step 5: README and atlas.** The README's key table and command list are
      both guarded by derived tests; adding a command must satisfy them.
- [x] **Step 6: Run the whole suite, `go vet` under all three tag sets, gofmt.**
- [x] **Step 7: Commit, then `sdlc close --issue 8`.**

---

## Verification

1. `go test ./...`, `go vet` under default, `pty` and `conformance` tags, `gofmt -l` clean.
2. `TestKnownCountsTheDeckNotTheLog` — the asymmetry that shapes the signature.
3. `TestStreaksAcrossDSTBoundaries` + `TestStreaksInFractionalOffsetZones` — the
   Done-when's timezone row, mutation-swept against a Duration implementation.
4. `TestAFlaggedQuestionIsNotAnAttempt` — bad material cannot lower a score.
5. `TestStatsOnAnEmptyDeckSaysSoRatherThanPrintingZeros` — the Done-when's empty row.
6. `TestEveryStatsFieldIsRendered` — a field with no line reddens.
7. Manual, once: `--stats` in the smoke deck after a sitting, and in an empty
   directory. A screen is a thing a person reads.

## Revisions

### 2026-09-08 — the close review, round 1

**BR-4 — the comment claiming DRY was not DRY.** `runStats` and
`runStatsCommand` each did their own `Deck()`, `Events()`, clock read,
`Summarise` and render loop, under a doc comment reading *"ONE FOLD, ONE
RENDERER, TWO ENTRY POINTS… everything below that seam is shared"*. Five
duplicated statements. `printStats` is where that sentence became true; the two
doors now differ only in where the store comes from and in the name a diagnostic
carries.

**BR-1 — the mode guard derived the wrong extent.** `#8` fixed
`TestModeCollision` to parse `run()`'s `modes` slice instead of hand-listing it,
and that was still not enough: the guard derives whatever is IN the list, so
deleting `{"-stats", *statsFlag}` left it deriving five modes and passing, while
`-stats` went on dispatching and colliding with nothing.

The extent that matters is the DISPATCH, not the list.
`TestEveryDispatchedModeIsInTheCollisionList` reads both out of `main.go` — every
`x := fs.Bool("name", …)` for the variable-to-name map, every `if *x { return
run…() }` for what run() actually treats as a mode — and requires them to agree.
Mutation-checked on `-stats` and on `-reflect`, so it is not specific to the
mode that exposed it. This is the both-directions closure `#46` BR-9 arrived at:
one side cannot hide what the other declares.

**BR-2, BR-3 — this plan's own drift.** The entity table listed `activeDays`,
which shipped as a local map plus `streaks(days, now)` rather than as a function;
the exit-code paragraph still said a nil deck returns 0, which it no longer does;
and unticked step boxes were suppressing `TestPlanTablesNameEntitiesThatExist` at
the boundary. All three are the family `workshop/targets/derived-restatement.md`
exists for, in the plan rather than in the code.

### 2026-09-08 — `StartOfDay` and `DaysBetween` are NOT called

**Reason.** The close review (BR-2, third pass) found this plan still claiming
both functions are used. They were, in the first implementation; `fee2e1a`
removed the last call site and the rows above went false.

**Delta.** The day set is keyed by `civilDay` — a bare year/month/day — because
`StartOfDay` returns an INSTANT and an instant can fail to exist. Where a DST
transition happens at local midnight, `time.Date` normalises 00:00 backwards:
Havana's `StartOfDay(2026-03-08)` is `2026-03-07T23:00`, a key on the previous
day's date, so any run through that night read as broken. `DaysBetween`'s RULE —
"b's location defines the calendar" — is still what decides whose calendar this
is, applied where the keys are built; the walk is `civilDay.add`, which does its
arithmetic at noon UTC where no clock has ever moved.

The entity table also gained the five rows it was missing: `civilDay`,
`countable`/`countableEvents`, `streaks`, `printStats` and `runStatsCommand`.

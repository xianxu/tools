# REPL command mode + `/history` Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A `/` in the first column switches the REPL line to command mode with type-ahead, and the first command — `/history` — lists words looked up in the last N local days, deduped, ordered by when each was first ever seen.

**Architecture:** Two seams already exist and both are reused rather than duplicated.

1. **`Apply(e, k, matches)` takes its candidate list from the caller**, so command-mode type-ahead needs no change to the pure editor — only a different match source when the line starts with `/`.
2. **`parseREPLLine` is the shared decision table both loops already route through** (`repl.go:130` for the piped loop, `replraw.go:99` for the raw editor, which carries an ARCH-DRY comment saying exactly why). Dispatch is therefore a new `cmdCommand` kind on that classifier — *not* a branch in `submitLine`, which the raw path reaches and the line loop does not.

Commands live in one table, so a second command is a row, not a code change.

**Tech Stack:** Go 1.26, existing `cmd/define` editor + `cmd/define/store` (YAML event log), `store.Clock` for injected time.

---

## Chunk 1: Core concepts

### The one hard part: "the last two days" is a LOCAL-time question over UTC-named files

`AppendEvent` names day files `events/<UTC date>.yaml` and says why:

> Day files are named in UTC so the grouping is stable across timezone changes.
> The stored timestamp keeps its OFFSET, so a local-day view is fully
> recoverable — `#8` must group by the timestamp, never by the filename.

`/history` is the first consumer to actually need that, so three rules follow:

1. **Never window by filename.** A lookup at 19:00 PDT on the 20th lives in
   `2026-08-21.yaml`. Filtering files by name drops exactly the evening lookups a
   two-day window most wants.
2. **`Events` already does the right thing.** It `ReadDir`s every day file and
   keeps `!ev.At.Before(since)` — a timestamp comparison. Measured, not assumed:
   `yaml.go:131-163`. So correctness here is a matter of computing `since`
   correctly and letting the store filter.
3. **The window is local midnights, not `now - 48h`.** "The last two days" means
   today and yesterday as a human reads a calendar, so the boundary is
   `time.Date(y, m, d, 0,0,0,0, time.Local).AddDate(0,0,-(days-1))`. Using
   `AddDate` rather than subtracting `48*time.Hour` is what makes a DST
   transition — a 23- or 25-hour local day — land on the right instant.

**Ordering needs the whole log, and that is free.** Membership is "queried inside
the window"; ordering is "when the word was FIRST ever seen", so a word you keep
returning to holds its position instead of churning to the top. Those are two
different time facts about the same word. Because `Events` reads every day file
whatever `since` says, requesting `Events(time.Time{})` costs exactly what
`Events(since)` costs and answers both from **one source** (ARCH-DRY) — no second
read of `Deck()` for `FirstSeen`.

> Recorded as a known cost, not a defect: `Events` is O(all history) per call. It
> is one read per `/history` invocation on a personal word list, so it is the
> right trade today. When it stops being right, the fix belongs in the store
> (an index or a filename pre-filter that still *decides* on timestamps), not in
> this feature.

### Pure entities

| Name | Lives in | Kind | Status |
|------|----------|------|--------|
| `command` | `cmd/define/command.go` | PURE | new |
| `commands` (the table) | `cmd/define/command.go` | PURE | new |
| `parseCommandLine` | `cmd/define/command.go` | PURE | new |
| `commandCompletions` | `cmd/define/command.go` | PURE | new |
| `nearestCommands` | `cmd/define/command.go` | PURE | new |
| `completionsFor` | `cmd/define/command.go` | PURE | new |
| `historyWindow` | `cmd/define/history_cmd.go` | PURE | new |
| `parseHistoryArgs` | `cmd/define/history_cmd.go` | PURE | new |
| `replKind.cmdCommand` | `cmd/define/repl.go` | PURE | modified |
| `summariseLookups` | `cmd/define/history_cmd.go` | PURE | new |
| `renderHistory` | `cmd/define/history_cmd.go` | PURE | new |

- **command** — `{name, summary string; run func(commandCtx, []string) int}`. The
  table is data; only `run` touches the world.
  - **DRY rationale:** "Adding a second command needs no change to the dispatch
    loop" is a Done-when. One table, one dispatch.
  - **Future extensions:** `/stats` (`#8`), `/play` (`#6`) are rows.
- **parseCommandLine** — `"/history 7"` → `("history", ["7"], true)`; a line not
  starting with `/` → `ok=false`. Leading/trailing space tolerated; a bare `/`
  yields `("", nil, true)` so the UI can list everything.
- **commandCompletions** — names with the typed prefix, for type-ahead. Returns
  them `/`-prefixed so the editor's existing `Suggestion`/`RenderLine` need no
  special case.
- **nearestCommands** — for an unknown command: prefix matches, else names within
  edit distance 2, else all names. Feeds "suggest rather than silently define".
- **completionsFor** — the switch: `parseCommandLine` says command, so complete
  from `commands`; otherwise from `hist.Prefix`. **This is the only place that
  decides which namespace the line is in**, and the four existing
  `hist.Prefix(e.WalkBase())` call sites in `replraw.go` collapse into it.
- **historyWindow** — `(now time.Time, days int) time.Time`. Local midnight,
  `days-1` days back. `days<1` clamps to 1. **Tested with a fixed non-UTC zone
  and across a DST boundary**, because that is the whole point of the function.
- **parseHistoryArgs** — `[]string` → `days int, err error`. Accepts `7`,
  `--days 7`, `--days=7`; rejects zero, negatives and non-numbers with a message
  naming the operand. **Bounded above at `maxHistoryDays = 3650`** — `AddDate`
  silently normalises a year-overflowing date rather than erroring, so an
  unbounded `--days 999999999` would return a garbage instant and print an empty
  list that looks like "you have no history". A refusal that names the limit is
  the only honest failure here.
- **replKind gains `cmdCommand`** — `parseREPLLine` classifies a leading `/` as a
  command and carries the raw line through. This is the single convergence point
  for "what did the user just submit": putting the `/` test anywhere else means
  one loop disagrees with the other, which is the defect `#4` spent ten rounds
  paying for. Both `replLines` and `runEditor` must handle the new kind, and a
  test asserts each does.
- **summariseLookups** — `([]ReviewEvent, since time.Time) []historyRow`. Keeps
  `Kind == EventLookedUp && Found`, groups by `store.Key`, and per group records
  `firstAt = min(At)` over ALL events and `inWindow = any(At >= since)`. Emits
  only `inWindow` rows, sorted `firstAt` descending, ties broken by word so the
  order is total and the test is not flaky.
  - **Why `Found`:** a typo is not vocabulary. `#14`'s up-arrow recall
    deliberately includes typos; `/history` is the words-queried view and filters
    them out. One log, two readers — the split the `History` doc comment
    describes.
- **renderHistory** — rows → lines, given a width and a `now` for relative dates
  ("today", "yesterday", then the date). Pure so the formatting is table-tested.

### Integration points

| Name | Lives in | Kind | Status | Wraps |
|------|----------|------|--------|-------|
| `commandCtx` | `cmd/define/command.go` | INTEGRATION | new | the deps a command may touch |
| `runHistory` | `cmd/define/history_cmd.go` | INTEGRATION | new | `store.Store.Events` + stdout |
| `dispatchCommand` | `cmd/define/command.go` | INTEGRATION | new | the `commands` table |

- **commandCtx** — `{deck store.Store; clock store.Clock; stdout, stderr io.Writer; width int}`.
  - **`clock` does not exist yet and Task 4 adds it.** `storeDeps` currently
    carries `history`, `capture`, `deck`; the only clock in the process is
    constructed inline at `main.go:124` (`newStoreCapturer(st, store.SystemClock(), warn)`).
    Task 4 lifts it to a `clock` field on `storeDeps`, built once in `openStore`
    and passed to both the capturer and `commandCtx`, so "what time is it" has
    one source and a test can move it.
  Deliberately NOT the whole `deps`: a command should not be able to reach the
  dictionary or the player by accident.
  - **Injected into:** every `run`. `clock` is injected rather than
    `time.Now()`-at-the-callsite, so the window is testable at a fixed instant in
    a fixed zone.
- **runHistory** — `Events(time.Time{})` → `summariseLookups` → `renderHistory`.
  A nil deck (the `DEFINE_NO_CAPTURE` case) reports why. **The message is
  EXTRACTED, not copied**: `main.go:400-406` currently owns that two-branch
  conditional inside `forgetWord`; Task 7 lifts it to a shared
  `noDeckMessage(opt) string` that both callers use. "The same message in two
  places" is how the atlas contradictions in `#4` started.
- **dispatchCommand** — looks up the name, runs it, or reports `nearestCommands`.

**Test surface.** Every PURE row above gets a colocated table test that runs with
no store and no IO. `runHistory` is tested against `store.Mem` (production code,
per ARCH-MOCK) with a fixed clock. No new external dependency, so no new fake.

**Test strategy — one line per risky function**, stating what each test is FOR.
The enumerations inside the tasks are the input corpus; these are the reasons
(PQ-4/BR-1).

| function | what its test exists to catch |
|---|---|
| `parseCommandLine` | a `/` that is not in column one being treated as a command, and the reverse |
| `commandCompletions` | completing from the wrong namespace, and case policy drifting from `Suggestion`'s byte-prefix match |
| `nearestCommands` | a near-miss reported as "nothing close" — which is what a count-based inference gets wrong once the registry has one row |
| `completionsFor` | the editor asking the wrong namespace; the fixture history is stocked so a wrong answer is visibly wrong |
| `historyWindow` | a window computed in UTC or as `now-48h`, which drops today's evening lookups and mis-sizes a DST day |
| `parseHistoryArgs` | a `--days` value that `AddDate` normalises into a garbage instant instead of being refused |
| `summariseLookups` | membership and ordering coming from the same timestamp, when they are deliberately different facts |
| `renderHistory` | width and relative-date formatting regressions, held away from IO |
| dispatch in each loop | one entry mode disagreeing with another about what a line means (the PQ-2 class) |

---

## Non-goals

Stated so they are decisions rather than omissions discovered at review.

- **Tab accepts; Return submits what was typed.** `/his` + Return dispatches
  `"his"` and gets `nearestCommands` ("did you mean /history?"), it does NOT
  auto-run the unique match. `#14` established that contract for words — *"Enter
  submits only what was typed"* — and command mode diverging from it would make
  Return mean two different things on one line. Tab is how you accept.
- **No multi-command list UI.** `/` with an empty rest shows the completion
  candidates through the EXISTING suggestion mechanism; `/help` prints the table.
  Nothing new is drawn.
- **The line loop gets dispatch, not type-ahead.** Completion is a raw-terminal
  affordance; `echo /history | define` runs the command but never suggests.
- **`/history` shows found lookups only.** Typos remain in up-arrow recall and
  stay out of the words-queried view. One log, two readers.
- **`/` can never be a headword.** Accepted in the issue's Done-when.
- **No new store method.** `Events` already answers this; adding a windowed query
  would be a second way to ask one question.

## Chunk 2: Tasks

### M1 — the `/` namespace and dispatch

Complete and reviewable without the store: `/` opens command mode, type-ahead
narrows, unknown commands suggest, and `/help` lists the table. No clock, no
events, no IO beyond stdout.

#### Task 1: The command table and line parsing

**Files:** Create `cmd/define/command.go`, `cmd/define/command_test.go`

- [x] **Step 1: Write the failing tests** — `parseCommandLine` (`"/history"`,
      `"/history 7"`, `"/"`, `"  /history  "`, `"hot dog"`, `""`),
      `commandCompletions` (`""` → all, `"his"` → `["/history"]`, `"zzz"` → none),
      `nearestCommands` (`"histry"` → `["/history"]`, `"zzzz"` → all).
- [x] **Step 2: Run; expect FAIL** (undefined symbols).
- [x] **Step 3: Implement** `command`, the `commands` table, and the three pure
      functions.
- [x] **Step 4: Run; expect PASS.**
- [x] **Step 5: Commit.**

#### Task 2: `completionsFor` — the namespace switch

**Files:** Modify `cmd/define/command.go`, `cmd/define/replraw.go:73,87,114,133`

- [x] **Step 1: Write the failing test** — `"/his"` completes from commands,
      `"syc"` from history, `"/"` lists every command.
- [x] **Step 2: Run; expect FAIL.**
- [x] **Step 3: Implement** `completionsFor(line string, hist History) []string`
      and replace all four `hist.Prefix(e.WalkBase())` sites with it.
- [x] **Step 4: Run the full suite.** **Stop condition:** if any `#14` editor test
      needs editing, the design is wrong — `Apply` must not have to change.
- [x] **Step 5: Commit.**

#### Task 3: `cmdCommand` and dispatch in BOTH loops

**Files:** Modify `cmd/define/repl.go`, `cmd/define/replraw.go`, `cmd/define/command.go`

- [x] **Step 1: Write the failing tests** — `parseREPLLine("/history", …)` yields
      `cmdCommand` with the line intact; **`replLines` dispatches it** (the piped
      loop, via `echo /help | define`); **`runEditor` dispatches it** (via
      `scriptKeys("/help\r")`); an unknown `/histry` suggests and does NOT reach
      the dictionary — asserted by a rig whose dictionary fails the test if
      called.
- [x] **Step 2: Run; expect FAIL.**
- [x] **Step 3: Implement** `cmdCommand`, `dispatchCommand`, `commandCtx` (deck
      and clock nil at this milestone), and `/help`.
- [x] **Step 4: Run the full suite + `-race`.**
- [x] **Step 5: `sdlc milestone-close --issue 15 --milestone M1`.**

### M2 — `/history`

#### Task 4: One clock, on `storeDeps`

**Files:** Modify `cmd/define/main.go`, `cmd/define/capture_test.go`

- [ ] **Step 1: Write the failing test** — `openStore` returns a `storeDeps`
      whose `clock` is non-nil, and a test-supplied clock reaches both the
      capturer and `commandCtx`.
- [ ] **Step 2: Run; expect FAIL.**
- [ ] **Step 3: Implement** — add `clock store.Clock` to `storeDeps`, build it
      once in `openStore`, and delete the inline `store.SystemClock()` at
      `main.go:124` so there is one source.
- [ ] **Step 4: Run; expect PASS.** — [ ] **Step 5: Commit.**

#### Task 5: The local-time window

**Files:** Create `cmd/define/history_cmd.go`, `cmd/define/history_cmd_test.go`

- [ ] **Step 1: Write the failing tests.** This is the task the operator flagged,
      so the tests ARE the specification:
      - `now = 2026-08-21T00:30:00-07:00`, `days=2` → `2026-08-20T00:00:00-07:00`
        — **not** `2026-08-19T00:30:00Z`;
      - **the evening case**: `now = 2026-08-21T20:00-07:00`, a lookup ten
        minutes earlier at `2026-08-21T19:50-07:00`, which is stored in
        `events/`**`2026-08-22`**`.yaml`, is inside the window. A filter over the
        local days `[08-20, 08-21]` never opens that file, so a lookup from ten
        minutes ago vanishes and `/history` reads "nothing today";
      - across the US DST boundary the window spans 23/25 local hours, not 24 —
        `2026-03-08` is 23 h and `2026-11-01` is 25 h, and `AddDate(0,0,-1)` from
        `2026-03-09` midnight lands on `2026-03-08T00:00-08:00`, changing offset
        (all three measured in this toolchain, not assumed);
      - `days=0`, `days=-3` clamp to 1; `days=999999999` is REFUSED naming the
        3650 limit, not silently normalised by `AddDate`.
- [ ] **Step 2: Run; expect FAIL.**
- [ ] **Step 3: Implement** `historyWindow`, `parseHistoryArgs`, `maxHistoryDays`.
- [ ] **Step 4: Run; expect PASS.** — [ ] **Step 5: Commit.**

#### Task 6: `summariseLookups`

**Files:** Modify `cmd/define/history_cmd.go`, `cmd/define/history_cmd_test.go`

- [ ] **Step 1: Write the failing tests** — dedupe by key (`"Sycophantic"` and
      `"sycophantic"` are one row); a word first seen BEFORE the window but
      re-queried inside it appears, ordered by its **first-ever** time; a word
      only seen before the window does not appear; `Found:false` never appears;
      ordering is `firstAt` desc with a total tiebreak.
- [ ] **Step 2: Run; expect FAIL.** — [ ] **Step 3: Implement.**
- [ ] **Step 4: Run; expect PASS.** — [ ] **Step 5: Commit.**

#### Task 7: `runHistory`, end to end

**Files:** Modify `cmd/define/history_cmd.go`, `cmd/define/main.go`, `cmd/define/command.go`

- [ ] **Step 1: Write the failing tests** — against `store.Mem` with a fixed
      clock: `/history` prints the expected rows; `/history 7` widens the window;
      `/history zzz` names the bad operand and exits non-zero; a nil deck prints
      the `DEFINE_NO_CAPTURE` message **through the shared helper**, asserted by
      `forgetWord` and `runHistory` producing the identical string.
- [ ] **Step 2: Run; expect FAIL.**
- [ ] **Step 3: Implement** `runHistory`, extract `noDeckMessage(opt)` from
      `forgetWord` (`main.go:400-406`), and register the row.
- [ ] **Step 4: Run the full suite + `-race`.** — [ ] **Step 5: Commit.**

#### Task 8: Docs and the atlas

- [ ] **Step 1:** `atlas/define.md` gains `## Command mode`: the `/` namespace,
      the match-source switch, `cmdCommand` as the shared dispatch point, the
      command table, and — **stated once** — the local-time rule with its reason.
      Add `/history` to the entry-modes table.
- [ ] **Step 2:** README: what `/history` shows, what it omits and why, `--days`.
- [ ] **Step 3:** `--help`: one line that `/` opens command mode.
- [ ] **Step 4:** `sdlc milestone-close --issue 15 --milestone M2`, then `sdlc close`.

---

## Verification

- Full suite + `-race` green; `go vet`; `GOOS=linux CGO_ENABLED=0 go build ./...`.
- **Mutation-checked (applied + compiled + `-count=1`):** replacing
  `historyWindow`'s local-midnight arithmetic with `now.Add(-48*time.Hour)` must
  redden the evening-lookup test. If it does not, the test is not pinning the
  thing this issue is about.
- **Mutation-checked:** deleting the `cmdCommand` case from `replLines` must
  redden a test. PQ-2 was exactly this gap in the first draft.
- Manual: `define` in a directory with a real deck — `/his<Tab>`, `/history`,
  `/history 7`, `/histry`, and `echo /history | define`.

## Risks

- **The editor tests are the regression net for Task 2.** If a `#14` test needs
  editing, stop and re-think rather than adjusting the test.
- **`Events` is O(all history) per call.** One read per `/history` on a personal
  word list, so it is the right trade today. When it stops being, the fix belongs
  in the store — an index, or a filename pre-filter that still *decides* on
  timestamps — not in this feature.

## Revisions

### 2026-08-21 — plan-quality gate, round 1 (3 Important, 3 Minor)

- **PQ-2 was the serious one, and it is `#4`'s defect in design form.** The draft
  put dispatch in `submitLine`, which only the raw editor reaches — `echo
  /history | define` would have gone to the dictionary. The fix is the seam that
  already exists: `parseREPLLine` is the shared classifier both loops route
  through, with an ARCH-DRY comment on the raw path saying so. Dispatch is now a
  `cmdCommand` kind there, and both loops are tested for it.
- **PQ-1:** `commandCtx.clock` named a dependency nothing provided. Task 4 adds
  `clock` to `storeDeps` and removes the inline `store.SystemClock()`.
- **PQ-3:** `parseHistoryArgs` was unbounded; `AddDate` normalises a
  year-overflowing date instead of failing, so a huge `--days` would have printed
  an empty list that reads as "no history". Bounded at 3650 with a named refusal.
- **Minors:** the Return-on-unique-prefix ambiguity is now an explicit non-goal
  (Tab accepts, Return submits what was typed — `#14`'s contract); the nil-deck
  message is extracted rather than copied; a Non-goals section exists.


### 2026-08-21 — M1 boundary review, and a corrected test spec

**The M1 review found five Important, and BR-3 was mine.**
`TestRawEditorDispatchesCommands` asserted that stdout contained `"/help"` — but
the raw editor ECHOES the submitted line, so the assertion matched the echo and
passed with dispatch deleted from `runEditor` (reproduced: mutation applied,
`BUILD_OK`, `ok`). Same shape as `#1`'s circular oracle: an observable that two
paths both produce. Both loop tests now assert on `"list the commands"`, which
only `runHelp` can emit, and the mutation reddens them.

BR-2 was the matching gap one level out: `completionsFor` returning the right
answer is not evidence the editor ASKS it. `TestEditorSuggestsFromCommands` types
`/hel` against a history stocked with `hibernate` and requires the grey tail to
come from the command set — mutation-verified by making `completionsFor` ignore
commands. BR-6 (`commandCtx` built at two call sites, with M2 about to add two
fields) became `newCommandCtx`. BR-4 put the `/` surface into `--help` and the
README, which M1 shipped without. BR-5 was fifteen unticked step boxes.

**Separately, Task 5's central test case was wrong and is corrected above.** It
claimed a `2026-08-20T19:00-07:00` lookup sits inside a `days=1` window; it does
not — that is yesterday in local terms. Worse, the case did not demonstrate the
bug it was written for: a filename filter over local days `[08-20, 08-21]` reads
`2026-08-21.yaml` anyway. The case that does break is TODAY's evening: a lookup
at `19:50` local is written to TOMORROW's UTC-named file, so a filename filter
drops a lookup from ten minutes ago. Measured before writing the test rather than
after it failed.

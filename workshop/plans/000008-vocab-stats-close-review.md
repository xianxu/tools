# Boundary Review — tools#8 (whole-issue close)

| field | value |
|-------|-------|
| issue | 8 — define --stats: deck, streak and mastery statistics |
| repo | tools |
| issue file | workshop/issues/000008-vocab-stats.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6cc7513b4ee056108f6900d482d73e0fee5def7b..ff962e52453079e77cabecec078b30bc782163a1 |
| command | sdlc close --issue 8 |
| reviewer | claude |
| timestamp | 2026-09-07T23:56:55-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fold is the strongest part of this diff and it holds up under attack: I mutated each of the four load-bearing decisions in a scratch copy of HEAD and every one reddened the test that claims to pin it, by name. The screen, the exit codes, the `/stats` door and the docs sync are all real and green (`go test ./...`, `go vet` under default/pty/conformance, `gofmt -l` all clean; both `--stats` paths smoke-run to the documented exit codes). What blocks SHIP is not the shipped behaviour — it is that the one thing this issue set out to *fix about its own guards* did not land: removing `{"-stats", *statsFlag}` from `run()`'s `modes` slice leaves the **entire** `cmd/define` suite green, which is the exact mutation the plan's Task 5 Step 2b promised to perform and watch fail. Alongside that, the plan's Core-concepts table names an entity the code never declares and states an exit code the code contradicts — and the repo's own guard for that is silenced because the plan's steps are still unticked.

## 1. Strengths

- **The DST/zone reasoning is genuinely pinned, not asserted.** Swapping `AddDate` for `Add(-24*time.Hour)` in `streaks` (`cmd/define/schedule/stats.go:203`) reddens *both* `TestStreaksAcrossDSTBoundaries` subtests by name; dropping `at.In(now.Location())` at `schedule/stats.go:127` reddens `TestDaysAreCountedInTheLearnersZone` with both its assertions. The `*Location`-pointer-in-map-key hazard is a real bug that a normal reviewer would miss, and it is caught at the seam where the keys are built.
- **The ARCH-SECURE skips are reachable.** Deleting the two-line guard inside `countable` (`schedule/stats.go:177`) reddens both `TestBadTimestampsAreSkippedNotBelieved` subtests, including the accuracy assertion. Skipped-not-clamped is the right call and the comment says why.
- **`TestEveryStatsFieldIsRendered` is a real derived guard.** I added a field to `Stats` and it failed with the intended message (`Stats gained the field "ScratchNewField" and this guard has no expectation for it`). The `notShown` escape hatch for `LastDay` is named with a reason, which is the right shape.
- **`TestMasteredAgreesWithTheScheduleFunction`** (`schedule/stats_test.go:78`) asserts agreement *and* checks the fixture actually distinguishes mastered from lapsed, so it cannot degenerate into comparing two zeros. That is the failure mode of most "agreement" tests.
- **The `commands` registry row is pinned.** Removing `{name: "stats", …}` from `command.go:27` reddens `TestDocsQuoteTheCommandList` with the exact atlas span — so the atlas/README command list genuinely derives.
- **`declaredModes` really does parse `main.go`'s `[]mode{…}` literal** (`harvest_test.go:795`). The derivation half of Task 5 Step 2b landed; it is the fail-closed half that did not (finding I-1).

## 2. Critical findings

None. Nothing in the shipped binary is wrong. I verified the mode guard fires in the tree as committed: `-stats -play`, `-stats -harvest` and `-stats -reflect` each exit 2 with `… are both modes; run them separately`.

## 3. Important findings

**I-1 — the `-stats` mode registration is unpinned; the mutation the plan promised does not redden.**
`cmd/define/main.go:609`, `cmd/define/harvest_test.go:837` and `:677`.
I deleted `{"-stats", *statsFlag}` from the `modes` slice and ran the full `cmd/define` package (109s, git-initialised scratch so the repo guards ran): **everything passed.** Two causes, both the same rule the issue was written to fix:
- `declaredModes`'s fail-closed floor is `if len(names) < 5` while `run()` now declares **six**, so an under-derivation of exactly one is invisible. The floor is itself a hand-maintained extent.
- `TestRunRefusesTwoModes` (`harvest_test.go:677`) hand-lists its arg pairs and `-stats` is not among them; the plan explicitly asked for "one that `-stats -play` collides", and it was never written.

Without the slice entry, `--stats --play` silently runs stats and drops `--play` (dispatch reaches `if *statsFlag` at `main.go:732` first) — "a dropped mode is a silently different command", which is `TestRunRefusesTwoModes`'s own words.
*Fix:* raise the floor to `< 6` **and** add `{"-stats", "-play"}` to `TestRunRefusesTwoModes`'s table. The floor alone catches the removal; the pair row alone catches it too, and neither is redundant — the floor guards the derivation, the row guards `run()`.

**I-2 — the plan's Core-concepts table contradicts the code in two places.**
`workshop/plans/000008-vocab-stats-plan.md:82` and `:140`.
- Row `| `activeDays` | `cmd/define/schedule/stats.go` | new |` — **no such symbol exists anywhere in the tree.** The figure is delivered by an inline `days` map inside `Summarise` plus `streaks`. The two helpers actually written, `countable` (`schedule/stats.go:177`) and `streaks` (`:203`), appear in no row.
- Plan line 140: *"A nil deck is 0 with the explanatory sentence — it is a statement about the directory, not a failure."* The code returns **1** (`cmd/define/stats.go:43`), and the issue's Revisions records the correction — but the plan still teaches the wrong exit code to anyone who reads it.
- Related: the ARCH-DRY table's `DaysBetween` row says "uses it"; `streaks` deliberately does not call it, and the code comment argues why. The table should say so.

The review contract rates a table/code contradiction Critical; I am calling it Important because nothing in the shipped binary diverges and `Stats.ActiveDays` does exist as a figure. It still blocks until disposed.

**I-3 — the plan's unticked steps suppress the repo's own guard for I-2.**
`cmd/define/repo_guard_test.go:797` — `if inProgress && status == "new" { continue }`, where `inProgress` is `strings.Contains(body, "- [ ] ")`. Every task step in the plan is still `- [ ]` at HEAD even though the work is complete and the issue's `## Plan` is ticked. I ticked them in a scratch copy and `TestPlanTablesNameEntitiesThatExist` failed immediately with exactly the right message:
> `000008-vocab-stats-plan.md names "activeDays" at cmd/define/schedule/stats.go, which does not declare it`

So the guard designed to fire at precisely this boundary is disabled by a stale checkbox. This is the generalisable half: **tick the plan's steps as part of crossing the boundary, before the review runs**, or the `new`-row check is permanently off for every issue.

**I-4 — ARCH-DRY: the two `/stats` doors duplicate the read path, and the comment claims otherwise.**
`cmd/define/stats.go:36` (`runStats`) and `:198` (`runStatsCommand`). The file says *"everything below that seam is shared"*, but only `renderStats(schedule.Summarise(…))` is — the `Deck()` read, the `Events(time.Time{})` read, the clock read and the print loop are copies, with divergent error phrasing (`could not read the deck` / `could not read the log` vs. a single `define: /stats: %v`). The plan names `--stats --days 30` as the next extension; a windowed read lands in one door only. `TestSlashStatsAndTheFlagPrintTheSameScreen` pins the *output* for one fixture, which would not catch a divergent `since` argument unless the fixture straddled the window — so the pin is weaker than the duplication it guards.
*Fix:* `func statsLines(deck store.Store, clk store.Clock) ([]string, error)`. Both `deps` (via `langDeps`) and `commandCtx` already carry `store.Store` + `store.Clock`, so the extraction is mechanical; each door keeps its own error phrasing around the returned error.

## 4. Minor findings

- **M-1** `cmd/define/README.md:407` — the sample screen prints `since   3 January`; `relativeDay` (`history_cmd.go:207`) formats `Jan 2`, so the real screen reads `since   Jan 3`. I rendered the README's exact `Stats` values: every other line matches byte-for-byte, so this is a one-word fix.
- **M-2** `cmd/define/schedule/stats.go:85` — `Fold(events)` runs on the **unfiltered** slice while every calendar and accuracy figure goes through `countable`. A hand-edited zero-`At` or future-dated `reviewed` event is excluded from `Accuracy`/`ActiveDays` but still promotes a word toward `Mastered`. This is very likely the *right* choice (filtering would make the screen disagree with `--play`'s queue — the drift `Mastered`'s own doc warns about), but the doc comment says "the fold VALIDATES rather than trusts" without naming the exception, and nothing pins it. Add the sentence and a row.
- **M-3** `cmd/define/README.md:521` — "One mode at a time. `-harvest`, `--play`, `--reflect`, `--stats`, `-forget` and `--llm-check`" is a hand-maintained restatement of `run()`'s `modes` slice, the same family `declaredModes` just fixed for the test. `declaredModes` now exists; a `doc_sync_test.go`-shaped guard over it is nearly free, and the repo has fourteen of that shape already.
- **M-4** `cmd/define/stats.go:36` — `ctx context.Context` is unused in `runStats`. Deliberate (it mirrors `runReflect`), but worth `_ context.Context` or one line saying so.

## 5. Test coverage notes

- The single gap is I-1. Every other behavioural claim in the diff is pinned by a test I confirmed reddens when the behaviour is removed.
- Nothing drives `/stats` through `dispatchCommand` with the **live** `commands` registry — `TestSlashStatsAndTheFlagPrintTheSameScreen` calls `runStatsCommand` directly. The registry row is covered indirectly by `TestDocsQuoteTheCommandList` (verified reddens), so this is a note rather than a finding.
- `TestStatsOnAnEmptyDeckSaysSoRatherThanPrintingZeros` asserts on the substring `" 0"`, which is a proxy for "no figures". It happens to be adequate (it would catch `streak 0 days`), but it reads stronger than it is.

## 6. Architectural notes

- **ARCH-DRY — flag** (I-4, M-3). The `Mastered`/`Fold`/`StartOfDay` reuse discipline is otherwise exemplary: the plan's "what this issue is NOT" table did its job and no figure is computed twice.
- **ARCH-PURE — pass.** `Summarise`, `renderStats`, `streakPhrase`, `formLabel` are all pure; `now` as a parameter rather than an injected clock makes every DST and fractional-offset case a table row with no double to keep honest. `TestSchedulePurity` enforces the package boundary mechanically. This deviation from the Done-when's "fake clock" is strictly stronger and is recorded in the issue's Revisions.
- **ARCH-PURPOSE — flag** (I-1). The finding named one instance (`TestModeCollision` hand-lists five modes); the *class* is "an extent restated by hand rather than derived from the code that owns it". The instance was fixed, the class was not — the fail-closed floor and `TestRunRefusesTwoModes`'s pair list are two more members sitting in the same file, and the README's mode list is a third.
- **ARCH-MOCK — pass.** No new external dependency. `store.Mem` behind the existing conformance-tested seam, `store.FixedClock` for the end-to-end rows; production and test flow share the same boundary.
- **ARCH-CONSTRAINTS — pass.** O(events + deck), one day map and one form map. The longest-run walk is amortised O(n) because only run-starts iterate forward. Both store implementations already read the whole `events/` directory regardless of `since`, so `Events(time.Time{})` costs nothing extra. No cache, no counter — and the plan states the escape hatch (a window, never a counter) if the log ever outgrows one fold.
- **ARCH-SECURE — pass with M-2.** Untrusted timestamps skipped rather than clamped, `Form` control runes dropped at the render boundary, unknown form names shown rather than filtered; all three pinned and mutation-verified. The one unexamined path is `Fold`'s.
- **ARCH-ORDER — pass.** `Summarise` carries no state between events; it is a single-shot fold over a snapshot the seam promises is chronological (`store/store.go:18`, both implementations sort). No cancellation path, no spawned work, no concurrency. The plan writes this out as "holds no state because X" rather than a bare `N/A`, which is what the principle asks for.

## 7. Plan revision recommendations

One `## Revisions` entry on `workshop/plans/000008-vocab-stats-plan.md`, dated, covering all four (AGENTS.md §1 — the `TestStreaksInAHalfHourOffsetZone` → `TestStreaksInFractionalOffsetZones` rename was overwritten in place rather than appended, which is the convention this entry also repairs):

1. `activeDays` never became a symbol — the day set is an inline map in `Summarise`; the entities actually written are `countable` and `streaks`. Replace the row with those two, both `new`, both at `cmd/define/schedule/stats.go`.
2. Nil deck is exit **1** on stderr, not 0 — every sibling (`--forget`, `--harvest`, `--reflect`, `/history`) is unanimous. The empty deck remains 0.
3. `streaks` does **not** call `store.DaysBetween`; it applies DaysBetween's *rule* (b's location defines the calendar) when the keys are built, then walks with `AddDate`. Amend the ARCH-DRY reuse table row.
4. Test rename `TestStreaksInAHalfHourOffsetZone` → `TestStreaksInFractionalOffsetZones` (Kolkata +05:30 and Kathmandu +05:45).

And tick the plan's `- [ ]` task steps as part of closing, per I-3.

```findings
findings:
  - id: new
    severity: Important
    family: hand-maintained-extent
    title: |
      Removing -stats from run()'s modes slice leaves the whole cmd/define suite green
    detail: |
      Verified by mutation on a git-initialised scratch copy of HEAD: deleting
      {"-stats", *statsFlag} from main.go:609 passes the full package (109s), so the
      registration the plan's Task 5 Step 2b promised to mutation-check is unpinned.
      declaredModes' floor is `len(names) < 5` (harvest_test.go:837) while run() now
      declares six, and TestRunRefusesTwoModes (harvest_test.go:677) hand-lists pairs
      that omit -stats. Raise the floor to 6 and add a {"-stats","-play"} row. Shipped
      behaviour is correct — -stats -play exits 2 in the tree as committed.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      Plan names an entity the code never declares and states the wrong nil-deck exit code
    detail: |
      workshop/plans/000008-vocab-stats-plan.md:82 lists `activeDays` as new at
      cmd/define/schedule/stats.go; no such symbol exists in the tree, and the helpers
      actually written (countable, streaks) appear in no row. Line 140 says a nil deck
      exits 0; cmd/define/stats.go:43 returns 1. The ARCH-DRY table also claims streaks
      uses store.DaysBetween, which it deliberately does not.
  - id: new
    severity: Important
    family: plan-table-drift
    title: |
      Unticked plan steps suppress TestPlanTablesNameEntitiesThatExist at the boundary
    detail: |
      repo_guard_test.go:797 skips `new` rows while the plan contains "- [ ] ". Every
      task step is still unticked at HEAD though the work is done, so the guard that
      exists to catch the finding above is off at exactly the gate it was written for.
      Ticking them in a scratch copy makes it fail naming "activeDays" verbatim. Tick
      the plan's steps as part of crossing the boundary.
  - id: new
    severity: Important
    family: duplicated-read-path
    title: |
      runStats and runStatsCommand duplicate the deck/log/clock read the comment says is shared
    detail: |
      cmd/define/stats.go:36 and :198 share only renderStats(Summarise(...)); the Deck()
      read, Events(time.Time{}) read, clock read and print loop are copies with divergent
      error phrasing. The plan names --stats --days 30 as the next extension, which would
      land in one door only, and the same-output test uses one fixture that a divergent
      `since` would not necessarily separate. Extract
      statsLines(deck store.Store, clk store.Clock) ([]string, error) — both deps and
      commandCtx already carry those two types (ARCH-DRY).
  - id: new
    severity: Minor
    family: docs-restate-unverified-output
    title: |
      README's sample screen prints "3 January"; the code prints "Jan 3"
    detail: |
      cmd/define/README.md:407. relativeDay (history_cmd.go:207) formats "Jan 2". Verified
      by rendering the README's exact Stats values — every other line matches byte-for-byte.
  - id: new
    severity: Minor
    family: untrusted-persisted-input
    title: |
      Fold runs on unfiltered events while every other figure goes through countable
    detail: |
      cmd/define/schedule/stats.go:85. A zero-At or future-dated hand-edited reviewed event
      is excluded from Accuracy and ActiveDays but still promotes a word toward Mastered.
      This is probably correct — filtering would make the screen disagree with --play's
      queue — but the doc comment claims the fold validates rather than trusts, without
      naming the exception, and nothing pins it.
  - id: new
    severity: Minor
    family: hand-maintained-extent
    title: |
      README's mode list is a hand-maintained restatement of run()'s modes slice
    detail: |
      cmd/define/README.md:521. Same family declaredModes just fixed for the test. A
      doc_sync_test.go-shaped guard reusing declaredModes is nearly free.
  - id: new
    severity: Minor
    family: unused-parameter
    title: |
      runStats takes a context.Context it never uses
    detail: |
      cmd/define/stats.go:36. Deliberate — it mirrors runReflect's signature — but worth
      `_ context.Context` or one line saying why.
```

---

## Re-review — 2026-09-08T00:06:24-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 8 — define --stats: deck, streak and mastery statistics |
| repo | tools |
| issue file | workshop/issues/000008-vocab-stats.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6cc7513b4ee056108f6900d482d73e0fee5def7b..ff962e52453079e77cabecec078b30bc782163a1 |
| command | sdlc close --issue 8 |
| reviewer | claude |
| timestamp | 2026-09-08T00:06:24-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

An independent pass over `6cc7513..ff962e5` reproduces every finding round 1 recorded (BR-1…BR-8, all still open at HEAD) and adds one the first round missed: **`streaks` computes the wrong number in any timezone whose DST transition falls at local midnight** — Havana, Beirut, Santiago among them — which is exactly the property the issue's second Done-when row is ticked for. The fold, the deck-vs-log asymmetry, the mastery agreement pin and the derived render guard are all genuinely good work, the suite is green at HEAD and `gofmt`/`go vet` are clean; what blocks the boundary is a demonstrated wrong figure in the screen's headline number, on top of four open Important findings from round 1. (The working tree currently carries *uncommitted* remediation for BR-1/BR-2/BR-3/BR-4 — a new `TestEveryDispatchedModeIsInTheCollisionList`, a `printStats` extraction, and a rewritten plan table. That is outside the pinned range and is not reviewed here.)

## 1. Strengths

- **The deck/log asymmetry is designed, documented and pinned.** `schedule/stats.go:70-95` states why `Known` comes from the deck and `AddedPerDay` from the log, and `TestKnownCountsTheDeckNotTheLog` (`schedule/stats_test.go:53`) asserts both halves in one fixture. This was the most likely wrong number on the screen and it is the one most carefully defended.
- **The mastery pin cannot pass vacuously.** `TestMasteredAgreesWithTheScheduleFunction` computes the reference the way `--play` does *and* asserts the fixture actually distinguishes a lapsed word from a mastered one (`stats_test.go:105-110`). A test that checks its own oracle is rare and correct here.
- **`TestDaysAreCountedInTheLearnersZone`** (`schedule/stats_test.go:222`) catches the `time.Time`-equality-includes-`*Location` trap. That is a real bug found before review and pinned with a test that names the mechanism.
- **`TestEveryStatsFieldIsRendered`** derives the field set by reflection, fails loudly on an unrecognised field, and names `LastDay` as a deliberate omission rather than an oversight — the `#12` BR-17 correction applied properly.
- **The atlas `/stats` row is generated, not remembered** — `TestDocsQuoteTheCommandList` owns that table, so the doc row is derived by construction.
- **The nil-deck correction** (`stats.go:27-33`) is the right call and is recorded honestly: stderr + exit 1 through `noDeckMessage`, matching all four siblings, pinned per-cause.

## 2. Critical findings

**C1 — `streaks` breaks a run across any DST transition that happens at local midnight (`cmd/define/schedule/stats.go:203-236`).**

The day set is keyed by `store.StartOfDay(at.In(now.Location()))` (`stats.go:127`). In a zone where local midnight does not exist on day *D*, `time.Date(y,m,D,0,0,0,0,loc)` normalises to **01:00**, so that day's key carries hour = 1. `streaks` walks with `AddDate`, which preserves the wall-clock hour — so from a 01:00 key it lands on 01:00 of the neighbour and misses its 00:00 key, and from a 00:00 key it lands on 00:00 of a day whose key is 01:00. The walk and the key construction disagree.

Reproduced twice against the committed `schedule.Summarise` (three consecutive days of events at 20:00 local, `now` = the following day 22:00):

```
America/Havana     midnight-transition 2026-03-08  ActiveDays=3 Current=2 Longest=2  (want 3/3/3)
Asia/Beirut        midnight-transition 2026-03-29  ActiveDays=3 Current=2 Longest=2  (want 3/3/3)
America/Santiago   midnight-transition 2026-09-06  ActiveDays=3 Current=2 Longest=2  (want 3/3/3)
America/New_York   no midnight transition in 2026
```

`ActiveDays` is correct; only the two streaks are wrong, and not transiently — a learner in Havana with a 100-day run spanning 2026-03-08 reads "streak 20 days" from then on, and `LongestStreak` is permanently truncated. The existing table covers New York (transition at 02:00) and two non-DST fractional-offset zones, so no row can see this. The `Done when` row *"Streak arithmetic verified across timezone boundaries and gaps"* is ticked over it.

Fix — minimal: normalise every stepped value, `store.StartOfDay(d.AddDate(0,0,-1))` at `:210` and `:225`, `store.StartOfDay(c.AddDate(0,0,1))` at `:229`. Better (ARCH-DRY / ARCH-SECURE): key the day set by a civil date (`struct{y int; m time.Month; d int}`) or by the UTC day index `store.DaysBetween` already computes internally — that makes the `*Location`-pointer hazard *and* the nonexistent-midnight hazard unrepresentable at once, and restores the plan's ARCH-DRY table claim that this code derives days the way `store` does. Add a regression row to `TestStreaksAcrossDSTBoundaries` using `America/Havana` 2026-03-08 (or `Asia/Beirut` 2026-03-29), and confirm it reddens before the fix.

## 3. Important findings

BR-1, BR-2, BR-3 and BR-4 are open and independently confirmed against HEAD; I am not re-raising them under new ids. Two notes from re-deriving them:

- **BR-1** re-measured cleanly this round: on a scratch copy of `ff962e5` with `{"-stats", *statsFlag}` deleted from `main.go:609`, the full `cmd/define` package is green — the only failures are the git-dependent repo guards, which fail for lack of a `.git`, not for the mutation. Both trees were verified byte-wise before the run.
- **BR-3**'s mechanism is confirmed: ticking the plan's steps in a scratch copy makes `TestPlanTablesNameEntitiesThatExist` fail with *`000008-vocab-stats-plan.md names "activeDays" at cmd/define/schedule/stats.go, which does not declare it`*. The unticked plan is what switched off the guard written for exactly this gate.

## 4. Minor findings

- **N2 —** `formLabel` sanitises at render (`stats.go:174`), so two distinct raw `Form` keys that differ only in control runes collapse to one label and print as two identical rows. Cosmetic, but it is the one place the ARCH-SECURE neutralisation is observable as a wrong screen rather than a safe one.
- BR-5 (README "3 January" vs the rendered "Jan 3"), BR-6 (`Fold` at `stats.go:85` runs on unfiltered events while every other figure goes through `countable`), BR-7 (README's mode list is a hand-maintained restatement of `run()`'s slice), BR-8 (unused `ctx`) remain open as recorded.
- The window also carries `#47`/`#48` tracker commits (`404215c`, `5ea5692`, `c99bbcc`, `bd33e96`, `790bc1f`). Not a defect — noted so the close's evidence line isn't read as `#8` having touched `workshop/projects/define-learn.md` for its own reasons.

## 5. Test coverage notes

- The streak table has no zone whose DST transition is at midnight — the blind spot C1 lives in. Everything else in that table (23h/25h days, +05:30, +05:45, mixed-offset events) is well chosen.
- Nothing reddens when `-stats` loses its mode registration (BR-1), measured.
- `TestSlashStatsAndTheFlagPrintTheSameScreen` pins one fixture on the happy path; the two doors' error paths are not compared, and their wording already differs at HEAD (BR-4).
- The plan's Verification item 3 claims the DST rows were mutation-swept against a `Sub()/24h` implementation. I did not re-run that sweep; the rows are meaningful for New York regardless.
- Everything else: `go test ./cmd/define/...` green at HEAD (110s), `gofmt -l` clean, `go vet` clean.

## 6. Architecture

- **ARCH-DRY — flag.** BR-4 (the two doors duplicate the read path under a comment asserting they don't). Also C1's root cause: `streaks` re-derives day stepping instead of using the day-index arithmetic `store.DaysBetween` already owns, and the zone that arithmetic was written to survive is the one that breaks.
- **ARCH-PURE — pass.** `Summarise`/`renderStats` are pure with `now` as a parameter; purity is enforced mechanically by `TestSchedulePurity`, not promised. The `now`-as-parameter deviation from the Done-when's "fake clock" is stronger than what was asked and is recorded in the issue.
- **ARCH-PURPOSE — flag.** BR-1 is the class/instance failure: the issue correctly identified "a hand-maintained extent is half a guard", fixed `TestModeCollision`'s set, and left the enumerable sibling `TestRunRefusesTwoModes` (`harvest_test.go:677`) hand-listed with no `-stats` row — and `declaredModes`' floor at `5` against six declared modes.
- **ARCH-MOCK — pass.** `store.Mem` behind the same seam production uses; no new external dependency, nothing to conform-check.
- **ARCH-CONSTRAINTS — pass.** O(events + deck) with the envelope stated and the "if it grows, window it, never a counter" rule written down. `streaks`' longest-run loop is O(days) overall, not quadratic. `/stats` re-reads the whole log per invocation inside the REPL; fine at the stated scale.
- **ARCH-SECURE — mostly pass, residue noted.** Zero-`At` and future-`At` skips are stated and pinned; control runes in `Form` are dropped and pinned; skipping rather than clamping is the right choice and is argued. Residue: BR-6 (`Fold` bypasses `countable`) and N2.
- **ARCH-ORDER — pass.** A pure fold over a snapshot; the `N/A` is written as a claim ("holds no state between calls… the log IS chronological", with the seam cited) rather than as a bare marker, and the min/max-over-timestamps choice is justified on a correct basis.

## 7. Plan revision recommendations

BR-2/BR-3 already specify most of this; the additions from this round:

- **`## Revisions` — the day set is not `store.DaysBetween`'s arithmetic, and that is now a defect, not just a deviation.** The ARCH-DRY table (`plan:56`) says this issue *"uses"* `store.DaysBetween`; `streaks` deliberately does not, and C1 is what that costs. Record the deviation, and if C1 is fixed by day-index or civil-date keying, update the row to describe what shipped.
- **`## Revisions` — the ARCH-SECURE section's claim that "the fold VALIDATES rather than trusts" needs its exception named**: `Fold` (`stats.go:85`) sees unfiltered events, so `Known`/`Mastered` are deliberately outside the skips in order to agree with `--play`'s queue (BR-6).
- **Verification section:** item 3 should name the midnight-transition zone row alongside the 23h/25h and fractional-offset ones, so the Done-when's timezone claim is backed by a test that covers the case it currently misses.

```findings
findings:
  - id: new
    severity: Critical
    family: local-calendar-arithmetic
    title: |
      streaks breaks a run across any DST transition that occurs at local midnight
    detail: |
      cmd/define/schedule/stats.go:203-236. Day keys are store.StartOfDay(...) values, which
      normalise to 01:00 in zones where local midnight does not exist that day; streaks steps
      with AddDate, which preserves the wall-clock hour, so the walk misses the neighbouring
      day's 00:00 key. Reproduced twice against the committed Summarise: America/Havana
      2026-03-08, Asia/Beirut 2026-03-29 and America/Santiago 2026-09-06 all return
      ActiveDays=3 with CurrentStreak=2 and LongestStreak=2 where 3/3 is correct. Not
      transient — a run spanning the transition stays truncated afterwards. The Done-when row
      "streak arithmetic verified across timezone boundaries" is ticked over it, and the test
      table covers only New York (02:00 transition) and non-DST fractional-offset zones. Fix:
      wrap every stepped value in store.StartOfDay (:210, :225, :229), or better key the day
      set by a civil date / UTC day index so neither this nor the *Location-pointer hazard is
      representable; add a midnight-transition row to TestStreaksAcrossDSTBoundaries and
      confirm it reddens first.
  - id: new
    severity: Minor
    family: untrusted-persisted-input
    title: |
      formLabel sanitises at render, so two distinct form keys can print as identical rows
    detail: |
      cmd/define/stats.go:174. Accuracy is keyed by the raw Form string, and control runes are
      stripped only when the label is drawn, so "meaning" and "meaning\x1b[2J" produce two rows
      that read the same. Neutralising is right; collapsing at the key or disambiguating the
      label would keep the screen honest about what the log actually holds.
```

---

## Re-review — 2026-09-08T00:21:13-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 8 — define --stats: deck, streak and mastery statistics |
| repo | tools |
| issue file | workshop/issues/000008-vocab-stats.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6cc7513b4ee056108f6900d482d73e0fee5def7b..15b94c39397ab5654396d07487fa6ea4663371bf |
| command | sdlc close --issue 8 |
| reviewer | claude |
| timestamp | 2026-09-08T00:21:13-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The round-1 findings that were claimed fixed genuinely are — `TestEveryDispatchedModeIsInTheCollisionList` fails on mutation of both `-stats` and `-reflect`, and `printStats` really does collapse the five duplicated statements the DRY comment was lying about. But the round-2 Critical is untouched: I reproduced BR-9 against the committed `Summarise` at HEAD (America/Havana 2026-03-08, Asia/Beirut 2026-03-29, America/Santiago 2026-09-06 each return `CurrentStreak=2, LongestStreak=2` where 3/3 is correct), and the Done-when row "streak arithmetic verified across timezone boundaries" is ticked over it. Six of the ten prior findings are still open, one of them Critical, so this blocks. Two new findings: the dispatch derivation that fixed BR-1 covers five of six modes — `-forget` is structurally invisible to it (mutation-verified) — and this window's closing commit deleted `workshop/plans/000048-play-from-the-loop-plan.md` while issue #48 (status `working`) still points at it and its gate ledger survives beside it.

## 1. Strengths

- **`TestEveryDispatchedModeIsInTheCollisionList` (`cmd/define/harvest_test.go:851`) is the right shape of fix.** Deriving the extent from the *dispatch* rather than from the list it is checking is the both-directions closure the finding asked for, and it is mutation-checked on a member the finding did not name. Verified: deleting `{"-stats", *statsFlag}` → `harvest_test.go:946` names `-stats`; deleting `{"-reflect", *reflect}` → names `-reflect`.
- **`printStats` (`cmd/define/stats.go:63`)** made a false comment true. Both doors now differ only in where the store comes from and in the `who` string; the previous "everything below that seam is shared" was five copies.
- **`Summarise`'s deck-vs-log asymmetry (`cmd/define/schedule/stats.go:88-100`) is genuinely the right call and well-pinned.** `TestKnownCountsTheDeckNotTheLog` asserts both halves in one fixture, and `TestMasteredAgreesWithTheScheduleFunction:106` includes a fixture-reaches-the-branch assertion so it cannot be comparing two zeros.
- **`TestEveryStatsFieldIsRendered` (`cmd/define/stats_test.go:45`)** derives the field set by reflection with a loud `default:` branch, and names `LastDay` as a decision rather than an oversight.
- Suite is green at HEAD (`go test ./cmd/define/...`, 110s), `gofmt` clean, `go vet` clean under default / `pty` / `conformance`.

## 2. Critical findings

**BR-9 (still open) — `cmd/define/schedule/stats.go:203-236`.** Reproduced at HEAD, not merely re-read. One correction to the finding's own fix sketch, which matters: **wrapping the stepped values in `store.StartOfDay` does not work.** In Havana, `StartOfDay(Mar 8 12:00)` returns `2026-03-07T23:00:00-05:00` — a key whose calendar date is Mar 7 — and `StartOfDay` of *that* returns `2026-03-07T00:00:00-05:00`, i.e. Mar 8's key collapses onto Mar 7's and `ActiveDays` drops to 2. Only the second option is correct: key the day set by a civil date (`struct{y int; m time.Month; d int}`) or by `dayIndex`'s UTC day number, which is the encoding `store.DaysBetween` already uses and which makes both this and the `*Location`-pointer hazard unrepresentable. Add a midnight-transition row to `TestStreaksAcrossDSTBoundaries` (the table covers only New York, whose transition is 02:00) and confirm it reddens first.

## 3. Important findings

**New — `cmd/define/harvest_test.go:851-947`, family `hand-maintained-extent`.** This is the 3rd finding in family `hand-maintained-extent`. BR-1 was fixed as an instance, so per the escalation: do not fix this instance — the rule is what needs fixing. The rule: *a derivation must fail closed against the count the other side declares, never against a hand-typed floor.* Three hand-typed floors survive the fix — `declaredModes` still says `len(names) < 5` while `run()` declares six (`harvest_test.go:837`), the new guard says `len(flagName) < 5` and `len(dispatched) < 3`. `-forget` dispatches through `if forgetting {` rather than `if *boolFlag {`, so the dispatch parser cannot see it at all, and the floor of 3 is far below 6 so the under-derivation never trips. Mutation-verified: deleting `{"-forget", forgetting}` from `main.go:605` leaves `TestEveryDispatchedModeIsInTheCollisionList` **green** — only the hand-listed `TestRunRefusesTwoModes` catches it, which is precisely the mechanism BR-1 said was insufficient. Measured prevalence: 5 of 6 modes covered, 3 of 3 floors hand-typed.

**New — `workshop/plans/`, family `artifact-deleted-to-satisfy-guard`.** `15b94c3` deleted `workshop/plans/000048-play-from-the-loop-plan.md`, added in `c208cf9` on this same branch. `workshop/issues/000048-...md:89` still reads *"Durable design: `workshop/plans/000048-play-from-the-loop-plan.md`"* and the issue is `status: working`; `workshop/plans/000048-play-a-sitting-without-leaving-the-loop-plan-gate.md` was *added* by the same commit and now ledgers a plan that exists on no branch tip (`git log --all` shows the file only in `c208cf9`). `workshop/lessons.md` records the reasoning as "keep a plan on the branch that implements it" — that is right, but the move was a deletion, not a move. `repo_guard_test.go:775-782` already warns about this exact rule one notch down ("a guard training authors to obfuscate their own citations, which is worse than the false positive"). Fix: restore the file onto a `#48` branch (`git show c208cf9:workshop/plans/000048-play-from-the-loop-plan.md`) before #8's close, or fix the issue pointer and move the gate ledger with it — do not leave the tree with a `working` issue citing a file no branch holds.

**BR-2 (still open, partial).** The `activeDays` row and the exit-code paragraph are fixed. What remains: `workshop/plans/000008-vocab-stats-plan.md:56` still claims the issue *"uses"* `store.DaysBetween`, and Task 2 Step 3 (`:294`) and Step 5's mutation sweep (`:296`) still say so, all ticked `[x]` — while `cmd/define/schedule/stats.go:194` states in as many words *"It does not call store.DaysBetween"*. A ticked mutation sweep that is unperformable as written is worse than an unticked one. Also still true: `countable`, `streaks` and the new `printStats` appear in no table row. `TestPlanTablesNameEntitiesThatExist` cannot reach any of these — its row regex requires the second cell to *begin* with a backticked `*.go` path, and the ARCH-DRY table's second cell begins with `` `store.DaysBetween` ``.

**BR-5, BR-6, BR-7, BR-8, BR-10 (still open).** Re-verified individually; details in the dispositions below.

## 4. Minor findings

- `cmd/define/README.md:407` still prints `since  3 January`; re-rendered the README's exact `Stats` values against HEAD's `renderStats` — every other line is byte-identical, that one is `since  Jan 3`.
- `store.StartOfDay` is documented and pinned as idempotent (`store/clock_test.go:85`), and is not: `StartOfDay(StartOfDay(t)) != StartOfDay(t)` for Havana 2026-03-08. Outside this window (unchanged file), so not a finding against this diff — but it is BR-9's root, and the pin uses a single zone whose transition is at 02:00.

## 5. Test coverage notes

- The DST table (`schedule/stats_test.go:168`) covers only `America/New_York`, whose transition is 02:00, and the fractional-offset table uses two non-DST zones. No test exercises a zone whose transition is *at local midnight* — which is exactly the class BR-9 lives in, and why a Done-when row claiming timezone verification is ticked over a reproducible bug.
- BR-4's fix has no test that reddens on revert. That is inherent to a de-duplication finding (`TestSlashStatsAndTheFlagPrintTheSameScreen` passes with the duplication too, since the copies produced identical output), so I'm disposing it addressed on structural verification — both entry points call `printStats` at `stats.go:48` and `:228` — rather than on a failing test. Worth knowing that this one is unguarded against re-divergence if a `--days` window lands in one door.
- No end-to-end test drives `-stats` through `run()` against a genuinely empty deck; the Done-when's empty-deck row is pinned at the `renderStats` unit only.

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — flag (BR-2 remainder, and the new `hand-maintained-extent` rule). Pass on the code: `printStats` and the `schedule.Mastered` / `store.Key` reuse are correct.
- **ARCH-PURE** — pass. `Summarise` and `renderStats` are pure with `now` as a parameter; `TestSchedulePurity` (`schedule/purity_test.go:17`) enforces the boundary mechanically rather than by promise, and `printStats` is the only thing touching IO.
- **ARCH-PURPOSE** — flag. The Done-when row "streak arithmetic verified across timezone boundaries" is ticked while a reproducible timezone bug ships; the *class* is DST-at-midnight, of which the shipped table tests the easy subset (02:00 transitions).
- **ARCH-MOCK** — pass. No new external dependency; `store.Mem` through the existing conformance-tested seam, `testDeps` unchanged.
- **ARCH-CONSTRAINTS** — pass. `O(events + deck)` with the counter explicitly refused; the `Events(time.Time{})` whole-log read matches `/history`'s existing cost on the same interactive path.
- **ARCH-SECURE** — flag (BR-6, BR-10, both open). The class rule worth writing down rather than patching site by site: **persisted event fields are normalised at the store's read boundary, not at each consumer's draw call.** `sanitiseItem` (`store/item.go:160`) already does exactly this for `Item.Form` — its own comment says Form "was the one persisted vocabulary in this store that did not" parse. `ReviewEvent.Form` is the same vocabulary, written by `Outcome.Form`, and is neither parsed nor neutralised at the seam — so `#8` became the first consumer that had to remember, and remembered at render time (`formLabel`, `stats.go:190`), leaving the map still keyed by the raw string. Same shape as `Fold(events)` running unfiltered while every other figure goes through `countable`. One fix at the store seam retires all three.
- **ARCH-ORDER** — pass. `Summarise` carries no state between events; the plan writes the `N/A` as a claim with a reason rather than a bare marker, and the chronological-order dependency is cited to `store/store.go:18` and checked against both implementations.

## 7. Plan revision recommendations

- `workshop/plans/000008-vocab-stats-plan.md` — a `## Revisions` entry: **"`streaks` does not call `store.DaysBetween`."** Correct the ARCH-DRY table row at `:56` from "uses it" to "deliberately does not — the fold needs *is the previous day present*, a map lookup, not *how many days between*", rewrite Task 2 Step 3 (`:294`) to name what was actually implemented, and either restate Step 5's mutation sweep against something the code performs or untick it. A ticked step describing an unperformable mutation is the ledger's own `plan-table-drift` family reporting that the enumeration was never written.
- Same entry: add `countable`, `streaks` and `printStats` to the Core-concepts / Integration-points tables with their true statuses, and note that the entity-table guard's row regex cannot reach a cell that does not *begin* with a backticked `.go` path — which is why BR-2 survived a round with the guard active.
- `workshop/issues/000008-vocab-stats.md` — untick "Streak arithmetic verified across timezone boundaries and gaps with a fake clock" until BR-9 is fixed and a midnight-transition row is red-first.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Mutation-verified on a scratch clone at HEAD: deleting -stats reddens harvest_test.go:946 by name, and so does deleting -reflect.
  - id: BR-2
    disposition: not-addressed
    note: |
      activeDays row and exit code fixed; plan:56 and Task 2 Steps 3/5 still claim store.DaysBetween is used, and countable/streaks/printStats are in no row.
  - id: BR-3
    disposition: addressed
    note: |
      Every step is ticked, so TestPlanTablesNameEntitiesThatExist now runs; it just cannot reach the ARCH-DRY row that keeps BR-2 open.
  - id: BR-4
    disposition: addressed
    note: |
      printStats at stats.go:63; both doors call it (:48, :228). No test reddens on revert — inherent to a de-duplication fix, verified structurally.
  - id: BR-5
    disposition: not-addressed
    note: |
      README:407 still "3 January"; re-rendered the README's exact values at HEAD — every other line byte-identical, that one is "Jan 3".
  - id: BR-6
    disposition: not-addressed
    note: |
      Fold(events) at stats.go:85 is still unfiltered, the doc comment still claims validation without naming the exception, nothing pins it.
  - id: BR-7
    disposition: not-addressed
    note: |
      README:521 still hand-lists the six modes; no doc_sync_test.go guard reuses declaredModes.
  - id: BR-8
    disposition: not-addressed
    note: |
      runStats still takes an unused ctx at stats.go:37 with no `_ context.Context` and no line saying why.
  - id: BR-9
    disposition: not-addressed
    note: |
      Reproduced at HEAD in three zones. Note the finding's first fix sketch is wrong: StartOfDay-wrapping collapses Havana's Mar 8 key onto Mar 7. Civil-date/UTC-day-index keying is the only correct option.
  - id: BR-10
    disposition: not-addressed
    note: |
      Accuracy is still keyed by the raw Form string with neutralisation only at formLabel (stats.go:190).
findings:
  - id: new
    severity: Important
    family: hand-maintained-extent
    title: |
      The dispatch derivation covers five of six modes and calibrates against three hand-typed floors
    detail: |
      This is the 3rd finding in family hand-maintained-extent; BR-1 was fixed as an
      instance, so state the rule rather than patching this site. The rule: a derivation
      must fail closed against the count the OTHER side declares, never against a
      hand-typed floor. declaredModes still says `len(names) < 5` while run() declares six
      (harvest_test.go:837); the new guard says `len(flagName) < 5` and
      `len(dispatched) < 3`. -forget dispatches via `if forgetting {` rather than
      `if *boolFlag {`, so the parser cannot see it, and the floor of 3 is far below 6 so
      the under-derivation never trips. Mutation-verified: deleting
      `{"-forget", forgetting}` from main.go:605 leaves
      TestEveryDispatchedModeIsInTheCollisionList green — only the hand-listed
      TestRunRefusesTwoModes catches it, which is the mechanism BR-1 called insufficient.
      Prevalence: 5 of 6 modes derived, 3 of 3 floors hand-typed.
  - id: new
    severity: Important
    family: artifact-deleted-to-satisfy-guard
    title: |
      The closing commit deleted issue 48's durable plan, leaving a working issue pointing at a file no branch holds
    detail: |
      15b94c3 deleted workshop/plans/000048-play-from-the-loop-plan.md, added by c208cf9 on
      this same branch, to make the plan-vs-code guards pass on #8's window.
      workshop/issues/000048-play-a-sitting-without-leaving-the-loop.md:89 still reads
      "Durable design: workshop/plans/000048-play-from-the-loop-plan.md" and the issue is
      status working; the same commit ADDED
      workshop/plans/000048-play-a-sitting-without-leaving-the-loop-plan-gate.md, which now
      ledgers a plan present on no branch tip (git log --all shows the file only in
      c208cf9). repo_guard_test.go:775-782 already records the rule one notch down: a guard
      worked around by mangling its input is worse than the false positive. Restore it onto
      a #48 branch via `git show c208cf9:workshop/plans/000048-play-from-the-loop-plan.md`,
      or fix the issue pointer and move the gate ledger with it.
```

---

## Re-review — 2026-09-08T14:52:48-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 8 — define --stats: deck, streak and mastery statistics |
| repo | tools |
| issue file | workshop/issues/000008-vocab-stats.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6cc7513b4ee056108f6900d482d73e0fee5def7b..fee2e1a06c1bc432c745622bd96e36b6bb780d91 |
| command | sdlc close --issue 8 |
| reviewer | claude |
| timestamp | 2026-09-08T14:52:48-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The two findings this round disposed as `addressed` — BR-9 (the local-midnight DST bug) and BR-11 (the mode-dispatch derivation) — are genuinely fixed, and I verified both by mutation rather than by reading the commit message: reverting `stats.go` to `15b94c3` reddens `TestStreaksSurviveAMidnightDSTTransition` in all three zones (2/3 streak vs 3), deleting `{"-forget", forgetting}` from `main.go` reddens `TestEveryDispatchedModeIsInTheCollisionList` by name, and hiding a dispatch shape from the parser reddens the count branch — so both directions of the new closure are reachable, not decorative. The shipped code is correct: the fold is pure and mechanically enforced (`TestSchedulePurity`), the deck-vs-log asymmetry is right and pinned, `Mastered` is called rather than re-derived, and `gofmt`/`go vet` under all three tag sets are clean with the whole `cmd/define` suite green. What blocks SHIP is not the code but the artifacts around it: `fee2e1a` changed the day-key mechanism and swept none of the prose that describes it, so `atlas/define.md` now records a premise (`time.Time` equality, location pointer) that the code no longer uses, and the plan's ARCH-DRY table and Task 2 steps still claim `store.StartOfDay`/`DaysBetween` are called when neither is. Both are cheap edits; neither is a code change.

## 1. Strengths

- **`civilDay` is the right fix, not the one the finding sketched.** BR-9 offered "wrap every stepped value in `store.StartOfDay`"; `cmd/define/schedule/stats.go:260` correctly rejected that (it collapses Havana's Mar 8 key onto Mar 7) and made the invalid state unrepresentable instead — three integers, `==`-comparable, arithmetic at noon UTC where no clock has ever moved (`:278`). That is ARCH-SECURE's "parse into a typed value" applied to a calendar.
- **The DST fixture asserts its own premise.** `TestTheMidnightZonesReallyLackAMidnight` (`stats_test.go:427`) checks that Havana/Santiago/Beirut actually lack a local midnight on those dates, so the row above it cannot pass for the wrong reason. Finding the zones by probing tzdata rather than by recall is the discipline the plan's own citation rule asks for.
- **BR-11's fix states the rule instead of patching the site.** `harvest_test.go:965-990`: the two derivations are each other's floor (`len(dispatched) < len(listed)`, plus `dispatched ⊄ listed`), and the `isSet(fs, "x")` reader takes the flag name from the *call* rather than guessing from the variable's spelling. The hand-typed `< 3` and `< 5` floors that certified the gap are gone from the closure.
- **`printStats` (`cmd/define/stats.go:63`) made BR-4's comment true.** Both doors (`:48`, `:228`) now differ only in where the store comes from and the name in a diagnostic, and `TestSlashStatsAndTheFlagPrintTheSameScreen` asserts byte-identical output.
- **`TestEveryStatsFieldIsRendered` fails closed on a new field** (`stats_test.go:90`) — a `Stats` field with no expectation errors rather than being silently skipped, and `LastDay`'s omission is a named decision.

## 2. Critical findings

None. BR-9 was the only Critical and it is fixed and mutation-verified.

## 3. Important findings

**`atlas/define.md:2643` records a keying mechanism the code abandoned in the same window** — Important, family `docs-restate-unverified-output`, **2nd in that family**. The atlas says the day map "is keyed in the LEARNER's zone, because `time.Time` equality includes the location pointer"; `stats.go:132` keys by `civilDayOf(...)`, a struct of three ints, in which the location pointer cannot participate in equality at all. `civilDay` — new terminology with a non-obvious rationale — appears nowhere in `atlas/`, the plan, or the README.

Per the escalation rule, the deliverable is the class, not the site. **The rule already exists in this repo, written down and unapplied:** `workshop/targets/derived-restatement.md` carries a sweep checklist to be run at *every* close, and `fee2e1a` skipped three of its rows — "`atlas/` — surface, flow, terminology, and the premises it records", "the plan — a `## Revisions` entry, never an overwrite", and "the doc comment on every symbol the diff reshaped". Measured enumeration for this window: atlas premise stale (1); README date wrong (BR-5); plan ARCH-DRY rows 55–56 and Task 2 Steps 3/5 stale, all ticked (BR-2); no plan `## Revisions` entry for the redesign; five new pure symbols in no entity row. **Six sites, one skipped checklist.** Also worth noting: `#8`'s issue frontmatter carries no `target: derived-restatement` reference despite four findings in the family — the target cannot defend an issue that does not point at it.

**BR-2 remains open and widened** (see dispositions). `fee2e1a` removed the last two `store.StartOfDay`/`DaysBetween` call sites from the fold, so the plan's two ARCH-DRY rows are now both false rather than one.

## 4. Minor findings

- `cmd/define/stats.go:190` — `formLabel` is a byte-for-byte copy of `store.oneLine` (`store/item.go:200`), under a comment asserting the equivalence with nothing pinning it. Family `duplicated-read-path`, **2nd in that family** — the rule is BR-4's: when a comment says a behaviour is shared, the next line should be the function it is shared through. Prevalence 2 (measured: `grep unicode.IsControl` finds exactly these two copies of the `IsControl && !IsSpace` + `Fields`/`Join` idiom).
- BR-5, BR-6, BR-7, BR-8, BR-10 all still stand unchanged; see dispositions.

## 5. Test coverage notes

Coverage is strong and the pins are real, not restatements of the implementation — I confirmed three of them redden under mutation. Two residual gaps, both already open: nothing pins that `Fold` runs on *unfiltered* events (BR-6), so the "the fold validates rather than trusts" doc comment is unguarded at its one exception; and nothing pins `formLabel ≡ store.oneLine`. The whole-suite claim in Task 5 Step 6 checks out: `go test ./cmd/define/...` green, `gofmt -l` clean, `go vet` clean under default, `pty` and `conformance`.

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — flag (`formLabel`/`oneLine`); otherwise strong (`Mastered` called not re-derived, `printStats` consolidated, `noDeckMessage` reused rather than copied).
- **ARCH-PURE** — pass. `Summarise`/`renderStats` are pure with `now` as a parameter, and `TestSchedulePurity` enforces "no IO, no hidden clock" mechanically rather than by promise.
- **ARCH-PURPOSE** — flag, as above: the code fulfils the issue, the shadow-sweep of its documentation does not.
- **ARCH-MOCK** — pass. No new external dependency; `store.Mem` sits behind the existing conformance-tested seam.
- **ARCH-CONSTRAINTS** — pass. O(events + deck), envelope declared, no counter and no cache; `/stats` re-reads the whole log per invocation, which is user-initiated and bounded.
- **ARCH-SECURE** — pass with the two open Minors. `countable` skips zero/future `At` (pinned both ways), `formLabel` neutralises control runes (pinned). Residual: raw `Form` as an `Accuracy` map key (BR-10), `Fold` on unfiltered events (BR-6).
- **ARCH-ORDER** — pass. `Summarise` carries no state between calls, and the plan writes the `N/A` as a claim with reasons rather than a bare marker. `streaks`' unbounded `for` terminates on a finite `days` map; worth one sentence saying so.
- **For `#48`:** `TestEveryDispatchedModeIsInTheCollisionList` still cannot see a mode wired through a shape neither AST reader recognises — both derivations would miss it identically. That is inherent to AST derivation, but `/play`'s dispatch should stay in one of the two known shapes, or the guard needs a third reader.
- **The highest-leverage next guard:** `TestPlanTablesNameEntitiesThatExist` checks that named entities exist, never that existing entities are named — so a *missing* row is invisible. The both-directions closure `#8` just built for modes is the same move: read the new symbols out of a file a plan's table names and require each to appear in some row. That would have caught five of this round's six sites mechanically.

## 7. Plan revision recommendations

Append one `## Revisions` entry dated 2026-09-08 (do not overwrite the round-1 entry) covering:

1. **The day set is keyed by a civil DATE, not by `store.StartOfDay`'s `time.Time`.** Rows at `workshop/plans/000008-vocab-stats-plan.md:55-56` claim the code "uses" `store.StartOfDay` and `store.DaysBetween`; it calls neither. Rewrite both rows to what shipped: `DaysBetween`'s *rule* (b's location defines the calendar) is applied where the keys are built, and the instant-based helpers are deliberately not called because an instant can fail to exist.
2. **Task 2 Step 3 (`:293-294`) and Step 5 (`:296`) are ticked over a design that no longer exists** — "implement over `store.StartOfDay` and `store.DaysBetween`", "replace `DaysBetween` with a `Sub()/24h` computation". Restate the mutation sweep that actually pins the row (revert to instant keys → `TestStreaksSurviveAMidnightDSTTransition` reddens in three zones). Same for the test-name comment at `:268`.
3. **Add the missing entity rows or say why the table is partial.** `civilDay`, `civilDayOf`, `countable`, `streaks` (`schedule/stats.go`) and `printStats`, `streakPhrase`, `accuracyLines`, `formLabel` (`cmd/define/stats.go`) are in no row, and `runStatsCommand` is in no Integration row. The plan's own note at `:119` explains why the day set did not become a named function — but `streaks` and `civilDay` *are* named functions, so that paragraph now argues the opposite of what shipped.

```findings
dispose:
  - id: BR-2
    disposition: not-addressed
    note: |
      activeDays row and exit code fixed; plan:55-56, Task 2 Steps 3/5 and the :268 comment still claim StartOfDay/DaysBetween are called — fee2e1a removed the last call site, so both rows are now false — and civilDay/countable/streaks/printStats/runStatsCommand are in no entity row.
  - id: BR-5
    disposition: not-addressed
    note: |
      README:407 still "3 January"; re-rendered all nine sample lines against the renderer's format strings at HEAD — eight are byte-identical, the date is "Jan 3".
  - id: BR-6
    disposition: not-addressed
    note: |
      Fold(events) at stats.go:85 is still unfiltered, the countable doc still claims the fold validates without naming the exception, and nothing pins it.
  - id: BR-7
    disposition: not-addressed
    note: |
      README:521 still hand-lists the six modes; no doc_sync_test.go guard reuses declaredModes, which now exists and is free to call.
  - id: BR-8
    disposition: not-addressed
    note: |
      runStats still takes an unused ctx at stats.go:37, with no `_ context.Context` and no line saying it mirrors runReflect deliberately.
  - id: BR-9
    disposition: addressed
    note: |
      civilDay keys the day set at stats.go:260. Mutation-verified: restoring 15b94c3's stats.go reddens TestStreaksSurviveAMidnightDSTTransition in Havana, Santiago and Beirut (2/3, want 3/3).
  - id: BR-10
    disposition: not-addressed
    note: |
      Accuracy is still keyed by the raw Form string with neutralisation only at formLabel (stats.go:190).
  - id: BR-11
    disposition: addressed
    note: |
      Mutation-verified twice: deleting {"-forget", forgetting} reddens the dispatched-not-listed branch by name, and hiding the `if forgetting {` shape from the parser reddens the count branch (5 found vs 6 declared). Both floors now derive from the other side.
  - id: BR-12
    disposition: addressed
    note: |
      Issue 48's Plan section now explains the plan lands with its own branch and cites BR-12; the cited guards exist at repo_guard_test.go:1306/:1271 and do walk every plan. The deleted content stays recoverable at c208cf9, an ancestor of HEAD.
findings:
  - id: new
    severity: Important
    family: docs-restate-unverified-output
    title: |
      atlas/define.md records the time.Time keying premise that fee2e1a replaced, and the target's own sweep checklist was not run for that window
    detail: |
      This is the 2nd finding in family docs-restate-unverified-output, so the deliverable is
      the class rather than the site. The rule is already written down and was not applied:
      workshop/targets/derived-restatement.md carries a sweep checklist for every close, and
      fee2e1a skipped the "atlas/ — surface, flow, terminology, and the premises it records",
      "the plan — a ## Revisions entry", and "the doc comment on every symbol the diff
      reshaped" rows. atlas/define.md:2643 says the day map "is keyed in the LEARNER's zone,
      because time.Time equality includes the location pointer" — stats.go:132 keys by
      civilDayOf(), a struct of three ints in which the location pointer cannot participate
      in equality at all, and civilDay appears nowhere in atlas/, the plan or the README.
      Measured prevalence for this one window: six sites — atlas premise, README date (BR-5),
      plan rows 55-56, plan Task 2 Steps 3/5, no plan Revisions entry, five new pure symbols
      in no entity row. Fix the sweep, not the sentence; and note that issue 8's frontmatter
      carries no `target: derived-restatement` reference despite four findings in the family.
  - id: new
    severity: Minor
    family: duplicated-read-path
    title: |
      formLabel is a byte-for-byte copy of store.oneLine under a comment asserting the equivalence, with nothing pinning it
    detail: |
      This is the 2nd finding in family duplicated-read-path, so state the rule rather than
      patching the site: when a comment says a behaviour is shared with X, the next line
      should be the call to X. cmd/define/stats.go:190-202 reproduces store/item.go:200-208
      exactly (unicode.IsControl && !unicode.IsSpace, then strings.Fields/Join) and its doc
      comment says "exactly as store's oneLine drops them". oneLine is unexported, so a
      hardening of the neutralisation rule — the ARCH-SECURE rule for hand-edited text
      reaching a terminal, per 12 BR-15 — would harden one copy and silently leave the other.
      Prevalence measured at 2: grep unicode.IsControl finds exactly these two copies of the
      idiom. Export it from store, or pin the equivalence with a shared table.
```

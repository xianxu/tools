# Boundary Review — tools#5 (whole-issue close)

| field | value |
|-------|-------|
| issue | 5 — spaced-repetition scheduling engine (Leitner, pure) |
| repo | tools |
| issue file | workshop/issues/000005-vocab-schedule.md |
| boundary | whole-issue close |
| milestone | — |
| window | a30bb786560befe4960d28eaf5098a3bc678efa6..118b564bafd4890c4669be1776d80c3bf9de82ad |
| command | sdlc close --issue 5 |
| reviewer | claude |
| timestamp | 2026-08-26T23:12:03-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The scheduling engine is genuinely well-built — the derived-not-stored decision is right and enforced (`store.Word` gained no fields), `Fold` applies `Answer` so the transition has one encoding, the two-tier queue is the correct answer to the starvation clause and its fixture actually discriminates the two orderings, and the purity guard is real rather than asserted. What blocks SHIP is one correctness defect at the heart of the deliverable — `store.DaysBetween` returns the wrong calendar-day count whenever its two arguments carry different UTC offsets, which makes `Due` report a word due on the day it was reviewed (verified empirically, below) — plus a Done-when row that is ticked but not delivered: the interval ladder's middle values are pinned by no test at all (mutating box 3 from 14 to 13 days leaves the whole suite green, verified by `go test -overlay`). Secondary but cheap: the purity guard has a hole exactly where a future edit would land (`time.Since` sails past both guards, verified), and the "imports `store` and `time` and nothing else" claim is now false in four artifacts — the third instance of the family this issue's own plan gate ran twice and wrote an enumeration for.

## 1. Strengths

- **`Fold` applies `Answer` (`progress.go:100`)** rather than reimplementing the transition, and `TestFoldOfOneEventEqualsOneAnswer` pins that structurally rather than by convention. This is PQ-2 actually delivered, not just claimed.
- **The starvation fixture is adversarial to its own test** (`queue_test.go:24`): `fresh` is 60 days old with 9 lookups, `overdue` is 17 days overdue with 1 lookup, so a single-axis sort produces a *different* answer. That is what makes the tier mutation die. `TestBudgetGoesToOverdueBeforeFresh` re-tests the same rule at the budget boundary where it actually bites.
- **`TestFoldOnlyCountsReviews` (`progress_test.go:118`) places the non-review events after two promotions** with the reason recorded in the comment — the fixture-that-cannot-discriminate problem was found and fixed rather than papered over.
- **Both purity guards refuse to pass vacuously** (`purity_test.go:57`, `:100`): `t.Fatal` when `go list` returns nothing and when no source files were checked. That is the failure mode most guard tests ship with.
- **Task 0 is a real ARCH-DRY collapse, not a cosmetic one** — two genuinely different encodings (`history_cmd.go:41` local midnight, `:195` UTC day-index closure) became one helper, and `/history`'s tests were left as the regression net.

## 2. Critical findings

**C1 — `store.DaysBetween` (`cmd/define/store/clock.go:53`) is wrong for arguments in different locations; `schedule.Due` is its first victim.**

`StartOfDay` is applied to each argument *in its own location*, then the loop compares instants. When the offsets differ, the two midnights are not a whole number of days apart and the count is off by one. Verified:

```
at = 2026-08-03T09:00Z (a YAML-parsed event stamp), now = 2026-08-03 20:00 America/Los_Angeles
  DaysBetween = 1     (same local calendar date under either reading — the answer is 0)
at = 2026-10-30 10:00 -0700 (fixed zone, as yaml.v3 parses it), now = 2026-11-05 10:00 LA (-0800)
  DaysBetween = 7     (calendar answer 6; the old dayIndex closure returned 6)
```

Consequence: `Due(Progress{Box:0, LastReviewed: <today, other offset>}, now)` returns **true on the same day the word was reviewed**, and `Queue`'s `overdue` (`queue.go:59`) is inflated by one for every word whose stamp predates a DST transition. `Fold` reads `At` straight off YAML-parsed events (fixed-offset zones), `now` comes from `SystemClock` (`time.Local`) — so the mixed-zone case is the *normal* state for half the year, not an edge case.

`relativeDay` escapes only because it normalises first (`history_cmd.go:187`, `at = at.In(now.Location())`); `Due` and `Queue` do not, and no current test crosses zones, which is why this is green. Fix: normalise inside the helper — `a = a.In(b.Location())` before `StartOfDay` (matching the choice `relativeDay` already makes), document that the *second* argument's location defines the calendar, and add a table row pairing `time.FixedZone("PDT", -7*3600)` against `America/Los_Angeles`. Family: `local-calendar-day-boundary` — the same family PQ-1 raised at plan time; the plan settled the *policy* and the implementation lost it on mixed inputs.

## 3. Important findings

**I1 — the interval ladder's middle rungs are pinned by nothing, and the plan promised otherwise.**
`box_test.go` asserts box 0 = 1 and `LastBox` = 90 plus strict increase; `TestDue` pins box 1 = 3. Boxes 2, 3 and 4 (7, 14, 30) are asserted by no test. Verified with `go test -overlay`: changing `intervalDays` to `{1,3,7,13,30,90}` leaves the entire suite **green**. Plan Task 2 Step 2 says "walking a word from box 0 to mastery and back down after a miss, **asserting the due date at each step**"; `TestMultiWeekSchedule` (`progress_test.go:162`) walks the steps but asserts dueness only at the endpoint. Fix: assert `Due`/`!Due` at each rung of `reviewDays`, which pins the whole table as a side effect. Family: `spec-constant-unpinned`.

**I2 — "imports `store` and `time` and nothing else" is false in four artifacts; the sweep this plan wrote down was not run when the rule changed.**
M2 rewrote the guard's rule to "no IO and no hidden clock" and `queue.go` now imports `sort`, but the claim still stands at `cmd/define/schedule/box.go:7`, `atlas/define.md:1142`, `workshop/projects/define-learn.md:485`, and `workshop/plans/000005-vocab-schedule-plan.md:27,33,62` ("asserts the set is exactly `store` + `time`"). The plan's own Revisions section names the enumeration — *"the issue file, the plan file, `atlas/`, and code comments — four places, checked by grepping the claim"* — and the M2 commit changed the rule without running it. This is the third instance of the family the plan gate raised twice (PQ-6, PQ-7). Family: `unbacked-claim-about-existing-code`.

**I3 — the purity guard misses every clock-reading call except `time.Now(`.**
`purity_test.go:91` greps one token. Verified by overlay: inserting `_ = time.Since(p.LastReviewed)` into `Due` leaves **both** guards passing. `time.Since`, `time.Until`, `time.After`, `time.Tick`, `time.NewTimer`/`NewTicker` all read the wall clock, and `time.Since(lastReviewed)` is the single most plausible edit someone would make to this package. Fix: loop over a banned-token slice instead of one `strings.Contains`. Family: `invariant-needs-mechanical-guard` (PQ-3's family).

**I4 — `LastBox` is an exported mutable package variable (`box.go:30`).**
Every clamp and `Mastered` depends on it, and any consumer can assign to it. This is a new internal package that `#6` and `#8` will import, so the surface should be stable. Cheap and verified to compile: make the table an array — `var intervalDays = [...]int{1, 3, 7, 14, 30, 90}` — and `len` becomes a constant expression, so `const LastBox = len(intervalDays) - 1`.

**I5 — `Queue` silently removes mastered words (`queue.go:46`), which is in neither the Spec nor the plan, and it is an absorbing state.**
The plan's `Queue` bullet describes tiers, budget, ties and the deck-vs-log rule; mastery-exclusion is not among them. It also contradicts `progress.go:61`, which says *"`#6`'s `--play` decides what to stop offering"* — here `Queue` decides. And because a mastered word is never offered again it can never be answered wrong, so `Mastered` is permanent: there is no path back to the rotation. Decide whether that is intended (a 90-day re-confirmation is the usual Leitner answer), then record it in the Spec/plan. Family: `undeclared-behavior`.

## 4. Minor findings

- `queue.go:65,74` — the two comparators share the same tail (lookups desc, then key asc); extract `byLookupsThenKey(a, b candidate) bool` (ARCH-DRY).
- `queue.go:81` — `make([]string, 0, budget)` panics `makeslice: cap out of range` on an absurd budget (verified). Cap at `min(budget, len(deck))`; it is also the tighter allocation.
- `clock.go:53` — `DaysBetween` steps one `AddDate` per day. `Due` on a years-old stamp loops thousands of times per word per queue build. Correct, but a divide-with-DST-correction or a date-index subtraction would be O(1) if this ever shows in a profile.
- `queue.go:34` — `Queue` returns normalised keys, not deck `Text`; the doc comment says "today's words". Worth naming the key-vs-text contract explicitly for `#6`.
- `queue.go:84` — `append(reviewed, fresh...)` writes into `reviewed`'s spare capacity; harmless here since neither slice is reused, but `slices.Concat` states the intent.

## 5. Test coverage notes

Coverage of the *stated* decisions is strong — transitions, clamps, streak reset, key normalisation, `Fold`-equals-`Answer`, both queue tiers, budget, determinism, the deck-is-the-roster rule and mastery exclusion each have a named test, and `FuzzFold`'s properties are the ones that actually hold over the domain (the permutation invariant was correctly rejected). Three real gaps: **(a)** no test crosses time zones anywhere, which is why C1 is invisible; **(b)** the interval table's middle values are unasserted (I1, mutation-verified); **(c)** `TestQueueRespectsTheBudget` exercises the budget only in the fresh tier with `prog == nil` — `TestBudgetGoesToOverdueBeforeFresh` covers the mixed case, so this is adequate, but a budget cut *inside* the overdue tier is untested. `TestStartOfDay`'s `t.Skipf` on missing tzdata is the right call, but it means CI without tzdata silently loses every DST assertion — worth a `//go:build` note or `time/tzdata` import if that platform is real.

## 6. Architectural notes

- **ARCH-DRY — pass, with I2/Minor caveats.** Task 0 is a textbook instance: three needs for one concept, collapsed into `store` beside the `Clock` that owns the time model, with the ownership decision argued rather than defaulted. `Fold`→`Answer` removes the second encoding of the transition. The residue is the duplicated comparator tail, and — more importantly — the *documentation* copies of the purity rule (I2), which is DRY applied to claims rather than code.
- **ARCH-PURE — pass.** `schedule` is genuinely pure: every `now` is a parameter, no package state, and the guard is mechanical rather than asserted. The IO stays in `#6`. The one wrinkle is that the guard's coverage is narrower than its claim (I3), and the plan's Core-concepts table is honest that every entity is PURE — verified: all seven exist at the stated paths, all `new`, all tested without fakes. The two guard tests do use `exec`/`os`, but they are *meta*-tests about the package rather than tests of a pure entity, and `repo_guard_test.go` is the established precedent; no promotion to INTEGRATION warranted.
- **ARCH-PURPOSE — pass on the deferral, flag on C1/I1.** The `Mastered`-consumed-by-two row is honestly under-delivered with the reason recorded in the Done-when itself (`#8` filed, unbuilt) — that is a genuine follow-up, not the deferred point of the issue. But two Done-when rows are ticked ahead of the evidence: "due-dates verified across a simulated multi-week schedule" is ticked while three of six intervals are unpinned (I1), and the whole "which words are due" purpose is defeated for mixed-zone stamps (C1). Fix the class, not the site: C1's fix belongs in `DaysBetween` with a cross-zone table row, not in `Due`.
- **ARCH-MOCK — pass, not applicable.** The diff introduces no external binary or service dependency in production code. The guard tests shell out to `go list` without a seam; that is test-only infrastructure with precedent, and inventing a fake toolchain for it would be worse than the disease.

## 7. Plan revision recommendations

1. **`unbacked-claim-about-existing-code`, third instance** — the M2 guard rule change was never swept over the four artifacts the plan itself enumerates. A `## Revisions` entry stating the new rule (*"no IO and no hidden clock"*, allowlist `store`, `time`, `sort`, `slices`, `cmp`, plus the `time.Now` source check), correcting lines 27, 33 and 62, and noting that the sweep must run whenever the *rule* changes, not only when a *claim* is found wrong.
2. **Task 2 Step 2 was not delivered as written** — "asserting the due date at each step" became an endpoint assertion. Either the test grows the per-rung assertions (preferred, it also closes I1) or the plan records the reduction and why.
3. **`Queue`'s mastery exclusion is undeclared** — add it to the `Queue` bullet with the absorbing-state question answered (does a mastered word ever return?), and reconcile with `progress.go:61`'s claim that `--play` is what decides.
4. **`store.DaysBetween`'s cross-location contract is unstated** — the plan settled "local calendar day" but never said which calendar when the two timestamps disagree. Record the decision (the `now` argument's location) alongside the `StartOfDay` ownership paragraph.

```findings
findings:
  - id: new
    severity: Critical
    family: local-calendar-day-boundary
    title: |
      store.DaysBetween miscounts by a day when its arguments carry different UTC offsets, so Due fires on the review day itself
    detail: |
      cmd/define/store/clock.go:53 applies StartOfDay to each argument in its OWN
      location and then compares instants, so two midnights at different offsets are
      not a whole number of days apart. Verified: an event stamped 2026-08-03T09:00Z
      against now = 2026-08-03 20:00 America/Los_Angeles returns 1, not 0, so a box-0
      word is Due the same calendar day it was reviewed; a -0700 stamp against a -0800
      now returns 7 where the old dayIndex closure returned 6. Fold reads At off
      YAML-parsed events (fixed-offset zones) while now comes from time.Local, so
      mixed offsets are the normal state for half the year. relativeDay escapes only
      because history_cmd.go:187 does at.In(now.Location()) first; Due and Queue do
      not. Normalise inside the helper and add a cross-zone table row.
  - id: new
    severity: Important
    family: spec-constant-unpinned
    title: |
      Three of the six Leitner intervals are asserted by no test, and the plan promised a due-date assertion at each step
    detail: |
      box_test.go pins box 0 and LastBox plus strict increase; TestDue pins box 1.
      Boxes 2, 3 and 4 (7, 14, 30) are unpinned — verified with go test -overlay that
      mutating the table to {1,3,7,13,30,90} leaves the whole suite green. Plan Task 2
      Step 2 says the multi-week test asserts "the due date at each step";
      progress_test.go:162 walks the rungs but asserts dueness only at the endpoint.
      Asserting Due/!Due at each rung closes both the plan item and the mutation.
  - id: new
    severity: Important
    family: unbacked-claim-about-existing-code
    title: |
      "imports store and time and nothing else" is now false in four artifacts; the enumeration the plan wrote was not run when the guard rule changed
    detail: |
      M2 rewrote the guard to "no IO and no hidden clock" and queue.go imports sort,
      but the two-imports claim still stands at cmd/define/schedule/box.go:7,
      atlas/define.md:1142, workshop/projects/define-learn.md:485 and
      workshop/plans/000005-vocab-schedule-plan.md:27,33,62. The plan's own Revisions
      section names the sweep — issue file, plan file, atlas, code comments — and the
      M2 commit changed the rule without running it. Third instance of the family the
      plan gate raised as PQ-6 and PQ-7.
  - id: new
    severity: Important
    family: invariant-needs-mechanical-guard
    title: |
      The clock guard greps only "time.Now(" and misses time.Since, time.Until, time.After and the timer constructors
    detail: |
      cmd/define/schedule/purity_test.go:91 is a single strings.Contains. Verified by
      overlay: inserting `_ = time.Since(p.LastReviewed)` into Due leaves BOTH purity
      guards passing, though time.Since reads the wall clock and is the most plausible
      edit anyone would make here. Loop over a banned-token slice instead.
  - id: new
    severity: Important
    family: mutable-exported-surface
    title: |
      LastBox is an exported mutable package variable that every clamp and Mastered depends on
    detail: |
      cmd/define/schedule/box.go:30. This package is about to be imported by #6 and #8,
      so the surface should be stable. Verified cheap fix: declare the table as an array
      (`var intervalDays = [...]int{...}`), which makes len a constant expression, then
      `const LastBox = len(intervalDays) - 1`.
  - id: new
    severity: Important
    family: undeclared-behavior
    title: |
      Queue silently drops mastered words — absent from Spec and plan, contradicting progress.go's own comment, and it is an absorbing state
    detail: |
      cmd/define/schedule/queue.go:46. The plan's Queue bullet covers tiers, budget,
      ties and the deck-vs-log rule but never mastery exclusion, and progress.go:61 says
      "#6's --play decides what to stop offering" while Queue decides. Because a
      mastered word is never offered it can never be answered wrong, so mastery is
      permanent with no path back into rotation. Decide, then record it.
  - id: new
    severity: Minor
    family: duplicated-comparator
    title: |
      The two sort.Slice comparators in Queue share an identical lookups-then-key tail
    detail: |
      cmd/define/schedule/queue.go:65 and :74 — extract byLookupsThenKey (ARCH-DRY).
  - id: new
    severity: Minor
    family: hostile-input-at-the-seam
    title: |
      Queue panics on an absurd budget where it deliberately handles budget <= 0
    detail: |
      cmd/define/schedule/queue.go:81 — make([]string, 0, budget) panics
      "makeslice: cap out of range" (verified). Cap the capacity at len(deck); it is
      also the tighter allocation.
  - id: new
    severity: Minor
    family: linear-scan-where-arithmetic-suffices
    title: |
      DaysBetween steps one AddDate per day, so Due on a years-old stamp loops thousands of times per word
    detail: |
      cmd/define/store/clock.go:53. Correct, and fine at current scale; noted for the
      first profile that shows it.
  - id: new
    severity: Minor
    family: undeclared-behavior
    title: |
      Queue returns normalised keys rather than deck Text, and the doc comment says "today's words"
    detail: |
      cmd/define/schedule/queue.go:34 — name the key-vs-text contract explicitly, since
      #6 has to map back to render.
```

---

## Re-review — 2026-08-26T23:32:18-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 5 — spaced-repetition scheduling engine (Leitner, pure) |
| repo | tools |
| issue file | workshop/issues/000005-vocab-schedule.md |
| boundary | whole-issue close |
| milestone | — |
| window | a30bb786560befe4960d28eaf5098a3bc678efa6..850c9127b4e2638622f8ecb6c8fe43147ea60dd4 |
| command | sdlc close --issue 5 |
| reviewer | claude |
| timestamp | 2026-08-26T23:32:18-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The Critical from round 1 is genuinely fixed and genuinely pinned — I reverted `a = a.In(b.Location())` in a scratch copy and watched two independent tests go red (`clock_test.go:130` and `progress_test.go:291`), and I re-verified every other claimed fix the same way rather than reading the commit messages. Five of ten prior findings are addressed with real failing tests behind them; four are untouched. What stops this from being a clean SHIP is that **BR-3 — the finding whose entire content was "you wrote an enumeration and did not run it" — was itself answered by running the enumeration on 2 of 5 sites**, and the same round's fixes minted two *new* instances of the identical class, one of which (`atlas/define.md:1165`) now tells a future `#6` implementor the exact opposite of what `Queue` does, and the behaviour it describes is the absorbing-state bug BR-6 just removed. Second: the only test pinning this round's Critical fix is behind a `t.Skipf` on missing tzdata, and the repo already has the one-line cure in-tree (`history_cmd_test.go:16`) that `store` and `schedule` did not copy. Both are text/one-line edits; neither needs re-verification beyond `go test`, which is why this isn't REWORK.

## 1. Strengths

- **`store.DaysBetween` is now right for the reason stated, not by coincidence.** `clock.go:71-74` normalises into `b`'s zone and then subtracts two UTC day indices — recovering exactly the property `#15`'s discarded closure had documented. The doc comment (`clock.go:56-70`) records *why* the other shape lost, which is the rarest kind of comment in a fix.
- **The regression test discriminates.** `progress_test.go:283` uses a `FixedZone("HST", -10*3600)` stamp whose *own* calendar date differs from its date in the learner's zone. That's the detail an easier fixture gets wrong, and the test comment says so explicitly. Same-date pairs stay green under the mutation; this one doesn't.
- **`LastBox` became a compile-time constant, not a tested variable** (`box.go:33-36`). `var intervalDays = [...]int{…}` + `const LastBox = len(intervalDays) - 1` makes the old defect *unrepresentable* rather than merely detected — stronger than the test BR-5 asked for.
- **BR-6's fix is a design correction, not a patch.** Removing the mastery exclusion from `Queue` and letting the 90-day rung do the rarity (`queue.go:46-57`) restores `progress.go:61`'s own contract that `--play` decides what to stop offering. `TestMasteryCanBeLost` (`queue_test.go:161`) pins the consequence, and re-adding the exclusion in a scratch copy reddens `queue_test.go:156`.
- **The purity guard now covers the room, not one door** (`purity_test.go:103-107`), and both guards `t.Fatal` on a vacuous run rather than passing silently. I confirmed `_ = time.Since(p.LastReviewed)` in `Due` reddens `TestScheduleNeverReadsTheClock` in a real (non-overlay) scratch copy — note that an overlay does *not* work here, since the guard reads the file off disk.

## 2. Critical findings

None. No shipped-code defect survives this round.

## 3. Important findings

**I-a — `store.DaysBetween`'s regression net is behind `t.Skipf`, and the discriminating rows don't need what they skip on.**
`cmd/define/store/clock_test.go:20,79,100,141` and `cmd/define/schedule/progress_test.go:276` all `t.Skipf("no tzdata")` on `LoadLocation("America/Los_Angeles")` failure. I simulated the failure in a scratch copy: **both packages report `ok` with every cross-zone and DST assertion gone** — including `TestDueDoesNotFireOnTheDayOfReview`, the sole product-level guard for this round's Critical. GitHub's `ubuntu-latest` runner does ship tzdata, so CI is not currently blind; a distroless/alpine container would be.

*This is the 2nd finding in family `invariant-needs-mechanical-guard`.* Do not just fix `progress_test.go:276`. The rule: **a guard only guards if it is unconditionally reachable in the environment the suite runs in — an assertion that can skip itself is documentation.** Two mechanical consequences, both cheap:
1. Import `_ "time/tzdata"` in the `store` and `schedule` test packages. **The fix already exists in-tree at `cmd/define/history_cmd_test.go:16` and was not reused** (ARCH-DRY) — `#15` hit this and solved it; this issue re-derived the skip instead.
2. Run the enumeration: `grep -rn "t.Skip" cmd/define --include='*.go'` returns 16 sites. Twelve are legitimate environment gates (network, pty, `afplay`, no model configured, no system word list). **Four of the five tzdata sites don't need tzdata at all** — `TestStartOfDayIsIdempotent` (`:79`) and the two mixed-offset rows in `TestDaysBetweenAcrossLocations` (`:100`) and `TestDueDoesNotFireOnTheDayOfReview` discriminate on *offset*, which `time.FixedZone` always provides. Only `TestDaysBetweenAcrossDST` (`:141`) genuinely needs a DST-carrying zone. Prevalence: 4/5.

**I-b — see BR-3 below (disposed `not-addressed`, not re-raised).** The enumeration is now 5 sites, three of them stale from round 1 and two minted by this round's own fixes.

## 4. Minor findings

- BR-7, BR-8, BR-10 remain open exactly as filed — see dispositions. BR-8 still reproduces: `Queue(deck, nil, now, 1<<62)` panics `makeslice: cap out of range` (`queue.go:86`).
- `atlas/define.md:1142-1150` folds three separate historical corrections into one paragraph about the guard's first draft. It reads as a changelog; the atlas is meant to describe the current map. The corrections belong in `workshop/lessons.md`, where two of them already are.
- `progress_test.go` and `queue_test.go` each rebuild the mastery fixture with the same `for i := 0; i < masteryStreak; i++` loop. One shared `masteredProgress(t)` helper. Same rule as BR-7; not worth a separate round.

## 5. Test coverage notes

Coverage is strong and, unusually, *verified* strong — the round's own log records five mutations with one survivor found and killed. I re-ran the two the findings named and both die now: `{1,3,7,13,30,90}` reddens `box_test.go:19`, and the mastery exclusion reddens `queue_test.go:156`. `TestMultiWeekSchedule` now asserts `Due`/`!Due` at each rung (`progress_test.go:200-210`), which closes plan Task 2 Step 2 and pins the whole ladder as a side effect — that is the right shape, since it makes the plan item and the mutation the same assertion.

Two gaps remain, both small: a budget cut *inside* the overdue tier (more overdue words than budget) is untested — `TestBudgetGoesToOverdueBeforeFresh` covers the mixed case, so this is the last uncovered slicing path; and no test fixes `time.Local` to a non-UTC zone, so `SystemClock` × stored-stamp — the pairing that produced the Critical — is exercised only through hand-built `FixedZone` values, never through the real seam.

## 6. Architectural notes

**ARCH-DRY — flag.** Two live instances: BR-7's comparator tail (`queue.go:74-77` and `:80-83`, extract `byLookupsThenKey`), and the `_ "time/tzdata"` pattern from `history_cmd_test.go:16` not reused (I-a). The day-boundary shadow-sweep otherwise **passes cleanly** — I enumerated every candidate encoding in `cmd/` and `internal/`: `time.Date(…,0,0,0,0,…)` and `/86400` appear only at `clock.go:43` and `clock.go:77`, both callers in `history_cmd.go` (`:41`, `:194`) derive from `store`, and `news.go:20`'s `7*24*time.Hour` is a cache TTL, correctly not a calendar question. `/history`'s behaviour is byte-faithful: `relativeDay` already did `at.In(now.Location())` then subtracted day indices, which is precisely what `DaysBetween` now does.

**ARCH-PURE — pass.** `schedule` is pure over explicit parameters; every instant arrives as an argument; both guards are real rather than asserted, and I verified the clock guard empirically. IO stays in the future caller (`#6`). `StartOfDay`/`DaysBetween` living in the `store` package is correct — they're pure functions beside the `Clock` seam that owns the time model, and putting them in `schedule` would have inverted the dependency.

**ARCH-PURPOSE — flag, on the documentation half only.** The code fulfills the purpose. The artifact sweep does not, and the principle's own text governs this exactly: *"a finding names one instance; the deliverable is the CLASS it belongs to."* BR-3 named the class and even quoted the four-artifact enumeration from the plan's own Revisions section; the answer fixed 2 of 5 sites. This is the second consecutive round in which the class was named and not swept, and the family is on its third appearance counting PQ-6/PQ-7. Separately, **pass** on the `Mastered`-consumed-by-two deferral: `#8` is filed and open, and the Done-when records the deferral honestly rather than ticking it — that is a separable extension, not the deferred point of the issue.

**ARCH-MOCK — pass.** No external service or binary in the shipped path; `#5` adds no seam, which is the deliverable. The only `exec.Command` is `go list` inside `purity_test.go`, a meta-test with in-repo precedent (`repo_guard_test.go` shells out to `git`), and it `t.Fatalf`s rather than skipping when the toolchain is absent — which is the correct choice, and the one I-a asks the tzdata sites to make.

## 7. Plan revision recommendations

The plan needs a `## Revisions` entry for the close round; it currently stops at the two plan-quality rounds. It should carry:

1. **`unbacked-claim-about-existing-code`, third occurrence — with the sweep actually run.** Correct `:27` (which still says both "imports only `store` and `time`" *and* the "would not compile" falsehood PQ-3 already retracted) and `:33` ("stays exactly `store` + `time`"), and record that the enumeration is **five** artifacts, not four: issue, plan, atlas, **project file**, code comments. The project file (`workshop/projects/define-learn.md:485`) was missed in both rounds and is not in the enumeration the plan wrote.
2. **The mastery decision, recorded where the plan states behaviour.** The `Queue` bullet still lists only tiers/budget/ties/deck-vs-log. Add: *mastered words stay in rotation; the 90-day rung supplies the rarity; mastery is a reporting status for `#8` and a presentation cue for `#6`, never a removal — because exclusion makes mastery absorbing.* This is BR-6's "decide, then record it" half.
3. **The Core concepts table doesn't name the package's exported surface.** It lists `Box` at `box.go`, but no `Box` identifier exists there — the concept is realised as `Progress.Box int` plus `IntervalDays` and `LastBox`, and those two exported symbols (the ones `#6` and `#8` will actually import) appear in no row. Replace the `Box` row with `IntervalDays` and `LastBox`. Everything else in the table verifies at its stated path.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Verified by revert: dropping `a = a.In(b.Location())` reddens clock_test.go:130 and progress_test.go:291.
  - id: BR-2
    disposition: addressed
    note: |
      Verified by mutation: {1,3,7,13,30,90} reddens box_test.go:19; TestMultiWeekSchedule now asserts Due/!Due at each rung.
  - id: BR-3
    disposition: not-addressed
    note: |
      Two of five sites fixed (box.go:7, atlas:1142); still stale at plan.md:27, plan.md:33,
      projects/define-learn.md:485 — and this round's own fixes minted two NEW instances of the
      same class: atlas/define.md:1165 says "A mastered word leaves the rotation" (the exact
      behaviour BR-6 removed, and what #6's implementor will read), and history_cmd.go:192 says
      "store.DaysBetween steps the calendar" when the BR-1 fix replaced the stepping with UTC
      day-index subtraction. The rule this family needs is not "fix these files": it is that a
      behaviour change must GREP the invalidated claim's distinctive phrase across a fixed
      enumeration at the moment of the change, and that the enumeration has five members, not
      four — the plan's own list omits the project file, which is why :485 survived both rounds.
  - id: BR-4
    disposition: addressed
    note: |
      Verified in a real scratch copy (overlays do not reach this guard — it reads files off disk): time.Since reddens purity_test.go:100.
  - id: BR-5
    disposition: addressed
    note: |
      const over an array literal — compiler-enforced, stronger than the test that was asked for.
  - id: BR-6
    disposition: addressed
    note: |
      Verified by mutation: re-adding the exclusion reddens queue_test.go:156. The "record it" half landed in code/issue but not atlas/plan — carried under BR-3, not re-raised here.
  - id: BR-7
    disposition: not-addressed
    note: |
      queue.go:74-77 and :80-83 still share the identical lookups-then-key tail.
  - id: BR-8
    disposition: not-addressed
    note: |
      queue.go:86 unchanged; re-verified that Queue(deck, nil, now, 1<<62) panics "makeslice: cap out of range".
  - id: BR-9
    disposition: addressed
    note: |
      Incidentally fixed by BR-1 — DaysBetween is now two dayIndex subtractions, no AddDate loop.
  - id: BR-10
    disposition: not-addressed
    note: |
      queue.go:26 still says "today's words"; nothing in the doc, the atlas or a test names the returned strings as normalised store.Key values rather than deck Text.
findings:
  - id: new
    severity: Important
    family: invariant-needs-mechanical-guard
    title: |
      The only test pinning this round's Critical fix skips itself when tzdata is absent, and the repo's own fix for that was not reused
    detail: |
      This is the 2nd finding in family `invariant-needs-mechanical-guard`; BR-4 was the 1st.
      Do not fix only progress_test.go:276. THE RULE: a guard guards only if it is
      unconditionally reachable in the environment the suite runs in — an assertion that can
      skip itself is documentation. Verified by simulating LoadLocation failure in a scratch
      copy: both store and schedule report `ok` with every cross-zone and DST assertion gone,
      including TestDueDoesNotFireOnTheDayOfReview, the sole product-level guard for BR-1.
      CI is ubuntu-latest, which does ship tzdata, so it is not blind today; a distroless or
      alpine image would be. THE ENUMERATION, run: `grep -rn "t.Skip" cmd/define` gives 16
      sites. Twelve are legitimate environment gates (network, pty, afplay, no model, no word
      list). Five are tzdata: clock_test.go:20,79,100,141 and progress_test.go:276 — and FOUR
      of those five do not need tzdata at all, because they discriminate on UTC OFFSET, which
      time.FixedZone always supplies; only TestDaysBetweenAcrossDST (clock_test.go:141) needs
      a real DST-carrying zone. Prevalence 4/5. The mechanical cure already exists in-tree and
      was not reused (ARCH-DRY): cmd/define/history_cmd_test.go:16 imports `_ "time/tzdata"`.
      Add that import to the store and schedule test packages, and rebuild the four
      offset-only fixtures on FixedZone so they assert unconditionally.
```

---

## Re-review — 2026-08-26T23:44:42-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 5 — spaced-repetition scheduling engine (Leitner, pure) |
| repo | tools |
| issue file | workshop/issues/000005-vocab-schedule.md |
| boundary | whole-issue close |
| milestone | — |
| window | a30bb786560befe4960d28eaf5098a3bc678efa6..26bad6e1e6481752f1ed81143c14ba3904b74b91 |
| command | sdlc close --issue 5 |
| reviewer | claude |
| timestamp | 2026-08-26T23:44:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The scheduling engine itself is correct and genuinely well-pinned — I verified four independent mutations die (zone normalisation removed → `clock_test.go:136` + `progress_test.go:298`; box 3 interval 14→13 → `box_test.go:19`; tier order swapped → two named queue tests; mastered-exclusion re-added → `queue_test.go:156`), the full suite is green, and the `store.Word`-gains-no-fields decision holds. Nothing in the code blocks SHIP. What holds it back from a clean SHIP is entirely artifact truth: `atlas/define.md:1165` still tells the next implementor "A mastered word leaves the rotation" — the exact absorbing-state behaviour BR-6 removed — and the same round's own fixes minted two *further* false claims about the code (`Due compares through store.StartOfDay`, which it does not; `DaysBetween steps the calendar`, which it no longer does). BR-11 was half-executed: the schedule pin is properly fixed and verified, but the store test package's four comments assert `time/tzdata is embedded above` when `go list -deps -test` shows it is not imported there at all. Plus one new verified hole: the purity allowlist admits `store` wholesale, so `store.NewYAML(dir, warn)` inside `schedule` passes both guards.

## 1. Strengths

- **The BR-1 fix is pinned at both levels, and the fixture actually discriminates.** Reverting `a = a.In(b.Location())` (`store/clock.go:69`) reddens `TestDaysBetweenAcrossLocations/b's_location_defines_the_calendar` *and* the product-level `TestDueDoesNotFireOnTheDayOfReview`. The travelling-learner fixture (`progress_test.go:292`, HST stamp vs LA now) is the version that survives the mutation — the comment records that the first, same-date attempt did not.
- **`store.DaysBetween` is now arithmetic, not a walk** (`clock.go:68-78`), and its doc comment is the best in the diff: it states *why* the discarded `dayIndex` shape was the right one, so the next consolidation has the evidence the last one threw away.
- **BR-2 closed properly, not cosmetically.** `TestMultiWeekSchedule` now asserts `!Due` at `interval-1` and `Due` at `interval` for **every** rung (`progress_test.go:181-198`); mutating any middle interval reddens `box_test.go:19`.
- **`TestMasteredWordsStillComeRoundAtTheLongInterval` and `TestMasteryCanBeLost`** (`queue_test.go:139`, `:162`) pin the BR-6 reversal from both sides — the word returns at 90 days, and the status can be lost. The fixture even self-checks (`t.Fatal("fixture is wrong: … so this test asserts nothing")`).
- **`historyWindow`/`relativeDay` refactor is behaviour-identical.** `relativeDay` keeps `at = at.In(now.Location())` (still load-bearing for `at.Format("Monday")`), and `DaysBetween` re-does the same normalisation idempotently — no drift, and `/history`'s tests remain a real net for the shapes they cover.
- **Both purity guards refuse to pass vacuously** (`purity_test.go:57`, `:105`) — the failure mode most guard tests ship with.

## 2. Critical findings

None.

## 3. Important findings

**I1 — BR-3, third round: the artifact enumeration was written down again and again not run, and this round's own fixes minted two more instances.** Verified sites, all currently false:

| site | claim | truth |
|---|---|---|
| `atlas/define.md:1165` | "A mastered word leaves the rotation" | BR-6 removed exactly this; `queue.go:56` queues mastered words |
| `atlas/define.md:1129` | "`Due` compares through `store.StartOfDay`" | `Due` calls `store.DaysBetween`; `StartOfDay` has **one** production caller, `history_cmd.go:41` |
| `cmd/define/schedule/progress.go:44` | "via `store.StartOfDay`" | same |
| `cmd/define/history_cmd.go:192` | "`store.DaysBetween` steps the calendar" | BR-9's fix replaced stepping with UTC day-index subtraction |
| `workshop/projects/define-learn.md:485` | "imports `store` and `time` and nothing else" | survived rounds 1 and 2 |
| `workshop/plans/000005-…-plan.md:31` | "`Due` therefore compares `startOfDay(LastReviewed)` plus N days against `startOfDay(now)`" | it compares via `DaysBetween` |
| `workshop/plans/000005-…-plan.md:33` | "Both callers then depend on `store`" | `schedule` is not a caller of `StartOfDay` |

`atlas/define.md:1165` is the one that costs real money: `#6` is the next issue, it was prioritised *because* the operator wants a practisable loop, and its implementor reads the atlas.

**I2 — BR-11 residue: `cmd/define/store/clock_test.go` asserts an import it does not have, in four places.** `go list -deps -test github.com/xianxu/tools/cmd/define/store | grep -c tzdata` → `0`; the same command for `schedule` → `1`. The four comments at `:20`, `:81`, `:104`, `:147` each say "NOT a skip: time/tzdata is embedded above". The safety property BR-11 named *is* achieved (`t.Fatalf`, so no silent green), but on a distroless/alpine image these four go red for the wrong reason while claiming they cannot. One-line fix: `_ "time/tzdata"` in `clock_test.go`'s import block.

**I3 [new] — the purity allowlist admits `store` wholesale, so real disk IO passes both guards.** *This is the 3rd finding in family `invariant-needs-mechanical-guard`* (BR-4 1st, BR-11 2nd). Do not just patch the allowlist. Verified in a scratch copy: inserting `_ = store.NewYAML("/tmp/whatever", nil)` into `Due` leaves `TestScheduleImportsOnlyStoreAndTime` and `TestScheduleNeverReadsTheClock` both `PASS`. `purity_test.go:37` allowlists `store` by name while `:33-35` states the criterion as "`os`, `net`, `bufio`, `io` are not [pure], and anything that can name a file is not" — `store` is all of those.

The rule the three instances share: **a guard is only worth its line count once a deliberate violation is a committed artifact, not a hand-run tree mutation.** BR-4 (one banned spelling of six), BR-11 (an assertion that can skip itself) and this (an allowlist entry that admits the IO package) all passed inspection and all failed the violation. The enumeration is small and writable: the import allowlist, the clock-reader token list, `repo_guard_test.go`'s git checks, and the tzdata-reachability property — four guards, each needing one negative case. The mechanical shape: extract the decision from the IO (`func violations(imports []string) []string`, `func clockReaders(src []byte) []string`) so each negative case is a table row over a synthetic input rather than a mutation someone has to remember to run. That is also the ARCH-PURE fix for `purity_test.go`, and it is why this hole was invisible.

## 4. Minor findings

- `queue.go:70-78` and `:79-84` still share the identical lookups-then-key tail (BR-7, unchanged).
- `queue.go:86` still panics on an absurd budget (BR-8, re-verified: `makeslice: cap out of range`).
- `queue.go:10` still says "today's words"; nothing names the returned strings as normalised `store.Key` values (BR-10, unchanged).
- **[new]** `Queue`'s degenerate-input domain is stated for two parameters and untested for the rest — *2nd in family `hostile-input-at-the-seam`*, BR-8 being the 1st. The rule: `Queue` is this package's public seam for `#6`, so every parameter needs a stated and tested behaviour on degenerate input. The enumeration, run: budget ≤0 ✓, budget > len(deck) ✓, absurd budget ✗ (panics); deck empty ✓, empty `Text` ✓, **duplicate keys ✗** — verified `Queue([]store.Word{{Text:"Define"},{Text:"define"}}, …)` returns `[define define]`, the same word twice, spending two of the budget; prog nil ✓, keys absent from deck ✓; `now` zero ✗ (untested).
- **[new]** `workshop/plans/000005-vocab-schedule-plan.md` has 20 `- [ ]` steps and 0 `- [x]` after both milestones closed; the plan's own header says the checkboxes are the tracking mechanism.
- `store.StartOfDay` and `store.dayIndex` (`clock.go:41`, `:75`) are two encodings of "the local calendar date of `t`" in one file — Task 0 existed to collapse exactly that, and after the BR-1 fix the count is back at two with `StartOfDay` down to a single production caller. Defensible (different return types), worth one sentence rather than silence.

## 5. Test coverage notes

Coverage is strong where it matters and I confirmed it by mutation rather than by reading. Gaps worth naming: no test asserts `Queue`'s return values are `store.Key` output rather than deck `Text` (so BR-10 is undetectable by the suite); no test covers a deck with colliding keys; `FuzzFold` exercises one word only, so nothing fuzzes the multi-word map path; and `TestScheduleImportsOnlyStoreAndTime`/`TestScheduleNeverReadsTheClock` have no negative case, which is I3. `TestQueueIsDeterministicOnTies` running the same input 20 times does not actually exercise Go's map/sort nondeterminism, since `Queue` iterates the *deck slice*, not the map — it passes for a reason unrelated to what it claims to test; the real determinism guarantee is the total order in the comparators, and `first != "alpha,beta,gamma"` is the assertion doing the work.

## 6. Architectural notes

- **ARCH-DRY — flag.** BR-7 (duplicated comparator tail) and the `StartOfDay`/`dayIndex` pair above. `Fold` applying `Answer` and the single interval table are the wins.
- **ARCH-PURE — flag.** The product code is a clean pure core with no IO seam, which is the deliverable. The flag is on the *guard*: `purity_test.go` fuses its decision with `exec.Command`/`os.ReadDir`, so it can only be validated by mutating the real tree — see I3.
- **ARCH-PURPOSE — flag.** The engine fulfils the issue. The shadow-sweep over the model's consumers is what fails: `atlas/define.md` is the derived artifact `#6` and `#8` read, and it restates the model with a reversed decision (I1). BR-6 was fixed in code and issue but not in the artifact that carries the decision forward — the instance, not the class.
- **ARCH-MOCK — pass.** No external binary or service in production code; `#6` supplies deck, log and clock through the existing `store` seams. The guard tests shell out to `go`, consistent with `repo_guard_test.go`'s `git` — the toolchain rather than a modelled dependency, so no fake is owed.

## 7. Plan revision recommendations

Add a `## Revisions` entry — "2026-08-26 — close round 3":

- **`Due` does not use `store.StartOfDay`.** Lines 31 and 33 still describe `Due` comparing `startOfDay(LastReviewed) + N` against `startOfDay(now)` and call `schedule` one of `StartOfDay`'s "both callers". The implementation compares `store.DaysBetween(p.LastReviewed, now) >= IntervalDays(p.Box)`; `StartOfDay` has one production caller, `history_cmd.go:41`. Record the actual shape and that Task 0's collapse left two helpers in `clock.go`, not one.
- **The BR-3 enumeration has five members and a trigger, and neither has held for three rounds.** Record that the sweep must be keyed to *the behaviour change*, not to the finding's wording, and that the two artifacts it keeps missing are the project file and the atlas — the two nobody edits while writing code.
- **Tick the 20 completed steps**, or state explicitly that the issue file's `## Plan` is the record of truth and the durable plan's checkboxes are not maintained.

```findings
dispose:
  - id: BR-3
    disposition: not-addressed
    note: |
      Seven false-claim sites remain, two minted by this round's own fixes; atlas/define.md:1165 still documents the behaviour BR-6 removed.
  - id: BR-7
    disposition: not-addressed
    note: |
      queue.go:70-78 and :79-84 unchanged; identical lookups-then-key tail.
  - id: BR-8
    disposition: not-addressed
    note: |
      queue.go:86 unchanged; re-verified the makeslice panic on budget 1<<62.
  - id: BR-10
    disposition: not-addressed
    note: |
      queue.go:10 still says "today's words"; no doc, atlas line or test names the store.Key contract.
  - id: BR-11
    disposition: not-addressed
    note: |
      Schedule pin fully fixed and mutation-verified; store test package still lacks the tzdata import its four comments claim it has.
findings:
  - id: new
    severity: Important
    family: invariant-needs-mechanical-guard
    title: |
      The purity allowlist admits store wholesale, so a real disk-IO constructor inside schedule passes both guards
    detail: |
      3rd in this family (BR-4 1st, BR-11 2nd). Verified in a scratch copy: inserting
      `_ = store.NewYAML("/tmp/whatever", nil)` into Due leaves both purity tests PASS,
      though purity_test.go:33-35 states the criterion as "anything that can name a file"
      is not pure and store is exactly that. Do not just patch the allowlist. THE RULE the
      three instances share - a guard is worth its line count only once a deliberate
      violation is a COMMITTED artifact rather than a hand-run tree mutation. THE
      ENUMERATION, four guards - the import allowlist, the clock-reader token list,
      repo_guard_test.go's git checks, and the tzdata-reachability property - each needing
      one negative case. Mechanical shape - extract the decision from the IO
      (violations(imports []string), clockReaders(src []byte)) so each negative case is a
      table row. That is also the ARCH-PURE fix for purity_test.go, and it is why this
      hole was invisible to inspection.
  - id: new
    severity: Minor
    family: hostile-input-at-the-seam
    title: |
      Queue's degenerate-input domain is stated for two parameters and untested for the rest; duplicate deck keys return the same word twice
    detail: |
      2nd in this family, BR-8 being the 1st. THE RULE - Queue is this package's public
      seam for #6, so every parameter needs a stated and tested behaviour on degenerate
      input. THE ENUMERATION, run - budget: <=0 ok, > len(deck) ok, absurd PANICS (BR-8);
      deck: empty ok, empty Text ok, DUPLICATE KEYS unhandled - verified that
      Queue([]store.Word{{Text:"Define"},{Text:"define"}}, nil, now, 10) returns
      [define define], the same word twice, spending two of the budget; prog: nil ok,
      keys absent from deck ok; now: zero untested. Prevalence 2/9.
  - id: new
    severity: Minor
    family: plan-record-staleness
    title: |
      The durable plan has 20 unchecked steps and 0 checked after both milestones closed
    detail: |
      workshop/plans/000005-vocab-schedule-plan.md - its own header says the checkbox
      syntax is the tracking mechanism, and the plan is the version-controlled record of
      truth per AGENTS.md section 1. The issue file's Plan section ticks M1/M2, which is
      what the close gate reads, so the plan silently stopped being a record.
```

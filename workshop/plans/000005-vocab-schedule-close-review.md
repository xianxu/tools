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

---
gate: boundary-review
issue: 8
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-07T23:56:55-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Removing -stats from run()'s modes slice leaves the whole cmd/define suite green
          detail: |-
            Verified by mutation on a git-initialised scratch copy of HEAD: deleting
            {"-stats", *statsFlag} from main.go:609 passes the full package (109s), so the
            registration the plan's Task 5 Step 2b promised to mutation-check is unpinned.
            declaredModes' floor is `len(names) < 5` (harvest_test.go:837) while run() now
            declares six, and TestRunRefusesTwoModes (harvest_test.go:677) hand-lists pairs
            that omit -stats. Raise the floor to 6 and add a {"-stats","-play"} row. Shipped
            behaviour is correct — -stats -play exits 2 in the tree as committed.
          family: hand-maintained-extent
          round: 1
        - id: BR-2
          severity: Important
          title: Plan names an entity the code never declares and states the wrong nil-deck exit code
          detail: |-
            workshop/plans/000008-vocab-stats-plan.md:82 lists `activeDays` as new at
            cmd/define/schedule/stats.go; no such symbol exists in the tree, and the helpers
            actually written (countable, streaks) appear in no row. Line 140 says a nil deck
            exits 0; cmd/define/stats.go:43 returns 1. The ARCH-DRY table also claims streaks
            uses store.DaysBetween, which it deliberately does not.
          family: plan-table-drift
          round: 1
        - id: BR-3
          severity: Important
          title: Unticked plan steps suppress TestPlanTablesNameEntitiesThatExist at the boundary
          detail: |-
            repo_guard_test.go:797 skips `new` rows while the plan contains "- [ ] ". Every
            task step is still unticked at HEAD though the work is done, so the guard that
            exists to catch the finding above is off at exactly the gate it was written for.
            Ticking them in a scratch copy makes it fail naming "activeDays" verbatim. Tick
            the plan's steps as part of crossing the boundary.
          family: plan-table-drift
          round: 1
        - id: BR-4
          severity: Important
          title: runStats and runStatsCommand duplicate the deck/log/clock read the comment says is shared
          detail: |-
            cmd/define/stats.go:36 and :198 share only renderStats(Summarise(...)); the Deck()
            read, Events(time.Time{}) read, clock read and print loop are copies with divergent
            error phrasing. The plan names --stats --days 30 as the next extension, which would
            land in one door only, and the same-output test uses one fixture that a divergent
            `since` would not necessarily separate. Extract
            statsLines(deck store.Store, clk store.Clock) ([]string, error) — both deps and
            commandCtx already carry those two types (ARCH-DRY).
          family: duplicated-read-path
          round: 1
        - id: BR-5
          severity: Minor
          title: README's sample screen prints "3 January"; the code prints "Jan 3"
          detail: |-
            cmd/define/README.md:407. relativeDay (history_cmd.go:207) formats "Jan 2". Verified
            by rendering the README's exact Stats values — every other line matches byte-for-byte.
          family: docs-restate-unverified-output
          round: 1
        - id: BR-6
          severity: Minor
          title: Fold runs on unfiltered events while every other figure goes through countable
          detail: |-
            cmd/define/schedule/stats.go:85. A zero-At or future-dated hand-edited reviewed event
            is excluded from Accuracy and ActiveDays but still promotes a word toward Mastered.
            This is probably correct — filtering would make the screen disagree with --play's
            queue — but the doc comment claims the fold validates rather than trusts, without
            naming the exception, and nothing pins it.
          family: untrusted-persisted-input
          round: 1
        - id: BR-7
          severity: Minor
          title: README's mode list is a hand-maintained restatement of run()'s modes slice
          detail: |-
            cmd/define/README.md:521. Same family declaredModes just fixed for the test. A
            doc_sync_test.go-shaped guard reusing declaredModes is nearly free.
          family: hand-maintained-extent
          round: 1
        - id: BR-8
          severity: Minor
          title: runStats takes a context.Context it never uses
          detail: |-
            cmd/define/stats.go:36. Deliberate — it mirrors runReflect's signature — but worth
            `_ context.Context` or one line saying why.
          family: unused-parameter
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-08T00:06:24-07:00"
      agent: claude
      findings:
        - id: BR-9
          severity: Critical
          title: streaks breaks a run across any DST transition that occurs at local midnight
          detail: |-
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
          family: local-calendar-arithmetic
          round: 2
        - id: BR-10
          severity: Minor
          title: formLabel sanitises at render, so two distinct form keys can print as identical rows
          detail: |-
            cmd/define/stats.go:174. Accuracy is keyed by the raw Form string, and control runes are
            stripped only when the label is drawn, so "meaning" and "meaning\x1b[2J" produce two rows
            that read the same. Neutralising is right; collapsing at the key or disambiguating the
            label would keep the screen honest about what the log actually holds.
          family: untrusted-persisted-input
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-08T00:21:13-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Mutation-verified on a scratch clone at HEAD: deleting -stats reddens harvest_test.go:946 by name, and so does deleting -reflect.'
          round: 3
        - id: BR-2
          disposition: not-addressed
          note: activeDays row and exit code fixed; plan:56 and Task 2 Steps 3/5 still claim store.DaysBetween is used, and countable/streaks/printStats are in no row.
          round: 3
        - id: BR-3
          disposition: addressed
          note: Every step is ticked, so TestPlanTablesNameEntitiesThatExist now runs; it just cannot reach the ARCH-DRY row that keeps BR-2 open.
          round: 3
        - id: BR-4
          disposition: addressed
          note: printStats at stats.go:63; both doors call it (:48, :228). No test reddens on revert — inherent to a de-duplication fix, verified structurally.
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: README:407 still "3 January"; re-rendered the README's exact values at HEAD — every other line byte-identical, that one is "Jan 3".
          round: 3
        - id: BR-6
          disposition: not-addressed
          note: Fold(events) at stats.go:85 is still unfiltered, the doc comment still claims validation without naming the exception, nothing pins it.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: README:521 still hand-lists the six modes; no doc_sync_test.go guard reuses declaredModes.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: runStats still takes an unused ctx at stats.go:37 with no `_ context.Context` and no line saying why.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: 'Reproduced at HEAD in three zones. Note the finding''s first fix sketch is wrong: StartOfDay-wrapping collapses Havana''s Mar 8 key onto Mar 7. Civil-date/UTC-day-index keying is the only correct option.'
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: Accuracy is still keyed by the raw Form string with neutralisation only at formLabel (stats.go:190).
          round: 3
      findings:
        - id: BR-11
          severity: Important
          title: The dispatch derivation covers five of six modes and calibrates against three hand-typed floors
          detail: |-
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
          family: hand-maintained-extent
          round: 3
        - id: BR-12
          severity: Important
          title: The closing commit deleted issue 48's durable plan, leaving a working issue pointing at a file no branch holds
          detail: |-
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
          family: artifact-deleted-to-satisfy-guard
          round: 3
      blocked: true
---

# Gate ledger — tools#8 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-07T23:56:55-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `hand-maintained-extent` Removing -stats from run()'s modes slice leaves the whole cmd/define suite green
  Verified by mutation on a git-initialised scratch copy of HEAD: deleting
  {"-stats", *statsFlag} from main.go:609 passes the full package (109s), so the
  registration the plan's Task 5 Step 2b promised to mutation-check is unpinned.
  declaredModes' floor is `len(names) < 5` (harvest_test.go:837) while run() now
  declares six, and TestRunRefusesTwoModes (harvest_test.go:677) hand-lists pairs
  that omit -stats. Raise the floor to 6 and add a {"-stats","-play"} row. Shipped
  behaviour is correct — -stats -play exits 2 in the tree as committed.
- **BR-2** [Important] `plan-table-drift` Plan names an entity the code never declares and states the wrong nil-deck exit code
  workshop/plans/000008-vocab-stats-plan.md:82 lists `activeDays` as new at
  cmd/define/schedule/stats.go; no such symbol exists in the tree, and the helpers
  actually written (countable, streaks) appear in no row. Line 140 says a nil deck
  exits 0; cmd/define/stats.go:43 returns 1. The ARCH-DRY table also claims streaks
  uses store.DaysBetween, which it deliberately does not.
- **BR-3** [Important] `plan-table-drift` Unticked plan steps suppress TestPlanTablesNameEntitiesThatExist at the boundary
  repo_guard_test.go:797 skips `new` rows while the plan contains "- [ ] ". Every
  task step is still unticked at HEAD though the work is done, so the guard that
  exists to catch the finding above is off at exactly the gate it was written for.
  Ticking them in a scratch copy makes it fail naming "activeDays" verbatim. Tick
  the plan's steps as part of crossing the boundary.
- **BR-4** [Important] `duplicated-read-path` runStats and runStatsCommand duplicate the deck/log/clock read the comment says is shared
  cmd/define/stats.go:36 and :198 share only renderStats(Summarise(...)); the Deck()
  read, Events(time.Time{}) read, clock read and print loop are copies with divergent
  error phrasing. The plan names --stats --days 30 as the next extension, which would
  land in one door only, and the same-output test uses one fixture that a divergent
  `since` would not necessarily separate. Extract
  statsLines(deck store.Store, clk store.Clock) ([]string, error) — both deps and
  commandCtx already carry those two types (ARCH-DRY).
- **BR-5** [Minor] `docs-restate-unverified-output` README's sample screen prints "3 January"; the code prints "Jan 3"
  cmd/define/README.md:407. relativeDay (history_cmd.go:207) formats "Jan 2". Verified
  by rendering the README's exact Stats values — every other line matches byte-for-byte.
- **BR-6** [Minor] `untrusted-persisted-input` Fold runs on unfiltered events while every other figure goes through countable
  cmd/define/schedule/stats.go:85. A zero-At or future-dated hand-edited reviewed event
  is excluded from Accuracy and ActiveDays but still promotes a word toward Mastered.
  This is probably correct — filtering would make the screen disagree with --play's
  queue — but the doc comment claims the fold validates rather than trusts, without
  naming the exception, and nothing pins it.
- **BR-7** [Minor] `hand-maintained-extent` README's mode list is a hand-maintained restatement of run()'s modes slice
  cmd/define/README.md:521. Same family declaredModes just fixed for the test. A
  doc_sync_test.go-shaped guard reusing declaredModes is nearly free.
- **BR-8** [Minor] `unused-parameter` runStats takes a context.Context it never uses
  cmd/define/stats.go:36. Deliberate — it mirrors runReflect's signature — but worth
  `_ context.Context` or one line saying why.

## Round 2 — 2026-09-08T00:06:24-07:00 (claude) — BLOCKED

### Raised

- **BR-9** [Critical] `local-calendar-arithmetic` streaks breaks a run across any DST transition that occurs at local midnight
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
- **BR-10** [Minor] `untrusted-persisted-input` formLabel sanitises at render, so two distinct form keys can print as identical rows
  cmd/define/stats.go:174. Accuracy is keyed by the raw Form string, and control runes are
  stripped only when the label is drawn, so "meaning" and "meaning\x1b[2J" produce two rows
  that read the same. Neutralising is right; collapsing at the key or disambiguating the
  label would keep the screen honest about what the log actually holds.

## Round 3 — 2026-09-08T00:21:13-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Mutation-verified on a scratch clone at HEAD: deleting -stats reddens harvest_test.go:946 by name, and so does deleting -reflect.
- BR-2 — not-addressed — activeDays row and exit code fixed; plan:56 and Task 2 Steps 3/5 still claim store.DaysBetween is used, and countable/streaks/printStats are in no row.
- BR-3 — addressed — Every step is ticked, so TestPlanTablesNameEntitiesThatExist now runs; it just cannot reach the ARCH-DRY row that keeps BR-2 open.
- BR-4 — addressed — printStats at stats.go:63; both doors call it (:48, :228). No test reddens on revert — inherent to a de-duplication fix, verified structurally.
- BR-5 — not-addressed — README:407 still "3 January"; re-rendered the README's exact values at HEAD — every other line byte-identical, that one is "Jan 3".
- BR-6 — not-addressed — Fold(events) at stats.go:85 is still unfiltered, the doc comment still claims validation without naming the exception, nothing pins it.
- BR-7 — not-addressed — README:521 still hand-lists the six modes; no doc_sync_test.go guard reuses declaredModes.
- BR-8 — not-addressed — runStats still takes an unused ctx at stats.go:37 with no `_ context.Context` and no line saying why.
- BR-9 — not-addressed — Reproduced at HEAD in three zones. Note the finding's first fix sketch is wrong: StartOfDay-wrapping collapses Havana's Mar 8 key onto Mar 7. Civil-date/UTC-day-index keying is the only correct option.
- BR-10 — not-addressed — Accuracy is still keyed by the raw Form string with neutralisation only at formLabel (stats.go:190).

### Raised

- **BR-11** [Important] `hand-maintained-extent` The dispatch derivation covers five of six modes and calibrates against three hand-typed floors
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
- **BR-12** [Important] `artifact-deleted-to-satisfy-guard` The closing commit deleted issue 48's durable plan, leaving a working issue pointing at a file no branch holds
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

## Open findings

- **BR-2** [Important] `plan-table-drift` Plan names an entity the code never declares and states the wrong nil-deck exit code
- **BR-5** [Minor] `docs-restate-unverified-output` README's sample screen prints "3 January"; the code prints "Jan 3"
- **BR-6** [Minor] `untrusted-persisted-input` Fold runs on unfiltered events while every other figure goes through countable
- **BR-7** [Minor] `hand-maintained-extent` README's mode list is a hand-maintained restatement of run()'s modes slice
- **BR-8** [Minor] `unused-parameter` runStats takes a context.Context it never uses
- **BR-9** [Critical] `local-calendar-arithmetic` streaks breaks a run across any DST transition that occurs at local midnight
- **BR-10** [Minor] `untrusted-persisted-input` formLabel sanitises at render, so two distinct form keys can print as identical rows
- **BR-11** [Important] `hand-maintained-extent` The dispatch derivation covers five of six modes and calibrates against three hand-typed floors
- **BR-12** [Important] `artifact-deleted-to-satisfy-guard` The closing commit deleted issue 48's durable plan, leaving a working issue pointing at a file no branch holds

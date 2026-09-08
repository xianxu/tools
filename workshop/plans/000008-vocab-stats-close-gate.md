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

## Open findings

- **BR-1** [Important] `hand-maintained-extent` Removing -stats from run()'s modes slice leaves the whole cmd/define suite green
- **BR-2** [Important] `plan-table-drift` Plan names an entity the code never declares and states the wrong nil-deck exit code
- **BR-3** [Important] `plan-table-drift` Unticked plan steps suppress TestPlanTablesNameEntitiesThatExist at the boundary
- **BR-4** [Important] `duplicated-read-path` runStats and runStatsCommand duplicate the deck/log/clock read the comment says is shared
- **BR-5** [Minor] `docs-restate-unverified-output` README's sample screen prints "3 January"; the code prints "Jan 3"
- **BR-6** [Minor] `untrusted-persisted-input` Fold runs on unfiltered events while every other figure goes through countable
- **BR-7** [Minor] `hand-maintained-extent` README's mode list is a hand-maintained restatement of run()'s modes slice
- **BR-8** [Minor] `unused-parameter` runStats takes a context.Context it never uses
- **BR-9** [Critical] `local-calendar-arithmetic` streaks breaks a run across any DST transition that occurs at local midnight
- **BR-10** [Minor] `untrusted-persisted-input` formLabel sanitises at render, so two distinct form keys can print as identical rows

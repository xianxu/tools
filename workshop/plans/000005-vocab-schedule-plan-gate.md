---
gate: plan-quality
issue: 5
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-26T22:34:00-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Plan never says whether a Leitner interval is N x 24h or N local calendar days
          detail: |-
            atlas/define.md:640 already settled this class for /history — "local
            midnights, AddDate(0,0,-(days-1)) ... never now - Nx24h", because a DST day
            is 23 or 25 hours. The answer fixes the interval table's type and Due's
            comparison, so it cannot be deferred to implementation.
          family: local-calendar-day-boundary
          round: 1
        - id: PQ-2
          severity: Important
          title: Fold and Answer will each encode the box transition; plan never links them
          detail: |-
            ARCH-DRY. Folding a log is applying Answer per EventReviewed. State that
            Fold is a reduce over Answer and that Answer is the only site of box/streak
            arithmetic, or the mutation-check in Task 2 Step 7 covers one copy of two.
          family: single-source-of-derivation
          round: 1
        - id: PQ-3
          severity: Important
          title: Purity Done-when has no guard, and "a test needing a fake would not compile" is false
          detail: |-
            Go imposes no import restriction on a subpackage; schedule's own _test.go
            may import anything. Reuse the repo idiom (atlas/repo-guards.md,
            cmd/define/repo_guard_test.go:156) and name a test asserting the package's
            import set is store + time.
          family: invariant-needs-mechanical-guard
          round: 1
        - id: PQ-4
          severity: Important
          title: FuzzFold's permutation invariant is false at tied timestamps, and its rationale misstates Events()
          detail: |-
            store.Store.Events documents chronological order and both impls sort stably
            on At (mem.go:95, yaml.go:203), so events do not arrive "in file order". Two
            EventReviewed sharing an At with differing Correct fold differently under
            permutation and ReviewEvent has no tiebreaker. Assert box-in-range and
            idempotence instead, and say what Fold's ordering contract is.
          family: invariant-total-over-domain
          round: 1
        - id: PQ-5
          severity: Minor
          title: masteryStreak has no value and Progress.Reviews has no stated reader
          detail: |-
            The Spec's "N consecutive correct" still has no N. Reviews is declared in
            Progress with no consumer named in this issue.
          family: undefined-constant
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-26T22:37:55-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Interval is a local calendar day, AddDate semantics, compared in the clock's location.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Fold applies Answer; a one-event fold equals Answer on zero Progress.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: TestScheduleImportsOnlyStoreAndTime named as the mechanical guard.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Ordering contract stated; box-in-range, non-negative streak, idempotence replace permutation.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: masteryStreak = 7 with a derivation; Progress.Reviews dropped for want of a reader.
          round: 2
      findings:
        - id: PQ-6
          severity: Important
          title: Plan reuses a startOfDay helper that does not exist, and neither names the shared helper's home nor has a step that moves it
          detail: |-
            No startOfDay exists anywhere in the tree; history_cmd.go:41 inlines
            time.Date(y,m,d,0,0,0,0,now.Location()) inside historyWindow and
            history_cmd.go:197 has a separate dayIndex closure in relativeDay. M1 Task 2
            Step 1, cited as the mover, only says "write the failing tests". The purity
            guard forbids a third package, so the helper must land in store or in
            schedule with cmd/define importing it back - an ownership decision the plan
            does not make (ARCH-DRY). Class sweep of claims about existing code: two
            others are also wrong - repo_guard_test.go uses git over the index and
            history, not go list, and issue 8 exists as an open issue file rather than
            not existing. Rule: every claim about existing code carries a file:line or
            is rewritten as the intent it stood in for.
          family: unbacked-claim-about-existing-code
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-26T22:40:52-07:00"
      agent: claude
      dispose:
        - id: PQ-6
          disposition: addressed
          note: 'Verified: no StartOfDay in tree, history_cmd.go:41-42 and :195-197 cited correctly, repo_guard_test.go:46/:82 runs git, issue 8 exists; helper ownership decided as store.StartOfDay with a stated reason.'
          round: 3
      findings:
        - id: PQ-7
          severity: Minor
          title: The rule PQ-6 produced was applied to the plan file but not swept over the issue file, which still asserts that issue 8 does not exist
          detail: |-
            This is the 2nd finding in family unbacked-claim-about-existing-code. Do NOT fix
            only the line. The rule is already stated in the plan's Revisions - every claim
            about existing code carries a file:line or is rewritten as the intent it stood
            in for - and what is missing is the enumeration it implies. The enumeration for
            this issue is small and complete: the issue file plus the plan file. The plan
            file was corrected; workshop/issues/000005-vocab-schedule.md:40 still reads
            "#8 does not exist", contradicted by workshop/issues/000008-vocab-stats.md and by
            the plan's own Risks section. Measured prevalence of the family: 4 claims flagged
            across two rounds, 3 fixed in the plan file, 1 left standing in the issue file -
            the residue is exactly the artifact the sweep did not enumerate. Rewrite that
            Done-when row as the intent it stood in for (#8 is filed but unbuilt, so this
            issue delivers the single definition and #8 is where "used by both" becomes true)
            and state the two-artifact enumeration in the plan so the next round has it.
          family: unbacked-claim-about-existing-code
          round: 3
      blocked: false
    - "n": 4
      timestamp: "2026-08-26T22:43:55-07:00"
      agent: claude
      dispose:
        - id: PQ-7
          disposition: addressed
          note: Issue Done-when rewritten as intent; plan's Revisions states the four-artifact sweep enumeration.
          round: 4
      blocked: false
content_hash: 63c28ba14181f3573e05c3f83fa1c11997b47decfe98288801836dfbef7266e4
---

# Gate ledger — tools#5 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-26T22:34:00-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `local-calendar-day-boundary` Plan never says whether a Leitner interval is N x 24h or N local calendar days
  atlas/define.md:640 already settled this class for /history — "local
  midnights, AddDate(0,0,-(days-1)) ... never now - Nx24h", because a DST day
  is 23 or 25 hours. The answer fixes the interval table's type and Due's
  comparison, so it cannot be deferred to implementation.
- **PQ-2** [Important] `single-source-of-derivation` Fold and Answer will each encode the box transition; plan never links them
  ARCH-DRY. Folding a log is applying Answer per EventReviewed. State that
  Fold is a reduce over Answer and that Answer is the only site of box/streak
  arithmetic, or the mutation-check in Task 2 Step 7 covers one copy of two.
- **PQ-3** [Important] `invariant-needs-mechanical-guard` Purity Done-when has no guard, and "a test needing a fake would not compile" is false
  Go imposes no import restriction on a subpackage; schedule's own _test.go
  may import anything. Reuse the repo idiom (atlas/repo-guards.md,
  cmd/define/repo_guard_test.go:156) and name a test asserting the package's
  import set is store + time.
- **PQ-4** [Important] `invariant-total-over-domain` FuzzFold's permutation invariant is false at tied timestamps, and its rationale misstates Events()
  store.Store.Events documents chronological order and both impls sort stably
  on At (mem.go:95, yaml.go:203), so events do not arrive "in file order". Two
  EventReviewed sharing an At with differing Correct fold differently under
  permutation and ReviewEvent has no tiebreaker. Assert box-in-range and
  idempotence instead, and say what Fold's ordering contract is.
- **PQ-5** [Minor] `undefined-constant` masteryStreak has no value and Progress.Reviews has no stated reader
  The Spec's "N consecutive correct" still has no N. Reviews is declared in
  Progress with no consumer named in this issue.

## Round 2 — 2026-08-26T22:37:55-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — Interval is a local calendar day, AddDate semantics, compared in the clock's location.
- PQ-2 — addressed — Fold applies Answer; a one-event fold equals Answer on zero Progress.
- PQ-3 — addressed — TestScheduleImportsOnlyStoreAndTime named as the mechanical guard.
- PQ-4 — addressed — Ordering contract stated; box-in-range, non-negative streak, idempotence replace permutation.
- PQ-5 — addressed — masteryStreak = 7 with a derivation; Progress.Reviews dropped for want of a reader.

### Raised

- **PQ-6** [Important] `unbacked-claim-about-existing-code` Plan reuses a startOfDay helper that does not exist, and neither names the shared helper's home nor has a step that moves it
  No startOfDay exists anywhere in the tree; history_cmd.go:41 inlines
  time.Date(y,m,d,0,0,0,0,now.Location()) inside historyWindow and
  history_cmd.go:197 has a separate dayIndex closure in relativeDay. M1 Task 2
  Step 1, cited as the mover, only says "write the failing tests". The purity
  guard forbids a third package, so the helper must land in store or in
  schedule with cmd/define importing it back - an ownership decision the plan
  does not make (ARCH-DRY). Class sweep of claims about existing code: two
  others are also wrong - repo_guard_test.go uses git over the index and
  history, not go list, and issue 8 exists as an open issue file rather than
  not existing. Rule: every claim about existing code carries a file:line or
  is rewritten as the intent it stood in for.

## Round 3 — 2026-08-26T22:40:52-07:00 (claude) — passed

### Disposed

- PQ-6 — addressed — Verified: no StartOfDay in tree, history_cmd.go:41-42 and :195-197 cited correctly, repo_guard_test.go:46/:82 runs git, issue 8 exists; helper ownership decided as store.StartOfDay with a stated reason.

### Raised

- **PQ-7** [Minor] `unbacked-claim-about-existing-code` The rule PQ-6 produced was applied to the plan file but not swept over the issue file, which still asserts that issue 8 does not exist
  This is the 2nd finding in family unbacked-claim-about-existing-code. Do NOT fix
  only the line. The rule is already stated in the plan's Revisions - every claim
  about existing code carries a file:line or is rewritten as the intent it stood
  in for - and what is missing is the enumeration it implies. The enumeration for
  this issue is small and complete: the issue file plus the plan file. The plan
  file was corrected; workshop/issues/000005-vocab-schedule.md:40 still reads
  "#8 does not exist", contradicted by workshop/issues/000008-vocab-stats.md and by
  the plan's own Risks section. Measured prevalence of the family: 4 claims flagged
  across two rounds, 3 fixed in the plan file, 1 left standing in the issue file -
  the residue is exactly the artifact the sweep did not enumerate. Rewrite that
  Done-when row as the intent it stood in for (#8 is filed but unbuilt, so this
  issue delivers the single definition and #8 is where "used by both" becomes true)
  and state the two-artifact enumeration in the plan so the next round has it.

## Round 4 — 2026-08-26T22:43:55-07:00 (claude) — passed

### Disposed

- PQ-7 — addressed — Issue Done-when rewritten as intent; plan's Revisions states the four-artifact sweep enumeration.

## Open findings

(none — every finding has been disposed)

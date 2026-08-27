---
gate: boundary-review
issue: 5
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-26T23:12:03-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: store.DaysBetween miscounts by a day when its arguments carry different UTC offsets, so Due fires on the review day itself
          detail: |-
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
          family: local-calendar-day-boundary
          round: 1
        - id: BR-2
          severity: Important
          title: Three of the six Leitner intervals are asserted by no test, and the plan promised a due-date assertion at each step
          detail: |-
            box_test.go pins box 0 and LastBox plus strict increase; TestDue pins box 1.
            Boxes 2, 3 and 4 (7, 14, 30) are unpinned — verified with go test -overlay that
            mutating the table to {1,3,7,13,30,90} leaves the whole suite green. Plan Task 2
            Step 2 says the multi-week test asserts "the due date at each step";
            progress_test.go:162 walks the rungs but asserts dueness only at the endpoint.
            Asserting Due/!Due at each rung closes both the plan item and the mutation.
          family: spec-constant-unpinned
          round: 1
        - id: BR-3
          severity: Important
          title: '"imports store and time and nothing else" is now false in four artifacts; the enumeration the plan wrote was not run when the guard rule changed'
          detail: |-
            M2 rewrote the guard to "no IO and no hidden clock" and queue.go imports sort,
            but the two-imports claim still stands at cmd/define/schedule/box.go:7,
            atlas/define.md:1142, workshop/projects/define-learn.md:485 and
            workshop/plans/000005-vocab-schedule-plan.md:27,33,62. The plan's own Revisions
            section names the sweep — issue file, plan file, atlas, code comments — and the
            M2 commit changed the rule without running it. Third instance of the family the
            plan gate raised as PQ-6 and PQ-7.
          family: unbacked-claim-about-existing-code
          round: 1
        - id: BR-4
          severity: Important
          title: The clock guard greps only "time.Now(" and misses time.Since, time.Until, time.After and the timer constructors
          detail: |-
            cmd/define/schedule/purity_test.go:91 is a single strings.Contains. Verified by
            overlay: inserting `_ = time.Since(p.LastReviewed)` into Due leaves BOTH purity
            guards passing, though time.Since reads the wall clock and is the most plausible
            edit anyone would make here. Loop over a banned-token slice instead.
          family: invariant-needs-mechanical-guard
          round: 1
        - id: BR-5
          severity: Important
          title: LastBox is an exported mutable package variable that every clamp and Mastered depends on
          detail: |-
            cmd/define/schedule/box.go:30. This package is about to be imported by #6 and #8,
            so the surface should be stable. Verified cheap fix: declare the table as an array
            (`var intervalDays = [...]int{...}`), which makes len a constant expression, then
            `const LastBox = len(intervalDays) - 1`.
          family: mutable-exported-surface
          round: 1
        - id: BR-6
          severity: Important
          title: Queue silently drops mastered words — absent from Spec and plan, contradicting progress.go's own comment, and it is an absorbing state
          detail: |-
            cmd/define/schedule/queue.go:46. The plan's Queue bullet covers tiers, budget,
            ties and the deck-vs-log rule but never mastery exclusion, and progress.go:61 says
            "#6's --play decides what to stop offering" while Queue decides. Because a
            mastered word is never offered it can never be answered wrong, so mastery is
            permanent with no path back into rotation. Decide, then record it.
          family: undeclared-behavior
          round: 1
        - id: BR-7
          severity: Minor
          title: The two sort.Slice comparators in Queue share an identical lookups-then-key tail
          detail: cmd/define/schedule/queue.go:65 and :74 — extract byLookupsThenKey (ARCH-DRY).
          family: duplicated-comparator
          round: 1
        - id: BR-8
          severity: Minor
          title: Queue panics on an absurd budget where it deliberately handles budget <= 0
          detail: |-
            cmd/define/schedule/queue.go:81 — make([]string, 0, budget) panics
            "makeslice: cap out of range" (verified). Cap the capacity at len(deck); it is
            also the tighter allocation.
          family: hostile-input-at-the-seam
          round: 1
        - id: BR-9
          severity: Minor
          title: DaysBetween steps one AddDate per day, so Due on a years-old stamp loops thousands of times per word
          detail: |-
            cmd/define/store/clock.go:53. Correct, and fine at current scale; noted for the
            first profile that shows it.
          family: linear-scan-where-arithmetic-suffices
          round: 1
        - id: BR-10
          severity: Minor
          title: Queue returns normalised keys rather than deck Text, and the doc comment says "today's words"
          detail: |-
            cmd/define/schedule/queue.go:34 — name the key-vs-text contract explicitly, since
            #6 has to map back to render.
          family: undeclared-behavior
          round: 1
      blocked: true
---

# Gate ledger — tools#5 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-26T23:12:03-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `local-calendar-day-boundary` store.DaysBetween miscounts by a day when its arguments carry different UTC offsets, so Due fires on the review day itself
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
- **BR-2** [Important] `spec-constant-unpinned` Three of the six Leitner intervals are asserted by no test, and the plan promised a due-date assertion at each step
  box_test.go pins box 0 and LastBox plus strict increase; TestDue pins box 1.
  Boxes 2, 3 and 4 (7, 14, 30) are unpinned — verified with go test -overlay that
  mutating the table to {1,3,7,13,30,90} leaves the whole suite green. Plan Task 2
  Step 2 says the multi-week test asserts "the due date at each step";
  progress_test.go:162 walks the rungs but asserts dueness only at the endpoint.
  Asserting Due/!Due at each rung closes both the plan item and the mutation.
- **BR-3** [Important] `unbacked-claim-about-existing-code` "imports store and time and nothing else" is now false in four artifacts; the enumeration the plan wrote was not run when the guard rule changed
  M2 rewrote the guard to "no IO and no hidden clock" and queue.go imports sort,
  but the two-imports claim still stands at cmd/define/schedule/box.go:7,
  atlas/define.md:1142, workshop/projects/define-learn.md:485 and
  workshop/plans/000005-vocab-schedule-plan.md:27,33,62. The plan's own Revisions
  section names the sweep — issue file, plan file, atlas, code comments — and the
  M2 commit changed the rule without running it. Third instance of the family the
  plan gate raised as PQ-6 and PQ-7.
- **BR-4** [Important] `invariant-needs-mechanical-guard` The clock guard greps only "time.Now(" and misses time.Since, time.Until, time.After and the timer constructors
  cmd/define/schedule/purity_test.go:91 is a single strings.Contains. Verified by
  overlay: inserting `_ = time.Since(p.LastReviewed)` into Due leaves BOTH purity
  guards passing, though time.Since reads the wall clock and is the most plausible
  edit anyone would make here. Loop over a banned-token slice instead.
- **BR-5** [Important] `mutable-exported-surface` LastBox is an exported mutable package variable that every clamp and Mastered depends on
  cmd/define/schedule/box.go:30. This package is about to be imported by #6 and #8,
  so the surface should be stable. Verified cheap fix: declare the table as an array
  (`var intervalDays = [...]int{...}`), which makes len a constant expression, then
  `const LastBox = len(intervalDays) - 1`.
- **BR-6** [Important] `undeclared-behavior` Queue silently drops mastered words — absent from Spec and plan, contradicting progress.go's own comment, and it is an absorbing state
  cmd/define/schedule/queue.go:46. The plan's Queue bullet covers tiers, budget,
  ties and the deck-vs-log rule but never mastery exclusion, and progress.go:61 says
  "#6's --play decides what to stop offering" while Queue decides. Because a
  mastered word is never offered it can never be answered wrong, so mastery is
  permanent with no path back into rotation. Decide, then record it.
- **BR-7** [Minor] `duplicated-comparator` The two sort.Slice comparators in Queue share an identical lookups-then-key tail
  cmd/define/schedule/queue.go:65 and :74 — extract byLookupsThenKey (ARCH-DRY).
- **BR-8** [Minor] `hostile-input-at-the-seam` Queue panics on an absurd budget where it deliberately handles budget <= 0
  cmd/define/schedule/queue.go:81 — make([]string, 0, budget) panics
  "makeslice: cap out of range" (verified). Cap the capacity at len(deck); it is
  also the tighter allocation.
- **BR-9** [Minor] `linear-scan-where-arithmetic-suffices` DaysBetween steps one AddDate per day, so Due on a years-old stamp loops thousands of times per word
  cmd/define/store/clock.go:53. Correct, and fine at current scale; noted for the
  first profile that shows it.
- **BR-10** [Minor] `undeclared-behavior` Queue returns normalised keys rather than deck Text, and the doc comment says "today's words"
  cmd/define/schedule/queue.go:34 — name the key-vs-text contract explicitly, since
  #6 has to map back to render.

## Open findings

- **BR-1** [Critical] `local-calendar-day-boundary` store.DaysBetween miscounts by a day when its arguments carry different UTC offsets, so Due fires on the review day itself
- **BR-2** [Important] `spec-constant-unpinned` Three of the six Leitner intervals are asserted by no test, and the plan promised a due-date assertion at each step
- **BR-3** [Important] `unbacked-claim-about-existing-code` "imports store and time and nothing else" is now false in four artifacts; the enumeration the plan wrote was not run when the guard rule changed
- **BR-4** [Important] `invariant-needs-mechanical-guard` The clock guard greps only "time.Now(" and misses time.Since, time.Until, time.After and the timer constructors
- **BR-5** [Important] `mutable-exported-surface` LastBox is an exported mutable package variable that every clamp and Mastered depends on
- **BR-6** [Important] `undeclared-behavior` Queue silently drops mastered words — absent from Spec and plan, contradicting progress.go's own comment, and it is an absorbing state
- **BR-7** [Minor] `duplicated-comparator` The two sort.Slice comparators in Queue share an identical lookups-then-key tail
- **BR-8** [Minor] `hostile-input-at-the-seam` Queue panics on an absurd budget where it deliberately handles budget <= 0
- **BR-9** [Minor] `linear-scan-where-arithmetic-suffices` DaysBetween steps one AddDate per day, so Due on a years-old stamp loops thousands of times per word
- **BR-10** [Minor] `undeclared-behavior` Queue returns normalised keys rather than deck Text, and the doc comment says "today's words"

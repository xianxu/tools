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
    - "n": 2
      timestamp: "2026-08-26T23:32:18-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Verified by revert: dropping `a = a.In(b.Location())` reddens clock_test.go:130 and progress_test.go:291.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'Verified by mutation: {1,3,7,13,30,90} reddens box_test.go:19; TestMultiWeekSchedule now asserts Due/!Due at each rung.'
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: |-
            Two of five sites fixed (box.go:7, atlas:1142); still stale at plan.md:27, plan.md:33,
            projects/define-learn.md:485 — and this round's own fixes minted two NEW instances of the
            same class: atlas/define.md:1165 says "A mastered word leaves the rotation" (the exact
            behaviour BR-6 removed, and what #6's implementor will read), and history_cmd.go:192 says
            "store.DaysBetween steps the calendar" when the BR-1 fix replaced the stepping with UTC
            day-index subtraction. The rule this family needs is not "fix these files": it is that a
            behaviour change must GREP the invalidated claim's distinctive phrase across a fixed
            enumeration at the moment of the change, and that the enumeration has five members, not
            four — the plan's own list omits the project file, which is why :485 survived both rounds.
          round: 2
        - id: BR-4
          disposition: addressed
          note: 'Verified in a real scratch copy (overlays do not reach this guard — it reads files off disk): time.Since reddens purity_test.go:100.'
          round: 2
        - id: BR-5
          disposition: addressed
          note: const over an array literal — compiler-enforced, stronger than the test that was asked for.
          round: 2
        - id: BR-6
          disposition: addressed
          note: 'Verified by mutation: re-adding the exclusion reddens queue_test.go:156. The "record it" half landed in code/issue but not atlas/plan — carried under BR-3, not re-raised here.'
          round: 2
        - id: BR-7
          disposition: not-addressed
          note: queue.go:74-77 and :80-83 still share the identical lookups-then-key tail.
          round: 2
        - id: BR-8
          disposition: not-addressed
          note: 'queue.go:86 unchanged; re-verified that Queue(deck, nil, now, 1<<62) panics "makeslice: cap out of range".'
          round: 2
        - id: BR-9
          disposition: addressed
          note: Incidentally fixed by BR-1 — DaysBetween is now two dayIndex subtractions, no AddDate loop.
          round: 2
        - id: BR-10
          disposition: not-addressed
          note: queue.go:26 still says "today's words"; nothing in the doc, the atlas or a test names the returned strings as normalised store.Key values rather than deck Text.
          round: 2
      findings:
        - id: BR-11
          severity: Important
          title: The only test pinning this round's Critical fix skips itself when tzdata is absent, and the repo's own fix for that was not reused
          detail: |-
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
          family: invariant-needs-mechanical-guard
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-26T23:44:42-07:00"
      agent: claude
      dispose:
        - id: BR-3
          disposition: not-addressed
          note: Seven false-claim sites remain, two minted by this round's own fixes; atlas/define.md:1165 still documents the behaviour BR-6 removed.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: queue.go:70-78 and :79-84 unchanged; identical lookups-then-key tail.
          round: 3
        - id: BR-8
          disposition: not-addressed
          note: queue.go:86 unchanged; re-verified the makeslice panic on budget 1<<62.
          round: 3
        - id: BR-10
          disposition: not-addressed
          note: queue.go:10 still says "today's words"; no doc, atlas line or test names the store.Key contract.
          round: 3
        - id: BR-11
          disposition: not-addressed
          note: Schedule pin fully fixed and mutation-verified; store test package still lacks the tzdata import its four comments claim it has.
          round: 3
      findings:
        - id: BR-12
          severity: Important
          title: The purity allowlist admits store wholesale, so a real disk-IO constructor inside schedule passes both guards
          detail: |-
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
          family: invariant-needs-mechanical-guard
          round: 3
        - id: BR-13
          severity: Minor
          title: Queue's degenerate-input domain is stated for two parameters and untested for the rest; duplicate deck keys return the same word twice
          detail: |-
            2nd in this family, BR-8 being the 1st. THE RULE - Queue is this package's public
            seam for #6, so every parameter needs a stated and tested behaviour on degenerate
            input. THE ENUMERATION, run - budget: <=0 ok, > len(deck) ok, absurd PANICS (BR-8);
            deck: empty ok, empty Text ok, DUPLICATE KEYS unhandled - verified that
            Queue([]store.Word{{Text:"Define"},{Text:"define"}}, nil, now, 10) returns
            [define define], the same word twice, spending two of the budget; prog: nil ok,
            keys absent from deck ok; now: zero untested. Prevalence 2/9.
          family: hostile-input-at-the-seam
          round: 3
        - id: BR-14
          severity: Minor
          title: The durable plan has 20 unchecked steps and 0 checked after both milestones closed
          detail: |-
            workshop/plans/000005-vocab-schedule-plan.md - its own header says the checkbox
            syntax is the tracking mechanism, and the plan is the version-controlled record of
            truth per AGENTS.md section 1. The issue file's Plan section ticks M1/M2, which is
            what the close gate reads, so the plan silently stopped being a record.
          family: plan-record-staleness
          round: 3
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

## Round 2 — 2026-08-26T23:32:18-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Verified by revert: dropping `a = a.In(b.Location())` reddens clock_test.go:130 and progress_test.go:291.
- BR-2 — addressed — Verified by mutation: {1,3,7,13,30,90} reddens box_test.go:19; TestMultiWeekSchedule now asserts Due/!Due at each rung.
- BR-3 — not-addressed — Two of five sites fixed (box.go:7, atlas:1142); still stale at plan.md:27, plan.md:33,
projects/define-learn.md:485 — and this round's own fixes minted two NEW instances of the
same class: atlas/define.md:1165 says "A mastered word leaves the rotation" (the exact
behaviour BR-6 removed, and what #6's implementor will read), and history_cmd.go:192 says
"store.DaysBetween steps the calendar" when the BR-1 fix replaced the stepping with UTC
day-index subtraction. The rule this family needs is not "fix these files": it is that a
behaviour change must GREP the invalidated claim's distinctive phrase across a fixed
enumeration at the moment of the change, and that the enumeration has five members, not
four — the plan's own list omits the project file, which is why :485 survived both rounds.
- BR-4 — addressed — Verified in a real scratch copy (overlays do not reach this guard — it reads files off disk): time.Since reddens purity_test.go:100.
- BR-5 — addressed — const over an array literal — compiler-enforced, stronger than the test that was asked for.
- BR-6 — addressed — Verified by mutation: re-adding the exclusion reddens queue_test.go:156. The "record it" half landed in code/issue but not atlas/plan — carried under BR-3, not re-raised here.
- BR-7 — not-addressed — queue.go:74-77 and :80-83 still share the identical lookups-then-key tail.
- BR-8 — not-addressed — queue.go:86 unchanged; re-verified that Queue(deck, nil, now, 1<<62) panics "makeslice: cap out of range".
- BR-9 — addressed — Incidentally fixed by BR-1 — DaysBetween is now two dayIndex subtractions, no AddDate loop.
- BR-10 — not-addressed — queue.go:26 still says "today's words"; nothing in the doc, the atlas or a test names the returned strings as normalised store.Key values rather than deck Text.

### Raised

- **BR-11** [Important] `invariant-needs-mechanical-guard` The only test pinning this round's Critical fix skips itself when tzdata is absent, and the repo's own fix for that was not reused
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

## Round 3 — 2026-08-26T23:44:42-07:00 (claude) — BLOCKED

### Disposed

- BR-3 — not-addressed — Seven false-claim sites remain, two minted by this round's own fixes; atlas/define.md:1165 still documents the behaviour BR-6 removed.
- BR-7 — not-addressed — queue.go:70-78 and :79-84 unchanged; identical lookups-then-key tail.
- BR-8 — not-addressed — queue.go:86 unchanged; re-verified the makeslice panic on budget 1<<62.
- BR-10 — not-addressed — queue.go:10 still says "today's words"; no doc, atlas line or test names the store.Key contract.
- BR-11 — not-addressed — Schedule pin fully fixed and mutation-verified; store test package still lacks the tzdata import its four comments claim it has.

### Raised

- **BR-12** [Important] `invariant-needs-mechanical-guard` The purity allowlist admits store wholesale, so a real disk-IO constructor inside schedule passes both guards
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
- **BR-13** [Minor] `hostile-input-at-the-seam` Queue's degenerate-input domain is stated for two parameters and untested for the rest; duplicate deck keys return the same word twice
  2nd in this family, BR-8 being the 1st. THE RULE - Queue is this package's public
  seam for #6, so every parameter needs a stated and tested behaviour on degenerate
  input. THE ENUMERATION, run - budget: <=0 ok, > len(deck) ok, absurd PANICS (BR-8);
  deck: empty ok, empty Text ok, DUPLICATE KEYS unhandled - verified that
  Queue([]store.Word{{Text:"Define"},{Text:"define"}}, nil, now, 10) returns
  [define define], the same word twice, spending two of the budget; prog: nil ok,
  keys absent from deck ok; now: zero untested. Prevalence 2/9.
- **BR-14** [Minor] `plan-record-staleness` The durable plan has 20 unchecked steps and 0 checked after both milestones closed
  workshop/plans/000005-vocab-schedule-plan.md - its own header says the checkbox
  syntax is the tracking mechanism, and the plan is the version-controlled record of
  truth per AGENTS.md section 1. The issue file's Plan section ticks M1/M2, which is
  what the close gate reads, so the plan silently stopped being a record.

## Open findings

- **BR-3** [Important] `unbacked-claim-about-existing-code` "imports store and time and nothing else" is now false in four artifacts; the enumeration the plan wrote was not run when the guard rule changed
- **BR-7** [Minor] `duplicated-comparator` The two sort.Slice comparators in Queue share an identical lookups-then-key tail
- **BR-8** [Minor] `hostile-input-at-the-seam` Queue panics on an absurd budget where it deliberately handles budget <= 0
- **BR-10** [Minor] `undeclared-behavior` Queue returns normalised keys rather than deck Text, and the doc comment says "today's words"
- **BR-11** [Important] `invariant-needs-mechanical-guard` The only test pinning this round's Critical fix skips itself when tzdata is absent, and the repo's own fix for that was not reused
- **BR-12** [Important] `invariant-needs-mechanical-guard` The purity allowlist admits store wholesale, so a real disk-IO constructor inside schedule passes both guards
- **BR-13** [Minor] `hostile-input-at-the-seam` Queue's degenerate-input domain is stated for two parameters and untested for the rest; duplicate deck keys return the same word twice
- **BR-14** [Minor] `plan-record-staleness` The durable plan has 20 unchecked steps and 0 checked after both milestones closed

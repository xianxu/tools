---
gate: plan-quality
issue: 76
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-18T12:21:24-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: 'Task 2 leaves the tree non-compiling: parse.go loses Entry.source while definitions.go still writes it'
          detail: |-
            `git checkout 62c6a66^ -- cmd/define/parse.go` removes `Entry.source`
            (parse.go:46 today; absent pre-#65), but `entry.source = section.source[entryIndex]`
            at definitions.go:87 is only removed in Task 3. Task 2's `go build ./...` therefore
            fails with "entry.source undefined", its stated expectation of "the baseline" is
            unreachable, and the commit it prescribes is a broken tree — which also breaks the
            committed-baseline requirement Task 3's mutation checks depend on. Fold the
            definitions.go write (and the conformance-tagged `entry.source` use at
            bilingual_conformance_test.go:123) into Task 2, or remove the writers first.
          family: intermediate-commit-must-build
          round: 1
        - id: PQ-2
          severity: Important
          title: The stated between-task red set contradicts what the plan guards actually do
          detail: |-
            `currentTruthOnly` (repo_guard_test.go:751-760) strips every occurrence of a name
            carried by a `| … | deleted |` row from the whole document before the plan guards
            scan it, so the plan's `deleted` rows are never checked — they are not "red-first
            pins under TestPlanTablesNameEntitiesThatExist until each deletion lands". Running
            the guards now fails only on the `projectDisplayText` row and the three unwritten
            test names. Conversely, after Task 1's commit TestPlanTableStatusMatchesTheChangeWindow
            (repo_guard_test.go:1325, checkPlanStatus:1572) will fail the
            `TestDictionaryMonolingualOriginAndDisabledTint | dictionary_language_test.go | modified`
            row, because the renames touch that file while that declaration is untouched until
            Task 3 — a red row outside the predicted set that the plan instructs the implementer
            to treat as a real failure.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-3
          severity: Minor
          title: Compress Task 1's three prose cases to a strategy line and add a property guard over arbitrary rendered text
          detail: |-
            `projectDisplayText` is a byte scanner over arbitrary rendered output (UTF-8 decode,
            ANSI-escape skip, span merge) and the fold makes one branch unconditional. Keep M-join
            and M-break; replace the enumerated cases with one line naming the adversarial class,
            and add a fuzz/property check (result.text == rendered always; spans in-bounds,
            ascending, non-overlapping) seeded with those cases — the tree already fuzzes the
            Oxford parser but nothing fuzzes the projection.
          family: test-strategy-enumeration
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-18T12:30:56-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Task 2 now removes parse.go's symbols together with every user, including the conformance-tagged one; CHECK type-checks tagged files.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: The unmeasured red-set prediction is removed and Task 1 now touches TestDictionaryMonolingualOriginAndDisabledTint's own body.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Cases compressed to a strategy line; FuzzDisplayProjection adds the property check.
          round: 2
      findings:
        - id: PQ-4
          severity: Minor
          title: M-join survives the cases the plan names; only a length-preserving substitution kills it
          detail: 'This is the 2nd finding in family `unbacked-existing-behavior-claim` (prevalence 2/2 rounds: PQ-2 predicted a guard red-set, this predicts a mutation kill). Do not just add a case — the rule is: a plan may state a predicted test/guard outcome only with a measurement beside it ("measured <date>"), and an unmeasured prediction is deleted rather than reasoned to. Instance, measured today by running the projection body with `mismatch := false`: display-mode `"red red"`→`"red green"` and `"an other"`→`"another"` both return neutral under the mutant, because the length mismatch is caught by the trailing-leftover guard at dictionary_language.go:109-111, not by the glyph branch. `"red red"`→`"red bed"` kills it. M-break is unaffected.'
          family: unbacked-existing-behavior-claim
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-18T12:48:14-07:00"
      agent: claude
      dispose:
        - id: PQ-4
          disposition: addressed
          note: Task 1, Task 3's M-join and the Done-when row now name "red red"→"red bed"; family rule applied plan-wide with measurements dated.
          round: 3
      blocked: false
content_hash: eb3f36682875ef432f5e3d1e3ded1ac001d5883dc7351c8502eb38156d2829c5
---

# Gate ledger — tools#76 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-18T12:21:24-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `intermediate-commit-must-build` Task 2 leaves the tree non-compiling: parse.go loses Entry.source while definitions.go still writes it
  `git checkout 62c6a66^ -- cmd/define/parse.go` removes `Entry.source`
  (parse.go:46 today; absent pre-#65), but `entry.source = section.source[entryIndex]`
  at definitions.go:87 is only removed in Task 3. Task 2's `go build ./...` therefore
  fails with "entry.source undefined", its stated expectation of "the baseline" is
  unreachable, and the commit it prescribes is a broken tree — which also breaks the
  committed-baseline requirement Task 3's mutation checks depend on. Fold the
  definitions.go write (and the conformance-tagged `entry.source` use at
  bilingual_conformance_test.go:123) into Task 2, or remove the writers first.
- **PQ-2** [Important] `unbacked-existing-behavior-claim` The stated between-task red set contradicts what the plan guards actually do
  `currentTruthOnly` (repo_guard_test.go:751-760) strips every occurrence of a name
  carried by a `| … | deleted |` row from the whole document before the plan guards
  scan it, so the plan's `deleted` rows are never checked — they are not "red-first
  pins under TestPlanTablesNameEntitiesThatExist until each deletion lands". Running
  the guards now fails only on the `projectDisplayText` row and the three unwritten
  test names. Conversely, after Task 1's commit TestPlanTableStatusMatchesTheChangeWindow
  (repo_guard_test.go:1325, checkPlanStatus:1572) will fail the
  `TestDictionaryMonolingualOriginAndDisabledTint | dictionary_language_test.go | modified`
  row, because the renames touch that file while that declaration is untouched until
  Task 3 — a red row outside the predicted set that the plan instructs the implementer
  to treat as a real failure.
- **PQ-3** [Minor] `test-strategy-enumeration` Compress Task 1's three prose cases to a strategy line and add a property guard over arbitrary rendered text
  `projectDisplayText` is a byte scanner over arbitrary rendered output (UTF-8 decode,
  ANSI-escape skip, span merge) and the fold makes one branch unconditional. Keep M-join
  and M-break; replace the enumerated cases with one line naming the adversarial class,
  and add a fuzz/property check (result.text == rendered always; spans in-bounds,
  ascending, non-overlapping) seeded with those cases — the tree already fuzzes the
  Oxford parser but nothing fuzzes the projection.

## Round 2 — 2026-09-18T12:30:56-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Task 2 now removes parse.go's symbols together with every user, including the conformance-tagged one; CHECK type-checks tagged files.
- PQ-2 — addressed — The unmeasured red-set prediction is removed and Task 1 now touches TestDictionaryMonolingualOriginAndDisabledTint's own body.
- PQ-3 — addressed — Cases compressed to a strategy line; FuzzDisplayProjection adds the property check.

### Raised

- **PQ-4** [Minor] `unbacked-existing-behavior-claim` M-join survives the cases the plan names; only a length-preserving substitution kills it
  This is the 2nd finding in family `unbacked-existing-behavior-claim` (prevalence 2/2 rounds: PQ-2 predicted a guard red-set, this predicts a mutation kill). Do not just add a case — the rule is: a plan may state a predicted test/guard outcome only with a measurement beside it ("measured <date>"), and an unmeasured prediction is deleted rather than reasoned to. Instance, measured today by running the projection body with `mismatch := false`: display-mode `"red red"`→`"red green"` and `"an other"`→`"another"` both return neutral under the mutant, because the length mismatch is caught by the trailing-leftover guard at dictionary_language.go:109-111, not by the glyph branch. `"red red"`→`"red bed"` kills it. M-break is unaffected.

## Round 3 — 2026-09-18T12:48:14-07:00 (claude) — passed

### Disposed

- PQ-4 — addressed — Task 1, Task 3's M-join and the Done-when row now name "red red"→"red bed"; family rule applied plan-wide with measurements dated.

## Open findings

(none — every finding has been disposed)

---
gate: plan-quality
issue: 61
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-14T17:31:58-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Replace enumerated test cases with named functions and adversarial strategies.
          detail: The plan enumerates cases at lines 63, 71, 83, 128 and 131 instead of providing the required function-level test strategy. Compress these into named test surfaces, including selectSpanishRecords, renderDefinitions, parseBilingualArgs and the settings parser/read-write functions; give each risky function its adversarial input class and mechanical guard, such as seeded malformed-record fuzzing or independent rendered-coordinate assertions (ARCH-PURE, ARCH-SECURE).
          family: function-level-test-strategy
          round: 1
        - id: PQ-2
          severity: Minor
          title: Specify when native conformance runs after implementation.
          detail: The plan supplies a strict native command and independent fixtures but no recurring cadence for detecting private DictionaryServices contract drift. Name the existing scheduled mechanism or an explicit recurring check policy (ARCH-MOCK).
          family: live-conformance-cadence
          round: 1
        - id: PQ-3
          severity: Minor
          title: Bound the user-visible cost of supplemental lookup.
          detail: Record and byte caps bound individual results, but todaysQuestions constructs full definitions across the queue before practice begins (cmd/define/play_loop.go:997). State the expected queue scale and lookup/startup latency budget, its basis, and behavior when exceeded (ARCH-CONSTRAINTS).
          family: explicit-operating-envelope
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-14T17:33:24-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Plan lines 148–162 supersede historical case lists with function-level adversarial strategies and independent mechanical guards.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Plan line 164 requires strict native conformance before dictionary-changing releases and after macOS upgrades, with recorded evidence.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Plan line 166 identifies the 20-word workload, per-word and queue acceptance budgets, synchronous-call limitations, and profiling/re-plan behavior when exceeded.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-15T08:48:06-07:00"
      agent: claude
      findings:
        - id: PQ-4
          severity: Important
          title: Name the tests that pin practice as offline and the narrowed invariant they must assert.
          detail: TestAClozeSittingNeverReachesForTheModel (cmd/define/cloze_test.go:350) and TestSessionRunsWithTheModelUnavailable (cmd/define/play_loop_test.go:233) panic the model seam inside todaysQuestions; atlas/define.md:1486-1491 calls it a promise. The plan only says "update offline-practice comments". State the new invariant (off never reaches; warm cache constructs no client; cold cache reaches once before the first question, never inside playSession) and keep the panicking-seam shape (ARCH-MOCK, ARCH-PURPOSE).
          family: invariant-reversal-names-enforcer
          round: 3
        - id: PQ-5
          severity: Important
          title: Extension task rows are prose case lists; no test is named per new production surface.
          detail: '2nd finding in this family. Rule: every revision adding a production surface adds a PQ-1 table row (surface, named test plus adversarial class, mechanical guard) and a Done-when row. Apply once to the six new concepts (checked translation, preparation client, cache store, form presentations, question-preparation boundary, author language) and compress the case lists.'
          family: function-level-test-strategy
          round: 3
        - id: PQ-6
          severity: Minor
          title: Assistance budgets state values without basis or derived first-run load.
          detail: '2nd finding in this family. Rule: each budget line carries value, basis, and derived load at the declared workload. A 20-word queue is roughly 80-100 sources, so 5-7 sequential proxy calls inside the 30 s cap on a cold cache; say whether that is measured or assumed (ARCH-CONSTRAINTS).'
          family: explicit-operating-envelope
          round: 3
        - id: PQ-7
          severity: Minor
          title: Opt-in live translation conformance has no cadence.
          detail: '2nd finding in this family. Rule: the PQ-2 cadence policy applies to every live check in the plan; reword PQ-2 to cover native dictionary and live translation checks alike (ARCH-MOCK).'
          family: live-conformance-cadence
          round: 3
        - id: PQ-8
          severity: Minor
          title: Undeclared overlap with in-flight issue 54 at the practice startup seam and per-deck writes.
          detail: Issue 54 (working, branch 000054-background-harvest) adds a session state machine, a lock for every dictionary call, and background per-deck writes. This extension adds foreground model preparation in todaysQuestions and a new per-deck cache file. Declare the ordering and the lock/concurrent-writer interaction (ARCH-ORDER).
          family: cross-issue-dep-declared
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-15T08:54:02-07:00"
      agent: claude
      dispose:
        - id: PQ-4
          disposition: addressed
          note: Three-clause invariant with named panicking/counting-seam tests; existing pins verified as English sittings.
          round: 4
        - id: PQ-5
          disposition: addressed
          note: Per-surface verification table and Done-when rows cover all six new concepts; case lists compressed.
          round: 4
        - id: PQ-6
          disposition: addressed
          note: Each budget has value, basis, derived load and exceeded behavior; latency marked assumed pending one live run.
          round: 4
        - id: PQ-7
          disposition: addressed
          note: PQ-2 cadence reworded to govern the native and live translation checks alike.
          round: 4
        - id: PQ-8
          disposition: addressed
          note: Issue 54 confirmed merged at c346955; lock, writers, ordering, model concurrency and permission stated against real symbols.
          round: 4
      blocked: false
content_hash: d6fdd8d75488d185414c49d3fd0adf7310a2926a4952bf55ecbe0d7b8d4a7c48
---

# Gate ledger — tools#61 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-14T17:31:58-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `function-level-test-strategy` Replace enumerated test cases with named functions and adversarial strategies.
  The plan enumerates cases at lines 63, 71, 83, 128 and 131 instead of providing the required function-level test strategy. Compress these into named test surfaces, including selectSpanishRecords, renderDefinitions, parseBilingualArgs and the settings parser/read-write functions; give each risky function its adversarial input class and mechanical guard, such as seeded malformed-record fuzzing or independent rendered-coordinate assertions (ARCH-PURE, ARCH-SECURE).
- **PQ-2** [Minor] `live-conformance-cadence` Specify when native conformance runs after implementation.
  The plan supplies a strict native command and independent fixtures but no recurring cadence for detecting private DictionaryServices contract drift. Name the existing scheduled mechanism or an explicit recurring check policy (ARCH-MOCK).
- **PQ-3** [Minor] `explicit-operating-envelope` Bound the user-visible cost of supplemental lookup.
  Record and byte caps bound individual results, but todaysQuestions constructs full definitions across the queue before practice begins (cmd/define/play_loop.go:997). State the expected queue scale and lookup/startup latency budget, its basis, and behavior when exceeded (ARCH-CONSTRAINTS).

## Round 2 — 2026-09-14T17:33:24-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Plan lines 148–162 supersede historical case lists with function-level adversarial strategies and independent mechanical guards.
- PQ-2 — addressed — Plan line 164 requires strict native conformance before dictionary-changing releases and after macOS upgrades, with recorded evidence.
- PQ-3 — addressed — Plan line 166 identifies the 20-word workload, per-word and queue acceptance budgets, synchronous-call limitations, and profiling/re-plan behavior when exceeded.

## Round 3 — 2026-09-15T08:48:06-07:00 (claude) — BLOCKED

### Raised

- **PQ-4** [Important] `invariant-reversal-names-enforcer` Name the tests that pin practice as offline and the narrowed invariant they must assert.
  TestAClozeSittingNeverReachesForTheModel (cmd/define/cloze_test.go:350) and TestSessionRunsWithTheModelUnavailable (cmd/define/play_loop_test.go:233) panic the model seam inside todaysQuestions; atlas/define.md:1486-1491 calls it a promise. The plan only says "update offline-practice comments". State the new invariant (off never reaches; warm cache constructs no client; cold cache reaches once before the first question, never inside playSession) and keep the panicking-seam shape (ARCH-MOCK, ARCH-PURPOSE).
- **PQ-5** [Important] `function-level-test-strategy` Extension task rows are prose case lists; no test is named per new production surface.
  2nd finding in this family. Rule: every revision adding a production surface adds a PQ-1 table row (surface, named test plus adversarial class, mechanical guard) and a Done-when row. Apply once to the six new concepts (checked translation, preparation client, cache store, form presentations, question-preparation boundary, author language) and compress the case lists.
- **PQ-6** [Minor] `explicit-operating-envelope` Assistance budgets state values without basis or derived first-run load.
  2nd finding in this family. Rule: each budget line carries value, basis, and derived load at the declared workload. A 20-word queue is roughly 80-100 sources, so 5-7 sequential proxy calls inside the 30 s cap on a cold cache; say whether that is measured or assumed (ARCH-CONSTRAINTS).
- **PQ-7** [Minor] `live-conformance-cadence` Opt-in live translation conformance has no cadence.
  2nd finding in this family. Rule: the PQ-2 cadence policy applies to every live check in the plan; reword PQ-2 to cover native dictionary and live translation checks alike (ARCH-MOCK).
- **PQ-8** [Minor] `cross-issue-dep-declared` Undeclared overlap with in-flight issue 54 at the practice startup seam and per-deck writes.
  Issue 54 (working, branch 000054-background-harvest) adds a session state machine, a lock for every dictionary call, and background per-deck writes. This extension adds foreground model preparation in todaysQuestions and a new per-deck cache file. Declare the ordering and the lock/concurrent-writer interaction (ARCH-ORDER).

## Round 4 — 2026-09-15T08:54:02-07:00 (claude) — passed

### Disposed

- PQ-4 — addressed — Three-clause invariant with named panicking/counting-seam tests; existing pins verified as English sittings.
- PQ-5 — addressed — Per-surface verification table and Done-when rows cover all six new concepts; case lists compressed.
- PQ-6 — addressed — Each budget has value, basis, derived load and exceeded behavior; latency marked assumed pending one live run.
- PQ-7 — addressed — PQ-2 cadence reworded to govern the native and live translation checks alike.
- PQ-8 — addressed — Issue 54 confirmed merged at c346955; lock, writers, ordering, model concurrency and permission stated against real symbols.

## Open findings

(none — every finding has been disposed)

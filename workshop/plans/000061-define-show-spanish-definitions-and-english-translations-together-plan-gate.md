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
content_hash: c0efa636ad42ecd0e665b0d0496ae70b33a3e8f24feb99ef16552ac7d69b4636
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

## Open findings

(none — every finding has been disposed)

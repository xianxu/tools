---
gate: plan-quality
issue: 65
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-15T13:59:04-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Specify decoder recovery states and their termination rules.
          detail: 'ARCH-ORDER: Plan lines 92–107 describe outcomes without enumerating the decoder''s legal states and transitions. For nested markers, specify whether the first closing marker or the outer closing marker ends neutral recovery, and how subsequent opening markers behave during invalid or oversized-segment recovery. Name the authoritative state/event model, require mutations through its transition function, and test chunk-independent ownership and rejected events across generated sequences.'
          family: explicit-recovery-state-contract
          round: 1
        - id: PQ-2
          severity: Minor
          title: Name the remaining test functions and consolidate repeated test prose.
          detail: The verification table at lines 128–130 names ownership validation/projection and form builders as surfaces rather than naming all functions to unit-test. Name those functions and consolidate the repeated control-input test enumeration at line 107 into the existing strategy rows, retaining behavioral contracts in the design.
          family: function-level-test-strategy
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-15T14:01:13-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The authoritative transition table specifies first-close recovery, opener handling, termination, exclusive state mutation and generated sequence invariants.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: The verification table names ownership/projection functions and form presentation methods and consolidates repeated test prose into strategy rows.
          round: 2
      blocked: false
content_hash: e8914b84c7ed1c0112480a949fb9583777a94ecc2de69a3d2356c40db2f132e9
---

# Gate ledger — tools#65 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-15T13:59:04-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `explicit-recovery-state-contract` Specify decoder recovery states and their termination rules.
  ARCH-ORDER: Plan lines 92–107 describe outcomes without enumerating the decoder's legal states and transitions. For nested markers, specify whether the first closing marker or the outer closing marker ends neutral recovery, and how subsequent opening markers behave during invalid or oversized-segment recovery. Name the authoritative state/event model, require mutations through its transition function, and test chunk-independent ownership and rejected events across generated sequences.
- **PQ-2** [Minor] `function-level-test-strategy` Name the remaining test functions and consolidate repeated test prose.
  The verification table at lines 128–130 names ownership validation/projection and form builders as surfaces rather than naming all functions to unit-test. Name those functions and consolidate the repeated control-input test enumeration at line 107 into the existing strategy rows, retaining behavioral contracts in the design.

## Round 2 — 2026-09-15T14:01:13-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — The authoritative transition table specifies first-close recovery, opener handling, termination, exclusive state mutation and generated sequence invariants.
- PQ-2 — addressed — The verification table names ownership/projection functions and form presentation methods and consolidates repeated test prose into strategy rows.

## Open findings

(none — every finding has been disposed)

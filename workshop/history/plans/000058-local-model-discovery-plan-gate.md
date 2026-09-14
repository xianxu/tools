---
gate: plan-quality
issue: 58
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-13T22:35:23-07:00"
      agent: codex
      findings:
        - id: PQ-1
          severity: Important
          title: Replace enumerated test recipes with named function-level strategies.
          detail: Tasks 1–4 enumerate test cases, beginning with “direct Claude wins over Codex,” rather than consistently naming each risky function and its adversarial input class plus mechanical guard. Compress these into strategy lines for SelectModel, Resolve, discoverModels, the auto-client methods, effective, and renderRequest; use fuzz/property checks for catalog parsing and ranking, controlled channels for cancellation ordering, and wire/hash assertions for rendering.
          family: function-level-test-strategy
          round: 1
        - id: PQ-2
          severity: Important
          title: 'Declare coordination with active issue #54 before changing reflection.'
          detail: 'Issue #54 is working and explicitly plans to extract a typed reflection core (workshop/issues/000054-background-harvest.md:161 and :235), while Task 4 changes client ownership and provenance in that same flow. Under ARCH-PURPOSE, state the ordering or ownership agreement and how selected-model reporting reaches the resulting shared core; record a dependency if implementation must wait.'
          family: cross-workstream-ownership
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-13T22:36:46-07:00"
      agent: codex
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Tasks now specify named function-level strategies with fuzz/property checks, controlled cancellation channels, and wire/hash oracles.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: 'The revision declares implementation ordering, limits reflection ownership, and requires SelectionOf(client) to survive #54''s shared-core extraction; #54''s log records the coordination.'
          round: 2
      blocked: false
content_hash: c3c20a43d2cdca4ca611b2377dad6d778e08f67ed4fbdc4204cb6d45bd374b20
---

# Gate ledger — tools#58 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-13T22:35:23-07:00 (codex) — BLOCKED

### Raised

- **PQ-1** [Important] `function-level-test-strategy` Replace enumerated test recipes with named function-level strategies.
  Tasks 1–4 enumerate test cases, beginning with “direct Claude wins over Codex,” rather than consistently naming each risky function and its adversarial input class plus mechanical guard. Compress these into strategy lines for SelectModel, Resolve, discoverModels, the auto-client methods, effective, and renderRequest; use fuzz/property checks for catalog parsing and ranking, controlled channels for cancellation ordering, and wire/hash assertions for rendering.
- **PQ-2** [Important] `cross-workstream-ownership` Declare coordination with active issue #54 before changing reflection.
  Issue #54 is working and explicitly plans to extract a typed reflection core (workshop/issues/000054-background-harvest.md:161 and :235), while Task 4 changes client ownership and provenance in that same flow. Under ARCH-PURPOSE, state the ordering or ownership agreement and how selected-model reporting reaches the resulting shared core; record a dependency if implementation must wait.

## Round 2 — 2026-09-13T22:36:46-07:00 (codex) — passed

### Disposed

- PQ-1 — addressed — Tasks now specify named function-level strategies with fuzz/property checks, controlled cancellation channels, and wire/hash oracles.
- PQ-2 — addressed — The revision declares implementation ordering, limits reflection ownership, and requires SelectionOf(client) to survive #54's shared-core extraction; #54's log records the coordination.

## Open findings

(none — every finding has been disposed)

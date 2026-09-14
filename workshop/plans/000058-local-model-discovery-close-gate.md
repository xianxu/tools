---
gate: boundary-review
issue: 58
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-13T22:49:27-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: New discovery conformance test bypasses strict-mode policy and fails the repository guard
          detail: internal/llm/discovery_conformance_test.go:18,21,26 directly skip instead of using the existing conformance policy. TestEverySkipIsRoutedOrWaived fails on all three sites, and a missing-credential run still exits successfully under CONFORMANCE_STRICT=1. Route absent dependencies through conformance.SkipOrFail and explicitly justify any inapplicable configuration waiver; rerun the guard and strict/default behavior checks (ARCH-DRY, ARCH-MOCK).
          family: conformance-skip-policy
          round: 1
      blocked: true
---

# Gate ledger — tools#58 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-13T22:49:27-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `conformance-skip-policy` New discovery conformance test bypasses strict-mode policy and fails the repository guard
  internal/llm/discovery_conformance_test.go:18,21,26 directly skip instead of using the existing conformance policy. TestEverySkipIsRoutedOrWaived fails on all three sites, and a missing-credential run still exits successfully under CONFORMANCE_STRICT=1. Route absent dependencies through conformance.SkipOrFail and explicitly justify any inapplicable configuration waiver; rerun the guard and strict/default behavior checks (ARCH-DRY, ARCH-MOCK).

## Open findings

- **BR-1** [Important] `conformance-skip-policy` New discovery conformance test bypasses strict-mode policy and fails the repository guard

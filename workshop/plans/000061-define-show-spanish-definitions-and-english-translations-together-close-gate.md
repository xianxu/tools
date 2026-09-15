---
gate: boundary-review
issue: 61
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-14T18:00:37-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Spanish factory reports missing Larousse when metadata availability is unknown
          detail: cmd/define/dict_darwin.go:295 constructs an enable/download error even when installedDictionaries returns nil for unavailable metadata APIs. Preserve that failure separately from confirmed absence and test both through factory composition; this violates the Spec and ARCH-SECURE.
          family: unknown-availability-is-not-absence
          round: 1
        - id: BR-2
          severity: Minor
          title: Core concepts omit PURE and INTEGRATION classifications
          detail: workshop/plans/000061-bilingual-definitions-plan.md:13 lacks the requested kind column and groups pure parsers with effectful command handlers. Add classifications and split mixed rows.
          family: explicit-core-concept-classification
          round: 1
      blocked: true
---

# Gate ledger — tools#61 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-14T18:00:37-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `unknown-availability-is-not-absence` Spanish factory reports missing Larousse when metadata availability is unknown
  cmd/define/dict_darwin.go:295 constructs an enable/download error even when installedDictionaries returns nil for unavailable metadata APIs. Preserve that failure separately from confirmed absence and test both through factory composition; this violates the Spec and ARCH-SECURE.
- **BR-2** [Minor] `explicit-core-concept-classification` Core concepts omit PURE and INTEGRATION classifications
  workshop/plans/000061-bilingual-definitions-plan.md:13 lacks the requested kind column and groups pure parsers with effectful command handlers. Add classifications and split mixed rows.

## Open findings

- **BR-1** [Critical] `unknown-availability-is-not-absence` Spanish factory reports missing Larousse when metadata availability is unknown
- **BR-2** [Minor] `explicit-core-concept-classification` Core concepts omit PURE and INTEGRATION classifications

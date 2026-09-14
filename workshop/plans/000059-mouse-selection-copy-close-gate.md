---
gate: boundary-review
issue: 59
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-14T06:50:00-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Selection cancellation is bypassed by saturated input and scoped SIGINT
          detail: cmd/define/selection_input.go:177-182 drops keys before router cancellation; after saturation is established, typing or scrolling during a fresh drag leaves it active and release can copy. Scoped SIGINT likewise reaches interrupts.Fire at cmd/define/interrupt.go:85 without selection invalidation, allowing release before the cancelled operation redraws. Apply cancellation at observation across all ingress paths and add deterministic zero-copy regression tests (ARCH-ORDER, ARCH-PURPOSE).
          family: cancellation-ingress-completeness
          round: 1
      blocked: true
---

# Gate ledger — tools#59 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-14T06:50:00-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `cancellation-ingress-completeness` Selection cancellation is bypassed by saturated input and scoped SIGINT
  cmd/define/selection_input.go:177-182 drops keys before router cancellation; after saturation is established, typing or scrolling during a fresh drag leaves it active and release can copy. Scoped SIGINT likewise reaches interrupts.Fire at cmd/define/interrupt.go:85 without selection invalidation, allowing release before the cancelled operation redraws. Apply cancellation at observation across all ingress paths and add deterministic zero-copy regression tests (ARCH-ORDER, ARCH-PURPOSE).

## Open findings

- **BR-1** [Critical] `cancellation-ingress-completeness` Selection cancellation is bypassed by saturated input and scoped SIGINT

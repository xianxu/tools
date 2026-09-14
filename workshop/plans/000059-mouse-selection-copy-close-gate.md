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
    - "n": 2
      timestamp: "2026-09-14T07:02:16-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: selection_input.go:185-193 cancels rejected input before dropping it; interrupt.go:52-60 invokes selection observation before foreground cancellation. Independently removing each fix through temporary Go overlays caused its selection_cancel_test.go regression to fail with an unwanted clipboard write.
          round: 2
      blocked: false
---

# Gate ledger — tools#59 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-14T06:50:00-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `cancellation-ingress-completeness` Selection cancellation is bypassed by saturated input and scoped SIGINT
  cmd/define/selection_input.go:177-182 drops keys before router cancellation; after saturation is established, typing or scrolling during a fresh drag leaves it active and release can copy. Scoped SIGINT likewise reaches interrupts.Fire at cmd/define/interrupt.go:85 without selection invalidation, allowing release before the cancelled operation redraws. Apply cancellation at observation across all ingress paths and add deterministic zero-copy regression tests (ARCH-ORDER, ARCH-PURPOSE).

## Round 2 — 2026-09-14T07:02:16-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — selection_input.go:185-193 cancels rejected input before dropping it; interrupt.go:52-60 invokes selection observation before foreground cancellation. Independently removing each fix through temporary Go overlays caused its selection_cancel_test.go regression to fail with an unwanted clipboard write.

## Open findings

(none — every finding has been disposed)

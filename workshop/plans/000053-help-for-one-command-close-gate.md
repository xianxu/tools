---
gate: boundary-review
issue: 53
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-12T17:01:33-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Nothing pins that runHelp and dispatchCommand pass cc.width to commandUsage; a hard-coded 0 passes every test
          detail: 'Both help tests build commandCtx with width 0 and compare to commandUsage(c, 0). Add one case with width: 20 asserting out == commandUsage(hist, 20) so the wrap actually reaches the screen.'
          family: io-shell-forwards-context
          round: 1
        - id: BR-2
          severity: Minor
          title: -h is accepted by asksForUsage but only the atlas mentions it; helpUsage and the bare-help line say --help only
          detail: Either add -h to helpUsage (the span propagates it to both docs) or drop -h from the contract.
          family: docs-name-every-accepted-form
          round: 1
        - id: BR-3
          severity: Minor
          title: The issue's Done-when boxes are still unticked while every Plan box is ticked
          detail: Tick the six Done-when items at close so the tracker reflects the verified state.
          family: issue-done-when-unticked
          round: 1
      blocked: true
---

# Gate ledger — tools#53 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-12T17:01:33-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `io-shell-forwards-context` Nothing pins that runHelp and dispatchCommand pass cc.width to commandUsage; a hard-coded 0 passes every test
  Both help tests build commandCtx with width 0 and compare to commandUsage(c, 0). Add one case with width: 20 asserting out == commandUsage(hist, 20) so the wrap actually reaches the screen.
- **BR-2** [Minor] `docs-name-every-accepted-form` -h is accepted by asksForUsage but only the atlas mentions it; helpUsage and the bare-help line say --help only
  Either add -h to helpUsage (the span propagates it to both docs) or drop -h from the contract.
- **BR-3** [Minor] `issue-done-when-unticked` The issue's Done-when boxes are still unticked while every Plan box is ticked
  Tick the six Done-when items at close so the tracker reflects the verified state.

## Open findings

- **BR-1** [Important] `io-shell-forwards-context` Nothing pins that runHelp and dispatchCommand pass cc.width to commandUsage; a hard-coded 0 passes every test
- **BR-2** [Minor] `docs-name-every-accepted-form` -h is accepted by asksForUsage but only the atlas mentions it; helpUsage and the bare-help line say --help only
- **BR-3** [Minor] `issue-done-when-unticked` The issue's Done-when boxes are still unticked while every Plan box is ticked

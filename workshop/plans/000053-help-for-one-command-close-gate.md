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
    - "n": 2
      timestamp: "2026-09-12T17:11:26-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: TestHelpRendersWithTheContextItIsGiven goes red when either call site renders at width 0 (verified by revert in a scratch worktree); control green.
          round: 2
        - id: BR-2
          disposition: addressed
          note: helpUsage and the bare-help line say "--help or -h"; the regenerated span carries it to README and atlas, and the -h routing is pinned by TestDashHelpPrintsTheUsageForEveryCommand.
          round: 2
        - id: BR-3
          disposition: addressed
          note: All six Done-when boxes are ticked in the issue file at head.
          round: 2
      findings:
        - id: BR-4
          severity: Minor
          title: The --help/-h pair is restated by hand at five sites; nothing checks helpUsage names every form asksForUsage accepts
          detail: 'Second finding in this family, so the rule rather than the instance: the accepted forms are one list, and every surface naming them derives from or is checked against it. A usageFlags slice driving asksForUsage and the test loop, plus one Contains assertion over helpUsage, closes the class. Non-blocking.'
          family: docs-name-every-accepted-form
          round: 2
      blocked: false
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

## Round 2 — 2026-09-12T17:11:26-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — TestHelpRendersWithTheContextItIsGiven goes red when either call site renders at width 0 (verified by revert in a scratch worktree); control green.
- BR-2 — addressed — helpUsage and the bare-help line say "--help or -h"; the regenerated span carries it to README and atlas, and the -h routing is pinned by TestDashHelpPrintsTheUsageForEveryCommand.
- BR-3 — addressed — All six Done-when boxes are ticked in the issue file at head.

### Raised

- **BR-4** [Minor] `docs-name-every-accepted-form` The --help/-h pair is restated by hand at five sites; nothing checks helpUsage names every form asksForUsage accepts
  Second finding in this family, so the rule rather than the instance: the accepted forms are one list, and every surface naming them derives from or is checked against it. A usageFlags slice driving asksForUsage and the test loop, plus one Contains assertion over helpUsage, closes the class. Non-blocking.

## Open findings

- **BR-4** [Minor] `docs-name-every-accepted-form` The --help/-h pair is restated by hand at five sites; nothing checks helpUsage names every form asksForUsage accepts

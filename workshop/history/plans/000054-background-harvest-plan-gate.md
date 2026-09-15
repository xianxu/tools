---
gate: plan-quality
issue: 54
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-12T17:41:41-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: countUnbanded counts a strict subset of the Spec's "new" words, and bands-first harvesting makes the backlog stall
          detail: runHarvest bands every unbanded word before runAuthoring authors any (harvest.go:144-206, :223), so under bgBudget=60 a backlog is banded across jobs while authoring gets little; once unbanded hits 0 the count stops firing though most words still have no item. Count what harvest would work on (no band OR no items) or make harvestDeck band only what it can also author.
          family: trigger-count-matches-spec-definition
          round: 1
        - id: PQ-2
          severity: Minor
          title: Enumerate /forget mid-job and state the job's context derives from the session ctx
          detail: Forget removes word/facts/items; a job mid-authoring re-creates facts/items for a forgotten word. interrupter.Fire with no scope cancels the session ctx (quit); a scoped Ctrl-C must not reach the runner.
          family: order-enumeration-names-interrupting-events
          round: 1
        - id: PQ-3
          severity: Minor
          title: modelLookups parses a hand-edited file; add a fuzz seeded with the table's malformed forms
          detail: cloze_fuzz_test.go is the precedent shape.
          family: fuzz-parsers-over-untrusted-input
          round: 1
        - id: PQ-4
          severity: Minor
          title: Log says the TUI has one goroutine; rawterm.go:87/:242 and interrupt.go:83 already run three. Forget returns (bool, error).
          detail: Neither changes the design; correct the Log line and expect the Task 1.3 snippet's Forget call to need the two-value form.
          family: unbacked-claims-about-existing-code
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-12T17:49:41-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: pendingWords is no-band-or-no-item; harvestDeck takes a batch and bands then authors it; TestABacklogDrainsOnBothHalves pins it.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Forget-mid-job and the job's context are in the ARCH-ORDER list; matches interrupt.go:50-58 and :83.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: FuzzModelLookups in Task 2.2, seeded from the table's malformed forms, with the invariant stated.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Log corrected to three goroutines; Task 1.3 calls Forget in the two-value form.
          round: 2
      findings:
        - id: PQ-5
          severity: Minor
          title: Task 1.6 cites config.go's Resolve comment as calling harvest batch-only; no such file or comment exists
          detail: 'cmd/define/config.go does not exist and internal/llm/config.go:86-97 says nothing about harvest; the batch-only comments are harvest.go:102-108, atlas/define.md:1451 and README.md:362. The guards section also says TestPlanTableStatusMatchesTheChangeWindow checks the first commit touching a file, but its window is merge-base..HEAD (repo_guard_test.go:1486). Second finding in this family: the rule is that prose citations get no mechanical check, so add a grep resolution pass over every backticked path and symbol to the plan''s guards step. Prevalence this round: 2 of about 30 checked.'
          family: unbacked-claims-about-existing-code
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-12T18:17:27-07:00"
      agent: claude
      dispose:
        - id: PQ-5
          disposition: addressed
          note: Task 1.6 names harvest.go:104, atlas/define.md:1451 and README.md:385; the guard is described as merge-base..HEAD; the plan states the prose-citation grep pass and every backticked symbol resolved this round.
          round: 3
      blocked: false
content_hash: adc0920fab4ea2281bf84e9ce46ef122163ec6f17ecaa53b8c783a4c1338b6d4
---

# Gate ledger — tools#54 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-12T17:41:41-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `trigger-count-matches-spec-definition` countUnbanded counts a strict subset of the Spec's "new" words, and bands-first harvesting makes the backlog stall
  runHarvest bands every unbanded word before runAuthoring authors any (harvest.go:144-206, :223), so under bgBudget=60 a backlog is banded across jobs while authoring gets little; once unbanded hits 0 the count stops firing though most words still have no item. Count what harvest would work on (no band OR no items) or make harvestDeck band only what it can also author.
- **PQ-2** [Minor] `order-enumeration-names-interrupting-events` Enumerate /forget mid-job and state the job's context derives from the session ctx
  Forget removes word/facts/items; a job mid-authoring re-creates facts/items for a forgotten word. interrupter.Fire with no scope cancels the session ctx (quit); a scoped Ctrl-C must not reach the runner.
- **PQ-3** [Minor] `fuzz-parsers-over-untrusted-input` modelLookups parses a hand-edited file; add a fuzz seeded with the table's malformed forms
  cloze_fuzz_test.go is the precedent shape.
- **PQ-4** [Minor] `unbacked-claims-about-existing-code` Log says the TUI has one goroutine; rawterm.go:87/:242 and interrupt.go:83 already run three. Forget returns (bool, error).
  Neither changes the design; correct the Log line and expect the Task 1.3 snippet's Forget call to need the two-value form.

## Round 2 — 2026-09-12T17:49:41-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — pendingWords is no-band-or-no-item; harvestDeck takes a batch and bands then authors it; TestABacklogDrainsOnBothHalves pins it.
- PQ-2 — addressed — Forget-mid-job and the job's context are in the ARCH-ORDER list; matches interrupt.go:50-58 and :83.
- PQ-3 — addressed — FuzzModelLookups in Task 2.2, seeded from the table's malformed forms, with the invariant stated.
- PQ-4 — addressed — Log corrected to three goroutines; Task 1.3 calls Forget in the two-value form.

### Raised

- **PQ-5** [Minor] `unbacked-claims-about-existing-code` Task 1.6 cites config.go's Resolve comment as calling harvest batch-only; no such file or comment exists
  cmd/define/config.go does not exist and internal/llm/config.go:86-97 says nothing about harvest; the batch-only comments are harvest.go:102-108, atlas/define.md:1451 and README.md:362. The guards section also says TestPlanTableStatusMatchesTheChangeWindow checks the first commit touching a file, but its window is merge-base..HEAD (repo_guard_test.go:1486). Second finding in this family: the rule is that prose citations get no mechanical check, so add a grep resolution pass over every backticked path and symbol to the plan's guards step. Prevalence this round: 2 of about 30 checked.

## Round 3 — 2026-09-12T18:17:27-07:00 (claude) — passed

### Disposed

- PQ-5 — addressed — Task 1.6 names harvest.go:104, atlas/define.md:1451 and README.md:385; the guard is described as merge-base..HEAD; the plan states the prose-citation grep pass and every backticked symbol resolved this round.

## Open findings

(none — every finding has been disposed)

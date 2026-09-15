---
gate: boundary-review
issue: 66
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-15T15:58:21-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Closing an answer removes its final row background
          detail: cmd/define/screen.go:882–883 deletes paint metadata on every plain write to a partial row. After ownedAnswerWrapWriter.Flush emits the final row, runAsk appends a plain newline on normal completion and cancellation, erasing its tint. A scratch overlay regression fails on this sequence. Preserve metadata for terminator-only writes and cover complete runAsk termination through live and append-only sinks. ARCH-PURPOSE and ARCH-ORDER.
          family: finalized-row-metadata-preservation
          round: 1
        - id: BR-2
          severity: Critical
          title: The Core concepts table contradicts the pinned implementation
          detail: The plan's lines 41 and 44 claim an unchanged languageText was modified and locate paintLanguageRow in the wrong file; line 54 attributes new streaming behavior to the legacy answerWrapWriter rather than ownedAnswerWrapWriter. The broader test run also fails the declaration-level status guard for liveScreen and answerWrapWriter. Reconcile the table with actual entities and append a Revisions entry; no wording-presence test is needed.
          family: core-concept-traceability
          round: 1
      blocked: true
---

# Gate ledger — tools#66 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-15T15:58:21-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `finalized-row-metadata-preservation` Closing an answer removes its final row background
  cmd/define/screen.go:882–883 deletes paint metadata on every plain write to a partial row. After ownedAnswerWrapWriter.Flush emits the final row, runAsk appends a plain newline on normal completion and cancellation, erasing its tint. A scratch overlay regression fails on this sequence. Preserve metadata for terminator-only writes and cover complete runAsk termination through live and append-only sinks. ARCH-PURPOSE and ARCH-ORDER.
- **BR-2** [Critical] `core-concept-traceability` The Core concepts table contradicts the pinned implementation
  The plan's lines 41 and 44 claim an unchanged languageText was modified and locate paintLanguageRow in the wrong file; line 54 attributes new streaming behavior to the legacy answerWrapWriter rather than ownedAnswerWrapWriter. The broader test run also fails the declaration-level status guard for liveScreen and answerWrapWriter. Reconcile the table with actual entities and append a Revisions entry; no wording-presence test is needed.

## Open findings

- **BR-1** [Critical] `finalized-row-metadata-preservation` Closing an answer removes its final row background
- **BR-2** [Critical] `core-concept-traceability` The Core concepts table contradicts the pinned implementation

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
    - "n": 2
      timestamp: "2026-09-15T16:08:13-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: output_screen.go:178 centralizes partial-row invalidation for plain and structured writes. Success/cancellation regressions pass; a scratch overlay restoring unconditional invalidation makes both live runAsk regressions fail.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The plan's Core concepts table now identifies unchanged declarations, locates paintLanguageRow in language_row.go, and assigns streaming behavior to ownedAnswerWrapWriter. The pinned correction includes a Revisions entry; declaration/status guards pass.
          round: 2
      findings:
        - id: BR-3
          severity: Critical
          title: Tinted one-shot output truncates overlong words
          detail: cmd/define/output_layout.go:221 passes wrapped logical lines directly to paintLanguageRow, whose language_row.go:55 stops at the terminal width. wrapText deliberately preserves overlong words, so the painter discards their remaining characters. A scratch regression through renderDefinitionOutput and serializeOutput at width 20 renders anticonstitucionalmente as anticonstitucionalme under both dark and light profiles; off preserves it. Split overflowing output into physical rows before painting, preserving text and projected metadata. Cover overlong headwords, body tokens, and wide display units while retaining the specified historical clipping policy. ARCH-PURPOSE.
          family: terminal-serialization-preserves-source
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-15T17:12:05-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Finalized-row metadata preservation remains covered by the passing answer termination tests.
          round: 3
        - id: BR-2
          disposition: addressed
          note: The revised Core concepts table matches the inspected declarations, locations and adapters.
          round: 3
        - id: BR-3
          disposition: addressed
          note: HEAD preserves overlong dictionary source and wide display units. Reverting output_layout.go's fix in a temporary overlay makes TestTerminalSerializationPreservesOverlongDictionarySource fail; HEAD passes.
          round: 3
      findings:
        - id: BR-4
          severity: Critical
          title: Practice click targets use obsolete geometry after physical wrapping
          detail: cmd/define/practice_language.go:48 maps actions with wrapMovedRegions, while line 45 renders text through outputWrappedRows. At width 20, a presentation containing 23 a characters followed by newline and hola places the hola action on the preceding aaa row. This is the 2nd finding in family terminal-serialization-preserves-source. Enforce one geometry projection for text and all associated metadata; enumerate structured-output consumers and remove parallel coordinate mappings. ARCH-DRY, ARCH-PURPOSE.
          family: terminal-serialization-preserves-source
          round: 3
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

## Round 2 — 2026-09-15T16:08:13-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — output_screen.go:178 centralizes partial-row invalidation for plain and structured writes. Success/cancellation regressions pass; a scratch overlay restoring unconditional invalidation makes both live runAsk regressions fail.
- BR-2 — addressed — The plan's Core concepts table now identifies unchanged declarations, locates paintLanguageRow in language_row.go, and assigns streaming behavior to ownedAnswerWrapWriter. The pinned correction includes a Revisions entry; declaration/status guards pass.

### Raised

- **BR-3** [Critical] `terminal-serialization-preserves-source` Tinted one-shot output truncates overlong words
  cmd/define/output_layout.go:221 passes wrapped logical lines directly to paintLanguageRow, whose language_row.go:55 stops at the terminal width. wrapText deliberately preserves overlong words, so the painter discards their remaining characters. A scratch regression through renderDefinitionOutput and serializeOutput at width 20 renders anticonstitucionalmente as anticonstitucionalme under both dark and light profiles; off preserves it. Split overflowing output into physical rows before painting, preserving text and projected metadata. Cover overlong headwords, body tokens, and wide display units while retaining the specified historical clipping policy. ARCH-PURPOSE.

## Round 3 — 2026-09-15T17:12:05-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — Finalized-row metadata preservation remains covered by the passing answer termination tests.
- BR-2 — addressed — The revised Core concepts table matches the inspected declarations, locations and adapters.
- BR-3 — addressed — HEAD preserves overlong dictionary source and wide display units. Reverting output_layout.go's fix in a temporary overlay makes TestTerminalSerializationPreservesOverlongDictionarySource fail; HEAD passes.

### Raised

- **BR-4** [Critical] `terminal-serialization-preserves-source` Practice click targets use obsolete geometry after physical wrapping
  cmd/define/practice_language.go:48 maps actions with wrapMovedRegions, while line 45 renders text through outputWrappedRows. At width 20, a presentation containing 23 a characters followed by newline and hola places the hola action on the preceding aaa row. This is the 2nd finding in family terminal-serialization-preserves-source. Enforce one geometry projection for text and all associated metadata; enumerate structured-output consumers and remove parallel coordinate mappings. ARCH-DRY, ARCH-PURPOSE.

## Open findings

- **BR-4** [Critical] `terminal-serialization-preserves-source` Practice click targets use obsolete geometry after physical wrapping

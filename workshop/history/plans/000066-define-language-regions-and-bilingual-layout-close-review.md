# Boundary Review — tools#66 (whole-issue close)

| field | value |
|-------|-------|
| issue | 66 — define: preserve bilingual layout and tint complete language regions |
| repo | tools |
| issue file | workshop/issues/000066-define-language-regions-and-bilingual-layout.md |
| boundary | whole-issue close |
| milestone | — |
| window | 05a8e8de6e0e22312e95c6649a74936ecb139591..77a5cfedbdebbcd6c7d3a47eb57b9c8ae9e0f2b5 |
| command | sdlc close --issue 66 |
| reviewer | codex |
| timestamp | 2026-09-15T15:58:21-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The structural parser and shared row painter are well covered, but answer termination removes the final row’s tint in interactive output. The plan also contradicts the implementation, and the broader test run fails its plan-status guard. No repository files were changed.

## 1. Strengths

- Oxford parsing consolidates structure and source ownership, validates correspondence, and preserves readable text on formatting failure.
- Paint metadata remains separate from selectable text; tests cover blank cells, exclusions, selection, resize, and nested-screen transfer.
- README and atlas describe the new section backgrounds and register the new conformance checks.

## 2. Critical findings

**Closing an answer erases its finalized row background.**
At `cmd/define/screen.go:882–883`, any plain write to a partial row deletes its paint metadata. `ownedAnswerWrapWriter.Flush` emits the final row without a newline; `runAsk` then calls `fmt.Fprintln(out)` on normal completion (`ask.go:216–217`) and cancellation (`ask.go:188`). Consequently, a one-row target-language answer loses all tint; longer answers lose their final row’s tint. **ARCH-PURPOSE, ARCH-ORDER.**

A scratch overlay regression reproduces this: finalize `[lang=es]hola[/lang]`, verify its background, append the closing newline, and observe the background disappear. Preserve metadata for a terminator-only write, and test the complete `runAsk` success/cancellation paths against live and append-only sinks.

**Core concepts table does not describe the delivered entities.**
In `workshop/plans/000066-define-language-regions-and-bilingual-layout-plan.md`:

- Line 41 claims `languageText` changed; its file is unchanged.
- Line 44 locates `paintLanguageRow` in `language_style.go`; it lives in `language_row.go`.
- Line 54 attributes the new streaming behavior to `answerWrapWriter`; production uses the new `ownedAnswerWrapWriter`.
- The repository’s declaration-level guard also rejects the `liveScreen` and `answerWrapWriter` “modified” rows.

Reconcile the table with the actual entities and append a revision explaining the changes. Severity follows the explicit Core concepts review contract.

## 3. Important findings

None additional.

## 4. Minor findings

None.

## 5. Test coverage notes

- Focused parser/definition tests: **passed**.
- Focused streaming, painting, output, and practice tests: **passed**.
- Scratch closing-newline regression: **failed**, confirming the defect.
- `go test ./cmd/define/... -count=1`: **failed** in `TestPlanTableStatusMatchesTheChangeWindow`; other listed packages passed.
- Pinned-range `git diff --check`: **passed**.
- Native/PTY conformance was inspected, not rerun.

Existing live/append comparison tests finish before `runAsk` appends its closing newline, leaving this consumer interaction uncovered.

## 6. Architectural notes

| Principle | Result |
|---|---|
| ARCH-DRY | Pass: shared structural parser and row painter. |
| ARCH-PURE | Pass: parsing, ownership, layout, and painting have direct pure tests. |
| ARCH-PURPOSE | **Flag:** final answer rows lose the promised treatment. |
| ARCH-MOCK | Pass: captured native records, stateful SSE fake, and conformance paths exist. |
| ARCH-CONSTRAINTS | Pass: source and pending-text limits remain explicit. |
| ARCH-SECURE | Pass: ownership requires validated provenance and correspondence. |
| ARCH-ORDER | **Flag:** the subsequent newline invalidates finalized ownership. |
| ARCH-FUNERAL | Pass: metadata follows existing response/screen lifetimes; no new persistent runtime family. |

## 7. Plan revision recommendations

Append a `## Revisions` entry correcting the entity names, paths, and statuses above. Add complete answer termination—including the closing newline—to the streaming verification contract.

```findings
findings:
  - id: new
    severity: Critical
    family: finalized-row-metadata-preservation
    title: |
      Closing an answer removes its final row background
    detail: |
      cmd/define/screen.go:882–883 deletes paint metadata on every plain write to a partial row. After ownedAnswerWrapWriter.Flush emits the final row, runAsk appends a plain newline on normal completion and cancellation, erasing its tint. A scratch overlay regression fails on this sequence. Preserve metadata for terminator-only writes and cover complete runAsk termination through live and append-only sinks. ARCH-PURPOSE and ARCH-ORDER.
  - id: new
    severity: Critical
    family: core-concept-traceability
    title: |
      The Core concepts table contradicts the pinned implementation
    detail: |
      The plan's lines 41 and 44 claim an unchanged languageText was modified and locate paintLanguageRow in the wrong file; line 54 attributes new streaming behavior to the legacy answerWrapWriter rather than ownedAnswerWrapWriter. The broader test run also fails the declaration-level status guard for liveScreen and answerWrapWriter. Reconcile the table with actual entities and append a Revisions entry; no wording-presence test is needed.
```

---

## Re-review — 2026-09-15T16:08:13-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 66 — define: preserve bilingual layout and tint complete language regions |
| repo | tools |
| issue file | workshop/issues/000066-define-language-regions-and-bilingual-layout.md |
| boundary | whole-issue close |
| milestone | — |
| window | 05a8e8de6e0e22312e95c6649a74936ecb139591..a365ee51fc839599f95be971beab43d0776e0d3e |
| command | sdlc close --issue 66 |
| reviewer | codex |
| timestamp | 2026-09-15T16:08:13-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Both prior findings are addressed. Structural parsing, section ownership, and termination handling are well covered. One new correctness bug blocks shipping: tinted one-shot output silently truncates words wider than the terminal.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      output_screen.go:178 centralizes partial-row invalidation for plain and structured writes. Success/cancellation regressions pass; a scratch overlay restoring unconditional invalidation makes both live runAsk regressions fail.
  - id: BR-2
    disposition: addressed
    note: |
      The plan's Core concepts table now identifies unchanged declarations, locates paintLanguageRow in language_row.go, and assigns streaming behavior to ownedAnswerWrapWriter. The pinned correction includes a Revisions entry; declaration/status guards pass.
findings:
  - id: new
    severity: Critical
    family: terminal-serialization-preserves-source
    title: |
      Tinted one-shot output truncates overlong words
    detail: |
      cmd/define/output_layout.go:221 passes wrapped logical lines directly to paintLanguageRow, whose language_row.go:55 stops at the terminal width. wrapText deliberately preserves overlong words, so the painter discards their remaining characters. A scratch regression through renderDefinitionOutput and serializeOutput at width 20 renders anticonstitucionalmente as anticonstitucionalme under both dark and light profiles; off preserves it. Split overflowing output into physical rows before painting, preserving text and projected metadata. Cover overlong headwords, body tokens, and wide display units while retaining the specified historical clipping policy. ARCH-PURPOSE.
```

### 1. Strengths

- Oxford structure and provenance share one bounded parser, with source-order conservation tests.
- Paint metadata stays separate from selectable text; resize, selection, and nested-screen transfer have direct coverage.
- BR-1 now has complete caller-path regressions, independently confirmed by mutation.
- README and atlas describe the changed presentation and new conformance coverage.

### 2. Critical findings

- **Overlong-word truncation:** `cmd/define/output_layout.go:221`, `cmd/define/language_row.go:55`. Fix terminal serialization as described above; the missing suffix is lost before reaching the terminal.

### 3. Important findings

None.

### 4. Minor findings

None.

### 5. Test coverage notes

Passed:

- `go test ./cmd/define/... -count=1`
- Focused race tests for termination, streaming, and screen output.
- Plan guards and `internal/conformance` tests.
- Pinned-range `git diff --check`.

Scratch regressions reproduced truncation through both direct serialization and definition rendering. Native/PTY conformance and visual inspection were not rerun. Repository files remain unchanged.

### 6. Architectural notes

| Principle | Result |
|---|---|
| ARCH-DRY | Pass: shared parser, painter, and partial-row invalidation rule. |
| ARCH-PURE | Pass: parsing, layout, and painting remain pure. |
| ARCH-PURPOSE | **Flag:** terminal serialization violates source preservation. |
| ARCH-MOCK | Pass: captured dictionary records and stateful SSE replay exercise existing seams. |
| ARCH-CONSTRAINTS | Pass: source and pending-stream bounds retained. |
| ARCH-SECURE | Pass: source validation and neutral fallback precede trusted ownership. |
| ARCH-ORDER | Pass: finalized ownership and terminating writes now agree; mutation verified. |
| ARCH-FUNERAL | Pass: metadata follows existing screen/response lifetimes; no new runtime artifact family. |

### 7. Plan revision recommendations

Append a `## Revisions` entry adding overlong-token conservation to serializer verification, explicitly distinguishing complete one-shot output from historical viewport clipping.

---

## Re-review — 2026-09-15T17:12:05-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 66 — define: preserve bilingual layout and tint complete language regions |
| repo | tools |
| issue file | workshop/issues/000066-define-language-regions-and-bilingual-layout.md |
| boundary | whole-issue close |
| milestone | — |
| window | 05a8e8de6e0e22312e95c6649a74936ecb139591..dc59b226d021760ce61bcfbb6555714199b4a260 |
| command | sdlc close --issue 66 |
| reviewer | codex |
| timestamp | 2026-09-15T17:12:05-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-3’s text-truncation fix is verified, but practice output still projects click targets using different geometry from its text. A regression probe confirms a target can land on the wrong word. The existing define suite passes; this uncovered case blocks shipping.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Finalized-row metadata preservation remains covered by the passing answer termination tests.
  - id: BR-2
    disposition: addressed
    note: |
      The revised Core concepts table matches the inspected declarations, locations and adapters.
  - id: BR-3
    disposition: addressed
    note: |
      HEAD preserves overlong dictionary source and wide display units. Reverting output_layout.go's fix in a temporary overlay makes TestTerminalSerializationPreservesOverlongDictionarySource fail; HEAD passes.
findings:
  - id: new
    severity: Critical
    family: terminal-serialization-preserves-source
    title: |
      Practice click targets use obsolete geometry after physical wrapping
    detail: |
      cmd/define/practice_language.go:48 maps actions with wrapMovedRegions, while line 45 renders text through outputWrappedRows. At width 20, a presentation containing 23 a characters followed by newline and hola places the hola action on the preceding aaa row. This is the 2nd finding in family terminal-serialization-preserves-source. Enforce one geometry projection for text and all associated metadata; enumerate structured-output consumers and remove parallel coordinate mappings. ARCH-DRY, ARCH-PURPOSE.
```

1. **Strengths**

   - Oxford parsing preserves ordered structure and provenance, with neutral fallback and a visible formatting diagnostic.
   - Shared row painting keeps synthetic padding separate from selectable source.
   - BR-3 regressions cover long headwords/body tokens, wide/combining units, exclusions and actions.
   - README, atlas and conformance registry updates describe the new surface.

2. **Critical findings**

   - [practice_language.go:48](/Users/xianxu/workspace/tools/cmd/define/practice_language.go:48): text and actions follow different wrapping paths. The temporary production-path probe produced rows `["", "aaaaaaaaaaaaaaaaaaaa", "aaa", "hola"]`, with the `hola` action attached to row 2. Project actions through the same physical layout as text, and add a regression through `writePracticePresentation` into `liveScreen`.

3. **Important findings**

   None separately.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed `go test ./cmd/define/... -count=1`.
   - Passed focused terminal-serialization and physical-output regressions.
   - BR-3 mutation failed as expected.
   - New practice-coordinate probe failed on HEAD.
   - Pinned diff check passed. Native/PTY conformance was inspected, not rerun. Repository files were unchanged.

6. **Architectural notes**

   - **ARCH-DRY — flag:** practice retains a parallel coordinate mapper.
   - **ARCH-PURE — pass:** parsing, layout and painting remain independently testable.
   - **ARCH-PURPOSE — flag:** preserving mouse regions is incomplete.
   - **ARCH-MOCK — pass:** existing capture/fake seams and native/PTY checks are extended.
   - **ARCH-CONSTRAINTS — pass:** bounded source, metadata and pending-stream handling remain.
   - **ARCH-SECURE — pass:** provenance validation and neutral fallback are preserved.
   - **ARCH-ORDER — pass:** finalized ownership and termination paths have behavioral coverage.
   - **ARCH-FUNERAL — pass:** metadata follows response/screen lifetime; no new runtime persistent store.

7. **Plan revision recommendations**

   Append a `## Revisions` entry naming the invariant: **text, actions and exclusions must share physical-row geometry**. Record the structured-consumer enumeration, the practice mapper correction and its regression evidence.

---

## Re-review — 2026-09-15T17:50:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 66 — define: preserve bilingual layout and tint complete language regions |
| repo | tools |
| issue file | workshop/issues/000066-define-language-regions-and-bilingual-layout.md |
| boundary | whole-issue close |
| milestone | — |
| window | 05a8e8de6e0e22312e95c6649a74936ecb139591..05f336cb2c5b97f72bd713c0fbb41d383cd15925 |
| command | sdlc close --issue 66 |
| reviewer | codex |
| timestamp | 2026-09-15T17:50:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned implementation satisfies the documented layout and language-region contracts. BR-4 is addressed: both production-path regressions pass at HEAD and fail when the preceding implementation is substituted through a temporary Go overlay. No new blocking findings. Confidence is limited by unavailable native dictionary access.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      practice_language.go and practice_output.go project actions through layoutOutput; output_screen.go also projects the regionWriter fallback. Both physical_output_test.go regressions pass at HEAD and fail against the preceding implementation with misplaced action rows.
  - id: BR-1
    disposition: addressed
    note: |
      invalidatePartialPaint preserves newline-only termination. Final-row success/cancellation and plain/structured termination regressions pass.
  - id: BR-2
    disposition: addressed
    note: |
      The revised Core concepts tables match the declarations and changes in output_layout.go, language_row.go, output_screen.go, practice_output.go and screen.go.
  - id: BR-3
    disposition: addressed
    note: |
      Physical splitting precedes terminal painting; overlong-token, wide-glyph, source-conservation and coordinate-projection regressions pass.
```

1. **Strengths**
   - Practice actions and rendered text now share physical geometry, with regression coverage for both affected sink paths.
   - Oxford parsing preserves structural hierarchy and checks source correspondence before trusting ownership.
   - Cell-level tests verify full backgrounds, answer exclusions, selection, resize and clean source text.
   - The define README and atlas document the revised behavior and conformance checks.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - `go test ./cmd/define/... -count=1`: passed.
   - Focused layout, parser, streaming and action tests under `-race`: passed.
   - BR-4 mutation check: both regressions failed without the fix.
   - Pinned-range `git diff --check`: passed.
   - Strict native/PTY conformance: blocked by inaccessible Oxford Spanish dictionary; actual native output was not independently verified this round.

6. **Architecture**
   - **ARCH-DRY — pass:** shared parser, geometry projection and painter.
   - **ARCH-PURE — pass:** parsing, layout and painting remain separate from terminal IO.
   - **ARCH-PURPOSE — pass:** lookup, full reveals, practice chrome and streamed answers are covered.
   - **ARCH-MOCK — pass:** captured native records and stateful SSE replay exercise existing seams; live checks exist.
   - **ARCH-CONSTRAINTS — pass:** source/depth limits and pending-stream bounds have coverage.
   - **ARCH-SECURE — pass:** invalid source correspondence falls back visibly and neutrally.
   - **ARCH-ORDER — pass:** ownership finalization, cancellation and pending resize have sequence coverage; screen ingress is synchronized.
   - **ARCH-FUNERAL — pass:** runtime metadata follows response/screen lifetime; no new persistent runtime artifacts.

7. **Plan revisions:** None required. The BR-4 revision states the invariant and enumerates consumers; close/publication remains correctly unchecked.

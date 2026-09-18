---
id: 000076
status: working
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T12:02:43-07:00
---

# define: delete or re-use #66's orphaned source-provenance chain

## Problem

#70 deleted the per-fragment language tint, which production never reached
(`definitions.go` zeroed the tint before `Render`; production tints whole
SECTIONS as row paint). That left #66's source-provenance data with no
production reader:

- `RenderOpts.Language` — written at `definitions.go:80`, read nowhere.
- `Entry.source`, set from `definitionSection.source` (`definitions.go:86-87`).
- `definitionSection.source`, filled from `bilingualDocument.native`
  (`definitions.go:142-144`).
- `bilingualDocument.native` (`bilingual_layout.go:105`) and its only producer,
  `projectDictionaryText`.
- `Sense.sourceAt`/`sourceKnown` and `Example.sourceAt`/`sourceKnown`
  (`parse.go:187-202`, written at `:760`, `:813`).
- `bilingualLanguageText` — no production caller.

`bilingualDocument.source` is NOT part of this: the bilingual layout reads it.
Go does not report unused struct fields, so nothing forces the question.

Tests that check only this residue, to go with it if it goes:
`TestDictionaryParserSourceOffsets`, `TestDictionarySourceProvenanceCorpus`, the
`projectDictionaryText` half of `TestDictionaryProjectionExactOccurrenceAndFallback`,
and the provenance check in `TestBilingualNativeLanguageOwnership` (tagged).

## Spec

Decide: DELETE the chain and its tests, or give it a consumer (a feature that
needs per-fragment language ownership). Default: delete — it is data no output
reads. Found during #70 M1 (see #70's Log, 2026-09-17, "M1 Task 3").

**Decided 2026-09-18: delete.** The operator said: "generally, I'd like to delete
unused features." Tracing the chain from its readers finds more than the list
above, and the sweep takes all of it (ARCH-PURPOSE):

- the parser's `sourceBase ...int` variadics and `sourceOffset`/`advanceSource`.
  These are every change `parse.go` has had since `62c6a66^`, so the file goes
  back to exactly that;
- the Oxford parser's `bilingualNode.lang` class→language switch and the spans
  on `bilingualDocument.source`, which becomes a plain string. The layout reads
  only its text;
- the dead `ro.Tint = tintPolicy{}` write in `renderDefinitionOutput`;
- `projectLanguageText`'s strict mode. `projectDisplayText`, used by practice
  output, is the one live function in the chain's file. It keeps the display
  mode and moves to `language_text.go`, and `dictionary_language.go` goes.

Output does not change. Durable plan:
`workshop/plans/000076-orphaned-source-provenance-plan.md`.

## Done when

- Either every member listed above is gone with its tests (grep proves no
  reader and no writer remains), or a consumer reads it and a test pins that.

## Plan

Single pass, one `sdlc close`. Task detail lives in the durable plan.

- [ ] Task 1: add tests that are valid today. That means a unit test and a fuzz
  pinning `projectDisplayText` in both directions, renaming the two tests whose
  names describe the deleted concept, and dropping the unread `Language:` from
  test literals.
- [ ] Task 2: remove every writer and reader. Restore `parse.go` to `62c6a66^`
  (an empty diff proves it), strip `definitions.go`, and delete the tests of
  what was removed.
- [ ] Task 3: delete the declarations and producers. Then run mutation checks
  M-join and M-break.
- [ ] Task 4: sweep the atlas, grep for absence, run the live conformance checks
  and diff the binary's output before and after. Log each result.

Every commit builds, including under `-tags conformance`, and passes the full
suite.

The tests the Problem section names, deleted here:

| test | file | status |
|---|---|---|
| `TestDictionaryParserSourceOffsets` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestDictionarySourceProvenanceCorpus` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestDictionaryProjectionExactOccurrenceAndFallback` | `cmd/define/dictionary_language_test.go` | deleted |
| `TestBilingualNativeLanguageOwnership` | `cmd/define/bilingual_conformance_test.go` | deleted |

## Log

### 2026-09-17

### 2026-09-18

- Claimed. Operator decision: delete ("generally, I'd like to delete unused
  features"). Traced the chain from its readers rather than from the listed
  members. That added the parser variadics/helpers, `bilingualNode.lang`,
  `doc.source`'s spans, `projectLanguageText`'s strict flag, and the dead
  `ro.Tint` zeroing. `diff 62c6a66^:cmd/define/parse.go` shows offsets are
  parse.go's only change since #65.
- Baseline on `main` (`go build`, `go vet`, `go test -count=1 ./...`): all green
  except `TestLanguagePromptStartup`, `TestLanguageTintInvocation` and
  `TestSavedSchemeGovernsALookup`. Those failed with "operation not permitted"
  under the Bash sandbox (pty), and passed when re-run outside it. So `main`
  is green.
- Durable plan written (full flow: five code files is outside the quick shell).
  I first wrote here that the plan's `deleted` rows are red-first pins under
  `TestPlanTablesNameEntitiesThatExist`. That was wrong: the plan gate (PQ-2)
  measured it. `currentTruthOnly` strips every name a `| … | deleted |` row
  carries, so those rows are never checked.
- Plan gate round 1: PQ-1 (Task 2 would not build) and PQ-2 (the predicted red
  set was unmeasured). The fix is the class rule rather than the two sites:
  tasks are ordered tests → writers/readers → declarations, so every commit is
  green. PQ-3 added `FuzzDisplayProjection`. Recorded in the plan's
  `## Revisions`.

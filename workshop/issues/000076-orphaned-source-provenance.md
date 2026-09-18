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

## Done when

- Either every member listed above is gone with its tests (grep proves no
  reader and no writer remains), or a consumer reads it and a test pins that.

## Plan

- [ ]

## Log

### 2026-09-17

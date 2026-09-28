---
id: '000076'
status: done
started: 2026-09-18T12:02:43-07:00
created: 2026-09-17
updated: 2026-09-18
estimate_hours: 1.92
actual_hours: 1.32
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

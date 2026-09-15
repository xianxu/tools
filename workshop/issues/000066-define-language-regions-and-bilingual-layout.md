---
id: 000066
status: working
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
started: 2026-09-15T14:48:20-07:00
---

# define: preserve bilingual layout and tint complete language regions

## Problem

The operator's screenshot of Spanish `rendir` shows patchwork text-run tint and
an Oxford Spanish–English supplement flattened into a dense paragraph. They
explicitly want complete language rows/regions colored, with the dictionary's
formatting preserved. This supersedes #65's text-only/no-padding tint decision.


## Spec

- Paint continuous language-region backgrounds across the full available row,
  including indentation, internal spacing, trailing area and blank rows inside
  the region. Preserve foreground colors, bold/italic emphasis, sense numbering,
  headings, examples, selection and semantic answer markings.
- Each returned dictionary section has one uniform full-width background,
  including its headings, blank rows and embedded foreign-language examples.
  Primary sections use verified source language; Oxford uses its explicit
  English-explanation presentation role. With `/lang es`, Spanish primary is
  tinted and all of Oxford is normal. Keep source provenance separate from this
  visual role; unknown fallback dictionaries remain neutral.
- Preserve Oxford's structural HTML through parsing/rendering: part-of-speech
  groups, senses, sub-senses, example/translation pairs and idioms must remain
  readable. Coloring must not collapse or reorder content.
- Keep each Spanish example and its English translation together on one logical
  row, wrapping naturally at terminal width. Preserve headings, sense structure
  and separate example pairs; never split a pair merely at its language boundary.
  Use the section background throughout. Full practice reveals follow this rule.
- Keep dark/light/off, no-color/pipes, clean clipboard/history, mouse regions,
  wrapping and terminal resize behavior. Cover ordinary lookup and full practice
  reveals; apply the same region treatment to practice and model output.


## Done when

- `rendir` at wide and narrow widths shows complete language panels, without
  strips or holes around shorter lines, foreground resets or wrapped examples.
- Oxford content has visible POS/sense/example/translation structure with all
  source content retained in source order, under both tint on and off.
- Dark and light actual output is reviewed visually as well as checked for ANSI
  bytes; tests verify background on blank cells, not only text cells.
- Selection/copy, historical scrollback, resize, click maps, practice answer marks,
  model streaming termination and no-color/plain output remain correct.
- Unknown dictionary source remains neutral; embedded foreign examples never
  introduce background stripes within an otherwise uniform dictionary section.


## Plan

- [ ] Trace source formatting and terminal painting; settle a concrete layout and
  durable implementation plan with regressions and operator review.
- [ ] Implement structural bilingual rendering and shared full-region painting;
  verify actual `rendir` output, all consumers and required checks, then close.


## Log

### 2026-09-15

- Created and claimed #66 immediately after operator screenshot feedback; ran
  start-plan. No runtime edits. This is a correction to #65's visual contract.
- Root causes: `styleLanguageText` deliberately limits background to ink bounds;
  `spanishDefinitions.supplement` retains Text plus language spans, while
  `renderDefinitions` reparses flat Oxford Text via ParseEntry. Structural HTML
  hierarchy and formatting never reach the renderer.
- Read-only investigations delegated for Oxford source structure and screen /
  selection / wrap region-painting seams. Durable plan and visual layout pending.

- Native `rendir` verification found explicit A/B/C gramb groups, nested numbered
  and lettered semb senses, and paired exg examples/translations. Read-only
  captures: `/tmp/define66-rendir-native.json` and `-tree.txt`.
- Screen investigation: padding must remain paint-only; stored strings currently
  feed selection cells and exit transcript. Extend structured output alongside
  click regions rather than infer language from ANSI or pad source strings.
- Draft [implementation plan](../plans/000066-define-language-regions-and-bilingual-layout-plan.md)
  and `/tmp/define66-region-preview.html` show full-width panels and restored Oxford
  rows. Fresh spec/plan review pending; implementation and estimate have not begun.

- Fresh plan review requested explicit physical-row streaming finalization for
  live and append-only sinks. Revised to bounded unfinished-row buffering with
  chunk/sink/cancel/resize agreement tests. Re-review approved with no remaining
  important findings. Concrete layout/plan ready for operator review; no runtime
  code changed and estimate still deferred.

## Revisions

### 2026-09-15 — Uniform dictionary sections

Operator accepts the general preview but rejects changing background within a
single dictionary definition. One dictionary result is one visual region, even
when it contains both languages. Supersedes the draft separate-language-row tint
policy for dictionary output; structural formatting and foreground emphasis stay.
The preview and durable plan now reflect this correction. Runtime remains unchanged.

### 2026-09-15 — Bilingual pairs stay together

Operator answered the pending layout question: “Keep bilingual pairs on one row;
use the section background.” The preview and plan now keep each example and
translation inline, with natural wrapping. Earlier separate-language-row drafts
are superseded; this layout question is resolved.

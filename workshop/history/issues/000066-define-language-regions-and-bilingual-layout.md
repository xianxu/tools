---
id: 000066
status: done
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours: 7.117
started: 2026-09-15T14:48:20-07:00
actual_hours: 4.00
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

- [x] Trace source formatting and terminal painting; settle a concrete layout and
  durable implementation plan with regressions and operator review.
- [x] Implement structural bilingual rendering and shared full-region painting;
  verify actual `rendir` output, all consumers and required checks, then close.


## Log

### 2026-09-15
- 2026-09-15: closed — Postcommit full Go suite passed (define 127.818s); focused race and strict installed Oxford/PTY dark/light/off checks passed. BR4 regressions fail before and pass after for practice-to-liveScreen and regionWriter actions following hard-wrapped source. Structured consumers enumerated in plan. Earlier vet/Linux build/fuzz/mutations and actual dark/light visual inspection passed.; review verdict: SHIP

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

- Implementation gate raised PQ-1 (function-level test strategy). Addressed by
  naming parser/ownership functions and mapping each risky function to adversarial
  classes and independent guards in the plan. Product design unchanged; gate
  recheck pending, no runtime edits or estimate yet.

- Change-code passed: plan-quality round 2 accepted PQ-1; estimate-quality INFO
  (no blocking findings); implementation branch created. Review notes optimistic
  streaming/verification allowances. Estimate is full-issue ship wall-clock,
  including completed planning, not remaining time. Plan review/revision is in
  issue-spec design; unit/race/fuzz/mutation execution is in each owning technical
  item; full-suite/build and final review verification are in close-review;
  actual native/PTY dark/light inspection is in real-api-discovery. These are
  small provisional allowances; do not reduce required verification to fit them.
- Context checkpoint before implementation under AGENTS §14. Final design and
  approval are durable; no runtime code changed yet. Next: failing source and
  cell-background tests, shared output contract, then bounded parallel work.

- Resumed from checksum-verified continuation; `sdlc state` confirmed the approved
  branch/gate. Preserved Pair thread/history and unrelated untracked #48.
- Regressions reproduced native Oxford flattened structure and unfilled terminal
  cells. Implemented shared unpadded output/row-paint contract; bounded parser,
  painter and streaming work delegated after interfaces settled (ARCH-DRY/PURE).
- Native parser, screen selection/resize, section ownership, practice metadata and
  streaming focused tests pass. Integration caught and fixed producer SGR loss,
  mutable exclusion aliases, width-zero stream override and key-decoration ownership.
- Strict actual native rendir conformance passes for es/en, dark/light, widths32/80.
  Actual ANSI/text captures and rendered HTML are under `/tmp/define66-actual*`.
  Source-boundary and mixed-row ownership mutations were killed; bounded parser
  and decoder fuzz passed. Full verification and SDLC close review remain.

- Real PTY checks passed: native rendir at 32/80 columns in dark/light/off, plus
  interactive language switches in all three profiles. Inspected actual ANSI-derived
  dark/light screenshot `/tmp/define66-visual.png`; shared it with the operator.
- Mutation verification killed parser boundary removal, mixed-row ownership,
  missing padding/reset, source-padding contamination and layout-background loss.
  Live footer metadata removal initially survived producer-only tests; added a
  DrawOutput/frame/cell regression and confirmed that mutation now fails.
- Full-suite recovery separated unpadded Transcript from PaintedTranscript; nested
  sitting handoff now transfers structured metadata. Cancellation tests wait for a
  completed row, matching the approved pending-row buffering policy. Focused tests,
  race checks, strict native/PTY checks, vet and Linux build pass after fixes.

- Final verification: `go test ./... -count=1` passed (define 131.235s);
  focused `go test -race ./cmd/define/...` passed; `go vet ./...`, Linux
  CGO-disabled build, strict native/PTY 32/80 dark/light/off and diff check passed.
  The amended cancellation readiness test passed three consecutive runs. Full
  history stays unpadded; PaintedTranscript is only terminal handback, and nested
  screens transfer OutputTranscript metadata. Preparing the single close review.

- Close round 1: REWORK, BR-1 finalized-row metadata and BR-2 Core concepts
  traceability. Reproduced BR-1 through real captured SSE on runAsk success and
  cancellation: append sinks passed, live final rows lost paint. Swept both plain
  and structured append paths; shared invalidatePartialPaint preserves newline /
  CRLF termination and clears ownership on additional source (including structured
  writes without metadata). New regressions fail before and pass after the fix.
- BR-2: reconciled entity declarations, paths and consumer names throughout the
  plan table/test matrix; appended the revision. Focused declaration-status tests
  pass; post-commit full verification follows before close round 2.

- 2026-09-15 checkpoint: close round 2 disposed BR1/BR2 but raised BR3
  (`terminal-serialization-preserves-source`): word wrapping leaves overlong words
  intact and the tinted painter clipped their suffixes. Shared physical layout now
  splits residual long rows before paint, retains click/exclusion coordinates,
  and preserves historical clipping. Dictionary ingress and practice ownership
  projection use this geometry; strict native provenance projection is unchanged.
  New headword/body dark/light/off, wide/combining glyph, click/exclusion, and
  practice/screen ingress tests pass. Focused integrated batch passed (1.700s);
  agent's core/screen race passed. Changes remain uncommitted; full postcommit
  suite, strict PTY recheck, next close review, PR and merge are still required.
  User asked about the UI framework: answered existing custom Go/ANSI liveScreen,
  not React-style library; this does not change the approved implementation scope.

- 2026-09-15: post-BR3 full suite passed (define 129.099s), race and strict
  native/PTY passed. Close round 3 accepted BR3 but found BR4 practice click
  targets retaining obsolete word-only geometry. Enumerated structured consumers;
  corrected practice source-action projection and regionWriter fallback together.
  Both production-path regressions failed before and pass after; focused batch
  passed (1.933s). Source actions are cloned before surrounding-line adjustment.

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

### 2026-09-15 — Approved for implementation

Operator approved the corrected preview (“yes, looks great”). The selected design
uses uniform dictionary-section backgrounds and inline bilingual example pairs.
Proceeding through change-code; estimate follows plan-quality acceptance.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only, derived after the plan-quality gate accepted
round 2 (PQ-1 addressed). Calibration is marked stale by `sdlc estimate-source`,
so this is provisional focused ship-time, not an elapsed delivery promise.

Decomposition follows the approved plan: one issue/spec and two visual iteration
rounds; Oxford parsing; structured layout; painter; screen/selection; streaming;
practice integration; regression harness; native/PTY discovery; docs; one close
review. All implementation values below are v2 table values times 0.4, once.
Familiarity is 1.0: these are the existing Go/XML/terminal/play/SSE seams from #65.
The three TUI units separately own pure painting, screen lifecycle/selection,
and streaming state, each including its direct tests; the harness unit only
covers shared cell-oracle/mutation plumbing.

Design derivation: spec 1.0 and each UX round 0.5 are not discounted (the operator
decisions were the work). Resolved technical units use ×0.2: layout 1.5→0.3;
three TUI units 1.5→0.3 each; integration 0.6→0.12; harness 0.3→0.06;
docs 0.15→0.03; review 0.1→0.02. Library check: existing encoding/xml and the
bounded ownership parser halve Oxford parser design 1.5→0.75 before ×0.2→0.15.
No library supplies our output/click/selection ownership contract; existing
geometry helpers are reused but that module's design is not halved.
Implementation bases respectively: spec 0.2; UX 0.2 each; parser/layout 0.8 each;
TUI 1.0 each; integration/harness 0.5 each; native discovery 0.6; docs 0.2;
review 0.5. Design subtotal 3.58, implementation subtotal 3.0; thorough-plan
buffer 15% on design gives 3.58 × 1.15 + 3.0 = 7.117 hours.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=1.0 impl=0.08
item: ux-rename-iteration design=0.5 impl=0.08
item: ux-rename-iteration design=0.5 impl=0.08
item: greenfield-go-module design=0.15 impl=0.32
item: greenfield-go-module design=0.3 impl=0.32
item: tui-screen design=0.3 impl=0.4
item: tui-screen design=0.3 impl=0.4
item: tui-screen design=0.3 impl=0.4
item: cross-cutting-refactor design=0.12 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.03 impl=0.08
item: milestone-review design=0.02 impl=0.2
design-buffer: 0.15
total: 7.117
```

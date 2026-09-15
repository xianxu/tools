---
id: 000065
status: done
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours: 5.526
started: 2026-09-15T13:08:11-07:00
actual_hours: 2.29
---

# define: distinguish response languages with subtle background tint

## Problem

Bilingual output makes the language of each passage harder to recognize at a
glance. The prompt indicator in #62 identifies the active language, but learners
also need to distinguish that language within definitions, practice questions,
and model answers containing multiple languages.

## Spec

**A subtle background tint means “the language I am studying.”** Apply the tint
to text whose language matches the effective `/lang`. Other-language text keeps
the normal background and remains comfortably readable. The meaning is relative
to the selected language, rather than a different color assigned to each language.

Cover dictionary definitions and full reveals, practice prompts/options/gloss
panels, and mixed-language model responses. In an English explanation containing
a Spanish phrase, with `/lang es`, tint only the Spanish phrase. A language switch
must affect subsequent output consistently with #62's prompt indicator.

Preserve existing foreground vocabulary colors and semantic answer markings.
Selection and answer-state styles take precedence over the subtle language tint.
Keep existing section labels; text remains understandable when color is disabled
or unavailable. Copied text contains the original prose, without language markup
or terminal escape sequences.

**Language ownership is explicit.** Reuse known dictionary and practice language
boundaries. Generated mixed-language replies should carry language annotations
that a shared renderer consumes (ARCH-DRY), including inline spans within a
sentence. Do not infer a word's language solely from its spelling: `red`, `son`,
and `once` are ambiguous across English and Spanish. Unknown or invalidly annotated
text remains neutral; model-provided annotations must not become raw terminal
control sequences. Buffering/streaming and validation details belong in the
implementation design.

Related work:

- [#62 — prompt language flag](../history/issues/000062-language-flag-prompt.md): identifies the
  current language; this task identifies its text within responses.
- [#64 — adaptive bilingual interaction](000064-adaptive-bilingual-interaction.md):
  chooses how much of each language to produce; this task presents the language
  portions visibly. Mixed model responses are in scope here, not deferred to #64.

These are related tasks, not established blocking dependencies.

Design questions to resolve before implementation:

- Choose a readable subtle tint for light and dark terminal backgrounds and the
  fallback when terminal color capabilities are limited.
- Define annotation format, streaming boundaries, and safe handling of incomplete
  or malformed model annotations; inventory every response producer/renderer.
- Decide tint coverage over whitespace, wrapped lines, and passages after a
  language switch, without confusing language styling with selection.

## Done when

- With `/lang es`, Spanish passages are subtly tinted and English help remains
  on the normal background across definitions, practice, and model responses.
- The same rule follows other selected languages, including `/lang en`.
- Inline mixed-language answers tint only the annotated matching-language spans;
  uncertain text stays neutral and ambiguous words are not guessed from spelling.
- Vocabulary colors, answer markings, selection/copy, wrapping, cursor position,
  and mouse targets remain correct, including wide Unicode and narrow terminals.
- Color-disabled output remains readable and copied text contains no annotations
  or escape sequences; annotation failures preserve readable neutral prose.
- Tests cover known and generated language boundaries, language changes, style
  precedence, malformed/incomplete annotations, and streaming splits. Model
  annotations have a stateful fake and a live conformance check.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration remains provisional.

The decomposition covers spec/design, shared ownership/style, source metadata,
parser provenance, practice layout, annotation decoding, terminal-control
filtering, answer-stream integration, live semantic discovery, docs and one
boundary review. Existing encoding/xml plus record captures halves source-module
design before the thorough-plan discount; existing terminal/LLM seams are reused.
No library implements our annotation recovery/provenance policy, so those units
retain their full primitive design allowance before the plan discount. Other
implementation-unit design uses ×0.2; the incurred issue/spec allowance is kept.
Implementation values are 40% of the v2 table; familiarity 1.0, design buffer 15%.
Live discovery covers previously unmeasured model annotations and dictionary
class semantics, not routine verification already included in implementation.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.8 impl=0.12
item: smaller-go-module design=0.06 impl=0.2
item: greenfield-go-module design=0.2 impl=0.32
item: cross-cutting-refactor design=0.2 impl=0.2
item: tui-screen design=0.4 impl=0.4
item: greenfield-go-module design=0.3 impl=0.32
item: greenfield-go-module design=0.2 impl=0.24
item: api-integration design=0.2 impl=0.4
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.2
design-buffer: 0.15
total: 5.526
```

## Plan

- [x] Design language ownership and shared styling across all response surfaces,
  choose and preview the tint, and write the durable implementation plan with
  annotation/streaming contracts and verification before implementation.

## Log

### 2026-09-15
- 2026-09-15: closed — BR-1 source provenance corrected across native assembly, wrappers, lookup, Choice/Cloze reveals and dictionary-derived practice glosses. Unknown and foreign-source matrix red on prior HEAD then green; full go test ./... 131.708s; focused race 5.573s; vet, Linux build, strict native ownership and dark/light/off PTY rechecks pass. Original decoder fuzz 42274 executions, live model capture and all five mutation classes passed. PTY bytes/geometry verified; subjective contrast across all fonts/themes not claimed.; review verdict: SHIP

- Captured the user's approved proposal: subtly highlight the background of text
  matching `/lang` so bilingual portions are immediately distinguishable.
- User approved a separate task linked to #62 and #64, with mixed-language model
  answers in scope from the start. Task capture only; implementation has not begun.
- Existing seams: `definitionSection`/`renderDefinitions` own dictionary sections;
  practice forms expose English `HelpLines`, consumed by `writeHelped`. Current
  styles dim English help. A shared explicit language representation should serve
  these consumers and generated replies rather than duplicate styling rules.

- After #62 merged, claimed #65 and entered planning. The durable design and interactive dark/light preview are ready. Fresh review found that the existing escape grammar cannot sanitize full OSC/DCS model controls; the revised plan adds bounded constant-state filtering before annotation decoding, rendering and transcript storage. Re-review approved with no remaining Important gaps. Runtime implementation awaits operator approval; estimate deferred to the gate.

- Operator approved the complete plan; plan-quality and estimate gates passed and
  implementation began on the issue branch. Shared style/CLI, source ownership,
  practice presentations and model decoder/adapter are implemented uncommitted.
  Focused tests and live production model annotations passed. Initial full suite
  found intended prompt-golden/docs drift, plan table formatting and misplaced
  doc comments; those are being corrected before final verification.
- Integration exposed physical wrap newlines retaining background and the real
  dictionary lock hiding its supplementary interface. Both require fixes and
  regressions; native Oxford ownership itself passed. No #64 runtime work.
- Estimate mapping: the three greenfield units cover dictionary source projection,
  annotation decoding and terminal-control filtering. Their implementation costs
  include adversarial/race/fuzz checks; the UI unit includes PTY/selection checks.
  Most dictionary class discovery used existing committed records within its
  source-module allowance; the live discovery allowance principally covers the
  new model semantics. Mutation checks and boundary review remain outstanding.

- Operator clarified they use both light and dark terminal backgrounds depending
  on the session. Both explicit invocation profiles are retained and documented;
  default dark remains the approved design, without automatic theme inference.

- Implementation complete before close review. Full `go test ./... -count=1`
  passed (define 125.250s); focused language/ask/practice/selection race suite
  passed (40.544s), dictionary locking race passed (2.071s); `go vet ./...`,
  Linux CGO-disabled build and diff check passed. Decoder fuzz passed 42,274
  executions. Strict native dictionary and dark/light/off PTY checks passed,
  including `/lang en`, subsequent tinted English output and terminal restoration.
  Strict production-prompt live model conformance passed (5.85s): captured
  Spanish greetings and coffee question belong to Spanish, explanatory prose
  and “good morning” to English. Real bytes are committed as stream-language.sse.
- All planned mutation classes rejected: whole-supplement English or Spanish
  ownership, missing board/footer ownership, foreground-reset losing tint,
  raw model text reaching history, and missing finish on cancelled answers.
  Mutations ran in isolated snapshots; shared code restored/unmodified.
- Dark/light/off ANSI profiles were exercised in a real PTY and captured at
  `/tmp/define65-language-tint-pty/`. Automated byte/geometry evidence and the
  browser color preview do not establish subjective contrast in every terminal
  font/theme; no claim of manual terminal screenshot inspection is made.

- First mandatory close review returned REWORK with BR-1: active-dictionary
  fallback can return prose in a different language from `/lang`. Reproduced
  across ordinary lookup, Choice and Cloze reveals in an isolated committed
  snapshot. Fix source provenance at dictionary assembly/definition sections;
  class sweep also covers dictionary-derived practice option and panel glosses.
  Source ownership is independently verified, with fallback unknown/neutral.

- BR-1 correction verified: source-selection matrix covers missing/native-API
  fallback, absent requested dictionary, mixed/bilingual metadata and known
  monolingual IDs; wrappers preserve verified ownership. Lookup/Choice/Cloze
  source-target matrix passed after failing on the reviewed implementation.
  Practice gloss regressions cover unknown, foreign and matching sources.
  Full Go suite, focused provenance/language/practice race (5.573s), vet, Linux
  build and diff check passed again. Strict native Oxford ownership and all
  dark/light/off PTY profiles passed again with the corrected source assembly.

- Close re-review SHIP: BR-1 addressed, no remaining findings. Reviewer independently
  verified package tests/vet and mutation rejection. Lessons capture source
  provenance, wrapper capabilities and physical-wrap style boundaries.

## Revisions

### 2026-09-15 — Work order and scope confirmed

User directed implementation of #62 first, then #65. Both concern bilingual
presentation. #64 remains separate and larger: it learns the learner's habits
and adapts how the program interacts with them. This ordering does not make
#64 a prerequisite for language styling. Fresh spec review approved this capture.


### 2026-09-15 — Concrete design after #62

#62 merged as PR #47. Claimed #65 and ran start-plan. The proposed
[implementation plan](../plans/000065-define-distinguish-response-languages-with-subtle-background-tint-plan.md) uses
explicit ownership and one background composer, with bounded inline model
annotations decoded before display and transcript storage. Oxford's supplement
contains mixed Spanish/English inside individual glosses; preserve validated
HTML provenance through selected records and parsed fields. Practice must retain
its import-free producer-owned layout, including board footer and full reveals.

Draft color choice is a neutral dark background, plus explicit light/off
invocation options; theme preference was requested asynchronously. No runtime
code changed and no estimate has been set. Plan and visual preview are for
operator review before implementation.

---
id: 000065
status: working
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
started: 2026-09-15T13:08:11-07:00
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

- [#62 — prompt language flag](000062-language-flag-prompt.md): identifies the
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

## Plan

- [ ] Design language ownership and shared styling across all response surfaces,
  choose and preview the tint, and write the durable implementation plan with
  annotation/streaming contracts and verification before implementation.

## Log

### 2026-09-15

- Captured the user's approved proposal: subtly highlight the background of text
  matching `/lang` so bilingual portions are immediately distinguishable.
- User approved a separate task linked to #62 and #64, with mixed-language model
  answers in scope from the start. Task capture only; implementation has not begun.
- Existing seams: `definitionSection`/`renderDefinitions` own dictionary sections;
  practice forms expose English `HelpLines`, consumed by `writeHelped`. Current
  styles dim English help. A shared explicit language representation should serve
  these consumers and generated replies rather than duplicate styling rules.

- After #62 merged, claimed #65 and entered planning. The durable design and interactive dark/light preview are ready. Fresh review found that the existing escape grammar cannot sanitize full OSC/DCS model controls; the revised plan adds bounded constant-state filtering before annotation decoding, rendering and transcript storage. Re-review approved with no remaining Important gaps. Runtime implementation awaits operator approval; estimate deferred to the gate.

## Revisions

### 2026-09-15 — Work order and scope confirmed

User directed implementation of #62 first, then #65. Both concern bilingual
presentation. #64 remains separate and larger: it learns the learner's habits
and adapts how the program interacts with them. This ordering does not make
#64 a prerequisite for language styling. Fresh spec review approved this capture.


### 2026-09-15 — Concrete design after #62

#62 merged as PR #47. Claimed #65 and ran start-plan. The proposed
[implementation plan](../plans/000065-language-response-tint-plan.md) uses
explicit ownership and one background composer, with bounded inline model
annotations decoded before display and transcript storage. Oxford's supplement
contains mixed Spanish/English inside individual glosses; preserve validated
HTML provenance through selected records and parsed fields. Practice must retain
its import-free producer-owned layout, including board footer and full reveals.

Draft color choice is a neutral dark background, plus explicit light/off
invocation options; theme preference was requested asynchronously. No runtime
code changed and no estimate has been set. Plan and visual preview are for
operator review before implementation.

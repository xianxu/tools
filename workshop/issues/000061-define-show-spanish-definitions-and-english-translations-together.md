---
id: 000061
status: working
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours: 6.02
started: 2026-09-14T16:22:35-07:00
---

# define: show Spanish definitions and English translations together

## Problem

Spanish lookups currently select only Larousse. Enabling Oxford Spanish–English in Dictionary.app does not add English explanations in define. The user wants both, Spanish first and English second.

## Spec

For a Spanish lookup, display separate labeled sections in this order: Spanish definitions from Larousse, then English explanations/translations from Oxford Spanish–English. Use installed dictionaries offline, automatically in the existing Spanish language mode; no new flag or LLM dependency. Preserve source wording and do not imply that senses from independent dictionaries align one-to-one.

The bilingual lookup must select the Spanish source entry, including ambiguous spellings such as `red`; English-to-Spanish records must never masquerade as English explanations of Spanish. Missing dictionaries, unavailable direction-selection APIs, missing words, and real lookup failures are distinct. Keep any successful section and explain an unavailable requested section with relevant setup guidance. If neither section succeeds, retain lookup failure semantics.

Use the same section renderer in one-shot/editor full lookups and full dictionary reveals in practice. Keep the Spanish deck identity, pronunciation, candidate generation, compact board glosses, and grading in Spanish. `--raw` continues to return the unmodified primary Spanish entry. English and Italian behavior remains unchanged. `/lang` reports the available sources; changing language rebuilds them together.

## Done when

- Spanish lookup output shows Spanish first, English second, and ambiguous spellings use the Spanish meaning.
- Partial availability is explicit without suppressing a valid section; raw and other languages retain their contracts.
- One-shot, editor, full practice reveal, click/selection geometry, language switching, capture, and audio behavior are verified.
- Native conformance and portable unit/integration tests pass, and setup docs identify both dictionaries.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration is marked stale; values are provisional.

The native bridge uses already-probed DictionaryServices and existing cgo patterns; parsing uses encoding/xml and existing Entry/Render helpers, and persistence reuses writeBytesAtomic. No external library replaces direction selection. Thorough plan discounts implementation-unit design by 0.2 and uses 15% design buffer; issue authoring retains its observed scope. Implementation values are 40% of v2 table values, familiarity 1.0 for this familiar codebase. Units below correspond in order to spec, native adapter, pure selector, persisted setting, command, composed display, consumer integration, docs, and one close review.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.08
item: api-integration design=0.2 impl=0.4
item: greenfield-go-module design=0.1 impl=0.2
item: smaller-go-module design=0.02 impl=0.12
item: smaller-go-module design=0.04 impl=0.16
item: tui-screen design=0.2 impl=0.28
item: cross-cutting-refactor design=0.2 impl=0.4
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0 impl=0.16
item: scope-pivot design=0.2 impl=0.2
item: greenfield-go-module design=0.1 impl=0.2
item: api-integration design=0.2 impl=0.4
item: smaller-go-module design=0.04 impl=0.16
item: tui-screen design=0.2 impl=0.28
item: cross-cutting-refactor design=0.1 impl=0.2
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0 impl=0.16
design-buffer: 0.15
total: 6.02
```

Extension (assisted practice before answering, added after the scope pivot recorded in Revisions): the pivot itself, then in order the pure assistance validation module, the translation task integration over the existing llm client and llmtest fake, the per-deck cache store, the three practice-form presentations, the queue-preparation/Cloze/author-language cross-cutting wiring, docs, and one further close review round. Same v3.1 method and familiarity; design allowances keep the 0.2 discount because the durable plan already fixes the seams.


## Plan

- [x] Complete and review [the durable plan](../plans/000061-bilingual-definitions-plan.md), including a native direction-selection probe.
- [x] Implement dictionary sections, shared rendering, and consumer wiring with regression tests.
- [x] Verify and document the original definition/reveal implementation.
- [x] Implement and verify pre-answer English practice assistance from the revised durable plan; prepare for the close review and merge gates.

## Log

### 2026-09-14

- Created and claimed at design start. User explicitly requires Spanish explanation first, then English.
- ARCH-DRY: keep primary dictionary lookup authoritative for training; share full-definition composition between presentation consumers.
- Live probe found DCSCopyTextDefinition(OxfordSpanish, "red") returns English red→Spanish rojo. Selecting the dictionary ID alone is insufficient; direction must be verified at the record level before implementation.

- Native probe confirmed usable record search and copied XHTML/flat text. Root record IDs distinguish direction; title preference and deduplication are necessary. Read-only independent corpus: `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/tools61-oxford-ch79mbjk`. Probe observed OxfordSpanish v1.1.

- Fresh spec/plan review approved after correcting audio identity: keep existing lookup-key pronunciation and use canonical headwords only for supplemental matching. Added inflection playback/replay regression and lesson. Durable plan awaits user approval before change-code.


### 2026-09-14 — Implementation reconciliation

- Implemented and locally tested persisted default-on toggle, strict Spanish-source Oxford selection, ordered sections, startup/language-switch wiring, lookup capture/audio preservation and Spanish grammar extraction. Choice and Cloze full dictionary reveals obey the toggle after answering; questions/options/compact glosses/grading stay Spanish.
- Focused verification passed: `go test ./cmd/define/... -run 'TestBilingual|TestDefinition|TestParseBilingual|TestSpanish' -count=1`. README/atlas command synchronization, conformance-table and anchor tests previously passed. Full verification and close review remain pending.


### Implementation and verification (2026-09-14)

- Implemented bounded native record selection, default-on per-deck setting, command/session wiring, Spanish-first composition, Choice/Cloze reveals and Spanish grammatical-label parsing. Off/raw skip the supplemental source; English prose remains selectable without Spanish deck actions. Source availability is explicit in /lang.
- `go test ./...` passed (define 111.545s). Focused race tests, `go vet ./...`, Linux build and strict native + release-stamp conformance passed. Native warm 20-lookup queue measured 20.84ms against 2s budget.
- Fuzz: rendering 61,742 executions; command 32,316; native selector 16,387; setting parser 313,690. Independent fixtures include ambiguous red and canonical inflections.
- Additional integration tests cover partial dictionary availability, malformed/native errors, one capture, typed Spanish audio/replay, copy/action coordinates, explicit idempotent settings, help without effects and real editor nested-practice state propagation.
- Estimate cross-cutting unit includes consumer regression tests and three presentation paths; native probe discovery was performed before the estimate and informs the reduced design allowance.

### 2026-09-15 — Plan gate passed for assisted practice

- Merged main (`c346955`), bringing #54's background preparation; resolved plan/lessons conflicts by keeping both revision trails.
- Plan gate: round 3 raised PQ-4..PQ-8 (offline invariant, per-surface tests, envelope basis, live-check cadence, #54 interaction); round 4 disposed all as addressed. Estimate extended to 6.02 h.
- Estimate-quality returned INFO, not a block. It judged the derivation genuine and additive, and flagged: one first-half `cross-cutting-refactor` impl above its scaled band; the extension's cross-cutting and TUI units likely light; no items for the toggle/parser scope events, the BR-1 rework cycle, or live-proxy discovery. Recorded rather than re-priced mid-flight so the close-time calibration sees the original derivation.
- Filed #64 for the generalisation the user asked for: per-deck interaction stages (English with sprinkled Spanish → Spanish only), inferred from how the learner asks or set by natural-language instruction.

### 2026-09-15 — Assisted practice verified

- Implemented exact displayed-text English help for Choice, Cloze and Board before answering, shared bounded model preparation with the foreground spinner, current-answer validation, and the atomic per-deck translation cache. Off and English decks never resolve the model; warm caches construct no client. Authored sentences explicitly use the deck language. README and atlas describe the new preparation boundary.
- Fresh verification: `go test ./...` passed (define 124.356s); focused bilingual/help/cache/author race tests passed; `go vet ./...`, Linux non-cgo build and `git diff --check` passed. New `TestBilingualPreparationScopesTheInterrupt` passes and fails with an overlay moving interrupt scoping after queue preparation (ARCH-ORDER), proving cancellation reaches the waiting sitting and restores the editor afterward.
- Strict native direction/limits/factory/private-symbol/source-selection/release-stamp checks passed; 20 warm native lookups took 20.67275ms against the 2s budget. The prior inaccessible-dictionary failure did not reproduce under unrestricted execution; historical Seatbelt restrictions and the existing conformance documentation support sandbox asset visibility as its cause.
- Live `TestPracticeHelpAgainstTheLiveService` passed in 4.38s, preserving the cloze blank and Spanish red→net/mesh meaning. Cache fuzz passed 714,373 executions in 10s. Recovered prior-session help-validator fuzz evidence (89,825 executions) and all 14 caught overlay mutations; these are historical evidence, distinct from this session's fresh checks.
- Close review and publication remain the next gates; no final review or merge is claimed by the checked implementation row.

## Revisions

### 2026-09-14 — User-controlled bilingual display

The user requests `/bilingual` to toggle the two behaviors: on means selected language first then English; off means selected language only. This supersedes unconditional bilingual display in the original Spec. The durable plan contains the command, persistence, startup/language-switch wiring and tests. Add explicit `/bilingual on|off`, save per deck like `/lang`, and report the new state. Off must perform no supplemental lookup and show no missing-English warning. New-deck default awaits the user preference; absent a reply, preserve current behavior with off. Plan remains awaiting approval.

### 2026-09-14 — Default confirmed

User chose **default on**. Missing or malformed settings resolve to on, including existing decks without this setting. Persisted off wins. This supersedes the previous fallback assumption.

- Fresh review approved the toggle/default-on revisions. Recorded callback and bool-resolution details so no-capture sessions can toggle and saved off cannot become default on.

### 2026-09-14 — Spanish practice audit

User asked about Spanish /play question construction and bilingual behavior. Fresh probe confirmed grammar-only correct answer for bonito and contaminated Spanish gloss prefixes. Extend implementation to repair common Spanish parsing and verify actual semantic targets across pool/board/harvest consumers. English pre-answer help preference pending; continue approved lookup/toggle/native work and the necessary extraction fix.

### 2026-09-14 — Practice scope and progress reconciled

Choice and Cloze full post-answer dictionary reveals are implemented and tested.
The optional question about English help before answering remains unanswered;
continue the approved post-answer-only baseline. Spanish prompts, options,
compact glosses and grading keep their primary-language identity. The plan now
names the actual Choice/Cloze, inflection-audio and Spanish-parser tests. Moved
Estimate before Plan so this Revisions section remains last, preserving all prior
revision entries. Progress checkboxes do not claim final verification or closure.

### 2026-09-14 — Close boundary prepared

Reworded the final checklist row to describe completed preparation. Close review and merge are the following SDLC gates, not work claimed complete before entering them. Full and committed-window guards passed.

### 2026-09-14 — Beginner assistance before answering

User clarified that English is necessary while answering, not merely afterward.
Bilingual on controls Choice option definitions, a blank-preserving Cloze context
translation, Board gloss help and full reveals. Off is Spanish-only throughout.
The revised durable plan proposes cached exact-text translations prepared through
CLIProxyAPI before practice, with the existing spinner and explicit outage behavior.
This supersedes prior post-answer-only scope; do not close/merge until implemented.
BR-1 metadata diagnosis has been fixed with an observed failing/passing factory
regression; BR-2 concept classifications are corrected. Updated practice plan
awaits approval before additional implementation.

### 2026-09-15 — Verification handoff reconciled

The assisted-practice implementation and verification are complete. The final plan row now records readiness for the separate SDLC close/merge gates rather than claiming those gates already ran. Existing dated design decisions remain as history.

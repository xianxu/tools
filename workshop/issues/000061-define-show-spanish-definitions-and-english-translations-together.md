---
id: 000061
status: working
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
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

## Plan

- [ ] Complete and review [the durable plan](../plans/000061-bilingual-definitions-plan.md), including a native direction-selection probe.
- [ ] Implement dictionary sections, shared rendering, and consumer wiring with regression tests.
- [ ] Verify, document, close review, and merge.

## Log

### 2026-09-14

- Created and claimed at design start. User explicitly requires Spanish explanation first, then English.
- ARCH-DRY: keep primary dictionary lookup authoritative for training; share full-definition composition between presentation consumers.
- Live probe found DCSCopyTextDefinition(OxfordSpanish, "red") returns English red→Spanish rojo. Selecting the dictionary ID alone is insufficient; direction must be verified at the record level before implementation.

- Native probe confirmed usable record search and copied XHTML/flat text. Root record IDs distinguish direction; title preference and deduplication are necessary. Read-only independent corpus: `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/tools61-oxford-ch79mbjk`. Probe observed OxfordSpanish v1.1.

- Fresh spec/plan review approved after correcting audio identity: keep existing lookup-key pronunciation and use canonical headwords only for supplemental matching. Added inflection playback/replay regression and lesson. Durable plan awaits user approval before change-code.

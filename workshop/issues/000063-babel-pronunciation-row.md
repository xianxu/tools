---
id: 000063
status: open
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
---

# define: multilingual babel pronunciation row

## Problem

The learner wants to compare a word across languages and hear each language's pronunciation without changing the active dictionary language.

## Spec

Add `/babel` to display a compact row of equivalent words in supported languages,
with a language flag before each word and a pronunciation link on each word.
Use `/babel` as the canonical spelling; `/bable` in the request is treated as a typo.

- `/babel en,es,zh` configures the ordered language list and remembers it in the
  current deck alongside language preferences. Keep this separate from the
  deck's selected dictionary language and `/bilingual` setting.
- Subsequent `/babel` uses the saved list. With no saved list, the requested
  default is all supported Babel languages; define this capability explicitly
  rather than equating accepted two-letter tags with supported translation/audio.
- Use the current lookup word and its source language as the proposed context.
  Specify the no-current-word behavior during design.
- Display translated equivalents, not repeated source spelling, in configured
  order: `[flag-en] word [flag-es] word [flag-zh] word`.
- Each word's pronunciation action uses that displayed word and its own target
  language/locale, including Chinese in the requested example. Missing translation
  or audio support must be explicit; never play another language's pronunciation.
- Reuse the flag mapping and terminal-width policy from #62 where available.
  Keep the row on one line when it fits; define narrow-terminal behavior without
  breaking pronunciation hit targets or mouse selection.

Design must settle translation/sense selection, supported language discovery,
regional voice/flag choices, cache and model behavior, and whether configuration
also immediately displays the current word. Chinese is desired scope, not a claim
that the current dictionary/pronunciation adapters already support it.

## Done when

- `/babel en,es,zh` persists the ordered Babel preference in this deck without changing its selected dictionary language.
- `/babel` restores that preference after restart and shows flag/translated-word pairs for the current word.
- Clicking each word requests pronunciation in that word's language and preserves the original lookup context.
- Invalid/duplicate tags, unsupported translation/audio, no current word, and narrow terminals have defined behavior and regression coverage.
- Help documents configuration, the default language set, and pronunciation behavior.

## Plan

- [ ] Design Babel context, language capabilities, translation and pronunciation behavior; implement deck persistence and the linked row with tests and docs.

## Log

### 2026-09-14

- Captured the user's request as a separate future feature. Example configuration:
  `/babel en,es,zh`; display a flag and translated word per language, with every
  word linked for pronunciation. Implementation has not started.

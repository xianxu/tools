---
id: 000023
status: open
deps: []
github_issue:
created: 2026-08-27
updated: 2026-08-27
estimate_hours:
---

# deck grouped by language, one language per --play session

## Problem

The deck has one namespace and `--play` reviews all of it. A learner working in
two languages gets `madrugar` and `sycophantic` in the same sitting, which is not
how anyone studies — and `#5`'s schedule interleaves them by due-date, so the
mixing is not even incidental.

Operator, after the first real `--play` session:

> words should be grouped in language and I think each invocation of
> `define --play` should just do one language.

This is `#18 M2`'s first bullet — *"a language dimension on the deck. One
directory per language works today and may be enough; decide deliberately rather
than by accident"* — pulled out because `#18 M2` is blocked on `#10`/`#12` for its
agreement-safe-distractor half, and this half is not.

## Spec

**`words/<lang>/`, one directory per language.** Operator's call, over a `lang:`
field on the word. Stronger separation: a Spanish session cannot see an English
word even by accident, and "what am I learning in Spanish" is `ls`. The cost is a
migration for existing decks and every store path learning about languages, and
that cost is accepted rather than discovered.

**`--play -lang es` reviews one language.** Default is English.

### Detection: MEASURED, and the measurement changes the design

The operator asked for the language to be inferred from the dictionary rather
than declared. Measured on this machine before planning, because the answer
turned out to constrain the whole feature:

**Inference works for Spanish-only words.** They come back with no IPA and with
gendered part-of-speech labels NOAD never uses:

```
madrugar      A intransitive verb to get up early ...
sobremesa     feminine noun 1 (período) ...
empalagoso    adjective (empalagosa) ‹tarta, licor› sickly ...
```

against an English entry's `syc·o·phan·tic | ˌsikəˈfan(t)ik | adjective`.

**Inference CANNOT work for words that also exist in English, and that is an API
limit rather than a heuristic one.** `dict_darwin.go` already records why:
`DCSCopyTextDefinition` is passed a NULL dictionary ref — "search every ACTIVE
dictionary" — and *"the SDK exports no public constructor for a
DCSDictionaryRef, so there is no way to select one"*. It returns ONE entry, and
for a shared spelling that entry is English. The Spanish entry is never returned,
so no amount of text analysis can recover it. Measured:

| word | Spanish meaning | what the dictionary returns |
|---|---|---|
| `mesa` | table | `me·sa \| ˈmāsə \| noun an isolated flat-topped hill` |
| `bonito` | pretty | `bo·ni·to \| bəˈnēdō \| noun a smaller relative of the tunas` |
| `pie` | foot | `pie \| pī \| noun a baked dish containing fruit` |
| `once` | eleven | `once \| wən(t)s \| adverb on one occasion` |
| `real` | royal | `re·al \| rē(ə)l \| adjective actually existing` |
| `red` | net | `red \| red \| adjective of a colour at the end of the spectrum` |

Every one is a word a Spanish learner actually wants, and every one is invisible
to inference.

**So: infer by default, `-lang` to override.** Not a hedge — the flag is
structurally required, because for a shared spelling there is nothing to infer
from. Inference removes the flag from the common case (`madrugar`, `sobremesa`,
`hablar`); the flag exists for the case inference provably cannot see.

`-lang es` on a LOOKUP also has to change what is fetched, not just where it is
filed: `#18 M1` measured that `madrugar_es_es_1.mp3` is a 200 while the `_en_us_`
form this tool asks for is a 404. That is `#18 M1`'s work and this issue should
not duplicate it — but the flag is shared, so they want designing together.

## Done when

- [ ] `words/<lang>/`, with existing decks migrated rather than orphaned.
- [ ] `--play -lang es` reviews Spanish only; the default reviews English only.
- [ ] A Spanish-only word looked up with no flag is filed as Spanish, on evidence
      from the returned entry rather than a guess.
- [ ] A word shared with English is filed as English unless `-lang` says
      otherwise, and that limit is documented where a user meets it rather than
      only here.
- [ ] `--forget` and `d`-in-`--play` remove from the right language's deck.
- [ ] The schedule and the event log are unchanged: language is a deck dimension,
      not an event one. A review event names a word; which deck it came from is
      the deck's business.

## Plan

- [ ] Design via `sdlc start-plan` before implementing. Coordinate with `#18 M1`,
      which owns the audio half of the same `-lang` flag.

## Log

### 2026-08-27

Filed from the operator's request after the first real `--play` session. The
detection measurement above was taken before planning, in the same spirit as
`#9`'s feed measurement and `#18`'s audio measurement: the API limit it found is
what turns "infer it" from a design into "infer it, with an override that is
required rather than convenient".

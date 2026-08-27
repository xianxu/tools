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

**`define` operates in ONE language at a time, and `/lang es` switches it.**
Operator's design, and it is better than the one this issue was filed with —
see Revisions. A mode, not a per-lookup flag.

- **`words/<lang>/`, one directory per language.** A Spanish session cannot see
  an English word even by accident, and "what am I learning in Spanish" is `ls`.
- **`/lang` reports the current language; `/lang es` switches it.** In the
  command namespace `#15` built, beside `/help`, `/history` and `/sound`.
- **The setting PERSISTS in the vocab directory**, unlike `/sound`, which is
  explicitly "for the rest of this session". Language cannot be session-scoped:
  a one-shot `define madrugar` has no session to inherit from, and re-declaring
  the language every time is exactly the friction this design removes. It is a
  property of the directory, which is already the unit everything else here
  scopes to.
- **Everything inherits the mode**: lookups file into that language's deck,
  `--play` reviews that language, and `#18 M1`'s audio asks for that language's
  recording — `madrugar_es_es_1.mp3` is a 200 where the `_en_us_` form this tool
  currently requests is a 404.
- **`-lang es` as a flag** for one-shot use without switching the mode, and
  because scripts should not have to mutate state to ask a question.

### The dictionary CAN be selected — measured, and it changes what is possible

This issue was first specified around a limit that does not exist. The claim —
carried in `dict_darwin.go`'s own comment and repeated into this issue — was that
`DCSCopyTextDefinition` must be passed NULL because the SDK exports no way to
build a `DCSDictionaryRef`. The operator pushed back; measurement says the claim
was wrong.

**True of the public header.** `DictionaryServices.h` declares exactly two
functions and documents the dictionary parameter as *"not supported for Leopard.
You should always pass NULL."*

**False of the framework.** `dlsym` resolves all of these:

```
DCSCopyAvailableDictionaries   DCSDictionaryGetName      DCSCopyDefinitionMarkup
DCSGetActiveDictionaries       DCSDictionaryGetIdentifier DCSDictionaryCreate
DCSDictionaryGetLanguages      DCSDictionaryGetShortName  DCSCopyRecordsForSearchString
```

87 dictionaries are available on this machine, including *Larousse Editorial
Diccionario General de la Lengua Española* and *Oxford Spanish Dictionary*. The
refs they return are accepted by `DCSCopyTextDefinition`. Measured, against the
six words this issue previously listed as unreachable:

| word | NULL (what the tool does today) | Spanish dictionary, selected |
|---|---|---|
| `mesa` | *an isolated flat-topped hill* | *nombre femenino — Mueble formado por un tablero horizontal* |
| `bonito` | *a smaller relative of the tunas* | *adjetivo (femenino bonita) — Que tiene belleza o atractivo* |
| `once` | *on one occasion* | *numeral cardinal — está 11 veces* |
| `real` | *actually existing as a thing* | *adjetivo — Que tiene existencia verdadera* |
| `madrugar` | *to get up early* (bilingual gloss) | *verbo intransitivo — Levantarse muy temprano, especialmente al amanecer* |
| `sycophantic` | the English entry | *(no entry)* — correctly not a Spanish word |

**Three things follow, and they make this issue bigger and better.**

1. **A language mode can be fully correct, not merely correct-at-filing.** In
   Spanish mode `mesa` is filed as Spanish AND defined as Spanish. The caveat
   this issue previously accepted — "filing it correctly is not the same as
   defining it correctly" — is gone.
2. **Monolingual beats bilingual for learning.** `madrugar` through NULL gives
   "to get up early"; through Larousse it gives a Spanish definition with a usage
   example. Reading the target language is the point of the exercise, and the
   better entry was there the whole time.
3. **"Not a word in this language" becomes answerable.** `sycophantic` in Spanish
   mode returns no entry, which is correct and which the tool cannot currently
   say about anything.

### Selecting the RIGHT dictionary is principled, not a name match

A second measurement, because the first probe matched dictionaries by name
substring and that is unsound: `DCSCopyAvailableDictionaries` returns a **CFSet**,
whose iteration order is unspecified, so `"Espa"` could match Larousse on one run
and Oxford Spanish on the next. The API offers a real key and real metadata:

```
New Oxford American Dictionary        id com.apple.dictionary.NOAD
  index=en_US  description=en_US                      -> monolingual English
Larousse Diccionario General          id com.apple.dictionary.es.DGLEV
  index=es     description=es                         -> monolingual SPANISH
Gran Diccionario Oxford               id com.apple.dictionary.OxfordSpanish
  index=es     description=es
  index=en     description=es                         -> bilingual
```

`DCSDictionaryGetIdentifier` gives a stable reverse-DNS id, and
`DCSDictionaryGetLanguages` gives an array of dictionaries keyed
`DCSDictionaryIndexLanguage` (what the headwords are) and
`DCSDictionaryDescriptionLanguage` (what the definitions are).

**So the selection rule writes itself, and is testable:** for language L, prefer a
dictionary whose index language is L *and* whose description language is also L —
monolingual, which is what a learner should be reading — and fall back to one
that merely indexes L. Not "the dictionary whose name contains Español".

This also means the rule degrades sensibly on a machine with different
dictionaries installed: no match for L means no entry, which is the honest
answer, rather than silently answering from English.

**The cost, recorded rather than discovered.** These symbols are private and
undocumented: they can change or disappear on an OS update, and nothing in the
SDK promises otherwise. So the seam must `dlsym` them at run time and FALL BACK
to today's NULL behaviour when any is missing — which degrades to exactly what
ships now, rather than to a crash. That fallback is a Done-when row, not a nicety,
and it wants a conformance check like `#9`'s: an on-demand test that says loudly
when the private surface has moved.

## Done when

- [ ] `words/<lang>/`, with existing decks migrated rather than orphaned.
- [ ] `--play -lang es` reviews Spanish only; the default reviews English only.
- [ ] `/lang` reports the current language; `/lang es` switches it; the setting
      survives the session ending.
- [ ] A one-shot `define madrugar` uses the persisted language, with no session
      to inherit from.
- [ ] A word shared with English is filed AND DEFINED in the current language —
      `mesa` in Spanish mode returns the Spanish entry, not the flat-topped hill.
- [ ] A word absent from the current language reports no entry rather than
      silently answering from another language's dictionary.
- [ ] The private DictionaryServices symbols are resolved at run time and the
      seam FALLS BACK to today's NULL behaviour if any is missing — degrading to
      what ships now rather than crashing.
- [ ] A live conformance check says loudly when the private surface moves, on
      demand like `#9`'s feed check rather than in merge-check.
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

## Revisions

### 2026-08-27 — a declared mode, not inference

**Reason.** Operator, on being shown the measurement above:

> ok, I guess there are same word different meaning in en/es. let `define`
> operate in a single language. add a `/lang en` in the define TUI to switch
> language.

**Delta.** Inference is dropped entirely. `define` has a language MODE, persisted
in the vocab directory, switched with `/lang`.

**Why this is better rather than merely different.** The measurement found a hard
API limit: for `mesa`, `bonito`, `pie`, `once`, `real`, `red` the dictionary
returns the English entry and never reveals a Spanish one exists. Infer-plus-
override would have handled those by being wrong and waiting for the learner to
notice. A declared mode removes the question — in Spanish mode `mesa` is Spanish
because the learner said so. The design that needs no heuristic beats the design
whose heuristic provably cannot see half its cases.

It also matches how the tool is actually used: one deck directory, one sitting,
one language. `/sound` is the precedent for a `/`-command that changes how a
session behaves — but language differs from it in one way that matters, and the
Spec now says so: `/sound` is explicitly session-scoped, while language must
persist, because a one-shot lookup has no session to inherit from.

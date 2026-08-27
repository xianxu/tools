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

### Detection is NOT needed, and that is what makes this design better

This issue was first specified as "infer the language from the dictionary, with a
flag to override". Measured on this machine before planning, and the measurement
killed the inference half:

**Inference works for Spanish-only words** — no IPA, gendered part-of-speech
labels NOAD never uses (`sobremesa feminine noun`, `madrugar A intransitive
verb`) against an English entry's `syc·o·phan·tic | ˌsikəˈfan(t)ik |`.

**Inference CANNOT work for a word that also exists in English**, and it is an
API limit rather than a heuristic one. `dict_darwin.go` already records why:
`DCSCopyTextDefinition` takes a NULL dictionary ref — "search every ACTIVE
dictionary" — and *"the SDK exports no public constructor for a
DCSDictionaryRef, so there is no way to select one"*. It returns ONE entry, and
for a shared spelling that entry is English. The Spanish entry is never returned,
so no text analysis can recover it:

| word | Spanish meaning | what the dictionary returns |
|---|---|---|
| `mesa` | table | `me·sa \| ˈmāsə \| noun an isolated flat-topped hill` |
| `bonito` | pretty | `bo·ni·to \| bəˈnēdō \| noun a smaller relative of the tunas` |
| `pie` | foot | `pie \| pī \| noun a baked dish containing fruit` |
| `once` | eleven | `once \| wən(t)s \| adverb on one occasion` |
| `real` | royal | `re·al \| rē(ə)l \| adjective actually existing` |
| `red` | net | `red \| red \| adjective of a colour at the end of the spectrum` |

**A declared mode makes every row of that table a non-problem.** In Spanish mode
`mesa` is a Spanish word because the learner said so, and there is nothing to
infer, override or get wrong. The measurement is kept here because it is the
reason inference was dropped rather than deferred — and because if anyone later
proposes "we could detect it automatically", this is the answer.

**What the mode cannot fix, and should say so:** the DEFINITION returned for
`mesa` in Spanish mode is still the English one, because the API cannot be asked
for the Spanish entry. Filing it correctly is not the same as defining it
correctly. That is a real limit of `DCSCopyTextDefinition` and it belongs in the
docs where a user meets it, not only here.

## Done when

- [ ] `words/<lang>/`, with existing decks migrated rather than orphaned.
- [ ] `--play -lang es` reviews Spanish only; the default reviews English only.
- [ ] `/lang` reports the current language; `/lang es` switches it; the setting
      survives the session ending.
- [ ] A one-shot `define madrugar` uses the persisted language, with no session
      to inherit from.
- [ ] A word shared with English is filed under the CURRENT language, with no
      inference involved — `mesa` in Spanish mode is Spanish.
- [ ] The docs say plainly that a shared spelling still returns the ENGLISH
      definition, because the dictionary API cannot be asked for another
      language's entry.
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

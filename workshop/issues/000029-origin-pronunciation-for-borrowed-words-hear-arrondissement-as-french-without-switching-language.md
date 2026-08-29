---
id: 000029
status: open
deps: []
github_issue:
created: 2026-08-28
updated: 2026-08-28
estimate_hours:
---

# origin pronunciation for borrowed words: hear arrondissement as French, without switching language

## Problem

`define arrondissement` gives the English entry and the English recording. The
learner wants to know how a French speaker says it.

Operator, filing this:

> while you are on spanish, can you also handle french and italian
> pronunciation? in the context I want to know authentic sound in the source
> language, for words like arrondissement

and, on the nuance that makes this its own issue:

> I'm not learning French/Italian in this case, but just want to know origin
> pronunciation of borrowed words.

**This is NOT `#23`'s language mode, and the distinction is the whole issue.**
`/lang fr` switches the deck, the dictionary and the review session to French —
it is for *learning French*. Here the session stays English, the word stays in
the English deck, and only the RECORDING comes from the source language. A mode
switch is the wrong shape: nobody wants to change what they are studying in order
to hear one word said properly.

## Spec

Not yet designed — two shapes are plausible and they are different sizes. What
follows is measured, so the design starts from facts rather than guesses.

### What already works, measured 2026-08-28

**`define -lang fr arrondissement` already plays the authentic French
recording.** No code change. `#23 M1`'s D2 rule — locale is the language code,
except English which is `us` — builds
`.../ar/arrondissement_fr_fr_1.mp3`, which is a **200**.

So the audio half of this issue is nearly free. What `-lang fr` does WRONG is
everything else: it would file the word into `words/fr/` and look it up in a
French dictionary. That is the mode, and the mode is what this issue does not
want.

**NOAD already carries the source pronunciation in NOTATION:**

```
arrondissement  ar·ron·disse·ment
/əˈrändəsmənt, eˌrändēsˈmäN, əˌrändēsˈmäN/
```

Three variants — the first anglicised, the last two French-ish with the nasal
`äN`. So the notation need is already met on this word; only the audio follows
the session's language rather than the word's origin.

### CDN coverage — the asymmetry that shapes scope

| language | probe | result |
|---|---|---|
| French `_fr_fr_` | `arrondissement`, `bonjour`, `croissant`, `fromage`, `chef` | **200** |
| French `_fr_fr_` | `déjeuner`, `rendez-vous` | 404, in every accent/hyphen form tried |
| Italian `_it_it_` | `ciao`, `grazie`, `pizza`, `caffè`, `parlare`, `mangiare` | **404**, 0 of 9 across 3 locale forms |
| German `_de_de_` | `schadenfreude` | **200** |

**French coverage is PARTIAL, not general** — that matters for the design,
because a miss must degrade to the English recording rather than to silence. The
`déjeuner` 404s survived an accent hypothesis (`d%C3%A9jeuner`) and a hyphen
hypothesis, so the cause is unknown and should not be guessed at.

**Italian audio does not exist in this CDN generation.** Nine probes, zero hits.
So for Italian this issue's actual ask — the authentic sound — cannot be
delivered from this source at all. Italian DEFINITIONS are a different matter
(see below). Recording it here so the limitation is inherited rather than
rediscovered.

### Dictionaries, measured — a separate and much cheaper win

All three are installed on this machine and **strictly monolingual**, so
`chooseDictionary` accepts them as-is:

| identifier | pairs |
|---|---|
| `com.apple.dictionary.fr.Multi` | `fr>fr` (monolingual despite the name) |
| `com.apple.dictionary.it.Devoto-Oli` | `it>it` |
| `com.apple.dictionary.de.DDDSI` | `de>de` |

Adding them to `curated` in `cmd/define/dictselect.go` is **one line each** and
makes `/lang fr|it|de` work as a full mode. That is worth doing and is NOT this
issue: it serves someone learning those languages, which the operator explicitly
is not. Split out so the cheap win is not blocked on this design.

### The two shapes to choose between

1. **A flag** — `define -pron fr arrondissement`. Explicit, small, composes with
   the English session: the deck, the dictionary and the highlight set all stay
   English, and only `voice.Lang` changes for this lookup. Sits naturally beside
   `-locale`, and `#27` is the neighbour that owns pronunciation as a parameter.
2. **Automatic** — the entry says `ORIGIN French, from arrondir`, and the CDN
   either serves `_fr_fr_` or does not. Trying and falling back is cheap.
   **But inferring language is what `#23` deliberately rejected** in favour of a
   declared mode, and that rejection was argued from measurement. Reversing it
   for a narrower case may be right — a loanword's origin is stated in the entry
   rather than guessed from the word — but it must be argued, not assumed.

A third possibility worth pricing: the notation is already there, so "play the
source recording" may matter less than **saying which pronunciation is which**.
`/əˈrändəsmənt, eˌrändēsˈmäN/` does not tell the reader that the first is
English and the second French.

## Done when

- [ ] A borrowed word can be heard in its source language **without changing the
      session's language** — the deck, dictionary and highlight set stay English.
- [ ] A source-language recording that does not exist degrades to the English
      one, audibly the same as any other miss — French coverage is partial and
      `déjeuner` is the case to test.
- [ ] Italian's absence from the CDN is reported honestly rather than as silence.
- [ ] Whether the language is declared or inferred is a DECISION with its reason
      recorded, reconciled against `#23`'s rejection of inference.

## Plan

- [ ] Design via `sdlc start-plan`. Coordinate with `#27`, which owns
      pronunciation as a parameter rather than a literal.

## Log

### 2026-08-28

Filed from the operator's request. Every measurement above was taken before
filing — the CDN probes, the dictionary language pairs, and the observation that
`-lang fr` already fetches the French audio — so the design starts from what is
true rather than from what seems likely. Two of my own guesses were falsified
while measuring: that French coverage was general (it is partial), and that the
`déjeuner` 404 was an accent-encoding problem (it is not).

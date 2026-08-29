---
id: 000034
status: open
deps: [tools#31]
github_issue:
created: 2026-08-29
updated: 2026-08-29
estimate_hours:
---

# French and German need parser work before curating: the entry collapses into one blob

## Problem

Split from `#31`, which narrowed to Italian once the three books were measured
rather than assumed. `#29`'s Spec had called all three "one line each"; that is
true of the `curated` map and false of the reading experience.

Measured 2026-08-29 by querying each dictionary directly through
`selectedDictionary{ids: …}`, five common words each:

| dictionary | senses the parser finds | longest single undifferentiated blob |
|---|---|---|
| `NOAD` (English, baseline) | 153 | 144 runes |
| `it.Devoto-Oli` (shipped in `#31`) | 245 | 645 runes |
| **`fr.Multi`** | **6** | **2,811 runes** |
| **`de.DDDSI`** | **5** | **4,404 runes** |

`Haus` puts its grammar table, its `TYPISCHE VERBINDUNGEN`, its whole `SYNONYME`
list and its `HERKUNFT` etymology into ONE example string. That is not a
definition a person reads; it is the raw entry with quote marks around it.

## Spec

Not designed. What follows is measured, so the design starts from facts.

### Why they collapse, and it is the same cause twice

`ParseEntry` is NOAD-shaped in two closed vocabularies, both in `parse.go`:

- **`posWords`** — `noun`, `verb`, `adjective`, … The French book writes
  `nom masculin`, the Italian `s.f.` / `inter.` / `agg.`, the German
  `Substantiv, feminin`. No match means no block boundary, so everything after
  the headword is one run.
- **`sectionWords`** — `ORIGIN`, `DERIVATIVES`, `PHRASES`, `USAGE`. The Italian
  writes `ETIMOLOGIA`, `DATA`, `ACCRESCITIVO`; the German `HERKUNFT`,
  `SYNONYME`, `TYPISCHE VERBINDUNGEN`. No match means the trailing apparatus is
  swallowed into the last sense.

Italian survives despite this because Devoto-Oli marks senses `1`, `2` and
sub-senses `•` — shapes `parseSenses` already knows. French and German do not
number in a way the parser reaches.

**So the question this issue must answer first is architectural, not lexical:**
does `ParseEntry` become dictionary-aware (a vocabulary per book, injected), or
does each book get a parser behind the existing `Dictionary` seam? Today
`ParseEntry(raw string)` takes only a string — deliberately, and that purity is
what makes the whole parser suite runnable with no CoreServices. Widening its
signature touches every English test that pins it, so it is a decision to argue
rather than a refactor to start.

### German's pronunciation field is LOSSY, not fabricated

Worth stating carefully, because a first reading of it is wrong. Duden marks the
stressed vowel: short vowels get a dot below, long vowels an underline.

```
Wasser   -> Wạsser            (U+1EA1, dot below — survives)
Hund     -> Hụnd              (survives)
Katze    -> ˈkatsə, Kạtze     (real IPA alongside)
Haus     -> Haus              (long vowel: underline, does NOT survive)
Freude   -> Freude            (same)
Schadenfreude -> Schadenfreude
```

9 of 15 sampled entries render the bare headword. `Render` then prints
`/Haus/` — visually a pronunciation, informationally nothing, and
indistinguishable from a real transcription to the reader.

**And it cannot be filtered by shape.** Two candidate rules both fail against
measurement:

- *"require a phonetic marker"* — English `bank`→`baNGk`, `set`→`set`,
  `man`→`man`, `thing`→`THiNG` carry none. 4 of 28 sampled English entries.
- *"reject when it equals the headword"* — English `set` has IPA `set`.

The distinguishing fact is the BOOK, not the string, which is the same
conclusion the vocabularies above reach by another road.

### `fr.Multi` is Québécois, not France French

Its examples are Outremont, Plateau-Mont-Royal and Fromagerie Tournevent, and it
cites the GDT. A learner choosing "French" would not expect that. It is a fine
dictionary and it is not the one most people mean, so wherever the book is named
this should be said.

`OxfordFrench` and `OxfordGerman` ARE installed and would be the alternative —
but both are bilingual (`fr>fr` AND `en>fr`), so `monolingualIn` rejects them.
That is the guard working as designed: a bilingual book in a French session
would put English glosses in front of a French learner. Preferring "the other
French dictionary" is therefore not available without changing that rule, and
`#23 M2` argued that rule from measurement.

## Done when

- [ ] `/lang fr` and `/lang de` produce an entry a person reads, not one blob —
      with a measured before/after on the blob table above rather than an
      impression.
- [ ] The dictionary-aware-parser question is a DECISION with its reason
      recorded, reconciled against `ParseEntry`'s current purity.
- [ ] German's lossy pronunciation field is decided rather than shipped: either
      suppressed when it carries no mark, or shown with what it is.
- [ ] Whatever is added does not enter unswept — the live raw-notation ratchet is
      English-only, which `#31` recorded as a standing gap.
- [ ] `fr.Multi` being Québécois is stated where a learner meets it.
- [ ] The English parser suite is unchanged in behaviour, proven by it staying
      green rather than by assertion.

## Plan

- [ ] Claim, then design via `sdlc start-plan`. Blocked on nothing — `#31` ships
      Italian independently — but read `#31`'s `## Revisions` first.

## Log

### 2026-08-29

Split from `#31` after measurement, on the operator's decision. Every number
above was taken before filing; none is inherited.

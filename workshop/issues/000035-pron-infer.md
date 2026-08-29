---
id: 000035
status: open
deps: [tools#29]
github_issue:
created: 2026-08-29
updated: 2026-08-29
estimate_hours:
---

# /pron with no language: infer the origin from ORIGIN, error only when it cannot be determined

## Problem

Operator, after using `#29`:

> improvement number one is just use `/pron` instead of `/pron fr`, basically
> infer origin language whereever possible and only error out on `/pron` if you
> can't determine.

`#29` shipped a DECLARED language, and the entry prints `ORIGIN French` two lines
above the prompt. So the tool displays the answer and then asks you to retype it
as a code — and to know that Japanese is `ja`.

## Spec

### This does NOT reopen `#29`'s D1

`#29` rejected **automatic** origin audio, and that rejection stands untouched.
The measurement behind it: NOAD writes `ORIGIN French` identically for
`arrondissement` and for `police`, and the CDN serves `police_fr_fr`,
`restaurant_fr_fr` and `machine_fr_fr` at **200**. Inferring on every lookup
would replace the English recording for a large class of naturalised words.

`/pron` is opt-in per word. A wrong inference is a thing you asked for, on one
word, and it is REPORTED — which is what makes inference affordable here and not
there. Same argument `#30` records for why clicking dissolves the ambiguity
objection: the user's gesture supplies the intent that automation cannot.

### The closed-table objection is weaker here than for `#23`/`#27`, and that is arguable rather than obvious

`ParseLang` refuses to enumerate languages and `localeFor` refuses to enumerate
locales, both on the same reasoning: **the CDN and the installed dictionaries own
what exists**, so a table here restates someone else's fact and goes stale the
day they add a row.

A `French → fr` map is a different kind of thing. It reads **NOAD's editorial
prose** — the set of language names a dictionary writes in its etymologies — which
is stable, small, and ours to read. It is not a claim about what the CDN serves;
the CDN still decides that, and an inferred language that has no recording
degrades exactly as a typed one does.

Stated as the decision it is, because the next reader will meet three refusals of
a table and one acceptance and deserves to know which argument separates them.

### Measured 2026-08-29

**On borrowings — the case this is for**, 25 words (`arrondissement`, `croissant`,
`jalapeno`, `tortilla`, `ciao`, `pizza`, `espresso`, `karaoke`, `schadenfreude`,
`kindergarten`, `tsunami`, `bonjour`, `chef`, `genre`, `nuance`, `debut`,
`ballet`, `piano`, `opera`, `macho`, `siesta`, `patio`, `aficionado`, `burrito`,
`guacamole`):

| outcome | count |
|---|---|
| determinate — exactly one modern language named | **23** |
| ambiguous — two named | 2 (`ballet`, `piano`) |
| no language named | 0 |

**On 220 ordinary English words**, sampled by stride from `/usr/share/dict/words`:

| outcome | count | what `/pron` should do |
|---|---|---|
| no ORIGIN section at all | 171 | error: nothing to infer from |
| ORIGIN names only HISTORICAL stages | 31 | error: `Old English`, `Latin`, `Middle Dutch` are not pronunciation targets |
| determinate | 15 | play it — `garage` really is French, and you asked |
| ambiguous | 3 (`bring`, `casque`, `knout`) | error and list them |

So on ordinary vocabulary the answer is usually "I cannot tell", which is the
correct outcome and the one the operator asked for.

### Historical stages must be excluded, and not because they 404

`Latin`, `Old English`, `Middle English`, `Old French`, `Old Norse`, `Middle
Dutch`, `Sanskrit` and `medieval Latin` appear constantly in etymologies. Several
have ISO codes (`la`, `ang`, `sa`), so "has a code" is not the filter. They are
excluded because **a dead or superseded stage of a language is not something a
speaker says today** — the concept `/pron` offers does not apply. Filtering them
by "the CDN has no recording" would be right by accident and wrong in kind, and
would silently start playing something if the CDN ever added Latin.

The exclusion also has to survive text like *"via Old French from Latin"* and
*"from French, from Latin"*: the first names no modern language, the second names
one.

### The rule, and the two halves that must be decided rather than assumed

1. **First-named modern language wins.** `opera` — *"from Italian, from Latin"* —
   is Italian; `ballet` — *"from French, from Italian balletto"* — is French, the
   route it entered English by.
2. **It says which it chose.** `arrondissement: ORIGIN says French` before it
   plays. A silent inference is one you cannot audit, and this repo's rule is
   that a record has to be true. It also makes `piano`'s contested case visible
   rather than decided behind your back.
3. **Ambiguity errors and lists.** Two modern languages named is "cannot
   determine", per the operator's instruction — not a coin flip.

`/pron fr` keeps working and keeps overriding, so a wrong inference always has a
one-word answer.

## Done when

- [ ] `/pron` with no argument plays the source recording for a word whose ORIGIN
      names exactly one modern language, and SAYS which language it chose.
- [ ] `/pron` errors, naming what it found, when ORIGIN is absent, names only
      historical stages, or names more than one modern language.
- [ ] Historical stages are excluded BY CATEGORY with the reason recorded, not by
      relying on the CDN to 404 them.
- [ ] `/pron fr` still overrides, and is unchanged.
- [ ] The language-name table is a DECISION with its reason recorded, reconciled
      against `ParseLang` and `localeFor` both refusing to enumerate.
- [ ] `#29`'s D1 — no AUTOMATIC origin audio — is demonstrably untouched, pinned
      by a test rather than by assertion.
- [ ] The extraction is pure and table-tested over captured ORIGIN text, not only
      exercised through the live dictionary.

## Plan

- [ ] Claim, then design via `sdlc start-plan`.

## Log

### 2026-08-29

Filed from the operator's request while discussing `#30`. Every number above was
measured before filing. Sequenced BEFORE `#30` deliberately: it makes both of
`#30`'s click targets thin wrappers over gestures that already exist — a headword
click is a replay, and a click on `ORIGIN French` is `/pron` with that language —
so the inference question is settled before the screen work starts rather than
tangled into it.

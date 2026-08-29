---
id: 000035
status: working
deps: [tools#29]
github_issue:
created: 2026-08-29
updated: 2026-08-29
estimate_hours: 1.39
started: 2026-08-29T14:56:14-07:00
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

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.35 impl=0.08
item: smaller-go-module        design=0.02 impl=0.12
item: smaller-go-module        design=0.03 impl=0.14
item: smaller-go-module        design=0.00 impl=0.08
item: cross-cutting-refactor   design=0.03 impl=0.12
item: milestone-review         design=0.00 impl=0.16
item: milestone-review         design=0.00 impl=0.08
item: milestone-review         design=0.00 impl=0.12
design-buffer: 0.15
total: 1.39
```

Derivation notes.

- **`issue-spec` design 0.35 is BELOW the 0.5–1.5 band floor, deliberately and
  with the measurement behind it.** The claim synced at `14:56` and the plan
  cleared plan-quality at `15:17` — **0.35h measured**, covering the origin-vocab
  survey over 300 entries, the corrected-rule measurement, filing the issue,
  reshaping `#30`, and three plan-gate rounds. `#29` priced this line at 0.70 and
  `#31` at 0.50; this design was genuinely smaller because the *question* was
  settled in conversation before the issue existed — the operator had already
  chosen inference, and the gate rounds refined a rule rather than choosing one.
  Recorded as a floor breach rather than rounded up to hide it, which is the
  accounting `#31`'s estimate gate asked for in the other direction.

- **Task 1 at 0.12 is the largest module line**, and it is the tables rather than
  the code: the corpus sweep asserts an outcome for all 34 fixtures, so the work
  is deciding 34 expected values against measured ORIGIN text, not writing
  mask-then-search.

- **Task 2 at 0.14 is the widest**, because `commandCtx` gains a field that three
  construction sites must fill and both loops must supply from `session.entry`.
  `#29` priced its equivalent at 0.18 when it also had to invent the closure;
  here the closure exists.

- **Task 4 is `cross-cutting-refactor`, not `atlas-docs`.** It adds a code-owned
  const with a doc-sync consumer AND sweeps two prose sites by hand across two
  files — `#29`'s review made exactly this reclassification when a docs task
  turned out to be a multi-site sweep, and `#31`'s block was corrected the other
  way when it genuinely was a docs pass. 0.12 sits inside the scaled 0.08–0.20.

- **Three `milestone-review` rows**, the shape `#29` and `#31` both settled on:
  running the boundary review (0.16), the manual verification pass (0.08), and
  REMEDIATING what the review returns (0.12). The last is not padding — `#29` took
  four rounds and `#31` four, and the two most recent boundary reviews each
  returned work on the first pass.

- **The declared total was 1.33 and the arithmetic is 1.39.** Caught by my own
  reconciliation check rather than by the gate, which let it through. Corrected
  rather than left, because a ledger row that does not add up is worse than one
  that is merely wrong: `#117` reads these to calibrate, and a 0.06 slip is
  indistinguishable from a deliberate adjustment.

- **No point forecast.** `#29` closed at 0.83× and `#31` at 0.90× against a
  `tools` v3.1 median near 0.7, so the recent rows sit closer to 1.0 than the
  median does. Two rows is not a trend; the ledger is the measurement.

## Plan

Designed. Durable plan: `workshop/plans/000035-pron-infer-plan.md` (4 tasks,
single pass, no `Mx`).

- [x] Claim, then design via `sdlc start-plan`.
- [ ] `OriginLanguage` — the pure inference, mask-then-search over two measured
      tables.
- [ ] `/pron` uses it: no argument infers, reports the choice, errors with the
      reason.
- [ ] Pin that #29's D1 is untouched — an ordinary lookup never infers.
- [ ] Docs: `pronHelp` (which both docs already consume) plus the atlas record
      of why bare Greek is ancient and why this table is accepted where
      ParseLang refuses one.

## Log

### 2026-08-29

Filed from the operator's request while discussing `#30`. Every number above was
measured before filing. Sequenced BEFORE `#30` deliberately: it makes both of
`#30`'s click targets thin wrappers over gestures that already exist — a headword
click is a replay, and a click on `ORIGIN French` is `/pron` with that language —
so the inference question is settled before the screen work starts rather than
tangled into it.

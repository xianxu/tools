---
id: 000029
status: working
deps: [tools#27]
github_issue:
created: 2026-08-28
updated: 2026-08-29
estimate_hours: 2.78
started: 2026-08-29T06:16:23-07:00
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

### Spanish borrowings are the strongest case, and they carry a constraint

Measured 2026-08-28, after the issue was first written. Spanish borrowings in
English are common, their anglicisation is often far from the source, and CDN
coverage is **much better than French**:

| word | `_es_es_` | `_en_us_` |
|---|---|---|
| `tortilla`, `chorizo`, `quesadilla`, `burrito`, `guacamole` | **200** | 200 |
| `jalapeño` | **200** (and `es_us` 200) | 200 (as `jalapeno`) |

Compare French, where `déjeuner` and `rendez-vous` are 404 in every form tried.
If this issue ships one language first, Spanish is the one that pays.

**THE CONSTRAINT, and it is not a detail:**

```
jalapeno_en_us   200   ← what define asks for today
jalapeño_es_es   200   ← the Spanish recording
jalapeno_es_es   404   ← the same word, Spanish locale, UNACCENTED
```

**The source recording is keyed on the source ORTHOGRAPHY.** English keys
`jalapeno`, Spanish keys `jalapeño`. So this issue cannot simply swap the
language field in the existing URL builder: `define jalapeno` — which is what a
person types — carries the wrong string for the Spanish URL.

The information is available: NOAD's headword renders `jalapeño` with the tilde
even though the word was typed without it. So the fix is reachable, but it means
the source SPELLING is an input to the candidate builder alongside the source
language, and `AudioCandidates` currently takes only the typed word.

### Relationship to `#27` — orthogonal, and this issue depends on it

They answer different questions and compose:

- **`#27` — which VARIANT of one language.** The Castilian/seseo split is
  phonemic, not an accent flavour: `cazar` /θ/ ≠ `casar` /s/ in `es_es`, both /s/
  in `es_us`. It applies whether or not a borrowing is involved.
- **`#29` — which LANGUAGE, per word, without switching mode.**

`jalapeño` needs both, and proves the dependency direction: `jalapeño_es_es` and
`jalapeño_es_us` are BOTH 200, so knowing the word is Spanish does not determine
which recording to fetch. This issue must defer to `#27`'s locale policy rather
than inventing one. **`#29` depends on `#27`, not the reverse.**

## Done when

- [ ] A borrowed word can be heard in its source language **without changing the
      session's language** — the deck, dictionary and highlight set stay English.
- [ ] A source-language recording that does not exist degrades to the English
      one, audibly the same as any other miss — French coverage is partial and
      `hotel` and `debut` are the cases to test. **NOT `déjeuner`**, which this
      Done-when originally named: it has no NOAD entry at all, so the lookup
      fails before audio is ever reached and it cannot exercise this path. See
      the 2026-08-29 Log entry.
- [ ] Italian's absence from the CDN is reported honestly rather than as silence.
- [ ] Whether the language is declared or inferred is a DECISION with its reason
      recorded, reconciled against `#23`'s rejection of inference.
- [ ] A word whose source spelling differs from its typed form is fetched
      correctly — `jalapeno` typed must reach `jalapeño_es_es`, not
      `jalapeno_es_es`, which is a 404.
- [ ] The locale for a source-language recording comes from `#27`'s policy rather
      than a second one invented here.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.70 impl=0.10
item: smaller-go-module        design=0.02 impl=0.12
item: smaller-go-module        design=0.02 impl=0.10
item: smaller-go-module        design=0.02 impl=0.08
item: smaller-go-module        design=0.03 impl=0.12
item: smaller-go-module        design=0.03 impl=0.16
item: smaller-go-module        design=0.02 impl=0.10
item: smaller-go-module        design=0.04 impl=0.18
item: cross-cutting-refactor   design=0.04 impl=0.18
item: smaller-go-module        design=0.00 impl=0.12
item: milestone-review         design=0.00 impl=0.20
item: milestone-review         design=0.00 impl=0.10
item: smaller-go-module        design=0.00 impl=0.16
design-buffer: 0.15
total: 2.78
```

Derivation notes.

- **`issue-spec` design 0.70 is the biggest line, and it is not the issue
  authoring.** `#27` established the rule and `#26` taught it: `sdlc actual`
  anchors on the CLAIM commit, so everything after the claim is inside the
  window. Here that is the whole design — ~15 further CDN probes, the 400-entry
  `(also …)` survey, the per-language notation measurement, two operator
  decision rounds, the plan, two plan-gate rounds, and filing `#30`. NOT
  discounted ×0.2: the Spec explicitly left the design open ("Not yet designed
  — two shapes are plausible and they are different sizes"), so the design
  happened here rather than being read off the page. The number is empirical
  rather than picked off the band — the claim commit is `2026-08-29T06:16:23`
  and the estimate gate cleared past `07:06`, so ~0.85h of wall clock is already
  spent with implementation not yet begun; 0.70 is that minus plausible idle.

- **Seven `smaller-go-module`s, all ×0.2 on design, because the plan now
  pre-resolves them.** Each carries a stated contract, so what is left is
  writing it. They are not equal: Task 3 (`SourceSpellings`) is 0.08 because it
  is assembly over two functions written in the tasks before it, while Task 7
  (`/pron` + `commandCtx.replay` + both loops) is 0.18 — the only one touching
  the raw/cooked boundary, where `workshop/lessons.md` records that playing
  inside the cooked block silently swallows Ctrl-C. Task 5 is 0.16 for breadth
  rather than depth: five `playAnnounced` call sites, where a missed one is a
  compile error rather than a silent bug.

- **Task 8 is `cross-cutting-refactor`, not `atlas-docs`.** It is a five-site
  sweep across `audiourl.go`, `audiourl_test.go` (twice), `atlas/define.md` and
  `README.md`, plus a new doc-sync test, `fs.Usage`, and five verification
  greps — a multi-file coordinated edit, which is what that primitive names.
  Filed as `atlas-docs` at first, which forced impl 0.10 past the v3.1-scaled
  0.02–0.08 ceiling for that primitive: the band was breached because the
  primitive was wrong, and 0.18 sits inside `cross-cutting-refactor`'s scaled
  0.08–0.20 honestly. The plan-quality gate found this sweep at three sites
  when it was written as one; pricing it as ordinary docs maintenance would
  make the same mistake in the other currency.

- **Two `milestone-review` rows for one boundary, because the boundary has two
  chunks.** 0.20 is the fresh-context review of a diff spanning nine files plus
  a new one — `#27` priced 0.16 as *below* the ceiling for a guard deletion, a
  const, two tests and doc lines, so this diff does not fit under that number.
  The second 0.10 is the plan's "Verification before close": seven live CLI
  invocations against the real CDN, the real dictionary and real audio playback
  — none of it automatable — plus `go test ./...` **and** the unfiltered
  `-tags conformance` suite. It was riding invisibly inside the review row.

- **Task 9 is `smaller-go-module`, NOT `real-api-discovery`.** It was filed as
  the latter and the note defending it said "there is no discovery left to
  budget — only four rows to write and one header comment to correct", which is
  a description of module work; the primitive was being used as a container for
  a number rather than as a claim about the work, and the ledger row would have
  read as "this issue paid to discover an API" when it did not. The seam, the
  `head()` helper and three sibling rows already exist and this issue has probed
  that CDN ~40 times. 0.12 is unchanged — it sits inside both bands, so only the
  label was wrong.

- **A line for REMEDIATING what the review returns, which nothing priced.** The
  0.20 row above buys *running* the fresh-context review; fixing its findings
  was free. The evidence that it is not: `#27`'s close review produced two
  follow-up commits (`c17d1c8` "the doc-sync reached one doc, and the wiring had
  no test", `c9ccf5e` "BR-8 — I skipped the human half of my own rename guard"),
  and this issue's own plan-quality gate returned seven findings in one round,
  two of them Important. A diff touching the raw/cooked boundary and a five-site
  invariant sweep is not a likely clean first pass. 0.16, design-free.

- **No `TUI screen + state machine` primitive**, despite `/pron` being an
  interactive surface. There is no new screen and no new state — D2 is
  specifically the decision NOT to add a mode, so the command's whole footprint
  is one registry row plus a closure. Had `/pron` been the session-scoped
  variant, this line would exist.

- **This will probably land low, but a POINT prediction is not warranted.**
  Computed over `calibration-ledger.tsv` rather than recalled: all 21 trusted
  `tools` rows are `estimate-logic-v3.1`, `ratio` (estimate ÷ actual) has
  **median 0.64**, and **16 of 21 are under 1.0**. So the under-bias is real and
  2.78 more likely lands near 4h than near 2.8h. But the spread is
  `[0.20 … 2.83]` with a genuine over-estimate tail (`#14` 2.83, `#9` 2.16,
  `#24` 2.16, `#5` 1.85, `#21` 1.55) and a mean of 0.91 — so "skews under almost
  uniformly" would be false, and naming a single expected figure would be
  reading a median as a forecast. An earlier draft of this note quoted only the
  sub-1.0 rows and inferred "median ≈ 0.65"; the number was near-right and the
  derivation was cherry-picked, which is worse than being wrong loudly.
  The estimate is NOT inflated to compensate: v3.1 is applied faithfully so the
  ledger row reads as model bias rather than estimator error, which is what
  `#127`'s recalibration needs from it. One thing that will NOT absorb the gap is
  fan-out — this plan is a dependency chain (Task 3 assembles 1–2, Task 4
  consumes 3, Task 5 consumes 4, Tasks 6–7 consume 5, Task 8 disposes of what
  Task 4 falsifies), so the actual should track the sequential sum.

## Plan

Designed. The durable plan is `workshop/plans/000029-origin-pronunciation-plan.md`
(9 tasks, one review boundary — single-pass, so no `Mx` tags per AGENTS.md §3).

- [x] Design via `sdlc start-plan`. Coordinate with `#27`, which owns
      pronunciation as a parameter rather than a literal. **Result: `#27` needs
      no coordination — `voiceFor`/`localeFor`/`defaultLocale` are reused
      UNCHANGED, so the locale is literally `#27`'s policy and not a second one.**
- [ ] `differsOnlyByDiacritics` + `Entry.AlsoSpellings` — the `(also …)` filter.
- [ ] `SourceSpellings` — headword, then diacritic-only alternatives, then typed.
- [ ] `utterance` — the whole walk: source spellings first, session as fallback.
- [ ] `speak`/`playAnnounced`/`reportVoice` — report the voice that ANSWERED.
- [ ] `-pron fr`, refused when there is no word to apply it to.
- [ ] `/pron fr` — one-shot replay, no mode left behind.
- [ ] Docs derive (`pronHelp` + doc-sync); rewrite the atlas's now-false
      "One language, no fallback".
- [ ] Live conformance rows for every measurement the design rests on.

## Log

### 2026-08-28

Filed from the operator's request. Every measurement above was taken before
filing — the CDN probes, the dictionary language pairs, and the observation that
`-lang fr` already fetches the French audio — so the design starts from what is
true rather than from what seems likely. Two of my own guesses were falsified
while measuring: that French coverage was general (it is partial), and that the
`déjeuner` 404 was an accent-encoding problem (it is not).

### 2026-08-29

Claimed, `start-plan`ed, designed. Plan at
`workshop/plans/000029-origin-pronunciation-plan.md`.

**Two decisions, both argued from NEW measurement rather than inherited.**

**D1 — declared, not inferred.** The issue left this open, requiring that
reversing `#23` "must be argued, not assumed". It is not reversed, and the
argument is now stronger than `#23`'s: NOAD's ORIGIN and the CDN BOTH fail to
discriminate a live loanword from a naturalised one.

```
arrondissement  ORIGIN French, from arrondir 'make round'.
police          ORIGIN … from French, from medieval Latin politia …
```

| `_fr_fr_` 200 | `police` `restaurant` `garage` `machine` `unique` `genre` `nuance` `montage` `bureau` |
| `_fr_fr_` 404 | `hotel` `debut` `ballet` |

So automatic origin audio would silently replace the English recording for a
large class of ordinary words. The CDN is not the discriminator the "try it and
fall back" shape assumed it was.

**D2 — `/pron fr` is an ACTION, not a mode.** `/sound` and `/lang` name settings;
"say it in French" is a thing you do once. Operator chose this over a session
mode. It is also exactly the call `#30`'s click will make.

**D3, which fell out of the design and is load-bearing:** `-pron` must NOT enter
`opt.voice`. That field is a SESSION value which `applyLang` re-derives on every
`/lang`, so an override living there would survive a switch and ask for `fr_fr`
recordings in a Spanish session — the drift `applyLang`'s enumeration comment
exists to prevent. Hence `-pron` is refused when there is no word to apply it to.

**A Done-when was wrong and is corrected above.** It named `déjeuner` as the
degrade test. `déjeuner` has NO NOAD entry, so `define déjeuner` fails at the
dictionary and never reaches audio — it cannot exercise the fallback. `hotel`
and `debut` are the real cases: `_fr_fr_` 404, `_en_us_` 200.

**The orthography constraint is WIDER than this issue recorded.** It documented
the headword as the source of the source spelling (`jalapeno` → `jalapeño`).
Measured: NOAD files the accented form on EITHER side of the headword.

| typed | headword | `(also …)` | the 200 |
|---|---|---|---|
| `jalapeno` `pinata` `senor` `cliche` `fiance` | accented | — | headword |
| `cafe` `naive` `facade` | **unaccented** | `café` `naïve` `façade` | **the alternative** |

Headword alone would have missed three of eight. So `(also …)` is a second
source — but surveyed over 400 live entries it is NOT a spelling list: it holds
phrases (`(also good as gold)`), compounds (`(also jalapeño pepper)`),
derivatives (`(also naïveness)`) and English variants (`(also advisor)`,
`(also caldron)`, `(also convertor)`). Admitting only alternatives that differ
from the headword by DIACRITICS ONLY took all three real gains and, across that
400-entry sample, nothing else at all. That rule needs no `x/text` and no table
of accented characters — same length, and every difference where one side left
ASCII.

Also measured: `Señor_es_es` is a 404 while `señor_es_es` is a 200, so
`AudioCandidates`' existing lowercase has to reach the headword-derived spelling
too. And Japanese joins Italian as absent from this CDN generation — `karaoke`
and `tsunami` are 404 in both `ja_jp` and `ja_ja`.

**Split out:** `#30` — clickable regions in the terminal, filed from the
operator's request in this session. It depends on this issue for the mechanism,
and its first two targets are the `ORIGIN` language and the headword. Its Spec
records why clicking dissolves the AMBIGUITY objection to ORIGIN inference
(`piano` names two languages; a pointer picks one) but NOT the closed-table one.

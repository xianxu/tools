---
id: 000031
status: working
deps: [tools#29]
github_issue:
created: 2026-08-29
updated: 2026-08-29
estimate_hours:
started: 2026-08-29T11:29:22-07:00
---

# curate the Italian dictionary so /lang it is a real mode (French and German split to #34)

## Problem

`#29`'s Spec promised this and nothing tracked it:

> Adding them to `curated` in `cmd/define/dictselect.go` is **one line each** and
> makes `/lang fr|it|de` work as a full mode. That is worth doing and is NOT this
> issue: it serves someone learning those languages, which the operator explicitly
> is not. Split out so the cheap win is not blocked on this design.

The split was correct — `#29`'s purpose was origin pronunciation without a mode
switch, and curating dictionaries serves the opposite need. But a promised split
with no tracker item evaporates; `#29`'s close review (BR Minor,
`split-out-not-filed`) found it surviving only as prose in `#30`.

## Spec

**Measured 2026-08-28.** All three are installed on this machine and **strictly
monolingual**, so `chooseDictionary` accepts them as-is:

| identifier | pairs |
|---|---|
| `com.apple.dictionary.fr.Multi` | `fr>fr` (monolingual despite the name) |
| `com.apple.dictionary.it.Devoto-Oli` | `it>it` |
| `com.apple.dictionary.de.DDDSI` | `de>de` |

**What is broken today, measured 2026-08-29.** `-lang fr` does not fail — it
silently falls back to searching every active dictionary and answers from NOAD:

```
$ define -raw -no-audio -lang fr bonjour
define: no known fr dictionary is installed; searching every active dictionary
bonjour | bänˈZHo͝or, bänˈZHôr | exclamation a French g…
```

That notation is NOAD's ANGLICISATION, not French. A learner asking for French
gets English answers wearing a French headword, which is worse than a refusal.

**This blocks a measurement `#30` needs.** `#30` wants to know whether each
language's dictionary carries pronunciation notation, because its click target
depends on it — English always does, Spanish provably never does (phonemic
orthography). French, Italian and German are currently **unmeasurable**, and the
row in `#30`'s Spec reads "unknown" for exactly this reason.

**Not just the one line each.** `#23 M2` made the dictionary follow the mode, so
adding a row means the fixture corpus and the conformance rows should follow:
`testdata/entries/<lang>/` exists for `en` and `es`, and
`TestSelectedDictionaryAnswersInItsOwnLanguage` is the live check that a curated
book answers in its own language rather than in English.

**Audio is a separate matter and mostly absent.** Measured in `#29`: German
`schadenfreude_de_de` is a 200, French coverage is partial, and Italian has **no
recordings at all** in this CDN generation (nine probes, zero hits). So a curated
Italian dictionary gives definitions and no sound. That is honest and worth
having; it should not be a surprise.

## Done when

Scope narrowed to Italian on 2026-08-29 — see `## Revisions`.

- [ ] `/lang it` answers from `it.Devoto-Oli` rather than falling back to NOAD.
- [ ] A curated book that is NOT installed still degrades loudly, as `#23 M2`'s
      three-outcome table requires — adding a row must not turn a missing
      dictionary into a silent English answer.
- [ ] Italian has a committed fixture corpus, so the parser suite and the
      no-data-loss invariant cover it rather than covering `en` and `es` and
      claiming more.
- [ ] Whether these dictionaries carry pronunciation notation is MEASURED and
      recorded, since `#30` reads that row. **Already done — see Revisions.**
- [ ] The absence of Italian audio is stated where a learner will meet it, not
      discovered as silence.
- [ ] The new language does NOT enter unswept: the live raw-notation ratchet is
      English-only (`live_property_test.go` walks `/usr/share/dict/words`), so
      either Italian gets a ratchet of its own or the gap is stated as a
      decision rather than left as an oversight.

## Plan

Designed. Durable plan: `workshop/plans/000031-curate-dictionaries-plan.md`
(4 tasks, single pass, no `Mx`).

- [x] Claim, then design via `sdlc start-plan`.
- [ ] Capture the Italian corpus — words chosen for what they prove, not for
      vocabulary, following `capture.sh`'s own convention.
- [ ] The `curated` row, plus rewriting the doc comment sentence that this
      issue's measurement made half wrong.
- [ ] Generalise the own-language conformance check to a table rather than
      copying the Spanish one.
- [ ] Docs: Italian, the English-only ratchet gap, and the notation table `#30`
      reads.

## Log

### 2026-08-29

Filed from `#29`'s close review. The measurements above were taken during `#29`
and are inherited rather than re-derived; the `-lang fr bonjour` fallback was
re-run today to confirm it is still the behaviour.

## Revisions

### 2026-08-29 — measured before designing; scope narrows to Italian

**Reason:** the Spec inherited `#29`'s "one line each" and that turned out to be
false for two of the three books. Measured by querying each dictionary directly
through `selectedDictionary{ids: …}`, over five common words each:

| dictionary | senses the parser finds | longest single undifferentiated blob |
|---|---|---|
| `NOAD` (English, baseline) | 153 | 144 runes |
| **`it.Devoto-Oli`** | **245** | 645 runes |
| `fr.Multi` | **6** | **2,811 runes** |
| `de.DDDSI` | **5** | **4,404 runes** |

**Italian parses at NOAD-comparable quality** because Devoto-Oli marks senses
`1`, `2` and sub-senses `•` — the shapes `parseSenses` already knows. French and
German collapse: the parser's part-of-speech vocabulary is English (`noun`,
`verb`, not `nom masculin` / `s.f.` / `Substantiv`) and its section names are
`ORIGIN`/`DERIVATIVES`, not `ETIMOLOGIA` / `HERKUNFT` / `SYNONYME`. So a whole
entry lands in one paragraph, and `Haus` puts its grammar table, its synonym
list and its etymology into a single 4,404-rune example.

**Delta:** curate `it.Devoto-Oli` only. French and German split to `#34` with
these measurements, so the work there starts from what is true rather than from
"one line each" a second time. Operator's decision, taken against the table above.

### The pronunciation-notation question `#30` needs is now ANSWERED

It was the reason this issue was called a precondition, and it turns out to be
answered by measurement rather than by curation — so `#30` is unblocked either
way:

| language | notation | detail |
|---|---|---|
| English | IPA, always | `| ˈrekərd |` |
| Spanish | **none** | phonemic orthography; the dictionary writes nothing |
| **French** | **none** | `fr.Multi` carries no pronunciation field at all |
| **Italian** | **not IPA** | parenthesised syllabification with stress: `(cià·o)`, `(pìz·za)`, `(e·sprès·so)`. Not recognised as notation by `isPronunciation`, and correctly so — it is a syllable break, not a phonetic transcription |
| **German** | **real, often lost** | Duden's field: `Wạsser`, `Hụnd`, `ˈkatsə, Kạtze`. But the LONG-vowel stress mark is an underline that does not survive plain-text extraction, so 9 of 15 sampled entries render the bare word — `/Haus/`, `/Schadenfreude/`. Informationally empty, visually indistinguishable from a real transcription |

That last row is the one to carry into `#34`: German's pronunciation field is not
fabricated, it is *lossy*, and it needs deciding rather than shipping.

### Two facts worth inheriting

- **`fr.Multi` is QUÉBÉCOIS, not France French.** Its examples are Outremont,
  Plateau-Mont-Royal and Fromagerie Tournevent, sourced from the GDT. A learner
  choosing "French" would not expect that, and `#34` should say so wherever the
  book is named.
- **`OxfordFrench`, `OxfordGerman` and `OxfordSpanish` are installed too**, and
  are BILINGUAL (`fr>fr` AND `en>fr`). `monolingualIn` correctly rejects them —
  which is the guard working, and the reason `#34` cannot simply prefer "the
  other French dictionary".

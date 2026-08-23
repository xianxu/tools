---
id: 000018
status: open
deps: [tools#1, tools#9, tools#10, tools#12]
github_issue:
created: 2026-08-22
updated: 2026-08-22
estimate_hours:
---

# Spanish support: pronunciation locale, language-aware deck and agreement-safe distractors

## Problem

The operator is learning Spanish alongside English, and their kids will too. Most
of `define` already works for Spanish — but the parts that don't, don't fail
loudly, and one of them (`--play`'s distractor selection) would produce items a
learner can solve without knowing any vocabulary.

Everything below was **measured on 2026-08-22**, not reasoned about.

## What already works, verified

- **Definitions.** `DCSCopyTextDefinition(NULL, …)` searches the user's active
  dictionary set, so Spanish needs no code at all. Both `Spanish - English`
  (bilingual Oxford) and `Spanish` (monolingual Vox) ship with macOS. Confirmed
  live: `madrugar` → *"A intransitive verb to get up early … a quien madruga Dios
  lo ayuda"*.
- **News grounding (#9).** The feed URL already takes `hl`/`gl`/`ceid`;
  `hl=es&gl=ES&ceid=ES:es` is a parameter change, not a design one.
- **Everything model-shaped.** `internal/llm` is language-agnostic. #10, #12, #13,
  #16, #17 need prompts that name the language, not new structure.
- **The store and scheduler.** `store.Key()` lowercases via Unicode, so accents
  survive round-trip.

## Spec

### M1 — pronunciation, and why the locale is load-bearing

`AudioCandidates` (`cmd/define/audiourl.go:41`) writes the language as a **literal**:

```go
out = append(out, audioBase+"/pronunciation/2022-03-02/audio/"+shard+"/"+esc+"_en_"+locale+"_"+n+".mp3")
```

So `madrugar` is only ever requested as an English word. Measured — all four
candidates 404 while the recording sits two characters away:

```
madrugar_en_us_1.mp3   404      ← what define asks for
madrugar--_us_1.mp3    404
madrugar_es_es_1.mp3   200      ← what exists
madrugar_es_us_1.mp3   200
```

Coverage is real, not incidental: `sobremesa`, `empalagoso`, `chapucero`,
`desvelarse` all return 200 on `es_es`.

**The locale is not cosmetic for Spanish, and this is the part worth designing
around.** Spanish orthography is phonemic — spelling plus the written accent
determines pronunciation exactly — so Spanish dictionary entries carry **no
phonetic notation at all**, unlike NOAD's `lig·a·ment | ˈliɡəmənt |`. Verified on
the live entries. That means:

- `ParseEntry` finding no pronunciation on a Spanish entry is **correct**. Do not
  "fix" it.
- The recording is therefore the *only* place pronunciation information exists,
  which makes audio more load-bearing for Spanish than for English.
- And the two locales encode a genuine phonemic split, not an accent flavour:
  `es_es` is Castilian (*cazar* /θ/ ≠ *casar* /s/), `es_us` is Latin American
  *seseo* (both /s/). Choosing one chooses which sound system a learner acquires.
  `es_419`, `es_mx` and `es_ar` are **not** valid — measured 404.

Design: language becomes a parameter beside locale. `AudioCandidates` already
returns an *ordered* list that `httpAudioSource` walks until the first 200, so
widening it costs no new mechanism — an unspecified language can try several,
and `-lang es` reorders rather than restricts. A conformance assertion belongs
beside `TestCDNStillServesTheExpectedPaths`, which pins the English ordering the
same way.

### M2 — the deck has to know what language a word is

Blocked on #10/#12 existing. Three things Spanish needs that English let us dodge:

- **A language dimension on the deck.** One directory per language works today
  and may be enough; decide deliberately rather than by accident.
- **Inflection.** English mostly gets away with surface forms; `hablar` has ~50.
  Looking up `hablaba` should not create a deck entry distinct from `hablar`.
  Store the lemma as identity and the encountered form as provenance — the model
  lemmatises reliably, and reviewing the lemma while recalling where you met it is
  the pedagogically right shape.
- **Gender is part of the word.** Knowing `mesa` without knowing it is feminine is
  not knowing it. Nouns carry gender in the deck, and a form should be able to
  test it.

### M2 — agreement-safe distractors (the one that would actually break #12)

#12 already requires that *"the blanked sentence never leaks the answer (stem,
plural, hyphenation)"*. Spanish makes that surface far larger, because **grammar
leaks the answer independently of meaning**:

> *La actitud del jefe era claramente ______.*

Every masculine option is eliminable **without knowing what any of the words
mean**. Same for number agreement, and for `un`/`una` preceding the blank.

So distractor selection needs a **grammatical-agreement filter** alongside the
semantic-distance one: candidates must match the answer in gender and number as
the stem requires. This is a new filter, not a tuning of the existing one, and it
is the difference between a real item and one a learner solves by inspection.

## Done when

M1:
- [ ] `define madrugar` plays a recording; asserted against the fake, and a
      conformance test measures the live CDN the way the English ordering is.
- [ ] `-lang` selects the language and `-locale` still selects the variant;
      `es_es` and `es_us` both reachable, and the help text says what the
      difference *is* (θ vs seseo) rather than naming two country codes.
- [ ] An English word with no recording still degrades to a warning, exit 0 —
      the existing behaviour does not regress.
- [ ] No pronunciation is rendered for a Spanish entry, and a test says that is
      expected rather than a gap.

M2:
- [ ] A Spanish word looked up in an inflected form lands in the deck once, under
      its lemma, with the encountered form kept.
- [ ] Nouns carry gender, and it survives round-trip.
- [ ] A cloze whose stem forces agreement never offers an option that disagrees —
      asserted with the `La actitud … era claramente ______` shape, where a
      masculine distractor is a defect.
- [ ] Authored items and the learner model both know which language they are for.

## Plan

- [ ] Design via `sdlc start-plan` before implementing. M1 is independently
      shippable and does not wait on #10.

## Log

### 2026-08-22

Created from the operator's question ("can the same thing work with Spanish?")
after measuring each layer against the live dictionary and CDN rather than
reasoning about it.

**A wrong claim worth recording.** I first reported "no Spanish dictionary is
enabled on this Mac" from a probe that returned `no dictionary entry` — while the
operator had already said the definition worked. It was a **sandbox artifact**:
`DCSCopyTextDefinition` cannot reach the system dictionary assets from inside the
sandbox and reports the failure as a clean miss, which is indistinguishable from
a genuinely absent word. Same binary, same word, sandbox off → full entry. Filed
to `workshop/lessons.md`; the operator's observation should have outranked my
probe.

---
id: 000031
status: open
deps: [tools#29]
github_issue:
created: 2026-08-29
updated: 2026-08-29
estimate_hours:
---

# curate the French, Italian and German dictionaries so /lang fr|it|de is a real mode

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

- [ ] `/lang fr`, `/lang it` and `/lang de` answer from the language's own
      dictionary rather than falling back to NOAD.
- [ ] A curated book that is NOT installed still degrades loudly, as `#23 M2`'s
      three-outcome table requires — adding rows must not turn a missing
      dictionary into a silent English answer.
- [ ] Each new language has a committed fixture corpus, so the parser suite and
      the no-data-loss invariant cover it rather than covering `en` and `es` and
      claiming more.
- [ ] Whether these dictionaries carry pronunciation notation is MEASURED and
      recorded, since `#30` reads that row.
- [ ] The absence of Italian audio is stated where a learner will meet it, not
      discovered as silence.

## Plan

- [ ] Claim, then design via `sdlc start-plan`.

## Log

### 2026-08-29

Filed from `#29`'s close review. The measurements above were taken during `#29`
and are inherited rather than re-derived; the `-lang fr bonjour` fallback was
re-run today to confirm it is still the behaviour.

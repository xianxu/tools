---
id: 000031
status: codecomplete
deps: [tools#29]
github_issue:
created: 2026-08-29
updated: 2026-08-29
estimate_hours: 1.81
started: 2026-08-29T11:29:22-07:00
actual_hours: 2.09
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

- [x] `/lang it` answers from `it.Devoto-Oli` rather than falling back to NOAD.
- [x] A curated book that is NOT installed still degrades loudly, as `#23 M2`'s
      three-outcome table requires — adding a row must not turn a missing
      dictionary into a silent English answer.
- [x] Italian has a committed fixture corpus, so the parser suite and the
      no-data-loss invariant cover it rather than covering `en` and `es` and
      claiming more.
- [x] Whether these dictionaries carry pronunciation notation is MEASURED and
      recorded, since `#30` reads that row. **Already done — see Revisions.**
- [x] The absence of Italian audio is stated where a learner will meet it, not
      discovered as silence.
- [x] The new language does NOT enter unswept: the live raw-notation ratchet is
      English-only (`live_property_test.go` walks `/usr/share/dict/words`), so
      either Italian gets a ratchet of its own or the gap is stated as a
      decision rather than left as an oversight.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.50 impl=0.08
item: smaller-go-module        design=0.02 impl=0.08
item: smaller-go-module        design=0.02 impl=0.08
item: smaller-go-module        design=0.02 impl=0.10
item: atlas-docs               design=0.03 impl=0.07
item: smaller-go-module        design=0.02 impl=0.10
item: smaller-go-module        design=0.00 impl=0.08
item: smaller-go-module        design=0.03 impl=0.12
item: milestone-review         design=0.00 impl=0.16
item: milestone-review         design=0.00 impl=0.08
item: milestone-review         design=0.00 impl=0.12
design-buffer: 0.15
total: 1.81
```

Derivation notes.

- **`issue-spec` design 0.50 sits AT the band floor, and the measurement is why
  rather than the band.** The claim synced at `11:29`; the plan cleared
  plan-quality at `11:54` — **0.42h measured**, covering the four-dictionary
  probe, the render-quality and pipes/IPA measurements, the operator's scope
  decision, filing `#34`, and three plan-gate rounds. 0.50 is that plus the
  estimate round still to run. `#29` priced this line at 0.70 for a window that
  measured longer and carried a genuinely open design question; here the design
  question was answered by a table.

- **Six `smaller-go-module`s, all ×0.2 on design**, and they are not equal.
  Task 2 is 0.08, the band floor, not the 0.06 first written: it is a full TDD
  cycle — a two-case `chooseDictionary` test including the bilingual
  `OxfordItalian` rejection, run red, the row, the comment rewrite, run green —
  plus the joint commit with Task 1. The Italian no-IPA sibling is 0.08 for the
  same reason. Both were below the scaled floor (0.08–0.20) in the first block,
  which is the asymmetry the estimate gate caught: it named `atlas-docs`' ceiling
  and then breached its own floor twice unremarked. Task 5's paired
  `curated`-derived guards are 0.12 because they are two tests plus their removal
  verification. Task 3 (0.10) is a rewrite rather than an addition: the
  own-language conformance check is Spanish-hardcoded and becomes a table, which
  is where a second language would otherwise become a second copy.

- **Task 1 is priced as a module (0.08) though most of it is data.** The bash
  edit is small; what costs is that the capture must run UNSANDBOXED and fails
  hard on a short read, so a failed run is a diagnose-and-retry loop rather than
  an error message.

- **`atlas-docs` 0.07, inside the v3.1-scaled 0.02–0.08.** `#29` had to
  reclassify its docs task as `cross-cutting-refactor` because a five-site
  invariant sweep is not a docs pass. This one genuinely is: one atlas section,
  one README paragraph, one notation table. The two stale comments D3 repairs are
  priced with Task 5a, where the test they describe changes.

- **No `real-api-discovery`.** `#29` mislabelled its conformance work that way
  and the review caught it. The dictionaries here are already behind
  `fakeDictionary` with a live half, and this issue probed them ~40 times during
  design; there is nothing left to discover.

- **Three `milestone-review` rows, because the boundary has three chunks.**
  Running the review (0.16), the manual verification pass (0.08), and
  REMEDIATING what the review returns (0.12). The last was first slugged
  `smaller-go-module`; it is post-review process overhead, which is what this
  primitive names, and mis-slugging it also made the block read as more code work
  than the plan carries. `#29` had four review rounds and `#27` two follow-up
  commits, so pricing remediation at zero is the one thing history rules out. The
  verification row at 0.08 is the manual
  pass — five CLI invocations plus `go test ./...` and the unfiltered conformance
  suite, which `#29`'s round 3 showed is not optional and not free.

- **Expect this to land near or above the estimate, not below**, and the reason
  is not the one first written here. `#29` closed at 0.83× (3.34 actual on 2.78)
  against a `tools` v3.1 median near 0.7. An earlier draft read that as "the
  under-bias did not hold for the most recent row" — wrong: 0.83 is still an
  UNDER-estimate, only a milder one, and the two most recent rows (`#27` 0.49,
  `#29` 0.83) point the same way. The operational conclusion stands and the
  framing did not. No point forecast; the ledger row is the measurement.

- **The calibration source is flagged `[stale]`** — `sdlc estimate-source`
  reports the ledger is newer than `estimate-logic-v3.1.md`, so the per-primitive
  hours may have drifted (`#127`). That matters here because two of these items
  sit exactly ON band edges which are themselves provisional. Derived against
  them anyway, as the tool instructs, and noted so the ledger row is not read as
  more precise than its bands.

## Plan

Designed. Durable plan: `workshop/plans/000031-curate-dictionaries-plan.md`
(5 tasks, single pass, no `Mx`).

- [x] Claim, then design via `sdlc start-plan`.
- [x] Capture the Italian corpus — words chosen for what they prove, not for
      vocabulary, following `capture.sh`'s own convention.
- [x] The `curated` row, plus rewriting the doc comment sentence this issue's
      measurement made half wrong. **Same commit as the capture**, because
      `TestCaptureScriptUsesTheCuratedDictionaries` compares the script against
      `curated` in both directions and either edit alone is red.
- [x] Generalise the own-language conformance check to a table rather than
      copying the Spanish one.
- [x] Docs: Italian, the ratchet gap D3 leaves open, and the notation table
      `#30` reads.
- [x] The four checks the captured directory does not give for free: sweep the
      raw-notation assertion over every captured language, pin Italian's absence
      of IPA, and make the README's language list and the corpus's coverage both
      derive from `curated`.

## Log

### 2026-08-29
- 2026-08-29: closed — Both suites on the FINAL head dcbb683: go test ./... 0 failures; go test -tags conformance ./cmd/define/ green unfiltered (306s) UNSANDBOXED so the NOAD rows executed. ROUND 3 named why rounds 1 and 2 kept finding more sites: the class was enforced PER-SITE, so each round pinned the surfaces its author could see. curated now has ONE registry (curatedSurfaces) of the surfaces obliged to name every curated language, each row supplying its own text and its own spelling — docs write "Italian", the flag help writes "it" — and one assertion ranges over registry x curated, replacing three per-site tests. The only hand-written list left in the family is "which surfaces exist". That closed the gap BR-10 had kept open, and the distinction is worth naming: TestTheLangFlagRegistersTheDerivedHelp pins DELIVERY (run passes langHelp, not a literal) while nothing pinned DERIVATION — hardcoding langHelp to {en,es} keeps delivery green and fails the registry row, verified both ways. Row 9: the language-keyed notation table cross-checks curated with an explicit exempt list carrying reasons, because "no row" and "deliberately no row" were indistinguishable and #34 will add German, whose Duden field is real, to a table asserting the opposite. BR-12 was mine to have caught: merging two notation tests and deleting a subsumed sweep are renames in the sense retiredSymbolNames exists for, and I added no rows — so the atlas, this plan and #30 all named tests the tree does not declare. Rows added, all three swept, and since the human half has failed twice (#27 is in that same map) the TRIGGER is now mechanical: TestARemovedDeclarationIsSweptOrRetired reads removed declarations out of the change window and demands a row or zero current-truth mentions, verified to fire on exactly that state (no row plus a stale atlas mention) and to stay quiet when either half is satisfied. The mapping still needs a person; noticing a removal does not. Minors: the bare block flattened, atlas spacing and wrapping fixed.; review verdict: FIX-THEN-SHIP

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

### 2026-08-29 — shipped

`/lang it` answers from the Devoto-Oli. Five tasks, one boundary. Plan:
`workshop/plans/000031-curate-dictionaries-plan.md`.

**The measurement was the work.** `#29` promised "one line each" for three
dictionaries; probing them turned that into one line for ONE dictionary and a
separate issue (`#34`) for the other two, because the parser's POS words and
section names are English and French/German collapse into a single blob of 2,811
and 4,404 runes.

**Three latent defects surfaced, each invisible until this issue's data reached
them:**

1. `TestCaptureScriptUsesTheCuratedDictionaries` matched identifiers with
   `[A-Za-z0-9._]+` — no hyphen — so `…it.Devoto-Oli` truncated to `…it.Devoto`
   and it reported the pair mismatched in both directions. Every curated
   identifier had been hyphen-free until now.
2. `TestNoRawPronunciationNotationSurvives` swept English while two committed
   comments called it a hard zero "over the committed corpus". Widened rather
   than the claim narrowed — measured first, so Italian passes the zero rather
   than needing an exemption, and both comments became true.
3. `TestDocsNameEveryCuratedLanguage`, which I wrote for this issue, **passed the
   moment it existed** — `#29` had left "Italian and Japanese have no recordings"
   in the `-pron` section, so free-text containment was satisfied while the
   dictionary paragraph still listed two languages.

**And I mis-verified my own guards once.** For the two `curated`-derived checks I
removed the `curated` row and found them still green — reading that as unpinned.
It is not: uncurating a language removes the obligation, so nothing is left to
check. The removal that pins an obligation must BREAK it, not withdraw it. Both
redden when the row stays and the README span or the corpus is emptied.

`#34` inherits the French/German measurements, including that `fr.Multi` is
Québécois and that German's Duden pronunciation field is lossy rather than
fabricated.

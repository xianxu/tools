# `/pron` Origin Inference Implementation Plan (`#35`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `/pron` with no argument plays the source recording for a word whose `ORIGIN` names exactly one modern language, says which language it chose, and errors with the reason when it cannot tell.

**Architecture:** One pure function over the parsed entry — `OriginLanguage(Entry)` — plus two measured tables and a mask-then-search order. `#29` already owns everything downstream: the command records a language, the loop replays it, `utteranceFor` builds the voice. This issue only changes where the language comes from when the user does not supply one.

**Tech Stack:** Go 1.26, stdlib. Tests: `go test ./...` and `-tags conformance`.

---

## Decisions

**D1 — this does not reopen `#29`'s rejection of AUTOMATIC origin audio, and a test says so.** `#29` measured that NOAD writes `ORIGIN French` identically for `arrondissement` and `police`, and that the CDN serves `police_fr_fr` at 200. That argument is about inference running on EVERY lookup. `/pron` is opt-in per word and reports its choice, so a wrong inference is one you asked for, on one word, and can see. The Done-when pins it with a test rather than a sentence: an ordinary lookup of a French-origin word must still request only the session's language.

**D0 — CUT cognate clauses before anything else, and this is the finding that reshaped the rule.** NOAD's etymologies name languages in two completely different roles, and only one is a source:

```
bring   Old English bringan, of Germanic origin; related to Dutch brengen and German bringen.
house   Old English hūs …, of Germanic origin; related to Dutch huis, German Haus …
water   Old English wæter …; related to Dutch water, German Wasser, … shared by Russian voda
casque  late 17th century: from French, from Spanish casco. Compare with cask.
```

The first three are Old English words with **no source language at all** — Dutch and German appear only as COGNATES, words sharing an ancestor. Searching the whole section would infer German for `bring`, which is `#29`'s D1 failure arriving by another road. So the text is truncated at the first cognate marker (`related to`, `compare with`, `cognate with`, `shared by`) before any language is looked for. `Germanic` is excluded outright: a family, not a language.

**And the same measurement dissolved the ambiguity branch.** `casque` — *"from French, from Spanish casco"* — is not ambiguous; it is a borrowing CHAIN, and NOAD's convention is that the first-named source is the immediate one. Same for `knout` (*"via French from Russian"*), `ballet` (*"from French, from Italian balletto"*) and `mesa` (*"Spanish … from Latin mensa"*). First-named-wins is therefore not a tiebreak but the correct reading, and after cutting cognates no case is left that needs an ambiguity error. `piano`'s *"either from French, or …"* is an editorial hedge; taking the first-named while REPORTING it is the honest answer, since the user sees `ORIGIN says French` and overrides in one word.

So "cannot determine" now means exactly one thing: **no modern language is named once cognates are cut and historical stages masked.** That is the operator's requested error, and the only one.

**D2 — mask historical stages BEFORE searching for modern ones, and the order is load-bearing.** Measured over 300 sampled entries: `Old French` occurs **11** times against `French`'s **8**. Searching for `French` first would call the majority case French. So: replace every historical stage in the ORIGIN text, then look for modern names in what remains.

**D3 — bare `Greek` is ANCIENT Greek and is excluded, though it has a code.** It is the second most common language word in the sample (6, behind `Latin`'s 19), and NOAD writes `modern Greek` when it means the living language. Including `Greek` would make a large class of ordinary words infer `el` — the same failure `#29`'s D1 rejected, arriving by a different road. `modern Greek` is left unhandled rather than special-cased: it did not occur in the sample, and a rule written for a case nobody has seen is a guess.

**D4 — historical stages are excluded BY CATEGORY, not because the CDN 404s them.** `Latin` has the code `la`, `Old English` has `ang`, `Sanskrit` has `sa`. Filtering on "no recording exists" would be right by accident and would silently start playing something the day Google adds Latin. The reason they are excluded is that a superseded stage of a language is not something a speaker says today — the concept `/pron` offers does not apply to it.

**D5 — the language-name table is accepted where `ParseLang` and `localeFor` refused one, and the difference is whose fact it is.** Those refusals were about what the **CDN and the installed dictionaries** serve — someone else's fact, changing without notice. This table reads **NOAD's editorial prose**, which is stable, small, and ours. It makes no claim about what the CDN has: an inferred language with no recording degrades through `#29`'s fallback exactly as a typed one does. Recorded because the next reader meets three refusals and one acceptance and deserves the distinction.

**D6 — NON-GOAL: the `-pron` FLAG does not infer.** It is the same feature on the one-shot side and cannot express "infer" — `fs.String` cannot distinguish an absent flag from an empty value without a sentinel like `-pron auto`, and inventing one for a path where the user has already typed the word is friction for nothing. `-pron fr` stays as `#29` shipped it. Recorded so the asymmetry is a decision rather than an oversight.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `originLanguages` | `cmd/define/origin.go` | new |
| `historicalStages` | `cmd/define/origin.go` | new |
| `OriginLanguage` | `cmd/define/origin.go` | new |
| `parsePronArgs` | `cmd/define/pron_cmd.go` | modified — no argument stops being an error |

- **`OriginLanguage(e Entry) (lang store.Lang, named string, err error)`** — the whole inference.
  - **Relationships:** 1:1 with an entry; consumed only by `runPron`.
  - **Returns the NAME it found as well as the code**, because the report must say `ORIGIN says French` rather than `fr` — the user is being told what the tool read, not what it derived.
  - **DRY rationale:** first occurrence. It is a new file rather than an addition to `parse.go` because it is a reading OF a parsed entry, not part of parsing one — `parse.go` knows nothing about languages and should keep it that way.
  - **Future extensions:** `modern Greek`, and `#30`'s click on a specific `ORIGIN` token, which supplies the name directly and needs only the name→code half.

- **`originLanguages` / `historicalStages`** — the two measured tables (D2, D3, D4).
  - **DRY rationale:** the only place a language name maps to a code. `#30` will consume the same map when a click names a language.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `commandCtx` | `cmd/define/command.go` | modified — gains `entry` | — |
| `runPron` | `cmd/define/pron_cmd.go` | modified | `commandCtx` |
| `newCommandCtx` | `cmd/define/command.go` | modified | — |

- **`commandCtx.entry`** — the raw dictionary text of the current word.
  - **Why data and not a capability:** `commandCtx` is deliberately narrower than `deps` — a command may not reach the dictionary or the player. The entry is text the session already has (`session.entry`), the same kind of thing as `lang` and `dictName`. `runPron` must own the inference because it owns the user-facing message; putting it in the loop would split the decision from its explanation.

**ARCH-MOCK.** No new external dependency. The inference reads text the `Dictionary` seam already returns, so the existing `fakeDictionary` corpus is the fixture set and `TestFixturesMatchLiveDictionary` remains its live half.

---

## Task 1: the inference, pure

**Files:** create `cmd/define/origin.go`, `cmd/define/origin_test.go`.

**Contract.** `OriginLanguage(e Entry) (store.Lang, string, error)`:

- no `ORIGIN` section, or one naming nothing in `originLanguages` after masking → error saying so;
- exactly one modern language → its code and the name as written;
- more than one → error naming the candidates, per the operator's "error out if you can't determine".

**The table test's cases come from the COMMITTED corpus**, so the fixtures are already captured and already conformance-checked against the live dictionary. Each is here because it fails a plausible simpler rule:

| fixture | ORIGIN, abbreviated | expect | kills the rule |
|---|---|---|---|
| `jalapeño` | `from Mexican Spanish (chile) jalapeño` | `es` | a qualified name must still match |
| `mesa` | `Spanish, literally 'table', from Latin mensa` | `es` | a chain: first-named wins, Latin masked |
| `concrete` | `from French concret or Latin concretus` | `fr` | same, with `or` rather than a comma |
| `parrot` | `probably from dialect French perrot` | `fr` | a hedge and a modifier before the name |
| **`even`** | `Old English efen …, of Germanic origin; related to Dutch even` | **declines** | **D0** — whole-section search says Dutch |
| **`read`** | `Old English rǣdan, …; related to Dutch raden and German raten` | **declines** | D0, with two cognates |
| `bank` | `Middle English: from Old Norse bakki, of Germanic origin` | declines | `Germanic` is a family, not a language |
| `content` | `via Old French from Latin contentus` | declines | D2 — search-before-mask says French |
| `ephemeral` | `from Greek ephēmeros` | declines | **D3** — bare Greek is ancient |
| `a priori` | `Latin, 'from what is before'` | declines | the most common shape in the corpus |
| `quokka` | `from Nyungar kwaka` | declines | a real modern language absent from the map: decline, do not guess |
| `gaslighting` | `1960s: see gaslight (verb)` | declines | an ORIGIN naming no language at all |

- [ ] **Step 1:** write the table; run it; watch it fail undefined.
- [ ] **Step 2:** implement mask-then-search.
- [ ] **Step 3:** a fuzz or property case over arbitrary section text — this reads dictionary prose, so it must not panic on empty, non-UTF-8, or a 10 KB `ORIGIN`.
- [ ] **Step 4:** commit.

---

## Task 2: `/pron` uses it

**Files:** `cmd/define/pron_cmd.go`, `cmd/define/command.go`, `cmd/define/repl.go`, `cmd/define/replraw.go`; tests in `cmd/define/pron_cmd_test.go`.

**Contract.**

- `parsePronArgs(nil)` returns `("", nil)` — no argument is now a request to infer, not a usage error. Two arguments stay an error.
- `commandCtx` gains `entry string`; `newCommandCtx` fills it from the session, and both loops already hold `sess.entry`.
- `runPron` with no language calls `OriginLanguage`. On success it prints `<word>: ORIGIN says French` and replays; on failure it prints the reason and exits 2, naming `/pron fr` as the override.

**The report is not decoration.** A silent inference cannot be audited, and this repo's rule is that a record has to be true. It is also what makes `piano`'s contested case visible instead of decided behind your back.

- [ ] **Step 1:** tests, all on committed fixtures — bare `/pron` on `concrete` replays in French and says `ORIGIN says French`; bare `/pron` on `read` errors because its only languages are cognates; bare `/pron` on `gaslighting` errors because nothing is named; `/pron fr` still overrides; `/pron` with nothing looked up still says so.
- [ ] **Step 2:** run; fail.
- [ ] **Step 3:** implement.
- [ ] **Step 4:** commit.

---

## Task 3: pin that D1 is untouched

**Files:** `cmd/define/main_test.go`.

**Contract.** An ordinary lookup — no `-pron`, no `/pron` — of a word whose `ORIGIN` names a modern language requests ONLY the session's language. `jalapeño` is the case, and it is already in the corpus: `ORIGIN from Mexican Spanish`, and `jalapeño_es_es` is a live 200 — so a regression toward automatic inference would be invisible to a test that only checks the word plays.

- [ ] **Step 1:** write it; verify it reddens by making `defineOnce` infer.
- [ ] **Step 2:** commit.

---

## Task 4: docs

**Files:** `README.md`, `atlas/define.md`, `cmd/define/voice.go` (`pronHelp`), `cmd/define/main.go` (`fs.Usage`).

- [ ] `pronHelp` says a language is optional and where the default comes from. It is the single source both docs already consume via `TestDocsQuoteThePronHelp`, so the docs follow mechanically.
- [ ] **The `/pron` COMMAND's argument rule goes in the `commands` registry summary**, not hand-written prose. `pronHelp` documents the `-pron` FLAG; the command is a different surface, and `#31` made the atlas's command table a derived consumer of `commands` pinned by `TestDocsQuoteTheCommandList`. The summary is what makes the rule swept.
- [ ] The atlas records D2–D5 — especially that bare `Greek` is ancient and that the table is accepted here while `ParseLang` refuses one, since that asymmetry is the thing a reader will trip on.
- [ ] Verify the additions by grep rather than asserting them.

---

## Verification before close

```bash
go test ./...                             # the WHOLE module
go test -tags conformance ./cmd/define/   # unsandboxed, so the NOAD rows run
```

Then in a scratch directory: `arrondissement` + `/pron` (French, and says so); `read` + `/pron` (errors — its Dutch and German are cognates); `granulate` + `/pron` (errors, no ORIGIN); `arrondissement` + `/pron it` (still overrides, falls back to English and says so).

**Done-when coverage.** Per `#29`'s rule — *pinned only by a NAMED TEST observed red when its wiring is removed* — every cell names a test and the removal that reddens it:

| # | Done-when | pinned by | red when |
|---|---|---|---|
| 1 | bare `/pron` infers and says which | `TestPronInfersTheOriginLanguage` | `runPron` stops calling `OriginLanguage` |
| 2 | errors with the reason | `TestOriginLanguage` table + `TestPronReportsWhyItCannotInfer` | the cognate cut is removed — `read` then infers Dutch |
| 3 | historical stages excluded by category | `TestOriginLanguage`'s Old French / Latin / Greek rows | `historicalStages` is emptied |
| 4 | `/pron fr` still overrides | `TestPronReplaysOnceAndLeavesNoMode` (existing) | the argument path is removed |
| 5 | the table is a recorded decision | atlas prose + `TestDocsQuoteThePronHelp` | the marked span is edited away |
| 6 | D1 untouched | `TestAnOrdinaryLookupNeverInfersTheOrigin` (Task 3) | `defineOnce` infers |
| 7 | extraction is pure and table-tested | `TestOriginLanguage` runs with no dictionary | — |

**Close:** single pass, plain checkboxes, no `Mx`.

## Revisions

### 2026-08-29 — plan-quality round 1

**PQ-1 (Critical) reshaped the rule and then simplified it.** The plan searched
the whole `ORIGIN` section, which infers from COGNATE clauses: `bring` is *"Old
English bringan, of Germanic origin; related to Dutch brengen and German
bringen"* — an Old English word with no source language, from which the rule
would have inferred German. That is `#29`'s D1 failure by another road. Cognates
are cut first now (D0), and `Germanic` is excluded as a family.

Measuring the fix dissolved a branch the plan had: `casque`, `knout`, `ballet`
and `mesa` are borrowing CHAINS, not ambiguities, and NOAD's convention is that
the first-named source is the immediate one. After cutting cognates, nothing is
left that needs an ambiguity error.

**PQ-2** — `pronHelp` documents the FLAG; the command's argument rule is a
different surface and moves into the `commands` registry summary, which `#31`
made a derived consumer pinned by `TestDocsQuoteTheCommandList`.

**PQ-3** — the plan's fixtures (`croissant`, `ballet`, `granulate`) are not in
the captured corpus. Redrawn from committed fixtures, which cover every case
including the two that matter most: `even` and `read` for cognates, `ephemeral`
for bare Greek. No new capture, and the table inherits
`TestFixturesMatchLiveDictionary`'s live check.

**Minor** — D6 records that the `-pron` flag deliberately does not infer.

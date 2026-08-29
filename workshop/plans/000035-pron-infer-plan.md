# `/pron` Origin Inference Implementation Plan (`#35`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `/pron` with no argument plays the source recording for a word whose `ORIGIN` names exactly one modern language, says which language it chose, and errors with the reason when it cannot tell.

**Architecture:** One pure function over the parsed entry — `OriginLanguage(Entry)` — plus two measured tables and a mask-then-search order. `#29` already owns everything downstream: the command records a language, the loop replays it, `utteranceFor` builds the voice. This issue only changes where the language comes from when the user does not supply one.

**Tech Stack:** Go 1.26, stdlib. Tests: `go test ./...` and `-tags conformance`.

---

## Decisions

**D1 — this does not reopen `#29`'s rejection of AUTOMATIC origin audio, and a test says so.** `#29` measured that NOAD writes `ORIGIN French` identically for `arrondissement` and `police`, and that the CDN serves `police_fr_fr` at 200. That argument is about inference running on EVERY lookup. `/pron` is opt-in per word and reports its choice, so a wrong inference is one you asked for, on one word, and can see. The Done-when pins it with a test rather than a sentence: an ordinary lookup of a French-origin word must still request only the session's language.

**D2 — mask historical stages BEFORE searching for modern ones, and the order is load-bearing.** Measured over 300 sampled entries: `Old French` occurs **11** times against `French`'s **8**. Searching for `French` first would call the majority case French. So: replace every historical stage in the ORIGIN text, then look for modern names in what remains.

**D3 — bare `Greek` is ANCIENT Greek and is excluded, though it has a code.** It is the second most common language word in the sample (6, behind `Latin`'s 19), and NOAD writes `modern Greek` when it means the living language. Including `Greek` would make a large class of ordinary words infer `el` — the same failure `#29`'s D1 rejected, arriving by a different road. `modern Greek` is left unhandled rather than special-cased: it did not occur in the sample, and a rule written for a case nobody has seen is a guess.

**D4 — historical stages are excluded BY CATEGORY, not because the CDN 404s them.** `Latin` has the code `la`, `Old English` has `ang`, `Sanskrit` has `sa`. Filtering on "no recording exists" would be right by accident and would silently start playing something the day Google adds Latin. The reason they are excluded is that a superseded stage of a language is not something a speaker says today — the concept `/pron` offers does not apply to it.

**D5 — the language-name table is accepted where `ParseLang` and `localeFor` refused one, and the difference is whose fact it is.** Those refusals were about what the **CDN and the installed dictionaries** serve — someone else's fact, changing without notice. This table reads **NOAD's editorial prose**, which is stable, small, and ours. It makes no claim about what the CDN has: an inferred language with no recording degrades through `#29`'s fallback exactly as a typed one does. Recorded because the next reader meets three refusals and one acceptance and deserves the distinction.

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

**The table test's cases, all measured 2026-08-29** — these are the fixture rationale, and each is here because it fails a plausible simpler rule:

| ORIGIN text | expect | kills the rule |
|---|---|---|
| `French, from arrondir` | `fr` | — |
| `via Old French from Latin natio(n-)` | error, no modern language | search-before-mask would say French |
| `late Middle English: from Old French, from Latin` | error | the most common shape in the sample |
| `from Mexican Spanish (chile) jalapeño` | `es` | a qualified name must still match |
| `from Italian, from Latin, literally 'labor'` | `it` | first-named wins where Latin follows |
| `early 17th century: from French, from Italian balletto` | error, two named | `ballet` — the operator asked for an error, not a coin flip |
| `either from French, or … Italian piano is not attested` | error, two named | `piano` — genuinely contested |
| `from Greek amarullis` | error | D3: bare Greek is ancient |
| `1970s: from Japanese, literally 'empty orchestra'` | `ja` | no CDN recording, and that is not this function's business |

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

- [ ] **Step 1:** tests — bare `/pron` on a French-origin word replays in French and says so; bare `/pron` on `ballet` errors naming both candidates; bare `/pron` on a word with no ORIGIN errors; `/pron fr` still overrides; `/pron` with nothing looked up still says so.
- [ ] **Step 2:** run; fail.
- [ ] **Step 3:** implement.
- [ ] **Step 4:** commit.

---

## Task 3: pin that D1 is untouched

**Files:** `cmd/define/main_test.go`.

**Contract.** An ordinary lookup — no `-pron`, no `/pron` — of a word whose `ORIGIN` names a modern language requests ONLY the session's language. `croissant` is the case: `ORIGIN … French`, and `croissant_fr_fr` is a live 200, so a regression toward automatic inference would be invisible to a test that only checks the word plays.

- [ ] **Step 1:** write it; verify it reddens by making `defineOnce` infer.
- [ ] **Step 2:** commit.

---

## Task 4: docs

**Files:** `README.md`, `atlas/define.md`, `cmd/define/voice.go` (`pronHelp`), `cmd/define/main.go` (`fs.Usage`).

- [ ] `pronHelp` says a language is optional and where the default comes from. It is the single source both docs already consume via `TestDocsQuoteThePronHelp`, so the docs follow mechanically.
- [ ] The atlas records D2–D5 — especially that bare `Greek` is ancient and that the table is accepted here while `ParseLang` refuses one, since that asymmetry is the thing a reader will trip on.
- [ ] Verify the additions by grep rather than asserting them.

---

## Verification before close

```bash
go test ./...                             # the WHOLE module
go test -tags conformance ./cmd/define/   # unsandboxed, so the NOAD rows run
```

Then in a scratch directory: `arrondissement` + `/pron` (French, and says so); `ballet` + `/pron` (errors naming both); `granulate` + `/pron` (errors, no modern language); `arrondissement` + `/pron it` (still overrides, falls back to English and says so).

**Done-when coverage.** Per `#29`'s rule — *pinned only by a NAMED TEST observed red when its wiring is removed* — every cell names a test and the removal that reddens it:

| # | Done-when | pinned by | red when |
|---|---|---|---|
| 1 | bare `/pron` infers and says which | `TestPronInfersTheOriginLanguage` | `runPron` stops calling `OriginLanguage` |
| 2 | errors with the reason | `TestOriginLanguage` table + `TestPronReportsWhyItCannotInfer` | the ambiguity branch returns a first match |
| 3 | historical stages excluded by category | `TestOriginLanguage`'s Old French / Latin / Greek rows | `historicalStages` is emptied |
| 4 | `/pron fr` still overrides | `TestPronReplaysOnceAndLeavesNoMode` (existing) | the argument path is removed |
| 5 | the table is a recorded decision | atlas prose + `TestDocsQuoteThePronHelp` | the marked span is edited away |
| 6 | D1 untouched | `TestAnOrdinaryLookupNeverInfersTheOrigin` (Task 3) | `defineOnce` infers |
| 7 | extraction is pure and table-tested | `TestOriginLanguage` runs with no dictionary | — |

**Close:** single pass, plain checkboxes, no `Mx`.

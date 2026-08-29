# Curate the Italian Dictionary Implementation Plan (`#31`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `/lang it` answers from the Devoto-Oli rather than silently falling back to NOAD and handing an Italian learner an anglicised English entry.

**Architecture:** One row in `curated`, and everything else is the corpus that row obliges. `#23 M2` built the seam so the fixture directory drives the suites — `capturedLanguages` reads `testdata/entries/` — so a captured `it/` enters the parser suite, the no-data-loss invariant and fixture conformance without wiring. What does NOT come free is the own-language conformance check, which is Spanish-hardcoded, and the raw-notation ratchet, which is English-only by construction.

**Tech Stack:** Go 1.26, stdlib. `testdata/capture.sh` (bash + `capture.py`, must run unsandboxed). Tests: `go test ./...` and `-tags conformance`.

**Level of detail:** contracts and the measured tables, not pre-written bodies — `#29`'s plan gate found the latter is a lossy pre-image of the diff.

---

## Decisions

**D1 — Italian only, and the measurement is why.** Over five common words each, the parser finds 245 senses in `it.Devoto-Oli` (NOAD: 153) but **6** in `fr.Multi` and **5** in `de.DDDSI`, whose longest single undifferentiated blob is 2,811 and 4,404 runes. Devoto-Oli numbers senses `1`/`2` and marks sub-senses `•` — shapes `parseSenses` already knows. French and German need the parser's POS vocabulary and section names, which are English. Split to `#34`; full evidence in `#31`'s `## Revisions`.

**D2 — no runtime claim that Italian has no audio.** Nine CDN probes found no Italian recordings (`#29`), and it is tempting to say so when `/lang it` is set. That would be a closed table of a fact **the CDN owns** — the argument `ParseLang` makes for not enumerating languages and `localeFor` makes for not enumerating locales, and it would go stale the day Google adds Italian. The per-word `no recorded pronunciation` already tells the truth every time and cannot go stale. The measurement is recorded in the ATLAS and README, which are documentation and are allowed to date themselves. **This revises the issue's fifth Done-when**, which asked for a runtime statement.

**D3 — the raw-notation ratchet stays English-only, and that is a pre-existing gap this issue does not widen.** `live_property_test.go` walks `/usr/share/dict/words` and asks `systemDictionary(store.DefaultLang)`; there is no equivalent Italian word list on the host, and `#23 M2` added Spanish under exactly the same condition without one. So Italian joins Spanish as a curated language whose notation is swept only at corpus width, not at dictionary width. Recorded in the atlas as a standing limitation rather than left for the next reader to discover — which is the whole reason it is a decision here rather than an omission.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `curated` | `cmd/define/dictselect.go` | modified — gains the `it` row |
| `chooseDictionary` | `cmd/define/dictselect.go` | unchanged — the row is data it already consumes |
| `monolingualIn` | `cmd/define/dictselect.go` | unchanged — and it is what rejects `OxfordItalian` |

- **`curated`** — the per-language preference list.
  - **Relationships:** 1:N language → identifiers, consumed only by `chooseDictionary`.
  - **DRY rationale:** already the single source; this issue adds a row rather than a mechanism.
  - **Future extensions:** `#34` adds `fr` and `de` here once their entries parse.

**Nothing else is a new entity, and that is the point of the seam.** `#23 M2` made the corpus directory the driver, so the work is data plus the two checks the data does not reach.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `testdata/capture.sh` | `cmd/define/testdata/capture.sh` | modified | `capture.py` → DictionaryServices |
| `fakeDictionary` | `cmd/define/dict_fake_test.go` | unchanged — REUSED | the installed dictionaries |
| `capturedLanguages` | `cmd/define/dict_fake_test.go` | unchanged — REUSED | `testdata/entries/` |

- **`capture.sh`** gains an `it_words` array and an `IT_DICT` identifier, following the file's stated convention: words chosen for **what they prove structurally**, not for vocabulary.
  - **Injected into:** nothing at runtime; it produces the corpus the fake serves.

**ARCH-MOCK.** No new external dependency: DictionaryServices already has `fakeDictionary` behind the `Dictionary` seam, with `TestFixturesMatchLiveDictionary` as its live conformance half. Both pick Italian up from the directory. The one live check that does NOT is `TestSelectedDictionaryAnswersInItsOwnLanguage`, which is Spanish-hardcoded — Task 3.

---

## Task 1: the corpus, chosen for what it proves

**Files:** modify `cmd/define/testdata/capture.sh`; create `cmd/define/testdata/entries/it/*.txt`.

**Contract.** `it_words` + `IT_DICT=com.apple.dictionary.it.Devoto-Oli`, captured through the identifier exactly as `ES_DICT` is — never a name match, because `DCSCopyAvailableDictionaries` returns an unordered SET and a substring like `Ital` matches the BILINGUAL `OxfordItalian` on some runs.

**The words, and why each** (the file documents its corpus this way and the next reader depends on it):

| word | proves |
|---|---|
| `pizza` | the headline case — it exists in NOAD **and** in Devoto-Oli, so the two entries differ and dictionary selection is visible. Spanish uses `mesa` for this; this is its Italian counterpart |
| `ciao` | multi-block `A. inter. … B. s.m.`, a shape English never produces; also carries `ETIMOLOGIA` and `DATA`, section names the parser does not know |
| `casa` | ordinary numbered senses, the common case |
| `parlare` | a homograph-numbered headword (`parlare 1`) whose number precedes the syllabification |
| `acqua` | a domain label (`chim.`) before the gloss |
| `essere` | the largest entry available — the blob ceiling, so a regression in sense splitting shows up as one giant example |

- [ ] **Step 1:** add `it_words`, `IT_DICT`, `mkdir -p entries/it`, and the capture loop, mirroring the Spanish block.
- [ ] **Step 2:** run `bash cmd/define/testdata/capture.sh` **UNSANDBOXED** — `DCSCopyTextDefinition` returns silence, not an error, without real access to `/System/Library/AssetsV2`, and the script's `MIN_BYTES` floor exists because a directory of empty fixtures makes `TestRenderLosesNothing` vacuously green.
- [ ] **Step 3:** `go test ./cmd/define` — the corpus is auto-discovered, so `TestRenderLosesNothing`, the no-data-loss invariant and `TestFixturesMatchLiveDictionary` should now cover `it` **without any wiring**. Confirm they actually ran on it rather than assuming: `-run TestRenderLosesNothing -v` should show an `it` subtest.
- [ ] **Step 4:** commit.

---

## Task 2: the curated row

**Files:** modify `cmd/define/dictselect.go`; test `cmd/define/dictselect_test.go`.

**Contract.** `"it": {"com.apple.dictionary.it.Devoto-Oli"}`.

**The doc comment must be updated, not just the map.** It currently reads *"fr.Multi, it.Devoto-Oli and de.DDDSI are installed here and all strictly monolingual, so each is one line — and each brings its own notation conventions with it."* That sentence is now measured and half wrong: one line is true of the map and false of the reading experience for two of the three. Rewrite it to point at `#34` with the numbers, so the next person to reach for that sentence gets the evidence rather than the invitation.

- [ ] **Step 1:** a `chooseDictionary` unit test — `it` resolves to Devoto-Oli; a metadata set containing only the BILINGUAL `OxfordItalian` (`it>it` AND `en>it`) resolves to **nothing**, which is `monolingualIn` doing its job and the reason "prefer the other Italian book" is not available.
- [ ] **Step 2:** run it, watch it fail.
- [ ] **Step 3:** add the row and rewrite the comment.
- [ ] **Step 4:** run; commit.

---

## Task 3: the own-language conformance check, generalised

**Files:** modify `cmd/define/dict_conformance_test.go`.

**Contract.** `TestSelectedDictionaryAnswersInItsOwnLanguage` is Spanish-hardcoded — `mesa`, `nombre femenino`, `sycophantic`. A second language must not become a second copy of it (ARCH-DRY; `#29` closed two instances of `one-predicate-two-spellings` and the second landed in the commit that fixed the first).

Make it a table whose row is `{lang, shared word, marker the entry must carry, word that must be ABSENT}`:

| lang | shared word | marker | absent |
|---|---|---|---|
| `es` | `mesa` | `nombre femenino` | `sycophantic` |
| `it` | `pizza` | `s.f.` | `sycophantic` |

Both halves matter and the second is the one `#23` was built for: the shared word proves selection took effect, and the absence proves an Italian session does not answer an English word from English.

- [ ] **Step 1:** convert to a table, add the `it` row, run under `-tags conformance` **unsandboxed** (it skips where NOAD is unreachable, which is what hid it at all four of `#29`'s gates).
- [ ] **Step 2:** commit.

---

## Task 4: docs, and the two limitations stated rather than discovered

**Files:** `atlas/define.md`, `README.md`.

- [ ] **Step 1:** the atlas's dictionary section gains Italian, plus **D3's ratchet gap** — the live raw-notation sweep is English-only, so `es` and now `it` are swept at corpus width only. Stated as a standing limitation with its reason (no host word list), not as a TODO.
- [ ] **Step 2:** record the notation table `#30` needs, since it was measured here and `#30` reads it: English IPA always; Spanish none; Italian **syllabification, not IPA** (`(cià·o)`); French none; German real but lossy. The last two are `#34`'s, and belong in the atlas anyway because the measurement is done.
- [ ] **Step 3:** README — `/lang it` in the language prose, and the honest sentence that Italian has **no recordings in this CDN generation**, so a session gives definitions and silence. Per D2 this is documentation, not a runtime claim.
- [ ] **Step 4:** verify the deletions/additions by grep rather than asserting them, then commit.

---

## Verification before close

```bash
go test ./...                                  # the WHOLE module — #29's C-1 was exactly this
go test -tags conformance ./cmd/define/        # unsandboxed, so the NOAD rows actually run
```

Then, in a scratch directory:

```bash
define -lang it pizza        # the Devoto-Oli entry, NOT NOAD's
define -lang it sycophantic  # no entry — correct, and what the mode buys
define pizza                 # still NOAD's
define -lang it ciao         # definition, and silence: no Italian recording exists
ls words/                    # it/ appears only after an it lookup
```

**Done-when coverage.** Per `#29`'s rule — *a Done-when is pinned only by a NAMED TEST observed red when its wiring is removed* — every cell names a test and the removal that reddens it:

| # | Done-when | pinned by | red when |
|---|---|---|---|
| 1 | `/lang it` answers from Devoto-Oli | `TestChooseDictionaryPicksTheCuratedItalian`, `TestSelectedDictionaryAnswersInItsOwnLanguage` | the `it` row is removed from `curated` |
| 2 | a missing curated book still degrades loudly | existing `dictionaryFor` tests | the `!ok` branch stops complaining |
| 3 | Italian has a committed corpus the suites cover | `TestRenderLosesNothing`, `TestFixturesMatchLiveDictionary` | `entries/it/` is emptied — both fail, and `loadFakeDictionary` refuses an empty corpus by design |
| 4 | the notation question is measured and recorded | doc-sync on the atlas section | the atlas span is edited away |
| 5 | Italian's audio absence is stated | **REVISED by D2** — documentation, not runtime. No test; the claim is dated prose in README + atlas, and pretending a test pins it would be the over-claim `#29` closed |
| 6 | the new language does not enter unswept | **D3 states the gap** rather than closing it; `TestRenderLosesNothing` covers `it` at corpus width, which is what is actually true |

**Close:** single pass, plain checkboxes, no `Mx`.

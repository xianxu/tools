# Curate the Italian Dictionary Implementation Plan (`#31`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `/lang it` answers from the Devoto-Oli rather than silently falling back to NOAD and handing an Italian learner an anglicised English entry.

**Architecture:** One row in `curated`, and everything else is the corpus that row obliges. `#23 M2` built the seam so the fixture directory drives three suites — `TestEveryCapturedLanguageLoads`, `TestRenderLosesNothing` (both `dict_fake_test.go`) and `TestFixturesMatchLiveDictionary` — all of which iterate `capturedLanguages`. **Three other suites are English-only and do NOT pick it up**, which is easy to mistake for coverage: `TestRenderLosesNothing` (`invariant_test.go`) uses `testDict(t)`, and `TestRenderLosesNothingOverLiveEntries` plus the raw-notation ratchet walk `/usr/share/dict/words` against `systemDictionary(store.DefaultLang)`. So Italian gets no-data-loss at CORPUS width and nothing at dictionary width.

**`curated` has THREE derived consumers, not two**, and the third constrains the commit order: `TestCaptureScriptUsesTheCuratedDictionaries` (`dictselect_test.go`) compares `capture.sh`'s identifiers against `curated` in BOTH directions. So the script edit and the map row must land in ONE commit — either order alone reddens one of its two loops.

**Tech Stack:** Go 1.26, stdlib. `testdata/capture.sh` (bash + `capture.py`, must run unsandboxed). Tests: `go test ./...` and `-tags conformance`.

**Level of detail:** contracts and the measured tables, not pre-written bodies — `#29`'s plan gate found the latter is a lossy pre-image of the diff.

---

## Decisions

**D1 — Italian only, and the measurement is why.** Over five common words each, the parser finds 245 senses in `it.Devoto-Oli` (NOAD: 153) but **6** in `fr.Multi` and **5** in `de.DDDSI`, whose longest single undifferentiated blob is 2,811 and 4,404 runes. Devoto-Oli numbers senses `1`/`2` and marks sub-senses `•` — shapes `parseSenses` already knows. French and German need the parser's POS vocabulary and section names, which are English. Split to `#34`; full evidence in `#31`'s `## Revisions`.

**D2 — no runtime claim that Italian has no audio.** Nine CDN probes found no Italian recordings (`#29`), and it is tempting to say so when `/lang it` is set. That would be a closed table of a fact **the CDN owns** — the argument `ParseLang` makes for not enumerating languages and `localeFor` makes for not enumerating locales, and it would go stale the day Google adds Italian. The per-word `no recorded pronunciation` already tells the truth every time and cannot go stale. The measurement is recorded in the ATLAS and README, which are documentation and are allowed to date themselves. **This revises the issue's fifth Done-when**, which asked for a runtime statement.

**D3 — Italian's notation is swept at CORPUS width by this issue, and left unswept at DICTIONARY width as a stated limitation.** The first draft of this decision said Italian "joins Spanish as a curated language whose notation is swept only at corpus width". That was false in both halves and the plan gate caught it: `TestNoRawPronunciationNotationSurvives` (`render_test.go`) and `TestRenderLosesNothing` (`invariant_test.go`) both take `testDict(t)`, which is ENGLISH. Italian would have entered swept at neither.

- **Corpus width — CLOSED here**, because it is cheap and measurable. `TestNoRawPronunciationNotationSurvives` becomes a sweep over `capturedLanguages`, the shape its sibling `TestRenderLosesNothing` already has. Measured across 15 Devoto-Oli entries: **0 raw pipes, 0 parsed IPA**, so Italian passes the hard zero rather than needing an exemption. This also repairs two committed comments that already over-claim — `capture.sh` and `rawnotation_test.go` both say that check runs "over the committed corpus" when it runs over English, and capturing `entries/it/` is what would make that phrasing actively misleading.
- **Dictionary width — NOT closed, and that is the decision.** The live ratchet walks `/usr/share/dict/words` against `systemDictionary(store.DefaultLang)`. There is no Italian word list on the host, and `#23 M2` added Spanish under the same condition without one. Recorded in the atlas as a standing limitation covering `es` and `it`, so the next reader inherits it rather than rediscovering it.

**D4 — Italian carries no IPA, and that gets a test rather than a sentence.** `isPronunciation` declines Devoto-Oli's `(cià·o)` / `(pìz·za)` syllabification — correctly, since a syllable break is not a phonetic transcription. `testDictFor(t, "it")` already exists, so the Italian sibling of `TestNonEnglishEntriesCarryNoPronunciationNotation` is ten lines and pins the measured fact `#30` reads.

**D5 — `curated` does NOT gain a display title.** The README names book TITLES ("Larousse *Diccionario General*") while `curated` holds bundle IDENTIFIERS, so no span can be generated from it as the command table is from `commands`. Of the three ways out — add a title field, replace the prose with an identifier table, or pin the LANGUAGES — the third is taken. Adding a human-readable title to production data purely so a doc test can render it inverts the dependency, and it changes the shape every consumer of `curated` reads (`chooseDictionary` iterates `[]string`, and so does `TestCaptureScriptUsesTheCuratedDictionaries`). The drift actually worth catching is *a language curated but not documented*, and pinning language names catches exactly that. The code→name map lives in the TEST, because it is a fact about English prose rather than about the dictionaries.


---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `curated` | `cmd/define/dictselect.go` | modified — gains the `it` row |
| `chooseDictionary` | `cmd/define/dictselect.go` | unchanged — the row is data it already consumes |
| `monolingualIn` | `cmd/define/dictselect.go` | unchanged — and it is what rejects `OxfordItalian` |
| `langHelp` | `cmd/define/voice.go` | new — the `-lang` help, derived from `curated` |

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

- [x] **Step 1:** add `it_words`, `IT_DICT`, `mkdir -p entries/it`, and the capture loop, mirroring the Spanish block — **including the script's closing summary**, whose `echo` counts English and Spanish only and whose `wc -c` line globs `entries/en` and `entries/es`. A summary that under-reports the corpus is how a short capture goes unnoticed, which is the same failure `MIN_BYTES` exists to prevent.
- [x] **Step 2:** run `bash cmd/define/testdata/capture.sh` **UNSANDBOXED** — `DCSCopyTextDefinition` returns silence, not an error, without real access to `/System/Library/AssetsV2`, and the script's `MIN_BYTES` floor exists because a directory of empty fixtures makes `TestRenderLosesNothing` vacuously green.
- [x] **Step 3:** confirm the corpus actually entered the suites rather than assuming it. Three tests iterate `capturedLanguages`, but only ONE emits a per-language subtest — `TestFixturesMatchLiveDictionary` does `t.Run(string(lang))`; `TestEveryCapturedLanguageLoads` and `TestRenderLosesNothing` iterate inline, so grepping their `-v` output for `/it` returns 0 on a perfectly good capture:

```bash
go test -tags conformance ./cmd/define/ -run TestFixturesMatchLiveDictionary -v | grep -c '/it'
```

  Non-zero, run UNSANDBOXED. For the other two, the honest confirmation is removal: empty `entries/it/` and watch them redden.

- [x] **Step 4:** do NOT commit yet — see Task 2.

---

## Task 2: the curated row (SAME COMMIT as Task 1)

**Files:** modify `cmd/define/dictselect.go`; test `cmd/define/dictselect_test.go`.

**Contract.** `"it": {"com.apple.dictionary.it.Devoto-Oli"}`.

**The doc comment must be updated, not just the map.** It currently reads *"fr.Multi, it.Devoto-Oli and de.DDDSI are installed here and all strictly monolingual, so each is one line — and each brings its own notation conventions with it."* That sentence is now measured and half wrong: one line is true of the map and false of the reading experience for two of the three. Rewrite it to point at `#34` with the numbers, so the next person to reach for that sentence gets the evidence rather than the invitation.

- [x] **Step 1:** a `chooseDictionary` unit test — `it` resolves to Devoto-Oli; a metadata set containing only the BILINGUAL `OxfordItalian` (`it>it` AND `en>it`) resolves to **nothing**, which is `monolingualIn` doing its job and the reason "prefer the other Italian book" is not available.
- [x] **Step 2:** run it, watch it fail.
- [x] **Step 3:** add the row and rewrite the comment.
- [x] **Step 4:** run, then commit Tasks 1 and 2 TOGETHER. `TestCaptureScriptUsesTheCuratedDictionaries` errors on a `com.apple.*` in the script that no curated list names AND on a curated identifier the script omits, so either edit alone is a red commit on plain `go test ./cmd/define`.

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

- [x] **Step 1:** convert to a table, add the `it` row, run under `-tags conformance` **unsandboxed** (it skips where NOAD is unreachable, which is what hid it at all four of `#29`'s gates).
- [x] **Step 2:** commit.

---

## Task 4: docs, and the two limitations stated rather than discovered

**Files:** `atlas/define.md`, `README.md`.

- [x] **Step 1:** the atlas's dictionary section gains Italian, plus **D3's ratchet gap** — the live raw-notation sweep is English-only, so `es` and now `it` are swept at corpus width only. Stated as a standing limitation with its reason (no host word list), not as a TODO.
- [x] **Step 2:** record the notation table `#30` needs, since it was measured here and `#30` reads it: English IPA always; Spanish none; Italian **syllabification, not IPA** (`(cià·o)`); French none; German real but lossy. The last two are `#34`'s, and belong in the atlas anyway because the measurement is done.
- [x] **Step 3:** README — `/lang it` in the language prose, and the honest sentence that Italian has **no recordings in this CDN generation**, so a session gives definitions and silence. Per D2 this is documentation, not a runtime claim.

**The README's book list is a hand-maintained restatement of `curated`** ("In English that is the New Oxford American Dictionary … plus Apple Dictionary … in Spanish it is the Larousse"), and Task 4 would make Italian its third unpinned entry. Do NOT just add a clause — that is the `doc-sweep-incomplete` shape `#29` closed twice, most recently by generating the atlas command table from the `commands` registry. Task 5 does the same here.
- [x] **Step 4:** verify the deletions/additions by grep rather than asserting them, then commit.

---

## Task 5: close what the corpus does not close by itself

**Files:** `cmd/define/render_test.go`, `cmd/define/invariant_test.go`, `cmd/define/dictselect_test.go`, `cmd/define/dict_conformance_test.go`, `cmd/define/dict_fake_test.go`, `cmd/define/voice.go`, `cmd/define/main.go`, `atlas/define.md`, `README.md`.

Four checks the captured directory does NOT give for free. Each is a third instance of a family `#29` closed with a mechanism, so each gets a mechanism.

- [x] **`TestNoRawPronunciationNotationSurvives` sweeps every captured language** (D3). It takes `testDict(t)` today, so the hard zero it asserts covers English while two committed comments say "the committed corpus". Convert it the way `TestRenderLosesNothing` is written, then **fix both comments** — `capture.sh` and `rawnotation_test.go` — because a corrected test with stale prose beside it is how the next reader is misled.
- [x] **The Italian no-IPA sibling** (D4): `testDictFor(t, "it")`, assert `ParseEntry(raw).IPA == ""` across the corpus, beside `TestNonEnglishEntriesCarryNoPronunciationNotation`. Measured 0/15, and it pins that `isPronunciation` declines `(pìz·za)`.
- [x] **`TestEverySurfaceNamesEveryCuratedLanguage`** (D5): every key in `curated` is named in the README's dictionary paragraph, via a code→name map local to the test. NOT a generated span — see D5 for why `curated` does not gain a title field.
- [x] **`TestEveryCuratedLanguageHasACorpus`**: every key in `curated` has a non-empty `testdata/entries/<lang>/`. This is the direction nothing checks — `capturedLanguages` derives from the DIRECTORY and guards only `len(out) >= 2`, so DELETING `entries/it/` leaves `en`+`es` and reddens nothing.
- [x] Each verified by removal, per `#29`'s rule: drop the `it` row and both `curated`-derived tests redden; delete `entries/it/` and the corpus guard reddens; empty it and the sweeps redden.

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
| 3 | Italian has a committed corpus the suites cover | `TestEveryCapturedLanguageLoads`, `TestFixturesMatchLiveDictionary`, and **`TestEveryCuratedLanguageHasACorpus`** (Task 5) | `entries/it/` is EMPTIED (`loadFakeDictionary` refuses an empty corpus) — and, because `capturedLanguages` only requires `len >= 2`, DELETING the directory reddens nothing without Task 5's guard. That asymmetry is why Task 5 exists |
| 4 | the notation question is measured and recorded | `TestNonEnglishEntriesCarryNoPronunciationNotation` (D4) for the measured half; the cross-language table is dated prose in the atlas | Devoto-Oli starts emitting a parsed IPA |
| 5 | Italian's audio absence is stated | **REVISED by D2** — documentation, not runtime. No test; the claim is dated prose, and pretending a test pins it would be the over-claim `#29` closed |
| 6 | the new language does not enter unswept | `TestNoRawPronunciationNotationSurvives`, now over every captured language (D3) | `entries/it/` is emptied. The DICTIONARY-width gap stays open and is stated in D3 rather than papered over |
| — | the docs name every curated language | `TestEverySurfaceNamesEveryCuratedLanguage` (D5) | **Italian is removed from the README span while the `curated` row STAYS** |
| — | a curated language always has a corpus | `TestEveryCuratedLanguageHasACorpus` | **`entries/it/` is emptied while the `curated` row STAYS** |

**On those last two removals specifically.** The obvious mutation — delete the
`it` row from `curated` — leaves both tests GREEN, and that is correct
behaviour, not a hole: uncurating a language removes the obligation, so there is
nothing left to check. The removal that pins an obligation has to BREAK it, not
withdraw it. Verified both ways: dropping the row is green, breaking what the row
obliges is red.

**Close:** single pass, plain checkboxes, no `Mx`.

## Revisions

### 2026-08-29 — plan-quality round 1 (PQ-1…PQ-5)

**PQ-1 (Critical) was a naming error with a substantive tail.** The plan said a
captured `it/` enters "the parser suite, the no-data-loss invariant and fixture
conformance" and named `TestRenderLosesNothing` — which takes `testDict(t)`, the
ENGLISH corpus. The claim's substance survives (three suites really do iterate
`capturedLanguages`) but under different names, and the three English-only
suites are now named explicitly so the distinction is legible rather than
inferable. Same class as `#29`'s `rôle`: a fact measured about one thing, cited
about another.

**PQ-3 and PQ-5 are third instances of families `#29` closed**, so both get
Task 5's mechanisms rather than an edit — the README's book list becomes a
consumer of `curated` the way the atlas command table became a consumer of
`commands`, and the corpus gains the row→tree direction that
`TestPlanTableStatusMatchesTheChangeWindow` was built for one artifact over.

**PQ-2 and PQ-4** corrected in place: the confirmation command now names tests
that can produce an `it` subtest, and Task 1 covers `capture.sh`'s closing
summary, which counts `en` and `es` only.

### 2026-08-29 — implementation notes

- **`TestEverySurfaceNamesEveryCuratedLanguage` passed the moment it was written**, and
  for the wrong reason: `#29` had left "Italian and Japanese have no recordings
  in this CDN generation" in the `-pron` section, so free-text containment over
  the whole README was satisfied while the dictionary paragraph still listed two
  languages. Scoped to a `<!-- curated-languages -->` span, it went red until the
  docs caught up. A check satisfied by an unrelated sentence certifies nothing.
- **The `capture.sh`/`curated` guard could not express a hyphen.** Its identifier
  regex was `com\.apple\.[A-Za-z0-9._]+`, so `…it.Devoto-Oli` truncated to
  `…it.Devoto` and it then reported the pair mismatched in BOTH directions.
  Latent, not absent: every curated identifier happened to be hyphen-free until
  this issue, while installed ones (`nl-en.oup`, `zh_TW-en.DrEye`) are not.
- **Widening `TestNoRawPronunciationNotationSurvives` made two committed comments
  TRUE** rather than requiring their repair, which is why D3 chose to widen the
  check rather than narrow the claim.

### 2026-08-29 — close review rounds 1 and 2

**Round 1 (BR-1…BR-5).** Three were mine. I anchored the Italian notation test on
a mid-comment line, so its doc block was appended to the Spanish test's and BOTH
were misdocumented; the review also said the two should have been one table, so
merging fixed both, with the REASON per row because Spanish writes nothing
(phonemic orthography) while Devoto-Oli writes syllabification. The STRONG
no-data-loss invariant — exact alnum counts, which catches insertion as well as
loss — was English-only while its weaker sibling already swept every language, so
a new corpus was checked only by the form that detects drops.

**Round 2 named the real finding, and it was about a claim I made rather than a
site I missed.** Commit `19b5ea1` said "every surface that enumerates the curated
languages derives from `curated`" — a CLASS claim — while fixing the three sites
I happened to know about. The review enumerated **six**, two wrong at HEAD:

| # | site | was |
|---|---|---|
| 1 | the live own-language table | no cross-check, so `#34` could add `fr`/`de` and acquire no live check at all |
| 2 | the order-independence test's `{"en","es"}` | Italian unchecked for a requirement that is real — the API returns a SET |
| 3 | `installedOnThisMachine()` | never gained the Italian books, which is why the Italian case re-declared them locally |
| 4 | "Three curated languages as of `#31`" | a COUNT four lines outside the guarded span |
| 5 | "`es` and `it` … six fixtures each" | **wrong** — `entries/es/` holds five |
| 6 | the `-lang` flag registration | derivation delivered but unpinned; reverting it to a literal was invisible to `go test ./...` |

All six closed, 4 and 5 by naming the list instead of counting it — the lesson
already on file as *"Give a count one producer, or delete the count"*, which is
the second time this repo's own written rule did not stop the instance.

**And the class fix's own mechanism had a hole:** the doc guard used
`strings.Contains`, so renaming the atlas row to `Italiano` kept it green. It is
word-boundary now, and the near-miss is reproduced as the removal check.

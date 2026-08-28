# Boundary Review — tools#23 (whole-issue close)

| field | value |
|-------|-------|
| issue | 23 — deck grouped by language, one language per --play session |
| repo | tools |
| issue file | workshop/issues/000023-deck-language.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2e929fc56d1edc6b04af61100116a02abb7d9146..b0b2e236984bbee0ffd0c8856def7e02b4b519e0 |
| command | sdlc close --issue 23 |
| reviewer | claude |
| timestamp | 2026-08-28T14:14:26-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M2 half is genuinely good work: `chooseDictionary` is a properly pure decision over metadata with an order-independence pin and three named mutation rows, the cgo seam uses `CFSetGetValues` as the measurement demanded, the fake widened to per-language and the live conformance check covers both the private surface and `mesa`-in-two-languages. Build, vet, gofmt and the full suite are green (`ok cmd/define 94.3s`), and I verified both open prior findings are actually pinned by reverting the fixes — BR-11's prose ratchet reddens when a false learner-model claim is planted in a current-truth section, and BR-12's relocated temp-file test reddens when `Deck()`'s `.yaml` suffix check is removed. What blocks SHIP is not a live user-facing bug: it is that the milestone's own governing rule — *"anything derived from the language BEFORE a switch must be re-derived BY it"*, written into `applyLang`'s doc comment as the class fix for C1 — was violated by the very commit that added the next language-derived member (`d.usage`'s news gate), and that the `comment-contract-drift` family reaches its 7th round with 11 measurable fresh instances because the rule's *symbol/model* half was never mechanised, only its `user-model.` half.

## 1. Strengths

- **`cmd/define/dictselect.go:88` — `chooseDictionary` is pure over a data slice with no CoreServices in it (ARCH-PURE pass).** `dictselect_test.go:20` uses the *measured* installed set rather than an invented fixture, and deliberately places NOAD last so a curation-dropping mutation cannot pass by fixture order. `TestChooseDictionaryDoesNotDependOnOrder` rotates every position — the right response to a CFSet, not a comment about one.
- **`dictselect_test.go:130` documents its own prior weakness honestly** ("the first version used OxfordSpanish, which is not the curated id, so it passed on the ID check and never reached the monolingual one"). A test that records why it used to assert nothing is worth more than one that merely passes.
- **`dict_darwin.go:118` — status 1 vs 3 kept distinct** ("this word is not Spanish" vs "the Spanish dictionary is not installed"). That distinction is the difference between the Done-when row and a lie, and it is stated at the C boundary where it is cheapest to get wrong.
- **`cmd/define/store/atomic_internal_test.go:25`** — the BR-12 fix is real: paths come from `y.wordsDir()`/`newTempFile()`, and the fixture body was made *valid* so the suffix check is the only thing under test. I reverted the check and it went red with `Text:bad` in the deck.
- **`workshop/lessons.md:2067-2154`** — four rules extracted, each naming the mechanism rather than the incident (§4 satisfied properly).

## 2. Critical findings

None.

## 3. Important findings

**I. `cmd/define/command.go:388` — `applyLang` does not re-derive `d.usage`, so the D6 news gate holds only at the boundary.** `newsFeedFor` is applied once in `openStore` (`main.go:316`) and `sessionUsage` (`main.go:250`). `applyLang` re-derives `d.lang`, the deck triple, `opt.voice`, `d.dict` and `*vocPtr`, and explicitly excludes `d.usage` as "not language-scoped" — a comment that was true before this range and false after it, since M2 made the feed's presence a function of the language. Verified with a scratch test: after `applyLang(&d, &opt, "es", …)` the session still holds the English `cachingFeed` (`bs2.news != nil`). The reverse is worse in a different way: a session started in `es` keeps `news == nil` after `/lang en`, silently losing the feed for the rest of the process.

Not Critical only because `UsageSource.Usages` has **no production caller at HEAD** — `d.usage` is plumbing for `#10`. It becomes a live defect, silently, the moment that wiring lands.

*Class fix (2nd in family — do not just add a line to `applyLang`):* the enumeration-in-a-doc-comment did not survive one milestone. Make the language-derived set *structural*: extend the single `newDeck` builder in `openStore` to build everything that is a function of the language (including the gated usage source) and have `applyLang` call exactly that builder, so a new member cannot be constructed at the boundary without also being switched. Pin it at `vocab_test.go:405`, which already asserts `d.history` identity across the switch and is the natural home for "and `d.usage`'s feed followed".

**II. `comment-contract-drift`, 7th finding — the rule's SYMBOL/MODEL half is still unenforced, 11 live instances.** Both ratchets count exactly one string, `"user-model."`. The rule they were written for binds "a runtime artifact's filename **or a symbol the code owns**", and the second half has now recurred four times (round 1 `deckDeps`, BR-6 `MigrateFlatDeck`, and now two more). Measured at HEAD:

| # | site | claim | truth |
|---|---|---|---|
| 1 | `dict_darwin.go:296` | "the nine symbols this file resolves" | three (`dcs_resolve`) |
| 2 | `dict_conformance_test.go:69` | "Nine undocumented symbols" | the test checks three |
| 3 | `atlas/define.md:1145` | "The nine symbols are private" | three |
| 4 | `workshop/projects/define-learn.md:464` | "The nine private … symbols are `dlsym`'d" | three |
| 5 | `vocab.go:160` | "warnTo is **the one place** the `define: ` prefix … is written" | `dict_darwin.go:359` is a byte-identical second (**ARCH-DRY**) |
| 6 | `README.md:331-333` | stranded pre-M2 paragraph: "enabling the Chinese dictionaries will return entries this tool does not format" | false on the curated path; also duplicates the paragraph 4 lines above |
| 7 | `atlas/index.md:9` | "define — **NOAD** word lookup" | the dictionary follows the language now |
| 8 | plan `:116`,`:132` | `dictChoice` | absent (`dictMeta`) |
| 9 | plan `:147`,`:153` | `dcsDictionaries` | absent (`installedDictionaries`) |
| 10 | plan `:279` **ticked `[x]`** | "assert all nine symbols resolve" | asserts three |
| 11 | plan D6 "Bonus" | "today a Spanish lookup pays a live Google News request per word" | `Usages` has no production caller |

*Class fix:* mechanise the symbol half, which is cheap because the plan already writes the enumeration down — one test asserting every `Name` cell in a plan's Core-concepts table resolves to a declared identifier at the stated path would have caught 8, 9 and both prior recurrences. For the "nine", give the count one producer (a `dcsSymbols` slice the resolver and the docs both read) rather than four hand-typed copies. Do **not** fix instances 1–11 individually.

**III. `dict_darwin.go:327` — `systemDictionary`'s two fallback branches have no automated test on any platform.** The Done-when row *"the seam FALLS BACK to today's NULL behaviour if any is missing"* and plan Task 9 Step 4 are both ticked on a manual misspelling experiment; `noadDictionary` and `everyActiveDictionary` are referenced from no non-conformance test. This is ARCH-PURE (the three-outcome policy and its warning text live inside the darwin-only cgo file, fused to `installedDictionaries()`'s IO) and ARCH-MOCK (the *selection* seam has no fake, so no test can run the production selection flow end to end — precisely what the M1 review sidecar warned about at `000023-deck-language-m1-review.md:289`). Extract `dictionaryFor(installed []dictMeta, lang) (ids []string, name string)` as a pure function and have the cgo shell be `installedDictionaries()` + that call; the three outcomes then unit-test on any platform. Worth doing now: I measured this machine's shell context returning **one** dictionary (`com.apple.dictionary.Wikipedia`), so `./define -lang es mesa` takes the untested fallback branch on every run here.

**IV. `atlas/repo-guards.md` — new guard surface undocumented (docs gate).** The page is the atlas catalogue of repo guards and was updated for `RuntimeFiles`, `TestRuntimeFilePatternsCoverWhatWeWrite` and `legacyRuntimeFilePaths`, but never names `TestRuntimeArtifactNamesAreSpelledOnceInSource` or `TestProseDoesNotSpellStaleRuntimeArtifactNames` — the two guards `lessons.md` calls this range's class fix, and the two a future contributor most needs to find before adding a doc line. Cheap: one table row each plus the records-vs-current-truth scope rule.

## 4. Minor findings

- `dict_darwin.go:270` — `lastErr` is overwritten per iteration, so a status-3 (dictionary vanished) on `NOAD` followed by a status-1 on `AppleDictionary` reports `ErrNoEntry`; that is exactly the collapse `dcs_lookup_in`'s own comment says must not happen. Keep the first non-`ErrNoEntry` error.
- `dict_darwin.go:189` — `installedDictionaries` calls `dcs_describe(i)` N times, each re-copying the CFSet and indexing position `i` across N *distinct* copies. The file's own comment says the order is unspecified; if two copies ever enumerate differently the result has duplicates and gaps. Copy the set once and describe all N from it. (Perf is not the concern — measured 46µs warm.)
- `testdata/capture.sh:60` captures the English corpus through the NULL search while production selects the curated ids, so capture path ≠ production path for `en`. Detectable only because `TestFixturesMatchLiveDictionary` compares them; worth passing the curated ids to `capture.py` so the two agree by construction (ARCH-MOCK).
- `workshop/projects/define-learn.md:457` carries `**actual:** see the issue's close` with `**closed:** 2026-08-28` already set — the deferral is honest (BR-8's lesson applied) but the portfolio view holds no number until `sdlc close` fills it.

## 5. Test coverage notes

- Green: `go build ./... && go vet ./... && go test ./...` all pass; `gofmt -l ./cmd/` empty.
- Mutation-verified by me: BR-11's prose ratchet reddens on a planted false claim in a non-record section (and correctly ignores one appended after `## Log` — the `currentTruthOnly` shape rule works); BR-12's `TestDeckIgnoresInterruptedWrites` reddens when `Deck()`'s `.yaml` check is removed.
- Gaps: `systemDictionary`'s fallback branches (III); `d.usage` after a `/lang` switch (I); `d`-in-`--play` deletion is argued to ride `d.deck` but only `--forget` is asserted per-language.

## 6. Architectural notes

- **ARCH-DRY — flag.** `dict_darwin.go:359`'s `warnf` duplicates `vocab.go:165`'s `warnTo` exactly, in a package that consolidated that shape once and documented it as "the one place". Also note it is a package-level function declared in a `//go:build darwin` file — a non-darwin caller would break the build.
- **ARCH-PURE — mixed.** `chooseDictionary`, `parseLangPairs`, `localeFor`/`voiceFor` are clean pure cores with unit tests and no IO. `systemDictionary` is the exception (finding III): policy and warning text inside the cgo shell.
- **ARCH-PURPOSE — flag.** Shadow-sweep over "the language in effect": deck/capture/vocab ✓, `opt.voice` ✓, `d.dict`/`d.dictName` ✓, editor `voc` ✓, news-feed gate ✗. Five of six derive from the switch; the sixth derives only at the boundary. That is the instance-vs-class distinction this issue has already paid for twice.
- **ARCH-MOCK — partial pass.** `fakeDictionary` is per-language, built from real captures, with an on-demand live conformance check that covers the private surface *and* the curated policy *and* `mesa` in both languages — genuinely good. Missing: a fake at the `installedDictionaries` boundary, so production and test do not share the selection seam (III).

## 7. Plan revision recommendations

Append one `## Revisions` entry to `workshop/plans/000023-deck-language-plan.md` — M2's design changed materially with no entry, which AGENTS.md §1 requires:

- **Core concepts corrected.** `dictChoice` → `dictMeta` + `langPair`; `dcsDictionaries` → `installedDictionaries`. Record that this is the **third and fourth** time this plan has named an entity the tree does not have (I4 `deckDeps`, BR-6 `MigrateFlatDeck`), and that the rule the plan already states — *a rename in code sweeps every prose restatement in the SAME commit* — is unenforced because both ratchets check only `"user-model."`.
- **Task 8 Step 1 — curation became an ordered LIST.** The ticked step still states `en`→`NOAD`; record why (selecting NOAD alone lost `iPhone`/`iPad`/`MacBook`, caught by `TestFixturesMatchLiveDictionary`) so the ticked expectation matches the shipped one.
- **Task 10 Step 1 — "all nine symbols" is not what shipped.** The seam resolves three (`DCSCopyAvailableDictionaries`, `DCSDictionaryGetIdentifier`, `DCSDictionaryGetLanguages`); correct the step and the Risks paragraph at `:285`.
- **D6 — the "Bonus" paragraph is not measurable at HEAD.** `UsageSource.Usages` has no production caller, so no Spanish lookup pays a Google News request today. Restate it as the cost `#10` will pay if the gate is absent, and add that the gate is applied at the boundary only — which is finding I.

```findings
dispose:
  - id: BR-11
    disposition: addressed
    note: |
      TestProseDoesNotSpellStaleRuntimeArtifactNames binds README/atlas/workshop/projects with a currentTruthOnly shape-based record filter; I planted the exact false sentence at define-learn.md:150 and it went red, and appending it after "## Log" correctly did not. Forget's doc comment is reattached at store/yaml.go:534. Residual symbol-half scope raised as a new finding.
  - id: BR-12
    disposition: addressed
    note: |
      TestDeckIgnoresInterruptedWrites moved to package store (atomic_internal_test.go) deriving paths from y.wordsDir() and newTempFile(); reverting Deck()'s .yaml suffix check turns it red with the half-written word in the deck. store.LangFileName() exported and used at lang_cmd_test.go:159 and :182.
findings:
  - id: new
    severity: Important
    family: language-derived-state-unscoped
    title: |
      applyLang does not re-derive d.usage, so D6's news gate holds at the boundary but not across a mid-session /lang
    detail: |
      2nd finding in this family — do NOT fix the instance by adding one line to applyLang. newsFeedFor is applied only in openStore (main.go:316) and sessionUsage (main.go:250), while applyLang (command.go:388) excludes d.usage as "not language-scoped" — true before this range, false after M2 made the feed's presence a function of the language. Verified with a scratch test: after applyLang(&d,&opt,"es",...) the session still holds the English cachingFeed (bs2.news != nil); symmetrically a session started in es keeps news==nil after /lang en and silently loses the feed. Latent rather than live only because UsageSource.Usages has no production caller at HEAD (grep: d.usage is referenced only at main.go:182-183); it becomes silent wrong-language data the moment #10 wires it. The class: applyLang's own doc states the generating rule ("anything derived from the language BEFORE a switch must be re-derived BY it") and the very next member added violated it, so an enumeration in a comment is not enforcement. Structural fix: extend the single newDeck builder to construct everything that is a function of the language (including the gated usage source) and have applyLang call exactly that builder, so a boundary-only derivation is unspellable. Pin at vocab_test.go:405, which already asserts d.history identity across the switch.
  - id: new
    severity: Important
    family: comment-contract-drift
    title: |
      The artifact-name rule's SYMBOL/MODEL half is still unenforced; 11 live restatements measured at HEAD
    detail: |
      7th finding in this family — do NOT fix these instances. Both ratchets count exactly one string, "user-model.", while the rule they enforce binds "a runtime artifact's filename OR A SYMBOL THE CODE OWNS". The unmechanised half has now recurred four times (I4 deckDeps, BR-6 MigrateFlatDeck, and two fresh ones). Measured at HEAD - (1) dict_darwin.go:296 "the nine symbols this file resolves"; (2) dict_conformance_test.go:69 "Nine undocumented symbols"; (3) atlas/define.md:1145; (4) workshop/projects/define-learn.md:464 - dcs_resolve resolves THREE. (5) vocab.go:160 claims warnTo is "the one place" the "define: " prefix is written while dict_darwin.go:359 is a byte-identical second (ARCH-DRY; also a package-level func inside a darwin-only build tag). (6) README.md:331-333 is a stranded pre-M2 paragraph duplicating the one four lines above and falsely saying enabling Chinese dictionaries affects formatting, which the curated L to L selection now prevents. (7) atlas/index.md:9 still says "NOAD word lookup". (8) plan :116/:132 name dictChoice, absent. (9) plan :147/:153 name dcsDictionaries, absent. (10) plan :279 ticked, "assert all nine symbols resolve". (11) plan D6's Bonus claims a live Google News request per Spanish lookup, which no code path makes. Class fix, cheap because the enumeration already exists - one test asserting every Name cell of a plan's Core-concepts table resolves to a declared identifier at the stated path would have caught 8, 9 and both prior recurrences; and give the symbol COUNT one producer (a dcsSymbols slice the resolver and the docs both read) instead of four hand-typed copies.
  - id: new
    severity: Important
    family: policy-inside-io-shell
    title: |
      systemDictionary's two fallback branches have no automated test on any platform, and the Done-when row is ticked on a manual experiment
    detail: |
      dict_darwin.go:327 fuses the three-outcome policy and its user-facing warning text to installedDictionaries()' cgo IO, so neither "the private surface is gone" nor "nothing curated matches" is reachable from a test. noadDictionary and everyActiveDictionary appear in no non-conformance test. The Done-when row "the seam FALLS BACK to today's NULL behaviour if any is missing" and plan Task 9 Step 4 are both ticked on a manual symbol-misspelling run. ARCH-PURE (extract dictionaryFor(installed []dictMeta, lang) (ids []string, name string) as pure and leave the cgo shell thin) and ARCH-MOCK (with the metadata source injected, production and test finally share the selection boundary — the standard the M1 review sidecar set at 000023-deck-language-m1-review.md:289 and this milestone did not meet). Not hypothetical: I measured this machine's shell context returning a single dictionary (com.apple.dictionary.Wikipedia), so ./define -lang es mesa takes the untested branch on every run here.
  - id: new
    severity: Important
    family: atlas-lags-new-surface
    title: |
      atlas/repo-guards.md does not name the two artifact-name ratchets this range added
    detail: |
      repo-guards.md is the atlas catalogue of repo guards and was updated for RuntimeFiles, TestRuntimeFilePatternsCoverWhatWeWrite and legacyRuntimeFilePaths, but never names TestRuntimeArtifactNamesAreSpelledOnceInSource or TestProseDoesNotSpellStaleRuntimeArtifactNames — the two guards workshop/lessons.md calls this range's class fix, and the ones a contributor most needs to find before adding a doc line. One table row each plus the records-versus-current-truth scope rule (## Revisions / ## Log / a block carrying **closed:**), so the deliberate scope decision is discoverable outside the test's own comment.
  - id: new
    severity: Minor
    family: comment-contract-drift
    title: |
      selectedDictionary.Lookup lets a later ErrNoEntry overwrite an earlier "dictionary unavailable", the collapse its own C comment forbids
    detail: |
      dict_darwin.go:270 assigns lastErr on every iteration, so status 3 on NOAD followed by status 1 on AppleDictionary reports ErrNoEntry — "this word does not exist in English" when the truth is "the primary dictionary vanished". dcs_lookup_in's comment at :118 states these two must not collapse. Keep the first non-ErrNoEntry error instead of the last error seen.
  - id: new
    severity: Minor
    family: policy-inside-io-shell
    title: |
      installedDictionaries indexes position i across N separately-copied CFSets, whose enumeration order the file itself calls unspecified
    detail: |
      dict_darwin.go:189 calls dcs_describe(i) once per dictionary and each call re-invokes f_copy_available() and CFSetGetValues on a fresh copy. dcs_describe's own comment states iteration order is UNSPECIFIED; if two copies ever enumerate differently the returned slice gains duplicates and drops entries, and a dropped com.apple.dictionary.es.DGLEV silently degrades Spanish to the NULL search. Copy the set once and describe all N from that copy. Performance is not the concern (measured 46us warm).
  - id: new
    severity: Minor
    family: policy-inside-io-shell
    title: |
      The English fixture corpus is captured through the NULL search while production selects the curated identifiers
    detail: |
      testdata/capture.sh:60 captures entries/en through DCSCopyTextDefinition(NULL) — "the host's ACTIVE dictionaries" — while systemDictionary(en) now selects com.apple.dictionary.NOAD and com.apple.dictionary.AppleDictionary. Capture path and production path therefore differ for English; they agree today only because this host's active set happens to match. Detectable (TestFixturesMatchLiveDictionary compares them, and did go red once), so this is a note rather than a defect — but passing the curated ids to capture.py would make the two agree by construction (ARCH-MOCK).
```

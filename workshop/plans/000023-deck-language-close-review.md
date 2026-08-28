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

---

## Re-review — 2026-08-28T14:44:27-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 23 — deck grouped by language, one language per --play session |
| repo | tools |
| issue file | workshop/issues/000023-deck-language.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2e929fc56d1edc6b04af61100116a02abb7d9146..83ecf712cca62e74f9e1c8062d0437cdcafced4b |
| command | sdlc close --issue 23 |
| reviewer | claude |
| timestamp | 2026-08-28T14:44:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've completed the review. Verifying with build, full test suite, four mutation experiments, a cross-platform vet, and a base-vs-HEAD worktree comparison.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The structural work in this range is genuinely good: `langDeps` turns the language-derived set into a type built by one builder, `dictionaryFor` pulls the selection policy out of the cgo shell so all three degradation outcomes unit-test anywhere, the curated-list design is honest about being curated, and the per-language fixture corpus is real captured output. I verified BR-13's fix by mutation (dropping `d.usage` from `applyLang`'s adoption reddens `vocab_test.go:463`), and BR-15, BR-16, BR-18, BR-19 all check out. What blocks SHIP is that two of the seven open findings are not actually fixed — BR-17's cgo fix is **inoperative** (its `case 1` still unconditionally overwrites, which is the exact collapse it claims to prevent), and BR-14's remaining instances are still live — and that BR-14's new class guard has a hole I proved by mutation: `TestPlanTablesNameEntitiesThatExist` is green only because a *stale comment* in `main.go` contains the string `newDeck `. The plan's Core-concepts table names an entity the tree renamed away in this same commit, and the guard built to catch exactly that passes it.

## 1. Strengths

- **`dictselect.go:126` `dictionaryFor`** — the right extraction. `installed == nil` vs `[]dictMeta{}` are kept as distinct states with distinct warnings, and `TestTheTwoFallbacksSayDifferentThings` (`dictselect_test.go:247`) pins that they don't converge. This is the ARCH-PURE move the M1 sidecar asked for, done properly.
- **`dictselect_test.go:150` `TestChooseDictionaryDoesNotDependOnOrder`** rotates the fixture through every position. Given the API returns a CFSet with genuinely unspecified order, this is the assertion that matters, and it is a real property test rather than a restatement.
- **`dictselect_test.go:133`** — the comment admitting the first version of that test used `OxfordSpanish` and therefore never reached the monolingual check is exactly the kind of self-correction that makes a suite trustworthy.
- **`dict_darwin.go:75` `dcs_describe_all`** — one CFSet copy, one pass. BR-18 fixed at the level of the invariant rather than the symptom, with the reason in the comment.
- **`testdata/capture.sh:88`** — the capture path now walks the same curated ids in the same order `selectedDictionary.Lookup` does, and `capture.py` exits 1 on no-entry so the shell's first-hit-wins loop genuinely mirrors production (I checked `capture.py:118-120`). ARCH-MOCK satisfied by construction rather than by coincidence.

## 2. Critical findings

None.

## 3. Important findings

**I-a. `cmd/define/repo_guard_test.go:595` — the new symbol guard passes on a comment, and a live stale row proves it.**
The check is `!declared.Match(src) && !assigned.Match(src) && !strings.Contains(string(src), name+" ")`. That third clause admits any occurrence anywhere in the file, including a comment. The plan's row `| `newDeck` (closure) | `cmd/define/main.go` | new |` (`workshop/plans/000023-deck-language-plan.md:144`) is stale — the tree declares `newLangDeps` — and the guard is green solely because `main.go:269` still says "`newDeck` stays nil on both of these paths". Mutation: rewording that one comment turns the test red naming `newDeck`. Removing the fallback clause entirely flags exactly that one row and nothing else, so no current row needs it.

**I-b. `atlas/define.md:463-476` — the atlas still describes the pre-BR-13 design.**
It says the rebuild goes through `newDeck(lang)` (renamed), that "`usage` reads `usage/`, so re-deriving them would put a second, unloaded `History`…" (false — `usage` **is** re-derived now; only `history` is not), and enumerates the members as "`d.lang`, `opt.voice`, the deck triple and `voc`" with `usage` absent. `atlas/define.md` was edited in `83ecf712` but only in the dictionary section around line 1142. The one current-truth doc describing the invariant this round was about now contradicts the code on that invariant.

**I-c. `cmd/define/command.go:388` / `cmd/define/main.go:597` — the set is a type, but adoption is still hand-enumerated, and the one member outside it has no test at all.**
`d.deck, d.capture, d.vocab, d.usage = ld.deck, ld.capture, ld.vocab, ld.usage` enumerates four fields; `openStore` enumerates the same four again into `storeDeps`. A fifth `langDeps` field is forgettable at both sites, which is the failure the commit message says a struct makes "unspellable". Separately, `d.newDict` is set at **zero** call sites in any test (`grep newDict *_test.go` → nothing), so the entire language→dictionary wiring is unpinned: I deleted both the boundary derivation (`main.go:597-599`) and the `/lang` re-derivation (`command.go:388-390`) and the full `cmd/define` suite stayed green in 94s — while production would then dereference a nil `Dictionary`. That is the milestone's headline Done-when row and the exact class that produced C1 and BR-13.

**I-d. `cmd/define/dict_darwin.go` — pure logic still inside the platform shell, in two places.**
(1) The status-folding policy in `selectedDictionary.Lookup` is untestable where it lives, which is why BR-17's fix shipped inoperative with no test. (2) `parseDictRecords`, `parseLangPairs` and `baseLang` carry doc comments saying "Pure, so the whole cgo boundary's format is testable without CoreServices" — but they sit behind `//go:build darwin` while `TestParseLangPairs` lives in the untagged `dict_fake_test.go:203`. `GOOS=linux go vet ./cmd/define/` now fails with `undefined: parseLangPairs`; I confirmed against a worktree at base `2e929fc5` that it exited 0 there, so this range broke it. `dict_stub.go:13` still claims it "keeps `go build ./...` and `go vet ./...` green off darwin".

**I-e. `cmd/define/dict_darwin.go:200` — `dcsPrivateSymbols` is documented as "ONE producer" but there are two.**
The Go slice names three symbols; `dcs_resolve` at `dict_darwin.go:43-45` hand-writes the same three string literals in C. `TestPrivateDictionarySurfaceStillResolves` walks the Go copy, so adding or renaming a `dlsym` in the C preamble leaves the conformance check green while the resolver needs a symbol nobody verifies. This is BR-7's shape exactly — a guard asserting coverage from a hand-typed restatement — applied to the list the whole M2 OS-version risk rests on.

## 4. Minor findings

- `cmd/define/main.go:129` and `cmd/define/main.go:220` — two stranded doc comments introduced by this commit: `storeDeps`' doc now heads `type langDeps`, and `openStore`'s doc now heads `func newsFeedFor`. Both target declarations are left undocumented. Same shape as BR-10, twice, in the commit that closed it.
- `cmd/define/main.go:435` — `--help`'s opening sentence still says "Looks the word up in macOS's active dictionaries — normally the New Oxford American Dictionary". After M2 that describes the *fallback* path; the normal path is the curated selection. README was updated, the binary's own help was not.
- `cmd/define/dict_conformance_test.go:91` — a missing private symbol routes through `conformance.SkipOrFail`, so by default it skips. The package's own four-class rule (`internal/conformance` doc) puts "the dependency's surface moved" under SHAPE drift, which "ALWAYS fail[s]". The Done-when row says this must say so *loudly*; by default it says so with a skip.
- `atlas/define.md:1171` — the conformance table's `dict_conformance_test.go` row still reads "live lookups still byte-match every fixture"; that file now also holds the private-surface and per-language checks.
- `cmd/define/dict_darwin.go:127-139` duplicates the copy-set-and-scan block from `dcs_describe_all` (ARCH-DRY); a shared `dcs_copy_values` helper would be one source. Not hot-path.

## 5. Test coverage notes

Pure-entity coverage is strong: `chooseDictionary`/`dictionaryFor` have five test functions including order-independence, an every-pair-monolingual case, and both fallback branches, and `parseLangPairs` covers region-stripping and junk. `TestLangSwitchKeepsOneHighlightSetAndItsTheNewLanguages` pins the feed gate in **both** directions, which is the harder half. The gaps are I-c (the dictionary wiring, provably uncovered by mutation), `deps.dictName` (asserted nowhere, so `/lang`'s "from …" line is unverified), `selectedDictionary.Lookup`'s status folding (no test, and the code is wrong), and `noadDictionary` (referenced by no non-conformance test).

## 6. Architectural notes

- **ARCH-DRY — flag (I-a, I-e).** One builder for `langDeps`, `applyVoice` with two callers, and the `warnf` duplicate removed are all right. The exceptions are the symbol list with two producers and the duplicated CFSet scan.
- **ARCH-PURE — flag (I-d).** The selection policy was correctly extracted; the status-folding policy and the record parsers were not, and the parsers' own comments claim a testability they don't have off-darwin.
- **ARCH-PURPOSE — flag (BR-14 residue, I-b).** Shadow-sweep over the single sources: `RuntimeFiles` → all three guards derive ✓; `curated` → resolver and conformance both derive ✓; `dcsPrivateSymbols` → the C resolver does **not** derive ✗; `langDeps` → both consumers hand-enumerate ✗. Three named instances from BR-14's measured eleven are still live in the plan (`:251-253`, `:279`, `:285`).
- **ARCH-MOCK — pass.** The fake widened from entries to a per-language corpus, the capture path now shares the production boundary, and the live checks compare fake against real. The one gap is that `installedDictionaries()` is called directly inside `systemDictionary` rather than injected, so the full seam (including its warning text) can't be driven from a fake installed set — worth doing when I-c's test lands.

## 7. Plan revision recommendations

- **`workshop/plans/000023-deck-language-plan.md:144`** — replace the `newDeck` row with `newLangDeps`, and update D1 (`:61`), D3 (`:84`), Task 4 (`:191`, `:196`) to match. This is the *third* time the plan has named an entity the tree does not have.
- **`:279` and `:285`** — "all nine symbols" / "rests on nine undocumented symbols". Task 10 Step 1 is ticked describing an assertion the code does not make; the resolver needs three. Say "the private symbols" and let `dcsPrivateSymbols` carry the count.
- **`:251-253`** — D6's "Bonus" claims a Spanish lookup pays a live Google News request per word. `UsageSource.Usages` has no production caller (only `*_test.go` sites), so no code path makes that request. State the real benefit (the gate is correctness-first; the saved request is prospective, once `#10` wires it).
- **A `## Revisions` entry** recording that BR-14's ratchet shipped with a fallback clause that admits comment mentions, and that BR-17's fix was inoperative — both are the kind of thing the next round will otherwise re-derive from scratch.

```findings
dispose:
  - id: BR-13
    disposition: addressed
    note: |
      Verified by mutation — dropping d.usage from applyLang's adoption reddens vocab_test.go:463 in both directions. See new finding on the residual hand-enumeration.
  - id: BR-14
    disposition: not-addressed
    note: |
      Class guard landed but has a comment-admitting loophole and three of the eleven measured instances are still live in the plan.
  - id: BR-15
    disposition: addressed
    note: |
      dictionaryFor is pure and all three outcomes plus the empty-set case are unit-tested.
  - id: BR-16
    disposition: addressed
    note: |
      Both ratchets are now table rows in atlas/repo-guards.md with the records-vs-current-truth scope rule.
  - id: BR-17
    disposition: not-addressed
    note: |
      The fix does not fire — case 1 still assigns lastErr unconditionally, so status 3 then status 1 still returns ErrNoEntry.
  - id: BR-18
    disposition: addressed
    note: |
      dcs_describe_all copies the set once and describes all N in a single pass.
  - id: BR-19
    disposition: addressed
    note: |
      capture.sh walks EN_DICTS in curated order; capture.py exits 1 on no-entry so the first-hit-wins loop matches selectedDictionary.Lookup.
findings:
  - id: new
    severity: Important
    family: comment-contract-drift
    title: |
      TestPlanTablesNameEntitiesThatExist passes on a COMMENT mention, and a stale `newDeck` row proves the hole
    detail: |
      9th finding in this family — do NOT fix the newDeck row alone. repo_guard_test.go:595 accepts
      `strings.Contains(string(src), name+" ")` as evidence a symbol is declared, so any mention anywhere
      in the file satisfies it. The plan's Core-concepts row `newDeck` at cmd/define/main.go
      (plan:144) is stale — the tree renamed it to newLangDeps in this very commit — and the guard is
      green only because main.go:269 still says "newDeck stays nil on both of these paths". Mutation
      proof: rewording that one comment turns the test RED naming newDeck; removing the fallback clause
      flags exactly that one row across all active plans and nothing else, so no legitimate row needs it.
      The rule the family keeps failing: a guard may not accept prose as evidence about code. Delete the
      fallback (declared/assigned already cover every current row), then sweep newDeck from plan:144,
      :61, :84, :191, :196, main.go:52, :69, :269 and atlas/define.md:465.
  - id: new
    severity: Important
    family: atlas-lags-new-surface
    title: |
      atlas/define.md still describes the pre-BR-13 /lang design, including the exact claim BR-13 disproved
    detail: |
      2nd finding in this family — the rule is that the atlas is updated in the SAME commit as the
      surface it maps, not the same range. atlas/define.md:463-476 says the rebuild goes through
      `newDeck(lang)` (renamed to newLangDeps), that "usage reads usage/, so re-deriving them would put
      a second, unloaded History beside the one runEditor already Load()ed" — false, usage IS re-derived
      now and only history is not — and enumerates applyLang's members as d.lang, opt.voice, the deck
      triple and voc, with usage absent. The commit edited atlas/define.md but only around line 1142.
      The one current-truth document describing this invariant now contradicts the code on it.
  - id: new
    severity: Important
    family: language-derived-state-unscoped
    title: |
      langDeps is adopted field-by-field at two sites, and d.dict — a member since M1 — has no test at all
    detail: |
      3rd finding in this family — do NOT fix by adding one assertion. The rule the fixes keep half-applying
      is: every member of the language-derived set must be (a) adopted as a WHOLE from the one builder and
      (b) pinned by a test that reddens when its re-derivation is removed. Neither half holds. Adoption is
      `d.deck, d.capture, d.vocab, d.usage = ld.deck, ld.capture, ld.vocab, ld.usage` (command.go:388) and
      the same four enumerated again into storeDeps (main.go:325-329), so a fifth langDeps field is
      forgettable at both sites — embedding langDeps in deps/storeDeps makes it one assignment and closes
      (a). For (b): d.newDict is set at ZERO call sites in any test, so deleting BOTH the boundary
      derivation (main.go:597-599) and the /lang re-derivation (command.go:388-390) leaves the whole
      cmd/define suite green (measured, 94s) while production would dereference a nil Dictionary. That is
      the milestone's headline Done-when row, and it is the same class as C1 and BR-13.
  - id: new
    severity: Important
    family: policy-inside-io-shell
    title: |
      Two pieces of pure logic still live inside the darwin cgo shell; one shipped an inoperative fix, the other broke GOOS=linux
    detail: |
      4th finding in this family — do NOT fix either instance alone. The rule: nothing that is a decision
      or a parse over data may live inside dict_darwin.go's build-tagged shell; dictselect.go is where it
      goes. The enumeration for this file is three items and only one is done. (1) selection policy →
      dictionaryFor, extracted ✓. (2) status-folding policy in selectedDictionary.Lookup — still inline,
      untestable, and consequently BR-17's fix shipped broken with no test to catch it. Extract
      foldLookupStatuses([]int, []string) error as pure. (3) parseDictRecords / parseLangPairs / baseLang —
      their own comments say "Pure, so the whole cgo boundary's format is testable without CoreServices",
      but they sit behind //go:build darwin while TestParseLangPairs is in the untagged dict_fake_test.go:203.
      Measured: `GOOS=linux go vet ./cmd/define/` now fails with `undefined: parseLangPairs`; it exited 0 at
      base 2e929fc5 (checked in a worktree), so this range regressed it, and dict_stub.go:13 still claims
      the stub keeps `go vet ./...` green off darwin.
  - id: new
    severity: Important
    family: runtime-artifact-guard-coverage
    title: |
      dcsPrivateSymbols is documented as the ONE producer, but dcs_resolve hand-writes the same three names in C
    detail: |
      4th finding in this family — do NOT fix by syncing the two lists. Identical shape to BR-7: a guard
      asserting coverage from a hand-typed restatement. dict_darwin.go:200 declares the slice and its doc
      says "ONE producer for the list", while dcs_resolve at dict_darwin.go:43-45 spells
      DCSCopyAvailableDictionaries / DCSDictionaryGetIdentifier / DCSDictionaryGetLanguages again as C
      string literals. TestPrivateDictionarySurfaceStillResolves walks the Go copy, so adding or renaming a
      dlsym in the C preamble leaves the conformance check reporting "all present" while the resolver needs
      a symbol nobody verifies — and this check IS the whole mitigation for M2's stated OS-version risk.
      The rule: a list the code owns has one producer, and where a second language forces a restatement, a
      ratchet asserts the two agree. Cheap here — one test regexing dlsym("...") out of the file's own
      source and comparing the set to dcsPrivateSymbols, in the same shape as the existing ratchets.
  - id: new
    severity: Minor
    family: comment-contract-drift
    title: |
      Two doc comments stranded onto the wrong declaration by this commit's insertions
    detail: |
      main.go:127-135 — storeDeps' doc ("the trio openStore produces") now runs into `type langDeps` with
      no blank line, so godoc attaches it to langDeps and storeDeps is undocumented. main.go:212-231 —
      openStore's doc runs into `func newsFeedFor` the same way, leaving openStore undocumented. Both
      declarations were inserted directly beneath an existing doc comment. Same shape as BR-10, twice, in
      the commit that closed it; a ratchet asserting a doc comment's first word matches the declaration it
      precedes is the mechanical form.
  - id: new
    severity: Minor
    family: comment-contract-drift
    title: |
      --help still says define looks words up in "macOS's active dictionaries", which is now the fallback path
    detail: |
      main.go:435. After M2 the normal path is the curated per-language selection and the whole-active-set
      search is the degradation. README.md:314 was updated for this; the binary's own usage text, which the
      same commit edited to add the /lang paragraph, was not.
  - id: new
    severity: Minor
    family: comment-contract-drift
    title: |
      The private-surface conformance check routes SHAPE drift through SkipOrFail, so "says loudly" is a skip by default
    detail: |
      dict_conformance_test.go:91. internal/conformance's own four-class rule puts an absent external
      dependency in SkipOrFail and "the dependency's surface moved" under SHAPE drift, which "ALWAYS
      fail[s] ... the very thing these suites exist to report". A vanished private symbol is the second
      class, and the issue's Done-when row asks it to say so LOUDLY. Under a plain
      `go test -tags conformance` it prints a skip, which the package doc itself says reads as green.
```

---

## Re-review — 2026-08-28T15:08:22-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 23 — deck grouped by language, one language per --play session |
| repo | tools |
| issue file | workshop/issues/000023-deck-language.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2e929fc56d1edc6b04af61100116a02abb7d9146..2a9112bd2cb1077183336423bec4152dc9ecfd93 |
| command | sdlc close --issue 23 |
| reviewer | claude |
| timestamp | 2026-08-28T15:08:22-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Now I have everything I need.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The substantive half of round 7 landed and holds up under mutation: `foldLookupError` and the three parsers are out of the cgo shell (`GOOS=linux go vet ./...` exits 0, verified in a worktree at both ends of the range), `langDeps` is embedded so a language switch adopts the whole set in one assignment, `d.newDict` is now pinned at both the boundary and the switch (deleting both derivations makes `TestTheDictionaryIsBuiltForTheLanguageAtBothMoments` panic — measured), the plan-table guard no longer passes on a comment (a stale `newDeck` row reddens it by name — measured), and the C-preamble/Go symbol-list ratchet catches an added `dlsym` (measured). `go build ./... && go vet ./... && go test ./...` is green; `gofmt -l` is empty; `go vet -tags conformance ./...` is clean. What remains is prose and one hand-typed list: three of the eight sweep sites BR-20 enumerated are untouched in the file it named, BR-25/BR-26/BR-27 are unchanged from the round that raised them, and `capture.sh` restates the curated dictionary list with nothing keeping the two copies in step — which is the same shape as the finding this range closed one commit earlier. None of it is a production correctness defect, so it does not block the gate.

**1. Strengths**

- `cmd/define/dictselect.go` is the right shape now: every decision M2 makes — narrowing, curation, the two fallbacks, the status fold, the three parsers — is pure, untagged and unit-tested, and `dict_darwin.go:210-219` is genuinely just the IO. `dictselect_test.go:250` even pins that the two fallbacks say *different* things, which is the distinction the cgo layer exists to preserve.
- `TestTheCResolverAndTheGoSymbolListAgree` (`dict_symbols_darwin_test.go:25`) is the honest answer to a copy cgo cannot unify: it compares by reading the source and fails vacuously-loudly at both ends. Mutation-verified.
- `TestTheDictionaryIsBuiltForTheLanguageAtBothMoments` (`lang_scope_test.go:328`) drives `run()` rather than replicating the derivation, and asserts on what the dictionary *returns*, not only which seam was called — the specific weakness its own comment records from the first draft.
- Embedding `langDeps` in `deps` (`main.go:125`, adopted at `command.go:391`) is the correct structural end-state for a set that went stale three times as a comment and once as a hand-copied struct.
- `atlas/define.md:454-500` now explains *why* the set is a type and embedded, and marks the `history` exclusion as a claim with a shelf life checked by an identity assertion. That is the atlas doing its job.

**2. Critical findings**

None.

**3. Important findings**

- **`cmd/define/testdata/capture.sh:82,85` — the curated dictionary list has two producers again.** `dictselect.go:76-79` declares `curated`, and `capture.sh` spells `com.apple.dictionary.NOAD`, `com.apple.dictionary.AppleDictionary` and `com.apple.dictionary.es.DGLEV` a second time, in order, with nothing comparing them. Add or reorder a book in `curated` and the fixture corpus is captured through a set production no longer selects — which is BR-19's defect returning through a different door. `TestFixturesMatchLiveDictionary` cannot cover it: it is conformance-tagged (on demand only) and compares text, so an appended book is invisible. Fix in the shape already in the tree: a test that regexes the identifiers out of `capture.sh` and compares the set to `curated`, exactly like `TestTheCResolverAndTheGoSymbolListAgree` does for the C preamble.
- **The symbol half of the artifact-name rule is enforced for plan *tables* only, and two surfaces stay open.** `TestRuntimeArtifactNamesAreSpelledOnceInSource` reads filenames in non-test Go; `TestPlanTablesNameEntitiesThatExist` reads Core-concepts rows. Neither reads Go *comments* or active-plan *prose*, and both residues are live: `workshop/plans/000023-deck-language-plan.md:288` still says M2 "rests on nine undocumented symbols" (three), and `cmd/define/main.go:42,59,259` still name `newDeck`. `atlas/repo-guards.md:118` says plans are "records wholesale and are not swept at all" while the plan-table guard treats a wrong row as "a lie" — the two rules disagree, and `:288` is in the gap. `currentTruthOnly` already splits a document by shape, so extending the prose ratchet to active plans minus their `## Revisions` is the cheap half.

**4. Minor findings**

- `cmd/define/testdata/capture.py:82-83` — `dictionary_by_id` returns a `DCSDictionaryRef` borrowed from the copied set, then `finally: _cf.CFRelease(dicts)` releases the set before the caller passes that ref to `DCSCopyTextDefinition`. Use-after-release; it works only because CoreServices happens to keep the dictionaries alive. The Go side gets this right (`dict_darwin.go:146-148` releases the set *after* the lookup) — move the release to after `lookup()`, or retain the ref.
- `cmd/define/main.go:117` and `:210` — two doc comments still run into the declaration inserted beneath them, so godoc attaches "storeDeps is the trio openStore produces" to `type langDeps` and the `openStore` doc to `func newsFeedFor`; `storeDeps` (`:135`) and `openStore` (`:253`) are undocumented.
- `cmd/define/main.go:423` — `--help` still says define "Looks the word up in macOS's active dictionaries", which after M2 is the degradation path, not the normal one. README:314 was updated; the usage text was not.
- `cmd/define/dict_conformance_test.go:91` — a vanished private symbol is routed through `conformance.SkipOrFail`, so the check the Done-when row asks to say so "loudly" prints a skip under a plain `go test -tags conformance`. The package's own four-class rule puts "the dependency's surface moved" under SHAPE drift, which always fails.

**5. Test coverage notes**

Coverage is strong and the pins are load-bearing rather than decorative — I confirmed three of them redden under mutation and did not find a test that reasserts its implementation. The one uncovered path is `selectedDictionary.Lookup`'s walk itself (cgo, darwin-only), which is acceptable now that the fold it delegates to is pure and table-tested including the exact regression case. The gap worth naming is the one in Important #1: the fixture-capture path has no automated agreement check with production selection, so `ARCH-MOCK`'s "the fake models what the seam does" rests on a comment.

**6. Architectural notes**

- **ARCH-DRY — flag (Important #1).** `newLangDeps` is one builder with two callers, `applyVoice` one derivation with two, `UserModelName`/`langFileName`/`newTempFile` are single producers that `RuntimeFiles` builds from, and `dcsPrivateSymbols` is now ratcheted against the C preamble. The one remaining second producer is `capture.sh`'s copy of `curated`.
- **ARCH-PURE — pass.** The extraction is complete and verifiable: `GOOS=linux go vet ./cmd/define/` exits 0 at HEAD, `dict_stub.go:16-18`'s claim is true again, and `dictionaryFor`'s two fallback branches — the ones a shell context hits on every run — are now testable on a machine with no dictionaries.
- **ARCH-PURPOSE — pass on the issue, flag on the finding axis.** Every Done-when row is delivered and reachable: `mesa` differs by language through a real seam, `sycophantic` in Spanish is `ErrNoEntry`, the deck/audio/dictionary all follow the mode, and degradation is pure-tested rather than manually demonstrated. The flag is BR-20's answer: its enumerated sweep names eight sites and five were done, which is the instance rather than the class, and Important #2 is where that shows.
- **ARCH-MOCK — pass with a note.** Production and test share one boundary (`deps.newDict`), the fake models per-language dictionary *identity* rather than a flat entry pile, and live conformance exists for both the fixtures and the private surface. The note is the Minor above: that private-surface check skips by default, so the OS-version mitigation is quieter than the issue promises.

**7. Plan revision recommendations**

- `## Risks`, `:288` — replace "rests on nine undocumented symbols" with "the private symbols"; `dcsPrivateSymbols` is the producer and prose should carry no count. This exact line was named in the round-6 sidecar alongside `:279`, and only `:279` was swept.
- Add a `## Revisions` entry for this round recording: BR-20's sweep residue (`main.go:42,59,259`), the `capture.sh`/`curated` second producer, and the decision on whether active-plan prose is current truth or a record — `atlas/repo-guards.md:118` and `TestPlanTablesNameEntitiesThatExist` currently answer that question differently, and the residue lived in the disagreement.

```findings
dispose:
  - id: BR-14
    disposition: addressed
    note: |
      All 11 enumerated instances swept and verified at HEAD; the plan-table ratchet landed and reddens on a stale row under mutation.
  - id: BR-17
    disposition: addressed
    note: |
      foldLookupError keeps the first non-absence error and is table-tested at dictselect_test.go:267 including the status-3-then-1 case.
  - id: BR-20
    disposition: not-addressed
    note: |
      Guard fallback removed and mutation-verified, plan row and atlas swept — but the enumerated sweep of `newDeck` is 3 sites short at cmd/define/main.go:42, :59, :259.
  - id: BR-21
    disposition: addressed
    note: |
      atlas/define.md:454-500 now describes the embedded langDeps design and the usage re-derivation correctly.
  - id: BR-22
    disposition: addressed
    note: |
      langDeps embedded, adopted whole at command.go:391; deleting both newDict derivations reddens TestTheDictionaryIsBuiltForTheLanguageAtBothMoments (measured).
  - id: BR-23
    disposition: addressed
    note: |
      Status fold, parsers and ErrLookupFailed are all off the darwin tag; GOOS=linux go vet ./... exits 0 at HEAD.
  - id: BR-24
    disposition: addressed
    note: |
      TestTheCResolverAndTheGoSymbolListAgree compares the C preamble's dlsym literals to dcsPrivateSymbols; adding a dlsym reddens it (measured).
  - id: BR-25
    disposition: not-addressed
    note: |
      Both stranded doc comments are unchanged — main.go:117 attaches to langDeps, main.go:210 attaches to newsFeedFor.
  - id: BR-26
    disposition: not-addressed
    note: |
      cmd/define/main.go:423 still describes the fallback path as the normal one.
  - id: BR-27
    disposition: not-addressed
    note: |
      dict_conformance_test.go:91 still routes a vanished private symbol through SkipOrFail.
findings:
  - id: new
    severity: Important
    family: runtime-artifact-guard-coverage
    title: |
      capture.sh hand-restates the curated dictionary list, so the capture path and production can diverge silently again
    detail: |
      5th finding in this family — do NOT fix by editing capture.sh. The rule, already
      stated twice in this range and mechanised twice: a list the code owns has ONE
      producer, and where a second language forces a restatement, a ratchet asserts the
      two agree. dictselect.go:76-79 declares `curated`; testdata/capture.sh:82 and :85
      spell com.apple.dictionary.es.DGLEV and (NOAD, AppleDictionary) again, in order,
      with nothing comparing them. Adding or reordering a curated book leaves the fixture
      corpus captured through a set production no longer selects — the same failure BR-19
      fixed one commit earlier, reached by a different route. TestFixturesMatchLiveDictionary
      cannot see it: it is conformance-tagged (on demand only) and compares entry TEXT, so
      an appended book is invisible and a reordered one only shows for words whose entries
      differ. Cheap, and the shape already exists: a test regexing the identifiers out of
      capture.sh's own source and comparing the set to `curated`, mirroring
      TestTheCResolverAndTheGoSymbolListAgree.
  - id: new
    severity: Important
    family: comment-contract-drift
    title: |
      The artifact-name rule's symbol half is enforced for plan TABLES only; Go comments and active-plan prose remain unswept
    detail: |
      13th finding in this family — do NOT fix the two instances. The mechanism that
      landed (TestPlanTablesNameEntitiesThatExist) reads Core-concepts rows; the other
      ratchet reads FILENAMES in non-test Go. Neither reads a Go comment or a plan's
      prose, and both residues are live at HEAD: plan :288 says M2 "rests on nine
      undocumented symbols" (dcs_resolve resolves three, and the round-6 sidecar named
      this very line beside :279, of which only :279 was swept), and main.go:42, :59,
      :259 name `newDeck`, renamed to newLangDeps in this range. The rule that covers
      both: a document or comment read as CURRENT TRUTH may not name a symbol the tree
      does not declare, or restate a count of a set the code owns. Two surfaces make it
      mechanical — (a) add active plans, minus the sections currentTruthOnly already
      strips, to TestProseDoesNotSpellStaleRuntimeArtifactNames's scan set; (b) a symbol
      arm over non-test Go comments. Note atlas/repo-guards.md:118 states plans are
      "records wholesale and are not swept at all" while the plan-table guard calls a
      wrong row "a lie" — the two rules disagree, and :288 lived in exactly that gap, so
      the scope decision has to be settled before the guard can be extended.
  - id: new
    severity: Minor
    family: external-handle-lifetime
    title: |
      capture.py releases the dictionary set before the borrowed ref is used, and the Go side does not
    detail: |
      testdata/capture.py:81-83 returns a DCSDictionaryRef borrowed from the copied
      CFSet from inside a try block whose finally does CFRelease(dicts), so the ref is
      passed to DCSCopyTextDefinition after its owning container is released. It works
      only because CoreServices happens to keep the dictionaries alive. dict_darwin.go
      :146-148 gets the same sequence right — the lookup runs before CFRelease(set) —
      so the two implementations of one boundary disagree on handle lifetime. Move the
      release to after lookup(), or CFRetain the ref before returning it.
```

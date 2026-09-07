# Boundary Review — tools#10 (whole-issue close)

| field | value |
|-------|-------|
| issue | 10 — authored practice items: level-tagged words, and stems the model writes offline |
| repo | tools |
| issue file | workshop/issues/000010-vocab-harvest.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0b8d9930762168cf52f77c5d0864599f678d3b5d..ead497dfad4280ec32e57154fbfcd87edd82f313 |
| command | sdlc close --issue 10 |
| reviewer | claude |
| timestamp | 2026-09-06T14:09:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The whole-issue window delivers both milestones honestly: `--harvest` bands the deck once and forever, authors stems the model writes and options the deck supplies, and every claim the issue makes is backed by a test I could redden by mutation. I verified round 8's three headline remediations by reverting them on a scratch copy — BR-42's tier-report fix, BR-43's pin, and BR-34's third member all go red, so the "fix without a failing test" pattern that produced the family is genuinely closed this round. Nothing here blocks the gate: `go build`, `go vet` under both tag sets, `gofmt` and `go test ./...` are all clean. What keeps it from SHIP is a cluster of open items whose *class-level* deliverables the head commit did not land — BR-41's retired-phrasing registry and BR-44's coverage assertion were both named as "the mechanism, not the lines", and `repo_guard_test.go` is untouched across the entire 10,000-line window — plus one measured material-quality defect (BR-24) that is more than the Minor it has been carried as, one README line that now *denies* a shipped feature, and a store lifecycle question this window created and never answered.

## 1. Strengths

- **`runWithin` is the right shape for the bug it fixes.** `cmd/define/harvest.go:93` makes it impossible to charge the budget without gating on the refusal, and the flag × pass table at `harvest_test.go:262` asserts each cell *entered* its pass before asserting the bound. That second half is what made the previous two `-limit` Criticals invisible; it is the correct generalisation.
- **The checkpoint changed the design, and the code shows where.** `renderAuthorPrompt`'s appositive ban (`harvest_item.go:398-406`) shows three wrong shapes rather than restating "do not define it", and `entailVerdict.Glosses` exists as a separate field precisely because a glossed stem entails perfectly. Both are traceable to a batch that was read, not to a design that was guessed.
- **`wordIndexIn`'s original-string offsets** (`harvest_item.go:481`) make the folded-offset panic *unavailable* rather than merely fixed — there is no folded string for a caller to index. That is the right level to fix a bug at.
- **`storetest/suite.go`** gained 9 rows across the new surface, including the damaged-record, canonical-casing, newest-first and cap contracts, so `Mem` and `YAML` are held to one contract by construction.
- **ARCH-DRY answered rather than assumed:** `harvest_item.go:131-162` records *why* `pickDistractors` is not `play.PickOptions` in four structural points, and `play.ShuffleInts` is exported rather than copied. `noadDomainLabels` now derives from `store.Domains()` with a computed longest-first ordering.

## 2. Critical findings

None.

## 3. Important findings

**`Forget` leaves `facts/` and `items/` behind, and nothing decides, documents or pins that.** `cmd/define/store/yaml.go:689` removes one word file; `store.go`'s `Forget` contract enumerates exactly one exemption ("does NOT remove events"), and `storetest/suite.go:239` asserts deck and events only. This window added two per-word persisted surfaces and no one walked the per-key lifecycle verb against them. Reachable consequence: forget a word because its item was bad, look it up again, re-harvest — `d.deck.Items(c.Word)` at `harvest.go:269` returns the stale item, `skipped++`, and the bad material is unregenerable through any documented path. This is the 4th finding in family `store-contract-unheld-by-suite`, so the deliverable is the class, not the cell: write the cross-product of per-key surfaces (`words`, `usage`, `facts`, `items`) × per-key lifecycle verbs (`Forget`) into the `Store` contract, decide each cell explicitly, and hold it in `storetest`. Measured prevalence at HEAD: 3 surfaces survive `Forget`, 2 of them new in this window.

**BR-41's and BR-44's stated mechanisms did not land.** Both findings said in terms that the deliverable was a mechanism rather than the sites; both were remediated by fixing the sites. `retiredPhrases` (`repo_guard_test.go:1131`) is exactly the registry BR-41 asked for and its own comment says "this map is where that grep gets written down so it runs on every later commit too" — a two-line addition that was not made. `repo_guard_test.go` has zero diff across the whole window. Details in the disposition block below.

## 4. Minor findings

- `stemUsesTheWord`/`blankOut` (BR-24): measured on real deck words — `light`→"The ___house at Portland Head", `set`→"The ___tlement", `run`→"The ___way", `bank`→"___ruptcy". Four of the checkpoint deck's five A1 words hit it. Re-disposed, with the measurement, below.
- README's exit-code table (`README.md:590-592`) declares itself an enumeration rather than a sample and then omits every code this window added: exit `2` for two modes on one line, and all of `--harvest`'s exit-`1` paths. 3rd in family `doc-understates-surface`.
- `store.Bands()` is now wired into `renderBandPrompt` (BR-15 addressed), but unlike the domain half it has no `TestBandPromptCarriesTheClosedBandSet` — reverting the derivation to a hardcoded `"A1, A2, …"` leaves the golden green.

## 5. Test coverage notes

Coverage is strong and, unusually, mutation-verified rather than asserted. The rig refuses `harvestRig(1)` outright because three M2 pins were once unfalsifiable that way — a good structural answer. Gaps that remain: the conformance row at `harvest_conformance_test.go:72` falls back to the bare shape *silently* if `dict.Lookup` errors (all 8 words are in the committed corpus today, and `TestBandingIsStable` logs `gloss %t` but asserts nothing), which is the same drift BR-11 was raised for, one guard short. And the two unreachable `errBudget` arms (BR-44) show there is no per-block reachability check over `harvest*.go` — the mechanism that would also have caught BR-38's counter.

## 6. Architectural notes

- **ARCH-DRY** — pass. `noadDomainLabels` derives, `play.ShuffleInts` is shared not copied, `seedFor` is reused. Flag: `optionsPerItem = 3` (`harvest.go:54`) restates `play.maxOptions - 1` in a comment across a seam that already exports two helpers (BR-39, open).
- **ARCH-PURE** — pass. `senseFacts` split out of `wordSense` so the judgement is table-testable with no dict fake; `pickDistractors`, `topicSpread`, `agreement`, `prune`, `parseLearnerBand` all test without IO. Core-concepts table verified row by row — every entity exists at its stated path with its stated status.
- **ARCH-PURPOSE** — flag. Decision 6's shadow-sweep passes: `--reflect` writes through `ParseBand`, `level:` is frontmatter, authoring reads back through the same type, no hand-maintained restatement survives. But BR-41 and BR-44 are the axis failing: a finding that names the class was answered at the instance, twice, in the commit that closes the issue.
- **ARCH-MOCK** — pass. `llmtest.Fake` is wire-level, so production and test share the boundary, and all four tasks now carry live conformance rows in both directions (the veto must reject `obsequious` *and* pass `quokka`). The stem row asserts a *rate*, not a never, which is the honest shape.
- **ARCH-CONSTRAINTS** — pass. `-limit` bounds calls structurally, `agreementMaxRounds` bounds the other end, `ItemCap` bounds disk. `runAuthoring` re-reads the deck and every word's facts a second time per run — O(2N) file reads against a declared scale of "a few thousand" — negligible beside 200 model calls, noted not flagged.
- **ARCH-SECURE** — pass, and well argued. `WordFacts` and `Items` re-parse on the way *out* with the reasoning written down (`yaml.go`: "the README documents these as inspectable, so hand-editing is an invited workflow"), a damaged record degrades visibly to unharvested rather than substituting a fabricated band, and `oneLine` neutralises in one pass over the struct rather than per-field.
- **ARCH-ORDER** — pass. The interrupting event that matters is process death mid-batch, and it is handled by construction: each word is written atomically, the cache is the progress marker, and a budget exhausted mid-item **abandons** the item rather than writing it short (`harvest.go:336-343`) because a later run would skip a word that has material. That is the ordering reasoning this lens asks for, written down and pinned.

## 7. Plan revision recommendations

- `workshop/plans/000010-vocab-harvest-plan.md:456` — the sweep block says "22 properties" over a list I counted at 24 entries, and records neither the mutation nor the named test per row, which the block's own paragraph two below calls the failure mode. Re-derive the count from the list and add the mutation + test name per row, or say why the prose form is right.
- `workshop/plans/000010-vocab-harvest-plan.md:492` — `- [ ] **The generated batch, read by the operator.**` is unticked while the issue's Plan row is ticked and the Log records three batches. This will meet `sdlc close`'s plan-unchecked gate; tick it or take `--no-plan-check` with the why in `--verified`.

```findings
dispose:
  - id: BR-11
    disposition: addressed
    note: |
      harvest_conformance_test.go:65-74 derives gloss+known via testDict + senseFacts; all 8 bandingWords verified present in testdata/entries/en.
  - id: BR-12
    disposition: addressed
    note: |
      atlas/define.md:1680-1686 states the set check and names it a breaking CLI change; README.md:433-436 carries it. The exit-code table gap is raised separately.
  - id: BR-13
    disposition: addressed
    note: |
      All seven members pinned through run() at harvest_test.go:696-720, plus TestRunHarvestThroughTheWiringHop for the withStore hop and the bare -agreement default.
  - id: BR-14
    disposition: not-addressed
    note: |
      Unswept, and M2 landing INVERTED it: README.md:459-461 now reads "Nothing writes this yet - authoring is the next milestone" about a surface --harvest shipped. The atlas is correct at 1419.
  - id: BR-15
    disposition: addressed
    note: |
      renderBandPrompt enumerates store.Bands() at harvest_band.go:68. No symmetric pin though - reverting to a hardcoded scale leaves the golden green, unlike the domain half.
  - id: BR-24
    disposition: not-addressed
    note: |
      Measured at HEAD on real deck words - light/"The ___house at Portland Head", set/"The ___tlement", run/"The ___way", bank/"___ruptcy" all pass stemUsesTheWord. That is 4 of the checkpoint deck's A1 words shipping an unusable item; more than cosmetic.
  - id: BR-34
    disposition: addressed
    note: |
      Third member verified by revert - moving widened[tier]++ above the veto loop reddens TestTheTierReportCountsOnlyWrittenItems. All three authoring outage branches covered by TestEveryAuthoringOutagePathReportsSurvivors.
  - id: BR-36
    disposition: not-addressed
    note: |
      Both unchanged. store/item.go:200 still reads "// : the same input prunes to the"; harvest_test.go:19-27 still stacks two harvestRig doc comments.
  - id: BR-37
    disposition: not-addressed
    note: |
      plan:456 still says 22 over a list I counted at 24; no mutation or test name per row; plan:492 still unticked while the issue's row is ticked.
  - id: BR-38
    disposition: addressed
    note: |
      harvest.go:408 now walks tierSameDomain first; verified by revert - dropping it reddens TestTheTierReportCoversEveryWrittenItem at 4 authored / 0 reported.
  - id: BR-39
    disposition: not-addressed
    note: |
      optionsPerItem = 3 unchanged at harvest.go:54; play.maxOptions still unexported although this window exported ShuffleInts across the same seam.
  - id: BR-40
    disposition: not-addressed
    note: |
      Measured at HEAD - one exhausted budget prints both "stopped at the --limit of 5" and "stopped authoring at the --limit of 5". prune's parameter is now named max, shadowing the Go builtin.
  - id: BR-41
    disposition: not-addressed
    note: |
      The three named lines are correct at HEAD, but the mechanism the finding named as the deliverable did not land - repo_guard_test.go has ZERO diff across the whole window, and retiredPhrases (:1131) is the registry it asked for. Fourth hand sweep of the same class.
  - id: BR-42
    disposition: addressed
    note: |
      Verified by revert on a scratch copy - removing tierSameDomain from the report loop reddens TestTheTierReportCoversEveryWrittenItem, so the doc sentence is now written from the code.
  - id: BR-43
    disposition: addressed
    note: |
      Verified by revert - the mutation the finding prescribed reddens the pin, and TestTheTierReportCoversEveryWrittenItem reddens on the report-loop mutation. Both checks re-run here rather than read.
  - id: BR-44
    disposition: not-addressed
    note: |
      harvest.go:82 still says runWithin is the ONLY path to a model while runHarvestAgreement calls llm.Run at :475; the errBudget arms at :166 and :281 are still unreachable, since each loop's bud.spent() exit runs with nothing decrementing in between.
findings:
  - id: new
    severity: Important
    family: store-contract-unheld-by-suite
    title: |
      Forget leaves facts/ and items/ behind, so a forgotten word's bad material is unregenerable
    detail: |
      This is the 4th finding in family store-contract-unheld-by-suite. Do NOT fix the
      cell. The rule: when a window adds a persisted per-key surface to Store, every
      per-key lifecycle verb is decided against it in the interface contract and the
      decision is held by storetest. Forget's contract (store.go) enumerates exactly one
      exemption - "does NOT remove events" - and storetest/suite.go:239 asserts deck and
      events only, so the two surfaces this window added were never walked. Reachable:
      forget a word whose item read badly, look it up again, re-harvest, and
      harvest.go:269 reads the stale item and skips authoring; nothing but hand-deleting
      items/<lang>/<key>.yaml clears it. Measured prevalence at HEAD - 3 surfaces survive
      Forget (usage/, facts/, items/), 2 of them new in this window. The deliverable is
      the cross-product written down and pinned, not a line in Forget.
  - id: new
    severity: Minor
    family: doc-understates-surface
    title: |
      README's exit-code table declares itself an enumeration and omits every code this window added
    detail: |
      This is the 3rd finding in family doc-understates-surface. Do NOT fix the two rows.
      The rule: a doc enumeration that states its own completeness is RE-DERIVED from the
      code whenever the window adds a member, and the re-derivation is what a guard can
      check - the same shape TestPlanTablesNameEntitiesThatExist already has one axis
      over. README.md:590 says "What produces each is enumerated rather than sampled,
      because a list of examples goes stale the moment a new one is added and nothing
      says so", and then: the `2` row omits two modes on one line (main.go:589-593,
      pinned by TestRunRefusesTwoModes) and every -limit/-agreement usage error
      (main.go:612-632, pinned by TestRunHarvestUsageErrors); the `1` row omits all of
      --harvest's exit-1 paths - no deck, empty deck, no model seam, unreadable facts,
      mid-run outage, and "nothing is banded yet" in agreement mode. Measured prevalence
      in this one table: 2 of 2 rows are missing members added by this window.
```

---

## Re-review — 2026-09-06T14:23:05-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 10 — authored practice items: level-tagged words, and stems the model writes offline |
| repo | tools |
| issue file | workshop/issues/000010-vocab-harvest.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0b8d9930762168cf52f77c5d0864599f678d3b5d..da5c395ae416e6ed0b934f83a830271bdb321cca |
| command | sdlc close --issue 10 |
| reviewer | claude |
| timestamp | 2026-09-06T14:23:05-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 10 disposes the ten open findings and reviews the one commit no prior round saw (`da5c395`). **BR-45 is genuinely addressed** — I reverted it in a scratch worktree and both halves went red (the `storetest` row fails against `Mem` *and* `YAML`; `TestPerWordDirsCoverEveryRuntimeDir` names `facts` and `items` when they are dropped from the classification), so the fix is the class, not the cell. **Nine findings remain not-addressed**, all carried from round 9 with no intervening commit touching them: BR-14, BR-24, BR-36, BR-37, BR-39, BR-40, BR-41, BR-44, BR-46. Suite is green (`go test ./...`), `go vet` clean under both tag sets, `gofmt` clean. Nothing blocks the gate under the severity contract — the one Important (BR-41) is a missing *mechanism*, not a wrong line — but two of the open Minors are user-visible defects at an issue close, and I would not close without them: **BR-24** ships permanently-cached broken items (`"The ___house at Portland Head"` for `light`, re-measured at HEAD), and **BR-14** now tells a README reader that authoring "is the next milestone" about a surface this milestone shipped.

**1. Strengths**

- `runWithin` (`cmd/define/harvest.go:91`) is the right shape for the `-limit` Critical: there is no way to reach a model without charging, and no way to charge whose refusal isn't an `error` the caller must already handle. Structural, not disciplined.
- `perWordDirs` + `TestPerWordDirsCoverEveryRuntimeDir` (`cmd/define/store/yaml.go:695`, `yaml_test.go:701`) answers BR-45 with an enumeration derived from `RuntimeDirs` rather than two `os.Remove` calls, and the guard fires on the exact mutation that caused the bug — verified by revert.
- The ARCH-DRY question the plan asked about `play.PickOptions` is *answered in the tree* (`harvest_item.go:126-160`), with four structural reasons, and `ShuffleInts` was exported rather than copied — a duplicate that had already shipped once with a different seeding step.
- `topicSpread` parses through `store.ParseDomain` before counting (`harvest_judge.go:33`), so the one no-model measure cannot be inflated by casing; and it is on the production path (`harvest.go:398`), not test-only.
- The checkpoint genuinely changed the design (appositive glosses 10/20 → 0, the `glosses` verdict field split from `entails`, the pre-blanked-stem check moved ahead of both judges). That is the Done-when row no test replaces, actually doing work.

**2. Critical findings** — none.

**3. Important findings**

- **BR-41 (carried, not-addressed)** — `cmd/define/repo_guard_test.go` has **zero diff** across the whole window, and `retiredPhrases` (:1131) with `currentTruthFiles` (:1623, which binds non-test `.go`, `README`, `atlas/`, `*-plan.md`) is precisely the registry+scope the finding asked for. The three named lines are correct at HEAD; the class mechanism is not. Fourth hand sweep.

**4. Minor findings**

- **New — `Forget` removes the deck entry first, so a mid-way failure reports `false` + error with the word already gone.** Reproduced: with `facts/en/sycophantic.yaml` as a non-empty directory, `Forget` returns `(false, ENOTEMPTY)` while `words/en/sycophantic.yaml` is deleted. `--forget` prints an error and exits 1 on a word it removed; `--play`'s drop path (`play_loop.go:455`) never calls `held.dropped`. Family `inconsistent-failure-reporting`, 3rd. ARCH-ORDER.
- **New — the classification guard has one axis where the fix crosses two.** `perWordDirs` mixes the language-scoped `words/`/`facts/`/`items/` with the flat `usage/`, so `define --forget red` in an `es` directory deletes the `en` deck's news cache. Family `language-scope-not-threaded`, 3rd. Consequence is a refetch, so it is Minor; the finding is the guard.
- BR-24, BR-36, BR-37, BR-39, BR-40, BR-44, BR-46, BR-14 — all carried; see dispositions.

**5. Test coverage notes**

Tests pin real logic here: `storetest` holds `Mem` and `YAML` to one contract, the four tasks all have goldens plus wire-level fake tests plus live conformance rows, and the M2 sweep's per-finding revert-checks are real. Two gaps remain visible from the diff: no test enters the two unreachable `errBudget` arms (BR-44 — a per-block coverage assertion over `harvest*.go` is the deliverable), and no test injects a mid-`Forget` removal failure (the new finding above; `os.Remove` on a non-empty directory is a sufficient seam).

**6. Architectural notes**

ARCH-DRY — flag (BR-39). ARCH-PURE — pass; every `Core concepts` PURE row exists at its stated path and its tests run with no IO. ARCH-PURPOSE — flag (BR-41, BR-46, BR-14: three enumerations still hand-maintained). ARCH-MOCK — pass. ARCH-CONSTRAINTS — pass. ARCH-SECURE — pass (`wordFileName` still the single traversal guard across all four dirs). ARCH-ORDER — flag (the new `Forget` finding). One residual worth carrying into `#12`/`#13`: `Mem.Forget` hand-lists its three maps while only `YAML` is covered by the classification guard, so a *future* per-word surface would be caught on one implementation and not the other.

**7. Plan revision recommendations**

A `## Revisions` entry that (a) corrects "M2's 22 properties" to the 24 rows the list actually contains, (b) records the mutation and the named test per row rather than a prose list of property names, and (c) either ticks `- [ ] The generated batch, read by the operator` — the issue's Plan row is ticked and the Log records three batches — or says why the plan's row stays open.

```findings
dispose:
  - id: BR-45
    disposition: addressed
    note: |
      Revert-verified in a scratch worktree: the storetest row reddens against BOTH Mem and YAML,
      and dropping facts/items from perWordDirs reddens TestPerWordDirsCoverEveryRuntimeDir.
      Residual: Mem.Forget hand-lists its maps and the guard covers YAML only.
  - id: BR-41
    disposition: not-addressed
    note: |
      repo_guard_test.go has ZERO diff across the window; retiredPhrases (:1131) over
      currentTruthFiles (:1623, binds non-test .go + README + atlas + plans) is exactly the
      mechanism asked for and no row was added. The three named lines are correct; the class is not.
  - id: BR-24
    disposition: not-addressed
    note: |
      Re-measured at HEAD by probe - "The settlement"/set, "The lighthouse"/light, "The runway"/run,
      "Bankruptcy"/bank all pass stemUsesTheWord and blankOut renders "The ___tlement". The fix is not
      a bare trailing-boundary check: wordIndexIn deliberately allows inflections, so it needs a
      bounded suffix set (s/es/ed/ing/'s).
  - id: BR-14
    disposition: not-addressed
    note: |
      Worse than round 9 recorded: README.md:459-461 still says "Nothing writes this yet - authoring
      is the next milestone" about a surface this issue shipped. At issue close that is a false
      statement in user-facing docs, not a forward-looking one.
  - id: BR-36
    disposition: not-addressed
    note: |
      Both sites unchanged (store/item.go:200 "// : the same input prunes to the"; harvest_test.go:19-27).
      A third member measured this round - harvest_item.go:318 documents "tierAnyBand", a constant that
      never existed (introduced in fd0767b as prose only); unexported names are exempt from the symbol guard.
  - id: BR-37
    disposition: not-addressed
    note: |
      plan:456 still claims 22 over a list I counted at 24; no mutation or test name per row;
      plan:492 "The generated batch, read by the operator" still unticked while the issue's row is ticked.
  - id: BR-39
    disposition: not-addressed
    note: |
      harvest.go:54 still hardcodes optionsPerItem = 3 with a comment naming play.maxOptions as the
      reason, across a seam this window already widened twice (SampleStrings, ShuffleInts).
  - id: BR-40
    disposition: not-addressed
    note: |
      Both halves stand at HEAD - the banding loop's exit (harvest.go:148) and runAuthoring's
      (harvest.go:264) each print for one exhausted budget; store/item.go:212 is func prune(items []Item, max int),
      shadowing the Go builtin max.
  - id: BR-44
    disposition: not-addressed
    note: |
      harvest.go:81 still says runWithin is the ONLY way this file reaches a model while
      runHarvestAgreement calls llm.Run at :475; the errBudget arms at :166 and :281 remain unreachable
      because each loop's bud.spent() exit runs with nothing decrementing before the call.
  - id: BR-46
    disposition: not-addressed
    note: |
      README.md:589-593 unchanged. The table still declares itself an enumeration and omits mode
      collision, every -limit/-agreement usage error, and all of --harvest's exit-1 paths.
findings:
  - id: new
    severity: Minor
    family: inconsistent-failure-reporting
    title: |
      Forget deletes the deck entry first, so a partial failure reports "nothing removed" for a word it removed
    detail: |
      This is the 3rd finding in family inconsistent-failure-reporting. Do NOT fix the loop order alone.
      The rule: a multi-step mutation orders its effects so the value it returns is true of what happened,
      and the enumeration - Forget's four removals, the banding loop's "banded N before stopping", the three
      authoring outage branches - is walked by a test that injects a failure at each step. Only the authoring
      branches have that today (TestEveryAuthoringOutagePathReportsSurvivors). Reproduced at HEAD: with
      facts/en/sycophantic.yaml made a non-empty directory, YAML.Forget returns (false, ENOTEMPTY) while
      words/en/sycophantic.yaml is already gone - so --forget exits 1 on a word it removed, and play_loop.go:455
      never calls held.dropped. os.Remove over a non-empty directory is a sufficient seam for the test.
      ARCH-ORDER: the error path unwinds the sequencing and drops the in-flight effect.
  - id: new
    severity: Minor
    family: language-scope-not-threaded
    title: |
      perWordDirs mixes the language-scoped dirs with flat usage/, so forgetting a word in one language clears another's news cache
    detail: |
      This is the 3rd finding in family language-scope-not-threaded. Do NOT fix the usage/ row.
      The rule: a per-word verb is scoped the same way the surface it touches is scoped, and
      TestPerWordDirsCoverEveryRuntimeDir - which exists precisely to classify every runtime directory -
      classifies on ONE axis (per-word vs history) while da5c395 crosses a second (scoped vs flat).
      yaml.go:695 lists wordsDir/usageDir/factsDir/itemsDir; usageDir is RuntimeDirs[2] with no lang segment
      (yaml.go:161), so `define --forget red` in an es directory removes usage/red.yaml that the en deck
      populated. Consequence is a refetch, which is why this is Minor; the deliverable is the second axis
      in the guard, so the next surface added is classified on both.
```

---

## Re-review — 2026-09-06T14:35:03-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 10 — authored practice items: level-tagged words, and stems the model writes offline |
| repo | tools |
| issue file | workshop/issues/000010-vocab-harvest.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0b8d9930762168cf52f77c5d0864599f678d3b5d..dc1130e58d2e11a9f514160c8817fae8ab9f4702 |
| command | sdlc close --issue 10 |
| reviewer | claude |
| timestamp | 2026-09-06T14:35:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Eleven rounds in, the code side of this window is in good shape: `go test ./...` green, `go vet` clean under both tag sets, `gofmt` clean, and the two new commits since round 10 do what they claim — `da5c395`'s Forget-takes-the-material fix is held by a storetest row that runs against both implementations, and `dc1130e`'s second-axis guard reddens under mutation (I flipped `usage/` to `scoped: true` in a scratch worktree and `TestPerWordDirsCoverEveryRuntimeDir` failed with the intended message). Of the eleven findings carried in, one is addressed (BR-48) and ten are unchanged at HEAD — nothing in `dc1130e` touched them. Nothing blocking correctness remains, so this is FIX-THEN-SHIP rather than REWORK; but **BR-14 should be fixed in the close commit itself**: `cmd/define/README.md:459-461` still tells a reader "Nothing writes this yet — authoring is the next milestone" about the `items/` surface that M2 shipped and that this close is recording as delivered. That is a false statement in user-facing docs at the exact moment the issue closes, and it is a two-line edit.

## 1. Strengths

- **`dc1130e` is the right shape of fix for a guard finding.** `perWordDir` (`cmd/define/store/yaml.go:707-717`) makes scoping a *declared* field and `yaml_test.go:718-723` asserts the declaration matches the path actually built. Mutation-confirmed here, not read: flipping `usage/`'s declaration reddens the guard. The flat-`usage/` behaviour is recorded in the struct comment rather than silently migrated, which is the honest handling of a #9-owned surface.
- **The storetest row for Forget asserts both halves** (`storetest/suite.go:264-327`): the derived material is gone *and* the event survives. Round 10 already revert-verified it against both `Mem` and `YAML`; the row's failure messages name the consequence ("`--harvest` will skip it as done"), not just the mismatch.
- **`Store.Forget`'s interface contract was rewritten with the behaviour** (`store/store.go:83-93`), including the deliberately narrow return semantics — reports on the *deck* entry, clears derived files regardless. That is the kind of contract text a future `Mem`/`YAML` divergence gets caught by.
- **The `#17` loop closes properly (ARCH-PURPOSE).** `reflect.go:260-283` drops an off-scale band as a third switch arm rather than coercing it, and `:277-281` canonicalises through `ParseBand` on the way to disk, so `parseLearnerBand` (`usermodel.go:276`) reads back the same spelling that was written. Every core-concepts row I checked exists at its stated path with the stated kind.
- **`glosslabel.go`'s longest-first ordering became computed** (`sortedByLengthDesc`, deriving from `store.Domains()`), turning a hand-maintained invariant into one that holds for any label added in either package (ARCH-DRY).

## 2. Critical findings

None.

## 3. Important findings

- **BR-41 remains open and is the one Important carried into the close.** `repo_guard_test.go` still has zero diff across the entire window. The three sites the finding named are correct at HEAD, so the tree is not currently wrong — what is missing is the mechanism (`retiredPhrases`, `repo_guard_test.go:1131`, over `currentTruthFiles`) that the finding named as the deliverable after the same class was swept by hand four times. Re-disposed `not-addressed` below rather than re-raised.
- **BR-14, at issue close, is no longer a Minor in substance.** `README.md:459-461` states a shipped surface is unbuilt. Filed as Minor by round 4 when it was a forward-looking line; at the close boundary it is a wrong user-facing doc. Disposed `not-addressed`; fix it in the close commit.

## 4. Minor findings

- One new finding raised: the atlas paragraph `da5c395` added (`atlas/define.md:1494-1497`) describes a one-axis guard that `dc1130e` made two-axis one commit later, and the cross-language `--forget` consequence lives only in a code comment. See the block below.
- Nine prior Minors re-disposed `not-addressed`, all verified unchanged at HEAD rather than assumed: BR-24 (`wordIndexIn` still checks only the leading boundary, so `set` matches "settlement"), BR-36 (`store/item.go:200` still reads `// : the same input prunes to the`; `harvest_test.go:19-27` still stacks two doc comments; `harvest_item.go:318` still names the never-existent `tierAnyBand`), BR-37, BR-39 (`harvest.go:53`), BR-40 (both stopped-lines and `func prune(items []Item, max int)` shadowing the builtin), BR-44 (`harvest.go:82` still claims `runWithin` is the only path to a model while `harvest.go:475` calls `llm.Run`; both `errBudget` arms still unreachable), BR-46, BR-47.
- Not filed separately, but worth folding into whoever touches BR-36: `prune` (`store/item.go:212-221`) duplicates `append`+`sortItems` across both branches — sort once, truncate conditionally.

## 5. Test coverage notes

The guard added this round is a real pin, verified by mutation rather than by reading. Coverage gaps that persist are the ones the open findings already name: no test enters either unreachable `errBudget` arm (BR-44 asks for a per-block coverage assertion over `harvest*.go`), and no test injects a failure at each step of `Forget`'s four removals (BR-47 — `os.Remove` over a non-empty directory is a sufficient seam). `Mem.Forget` still hand-lists its four maps with no guard on the `Mem` side; it is complete today and the storetest row covers all four current surfaces, so I did not file it — but a fifth per-word map added to `Mem` is caught only by review.

## 6. Architectural notes

- **ARCH-DRY** — pass with the open BR-39. `play.ShuffleInts`/`SampleStrings`/`seedFor` are reused rather than copied, and the `PickOptions`-vs-`pickDistractors` question is answered in writing at `harvest_item.go:131-159` with structural reasons, not stylistic ones.
- **ARCH-PURE** — pass. `agreement`, `topicSpread`, `pickDistractors`, `prune`, `parseLearnerBand` are all pure over values and table-tested without IO; `runWithin` is the thin charged seam.
- **ARCH-PURPOSE** — the issue's purpose is delivered (bands cached forever, items authored offline, veto exercised on real material, the checkpoint actually run and read). The open findings are the *class-vs-instance* pattern: BR-41 and BR-46 both asked for a mechanism and got, or would get, a hand sweep.
- **ARCH-MOCK** — pass. Wire-level `llmtest.Fake` behind the same seam production uses, with live conformance rows for `bandTask`/`authorTask`/`entailTask`/`vetoTask` (`harvest_conformance_test.go`), including the veto asserted in both directions.
- **ARCH-CONSTRAINTS** — pass. `harvestLimit`, `poolCap`, `agreementMaxRounds`, `ItemCap` each bound a named resource, and `-limit` now charges structurally at the only call site.
- **ARCH-SECURE** — pass. Model text is neutralised once at the store boundary (`sanitiseItems`/`sanitiseItem`), closed vocabularies refuse at parse, a damaged facts record reads as unharvested, and `Forget` cannot escape the words directory (suite row).
- **ARCH-ORDER** — flagged by the open BR-47: `Forget` performs four removals and returns `false, err` on a mid-sequence failure even though the deck entry is already gone, so the error path unwinds the sequencing and drops the in-flight effect.

## 7. Plan revision recommendations

- `workshop/plans/000010-vocab-harvest-plan.md` §Verification, M2 sweep block (~line 456): the prose says "22 properties" over a list of 24, records no mutation and no test name per row, and `- [ ] The generated batch, read by the operator` (~line 492) is unticked while the issue's Plan row for the same checkpoint is ticked. A `## Revisions` entry should correct the count and either tick the row or say why it stays open. (This is BR-37 and stays open; no new revision beyond it.)

```findings
dispose:
  - id: BR-48
    disposition: addressed
    note: |
      Mutation-verified in a scratch worktree — flipping usage/ to scoped:true reddens TestPerWordDirsCoverEveryRuntimeDir with the intended message; the second axis is declared at yaml.go:707 and asserted at yaml_test.go:718.
  - id: BR-14
    disposition: not-addressed
    note: |
      README.md:459-461 still reads "Nothing writes this yet - authoring is the next milestone" about the items/ surface M2 shipped; at the CLOSE boundary that is a false statement in user docs, not a forward-looking one.
  - id: BR-24
    disposition: not-addressed
    note: |
      Unchanged at HEAD - wordIndexIn (harvest_item.go:481) checks only the leading boundary and allows any suffix, so `set` is satisfied by "The settlement" and blankOut renders "The ___tlement".
  - id: BR-36
    disposition: not-addressed
    note: |
      All three sites unchanged - store/item.go:200 "// : the same input prunes to the", harvest_test.go:19-27 two stacked doc comments, harvest_item.go:318 naming tierAnyBand.
  - id: BR-37
    disposition: not-addressed
    note: |
      plan:456 still says 22 over a list I re-counted at 24; no mutation or test name per row; plan:492 still unticked while the issue's row is ticked.
  - id: BR-39
    disposition: not-addressed
    note: |
      harvest.go:53 still hardcodes optionsPerItem = 3 with a comment naming play.maxOptions as the reason.
  - id: BR-40
    disposition: not-addressed
    note: |
      Both halves stand - harvest.go:149/167 and :266/282 each print for one exhausted budget, and store/item.go:212 is func prune(items []Item, max int), shadowing the Go builtin.
  - id: BR-41
    disposition: not-addressed
    note: |
      repo_guard_test.go has ZERO diff across the whole window; retiredPhrases (:1131) is the registry the finding named and no row was added. The three named lines are correct at HEAD; the class is still unmechanised.
  - id: BR-44
    disposition: not-addressed
    note: |
      harvest.go:82 still says runWithin is the ONLY way this file reaches a model while runHarvestAgreement calls llm.Run at :475; both errBudget arms remain unreachable because each loop's bud.spent() exit runs with nothing decrementing before the call.
  - id: BR-46
    disposition: not-addressed
    note: |
      README.md:585-593 unchanged - the table still declares itself an enumeration and omits mode collision, every -limit/-agreement usage error, and all of --harvest's exit-1 paths.
  - id: BR-47
    disposition: not-addressed
    note: |
      yaml.go:743-757 unchanged - the loop removes words/ first and every non-ENOENT error returns (false, err), so a partial failure reports "nothing removed" for a word it removed.
findings:
  - id: new
    severity: Minor
    family: doc-understates-surface
    title: |
      The atlas describes Forget's guard as one-axis, one commit after dc1130e made it two, and the cross-language forget it recorded reaches no user-facing doc
    detail: |
      This is the 4th finding in family doc-understates-surface. Do NOT fix the sentence alone.
      The rule is the one BR-42 already stated and BR-46 restated: a doc sentence describing a
      code behaviour is a CONSUMER of that behaviour and is re-derived from the code at HEAD,
      in the same commit that changes the code. da5c395 added atlas/define.md:1494-1497 saying
      TestPerWordDirsCoverEveryRuntimeDir "fails when a new runtime directory is added without
      being classified as per-word or as history"; dc1130e changed the guard to classify on TWO
      axes fourteen minutes later and updated neither the atlas nor the README. Separately, the
      behaviour dc1130e recorded - `define --forget red` in an es directory removes the usage/
      cache the en deck filled - is user-visible and lives only in a struct comment at
      yaml.go:701-706, while README.md:528 now advertises "drop a word and its material".
      Measured prevalence in this window's last two commits: 1 of 1 doc paragraph describing the
      guard is stale, and 1 of 1 newly-recorded user-visible caveat is undocumented.
```

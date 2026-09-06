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

# Boundary Review — tools#10 (milestone M2)

| field | value |
|-------|-------|
| issue | 10 — authored practice items: level-tagged words, and stems the model writes offline |
| repo | tools |
| issue file | workshop/issues/000010-vocab-harvest.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | aef9d75836bdac7500bd87c58c0bea31725a9e88..0051c91cf3a5d62e2a1191153dd001d493d8b755 |
| command | sdlc milestone-close --issue 10 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-04T16:37:30-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2 delivers real work: the model writes stems and nothing else (`authoredStem` has no distractors field, which makes "selected, never invented" structural), the veto and entailment judges both fire on committed known-bad cases, and the checkpoint was genuinely run and read — three live batches, four design changes that no green suite could have produced. `go test ./...`, `go vet`, `gofmt` are all clean. What blocks SHIP is two measured things. First, `--limit` does not bound the authoring pass's model calls: with `-limit 2` over a deck of 8 and the judge rejecting every stem, the run makes **8 author calls + 8 entail calls** — README:374, `atlas/define.md:1431` and `harvestLimit`'s own doc all say the flag bounds a run's calls, and this is the one path in the program that spends money on the exact scenario the constant was created to prevent. Second, I ran a 22-property mutation sweep over M2's stated behaviour: **13 reddened a named test, 9 did not** — and three of the nine are the milestone's headline mechanisms (the `named` requirement, the batch diversity pressure, the authoring `--limit`), each confirmed inert by direct probe rather than by suite-green alone. The plan's `## Verification` sweep row is ticked `[x]` while its table still records only M1's 13 properties; that gap is what let these ship.

## 1. Strengths

- **`stemUsesTheWord` (`harvest_item.go:418`) is the right kind of fix.** Deterministic, free, runs before either judge is paid, and it catches the subtle case — the shipped `mesa` stem *does* contain `mesa` in "First Mesa", so containment alone passes it; the defect is the `___`. Mutation-confirmed: removing either the blank check or the leading-boundary check reddens `TestStemUsesTheWord` and `TestAStemWithoutItsWordNeverReachesAJudge`.
- **The `glosses` field is separated from `entails` for a measured reason** (`harvest_judge.go:56`), and the reframing to multiple-choice is the correct diagnosis of a genuine requirement conflict, not a prompt tweak. Mutation-confirmed: ignoring `verdict.Glosses` reddens `TestAGlossedStemIsRejectedEvenThoughItEntails`.
- **The judge-before-select ordering is asserted on the wire** (`TestARejectedStemNeverReachesTheVeto`), not by reading the code — a rejected stem must cost zero veto calls, and the mutation reddens it.
- **`sanitiseItems` enforces the cap at the store** (`store/item.go:153`) rather than asking callers to remember, and `prune`'s tie-break on `Stem` is the right answer to the two-items-one-timestamp case.
- **The ARCH-DRY answer on `play.PickOptions` was actually written down** (`harvest_item.go:118-146`), with four structural reasons rather than a shrug — this is what the plan asked for and it is a good record.

## 2. Critical findings

**C1 — `--limit` does not bound the authoring pass (`cmd/define/harvest.go:205`).**
`if authored >= limit` counts *successes*, not calls. Every word whose stem is rejected by `stemUsesTheWord`, by the judge, or by a total veto falls through `continue` without charging the limit, so the run's cost is bounded by the deck, not by the flag. Measured on a scratch copy at HEAD: 8-word deck, all pre-banded, `-limit 2`, entail judge rejecting everything → **8 author calls, 8 entail calls**, and the "stopped authoring at the --limit" line never printed. On a deck of thousands that is thousands of paid calls against a documented ceiling of 200. Secondary: the same `limit` value is spent twice in one invocation (up to 200 banded *plus* 200 authored), which `--limit`'s documented meaning does not cover.

> **This is the 2nd finding in family `flag-silently-ignored`.** Do not fix this instance alone. The rule: *a flag's declared effect must hold on every pass the run takes, and the counter it increments must be the resource it names.* `--limit` names model calls, so it must be decremented by an `asked`-style counter at the call site in both passes — the concrete shape is one `budget` value threaded into `runHarvest` and `runAuthoring` and charged on every `llm.Run`, including the veto's inner loop. Enumerate the flags `runHarvest` accepts (`-limit`, `-agreement`) against the passes each run reaches, and pin each cell.

Fix sketch: replace the two independent counters with a single `calls` budget decremented next to each `llm.Run`; then `TestHarvestStopsAtTheLimit` can assert `countTask(fake, markAuthor)+countTask(fake, markEntail)+countTask(fake, markBand) <= limit` on a rig where the pool is larger than the limit (see C2 — the current assertion cannot fail).

**C2 — three of M2's stated properties have no pin; the sweep row that would have caught them is ticked but was never run for M2 (`workshop/plans/000010-vocab-harvest-plan.md:414`).**
Measured, each confirmed by direct probe rather than inferred:

| property | pin | why it cannot fail |
|---|---|---|
| the `named` requirement rejects a stem | `harvest_judge_test.go:176` | `harvestRig(t, 1)` gives a pool of 1, so `runAuthoring` bails at `len(pool) < 2` — **0 author, 0 entail, 0 veto calls**. The assertion `len(items) > 0` is satisfied by a word that was never authored for an unrelated reason. Dropping `!verdict.Named` from `harvest.go:244` leaves the suite green. |
| batch diversity pressure | `harvest_item_test.go:309` | With pressure fully disabled (`nil` map) the same pool gives `worst=3, distinct=9` — both assertions (`worst > 3`, `len(served) < 6`) pass. Disabling the sort at `harvest_item.go:235` leaves the suite green. |
| `--limit` reaches the second pass | `harvest_test.go:211` | Banding stops at 2, so the pool *is* 2 and authoring cannot exceed the limit whatever the code does. Deleting the check entirely leaves the full suite green. |

Six more mutations were also green: `prune`'s tie-break, "the answer is never its own distractor", `sortedBanded`'s stability, `Form`'s refusal in `sanitiseItem`, the learner-band fallback in `pickDistractors`, and the `## Corrections`-section guard in `parseLearnerDomains` (that last one uses `Astrology`, which `ParseDomain` refuses anyway, so the fixture cannot distinguish the guard from the parse).

> **This is the 3rd finding in family `property-without-a-pin`.** Earlier rounds fixed instances; `workshop/lessons.md` already carries the rule ("A pin that cannot fail is not a pin") and this window added the M1 sweep table right beside it. Do NOT fix these nine sites one at a time. The rule that covers them: *a milestone's Verification sweep row is not satisfied by a previous milestone's table — enumerate THIS milestone's properties, revert each, and record the named test that reddens.* Two structural causes are worth fixing at the class level rather than per-test: (a) rigs sized below `len(pool) >= 2` silently skip the entire authoring pass, so `harvestRig` should refuse or the pass should say why it skipped in a way a test can read; (b) selection assertions that depend on which seed happened to be passed pass by luck — assert over a range of seeds, as `TestPickDistractorsIsDeterministic` already does for the varies-branch.

## 3. Important findings

**I1 — no live conformance row for any of the three new tasks (ARCH-MOCK).**
The plan's Test surface commits: *"The three tasks have goldens plus fake-driven tests, and a live conformance row each"* (`plan:94`), and the Integration-points block repeats it. `cmd/define/harvest_conformance_test.go` contains only `TestBandingIsStableAgainstTheLiveService` and `TestBandClaimShapeAgainstTheLiveService` — M1's. `authorTask`, `entailTask` and `vetoTask` have zero live rows, so nothing detects drift between `llmtest.Fake`'s canned JSON and what the real service returns for these schemas. This is the milestone whose judges are the whole product; a `vetoVerdict` whose `fits` field the model stops emitting decodes to a missing-required error only if `requireSchemaFields` sees the real body. At minimum: one live row per task asserting the schema round-trips, plus the veto's `obsequious`/`sycophantic` pair — which the checkpoint already showed the live model gets right in both directions, so it is a cheap row to write.

**I2 — `learnerFacts.Domains` derives nothing; the Spec's learner-domain requirement is unmet (ARCH-PURPOSE).**
`readLearner` computes `Domains: parseLearnerDomains(md)` at `harvest_item.go:55`, and grep finds **zero production readers** — `pickDistractors` filters on `target.Domain` (the *answer word's* domain) and never sees the learner's. The field's own doc comment says the typed halves "are what selection does arithmetic on"; selection does no arithmetic on this one. The Spec's row is explicit: *"reads `#17`'s `user-model.md` so items are pitched at the right level **and drawn from the domains the learner actually reads in**."* The band half derives; the domain half is documentation. The raw model text does reach the author prompt, but that is the prompt seeing it, not selection deriving from it.

> **This is the 3rd finding in family `inert-mechanism`.** The rule: *a parsed value that no production path reads is not a delivered consumer — either wire it or delete it, and if the Spec names it, wiring it is the deliverable.* The enumeration to sweep in the same round: every field of `learnerFacts` and every return value of `pickDistractors`, checked against `grep` for a non-test reader. `Domains` and (per C2) the `selectionTier` return are the two that fail today.

**I3 — `renderEntailPrompt` and `renderVetoPrompt` take no `store.Lang` (`harvest_judge.go:85`, `:146`).**
`renderBandPrompt` threads it and explains at length why (`bandSystem`'s comment: facts are stored per-language and a Spanish working directory is a shipped path). `renderAuthorPrompt` threads it. The two judges do not, and the veto's worked example is English (`obsequious`/`sycophantic`). `TestHarvestSendsTheDecksLanguage` asserts only `reqs[0]` — the band prompt — so nothing covers the other three. With `#18` open and `d.lang = "es"` already exercised, a Spanish stem is judged by a prompt that never says which language it is reading.

> **This is the 2nd finding in family `language-scope-not-threaded`.** The rule: *every request-rendering function on a per-language path takes `store.Lang` and states it in the prompt; the wire test asserts it on every request the run sends, not on `reqs[0]`.* Sweep the enumeration — `renderBandPrompt`, `renderAuthorPrompt`, `renderEntailPrompt`, `renderVetoPrompt` — in one change, and make the language test iterate `fake.Requests()`.

**I4 — the project row records `actual: 1.99h` and `closed: 2026-09-04` before this gate ran (`workshop/projects/define-learn.md:473-474`).**
Committed in `0051c91`, the last commit of the window, one commit after `aa60fda` added to `workshop/lessons.md` the rule *"Do not write the calibration prose before the gate runs… Predeclaring an outcome and then measuring it is how a calibration ledger stops being evidence."* The paragraph hedges honestly ("This figure is pre-review"), which mitigates but does not remove the problem: the ledger now contains a number that will be wrong by the same 3x the paragraph predicts.

> **This is the 3rd finding in family `doc-predeclares-outcome`.** The rule: *`**actual:**` and `**closed:**` on a project row are written by the close gate from measurement, after the verdict — never hand-authored ahead of it.* `sdlc milestone-close` already ticks every referencing project (AGENTS.md §8), so the fix is to stop hand-writing those two fields and let the gate write them; if a placeholder is needed before the gate, it must be `pending`, not a number. The enumeration to sweep: every `**actual:**` in `workshop/projects/` whose sibling `**closed:**` predates its issue's `Review-Verdict:` trailer.

**I5 — the item cap is asserted against `Mem` only (`cmd/define/store/item_test.go`, `TestTheStoreCapsAWordsItems`).**
`storetest/suite.go` was not touched in this window, so "growth is bounded" — a `Store` guarantee — is pinned by a test that constructs `store.NewMem()` directly. It happens to hold for `YAML` because both call the shared `sanitiseItems`, but that is an implementation coincidence the suite exists to stop relying on.

> **This is the 2nd finding in family `store-contract-unheld-by-suite`.** The rule: *any promise stated on the `Store` interface is asserted in `storetest/suite.go`, never in a per-implementation test.* Sweep the enumeration: the doc comments on `Items`, `SetItems`, `WordFacts`, `SetWordFacts` in `store.go` — the cap, the newest-first read order (see M1 below), and the read-side canonicalisation rule each need a suite row.

## 4. Minor findings

- **M1 — `Items()` ordering changed and the contract does not say so.** `sanitiseItems` now sorts (newest-first, ties by stem) on both read and write, so items no longer come back in insertion order. The atlas records it under "Bounded growth"; `Store.Items`' doc comment at `store/store.go:60` — which is where `#12` will read the contract — does not. Family `behaviour-change-undocumented` (2nd; the rule: *a behaviour promised to a consumer belongs on the interface it is promised through, not only in the atlas*).
- **M2 — `stemUsesTheWord` and `blankOut` locate the word by different rules.** `stemUsesTheWord` (`harvest_item.go:418`) checks the leading boundary only, so `set` is satisfied by "The **set**tlement was reached"; `blankOut` (`harvest_judge.go:173`) then renders `The ___tlement was reached` into the veto prompt. Also, checking only the *first* occurrence means a stem where the word appears as a substring before appearing properly is falsely rejected. Family `two-spellings-of-one-predicate`.
- **M3 — `harvestPRNG`/`shuffleInts` (`harvest_item.go:378-397`) duplicates `play.prng`/`play.shuffle` step-for-step.** The stated reason ("reaching into it would export an internal or drag the store's vocabulary across that seam") is weaker than it reads: `play.SampleStrings` is already exported to `main` for exactly this, and an `int`-slice shuffle needs no store types. Note also the seeding differs (`seed*6364136223846793005+1` vs play's zero-seed guard), so "same algorithm" produces *different* sequences — the comment overstates the relationship. Family `helper-copied-not-shared`.
- **M4 — the tier report counts items that were never written.** `widened[tier]++` at `harvest.go:262` runs before the veto; an item whose candidates are all vetoed still contributes to "N item(s) drew options from X". Family `measurement-off-the-production-path` (2nd; the rule: *a batch statistic is taken over what the batch shipped, not over what it attempted* — same class as `topicSpread`, which correctly appends only on success at `harvest.go:298`).
- **M5 — inconsistent failure reporting.** An author-call failure prints "authored N item(s) before stopping; they are saved" (`harvest.go:229`); entail and veto failures return 1 with no such line, so an outage during judging leaves the operator without the count the author path gives them. Family `inconsistent-failure-reporting`.
- **M6 — README understates the new surface.** `cmd/define/README.md` says "Two things are checked" where four conditions reject (the free stem check, `entails`, `glosses`, `named`) plus the veto; and "the sentence must give the word away" is close to the *unique-recoverability* framing the checkpoint explicitly retired. Family `doc-understates-surface`.
- **M7 — `PruneForTest` is exported production API for tests only,** and `prune`'s truncation branch is unreachable from production: `SetItems` has exactly one non-test caller (`harvest.go:290`), always with one item, only for words with zero items. `ItemCap` is a guard for `#13`, which is fine — but say so, and prefer an in-package `export_test.go` alias over a permanent exported symbol.
- **M8 — `internal/llm/golden_schema_test.go:11-14`:** the edited comment leaves a 100-plus-character run-on where the new sentence was spliced into the old line.

## 5. Test coverage notes

The pure layer is well covered and genuinely pure — `pickDistractors`, `topicSpread`, `stemUsesTheWord`, `prune`, `blankOut` and the four renderers all run with no IO, no fake, no clock. The four goldens are the right artifact and the prompt-content assertions (`TestEntailPromptDoesNotDemandUniqueRecoverability` especially) pin the *reasoning* behind each requirement, not just its presence. The wire-level fake with per-task markers (`scriptAll`/`authorKey`/`countTask`) is a real improvement over positional scripting and is what makes "a rejected stem never reaches the veto" assertable at all.

The gap is concentrated in one place: **properties that only exist inside `runAuthoring`**. Everything the diff added to that function — the limit, the `served` accumulation, the tier tally, the `len(pool) < 2` bail — is reachable only through the wire fake, and four of the nine unpinned properties live there. `TestAStemThatNamesNobodyIsRejected`'s rig sizing bug (pool of 1 → the whole pass is skipped) is the sharpest instance: a rig that silently skips the code under test is worse than no test, because it reports green.

Full sweep, run against a scratch copy at `0051c91`: 22 mutations applied, **13 red / 9 green**. The 13 red: blank detection, leading word-boundary, the free stem check, `glosses`, `entails`, the veto, the store cap, `topicSpread`'s parse, `blankOut`, the band ceiling, the one-below rule, the seeded shuffle, `oneLine` on distractors.

## 6. Architectural notes

- **ARCH-DRY — flag (Minor).** M3 above. Otherwise good: `seedFor` reused with its reason stated, `store.Bands()`/`Domains()` enumerated rather than restated, `sanitiseItems` shared by both stores, the three `*Task` constructors as the single request-construction point.
- **ARCH-PURE — flag.** The pure core is real and well tested, but `runAuthoring` (`harvest.go:169-311`) holds batch *policy* — which words to author, how many, in what order, how `served` accumulates, which tier gets tallied — inside the IO shell. That is exactly where the unpinned properties are, and it is not a coincidence: policy in the shell is only testable through the fake, and a fake-driven test is easy to write so it passes for the wrong reason (C2). Recommend extracting a pure `plan(pool, existing, budget) []authoringStep` and a pure `charge(served, kept)`, leaving the shell to call the model and write.
- **ARCH-PURPOSE — flag.** The shadow-sweep: `Band` has two deriving consumers as promised (`--reflect` writes through `ParseBand`, `parseLearnerBand` reads back) ✓; `Domain` derives from `store`'s closed set in `glosslabel.go` ✓; `topicSpread` parses rather than trusting ✓. The one remaining hand-maintained restatement is the learner's *domains* (I2) — parsed into a typed value that nothing consumes, while the Spec names it as a selection input.
- **ARCH-MOCK — flag.** I1: three new external-service tasks with no live conformance row. The fake/production boundary is otherwise correct — both flows go through `llm.Run` and both stores go through `sanitiseItems`.
- **ARCH-CONSTRAINTS — flag (Critical).** C1: the declared envelope is not enforced on the pass that added the most calls. Note also that a single `--harvest` now makes up to `1 + 1 + 3 = 5` calls per word rather than 1, which the envelope section of the plan still describes as "one band call per unbanded word, one author call per item."
- **ARCH-SECURE — pass, with one note.** Model text is neutralised in one pass at the write; `Band`, `Domain` and `Form` all refuse off-set values; a damaged facts file reads as unharvested and is re-asked; the YAML read path re-canonicalises. The note: a hand-edited `items/<lang>/<key>.yaml` holding five items is silently truncated to four on *read* with no diagnostic — bounded and documented, but it degrades quietly rather than visibly.

## 7. Plan revision recommendations

The plan needs a `## Revisions` entry covering four things it currently claims and the code does not deliver:

1. **The `## Verification` mutation-sweep row (line 414) must be un-ticked or scoped.** Its table records M1's 13 properties only; M2's sweep has not been run, and running it finds 9 green. Either add an M2 table beside M1's, or change the row to per-milestone rows so a ticked box cannot span a boundary it never covered.
2. **The `## Verification` conformance row (line ~403) must say it covers M1 only,** and the Test-surface claim at line 94 ("a live conformance row each") must either be delivered for `authorTask`/`entailTask`/`vetoTask` or restated as deferred with the trigger named.
3. **Task 4 Step 1's learner-domain clause is not delivered.** The plan says the learner's domains reach selection "through the case-insensitive fold onto the closed set"; `pickDistractors` never receives them. Either wire them or record the narrowing explicitly as a deferral, and say what it costs (the Spec's stated purpose).
4. **The Core-concepts Integration-points table omits `entailTask`,** a third new `llm.Task` shipped in this milestone. Add the row so the table matches the code.

Additionally, the operating-envelope bullet ("One band call per unbanded word… one author call per item") should be restated to reflect the actual per-word cost — one band + one author + one entail + up to three veto calls — and to say what `--limit` bounds once C1 is fixed.

```findings
findings:
  - id: new
    severity: Critical
    family: flag-silently-ignored
    title: |
      --limit does not bound the authoring pass's model calls
    detail: |
      harvest.go:205 counts successes, not calls: every stem rejected by
      stemUsesTheWord, by the entailment judge, or by a total veto skips the
      limit via continue. Measured on a scratch copy at HEAD — 8-word deck, all
      pre-banded, -limit 2, judge rejecting all — the run made 8 author calls
      and 8 entail calls, and never printed the "stopped authoring" line.
      README.md:374, atlas/define.md:1431 and harvestLimit's own doc comment all
      say the flag bounds a run's calls. This is the 2nd finding in family
      flag-silently-ignored: do not patch this site. The rule is that a flag's
      declared effect must hold on every pass the run takes and the counter it
      increments must be the resource it names — thread one call budget into
      both runHarvest and runAuthoring, charged next to every llm.Run including
      the veto's inner loop, and enumerate flag-by-pass cells in the pin.
  - id: new
    severity: Critical
    family: property-without-a-pin
    title: |
      Three headline M2 properties have pins that cannot fail; the sweep row is ticked for M1 only
    detail: |
      Measured by a 22-mutation sweep at 0051c91: 13 red, 9 green. Three greens
      are confirmed by direct probe. TestAStemThatNamesNobodyIsRejected
      (harvest_judge_test.go:176) uses harvestRig(t, 1), so runAuthoring bails at
      len(pool) < 2 and makes zero author, entail and veto calls — the `named`
      requirement has no pin at all. TestPickDistractorsSpreadsAcrossABatch
      (harvest_item_test.go:309) passes with pressure fully disabled
      (worst=3, distinct=9 on a nil map). TestHarvestStopsAtTheLimit's author
      assertion cannot exceed the limit because banding already capped the pool
      at 2. Six more mutations were green: prune's tie-break, answer-never-its-
      own-distractor, sortedBanded, Form's refusal, the learner-band fallback,
      and the Corrections-section guard. This is the 3rd finding in family
      property-without-a-pin and workshop/lessons.md already carries the rule.
      Do not fix nine sites. The rule: a milestone's Verification sweep row is
      not satisfied by a previous milestone's table — enumerate THIS milestone's
      properties, revert each, record the named test that reddens. Two class
      causes to fix structurally: a rig sized below len(pool) >= 2 silently
      skips the whole authoring pass, and selection assertions that hold only
      for the one seed passed should assert over a range of seeds.
  - id: new
    severity: Important
    family: task-without-live-conformance
    title: |
      authorTask, entailTask and vetoTask ship with no live conformance row
    detail: |
      The plan's Test surface (line 94) and its Integration-points block both
      commit to "a live conformance row each". harvest_conformance_test.go holds
      only M1's two band rows. Nothing detects drift between llmtest.Fake's
      canned JSON and the real service for the three schemas this milestone's
      whole value rests on (ARCH-MOCK). The veto's obsequious/sycophantic pair
      is already known to work live from the checkpoint, so it is a cheap row.
  - id: new
    severity: Important
    family: inert-mechanism
    title: |
      learnerFacts.Domains is computed at zero production call sites
    detail: |
      readLearner sets Domains at harvest_item.go:55 and grep finds no non-test
      reader. pickDistractors filters on target.Domain — the answer word's — and
      never sees the learner's. The Spec row says items are "drawn from the
      domains the learner actually reads in"; the field's own doc says the typed
      halves are "what selection does arithmetic on". Neither is true today.
      This is the 3rd finding in family inert-mechanism. The rule: a parsed value
      no production path reads is not a delivered consumer — wire it or delete
      it, and if the Spec names it, wiring it is the deliverable. Enumeration to
      sweep in the same round: every learnerFacts field and every pickDistractors
      return value, grepped for a non-test reader.
  - id: new
    severity: Important
    family: language-scope-not-threaded
    title: |
      The entailment and veto prompts carry no store.Lang
    detail: |
      renderEntailPrompt (harvest_judge.go:85) and renderVetoPrompt (:146) take
      no language, while renderBandPrompt and renderAuthorPrompt both thread it
      and bandSystem's comment explains why. TestHarvestSendsTheDecksLanguage
      asserts only reqs[0]. With #18 open and d.lang="es" already exercised, a
      Spanish stem is judged by an unlabelled prompt whose worked example is
      English. This is the 2nd finding in family language-scope-not-threaded.
      The rule: every request-rendering function on a per-language path takes
      store.Lang and states it, and the wire test asserts it over every request
      the run sends rather than over reqs[0]. Sweep all four renderers at once.
  - id: new
    severity: Important
    family: doc-predeclares-outcome
    title: |
      The project row records actual and closed before this gate ran
    detail: |
      workshop/projects/define-learn.md:473-474 states actual 1.99h and
      closed 2026-09-04, committed in 0051c91 — one commit after aa60fda added
      the rule to workshop/lessons.md that calibration prose must not be written
      before the gate runs. The paragraph hedges honestly, but the ledger now
      holds a number the paragraph itself predicts will move by 3x. This is the
      3rd finding in family doc-predeclares-outcome. The rule: a project row's
      actual and closed fields are written by the close gate from measurement,
      after the verdict, never hand-authored ahead of it; a pre-gate placeholder
      must read "pending", not a number. Sweep every actual in workshop/projects
      whose closed predates its issue's Review-Verdict trailer.
  - id: new
    severity: Important
    family: store-contract-unheld-by-suite
    title: |
      The item cap is asserted against Mem only, not in storetest/suite.go
    detail: |
      TestTheStoreCapsAWordsItems constructs store.NewMem() directly;
      storetest/suite.go was not touched in this window, so YAML is not held to
      "growth is bounded". It happens to hold because both call the shared
      sanitiseItems — an implementation coincidence the suite exists to stop
      relying on. This is the 2nd finding in family store-contract-unheld-by-
      suite. The rule: any promise stated on the Store interface is asserted in
      storetest/suite.go, never in a per-implementation test. Enumeration to
      sweep: the doc comments on Items, SetItems, WordFacts and SetWordFacts —
      the cap, the newest-first read order, and the read-side canonicalisation
      rule each need a suite row.
  - id: new
    severity: Minor
    family: behaviour-change-undocumented
    title: |
      Items() now returns newest-first and the interface contract does not say so
    detail: |
      sanitiseItems calls prune, which sorts on both the read and the write, so
      items no longer come back in insertion order. atlas/define.md records it;
      Store.Items' doc comment at store/store.go:60 — where #12 will read the
      contract — does not. This is the 2nd finding in family behaviour-change-
      undocumented. The rule: a behaviour promised to a consumer belongs on the
      interface it is promised through, not only in the atlas.
  - id: new
    severity: Minor
    family: two-spellings-of-one-predicate
    title: |
      stemUsesTheWord and blankOut locate the word by different rules
    detail: |
      stemUsesTheWord (harvest_item.go:418) checks only the leading boundary, so
      `set` is satisfied by "The settlement was reached"; blankOut
      (harvest_judge.go:173) then renders "The ___tlement was reached" into the
      veto prompt. Checking only the first occurrence also falsely rejects a stem
      where the word appears as a substring before appearing properly.
  - id: new
    severity: Minor
    family: helper-copied-not-shared
    title: |
      harvestPRNG duplicates play.prng step-for-step
    detail: |
      harvest_item.go:378-397 restates play/pick.go:168-215. The stated reason
      (exporting an internal, or dragging store vocabulary across the seam) is
      weaker than it reads — play.SampleStrings is already exported to main for
      exactly this, and an int-slice shuffle needs no store types. The seeding
      also differs, so "same algorithm" produces different sequences for the same
      seed; the comment overstates the relationship.
  - id: new
    severity: Minor
    family: measurement-off-the-production-path
    title: |
      The tier report counts items that were never written
    detail: |
      widened[tier]++ at harvest.go:262 runs before the veto, so an item whose
      candidates are all vetoed still contributes to "N item(s) drew options from
      X". This is the 2nd finding in family measurement-off-the-production-path.
      The rule: a batch statistic is taken over what the batch shipped, not over
      what it attempted — topicSpread already gets this right at harvest.go:298
      by appending only on success.
  - id: new
    severity: Minor
    family: inconsistent-failure-reporting
    title: |
      Only the author-call failure reports what was saved before stopping
    detail: |
      harvest.go:229 prints "authored N item(s) before stopping; they are saved";
      the entail (:241) and veto (:270) failures return 1 with no such line, so
      an outage during judging leaves the operator without the count the author
      path gives them.
  - id: new
    severity: Minor
    family: doc-understates-surface
    title: |
      README describes two checks where four conditions reject
    detail: |
      cmd/define/README.md says "Two things are checked" but an item is dropped
      by the free stem check, by entails, by glosses, by named, or by a total
      veto. "The sentence must give the word away" is also close to the unique-
      recoverability framing the checkpoint explicitly retired.
  - id: new
    severity: Minor
    family: test-only-symbol-in-production-api
    title: |
      PruneForTest is exported production API, and prune's truncation is unreachable
    detail: |
      store/item.go:200 exports a symbol for tests only. SetItems has one
      non-test caller (harvest.go:290), always with exactly one item and only
      for words with zero items, so ItemCap's truncation branch cannot be reached
      in production today. That is defensible as a guard for #13 — but say so,
      and prefer an in-package export_test.go alias over a permanent exported
      symbol.
```

---

## Re-review — 2026-09-06T10:14:16-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 10 — authored practice items: level-tagged words, and stems the model writes offline |
| repo | tools |
| issue file | workshop/issues/000010-vocab-harvest.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | aef9d75836bdac7500bd87c58c0bea31725a9e88..915089b8c0a460e9f174820cf9b918ffd95b22e7 |
| command | sdlc milestone-close --issue 10 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-06T10:14:16-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The remediation commit (`bc74543`) is real work, and I verified it rather than read it: I re-ran an 18-mutation sweep against `915089b` and the M2 pins BR-16/BR-17 named now redden (the `named` requirement, batch diversity pressure, the answer-never-its-own-distractor rule, `Form`'s refusal, the learner-band fallback, the `## Corrections` guard, `prune`'s tie-break, the budget-charges-the-veto rule — all confirmed RED by reverting the code). The learner-domain tier (BR-19) is wired and pinned, three live conformance rows exist (BR-18) and one of them records a real live observation, and the project row's `actual:` is back to `pending` (BR-21). What blocks SHIP is one new Critical I measured: `blankOut` computes byte offsets on `strings.ToLower(stem)` and slices the **original** stem with them, so `blankOut("The Ⱥ institute laid the keel", "keel")` **panics** (`slice bounds out of range [31:30]`) and `blankOut("İstanbul shipwrights laid the keel yesterday.", "keel")` returns `"İstanbul shipwrights laid the___l yesterday."` — the blank eats the space, leaves part of the answer visible, and with a few more such runes emits invalid UTF-8 while leaving `keel` unblanked entirely, i.e. leaking the answer into the veto prompt this function exists to hide it from. Secondarily, a `-limit` that runs out mid-veto still **writes** the item with one or two distractors and caches it forever (a later unbounded run skips it), and the sweep table added to the plan contains a row (`the judges take the language`) that I reverted and found green.

## 1. Strengths

- **The class fixes are real fixes, not paraphrases.** `harvestRig` now `t.Fatal`s on a size that cannot exercise authoring (`harvest_test.go:32`), and `runAuthoring` says `skipped authoring: N banded word(s)` on stdout (`harvest.go:210`) — so the *structural* cause of BR-17 is closed, not just the three tests. `TestAStemThatNamesNobodyIsRejected` now asserts the pass ran before asserting what it produced.
- **The diversity pin became a comparison instead of a threshold** (`harvest_item_test.go:325-350`): pressure-off vs pressure-on over the same pool and seeds. Mutation-confirmed — deleting the sort block reddens it.
- **`prune`'s determinism is now proved by shuffling before each call** (`store/item_test.go:38-50`), which is exactly the "the fixture could not reach the branch" failure the previous round found; disabling `sortItems` reddens both prune tests.
- **`play.ShuffleInts` (`play/pick.go:238`) is the right answer to BR-25** — one shuffle, one seeding step, and the comment now says what was actually wrong with the copy instead of claiming kinship it did not have.
- **The three live conformance rows are well-designed** (`harvest_conformance_test.go:144-265`): the author row asserts a *rate* rather than a never, and records that the live model really did return a pre-blanked stem — a measured fact, not a hypothetical. The veto row checks both directions so "rejects everything" cannot pass as success.
- **`isWordByte`'s `b >= 0x80` arm is reused rather than restated**, which is what keeps `wordIndexIn`'s boundary rule correct for the Spanish path `#18` will open (ARCH-DRY).

## 2. Critical findings

**`blankOut` slices the original stem with offsets from its lowercased copy (`cmd/define/harvest_judge.go:189-193`).**
`wordIndexIn(lower, target)` returns a byte index into `strings.ToLower(stem)`; line 193 applies it to `stem`. Go's `strings.ToLower` changes byte length for 25 runes — 23 shorten (`İ` U+0130, `ẞ` U+1E9E, `Ω` U+2126, `K` U+212A, `Å` U+212B, …) and 2 lengthen (`Ⱥ` U+023A, `Ⱦ` U+023E). Measured at HEAD:

| input | result |
|---|---|
| `blankOut("The Ⱥ institute laid the keel", "keel")` | **panic**: `slice bounds out of range [31:30]` |
| `blankOut("İstanbul shipwrights laid the keel yesterday.", "keel")` | `"İstanbul shipwrights laid the___l yesterday."` |
| `blankOut("İİİİİİİİİİ keel", "keel")` | `"İİİİİ\xc4___\xb0İİ keel"` — invalid UTF-8, and **`keel` is not blanked at all** |

The stem is model-authored text — untrusted input crossing into this process (ARCH-SECURE) — and the author prompt *requires* naming real people and places, which is precisely where `İstanbul`, `Å`ngström and `Ω` come from. `stemUsesTheWord` uses the same index only as a boolean, so such a stem passes the free check and reaches `renderVetoPrompt` regardless. The failure modes are all three of the bad ones: a crash that kills a paid batch run, a corrupted prompt body, and the answer leaking to the veto — which the function's own doc comment says "would make every candidate look wrong".

Fix sketch: have `wordIndexIn` operate in the caller's coordinate space — either fold case per-rune while walking the original (`strings.EqualFold` on the candidate slice) or return `(start, end)` computed against the original string. Pin it with a table row per length-changing class (`Ⱥ` for the panic, `İ` for the leak) in `TestBlankOut`.

## 3. Important findings

**I1 — a budget-truncated veto loop still writes the item, permanently (`cmd/define/harvest.go:300`, `:326`, `:332`).**
When `bud.spend()` fails mid-item the loop `break`s with a short `kept`, and because `len(kept) != 0` the item is written anyway. Measured on a pre-banded 8-word rig: `-limit 3` writes `certiorari` with **one** distractor, `-limit 4` with two. A subsequent `-limit 100` run does **not** repair it — `len(existing) > 0` skips the word — so a two-option question is cached forever. This contradicts `TestEveryCandidateVetoedLeavesTheWordUnauthored`'s stated invariant ("an item whose every candidate is vetoed is NOT written with two options"), which holds for the veto and not for the budget. ARCH-ORDER: a bound-exceeded path commits its in-flight effect instead of unwinding it. Fix: require the whole candidate set to be affordable before entering the veto loop, or drop the item when the loop was cut short — a capped run is a partial run, and the item it could not finish is one it did not author.

**I2 — the flag still overruns by one call, and the two tests that claim to pin it never reach the pass (`cmd/define/harvest.go:273`; `harvest_test.go:236`, `:271`).**
> **This is the 3rd finding in family `flag-silently-ignored`.** Do not patch line 273 alone. The rule that covers it: *every `spend` is a gate as well as a charge — a budget whose refusal is discarded is a counter, not a bound; and a flag×pass cell is pinned only by a test that reaches that pass.*

Line 273 charges the entail call but discards `spend()`'s `false`, so the call is made anyway. Measured: pre-banded 8-word rig, `-limit 6` → **7 model calls**. And the enumeration BR-16 asked for was not written: `TestHarvestStopsAtTheLimit` (rig 8, limit 5) and `TestTheLimitHoldsWhenEveryStemIsRejected` (rig 8, limit 6) both spend the entire budget on **banding** — instrumented, both make `author=0 entail=0 veto=0`, so the second test's scripted rejection reply is never served and the property in its name is untested. Only `TestTheBudgetChargesTheVeto` (which calls `preBand`) reaches authoring. Write the flag×pass table (`-limit`×banding, `-limit`×authoring, `-limit`×veto-inner-loop, `-agreement`×sample) and give each cell a test that provably enters it.

**I3 — the M2 sweep table records a revert that leaves the suite green (`workshop/plans/000010-vocab-harvest-plan.md:465`).**
> **This is the 4th finding in family `property-without-a-pin`.** Earlier rounds fixed instances. Do NOT fix this instance — state the rule that covers all of them, and fix that.

The row *"the judges take the language"* is listed among 22 properties "each reverted, each reddening a named test". I reverted it: hardcoding `store.DefaultLang` in `renderEntailPrompt`, `renderVetoPrompt` **and** `renderAuthorPrompt` builds clean and leaves the entire suite green. 20 of the other rows I re-derived independently and they hold, so the table is mostly honest — which is the problem: a hand-written table's one false row is indistinguishable from its twenty true ones, and the only way to find it is to redo the work. The rule: *a milestone's sweep is a runnable artifact, not prose* — commit the reverts as a script (or a `-tags mutation` harness) that emits the table, so a row cannot be recorded without the revert having executed. The family's measured prevalence is now 4 rounds running; each round fixed the sites the previous round named and the enumeration was re-verified by hand each time.

**I4 — `-limit` changed meaning and three of the four places that state it were not updated (`cmd/define/main.go:432`, `:628`; `cmd/define/README.md:374`).**
> **This is the 3rd finding in family `behaviour-change-undocumented`.** Do NOT fix the README line alone. The rule: *when a fix changes what a user-facing contract MEANS, every statement of that meaning is part of the change — enumerate them (flag help string, in-code guard comments, README, atlas, the constant's doc comment) and re-derive each.*

The flag help still reads `"words --harvest may ask the model about in one run"`, the mode-guard comment still says `-limit bounds how many words are ASKED ABOUT`, and README:374 still says `caps how many words one run will ask about`. Only `atlas/define.md:1431` and `harvestLimit`'s own comment were updated to "calls". A user-visible consequence also went unrecorded: because both passes now share one budget, a fresh 200-word deck at the default spends all 200 calls banding and authors **nothing** until a second run, while README:383 says "It then writes the practice items."

**I5 — the new selection tier is documented nowhere, and the atlas enumeration written this window is now wrong (`atlas/define.md:1541`; `workshop/plans/000010-vocab-harvest-plan.md:69`, `:94`).**
> **This is the 2nd finding in family `doc-understates-surface`.** Do NOT fix the atlas sentence alone. The rule: *a doc that ENUMERATES a code-side set is a consumer of that set — when the set changes, the enumeration is re-derived, not left at its previous count.*

`tierLearnerDomain` was added in `bc74543` (BR-19's fix) as the **second-priority** tier, so it changes which words are selected; `grep` finds it in no atlas or README text. The atlas paragraph added in `47ed772` still lists four tiers ("same-domain-at-band, then general-at-band, then any-domain-at-or-below, and only as a last resort above the learner") where the code has five. The same rule catches the plan: the Integration-points table and the Test-surface line both say **three** `llm.Task`s while M2 ships four (`entailTask` is absent from both, though it did get a conformance row). Enumeration to sweep: the atlas tier list, the plan's task table and its "three tasks" sentence, and the README's tier-report output lines (`N item(s) drew options from …`), which appear in no doc at all.

**I6 — `Store.Items`' newest-first promise, added this window, has no suite row (`cmd/define/store/store.go:62`; `storetest/suite.go`).**
BR-22's cap row landed and a bonus `Form` row with it, both mutation-confirmed. But BR-22's enumeration explicitly named three members, and the newest-first read order is still asserted only by the pure `PruneForTest` tests — disabling `sortItems` reddens `TestPruneIsDeterministic`/`TestPruneKeepsTheNewest` and no suite row. The third member (read-side canonicalisation) is correctly recorded as structurally out of scope at `yaml.go:660`, which is the right handling; this one is just missing. One row that writes two items out of order and asserts the order back closes it.

## 4. Minor findings

- `cmd/define/store/item.go:201` — `prune`'s doc comment has a mangled fragment: `// : the same input prunes to the / // same output, every time.` The subject ("DETERMINISTIC") was lost in an edit.
- `cmd/define/store/item.go:207` — `func prune(items []Item, cap int)` shadows the builtin `cap`.
- `cmd/define/harvest.go:346` — the tier report loop skips `tierSameDomain`, so `widened[tierSameDomain]` is incremented and never read.
- `cmd/define/harvest_test.go:271` — `TestTheLimitHoldsWhenEveryStemIsRejected` is a duplicate of `TestHarvestStopsAtTheLimit` under the current rig (see I2); its distinct fixture is dead.
- A single cap prints two "stopped" lines (`stopped at the --limit` then `stopped authoring at the --limit`) for one exhausted budget.
- `internal/llm/golden_schema_test.go:15` — the reflowed comment left one line running well past the file's wrap width.

## 5. Test coverage notes

- **The three authoring-pass outage branches are untested.** `authoring stopped` (`harvest.go:255`), `judging stopped` (`:275`) and `the veto stopped` (`:305`) have no test; `grep` finds no assertion on any of those strings. Done-when 6 is ticked and pinned only for the banding pass (`TestHarvestOutageKeepsWhatWasAlreadyBought`, rig 4). Pinning the entail-outage path would also pin BR-27's fix, which currently has no failing-test evidence.
- **BR-26's fix has no pin either**: moving `widened[tier]++` after the write is invisible to the suite — I reverted it to its pre-fix position and nothing reddened. Asserting the absence of a `drew options from` line inside `TestEveryCandidateVetoedLeavesTheWordUnauthored` is a one-line pin.
- Likewise the `served[k]++`-only-for-survivors rule (`harvest.go:320`): charging every candidate instead leaves the suite green.
- **Live rows ran, but the plan does not say so.** The author-shape row's comment records a real live observation, so it was executed; the Verification section's conformance row is still scoped to "at the M1 boundary" and should record M2's three rows and what they returned.
- `go test ./...`, `go vet ./...`, `gofmt -l` and `go vet -tags conformance ./...` are all clean at `915089b` — verified, not assumed.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** `play.ShuffleInts`, `wordIndexIn`, `seedFor` and `isWordByte` are all reuse rather than restatement, and the `play.PickOptions` question was answered in writing with four structural reasons (`harvest_item.go:118-146`) rather than assumed. This is the strongest axis in the diff.
- **ARCH-PURE — pass, with one note.** `pickDistractors`, `topicSpread`, `prune`, `stemUsesTheWord`, `blankOut` and the four renderers are pure and unit-tested with no IO. `runAuthoring` is now ~150 lines carrying the batch loop, the judging policy, the statistics and the IO together; `#12`/`#13` will add a second `Form` to it. A pure `decideItem(stem, verdict, candidates, vetoes) -> (Item, reason)` would make the reject/keep policy table-testable without the fake, and would have made I1 visible as a returned reason rather than a `break`.
- **ARCH-PURPOSE — pass on the code, flag on the docs.** The learner-domain tier closed the deferred half of the Spec's row, and `authoredStem` having no distractors field makes "selected, never invented" structural. I4 and I5 are the residue: two fixes changed what the system *means* and the consumers that restate that meaning were not re-derived.
- **ARCH-MOCK — pass.** `llmtest.Fake` is at the wire level and `harvestRig` drives the production path through it, so production flow and test flow share the boundary; three live conformance rows now cover the drift `#11`'s pattern was built for.
- **ARCH-CONSTRAINTS — flag (I1, I2).** The declared envelope is now enforced by a real budget, which is the improvement; what is left is that exceeding it by one call is unbounded-by-check rather than by design, and that hitting it degrades a *permanent* artifact instead of stopping cleanly.
- **ARCH-SECURE — flag (Critical).** The model's stem is untrusted input and is correctly neutralised at the store boundary; it is not treated as untrusted by `blankOut`, which assumes case-folding preserves byte offsets. The general lesson for `#12`, which will do the real blanking: parse the stem into a typed `(text, answerSpan)` at the point it arrives, rather than re-locating the answer with string arithmetic at each use.
- **ARCH-ORDER — flag (I1).** The run's carried state is small and its resumability is genuinely designed (the cache is the progress marker, each word written atomically). The gap is the interrupting event the caller cannot block: budget exhaustion arriving mid-item unwinds the sequencing but keeps the partial effect. Name which of cancel/queue/preempt/ignore governs it — "ignore and commit what we have" is a choice, but it needs to be a stated one, and here it conflicts with "items are stored COMPLETE".

## 7. Plan revision recommendations

- **`## Revisions` — "M2 boundary review round 5: 2 Critical, 5 Important, 8 Minor, remediated"**: the plan has no Revisions entry for this round at all; the M1 entries stop at round 4. Record what BR-16..BR-29 changed, so the plan's own history matches the code's.
- **Integration points table (line 69) and Test surface (line 94)**: change "three tasks" to four and add `entailTask` to the table. The "a live conformance row each" sentence is what BR-18 was derived from; it should name the tasks it applies to.
- **`## Verification`, conformance row**: extend to record M2's three live rows and what they returned (including the pre-blanked-stem observation), rather than leaving the row scoped to M1's banding measurement.
- **`## Verification`, sweep row**: replace the prose table with a pointer to a runnable sweep artifact (I3), and correct the `the judges take the language` row — it is currently recorded as red and is green.
- **Core concepts, `pickDistractors` bullet**: the tier list in the plan predates `tierLearnerDomain`; add it and say why it sits second (the answer's own domain still wins), matching `harvest_item.go:262-276`.

---

## Re-review — 2026-09-06T13:28:28-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 10 — authored practice items: level-tagged words, and stems the model writes offline |
| repo | tools |
| issue file | workshop/issues/000010-vocab-harvest.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | aef9d75836bdac7500bd87c58c0bea31725a9e88..94083a2b3b9e815bd56c88342e8a106644dfed6a |
| command | sdlc milestone-close --issue 10 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-06T13:28:28-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code half of this milestone is genuinely good and the round-5 remediation is real work, not paraphrase — I verified it by reverting rather than reading: `-limit`'s single threaded budget, the language threading through all four tasks, `prune`'s tie-break, the answer-never-its-own-distractor rule, the learner-band fallback, the `## Corrections` guard and the batch diversity pressure all redden a named test when mutated, and `harvestRig` now `t.Fatal`s on a size that cannot exercise authoring, which closes BR-17 structurally rather than at three sites. `go test ./...`, `go vet ./...`, `gofmt -l` and `go vet -tags conformance ./...` are all clean at HEAD — verified, not assumed. What keeps this from SHIP is not a new defect but a pattern in the remediation: **round 6 returned REWORK with 1 Critical, 6 Important and 6 Minor, and the two commits since took the Critical, I1, I3 and three-fifths of I4 — I2, I5, I6, every Minor and every test-coverage note are untouched at HEAD, and I re-measured each one rather than inferring it.** Separately, three fixes from the last two rounds (BR-24 partially, BR-26, BR-27) ship with no test that fails without them; I reverted each and the suite stayed green.

## 1. Strengths

- **The class fixes are structural, not site fixes.** `harvestRig` refuses a size that cannot exercise the pass (`harvest_test.go:31`) and `runAuthoring` announces `skipped authoring: N banded word(s)` on stdout (`harvest.go:218`), so the *cause* of BR-17's three vacuous pins is unavailable rather than corrected. Same shape in `wordIndexIn` (`harvest_item.go:449`): returning original-string offsets means there is no folded string left for a caller to index into.
- **`-limit` is a real budget now, and I measured the exact scenario BR-16 filed.** 8-word deck, all pre-banded, `-limit 2`, judge rejecting all: **2 calls total** (was 16), and the "stopped authoring" line prints. Reverting the threading to a per-pass budget reddens both limit tests.
- **The diversity pin became a comparison rather than a threshold** (`harvest_item_test.go:322-364`) — pressure-off vs pressure-on over the same pool and seeds, so it cannot pass with the feature deleted.
- **The language test iterates every request and asserts all four tasks were sent** (`harvest_test.go:~205`), so a run that stopped reaching one cannot pass by having nothing to check. Confirmed: hardcoding `store.DefaultLang` in `entailTask`/`vetoTask` reddens it.
- **The three live conformance rows are well-designed** (`harvest_conformance_test.go:144-265`): the author row asserts a *rate* not a never, and records a real live observation (a pre-blanked stem); the veto row checks both directions so "rejects everything" cannot read as success.

## 2. Critical findings

None survive.

## 3. Important findings

**N1 — `-limit N` still makes N+1 calls, and the two tests named for it never reach the pass they claim to bound** (`cmd/define/harvest.go:271`; `harvest_test.go:253`, `:288`).
> **This is the 4th finding in family `flag-silently-ignored`** (round 6's I2 was the 3rd and is unfixed at HEAD). Do not patch line 271. The rule: *every `spend` is a gate as well as a charge — a budget whose refusal is discarded is a counter, not a bound; and a flag×pass cell is pinned only by a test that provably enters that pass.*

Measured at HEAD on a pre-banded 8-word rig: `-limit 1` → 2 calls, `-limit 6` → 7 calls. Line 271 charges the entail call and discards `spend()`'s `false`, so the call goes out anyway. And the enumeration BR-16 asked for still is not written: `TestHarvestStopsAtTheLimit` (rig 8, limit 5) and `TestTheLimitHoldsWhenEveryStemIsRejected` (rig 8, limit 6) both spend the whole budget on **banding** — instrumented, both produce `author=0 entail=0 veto=0`, so the second test's scripted rejection reply is never served and the property in its name is untested. Only `TestTheBudgetChargesTheVeto` (which calls `preBand`) reaches authoring. Write the flag×pass table and give each cell a test that enters it.

**N2 — the `-limit` meaning sweep missed the fifth member of the enumeration round 6 wrote** (`cmd/define/main.go:628`).
> **This is the 3rd finding in family `behaviour-change-undocumented`.** The rule is already stated: *when a fix changes what a user-facing contract MEANS, every statement of that meaning is part of the change — enumerate them (flag help, in-code guard comments, README, atlas, the constant's doc comment) and re-derive each.*

`94083a2` says "swept all three places rather than the one I noticed" and updated the flag help, README and atlas. The mode-guard comment at `main.go:628` still reads `-limit bounds how many words are ASKED ABOUT`. The enumeration was written down in the finding and used as a list of noticed sites again.

**N3 — the atlas enumerates four selection tiers where the code has five, and the plan says three `llm.Task`s where M2 ships four** (`atlas/define.md:1555`, `:1578`; `workshop/plans/000010-vocab-harvest-plan.md:69`, `:94`).
> **This is the 3rd finding in family `doc-understates-surface`** (round 6's I5 was the 2nd and is unfixed at HEAD). The rule: *a doc that ENUMERATES a code-side set is a consumer of that set — when the set changes the enumeration is re-derived, not left at its previous count.*

`tierLearnerDomain` is second-priority and changes which words are selected; `grep` finds it in no atlas or README text. Worse, `atlas/define.md:1578` still records as an open question for `#12` ("whether other SPECIALIST domains at band would beat [general vocabulary]") the exact thing `harvest_item.go:262-276` says it answers. On the plan side, `entailTask` is absent from the Integration-points table and the Test-surface sentence "The three tasks have goldens plus fake-driven tests, and a live conformance row each" is false by count — there are four. Sweep: the atlas tier list, the atlas open-question paragraph, the plan's task table, the plan's "three tasks" sentence, and the README's tier-report lines (`N item(s) drew options from …`), which appear in no doc.

**N4 — `Store.Items`' newest-first promise, added this window, has no `storetest` row** (`cmd/define/store/store.go:62`; `storetest/suite.go`).
> **This is the 3rd finding in family `store-contract-unheld-by-suite`.** The rule stands from BR-22: *any promise stated on the `Store` interface is asserted in `storetest/suite.go`, never in a per-implementation test.*

BR-22's cap row landed (mutation-confirmed) and a `Form` row with it, but BR-22 named a three-member enumeration and the newest-first read order is still asserted only through the pure `PruneForTest` tests. `grep` finds no ordering assertion in the suite; the "a word may hold several items" row writes `day(1)`/`day(2)` and checks only that the `Form` discriminator survived. Read-side canonicalisation, the third member, is correctly recorded as out of scope at `yaml.go:660`.

**N5 — three fixes from the last two rounds have no test that fails without them** (`cmd/define/harvest.go:241`, `:270`, `:262`).
> **This is the 5th finding in family `property-without-a-pin`.** Do not add three assertions. The rule: *a finding is disposed by a test that reddens on revert, so remediation for a behavioural finding lands the revert-check with the fix, in the same commit.*

Measured by reverting on a scratch copy at HEAD, all green: (a) BR-27's "authored N item(s) before stopping; they are saved" on the entail and veto failure paths — `grep` finds no test asserting any of `authoring stopped`, `judging stopped`, `the veto stopped` or `before stopping`; (b) BR-26's move of `widened[tier]++` to after the write — putting it back before the veto reddens nothing; (c) `sortedBanded` — replacing the sort with a reversal leaves every behavioural test green, and it is the one member of BR-17's own six-item green list that was never swept. Related: Done-when 6 ("an outage leaves the store usable") is ticked and pinned for the **banding** pass only; the three authoring-pass outage branches are unreachable by the suite.

**N6 — `#12`'s three moved Done-when rows are still unwritten, and its own Revision names this milestone as the trigger** (`workshop/issues/000012-vocab-form-cloze.md:70-190`).
> **This is the 2nd finding in family `plan-element-ticked-unbuilt`.** The rule: *a deferral whose trigger is "when Mx lands" is swept at Mx's boundary by the issue that wrote it, not left for the consuming issue to discover.*

Task 5 Step 0 is ticked and its Revision entry is real, but it closes with "Not yet applied to the rows above. They are rewritten when `#10 M2` lands." M2 is landing at this gate and `#12`'s Done-when still reads `- [ ] Options are drawn from the pool + deck, never model-generated`, `- [ ] A near-synonym … sycophantic/obsequious`, `- [ ] The form works with the LLM seam unavailable`. A reader of `#12` would build all three again.

## 4. Minor findings

- `store/item.go:200` — `prune`'s doc comment carries an orphaned fragment (`// : the same input prunes to the / // same output, every time.`); the "DETERMINISTIC" subject was lost in an edit. Round 6's Minor, unfixed. Same class: `harvest_test.go:19-27` stacks two doc comments for `harvestRig`, the second starting mid-block.
- `store/item.go:207` — `func prune(items []Item, cap int)` shadows the builtin `cap`.
- `harvest.go:262` / `:365` — `widened[tierSameDomain]` is incremented and never read; the report loop deliberately skips that tier.
- `harvest.go:126` / `:236` — one exhausted budget prints two "stopped" lines (`stopped at the --limit` then `stopped authoring at the --limit`). Measured.
- `harvest_test.go:288` — `TestTheLimitHoldsWhenEveryStemIsRejected` is a duplicate of `TestHarvestStopsAtTheLimit` under the current rig (see N1); its distinct fixture is dead.
- `internal/llm/golden_schema_test.go:15` — the reflowed comment leaves one line running well past the file's wrap width.
- `harvest.go:46` — `optionsPerItem = 3` restates `play.maxOptions - 1` across a seam that already exports `SampleStrings` and `ShuffleInts`.
- `plan:465` — the M2 sweep block says "22 properties" over a 24-row list, and records neither the mutation nor the named test per row — which is the rule the paragraph directly beneath it states ("recording the verdict without recording the mutation is how a green row reads as red"). M1's table at least had a verdict column.
- `plan:~480` — `## Verification`'s "The generated batch, read by the operator" is unticked while the `## Plan`'s checkpoint row is ticked and the Log records three batches read.

## 5. Test coverage notes

- **Reverted and confirmed green at HEAD** (so unpinned): BR-26's tier counter, BR-27's saved-count lines, `sortedBanded`. See N5.
- **Reverted and confirmed red** (so pinned): the shared budget across passes, the entail/veto language threading, `prune`'s tie-break, the answer-never-its-own-distractor rule, the learner-band fallback, the `## Corrections` guard, the mid-item budget abandonment.
- The three authoring-pass outage branches (`harvest.go:241`, `:249`, `:270`) have no test at all; pinning the entail-outage path would also pin BR-27.
- The conformance rows **ran** — `TestAuthoredStemShapeAgainstTheLiveService`'s comment records a real live observation — but `## Verification`'s conformance row is still scoped to "at the M1 boundary" and does not record M2's three rows or what they returned.
- `store/item_test.go:TestTheStoreCapsAWordsItems` now duplicates the `storetest` cap row; harmless, but the per-implementation copy is what BR-22 objected to.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** `play.ShuffleInts`, `wordIndexIn`, `seedFor` and `isWordRune` are reuse rather than restatement, and the `play.PickOptions` question is answered in writing with four structural reasons (`harvest_item.go:118-146`) instead of assumed. `optionsPerItem` is the one restated constant (Minor).
- **ARCH-PURE — pass, with one note.** `pickDistractors`, `topicSpread`, `prune`, `stemUsesTheWord`, `blankOut`, `wordIndexIn` and the four renderers are pure and unit-tested with no IO. `runAuthoring` now carries the batch loop, the judging policy, the statistics and the IO together, and `#12`/`#13` will add a second `Form` to it — a pure `decideItem(stem, verdict, candidates, vetoes) -> (Item, reason)` would make the keep/reject policy table-testable without the fake.
- **ARCH-PURPOSE — pass on the code, flag on the docs (N2, N3).** The learner-domain tier closed the deferred half of the Spec's row and `authoredStem` having no distractors field makes "selected, never invented" structural. The shadow-sweep finds the residue on the doc side: two fixes changed what the system *means* and the enumerations that restate that meaning were re-derived only where someone noticed.
- **ARCH-MOCK — pass.** `llmtest.Fake` is at the wire level and `harvestRig` drives the production path through it, so production and test flow share the boundary; three live conformance rows now cover the drift `#11`'s pattern exists for, and one records a measured live fact.
- **ARCH-CONSTRAINTS — flag (N1).** The envelope is enforced by a real budget, which is the win. What remains is that exceeding it by one call is unbounded-by-check rather than by design, and that a fresh deck at the default spends all 200 calls banding and authors nothing on the first run — true and now only obliquely stated in the README.
- **ARCH-SECURE — pass.** The model's stem is untrusted input crossing into the process; it is neutralised at the store boundary and `wordIndexIn` no longer assumes case-folding preserves byte offsets. For `#12`, which does the real blanking: parse the stem into a typed `(text, answerSpan)` where it arrives rather than re-locating the answer with string arithmetic at each use — that is the general form of the panic round 6 found, and of BR-24's residual below.
- **ARCH-ORDER — pass.** Budget exhaustion mid-item now abandons rather than commits (`truncated`), which was round 6's I1 and is pinned. The run's carried state is small, resumability is designed (the cache is the progress marker, each word written atomically), and the interrupting events are enumerated in the code. No goroutines, no unbounded extent.

## 7. Plan revision recommendations

- **`## Revisions` — "M2 boundary review, rounds 5-7"**: the plan still has no Revisions entry for any M2 round; the entries stop at M1 round 4. Record what BR-16..BR-29 and round 6 changed so the plan's history matches the code's.
- **Integration points table (line 69) and Test surface (line 94)**: four tasks, not three; add `entailTask` and name the tasks the "a live conformance row each" sentence applies to.
- **`## Verification`, conformance row**: extend to record M2's three live rows and what they returned, rather than leaving the row scoped to M1's banding measurement.
- **`## Verification`, sweep row**: correct the count (22 → 24) and give each row its mutation and its named test, or replace the prose table with a pointer to a runnable sweep artifact — the rule the block itself states.
- **Core concepts, `pickDistractors` bullet**: the tier list predates `tierLearnerDomain`; add it and say why it sits second (the answer's own domain still wins), matching `harvest_item.go:262-276`.

```findings
dispose:
  - id: BR-16
    disposition: addressed
    note: |
      Measured: 8-word pre-banded deck, -limit 2, judge rejecting all = 2 calls total (was 16); a per-pass budget reddens both limit tests. Residual one-call overrun raised as a new finding.
  - id: BR-17
    disposition: addressed
    note: |
      All three named pins fixed and mutation-confirmed; harvestRig now refuses size 1. Five of the six named greens redden on revert; sortedBanded is the exception, folded into the new pin finding.
  - id: BR-18
    disposition: addressed
    note: |
      Three live rows exist for authorTask, entailTask and vetoTask, and the author row records a real live observation, so they ran.
  - id: BR-19
    disposition: addressed
    note: |
      learnerFacts.Domains is now read by learnerFacts.reads through tierLearnerDomain; every learnerFacts field and both pickDistractors return values have a non-test reader.
  - id: BR-20
    disposition: addressed
    note: |
      Verified by revert: hardcoding store.DefaultLang in entailTask and vetoTask reddens TestHarvestSendsTheDecksLanguage, which now iterates every request.
  - id: BR-21
    disposition: addressed
    note: |
      The project row reads "actual: pending — written by the close gate, after the verdict".
  - id: BR-22
    disposition: addressed
    note: |
      The cap row landed in storetest/suite.go. The newest-first member of its own enumeration is still missing and is raised separately.
  - id: BR-23
    disposition: addressed
    note: |
      Store.Items' doc comment now states the newest-first order; the suite row that would hold it is raised separately.
  - id: BR-24
    disposition: not-addressed
    note: |
      The two spellings were unified into wordIndexIn, but the looser rule was chosen for both, so the defect the finding measured survives.
  - id: BR-25
    disposition: addressed
    note: |
      harvestPRNG is gone; play.ShuffleInts is exported and used, with one seeding step.
  - id: BR-26
    disposition: not-addressed
    note: |
      The move is in the code but has no pin: restoring widened[tier]++ to its pre-veto position leaves the whole suite green.
  - id: BR-27
    disposition: not-addressed
    note: |
      The lines were added but no test asserts them: grep finds no assertion on "before stopping", and deleting both lines leaves the suite green.
  - id: BR-28
    disposition: addressed
    note: |
      The README now describes four checks and drops the unique-recoverability framing.
  - id: BR-29
    disposition: addressed
    note: |
      PruneForTest moved to store/export_test.go, and prune's comment states the truncation branch is a guard for #13.
findings:
  - id: new
    severity: Important
    family: flag-silently-ignored
    title: |
      -limit N still makes N+1 calls, and the two tests named for it never reach the authoring pass
    detail: |
      4th in family; round 6's I2 is unfixed at HEAD, which only changed docs. Do not
      patch harvest.go:271 alone. The rule: every spend is a gate as well as a charge —
      a budget whose refusal is discarded is a counter, not a bound; and a flag-by-pass
      cell is pinned only by a test that provably enters that pass. Measured on a
      pre-banded 8-word rig: -limit 1 gives 2 calls, -limit 6 gives 7. Line 271 charges
      the entail call and discards spend()'s false. TestHarvestStopsAtTheLimit (rig 8,
      limit 5) and TestTheLimitHoldsWhenEveryStemIsRejected (rig 8, limit 6) both spend
      the entire budget on banding — instrumented, both make author=0 entail=0 veto=0,
      so the second test's scripted rejection reply is never served. Write the
      flag-by-pass table and give each cell a test that enters it.
  - id: new
    severity: Important
    family: behaviour-change-undocumented
    title: |
      main.go:628 still states the old -limit meaning; the sweep missed the enumeration's fifth member
    detail: |
      3rd in family. The rule is already written: when a fix changes what a user-facing
      contract MEANS, every statement of that meaning is part of the change — flag help,
      in-code guard comments, README, atlas, the constant's doc comment. Commit 94083a2
      says it "swept all three places" and updated the flag help, README and atlas; the
      mode-guard comment at main.go:628 still reads "-limit bounds how many words are
      ASKED ABOUT". The enumeration existed in the finding and was used as a list of
      noticed sites again.
  - id: new
    severity: Important
    family: doc-understates-surface
    title: |
      The atlas lists four selection tiers where the code has five, and the plan says three tasks where M2 ships four
    detail: |
      3rd in family; round 6's I5 is unfixed at HEAD. The rule: a doc that ENUMERATES a
      code-side set is a consumer of that set — when the set changes the enumeration is
      re-derived, not left at its previous count. tierLearnerDomain is second-priority
      and changes which words are selected; grep finds it in no atlas or README text.
      atlas/define.md:1578 still records as an open question for #12 the exact thing
      harvest_item.go:262-276 says the tier answers. The plan's Integration-points table
      omits entailTask and its Test-surface sentence says "the three tasks" where there
      are four. Sweep: the atlas tier list, the atlas open-question paragraph, the plan
      table, the plan's "three tasks" sentence, and the README's tier-report lines.
  - id: new
    severity: Important
    family: store-contract-unheld-by-suite
    title: |
      Store.Items' newest-first promise, added this window, has no storetest row
    detail: |
      3rd in family. The rule stands from BR-22: any promise stated on the Store
      interface is asserted in storetest/suite.go, never in a per-implementation test.
      BR-22's cap row landed and is mutation-confirmed, but its enumeration named three
      members; the newest-first read order is asserted only through the pure
      PruneForTest tests. The suite's "a word may hold several items" row writes day(1)
      and day(2) and checks only that the Form discriminator survived. Read-side
      canonicalisation, the third member, is correctly out of scope at yaml.go:660.
  - id: new
    severity: Important
    family: property-without-a-pin
    title: |
      Three fixes from the last two rounds have no test that fails without them
    detail: |
      5th in family. Do not add three assertions. The rule: a finding is disposed by a
      test that reddens on revert, so remediation for a behavioural finding lands the
      revert-check with the fix, in the same commit. Reverted on a scratch copy at HEAD,
      all green: BR-27's "before stopping; they are saved" lines on the entail and veto
      paths (grep finds no test asserting any of "authoring stopped", "judging stopped",
      "the veto stopped", "before stopping"); BR-26's move of widened[tier]++ to after
      the write; and sortedBanded, replaceable by a reversal, which is the one member of
      BR-17's own six-item green list never swept. Related: Done-when 6 is ticked and
      pinned for the banding pass only — the three authoring-pass outage branches
      (harvest.go:241, :249, :270) are unreachable by the suite.
  - id: new
    severity: Important
    family: plan-element-ticked-unbuilt
    title: |
      #12's three moved Done-when rows are still unwritten, and its Revision names this milestone as the trigger
    detail: |
      2nd in family. The rule: a deferral whose trigger is "when Mx lands" is swept at
      Mx's boundary by the issue that wrote it, not left for the consuming issue to
      discover. Task 5 Step 0 is ticked and its Revision on #12 is real, but it closes
      with "Not yet applied to the rows above. They are rewritten when #10 M2 lands."
      M2 is landing, and #12's Done-when still owns "Options are drawn from the pool +
      deck, never model-generated", the sycophantic/obsequious row, and "The form works
      with the LLM seam unavailable".
  - id: new
    severity: Minor
    family: doc-comment-mangled-by-edit
    title: |
      prune's doc comment carries an orphaned fragment, and harvestRig stacks two doc comments
    detail: |
      store/item.go:200 reads "// : the same input prunes to the / // same output, every
      time." — the DETERMINISTIC subject was lost in an edit. harvest_test.go:19-27
      stacks two doc comments for harvestRig, the second beginning mid-block. Round 6
      filed the first; both are unfixed at HEAD.
  - id: new
    severity: Minor
    family: measurement-off-the-production-path
    title: |
      The M2 sweep block says 22 properties over a 24-row list and records neither the mutation nor the named test
    detail: |
      3rd in family. The rule the block itself states two paragraphs below is "a sweep
      row is only as good as the mutation behind it — recording the verdict without
      recording the mutation is how a green row reads as red". M1's table at least
      carried a verdict column; M2's is a prose list of property names. I re-derived
      seven rows by reverting and they hold, which is the problem: a hand-written
      table's one false row is indistinguishable from its true ones. Also unticked in
      the same section: "The generated batch, read by the operator", while the Plan's
      checkpoint row is ticked and the Log records three batches read.
  - id: new
    severity: Minor
    family: inert-mechanism
    title: |
      widened[tierSameDomain] is incremented and never read
    detail: |
      4th in family. harvest.go:262 counts every tier; the report loop at :365 walks
      only tierLearnerDomain, tierGeneral, tierAnyDomain and tierAboveBand. The rule
      already written for this family: a value no production path reads is wired or
      deleted. Also in this class: TestTheLimitHoldsWhenEveryStemIsRejected's distinct
      fixture is dead under the current rig (see the -limit finding).
  - id: new
    severity: Minor
    family: helper-copied-not-shared
    title: |
      optionsPerItem restates play.maxOptions - 1 across a seam that already exports two helpers
    detail: |
      2nd in family. harvest.go:46 hardcodes 3 with a comment naming play.maxOptions as
      the reason; play already exports SampleStrings and ShuffleInts to main for exactly
      this, and an option count needs none of the store's vocabulary.
  - id: new
    severity: Minor
    family: inconsistent-failure-reporting
    title: |
      One exhausted budget prints two stopped lines, and prune shadows the builtin cap
    detail: |
      2nd in family. Measured: -limit 5 on a fresh 8-word deck prints "stopped at the
      --limit of 5 model call(s)" then "stopped authoring at the --limit of 5 model
      call(s)" for one budget. Also cosmetic: store/item.go:207 declares
      func prune(items []Item, cap int), shadowing the builtin, and
      internal/llm/golden_schema_test.go:15 leaves one comment line past the file's wrap.
```

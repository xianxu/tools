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

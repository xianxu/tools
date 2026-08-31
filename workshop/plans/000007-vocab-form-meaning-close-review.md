# Boundary Review — tools#7 (whole-issue close)

| field | value |
|-------|-------|
| issue | 7 — review form 2.3: meaning multiple choice from the local deck |
| repo | tools |
| issue file | workshop/issues/000007-vocab-form-meaning.md |
| boundary | whole-issue close |
| milestone | — |
| window | a8962469ff08154f5377a7572586c575c2d4610f..15be1034b0c8a6e3544c0450cc6c9c45302a4d3e |
| command | sdlc close --issue 7 |
| reviewer | claude |
| timestamp | 2026-08-30T19:33:02-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The form is built with real care — `readGloss`'s corpus-measured walk, the empty-import purity guard held intact, the axis carried out through an optional `Missed` capability rather than a type switch, and four of the boundary's pins mutation-verified as genuinely red-when-broken. What blocks SHIP is a behavioural defect I measured rather than inferred: `PickOptions` takes the `seed` but uses it **only for the final shuffle**, never for selection — pass 1 and pass 2 both scan the pool in fixed order, so every question in a sitting draws the same first-matching domain / register / general candidate. Over a 20-word deck, **17 of 20 questions share an identical distractor set** on every day I sampled; the only variation is which slot the answer lands in. After question 1 the learner can answer the rest of the sitting correctly by elimination without knowing a single word, and the recorded axis — the thing the issue exists to produce — is a choice among the same three glosses every time. Secondarily, the plan's Core concepts table names five entities that do not exist in the tree.

## 1. Strengths

- **`readGloss` (`cmd/define/glosslabel.go:107`) is the right call, and the measurement behind it is real.** I dumped every parsed gloss in the 34-entry corpus: the walk correctly extracts `Military` from `[no object] Military (of a soldier) …`, `informal` from `(the runs) informal …`, and `Law` from `Law historical …`, and correctly rejects `bases` (all cross-references) so it falls back to form 2.1. A prefix match really would have missed all of these.
- **The word-boundary guard is mutation-verified.** Replacing `hasLabelPrefix`'s boundary check with `return true` turns four `TestReadGloss` rows red (`Lawrence`, `rarefied`, `informally`, `Musical`). The synthetic rows are earning their place.
- **Done-when 8 is genuinely pinned.** Moving `Missed` below `At` in `store/event.go` fails `TestYAMLWritesAtLastWhateverFieldsAreSet` with the exact diagnostic it promises. This is a property test, not a file-state claim — the plan's own PQ-8 rule applied correctly.
- **The `Keys()` fix is pinned from two sides.** Making `gradePrompt` ignore its argument and return the old const fails `doc_sync_test.go:62` via the duplicate-prompt check. `play_loop.go:152`'s axis wiring is likewise red under mutation (`TestAMissRecordsTheAxisItChose`).
- **ARCH-PURE holds mechanically.** `go list -f '{{join .Imports}}' ./cmd/define/play` still returns empty; `go test ./cmd/define -race` passes (115s, exit 0); `gofmt -l` and `go vet` (both tag sets) are clean.

## 2. Critical findings

**C1 — `PickOptions` ignores the seed when choosing WHICH distractors, so a whole sitting shares one option set (`cmd/define/play/pick.go:47-83`).**

`seed` reaches only `newPRNG(seed).shuffleOptions(opts)` at line 82. Pass 1 (`for _, c := range pool { if c.Axis == want && !used[c.Word] { take(c); break } }`) and pass 2 both walk `pool` in fixed order, and `pool` is built once per sitting (`play_loop.go:262`). Measured on a 20-word deck seeded from the committed corpus:

```
day+30: 20 questions, 4 distinct distractor sets: {desert,minute,pulp: 17, …3 singletons}
day+31: 20 questions, 4 distinct sets: {man,parrot,pulp: 17, …}
day+32: 20 questions, 4 distinct sets: {parrot,subject,thing: 17, …}
```

The three singletons are exactly the questions whose *target* is one of the three fixed distractors. Concretely, `sycophantic` and `quokka` and `mesa` all offer `{use, ephemeral, set, <answer>}`.

This defeats the Spec's stated purpose ("Recognition needs testing"; "the wrong answers are words they are actually confusing right now") and degrades the axis record the issue was folded around. It is also the same class the pool sampling was written to prevent — `SampleStrings`' comment says "a deck's alphabetically first forty words would otherwise supply every distractor forever"; the fix removed that across sittings and reproduced it within one (ARCH-PURPOSE: the instance, not the class).

*Fix sketch:* in `PickOptions`, build a `[]int` index permutation of `pool` shuffled with `newPRNG(seed)` and iterate that in both passes instead of `pool` directly (don't mutate the caller's slice — `choiceFor` reuses the sitting pool across questions). Then pin it: a test asserting that N targets over a rich pool yield ≥K distinct distractor sets, which goes red on today's code.

**C2 — the plan's Core concepts table names five entities that do not exist at the stated paths (`workshop/plans/000007-vocab-form-meaning-plan.md`, "Core concepts"; mirrored in the issue's `## Plan` T1/T3).**

| table says | tree has |
|---|---|
| `senseLabel` in `glosslabel.go` | `readGloss` + `glossFacts` + `leadingLabel` |
| `noadLabels` in `glosslabel.go` | `noadDomainLabels`, `noadRegisterLabels`, `noadRegionalLabels` |
| `excludeCrossReferenced` in `glosslabel.go` | `crossReferenced` (+ `mentions`) |
| `pickOptions` in `play/choice.go` | `PickOptions`, exported, in `play/pick.go` |
| `shuffle` in `play/choice.go` | `prng.shuffleOptions` in `play/pick.go` |

Two *new exported* entities in `play` are absent from the table entirely: `play.Candidate` and `play.SampleStrings`. The 2026-08-30 Revisions explain the design change that caused the rename but never update the table, so the plan still asserts a surface the code doesn't have — and `SampleStrings`, a generic string sampler exported from a forms package purely so `main` can share one PRNG, is new downstream-visible API that no plan row reviewed.

*Fix sketch:* a `## Revisions` entry rewriting the Core concepts rows to the shipped names/paths and adding `Candidate` + `SampleStrings`, plus the same correction to the issue's T1/T3 lines.

## 3. Important findings

**I1 — `TestEveryAxisIsSelectable` is vacuous for the property it claims (`cmd/define/play/pick_test.go:139`).** Done-when 6 says it goes red "when an axis is added that nothing can select". I added `AxisConnotation` to the const block (before `numAxes`) with a `String()` case and did **not** add it to `distractorAxes`: the test passes. Cause — the test's pool contains only candidates of axis `a`, so `PickOptions`' pass 2 fills the slots from any axis regardless of priority. Only the separate `String() == ""` assertion catches a forgotten axis, which is a different property. *Fix:* make the pool three `AxisGeneral` candidates **followed by** one candidate of axis `a`; then only pass 1's `distractorAxes` walk can reach it, and a missing entry fails.

**I2 — the hand-rolled PRNG and hash are justified by "pinned by this repo's tests", and no test pins them (`play/pick.go:92-118`, `optionpool.go:130-150`).** I changed the xorshift shift `13`→`12` *and* the FNV offset basis by 1 simultaneously: `./cmd/define/play` and `./cmd/define` both stay green. `TestPickOptionsIsDeterministic` only compares runs inside one binary, which `math/rand` would also satisfy — so the stated reason for rejecting `math/rand`/`hash/fnv` isn't backed. *Fix:* one golden assertion each (`newPRNG(7).next()` == a literal; `seedFor("bank","2026-08-30")` == a literal).

**I3 — nothing in `main` that the form depends on is directly tested (`cmd/define/optionpool.go`).** `buildPool`, `poolCap`, `optionCandidates`, `targetCandidate`, `choiceFor` and `seedFor` appear in zero test files. Consequences: (a) the ARCH-CONSTRAINTS budget is unenforced — the quadratic regression `play_loop.go:255`'s own comment warns about ("per-question it would be one lookup per deck word per due word") would be caught by no test, and neither would `poolCap` being bypassed; (b) `targetCandidate` returning false → `Recall` fallback (`bases` in the corpus is exactly this) is exercised by nothing — `TestASittingFallsBackToRecall` only covers the one-word-deck case. *Fix:* a counting dictionary fake asserting `lookups <= poolCap + len(due)` on a >40-word deck, and a `TestASittingFallsBackToRecall` row for a word with no usable sense.

**I4 — README's fallback threshold is off by one (`cmd/define/README.md`, "Recall, on a young deck").** "With fewer than two other words to draw on there is nothing to choose between" — but `choiceFor` only refuses at **zero** usable distractors, and `TestASittingFallsBackToRecall` pins a two-word deck producing `*play.Choice` (a 2-option question). Should read "with no other word to draw on". The same section says "The word appears with four definitions" without noting a young deck gets two or three.

## 4. Minor findings

- `cmd/define/play/choice_test.go:104-121` hand-rolls `linesOf` and `contains`; `recall_test.go` in the same package already imports `strings`, and the purity guard reads only non-test imports (`puretest.go:57` uses `.Imports`, not `.TestImports`) — ARCH-DRY.
- `pty_conformance_test.go` D8 assertion is gated on `n > 0` (at least one correct answer), but the test presses `1` for every question, so a run where all five are wrong skips the D8 check silently. Consider forcing one correct answer via `Options()`.
- `choice.go:150` `Keys()` returns `"no options"` for `len < 2`; unreachable through `choiceFor`. Dead branch.
- `glosslabel.go:88` `crossRefLeads` includes bare `"see "`, which will mark a legitimate gloss beginning "see …" unusable. Conservative (loses a candidate), not wrong-answer producing.
- `doc_sync_test.go:5` puts `.../cmd/define/play` inside the stdlib import group; gofmt-clean but inconsistent with every other file in the package.
- `atlas/define.md` — the new "Below two options it is not a question" paragraph runs into the pre-existing `TestSessionIsFormAgnostic` sentence on one unwrapped line.
- Pressing Enter/space before answering a `Choice` calls `Reveal()`, which prints the correct option; the learner can then press that digit and record `Correct`. Inherited from form 2.1's self-rating model and documented in README ("see the answer first"), but it means a recognition question can be answered for free. Worth a deliberate note rather than leaving it implicit.

## 5. Test coverage notes

Mutation-verified as genuinely load-bearing: the loop's `out.Axis` wiring, `hasLabelPrefix`'s boundary check, `At`-last field ordering, and per-form `Keys()`. Verified vacuous or absent: `TestEveryAxisIsSelectable` (I1), PRNG/FNV constants (I2), everything in `optionpool.go` (I3), and no test at all would catch C1. `go test ./...` and `go test ./cmd/define -race` both pass here; **`go test -tags conformance ./cmd/define` skips in this environment** (`no pty available: operation not permitted`), so `TestPTYPlayChoiceOffersOptionsAndRecordsTheAxis` is unverified by me — I could not confirm the issue Log's claim that it was run on a real terminal.

## 6. Architectural notes

- **ARCH-DRY — pass** (one Minor, above). No duplicated logic in the diff; exporting `SampleStrings` so `main` shares `play`'s PRNG instead of growing a second one is the right call.
- **ARCH-PURE — pass.** `play` still imports nothing; all prose handling sits in `main`; `NoWallClock` intact; the day arrives as a parameter.
- **ARCH-PURPOSE — flag (C1).** The diff delivers the form but not its point: after question 1 the sitting no longer tests recognition. Separately, `distractorAxes` (`choice.go:52`) is a hand-maintained restatement of the `Axis` set that nothing derives or enforces — I1 is the guard that was supposed to enforce it failing to.
- **ARCH-MOCK — pass.** `buildPool` goes through the same `d.dict` seam as `todaysQuestions`, so the fake dictionary + committed corpus back both production and test flow; the offline claim is pinned by making `newLLM`/`getenv` panic rather than by a nil check. Caveat: the live conformance check exists but I could not execute it here.
- **ARCH-CONSTRAINTS — partial flag (I3).** The envelope is declared (`poolCap = 40`, one lookup per pool word, nothing on the keystroke path) and correctly implemented, but no representative measurement or test enforces it, so it can drift silently.

## 7. Plan revision recommendations

1. **`## Revisions` — "the Core concepts table names entities the tree does not have."** Rewrite the eight pure-entity rows to `readGloss`/`glossFacts`/`leadingLabel`, the three `noad*Labels` tables, `crossReferenced`, `PickOptions` and `prng.shuffleOptions` at their real paths (`play/pick.go`, not `play/choice.go`), and add rows for the two new exported types the plan never reviewed: `play.Candidate` and `play.SampleStrings`. Mirror the rename into the issue's `## Plan` T1 and T3 lines.
2. **`## Revisions` — "Done-when 6's pin does not test its claim."** Record that `TestEveryAxisIsSelectable` passes with an axis absent from `distractorAxes`, and restate the row's "red when" against a pool shaped so only pass 1 can reach the axis under test.
3. **`## Revisions` — "selection is unseeded, and Done-when 1 does not notice."** Record the measurement (17/20 questions sharing one distractor set) and add a Done-when row pinning distractor variety across a sitting, since rows 1 and 3 are both green on the defective behaviour.
4. **`## Revisions` — "the determinism argument for hand-rolling needs a golden."** D5a and `seedFor`'s comment both rest on "pinned by this repo's tests"; note that mutating either constant leaves the suite green, and name the golden assertions that would make the claim true.

```findings
findings:
  - id: new
    severity: Critical
    family: seeded-selection-not-just-ordering
    title: |
      PickOptions uses the seed only to shuffle order, so one sitting shares a single distractor set
    detail: |
      cmd/define/play/pick.go:47-83 walks `pool` in fixed order in both passes and
      applies `seed` only at line 82's shuffleOptions, so every target draws the same
      first-matching domain/register/general candidate from the once-per-sitting pool.
      Measured on a 20-word corpus deck: 17 of 20 questions have an identical distractor
      set on each of five days sampled; the three exceptions are the questions whose
      target IS one of those three words. After question one the learner answers the
      rest by elimination, which defeats the Spec's recognition goal and makes the
      recorded axis a choice among the same three glosses. Fix by iterating a
      seed-shuffled index permutation of the pool in both passes, without mutating the
      caller's slice, and pin it with a distinct-set-count test.
  - id: new
    severity: Critical
    family: plan-artifact-must-match-tree
    title: |
      The plan's Core concepts table names five entities that do not exist in the tree
    detail: |
      senseLabel, noadLabels and excludeCrossReferenced are not in cmd/define/glosslabel.go
      (the tree has readGloss/glossFacts/leadingLabel, three noad*Labels tables, and
      crossReferenced); pickOptions and shuffle are not in cmd/define/play/choice.go
      (they are PickOptions and prng.shuffleOptions in play/pick.go). Two new EXPORTED
      entities, play.Candidate and play.SampleStrings, are absent from the table
      entirely. The 2026-08-30 Revisions explain the design change but never update the
      table, so the plan still claims a surface the code does not have. Same divergence
      in the issue's Plan T1/T3 lines.
  - id: new
    severity: Important
    family: guard-passes-without-the-property
    title: |
      TestEveryAxisIsSelectable passes for an axis that distractorAxes never selects
    detail: |
      Verified by adding AxisConnotation before numAxes with a String() case and leaving
      distractorAxes untouched: the test stays green. Its pool holds only candidates of
      the axis under test, so PickOptions' pass 2 fills the slots regardless of
      priority. Done-when 6 claims the row goes red when "an axis is added that nothing
      can select"; only the separate String() assertion catches it, which is a different
      property. Shape the pool as three AxisGeneral candidates followed by one of axis
      `a`, so only pass 1 can reach it.
  - id: new
    severity: Important
    family: guard-passes-without-the-property
    title: |
      The hand-rolled PRNG and FNV hash are justified by tests that do not pin them
    detail: |
      D5a rejects math/rand and hash/fnv because "a PRNG defined here is pinned by this
      repo's own tests". Changing play/pick.go's xorshift shift 13 to 12 AND
      optionpool.go's FNV offset basis by one, together, leaves ./cmd/define and
      ./cmd/define/play both green — TestPickOptionsIsDeterministic only compares runs
      within one binary, which math/rand would also satisfy. Add one golden literal each
      for newPRNG(seed).next() and seedFor(...).
  - id: new
    severity: Important
    family: declared-envelope-unenforced
    title: |
      buildPool, poolCap, optionCandidates, targetCandidate, choiceFor and seedFor have no direct tests
    detail: |
      A grep of cmd/define/*_test.go finds none of these names. Two gaps follow. The
      ARCH-CONSTRAINTS budget is unenforced: the quadratic regression play_loop.go's own
      comment warns about, and any bypass of poolCap, would be caught by nothing — a
      counting dictionary fake asserting lookups <= poolCap + len(due) on a >40-word deck
      would fix it. And targetCandidate returning false, which routes an entry with no
      usable sense to form 2.1 (the corpus word `bases` is exactly this), is exercised by
      no test; TestASittingFallsBackToRecall only covers the one-word deck.
  - id: new
    severity: Important
    family: docs-restate-behaviour-inaccurately
    title: |
      README states the recall fallback threshold one word too high
    detail: |
      cmd/define/README.md's "Recall, on a young deck" says "With fewer than two other
      words to draw on there is nothing to choose between", but choiceFor only refuses at
      ZERO usable distractors and TestASittingFallsBackToRecall pins a two-word deck
      producing *play.Choice. Should read "with no other word to draw on". The same
      section says "The word appears with four definitions" without noting that a young
      deck gets two or three.
  - id: new
    severity: Minor
    family: duplicate-stdlib-helper
    title: |
      choice_test.go hand-rolls linesOf and contains though the package's tests may import strings
    detail: |
      play/choice_test.go:104-121. The purity guard reads only non-test imports
      (puretest.go:57 uses .Imports), and recall_test.go in the same package already
      imports strings.
  - id: new
    severity: Minor
    family: conditional-assertion
    title: |
      The pty test's D8 check is skipped whenever the learner gets nothing right
    detail: |
      pty_conformance_test.go gates the "no axis on a correct answer" assertion on
      n > 0 ("correct: true" appearing at least once), but the test presses `1` for every
      question, so an all-wrong run silently skips it. Force one correct answer by reading
      Options().
  - id: new
    severity: Minor
    family: unreachable-branch
    title: |
      Choice.Keys() has a dead "no options" branch
    detail: |
      choice.go:150 returns "no options" for len(options) < 2, which choiceFor's
      len(opts) < 2 guard makes unreachable.
  - id: new
    severity: Minor
    family: docs-restate-behaviour-inaccurately
    title: |
      Revealing before answering a Choice hands the learner the correct option
    detail: |
      Enter/space calls Reveal(), which prints the correct option line; the learner can
      then press that digit and record Correct. Inherited from form 2.1's self-rating
      model and documented as "see the answer first", but it means a recognition question
      can be answered for free. Worth stating deliberately rather than leaving implicit.
  - id: new
    severity: Minor
    family: formatting-nit
    title: |
      Import grouping and an unwrapped atlas line
    detail: |
      doc_sync_test.go:5 places cmd/define/play inside the stdlib import group;
      atlas/define.md runs the new "Below two options it is not a question" paragraph into
      the pre-existing TestSessionIsFormAgnostic sentence on one long line.
```

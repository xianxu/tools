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

---

## Re-review — 2026-08-30T20:12:15-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 7 — review form 2.3: meaning multiple choice from the local deck |
| repo | tools |
| issue file | workshop/issues/000007-vocab-form-meaning.md |
| boundary | whole-issue close |
| milestone | — |
| window | a8962469ff08154f5377a7572586c575c2d4610f..b423c58c577214ec8f2426209692bae69a434595 |
| command | sdlc close --issue 7 |
| reviewer | claude |
| timestamp | 2026-08-30T20:12:15-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All eleven prior findings are genuinely addressed, and the four that were cheap to check I mutation-verified rather than took on trust: reverting the `order` permutation in `pick.go` turns `TestPickOptionsVariesTheDistractorsAcrossASitting` red (6 sets); adding `AxisConnotation` outside `distractorAxes` turns `TestEveryAxisIsSelectable` red; moving the xorshift shift and the FNV basis turns both golden tests red; deleting `keys = keys[:poolCap]` turns `TestSittingCostIsBoundedByTheCap` red. I also measured BR-1's fix end-to-end rather than only in the unit test — a 24-word corpus deck through the real `todaysQuestions` now yields **19 distinct distractor sets across 19 form-2.3 questions** (was 4 sets with 17 repeats), with domain, register and general distractors present in all 19. `go test ./...`, `-race` (115s) and `gofmt`/`vet` are clean. Nothing in the code blocks the boundary. What remains is one Important plan-artifact finding — BR-2's *instance* was fixed and its *class* was not — plus two Minors.

## 1. Strengths

- **BR-1's fix is the right shape and is measurably effective.** A seed-shuffled index permutation walked by both passes (`cmd/define/play/pick.go:57-73`), rather than shuffling the caller's slice, correctly preserves the property that `choiceFor` reuses one sitting-wide pool. Verified end-to-end, not just in the unit test.
- **`TestEveryAxisIsSelectable` now asserts the property it names** (`play/pick_test.go:151-167`). Moving from a behavioural probe (which pass 2 always satisfied) to a direct membership assertion over `AxisGeneral..numAxes` is the correct diagnosis — the review's own suggested fix (reshaping the pool) would not have worked, and the plan's Revisions say so.
- **The goldens are real goldens.** `TestPRNGSequenceIsPinned` and `TestSeedForIsPinned` pin literal sequences, and the zero-seed fixed-point check beside the first one is a genuinely load-bearing extra.
- **`TestSittingCostIsBoundedByTheCap` turns a declared envelope into an enforced one** through the same `d.dict` seam production uses (ARCH-MOCK holds: one boundary, both flows).
- **Labels are stripped from option text** (`optionCandidates` uses `f.Text`, not `s.Gloss`), so a `Law`-axis distractor doesn't print "Law" and give its own axis away. Not called out anywhere, but it's necessary for the form to work and it's right.

## 2. Critical findings

None.

## 3. Important findings

**I1 — BR-2 fixed the five rows it named and left the class unswept (`workshop/plans/000007-vocab-form-meaning-plan.md`, "Core concepts" + "Done when").**

> **This is the 2nd finding in family `plan-artifact-must-match-tree`.** Round 1 fixed instances. Do NOT fix these three instances one at a time — state and apply the rule.

The rule: **the plan's entity tables and Done-when rows are derived from the tree at the close, not hand-patched against whatever a reviewer happened to enumerate.** The enumeration is mechanical — `go doc ./cmd/define/play` for exported surface, `grep '^func Test' ` for pins, and each `D*` decision re-read against its implementing function. Three siblings survived round 1's patch:

1. `play.Missed` (`play/session.go:265`) is a **new exported interface** in `play` and appears in neither table. This is exactly the "new downstream API arrives unnoticed" that BR-2 was about; the fix added the two entities BR-2 named (`Candidate`, `SampleStrings`) and stopped there.
2. **D4a diverges from the code.** D4a says the `AxisGeneral` sense is "the first sense of the first block". `optionCandidates` (`cmd/define/optionpool.go:78-92`) walks *all* blocks and takes the first *unlabelled* usable sense. Measured on `defenestrate`: its first usable sense is `rare throw (someone) out of a window` (register), so the general candidate is a later sense, `remove or dismiss (someone) from a position of power`. The code is right — a "general" candidate that carried a label would misreport the axis — so the correction belongs in D4a, and `#12` will reuse this selection rule.
3. **`TestPickOptionsVariesTheDistractorsAcrossASitting` never became a Done-when row.** The Revisions describe it; the table still has eight rows and row 3's "red when" still reads "selection reaches for map order or wall-clock", which was green on the defect. The table's own preamble warns about exactly this drift.

*Fix:* one `## Revisions` entry that adds the `Missed` row, corrects D4a's `AxisGeneral` clause, and adds Done-when row 9 — plus a line stating the derive-from-the-tree rule so the next round doesn't produce a fourth instance.

## 4. Minor findings

**M1 — five doc comments assert behaviour the code doesn't have.**

> **This is the 3rd finding in family `docs-restate-behaviour-inaccurately`.** Rounds 1 and 2 fixed instances (README threshold, reveal-leaks-the-answer). The rule: **a comment that states a falsifiable behavioural claim must either be derived/pinned — the `doc_sync_test.go` pattern — or be weakened to the claim that is actually true.** Enumerated:

- `optionpool.go:82` — "general: the FIRST usable sense of the first block" (see I1.2).
- `optionpool.go:95` — `targetCandidate` "first usable one, first block"; the loop walks all blocks.
- `glosslabel.go:157` — "Neither overwrites a label already found", immediately above a branch where a domain label *does* overwrite a register label already found.
- `optionpool.go:130-138` and `pick.go:110-116` — the justification for hand-rolling FNV and xorshift is that a question must be "reproducible from a log indefinitely". It isn't: the option set also depends on the pool, which depends on the deck's contents and `LastSeen` ordering at the time (`store.sortDeck`), neither of which is logged. The `play` half stands on the empty-import guard regardless; the `seedFor` half in `main` has only this argument, which is why it's worth restating honestly (ARCH-DRY: `hash/fnv` is available there and FNV-1a's constants are a published standard).
- `optionpool_test.go:58` — "a pool of zero would satisfy the bound above while silently disabling form 2.3", but the assertion below it is on *lookup count*, which a pool of zero also satisfies (40 sampled filler words that all fail to look up). The property is covered elsewhere by `TestOptionCountGrowsWithTheDeck`; the comment overstates what this check does.
- Also: `pty_conformance_test.go:734-739` keeps a superseded comment paragraph ("so count instead: misses >= missed lines") directly above the paragraph that replaced it.
- Also: `cmd/define/README.md:251` still lists `kinds: looked-up, asked` with no `reviewed` — pre-existing for the kind, newly relevant now that `missed:` is written to that file.

**M2 — the issue file has two `## Log` sections** (`workshop/issues/000007-vocab-form-meaning.md:147` and `:247`), the second after `## Revisions`, outside the canonical order. Measured: #7 is the only one of 15 active issues with a duplicate. `sdlc issue validate` passes (presence-only), but any reader or tool taking "the Log" gets the 2026-08-20/27 stub and misses the build log entirely. Merge them under the single `## Log`.

**M3 —** `targetCandidate` sets `Axis: play.AxisGeneral` on the returned target, which `PickOptions` never reads (the correct option is built with no axis). Dead assignment; either drop it or say why it's there.

## 5. Test coverage notes

Mutation-verified as load-bearing this round: the pool permutation, `distractorAxes` membership, both golden constants, and the `poolCap` truncation. Previously verified and still green: the loop's `out.Axis` wiring, `hasLabelPrefix`'s word boundary, `at:`-last ordering, per-form `Keys()`.

The one gap I could not close: **`go test -tags conformance ./cmd/define -run TestPTYPlayChoiceOffersOptionsAndRecordsTheAxis` skips here** (`no pty available: operation not permitted`), so the real-terminal path — including the D8 identity assertion that replaced BR-8's conditional — is unexecuted by me for the second round running. The assertion reads correctly (`reviewed - correct:true == count("missed:")`, unconditional), and `Correct` is `omitempty` so the count is sound. Worth the operator confirming it was actually run on a terminal before the close, since the issue Log claims it as the manual verification.

## 6. Architectural notes

- **ARCH-DRY — pass, one note (M1).** No duplicated logic in the diff. Exporting `SampleStrings` so `main` shares `play`'s PRNG instead of growing a second one remains the right call. `seedFor` re-implementing FNV-1a in a package that can import `hash/fnv` is the one arguable duplication, and its stated justification is the weakest of the three.
- **ARCH-PURE — pass.** `go list -f '{{join .Imports}}' ./cmd/define/play` is still empty. All prose handling (`readGloss`, `crossReferenced`, the label tables) sits in `main`; `PickOptions` receives finished `Candidate`s. `Choice`/`Option`/`Axis`/`prng` are unit-tested with no IO, no fakes.
- **ARCH-PURPOSE — pass on the code, flag on the artifacts (I1).** The shadow-sweep on the behaviour comes out clean: distractors vary across a sitting (19/19 measured), all three axes reach real questions, the axis reaches the log through a capability rather than a type switch, and the fallback covers both a young deck and a definition-less entry. The instance-vs-class failure is confined to the plan artifacts, which is why it's Important rather than Critical.
- **ARCH-MOCK — pass.** `buildPool` and `todaysQuestions` share the `d.dict` seam; `countingDict` wraps it rather than replacing it; the offline claim is pinned by making `newLLM`/`getenv` **panic** rather than by a nil check, which is the stronger form. Live conformance exists (`-tags conformance`) but is unexecutable in this environment.
- **ARCH-CONSTRAINTS — pass, upgraded from round 1's flag.** `poolCap = 40` is now enforced by a counting test that goes red on the exact regression the code comment warns about, and nothing new sits on the keystroke path (`Prompt()` concatenates ≤4 lines per redraw).

## 7. Plan revision recommendations

1. **`## Revisions` — "the entity tables and Done-when rows must be DERIVED, not patched."** State the rule, then apply it in one sweep: add a Core-concepts row for the new exported `play.Missed` interface; correct D4a's `AxisGeneral` clause to "the first usable sense carrying no label, in document order" and cite `defenestrate` as the measurement; add Done-when row 9 — *a sitting is not one question repeated*, pinned by `TestPickOptionsVariesTheDistractorsAcrossASitting`, red when the seed reaches only the shuffle.
2. **`## Revisions` — the doc-accuracy rule (M1).** Record the enumeration and the rule: a comment stating falsifiable behaviour is derived, pinned, or weakened. In particular, restate `seedFor`'s and `prng`'s justification as "pinned against silent drift by this repo's goldens" rather than "a question is reproducible from a log indefinitely", which the unlogged pool state does not support.
3. **Issue file** — merge the two `## Log` sections into one under the canonical position (M2).

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Mutation-verified: reverting to fixed pool order gives 6 sets (test red). End-to-end on a 24-word corpus deck: 19 distinct distractor sets across 19 questions.
  - id: BR-2
    disposition: addressed
    note: |
      The five named rows are corrected in plan and issue; the class is not swept — see the new plan-artifact-must-match-tree finding.
  - id: BR-3
    disposition: addressed
    note: |
      Mutation-verified: adding AxisConnotation outside distractorAxes turns the membership assertion red.
  - id: BR-4
    disposition: addressed
    note: |
      Mutation-verified: xorshift shift 13 to 12 and the FNV offset basis each turn their golden test red.
  - id: BR-5
    disposition: addressed
    note: |
      countingDict pins the envelope (red when the poolCap truncation is removed); targetCandidate, choiceFor, optionCandidates and seedFor all now have direct tests.
  - id: BR-6
    disposition: addressed
    note: |
      README now reads "With no other word to draw on" and states that a young deck gets two or three options.
  - id: BR-7
    disposition: addressed
    note: |
      choice_test.go imports strings with the puretest .Imports rationale in the import block.
  - id: BR-8
    disposition: addressed
    note: |
      The D8 check is now an unconditional identity (reviewed minus correct equals missed-line count); no n > 0 gate remains.
  - id: BR-9
    disposition: addressed
    note: |
      The dead branch is gone and replaced by a comment explaining why no branch is needed.
  - id: BR-10
    disposition: addressed
    note: |
      Named deliberately in the README key table and in a dedicated atlas paragraph.
  - id: BR-11
    disposition: addressed
    note: |
      Import group split in doc_sync_test.go; the atlas paragraph is wrapped and no longer runs into the TestSessionIsFormAgnostic sentence.
findings:
  - id: new
    severity: Important
    family: plan-artifact-must-match-tree
    title: |
      BR-2 fixed the five rows it named; three enumerable siblings of the same class remain
    detail: |
      2nd finding in this family, so the deliverable is the rule, not the instances.
      Rule - the plan's entity tables and Done-when rows are DERIVED from the tree at
      the close (go doc for exported surface, grep for pins, each D-decision re-read
      against its implementing function), never hand-patched against a reviewer's
      enumeration. Survivors of round 1's patch - (a) play.Missed, a new EXPORTED
      interface at play/session.go:265, is in neither table, which is the same
      unnoticed-downstream-API failure BR-2 named; (b) D4a says the AxisGeneral sense
      is "the first sense of the first block" but optionCandidates walks all blocks and
      takes the first UNLABELLED usable sense - measured on defenestrate, whose general
      candidate is a later sense because its first usable one is register-labelled (the
      code is right, D4a is stale, and issue 12 reuses this rule); (c)
      TestPickOptionsVariesTheDistractorsAcrossASitting is described in the Revisions
      but never became a Done-when row, so the table still has eight rows and row 3's
      "red when" is the wording that was green on the defect.
  - id: new
    severity: Minor
    family: docs-restate-behaviour-inaccurately
    title: |
      Five doc comments state behaviour the code does not have
    detail: |
      3rd finding in this family, so the deliverable is the rule - a comment stating a
      falsifiable behavioural claim must be derived or pinned (the doc_sync_test.go
      pattern) or weakened to the true claim. Measured instances - optionpool.go:82
      "first usable sense of the first block"; optionpool.go:95 same for
      targetCandidate; glosslabel.go:157 "Neither overwrites a label already found"
      above a branch where domain does overwrite register; optionpool.go:130 and
      pick.go:110 justify hand-rolling FNV and xorshift by "reproducible from a log
      indefinitely", which the unlogged pool state (deck contents plus LastSeen
      ordering via store.sortDeck) does not support; optionpool_test.go:58 claims the
      lookup-count check proves the pool is non-empty, which it does not. Also a
      superseded comment paragraph left above its replacement at
      pty_conformance_test.go:734, and README.md:251 still lists only "looked-up,
      asked" as event kinds although this diff writes a new key into that file.
  - id: new
    severity: Minor
    family: artifact-violates-its-schema
    title: |
      The issue file has two "## Log" sections, the second outside the canonical order
    detail: |
      workshop/issues/000007-vocab-form-meaning.md carries "## Log" at line 147 and
      again at line 247, after "## Revisions". Measured - number 7 is the only one of
      15 active issues with a duplicate section. sdlc issue validate passes because it
      checks presence only, so any reader or tool taking "the Log" gets the
      2026-08-20/27 stub and misses the entire build log. Merge them under the single
      canonical heading.
```

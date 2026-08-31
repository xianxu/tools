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

---

## Re-review — 2026-08-30T20:34:03-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 7 — review form 2.3: meaning multiple choice from the local deck |
| repo | tools |
| issue file | workshop/issues/000007-vocab-form-meaning.md |
| boundary | whole-issue close |
| milestone | — |
| window | a8962469ff08154f5377a7572586c575c2d4610f..87e848c32ea068876cfa3a51a2952cd82ce22580 |
| command | sdlc close --issue 7 |
| reviewer | claude |
| timestamp | 2026-08-30T20:34:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Form 2.3 is genuinely built and genuinely wired: `go build`, `go vet`, `go test ./...`, `go test ./cmd/define/ -race` and `gofmt -l` are all clean here, `go list` confirms `play` still imports nothing, and every one of the eleven tests the plan's Done-when rows cite exists in the tree. Round 1's Critical (BR-1) is fixed the right way — a seed-shuffled index permutation in both selection passes rather than a shuffled slice — and pinned by a property test that goes red on the defect rather than green on the fix. BR-12's rule ("derive the plan tables from the tree") was applied and the three named siblings are all delivered, and BR-14 is fixed. What stops SHIP is two things: BR-13's *rule* was written into the plan Revisions but never swept, so three of its eight named instances survive and four fresh siblings joined them in this same window; and a behavioural gap nothing has raised yet — on a derivative lookup (`bargainer` → NOAD's `bargain` entry) form 2.3 marks a gloss "correct" that defines a different word, which form 2.1 never did because it showed the whole entry.

### 1. Strengths

- **`cmd/define/play/pick.go:50-71`** — the BR-1 fix is the right shape and the comment explains *why a permutation and not a shuffled slice* (`choiceFor` reuses the sitting's pool, so reordering it would couple each question to the ones before). `TestPickOptionsVariesTheDistractorsAcrossASitting` (`play/pick_test.go:200`) asserts ≥8 distinct distractor sets over 20 targets — a property the defect cannot satisfy, not a restatement of the implementation.
- **`play/pick_test.go:158`** — `TestEveryAxisIsSelectable` now derives from the `numAxes` sentinel and asserts *membership of `distractorAxes`* directly, with the behavioural check kept beside it. The comment states honestly why the behavioural check alone was vacuous (pass 2 reaches any candidate by design). This is the correct answer to BR-3.
- **`store/yaml_test.go:443-473`** — `TestYAMLWritesAtLastWhateverFieldsAreSet` derives "`at:` is the last key" from the *written record* rather than comparing two named fields. That is exactly the pin D6 needs, and it survives future field additions.
- **`play_loop_test.go:110-124`** — `gradeKey` fixes a real, quiet hazard: once a two-word rig produces form 2.3, `"y"`/`"n"` stopped being answers at all, so session-level tests would have been asserting about sessions that never graded anything.
- **`play_loop_test.go:857`** — `TestSittingWithNoModelAndNoNetwork` *panics* the model seam and `getenv` instead of nil-ing them, which is the difference between proving the path is offline and proving it has a `!= nil` guard.
- **`doc_sync_test.go:40-77`** — the README prompt guard now iterates shipped forms and composes the line exactly as the loop does, and states its residual (a third form must be added by hand) rather than hiding it.

### 2. Critical findings

None.

### 3. Important findings

**A derivative lookup makes the "correct" option the definition of a different word** — `cmd/define/optionpool.go:106` (`targetCandidate`), reachable from `choiceFor` at `optionpool.go:124` and `play_loop.go:284`.

`targetCandidate` takes the first usable gloss of whatever entry `d.dict.Lookup(word)` returned, without checking that the entry actually defines `word`. NOAD redirects derived forms to the base headword. Measured over the committed corpus (`cmd/define/testdata/entries/en`): looking up **`bargainer`** returns the entry whose headword is `bargain`, and `targetCandidate("bargainer", …)` yields `"an agreement between two or more parties as to what each party will do for the other"` — `bargain`'s definition, marked `Correct: true` and offered as the answer to "what does *bargainer* mean?". One of 34 corpus entries has this shape; `a priori`, `hot dog` and `jalapeño` also mismatch a naive headword comparison but are false positives (multi-word head tokenization and diacritics), so the real rate is ~3% of the corpus and higher in practice, since looking a derived form up is an ordinary way to use a dictionary.

Form 2.1 was immune: it showed the whole rendered entry, `DERIVATIVES bargainer | ˈbärɡ(ə)nər | noun` included, and the learner self-rated. Form 2.3 asserts a single gloss *is* the word's meaning, records `Correct`, and promotes the word in the schedule. The full entry does appear at reveal, so the learner can notice — but only after being graded right for a wrong association.

*Fix sketch:* gate `targetCandidate` (or `choiceFor`) on the entry defining the prompted word, and fall back to form 2.1 otherwise — the route `bases` already takes, so no new failure mode. `Entry.Headword()` alone is **not** sufficient: `parse.go:436` builds the head from `fields[0]`, so it returns `hot` for *hot dog* and `a` for *a priori*. Compare against the head token run (with the existing `differsOnlyByDiacritics` for accents), or check whether the word appears only under the parsed `DERIVATIVES` section (`parse.go:214` already splits it out). Pin it with a corpus row for `bargainer` beside the existing `bases` row in `TestASittingFallsBackForAnEntryWithNoDefinition`.

### 4. Minor findings

- **`play/pick.go:68-71` and `play/pick.go:153-158`** are the same reverse Fisher–Yates written twice, once over `[]int` and once over `[]Option` (ARCH-DRY). A generic `func shuffle[T any](p *prng, xs []T)` unifies them and needs no import, so the empty-allowlist guard is not the reason they are separate.

### 5. Test coverage notes

- The suite is strong where it was weak last round: `optionpool_test.go` now covers `buildPool`'s cost envelope, `optionCandidates`, `targetCandidate`, `choiceFor` and `seedFor`, and both golden pins (`TestPRNGSequenceIsPinned`, `TestSeedForIsPinned`) do what BR-4 asked.
- **The cost-envelope test's lower bound is vacuous.** In `TestSittingCostIsBoundedByTheCap` the deck is 8 real words plus 120 `filler*` words that are not in the corpus, so `counting.lookups >= poolCap` is true no matter how many lookups *succeeded*. Measured: the pool this rig actually builds is **2 candidates** at the fixed test clock, and 1–8 across 60 day-seeds — so the assertion at `optionpool_test.go:58` cannot fail for the reason its comment gives, and nothing in the test asserts a `*play.Choice` was produced. (This is BR-13's named instance; see the disposition.)
- `TestPTYPlayChoiceOffersOptionsAndRecordsTheAxis` **skips** in this environment (`no pty available: operation not permitted`), including with the sandbox off. Its D8 identity assertion reads correct on inspection, but I could not execute it; that half of the verification rests on the operator's real-terminal run recorded in the issue Log.
- Nothing pins the "one usable sense per axis, general = first *unlabelled*" rule that `optionCandidates` implements — `TestOptionCandidatesPicksOneSensePerAxis` asserts at most one per axis and that a domain candidate exists, but never that the general candidate is the first unlabelled one. `defenestrate` is the ready fixture (its general candidate is a later sense, because its first usable one is `rare`), and `#12` reuses the rule.

### 6. Architectural notes

- **ARCH-DRY** — flagged once (Minor, above). `optionCandidates`/`targetCandidate` both walk blocks×senses but implement genuinely different selection rules; keeping them apart is right. `SampleStrings` being exported so `main` shares `play`'s PRNG rather than growing a second one is the correct call.
- **ARCH-PURE** — pass. `go list -f '{{join .Imports}}' ./cmd/define/play` returns empty, so the determinism claim sits inside a mechanically enforced guard. All prose handling landed in `main` (D5/D5a) and `play` receives finished `Candidate`s. `buildPool` is the only IO in `optionpool.go` and the pure functions around it take `Entry` values.
- **ARCH-PURPOSE** — one flag (the Important finding). The shadow-sweep on this window's single sources: `Question.Keys()` → `draw` derives, README derives via `doc_sync_test.go`; the `Axis` set → `distractorAxes` and `String()` both derived-guarded from `numAxes`. The one hand-maintained restatement left standing is `README.md:252`'s `kinds: looked-up, asked`, which restates `store.EventKind` by hand, omits `reviewed`, and does not mention the `missed:` key this diff writes into that same file. That is a deferred consumer, and it is inside BR-13.
- **ARCH-MOCK** — pass. No new external dependency; the dictionary fake plus the committed corpus back every new test, `countingDict` wraps the same seam production uses, and the pty conformance test exercises the real binary against a real deck.
- **ARCH-CONSTRAINTS** — pass on the declared bound, with the caveat above. `poolCap = 40` is stated with its reasoning and `TestSittingCostIsBoundedByTheCap` enforces `lookups <= poolCap + count`; the pool is built once per sitting, not per question. Nothing new is on the per-keystroke path.

### 7. Plan revision recommendations

- A `## Revisions` entry recording that the option material is only valid when the entry *defines the prompted word* — D4 ("the gloss is the option text, and it already exists") and D4a both assume the looked-up word and the entry's headword are the same, which NOAD's derivative redirects make false. Name `bargainer` as the measured instance and state the fallback (form 2.1, the `bases` route) as the decision.
- A `## Revisions` entry closing BR-13 as a *sweep* rather than a list: state that the rule applies to test doc comments, README prose and the atlas equally, and record the enumeration that was run (e.g. `go doc` over each touched package plus a grep for the repudiated phrasings) — otherwise the next round finds the fifth instance of the same family.

---

```findings
dispose:
  - id: BR-12
    disposition: addressed
    note: |
      Verified in the tree - play.Missed is in the Integration points table (plan:149), D4a corrected (plan:61-63), Done-when row 9 added and row 3 widened (plan:179,184); all 11 cited tests exist; go doc -short on play gives 15 exported names, 11 tabled, 4 belonging to issue 6.
  - id: BR-13
    disposition: not-addressed
    note: |
      5 of 8 named instances fixed; 3 survive and 4 fresh siblings joined them, so the rule was written down but never swept.
  - id: BR-14
    disposition: addressed
    note: |
      grep -n '^## ' on the issue file shows one Log, at line 147, in canonical order.
findings:
  - id: new
    severity: Important
    family: answer-must-define-the-prompted-word
    title: |
      A derivative lookup makes form 2.3's "correct" option the definition of a different word
    detail: |
      cmd/define/optionpool.go:106 (targetCandidate) takes the first usable gloss of
      whatever entry the dictionary returned, without checking that the entry defines
      the prompted word. NOAD redirects derived forms to the base headword. Measured
      over the committed corpus - looking up `bargainer` returns the `bargain` entry,
      and targetCandidate("bargainer", ...) yields "an agreement between two or more
      parties as to what each party will do for the other", marked Correct:true and
      offered as the answer to "what does bargainer mean?". 1 of 34 corpus entries has
      this shape. Form 2.1 was immune because it showed the whole rendered entry
      including "DERIVATIVES bargainer"; form 2.3 asserts one gloss IS the meaning,
      records Correct, and promotes the word in the schedule. Fix - gate choiceFor on
      the entry defining the prompted word and fall back to form 2.1 otherwise, the
      route `bases` already takes. Entry.Headword() alone is NOT sufficient: parse.go:436
      builds the head from fields[0], so it returns "hot" for `hot dog` and "a" for
      `a priori`. Compare against the head token run (differsOnlyByDiacritics already
      exists for accents) or check the parsed DERIVATIVES section. Pin with a
      `bargainer` row beside the `bases` row in
      TestASittingFallsBackForAnEntryWithNoDefinition.
  - id: new
    severity: Minor
    family: duplicated-algorithm-should-be-one-helper
    title: |
      The same reverse Fisher-Yates is written twice in pick.go
    detail: |
      cmd/define/play/pick.go:68-71 shuffles []int and pick.go:153-158 shuffles
      []Option with an identical loop (ARCH-DRY). A generic
      `func shuffle[T any](p *prng, xs []T)` unifies them and needs no import, so the
      package's empty-allowlist guard is not the reason they are separate.
```

**Note on the BR-13 disposition — the four surviving siblings, since the rule is the deliverable.** The rule ("a comment making a falsifiable behavioural claim must be derived, pinned, or weakened to the true claim") was recorded in the plan's Revisions but the enumeration it implies was never run, so round 2 fixed the five named code comments and left everything else:

1. **`cmd/define/optionpool_test.go:66-67`** — `"D4a: at most one sense per axis, and the general one is the FIRST usable sense of the first block."` This is verbatim the stale claim BR-12b/BR-13 corrected, surviving **two lines above** the only edit round 2 made to this file (line 96). `optionCandidates` takes the first *unlabelled* usable sense across all blocks; `defenestrate` proves the difference.
2. **`cmd/define/play/pick_test.go:239-241`** — attributes to D5a the claim that a fixed seed must reproduce a question *"forever, so that a question can be reproduced from a log"*. D5a says no such thing, and round 2 repudiated exactly that phrasing in `pick.go` and `optionpool.go`.
3. **`cmd/define/play/recall.go:32-38`** — `Keys()` was inserted between `// Grade reads the self-rating.` and `Grade`. Verified: `go doc ./cmd/define/play Recall.Keys` renders "Grade reads the self-rating… Anything else returns false — a stray key is not a silent wrong answer", and `Recall.Grade` is now undocumented.
4. **`cmd/define/README.md:70`** — "A word is never offered as a distractor against a word whose dictionary definition mentions it." `mentions` (`glosslabel.go:268`) returns `false` for any headword shorter than `minCrossRefWord` (6), so the guard never fires for short words. "Never" should be the measured claim.

And the three from BR-13's own list that round 2 did not touch: `optionpool_test.go:56-59` (the lookup-count check does not establish a non-empty pool — measured above), `pty_conformance_test.go:743-745` (the superseded `misses >= missed lines` paragraph still sits above its replacement), and `README.md:252` (`kinds: looked-up, asked`, missing `reviewed` and the new `missed:` key this diff writes into that block).

---

## Re-review — 2026-08-30T21:00:40-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 7 — review form 2.3: meaning multiple choice from the local deck |
| repo | tools |
| issue file | workshop/issues/000007-vocab-form-meaning.md |
| boundary | whole-issue close |
| milestone | — |
| window | a8962469ff08154f5377a7572586c575c2d4610f..6e79cad0aa7c619822a0036d77f2441f58728bea |
| command | sdlc close --issue 7 |
| reviewer | claude |
| timestamp | 2026-08-30T21:00:40-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Round 3's `entryDefines` gate is real, correctly reasoned, and pinned by a test that goes red when I disable it — I verified by reverting in a scratch worktree. But the gate was applied to **one of the two sites that turn a (word, entry) pair into option material**. `buildPool` → `optionCandidates` is ungated, so a deck word NOAD redirects still contributes its *base entry's* glosses under the derived word's name. Measured against the committed corpus: a deck of `{bargain, bargainer, mesa, quokka}` produces the question "bargain" with options 1 and 2 **byte-identical**, one marked Correct and one graded Wrong — the learner who picks the duplicate is recorded as a miss with a fabricated axis and the word is demoted. That is ARCH-PURPOSE's "the instance, not the class" clause, on the very finding that named the class. Everything else in the diff is strong: the seam widening is clean, the tests are properties rather than restatements, README and atlas are substantially updated. Full suite, `-race` and `go vet` are green (the pty conformance test skips here — no pty available in this environment, so I could not re-verify the terminal check).

**1. Strengths**

- `cmd/define/play/session.go:265` — `Missed` as an *optional capability* rather than a widened `Grade` is the right seam: `Apply` names a capability, never a form, so Done-when 7 stays a property. `missed_test.go:44` pins the form-agnostic half.
- `cmd/define/capture.go:141` — D8 enforced by the type (`AxisNone.String() == ""` + `omitempty`) instead of a branch a caller could forget. That is the difference between an invariant and a convention.
- `cmd/define/store/yaml_test.go:434` — the torn-record test rewritten to derive `at`-is-last from the record itself rather than comparing against one named field. The old form would have stayed green for this very diff's new field.
- `cmd/define/optionpool_test.go:29` — `countingDict` makes the ARCH-CONSTRAINTS budget falsifiable, and round 3 correctly closed the "a bound is satisfied by zero" hole by also asserting a `*play.Choice` actually comes out.
- `cmd/define/doc_sync_test.go:50` — the README prompt check now iterates *shipped forms* and composes the line the way `draw` does, so the y/n-under-digits bug cannot recur silently.

**2. Critical findings**

**`cmd/define/optionpool.go:52` — the pool is not gated by `entryDefines`, so a redirected entry's glosses enter questions under the wrong word.** *(2nd finding in family `answer-must-define-the-prompted-word` — do not fix only this call site; the rule below is the deliverable.)*

The rule: **every gloss that enters a question — target *or* distractor — must come from an entry that defines the word it is attributed to, and no two options in one set may carry the same gloss.** The enumerable sites are exactly two: `targetCandidate` (gated at `optionpool.go:170`) and `optionCandidates` via `buildPool` (ungated). Three measured consequences:

- *Two identical options, one graded wrong.* Deck `{bargain, bargainer, mesa, quokka}`, `bargain` served the same fixture NOAD returns for it (`bargainer.txt` **is** the `bargain` entry): options 1 and 2 are both "an agreement between two or more parties as to what each party will do for the other"; option 2 is `Correct:false`. Picking it writes `missed: general` and demotes the word.
- *Two identical distractors.* Same run: the `mesa` and `quokka` questions each carry the `bargain` gloss twice, once as `bargain` and once as `bargainer`.
- *The near-synonym guard is systematically blind for redirected candidates* — `crossReferenced(target.Word, target.Gloss, c.Word, c.Gloss)` at `optionpool.go:182` matches on the *attributed* word (`bargainer`), which by construction never appears in the gloss it is attached to. This one is always on, not coincidence-dependent, and the atlas already calls inflected-form redirects "the common case" (`atlas/define.md:429`).

Fix (verified in a scratch worktree — it makes the collision disappear and the suite stays green):

```go
// buildPool
pe := ParseEntry(text)
if !entryDefines(k, pe) {
        continue // the glosses belong to the base word, not to k
}
pool = append(pool, optionCandidates(k, pe)...)
```

plus a gloss-level dedup in `PickOptions` (`used` is keyed on `Word` only at `pick.go:60`), which also covers two genuinely distinct words that share a gloss. Pin both halves: a `play_loop_test.go` row with a base and its derived form in one deck asserting no two options share a gloss, and a `pick_test.go` row where a pool candidate's gloss equals the target's.

**3. Important findings**

**`workshop/plans/000007-vocab-form-meaning-plan.md:125` — the Core concepts table names entities the head commit deleted or never added.** *(3rd finding in family `plan-artifact-must-match-tree`.)* Measured, all three introduced by the round that recorded the derivation rule:

- row `prng` / `shuffleOptions` → `cmd/define/play/pick.go` — `shuffleOptions` was **deleted by this commit** (BR-16's fix renamed it to the generic `shuffle`);
- `entryDefines` (`optionpool.go:134`), added by this commit, has **no row** — it appears only in the Revisions prose;
- the bullet at `plan.md:138` still calls the function `pickOptions`, which the round-3 sweep's own grep hit and did not fix, while the plan claims at `plan.md:442` that "re-running the sweep is now clean".

The rule was already stated in round 2 ("entity tables are DERIVED from the tree at the close") and then not executed — the same meta-failure BR-13 diagnosed. A discipline that has now failed twice in consecutive rounds should be made mechanical: this repo already owns the pattern (`doc_sync_test.go`), and the check is cheap — every backticked identifier in the Name column of the plan's Core-concepts / Integration-points tables must resolve in the file its "Lives in" column names. (The review contract nominally grades a table/code contradiction Critical; I am ranking it Important because it has no runtime effect and the Critical slot belongs to the defect that breaks a sitting.)

**4. Minor findings**

- **BR-13 is disposed `not-addressed`** — the five named instances are fixed, the class is not. Survivors the literal-string sweep could not reach: `optionpool_test.go:79-80` states *"the general one is the FIRST usable sense of the first block"* — the exact claim BR-13 named, missed only because the grep was `first sense of the first block` and this one has "usable" in it, in the file added this round to pin D4a; `pick_test.go:238` says *"The PRNG and the option shuffle are GOLDEN"* when only the PRNG sequence has literals (no test pins `(target, pool, seed)` → option order); `choice.go:70` says `Word` "is here for the reveal, which names what they picked", but `Reveal()` prints `optionLine(i, gloss)` and `Option.Word` is read by nothing in production; and `README.md:83-85` plus the atlas's new form-2.3 section both enumerate the fallback reasons as *one-word deck / all cross-references* when round 3 added a third. The enforceable rule: a literal grep finds the phrasing the last reviewer used, not the claim — derive the claim (`doc_sync_test.go`) where it is mechanical, and **delete** it rather than restate it where it is not.
- **`cmd/define/play_loop.go:355`** — `gradePrompt`'s `if q == nil` is unreachable: `draw` returns at `play_loop.go:306` when `Current()` is nil, and the only other caller is `doc_sync_test.go:60`, which always passes a form. *(2nd in family `unreachable-branch`; the rule: a guard whose precondition every caller already establishes is dead code that reads as protection.)*
- `buildPool` takes the whole `deps` struct to use one field (`d.dict`); a `Dictionary` parameter would make its purity boundary obvious.
- Due words that are also sampled into the pool are looked up and parsed twice. Inside the declared envelope, noted only because `poolCap` is tuned against it.

**5. Test coverage notes**

- The bug in §2 is invisible to the suite because no test asserts **option-set distinctness**. `TestPickOptionsHasOneAnswer` checks that the `Correct` option carries the target's gloss but never that no *distractor* does; `TestPickOptionsNeverRepeatsAWord` dedups on `Word`, which is exactly the key the defect slips past. One assertion — "no two options in a set share a gloss" — added to both would have caught it.
- Everything else about the pool path is covered end to end: cost envelope with a counting seam, axis selection derived from the `Axis` set, sitting-level variety (≥8 distinct sets over 20 questions), offline sitting with a panicking model seam, and the axis reaching a real event file.
- I could not exercise `TestPTYPlayChoiceOffersOptionsAndRecordsTheAxis` — no pty available here, so it skips. The issue's Log claims a real-terminal run; that claim is unverified from this seat.

**6. Architectural notes**

- **ARCH-DRY — pass.** BR-16 is genuinely closed: `shuffle[T any]` at `pick.go:156` is called from both sites and `shuffleOptions` is gone from the tree. `SampleStrings` correctly reuses `play`'s PRNG rather than growing a second one in `main`.
- **ARCH-PURE — pass.** `play` stays import-free; all prose parsing is in `main`; `choiceFor`/`PickOptions`/`readGloss` are pure and tested without IO. The dictionary is the only injected seam.
- **ARCH-PURPOSE — flag.** See §2. The purpose is "four options, exactly one correct"; the shadow-sweep over consumers of "the entry must define the word" finds one derived (`targetCandidate`) and one hand-waved (`optionCandidates`). Done-when 1 is ticked in the issue while a reachable deck makes it false.
- **ARCH-MOCK — pass.** The committed corpus + `fakeDictionary` back every form test through the same seam production uses; the accent-insensitive miss path reuses the production predicate.
- **ARCH-CONSTRAINTS — pass.** `poolCap` is declared, implemented, and enforced by a counting seam that now also asserts the output is non-degenerate. Nothing new sits on the per-keystroke path.

**7. Plan revision recommendations**

Two `## Revisions` entries, both under a round-4 heading:

1. *"The entry-defines gate belongs to every site that builds option material, not just the target."* Record that `buildPool`/`optionCandidates` was the second consumer, that the near-synonym guard is blind for redirected candidates because it matches the attributed word, and that the fix is a gate at pool-build plus a gloss-level dedup in `PickOptions`. Add a Done-when row: *"no two options in a set carry the same gloss"*, pinned by the new `play_loop_test.go` and `pick_test.go` rows.
2. *"The Core concepts table drifted in the round that recorded the rule for keeping it current."* Correct the `shuffleOptions` row to `shuffle`, add an `entryDefines` row, rename the `pickOptions` bullet, and replace the derivation *discipline* with a derivation *check* — state the mechanism (every Name-column identifier must resolve in its Lives-in file) rather than restating the instruction that has now failed twice.

```findings
dispose:
  - id: BR-13
    disposition: not-addressed
    note: |
      Named instances fixed; four survivors remain because the sweep grepped literal phrasings, not claims.
  - id: BR-15
    disposition: addressed
    note: |
      Verified by reverting the gate in a scratch worktree - the bargainer row goes red without it.
  - id: BR-16
    disposition: addressed
    note: |
      shuffle[T any] is called from both sites; shuffleOptions is gone from the tree.
findings:
  - id: new
    severity: Critical
    family: answer-must-define-the-prompted-word
    title: |
      buildPool is not gated by entryDefines, so a redirect puts a base entry's gloss into questions under the derived word
    detail: |
      The round-3 fix gated targetCandidate/choiceFor and left optionCandidates via buildPool
      ungated - the instance, not the class. The rule: every gloss entering a question,
      target OR distractor, must come from an entry that defines the word it is attributed
      to, and no two options in one set may share a gloss. Measured over the committed
      corpus with deck {bargain, bargainer, mesa, quokka} (bargainer.txt IS the bargain
      entry, which is what NOAD returns for both): the "bargain" question offers options 1
      and 2 byte-identical, option 2 marked Correct:false, so picking it records a miss with
      a fabricated axis and demotes the word; the mesa and quokka questions each carry the
      same gloss twice. Separately and always-on, crossReferenced (optionpool.go:182)
      matches on the attributed word, which by construction never appears in the gloss it is
      attached to, so the near-synonym guard is blind for every redirected candidate.
      Fix verified in a scratch worktree - skip a pool word when !entryDefines(k, pe), and
      dedup PickOptions on gloss as well as Word (pick.go:60). Pin with a play_loop_test.go
      row putting a base and its derived form in one deck, and a pick_test.go row where a
      pool candidate's gloss equals the target's.
  - id: new
    severity: Important
    family: plan-artifact-must-match-tree
    title: |
      The plan's Core concepts table names shuffleOptions, which this commit deleted, and omits entryDefines, which it added
    detail: |
      Third finding in this family, so the deliverable is the mechanism rather than the rows.
      Measured drift, all introduced by the round that recorded the derivation rule -
      plan.md:125 names `prng` / `shuffleOptions` at play/pick.go but BR-16's fix renamed it
      to the generic `shuffle`; entryDefines (optionpool.go:134) has no row and appears only
      in the Revisions prose; plan.md:138 still calls the function `pickOptions`, a hit the
      round-3 sweep's own grep returned while plan.md:442 claims the sweep is clean. Round 2
      recorded "entity tables are DERIVED from the tree at the close" and round 3 changed
      the tree without re-running it. Make it mechanical, as doc_sync_test.go already does
      for the README and atlas - every backticked identifier in the Name column must resolve
      in the file its Lives-in column names.
  - id: new
    severity: Minor
    family: unreachable-branch
    title: |
      gradePrompt's nil guard is dead - draw already returns when Current() is nil
    detail: |
      play_loop.go:355. draw returns at play_loop.go:306 when s.Current() is nil, and the
      only other caller is doc_sync_test.go:60, which always passes a form. Second in this
      family after BR-9's Choice.Keys() branch, so the rule is the deliverable: a nil/empty
      guard whose precondition every caller already establishes is dead code that reads as
      protection, and it hides which layer actually owns the invariant.
```

---

## Re-review — 2026-08-30T21:29:47-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 7 — review form 2.3: meaning multiple choice from the local deck |
| repo | tools |
| issue file | workshop/issues/000007-vocab-form-meaning.md |
| boundary | whole-issue close |
| milestone | — |
| window | a8962469ff08154f5377a7572586c575c2d4610f..67f76c660faed3c77814ef61c11c38569f2e1c40 |
| command | sdlc close --issue 7 |
| reviewer | claude |
| timestamp | 2026-08-30T21:29:47-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Form 2.3 is genuinely built, wired and well tested: `go build`, `go vet`, `gofmt -l`, `go test ./cmd/define/...` and `-race` are all clean here, `go list` confirms `play` still imports nothing, and this round's two headline claims check out under mutation — removing either `entryDefines` call turns `TestARedirectSuppliesNoOptionMaterialAtAll` and `TestNoCandidateEverCarriesAnotherWordsGloss` red, and breaking a Core-concepts Name cell now fires `TestPlanTablesNameEntitiesThatExist` (verified in a scratch worktree with `git init`). What blocks SHIP is that **BR-17 stated a two-clause rule and only the first clause was implemented**. The second — *"no two options in one set may share a gloss"*, with the `pick_test.go` pin the finding named — was not written, and it is still reachable in production: with a deck holding both `jalapeño` and `jalapeno` (the unaccented-typing path `#29` exists for, and which the real dictionary resolves to the same entry), the `jalapeno` question offers option 2 `Correct:true` and option 3 `Correct:false` with **byte-identical glosses**, so picking option 3 records a miss with a fabricated axis and demotes the word. `PickOptions` still dedups on `Word` only (`play/pick.go:46`). BR-13, BR-18 and BR-19 also carry survivors, measured below.

### 1. Strengths

- **`cmd/define/optionpool.go:80-90, 116-119`** — moving the guard from the caller into *both producers* is the right structural answer to "the instance, not the class", and the comment says exactly why a caller-level check failed. Mutation-verified: deleting either guard turns two tests red, and `TestNoCandidateEverCarriesAnotherWordsGloss` states the property over the whole corpus with a `checked == 0` vacuity trap, so a future fixture cannot reintroduce it silently.
- **`cmd/define/repo_guard_test.go:757-800`** — `TestPlanNamedTestsExist` was globbing `cmd/define/*_test.go` flat, so six tests this plan pins in `play/` read as nonexistent. Fixing the *guard* rather than editing the plan to appease it is the correct call, and the comment ("a guard that fails on the arrangement the architecture asks for teaches people to weaken the guard") is the right generalisation. ARCH-PURE.
- **`workshop/lessons.md:2644-2675`** — "an unticked checkbox can silently DISABLE a repo guard" is the highest-value artifact in this window. Diagnosing *why the guard never fired* instead of hand-fixing a fourth round of rows is exactly ARCH-PURPOSE's class-over-instance discipline.
- **`cmd/define/capture.go:141-145` + `store/event.go:32-42`** — D8 is enforced by the type (`AxisNone.String() == ""` → `omitempty` drops it) rather than by a branch a caller can forget, and `Missed` sits above `At` with the torn-record reason restated at both ends.
- **`cmd/define/doc_sync_test.go:40-77`** — the README prompt guard iterates shipped forms and composes the line the way `draw` does, including a duplicate-prompt check, and states its residual instead of hiding it.

### 2. Critical findings

**BR-17's second clause is unimplemented and reachable — `cmd/define/play/pick.go:46`.** `used := map[string]bool{target.Word: true}` keys on `Word` only; nothing anywhere compares glosses. Reproduction (measured, in a scratch copy of HEAD):

```go
d, opt, _ := playRig(t, "jalapeño", "jalapeno", "sycophantic", "quokka", "mesa", "parrot", "concrete")
// question "jalapeno" →
//   opt2 word=jalapeno  correct=true  "a very hot green chili pepper, used especially in Mexic…"
//   opt3 word=jalapeño  correct=false "a very hot green chili pepper, used especially in Mexic…"
```

Both deck words survive `entryDefines` (via `differsOnlyByDiacritics`, `optionpool.go:164`), `store.Key` folds case and whitespace but *not* diacritics (`store/word.go:29-31`), and `crossReferenced` cannot see it because neither headword appears in the shared gloss. The deck shape is ordinary: `define jalapeno` and `define jalapeño` are two `Upsert`s under two keys (`capture.go:111`), and the accent-insensitive lookup is documented production behaviour (`dict_fake_test.go:67-72`, pinned live by `TestLiveDictionaryResolvesAnUnaccentedQuery`). *Fix:* dedup on gloss as well as word in `PickOptions` — including against `target.Gloss` — and add the `pick_test.go` row BR-17 asked for.

### 3. Important findings

**Done-when row 1's "red when" is false of the test it names — `workshop/plans/000007-vocab-form-meaning-plan.md:178`.** The row claims `TestPickOptionsHasOneAnswer` goes red when "a second option's gloss is the target's". Mutation-verified: changing `testPool()`'s first candidate gloss to `target.Gloss` leaves the test **PASS** — it counts `Correct` flags and never compares distractor glosses. This is the 3rd finding in `guard-passes-without-the-property`; see the machine block for the rule.

**The README and atlas enumerate the form-2.1 fallback causes and omit the one two Critical rounds produced.** `cmd/define/README.md:78-81` says the fallback is for "a one-word deck, or a word whose entry is nothing but cross-references"; `atlas/define.md:2007` covers only "below two options it is not a question". `choiceFor`'s own doc (`optionpool.go:173-176`) names three causes. A learner reviewing `bargainer` gets Recall and nothing user-facing explains why. 4th in `docs-restate-behaviour-inaccurately`.

### 4. Minor findings

- `cmd/define/optionpool_test.go:252-258` — the "the guard is too broad" check is nested under `if err == nil`, and `d.Lookup("bargain")` **always errors** (measured: the corpus has `bargainer.txt`, no `bargain.txt`, and the accent fallback does not match). The check never runs.
- No live conformance row measures the derivative-redirect model. The fake models it with one fixture; `entryDefines` refuses 1 of 34 corpus entries (measured), and nothing bounds the rate against the real dictionary, which `entryDefines` can silently push to form 2.1. The live dictionary is unreachable in this environment (`systemDictionary` → "every active dictionary"), so I could not measure it.
- `TestSittingCostIsBoundedByTheCap` sits exactly at its bound (40 pool + 5 due = 45 ≤ 45) with zero slack — correct, but worth knowing it has no headroom.
- The Done-when table lists row 9 before row 8 (`plan.md:185-186`).

### 5. Test coverage notes

- The suite is strong and mostly property-shaped: `TestPickOptionsVariesTheDistractorsAcrossASitting`, `TestNoCandidateEverCarriesAnotherWordsGloss` and `TestYAMLWritesAtLastWhateverFieldsAreSet` all assert properties the defect cannot satisfy rather than restating the implementation.
- The gap the Critical exposes is systematic: **no test anywhere compares two options' glosses.** `TestPickOptionsHasOneAnswer` counts flags, `TestPickOptionsNeverRepeatsAWord` compares words. A `for i, j` gloss-inequality assertion inside `PickOptions`' result would have caught both the `bargainer` symptom (round 4) and the `jalapeño` one (still live).
- `TestPTYPlayChoiceOffersOptionsAndRecordsTheAxis` skips here (no pty). Its D8 identity assertion reads correct, but that half rests on the operator's real-terminal run in the issue Log.
- `gradePrompt`'s nil branch is unreached by the entire suite — verified by replacing it with a `panic` and running `./cmd/define/...` clean.

### 6. Architectural notes

- **ARCH-DRY — pass.** `shuffle[T any]` (`pick.go:156`) serves both call sites; `SampleStrings` is exported so `main` shares `play`'s PRNG rather than growing a second. `optionCandidates`/`targetCandidate` implement genuinely different selection rules and are rightly separate.
- **ARCH-PURE — pass.** `go list -f '{{join .Imports}}' ./cmd/define/play` returns empty. All prose handling is in `main`; `play` receives finished `Candidate`s. `buildPool` is the only IO in `optionpool.go`.
- **ARCH-PURPOSE — flag (the Critical).** Shadow-sweep of this window's single sources: `Question.Keys()` → `draw` and the README both derive (`doc_sync_test.go`); the `Axis` set → `distractorAxes` and `String()` both derived from `numAxes`; the plan's Core-concepts Name column → now derived (`TestPlanTablesNameEntitiesThatExist`, verified firing). The hand-maintained restatements still standing are the plan's `## Tasks` bullets (`senseLabel`, `pickOptions`) and the README/atlas fallback enumeration. And BR-17's rule was answered at the clause the commit message narrated, not the clause the finding stated — the instance again, one level up.
- **ARCH-MOCK — flag (Minor).** `countingDict` wraps the same seam production uses, the pty test drives the real binary against a real deck, and the corpus is committed. Missing: a live conformance row for the redirect behaviour this issue now gates on.
- **ARCH-CONSTRAINTS — pass.** `poolCap = 40`, pool built once per sitting, `TestSittingCostIsBoundedByTheCap` enforces `lookups ≤ poolCap + count` **and** asserts a `*play.Choice` was produced, so a bound-satisfied-by-zero build fails. Nothing new on the keystroke path.

### 7. Plan revision recommendations

1. **`## Revisions` — "the option-set invariant has two clauses, not one."** Record that a gloss must come from an entry that defines its word *and* that no two options in a set may share a gloss; name `jalapeño`/`jalapeno` as the measured second instance and `differsOnlyByDiacritics` + non-folding `store.Key` as the mechanism. Fix Done-when row 1's `red when` so it is a mutation the named test actually fails on.
2. **`## Tasks` — rewrite T1 and T3 to name `readGloss` and `PickOptions`.** The issue's own `## Plan` T1 already does this correctly (`issue:138`); the plan doc was never mirrored, and neither `TestPlanTablesNameEntitiesThatExist` (Name column only) nor `TestARemovedDeclarationIsSweptOrRetired` (only names the tree once declared) can reach a task bullet.
3. **`## Revisions` — close BR-13 as a sweep with its enumeration written down**, since round 4 disposed it `not-addressed` for the third consecutive round without touching any of the four files it names.

```findings
dispose:
  - id: BR-13
    disposition: not-addressed
    note: |
      3 of 7 named instances survive; the round-4 commit touched none of their files.
  - id: BR-17
    disposition: not-addressed
    note: |
      Clause 1 fixed and mutation-verified; clause 2 (no two options share a gloss) unimplemented and measured reachable.
  - id: BR-18
    disposition: not-addressed
    note: |
      Mechanism delivered and verified firing, but the finding's own named instance (plan.md pickOptions) sits outside its scope.
  - id: BR-19
    disposition: not-addressed
    note: |
      Branch still at play_loop.go:359; replacing it with a panic leaves the whole suite green.
findings:
  - id: new
    severity: Important
    family: guard-passes-without-the-property
    title: |
      Done-when row 1 claims a red-when that TestPickOptionsHasOneAnswer cannot deliver
    detail: |
      3rd finding in family guard-passes-without-the-property, so the deliverable is the
      rule: a Done-when "red when" cell is a claim about a MUTATION, and it must be
      verified by performing that mutation, not by reading the test. Measured -
      plan.md:178 says TestPickOptionsHasOneAnswer goes red when "a second option's gloss
      is the target's"; setting testPool()[0].Gloss = target.Gloss in a scratch copy of
      HEAD leaves the test PASS, because it counts Correct flags and never compares
      distractor glosses. The repo already mechanises test EXISTENCE
      (TestPlanNamedTestsExist) and Name-column resolution
      (TestPlanTablesNameEntitiesThatExist); the red-when column, which is where a plan
      makes its load-bearing claim, is checked by nobody. The cheapest honest form of the
      rule: every row's red-when must be reproduced once, by hand, at the close, and the
      reproduction recorded beside the row - or the cell weakened to what the test does
      assert. This row's failure is the same defect as BR-17's open clause, which is how
      a false red-when hides a live bug.
  - id: new
    severity: Important
    family: docs-restate-behaviour-inaccurately
    title: |
      README and atlas enumerate the form-2.1 fallback causes and omit the derivative redirect
    detail: |
      4th finding in family docs-restate-behaviour-inaccurately, so the deliverable is the
      rule, not the two lines: a doc sentence that ENUMERATES ("X, or Y") is a closed
      claim about the code and must be derived from the same list the code branches on,
      or be written open ("for example"). Measured - choiceFor (optionpool.go:173-176)
      names three refusals; README.md:78-81 names two ("a one-word deck, or a word whose
      entry is nothing but cross-references") and atlas/define.md:2007 names one ("below
      two options it is not a question"). The third, added by BR-15/BR-17 across two
      rounds and given its own Core-concepts row, is user-visible: a learner reviewing
      `bargainer` silently gets Recall. Two enumerations of the same fact in two files,
      both hand-maintained, is the shape doc_sync_test.go already fixed for the prompt
      lines - the fallback reasons are the next candidate for the same treatment. The
      sibling instance in the same family, still open: README.md:68 says a word is "never"
      offered as a distractor against a word whose gloss mentions it, while mentions()
      returns false for any headword under 6 characters.
  - id: new
    severity: Minor
    family: conditional-assertion
    title: |
      The over-breadth half of TestARedirectSuppliesNoOptionMaterialAtAll never executes
    detail: |
      2nd finding in family conditional-assertion, so the rule is the deliverable: a test
      may not nest an assertion under a runtime condition no fixture can satisfy - the
      negative branch must Fatal, or the condition must go. Measured -
      optionpool_test.go:253 does `base, err := d.Lookup("bargain"); if err == nil { ... }`,
      and the corpus has bargainer.txt with no bargain.txt, so Lookup returns ErrNoEntry
      on every run and the "the guard is too broad" check has never run. Sibling shape,
      same round: the round-2 fix for BR-8 correctly replaced an `if n > 0` gate with an
      unconditional identity, and this file reintroduced the pattern nine lines later. The
      property itself is in fact covered elsewhere (TestSittingCostIsBoundedByTheCap
      asserts a *play.Choice is produced; TestASittingFallsBackForAnEntryThatCannotBeAsked
      asserts the rest of the deck gets Choice), so the block should be deleted rather
      than repaired - a dead check that duplicates a live one is worse than neither.
  - id: new
    severity: Minor
    family: fake-behaviour-lacks-live-conformance
    title: |
      The derivative-redirect model that entryDefines gates on has no live conformance check
    detail: |
      ARCH-MOCK. entryDefines is a new gate that can silently route a word from form 2.3 to
      form 2.1, and the behaviour it models - NOAD redirecting a derived form to its base
      headword - is represented by exactly one committed fixture (bargainer.txt). Measured
      over the corpus: 1 of 34 entries refuses. Nothing measures the rate against the real
      dictionary, so an entryDefines that is too strict on inflected, multi-word or variant
      heads would quietly disable the form for a large share of a real deck while every test
      stays green. The repo already has the seam and the cadence for this
      (live_property_test.go walks /usr/share/dict/words under `-tags conformance`); a row
      there reporting the refusal rate and a sample of refused words would validate the model
      and bound the degradation. I could not measure it here - systemDictionary returns
      "every active dictionary" and Lookup finds nothing in this environment.
```

---

## Re-review — 2026-08-30T22:01:29-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 7 — review form 2.3: meaning multiple choice from the local deck |
| repo | tools |
| issue file | workshop/issues/000007-vocab-form-meaning.md |
| boundary | whole-issue close |
| milestone | — |
| window | a8962469ff08154f5377a7572586c575c2d4610f..def5aea43e28874c5a91864ca47ef5f4511eb95a |
| command | sdlc close --issue 7 |
| reviewer | claude |
| timestamp | 2026-08-30T22:01:29-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Round 5 closes BR-17 properly — both clauses are implemented and both are mutation-verified (I reverted each in a scratch worktree and watched the named test go red), BR-18's guard is now armed and fires within seconds on a bogus table row, BR-19's dead branch is gone, and BR-22's conditional assertion is unconditional. The suite is green (`go test ./...`, 107s) and `play` still imports nothing. What blocks SHIP is that the option-set invariant was fixed at its two *proxies* and not at its key: `PickOptions` dedups on `Word` and now on `Gloss`, but two deck keys differing only by diacritics resolve to one dictionary entry (`differsOnlyByDiacritics`, `optionpool.go:164` — the very mechanism round 5 relied on), and if that entry has more than one usable sense the two keys yield *different* glosses under *different* words. I reproduced a live option set where both "existing in a material or physical form; not abstract" (Correct) and "form (something) into a mass; solidify" (marked wrong, axis `register`) are offered for the prompted word — the same harm round 5 fixed, reached by the sibling path. Secondarily, BR-20 and BR-21 each had their named instance fixed and their stated *rule* left unapplied.

**1. Strengths**

- `PickOptions`' seeded permutation (`pick.go:75-88`) is the right shape: a permutation rather than reordering the caller's slice, so `choiceFor` can reuse one sitting-wide pool without each question's selection depending on the previous ones. The comment explains why, and `TestPickOptionsVariesTheDistractorsAcrossASitting` pins it with a threshold (≥8 of 20) that sits far from both the correct and the buggy value.
- `TestSittingCostIsBoundedByTheCap` (`optionpool_test.go:30`) is the model for an ARCH-CONSTRAINTS pin: it asserts the upper bound *and* that the output actually contains a form 2.3 question, so "the pool was capped to nothing" cannot pass as "working". A bound alone is satisfied by zero, and the test says so.
- `Missed` as an optional interface (`play/session.go:266-282`) keeps `Apply` free of any form name while still carrying the axis out — verified: `awk '/^func Apply/,/^}/' | grep 'Choice\|Recall'` is empty.
- `TestYAMLWritesAtLastWhateverFieldsAreSet` (`store/yaml_test.go:434`) derives the ordering property from the written record rather than comparing two named keys, which is what makes it survive the field this issue adds.
- `entryDefines` walking the `HeadWord`/`HeadOther` run instead of `Headword()` (`optionpool.go:150-168`) is correct and measured: `hot dog`, `a priori`, `jalapeño`, `MacBook` all pass, `bargainer` refuses — I re-measured the whole corpus at 1 refusal in 34.

**2. Critical findings**

- **`cmd/define/play/pick.go:61,90` — an option set may still carry two options that both define the prompted word.** This is the **3rd finding in family `answer-must-define-the-prompted-word`**, so the deliverable is the rule, not this instance. The rule: **an option set may contain at most one option per source ENTRY.** `Word` and `Gloss` are both proxies for entry identity, and each fails on a case the other doesn't — `Word` failed on `jalapeño`/`jalapeno` (BR-17 clause 2), `Gloss` fails as soon as the shared entry has more than one usable sense. Measured: `differsOnlyByDiacritics` (`optionpool.go:164`) admits both deck keys against the one entry; `optionCandidates` (`optionpool.go:80`) then returns one candidate per axis, so 17 of 34 corpus entries yield ≥2 differently-glossed candidates; `choiceFor`'s `c.Word == target.Word` filter does not match across the spelling variants; `crossReferenced` is blind because neither headword is in either gloss. Reproduced end to end in a scratch worktree using the committed `concrete` entry under a second accented deck key — the set offered the target's sense as Correct and the entry's own `register` sense as a distractor, so a learner picking it is recorded as a miss with a fabricated `register` axis and the word is demoted for a defensible answer. *Fix sketch:* give `play.Candidate` (and the target) an entry-identity field — the parsed `Headword()`, or the key the lookup resolved to — set in `optionCandidates`/`targetCandidate`, and dedup `PickOptions` on it; that subsumes both existing dedups instead of adding a third. Pin with a `pick_test.go` row where two candidates share an entry id and differ in gloss, and an `optionpool_test.go` row driving a multi-sense corpus entry through two deck keys.

**3. Important findings**

- **BR-20 re-raised (`workshop/plans/000007-vocab-form-meaning-plan.md:184`) — the red-when rule was never applied.** Round 5 fixed row 1, which is what BR-20 named, and neither stated the rule above the table nor reproduced any other row's mutation. Measured: row 7's second predicate — *"no form name (`Choice`, `Recall`) appears inside `playSession` or `play.Apply` — a grep, not a file-state claim"* — is run by nothing in the tree. I ran it by hand and it holds today, but no test goes red if it stops. That is the same defect the row's own preamble ("a pin is a predicate over behaviour") exists to forbid.
- **BR-21 re-raised (`atlas/define.md:2007-2018`) — the enumeration derives in one of the two files it drifted in.** `TestREADMENamesEveryFallbackReason` now checks `README.md` against `fallbackReasons`; the atlas's *"Three things send a word to form 2.1"* list is still hand-maintained, and it was the file that had drifted furthest (one of three). `doc_sync_test.go` already reads the atlas elsewhere (`TestAtlasDescribesEveryRenderOpt`), so this is one more loop in the same test. Separately, `fallbackReasons` (`optionpool.go:179`) sits *beside* `choiceFor`'s branches rather than being consumed by them, so it single-sources the docs but not the code — a fourth refusal added to `choiceFor` would not appear in the list.

**4. Minor findings**

- `cmd/define/play/recall.go:38` — `Keys()` was inserted between `Grade`'s doc comment and `Grade`, so `go doc Recall.Keys` prints Grade's contract and `Grade` is undocumented. (Verified with `go doc`.)
- `cmd/define/optionpool_test.go:79` — still claims the general candidate is "the FIRST usable sense of the first block"; the phrase wraps a line, which is why round 3's grep sweep missed it.
- `cmd/define/optionpool_test.go:141` — "choiceFor's two refusals" now that `fallbackReasons` declares three.
- `workshop/plans/…-plan.md` Core concepts — no row for `fallbackReasons`, a new declaration a guard now depends on.
- `cmd/define/pty_conformance_test.go:717` — "answer everything with `1`" assumes at least one miss; the outcome is a deterministic function of today's date via `seedFor(key, day)`, so ~0.1% of days would fail the `missed:` assertion for the wrong reason. Also assumes every one of the five words gets form 2.3; a live-dictionary fallback would stall the sitting rather than fail clearly.
- `cmd/define/play_loop_test.go:871` — `TestOptionCountGrowsWithTheDeck` indexes `qs[0]` without a length check.

**5. Test coverage notes**

Coverage of the *shipped* behaviour is genuinely strong: every seam BR-5 named now has a direct test, the goldens pin the PRNG and the hash, and the two round-5 fixes are mutation-verified (I confirmed both, independently). The gap is a class the tests keep re-discovering rather than enumerating: the option-set invariant is pinned by three tests that each assert one *proxy* (`…NeverRepeatsAWord`, `…NeverRepeatsAGloss`, `…HasOneAnswer`) and none that asserts the property — "no two options a learner can defend". A single property test over the corpus, in the shape of `TestNoCandidateEverCarriesAnotherWordsGloss`, would have caught the Critical above and would catch the next proxy failure too.

**6. Architectural notes**

- **ARCH-DRY — pass.** `shuffle` generic over `[]int` and `[]Option` closes BR-16; `SampleStrings` is exported precisely so `main` reuses `play`'s PRNG instead of growing a second. `isWordByte` (`glosslabel.go:215`) and `isBoundary` (`parse.go:876`) are different facts, not duplication.
- **ARCH-PURE — pass, and this is the cleanest part of the diff.** `go list -f '{{.Imports}}' ./cmd/define/play` is empty, `TestPlayPurity/imports` is green against an empty allowlist, and `pick_test.go` runs with `testing` alone. All prose handling sits in `main` behind the `Candidate` seam.
- **ARCH-PURPOSE — flag.** Two half-swept single-source changes: the fallback enumeration derives for the README and not the atlas, and the option-set fix reached two proxies and not the key. Both are the "instance, not the class" shape, and both are the *third* round on their family.
- **ARCH-MOCK — pass with a note.** No new external dependency; the committed corpus plus `dict_fake_test.go` back every form test, and `TestLiveDictionaryRedirectsADerivedForm` gives the new redirect model a live check in both directions (it skips here — NOAD is unavailable in this environment). BR-23's refusal-*rate* row was not delivered; the offline bound I measured is 1/34, with every risky head shape passing.
- **ARCH-CONSTRAINTS — pass.** `poolCap` is declared, enforced in `buildPool`, built once per sitting, and pinned in both directions.

**7. Plan revision recommendations**

1. `## Revisions` — **"the option-set invariant's key is the ENTRY, not the word or the gloss."** Record that `Word` and `Gloss` are proxies that each fail on a case the other covers, name the accented-variant-plus-multi-sense measurement (17/34 entries yield ≥2 candidates; `differsOnlyByDiacritics` admits both keys), and state the rule as one dedup on entry identity that subsumes the other two.
2. Done-when row 1 — widen the claim and its `red when` from "no two options with the same gloss" to "no two options from the same entry", and name the new pin.
3. Done-when — add the rule BR-20 asked for above the table (*a `red when` cell is a claim about a mutation and must be reproduced once, at close, or weakened*), and either weaken row 7's second predicate to what is actually checked or add the grep as a test.
4. Core concepts — add a row for `fallbackReasons` (`cmd/define/optionpool.go`, new, PURE), and record that the round-2 derivation procedure (`go doc -short`) returns nothing for package `main`, so it can only ever have covered `play/` — which is why the last two rounds' additions (`entryDefines`, `fallbackReasons`) both arrived unnoticed.

```findings
dispose:
  - id: BR-13
    disposition: not-addressed
    note: |
      Named instances fixed, but the rule's sweep is a line-oriented phrase grep and three claims stand: optionpool_test.go:79 still says "the FIRST usable sense of the first block" (the phrase wraps, so the grep could not see it); optionpool_test.go:141 says "choiceFor's two refusals" while fallbackReasons declares three; and play/recall.go:38 now carries Grade's doc comment verbatim, so `go doc Recall.Keys` prints Grade's contract and Grade is undocumented — a shape no phrase grep can find, but a ~20-line AST check ("a doc comment opens with the name of the declaration it precedes") finds it, and finds exactly one new instance in this window.
  - id: BR-17
    disposition: addressed
    note: |
      Both clauses mutation-verified in a scratch worktree: dropping the entryDefines gate from optionCandidates turns TestNoCandidateEverCarriesAnotherWordsGloss and TestARedirectSuppliesNoOptionMaterialAtAll red; reverting `free` to word-only dedup turns TestPickOptionsNeverRepeatsAGloss red. See the new finding for the sibling path the gloss key does not cover.
  - id: BR-18
    disposition: addressed
    note: |
      Ticking the plan's task list arms TestPlanTablesNameEntitiesThatExist; I injected a bogus Name-cell identifier and the guard failed within seconds, and TestPlanNamedTestsExist now walks the tree.
  - id: BR-19
    disposition: addressed
    note: |
      The guard is deleted and replaced by a comment naming the invariant's owner; draw returns at play_loop.go:306 when Current() is nil, confirmed.
  - id: BR-20
    disposition: not-addressed
    note: |
      Row 1 (the instance) was fixed; the rule was neither stated above the table nor applied. Measured: row 7's second predicate — "no form name appears inside playSession or play.Apply, a grep not a file-state claim" — is run by no test. I ran it by hand and it holds today, but nothing goes red if it stops.
  - id: BR-21
    disposition: not-addressed
    note: |
      README now derives from fallbackReasons via TestREADMENamesEveryFallbackReason, but atlas/define.md:2007-2018 still hand-maintains the same closed enumeration — and the atlas was the file that had drifted furthest. doc_sync_test.go already checks the atlas elsewhere, so this is one more loop in the same test. Also fallbackReasons sits beside choiceFor's branches rather than being consumed by them. The README "never offered as a distractor" half is fixed.
  - id: BR-22
    disposition: addressed
    note: |
      The block is unconditional and now asserts on the same entry, which is the object that shows the gate/ban distinction. Swept the window's test files and the repo for the shape; every remaining `if err == nil` is an expected-error check.
  - id: BR-23
    disposition: addressed
    note: |
      TestLiveDictionaryRedirectsADerivedForm pins the model in both directions (skips here — NOAD unavailable). The refusal-RATE row asked for was not delivered; offline bound re-measured at 1 of 34, with hot dog / a priori / jalapeño / MacBook all passing.
findings:
  - id: new
    severity: Critical
    family: answer-must-define-the-prompted-word
    title: |
      Two options in one set can both define the prompted word, because dedup keys on Word and Gloss but not on the source ENTRY
    detail: |
      3rd finding in this family, so the deliverable is the rule: an option set may contain at most one option per source ENTRY. Word and Gloss are proxies that each fail where the other holds — Word failed on jalapeño/jalapeno (BR-17 clause 2), Gloss fails the moment the shared entry has more than one usable sense. Measured: differsOnlyByDiacritics (optionpool.go:164) admits both deck keys against one entry; optionCandidates (optionpool.go:80) returns one candidate per axis, and 17 of 34 corpus entries yield >=2 differently-glossed candidates; choiceFor's `c.Word == target.Word` filter does not match across the spelling variants and crossReferenced is blind because neither headword is in either gloss. Reproduced in a scratch worktree over the committed `concrete` entry under a second accented deck key: the set offered "existing in a material or physical form; not abstract" as Correct and the same entry's "form (something) into a mass; solidify" as a register distractor, so a learner picking it records a miss with a fabricated axis and the word is demoted for a defensible answer. Fix: carry entry identity (Headword(), or the resolved lookup key) on play.Candidate and the target, set it in optionCandidates/targetCandidate, and dedup PickOptions on it — subsuming both existing dedups rather than adding a third. Pin with a pick_test.go row where two candidates share an entry id with different glosses, and an optionpool_test.go row driving a multi-sense corpus entry through two deck keys.
  - id: new
    severity: Minor
    family: plan-artifact-must-match-tree
    title: |
      fallbackReasons has no Core-concepts row, and the recorded derivation procedure cannot see package main
    detail: |
      5th finding in this family, so the rule rather than the row. Round 2 recorded the derivation as `go doc -short` per touched package; that command returns nothing for package main, so the procedure could only ever have covered play/ — and both entities added since (entryDefines, fallbackReasons) live in cmd/define. entryDefines got a row only after a reviewer named it; fallbackReasons, which a doc guard now depends on, has none. BR-18's armed guard checks table-to-tree only. The mechanical form of the rule is a declaration scan of the named files (go/ast or a `^func|^var|^const|^type` grep) run against the Name column at close, in place of `go doc -short`.
  - id: new
    severity: Minor
    family: uncontrolled-test-input
    title: |
      The pty form-2.3 test's "at least one miss" assertion depends on today's date
    detail: |
      pty_conformance_test.go:717 answers every question with `1` and then requires a `missed:` line. The answer's slot is a deterministic function of seedFor(key, day), so on roughly 1 day in 1000 all five words put the answer in slot 1 and the assertion fails for the wrong reason. The same block also assumes all five words get form 2.3 — a live-dictionary fallback to Recall would stall the sitting rather than fail legibly. Drive the keys from the rendered option lines, or seed the sitting through a fixed clock.
```

---

## Re-review — 2026-08-30T22:38:27-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 7 — review form 2.3: meaning multiple choice from the local deck |
| repo | tools |
| issue file | workshop/issues/000007-vocab-form-meaning.md |
| boundary | whole-issue close |
| milestone | — |
| window | a8962469ff08154f5377a7572586c575c2d4610f..1363dcdb028e7a5456c413e2c0c1f0e9d85b4972 |
| command | sdlc close --issue 7 |
| reviewer | claude |
| timestamp | 2026-08-30T22:38:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 6's Critical (BR-24) is genuinely closed: `Candidate.Source` is set by both producers, `PickOptions` dedups on all three keys, and I mutation-verified the pin — reverting `free` to word+gloss turns `TestOneEntryMaySupplyOnlyOneOption` red at seed 0 with the exact diagnostic it promises. `go test ./...` is green. What stops SHIP being unconditional is not a wrong question reaching a learner; it is that this round's *own* claims about its pins do not survive checking. The second pin the commit message names (`TestNoQuestionDrawsTwoOptionsFromOneEntry`) passes with `Source` deleted from both producers, so "both mutation-verified" is false; `SampleStrings` — the one exported helper the plan itself calls "genuinely new downstream API" — can be replaced by an empty body with the entire suite still green; and the atlas was not touched this round, so it still describes a two-key dedup against a three-key implementation. Three of the six prior findings (BR-13, BR-20, BR-21, BR-25, BR-26 — five, in fact) were carried unmodified: round 6 fixed only BR-24. None of the remainder is a correctness defect, so the gate is not blocked, but the family that has now produced four findings is the one about guards that pass without the property, and this round added an instance rather than the rule.

## 1. Strengths

- **The root cause was actually found, not patched again.** `pick.go:22-42`'s `Source` doc and `workshop/lessons.md`'s "a deck holds KEYS, a dictionary holds ENTRIES" name the unit of meaning instead of a fourth symptom. `TestOneEntryMaySupplyOnlyOneOption` (`pick_test.go:314`) asserts all three keys as one property over 60 seeds rather than three separate tests — the right shape for a class fix.
- **`sourceOf`'s empty-Source fallback to `Word` (`pick.go:49-56`) is the correct defensive choice.** Without it, every `Source`-less candidate would collide under `""` and a test fixture would silently get one option; I confirmed `TestEveryAxisIsSelectable` and `TestPickOptionsVariesTheDistractorsAcrossASitting` both run on `Source`-less pools and still pass.
- **`TestYAMLWritesAtLastWhateverFieldsAreSet` (`store/yaml_test.go:434`) is a real property test.** It derives the key list from the written record rather than comparing `at:` against one named sibling, so it goes red for *any* field appended after `at:`, not just the one this issue added.
- **`TestPlanTablesNameEntitiesThatExist` is armed and I verified it.** Injecting `seedForZZZ` into the Core-concepts Name column fails within a second with the intended message. `inProgress` is computed per plan file, so `#38`'s unticked plan does not disarm `#7`'s.
- **`readGloss` (`glosslabel.go:113`) and its table (`glosslabel_test.go:13`) are pure, IO-free, and driven by real corpus strings.** ARCH-PURE holds: `play` still imports nothing, and every `main`-side "pure" entity is tested against read-only committed fixtures, not mocks.

## 2. Critical findings

None.

## 3. Important findings

**I-1 · `cmd/define/play/pick.go:227` — `SampleStrings` is pinned by nothing.** I replaced its entire body with `_ = n; _ = seed` and ran `go test ./cmd/define/...`: everything passed (the only two failures were `TestPlanTableStatusMatchesTheChangeWindow` / `TestARemovedDeclarationIsSweptOrRetired`, artifacts of my scratch git setup). With the sampler inert, `buildPool` takes the deck's first 40 keys in deck order forever — exactly the failure its own comment says it exists to prevent ("the same forty would make every sitting draw from one corner of the deck forever"). This is the 2nd finding in family `seeded-selection-not-just-ordering`, so the deliverable is the rule and its enumeration, not one test: **every site where the design claims the seed changes WHICH items are chosen rather than their order must have a test that goes red when the seeding is removed.** The enumeration for this issue is exactly two sites — `PickOptions`' index permutation (pinned, by `TestPickOptionsVariesTheDistractorsAcrossASitting`) and `buildPool`'s pool sample (unpinned). Fix sketch: a `pick_test.go` row asserting `SampleStrings` over a 100-element slice yields different 40-prefixes for two seeds, the same prefix for a repeated seed, and a permutation of the input (no loss, no duplication); plus a `buildPool` row that two pool seeds over one deck select different key sets.

**I-2 · `cmd/define/optionpool_test.go:306` — the sitting-level pin for BR-24 passes with and without the fix.** Deleting `Source: e.Headword()` from both producers in `optionpool.go` leaves `TestNoQuestionDrawsTwoOptionsFromOneEntry` green; so does reverting `free` to word+gloss. The reason is measurable: the fixture drives `jalapeño`/`jalapeno`, and that entry yields exactly **one** candidate (`optionCandidates` returns 1), so the two-senses-from-one-entry shape it claims to cover cannot occur. BR-24's own fix sketch asked for "an optionpool_test.go row driving a **multi-sense** corpus entry through two deck keys" — `concrete` yields 2 (general + register), which is what BR-24 reproduced with. Yet `plan.md:558` and commit `5db1d38`'s message both state "Both mutation-verified." **This is the 4th finding in family `guard-passes-without-the-property`.** Do not fix this one test — the rule is BR-20's, restated at the site that broke it this round: *a written claim that a test is mutation-verified is a claim about a mutation, and it is only true if the mutation was performed and the test observed red.* The mechanical form is cheap and this round is the argument for it: record the mutation performed and the test that went red, beside the claim, in the same edit that makes the claim. Fixing the fixture (a second accented deck key over `concrete`, as BR-24 did) is necessary too, but secondary.

**I-3 · `atlas/define.md:2032` — the atlas describes a two-key dedup and the code has three.** Round 6 touched no docs; the atlas paragraph still reads "**An option set also dedups on GLOSS, not only on word**", and `Source` / the KEYS-vs-ENTRIES distinction — new exported API on a package `#12` will consume, and new terminology by any reading of AGENTS.md §8 — appears nowhere in `atlas/`. **This is the 5th finding in family `docs-restate-behaviour-inaccurately`.** The rule is BR-21's and it is half-implemented: `TestREADMENamesEveryFallbackReason` derives the README from `fallbackReasons`, and `atlas/define.md` — the file that had drifted furthest and is the one this round left behind — derives from nothing. `doc_sync_test.go` already runs atlas loops (`TestAtlasDescribesEveryRenderOpt`, the raw-notation ratchet), so extending the same loop to the atlas is one function, not a new mechanism. Until it exists, every atlas sentence about form 2.3 is a hand-maintained restatement and this finding will recur.

**I-4 · BR-20 remains open and this round supplied a fresh instance.** No rule was stated above the Done-when table, no red-when reproduction was recorded beside any row, and row 7's second predicate — "no form name appears inside `playSession` or `play.Apply`, a grep not a file-state claim" (`plan.md:184`) — is run by no test; I grepped and it holds today, but nothing goes red if it stops. See I-2 for the instance.

**I-5 · BR-21 remains open.** README derives; `atlas/define.md:2007-2018` hand-maintains the same closed enumeration (it is *currently* correct, which is the trap); and `fallbackReasons` (`optionpool.go:170`) is still adjacent to `choiceFor`'s branches rather than consumed by them — it has zero production call sites, so the "declared list" is a second copy of the model, not the source the code derives from (ARCH-PURPOSE: the source is documentation a surface restates, not enforcement).

## 4. Minor findings

- `pick.go:40` — `Source: e.Headword()` is not entry identity. Measured: `Headword()` returns `"hot"` for `hot dog` and `"a"` for `a priori` (it is `fields[0]`, which `entryDefines`' own doc at `optionpool.go:139` says is why `Headword()` "is NOT the check"). Two entries sharing a first head word collide and one loses a distractor. No wrong-answer path — over-dedup only — but the field's doc claims it "identifies the dictionary ENTRY", which is false for every multi-word headword. **4th finding in family `answer-must-define-the-prompted-word`**; the rule is that an identifier used as an identity key must be injective over what it identifies, and this one is documented three files away as not being. BR-24 offered the alternative: the resolved lookup key, or `Headword()+Homograph()+Syllables()`.
- BR-13 remains open, with one instance added since round 6 measured it: `optionpool_test.go:79-80` still says the general candidate is "the FIRST usable sense of the first block" (it is the first *unlabelled* usable sense across all blocks — the exact claim the round-3 sweep was written to kill, missed because the sweep grep is line-oriented and the phrase wraps); `optionpool_test.go:141` says "choiceFor's two refusals" while `fallbackReasons` declares three; `play/recall.go:32-38` puts `Grade`'s doc comment on `Keys` — I ran `go doc ./cmd/define/play Recall.Keys` and it prints Grade's contract while `Grade` is undocumented; and `choice.go:69` says "Word is here for the reveal, which names what they picked" while `Reveal` renders only `optionLine(i, o.Gloss)` and `Option.Word` is read at zero production sites.
- BR-25 remains open: `fallbackReasons` still has no Core-concepts row, and the declaration-scan procedure was not implemented. Round 6 added a row for the entity it introduced (`Candidate`/`sourceOf`) and did not sweep for the one already missing — instance, not class.
- BR-26 remains open: `pty_conformance_test.go:740` still requires a `missed:` line after pressing `1` five times, and the answer's slot is `seedFor(key, day)`.
- `optionpool.go:42-46` — `SampleStrings` mutates in place and returns nothing, so every caller must remember a second statement (`keys = keys[:poolCap]`) or silently look up the whole deck. `buildPool`'s omission would be caught by `TestSittingCostIsBoundedByTheCap`; `#12`'s would not. A `Sample(ss, n, seed) []string` returning the prefix removes the obligation.

## 5. Test coverage notes

- Verified red-when by mutation, in a scratch tree at HEAD: `TestOneEntryMaySupplyOnlyOneOption` ✅ red; `TestNoQuestionDrawsTwoOptionsFromOneEntry` ❌ stays green under both mutations; `SampleStrings` no-op ❌ whole suite green; `TestPlanTablesNameEntitiesThatExist` ✅ red on an injected bogus Name cell.
- `go test ./...` green (`cmd/define` 107s). `-tags conformance`: `TestPTYPlayChoiceOffersOptionsAndRecordsTheAxis` and `TestLiveDictionaryRedirectsADerivedForm` both **skip** here (no pty, no NOAD), so the live half of ARCH-MOCK is asserted but unobserved in this environment — worth running on the operator's machine before the close, since it is the plan's manual verification.
- Coverage is otherwise strong: `TestSittingCostIsBoundedByTheCap` correctly refuses to be satisfied by zero, `TestASittingFallsBackToRecall` exercises the fallback *and* answers it, and `TestAMissRecordsTheAxisItChose` drives the loop rather than `CaptureReview` so the wiring is what is pinned.

## 6. Architectural notes

- **ARCH-DRY — pass.** The generic `shuffle` consolidates the two Fisher-Yates bodies; `sourceOf` is one helper; `differsOnlyByDiacritics` is reused rather than re-implemented in `entryDefines`. The one residual duplication is the enumeration in I-5 (`fallbackReasons` beside `choiceFor`'s branches).
- **ARCH-PURE — pass.** `play` imports nothing (guard intact with an empty allowlist), all its tests run without IO, and the prose work stayed in `main`. `main`'s pure entities are tested against read-only committed fixtures, which is data, not mocking.
- **ARCH-PURPOSE — flag (I-3, I-5).** Shadow-sweep of the single sources: `Axis`+`numAxes` → derived (`TestEveryAxisIsSelectable`) ✅; the loop's prompt → derived (`doc_sync_test.go` over shipped forms) ✅; `fallbackReasons` → README derived ✅, atlas hand-maintained ❌, `choiceFor` not derived ❌; the dedup-key rule → no doc consumer at all ❌. Two of four consumers are still hand-maintained restatements.
- **ARCH-MOCK — pass with a caveat.** The fake dictionary models the accent-insensitive resolve using the production predicate, and `TestLiveDictionaryRedirectsADerivedForm` gives the redirect model its live conformance check in both directions. Caveat: the fake's corpus does not contain a multi-sense entry reachable under two deck keys, which is precisely why I-2's test cannot fail.
- **ARCH-CONSTRAINTS — pass.** `poolCap = 40` is declared with a basis and *enforced* by `TestSittingCostIsBoundedByTheCap`, which also refuses the zero-lookup degenerate. Nothing new is on the per-keystroke path; pool construction is once per sitting, not per question.

## 7. Plan revision recommendations

- **A `## Revisions` entry correcting the round-6 claim.** `plan.md:558` says `TestOneEntryMaySupplyOnlyOneOption` and `TestNoQuestionDrawsTwoOptionsFromOneEntry` were "both mutation-verified"; the second is not discriminating (measured: green with `Source` deleted from both producers, and the `jalapeño` entry yields one candidate). Record which mutation was performed for each, and state that the sitting-level pin needs a multi-sense entry under a second deck key before it asserts anything.
- **A Done-when table amendment implementing BR-20's rule**, not another corrected cell: add the rule sentence above the table, and a reproduction column or note per row naming the mutation performed and the observed failure. Row 7's grep predicate should either become a test or be struck.
- **A Core-concepts row for `fallbackReasons`** (`cmd/define/optionpool.go`, new, PURE), and a `## Revisions` note that the derivation procedure recorded in round 2 (`go doc -short`) returns nothing for package `main` and is replaced by a declaration scan of the named files.
- **A Core-concepts amendment or `## Revisions` note for `Candidate.Source`** stating what it actually is — the entry's first head word, via `Headword()` — rather than "entry identity", or changing the key so the table's claim becomes true.

---

## Re-review — 2026-08-30T23:05:02-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 7 — review form 2.3: meaning multiple choice from the local deck |
| repo | tools |
| issue file | workshop/issues/000007-vocab-form-meaning.md |
| boundary | whole-issue close |
| milestone | — |
| window | a8962469ff08154f5377a7572586c575c2d4610f..af2d97347c6437b30466a495206b5045a7f2e332 |
| command | sdlc close --issue 7 |
| reviewer | claude |
| timestamp | 2026-08-30T23:05:02-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The Critical from round 6 (BR-24) is genuinely closed: `Candidate.Source` is set by both producers, `PickOptions` dedups on it, and I mutation-verified it two ways — reverting the `usedSource` key in `free` (pick.go:128) reddens `TestOneEntryMaySupplyOnlyOneOption` at seed 0, and BR-24's own reproduction deck (`concrete` + `cóncrete`) is green at HEAD and red without the key, with the quokka and parrot questions each offering two senses of the `concrete` entry. The full suite is green (`go test ./cmd/define/...`, 107s), the two doc/plan guards are armed (I injected a bogus fallback reason and a bogus Name-cell identifier; both went red within a second), and the shipped behaviour is correct. What blocks a clean SHIP is entirely test-quality and doc-derivation: the round-7 fix for BR-26 does not do what it claims (`optionNumberIn` reads the redrawn prompt, not the reveal, so it returns `'1'` unconditionally), the sitting-level guard for BR-24 passes with the fix removed, and the enumerating atlas paragraph still restates `fallbackReasons` by hand. All are cheap.

**1. Strengths**

- `Candidate.Source` + `sourceOf` (`cmd/define/play/pick.go:40,50`) is the right root fix, not a third patch: it names *the entry* as the unit of meaning and subsumes the Word and Gloss keys rather than sitting beside them. The doc block at pick.go:25-42 explains why all three keys stay, which is the thing a future reader needs.
- `TestOneEntryMaySupplyOnlyOneOption` (`cmd/define/play/pick_test.go:314`) asserts all three dedup keys as one property over 60 seeds — one assertion for one class, mutation-verified red.
- `TestREADMENamesEveryFallbackReason` (`cmd/define/doc_sync_test.go:324`) and `TestPlanTablesNameEntitiesThatExist` (`cmd/define/repo_guard_test.go:659`) are both armed and produce genuinely actionable failure text. The `TestPlanNamedTestsExist` fix to walk subpackages (repo_guard_test.go:765) is the right call — a guard that failed on the ARCH-PURE arrangement was teaching people to weaken it.
- `TestYAMLWritesAtLastWhateverFieldsAreSet` (`cmd/define/store/yaml_test.go:434`) replaces a file-state claim with the actual on-disk ordering property, derived from the record rather than from a hardcoded neighbour key.
- `Missed` as an optional capability (`play/session.go:270`) keeps `Apply` form-agnostic while carrying #17 M2's finding out — no type switch on `*Choice` anywhere in the session.

**2. Critical findings** — none.

**3. Important findings**

- `cmd/define/optionpool_test.go:306` — `TestNoQuestionDrawsTwoOptionsFromOneEntry` passes with the fix it exists to pin removed. Detail in the findings block.
- BR-20 and BR-21 remain open (re-disposed `not-addressed` below).

**4. Minor findings** — BR-13, BR-25, BR-26 re-disposed; plus `Source = Entry.Headword()` using a key the same file documents as insufficient.

**5. Test coverage notes**

`ARCH-MOCK` passes for the dictionary (fake + `TestLiveDictionaryRedirectsADerivedForm` in both directions) and for the terminal (pty suite behind `darwin && conformance`). Both conformance suites **skip** in this environment (NOAD returns no entry; no pty available), so the round-7 pty change is untested by any run — which is how a fix that returns the wrong digit on every call shipped green. `ARCH-CONSTRAINTS` passes: `poolCap` is enforced by `TestSittingCostIsBoundedByTheCap`, and that test now asserts the output too, not just the bound.

**6. Architectural notes**

- `ARCH-DRY` — pass, with one flag: `entryDefines` (optionpool.go:158) computes entry identity as a token run, and optionpool.go:100/123 compute it as `Headword()`. Two notions of one fact, in one file, one of them documented as wrong. `#12` reuses this pool machinery, so it inherits both.
- `ARCH-PURE` — pass. `play` has an empty import allowlist; all dictionary prose stays in `main`; `PickOptions` is tested with no IO.
- `ARCH-PURPOSE` — flag. The shadow-sweep over the `fallbackReasons` single-source change finds three consumers and one that derives: README (derived), `atlas/define.md:2007` (hand-maintained), and `choiceFor`'s own doc comment (optionpool.go:183, hand-maintained, directly above the declared list). That is the "one consumer wired, the rest left as documentation" shape.
- `ARCH-MOCK` — pass, with the coverage caveat above.
- `ARCH-CONSTRAINTS` — pass.

**7. Plan revision recommendations**

- The 2026-08-30 round-6 revision says of `TestOneEntryMaySupplyOnlyOneOption` and `TestNoQuestionDrawsTwoOptionsFromOneEntry`: "Both mutation-verified." Only the first is. Correct the claim and name the mutation performed for each.
- The 2026-08-30 round-7 revision says the declaration scan "names six declarations with no row … and all six are correctly absent". Re-run over the same two files: it is seven, and the omitted one is `poolCap`. The scan was also not run over `choice.go` or `pick.go`, where `distractorAxes` and `maxOptions` have no row.
- Done-when row 7 (plan.md:185) still asserts a grep no test performs; either mechanise it or weaken the cell.

```findings
dispose:
  - id: BR-24
    disposition: addressed
    note: |
      Mutation-verified twice: removing usedSource from free (pick.go:128) reddens TestOneEntryMaySupplyOnlyOneOption at seed 0, and BR-24's own deck (concrete + cóncrete) is green at HEAD and red without the key. See the new finding for the sitting-level guard.
  - id: BR-13
    disposition: not-addressed
    note: |
      All three survivors round 6 named are still in the tree, untouched by rounds 6 and 7: recall.go:32-38 (Grade's doc comment above Keys, so `go doc Recall.Keys` prints Grade's contract), optionpool_test.go:80 ("the FIRST usable sense of the first block"), optionpool_test.go:141 ("choiceFor's two refusals" against three fallbackReasons). The ~20-line AST check round 6 specified was not written. README.md:260 and glosslabel.go's label-precedence comment ARE fixed.
  - id: BR-20
    disposition: not-addressed
    note: |
      Row 8's second predicate is now genuinely pinned (TestYAMLWritesAtLastWhateverFieldsAreSet), but row 7's is not: plan.md:185 claims a grep for form names inside playSession and play.Apply that no test runs. It holds today by hand. The rule is still unstated above the table, and this round produced a fresh instance of exactly it — the round-6 revision's "Both mutation-verified" is false for one of the two tests it names.
  - id: BR-21
    disposition: not-addressed
    note: |
      The atlas enumeration at atlas/define.md:2007-2018 now lists all three reasons but is still hand-maintained; TestREADMENamesEveryFallbackReason reads README.md only. A third hand-maintained restatement sits at optionpool.go:183, directly above the declared list. One of three consumers derives, so the rule's mechanism covers a third of the class. The fix is one loop over both doc paths in the existing test.
  - id: BR-25
    disposition: not-addressed
    note: |
      The row and the corrected procedure both landed, but the procedure is prose and its first execution already dropped an entity. Re-running the recorded scan over optionpool.go and glosslabel.go names SEVEN declarations without a row, not six; the omitted one is poolCap — the ARCH-CONSTRAINTS budget a test already pins, which is the same argument that earned fallbackReasons its row. The scan was never run over choice.go or pick.go, where distractorAxes and maxOptions also have no row.
  - id: BR-26
    disposition: not-addressed
    note: |
      The fix reads the prompt, not the reveal. draw (play_loop.go:308) writes q.Prompt() — which contains the numbered option lines — before q.Reveal(), so optionNumberIn returns '1' for every question; reproduced in a scratch copy with a Choice whose correct option is slot 3 (got '1'). So `wrong` is always '2' and the calendar dependency is shifted from slot 1 to slot 2, not removed; and `if correct == 0 { break }` (pty_conformance_test.go:733) is unreachable.
findings:
  - id: new
    severity: Important
    family: guard-passes-without-the-property
    title: |
      TestNoQuestionDrawsTwoOptionsFromOneEntry passes with BR-24's fix removed, and with Source never set at all
    detail: |
      4th finding in family guard-passes-without-the-property, so the deliverable is the rule: a test written to pin a fix must be mutation-verified against THAT fix — revert the fix, the test must go red — and a "mutation-verified" claim recorded in a plan must name the mutation so a later reader can re-run it. Measured in a scratch copy of HEAD: optionpool_test.go:306 stays PASS both when usedSource is dropped from free (pick.go:128) and when `Source: e.Headword()` is deleted from optionCandidates (optionpool.go:100). Its fixture cannot produce the defect — the shared-entry pair jalapeño/jalapeno has one usable sense, which Gloss-dedup already covers, and the multi-sense entry in the deck (concrete, 2 candidates) appears under a single key. BR-24's own reproduction deck does work: playRig with "concrete", "cóncrete", "quokka", "mesa", "parrot" is green at HEAD and, with usedSource removed, reports the quokka and parrot questions each drawing two senses of the concrete entry. Swap the fixture, and correct the plan's round-6 "Both mutation-verified" to what was actually run.
  - id: new
    severity: Minor
    family: answer-must-define-the-prompted-word
    title: |
      Source is Entry.Headword(), which the same file documents as not being the entry's identity
    detail: |
      4th finding in this family, so the rule rather than the site: there is ONE entry identity, and it is the head token run entryDefines already walks (optionpool.go:158-168) — not Headword(). optionpool.go:100 and :123 set Source from Headword(), which parse.go builds from fields[0]; measured on the committed corpus, that is "hot" for `hot dog` and "a" for `a priori`. Two different entries can therefore share a Source, and PickOptions' usedSource key silently drops one of their options — a learner with both `hot dog` and `hot` in the deck loses a distractor, and on a small deck loses the form entirely to Recall. The direction of failure is over-dedup, never a wrong answer, which is why this is Minor and not a repeat of the Criticals. Fix: extract the token run as `entryIdentity(e Entry) string`, have entryDefines compare against it and both producers set Source from it, so the file stops carrying two answers to "which entry is this".
```

---
gate: boundary-review
issue: 7
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-30T19:33:02-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: PickOptions uses the seed only to shuffle order, so one sitting shares a single distractor set
          detail: |-
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
          family: seeded-selection-not-just-ordering
          round: 1
        - id: BR-2
          severity: Critical
          title: The plan's Core concepts table names five entities that do not exist in the tree
          detail: |-
            senseLabel, noadLabels and excludeCrossReferenced are not in cmd/define/glosslabel.go
            (the tree has readGloss/glossFacts/leadingLabel, three noad*Labels tables, and
            crossReferenced); pickOptions and shuffle are not in cmd/define/play/choice.go
            (they are PickOptions and prng.shuffleOptions in play/pick.go). Two new EXPORTED
            entities, play.Candidate and play.SampleStrings, are absent from the table
            entirely. The 2026-08-30 Revisions explain the design change but never update the
            table, so the plan still claims a surface the code does not have. Same divergence
            in the issue's Plan T1/T3 lines.
          family: plan-artifact-must-match-tree
          round: 1
        - id: BR-3
          severity: Important
          title: TestEveryAxisIsSelectable passes for an axis that distractorAxes never selects
          detail: |-
            Verified by adding AxisConnotation before numAxes with a String() case and leaving
            distractorAxes untouched: the test stays green. Its pool holds only candidates of
            the axis under test, so PickOptions' pass 2 fills the slots regardless of
            priority. Done-when 6 claims the row goes red when "an axis is added that nothing
            can select"; only the separate String() assertion catches it, which is a different
            property. Shape the pool as three AxisGeneral candidates followed by one of axis
            `a`, so only pass 1 can reach it.
          family: guard-passes-without-the-property
          round: 1
        - id: BR-4
          severity: Important
          title: The hand-rolled PRNG and FNV hash are justified by tests that do not pin them
          detail: |-
            D5a rejects math/rand and hash/fnv because "a PRNG defined here is pinned by this
            repo's own tests". Changing play/pick.go's xorshift shift 13 to 12 AND
            optionpool.go's FNV offset basis by one, together, leaves ./cmd/define and
            ./cmd/define/play both green — TestPickOptionsIsDeterministic only compares runs
            within one binary, which math/rand would also satisfy. Add one golden literal each
            for newPRNG(seed).next() and seedFor(...).
          family: guard-passes-without-the-property
          round: 1
        - id: BR-5
          severity: Important
          title: buildPool, poolCap, optionCandidates, targetCandidate, choiceFor and seedFor have no direct tests
          detail: |-
            A grep of cmd/define/*_test.go finds none of these names. Two gaps follow. The
            ARCH-CONSTRAINTS budget is unenforced: the quadratic regression play_loop.go's own
            comment warns about, and any bypass of poolCap, would be caught by nothing — a
            counting dictionary fake asserting lookups <= poolCap + len(due) on a >40-word deck
            would fix it. And targetCandidate returning false, which routes an entry with no
            usable sense to form 2.1 (the corpus word `bases` is exactly this), is exercised by
            no test; TestASittingFallsBackToRecall only covers the one-word deck.
          family: declared-envelope-unenforced
          round: 1
        - id: BR-6
          severity: Important
          title: README states the recall fallback threshold one word too high
          detail: |-
            cmd/define/README.md's "Recall, on a young deck" says "With fewer than two other
            words to draw on there is nothing to choose between", but choiceFor only refuses at
            ZERO usable distractors and TestASittingFallsBackToRecall pins a two-word deck
            producing *play.Choice. Should read "with no other word to draw on". The same
            section says "The word appears with four definitions" without noting that a young
            deck gets two or three.
          family: docs-restate-behaviour-inaccurately
          round: 1
        - id: BR-7
          severity: Minor
          title: choice_test.go hand-rolls linesOf and contains though the package's tests may import strings
          detail: |-
            play/choice_test.go:104-121. The purity guard reads only non-test imports
            (puretest.go:57 uses .Imports), and recall_test.go in the same package already
            imports strings.
          family: duplicate-stdlib-helper
          round: 1
        - id: BR-8
          severity: Minor
          title: The pty test's D8 check is skipped whenever the learner gets nothing right
          detail: |-
            pty_conformance_test.go gates the "no axis on a correct answer" assertion on
            n > 0 ("correct: true" appearing at least once), but the test presses `1` for every
            question, so an all-wrong run silently skips it. Force one correct answer by reading
            Options().
          family: conditional-assertion
          round: 1
        - id: BR-9
          severity: Minor
          title: Choice.Keys() has a dead "no options" branch
          detail: |-
            choice.go:150 returns "no options" for len(options) < 2, which choiceFor's
            len(opts) < 2 guard makes unreachable.
          family: unreachable-branch
          round: 1
        - id: BR-10
          severity: Minor
          title: Revealing before answering a Choice hands the learner the correct option
          detail: |-
            Enter/space calls Reveal(), which prints the correct option line; the learner can
            then press that digit and record Correct. Inherited from form 2.1's self-rating
            model and documented as "see the answer first", but it means a recognition question
            can be answered for free. Worth stating deliberately rather than leaving implicit.
          family: docs-restate-behaviour-inaccurately
          round: 1
        - id: BR-11
          severity: Minor
          title: Import grouping and an unwrapped atlas line
          detail: |-
            doc_sync_test.go:5 places cmd/define/play inside the stdlib import group;
            atlas/define.md runs the new "Below two options it is not a question" paragraph into
            the pre-existing TestSessionIsFormAgnostic sentence on one long line.
          family: formatting-nit
          round: 1
      blocked: true
---

# Gate ledger — tools#7 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-30T19:33:02-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `seeded-selection-not-just-ordering` PickOptions uses the seed only to shuffle order, so one sitting shares a single distractor set
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
- **BR-2** [Critical] `plan-artifact-must-match-tree` The plan's Core concepts table names five entities that do not exist in the tree
  senseLabel, noadLabels and excludeCrossReferenced are not in cmd/define/glosslabel.go
  (the tree has readGloss/glossFacts/leadingLabel, three noad*Labels tables, and
  crossReferenced); pickOptions and shuffle are not in cmd/define/play/choice.go
  (they are PickOptions and prng.shuffleOptions in play/pick.go). Two new EXPORTED
  entities, play.Candidate and play.SampleStrings, are absent from the table
  entirely. The 2026-08-30 Revisions explain the design change but never update the
  table, so the plan still claims a surface the code does not have. Same divergence
  in the issue's Plan T1/T3 lines.
- **BR-3** [Important] `guard-passes-without-the-property` TestEveryAxisIsSelectable passes for an axis that distractorAxes never selects
  Verified by adding AxisConnotation before numAxes with a String() case and leaving
  distractorAxes untouched: the test stays green. Its pool holds only candidates of
  the axis under test, so PickOptions' pass 2 fills the slots regardless of
  priority. Done-when 6 claims the row goes red when "an axis is added that nothing
  can select"; only the separate String() assertion catches it, which is a different
  property. Shape the pool as three AxisGeneral candidates followed by one of axis
  `a`, so only pass 1 can reach it.
- **BR-4** [Important] `guard-passes-without-the-property` The hand-rolled PRNG and FNV hash are justified by tests that do not pin them
  D5a rejects math/rand and hash/fnv because "a PRNG defined here is pinned by this
  repo's own tests". Changing play/pick.go's xorshift shift 13 to 12 AND
  optionpool.go's FNV offset basis by one, together, leaves ./cmd/define and
  ./cmd/define/play both green — TestPickOptionsIsDeterministic only compares runs
  within one binary, which math/rand would also satisfy. Add one golden literal each
  for newPRNG(seed).next() and seedFor(...).
- **BR-5** [Important] `declared-envelope-unenforced` buildPool, poolCap, optionCandidates, targetCandidate, choiceFor and seedFor have no direct tests
  A grep of cmd/define/*_test.go finds none of these names. Two gaps follow. The
  ARCH-CONSTRAINTS budget is unenforced: the quadratic regression play_loop.go's own
  comment warns about, and any bypass of poolCap, would be caught by nothing — a
  counting dictionary fake asserting lookups <= poolCap + len(due) on a >40-word deck
  would fix it. And targetCandidate returning false, which routes an entry with no
  usable sense to form 2.1 (the corpus word `bases` is exactly this), is exercised by
  no test; TestASittingFallsBackToRecall only covers the one-word deck.
- **BR-6** [Important] `docs-restate-behaviour-inaccurately` README states the recall fallback threshold one word too high
  cmd/define/README.md's "Recall, on a young deck" says "With fewer than two other
  words to draw on there is nothing to choose between", but choiceFor only refuses at
  ZERO usable distractors and TestASittingFallsBackToRecall pins a two-word deck
  producing *play.Choice. Should read "with no other word to draw on". The same
  section says "The word appears with four definitions" without noting that a young
  deck gets two or three.
- **BR-7** [Minor] `duplicate-stdlib-helper` choice_test.go hand-rolls linesOf and contains though the package's tests may import strings
  play/choice_test.go:104-121. The purity guard reads only non-test imports
  (puretest.go:57 uses .Imports), and recall_test.go in the same package already
  imports strings.
- **BR-8** [Minor] `conditional-assertion` The pty test's D8 check is skipped whenever the learner gets nothing right
  pty_conformance_test.go gates the "no axis on a correct answer" assertion on
  n > 0 ("correct: true" appearing at least once), but the test presses `1` for every
  question, so an all-wrong run silently skips it. Force one correct answer by reading
  Options().
- **BR-9** [Minor] `unreachable-branch` Choice.Keys() has a dead "no options" branch
  choice.go:150 returns "no options" for len(options) < 2, which choiceFor's
  len(opts) < 2 guard makes unreachable.
- **BR-10** [Minor] `docs-restate-behaviour-inaccurately` Revealing before answering a Choice hands the learner the correct option
  Enter/space calls Reveal(), which prints the correct option line; the learner can
  then press that digit and record Correct. Inherited from form 2.1's self-rating
  model and documented as "see the answer first", but it means a recognition question
  can be answered for free. Worth stating deliberately rather than leaving implicit.
- **BR-11** [Minor] `formatting-nit` Import grouping and an unwrapped atlas line
  doc_sync_test.go:5 places cmd/define/play inside the stdlib import group;
  atlas/define.md runs the new "Below two options it is not a question" paragraph into
  the pre-existing TestSessionIsFormAgnostic sentence on one long line.

## Open findings

- **BR-1** [Critical] `seeded-selection-not-just-ordering` PickOptions uses the seed only to shuffle order, so one sitting shares a single distractor set
- **BR-2** [Critical] `plan-artifact-must-match-tree` The plan's Core concepts table names five entities that do not exist in the tree
- **BR-3** [Important] `guard-passes-without-the-property` TestEveryAxisIsSelectable passes for an axis that distractorAxes never selects
- **BR-4** [Important] `guard-passes-without-the-property` The hand-rolled PRNG and FNV hash are justified by tests that do not pin them
- **BR-5** [Important] `declared-envelope-unenforced` buildPool, poolCap, optionCandidates, targetCandidate, choiceFor and seedFor have no direct tests
- **BR-6** [Important] `docs-restate-behaviour-inaccurately` README states the recall fallback threshold one word too high
- **BR-7** [Minor] `duplicate-stdlib-helper` choice_test.go hand-rolls linesOf and contains though the package's tests may import strings
- **BR-8** [Minor] `conditional-assertion` The pty test's D8 check is skipped whenever the learner gets nothing right
- **BR-9** [Minor] `unreachable-branch` Choice.Keys() has a dead "no options" branch
- **BR-10** [Minor] `docs-restate-behaviour-inaccurately` Revealing before answering a Choice hands the learner the correct option
- **BR-11** [Minor] `formatting-nit` Import grouping and an unwrapped atlas line

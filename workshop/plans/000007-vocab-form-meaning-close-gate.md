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
    - "n": 2
      timestamp: "2026-08-30T20:12:15-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Mutation-verified: reverting to fixed pool order gives 6 sets (test red). End-to-end on a 24-word corpus deck: 19 distinct distractor sets across 19 questions.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: The five named rows are corrected in plan and issue; the class is not swept — see the new plan-artifact-must-match-tree finding.
          round: 2
        - id: BR-3
          disposition: addressed
          note: 'Mutation-verified: adding AxisConnotation outside distractorAxes turns the membership assertion red.'
          round: 2
        - id: BR-4
          disposition: addressed
          note: 'Mutation-verified: xorshift shift 13 to 12 and the FNV offset basis each turn their golden test red.'
          round: 2
        - id: BR-5
          disposition: addressed
          note: countingDict pins the envelope (red when the poolCap truncation is removed); targetCandidate, choiceFor, optionCandidates and seedFor all now have direct tests.
          round: 2
        - id: BR-6
          disposition: addressed
          note: README now reads "With no other word to draw on" and states that a young deck gets two or three options.
          round: 2
        - id: BR-7
          disposition: addressed
          note: choice_test.go imports strings with the puretest .Imports rationale in the import block.
          round: 2
        - id: BR-8
          disposition: addressed
          note: The D8 check is now an unconditional identity (reviewed minus correct equals missed-line count); no n > 0 gate remains.
          round: 2
        - id: BR-9
          disposition: addressed
          note: The dead branch is gone and replaced by a comment explaining why no branch is needed.
          round: 2
        - id: BR-10
          disposition: addressed
          note: Named deliberately in the README key table and in a dedicated atlas paragraph.
          round: 2
        - id: BR-11
          disposition: addressed
          note: Import group split in doc_sync_test.go; the atlas paragraph is wrapped and no longer runs into the TestSessionIsFormAgnostic sentence.
          round: 2
      findings:
        - id: BR-12
          severity: Important
          title: BR-2 fixed the five rows it named; three enumerable siblings of the same class remain
          detail: |-
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
          family: plan-artifact-must-match-tree
          round: 2
        - id: BR-13
          severity: Minor
          title: Five doc comments state behaviour the code does not have
          detail: |-
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
          family: docs-restate-behaviour-inaccurately
          round: 2
        - id: BR-14
          severity: Minor
          title: The issue file has two "## Log" sections, the second outside the canonical order
          detail: |-
            workshop/issues/000007-vocab-form-meaning.md carries "## Log" at line 147 and
            again at line 247, after "## Revisions". Measured - number 7 is the only one of
            15 active issues with a duplicate section. sdlc issue validate passes because it
            checks presence only, so any reader or tool taking "the Log" gets the
            2026-08-20/27 stub and misses the entire build log. Merge them under the single
            canonical heading.
          family: artifact-violates-its-schema
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-30T20:34:03-07:00"
      agent: claude
      dispose:
        - id: BR-12
          disposition: addressed
          note: Verified in the tree - play.Missed is in the Integration points table (plan:149), D4a corrected (plan:61-63), Done-when row 9 added and row 3 widened (plan:179,184); all 11 cited tests exist; go doc -short on play gives 15 exported names, 11 tabled, 4 belonging to issue 6.
          round: 3
        - id: BR-13
          disposition: not-addressed
          note: 5 of 8 named instances fixed; 3 survive and 4 fresh siblings joined them, so the rule was written down but never swept.
          round: 3
        - id: BR-14
          disposition: addressed
          note: grep -n '^## ' on the issue file shows one Log, at line 147, in canonical order.
          round: 3
      findings:
        - id: BR-15
          severity: Important
          title: A derivative lookup makes form 2.3's "correct" option the definition of a different word
          detail: |-
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
          family: answer-must-define-the-prompted-word
          round: 3
        - id: BR-16
          severity: Minor
          title: The same reverse Fisher-Yates is written twice in pick.go
          detail: |-
            cmd/define/play/pick.go:68-71 shuffles []int and pick.go:153-158 shuffles
            []Option with an identical loop (ARCH-DRY). A generic
            `func shuffle[T any](p *prng, xs []T)` unifies them and needs no import, so the
            package's empty-allowlist guard is not the reason they are separate.
          family: duplicated-algorithm-should-be-one-helper
          round: 3
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

## Round 2 — 2026-08-30T20:12:15-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Mutation-verified: reverting to fixed pool order gives 6 sets (test red). End-to-end on a 24-word corpus deck: 19 distinct distractor sets across 19 questions.
- BR-2 — addressed — The five named rows are corrected in plan and issue; the class is not swept — see the new plan-artifact-must-match-tree finding.
- BR-3 — addressed — Mutation-verified: adding AxisConnotation outside distractorAxes turns the membership assertion red.
- BR-4 — addressed — Mutation-verified: xorshift shift 13 to 12 and the FNV offset basis each turn their golden test red.
- BR-5 — addressed — countingDict pins the envelope (red when the poolCap truncation is removed); targetCandidate, choiceFor, optionCandidates and seedFor all now have direct tests.
- BR-6 — addressed — README now reads "With no other word to draw on" and states that a young deck gets two or three options.
- BR-7 — addressed — choice_test.go imports strings with the puretest .Imports rationale in the import block.
- BR-8 — addressed — The D8 check is now an unconditional identity (reviewed minus correct equals missed-line count); no n > 0 gate remains.
- BR-9 — addressed — The dead branch is gone and replaced by a comment explaining why no branch is needed.
- BR-10 — addressed — Named deliberately in the README key table and in a dedicated atlas paragraph.
- BR-11 — addressed — Import group split in doc_sync_test.go; the atlas paragraph is wrapped and no longer runs into the TestSessionIsFormAgnostic sentence.

### Raised

- **BR-12** [Important] `plan-artifact-must-match-tree` BR-2 fixed the five rows it named; three enumerable siblings of the same class remain
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
- **BR-13** [Minor] `docs-restate-behaviour-inaccurately` Five doc comments state behaviour the code does not have
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
- **BR-14** [Minor] `artifact-violates-its-schema` The issue file has two "## Log" sections, the second outside the canonical order
  workshop/issues/000007-vocab-form-meaning.md carries "## Log" at line 147 and
  again at line 247, after "## Revisions". Measured - number 7 is the only one of
  15 active issues with a duplicate section. sdlc issue validate passes because it
  checks presence only, so any reader or tool taking "the Log" gets the
  2026-08-20/27 stub and misses the entire build log. Merge them under the single
  canonical heading.

## Round 3 — 2026-08-30T20:34:03-07:00 (claude) — BLOCKED

### Disposed

- BR-12 — addressed — Verified in the tree - play.Missed is in the Integration points table (plan:149), D4a corrected (plan:61-63), Done-when row 9 added and row 3 widened (plan:179,184); all 11 cited tests exist; go doc -short on play gives 15 exported names, 11 tabled, 4 belonging to issue 6.
- BR-13 — not-addressed — 5 of 8 named instances fixed; 3 survive and 4 fresh siblings joined them, so the rule was written down but never swept.
- BR-14 — addressed — grep -n '^## ' on the issue file shows one Log, at line 147, in canonical order.

### Raised

- **BR-15** [Important] `answer-must-define-the-prompted-word` A derivative lookup makes form 2.3's "correct" option the definition of a different word
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
- **BR-16** [Minor] `duplicated-algorithm-should-be-one-helper` The same reverse Fisher-Yates is written twice in pick.go
  cmd/define/play/pick.go:68-71 shuffles []int and pick.go:153-158 shuffles
  []Option with an identical loop (ARCH-DRY). A generic
  `func shuffle[T any](p *prng, xs []T)` unifies them and needs no import, so the
  package's empty-allowlist guard is not the reason they are separate.

## Open findings

- **BR-13** [Minor] `docs-restate-behaviour-inaccurately` Five doc comments state behaviour the code does not have
- **BR-15** [Important] `answer-must-define-the-prompted-word` A derivative lookup makes form 2.3's "correct" option the definition of a different word
- **BR-16** [Minor] `duplicated-algorithm-should-be-one-helper` The same reverse Fisher-Yates is written twice in pick.go

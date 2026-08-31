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
    - "n": 4
      timestamp: "2026-08-30T21:00:40-07:00"
      agent: claude
      dispose:
        - id: BR-13
          disposition: not-addressed
          note: Named instances fixed; four survivors remain because the sweep grepped literal phrasings, not claims.
          round: 4
        - id: BR-15
          disposition: addressed
          note: Verified by reverting the gate in a scratch worktree - the bargainer row goes red without it.
          round: 4
        - id: BR-16
          disposition: addressed
          note: shuffle[T any] is called from both sites; shuffleOptions is gone from the tree.
          round: 4
      findings:
        - id: BR-17
          severity: Critical
          title: buildPool is not gated by entryDefines, so a redirect puts a base entry's gloss into questions under the derived word
          detail: |-
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
          family: answer-must-define-the-prompted-word
          round: 4
        - id: BR-18
          severity: Important
          title: The plan's Core concepts table names shuffleOptions, which this commit deleted, and omits entryDefines, which it added
          detail: |-
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
          family: plan-artifact-must-match-tree
          round: 4
        - id: BR-19
          severity: Minor
          title: gradePrompt's nil guard is dead - draw already returns when Current() is nil
          detail: |-
            play_loop.go:355. draw returns at play_loop.go:306 when s.Current() is nil, and the
            only other caller is doc_sync_test.go:60, which always passes a form. Second in this
            family after BR-9's Choice.Keys() branch, so the rule is the deliverable: a nil/empty
            guard whose precondition every caller already establishes is dead code that reads as
            protection, and it hides which layer actually owns the invariant.
          family: unreachable-branch
          round: 4
      blocked: true
    - "n": 5
      timestamp: "2026-08-30T21:29:47-07:00"
      agent: claude
      dispose:
        - id: BR-13
          disposition: not-addressed
          note: 3 of 7 named instances survive; the round-4 commit touched none of their files.
          round: 5
        - id: BR-17
          disposition: not-addressed
          note: Clause 1 fixed and mutation-verified; clause 2 (no two options share a gloss) unimplemented and measured reachable.
          round: 5
        - id: BR-18
          disposition: not-addressed
          note: Mechanism delivered and verified firing, but the finding's own named instance (plan.md pickOptions) sits outside its scope.
          round: 5
        - id: BR-19
          disposition: not-addressed
          note: Branch still at play_loop.go:359; replacing it with a panic leaves the whole suite green.
          round: 5
      findings:
        - id: BR-20
          severity: Important
          title: Done-when row 1 claims a red-when that TestPickOptionsHasOneAnswer cannot deliver
          detail: |-
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
          family: guard-passes-without-the-property
          round: 5
        - id: BR-21
          severity: Important
          title: README and atlas enumerate the form-2.1 fallback causes and omit the derivative redirect
          detail: |-
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
          family: docs-restate-behaviour-inaccurately
          round: 5
        - id: BR-22
          severity: Minor
          title: The over-breadth half of TestARedirectSuppliesNoOptionMaterialAtAll never executes
          detail: |-
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
          family: conditional-assertion
          round: 5
        - id: BR-23
          severity: Minor
          title: The derivative-redirect model that entryDefines gates on has no live conformance check
          detail: |-
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
          family: fake-behaviour-lacks-live-conformance
          round: 5
      blocked: true
    - "n": 6
      timestamp: "2026-08-30T22:01:29-07:00"
      agent: claude
      dispose:
        - id: BR-13
          disposition: not-addressed
          note: 'Named instances fixed, but the rule''s sweep is a line-oriented phrase grep and three claims stand: optionpool_test.go:79 still says "the FIRST usable sense of the first block" (the phrase wraps, so the grep could not see it); optionpool_test.go:141 says "choiceFor''s two refusals" while fallbackReasons declares three; and play/recall.go:38 now carries Grade''s doc comment verbatim, so `go doc Recall.Keys` prints Grade''s contract and Grade is undocumented — a shape no phrase grep can find, but a ~20-line AST check ("a doc comment opens with the name of the declaration it precedes") finds it, and finds exactly one new instance in this window.'
          round: 6
        - id: BR-17
          disposition: addressed
          note: 'Both clauses mutation-verified in a scratch worktree: dropping the entryDefines gate from optionCandidates turns TestNoCandidateEverCarriesAnotherWordsGloss and TestARedirectSuppliesNoOptionMaterialAtAll red; reverting `free` to word-only dedup turns TestPickOptionsNeverRepeatsAGloss red. See the new finding for the sibling path the gloss key does not cover.'
          round: 6
        - id: BR-18
          disposition: addressed
          note: Ticking the plan's task list arms TestPlanTablesNameEntitiesThatExist; I injected a bogus Name-cell identifier and the guard failed within seconds, and TestPlanNamedTestsExist now walks the tree.
          round: 6
        - id: BR-19
          disposition: addressed
          note: The guard is deleted and replaced by a comment naming the invariant's owner; draw returns at play_loop.go:306 when Current() is nil, confirmed.
          round: 6
        - id: BR-20
          disposition: not-addressed
          note: 'Row 1 (the instance) was fixed; the rule was neither stated above the table nor applied. Measured: row 7''s second predicate — "no form name appears inside playSession or play.Apply, a grep not a file-state claim" — is run by no test. I ran it by hand and it holds today, but nothing goes red if it stops.'
          round: 6
        - id: BR-21
          disposition: not-addressed
          note: README now derives from fallbackReasons via TestREADMENamesEveryFallbackReason, but atlas/define.md:2007-2018 still hand-maintains the same closed enumeration — and the atlas was the file that had drifted furthest. doc_sync_test.go already checks the atlas elsewhere, so this is one more loop in the same test. Also fallbackReasons sits beside choiceFor's branches rather than being consumed by them. The README "never offered as a distractor" half is fixed.
          round: 6
        - id: BR-22
          disposition: addressed
          note: The block is unconditional and now asserts on the same entry, which is the object that shows the gate/ban distinction. Swept the window's test files and the repo for the shape; every remaining `if err == nil` is an expected-error check.
          round: 6
        - id: BR-23
          disposition: addressed
          note: TestLiveDictionaryRedirectsADerivedForm pins the model in both directions (skips here — NOAD unavailable). The refusal-RATE row asked for was not delivered; offline bound re-measured at 1 of 34, with hot dog / a priori / jalapeño / MacBook all passing.
          round: 6
      findings:
        - id: BR-24
          severity: Critical
          title: Two options in one set can both define the prompted word, because dedup keys on Word and Gloss but not on the source ENTRY
          detail: '3rd finding in this family, so the deliverable is the rule: an option set may contain at most one option per source ENTRY. Word and Gloss are proxies that each fail where the other holds — Word failed on jalapeño/jalapeno (BR-17 clause 2), Gloss fails the moment the shared entry has more than one usable sense. Measured: differsOnlyByDiacritics (optionpool.go:164) admits both deck keys against one entry; optionCandidates (optionpool.go:80) returns one candidate per axis, and 17 of 34 corpus entries yield >=2 differently-glossed candidates; choiceFor''s `c.Word == target.Word` filter does not match across the spelling variants and crossReferenced is blind because neither headword is in either gloss. Reproduced in a scratch worktree over the committed `concrete` entry under a second accented deck key: the set offered "existing in a material or physical form; not abstract" as Correct and the same entry''s "form (something) into a mass; solidify" as a register distractor, so a learner picking it records a miss with a fabricated axis and the word is demoted for a defensible answer. Fix: carry entry identity (Headword(), or the resolved lookup key) on play.Candidate and the target, set it in optionCandidates/targetCandidate, and dedup PickOptions on it — subsuming both existing dedups rather than adding a third. Pin with a pick_test.go row where two candidates share an entry id with different glosses, and an optionpool_test.go row driving a multi-sense corpus entry through two deck keys.'
          family: answer-must-define-the-prompted-word
          round: 6
        - id: BR-25
          severity: Minor
          title: fallbackReasons has no Core-concepts row, and the recorded derivation procedure cannot see package main
          detail: 5th finding in this family, so the rule rather than the row. Round 2 recorded the derivation as `go doc -short` per touched package; that command returns nothing for package main, so the procedure could only ever have covered play/ — and both entities added since (entryDefines, fallbackReasons) live in cmd/define. entryDefines got a row only after a reviewer named it; fallbackReasons, which a doc guard now depends on, has none. BR-18's armed guard checks table-to-tree only. The mechanical form of the rule is a declaration scan of the named files (go/ast or a `^func|^var|^const|^type` grep) run against the Name column at close, in place of `go doc -short`.
          family: plan-artifact-must-match-tree
          round: 6
        - id: BR-26
          severity: Minor
          title: The pty form-2.3 test's "at least one miss" assertion depends on today's date
          detail: pty_conformance_test.go:717 answers every question with `1` and then requires a `missed:` line. The answer's slot is a deterministic function of seedFor(key, day), so on roughly 1 day in 1000 all five words put the answer in slot 1 and the assertion fails for the wrong reason. The same block also assumes all five words get form 2.3 — a live-dictionary fallback to Recall would stall the sitting rather than fail legibly. Drive the keys from the rendered option lines, or seed the sitting through a fixed clock.
          family: uncontrolled-test-input
          round: 6
      blocked: true
    - "n": 7
      timestamp: "2026-08-30T22:38:27-07:00"
      agent: claude
      blocked: true
      protocol_error: no valid findings block
    - "n": 8
      timestamp: "2026-08-30T23:05:02-07:00"
      agent: claude
      dispose:
        - id: BR-24
          disposition: addressed
          note: 'Mutation-verified twice: removing usedSource from free (pick.go:128) reddens TestOneEntryMaySupplyOnlyOneOption at seed 0, and BR-24''s own deck (concrete + cóncrete) is green at HEAD and red without the key. See the new finding for the sitting-level guard.'
          round: 8
        - id: BR-13
          disposition: not-addressed
          note: 'All three survivors round 6 named are still in the tree, untouched by rounds 6 and 7: recall.go:32-38 (Grade''s doc comment above Keys, so `go doc Recall.Keys` prints Grade''s contract), optionpool_test.go:80 ("the FIRST usable sense of the first block"), optionpool_test.go:141 ("choiceFor''s two refusals" against three fallbackReasons). The ~20-line AST check round 6 specified was not written. README.md:260 and glosslabel.go''s label-precedence comment ARE fixed.'
          round: 8
        - id: BR-20
          disposition: not-addressed
          note: 'Row 8''s second predicate is now genuinely pinned (TestYAMLWritesAtLastWhateverFieldsAreSet), but row 7''s is not: plan.md:185 claims a grep for form names inside playSession and play.Apply that no test runs. It holds today by hand. The rule is still unstated above the table, and this round produced a fresh instance of exactly it — the round-6 revision''s "Both mutation-verified" is false for one of the two tests it names.'
          round: 8
        - id: BR-21
          disposition: not-addressed
          note: The atlas enumeration at atlas/define.md:2007-2018 now lists all three reasons but is still hand-maintained; TestREADMENamesEveryFallbackReason reads README.md only. A third hand-maintained restatement sits at optionpool.go:183, directly above the declared list. One of three consumers derives, so the rule's mechanism covers a third of the class. The fix is one loop over both doc paths in the existing test.
          round: 8
        - id: BR-25
          disposition: not-addressed
          note: The row and the corrected procedure both landed, but the procedure is prose and its first execution already dropped an entity. Re-running the recorded scan over optionpool.go and glosslabel.go names SEVEN declarations without a row, not six; the omitted one is poolCap — the ARCH-CONSTRAINTS budget a test already pins, which is the same argument that earned fallbackReasons its row. The scan was never run over choice.go or pick.go, where distractorAxes and maxOptions also have no row.
          round: 8
        - id: BR-26
          disposition: not-addressed
          note: The fix reads the prompt, not the reveal. draw (play_loop.go:308) writes q.Prompt() — which contains the numbered option lines — before q.Reveal(), so optionNumberIn returns '1' for every question; reproduced in a scratch copy with a Choice whose correct option is slot 3 (got '1'). So `wrong` is always '2' and the calendar dependency is shifted from slot 1 to slot 2, not removed; and `if correct == 0 { break }` (pty_conformance_test.go:733) is unreachable.
          round: 8
      findings:
        - id: BR-27
          severity: Important
          title: TestNoQuestionDrawsTwoOptionsFromOneEntry passes with BR-24's fix removed, and with Source never set at all
          detail: '4th finding in family guard-passes-without-the-property, so the deliverable is the rule: a test written to pin a fix must be mutation-verified against THAT fix — revert the fix, the test must go red — and a "mutation-verified" claim recorded in a plan must name the mutation so a later reader can re-run it. Measured in a scratch copy of HEAD: optionpool_test.go:306 stays PASS both when usedSource is dropped from free (pick.go:128) and when `Source: e.Headword()` is deleted from optionCandidates (optionpool.go:100). Its fixture cannot produce the defect — the shared-entry pair jalapeño/jalapeno has one usable sense, which Gloss-dedup already covers, and the multi-sense entry in the deck (concrete, 2 candidates) appears under a single key. BR-24''s own reproduction deck does work: playRig with "concrete", "cóncrete", "quokka", "mesa", "parrot" is green at HEAD and, with usedSource removed, reports the quokka and parrot questions each drawing two senses of the concrete entry. Swap the fixture, and correct the plan''s round-6 "Both mutation-verified" to what was actually run.'
          family: guard-passes-without-the-property
          round: 8
        - id: BR-28
          severity: Minor
          title: Source is Entry.Headword(), which the same file documents as not being the entry's identity
          detail: '4th finding in this family, so the rule rather than the site: there is ONE entry identity, and it is the head token run entryDefines already walks (optionpool.go:158-168) — not Headword(). optionpool.go:100 and :123 set Source from Headword(), which parse.go builds from fields[0]; measured on the committed corpus, that is "hot" for `hot dog` and "a" for `a priori`. Two different entries can therefore share a Source, and PickOptions'' usedSource key silently drops one of their options — a learner with both `hot dog` and `hot` in the deck loses a distractor, and on a small deck loses the form entirely to Recall. The direction of failure is over-dedup, never a wrong answer, which is why this is Minor and not a repeat of the Criticals. Fix: extract the token run as `entryIdentity(e Entry) string`, have entryDefines compare against it and both producers set Source from it, so the file stops carrying two answers to "which entry is this".'
          family: answer-must-define-the-prompted-word
          round: 8
      blocked: false
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

## Round 4 — 2026-08-30T21:00:40-07:00 (claude) — BLOCKED

### Disposed

- BR-13 — not-addressed — Named instances fixed; four survivors remain because the sweep grepped literal phrasings, not claims.
- BR-15 — addressed — Verified by reverting the gate in a scratch worktree - the bargainer row goes red without it.
- BR-16 — addressed — shuffle[T any] is called from both sites; shuffleOptions is gone from the tree.

### Raised

- **BR-17** [Critical] `answer-must-define-the-prompted-word` buildPool is not gated by entryDefines, so a redirect puts a base entry's gloss into questions under the derived word
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
- **BR-18** [Important] `plan-artifact-must-match-tree` The plan's Core concepts table names shuffleOptions, which this commit deleted, and omits entryDefines, which it added
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
- **BR-19** [Minor] `unreachable-branch` gradePrompt's nil guard is dead - draw already returns when Current() is nil
  play_loop.go:355. draw returns at play_loop.go:306 when s.Current() is nil, and the
  only other caller is doc_sync_test.go:60, which always passes a form. Second in this
  family after BR-9's Choice.Keys() branch, so the rule is the deliverable: a nil/empty
  guard whose precondition every caller already establishes is dead code that reads as
  protection, and it hides which layer actually owns the invariant.

## Round 5 — 2026-08-30T21:29:47-07:00 (claude) — BLOCKED

### Disposed

- BR-13 — not-addressed — 3 of 7 named instances survive; the round-4 commit touched none of their files.
- BR-17 — not-addressed — Clause 1 fixed and mutation-verified; clause 2 (no two options share a gloss) unimplemented and measured reachable.
- BR-18 — not-addressed — Mechanism delivered and verified firing, but the finding's own named instance (plan.md pickOptions) sits outside its scope.
- BR-19 — not-addressed — Branch still at play_loop.go:359; replacing it with a panic leaves the whole suite green.

### Raised

- **BR-20** [Important] `guard-passes-without-the-property` Done-when row 1 claims a red-when that TestPickOptionsHasOneAnswer cannot deliver
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
- **BR-21** [Important] `docs-restate-behaviour-inaccurately` README and atlas enumerate the form-2.1 fallback causes and omit the derivative redirect
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
- **BR-22** [Minor] `conditional-assertion` The over-breadth half of TestARedirectSuppliesNoOptionMaterialAtAll never executes
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
- **BR-23** [Minor] `fake-behaviour-lacks-live-conformance` The derivative-redirect model that entryDefines gates on has no live conformance check
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

## Round 6 — 2026-08-30T22:01:29-07:00 (claude) — BLOCKED

### Disposed

- BR-13 — not-addressed — Named instances fixed, but the rule's sweep is a line-oriented phrase grep and three claims stand: optionpool_test.go:79 still says "the FIRST usable sense of the first block" (the phrase wraps, so the grep could not see it); optionpool_test.go:141 says "choiceFor's two refusals" while fallbackReasons declares three; and play/recall.go:38 now carries Grade's doc comment verbatim, so `go doc Recall.Keys` prints Grade's contract and Grade is undocumented — a shape no phrase grep can find, but a ~20-line AST check ("a doc comment opens with the name of the declaration it precedes") finds it, and finds exactly one new instance in this window.
- BR-17 — addressed — Both clauses mutation-verified in a scratch worktree: dropping the entryDefines gate from optionCandidates turns TestNoCandidateEverCarriesAnotherWordsGloss and TestARedirectSuppliesNoOptionMaterialAtAll red; reverting `free` to word-only dedup turns TestPickOptionsNeverRepeatsAGloss red. See the new finding for the sibling path the gloss key does not cover.
- BR-18 — addressed — Ticking the plan's task list arms TestPlanTablesNameEntitiesThatExist; I injected a bogus Name-cell identifier and the guard failed within seconds, and TestPlanNamedTestsExist now walks the tree.
- BR-19 — addressed — The guard is deleted and replaced by a comment naming the invariant's owner; draw returns at play_loop.go:306 when Current() is nil, confirmed.
- BR-20 — not-addressed — Row 1 (the instance) was fixed; the rule was neither stated above the table nor applied. Measured: row 7's second predicate — "no form name appears inside playSession or play.Apply, a grep not a file-state claim" — is run by no test. I ran it by hand and it holds today, but nothing goes red if it stops.
- BR-21 — not-addressed — README now derives from fallbackReasons via TestREADMENamesEveryFallbackReason, but atlas/define.md:2007-2018 still hand-maintains the same closed enumeration — and the atlas was the file that had drifted furthest. doc_sync_test.go already checks the atlas elsewhere, so this is one more loop in the same test. Also fallbackReasons sits beside choiceFor's branches rather than being consumed by them. The README "never offered as a distractor" half is fixed.
- BR-22 — addressed — The block is unconditional and now asserts on the same entry, which is the object that shows the gate/ban distinction. Swept the window's test files and the repo for the shape; every remaining `if err == nil` is an expected-error check.
- BR-23 — addressed — TestLiveDictionaryRedirectsADerivedForm pins the model in both directions (skips here — NOAD unavailable). The refusal-RATE row asked for was not delivered; offline bound re-measured at 1 of 34, with hot dog / a priori / jalapeño / MacBook all passing.

### Raised

- **BR-24** [Critical] `answer-must-define-the-prompted-word` Two options in one set can both define the prompted word, because dedup keys on Word and Gloss but not on the source ENTRY
  3rd finding in this family, so the deliverable is the rule: an option set may contain at most one option per source ENTRY. Word and Gloss are proxies that each fail where the other holds — Word failed on jalapeño/jalapeno (BR-17 clause 2), Gloss fails the moment the shared entry has more than one usable sense. Measured: differsOnlyByDiacritics (optionpool.go:164) admits both deck keys against one entry; optionCandidates (optionpool.go:80) returns one candidate per axis, and 17 of 34 corpus entries yield >=2 differently-glossed candidates; choiceFor's `c.Word == target.Word` filter does not match across the spelling variants and crossReferenced is blind because neither headword is in either gloss. Reproduced in a scratch worktree over the committed `concrete` entry under a second accented deck key: the set offered "existing in a material or physical form; not abstract" as Correct and the same entry's "form (something) into a mass; solidify" as a register distractor, so a learner picking it records a miss with a fabricated axis and the word is demoted for a defensible answer. Fix: carry entry identity (Headword(), or the resolved lookup key) on play.Candidate and the target, set it in optionCandidates/targetCandidate, and dedup PickOptions on it — subsuming both existing dedups rather than adding a third. Pin with a pick_test.go row where two candidates share an entry id with different glosses, and an optionpool_test.go row driving a multi-sense corpus entry through two deck keys.
- **BR-25** [Minor] `plan-artifact-must-match-tree` fallbackReasons has no Core-concepts row, and the recorded derivation procedure cannot see package main
  5th finding in this family, so the rule rather than the row. Round 2 recorded the derivation as `go doc -short` per touched package; that command returns nothing for package main, so the procedure could only ever have covered play/ — and both entities added since (entryDefines, fallbackReasons) live in cmd/define. entryDefines got a row only after a reviewer named it; fallbackReasons, which a doc guard now depends on, has none. BR-18's armed guard checks table-to-tree only. The mechanical form of the rule is a declaration scan of the named files (go/ast or a `^func|^var|^const|^type` grep) run against the Name column at close, in place of `go doc -short`.
- **BR-26** [Minor] `uncontrolled-test-input` The pty form-2.3 test's "at least one miss" assertion depends on today's date
  pty_conformance_test.go:717 answers every question with `1` and then requires a `missed:` line. The answer's slot is a deterministic function of seedFor(key, day), so on roughly 1 day in 1000 all five words put the answer in slot 1 and the assertion fails for the wrong reason. The same block also assumes all five words get form 2.3 — a live-dictionary fallback to Recall would stall the sitting rather than fail legibly. Drive the keys from the rendered option lines, or seed the sitting through a fixed clock.

## Round 7 — 2026-08-30T22:38:27-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 8 — 2026-08-30T23:05:02-07:00 (claude) — passed

### Disposed

- BR-24 — addressed — Mutation-verified twice: removing usedSource from free (pick.go:128) reddens TestOneEntryMaySupplyOnlyOneOption at seed 0, and BR-24's own deck (concrete + cóncrete) is green at HEAD and red without the key. See the new finding for the sitting-level guard.
- BR-13 — not-addressed — All three survivors round 6 named are still in the tree, untouched by rounds 6 and 7: recall.go:32-38 (Grade's doc comment above Keys, so `go doc Recall.Keys` prints Grade's contract), optionpool_test.go:80 ("the FIRST usable sense of the first block"), optionpool_test.go:141 ("choiceFor's two refusals" against three fallbackReasons). The ~20-line AST check round 6 specified was not written. README.md:260 and glosslabel.go's label-precedence comment ARE fixed.
- BR-20 — not-addressed — Row 8's second predicate is now genuinely pinned (TestYAMLWritesAtLastWhateverFieldsAreSet), but row 7's is not: plan.md:185 claims a grep for form names inside playSession and play.Apply that no test runs. It holds today by hand. The rule is still unstated above the table, and this round produced a fresh instance of exactly it — the round-6 revision's "Both mutation-verified" is false for one of the two tests it names.
- BR-21 — not-addressed — The atlas enumeration at atlas/define.md:2007-2018 now lists all three reasons but is still hand-maintained; TestREADMENamesEveryFallbackReason reads README.md only. A third hand-maintained restatement sits at optionpool.go:183, directly above the declared list. One of three consumers derives, so the rule's mechanism covers a third of the class. The fix is one loop over both doc paths in the existing test.
- BR-25 — not-addressed — The row and the corrected procedure both landed, but the procedure is prose and its first execution already dropped an entity. Re-running the recorded scan over optionpool.go and glosslabel.go names SEVEN declarations without a row, not six; the omitted one is poolCap — the ARCH-CONSTRAINTS budget a test already pins, which is the same argument that earned fallbackReasons its row. The scan was never run over choice.go or pick.go, where distractorAxes and maxOptions also have no row.
- BR-26 — not-addressed — The fix reads the prompt, not the reveal. draw (play_loop.go:308) writes q.Prompt() — which contains the numbered option lines — before q.Reveal(), so optionNumberIn returns '1' for every question; reproduced in a scratch copy with a Choice whose correct option is slot 3 (got '1'). So `wrong` is always '2' and the calendar dependency is shifted from slot 1 to slot 2, not removed; and `if correct == 0 { break }` (pty_conformance_test.go:733) is unreachable.

### Raised

- **BR-27** [Important] `guard-passes-without-the-property` TestNoQuestionDrawsTwoOptionsFromOneEntry passes with BR-24's fix removed, and with Source never set at all
  4th finding in family guard-passes-without-the-property, so the deliverable is the rule: a test written to pin a fix must be mutation-verified against THAT fix — revert the fix, the test must go red — and a "mutation-verified" claim recorded in a plan must name the mutation so a later reader can re-run it. Measured in a scratch copy of HEAD: optionpool_test.go:306 stays PASS both when usedSource is dropped from free (pick.go:128) and when `Source: e.Headword()` is deleted from optionCandidates (optionpool.go:100). Its fixture cannot produce the defect — the shared-entry pair jalapeño/jalapeno has one usable sense, which Gloss-dedup already covers, and the multi-sense entry in the deck (concrete, 2 candidates) appears under a single key. BR-24's own reproduction deck does work: playRig with "concrete", "cóncrete", "quokka", "mesa", "parrot" is green at HEAD and, with usedSource removed, reports the quokka and parrot questions each drawing two senses of the concrete entry. Swap the fixture, and correct the plan's round-6 "Both mutation-verified" to what was actually run.
- **BR-28** [Minor] `answer-must-define-the-prompted-word` Source is Entry.Headword(), which the same file documents as not being the entry's identity
  4th finding in this family, so the rule rather than the site: there is ONE entry identity, and it is the head token run entryDefines already walks (optionpool.go:158-168) — not Headword(). optionpool.go:100 and :123 set Source from Headword(), which parse.go builds from fields[0]; measured on the committed corpus, that is "hot" for `hot dog` and "a" for `a priori`. Two different entries can therefore share a Source, and PickOptions' usedSource key silently drops one of their options — a learner with both `hot dog` and `hot` in the deck loses a distractor, and on a small deck loses the form entirely to Recall. The direction of failure is over-dedup, never a wrong answer, which is why this is Minor and not a repeat of the Criticals. Fix: extract the token run as `entryIdentity(e Entry) string`, have entryDefines compare against it and both producers set Source from it, so the file stops carrying two answers to "which entry is this".

## Open findings

- **BR-13** [Minor] `docs-restate-behaviour-inaccurately` Five doc comments state behaviour the code does not have
- **BR-20** [Important] `guard-passes-without-the-property` Done-when row 1 claims a red-when that TestPickOptionsHasOneAnswer cannot deliver
- **BR-21** [Important] `docs-restate-behaviour-inaccurately` README and atlas enumerate the form-2.1 fallback causes and omit the derivative redirect
- **BR-25** [Minor] `plan-artifact-must-match-tree` fallbackReasons has no Core-concepts row, and the recorded derivation procedure cannot see package main
- **BR-26** [Minor] `uncontrolled-test-input` The pty form-2.3 test's "at least one miss" assertion depends on today's date
- **BR-27** [Important] `guard-passes-without-the-property` TestNoQuestionDrawsTwoOptionsFromOneEntry passes with BR-24's fix removed, and with Source never set at all
- **BR-28** [Minor] `answer-must-define-the-prompted-word` Source is Entry.Headword(), which the same file documents as not being the entry's identity

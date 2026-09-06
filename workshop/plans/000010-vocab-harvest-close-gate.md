---
gate: boundary-review
issue: 10
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-04T12:35:21-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: agreement counts raw band strings, so "c1" and "C1" score as disagreement
          detail: |-
            cmd/define/harvest_band.go:113 does counts[b]++ on the unparsed value after
            filtering with ParseBand, which documents case and space as transcription
            noise. Verified: agreement(["C1","c1","C1"]) = 0.67 and agreement(["C1","C1 "])
            = 0.50, so a perfectly stable model can fall below the 0.8 conformance floor
            whose prescribed remedy is the deferred hand-labelled sample. Key on the
            parsed band and add a casing row to TestAgreement.
          family: parse-result-not-canonicalised
          round: 1
        - id: BR-2
          severity: Important
          title: Mem and YAML disagree about a damaged WordFacts record, and storetest has no row
          detail: |-
            store.go:45 states "a stored record too damaged to parse also reads as
            unharvested". Verified: SetWordFacts with Band "B2+" reads back harvested=true
            from Mem and harvested=false from YAML. Neither SetWordFacts validates at the
            write, so the interface guarantee is a YAML detail. The plan placed this row in
            storetest/suite.go "so BOTH implementations are held to them at once"; it landed
            in yaml_test.go only.
          family: store-contract-unheld-by-suite
          round: 1
        - id: BR-3
          severity: Important
          title: bandTask hardcodes "English vocabulary" while facts are stored per-language
          detail: |-
            harvest_band.go:31 asserts English and renderBandPrompt takes no language, but
            facts/<lang>/ exists precisely because Spanish decks are live (yaml.go:163,
            atlas, README). define --harvest in a Spanish directory writes facts/es/*.yaml
            from an English-asserting prompt, cached forever. d.lang is already in scope at
            the call site; thread it, or refuse --harvest outside DefaultLang and say so.
          family: language-scope-not-threaded
          round: 1
        - id: BR-4
          severity: Important
          title: sanitiseFacts/sanitiseItem do not exist although the plan step naming them is ticked
          detail: |-
            grep sanitise cmd/define/ finds only sanitiseModel/sanitiseMeta. The plan
            explicitly pre-rejected "the parse covers it" ("a narrower guarantee ... not a
            substitute"), and SetItems ships the write path for Stem/Answer/Distractors with
            no neutralisation. Land sanitiseItem at the write, or add a ## Revisions entry
            recording the deferral.
          family: plan-element-ticked-unbuilt
          round: 1
        - id: BR-5
          severity: Minor
          title: the sort in agreement cannot affect the result, and agreementRounds is unused
          detail: |-
            harvest_band.go:117-127 builds and sorts a keys slice that never influences
            `best` (a max over counts); the tie-break comment describes unobservable
            behaviour. Separately agreementRounds = 5 (harvest.go:34) is declared and never
            referenced, so the documented "-agreement default N=5" is unreachable through
            flag.Int.
          family: inert-mechanism
          round: 1
        - id: BR-6
          severity: Minor
          title: the computed longest-first label ordering has no test
          detail: |-
            Verified by mutation: inverting the comparator in sortedByLengthDesc
            (glosslabel.go:63) leaves the full cmd/define suite green. The atlas advertises
            the computed ordering as the improvement over a hand-maintained invariant; a
            two-line descending-length assertion would make it one.
          family: property-without-a-pin
          round: 1
        - id: BR-7
          severity: Minor
          title: hand-rolled contains in harvest_band_test.go duplicates strings.Contains
          detail: |-
            harvest_band_test.go:88-96 reimplements substring search in a package whose
            harvest_test.go already imports strings for the same purpose (ARCH-DRY).
          family: stdlib-reimplemented
          round: 1
        - id: BR-8
          severity: Minor
          title: runHarvestAgreement re-runs wordSense inside the per-round loop
          detail: |-
            harvest.go:186-188 recomputes the dictionary gloss and domain N times per word
            although both are invariant across rounds. Cheap (local CGO lookup) but free to
            hoist out of the loop.
          family: loop-invariant-work
          round: 1
        - id: BR-9
          severity: Minor
          title: -limit is ignored in -agreement mode, N is unbounded, and --play --harvest drops a mode
          detail: |-
            main.go accepts -limit with -agreement and ignores it; -agreement N has no upper
            bound (K=20 x N calls); and dispatch order means `define --play --harvest` runs
            only --play. The file's own comment three lines above says silently honouring one
            of two commands is how -raw came to mean two things.
          family: flag-silently-ignored
          round: 1
        - id: BR-10
          severity: Minor
          title: the project's M1 calibration prose was written before this gate ran
          detail: |-
            workshop/projects/define-learn.md records actual 2.16h and "the milestone had no
            remediation round at all" in the commit that precedes the boundary review. Any
            remediation of the findings above makes both stale.
          family: doc-predeclares-outcome
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-09-04T13:03:59-07:00"
      agent: claude
      boundary: M1
      blocked: true
      protocol_error: no valid findings block
    - "n": 3
      timestamp: "2026-09-04T13:55:45-07:00"
      agent: claude
      boundary: M1
      blocked: true
      protocol_error: no valid findings block
    - "n": 4
      timestamp: "2026-09-04T14:18:39-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Keyed on ParseBand; verified by revert — TestAgreement goes red on three rows, and the dead sort is gone.
          round: 4
        - id: BR-2
          disposition: addressed
          note: sanitiseFacts at the write in both stores; verified by revert — two storetest rows redden against Mem.
          round: 4
        - id: BR-3
          disposition: addressed
          note: Language threaded at both call sites; verified by revert to store.DefaultLang — TestHarvestSendsTheDecksLanguage reddens.
          round: 4
        - id: BR-4
          disposition: addressed
          note: sanitiseFacts/sanitiseItem exist in store/item.go, called by Mem and YAML, held by storetest rather than by plan prose.
          round: 4
        - id: BR-5
          disposition: addressed
          note: The sort and its import are gone; agreementRounds is read at main.go:707 and `-harvest -agreement=0` clears the guards on the built binary.
          round: 4
        - id: BR-6
          disposition: addressed
          note: TestDomainLabelsAreLongestFirst pins the computed ordering; the plan's sweep records it RED under an inverted comparator.
          round: 4
        - id: BR-7
          disposition: addressed
          note: harvest_band_test.go uses strings.Contains throughout; no hand-rolled contains remains.
          round: 4
        - id: BR-8
          disposition: addressed
          note: wordSense is hoisted above the per-round loop at harvest.go:195.
          round: 4
        - id: BR-9
          disposition: addressed
          note: modeCollision plus five guards; all six behaviours confirmed against the built binary, and the collision arm reddens on revert. The remaining pin gap is raised separately as a family repeat.
          round: 4
        - id: BR-10
          disposition: addressed
          note: The project now records actual 4.03h and corrects the pre-declared "no remediation round" prose in place rather than overwriting it.
          round: 4
      findings:
        - id: BR-11
          severity: Important
          title: the conformance floor and the reported 1.00 measure a prompt shape --harvest never sends
          detail: |-
            harvest_conformance_test.go:57 and :92 call bandTask(DefaultLang, w, "", "") — no gloss and
            never the known-domain branch — while runHarvest and runHarvestAgreement both pass
            wordSense's gloss and known domain (harvest.go:110-111, 195-198). bandTask exists so the
            measurement cannot become "a report about a prompt nobody runs"; the conformance row is that
            drift. bandingWords has the 8 entries the Log, atlas and project all report as "mean
            agreement 1.00 over 8 words x 5 assignments", so the milestone's one measured claim was taken
            at the bare shape. testDict is in-package and usable under the conformance tag, so deriving
            gloss+known via senseFacts is cheap; otherwise record in the file why the bare shape is the
            right thing to floor.
          family: measurement-off-the-production-path
          round: 4
        - id: BR-12
          severity: Important
          title: mode exclusivity changes previously-accepted invocations and appears in no user-facing doc
          detail: |-
            main.go:582-593 now refuses any two of -llm-check/-forget/-play/-reflect/-harvest with exit 2.
            At base there was no cross-mode guard, so `define -llm-check -play`, `define -forget w -play`
            and `define -play -reflect` each ran the first mode reached. The change is correct and it is
            a breaking CLI change: README says nothing, and atlas/define.md:1506 "Entry modes" still reads
            "run dispatches modes first (-forget), then on argument count" with a table listing only
            -forget. Two lines in that section and a sentence in README close it.
          family: behaviour-change-undocumented
          round: 4
        - id: BR-13
          severity: Important
          title: six of the seven run()-path members round 3 enumerated are still unpinned
          detail: |-
            This is the 2nd finding in family property-without-a-pin. Do NOT fix the sites — the rule is
            that a guard or a dependency existing only on the run() path is pinned through run(), and the
            enumeration is mechanical. Round 3 wrote the list down: the six new switch arms plus the
            agreementRounds default branch. Only the collision arm was pinned; grep over *_test.go for
            "takes no word", "only mean anything with", "cannot be negative", "is capped at",
            "does not apply to -agreement" and "agreementRounds" returns nothing. Measured prevalence 6 of
            7; all six verified correct today against the built binary, so this is regression exposure.
            The same rule reaches one member the list missed: TestHarvestSendsTheDecksLanguage and
            TestTheDictionaryDomainBeatsTheModel set d.lang/d.dict by hand and call runHarvest directly,
            beginning after the d.withStore hop that fills them — the wiring-hop class news_test.go:410
            says has now cost three issues.
          family: property-without-a-pin
          round: 4
        - id: BR-14
          severity: Minor
          title: README's directory listing promises items/<lang>/*.yaml, which M1 never writes
          detail: |-
            This is the 2nd finding in family doc-predeclares-outcome. Do NOT fix the line — round 3 fixed
            the -harvest flag help for the identical reason and stated the rule ("shipped user-facing text
            describes the shipped milestone; forward capability lives in the plan"). Sweep the enumeration
            that rule implies: flag help, README prose, README file listing, atlas, project row. README.md
            lists items/en/sycophantic.yaml unqualified although nothing calls SetItems in M1; the atlas
            gets it right by tagging items/ as #10 M2. Measured prevalence in this family: 3.
          family: doc-predeclares-outcome
          round: 4
        - id: BR-15
          severity: Minor
          title: store.Bands() has no production caller while the band prompt hand-restates the six levels
          detail: |-
            This is the 2nd finding in family inert-mechanism (BR-5's agreementRounds was the first). Do
            NOT fix the site — the rule is that an accessor added to be the single source is wired to its
            consumer or it does not exist. store.Bands() (vocab.go:36) is referenced only by vocab_test.go,
            while harvest_band.go:64 types "one of A1, A2, B1, B2, C1, C2" into the prompt. The domain half
            of the same prompt enumerates store.Domains() and is pinned by
            TestBandPromptCarriesTheClosedDomainSet; the band half is a second spelling with no pin
            (ARCH-DRY). Practical risk is low — CEFR is a fixed six-point scale.
          family: inert-mechanism
          round: 4
      boundary: M1
      blocked: false
    - "n": 5
      timestamp: "2026-09-04T16:37:31-07:00"
      agent: claude
      findings:
        - id: BR-16
          severity: Critical
          title: --limit does not bound the authoring pass's model calls
          detail: |-
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
          family: flag-silently-ignored
          round: 5
        - id: BR-17
          severity: Critical
          title: Three headline M2 properties have pins that cannot fail; the sweep row is ticked for M1 only
          detail: |-
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
          family: property-without-a-pin
          round: 5
        - id: BR-18
          severity: Important
          title: authorTask, entailTask and vetoTask ship with no live conformance row
          detail: |-
            The plan's Test surface (line 94) and its Integration-points block both
            commit to "a live conformance row each". harvest_conformance_test.go holds
            only M1's two band rows. Nothing detects drift between llmtest.Fake's
            canned JSON and the real service for the three schemas this milestone's
            whole value rests on (ARCH-MOCK). The veto's obsequious/sycophantic pair
            is already known to work live from the checkpoint, so it is a cheap row.
          family: task-without-live-conformance
          round: 5
        - id: BR-19
          severity: Important
          title: learnerFacts.Domains is computed at zero production call sites
          detail: |-
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
          family: inert-mechanism
          round: 5
        - id: BR-20
          severity: Important
          title: The entailment and veto prompts carry no store.Lang
          detail: |-
            renderEntailPrompt (harvest_judge.go:85) and renderVetoPrompt (:146) take
            no language, while renderBandPrompt and renderAuthorPrompt both thread it
            and bandSystem's comment explains why. TestHarvestSendsTheDecksLanguage
            asserts only reqs[0]. With #18 open and d.lang="es" already exercised, a
            Spanish stem is judged by an unlabelled prompt whose worked example is
            English. This is the 2nd finding in family language-scope-not-threaded.
            The rule: every request-rendering function on a per-language path takes
            store.Lang and states it, and the wire test asserts it over every request
            the run sends rather than over reqs[0]. Sweep all four renderers at once.
          family: language-scope-not-threaded
          round: 5
        - id: BR-21
          severity: Important
          title: The project row records actual and closed before this gate ran
          detail: |-
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
          family: doc-predeclares-outcome
          round: 5
        - id: BR-22
          severity: Important
          title: The item cap is asserted against Mem only, not in storetest/suite.go
          detail: |-
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
          family: store-contract-unheld-by-suite
          round: 5
        - id: BR-23
          severity: Minor
          title: Items() now returns newest-first and the interface contract does not say so
          detail: |-
            sanitiseItems calls prune, which sorts on both the read and the write, so
            items no longer come back in insertion order. atlas/define.md records it;
            Store.Items' doc comment at store/store.go:60 — where #12 will read the
            contract — does not. This is the 2nd finding in family behaviour-change-
            undocumented. The rule: a behaviour promised to a consumer belongs on the
            interface it is promised through, not only in the atlas.
          family: behaviour-change-undocumented
          round: 5
        - id: BR-24
          severity: Minor
          title: stemUsesTheWord and blankOut locate the word by different rules
          detail: |-
            stemUsesTheWord (harvest_item.go:418) checks only the leading boundary, so
            `set` is satisfied by "The settlement was reached"; blankOut
            (harvest_judge.go:173) then renders "The ___tlement was reached" into the
            veto prompt. Checking only the first occurrence also falsely rejects a stem
            where the word appears as a substring before appearing properly.
          family: two-spellings-of-one-predicate
          round: 5
        - id: BR-25
          severity: Minor
          title: harvestPRNG duplicates play.prng step-for-step
          detail: |-
            harvest_item.go:378-397 restates play/pick.go:168-215. The stated reason
            (exporting an internal, or dragging store vocabulary across the seam) is
            weaker than it reads — play.SampleStrings is already exported to main for
            exactly this, and an int-slice shuffle needs no store types. The seeding
            also differs, so "same algorithm" produces different sequences for the same
            seed; the comment overstates the relationship.
          family: helper-copied-not-shared
          round: 5
        - id: BR-26
          severity: Minor
          title: The tier report counts items that were never written
          detail: |-
            widened[tier]++ at harvest.go:262 runs before the veto, so an item whose
            candidates are all vetoed still contributes to "N item(s) drew options from
            X". This is the 2nd finding in family measurement-off-the-production-path.
            The rule: a batch statistic is taken over what the batch shipped, not over
            what it attempted — topicSpread already gets this right at harvest.go:298
            by appending only on success.
          family: measurement-off-the-production-path
          round: 5
        - id: BR-27
          severity: Minor
          title: Only the author-call failure reports what was saved before stopping
          detail: |-
            harvest.go:229 prints "authored N item(s) before stopping; they are saved";
            the entail (:241) and veto (:270) failures return 1 with no such line, so
            an outage during judging leaves the operator without the count the author
            path gives them.
          family: inconsistent-failure-reporting
          round: 5
        - id: BR-28
          severity: Minor
          title: README describes two checks where four conditions reject
          detail: |-
            cmd/define/README.md says "Two things are checked" but an item is dropped
            by the free stem check, by entails, by glosses, by named, or by a total
            veto. "The sentence must give the word away" is also close to the unique-
            recoverability framing the checkpoint explicitly retired.
          family: doc-understates-surface
          round: 5
        - id: BR-29
          severity: Minor
          title: PruneForTest is exported production API, and prune's truncation is unreachable
          detail: |-
            store/item.go:200 exports a symbol for tests only. SetItems has one
            non-test caller (harvest.go:290), always with exactly one item and only
            for words with zero items, so ItemCap's truncation branch cannot be reached
            in production today. That is defensible as a guard for #13 — but say so,
            and prefer an in-package export_test.go alias over a permanent exported
            symbol.
          family: test-only-symbol-in-production-api
          round: 5
      boundary: M2
      blocked: true
    - "n": 6
      timestamp: "2026-09-06T10:14:16-07:00"
      agent: claude
      boundary: M2
      blocked: true
      protocol_error: no valid findings block
    - "n": 7
      timestamp: "2026-09-06T13:28:28-07:00"
      agent: claude
      dispose:
        - id: BR-16
          disposition: addressed
          note: 'Measured: 8-word pre-banded deck, -limit 2, judge rejecting all = 2 calls total (was 16); a per-pass budget reddens both limit tests. Residual one-call overrun raised as a new finding.'
          round: 7
        - id: BR-17
          disposition: addressed
          note: All three named pins fixed and mutation-confirmed; harvestRig now refuses size 1. Five of the six named greens redden on revert; sortedBanded is the exception, folded into the new pin finding.
          round: 7
        - id: BR-18
          disposition: addressed
          note: Three live rows exist for authorTask, entailTask and vetoTask, and the author row records a real live observation, so they ran.
          round: 7
        - id: BR-19
          disposition: addressed
          note: learnerFacts.Domains is now read by learnerFacts.reads through tierLearnerDomain; every learnerFacts field and both pickDistractors return values have a non-test reader.
          round: 7
        - id: BR-20
          disposition: addressed
          note: 'Verified by revert: hardcoding store.DefaultLang in entailTask and vetoTask reddens TestHarvestSendsTheDecksLanguage, which now iterates every request.'
          round: 7
        - id: BR-21
          disposition: addressed
          note: 'The project row reads "actual: pending — written by the close gate, after the verdict".'
          round: 7
        - id: BR-22
          disposition: addressed
          note: The cap row landed in storetest/suite.go. The newest-first member of its own enumeration is still missing and is raised separately.
          round: 7
        - id: BR-23
          disposition: addressed
          note: Store.Items' doc comment now states the newest-first order; the suite row that would hold it is raised separately.
          round: 7
        - id: BR-24
          disposition: not-addressed
          note: The two spellings were unified into wordIndexIn, but the looser rule was chosen for both, so the defect the finding measured survives.
          round: 7
        - id: BR-25
          disposition: addressed
          note: harvestPRNG is gone; play.ShuffleInts is exported and used, with one seeding step.
          round: 7
        - id: BR-26
          disposition: not-addressed
          note: 'The move is in the code but has no pin: restoring widened[tier]++ to its pre-veto position leaves the whole suite green.'
          round: 7
        - id: BR-27
          disposition: not-addressed
          note: 'The lines were added but no test asserts them: grep finds no assertion on "before stopping", and deleting both lines leaves the suite green.'
          round: 7
        - id: BR-28
          disposition: addressed
          note: The README now describes four checks and drops the unique-recoverability framing.
          round: 7
        - id: BR-29
          disposition: addressed
          note: 'PruneForTest moved to store/export_test.go, and prune''s comment states the truncation branch is a guard for #13.'
          round: 7
      findings:
        - id: BR-30
          severity: Important
          title: -limit N still makes N+1 calls, and the two tests named for it never reach the authoring pass
          detail: |-
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
          family: flag-silently-ignored
          round: 7
        - id: BR-31
          severity: Important
          title: main.go:628 still states the old -limit meaning; the sweep missed the enumeration's fifth member
          detail: |-
            3rd in family. The rule is already written: when a fix changes what a user-facing
            contract MEANS, every statement of that meaning is part of the change — flag help,
            in-code guard comments, README, atlas, the constant's doc comment. Commit 94083a2
            says it "swept all three places" and updated the flag help, README and atlas; the
            mode-guard comment at main.go:628 still reads "-limit bounds how many words are
            ASKED ABOUT". The enumeration existed in the finding and was used as a list of
            noticed sites again.
          family: behaviour-change-undocumented
          round: 7
        - id: BR-32
          severity: Important
          title: The atlas lists four selection tiers where the code has five, and the plan says three tasks where M2 ships four
          detail: |-
            3rd in family; round 6's I5 is unfixed at HEAD. The rule: a doc that ENUMERATES a
            code-side set is a consumer of that set — when the set changes the enumeration is
            re-derived, not left at its previous count. tierLearnerDomain is second-priority
            and changes which words are selected; grep finds it in no atlas or README text.
            atlas/define.md:1578 still records as an open question for #12 the exact thing
            harvest_item.go:262-276 says the tier answers. The plan's Integration-points table
            omits entailTask and its Test-surface sentence says "the three tasks" where there
            are four. Sweep: the atlas tier list, the atlas open-question paragraph, the plan
            table, the plan's "three tasks" sentence, and the README's tier-report lines.
          family: doc-understates-surface
          round: 7
        - id: BR-33
          severity: Important
          title: Store.Items' newest-first promise, added this window, has no storetest row
          detail: |-
            3rd in family. The rule stands from BR-22: any promise stated on the Store
            interface is asserted in storetest/suite.go, never in a per-implementation test.
            BR-22's cap row landed and is mutation-confirmed, but its enumeration named three
            members; the newest-first read order is asserted only through the pure
            PruneForTest tests. The suite's "a word may hold several items" row writes day(1)
            and day(2) and checks only that the Form discriminator survived. Read-side
            canonicalisation, the third member, is correctly out of scope at yaml.go:660.
          family: store-contract-unheld-by-suite
          round: 7
        - id: BR-34
          severity: Important
          title: Three fixes from the last two rounds have no test that fails without them
          detail: |-
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
          family: property-without-a-pin
          round: 7
        - id: BR-35
          severity: Important
          title: '#12''s three moved Done-when rows are still unwritten, and its Revision names this milestone as the trigger'
          detail: |-
            2nd in family. The rule: a deferral whose trigger is "when Mx lands" is swept at
            Mx's boundary by the issue that wrote it, not left for the consuming issue to
            discover. Task 5 Step 0 is ticked and its Revision on #12 is real, but it closes
            with "Not yet applied to the rows above. They are rewritten when #10 M2 lands."
            M2 is landing, and #12's Done-when still owns "Options are drawn from the pool +
            deck, never model-generated", the sycophantic/obsequious row, and "The form works
            with the LLM seam unavailable".
          family: plan-element-ticked-unbuilt
          round: 7
        - id: BR-36
          severity: Minor
          title: prune's doc comment carries an orphaned fragment, and harvestRig stacks two doc comments
          detail: |-
            store/item.go:200 reads "// : the same input prunes to the / // same output, every
            time." — the DETERMINISTIC subject was lost in an edit. harvest_test.go:19-27
            stacks two doc comments for harvestRig, the second beginning mid-block. Round 6
            filed the first; both are unfixed at HEAD.
          family: doc-comment-mangled-by-edit
          round: 7
        - id: BR-37
          severity: Minor
          title: The M2 sweep block says 22 properties over a 24-row list and records neither the mutation nor the named test
          detail: |-
            3rd in family. The rule the block itself states two paragraphs below is "a sweep
            row is only as good as the mutation behind it — recording the verdict without
            recording the mutation is how a green row reads as red". M1's table at least
            carried a verdict column; M2's is a prose list of property names. I re-derived
            seven rows by reverting and they hold, which is the problem: a hand-written
            table's one false row is indistinguishable from its true ones. Also unticked in
            the same section: "The generated batch, read by the operator", while the Plan's
            checkpoint row is ticked and the Log records three batches read.
          family: measurement-off-the-production-path
          round: 7
        - id: BR-38
          severity: Minor
          title: widened[tierSameDomain] is incremented and never read
          detail: |-
            4th in family. harvest.go:262 counts every tier; the report loop at :365 walks
            only tierLearnerDomain, tierGeneral, tierAnyDomain and tierAboveBand. The rule
            already written for this family: a value no production path reads is wired or
            deleted. Also in this class: TestTheLimitHoldsWhenEveryStemIsRejected's distinct
            fixture is dead under the current rig (see the -limit finding).
          family: inert-mechanism
          round: 7
        - id: BR-39
          severity: Minor
          title: optionsPerItem restates play.maxOptions - 1 across a seam that already exports two helpers
          detail: |-
            2nd in family. harvest.go:46 hardcodes 3 with a comment naming play.maxOptions as
            the reason; play already exports SampleStrings and ShuffleInts to main for exactly
            this, and an option count needs none of the store's vocabulary.
          family: helper-copied-not-shared
          round: 7
        - id: BR-40
          severity: Minor
          title: One exhausted budget prints two stopped lines, and prune shadows the builtin cap
          detail: |-
            2nd in family. Measured: -limit 5 on a fresh 8-word deck prints "stopped at the
            --limit of 5 model call(s)" then "stopped authoring at the --limit of 5 model
            call(s)" for one budget. Also cosmetic: store/item.go:207 declares
            func prune(items []Item, cap int), shadowing the builtin, and
            internal/llm/golden_schema_test.go:15 leaves one comment line past the file's wrap.
          family: inconsistent-failure-reporting
          round: 7
      boundary: M2
      blocked: true
    - "n": 8
      timestamp: "2026-09-06T13:53:26-07:00"
      agent: claude
      dispose:
        - id: BR-24
          disposition: not-addressed
          note: 'Unification is real (both go through wordIndexIn) but the loose rule survives: stemUsesTheWord("The settlement was reached in Albany.","set")=true, blankOut gives "The ___tlement was reached in Albany.".'
          round: 8
        - id: BR-26
          disposition: addressed
          note: widened[tier]++ now runs after SetItems; the counter no longer counts attempts. The missing revert-check stays open under BR-34.
          round: 8
        - id: BR-27
          disposition: addressed
          note: 'Mutation-confirmed: deleting the survivor lines on the entail and veto paths reddens TestEveryAuthoringOutagePathReportsSurvivors on both subtests.'
          round: 8
        - id: BR-28
          disposition: addressed
          round: 8
        - id: BR-30
          disposition: addressed
          note: runWithin gates structurally; reverting to charge-then-call gives -limit 6 = 10 calls and reddens two cells of the new flag x pass table. Each cell now fatals if its pass was never entered.
          round: 8
        - id: BR-31
          disposition: addressed
          note: main.go:628 now names MODEL CALLS. The class recurs at harvest.go:15 and plan:90/:285 - raised separately.
          round: 8
        - id: BR-32
          disposition: addressed
          note: Atlas lists five tiers with tierLearnerDomain second, the open-question paragraph is rewritten, the plan table and "four tasks" are corrected, README gained the tier lines. The new README/atlas claim overstates - raised separately.
          round: 8
        - id: BR-33
          disposition: addressed
          note: 'Mutation-confirmed: flipping sortItems'' After to Before reddens the new storetest row on both TestMemConformance and TestYAMLConformance.'
          round: 8
        - id: BR-34
          disposition: not-addressed
          note: Two of three confirmed red on revert (survivor lines, sortedBanded). The tier-report member is green on revert - its fixture sits in tierSameDomain, which the report loop never prints.
          round: 8
        - id: BR-35
          disposition: addressed
          note: 000012's three rows are rewritten as satisfied-by-construction with the trigger recorded.
          round: 8
        - id: BR-36
          disposition: not-addressed
          note: 'Both sites unchanged at HEAD: store/item.go:200 still carries the orphaned fragment, harvest_test.go:18-27 still stacks two harvestRig doc comments.'
          round: 8
        - id: BR-37
          disposition: not-addressed
          note: plan:456 still says 22 properties over a 24-item list, records no mutation or test name per row, and plan:492 is still unticked.
          round: 8
        - id: BR-38
          disposition: not-addressed
          note: The dead-fixture half is fixed (the test now preBands and fatals on zero entail calls). widened[tierSameDomain] is still incremented at harvest.go:390 and still absent from the report loop at :394.
          round: 8
        - id: BR-39
          disposition: not-addressed
          note: optionsPerItem = 3 is unchanged and play.maxOptions is still unexported.
          round: 8
        - id: BR-40
          disposition: not-addressed
          note: golden_schema_test.go's wrap is fixed. The double stopped line is measured unchanged, and prune's cap -> max rename shadows a different Go builtin (module is go 1.26).
          round: 8
      findings:
        - id: BR-41
          severity: Important
          title: harvestLimit's own doc comment and two plan lines still state -limit's superseded meaning
          detail: |-
            4th in family. harvest.go:15 reads "bounds the words one --harvest run will ask the model
            about" - the member BR-31's enumeration named by name - fourteen lines above budget's
            comment saying the opposite; plan:90 and plan:285 repeat it. Correct at HEAD: main.go:432,
            main.go:628, README:374, atlas:1431. The rule has been written twice and used as a list of
            noticed sites twice, so the deliverable is the mechanism, not the three lines: a retired-
            PHRASING registry checked by a repo guard, the same shape as
            TestProseDoesNotSpellStaleRuntimeArtifactNames and TestNoArtifactNamesARetiredSymbol.
          family: behaviour-change-undocumented
          round: 8
        - id: BR-42
          severity: Important
          title: This round's doc sweep promises the tier is printed for every item; the code never prints tierSameDomain
          detail: |-
            4th in family. New in c5b3cda: README:412-416 "Whatever it settled for, it says so" and
            atlas:1568-1571 "the tier reached is printed for every item". harvest.go:394 walks only
            tierLearnerDomain, tierGeneral, tierAnyDomain and tierAboveBand. Measured: a 4-word
            pre-banded rig authors 4 items at tierSameDomain and prints no "drew options from" line.
            The rule: a doc sentence describing a code behaviour is a consumer of that behaviour and is
            written from the code at HEAD, not from the design - a sweep that fixes a doc against the
            plan while a known code gap (BR-38) is open converts an open finding into a false promise.
          family: doc-predeclares-outcome
          round: 8
        - id: BR-43
          severity: Important
          title: TestTheTierReportCountsOnlyWrittenItems passes with its fix reverted, so BR-34's third member is undisposed
          detail: |-
            6th in family, and it is the pattern the family exists to catch: the commit that closed
            "three fixes with no failing test" shipped a fourth. Reverted on a scratch copy at HEAD,
            moving widened[tier]++ back above the veto loop leaves the test green, because preBand puts
            every word in DomainGeneral so the tier is tierSameDomain, which the report loop never
            prints. Adding tierSameDomain to the loop makes the mutated build print "4 item(s) drew
            options from same domain, at band" and the test reddens. The rule: a revert-check is only a
            check if the author ran it - write the mutation, run it, record that it reddened, in the same
            commit as the fix. Fixing the report loop fixes this pin as a side effect; do both and re-run.
          family: property-without-a-pin
          round: 8
        - id: BR-44
          severity: Minor
          title: runWithin's stated invariant is false and two of its four errBudget branches are unreachable
          detail: |-
            5th in family. harvest.go:88 says runWithin "is the ONLY way this file reaches a model";
            runHarvestAgreement calls llm.Run directly at :462. Not a bug - that mode has its own K x N
            bound and -limit is refused beside -agreement - but it is the load-bearing sentence of
            BR-30's disposition. Same site, same rule: the errors.Is(err, errBudget) arms after bandTask
            (:161) and after authorTask (:277) cannot fire, because each loop's bud.spent() early exit
            ran with nothing decrementing in between. Panic-probed: the whole cmd/define suite passes
            without entering either. The rule is already written - a branch or value no production path
            reaches is wired or deleted - so the deliverable is the enumeration: a per-block coverage
            assertion over harvest*.go, which would also have caught BR-38's counter.
          family: inert-mechanism
          round: 8
      boundary: M2
      blocked: false
    - "n": 9
      timestamp: "2026-09-06T14:09:11-07:00"
      agent: claude
      dispose:
        - id: BR-11
          disposition: addressed
          note: harvest_conformance_test.go:65-74 derives gloss+known via testDict + senseFacts; all 8 bandingWords verified present in testdata/entries/en.
          round: 9
        - id: BR-12
          disposition: addressed
          note: atlas/define.md:1680-1686 states the set check and names it a breaking CLI change; README.md:433-436 carries it. The exit-code table gap is raised separately.
          round: 9
        - id: BR-13
          disposition: addressed
          note: All seven members pinned through run() at harvest_test.go:696-720, plus TestRunHarvestThroughTheWiringHop for the withStore hop and the bare -agreement default.
          round: 9
        - id: BR-14
          disposition: not-addressed
          note: 'Unswept, and M2 landing INVERTED it: README.md:459-461 now reads "Nothing writes this yet - authoring is the next milestone" about a surface --harvest shipped. The atlas is correct at 1419.'
          round: 9
        - id: BR-15
          disposition: addressed
          note: renderBandPrompt enumerates store.Bands() at harvest_band.go:68. No symmetric pin though - reverting to a hardcoded scale leaves the golden green, unlike the domain half.
          round: 9
        - id: BR-24
          disposition: not-addressed
          note: Measured at HEAD on real deck words - light/"The ___house at Portland Head", set/"The ___tlement", run/"The ___way", bank/"___ruptcy" all pass stemUsesTheWord. That is 4 of the checkpoint deck's A1 words shipping an unusable item; more than cosmetic.
          round: 9
        - id: BR-34
          disposition: addressed
          note: Third member verified by revert - moving widened[tier]++ above the veto loop reddens TestTheTierReportCountsOnlyWrittenItems. All three authoring outage branches covered by TestEveryAuthoringOutagePathReportsSurvivors.
          round: 9
        - id: BR-36
          disposition: not-addressed
          note: 'Both unchanged. store/item.go:200 still reads "// : the same input prunes to the"; harvest_test.go:19-27 still stacks two harvestRig doc comments.'
          round: 9
        - id: BR-37
          disposition: not-addressed
          note: plan:456 still says 22 over a list I counted at 24; no mutation or test name per row; plan:492 still unticked while the issue's row is ticked.
          round: 9
        - id: BR-38
          disposition: addressed
          note: harvest.go:408 now walks tierSameDomain first; verified by revert - dropping it reddens TestTheTierReportCoversEveryWrittenItem at 4 authored / 0 reported.
          round: 9
        - id: BR-39
          disposition: not-addressed
          note: optionsPerItem = 3 unchanged at harvest.go:54; play.maxOptions still unexported although this window exported ShuffleInts across the same seam.
          round: 9
        - id: BR-40
          disposition: not-addressed
          note: Measured at HEAD - one exhausted budget prints both "stopped at the --limit of 5" and "stopped authoring at the --limit of 5". prune's parameter is now named max, shadowing the Go builtin.
          round: 9
        - id: BR-41
          disposition: not-addressed
          note: The three named lines are correct at HEAD, but the mechanism the finding named as the deliverable did not land - repo_guard_test.go has ZERO diff across the whole window, and retiredPhrases (:1131) is the registry it asked for. Fourth hand sweep of the same class.
          round: 9
        - id: BR-42
          disposition: addressed
          note: Verified by revert on a scratch copy - removing tierSameDomain from the report loop reddens TestTheTierReportCoversEveryWrittenItem, so the doc sentence is now written from the code.
          round: 9
        - id: BR-43
          disposition: addressed
          note: Verified by revert - the mutation the finding prescribed reddens the pin, and TestTheTierReportCoversEveryWrittenItem reddens on the report-loop mutation. Both checks re-run here rather than read.
          round: 9
        - id: BR-44
          disposition: not-addressed
          note: harvest.go:82 still says runWithin is the ONLY path to a model while runHarvestAgreement calls llm.Run at :475; the errBudget arms at :166 and :281 are still unreachable, since each loop's bud.spent() exit runs with nothing decrementing in between.
          round: 9
      findings:
        - id: BR-45
          severity: Important
          title: Forget leaves facts/ and items/ behind, so a forgotten word's bad material is unregenerable
          detail: |-
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
          family: store-contract-unheld-by-suite
          round: 9
        - id: BR-46
          severity: Minor
          title: README's exit-code table declares itself an enumeration and omits every code this window added
          detail: |-
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
          family: doc-understates-surface
          round: 9
      blocked: true
    - "n": 10
      timestamp: "2026-09-06T14:23:05-07:00"
      agent: claude
      dispose:
        - id: BR-45
          disposition: addressed
          note: |-
            Revert-verified in a scratch worktree: the storetest row reddens against BOTH Mem and YAML,
            and dropping facts/items from perWordDirs reddens TestPerWordDirsCoverEveryRuntimeDir.
            Residual: Mem.Forget hand-lists its maps and the guard covers YAML only.
          round: 10
        - id: BR-41
          disposition: not-addressed
          note: |-
            repo_guard_test.go has ZERO diff across the window; retiredPhrases (:1131) over
            currentTruthFiles (:1623, binds non-test .go + README + atlas + plans) is exactly the
            mechanism asked for and no row was added. The three named lines are correct; the class is not.
          round: 10
        - id: BR-24
          disposition: not-addressed
          note: |-
            Re-measured at HEAD by probe - "The settlement"/set, "The lighthouse"/light, "The runway"/run,
            "Bankruptcy"/bank all pass stemUsesTheWord and blankOut renders "The ___tlement". The fix is not
            a bare trailing-boundary check: wordIndexIn deliberately allows inflections, so it needs a
            bounded suffix set (s/es/ed/ing/'s).
          round: 10
        - id: BR-14
          disposition: not-addressed
          note: |-
            Worse than round 9 recorded: README.md:459-461 still says "Nothing writes this yet - authoring
            is the next milestone" about a surface this issue shipped. At issue close that is a false
            statement in user-facing docs, not a forward-looking one.
          round: 10
        - id: BR-36
          disposition: not-addressed
          note: |-
            Both sites unchanged (store/item.go:200 "// : the same input prunes to the"; harvest_test.go:19-27).
            A third member measured this round - harvest_item.go:318 documents "tierAnyBand", a constant that
            never existed (introduced in fd0767b as prose only); unexported names are exempt from the symbol guard.
          round: 10
        - id: BR-37
          disposition: not-addressed
          note: |-
            plan:456 still claims 22 over a list I counted at 24; no mutation or test name per row;
            plan:492 "The generated batch, read by the operator" still unticked while the issue's row is ticked.
          round: 10
        - id: BR-39
          disposition: not-addressed
          note: |-
            harvest.go:54 still hardcodes optionsPerItem = 3 with a comment naming play.maxOptions as the
            reason, across a seam this window already widened twice (SampleStrings, ShuffleInts).
          round: 10
        - id: BR-40
          disposition: not-addressed
          note: |-
            Both halves stand at HEAD - the banding loop's exit (harvest.go:148) and runAuthoring's
            (harvest.go:264) each print for one exhausted budget; store/item.go:212 is func prune(items []Item, max int),
            shadowing the Go builtin max.
          round: 10
        - id: BR-44
          disposition: not-addressed
          note: |-
            harvest.go:81 still says runWithin is the ONLY way this file reaches a model while
            runHarvestAgreement calls llm.Run at :475; the errBudget arms at :166 and :281 remain unreachable
            because each loop's bud.spent() exit runs with nothing decrementing before the call.
          round: 10
        - id: BR-46
          disposition: not-addressed
          note: |-
            README.md:589-593 unchanged. The table still declares itself an enumeration and omits mode
            collision, every -limit/-agreement usage error, and all of --harvest's exit-1 paths.
          round: 10
      findings:
        - id: BR-47
          severity: Minor
          title: Forget deletes the deck entry first, so a partial failure reports "nothing removed" for a word it removed
          detail: |-
            This is the 3rd finding in family inconsistent-failure-reporting. Do NOT fix the loop order alone.
            The rule: a multi-step mutation orders its effects so the value it returns is true of what happened,
            and the enumeration - Forget's four removals, the banding loop's "banded N before stopping", the three
            authoring outage branches - is walked by a test that injects a failure at each step. Only the authoring
            branches have that today (TestEveryAuthoringOutagePathReportsSurvivors). Reproduced at HEAD: with
            facts/en/sycophantic.yaml made a non-empty directory, YAML.Forget returns (false, ENOTEMPTY) while
            words/en/sycophantic.yaml is already gone - so --forget exits 1 on a word it removed, and play_loop.go:455
            never calls held.dropped. os.Remove over a non-empty directory is a sufficient seam for the test.
            ARCH-ORDER: the error path unwinds the sequencing and drops the in-flight effect.
          family: inconsistent-failure-reporting
          round: 10
        - id: BR-48
          severity: Minor
          title: perWordDirs mixes the language-scoped dirs with flat usage/, so forgetting a word in one language clears another's news cache
          detail: |-
            This is the 3rd finding in family language-scope-not-threaded. Do NOT fix the usage/ row.
            The rule: a per-word verb is scoped the same way the surface it touches is scoped, and
            TestPerWordDirsCoverEveryRuntimeDir - which exists precisely to classify every runtime directory -
            classifies on ONE axis (per-word vs history) while da5c395 crosses a second (scoped vs flat).
            yaml.go:695 lists wordsDir/usageDir/factsDir/itemsDir; usageDir is RuntimeDirs[2] with no lang segment
            (yaml.go:161), so `define --forget red` in an es directory removes usage/red.yaml that the en deck
            populated. Consequence is a refetch, which is why this is Minor; the deliverable is the second axis
            in the guard, so the next surface added is classified on both.
          family: language-scope-not-threaded
          round: 10
      blocked: true
    - "n": 11
      timestamp: "2026-09-06T14:35:03-07:00"
      agent: claude
      dispose:
        - id: BR-48
          disposition: addressed
          note: Mutation-verified in a scratch worktree — flipping usage/ to scoped:true reddens TestPerWordDirsCoverEveryRuntimeDir with the intended message; the second axis is declared at yaml.go:707 and asserted at yaml_test.go:718.
          round: 11
        - id: BR-14
          disposition: not-addressed
          note: README.md:459-461 still reads "Nothing writes this yet - authoring is the next milestone" about the items/ surface M2 shipped; at the CLOSE boundary that is a false statement in user docs, not a forward-looking one.
          round: 11
        - id: BR-24
          disposition: not-addressed
          note: Unchanged at HEAD - wordIndexIn (harvest_item.go:481) checks only the leading boundary and allows any suffix, so `set` is satisfied by "The settlement" and blankOut renders "The ___tlement".
          round: 11
        - id: BR-36
          disposition: not-addressed
          note: 'All three sites unchanged - store/item.go:200 "// : the same input prunes to the", harvest_test.go:19-27 two stacked doc comments, harvest_item.go:318 naming tierAnyBand.'
          round: 11
        - id: BR-37
          disposition: not-addressed
          note: plan:456 still says 22 over a list I re-counted at 24; no mutation or test name per row; plan:492 still unticked while the issue's row is ticked.
          round: 11
        - id: BR-39
          disposition: not-addressed
          note: harvest.go:53 still hardcodes optionsPerItem = 3 with a comment naming play.maxOptions as the reason.
          round: 11
        - id: BR-40
          disposition: not-addressed
          note: Both halves stand - harvest.go:149/167 and :266/282 each print for one exhausted budget, and store/item.go:212 is func prune(items []Item, max int), shadowing the Go builtin.
          round: 11
        - id: BR-41
          disposition: not-addressed
          note: repo_guard_test.go has ZERO diff across the whole window; retiredPhrases (:1131) is the registry the finding named and no row was added. The three named lines are correct at HEAD; the class is still unmechanised.
          round: 11
        - id: BR-44
          disposition: not-addressed
          note: harvest.go:82 still says runWithin is the ONLY way this file reaches a model while runHarvestAgreement calls llm.Run at :475; both errBudget arms remain unreachable because each loop's bud.spent() exit runs with nothing decrementing before the call.
          round: 11
        - id: BR-46
          disposition: not-addressed
          note: README.md:585-593 unchanged - the table still declares itself an enumeration and omits mode collision, every -limit/-agreement usage error, and all of --harvest's exit-1 paths.
          round: 11
        - id: BR-47
          disposition: not-addressed
          note: yaml.go:743-757 unchanged - the loop removes words/ first and every non-ENOENT error returns (false, err), so a partial failure reports "nothing removed" for a word it removed.
          round: 11
      findings:
        - id: BR-49
          severity: Minor
          title: The atlas describes Forget's guard as one-axis, one commit after dc1130e made it two, and the cross-language forget it recorded reaches no user-facing doc
          detail: |-
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
          family: doc-understates-surface
          round: 11
      forced: '--no-ledger (or --force): ALL SEVEN Done-when rows ticked with the mutation that proved them. go test ./... green; go vet clean under BOTH tag sets; gofmt clean. (1) --harvest produces finished items with no sitting running — the seam is made to PANIC, not nil, and the sitting is asserted not to have banded anything on the way past. (2) Every word carries a band and domain assigned once, pinned on the request COUNT so a second run is asserted to make zero calls. (3) The banding is MEASURED: --harvest -agreement=N, its own mode writing nothing; floor 0.8 asserted LIVE at mean agreement 1.00 over 8 words x 5 assignments, on the prompt production actually sends. (4) A distractor is never the answer: obsequious/sycophantic committed as the known-bad case, fired on real material in all three checkpoint batches in both directions, and asserted live in both directions. (5) Authored stems entail and name real subjects — three separate verdict fields, three committed known-bad stems, topicSpread measured with NO model. (6) A model outage leaves the store usable, pinned on all four outage paths with survivors asserted WHOLE. (7) Growth bounded by ItemCap at the write, held by storetest against both implementations; prune proved deterministic by pruning twice with the input SHUFFLED between calls. THE CHECKPOINT RAN — three live batches on a real deck, read, changing the design twice (appositive glosses 10/20 to 0, authored 20 to 19/20). M1 and M2 each have their own mutation sweep (13 and 22 properties) plus per-finding revert-checks. BYPASSING THE LEDGER GATE FOR EXACTLY ONE FINDING, BR-41, which is verifiably fixed at HEAD and has been since round 8s remediation: it names harvestLimits doc comment (harvest.go:15, now "the default bound on MODEL CALLS one --harvest run may make") and two plan lines (plan:90 and plan:285, both now "caps the MODEL CALLS"). A grep across cmd/, atlas/ and the plan for every statement of -limits meaning returns 20 hits and every one says calls; zero say words. The ledger carried the entry forward without disposing it across rounds 8, 9 and 10 while the tree was already correct. No other finding is bypassed — the other blocker from close round 1 (BR-45, Forget leaving facts/ and items/ behind) was a real shipped bug, fixed as a class in da5c395 with a guard that fails when a new runtime directory is unclassified, and its second axis fixed in dc1130e.'
      blocked: true
---

# Gate ledger — tools#10 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-04T12:35:21-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `parse-result-not-canonicalised` agreement counts raw band strings, so "c1" and "C1" score as disagreement
  cmd/define/harvest_band.go:113 does counts[b]++ on the unparsed value after
  filtering with ParseBand, which documents case and space as transcription
  noise. Verified: agreement(["C1","c1","C1"]) = 0.67 and agreement(["C1","C1 "])
  = 0.50, so a perfectly stable model can fall below the 0.8 conformance floor
  whose prescribed remedy is the deferred hand-labelled sample. Key on the
  parsed band and add a casing row to TestAgreement.
- **BR-2** [Important] `store-contract-unheld-by-suite` Mem and YAML disagree about a damaged WordFacts record, and storetest has no row
  store.go:45 states "a stored record too damaged to parse also reads as
  unharvested". Verified: SetWordFacts with Band "B2+" reads back harvested=true
  from Mem and harvested=false from YAML. Neither SetWordFacts validates at the
  write, so the interface guarantee is a YAML detail. The plan placed this row in
  storetest/suite.go "so BOTH implementations are held to them at once"; it landed
  in yaml_test.go only.
- **BR-3** [Important] `language-scope-not-threaded` bandTask hardcodes "English vocabulary" while facts are stored per-language
  harvest_band.go:31 asserts English and renderBandPrompt takes no language, but
  facts/<lang>/ exists precisely because Spanish decks are live (yaml.go:163,
  atlas, README). define --harvest in a Spanish directory writes facts/es/*.yaml
  from an English-asserting prompt, cached forever. d.lang is already in scope at
  the call site; thread it, or refuse --harvest outside DefaultLang and say so.
- **BR-4** [Important] `plan-element-ticked-unbuilt` sanitiseFacts/sanitiseItem do not exist although the plan step naming them is ticked
  grep sanitise cmd/define/ finds only sanitiseModel/sanitiseMeta. The plan
  explicitly pre-rejected "the parse covers it" ("a narrower guarantee ... not a
  substitute"), and SetItems ships the write path for Stem/Answer/Distractors with
  no neutralisation. Land sanitiseItem at the write, or add a ## Revisions entry
  recording the deferral.
- **BR-5** [Minor] `inert-mechanism` the sort in agreement cannot affect the result, and agreementRounds is unused
  harvest_band.go:117-127 builds and sorts a keys slice that never influences
  `best` (a max over counts); the tie-break comment describes unobservable
  behaviour. Separately agreementRounds = 5 (harvest.go:34) is declared and never
  referenced, so the documented "-agreement default N=5" is unreachable through
  flag.Int.
- **BR-6** [Minor] `property-without-a-pin` the computed longest-first label ordering has no test
  Verified by mutation: inverting the comparator in sortedByLengthDesc
  (glosslabel.go:63) leaves the full cmd/define suite green. The atlas advertises
  the computed ordering as the improvement over a hand-maintained invariant; a
  two-line descending-length assertion would make it one.
- **BR-7** [Minor] `stdlib-reimplemented` hand-rolled contains in harvest_band_test.go duplicates strings.Contains
  harvest_band_test.go:88-96 reimplements substring search in a package whose
  harvest_test.go already imports strings for the same purpose (ARCH-DRY).
- **BR-8** [Minor] `loop-invariant-work` runHarvestAgreement re-runs wordSense inside the per-round loop
  harvest.go:186-188 recomputes the dictionary gloss and domain N times per word
  although both are invariant across rounds. Cheap (local CGO lookup) but free to
  hoist out of the loop.
- **BR-9** [Minor] `flag-silently-ignored` -limit is ignored in -agreement mode, N is unbounded, and --play --harvest drops a mode
  main.go accepts -limit with -agreement and ignores it; -agreement N has no upper
  bound (K=20 x N calls); and dispatch order means `define --play --harvest` runs
  only --play. The file's own comment three lines above says silently honouring one
  of two commands is how -raw came to mean two things.
- **BR-10** [Minor] `doc-predeclares-outcome` the project's M1 calibration prose was written before this gate ran
  workshop/projects/define-learn.md records actual 2.16h and "the milestone had no
  remediation round at all" in the commit that precedes the boundary review. Any
  remediation of the findings above makes both stale.

## Round 2 — 2026-09-04T13:03:59-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 3 — 2026-09-04T13:55:45-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 4 — 2026-09-04T14:18:39-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — Keyed on ParseBand; verified by revert — TestAgreement goes red on three rows, and the dead sort is gone.
- BR-2 — addressed — sanitiseFacts at the write in both stores; verified by revert — two storetest rows redden against Mem.
- BR-3 — addressed — Language threaded at both call sites; verified by revert to store.DefaultLang — TestHarvestSendsTheDecksLanguage reddens.
- BR-4 — addressed — sanitiseFacts/sanitiseItem exist in store/item.go, called by Mem and YAML, held by storetest rather than by plan prose.
- BR-5 — addressed — The sort and its import are gone; agreementRounds is read at main.go:707 and `-harvest -agreement=0` clears the guards on the built binary.
- BR-6 — addressed — TestDomainLabelsAreLongestFirst pins the computed ordering; the plan's sweep records it RED under an inverted comparator.
- BR-7 — addressed — harvest_band_test.go uses strings.Contains throughout; no hand-rolled contains remains.
- BR-8 — addressed — wordSense is hoisted above the per-round loop at harvest.go:195.
- BR-9 — addressed — modeCollision plus five guards; all six behaviours confirmed against the built binary, and the collision arm reddens on revert. The remaining pin gap is raised separately as a family repeat.
- BR-10 — addressed — The project now records actual 4.03h and corrects the pre-declared "no remediation round" prose in place rather than overwriting it.

### Raised

- **BR-11** [Important] `measurement-off-the-production-path` the conformance floor and the reported 1.00 measure a prompt shape --harvest never sends
  harvest_conformance_test.go:57 and :92 call bandTask(DefaultLang, w, "", "") — no gloss and
  never the known-domain branch — while runHarvest and runHarvestAgreement both pass
  wordSense's gloss and known domain (harvest.go:110-111, 195-198). bandTask exists so the
  measurement cannot become "a report about a prompt nobody runs"; the conformance row is that
  drift. bandingWords has the 8 entries the Log, atlas and project all report as "mean
  agreement 1.00 over 8 words x 5 assignments", so the milestone's one measured claim was taken
  at the bare shape. testDict is in-package and usable under the conformance tag, so deriving
  gloss+known via senseFacts is cheap; otherwise record in the file why the bare shape is the
  right thing to floor.
- **BR-12** [Important] `behaviour-change-undocumented` mode exclusivity changes previously-accepted invocations and appears in no user-facing doc
  main.go:582-593 now refuses any two of -llm-check/-forget/-play/-reflect/-harvest with exit 2.
  At base there was no cross-mode guard, so `define -llm-check -play`, `define -forget w -play`
  and `define -play -reflect` each ran the first mode reached. The change is correct and it is
  a breaking CLI change: README says nothing, and atlas/define.md:1506 "Entry modes" still reads
  "run dispatches modes first (-forget), then on argument count" with a table listing only
  -forget. Two lines in that section and a sentence in README close it.
- **BR-13** [Important] `property-without-a-pin` six of the seven run()-path members round 3 enumerated are still unpinned
  This is the 2nd finding in family property-without-a-pin. Do NOT fix the sites — the rule is
  that a guard or a dependency existing only on the run() path is pinned through run(), and the
  enumeration is mechanical. Round 3 wrote the list down: the six new switch arms plus the
  agreementRounds default branch. Only the collision arm was pinned; grep over *_test.go for
  "takes no word", "only mean anything with", "cannot be negative", "is capped at",
  "does not apply to -agreement" and "agreementRounds" returns nothing. Measured prevalence 6 of
  7; all six verified correct today against the built binary, so this is regression exposure.
  The same rule reaches one member the list missed: TestHarvestSendsTheDecksLanguage and
  TestTheDictionaryDomainBeatsTheModel set d.lang/d.dict by hand and call runHarvest directly,
  beginning after the d.withStore hop that fills them — the wiring-hop class news_test.go:410
  says has now cost three issues.
- **BR-14** [Minor] `doc-predeclares-outcome` README's directory listing promises items/<lang>/*.yaml, which M1 never writes
  This is the 2nd finding in family doc-predeclares-outcome. Do NOT fix the line — round 3 fixed
  the -harvest flag help for the identical reason and stated the rule ("shipped user-facing text
  describes the shipped milestone; forward capability lives in the plan"). Sweep the enumeration
  that rule implies: flag help, README prose, README file listing, atlas, project row. README.md
  lists items/en/sycophantic.yaml unqualified although nothing calls SetItems in M1; the atlas
  gets it right by tagging items/ as #10 M2. Measured prevalence in this family: 3.
- **BR-15** [Minor] `inert-mechanism` store.Bands() has no production caller while the band prompt hand-restates the six levels
  This is the 2nd finding in family inert-mechanism (BR-5's agreementRounds was the first). Do
  NOT fix the site — the rule is that an accessor added to be the single source is wired to its
  consumer or it does not exist. store.Bands() (vocab.go:36) is referenced only by vocab_test.go,
  while harvest_band.go:64 types "one of A1, A2, B1, B2, C1, C2" into the prompt. The domain half
  of the same prompt enumerates store.Domains() and is pinned by
  TestBandPromptCarriesTheClosedDomainSet; the band half is a second spelling with no pin
  (ARCH-DRY). Practical risk is low — CEFR is a fixed six-point scale.

## Round 5 — 2026-09-04T16:37:31-07:00 (claude) — BLOCKED

### Raised

- **BR-16** [Critical] `flag-silently-ignored` --limit does not bound the authoring pass's model calls
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
- **BR-17** [Critical] `property-without-a-pin` Three headline M2 properties have pins that cannot fail; the sweep row is ticked for M1 only
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
- **BR-18** [Important] `task-without-live-conformance` authorTask, entailTask and vetoTask ship with no live conformance row
  The plan's Test surface (line 94) and its Integration-points block both
  commit to "a live conformance row each". harvest_conformance_test.go holds
  only M1's two band rows. Nothing detects drift between llmtest.Fake's
  canned JSON and the real service for the three schemas this milestone's
  whole value rests on (ARCH-MOCK). The veto's obsequious/sycophantic pair
  is already known to work live from the checkpoint, so it is a cheap row.
- **BR-19** [Important] `inert-mechanism` learnerFacts.Domains is computed at zero production call sites
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
- **BR-20** [Important] `language-scope-not-threaded` The entailment and veto prompts carry no store.Lang
  renderEntailPrompt (harvest_judge.go:85) and renderVetoPrompt (:146) take
  no language, while renderBandPrompt and renderAuthorPrompt both thread it
  and bandSystem's comment explains why. TestHarvestSendsTheDecksLanguage
  asserts only reqs[0]. With #18 open and d.lang="es" already exercised, a
  Spanish stem is judged by an unlabelled prompt whose worked example is
  English. This is the 2nd finding in family language-scope-not-threaded.
  The rule: every request-rendering function on a per-language path takes
  store.Lang and states it, and the wire test asserts it over every request
  the run sends rather than over reqs[0]. Sweep all four renderers at once.
- **BR-21** [Important] `doc-predeclares-outcome` The project row records actual and closed before this gate ran
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
- **BR-22** [Important] `store-contract-unheld-by-suite` The item cap is asserted against Mem only, not in storetest/suite.go
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
- **BR-23** [Minor] `behaviour-change-undocumented` Items() now returns newest-first and the interface contract does not say so
  sanitiseItems calls prune, which sorts on both the read and the write, so
  items no longer come back in insertion order. atlas/define.md records it;
  Store.Items' doc comment at store/store.go:60 — where #12 will read the
  contract — does not. This is the 2nd finding in family behaviour-change-
  undocumented. The rule: a behaviour promised to a consumer belongs on the
  interface it is promised through, not only in the atlas.
- **BR-24** [Minor] `two-spellings-of-one-predicate` stemUsesTheWord and blankOut locate the word by different rules
  stemUsesTheWord (harvest_item.go:418) checks only the leading boundary, so
  `set` is satisfied by "The settlement was reached"; blankOut
  (harvest_judge.go:173) then renders "The ___tlement was reached" into the
  veto prompt. Checking only the first occurrence also falsely rejects a stem
  where the word appears as a substring before appearing properly.
- **BR-25** [Minor] `helper-copied-not-shared` harvestPRNG duplicates play.prng step-for-step
  harvest_item.go:378-397 restates play/pick.go:168-215. The stated reason
  (exporting an internal, or dragging store vocabulary across the seam) is
  weaker than it reads — play.SampleStrings is already exported to main for
  exactly this, and an int-slice shuffle needs no store types. The seeding
  also differs, so "same algorithm" produces different sequences for the same
  seed; the comment overstates the relationship.
- **BR-26** [Minor] `measurement-off-the-production-path` The tier report counts items that were never written
  widened[tier]++ at harvest.go:262 runs before the veto, so an item whose
  candidates are all vetoed still contributes to "N item(s) drew options from
  X". This is the 2nd finding in family measurement-off-the-production-path.
  The rule: a batch statistic is taken over what the batch shipped, not over
  what it attempted — topicSpread already gets this right at harvest.go:298
  by appending only on success.
- **BR-27** [Minor] `inconsistent-failure-reporting` Only the author-call failure reports what was saved before stopping
  harvest.go:229 prints "authored N item(s) before stopping; they are saved";
  the entail (:241) and veto (:270) failures return 1 with no such line, so
  an outage during judging leaves the operator without the count the author
  path gives them.
- **BR-28** [Minor] `doc-understates-surface` README describes two checks where four conditions reject
  cmd/define/README.md says "Two things are checked" but an item is dropped
  by the free stem check, by entails, by glosses, by named, or by a total
  veto. "The sentence must give the word away" is also close to the unique-
  recoverability framing the checkpoint explicitly retired.
- **BR-29** [Minor] `test-only-symbol-in-production-api` PruneForTest is exported production API, and prune's truncation is unreachable
  store/item.go:200 exports a symbol for tests only. SetItems has one
  non-test caller (harvest.go:290), always with exactly one item and only
  for words with zero items, so ItemCap's truncation branch cannot be reached
  in production today. That is defensible as a guard for #13 — but say so,
  and prefer an in-package export_test.go alias over a permanent exported
  symbol.

## Round 6 — 2026-09-06T10:14:16-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 7 — 2026-09-06T13:28:28-07:00 (claude) — BLOCKED

### Disposed

- BR-16 — addressed — Measured: 8-word pre-banded deck, -limit 2, judge rejecting all = 2 calls total (was 16); a per-pass budget reddens both limit tests. Residual one-call overrun raised as a new finding.
- BR-17 — addressed — All three named pins fixed and mutation-confirmed; harvestRig now refuses size 1. Five of the six named greens redden on revert; sortedBanded is the exception, folded into the new pin finding.
- BR-18 — addressed — Three live rows exist for authorTask, entailTask and vetoTask, and the author row records a real live observation, so they ran.
- BR-19 — addressed — learnerFacts.Domains is now read by learnerFacts.reads through tierLearnerDomain; every learnerFacts field and both pickDistractors return values have a non-test reader.
- BR-20 — addressed — Verified by revert: hardcoding store.DefaultLang in entailTask and vetoTask reddens TestHarvestSendsTheDecksLanguage, which now iterates every request.
- BR-21 — addressed — The project row reads "actual: pending — written by the close gate, after the verdict".
- BR-22 — addressed — The cap row landed in storetest/suite.go. The newest-first member of its own enumeration is still missing and is raised separately.
- BR-23 — addressed — Store.Items' doc comment now states the newest-first order; the suite row that would hold it is raised separately.
- BR-24 — not-addressed — The two spellings were unified into wordIndexIn, but the looser rule was chosen for both, so the defect the finding measured survives.
- BR-25 — addressed — harvestPRNG is gone; play.ShuffleInts is exported and used, with one seeding step.
- BR-26 — not-addressed — The move is in the code but has no pin: restoring widened[tier]++ to its pre-veto position leaves the whole suite green.
- BR-27 — not-addressed — The lines were added but no test asserts them: grep finds no assertion on "before stopping", and deleting both lines leaves the suite green.
- BR-28 — addressed — The README now describes four checks and drops the unique-recoverability framing.
- BR-29 — addressed — PruneForTest moved to store/export_test.go, and prune's comment states the truncation branch is a guard for #13.

### Raised

- **BR-30** [Important] `flag-silently-ignored` -limit N still makes N+1 calls, and the two tests named for it never reach the authoring pass
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
- **BR-31** [Important] `behaviour-change-undocumented` main.go:628 still states the old -limit meaning; the sweep missed the enumeration's fifth member
  3rd in family. The rule is already written: when a fix changes what a user-facing
  contract MEANS, every statement of that meaning is part of the change — flag help,
  in-code guard comments, README, atlas, the constant's doc comment. Commit 94083a2
  says it "swept all three places" and updated the flag help, README and atlas; the
  mode-guard comment at main.go:628 still reads "-limit bounds how many words are
  ASKED ABOUT". The enumeration existed in the finding and was used as a list of
  noticed sites again.
- **BR-32** [Important] `doc-understates-surface` The atlas lists four selection tiers where the code has five, and the plan says three tasks where M2 ships four
  3rd in family; round 6's I5 is unfixed at HEAD. The rule: a doc that ENUMERATES a
  code-side set is a consumer of that set — when the set changes the enumeration is
  re-derived, not left at its previous count. tierLearnerDomain is second-priority
  and changes which words are selected; grep finds it in no atlas or README text.
  atlas/define.md:1578 still records as an open question for #12 the exact thing
  harvest_item.go:262-276 says the tier answers. The plan's Integration-points table
  omits entailTask and its Test-surface sentence says "the three tasks" where there
  are four. Sweep: the atlas tier list, the atlas open-question paragraph, the plan
  table, the plan's "three tasks" sentence, and the README's tier-report lines.
- **BR-33** [Important] `store-contract-unheld-by-suite` Store.Items' newest-first promise, added this window, has no storetest row
  3rd in family. The rule stands from BR-22: any promise stated on the Store
  interface is asserted in storetest/suite.go, never in a per-implementation test.
  BR-22's cap row landed and is mutation-confirmed, but its enumeration named three
  members; the newest-first read order is asserted only through the pure
  PruneForTest tests. The suite's "a word may hold several items" row writes day(1)
  and day(2) and checks only that the Form discriminator survived. Read-side
  canonicalisation, the third member, is correctly out of scope at yaml.go:660.
- **BR-34** [Important] `property-without-a-pin` Three fixes from the last two rounds have no test that fails without them
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
- **BR-35** [Important] `plan-element-ticked-unbuilt` #12's three moved Done-when rows are still unwritten, and its Revision names this milestone as the trigger
  2nd in family. The rule: a deferral whose trigger is "when Mx lands" is swept at
  Mx's boundary by the issue that wrote it, not left for the consuming issue to
  discover. Task 5 Step 0 is ticked and its Revision on #12 is real, but it closes
  with "Not yet applied to the rows above. They are rewritten when #10 M2 lands."
  M2 is landing, and #12's Done-when still owns "Options are drawn from the pool +
  deck, never model-generated", the sycophantic/obsequious row, and "The form works
  with the LLM seam unavailable".
- **BR-36** [Minor] `doc-comment-mangled-by-edit` prune's doc comment carries an orphaned fragment, and harvestRig stacks two doc comments
  store/item.go:200 reads "// : the same input prunes to the / // same output, every
  time." — the DETERMINISTIC subject was lost in an edit. harvest_test.go:19-27
  stacks two doc comments for harvestRig, the second beginning mid-block. Round 6
  filed the first; both are unfixed at HEAD.
- **BR-37** [Minor] `measurement-off-the-production-path` The M2 sweep block says 22 properties over a 24-row list and records neither the mutation nor the named test
  3rd in family. The rule the block itself states two paragraphs below is "a sweep
  row is only as good as the mutation behind it — recording the verdict without
  recording the mutation is how a green row reads as red". M1's table at least
  carried a verdict column; M2's is a prose list of property names. I re-derived
  seven rows by reverting and they hold, which is the problem: a hand-written
  table's one false row is indistinguishable from its true ones. Also unticked in
  the same section: "The generated batch, read by the operator", while the Plan's
  checkpoint row is ticked and the Log records three batches read.
- **BR-38** [Minor] `inert-mechanism` widened[tierSameDomain] is incremented and never read
  4th in family. harvest.go:262 counts every tier; the report loop at :365 walks
  only tierLearnerDomain, tierGeneral, tierAnyDomain and tierAboveBand. The rule
  already written for this family: a value no production path reads is wired or
  deleted. Also in this class: TestTheLimitHoldsWhenEveryStemIsRejected's distinct
  fixture is dead under the current rig (see the -limit finding).
- **BR-39** [Minor] `helper-copied-not-shared` optionsPerItem restates play.maxOptions - 1 across a seam that already exports two helpers
  2nd in family. harvest.go:46 hardcodes 3 with a comment naming play.maxOptions as
  the reason; play already exports SampleStrings and ShuffleInts to main for exactly
  this, and an option count needs none of the store's vocabulary.
- **BR-40** [Minor] `inconsistent-failure-reporting` One exhausted budget prints two stopped lines, and prune shadows the builtin cap
  2nd in family. Measured: -limit 5 on a fresh 8-word deck prints "stopped at the
  --limit of 5 model call(s)" then "stopped authoring at the --limit of 5 model
  call(s)" for one budget. Also cosmetic: store/item.go:207 declares
  func prune(items []Item, cap int), shadowing the builtin, and
  internal/llm/golden_schema_test.go:15 leaves one comment line past the file's wrap.

## Round 8 — 2026-09-06T13:53:26-07:00 (claude) — passed

### Disposed

- BR-24 — not-addressed — Unification is real (both go through wordIndexIn) but the loose rule survives: stemUsesTheWord("The settlement was reached in Albany.","set")=true, blankOut gives "The ___tlement was reached in Albany.".
- BR-26 — addressed — widened[tier]++ now runs after SetItems; the counter no longer counts attempts. The missing revert-check stays open under BR-34.
- BR-27 — addressed — Mutation-confirmed: deleting the survivor lines on the entail and veto paths reddens TestEveryAuthoringOutagePathReportsSurvivors on both subtests.
- BR-28 — addressed
- BR-30 — addressed — runWithin gates structurally; reverting to charge-then-call gives -limit 6 = 10 calls and reddens two cells of the new flag x pass table. Each cell now fatals if its pass was never entered.
- BR-31 — addressed — main.go:628 now names MODEL CALLS. The class recurs at harvest.go:15 and plan:90/:285 - raised separately.
- BR-32 — addressed — Atlas lists five tiers with tierLearnerDomain second, the open-question paragraph is rewritten, the plan table and "four tasks" are corrected, README gained the tier lines. The new README/atlas claim overstates - raised separately.
- BR-33 — addressed — Mutation-confirmed: flipping sortItems' After to Before reddens the new storetest row on both TestMemConformance and TestYAMLConformance.
- BR-34 — not-addressed — Two of three confirmed red on revert (survivor lines, sortedBanded). The tier-report member is green on revert - its fixture sits in tierSameDomain, which the report loop never prints.
- BR-35 — addressed — 000012's three rows are rewritten as satisfied-by-construction with the trigger recorded.
- BR-36 — not-addressed — Both sites unchanged at HEAD: store/item.go:200 still carries the orphaned fragment, harvest_test.go:18-27 still stacks two harvestRig doc comments.
- BR-37 — not-addressed — plan:456 still says 22 properties over a 24-item list, records no mutation or test name per row, and plan:492 is still unticked.
- BR-38 — not-addressed — The dead-fixture half is fixed (the test now preBands and fatals on zero entail calls). widened[tierSameDomain] is still incremented at harvest.go:390 and still absent from the report loop at :394.
- BR-39 — not-addressed — optionsPerItem = 3 is unchanged and play.maxOptions is still unexported.
- BR-40 — not-addressed — golden_schema_test.go's wrap is fixed. The double stopped line is measured unchanged, and prune's cap -> max rename shadows a different Go builtin (module is go 1.26).

### Raised

- **BR-41** [Important] `behaviour-change-undocumented` harvestLimit's own doc comment and two plan lines still state -limit's superseded meaning
  4th in family. harvest.go:15 reads "bounds the words one --harvest run will ask the model
  about" - the member BR-31's enumeration named by name - fourteen lines above budget's
  comment saying the opposite; plan:90 and plan:285 repeat it. Correct at HEAD: main.go:432,
  main.go:628, README:374, atlas:1431. The rule has been written twice and used as a list of
  noticed sites twice, so the deliverable is the mechanism, not the three lines: a retired-
  PHRASING registry checked by a repo guard, the same shape as
  TestProseDoesNotSpellStaleRuntimeArtifactNames and TestNoArtifactNamesARetiredSymbol.
- **BR-42** [Important] `doc-predeclares-outcome` This round's doc sweep promises the tier is printed for every item; the code never prints tierSameDomain
  4th in family. New in c5b3cda: README:412-416 "Whatever it settled for, it says so" and
  atlas:1568-1571 "the tier reached is printed for every item". harvest.go:394 walks only
  tierLearnerDomain, tierGeneral, tierAnyDomain and tierAboveBand. Measured: a 4-word
  pre-banded rig authors 4 items at tierSameDomain and prints no "drew options from" line.
  The rule: a doc sentence describing a code behaviour is a consumer of that behaviour and is
  written from the code at HEAD, not from the design - a sweep that fixes a doc against the
  plan while a known code gap (BR-38) is open converts an open finding into a false promise.
- **BR-43** [Important] `property-without-a-pin` TestTheTierReportCountsOnlyWrittenItems passes with its fix reverted, so BR-34's third member is undisposed
  6th in family, and it is the pattern the family exists to catch: the commit that closed
  "three fixes with no failing test" shipped a fourth. Reverted on a scratch copy at HEAD,
  moving widened[tier]++ back above the veto loop leaves the test green, because preBand puts
  every word in DomainGeneral so the tier is tierSameDomain, which the report loop never
  prints. Adding tierSameDomain to the loop makes the mutated build print "4 item(s) drew
  options from same domain, at band" and the test reddens. The rule: a revert-check is only a
  check if the author ran it - write the mutation, run it, record that it reddened, in the same
  commit as the fix. Fixing the report loop fixes this pin as a side effect; do both and re-run.
- **BR-44** [Minor] `inert-mechanism` runWithin's stated invariant is false and two of its four errBudget branches are unreachable
  5th in family. harvest.go:88 says runWithin "is the ONLY way this file reaches a model";
  runHarvestAgreement calls llm.Run directly at :462. Not a bug - that mode has its own K x N
  bound and -limit is refused beside -agreement - but it is the load-bearing sentence of
  BR-30's disposition. Same site, same rule: the errors.Is(err, errBudget) arms after bandTask
  (:161) and after authorTask (:277) cannot fire, because each loop's bud.spent() early exit
  ran with nothing decrementing in between. Panic-probed: the whole cmd/define suite passes
  without entering either. The rule is already written - a branch or value no production path
  reaches is wired or deleted - so the deliverable is the enumeration: a per-block coverage
  assertion over harvest*.go, which would also have caught BR-38's counter.

## Round 9 — 2026-09-06T14:09:11-07:00 (claude) — BLOCKED

### Disposed

- BR-11 — addressed — harvest_conformance_test.go:65-74 derives gloss+known via testDict + senseFacts; all 8 bandingWords verified present in testdata/entries/en.
- BR-12 — addressed — atlas/define.md:1680-1686 states the set check and names it a breaking CLI change; README.md:433-436 carries it. The exit-code table gap is raised separately.
- BR-13 — addressed — All seven members pinned through run() at harvest_test.go:696-720, plus TestRunHarvestThroughTheWiringHop for the withStore hop and the bare -agreement default.
- BR-14 — not-addressed — Unswept, and M2 landing INVERTED it: README.md:459-461 now reads "Nothing writes this yet - authoring is the next milestone" about a surface --harvest shipped. The atlas is correct at 1419.
- BR-15 — addressed — renderBandPrompt enumerates store.Bands() at harvest_band.go:68. No symmetric pin though - reverting to a hardcoded scale leaves the golden green, unlike the domain half.
- BR-24 — not-addressed — Measured at HEAD on real deck words - light/"The ___house at Portland Head", set/"The ___tlement", run/"The ___way", bank/"___ruptcy" all pass stemUsesTheWord. That is 4 of the checkpoint deck's A1 words shipping an unusable item; more than cosmetic.
- BR-34 — addressed — Third member verified by revert - moving widened[tier]++ above the veto loop reddens TestTheTierReportCountsOnlyWrittenItems. All three authoring outage branches covered by TestEveryAuthoringOutagePathReportsSurvivors.
- BR-36 — not-addressed — Both unchanged. store/item.go:200 still reads "// : the same input prunes to the"; harvest_test.go:19-27 still stacks two harvestRig doc comments.
- BR-37 — not-addressed — plan:456 still says 22 over a list I counted at 24; no mutation or test name per row; plan:492 still unticked while the issue's row is ticked.
- BR-38 — addressed — harvest.go:408 now walks tierSameDomain first; verified by revert - dropping it reddens TestTheTierReportCoversEveryWrittenItem at 4 authored / 0 reported.
- BR-39 — not-addressed — optionsPerItem = 3 unchanged at harvest.go:54; play.maxOptions still unexported although this window exported ShuffleInts across the same seam.
- BR-40 — not-addressed — Measured at HEAD - one exhausted budget prints both "stopped at the --limit of 5" and "stopped authoring at the --limit of 5". prune's parameter is now named max, shadowing the Go builtin.
- BR-41 — not-addressed — The three named lines are correct at HEAD, but the mechanism the finding named as the deliverable did not land - repo_guard_test.go has ZERO diff across the whole window, and retiredPhrases (:1131) is the registry it asked for. Fourth hand sweep of the same class.
- BR-42 — addressed — Verified by revert on a scratch copy - removing tierSameDomain from the report loop reddens TestTheTierReportCoversEveryWrittenItem, so the doc sentence is now written from the code.
- BR-43 — addressed — Verified by revert - the mutation the finding prescribed reddens the pin, and TestTheTierReportCoversEveryWrittenItem reddens on the report-loop mutation. Both checks re-run here rather than read.
- BR-44 — not-addressed — harvest.go:82 still says runWithin is the ONLY path to a model while runHarvestAgreement calls llm.Run at :475; the errBudget arms at :166 and :281 are still unreachable, since each loop's bud.spent() exit runs with nothing decrementing in between.

### Raised

- **BR-45** [Important] `store-contract-unheld-by-suite` Forget leaves facts/ and items/ behind, so a forgotten word's bad material is unregenerable
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
- **BR-46** [Minor] `doc-understates-surface` README's exit-code table declares itself an enumeration and omits every code this window added
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

## Round 10 — 2026-09-06T14:23:05-07:00 (claude) — BLOCKED

### Disposed

- BR-45 — addressed — Revert-verified in a scratch worktree: the storetest row reddens against BOTH Mem and YAML,
and dropping facts/items from perWordDirs reddens TestPerWordDirsCoverEveryRuntimeDir.
Residual: Mem.Forget hand-lists its maps and the guard covers YAML only.
- BR-41 — not-addressed — repo_guard_test.go has ZERO diff across the window; retiredPhrases (:1131) over
currentTruthFiles (:1623, binds non-test .go + README + atlas + plans) is exactly the
mechanism asked for and no row was added. The three named lines are correct; the class is not.
- BR-24 — not-addressed — Re-measured at HEAD by probe - "The settlement"/set, "The lighthouse"/light, "The runway"/run,
"Bankruptcy"/bank all pass stemUsesTheWord and blankOut renders "The ___tlement". The fix is not
a bare trailing-boundary check: wordIndexIn deliberately allows inflections, so it needs a
bounded suffix set (s/es/ed/ing/'s).
- BR-14 — not-addressed — Worse than round 9 recorded: README.md:459-461 still says "Nothing writes this yet - authoring
is the next milestone" about a surface this issue shipped. At issue close that is a false
statement in user-facing docs, not a forward-looking one.
- BR-36 — not-addressed — Both sites unchanged (store/item.go:200 "// : the same input prunes to the"; harvest_test.go:19-27).
A third member measured this round - harvest_item.go:318 documents "tierAnyBand", a constant that
never existed (introduced in fd0767b as prose only); unexported names are exempt from the symbol guard.
- BR-37 — not-addressed — plan:456 still claims 22 over a list I counted at 24; no mutation or test name per row;
plan:492 "The generated batch, read by the operator" still unticked while the issue's row is ticked.
- BR-39 — not-addressed — harvest.go:54 still hardcodes optionsPerItem = 3 with a comment naming play.maxOptions as the
reason, across a seam this window already widened twice (SampleStrings, ShuffleInts).
- BR-40 — not-addressed — Both halves stand at HEAD - the banding loop's exit (harvest.go:148) and runAuthoring's
(harvest.go:264) each print for one exhausted budget; store/item.go:212 is func prune(items []Item, max int),
shadowing the Go builtin max.
- BR-44 — not-addressed — harvest.go:81 still says runWithin is the ONLY way this file reaches a model while
runHarvestAgreement calls llm.Run at :475; the errBudget arms at :166 and :281 remain unreachable
because each loop's bud.spent() exit runs with nothing decrementing before the call.
- BR-46 — not-addressed — README.md:589-593 unchanged. The table still declares itself an enumeration and omits mode
collision, every -limit/-agreement usage error, and all of --harvest's exit-1 paths.

### Raised

- **BR-47** [Minor] `inconsistent-failure-reporting` Forget deletes the deck entry first, so a partial failure reports "nothing removed" for a word it removed
  This is the 3rd finding in family inconsistent-failure-reporting. Do NOT fix the loop order alone.
  The rule: a multi-step mutation orders its effects so the value it returns is true of what happened,
  and the enumeration - Forget's four removals, the banding loop's "banded N before stopping", the three
  authoring outage branches - is walked by a test that injects a failure at each step. Only the authoring
  branches have that today (TestEveryAuthoringOutagePathReportsSurvivors). Reproduced at HEAD: with
  facts/en/sycophantic.yaml made a non-empty directory, YAML.Forget returns (false, ENOTEMPTY) while
  words/en/sycophantic.yaml is already gone - so --forget exits 1 on a word it removed, and play_loop.go:455
  never calls held.dropped. os.Remove over a non-empty directory is a sufficient seam for the test.
  ARCH-ORDER: the error path unwinds the sequencing and drops the in-flight effect.
- **BR-48** [Minor] `language-scope-not-threaded` perWordDirs mixes the language-scoped dirs with flat usage/, so forgetting a word in one language clears another's news cache
  This is the 3rd finding in family language-scope-not-threaded. Do NOT fix the usage/ row.
  The rule: a per-word verb is scoped the same way the surface it touches is scoped, and
  TestPerWordDirsCoverEveryRuntimeDir - which exists precisely to classify every runtime directory -
  classifies on ONE axis (per-word vs history) while da5c395 crosses a second (scoped vs flat).
  yaml.go:695 lists wordsDir/usageDir/factsDir/itemsDir; usageDir is RuntimeDirs[2] with no lang segment
  (yaml.go:161), so `define --forget red` in an es directory removes usage/red.yaml that the en deck
  populated. Consequence is a refetch, which is why this is Minor; the deliverable is the second axis
  in the guard, so the next surface added is classified on both.

## Round 11 — 2026-09-06T14:35:03-07:00 (claude) — BLOCKED

**Forced past** (`--force`): --no-ledger (or --force): ALL SEVEN Done-when rows ticked with the mutation that proved them. go test ./... green; go vet clean under BOTH tag sets; gofmt clean. (1) --harvest produces finished items with no sitting running — the seam is made to PANIC, not nil, and the sitting is asserted not to have banded anything on the way past. (2) Every word carries a band and domain assigned once, pinned on the request COUNT so a second run is asserted to make zero calls. (3) The banding is MEASURED: --harvest -agreement=N, its own mode writing nothing; floor 0.8 asserted LIVE at mean agreement 1.00 over 8 words x 5 assignments, on the prompt production actually sends. (4) A distractor is never the answer: obsequious/sycophantic committed as the known-bad case, fired on real material in all three checkpoint batches in both directions, and asserted live in both directions. (5) Authored stems entail and name real subjects — three separate verdict fields, three committed known-bad stems, topicSpread measured with NO model. (6) A model outage leaves the store usable, pinned on all four outage paths with survivors asserted WHOLE. (7) Growth bounded by ItemCap at the write, held by storetest against both implementations; prune proved deterministic by pruning twice with the input SHUFFLED between calls. THE CHECKPOINT RAN — three live batches on a real deck, read, changing the design twice (appositive glosses 10/20 to 0, authored 20 to 19/20). M1 and M2 each have their own mutation sweep (13 and 22 properties) plus per-finding revert-checks. BYPASSING THE LEDGER GATE FOR EXACTLY ONE FINDING, BR-41, which is verifiably fixed at HEAD and has been since round 8s remediation: it names harvestLimits doc comment (harvest.go:15, now "the default bound on MODEL CALLS one --harvest run may make") and two plan lines (plan:90 and plan:285, both now "caps the MODEL CALLS"). A grep across cmd/, atlas/ and the plan for every statement of -limits meaning returns 20 hits and every one says calls; zero say words. The ledger carried the entry forward without disposing it across rounds 8, 9 and 10 while the tree was already correct. No other finding is bypassed — the other blocker from close round 1 (BR-45, Forget leaving facts/ and items/ behind) was a real shipped bug, fixed as a class in da5c395 with a guard that fails when a new runtime directory is unclassified, and its second axis fixed in dc1130e.

### Disposed

- BR-48 — addressed — Mutation-verified in a scratch worktree — flipping usage/ to scoped:true reddens TestPerWordDirsCoverEveryRuntimeDir with the intended message; the second axis is declared at yaml.go:707 and asserted at yaml_test.go:718.
- BR-14 — not-addressed — README.md:459-461 still reads "Nothing writes this yet - authoring is the next milestone" about the items/ surface M2 shipped; at the CLOSE boundary that is a false statement in user docs, not a forward-looking one.
- BR-24 — not-addressed — Unchanged at HEAD - wordIndexIn (harvest_item.go:481) checks only the leading boundary and allows any suffix, so `set` is satisfied by "The settlement" and blankOut renders "The ___tlement".
- BR-36 — not-addressed — All three sites unchanged - store/item.go:200 "// : the same input prunes to the", harvest_test.go:19-27 two stacked doc comments, harvest_item.go:318 naming tierAnyBand.
- BR-37 — not-addressed — plan:456 still says 22 over a list I re-counted at 24; no mutation or test name per row; plan:492 still unticked while the issue's row is ticked.
- BR-39 — not-addressed — harvest.go:53 still hardcodes optionsPerItem = 3 with a comment naming play.maxOptions as the reason.
- BR-40 — not-addressed — Both halves stand - harvest.go:149/167 and :266/282 each print for one exhausted budget, and store/item.go:212 is func prune(items []Item, max int), shadowing the Go builtin.
- BR-41 — not-addressed — repo_guard_test.go has ZERO diff across the whole window; retiredPhrases (:1131) is the registry the finding named and no row was added. The three named lines are correct at HEAD; the class is still unmechanised.
- BR-44 — not-addressed — harvest.go:82 still says runWithin is the ONLY way this file reaches a model while runHarvestAgreement calls llm.Run at :475; both errBudget arms remain unreachable because each loop's bud.spent() exit runs with nothing decrementing before the call.
- BR-46 — not-addressed — README.md:585-593 unchanged - the table still declares itself an enumeration and omits mode collision, every -limit/-agreement usage error, and all of --harvest's exit-1 paths.
- BR-47 — not-addressed — yaml.go:743-757 unchanged - the loop removes words/ first and every non-ENOENT error returns (false, err), so a partial failure reports "nothing removed" for a word it removed.

### Raised

- **BR-49** [Minor] `doc-understates-surface` The atlas describes Forget's guard as one-axis, one commit after dc1130e made it two, and the cross-language forget it recorded reaches no user-facing doc
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

## Open findings

- **BR-14** [Minor] `doc-predeclares-outcome` README's directory listing promises items/<lang>/*.yaml, which M1 never writes
- **BR-24** [Minor] `two-spellings-of-one-predicate` stemUsesTheWord and blankOut locate the word by different rules
- **BR-36** [Minor] `doc-comment-mangled-by-edit` prune's doc comment carries an orphaned fragment, and harvestRig stacks two doc comments
- **BR-37** [Minor] `measurement-off-the-production-path` The M2 sweep block says 22 properties over a 24-row list and records neither the mutation nor the named test
- **BR-39** [Minor] `helper-copied-not-shared` optionsPerItem restates play.maxOptions - 1 across a seam that already exports two helpers
- **BR-40** [Minor] `inconsistent-failure-reporting` One exhausted budget prints two stopped lines, and prune shadows the builtin cap
- **BR-41** [Important] `behaviour-change-undocumented` harvestLimit's own doc comment and two plan lines still state -limit's superseded meaning
- **BR-44** [Minor] `inert-mechanism` runWithin's stated invariant is false and two of its four errBudget branches are unreachable
- **BR-46** [Minor] `doc-understates-surface` README's exit-code table declares itself an enumeration and omits every code this window added
- **BR-47** [Minor] `inconsistent-failure-reporting` Forget deletes the deck entry first, so a partial failure reports "nothing removed" for a word it removed
- **BR-49** [Minor] `doc-understates-surface` The atlas describes Forget's guard as one-axis, one commit after dc1130e made it two, and the cross-language forget it recorded reaches no user-facing doc

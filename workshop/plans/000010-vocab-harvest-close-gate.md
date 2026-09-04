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

## Open findings

- **BR-11** [Important] `measurement-off-the-production-path` the conformance floor and the reported 1.00 measure a prompt shape --harvest never sends
- **BR-12** [Important] `behaviour-change-undocumented` mode exclusivity changes previously-accepted invocations and appears in no user-facing doc
- **BR-13** [Important] `property-without-a-pin` six of the seven run()-path members round 3 enumerated are still unpinned
- **BR-14** [Minor] `doc-predeclares-outcome` README's directory listing promises items/<lang>/*.yaml, which M1 never writes
- **BR-15** [Minor] `inert-mechanism` store.Bands() has no production caller while the band prompt hand-restates the six levels

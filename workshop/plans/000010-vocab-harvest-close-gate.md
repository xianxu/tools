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

## Open findings

- **BR-1** [Important] `parse-result-not-canonicalised` agreement counts raw band strings, so "c1" and "C1" score as disagreement
- **BR-2** [Important] `store-contract-unheld-by-suite` Mem and YAML disagree about a damaged WordFacts record, and storetest has no row
- **BR-3** [Important] `language-scope-not-threaded` bandTask hardcodes "English vocabulary" while facts are stored per-language
- **BR-4** [Important] `plan-element-ticked-unbuilt` sanitiseFacts/sanitiseItem do not exist although the plan step naming them is ticked
- **BR-5** [Minor] `inert-mechanism` the sort in agreement cannot affect the result, and agreementRounds is unused
- **BR-6** [Minor] `property-without-a-pin` the computed longest-first label ordering has no test
- **BR-7** [Minor] `stdlib-reimplemented` hand-rolled contains in harvest_band_test.go duplicates strings.Contains
- **BR-8** [Minor] `loop-invariant-work` runHarvestAgreement re-runs wordSense inside the per-round loop
- **BR-9** [Minor] `flag-silently-ignored` -limit is ignored in -agreement mode, N is unbounded, and --play --harvest drops a mode
- **BR-10** [Minor] `doc-predeclares-outcome` the project's M1 calibration prose was written before this gate ran

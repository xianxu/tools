---
gate: plan-quality
issue: 35
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-29T15:06:12-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Mask-then-search over the whole ORIGIN text infers from cognate clauses and from "Germanic"
          detail: |-
            The rule is unscoped on two axes and the repo's own corpus falsifies it.
            thing.txt ("Old English ...; related to German Ding") infers de; bargainer.txt
            ("from Old French ...; related to German borgen") infers de; bank.txt infers fr
            off a clause about another sense; and substring matching makes "of Germanic
            origin" match German, which 8 of 34 en fixtures carry, so run.txt infers de
            with no cognate clause at all. Decide word-boundary matching and which clause
            of ORIGIN may supply a match, record it as a decision, and add table rows for
            thing, bargainer, run and bank. The Spec's "31 = historical stages only" bucket
            measured a different rule than the one designed.
          family: origin-match-scope
          round: 1
        - id: PQ-2
          severity: Important
          title: pronHelp documents the -pron FLAG; the /pron command's argument rule is hand-written prose no test pins
          detail: |-
            atlas/define.md:760 ("/pron REQUIRES a language"), atlas/define.md:1145 and
            README.md:340-343 sit outside every marked span, go false with this change, and
            TestDocsQuoteThePronHelp stays green. Conversely, putting "the language is
            optional" into pronHelp makes the -pron flag's help false — it is an fs.String
            (main.go:412) that cannot be given bare, and main.go:513 treats "" as not-given.
            Name the real doc surfaces, or extend the derived-span mechanism to the command.
          family: doc-sweep-incomplete
          round: 1
        - id: PQ-3
          severity: Important
          title: Task 2 and Task 3 name fixtures that are not in the captured corpus
          detail: |-
            Neither croissant nor ballet is in cmd/define/testdata/entries/en/ (34 curated
            entries, captured via testdata/capture.sh outside a sandbox), so the ARCH-MOCK
            claim that the existing corpus is the fixture set and TestFixturesMatchLiveDictionary
            its live half does not hold for these tests. concrete.txt, jalapeño.txt,
            content.txt and ephemeral.txt already cover determinate-fr, determinate-es,
            historical-only and bare-Greek; say how the two-languages case gets its fixture.
          family: fixture-provenance
          round: 1
        - id: PQ-4
          severity: Minor
          title: newCommandCtx has three call sites and does not take a session
          detail: |-
            main.go:651 is a one-shot with no session, and both loops assign cc.replay after
            construction (repl.go:364, replraw.go:241). "newCommandCtx fills entry from the
            session" needs either a new parameter threaded through all three sites or the
            same post-construction assignment.
          family: unbacked-existing-behavior
          round: 1
        - id: PQ-5
          severity: Minor
          title: No non-goal stated for the -pron flag / one-shot path
          detail: |-
            D3 defers "modern Greek" explicitly, which is good. The -pron flag is the same
            feature on the one-shot side and cannot express inference at all; say so as a
            recorded non-goal rather than leaving it unmentioned.
          family: unstated-non-goal
          round: 1
      blocked: true
---

# Gate ledger — tools#35 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-29T15:06:12-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `origin-match-scope` Mask-then-search over the whole ORIGIN text infers from cognate clauses and from "Germanic"
  The rule is unscoped on two axes and the repo's own corpus falsifies it.
  thing.txt ("Old English ...; related to German Ding") infers de; bargainer.txt
  ("from Old French ...; related to German borgen") infers de; bank.txt infers fr
  off a clause about another sense; and substring matching makes "of Germanic
  origin" match German, which 8 of 34 en fixtures carry, so run.txt infers de
  with no cognate clause at all. Decide word-boundary matching and which clause
  of ORIGIN may supply a match, record it as a decision, and add table rows for
  thing, bargainer, run and bank. The Spec's "31 = historical stages only" bucket
  measured a different rule than the one designed.
- **PQ-2** [Important] `doc-sweep-incomplete` pronHelp documents the -pron FLAG; the /pron command's argument rule is hand-written prose no test pins
  atlas/define.md:760 ("/pron REQUIRES a language"), atlas/define.md:1145 and
  README.md:340-343 sit outside every marked span, go false with this change, and
  TestDocsQuoteThePronHelp stays green. Conversely, putting "the language is
  optional" into pronHelp makes the -pron flag's help false — it is an fs.String
  (main.go:412) that cannot be given bare, and main.go:513 treats "" as not-given.
  Name the real doc surfaces, or extend the derived-span mechanism to the command.
- **PQ-3** [Important] `fixture-provenance` Task 2 and Task 3 name fixtures that are not in the captured corpus
  Neither croissant nor ballet is in cmd/define/testdata/entries/en/ (34 curated
  entries, captured via testdata/capture.sh outside a sandbox), so the ARCH-MOCK
  claim that the existing corpus is the fixture set and TestFixturesMatchLiveDictionary
  its live half does not hold for these tests. concrete.txt, jalapeño.txt,
  content.txt and ephemeral.txt already cover determinate-fr, determinate-es,
  historical-only and bare-Greek; say how the two-languages case gets its fixture.
- **PQ-4** [Minor] `unbacked-existing-behavior` newCommandCtx has three call sites and does not take a session
  main.go:651 is a one-shot with no session, and both loops assign cc.replay after
  construction (repl.go:364, replraw.go:241). "newCommandCtx fills entry from the
  session" needs either a new parameter threaded through all three sites or the
  same post-construction assignment.
- **PQ-5** [Minor] `unstated-non-goal` No non-goal stated for the -pron flag / one-shot path
  D3 defers "modern Greek" explicitly, which is good. The -pron flag is the same
  feature on the one-shot side and cannot express inference at all; say so as a
  recorded non-goal rather than leaving it unmentioned.

## Open findings

- **PQ-1** [Critical] `origin-match-scope` Mask-then-search over the whole ORIGIN text infers from cognate clauses and from "Germanic"
- **PQ-2** [Important] `doc-sweep-incomplete` pronHelp documents the -pron FLAG; the /pron command's argument rule is hand-written prose no test pins
- **PQ-3** [Important] `fixture-provenance` Task 2 and Task 3 name fixtures that are not in the captured corpus
- **PQ-4** [Minor] `unbacked-existing-behavior` newCommandCtx has three call sites and does not take a session
- **PQ-5** [Minor] `unstated-non-goal` No non-goal stated for the -pron flag / one-shot path

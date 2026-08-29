---
gate: plan-quality
issue: 26
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-28T18:03:09-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Plan row 2 asserts ratio/glop/logarithmic are "no longer reachable" — they render clean, they are not unreachable
          detail: |-
            Their absence from the 26-survivor enumeration means they stopped rendering raw
            notation, not that the curated path cannot reach them. NOAD leads curated["en"]
            (cmd/define/dictselect.go:76-79) and ratio/logarithmic are ordinary NOAD headwords.
            Probe each word before editing, so the atlas correction does not install a second
            false claim in place of the first.
          family: unbacked-behavior-claim
          round: 1
        - id: PQ-2
          severity: Important
          title: The doc sweep names only the Limits entry; four more measured claims are now false
          detail: |-
            atlas/define.md:83 and :87 pin the width at 70,897 against the re-measured 70,886;
            atlas/define.md:88 says "The 530 non-Latin entries ... are counted and excluded" where
            the run now reports 0 non-Latin; atlas/define.md:80 says "the 29 captured fixtures"
            against 33 in testdata/entries/en alone; and cmd/define/live_property_test.go:65-69 —
            inside the file the plan already edits — still claims DCSCopyTextDefinition searches
            every ACTIVE dictionary with no API to select one, untrue since #23 M2. Produce the
            site list with a grep that ran (workshop/lessons.md:1962-1968), not by hand.
          family: doc-sweep-incomplete
          round: 1
        - id: PQ-3
          severity: Important
          title: Hand-editing 27 to 26 in the atlas is the fourth sweep of a number knownRawNotationEntries owns
          detail: |-
            workshop/lessons.md:2038-2056 states the rule: a doc restating a fact the code owns,
            having drifted twice, becomes a CONSUMER. This section has drifted three times
            (lessons.md:13-15). The working precedent is TestREADMEQuotesThePromptsTheLoopActuallyPrints
            at cmd/define/doc_sync_test.go:27. Either make the Limits line derive from the constant
            or state the deferral as an explicit non-goal with its reason (ARCH-DRY, ARCH-PURPOSE).
          family: restatement-not-consumer
          round: 1
        - id: PQ-4
          severity: Important
          title: Done-when row 3 wants the next drift attributable; the plan delivers a comment the test cannot enforce
          detail: |-
            cmd/define/live_property_test.go:90 caps diagnostics at rawPipes <= 3, which is why the
            Log records "A full list needs the report cap raised for one run" — a cost this issue
            already paid once. A future 27 still prints one number and three samples, so the manual
            enumeration recurs. Classify each survivor by cause and log per-cause counts, so the run
            itself names which group moved.
          family: attribution-not-mechanized
          round: 1
        - id: PQ-5
          severity: Important
          title: No test surface named, and the corpus hard-zero assertion collides with the new taxonomy
          detail: |-
            TestNoRawPronunciationNotationSurvives (cmd/define/render_test.go:230-247) asserts ZERO raw
            pipes over the committed corpus, passing only because none of the 26 entries is in
            testdata/entries/en. The repo's precedent captures the exemplar — bases.txt and parrot.txt
            exist for the shapes atlas Limits names. Either capture one fixture per newly-named cause
            and say how the hard-zero test accommodates them, or state as an explicit non-goal that
            the taxonomy stays live-only, with the reason (ARCH-MOCK). The plan states no non-goals.
          family: taxonomy-live-only
          round: 1
        - id: PQ-6
          severity: Important
          title: The triage rests on a negative probe of "define brent" with no positive control stated
          detail: |-
            Building ./cmd/define here yields "no dictionary entry" for brent, charge and ratio alike,
            under "no known en dictionary is installed; searching every active dictionary" — a context
            where every lookup fails and the curated path is never exercised. Done-when row 1's
            disposition needs the control named: same session, charge returns an entry, no
            "no known en dictionary" warning, checked around 70,886. Plan row 4 re-runs the sweep
            anyway, so this is free.
          family: negative-probe-no-control
          round: 1
        - id: PQ-7
          severity: Minor
          title: 'deps: [tools#27] stays in the frontmatter while the Spec declares it moot'
          detail: |-
            The Spec says the dependency existed only to decide how to render a shape that no longer
            arrives, but the frontmatter still carries the blocking edge. No Plan row removes it.
          family: stale-frontmatter
          round: 1
      blocked: true
---

# Gate ledger — tools#26 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-28T18:03:09-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `unbacked-behavior-claim` Plan row 2 asserts ratio/glop/logarithmic are "no longer reachable" — they render clean, they are not unreachable
  Their absence from the 26-survivor enumeration means they stopped rendering raw
  notation, not that the curated path cannot reach them. NOAD leads curated["en"]
  (cmd/define/dictselect.go:76-79) and ratio/logarithmic are ordinary NOAD headwords.
  Probe each word before editing, so the atlas correction does not install a second
  false claim in place of the first.
- **PQ-2** [Important] `doc-sweep-incomplete` The doc sweep names only the Limits entry; four more measured claims are now false
  atlas/define.md:83 and :87 pin the width at 70,897 against the re-measured 70,886;
  atlas/define.md:88 says "The 530 non-Latin entries ... are counted and excluded" where
  the run now reports 0 non-Latin; atlas/define.md:80 says "the 29 captured fixtures"
  against 33 in testdata/entries/en alone; and cmd/define/live_property_test.go:65-69 —
  inside the file the plan already edits — still claims DCSCopyTextDefinition searches
  every ACTIVE dictionary with no API to select one, untrue since #23 M2. Produce the
  site list with a grep that ran (workshop/lessons.md:1962-1968), not by hand.
- **PQ-3** [Important] `restatement-not-consumer` Hand-editing 27 to 26 in the atlas is the fourth sweep of a number knownRawNotationEntries owns
  workshop/lessons.md:2038-2056 states the rule: a doc restating a fact the code owns,
  having drifted twice, becomes a CONSUMER. This section has drifted three times
  (lessons.md:13-15). The working precedent is TestREADMEQuotesThePromptsTheLoopActuallyPrints
  at cmd/define/doc_sync_test.go:27. Either make the Limits line derive from the constant
  or state the deferral as an explicit non-goal with its reason (ARCH-DRY, ARCH-PURPOSE).
- **PQ-4** [Important] `attribution-not-mechanized` Done-when row 3 wants the next drift attributable; the plan delivers a comment the test cannot enforce
  cmd/define/live_property_test.go:90 caps diagnostics at rawPipes <= 3, which is why the
  Log records "A full list needs the report cap raised for one run" — a cost this issue
  already paid once. A future 27 still prints one number and three samples, so the manual
  enumeration recurs. Classify each survivor by cause and log per-cause counts, so the run
  itself names which group moved.
- **PQ-5** [Important] `taxonomy-live-only` No test surface named, and the corpus hard-zero assertion collides with the new taxonomy
  TestNoRawPronunciationNotationSurvives (cmd/define/render_test.go:230-247) asserts ZERO raw
  pipes over the committed corpus, passing only because none of the 26 entries is in
  testdata/entries/en. The repo's precedent captures the exemplar — bases.txt and parrot.txt
  exist for the shapes atlas Limits names. Either capture one fixture per newly-named cause
  and say how the hard-zero test accommodates them, or state as an explicit non-goal that
  the taxonomy stays live-only, with the reason (ARCH-MOCK). The plan states no non-goals.
- **PQ-6** [Important] `negative-probe-no-control` The triage rests on a negative probe of "define brent" with no positive control stated
  Building ./cmd/define here yields "no dictionary entry" for brent, charge and ratio alike,
  under "no known en dictionary is installed; searching every active dictionary" — a context
  where every lookup fails and the curated path is never exercised. Done-when row 1's
  disposition needs the control named: same session, charge returns an entry, no
  "no known en dictionary" warning, checked around 70,886. Plan row 4 re-runs the sweep
  anyway, so this is free.
- **PQ-7** [Minor] `stale-frontmatter` deps: [tools#27] stays in the frontmatter while the Spec declares it moot
  The Spec says the dependency existed only to decide how to render a shape that no longer
  arrives, but the frontmatter still carries the blocking edge. No Plan row removes it.

## Open findings

- **PQ-1** [Important] `unbacked-behavior-claim` Plan row 2 asserts ratio/glop/logarithmic are "no longer reachable" — they render clean, they are not unreachable
- **PQ-2** [Important] `doc-sweep-incomplete` The doc sweep names only the Limits entry; four more measured claims are now false
- **PQ-3** [Important] `restatement-not-consumer` Hand-editing 27 to 26 in the atlas is the fourth sweep of a number knownRawNotationEntries owns
- **PQ-4** [Important] `attribution-not-mechanized` Done-when row 3 wants the next drift attributable; the plan delivers a comment the test cannot enforce
- **PQ-5** [Important] `taxonomy-live-only` No test surface named, and the corpus hard-zero assertion collides with the new taxonomy
- **PQ-6** [Important] `negative-probe-no-control` The triage rests on a negative probe of "define brent" with no positive control stated
- **PQ-7** [Minor] `stale-frontmatter` deps: [tools#27] stays in the frontmatter while the Spec declares it moot

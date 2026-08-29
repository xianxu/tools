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
    - "n": 2
      timestamp: "2026-08-28T18:07:53-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Spec now states the three render clean and calls the "unreachable" draft wrong.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: All four named instances appear in M2; the class is not swept — see the new finding.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Done-when row 5 and M2 commit to deriving from the constant on the doc_sync precedent.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: M1 classifies by cause and reports per-cause counts.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: testdata/rawnotation is outside capturedLanguages and testDict; pipe-family non-goal stated.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: 'Positive control named: charge returns an entry, no "no known en dictionary" warning.'
          round: 2
        - id: PQ-7
          disposition: addressed
          note: 'Frontmatter is deps: [] and the Spec says why.'
          round: 2
      findings:
        - id: PQ-8
          severity: Important
          title: Second in this family — the sweep enumeration is still hand-made; fix the rule, not the sites
          detail: |-
            Estimate says "six sites across two files"; an actual tree-wide grep returns a third
            file, cmd/define/parse.go:535, restating "27 of 70,897 live entries (0.04%)" in
            production code, plus atlas/define.md:92 ("at full width it was 27") and
            atlas/define.md:144 ("all 71,427 reachable entries" — a divergent fourth width).
            Measured prevalence: the count restated at parse.go:535, atlas:92, atlas:133; the
            width at atlas:83, :87, :133, parse.go:535, :144. Rule: paste the grep output into
            the Log as the enumeration, scope it to .go as well as .md, and single-source a
            number restated in code as well as docs (ARCH-DRY, ARCH-PURPOSE).
          family: doc-sweep-incomplete
          round: 2
        - id: PQ-9
          severity: Important
          title: The atlas cannot consume knownRawNotationEntries — the const is behind darwin and conformance
          detail: |-
            knownRawNotationEntries is declared in cmd/define/live_property_test.go:37 under
            //go:build darwin && conformance (:1). The cited precedent, doc_sync_test.go:27, is
            untagged and exists precisely so a doc claim fails the default build. An untagged
            consumer test will not compile against the tagged const; a tagged one only runs
            unsandboxed on darwin, which is the "ratchet had effectively never run" condition
            the Log already records. State where the constant lives after M2.
          family: derivation-behind-build-tag
          round: 2
        - id: PQ-10
          severity: Minor
          title: Second in this family — no classifier function named, and the classification is not stated as total
          detail: |-
            The gate asks for the unit-tested functions by name plus one strategy line per risky
            one; M1 says only "the classifier". The rule that covers the family: classification
            must be TOTAL — an explicit unclassified bucket, and an assertion that the per-cause
            counts sum to rawPipes. Without it a survivor matching no cause is dropped silently
            and the next drift is unattributable again, which is what PQ-4 asked to prevent.
            Risky-input class for the strategy line: entries whose content legitimately contains
            a pipe, and entries matching two causes at once.
          family: attribution-not-mechanized
          round: 2
        - id: PQ-11
          severity: Minor
          title: M1 and M2 are two review boundaries for a diff of one test file, one fixture dir and two docs
          detail: |-
            AGENTS.md section 3: each Mx commits to its own sdlc milestone-close plus a
            fresh-eyes review. At 1.22 estimated hours with a single coherent deliverable, plain
            checkboxes closing in one sdlc close look right; keep the split only if you genuinely
            intend to close M1 separately.
          family: over-split-milestones
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-28T18:11:48-07:00"
      agent: claude
      dispose:
        - id: PQ-8
          disposition: not-addressed
          note: Rule landed but scopes to "a document"; parse.go:535 and the Estimate's "six sites across two files" both survive.
          round: 3
        - id: PQ-9
          disposition: addressed
          note: Row 1 names rawnotation_test.go, package main, untagged, with the three consumers.
          round: 3
        - id: PQ-10
          disposition: not-addressed
          note: Risky-input classes named; classifier function still unnamed and totality still unstated.
          round: 3
        - id: PQ-11
          disposition: addressed
          note: Plain checkboxes, one sdlc close, stated at the top of the Plan.
          round: 3
      findings:
        - id: PQ-12
          severity: Important
          title: The live oracle is a two-oracle disjunction, not the pipe check the plan says it is
          detail: |-
            Plan and Spec both state "The check is IndexByte(out, '|') >= 0";
            live_property_test.go:84 is strayStress(out) != "" || IndexByte(out,'|') >= 0,
            with strayStress at invariant_test.go:42. By the plan's own taxonomy 14 of 26
            survivors (the headword-glued and phrase-block pronunciations) are stress-mark
            hits carrying no pipe, so a pipe-keyed classifier mis-handles the majority of
            the population Done-when row 3 wants attributed. Second in this family, so the
            deliverable is the rule, not the sentence: every claim about existing behavior
            in Spec or Plan carries a file:line and is read against it before being
            written. PQ-1 was falsified by running the code; this one by reading it.
            Prevalence: 2 of 2 behavioral claims this gate has checked were wrong.
          family: unbacked-behavior-claim
          round: 3
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

## Round 2 — 2026-08-28T18:07:53-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — Spec now states the three render clean and calls the "unreachable" draft wrong.
- PQ-2 — addressed — All four named instances appear in M2; the class is not swept — see the new finding.
- PQ-3 — addressed — Done-when row 5 and M2 commit to deriving from the constant on the doc_sync precedent.
- PQ-4 — addressed — M1 classifies by cause and reports per-cause counts.
- PQ-5 — addressed — testdata/rawnotation is outside capturedLanguages and testDict; pipe-family non-goal stated.
- PQ-6 — addressed — Positive control named: charge returns an entry, no "no known en dictionary" warning.
- PQ-7 — addressed — Frontmatter is deps: [] and the Spec says why.

### Raised

- **PQ-8** [Important] `doc-sweep-incomplete` Second in this family — the sweep enumeration is still hand-made; fix the rule, not the sites
  Estimate says "six sites across two files"; an actual tree-wide grep returns a third
  file, cmd/define/parse.go:535, restating "27 of 70,897 live entries (0.04%)" in
  production code, plus atlas/define.md:92 ("at full width it was 27") and
  atlas/define.md:144 ("all 71,427 reachable entries" — a divergent fourth width).
  Measured prevalence: the count restated at parse.go:535, atlas:92, atlas:133; the
  width at atlas:83, :87, :133, parse.go:535, :144. Rule: paste the grep output into
  the Log as the enumeration, scope it to .go as well as .md, and single-source a
  number restated in code as well as docs (ARCH-DRY, ARCH-PURPOSE).
- **PQ-9** [Important] `derivation-behind-build-tag` The atlas cannot consume knownRawNotationEntries — the const is behind darwin and conformance
  knownRawNotationEntries is declared in cmd/define/live_property_test.go:37 under
  //go:build darwin && conformance (:1). The cited precedent, doc_sync_test.go:27, is
  untagged and exists precisely so a doc claim fails the default build. An untagged
  consumer test will not compile against the tagged const; a tagged one only runs
  unsandboxed on darwin, which is the "ratchet had effectively never run" condition
  the Log already records. State where the constant lives after M2.
- **PQ-10** [Minor] `attribution-not-mechanized` Second in this family — no classifier function named, and the classification is not stated as total
  The gate asks for the unit-tested functions by name plus one strategy line per risky
  one; M1 says only "the classifier". The rule that covers the family: classification
  must be TOTAL — an explicit unclassified bucket, and an assertion that the per-cause
  counts sum to rawPipes. Without it a survivor matching no cause is dropped silently
  and the next drift is unattributable again, which is what PQ-4 asked to prevent.
  Risky-input class for the strategy line: entries whose content legitimately contains
  a pipe, and entries matching two causes at once.
- **PQ-11** [Minor] `over-split-milestones` M1 and M2 are two review boundaries for a diff of one test file, one fixture dir and two docs
  AGENTS.md section 3: each Mx commits to its own sdlc milestone-close plus a
  fresh-eyes review. At 1.22 estimated hours with a single coherent deliverable, plain
  checkboxes closing in one sdlc close look right; keep the split only if you genuinely
  intend to close M1 separately.

## Round 3 — 2026-08-28T18:11:48-07:00 (claude) — BLOCKED

### Disposed

- PQ-8 — not-addressed — Rule landed but scopes to "a document"; parse.go:535 and the Estimate's "six sites across two files" both survive.
- PQ-9 — addressed — Row 1 names rawnotation_test.go, package main, untagged, with the three consumers.
- PQ-10 — not-addressed — Risky-input classes named; classifier function still unnamed and totality still unstated.
- PQ-11 — addressed — Plain checkboxes, one sdlc close, stated at the top of the Plan.

### Raised

- **PQ-12** [Important] `unbacked-behavior-claim` The live oracle is a two-oracle disjunction, not the pipe check the plan says it is
  Plan and Spec both state "The check is IndexByte(out, '|') >= 0";
  live_property_test.go:84 is strayStress(out) != "" || IndexByte(out,'|') >= 0,
  with strayStress at invariant_test.go:42. By the plan's own taxonomy 14 of 26
  survivors (the headword-glued and phrase-block pronunciations) are stress-mark
  hits carrying no pipe, so a pipe-keyed classifier mis-handles the majority of
  the population Done-when row 3 wants attributed. Second in this family, so the
  deliverable is the rule, not the sentence: every claim about existing behavior
  in Spec or Plan carries a file:line and is read against it before being
  written. PQ-1 was falsified by running the code; this one by reading it.
  Prevalence: 2 of 2 behavioral claims this gate has checked were wrong.

## Open findings

- **PQ-8** [Important] `doc-sweep-incomplete` Second in this family — the sweep enumeration is still hand-made; fix the rule, not the sites
- **PQ-10** [Minor] `attribution-not-mechanized` Second in this family — no classifier function named, and the classification is not stated as total
- **PQ-12** [Important] `unbacked-behavior-claim` The live oracle is a two-oracle disjunction, not the pipe check the plan says it is

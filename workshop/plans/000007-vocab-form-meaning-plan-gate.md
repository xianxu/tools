---
gate: plan-quality
issue: 7
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-30T17:23:50-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: 'T5 edits play_loop.go lines that in-flight #38 is rewriting, and Choice''s multi-line Prompt destroys #38''s load-bearing premise'
          detail: |-
            #38 is status working (estimate 2.18, started 2026-08-30T16:08) and its plan D7 rebuilds the render/construct
            loop at play_loop.go:262-265 inside todaysQuestions, changes playSession's signature, and deletes a test from
            play_loop_test.go. Its stated simplification is that "a recall form's prompt IS the word (play/recall.go:29)
            … so there is no region-finding problem"; T2 makes Prompt the word plus four numbered glosses. Neither plan
            names the other and this issue's deps is ["tools#6"] only. State the ordering and which plan absorbs the other's
            change.
          family: undeclared-inflight-dep
          round: 1
        - id: PQ-2
          severity: Important
          title: The reduced taxonomy's yield is never measured against real deck sizes, so the settled axes will usually degrade to general
          detail: |-
            Measured over the committed corpus: 7 of 34 entries contain any domain label and 13 of 34 any register label,
            counting whole files, so gloss-leading labels are strictly fewer — on a corpus stocked with unusually polysemous
            words. A twelve-word deck will normally yield three general distractors (D3's stated degradation), which is the
            pre-settlement outcome the 2026-08-30 revision was written to avoid, while TestEveryAxisIsSelectable still passes
            on a synthetic deck. State the expected fill rate, the behaviour when axes cannot be filled, and what a
            domain-labelled distractor means when the target carries no domain label. ARCH-PURPOSE, ARCH-CONSTRAINTS.
          family: unmeasured-data-yield
          round: 1
        - id: PQ-3
          severity: Important
          title: D1's "never a near-synonym" guard names no mechanism, and Done-when 1 pins only literal gloss identity
          detail: |-
            The one-defensible-answer property is what this form trades its model veto for, but the issue's own settlement
            argues near-synonymy needs semantics, which this form refuses by design — so pickOptions cannot test the
            predicate it must satisfy. Done-when 1's red condition ("a second option's gloss is the target's") catches only
            exact duplication. Name the structural proxy actually in use and state that residual collisions are accepted,
            with the reason.
          family: unmechanized-guarantee
          round: 1
        - id: PQ-4
          severity: Important
          title: Which sense of a multi-sense entry becomes an option is never defined, though determinism, one-correct-answer and the axis all depend on it
          detail: |-
            D4 treats the option text as already existing, citing Sense.Gloss (parse.go:181-186), but an entry holds many
            senses across many blocks and nothing today selects one — todaysQuestions hands Recall the whole rendered entry
            (play_loop.go:262-265). T3's "candidates" is undefined: a word, or a (word, sense) pair. Since the NOAD label
            sits on a sense, choosing the axis is choosing the sense, so this is the centre of the design rather than a
            detail.
          family: undefined-selection-input
          round: 1
        - id: PQ-5
          severity: Important
          title: Done-when 7 pins play_loop_test.go as unchanged while T5 and Done-when 4 and 5 require changing it
          detail: |-
            T5 modifies todaysQuestions in play_loop.go, and TestASittingFallsBackToRecall and
            TestSittingWithNoModelAndNoNetwork are sitting-level tests belonging beside TestSessionRunsWithTheModelUnavailable
            (play_loop_test.go:161). Restate the pin as the property actually meant — playSession grows no per-form case —
            with a mechanical check that survives the plan's own edits.
          family: unsatisfiable-donewhen-pin
          round: 1
        - id: PQ-6
          severity: Minor
          title: pickOptions is placed in main though it is the pure heart of the form and needs no main type
          detail: |-
            play is mechanically guarded pure (play/purity_test.go: ImportsOnly with an empty allowlist, plus NoWallClock)
            and its doc says it decides how to ask and what an answer means (play/question.go:16-18). Once D5 has finished
            option material crossing the boundary, the seeded selection rule needs nothing from main, and placing it there
            leaves Done-when 3's determinism claim outside the guard that would enforce it. ARCH-PURE.
          family: pure-rule-outside-guard
          round: 1
      blocked: true
---

# Gate ledger — tools#7 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-30T17:23:50-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `undeclared-inflight-dep` T5 edits play_loop.go lines that in-flight #38 is rewriting, and Choice's multi-line Prompt destroys #38's load-bearing premise
  #38 is status working (estimate 2.18, started 2026-08-30T16:08) and its plan D7 rebuilds the render/construct
  loop at play_loop.go:262-265 inside todaysQuestions, changes playSession's signature, and deletes a test from
  play_loop_test.go. Its stated simplification is that "a recall form's prompt IS the word (play/recall.go:29)
  … so there is no region-finding problem"; T2 makes Prompt the word plus four numbered glosses. Neither plan
  names the other and this issue's deps is ["tools#6"] only. State the ordering and which plan absorbs the other's
  change.
- **PQ-2** [Important] `unmeasured-data-yield` The reduced taxonomy's yield is never measured against real deck sizes, so the settled axes will usually degrade to general
  Measured over the committed corpus: 7 of 34 entries contain any domain label and 13 of 34 any register label,
  counting whole files, so gloss-leading labels are strictly fewer — on a corpus stocked with unusually polysemous
  words. A twelve-word deck will normally yield three general distractors (D3's stated degradation), which is the
  pre-settlement outcome the 2026-08-30 revision was written to avoid, while TestEveryAxisIsSelectable still passes
  on a synthetic deck. State the expected fill rate, the behaviour when axes cannot be filled, and what a
  domain-labelled distractor means when the target carries no domain label. ARCH-PURPOSE, ARCH-CONSTRAINTS.
- **PQ-3** [Important] `unmechanized-guarantee` D1's "never a near-synonym" guard names no mechanism, and Done-when 1 pins only literal gloss identity
  The one-defensible-answer property is what this form trades its model veto for, but the issue's own settlement
  argues near-synonymy needs semantics, which this form refuses by design — so pickOptions cannot test the
  predicate it must satisfy. Done-when 1's red condition ("a second option's gloss is the target's") catches only
  exact duplication. Name the structural proxy actually in use and state that residual collisions are accepted,
  with the reason.
- **PQ-4** [Important] `undefined-selection-input` Which sense of a multi-sense entry becomes an option is never defined, though determinism, one-correct-answer and the axis all depend on it
  D4 treats the option text as already existing, citing Sense.Gloss (parse.go:181-186), but an entry holds many
  senses across many blocks and nothing today selects one — todaysQuestions hands Recall the whole rendered entry
  (play_loop.go:262-265). T3's "candidates" is undefined: a word, or a (word, sense) pair. Since the NOAD label
  sits on a sense, choosing the axis is choosing the sense, so this is the centre of the design rather than a
  detail.
- **PQ-5** [Important] `unsatisfiable-donewhen-pin` Done-when 7 pins play_loop_test.go as unchanged while T5 and Done-when 4 and 5 require changing it
  T5 modifies todaysQuestions in play_loop.go, and TestASittingFallsBackToRecall and
  TestSittingWithNoModelAndNoNetwork are sitting-level tests belonging beside TestSessionRunsWithTheModelUnavailable
  (play_loop_test.go:161). Restate the pin as the property actually meant — playSession grows no per-form case —
  with a mechanical check that survives the plan's own edits.
- **PQ-6** [Minor] `pure-rule-outside-guard` pickOptions is placed in main though it is the pure heart of the form and needs no main type
  play is mechanically guarded pure (play/purity_test.go: ImportsOnly with an empty allowlist, plus NoWallClock)
  and its doc says it decides how to ask and what an answer means (play/question.go:16-18). Once D5 has finished
  option material crossing the boundary, the seeded selection rule needs nothing from main, and placing it there
  leaves Done-when 3's determinism claim outside the guard that would enforce it. ARCH-PURE.

## Open findings

- **PQ-1** [Critical] `undeclared-inflight-dep` T5 edits play_loop.go lines that in-flight #38 is rewriting, and Choice's multi-line Prompt destroys #38's load-bearing premise
- **PQ-2** [Important] `unmeasured-data-yield` The reduced taxonomy's yield is never measured against real deck sizes, so the settled axes will usually degrade to general
- **PQ-3** [Important] `unmechanized-guarantee` D1's "never a near-synonym" guard names no mechanism, and Done-when 1 pins only literal gloss identity
- **PQ-4** [Important] `undefined-selection-input` Which sense of a multi-sense entry becomes an option is never defined, though determinism, one-correct-answer and the axis all depend on it
- **PQ-5** [Important] `unsatisfiable-donewhen-pin` Done-when 7 pins play_loop_test.go as unchanged while T5 and Done-when 4 and 5 require changing it
- **PQ-6** [Minor] `pure-rule-outside-guard` pickOptions is placed in main though it is the pure heart of the form and needs no main type

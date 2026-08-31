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
    - "n": 2
      timestamp: "2026-08-30T17:40:34-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: D1a settles it by constraint (word alone on line 0) and states the ordering; the peer-side record is a separate new finding.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: D2a's counts reproduce exactly against the 34-entry corpus, and it states the fill rate, the domain/register/general priority and the general fallback.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: D3a names the mechanism (NOAD cross-reference), measures it, and states it as a reduction with the residual explicitly accepted.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: D4a fixes the correct option to first sense of first block and a distractor's sense to the one carrying the sought axis.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: The row no longer pins play_loop_test.go; its replacement clause is a new instance of the same family, raised below.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: pickOptions moves into play; the guard consequence of that move is raised below.
          round: 2
      findings:
        - id: PQ-7
          severity: Important
          title: pickOptions and Choice move into play, whose purity guard allows ZERO imports, and the plan does not say which side of that the code lands on
          detail: |-
            play/purity_test.go:21 is ImportsOnly(t, playPkg, []string{}) and `go list -f '{{join .Imports}}' ./cmd/define/play`
            returns empty today; puretest.go calls zero imports "the strongest possible version of the claim", and
            puretest_test.go:63 pins a dedicated `nothing` fixture because play importing nothing was its only pin (BR-41).
            T2's numbered-option Prompt, T3's seeded selection and D3a's word-boundary gloss match all want fmt/strings/math-rand.
            State the choice: widen the allowlist (with the reason, since the package doc says that must be conscious) or write
            Choice and pickOptions import-free with a hand-rolled digit, PRNG and scan. ARCH-PURE.
          family: guard-allowlist-unstated
          round: 2
        - id: PQ-8
          severity: Important
          title: Done-when 7's replacement pin says "play/* unchanged" while T2, T3 and T4 all write into play/
          detail: |-
            This is the 2nd finding in family unsatisfiable-donewhen-pin. Earlier rounds fixed the play_loop_test.go instance
            and the same defect reappeared one directory up. Do NOT fix this instance alone — state the rule: a Done-when pin
            must be a predicate over behaviour (a named test, or a grep for a form name inside playSession/Apply), never
            "file X unchanged" when any task in the plan writes into X. Then sweep all eight rows against T1-T6's write set,
            since T2 creates play/choice.go, T3 puts pickOptions there and T4 adds a field to Outcome in play/session.go.
            TestSessionIsFormAgnostic (play/session_test.go:327) already pins the property on its own. ARCH-PURPOSE.
          family: unsatisfiable-donewhen-pin
          round: 2
        - id: PQ-9
          severity: Important
          title: The coupling note claimed to be in 38's Log is a separate new issue file, and it breaks sdlc issue resolution for 38
          detail: |-
            This is the 2nd finding in family undeclared-inflight-dep. Do NOT fix only this instance — state the rule: a
            peer-issue write is not done until the peer's own file is read back and `sdlc issue show <peer>` still resolves to
            exactly one file. Measured: commit 3bb58b1 added workshop/issues/000038-play-clickable-words.md, a frontmatter-less
            new file; workshop/issues/000038-play-clickable.md contains no mention of 7; and `sdlc issue show 38` now errors
            with "multiple issue files match", which takes every 38 lifecycle verb with it. D1a's claim that the note is
            recorded where a resumer meets it is not true of the tree.
          family: undeclared-inflight-dep
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-30T17:43:07-07:00"
      agent: claude
      dispose:
        - id: PQ-7
          disposition: addressed
          note: D5a keeps the empty allowlist and moves the prose work to main; verified go list returns no imports and purity_test.go:21 is still []string{}.
          round: 3
        - id: PQ-8
          disposition: addressed
          note: Rule stated above the table, all eight rows swept to behavioural predicates; TestSessionIsFormAgnostic confirmed at play/session_test.go:327.
          round: 3
        - id: PQ-9
          disposition: addressed
          note: Stray 000038-play-clickable-words.md deleted, note lives in 000038-play-clickable.md:182-201, and sdlc issue show 38 resolves to one file.
          round: 3
      blocked: false
content_hash: fbdf83f929f6a33bb8ab5f371d6aadbeb84b2e9a99b3527af1ef0b77c1486ea8
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

## Round 2 — 2026-08-30T17:40:34-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — D1a settles it by constraint (word alone on line 0) and states the ordering; the peer-side record is a separate new finding.
- PQ-2 — addressed — D2a's counts reproduce exactly against the 34-entry corpus, and it states the fill rate, the domain/register/general priority and the general fallback.
- PQ-3 — addressed — D3a names the mechanism (NOAD cross-reference), measures it, and states it as a reduction with the residual explicitly accepted.
- PQ-4 — addressed — D4a fixes the correct option to first sense of first block and a distractor's sense to the one carrying the sought axis.
- PQ-5 — addressed — The row no longer pins play_loop_test.go; its replacement clause is a new instance of the same family, raised below.
- PQ-6 — addressed — pickOptions moves into play; the guard consequence of that move is raised below.

### Raised

- **PQ-7** [Important] `guard-allowlist-unstated` pickOptions and Choice move into play, whose purity guard allows ZERO imports, and the plan does not say which side of that the code lands on
  play/purity_test.go:21 is ImportsOnly(t, playPkg, []string{}) and `go list -f '{{join .Imports}}' ./cmd/define/play`
  returns empty today; puretest.go calls zero imports "the strongest possible version of the claim", and
  puretest_test.go:63 pins a dedicated `nothing` fixture because play importing nothing was its only pin (BR-41).
  T2's numbered-option Prompt, T3's seeded selection and D3a's word-boundary gloss match all want fmt/strings/math-rand.
  State the choice: widen the allowlist (with the reason, since the package doc says that must be conscious) or write
  Choice and pickOptions import-free with a hand-rolled digit, PRNG and scan. ARCH-PURE.
- **PQ-8** [Important] `unsatisfiable-donewhen-pin` Done-when 7's replacement pin says "play/* unchanged" while T2, T3 and T4 all write into play/
  This is the 2nd finding in family unsatisfiable-donewhen-pin. Earlier rounds fixed the play_loop_test.go instance
  and the same defect reappeared one directory up. Do NOT fix this instance alone — state the rule: a Done-when pin
  must be a predicate over behaviour (a named test, or a grep for a form name inside playSession/Apply), never
  "file X unchanged" when any task in the plan writes into X. Then sweep all eight rows against T1-T6's write set,
  since T2 creates play/choice.go, T3 puts pickOptions there and T4 adds a field to Outcome in play/session.go.
  TestSessionIsFormAgnostic (play/session_test.go:327) already pins the property on its own. ARCH-PURPOSE.
- **PQ-9** [Important] `undeclared-inflight-dep` The coupling note claimed to be in 38's Log is a separate new issue file, and it breaks sdlc issue resolution for 38
  This is the 2nd finding in family undeclared-inflight-dep. Do NOT fix only this instance — state the rule: a
  peer-issue write is not done until the peer's own file is read back and `sdlc issue show <peer>` still resolves to
  exactly one file. Measured: commit 3bb58b1 added workshop/issues/000038-play-clickable-words.md, a frontmatter-less
  new file; workshop/issues/000038-play-clickable.md contains no mention of 7; and `sdlc issue show 38` now errors
  with "multiple issue files match", which takes every 38 lifecycle verb with it. D1a's claim that the note is
  recorded where a resumer meets it is not true of the tree.

## Round 3 — 2026-08-30T17:43:07-07:00 (claude) — passed

### Disposed

- PQ-7 — addressed — D5a keeps the empty allowlist and moves the prose work to main; verified go list returns no imports and purity_test.go:21 is still []string{}.
- PQ-8 — addressed — Rule stated above the table, all eight rows swept to behavioural predicates; TestSessionIsFormAgnostic confirmed at play/session_test.go:327.
- PQ-9 — addressed — Stray 000038-play-clickable-words.md deleted, note lives in 000038-play-clickable.md:182-201, and sdlc issue show 38 resolves to one file.

## Open findings

(none — every finding has been disposed)

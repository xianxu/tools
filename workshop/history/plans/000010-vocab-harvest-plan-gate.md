---
gate: plan-quality
issue: 10
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-04T11:29:22-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: 'Band''s DRY rationale rests on a false claim about #17, and the learner-band read has no named seam'
          detail: |-
            The plan says Band "reuses that vocabulary rather than defining a parallel enum" because
            "#17 already assigns the learner one". In fact levelClaim.Band is a free-form string
            (cmd/define/reflect.go:128) with no CEFR validation, reachable only as raw markdown via
            Store.UserModel() (cmd/define/store/store.go), whose sole consumer pastes the whole document
            into a prompt (cmd/define/ask.go:268). As planned, store.Band IS the parallel spelling, and
            pickDistractors' other input — the learner's band — has no stated source. Either make
            levelClaim.Band derive from store.Band (ARCH-PURPOSE: single-source is not done until the
            motivating consumer derives) or name the parse and its absent-band behavior.
          family: unbacked-existing-behavior
          round: 1
        - id: PQ-2
          severity: Important
          title: facts/ is flat with no stated reason, and a band is language-derived state cached forever
          detail: |-
            cmd/define/store/yaml.go:125-134 states the repo's rule — per-language iff DERIVED from the
            language-scoped deck — and records user-model.md as the round that rule cost (#23). A CEFR
            band and a domain are language-derived, #18 (spanish) is open, and homographs like red,
            once, sensible and actual collide. Combined with "cached forever" and the planned
            "second write REPLACES" row, the collision is permanent and reversing it is a migration.
            The plan must say facts/<lang>/<key>.yaml or say explicitly why flat like usage/ is right.
          family: derived-state-scoping
          round: 1
        - id: PQ-3
          severity: Important
          title: 'Item carrying finished Distractors moves selection and the veto from #12 to #10, undeclared'
          detail: |-
            workshop/issues/000012-vocab-form-cloze.md:53-57 owns "options drawn from pool + deck, never
            model-generated", the sycophantic/obsequious near-synonym rejection, and the seam-unavailable
            veto skip; internal/llm/golden_schema_test.go:11 records the veto verdict type as "#12 will
            define its own". The plan builds all of it in M2 Task 5 and refers to #12 only as a consumer
            that "picks among them". Declare the ownership move and say what happens to #12's Done-when
            rows, or leave selection at question time.
          family: undeclared-cross-issue-scope
          round: 1
        - id: PQ-4
          severity: Important
          title: WordFacts.Domain is a third domain vocabulary; topicSpread's arithmetic depends on which
          detail: |-
            noadDomainLabels (cmd/define/glosslabel.go:47) is a deliberately CLOSED table this repo owns,
            and #17's domainClaim.Name (cmd/define/reflect.go:143) is free model text. The plan adds a
            third without naming either. topicSpread counts distinct domains and is the one measure taken
            with no model, so an open vocabulary lets "Medicine"/"medicine"/"med" inflate it; and #12's
            Spec wants a DIFFERENT NOAD domain label where this plan selects at the SAME domain (ARCH-DRY).
          family: parallel-vocabulary
          round: 1
        - id: PQ-5
          severity: Important
          title: agreement over N assignments contradicts one-call-per-word, so Done-when 3 has no mechanism
          detail: |-
            The envelope says "one band call per unbanded word" and Done-when 2 asserts a second run makes
            zero calls, yet --harvest is to "print the agreement it measured" over N assignments. N is
            never sourced. The unit test drives a fake "seeded to vary", which measures the fake, not the
            model; the conformance row is binary rather than a fraction with a floor. State which words
            are asked N times, when, and the floor.
          family: unstated-measurement-mechanism
          round: 1
        - id: PQ-6
          severity: Important
          title: |-
            Model-authored stems, answers, distractors and domains reach disk and a terminal with no
            neutralising seam named
          detail: |-
            cmd/define/usermodel.go:213 exists because a model-supplied band carrying a newline forged a
            "define: ..." diagnostic line, and the fix was one pass where structure is built. This plan
            persists four new free-text model fields and later renders them onto the board, naming a
            refusing parse for Band only. Say where the neutralisation happens and how a truncated or
            hand-edited facts/*.yaml degrades (ARCH-SECURE).
          family: untrusted-model-text
          round: 1
        - id: PQ-7
          severity: Minor
          title: Task 2's golden is placed in internal/llm/llmtest/testdata/, not the consuming package
          detail: |-
            llmtest.AssertGolden(t, "testdata", ...) writes into the caller's package — see
            cmd/define/askctx_test.go:29 and cmd/define/reflectprompt_test.go:24, both landing in
            cmd/define/testdata/golden/. internal/llm/llmtest/testdata/ holds llmtest's own wire fixtures.
          family: convention-misplacement
          round: 1
        - id: PQ-8
          severity: Minor
          title: No per-run bound on model calls; the envelope defers to "bounded in practice by the deck"
          detail: |-
            M2 authors one call per item over a deck of a few thousand. cmd/define/optionpool.go:24
            (poolCap = 40) is this repo's own precedent for bounding batch work with a stated reason.
            A --limit, or an explicit statement that a full-deck run is the intended cost, would close it.
          family: unbounded-batch-cost
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-04T11:37:44-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: 'Rationale corrected and the loop closed: #17 writes through ParseBand, renderUserModel emits level:, parseLearnerBand reads it back.'
          round: 2
        - id: PQ-2
          disposition: addressed
          note: facts/<lang>/<key>.yaml, citing yaml.go's derivation rule; factsDir mirrors wordsDir.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: 'Task 5 Step 0 declares the move, rewrites #12''s rows, and corrects golden_schema_test.go:11.'
          round: 2
        - id: PQ-4
          disposition: addressed
          note: 'One closed set owned by store, dictionary as primary source; the same-domain-plus-veto rule supersedes #12''s different-label heuristic via Step 0''s rewrite.'
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Separate --agreement mode over K=20, N=5, writing nothing; floor asserted in conformance, arithmetic in the unit test.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: sanitiseFacts/sanitiseItem one pass at the write; unparseable facts read as absent.
          round: 2
        - id: PQ-7
          disposition: addressed
          round: 2
        - id: PQ-8
          disposition: addressed
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-04T11:41:20-07:00"
      agent: claude
      blocked: false
      protocol_error: no valid findings block
content_hash: b21f3822f44ee04d0896d87a4cf22b48c69596a0c2106553bdd16149adaa5d64
---

# Gate ledger — tools#10 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-04T11:29:22-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `unbacked-existing-behavior` Band's DRY rationale rests on a false claim about #17, and the learner-band read has no named seam
  The plan says Band "reuses that vocabulary rather than defining a parallel enum" because
  "#17 already assigns the learner one". In fact levelClaim.Band is a free-form string
  (cmd/define/reflect.go:128) with no CEFR validation, reachable only as raw markdown via
  Store.UserModel() (cmd/define/store/store.go), whose sole consumer pastes the whole document
  into a prompt (cmd/define/ask.go:268). As planned, store.Band IS the parallel spelling, and
  pickDistractors' other input — the learner's band — has no stated source. Either make
  levelClaim.Band derive from store.Band (ARCH-PURPOSE: single-source is not done until the
  motivating consumer derives) or name the parse and its absent-band behavior.
- **PQ-2** [Important] `derived-state-scoping` facts/ is flat with no stated reason, and a band is language-derived state cached forever
  cmd/define/store/yaml.go:125-134 states the repo's rule — per-language iff DERIVED from the
  language-scoped deck — and records user-model.md as the round that rule cost (#23). A CEFR
  band and a domain are language-derived, #18 (spanish) is open, and homographs like red,
  once, sensible and actual collide. Combined with "cached forever" and the planned
  "second write REPLACES" row, the collision is permanent and reversing it is a migration.
  The plan must say facts/<lang>/<key>.yaml or say explicitly why flat like usage/ is right.
- **PQ-3** [Important] `undeclared-cross-issue-scope` Item carrying finished Distractors moves selection and the veto from #12 to #10, undeclared
  workshop/issues/000012-vocab-form-cloze.md:53-57 owns "options drawn from pool + deck, never
  model-generated", the sycophantic/obsequious near-synonym rejection, and the seam-unavailable
  veto skip; internal/llm/golden_schema_test.go:11 records the veto verdict type as "#12 will
  define its own". The plan builds all of it in M2 Task 5 and refers to #12 only as a consumer
  that "picks among them". Declare the ownership move and say what happens to #12's Done-when
  rows, or leave selection at question time.
- **PQ-4** [Important] `parallel-vocabulary` WordFacts.Domain is a third domain vocabulary; topicSpread's arithmetic depends on which
  noadDomainLabels (cmd/define/glosslabel.go:47) is a deliberately CLOSED table this repo owns,
  and #17's domainClaim.Name (cmd/define/reflect.go:143) is free model text. The plan adds a
  third without naming either. topicSpread counts distinct domains and is the one measure taken
  with no model, so an open vocabulary lets "Medicine"/"medicine"/"med" inflate it; and #12's
  Spec wants a DIFFERENT NOAD domain label where this plan selects at the SAME domain (ARCH-DRY).
- **PQ-5** [Important] `unstated-measurement-mechanism` agreement over N assignments contradicts one-call-per-word, so Done-when 3 has no mechanism
  The envelope says "one band call per unbanded word" and Done-when 2 asserts a second run makes
  zero calls, yet --harvest is to "print the agreement it measured" over N assignments. N is
  never sourced. The unit test drives a fake "seeded to vary", which measures the fake, not the
  model; the conformance row is binary rather than a fraction with a floor. State which words
  are asked N times, when, and the floor.
- **PQ-6** [Important] `untrusted-model-text` Model-authored stems, answers, distractors and domains reach disk and a terminal with no
neutralising seam named
  cmd/define/usermodel.go:213 exists because a model-supplied band carrying a newline forged a
  "define: ..." diagnostic line, and the fix was one pass where structure is built. This plan
  persists four new free-text model fields and later renders them onto the board, naming a
  refusing parse for Band only. Say where the neutralisation happens and how a truncated or
  hand-edited facts/*.yaml degrades (ARCH-SECURE).
- **PQ-7** [Minor] `convention-misplacement` Task 2's golden is placed in internal/llm/llmtest/testdata/, not the consuming package
  llmtest.AssertGolden(t, "testdata", ...) writes into the caller's package — see
  cmd/define/askctx_test.go:29 and cmd/define/reflectprompt_test.go:24, both landing in
  cmd/define/testdata/golden/. internal/llm/llmtest/testdata/ holds llmtest's own wire fixtures.
- **PQ-8** [Minor] `unbounded-batch-cost` No per-run bound on model calls; the envelope defers to "bounded in practice by the deck"
  M2 authors one call per item over a deck of a few thousand. cmd/define/optionpool.go:24
  (poolCap = 40) is this repo's own precedent for bounding batch work with a stated reason.
  A --limit, or an explicit statement that a full-deck run is the intended cost, would close it.

## Round 2 — 2026-09-04T11:37:44-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Rationale corrected and the loop closed: #17 writes through ParseBand, renderUserModel emits level:, parseLearnerBand reads it back.
- PQ-2 — addressed — facts/<lang>/<key>.yaml, citing yaml.go's derivation rule; factsDir mirrors wordsDir.
- PQ-3 — addressed — Task 5 Step 0 declares the move, rewrites #12's rows, and corrects golden_schema_test.go:11.
- PQ-4 — addressed — One closed set owned by store, dictionary as primary source; the same-domain-plus-veto rule supersedes #12's different-label heuristic via Step 0's rewrite.
- PQ-5 — addressed — Separate --agreement mode over K=20, N=5, writing nothing; floor asserted in conformance, arithmetic in the unit test.
- PQ-6 — addressed — sanitiseFacts/sanitiseItem one pass at the write; unparseable facts read as absent.
- PQ-7 — addressed
- PQ-8 — addressed

## Round 3 — 2026-09-04T11:41:20-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

(none — every finding has been disposed)

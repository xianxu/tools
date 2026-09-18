---
gate: plan-quality
issue: 72
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-17T15:12:11-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: own-at-open breaks owned-path vocabulary highlighting; the plan never names answer_language.go
          detail: |-
            languageAnswer.accept renders owned spans via highlightRegion (answer_language.go:42;
            highlightwriter.go:269), which builds a fresh highlightWriter per call and flushes it — it
            only works because the owned span arrives whole today. Emitting at open makes each owned
            call one rune, so no deck word inside a lang passage highlights. Measured: three one-rune
            spans yield knownOn count 0, one three-rune span yields 1;
            TestLanguageAnswerForeignHomographDoesNotUseTargetVocabulary
            (answer_language_test.go:71) fails, and Done-when's "highlighted as before" is unmet. The
            plan must add a step routing owned text through a persistent highlighter per ownership run
            (flushed at ownership change as answer_language.go:33 already does, vocabulary swapped per
            answer_language.go:39-41), and state the preserved invariant that a word must not highlight
            across an ownership boundary.
          family: whole-chunk-assumption
          round: 1
        - id: PQ-2
          severity: Important
          title: the resulting state x event -> effect table is not written, though a test pins it
          detail: |-
            TestLanguageDecodeTransitions (language_decode_test.go:90-92) asserts an independent 3x7
            next-state and 3x6 effect matrix. Own-at-open kills decodeFlushOwned (language_decode.go:146),
            decodeFlushNeutral (:150) and decodeLimit, whose only producer is :141. Say whether each
            leaves the enum or survives as a no-op, what becomes of the ok=false rejection rule documented
            at :42-44, and where d.lang is cleared once the flush effects stop clearing it (:149, :153).
            ARCH-ORDER wants the enumeration; ARCH-PURPOSE wants the whole class swept, not just d.body
            and languageBodyLimit.
          family: state-table-not-enumerated
          round: 1
        - id: PQ-3
          severity: Important
          title: the stated decoder hold omits the entity buffer
          detail: |-
            The envelope says "hold <= 64 bytes (an incomplete marker candidate) plus one partial rune"
            and ARCH-FUNERAL says that candidate is "the only hold left". d.entity
            (language_decode.go:230-243) retains a pending entity up to 64 bytes, and there are two
            answerTextFilter instances each holding up to 3 bytes of a partial rune (answer_text.go:30-32).
            Correct the figure — it is the oracle the replacement bound assertion will use.
          family: unbacked-existing-behavior
          round: 1
        - id: PQ-4
          severity: Important
          title: |-
            no functions named, no strategy per risky function, and the memory-bound guard is deleted
            with nothing named to replace it
          detail: |-
            Row 1 says only "failing test first ... through the production chain". Name the functions:
            stepLanguageDecode (table), languageDecoder.lex (rune scanner over untrusted model output ->
            extend FuzzLanguageDecoderChunks at language_decode_test.go:75, asserting retained bytes after
            each chunk and seeded with own-at-open forms), and the restructured owned path in accept.
            TestAnswerControlPayloadAndAnnotationMemoryAreBounded (:119-128) reads d.body.Len() >
            languageBodyLimit and dies with the constant — name the invariant that replaces it. Also say
            what happens to the 16385-byte over-limit fixture at :37, since the over-limit case ceases to
            exist while the Spec still says a segment that "exceeds a bound" keeps its announced language.
          family: test-surface-not-named
          round: 1
        - id: PQ-5
          severity: Minor
          title: the 0.5 s first-text budget is verified only by a one-off Log measurement
          detail: |-
            The Spec already replayed recorded deltas at their real 60 ms spacing through the production
            chain to get the 0.3 s figure — name that replay as the seam the timing assertion runs
            against (ARCH-MOCK), so the ARCH-CONSTRAINTS budget has a mechanical guard rather than a
            number recorded once in the Log.
          family: envelope-guard-unautomated
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-17T15:14:04-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Plan unchanged since round 1; answer_language.go still unnamed, highlightRegion still per-call.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: Spec still says "what changes is its effect table" without writing the table.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: Envelope bullet still reads 64 bytes plus one partial rune; d.entity and two filters unaccounted.
          round: 2
        - id: PQ-4
          disposition: not-addressed
          note: Row 1 verbatim unchanged; no functions named, memory-bound test still has no replacement invariant.
          round: 2
        - id: PQ-5
          disposition: not-addressed
          note: Minor; timing budget still guarded only by a Log measurement.
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-17T15:21:59-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: 'New Spec section plus plan row 4: persistent per-ownership-run highlighter, boundary invariant stated.'
          round: 3
        - id: PQ-2
          disposition: addressed
          note: 3x5 table written; two flush effects, decodeLimit event and the ok=false rule disposed; one d.lang rule.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: Envelope now enumerates marker, entity, both filter pendings and literalText; total <= 200 B.
          round: 3
        - id: PQ-4
          disposition: addressed
          note: Three functions named with a strategy line each; replacement retention invariant and the 16385-byte row disposed.
          round: 3
        - id: PQ-5
          disposition: addressed
          note: Budget guarded by the AfterText/FinishRelease ordering barrier rather than a flaky wall-clock assertion.
          round: 3
      blocked: false
content_hash: 7f8dd84562c438bb648459a1b63fa396968a71b1d3951d81e12f7f5e2ce75cac
---

# Gate ledger — tools#72 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-17T15:12:11-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `whole-chunk-assumption` own-at-open breaks owned-path vocabulary highlighting; the plan never names answer_language.go
  languageAnswer.accept renders owned spans via highlightRegion (answer_language.go:42;
  highlightwriter.go:269), which builds a fresh highlightWriter per call and flushes it — it
  only works because the owned span arrives whole today. Emitting at open makes each owned
  call one rune, so no deck word inside a lang passage highlights. Measured: three one-rune
  spans yield knownOn count 0, one three-rune span yields 1;
  TestLanguageAnswerForeignHomographDoesNotUseTargetVocabulary
  (answer_language_test.go:71) fails, and Done-when's "highlighted as before" is unmet. The
  plan must add a step routing owned text through a persistent highlighter per ownership run
  (flushed at ownership change as answer_language.go:33 already does, vocabulary swapped per
  answer_language.go:39-41), and state the preserved invariant that a word must not highlight
  across an ownership boundary.
- **PQ-2** [Important] `state-table-not-enumerated` the resulting state x event -> effect table is not written, though a test pins it
  TestLanguageDecodeTransitions (language_decode_test.go:90-92) asserts an independent 3x7
  next-state and 3x6 effect matrix. Own-at-open kills decodeFlushOwned (language_decode.go:146),
  decodeFlushNeutral (:150) and decodeLimit, whose only producer is :141. Say whether each
  leaves the enum or survives as a no-op, what becomes of the ok=false rejection rule documented
  at :42-44, and where d.lang is cleared once the flush effects stop clearing it (:149, :153).
  ARCH-ORDER wants the enumeration; ARCH-PURPOSE wants the whole class swept, not just d.body
  and languageBodyLimit.
- **PQ-3** [Important] `unbacked-existing-behavior` the stated decoder hold omits the entity buffer
  The envelope says "hold <= 64 bytes (an incomplete marker candidate) plus one partial rune"
  and ARCH-FUNERAL says that candidate is "the only hold left". d.entity
  (language_decode.go:230-243) retains a pending entity up to 64 bytes, and there are two
  answerTextFilter instances each holding up to 3 bytes of a partial rune (answer_text.go:30-32).
  Correct the figure — it is the oracle the replacement bound assertion will use.
- **PQ-4** [Important] `test-surface-not-named` no functions named, no strategy per risky function, and the memory-bound guard is deleted
with nothing named to replace it
  Row 1 says only "failing test first ... through the production chain". Name the functions:
  stepLanguageDecode (table), languageDecoder.lex (rune scanner over untrusted model output ->
  extend FuzzLanguageDecoderChunks at language_decode_test.go:75, asserting retained bytes after
  each chunk and seeded with own-at-open forms), and the restructured owned path in accept.
  TestAnswerControlPayloadAndAnnotationMemoryAreBounded (:119-128) reads d.body.Len() >
  languageBodyLimit and dies with the constant — name the invariant that replaces it. Also say
  what happens to the 16385-byte over-limit fixture at :37, since the over-limit case ceases to
  exist while the Spec still says a segment that "exceeds a bound" keeps its announced language.
- **PQ-5** [Minor] `envelope-guard-unautomated` the 0.5 s first-text budget is verified only by a one-off Log measurement
  The Spec already replayed recorded deltas at their real 60 ms spacing through the production
  chain to get the 0.3 s figure — name that replay as the seam the timing assertion runs
  against (ARCH-MOCK), so the ARCH-CONSTRAINTS budget has a mechanical guard rather than a
  number recorded once in the Log.

## Round 2 — 2026-09-17T15:14:04-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — not-addressed — Plan unchanged since round 1; answer_language.go still unnamed, highlightRegion still per-call.
- PQ-2 — not-addressed — Spec still says "what changes is its effect table" without writing the table.
- PQ-3 — not-addressed — Envelope bullet still reads 64 bytes plus one partial rune; d.entity and two filters unaccounted.
- PQ-4 — not-addressed — Row 1 verbatim unchanged; no functions named, memory-bound test still has no replacement invariant.
- PQ-5 — not-addressed — Minor; timing budget still guarded only by a Log measurement.

## Round 3 — 2026-09-17T15:21:59-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — New Spec section plus plan row 4: persistent per-ownership-run highlighter, boundary invariant stated.
- PQ-2 — addressed — 3x5 table written; two flush effects, decodeLimit event and the ok=false rule disposed; one d.lang rule.
- PQ-3 — addressed — Envelope now enumerates marker, entity, both filter pendings and literalText; total <= 200 B.
- PQ-4 — addressed — Three functions named with a strategy line each; replacement retention invariant and the 16385-byte row disposed.
- PQ-5 — addressed — Budget guarded by the AfterText/FinishRelease ordering barrier rather than a flaky wall-clock assertion.

## Open findings

(none — every finding has been disposed)

---
gate: plan-quality
issue: 21
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-26T13:20:30-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: M3's end-to-end test assumes llmtest.Fake can serve invented streamed text; it cannot
          detail: |-
            misapplied() at internal/llm/llmtest/fake.go:188-215 rejects a scripted
            Reply{Text} on a streaming request with a 400, and serveStream at :461-472
            always replays a committed .sse capture — so Task 8 Step 1 cannot choose an
            answer with a known word split across deltas. Name the real route: derive the
            seeded word from a delta boundary in stream-sample.sse (it already splits
            "rather" as "...authority r"/"ather than" and "painstakingly" as
            "(painstak"/"ingly"), computed from the capture rather than hardcoded.
          family: fake-capability-mismatch
          round: 1
        - id: PQ-2
          severity: Important
          title: highlightWriter's rules for held-back text and pass-through escapes conflict on order
          detail: |-
            "Pass escape sequences through untouched" and "hold back a tail that could
            extend into a phrase" contradict when an escape arrives mid-hold — e.g. deck
            "hot dog" against Render's `\x1b[1;36mhot\x1b[0m` (render.go:26). The plan's
            properties cannot catch the resulting reorder: both the escape-stripped
            equality and the one-call equivalence are blind to it. State the ordering rule
            and assert on full bytes.
          family: writer-emission-order
          round: 1
        - id: PQ-3
          severity: Important
          title: Plan never says what highlightWriter does on a downstream short write or error
          detail: |-
            Task 6's "Write returns len(p) and a nil error even when holding bytes back"
            reads as "always nil error". highlightWriter wraps crlfWriter
            (replraw.go:166), which returns a short count on partial writes and is tested
            for it (crlf_test.go:50, :78). Byte loss here is the plan's own stated worst
            failure. Specify error propagation plus caller-unit mapping, and test against a
            short writer.
          family: writer-downstream-contract
          round: 1
        - id: PQ-4
          severity: Minor
          title: Tasks 2, 5 and 6 enumerate test cases in prose instead of naming a strategy
          detail: |-
            Compress each to one strategy line per risky function. Separately, wordRuns is
            the byte-offset source of truth for both consumers but has no property of its
            own — add one (offsets increasing, non-overlapping, every run slicing cleanly).
          family: test-enumeration-in-prose
          round: 1
        - id: PQ-5
          severity: Minor
          title: storeVocabulary and memVocabulary each grow their own set (ARCH-DRY)
          detail: |-
            Task 1 Steps 3 and 5 describe a map+maxWords in memVocabulary and "reads Deck()
            once into a set" in storeVocabulary. Say explicitly that storeVocabulary embeds
            memVocabulary so Add/Has/MaxPhraseWords have one implementation.
          family: duplicate-implementation
          round: 1
        - id: PQ-6
          severity: Minor
          title: Plan does not say whether a phrase may match across a newline
          detail: |-
            store.Key collapses all whitespace (store/word.go:29-31), so a Render-wrapped
            "hot\n  dog" forms the candidate "hot dog" and matches — and the green run then
            spans the wrap indent. Decide and state the rule either way.
          family: unstated-matching-rule
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-26T13:23:49-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: 'Verified: capture splits `rather` and `painstakingly`; test now derives the word and Fatalfs rather than skipping.'
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Rules unified into normative contract 1+2; byte-exact assertions replace the blind properties.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Contract rule 4 specifies error propagation, caller-unit counts, no double-emit; reuses crlf_test.go's shortWriter.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Tasks 2/5/6 compressed to one strategy line each; wordRuns gained FuzzWordRuns.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: storeVocabulary embeds memVocabulary; Add/Has/MaxPhraseWords have one implementation.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Contract rule 2 states a phrase may not span a line break; store.Key claim verified at word.go:29-31.
          round: 2
      blocked: false
content_hash: 5289b495074fce953678250a4192a785d5176aec7a59464ae970cf6b714af38c
---

# Gate ledger — tools#21 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-26T13:20:30-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `fake-capability-mismatch` M3's end-to-end test assumes llmtest.Fake can serve invented streamed text; it cannot
  misapplied() at internal/llm/llmtest/fake.go:188-215 rejects a scripted
  Reply{Text} on a streaming request with a 400, and serveStream at :461-472
  always replays a committed .sse capture — so Task 8 Step 1 cannot choose an
  answer with a known word split across deltas. Name the real route: derive the
  seeded word from a delta boundary in stream-sample.sse (it already splits
  "rather" as "...authority r"/"ather than" and "painstakingly" as
  "(painstak"/"ingly"), computed from the capture rather than hardcoded.
- **PQ-2** [Important] `writer-emission-order` highlightWriter's rules for held-back text and pass-through escapes conflict on order
  "Pass escape sequences through untouched" and "hold back a tail that could
  extend into a phrase" contradict when an escape arrives mid-hold — e.g. deck
  "hot dog" against Render's `\x1b[1;36mhot\x1b[0m` (render.go:26). The plan's
  properties cannot catch the resulting reorder: both the escape-stripped
  equality and the one-call equivalence are blind to it. State the ordering rule
  and assert on full bytes.
- **PQ-3** [Important] `writer-downstream-contract` Plan never says what highlightWriter does on a downstream short write or error
  Task 6's "Write returns len(p) and a nil error even when holding bytes back"
  reads as "always nil error". highlightWriter wraps crlfWriter
  (replraw.go:166), which returns a short count on partial writes and is tested
  for it (crlf_test.go:50, :78). Byte loss here is the plan's own stated worst
  failure. Specify error propagation plus caller-unit mapping, and test against a
  short writer.
- **PQ-4** [Minor] `test-enumeration-in-prose` Tasks 2, 5 and 6 enumerate test cases in prose instead of naming a strategy
  Compress each to one strategy line per risky function. Separately, wordRuns is
  the byte-offset source of truth for both consumers but has no property of its
  own — add one (offsets increasing, non-overlapping, every run slicing cleanly).
- **PQ-5** [Minor] `duplicate-implementation` storeVocabulary and memVocabulary each grow their own set (ARCH-DRY)
  Task 1 Steps 3 and 5 describe a map+maxWords in memVocabulary and "reads Deck()
  once into a set" in storeVocabulary. Say explicitly that storeVocabulary embeds
  memVocabulary so Add/Has/MaxPhraseWords have one implementation.
- **PQ-6** [Minor] `unstated-matching-rule` Plan does not say whether a phrase may match across a newline
  store.Key collapses all whitespace (store/word.go:29-31), so a Render-wrapped
  "hot\n  dog" forms the candidate "hot dog" and matches — and the green run then
  spans the wrap indent. Decide and state the rule either way.

## Round 2 — 2026-08-26T13:23:49-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Verified: capture splits `rather` and `painstakingly`; test now derives the word and Fatalfs rather than skipping.
- PQ-2 — addressed — Rules unified into normative contract 1+2; byte-exact assertions replace the blind properties.
- PQ-3 — addressed — Contract rule 4 specifies error propagation, caller-unit counts, no double-emit; reuses crlf_test.go's shortWriter.
- PQ-4 — addressed — Tasks 2/5/6 compressed to one strategy line each; wordRuns gained FuzzWordRuns.
- PQ-5 — addressed — storeVocabulary embeds memVocabulary; Add/Has/MaxPhraseWords have one implementation.
- PQ-6 — addressed — Contract rule 2 states a phrase may not span a line break; store.Key claim verified at word.go:29-31.

## Open findings

(none — every finding has been disposed)

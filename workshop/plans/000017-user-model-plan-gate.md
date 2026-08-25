---
gate: plan-quality
issue: 17
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-25T16:54:33-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: foldLookups re-implements the existing summariseLookups fold without naming it
          detail: |-
            cmd/define/history_cmd.go:105-144 already folds []store.ReviewEvent into
            per-word rows with the same EventLookedUp-and-Found filter, the same
            store.Key keying, and the same FirstAt/LastAt/Lookups accumulation;
            historyRow (history_cmd.go:78-86) is the plan's wordRow. The plan's DRY
            rationale claims to be creating "the one place" that decides this, and
            that place exists (ARCH-DRY). Name it and either reuse/generalise it or
            state why a second fold is warranted.
          family: reuse-existing-helper
          round: 1
        - id: PQ-2
          severity: Important
          title: D5 sites --reflect where d.deck and d.clock are still nil
          detail: |-
            --llm-check dispatches at main.go:307-309 with os.Getenv/llm.New because it
            needs no directory; d = d.withStore(opt, stderr) is main.go:339. D4 relies
            on d.clock and the store being injectable, so dispatching "before the
            argument count is judged" gets nil for both. The store-needing mode's
            pattern is --forget: arity checked in the switch at main.go:317-329,
            dispatched at main.go:341-343 after withStore. Also state what --reflect
            does when d.deck is nil under DEFINE_NO_CAPTURE (noDeckMessage, main.go:598).
          family: entrypoint-dependency-availability
          round: 1
        - id: PQ-3
          severity: Important
          title: spliceCorrections needs a property over malformed input, not five examples
          detail: |-
            It is a line scanner over human-edited text whose failure mode is silently
            discarding the learner's own writing, and byte-for-byte survival is a
            Done-when row. The five table cases miss the malformed class by
            construction: tilde fences, unterminated fences, indented fences, a marker
            with trailing whitespace, and CRLF (this repo already carries
            cmd/define/crlf.go). One line: a fuzz or property asserting everything from
            the first out-of-fence marker is preserved verbatim.
          family: malformed-input-property
          round: 1
        - id: PQ-4
          severity: Minor
          title: deckEvidence carries per-word lookups from the deck and the total from events
          detail: |-
            The Task 1 test asserts Words[0].Lookups == 4 (from store.Word.Lookups) and
            Lookups == 1 (counted from events), so the counts the prompt carries do not
            sum to the total stated beside them. D6 should say which source is
            authoritative for "how many lookups".
          family: single-source-of-truth
          round: 1
        - id: PQ-5
          severity: Minor
          title: assertGoldenFile is new surface that appears in no entity or integration table
          detail: |-
            llmtest.AssertGolden (llmtest/golden.go:30) takes an llm.Request, not a
            string, so Task 3's helper is genuinely new. The milestone-review judge
            greps the tables against the diff. Add the row, and say it reuses llmtest's
            -update flag rather than registering a second one.
          family: entity-table-completeness
          round: 1
      blocked: true
---

# Gate ledger — tools#17 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-25T16:54:33-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `reuse-existing-helper` foldLookups re-implements the existing summariseLookups fold without naming it
  cmd/define/history_cmd.go:105-144 already folds []store.ReviewEvent into
  per-word rows with the same EventLookedUp-and-Found filter, the same
  store.Key keying, and the same FirstAt/LastAt/Lookups accumulation;
  historyRow (history_cmd.go:78-86) is the plan's wordRow. The plan's DRY
  rationale claims to be creating "the one place" that decides this, and
  that place exists (ARCH-DRY). Name it and either reuse/generalise it or
  state why a second fold is warranted.
- **PQ-2** [Important] `entrypoint-dependency-availability` D5 sites --reflect where d.deck and d.clock are still nil
  --llm-check dispatches at main.go:307-309 with os.Getenv/llm.New because it
  needs no directory; d = d.withStore(opt, stderr) is main.go:339. D4 relies
  on d.clock and the store being injectable, so dispatching "before the
  argument count is judged" gets nil for both. The store-needing mode's
  pattern is --forget: arity checked in the switch at main.go:317-329,
  dispatched at main.go:341-343 after withStore. Also state what --reflect
  does when d.deck is nil under DEFINE_NO_CAPTURE (noDeckMessage, main.go:598).
- **PQ-3** [Important] `malformed-input-property` spliceCorrections needs a property over malformed input, not five examples
  It is a line scanner over human-edited text whose failure mode is silently
  discarding the learner's own writing, and byte-for-byte survival is a
  Done-when row. The five table cases miss the malformed class by
  construction: tilde fences, unterminated fences, indented fences, a marker
  with trailing whitespace, and CRLF (this repo already carries
  cmd/define/crlf.go). One line: a fuzz or property asserting everything from
  the first out-of-fence marker is preserved verbatim.
- **PQ-4** [Minor] `single-source-of-truth` deckEvidence carries per-word lookups from the deck and the total from events
  The Task 1 test asserts Words[0].Lookups == 4 (from store.Word.Lookups) and
  Lookups == 1 (counted from events), so the counts the prompt carries do not
  sum to the total stated beside them. D6 should say which source is
  authoritative for "how many lookups".
- **PQ-5** [Minor] `entity-table-completeness` assertGoldenFile is new surface that appears in no entity or integration table
  llmtest.AssertGolden (llmtest/golden.go:30) takes an llm.Request, not a
  string, so Task 3's helper is genuinely new. The milestone-review judge
  greps the tables against the diff. Add the row, and say it reuses llmtest's
  -update flag rather than registering a second one.

## Open findings

- **PQ-1** [Important] `reuse-existing-helper` foldLookups re-implements the existing summariseLookups fold without naming it
- **PQ-2** [Important] `entrypoint-dependency-availability` D5 sites --reflect where d.deck and d.clock are still nil
- **PQ-3** [Important] `malformed-input-property` spliceCorrections needs a property over malformed input, not five examples
- **PQ-4** [Minor] `single-source-of-truth` deckEvidence carries per-word lookups from the deck and the total from events
- **PQ-5** [Minor] `entity-table-completeness` assertGoldenFile is new surface that appears in no entity or integration table

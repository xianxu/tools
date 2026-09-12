---
gate: plan-quality
issue: 51
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-11T17:18:23-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: '"Only the listed edits" is a Done-when with no list; enumerate the expected line-level diff.'
          detail: Name the headings removed and added, the corrected sentence, the new command-table span, connective lines, the root README block, and where the 13-line usage block splits. Without it the multiset diff cannot be judged.
          family: acceptance-criteria-unverifiable
          round: 1
        - id: PQ-2
          severity: Minor
          title: The stale sentence's "same as --forget, --harvest, --reflect and /history" clause also needs correcting.
          detail: Siblings exit 1 only on a nil deck; on the empty deck a non-deck directory now gets, their codes differ. The plan says only that the exit code is corrected.
          family: unbacked-existing-behavior
          round: 1
        - id: PQ-3
          severity: Minor
          title: Superseded-claims guard uses a 400-char positional window, contradicting "every guard is header- or span-anchored".
          detail: cmd/define/deckasker_test.go:364-376. A move that separates a claim from its qualifying paragraph fails it. Suite covers it, but the Spec's claim is overstated.
          family: unbacked-existing-behavior
          round: 1
        - id: PQ-4
          severity: Minor
          title: Extend TestDocsQuoteTheCommandList over derivedDocs, not a hand-listed pair of paths.
          detail: cmd/define/dictselect_test.go:491 already holds both pages; TestDocsQuoteTheLocaleHelp shows the pattern (ARCH-DRY).
          family: reuse-existing-helper
          round: 1
        - id: PQ-5
          severity: Minor
          title: Plan states no explicit non-goals.
          detail: 'Say outright: no prose rewrite, no code changes beyond two tests, no atlas restructuring, no change to per-command reference content.'
          family: non-goals-unstated
          round: 1
        - id: PQ-6
          severity: Minor
          title: The line-multiset diff has no named command.
          detail: State how it is run so the close review can rerun it.
          family: verification-unspecified
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-11T17:23:51-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Eleven-row expected-diff table plus the exact diff command; line-count and table-length nits (873 not 874, 11-line span) are cosmetic.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Row 6 narrows the sentence to the nil-deck causes, under which the sibling comparison is true again (play_loop.go:30, main.go:1306).
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: Spec sentence unchanged; the guard is claim-anchored with a 400-char window (deckasker_test.go:364-372). Harmless under the verbatim-move non-goal.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Plan iterates derivedDocs (dictselect_test.go:491).
          round: 2
        - id: PQ-5
          disposition: addressed
          round: 2
        - id: PQ-6
          disposition: addressed
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-11T17:29:43-07:00"
      agent: claude
      dispose:
        - id: PQ-3
          disposition: addressed
          note: The universal "already holds" claim is gone; Constraints now states the rule and names keyTableIn as the instance that broke it. The superseded guard's claim-anchored 400-char window (deckasker_test.go:369-373) is still unmentioned but harmless under the verbatim move; the new row-6 phrase must not trip it on the corrected sentence.
          round: 3
      findings:
        - id: PQ-7
          severity: Minor
          title: TestREADMEAnchorsResolve names no slug rule; the heading-to-anchor function is the risky pure input and is unstated.
          detail: '2nd finding in family verification-unspecified. Rule: every mechanical check the plan relies on is named precisely enough for a stranger to rerun it, with its input class. Here that is GitHub''s slug rule (lowercase; keep letters, digits, spaces, hyphens; spaces to hyphens), seeded with this README''s backtick, colon, slash and comma headings. Prevalence: 2 of 3 mechanical checks in this plan started without it.'
          family: verification-unspecified
          round: 3
      blocked: false
content_hash: cd4f48c5610ca66dec455e1987a0939e209ee8c87d1daddbc71ff7ae4e331766
---

# Gate ledger — tools#51 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-11T17:18:23-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `acceptance-criteria-unverifiable` "Only the listed edits" is a Done-when with no list; enumerate the expected line-level diff.
  Name the headings removed and added, the corrected sentence, the new command-table span, connective lines, the root README block, and where the 13-line usage block splits. Without it the multiset diff cannot be judged.
- **PQ-2** [Minor] `unbacked-existing-behavior` The stale sentence's "same as --forget, --harvest, --reflect and /history" clause also needs correcting.
  Siblings exit 1 only on a nil deck; on the empty deck a non-deck directory now gets, their codes differ. The plan says only that the exit code is corrected.
- **PQ-3** [Minor] `unbacked-existing-behavior` Superseded-claims guard uses a 400-char positional window, contradicting "every guard is header- or span-anchored".
  cmd/define/deckasker_test.go:364-376. A move that separates a claim from its qualifying paragraph fails it. Suite covers it, but the Spec's claim is overstated.
- **PQ-4** [Minor] `reuse-existing-helper` Extend TestDocsQuoteTheCommandList over derivedDocs, not a hand-listed pair of paths.
  cmd/define/dictselect_test.go:491 already holds both pages; TestDocsQuoteTheLocaleHelp shows the pattern (ARCH-DRY).
- **PQ-5** [Minor] `non-goals-unstated` Plan states no explicit non-goals.
  Say outright: no prose rewrite, no code changes beyond two tests, no atlas restructuring, no change to per-command reference content.
- **PQ-6** [Minor] `verification-unspecified` The line-multiset diff has no named command.
  State how it is run so the close review can rerun it.

## Round 2 — 2026-09-11T17:23:51-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Eleven-row expected-diff table plus the exact diff command; line-count and table-length nits (873 not 874, 11-line span) are cosmetic.
- PQ-2 — addressed — Row 6 narrows the sentence to the nil-deck causes, under which the sibling comparison is true again (play_loop.go:30, main.go:1306).
- PQ-3 — not-addressed — Spec sentence unchanged; the guard is claim-anchored with a 400-char window (deckasker_test.go:364-372). Harmless under the verbatim-move non-goal.
- PQ-4 — addressed — Plan iterates derivedDocs (dictselect_test.go:491).
- PQ-5 — addressed
- PQ-6 — addressed

## Round 3 — 2026-09-11T17:29:43-07:00 (claude) — passed

### Disposed

- PQ-3 — addressed — The universal "already holds" claim is gone; Constraints now states the rule and names keyTableIn as the instance that broke it. The superseded guard's claim-anchored 400-char window (deckasker_test.go:369-373) is still unmentioned but harmless under the verbatim move; the new row-6 phrase must not trip it on the corrected sentence.

### Raised

- **PQ-7** [Minor] `verification-unspecified` TestREADMEAnchorsResolve names no slug rule; the heading-to-anchor function is the risky pure input and is unstated.
  2nd finding in family verification-unspecified. Rule: every mechanical check the plan relies on is named precisely enough for a stranger to rerun it, with its input class. Here that is GitHub's slug rule (lowercase; keep letters, digits, spaces, hyphens; spaces to hyphens), seeded with this README's backtick, colon, slash and comma headings. Prevalence: 2 of 3 mechanical checks in this plan started without it.

## Open findings

- **PQ-7** [Minor] `verification-unspecified` TestREADMEAnchorsResolve names no slug rule; the heading-to-anchor function is the risky pure input and is unstated.

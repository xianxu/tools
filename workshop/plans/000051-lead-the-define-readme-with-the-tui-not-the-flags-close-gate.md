---
gate: boundary-review
issue: 51
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-11T17:44:27-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: TestREADMEAnchorsResolve names no slug rule; the heading-to-anchor function is the risky pure input and is unstated.
          detail: |-
            2nd finding in family verification-unspecified. Rule: every mechanical check the plan relies on is named precisely enough for a stranger to rerun it, with its input class. Here that is GitHub's slug rule (lowercase; keep letters, digits, spaces, hyphens; spaces to hyphens), seeded with this README's backtick, colon, slash and comma headings. Prevalence: 2 of 3 mechanical checks in this plan started without it.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: verification-unspecified
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-11T17:44:27-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Minor
          title: deckasker_test.go:356-357 — the "persists when\ngiven." entry's comment now dangles after the new superseded-claim entry.
          detail: 'The trailing "// the atlas''s unconditional form" moved off its own line onto the new #51 line. Cosmetic; move it back.'
          family: comment-misattributed
          round: 2
        - id: BR-3
          severity: Minor
          title: TestREADMEAnchorsResolve counts any line starting with "#" as a heading and does not model GitHub's -1 suffix for repeated headings.
          detail: Both make the guard lenient, not strict. Not reachable today (no fenced "#" lines, no duplicate slugs), so note for when the README gains a shell snippet with comments.
          family: guard-input-not-scoped
          round: 2
        - id: BR-4
          severity: Minor
          title: 'play_loop.go:35 comment still says an ordinary directory has no deck; since #50 it has an EMPTY deck.'
          detail: Outside this window. Same stale-claim class the issue swept across doc surfaces, surviving in a code comment.
          family: superseded-claim-in-code-comment
          round: 2
      blocked: false
---

# Gate ledger — tools#51 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-11T17:44:27-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `verification-unspecified` TestREADMEAnchorsResolve names no slug rule; the heading-to-anchor function is the risky pure input and is unstated.
  2nd finding in family verification-unspecified. Rule: every mechanical check the plan relies on is named precisely enough for a stranger to rerun it, with its input class. Here that is GitHub's slug rule (lowercase; keep letters, digits, spaces, hyphens; spaces to hyphens), seeded with this README's backtick, colon, slash and comma headings. Prevalence: 2 of 3 mechanical checks in this plan started without it.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-09-11T17:44:27-07:00 (claude) — passed

### Raised

- **BR-2** [Minor] `comment-misattributed` deckasker_test.go:356-357 — the "persists when\ngiven." entry's comment now dangles after the new superseded-claim entry.
  The trailing "// the atlas's unconditional form" moved off its own line onto the new #51 line. Cosmetic; move it back.
- **BR-3** [Minor] `guard-input-not-scoped` TestREADMEAnchorsResolve counts any line starting with "#" as a heading and does not model GitHub's -1 suffix for repeated headings.
  Both make the guard lenient, not strict. Not reachable today (no fenced "#" lines, no duplicate slugs), so note for when the README gains a shell snippet with comments.
- **BR-4** [Minor] `superseded-claim-in-code-comment` play_loop.go:35 comment still says an ordinary directory has no deck; since #50 it has an EMPTY deck.
  Outside this window. Same stale-claim class the issue swept across doc surfaces, surviving in a code comment.

## Open findings

- **BR-1** [Minor] `verification-unspecified` TestREADMEAnchorsResolve names no slug rule; the heading-to-anchor function is the risky pure input and is unstated.
- **BR-2** [Minor] `comment-misattributed` deckasker_test.go:356-357 — the "persists when\ngiven." entry's comment now dangles after the new superseded-claim entry.
- **BR-3** [Minor] `guard-input-not-scoped` TestREADMEAnchorsResolve counts any line starting with "#" as a heading and does not model GitHub's -1 suffix for repeated headings.
- **BR-4** [Minor] `superseded-claim-in-code-comment` play_loop.go:35 comment still says an ordinary directory has no deck; since #50 it has an EMPTY deck.

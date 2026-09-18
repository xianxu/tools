---
gate: boundary-review
issue: 77
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-18T11:32:06-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: 'Done-when requires the terminal app named; the record says "app not named", with no ## Revisions entry'
          detail: |-
            atlas/define.md:556 records the light check as "(app not named)" while the issue's
            Done-when (000077:36-37) requires "with the terminal app named" and the Spec (:28-29)
            requires app + appearance + exact report line; only the appearance was obtained, both
            Plan boxes are ticked (:41-43), and no ## Revisions entry relaxes the criterion
            (AGENTS.md section 1). Family in this window: (1) atlas/define.md:556 app not named;
            (2) atlas/define.md:556-557 the reply is stated as the criterion's line, not quoted,
            though the Log says the verbatim line was not reported; (3) 000077:41-43 Plan ticked
            against an unmet clause with no Revisions; (4) pre-existing sibling atlas/define.md:551
            the #70 dark record is also "(app not named)", so the atlas's own Re-check rule at :558
            ("record the terminal, appearance and reply") has zero compliant records and this diff
            adds a second non-compliant one. Sweep both records at 551 and 556 in one round: obtain
            the app name and verbatim line, or add the Revisions entry and soften the Re-check rule
            to what a pass/fail report can actually supply. ARCH-PURPOSE: the fidelity of the live
            record is the purpose here, since the pty suite cannot stand in for it.
          family: evidence-weaker-than-stated-criterion
          round: 1
        - id: BR-2
          severity: Minor
          title: Both hand-check records quote the report line without its "scheme " prefix
          detail: |-
            The real output is "scheme light (detected)" / "scheme dark (detected)"
            (cmd/define/scheme.go:297, pinned at cmd/define/pty_conformance_test.go:1545-1546).
            atlas/define.md:551 and :557 both drop the prefix. Harmless in prose, but this
            paragraph's job is verbatim evidence; fix in the same sweep.
          family: evidence-weaker-than-stated-criterion
          round: 1
      recipe: small-diff-review
      blocked: true
---

# Gate ledger — tools#77 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-18T11:32:06-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `evidence-weaker-than-stated-criterion` Done-when requires the terminal app named; the record says "app not named", with no ## Revisions entry
  atlas/define.md:556 records the light check as "(app not named)" while the issue's
  Done-when (000077:36-37) requires "with the terminal app named" and the Spec (:28-29)
  requires app + appearance + exact report line; only the appearance was obtained, both
  Plan boxes are ticked (:41-43), and no ## Revisions entry relaxes the criterion
  (AGENTS.md section 1). Family in this window: (1) atlas/define.md:556 app not named;
  (2) atlas/define.md:556-557 the reply is stated as the criterion's line, not quoted,
  though the Log says the verbatim line was not reported; (3) 000077:41-43 Plan ticked
  against an unmet clause with no Revisions; (4) pre-existing sibling atlas/define.md:551
  the #70 dark record is also "(app not named)", so the atlas's own Re-check rule at :558
  ("record the terminal, appearance and reply") has zero compliant records and this diff
  adds a second non-compliant one. Sweep both records at 551 and 556 in one round: obtain
  the app name and verbatim line, or add the Revisions entry and soften the Re-check rule
  to what a pass/fail report can actually supply. ARCH-PURPOSE: the fidelity of the live
  record is the purpose here, since the pty suite cannot stand in for it.
- **BR-2** [Minor] `evidence-weaker-than-stated-criterion` Both hand-check records quote the report line without its "scheme " prefix
  The real output is "scheme light (detected)" / "scheme dark (detected)"
  (cmd/define/scheme.go:297, pinned at cmd/define/pty_conformance_test.go:1545-1546).
  atlas/define.md:551 and :557 both drop the prefix. Harmless in prose, but this
  paragraph's job is verbatim evidence; fix in the same sweep.

## Open findings

- **BR-1** [Important] `evidence-weaker-than-stated-criterion` Done-when requires the terminal app named; the record says "app not named", with no ## Revisions entry
- **BR-2** [Minor] `evidence-weaker-than-stated-criterion` Both hand-check records quote the report line without its "scheme " prefix

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
    - "n": 2
      timestamp: "2026-09-18T11:39:59-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'atlas/define.md:550 names Terminal.app for both records; Done-when''s three clauses now hold and the #70 sibling was swept in the same round, so no Revisions entry is owed.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: atlas/define.md:551,553 now carry the scheme prefix, matching scheme_cmd.go:18 over describeScheme at scheme.go:293-297.
          round: 2
      findings:
        - id: BR-3
          severity: Important
          title: The atlas hand-quotes the scheme report line in two places with no guard deriving it and no Re-check trigger naming its owning symbol
          detail: |-
            atlas/define.md:551 and :553 quote `scheme dark (detected)` / `scheme light (detected)`,
            strings composed by cmd/define/scheme_cmd.go:18 over describeScheme (cmd/define/scheme.go:293-297).
            No test in cmd/define/doc_sync_test.go derives them, and repo_guard_test.go:698-734
            (currentTruthOnly) confirms this paragraph is current truth, not an exempt RECORD --
            so workshop/targets/derived-restatement.md binds it. BR-2 is the proof the drift is real:
            the prefix was already wrong in both records and was repaired by hand, i.e. the instance
            was fixed and the class was not. Family in this window: (1) :551 dark line unguarded;
            (2) :553 light line unguarded; (3) :557-559 the Re-check trigger list names decodeOSC,
            parseBackgroundColour and backgroundQuery but not describeScheme or scheme_cmd.go's
            format string, so even the sweep branch misses both quotations. Preferred fix, following
            the tree's own pattern (doc_sync_test.go:252, :309): a block-scoped, fail-closed
            TestAtlasQuotesTheSchemeReportItPrints composing "scheme " + describeScheme(st, true)
            for both shades -- compose the wording, not scheme_cmd.go's two-space screen indent.
            Cheap minimum if a guard is judged disproportionate for a docs-only close: add
            describeScheme to the Re-check trigger sentence at :559, one clause. ARCH-DRY: the
            report's wording has two sources of truth and only one of them compiles.
          family: derived-restatement
          round: 2
        - id: BR-4
          severity: Minor
          title: The "/scheme auto also removed the saved file" sentence lost the run it was observed in
          detail: |-
            atlas/define.md:554. In the pre-window text this clause sat inside the dark run's own
            sentence; the rewrite left it floating after a two-run summary, where it reads against
            the preceding "each with nothing saved" and no longer says which check observed it.
            The behaviour is safe -- pinned by TestClearSchemeRemovesOnlyWhatIsOurs
            (cmd/define/store/scheme_test.go:78) -- so this is attribution, not accuracy. Re-attach
            it to the dark run, or drop it from the hand-check record since it is test-derived
            rather than observed. This is the "diff's neighbourhood" class: the only other moved
            claim in the window is the Re-check sentence at :557-559, which survived intact.
          family: claim-detached-from-its-evidence
          round: 2
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

## Round 2 — 2026-09-18T11:39:59-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — atlas/define.md:550 names Terminal.app for both records; Done-when's three clauses now hold and the #70 sibling was swept in the same round, so no Revisions entry is owed.
- BR-2 — addressed — atlas/define.md:551,553 now carry the scheme prefix, matching scheme_cmd.go:18 over describeScheme at scheme.go:293-297.

### Raised

- **BR-3** [Important] `derived-restatement` The atlas hand-quotes the scheme report line in two places with no guard deriving it and no Re-check trigger naming its owning symbol
  atlas/define.md:551 and :553 quote `scheme dark (detected)` / `scheme light (detected)`,
  strings composed by cmd/define/scheme_cmd.go:18 over describeScheme (cmd/define/scheme.go:293-297).
  No test in cmd/define/doc_sync_test.go derives them, and repo_guard_test.go:698-734
  (currentTruthOnly) confirms this paragraph is current truth, not an exempt RECORD --
  so workshop/targets/derived-restatement.md binds it. BR-2 is the proof the drift is real:
  the prefix was already wrong in both records and was repaired by hand, i.e. the instance
  was fixed and the class was not. Family in this window: (1) :551 dark line unguarded;
  (2) :553 light line unguarded; (3) :557-559 the Re-check trigger list names decodeOSC,
  parseBackgroundColour and backgroundQuery but not describeScheme or scheme_cmd.go's
  format string, so even the sweep branch misses both quotations. Preferred fix, following
  the tree's own pattern (doc_sync_test.go:252, :309): a block-scoped, fail-closed
  TestAtlasQuotesTheSchemeReportItPrints composing "scheme " + describeScheme(st, true)
  for both shades -- compose the wording, not scheme_cmd.go's two-space screen indent.
  Cheap minimum if a guard is judged disproportionate for a docs-only close: add
  describeScheme to the Re-check trigger sentence at :559, one clause. ARCH-DRY: the
  report's wording has two sources of truth and only one of them compiles.
- **BR-4** [Minor] `claim-detached-from-its-evidence` The "/scheme auto also removed the saved file" sentence lost the run it was observed in
  atlas/define.md:554. In the pre-window text this clause sat inside the dark run's own
  sentence; the rewrite left it floating after a two-run summary, where it reads against
  the preceding "each with nothing saved" and no longer says which check observed it.
  The behaviour is safe -- pinned by TestClearSchemeRemovesOnlyWhatIsOurs
  (cmd/define/store/scheme_test.go:78) -- so this is attribution, not accuracy. Re-attach
  it to the dark run, or drop it from the hand-check record since it is test-derived
  rather than observed. This is the "diff's neighbourhood" class: the only other moved
  claim in the window is the Re-check sentence at :557-559, which survived intact.

## Open findings

- **BR-3** [Important] `derived-restatement` The atlas hand-quotes the scheme report line in two places with no guard deriving it and no Re-check trigger naming its owning symbol
- **BR-4** [Minor] `claim-detached-from-its-evidence` The "/scheme auto also removed the saved file" sentence lost the run it was observed in

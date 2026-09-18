# Boundary Review — tools#77 (whole-issue close)

| field | value |
|-------|-------|
| issue | 77 — define: confirm light detection in a real light terminal |
| repo | tools |
| issue file | workshop/issues/000077-light-detection-live-check.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2d36c9d5d0b47b6be8b94e0fdb8c1ff71af73b89..b507d9d47e5bcd862d19eae543df77c4b777c183 |
| command | sdlc close --issue 77 |
| reviewer | claude |
| timestamp | 2026-09-18T11:32:06-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

A docs-only boundary: one atlas paragraph replaces the **Owed** sentence with a record of the light-side hand check, and the issue's Plan/Log are filled in. The removal is clean and complete — `grep -rn "Owed"` over the non-history tree returns only #77's own file describing the removal, so no stale "light detection is unproven" claim survives anywhere (atlas, README, targets). The Log is honest in exactly the way that makes this reviewable: it states outright that the terminal app and the verbatim line were *not* reported. That honesty is also what blocks SHIP: the issue's `## Done when` requires the record to name the terminal app, and the `## Spec` requires "the terminal app, its appearance, and the exact report line" — of those three, only the appearance was obtained, and no `## Revisions` entry relaxes the criterion. Ticking the Plan and closing against an unmet, unrevised Done-when clause is the one thing to dispose before crossing.

**1. Strengths**
- `workshop/issues/000077-light-detection-live-check.md:49-54` — the Log names what was *not* reported rather than rounding "#77 verified" up to a full record. This is the difference between a disposable gap and a hidden one.
- The **Owed** removal is total: no parallel restatement left behind (checked atlas/, README.md, workshop/issues/, workshop/targets/).
- `atlas/define.md:555-558` — the new record is appended into the existing *Terminals checked by hand* paragraph beside the #70 dark record, not spun into a parallel section. ARCH-DRY pass: one place answers "what have we seen in a real terminal".
- The **Re-check** trigger sentence (`atlas/define.md:558-560`, naming `decodeOSC` / `parseBackgroundColour` / `backgroundQuery`) is preserved intact through the edit — the hoist did not orphan it from its paragraph.

**2. Critical findings**
None.

**3. Important findings**

- `atlas/define.md:556` (+ `workshop/issues/000077-light-detection-live-check.md:36-37,41-43`) — **the recorded evidence is weaker than the criterion the issue set for itself, and the criterion was not revised.** Done-when: "with the terminal app named"; the atlas record says "(app not named)". Spec asks for app + appearance + exact report line; only the appearance is recorded. Both Plan boxes are ticked and the issue is being closed against a clause that is plainly unmet. Family enumeration in this window and its immediate neighbourhood:
  1. `atlas/define.md:556` — the new #77 record: app not named.
  2. `atlas/define.md:556-557` — the reply is given as the criterion's line ("reading `light (detected)`") rather than as a quoted observation; the Log says the verbatim line was not reported.
  3. `workshop/issues/000077-light-detection-live-check.md:41-43` — Plan item 1 ticked ("Record the operator's light-terminal check … ") while the Done-when clause it serves is unmet; no `## Revisions` entry (AGENTS.md §1 requires one for a mid-stream criterion change).
  4. Pre-existing sibling, same rule: `atlas/define.md:550-551` — the #70 dark record is also "(app not named)". So the atlas's own **Re-check** instruction at line 558 ("record the terminal, appearance and reply") currently has **zero** compliant records; the diff adds a second non-compliant one.
  
  Fix sweep (one round): ask the operator for the app name and the verbatim `/scheme` line, amend **both** records at 551 and 556; **or**, if the operator can't supply them, append a `## Revisions` entry to #77 (timestamp + reason + delta) relaxing Done-when to "verified against the criterion; app unnamed", and soften the **Re-check** sentence to ask only for what a pass/fail report can supply — otherwise the rule keeps generating records that violate it. ARCH-PURPOSE: the purpose here *is* the fidelity of the live-conformance record (it is the one thing the pty suite cannot stand in for), so shipping the pass/fail subset is the easy win, not the purpose.

**4. Minor findings**
- `atlas/define.md:551` and `:557` — the actual report is `scheme light (detected)` / `scheme dark (detected)` (`cmd/define/scheme.go:297`, pinned at `cmd/define/pty_conformance_test.go:1545-1546`); both records quote it without the `scheme ` prefix. Harmless in prose, but this paragraph's job is verbatim evidence — worth fixing in the same sweep as the Important finding.

**5. Test coverage notes**
Docs-only, no executable change, so no regression test is expected or required (per the prose-only rule). The behaviour being documented is already pinned in both modes: `cmd/define/pty_conformance_test.go:1545-1546` covers light **and** dark classification end to end, and `cmd/define/scheme_test.go:282` pins the `(detected)` report suffix — so the "both states exercised" check on the Done-when's light/dark axis passes at the code level. The gap is precisely the one no test can close (a real terminal), which is why the record's evidence standard is the only control and why finding 3.1 is Important rather than Minor.

**6. Architectural notes**
- **ARCH-DRY — pass.** One record, one paragraph; the issue Log and the atlas serve different roles (narrative vs. durable map), not a duplicated source of truth.
- **ARCH-PURE — pass (N/A).** No code in the window.
- **ARCH-PURPOSE — flagged**, see Important finding. Forward note: the *Terminals checked by hand* paragraph is accumulating records in prose. If a third or fourth terminal lands, promote it to a small table (terminal · appearance · date · verbatim reply · issue) — the table shape makes an omitted column visible, which is exactly the failure mode both existing records share.

**7. Plan revision recommendations**
Append to `workshop/issues/000077-light-detection-live-check.md`:

```
## Revisions

### 2026-09-18 — Done-when relaxed: app name and verbatim line not obtainable
Reason: the operator reported the check as "#77 verified" against the Spec's
criterion, without the terminal app's name or the verbatim /scheme line.
Delta: Done-when's "with the terminal app named" is dropped; the atlas records
the check as verified against the criterion, app not named. The atlas's
**Re-check** sentence is likewise softened to ask for appearance and outcome.
```

(Skip this entry only if the app name and exact line are instead recovered from the operator and written into `atlas/define.md:551` and `:556` — that path satisfies the Done-when as written and needs no revision.)

```findings
findings:
  - id: new
    severity: Important
    family: |
      evidence-weaker-than-stated-criterion
    title: |
      Done-when requires the terminal app named; the record says "app not named", with no ## Revisions entry
    detail: |
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
  - id: new
    severity: Minor
    family: |
      evidence-weaker-than-stated-criterion
    title: |
      Both hand-check records quote the report line without its "scheme " prefix
    detail: |
      The real output is "scheme light (detected)" / "scheme dark (detected)"
      (cmd/define/scheme.go:297, pinned at cmd/define/pty_conformance_test.go:1545-1546).
      atlas/define.md:551 and :557 both drop the prefix. Harmless in prose, but this
      paragraph's job is verbatim evidence; fix in the same sweep.
```

---

## Re-review — 2026-09-18T11:39:59-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 77 — define: confirm light detection in a real light terminal |
| repo | tools |
| issue file | workshop/issues/000077-light-detection-live-check.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2d36c9d5d0b47b6be8b94e0fdb8c1ff71af73b89..7dc0dfb3f7a48c5561f031eba223c813561c6a15 |
| command | sdlc close --issue 77 |
| reviewer | claude |
| timestamp | 2026-09-18T11:39:59-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both prior findings are genuinely addressed, and the sweep was done at the class level rather than the instance: `atlas/define.md:549-559` now merges the #70 dark record and the #77 light record into one sentence that names the app (**Terminal.app**) for *both*, gives both appearances, and gives both report lines with the `scheme ` prefix that BR-2 flagged missing — verified against the real format at `cmd/define/scheme_cmd.go:18` (`"  scheme %s\n"`) + `cmd/define/scheme.go:293-297` (`describeScheme` → `"light (detected)"`), which together compose `scheme light (detected)`. The `**Owed**` sentence is gone with no stale restatement anywhere (`grep -rn "Owed"` over the non-history tree hits only #77's own file describing the removal; README carries no `scheme` surface at all, so there is no README gate here). The Done-when's three clauses are all met and the record now labels provenance honestly — dark "(quoted by the operator)", light "verified against #77's criterion" — so no `## Revisions` entry is owed. What stops a bare SHIP is one thing the round fixed as an instance without fixing as a class: BR-2 was a *drift* between a code-owned string and an `atlas/` quotation of it, and nothing now derives or sweeps that quotation, in a repo that has a named target for exactly this and a dozen existing guards implementing it.

**1. Strengths**

- `atlas/define.md:550-553` — the sweep covered the **pre-existing #70 sibling**, not just the record this issue added. BR-1's item (4) asked for both records at the old 551 and 556; the fix merged them so the atlas's own Re-check rule ("record the terminal, appearance and reply") now has two compliant records instead of zero. ARCH-PURPOSE: this is the class, not the named instance.
- `atlas/define.md:551-553` — the asymmetric labelling is the right call. Dark is marked `(quoted by the operator)`; light is marked "verified against #77's criterion". A reader can tell which line is a transcript and which is an attestation, which is exactly what BR-1 item (2) was about. Rounding the light one up to a quote would have been the easy and wrong move.
- `atlas/define.md:557-559` — the **Re-check** trigger sentence survived the rewrite intact and still names `decodeOSC` / `parseBackgroundColour` / `backgroundQuery`; the paragraph restructuring did not orphan it.
- Commits are clean per §12: `#77: atlas: <subject>`, bodies say *why*, `Co-Authored-By` trailer present.
- The behaviour being recorded is pinned in **both** states at the code level: `cmd/define/pty_conformance_test.go:1545-1546` covers light and dark, and `TestDescribeScheme` pins the `(detected)` suffix — both green here (`TestPTYBackgroundDetection` skips in this environment on `bilingualNativeProbe`'s dictionary probe, which is the pre-existing conformance gate, not a window regression).

**2. Critical findings**

None.

**3. Important findings**

- `atlas/define.md:551` and `:553` — **two code-owned strings are now hand-quoted in `atlas/` with nothing deriving or sweeping them.** `scheme dark (detected)` / `scheme light (detected)` are composed by `cmd/define/scheme_cmd.go:18` (`"  scheme %s\n"`) over `cmd/define/scheme.go:293-297` (`describeScheme`). No guard reads them: `cmd/define/doc_sync_test.go` has seventeen doc-derivation tests and none covers the scheme report. The repo's own target `workshop/targets/derived-restatement.md` binds `atlas/` as current truth and says a machine-readable fact belongs in the derived **table**, not the hand-sweep checklist — and `cmd/define/repo_guard_test.go:698-734` (`currentTruthOnly`) confirms this paragraph is *not* an exempt RECORD by the repo's own self-identifying rule (only `## Revisions` / `## Log` / `**closed:**` sections are). BR-2 is the proof the drift is not hypothetical: the prefix was already wrong in **both** records before this round, and was repaired by hand.

  Family enumeration in this window (`derived-restatement`), so the fix sweeps the class:
  1. `atlas/define.md:551` — `scheme dark (detected)`, hand-maintained, no guard.
  2. `atlas/define.md:553` — `scheme light (detected)`, hand-maintained, no guard.
  3. `atlas/define.md:557-559` — even the *sweep* branch misses them: the Re-check trigger list names the detection-path symbols (`decodeOSC`, `parseBackgroundColour`, `backgroundQuery`) but **not** the symbols that own the quoted wording (`describeScheme`, `scheme_cmd.go`'s `"  scheme %s\n"`). A rename of `(detected)` → `(auto)` trips no trigger and no test.

  Fix, preferred form (the target's "derive" branch, pattern already in the tree at `doc_sync_test.go:252` `TestAtlasQuotesTheRawNotationCount` and `:309` `TestDocsQuoteThePronHelp`): add `TestAtlasQuotesTheSchemeReportItPrints` — compose `"scheme " + describeScheme(st, true)` for `sourceDetected` in both shades, locate the *Terminals checked by hand* block by its own heading (scope the guard to the block, per the target's first hard-learned rule), assert both lines present, and fail closed if the block isn't found or fewer than two lines matched. Compose the **wording**, not the screen indent — `scheme_cmd.go` prefixes two spaces that prose should not carry. Minimum acceptable alternative if you judge a guard disproportionate for a docs-only close: extend the Re-check sentence at `:559` to `… or when \`decodeOSC\`, \`parseBackgroundColour\`, \`backgroundQuery\` or \`describeScheme\` changes` — one clause, and it moves these two quotations from "unprotected" to "swept". ARCH-DRY: the report's wording currently has two sources of truth, one of which cannot be compiled.

**4. Minor findings**

- `atlas/define.md:554` — "`/scheme auto` also removed the saved file and its emptied directory" lost its run in the rewrite. It used to sit inside the dark run's clause; it now floats after a two-run summary and reads mildly against the preceding "each with nothing saved". Behaviour itself is safe (pinned by `cmd/define/store/scheme_test.go:78` `TestClearSchemeRemovesOnlyWhatIsOurs`) — this is attribution, not accuracy. Re-attach it ("in the dark run, `/scheme auto` also removed …") or move it out of the hand-check record entirely, since it is test-derived rather than observed.
- `atlas/define.md:550` — "Both in **Terminal.app**" opens with a pronoun whose antecedent arrives one clause later. "Two hand checks, both in **Terminal.app** …" reads in one pass.
- `atlas/define.md:549` — the heading is *Terminals* (plural) over a record of exactly one app in two appearances. Correct going forward; worth a glance if a second app never lands.

**5. Test coverage notes**

Docs-only window, no executable change, so no regression test is owed for the diff itself — the prose-only rule applies and BR-2's repair is verified by inspection against `scheme_cmd.go:18` + `scheme.go:293-297`, not by a wording test. Ran `go test ./cmd/define/... -run 'TestDescribeScheme|TestPTYBackgroundDetection|TestClearSchemeRemovesOnlyWhatIsOurs|TestRawEditorBackgroundReplyRepaints' -count=1` — green; and `-tags conformance -run TestPTYBackgroundDetection -v` — SKIP on this machine via `bilingualNativeProbe`'s dictionary probe (`cmd/define/bilingual_conformance_test.go:16-27`), an environment gap, not a window defect. The one gap no test can close is precisely the live-terminal record, which is why finding 3's protection of that record's fidelity is the relevant control rather than a behavioural test.

**6. Architectural notes**

- **ARCH-DRY — flagged**, see the Important finding: the scheme report wording now exists in `describeScheme`/`scheme_cmd.go` *and* in two atlas quotations, with no shared source. Pass on the other axis: the two hand-check records were consolidated into one sentence rather than left as parallel entries, so "what have we seen in a real terminal" still has one home.
- **ARCH-PURE — pass (N/A).** No code in the window.
- **ARCH-PURPOSE — pass.** The purpose was the fidelity of the live record, and the round delivered the class (both records, both appearances, app named, provenance labelled), not the instance BR-1 pointed at. Forward note unchanged from round 1: if a third terminal lands, promote this paragraph to a table (terminal · appearance · date · reply · quoted? · issue) — a table makes an omitted column visible, which is the failure mode both rounds of this gate found.

**7. Plan revision recommendations**

None. The Done-when is met as written (app named, `/scheme` line recorded, `**Owed**` removed), so the `## Revisions` entry recommended in round 1 is correctly *not* needed — the operator took the "obtain the app name" branch rather than the "relax the criterion" branch. Both Plan boxes at `workshop/issues/000077-light-detection-live-check.md:41-43` are now ticked against satisfied clauses. Note for the closer: the first Log paragraph (`:49-54`) still reads "app not named" as the atlas's state; the correction immediately below it (`:56-61`) supersedes it, which is the right append-only shape for a `## Log` — no edit needed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      atlas/define.md:550 names Terminal.app for both records; Done-when's three clauses now hold and the #70 sibling was swept in the same round, so no Revisions entry is owed.
  - id: BR-2
    disposition: addressed
    note: |
      atlas/define.md:551,553 now carry the scheme prefix, matching scheme_cmd.go:18 over describeScheme at scheme.go:293-297.
findings:
  - id: new
    severity: Important
    family: |
      derived-restatement
    title: |
      The atlas hand-quotes the scheme report line in two places with no guard deriving it and no Re-check trigger naming its owning symbol
    detail: |
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
  - id: new
    severity: Minor
    family: |
      claim-detached-from-its-evidence
    title: |
      The "/scheme auto also removed the saved file" sentence lost the run it was observed in
    detail: |
      atlas/define.md:554. In the pre-window text this clause sat inside the dark run's own
      sentence; the rewrite left it floating after a two-run summary, where it reads against
      the preceding "each with nothing saved" and no longer says which check observed it.
      The behaviour is safe -- pinned by TestClearSchemeRemovesOnlyWhatIsOurs
      (cmd/define/store/scheme_test.go:78) -- so this is attribution, not accuracy. Re-attach
      it to the dark run, or drop it from the hand-check record since it is test-derived
      rather than observed. This is the "diff's neighbourhood" class: the only other moved
      claim in the window is the Re-check sentence at :557-559, which survived intact.
```

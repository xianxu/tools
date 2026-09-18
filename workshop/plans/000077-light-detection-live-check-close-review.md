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

---

## Re-review — 2026-09-18T11:49:01-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 77 — define: confirm light detection in a real light terminal |
| repo | tools |
| issue file | workshop/issues/000077-light-detection-live-check.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2d36c9d5d0b47b6be8b94e0fdb8c1ff71af73b89..09a55e8f9da6f2061c1ba9469c15be7a309a60ac |
| command | sdlc close --issue 77 |
| reviewer | claude |
| timestamp | 2026-09-18T11:49:01-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Both prior findings are fixed, and the fix is proven by a test that fails without it. The `/scheme` quotations at `atlas/define.md:551` and `:554` now sit in marked spans. `TestAtlasQuotesTheSchemeReportItPrints` (`cmd/define/doc_sync_test.go:871`) builds each one from `describeScheme`. I ran mutations against it: it goes red when the atlas drops the `scheme ` prefix (the BR-2 drift), red when `describeScheme`'s wording changes, and red when the markers are deleted, so it fails closed. The Re-check trigger now names `describeScheme`, and the "removed the saved file" clause is back on the dark run that observed it. The Done-when is met: Terminal.app is named for a LIGHT appearance, the line is recorded, and `Owed` is gone from the atlas. Nothing blocks SHIP. Two Minor findings remain. First, the guard's `scheme ` prefix is still a hard-coded string, and describeScheme's wording is still hand-quoted elsewhere in the tree. Second, the test's failure message tells the reader to edit the quotation instead of repeating the hand check.

**1. Strengths**
- The guard copies the tree's own marked-span pattern (`TestAtlasQuotesTheRawNotationCount`, `TestDocsQuoteThePronHelp`), as the `derived-restatement` target asks. It builds the words from `describeScheme(schemeState{}.withDetected(sc), true)` and leaves out the two-space screen indent.
- It fails closed: deleting the markers turns it red, so it can't pass by checking nothing.
- The record is honest about where each line came from. Dark is "(quoted by the operator)"; light is "verified against #77's criterion". Neither is presented as a verbatim quote it isn't.
- `describeScheme` joined the Re-check trigger (`atlas/define.md:559-562`), which closes BR-3 item (3).

**2. Critical:** none.

**3. Important:** none.

**4. Minor**
- **`derived-restatement`, 2nd finding in this family.** The printed line is `"  scheme %s\n"` in `runScheme` (`scheme_cmd.go:18`, `:31`) wrapped around `describeScheme`. The guard types `scheme ` itself. I changed both format strings to `"  colour scheme %s\n"` and the guard stayed green. That makes two claims overstated: the test comment calling `describeScheme` "the report's one source" (`doc_sync_test.go:866-867`), and the atlas saying "Both quotations derive from `describeScheme`" (`:555-556`). The same wording is also quoted by hand elsewhere in the tree (details in the findings block). Stating the rule rather than fixing one more site: every doc quotation of `/scheme`'s output, whether the whole line or just the `(source)` part, sits in a marked span built from the one function that prints it, and there is exactly one such function.
- **`claim-detached-from-its-evidence`, 2nd finding in this family.** The guard ties a dated observation to current code. Its failure text, "describeScheme owns this line; the atlas consumes it" (`:880`), invites someone to re-quote the 2026-09-18 Terminal.app record after a wording change. The record would then claim a line nobody saw. `currentTruthOnly` warns against exactly this: "revising it to match today is the lie". Rule: an observation record changes only with a new observation, so a guard over one must name repeating the observation as the fix.
- The guard uses `strings.Contains`. Unlike its sibling raw-notation test (`doc_sync_test.go:283-289`), it doesn't check that the number of markers equals the number of matching spans, so a second, stale `scheme-report:*` span would pass.

**5. Test coverage**
- The new guard is a pure read of the doc, and its reachability is shown by the mutations above.
- `go test ./cmd/define/...`: everything passes except three pty tests (`TestLanguagePromptStartup`, `TestLanguageTintInvocation`, `TestSavedSchemeGovernsALookup`). They fail because `pty.Open` returns "operation not permitted" in this review environment. The window doesn't touch them.
- Process note: my first mutation run couldn't create its scratch directory and ran in the real checkout. Each edit was reverted with `git checkout`; the tree is clean and HEAD is still `09a55e8`.

**6. Architecture**
- **ARCH-DRY: flagged (Minor).** The `scheme ` prefix now has three copies: two in `runScheme` and one in the guard. The shared helper they should become is a `schemeReport(st, fullScreen)` function.
- **ARCH-PURE: passes.** `describeScheme` is pure, and the guard only reads a file, the same doc-sync seam the other guards use.
- **ARCH-PURPOSE: passes on the issue's purpose.** The light check is recorded, the app is named, and `Owed` is removed. The sweep of the tree still finds three hand-quoted uses of `describeScheme`'s wording outside the window, listed under the first Minor finding and in the findings block.

**7. Plan revisions:** none needed. The issue's `## Log` matches the code.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      Guard doc_sync_test.go:871 composes both spans from describeScheme; mutation-verified red on atlas drift, wording change, marker deletion; describeScheme joins Re-check (atlas:559-562). Prefix residual raised new.
  - id: BR-4
    disposition: addressed
    note: |
      atlas/define.md:552 re-attaches the clause to the dark run ("that run's /scheme auto also removed..."), matching the pre-window text where the same run observed it.
findings:
  - id: new
    severity: Minor
    family: derived-restatement
    title: |
      The guard types the "scheme " prefix itself, and describeScheme's wording is still hand-quoted elsewhere in the tree
    detail: |
      2nd in family. Rule: every doc quotation of /scheme output (whole line or source suffix) sits in a marked span composed from the ONE function that prints it.
      Instances: (1) doc_sync_test.go:877 types "scheme " itself while runScheme owns it at scheme_cmd.go:18 and :31. Changing both to "colour scheme" left the guard green (verified), so the claims at doc_sync_test.go:866-867 ("the report's one source") and atlas/define.md:555-556 overstate.
      (2) atlas/define.md:1051 quotes (session only; not saved) with no guard.
      (3) atlas/define.md:1052-1053 lists describeScheme's sources as saved, -scheme flag, session only, or the default. It omits detected, so the list is incomplete today.
      (4) cmd/define/README.md:364 quotes (detected) with no guard.
      (5) The guard lacks the sibling test's check that the number of markers equals the number of matching spans (doc_sync_test.go:283-289).
      Fix: add schemeReport(st, fullScreen) = "scheme " + describeScheme(...), called by both runScheme sites and the guard. Put each quoted suffix in a derived span. Correct the source list.
  - id: new
    severity: Minor
    family: claim-detached-from-its-evidence
    title: |
      The guard's failure message tells the reader to re-quote a dated observation instead of repeating the hand check
    detail: |
      2nd in family. Rule: an observation record changes only with a new observation, so a guard over one names repeating the observation as the fix.
      doc_sync_test.go:880 says "describeScheme owns this line; the atlas consumes it", and the comment at :868-870 frames the fix as re-quoting. After a wording change, following that rewrites the 2026-09-18 Terminal.app record into a line never observed. currentTruthOnly calls this "the lie".
      The atlas Re-check rule (atlas/define.md:559-562) already gives the right action; the test should point at it. Fix: reword the failure message and comment to say "re-run the hand check in a real terminal and record the new date, terminal and line".
```

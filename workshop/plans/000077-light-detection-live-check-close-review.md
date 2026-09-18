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

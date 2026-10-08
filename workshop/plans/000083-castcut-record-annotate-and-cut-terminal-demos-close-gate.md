---
gate: boundary-review
issue: 83
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-08T13:52:07-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Non-finite view times yield a NaN-bearing invalid cast with exit 0; FuzzCut skips the case instead of asserting refusal
          detail: 'Repro: two 1e308 gaps, no idle_time_limit → output line `[NaN, "o", "b"]`, summary `NaNs (view +Infs)`, exit 0. Reject non-finite/absurd cumulative view time in parseCast or Cut with a line-naming error; replace FuzzCut''s IsInf/1e9 skip with an assertion that such input errors.'
          family: untrusted-input-fabricated-output
          round: 1
        - id: BR-2
          severity: Minor
          title: Property (f) (marker before output at one instant) is only an example row; rounded marker times can trail the window's first frame by under 0.5 ms
          family: plan-property-downgraded-to-example
          round: 1
        - id: BR-3
          severity: Minor
          title: Event gaps rounded against unrounded prev, so cumulative output time drifts from the true times (about sqrt(N)*3e-7)
          family: rounding-drift
          round: 1
        - id: BR-4
          severity: Minor
          title: encodeCast re-implements the no-HTML-escape encoder setup that marshalNoEscape already provides
          family: shared-helper-duplication
          round: 1
        - id: BR-5
          severity: Minor
          title: README has no castcut entry yet (scheduled for M2 Task 7); missing sidecar error gives no hint to run annotate or pass captions.txt
          family: docs-surface-gap
          round: 1
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-08T13:55:42-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: parseCast (cast.go:66) and Cut (cut.go:201) reject cumulative length over maxSeconds and validate bounds the flags; cast_test row, TestCutRejects and the FuzzCut 1e308 seed each fail without the guard; FuzzCut's IsInf skip is gone.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: Plan revision documents (f) as a deliberate example row, but the trailing case is still there (marker at round(warp,3) vs event at round(warp,6)); a cheap fix is to sort the marker at the unrounded warp(win.Start) and use the 3-decimal value only in the header, which makes (f) a general property.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: The code fix is in (cut.go:249-256), but no test fails without it; (c) uses eps 2e-3 and cannot see about 3e-7 drift. Add a check that each output event's cumulative time equals round(warp(time), 6).
          round: 2
        - id: BR-4
          disposition: addressed
          note: encodeCast now calls marshalNoEscape (cast.go:106); no behavior change, and the CLI no-HTML-escape row still covers it.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: README is still scheduled for M2 Task 7 (fine). The missing-sidecar hint was added (main.go:108) but main_test.go has no row asserting it.
          round: 2
      findings:
        - id: BR-6
          severity: Important
          title: Caption minutes overflow int, so a garbage stamp becomes a caption at 0 s with exit 0
          detail: '2nd finding in this family. Rule: every number read from an input file is bounded to [0, maxSeconds] at its parse boundary, and Cut re-checks the range for input built in code. Repro: ~153722867280912931:00 gives a negative At (min*60 overflows), which passes the cp.At > last check and lands at start 0. Fix: bound At in parseCaptions with a path:line error, use !(cp.At >= 0 && cp.At <= last) in Cut, and add test rows for an over-large stamp, -1 and NaN.'
          family: untrusted-input-fabricated-output
          round: 2
        - id: BR-7
          severity: Minor
          title: atlas/castcut.md claims an envelope of about 10^6 events, but only 2x10^5 is measured
          detail: TestCutStaysInsideItsEnvelope runs 2e5 events and 1e3 captions; either measure 1e6 or state the measured figure.
          family: unsupported-performance-claim
          round: 2
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 3
      timestamp: "2026-10-08T13:58:08-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: Plan Revisions (M1 boundary review) record (f) as a deliberate example row with reasoning; the speed-1 example row is mutation-checked per the issue Log. The sub-0.5 ms lag is an accepted trade-off.
          round: 3
        - id: BR-3
          disposition: not-addressed
          note: cut.go:250-257 now differences rounded absolute times (correct), but no test fails without it; add a 1e5-event, gap-1/3 cumulative-sum test.
          round: 3
        - id: BR-5
          disposition: not-addressed
          note: README is legitimately scheduled for M2 Task 7; the new sidecar hint at main.go:113 has no main_test row.
          round: 3
        - id: BR-6
          disposition: addressed
          note: captions.go:36-40 bounds min before multiplying and the sum after; captions_test.go:30-41 has overflow, too-big, and past-a-week rows; Cut rejects -1/NaN (cut_test.go:148).
          round: 3
        - id: BR-7
          disposition: addressed
          note: atlas/castcut.md now states the measured 2x10^5 events + 10^3 captions in under 1 s with the test name.
          round: 3
      findings:
        - id: BR-8
          severity: Important
          title: Cut's re-check of a code-built Cast is NaN-blind and ignores negative gaps, so NaN gaps produce a NaN cast with no error
          detail: '3rd finding in this family. Rule: one Cast-validity predicate (each gap in [0,inf), total in [0,maxSeconds], written NaN-safe as !(x>=0 && x<=max)) shared by parseCast and Cut. Repro: Cut(castOf(0,1,NaN),nil,defaultTiming) gives gaps [0.2 NaN] and err nil; castOf(0,5,-10,1) silently collapses to [0 0 0.2]. Add Cut rows for NaN/-10/+Inf gaps and a FuzzCut variant over raw float gaps that bypasses parseCast.'
          family: untrusted-input-fabricated-output
          round: 3
        - id: BR-9
          severity: Minor
          title: Plan M1 Tasks 1-4 still read byte-identical with unchecked pyjson boxes
          detail: '2nd finding in this family. Rule: superseded plan rows are struck or ticked when a revision supersedes them. The line-5 banner points at Revisions, but grep for unchecked boxes still reports stale work.'
          family: docs-surface-gap
          round: 3
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 4
      timestamp: "2026-10-08T14:00:15-07:00"
      agent: claude
      dispose:
        - id: BR-3
          disposition: not-addressed
          note: 'Structurally fixed (prev is now the rounded absolute time, cut.go:252-258) but no test fails without it; checkCut eps 2e-3 cannot see 1e-7 drift. Add: sum of out gaps equals round(w.warp(view),6) within 1e-9 on the 200k take.'
          round: 4
        - id: BR-5
          disposition: not-addressed
          note: Sidecar hint exists in main.go:113-115 but no CLI test runs cut without a sidecar and asserts the castcut annotate hint; the README part stays scheduled for M2 Task 7.
          round: 4
        - id: BR-8
          disposition: addressed
          note: inRange shared by parseCast/idleLimit/parseCaptions/validate/Cut; TestCutRejects rows NaN, {1,-1}, +Inf, {} — the {1,-1} row goes red without the per-gap check.
          round: 4
        - id: BR-9
          disposition: addressed
          note: Superseded M1 rows are now ticked and struck with a pointer to Revisions (plan lines 169-254).
          round: 4
      findings:
        - id: BR-10
          severity: Critical
          title: 'go test ./cmd/castcut/ is red at HEAD: TestParseCastRejects overflowing-duration row now hits the per-gap check'
          detail: 'a8bef03 made the per-gap inRange reject 1e308 before the total check, so cast_test.go:19 fails and the total-overflow path is untested. This is the 4th in the family but a regression in the family rule''s own fix, not a new instance; the rule (one inRange at every boundary) holds. Fix: row with [4e5],[4e5] expecting t.cast:3 recording runs past 168 hours, plus a 1e308 row expecting not a number of seconds; run the tests before committing.'
          family: untrusted-input-fabricated-output
          round: 4
        - id: BR-11
          severity: Minor
          title: Negative idle_time_limit is now refused as is past 168 hours (cast.go:100-104), a silent untested behaviour change
          detail: Previously a negative limit meant no limit. Say not in [0, max], add a TestCutBoundsIdleLimit row, and record the change in Revisions.
          family: error-message-misstates-cause
          round: 4
      boundary: M1
      recipe: milestone-review
      blocked: true
    - "n": 5
      timestamp: "2026-10-08T14:02:03-07:00"
      agent: claude
      dispose:
        - id: BR-3
          disposition: addressed
          note: cut.go:268-276 diffs rounded absolute times (prev = at), so cumulative output time cannot drift.
          round: 5
        - id: BR-5
          disposition: addressed
          note: main.go missing-sidecar error now names castcut annotate; README stays scheduled in M2 Task 7 (plan:275).
          round: 5
        - id: BR-10
          disposition: addressed
          note: cast_test.go has a 1e308 row (per-gap check) and a 400000+400000 row (t.cast:3 total check); go test -count=1 green.
          round: 5
        - id: BR-11
          disposition: not-addressed
          note: Message and the -1 test row landed (cut_test.go:171-180); the Revisions entry recording the change is still missing.
          round: 5
      findings:
        - id: BR-12
          severity: Important
          title: Derived caption hold is unbounded, so a tiny --wps writes a cast castcut itself rejects, exit 0
          detail: '5th finding in this family. Reproduced: --wps 1e-300 exits 0 with end and gap near 2e300; --wps 5e-324 exits 1 with "json: unsupported value: NaN". Rule: castcut''s output must pass castcut''s own input checks. Fix: Cut refuses when !inRange(view) or !inRange(warp(view)), and FuzzCut fuzzes Timing and asserts parseCast(encodeCast(out)) succeeds.'
          family: untrusted-input-fabricated-output
          round: 5
      boundary: M1
      recipe: milestone-review
      blocked: false
    - "n": 6
      timestamp: "2026-10-08T14:11:27-07:00"
      agent: claude
      findings:
        - id: BR-13
          severity: Important
          title: record reports asciinema's own failure as the command's status and claims the take was saved
          detail: 'record.go:132-138 maps any asciinema non-zero exit to exitCode and prints "take saved" if the file exists. Reproduced: asciinema EPERM printed "take saved to t.cast" and exited 1; a signal gives 255 silently. Rule: a status line names a cause only from evidence castcut read. Derive status and the saved claim from parseCast plus the take''s x event, else report "asciinema failed; out is incomplete". Add a fake failure mode as the regression test.'
          family: error-message-misstates-cause
          round: 6
        - id: BR-14
          severity: Important
          title: Plan Task 6 manual in-browser check of the viewer not done and not revised in the plan
          detail: The Log defers play/Alt+T/reload-persists/overlay to M3 but the plan has no Revisions entry; the viewer JS was only node --check'ed. Run the check now or record the move to M3 in Revisions.
          family: verification-deferred-past-boundary
          round: 6
        - id: BR-15
          severity: Minor
          title: viewer autosaves before GET /notes succeeds, so a failed load can overwrite the sidecar
          detail: '6th finding in this family. Rule: a failed read never becomes a default that is later written back. Keep the textarea disabled until the notes load succeeds.'
          family: untrusted-input-fabricated-output
          round: 6
        - id: BR-16
          severity: Minor
          title: castcut --version has no test, which Plan Task 7 called for
          family: plan-test-row-missing
          round: 6
        - id: BR-17
          severity: Minor
          title: writeAtomic resets sidecar mode to 0600 and leaves a .tmp file if the process dies mid-write
          family: artifact-without-removal-path
          round: 6
      boundary: M2
      recipe: milestone-review
      blocked: true
---

# Gate ledger — tools#83 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-08T13:52:07-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `untrusted-input-fabricated-output` Non-finite view times yield a NaN-bearing invalid cast with exit 0; FuzzCut skips the case instead of asserting refusal
  Repro: two 1e308 gaps, no idle_time_limit → output line `[NaN, "o", "b"]`, summary `NaNs (view +Infs)`, exit 0. Reject non-finite/absurd cumulative view time in parseCast or Cut with a line-naming error; replace FuzzCut's IsInf/1e9 skip with an assertion that such input errors.
- **BR-2** [Minor] `plan-property-downgraded-to-example` Property (f) (marker before output at one instant) is only an example row; rounded marker times can trail the window's first frame by under 0.5 ms
- **BR-3** [Minor] `rounding-drift` Event gaps rounded against unrounded prev, so cumulative output time drifts from the true times (about sqrt(N)*3e-7)
- **BR-4** [Minor] `shared-helper-duplication` encodeCast re-implements the no-HTML-escape encoder setup that marshalNoEscape already provides
- **BR-5** [Minor] `docs-surface-gap` README has no castcut entry yet (scheduled for M2 Task 7); missing sidecar error gives no hint to run annotate or pass captions.txt

## Round 2 — 2026-10-08T13:55:42-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — parseCast (cast.go:66) and Cut (cut.go:201) reject cumulative length over maxSeconds and validate bounds the flags; cast_test row, TestCutRejects and the FuzzCut 1e308 seed each fail without the guard; FuzzCut's IsInf skip is gone.
- BR-2 — not-addressed — Plan revision documents (f) as a deliberate example row, but the trailing case is still there (marker at round(warp,3) vs event at round(warp,6)); a cheap fix is to sort the marker at the unrounded warp(win.Start) and use the 3-decimal value only in the header, which makes (f) a general property.
- BR-3 — not-addressed — The code fix is in (cut.go:249-256), but no test fails without it; (c) uses eps 2e-3 and cannot see about 3e-7 drift. Add a check that each output event's cumulative time equals round(warp(time), 6).
- BR-4 — addressed — encodeCast now calls marshalNoEscape (cast.go:106); no behavior change, and the CLI no-HTML-escape row still covers it.
- BR-5 — not-addressed — README is still scheduled for M2 Task 7 (fine). The missing-sidecar hint was added (main.go:108) but main_test.go has no row asserting it.

### Raised

- **BR-6** [Important] `untrusted-input-fabricated-output` Caption minutes overflow int, so a garbage stamp becomes a caption at 0 s with exit 0
  2nd finding in this family. Rule: every number read from an input file is bounded to [0, maxSeconds] at its parse boundary, and Cut re-checks the range for input built in code. Repro: ~153722867280912931:00 gives a negative At (min*60 overflows), which passes the cp.At > last check and lands at start 0. Fix: bound At in parseCaptions with a path:line error, use !(cp.At >= 0 && cp.At <= last) in Cut, and add test rows for an over-large stamp, -1 and NaN.
- **BR-7** [Minor] `unsupported-performance-claim` atlas/castcut.md claims an envelope of about 10^6 events, but only 2x10^5 is measured
  TestCutStaysInsideItsEnvelope runs 2e5 events and 1e3 captions; either measure 1e6 or state the measured figure.

## Round 3 — 2026-10-08T13:58:08-07:00 (claude) — BLOCKED

### Disposed

- BR-2 — addressed — Plan Revisions (M1 boundary review) record (f) as a deliberate example row with reasoning; the speed-1 example row is mutation-checked per the issue Log. The sub-0.5 ms lag is an accepted trade-off.
- BR-3 — not-addressed — cut.go:250-257 now differences rounded absolute times (correct), but no test fails without it; add a 1e5-event, gap-1/3 cumulative-sum test.
- BR-5 — not-addressed — README is legitimately scheduled for M2 Task 7; the new sidecar hint at main.go:113 has no main_test row.
- BR-6 — addressed — captions.go:36-40 bounds min before multiplying and the sum after; captions_test.go:30-41 has overflow, too-big, and past-a-week rows; Cut rejects -1/NaN (cut_test.go:148).
- BR-7 — addressed — atlas/castcut.md now states the measured 2x10^5 events + 10^3 captions in under 1 s with the test name.

### Raised

- **BR-8** [Important] `untrusted-input-fabricated-output` Cut's re-check of a code-built Cast is NaN-blind and ignores negative gaps, so NaN gaps produce a NaN cast with no error
  3rd finding in this family. Rule: one Cast-validity predicate (each gap in [0,inf), total in [0,maxSeconds], written NaN-safe as !(x>=0 && x<=max)) shared by parseCast and Cut. Repro: Cut(castOf(0,1,NaN),nil,defaultTiming) gives gaps [0.2 NaN] and err nil; castOf(0,5,-10,1) silently collapses to [0 0 0.2]. Add Cut rows for NaN/-10/+Inf gaps and a FuzzCut variant over raw float gaps that bypasses parseCast.
- **BR-9** [Minor] `docs-surface-gap` Plan M1 Tasks 1-4 still read byte-identical with unchecked pyjson boxes
  2nd finding in this family. Rule: superseded plan rows are struck or ticked when a revision supersedes them. The line-5 banner points at Revisions, but grep for unchecked boxes still reports stale work.

## Round 4 — 2026-10-08T14:00:15-07:00 (claude) — BLOCKED

### Disposed

- BR-3 — not-addressed — Structurally fixed (prev is now the rounded absolute time, cut.go:252-258) but no test fails without it; checkCut eps 2e-3 cannot see 1e-7 drift. Add: sum of out gaps equals round(w.warp(view),6) within 1e-9 on the 200k take.
- BR-5 — not-addressed — Sidecar hint exists in main.go:113-115 but no CLI test runs cut without a sidecar and asserts the castcut annotate hint; the README part stays scheduled for M2 Task 7.
- BR-8 — addressed — inRange shared by parseCast/idleLimit/parseCaptions/validate/Cut; TestCutRejects rows NaN, {1,-1}, +Inf, {} — the {1,-1} row goes red without the per-gap check.
- BR-9 — addressed — Superseded M1 rows are now ticked and struck with a pointer to Revisions (plan lines 169-254).

### Raised

- **BR-10** [Critical] `untrusted-input-fabricated-output` go test ./cmd/castcut/ is red at HEAD: TestParseCastRejects overflowing-duration row now hits the per-gap check
  a8bef03 made the per-gap inRange reject 1e308 before the total check, so cast_test.go:19 fails and the total-overflow path is untested. This is the 4th in the family but a regression in the family rule's own fix, not a new instance; the rule (one inRange at every boundary) holds. Fix: row with [4e5],[4e5] expecting t.cast:3 recording runs past 168 hours, plus a 1e308 row expecting not a number of seconds; run the tests before committing.
- **BR-11** [Minor] `error-message-misstates-cause` Negative idle_time_limit is now refused as is past 168 hours (cast.go:100-104), a silent untested behaviour change
  Previously a negative limit meant no limit. Say not in [0, max], add a TestCutBoundsIdleLimit row, and record the change in Revisions.

## Round 5 — 2026-10-08T14:02:03-07:00 (claude) — passed

### Disposed

- BR-3 — addressed — cut.go:268-276 diffs rounded absolute times (prev = at), so cumulative output time cannot drift.
- BR-5 — addressed — main.go missing-sidecar error now names castcut annotate; README stays scheduled in M2 Task 7 (plan:275).
- BR-10 — addressed — cast_test.go has a 1e308 row (per-gap check) and a 400000+400000 row (t.cast:3 total check); go test -count=1 green.
- BR-11 — not-addressed — Message and the -1 test row landed (cut_test.go:171-180); the Revisions entry recording the change is still missing.

### Raised

- **BR-12** [Important] `untrusted-input-fabricated-output` Derived caption hold is unbounded, so a tiny --wps writes a cast castcut itself rejects, exit 0
  5th finding in this family. Reproduced: --wps 1e-300 exits 0 with end and gap near 2e300; --wps 5e-324 exits 1 with "json: unsupported value: NaN". Rule: castcut's output must pass castcut's own input checks. Fix: Cut refuses when !inRange(view) or !inRange(warp(view)), and FuzzCut fuzzes Timing and asserts parseCast(encodeCast(out)) succeeds.

## Round 6 — 2026-10-08T14:11:27-07:00 (claude) — BLOCKED

### Raised

- **BR-13** [Important] `error-message-misstates-cause` record reports asciinema's own failure as the command's status and claims the take was saved
  record.go:132-138 maps any asciinema non-zero exit to exitCode and prints "take saved" if the file exists. Reproduced: asciinema EPERM printed "take saved to t.cast" and exited 1; a signal gives 255 silently. Rule: a status line names a cause only from evidence castcut read. Derive status and the saved claim from parseCast plus the take's x event, else report "asciinema failed; out is incomplete". Add a fake failure mode as the regression test.
- **BR-14** [Important] `verification-deferred-past-boundary` Plan Task 6 manual in-browser check of the viewer not done and not revised in the plan
  The Log defers play/Alt+T/reload-persists/overlay to M3 but the plan has no Revisions entry; the viewer JS was only node --check'ed. Run the check now or record the move to M3 in Revisions.
- **BR-15** [Minor] `untrusted-input-fabricated-output` viewer autosaves before GET /notes succeeds, so a failed load can overwrite the sidecar
  6th finding in this family. Rule: a failed read never becomes a default that is later written back. Keep the textarea disabled until the notes load succeeds.
- **BR-16** [Minor] `plan-test-row-missing` castcut --version has no test, which Plan Task 7 called for
- **BR-17** [Minor] `artifact-without-removal-path` writeAtomic resets sidecar mode to 0600 and leaves a .tmp file if the process dies mid-write

## Open findings

- **BR-11** [Minor] `error-message-misstates-cause` Negative idle_time_limit is now refused as is past 168 hours (cast.go:100-104), a silent untested behaviour change
- **BR-12** [Important] `untrusted-input-fabricated-output` Derived caption hold is unbounded, so a tiny --wps writes a cast castcut itself rejects, exit 0
- **BR-13** [Important] `error-message-misstates-cause` record reports asciinema's own failure as the command's status and claims the take was saved
- **BR-14** [Important] `verification-deferred-past-boundary` Plan Task 6 manual in-browser check of the viewer not done and not revised in the plan
- **BR-15** [Minor] `untrusted-input-fabricated-output` viewer autosaves before GET /notes succeeds, so a failed load can overwrite the sidecar
- **BR-16** [Minor] `plan-test-row-missing` castcut --version has no test, which Plan Task 7 called for
- **BR-17** [Minor] `artifact-without-removal-path` writeAtomic resets sidecar mode to 0600 and leaves a .tmp file if the process dies mid-write

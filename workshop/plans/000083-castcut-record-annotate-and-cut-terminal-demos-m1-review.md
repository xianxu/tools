# Boundary Review — tools#83 (milestone M1)

| field | value |
|-------|-------|
| issue | 83 — castcut: record, annotate and cut terminal demos |
| repo | tools |
| issue file | workshop/issues/000083-castcut-record-annotate-and-cut-terminal-demos.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 4c1444f900957224cb3a10cc5bfeb9542988f2cf..3de67d191f18a7a99728f9db8f29309b1ed14dd5 |
| command | sdlc milestone-close --issue 83 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-08T13:52:07-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M1 delivers what the revised plan says it should. `castcut cut` is a pure timing model (`planWindows` → `buildSegments` → `warper` → `Cut`) behind a thin `run` shell, with Python-free stdlib JSON. Property tests cover order preservation, monotone output, real-time windows, the idle bound and disjoint captions (properties (a)–(e)), and they run over 500 random takes, a 2×10⁵-event envelope run and a naive-`warp` reference. There are fuzz targets for all three parsers. `go test`, `go vet` and `GOOS=linux go build` are all green. One defect should be fixed before crossing: the plan says `FuzzCut` must produce either an error or a valid cast, but the fuzz test skips non-finite gaps instead, and those gaps produce an invalid cast. I reproduced it: two `1e308` gaps and no `idle_time_limit` make `castcut cut` exit 0 after writing `[NaN, "o", "b"]` and printing `NaNs (view +Infs)`. Everything else is minor or belongs to M2.

**1. Strengths**
- **Pure core, thin IO shell.** `Cut` (`cmd/castcut/cut.go:163`) is pure and doesn't change its input (`TestCutDoesNotMutateItsInput`). `runCut` only reads files, parses, calls `Cut`, encodes and writes, and nothing is written on error (tested).
- **Cumulative-output warper.** The warper (`cut.go:127`) uses cumulative output time plus binary search, and a naive reference implementation that lives only in the tests pins it.
- **PQ-1 (hold the last frame).** When a caption's window outlasts the take, a trailing empty `o` event holds the final frame. `TestCutHoldsTheLastFrameForALateCaption` covers it, and property (a) tolerates exactly one such hold event.
- **CLI test checks the embed contract.** `main_test.go` checks the header captions that `CastEmbed.astro` reads, that the output is not HTML-escaped, the sidecar default, and flags typed after positionals.
- **Mutation checks are logged.** The `## Log` records a mutation check for each property, including one test that first passed for the wrong reason and was fixed.

**2. Critical findings**
None.

**3. Important findings**
- **Non-finite times produce an invalid cast with exit 0** (`cast.go:56`, `cut.go:178`, `cut_prop_test.go:FuzzCut`). Two very large gaps with no idle cap sum to +Inf, which `Cut` turns into NaN; `encodeCast` then writes `NaN`, which is not valid JSON. `FuzzCut` hides this with `if math.IsInf(e.Gap,0) || e.Gap > 1e9 { return }`, but the plan says FuzzCut must yield an error or a valid cast. Fix: in `parseCast` (or right after the times loop in `Cut`), reject a gap or a running view time that is not finite or exceeds a sane bound, with an error naming the line. Then replace the skip in `FuzzCut` with an assertion that such input is refused. ARCH-SECURE: the output is fabricated rather than the failure being reported.

**4. Minor findings**
- **README not updated for `castcut cut`.** The plan schedules the README section for M2 Task 7 and the tool isn't released yet, so this is acceptable as long as M2 does it.
- **Property (f) is tested only as an example.** The plan asked for "marker precedes output at the same instant" as a property; it is covered only by `TestCutCaptionPlaysInRealTimeWithMarkerFirst`. Markers sort by their 3-decimal rounded start while output events use unrounded times. A marker can therefore land up to 0.5 ms after the first frame of its window. That doesn't break the contract, but the reason (f) isn't a property should be recorded.
- **Output timestamps can drift.** `cut.go:221` computes `e.Gap = round(it.at-prev, 6)` with `prev` unrounded, so the sum of the rounded gaps drifts from the true output times. The drift is about √N·3e-7 (tiny), and none of the properties checks absolute time. Rounding absolute times and taking differences would make it exact.
- **Duplicated encoder setup** (ARCH-DRY). `encodeCast` repeats the `json.NewEncoder` + `SetEscapeHTML(false)` setup that `marshalNoEscape` already does. The header could use `marshalNoEscape(c.Header)`.
- **Unhelpful error for a missing sidecar.** When the captions file is defaulted and missing, the error is `open take.captions.txt: no such file`. A hint like "run castcut annotate, or pass captions.txt" would help.
- **Non-atomic output write.** `os.WriteFile` can leave a partial file if it fails partway, which bends the "no output file on error" rule. Low risk.

**5. Test coverage notes**
Properties (a)–(e), the envelope run and the warp reference are real tests of the logic; nothing is mocked. The gaps are the non-finite fuzz escape (the Important finding above) and property (f) being only an example.

**6. Architecture check, principle by principle**
- **ARCH-DRY:** pass, apart from the minor encoder duplication.
- **ARCH-PURE:** pass.
- **ARCH-PURPOSE:** pass. The "Done when" timing properties are delivered.
- **ARCH-MOCK:** N/A for M1 (no external calls). M2 must deliver the fake `asciinema` plus the PQ-3 live conformance test.
- **ARCH-CONSTRAINTS:** pass. The envelope is tested with a 3 s bound while the atlas claims < 1 s; that's acceptable as a CI margin.
- **ARCH-SECURE:** flagged (the NaN output above).
- **ARCH-ORDER:** pass. `cut` keeps no state between events because it is a single-shot batch transform.
- **ARCH-FUNERAL:** pass. The only durable artifact is the `-o` output, which the operator owns.

For M2: the annotate server's Host/Origin/size guards and the serialized-PUT behaviour (PQ-6) are where ARCH-SECURE and ARCH-ORDER will actually apply.

**7. Plan revision recommendations**
- Add a `## Revisions` note that property (f) is pinned by an example row rather than a property, and why (rounded marker times).
- Once fixed, add a note that non-finite or absurdly large view times are rejected at parse time, so FuzzCut's "error or valid cast" invariant holds without a skip.

```findings
findings:
  - id: new
    severity: Important
    family: untrusted-input-fabricated-output
    title: |
      Non-finite view times yield a NaN-bearing invalid cast with exit 0; FuzzCut skips the case instead of asserting refusal
    detail: |
      Repro: two 1e308 gaps, no idle_time_limit → output line `[NaN, "o", "b"]`, summary `NaNs (view +Infs)`, exit 0. Reject non-finite/absurd cumulative view time in parseCast or Cut with a line-naming error; replace FuzzCut's IsInf/1e9 skip with an assertion that such input errors.
  - id: new
    severity: Minor
    family: plan-property-downgraded-to-example
    title: |
      Property (f) (marker before output at one instant) is only an example row; rounded marker times can trail the window's first frame by under 0.5 ms
  - id: new
    severity: Minor
    family: rounding-drift
    title: |
      Event gaps rounded against unrounded prev, so cumulative output time drifts from the true times (about sqrt(N)*3e-7)
  - id: new
    severity: Minor
    family: shared-helper-duplication
    title: |
      encodeCast re-implements the no-HTML-escape encoder setup that marshalNoEscape already provides
  - id: new
    severity: Minor
    family: docs-surface-gap
    title: |
      README has no castcut entry yet (scheduled for M2 Task 7); missing sidecar error gives no hint to run annotate or pass captions.txt
```

---

## Re-review — 2026-10-08T13:55:42-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 83 — castcut: record, annotate and cut terminal demos |
| repo | tools |
| issue file | workshop/issues/000083-castcut-record-annotate-and-cut-terminal-demos.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 4c1444f900957224cb3a10cc5bfeb9542988f2cf..fd911066c9ad6d473929afc3c58a03cf7dda7a9d |
| command | sdlc milestone-close --issue 83 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-08T13:55:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

BR-1 is fixed. `parseCast` now stops the take's cumulative length at `maxSeconds` and names the offending line. `Cut` has the same guard for casts built in code, and `Timing.validate` bounds every flag. Three tests cover it: a `cast_test.go` row, `TestCutRejects`, and a 1e308 seed in `FuzzCut`. Each one fails if the guard is removed. `go test ./cmd/castcut/` passes.

The same class of bug is still open on the captions side. A minutes field of 18 digits overflows `int` in `parseCaptions`. That gives a negative stamp, and `Cut` silently places the caption at 0 s and exits 0. I reproduced it: `~153722867280912931:00  overflowed` became `{"start":0,"end":4,...}` with exit 0. This is the second finding in family `untrusted-input-fabricated-output`. The fix should apply the rule to the whole family, not just this field.

1. **Strengths**
   - The duration bound has one source, `maxSeconds` (`cast.go:31`). The parser, `Cut` and the flag checks all use it, and the error names the line (`cast.go:66`).
   - `encodeCast` now uses `marshalNoEscape` (`cast.go:106`), which settles BR-4.
   - Output intervals are now differences of rounded absolute times (`cut.go:249-256`), so rounding no longer accumulates.
   - The design keeps a pure core with a thin shell: `Cut` is pure, and `runCut` only does IO and wiring.
   - The property suite runs captions through the real parser via `captionsText`. It also checks `warp` against a naive reference implementation and runs a 2×10⁵-event envelope test.

2. **Critical findings:** none.

3. **Important findings**
   - **Caption minutes overflow, so a garbage stamp becomes a caption at 0 s (exit 0)** (`captions.go:37-42`, `cut.go:204`). ARCH-SECURE; second finding in family `untrusted-input-fabricated-output`.
     - **The rule:** every number read from an input file is turned into a bounded value at its parse boundary, in [0, `maxSeconds`], and `Cut` re-checks the same range for input built in code.
     - **Where it applies:**
       - cast gaps (done)
       - `idle_time_limit` (safe: it only caps gaps, and non-positive values mean no limit)
       - caption minutes and seconds (open)
       - `Caption.At` in `Cut`: it only checks `cp.At > last`, so a negative or NaN stamp gets through.
     - **Fix sketch:**
       - In `parseCaptions`, compute `At` in float or check `min <= maxSeconds/60`, and reject anything over `maxSeconds` with a `path:line` error.
       - In `Cut`, change the check to `!(cp.At >= 0 && cp.At <= last)`.
       - Add a captions-parse row for an over-large stamp and a `Cut` row for `At: -1` and `At: NaN`.

4. **Minor findings**
   - Disposition notes for BR-2, BR-3 and BR-5 are in the findings block below.
   - `atlas/castcut.md` claims an envelope of "~10⁶ events", but only 2×10⁵ is measured. Either measure 10⁶ or state the measured figure.

5. **Test coverage notes**
   - The duration guard is pinned at both seams, and each test fails without it.
   - The fix for rounding drift and the missing-sidecar hint each have no test that fails without them.
   - A 20 s `FuzzCut` run is unlikely to produce an 18-digit minutes field. That is why the overflow got through, so add an explicit seed for it.

6. **Architectural notes for upcoming work**
   - ARCH-DRY: pass.
   - ARCH-PURE: pass.
   - ARCH-PURPOSE: pass for M1's scope. The cut contract and its properties are delivered.
   - ARCH-MOCK: N/A at M1. M1 calls no external binary. M2 adds `asciinema`, which needs the planned fake on PATH plus a live conformance check.
   - ARCH-CONSTRAINTS: pass. The envelope test has a 3 s budget.
   - ARCH-SECURE: flag (the Important finding above).
   - ARCH-ORDER: N/A at M1. `cut` is a single pure transform and holds no state between events. M2's annotate server (dirty flag, last writer wins) does need the state and event enumeration.
   - ARCH-FUNERAL: N/A at M1. The only durable thing `cut` creates is the output file the operator names. M2's sidecar and temp files should each say what removes them.

7. **Plan revision recommendations**
   - In the 2026-10-08 M1 boundary revision, extend "Durations are bounded at the door" to cover caption stamps, so it lists every input number.
   - If BR-2 is taken up, replace "either vacuous or flaky" with the marker-at-unrounded-time approach.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      parseCast (cast.go:66) and Cut (cut.go:201) reject cumulative length over maxSeconds and validate bounds the flags; cast_test row, TestCutRejects and the FuzzCut 1e308 seed each fail without the guard; FuzzCut's IsInf skip is gone.
  - id: BR-2
    disposition: not-addressed
    note: |
      Plan revision documents (f) as a deliberate example row, but the trailing case is still there (marker at round(warp,3) vs event at round(warp,6)); a cheap fix is to sort the marker at the unrounded warp(win.Start) and use the 3-decimal value only in the header, which makes (f) a general property.
  - id: BR-3
    disposition: not-addressed
    note: |
      The code fix is in (cut.go:249-256), but no test fails without it; (c) uses eps 2e-3 and cannot see about 3e-7 drift. Add a check that each output event's cumulative time equals round(warp(time), 6).
  - id: BR-4
    disposition: addressed
    note: |
      encodeCast now calls marshalNoEscape (cast.go:106); no behavior change, and the CLI no-HTML-escape row still covers it.
  - id: BR-5
    disposition: not-addressed
    note: |
      README is still scheduled for M2 Task 7 (fine). The missing-sidecar hint was added (main.go:108) but main_test.go has no row asserting it.
findings:
  - id: new
    severity: Important
    family: untrusted-input-fabricated-output
    title: |
      Caption minutes overflow int, so a garbage stamp becomes a caption at 0 s with exit 0
    detail: |
      2nd finding in this family. Rule: every number read from an input file is bounded to [0, maxSeconds] at its parse boundary, and Cut re-checks the range for input built in code. Repro: ~153722867280912931:00 gives a negative At (min*60 overflows), which passes the cp.At > last check and lands at start 0. Fix: bound At in parseCaptions with a path:line error, use !(cp.At >= 0 && cp.At <= last) in Cut, and add test rows for an over-large stamp, -1 and NaN.
  - id: new
    severity: Minor
    family: unsupported-performance-claim
    title: |
      atlas/castcut.md claims an envelope of about 10^6 events, but only 2x10^5 is measured
    detail: |
      TestCutStaysInsideItsEnvelope runs 2e5 events and 1e3 captions; either measure 1e6 or state the measured figure.
```

---

## Re-review — 2026-10-08T13:58:08-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 83 — castcut: record, annotate and cut terminal demos |
| repo | tools |
| issue file | workshop/issues/000083-castcut-record-annotate-and-cut-terminal-demos.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 4c1444f900957224cb3a10cc5bfeb9542988f2cf..cd260ebfbac30c17a342b4e4782ccaeeb84addaf |
| command | sdlc milestone-close --issue 83 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-08T13:58:08-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

`castcut cut` does what the M1 Plan row promises. The timing model is a pure pipeline (`planWindows` → `buildSegments` → `warper` → `Cut`). It has property tests over random takes, an envelope run, a naive-`warp` reference and three fuzz targets, plus CLI wiring through `run(args, stdout, stderr)`. `go test` and `go vet` pass. Of the prior findings, BR-6 and BR-7 are properly addressed, and BR-2 is now a recorded, reasoned decision. The one new problem is in the same family as BR-1 and BR-6. The last commit says Cut re-checks ranges for input built in code, but Cut only re-checks the total with `last > maxSeconds`, and that check is not NaN-safe. I confirmed this with a scratch probe, since deleted. `Cut(castOf(0, 1, NaN), nil, defaultTiming)` returns no error and writes gaps `[0.2, NaN]`. `castOf(0, 5, -10, 1)` is quietly collapsed to gaps `[0, 0, 0.2]`. The CLI can't reach either case today because `parseCast` refuses them. Still, the stated rule doesn't hold yet, and the fix is cheap. The other open findings are Minor and lack regression tests.

1. **Strengths**
   - Each number from a file is bounded at its parse boundary and the error names the line: intervals and the running total in `cast.go:58-64`, stamps in `captions.go:36-40`, and `idle_time_limit` in `cast.go:96-98`. The overflow stamp `~153722867280912931:00` and its neighbours are test rows (`captions_test.go:30-41`).
   - `warper` uses cumulative output time plus binary search, and is pinned against a naive reference within 1e-9 (`cut_prop_test.go:216`). That is an O(log S) lookup with an independent oracle.
   - `Cut` doesn't mutate its input: it copies the header before `idleLimit` pops a key, and `TestCutDoesNotMutateItsInput` covers this.
   - `FuzzCut` checks the properties on every accepted input instead of skipping hard cases, and the 1e308 seed is kept.
   - The atlas now quotes the measured envelope (2×10⁵ events in under 1 s) instead of the 10⁶ that was never measured.

2. **Critical:** none.

3. **Important**
   - **This is the 3rd finding in family `untrusted-input-fabricated-output`.** Two earlier rounds fixed one instance each. **The rule:** a `Cast` is valid only if every gap is in [0, ∞) and the running total is in [0, maxSeconds]. That predicate should live in one place, a `func (c Cast) validate() error` or `checkGap`/`checkTotal` helpers written as `!(x >= 0 && x <= max)`. Both `parseCast` (`cast.go:58-64`) and `Cut` (`cut.go:183-186`) should call it, so the file path and the in-code path can't drift apart (ARCH-DRY, ARCH-SECURE). `Cut` currently checks only `last > maxSeconds`. That check is false for NaN, and it never looks at negative gaps. Add `Cut` test rows for `NaN`, `-10` and `+Inf` gaps. Also add a `FuzzCut` variant that builds a `Cast` from raw fuzzed float gaps, bypassing `parseCast`, and asserts "error, or `checkCut` holds".

4. **Minor**
   - The plan's M1 Tasks 1–4 still read "byte-identical" with unchecked pyjson boxes. The banner on line 5 points to the Revisions, which is enough. Ticking or striking the superseded rows would stop `grep '- \[ \]'` from reporting stale work.
   - The "no captions at … stamp them with `castcut annotate`" hint (`main.go:113`) has no test row in `main_test.go`. This is the BR-5 remainder.

5. **Test coverage notes**
   - No test pins the BR-3 fix. A regression test would cut ~10⁵ events with gaps of 1/3 and assert that the cumulative sum of `Gap` matches `round(warp(times[i]), 6)` within 1e-9. The old code drifts by about 1e-4, so the test would go red without the fix.
   - The idle_time_limit bound is tested only through `Cut` (`TestCutBoundsIdleLimit`), which is fine because `idleLimit` is reached only from there.

6. **Architecture**
   - **ARCH-DRY:** flag. The duration-validity predicate is spread over `parseCast` and `Cut` in two different, inconsistent forms (see Important). `marshalNoEscape` is now shared, so BR-4 holds.
   - **ARCH-PURE:** pass. `Cut` and the parse functions are pure; `runCut` is a thin IO shell.
   - **ARCH-PURPOSE:** pass. M1 delivers the contract (`captions` header, `m` markers, `idle_time_limit` consumed) and the timing properties the Done-when names.
   - **ARCH-MOCK:** N/A for M1. There's no external call; asciinema arrives in M2 with a planned fake and a conformance test (PQ-3).
   - **ARCH-CONSTRAINTS:** pass. The envelope is declared, measured at 2×10⁵ events, and the atlas states the measured figure.
   - **ARCH-SECURE:** flag. The in-code `Cast` path doesn't enforce the invariants that the file path does (same finding as above).
   - **ARCH-ORDER:** N/A. `Cut` holds no state between events; it is a single batch transform over an in-memory slice. Only `buildSegments` has cursor state, and it is local to the call.
   - **ARCH-FUNERAL:** pass. The only durable artifact is the `-o` output file, which the operator names and owns. Nothing is cached or kept as a sidecar in M1.
   - For M2: when `annotate` writes `take.captions.txt`, make it go through `parseCaptions` round-trip validation before writing, so the "bounded at the door" rule covers the writer side too.

7. **Plan revision recommendations**
   - Under `2026-10-08 — M1 boundary review`, add a line: "Cut and parseCast share one Cast-validity predicate (gap ≥ 0, total ≤ maxSeconds, NaN-safe form); round 3 found Cut's re-check NaN-blind."

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      Plan Revisions (M1 boundary review) record (f) as a deliberate example row with reasoning; the speed-1 example row is mutation-checked per the issue Log. The sub-0.5 ms lag is an accepted trade-off.
  - id: BR-3
    disposition: not-addressed
    note: |
      cut.go:250-257 now differences rounded absolute times (correct), but no test fails without it; add a 1e5-event, gap-1/3 cumulative-sum test.
  - id: BR-5
    disposition: not-addressed
    note: |
      README is legitimately scheduled for M2 Task 7; the new sidecar hint at main.go:113 has no main_test row.
  - id: BR-6
    disposition: addressed
    note: |
      captions.go:36-40 bounds min before multiplying and the sum after; captions_test.go:30-41 has overflow, too-big, and past-a-week rows; Cut rejects -1/NaN (cut_test.go:148).
  - id: BR-7
    disposition: addressed
    note: |
      atlas/castcut.md now states the measured 2x10^5 events + 10^3 captions in under 1 s with the test name.
findings:
  - id: new
    severity: Important
    family: untrusted-input-fabricated-output
    title: |
      Cut's re-check of a code-built Cast is NaN-blind and ignores negative gaps, so NaN gaps produce a NaN cast with no error
    detail: |
      3rd finding in this family. Rule: one Cast-validity predicate (each gap in [0,inf), total in [0,maxSeconds], written NaN-safe as !(x>=0 && x<=max)) shared by parseCast and Cut. Repro: Cut(castOf(0,1,NaN),nil,defaultTiming) gives gaps [0.2 NaN] and err nil; castOf(0,5,-10,1) silently collapses to [0 0 0.2]. Add Cut rows for NaN/-10/+Inf gaps and a FuzzCut variant over raw float gaps that bypasses parseCast.
  - id: new
    severity: Minor
    family: docs-surface-gap
    title: |
      Plan M1 Tasks 1-4 still read byte-identical with unchecked pyjson boxes
    detail: |
      2nd finding in this family. Rule: superseded plan rows are struck or ticked when a revision supersedes them. The line-5 banner points at Revisions, but grep for unchecked boxes still reports stale work.
```

---

## Re-review — 2026-10-08T14:00:15-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 83 — castcut: record, annotate and cut terminal demos |
| repo | tools |
| issue file | workshop/issues/000083-castcut-record-annotate-and-cut-terminal-demos.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 4c1444f900957224cb3a10cc5bfeb9542988f2cf..a8bef03eb806a5d6541f7d0ae6a8b1fbddf21d24 |
| command | sdlc milestone-close --issue 83 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-08T14:00:15-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The fix for BR-8 is correct, but it broke the test suite. Commit a8bef03 now runs every number castcut reads through one range check, `inRange` (`cmd/castcut/cast.go:35`), which correctly rejects NaN, at every input boundary and again inside `Cut`. The trouble is that each event's interval is now checked before the running total. So the `overflowing duration` row in `TestParseCastRejects` stops at the per-event check and never reaches the "recording runs past 168 hours" check it was written for. `go test ./cmd/castcut/` fails at HEAD:

```
--- FAIL: TestParseCastRejects (0.00s)
    cast_test.go:23: overflowing duration: err = t.cast:2: event interval is not a number of seconds in [0, 604800], want "t.cast:2: recording runs past 168 hours"
FAIL	github.com/xianxu/tools/cmd/castcut
```

The boundary can't be crossed while the suite is red. The fix is small: change that one test row (see below) and re-run.

**1. Strengths**
- **One shared range check.** `inRange` (`cast.go:35`) is written as `x >= 0 && x <= max`, so NaN fails it by construction. It is now used by `parseCast`, `idleLimit`, `parseCaptions`, `Timing.validate` and `Cut`. That is the right answer to a family of findings that kept recurring.
- **`Cut` re-checks inputs built in code** (`cut.go:190-216`). It rejects empty events, NaN, negative and infinite intervals, and caption times out of range. `TestCutRejects` (`cut_test.go:148-157`) has a row for each case. The `{1, -1}` row only passes because of the new per-event check, so that test really guards the fix.
- **Pure core, thin shell.** `Cut`, `planWindows`, `buildSegments` and the warp step are pure. `runCut` only reads files, writes the output and prints.
- **Property tests are real.** `checkCut` states properties (a)–(e) on their own terms rather than restating the implementation, and the 200k-event envelope test measures speed against a 3-second budget.

**2. Critical**
- **`cmd/castcut/cast_test.go:19`: the test suite is red at HEAD.** The `overflowing duration` row feeds an interval of `1e308`, which the per-event check now rejects first, so the total check is no longer tested.
  - Fix: use intervals that are each in range but add up to more than the limit, e.g. `[4e5, "o", "a"]` then `[4e5, "o", "b"]`, and expect `t.cast:3: recording runs past 168 hours`. Keep a separate row for a `1e308` interval expecting `not a number of seconds`.
  - Then re-run `go test ./cmd/castcut/` before `milestone-close`.
  - Add a lesson to `workshop/lessons.md`: a commit that tightens a check must run the package's tests before it is committed.

**3. Important**
- None new.

**4. Minor**
- **`cast.go:100-104`: a negative `idle_time_limit` gets the wrong error message.** Changing `*v <= 0` to `*v == 0` means a negative limit, which used to mean "no limit", is now rejected as "idle_time_limit -1 is past 168 hours". Rejecting it is reasonable, but the message names the wrong cause, the behaviour change isn't recorded anywhere, and no test covers it. Say "not in [0, …]" instead, and add a row to `TestCutBoundsIdleLimit`.
- **Plan line 7 still says the goal is "byte-identical".** The Core concepts table (lines 74-75) still lists `pyjson.go`. The banner on line 5 covers this, so it's only worth noting.

**5. Test coverage notes**
- The hint printed when the default sidecar file is missing (`main.go:113-115`) is never exercised by a test. See BR-5 below.
- The fix for output-time drift (BR-3) has no test that would fail without it. The property checks allow 2e-3 of error, which would hide drift of about 1e-7.

**6. Architecture**

| Marker | Result | Note |
|---|---|---|
| ARCH-DRY | pass | `inRange` is single-sourced. `parseCast` and `Cut` each loop over the events, but share the same check. |
| ARCH-PURE | pass | |
| ARCH-PURPOSE | pass | M1 delivers `cut`, with the scope as revised. |
| ARCH-MOCK | N/A | M1 calls no external binary. M2's asciinema seam is the place to apply it. |
| ARCH-CONSTRAINTS | pass | The 200k-event, 3-second budget is measured by a test. |
| ARCH-SECURE | pass | Every number read is bounded at parsing, and errors are visible. The only gap is the idle-limit message above. |
| ARCH-ORDER | pass | Nothing persists between events: `Cut` is a single-shot transform. |
| ARCH-FUNERAL | pass | The tool writes only the file the operator names with `-o`. The untracked `c.txt`, `o.cast` and `t.cast` in the working tree are outside the review window, but should be cleaned up. |

**7. Plan revision recommendations**
- Add a `## Revisions` entry saying a negative `idle_time_limit` is now refused rather than treated as "no limit".
- Optionally strike the stale goal line (line 7) and the `pyjson` rows in the Core concepts table.

```findings
dispose:
  - id: BR-3
    disposition: not-addressed
    note: |
      Structurally fixed (prev is now the rounded absolute time, cut.go:252-258) but no test fails without it; checkCut eps 2e-3 cannot see 1e-7 drift. Add: sum of out gaps equals round(w.warp(view),6) within 1e-9 on the 200k take.
  - id: BR-5
    disposition: not-addressed
    note: |
      Sidecar hint exists in main.go:113-115 but no CLI test runs cut without a sidecar and asserts the castcut annotate hint; the README part stays scheduled for M2 Task 7.
  - id: BR-8
    disposition: addressed
    note: |
      inRange shared by parseCast/idleLimit/parseCaptions/validate/Cut; TestCutRejects rows NaN, {1,-1}, +Inf, {} — the {1,-1} row goes red without the per-gap check.
  - id: BR-9
    disposition: addressed
    note: |
      Superseded M1 rows are now ticked and struck with a pointer to Revisions (plan lines 169-254).
findings:
  - id: new
    severity: Critical
    family: untrusted-input-fabricated-output
    title: |
      go test ./cmd/castcut/ is red at HEAD: TestParseCastRejects overflowing-duration row now hits the per-gap check
    detail: |
      a8bef03 made the per-gap inRange reject 1e308 before the total check, so cast_test.go:19 fails and the total-overflow path is untested. This is the 4th in the family but a regression in the family rule's own fix, not a new instance; the rule (one inRange at every boundary) holds. Fix: row with [4e5],[4e5] expecting t.cast:3 recording runs past 168 hours, plus a 1e308 row expecting not a number of seconds; run the tests before committing.
  - id: new
    severity: Minor
    family: error-message-misstates-cause
    title: |
      Negative idle_time_limit is now refused as is past 168 hours (cast.go:100-104), a silent untested behaviour change
    detail: |
      Previously a negative limit meant no limit. Say not in [0, max], add a TestCutBoundsIdleLimit row, and record the change in Revisions.
```

# Boundary Review — tools#83 (whole-issue close)

| field | value |
|-------|-------|
| issue | 83 — castcut: record, annotate and cut terminal demos |
| repo | tools |
| issue file | workshop/issues/000083-castcut-record-annotate-and-cut-terminal-demos.md |
| boundary | whole-issue close |
| milestone | — |
| window | 42e4a273f8d0ab25e30802534f4ae8bd68f611d1..21ee5e964b96f7f188fac7c4d94a2a7d51547139 |
| command | sdlc close --issue 83 |
| reviewer | claude |
| timestamp | 2026-10-08T17:44:35-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

castcut is in good shape: the cut timing model, record's status handling and the annotate server are solid, and `go test -count=1 ./cmd/castcut/` passes at HEAD. Two things stop a clean SHIP. First, BR-12 is still open. Its fix bounds the view length but not the output length, so a tiny `--speed` still makes `cut` write a cast that castcut's own reader refuses, and it exits 0. Second, the issue is being closed while the M3 row is unchecked and two of its Done-when bullets have no evidence: the brew install and the parley.nvim follow-up.

**1. Strengths**
- `inRange` at `cmd/castcut/cast.go:35` is one range check, written so NaN fails it, and every read boundary uses it: each interval, the running total, the idle limit, caption stamps and the timing flags. This is the rule-level fix the earlier rounds asked for.
- Each event's interval is computed from rounded absolute times (`cut.go:264-272`), so rounding error cannot build up along the stream.
- `warp` builds a cumulative sum once and then uses binary search (`cut.go:134-152`). It is checked against a naive reference (`TestWarpMatchesTheNaiveReference`) and an envelope test of 2×10⁵ events.
- `record` takes the command's status from the `x` event in the take, not from asciinema's exit code. The same `recordContract` is tested against a fake asciinema and against the real one live.
- `help_test.go` derives every flag from each command's `-h`, which keeps the help text from drifting from the code.

**2. Critical findings:** none.

**3. Important findings**
- **BR-12 is not fixed (family `untrusted-input-fabricated-output`).** `cut.go:227` checks `inRange(view)` but never `inRange(w.warp(view))`, and the finding asked for both. Between captions the rate is `squeeze/Speed` (`cut.go:117`), and `validate` only requires `Speed > 0`. Reproduced at HEAD:
  - `--speed 1e-10` exits 0 with a total of 130400000016.1 s. Running `cut` again on that output fails with `event interval is not a number of seconds in [0, 604800]`.
  - `--speed 1e-300` exits 0 with times around 1e300.

  **The rule:** every duration castcut writes must pass the same `inRange` check as every duration it reads. Because output time only grows, checking the total is enough. **Fix:** in `Cut`, refuse unless `inRange(w.warp(view))`, with an error naming `--speed` as well as the caption flags. Then make `FuzzCut` fuzz `Timing` instead of always using `defaultTiming`, and have it assert that `parseCast(encodeCast(out))` succeeds.
- **Done-when is not met at whole-issue close (family `verification-deferred-past-boundary`, 2nd finding).** The M3 row (Homebrew formula and tag, parley.nvim follow-up issue) is still `- [ ]`. The Log says the formula is uncommitted with its sha256 waiting on the tag, and there is no evidence that `brew install xianxu/tools/castcut` works or that the parley.nvim issue exists. **The rule:** a close's evidence must cover every Done-when bullet, or a Revisions entry must move that bullet out of the issue. **Fix:** finish the outward steps (they need operator approval), or revise Done-when to say they continue after close in a named follow-up.

**4. Minor findings**
- The in-browser check that the notes survive a page reload was never confirmed (the Log says so itself). Re-run it during the next real take.

**5. Test coverage notes**
- The property tests and the fuzz test both run only with `defaultTiming` (or the narrow ranges in `randomTake`). That is why the timing-flag side of BR-12 got past them.
- BR-11's change is tested: `TestCutBoundsIdleLimit` has a `-1` row that checks the `not in [0,` message.

**6. Architectural notes**
- **ARCH-DRY:** pass. One range check; `marshalNoEscape` is reused.
- **ARCH-PURE:** pass. `Cut`, `planWindows`, `buildSegments` and `warp` are pure; the IO stays in `main`, `record` and `annotate`.
- **ARCH-PURPOSE:** flag. See the Done-when finding.
- **ARCH-MOCK:** pass. A stateful fake asciinema plus a live conformance test.
- **ARCH-CONSTRAINTS:** pass. The envelope is measured, not claimed.
- **ARCH-SECURE:** flag (BR-12). An output still escapes castcut's own input bound. The annotate server checks Host and Origin and caps request size.
- **ARCH-ORDER:** pass. The viewer keeps notes disabled until the first load succeeds, so a failed load cannot be autosaved over the sidecar. Otherwise nothing holds state between events.
- **ARCH-FUNERAL:** pass. `annotate` sweeps its own stale `.tmp` files, and record refuses to overwrite an existing take.

**7. Plan revision recommendations**
- The BR-12 entry in the plan's Revisions says castcut "never writes a cast it would itself refuse to read". That is false for `--speed`. Correct it once the fix lands.
- Add a Revisions entry that either closes M3's outward steps with evidence or moves them out of Done-when.

```findings
dispose:
  - id: BR-11
    disposition: addressed
    note: |
      TestCutBoundsIdleLimit has a -1 row asserting the "not in [0," message (cut_test.go:176-185); the refusal is recorded in the issue Log at the M1 close.
  - id: BR-12
    disposition: not-addressed
    note: |
      Only the view is bounded (cut.go:227), not warp(view). --speed 1e-10 exits 0 and the re-cut is refused (interval past 604800). FuzzCut still uses defaultTiming and has no round-trip assert.
findings:
  - id: new
    severity: Important
    family: verification-deferred-past-boundary
    title: |
      Whole-issue close with the M3 row unchecked; brew install and the parley.nvim follow-up have no evidence
    detail: |
      2nd in family. Rule: close evidence covers every Done-when bullet, or a Revisions entry moves the bullet out. The formula is uncommitted (sha256 waits on the tag); the parley issue is not shown to exist.
```

# Boundary Review — tools#55 (whole-issue close)

| field | value |
|-------|-------|
| issue | 55 — define: wrap streamed LLM responses to terminal width |
| repo | tools |
| issue file | workshop/issues/000055-define-wrap-streamed-llm-responses-to-terminal-width.md |
| boundary | whole-issue close |
| milestone | — |
| window | c4123297dc2a28907227958e2ccfe13d8e94f4f7..2c76fe5def2678869b09b267bb7d238ad382ddd1 |
| command | sdlc close --issue 55 |
| reviewer | codex |
| timestamp | 2026-09-13T13:01:57-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation meets the text-wrapping and flush requirements, and all verification passed. One styling defect blocks shipping: a wrapped highlighted phrase loses its highlight when its opening row leaves the viewport.

1. **Strengths**
   - Reuses `visibleCells` and `scanEscape`, preserving existing width semantics.
   - Flushes highlighting before wrapping on every exit, retaining the first error.
   - Tests exercise chunk boundaries, Unicode, input limits, partial streams, and unchanged session history.
   - Atlas documentation accurately describes the wrapping policy and limits.

2. **Critical findings**
   - **Wrapped continuation rows depend on offscreen styling** — [answerwrap.go:107](/Users/xianxu/workspace/tools/cmd/define/answerwrap.go:107). At width 20, `123456789012345 ` followed by highlighted `alpha beta` produces `123456789012345 <green>alpha\nbeta<reset>`. When only the second row remains visible, [screen.go:530](/Users/xianxu/workspace/tools/cmd/define/screen.go:530) paints it without the opening green sequence. Preserve active SGR state across inserted boundaries so continuation rows render independently; reuse `sgrState`. Add a regression that paints a viewport beginning inside a wrapped highlighted phrase. **ARCH-PURPOSE**.

3. **Important findings**
   - None.

4. **Minor findings**
   - None.

5. **Test coverage notes**
   - Passed: `go test ./cmd/define/... -count=1`, focused wrapping tests, `go vet ./cmd/define/...`, and pinned-range `git diff --check`.
   - Current highlighting assertions cover individual words and whole transcripts, not a viewport starting inside a styled span.

6. **Architectural notes**
   - **ARCH-DRY: pass** — shared width and escape helpers.
   - **ARCH-PURE: pass** — wrapping decisions tested in memory; output injected.
   - **ARCH-PURPOSE: flag** — viewport styling preservation remains incomplete.
   - **ARCH-MOCK: pass** — existing wire fake and portable temporary stores.
   - **ARCH-CONSTRAINTS: pass** — pending-input limits enforced and tested.
   - **ARCH-SECURE: pass** — malformed tails handled explicitly; no new credentials.
   - **ARCH-ORDER: pass** — sequential ownership, ordered flushing, retained errors.
   - **ARCH-FUNERAL: pass** — no new durable runtime artifacts.

7. **Plan revision recommendations**
   - Append a `## Revisions` entry requiring independently styled continuation rows and viewport-level coverage for highlighted phrases and enclosing SGR styles.

```findings
findings:
  - id: new
    severity: Critical
    family: viewport-independent-styling
    title: |
      Wrapped highlighted phrases lose styling when their opening row leaves the viewport.
    detail: |
      cmd/define/answerwrap.go:107 inserts a newline without restoring active SGR state on the continuation row. screen.Paint paints only visible rows (cmd/define/screen.go:530), so a viewport starting at that continuation displays it without its highlight, violating the styling-preservation contract (ARCH-PURPOSE). Reuse sgrState to preserve styling independently across wrapped rows and add a viewport-paint regression covering a multiword highlighted phrase.
```

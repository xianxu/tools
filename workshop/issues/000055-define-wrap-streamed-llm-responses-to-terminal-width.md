---
id: 000055
status: codecomplete
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-13
estimate_hours: 1.09
started: 2026-09-13T12:39:30-07:00
actual_hours: 0.47
---

# define: wrap streamed LLM responses to terminal width

## Problem

LLM answers in the interactive `define` screen extend past the right edge and
are clipped. The supplied screenshot shows long paragraphs cut off mid-sentence.
`runAsk` streams through `highlightWriter` into `liveScreen.Write`; the latter
deliberately wraps only pinned (`--play`) screens. Dictionary rendering already
wraps, but streamed answers have no corresponding step.

## Spec

Word-wrap LLM output to `opt.width`, using visible terminal cells rather than
bytes. Preserve streaming, paragraphs, learned-word highlighting, and the raw
answer stored for follow-up context. Width zero keeps piped output unchanged.
Retain the existing policy for a single word wider than the terminal (do not
split it) and terminals narrower than `minWrapWidth` (wrapping disabled).

## Done when

- Long answer paragraphs are readable in the interactive screen.
- Chunk boundaries do not change wrapping or lose words, Unicode, or styling.
- Every stream exit flushes pending output, including interruption and errors.
- Focused regression tests and the define test suite pass.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Calibration is marked stale/provisional.*

The spec/plan capture uses the low-end issue-spec primitive (0.5 design × 0.5
for a clear bug report; 0.1 implementation × 0.4). Two smaller-Go-module
primitives cover the writer and its ask integration with tests: design 0.3 and
0.2 × 0.2 for the approved plan; implementation 0.5 each × 0.4. This extends
the established highlightWriter contract and reuses visibleCells/scanEscape;
no new library or external-service integration. Atlas design 0.05 × 0.2,
implementation 0.1 × 0.4; one boundary review uses implementation 0.5 × 0.4.
Familiarity 1.0; thorough-plan design buffer 15%.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.25 impl=0.04
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: atlas-docs design=0.01 impl=0.04
item: milestone-review design=0 impl=0.20
design-buffer: 0.15
total: 1.09
```

## Plan

- [x] Implement and test the answer-stream wrapper per [plan](../plans/000055-define-answer-wrap-plan.md).
- [x] Verify real ask wiring with the existing wire-level LLM fake and a narrow live screen.
- [x] Run verification, update the atlas, and close through the SDLC review gate.

## Log

### 2026-09-13
- 2026-09-13: closed — BR-1 fixed in 7815f83: viewport-start regressions fail before and pass after for a highlighted phrase, enclosing styles, and explicit/new wrapping; shared sgrState replay preserves independent rows. go test ./cmd/define/... passed (109.307s), focused tests with -race passed, go vet ./cmd/define/... and git diff --check clean. Original width20 regression preserves every word and raw history. Local candidate retaining installed #54 also passes focused tests, build, and dictionary smoke.; review verdict: SHIP

- Claimed issue; traced the clipping to `liveScreen.Write`'s explicit streaming
  exception. `wrapWritten` cannot be applied independently to arbitrary deltas:
  each chunk resets its column, and a chunk can end inside a word or escape.
- ARCH-DRY: reuse `visibleCells` and `scanEscape`; keep wrapping out of the
  viewport/click-map machinery. Durable plan prepared; implementation awaits
  the repository's required approval for non-trivial work.
- Fresh-context plan review found no architectural blockers. Added explicit
  malformed-stream deferred-flush coverage and the minimum-width boundary to
  the plan's Revisions section. No implementation code changed yet.
- Operator approved implementation by asking to continue #55. Plan-quality
  round 2 accepted the pending-input bounds and function-level test strategies;
  estimate gate passed. Rebased #55 onto origin/main to leave #54's pending
  publication separate. #56's requested task capture is preserved.
- Regression before wiring: the real SSE capture produced rows of 251, 153,
  and 104 display cells at width 20. Wrapper tests first failed with a
  pass-through implementation; the same focused tests now pass after wrapping.
  Every byte split and byte-at-a-time styled Unicode, oversized words, tail
  flushing, poisoned writers, input caps, and pipe pass-through are covered.
- Integration covers complete, truncated, interrupted, and injected-malformed
  endings, highlighted words, and raw history preservation. JunkFrame itself is
  truncation in this transport; the malformed branch uses a real-stream error
  adapter. Existing TestApplyShapeSetsBothWidths covers the minimum-width policy.
- Close round 1 requested rework (BR-1): a styled continuation lost its opening
  SGR when the preceding row scrolled offscreen. Three viewport regressions
  reproduced the loss for inserted/explicit newlines and enclosing styles.
  Fixed in 7815f83 by replaying the shared bounded sgrState at both kinds of
  boundary, observing original escapes only. A real highlightWriter phrase
  regression also passes. Focused race tests and vet are clean after the fix;
  the full suite is being rerun. Added the viewport-level testing rule to lessons.
- After BR-1: full `go test ./cmd/define/...` passed (109.307s); focused wrapping
  and ask tests passed with `-race`; vet and diff whitespace checks passed.
  Local candidate 7cf97ef combines the installed #54 baseline and both #55 code
  commits, passes the focused integration tests, builds cleanly, and passes help
  plus a no-audio/no-capture dictionary lookup smoke check.
- Close round 2 returned SHIP with no remaining findings. The reviewer also
  removed both SGR replay calls in a scratch mutation and all four viewport
  cases failed. Installed local candidate 7cf97ef atomically at `bin/define`,
  preserving #54; the PATH symlink resolves to that revision and help succeeds.
  Existing processes load the fix on restart.

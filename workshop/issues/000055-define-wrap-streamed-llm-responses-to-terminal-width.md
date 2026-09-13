---
id: 000055
status: working
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-13
estimate_hours: 1.09
started: 2026-09-13T12:39:30-07:00
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

- [ ] Implement and test the answer-stream wrapper per [plan](../plans/000055-define-answer-wrap-plan.md).
- [ ] Verify real ask wiring with the existing wire-level LLM fake and a narrow live screen.
- [ ] Run verification, update the atlas, and close through the SDLC review gate.

## Log

### 2026-09-13

- Claimed issue; traced the clipping to `liveScreen.Write`'s explicit streaming
  exception. `wrapWritten` cannot be applied independently to arbitrary deltas:
  each chunk resets its column, and a chunk can end inside a word or escape.
- ARCH-DRY: reuse `visibleCells` and `scanEscape`; keep wrapping out of the
  viewport/click-map machinery. Durable plan prepared; implementation awaits
  the repository's required approval for non-trivial work.
- Fresh-context plan review found no architectural blockers. Added explicit
  malformed-stream deferred-flush coverage and the minimum-width boundary to
  the plan's Revisions section. No implementation code changed yet.

---
id: 000055
status: working
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-13
estimate_hours:
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

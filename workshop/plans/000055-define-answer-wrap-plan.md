# Define answer wrapping implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy.
> Use superpowers-executing-plans for this small, sequential fix. Steps use
> checkboxes for tracking.

**Goal:** Read streamed LLM answers without paragraphs disappearing past the
terminal's right edge.

**Architecture:** Put a streaming word wrapper between `highlightWriter` and
stdout in `runAsk`. Keep the screen's buffer, paint, and click-map contracts
unchanged. Reuse the existing display-cell counter and escape scanner.

**Tech stack:** Go, existing define rendering helpers, existing LLM wire fake.

## Core concepts

| Name | Lives in | Status |
|------|----------|--------|
| `answerWrapWriter` | `cmd/define/answerwrap.go` | new |

`answerWrapWriter` owns one answer's unfinished token, pending whitespace,
current display column, and first downstream error. It accepts an `io.Writer`
and width, and exposes `Write([]byte) (int, error)` and `Flush() error`.
Its output can be checked directly with a bytes buffer; no terminal or model
is needed for unit tests. Existing `visibleCells` and `scanEscape` remain the
only width and escape-grammar implementations.

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `runAsk` | `cmd/define/ask.go` | modified | highlight → wrap → stdout |

The existing `opt.width` comes from terminal measurement and is zero for pipes.
The REPL already updates it on resize. The wrapper captures this width for each
answer, consistent with dictionary rendering. Reflowing old output when a
terminal shrinks during an answer is outside this fix.

## Behavior and constraints

- ARCH-PURPOSE: apply at `runAsk`, shared by interactive, forced, and one-shot
  questions. Preserve the unwrapped `answer` builder and session history.
- ARCH-DRY / ARCH-PURE: use existing `visibleCells` and `scanEscape`; all new
  wrap decisions are testable in memory. No screen arithmetic or extra renderer.
- ARCH-CONSTRAINTS: streaming UI; release each completed word immediately,
  holding only the unfinished word/escape and intervening whitespace. Do not
  buffer a paragraph until newline. Width zero bypasses buffering and rewriting.
  A word wider than the width is emitted intact on its own row, matching
  `wrapText`. Keep input whitespace when it fits; replace inter-word whitespace
  with a newline when a wrap is needed. Explicit newlines and blank lines survive.
- ARCH-SECURE: text is external model output; parse complete escapes using the
  established scanner, and retain incomplete UTF-8/escape tails until more data
  arrives. On final flush preserve any remaining bytes, as `highlightWriter`
  does. No new credentials, persisted input formats, or terminal-control policy.
- ARCH-MOCK: reuse `llmtest.Fake` and `stream-sample.sse`; no live API call is
  needed to reproduce a rendering defect.
- ARCH-ORDER: one request owns one writer, without goroutines. State transitions:
  text extends the pending token; whitespace closes a token; an explicit newline
  emits pending text and resets the column; `Flush` emits the final token;
  downstream error poisons the writer and all subsequent calls return it without
  emitting. Escapes are zero-width token content and never overtake held text.
  Flush highlighting first, then wrapping, on every return path. Report the
  first write/flush failure through the existing answer-write diagnostic.
- ARCH-FUNERAL: the writer creates nothing durable and dies with `runAsk`.
  Tests use temporary stores; answer persistence is unchanged.

## Implementation

### Task 1: Streaming wrapper

**Files:** create `cmd/define/answerwrap.go` and
`cmd/define/answerwrap_test.go`.

- [ ] Write table tests for narrow text, paragraph breaks, width zero, words
  longer than the width, indentation/whitespace, and a final word without newline.
  Pin literal expected output for representative text. For every byte split of
  styled Unicode input, require output equal to that literal expectation;
  include splits inside UTF-8 and ANSI sequences. Assert completed words appear
  before `Flush`, and no bytes are duplicated after a short/error write.
- [ ] Run `go test ./cmd/define -run '^TestAnswerWrap' -count=1`; confirm failure
  from the absent behavior, then implement the writer.
- [ ] Implement token scanning with `scanEscape`: complete escape sequences are
  copied into the pending token without changing its width; incomplete sequences
  wait. Space/tab boundaries decide the previous token; newlines also reset the
  display column. Use `visibleCells(token)` when deciding whether the token and
  pending gap fit. Emit a newline before a token that does not fit when the row
  already has content. Never add repeated empty rows for an oversized token.
  Preserve leading indentation, with the same no-word-splitting policy as the
  existing renderer. Flush drains the tail exactly once.
- [ ] Re-run focused tests until green. Use the existing `shortWriter` test
  double where suitable; assert `io.ErrShortWrite` for a nil-error short write.

### Task 2: Wire and verify the real ask path

**Files:** modify `cmd/define/ask.go`, `cmd/define/askrun_test.go`, and
`atlas/define.md`.

- [ ] Add a regression using `askRig`, the committed SSE capture, and a narrow
  `liveScreen`. Check answer buffer rows fit the chosen width for ordinary
  words, and compare the full whitespace-normalized answer with a width-zero
  control. Include highlighting and an interrupted/truncated stream; retain the
  full raw answer in session context. Run the test before wiring and see it fail.
- [ ] Construct the wrapper before `newHighlightWriter`; pass the wrapper as
  the highlighter's output. In the existing deferred flush, flush highlighting
  before wrapping and report errors through the existing stderr diagnostic.
  Keep all answer accumulation and capture logic unchanged.
- [ ] Update the atlas's answer-output description with terminal wrapping and
  the existing narrow-terminal/oversized-word limits.
- [ ] Run `gofmt` on changed Go files, `go test ./cmd/define/...`,
  `go vet ./cmd/define/...`, and `git diff --check`. Check the narrow-screen test
  observes every word rather than merely the presence of added newlines.
- [ ] Commit with issue reference and model co-author trailer, tick issue and
  plan tasks, and invoke `sdlc close --issue 55 --verified '<actual evidence>'`.
  The close gate owns the fresh-context review. Resolve blocking findings before
  publishing through `sdlc pr` and `sdlc merge`.

## Approval

Pending operator approval under AGENTS.md §2: this change spans more than three
files and is expected to exceed 100 lines including regression tests.

## Revisions

### 2026-09-13 — Fresh-context plan review

No architectural blockers. Add two explicit verification requirements:

- Task 2 must include a partial stream ending in `ErrMalformed` or `ErrRequest`
  without a trailing newline. Interrupted/truncated paths already append a
  newline, so they alone cannot prove that the deferred flush preserves the
  unfinished final word. Assert both its output and the diagnostic.
- Retain terminal sizing's sub-`minWrapWidth` normalization to zero and test
  that boundary. The narrow live-screen regression must use a width at least
  `minWrapWidth`; width zero alone does not cover this policy.

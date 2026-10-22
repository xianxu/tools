# Boundary Review — tools#85 (milestone M2)

| field | value |
|-------|-------|
| issue | 85 — llm: public seam for echo — multi-turn, tools, streamed tool-call deltas, cancel |
| repo | tools |
| issue file | workshop/issues/000085-llm-public-seam-for-echo-multi-turn-tools-streamed-tool-call-deltas-cancel.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 5e284c7d6e4f6705f15b35cbb69e720180c35c25..248c58864b4c19ce4bd4e2f860eca3e0f7d79f31 |
| command | sdlc milestone-close --issue 85 --milestone M2 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | checkout |
| timestamp | 2026-10-21T21:31:26-07:00 |
| verdict | REWORK |

## Review

Reviewing the M2 seam diff — reading core `llm` files now, then the anthropic client, wire, fake and tests.

Checking how the SDK accumulator builds in-flight tool_use blocks, to test whether a cancelled partial Response leaks a half-streamed call through `Blocks`.

```verdict
verdict: REWORK
confidence: high
```

M2 delivers almost everything it claims. Messages, Tools, ToolCalls, the `Delta` stream, `ErrCanceled`, `BeforeAttempt` and `Usage.Attempts` are all in place. define now uses `TextOnly`, and the single-turn hash is pinned to the committed golden. `go build`, `go vet` and `go test -race ./llm/... ./conformance/...` all pass. The `cmd/define` failures (`TestLanguage*` "operation not permitted", and `TestPlanCitesTestsThatExist` on plan 000082) also fail at the base commit, so this range did not cause them. One defect blocks SHIP. When a stream is cancelled, truncated or malformed partway through a tool call, the partial `Response` still contains the half-streamed call in `Blocks` and `Message()`. The contract says such calls are "dropped, not half-reported". I reproduced it on the fake. The fix is cheap and local to `partial()`.

## 1. Strengths
- **One owner for how a call ends.** `outcome()` (`llm/attempt.go`) checks rejection, then caller cancel, then the transport's own mapping. `errBudget` set via `WithTimeoutCause` (`llm/errors.go`) is a clean way to tell the package's own Timeout from the caller's deadline. Without it, the auto client's bounded context would read as a caller cancel.
- **The abort design works.** `callState` rides the context and the middleware cancels a derived context on rejection. This sidesteps the SDK retrying a middleware that returns no response. `TestBeforeAttempt` checks it for both Complete and Stream: hook `Number`s, wire attempt count and `Usage.Attempts` on every row, including "rejected on the retry" → 1.
- **`deltaState` is pure and tested accordingly.** It is fuzzed over interleaved fragments (`FuzzDeltaAccumulator`). The parse-once bound is counted (`TestDeltaForLinear` via `validations`). Ownership is tested (`TestDeltasOwnedByReceiver`), and the End copy is kept apart from `done`.
- **`TestSingleTurnHashUnchanged`** (`llm/golden_schema_test.go`) ties the hash to the committed golden's bytes, so re-recording a golden alongside a renderer change can't hide a moved cassette key.
- **The define audit is real.** `stopMeans` gains an explicit `ErrCanceled` arm with a test (`cmd/define/background.go:451`). `helpCause` is tested for `ErrCanceled` + `DeadlineExceeded`. A plain `Canceled` can't reach it because of the `ctx.Err() == nil` guard at `practice_help_client.go:147`.

## 2. Critical findings
- **A half-streamed tool call is still reported in the partial `Response.Blocks` and `Message()`** (`llm/anthropic.go`, the `partial` closure in `Stream`).
  - `partial()` overrides `ToolCalls` with `deltas.done` but builds `Blocks` from the SDK accumulator, which adds a tool_use block at `content_block_start`.
  - Reproduced on the fake with a `toolReply()` that stalls, cancelled after the first End. Result: `ErrCanceled`, 1 ToolCall, but `Blocks[2]` is `tool_use tu_2` with `ToolCall.Input = "{\"cit"` (invalid JSON). Its `Raw` is `{"id":"tu_2","input":{},…}`, which disagrees with that Input. `Message()` carries 3 blocks.
  - A consumer that keeps the partial assistant turn would send back a tool_use with no tool_result, which the provider rejects with a 400. A consumer reading `Blocks` sees a fabricated call.
  - The same `partial()` serves the `ErrTruncated` (stall/salvage) and `ErrMalformed` paths.
  - **Fix:** in `partial()`, keep a `tool_use` block only if its ID is in `deltas.done`. That covers both still-open calls and the call that failed validation.
  - **Regression test:** extend `TestCancelOutcomes` "mid tool call after one completed" so every tool_use Block's ID must be in `resp.ToolCalls` and its Input must be `json.Valid`. Add the same check on a malformed-End stream.

## 3. Important findings
- **ARCH-ORDER: two Suite obligations can only pass on one ordering of a race the plan defines as legal** (`llm/llmtest/suite.go`, "a cancelled stream…" and "concurrent streams…").
  - The plan's state table says cancel racing normal completion ends as "whichever the read loop saw first". Both obligations cancel from inside a callback and then require `ErrCanceled`.
  - Against the fake this is deterministic, because `FinishRelease` holds the stream (`llm/suite_test.go:30-35`). Against a live service (`conformance_test.go:41`, run in M3), the victim's short echo stream may already be fully buffered. A translating proxy is especially likely to buffer, and then `Stream` correctly returns success and the obligation fails.
  - **Fix:** on the live path, accept either `ErrCanceled` or a complete, self-consistent success for the victim. Keep the strict assertion only where the double holds the stream. Alternatively, give the Suite a way to say whether the client under test can hold a stream.

## 4. Minor findings
- `Complete` doesn't check that tool_use input is a JSON object; `Stream` fails with `ErrMalformed` on the same input. The two paths handle the same input differently (`buildResponse` vs `deltaState.next`).
- ARCH-DRY: the "empty input → `{}`" default appears four times: `blockJSON`, `buildResponse`, `deltaState.next`, and `llmtest.toolInput`.
- In `serveToolStream`, the Stall check is a substring match on `"index":N`, which also matches `"index":1N` once there are 10 or more blocks.
- The middleware treats any hook error returned while `req.Context().Err() != nil` as a cancel. This includes a real policy rejection that coincides with the caller's cancel, so production can only reach the "rejection beats cancel" order when the rejection happens first. This is disclosed in Revisions; the plan text in the API section should say so too.
- The ticked Tasks 2–4 quote commit subjects that don't exist. The work landed as `46a25c8` "widen the seam" plus a tests commit. This is the 2nd finding in family `tracker-checkbox-state-diverges` (details in the findings block).
- The docs gate is not met inside this range: `atlas/llm.md` still documents `Stream(func(string))` and has no `ErrCanceled` or `BeforeAttempt`. The plan explicitly schedules the rewrite in M3 (Task 6). It's recorded here so the close gate checks for it.

## 5. Test coverage notes
- The cancel table covers four cancel points; `TestBeforeAttempt` covers 5 cases × 2 paths. `TestOutcomeOrderUnit` tests `outcome()` on its own, but the "rejection beats a later cancel" case never goes through the middleware.
- Nothing asserts `Blocks` on a partial response, which is why the Critical finding shipped.
- `BeforeAttempt` was dropped as a Suite obligation, with a reason given in Revisions. Its only live coverage is the M3 check, so the gateway's main dependency is fake-only until then.

## 6. Architectural notes for upcoming work
- **ARCH-DRY:** pass, apart from the minor `{}`-default duplication.
- **ARCH-PURE:** pass. `wire.go` and `delta.go` are pure and table- or fuzz-tested; the IO stays in `anthropic.go` and the middleware.
- **ARCH-PURPOSE:** pass. All the M2 surface is delivered; the `BeforeAttempt` Suite deferral is disclosed and justified.
- **ARCH-MOCK:** pass. The fake is stateful and runs behind the real client over the wire.
- **ARCH-CONSTRAINTS:** pass. O(E+B) is enforced by a counted test, and a blocking hook is bounded by the context.
- **ARCH-SECURE:** flag. The partial `Blocks` carry a fabricated tool_use (the Critical finding). Consumer `Raw` is checked with `json.Valid`.
- **ARCH-ORDER:** flag. See the Important finding. Otherwise `outcome()` is a good single owner.
- **ARCH-FUNERAL:** pass. M2 creates nothing durable: `callState` dies with the call's context and the fake is in-memory.
- **ARCH-EVOLVE:** pass. Render sections are added only when set, and the golden hash is pinned.
- **For M3:** forced `tool_choice` (`echoRequest`'s `ToolChoice: "echo"`) together with the adaptive thinking that `effective()` turns on for openai/antigravity may be refused by live providers. Check this before certifying the live Suite.

## 7. Plan revision recommendations
- Add to Revisions: "Partial Responses (cancel/truncate/malformed) drop in-flight tool_use blocks from Blocks/Message() as well as ToolCalls; tested in TestCancelOutcomes."
- Add to Revisions: the live Suite's cancel obligations accept either side of the cancel-vs-completion race; the strict side is only checked where the double holds the stream.
- Reword the ticked Task 2–4 commit lines to name the commits that actually shipped (`46a25c8`, `33f3c3a`).

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: partial-response-reports-incomplete-state
    title: |
      Cancelled/truncated/malformed stream's partial Response keeps the half-streamed tool_use in Blocks and Message()
    detail: |
      llm/anthropic.go Stream partial() replaces ToolCalls with deltas.done but builds Blocks from the SDK accumulator, which adds tool_use at content_block_start. Reproduced on the fake: Blocks[2] tool_use tu_2 has ToolCall.Input "{\"cit" (invalid JSON), Raw input {}, and Message() echoes it back, which breaks the "dropped, not half-reported" contract. Fix: keep a tool_use block only if its ID is in deltas.done. Regression: assert Blocks in TestCancelOutcomes and on a malformed-End stream.
  - id: new
    severity: Important
    kind: defect
    family: test-observes-one-interleaving
    title: |
      Suite cancel obligations (single cancel and ConcurrentStreams victim) assert ErrCanceled on a race the plan declares legal (ARCH-ORDER)
    detail: |
      llm/llmtest/suite.go cancels inside the callback and requires ErrCanceled. The fake holds the stream with FinishRelease, so that run is deterministic. Live (conformance_test.go:41, M3), a buffered short echo stream can legitimately finish first, and Stream returns success. Fix: on the live path accept ErrCanceled or a complete, consistent success; keep the strict check where the double holds the stream.
  - id: new
    severity: Minor
    kind: defect
    family: tracker-checkbox-state-diverges
    title: |
      Ticked plan Tasks 2-4 quote commit subjects that were never made (work landed as 46a25c8 and 33f3c3a)
    detail: |
      This is the 2nd finding in family tracker-checkbox-state-diverges. Rule: a ticked plan item may quote only artifacts that exist at the head, whether test names, commit subjects or milestone states. Extend the TestPlanCitesTestsThatExist guard to quoted commit subjects, or stop quoting planned commit names in task items. Prevalence: 3 of 3 M2 implementation tasks.
  - id: new
    severity: Minor
    kind: defect
    family: inconsistent-validation-across-paths
    title: |
      Complete accepts non-object tool_use input that Stream rejects with ErrMalformed
    detail: |
      buildResponse does not apply the JSON-object check that deltaState.next enforces at End, so the same tool_use input succeeds through Complete and fails through Stream.
  - id: new
    severity: Minor
    kind: defect
    family: docs-deferred-to-later-milestone
    title: |
      atlas/llm.md still documents the pre-M2 seam (func(string) Stream, no ErrCanceled or BeforeAttempt)
    detail: |
      M3 (Task 6) explicitly schedules the rewrite. Recorded so the close gate checks it; it does not block M2.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"394b0419-49e1-4d99-a9cd-e7752a97b1b1","repository":"github.com/xianxu/tools","issue_id":"000085","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/d10f3a74e7609fba","reviewed_head":"248c58864b4c19ce4bd4e2f860eca3e0f7d79f31","slot":"tools:1","requested_at":"2026-10-22T04:20:59.954303Z"},"phase":"finished","pid":16776,"started_at":"2026-10-22T04:21:00.047884Z","finished_at":"2026-10-22T04:31:26.326439Z","session_id":"311ecca3-7410-46da-b1fb-c132f79e41b4","outcome":"returned"}] -->

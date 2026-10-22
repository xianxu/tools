---
collaboration-mode: discussion
---

# llm: public seam for echo — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish `internal/llm` as `github.com/xianxu/tools/llm` (v0.1.0) and widen its provider-neutral seam
to multi-turn history, tool definitions, streamed tool-call deltas interleaved with text, caller cancellation
and a per-attempt admission hook. The wire fake, cassettes, goldens and conformance suite grow with it.

**Architecture:** The seam keeps its two methods. `Stream`'s callback changes from `func(string)` to
`func(Delta)`, and an adapter keeps define's text-only callers on one line each. `Request` gains `Messages`
and `Tools`; `Response` gains `ToolCalls`. Everything still crosses the Anthropic Messages wire through the
SDK. Retries stay in the transport (the SDK's loop). A `Config.BeforeAttempt` hook is installed as SDK
middleware, which runs once per physical HTTP attempt inside that loop, so a consumer can count or throttle
attempts without writing a retry loop of its own.

**Tech Stack:** Go 1.26, `github.com/anthropics/anthropic-sdk-go` v1.66.0, `net/http/httptest`.

**Consumers:** `cmd/define` (in-repo, migrates in M1); echo#4's gateway actor (echo:1), which imports
`llm` and `llm/llmtest` as an ordinary module at `v0.1.0` and adds no provider, client or fake copy of its own.

---

## Public API (for review by ops:0 and echo:1 before change-code)

### Versioning

The package doc states that v0.x may change incompatibly at a minor-version bump, so consumers pin an exact
version. `v0.1.0` is tagged only after the change lands on main (ops:0 markup).

### Import paths

| path | was | holds |
|---|---|---|
| `github.com/xianxu/tools/llm` | `internal/llm` | seam, config, taxonomy, typed tasks |
| `github.com/xianxu/tools/llm/llmtest` | `internal/llm/llmtest` | wire fake, cassettes, goldens, `Suite` |
| `github.com/xianxu/tools/conformance` | `internal/conformance` | skip-or-fail policy for live checks |

`conformance` has to move too: `llmtest/reachable.go` imports it, and Go refuses an `internal/` import
across module boundaries, so a public `llmtest` cannot compile in echo without it. The package is 116 lines
and holds no domain logic. The alternative, copying `SkipOrFail` into `llmtest`, breaks the single-owner
rule that `internal/conformance/guard_test.go` enforces.

**One module, not a nested one.** `llmtest` stays at `github.com/xianxu/tools`, tagged `v0.1.0` at the root.
Module-graph pruning (Go ≥ 1.17) means echo's `go.sum` picks up only the packages `llm` and `llmtest`
actually import (the Anthropic SDK and the standard library). It does not pick up define's dependencies.
A nested `llm/go.mod` would give a smaller graph at the cost of a second module to version, and nothing
needs that yet.

**Repository rule update.** `AGENTS.local.md`'s carve-out names `internal/llm`. It is reworded: an external
service transport may be *public* when another repository consumes it, and the operator decision for this
change is recorded alongside the original (tools#11).

### The seam

```go
type Client interface {
    Complete(ctx context.Context, r Request) (Response, error)
    // Stream delivers events as they arrive and still returns the final Response.
    // onEvent runs synchronously on the calling goroutine and is never called
    // after Stream returns. It may be nil.
    Stream(ctx context.Context, r Request, onEvent func(Delta)) (Response, error)
}

// TextOnly adapts a text callback: answer text is forwarded and every other
// event dropped. It is define's whole migration: Stream(ctx, r, llm.TextOnly(fn)).
func TextOnly(fn func(string)) func(Delta)
```

Two methods, not a third `StreamEvents`: one streaming path means one stall timer, one salvage branch and
one set of suite obligations. The implementers that change signature are `autoClient.Stream` (`auto.go`),
define's `activityClient` (`llm_activity.go`), and the test doubles in `cmd/define`
(`answer_termination_rows_test.go`, `background_loop_test.go`, `answer_rows_test.go`, `llm_activity_test.go`,
`askrun_test.go`, `ask_language_test.go` ×2). `practice_help_client.go` goes through `llm.Run` and is
unaffected.

### Request

```go
type Request struct {
    Task      string // as today: names goldens, keys cassettes; NOT sent
    Model     string
    Effort    string
    MaxTokens int64

    System string

    // Messages is the conversation so far, oldest first. Prompt, when
    // non-empty, is appended as a final user text turn, so every existing
    // single-turn caller keeps working unchanged. A request with neither is
    // refused with ErrRequest before anything is sent.
    Messages []Message
    Prompt   string

    // Tools the model may call. ToolChoice is "" (provider default: auto),
    // "auto", "any", "none", or a tool's Name to force it.
    Tools      []Tool
    ToolChoice string

    Schema map[string]any
}

type Role string

const (
    RoleUser      Role = "user"
    RoleAssistant Role = "assistant"
)

// Message is one turn. Content reuses Block, so an assistant turn can be fed back
// exactly as it arrived: resp.Message() returns it ready to append.
type Message struct {
    Role    Role
    Content []Block
}

// Constructors are the only supported way to build outgoing blocks.
func Text(s string) Block
func ToolResult(callID, content string, isError bool) Block

type Tool struct {
    Name        string
    Description string
    InputSchema map[string]any // JSON Schema; nil means {"type":"object"}
}
```

**Name collision rule (ops:0 markup).** `ToolChoice` shares one string between keywords and tool names, so a
`Tool` named `auto`, `any` or `none` is refused with `ErrRequest` before anything is sent. A `ToolChoice` that
is neither a keyword nor a declared tool's name is refused the same way.

### Block, Response and tool calls

```go
type Block struct {
    Type      string          // "text" | "thinking" | "tool_use" | "tool_result" | future
    Text      string          // text / thinking content, or a tool_result's content
    Signature string          // thinking only; round-trips unchanged
    ToolCall  *ToolCall       // set when Type == "tool_use"
    CallID    string          // set when Type == "tool_result"
    IsError   bool            // tool_result only
    Raw       json.RawMessage // as received; echoed back verbatim when present
}

type ToolCall struct {
    ID    string
    Name  string
    Input json.RawMessage // complete JSON object; "{}" when the model sent no input
}

type Response struct {
    // ... every existing field, unchanged ...
    ToolCalls []ToolCall // every tool_use block, in arrival order
}

// Message returns the assistant turn to append to the next Request.Messages.
// It carries every block's Raw, so thinking signatures survive the round trip.
func (r Response) Message() Message
```

**Outgoing rule:** a block with `Raw` is sent as exactly those bytes (the byte-for-byte thinking echo the
current `Block` doc already promises). A block without `Raw` is rendered from its typed fields. `Raw` is
opaque provider data: a consumer must not author it, only carry it from a `Response` back to the next
`Request`.

`Stop == "tool_use"` stays a success (it already is in `classifyStop`). The consumer runs the tools, appends
`resp.Message()` and a user turn of `ToolResult` blocks, and calls again.

### Stream events

```go
type DeltaKind int

const (
    DeltaText          DeltaKind = iota + 1 // Text: answer text
    DeltaToolCallStart                      // Index, Call.ID, Call.Name
    DeltaToolCallInput                      // Index, Text: a fragment of the input JSON
    DeltaToolCallEnd                        // Index, Call: complete, Input parsed and valid
)

type Delta struct {
    Kind  DeltaKind
    Index int       // content-block index; ties a call's Start/Input/End together
    Text  string
    Call  *ToolCall // Start and End only
}
```

Guarantees, each one a `Suite` obligation:
1. Events arrive in wire order. Text and tool-call events interleave as the provider sends them, and a
   call's fragments never cross into another call's `Index`.
2. Each `Start` gets exactly one `End`, unless the stream fails or is cancelled first. The `End`'s `Input`
   equals the `ToolCall` that `Response.ToolCalls` reports.
3. Thinking is never forwarded (as today).
4. Concatenating every `DeltaText` reproduces `Response.Text`, and concatenating a call's `DeltaToolCallInput`
   fragments reproduces its `Input` bytes.

An `End` whose accumulated input does not parse as a JSON object fails the call with `ErrMalformed`. The
partial `Response` is still returned, as it is for truncation.

### Ownership of delivered values (echo:1 review)

Every `Delta`, and everything it points to (`Call`, `Call.Input`), is freshly allocated for that event and never
touched by the transport again. The receiver owns it and may keep it or enqueue it to another goroutine without
copying. The same holds for a returned `Response` (`Blocks[i].Raw`, `ToolCalls[i].Input`): nothing aliases an
SDK buffer. Tested: retain every `Delta` from a stream, finish the stream, and compare the retained values to a
deep copy taken inside the callback.

### Concurrency (echo TL review)

The package doc states it: a `Client` from `New`, including the auto-discovering one, and an `llmtest.Fake` are
safe for concurrent use by any number of goroutines. Calls share no per-call state. Each call has its own stall
timer, accumulator and callback, so one call's events never reach another's callback, and cancelling one
call's `ctx` affects only that call. The auto client's single discovery flight is already shared under a mutex
(`auto.go`). The `Suite` obligation `ConcurrentStreams`: N = 8 streams on one client, each scripted with a
distinct answer and tool call; every callback sees only its own events; cancelling one mid-stream yields
`ErrCanceled` for it alone while the others complete. It runs under `-race`.

### Cancellation

The caller cancels by cancelling `ctx`. That's the only control, and it covers both `Complete` and `Stream`.

- **Prompt return.** The SDK aborts the HTTP request (or its backoff wait). `Stream` returns once the read
  loop sees the closed body, and no `onEvent` call happens after it returns.
- **New sentinel `ErrCanceled`.** Today a cancelled context reads as `ErrUnavailable` (before the first
  frame) or `ErrTruncated` (after it). Both are wrong for a caller that cancelled on purpose: one says
  "outage, back off" and the other says "skip". If the caller's own `ctx` is done (`ctx.Err() != nil` on the
  context passed in, checked before mapping), the error wraps both `ErrCanceled` and `ctx.Err()`, so
  `errors.Is(err, context.Canceled)` and `errors.Is(err, context.DeadlineExceeded)` both work. The package's
  own `Timeout` and `StallAfter` still map as today, because they fire on a derived context and the caller's
  `ctx` stays live.
- **Partial result.** Under `ErrCanceled` the returned `Response` holds the text so far and every tool call
  that reached `End`. Calls still in flight are dropped, not half-reported.
- `Usage` on a cancelled call is whatever the provider had reported by then (often only input tokens).

### Retries and admission

- **The transport owns all retries**, as today: the SDK's loop, `MaxRetries = 2`, on 408/409/429/5xx,
  honouring `retry-after`, with backoff bounded by `Timeout` and the caller's `ctx`. A stream is retried only
  before its 200 arrives. Once frames flow there is no retry: a later failure is `ErrTruncated`, and a
  consumer that wants a second answer issues a new logical call.
- **A consumer writes no retry loop.** After `ErrUnavailable` the attempts are already spent. Re-issuing is a
  product decision (show "model unavailable", try later), not a loop.
- **New `Config.BeforeAttempt func(ctx context.Context, a Attempt) error`.** It's installed with
  `option.WithMiddleware`, which the SDK calls inside its retry loop, so it runs once per **physical** HTTP
  attempt, retries included. It may block, which is how a throttle works; the wait is bounded by `ctx`, the
  same deadline that covers backoff. A non-nil return aborts the call without sending, maps to
  `ErrUnavailable` wrapping the hook's error (`errors.Is(err, yourAdmissionErr)` holds, so the gateway can tell
  its own rejection from an outage), and is not retried. Rejecting immediately, without blocking, is the
  expected use for a gateway that sheds load. Model discovery (`GET /v1/models`) does not
  pass through it.

  ```go
  type Attempt struct {
      Task   string // Request.Task
      Model  string // effective model
      Number int    // 0 for the first attempt, 1.. for retries
  }
  ```
- **How a rejection aborts without a retry.** The SDK retries when a middleware returns no response
  (`requestconfig.go` `shouldRetry`: `res == nil` means retry), so returning the hook's error directly would buy
  backoff plus two more hook calls. Instead each call carries a private `callState` (the hook error and the
  attempt count) in its context, and its SDK call runs on a context derived from the caller's with its own
  cancel. On a rejection the middleware stores the error, cancels that derived context, and returns. The SDK
  loop then exits at its `ctx.Err()` check, before backoff. The client checks `callState.hookErr` **first**,
  before the caller-ctx check that yields `ErrCanceled` and before `mapError`, and returns
  `fmt.Errorf("%w: admission: %w", ErrUnavailable, hookErr)`. So `errors.Is` holds for both, and
  `Usage.Attempts` is the admitted count, 0 for a first-attempt rejection. One owner decides how a call ended,
  in a fixed order: hook rejection, then caller cancel, then the package's stall and timeout, then status
  mapping. The order is pinned by a test where the hook rejects while the caller's ctx is also cancelled.
- **What this means for echo's gateway.** It rate-limits **logical** calls for fairness and admission between
  actors. It counts or throttles **physical** attempts against provider limits through `BeforeAttempt`. A new
  `Response.Usage.Attempts int` field counts **dispatched** HTTP attempts only, the ones the hook admitted and
  that went on the wire. Hook invocations that returned an error don't count, so a rejection on the first
  attempt reconciles to `Attempts == 0`, and a rejection of retry 2 after one dispatch gives `1`. It is set on
  **every** return path, errors included: a failed call returns a `Response` whose only populated field may be
  `Usage.Attempts`.

### llmtest controls (what echo's tests drive)

Existing controls, now public at `llm/llmtest`:

- `NewFake(t) *Fake` is an httptest server. A `Config{BaseURL: f.URL, APIKey: "test"}` points a real client at
  it.
- `f.Script(match string, replies ...Reply)` serves the replies in order to requests whose last user turn
  contains `match`. `match` is tested against text **and** tool-result content, which is new.
- `Reply` covers transport shapes: `Status` (any HTTP status, so 429/529 drive the retry path), `Stall`
  (silence after the first text delta), `StallEarly` (silence before any frame), `JunkFrame` (an undecodable
  frame mid-stream), `SplitText`, and the barriers `Started`/`Release`, `TextStarted`/`TextRelease`,
  `AfterText`/`FinishRelease`. Those barriers are the deterministic hooks for cancel tests: hold the response,
  cancel `ctx`, assert the return.
- `f.Requests() []Recorded` returns every request as the server saw it, **one per physical attempt**, so a
  retry test counts attempts. `Recorded` has `System()`, `Prompt()`, `Schema()` and `Streaming()`, plus the
  raw `Body` and `Headers`.

New in this issue:

- `Reply.ToolCalls []llm.ToolCall` serves invented `tool_use` blocks (`stop_reason: tool_use`). On the stream
  path each input is chunked across several `input_json_delta` frames, and `Reply.Text` can come before them,
  so interleaving is exercised. Inventing them is allowed under the fake's rule (block shape is protocol, not
  judgment). Real tool-use content still comes only from committed captures (M3).
- `Recorded.Messages() []RecordedMessage` (role plus typed content), `Recorded.Tools()` and
  `Recorded.ToolChoice()`, so a test asserts what history and tool definitions actually went on the wire.
- `f.Attempts() int` is shorthand for `len(f.Requests())` on the messages path.
- `llmtest.Suite` gains the obligations above (event ordering, End = ToolCalls, cancel → `ErrCanceled` with
  no late callback, a multi-turn round trip carrying thinking `Raw`, `BeforeAttempt` called once per attempt).

### Compatibility

| change | define | other code |
|---|---|---|
| import path | mechanical rewrite in M1 | — |
| `Stream` callback type | `llm.TextOnly(fn)` at each call site; decorators change signature | a custom `Client` implementation must update |
| `Request.Messages`/`Tools` | unused; zero values reproduce today's wire bytes exactly (golden-pinned) | — |
| `ErrCanceled` | define's cancel paths see `ErrCanceled` where they saw `ErrUnavailable`; each site is audited in M2 | — |
| `renderRequest` | renders Messages/Tools only when non-empty, so every committed golden and cassette hash stays the same | — |

The last row is load-bearing. A single-turn request must hash exactly as it does today, or every committed
cassette misses. A test pins one existing golden's hash across the change.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Request` / `Block` / `Response` / `Usage` | `llm/llm.go` | modified |
| `Message` / `Role` / `Tool` / `ToolCall` / `Text` / `ToolResult` | `llm/llm.go` | new |
| `Delta` / `DeltaKind` / `TextOnly` | `llm/delta.go` | new |
| `Attempt` | `llm/llm.go` | new |
| `ErrCanceled` | `llm/errors.go` | new |
| `renderRequest` (Messages/Tools sections) | `llm/render.go` | modified |
| `wireMessages` (Request → `[]anthropic.MessageParam`) | `llm/wire.go` | new |
| `deltaFor` (SDK stream event → `[]Delta`) | `llm/wire.go` | new |
| `buildResponse` (tool_use blocks) | `llm/anthropic.go` | modified |

- **wireMessages**: a pure translation of `Messages` + `Prompt` + `Tools` + `ToolChoice` into SDK params.
  Raw-first per block. Unit-tested from table inputs with no server.
  - **DRY rationale:** the one place the neutral types meet the Anthropic shape. `params()` calls it and so
    does nothing else.
- **deltaFor**: maps one SDK stream event to zero or more `Delta`s. It's pure, so the interleaving and Index
  rules are unit-tested on fabricated events and pinned end to end on the wire.
- **Delta**: the provider-neutral stream event.
  - **Future extensions:** `DeltaThinking`, only if a consumer ever needs it (none does).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `anthropicClient.Stream/Complete` | `llm/anthropic.go` | modified | Anthropic Messages API via SDK |
| `attemptMiddleware` | `llm/anthropic.go` | new | SDK retry loop (`option.WithMiddleware`) |
| `Fake` (tool_use, history, attempts) | `llm/llmtest/fake.go` | modified | httptest Anthropic wire |
| `Cassette` | `llm/llmtest/cassette.go` | modified | `http.RoundTripper` |
| `Suite` | `llm/llmtest/suite.go` | modified | fake + live service |
| `SkipOrFail` | `conformance/conformance.go` | modified | `testing.T` + env |

- **Fake state model:** a FIFO of scripted replies per match string, a record of every physical request, and
  the model catalog. New: a tool_use reply shape and history-aware matching. It's reused, not replaced.

---

## Milestones

- [ ] M1 — Publish: move `internal/llm` → `llm`, `internal/llm/llmtest` → `llm/llmtest`,
      `internal/conformance` → `conformance`; rewrite imports; update `AGENTS.local.md`, atlas, scripts.
      No behavior change; the full suite passes unchanged.
- [ ] M2 — Widen the seam: Messages, Tools, ToolCalls, Delta stream, ErrCanceled, BeforeAttempt,
      Usage.Attempts; the fake and Suite grow with each; define migrated to `TextOnly`; goldens unchanged for
      single-turn requests.
- [ ] M3 — Prove against reality: record real tool-use and multi-turn captures/cassettes via
      `scripts/llm-probe.sh`; live conformance for tool streaming and cancel; atlas `llm.md` rewritten;
      land; tag `v0.1.0`; `go get github.com/xianxu/tools@v0.1.0` verified from a scratch module.

### Task 1 (M1): move packages

**Files:** `git mv internal/llm llm`, `git mv internal/conformance conformance`; every `*.go` importing
them; `internal/conformance/guard_test.go`'s path assumptions; `scripts/llm-probe.sh`; `Makefile`
targets naming `./internal/llm/...`; `atlas/llm.md`, `atlas/index.md`; `AGENTS.local.md` carve-out text.

- [x] `git mv` both trees; rewrite imports with
      `gofmt -r` / `sed 's#github.com/xianxu/tools/internal/llm#github.com/xianxu/tools/llm#'` (and conformance).
- [x] `grep -rn "internal/llm\|internal/conformance" --exclude-dir=.git --exclude-dir=workshop .` → only
      historical prose remains; fix every live reference (scripts, Makefile, atlas, guard tests).
- [x] `go build ./... && go vet ./...`; `go test ./llm/... ./conformance/... ./cmd/define/...` all pass.
- [x] External-consumer check: a scratch module under `$TMPDIR` with
      `replace github.com/xianxu/tools => <slot path>` importing `llm` and `llmtest` builds and runs
      `llmtest.Suite` against `NewFake`. This proves no `internal/` import leaks.
- [x] Commit `#85 M1: publish llm, llmtest and conformance as public packages`; `sdlc milestone-close --milestone M1`.

Planned test names below are unquoted and marked "(planned)". The repo guard
`TestPlanCitesTestsThatExist` holds that a backticked test name claims the test exists, so each one gets its
backticks once it is written.

### Task 2 (M2): neutral types + wire translation (pure, TDD)

**Files:** `llm/llm.go`, `llm/wire.go`, `llm/wire_test.go`, `llm/render.go`, `llm/render_test.go`.

- [ ] TestWireMessages (planned): a table over the Request shapes in the API section (Prompt-only equals today's
      params; Raw-first blocks; tool_result; tools; every ToolChoice form; the refusals: empty request, a tool
      named with a keyword, an unknown ToolChoice). Strategy: one table, asserted on the marshalled SDK
      params, with no server.
- [ ] TestSingleTurnHashUnchanged (planned) pins the `schema-veto-verdict` request's `RequestHash` to its pre-change
      value; TestRenderMultiTurn (planned) checks the new sections are deterministic.
- [ ] Implement; `params()` delegates to `wireMessages`. Commit `#85 M2: neutral message, tool and block types on the wire`.

### Task 3 (M2): tool calls in responses + Delta stream

**Files:** `llm/delta.go`, `llm/delta_test.go`, `llm/anthropic.go`, `llm/response_test.go`, `llm/auto.go`,
`cmd/define/llm_activity.go` and the test doubles listed in the API section.

- [ ] FuzzDeltaAccumulator (planned): from a seed of captured tool-use frames (M3 adds the real ones; M2 uses fake
      frames), the fuzzer splits each call's input at arbitrary points and interleaves events from several
      Index values. It asserts guarantees 1, 2 and 4 and that `End.Input` equals `ToolCalls`, and that the
      result never panics. TestDeltaForLinear (planned) counts `json.Valid` calls at 8 and 16 fragments (one per call).
- [ ] TestBuildResponseToolUse (planned), TestResponseMessageRoundTripsRaw (planned), TestTextOnly (planned),
      TestDeltasOwnedByReceiver (planned).
- [ ] Implement; switch `Stream` to `func(Delta)`; migrate every implementer and call site
      (`llm.TextOnly(fn)`); `go test ./cmd/define/...` green. Commit `#85 M2: tool calls and interleaved stream deltas`.

### Task 4 (M2): cancel and attempts

**Files:** `llm/errors.go`, `llm/anthropic.go`, `llm/attempt.go`, `llm/anthropic_test.go`; define match sites.

- [ ] TestCancelOutcomes (planned): a table over the cancel points (held at `Started`; after the first text; caller
      deadline), driven by the fake's barriers. Each expects `ErrCanceled` plus the matching ctx error, the
      partial content defined above, and no callback after return. The package's own Timeout and StallAfter
      keep their current mapping.
- [ ] TestBeforeAttempt (planned): a table over scripted statuses × hook verdicts. It asserts the hook's Numbers,
      `f.Requests()` count, `Usage.Attempts` on every return path, `errors.Is(err, hookErr)`, and that no request
      follows a rejection. TestOutcomeOrder (planned): the hook rejects while the caller's ctx is cancelled, and the
      hook's error wins.
- [ ] Implement `callState` + `attemptMiddleware` and the outcome order.
- [ ] Audit **every** `ErrUnavailable` match site in define: `ask.go:228` (already checks `ctx.Err()` first, at
      `:213`), `practice_help_client.go:195`, and `background.go:451`, whose loop is cancelled at shutdown and
      would otherwise fall to `default`. Each site gets an explicit `ErrCanceled` arm or a test showing it can't
      see one. Commit `#85 M2: ErrCanceled and per-attempt admission hook`.

### Task 5 (M2): fake + Suite growth

**Files:** `llm/llmtest/fake.go`, `fake_test.go`, `suite.go`.

- [ ] TestFakeToolCalls (planned) (JSON and stream; input chunked across ≥2 frames, after Text), TestFakeMatchesToolResult (planned),
      TestRecordedAccessors (planned), and the `misapplied` matrix extended to cover ToolCalls.
- [ ] `Suite` obligations from the API section, including `ConcurrentStreams` (8 streams, one cancelled alone),
      run against the fake under `-race`. The package docs for `llm` and `llmtest` state concurrent-use safety.
- [ ] Commit; `sdlc milestone-close --milestone M2`.

### Task 6 (M3): captures, conformance, docs, tag

- [ ] `scripts/llm-probe.sh record` gains a tool-use capture (one tool, forced) and a two-turn tool-result
      continuation (thinking echoed); README table rows; captures replayed in `captures_test.go`.
- [ ] Conformance (`-tags conformance`): `Suite` already runs live; add a live tool-streaming check and a live
      mid-stream cancel check; drift test asserts `input_json_delta` still arrives in >1 frame. Run against
      the auto-selected provider and report which providers were certified.
- [ ] Package doc (`llm/llm.go`) states the v0.x compatibility rule: minor bumps may break; pin a version.
- [ ] Rewrite `atlas/llm.md` for the public path, the new seam sections, cancellation and admission.
- [ ] Close, merge; after landing: `git tag v0.1.0 <landed main sha>` + push (tag push confirmed with the
      hub first); scratch module `go get github.com/xianxu/tools@v0.1.0` builds against `llm` + `llmtest`.

## State design table

Actions a consumer can take: start a call, cancel it (ctx), let it time out, reject an attempt (hook).

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| caller cancels before first frame | `ErrCanceled`, empty Response | none / re-issue | nothing sent back |
| caller cancels mid-stream | `ErrCanceled`, partial text + completed calls | re-issue if wanted | in-flight tool call's partial input |
| caller deadline | `ErrCanceled` + `DeadlineExceeded` | same as cancel | same |
| transient failures exhausted | `ErrUnavailable`, `Usage.Attempts` = 3 | back off (product decision) | the call |
| hook rejects attempt | `ErrUnavailable` wrapping hook error | admit later | the call |
| stream dies / stalls after frames | `ErrTruncated`, partial | new logical call | rest of answer |
| tool input JSON invalid at End | `ErrMalformed`, partial | new logical call | that tool call |
| cancel races normal completion | whichever the read loop saw first; never both | — | none |

Not covered: resuming a cut stream (the Messages API has no resume). Each `Request` is stateless; history is
the consumer's to keep.

## Hot-path cost

Per stream event: `deltaFor` is O(1) per event plus O(len fragment) appended to the call's buffer; End parses
each call's input once. Growth drivers: events E, tool-call input bytes B. Expected: O(E + B). Counted test:
TestDeltaForLinear (planned) feeds a call split into 8 and 16 fragments and asserts the buffer is appended, never
re-scanned (`json.Valid` invocations == 1 per call).

## Revisions

- 2026-10-21: ops:0 (tools owner) approved the public API on all five points, with two markups folded in
  above: (1) a Tool named `auto`/`any`/`none`, or a ToolChoice naming no declared tool, is refused with
  `ErrRequest` and tested; (2) the package doc states that v0.x may break at minor bumps, and the tag waits
  for landing on main.
- 2026-10-21: echo:1 consumer review: `Usage.Attempts` is defined as dispatched attempts and set on every
  return path (a hook rejection gives 0); ownership of delivered `Delta`/`Response` values is documented and
  tested; a hook rejection is identifiable with `errors.Is`. echo:1 also asked for cache controls. The echo TL
  scoped them out of this issue, so they are filed as a separate tools follow-up for a later v0.x.
- 2026-10-21: echo TL (ops-echo:0) approved, with one addition folded in: concurrent-use safety stated in the
  package doc, and a `ConcurrentStreams` Suite obligation (review:
  ops-echo `teams/echo/projects/echo-control-board/reviews/tools-85-plan.md`).
- 2026-10-21: plan-quality PQ-1 fixed. A `BeforeAttempt` rejection now aborts through a per-call `callState`
  and the cancel of a derived context, because the SDK retries a middleware that returns no response.
  Outcome order is fixed and tested. Minors fixed too: the implementer inventory is corrected, the
  `ErrCanceled` audit now covers every `ErrUnavailable` match site (adds `background.go:451`), test prose is
  cut to names plus strategy, and `FuzzDeltaAccumulator` is added.

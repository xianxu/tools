# LLM Harness (`internal/llm`) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** One transport, one contract, one stateful fake and one live conformance
suite for every model call `tools/` will ever make — so the five consumers that
follow (#10 authoring, #12 veto, #13 grading, #16 Q&A, #17 reflection) write a
prompt and a result type, and nothing else.

**Architecture:** `internal/llm` owns a two-method `Client` (structured
completion + streamed prose) implemented over the official `anthropic-sdk-go`,
plus a pure typed-task layer (`Task[T]` / `Run[T]`) that derives a JSON schema
from the caller's Go result type and decodes defensively into it. **The fake is an
`httptest` server speaking the Anthropic wire protocol, not a stubbed `Client`** —
so tests drive the real SDK code path (serialization, retries, SSE parsing), and
one shared obligation suite runs against both the fake and the live proxy.

**Tech Stack:** Go 1.26 · `github.com/anthropics/anthropic-sdk-go` v1.66.0 ·
`github.com/invopop/jsonschema` (already transitive) · `net/http/httptest` ·
local `cli-proxy-api` v7.1.71 on `127.0.0.1:8317`

---

## Measured facts this plan rests on

Recorded because the plan's shape depends on them, and because a later reader
should be able to re-run the measurement rather than trust the prose.

1. **`cli-proxy-api` is Anthropic-wire-shaped.** It routes `POST /v1/messages` and
   `POST /v1/messages/count_tokens` (`internal/api/server.go:442-443`). A probe
   from this machine reached it and was rejected on the key alone —
   `{"error":"Invalid API key"}`, 401, with the request body logged intact at
   `~/.cli-proxy-api/logs/error-v1-messages-2026-08-22T171220-*.log`. So the
   transport question is settled and only credentials are open.
2. **The operator's proxy currently rejects every key we have.** `~/.cli-proxy-api/config.yaml`
   (133 bytes) declares no `api-keys:`, yet the running v7.1.71 demands one.
   **This blocks the live conformance check only.** Everything else in this plan
   runs against the fake, so implementation is not blocked — see *Open question*
   at the end.
3. **The SDK is fetchable and costs 12 transitive dependencies.** `go get
   github.com/anthropics/anthropic-sdk-go@latest` → v1.66.0, adding `bahlo/generic-list-go`,
   `buger/jsonparser`, `invopop/jsonschema`, `pb33f/ordered-map/v2`,
   `standard-webhooks/libraries`, `tidwall/{gjson,match,pretty,sjson}`,
   `yaml/v4`, `golang.org/x/sync`. This repo has 3 direct deps today, so that is a
   real cost — accepted, because hand-rolling the wire format is how you end up
   maintaining an SSE parser (see *Alternatives rejected*).
4. **SDK bindings, read from the module source, not recalled.**
   `anthropic.NewClient(option.WithBaseURL, option.WithAPIKey, option.WithMaxRetries, option.WithRequestTimeout)`;
   `client.Messages.New(ctx, MessageNewParams) (*Message, error)`;
   `client.Messages.NewStreaming(ctx, MessageNewParams) *ssestream.Stream[MessageStreamEventUnion]`
   — **no `GetFinalMessage()`**, accumulate with `message.Accumulate(stream.Current())`;
   structured output is `MessageNewParams.OutputConfig = anthropic.OutputConfigParam{Effort: …, Format: anthropic.JSONOutputFormatParam{Schema: map[string]any{…}}}`
   (`message.go:5073, 6928, 13445`); API errors are `*anthropic.Error` with `.StatusCode`.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Request` | `internal/llm/llm.go` | new |
| `Response` | `internal/llm/llm.go` | new |
| `Usage` | `internal/llm/llm.go` | new |
| `Config` + `Resolve` | `internal/llm/config.go` | new |
| error taxonomy | `internal/llm/errors.go` | new |
| `Task[T]` + `Run[T]` | `internal/llm/task.go` | new |
| `SchemaFor[T]` | `internal/llm/schema.go` | new |
| `decode` | `internal/llm/task.go` | new |
| `Redact` | `internal/llm/config.go` | new |

- **`Request` / `Response` / `Usage`** — the wire-independent contract. `Request`
  carries `Task` (a stable name), `Model`, `System`, `Prompt`, `MaxTokens`,
  `Effort` and an optional `Schema`; `Response` carries `Text`, `Model`, `Stop`
  and `Usage`. Nothing in the contract mentions Anthropic, which is what lets the
  fake, the SDK client and any future backend share one obligation suite.
  - **Relationships:** 1:1 `Request`→`Response` per call; `Task[T]` renders to
    exactly one `Request` (N:1 — many task types, one request shape).
  - **DRY rationale:** First occurrence. Without it each consumer would build
    `anthropic.MessageNewParams` itself and five call sites would drift on
    model, effort, token cap and error handling.
  - **Future extensions:** Multi-turn (`Prompt` becomes a `[]Turn`) is the known
    next axis — #16 needs follow-up questions. Widen there, not here.

- **`Config` + `Resolve`** — where the model lives and who we are. `Resolve` is
  **pure over an env lookup function**, not over `os.Getenv`, so precedence is
  table-tested rather than exercised by mutating process state.
  - **Precedence:** `DEFINE_LLM_BASE_URL` → `http://127.0.0.1:8317` (the local
    proxy is the *default*, per the 2026-08-22 decision); key from
    `DEFINE_LLM_API_KEY` → `ANTHROPIC_API_KEY`; model from `DEFINE_LLM_MODEL` →
    `claude-opus-5`; effort from `DEFINE_LLM_EFFORT` → `high`.
  - **No startup probe.** Reachability is *not* checked when `define` starts —
    that would put a network round trip on the definition path, which must stay
    instant and offline. Availability is decided by config alone; an unreachable
    proxy surfaces as `ErrUnavailable` at the first actual call, which is the
    same degradation path as having no key.
  - **DRY rationale:** One answer to "which model, where, as whom" for every
    consumer and for the `--llm-check` diagnostic.

- **error taxonomy** — `ErrUnavailable`, `ErrRequest`, `ErrRefused`, `ErrMalformed`.
  The load-bearing split is `ErrUnavailable` (degrade quietly: no key, network
  down, 429, 5xx) versus `ErrRequest` (a 4xx that is **our bug** — a bad schema, a
  bad model name — and must be loud). This deliberately mirrors `fetch.go`'s
  `ErrNoAudio` vs `ErrFetchFailed` distinction, which exists for the same reason:
  "this is a normal outcome" and "something is broken" must not collapse into one
  error (ARCH-DRY — same idea, and the comment in `errors.go` says so).
  - **Future extensions:** `ErrBudget` if budgeting ever returns (it was
    explicitly dropped, 2026-08-22).

- **`Task[T]` + `Run[T]`** — a task is a prompt plus a result type; `Run` is the
  only way a consumer gets a typed answer. Go forbids type parameters on methods,
  so `Run` is a package function taking a `Client` — which is *better* here: the
  transport interface stays two methods wide and un-generic, so the fake and the
  live client implement the same tiny surface.
  - **Relationships:** `Task[T]` 1:1 with a golden file and 1:1 with a schema.
  - **DRY rationale:** The issue asked for "one method per task". Five per-task
    methods on one interface would put authoring, grading and reflection prompts
    *inside* the transport — domain logic in infrastructure. `Run[T]` keeps the
    per-task unit (the thing worth testing) with its consumer while there is
    still exactly one code path through the transport.
  - **Future extensions:** `RunStream[T]`, and a `Retry` policy field if a task
    ever wants different retry behaviour from the transport default.

- **`SchemaFor[T]`** — reflects a JSON Schema out of the caller's result type via
  `invopop/jsonschema`, memoised per type.
  - **DRY rationale:** The alternative is a hand-written `map[string]any` beside
    every result struct — two sources of truth for one shape, which drift the
    first time a field is added. The generated schema is snapshotted into a golden
    file, so it is still a visible artifact in a diff (which is the only property
    the hand-written version actually had).

- **`decode`** — parse `Response.Text` into `T`, defensively: strip a markdown
  fence if present, require the whole payload to be consumed, and return
  `ErrMalformed` wrapping the raw text (truncated) on any failure. **Never
  panics, never partially populates.**

- **`Redact`** — replaces a key with `sk-…abcd` shape. Every error string and log
  line goes through it. Tested by asserting the key literal is absent from the
  rendered error.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Client` (interface) | `internal/llm/llm.go` | new | — |
| `anthropicClient` | `internal/llm/anthropic.go` | new | Anthropic Messages API / `cli-proxy-api` |
| `llmtest.Fake` | `internal/llm/llmtest/fake.go` | new | the Anthropic wire protocol |
| `llmtest.Suite` | `internal/llm/llmtest/suite.go` | new | — (obligations both backends satisfy) |
| `llmtest.Golden` | `internal/llm/llmtest/golden.go` | new | `testdata/` files |
| `--llm-check` | `cmd/define/llmcheck.go` | new | the harness, from the binary |

- **`Client`** — two methods, deliberately:
  ```go
  Complete(ctx context.Context, r Request) (Response, error)
  Stream(ctx context.Context, r Request, onDelta func(string)) (Response, error)
  ```
  `Stream` still returns the final `Response`, so usage is reported on both paths
  and the caller never needs a second call to learn what it cost. Streaming is in
  scope now rather than in #16 because adding it later changes this interface, the
  fake and the suite at once — and #16 is the very next issue, so it is not
  speculative (YAGNI holds; ARCH-PURPOSE says don't under-deliver the committed
  purpose either).
  - **Injected into:** `Run[T]`, and every consumer. Consumers accept a `Client`,
    never construct one.

- **`anthropicClient`** — the real transport. Owns SDK construction, the
  `output_config` plumbing, the SSE accumulation loop, the `*anthropic.Error`
  → taxonomy mapping, and timing. Retries are the SDK's (`WithMaxRetries(2)`,
  which covers 408/409/429/5xx and connection errors) rather than a second
  hand-rolled loop.

- **`llmtest.Fake`** — **the design decision this plan most wants reviewed.** It is
  an `httptest.Server` implementing `POST /v1/messages`, not a `Client`
  implementation.
  - **Why beneath the SDK:** `lessons.md` (define #4) — *"a fake at the seam cannot
    see bugs below it… where you inject the double decides what the test can
    see."* A stubbed `Client` cannot catch a mis-serialized `output_config`, a
    dropped `anthropic-version` header, a retry that re-sends a consumed body, or
    an SSE frame the parser mishandles. Every one of those is a bug this harness
    can actually have. Placing the double on the wire makes all of them visible,
    and it is the same shape `fakeCDN` already uses in this repo (ARCH-DRY).
  - **State model, across calls:** requests recorded **in order** (method, path,
    headers, decoded body); canned replies keyed by task name with a
    `[]reply` queue per key so a *sequence* can be scripted; a scripted-failure
    list (`429` then success, to prove retry; a body that is not valid JSON, to
    prove `ErrMalformed`; a `stop_reason: "refusal"`, to prove `ErrRefused`); and
    a served-request counter so "the second call was served from cache" is
    assertable by consumers later.
  - **SSE is grounded, not invented.** A `-tags conformance` test records one real
    streamed response from the live proxy into
    `internal/llm/llmtest/testdata/stream-sample.sse`, and the fake replays that
    frame sequence with substituted text. Hand-writing the event sequence from
    memory is exactly the "modeled behaviour drifts from the real service" failure
    ARCH-MOCK exists to prevent.

- **`llmtest.Suite`** — the obligations, run against **both** backends: against the
  fake in the normal suite, and against the live proxy under `-tags conformance`.
  This is `storetest.Suite`'s shape applied to a service instead of a store
  (ARCH-DRY), and it is what makes "the fake behaves like the real thing" a test
  rather than a claim. Obligations are therefore **shape** obligations both can
  satisfy — a completion returns non-empty text and non-zero usage; a schema'd
  request returns payload that decodes into the target type; a stream delivers at
  least one delta and then a terminal response whose text equals the concatenated
  deltas; a cancelled context returns promptly with `ctx.Err()`; an unknown model
  returns `ErrRequest`, not a panic.

- **`llmtest.Golden`** — `AssertGolden(t, name, req)` renders a `Request`
  (model, effort, system, prompt, schema) to a stable text form and compares it
  with `testdata/golden/<name>.txt`, rewriting under `-update`. This is the
  mechanism the issue's *"prompt regressions are visible in a diff"* requires;
  the golden **files** for authoring/grading/etc. live with their consumers.

- **`--llm-check`** — a diagnostic that runs one trivial task through the real
  transport and prints resolved base URL, model, latency, usage and verdict. It
  exists so the harness is reachable from the binary in this issue rather than
  first exercised in #16, and because the operator needs a "did my proxy config
  take" answer (see measured fact 2). Prints the base URL and **never** the key.

---

## Alternatives rejected

- **Hand-rolled HTTP client (no SDK).** Saves 12 dependencies; costs an SSE
  parser, a retry policy, and a request/response schema we would have to keep in
  step with the API by hand. The dependency is the cheaper liability.
- **Stubbed `Client` as the test double.** Rejected above — it cannot see the bugs
  the harness can have.
- **Per-task methods on the transport interface** (`VetoDistractor`,
  `GradeSentence`, …). That is what the issue originally said, and it puts five
  domain prompts inside infrastructure; the sixth consumer then edits the
  transport. `Run[T]` keeps the per-task testable unit without that coupling.
- **Reusing ariadne's `cmd/sdlc/internal/judge`.** It shells out to a CLI agent
  (`claude`/`codex`/`gemini`) for fresh-context review — wrong shape (process
  spawn, minutes of latency, prose verdicts) and, being under `internal/` of
  another module, not importable.

---

## Risks

| Risk | Mitigation |
|---|---|
| `cli-proxy-api` may strip `output_config` (structured outputs) on the way upstream | The suite asserts *decodability*, not that the server enforced the schema; `decode` degrades to `ErrMalformed`, and consumers already must handle that. A conformance test records whether the live proxy honours it, so we learn the answer instead of assuming one. |
| SSE frame shape differs between the proxy and `api.anthropic.com` | The fake replays a **recorded** sample; the conformance test re-records and fails on drift. |
| No working proxy credential today | Blocks conformance only; the whole implementation and its suite run against the fake. Flagged to the operator rather than worked around. |
| `internal/` first-consumer rule | Operator override recorded in the issue's `## Revisions` and the project's decisions; `AGENTS.local.md` gets the carve-out in M1's last task, in the same change that creates the package. |

---

## Robustness bar — what "library level robust" means here

Operator, 2026-08-22: *"the golang implementation should be library level robust,
as it will form the base of many projects"*, and pointed at two working
implementations to learn from. Both were read; what follows is what transferred.

### Prior art

**`kbench/competition/arc-agi-3/metis/model.py`** (Python, stdlib-only, 467 lines).
Its module docstring is a field report of four successive fixes to the same class
of bug, each with the measurement that exposed it. The transferable findings:

1. **A socket timeout is not a deadline.** It is per-operation and resets on every
   byte, so a trickling peer is bounded by nothing. Measured: one run spent 4.41
   of 4.72 hours inside six turns against a median turn of 11 seconds.
2. **A deadline must cover every phase**, including the header read — which in
   Python happens inside `getresponse()`, where no user-level check can see it.
   Measured at 25.2s against a 2s deadline.
3. **Closing the socket under it is not enough on macOS/BSD**, where a thread
   parked in `recv` may simply stay there. Three calls in one segment ran 1004s,
   490s and 928s against a 180s deadline with no retry and no error recorded.
4. **Retries spend the same budget**, and backoff never sleeps past it.
5. **An outage is not a stall.** `is_outage()` deliberately excludes 408 and
   deadline misses: a stream that disconnects mid-reply and a connection being
   refused need opposite responses, and conflating them either parks a healthy run
   or abandons a recoverable one.
6. **Record the phase.** The worker publishes `connect|send|headers|body|done`, so
   a slow call can answer "slow *where*". Two days were spent unable to.

**`parley.nvim`** (Lua) — `lua/parley/sse.lua` and `lua/parley/cliproxy.lua`:

7. **A malformed chunk must never abort a stream.** `safe_json_decode` returns nil
   rather than raising, because SSE streams routinely carry partial or non-JSON
   lines.
8. **Optional fields need a reader, not a truthiness check.** `vim.NIL` is truthy
   in Lua, so `if obj.name then …` accepts an explicit JSON null and overwrites a
   good value. Go's zero values make this specific trap unreachable, but the
   general rule — *a continuation chunk may repeat as null a field an earlier chunk
   set properly* — is a real streaming behaviour our accumulator must not regress on.
9. **Health, repair and their budget belong to parley, not to us.** It owns a
   health probe, a recovery ladder, one-shot repair guards so a repair cannot loop,
   and a wall-clock repair budget every rung must sum under.

### What this library does, and deliberately does not, do

**Does not: heal the proxy.** Parley already owns that ladder, with a budget and
loop guards. A second healing mechanism in `define` would be a parallel track
where one exists — the operator's standing preference is one mechanism with
several triggers. Our job is to *distinguish* "the proxy is down" from "the model
refused" and say which, so parley's machinery (or a human) acts on the right one.

**Does: bound a call, and prove the bound.** Go is better positioned than Python
here and the difference must be stated rather than assumed:

- `net/http` honours context cancellation through the header read and the body
  read, so findings 1–3 above do **not** require a worker-abandonment hack in Go.
- The SDK propagates the caller's ctx into retries: read at
  `internal/requestconfig/requestconfig.go:450-478`, it checks `ctx.Err()` before
  each attempt and selects on `ctx.Done()` during backoff. So one
  `context.WithTimeout` bounds attempts **plus** backoff **plus** body read —
  finding 4, for free.
- Its retry set (`requestconfig.go:270-274`) is 408, 409, 429 and **every** 5xx,
  which covers kbench's measured set including 529, and it honours `retry-after`
  and `x-should-retry`.

**Because Go gives this for free, it must be TESTED rather than trusted** — that is
the whole lesson of the four Python versions. Task 5 pins it with kbench's exact
failure mode: a server that dribbles headers one byte at a time, and a short ctx
that must still return promptly.

**Does: detect a stalled stream, which no deadline can.** A total deadline is the
wrong instrument for streaming — you want minutes of headroom for a long answer
and seconds of patience for a dead one. So `Stream` carries an **idle timeout**:
no delta for `StallAfter` and the call aborts as `ErrUnavailable`. This is the
Go-shaped answer to findings 1–3, and it is the one thing here the SDK does not
provide.

**Does: say where the time went.** `Progress{Phase, Elapsed, Bytes}` via an
optional `OnSlow` hook, phases `connect|waiting|streaming|done`, fired on a ticker.
Errors name the phase. Finding 6 cost two days of not being able to ask.

**Does: preserve what it does not understand.** `Response.Blocks` keeps every
content block as received, including `thinking` blocks and their signatures. #16's
follow-up questions must echo them back byte-for-byte on the same model or the
continuation is rejected; a library that flattens the response to a string makes
that impossible for every future consumer. Finding: kbench's "reasoning is kept,
not interpreted" — nothing here decides what reasoning *is*.

---

## Chunk 1: M1 — transport, contract, and the wire fake

### Task 1: Package skeleton and the contract

**Files:**
- Create: `internal/llm/llm.go`
- Create: `internal/llm/doc.go`
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

```bash
cd /Users/xianxu/workspace/tools
GOFLAGS=-mod=mod go get github.com/anthropics/anthropic-sdk-go@v1.66.0
GOFLAGS=-mod=mod go get github.com/invopop/jsonschema@latest
```

Expected: both appear in `go.mod` as `// indirect` — nothing imports them yet.
**Do not run `go mod tidy` here**: with no importer, tidy removes both again. Tidy
runs at the end of Task 5 (SDK) and Task 9 (jsonschema), which is where the
`// indirect` markers actually drop.

- [ ] **Step 2: Write the contract**

`internal/llm/llm.go`:

```go
// Package llm is the one way anything in tools/ talks to a language model.
//
// It owns the transport, the error taxonomy, and the typed-task layer. It owns
// no prompts: a prompt is domain knowledge and lives with the consumer that
// needs it, as a Task[T] whose result type IS its schema.
package llm

import (
	"context"
	"time"
)

// Client is the transport. Two methods, because there are two shapes of answer:
// a structured one a program consumes, and prose a person reads as it arrives.
//
// Nothing here mentions Anthropic. That is what lets the wire fake, the real
// client and the live conformance run share one obligation suite (llmtest.Suite).
type Client interface {
	// Complete returns the whole answer at once.
	Complete(ctx context.Context, r Request) (Response, error)
	// Stream delivers text as it arrives and still returns the final Response, so
	// a caller never needs a second call to learn what the answer cost.
	// onDelta may be nil, which makes Stream a slower Complete — useful only for
	// testing that both paths agree.
	Stream(ctx context.Context, r Request, onDelta func(string)) (Response, error)
}

// Request is one call, described independently of any provider's wire format.
type Request struct {
	// Task is a stable identifier — "author-cloze", "veto-distractor". It names
	// the golden file, keys the fake's canned replies, and labels usage. It is
	// NOT sent to the model.
	Task string
	// Model, Effort and MaxTokens are per-request overrides; zero means "use the
	// resolved Config default", so a consumer that does not care says nothing.
	Model     string
	Effort    string
	MaxTokens int64

	System string
	Prompt string

	// Schema, when non-nil, asks the provider to constrain output to it. Callers
	// must still handle ErrMalformed: an intermediary may drop the field, and a
	// constraint we did not verify is not a guarantee.
	Schema map[string]any
}

// Response is one answer.
type Response struct {
	Text  string
	Model string
	// Blocks is every content block as received, unflattened — including
	// thinking blocks and their signatures.
	//
	// Kept, not interpreted (the rule kbench's model.py arrived at the hard way).
	// Two reasons it cannot be derived later from Text: a multi-turn follow-up on
	// the same model must echo thinking blocks back BYTE FOR BYTE or the
	// continuation is rejected (#16 needs exactly this), and a library that
	// flattens to a string forecloses that for every future consumer.
	Blocks []Block
	// Stop is the provider's stop reason, passed through rather than
	// interpreted — except "refusal", which Complete turns into ErrRefused.
	Stop  string
	Usage Usage
}

// Usage is what the call cost. Reported, not budgeted against: the operator
// decided on 2026-08-22 that quality wins over cost for this tool, so this
// exists to be shown (--llm-check, --stats later), not to gate anything.
type Usage struct {
	InputTokens  int64
	OutputTokens int64
	Duration     time.Duration
	// ThinkingTokens is billed but invisible: opus-5 returns a thinking block
	// whose text is empty by default. Without this the output token count looks
	// inexplicably high, which is how it becomes folklore.
	ThinkingTokens int64
	// PreambleTokens is what the PROXY prepended, not what we sent — measured at
	// ~1,900 cached tokens on the parley instance. Surfaced so nobody tunes a
	// prompt believing our System field is the only system prompt.
	PreambleTokens int64
}

// Block is one content block, preserved as received.
type Block struct {
	Type      string // "text" | "thinking" | anything a future model adds
	Text      string
	Signature string // thinking blocks carry one; it must round-trip unchanged
}

// Progress reports where a slow call currently is. Fired on a ticker via
// Config.OnSlow, never on the happy path.
//
// The point is answering "slow WHERE" — kbench spent two days unable to, because
// a call that is stuck connecting and a call that is stuck mid-body look
// identical from outside.
type Progress struct {
	Task    string
	Phase   string // "connect" | "waiting" | "streaming" | "done"
	Elapsed time.Duration
	Bytes   int
}
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./internal/... && go vet ./internal/...`
Expected: no output, exit 0.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum internal/llm/
git commit -m "#11 M1: llm contract — Request/Response/Client, no provider in the types"
```

---

### Task 2: Error taxonomy

**Files:**
- Create: `internal/llm/errors.go`
- Test: `internal/llm/errors_test.go`

- [ ] **Step 1: Write the failing test**

`internal/llm/errors_test.go`:

```go
package llm

import (
	"errors"
	"net/http"
	"testing"
)

// The split that matters: a caller degrades on ErrUnavailable and must NOT
// degrade on ErrRequest, because ErrRequest means we sent something wrong and
// silently falling back would hide our own bug forever.
func TestClassifyStatus(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusTooManyRequests, ErrUnavailable},
		{http.StatusInternalServerError, ErrUnavailable},
		{http.StatusBadGateway, ErrUnavailable},
		{http.StatusServiceUnavailable, ErrUnavailable},
		{http.StatusUnauthorized, ErrUnavailable}, // bad/absent credential: degrade, don't crash a review
		{http.StatusBadRequest, ErrRequest},
		{http.StatusNotFound, ErrRequest},
		{http.StatusUnprocessableEntity, ErrRequest},
	}
	for _, c := range cases {
		got := classifyStatus(c.status, errors.New("boom"))
		if !errors.Is(got, c.want) {
			t.Errorf("classifyStatus(%d) = %v, want %v", c.status, got, c.want)
		}
	}
}

// The cause must survive classification, or a transport failure becomes
// undiagnosable from the error alone.
func TestClassifyKeepsCause(t *testing.T) {
	cause := errors.New("dial tcp 127.0.0.1:8317: connection refused")
	got := classifyStatus(0, cause)
	if !errors.Is(got, ErrUnavailable) || !errors.Is(got, cause) {
		t.Errorf("got %v; want both ErrUnavailable and the cause", got)
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/llm/ -run Classify -count=1`
Expected: FAIL — `undefined: ErrUnavailable`, `undefined: classifyStatus`.

- [ ] **Step 3: Implement**

`internal/llm/errors.go`:

```go
package llm

import (
	"errors"
	"fmt"
	"net/http"
)

// The taxonomy. The load-bearing distinction is the first two: one is a normal
// outcome a caller absorbs, the other is a defect a caller must not hide.
//
// This is deliberately the same split fetch.go draws between ErrNoAudio ("this
// word has no recording" — normal) and ErrFetchFailed ("the network is down" —
// not). Collapsing the two is what makes a broken feature look like a quiet one.
var (
	// ErrUnavailable means the model could not be reached or is not configured.
	// Every consumer degrades on this: no key, no network, rate-limited, 5xx.
	// The learner's review is never blocked on a third party.
	ErrUnavailable = errors.New("llm: unavailable")
	// ErrRequest means WE sent something the provider rejected — a bad schema, an
	// unknown model, a malformed body. Loud on purpose: degrading here would hide
	// our own bug behind the same silence as a flight-mode fallback.
	ErrRequest = errors.New("llm: bad request")
	// ErrRefused means the model declined. Distinct from ErrRequest (the request
	// was well-formed) and from ErrUnavailable (the service is fine).
	ErrRefused = errors.New("llm: refused")
	// ErrMalformed means the answer did not decode into the caller's type. The
	// caller skips this question; it does not crash the session.
	ErrMalformed = errors.New("llm: malformed response")
)

// classifyStatus maps an HTTP status (0 when the request never got one) onto the
// taxonomy, preserving the cause so errors.Is reaches it.
func classifyStatus(status int, cause error) error {
	switch {
	case status == 0: // never reached the server: DNS, refused, timeout
		return fmt.Errorf("%w: %w", ErrUnavailable, cause)
	case status == http.StatusTooManyRequests, status >= 500:
		return fmt.Errorf("%w: %w", ErrUnavailable, cause)
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		// A missing or wrong credential is an operator configuration state, not a
		// crash: `define` still defines words. --llm-check is where it is loud.
		return fmt.Errorf("%w: %w", ErrUnavailable, cause)
	default:
		return fmt.Errorf("%w: %w", ErrRequest, cause)
	}
}
```

- [ ] **Step 4: Run and verify green**

Run: `go test ./internal/llm/ -run Classify -count=1 -v`
Expected: PASS, both tests.

- [ ] **Step 5: Mutation-check the split**

Change `case status == http.StatusUnauthorized, status == http.StatusForbidden:`
to fall through to `default`. Run `go test ./internal/llm/ -count=1`.
Expected: FAIL on the 401 row. **Restore by copying the file back from a
scratch copy — not `git checkout`**, which discards every other uncommitted edit
in the file (`lessons.md`, define #2 round 4).

- [ ] **Step 6: Commit**

```bash
git add internal/llm/errors.go internal/llm/errors_test.go
git commit -m "#11 M1: error taxonomy — degrade on unavailable, stay loud on our own bad request"
```

---

### Task 3: Config resolution, pure over an env lookup

**Files:**
- Create: `internal/llm/config.go`
- Test: `internal/llm/config_test.go`

- [ ] **Step 1: Write the failing test**

```go
package llm

import (
	"errors"
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestResolveDefaultsToTheLocalProxy(t *testing.T) {
	c, err := Resolve(env(map[string]string{"ANTHROPIC_API_KEY": "sk-test-key"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL != defaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, defaultBaseURL)
	}
	if c.Model != defaultModel {
		t.Errorf("Model = %q, want %q", c.Model, defaultModel)
	}
}

func TestResolvePrecedence(t *testing.T) {
	c, err := Resolve(env(map[string]string{
		"ANTHROPIC_API_KEY":   "sk-generic",
		"DEFINE_LLM_API_KEY":  "sk-specific",
		"DEFINE_LLM_BASE_URL": "http://elsewhere:9000",
		"DEFINE_LLM_MODEL":    "claude-sonnet-5",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if c.APIKey != "sk-specific" {
		t.Errorf("APIKey = %q, want the DEFINE_-scoped one to win", c.APIKey)
	}
	if c.BaseURL != "http://elsewhere:9000" || c.Model != "claude-sonnet-5" {
		t.Errorf("got %+v", c)
	}
}

// No key is not an error to shout about — it is the ordinary offline state, and
// it must be recognisable with errors.Is so callers degrade uniformly.
func TestResolveWithNoKeyIsUnavailable(t *testing.T) {
	_, err := Resolve(env(nil))
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

func TestRedactNeverLeaksTheKey(t *testing.T) {
	const key = "sk-ant-api03-SUPERSECRETVALUE"
	got := Redact(key)
	if strings.Contains(got, "SUPERSECRET") {
		t.Fatalf("Redact leaked the key: %q", got)
	}
	if got == "" {
		t.Fatal("Redact returned nothing; an empty diagnostic is useless")
	}
}

func TestRedactShortAndEmpty(t *testing.T) {
	if Redact("") != "(unset)" {
		t.Errorf("Redact(\"\") = %q", Redact(""))
	}
	if strings.Contains(Redact("abc"), "abc") {
		t.Error("a short key must not be echoed whole")
	}
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `go test ./internal/llm/ -run 'Resolve|Redact' -count=1`
Expected: FAIL — undefined `Resolve`, `Redact`, `defaultBaseURL`, `defaultModel`.

- [ ] **Step 3: Implement**

`internal/llm/config.go`:

```go
package llm

import (
	"fmt"
	"time"
)

const (
	// The local cli-proxy-api is the DEFAULT, not a fallback (operator,
	// 2026-08-22): it fronts a subscription plan, so the good model is the cheap
	// path. api.anthropic.com is reached by setting DEFINE_LLM_BASE_URL.
	defaultBaseURL = "http://127.0.0.1:8317"
	defaultModel   = "claude-opus-5"
	defaultEffort  = "high"
	// Generous on purpose. Authoring a practice item is a thinking task, and it
	// runs in a batch nobody is waiting on (#10).
	defaultMaxTokens = 8192
	defaultTimeout   = 5 * time.Minute
)

// Config is where the model lives and who we are.
type Config struct {
	BaseURL   string
	APIKey    string
	Model     string
	Effort    string
	MaxTokens int64
	// Timeout is the TOTAL budget for a call: attempts, backoff and body read.
	// Enforced as a context deadline, which in Go covers every phase — including
	// the header read that in Python needed a worker to bound (see the Robustness
	// bar). Pinned by TestSlowHeadersStillHitTheDeadline.
	Timeout time.Duration
	// StallAfter bounds SILENCE inside a stream, which Timeout cannot: a long
	// answer legitimately takes minutes, while a dead connection should fail in
	// seconds. Zero disables it.
	StallAfter time.Duration
	// OnSlow, when set, is called on a ticker while a call is still running, with
	// the phase it is in. Optional, off by default, and never called on a fast
	// path — this is for answering "slow where", not for logging.
	OnSlow func(Progress)
}

// Resolve reads configuration from an env lookup function.
//
// It takes the lookup rather than calling os.Getenv so precedence is a table
// test instead of a process-state mutation, and so a test can assert what
// happens with NOTHING set without unsetting the developer's own environment
// (ARCH-PURE: this is the pure half; main() supplies os.Getenv).
//
// It deliberately does NOT probe reachability. A probe here would put a network
// round trip on `define <word>`, whose whole promise is that it is instant and
// offline. An unreachable proxy shows up as ErrUnavailable at the first real
// call, which is the same path as having no key at all — one degradation story,
// not two.
func Resolve(getenv func(string) string) (Config, error) {
	first := func(keys ...string) string {
		for _, k := range keys {
			if v := getenv(k); v != "" {
				return v
			}
		}
		return ""
	}
	c := Config{
		BaseURL:   orDefault(first("DEFINE_LLM_BASE_URL"), defaultBaseURL),
		APIKey:    first("DEFINE_LLM_API_KEY", "ANTHROPIC_API_KEY"),
		Model:     orDefault(first("DEFINE_LLM_MODEL"), defaultModel),
		Effort:    orDefault(first("DEFINE_LLM_EFFORT"), defaultEffort),
		MaxTokens: defaultMaxTokens,
		Timeout:   defaultTimeout,
	}
	if c.APIKey == "" {
		return c, fmt.Errorf("%w: no API key (set DEFINE_LLM_API_KEY or ANTHROPIC_API_KEY)", ErrUnavailable)
	}
	return c, nil
}

func orDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// Redact renders a credential safe to print. Every diagnostic path goes through
// it; --llm-check prints the base URL and this, never the key.
func Redact(key string) string {
	if key == "" {
		return "(unset)"
	}
	if len(key) <= 8 {
		return "(set, short)"
	}
	return key[:6] + "…" + key[len(key)-4:]
}
```

- [ ] **Step 4: Run and verify green**

Run: `go test ./internal/llm/ -run 'Resolve|Redact' -count=1 -v`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/llm/config.go internal/llm/config_test.go
git commit -m "#11 M1: config resolution, pure over an env lookup; no startup probe"
```

---

### Task 4: The wire fake

**Files:**
- Create: `internal/llm/llmtest/fake.go`
- Test: exercised by Task 6's suite; this task ends with a self-test that the
  fake serves a well-formed non-streaming body.

- [ ] **Step 1: Write the fake**

`internal/llm/llmtest/fake.go`:

```go
// Package llmtest holds the stateful fake every llm.Client test runs against,
// and the obligation suite the fake and the live service must BOTH satisfy.
//
// The fake is an httptest server speaking the Anthropic wire protocol — not a
// stub of llm.Client. That placement is the point: a stubbed Client cannot see a
// mis-serialized output_config, a dropped anthropic-version header, a retry that
// re-sends a consumed body, or an SSE frame the parser mishandles, because all
// of those live BELOW the seam it replaces (lessons.md, define #4: "where you
// inject the double decides what the test can see"). Putting the double on the
// wire means production flow and test flow share the same boundary, which is the
// only arrangement that satisfies ARCH-MOCK.
package llmtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// Recorded is one request as the server saw it.
type Recorded struct {
	Path    string
	Headers http.Header
	Body    map[string]any
}

// System returns the request's system prompt as a single string, or "".
func (r Recorded) System() string { return joinBlocks(r.Body["system"]) }

// Prompt returns the last user message's text.
func (r Recorded) Prompt() string {
	msgs, _ := r.Body["messages"].([]any)
	for i := len(msgs) - 1; i >= 0; i-- {
		m, _ := msgs[i].(map[string]any)
		if m["role"] == "user" {
			return joinBlocks(m["content"])
		}
	}
	return ""
}

// Schema returns the request's output_config.format.schema, or nil.
func (r Recorded) Schema() map[string]any {
	oc, _ := r.Body["output_config"].(map[string]any)
	f, _ := oc["format"].(map[string]any)
	s, _ := f["schema"].(map[string]any)
	return s
}

func joinBlocks(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		var b strings.Builder
		for _, e := range t {
			m, _ := e.(map[string]any)
			if s, ok := m["text"].(string); ok {
				b.WriteString(s)
			}
		}
		return b.String()
	}
	return ""
}

// Reply is one scripted answer. Exactly one of Text / Status / Raw is used, in
// that precedence — Status models a transport failure, Raw models a body that is
// not what we expect (to prove defensive decoding).
type Reply struct {
	Text   string
	Stop   string // "" means "end_turn"
	Status int    // non-zero: reply with this HTTP status instead
	Raw    string // non-empty: send this as the text content verbatim
	Model  string
}

// Fake is a stateful Anthropic-shaped server.
//
// State that persists across calls, and why each is needed:
//   - requests, IN ORDER — the ordering is what proves a retry happened, and
//     what lets a consumer assert its prompt (via Recorded) rather than assert
//     that some function was called.
//   - a QUEUE of replies per task — so a sequence can be scripted: 429 then
//     success proves the SDK's retry reaches us; success then 429 proves a later
//     failure degrades. A single canned reply per key cannot express either.
//   - served count — so "the second call was served from cache" stays assertable
//     by consumers that add caching later (#9/#10 will).
type Fake struct {
	*httptest.Server

	mu       sync.Mutex
	requests []Recorded
	// matchers is ORDERED — see next(). A map here made overlapping keys flaky.
	matchers []matcher
	fallback Reply
	// streamSSE is the recorded frame template; see stream-sample.sse.
	streamSSE []string
}

// NewFake starts a fake and registers cleanup.
func NewFake(t *testing.T) *Fake {
	t.Helper()
	f := &Fake{
		replies:  map[string][]Reply{},
		fallback: Reply{Text: "ok"},
	}
	f.Server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.Close)
	return f
}

// Script queues replies for requests whose prompt contains match. Matching on
// the prompt rather than on a header keeps the fake honest: it knows only what a
// real server would know.
func (f *Fake) Script(match string, replies ...Reply) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.replies[match] = append(f.replies[match], replies...)
}

// Requests returns everything received, in order.
func (f *Fake) Requests() []Recorded {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Recorded(nil), f.requests...)
}

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	dec := json.NewDecoder(r.Body)
	_ = dec.Decode(&body)

	rec := Recorded{Path: r.URL.Path, Headers: r.Header.Clone(), Body: body}
	f.mu.Lock()
	f.requests = append(f.requests, rec)
	reply := f.next(rec.Prompt())
	sse := append([]string(nil), f.streamSSE...)
	f.mu.Unlock()

	if reply.Status != 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(reply.Status)
		fmt.Fprintf(w, `{"type":"error","error":{"type":"api_error","message":"scripted %d"}}`, reply.Status)
		return
	}
	if streaming, _ := body["stream"].(bool); streaming {
		f.serveStream(w, reply, sse)
		return
	}
	f.serveJSON(w, reply)
}

// next pops the queue whose key matches the prompt, else the fallback.
// Callers hold f.mu.
//
// The matcher list is ORDERED and first-match-wins. An earlier draft ranged over
// a map, so two scripted keys both matching one prompt served a random reply and
// the test flaked once in a while — the worst kind of test failure, because it
// looks like the code.
func (f *Fake) next(prompt string) Reply {
	for i := range f.matchers {
		m := &f.matchers[i]
		if len(m.queue) > 0 && strings.Contains(prompt, m.match) {
			r := m.queue[0]
			m.queue = m.queue[1:]
			return r
		}
	}
	return f.fallback
}
```

- [ ] **Step 2: Build `serveJSON` from the RECORDED envelope, not from memory**

**The envelope is multi-block, and `content[0]` is not the text.** Measured
2026-08-22 against the live proxy and committed at
`internal/llm/llmtest/testdata/message-thinking.json`: `claude-opus-5` answering a
non-trivial prompt returns

```
content: [ {type: "thinking", thinking: ""}, {type: "text", text: "…"} ]
usage.output_tokens_details.thinking_tokens: 212
```

The thinking block's text is empty (`display` defaults to `"omitted"` on opus-5)
but **the block is present and it is first**. Thinking is on by default on this
model and cannot be turned off at `xhigh`/`max` effort, so this is the normal
shape, not an edge case.

This nearly went in wrong: the first probe of this proxy used a trivial prompt,
got `["text"]` alone, and would have produced a single-block fake that every test
passed against while the live service returned `""`. A fake modelled on the easy
case is how a suite goes green against a broken client.

So `serveJSON` emits **thinking-then-text by default**, with a `Reply.NoThinking`
escape for the trivial-prompt shape (also real — the second committed artifact,
`message-schema.json`, shows it). `serveStream` replays
`testdata/stream-sample.sse`, which carries the `ping` event and the
space-padded `data:` payloads.

- [ ] **Step 3: Add `Cassette` — replies are RECORDED, never invented**

```go
// Cassette is a real response, frozen. Keyed by renderRequest's hash so a prompt
// edit misses LOUDLY rather than passing against a stale recording.
//
// The alternative — Reply{Text: "flattering, servile"} — asserts against the
// plan author's guess at what a model says. You cannot fake judgment; you can
// freeze a real answer and pin OUR HANDLING of it. What a cassette does NOT
// establish: that the model reliably produces that answer. It is one sample of a
// stochastic process, and the live conformance run is what detects drift.
func Cassette(t *testing.T, r llm.Request) Reply
```

- Stored at `testdata/cassettes/<hash>.json`; `-record` re-runs against the live
  proxy and rewrites, so a diff shows how the answer moved.
- A miss without `-record` fails naming the task and hash. Never a fallback.
- `Script` survives for transport shapes — a `429`, a refusal, a truncated body.
  Inventing *those* is correct; they are protocol, not judgment.
- Hashing uses the **one** `renderRequest` that Task 11's golden printer uses. Two
  renderers would drift and a golden update would leave a stale cassette still
  matching (ARCH-DRY).

- [ ] **Step 4: Self-test that the envelope is well-formed**

`internal/llm/llmtest/fake_test.go` — POST a minimal body with `net/http`, assert
200, that `content` is `[thinking, text]` in that order, that the text is the
scripted one, and that the recorded request carries the prompt. This is the
*floor*: if the fake's own envelope is wrong, every later failure is misattributed
to the client.

Run: `go test ./internal/llm/llmtest/ -count=1`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/llm/llmtest/
git commit -m "#11 M1: stateful wire fake — httptest server, not a stubbed Client"
```

---

### Task 5: The real client over the SDK

**Files:**
- Create: `internal/llm/anthropic.go`
- Test: `internal/llm/anthropic_test.go`

- [ ] **Step 1: Write the failing tests** — drive the **real** client at the fake:

```go
func TestCompleteRoundTrip(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("define sycophantic", llmtest.Reply{Text: "flattering, servile"})
	c := llm.New(llm.Config{BaseURL: f.URL, APIKey: "sk-test-1234567890", Model: "claude-opus-5", MaxTokens: 256, Timeout: time.Minute})

	got, err := c.Complete(t.Context(), llm.Request{Task: "t", Prompt: "define sycophantic"})
	if err != nil { t.Fatal(err) }
	if got.Text != "flattering, servile" { t.Errorf("Text = %q", got.Text) }
	if got.Usage.Duration == 0 { t.Error("Duration not recorded") }

	reqs := f.Requests()
	if len(reqs) != 1 { t.Fatalf("%d requests, want 1", len(reqs)) }
	if reqs[0].Path != "/v1/messages" { t.Errorf("path = %q", reqs[0].Path) }
	if reqs[0].Headers.Get("anthropic-version") == "" {
		t.Error("anthropic-version header absent — the SDK is expected to send it")
	}
}

// The retry belongs to the SDK; this asserts we actually get it, and that a
// retried request arrives INTACT (a consumed body would arrive empty).
func TestRetriesOn429(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("hello", llmtest.Reply{Status: 429}, llmtest.Reply{Text: "second try"})
	c := llm.New(...)
	got, err := c.Complete(t.Context(), llm.Request{Task: "t", Prompt: "hello"})
	if err != nil { t.Fatal(err) }
	if got.Text != "second try" { t.Errorf("Text = %q", got.Text) }
	reqs := f.Requests()
	if len(reqs) != 2 { t.Fatalf("%d requests, want 2 (a retry)", len(reqs)) }
	if reqs[1].Prompt() != "hello" { t.Errorf("retried body = %q, want the original prompt", reqs[1].Prompt()) }
}

// The regression this exists for: Content[0] is a THINKING block on any
// non-trivial prompt, so a naive implementation returns "" live while passing
// against a single-block fake.
func TestTextSkipsThinkingBlock(t *testing.T) {
	f := llmtest.NewFake(t)
	f.ServeRecorded(t, "message-thinking.json") // the committed live capture
	c := llm.New(...)
	got, err := c.Complete(t.Context(), llm.Request{Task: "t", Prompt: "which also fits"})
	if err != nil { t.Fatal(err) }
	if got.Text == "" { t.Fatal("Text is empty — reading content[0] instead of the text blocks") }
	if strings.Contains(got.Text, "thinking") { t.Errorf("thinking leaked into Text: %q", got.Text) }
}

func TestRefusalBecomesErrRefused(t *testing.T) { /* Reply{Text:"", Stop:"refusal"} */ }
func TestBadRequestIsLoud(t *testing.T)         { /* Reply{Status:400} → errors.Is(err, llm.ErrRequest) */ }
func TestServerDownIsUnavailable(t *testing.T)  { /* point at a closed port → ErrUnavailable */ }
func TestStreamDeltasConcatToText(t *testing.T) { /* deltas collected == Response.Text */ }
func TestStreamCancellation(t *testing.T)       { /* cancel mid-stream → ctx.Err(), returns promptly */ }

// kbench's EXACT failure mode, ported: a peer that dribbles headers one byte at a
// time is bounded by no socket timeout, and in Python it took three attempts to
// bound it (measured there at 25.2s against a 2s deadline, then 1004s against
// 180s). Go's net/http honours ctx through the header read, so this should pass
// as written — which is precisely why it must be ASSERTED rather than assumed.
// If Go ever stops honouring it, this test is the alarm.
func TestSlowHeadersStillHitTheDeadline(t *testing.T) {
	srv := llmtest.DribblingServer(t, time.Hour) // one header byte per 50ms, forever
	c := llm.New(llm.Config{BaseURL: srv.URL, APIKey: "sk-test-1234567890"})
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	start := time.Now()
	_, err := c.Complete(ctx, llm.Request{Task: "t", Prompt: "hello"})
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("took %s against a 2s deadline — the header read is unbounded", elapsed)
	}
	if !errors.Is(err, llm.ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
}

// A deadline is the wrong instrument for a stream: minutes of headroom for a long
// answer, seconds of patience for a dead one. StallAfter is the difference, and it
// is the one bound the SDK does not provide.
func TestStreamStallIsDetected(t *testing.T) {
	f := llmtest.NewFake(t)
	f.StallAfterFirstDelta()  // one delta, then silence, no FIN
	c := llm.New(llm.Config{BaseURL: f.URL, APIKey: "sk-test-1234567890", StallAfter: 500 * time.Millisecond})

	start := time.Now()
	_, err := c.Stream(t.Context(), llm.Request{Task: "t", Prompt: "hello"}, nil)
	if !errors.Is(err, llm.ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("stall took %s to detect with StallAfter=500ms", elapsed)
	}
}

// Retries spend the SAME budget as the call (kbench finding 4). Verified read of
// requestconfig.go:450-478 says the SDK selects on ctx.Done() during backoff;
// this asserts the behaviour rather than the reading.
func TestRetryBackoffRespectsTheDeadline(t *testing.T) {
	f := llmtest.NewFake(t)
	f.Script("hello", llmtest.Reply{Status: 429}, llmtest.Reply{Status: 429}, llmtest.Reply{Status: 429})
	c := llm.New(...)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	start := time.Now()
	_, err := c.Complete(ctx, llm.Request{Task: "t", Prompt: "hello"})
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("backoff ran %s past a 1s deadline", elapsed)
	}
	if !errors.Is(err, llm.ErrUnavailable) { t.Errorf("err = %v", err) }
}

// Finding 7/8: a malformed or unexpected frame must not take down the stream, and
// a later chunk repeating a field as null must not erase what an earlier one set.
func TestStreamSurvivesAMalformedFrame(t *testing.T) {
	f := llmtest.NewFake(t)
	f.StreamFrames("one", "<<not json>>", " two")
	// deltas "one" and " two" arrive; the junk frame is skipped, not fatal.
}

// Phases exist so a slow call can say WHERE. Asserted by driving a server that
// hangs in a chosen phase and reading what OnSlow reported.
func TestOnSlowNamesThePhase(t *testing.T) { /* waiting vs streaming */ }
```

- [ ] **Step 2: Run and watch them fail** — `go test ./internal/llm/ -count=1`

- [ ] **Step 3: Implement `anthropic.go`**

Key points, each of which a test above pins:

```go
func New(c Config) Client {
	opts := []option.RequestOption{
		option.WithBaseURL(c.BaseURL),
		option.WithAPIKey(c.APIKey),
		option.WithMaxRetries(2),              // SDK default; stated so it is a decision
		option.WithRequestTimeout(c.Timeout),
	}
	return &anthropicClient{cfg: c, api: anthropic.NewClient(opts...)}
}

func (a *anthropicClient) params(r Request) anthropic.MessageNewParams {
	p := anthropic.MessageNewParams{
		Model:     anthropic.Model(orDefault(r.Model, a.cfg.Model)),
		MaxTokens: orInt64(r.MaxTokens, a.cfg.MaxTokens),
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(r.Prompt))},
	}
	if r.System != "" {
		p.System = []anthropic.TextBlockParam{{Text: r.System}}
	}
	oc := anthropic.OutputConfigParam{Effort: anthropic.OutputConfigEffort(orDefault(r.Effort, a.cfg.Effort))}
	if r.Schema != nil {
		oc.Format = anthropic.JSONOutputFormatParam{Schema: r.Schema}
	}
	p.OutputConfig = oc
	return p
}
```

- Thinking is **left unset**: on `claude-opus-5` that runs adaptive by default
  (per the claude-api skill's model table), which is what we want, and setting
  `budget_tokens` would 400.
- **`Response.Text` joins every `TextBlock` and skips `ThinkingBlock`.** Not
  `Content[0].Text` — measured, the first block is a thinking block whenever the
  prompt is non-trivial (`testdata/message-thinking.json`). Pinned by a test that
  serves the recorded two-block envelope and asserts the text is the *text*
  block's; that test fails against a `Content[0]` implementation, which is the
  point of writing it.
- Error mapping: `errors.As(err, &apierr)` where `apierr` is `*anthropic.Error`,
  then `classifyStatus(apierr.StatusCode, err)`; a non-API error (dial failure,
  timeout) is `classifyStatus(0, err)`.
- `Stop == "refusal"` → `ErrRefused` **before** returning text.
- `Stream` uses `client.Messages.NewStreaming`, accumulates with
  `message.Accumulate(stream.Current())`, calls `onDelta` on each
  `anthropic.TextDelta`, and checks `stream.Err()` — **note there is no
  `GetFinalMessage()` in the Go SDK**; the accumulated `message` is the result.
- Both paths stamp `Usage.Duration` from a single `time.Since(start)`, computed in
  one helper so the two paths cannot disagree (ARCH-DRY).
- **`Response.Blocks` is populated on both paths**, preserving thinking blocks and
  their signatures verbatim. The accumulator writes `Text` from the text blocks
  only; nothing else interprets a block's meaning.
- **`StallAfter` wraps the stream read**, not the whole call: a timer reset on
  every delta, firing `ErrUnavailable` naming the phase. Implemented once, in the
  streaming loop — a second copy in `Complete` would be a bound on the wrong thing.
- **`OnSlow` fires from a ticker owned by one helper** shared by both paths, so the
  phase vocabulary cannot fork.
- **We do not probe, restart or heal the proxy.** Parley owns that ladder with its
  own budget and loop guards; our contribution is an error that says *which* thing
  is wrong so the right machinery acts. Adding a second healing path here would be
  a parallel mechanism where one already works.

- [ ] **Step 4: Green** — `go test ./internal/llm/ -count=1 -v`

- [ ] **Step 5: Mutation-check the retry test**

Set `option.WithMaxRetries(0)`; expect `TestRetriesOn429` to FAIL with 1 request.
Restore from a scratch copy. Confirm the mutation **applied and compiled** before
reading the result, and run with `-count=1` (`lessons.md`).

- [ ] **Step 6: Commit**

```bash
git add internal/llm/anthropic.go internal/llm/anthropic_test.go
git commit -m "#11 M1: real client over anthropic-sdk-go, driven at the wire fake"
```

---

### Task 6: The obligation suite

**Files:**
- Create: `internal/llm/llmtest/suite.go`
- Create: `internal/llm/suite_test.go`

- [ ] **Step 1: Write `Suite`**

```go
// Suite is every obligation an llm.Client must meet, stated so that BOTH the
// fake and the live service can satisfy it.
//
// That constraint is what makes it useful: assertions are about SHAPE (text is
// non-empty, usage is non-zero, deltas concatenate to the final text, a bad
// model is ErrRequest), never about specific content, so the same file runs in
// the normal suite against the fake and under -tags conformance against the live
// proxy. "The fake behaves like the real thing" becomes a test rather than a
// claim — the same job storetest.Suite does for Store (ARCH-DRY, ARCH-MOCK).
func Suite(t *testing.T, newClient func(t *testing.T) llm.Client) { … }
```

Obligations: non-empty text + non-zero usage; schema'd request decodes into the
target type; stream deltas concatenate to the final text; a cancelled context
returns promptly with `ctx.Err()`; an unknown model returns `ErrRequest`.

- [ ] **Step 2: Run it against the fake**

`internal/llm/suite_test.go` calls `llmtest.Suite(t, func(t *testing.T) llm.Client { … fake … })`.

Run: `go test ./internal/llm/... -count=1`
Expected: PASS.

- [ ] **Step 3: Commit**

---

### Task 7: The `AGENTS.local.md` carve-out

**Files:**
- Modify: `AGENTS.local.md`

- [ ] **Step 1: Amend the rule in the same change that breaks it**

The rule reads *"`internal/` is earned, not anticipated. Code moves there on the
second consumer, never the first."* Add the recorded exception — an external
service transport, with its seam, stateful fake and conformance suite, is repo
infrastructure from the first consumer, because the alternative is the second
tool copying it. Name the operator decision and its date.

- [ ] **Step 2: Verify nothing else in the file now contradicts it**

Run: `grep -n "internal/" AGENTS.local.md` and read every hit.

- [ ] **Step 3: Commit**

---

### M1 boundary

- [ ] **`sdlc milestone-close --issue 11 --milestone M1`** — the binary
      auto-dispatches the one mandatory fresh-context review (AGENTS.md §3); fix
      Critical/Important before crossing, and log the verdict in `## Log`.

---

## Chunk 2: M2 — typed tasks, goldens, conformance, and a way to check it

### Task 8: Assert the committed captures still describe reality

**Files:**
- Already committed: `internal/llm/llmtest/testdata/{stream-sample.sse,message-thinking.json,message-schema.json}`
- Already committed: `scripts/llm-probe.sh`
- Create: `internal/llm/capture_conformance_test.go` (`//go:build conformance`)

The captures exist — recorded 2026-08-22 from the parley-managed proxy by
`scripts/llm-probe.sh record`, which is committed so the recording is reproducible
rather than a one-off someone did in a terminal once. So this task is no longer
"record them"; it is "detect when they stop being true".

- [ ] **Step 1:** Write a conformance-tagged test that re-runs each probe against
      the live proxy and compares *shape* with the committed capture: the block
      sequence (`thinking`, `text`), the SSE event sequence (`message_start`,
      `content_block_start`, `ping`, `content_block_delta`+, `content_block_stop`,
      `message_delta`, `message_stop`), and that a schema'd request still returns
      decodable JSON. Compare shape, never content — the text differs every run.
- [ ] **Step 2:** Skip, do not fail, when the service is unavailable — the
      `t.Skipf("network unavailable: %v", err)` shape `fetch_conformance_test.go`
      already uses.
- [ ] **Step 3:** On drift, the failure message says `scripts/llm-probe.sh record`.
      A conformance test that detects drift but does not say how to fix it just
      becomes the test everyone skips.

Run: `go test -tags conformance -run Capture ./internal/llm/ -count=1`

### Task 9: `SchemaFor[T]` + golden snapshot

**Files:** `internal/llm/schema.go`, `internal/llm/schema_test.go`,
`internal/llm/testdata/golden/`

- [ ] Reflect a schema from a Go type with `invopop/jsonschema`; memoise per type;
      set `additionalProperties: false` and require every field, so a provider
      that honours the schema cannot return a partial object.
- [ ] Snapshot a representative type's schema to a golden file, with `-update`.
      **Assert the golden is byte-identical**, so a struct field added without
      thought shows up in a diff — which is the whole reason the schema is
      generated rather than hand-written.

### Task 10: `Task[T]` + `Run[T]` + defensive decode

**Files:** `internal/llm/task.go`, `internal/llm/task_test.go`

- [ ] **State the decode strategy once, then test the property rather than
      enumerating inputs.** The strategy: *strip one optional markdown fence,
      decode exactly one JSON value, require the whole payload consumed, require
      every schema-required field present, allow unknown fields.* (Unknown fields
      are allowed deliberately — a provider adding a field must not break us.
      Missing required fields are not, because a half-populated result is worse
      than no result.)
- [ ] **Add a fuzz target**, seeded with the shapes that motivated the strategy:
      a fenced payload, trailing prose after the object, an empty body, an array
      where an object was expected, a truncated object, a doubled object, and a
      real captured response body. The property is invariant and worth stating as
      one line rather than seven cases:

      *`decode` either returns a fully populated `T` and `nil`, or the zero `T`
      and an error matching `ErrMalformed`. It never panics, and it never returns
      a partially populated value with a nil error.*

      Run: `go test ./internal/llm/ -run FuzzDecode -fuzz FuzzDecode -fuzztime 60s`
      Enumerated cases are what a fuzz corpus is for; prose enumerations rot
      because nobody adds the eighth case.
- [ ] `Run[T]` renders the `Task[T]` to a `Request` with `SchemaFor[T]()`, calls
      `Complete`, decodes, and wraps failures with the task name and a truncated
      raw body — a malformed-response error that does not show the response is
      undiagnosable.

### Task 11: `renderRequest` + `llmtest.Golden`

**Files:** `internal/llm/render.go`, `internal/llm/llmtest/golden.go`, their tests

**`renderRequest` is the single source and it comes first** — Task 4's `Cassette`
already hashes it, and this prints it. Two renderers of the same tuple drift, and
the drift is silent in the worst direction: a golden updated for a prompt edit
would leave a cassette that still matches the old hash, so the prompt change would
show in one artifact and not the other (ARCH-DRY).

```go
// renderRequest is THE canonical text form of a Request: model, effort, system,
// prompt, schema, in a fixed order with fixed separators.
//
// Exactly one renderer, consumed by two things that must never disagree:
// llmtest.Golden prints it for humans to diff, and llmtest.Cassette hashes it to
// key a recording. If they rendered separately, a prompt edit could move the
// golden while the cassette kept matching — the change would be visible in one
// artifact and invisible in the other.
func renderRequest(r Request) string
func RequestHash(r Request) string   // sha256 of renderRequest, first 12 hex
```

- [ ] Test that `renderRequest` is **stable under map iteration order** — the
      schema is a `map[string]any`, so it must be marshalled with sorted keys or
      the hash changes run to run and every cassette misses at random.
- [ ] `AssertGolden(t, name, req)` diffs `renderRequest(req)` against
      `testdata/golden/<name>.txt`; `-update` rewrites. Consumers own their golden
      files; this owns the mechanism.
- [ ] Test that a changed prompt moves **both** the golden and the hash — the
      coupling is the requirement, so assert it rather than trusting it.
- [ ] Test that a changed prompt **fails** the assertion (plant the change,
      observe red, remove it, observe green — two runs, not one).

### Task 12: Live conformance

**Files:** `internal/llm/conformance_test.go` (`//go:build conformance`)

- [ ] Run `llmtest.Suite` against a client resolved from the real environment,
      skipping when unavailable.
- [ ] **`output_config` pass-through is now MEASURED** (2026-08-22: a schema'd
      request returned JSON that decoded first try; capture committed at
      `testdata/message-schema.json`). The conformance run therefore *guards* it
      rather than discovering it: assert the response still decodes, and fail
      loudly if it stops — a silent regression here degrades every authored item
      to free-text parsing.
- [ ] **Assert the injected preamble has not changed materially.** Measured at
      ~1,900 cached tokens (Claude Code's system prompt, via the proxy's OAuth
      path). Assert it is present and within a broad band; a sudden change means
      the proxy's upstream shape moved, and every prompt in the repo was tuned
      against the old one.
- [ ] Document the cadence in the test's doc comment, matching
      `fetch_conformance_test.go`'s: on-demand, not in merge-check.

### Task 13: `define --llm-check`

**Files:** `cmd/define/llmcheck.go`, `cmd/define/llmcheck_test.go`, `main.go` (flag + usage)

- [ ] Runs one trivial task through the real transport; prints resolved base URL,
      model, latency, tokens, **the injected-preamble token count**, and a verdict.
      Prints `Redact(key)`, never the key — asserted by a test that greps the
      output for the key literal.
- [ ] The no-key error names both env vars **and** where the parley-managed proxy
      keeps its key (`~/.local/share/nvim/parley/cliproxy/config.yaml`). A
      diagnostic that says "unavailable" without saying what to do is half a
      diagnostic; this is a hint in a message, not a dependency on parley.
- [ ] Unavailable → a **non-zero exit** with the specific reason (no key vs.
      refused vs. unreachable). This is a diagnostic; a cheerful empty result is
      the failure mode `AGENTS.local.md` explicitly forbids.
- [ ] Driven through `run()` — not by calling the helper directly. A wiring only
      the loop shell supplies must be pinned by a test that drives that shell
      (`lessons.md`, define #15).

### Task 14: Atlas

**Files:** `atlas/llm.md`, `atlas/index.md`

- [ ] One page: the seam, the taxonomy, the fake's placement and why, the
      conformance cadence, the config precedence. Current state only — describe
      what exists after this issue, not what #10–#17 will do with it.
- [ ] Link it from `atlas/index.md` under a new **Libraries** heading.

### Issue close

- [ ] `sdlc close --issue 11 --verified '<evidence>'` — the boundary review runs
      here. Let `--actual` be **measured**, not typed.

---

## Open question for the operator

**The live conformance check cannot pass until `cli-proxy-api` accepts a key from
us.** Measured today: `~/.cli-proxy-api/config.yaml` declares no `api-keys:`, and
the running v7.1.71 rejects both an absent key (`Missing API key`) and an
arbitrary one (`Invalid API key`).

Nothing in M1 is blocked — it all runs against the fake. Blocked in M2: Task 8
(recording the real SSE sample) and Task 12 (live conformance). Both are
`t.Skipf`-shaped, so the suite stays green; they would simply not be *evidence*.

Either add an `api-keys:` entry to the proxy config and tell me the value to put
in `DEFINE_LLM_API_KEY`, or point `DEFINE_LLM_BASE_URL` at `api.anthropic.com`
with a direct key for conformance runs. I will not hand-write the SSE sample as a
substitute — a fake modelling invented behaviour is the failure ARCH-MOCK exists
to prevent, and it would be discovered by #16 at the worst possible time.

---

## Revisions

### 2026-08-22 — cassettes replace invented replies; proxy facts corrected

**Reason.** Operator asked two things this plan answered badly: *"mocking an LLM
maybe hard? what's your plan to mock?"* and *"which proxy are you hitting, is it
the one managed by parley or one installed locally"*. The second exposed a
measurement error, and the first exposed a design weakness.

**Delta 1 — the proxy identity was wrong, and the `## Open question` is retracted.**

The running instance is **parley-managed**:

```
/Users/xianxu/.local/share/nvim/parley/cliproxy/bin/cli-proxy-api
  -config /Users/xianxu/.local/share/nvim/parley/cliproxy/config.yaml
```

Measured fact 2 above read `~/.cli-proxy-api/config.yaml` instead — the homebrew
install's config, which the operator has since removed. Parley's config declares
`api-keys`, and parley's instance is the target from here (operator, 2026-08-22:
it carries auto-healing the standalone install does not). **Nothing is blocked.**
Verified live: `claude-opus-5` answers, and 31 models are reachable including
`claude-fable-5` and `claude-sonnet-5`, so `defaultModel = "claude-opus-5"` stands.

**Delta 2 — the proxy speaks both protocols.** `internal/api/server.go:429-446`
routes OpenAI (`/v1/chat/completions`, `/v1/completions`, `/v1/responses`) *and*
Anthropic (`/v1/messages`, `/v1/messages/count_tokens`). We take the Anthropic
path because we use the Anthropic SDK; recorded because the OpenAI path is how
parley itself drives it, so "which protocol does cliproxyapi use" has two right
answers and only one of them is ours.

**Delta 3 — two named risks retired by measurement.**

- `output_config.format` **passes through**. A schema'd request returned JSON that
  decoded first try. The Risks table's first row is answered: keep the defensive
  decode (an intermediary could still drop it), but it is no longer an open
  question.
- The SSE frame sequence is **captured**: `message_start`, `content_block_start`,
  `ping`, `content_block_delta`×N, `content_block_stop`, `message_delta`,
  `message_stop` — with `data:` payloads right-padded with spaces, and a `ping`
  event that hand-written frames would have omitted. Task 8 drops from "record the
  sample" to "commit the captured sample and assert it still matches".

**Delta 4 — new measured fact: the proxy prepends ~1,900 tokens of system prompt.**

A 22-token user message reported `cache_creation_input_tokens: 1902`, and the next
call reported `cache_read_input_tokens: 1902`. That is Claude Code's own system
prompt riding the OAuth path. Consequences, both worth stating before anyone tunes
a prompt: our `Request.System` is **additive to a preamble we do not control**, and
token accounting for a task will never be just our prompt. Cost impact is nil (it
caches); behavioural impact is not. `--llm-check` should print it so it is visible
rather than folklore.

**Delta 5 — the fake replays RECORDED responses, not invented ones.**

The original Task 4/5 scripted replies as literals (`Reply{Text: "flattering,
servile"}`). That is *my guess at what a model says*, and every consumer test for
#12 and #13 would then assert against that guess. Three things a fake must be kept
honest about, and only the first two are testable at all:

| concern | mechanism | model involved |
|---|---|---|
| transport correctness — protocol, retry, SSE, degradation, redaction | wire fake | no |
| prompt stability — did this prompt change unintentionally | golden files | no |
| output quality — is the authored question any good | **not a test** — see below | yes |

So: **`llmtest.Cassette`**. A recorded real response, keyed by a stable hash of the
rendered `Request` (model, effort, system, prompt, schema), stored at
`internal/llm/llmtest/testdata/cassettes/<hash>.json`, replayed byte-for-byte.

- `-record` re-runs the request against the live proxy and rewrites the cassette;
  the diff is then a visible record of how the model's answer moved.
- A **miss** in non-record mode is a loud failure naming the task and the hash, not
  a fallback. A prompt edit therefore *cannot* silently pass against a stale
  recording.
- `Fake.Script` survives for the cases where an invented body is the point —
  scripted `429`s, a refusal, a truncated payload. Those are transport shapes, and
  inventing them is correct.

Limits, stated now rather than discovered by #12: a cassette is **one sample of a
stochastic process**, so it pins *our handling* of a real answer, not the model's
reliability at producing one; and cassettes go stale as models move, which is
exactly what the live conformance run exists to detect (ARCH-MOCK).

**Delta 6 — output quality is explicitly out of scope here.** It is an evaluation
problem, not a transport problem, and its home is #10's material-quality
checkpoint. Recorded so that "the harness has tests" is never mistaken for "the
generated questions are good" — different claims, different evidence.

**Task list changes.** Task 4 gains `Cassette` and loses its invented reply
literals; Task 5's tests use cassettes for content and `Script` for failure shapes;
Task 8 becomes "commit the captured SSE sample + assert no drift"; Task 12 adds an
assertion that the recorded preamble size has not changed materially; Task 13
prints the preamble token count.

### 2026-08-22 — plan-quality gate findings, and the robustness bar from prior art

**Reason.** `sdlc change-code` returned six blocking findings; separately the
operator raised the bar — *"library level robust, as it will form the base of many
projects"* — and pointed at two working implementations to learn from.

**Gate findings, each fixed at the class rather than the site.**

- *PQ-3 (the sharpest).* The fake's envelope was single-text-block; `opus-5`
  returns `[thinking, text]` on any non-trivial prompt, so `Content[0].Text` would
  return `""` live while the whole suite passed. **Verified by measurement** — the
  first probe used a trivial prompt and returned `["text"]` alone, which is exactly
  how the wrong envelope nearly shipped. Fixed in three places, not one: the fake
  emits thinking-then-text, `Response.Text` joins text blocks and skips thinking,
  and a regression test serves the committed capture and fails against a
  `Content[0]` implementation.
- *PQ-2.* The captures now exist in the repo —
  `internal/llm/llmtest/testdata/{stream-sample.sse,message-thinking.json,message-schema.json}`
  plus `scripts/llm-probe.sh` that recorded them, committed so the recording is
  reproducible rather than something someone did in a terminal once. Task 8 becomes
  drift detection.
- *PQ-4.* One `renderRequest`, printed by `Golden` and hashed by `Cassette`. Two
  renderers would drift in the worst direction: a prompt edit visible in the golden
  and invisible to a still-matching cassette.
- *PQ-5.* Seven prose decode cases became one stated strategy plus a fuzz target
  with an invariant. Prose enumerations rot; nobody adds the eighth case.
- *PQ-6.* The relocated Done-when now exists in every destination — #6 (loop
  degrades), #13 (form skipped, distinct from its malformed/refused row), #12
  (already had it, verified rather than assumed), and #11's own row is struck in
  place with the reason visible.
- *Minors.* Ordered matcher list (a ranged map served random replies when two keys
  matched); `go mod tidy` moved after the first import (with no importer it removes
  what was just added).

**The robustness bar.** New `## Robustness bar` section above, with the findings
transferred from `kbench/.../metis/model.py` and `parley.nvim`. Contract changes it
forced: `Response.Blocks` (thinking preserved verbatim for #16's multi-turn
round-trip), `Usage.ThinkingTokens` / `Usage.PreambleTokens`, `Config.StallAfter`,
`Config.OnSlow` + `Progress`.

Two conclusions worth keeping separate from the code:

1. **Go gets most of it for free, and that is a reason to test it, not to skip it.**
   `net/http` honours ctx through the header read, and the SDK selects on
   `ctx.Done()` during backoff (`requestconfig.go:450-478`, read not recalled), so
   one deadline bounds attempts + backoff + body. That is four Python versions'
   worth of pain that does not reproduce here — so kbench's exact failure mode
   (a peer dribbling headers) is now a test, as the alarm for if that ever changes.
2. **We do not heal the proxy.** Parley owns a health probe, a recovery ladder,
   one-shot repair guards and a wall-clock repair budget. Building a second one
   would be a parallel mechanism where one already works; ours is to say *which*
   thing is broken so parley's machinery acts on the right one.

The one bound Go does not provide is stall detection inside a stream, because a
total deadline cannot express "minutes for a long answer, seconds for a dead
connection". `StallAfter` is that, and it is implemented once in the streaming loop.

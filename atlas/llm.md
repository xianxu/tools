# llm

The one way anything in `tools/` talks to a language model. One transport, one
error taxonomy, one stateful fake, one obligation suite.

**Public from v0.2.0 (tools#85):** `github.com/xianxu/tools/llm`, with `llm/llmtest`
(fake, cassettes, goldens, `Suite`) and `conformance` (the skip-or-fail policy
`llmtest` imports). echo consumes it as an ordinary module; v0.x may break at a
minor bump, so consumers pin a tag. Earlier tags (v0.1.0 to v0.1.8) predate it.

It owns **no prompts**. A prompt is domain knowledge and lives with the consumer
that needs it; this package owns the wire and the contract. That split is why
it became shared infrastructure on its first consumer, under a repo rule that
says `internal/` is earned on the second, and why it could go public for another
repository: see the carve-out in `AGENTS.local.md`.

## The seam

```go
type Client interface {
    Complete(ctx, Request) (Response, error)
    Stream(ctx, Request, onEvent func(Delta)) (Response, error)
}
```

Nothing in `Request`/`Response` names Anthropic. That is what lets the fake, the
real client and the live conformance run share one obligation suite.

`Stream` returns the final `Response` too, so a caller never needs a second call
to learn what the answer cost. `onEvent` runs synchronously on the calling
goroutine and never after `Stream` returns. Thinking is accumulated into `Blocks`
but never forwarded. A text-only caller wraps its callback: `llm.TextOnly(fn)`.

**Stability and concurrency.** v0.x may break at a minor bump, so consumers pin
a tag. A `Client` from `New` and an `llmtest.Fake` are safe for concurrent use:
calls share no per-call state, so events never cross callbacks and cancelling
one call touches no other (`Suite`'s `ConcurrentStreams` obligation, under `-race`).

### Conversations and tools (tools#85)

`Request.Messages` is the history, oldest first. `Prompt`, when set, is appended
as a final user turn, so a single-turn caller is unchanged. `Message.Content`
reuses `Block`: `resp.Message()` returns the assistant turn ready to append, and
each block's `Raw` goes back as received, so thinking signatures survive. `Raw` is
opaque and carry-only: a consumer builds outgoing blocks with `llm.Text` and
`llm.ToolResult`, never by authoring provider JSON. `wire.go` is the one
translation to the Anthropic shape, and `checkRequest` refuses what can't be sent
(`ErrRequest`, nothing on the wire): an empty request, a tool named
`auto`/`any`/`none` (keywords share `ToolChoice`'s string), or a `ToolChoice`
naming no declared tool. `Response.ToolCalls` lists every `tool_use` block, and
`Stop == "tool_use"` is a success. A tool input that isn't a JSON object is
`ErrMalformed` on both paths (one check, `toolInputObject`).

### Stream events

`Delta{Kind, Index, Text, Call}` comes in four kinds: `DeltaText`,
`DeltaToolCallStart`, `DeltaToolCallInput` (a fragment of the input JSON) and
`DeltaToolCallEnd` (complete, validated, byte-equal to `Response.ToolCalls[i]`).
Events arrive in wire order and interleave by content-block `Index`. The per-call
`deltaState` appends each fragment once and parses each call once, so a call is
O(fragments + bytes); `FuzzDeltaAccumulator` checks it against the SDK's own
accumulator. Every delivered value is freshly allocated, so the receiver owns it
and may enqueue it without copying.

### Cancellation

The caller cancels `ctx`, and that is the only control. If the caller's own
context ended, the error is `ErrCanceled` wrapping `ctx.Err()`. The package's own
`Timeout` and `StallAfter` keep their mapping (`ErrUnavailable` before any frame,
`ErrTruncated` after). They are told apart by cause: every internal deadline
carries `errBudget` (`context.WithTimeoutCause`). Without it, the auto client's
`Timeout` would read as the caller's deadline. A failed or cancelled stream's
partial `Response` holds only **complete** state: blocks that reached their stop,
the text so far, and the calls that reached `End`. A half-streamed tool call, or a
thinking block cut before its signature, is dropped, never half-reported.

### Retries and admission

The transport owns all retries: the SDK's loop, 2 retries, on 408/409/429/5xx.
A stream is retried only before its first frame. `Config.BeforeAttempt` is SDK
middleware, so it runs once per **physical** attempt inside that loop. It may
block to throttle, bounded by ctx. A rejection sends nothing and is not retried.
The SDK would retry a middleware that returns no response, so the rejection is
recorded in the call's `callState` and the call's derived context is cancelled;
the SDK's loop exits on `ctx.Err()` before backoff. The error is `ErrUnavailable`
wrapping the hook's, so `errors.Is` reaches both. `Usage.Attempts` counts
dispatched attempts on every return path (0 when the first is rejected).
`outcome` decides how a failed call ended, in one fixed order: rejection, then
caller cancel, then the transport's mapping.

## Where it points

`Config.BaseURL` defaults to `http://127.0.0.1:8317`, the **parley-managed**
`cli-proxy-api` (`~/.local/share/nvim/parley/cliproxy/`), which fronts a
subscription plan and carries auto-healing. `api.anthropic.com` is reached by
setting `DEFINE_LLM_BASE_URL`.

Precedence, all resolved by a pure `Resolve(getenv)`:

| setting | env | default |
|---|---|---|
| base URL | `DEFINE_LLM_BASE_URL` | `http://127.0.0.1:8317` |
| key | `DEFINE_LLM_API_KEY` → `ANTHROPIC_API_KEY` | `parley-local` **for the local proxy only**; anywhere else, none → `ErrUnavailable` |
| model | `DEFINE_LLM_MODEL` | automatic discovery at the default local endpoint; `claude-opus-5` for other endpoints |
| effort | `DEFINE_LLM_EFFORT` | `high` |
| timeout | `DEFINE_LLM_TIMEOUT` (a duration, e.g. `90s`) | 5m |

**The local proxy supplies its own key**, so `define` answers questions with no
environment at all on a machine where the parley-managed cliproxyapi runs. That
token is a loopback HANDSHAKE rather than a credential: cliproxyapi's `api-keys`
is an inbound allowlist — it authenticates its callers and 401s without a match —
and parley renders the same constant into the proxy's config and sends it as the
bearer.

Hardcoding a peer's constant is what `ParseLang` and `-locale` refuse, and the
difference is whose fact it is at what cost. The ENDPOINT was already parley's
fact, hardcoded deliberately; without the token that default was inert, so every
operator whose parley worked still met "no model configured" and had to export a
secret a program on the same machine already knew. If parley changes it, the
failure is a 401 naming the key — the same shape as the missing-key path, not a
worse one.

**Only for that endpoint.** A remote provider, or another local port, still
declines: a token invented here would be a credential sent to a host we know
nothing about. An explicit key always wins.

**No startup probe.** Reachability is never checked when `define` starts — that
would put a network round trip on the definition path, which must stay instant
and offline. An unreachable proxy surfaces as `ErrUnavailable` at the first real
call, the same path as having no key, so there is one degradation story.

**Provider-first discovery.** `Resolve` records automatic intent in
`Config.AutoModel`; it stays pure. `New` creates an `autoClient` which fetches
authenticated `GET /v1/models` on its first Complete/Stream request. The pure
`SelectModel` in `models.go` ranks ownership before model versions: direct Claude
(`anthropic`) Opus, then Codex (`openai`, GPT-5.6 before GPT-6 before other
recognized GPT versions), then Antigravity (`antigravity`, Flash before Pro).
Aliases outside the supported numeric grammars require an explicit model.
The proxy can collapse identical IDs across providers; its returned ownership
is the routing view used here, not a complete inventory of configured accounts.

`discovery.go` bounds the catalog to 1 MiB/4096 entries and five seconds within
the total request deadline, validates identifiers, and refuses redirects.
`auto.go` owns one discovery flight, with cancellable waiters, retry after a
failed flight, and a successful immutable delegate retained for the client
lifetime. It creates no background goroutines or durable cache. An explicit
config/request model bypasses discovery; inference errors do not switch models.

`SelectionOf` reports the pinned default without IO. Diagnostics and reflection
use it for selected-model reporting/provenance. Unknown custom Client
implementations have no reporter; reflection retains its configured fallback.
For auto-selected Codex/Antigravity the effective request carries adaptive
thinking and schema-derived JSON instructions because the proxy's Claude-source
translators omit the JSON-schema output setting. The same effective request feeds
both wire parameters and cassette hashing; typed decoding remains the local
validation boundary, not a claim of upstream schema enforcement.

**This package does not heal the proxy.** Parley owns that ladder, with a repair
budget and one-shot guards. Ours is to say *which* thing is wrong.

## The taxonomy

The load-bearing distinction is the first two: one is absorbed, one must not be.

| error | means | caller |
|---|---|---|
| `ErrUnavailable` | no key, no network, 401/403, and anything the transport itself retries (408, 409, 429, every 5xx) | degrade quietly |
| `ErrRequest` | any other 4xx — **our** bad schema/model/body | stay loud |
| `ErrRefused` | the model declined (`StopDetails` says why) | skip |
| `ErrTruncated` | cut off: `max_tokens`, or a stream that died mid-reply | skip |
| `ErrMalformed` | did not decode, or an unknown `stop_reason` | skip |

The transient set is **derived** from the SDK's own retry policy rather than
restated, which is what keeps 408 from being retried twice by the transport and
then reported as our bad request.

`ErrTruncated` exists because **a truncated structured answer usually parses** —
the committed specimen decodes to `{"verdict":"yes","reason":": Ā"}` with every
required field present. Only the stop reason can catch that, so the transport is
the only place it can be caught.

A stream that dies mid-reply returns its **partial text** with `ErrTruncated`
rather than `ErrUnavailable`: once a frame has arrived the service is
demonstrably reachable, and "skip this question" and "stop trying for a while"
need opposite responses.

## Bounds

- `Timeout` (default 5m) is the **total** budget — attempts, backoff and body
  read — enforced as a context deadline. Go's `net/http` honours ctx through the
  header read and the SDK selects on `ctx.Done()` during backoff, so one deadline
  covers every phase. Asserted, not assumed: a test drives a peer that dribbles
  headers one byte at a time.
- `StallAfter` (default 90s) bounds **silence inside a stream**, which a total
  deadline cannot express — minutes of headroom for a long answer, seconds of
  patience for a dead connection. The one bound the SDK does not provide. Zero
  means "use the default" (`New` applies it); a **negative** value disables it.
  `New` defaults every `Config` field, so a minimal `Config{BaseURL, APIKey}`
  still gets the stall bound.
- Retries are the SDK's: 408, 409, 429 and every 5xx, honouring `retry-after`.

## Typed tasks

A consumer writes a prompt and a result type; nothing else.

```go
type verdict struct {
    Fits   bool   `json:"fits"`
    Reason string `json:"reason"`
}
v, err := llm.Run(ctx, client, llm.Task[verdict]{Name: "veto-distractor", Prompt: ...})
```

`SchemaFor[T]` reflects the JSON Schema from the result type (memoised per type,
`additionalProperties:false`), so the struct is the single source of the shape.

**`Run` checks the stop reason BEFORE decoding.** Not an ordering nicety: a
truncated structured answer commonly parses, so a decode-first implementation
returns success on garbage and nothing downstream can tell.

`decode`'s strategy, once: strip one optional markdown fence, decode exactly one
JSON value, require the payload **consumed to EOF**, require every field the
schema marks required — at every point of the schema tree — and allow unknown
fields. The walk is driven by the generator's nesting vocabulary (`properties`,
`items`, `additionalProperties`), not by payload shapes, because four successive
fixes each covered the example a finding used and missed the next one;
`TestSchemaKeywordsAreCovered` fails if a schema ever arrives carrying a keyword
the walk does not know. The required check derives its
set from `SchemaFor[T]` rather than restating it — without it `encoding/json`
zero-fills, so `{}` decodes to a veto verdict of `Fits:false` that a consumer
cannot tell from a real "no".

Its invariant — a fully populated `T` and `nil`, or the zero `T` and
`ErrMalformed`, never a partial value, never a panic — is asserted on **both**
branches by `FuzzDecode`.

## Testing: `llmtest`

The fake is an **httptest server speaking the Anthropic wire protocol**, not a
stubbed `Client`. Placement decides what a test can see: a stubbed `Client` sits
above a mis-serialized `output_config`, a dropped header, a retry that re-sends a
consumed body, and an SSE frame the parser mishandles.

**Content comes from committed captures, never literals** — you cannot fake
judgment, but you can freeze a real answer and pin our handling of it. Only
transport shapes are invented (a 429, a refusal, a stall), because those are
protocol.

`llmtest.Suite` is the obligation set, run against the fake in the normal suite
and against the live proxy under `-tags conformance`. Assertions are about shape,
never content, so both backends can satisfy them — which is what makes "the fake
behaves like the real thing" a test rather than a claim.

**Goldens** snapshot a request's canonical form via `AssertGolden`, which renders
through the single `renderRequest` that `RequestHash` also hashes. A committed
example lives at `llm/testdata/golden/schema-veto-verdict.txt`; it is what
makes reflecting the schema safe, because a struct field added without thought
shows up in its diff.

A **cassette** is a real exchange frozen at the WIRE — request body, status,
content type and response body — keyed by `RequestHash`, the same renderer
`AssertGolden` prints. The in-flight `Request` reaches the transport by context,
because `Task` is never sent: keying on the wire body alone made two Requests
differing only in `Task` collide, and the second recording overwrote the first.

Streaming records too — the body is stored as text with its content type, so
replay runs the SDK's SSE parser and thinking signatures survive. And a problem
with the *harness* (a missing or corrupt recording) is returned as `ErrRequest`,
never `ErrUnavailable`: a broken cassette that reads as an outage looks exactly
like flight mode.

It sits **beneath** the seam, as an `http.RoundTripper` (`Config.Transport`), for
the same reason the fake does: replacing `Client` would mean a replayed test never
serialises a request, never runs a retry and never parses SSE. A consequence worth
naming — the taxonomy survives replay *structurally*, because a recorded 400 is
replayed as a 400 and reaches `classifyStatus` as `ErrRequest`. Re-deriving it
from a stored string is how a loud error becomes a quiet one. You cannot fake
judgment; you can freeze a real answer and pin our handling of it. A miss fails
loudly naming the task and path — never a fallback, because falling back is how an
edited prompt comes to pass against a recording of the question it no longer asks.
`-update` re-records against the live service. What a cassette does *not*
establish is that the model reliably produces that answer: it is one sample of a
stochastic process, which is what the conformance run is for.

`llm/llmtest/testdata/README.md` tabulates the captures and the rule
they enforce: *a capture is evidence only for the shape its recording conditions
elicit.* Regenerate with `scripts/llm-probe.sh record`, which stages, verifies,
and only then promotes.

### Fake controls for consumers (tools#85)

`Reply.ToolCalls` serves invented `tool_use` blocks, which is protocol shape, not
judgment. On a stream each input is cut across several `input_json_delta` frames
after `Text`, and `Stall` goes silent inside the last call. `Script` matches the
last user turn's text **and** tool-result content. `Recorded` adds `Messages()`,
`Tools()` and `ToolChoice()`, and `Fake.Attempts()` counts physical requests. The
barriers (`Started`/`Release`, `AfterText`/`FinishRelease`) are how a consumer
cancels deterministically mid-call. `Suite` obligations added: tool-call events
agree with the response, a conversation continues from `Message()`, a cancelled
stream ends in `ErrCanceled` or a complete success (the legal race), and
`ConcurrentStreams`.

## Checking a configuration

`define --llm-check` runs one trivial task through the real transport and reports
base URL, model, latency, tokens, the injected preamble size, and the answer.
Non-zero and specific when unavailable — the one surface where the seam is LOUD,
since every other model-shaped feature degrades silently by design.

```
  base url  http://127.0.0.1:8317
  model     auto (effort high)
  key       parley…ocal
  provider  anthropic
  model     claude-opus-5 (effort high)
  latency   1.379s
  tokens    22 in, 5 out (0 thinking)
  preamble  1902 tokens injected upstream (not ours)
  answer    "PONG"
  ok
```

## Conformance

Five tagged suites, all on-demand (`-tags conformance`), none in merge-check:

- `TestConformanceAgainstTheLiveService` — the obligation suite against the real
  service, so "the fake behaves like the real thing" is a test.
- `TestTypedTaskAgainstTheLiveService` — `Run[T]` end to end against the real
  model, flat and nested result types. The only place the layer consumers write
  against is proven rather than argued: a Go struct in, a populated struct out,
  through a reflected schema and a defensive decode. Asserts shape, never the
  model's judgment — whether a verdict is *correct* is #12's problem.
- `TestCaptureDriftAgainstTheLiveService` — do the committed captures still
  describe reality: thinking blocks still returned, `output_config` still passing
  the proxy, signature deltas still in the stream, preamble still ~1,900 tokens, and a streamed tool call's input still arriving in several frames (which the fake's chunking models).
  Every failure names `scripts/llm-probe.sh record`, because a drift test that
  doesn't say what to do becomes the test everyone skips — and an **unreachable**
  service skips rather than reporting drift, since "not running" is not "changed".
- `TestConformanceAutoSelection` — discovers the default local provider and
  exercises plain, streaming, and schema-shaped answers with the selected model.
  Missing configuration/providers use the shared conformance policy: skip by
  default, fail when `CONFORMANCE_STRICT` is set. It verifies the selected provider
  only; other advertised providers are not implicitly certified by that run.
- `TestBeforeAttemptAgainstTheLiveService`: the admission hook against the real
  transport. It numbers attempts from 0, `Usage.Attempts` matches, and a rejection
  sends nothing. Kept out of `Suite`, which receives a built client.

## Known limitations

- **An unknown model reads as `ErrUnavailable`, not `ErrRequest`.** Measured: the
  proxy answers `502` where `api.anthropic.com` answers `400`, so a typo — our
  bug — is absorbed rather than loud. The fake models the 502 so the suite does
  not diverge from reality.
- **A malformed SSE frame ends the stream.** The SDK's decoder owns framing;
  there is no skipping a frame it refuses. Partial text is salvaged.
- **The proxy prepends ~1,900 cached tokens of system preamble** (Claude Code's,
  via its OAuth path). `Request.System` is therefore *additive* to a prompt we do
  not control, and `Usage.PreambleTokens()` reports it.


## Foreground waiting display

The transport keeps UI outside its seam. Define's `activityClient` decorates
foreground calls and uses a shared Braille display lease, including lazy model
discovery before inference. It clears on the first nonempty answer delta or
Complete return. Callers query `SelectionOf` on the original client, preserving
its private selection metadata. `llmtest.Reply` provides cancellable barriers
before the response and before/after its first text delta; these hold recorded
responses without changing their bytes, so consumer tests verify waiting and
cleanup against the real wire client.

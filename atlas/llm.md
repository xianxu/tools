# internal/llm

The one way anything in `tools/` talks to a language model. One transport, one
error taxonomy, one stateful fake, one obligation suite.

It owns **no prompts**. A prompt is domain knowledge and lives with the consumer
that needs it; this package owns the wire and the contract. That split is why
`internal/llm` exists at all under a repo rule that says `internal/` is earned on
the second consumer — see the carve-out in `AGENTS.local.md`.

## The seam

```go
type Client interface {
    Complete(ctx, Request) (Response, error)
    Stream(ctx, Request, onDelta func(string)) (Response, error)
}
```

Nothing in `Request`/`Response` names Anthropic. That is what lets the fake, the
real client and the live conformance run share one obligation suite.

`Stream` returns the final `Response` too, so a caller never needs a second call
to learn what the answer cost. `onDelta` receives **answer text only** — thinking
deltas are accumulated into `Blocks` but never forwarded.

## Where it points

`Config.BaseURL` defaults to `http://127.0.0.1:8317`, the **parley-managed**
`cli-proxy-api` (`~/.local/share/nvim/parley/cliproxy/`), which fronts a
subscription plan and carries auto-healing. `api.anthropic.com` is reached by
setting `DEFINE_LLM_BASE_URL`.

Precedence, all resolved by a pure `Resolve(getenv)`:

| setting | env | default |
|---|---|---|
| base URL | `DEFINE_LLM_BASE_URL` | `http://127.0.0.1:8317` |
| key | `DEFINE_LLM_API_KEY` → `ANTHROPIC_API_KEY` | none → `ErrUnavailable` |
| model | `DEFINE_LLM_MODEL` | `claude-opus-5` |
| effort | `DEFINE_LLM_EFFORT` | `high` |

**No startup probe.** Reachability is never checked when `define` starts — that
would put a network round trip on the definition path, which must stay instant
and offline. An unreachable proxy surfaces as `ErrUnavailable` at the first real
call, the same path as having no key, so there is one degradation story.

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
JSON value, require the payload **consumed to EOF**, allow unknown fields, reject
missing ones. Its invariant — a fully populated `T` and `nil`, or the zero `T` and
`ErrMalformed`, never a partial value, never a panic — is held by a fuzz target.

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

**Goldens and cassettes are two views of one request**, both deriving from the
single `renderRequest`: `AssertGolden` prints it, `RequestHash` hashes it. Two
renderers would drift in the worst direction — a prompt edit visible in the golden
while a stale cassette kept matching. `MaxTokens` deliberately stays out of the
hash: it changes how much room the answer had, not what was asked.

A **cassette** is a real response frozen and keyed by that hash. You cannot fake
judgment; you can freeze a real answer and pin our handling of it. A miss fails
loudly naming the task and path — never a fallback, because falling back is how an
edited prompt comes to pass against a recording of the question it no longer asks.
`-update` re-records against the live service. What a cassette does *not*
establish is that the model reliably produces that answer: it is one sample of a
stochastic process, which is what the conformance run is for.

`internal/llm/llmtest/testdata/README.md` tabulates the captures and the rule
they enforce: *a capture is evidence only for the shape its recording conditions
elicit.* Regenerate with `scripts/llm-probe.sh record`, which stages, verifies,
and only then promotes.

## Checking a configuration

`define --llm-check` runs one trivial task through the real transport and reports
base URL, model, latency, tokens, the injected preamble size, and the answer.
Non-zero and specific when unavailable — the one surface where the seam is LOUD,
since every other model-shaped feature degrades silently by design.

```
  base url  http://127.0.0.1:8317
  model     claude-opus-5 (effort high)
  key       (set, short)
  latency   1.379s
  tokens    22 in, 5 out (0 thinking)
  preamble  1902 tokens injected upstream (not ours)
  answer    "PONG"
  ok
```

## Conformance

Two tagged suites, both on-demand (`-tags conformance`), neither in merge-check:

- `TestConformanceAgainstTheLiveService` — the obligation suite against the real
  service, so "the fake behaves like the real thing" is a test.
- `TestCaptureDriftAgainstTheLiveService` — do the committed captures still
  describe reality: thinking blocks still returned, `output_config` still passing
  the proxy, signature deltas still in the stream, preamble still ~1,900 tokens.
  Every failure names `scripts/llm-probe.sh record`, because a drift test that
  doesn't say what to do becomes the test everyone skips.

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

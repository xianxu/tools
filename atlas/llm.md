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
| `ErrUnavailable` | no key, no network, 429, 5xx | degrade quietly |
| `ErrRequest` | a 4xx — **our** bad schema/model/body | stay loud |
| `ErrRefused` | the model declined (`StopDetails` says why) | skip |
| `ErrTruncated` | cut off: `max_tokens`, or a stream that died mid-reply | skip |
| `ErrMalformed` | did not decode, or an unknown `stop_reason` | skip |

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
  patience for a dead connection. The one bound the SDK does not provide.
- Retries are the SDK's: 408, 409, 429 and every 5xx, honouring `retry-after`.

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

`internal/llm/llmtest/testdata/README.md` tabulates the captures and the rule
they enforce: *a capture is evidence only for the shape its recording conditions
elicit.* Regenerate with `scripts/llm-probe.sh record`, which stages, verifies,
and only then promotes.

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

---
id: 000019
status: open
deps: []
github_issue:
created: 2026-08-23
updated: 2026-08-23
estimate_hours:
---

# an overloaded upstream reads as our bug: 200 with an error body classifies as ErrRequest

## Problem

Measured while hand-verifying #16 M2 against the live proxy, 2026-08-23:

```
$ printf 'is it always negative?\n' | define -no-audio
define: llm: bad request: POST "http://127.0.0.1:8317/v1/messages": 200 OK
{"type":"error","error":{"details":null,"type":"overloaded_error","message":"Overloaded"},
 "request_id":"req_011CeM2bi5qDqpw8WwqYpAcH"}
```

The service was **overloaded** — transient, and squarely what `ErrUnavailable`
exists for ("degrade quietly"). It reached the user as `ErrRequest`, whose whole
contract is the opposite: *our* bad schema/model/body, stay loud. So `define`
printed a raw JSON blob and exited 1 for a condition it should have absorbed the
way it absorbs a dead proxy.

The mechanism: the proxy answers **HTTP 200** with an error body. `classifyStatus`
(`internal/llm/errors.go:55`) switches on the STATUS, and 200 is neither
transient nor 401/403, so it falls to the `default:` arm — `ErrRequest`. Nothing
retries it either, since the SDK's retry policy is also status-driven.

This is the same shape as the known limitation already recorded in `atlas/llm.md`
— *"an unknown model reads as ErrUnavailable, not ErrRequest"* — where the proxy's
status differs from `api.anthropic.com`'s. Both are the proxy putting the truth
somewhere the status does not carry it.

## Spec

`classifyStatus` cannot answer this alone: the signal is in the BODY. The fix has
to read the error body's `type` and let a transient one (`overloaded_error`,
`rate_limit_error`, `api_error`) reach `ErrUnavailable`, while anything else at
200 stays whatever the status says.

Two obligations come with it, and they are the reason this is an issue rather
than a five-line patch:

- **`llmtest` must model the shape** — a 200 carrying an error body — or the fake
  diverges from the service on exactly the case that motivated the change
  (ARCH-MOCK). `Reply` today can script a Status; it cannot script a 200 whose
  body is an error.
- **The capture-drift suite should know about it**, since "the proxy answers 200
  for an overload" is a claim about the real service that can change.

Worth deciding at the same time: whether a 200-with-error-body should be
RETRIED rather than merely absorbed. `Timeout` is a total budget and the SDK
honours `retry-after`, so a retry is cheap and an overload is exactly the case
that clears — but retrying inside the transport changes the latency story for
every consumer, so it is a decision, not an implementation detail.

## Done when

- [ ] A 200 carrying `overloaded_error` classifies as `ErrUnavailable`, asserted
      against the wire fake rather than a stubbed client.
- [ ] The non-transient 200-with-error-body cases still classify as before —
      the enumeration of body types is written down, not assumed.
- [ ] `llmtest` can script the shape, and the obligation suite covers it against
      both the fake and (under `-tags conformance`) the live service.
- [ ] `atlas/llm.md`'s taxonomy table and Known Limitations say what a 200 with
      an error body means.
- [ ] `define` degrades quietly on an overload — no raw JSON at the user.

## Plan

- [ ] Design via `sdlc start-plan` before implementing — the fix touches
      `internal/llm`'s taxonomy, which has its own fake and conformance suite.
- [ ] Decide the retry question first: a 200-with-error-body is transient, so
      whether the transport RETRIES it or merely absorbs it changes the latency
      story for every consumer. It is a decision, not an implementation detail.
- [ ] Read the error body's `type` in `classifyStatus`'s neighbourhood and route
      the transient kinds to `ErrUnavailable`, with the enumeration of body types
      written down rather than sampled.
- [ ] Teach `llmtest` to script a 200 carrying an error body, and cover it in the
      obligation suite against both the fake and the live service.
- [ ] Add a capture-drift row: "the proxy answers 200 for an overload" is a claim
      about the real service that can change.
- [ ] Sweep `atlas/llm.md`'s taxonomy table and Known Limitations.

## Log

### 2026-08-23

Filed from #16 M2's hand verification rather than fixed there: the defect is in
`internal/llm`'s taxonomy, which has its own fake and conformance obligations,
and folding it into #16's close would have shipped a change to #11's contract
without its own review boundary.

Intermittent by nature — the same three-line session succeeded minutes earlier
and produced a visibly adapted answer. That is what makes it worth a test rather
than a retry-and-hope.

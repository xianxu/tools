---
id: 000019
status: open
created: 2026-08-23
updated: 2026-08-23
estimate_hours:
github_issue:
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

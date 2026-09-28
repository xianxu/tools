---
id: '000036'
status: done
started: 2026-09-13T23:05:26-07:00
created: 2026-08-30
updated: 2026-09-14
estimate_hours: 2.57
actual_hours: 1.42
---

# a question has no progress indicator, and it is the one slow path

## Problem

Operator, 2026-08-30:

> when we query LLM, we should display spinner as it is a much slower path
> compared to other `define` operations.

Right, and the asymmetry is the point. A lookup is instant and offline — that is
the tool's whole promise. A question is seconds, sometimes tens of seconds, and
until the first token arrives the screen shows **nothing at all**: no line, no
mark, no cursor movement. A user cannot tell "thinking" from "hung", and the two
call for opposite responses.

**The seam already exists and has no consumer.** `llm.Config` carries
`OnSlow func(Progress)` and `SlowEvery`, built for exactly this:

```go
// OnSlow, when set, is called on a ticker while a call is still running, with
// the phase it is in. Optional, off by default, never called on a fast path —
// this is for answering "slow where", not for logging.
```

`Progress` carries `{Task, Phase, Elapsed}` where Phase is `"waiting"` (no
response yet) or `"streaming"` (mid-body). Grepped 2026-08-30: no non-test caller
sets `OnSlow`. So this issue is mostly WIRING plus a UI decision, not new
transport work.

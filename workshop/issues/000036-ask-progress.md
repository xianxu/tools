---
id: 000036
status: open
deps: []
github_issue:
created: 2026-08-30
updated: 2026-08-30
estimate_hours:
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

## Spec

Not designed. What follows is what a design has to answer.

### The indicator is EPHEMERAL, and this repo already has that doctrine

`♫ playing 3×` is the precedent: drawn while playback blocks, erased after, and
deliberately absent from a pipe — because `define x > out.txt` is a RECORD and a
record must be true. `defaultIndicator` (`main.go`) is where that split lives,
and a spinner is the same shape. Reuse the decision; do not re-litigate it.

### Where it draws differs by loop, and #30 changed one of them

- **The raw editor** owns a screen (`#30`). A spinner is part of the LIVE EDGE,
  like the prompt — not a buffer line, or every tick would append to the
  transcript. `liveScreen.Draw` already takes the live edge, so the question is
  whether the spinner is a third argument or rides the prompt string.
- **The piped/one-shot path** writes to a real stdout with no frame. Same rule as
  the indicator: nothing, or a record-shaped line.

### Open questions a design must settle

1. **When does it appear?** `SlowEvery` defaults to 10s, which is a "why is this
   slow" ticker, not a spinner cadence. A spinner wants ~100ms. Either
   `SlowEvery` is set small for this consumer, or the spinner is driven locally
   and `OnSlow` is used only to say WHAT phase it is in. The second is probably
   right: the transport should not be asked to drive a UI clock.
2. **What happens when streaming starts?** The answer streams token by token, so
   the spinner has done its job the moment the first delta lands. Phase already
   distinguishes this. Does it vanish, or become something else while the answer
   is still arriving?
3. **Does it interact with the repaint throttle?** `#30`'s screen paints at most
   every 16ms with a trailing flush. A spinner ticking faster than that is
   invisible work; slower and it stutters. Pick a cadence that respects the
   budget already declared in `atlas/define.md`.
4. **Ctrl-C during a question is already scoped** (`#16` D5) — it cancels the
   answer, not the session. Whatever is drawn must be taken back cleanly on that
   path, which is the `eraseLine` gesture the screen already honours.
5. **`--play` and `reflect`** also reach the model. Do they get it too, or is
   this the ask path only? One mechanism with several consumers is this repo's
   preference (`#30` is filed that way).

## Done when

- [ ] A question shows movement within a second of being asked, so "thinking" is
      distinguishable from "hung".
- [ ] The indicator is EPHEMERAL on a terminal and ABSENT from a pipe, `-raw`
      and `> out.txt` — the same rule the playback indicator follows, reached
      through the same decision rather than a second one.
- [ ] It is gone the moment the answer starts arriving, and taken back cleanly
      when Ctrl-C cancels the answer.
- [ ] It never appears on the FAST path: a lookup is instant and must stay
      visually silent.
- [ ] `OnSlow` has a production consumer, or the design says why it is still the
      wrong seam for this.

## Plan

- [ ] Brainstorm the five questions above, then design if it is more than wiring.

## Log

### 2026-08-30

Filed while diagnosing an unrelated key problem: with no model configured, a
question fails instantly and loudly, so the SLOWNESS of the working path had not
been felt in a while. Once it worked, the wait was the first thing noticed.

Measured: `OnSlow` has no non-test consumer, and `Progress.Phase` already
distinguishes waiting from streaming — so the transport half of this exists.

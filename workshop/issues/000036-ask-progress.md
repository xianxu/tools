---
id: 000036
status: working
deps: []
github_issue:
created: 2026-08-30
updated: 2026-09-13
estimate_hours: 2.57
started: 2026-09-13T23:05:26-07:00
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

The original questions below are retained as design history. The proposed
2026-09-13 design following them is the current specification.

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

### Proposed design — 2026-09-13

Use a common UI activity component in `cmd/define/activity.go`, with a pure
Braille frame selector and a small lifecycle runner. It is reusable for any
waiting operation through a display host, not coupled to an LLM SDK
or one command. All consumers currently belong to define, so no speculative
cross-binary package is introduced (AGENTS.local.md; ARCH-DRY).

The standard cycle is `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`, advancing every 80 ms. Show the first
frame immediately, with no label or other text. Start only at an actual model call, before
local model discovery. Complete calls remain active until return; Stream calls
stop before forwarding the first nonempty answer-text delta. Empty deltas and
thinking/tool events do not finish the wait. Error, timeout, cancellation, and
an empty response all clear the spinner. A stopped activity cannot repaint.
First answer text is the proposed streaming default; the user was offered the
alternative of keeping it visible through completion.

A display host owns placement and terminal writes. In the raw REPL use a live
screen overlay outside the transcript; ticks repaint through the existing screen
lock/throttle and preserve prompt, footer, scroll position and partial output.
In a one-shot terminal use one transient line and synchronously clear it before
answer/error output. Serialize the animation's paints and teardown through that
host. Use the existing opt.tty capability decision with an explicit !opt.raw guard:
no animation or escape bytes
in pipes, redirected output, or other non-erasable output. No spinner on ordinary
dictionary lookups or fully cached/no-work operations.

Reuse the same component through a define-local LLM client decorator for ask,
reflect, harvest (including agreement mode), and --llm-check. The decorator starts
an activity around each Complete/Stream and joins its ticker worker before
returning or forwarding first text. Keep the original client available for
llm.SelectionOf in diagnostics/reflection: a wrapper must not regress #58's
selected-model provenance. No model, retry, timeout, or error taxonomy changes.
The transport's OnSlow remains a diagnostic callback rather than an animation
clock: it starts after discovery, delays ten seconds by default, and its phase
names do not identify the first user-visible text token.

The current play/review implementation makes no LLM calls, so there is no
invented spinner there. The component's display-host API also supports its screen
when future model-backed work is introduced. Issue #54 plans background harvest
and reflection: it must reuse the component in an appropriate status area and
preserve the foreground prompt. This issue owns shared activity display and
existing synchronous model-call adapters, not #54's background job lifecycle.
Sequence this issue before #54's activity integration and preserve the selected
model metadata added in #58. Decorate foreground clients at their UI entry
points, not the global d.newLLM factory; #54 background cores should accept
undecorated clients so background work cannot take over a foreground spinner.

One alternative is using OnSlow as the animation clock; it misses discovery and
couples UI cadence to transport diagnostics. Another is separate command-specific
spinners; it duplicates cleanup/output ownership. The reusable activity component
with screen and plain-terminal hosts satisfies all current call sites with one
lifecycle (ARCH-PURE/DRY/PURPOSE).

Tests name the risky functions and properties: activityFrame has periodicity
and Unicode display-width oracles; activity lifecycle uses injected ticks and
barriers to prove stop-before-output, cancellation, idempotent cleanup and no
post-stop writes; display hosts use a stateful terminal model to assert cursor,
transcript and footer preservation under resize and output failure. LLM adapters
use the existing stateful llmtest fake to hold discovery/response frames, then
release them and verify every real consumer. Run race tests and a focused PTY
conformance check with all unavailable dependencies routed through the existing
conformance.SkipOrFail policy (ARCH-MOCK/SECURE).

ARCH-CONSTRAINTS: at most 12.5 repaint requests/sec per visible activity, within
the screen's existing throttle; one bounded ticker worker per operation, joined
on stop; no extra network calls or durable state. ARCH-ORDER: idle -> waiting ->
stopped; first text/return/cancel/write-failure all stop, later ticks are ignored,
and concurrent activities on a host must have distinct ownership so a stale stop
cannot clear a newer activity. Host replacement/overlap policy must be made
explicit in the implementation plan. ARCH-FUNERAL: stopping releases timer,
worker and overlay; no spinner frames remain in history or disk.

## Done when

- [x] A question shows movement within a second of being asked, so "thinking" is
      distinguishable from "hung".
- [x] The indicator is ephemeral on a terminal and absent from piped/redirected
      output, `-raw` and `-no-color`, using existing terminal capability.
- [x] It is gone the moment the answer starts arriving, and taken back cleanly
      when Ctrl-C cancels the answer.
- [x] It never appears on the FAST path: a lookup is instant and must stay
      visually silent.
- [x] `OnSlow` has a production consumer, or the design says why it is still the
      wrong seam for this.

## Plan

- [x] Resolve presentation and lifecycle; review the durable plan in
      `workshop/plans/000036-ask-progress-plan.md`.
- [x] Obtain plan approval, implement the shared spinner and foreground adapters,
      verify terminal lifecycle and consumer paths, then close through SDLC.

## Log

### 2026-08-30

Filed while diagnosing an unrelated key problem: with no model configured, a
question fails instantly and loudly, so the SLOWNESS of the working path had not
been felt in a while. Once it worked, the wait was the first thing noticed.

Measured: `OnSlow` has no non-test consumer, and `Progress.Phase` already
distinguishes waiting from streaming — so the transport half of this exists.


## Revisions

- 2026-09-13: User requests a reusable Braille spinner across LLM waiting states.
  Claimed #36 and ran start-plan. Expanded the original ask-only sketch to all
  current actual LLM consumers. Confirmed current play/review makes no model calls.
  Corrected prior assumptions: defaultIndicator keeps playback records in pipes;
  spinner suppression should reuse terminal capability, not playback record text.
  OnSlow does not cover discovery or first visible text. No implementation yet.

- 2026-09-13: Confirmed shared-component placement in cmd/define because the
  current consumers are all one binary. Foreground adapters retain original
  clients for SelectionOf metadata and do not decorate the global client factory,
  preserving a clean boundary for #54's future background work. Optional spinner
  lifetime preference has received no change request; first visible text remains
  the proposed default.

- 2026-09-13: Fresh-context spec review approved implementation planning. Carry
  forward: explicit opt.tty && !opt.raw eligibility; separate --llm-check wiring;
  activity overlay independent of Draw(prompt, footer); visible host write-failure
  detection despite current screen Paint discarding errors; releasable fake
  completion/stream barriers; ownership on stale stop, suspension and shutdown.
  No product code changed; design approval is the next checkpoint.

- 2026-09-13: User approved the pattern and corrected presentation: glyph only,
  no Thinking label; stop when the response arrives. For streaming this means
  clearing before the first nonempty answer text; for Complete, on return.
  Proceeding to a durable implementation plan with that correction.

- 2026-09-13: Implementation plan authored with named component/host/client
  seams and function-level test strategies. Plan review refined raw-screen
  placement to a separate transient glyph row, preserving RenderLine cursor
  controls and shared paint/hit-test geometry. On a screen with no spare row,
  preserve the user's prompt and omit the glyph until space becomes available.

- 2026-09-13: Fresh-context plan review approved after separating activity from
  prompt text in the shared screen painter. Existing displayRows does not count
  embedded newlines; explicitly budget the activity row and test exact footer and
  cursor geometry. Durable plan is ready for operator approval. Product code is
  unchanged; next action after approval is sdlc change-code --issue 36.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only; calibration is marked stale by estimate-source,
so the values remain provisional. Derived after plan-quality accepted the design.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.2 impl=0.32
item: tui-screen design=0.25 impl=0.4
item: smaller-go-module design=0.03 impl=0.2
item: cross-cutting-refactor design=0.12 impl=0.28
item: smaller-go-module design=0.03 impl=0.2
item: atlas-docs design=0.04 impl=0.12
item: milestone-review design=0 impl=0.28
design-buffer: 0.15
total: 2.57
```

- Common lifecycle: design 1.0 × 0.2 for the resolved ownership contract;
  implementation 0.8 × 0.4. Library check: standard context/timers and existing
  renderer suffice; no external spinner package removes our lease ownership work.
- Display hosts: design 1.25 × 0.2; implementation 1.0 × 0.4, with existing
  terminal oracle and screen painter reused.
- Foreground decorator and fake barriers: each design 0.15 × 0.2 and
  implementation 0.5 × 0.4; both extend established seams.
- Consumer wiring: design 0.6 × 0.2, implementation 0.7 × 0.4 across all paths.
- Docs: two surfaces at design 0.02 and implementation 0.15 × 0.4 each.
- Sole close boundary: allow two review rounds at 0.35 × 0.4 each.
- Thorough-plan buffer adds 15% to design subtotal 0.67; implementation totals
  1.80, familiarity 1.0. Total 2.5705 rounded to 2.57 hours.


### 2026-09-14 implementation progress

User approved implementation. Plan-quality passed with advisory PQ-1 (compressed
in the durable plan); estimate-quality passed with informational scope notes.
Verification time is included in each implementation primitive; PTY uses the
existing harness. The docs allowance groups the two atlas files as one map update
plus README. Consumer wiring's 0.7 unscaled hours intentionally exceeds the usual
0.5 maximum to cover five call routes and provenance. Residual design covers exact
interfaces and terminal edge behavior; full-width RenderLine and short-write tests
found concrete cases during implementation. These are estimate clarifications,
not a retroactive change to the 2.57-hour estimate.

Common runner/plain/screen hosts and all foreground consumers implemented.
Test-first failures observed for missing activity API and missing consumer output;
controlled wire barriers preserve captures. Focused component race and frame fuzz
passed; strict real-PTY response/cancel checks passed three runs and a no-spinner
build overlay failed as expected. Broad suite and final committed checks pending.
ARCH-ORDER: stop joins animation and clears before forwarding response text.
ARCH-DRY: shared short-write recorder and painter keep display/error accounting in
one place; the original clients retain selected-model metadata.

- 2026-09-14: Updated the original Done-when wording to the approved suppression
  rule: playback can leave non-terminal records, whereas activity emits nothing.
  Initial full-suite run caught an overly broad test count (the capture mentions
  Obsequious twice per answer); corrected the independent answer-title count.
  Focused paths and race checks passed after the correction; no product fix was
  needed for that test assertion.


### Verification — 2026-09-14

On implementation commit 4856c97: `go test ./... -count=1` passed (define
108.965s); focused activity/consumer/fake-barrier race tests passed; `go vet ./...`
passed; shared `internal/conformance` checks passed. Strict
`CONFORMANCE_STRICT=1 go test -tags conformance ./cmd/define -run '^TestPTYActivityWaitingAndCleanup$' -count=1`
passed (4.260s). The PTY test was also repeated three times and a no-spinner
build overlay correctly failed. Pure frame fuzz passed 23,410 executions.
`git diff --check` passed. All implementation and verification tasks delivered;
submitting the sole close boundary review next.

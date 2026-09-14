# Shared Braille activity spinner implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Show only an animated Braille glyph while define waits for an LLM response, using one reusable activity component across current consumers.

**Architecture:** A pure frame selector and cancellable activity runner drive display leases owned by the plain terminal or live screen. A define-local foreground client decorator covers discovery and Complete/Stream calls, stopping synchronously before response output. Transport behavior and selected-model provenance remain unchanged.

**Tech Stack:** Go, existing liveScreen renderer, context/timer channels, stateful llmtest HTTP fake and terminal tests.

**Spec:** `workshop/issues/000036-ask-progress.md`, proposed 2026-09-13 design plus user's glyph-only correction. One atomic close boundary; tasks are not milestones.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| `activityFrame` | `cmd/define/activity.go` | new |
| `activityEnabled` | `cmd/define/activity.go` | new |
| `activityRow` | `cmd/define/activity_screen.go` | new |

`activityFrame` selects from `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` at 80 ms intervals. It produces
one display cell and no label, elapsed time, status text or trailing newline.
`activityEnabled` uses `opt.tty && !opt.raw`, reusing the existing terminal
capability fact. `activityRow(prompt, glyph string, rows, cols int)` grants a separate
transient glyph row when space permits, without modifying the prompt string,
stored footer or transcript content. See exact placement rules below.
These policies are directly tested without IO. No transport diagnostic phase
is used as a proxy for first answer text.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| `startActivity` | `cmd/define/activity.go` | new | injected ticks and teardown |
| `activityLease` | `cmd/define/activity.go` | new | cancellation and display ownership |
| `plainActivityHost` | `cmd/define/activity_terminal.go` | new | transient line |
| `screenActivityHost` | `cmd/define/activity_screen.go` | new | screen-owned lease |
| `screen.Paint` | `cmd/define/screen.go` | modified | delegates to shared painter with empty activity |
| `screen.paintActivity` | `cmd/define/screen.go` | new | explicit activity row and shared frame geometry |
| `liveScreen` | `cmd/define/screen.go` | modified | overlay, paint error, suspend/stop |
| `activityClient` | `cmd/define/llm_activity.go` | new | Complete/Stream decoration |
| `runAsk` | `cmd/define/ask.go` | modified | foreground activity |
| `runReflect` | `cmd/define/reflect.go` | modified | foreground activity and provenance |
| `runHarvest` | `cmd/define/harvest.go` | modified | foreground activity |
| `runLLMCheck` | `cmd/define/llmcheck.go` | modified | explicit eligibility and activity |
| `run` | `cmd/define/main.go` | modified | diagnostic eligibility wiring |
| `Reply` | `internal/llm/llmtest/fake.go` | modified | releasable response barriers |

The activity API is operation-independent: begin an activity on a display host,
animate its glyph, then stop. The LLM decorator is one consumer, so future waiting
operations can reuse it without depending on an SDK or request type. All current
users are in one binary; keep the component in cmd/define rather than introducing
a speculative cross-binary library (AGENTS.local.md, ARCH-DRY).

## Behavior, ownership, and bounds

- Begin synchronously, before invoking the underlying client: this includes #58's
  lazy model discovery. Complete ends the activity before returning. Stream ends
  it before forwarding the first nonempty text delta, then forwards all deltas
  unchanged. Empty deltas do not stop it. Defer cleanup for every return path.
- No response label: only the glyph. No timer, worker or writes when disabled.
  No UI for dictionary/cache-only paths because only actual model calls start it.
- One ticker worker per visible activity, with injected ticks in tests. Stop is
  idempotent and closes the worker's stop channel, joins it, then releases/clears
  the owned display. Cancellation, supersession and write failure stop animation;
  no worker survives Stop. Stop must never be called from the ticker worker in
  a way that waits on itself: worker exit and caller join are separate actions.
- Host Begin returns a lease with a unique identity and a superseded channel.
  A new lease invalidates the old one (last begin wins). Set/Clear act only for
  the current identity under the host mutex. Thus a late tick or stop from an
  earlier activity cannot overwrite/erase a newer one. Never join workers while
  holding a host/screen lock.
- Plain terminal: own the current empty output line; first paint is a glyph,
  later paints use carriage-return replacement, and final clear removes that
  owned line without appending transcript records. Establish the line-boundary
  precondition at every current call site; do not erase arbitrary partial prose.
  The host serializes its paints/clear. Existing consumers forward first text or
  diagnostics only after the decorator has synchronously stopped.
- Screen: store activity independently of prompt/footer in liveScreen. Repaint
  composes the glyph into the live edge; activity never calls Write or replaces
  Draw's chrome. Begin and clear force a frame so the first glyph is immediate
  and cleared before answer text; ticks use the existing 16 ms throttle. Resize
  and scrolling repaint the current activity with current dimensions.
  Placement is explicit: pass the glyph separately to screen.paintActivity,
  never concatenate multiline content into the prompt. Existing screen.Paint
  delegates to the same painter with empty activity, preserving its signature.
  Clip/budget the original prompt first. If it consumes all terminal rows, omit
  activity; otherwise grant one row before the prompt and subtract that row from
  remaining footer/gap/buffer budgets. With no prompt, the glyph is the sole live
  edge row. Render glyph then CRLF only when another live-edge row follows.
  Record footerTop with the granted activity row included. Restore the cursor
  relative to the original prompt rows only, reprinting only the prompt, never
  the glyph. Thus RenderLine's erase/cursor controls remain on the editor row.
  Existing displayRows remains a single-line helper: do not feed it CRLF content.
  The same painter owns geometry for RegionAtRow/FooterRowAt, including pinned
  screens. Recompute on resize. Test activityRow and screen.paintActivity with generated dimensions and prompt
  widths against independent row-allocation, cursor and hit-test oracles.
- Suspend invalidates/clears the current lease before yielding terminal ownership;
  resume must not resurrect it. Stop invalidates the lease and stops any pending
  activity paint. A stale timer cannot repaint after screen stop or suspension.
- Screen Paint currently discards writer errors. Wrap the tty writer in repaint
  with a small first-error recorder (reuse an existing suitable writer if found),
  store the resulting error under the screen mutex, and make activity Set return
  it. Do not change Paint's public signature or unrelated output policy. A host
  error stops animation and permits the normal model result/error path to proceed;
  do not turn an animation failure into an LLM transport error or retry.
- Bound requested animation to 12.5 Hz and one cell. No extra network calls,
  durable files, cached activity history or global background worker. No new
  dependencies are needed: Go timers and existing screen/terminal primitives
  already provide the required mechanisms.

### State/event contract

| State/event | Result |
|---|---|
| Disabled / begin | no-op stop, no worker or IO |
| Idle / begin | acquire lease, paint first frame, start ticker |
| Waiting / tick | update current lease; exit worker if revoked or write failed |
| Waiting / first nonempty delta | stop, join, clear; then forward delta |
| Waiting / Complete return, error, cancel | stop, join, clear; preserve original result |
| Waiting / replacement or host suspend/Stop | revoke lease; worker exits; stale clear ignored |
| Stopped / tick, repeated stop, late cancellation | no writes or state resurrection |

ARCH-ORDER is tested with controlled event order, including cancellation racing
first text and timer racing clear. ARCH-SECURE: glyphs are fixed program data;
no provider text or keys are rendered as spinner state. ARCH-FUNERAL: leases,
timer and worker end with the activity; no animation remains in scrollback.

## Chunk 1: Component, hosts and foreground integration

### Task 1 — Common animation lifecycle

**Files:** create `cmd/define/activity.go`, `cmd/define/activity_test.go`.

- [x] Write failing tests for activityFrame/activityEnabled with periodicity,
  display-width and eligibility oracles; test the runner against injected tick,
  cancel and replacement orderings with write logs and worker-exit barriers.
- [x] Run `go test ./cmd/define -run '^TestActivity' -count=1` and observe red.
- [x] Implement the pure frame selector and lease-based runner under the ownership
  contract above. Keep worker exit separate from caller join and make stop safe
  to call repeatedly, including after an output failure.
- [x] Rerun focused tests to green; commit the tested component with #36 and the
  authoring-model trailer. Run `go test ./cmd/define -count=1` on each resulting
  committed window to exercise declaration/status and other repository guards.

### Task 2 — Terminal and screen hosts

**Files:** create `cmd/define/activity_terminal.go`, `activity_screen.go`,
`activity_terminal_test.go`, `activity_screen_test.go`; modify `cmd/define/screen.go`.

- [x] Test host begin/set/clear and activityRow with a stateful terminal
  oracle: adversarial writes, resize, suspension and replacement must preserve
  prose, cursor/chrome ownership and final transcript. Include painted geometry
  and RegionAtRow/FooterRowAt oracles for activityRow rather than
  checking only transcript bytes. Plain-host callers must prove line boundaries
  for one-shot and scanner REPL, including redirected stdin with terminal stdout.
  Use deterministic ticks
  and failure-injecting writers, not sleeping screenshot assertions.
- [x] Run `go test ./cmd/define -run 'TestActivityHost|TestActivityScreen' -count=1`
  and observe red. Implement independent screen overlay and plain transient-line
  ownership, with first-error recording at the screen paint boundary.
- [x] Rerun to green, then `go test -race ./cmd/define -run 'TestActivity|TestLiveScreen' -count=1`.
  Commit the tested hosts; run `go test ./cmd/define -count=1` on the commit.

### Task 3 — Shared LLM adapter and real consumer wiring

**Files:** create `cmd/define/llm_activity.go`, `llm_activity_test.go`,
`activity_paths_test.go`; modify `ask.go`, `reflect.go`, `harvest.go`,
`llmcheck.go`, `llmcheck_test.go`, `main.go`, and `internal/llm/llmtest/fake.go`
plus its colocated tests for the narrow hold/release extension.

- [x] Test activityClient.Complete/Stream against held discovery and response
  boundaries through llmtest; extend replies with releasable barriers where the
  fake currently only supports indefinite stalls. Verify the fake honors the
  same stream/text distinctions as the consumed protocol (ARCH-MOCK).
- [x] Test each foreground consumer against no-work/disabled/streaming outcomes
  using real host and fake model logs; model provenance and plain output are
  independent oracles. Run focused `TestLLMActivity|TestActivityPaths` tests red.
- [x] Implement the decorator. Retain the original client for SelectionOf in
  reflect/llmcheck; pass the decorated client to llm.Run/Complete. Capture the
  activity host from original stdout before ask adds wrapping/highlighting.
  In harvest decorate the client once so agreement and every typed subtask are
  covered. --llm-check has its own main dispatch bypassing deps.newLLM: pass
  terminal eligibility explicitly through its existing function seam and update
  callers/tests. Select the live-screen host when stdout is a liveScreen.
- [x] Do not decorate deps.newLLM globally. #54's background cores must continue
  accepting undecorated clients; foreground UI owns its spinner. Current play
  makes no model calls. Its pinned screen remains a supported host, with no
  fabricated work or spinner added to review.
- [x] Rerun focused tests and race tests to green; commit the integrated paths.

### Task 4 — Verification, documentation and closure

**Files:** update `cmd/define/README.md`, `atlas/define.md`, `atlas/llm.md`;
extend `cmd/define/pty_conformance_test.go` or add
`cmd/define/activity_conformance_test.go` if isolation is clearer.

- [x] Add a focused PTY check for animated waiting followed by response/cancel
  cleanup. Route absent external dependencies through conformance.SkipOrFail;
  a missing in-repo artifact is a failure. The live-model transport remains
  replaced by the stateful fake so verification needs no paid inference.
- [x] Run `go test ./... -count=1`, focused activity race tests, the new PTY
  conformance check, `go test ./internal/conformance -count=1`, and
  `git diff --check`. Record actual command outcomes, including any missing PTY.
- [x] Update docs with the glyph-only wait, first-text/end-of-call cleanup,
  all actual consumers and pipe/raw suppression; keep atlas test inventory current.
  Record evidence and tick delivered issue/plan tasks.

After deliverables pass, use `sdlc close --issue 36 --verified '<measured evidence>'`
for the sole boundary review, resolve findings and update lessons, then publish
through `sdlc pr` and `sdlc merge`. Do not add a duplicate ad-hoc code review.

## Coordination and non-goals

Issue #54 is working and plans background harvest/reflection and typed cores.
#36 owns common activity display and current foreground adapters first; #54 must
preserve those adapters and #58 provenance when extracting cores. #36 does not
depend on #54 and does not implement its scheduling, notices, or background UI.
Recheck main before merge for overlapping core changes. Scope excludes model
selection policy, retry behavior, diagnostic OnSlow semantics, playback-indicator
redesign, and claiming that a spinner makes the model faster.

## Approval

User approved implementation on 2026-09-14 after correcting presentation to
pattern only, no Thinking label. The plan and estimate gates passed; work is on
branch 000036-ask-progress.


## Revisions

- 2026-09-13: Plan review identified RenderLine erase/cursor controls as a
  placement constraint. Specified a separate glyph row with prompt-first space
  budgeting and one shared screen.Paint geometry for painting and hit-testing.
  Quoted actual declaration names/paths in concept tables and added full-package
  verification on committed windows so repository guards cannot be bypassed by
  focused tests. These refinements preserve the requested glyph-only behavior.

- 2026-09-13: Second review caught that displayRows intentionally ignores embedded
  newlines. Replaced composed multiline prompt with a separately budgeted activity
  row in one shared painter; existing Paint delegates without activity. Required
  exact two-row and footer/cursor assertions to prevent geometry drift.

- 2026-09-13: Final bounded fresh-context review approved the revised plan with
  no remaining blockers. Implementation awaits user plan approval.

- 2026-09-14: Implementation approved; change-code passed both gates. Components
  built in parallel against a shared lease API and will commit as one integrated
  change so all entity declarations and consumer wiring are present together.
  PQ-1 addressed by compressing placement tests to function-level oracle strategy.
  Tests discovered redundant RenderLine leading CR counted as a display cell;
  shared painter removes that redundant control before measuring. Exact-width
  regression covers Paint with and without activity. Shared writer records short
  writes as io.ErrShortWrite for both hosts; no transport errors are altered.

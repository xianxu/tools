---
id: 000059
status: working
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours: 4.56
started: 2026-09-14T01:07:15-07:00
---

# define: mouse text selection and clipboard copy

## Problem

User: “define should support mouse selection of text, and copy the selection to
clipboard.” The full-screen editor and practice screens capture the mouse for
click actions and scrolling, so ordinary dragging no longer selects text. Today
help documents a terminal modifier override. The requested feature makes selection
part of define itself while preserving its existing mouse actions.

## Spec

### Proposed interaction

- Left-button drag highlights a linear text range across displayed rows. Releasing
  a nonempty drag copies the selected text to the macOS clipboard. The user
  approved this copy-on-release design on 2026-09-14. A click with no drag keeps its current action, once, on release.
- Covers displayed definitions, answers, and review screens, including visible
  prompt/footer text. Selection is text-only: ANSI styling, click markers added
  only as decoration, spinner glyphs and layout padding are not clipboard content.
  Actual visible punctuation and text remain. Keep internal spaces, omit trailing
  row padding, join selected physical rows with newlines. Copy complete Unicode
  characters, including combining marks and both cells of a wide character.
- Dragging across clickable text or a review choice must never play audio, look up
  a word, or record a grade. A drag remains a drag even if the pointer returns to
  its starting cell before release. An empty selection does not change clipboard.
- The highlight remains until the next gesture, key input or content/layout
  change. Resize, scroll, screen switch, cancellation and exit cancel an unfinished
  gesture without copying. A changed selectable frame invalidates its gesture;
  cosmetic spinner ticks alone do not. No stale event may act on a new screen.
- Selection must not wait behind an LLM request or playback. Mouse gesture handling
  belongs before the loop's type-ahead queue, using the active screen's protected
  displayed-frame snapshot. Ordinary click actions still run on the owning loop,
  tagged with their screen/frame identity and discarded if stale. Keyboard and
  interrupt behavior retain their existing ownership.
- Clipboard work runs outside the screen lock and input decoder, with bounded
  concurrency/lifetime and explicit error reporting on the owning screen. A
  copy failure must not be reported as success or lose the selected text silently.
  No success banner is needed for normal copying. Use a clipboard seam with a
  native macOS implementation that writes literal Unicode text; keep portable
  builds valid with an unsupported-platform implementation.

### Architecture and alternatives

Use one pure pointer-gesture/range policy and one shared screen selection overlay
for both editor and practice. The existing shared painter owns the displayed-cell
snapshot used for highlight and extraction; selection must not reconstruct the
layout independently or mutate transcript text (ARCH-DRY/PURE).

Enable button-motion reporting and decode press, held-button motion and release
for SGR and legacy mouse encodings. Defer the current click actions until the
shared gesture policy identifies a click. Xterm's button-event mode reports motion
only while a button is held; no idle hover stream is necessary. Source:
https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h2-Button-event-tracking

Alternatives considered: keeping the terminal's modifier-selection workaround
requires the extra gesture the request seeks to remove; disabling mouse reporting
would lose existing click/scroll behavior. Application-owned selection preserves
both. Native clipboard writing supplies a detectable result on the local Mac;
OSC 52/remote terminal clipboard forwarding is outside this first scope.

The native writer must force plain text. Local `man pbcopy` confirms pbcopy infers
RTF/EPS from leading bytes, so blindly piping selected text to it is not sufficient
for the literal-text contract. Implementation planning will select the native
pasteboard binding and isolated conformance seam.

### State and bounds

Gesture states are idle, pressed, dragging, selected, and cancelled-awaiting-release.
Press starts on the active frame; movement makes a drag sticky; release emits
exactly one click or copy effect. Invalidating events cannot convert a cancelled
drag into a click. Entering/leaving nested /play switches the same input router's
active screen atomically (ARCH-ORDER).

Only the visible frame is selectable in this scope; no auto-scroll, rectangular
selection, word/line multi-click gestures or new keyboard-copy shortcuts. The
frame/range snapshot is bounded by terminal dimensions and capped before allocation;
clipboard payload is bounded explicitly in the plan. Clipboard workers end with
the console and cannot publish stale completion UI. Selection creates no durable
files/history; clipboard contents persist by the user's copy action (ARCH-CONSTRAINTS,
ARCH-FUNERAL). Tests use an isolated pasteboard/fake and never overwrite the user's
clipboard (ARCH-MOCK/SECURE).

### Verification strategy

- Mouse decoder: fuzz partial/malformed SGR and legacy reports; assert no bytes
  leak into typed text and every enabled mode is decoded/restored.
- Gesture policy: generated event sequences check exclusive click/copy effects
  and invalidation across frame identity, screen switches and interruption.
- Text extraction/highlight: Unicode/ANSI/display-width inputs checked through
  the terminal oracle against exact clipboard text and unchanged transcript.
- Shared routing: stateful editor and practice tests prove drag cannot grade/play,
  click still acts once, and busy streams/playback cannot replay stale gestures.
- Clipboard seam: stateful fake plus isolated native conformance for literal text,
  failures, bounded operations and shutdown; no real user pasteboard mutation.
- Real PTY: press/drag/release, highlight, copy seam, resizing and mode restoration.

## Done when

- [x] Drag selects text visibly and copies the exact selected text on release.
- [x] Editor, answers and review screens share selection behavior; dragging over
      actions never activates them, while ordinary clicks still work once.
- [x] Unicode, multiline/wrapped content and clipped rows copy without styles,
      spinner glyphs or layout padding.
- [x] Streaming/playback, resize, scrolling, screen switching and exit cannot
      turn stale gestures into a copy or review action.
- [x] Clipboard errors are visible, tests isolate the user's clipboard, and
      portable builds retain a valid unsupported-platform seam.
- [x] Help/README and atlas reflect app-owned selection; tests and terminal
      conformance demonstrate the behavior.

## Plan

- [x] Review the interaction/spec, incorporate the copy-trigger preference.
- [x] Write and review the durable implementation plan, obtain approval, then
      enter implementation with sdlc change-code.
- [x] Implement and verify shared selection, clipboard integration and consumers;
      close through the sole SDLC boundary and publish via PR.

## Log

### 2026-09-14

Created and claimed #59; ran start-plan. Read mouse mode/decoder/consumer seams.
Existing KeyClick acts on press in both editor and practice, including stored review
marks; gesture arbitration must precede both. Existing readKeys keeps decoding
while loops block on model/audio, so pointer handling must live before its buffered
key delivery. paintActivity owns actual layout including #36's spinner; use its
frame geometry. Native clipboard text-type behavior verified via local pbcopy
manual; terminal motion/release encoding verified against primary xterm docs.
No product code changed. Optional copy-on-release preference question pending;
proposed default is copy on release.


## Revisions

- 2026-09-14: Fresh-context spec review approved for planning, with the copy trigger
  still provisional. Carry three explicit obligations into the implementation plan:
  resize invalidates pointer geometry at the watcher, not only when a busy loop
  finally consumes the notification; keyboard type-ahead saturation must not block
  pointer/interrupt dispatch; clipboard writes are serialized in gesture order and
  failure notification must preserve access to the failed selection rather than
  invalidating it by appending content. No product code changed.
- 2026-09-14: Confirmed a native clipboard seam in the installed macOS SDK:
  PasteboardCreate supports a unique isolated pasteboard, and PasteboardPutItemFlavor
  accepts explicitly typed byte data. These APIs are not thread-safe and therefore
  need one serialized owner. The existing binary already links macOS frameworks;
  this avoids a shell command that infers RTF/EPS from selected text. Primary API
  reference: https://developer.apple.com/documentation/applicationservices/applicationservices_functions
  Final binding, lifecycle bounds and isolated conformance wiring belong in the
  durable implementation plan after spec approval.

- 2026-09-14: User approved the written spec, including copy on release. Authored
  workshop/plans/000059-mouse-selection-copy-plan.md via writing-plans skill.
  Clipboard lifetime uses a cancellable private helper in the existing executable
  because native calls cannot be interrupted in-process. Failure retains a bounded
  payload with a click-to-retry notice; type-ahead overflow explicitly rejects
  newest input with feedback so pointer/interrupt decoding remains live. Click
  validation and target capture share one lock to avoid action after layout drift.
  Plan review and artifact guards are running. No product code changed.

- 2026-09-14: Durable plan review approved after resolving atomic target capture
  and isolated foreground PTY clipboard injection. Conformance-only target factory
  fails closed, while normal builds always use the canonical clipboard.
  `go test ./cmd/define -run 'TestPlan|TestNoArtifact|TestARemoved' -count=1`
  passed (1.252s), and git diff --check passed. Awaiting durable-plan approval;
  implementation has not started.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=1.0 impl=0.08
item: greenfield-go-module design=0.2 impl=0.32
item: tui-screen design=0.3 impl=0.8
item: api-integration design=0.3 impl=0.6
item: cross-cutting-refactor design=0.12 impl=0.2
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.02 impl=0.2
design-buffer: 0.15
total: 4.56
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only; calibration is marked stale by estimate-source.
Issue/spec authoring counts the already completed design at the v2 midpoint,
without discount. The remaining approved-plan design uses ×0.2: core 1.0,
two TUI concerns (painter and routing) 1.5 combined, clipboard API 1.5,
consumer refactor 0.6, docs 0.2, review 0.1. Implementation uses v3.1 ×0.4:
authoring 0.2, core 0.8, two TUI concerns 2.0, native API 1.0 with bounded-novel
familiarity ×1.5, refactor 0.5, docs 0.2, review 0.5. Other work is familiar ×1.
Library check: existing painter/decoder/sgrState and standard os/exec supply the
reusable pieces; a small Pasteboard transaction preserves explicit literal bytes
and isolated named-board tests, so no additional library discount is claimed.
Design 1.98 ×1.15 + implementation 2.28 = 4.557, rounded to 4.56 hours.

## Revisions (implementation entry)

- 2026-09-14: User approved the durable plan. Plan-quality accepted PQ-1's
  independent child deadline refinement; derived estimate after acceptance.


## Implementation log

2026-09-14: Implemented shared physical frames and gesture arbitration, input-side
pointer routing, immutable click capture, ordered native clipboard writes and
nested console ownership. Existing click fixtures now send press/release gestures;
real editor and practice tests assert clipboard bytes and absence of accidental
grades. Held model and nested-screen tests exercise the shared input seam.

Focused race checks passed (8.077s); strict native and real PTY checks all ran and
passed (8.871s). Native text remains literal across Unicode/NUL/RTF/EPS prefixes;
all native tests use isolated boards. Mouse decoder fuzz passed 33,961 executions;
selection text/gesture fuzz passed 26,802/539 executions. Portable Linux build/vet,
repository vet, conformance skip guard and artifact checks pass. Full repository
suite and committed-window verification remain before the sole close review.

ARCH-DRY/PURE: layout and click target resolution have single owners. ARCH-ORDER:
watcher-side resize invalidation and router generation prevent stale actions;
clipboard child/queue shutdown is bounded and joins before terminal restoration.
ARCH-MOCK: real PTY + isolated native board checks complement stateful clipboard,
model, audio and screen fixtures. Nested ownership mutation was rejected by its
integration test. Documentation and lessons updated with the new interaction.

2026-09-14: Full repository tests passed (define 111.711s). Implementation
deliverables are checked; proceeding to committed-window verification and close.


2026-09-14: Close review BR-1 (cancellation-ingress-completeness) reproduced with
stateful clipboard barriers: dropped typing/page/wheel/byte-interrupt events and
scoped SIGINT could leave a drag alive. Cancellation now precedes admission and
foreground interrupt callbacks across the complete ingress set. Observer lifetime
belongs to the console and is detached on Stop. New tests failed before the fix;
focused race verification passed. Re-running verification before a second close.

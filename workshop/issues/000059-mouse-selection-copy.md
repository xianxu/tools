---
id: 000059
status: working
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
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
  a nonempty drag copies the selected text to the macOS clipboard. This is the
  proposed default; the optional question about copy-on-release versus a shortcut
  is still open. A click with no drag keeps its current action, once, on release.
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

- [ ] Drag selects text visibly and copies the exact selected text on release.
- [ ] Editor, answers and review screens share selection behavior; dragging over
      actions never activates them, while ordinary clicks still work once.
- [ ] Unicode, multiline/wrapped content and clipped rows copy without styles,
      spinner glyphs or layout padding.
- [ ] Streaming/playback, resize, scrolling, screen switching and exit cannot
      turn stale gestures into a copy or review action.
- [ ] Clipboard errors are visible, tests isolate the user's clipboard, and
      portable builds retain a valid unsupported-platform seam.
- [ ] Help/README and atlas reflect app-owned selection; tests and terminal
      conformance demonstrate the behavior.

## Plan

- [ ] Review the interaction/spec, incorporate the copy-trigger preference.
- [ ] Write and review the durable implementation plan, obtain approval, then
      enter implementation with sdlc change-code.
- [ ] Implement and verify shared selection, clipboard integration and consumers;
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

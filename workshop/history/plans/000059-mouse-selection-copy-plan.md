# Mouse selection and clipboard copy implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded independent units or superpowers-executing-plans for session-warm integration. Use one atomic SDLC close boundary, not per-task milestones.

**Goal:** Drag to highlight visible text and release to copy it, preserving normal clicks and preventing accidental review actions.

**Architecture:** The shared painter produces a bounded selectable frame. A pure gesture policy distinguishes click from drag; a console-owned input router handles pointers before type-ahead delivery and validates deferred clicks against screen/frame identity. A serialized clipboard queue invokes a native helper inside the same executable.

**Tech stack:** Go, existing raw terminal and screen renderer, SGR/legacy mouse protocols, Darwin Pasteboard APIs through cgo, subprocess cancellation, existing PTY and conformance harnesses.

**Spec:** `workshop/issues/000059-mouse-selection-copy.md`, approved 2026-09-14 including copy on release. Implementation awaits durable-plan approval and `sdlc change-code --issue 59`; estimate follows the plan gate.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| `selectionFrame` | `cmd/define/selection_frame.go` | new |
| `selectionCells` | `cmd/define/selection_frame.go` | new |
| `selectedText` | `cmd/define/selection_frame.go` | new |
| `selectionGesture` | `cmd/define/selection.go` | new |
| `selectionStep` | `cmd/define/selection.go` | new |
| `pointerClick` | `cmd/define/selection.go` | new |
| `formCell` | `cmd/define/play_loop.go` | modified |
| `Key` | `cmd/define/key.go` | modified |
| `decodeWheel` | `cmd/define/key.go` | modified |
| `decodeX10Mouse` | `cmd/define/key.go` | modified |

The frame owns physical rows and cell-to-text spans, with a role distinguishing
selectable text from layout/decorative cells. Each occupied cell maps to the entire
UTF-8 character span; attach zero-width combining marks to their base. Use the
renderer’s existing ANSI and width rules (`visibleIndex`, `stripEscapes`, `cellWidth`),
not a new global typography model. The gesture owns at most an anchor/end pair,
sticky drag state and frame identity. Its effects are none, click or copy; it
cannot perform IO. `pointerClick` carries screen identity and frame generation,
plus a resolved target classification, so a release can be validated before acting.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| `screen.layoutSelectionFrame` | `cmd/define/screen.go` | new | existing frame budgeting and hit geometry |
| `screen.paintActivity` | `cmd/define/screen.go` | modified | one shared layout and selection overlay |
| `liveScreen` | `cmd/define/screen.go` | modified | snapshot, selection and copy-failure state under mu |
| `interrupter` | `cmd/define/interrupt.go` | modified | console-owned input observation alongside foreground scope |
| `interrupter.Fire` | `cmd/define/interrupt.go` | modified | byte and signal cancellation observation before foreground callback |
| `interrupter.Observe` | `cmd/define/interrupt.go` | new | scoped console observer registration and cleanup |
| `pointerRouter` | `cmd/define/selection_input.go` | new | active screen and input-side gesture handling |
| `readKeys` | `cmd/define/rawterm.go` | modified | delegates to shared decoder/delivery implementation |
| `readInput` | `cmd/define/selection_input.go` | new | decoding, routing and bounded type-ahead |
| `clipboardWriter` | `cmd/define/clipboard.go` | new | context-aware text write seam |
| `clipboardQueue` | `cmd/define/clipboard.go` | new | ordered submissions and completion ownership |
| `processClipboardWriter` | `cmd/define/clipboard_process.go` | new | own executable, bounded stdin/diagnostics, kill and reap |
| `runClipboardHelper` | `cmd/define/clipboard_process.go` | new | strict private helper protocol |
| `writeNativeClipboard` | `cmd/define/clipboard_darwin.go` | new | synchronous native literal-text transaction |
| `writeNativeClipboard` | `cmd/define/clipboard_stub.go` | new | unsupported-platform result |
| `memoryClipboard` | `cmd/define/clipboard_test.go` | new | stateful ordered/failing clipboard fake |
| `deps` | `cmd/define/main.go` | modified | injected clipboard factory |
| `clipboardTarget` | `cmd/define/clipboard_target.go` | new | production canonical pasteboard name |
| `clipboardTarget` | `cmd/define/clipboard_target_conformance.go` | new | conformance-only isolated target injection |
| `realDeps` | `cmd/define/main.go` | modified | production clipboard factory |
| `main` | `cmd/define/main.go` | modified | private helper dispatch before ordinary startup |
| `console` | `cmd/define/replraw.go` | modified | shared pointer/clipboard lifetime |
| `newConsole` | `cmd/define/replraw.go` | modified | router, watcher and screen assembly |
| `replRaw` | `cmd/define/replraw.go` | modified | construct console before its input reader |
| `runEditor` | `cmd/define/replraw.go` | modified | validated click effects only |
| `runPlay` | `cmd/define/play_loop.go` | modified | same console/input assembly |
| `playSession` | `cmd/define/play_loop.go` | modified | validated click before formCell/region dispatch |
| `sittingInPlace` | `cmd/define/play_cmd.go` | modified | atomic router handoff on nested screen switch |

All consumers are in define: keep the component there (AGENTS.local.md). No
clipboard dependency belongs in the pure play package or internal/llm. The native
helper is an execution boundary, not a second installed program (ARCH-DRY/PURE).

## Display and gesture contract

Move the existing budget/placement calculation from paintActivity into
screen.layoutSelectionFrame, returning the clipped rendered rows, original prompt
cursor controls, role/region mappings and footer origin. Painting and selection
must consume that same result. Retain existing viewport clamping, pinned padding,
gap priority, activity-row budgeting and cursor restoration. Build selectable
cells as those rows are placed; never replay an independently invented layout.
Selection highlighting uses reverse-video spans over rendered glyphs, restoring
existing SGR state at span boundaries and preserving cursor placement. It never
reaches screen.Write or the exit transcript.

Ranges use inclusive occupied-cell endpoints, normalized for backward drags.
Selecting either half of a wide glyph copies that glyph once; attached combining
marks stay with it. Copy visible physical rows joined with LF, preserve internal
spaces and omit trailing layout padding. Blank layout-only rows and activity
cells contribute no text; a selected genuine empty content row contributes its
newline. Clipped-off characters cannot enter the clipboard. Ordinary click marks
are currently SGR decoration and therefore contribute no extra text.

A snapshot has at most 262,144 cells and 1 MiB of text/span source. Check dimensions
with overflow-safe arithmetic and limits before allocating. Exceeding either bound
disables selection for that frame and produces a visible refusal on an attempted
drag; rendering continues normally. The copy payload limit is also 1 MiB. Do not
truncate a copy silently. No auto-scroll, rectangular selection, multi-click word
selection, or global emoji/grapheme-width rewrite is included.

Frame generation changes when selectable text, layout, viewport, dimensions or
click targets change. Compare complete bounded frame values rather than using a
collision-prone content hash. Cosmetic SGR/highlight/activity-glyph changes alone
do not advance it if selectable text and geometry stay the same. A frame is
published for pointer use only after its terminal write succeeds. Failed writes,
suspension and Stop invalidate selection immediately.

| State/event | Effect |
|---|---|
| Idle/selected + left press | clear old selection; anchor current valid frame |
| Pressed + held motion to another cell | enter dragging; highlight normalized range |
| Dragging + motion back to anchor | remain dragging; never regain click semantics |
| Pressed + release at original cell | queue one frame-tagged click |
| Pressed + release at different cell | treat as drag even if motion reports were omitted |
| Dragging + release | submit nonempty copied text once; retain highlight |
| Pressed/dragging + invalidation | clear overlay; cancel until release/new press |
| Cancelled + release | consume without click or copy |
| Any + screen handoff/exit | invalidate; no old gesture or completion may alter new screen |

Clamp a drag endpoint outside valid bounds to the displayed viewport; a press
outside it is inert. Ignore right/middle/extended buttons, unheld motion, and
unsolicited releases. Legacy release has no button identity and can end only an
active left gesture. Preserve the terminal’s own modifier override when it keeps
reports from the app. No timer threshold distinguishes click from drag.

## Input routing, concurrency and ownership

Change mouse reporting from 1000 to button-event mode 1002, retaining 1006 encoding
and symmetric restoration. Decode raw press/motion/release into explicit key kinds;
KeyClick becomes a completed, validated application gesture, never a raw press.
Update press-only consumer tests to use complete gestures; no compatibility bypass
may grade a board on press.

newConsole creates a pointerRouter and shared clipboardQueue before readInput starts.
The reader handles pointer events synchronously against the router’s active
liveScreen; only completed clicks enter the loop queue. Text extraction happens
under the screen lock from the published frame, then clipboard submission happens
outside it. The loop validates pointerClick identity and captures its immutable region/footer
hit target while holding router then screen locks in one validation operation,
including active-screen ownership. A changed frame discards the click.
After releasing the lock, dispatch only that captured target: never re-read live
RegionAtRow/FooterRowAt by coordinate, because a clipboard notice or resize could
otherwise shift the target between validation and action. Change formCell to a
pure function of current question plus captured footer-entry/continuation/column;
the loop is the sole owner of question mutation. Use one shared validation helper
in both loops; never act on raw mouse events there.

The existing 256-entry type-ahead queue must not stop decoding mouse/interrupt input.
Keep accepted ordinary keys in FIFO order. When full, refuse the newest ordinary
key/click with one visible input-overflow notice per saturation episode; do not
silently drop old accepted input or create an unbounded spill queue. Mouse events
are consumed before admission; Ctrl-C fires its existing interrupter immediately.
This explicitly changes only overload behavior, which currently blocks all further
decoding. Keyboard/page/wheel input cancels active selection before being queued.

The resize watcher’s measurement callback invalidates the active router geometry
before delivering/coalescing the resize to either loop. Until a successful paint
with the observed dimensions arrives, new pointer gestures are refused. This
prevents selection against old dimensions while a loop is blocked. Viewport events
also invalidate pending click tickets at observation time, before deferred scroll.

Nested /play borrows the parent’s router, clipboard queue, key reader and watcher.
Switch active screen and increment the ownership generation before exposing the
sitting, and restore it on hand-back; never create a second reader or worker.
Lock order is router then screen; no screen method calls back into the router.
Native IO, worker joins and clipboard callbacks never run while either lock is held.

## Clipboard execution and failures

clipboardWriter.Write(ctx, text) returns only after the write attempt ends.
clipboardQueue has one worker, one active child and eight pending submissions.
Accepted copies execute FIFO, so an earlier write cannot overwrite a later copy.
Oversize/full-queue submissions fail visibly. Completion carries screen identity
and request sequence; only the newest relevant request may change feedback UI.
Screen switches suppress stale feedback but do not silently abandon accepted copies.

processClipboardWriter invokes os.Executable directly, without PATH or a shell,
with private arguments `--internal-clipboard-write <pasteboard-name>` and text on
stdin. Production uses the canonical macOS clipboard name. Each call has a two-second
context deadline, bounded 4 KiB diagnostics, and waits for the child to be reaped.
Set exec.Cmd.WaitDelay so inherited pipes cannot defeat cancellation. The helper
validates exact arguments, UTF-8 and a limit+1 bounded input read before touching
native state. Dispatch before realDeps/signal setup/dictionary/store acquisition.
Reject malformed private invocations; do not expose helper details in normal help.
The helper arms its own three-second time.AfterFunc watchdog at dispatch, before
reading stdin or entering native code; its callback calls os.Exit(124). This runs
in the child independently of parent cancellation and also ends an orphan after
a parent crash or SIGKILL. Normal return stops the watchdog. Process conformance
kills a parent while its helper is blocked and observes that child termination
occurs within the independent deadline. Parent cancellation still kills/reaps
promptly; the watchdog bounds survival when that owner disappears.

For foreground PTY conformance, use a dedicated `define_clipboard_conformance`
build tag that replaces only clipboardTarget. The normal implementation (negated
tag) always returns the canonical clipboard and ignores test environment variables.
The tagged implementation requires an explicit isolated pasteboard name from the
harness, rejects missing/empty/canonical names, and returns an error without starting
a writer on invalid configuration. The harness creates and retains the named board
before launching this build of the actual main entrypoint. Its child helper uses
the same tagged executable but dispatches before factory creation, with the validated
name passed explicitly. Thus the real input, painter, queue and native helper paths
run without any test's fallback reaching the user's clipboard. Keep the existing
normal builtBinary helper unchanged; a separate conformance builder supplies the
extra tag and executable to the shared PTY launcher.

The Darwin helper calls one synchronous C transaction: allocate target CFString and
explicit-length CFData, PasteboardCreate, clear, PutItemFlavor with
`public.utf8-plain-text` and item ID 1, then release every reference on every path.
No deferred data promises. Thus NUL and RTF/EPS-looking prefixes remain literal
text. A helper has one caller of the non-thread-safe Pasteboard API; blocking native
code is bounded by killing/reaping the child, not abandoning an in-process cgo
worker. A timeout is not rollback and may race a completed clipboard mutation;
report failure without claiming the previous clipboard was preserved.

Copy failure retains the failed payload and exposes a transient, non-transcript
footer notice with a click-to-retry action. This notice may invalidate the old
visual range but must not discard the retained payload. Its retry target is handled
by the router before ordinary region/form clicks, never by a new keyboard shortcut.
Only the current failure owns that payload; a new selection, dismissal, screen stop
or successful retry releases it. If no footer row fits, temporarily replace the
prompt display with the notice while preserving the stored editor prompt, restoring
it on dismissal/input. Feedback itself is nonselectable and cannot recursively copy.

Console shutdown stops admission, cancels active execution, drops pending requests,
kills/reaps the helper and joins the worker before terminal restoration. Cleanup is
idempotent, outside screen locks. Copy snapshots and failed payloads die with that
owner; only explicitly copied clipboard text persists (ARCH-ORDER/FUNERAL).

## Chunk 1: Deliver and verify the complete interaction

### Task 1 — Pure frame and gesture policies

Files: create selection.go, selection_test.go, selection_frame.go and
selection_frame_test.go under cmd/define.

- [x] Write selectionStep tests with generated event sequences and exclusive-effect
  invariants; test selectionCells/selectedText with bounded Unicode/ANSI inputs
  against exact text and whole-character coverage oracles. No IO mocks.
- [x] Run focused selection tests red; implement the policies and rerun green.
  Commit this independent core, then run the full cmd/define package so committed
  declaration/status guards are exercised.

### Task 2 — Ordered clipboard seam and isolated native conformance

Files: create clipboard.go, clipboard_test.go, clipboard_process.go,
clipboard_process_test.go, clipboard_darwin.go, clipboard_stub.go,
clipboard_conformance_test.go, clipboard_target.go and
clipboard_target_conformance.go under cmd/define; modify main.go.

- [x] Test clipboardQueue against memoryClipboard with controlled completion,
  failure and cancellation order, checking accepted-write FIFO and bounded workers.
  Test processClipboardWriter/runClipboardHelper against a stateful child process
  harness for input/output limits, literal input, exit status and kill/reap.
- [x] Run focused clipboard tests red; implement queue, private helper dispatch,
  native transaction and portable stub. Wire only the factory in deps/realDeps;
  foreground activation lands with router integration.
- [x] Native conformance creates a unique pasteboard with PasteboardCreate(NULL),
  retains it, obtains its name via PasteboardCopyName, and invokes the real helper
  against that name. Inspect exact UTF-8 bytes and flavor inventory; clear/release
  the isolated board at cleanup. Never use the general clipboard in tests.
- [x] Run focused/race/native tests green, shared conformance guard, and Linux
  build/vet. Commit; run the full cmd/define package on the committed window.

### Task 3 — Shared painter and live selection overlay

Files: modify screen.go; create selection_screen.go/selection_screen_test.go;
extend screen_test.go as needed without deleting still-valid painter obligations.

- [x] Test layoutSelectionFrame and live selection against the independent terminal
  oracle over generated dimensions, decorated text and cursor positions, checking
  copy/highlight/region agreement and unchanged transcript. Control invalidation
  and write failure with stateful terminal writers.
- [x] Run focused tests red; extract the existing single layout and add its frame
  snapshot, overlay and scoped copy feedback. Preserve every existing geometry
  obligation, including #36's glyph row and exact-width RenderLine normalization.
- [x] Run screen/selection tests and race checks green. Keep Tasks 3–5 as one
  integrated commit: decoder, console and consumer contracts must switch together.

### Task 4 — Pointer routing, decoder and console lifetime

Files: modify key.go, key_test.go, rawterm.go, rawterm_test.go, replraw.go,
play_loop.go, play_cmd.go; create selection_input.go/selection_input_test.go.

- [x] Fuzz decodeWheel/decodeX10Mouse through decodeKey with malformed/partial
  reports and coordinate bounds; test readInput with saturated type-ahead and
  independent pointer/interrupt barriers. Use a stateful active-screen/router rig
  for resize observation, ownership handoff and stale click tickets. Force a
  repaint or active-screen switch between release and consumer dispatch; the
  captured-target oracle must show no lookup, playback or grade against new geometry.
- [x] Run focused tests red; implement 1002 mode/restoration, raw pointer kinds,
  router admission and watcher-time invalidation. Build consoles before readers,
  share router/worker through nested /play, and join cleanup before handBack.
- [x] Route completed clicks through shared validation in runEditor/playSession
  before clicked/formCell/playRegion. Replace press-only integration fixtures with
  decoded press/release gestures. Leave pure play.Input unchanged.

### Task 5 — Real consumers, docs and close

Files: create selection_paths_test.go and selection_conformance_test.go; update
existing mouse/ask/play integration fixtures, cmd/define/pty_conformance_test.go,
cmd/define/command.go,
cmd/define/README.md, atlas/define.md, and workshop/lessons.md when findings warrant.

- [x] Test actual editor, standalone practice and nested /play against stateful
  clipboard and display seams: recorded grades/audio and clipboard contents are
  independent effect oracles. Hold real fake LLM/audio operations while injecting
  gestures, shape changes and saturated type-ahead to expose delayed-input races.
- [x] PTY conformance drives raw mouse reports through the tagged build of the
  actual main entrypoint described above and copies only to its isolated board.
  Verify the target factory fails closed for absent or invalid test configuration;
  normal builds ignore the test configuration entirely. Verify painted highlight, exact copied text,
  click-vs-drag outcomes, busy-operation handling and terminal restoration. Reuse
  shared SkipOrFail for absent external dependencies; update atlas inventory.
- [x] Replace modifier-only copying guidance with drag/release behavior, literal
  text semantics, failure retry and local clipboard scope. Document queue overload
  behavior and viewport-only selection without exposing the internal helper.
- [x] Run `go test ./... -count=1`, focused selection/clipboard race and bounded
  fuzz tests, strict targeted PTY/native conformance, `go test ./internal/conformance`,
  `go vet ./...`, `GOOS=linux CGO_ENABLED=0 go build ./...` and corresponding vet,
  plus `git diff --check`. Commit Tasks 3–5 and run the full cmd/define package on
  the committed window. Record evidence, tick deliverables, then sdlc close --issue
  59 for the sole fresh-context boundary review; resolve findings and ship via
  sdlc pr → sdlc merge. Do not add a duplicate boundary review.

## Scope, coordination and approval

ARCH-PURPOSE: both screens and both review entry paths share the gesture decision;
clipboard failure and busy-operation input are part of the delivered behavior.
ARCH-MOCK: stateful clipboard/process/display fakes and isolated native/PTY checks
cover the actual seams. ARCH-SECURE: copy only the explicitly selected bounded text,
never interpolate it into argv/shell/logs; validate helper input before mutation.
ARCH-CONSTRAINTS: frame/payload/queue/deadline bounds above keep work finite.

#54 plans background tasks and #45 playback responsiveness. This plan does not
restructure their domain work or make LLM/audio operations asynchronous; input-side
routing supplies selection responsiveness independently. Recheck overlapping
console changes before integration. Preserve #36 activity ownership and #58 model
provenance. Scope excludes remote/OSC52 forwarding, paste, terminal-native modifier
configuration, multi-click/rectangular selection and auto-scroll.

The user approved the spec and copy-on-release behavior on 2026-09-14. This durable
plan passed fresh-context review and artifact guards; it awaits user approval
before change-code.


## Revisions

- 2026-09-14: Resolved click validation's check/use race: capture immutable hit
  metadata with validation under one lock, then make formCell consume that value
  rather than re-querying the live display after release. Located help guidance
  in command.go. These details preserve the approved interaction.

- 2026-09-14: Plan review found that the foreground PTY path lacked an isolated
  clipboard target seam. Added a conformance-only build-tag factory with fail-closed
  validation and a dedicated tagged binary builder; production uses the canonical
  target unconditionally. Expanded click validation to include router ownership
  under the established lock order, with a deterministic intervening repaint oracle.

- 2026-09-14: Fresh-context re-review approved both corrections with no remaining
  blockers. Plan/artifact guard tests passed (1.252s); no product code changed.

- 2026-09-14: User approved implementation. Plan gate PQ-1 identified orphan
  lifetime after abrupt parent death; specified a child-owned three-second exit
  watchdog before input/native work, with parent-death process verification.

- 2026-09-14: Implementation found the existing displayRows count under-budgeted
  wide characters when a column remained before a soft wrap. The painter now
  shares physical-row traversal for counting, clipping and snapshots; independent
  terminal geometry tests reproduce and verify this fix. Related clipped-region
  hit tests refuse invisible glyph cells. Input, painter and consumer code were
  developed concurrently in isolated file ownership; code commits are consolidated
  to keep the coupled declaration/fixture contracts reviewable and green.

- 2026-09-14: Implementation deliverables and verification completed. Full
  repository tests passed (define 111.711s), focused race 8.077s, strict native/PTY
  8.871s, mouse fuzz 33,961 executions, vet and Linux build/vet passed. Checkboxes
  record delivered code/test work; committed-window checks and the SDLC close/PR
  gates follow before publication. The previously pending plan approval was
  granted by the user before change-code.

- 2026-09-14: Close BR-1 exposed cancellation paths that bypassed admission.
  Enumerated admitted/rejected keyboard, page and wheel events, byte Ctrl-C
  (scoped and unscoped), and scoped SIGINT. All now cancel at observation;
  interrupter.Fire notifies a console-owned observer before its foreground callback.
  Router Stop detaches that observer; stale cleanup cannot remove a newer owner.
  Deterministic clipboard barriers prove zero selection writes after cancellation,
  without relying on a later foreground repaint. Both original bypasses failed
  the new tests before the fix. This completes the existing cancellation contract.

- 2026-09-14: Close review returned SHIP after independently rejecting both
  BR-1 mutations; no remaining findings. Product code and verification complete,
  with publication following through the deterministic SDLC PR/merge gates.

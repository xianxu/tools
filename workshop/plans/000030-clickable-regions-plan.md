# Clickable Regions Implementation Plan (`#30`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Click the headword to hear it; click the language after `ORIGIN` to hear it in that language.

**Architecture:** `define`'s interactive loop takes the alternate screen and owns every line it shows. The screen is an `io.Writer` with a line buffer and a viewport, so `Render`, the ask stream, command output and the indicator all feed it without changing their code. A click at viewport row R maps to buffer line `R + offset`, exactly, because nothing else can scroll.

**Tech Stack:** Go 1.26, `golang.org/x/term` (already a dependency), `os/signal` for SIGWINCH. Tests: the existing `creack/pty` conformance harness, plus unit tests on the pure screen model.

**Two milestones, ONE publish.** `M1` is the screen layer and `M2` is the clicks; each is a review boundary because each can be wrong in a different way, and AGENTS.md publishes once at issue close so the scrollback change and the clicks reach `main` together.

---

## Decisions

**D1 — the alternate screen answers the scrollback question by construction, and no measurement is needed.** `#30`'s Spec worried that click coordinates go stale when the entry scrolls, and proposed measuring whether mouse tracking captures the wheel. That measurement is moot: **the alternate screen buffer has no scrollback.** There is nothing above the viewport for the terminal to show, so there is no offset `define` does not own. Terminals that offer "scroll in the alt screen" implement it by SENDING KEYS to the application, which the application then handles — still its own scroll. The wheel question survives only as UX (should the wheel scroll our buffer: yes), not as correctness.

**D2 — the operator's other two candidates are rejected, with reasons, so they are not reopened.** *Colour as the carrier*: the palette really is markup by meaning already (`head`, `ipa`, `pos`, `num`, `ex`, `sect`), but **there is no escape sequence for "report the attributes at row R, column C"** — mouse reporting sends coordinates and a button. The terminal remembers the colour and can never be asked, so colour is markup a HUMAN reads. *OSC 8*: the correct implementation of "markup the terminal carries", rejected on cost — Terminal.app does not support it at all, and a click opens a URL through the OS, so reaching the running process needs a custom scheme plus a helper.

**D3 — the session transcript is printed to the normal buffer on exit.** The alternate screen tears down on quit, so without this a session's entries vanish from the terminal's history — and today `define arrondissement` leaves the entry where you can scroll back to it tomorrow or copy from it. Losing that silently is a regression a user meets immediately. The screen already holds every line, so this is a loop over the buffer at exit. Stated as a decision because the alternative — accept the loss — is defensible and the operator may prefer it.

**D4 — the cooked/raw split is REMOVED inside the screen, and that is the milestone's largest simplification.** `cooked()` exists so a definition's bare `\n`s translate while a definition is printed, and `#14` established "render cooked, play raw" because flapping modes is what makes Ctrl-C reach the key reader. Inside an app-owned screen there is nothing to flap: the screen owns line placement, so no output depends on the line discipline, and raw mode is continuous — which is what Ctrl-C wanted all along. This deletes the hazard `#29`'s `/pron` had to defer a replay around and that `workshop/lessons.md` records twice. It is also the riskiest edit in the issue, which is why `M1` is its own boundary.

**D5a — this REPLACES the seam open issue `#32` is filed against, and `#32` narrows rather than dies.** `#32` says `reportVoice` writes a bare `\n` to a raw terminal and that the fix belongs at the seam — the seam being `crlfWriter`, which is wrapped for the ask path and the review loop but not for `replayInPlace`. Inside the screen the question dissolves for the INTERACTIVE path: the screen owns line placement, so no caller's line endings matter there. `#32` keeps its `--play` half, which continues to draw its own frames through `crlfWriter` and is untouched by this issue. Whichever lands first, the other must be re-read: recorded here rather than left for one of us to discover.

**D5b — STDERR routes through the screen too.** It has to: a diagnostic written straight to the terminal while the alternate screen is up lands wherever the cursor happens to be and corrupts the frame. So `define:` lines become buffer lines like any other — which also means they scroll with the transcript and survive to the exit dump, where today they are simply gone. The one-shot and piped paths keep writing to the real stderr (D6), so a script's `2>` still works.

**D5 — the screen is an `io.Writer`, so its callers do not change.** `Render` returns a string; the ask path already streams through `crlfWriter`; commands and the indicator write to `stdout`. Swapping the writer means every one of them feeds the buffer unchanged, and `Render`'s purity, the no-data-loss invariant and the whole fixture corpus stay untouched. The screen REPLACES `crlfWriter` on this path: both translate for a raw terminal, and having two things own line endings is how they drift.

**D6 — only the interactive loop.** `define <word>`, `echo w | define`, `-raw` and `> out.txt` keep today's behaviour exactly, including the "ephemeral UI vs record" doctrine — the `♫ playing 3×` line, `-no-color` making output a record. Inside the alternate screen that doctrine is vacuous, because everything is ephemeral. Both halves stay true; the split is stated rather than the doctrine discarded.

**D7 — NON-GOAL: text selection.** With mouse tracking on, drag-select goes to the application, so copying needs Option (iTerm2, Terminal.app) or Shift. Implementing selection is a second product — a whole model of anchors, extents and clipboard integration — and this issue does not attempt it. The keybinding is documented where a user meets it.

---

## What this plan asserts about the existing tree, verified

**THE RULE, and it is the deliverable of PQ-10 rather than the three sites it
named: every claim this plan makes about current `cmd/define` behaviour carries a
`file:line` and was checked.** Three such claims were wrong in one round — and
this session has produced the same class four times before (a `rôle` measurement
cited about the wrong thing, two test names that did not exist, a comment about a
mask that measurement disproved). A plan is read as a description of the tree, so
an unchecked claim about it is worse than a missing one.

| claim | verified at | status |
|---|---|---|
| `Render` colours the section NAME, not the language inside it | `render.go:200-218` | true — hence a new pass (PQ-1) |
| `crlfWriter` is the raw-mode line-ending seam | `crlf.go:14-20` | true; wrapped for the ask path and `--play`, NOT for `replayInPlace` |
| the menu's erase arithmetic has a documented known limit | `replraw.go:90-97` | true — "if the menu does not fit below the cursor the terminal scrolls and the cursor-up count lands a row off" |
| `cooked()` drops raw mode around a lookup | `replraw.go:38-51` | true |
| `enterRaw` / `restore` own terminal state | `rawterm.go:21,29` | true — so `enterAlt`/`leaveAlt` belong beside them |
| the CSI scanner delimits `ESC[5~`/`ESC[6~` correctly | `key.go:102-116` | **half true** — it delimits, then returns `KeyUnknown`. New `KeyKind`s are needed; "decoded by the existing scanner" was wrong |
| **Ctrl-U and Ctrl-D are free for scrolling** | `key.go:48-55` | **FALSE** — `0x04` is `KeyEOF` (ends the session on an empty line, `editor.go:107`) and `0x15` is `KeyKillLine` (`editor.go:100`). Rebinding either is a regression, and a row testing only PageDown would ship it green |
| `newPalette` spends six colours plus deck-word bold-green | `render.go:28-37`, `highlight.go:15` | true |
| `sgrState.resume()` splices an attribute into already-styled text | `sgr.go:27,57` | true — this is the machinery M2.5 uses, not a new one |
| `OriginLanguage`'s stage mask is length-changing | `origin.go:156` | true, `strings.ReplaceAll(text, stage, " ")` — see the offset correction below |
| the width probe runs once, at flag parse | `main.go:961` | true — there is no resize handling to build on |

**The offset bug this sweep caught before it shipped.** `Mention.Offset` was to be
the position of a language name, and the search runs AFTER the stage mask
replaces variable-length stage names with a single space. That offset indexes the
MASKED text, not `sec.Text` — measured on the `concrete` fixture, a 13-character
shift before "French" — so every `ORIGIN` region would have been drawn in the
wrong column. The mask must therefore be length-PRESERVING (replace each stage
with spaces of the same width) so offsets survive it, which is a one-line change
to `#35`'s code and is where `M2.1` starts.

---

## Milestone M1 — the screen owns the terminal

### Core concepts

#### Pure entities

| Name | Lives in | Status | Kind |
|------|----------|--------|------|
| `screen` | `cmd/define/screen.go` | new | PURE |
| `screen.Write` | `cmd/define/screen.go` | new | PURE |
| `screen.Frame` | `cmd/define/screen.go` | new | PURE |
| `screen.Scroll` / `screen.clamp` | `cmd/define/screen.go` | new | PURE — one place spells the viewport's limits |
| `screen.Page` | `cmd/define/screen.go` | new | PURE (M1.4a) — a screenful less one line of overlap |
| `screen.Paint` | `cmd/define/screen.go` | new | PURE, given the writer |
| `screen.Transcript` | `cmd/define/screen.go` | new | PURE (M1.5) |
| `displayRows` / `clipVisible` | `cmd/define/screen.go` | new | PURE (rework) — the frame is budgeted in DISPLAY ROWS |
| `visibleCells` | `cmd/define/render.go` | modified | PURE (rework) — was `visibleLen`; the ONE owner of "how wide is this", now measured in terminal COLUMNS |
| `cellWidth` / `isWide` | `cmd/define/render.go` | new | PURE (rework) — a combining mark is 0 columns and a CJK rune is 2, both daily traffic for a dictionary |
| `decodeWheel` / `atoiPrefix` | `cmd/define/key.go` | new | PURE (M1.4b) |
| `decodeX10Mouse` | `cmd/define/key.go` | new | PURE (rework) — the encoding mode 1000 falls back to |
| `KeyPageUp` / `KeyPageDown` / `KeyWheelUp` / `KeyWheelDown` | `cmd/define/key.go` | new | PURE |
| `winSize` | `cmd/define/rawterm.go` | new | PURE (M1.4) |
| `crlfWriter` | `cmd/define/crlf.go` | unchanged | INTEGRATION — still used by `--play`, which keeps its own drawing |
| `truncate` | `cmd/define/command.go` | modified | PURE (rework) — delegates to `clipVisible`, so cutting and measuring have one owner |

- **`screen`** — a line buffer plus a viewport: `lines []string`, `offset int`, `rows, cols int`.
  - **Relationships:** 1:1 with an interactive session; owns every line it displays.
  - **`Write` is the seam** (D5): it splits incoming bytes on `\n`, appending to `lines` and continuing the last partial line, so a streamed reply that arrives in fragments lands as text rather than as frames. It performs no terminal IO.
  - **`Frame` is PURE**: `(lines, offset, rows, cols) -> []string`, the rows to paint. Unit-testable with no terminal, which is what keeps the scroll arithmetic out of a pty test.
  - **DRY rationale:** replaces `crlfWriter` on this path and subsumes `menuDrawn`'s erase arithmetic, whose documented known limit — "if the menu does not fit below the cursor the terminal scrolls and the cursor-up count lands a row off" — cannot occur when the app places every row.
  - **Future extensions:** `M2`'s region map hangs off the same per-line record.

#### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `liveScreen` | `cmd/define/screen.go` | new | the terminal — the ONLY part of the screen that does IO |
| `display` | `cmd/define/replraw.go` | new | the loop's whole view of the terminal: `Draw`, `Page`, `Scroll`, `Resize` |
| `enterAlt` / `leaveAlt` | `cmd/define/rawterm.go` | new | `\x1b[?1049h/l` |
| `enterMouse` / `leaveMouse` | `cmd/define/rawterm.go` | new (M1.4b) | `\x1b[?1000h` + `\x1b[?1006h` |
| `rawSession.control` | `cmd/define/rawterm.go` | new (rework) | where mode sequences go — an `io.Writer`, so the restore protocol is assertable with no terminal |
| `watchResize` | `cmd/define/rawterm.go` | new | SIGWINCH |
| `terminalRows` | `cmd/define/main.go` | new | `term.GetSize`, beside `terminalWidth` |
| `terminalSize` / `terminalCols` | `cmd/define/main.go` | new | `term.GetSize` — the TRUE shape, which cannot return `terminalWidth`'s "do not wrap" sentinel |
| `handBack` / `onceHandBack` | `cmd/define/replraw.go` | new | the exit sequence: stop painting, restore, print the session — as a function, so it is pinnable without a pty |
| `wheelFromButton` | `cmd/define/key.go` | new | the one reading of a mouse report's button byte, shared by both encodings |
| `replRaw` / `runEditor` | `cmd/define/replraw.go` | modified — `cooked` deleted (D4) | the terminal |

- **`watchResize`** — there is NO resize handling today; width is read once at flag parse (`main.go`). A full-screen program must handle it or the frame is wrong after the first drag.
  - **Injected into:** the editor loop as a channel, beside `keys`, so the loop's select grows one case rather than the screen learning about signals.

**ARCH-MOCK.** The external dependency is the terminal, and the repo already has its stateful double: the `creack/pty` conformance harness (`startDefine`, `watch`). `M1` adds rows there — alt screen entered and left, a resize redrawing, the transcript appearing after exit — and the pure `Frame` tests carry the arithmetic.

### Tasks

- [x] **M1.1 — `screen` as a pure model.** `Write`, `Frame`, `Scroll`, plus rows/cols. Table tests: a partial write continues the last line; a write containing `\n\n` appends an empty line; `Frame` clamps the offset at both ends; a viewport taller than the buffer pads rather than repeating. No terminal.
- [x] **M1.2 — `Paint` and the alt screen.** `enterAlt`/`leaveAlt` on `rawSession`, so a Ctrl-C or a panic leaves the terminal restored — the same obligation `enterRaw` already carries and the reason this belongs there rather than in `screen`.
- [x] **M1.3 — the editor draws through the screen**, `cooked` deleted (D4). Every current writer keeps writing; only the destination changes.
- [x] **M1.4a — KEYS that move the viewport.** Without this `M1` ships a scroll model nothing exercises and a user cannot reach: the wheel is `M2`, and no key scrolls today.
      **PageUp/PageDown ONLY. NOT Ctrl-U/Ctrl-D**, which an earlier draft proposed and which are already bound: `0x04` is `KeyEOF` and ends the session on an empty line (`key.go:48`, `editor.go:107`), `0x15` is `KeyKillLine` (`key.go:50`, `editor.go:100`). Taking either is a silent regression in an editor people already use.
      The CSI scanner DELIMITS `ESC[5~`/`ESC[6~` correctly (`key.go:102-116`) and then returns `KeyUnknown`, so this adds two `KeyKind`s — not "already decoded", as the same draft said.
- [x] **M1.4b — the WHEEL scrolls** (added mid-stream, see Revisions). Mouse
      tracking moves from `M2.2` to here, because the wheel is a viewport gesture
      and the viewport is `M1`'s: in the alternate screen a terminal delivers the
      wheel as ARROW KEYS, which this editor binds to the history walk, and the
      bytes are identical so nothing can tell them apart. `1000`+`1006` on
      `rawSession` beside the alt screen, `decodeWheel` in the CSI scanner, and
      the drag-select cost of D7 paid now — stated in `/help`, where a user meets
      it. Clicks stay inert: coordinates mean nothing until `M2` has a region map.
- [x] **M1.4 — resize.** SIGWINCH → re-measure → repaint. The one thing that cannot be unit-tested is the signal, so the pty row drives a real `TIOCSWINSZ`.
- [x] **M1.5 — the transcript on exit** (D3), and the pty row that it survives.
- [x] **M1.6 — docs**, and the row is wider than it was written: a docs sweep follows ANY change to this surface, including the ones a boundary review produces after the row is ticked. The rework added the display-row budget, the cell-width owner, the throttle, the X10 fallback and `handBack`, and each landed with the code rather than after it. Originally: the atlas's raw-mode section, which currently explains the cooked/raw dance that D4 removes. That prose goes false, so it is rewritten rather than appended to.

### M1 Done-when

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | the viewport arithmetic is right | `TestScreenFrame` | `Frame` stops clamping the offset |
| 1b | **the frame FITS the terminal, in display rows** | `TestScreenFrameFitsTheTerminalInDisplayRows`, `TestScreenClipsTheViewNotTheBuffer` | a line wider than the terminal is counted as one row, so the frame overflows and the terminal scrolls |
| 1c | **a click types nothing, in either mouse encoding** | `TestDecodeX10Mouse`, `TestX10ClickTypesNothing`, `TestDecodeWheel` | a report is delimited but its payload is not consumed |
| 2 | a streamed fragment lands as text, not a frame | `TestScreenWriteBuildsLines/a partial line CONTINUES` | `Write` splits on every call boundary |
| 3 | the terminal is restored on every exit — raw mode, the alt screen and mouse reporting, in that order | `TestRestoreHandsBackEveryTerminalState` (asserts the bytes AND the order, verified falsifiable), `TestRestoreSendsNothingItDidNotTake`, `TestEnterDoesNotClaimAStateItCouldNotWrite`; on a real pty the Fatal in `TestPTYTranscriptIsPrintedOnExit` | any leave is dropped from the restore path |
| 3b | **mouse reporting is given back** (M1.4b) | `TestPTYMouseTrackingIsAskedForAndGivenBack` | the disable is dropped, and the next program run in that terminal gets escape sequences typed into it |
| 4 | the viewport can be moved by a user | `TestEditorPageKeysScroll` | the key case is removed from the select |
| 4b | **the keys it already had still work** | `TestCtrlDStillEndsTheSession`, `TestCtrlUStillKillsTheLine` | Ctrl-U or Ctrl-D is rebound to scrolling |
| 5 | a resize repaints | `TestPTYResizeRepaints` | the SIGWINCH case is removed from the select |
| 6 | the transcript survives exit | `TestPTYTranscriptIsPrintedOnExit` | D3's loop is removed |
| 7 | the one-shot and piped paths are untouched | the existing suite, unchanged | any of them starts entering the alt screen |
| 8 | **the wheel scrolls rather than walking history** (M1.4b) | `TestWheelScrollsRatherThanWalkingHistory`, `TestDecodeWheel`, `FuzzDecodeWheelIsBounded` | tracking is not enabled, so the terminal sends arrows and Up/Down recall words |
| 9 | **a prompt is only shown when the loop is waiting** (M1.3b) | `TestNothingIsWrittenWhileAPromptIsShown` | the live edge is left standing through a lookup, and the submitted line is repainted under its own definition |

---

## Milestone M2 — the clicks

### Core concepts

| Name | Lives in | Status | Kind |
|------|----------|--------|------|
| `Region` | `cmd/define/render.go` | new | PURE |
| `Render` | `cmd/define/render.go` | modified | its BODY was swept by M1's `visibleLen`→`visibleCells` rename; the SIGNATURE change — the second return value — is still M2.1's, and this row says `modified` because the status column describes this window's diff, not the milestone's intent |
| `decodeMouse` | `cmd/define/key.go` | new | PURE — extends M1's `decodeWheel`/`decodeX10Mouse` to buttons |
| `screen.RegionAt` | `cmd/define/screen.go` | new | PURE |

- **`Region`** — `{Kind, Text, Lang, Line, Col, Width}`: what a span of rendered text OFFERS.
  - **The headword falls out of the existing walk; the ORIGIN language does NOT, and an earlier draft of this plan claimed it did.** `Render` colours `sec.Name` — the word "ORIGIN" — and passes `sec.Text` through `opt.prose(wrapText(...))`, which highlights DECK words. Nothing isolates "French" inside that text. So the language region needs a new pass over the section text, and that pass is the same matching `#35` already does.
  - **EVERY modern language mentioned, not the first.** `#35`'s `OriginLanguage` is first-named-wins, which is correct for `/pron` with no argument — NOAD's convention is that the first source named is the immediate one. Wrapping it here would make only one language clickable and would contradict the insight this issue is FOUNDED on, recorded in its own Spec: *"a click has nothing to pick — `French` and `Italian` are two separate targets and the user points at the one they meant."* `piano` names both; both must be clickable.
  - So `#35`'s origin reader is refactored, not wrapped: `OriginLanguageMentions(Entry) []Mention{Name, Lang, Offset}` becomes the producer, and `OriginLanguage` becomes "the first of those". One cut-and-mask, two consumers — rather than a second spelling of the rule `#35`'s close review already spent four findings getting right.
  - Positions are reported AFTER wrapping, since wrapping decides a column.

- **`decodeMouse`** — `ESC[<b;x;yM/m` → button, column, row. `key.go` already scans to the CSI final byte (`#14`), so the reader delimits the sequence correctly and this only interprets what is inside it.
  - **It is this issue's one adversarial-input surface**, and `#14`'s lesson is exactly here: *"Don't assume an escape sequence's length"* — special-casing `ESC[3~` consumed four bytes of a six-byte `ESC[3;5~` and injected `5~` into the word being typed. A mouse sequence has THREE numeric parameters and two possible final bytes, so it has more ways to be malformed than that one did.
  - **Strategy:** a table over well-formed sequences (press, release, each button, wheel up/down, coordinates past 223 where the legacy encoding would have wrapped), plus a fuzz target asserting the decoder consumes a bounded number of bytes and never returns a coordinate it did not read. A decoder that over-consumes eats the next keystroke, which is the failure `#14` shipped once already.

### Tasks

- [ ] **M2.1 — `Render` emits regions.** The signature change is the risk: every caller and every golden test touches it. Keep the string identical — assert byte-equality against the current output over the whole corpus, so a regions change cannot silently alter what is drawn.
- [ ] **M2.2 — `decodeMouse` for BUTTONS.** The tracking enable/disable and the wheel half of the decoder landed in `M1.4b`, paired with the alt screen on `rawSession` so a crash cannot leave tracking on, with the bounded-consumption fuzz target this row called for. What is left is the press: its coordinates, which are inert today because nothing can look them up yet.
- [ ] **M2.3 — hit test**: `screen.RegionAt(row, col)`, which is a lookup in the per-line region list. Pure.
- [ ] **M2.4 — the two actions.** Headword → replay. `ORIGIN` language → `/pron <that language>`, which after `#35` is a call into `OriginLanguage`'s map rather than new inference.
- [ ] **M2.5 — discoverability, STATIC rather than on hover** — and the tracking mode is the reason. Hover needs `1003` (any-event tracking), which streams an event for every cell the pointer crosses, so the loop would wake constantly to redraw an underline. `1000` (button press only) is what this issue enables, and with it the app never learns where the pointer is. So a clickable span is marked in the FRAME: the palette (`newPalette`) already spends `head`, `ipa`, `pos`, `num`, `ex`, `sect` and bold-green for deck words, so the mark is an ATTRIBUTE — underline — added to the span's existing colour rather than a seventh colour competing with them.
      It is spliced by the SCREEN, not by `Render`, because D6 promises the one-shot and `-raw` bytes are unchanged. `sgr.go`'s `sgrState.observe`/`resume` (`sgr.go:27,57`) is the existing machinery for reopening styles around an inserted attribute; this uses it rather than a second one.
- [ ] **M2.6 — degrade**, and the case is NOT only "a terminal that reports no mouse". The exposure that actually bit was a terminal that reports the mouse in an encoding we did not ask for: mode `1000` falls back to X10 (`ESC[M` + three raw bytes), which the CSI scan delimited at `M` and left three payload bytes to be typed into the line. Fixed in M1's rework (`decodeX10Mouse`); this row keeps the rule that produced it — **for every mode we enable, the decoder answers every encoding that mode can reply in** — and applies it to whatever M2 turns on.

### M2 Done-when

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | clicking the headword plays it | `TestClickOnHeadwordReplays` | `RegionAt` stops matching the headword span |
| 2 | clicking `ORIGIN French` plays French | `TestClickOnOriginLanguagePlaysIt` | the region's language is dropped |
| 3 | rendering is byte-identical | `TestRenderOutputUnchangedByRegions` | `Render` alters a byte while collecting |
| 4 | a clickable span is visibly clickable before it is clicked | `TestScreenMarksClickableSpans` — the SCREEN, not `Render`: D6 promises `define <word>` and `-raw` keep today's bytes, so an underline emitted by `Render` would leak to them and redden row 3 | the underline attribute is dropped from the frame |
| 5 | the mouse decoder is bounded and correct | `TestDecodeMouse` + `FuzzDecodeMouseIsBounded` | it consumes past the final byte |
| 6 | a mouse-less terminal is unaffected | `TestPTYWithoutMouseBehavesAsBefore` | the enable is emitted unconditionally |
| 7 | regions are one registry, not two special cases | `TestEveryRegionKindIsActionable` | a kind is added with no action |
| 8 | tracking is disabled on exit | `TestPTYMouseTrackingIsLeftOnExit` | the disable is dropped |

---

## Verification before close

```bash
go test ./...
go test -tags conformance ./cmd/define/    # unsandboxed; the pty rows are here
```

Then, on a real terminal: look a word up, click the headword, click `ORIGIN French`, scroll with the wheel, resize the window, quit and confirm the transcript is in the scrollback.

**Close:** two milestones, each with `sdlc milestone-close`; one `sdlc close` and one publish.

## Revisions

### 2026-08-29 — plan-quality round 1 (PQ-1…PQ-6)

- **PQ-1 was a false claim about existing code**, of the kind this session has
  produced twice before. The plan said the `ORIGIN` language region "falls out of
  the same walk that emits the escape codes"; `render.go` colours the section
  NAME and passes its text through deck-word highlighting, and nothing isolates
  "French". A new pass is needed and the plan now says so.
- **PQ-2 caught the plan contradicting the issue it belongs to.** Wrapping
  `#35`'s `OriginLanguage` — first-named-wins — makes exactly one language
  clickable, while `#30`'s own Spec records that clicking is worth having
  BECAUSE it dissolves ambiguity: `piano` names French and Italian and the user
  points at one. `#35`'s reader is refactored into a mentions producer with
  `OriginLanguage` as its first element, so there is one cut-and-mask rather than
  a second spelling of a rule that took four review findings to get right.
- **PQ-3** — `#32` is filed against the very seam D5 replaces. Recorded in D5a
  with what survives (`--play`) and what dissolves (the interactive path), so
  whichever lands first the other is re-read. D5b adds the stderr routing the
  plan had simply not mentioned: a diagnostic written past the screen corrupts
  the frame.
- **PQ-4** — `M1` shipped a viewport nothing could move, since the wheel is `M2`.
  Keyboard scrolling moves into `M1`, which is also what makes it independently
  usable rather than a layer waiting for its consumer.
- **PQ-5** — the mouse decoder is this issue's adversarial-input surface and had
  no strategy and no row. `#14` shipped an over-consuming escape decoder once
  already; a mouse sequence has three parameters and two final bytes, so it has
  more ways to be malformed.
- **PQ-6** — discoverability is now a decision rather than a deferral, and it
  turns on the tracking mode: hover needs `1003`, which streams an event per cell
  crossed, so `1000` is enabled and the mark is static — an underline attribute
  on the span's existing colour, not a seventh colour.

### 2026-08-29 — plan-quality round 2 (PQ-10, PQ-11)

**PQ-10 is the second appearance of `unbacked-existing-behavior`, and the rule is
the deliverable.** Three claims about the current tree were wrong in one round:
Ctrl-U/Ctrl-D are already bound (`KeyKillLine`, `KeyEOF`), PageUp/PageDown are
delimited but return `KeyUnknown` rather than being "already decoded", and
`Mention.Offset` would have indexed the MASKED text rather than `sec.Text`.

So the plan now carries a verified table of every claim it makes about
`cmd/define`, with `file:line`. That sweep caught a bug before it shipped: the
stage mask is `strings.ReplaceAll(text, stage, " ")`, which changes length, so
every `ORIGIN` region would have been drawn in the wrong column — measured as a
13-character shift on the `concrete` fixture. The mask becomes length-preserving,
which is a one-line change to `#35`.

This class has now cost five findings across four issues this session. The
pattern is always the same: a plan describes the tree from memory, and memory is
usually right, which is what makes the wrong ones expensive.

**PQ-11** — the underline belongs to the SCREEN, not `Render`: D6 promises
`define <word>` and `-raw` keep today's bytes, and an underline emitted by
`Render` would leak into both and contradict the row asserting rendering is
byte-identical. `sgr.go`'s `sgrState` is named as the existing machinery for
splicing an attribute into already-styled text.

### 2026-08-29 — M1.3 as built: three deltas worth recording

- **The buffer HONOURS `eraseLine`, and M1.1's "a bare CR carries no
  information" was too broad.** D5 promises the indicator feeds the buffer
  unchanged; `♫ playing 3×` is written and then taken back with `\r\x1b[K`, so a
  buffer that stripped the gesture would keep the indicator — and the exit
  transcript (D3) would then file a claim that playback happened, which is
  exactly the "ephemeral UI vs record" doctrine failing in the direction it
  exists to prevent. `screen.Write` therefore drops the OPEN line on an erase.
  Pinned by `TestScreenTakesBackAnErasedLine` and, end to end, by
  `TestEditorLoopWritesThroughAScreen`.

- **`liveScreen` is the type the plan did not name.** The buffer alone is
  invisible: a streamed answer arrives token by token and the indicator must
  appear while playback blocks for seconds, so a write has to repaint. `screen`
  stays pure and unit-tested with no terminal; `liveScreen` holds the tty, the
  row count (M1.4's one field) and the live edge, and is the only part that does
  IO. It also owns `Stop()`, because a frame painted after `restore` lands on the
  NORMAL screen over whatever was there before.

- **Three tests lost their subject with `cooked()` and were rewritten, not
  deleted.** `TestRawEditorPronPlaysOutsideTheCookedBlock` →
  `…PronReplaysThroughTheLoop` (the CDN order and play count survive; the
  cooked-block instrument does not). `TestSubmitClearsTheMenuBeforeOutput` →
  `TestSubmitLeavesNoMenuInTheFrame`, asserting the frame's menu argument rather
  than erase arithmetic that no longer exists. `TestRawLoopMessagePlacement` now
  drives a `screen` as stderr and asserts each message lands as its own line,
  which is the successor of "every message carries its own `\r\n`" — the CRLF
  writer left this path with D5, taking `assertCRLFTerminated` and
  `streamedAnswer` with it.

### 2026-08-29 — M1.4b: the wheel forced mouse tracking into M1

**Operator-reported after M1.4a: the wheel did not scroll — it walked history.**

The cause is a terminal convention this plan did not account for. In the
alternate screen a terminal translates the wheel into ARROW KEYS, which is what
makes `less` scroll without any mouse support at all. This program binds Up/Down
to the history walk, so every scroll recalled a word. The two are the SAME BYTES,
so no decoder, heuristic or timing rule can separate them: the only way to be
handed the gesture the user actually made is to ask the terminal to report the
mouse.

So `M2.2`'s enable/disable moves into `M1`, and the milestone boundary still
means what it did — `M1` is the screen layer, and scrolling is the screen's.
What follows from it:

- **`1000` + `1006`, on `rawSession` beside the alternate screen**, so one
  restore guarantee covers raw mode, the alt screen and tracking rather than
  three that each cover part of the exit paths. Disabled FIRST on the way out: a
  terminal left reporting the mouse types escape sequences into the next program
  the user runs, and unlike raw mode nothing about the shell looks wrong, so
  there is no `reset` reflex to save them.
- **The decoder answers only the WHEEL.** A press carries coordinates that mean
  nothing until `M2` has a region map, and a Key kind nothing reads is a kind
  that drifts — so a click stays `KeyUnknown`: consumed whole, inert, verified
  inert on a real pty. `M2.2`'s bounded-consumption fuzz target is written now,
  because the decoder is.
- **D7's cost is paid a milestone early.** Drag-select now belongs to the
  program, so copying needs Option or Shift. The plan already said this would
  happen; what changed is only that it happens in `M1`. It is stated in `/help`,
  which D7 asked for ("documented where a user meets it") and is the one screen
  that lists what the console understands.
- **A notch is three lines**, matching what the terminal itself means by one:
  left alone it sends three arrow keys per notch.

### 2026-08-29 — M1 done-when: two rows named tests that do not exist

Caught before the boundary review, and it is the same class the plan spent PQ-10
on: a claim about the tree written from intent rather than checked. Both claims
ARE pinned; the names were invented at planning time and the code chose others.

- Row 2's `TestScreenWriteContinuesAPartialLine` is a SUBTEST of
  `TestScreenWriteBuildsLines`.
- Row 3's `TestPTYAltScreenIsLeftOnExit` was never written. In-process the claim
  is `TestRestoreLeavesTheAlternateScreen`; on a real terminal it is the Fatal
  inside `TestPTYTranscriptIsPrintedOnExit`, which cannot find the transcript
  without first finding the teardown. A dedicated row would assert the same
  bytes twice.

Rows 3b, 8 and 9 are new: M1.4b's mouse tracking and wheel, and M1.3b's rule that
a prompt means the loop is waiting. All three are behaviour M1 ships that the
table did not cover, because two of them are answers to operator reports.

### 2026-08-29 — M1 boundary review: REWORK, and what the findings had in common

Two rounds, twelve findings, nine blocking. Fixed in this window; the classes
matter more than the sites, so they are recorded as classes.

**1. A mode enabled is a grammar accepted (BR-11 Critical, BR-5).** `M1.4b` turned
on mouse reporting with `1000`+`1006` and taught the decoder only the `1006`
form. A terminal that honours `1000` and ignores `1006` answers in X10 — `ESC[M`
plus three RAW bytes — and the CSI scan stopped at `M` as a final byte, leaving
the payload to be typed into the word being looked up: a left click typed `" !!"`.
This is `#14`'s family one encoding over, in the commit that enabled the mode.
`decodeX10Mouse` consumes six bytes or none, and `M2.6` now carries the rule:
**for every mode we enable, the decoder answers every encoding that mode can
reply in.**

**2. A frame is budgeted in DISPLAY ROWS, not lines (BR-12, BR-6).** `Paint`
charged one row per buffer line. A line wider than the terminal wraps, so the
frame was too tall, so the terminal scrolled — moving every row the app believes
it placed, which is the exact property the alternate screen was taken for and
`M2`'s `RegionAt` depends on. Two routine ways in: narrow the window (buffer lines
keep their wrapping, by decision) or type a line longer than the terminal is
wide. `screen.cols` was declared in this plan and inert in the code; it is real
now. The prompt and each menu row are charged their true height, buffer lines are
CLIPPED to the width at paint time — so the transcript and `M2`'s click map keep
the whole text. Measured on a real pty at 40 columns: every row fits, the
transcript does not.

**3. A pin that cannot fail is worse than no pin (BR-13, BR-4, and the `eraseLine`
Minor).** `TestRestoreLeavesTheAlternateScreen` built a session with a nil file,
so every enter and every leave returned at the same guard and the assertion
checked a field nothing had set — deleting BOTH leaves from `restore()` left the
suite green. The fix is structural: `rawSession` writes its mode sequences to an
`io.Writer`, which is also where they belong (they change the screen the frames
are drawn on, not the stdin handle they were going to). The restore protocol —
mouse off, then alt screen, then raw — is now asserted in process, bytes and
order, and verified falsifiable.

**4. Prose is swept by CLASS or not at all (BR-14).** `M1.6` rewrote the atlas
sections it was looking at; five other sites still said the raw loop wraps stdout
in `crlfWriter`, including a TEST — `TestHighlightingNestsInsideCRLFTranslation`
— that built a composition no production path builds any more. A test asserting a
dead composition reads as coverage. It is now
`TestHighlightingSeesLogicalTextAndTheScreenPlacesIt`, over the screen.

**5. ARCH-CONSTRAINTS: the envelope, stated (BR-16, and the buffer-growth Minor).**
A full-screen program needs one and this plan declared none.

- **Repaint:** a write paints at most every 16 ms (`paintInterval`), with a
  TRAILING flush so a held frame goes out whether or not another write follows.
  That trailing half is not an optimisation — the `♫ playing 3×` indicator is
  written and then playback blocks for seconds, so a throttle waiting for the
  next write would hide it for the whole recording. Draw, Page, Scroll and Stop
  paint unconditionally. Bound: unpainted text is never more than one interval's
  worth.
- **Buffer:** `screen.lines` holds the whole session and is printed at exit, by
  design (D3). No cap, deliberately: a cap would silently truncate the record the
  transcript exists to be, and the envelope is a human session — a very long one
  is a few thousand lines, ~1 MB. If a session ever needs to outlive that, it
  needs a file, not a smaller buffer.

**6. Tests observe across goroutines with a lock (BR-15).** Two new tests polled
fields written by another goroutine; `-race` failed where the base commit was
clean. `recordDisplay` is mutex-guarded with reader methods, and the resize
watcher's measurements are reported over a channel. `go test -race ./cmd/define/`
is green.

### 2026-08-29 — M1 boundary review round 3: four more, and one of them was mine

**The Critical was a verification claim, not a defect (BR-23).** `go test ./...`
was red at HEAD on two plan-table guards and the Log said green — because I ran
the suite, then edited this plan's tables, then committed. The guards parse the
THIRD cell as status against a controlled vocabulary; a "Kind" column inserted
before it made every row unparseable, and a `Render` row calling itself
`modified` described work M2 has not done. Both are exactly what those guards
exist to catch. **The rule: prose in this repo is CODE to a guard — re-run the
suite after editing a plan, not before.**

**The residual on BR-12: a sentinel leaked into arithmetic.** The display-row
budget was right, but `terminalWidth` returns 0 for a terminal under 20 columns —
that 0 means "do not wrap", a POLICY answer — and the screen was reading it as a
column count. `terminalSize`/`terminalCols` answer the other question and cannot
return a sentinel; the loop derives the wrap policy from the true shape.

**BR-26 — one owner, measuring CELLS.** Three counters disagreed: `visibleLen`
counted runes, the menu's cursor-up counted entries, and the width probe could
answer 0. Runes are wrong in both directions for THIS program: `bänˈZHo͝or`
carries a combining double breve (10 runes, 9 columns — text that fits gets cut)
and a Japanese entry is full-width (10 runes, 20 columns — the frame is twice as
tall as measured and the terminal scrolls). `visibleCells` + `cellWidth` are now
the single owner, read by every wrap, budget and clip; the cursor walks back by
the rows the terminal actually moved.

**BR-24 — a pty row is a conformance check, not a pin.** `replRaw` has no
in-process caller, so the exit sequence rested entirely on rows that skip
wherever no pty exists — including inside the review. `handBack` is now a named
function over two small interfaces: stop painting, restore, print the session,
in that order, each step wrong in a different way alone. `onceHandBack` carries
the once-only property. Both are pinned in process, and the cursor-escape fix
(BR-20) with them — verified falsifiable.

**BR-25 — the atlas lagged its own milestone.** The display-row budget, the clip,
the cell-width owner, the throttle and its trailing flush, the X10 fallback and
`handBack` are all in "The screen" now. The pattern across rounds 2 and 3 is one
rule: **the sweep is by CLASS — every site that states the fact — and the atlas
is one of the sites, not a follow-up.**

### 2026-08-29 — M1 review round 4: the guard reads GIT, and a rune is not a column

**Why BR-23 recurred, and it is not "I forgot to re-run".** I did re-run
`go test ./...` — before committing, and the plan-table guards compare the plan
against `git diff base..HEAD`. An uncommitted `render.go` edit is not in that
window, so the guard passed; the same tree one commit later fails. **The rule:
the plan-table guards are answered by the COMMIT, so the suite runs after
committing, not before.** Recorded in `workshop/lessons.md`, because nothing in
the guard's own message says so.

The M2 `Render` row now reads `modified`, and the reason is the convention the
review asked for: the status column describes THIS WINDOW's diff, not the
milestone's intent. M1's `visibleLen`→`visibleCells` rename swept `Render`'s
body; M2.1's signature change is still ahead.

**The one-owner rule was implemented at half its sites.** `visibleCells` counted
cells while `clipVisible` cut by runes and `truncate` cut by runes, so a
100-rune CJK line still painted 19 display rows into a 10-row terminal. Cutting
is measuring: `clipVisible` counts cells and never splits a two-cell rune,
`truncate` delegates to it, and the frame's row accounting is now ONE pass that
both budgets the buffer and says where the cursor walks back to — the second
summation had omitted the prompt's own height, which reprinted it over the menu.

**The envelope's column half, stated:** below 20 columns `opt.width` turns
wrapping OFF (an entry cannot be broken that narrowly and stay readable) while
the screen keeps the true column count from `terminalCols` and still fits the
frame. The two answers differ on purpose, and only the policy one may be zero.

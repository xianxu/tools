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

## Milestone M1 — the screen owns the terminal

### Core concepts

#### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `screen` | `cmd/define/screen.go` | new |
| `screen.Write` | `cmd/define/screen.go` | new |
| `screen.Frame` | `cmd/define/screen.go` | new |
| `screen.Scroll` | `cmd/define/screen.go` | new |
| `crlfWriter` | `cmd/define/crlf.go` | unchanged — still used by `--play`, which keeps its own drawing |

- **`screen`** — a line buffer plus a viewport: `lines []string`, `offset int`, `rows, cols int`.
  - **Relationships:** 1:1 with an interactive session; owns every line it displays.
  - **`Write` is the seam** (D5): it splits incoming bytes on `\n`, appending to `lines` and continuing the last partial line, so a streamed reply that arrives in fragments lands as text rather than as frames. It performs no terminal IO.
  - **`Frame` is PURE**: `(lines, offset, rows, cols) -> []string`, the rows to paint. Unit-testable with no terminal, which is what keeps the scroll arithmetic out of a pty test.
  - **DRY rationale:** replaces `crlfWriter` on this path and subsumes `menuDrawn`'s erase arithmetic, whose documented known limit — "if the menu does not fit below the cursor the terminal scrolls and the cursor-up count lands a row off" — cannot occur when the app places every row.
  - **Future extensions:** `M2`'s region map hangs off the same per-line record.

#### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `screen.Paint` | `cmd/define/screen.go` | new | the terminal |
| `enterAlt` / `leaveAlt` | `cmd/define/rawterm.go` | new | `\x1b[?1049h/l` |
| `watchResize` | `cmd/define/rawterm.go` | new | SIGWINCH |
| `replRaw` / `runEditor` | `cmd/define/replraw.go` | modified — `cooked` deleted (D4) | the terminal |

- **`watchResize`** — there is NO resize handling today; width is read once at flag parse (`main.go`). A full-screen program must handle it or the frame is wrong after the first drag.
  - **Injected into:** the editor loop as a channel, beside `keys`, so the loop's select grows one case rather than the screen learning about signals.

**ARCH-MOCK.** The external dependency is the terminal, and the repo already has its stateful double: the `creack/pty` conformance harness (`startDefine`, `watch`). `M1` adds rows there — alt screen entered and left, a resize redrawing, the transcript appearing after exit — and the pure `Frame` tests carry the arithmetic.

### Tasks

- [ ] **M1.1 — `screen` as a pure model.** `Write`, `Frame`, `Scroll`, plus rows/cols. Table tests: a partial write continues the last line; a write containing `\n\n` appends an empty line; `Frame` clamps the offset at both ends; a viewport taller than the buffer pads rather than repeating. No terminal.
- [ ] **M1.2 — `Paint` and the alt screen.** `enterAlt`/`leaveAlt` on `rawSession`, so a Ctrl-C or a panic leaves the terminal restored — the same obligation `enterRaw` already carries and the reason this belongs there rather than in `screen`.
- [ ] **M1.3 — the editor draws through the screen**, `cooked` deleted (D4). Every current writer keeps writing; only the destination changes.
- [ ] **M1.4a — KEYS that move the viewport.** Without this `M1` ships a scroll model nothing exercises and a user cannot reach: the wheel is `M2`, and no key scrolls today. PageUp/PageDown and Ctrl-U/Ctrl-D, decoded by the existing CSI scanner. This is what makes `M1` independently usable rather than a layer waiting for `M2`.
- [ ] **M1.4 — resize.** SIGWINCH → re-measure → repaint. The one thing that cannot be unit-tested is the signal, so the pty row drives a real `TIOCSWINSZ`.
- [ ] **M1.5 — the transcript on exit** (D3), and the pty row that it survives.
- [ ] **M1.6 — docs**: the atlas's raw-mode section, which currently explains the cooked/raw dance that D4 removes. That prose goes false, so it is rewritten rather than appended to.

### M1 Done-when

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | the viewport arithmetic is right | `TestScreenFrame` | `Frame` stops clamping the offset |
| 2 | a streamed fragment lands as text, not a frame | `TestScreenWriteContinuesAPartialLine` | `Write` splits on every call boundary |
| 3 | the terminal is restored on every exit | `TestPTYAltScreenIsLeftOnExit` | `leaveAlt` is dropped from the restore path |
| 4 | the viewport can be moved by a user | `TestEditorPageDownScrolls` | the key case is removed from the select |
| 5 | a resize repaints | `TestPTYResizeRepaints` | the SIGWINCH case is removed from the select |
| 6 | the transcript survives exit | `TestPTYTranscriptIsPrintedOnExit` | D3's loop is removed |
| 7 | the one-shot and piped paths are untouched | the existing suite, unchanged | any of them starts entering the alt screen |

---

## Milestone M2 — the clicks

### Core concepts

| Name | Lives in | Status |
|------|----------|--------|
| `Region` | `cmd/define/render.go` | new |
| `Render` | `cmd/define/render.go` | modified — also returns regions |
| `decodeMouse` | `cmd/define/key.go` | new |
| `screen.RegionAt` | `cmd/define/screen.go` | modified |

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
- [ ] **M2.2 — `decodeMouse` + tracking enable/disable**, paired with the alt screen so a crash cannot leave tracking on.
- [ ] **M2.3 — hit test**: `screen.RegionAt(row, col)`, which is a lookup in the per-line region list. Pure.
- [ ] **M2.4 — the two actions.** Headword → replay. `ORIGIN` language → `/pron <that language>`, which after `#35` is a call into `OriginLanguage`'s map rather than new inference.
- [ ] **M2.5 — discoverability, STATIC rather than on hover** — and the tracking mode is the reason. Hover needs `1003` (any-event tracking), which streams an event for every cell the pointer crosses, so the loop would wake constantly to redraw an underline. `1000` (button press only) is what this issue enables, and with it the app never learns where the pointer is. So a clickable span is marked in the FRAME: the palette (`newPalette`) already spends `head`, `ipa`, `pos`, `num`, `ex`, `sect` and bold-green for deck words, so the mark is an ATTRIBUTE — underline — added to the span's existing colour rather than a seventh colour competing with them.
- [ ] **M2.6 — degrade.** A terminal that reports no mouse must behave exactly as `M1` does.

### M2 Done-when

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | clicking the headword plays it | `TestClickOnHeadwordReplays` | `RegionAt` stops matching the headword span |
| 2 | clicking `ORIGIN French` plays French | `TestClickOnOriginLanguagePlaysIt` | the region's language is dropped |
| 3 | rendering is byte-identical | `TestRenderOutputUnchangedByRegions` | `Render` alters a byte while collecting |
| 4 | a clickable span is visibly clickable before it is clicked | `TestRenderMarksClickableSpans` | the underline attribute is dropped |
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

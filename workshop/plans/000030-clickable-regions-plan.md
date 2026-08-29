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
- [ ] **M1.4 — resize.** SIGWINCH → re-measure → repaint. The one thing that cannot be unit-tested is the signal, so the pty row drives a real `TIOCSWINSZ`.
- [ ] **M1.5 — the transcript on exit** (D3), and the pty row that it survives.
- [ ] **M1.6 — docs**: the atlas's raw-mode section, which currently explains the cooked/raw dance that D4 removes. That prose goes false, so it is rewritten rather than appended to.

### M1 Done-when

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | the viewport arithmetic is right | `TestScreenFrame` | `Frame` stops clamping the offset |
| 2 | a streamed fragment lands as text, not a frame | `TestScreenWriteContinuesAPartialLine` | `Write` splits on every call boundary |
| 3 | the terminal is restored on every exit | `TestPTYAltScreenIsLeftOnExit` | `leaveAlt` is dropped from the restore path |
| 4 | a resize repaints | `TestPTYResizeRepaints` | the SIGWINCH case is removed from the select |
| 5 | the transcript survives exit | `TestPTYTranscriptIsPrintedOnExit` | D3's loop is removed |
| 6 | the one-shot and piped paths are untouched | the existing suite, unchanged | any of them starts entering the alt screen |

---

## Milestone M2 — the clicks

### Core concepts

| Name | Lives in | Status |
|------|----------|--------|
| `Region` | `cmd/define/render.go` | new |
| `Render` | `cmd/define/render.go` | modified — also returns regions |
| `decodeMouse` | `cmd/define/key.go` | new |
| `screen.RegionAt` | `cmd/define/screen.go` | modified |

- **`Region`** — `{Kind, Text, Line, Col, Width}`: what a span of rendered text OFFERS.
  - `Render` already knows where the headword and the `ORIGIN` section are — it colours them — so the regions fall out of the same walk that emits the escape codes. It must report positions AFTER wrapping, since wrapping is what decides a column.
  - **Two kinds to start:** `RegionHeadword` and `RegionOriginLanguage`. A third is a row, which is the Done-when about a registry.

- **`decodeMouse`** — `ESC[<b;x;yM/m` → button, column, row. `key.go` already scans to the CSI final byte (`#14`), so this is a decoder for a sequence the reader already delimits correctly, not new parsing.

### Tasks

- [ ] **M2.1 — `Render` emits regions.** The signature change is the risk: every caller and every golden test touches it. Keep the string identical — assert byte-equality against the current output over the whole corpus, so a regions change cannot silently alter what is drawn.
- [ ] **M2.2 — `decodeMouse` + tracking enable/disable**, paired with the alt screen so a crash cannot leave tracking on.
- [ ] **M2.3 — hit test**: `screen.RegionAt(row, col)`, which is a lookup in the per-line region list. Pure.
- [ ] **M2.4 — the two actions.** Headword → replay. `ORIGIN` language → `/pron <that language>`, which after `#35` is a call into `OriginLanguage`'s map rather than new inference.
- [ ] **M2.5 — discoverability.** A region a reader cannot see is not an affordance: underline on hover, or a footer line. Decide with a measurement of what the palette already spends.
- [ ] **M2.6 — degrade.** A terminal that reports no mouse must behave exactly as `M1` does.

### M2 Done-when

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | clicking the headword plays it | `TestClickOnHeadwordReplays` | `RegionAt` stops matching the headword span |
| 2 | clicking `ORIGIN French` plays French | `TestClickOnOriginLanguagePlaysIt` | the region's language is dropped |
| 3 | rendering is byte-identical | `TestRenderOutputUnchangedByRegions` | `Render` alters a byte while collecting |
| 4 | a mouse-less terminal is unaffected | `TestPTYWithoutMouseBehavesAsBefore` | the enable is emitted unconditionally |
| 5 | regions are one registry, not two special cases | `TestEveryRegionKindIsActionable` | a kind is added with no action |
| 6 | tracking is disabled on exit | `TestPTYMouseTrackingIsLeftOnExit` | the disable is dropped |

---

## Verification before close

```bash
go test ./...
go test -tags conformance ./cmd/define/    # unsandboxed; the pty rows are here
```

Then, on a real terminal: look a word up, click the headword, click `ORIGIN French`, scroll with the wheel, resize the window, quit and confirm the transcript is in the scrollback.

**Close:** two milestones, each with `sdlc milestone-close`; one `sdlc close` and one publish.

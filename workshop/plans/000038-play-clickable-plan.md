# Clickable Words in the Review Loop Implementation Plan (`#38`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Click the word `--play` is asking about to hear it, and click anything in a revealed definition exactly as the interactive loop already allows.

**Architecture:** `--play` stops drawing its own frames and writes into `#30`'s `liveScreen`. It gains the alternate screen, a click map, scrolling and the exit transcript from the machinery that already exists, and the click action becomes a shared function rather than a second switch. The cooked/raw dance around playback is DELETED, exactly as `#30` D4 deleted it from the editor — which is also what stops it tearing the new screen down.

**Tech Stack:** Go 1.26. Everything this needs was built by `#30`; no new dependency and no new mechanism.

---

## Decisions

**D1 — `--play` joins the screen rather than tracking a cursor, and the operator chose this.** The alternative was mouse reporting with a remembered cursor position, which `#30` D1 rejected on measurement: the terminal scrolls whenever the loop writes, so the app tracks an offset it never observes and a click after any scroll lands wrong. The screen makes the mapping exact by construction.

**D2 — the cooked/raw dance around playback is DELETED, and that is the milestone's largest simplification.** `play_loop.go:174-198` calls `raw.sess.restore()` before playback and `enterRaw` after, so the indicator prints in cooked mode (`#16`). Two facts make that fatal here and unnecessary anyway:

- **Fatal:** since `#30`, `restore()` also leaves the alternate screen and mouse reporting (`rawterm.go:50-61`), and `enterRaw` returns a session with `alt=false, mouse=false` (`rawterm.go:42-48`). So the first reveal would tear the screen down and never restore it — the sitting would fall back to the normal buffer and clicks would stop, permanently, with nothing said.
- **Unnecessary:** with the screen owning line placement, no output depends on the line discipline. This is `#30` D4 verbatim, applied to the other loop: *"Inside an app-owned screen there is nothing to flap."* Staying raw is also what lets Ctrl-C reach the key reader during playback rather than the line discipline swallowing it.

Deleting it removes the `lost the terminal after playback` path (`play_loop.go:182-197`) with it, as D4 removed `lostTerminal` from the editor.

**D3 — `draw` DOES change, and the seam already exists.** An earlier draft claimed `draw` was untouched because it takes an `io.Writer`. False: writing a region map is not a plain write. `writeRendered` (`main.go`) is the seam — a writer that can hold a click map gets one, everything else gets bytes — so `draw` calls it for the reveal and writes the prompt word's own region. The `io.Writer` signature still means a test drives `draw` with no terminal.

**D4 — the ACTION is shared, and it is `playAnnounced` rather than `replayInPlace`.** The editor's click reaches `replayInPlace`, which reads a `session` `--play` does not have; `--play`'s own playback is `playAnnounced` with `defaultIndicator(opt)` (`play_loop.go:178-181`). The shared thing beneath both is `playAnnounced`, so `playRegion` builds the utterance from a region and calls it, taking the INDICATOR as a parameter — the editor's erasable one, `--play`'s record-shaped one. One switch on `RegionKind`, two callers, which is what `#30` Done-when 7 requires.

**D5 — the visual result stays close.** `draw` (`play_loop.go:283-291`) is APPEND-ONLY: it writes `"\n%s\n"` per question and never clears, so a sitting stacks up and the terminal scrolls. `liveScreen` has the same shape — `Write` appends to a line buffer, `Paint` shows its tail — so a sitting reads as it does today and gains scrolling.

**D6 — the scrollback trade is inherited deliberately.** The alternate screen means a sitting is not in the terminal's scrollback while it runs; it is printed back on exit (`#30` D3, `handBack`). Smaller loss here than for lookups — a review sitting is a transaction you finish, not a reference you scroll back to mid-flight. The operator was asked and chose it.

**D7 — the reveal's regions come from `Render`, and `play` must never see them.** `play` is pure and does not import `main`; `Region` lives in `main`. `play.NewRecall(key, rendered)` (`play_loop.go:265`) stores only the rendered string, so the loop keeps its own `word -> []Region` map built where the entries are rendered (`play_loop.go:262`, which already discards the regions).

**D8 — a click is a REPLAY and never an answer.** Hearing the word is what `y`/`n` are answering *about*. A click that recorded a review would corrupt the schedule silently, which is the worst kind of bug here: the damage is to data the learner cannot see.

**D10 — adopting `#30`'s loop shape means adopting its CARRIERS, not re-deriving them.** Three exist for exactly this and an earlier draft reinvented all three:

- **`console`** (`replraw.go:94`) already bundles view, resizes, finish, stdout and stderr — it exists because five of `runEditor`'s parameters were one concept, and two adjacent `io.Writer`s swap silently. `playSession` takes it rather than "a display and the region map".
- **`onceHandBack`** (`replraw.go:140`), not bare `handBack` (`replraw.go:170`). `playSession` returns from several paths and `runPlay` also has `defer sess.restore()` (`play_loop.go:53`), so a bare `handBack` prints the transcript twice — which is precisely why the once-wrapper exists.
- **`watchResize`** (`rawterm.go:230`) is the SIGWINCH carrier, which is the half of PQ-3 the first fix missed (T6).

The rule: a second loop adopting the first loop's shape adopts its carriers, or the divergence this issue exists to close reopens one field at a time.

**D11 — `playRegion` carries the audio-off guard, because `playAnnounced` does not.** `replayInPlace` guards above it (`replraw.go:534`, `opt.noAudio || opt.times <= 0` → "nothing to replay") and `--play` guards above it too (`play_loop.go:161`, `if !opt.noAudio && opt.times > 0`). Converging on `playAnnounced` alone would drop BOTH guards, so `-no-audio` would make a click attempt playback and say nothing. The guard moves INTO `playRegion` — one place, both callers — which is the same consolidation the switch itself is.

**D9 — `--play`'s `crlfWriter` goes, which is `#32`'s remaining half.** `#30` D5a: *"`#32` keeps its `--play` half, which continues to draw its own frames through `crlfWriter`."* Once `--play` writes into the screen, the screen owns line placement and the second writer is a second owner. Re-read `#32` when this lands.

---

## What this plan asserts about the existing tree, verified

`#30` spent a finding on unchecked claims about the tree (PQ-10), so every claim carries a `file:line` and was read.

| claim | verified at | status |
|---|---|---|
| `draw` is append-only and never clears | `play_loop.go:283-291` | true — `fmt.Fprintf(w, "\n%s\n", q.Prompt())` |
| a recall form's prompt IS the word | `play/recall.go:29` | true — `Prompt() string { return r.word }` |
| `--play` wraps stdout/stderr in `crlfWriter` | `play_loop.go:64-65` | true |
| entries are pre-rendered and their regions discarded | `play_loop.go:262-265` | true |
| the playback dance restores and re-enters raw mode | `play_loop.go:174-198` | true — and `restore` now also leaves alt + mouse (D2) |
| `restore` leaves mouse and the alt screen | `rawterm.go:50-61` | true |
| `enterRaw` returns a session with `alt`/`mouse` false | `rawterm.go:42-48` | true — so the dance cannot be patched, only removed |
| `--play` plays through `playAnnounced` + `defaultIndicator` | `play_loop.go:178-181` | true — NOT `replayInPlace` (D4) |
| the editor's click registry closes over its `session` | `replraw.go:264` | true — so sharing it means lifting it |
| `writeRendered` is the region seam | `main.go:801` | true |
| `handBack` is the exit sequence | `replraw.go:170` | true |
| the loop's viewport keys are `KeyPageUp`/`KeyPageDown`/wheel | `replraw.go:366-380` | true — `--play` needs its own cases (T6) |

---

## Core concepts

### Pure entities

| Name | Lives in | Status | Kind |
|------|----------|--------|------|
| `playRegions` | `cmd/define/play_loop.go` | new | PURE — the `word -> []Region` map, because `play` cannot hold a `main` type (D7) |
| `promptRegionFor` | `cmd/define/play_loop.go` | new | PURE — the region for the word `draw` is about to write |

- **`playRegions`** — questions are pre-rendered before the sitting starts, so their regions are known then and needed later.
  - **Relationships:** 1:1 with the question queue, keyed by `play.Question.Word()`.
  - **DRY rationale:** avoids re-rendering an entry to recover its regions, and avoids teaching `play` about `main`.
  - **Future extensions:** a second review form (`#7`, `#12`, `#13`) renders differently; keyed by word rather than by form, so it widens without changing shape.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `playRegion` | `cmd/define/replraw.go` | new | `playAnnounced` — the one switch on `RegionKind`, shared by both loops (D4) |
| `runPlay` | `cmd/define/play_loop.go` | modified | `liveScreen` — replaces both `crlfWriter`s (D9) |
| `playSession` | `cmd/define/play_loop.go` | modified | takes `console` (`replraw.go:94`) and the region map — D10 |
| `draw` | `cmd/define/play_loop.go` | modified | `writeRendered` for the reveal (D3) |

- **`playRegion`** — given a region, what to play it against and an indicator, perform the playback.
  - **NOT pure**, and the label matters: it reaches `playAnnounced`, which fetches audio and writes. Its test uses the player and CDN fakes the repo already has, not a colocated unit test.
  - **Injected into:** both loops. The editor passes its erasable indicator; `--play` passes `defaultIndicator(opt)`.
  - **DRY rationale:** `#30` Done-when 7 — "one registry, not two special cases". A second switch breaks that the day a third `RegionKind` lands.

**ARCH-MOCK.** The terminal's double already exists: `playSession` is split from `runPlay` precisely so a test can drive it with a scripted key channel and no terminal (`play_loop.go:83-85`). The pty rows cover what only a terminal answers.

**ARCH-CONSTRAINTS.** `#30`'s envelope, inherited unchanged: paint at most every 16 ms with a trailing flush, uncapped line buffer. A sitting is bounded by `-count` (default 20) and writes a word plus at most one definition each, so the buffer stays the same order as an interactive session, which that envelope already covers. No new budget claimed.

---

## Tasks

Plain checkboxes, not `Mx` tags: this is single-pass work with ONE boundary, and AGENTS.md §3 says an `Mx` tag commits to its own `milestone-close`.

- [ ] **T1 — lift the click registry.** `playRegion(ctx, d, opt, r Region, entry string, ind indicator, stdout, stderr)`: the audio-off guard (D11), then the switch on `RegionKind`, building the utterance and calling `playAnnounced`. The editor's `clicked` (`replraw.go:264`) becomes a call to it. NO behaviour change — `TestEveryRegionKindIsActionable`, `TestClickOnHeadwordReplays` and `TestClickOnOriginLanguagePlaysIt` pass untouched, which is what proves the lift was a lift.
- [ ] **T2 — delete the playback dance** (D2). `play_loop.go:174-198` loses `restore`/`enterRaw` and the `lost the terminal after playback` path. Pinned by the existing `TestPTYCtrlCDuringPlaybackExitsPromptly` and `TestRawEditorPronReplaysThroughTheLoop`'s sibling for `--play`; a new row asserts the alternate screen is STILL up after a reveal, which is the regression this deletion prevents.
- [ ] **T3 — `--play` writes into a `liveScreen`.** `enterAlt`, `enterMouse`, `newLiveScreen`, `handBack` on exit, replacing both `crlfWriter`s (D9).
- [ ] **T4 — the prompt word is a region.** `draw` is append-only, so the word lands on the line about to be written: one `RegionHeadword` at column 0, width `visibleCells(word)`.
- [ ] **T5 — the revealed definition carries its regions** (D7, operator's choice), through `writeRendered` (D3).
- [ ] **T6 — the viewport, all three parts.** The alternate screen has NO scrollback, so without this a sitting cannot be scrolled at all — worse than today, where the terminal keeps it.
      - **Scroll:** PageUp/PageDown and the wheel, the same cases `replraw.go:366-380` already has.
      - **Resize:** `watchResize` (`rawterm.go:230`) into the loop's select, as `runEditor` does — the first fix for this finding covered scrolling and left SIGWINCH out, so a window change mid-sitting would have left the frame wrong.
      - Neither a scroll nor a resize reaches `play.Apply`.
- [ ] **T7 — the click acts and does not answer** (D8).
- [ ] **T8 — docs**: `cmd/define/README.md`'s review-loop section, and `atlas/define.md`'s clickable-regions section, which names the interactive loop as the only consumer.

## Done when

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | the word being asked about is clickable | `TestPlayClickOnThePromptWordPlaysIt` | the region is not written with the word |
| 2 | a click NEVER answers | `TestPlayClickIsNotAnAnswer` — no review recorded, `Right`/`Wrong` unchanged, the question still current | the click reaches `play.Apply` |
| 3 | a revealed definition is clickable like anywhere else | `TestPlayClickOnARevealedHeadword` | the reveal is written without its regions |
| 4 | one registry, both loops | `TestEveryRegionKindIsActionable` extended to drive `playRegion` directly | a kind acts in one loop and not the other |
| 4b | **`-no-audio` says so rather than playing silence** | `TestPlayRegionRespectsAudioOff` | the guard is left in the callers, so converging on `playAnnounced` drops it |
| 6b | **a resize repaints mid-sitting** | `TestPTYPlayResizeRepaints` | `watchResize` is not wired into the loop's select |
| 5 | **playback does not tear the screen down** | `TestPTYPlayKeepsTheAlternateScreenAcrossAReveal` | the restore/re-enter dance comes back |
| 6 | a sitting can be scrolled | `TestPlayPageKeysScroll` | the viewport cases are dropped, leaving no scrollback at all |
| 7 | the terminal is handed back | the existing `--play` pty rows, unchanged | `handBack` is dropped from an exit path |
| 8 | a mouse-less terminal is unaffected | the existing `--play` pty rows, unchanged | the loop needs a click to proceed |

---

## Verification before close

```bash
go test ./... && go test ./cmd/define/ -race
go test -tags conformance ./cmd/define/    # unsandboxed; the pty rows
```

Then, on a real terminal: `define --play`, click the word, hear it; press `n`, confirm the screen is STILL the alternate one, click the headword and the `ORIGIN` language in the revealed entry; PageUp to a word from earlier in the sitting and click it; quit and confirm the transcript is in the scrollback.

**Close:** one boundary, one `sdlc close`, one publish.

## Revisions

### 2026-08-30 — plan-quality round 1 (PQ-1…PQ-4 and three Minors)

- **PQ-1 was a Critical the first draft would have shipped**, and it is the reason D2 is now the plan's largest task rather than absent. `--play` restores raw mode around playback; since `#30` folded the alternate screen and mouse reporting into `rawSession`, `restore` tears both down and `enterRaw` returns a session that has neither. The first reveal would have dropped the sitting back to the normal buffer and killed clicks for the rest of the run, silently. The fix is `#30` D4's own deletion applied to the other loop — which is smaller than the patch would have been.
- **PQ-2** — the first draft claimed `draw` does not change because it takes an `io.Writer`. Writing a click map is not a plain write; `writeRendered` is the seam and `draw` calls it (D3).
- **PQ-3** — no task wired scrolling. The alternate screen has no scrollback, so adopting it without the viewport keys makes a sitting LESS navigable than today, where the terminal keeps the history. T6.
- **PQ-4** — `--play` plays through `playAnnounced`, not `replayInPlace`, so a shared action built on the latter would have been a second caller in a loop with no `session`. The shared thing is `playAnnounced`, and the indicator is the parameter that differs (D4).
- **Minors:** `playRegion` is an INTEGRATION point, not a PURE entity — it plays audio, so its test needs the fakes; the issue's `## Plan` now carries the tickable steps rather than "decide A vs B"; and the `M1` tag is gone, since single-pass work with one boundary takes plain checkboxes (AGENTS.md §3).

### 2026-08-30 — plan-quality round 2 (PQ-3 again, PQ-8, PQ-9)

- **PQ-9 is the one worth stating as a rule**: *adopting `#30`'s loop shape means
  adopting its carriers.* An earlier draft reinvented three — `console`,
  `onceHandBack`, `watchResize` — each of which exists because the editor loop
  hit the exact problem `--play` is about to. `onceHandBack` is the sharpest:
  `playSession` returns from several paths and `runPlay` also defers a restore,
  so a bare `handBack` prints the transcript twice. Now D10.
- **PQ-8** — `playAnnounced` sits BELOW the audio-off guard in both loops
  (`replraw.go:534`, `play_loop.go:161`), so converging on it would have made a
  click under `-no-audio` attempt playback and say nothing. The guard moves into
  `playRegion`: one place, both callers, which is the same consolidation the
  switch is. Now D11.
- **PQ-3 a second time** — round 1's fix covered scrolling and left SIGWINCH out,
  which is half a viewport. T6 now names scroll, wheel AND resize, and the
  Done-when table gains a row for each.
- **Minor: four line anchors pointed at the wrong lines.** Every claim was
  substantively true, which is exactly why it matters — a table that exists to be
  audited sends the auditor to code that does not say what the row claims. All
  re-measured.

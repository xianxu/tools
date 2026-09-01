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

**D11 — the audio-off guard becomes ONE NAMED PREDICATE, applied inside `playAnnounced`.** An earlier draft put it in `playRegion`, which was a fifth hand-copy of a class of four — the instance again, not the rule. Measured, in two spellings:

```
main.go:783        !opt.noAudio && opt.times > 0     decides lookupOutcome.play
play_loop.go:161   !opt.noAudio && opt.times > 0     skips the reveal's playback
repl.go:312         opt.noAudio || opt.times <= 0    says "nothing to replay"
replraw.go:534      opt.noAudio || opt.times <= 0    says "nothing to replay"
```

And `playAnnounced`'s own doc comment (`main.go:827-830`) already lists this as a divergence it was built to end: *"Both entry paths ran their own copy and had diverged three ways — which terminal they gated on, **whether the audio-off guard applied**, and the duplicated literal."* It ended two of the three; the guard stayed ABOVE it in every caller, which is why a fifth caller could silently sit below it.

So `options.playsAudio()` is the predicate, `playAnnounced` applies it itself — being below it becomes impossible — and the four sites ask the predicate rather than re-deriving it. Callers keep their own MESSAGES; only the condition is shared.

**D9 — `--play`'s `crlfWriter` goes, which is `#32`'s remaining half.** `#30` D5a: *"`#32` keeps its `--play` half, which continues to draw its own frames through `crlfWriter`."* Once `--play` writes into the screen, the screen owns line placement and the second writer is a second owner. Re-read `#32` when this lands.

---

## What this plan asserts about the existing tree, verified

`#30` spent a finding on unchecked claims about the tree (PQ-10), so every claim carries a `file:line` and was read.

| claim | verified at | status |
|---|---|---|
| ~~`draw` is append-only and never clears~~ | `play_loop.go:283-291` | true when written; **no longer the tree** — `#41` T4 deleted `draw`, and the question is written to the buffer once per question rather than once per frame |
| a recall form's prompt IS the word | `play/recall.go:29` | true — `Prompt() string { return r.word }` |
| ~~`--play` wraps stdout/stderr in `crlfWriter`~~ | `play_loop.go:64-65` | true when written; **no longer the tree** — `#41` T3 landed D9 |
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
| `clickable` | `cmd/define/play_loop.go` | new | PURE — one entry's rendered TEXT beside the regions in it, because their coordinates are relative to that text and the loop writes something larger. Named `playRegions` and specified as a bare `word -> []Region` map while the plan still assumed `--play` appended; the text has to travel with them (D7) |
| `sittingDeck.marksIn` | `cmd/define/play_loop.go` | new | PURE — the regions for a word, moved into the coordinates of the text about to be written, by LOCATING the render inside a form's reveal rather than counting from a formula that would duplicate a layout the form owns |
| `wrapMovedRegions` | `cmd/define/playbar.go` | new | PURE — re-points regions across a wrap: a line the wrap does not break keeps its columns and moves down, a line it breaks loses its regions. Called only by `liveScreen.WriteRegions`, which holds the width (`#41` BR-25) |

- **`clickable`** — questions are pre-rendered before the sitting starts, so their regions are known then and needed later. **`promptRegionFor` was never built**: the prompt word's region is one struct literal at the write site (line 1, column 0, `visibleCells(word)`), and a named constructor for a three-field literal used once would be indirection rather than reuse.
  - **Relationships:** 1:1 with the question queue, keyed by `play.Question.Word()`.
  - **DRY rationale:** avoids re-rendering an entry to recover its regions, and avoids teaching `play` about `main`.
  - **Future extensions:** a second review form (`#7`, `#12`, `#13`) renders differently; keyed by word rather than by form, so it widens without changing shape.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `playRegion` | `cmd/define/replraw.go` | new | `playAnnounced` — the one switch on `RegionKind`, shared by both loops (D4) |
| `runPlay` | `cmd/define/play_loop.go` | unchanged | `liveScreen` — replaced both `crlfWriter`s (D9). **Landed in `#41`'s window, not this one** (`newConsole`, the builder both loops share), so it is untouched here and the status column says so |
| `playSession` | `cmd/define/play_loop.go` | modified | takes `console` (**already, via `#41` T3**) and the region map — D10 |
| `playSession`'s reveal write | `cmd/define/play_loop.go` | modified | `writeRendered` for the reveal (D3). This was a row for `draw`, which `#41` T4 **deleted**: the question, the reveal and the grading keys have three different lifetimes and a scrolling terminal could not express the difference, so the reveal is now written by the loop on `OutcomeReveal` and the keys are the frame's prompt (`livePrompt`). The seam D3 names is unchanged — it is the call that gains the click map |

- **`playRegion`** — given a region, what to play it against and an indicator, perform the playback.
  - **NOT pure**, and the label matters: it reaches `playAnnounced`, which fetches audio and writes. Its test uses the player and CDN fakes the repo already has, not a colocated unit test.
  - **Injected into:** both loops. The editor passes its erasable indicator; `--play` passes `defaultIndicator(opt)`.
  - **DRY rationale:** `#30` Done-when 7 — "one registry, not two special cases". A second switch breaks that the day a third `RegionKind` lands.

**ARCH-MOCK.** The terminal's double already exists: `playSession` is split from `runPlay` precisely so a test can drive it with a scripted key channel and no terminal (`play_loop.go:83-85`). The pty rows cover what only a terminal answers.

**ARCH-CONSTRAINTS.** `#30`'s envelope, inherited unchanged: paint at most every 16 ms with a trailing flush, uncapped line buffer. A sitting is bounded by `-count` (default 20) and writes a word plus at most one definition each, so the buffer stays the same order as an interactive session, which that envelope already covers. No new budget claimed.

---

## Tasks

Plain checkboxes, not `Mx` tags: this is single-pass work with ONE boundary, and AGENTS.md §3 says an `Mx` tag commits to its own `milestone-close`.

- [x] **T0 — one audio-off predicate** (D11). `options.playsAudio()`, applied inside `playAnnounced` so no caller can be below it, and the four hand-copies replaced by a call. Behaviour-preserving: every existing audio test passes untouched, and a new row asserts `playAnnounced` fetches NOTHING when audio is off — which none of them do today, since the callers never let it get that far.
- [x] **T1 — lift the click registry.** `playRegion(ctx, d, opt, r Region, entry string, ind indicator, stdout, stderr)`: the switch on `RegionKind`, building the utterance and calling `playAnnounced` — which now carries the guard itself (T0). The editor's `clicked` (`replraw.go:264`) becomes a call to it. NO behaviour change — `TestEveryRegionKindIsActionable`, `TestClickOnHeadwordReplays` and `TestClickOnOriginLanguagePlaysIt` pass untouched, which is what proves the lift was a lift.
- [x] **T2 — delete the playback dance** (D2), AND re-home the invariant that dies with it. **LANDED BY `#41` T3/T4** — see the 2026-08-31 revision below.
      `play_loop.go:174-198` loses `restore`/`enterRaw` and the `lost the terminal after playback` path. A new row asserts the alternate screen is STILL up after a reveal, which is the regression the deletion prevents.
      **The exit-1 test that drove the re-entry failure went with it, and it was the ONLY pin for the outcome-ORDER obligation** — `play_loop.go:126-141` enumerates three consumer obligations and names that test for `order`, recording that reversing the iteration once left the whole suite green (BR-13). Its premise is a `rawTerm` on `/dev/null` so re-entry fails, which this deletion makes unreachable.
      The replacement must OBSERVE THE ORDER, not a consequence of it. A first draft asserted "the record survives a failed playback", and the gate measured it green under a reversed iteration — correctly: once the early `return 1` is gone, both orders write the record, so the consequence stops discriminating. The old test worked only because the reveal arm could abort the loop.
      So the fake records a SEQUENCE: `capture.CaptureReview` and the player each append to one ordered log, and the assertion is that the record's entry precedes the playback's. That is falsifiable by reversing the `outs` iteration and by nothing else — which is what BR-13 needed and what a consequence-based test could not give once the abort was deleted.
      **The rule: a task that deletes code re-homes every invariant whose only pin lives there, in the same task — and re-homing means finding an observable that still discriminates, not porting the old assertion.**
- [x] **T3 — `--play` writes into a `liveScreen`.** `enterAlt`, `enterMouse`, `newLiveScreen`, `handBack` on exit, replacing both `crlfWriter`s (D9). **LANDED BY `#41` T3** as `newConsole(ctx, d, sess, stdout, newScreen)`, which both loops call — `--play` passes `newPinnedScreen` rather than `newLiveScreen`, because a status bar belongs at the terminal's bottom edge.
- [x] **T4 — the prompt word is a region.** One `RegionHeadword` at column 0, width `visibleCells(word)`, on **line 1 — not line 0**. `draw` is gone (`#41`); the loop writes `"\n" + q.Prompt() + "\n"`, and `addRegions` anchors at the line the write STARTS on, so the leading blank is line 0. Both forms put the headword on their first line.
- [x] **T5 — the revealed definition carries its regions** (D7, operator's choice), through `writeRendered` (D3). It adopts `Render` and `WriteRegions`, so it carries the three obligations THEY document at HEAD — the plan originally carried none, and two were live defects:
      - **`RenderOpts.Word` must be the DECK'S KEY.** Empty means "no click map wanted" and falls back to `Entry.Headword()`; `jalapeno` in the deck against `jalapeño` on the head line are different URLs at the CDN. Pinned by `TestARegionAnswersForTheDeckKeyNotTheHeadword`.
      - **the regions are relative to the RENDER**, and a form's reveal is larger, so the offset is LOCATED rather than computed from a formula.
      - **a pinned screen WRAPS between the caller and the buffer**, so the map is moved by the screen's own width — which is why that arithmetic lives in `liveScreen.WriteRegions` and not here (row 3b).
- [x] **T6 — the viewport, all three parts.** **LANDED BY `#41` T7/T8.** The alternate screen has NO scrollback, so without this a sitting cannot be scrolled at all — worse than today, where the terminal keeps it.
      - **Scroll:** PageUp/PageDown and the wheel, the same cases `replraw.go:366-380` already has.
      - **Resize:** `watchResize` (`rawterm.go:230`) into the loop's select, as `runEditor` does — the first fix for this finding covered scrolling and left SIGWINCH out, so a window change mid-sitting would have left the frame wrong.
      - Neither a scroll nor a resize reaches `play.Apply`.
- [x] **T7 — the click acts and does not answer** (D8). It stops beside the viewport gestures, before `toInput`, so `play.Apply` never learns a mouse exists.
- [x] **T8 — docs**: `cmd/define/README.md`'s review-loop section, and `atlas/define.md`'s clickable-regions section, which names the interactive loop as the only consumer.

## Done when

Every `red when` cell here was EXECUTED as a mutation at the close boundary, not
reasoned about — which is how row 2 was found to survive its own (`#38` BR-16).

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | the word being asked about is clickable | `TestPlayClickOnThePromptWordPlaysIt` | the region is not written with the word |
| 2 | a click ACTS and never answers | `TestPlayClickActsAndIsNotAnAnswer` — driven into the GRADED state on a TWO-word deck, and asserting both halves: the click played, and the sitting did not advance | the click reaches `play.Apply`. **Both mutations RUN**, because the first version of this row survived its own: disabling the `KeyClick` branch, and routing a click into `Apply` as an `InputReveal`. The second needs two questions to be observable — with one, advancing off the last question just ends the sitting, which looks the same |
| 3 | a revealed definition is clickable like anywhere else | `TestPlayARevealedDefinitionCarriesItsRegions` — drives the LOOP against a real screen and asserts a CLICK on the headword inside the definition resolves, so the offset arithmetic and the screen's rebasing are pinned at their joint | the reveal is written without its regions, or with regions never shifted past the lines `Choice.Reveal` puts above the entry |
| 4 | one registry, both loops | `TestEveryRegionKindIsActionableThroughTheSharedRegistry` drives `playRegion` itself, beside `TestEveryRegionKindIsActionable` which still drives the editor; `TestAnUnknownRegionKindPlaysNothing` pins that it refuses rather than guessing | a kind acts in one loop and not the other |
| 4b | **`-no-audio` fetches nothing, from any caller** | `TestPlayAnnouncedFetchesNothingWithAudioOff` — on `playAnnounced` itself, since that is where the guard now lives | the predicate is left in the callers, so a fifth one sits below it |
| 4c | **the outcome ORDER survives its pin's deletion** | `TestAMissRecordsBeforeItPlays` — one ordered log written by both the capturer and the player, replacing the deleted exit-1 test. **DONE:** landed with `#41` T3, mutation-verified against a reversed iteration | the `outs` iteration is reversed |
| 6b | **a resize repaints mid-sitting** | `TestPTYPlayResizeRepaints` | `watchResize` is not wired into the loop's select |
| 5 | **playback does not tear the screen down** | `TestPTYPlayKeepsTheAlternateScreenAcrossAReveal` | the restore/re-enter dance comes back |
| 6 | a sitting can be scrolled | **DONE by `#41` T7** — `TestPagingIsNotAnAnswer` (the loop pages and grades nothing) and `TestALongRevealPagesRatherThanScrollingTheWordAway` (the word comes back), both mutation-verified | the viewport cases are dropped, leaving no scrollback at all |
| 3a | **a wrap MOVES the click map rather than dropping it or misplacing it** (`#41` BR-25) | `TestAWrapMovesTheClickMapRatherThanDroppingIt` (the arithmetic: a region on an unbroken line moves down, one on a broken line is dropped) and `TestWriteRegionsMovesTheMapByTheScreensOwnWidth` (the seam) | a region on an unbroken line is thrown away — which made the ASKED word inert on every form-2.3 question — or one is kept whose column now belongs to a continuation |
| 3c | **a region answers for the deck's key, not the entry's headword** | `TestARegionAnswersForTheDeckKeyNotTheHeadword` | `RenderOpts.Word` is left empty, so `regionsIn` falls back to `Entry.Headword()` and a click on a normalised deck word fetches a different recording |
| 3b | **the map is moved by the width the SCREEN wraps at, not one the caller supplies** | `TestAResizeDoesNotMisplaceTheClickMap` — narrow mid-sitting, then reveal, and every region still names the line it points at | the arithmetic runs on a second ruler, so after a resize a headword region lands on a blank line |
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

### 2026-08-30 — plan-quality round 3 (PQ-8 again, PQ-11)

- **PQ-8 came back because round 2 fixed the instance.** Putting the audio-off
  guard in `playRegion` made a FIFTH hand-copy of a class of four. The class is
  enumerable and `playAnnounced`'s own doc comment already names it as a
  divergence it was built to end — and ended two of three, leaving this one above
  it in every caller. One named predicate, applied INSIDE `playAnnounced`, is the
  rule; D11 rewritten.
- **PQ-11 — a task that deletes code must re-home every invariant whose ONLY pin
  lives there, in the same task.** T2's deletion strands
  `TestLosingTheTerminalAfterPlaybackExitsOne`, whose premise is a re-entry that
  can fail — and `play_loop.go:126-141` names it as the sole pin for the
  outcome-ORDER obligation, recording that reversing the iteration once left the
  whole suite green. A deletion that quietly removes a pin is a regression with a
  green suite, which is the same shape as the vacuous guards `#30` kept finding.

### 2026-08-30 — plan-quality round 4: the replacement pin did not discriminate

PQ-11's first replacement — "the record survives a failed playback" — was
measured GREEN under a reversed `outs` iteration, and the gate was right. The
old test discriminated only because the reveal arm could abort the loop with
`return 1`; with that abort deleted (T2), both orders write the record and the
consequence stops separating them.

So the replacement observes the ORDER DIRECTLY: one ordered log that both the
capturer and the player append to, asserting the record's entry comes first.
The wider rule, which is the part worth keeping: **re-homing an invariant means
finding an observable that still discriminates, not porting the old assertion to
a world where its mechanism is gone.**

### 2026-08-31 — `#41` landed T2, T3 and D9 first

`#41` (`--play` paints frames through `screen`, and gains a status bar) needed
the same seam this plan's T3 describes, and needed it for a different reason: a
status bar has nowhere to go on a surface that owns no coordinates. So it landed
T2 and T3 ahead of this issue rather than duplicating them.

What is now IN THE TREE, and no longer this plan's to do:

- **T3 / D9** — `newConsole` builds the console for both loops, and
  both `crlfWriter`s are gone. One difference from what T3 wrote: the screen is
  `newPinnedScreen`, not `newLiveScreen`, so the buffer region fills and the
  footer sits on the bottom row (`#41` D3a).
- **T2 / D2** — the reveal's `restore`/`enterRaw` pair and its exit-1 branch are
  deleted. `#41` D5a found the same Critical this plan's PQ-1 did, from the other
  side: `enterAlt` is opt-in and `restore` leaves the alternate screen, so a
  frame-drawing sitting loses it on the first reveal.
- **4c** — the outcome-ORDER pin is rebuilt as `TestAMissRecordsBeforeItPlays`,
  built to THIS plan's round-4 specification: one ordered log written by both the
  capturer and the player, because a consequence-based assertion measured green
  under a reversed iteration once the abort was gone. Verified by mutation.

**What this plan still owns, and what changed under it:** the click map. `draw`
is gone — `#41` T4 split it by lifetime, so the question and the reveal are
buffer writes the loop makes on transitions and the grading keys are the frame's
prompt (`livePrompt`). D3's seam is unchanged in substance: the reveal's write is
still the call that has to carry regions, it is just made from the loop rather
than from a function called `draw`. D5's "the visual result stays close" now has
a stronger form than it claimed — the sitting is a frame, with paging.

Re-read D3, D5 and T4 against the current `playSession` before implementing.

### 2026-08-31 — re-read against the tree `#41` left, before implementing

`#41` merged (PR #25). Re-reading D3, D5 and T4 against the current
`playSession`, as the previous revision asked. Four deltas, one of them a
constraint that did not exist when this plan was written.

**T4's premise is gone; its conclusion survives.** The row says *"`draw` is
append-only, so the word lands on the line about to be written"*. `draw` no
longer exists — `#41` T4 split it by lifetime, and the loop writes `q.Prompt()`
once per question in `playSession`'s `show()` closure. The word is still line 0,
column 0 of that write, for BOTH forms: `Recall.Prompt()` is the word, and
`Choice.Prompt()` is the word, a blank, then the options. So the region is
computed at the same place, from a different function.

**T5 GAINS A CONSTRAINT, and it is the one `#41` BR-25 predicted.** A pinned
screen's `WriteRegions` now goes through `writeBuffer`, which WRAPS — because a
frame clips an over-wide line and every path into the buffer had to be covered.
`#41` wrote the consequence down for whoever arrived first, and this issue is
that consumer:

> A REGION's column is relative to the text it was computed from, so a wrap that
> moves a word moves what a click there means. That is a real cost and it belongs
> to whoever first writes regions into a pinned screen: they must wrap BEFORE
> computing the regions.

So T5 is no longer "call `writeRendered` and the regions ride along". The order is
**wrap, then compute regions against the wrapped text, then write**. A reveal
whose regions were computed pre-wrap would put the underline and the hit test on
different words — silently, and only on entries long enough to wrap, which is
most of them. This is the task's real risk and it was not in the estimate.

**D3's seam is intact.** `writeRendered` still dispatches on the `regionWriter`
interface (`main.go:801`), and the console's stdout is the `liveScreen`, which
implements it. Nothing to change there.

**A sitting now REFUSES without a terminal or with `-no-color` (`#41` BR-3), and
`playRig` runs coloured.** Both matter for this issue's tests: a click test drives
`playSession` with a recorder, which is unaffected, but any test reaching
`runPlay` meets the gate, and assertions over frame text now meet SGR — use
`unstyled`, which `#41` moved out of the conformance file for exactly this.

**Still to do: T0, T1, T4, T5, T7, T8.** T2, T3 and T6 landed with `#41`, and its
close verified two of this plan's Done-when rows on real hardware under the exact
test names this plan predicted (`TestPTYPlayKeepsTheAlternateScreenAcrossAReveal`,
`TestPTYPlayResizeRepaints`), plus row 4c's replacement pin
(`TestAMissRecordsBeforeItPlays`) built to this plan's round-4 specification.

**The estimate is stale in both directions** and is re-derived at `change-code`:
three tasks are gone, and T5 grew a wrap-ordering problem the original 2.18h did
not price.

### 2026-08-31 — T1, T4, T5, T7, T8 landed; the Done-when names what exists

Three rows named tests that were never written under those names, which is the
`plan-table-vs-tree` family `#41` was caught by four times. Corrected rather than
left for the close to find:

- **Row 3** is `TestPlayARevealedDefinitionCarriesItsRegions`, not
  `TestPlayClickOnARevealedHeadword`. Its first draft called `marksIn` directly
  and passed under a mutation that stopped the LOOP calling it — the gap
  `lessons.md` records verbatim ("deleting the loop's call leaves every unit test
  green"). It now drives the loop against a real screen and asserts a CLICK
  resolves, so the loop's offset arithmetic and the screen's buffer rebasing are
  pinned at their JOINT rather than separately.
- **Row 4** gained `TestEveryRegionKindIsActionableThroughTheSharedRegistry`
  beside the editor's, plus `TestAnUnknownRegionKindPlaysNothing`: a kind with no
  row must refuse, not fall back to the headword, because a plausible fallback is
  how a missing case ships.
- **Row 6** was `#41`'s work and is named as such.
- **Row 3a is NEW**, and it is the constraint `#41` BR-25 handed forward.

**T5's resolution, since the plan priced it as the risk and it was.** A region's
coordinates are relative to the text they were computed from, and `#41` put a
wrap between the caller and the buffer. `writeClickable` applies the wrap first
and passes the regions along only if it changed nothing — so at a sitting's own
width everything is exact, and after a NARROWING resize the underlines stop until
the next question is written. **An underline that plays the word beside the one
you pointed at is worse than no underline: losing an affordance is visible, a
wrong click is not.** The alternative — re-rendering each remaining entry at the
new width so the regions are correct again — is real and cheap in IO (the entries
are parsed and held), but it rebuilds `play.Question` values the forms own, and
that is a design change rather than a fix.

### 2026-08-31 — the operator found it inert, and the rig is why

> tried `define --lang en --play` nothing seems clickable (for sound) though?

Reproduced at a real terminal width in one probe: on a form-2.3 question the
PROMPT wraps — it is the headword, a blank, then four glosses, and a gloss
routinely runs past 80 columns — so `writeClickable`'s all-or-nothing rule
("drop the whole map if the wrap changed anything") threw away the region for the
word the sitting is asking about. The feature was inert in the case it exists
for. The reveal's regions survived, because the definition is rendered at the
sitting's own width and needs no wrap.

**The rule is now per LINE.** `wrapMovedRegions` asks `wrapWritten` itself how
many rows each line becomes — the same function, so the erase-gesture exemption
and the sub-20 policy cannot differ between the two sides — then moves a region
on an unbroken line down by the rows the wrap added above it, and drops only the
regions on lines the wrap actually broke. That keeps the wrong-click guarantee
(a column past a break belongs to a continuation, and guessing which is the bug)
while losing nothing that did not move.

**Why four tests written for this feature were green over it: `playRig` carried
`width: 0`.** That is `terminalWidth`'s "do not wrap" sentinel, and a sitting
never has it — `--play` refuses unless stdout is a terminal. So the wrap every
region has to survive was OFF in every in-process test. This is the SECOND time
in two issues that this rig's defaults hid a real defect: `#41` BR-24 was
`color: false`, a configuration `--play` refuses outright.

`lessons.md` now carries the generalisation rather than the instance: **every
sentinel-valued default in a rig is a state production may not have** — `0`
meaning off, `""` meaning none, a nil clock — and each one silently removes the
behaviour the test was written to check. The rig runs at `defaultCols` now.

### 2026-08-31 — boundary review round 1: a Critical of the same shape as the one before it

The review returned REWORK on a Critical (BR-3) that is the operator's bug wearing
the other face, and the pair together is the finding worth keeping:

- **First version:** all-or-nothing — drop the whole map if the wrap changed
  anything. Made the ASKED word inert on every form-2.3 question, because the
  prompt puts four glosses under the headword and a gloss routinely wraps.
- **Second version:** per line, measured with `opt.width` — the width the LOOP
  was handed at startup — while the screen wraps at `l.cols`, which `Resize`
  updates. After a narrowing resize the two rulers disagree and every region
  lands on a line that does not contain its text: a headword region on a blank
  line, an ORIGIN region on a quotation. **That is the wrong click the rule
  exists to forbid**, reproduced by the reviewer in the real loop.

**THE RULE: the wrap and the map must be measured by ONE ruler, and only the
screen holds it.** `wrapMovedRegions` now runs inside `liveScreen.WriteRegions`,
which already holds `l.cols` and already routes through `writeBuffer`. The
caller-supplied width is gone, so the second ruler is not merely unused — it is
unexpressible. That is the same shape as `#41` BR-20's resolution (wrap at the
seam, because a rule every caller must remember has a caller who will not), and I
did not apply it here until a review measured the consequence.

**BR-1, and it is the plan's failure rather than the code's.** T4 and T5
specified coordinates against premises `WriteRegions` and `Render` do not hold,
and named none of the three obligations those two document at HEAD: the leading
newline means the prompt word is line 1 rather than 0; `RenderOpts.Word` must be
the deck's KEY, because empty falls back to `Entry.Headword()` and `jalapeno`
against `jalapeño` are different URLs at the CDN; and the map has to be moved by
the screen's width. All three are now rows in T5, and the `Word` one was a live
defect — a click on a normalised deck word would have fetched the wrong
recording.

**A task adopting an existing mechanism carries that mechanism's
HEAD-documented obligations as checkable items.** Reading `WriteRegions` and
`RenderOpts` at the moment T5 was written would have produced all three; the plan
was written before `#41` existed and never re-read them.

### 2026-08-31 — round 2: the plan's own tables were the Critical, and a fix with no pin

Three findings, and two of them are about this document rather than the code.

- **BR-11 (Critical) — the Core-concepts PURE table named `playRegions` and
  `promptRegionFor`, and the tree declares neither.** `playRegions` was specified
  as a bare `word -> []Region` map back when `--play` still appended; the regions'
  coordinates are relative to a TEXT, so the text has to travel with them, and the
  entity that exists is `clickable`. `promptRegionFor` was never built at all: the
  prompt word's region is one struct literal at the write site, and a named
  constructor for a three-field literal used once is indirection rather than
  reuse. Both rows now name what the tree has, and the "never built" is recorded
  rather than quietly dropped.

- **BR-12 — the Tasks showed seven of nine unticked while the issue ticked all
  nine and the code had landed.** My earlier edit to these rows used `replace`
  without an assert against text that had already changed, so it no-opped in
  silence. **A scripted edit to an artifact asserts on what it expects to find,
  or it reports success for having done nothing** — the same discipline the code
  guards enforce, applied to the tool doing the editing.

- **The `RenderOpts.Word` fix had no test**, which the review measured by deleting
  `Word: key` and finding the whole suite green. Fixing an obligation is not
  discharging it: `regionsIn` falls back to `Entry.Headword()`, so the divergence
  the fix commit called "a live defect" was unpinned.
  `TestARegionAnswersForTheDeckKeyNotTheHeadword` drives a deck holding
  `jalapeno` against an entry reading `jalapeño` — different URLs at the CDN —
  and reddens under that deletion. Row 3c.

**The rule these leave, and it is one rule: a claim is discharged by something
that can fail.** A table row is a claim about the tree; a fix is a claim about
behaviour. Neither is worth anything until something reddens when it stops being
true — which is exactly what this plan's own Done-when preamble demands of its
rows and what BR-1's family has now said three times.

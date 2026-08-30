# Clickable Words in the Review Loop Implementation Plan (`#38`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Click the word `--play` is asking about to hear it, and click anything in a revealed definition exactly as the interactive loop already allows.

**Architecture:** `--play` stops drawing its own frames and writes into `#30`'s `liveScreen`, which is an `io.Writer` — so `draw` does not change. It gains the alternate screen, a click map, scrolling and the exit transcript from the same machinery the interactive loop uses, and the click actions become rows in the registry that already exists rather than a second path.

**Tech Stack:** Go 1.26. Everything this needs was built by `#30`; no new dependency, and no new mechanism.

---

## Decisions

**D1 — `--play` joins the screen rather than tracking a cursor, and the operator chose this.** The alternative was mouse reporting with a remembered cursor position, which `#30` D1 rejected on measurement: the terminal scrolls whenever the loop writes, so the app tracks an offset it never observes. A click after any scroll lands wrong. Adopting the screen makes the mapping exact by construction, which is the property the whole affordance rests on.

**D2 — the visual result stays close, and that is why this is affordable.** `draw` (`play_loop.go:283-291`) is APPEND-ONLY: it writes `"\n%s\n"` per question and never clears, so the terminal scrolls and a sitting stacks up. `liveScreen` has the same shape — `Write` appends to a line buffer and `Paint` shows its tail — so a sitting looks as it does today, gains scrolling, and gains the exit transcript. **`draw` does not change at all**: it takes an `io.Writer` already.

**D3 — the scrollback trade is inherited deliberately, not by accident.** The alternate screen means a sitting is not in the terminal's scrollback WHILE it runs; it is printed back on exit (`#30` D3, `handBack`). For `--play` this is a smaller loss than for lookups — a review sitting is a transaction you finish, not a reference you scroll back to mid-flight — and the transcript still lands. Stated because the operator was asked and chose it, so a later reader does not re-open it.

**D4 — `--play` has NO live edge, and that is a difference from the interactive loop worth naming.** `liveScreen.Draw(prompt, menu)` exists because the editor's prompt is rewritten per keystroke. `--play` has no editable line: its `y = got it…` line is printed once per question and is as much a part of the record as the word above it. So `--play` writes everything as BUFFER lines and never calls `Draw` — which also means nothing here can hit `#30`'s "the prompt belongs to a loop that is waiting" rule, because there is no prompt to leave stale.

**D5 — the reveal's regions come from `Render`, and `play` must never see them.** `play` is a pure package that does not import `main`; `Region` lives in `main` (`render.go`). `play.NewRecall(key, rendered)` (`play_loop.go:265`) stores only the rendered string. So the loop keeps its own `word -> []Region` map built where the entries are rendered (`play_loop.go:262`, which already discards the regions), and looks it up when a reveal is drawn. `play`'s purity is the reason, not an accident of layering.

**D6 — a click is a REPLAY and never an answer.** Hearing the word is what `y`/`n` are answering *about*. A click that recorded a review would corrupt the schedule silently — the worst kind of bug this program can have, since the damage is to data the learner cannot see. The click performs playback and returns to the same state.

**D7 — `--play` keeps `crlfWriter` for nothing, and that is the `#32` half falling due.** `#30` D5a said: "`#32` keeps its `--play` half, which continues to draw its own frames through `crlfWriter`". Once `--play` writes into the screen, the screen owns line placement and the second `crlfWriter` goes — one line-ending owner, which is what `#32` is filed about. This issue closes that half; `#32` should be re-read when it does.

---

## What this plan asserts about the existing tree, verified

`#30` spent a finding on unchecked claims about the tree (PQ-10), so every claim below carries a `file:line` and was read.

| claim | verified at | status |
|---|---|---|
| `draw` is append-only and never clears | `play_loop.go:283-291` | true — `fmt.Fprintf(w, "\n%s\n", q.Prompt())` |
| `draw` already takes an `io.Writer` | `play_loop.go:283` | true — so it needs no change |
| a recall form's prompt IS the word | `play/recall.go:29` | true — `Prompt() string { return r.word }` |
| `--play` wraps stdout/stderr in `crlfWriter` | `play_loop.go:64-65` | true |
| the entries are pre-rendered and their regions discarded | `play_loop.go:262-265` | true — `rendered, _ := Render(...)`, then `play.NewRecall(key, rendered)` |
| `enterRaw` takes the control writer | `play_loop.go:48` | true — `enterRaw(f, stdout)`, so `enterAlt`/`enterMouse` are available on the session |
| playback borrows and returns raw mode | `play_loop.go:174-198` | true — `raw.sess.restore()` then `enterRaw` again |
| `handBack` is the exit sequence, over two interfaces | `replraw.go:170` | true — `Stop`, `restore`, print transcript |
| `liveScreen` is an `io.Writer` with `WriteRegions` / `RegionAtRow` | `screen.go:558,568` | true |
| the click actions are a registry keyed by `RegionKind` | `replraw.go:264` | true — but `clicked` closes over the editor loop's `sess`, so it is not reusable as-is (see M1.1) |

**The one thing that is NOT reusable, stated up front:** the interactive loop's `clicked` closure is defined inside `runEditor` and captures `sess`, `d`, `opt`, `stdout`, `stderr`. `--play` has no `session`. So the ACTION has to be lifted to a function both loops call, or the registry is spelled twice — which is exactly what `#30`'s Done-when 7 forbids.

---

## Milestone M1 — the review loop draws on the screen

### Core concepts

#### Pure entities

| Name | Lives in | Status | Kind |
|------|----------|--------|------|
| `playRegions` | `cmd/define/play_loop.go` | new | PURE — the `word -> []Region` map the loop keeps because `play` cannot hold them (D5) |
| `promptRegion` | `cmd/define/play_loop.go` | new | PURE — the region for the word `draw` is about to write |
| `replayRegion` | `cmd/define/replraw.go` | new | PURE dispatch — the click registry, lifted out of `runEditor`'s closure so both loops share one |

- **`playRegions`** — questions are pre-rendered before the sitting starts, so their regions are known then and needed later.
  - **Relationships:** 1:1 with the question queue; keyed by the deck word, which is `play.Question.Word()`.
  - **DRY rationale:** avoids re-rendering an entry to recover its regions, and avoids teaching `play` about a `main` type.
  - **Future extensions:** a second form (`#7`, `#12`, `#13`) renders differently; the map is keyed by word, not by form, so it widens without changing shape.

- **`replayRegion`** — given a region and what to play it against, perform the replay.
  - **DRY rationale:** THE point of the milestone. `#30` Done-when 7 says regions are "one registry, not two special cases"; a second loop with its own switch would break that the day a third `RegionKind` is added. `TestEveryRegionKindIsActionable` must cover both callers after this.

#### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `runPlay`'s screen | `cmd/define/play_loop.go` | modified | `liveScreen` — replaces the two `crlfWriter`s (D7) |
| `playSession` | `cmd/define/play_loop.go` | modified | takes a `display` and a click map instead of raw writers |

**ARCH-MOCK.** The external dependency is the terminal and its double already exists: `playSession` is split from `runPlay` precisely so a test can drive it with a scripted key channel and no terminal (`play_loop.go:83-85`). That seam is why this milestone is testable in process; the pty rows cover what only a terminal can answer.

**ARCH-CONSTRAINTS.** The envelope is `#30`'s, inherited unchanged: paint at most every 16 ms with a trailing flush, and an uncapped line buffer. A sitting is bounded by `-count` (default 20) and each question writes a word plus at most one definition, so the buffer is thousands of lines at worst — the same order as an interactive session, which the envelope already covers. No new budget is claimed.

### Tasks

- [ ] **M1.1 — lift the click registry out of `runEditor`.** `replayRegion(ctx, d, opt, r Region, entry string, stdout, stderr)` — the switch on `RegionKind`, with the entry text passed rather than a `session` captured. `runEditor`'s `clicked` becomes a call to it. NO behaviour change: `TestEveryRegionKindIsActionable`, `TestClickOnHeadwordReplays` and `TestClickOnOriginLanguagePlaysIt` must pass untouched, which is what proves the lift was a lift.
- [ ] **M1.2 — `--play` writes into a `liveScreen`.** `enterAlt`, `enterMouse`, `newLiveScreen`, and `handBack` on exit — replacing both `crlfWriter`s (D7). `draw` and `finish` are untouched (D2). The pty row: a sitting still shows its word, still grades, and the terminal is restored.
- [ ] **M1.3 — the prompt word is a region.** `draw` is append-only, so the loop knows the word lands on the line it is about to write: `WriteRegions` with one `RegionHeadword` at column 0, width `visibleCells(word)`. Pinned in process by resolving a click at that cell back to the word.
- [ ] **M1.4 — the revealed definition carries its regions** (operator's choice). The map from D5, built at `play_loop.go:262` where the regions are currently discarded, written with the reveal.
- [ ] **M1.5 — the click acts, and does not answer** (D6). `KeyClick` reaches `replayRegion` and never `play.Apply`, so no verdict is recorded and the sitting does not advance.
- [ ] **M1.6 — docs**: `cmd/define/README.md`'s review-loop section, and `atlas/define.md`'s — the clickable-regions section names the interactive loop as the only consumer, which this makes false.

### M1 Done-when

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | the word being asked about is clickable | `TestPlayClickOnThePromptWordPlaysIt` | the region is not written with the word |
| 2 | a click NEVER answers | `TestPlayClickIsNotAnAnswer` — asserts no review recorded, `Right`/`Wrong` unchanged, the question still current | the click reaches `play.Apply` |
| 3 | a revealed definition is clickable like anywhere else | `TestPlayClickOnARevealedHeadword` | the region map is not written with the reveal |
| 4 | one registry, both loops | `TestEveryRegionKindIsActionable` extended to cover `replayRegion` directly | a kind acts in one loop and not the other |
| 5 | the terminal is handed back | `TestPTYPlayLeavesTheTerminalRestored` (existing rows still green) | `handBack` is dropped from an exit path |
| 6 | a mouse-less terminal is unaffected | the existing `--play` pty rows, unchanged | the enable is emitted conditionally, or the loop needs a click |

---

## Verification before close

```bash
go test ./... && go test ./cmd/define/ -race
go test -tags conformance ./cmd/define/    # unsandboxed; the pty rows
```

Then, on a real terminal: `define --play`, click the word, hear it; press `n`, click the headword and the `ORIGIN` language in the revealed entry; scroll back with PageUp and click a word from earlier in the sitting; quit and confirm the transcript is in the scrollback.

**Close:** one milestone, one `sdlc close`, one publish.

# Play Frame Chrome Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the band at the bottom of a `--play` frame read as chrome — separated from the record by a reserved row, dimmed so the border is visible — and stop playback committing a blank line to the buffer every time it runs.

**Architecture:** Three independent changes to one surface. The separation is a row the frame RESERVES in `Paint`'s existing component budget (never a newline inside the prompt string, which `displayRows` would mis-measure and raw mode would mis-write), and it is the first component sacrificed when the terminal is too short to hold everything. The colour is applied to the prompt and bar strings where they are handed to `Draw`, leaving the plain text the README pins untouched. The indicator gets one shape for "inside a screen": `playRegion`'s parameter is deleted outright, since a click can only happen inside one, and the three sites still free to choose are held to it by a guard that matches on the ARGUMENT'S TYPE rather than on a callee name.

**Tech Stack:** Go, `cmd/define` (package `main`) — the screen, the sitting loop, and the playback announcer. No new dependencies.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `chromeGap` | `cmd/define/screen.go` | new |
| `screen.gap` | `cmd/define/screen.go` | new |
| `screen.Paint` | `cmd/define/screen.go` | modified |
| `grantedGap` | `cmd/define/screen.go` | new |
| `asChrome` | `cmd/define/playbar.go` | new |

- **`chromeGap`** — the number of rows the frame holds empty between the record and the live edge. One, and named rather than spelled `1` at the three sites that must agree (`Paint`'s budget, `Paint`'s `footerTop`, `fitsABoard`).
  - **Relationships:** 1:1 with a pinned screen — the editor's screen sets it to zero.
  - **DRY rationale:** The board already discovered this gap and paid for it out of the append-only buffer (`play_loop.go`, `written != s.Index`). Making it a frame row gives every form one rule instead of one form an exception, and takes the blank out of the exit transcript.
  - **Future extensions:** If the editor wants the same border, it is one line in `newLiveScreen`.

- **`screen.gap`** — the reserved-row count this screen honours. A field rather than a constant read directly, because the editor and the sitting want different answers and `newPinnedScreen` is where that difference is already made visible.
  - **Relationships:** N:1 with `screen` (one per screen).
  - **DRY rationale:** Keeps the two surfaces' difference at the constructor, where `pinned` already lives, instead of at every `Paint`.

- **`screen.Paint`** — the frame budget gains one component. Three call sites move together: `s.rows` gives the row up, the gap rows are emitted between the buffer and the prompt, and `s.footerTop` counts it.
  - **DRY rationale:** The row accounting is already single-owner ("ONE row accounting, used twice"); this adds a term to it rather than a second sum.

- **`grantedGap`** — whether a frame of this shape gets its gap: `want` when a buffer row survives beside it, else zero.
  - **Relationships:** 1:2 — `Paint` asks it when drawing, `fitsABoard` when deciding whether Enter may spend a board.
  - **DRY rationale:** The gap is the only component that can be sacrificed, so it is the only one whose charge and emission can disagree. Two spellings of "is there room" is exactly the divergence `boardFitsIn` was consolidated to end; this keeps the new term inside that consolidation instead of beside it.

- **`asChrome`** — plain text in, dimmed text out; the identity when the palette is off.
  - **Relationships:** used by the sitting's prompt and its bar, so the whole band is styled by one rule.
  - **DRY rationale:** First occurrence; without it the dim sequence would be spelled at each of the three prompt strings (`gradePrompt`, `gradedPrompt`, `boardRefusal`) plus the bar.

- **`fitsABoard`** — UNCHANGED, and see `## Revisions`: charging the gap here turned out to be provably a no-op, so the deliverable became the proof rather than the arithmetic.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `newPinnedScreen` | `cmd/define/screen.go` | modified | the sitting's terminal |
| `screenIndicator` | `cmd/define/main.go` | new | the playback announcer's cursor contract |

- **`newPinnedScreen`** — sets `gap: chromeGap` beside `pinned`.
  - **Injected into:** `newConsole`, from `play_loop.go:106` — the one production caller.

- **`screenIndicator`** — the `indicator` shape for a caller writing into a screen: erasable, with **no `before`**, because inside an append-only buffer a newline is content rather than cursor movement.
  - **Injected into:** `playAnnounced`, from the three screen-hosted sites that still SUPPLY one after `playRegion`'s parameter is deleted — `play_loop.go:507` (reveal), `replraw.go:633` (`/pron`), `:653` (submit).
  - **Future extensions:** A fourth screen-hosted path takes the constructor rather than re-deriving the rule — enforced by the guard below, not left to the next reader.

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `playRegion` | `cmd/define/replraw.go` | modified | playback for a clicked span |

- **`playRegion`** — loses its `ind` parameter and calls `screenIndicator()` itself.
  - **DRY rationale:** **A click is only possible inside a screen** — it arrives as SGR mouse reporting in raw mode — so the parameter offered a choice with exactly one right answer, and its two callers took different ones. Its own doc comment calls the indicator what they "legitimately differ on"; that sentence IS the bug, and deleting the parameter removes two supplying sites structurally instead of guarding them (Simplicity First: the argument that cannot be wrong is the one that is not there).
  - **Note:** two test callers (`editorloop_test.go:946,967`) pass an indicator and lose the argument with it.

| Name | Lives in | Status |
|------|----------|--------|
| `TestEveryScreenPlaybackTakesTheScreenIndicator` | `cmd/define/indicator_guard_test.go` | new |

- **`TestEveryScreenPlaybackTakesTheScreenIndicator`** — a source-level guard over the screen-hosted files.
  - **The predicate is BY TYPE, not by callee name.** "Every `playAnnounced(` call passes `screenIndicator()`" is the instance-shaped version, and it is blind to exactly the site this issue is named for: the sitting's click goes through `playRegion`, a forwarder, so the walk would never reach it. The rule is: **every call argument of type `indicator`, in these files, is a call to `screenIndicator()`** — with a bare identifier permitted when the enclosing function itself takes an `indicator` (no such function survives here, but the class is what is being stated).
  - **DRY rationale:** Five sites drifted three ways because nothing enforced them, and the first draft of this guard drifted the same way for the same reason — it enumerated a callee instead of naming the property. Same shape as this package's purity guard (import allowlist + wall-clock grep) and its doc-sync tests.
  - **Future extensions:** A new screen-hosted file joins the list; the guard fatals on an empty list, and on a file in which it finds no `indicator` argument at all, so it can never certify nothing.

**Test surface.** All four pure entities are exercised through `screen_test.go`'s `readFrame`, which interprets an emitted frame the way a terminal would — placement, not substring search. No IO mocks: `screen` is already the pure half of the ARCH-MOCK split, with `liveScreen` the only part that touches a tty.

---

## Chunk 1: the three changes

### Task 1: playback into a screen carries no `before` — at all five sites

Independent of the other two and fixes a live bug on the DEFAULT path, so it goes first.

**Files:**
- Modify: `cmd/define/main.go` (beside `defaultIndicator`, ~line 822)
- Modify: `cmd/define/replraw.go:584` (`playRegion` loses `ind`), `:343`, `:633`, `:653`
- Modify: `cmd/define/play_loop.go:359` (drops the argument), `:507` (reveal)
- Test: `cmd/define/play_loop_test.go`, `cmd/define/editorloop_test.go:946,967`, `cmd/define/indicator_guard_test.go`

- [ ] **Step 1: Write the failing behaviour test**

Drive the REVEAL path, not the click path: the reveal fires on every answered question, which is what makes this a per-question leak rather than a per-click one. Twice, because one leftover row is indistinguishable from ordinary spacing — which is how it shipped.

```go
// Playback in a sitting must COMMIT nothing to the buffer. The indicator is
// ephemeral and takes its own line back — but `defaultIndicator` also writes a
// newline BEFORE it, and inside an append-only buffer that newline is content the
// erase cannot reach. Operator, 2026-09-02: "after clicking on pronunciation in
// the daily play, one additional line's inserted".
//
// THE REVEAL, not the click: the click is where it was seen and the reveal is
// where it lives, firing once per answered question. TWICE, because one row reads
// as spacing.
func TestSittingPlaybackCommitsNothingToTheBuffer(t *testing.T) {
	// … build the rig, answer two questions that each play audio, then:
	if grew := after - before; grew != wantContentLines {
		t.Errorf("the buffer grew by %d lines over two playbacks, want %d — "+
			"the indicator's leading newline is committed and the erase cannot take it back",
			grew, wantContentLines)
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./cmd/define -run TestSittingPlaybackCommitsNothingToTheBuffer -v`
Expected: FAIL, the buffer grew by two more lines than the content written.

- [ ] **Step 2a: Write the failing GUARD test**

The sweep is the instance; this is the class. Without it the sixth site is free to be written wrong, which is exactly how five sites came to disagree three ways.

```go
// EVERY indicator ARGUMENT in a screen-hosted file is screenIndicator().
//
// Source-level, because the rule is about a call site's argument and no type can
// express "this writer is a screen" — `playAnnounced` takes an io.Writer, and
// that is right: the one-shot and piped paths genuinely want `before`.
//
// BY ARGUMENT TYPE, NOT BY CALLEE. The first draft of this guard walked calls to
// `playAnnounced` and would have been blind to the sitting's click, which reaches
// it through `playRegion` — the exact site the issue is named for. Naming a callee
// enumerates instances; naming the type states the class.
//
// FATALS on an empty file list, and on a file where it matches nothing: a guard
// that certifies nothing is the failure `fallbackReasons` already fixed once.
func TestEveryScreenPlaybackTakesTheScreenIndicator(t *testing.T) {
	for _, f := range []string{"play_loop.go", "replraw.go"} {
		// parse; for every call expression, for every argument whose parameter
		// type is `indicator`, assert the argument is a call to screenIndicator
		// (or a bare identifier when the enclosing func itself takes one)
	}
}
```

Prefer `go/ast` over a regexp, and **reuse `repo_guard_test.go`'s walker** — that file (package `main_test`) already imports `go/ast`/`go/parser` and already carries the Fatal-never-Skip discipline this guard wants ("a guard that reports nothing when it cannot run certifies nothing"). `dict_symbols_darwin_test.go` is NOT the precedent: it compares a C resolver's symbol list against a Go list and parses no Go source. Resolving a parameter's TYPE across files needs `go/types` (or the argument position of the few known callees); if `go/types` is more machinery than this earns, match on the callee's parameter position derived from its own declaration in the same package, and say in the comment that that is what makes it package-local.

- [ ] **Step 3: Delete the choice, then sweep what is left**

First, `playRegion` loses `ind` and calls `screenIndicator()` itself — two supplying sites gone rather than guarded. Its doc comment currently says the indicator is what the two callers "legitimately differ on"; replace that sentence, because it is the bug stated as a design note.

Then the three that remain: `play_loop.go:507`, `replraw.go:633`, `replraw.go:653`.

```go
// screenIndicator is the indicator for a caller writing into a SCREEN.
//
// NO `before`, and that is the whole difference. On a cooked terminal "\n" is
// cursor movement and the erase that follows clears the line it moved to; inside
// an append-only buffer the newline is CONTENT, `eraseOpenLine` takes back only
// the open line the indicator itself wrote, and the blank stays — one per
// playback, forever.
//
// FIVE sites wrote this three ways — three copies of a literal and two of
// `defaultIndicator`, which is the one-shot path's — so the rule existed only as
// whatever each author happened to type (ARCH-DRY). Naming it is what makes the
// next site right by default; deleting playRegion's parameter is what makes two
// of them unable to be wrong at all.
func screenIndicator() indicator {
	return indicator{show: true, erase: eraseLine}
}
```

Correct the comment at `play_loop.go:356`, which claims `defaultIndicator` is "what every other playback on this path already uses": true of the one-shot path, false of every screen.

- [ ] **Step 4: Run the tests and the neighbours**

Run: `go test ./cmd/define -run 'Playback|Indicator|AClickPlays|Pron' -v`
Expected: PASS. The editor's own playback tests are the ones that catch a wrong sweep at `replraw.go:653`.

- [ ] **Step 5: Commit**

```bash
git add cmd/define/main.go cmd/define/play_loop.go cmd/define/replraw.go \
        cmd/define/play_loop_test.go cmd/define/indicator_guard_test.go
git commit -m "#44: playback into a screen carries no leading newline, and a guard says so"
```

---

### Task 2: the frame reserves a row between the record and the live edge

**Files:**
- Modify: `cmd/define/screen.go` (`screen` struct ~line 60, `Paint` ~line 443, `newPinnedScreen` ~line 602)
- Modify: `cmd/define/play_loop.go` (`fitsABoard` ~line 697; delete the board's blank-buffer-line write in `show()`)
- Test: `cmd/define/screen_test.go`, `cmd/define/play_loop_test.go`

- [ ] **Step 1: Write the failing placement test**

Read the frame, do not search it. The buffer must be FULL — that is the case the pinned padding was hiding.

```go
// THE GAP IS A ROW THE FRAME RESERVES, and the case that matters is a buffer with
// more text than fits: the pinned padding fakes a gap whenever there is spare
// room, so a test on a short buffer would pass with no gap at all.
func TestAFullBufferStillLeavesARowAboveThePrompt(t *testing.T) {
	var tty bytes.Buffer
	live := newPinnedScreen(&tty, 10, 40)
	for i := range 40 {
		fmt.Fprintf(live, "line %d\n", i)
	}
	tty.Reset()
	live.Draw("1-4 = pick the definition", []string{"7 of 13"})

	g := readFrame(t, tty.String(), 40)
	// The row directly above the prompt is empty, and the one above THAT is the
	// last line of the record — so the gap is one row, not zero and not two.
	if got := g.rowAt(g.promptRow - 1); got != "" {
		t.Errorf("the row above the prompt is %q, want empty — the record butts the chrome", got)
	}
	if got := g.rowAt(g.promptRow - 2); got == "" {
		t.Error("two empty rows above the prompt; the gap is one row")
	}
}
```

(`rowAt`/`promptRow` may need adding to `frameGeometry`; follow whatever `readFrame` already exposes rather than inventing a parallel reader.)

- [ ] **Step 2: Write the failing click-map test**

The gap sits between the buffer and the prompt, so `footerTop` moves with it. A footer click that lands one row out marks the WRONG WORD on a board, and a mark is irreversible — this is the row of arithmetic that must not be wrong.

```go
// footerTop COUNTS THE GAP. A click on the board's first grid row must still
// report grid row 0; off by one here is a permanent mark on a word the learner
// never pointed at.
func TestAFooterClickIsUnmovedByTheChromeGap(t *testing.T) { /* … */ }
```

- [ ] **Step 3: Run both and watch them fail**

Run: `go test ./cmd/define -run 'ChromeGap|AFullBufferStillLeaves' -v`
Expected: FAIL — no gap drawn, `footerTop` unchanged.

- [ ] **Step 4: Reserve the row**

In `screen.go`: add `gap int` to `screen` beside `pinned`, and the constant:

```go
// chromeGap is the rows a frame holds EMPTY between the record and the live edge.
//
// The live edge is a legend of what you can press; the buffer is what you are
// reading. With nothing between them the action row butts the last line of the
// definition it belongs under and the two read as one block — which is what the
// operator saw in a real sitting.
//
// A ROW THE FRAME RESERVES, never a "\n" inside the prompt: `displayRows` measures
// the prompt in visible CELLS and would charge a two-line prompt one row, making
// the frame one row too tall — the terminal scrolls and every placed row moves.
// Raw mode would also want "\r\n" and the prompt is written with neither.
const chromeGap = 1
```

In `Paint`, the gap is **taken from what is left, never assumed** — and this is the step to get right. The existing `if s.rows < 0 { s.rows = 0 }` clamp absorbs a shortfall in the BUFFER's share, so subtracting an unconditional gap and then writing it regardless makes the frame `termRows+1` rows on a tight window: the terminal scrolls, and every row the app believes it placed moves. That is precisely the failure the whole budget exists to prevent.

**The gap is FIRST in the order of sacrifice**, because it is the only component that carries no information. The prompt survives first, then footer rows, then buffer rows — and the gap goes before any of them:

```go
// grantedGap is whether a frame of this shape gets its chrome gap — and it is ONE
// owner because two consumers ask: Paint, when it draws, and the board's fit, when
// it decides whether Enter may spend a board. Two answers here means a board drawn
// whole and refused in the same breath (PQ-8).
//
// Decoration, so it is the first component given up: a frame that scrolls has lost
// every coordinate on it, and a border is not worth that. Granted only when a
// buffer row survives beside it — at that size the reader needs the content more
// than the border.
func grantedGap(want, termRows, promptRows, footerRows int) int {
	if termRows-promptRows-footerRows >= want+1 {
		return want
	}
	return 0
}
```

In `Paint`:

```go
gap := grantedGap(s.gap, termRows, promptRows, footerRows)
s.rows = termRows - promptRows - footerRows - gap
```

`fitFooter` keeps its existing budget (`termRows-promptRows`): the footer is worth more than the gap, so it is sized first and the gap takes from what remains.

Then emit `gap` blank rows after the buffer rows AND after the pinned padding, so the gap is the last thing before the prompt, and count it in the footer's origin:

```go
s.footer, s.footerTop = footer, bufRows+gap+promptRows
```

In `newPinnedScreen`: `l.s.gap = chromeGap`.

- [ ] **Step 4a: Pin the short-terminal boundary**

```go
// A frame NEVER exceeds the terminal, gap or no gap. The clamp hid this once:
// s.rows floors at zero while the gap rows were still written, so the frame came
// out one row too tall and the terminal scrolled.
func TestTheChromeGapIsGivenUpBeforeTheFrameOverflows(t *testing.T) {
	// at each height from 1 row up to comfortably tall, assert the emitted frame
	// occupies no more rows than the terminal has
}
```

Sweep a RANGE of heights rather than one — the failure is a boundary and the boundary moves with the prompt's wrapped height.

- [ ] **Step 5: Charge it in the board's fit**

```go
func fitsABoard(termRows, boardRows, promptRows int) bool {
	// The board and the bar ARE the footer, so the gap is granted against the
	// same shape Paint will grant it against — through the same function.
	footerRows := boardRows + barRows
	gap := grantedGap(chromeGap, termRows, promptRows, footerRows)
	return footerRows + promptRows + gap <= termRows
}
```

**Charged through `grantedGap`, not unconditionally, and PQ-8 is why.** `boardFitsIn` is asked at SELECTION *and* at every DRAW, where its answer decides whether Enter may spend the board (R17). An unconditional charge disagrees with `Paint` at exactly the height where `Paint` declines the gap: the existing table row `{termRows: 8, boardRows: 6, promptRows: 1} → true` would draw the board whole while the fit said false, so `boardRefusal` would replace the keys row and Enter would be held over a board with every cell on screen. Safe in direction, wrong on the screen — and the rule the family names is that **the charge and the emission must agree wherever either is consulted**.

Verify the two boundaries by hand while writing this, and pin them: at `termRows` 8 the board is whole with no gap (both answer so), and the gap first appears at 10.

`chromeGap` rather than a screen's `s.gap` is ONE owner, not two: `newPinnedScreen` is the only production caller (`play_loop.go:106`), so the sitting's screen is the only screen a board is ever drawn on. State that in the comment rather than leaving it implied — if a second pinned screen appears, this is the line that has to change with it.

- [ ] **Step 6: Delete the board's blank-buffer-line write**

In `show()`, the `if written != s.Index { fmt.Fprintln(stdout) }` arm inside the `play.Grid` branch goes. Its reasoning is now the frame's, and it says so in the atlas rather than in a special case. Keep `written = s.Index` if anything else reads it; check before deleting the assignment.

- [ ] **Step 7: Run the suite and fix the shifted expectations**

Run: `go test ./cmd/define/...`
Expected: several pinned-screen tests shift by one row. That churn is the pin working — move the expectations, and for each one ask whether it was asserting a placement (update it) or accidentally depending on the old geometry (say so in the commit).

- [ ] **Step 8: Commit**

```bash
git add cmd/define/screen.go cmd/define/play_loop.go cmd/define/screen_test.go cmd/define/play_loop_test.go
git commit -m "#44: the frame reserves a row between the record and the live edge"
```

---

### Task 3: the chrome is dimmed

**Files:**
- Modify: `cmd/define/playbar.go` (add `asChrome`)
- Modify: `cmd/define/play_loop.go` (the two `view.Draw` calls, `boardFooter`)
- Test: `cmd/define/playbar_test.go`, `cmd/define/play_loop_test.go`

- [ ] **Step 1: Write the failing test**

```go
// The band reads as chrome, and BOTH rows of it do. Dimming only the action row
// would leave the figure line brighter than the controls above it, which inverts
// what they are worth.
func TestTheChromeBandIsDimmedTogether(t *testing.T) { /* prompt and bar both carry pal.dim */ }

// And carries no escape at all when the palette is off — `--play` refuses
// -no-color (BR-3), so this is belt; a rig running colourless is exactly how
// #40's wrap Critical stayed invisible, so the styled path is what the frame
// tests drive.
func TestTheChromeBandIsPlainWithoutAPalette(t *testing.T) { /* … */ }
```

- [ ] **Step 2: Run and watch it fail**

Run: `go test ./cmd/define -run TestTheChromeBand -v`
Expected: FAIL — no escape sequences present.

- [ ] **Step 3: Implement**

```go
// asChrome styles the live edge as CHROME rather than as content.
//
// The plain text is left alone at its source: README.md quotes the prompt lines
// verbatim and doc_sync_test.go pins that, so `gradePrompt` and `sittingBar` stay
// unstyled and the dim is applied where the strings are handed to the frame.
// Escapes cost no columns and every measuring helper here skips them
// (visibleCells, clipVisible, displayRows), so nothing about the budget moves.
func asChrome(text string, pal palette) string {
	if pal.dim == "" || text == "" {
		return text
	}
	return pal.dim + text + pal.off
}
```

**Four sites, not two** — the prompt and the bar at each of the two `Draw` calls:

| site | what to style |
|---|---|
| `play_loop.go:220` | `boardPrompt(q, boardWhole)` |
| `play_loop.go:220` | the bar inside `boardFooter` |
| `play_loop.go:247` | `livePrompt(s)` |
| `play_loop.go:247` | `[]string{sittingBar(fig)}` — the COMMON sitting, and the one an earlier draft of this plan missed |

The palette comes from `newPalette(opt.color)`, from `main`, on the same seam `boardPalette` already sits on.

`boardFooter` gains a palette parameter. Its doc comment documents a load-bearing ordering — "THE FORM IS FIRST … formCell reads a footer entry index straight back as a grid row" — so the edit adds an argument and must not disturb the assembly order. Say so in the commit; a reviewer seeing that function touched will look for exactly that.

- [ ] **Step 4: Re-read the width pin**

`TestTheRefusalRowIsNoWiderThanTheKeysRow` compares two prompt strings. Confirm it compares VISIBLE width (`visibleCells`) and not `len` — if it compares bytes, it starts passing for the wrong reason the moment either string carries an escape.

- [ ] **Step 5: Run and commit**

```bash
go test ./cmd/define/...
git add cmd/define/playbar.go cmd/define/play_loop.go cmd/define/playbar_test.go cmd/define/play_loop_test.go
git commit -m "#44: the action row and the bar read as one dimmed band"
```

---

### Task 4: the docs follow

**Files:**
- Modify: `cmd/define/README.md`
- Modify: `atlas/define.md` ("The screen" section)

- [ ] **Step 1: README**

The board block is DERIVED (#40 BR-18) — regenerate rather than hand-editing it. The prompt lines are quoted verbatim and unchanged, since the plain text did not move.

- [ ] **Step 2: Atlas**

Under "The screen", the frame-budget passage gains the gap as a component and loses the board's special case. Two claims to state rather than imply: that the gap is a reserved ROW (with why a newline in the prompt is wrong), and that `footerTop` counts it, because that is the one arithmetic whose failure is a permanent mark on the wrong word.

Add the indicator rule beside the existing `eraseLine` paragraph: inside a screen a newline is content, so a screen-hosted playback carries no `before`.

- [ ] **Step 3: Run the doc pins and commit**

```bash
go test ./cmd/define -run 'README|Atlas' -v
git add cmd/define/README.md atlas/define.md
git commit -m "#44: docs — the gap, and the indicator's missing before"
```

---

## Verification

- [ ] `go test ./...` green.
- [ ] **`go test -tags conformance ./cmd/define`** green. `pty_conformance_test.go` is `//go:build darwin && conformance`, so the plain run cannot even COMPILE it — and its SGR-1006 click is the only end-to-end proof that a real terminal's click still maps to the intended cell after `footerTop` moved. This plan calls that the arithmetic that must not be wrong; running the suite that checks it is what makes the claim more than an assertion. (`#37` is open on exactly this: these tags run in nothing automated.)
- [ ] `go vet ./...` clean.
- [ ] A real sitting: `go build -o define ./cmd/define && ./define --play` — click a word several times and confirm the frame does not drift; confirm the band reads as chrome; confirm a board still draws whole.
- [ ] Done-when rows in `workshop/issues/000044-play-chrome.md` ticked with the evidence that ticked them.


## Revisions

### 2026-09-02 — `fitsABoard` is not modified after all

**Delta:** the Core-concepts table listed `fitsABoard` as `modified` and Task 2
Step 5 rewrote it to charge `chromeGap`. It is now unchanged, and the deliverable
is a proof plus a pin instead.

**Reason:** working the arithmetic through while implementing, charging the gap
there is *provably equivalent* to not charging it. `grantedGap` hands out the row
only when `T - P - F >= 2`, which gives `F + P + 1 <= T - 1`; so whenever the gap
exists the un-charged sum already had slack for it, and whenever it does not the
sum is unchanged. Writing the term would have altered no answer while LOOKING
like the two consumers had been reconciled — worse than leaving it out, because
the next reader would trust the appearance.

PQ-8's actual requirement — that the charge and the emission agree everywhere
either is consulted — is met more strongly this way: they agree *by construction*
rather than by two formulas being kept in step. `TestTheChromeGapNeverChangesWhetherABoardFits`
exhausts the shape space (0–40 × 1–30 × 1–5) so the equivalence is a pin rather
than a claim in a comment, and `fitsABoard`'s doc says why the term is absent so
nobody adds it back as a "fix".

Caught by `TestPlanTableStatusMatchesTheChangeWindow`, which failed on the row
describing work that did not happen.

---
id: 000044
status: working
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
started: 2026-09-02T12:50:12-07:00
---

# the play frame's chrome: separate it, colour it, and stop it growing a line per click

## Problem

Three findings from one operator sitting, 2026-09-02, all on the same surface:
the band at the bottom of a `--play` frame where the record stops and the
controls begin.

**1. Nothing separates the action row from the form.** `Paint` stacks buffer
rows, then the prompt, then the footer, with no gap — so
`1-4 = pick the definition, d = remove from deck, Ctrl-C to stop` butts directly
against the last line of the definition it belongs under. On a short entry the
pinned buffer's padding fakes a gap; on an entry that fills the screen there is
none, and the two read as one block. Operator: *"there should be a blank line
between the action footer and forms."*

**2. The action row is the same weight as the text above it.** Both are plain,
unstyled ASCII, so nothing marks where the thing you READ ends and the thing you
PRESS begins. Operator: *"'1-4 = pick the definition ...' should be colorized to
make the border clear."* Every other surface in this program is already coloured
— the definition, the highlighted deck words, the board's marks — which makes
the chrome the one unstyled thing on screen.

**3. A click-to-play grows the buffer by one blank line, every time.** Operator:
*"after clicking on pronunciation in the daily play, one additional line's
inserted"*, with a screenshot of a sitting whose frame had drifted up.

The mechanism, traced:

- `defaultIndicator` (`main.go:824`) is `{show: true, before: "\n", erase: eraseLine}`.
- `playAnnounced` writes `ind.before`, then `  ♫ playing 3×` with NO trailing
  newline, then `ind.erase`.
- `screen.Write` splits on `eraseLine` and calls `eraseOpenLine`, which drops the
  **open** line only — deliberately, since "a completed line is scrollback".
- So the erase takes back the indicator's own partial line, and the `"\n"` that
  preceded it stays as a committed blank line. One per click, forever.

`before: "\n"` is cursor positioning, which is what it meant on a cooked
terminal. **Inside a screen a newline is CONTENT**, and the buffer is
append-only. The editor's click path already knows this — `replraw.go:343` passes
`indicator{show: true, erase: eraseLine}` with no `before` at all — so there are
two spellings of "the indicator, inside a screen" and only one of them is right
(ARCH-DRY). The sitting got the other one, under a comment claiming
`defaultIndicator` is "what every other playback on this path already uses"
(`play_loop.go:356`), which is true of the one-shot path and false of the screen.

## Spec

**One owner for "the chrome is separated from the record above it, and looks
like chrome."**

### The live prompt carries its own blank line and its own style

`gradePrompt(q)` stays PLAIN and unchanged — `README.md` quotes it verbatim and
`TestREADMEQuotesThePromptsTheLoopActuallyPrints` pins that, so styling it in
place would either break the pin or push escape sequences into the README.

The blank line and the colour go on a new seam applied where the prompt is
handed to the frame: plain text in, `"\n"` + styled text out. Both consumers read
it, which is the point:

- `view.Draw(...)` — so the frame shows it;
- `boardFitsIn` — so the row budget CHARGES it. `Paint` already measures
  `displayRows(prompt, cols)` and `fitsABoard` already takes a measured
  `promptRows`, so a blank line inside the string is costed automatically by
  machinery that exists. Nothing gets a new constant.

**This deletes the board's special-case blank.** `show()` writes one blank buffer
line the first time a board is drawn (`play_loop.go`, guarded by `written !=
s.Index`) because "a board writes nothing else to the buffer, so without it the
grid begins immediately under the previous question's last line". The prompt sits
between that buffer and the grid, so a prompt that carries its own leading blank
makes the board's copy redundant — one rule for every form instead of a rule plus
an exception.

### The chrome is the prompt row AND the bar

Dimmed together, as one band. Dimming only the action row would leave the figure
line below it brighter than the controls above it, which inverts their
importance — the bar is a number you glance at, the action row is what you press.
Recorded as a decision the operator can reverse in a sitting: if the bar should
stay bright, it is a one-line change.

Through `newPalette`'s existing `dim`, from `main`, because `main` owns the
terminal's colours — the same seam `boardPalette` sits on. Derived from
`opt.color` rather than assumed: `--play` refuses `-no-color` (BR-3), so this is
belt, but a rig that runs colourless is exactly how #40's wrap Critical stayed
invisible, so the styled path is what the tests must drive.

### The indicator inside a screen has no `before`

The sitting's click path takes the editor's shape. Rather than copying the
literal a third time, the two screen paths share one named constructor — the
screen is where the rule "a newline is content, not cursor movement" is true, so
that is what the name says. `defaultIndicator` keeps `before: "\n"` for the
one-shot and piped paths, where the cursor genuinely has to move.

## Done when

- [ ] A frame whose buffer fills the screen still shows one blank line between the last content row and the action row, pinned by a frame test that reads the placement rather than searching for a substring.
- [ ] The action row and the bar carry the dim style in a coloured sitting, and neither carries an escape sequence when the palette is off.
- [ ] The blank line is CHARGED: a board offered at a given height is still drawn whole, pinned by a test at the boundary height where one uncharged row would overflow.
- [ ] Clicking a word N times in a sitting leaves the buffer the same height it was, pinned by a test that clicks more than once — one click passing is what a `before` bug looks like when the count is one.
- [ ] The board's own blank-buffer-line special case is gone, and a board still reads as separated from the question above it.
- [ ] `README.md` still quotes the prompt lines verbatim and the doc pin still passes — the plain text is unchanged.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-09-02

Filed mid-planning on `#42`, from three operator messages during one sitting.
Sequenced AHEAD of `#42` on the operator's call: the blank line changes the row
budget (`fitsABoard` charges the prompt's measured height) and `#42` reworks
exactly that arithmetic, so doing chrome first means the fit math is written once
against the final chrome. `#42` is parked `blocked` on this.

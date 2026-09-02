---
id: 000044
status: working
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours: 2.16
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

**3. Playback grows the buffer by one blank line, every time.** Operator:
*"after clicking on pronunciation in the daily play, one additional line's
inserted"*, with a screenshot of a sitting whose frame had drifted up.

**A click is where the operator SAW it, not where it lives.** The plan gate
(PQ-1/PQ-2) found the same defect on the sitting's ordinary reveal playback
(`play_loop.go:507`) and in the editor's own submit path (`replraw.go:653`,
`before: "\r\n"`, which `screen.Write` normalises to `"\n"` and then commits). So
it is one blank line per PLAYBACK — per answered question in a sitting, per
looked-up word in the editor — and a click only made it repeatable on one word.
The first draft of this issue said "one per click" and claimed the editor
"already knows this"; both were generalisations from the one site that had been
looked at.

The mechanism, traced:

- `defaultIndicator` (`main.go:824`) is `{show: true, before: "\n", erase: eraseLine}`.
- `playAnnounced` writes `ind.before`, then `  ♫ playing 3×` with NO trailing
  newline, then `ind.erase`.
- `screen.Write` splits on `eraseLine` and calls `eraseOpenLine`, which drops the
  **open** line only — deliberately, since "a completed line is scrollback".
- So the erase takes back the indicator's own partial line, and the `"\n"` that
  preceded it stays as a committed blank line. One per playback, forever.

`before: "\n"` is cursor positioning, which is what it meant on a cooked
terminal. **Inside a screen a newline is CONTENT**, and the buffer is
append-only.

**Five call sites write playback into a screen, and they disagree three ways**
— which is the actual defect, the blank lines being its symptom:

| site | passes | |
|---|---|---|
| `play_loop.go:356` — click | `defaultIndicator(opt)` | commits a blank |
| `play_loop.go:507` — reveal | `defaultIndicator(opt)` | commits a blank |
| `replraw.go:653` — submit | literal, `before: "\r\n"` | commits a blank |
| `replraw.go:343` — click | literal, no `before` | correct |
| `replraw.go:633` — `/pron` | literal, no `before` | correct |

Three copies of a literal and two of a constructor meant for a different
surface, with nothing naming the rule they are all instances of (ARCH-DRY). The
comment at `play_loop.go:356` even states the wrong one — `defaultIndicator` is
"what every other playback on this path already uses" — which is true of the
one-shot path and false of every screen.

## Spec

**One owner for "the chrome is separated from the record above it, and looks
like chrome."**

### The gap is a RESERVED ROW in the frame, not a newline in a string

The obvious move — prefix `"\n"` to the prompt — is wrong twice over, and both
are the frame arithmetic this program has already been burned by:
`Paint` measures the prompt with `displayRows(prompt, cols)`, which counts
VISIBLE CELLS and knows nothing about an embedded newline, so a two-line prompt
would be charged one row and the frame would be one row too tall — the terminal
scrolls, and every row the app believes it placed moves. And `Paint` writes the
prompt with a bare `WriteString`, where in raw mode a `\n` moves down without
returning the carriage.

So the gap is a row the frame RESERVES, in the one accounting that already
budgets every component:

- `s.rows` (the buffer's share) gives the row up, exactly as it already gives
  rows to the prompt and the footer;
- the gap is emitted after the buffer rows and before the prompt;
- **`s.footerTop` includes it**, or every footer click lands one row out — this
  is the part that must not be got wrong, because on a board a mis-mapped click
  marks the WRONG WORD and the mark is irreversible;
- `fitsABoard` charges it, so a board is still only offered when it can be drawn
  whole.

The cursor walk-back (`footerRows+promptRows-1`) is unaffected: the gap is above
the prompt, and the walk-back only climbs from the footer to the prompt's first
row.

**It belongs to the SITTING's screen, not to every screen.** `newPinnedScreen`
already exists for exactly this distinction — "the two surfaces want opposite
things and the difference should be visible where the screen is BUILT" — so the
gap is set there, beside `pinned`, and named for what it is rather than folded
into `pinned`'s meaning. The editor's frame is untouched: its prompt is the line
you are TYPING, a continuation of what is above it, where the play frame's prompt
is a legend of what you can press. Different things, and only one of them wants a
border. (If the operator wants the editor to adopt it too, it is one line at the
other constructor.)

**This deletes the board's special-case blank.** `show()` writes one blank buffer
line the first time a board is drawn (`play_loop.go`, guarded by `written !=
s.Index`) because "a board writes nothing else to the buffer, so without it the
grid begins immediately under the previous question's last line". That is this
same gap, discovered once for one form and paid for out of the append-only
buffer. A reserved frame row makes it every form's, and makes it a row that
cannot end up in the exit transcript.

### The style goes on the strings; the plain text is untouched

`gradePrompt(q)` and `sittingBar(f)` stay PLAIN — `README.md` quotes the prompt
lines verbatim and `TestREADMEQuotesThePromptsTheLoopActuallyPrints` pins that,
so styling them in place would either break the pin or push escape sequences into
the README. The dim is applied where the strings are handed to `Draw`.

Escapes cost no columns and every measuring helper here already skips them
(`visibleCells`, `clipVisible`, `displayRows`), so styling changes no arithmetic.
The one pin that must be re-read is
`TestTheRefusalRowIsNoWiderThanTheKeysRow`, which compares two prompt strings and
must keep comparing VISIBLE width.

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

### The indicator inside a screen has no `before` — and a guard says so

All five sites take one named constructor, `screenIndicator()`. The screen is
where the rule "a newline is content, not cursor movement" is true, so that is
what the name says. `defaultIndicator` keeps `before: "\n"` for the one-shot and
piped paths, where the cursor genuinely does have to move.

**A sweep is not a fix — the rule gets a guard.** Five sites drifted three ways
precisely because nothing enforced them; correcting three of them by hand leaves
the sixth free to be written wrong (ARCH-PURPOSE: the deliverable is the CLASS,
not the instances). So a source-level test enumerates every `playAnnounced` call
in the screen-hosted files and fails on one that does not pass
`screenIndicator()` — the same shape as this package's existing purity guard (an
import allowlist plus a wall-clock grep) and its doc-sync tests, and the reason
those keep working where hand-maintained enumerations did not.

## Done when

- [ ] A frame whose buffer fills the screen still shows one blank line between the last content row and the action row, pinned by a frame test that reads the placement rather than searching for a substring.
- [ ] The action row and the bar carry the dim style in a coloured sitting — at BOTH bar sites, the board's and the common one — and neither carries an escape sequence when the palette is off.
- [ ] The gap is CHARGED where a board is offered and SACRIFICED FIRST where a frame is drawn, pinned at both boundary heights: the height where one uncharged row would push a board off the bottom, and the height where an unconditional gap would make the frame taller than the terminal.
- [ ] A sitting that plays audio N times leaves the buffer the height it was — driven through the REVEAL path as well as the click path, since the reveal is the one that fires on every question.
- [ ] No `playAnnounced` call in a screen-hosted file passes anything but `screenIndicator()`, enforced by a test rather than by having swept the five that exist today.
- [ ] The board's own blank-buffer-line special case is gone, and a board still reads as separated from the question above it.
- [ ] `README.md` still quotes the prompt lines verbatim and the doc pin still passes — the plain text is unchanged.
- [ ] `go test -tags conformance ./cmd/define` passes: the pty suite's SGR-1006 click is the only end-to-end proof that `footerTop` still maps a real terminal's click to the intended cell.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Design is light throughout because the plan doc resolved the decisions — the
`design-buffer` is 0.15 rather than 0.30 for exactly that reason. The two
expensive rows are the frame budget (an arithmetic change on the surface where a
wrong row means a permanent mark on the wrong word) and the test churn behind it:
`newPinnedScreen` has ~30 test call sites and the gap moves every placement
expectation among them.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: cross-cutting-refactor   design=0.15 impl=0.14
item: smaller-go-module        design=0.1  impl=0.2
item: tui-screen               design=0.4  impl=0.36
item: cross-cutting-refactor   design=0.05 impl=0.16
item: smaller-go-module        design=0.05 impl=0.12
item: atlas-docs               design=0.1  impl=0.06
item: milestone-review         design=0.0  impl=0.14
design-buffer: 0.15
total: 2.16
```

| row | what it is |
|---|---|
| `cross-cutting-refactor` #1 | the five-site indicator sweep + deleting `playRegion`'s parameter |
| `smaller-go-module` #1 | the AST guard, matching on argument type |
| `tui-screen` | `chromeGap`: `Paint`'s budget, the sacrifice order, `footerTop`, `fitsABoard` |
| `cross-cutting-refactor` #2 | the shifted placement expectations across the pinned-screen tests |
| `smaller-go-module` #2 | `asChrome` at its four sites |
| `atlas-docs` | README (derived block regenerated) + the atlas's screen section |
| `milestone-review` | the one boundary review at close |

## Plan

Durable design: `workshop/plans/000044-play-chrome-plan.md`.

Single-pass: one review boundary, so plain checkboxes rather than `Mx` tags.

- [ ] The indicator inside a screen has no `before` — `screenIndicator()`, shared by the sitting's click path and the editor's, and a test that clicks TWICE.
- [ ] The frame reserves `chromeGap` between the record and the live edge — `Paint`'s budget, its `footerTop`, and `newPinnedScreen`; pinned by a placement test on a FULL buffer and a footer-click test.
- [ ] `fitsABoard` charges the gap, so a board is still offered only when it can be drawn whole.
- [ ] The board's blank-buffer-line special case is deleted, its reasoning now the frame's.
- [ ] `asChrome` dims the action row and the bar together, leaving `gradePrompt`/`sittingBar` plain so the README pin still holds.
- [ ] README + atlas follow; the derived board block is regenerated rather than hand-edited.

## Log

### 2026-09-02

Filed mid-planning on `#42`, from three operator messages during one sitting.
Sequenced AHEAD of `#42` on the operator's call: the blank line changes the row
budget (`fitsABoard` charges the prompt's measured height) and `#42` reworks
exactly that arithmetic, so doing chrome first means the fit math is written once
against the final chrome. `#42` is parked `blocked` on this.

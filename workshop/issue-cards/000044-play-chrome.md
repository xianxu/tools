---
id: '000044'
status: done
started: 2026-09-02T12:50:12-07:00
created: 2026-09-02
updated: 2026-09-02
estimate_hours: 2.16
actual_hours: 4.60
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

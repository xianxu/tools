---
id: 000069
status: open
deps: []
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
---

# define: pin the looked-up word while its entry is still on screen

## Problem

A session's buffer is the whole session. `screen.lines` is *"everything the
session has shown, oldest first"* (`screen.go:30`), so after looking up `zenith`,
`nadir` and `meridian` the reader scrolls back through three NOAD entries — and a
NOAD entry runs several screenfuls. The headword is line 0 of its own render
(`render.go:390`), so it is the FIRST thing off the top: you end up reading a
sense list with no indication whose it is.

`#41` fixed the same complaint for the sitting, and only for the sitting — *"A
long reveal PAGES instead of scrolling the word away … before frames the word
being asked about was simply gone off the top"* (`atlas/define.md`). The editor
got the shared viewport gestures out of that work (`viewportGesture`,
`replraw.go:156`) but never got the thing those gestures were added to protect.

## Spec

A sticky section header, one row, at the top of the viewport. While the viewport's
top line falls inside entry E, the row shows E's headword. When E's last line
scrolls past, the next entry's headword takes the row — displaced by the incoming
one rather than blanked between them, so the row is never empty mid-buffer.

Three constraints, each load-bearing and each already decided elsewhere in this
file. They are the reason this is not a one-row `Paint` change.

**1. The header is NOT a buffer line.** `Lines()` is the exit transcript (D3) and
a header written into the buffer files a duplicate headword — one per scroll
position — into the record. The established shape for chrome that is not content
is `pinned` and `gap`: *"blank rows emitted at PAINT time, never lines appended to
the buffer: the transcript and the click map must not gain rows that exist only
because the terminal is tall"* (`screen.go:30`). The header follows that rule.

**2. It breaks the click invariant unless Paint reports where it landed.** The
whole reason this program took the alternate screen is that *"a click at viewport
row R is buffer line `R + offset` by construction rather than by tracking
something the app never observes"* (`screen.go:18`). A header occupying row 0
makes that off by one for every row under it, and row 0 maps to no buffer line at
all. That failure has a recorded shape here already — *"the click map detached
from the text: the underline painted on one row while the region answered on
another"* (`visible()`, `screen.go:~275`) — and a recorded fix: `footer` /
`footerTop` are *"WHERE THE LIVE EDGE ENDED UP, recorded by the last Paint so a
click can be resolved against it (#40 D10)"*. The header needs that counterpart,
and `RegionAtRow` (`screen.go:958`) must subtract it. `FooterRowAt`
(`screen.go:247`) is the query to model it on, including its doctrine: answered
from the LAST PAINT, *"because that is what the person clicking was looking at"*.

**3. The row comes OUT of the viewport budget, never added to it.** `visible()`
slices `s.rows`; the header takes its row there. A frame one row too tall makes
the terminal scroll, which moves every coordinate — the failure mode the atlas
records four separate times.

**What is a block, and the one open question.** The buffer has no section
structure: `regions` is a sparse per-line map (`screen.go:158`) and nothing
records that lines 40–96 are `zenith`. But `RegionHeadword` already marks each
entry's first line — `render.go:390` emits it at `Line: 0` of the entry's own
render and `addRegions` resolves that to an absolute buffer line — so *the owner
of line L is the last `RegionHeadword` at or before L* may need no new bookkeeping
at all.

The question that decides the design: **a lookup writes more than its entry.**
`define: … no dictionary entry` routes through the screen rather than past it, the
ephemeral indicator opens and erases a line, and commands print. Those lines sit
between entries and would inherit the previous word's header. Either that is
correct — they belong to the lookup that produced them — or a block needs an
explicit end and the header must be able to show nothing. Answer this before
building; it is the difference between reusing `RegionHeadword` and introducing a
block type.

**Non-goals, stated so the second consumer is a row rather than a surprise.**
`--play` and the board (`#40`) share `viewportGesture`, `newConsole` and
`wrapWritten` and have the same problem. This issue does the EDITOR only. Shape
the mechanism so the sitting can call it, but the sitting's own chrome —
`sittingBar`, the pinned footer, `chromeGap` — is out of scope here.

## Done when

- Scrolling back through a multi-lookup session shows the owning headword on the
  top row for every scroll position inside that entry, and the next headword
  displaces it when the entry's last line passes.
- The header is absent from `Lines()`, and the exit transcript of a scrolled
  session is byte-identical to the same session never scrolled. Tested.
- A click resolves to the same region with the header present as without it, at
  every viewport row — the regression `visible()`'s doc comment describes, pinned
  by a test that fails if the header's row is not subtracted.
- The painted frame is never taller with the header than without it, asserted at
  the `rows` boundary rather than by eyeballing a terminal.
- The between-entries question above is answered in `## Log` with the case that
  decided it, and the chosen behavior has a test whose fixture is a `define: … no
  dictionary entry` note between two successful lookups.
- `--play` and the board are unchanged, asserted rather than assumed.
- Unit-tested with no pty, like the rest of `screen` — *"Write and Frame do no
  terminal IO"* (`screen.go:28`) and this must not be the thing that changes it.

## Plan

Needs a brainstorm on the block question before a durable plan.

- [ ] brainstorm: does a block end, or does everything after a headword belong to it
- [ ] `sdlc claim`, then `sdlc start-plan`, then the durable plan

## Log

### 2026-09-16

Filed from a session in `brain`. Surface chosen by the operator: **editor REPL
only**, out of editor / sitting / shared-layer.

Facts gathered while filing, so the plan does not re-derive them: the click
invariant (`screen.go:18`), the paint-time-chrome precedent (`pinned` and `gap`,
`screen.go:30`), the live-edge-position precedent (`footer`/`footerTop`, `#40`
D10), the last-paint doctrine (`FooterRowAt`, `screen.go:247`), and
`RegionHeadword` at `Line: 0` (`render.go:390`) as the candidate block boundary
that needs no new structure.

Two things checked at `deckwords.go:114-121` that the plan must not re-learn:

- **Line 0 is not universal.** In a `Choice` prompt *"`RegionHeadword` sits at line
  1 column 0 and the headword is a deck word, so both producers emit there"* —
  two producers on one line, resolved by `mergeRegions` precedence. So "the last
  `RegionHeadword` at or before L" is still sound as a boundary rule, but "the
  block starts at the headword's own line" is not; the entry may begin a row above
  it.
- **The region already carries what the bar should say.** *"A `RegionHeadword`
  carries the entry's LOOKUP KEY … `define jalapeno` renders `jalapeño`"* — so the
  header has both the key and the rendered form available, and should decide
  deliberately which one it shows. Showing the key would put `jalapeno` in the bar
  above an entry headed `jalapeño`.

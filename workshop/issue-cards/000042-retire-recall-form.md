---
id: '000042'
status: done
started: 2026-09-02T12:31:27-07:00
created: 2026-09-01
updated: 2026-09-04
estimate_hours: 3.89
actual_hours: 10.85
---

# retire form 2.1: the board is the fallback when a real test cannot be built

## Problem

**Form 2.1 and the board are the same instrument, and the board is sixteen times
cheaper.** Both implement `SelfRated`: `y` means "I knew it" and nothing checked.
Neither retrieves anything. 2.1 spends a whole screen and a keystroke per word to
collect a claim the board collects sixteen at a time.

Operator, 2026-09-01, from a real sitting — with the board on screen and form 2.1
having just asked about `comports`: *"with the better recall form, this is not
useful anymore I think?"*

**Its remaining population is already small and is the worst place for
self-report.** `#40`'s `boardsFor` partitions on the box FIRST, so form 2.1 now
fires only for a word that is BOTH box ≤ 2 AND one the deck cannot build
distractors for — a young deck's first sittings. Self-report is least reliable
exactly there, because the word has not been learned yet and the illusion of
knowing is strongest.

**The one thing 2.1 does that the board does not is REVEAL THE WHOLE ENTRY on a
miss.** `Recall.Reveal()` returns the full rendered entry; the board's panel is
one line, deliberately — a twenty-row panel would mean a board no terminal fits
(`#40` R1). That is the trade this issue accepts.

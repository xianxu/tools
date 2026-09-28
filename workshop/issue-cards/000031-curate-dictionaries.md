---
id: '000031'
status: done
started: 2026-08-29T11:29:22-07:00
created: 2026-08-29
updated: 2026-08-29
estimate_hours: 1.81
actual_hours: 2.09
---

# curate the Italian dictionary so /lang it is a real mode (French and German split to #34)

## Problem

`#29`'s Spec promised this and nothing tracked it:

> Adding them to `curated` in `cmd/define/dictselect.go` is **one line each** and
> makes `/lang fr|it|de` work as a full mode. That is worth doing and is NOT this
> issue: it serves someone learning those languages, which the operator explicitly
> is not. Split out so the cheap win is not blocked on this design.

The split was correct — `#29`'s purpose was origin pronunciation without a mode
switch, and curating dictionaries serves the opposite need. But a promised split
with no tracker item evaporates; `#29`'s close review (BR Minor,
`split-out-not-filed`) found it surviving only as prose in `#30`.

---
id: 000077
status: open
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
---

# define: confirm light detection in a real light terminal

## Problem

#70's detection (OSC 11 asked at raw-mode entry, the reply read as a
`KeyBackground`) is proven live only for a DARK terminal: on 2026-09-18 the
operator's terminal, after `/scheme auto`, reported `scheme dark (detected)`.
LIGHT detection is evidenced only by the modelled pty terminal
(`TestPTYBackgroundDetection/light`) and the in-process reply
(`TestRawEditorBackgroundReplyRepaints`) — not by a real terminal. This is the
ARCH-MOCK live conformance check #70 owes (see #70's Log and the atlas's
*Terminals checked by hand*).

## Spec

In a real terminal set to a LIGHT appearance: run `define`, `/scheme auto`, then
`/scheme`. Record the terminal app, its appearance, and the exact report line in
the atlas's *Terminals checked by hand*. `light (detected)` closes this; any other
answer (`dark (detected)` on a light background, or `(default: …)`) is a
detection bug or an unanswering terminal — record it and open the fix.

## Done when

- The atlas records a real light terminal's `/scheme` line after `/scheme auto`,
  with the terminal app named, and the "Owed" sentence there is removed.

## Plan

- [ ]

## Log

### 2026-09-18

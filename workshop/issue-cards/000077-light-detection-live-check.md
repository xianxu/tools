---
id: '000077'
status: done
started: 2026-09-18T11:29:09-07:00
created: 2026-09-18
updated: 2026-09-18
actual_hours: 0.38
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

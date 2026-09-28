---
id: '000055'
status: done
started: 2026-09-13T12:39:30-07:00
created: 2026-09-13
updated: 2026-09-13
estimate_hours: 1.09
actual_hours: 0.47
---

# define: wrap streamed LLM responses to terminal width

## Problem

LLM answers in the interactive `define` screen extend past the right edge and
are clipped. The supplied screenshot shows long paragraphs cut off mid-sentence.
`runAsk` streams through `highlightWriter` into `liveScreen.Write`; the latter
deliberately wraps only pinned (`--play`) screens. Dictionary rendering already
wraps, but streamed answers have no corresponding step.

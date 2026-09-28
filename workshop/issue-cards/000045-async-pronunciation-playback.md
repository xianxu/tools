---
id: 000045
status: open
created: 2026-09-03
updated: 2026-09-03
estimate_hours:
github_issue:
---

# Play pronunciation without blocking the UI

## Problem

Pronunciation playback blocks the interactive loops. `playAnnounced`
(`cmd/define/main.go:892`) calls `speak` synchronously; `speak`
(`main.go:991`) fetches the recording over the network, writes it to a temp
file, and calls `playN`, which runs `afplay` to completion —
`exec.CommandContext(...).Run()` (`player.go:36`) — with a 250ms gap between
repeats. At the default repeat count the terminal is unresponsive for the whole
fetch plus several seconds of audio.

It blocks in seven places: the REPL (`repl.go:316`), the raw REPL
(`replraw.go:608,636,654`), the review loop (`play_loop.go:506`) and the
one-shot path (`main.go:720`).

---
id: '000053'
status: done
started: 2026-09-12T15:55:34-07:00
created: 2026-09-12
updated: 2026-09-12
estimate_hours: 0.49
actual_hours: 1.71
---

# define: /help history prints the whole list, so no command explains its arguments

## Problem

`/help` throws its arguments away (`runHelp(c commandCtx, _ []string)`,
`cmd/define/command.go:265`), so `/help history` prints the same list as a bare
`/help`. `/history --help` is worse: it is read as a number of days and fails with
`define: /history: "--help" is not a number of days` (exit 2).

A command's arguments are written down only in the README and the atlas, which say
so on purpose: the summary is what `/help` prints, and syntax would stop it
matching the screen. That leaves nowhere on screen that explains them. A user at
the prompt cannot learn that `/history` takes `N`, `--days N` or `--days=N`, or
that `/sound`, `/lang` and `/pron` take an argument at all.

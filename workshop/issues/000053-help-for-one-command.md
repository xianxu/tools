---
id: 000053
status: open
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours:
---

# /help history prints the whole list, so no command explains its arguments

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

## Spec

- **Each row of the `commands` registry (`command.go:25`) gains a usage text.**
  The registry owns it, and the README and atlas quote it inside marked spans,
  pinned by `doc_sync_test.go` the way `TestDocsQuoteTheCommandList` pins the
  summary table (ARCH-DRY). `pronCommandHelp` (`pron_cmd.go:98`) is already
  `/pron`'s usage and already quoted this way; it becomes that row's usage rather
  than a second copy.
- **`/help <command>` prints that command's usage**, with or without the slash
  (`/help history`, `/help /history`).
- **`/help <not-a-command>` suggests the nearest command** through
  `nearestCommands`, in the wording dispatch already uses for an unknown command
  (`command.go:257`), not a second phrasing.
- **`/<command> --help` prints the same usage** instead of failing on it. Route it
  once in dispatch rather than in each command's argument parser: one owner, not
  seven.
- **Bare `/help` says `/help <command>` exists.** Otherwise the feature is
  invisible, the same reason `runHelp` already gives for listing the `?` and `\`
  hatches.
- **The command line gets it for free** (`define /help history`): it is the same
  dispatcher.

Usage each command needs: `/help [command]`; `/history [N | --days N |
--days=N]` (default 2 days); `/sound [N]`; `/lang [language]`; `/pron
[language]`. `/stats` and `/play` take nothing, and their usage says so.

Open, for the plan: completing the argument (`/help hi` + Tab → `/help
history`). `menuLines` stops at the first argument on purpose ("an argument means
the command is settled"), so this would be a deliberate change to that rule. It is
separable.

Constraint: `cmd/define/README.md` has an uncommitted rewrite in progress by the
operator (2026-09-12). Start the README part after it lands, not over it.

## Done when

- [ ] `/help history` prints `/history`'s usage, in the TUI and as
      `define /help history`, asserted by a test.
- [ ] Every registered command has a usage text, and a test fails when a new row
      lacks one.
- [ ] `--help` after any command prints its usage, asserted over the whole
      registry rather than one example.
- [ ] `/help nosuch` suggests the nearest command, in dispatch's wording.
- [ ] The README and atlas quote each usage from the registry, pinned by a
      doc-sync test; `pronCommandHelp` is folded in, not duplicated.
- [ ] Bare `/help` mentions `/help <command>`.

## Plan

- [ ] Wait for the README rewrite in progress to land.
- [ ] `sdlc claim`, then `sdlc start-plan`. Single pass: plain checkboxes, no Mx.

## Log

### 2026-09-12

Asked whether `/help history` is supported. Built main and ran it in an empty
directory: `define /help history` and `define /help` print identical output, and
`define /history --help` exits 2 with the day-count error.

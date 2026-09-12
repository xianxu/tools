---
id: 000053
status: working
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours:
started: 2026-09-12T15:55:34-07:00
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

Detailed plan: `workshop/plans/000053-help-for-one-command-plan.md`. Single pass:
plain checkboxes, no Mx.

- [ ] Task 0 — side-quest: the README rewrite (`6cf5414`, carried by this
      branch) keeps the text the nine doc-sync tests pin; `go test ./cmd/define/`
      green.
- [ ] Task 1 — every `commands` row carries `args` and `usage`, each text beside
      its parser, limits taken from the parsers' constants.
- [ ] Task 2 — `/help <command>` explains one (`findCommand`, `unknownCommand`,
      `usageText`); bare `/help` says so; `/help`'s summary and both command-list
      spans updated.
- [ ] Task 3 — `--help` and `-h` answered in `dispatchCommand` for every command,
      one-shot included.
- [ ] Task 4 — the README and atlas quote the generated `command-usage` span;
      `pronCommandHelp`, its span and its test absorbed and removed.
- [ ] Task 5 — mutation-verify each guard; full suite, vet, gofmt; run the binary
      and record the output.

## Log

### 2026-09-12

Asked whether `/help history` is supported. Built main and ran it in an empty
directory: `define /help history` and `define /help` print identical output, and
`define /history --help` exits 2 with the day-count error.

Planning, after `sdlc claim` and `sdlc start-plan`. The branch starts from the
operator's README rewrite (`6cf5414`, on `define-readme-rewrite`), which fails
nine doc-sync tests in `go test ./cmd/define/`:
`TestEverySurfaceDescribingCaptureMentionsTheQuestion` (no `-here`),
`TestTheREADMEQuotesTheRealPrompt` (no `Create one here? [y/N]`),
`TestEverySurfaceNamesEveryCuratedLanguage` (no curated-languages span),
`TestREADMEQuotesThePromptsTheLoopActuallyPrints` (the two graded prompts),
`TestDocsQuoteThePronHelp`, `TestDocsQuoteTheLocaleHelp`,
`TestREADMEKeyTableNamesEveryLiveKey` (no review-keys span), and
`TestREADMEAnchorsResolve` (1 in-page link, floor 3; its one link also points at
a heading the rewrite removed). Task 0 fixes them.

## Revisions

### 2026-09-12 — planning

- **The README constraint is met differently.** The Spec said to start the README
  part after the operator's rewrite lands. The operator committed the rewrite on
  `define-readme-rewrite` (`6cf5414`) and asked for #53 next. The rewrite fails
  nine doc-sync tests, so it cannot land on its own: this branch carries it,
  Task 0 makes it green, and one PR lands both.
- **Argument completion stays out.** The Spec left `/help hi` + Tab open. The
  purpose is a command that explains itself; completion is a separable extension.
- **Two fields, not one.** Each row gets `args` (the synopsis) and `usage`
  (prose), so the terminal and the docs build the synopsis the same way and the
  docs can set it as code.

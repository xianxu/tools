---
id: 000053
status: working
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours: 0.49
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

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module   design=0.03 impl=0.16
item: atlas-docs          design=0.04 impl=0.11
item: milestone-review    design=0.0 impl=0.14
design-buffer: 0.15
total: 0.49
```

- smaller-go-module: the registry fields and usages, `/help <command>`, the
  `--help` routing and their tests. v2 design 0–0.3 at 0.15, × 0.2 because the
  plan pre-resolves every decision = 0.03. v2 impl 0.2–0.5 at 0.4 (seven new
  tests and a registry-wide sweep) × 0.4 = 0.16.
- atlas-docs: two passes, Task 0's README restoration and Task 3's usage span in
  two docs. Design 2 × (0.1 × 0.2) = 0.04; impl 2 × 0.14 × 0.4 ≈ 0.11.
- milestone-review: one boundary, since this is single-pass work. v2 impl 0.2–0.5
  at 0.35 × 0.4 = 0.14.
- Design buffer +15% for a thorough plan doc. 0.07 × 1.15 + 0.41 = 0.49.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

## Plan

Detailed plan: `workshop/plans/000053-help-for-one-command-plan.md`. Single pass:
plain checkboxes, no Mx.

- [x] Task 0 — side-quest: the README rewrite (`6cf5414`, carried by this
      branch) keeps the text the nine doc-sync tests pin; `go test ./cmd/define/`
      green, plan guards included.
- [x] Task 1 — every `commands` row carries `args` and `usage` (each text beside
      its parser, limits from the parsers' constants), and `/help <command>`
      explains one (`findCommand`, `unknownCommand`, `commandUsage`); bare `/help`
      says so; `/help`'s summary and both command-list spans updated.
- [x] Task 2 — `--help` and `-h` answered in `dispatchCommand` for every command,
      one-shot included.
- [x] Task 3 — the README and atlas quote the generated `command-usage` span;
      `pronCommandHelp`, its span and its test absorbed and removed, and the
      retirement recorded.
- [x] Task 4 — mutation-verify each guard; full suite, vet, gofmt; run the binary
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

Plan-quality round 1 blocked on two Important findings, and one of them is
about the plan file itself: with it in `workshop/plans/`, two repo guards also
fail. TestPlanTablesNameEntitiesThatExist read the `command` row's backticked
new fields as symbols claimed to exist, and `pronCommandHelp` marked deleted
while still declared; TestPlanCitesTestsThatExist found five unwritten tests
cited in backticks. So Task 0's exit criterion is the package green INCLUDING
the plan guards, eleven failures in all, not nine. The other finding:
`usageText` is already taken by the `--help` capture helper in
`deckasker_test.go`, so the renderer is `commandUsage`; every other new name was
checked against the package, tests included, and is free.

Implementation, Tasks 0–4 (commits `cae3eb2`, `d6db417`, `3f3bdae`, `c21195a`,
`d49af6a`):

- Task 0: all eight anchors matched once, and the package went green with the
  plan guards included. A correction to the count above: "nine doc-sync tests"
  is nine failures across eight tests, because
  `TestREADMEQuotesThePromptsTheLoopActuallyPrints` reports two.
- Tasks 1–3 each went red before the code and green after.
- One full run after Task 3's edits failed on a test I did not capture (the tail
  showed only a board footer), and two re-runs of the identical tree passed.
  Recorded as a flake, not diagnosed.
- Task 4 found a real miss: `TestARemovedDeclarationIsSweptOrRetired` failed the
  whole-repo run because the plan still named the test Task 3 removed. My
  post-commit check had been filtered to `TestPlan*`, which skipped it. Fixed in
  `d49af6a` (mention swept, removal mapped in `retiredSymbolNames`); the whole
  suite passed after that commit.
- Mutation check in a throwaway worktree at `c21195a`. Every mutation was
  caught, and the unmutated control was green:

  | mutation | went red |
  |---|---|
  | `runHelp` ignores its argument | TestHelpExplainsOneCommand |
  | no `--help` routing | TestDashHelpPrintsTheUsageForEveryCommand, 14 of 14 pairs |
  | one row's usage blanked | TestEveryRegisteredCommandIsRunnable |
  | `/help` words an unknown name itself | TestHelpForAnUnknownNameSaysWhatDispatchSays |
  | one usage word drifted in the atlas, then the README | TestDocsQuoteTheCommandUsage, each page |
  | the bare-help line dropped | TestBareHelpSaysHowToExplainOne |
  | `synopsis` pads a command with no args | TestCommandUsageWraps |

- gofmt clean; `go vet ./...` clean; `go test ./...` green (`cmd/define` after
  `d49af6a`; the other packages are unchanged and green).
- The built binary, in an empty directory: `define /help history`,
  `/history --help`, `/history -h`, `/help /history` and `/help HISTORY` each
  print the synopsis and the usage (exit 0). `/help histry` says "did you mean
  /history?" and `/help nosuch` lists the commands (exit 2). `/help a b` says it
  takes one command (exit 2). Bare `/help` lists the commands and the new
  `/help <command>` line. The TUI smoke test is the operator's, before merge.
- The estimate-quality judge (INFO) expects 0.49h to run 0.1–0.2h low, because
  the mutation pass had no line of its own.

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

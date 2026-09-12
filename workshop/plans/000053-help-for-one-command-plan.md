# /help <command> Implementation Plan (#53)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `/help history` and `/history --help` say how to use `/history`, and the same holds for every command, from one text per command that the docs quote.

**Architecture:** Each `commands` row gains two fields, `args` (the synopsis after the name) and `usage` (a sentence or two). `/help <name>`, `/<name> --help` and `/<name> -h` print them through one pure renderer. `dispatchCommand` answers `--help` once, for every command. The README and atlas quote a `command-usage` span generated from the registry and pinned by a doc-sync test; it replaces the atlas's hand-written argument paragraph and absorbs `pronCommandHelp`.

**Tech Stack:** Go; `cmd/define` package tests (`go test ./cmd/define/`).

**Branch.** `sdlc change-code` branches in place from `define-readme-rewrite`, so this branch carries the operator's README rewrite (`6cf5414`). That rewrite fails nine doc-sync tests. #53's README half needs the rewrite, and the rewrite cannot land red, so Task 0 makes it green and one PR lands both.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `command` | `cmd/define/command.go` | modified |
| `command.synopsis` | `cmd/define/command.go` | new |
| `commandUsage` | `cmd/define/command.go` | new |
| `findCommand` | `cmd/define/command.go` | new |
| `asksForUsage` | `cmd/define/command.go` | new |
| `helpUsage` | `cmd/define/command.go` | new |
| `historyUsage` | `cmd/define/history_cmd.go` | new |
| `statsUsage` | `cmd/define/stats.go` | new |
| `playUsage` | `cmd/define/play_cmd.go` | new |
| `soundUsage` | `cmd/define/sound_cmd.go` | new |
| `langUsage` | `cmd/define/lang_cmd.go` | new |
| `pronUsage` | `cmd/define/pron_cmd.go` | new |
| `pronCommandHelp` | `cmd/define/pron_cmd.go` | deleted |

- **`command`** gains `args` and `usage`: the one statement of how a command is used, 1:1 with a registry row.
  - **DRY rationale:** today the argument forms are written by hand in the atlas (`atlas/define.md` 1126–1131) and in `pronCommandHelp`; afterwards both derive from the row.
  - **Future extensions:** argument completion for `/help <name>` would read the same rows.
- **The usage texts** live beside the parser that implements them (`historyUsage` next to `parseHistoryArgs`, and so on), because the two change together. Numbers come from the parsers' own constants (`defaultHistoryDays`, `maxHistoryDays`, `maxSoundTimes`) through `fmt.Sprintf`, so the text cannot drift from the limit.
- **`pronUsage` absorbs `pronCommandHelp`.** Task 3 deletes `pronCommandHelp` and, in the same commit, adds its `deleted` row to this table (see "Plan guards at every commit").
- **`command.synopsis`** — `/name`, plus a space and `args` only when `args` is non-empty. The one place the terminal and the docs both build the synopsis (ARCH-DRY).
- **`commandUsage(c command, width int) string`** — the synopsis line, then the usage wrapped to width by `wrapText` (`render.go:633`) under the same two-space indent the command list uses, each line ending in a newline. Pure; unit-tested without IO.
- **`findCommand(name string, cmds []command) (command, bool)`** — the `EqualFold` lookup, moved out of `dispatchCommand` so `/help history` resolves a name exactly as dispatch does: `/help HISTORY` works because `/HISTORY` does.
- **`asksForUsage(args []string) bool`** — true for exactly one argument that is `--help` or `-h`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `runHelp` | `cmd/define/command.go` | modified | writes to commandCtx.stdout and stderr |
| `dispatchCommand` | `cmd/define/command.go` | modified | routes a parsed command line |
| `unknownCommand` | `cmd/define/command.go` | new | writes the unknown-command message |

- **`runHelp`** — no argument: the list, plus one new line saying `/help <command>` exists. One argument: strip a leading `/` (an empty name falls back to the list), `findCommand`, then print `commandUsage` or call `unknownCommand`. Two or more: a usage error, exit 2, `define: /help takes one command, not "a b"` (the shape `/sound` and `/lang` use).
- **`dispatchCommand`** — `findCommand`; not found → `unknownCommand`; `asksForUsage(args)` → print `commandUsage` and return 0 without running the command; otherwise run it.
- **`unknownCommand(stderr io.Writer, name string, cmds []command) int`** — today's two messages, moved verbatim out of `dispatchCommand`, so `/help histry` and `/histry` print the same bytes.

Nothing here leaves the process (ARCH-MOCK: N/A).

### Architecture notes

- **ARCH-PURPOSE, the consumer sweep.** Every place a command's argument rule is stated, and what it becomes:

  | consumer | today | after |
  |---|---|---|
  | `/help <name>` | ignores the name | prints `commandUsage` |
  | `/<name> --help`, `-h` | parsed as an argument (`/history`: "not a number of days") | prints `commandUsage` |
  | atlas command section, 1126–1131 | hand-written forms for `/history`, `/sound`, `/lang` | quotes the generated `command-usage` span |
  | atlas 1133–1137 | `pron-command-help` span from `pronCommandHelp` | removed; `/pron`'s row is in the span |
  | `cmd/define/README.md`, `/` section | "To check additional help for commands, type `/help [command]`." | the same sentence, then the span |
  | root `README.md:59` | "`/help` lists the rest" | unchanged, still true |
  | parse errors (`/sound takes one count, not …`) | error text | unchanged: errors, not the rule |

  Argument completion for `/help <name>` is out of scope. The purpose is a command that explains itself; completion is a separable extension, as the issue's Spec already says.
- **ARCH-DRY** — one wording for an unknown name (`unknownCommand`), one resolver (`findCommand`), one synopsis builder, one usage per row.
- **ARCH-PURE** — the renderer and the resolver are pure; `runHelp` and `dispatchCommand` only choose and write.
- **ARCH-CONSTRAINTS** — interaction path: a command submitted at the prompt. The work is formatting a handful of strings, O(number of commands), no IO beyond the writer. Nothing to budget beyond "instant".
- **ARCH-SECURE** — the only input is what the user typed after `/help`, echoed in errors the way dispatch already echoes an unknown name. No persisted input, no credentials.
- **ARCH-ORDER** — holds no state between events, because `/help` is a function of the registry, its arguments and the width.
- **A contract change, stated.** Answering `--help` and `-h` in dispatch means no command can ever take either as data. None does today (`/history` wants a number, `/sound` a count, `/lang` and `/pron` a two-letter tag), and the one-shot path reaches it too, because Go's flag parsing stops at the first non-flag argument (`/history`).

### Plan guards at every commit

This file is read by repo guards in `cmd/define/repo_guard_test.go`, and each task below ends with `go test ./cmd/define/` passing, so the file has to hold at every commit, not only at the end:

- `TestPlanTablesNameEntitiesThatExist` — the `modified` rows (`command`, `runHelp`, `dispatchCommand`) exist from the start; the `new` rows are promises while any box is unticked.
- `TestPlanTableStatusMatchesTheChangeWindow` — a `modified` row must be touched once its file is. Task 1 is the first commit to touch `command.go`, and it touches all three `modified` symbols, which is why the registry change and `/help <command>` are one task. Task 0 touches no file a row names.
- `TestPlanNamedTestsExist`, `TestPlanCitesTestsThatExist` — a test is written in backticks here only if it exists. The tests this plan adds are named plainly until they exist, and the one Task 3 deletes is not named at all.
- `TestNoArtifactNamesARetiredSymbol` — Task 3 adds `pronCommandHelp` → `pronUsage` to `retiredSymbolNames`; code, atlas and README are swept in the same commit, and this plan stays exempt through its `deleted` row.
- `TestPlanStatusNormalisesToTheVocabulary` — statuses are `new`, `modified` or `deleted`.
- `TestARemovedDeclarationIsSweptOrRetired` — a page that names a declaration the window removed fails unless `retiredSymbolNames` maps it. Task 3 removes a test as well as `pronCommandHelp`, so both are mapped, and no current page names either (found at Task 4: the post-commit check had been filtered to `TestPlan*`, which this guard is not).

---

## Task 0: side-quest — the README rewrite keeps the text the doc-sync tests pin

The rewrite (`6cf5414`) dropped text nine tests require. Put each piece back into the rewrite's structure and leave the rest of its prose alone. The source is main's copy: `git show main:cmd/define/README.md`.

**Files:** Modify `cmd/define/README.md`.

- [x] **Step 1: Confirm the failures.**
  Run: `go test ./cmd/define/ -run 'TestEverySurfaceDescribingCaptureMentionsTheQuestion|TestTheREADMEQuotesTheRealPrompt|TestEverySurfaceNamesEveryCuratedLanguage|TestREADMEQuotesThePromptsTheLoopActuallyPrints|TestDocsQuoteThePronHelp|TestDocsQuoteTheLocaleHelp|TestREADMEKeyTableNamesEveryLiveKey|TestREADMEAnchorsResolve'`
  Expected: FAIL with the nine messages recorded in the issue Log.
- [x] **Step 2: Restore each piece.** Every insertion is anchored on text that occurs exactly once, and the script asserts the count before writing.

  | piece | where in the rewrite | fixes |
  |---|---|---|
  | `### The directory is the deck, so it asks first` (main 51–80: the `Create one here? [y/N]` prompt, `--here`) | after `### Install` | `-here`; the real prompt; the rewrite's own link to that heading |
  | cloze graded prompt (main 283–286) | before `### Multiple choice, once your deck can supply distractors` | graded prompt for `*play.Cloze` |
  | `any key = next word, d = remove from deck, Ctrl-C to stop`, introduced as "Once you have answered, the prompt changes:" (main's lead-in says "After an `n`", which names the retired form 2.1, so it is not copied) | after the multiple-choice example | graded prompt for `*play.Choice` |
  | `### Keys in a review` holding the `review-keys` span (main 408–421) | before `### On a Schedule` | review-keys span |
  | `curated-languages` span (main 574–582) with a one-line lead-in | end of `### Languages` | curated languages |
  | `-pron` and `-locale` spans (main 611 and 627) as two list items | after the `### From the command line` example block | pron-help, locale-help |
  | the two "see `--llm-check`" mentions become links to `#checking-the-llm-model-connection` | learner-model and practice-material sections | three in-page links |
- [x] **Step 3: Run the package.** `go test ./cmd/define/` → PASS, the plan guards included (eleven failures before: the nine above and the two plan-guard failures round 1 found).
- [x] **Step 4: Commit.** `#53: side-quest: README: the rewrite keeps the text the doc-sync tests pin`

## Task 1: every command carries its usage, and `/help <command>` explains one

**Files:** Modify `cmd/define/command.go`, `history_cmd.go`, `stats.go`, `play_cmd.go`, `sound_cmd.go`, `lang_cmd.go`, `pron_cmd.go`, plus the `command-list` spans in `cmd/define/README.md` and `atlas/define.md`. Test `cmd/define/command_test.go`.

- [x] **Step 1: Write the failing tests.** Extend `TestEveryRegisteredCommandIsRunnable`:
  ```go
  if c.usage == "" {
  	t.Errorf("command /%s has no usage; /help %s would print only its name", c.name, c.name)
  }
  ```
  and add:
  ```go
  func TestHelpExplainsOneCommand(t *testing.T) {
  	hist, _ := findCommand("history", commands)
  	for _, arg := range []string{"history", "/history", "HISTORY"} {
  		var out, errb bytes.Buffer
  		code := runHelp(commandCtx{cmds: commands, stdout: &out, stderr: &errb}, []string{arg})
  		if code != 0 || out.String() != commandUsage(hist, 0) {
  			t.Errorf("/help %s: exit %d, out %q", arg, code, out.String())
  		}
  		if !strings.Contains(out.String(), "/history [N | --days N | --days=N]") {
  			t.Errorf("/help %s does not show the synopsis: %q", arg, out.String())
  		}
  	}
  	// The disagreeing case: a different name must print a different usage,
  	// so a runHelp that ignored its argument cannot pass.
  	var out bytes.Buffer
  	runHelp(commandCtx{cmds: commands, stdout: &out, stderr: io.Discard}, []string{"sound"})
  	if strings.Contains(out.String(), "/history") || !strings.Contains(out.String(), "/sound [N]") {
  		t.Errorf("/help sound printed %q", out.String())
  	}
  }

  func TestHelpForAnUnknownNameSaysWhatDispatchSays(t *testing.T) {
  	for _, name := range []string{"histry", "qqqqqq"} {
  		var viaHelp, viaDispatch bytes.Buffer
  		c1 := runHelp(commandCtx{cmds: commands, stdout: io.Discard, stderr: &viaHelp}, []string{name})
  		c2 := dispatchCommand(parseREPLLine("/"+name, false), commands, commandCtx{stdout: io.Discard, stderr: &viaDispatch})
  		if c1 != 2 || c2 != 2 || viaHelp.Len() == 0 || viaHelp.String() != viaDispatch.String() {
  			t.Errorf("%s: /help said %q (exit %d), dispatch said %q (exit %d)", name, viaHelp.String(), c1, viaDispatch.String(), c2)
  		}
  	}
  }

  func TestHelpTakesOneCommand(t *testing.T) {
  	var errb bytes.Buffer
  	if code := runHelp(commandCtx{cmds: commands, stdout: io.Discard, stderr: &errb}, []string{"history", "sound"}); code != 2 {
  		t.Errorf("exit = %d, want 2", code)
  	}
  	if !strings.Contains(errb.String(), "takes one command") {
  		t.Errorf("stderr = %q", errb.String())
  	}
  }

  // The renderer over every row and three widths. /stats and /play have no
  // args, so a synopsis with a trailing space would diverge silently from the
  // doc span, which sets it in backticks.
  func TestCommandUsageWraps(t *testing.T) {
  	for _, c := range commands {
  		for _, width := range []int{0, 20, 80} {
  			lines := strings.Split(strings.TrimSuffix(commandUsage(c, width), "\n"), "\n")
  			if lines[0] != "  "+c.synopsis() || strings.HasSuffix(c.synopsis(), " ") {
  				t.Errorf("/%s at %d: synopsis line %q", c.name, width, lines[0])
  			}
  			body := lines[1:]
  			if width == 0 && len(body) != 1 {
  				t.Errorf("/%s at width 0 wrapped into %d lines", c.name, len(body))
  			}
  			var words []string
  			for _, l := range body {
  				if width > 0 && visibleCells(l) > width && len(strings.Fields(l)) > 1 {
  					t.Errorf("/%s at %d: %q is %d cells wide", c.name, width, l, visibleCells(l))
  				}
  				words = append(words, strings.Fields(l)...)
  			}
  			if strings.Join(words, " ") != strings.Join(strings.Fields(c.usage), " ") {
  				t.Errorf("/%s at %d: wrapping changed the words", c.name, width)
  			}
  		}
  	}
  }

  func TestBareHelpSaysHowToExplainOne(t *testing.T) {
  	var out bytes.Buffer
  	runHelp(commandCtx{cmds: commands, stdout: &out, stderr: io.Discard}, nil)
  	if !strings.Contains(out.String(), "/help <command>") {
  		t.Errorf("bare /help never mentions /help <command>:\n%s", out.String())
  	}
  }
  ```
- [x] **Step 2: Run them.** `go test ./cmd/define/ -run 'TestEveryRegisteredCommandIsRunnable|TestHelp|TestBareHelp|TestCommandUsage'` → FAIL (no field `usage`; `findCommand` and `commandUsage` undefined).
- [x] **Step 3: Implement.** Add `args` and `usage` to `command`, and `command.synopsis`. Fill the rows:
  ```go
  {name: "help", summary: "list the commands, or explain one", args: "[command]", usage: helpUsage, run: runHelp},
  {name: "history", summary: "words looked up recently", args: "[N | --days N | --days=N]", usage: historyUsage, run: runHistory},
  {name: "stats", summary: "deck, streak and accuracy figures", usage: statsUsage, run: runStatsCommand},
  {name: "play", summary: "review the words due today", usage: playUsage, run: runPlayCommand},
  {name: "sound", summary: "how many times to play a pronunciation", args: "[N]", usage: soundUsage, run: runSound},
  {name: "lang", summary: "the language this deck is in", args: "[language]", usage: langUsage, run: runLang},
  {name: "pron", summary: "replay this word in its source language, once", args: "[language]", usage: pronUsage, run: runPron},
  ```
  The texts are plain, with no markdown, because they print on a terminal:
  - `helpUsage` = "With nothing, list the commands. With a command's name, say how to use it, which --help after any command also does."
  - `historyUsage` = `fmt.Sprintf("The words looked up in the last N days. With nothing, the last %d; N is at most %d.", defaultHistoryDays, maxHistoryDays)`
  - `statsUsage` = "The deck, streak and accuracy figures for this directory. Takes no arguments."
  - `playUsage` = "Review the words due today; Ctrl-C stops and keeps every answer. Takes no arguments."
  - `soundUsage` = `fmt.Sprintf("With nothing, how many times each pronunciation plays. With N, play it N times for the rest of this session; 0 turns playback off, and %d is the most.", maxSoundTimes)`
  - `langUsage` = "With nothing, the language in effect and the dictionary answering it. With a two-letter tag like es, switch to that language: saved when this directory is a deck, for this session otherwise."
  - `pronUsage` = "Replay this word once in another language. With nothing, it reads the source language off the entry's ORIGIN and says which it chose. It declines when ORIGIN names only historical stages (Old French, Latin) or cognates (\"related to Dutch …\"), because neither is a language anyone speaks the word in today."

  Then `findCommand`, `unknownCommand` (moved out of `dispatchCommand`, which now calls it and `findCommand`), `commandUsage`, and `runHelp` as described under Integration points. Bare `/help` gains `  /help <command>, or --help after one, says how to use it` between the list and the hatches.

  `/help`'s summary becomes "list the commands, or explain one". It keeps "list the commands" as its prefix on purpose: eight assertions Contains-check that string (commandloop_test.go lines 77, 92, 181, 193, 202, 310; pty_conformance_test.go lines 288, 309), and all of them, the two absence checks included, stay green. The `command-list` span in `cmd/define/README.md` and `atlas/define.md` is updated to match (`TestDocsQuoteTheCommandList` enforces it).

  `pronCommandHelp` stays until Task 3, which moves every doc to the new span in one step.
- [x] **Step 4: Run.** The tests from Step 1 PASS; `go test ./cmd/define/` → PASS.
- [x] **Step 5: Commit.** `#53: every command carries its usage, and /help <command> explains one`

## Task 2: `--help` and `-h` after any command

**Files:** Modify `cmd/define/command.go`. Test `cmd/define/commandloop_test.go`.

- [x] **Step 1: Write the failing tests.**
  ```go
  // Over the WHOLE registry: without the routing, every one of these fails in its
  // own parser (/history reads --help as a number of days, /play and /stats
  // refuse arguments, /lang and /pron reject it as a language, /help as a name).
  func TestDashHelpPrintsTheUsageForEveryCommand(t *testing.T) {
  	for _, c := range commands {
  		for _, flag := range []string{"--help", "-h"} {
  			var out, errb bytes.Buffer
  			code := dispatchCommand(parseREPLLine("/"+c.name+" "+flag, false), commands, commandCtx{stdout: &out, stderr: &errb})
  			if code != 0 || out.String() != commandUsage(c, 0) || errb.Len() != 0 {
  				t.Errorf("/%s %s: exit %d, out %q, err %q", c.name, flag, code, out.String(), errb.String())
  			}
  		}
  	}
  }

  func TestDashHelpDoesNotRunTheCommand(t *testing.T) {
  	var out bytes.Buffer
  	dispatchCommand(parseREPLLine("/history --help", false), dispatchCmds, commandCtx{stdout: &out, stderr: io.Discard})
  	if strings.Contains(out.String(), "HISTORY RAN") {
  		t.Errorf("--help ran the command: %q", out.String())
  	}
  }

  func TestOneShotHelpExplainsOneCommand(t *testing.T) {
  	rig := newAudioRig(t, "sycophantic", true)
  	rig.deps.dict = refusingDict{t}
  	rig.deps.stdinIsTerminal = func() bool { return false }
  	rig.deps.history, rig.deps.capture, rig.deps.deck = nil, nil, nil
  	rig.deps.newStore = openStore
  	t.Chdir(t.TempDir())
  	for _, args := range [][]string{{"-no-audio", "/help", "history"}, {"-no-audio", "/history", "--help"}} {
  		var out, errb bytes.Buffer
  		code := run(t.Context(), args, rig.deps, strings.NewReader(""), &out, &errb)
  		if code != 0 || !strings.Contains(out.String(), "/history [N | --days N | --days=N]") {
  			t.Errorf("define %v: exit %d, out %q, err %q", args, code, out.String(), errb.String())
  		}
  	}
  }
  ```
  The fixture rows in `dispatchCmds` gain a `usage`, so the fixture test has something to print.
- [x] **Step 2: Run them.** → FAIL.
- [x] **Step 3: Implement** `asksForUsage` and its branch in `dispatchCommand`.
- [x] **Step 4: Run.** → PASS; `go test ./cmd/define/` → PASS.
- [x] **Step 5: Commit.** `#53: --help after any command prints its usage`

## Task 3: the docs quote the usage from the registry

**Files:** Modify `cmd/define/README.md`, `atlas/define.md`, `cmd/define/pron_cmd.go`, `cmd/define/doc_sync_test.go`, `cmd/define/repo_guard_test.go`, and this plan's Pure entities table.

- [x] **Step 1: Write the failing test.**
  ```go
  func commandUsageSpan() string {
  	var b strings.Builder
  	b.WriteString("<!-- command-usage -->\n")
  	for _, c := range commands {
  		fmt.Fprintf(&b, "- `%s` — %s\n", c.synopsis(), c.usage)
  	}
  	b.WriteString("<!-- /command-usage -->")
  	return b.String()
  }

  func TestDocsQuoteTheCommandUsage(t *testing.T) {
  	want := commandUsageSpan()
  	for _, doc := range derivedDocs {
  		raw, err := os.ReadFile(doc)
  		if err != nil {
  			t.Fatalf("%s unreadable: %v", doc, err)
  		}
  		if !strings.Contains(string(raw), want) {
  			t.Errorf("%s does not quote the command usage the registry produces.\nwant the marked span to read:\n%s\n"+
  				"`commands` owns each usage; the page consumes it.", doc, want)
  		}
  	}
  }
  ```
- [x] **Step 2: Run it.** → FAIL for both docs.
- [x] **Step 3: Implement.**
  - `cmd/define/README.md`: the span goes right after "To check additional help for commands, type `/help [command]`."
  - `atlas/define.md`: the paragraph at 1126–1131 becomes the rationale (the summary stays short because it is what a bare `/help` prints; each row's `args` and `usage` are what `/help <command>` and `/<command> --help` print; `--help` is answered once in `dispatchCommand` because `/history` used to read it as a number of days), followed by the span. The `pron-command-help` paragraph and span (1133–1137) are deleted.
  - Delete `pronCommandHelp` and the doc-sync test that pinned its span. Rewrite the comment at `pron_cmd.go:85–97` to say `/pron`'s rule is its row's usage. Update `TestDocsQuoteTheCommandList`'s comment: argument syntax lives in `usage`, quoted by `command-usage`.
  - Add `"pronCommandHelp": "pronUsage"` to `retiredSymbolNames` in `repo_guard_test.go`.
  - Add a row for `pronCommandHelp` at `cmd/define/pron_cmd.go`, status deleted, to this plan's Pure entities table, in this same commit.
- [x] **Step 4: Sweep the deletion.** `git grep -n 'pronCommandHelp\|pron-command-help' -- ':!workshop/history'` → only the #53 issue, this plan and the `retiredSymbolNames` entry.
- [x] **Step 5: Run.** `go test ./cmd/define/` → PASS.
- [x] **Step 6: Commit.** `#53: the docs quote each command's usage from the registry`

## Task 4: verify

- [ ] **Step 1: Mutation-verify each guard** in a throwaway worktree, each mutation applied by a script that asserts its match count (lessons: "Mutation testing has to be done, and the tooling for it has to be safe"; "An unasserted string replacement … is a silent no-op"):

  | mutation | guard that must go red |
  |---|---|
  | `runHelp` ignores its argument | TestHelpExplainsOneCommand |
  | remove the `asksForUsage` branch | TestDashHelpPrintsTheUsageForEveryCommand, for all seven |
  | blank one row's `usage` | `TestEveryRegisteredCommandIsRunnable` |
  | `runHelp` prints its own unknown-name wording | TestHelpForAnUnknownNameSaysWhatDispatchSays |
  | change one word of one usage in the atlas span, then in the README span | TestDocsQuoteTheCommandUsage |
  | drop the bare-help line | TestBareHelpSaysHowToExplainOne |
  | `synopsis` appends a space when `args` is empty | TestCommandUsageWraps |

  Plus a control run of the unmutated tree, all green.
- [ ] **Step 2:** `gofmt -l cmd/define` (empty), `go vet ./...`, `go test ./...` → PASS.
- [ ] **Step 3: Run the program.** Build, then in an empty directory: `define /help history`, `define /history --help`, `define /help nosuch`, `define /help a b`, `define /help`. Record the output in the Log. The operator smoke-tests the TUI.

## Operating envelope

N/A beyond "instant": a command at the prompt, a handful of string operations, no IO but the writer.

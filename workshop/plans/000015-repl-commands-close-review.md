# Boundary Review — tools#15 (whole-issue close)

| field | value |
|-------|-------|
| issue | 15 — REPL command mode: /-prefixed commands with type-ahead, starting with /history |
| repo | tools |
| issue file | workshop/issues/000015-repl-commands.md |
| boundary | whole-issue close |
| milestone | — |
| window | c7d07575cb125fcb438bcb3fe45ff3763282690b..1ff0d5e8aa930f4f7ffb4b0a2473de6239f495d4 |
| command | sdlc close --issue 15 |
| reviewer | claude |
| timestamp | 2026-08-21T17:17:54-07:00 |
| verdict | REWORK |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: REWORK
confidence: high
```

The `/` namespace is genuinely well-built: `parseREPLLine` is the single classifier both loops route through, the M1b menu is pure and its column arithmetic is mutation-pinned, and `1ff0d5e`'s removal of `draw`'s parameter is the right shape of fix — it deletes the class rather than the instance, and `workshop/lessons.md` records the generalisation correctly. `go vet`, `gofmt -l`, `go test -count=1`, and `go test -tags conformance -run PTY` (on a real pty, outside the sandbox) are all clean. Two things block the boundary, both measured rather than inferred. First, `relativeDay` (`history_cmd.go:191`) converts a `time.Duration` to days by dividing by 24 — the exact elapsed-hours mistake `historyWindow` was written to avoid, and which its own doc comment disclaims — so for the week after a spring-forward transition every date in `/history` is off by one and yesterday's lookup reads "today". Second, this window commits the developer's own deck (`cmd/define/events/2026-08-{21,22}.yaml`, `cmd/define/words/sycophantic.yaml`) into the source tree, and I reproduced the consequence: running the conformance suite rewrote both tracked files, because the pty harness launches the real binary with cwd set to `cmd/define/`. Beneath those, four separate loop-shell wirings shipped this round — `/sound` in the raw editor, the one-shot dispatch, the piped exit code, `hist.Add` of a submitted command — are pinned by nothing; I removed each one and the suite stayed green. That is the fourth round of `unpinned-production-wiring` on this issue, so it needs a rule, not a fifth patch.

## 1. Strengths

- **`draw()` taking no parameter (`replraw.go:105-118`) is the correct fix for the reported bug**, not the local one. Removing the argument removes the only place a stale list could enter, and `TestSuggestionMatchesWhatTabAccepts` asserts both halves (grey tail is `/help`, *and* Tab commits it) rather than just the symptom. The `lessons.md` entry generalises it honestly, including the corollary that "it was already like that and nothing broke" is not evidence.
- **`historyWindow` is right and its tests prove it.** I re-derived the arithmetic independently: `AddDate` from local midnight lands on `2026-03-08T00:00-08:00` and `2026-11-01T00:00-07:00`, spanning 23 and 25 local hours as intended. Embedding `_ "time/tzdata"` (`history_cmd_test.go:14`) so the DST rows cannot silently skip applies the repo's own "a guard that reports nothing certifies nothing" lesson to a test's *inputs*.
- **`TestWindowIncludesThisEveningDespiteTomorrowsFilename` asserts its own premise** (`t.Fatalf("premise wrong: the lookup is stored in %s")`) before asserting the claim. A test that verifies the setup still produces the condition it was written for is worth copying.
- **BR-9 and BR-8 are pinned, verified by reverting them.** Restoring `dispatchCommand`'s `len(near) != len(cmds)` inference reddens `TestNearMissWithOneRegisteredCommand`; restoring the `strings.ToLower(prefix)` in `commandCompletions` reddens `TestCommandCompletions`. Both fixes hold under mutation.
- **`menuNameWidth` computed over all commands is pinned** — swapping it for `len(c.name)+2` reddens `TestMenuLines` (verified) — and the pty test now derives its cursor-up count from the frame (`rows := strings.Count(frame, "\r\n")`) instead of hardcoding `\x1b[1A`, so registering a fourth command will not break it.

## 2. Critical findings

**C-1 — `relativeDay` answers a calendar question with elapsed hours, and is off by one for a week after every spring-forward.** `cmd/define/history_cmd.go:191`:

```go
switch days := int(dayOf(now).Sub(dayOf(at)).Hours() / 24); {
```

Two local midnights one calendar day apart are 23 h apart across a spring-forward and 25 h across a fall-back. `int(23.0/24)` is `0`. Measured against the shipped function body, `America/Los_Angeles`:

| `at` | `now` | prints | should print |
|---|---|---|---|
| 2026-03-08 | 2026-03-09 | `today` | `yesterday` |
| 2026-03-07 | 2026-03-09 | `yesterday` | `Saturday` |
| 2026-03-02 | 2026-03-09 | `Monday` | `Mar 2` |
| 2026-11-01 | 2026-11-02 | `yesterday` | `yesterday` ✓ |

Every relative date shifts by one for the ~7 days following the transition, including the `days < 7` weekday cutoff. The function's own comment three lines up says it is "computed on LOCAL CALENDAR DAYS, not elapsed hours" — this is behaviour drift from a contract the code states about itself, in the one computation the issue centres on (ARCH-DRY: `historyWindow` already owns "local calendar day" and gets it right via `AddDate`; `relativeDay` is a second, wrong implementation of the same concept). Fix sketch: round to the nearest day rather than truncating — `int(dayOf(now).Sub(dayOf(at)).Round(24*time.Hour) / (24 * time.Hour))` — or count with `AddDate`. Pin it with two rows in `TestRenderHistoryRelativeDatesAreCalendarDays` using the DST dates the sibling test already uses; the `la(t)` helper and embedded tzdata are already in the file.

**C-2 — the developer's deck is committed, and the conformance suite rewrites it.** `cmd/define/events/2026-08-21.yaml`, `cmd/define/events/2026-08-22.yaml`, `cmd/define/words/sycophantic.yaml` are tracked (`git ls-files` confirms), added by `fc071af` and `1ff0d5e` — two separate commits in this window, so it is a recurring sweep, not one slip. No test or fixture references them (`grep` over `*.go` finds nothing). Reproduced: after `go test -tags conformance -run PTY ./cmd/define/`, `git status` reports both files **modified** — `events/2026-08-22.yaml` gained two records and `words/sycophantic.yaml` went from `lookups: 12` to `14`. The cause is `pty_conformance_test.go:47`, which launches `../../bin/define` inheriting the test's cwd (`cmd/define/`), and `define` writes its deck to the current directory. So the suite is not idempotent against the index, and a `git add -A` picks up deck churn as part of an unrelated commit — which is exactly how these landed. `.gitignore:16-19` already anticipates this class but anchors at the root (`/words/`, `/events/`), which does not cover `cmd/*/`. Fix sketch: `git rm --cached` the three files (rewrite the two commits if the private data matters — `repo_guard_test.go`'s own doctrine is that removing in a follow-up leaves the blob reachable), extend `.gitignore` to `words/` and `events/` unanchored, give `startDefine` a `cmd.Dir = t.TempDir()` so the live binary writes to a scratch deck, and add a `repo_guard_test.go` case for tracked deck artifacts alongside `TestNoCommittedBinaries` — the binaries guard exists precisely because this class recurs.

## 3. Important findings

**I-1 — four loop-shell wirings shipped this round with no test that fails without them.**

> **This is the 4th finding in family `unpinned-production-wiring`** (BR-2 was the first, and BR-16's fix below is another). Earlier rounds fixed instances. Do NOT fix these four instances one by one — state the rule and fix that.

Measured this round; each mutation applied, `BUILD_OK`, full suite `-count=1` **green**:

| site | mutation | result |
|---|---|---|
| `replraw.go:166` `cc.setTimes = …` | deleted | green |
| `main.go:292-294` one-shot `dispatchCommand` block | deleted | green |
| `repl.go:139-142` `cmdCode` return | deleted | green |
| `replraw.go:167` `hist.Add(submitted.String())` | deleted | green |

The `replraw.go:166` one is the sharpest: M3 exists because the operator asked for "`/sound 1` supported in TUI", and without that line `/sound 1` at the real prompt prints *"needs an interactive session"* — the exact opposite of the requirement — with the suite still green. `TestSoundChangesPlaybackForTheRestOfTheSession` drives `replLines`, the **piped** loop, not the TUI.

**The rule:** *a value or effect that only a loop shell supplies must be pinned by a test that drives that loop shell — `runEditor`, `replLines`, or `run` — never by a test that builds the callee's context by hand.* Every `commandCtx{…}` literal in a test (`sound_cmd_test.go:52,64,77,88`, `commandloop_test.go:131`, `history_cmd_test.go:271,354`) is a place this rule can be violated invisibly: the literal supplies the field the loop was supposed to, so the pure function is proven and the wiring is not. **The rule-level fix:** one table in `commandloop_test.go` that runs the same command scripts through *both* `runEditor` and `replLines` and asserts the same observable — a field wired in one loop and forgotten in the other then reddens by construction, which is the failure mode `newCommandCtx` (BR-6) made compile-time-visible for *construction* but not for *post-construction assignment*, which is what `setTimes` is.

**I-2 — `define /history 7` still fails with a usage dump; only zero-argument one-shot commands work.**

> **This is the 2nd finding in family `entry-mode-inconsistency`** (BR-13 was the first). Do NOT patch this instance — state the rule and fix that.

Probed against the built binary:

```
$ define /history 7        →  usage: define [flags] [word] … (exit 2)
$ echo '/history 7' | define →  "  nothing looked up in the last 7 days" (exit 0)
```

`main.go:266` rejects `fs.NArg() > 1` before `main.go:292` ever tests for a command, and `main.go:292` classifies `fs.Arg(0)` alone rather than a line. README:96 claims "`define /help`, `echo /help | define`, and `/help` typed at the prompt are one thing" and `atlas/define.md` says "**Every entry mode reaches it.**" — both true only for the arity-1 subset.

**The rule:** *every entry mode must hand `parseREPLLine` the same input — a whole line.* The two loops do; the argv path hands it one token. Fix at that level: when `fs.Arg(0)` opens command mode, classify `strings.Join(fs.Args(), " ")` and dispatch it *before* the arity guard — rather than extending the arity guard's special cases each time a command grows an argument. BR-13's fix stopped at the zero-argument case, which is why the same family is back.

**I-3 — `plan-artifact-lags-code`, 2nd instance: the Core-concepts table does not name the entities two scope events added, and six step boxes are unticked.**

> **This is the 2nd finding in family `plan-artifact-lags-code`** (BR-5 was the first). Do NOT tick the boxes and move on — state the rule.

`workshop/plans/000015-repl-commands-plan.md` lists eleven PURE rows and three INTEGRATION rows; `menuLines`, `menuNameWidth` (M1b) and `parseSoundArgs`, `runSound`, `soundTimes`, the whole `sound_cmd.go` file (M3) appear in no row, though `menuLines`/`parseSoundArgs`/`runSound` *do* appear in the test-strategy tables. Six `- [ ]` boxes remain at lines 277, 300, 311, 312, 326 — Task 6 Step 3 ("Implement", which plainly happened) and the Step-5 "Commit" boxes of Tasks 4-7 — which `sdlc close`'s plan-unchecked guard reads. **The rule:** *the Core-concepts table is the contract this review cross-checks against the filesystem, so a scope event that adds an entity adds its row in the same commit that adds the code, and a task's boxes are ticked by the commit that lands it.* The `## Revisions` entries for M1b and M3 describe the work in prose but never amend the table, which is what made both gaps possible.

## 4. Minor findings

- `--sound 1000` is accepted while `/sound 1000` is refused at `maxSoundTimes = 20` (`sound_cmd.go:12`) — probed both. `atlas/define.md` says "`--sound` is the same setting for one run"; one setting, two validation policies.
- `main.go:229` prints `define: -sound must not be negative` for `-times -1` — verified — naming a flag the user did not type. `main_test.go:253` asserts only the exit code, so it passes either way.
- Two implementations of "pad a name column, truncate to width": `menuLines` (`command.go:239`) slices **bytes** and could split a rune; `renderHistory` (`history_cmd.go:172`) slices **runes**. `renderHistory`'s `nameW`/`dateW` are measured with `len()` (bytes) while `%-*s` pads in runes, so a non-ASCII headword like `café` widens the whole table's gutter (alignment survives; the column is just wider than asked for). ARCH-DRY: one `truncateToWidth` helper.
- `renderHistory`'s truncation branch is never exercised — both call sites in `history_cmd_test.go` pass `width = 0`, and nothing asserts `runHistory` forwards `c.width`. The equivalent branch in `menuLines` *is* covered (`{"narrow terminals truncate", "/his", 14, …}`).
- `relativeDay` with a future `at` (clock skew, or a synced deck from a machine ahead) yields a negative `days`, which falls into `days < 7` and prints a weekday name.

## 5. Test coverage notes

- Verified green: `go vet ./cmd/...`, `go vet -tags conformance ./cmd/define/`, `gofmt -l cmd/` (empty), `go test ./cmd/define/... -count=1`, and `go test -tags conformance -run PTY` — all four pty tests including `TestPTYCommandMenuAppearsAndClears` **PASS** on a real pty (they `t.Skip` under this sandbox, so they need the unsandboxed run to mean anything).
- Verified pinned by reverting the fix: BR-9 (`dispatchCommand` re-deriving `close` from a count → `TestNearMissWithOneRegisteredCommand` reddens), BR-8 (`commandCompletions` lowercasing the prefix → `TestCommandCompletions` reddens), `menuNameWidth` over all commands (→ `TestMenuLines` reddens).
- Verified **unpinned**: the four sites in I-1, plus `clearMenu()` before submit (`replraw.go:150`) — deleting it leaves the suite green, and the pty test never submits a command, so nothing checks the menu is gone before a command's output scrolls over it.
- The gap that would have caught C-1 already exists in the file: `TestHistoryWindow` has four DST rows; `TestRenderHistoryRelativeDatesAreCalendarDays` has two rows, both in August. The plan's test-strategy line for `renderHistory` says its tests exist to catch "relative-date formatting regressions", and this is the one the domain guarantees.
- `TestEveryRegisteredCommandIsRunnable` (BR-10) is well-formed — it iterates the *live* registry, `t.Fatal`s if it is empty rather than passing vacuously, and also checks `summary != ""`.

## 6. Architectural notes

- **ARCH-DRY — flag.** `parseREPLLine` as one classifier and `completionsFor` as one namespace switch both hold. Flagged: `relativeDay` is a second implementation of the local-calendar-day concept `historyWindow` owns, and it is the wrong one (C-1); `menuLines` and `renderHistory` carry divergent truncate-and-pad logic (Minor).
- **ARCH-PURE — pass.** `parseCommandLine`, `commandCompletions`, `nearestCommands`, `editDistance`, `completionsFor`, `menuLines`, `menuNameWidth`, `historyWindow`, `parseHistoryArgs`, `summariseLookups`, `renderHistory`, `parseSoundArgs` are all pure and table-tested with no store, clock, terminal or network — I confirmed none of their tests needs a mock to run. `runHistory`/`runSound`/`dispatchCommand` take writers and are the thin shell. `commandCtx` narrower than `deps` keeps "a command cannot reach the dictionary or the player" structural. The one caution is that this purity is what makes I-1 easy to miss: proving the pure half is cheap, so the wiring goes unproven.
- **ARCH-PURPOSE — flag.** Shadow-sweep of "what does a line mean": `replLines` derives ✓, `runEditor` derives ✓, `menuLines`/`completionsFor` derive ✓, the argv path derives **partially** (I-2). Sweep of "what commands exist": `/help` lists `c.cmds` ✓; `atlas/define.md`'s three-row table and `main.go`'s usage prose are hand-maintained restatements, acceptable as documentation since the enforced consumer derives. The `/sound` seam does fulfil the operator's ask in design — it is the *test* that stops at the piped loop, not the code.
- **ARCH-MOCK — flag.** `store.Mem` + `store.FixedClock` are production code behind the same interface the YAML store implements, and `refusingDict` asserts a negative interaction, which is stronger than an output check. The flag is C-2's other face: the live conformance run and the fake do **not** share a storage boundary — the fake writes to memory, the live binary writes to the repo's own tracked deck. "A fake satisfies this only when production flow and test flow share the same boundary"; giving `startDefine` a `cmd.Dir` of a temp dir restores that.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000015-repl-commands-plan.md`:

- **Amend the Core-concepts table for both scope events.** Add PURE rows for `menuLines` and `menuNameWidth` (`cmd/define/command.go`, M1b) and for `parseSoundArgs` and `soundTimes` (`cmd/define/sound_cmd.go`, M3), plus an INTEGRATION row for `runSound`. The M1b and M3 revision prose describes the work but never amends the table it invalidated, which is what this review cross-checks (I-3).
- **Record that `relativeDay` is an entity the plan never named.** It is where `/history`'s output is actually decided and it carries C-1; the plan's `renderHistory` row silently covers it. Either name it as its own PURE row or state that it is internal to `renderHistory` and that `renderHistory`'s test corpus must therefore include the DST rows `historyWindow`'s already has.
- **Correct the Task 2 Step 3 signature.** The plan still specifies `completionsFor(line string, hist History) []string`; the shipped function is `completionsFor(base string, hist History, cmds []command) []string`. Carried unfixed from the M1 review's recommendation.
- **Tick or resolve the six remaining boxes** (lines 277, 300, 311, 312, 326) — Task 6 Step 3 and the Step-5 commit boxes of Tasks 4-7, all of which have shipped commits.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The plan now carries a per-function "what its test exists to catch" table (plan lines 154-165) plus one for M3.
  - id: BR-7
    disposition: addressed
    note: |
      width is read by renderHistory and menuLines, and newCommandCtx takes opt.width rather than re-deriving via terminalWidth.
  - id: BR-8
    disposition: addressed
    note: |
      Mutation-verified: restoring strings.ToLower(prefix) in commandCompletions reddens TestCommandCompletions.
  - id: BR-9
    disposition: addressed
    note: |
      Mutation-verified: restoring dispatchCommand's len(near) != len(cmds) inference reddens TestNearMissWithOneRegisteredCommand.
  - id: BR-10
    disposition: addressed
    note: |
      TestEveryRegisteredCommandIsRunnable iterates the live registry and Fatals on an empty one, so it cannot pass vacuously.
  - id: BR-11
    disposition: addressed
    note: |
      Both arg comparisons now use reflect.DeepEqual; sameSet survives only in TestCompletionsFor, where the data really is unordered.
  - id: BR-12
    disposition: addressed
    note: |
      The count is gone from command_test.go:130 — the comment no longer names a number that a fifth call site can falsify.
  - id: BR-13
    disposition: not-addressed
    note: |
      Behaviour verified correct for `define /help`, but no test pins it (deleting main.go:292-294 leaves the suite green) and `define /history 7` still prints a usage dump — see the entry-mode-inconsistency rule finding.
  - id: BR-14
    disposition: addressed
    note: |
      main.go:4-13 now has stdlib in one group and the store import in the second, no stray blank line.
  - id: BR-15
    disposition: not-addressed
    note: |
      A prose section on the / surface did land (README:92-115), but the editor key table (README:42-51) still has no / row and its "Enter | define what you typed" row is still incomplete — the specific structure the finding named is untouched.
  - id: BR-16
    disposition: not-addressed
    note: |
      The cmdCode plumbing is present and `echo /qqqqqq | define` exits 2 as probed, but nothing pins it: deleting the `if cmdCode != 0 { return cmdCode }` block leaves the suite green (mutation applied, BUILD_OK, -count=1 ok).
findings:
  - id: new
    severity: Critical
    family: elapsed-hours-for-calendar-days
    title: |
      relativeDay divides a Duration by 24h, so every /history date is off by one for a week after each spring-forward
    detail: |
      history_cmd.go:191 computes `int(dayOf(now).Sub(dayOf(at)).Hours() / 24)`. Two local
      midnights one calendar day apart are 23h across a spring-forward, and int(23.0/24) is 0.
      Measured against the shipped function body in America/Los_Angeles: at=2026-03-08,
      now=2026-03-09 prints "today" for a lookup that was yesterday; at=2026-03-07 prints
      "yesterday" for two days ago; at=2026-03-02, now=2026-03-09 prints "Monday" though the
      days<7 rule should give "Mar 2". The function's own comment says it is computed on local
      calendar days rather than elapsed hours, which is the contract it breaks, and it is a
      second implementation of what historyWindow already gets right with AddDate (ARCH-DRY).
      TestRenderHistoryRelativeDatesAreCalendarDays has two rows, both in August, while the
      sibling TestHistoryWindow has four DST rows and the file already embeds time/tzdata.
  - id: new
    severity: Critical
    family: runtime-output-tracked-in-source-tree
    title: |
      The developer's deck is committed under cmd/define/, and the conformance suite rewrites those tracked files
    detail: |
      cmd/define/events/2026-08-21.yaml, events/2026-08-22.yaml and words/sycophantic.yaml are
      tracked, added by fc071af and 1ff0d5e — two separate commits in this window. No test or
      fixture references them. Reproduced: after `go test -tags conformance -run PTY`, git status
      reports both events/2026-08-22.yaml and words/sycophantic.yaml MODIFIED (lookups 12 to 14),
      because pty_conformance_test.go:47 launches ../../bin/define with the test's cwd, and define
      writes its deck to the current directory. So the suite is not idempotent against the index
      and `git add -A` sweeps deck churn into unrelated commits. .gitignore:16-19 anticipates this
      class but anchors at the root, which does not cover cmd/*/. It is also the ARCH-MOCK gap:
      the live conformance flow and the in-process fake do not share a storage boundary.
  - id: new
    severity: Important
    family: unpinned-production-wiring
    title: |
      Four loop-shell wirings shipped this round with no test that fails without them
    detail: |
      This is the 4th finding in family unpinned-production-wiring (BR-2 first, BR-16's fix
      another). Do NOT fix the four instances individually. Measured, each mutation applied with
      BUILD_OK and the full suite green at -count=1: replraw.go:166 cc.setTimes deleted;
      main.go:292-294 one-shot dispatch deleted; repl.go:139-142 cmdCode return deleted;
      replraw.go:167 hist.Add deleted. The setTimes one is the sharpest — without it, /sound 1 at
      the real TUI prompt prints "needs an interactive session", the opposite of M3's requirement,
      and TestSoundChangesPlaybackForTheRestOfTheSession drives replLines, the piped loop.
      THE RULE - a value or effect that only a loop shell supplies must be pinned by a test that
      drives that loop shell (runEditor, replLines, run), never by one that builds the callee's
      context by hand. Every commandCtx literal in a test (sound_cmd_test.go:52,64,77,88;
      commandloop_test.go:131; history_cmd_test.go:271,354) is where the rule breaks invisibly.
      The rule-level fix is one table in commandloop_test.go running the same command scripts
      through BOTH runEditor and replLines, so a field wired in one loop and not the other reddens
      by construction.
  - id: new
    severity: Important
    family: entry-mode-inconsistency
    title: |
      define /history 7 prints a usage dump while echo '/history 7' | define runs it
    detail: |
      This is the 2nd finding in family entry-mode-inconsistency (BR-13 first). Do NOT patch this
      instance. Probed against the built binary - `define /history 7` exits 2 with the full usage
      text; `echo '/history 7' | define` prints "nothing looked up in the last 7 days" and exits 0.
      main.go:266 rejects fs.NArg() > 1 before main.go:292 tests for a command, and main.go:292
      classifies fs.Arg(0) alone rather than a line. README:96 claims all three entry modes are
      "one thing" and atlas/define.md says "Every entry mode reaches it" — true only for
      zero-argument commands.
      THE RULE - every entry mode must hand parseREPLLine the same input, a whole LINE. The two
      loops do; the argv path hands it one token. Classify strings.Join(fs.Args(), " ") when
      fs.Arg(0) opens command mode and dispatch before the arity guard, rather than extending the
      arity guard each time a command grows an argument. BR-13's fix stopped at the zero-argument
      case, which is why the family recurred.
  - id: new
    severity: Important
    family: plan-artifact-lags-code
    title: |
      The plan's Core-concepts table names none of the entities the two scope events added, and six step boxes are unticked
    detail: |
      This is the 2nd finding in family plan-artifact-lags-code (BR-5 first). Do NOT just tick the
      boxes. menuLines and menuNameWidth (M1b) and parseSoundArgs, runSound, soundTimes and the
      whole sound_cmd.go file (M3) appear in no Core-concepts row, though three of them do appear
      in the test-strategy tables. Six "- [ ]" boxes remain at plan lines 277, 300, 311, 312, 326
      — Task 6 Step 3 and the Step-5 commit boxes of Tasks 4-7 — which sdlc close's plan-unchecked
      guard reads.
      THE RULE - the Core-concepts table is the greppable contract this gate cross-checks against
      the filesystem, so a scope event that adds an entity adds its row in the same commit as the
      code, and a task's boxes are ticked by the commit that lands it. The M1b and M3 Revisions
      entries describe the work in prose but never amend the table, which is what made both gaps
      possible.
  - id: new
    severity: Minor
    family: same-setting-two-validation-policies
    title: |
      --sound 1000 is accepted while /sound 1000 is refused at the 20 cap
    detail: |
      Probed both. sound_cmd.go:33 bounds /sound at maxSoundTimes = 20; main.go:228 rejects only
      negatives for -sound/-times. atlas/define.md calls them the same setting.
  - id: new
    severity: Minor
    family: error-names-a-flag-the-user-did-not-type
    title: |
      define -times -1 reports "define: -sound must not be negative"
    detail: |
      main.go:229. main_test.go:253 asserts only the exit code, so the message text is unpinned in
      either direction.
  - id: new
    severity: Minor
    family: duplicated-width-arithmetic
    title: |
      menuLines truncates in bytes while renderHistory truncates in runes, and renderHistory measures columns in bytes but pads in runes
    detail: |
      command.go:239 slices line[:width] (bytes, can split a rune); history_cmd.go:172 slices
      runes. history_cmd.go's nameW/dateW use len() while %-*s pads in runes, so a non-ASCII
      headword widens the whole gutter — alignment survives, the column is just wider than asked
      for. One shared truncateToWidth helper covers both (ARCH-DRY).
  - id: new
    severity: Minor
    family: uncovered-branch-at-boundary
    title: |
      renderHistory's truncation branch is never exercised — both test call sites pass width 0
    detail: |
      history_cmd_test.go:227 and :250 pass 0, and nothing asserts runHistory forwards c.width.
      The equivalent branch in menuLines is covered by TestMenuLines' width=14 row.
```

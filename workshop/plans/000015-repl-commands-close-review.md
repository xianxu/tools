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

---

## Re-review — 2026-08-21T17:39:51-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 15 — REPL command mode: /-prefixed commands with type-ahead, starting with /history |
| repo | tools |
| issue file | workshop/issues/000015-repl-commands.md |
| boundary | whole-issue close |
| milestone | — |
| window | c7d07575cb125fcb438bcb3fe45ff3763282690b..4b019e080465c7ee3861969354ed2e58571d4007 |
| command | sdlc close --issue 15 |
| reviewer | claude |
| timestamp | 2026-08-21T17:39:51-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both Criticals from round 4 are genuinely closed, and I closed them by re-running the measurements that produced them rather than reading the commit message. Reverting `math.Round` to truncation reddens three DST rows in `TestRenderHistoryRelativeDatesAreCalendarDays`; planting a tracked deck file reddens `TestNoTrackedRuntimeState`; and running the conformance suite on a real pty (all four PTY tests PASS, unsandboxed) now leaves `git status` clean and creates no deck under `cmd/define/`. `go vet`, `gofmt -l`, and `go test -count=1` are all clean. Nothing new is Critical, so this does not block. What keeps it off SHIP is that the two Important findings the round asked for as *rules* shipped as unpinned code. `define /history 7` works — I probed it — but I applied two separate mutations to its fix (revert the arity-guard exemption; classify `fs.Arg(0)` instead of the joined line) and the **full suite stayed green under both**, with the first restoring BR-20's exact reported symptom and the second silently printing a 2-day window for `/history 7`. That is BR-19's own rule violated by the commit that closed BR-19, and BR-19's rule was never written to `workshop/lessons.md` (three entries landed this round; none of them is it). Separately, the BR-20 fix exempts command lines from the arity guard, which drops them past `d = d.withStore(...)` — so `define /qqqqqq` now reads the whole event log and emits a torn-record recovery warning before reporting the usage error, contradicting the invariant stated four lines above it at `main.go:252`.

## 1. Strengths

- **The BR-17 fix is pinned and the diagnosis was right.** Reverting `history_cmd.go:199` to `int(…Hours()/24)` reddens exactly the three predicted rows ("today" for yesterday, "yesterday" for two days ago, "Monday" where "Mar 2" is due). The four DST rows added to `TestRenderHistoryRelativeDatesAreCalendarDays` are the corpus the sibling window test always had and the render test lacked.
- **BR-18 is fixed at the level that actually holds.** Un-anchored `words/`/`events/` is the load-bearing change, and `TestNoTrackedRuntimeState` is plant-verified in both directions — I added `cmd/define/words/planted.yaml` to the index and it failed with the file named. `cmd.Dir = t.TempDir()` genuinely closes the ARCH-MOCK half: the live conformance flow and the in-process fake now share a storage boundary, verified by a clean tree after a real-pty run.
- **The four BR-19 instances are each mutation-verified.** I deleted all four wirings independently: `cc.setTimes` → `TestRawEditorSoundChangesTheSession` fails ("played 3 times … want 1"); `hist.Add` → `TestRawEditorRecallsSubmittedCommands` fails; `cmdCode` return → exit 1 instead of 2; one-shot dispatch → `refusingDict` fires. The recall test's fix (asserting only after `"playing"`) is the right repair for the echo-shares-the-observable trap.
- **`TestNoTrackedRuntimeState` refuses to pass vacuously** (`t.Fatal` when `seen == 0`), matching the doctrine `scanForExecutables` established. `TestEveryRegisteredCommandIsRunnable` does the same against the live registry.
- **BR-21 is fully closed** — zero `- [ ]` boxes remain, and every entity in the Core-concepts table (`menuLines`, `menuNameWidth`, `parseSoundArgs`, `soundTimes`, `runSound`, `relativeDay`, `historyRow`, `days2str`, `noDeckMessage`) exists at its stated path.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — a command line skips the "usage errors are settled before a store is opened" invariant, and reads the whole event log.** `cmd/define/main.go:272` exempts `cmdCommand` from the arity guard, so command lines fall through to `main.go:279` `d = d.withStore(opt, stderr)` before anything validates the command. Measured with the same torn-log fixture `TestUsageErrorsDoNotOpenTheLog` uses:

```
define /qqqqqq        → "define: 2020-01-01.yaml: recovered 0 event(s), dropped 1 torn record(s)"
                        "define: unknown command /qqqqqq. Commands: /help /history /sound"  (exit 2)
define /qqqqqq zzz    → same
define /history zzz   → same warning, then "\"zzz\" is not a number of days"  (exit 2)
```

`main.go:252` states the contract four lines above the guard that breaks it: *"Usage errors are settled BEFORE a store is opened. A mistyped command must not be the thing that creates words/ and events/."* The directory-creation half still holds (`NewYAML` is lazy — I confirmed no `words/`/`events/` appears), but the read half does not, and `TestUsageErrorsDoNotOpenTheLog` (`capture_test.go:423`) has rows for `-forget` and two words and none for a command. Fix sketch: resolve the command name against `commands` before `withStore` and return the unknown-command error there, or move `withStore` below the dispatch; either way add the three rows above to `TestUsageErrorsDoNotOpenTheLog`.

**I-2 — `--days` / `--days=N` ship, are tested, and are documented nowhere; the plan box promising them is ticked.**

> **This is the 2nd finding in family `docs-consumer-not-updated`** (BR-4 was the first). Do NOT patch this instance — state the rule and fix that.

`parseHistoryArgs` accepts `7`, `--days 7`, and `--days=7`, all pinned by `TestParseHistoryArgs`. `grep -- "--days"` finds zero occurrences in `README.md`, zero in `atlas/define.md`, and zero in the `--help` text; `/help`'s own summary is "words looked up recently". Only the positional `[N]` form is documented. Meanwhile plan line 352 reads `- [x] **Step 2:** README: what /history shows, what it omits and why, --days.` — a ticked box asserting delivery of the one thing that did not land. **The rule:** *a flag or argument form is not shipped until at least one consumer a user can reach derives it — the box is ticked by the commit that updates that consumer, not by the commit that adds the parser.* Shadow-sweep of "what windows can I ask for": parser ✓, tests ✓, README ✗, atlas ✗, `--help` ✗, `/help` summary ✗ (ARCH-PURPOSE).

**I-3 — the deck blobs are still reachable from HEAD, and the guard written for BR-18 cannot see them.**

> **This is the 2nd finding in family `runtime-output-tracked-in-source-tree`** (BR-18 was the first). Do NOT patch this instance — state the rule and fix that.

`git show 1ff0d5e:cmd/define/words/sycophantic.yaml` still returns the developer's word with `first_seen`/`last_seen` timestamps and `lookups: 12`; `cmd/define/events/2026-08-21.yaml` and `2026-08-22.yaml` likewise, added by `fc071af` and `1ff0d5e`. `4b019e0` removed them from the index only. `TestNoTrackedRuntimeState` checks `git ls-files` — the index — while its sibling `TestNoBinariesInHistory` walks *history* precisely because, in that test's own words, *"deleting it in a later commit does not remove the cost."* **The rule:** *a guard for a committed-artifact class must check the scope where that class's cost lives — the index for what is checked out, history for what every clone fetches — and a fix that only untracks is incomplete against a history-scoped guard.* Cheap now (the branch is unmerged); permanent after merge. Either rewrite `fc071af` and `1ff0d5e`, or extend `TestNoTrackedRuntimeState` to the history scope and record explicitly that the deck content is accepted as public.

## 4. Minor findings

- **`relativeDay`'s rounding is a heuristic where an exact computation is one line away, and the comment again asserts a property the code lacks.** *This is the 2nd finding in family `elapsed-hours-for-calendar-days`* — do not patch the instance. `history_cmd.go:199`'s comment says "The true gap is always N days ± 1 hour, which makes rounding exact"; measured in `Pacific/Apia`, `at=2011-12-29`, `now=2011-12-31` prints **"yesterday"** for a two-calendar-day gap, because Samoa's date-line change deleted 2011-12-30 and only 24 hours elapsed. **The rule:** *count calendar days on calendar-day numbers, never on elapsed time — normalise both dates into a fixed-offset zone before differencing, so no offset arithmetic enters the count.* Verified: changing `dayOf` to build in `time.UTC` and dropping `math.Round` makes Apia return "Thursday", keeps the whole suite green including all four DST rows, and removes the `math` import.
- **`replraw.go:67` is orphaned by `1ff0d5e`.** *2nd in family `comment-orphaned-by-insertion`* — do not patch the instance. It still reads "Resolve candidates ONCE per keystroke and use the same slice for both the state machine and the suggestion. Querying twice doubled the work the History seam will do once #3 backs it with a store." Since `draw()` computes its own list, `completionsFor` now runs twice per keystroke — the exact thing the comment says was avoided, and `#3` has landed. **The rule:** *when a fix changes what a block does, the comment above it is part of the diff; a comment that survives a behaviour change is a false claim, which is how BR-17 stayed invisible.*
- BR-15's key table (`README.md:43-50`) is still untouched: no `/` row, and `Enter | define what you typed` still omits command dispatch. The `sh` block half of that finding is now fine.
- `--sound 1000` is still accepted while `/sound 1000` is refused at 20 (BR-22, re-probed).
- `define -times -1` still reports `define: -sound must not be negative` (BR-23, re-probed).
- `command.go:239` truncates in bytes, `history_cmd.go:178` in runes (BR-24, unchanged).
- `renderHistory`'s truncation branch still has no test at non-zero width (BR-25, unchanged).
- `relativeDay` with a future `at` (clock skew, or a deck synced from a machine ahead) gives negative days, falls into `days < 7`, and prints a weekday name.

## 5. Test coverage notes

- Verified green: `go build ./...`, `go vet ./cmd/...`, `gofmt -l cmd/` (empty), `go test ./cmd/define/... -count=1`, and `go test -tags conformance -run PTY` — all four pty tests PASS on a real terminal, run unsandboxed.
- Verified pinned by reverting the fix: BR-17 (3 DST rows redden), BR-18 (planted index entry reddens `TestNoTrackedRuntimeState`), and all four BR-19 wirings.
- Verified **unpinned**: both halves of BR-20's fix (two mutations, full suite green each time, one restoring the reported symptom and one producing a silent wrong window), and `clearMenu()` before submit at `replraw.go:150` — deleting it leaves both the unit suite and the pty suite green. I diffed the real pty frames with and without it: at HEAD the three menu rows are erased before the echoed line; without it they are overwritten only by luck, since `/` draws three rows and `runHelp` prints three. A prefix like `/h` (two menu rows) followed by Enter (one line of stderr) leaves a row behind.
- Verified **uncovered**: `TestUsageErrorsDoNotOpenTheLog` has no command row (I-1); `renderHistory` at non-zero width (BR-25).

## 6. Architectural notes

- **ARCH-DRY — flag.** `parseREPLLine` as the one classifier and `completionsFor` as the one namespace switch both hold, and `noDeckMessage` is genuinely shared. Flagged: `relativeDay` remains a second, heuristic implementation of the calendar-day concept `historyWindow` owns exactly (Minor above); `menuLines`/`renderHistory` still carry divergent truncate-and-pad logic (BR-24).
- **ARCH-PURE — pass.** Every PURE row in the Core-concepts table runs its tests with no store, clock, terminal, or network — I confirmed none needs a mock. `runHistory`/`runSound`/`dispatchCommand` take writers and are the thin shell, and `commandCtx` narrower than `deps` keeps "a command cannot reach the dictionary or the player" structural. The caution stands: cheap purity is what makes the wiring gaps easy to miss.
- **ARCH-PURPOSE — flag.** The entry-mode sweep now passes: `define /history 7`, `echo '/history 7' | define`, and the raw prompt all reach `parseREPLLine` with a whole line, which is what BR-20 asked for behaviourally. The failing sweep is `--days` (I-2): the parser is the only consumer that derives it.
- **ARCH-MOCK — pass.** `store.Mem` and `store.FixedClock` are production files in `cmd/define/store/`, not test doubles, and `storeHistory.Add` is memory-only, so submitted commands never contaminate the deck (checked — `/sound` lines are recall-only). BR-18's ARCH-MOCK half is closed: production flow and conformance flow now share the storage boundary, verified by a clean tree after a real-pty run.

## 7. Plan revision recommendations

- **Untick Task 8 Step 2, or land the README `--days` text it claims** (plan line 352). It is currently the only ticked box in the plan whose deliverable does not exist, which inverts BR-21's failure mode rather than closing it.
- **Add a `## Revisions` entry recording that BR-19's rule was not durably captured.** The commit message and this plan's Revisions state it; `workshop/lessons.md` — the artifact AGENTS.md §4 designates for cross-issue rules — gained three entries this round and none of them is it. The rule belongs there, because the family's next instance will be in a different file.
- **Record `relativeDay`'s exactness caveat where the plan already discusses the local-day question**, or take the one-line UTC-normalised fix. The plan's Chunk 1 makes the case for `AddDate` over Durations for `historyWindow`; `relativeDay` now sits one section away using a different, weaker technique for the same concept.

```findings
dispose:
  - id: BR-13
    disposition: addressed
    note: |
      Probed: `define /help` runs and exits 0; pinned by TestOneShotCommandGoesThroughRun (deleting the dispatch block reddens it).
  - id: BR-15
    disposition: not-addressed
    note: |
      The sh block is fine now, but README.md:43-50 still has no / row and "Enter | define what you typed" still omits command dispatch — the exact structure the finding named.
  - id: BR-16
    disposition: addressed
    note: |
      Probed `echo /qqqqqq | define` exits 2; mutation-verified — deleting the cmdCode return reddens TestPipedLoopReturnsTheCommandsExitCode.
  - id: BR-17
    disposition: addressed
    note: |
      Mutation-verified: reverting math.Round to truncation reddens three DST rows. Residual exactness gap raised separately as a Minor.
  - id: BR-18
    disposition: addressed
    note: |
      Files untracked, TestNoTrackedRuntimeState plant-verified to fail, and a real-pty conformance run leaves git status clean with no deck under cmd/define/. History residue raised separately.
  - id: BR-19
    disposition: not-addressed
    note: |
      All four instances are now mutation-verified pinned, but the RULE the finding demanded is absent from workshop/lessons.md and was violated in the same commit — BR-20's fix and clearMenu are both unpinned (measured).
  - id: BR-20
    disposition: not-addressed
    note: |
      Behaviour correct when probed, but unpinned: reverting the arity-guard exemption restores the reported usage dump, and classifying fs.Arg(0) makes /history 7 print a 2-day window — full suite green under both.
  - id: BR-21
    disposition: addressed
    note: |
      Zero unticked boxes remain, and every entity named in the Core-concepts table exists at its stated path (grep-verified).
  - id: BR-22
    disposition: not-addressed
    note: |
      Re-probed: --sound 1000 accepted, /sound 1000 refused at the 20 cap.
  - id: BR-23
    disposition: not-addressed
    note: |
      Re-probed: define -times -1 still reports "define: -sound must not be negative".
  - id: BR-24
    disposition: not-addressed
    note: |
      command.go:239 still slices bytes, history_cmd.go:178 still slices runes.
  - id: BR-25
    disposition: not-addressed
    note: |
      Both renderHistory call sites still pass width 0, and nothing asserts runHistory forwards c.width.
findings:
  - id: new
    severity: Important
    family: guard-bypassed-by-new-kind
    title: |
      A command line skips the "usage errors are settled before a store is opened" invariant and reads the whole event log
    detail: |
      main.go:272 exempts cmdCommand from the arity guard, so a command line reaches
      main.go:279 d.withStore before anything validates it. Measured with the same torn-log
      fixture TestUsageErrorsDoNotOpenTheLog uses - `define /qqqqqq`, `define /qqqqqq zzz`
      and `define /history zzz` all print "recovered 0 event(s), dropped 1 torn record(s)"
      before their usage error. main.go:252 states the contract four lines above the guard
      that breaks it. The directory-creation half still holds (NewYAML is lazy, confirmed),
      but the read half does not, and capture_test.go:423 has rows for -forget and two words
      and none for a command, so the drift is untested in either direction. Fix by resolving
      the command name against `commands` before withStore, and add the three rows.
  - id: new
    severity: Important
    family: docs-consumer-not-updated
    title: |
      --days and --days=N ship and are tested but appear in no user-facing doc, while the plan box promising them is ticked
    detail: |
      This is the 2nd finding in family docs-consumer-not-updated (BR-4 first). Do NOT patch
      this instance. parseHistoryArgs accepts 7, --days 7 and --days=7, all pinned by
      TestParseHistoryArgs; grep finds zero occurrences of --days in README.md, atlas/define.md
      and the --help text, and /help's summary is "words looked up recently". Only the
      positional [N] form is documented. Plan line 352 reads "- [x] Step 2: README: what
      /history shows, what it omits and why, --days" - a ticked box asserting the one
      deliverable that did not land.
      THE RULE - a flag or argument form is not shipped until at least one consumer a user can
      reach derives it, and the plan box is ticked by the commit that updates that consumer,
      not by the commit that adds the parser (ARCH-PURPOSE shadow-sweep: parser and tests
      derive, README/atlas/--help//help do not).
  - id: new
    severity: Important
    family: runtime-output-tracked-in-source-tree
    title: |
      The deck blobs are still reachable from HEAD, and the guard written for BR-18 checks only the index
    detail: |
      This is the 2nd finding in family runtime-output-tracked-in-source-tree (BR-18 first). Do
      NOT patch this instance. `git show 1ff0d5e:cmd/define/words/sycophantic.yaml` still
      returns the word with its timestamps and lookups: 12; the two events files likewise, added
      by fc071af and 1ff0d5e. 4b019e0 removed them from the index only. TestNoTrackedRuntimeState
      reads `git ls-files` - the index - while its sibling TestNoBinariesInHistory walks history
      precisely because, in its own words, deleting in a later commit does not remove the cost.
      THE RULE - a guard for a committed-artifact class must check the scope where that class's
      cost lives: the index for what is checked out, history for what every clone fetches. A fix
      that only untracks is incomplete against a history-scoped guard. Cheap now while the branch
      is unmerged; permanent after.
  - id: new
    severity: Minor
    family: elapsed-hours-for-calendar-days
    title: |
      relativeDay's rounding is a heuristic where an exact computation is one line away, and the comment asserts an exactness the code lacks
    detail: |
      This is the 2nd finding in family elapsed-hours-for-calendar-days (BR-17 first). Do NOT
      patch this instance. history_cmd.go:199's comment claims "The true gap is always N days
      +/- 1 hour, which makes rounding exact"; measured in Pacific/Apia, at=2011-12-29,
      now=2011-12-31 prints "yesterday" for a two-calendar-day gap, because the 2011 date-line
      change deleted 2011-12-30 and only 24 hours elapsed.
      THE RULE - count calendar days on calendar-day numbers, never on elapsed time: normalise
      both dates into a fixed-offset zone before differencing, so no offset arithmetic enters
      the count at all. Verified - building dayOf in time.UTC and dropping math.Round returns
      "Thursday" for Apia, keeps the full suite green including all four DST rows, and removes
      the math import.
  - id: new
    severity: Minor
    family: comment-orphaned-by-insertion
    title: |
      replraw.go:67 still claims candidates are resolved once per keystroke, but draw() now computes its own list
    detail: |
      This is the 2nd finding in family comment-orphaned-by-insertion (BR-12 first). Do NOT patch
      this instance. The comment reads "Resolve candidates ONCE per keystroke and use the same
      slice for both the state machine and the suggestion. Querying twice doubled the work the
      History seam will do once #3 backs it with a store." Since 1ff0d5e removed draw's parameter,
      completionsFor runs twice per keystroke - once for Apply at replraw.go:129 and once inside
      draw - and #3 has landed.
      THE RULE - when a fix changes what a block does, the comment above it is part of the diff.
      A comment that survives a behaviour change is a false claim about the code beneath it,
      which is exactly what let BR-17's DST bug read as correct for four rounds.
```

---

## Re-review — 2026-08-21T18:00:13-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 15 — REPL command mode: /-prefixed commands with type-ahead, starting with /history |
| repo | tools |
| issue file | workshop/issues/000015-repl-commands.md |
| boundary | whole-issue close |
| milestone | — |
| window | c7d07575cb125fcb438bcb3fe45ff3763282690b..e5719f936bc7e0ce03b642a60993326ad4e1e92b |
| command | sdlc close --issue 15 |
| reviewer | claude |
| timestamp | 2026-08-21T18:00:13-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Both round-5 Importants that carried real behaviour — BR-20 and BR-26's main class — are genuinely closed, and I closed them by reverting the fix rather than reading the commit message: reverting the arity-guard exemption and, separately, classifying `fs.Arg(0)` instead of the joined line each redden `TestOneShotCommandTakesArguments`; forcing `withStore` unconditionally reddens `TestCommandThatReadsNothingDoesNotOpenTheLog`; deleting `clearMenu()` before submit reddens `TestSubmitClearsTheMenuBeforeOutput`. BR-28 is fixed at the scope where it costs — `git rev-list --objects HEAD` returns zero deck objects, the three named commits no longer exist, and `TestNoRuntimeStateInHistory` now shares `historyPaths` with the binary guard. `go build`, `go vet`, `gofmt -l`, `go test -count=1`, `go test -race`, and the four pty conformance tests on a real terminal (unsandboxed) are all clean, and the conformance run leaves `git status` empty with no deck under `cmd/define/`. Nothing here is Critical, so it does not block. What keeps it off SHIP is that the `needsDeck` exemption BR-26's fix introduced drops an invariant nobody re-checked: `withStore` was the only thing supplying `deps.clock`, so a registry row with `needsDeck: false` now receives a **nil** clock — I added one such row and `define /probe` panicked with a nil dereference at `command.go`, while `TestOpenStoreSuppliesOneClock` and `TestWithStoreCarriesTheClock`, the two tests written to prevent exactly that, stayed green because they test the path the exemption routes around. Separately, `/history` reads the whole event log **twice** and prints every store warning twice (measured), against an atlas that records "One read per `/history`" as the accepted cost.

## 1. Strengths

- **BR-28 was fixed at the scope where the cost lives, and the commit message says how it nearly wasn't.** `git rev-list --objects HEAD | grep -E '(^|/)(words|events)/'` returns zero; `git cat-file -t 1ff0d5e` / `fc071af` / `4b019e0` all report "not a valid object name". `historyPaths` (`repo_guard_test.go:216`) is now shared by both history guards, and both refuse to pass vacuously. The message's note that the first filter-branch attempt failed while clone size alone would have read as success is the right lesson recorded at the right moment.
- **`TestCommandThatReadsNothingDoesNotOpenTheLog` asserts its own discriminating power.** After checking `/help` and `/histry` stay silent against a torn log, it runs `/history` and requires the warning *to appear* — without that last assertion the test could not tell "did not read it" from "there was nothing to read". That is the fix for the marker-matched-the-wrong-thing class the same commit adds to `lessons.md`.
- **BR-19's rule landed where it will be re-read** (`workshop/lessons.md:367`), and both instances the round-5 review said the fix itself created are now mutation-pinned. The `TestRawEditorRecallsSubmittedCommands` comment naming its own near-miss ("that is BR-3's mistake, repeated inside the fix for BR-19") is honest in a way that survives.
- **The plan's Core-concepts table cross-checks clean against the filesystem** — I grepped all 28 named entities at their stated paths and every one exists, PURE rows in the files the table claims and INTEGRATION rows likewise.
- **`storeHistory.Add` really is memory-only** (`history_store.go:44-57`), so `hist.Add(submitted.String())` in the raw editor recalls `/sound` without any command ever reaching the deck — the recall seam and the capture seam stay separate, as `atlas/define.md` claims.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — the `needsDeck` exemption strands `deps.clock`, and a registry row is one line from a nil dereference.**

> **This is the 2nd finding in family `guard-bypassed-by-new-kind`** (BR-26 is the first, still open below). Do NOT patch this instance — state the rule and fix that.

`cmd/define/main.go:285` now reads `if oneShot.kind != cmdCommand || commandNeedsDeck(oneShot, commands) { d = d.withStore(opt, stderr) }`. `realDeps()` (`main.go:50`) never sets `clock`; `withStore` (`main.go:99`) is the *only* thing that does. So for every command with `needsDeck: false`, `newCommandCtx` (`command.go:184`) copies a nil `d.clock` into `commandCtx.clock`. Measured: I added `{name: "probe", needsDeck: false, run: …c.clock.Now()…}` to the registry and `define -no-audio /probe` panicked —

```
panic: runtime error: invalid memory address or nil pointer dereference
main.dispatchCommand(...) cmd/define/command.go:214
```

— with the whole suite still green. `capture_test.go:468` and `:494` exist precisely to stop this ("a nil clock there is a panic waiting for the first opt-out user to run a command") and both pass, because they exercise `openStore`/`withStore`, which this branch skips. The issue's Done-when says *"Adding a second command needs no change to the dispatch loop"*; it now carries an undocumented precondition — declare `needsDeck` or you get no clock — that neither `atlas/define.md`'s "A command declares `needsDeck` when it reads the store" nor the `commandCtx` doc comment mentions.

**The rule:** *when you exempt a new kind from a shared setup path, enumerate everything that path guaranteed and re-supply it — an exemption drops invariants you were not thinking about, not just the cost you were avoiding.* BR-26 was written about one cost (the log read) and the fix dropped a second guarantee (a non-nil clock) in the same edit. Rule-level fix: the clock has no store dependency (`store.SystemClock()` is a constructor over nothing), so supply it *outside* `withStore` — in `run()` before the branch, or as a fallback inside `newCommandCtx` — so no exemption can strand it. Then extend `TestEveryRegisteredCommandIsRunnable` to assert the ctx a one-shot actually builds has every field a row is allowed to read, which makes the registry contract checkable rather than conventional.

**I-2 — `/history` reads the whole event log twice, and prints every store warning twice.** Measured against a torn-log fixture, all entry modes:

```
$ define -no-audio /history
define: 2020-01-01.yaml: recovered 1 event(s), dropped 1 torn record(s)
define: 2020-01-01.yaml: recovered 1 event(s), dropped 1 torn record(s)
  nothing looked up in the last 2 days
$ echo /history | define -no-audio        # identical, also doubled
$ define -no-audio zzzznotaword           # one warning — storeHistory's read
$ define -no-audio /help                  # zero — BR-26's fix
```

Read 1 is `newStoreHistory` → `st.Events(time.Time{})` (`history_store.go:31`), built by `withStore` because `commandNeedsDeck` said yes. Read 2 is `runHistory` → `c.deck.Events(time.Time{})` (`history_cmd.go:228`). In the one-shot path the first read is pure waste — nothing on that path consults recall — and `replLines` never touches `d.history` at all, so it is waste in the piped path too. `atlas/define.md:446` and `plan:374` both record "**One read per `/history`** on a personal word list is the right trade today" as a deliberate decision; the code does two, so the recorded trade was never the one being made. The doubled diagnostic is the user-visible half. Fix sketch: gate on what is actually needed rather than on "open the store bundle" — `needsDeck` should yield a `store.Store`, not a full `withStore`; or have `runHistory` receive the events `storeHistory` already read. Pin it with a warning-count assertion in `TestCommandThatReadsNothingDoesNotOpenTheLog`, which today only checks presence.

**I-3 — the atlas never learned about the runtime-state guards this window added, and the README still teaches the deprecated flag name.**

> **This is the 3rd finding in family `docs-consumer-not-updated`** (BR-4, BR-27). Earlier rounds fixed instances. Do NOT patch these — state the rule and fix that.

Measured prevalence, both from this window: (1) `atlas/repo-guards.md` documents exactly one guard class — "No executable image is ever tracked, or ever reachable" — with an index/history table naming `TestNoCommittedBinaries` and `TestNoBinariesInHistory`. This window added a second class with the identical split (`TestNoTrackedRuntimeState` / `TestNoRuntimeStateInHistory`) and the same doctrine, and neither that file nor `atlas/index.md:13-14` ("no executable image in the index or reachable from `HEAD`") mentions it — so the index page now understates what the file it points at enforces. (2) `README.md:84` still reads "Flags are session settings — `define -times 1` opens the loop with single playback," teaching the name that `README.md:118` calls "the older name for `--sound`" thirty lines later.

**The rule:** *a surface is not shipped until every doc that already describes its class is updated in the same commit — and "its class" means the page that would be wrong by omission, not just the page that names the new thing.* BR-4's fix updated the README because the finding named the README; BR-27's updated `--days` in three places because the finding listed three. Neither adopted the sweep, which is why a whole new guard class landed with an atlas page one directory away that enumerates guards and does not enumerate it. The enforcement worth having: `atlas/repo-guards.md`'s table is a hand-maintained restatement of the `Test*` functions in `repo_guard_test.go` (ARCH-PURPOSE — a restatement that does not derive is a deferred consumer). A test that greps the atlas table for every `func TestNo…` in that file would make the page derive instead of drift.

**I-4 — the Core-concepts table lost five entities again, and the plan has no Revisions entry for round 5 or its fixes.**

> **This is the 3rd finding in family `plan-artifact-lags-code`** (BR-5, BR-21). Do NOT just add the rows.

`commandNeedsDeck`, `command.needsDeck`, `editDistance`, `historyPaths` and `TestNoRuntimeStateInHistory` appear in no row (grep-verified: zero occurrences of each name in the plan). Four of the five were added by `e5719f9`, the commit that closed BR-21's own family. The plan's last `## Revisions` heading is "close round 4"; rounds 5 and 6 — which produced the `needsDeck` design, the history rewrite, and the `--days` docs — are unrecorded, so the artifact the gate cross-checks stopped tracking the design two commits ago. Box-ticking is clean (zero unticked).

**The rule:** *the Core-concepts table is a hand-maintained restatement of what the package declares, so it drifts by default; the fix is to make it derive, not to remember harder.* BR-21 was disposed `addressed` and the very next commit re-broke it, which is the measurement that a habit-level fix does not hold here. A `go test` guard that compares the table's `Name | path` pairs against top-level declarations in the named files would fail the same commit that adds an unnamed entity — the same shape as `TestNoTrackedRuntimeState`, which this repo has already shown itself willing to write.

## 4. Minor findings

- **Two registry lookups, two matching rules, and the agreement is unpinned.** `commandNeedsDeck` (`command.go:192-199`) and `dispatchCommand` (`command.go:211-215`) each walk `cmds` with `strings.EqualFold`. Mutating `commandNeedsDeck` to `cmd.name == c.name` leaves the **entire suite green** while `define /HISTORY` reports `define: no deck in this directory` and exits 1 — a lie about the user's data — because dispatch still accepts loosely and the deck gate no longer does. *2nd in family `parallel-construction-drift`* (BR-6 first); the rule is that two sites that must agree about one fact should be one site — a `findCommand(name, cmds) (command, bool)` both call.
- BR-15's key table (`README.md:43-50`) is still untouched: no `/` row, and `Enter | define what you typed (never the suggestion)` still omits command dispatch.
- `--sound 1000` prints `playing 1000×` while `/sound 1000` is refused at the cap (BR-22, re-probed).
- `define -times -1` still reports `define: -sound must not be negative` (BR-23, re-probed).
- `menuLines` truncates bytes (`command.go:255`), `renderHistory` truncates runes (`history_cmd.go:177`) (BR-24, unchanged).
- `renderHistory`'s truncation branch still has no non-zero-width test, and nothing asserts `runHistory` forwards `c.width` (BR-25, unchanged).
- `relativeDay` still rounds elapsed hours, comment unchanged (BR-29); `replraw.go:67-69` still claims candidates are resolved once per keystroke while `completionsFor` runs at both `:113` and `:129` (BR-30).
- `relativeDay` with a future `at` (clock skew, or a deck synced from a machine ahead) yields negative days, falls into `days < 7`, and prints a weekday name. Noted in prose in rounds 4 and 5; still true.
- `define /help zzz` silently ignores its arguments rather than reporting them, where `/history` and `/sound` both refuse a surplus operand.

## 5. Test coverage notes

- Verified green: `go build ./...`, `go vet ./cmd/...`, `gofmt -l cmd/` (empty), `go test ./cmd/define/... -count=1`, `go test -race`, and `go test -tags conformance -run PTY -v` — all four pty tests **PASS** on a real terminal, run unsandboxed, and `git status` is empty afterwards with no `cmd/define/words` or `cmd/define/events`.
- Verified pinned by reverting the fix (each mutation compiled, full suite `-count=1`): BR-20 arity exemption → `TestOneShotCommandTakesArguments` fails naming the usage dump; BR-20 line-vs-token classification → same test fails; BR-26 unconditional `withStore` → `TestCommandThatReadsNothingDoesNotOpenTheLog` fails; `clearMenu()` before submit → `TestSubmitClearsTheMenuBeforeOutput` fails.
- Verified **unpinned**: the clock reaching a one-shot command (I-1 — no test builds the ctx `run()` actually builds); `commandNeedsDeck`'s case policy (Minor above — case-sensitive mutation leaves the suite green); `renderHistory` at non-zero width (BR-25); `define /history zzz` not reading the log (the third row BR-26 asked for and the one that still fails).
- `TestNoRuntimeStateInHistory` and `TestNoTrackedRuntimeState` both `t.Fatal` rather than pass vacuously, matching the doctrine `scanForExecutables` established.

## 6. Architectural notes

- **ARCH-DRY — flag.** `parseREPLLine` as the one classifier and `completionsFor` as the one namespace switch both hold, `noDeckMessage` is genuinely shared, and `historyPaths` extracted for both history guards is a real consolidation this round. Flagged: the event log is read into two independent in-memory shapes per `/history` (I-2) — one source, two readers that do not know about each other; the registry is walked by two functions with independently-written matching rules (Minor); `relativeDay` vs `historyWindow` (BR-29) and `menuLines` vs `renderHistory` truncation (BR-24) both still stand.
- **ARCH-PURE — pass.** Every PURE row in the Core-concepts table runs its tests with no store, clock, terminal or network — I confirmed none needs a mock, and `memHistory`/`store.Mem`/`store.FixedClock` are production code behind the same interfaces the YAML store implements. `runHistory`/`runSound`/`dispatchCommand` take writers and are the thin shell; `commandCtx` narrower than `deps` keeps "a command cannot reach the dictionary or the player" structural. The caution stands and I-1 is its instance: proving the pure half is cheap, so the assembly of `commandCtx` at the boundary is where the holes are.
- **ARCH-PURPOSE — flag.** Shadow-sweep of "what commands exist and what arguments do they take": the `commands` registry is the source; `/help`, `menuLines`, `completionsFor`, `nearestCommands` and `commandNeedsDeck` all derive ✓; README, `atlas/define.md` and `--help` are hand-maintained restatements that are now *accurate* for `--days` (BR-27 closed) but stale for the deprecated flag name (I-3). Sweep of "what invariants do the repo guards enforce": `repo_guard_test.go` is the source, `atlas/repo-guards.md` is a restatement that no longer derives (I-3). Entry-mode sweep passes: `define /history 7`, `echo '/history 7' | define` and the prompt all hand `parseREPLLine` a whole line.
- **ARCH-MOCK — pass.** BR-18's half of this is closed and I re-measured it: `cmd.Dir = t.TempDir()` means the live conformance flow and the in-process fake share the storage boundary, and a real-pty run leaves the tree clean. Live conformance checks exist behind build tags for the dictionary, the fetcher and the player, and `refusingDict` asserts a negative interaction rather than an output shape.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000015-repl-commands-plan.md` covering rounds 5–6 (the last entry is "close round 4"), and within it:

- **Amend the Core-concepts table** with `commandNeedsDeck` and `editDistance` (PURE, `cmd/define/command.go`), `command.needsDeck` as a modified row on `command`, and `historyPaths` / `TestNoRuntimeStateInHistory` (INTEGRATION, `cmd/define/repo_guard_test.go`) — and record that the table has now gone stale twice after being repaired, which is the argument for deriving it rather than maintaining it (I-4).
- **Correct the stated cost.** Plan line 374 and `atlas/define.md:446` both say "One read per `/history`"; it is two (I-2). Either fix the code and keep the claim, or restate the claim — a recorded design decision that the code does not implement is the same failure mode as BR-17's comment.
- **Record the `needsDeck` precondition.** The Integration-points section describes `commandCtx` as "the deps a command may touch" without noting that a `needsDeck: false` row receives a nil `deck` *and* a nil `clock`. Until I-1 is fixed at the rule level, that precondition is the registry's real contract and belongs next to the row that states the Done-when it qualifies.
- **Correct the Task 2 Step 3 signature**, still open from the M1 review: the plan specifies `completionsFor(line string, hist History) []string`; the shipped function is `completionsFor(base string, hist History, cmds []command) []string`.

```findings
dispose:
  - id: BR-15
    disposition: not-addressed
    note: |
      README.md:43-50 still has no / row and "Enter | define what you typed (never the suggestion)" still omits command dispatch.
  - id: BR-19
    disposition: addressed
    note: |
      Rule recorded at lessons.md:367; both new instances mutation-verified (arity exemption and clearMenu each redden a loop-driving test).
  - id: BR-20
    disposition: addressed
    note: |
      Both halves mutation-verified — reverting the arity exemption and classifying fs.Arg(0) each redden TestOneShotCommandTakesArguments.
  - id: BR-22
    disposition: not-addressed
    note: |
      Re-probed against the built binary — "define --sound 1000 /sound" prints "playing 1000x" and exits 0.
  - id: BR-23
    disposition: not-addressed
    note: |
      Re-probed — "define -times -1 x" still reports "define: -sound must not be negative".
  - id: BR-24
    disposition: not-addressed
    note: |
      command.go:255 still slices bytes, history_cmd.go:177 still slices runes.
  - id: BR-25
    disposition: not-addressed
    note: |
      Both renderHistory call sites still pass width 0, and nothing asserts runHistory forwards c.width.
  - id: BR-26
    disposition: not-addressed
    note: |
      Two of the three measured cases are fixed and mutation-pinned, but "define /history zzz" still reads the whole log before its usage error, and that is the one row the finding asked for that was not added.
  - id: BR-27
    disposition: addressed
    note: |
      --days now derives in three reachable consumers — README.md:102, atlas/define.md:401, and the --help text at main.go:206.
  - id: BR-28
    disposition: addressed
    note: |
      git rev-list --objects HEAD returns zero deck objects, the three named commits no longer exist, and TestNoRuntimeStateInHistory covers the history scope.
  - id: BR-29
    disposition: not-addressed
    note: |
      history_cmd.go:199 still uses math.Round on elapsed hours, and the exactness comment is unchanged.
  - id: BR-30
    disposition: not-addressed
    note: |
      replraw.go:67-69 unchanged; completionsFor still runs twice per keystroke, at :113 and :129.
findings:
  - id: new
    severity: Important
    family: guard-bypassed-by-new-kind
    title: |
      The needsDeck exemption strands deps.clock, so a registry row with needsDeck false gets a nil clock and panics
    detail: |
      This is the 2nd finding in family guard-bypassed-by-new-kind (BR-26 first). Do NOT patch
      this instance. main.go:285 skips withStore for a command that does not need the deck, and
      withStore (main.go:99) is the ONLY thing that ever sets deps.clock - realDeps at main.go:50
      does not. Measured: adding a registry row with needsDeck false whose run calls c.clock.Now()
      makes "define -no-audio /probe" panic with a nil pointer dereference, while the full suite
      stays green. capture_test.go:468 and :494 exist to prevent exactly this and both pass,
      because they exercise openStore and withStore, the path the exemption routes around. The
      issue's Done-when "adding a second command needs no change to the dispatch loop" now carries
      an undocumented precondition that neither the commandCtx doc comment nor atlas/define.md's
      needsDeck paragraph states.
      THE RULE - when a new kind is exempted from a shared setup path, enumerate everything that
      path guaranteed and re-supply it; the exemption drops invariants you were not thinking about,
      not just the cost you were avoiding. Rule-level fix - the clock has no store dependency, so
      supply it outside withStore (in run before the branch, or as a fallback in newCommandCtx) so
      no exemption can strand it, and extend TestEveryRegisteredCommandIsRunnable to assert the ctx
      a one-shot actually builds carries every field a row may read.
  - id: new
    severity: Important
    family: same-source-read-twice
    title: |
      /history reads the whole event log twice and prints every store warning twice
    detail: |
      Measured against a torn-log fixture - "define -no-audio /history" and "echo /history |
      define" each emit "recovered 1 event(s), dropped 1 torn record(s)" TWICE before the
      output, while a plain word lookup emits it once and "define /help" not at all. Read 1 is
      newStoreHistory calling st.Events(time.Time{}) at history_store.go:31, built by withStore
      because commandNeedsDeck said yes; read 2 is runHistory calling c.deck.Events(time.Time{})
      at history_cmd.go:228. In the one-shot path the first read is pure waste, and replLines
      never touches d.history at all so it is waste in the piped path too. atlas/define.md:446
      and plan line 374 both record "One read per /history on a personal word list is the right
      trade today" as the accepted cost, so the recorded trade is not the one the code makes.
      Fix by gating on the specific dependency rather than the whole store bundle, or by handing
      runHistory the events storeHistory already read, and assert the warning COUNT in
      TestCommandThatReadsNothingDoesNotOpenTheLog, which today only checks presence.
  - id: new
    severity: Important
    family: docs-consumer-not-updated
    title: |
      The atlas never learned about the runtime-state guards, and the README still teaches the deprecated flag name
    detail: |
      This is the 3rd finding in family docs-consumer-not-updated (BR-4, BR-27). Do NOT patch
      these instances. Measured prevalence, both from this window - (1) atlas/repo-guards.md
      documents one guard class with an index/history table naming TestNoCommittedBinaries and
      TestNoBinariesInHistory; this window added a second class with the identical split
      (TestNoTrackedRuntimeState / TestNoRuntimeStateInHistory) and neither that page nor
      atlas/index.md:13-14 ("no executable image in the index or reachable from HEAD") mentions
      it, so the index understates what the file it points at enforces. (2) README.md:84 still
      reads "Flags are session settings - define -times 1 opens the loop with single playback",
      teaching the name README.md:118 calls "the older name for --sound".
      THE RULE - a surface is not shipped until every doc that already describes its class is
      updated in the same commit, where "its class" means the page that is wrong by omission,
      not only the page that names the new thing. BR-4 updated the README because the finding
      named the README and BR-27 updated three places because the finding listed three; neither
      adopted the sweep. Enforcement worth having (ARCH-PURPOSE) - atlas/repo-guards.md's table
      is a hand-maintained restatement of the Test functions in repo_guard_test.go, so a test
      that greps the table for every "func TestNo..." in that file makes the page derive.
  - id: new
    severity: Important
    family: plan-artifact-lags-code
    title: |
      The Core-concepts table lost five entities again, and the plan has no Revisions entry for rounds 5 or 6
    detail: |
      This is the 3rd finding in family plan-artifact-lags-code (BR-5, BR-21). Do NOT just add
      the rows. commandNeedsDeck, command.needsDeck, editDistance, historyPaths and
      TestNoRuntimeStateInHistory appear in no row (grep-verified, zero occurrences of each name
      in the plan); four of the five were added by e5719f9, the commit that closed BR-21's own
      family. The plan's last Revisions heading is "close round 4", so rounds 5 and 6 - which
      produced the needsDeck design, the history rewrite and the --days docs - are unrecorded.
      Box-ticking is clean and every entity the table DOES name exists at its stated path.
      THE RULE - the Core-concepts table is a hand-maintained restatement of what the package
      declares, so it drifts by default; the fix is to make it derive, not to remember harder.
      BR-21 was disposed addressed and the very next commit re-broke it, which is the measurement
      that a habit-level fix does not hold. A guard comparing the table's Name-and-path pairs
      against top-level declarations in the named files would fail the same commit that adds an
      unnamed entity - the same shape as TestNoTrackedRuntimeState.
  - id: new
    severity: Minor
    family: parallel-construction-drift
    title: |
      Two registry lookups carry independently-written matching rules and nothing pins that they agree
    detail: |
      This is the 2nd finding in family parallel-construction-drift (BR-6 first). Do NOT patch
      this instance. commandNeedsDeck (command.go:192-199) and dispatchCommand (command.go:211-215)
      each walk cmds with strings.EqualFold. Mutating commandNeedsDeck to an exact comparison
      leaves the ENTIRE suite green while "define /HISTORY" reports "define: no deck in this
      directory" and exits 1 - a lie about the user's data - because dispatch still accepts
      loosely and the deck gate no longer does. atlas/define.md states the loose-accept policy as
      a design decision, so the two lookups implementing it must agree by construction.
      THE RULE - two sites that must agree about one fact should be one site: a single
      findCommand(name, cmds) (command, bool) that both call, so the matching rule cannot drift.
```

---

## Re-review — 2026-08-21T18:10:14-07:00 (unknown)

| field | value |
|-------|-------|
| issue | 15 — REPL command mode: /-prefixed commands with type-ahead, starting with /history |
| repo | tools |
| issue file | workshop/issues/000015-repl-commands.md |
| boundary | whole-issue close |
| milestone | — |
| window | c7d07575cb125fcb438bcb3fe45ff3763282690b..8f04434783a169d762fb781618c390f47fcf0a90 |
| command | sdlc close --issue 15 |
| reviewer | claude |
| timestamp | 2026-08-21T18:10:14-07:00 |
| verdict | unknown |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
I'll start by getting oriented in the repo and reading the issue, plan, and current code state.
API Error: Connection lost mid-response. The response above may be incomplete.

---

## Re-review — 2026-08-21T18:24:16-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 15 — REPL command mode: /-prefixed commands with type-ahead, starting with /history |
| repo | tools |
| issue file | workshop/issues/000015-repl-commands.md |
| boundary | whole-issue close |
| milestone | — |
| window | c7d07575cb125fcb438bcb3fe45ff3763282690b..8f04434783a169d762fb781618c390f47fcf0a90 |
| command | sdlc close --issue 15 |
| reviewer | claude |
| timestamp | 2026-08-21T18:24:16-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Ignoring 6 permissions.allow entries from .claude/settings.json: this workspace has not been trusted. Run Claude Code interactively here once and accept the trust dialog, or set projects["/Users/xianxu/workspace/tools"].hasTrustDialogAccepted: true in /Users/xianxu/.claude.json.
```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 7 closes the three findings that carried real behaviour, and it closes them by deleting the design that caused them rather than patching around it — `needsDeck` is gone, `withStore` is unconditional again, and reading the log moved out of `storeHistory`'s constructor into `History.Load`, called only by the loop that recalls. I verified every claim by reverting it rather than reading the commit message: reverting `Load` to eager reddens `TestCommandThatReadsNothingDoesNotOpenTheLog`; removing `withStore`'s clock fallback reddens `TestWithStoreCarriesTheClock`; nil-ing the clock in `newCommandCtx` panics the suite; deleting `hist.Load()` from `runEditor` reddens `TestEditorPersistsThroughDeps`; and killing the torn-log fixture path makes every row of `TestUsageErrorsDoNotOpenTheLog` fail loudly instead of passing on nothing. Probed against the built binary with a torn log: `/qqqqqq`, `/qqqqqq zzz` and `/history zzz` now read nothing, and `/history` warns **once** where it warned twice. `go build`, `go vet` (both tag sets), `gofmt -l`, `go test -count=1`, `-race`, and all four pty conformance tests on a real terminal are green, with `git status` clean and zero deck objects in the index or reachable from `HEAD`. Nothing is Critical, so this does not block. What keeps it off SHIP is that the same commit which fixed `BR-33`'s documentation left `atlas/define.md` describing `needsDeck` — a field it deleted — plus two more paragraphs claiming the log is read "at construction", and `history_store.go`'s own type comment contradicts its `Load` doc fifteen lines below. The atlas is this repo's designated always-current map; it now documents a design that does not exist.

## 1. Strengths

- **Removing the cost beat routing around it, and the commit says so.** `History.Load` (`history_store.go:40`) made three findings one change: nothing needs exempting from `withStore`, so nothing can strand what `withStore` supplies. Measured downstream benefit beyond the findings — a plain one-shot word lookup no longer reads the event log either.
- **The vacuity repair is the best thing in this window.** `capture_test.go:461-465` pairs every absence assertion with a control that makes the same thing happen. I broke the fixture path and all three rows failed with *"the fixture is not live — /history did not warn either, so this row proves nothing"*. That is the correct shape for "this did not happen" tests and it is worth copying repo-wide.
- **`hist.Load()`'s production wiring is pinned by a loop-driving test**, not a hand-built context: deleting `replraw.go:67` reddens `TestEditorPersistsThroughDeps`. BR-19's rule held on the very next wiring the issue introduced, which is the evidence that a rule in `lessons.md` is doing work.
- **BR-35 is closed by construction, not by discipline.** Deleting `commandNeedsDeck` leaves exactly one `strings.EqualFold` walk over the registry (`command.go:195`), so the two matching rules can no longer disagree — the "two sites that must agree should be one site" fix, taken rather than deferred.
- **ARCH-MOCK is genuinely satisfied**: `store.Mem` and `store.FixedClock` are production files (`store/mem.go`, `store/clock.go`), five live conformance suites sit behind a build tag, and `cmd.Dir = t.TempDir()` means the pty flow and the in-process fake now share the storage boundary — re-confirmed by a clean tree after a real-pty run.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `atlas/define.md` documents `needsDeck`, a field this window's last commit deleted, and two more paragraphs describe a load that no longer happens at construction.**

> **This is the 4th finding in family `docs-consumer-not-updated`** (BR-4, BR-27, BR-33). Earlier rounds fixed named instances. Do NOT patch these instances — state the rule and fix that.

Measured prevalence, all four from commit `8f04434` — the commit whose message closes BR-33 with *"atlas/repo-guards.md now documents both runtime-state guards"*:

| location | claim | actual |
|---|---|---|
| `atlas/define.md:404-405` | "A command declares `needsDeck` when it reads the store. Opening the store constructs `storeHistory`, which READS the whole event log" | `grep needsDeck cmd/` → zero hits; opening the store reads nothing |
| `atlas/define.md:307` | "`storeHistory` … loads the log once at construction" | loads in `Load()`, `history_store.go:40` |
| `atlas/define.md:284` | "since `#4` it only reads at construction" | same |
| `atlas/index.md:13-14` | "no executable image in the index or reachable from `HEAD`" | `repo-guards.md` now enforces a second class (`TestNoTrackedRuntimeState` / `TestNoRuntimeStateInHistory`) the index does not mention — the exact instance BR-33 named |

**The rule** BR-33 stated and this round did not adopt: *a surface is not shipped until every doc that already describes its class is updated in the same commit, where "its class" is the page that is wrong by omission or contradiction, not only the page that names the new thing.* The measurement that a habit-level fix does not hold is now three rounds deep: BR-4 fixed the README because the finding named the README, BR-27 fixed three places because the finding listed three, BR-33 fixed `repo-guards.md` and the README flag name and left `index.md`, and the same commit created three fresh contradictions in a file it did not think to re-read. **The enforcement worth having** (ARCH-PURPOSE): `atlas/repo-guards.md`'s table is a hand-maintained restatement of the `func TestNo…` declarations in `repo_guard_test.go`; a test that greps the table for each one makes that page derive. There is no equivalent for `atlas/define.md`, so the cheap durable half is a rule the diff can be checked against: *when a commit deletes an identifier, grep `atlas/` and `README.md` for it before committing.*

**I-2 — `editDistance` is still absent from the Core-concepts table, and so is `runHelp`.**

> **This is the 4th finding in family `plan-artifact-lags-code`** (BR-5, BR-21, BR-34). Do NOT just add the rows.

BR-34 named five missing entities. Two (`commandNeedsDeck`, `command.needsDeck`) became correct by being *deleted*, not by being maintained. Two (`historyPaths`, `TestNoRuntimeStateInHistory`) were added. **`editDistance` (`command.go:124`) is still absent** — grep-verified, zero occurrences in the plan — and cross-checking every top-level declaration in the three new files against the tables surfaces a sixth the finding never named: **`runHelp` (`command.go:211`)**, the `/help` command itself, has no row and never has. Box-ticking is clean (zero `- [ ]` in plan and issue), the round-5/6 Revisions entry landed, and every entity the tables *do* name exists at its stated path.

**The rule**, restated because round 6's prediction came true: *the Core-concepts table is a hand-maintained restatement of what the package declares, so it drifts by default; the fix is to make it derive, not to remember harder.* A guard comparing the table's `Name | path` pairs against top-level declarations in the named files would have failed the commit that added `editDistance` and the commit that added `runHelp` — the same shape as `TestNoTrackedRuntimeState`, which this repo has already shown itself willing to write. Four rounds of remembering harder is the measurement that remembering does not work here.

## 4. Minor findings

- **`history_store.go:21-22` contradicts `history_store.go:35-38` inside one file.** *3rd in family `comment-orphaned-by-insertion`* (BR-12, BR-30) — do not patch the instance. The type doc says "the log is read once **at construction** and everything after is memory"; `Load`'s doc fifteen lines below says "It used to happen in the constructor". Same commit wrote the second and left the first. **The rule** (already stated for BR-30, now measurably unadopted at prevalence 3): *when a fix changes what a block does, every comment describing that block is part of the diff — including the type's own doc comment, which is the one nobody re-reads.*
- BR-15's key table is untouched for the fourth round: `README.md:42-49` has no `/` row and `Enter | define what you typed (never the suggestion)` still omits command dispatch.
- `define --sound 1000 /sound` prints `playing 1000×` and exits 0 while `/sound 1000` is refused at the cap (BR-22, re-probed).
- `define -times -1 x` still reports `define: -sound must not be negative` (BR-23, re-probed).
- Truncation is still bytes in `command.go:238-239` and runes in `history_cmd.go:177-178`; a third name-column width now exists in `runHelp`'s hardcoded `%-10s` (`command.go:211`) beside `menuNameWidth`'s computed 9 (BR-24, prevalence now 3).
- `renderHistory`'s truncation branch is still entered by no test — `history_cmd_test.go:225` and `:268` both pass width 0 (BR-25).
- `relativeDay` still rounds elapsed hours (`history_cmd.go:199`); re-reproduced in `Pacific/Apia`, `at=2011-12-29`, `now=2011-12-31` → `"yesterday"` for a two-calendar-day gap, exactly 24 elapsed hours (BR-29).
- `replraw.go:70` still claims candidates are resolved once per keystroke; `completionsFor` runs at `:113` and `:129` (BR-30).
- **`storeHistory.Load` (`history_store.go:40`) mutates `h.loaded` and `h.lines` without taking `h.mu`**, while `Add` and `Prefix` both take it. Unreachable concurrently today — the only caller is `runEditor` before the key loop, and `-race` is green — but it is a mutating method on a mutex-guarded struct that skips the mutex, added this round.
- `define /history zzz` reading the whole log before its usage error is untested — see test coverage below.

## 5. Test coverage notes

- Verified green: `go build ./...`, `go vet ./cmd/...`, `go vet -tags conformance ./cmd/define/`, `gofmt -l cmd/` (empty), `go test ./cmd/define/... -count=1`, `-race`, and `go test -tags conformance -run PTY -v` — **all four pty tests PASS on a real terminal**, run unsandboxed, with `git status` clean afterwards and no deck under `cmd/define/`.
- Verified pinned by reverting the fix (each mutation compiled, full suite `-count=1`): `History.Load` laziness → `TestCommandThatReadsNothingDoesNotOpenTheLog` fails naming both commands; `withStore`'s clock fallback → `TestWithStoreCarriesTheClock` fails; `newCommandCtx` clock → nil-pointer panic through `run()`; `hist.Load()` in `runEditor` → `TestEditorPersistsThroughDeps` fails; the torn-log fixture path → all three `TestUsageErrorsDoNotOpenTheLog` rows fail on the control.
- Verified **unpinned**, two mutations, full suite green each time: (1) inserting a second `c.deck.Events(time.Time{})` into `runHistory` — the exact BR-32 symptom, and the count assertion that finding asked for was not added; (2) moving the `Events` read *above* `parseHistoryArgs` in `runHistory` — the exact BR-26 symptom for `/history zzz`, and the three rows that finding asked for were not added. **This is the 2nd finding in family `uncovered-branch-at-boundary`** (BR-25 first). Do not add these two rows and stop. **The rule:** *when a finding states the inputs it measured, those inputs are the regression fixture — the fix is closed by a test that runs them, not by a fix that happens to make them pass.* BR-25 is the same shape: `renderHistory`'s width branch is prose-correct and fixture-free.
- `TestNoTrackedRuntimeState` and `TestNoRuntimeStateInHistory` both `t.Fatal` on an empty scan, so neither can pass vacuously; `historyPaths` is genuinely shared by both history guards.

## 6. Architectural notes

- **ARCH-DRY — pass with flags.** Real consolidations this round: `commandNeedsDeck` deleted so one `EqualFold` walk remains; `historyPaths` shared by both history guards; `noDeckMessage` shared by `--forget` and `/history`; the event log now has exactly one reader per invocation. Flagged: three independent name-column width computations (`runHelp`'s `%-10s`, `menuNameWidth`, `renderHistory`'s `nameW`) and two truncation implementations (BR-24, prevalence 3); `relativeDay` vs `historyWindow` still two techniques for one calendar-day concept (BR-29).
- **ARCH-PURE — pass.** Every PURE row in the Core-concepts table runs its tests with no store, clock, terminal or network; I confirmed none needs a mock, and the doubles that appear (`memHistory`, `store.Mem`, `store.FixedClock`) are production code behind the same interfaces the YAML store implements. `runHistory`/`runSound`/`dispatchCommand` take writers and are the thin shell, and `commandCtx` narrower than `deps` keeps "a command cannot reach the dictionary or the player" structural rather than conventional.
- **ARCH-PURPOSE — flag.** Entry-mode sweep passes: `define /history 7`, `echo '/history 7' | define` and the raw prompt all hand `parseREPLLine` a whole line. Registry sweep passes: `commands` is the source and `/help`, `menuLines`, `completionsFor`, `nearestCommands` all derive from it. The failing sweep is the reverse direction — `atlas/define.md` and `atlas/index.md` are hand-maintained restatements that now *contradict* the source rather than merely lag it (I-1), and the plan's Core-concepts table is a restatement that has drifted after repair three times (I-2). Both are the ARCH-PURPOSE point precisely: a restatement that does not derive is a deferred consumer.
- **ARCH-MOCK — pass.** No new external dependency. Production flow and conformance flow share the storage boundary (`cmd.Dir` to a temp dir, re-verified by a clean tree after a real-pty run), five live conformance suites exist behind `//go:build conformance`, and `refusingDict` asserts a negative interaction rather than an output shape.

## 7. Plan revision recommendations

The plan needs one more `## Revisions` entry (the last is "close rounds 5 and 6"), covering:

- **Record that `needsDeck` was designed, shipped, and deleted within one issue**, and why: the exemption it created dropped `deps.clock` and did not even achieve its own goal. That is the load-bearing design history for anyone who later wonders why commands do not declare their dependencies — and it is what makes `atlas/define.md:404`'s survival legible as an omission rather than a disagreement.
- **Add the two missing Core-concepts rows** — `editDistance` (PURE, `cmd/define/command.go`) and `runHelp` (INTEGRATION, `cmd/define/command.go`) — *and* record that the table has now been repaired and re-broken across four rounds, which is the argument for the derive-guard over another repair (I-2).
- **Correct the Task 2 Step 3 signature**, open since the M1 review: the plan specifies `completionsFor(line string, hist History) []string`; the shipped function is `completionsFor(base string, hist History, cmds []command) []string`.
- **Restate the `Events` cost note.** Chunk 1's "Known cost: `Events` is O(all history) per call. It is one read per `/history` invocation" is now true — it was not when written — but the *mechanism* changed underneath it (laziness in `History.Load`, not eager construction), and the paragraph still reads as if the store were opened-and-read. One sentence naming `Load` keeps the plan's account matching the code's.

```findings
dispose:
  - id: BR-15
    disposition: not-addressed
    note: |
      README.md:42-49 still has no / row and "Enter | define what you typed (never the suggestion)" still omits command dispatch.
  - id: BR-22
    disposition: not-addressed
    note: |
      Re-probed against the built binary — "define --sound 1000 /sound" prints "playing 1000x" and exits 0, while "/sound 1000" is refused at 20.
  - id: BR-23
    disposition: not-addressed
    note: |
      Re-probed — "define -times -1 x" still reports "define: -sound must not be negative".
  - id: BR-24
    disposition: not-addressed
    note: |
      command.go:238-239 still slices bytes, history_cmd.go:177-178 still slices runes, and runHelp's hardcoded %-10s at command.go:211 is now a third name-column width beside menuNameWidth.
  - id: BR-25
    disposition: not-addressed
    note: |
      history_cmd_test.go:225 and :268 still pass width 0, and nothing asserts runHistory forwards c.width.
  - id: BR-26
    disposition: addressed
    note: |
      Probed with the torn-log fixture — /qqqqqq, /qqqqqq zzz and /history zzz all read nothing now; mutation-verified (reverting Load to eager reddens TestCommandThatReadsNothingDoesNotOpenTheLog). The three rows were still not added; raised separately.
  - id: BR-29
    disposition: not-addressed
    note: |
      Reproduced in Pacific/Apia — at=2011-12-29, now=2011-12-31 prints "yesterday" for a two-calendar-day gap. history_cmd.go:199 unchanged, exactness comment unchanged.
  - id: BR-30
    disposition: not-addressed
    note: |
      replraw.go:70 unchanged; completionsFor still runs twice per keystroke, at :113 and :129.
  - id: BR-31
    disposition: addressed
    note: |
      needsDeck deleted and withStore is unconditional; mutation-verified two ways — removing withStore's clock fallback reddens TestWithStoreCarriesTheClock, and nil-ing clock in newCommandCtx panics the suite through run().
  - id: BR-32
    disposition: addressed
    note: |
      Probed — /history warns ONCE now, one-shot and piped; pinned by the Load-laziness mutation. The warning-COUNT assertion the finding asked for is still absent; raised separately.
  - id: BR-33
    disposition: not-addressed
    note: |
      repo-guards.md gained the section and README.md:84 now teaches --sound, but atlas/index.md:13-14 — named explicitly in the finding — still mentions only the executable-image class, and the same commit created three fresh stale claims in atlas/define.md.
  - id: BR-34
    disposition: not-addressed
    note: |
      commandNeedsDeck and needsDeck became correct by deletion rather than maintenance; historyPaths and TestNoRuntimeStateInHistory landed; editDistance is still absent (grep-verified), and the derive-guard the rule called for was not written.
  - id: BR-35
    disposition: addressed
    note: |
      commandNeedsDeck is deleted, so exactly one strings.EqualFold walk over cmds remains (command.go:195) — the two rules cannot drift by construction.
findings:
  - id: new
    severity: Important
    family: docs-consumer-not-updated
    title: |
      atlas/define.md documents needsDeck, a field the same commit deleted, and two more paragraphs describe a load that no longer happens at construction
    detail: |
      This is the 4th finding in family docs-consumer-not-updated (BR-4, BR-27, BR-33). Do NOT
      patch these instances. Measured prevalence, all four from 8f04434 - the commit whose
      message closes BR-33: (1) atlas/define.md:404-405 "A command declares needsDeck when it
      reads the store. Opening the store constructs storeHistory, which READS the whole event
      log" - grep needsDeck over cmd/ returns zero hits and opening the store reads nothing;
      (2) atlas/define.md:307 "loads the log once at construction"; (3) atlas/define.md:284
      "since #4 it only reads at construction" - both falsified by History.Load; (4)
      atlas/index.md:13-14 still names only the executable-image class, the exact instance
      BR-33 listed and the one half of it that was not fixed.
      THE RULE, stated by BR-33 and still unadopted - a surface is not shipped until every doc
      that already describes its class is updated in the same commit, where "its class" means
      the page that is wrong by omission or contradiction, not only the page that names the new
      thing. Three rounds of fixing exactly the pages a finding lists is the measurement that
      the sweep is not happening. Cheap durable half - when a commit DELETES an identifier,
      grep atlas/ and README.md for it before committing; enforcement worth having
      (ARCH-PURPOSE) is the repo-guards table guard BR-33 already described.
  - id: new
    severity: Important
    family: plan-artifact-lags-code
    title: |
      editDistance is still missing from the Core-concepts table, and runHelp was never in it
    detail: |
      This is the 4th finding in family plan-artifact-lags-code (BR-5, BR-21, BR-34). Do NOT
      just add the rows. Of BR-34's five named entities, two (commandNeedsDeck,
      command.needsDeck) became correct by DELETION rather than maintenance, two (historyPaths,
      TestNoRuntimeStateInHistory) were added, and editDistance at command.go:124 is still
      absent - grep-verified, zero occurrences in the plan. Cross-checking every top-level
      declaration in command.go, history_cmd.go and sound_cmd.go against both tables surfaces
      a sixth the finding never named - runHelp at command.go:211, the /help command itself,
      which has no row and never had one. Box-ticking is clean, the round-5/6 Revisions entry
      landed, and every entity the tables DO name exists at its stated path.
      THE RULE, unchanged from BR-34 and now four rounds deep - the table is a hand-maintained
      restatement of what the package declares, so it drifts by default; the fix is to make it
      derive, not to remember harder. A guard comparing the table's Name-and-path pairs against
      top-level declarations in the named files would have failed the commit that added
      editDistance and the commit that added runHelp - the same shape as TestNoTrackedRuntimeState.
  - id: new
    severity: Minor
    family: comment-orphaned-by-insertion
    title: |
      history_store.go's type comment says the log is read at construction; its own Load doc fifteen lines below says it moved out of the constructor
    detail: |
      This is the 3rd finding in family comment-orphaned-by-insertion (BR-12, BR-30). Do NOT
      patch this instance. history_store.go:21-22 reads "the log is read once at construction
      and everything after is memory", while history_store.go:35-38 - written by the same
      commit - reads "It used to happen in the constructor". A direct contradiction inside one
      file, in the doc comment of the type the change is about.
      THE RULE, already stated for BR-30 and measurably unadopted at prevalence 3 - when a fix
      changes what a block does, every comment describing that block is part of the diff,
      including the TYPE's own doc comment, which is the one nobody re-reads because it sits
      above the declaration rather than above the changed lines. The same two sentences are
      restated at atlas/define.md:284 and :307, which is why this and the docs finding are one
      change to make.
  - id: new
    severity: Minor
    family: uncovered-branch-at-boundary
    title: |
      The specific inputs BR-26 and BR-32 measured are still entered by no fixture; both regressions restore silently
    detail: |
      This is the 2nd finding in family uncovered-branch-at-boundary (BR-25 first). Do NOT just
      add these rows. Two mutations, each compiled with the full suite green at -count=1:
      (1) inserting a second c.deck.Events(time.Time{}) into runHistory restores BR-32's exact
      doubled-warning symptom - the count assertion that finding asked for was never added;
      (2) moving the Events read above parseHistoryArgs in runHistory restores BR-26's exact
      "define /history zzz reads the whole log before its usage error" symptom - the three rows
      that finding asked for were never added. The behaviour is correct in both cases and the
      shared mechanism (History.Load laziness) IS pinned, so this is about the specific cases,
      not the fix.
      THE RULE - when a finding states the inputs it measured, those inputs ARE the regression
      fixture; a fix is closed by a test that runs them, not by a change that happens to make
      them pass. BR-25 is the same shape at a different site - renderHistory's width branch is
      prose-correct and fixture-free because both call sites pass 0.
  - id: new
    severity: Minor
    family: partial-lock-discipline
    title: |
      storeHistory.Load mutates lines and loaded without taking the mutex that Add and Prefix both take
    detail: |
      history_store.go:40. Load appends to h.lines and sets h.loaded outside h.mu, while
      Add (:62) and Prefix (:73) both lock. Unreachable concurrently today - the only production
      caller is runEditor at replraw.go:67, before the key loop starts, and go test -race is
      green - so this is latent, not live. But a struct that locks for some mutations and not
      others invites the next caller to assume the wrong thing, and Load is the newest method
      on it.
```

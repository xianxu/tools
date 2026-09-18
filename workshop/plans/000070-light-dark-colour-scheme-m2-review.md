# Boundary Review — tools#70 (milestone M2)

| field | value |
|-------|-------|
| issue | 70 — define: switch between a light and a dark colour scheme |
| repo | tools |
| issue file | workshop/issues/000070-light-dark-colour-scheme.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 4ae29a7278cd00310097a8ce66e253e8f0fde249..b95a983156d1082c5bce23c22009daf98e753ad3 |
| command | sdlc milestone-close --issue 70 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-09-17T23:46:34-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**VERDICT: FIX-THEN-SHIP**

M2 delivers what it claims. There is a store for the saved scheme with a size cap and a closed parse. `deps.configDir` is its own seam and accepts absolute paths only. `applyScheme` saves before it switches. `describeScheme` gives every report the spec asks for. `/scheme` works in the editor, the piped loop and the one-shot form, and the pty harness now gives every binary launch its own config directory. The docs are updated in the same window. I checked the implementor's mutation claims myself: every mutation I applied turned its test red. One Important finding needs fixing before the boundary. The issue `## Log` has no M2 entry, yet several ticked plan steps say their evidence is recorded there, and one of those claims doesn't match what M1 already logged. The three Minor findings are architectural and don't block.

**How I inspected it:** I ran the stat, name-status and full-diff commands. I ran the non-pty suite (green), `-race` on the new tests (green), `go vet` including the conformance tag (clean) and `GOOS=linux go build ./...` (clean). I applied 8 mutations in a scratch worktree and all 8 turned red. I could not run the pty-backed tests: this environment refuses pty allocation outright (Python's `os.openpty()` also fails with EPERM). So `TestSavedSchemeGovernsALookup` and the `TestPTY*` changes were reviewed by reading only.

### 1. Strengths
- **Save first, then switch, with both failure paths pinned.** `cmd/define/scheme.go` `applyScheme`: moving `h.choose` so it no longer runs after a save turned the editor test, the piped test and `TestApplyScheme` red. A failed save and a failed clear each have a row showing the holder is unchanged.
- **Each loop's settings are pinned by a test that drives that loop.** Removing the editor's `cc.session` or `cc.fullScreen`, or the piped loop's `cc.session`, each broke exactly the test named for it. Removing the startup read broke `TestOneShotScheme`'s bare `/scheme` without needing a pty.
- **`ClearScheme` removes only files it owns** (`store/scheme.go`). Removing the `Lstat`/`IsDir` guard made the symlinked-config-dir test fail. The cap and the directory removal are pinned the same way.
- **Harness isolation covers every launcher.** Every binary launch goes through `startDefineBinary` or the layout test, and both now get an isolated config directory. `--version` returns before the holder is built. The clipboard tests launch the test binary, not `define`. `testDeps` leaves `configDir` nil, so no in-process test can reach a real config.
- **The recolour needs no code in the command.** The editor's usual `draw()` after dispatch repaints what is on screen, and `TestRawEditorSchemeRepaintsWhatIsOnScreen` pins both the frame and the exit transcript. `runLookup` also removed some duplicated pty setup.

### 2. Critical findings
None.

### 3. Important findings
- **Plan steps are ticked but the Log has no M2 entry** (`workshop/issues/000070-light-dark-colour-scheme.md` `## Log`; plan Task 7 Step 6, Task 11 Steps 4 and 6).
  - Task 7 Step 6 says `WriteScheme`'s atomic rename turns a symlinked `scheme` file into a regular file, and that this is "noted in the Log rather than changed here". The Log has no such note. With GNU stow, a real `~/.config/define/` holding a linked `scheme` file is plausible: `/scheme light` would silently replace the link with a regular file, and `/scheme auto` would delete the link.
  - Task 11 Step 4 claims `go test -tags conformance -run 'TestPTY'` PASSED and asks to "name what ran vs skipped". But the M1 Log records 13 `TestPTY*` tests that were already failing in this environment. A clean PASS contradicts that, and nothing records what actually ran.
  - The Done-when requires every new test to have been seen failing without its fix. There is no M2 record of this, and Task 11 Step 6 (the harness mutation) can't be re-checked without a pty.
  - **Fix:** add an M2 Log entry that covers the symlinked-file behaviour, the pty run broken into passed / pre-existing failures / skipped, and the list of mutations.

### 4. Minor findings
- **Two pairs of fields allow combinations that mean nothing** (ARCH-ORDER). **This is the 2nd finding in family `state-shape-admits-illegal-combinations`.**
  - `commandCtx.session` and `commandCtx.fullScreen` (`command.go`) allow `fullScreen && !session`, which has no meaning.
  - `schemeArg{auto, value}` allows `{auto: true, value: light}`, and its zero value means "choose the empty scheme". If a zero `schemeArg` ever reached `applyScheme`, it would save `"\n"`, and every later startup would warn about a garbled file.
  - Neither combination is reachable today.
  - **Rule:** when fields depend on each other, they should be one tagged value whose members are exactly the legal combinations. Here that means a `loopKind` of `{oneShot, piped, editor}`, and a choice that is either nil (auto) or a scheme.
- **The startup read skips the persister seam** (ARCH-DRY, ARCH-PURE; `main.go:870-883` compared with `deps.schemePersister()`).
  - `d.configDir()` is resolved in two places, and `schemePersister` has save and clear but no load.
  - The flag → saved → default order lives inline in `run()`. "Flag beats saved" and "garbled file warns once" are only tested through a real pty.
  - **Fix:** add `load()` to `schemePersister`, and make the precedence a pure `initialSchemeState(flag, persister, warn)` that can be unit-tested without a pty.
- `schemeUsage`, the README and the atlas all say light or dark "saves it for every session". That isn't true when no config directory exists. The report itself is accurate.

### 5. Test coverage notes
- The in-process wiring is strongly pinned, and mutation checks confirm it.
- The pty-only pieces are the harness line, the `default` subtest, `TestPTYSavedSchemeSurvivesARestart`, and the two precedence cases in `TestSavedSchemeGovernsALookup`. They read correctly, but I couldn't run them here.
- `runLookup` calls `t.Fatal` when a pty can't be opened, where it could skip. This behaviour came over from `TestLanguageTintInvocation`, but it means a pty-less CI reports a failure rather than a skip.

### 6. Architecture notes
- **ARCH-DRY:** flagged (Minor above). `runLookup` is a good consolidation. `applyScheme` rewrites `/bilingual`'s save rule rather than sharing it with `sessionSetBilingual`. That's acceptable today, but two copies of one policy can drift.
- **ARCH-PURE:** flagged (merged into the Minor above). `configDirFrom` and `describeScheme` are pure, and `applyScheme` has its holder and persister injected.
- **ARCH-PURPOSE:** passes. I checked every consumer of the saved choice: the startup read, all three command shells, sittings (which run no commands but share the holder), `--play` (which reads the holder built at startup), every binary launcher, and all the docs (README, `-h`, atlas command list/usage, atlas store section).
- **ARCH-MOCK:** passes. The filesystem is backed by temp folders, `fakePersister` keeps state, and the tests of each loop go through the same `configDir` seam as production.
- **ARCH-CONSTRAINTS:** passes. Startup adds one open of a file of at most 65 bytes, and none on the `--version`/`--llm-check` paths.
- **ARCH-SECURE:** passes. The file is capped and parsed into a closed set, bases must be absolute, and no test can write real config. One note: a FIFO placed at `scheme` would block `os.Open` at startup. That needs a deliberately odd setup, so I'm noting it, not raising it.
- **ARCH-ORDER:** flagged (Minor above). Transitions only go through the holder, and the save succeeds before the switch. Because rename is the last step, a failed save really does change nothing on disk. A cross-process note: after session B runs `/scheme auto`, session A still reports `light (saved)`. That report is about an earlier event, which is fine, but M3's wording shouldn't claim more.
- **ARCH-FUNERAL:** passes. At most one directory and one file are left behind, and `/scheme auto` removes them. A save that fails after `MkdirAll` can leave an empty `define/` directory, which is trivial.

### 7. Plan revision recommendations
- If the state-shape fix is taken, add a `## Revisions` entry: "`schemeArg` → nil-able choice; `commandCtx.session`/`fullScreen` → `loopKind`". Chunk 2's code blocks then no longer match the code.
- No other revisions needed. The existing Revisions entry already says Chunk 1's code shows the old names.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      The fix belongs in future plans, and no lessons.md rule records it; Chunk 1's code blocks have already drifted from the code (the plan's own Revisions says so), which shows the cost. Minor, not blocking.
findings:
  - id: new
    severity: Important
    family: ticked-step-lacks-its-evidence
    title: |
      M2 plan steps are ticked but the issue Log has no M2 entry
    detail: |
      Task 7 Step 6 says the WriteScheme symlink-to-regular-file behaviour is "noted in the Log"; it is not. Task 11 Step 4 claims the TestPTY conformance run PASSED and should name what ran and what was skipped, but the M1 Log records 13 TestPTY tests already failing, so a clean PASS contradicts it. The mutation record the Done-when requires is also missing. Add an M2 Log entry covering the symlinked-file behaviour, the pty run broken into passed, pre-existing failures and skipped, and the mutation list.
  - id: new
    severity: Minor
    family: state-shape-admits-illegal-combinations
    title: |
      commandCtx session/fullScreen and schemeArg auto/value allow combinations that mean nothing
    detail: |
      This is the 2nd finding in this family. Rule: when fields depend on each other, they should be ONE tagged value whose members are exactly the legal combinations. Remaining instances in #70: commandCtx.session plus fullScreen (fullScreen without session is representable), and schemeArg auto plus value (its zero value would save a blank line, which every later startup warns about). Neither is reachable today. Measured prevalence in #70: 3 instances; choice/chosenBy was fixed at M1. Fix: a loopKind enum {oneShot, piped, editor}, and a nil-able choice where nil means auto.
  - id: new
    severity: Minor
    family: store-access-bypasses-its-seam
    title: |
      The startup read skips schemePersister, and the precedence order is tested only through a pty
    detail: |
      run() resolves d.configDir and calls store.ReadScheme inline, while deps.schemePersister resolves it again for save and clear (ARCH-DRY). The flag, then saved, then default order lives in run() glue; "flag beats saved" and "garbled file warns once" are pinned only by TestSavedSchemeGovernsALookup through a real pty (ARCH-PURE). Fix: add load() to schemePersister and extract a pure initialSchemeState(flag, persister, warn) that can be unit-tested without a pty.
```

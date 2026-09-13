# Boundary Review — tools#53 (whole-issue close)

| field | value |
|-------|-------|
| issue | 53 — define: /help history prints the whole list, so no command explains its arguments |
| repo | tools |
| issue file | workshop/issues/000053-help-for-one-command.md |
| boundary | whole-issue close |
| milestone | — |
| window | ebfab6b4dc373f8da2e3746f5cfad6d1b8668666..9d858ba82e06d9fecde187ec490a424f14a69ac6 |
| command | sdlc close --issue 53 |
| reviewer | claude |
| timestamp | 2026-09-12T17:01:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The boundary delivers the issue's purpose end to end. Every registry row carries `args` and `usage`, the resolver, unknown-name wording and usage renderer are each single-sourced, `--help`/`-h` are answered once in `dispatchCommand`, and both docs quote a generated span pinned by `TestDocsQuoteTheCommandUsage`. I rebuilt the binary and ran seventeen invocations in an empty directory: `/help history`, `/help /history`, `/help HISTORY`, `/history --help`, `/history -h`, `/help --help`, `/help /`, `/help histry`, `/help nosuch`, `/help history sound`, `/help //history` and bare `/help` all print exactly what the Spec and Log claim, with the right exit codes. `go test ./cmd/define/` is green (110s), `go vet` and `gofmt -l` are clean, and the retired-symbol sweep finds `pronCommandHelp` only in the tracker artifacts and the `retiredSymbolNames` map. The one Important finding is a test gap, not a shipped bug: nothing pins that `runHelp` and `dispatchCommand` forward `cc.width` to the renderer, so a `commandUsage(cmd, 0)` slip would pass the whole suite. It is a one-assertion fix.

## 1. Strengths

- `cmd/define/command.go:269-284` — dispatch now branches on outcome only (found / not found / asks for usage / run); the "never grows a case per command" Done-when from the original registry design still holds.
- `cmd/define/command.go:311-322` — `unknownCommand` is moved verbatim, and `TestHelpForAnUnknownNameSaysWhatDispatchSays` compares bytes against dispatch rather than a restated message, with a `Len() == 0` guard so empty-equals-empty cannot pass. That is the right oracle (ARCH-DRY).
- `cmd/define/history_cmd.go:25` and `sound_cmd.go:16` — limits in the usage text come from the parsers' own constants via `Sprintf`, so the docs span (`N is at most 3650`, `20 is the most`) cannot drift from what the parser enforces.
- `cmd/define/command_test.go:329-353` — `TestCommandUsageWraps` runs the renderer over every row at three widths and checks word preservation, trailing-space on empty `args`, and per-line width; a real property test over the registry rather than one example.
- The Log's mutation table is credible: the mutations named map 1:1 to the guards I read, and my own reading of each test confirms each would go red under its mutation.

## 2. Critical findings

None.

## 3. Important findings

- **`cmd/define/command.go:336` and `:283` — width forwarding from the IO shell to the pure renderer is untested.** `TestHelpExplainsOneCommand` and `TestDashHelpPrintsTheUsageForEveryCommand` both build a `commandCtx` with zero width and compare to `commandUsage(c, 0)`, so replacing `cc.width`/`c.width` with `0` at either call site passes the suite. On a narrow terminal the usage line would overflow instead of wrap. Fix sketch: in one of the two tests, add a case with `width: 20` and assert `out.String() == commandUsage(hist, 20)` (the two must differ from the width-0 rendering for the long `/pron` or `/lang` row). Family `io-shell-forwards-context`. (ARCH-PURE: the pure/glue split is right; the seam just needs one pin.)

## 4. Minor findings

- `-h` is accepted (`asksForUsage`) but only the atlas names it; `helpUsage` and the bare-help line say "--help after any command". A user reading `/help` or the README will not learn `-h` exists. Either mention `-h` in `helpUsage` (the span propagates it) or drop `-h` from the contract.
- The issue's six `## Done when` boxes are still `[ ]` while every `## Plan` box is `[x]`; tick them at close so the tracker matches the verified state.
- `TestHelpExplainsOneCommand` and `TestOneShotHelpExplainsOneCommand` both hard-code `/history [N | --days N | --days=N]`. Deliberate pins, fine; noting only so a future `args` change knows to touch two files.

## 5. Test coverage notes

- Unit: renderer, resolver, unknown-name wording, arity error, bare-help line, usage-required-per-row. All pure, no IO (ARCH-PURE pass).
- Integration: `--help`/`-h` over the whole registry through `dispatchCommand`; fixture test proves `--help` does not run the command; one-shot path via `run()` with `refusingDict` and a temp-dir store (ARCH-MOCK pass: the store is the owned component on a portable folder, the dictionary is a fake behind the seam).
- Docs: `TestDocsQuoteTheCommandUsage` over `derivedDocs` (both pages); `TestDocsQuoteTheCommandList` updated for the new `/help` summary.
- Gap: width forwarding (Important above). The TUI raw loop shares `dispatchCommand` and the screen writer normalizes newlines (`screen.go:98`), so no raw-mode-specific test is needed for this change.

## 6. Architectural notes

- **ARCH-DRY** pass: one resolver, one unknown-name wording, one synopsis builder, one usage per row, one span generator.
- **ARCH-PURE** pass with the coverage caveat above.
- **ARCH-PURPOSE** pass: shadow-sweep of README and atlas outside the span finds no hand-restated argument forms; the atlas paragraph and `pronCommandHelp` span are gone; root README's "`/help` lists the rest" remains true. Argument completion was declared out of scope in the Spec and Revisions, and it is genuinely separable.
- **ARCH-MOCK** pass: nothing new leaves the process.
- **ARCH-CONSTRAINTS** pass: O(rows) string work, `wrapText` returns early at width 0.
- **ARCH-SECURE** pass: the only untrusted input is the typed name, echoed via `%s`/`%q` exactly as dispatch already did.
- **ARCH-ORDER** pass: no state carried between events; the plan states why rather than a bare N/A.
- The `asksForUsage` contract ("no command may take `--help`/`-h` as data") is stated in code, atlas and plan. Any future command taking free-text arguments will need to revisit it; worth a line in `atlas/define.md`'s command-adding recipe if one exists.

## 7. Plan revision recommendations

None. The Core concepts table matches the code row for row (all `new` entities exist at the stated paths, `pronCommandHelp` is deleted, the three `modified` entities are touched), and the Revisions section already records the README-carry and the two-field decision.

```findings
findings:
  - id: new
    severity: Important
    family: io-shell-forwards-context
    title: |
      Nothing pins that runHelp and dispatchCommand pass cc.width to commandUsage; a hard-coded 0 passes every test
    detail: |
      Both help tests build commandCtx with width 0 and compare to commandUsage(c, 0). Add one case with width: 20 asserting out == commandUsage(hist, 20) so the wrap actually reaches the screen.
  - id: new
    severity: Minor
    family: docs-name-every-accepted-form
    title: |
      -h is accepted by asksForUsage but only the atlas mentions it; helpUsage and the bare-help line say --help only
    detail: |
      Either add -h to helpUsage (the span propagates it to both docs) or drop -h from the contract.
  - id: new
    severity: Minor
    family: issue-done-when-unticked
    title: |
      The issue's Done-when boxes are still unticked while every Plan box is ticked
    detail: |
      Tick the six Done-when items at close so the tracker reflects the verified state.
```

---

## Re-review — 2026-09-12T17:11:26-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 53 — define: /help history prints the whole list, so no command explains its arguments |
| repo | tools |
| issue file | workshop/issues/000053-help-for-one-command.md |
| boundary | whole-issue close |
| milestone | — |
| window | ebfab6b4dc373f8da2e3746f5cfad6d1b8668666..941bc789550965a836314127982a1263388f55a6 |
| command | sdlc close --issue 53 |
| reviewer | claude |
| timestamp | 2026-09-12T17:11:26-07:00 |
| verdict | SHIP |

## Review

All checks are done: mutations verified, binary exercised, docs and callers inspected. Writing up the verdict now.

```verdict
verdict: SHIP
confidence: high
```

The round-2 commit (`b6dd99c`) closes every open finding with evidence I could reproduce, and nothing new blocks. `go build ./...`, `go vet ./...`, `gofmt -l cmd/define` and `go test ./cmd/define/` are all clean on the pinned head. I reverted BR-1's fix in a scratch worktree twice (once per call site) and `TestHelpRendersWithTheContextItIsGiven` went red both times with the unwrapped `/pron` text; dropping `-h` from `asksForUsage` reddens `TestDashHelpPrintsTheUsageForEveryCommand` on the first `-h` pair; the unmutated control is green. The built binary in an empty directory prints the usage for `/help history`, `/history --help`, `/history -h`, `/help --help` and `/sound -h` (exit 0), the list for bare `/help` and `/help /`, dispatch's own wording for `/help histry` and `/help nosuch` (exit 2), and the arity error for `/help a b` (exit 2). The Core concepts table matches the tree row for row, and both derived pages quote the regenerated span that now names `-h`. One Minor, non-blocking note below, which is a repeat family and is written as the rule rather than the instance.

## 1. Strengths

- `cmd/define/command_test.go:361-392` — the new width test asserts a vacuity guard first (`commandUsage(pron, 20) != commandUsage(pron, 0)`) before comparing, so it cannot pass by both sides being the unwrapped string. It pins three forwarded values (width at `runHelp`, width at `dispatchCommand`, the registry `/help` resolves against), each with a value that changes the answer. Confirmed by revert.
- `cmd/define/command.go:269-284` — dispatch branches on outcome only (found / unknown / asks-for-usage / run). Adding a command is still one registry row; the loop did not grow a case.
- `cmd/define/command.go:311-322` and `TestHelpForAnUnknownNameSaysWhatDispatchSays` — one wording for `/histry` and `/help histry`, pinned byte-for-byte against dispatch with a `Len() == 0` guard (ARCH-DRY).
- `history_cmd.go:25`, `sound_cmd.go:16` — limits in the usage text are `Sprintf`'d from the parser's constants, so `N is at most 3650` and `20 is the most` cannot drift from what is enforced.
- `workshop/lessons.md` gains the BR-1 rule in general form ("a shell that forwards context is pinned with a value that changes the answer"), which is the §4 loop working as intended.

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

- **2nd finding in family `docs-name-every-accepted-form`.** BR-2 was fixed as the instance: `-h` was typed by hand into `helpUsage` (`command.go:56`) and the bare-help line (`command.go:355`). The rule that covers the class: *the set of forms a parser accepts is one list, and every surface that names them is derived from or checked against it.* Today the pair `--help`/`-h` is restated at five sites: `asksForUsage` (`command.go:293`), `helpUsage`, the bare-help line, the loop literal in `TestDashHelpPrintsTheUsageForEveryCommand` (`commandloop_test.go:491`), and atlas prose (`atlas/define.md:1144`). No test fails if `asksForUsage` gains or loses a token while the prose stays put. Cheapest fix of the rule: a `usageFlags` slice drives `asksForUsage` and the test loop, and one assertion checks `helpUsage` names each entry. Non-blocking.

## 5. Test coverage notes

- Unit (no IO): renderer over every row at three widths, resolver, unknown-name parity, arity error, bare-help line, usage-required-per-row, and now width/registry forwarding. ARCH-PURE holds; `command_test.go` uses only `bytes.Buffer`.
- Integration: `--help`/`-h` over the whole registry through `dispatchCommand`; fixture proves `--help` does not run the command; one-shot path via `run()` with the refusing dictionary and a temp-dir store.
- Docs: `TestDocsQuoteTheCommandUsage` over both derived pages; `TestDocsQuoteTheCommandList` updated for the new `/help` summary. The three README anchors and eight restored doc-sync tests from Task 0 are green.
- Verified by revert this round: M8a, M8b, and the `-h` routing.

## 6. Architectural notes

- **ARCH-DRY** pass, with the Minor above as the one remaining hand-kept restatement.
- **ARCH-PURE** pass: `commandUsage`, `findCommand`, `asksForUsage`, `synopsis` are pure; `runHelp`/`dispatchCommand` choose and write.
- **ARCH-PURPOSE** pass: shadow-sweep of README, atlas and root README finds no hand-written argument forms outside the span; the `pronCommandHelp` span and the atlas paragraph are gone; root README's "`/help` lists the rest" stays true. Argument completion was declared out of scope in Spec and Revisions and is separable.
- **ARCH-MOCK** pass: nothing new leaves the process; the one-shot test boots on a temp-dir store.
- **ARCH-CONSTRAINTS** pass: O(rows) string work; `wrapText` returns early at width 0.
- **ARCH-SECURE** pass: the typed name is echoed with `%s`/`%q` as dispatch already did; no persisted input, no credentials.
- **ARCH-ORDER** pass: no state carried between events, and the plan says why rather than a bare N/A.
- For upcoming work: `asksForUsage` is a contract that no command takes `--help`/`-h` as data, and it only fires on exactly one argument. A future command with free-text arguments will need to revisit it; the atlas states the contract, which is enough for now.

## 7. Plan revision recommendations

None. The table's `new`/`modified`/`deleted` rows all match the tree, and the Revisions section already records the README carry and the two-field decision.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      TestHelpRendersWithTheContextItIsGiven goes red when either call site renders at width 0 (verified by revert in a scratch worktree); control green.
  - id: BR-2
    disposition: addressed
    note: |
      helpUsage and the bare-help line say "--help or -h"; the regenerated span carries it to README and atlas, and the -h routing is pinned by TestDashHelpPrintsTheUsageForEveryCommand.
  - id: BR-3
    disposition: addressed
    note: |
      All six Done-when boxes are ticked in the issue file at head.
findings:
  - id: new
    severity: Minor
    family: docs-name-every-accepted-form
    title: |
      The --help/-h pair is restated by hand at five sites; nothing checks helpUsage names every form asksForUsage accepts
    detail: |
      Second finding in this family, so the rule rather than the instance: the accepted forms are one list, and every surface naming them derives from or is checked against it. A usageFlags slice driving asksForUsage and the test loop, plus one Contains assertion over helpUsage, closes the class. Non-blocking.
```

---

## Re-review — 2026-09-12T17:23:03-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 53 — define: /help history prints the whole list, so no command explains its arguments |
| repo | tools |
| issue file | workshop/issues/000053-help-for-one-command.md |
| boundary | whole-issue close |
| milestone | — |
| window | ebfab6b4dc373f8da2e3746f5cfad6d1b8668666..44996ec82cb9af722211025f35b7fe89c13aa1c6 |
| command | sdlc close --issue 53 |
| reviewer | claude |
| timestamp | 2026-09-12T17:23:03-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The window delivers every Spec bullet and every Done-when box, and I verified each against the code rather than the Log: `go build`, `go vet`, `gofmt -l` and the whole `cmd/define` suite are green at head; the working tree is clean at the pinned commit. The BR-4 fix is real, not cosmetic: in a scratch worktree, three mutations against `usageFlags` (the `/help` usage naming only `--help`, the bare-help line naming only `--help`, the parser accepting only the first flag) each turn `TestEveryUsageFlagIsNamedWhereUsersRead` red, and the parser mutation also reddens the whole-registry `--help` test for `/sound`, `/lang` and `/pron`; the control run is green. As a spot-check of the earlier-disposed BR-1, dropping the width at either call site reddens `TestHelpRendersWithTheContextItIsGiven`. The docs gate passes: atlas and package README both quote the generated `command-usage` span, the hand-written argument paragraph and `pron-command-help` span are gone, the root README's "`/help` lists the rest" is still true, and a grep of both pages finds no hand-written restatement of any argument rule outside the span. Nothing blocks SHIP; the one finding is a Minor plan-hygiene gap.

**1. Strengths**

- `cmd/define/command.go:290` — `--help`/`-h` answered once in `dispatchCommand`, before `cmd.run`, so seven parsers stay ignorant of it; `TestDashHelpPrintsTheUsageForEveryCommand` sweeps the registry × `usageFlags` rather than one example, and `TestDashHelpDoesNotRunTheCommand` pins "instead of", not "before".
- `cmd/define/command.go:319` — `unknownCommand` is one wording for two routes, and the test compares `/help histry` against dispatch byte-for-byte instead of against a restated string, so a plausible second phrasing cannot pass.
- `cmd/define/history_cmd.go:25`, `sound_cmd.go:16` — usage text built from `defaultHistoryDays`, `maxHistoryDays`, `maxSoundTimes`; the doc span therefore carries "2", "3650", "20" from the parsers' constants, not from prose.
- `cmd/define/command_test.go:398` — the BR-4 test checks the joined phrase `"--help or -h"`, correctly noting `-h` is a substring of `--help`; a per-flag `Contains` would have been vacuous.
- `cmd/define/command_test.go:365` — the vacuity guard (`want == commandUsage(pron, 0)` fatals) is exactly the "value that changes the answer" discipline the lessons entry names.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**

- `workshop/plans/000053-help-for-one-command-plan.md` — `usageFlags` (new pure entity, `command.go:299`) and the two review-driven tests (`TestHelpRendersWithTheContextItIsGiven`, `TestEveryUsageFlagIsNamedWhereUsersRead`) exist only in the issue Log; the plan's Core concepts table and Task 1/2 test lists don't name them and the plan has no `## Revisions` section. Not a contradiction (nothing in the table is false), so Minor; see §7.

**5. Test coverage notes**

- The kind of bug this diff could ship is covered: dropped argument (`TestHelpExplainsOneCommand`'s disagreeing `/sound` case), missing routing (whole-registry sweep), blanked usage (`TestEveryRegisteredCommandIsRunnable`), trailing-space synopsis (`TestCommandUsageWraps`), doc drift (`TestDocsQuoteTheCommandUsage` over `derivedDocs`), dropped width (BR-1 test), hand-restated flags (BR-4 test).
- `TestOneShotHelpExplainsOneCommand` boots against a temp dir with `openStore` and a refusing dict; no live dependency (ARCH-MOCK pass).
- `retiredSymbolNames` maps both `pronCommandHelp` and the deleted `TestDocsQuoteThePronCommandHelp`, so the removed-declaration guard is satisfied rather than bypassed.

**6. Architectural notes for upcoming work**

- ARCH-DRY: pass. One list (`usageFlags`), one resolver (`findCommand`), one wording (`unknownCommand`), one synopsis builder.
- ARCH-PURE: pass. `commandUsage`, `findCommand`, `asksForUsage`, `synopsis` are pure and tested without IO; `runHelp` and `dispatchCommand` only choose and write.
- ARCH-PURPOSE: pass. Shadow-sweep of consumers: `/help <name>`, `/<name> --help|-h`, one-shot path, atlas span, README span, bare-help line, `pronCommandHelp` folded. No hand-maintained restatement of an argument rule remains in either page. Argument completion for `/help hi` was correctly left as a separable extension.
- ARCH-MOCK: pass (N/A, nothing leaves the process).
- ARCH-CONSTRAINTS: pass. O(commands) string work on a prompt submission.
- ARCH-SECURE: pass. The only untrusted input is the typed name, echoed via `%q` the way dispatch already did.
- ARCH-ORDER: pass. `/help` is a function of registry, args and width; no state carried between events.
- Design note, not a finding: `asksForUsage` is deliberately "exactly one argument", so `/sound 3 --help` reaches the parser and fails as "not a number of times". The plan states the contract and the atlas documents it, so it's a settled choice; if a future issue wants `--help` anywhere in the argument list, the change is one line in `asksForUsage` and the sweep test already covers the registry.

**7. Plan revision recommendations**

- Append a `## Revisions` section to `workshop/plans/000053-help-for-one-command-plan.md` with a 2026-09-12 entry: (a) add `usageFlags` | `cmd/define/command.go` | new to the Pure entities table and note `asksForUsage` derives from it; (b) record that close rounds 1 and 2 added `TestHelpRendersWithTheContextItIsGiven` (BR-1) and `TestEveryUsageFlagIsNamedWhereUsersRead` (BR-4) beyond Task 1/2's listed tests. Keep it to the plan; the issue Log already carries the narrative.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      usageFlags is the one list; three mutations (usage text, bare-help line, parser) each redden TestEveryUsageFlagIsNamedWhereUsersRead, control green.
findings:
  - id: new
    severity: Minor
    family: plan-records-review-driven-changes
    title: |
      The plan's Core concepts table and task test lists omit usageFlags and the two review-added tests; no Revisions section records them
    detail: |
      usageFlags (command.go), TestHelpRendersWithTheContextItIsGiven and TestEveryUsageFlagIsNamedWhereUsersRead exist only in the issue Log. Append a Revisions entry and a usageFlags row so the plan stops under-claiming what the code delivers.
```

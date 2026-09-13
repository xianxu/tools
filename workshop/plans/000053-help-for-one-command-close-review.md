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

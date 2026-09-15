# Language flag prompt implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Use superpowers-executing-plans for this small, connected change; the SDLC close gate owns its fresh-context code review.

**Goal:** Make the effective language visible before every interactive definition prompt without changing terminal geometry or submitted input.

**Architecture:** A pure language-prompt helper formats a flag or text code. Both interactive shells derive it from current session language rather than caching it. One shared display-unit reader keeps flag pairs intact in all column-based rendering consumers.

**Tech Stack:** Go, existing ANSI screen and selection code, existing stateful terminal/dictionary test seams and PTY harness. No new dependency or persistent setting.

**Status:** Proposed; awaiting operator approval before `sdlc change-code`.

## Behavior

- Default interactive prompts: `🇺🇸 › ` for en, `🇪🇸 › ` for es, `🇮🇹 › ` for it. Map fr→FR, de→DE, pt→PT, zh→CN, ja→JP, ko→KR in the same presentation table. Do not derive flags from arbitrary language tags or pronunciation locales.
- Unknown valid language codes use `[xx] › `. Empty language means the existing `store.DefaultLang`; defensively invalid tags display `[??] › ` rather than echoing controls. Mapping does not advertise dictionary availability.
- `-no-flags` explicitly requests `[es] › ` and retains normal color/raw editing. `-no-color` also chooses the code prompt through the existing plain-output path. There is no capability-detection claim: terminals that do not render a flag as two cells use the explicit fallback.
- At a true terminal width below 2 columns, use the code fallback automatically. Raw prompts read `view.Size()` at render time; line prompts read `terminalCols(stdout)`. Do not use `opt.width`: it becomes 0 below the entry-wrapping threshold. Historical submitted flags are already clipped through `clipVisible(..., s.cols)` before painting; at width 1 the entire leading flag is omitted from the viewport, while its transcript stays intact. Resizing back restores it. Live code prompts contain only one-cell display units, so painting, row budgets and selection agree even at width 1. This does not claim arbitrary user-typed wide glyphs fit a one-column terminal.
- The current language is read at every live render. A submitted `/lang es` line retains the old language under which it was entered; the next prompt uses the new effective language. A rejected or failed change keeps the old prompt. Session-only successful changes follow their actual effect.
- Flag/code appears on the live editor prompt, echoed submitted lines, and the interactive line-loop prompt. Piped input or redirected stdout remains prompt-free. Practice control prompts and response backgrounds are separate surfaces; #65 owns response styling.
- Language decoration is outside editable text: history, completion matching and dispatched text contain only user input. Render the two flag runes contiguously, with ANSI styles only outside the pair.
- Flag mode assumes each adjacent regional-indicator pair occupies two columns. Clipping, wrapping, region offsets, selection and copying treat it as one unit. Lone indicators retain existing one-cell behavior; no general emoji-width-policy change.

## Core concepts

| Name | Kind | Lives in | Status | Responsibility |
|---|---|---|---|---|
| `languagePrompt` and flag table | PURE | `cmd/define/language_prompt.go` | new | Validated effective language + flag preference → bounded prefix string |
| `nextDisplayUnit` | PURE | `cmd/define/display_unit.go` | new | Adjacent RI pair → complete bytes and 2 cells; otherwise existing rune width |
| `RenderLine` | PURE | `cmd/define/editor.go` | modified | Accept explicit prefix and preserve input/suggestion/cursor behavior |
| Column-based consumers | PURE | `render.go`, `screen.go`, `selection_frame.go` | modified | Share display-unit boundaries for measurement, wrapping, clipping, regions and selection |
| Prompt policy and loop wiring | INTEGRATION | `main.go`, `repl.go`, `replraw.go` | modified | Parse fallback flag; render effective session language through existing terminal seams |

ARCH-DRY: `languagePrompt(lang store.Lang, flags bool, cols int) string` owns the mapping and code fallback. It appends the existing `prompt` marker. Both loop shells and both raw render sites use it. `options.noFlags` stores only the invocation preference; language remains owned by deps/sessionSetLang.

ARCH-PURE: `RenderLine` takes an explicit prefix argument, not deps or environment access. Update every call in `replraw.go`, `editorloop_test.go`, `highlight_test.go`, and `activity_screen_test.go`; existing unrelated editor fixtures explicitly pass `prompt` to retain their isolated input-style assertions.

Display unit contract (caller has already skipped escape sequences):

```go
func nextDisplayUnit(s string) (size, cells int) {
    if s == "" { return 0, 0 }
    r, n := utf8.DecodeRuneInString(s)
    if r >= 0x1f1e6 && r <= 0x1f1ff && n < len(s) {
        next, m := utf8.DecodeRuneInString(s[n:])
        if next >= 0x1f1e6 && next <= 0x1f1ff { return n+m, 2 }
    }
    return n, cellWidth(r)
}
```

Consume complete units in `visibleCells`, `visibleIndex`, `clipVisible`, `walkSelectionRows`, `selectionCells`, and `markClickable`. Audit `cellSlice`, the test-only slice helper, for the same contract. `wrapText` already uses `visibleCells`. Byte-only ANSI scans stay unchanged because they do not decide columns. Test RI pairs directly with independently stated strings/columns, not only the existing terminal simulator, which also uses rune widths.

## Constraints and architecture

- ARCH-PURPOSE: cover live/raw, submitted/raw, and line-mode prompts, plus effective startup and switched language. Inventory all `RenderLine` and prompt writes. No inferred response-language work from #65 or adaptation from #64.
- ARCH-CONSTRAINTS: at most one bounded prefix per redraw; flag prefix 5 columns, code prefix 7. Mapping is constant-time; display walks remain O(input bytes), with at most one-rune lookahead. No network, probe wait, new goroutine or unbounded buffer. Existing screen bounds still govern narrow/short terminals; no half-flag may survive clipping.
- ARCH-SECURE: validate the tag through `store.ParseLang` before fallback rendering. Unknown valid tags stay visible; malformed strings cannot inject ANSI. Flag mode is an explicit two-cell rendering policy with a user-selectable fallback.
- ARCH-ORDER: existing `sessionSetLang` owns the transition. Startup/render reads current d.lang; switch success or session-only success changes next render; failure leaves it unchanged. Echo happens before dispatch, preserving the old indicator on that historical line. No new state machine or asynchronous state.
- ARCH-MOCK: reuse real editor/screen state, fake dictionaries/stores, memory clipboard, and PTY child-process harness. PTY verifies emitted bytes and lifecycle, not the host font. Direct physical-row/copy assertions provide independent layout expectations. Manually inspect flag and fallback in the actual terminal before release; record font-width limitations honestly.
- ARCH-FUNERAL: creates no durable runtime artifact; prefix strings and invocation preference end with the session. Issue and plan follow the normal archive lifecycle.

## Chunk 1: Complete prompt indicator

### Task 1: Pure prompt policy and display units

Files: create `language_prompt.go`, `language_prompt_test.go`, `display_unit.go`, `display_unit_test.go`; modify `render.go`, `screen.go`, `selection_frame.go` and their colocated tests as needed.

- [ ] Write table tests `TestLanguagePrompt` for every mapping, empty/default, unknown valid language, invalid/control-bearing tags, flag preference and plain fallback. Assert exact prefixes.
- [ ] Write `TestFlagDisplayBoundaries` with literal expectations: flag prefix width 5; code prefix width 7; clipping `🇪🇸x` at 1 yields empty and at 2 yields the full flag; physical rows for `a🇪🇸b` at width 2 equal `a`, `🇪🇸`, `b`. Include colored versions, adjacent flags, lone indicators, CJK and combining marks.
- [ ] Write `TestFlagSelectionCells` proving either of the two occupied columns selects/copies the whole flag; `visibleIndex` assigns all flag bytes the same starting column. Region marking must not insert escape sequences between RI runes. Include clips at odd/even widths. At a tiny 1-column screen, assert the live prefix is code-only, its emitted rows and selection agree, and an old leading-flag record emits no partial flag but returns intact after widening.
- [ ] Run `go test ./cmd/define -run 'TestLanguagePrompt|TestFlag' -count=1`; observe the missing API or geometry assertions fail before production changes.
- [ ] Implement the two pure helpers and route the enumerated column consumers through the display-unit reader. Preserve existing combining-mark attachment and ANSI styling behavior.
- [ ] Rerun focused tests. Add `FuzzFlagDisplayBoundaries` with independent RI-pair seeds, arbitrary surrounding text and widths: terminating walks, valid generated UTF-8, no partial generated pair under clipping, and round-trip complete flag selection. Keep byte-invalid input behavior compatible with existing functions.

### Task 2: Connect current language to every prompt

Files: modify `main.go`, `editor.go`, `repl.go`, `replraw.go`; add integration tests in `language_prompt_test.go`; update explicit-prefix calls in existing editor/highlight/activity tests.

- [ ] Write `TestLanguagePromptResolvedStartup` through `run` and a captured console in a temporary deck: persist es, supply no `-lang`, and assert the first prompt is Spanish; supply `-lang it` against that deck and assert Italian wins. Include default startup with no saved language.
- [ ] Write `TestLanguagePromptStartupAndSwitch` through both `runEditor` and interactive `replLines` using existing session/dictionary seams: default en, startup es, en→es→it, failed persistence, session-only switch, unknown tag. Assert submitted-line versus next-prompt identity and dispatched text/history contain no decoration.
- [ ] Write `TestNoFlagsKeepsEditorAndColor` through flag parsing and the existing console seam, and extend `TestREPLPromptRequiresBothStreams`/`TestREPLPromptOnlyWhenInteractive` for no prompt under pipes/redirects and code fallback under `-no-color`. Test invalid flag arguments normally fail usage.
- [ ] Write `TestLanguagePromptEditorGeometry` using a real pinned screen with completion, edits in the middle of input, narrow widths, and resize. Assert literal cursor/footer positions or an independent terminal oracle, and copy an existing definition word beneath the changed prompt budget. Do not calculate expected positions with production display helpers.
- [ ] Run `go test ./cmd/define -run 'TestLanguagePrompt|TestNoFlags|TestREPLPrompt' -count=1` and observe the missing indicator/policy failures.
- [ ] Add `-no-flags` and `options.noFlags`; change `RenderLine` to accept prefix explicitly. Supply `languagePrompt(d.lang, !opt.noFlags && opt.color, cols)` at live and submitted raw sites, where cols is read from `view.Size()` inside one local current-prefix closure. Use the same helper with `terminalCols(stdout)` at the interactive line-loop prompt. Do not cache the prefix across /lang. Update all required call sites.
- [ ] Run those tests plus `go test ./cmd/define -run 'RenderLine|Highlight|Activity|Selection|Screen|LangSwitch' -count=1`. Mutation-check removing the live prefix, freezing its language, omitting submitted decoration, and reverting display-unit consumers; relevant tests must fail.

### Task 3: Documentation, terminal check and close readiness

Files: modify `cmd/define/README.md`, `atlas/define.md`; add `TestPTYLanguagePrompt` to existing `pty_conformance_test.go`; update this plan and issue log.

- [ ] Document the mapping as visual conventions, unknown-code behavior, `-no-flags`, and plain-mode fallback. Update flag/help synchronization expectations using existing doc guards. Atlas explains the shared prompt helper and complete flag units; existing atlas index entry already links define.
- [ ] Add PTY test that starts with `-lang es -no-audio` in a temporary directory, observes the Spanish indicator, submits `/lang en`, observes the next English indicator, and exits cleanly; add a `-no-flags` case. Use the existing no-background environment and deck-permission harness so no real user data/model is touched.
- [ ] Run `go test ./... -count=1`, focused `go test -race ./cmd/define -run 'LanguagePrompt|NoFlags|Flag|REPLPrompt' -count=1`, `go vet ./...`, `GOOS=linux CGO_ENABLED=0 go build ./...`, and `git diff --check`; all must pass.
- [ ] Run `go test ./cmd/define -run '^$' -fuzz '^FuzzFlagDisplayBoundaries$' -fuzztime 10s` and `CONFORMANCE_STRICT=1 go test -tags conformance ./cmd/define -run '^TestPTYLanguagePrompt$' -count=1 -v`. Record results. Manually compare flag/code rendering in the operator's terminal, including a narrow resize; if not directly observable, record the remaining host-font check rather than claiming PTY proves it.
- [ ] Update issue log, check only evidenced tasks, commit implementation and run `sdlc close --issue 62 --verified '<behavior evidence>'`. Resolve boundary findings, commit the review record, then `sdlc pr` and `sdlc merge --yes` within the authorized work sequence. Proceed to #65 after #62 is complete.

## Revisions

- 2026-09-15: Initial plan after claim/start-plan and read-only prompt/geometry exploration. User directed #62 before #65 and distinguished #64's adaptive-learning scope. Runtime implementation awaits plan approval.

- 2026-09-15: Fresh review found undefined one-column behavior and missing persisted-startup coverage. Added true-width code fallback using the display Size seam (not wrapping-policy width), historical-record clipping/resize expectations, and run-level saved-language plus explicit-override tests.

- 2026-09-15: Fresh plan re-review approved both corrections with no remaining gaps. Operator approval remains pending.

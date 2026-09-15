# Language flag prompt implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Use superpowers-executing-plans for this small, connected change; the SDLC close gate owns its fresh-context code review.

**Goal:** Make the effective language visible before every interactive definition prompt without changing terminal geometry or submitted input.

**Architecture:** A pure language-prompt helper formats a flag or text code. Both interactive shells derive it from current session language rather than caching it. One shared display-unit reader keeps flag pairs intact in all column-based rendering consumers.

**Tech Stack:** Go, existing ANSI screen and selection code, existing stateful terminal/dictionary test seams and PTY harness. No new dependency or persistent setting.

**Status:** Implemented and verified 2026-09-15; awaiting SDLC close review.

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

`nextDisplayUnit` reads one rune or one adjacent regional-indicator pair after the caller skips ANSI escapes. It returns byte length and display columns; a complete pair occupies two cells, empty input is zero, and other input retains `cellWidth` behavior.

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

Files: `cmd/define/language_prompt.go`, `language_prompt_test.go`, `display_unit.go`, `display_unit_test.go`, `render.go`, `screen.go`, `selection_frame.go`, and their existing tests.

- [x] Implement and verify the shared prompt policy and whole-flag display units with test-first regression coverage; preserve current non-flag Unicode and ANSI behavior.

### Task 2: Prompt integration and terminal behavior

Files: `cmd/define/main.go`, `editor.go`, `repl.go`, `replraw.go`, `language_prompt_paths_test.go`, and existing editor/highlight/activity tests.

- [x] Integrate effective-language prefixes into both interactive shells and submitted lines, with explicit flag fallback and independent geometry regressions.

The existing `opt.tty` is permission for ANSI/raw editing, not proof that stdout is a terminal. Plain mode currently suppresses every prompt. To deliver the approved code fallback, distinguish interactive prompt ownership (both streams are terminals) from raw editing permission. Reuse the stdout terminal probe and existing injectable opt.tty test path; no-color sessions use a code prompt and line input, while pipes/redirects remain prompt-free. Entry-point tests must use a real PTY stdout because `run` probes it; non-file scripted stdin deliberately exercises the existing line fallback. Do not invent a new console-factory dependency just for these tests.

### Task 3: Documentation and close readiness

Files: `cmd/define/README.md`, `atlas/define.md`, `cmd/define/pty_conformance_test.go`, issue and plan records.

- [x] Document prompt mappings and fallback, verify actual terminal integration, and record evidence before close review and publication.

## Function-level verification strategy

| Production surface | Adversarial strategy | Independent mechanical guard |
|---|---|---|
| `languagePrompt` | Table-driven normalized, empty, unknown, malformed language and terminal-policy inputs | Literal expected prefixes; no control-bearing fallback; language remains visible when flag policy is disabled |
| `nextDisplayUnit` | Boundary and fuzz tests over RI runs, surrounding Unicode, truncation and widths | Positive byte progress for nonempty input; complete generated pairs remain indivisible; existing non-flag widths unchanged |
| `visibleCells` / `visibleIndex` | Composed flag/text/control sequences | Independently specified column totals and byte-to-column maps; neither indicator's bytes acquire a separate starting column |
| `clipVisible` / `walkSelectionRows` | Boundary-straddling complete flags with ANSI styles and narrow widths | Literal clipped strings/physical rows; no half-flag in output; width-one prompt policy and historical viewport clipping agree with painting |
| `selectionCells` / `markClickable` | Select/mark either column of a flag beside ordinary selectable text | Actual copy result preserves the entire original glyph; styles cannot split its bytes; neighboring action identity remains correct |
| `RenderLine` | Editing, completion and cursor positioning with varying prefixes | Independently specified emitted prefix and cursor controls; editable/history text excludes decoration |
| `run` / `repl` | Real PTY stdout plus scripted stdin, temporary saved deck, explicit overrides, default startup, plain-mode and redirection | First emitted prompt reflects resolved language and flag preference; no ANSI in no-color; no prompt when either stream is noninteractive |
| `runEditor` / `replLines` with `sessionSetLang` | Stateful dictionary/store seams driving successful, failed and session-only changes | Next observed prompt follows actual language effect; submitted line keeps pre-dispatch identity; recorded input never contains decoration |
| Pinned screen with live language prompt | Input/resize/selection sequences against an independently specified terminal layout | Cursor/footer coordinates and copied definition text stay correct; one-column fallback and later widening never expose half a generated flag |
| PTY child process | Existing isolated terminal harness, startup/switch/fallback/exit sequences with background work disabled | Emitted flag/code prompts follow commands and process exits cleanly; no real user deck/model/clipboard access |

Use mutation checks to demonstrate that incorrect prefix omission, stale-language caching, missing submitted decoration and split-unit consumers fail the corresponding regressions. Fuzz `FuzzFlagDisplayBoundaries`; compare results to independently constructed flag tokens rather than using production widths as the oracle. These tests implement the behavioral requirements above; individual cases belong in executable tests.

Verification commands (all must pass):

```sh
go test ./... -count=1
go test -race ./cmd/define -run 'LanguagePrompt|NoFlags|Flag|REPLPrompt' -count=1
go test ./cmd/define -run '^$' -fuzz '^FuzzFlagDisplayBoundaries$' -fuzztime 10s
go vet ./...
GOOS=linux CGO_ENABLED=0 go build ./...
CONFORMANCE_STRICT=1 go test -tags conformance ./cmd/define -run '^TestPTYLanguagePrompt$' -count=1 -v
git diff --check
```

PTY checks establish emitted bytes and raw-session lifecycle, not a host font's glyph width. Inspect flag/code rendering and a narrow resize in the operator's actual terminal before release; if that is not directly observable, record the remaining visual check explicitly. The fallback remains available regardless of that check.

After verification: update issue/plan evidence, commit implementation, run the single `sdlc close --issue 62 --verified '<behavior evidence>'` boundary review, resolve findings, commit the verdict and publish with `sdlc pr` → `sdlc merge --yes`. Proceed to #65 after #62 completes.

## Revisions

- 2026-09-15: Initial plan after claim/start-plan and read-only prompt/geometry exploration. User directed #62 before #65 and distinguished #64's adaptive-learning scope. Runtime implementation awaits plan approval.

- 2026-09-15: Fresh review found undefined one-column behavior and missing persisted-startup coverage. Added true-width code fallback using the display Size seam (not wrapping-policy width), historical-record clipping/resize expectations, and run-level saved-language plus explicit-override tests.

- 2026-09-15: Fresh plan re-review approved both corrections with no remaining gaps. Operator approval remains pending.

- 2026-09-15: Operator approved implementation. PQ-1 requested function-level strategies rather than test-case/procedural inventories; compressed task sections into named surfaces, adversarial strategies and independent guards while preserving the approved behavior. Inspection also clarified plain-mode prompt ownership and the real PTY stdout needed for entry-point coverage.

- 2026-09-15: Implementation complete. Literal screen, copy, cursor and resize tests cover both flag cells and width 1; local review also corrected clickable regions beginning inside a flag. Full suite, focused race, 10s fuzz, vet, Linux build and strict PTY passed. Mutation checks caught omitted live/submitted prefixes, frozen language, wrapping-policy width and rune-only geometry; originals restored. Host-font rendering is not directly observable here, so that visual check remains explicit.

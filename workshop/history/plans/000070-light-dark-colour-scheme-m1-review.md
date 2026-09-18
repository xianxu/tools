# Boundary Review — tools#70 (milestone M1)

| field | value |
|-------|-------|
| issue | 70 — define: switch between a light and a dark colour scheme |
| repo | tools |
| issue file | workshop/issues/000070-light-dark-colour-scheme.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 75370a2905a22a70f2935efec78e96d1f1b7c9e5..ad4a21db52dda0a6d174f93f57e7753de6760d72 |
| command | sdlc milestone-close --issue 70 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-09-17T22:58:39-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

M1 delivers what the plan promised, and I found nothing Critical or Important. A row now records only whether it is tinted (`rowPaint.tinted`). The shade comes from `schemeTint` at paint time, using one process-wide `schemeHolder` (`atomic.Pointer` to an immutable `schemeState`). Both places production builds a screen attach the holder before any other goroutine sees it: `replraw.go:83` and `play_cmd.go:115`. The frame reads it once in `layoutSelectionFrame`, and `paintedTranscript` reads it once. The dead tint paths are gone and the flags are narrowed. At `ad4a21d` the build, `go vet` and `go vet -tags conformance` are clean. `go test ./cmd/define/...` passes except two pty-backed tests, `TestLanguageTintInvocation` and `TestLanguagePromptStartup`. Both fail at `pty.Open` with "operation not permitted", even outside the sandbox, so this environment can't open a pty and the code isn't at fault. The `-race` run over the scheme, tint, sitting and console tests reported no data race. Confidence is medium because I could not run `TestLanguageTintInvocation` myself. It is the only end-to-end check that `-scheme light` gets through `run()` to the holder; the code looks right on reading. The four findings are all Minor.

1. **Strengths**
   - Changing `rowPaint.background string` to `tinted bool` removes a whole class of bad input rather than filtering it. The deleted "untrusted background string" test cases say so honestly (`language_row_test.go`, `output_layout_test.go`), and the frozen-at-production rule for `/lang` is kept in the rewritten comment at `output_layout.go:11-15`.
   - The ARCH-ORDER structure is real. `choose`, `forget` and `detect` are the only writers, each reports whether the painted shade changed, and `TestSchemeHolder` tests those reports with a *differing* reply under a choice (it doesn't just repeat the earlier one).
   - The wiring tests drive the real attach points. `TestNewConsoleAttachesTheSchemeHolder` goes through `newConsole`, and `TestASittingPaintsInTheEditorsScheme` goes through the real `sittingInPlace`, so removing either `attachScheme` call turns them red. The console test also shows that `writeOutput`'s `sc` argument is ignored when the sink is a screen.
   - The deletion was justified with evidence. The simulated deletion found exactly five failures, all direct `Render`/painter calls with a non-zero tint. I confirmed `projectLanguageText` returns `text: rendered`, so replacing each `dictionaryText` call with its `rendered` argument doesn't change output (`render.go:146,195,213,235`).
   - `TestTintForGatesOnColour` pins the colour gate exactly where there is no second check (practice output and the answer writer). The two-pass rewrite of `TestOutputLayoutWrapsPaintAndActions` keeps what the test could tell apart after two shades became one bit.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **`scheme.go:30,49` (ARCH-ORDER): the state can hold combinations the spec ruled out.** The spec shape was `choice *{value, source: flag|saved|session}`. The code uses `choice` + `chosenBy schemeSource`, where `sourceDefault` means "no choice", and `withChoice` accepts any source. So `withChoice(v, sourceDetected)` reports a user's choice as detected, `withChoice(v, sourceDefault)` keeps a hidden choice, and `withChoice("", sourceFlag)` makes `effective()` return an empty scheme. Current callers are all legal, but M2's `applyScheme`/`describeScheme` will print this source to the user. Fix sketch: give the choice its own source type covering only flag, saved and session, or use `*schemeChoice`, before M2 builds on it.
   - **`render_helpers_test.go:82` and `play_loop_test.go:2417,3049` (ARCH-DRY, test fidelity): two tests check the live board footer through a copy production never calls.**
     - `boardFooter` is a test-only duplicate of production's `boardFooterOutput` (`practice_output.go:154`). It now carries the only copy of the "THE FORM IS FIRST, which is load-bearing" explanation, and still calls itself "the live edge".
     - `TestBoardFooterPutsTheFormsOwnRowsFirst` and the `fitsABoard` row count use this copy. A row-order regression in `boardFooterOutput` would pass both.
     - This predates #70 (`boardFooter` was already test-only at the base commit); M1 just moved it into a test file.
     - Fix sketch: point both tests at `paintedBoardFooterForTest`, move the explanation onto `boardFooterOutput`, and delete `boardFooter`.
   - **`atlas/define.md:503-515`: the paragraph describes more than M1 ships.** It covers `saved`/`session` sources, `forget`/`detect`, and "what the terminal reported" as if they were live. In M1 none of those has a production caller, and the plan says docs should describe only what each milestone ships. One clause saying detection and saving arrive in M2/M3 would fix it.
   - **Issue Done-when (line 320) and Spec §Repaint say "deleted … with its tint assertions ported".** What actually happened: the five renderers were moved into `render_helpers_test.go`, and the per-fragment tint assertions were dropped because they described behaviour production never had. The Log's Task 3 entry explains this, but the list of "Spec revisions the plan makes" leaves it out.

5. **Test coverage**
   - The new tests pin real behaviour: the pure transition sequences, holder change-reporting, a screen repainting history plus the exit transcript, and both production attach points. The mutation checks the plan claims hold up on reading; each named mutation would turn its test red.
   - The only in-process run through `run()` that checks `-scheme dark|light|auto` reaches the painted shade is `TestLanguageTintInvocation`, which needs a pty. Environments that block ptys (like this one) lose that check, but it fails loudly rather than skipping.
   - `TestSchemeHolderConcurrentReaders` catches the plain-pointer race. It doesn't check that one frame uses one shade; that single-read property is only visible in the code (`screen.go:600`, `output_screen.go:78`).
   - Pty tests don't set their own `XDG_CONFIG_HOME` yet. That's correct for M1 (nothing is saved), but it is a required part of M2 (Task 11).

6. **Architecture notes for upcoming work**
   - ARCH-DRY: pass. `schemeTint` is the only source of the shade, and `parseSchemeArg` is shared by the flag and (in M2) the command. The only leftover is `boardFooter` (above).
   - ARCH-PURE: pass. `paintLanguageRow`, `serializeOutput` and `paintOutputChunk` take `sc` as a parameter. Only the screen and the writers read the holder. `tintPolicy` now carries a mutable holder pointer, but only the answer writer reads it.
   - ARCH-PURPOSE: pass. Every shade consumer derives from `schemeTint(sc)`; no `48;5;236`/`254` literal remains outside `language_style.go:10-11`. Both production screen constructions attach the holder, and the non-screen writers (`main.go:1178`, `practice_language.go:48,128`, `answerwrap.go:354`) read it at write time.
   - ARCH-MOCK: N/A for M1, which touches no external dependency. The terminal fake is M3.
   - ARCH-CONSTRAINTS: pass. One atomic load per frame and no new waits.
   - ARCH-SECURE: pass. `-scheme` and `-language-tint` are parsed into closed values at the flag boundary. No persisted input until M2; `ReadScheme`'s 64-byte cap and enum parse are already planned.
   - ARCH-ORDER: pass with the Minor above. The load-then-store without compare-and-swap is safe only while there is a single writer. M1 has no writer after construction, so M3 must keep `detect` on the loop goroutine that is currently running, including inside a sitting.
   - ARCH-FUNERAL: pass. M1 creates nothing durable because the holder is in-memory and dies with the process. M2's saved file already has `ClearScheme` and a size bound planned.
   - For M2: `selectionFrame.same` doesn't compare the scheme, so a `/scheme` switch mid-drag keeps the selection. That is correct, since the text is unchanged; keep it that way.

7. **Plan revision recommendations**
   - Issue: add a Revisions/Log line reconciling Done-when line 320 and Spec §Repaint ("Dead code is deleted, not converted … PORTED") with what was done: the renderers moved to `render_helpers_test.go` as test helpers, and the per-fragment tint assertions were dropped (production tints whole sections, which `TestDefinitionOutputUniformSections` pins).
   - Plan: if the `schemeState` shape changes per the ARCH-ORDER Minor, add a Revisions entry to Task 2 so the plan stops showing `chosenBy schemeSource`.

```findings
findings:
  - id: new
    severity: Minor
    family: state-shape-admits-illegal-combinations
    title: |
      schemeState.withChoice accepts any schemeSource, so choice-by-detected, choice-by-default and empty-choice states are representable
    detail: |
      The spec's choice *{value, source: flag|saved|session} is implemented as choice plus chosenBy schemeSource with sourceDefault as the absent marker (cmd/define/scheme.go:30,49). M2's describeScheme will print this source to the user; give the choice its own source type or use *schemeChoice before M2 builds on it (ARCH-ORDER).
  - id: new
    severity: Minor
    family: tests-pin-a-shadow-of-the-live-path
    title: |
      Two live-edge footer invariants are tested through the test-only boardFooter, not production's boardFooterOutput
    detail: |
      TestBoardFooterPutsTheFormsOwnRowsFirst and the fitsABoard row count (play_loop_test.go:2417,3049) call boardFooter (render_helpers_test.go:82), which also now holds the only copy of the load-bearing rationale. Point them at paintedBoardFooterForTest, move the comment to boardFooterOutput, and delete boardFooter (ARCH-DRY). Pre-existing; M1 moved it into a test file.
  - id: new
    severity: Minor
    family: docs-describe-unshipped-surface
    title: |
      The atlas paragraph on paint-time shade describes saved/session sources and detect/forget as live in M1
    detail: |
      atlas/define.md:503-515 describes the terminal-reported value and the saved and session sources as if they worked now, but none has a production caller until M2/M3. The plan says docs should describe only what each milestone ships; add one clause saying so.
  - id: new
    severity: Minor
    family: spec-done-when-drifts-from-delivery
    title: |
      Done-when says dead paths are deleted with tint assertions ported; the renderers were moved to test helpers and fragment-tint assertions dropped
    detail: |
      The Log's Task 3 entry explains why, but the list of spec revisions the plan makes omits it, so the issue's Done-when (line 320) and Spec section on Repaint still claim what was not done.
```

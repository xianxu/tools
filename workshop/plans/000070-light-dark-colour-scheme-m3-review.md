# Boundary Review — tools#70 (milestone M3)

| field | value |
|-------|-------|
| issue | 70 — define: switch between a light and a dark colour scheme |
| repo | tools |
| issue file | workshop/issues/000070-light-dark-colour-scheme.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 97815dbf85152eb55bc576926ef56b6783238dd5..92866197d8b65e65724887d17094d57ce0c48a04 |
| command | sdlc milestone-close --issue 70 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-18T09:36:44-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M3 detection code is correct. The OSC decoder can't leak a reply as typing, the query goes out once and only where a tint can appear, and every consumer of `KeyBackground` handles it. Four mutations I ran in a scratch worktree confirmed that the key guards are pinned. Two gaps stand between this and SHIP, and both are cheap to fix. First, two mutations survive: removing the sitting's repaint after a reply, and removing the paired-ink reset. Second, several ticked M3 steps name evidence that was never written down. The biggest is the three-terminal × two-appearance manual check, which the atlas says was "not itemised". I found no Critical issues.

**Strengths**
1. `decodeOSC` (`key.go:482-521`) keeps the swallow separate from the parse. Any byte that can't continue a reply aborts to exactly the decoding that existed before #70. The tests cover every strict prefix, the 64/65-byte cap with both terminators, and split writes through `readInput`. Removing the printable-range abort turns `TestDecodeOSCAbortsAsToday` red (verified).
2. `terminalQueries` (`rawterm.go:137-149`) is a list that `TestEveryEnabledInputModeIsDecoded` derives from, and it fails closed when a query has no row. A new question can't ship without a decoded answer.
3. The dropped-reply guard (`selection_input.go:193-195`) leaves the `saturated` latch untouched. `TestADroppedReplyIsSilent` uses a zero-length write on an `io.Pipe` as a barrier, which is a deterministic ordering seam. The comment explaining why the `default:` arm has no guard (this goroutine is the channel's only sender) holds up.
4. `TestRawEditorBackgroundReplyRepaints` sends real bytes through `readInput` → `runEditor`, and `countingDict` proves no reply byte reached the line. Removing `draw()` turns it red, and removing the `cancelPointerInput` guard turns `TestAReplyMidDragKeepsTheSelection` red (both verified).
5. The paired ink goes through `paintLanguageRow`, the only tint painter, so the screen frame, the exit transcript and `serializeOutput` all get it. `sourceColours` reads background and foreground in one parse.

**Critical findings:** none.

**Important findings**
- **The sitting's repaint after a reply is unpinned** (`play_loop.go:315-317`). I removed `show()` and the whole default package stayed green; the only failures are the three pty-backed tests that also fail on unmutated HEAD in this environment. `TestASittingIgnoresABackgroundReply` checks `d.scheme.Scheme()`, which is shared state. The plan's Task 16 test 2 promised "the sitting's screen repaints light". This repaint is the path that paints `--play`'s first question correctly on a light terminal. See the family rule in the findings block.
- **Ticked M3 steps lack their evidence.** Task 18's matrix is not recorded, and neither are the run results from Tasks 17 and 18 or the mutation names. The README names three terminals without a record behind it. The full list is in the findings block.

**Minor findings**
- `unfill`'s `inkOff` (`language_row.go:34-37`): removing it leaves the suite green, so ink could leak onto a producer's background or an excluded cell. The atlas says the ink steps aside for either.
- ARCH-DRY: the `KeyBackground` intercept is copied in `runEditor` (`replraw.go:553-561`) and `playSession` (`play_loop.go:311-318`). The neighbouring `viewportGesture` helper exists precisely so the two loops can't disagree.
- Not raised as a finding: the `-h` text (`main.go:611-613`) still doesn't say that `auto` asks the terminal. The flag help does say it.

**Test coverage notes**
- `go vet -tags conformance ./cmd/define` and `GOOS=linux go build` are clean. The targeted M3 tests pass.
- I couldn't run the pty conformance tests here: opening a pty fails with "operation not permitted". `TestLanguagePromptStartup`, `TestLanguageTintInvocation` and `TestSavedSchemeGovernsALookup` fail the same way on unmutated HEAD. This is an environment limit, not a finding.
- `TestAReplyDuringPlayReachesTheEditor` checks `PaintedTranscript`. That re-reads the holder when called, so it adds nothing beyond the `effective()` check before it.

**Architecture**
- **ARCH-DRY:** flag (Minor, above).
- **ARCH-PURE:** pass. The decoder, the colour parse and `wantsBackground` are pure, and the intercepts are thin.
- **ARCH-PURPOSE:** pass on code. The sweep over `KeyBackground` consumers is complete: editor, sitting, `sittingKeyHandling`, the router via `cancelPointerInput`, and the length-check drop site. The live-check part of the purpose is under-recorded; that's covered by the evidence finding.
- **ARCH-MOCK:** the seam passes. Bytes come in through `readInput`, go out through `rawSession.control`, and pty tests play light, dark and silent terminals. The record of the live conformance check is thin (evidence finding).
- **ARCH-CONSTRAINTS:** pass. The query is 8 bytes, nothing waits for the answer, replies are capped at 64 bytes, and each read rescans at most 64 bytes.
- **ARCH-SECURE:** pass. The reply is treated as untrusted and bounded, then parsed into a closed enum. Anything unparseable becomes `KeyUnknown`, so no value is invented.
- **ARCH-ORDER:** pass. `detect` is a pure transition on a holder with one writer. Late, duplicate and outranked replies are tested as sequences, and byte streams make arrival order controllable. `Key.Background` is a kind-specific field on an event, which matches `Key`'s existing shape. It is not state carried between events.
- **ARCH-FUNERAL:** pass. M3 creates nothing durable because detection lives in the in-memory holder and dies with the process. The only leftover, a reply echoed to the shell after a fast quit, is documented and limited to one reply.

**Plan revision recommendations**
- Task 18 manual check: either record the matrix, or add a Revisions entry that narrows Done-when bullet 1 to what was actually checked.
- Task 16 test 2: restore the assertion on the sitting's frame (preferred), or record why it was dropped. Test 3 was built with `con.newSitting` and a hand-built key rather than `/play` plus a `/scheme` report. Test 4 checks the `dragging` flag rather than finishing the drag to a copy. Both are acceptable, but they should be noted.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      M3 Tasks 13-15 still carry full implementations, and the window adds no lessons.md rule telling future plans to name functions plus one strategy line. Record that rule at close; the plan itself needs no rewrite.
findings:
  - id: new
    severity: Important
    family: tests-pin-a-shadow-of-the-live-path
    title: |
      The sitting's repaint after a background reply is unpinned: removing show() at play_loop.go:315-317 leaves the suite green
    detail: |
      This is the 2nd finding in family tests-pin-a-shadow-of-the-live-path. Rule: a consumer's test checks what that consumer itself outputs (the frame it paints, the answer it records, the notice it posts), never only the shared state it writes. PaintedTranscript is not evidence of a repaint, because it re-reads the holder when called. Consumers of KeyBackground: editor frame pinned (dropping draw() goes red, verified); router drag and drop-site notice pinned; sitting frame NOT pinned (TestASittingIgnoresABackgroundReply checks d.scheme.Scheme(); dropping show() stayed green, verified), and this is the path that paints --play's first question on a light terminal. TestAReplyDuringPlayReachesTheEditor's PaintedTranscript check repeats its effective() check. Fix: follow TestASittingPaintsInTheEditorsScheme (a tinted row on the sitting's screen), send the reply, and assert lastFrame of the tty carries languageLight. Record the rule in lessons.md.
  - id: new
    severity: Important
    family: ticked-step-lacks-its-evidence
    title: |
      Ticked M3 steps name evidence the record does not hold, including the three-terminal x two-appearance manual check
    detail: |
      This is the 2nd finding in family ticked-step-lacks-its-evidence; the 1st was M2's BR-6. Rule: tick a step that names evidence (a run result, "name what ran", a mutation list, a recorded matrix) only in the commit that writes that evidence where the step says it goes. If a step was done differently, add a Revisions entry saying what was done instead. Steps to sweep: Task 14 Step 4 (three 30 s fuzz runs); the mutation steps of Tasks 13-17 (the Log gives counts, not names as M1 and M2 did); Task 17 Step 3 (which TestPTY tests ran or skipped under CONFORMANCE_STRICT; TestPTYBackgroundDetection sits behind bilingualNativeProbe); Task 18 step 1 (full suite, -race, vet, linux build, strict results); Task 18 step 2 (the matrix; the atlas says "not itemised"). Done-when bullet 1 still claims Terminal.app, iTerm2 and Ghostty in both appearances. Bullet 6 claims Alt-] and #rrggbb are pinned "through each loop shell", but only the decoder and readInput test them. The README says those three terminals reply. Either record the matrix or add a Revisions entry that narrows bullet 1, and make the README match what is recorded. Record the rule in lessons.md.
  - id: new
    severity: Minor
    family: branch-survives-its-mutation
    title: |
      The paired-ink reset in unfill (language_row.go:34-37) is unpinned; removing inkOff leaves the suite green
    detail: |
      Verified by mutation. If it regressed, 235 or 252 ink would carry onto a producer's own background (the \x1b[42mc cell, column 3, in TestLanguageRowStylesAndExclusions) and onto uncoloured excluded answer cells. The atlas says the ink steps aside for either. Fix: assert cells[3].fg == -1 there, and add an uncoloured excluded cell with fg == -1.
  - id: new
    severity: Minor
    family: duplicated-logic-not-extracted
    title: |
      The KeyBackground intercept is copied in runEditor (replraw.go:553-561) and playSession (play_loop.go:311-318)
    detail: |
      ARCH-DRY. The neighbouring viewportGesture helper exists precisely so the two loops cannot disagree about a shared key. A one-line helper (apply the report through d.scheme.detect and report whether to repaint) would make the report rule a single source.
```

---

## Re-review — 2026-09-18T09:55:13-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 70 — define: switch between a light and a dark colour scheme |
| repo | tools |
| issue file | workshop/issues/000070-light-dark-colour-scheme.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 97815dbf85152eb55bc576926ef56b6783238dd5..e4d5c9eedabef2adf51a8eaeb8eb8cd865fc99e6 |
| command | sdlc milestone-close --issue 70 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-18T09:55:13-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All five open findings are fixed, and I confirmed each fix in a scratch copy of HEAD where one was testable. The sitting's repaint and the ink reset are now pinned: removing either turns its new test red. The editor-after-play test now checks the frame the editor paints on resume. When I removed that repaint, the test went red; the old `PaintedTranscript` check would have stayed green. Both loops now share one `terminalReport` helper. The M3 evidence is written out by name, and two named mutations I re-ran do turn their tests red. There are no Critical findings. One Important finding is new and cheap to fix. The only real-terminal evidence cited for detection is the operator's "working" check. The record cannot tell that result apart from a saved `light` choice doing the painting, and the record also says a saved `light` was in force.

**1. Strengths**
- `terminalReport` (`replraw.go:144-153`) makes the report rule a single source for `runEditor` and `playSession`, following `viewportGesture`'s precedent (ARCH-DRY).
- `TestAReplyDuringPlayReachesTheEditor` now reads `lastFrame` of the tty. Removing `l.repaint()` from `resume()` (`screen.go:1188-1196`) turns it red with `""`, so the check pins the repaint on resume itself, not the holder.
- `TestTheInkStepsAsideWithTheTint` (`language_row_test.go:216-231`) covers an excluded answer cell and a producer's own background in one row. Removing `inkOff` turns both assertions red.
- The loop shells now test the `#hex` and Alt-] cases (`key_background_test.go:57-58, 91-94`). In the sitting, `gradeKey` returns `"1"` on every run (I checked), so `ESC ] 1` really does enter the waiting prefix before the abort, as the comment claims.
- The issue Log now names every M3 mutation, the three fuzz runs, the strict conformance set (all five tests ran, none skipped) and the boundary runs. Two spot-checks held: dropping the ESC-case cap check turns `TestDecodeOSCCap` red, and dropping the length-check report guard turns `TestADroppedReplyIsSilent` red.

**2. Critical findings**
None.

**3. Important findings**
- **Done-when bullet 1 cites the operator check as evidence of detection, but the record cannot tell detection from a saved choice.** This is the 3rd finding in family `tests-pin-a-shadow-of-the-live-path`.
  - The Log's evidence list for "Light terminal → 254, dark → 236, no flag, no saved file" ends with "operator check".
  - The only recorded detail of that check is the first run, and the Log says a saved `light` was in force then (`/scheme light` at 07:25, "not yet `/scheme auto`'d"). The re-check is recorded only as "working", with no `/scheme` report.
  - The atlas calls that first run "a light profile". The Log describes a terminal whose default text is white, which is the dark-theme mismatch that motivated the paired ink. The two records contradict each other.
  - **Rule for the whole family:** evidence for a path, whether a test assertion or a manual check, must be something only that path can produce. The shared holder, `PaintedTranscript`'s re-read, and a light tint that a saved choice also paints all fail this test.
  - **Sweep of the Done-when evidence list:** only this item fails. The pty `(detected)` reports, the in-process light-reply frame and the late-reply repaint all distinguish detection from its fallbacks.
  - **Fix:**
    - Extend the new lessons.md rule so it covers manual evidence, not just tests.
    - For this instance, choose one:
      - Record one real-terminal `/scheme` report reading `light (detected)` / `dark (detected)` after `/scheme auto`.
      - Or state that the live check did not establish detection, so detection's only evidence is the modelled pty terminal and the ARCH-MOCK live check is still owed.
    - Either way, fix the atlas wording "a light profile".

**4. Minor findings**
None newly raised. Not raised:
- `key.go:5` puts the `store` import inside the standard-library import group. gofmt accepts it, but it doesn't match the rest of the package.
- `TestASittingRepaintsOnABackgroundReply` sets `nested.interval` without the lock while the sitting's goroutine is running. It was clean in 30 runs under `-race`, because `Draw` paints without reading `interval`.

**5. Test coverage notes**
- `go test ./cmd/define/` is green here except `TestLanguagePromptStartup`, `TestLanguageTintInvocation` and `TestSavedSchemeGovernsALookup`. All three fail with "operation not permitted" when opening a pty, which is an environment limit here, not a code problem; the previous round saw the same failures on unmutated HEAD.
- I could not run the pty conformance tests here.
- The M3 consumer tests pass 3 times under `-race`, and the new sitting test passes 30 times.
- `go vet` is clean with and without `-tags conformance`.
- gofmt flags two files, but neither is in this review window.
- Every test named in the Done-when evidence list exists.
- Mutations I ran, each restored afterwards:

  | Mutation | Test that went red |
  |---|---|
  | `show()` removed from the sitting | `TestASittingRepaintsOnABackgroundReply` (timeout) |
  | `resume()`'s repaint removed | `TestAReplyDuringPlayReachesTheEditor` |
  | `inkOff` removed | `TestTheInkStepsAsideWithTheTint` |
  | ESC-case cap check removed | `TestDecodeOSCCap` |
  | Length-check report guard removed | `TestADroppedReplyIsSilent` |

**6. Architectural notes**

| Marker | Result | Notes |
|---|---|---|
| ARCH-DRY | pass | `terminalReport` is the one report rule. The test-side `oscLen` models a terminal, so it is not a copy of production logic. |
| ARCH-PURE | pass | The decoder, the colour parse, `wantsBackground` and `terminalReport` are pure or thin. |
| ARCH-PURPOSE | code passes; live evidence flagged | Only two goroutines read the key channel (the editor's and the sitting's). Every `KeyBackground` consumer is handled: both loops, `sittingKeyHandling`, `cancelPointerInput`, and the length-check drop. The live-terminal half of the purpose is the Important finding. |
| ARCH-MOCK | seam passes; live check flagged | The pty tests play light, dark and silent terminals, but the live check does not yet show a real terminal's reply being decoded (the Important finding). |
| ARCH-CONSTRAINTS | pass | 8-byte query, no waiting, replies capped at 64 bytes. |
| ARCH-SECURE | pass | The reply is bounded and parsed into a closed enum; anything else becomes `KeyUnknown`. |
| ARCH-ORDER | pass | There is one writer: the loop in force. The tests drive ordering through key channels or byte streams. |
| ARCH-FUNERAL | pass | M3 creates nothing durable: detection lives in the in-memory holder and dies with the process. |

**7. Plan revision recommendations**
- Amend the 2026-09-18 M3 boundary-review Revisions entry for Task 18 step 2. It should say either that a `(detected)` report was observed in a real terminal (naming the terminal, the appearance and the reply), or that the manual check did not establish detection and the live conformance check is still owed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      lessons.md now carries "A plan names functions and one strategy line each, not their code (#70)", the rule the prior round asked for at close; the plan itself needs no rewrite.
  - id: BR-10
    disposition: addressed
    note: |
      TestASittingRepaintsOnABackgroundReply reads the sitting's painted frame; removing show() turns it red (timeout). The editor-after-play test reads lastFrame; removing resume()'s repaint turns it red. Both verified by mutation in a scratch copy of HEAD.
  - id: BR-11
    disposition: addressed
    note: |
      Log names every M3 mutation, fuzz run, strict set and boundary run (two spot-checked red); bullet 1 narrowed by a Log REVISION and plan Revisions; README no longer names terminals; the loop shells now test #hex and Alt-]. The separate distinguishing-evidence gap is raised as a new finding.
  - id: BR-12
    disposition: addressed
    note: |
      TestTheInkStepsAsideWithTheTint pins the excluded cell and the producer background at fg -1; removing inkOff reddens both (verified).
  - id: BR-13
    disposition: addressed
    note: |
      terminalReport (replraw.go:144-153) is the one report rule, used by runEditor and playSession.
findings:
  - id: new
    severity: Important
    family: tests-pin-a-shadow-of-the-live-path
    title: |
      Done-when bullet 1 cites the operator check as evidence of detection, but the record cannot tell detection from a saved choice
    detail: |
      This is the 3rd finding in family tests-pin-a-shadow-of-the-live-path. Rule for the whole family: evidence for a path (a test assertion OR a manual check) must be something only that path can produce; the shared holder, PaintedTranscript's re-read, and a light tint that a saved choice also paints all fail this. The Log's evidence for "light -> 254, dark -> 236, no flag, no saved file" ends with "operator check". The only recorded detail of that check is the first run, and the Log says a saved light was in force then ("not yet /scheme auto'd"). The re-check is recorded only as "working", with no /scheme report. The atlas also calls that run "a light profile", while the Log describes a terminal with white default text. Sweep of the Done-when evidence list: only this item fails; the pty (detected) reports, the in-process light-reply frame and the late-reply repaint all distinguish detection. Fix: extend the lessons.md rule from tests to manual evidence. For this instance, either record one real-terminal /scheme report reading "(detected)" in each appearance after /scheme auto, or state that the live check did not establish detection, so detection's only evidence is the modelled pty terminal and the ARCH-MOCK live check is still owed. Either way, correct the atlas wording.
```

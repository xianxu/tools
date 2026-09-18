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

# Boundary Review — tools#81 (whole-issue close)

| field | value |
|-------|-------|
| issue | 81 — define: editor diagnostics are clipped at the terminal edge |
| repo | tools |
| issue file | workshop/issues/000081-define-editor-diagnostics-are-clipped-at-the-terminal-edge.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1236de2a5e303b3626d8f6bbded85d845bee6752..75084f2da2af98aaf975d1d11a62ba4d350157e6 |
| command | sdlc close --issue 81 |
| reviewer | claude |
| timestamp | 2026-09-23T17:12:26-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The fix is small and correctly placed, and it reuses the existing wrap. `liveScreen.Diagnostics()` (`cmd/define/screen.go:935-950`) sends the editor's stderr through `wrapWritten` at the screen's current `l.cols`, and `replraw.go:129` wires it in. The same writer also reaches sittings, because `con.newSitting` receives the editor's `stderr`. The double wrap on a pinned screen really is idempotent: `wrapText` returns a line unchanged when it already fits, and a single overflowing word stays unchanged. Two gaps keep this from a clean SHIP, and neither blocks the gate. First, `wrapText` deliberately never breaks a word, and a provider's JSON error body is mostly one long word, so on a narrower terminal the part the operator wanted to see is still clipped. Second, the only test is a `darwin && conformance` pty test, which plain `go test` never runs, and nothing checks the second Done-when clause (every row fits the width).

1. **Strengths**
   - ARCH-DRY holds: it reuses `wrapWritten`, the same wrap the pinned screen uses, instead of adding a second wrapper (`screen.go:945`).
   - The design keeps stdout unwrapped on purpose, which protects `answerWrapWriter`'s word-carrying across streamed chunks. The doc comment explains why.
   - `Write` returns `len(p)` (the caller's byte count), consistent with `liveScreen.Write`, and it takes the lock the same way.
   - The pty test first confirms the scripted 400 actually showed up ("bad request") and only then checks the tail, so a misconfigured fake cannot make it pass by accident.

2. **Critical:** none.

3. **Important**
   - **Words wider than the terminal are still clipped** (ARCH-PURPOSE). `wrapText` puts an over-long word on its own row "rather than being cut" (`render.go:667-670`), and `Paint` then clips that row. A real Anthropic 400 body starts with the single word `{"type":"error","error":{"type":"invalid_request_error","message":"<first word>`, which is about 71 cells or more. Below roughly 72 columns, the start of the provider's message is cut off again. That is the operator's original complaint ("I can't see the details").
     - The test fixture's word is `{"type":"error","error":{"type":"api_error","message":"scripted`, about 60 cells, so at 80 columns the test never reaches this case.
     - Every word of this kind in the window: the JSON body word, and the `POST "<url>":` URL word (long URLs).
     - Fix sketch: in the diagnostics path only, hard-break any word wider than `cols` into pieces of `cols` cells (`cellSlice` exists at `render.go:682`). Test it with an error body that has no spaces at 40 columns.
   - **Done-when is covered only by a test that normal runs skip.** The pty test is `//go:build darwin && conformance`, so `go test ./cmd/define/...` does not run it. It only checks that the tail string is present. Nothing asserts clause 2, "every buffer row written through the editor's stderr fits the screen width".
     - Fix sketch: add an in-process test. Build `newLiveScreen(&buf, 24, 40)` with a negative interval, write a 200-character message through `Diagnostics()`, and assert that every buffer row is at most 40 cells wide. Add a second case with a word longer than the width, which pins the finding above. This is pure and runs everywhere (ARCH-PURE).

4. **Minor**
   - Wrapping a `define: …\r\n` line drops its trailing `\r`, because `strings.Fields` treats it as whitespace. A line that fits keeps it. This is harmless in the buffer today, but the two cases behave differently.
   - The comment "each is one whole `Fprintf`" is true of the call sites I read (`replraw.go:395, 663, 798`) but nothing enforces it. A later partial-line write to stderr would be wrapped one fragment at a time.

5. **Test coverage notes:** The red-then-green claim rests on the pty run in the Log. I did not re-run it here: the tag needs a real pty, and the sandbox refuses one. `go build ./...` and `go vet ./cmd/define/` passed. `go test ./cmd/define/` was still running in the background when I wrote this, so it is unconfirmed. The Log reports 13 pre-existing `TestPTY*` failures caused by the first-run deck prompt; they should get their own issue, as the Log says.

6. **Architectural notes**
   - ARCH-DRY: pass. The wrap is reused, and `diagnostics.Write` has the same shape as `liveScreen.Write` apart from the pre-wrap. That duplication is acceptable at two sites.
   - ARCH-PURE: pass for the code. The test gap is the one above: the pure wrap-to-width behaviour has no pure test.
   - ARCH-PURPOSE: flagged. Words wider than the terminal still defeat the purpose (first Important finding).

7. **Plan revision recommendations:** Add a `## Revisions` entry noting that diagnostics hard-break words wider than the terminal (unlike `wrapText`'s policy for definitions), and that an in-process width test backs clause 2.

```findings
findings:
  - id: new
    severity: Important
    family: wrap-leaves-overlong-token-clipped
    title: |
      Diagnostic tokens wider than the terminal are still clipped (wrapText never breaks a word)
    detail: |
      wrapText (render.go:667) puts an over-long word on its own row and Paint clips it; a real provider 400 body's first token is about 71 cells, so below about 72 cols the provider message is cut again. Instances in the window: the JSON body token and the POST url token. Fix: hard-break tokens over cols in the diagnostics path only, and test at 40 cols with a no-space body.
  - id: new
    severity: Important
    family: done-when-clause-untested
    title: |
      No in-process test for Diagnostics; clause 2 (every row fits width) is never asserted
    detail: |
      The only test is darwin+conformance tagged and checks tail presence only. Add a pure newLiveScreen test writing a long message through Diagnostics() and asserting every buffer row is at most cols cells, including an overlong-token case.
  - id: new
    severity: Minor
    family: wrap-drops-carriage-return
    title: |
      Wrapping a CRLF-terminated diagnostic drops its trailing CR while fitting lines keep it
```

---

## Re-review — 2026-09-23T17:18:09-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 81 — define: editor diagnostics are clipped at the terminal edge |
| repo | tools |
| issue file | workshop/issues/000081-define-editor-diagnostics-are-clipped-at-the-terminal-edge.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1236de2a5e303b3626d8f6bbded85d845bee6752..42dfcfb67aca98e39916ce34204ec9c183ab2cb3 |
| command | sdlc close --issue 81 |
| reviewer | claude |
| timestamp | 2026-09-23T17:18:09-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Both Important findings from round 1 are fixed, and I confirmed each fix by reverting it in a scratch copy. `wrapDiagnostic` (`cmd/define/screen.go:962-977`) first uses the existing `wrapWritten` word wrap, then hard-breaks any line that still doesn't fit using the screen's own soft-wrap splitter, `selectionPhysicalRows`. With the hard-break loop removed, `TestDiagnosticsFitTheScreenAndLoseNothing` fails: the overlong line exceeds 40 cells in both the plain and coloured cases. The new test is pure and runs under plain `go test`, so Done-when clause 2 no longer depends on a conformance-tagged pty test. One leftover remains. BR-3's premise was wrong: the buffer's own `Write` already turns CRLF into LF and drops any bare `\r` (`screen.go:127`, `:160`). That makes the new CRLF normalisation redundant, and the comment that justifies it is false. This is Minor and doesn't block.

1. **Strengths**
   - The hard break reuses `selectionPhysicalRows` (`screen.go:715`) instead of building a new splitter, so styling carries over onto continuation rows. The coloured case of the test exercises that path (ARCH-DRY).
   - The erase gesture exemption and the sub-20-column policy match `wrapWritten` exactly (`screen.go:966-970`), so this path follows the same rules as the pinned-screen wrap.
   - The test checks three things together: every row fits, no text is lost (joined with spaces removed), and both plain and coloured styling. That catches both clipping and dropped fragments.
   - The claim that wrapping twice on a pinned screen is harmless holds. Rows from the hard break already fit, so `wrapWritten` in `writeBuffer` leaves them unchanged.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **The CRLF normalisation in `wrapDiagnostic` repeats what `screen.Write` already does, and its comment is false.** This is the 2nd finding in family `wrap-drops-carriage-return`.
     - **The rule:** `\r` handling belongs to the buffer boundary, which is `screen.Write`. No writer upstream of it should normalise line endings.
     - **Evidence:** I removed the `ReplaceAll` in a scratch copy. The test stayed green, and a fitting `"short\r\n"` still reached the buffer as `"short"`.
     - **Every instance in the window:**
       - the `ReplaceAll` at `screen.go:963`
       - the comment paragraph at `:959-961`, which claims "a line that fits keeps it"
       - the test's "kept a CR" assertion at `screen_test.go:1720`, which cannot fail.
     - **Fix:** delete all three, or keep the assertion only as a documented invariant of the buffer rather than of this fix.
   - A partial write, meaning text with no trailing newline followed by another write, is wrapped one piece at a time. I measured a 43-cell row in a 40-column screen. Today every editor stderr call site writes a whole line (`replraw.go:28,395,663,798`, `main.go:1307`), so this is only a premise the doc comment states. Noted from round 1; no action needed.
   - Rows produced by the hard break start at column 0 and don't keep the line's hanging indent. That's acceptable for diagnostics.

5. **Test coverage notes:**
   - `go test -run TestDiagnostics ./cmd/define` passes.
   - The hard-break mutation turns it red, which confirms the fix for BR-1 and BR-2.
   - The CR mutation stays green (see Minor).
   - I did not re-run the pty conformance test, because it needs an unsandboxed pty.

6. **Architectural notes**
   - ARCH-DRY: pass for the wrap and the hard break. Minor flag for the CRLF normalisation, which duplicates `screen.Write:127`.
   - ARCH-PURE: pass. `wrapDiagnostic` is a pure function, and `diagnostics.Write` is a thin wrapper around it that takes the lock.
   - ARCH-PURPOSE: pass. Tokens wider than the screen, the case the operator actually hit, now reach the screen whole.

7. **Plan revision recommendations:** None needed. The existing Revisions entry already records the hard-break policy and the in-process test. If the CRLF normalisation is removed, the Log line "Minor CR finding folded in" should say BR-3 was withdrawn, because the buffer already drops `\r`.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      wrapDiagnostic hard-breaks via selectionPhysicalRows; removing that loop turns TestDiagnosticsFitTheScreenAndLoseNothing red (2 over-width rows).
  - id: BR-2
    disposition: addressed
    note: |
      Pure in-process TestDiagnosticsFitTheScreenAndLoseNothing asserts every row fits 40 cols plus no text lost, plain and coloured, overlong token included.
  - id: BR-3
    disposition: withdrawn
    note: |
      Mistaken premise: screen.Write (screen.go:127,160) already turns CRLF into LF and drops bare CR, so a fitting line never kept its CR in the buffer.
findings:
  - id: new
    severity: Minor
    family: wrap-drops-carriage-return
    title: |
      CRLF normalisation in wrapDiagnostic duplicates screen.Write and its comment claim is false
    detail: |
      2nd finding in the family. Rule: CR handling belongs to the buffer boundary (screen.Write:127,160); no upstream writer should normalise. Instances in the window: the ReplaceAll at screen.go:963, the comment at screen.go:959-961 ("a line that fits keeps it"), and the CR assertion at screen_test.go:1720, which stays green with the normalisation removed. Delete all three, or keep the assertion only as a documented invariant of the buffer.
```

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

---
id: 000081
status: working
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T16:58:08-07:00
flow: {kind: quick, provenance: inferred, spec: "da43ce69", done: "253f1ad1"}
---

# define: editor diagnostics are clipped at the terminal edge

## Problem

Operator, 2026-09-23, after a failed `?question`:

> define: llm: bad request: POST "http://127.0.0.1:8317/v1/messages": 400 Bad Request {"type":"er
>
> the error message needs line wrap, otherwise I can't see the details

The interactive editor routes stderr into the live screen (`replraw.go`, D5b),
and `Paint` clips every buffer line at the terminal's width (`screen.go`,
`clipVisible(painted, s.cols)`). The PINNED screen (a sitting) wraps in
`writeBuffer`; the editor's screen does not, on the premise that its writers
pre-wrap — true for `Render` and `answerWrapWriter`, false for every
`fmt.Fprintf(stderr, …)` diagnostic. So the part of an error that says what
went wrong — the provider's message — is exactly the part cut off.

## Spec

The editor's stderr is a writer that wraps at the screen's current width
before the text reaches the buffer — `wrapWritten`, the one wrap the pinned
screen already uses (ARCH-DRY), applied to the diagnostics stream only.

stdout stays unwrapped: a streamed answer arrives in chunks and
`answerWrapWriter` carries word boundaries across them; wrapping per chunk
would break words at chunk edges. Diagnostics are written as whole messages
(`Fprintf` per line), which is the premise the pinned screen already rests on
("A sitting writes whole messages").

## Done when

- A pty test at 80 columns scripts a 400 from the model fake and asks a
  question; the provider's message (the tail past column 80) is on screen.
  Red before the fix.
- Every buffer row written through the editor's stderr fits the screen width.

## Plan

- [x] pty test, red
- [x] `liveScreen.Diagnostics()` writer wrapping with `wrapWritten` at `l.cols`;
      `replraw.go` wires `stderr` to it
- [x] green; `go test ./cmd/define/...`; `sdlc close`

## Log

### 2026-09-23

Filed from the operator's report while diagnosing the proxy 400 (the proxy
advertises `claude-opus-5-5` and then refuses it; separate matter).

Red then green: `TestPTYAWideDiagnosticIsWrappedNotClipped` (conformance tag,
needs a real pty — the sandbox refuses one, so run unsandboxed) showed the line
cut at `400 Bad Req` at 80 columns before the fix, and the full provider
message after. Fix is `liveScreen.Diagnostics()` reusing `wrapWritten`
(ARCH-DRY), wired as the editor's stderr in `replraw.go`.

Pre-existing, not this issue: 13 other `TestPTY*` rows fail identically with
this change stashed. The failures read as the first-run "not a deck yet.
Create one here?" prompt eating the tests' first line of input (the new test
had to decline it explicitly). Worth its own issue.

Close review round 1 (FIX-THEN-SHIP): BR-1 a token wider than the terminal
was still clipped — `wrapDiagnostic` now hard-breaks what the word wrap
cannot fit, via the screen's own `selectionPhysicalRows`; BR-2 no in-process
width test — `TestDiagnosticsFitTheScreenAndLoseNothing` (plain + coloured,
overlong token, no CR kept, no text lost), mutation-checked red against the
bare `wrapWritten`. Minor CR finding folded in (CRLF normalised first).

## Revisions

- 2026-09-23 — close review: diagnostics HARD-BREAK tokens wider than the
  screen (unlike `wrapText`'s keep-words-whole policy for definitions), and
  Done-when 2 is pinned by an in-process test rather than the pty row alone.

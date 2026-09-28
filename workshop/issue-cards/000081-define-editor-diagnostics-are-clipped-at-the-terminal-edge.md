---
id: '000081'
status: done
started: 2026-09-23T16:58:08-07:00
created: 2026-09-23
updated: 2026-09-27
actual_hours: 0.36
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

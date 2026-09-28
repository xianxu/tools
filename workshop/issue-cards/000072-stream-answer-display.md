---
id: '000072'
status: done
started: 2026-09-17T14:26:02-07:00
created: 2026-09-16
updated: 2026-09-17
estimate_hours: 2.47
actual_hours: 1.92
---

# define: stream the model's answer to the screen as it arrives

## Problem

The operator's observation: the spinner stops, there is a pause, and then the
whole answer appears at once. That reads as non-streaming because it **is** —
the transport streams and the display does not.

Measured 2026-09-17:

| where | what arrives when |
|---|---|
| the wire (cliproxyapi → `claude-opus-5`, effort high) | first `text_delta` at **1.25 s**, then 126 deltas ~60 ms apart over ~10 s |
| `define "?What is the difference between sycophantic and obsequious…"`, piped | the entire 1422-byte answer in **one write at 9.4 s**, nothing before it |

**Cause.** `sharedLanguageGrammar` (`askctx.go:173`) asks the model to wrap its
prose in `[lang=xx]…[/lang]`, and `language_decode.go` holds an open segment's
text in `d.body`, emitting nothing until the close marker. Isolated: annotated
deltas emit **0** until `[/lang]`, while untagged prose emits per rune. An answer
is normally one passage, so the whole answer is held for the whole generation.

**Why it holds.** Ownership is revocable today — a nested, malformed, over-limit
or unterminated segment flushes as *neutral* (`atlas/define.md`: "malformed/
nested/incomplete segments preserve neutral prose"). Text cannot be un-painted
once it is on screen, so the buffer is the price of that revocation. The whole
cost of the feature is paid on every well-formed answer to preserve a fallback
that fires on malformed ones.

**Why no test caught it.** Every streaming test asserts the answer's final
*content*, and a ten-second buffer produces byte-identical final content. The
missing observable is *when* text reaches the sink relative to the end of the
stream — `workshop/lessons.md`, "Test helpers that hide the bug".

**A second, smaller hold, deliberately out of scope.**
`ownedAnswerWrapWriter.finalize` (`answerwrap.go:333`) commits one physical row
at a time, because a row's background tint is computed from whole-row ownership.
Measured by replaying the recorded deltas at their real 60 ms spacing through the
production chain at width 100: first row at **0.3 s**, rows ~**0.5 s** apart, 18
writes over 7.2 s. That reads as streaming, so the row stays the commit unit and
this issue does not touch the wrapper.

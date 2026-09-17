---
id: 000072
status: working
deps: []
github_issue:
created: 2026-09-16
updated: 2026-09-17
estimate_hours:
started: 2026-09-17T14:26:02-07:00
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

## Spec

Operator decision, 2026-09-17: **stream now, and decide the tint's fate in #64.**
Streaming and the annotation turned out to be separable — the fix below needs no
prompt change and deletes none of #65/#66.

### Ownership is decided when a passage OPENS, not when it closes

`stepLanguageDecode` stays the single source of transitions (ARCH-ORDER); what
changes is its effect table. `decodeAppend` emits the text immediately, owned by
the language the opening marker announced, instead of accumulating it;
`decodeFlushOwned` and `decodeFlushNeutral` then have nothing left to flush.

**Accepted consequence, stated so it is not mistaken for an oversight:** a
segment that nests, carries a bad header, exceeds a bound or never closes keeps
the language it announced rather than degrading to neutral. Already-painted text
cannot be revoked, and the alternative is a ten-second blank on every well-formed
answer — a rare cosmetic wrong tint against a certain, universal cost. Recovery
still suppresses ownership for text arriving *after* the malformed event; only
text already emitted keeps its announced language.

`languageBodyLimit` (16 KiB) goes with the buffer. Nothing accumulates, so the
bound is satisfied by construction rather than by a check (ARCH-FUNERAL: the
decoder then creates no growing structure at all; the 64-byte incomplete-marker
candidate is the only hold left, and it is already bounded).

### Operating envelope (ARCH-CONSTRAINTS)

Interaction path: streamed UI response, one answer at a time, no concurrency.

- first text visible ≤ 0.5 s after the model's first text delta — measured basis 0.3 s at width 100
- display cadence ~0.5 s per row at width 100 with ~60 ms deltas — measured
- decoder hold ≤ 64 bytes (an incomplete marker candidate) plus one partial rune
- the segment-body bound disappears; there is no longer a quantity that can exceed one

### What #64 inherits

Whether an answer should be tinted *at all* before a reply-language policy
exists. `renderAskPrompt` names a study language but never requests a reply
language, and `/bilingual` never reaches the ask path — the request is
byte-identical with it on and off. So the answer tint colours a language the
model self-reports and nothing asked for, unlike the definition tint, which comes
from verified `dictionarySourceLanguage` metadata known before a byte is painted.
That question belongs with #64's stage model, not here.

## Done when

- A test asserts text reaches the sink **before the stream ends**, on annotated
  input, through the production chain — the observable no existing test had.
- The piped one-shot writes its first bytes seconds before the answer completes,
  measured and recorded in `## Log` as a before/after.
- A nested, malformed or unterminated segment keeps its announced language, with
  the termination-row assertions updated to state that rather than neutral.
- `d.body` and `languageBodyLimit` are gone, not merely unused.
- Ctrl-C mid-answer still keeps what arrived, highlighted as before.
- `atlas/define.md` states the streaming behaviour and replaces the sentence
  "malformed/nested/incomplete segments preserve neutral prose".
- #64 records the inherited tint question.

## Plan

- [ ] failing test first: emission before the close marker, through the production chain
- [ ] own-at-open in `stepLanguageDecode`'s effects; delete `d.body` and `languageBodyLimit`
- [ ] update the malformed/nested/unterminated assertions to announced-language
- [ ] re-measure the piped one-shot; record before/after in `## Log`
- [ ] atlas, the #64 note, then `sdlc close`

## Log

### 2026-09-17

Claimed before brainstorming (#113). The title's literal ask looked already
shipped — `runAsk` calls `client.Stream` and four tests defend streaming — so the
first move was to measure rather than to trust either the code reading or the
report. The wire probe and the piped one-shot together located the hold in our
own pipeline, and a scratch test on the decoder alone made it exact: 0 emissions
until `[/lang]`.

Two false starts worth keeping. Dropping the `[lang=xx]` annotation from the ask
prompt was proposed and accepted, then withdrawn when the blast radius turned out
to reach four cassette-replaying test files and, transitively, the whole answer-
side ownership pipeline — most of #65/#66. Measuring the row cadence is what
dissolved the dilemma: at ~0.5 s per row the annotation costs nothing visible
once the passage buffer is gone, so streaming needed no prompt change at all.

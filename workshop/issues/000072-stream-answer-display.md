---
id: 000072
status: open
deps: []
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
---

# define: stream the model's answer to the screen as it arrives

## Problem

Reported from use: the spinner stops — so the answer has started arriving — and
then there is a visible pause before any text appears. The reader is looking at a
still screen during the part of the wait that streaming exists to remove.

The transport is not the problem. `/ask` already streams:
`client.Stream(ctx, req, answer.decoder.Write)` (`ask.go:174`), over
`anthropicClient.Stream`'s real SSE loop (`internal/llm/anthropic.go:144`). The
deltas arrive incrementally. Something between the delta and the cell holds them.

**The prime suspect, to be confirmed by measurement rather than asserted.**
`askctx.go:168` instructs the model to annotate its answer:
`[lang=es]Spanish text[/lang] and [lang=en]English text[/lang]`. The decoder that
reads those tags buffers: `stepLanguageDecode` maps `decodeText` in
`decodeSegment` to `decodeAppend` (`language_decode.go:47`), accumulating the
body up to `languageBodyLimit` (16 KiB) and emitting only on `decodeClose`
(`decodeFlushOwned`). A segment is whatever the model chose to wrap — routinely a
whole sentence or paragraph. For the length of that segment the screen gets
nothing.

Two other candidates, both cheap to rule in or out in the same measurement:

- The spinner stops on the first NON-EMPTY delta (`llm_activity.go:49`) without
  asking whether that delta produced a visible cell. The first delta is often
  `[lang=en]` — a header that decodes to no text at all. So the spinner can
  legitimately stop while there is, by construction, nothing yet to show.
- `liveScreen`'s redraw cadence between `wrap` and the terminal.

Two mechanisms that are already bounded and are NOT suspects, recorded so the
investigation does not re-derive them: `highlightWriter` holds only the tail that
could still change — a token that may grow another letter, plus up to
`maxWords-1` tokens for a phrase (`highlightwriter.go:175`) — and
`answerWrapWriter` holds one unfinished word. Both are word-scale.

**The insight worth checking first:** the language tint needs the language, and
the language is known at the OPENING tag, not the closing one. If that holds,
the segment buffer is lookahead no style actually pays for.

## Spec

**The requirement, stated as a bound.** Text reaches the screen within one
bounded hold of arriving, and the bound is **one word** — the phrase lookahead
`highlightWriter` already justifies — never a segment, a sentence, a paragraph,
or a whole answer. Every style define applies must be shown to fit inside that
window, or its lookahead named and justified as an exception.

**Every style survives. This is not "dump the text raw".** The styles and what
each one actually needs to know before it can draw:

| style | needs | available after |
| --- | --- | --- |
| deck highlighting (`knownOn`) | the complete word, and up to `maxWords-1` more for a phrase | the word — already correct, `decidedEnd` |
| language tint | the segment's language | the OPENING `[lang=xx]` tag |
| wrapping | the word's display width | the word |
| pronunciation / clickable regions | the line, since a `Region` is registered per rendered line | the line ends |
| markdown (#71) | the closing delimiter of an inline span | see below |

So the target is: tint from the open tag, highlight and wrap per word, regions
registered as each line completes. The decoder hands `accept` a complete
`languageText` with spans today; the change is to let it hand over a segment
INCREMENTALLY — the language is already decided, so the spans it would have
reported are knowable as it goes.

**Regions are the one genuine line-granularity constraint** (ARCH-CONSTRAINTS —
name the envelope). A clickable word is registered against a finished line, so
clickability arrives at end of line even when the text arrived mid-line. That is
acceptable and should be stated rather than discovered: the reader is reading,
not clicking, the line being written. What is NOT acceptable is the line's TEXT
waiting for it.

**Optimistic drawing is allowed only where it can be taken back.** `lessons.md`'s
"Ephemeral UI vs. a record": `screen.lines` is immutable with colour baked in at
write time (`passage.go:17`), so a committed line can never be restyled — but the
line still being written has not been committed. If a style genuinely cannot be
decided within a word, drawing unstyled and repainting the current line is
available; repainting scrollback is not. Any such repaint must be invisible to a
pipe, which is a record and must never contain a correction.

**The `-no-color`, piped and `-raw` paths must not regress.** Output there is
byte-identical to today's, byte-for-byte, and the streaming change is invisible
in it — the same answer arriving in different chunk splits produces identical
bytes.

**Interruption keeps working.** `runAsk` keys several outcomes on
`answer.plain.Len() > 0` — Ctrl-C mid-stream records the partial exchange, and
#67's marks clear iff an answer reached the reader (`ask.go:190`). Releasing text
earlier changes WHEN `plain` becomes non-empty, so those predicates need
re-checking, not just the display path. `TestEditorCtrlCMidStreamReturnsToThePrompt`
and the truncation tests are the existing guard.

**Measure first, and measure at the right width** (`lessons.md`: a claim must not
outrun the width it was measured at). The first plan step is instrumentation, not
a fix: for one real answer, the timestamp of each delta, of each byte reaching
the terminal, and of the spinner's stop. That distinguishes the three candidates
above and gives the before/after number this issue is judged on. A fix shipped
against an unmeasured cause is a guess that happened to compile.

## Done when

- A trace of one real answer shows the interval from first delta to first visible
  cell, before and after, and the after is within one word of the first delta
  that carries text.
- A test asserts the BOUND, not the behaviour's shape: a fake stream delivering a
  long `[lang=en]…[/lang]` segment one word at a time produces output after each
  word, driven through a writer that timestamps or counts flushes. This must fail
  against today's code — a streaming test that passes before the change is
  testing the fake (`lessons.md`: before ticking a box, delete the fix and watch
  it fail).
- Deck words inside a streamed segment are still green, and a phrase (`hot dog`)
  split across two deltas still matches as a phrase.
- The language tint is applied to a segment's text from its first word, with no
  segment-length buffer.
- Clickable pronunciation regions still register for every deck word in the
  answer; the line at which they become clickable is stated in the atlas.
- Feeding one answer in every chunk split yields byte-identical output, and the
  piped/`-no-color`/`-raw` bytes are unchanged from today.
- Ctrl-C mid-stream still records the partial exchange, and #67's marks still
  clear iff text reached the reader.
- `atlas/define.md` records the hold bound and the per-style lookahead table.

## Plan

- [ ] instrument: delta arrival, spinner stop, first visible cell, on one real
      answer — confirm or kill the segment-buffer suspect before designing
- [ ] brainstorm the result; decide whether the spinner should stop on the first
      VISIBLE cell rather than the first delta
- [ ] `sdlc start-plan`, then the durable plan in `workshop/plans/`
- [ ] the decoder emits a segment incrementally; the failing bound test first
- [ ] the per-style lookahead table verified style by style — tint, highlight,
      wrap, regions
- [ ] the piped/`-raw`/`-no-color` byte-identity guard, and the interruption
      predicates re-checked
- [ ] atlas, then `sdlc close`

## Log

### 2026-09-16

Filed from a use report: spinner stops, then a visible pause before text. Read
the path far enough to name a prime suspect — `stepLanguageDecode`'s
`decodeAppend` accumulating a `[lang=…]` segment to its closing tag — and to rule
`highlightWriter`/`answerWrapWriter` out as word-scale. Not measured yet; step
one of the plan is the measurement, deliberately, so the fix is not aimed at a
guess.

Cross-reference: #71's markdown renderer introduces another lookahead-dependent
style (an inline span whose closer has not arrived). The hold bound this issue
establishes is the constraint that renderer has to obey, and #71's streaming
requirement should be read against it.

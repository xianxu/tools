---
id: 000072
status: working
deps: []
github_issue:
created: 2026-09-16
updated: 2026-09-17
estimate_hours: 2.47
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
changes is its effect table, and the resulting enumeration is the deliverable:

| state | `text` | `open` | `close` | `badHeader` | `finish` |
|---|---|---|---|---|---|
| `neutral` | `emitNeutral`, stay | → `segment` | → `neutral` | → `recovery` | → `neutral` |
| `segment` | **`emitOwned`**, stay | → `recovery` | → `neutral` | → `recovery` | → `neutral` |
| `recovery` | `emitNeutral`, stay | stay | → `neutral` | stay | → `neutral` |

Three deletions fall out of it. `decodeAppend` becomes `decodeEmitOwned`.
`decodeFlushOwned` and `decodeFlushNeutral` go, because nothing is buffered to
flush. And **`decodeLimit` goes as an EVENT** — its sole producer was the
`decodeAppend` arm (`language_decode.go:141`), so the `ok=false` rejection rule at
`:42-44` disappears with it and `stepLanguageDecode` becomes total.

`d.lang` is cleared today only inside the two flush effects (`:149`, `:153`).
With those gone it gets ONE rule in `event()`: cleared on every transition that
leaves `decodeSegment`. One site, so the three ways out (close, nested open, bad
header, finish) cannot drift apart.

**Accepted consequence, stated so it is not mistaken for an oversight:** a
segment that nests, carries a bad header or never closes keeps the language it
announced rather than degrading to neutral. Already-painted text cannot be
revoked, and the alternative is a ten-second blank on every well-formed answer —
a rare cosmetic wrong tint against a certain, universal cost. Recovery still
suppresses ownership for text arriving *after* the malformed event; only text
already emitted keeps its announced language. The fourth case, *exceeds a bound*,
ceases to exist rather than changing behaviour: with nothing accumulating there is
no bound to exceed.

### The owned path has to become a streaming path too (ARCH-DRY)

`answer_language.go:42` renders an owned span with `highlightRegion`, which
builds a **fresh** `newHighlightWriter` per call and flushes it
(`highlightwriter.go:269-282`). That works today only because an owned span
arrives as one complete buffered passage. Under own-at-open each owned emission
is a rune, so `obsequious` could never match and
`TestLanguageAnswerForeignHomographDoesNotUseTargetVocabulary` would drop from 1
highlight to 0.

So the owned path gets the same persistent `highlightWriter` the neutral path
already uses. `languageAnswer` holds the current ownership; its sink tags writes
with it; and at an ownership CHANGE the highlighter is flushed and replaced with
one carrying that run's vocabulary (nil when the span's language is not the
session's, per the existing `:37-41` rule). Flush-then-replace is what preserves
the invariant the homograph test exists for: **a word may not highlight across an
ownership boundary.** This retires the one-shot `highlightRegion` call from the
answer path, leaving one mechanism where there were two — the atlas already
claims `highlightWriter` is why definitions and answers are one mechanism, and
the owned span was the exception to it.

### Operating envelope (ARCH-CONSTRAINTS)

Interaction path: streamed UI response, one answer at a time, no concurrency.

- first text visible ≤ 0.5 s after the model's first text delta — measured basis 0.3 s at width 100
- display cadence ~0.5 s per row at width 100 with ~60 ms deltas — measured
- decoder retention, every component of it, once `d.body` is gone:
  `d.marker` ≤ 64 B (`languageHeaderLimit`), `d.entity` ≤ 64 B (`:241`),
  `d.filter.pending` and `d.literal.pending` ≤ 3 B each — **two** filters, one
  partial rune each (`answer_text.go:30-32`) — and `d.literalText` ≤ one emitted
  rune. Total ≤ 200 B, bounded by the marker and entity grammars rather than by a
  size check.
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
- A deck word split across deltas **inside an owned passage** still highlights,
  and still does not highlight across an ownership boundary — the owned path
  proven to stream, not just the decoder.
- `d.body`, `languageBodyLimit` and the `decodeLimit` event are gone, not merely
  unused, and `stepLanguageDecode` is total.
- The retention bound is asserted as an invariant over every component named in
  the envelope, replacing the `d.body.Len()` guard that dies with the field.
- Ctrl-C mid-answer still keeps what arrived, highlighted as before.
- `atlas/define.md` states the streaming behaviour and replaces the sentence
  "malformed/nested/incomplete segments preserve neutral prose".
- #64 records the inherited tint question.

## Plan

- [ ] failing test first, pure and deterministic (ARCH-PURE): `languageDecoder`
      emits owned text BEFORE the close marker — the observable no existing test
      had. No clock, no socket.
- [ ] own-at-open in `stepLanguageDecode`: `decodeEmitOwned`, drop the two flush
      effects and the `decodeLimit` event, one `d.lang` clearing rule. Update
      `TestLanguageDecodeTransitions`' independent matrices to the table above —
      it stays independently stated, not read off the implementation.
- [ ] `languageDecoder.lex` is the rune scanner over untrusted model output
      (ARCH-SECURE): extend `FuzzLanguageDecoderChunks` to assert the retention
      invariant after **every** chunk, seeded with own-at-open forms. Replace
      `TestAnswerControlPayloadAndAnnotationMemoryAreBounded`'s `d.body.Len()`
      check with that invariant over `marker`/`entity`/both filters/`literalText`.
- [ ] `languageAnswer.accept`: the persistent per-ownership-run highlighter.
      Tests — the homograph row unchanged (it is the boundary invariant), plus a
      new row for a deck word split across deltas inside an owned passage,
      mirroring `TestStreamedAnswerHighlightsAWordSplitAcrossDeltas` on the
      neutral path. The enumeration is {neutral, owned} × {split across deltas}.
- [ ] fixture rows whose expectations change with ownership-at-open:
      `TestLanguageDecoderRecoveryAndSplits` nested (`red` becomes owned),
      unterminated (`unfinished` becomes owned) and the 16385-byte row (no longer
      an over-limit case). Each is a behaviour change stated in the Spec, not a
      test bent to fit.
- [ ] end-to-end, the level the operator saw it at: record a LONG-PASSAGE capture
      via `scripts/llm-probe.sh record`. The committed `stream-language.sse`
      closes its passages after ~6 of its 87 deltas, so it cannot exhibit this
      bug at all — `llmtest/testdata/README.md`'s own rule, a capture is evidence
      only for the shape its recording conditions elicit. Drive it through
      `Reply{AfterText, FinishRelease}` and assert the sink holds text while the
      stream is still open (ARCH-MOCK: the barrier is the seam, no wall clock in
      the assertion).
- [ ] re-measure the piped one-shot; record before/after in `## Log`
- [ ] atlas — the streaming behaviour, the replaced "malformed/nested/incomplete
      segments preserve neutral prose" sentence, and the retired `highlightRegion`
      exception — plus the #64 note, then `sdlc close`

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec             design=1.0  impl=0.1
item: smaller-go-module      design=0.2  impl=0.15
item: smaller-go-module      design=0.2  impl=0.2
item: real-api-discovery     design=0.0  impl=0.15
item: atlas-docs             design=0.05 impl=0.05
item: milestone-review       design=0.0  impl=0.15
design-buffer: 0.15
total: 2.47
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Derivation notes, so the numbers can be argued with rather than just checked:

- **issue-spec design=1.0** is the top of the 0.5–1.5 band and earns it: the
  brainstorm required a wire probe, a piped measurement, an isolated decoder
  test and a delta-replay cadence measurement, plus two reversals of direction
  (drop-the-annotation proposed, accepted, then withdrawn on blast radius) and a
  plan-quality round. This is the primitive the hours actually went into.
- **two `smaller-go-module` rows, not one.** The decoder change and the
  `answer_language.go` highlighter are separate concerns with separate failure
  modes — the second is the one the gate caught, so collapsing them into one row
  would hide exactly the risk PQ-1 surfaced. Both are "extend, well-specced"
  (design 0–0.3), which is what a gate-cleared plan buys.
- **real-api-discovery impl=0.15** is the long-passage capture: a live recording
  through `scripts/llm-probe.sh record`, stage-verify-promote, against a service
  whose passage shapes are not ours to choose.
- **design-buffer 0.15**, not 0.30: the plan is thorough and itemized to the
  function level and cleared plan-quality, which is the condition the buffer rule
  is about — though it lives in the issue rather than a separate
  `workshop/plans/` doc, so this is the judgment call most open to challenge.
- **familiarity 1.0**, the default. The subsystem is now read closely, which
  would argue for less, but a multiplier claimed on one session's reading is not
  calibration.

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

## Revisions

### 2026-09-17 — plan-quality round 1 (4 blocking findings)

Reason: `sdlc change-code`'s gate found a Critical the brainstorm missed, and
three places where the plan asserted less than it needed to.

- **PQ-1 (Critical), own-at-open breaks owned-path highlighting.** Verified
  against the tree rather than taken on trust: `highlightRegion` builds a fresh
  writer per call, so per-rune owned emissions match nothing. The Spec gains a
  section and the Plan a step; the fix turns out to REMOVE a mechanism rather
  than add one, which is why it reads as an ARCH-DRY win.
- **PQ-2**, the post-change state × event → effect table is now written out,
  including the two effects and the one EVENT that disappear, and the single new
  `d.lang` clearing rule.
- **PQ-3**, the envelope's retention figure was wrong — it omitted `d.entity` and
  counted one partial rune where there are two filters. Corrected before the
  assertion that will use it as its oracle is written.
- **PQ-4**, the plan now names `stepLanguageDecode`, `languageDecoder.lex` and
  `languageAnswer.accept` with a strategy line each, says what replaces the
  `d.body.Len()` memory guard, and disposes of the three fixture rows whose
  expectations change.
- **PQ-5 (Minor)**, the timing budget gets a mechanical guard: the `AfterText`/
  `FinishRelease` barrier asserts ORDERING (text on screen while the stream is
  open) rather than a wall-clock duration, which is deterministic where a timing
  assertion would be flaky. The 0.5 s figure stays a measured `## Log` fact.

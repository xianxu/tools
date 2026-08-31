---
id: 000040
status: open
deps: ["tools#39", "tools#41"]
github_issue:
created: 2026-08-31
updated: 2026-08-31
estimate_hours:
---

# form 2.5: the board — grid triage for mature words

## Problem

The ladder's cost is dominated by MATURE words. Daily load is
`Σ 1/IntervalDays(box)` over the deck, and with `#39`'s unbounded ladder most of
a grown deck sits in the long tail — hundreds of words each costing a few reviews
a year, which adds up to most of the day's work.

One-at-a-time multiple choice costs the same per word whatever its box. So a deck
large enough to be worth having becomes a deck too expensive to maintain, and the
learner's only lever is to stop adding words — which is the wrong lever.

Separately, `#39`'s ladder distinguishes a confident answer from a merely correct
one, and no form reports confidence for a word it did not test.

## Spec

**Form 2.5 is a grid: sixteen words at once, one keystroke each.** The learner
marks each word `firm` / `unsure` / `no idea`. It is triage, not practice, and it
exists to make a large deck affordable: a hundred mature words swept in a grid
cost what ten fragile ones cost in multiple choice.

**The SCHEDULER picks the form, not the learner's mood.**

| box | state | form |
|---|---|---|
| 0–3 | fragile, being acquired | 2.3 `/meaning` — real retrieval |
| 4–7 | consolidating | 2.3 |
| 8+ | mature | 2.5 the board |

This is the load argument made concrete, and it is also a correctness argument. A
grid is SELF-REPORT WITHOUT RETRIEVAL, and the illusion of knowing runs exactly
that direction — a familiar-looking word feels known. If the learner chooses the
form, they will choose the cheap one, and the whole ladder ends up driven by
overconfidence. Binding the form to the box means the cheap form is only used
where being wrong is cheap: a box-10 word marked `firm` in error costs one missed
retrieval on a word already recalled ten times.

`/board` stays available manually as an explicit "I am short on time today"
escape. It must not be the default path.

**The mapping to `#39`'s transitions is deliberately CONSERVATIVE:**

| mark | effect |
|---|---|
| `firm` | `box + 1` — the same as a correct answer, NOT `+2` |
| `unsure` | box unchanged, re-asked sooner, and next time through form 2.3 |
| `no idea` | `box / 2`, exactly as a wrong answer |

**`firm` earns `+1` and not `+2`, and the reason belongs in the code.** `#39`
reserves `+2` for a confident answer, and self-report is not that. The real
producer of confidence is already available and objective: `Apply` knows
`s.Revealed` at grading time, so *a correct answer in form 2.3 with no reveal
first* means the learner knew it cold. That is measured rather than claimed, and
it is where `+2` should come from. This issue should carry a Revision to `#39`
saying so.

**`unsure` promotes a word back to a real test.** It is the signal that triage
was the wrong instrument for that word, so the answer is to test it properly, not
to guess at a box change.

**Selection is a cursor over a grid**, which is why this depends on `#41`: a grid
cannot be drawn by appending lines.

## Done when

- [ ] A sitting containing mature words presents them as a grid, sixteen at a time.
- [ ] Every word in the grid can be marked, and the marks reach the event log with the same "recorded as it happens" guarantee a single answer has.
- [ ] The scheduler chooses 2.5 for box ≥ 8 and 2.3 below, pinned by a test over a deck spanning both.
- [ ] `unsure` re-asks the word through form 2.3 rather than changing its box.
- [ ] `/board` forces the form for a sitting; it is not the default.
- [ ] The `Question` interface is unchanged, or the change is form-agnostic — the session still learns nothing about which form is asking (`#6`'s Done-when, `TestSessionIsFormAgnostic`).
- [ ] Measured: a sitting of N mature words through the board takes materially fewer keystrokes than the same N through form 2.3.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-31

Filed from a design conversation. The operator proposed the 4x4 board as "a
faster way to glimpse through today's recall work"; the analysis that turned it
into a scheduler decision rather than a user preference is in the Spec — the
grid is the maintenance form, and binding it to the box is what keeps the
illusion of knowing from driving the ladder.

`/synonym` and `/acronym` were discussed and are DEFERRED to their own issues.
`/synonym` has a real data source — `com.apple.dictionary.OAWT` is installed and
active, and `dictselect.go` already names it while deliberately filtering
thesauruses out of the curated general-dictionary list — so that issue starts
with a measurement of OAWT's output shape, not a design.

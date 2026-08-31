---
id: 000041
status: working
deps: ["tools#39"]
github_issue:
created: 2026-08-31
updated: 2026-08-31
estimate_hours: 3.31
started: 2026-08-31T13:26:27-07:00
---

# play mode paints frames through screen, and gains a status bar

## Problem

There are two interactive surfaces in this binary and they render two different
ways. The editor loop draws whole frames through `screen`/`liveScreen` (`#30`);
`--play` writes lines through `crlfWriter` to a scrolling terminal. `atlas/define.md`
records the consequence directly: *"`play_loop.go` writes through `crlfWriter` to
a scrolling terminal and therefore owns no coordinates. `#30` D5a predicted this
seam."*

Three things follow from owning no coordinates, and they are all user-visible:

- **A long entry scrolls the question away.** Form 2.3's reveal shows the whole
  rendered definition; on a word like `run` or `bank` that is far more than a
  screenful, and the word being asked about is gone off the top. `screen` already
  has a viewport and paging; `--play` cannot use them.
- **There is nowhere to put a status bar.** A learner in a sitting cannot see how
  many words are left, how many are due today, or what the day costs. That number
  exists once `#39` lands and has nowhere to go.
- **Every future mode inherits the limitation.** `#40`'s board is a grid with a
  cursor; a grid cannot be drawn by appending lines.

## Spec

**`--play` draws through the same seam the editor uses.** Not a new abstraction:
`screen` is a pure line buffer plus viewport with a `display` interface over it
(`Draw`, `Page`, `Scroll`, `Resize`), and `liveScreen` is the only part that
touches a terminal. Adopting it is what closes `#30` D5a's predicted divergence.

**A status bar, pinned to the bottom.** What it shows:

```
18 due · 7 answered · 11 left            ~14 reviews/day at your current mix
```

The right-hand number is `#39`'s load figure — `Σ 1/IntervalDays(box)` over the
deck. It is the number that should govern how many new words the learner takes
on, and today it is computed nowhere and shown nowhere. Hence the dependency.

**The transcript must survive exit, and that is already solved.** `#30` M1.2b is
*"Paint draws a whole frame, and the transcript survives the alt screen"* — a
sitting's words should still be on screen after quitting, not wiped by the
alt-screen restore. Reuse that behaviour rather than re-deciding it.

**Paging comes for free**, and is the point of the first bullet above: a reveal
longer than the viewport pages rather than pushing the question off the top.
`#30` already decodes the wheel and PageUp/PageDown into `Page`/`Scroll`.

**Not in scope:** any new question form. This issue changes how `--play` DRAWS
and nothing about what it asks. `#40` is the first consumer.

## Done when

- [x] `--play` paints whole frames through `display`; no path appends bare lines.
- [x] A status bar stays pinned at the bottom across question, reveal and resize.
- [x] A reveal longer than the terminal PAGES; the prompt word stays on screen.
- [x] The transcript survives exit, pinned by the same test shape `#30` used.
- [x] SIGWINCH repaints correctly mid-sitting.
- [x] A pty conformance test drives a real sitting and asserts the bar is present, updates as answers land, and is still there after a reveal.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.45 impl=0.08
item: cross-cutting-refactor   design=0.02 impl=0.12
item: smaller-go-module        design=0.03 impl=0.12
item: greenfield-go-module     design=0.06 impl=0.32
item: smaller-go-module        design=0.04 impl=0.16
item: greenfield-go-module     design=0.05 impl=0.28
item: cross-cutting-refactor   design=0.04 impl=0.24
item: smaller-go-module        design=0.03 impl=0.14
item: smaller-go-module        design=0.02 impl=0.10
item: smaller-go-module        design=0.02 impl=0.16
item: atlas-docs               design=0.03 impl=0.06
item: milestone-review         design=0.0  impl=0.30
item: milestone-review         design=0.0  impl=0.32
design-buffer: 0.15
total: 3.31
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* The calibration doc is tagged **stale** by
`sdlc estimate-source`, so the per-primitive hours are provisional.

| item | task | why this primitive |
|---|---|---|
| `issue-spec` 0.45/0.08 | the design carrier | the Spec, the plan, and FOUR plan-quality rounds — more than `#39`'s two, and the rounds found a Critical |
| `cross-cutting-refactor` 0.02/0.12 | T1 `menu` → `footer` | mechanical across `screen.go` and the editor's call sites; the suite is the proof |
| `smaller-go-module` 0.03/0.12 | T2 `sittingBar` | a pure formatter sharing `finish`'s wording |
| `greenfield-go-module` 0.06/0.32 | T3 the console, and DELETING the playback dance | the highest-risk task: it removes a `restore`/`enterRaw` pair and its error branch on the path a reveal takes |
| `smaller-go-module` 0.04/0.16 | T4 write the question once | the naive-port trap, one `int` of state |
| `greenfield-go-module` 0.05/0.28 | T5 the live edge, `newPinnedScreen`, padding | new paint behaviour that must not reach the editor |
| `cross-cutting-refactor` 0.04/0.24 | T6 figures in memory | `todaysQuestions`' signature widens, four test call sites move, `finish` stops reading |
| `smaller-go-module` 0.03/0.14 | T7 paging | intercepted in the loop; `toInput` untouched |
| `smaller-go-module` 0.02/0.10 | T8 SIGWINCH | the editor's case, mirrored |
| `smaller-go-module` 0.02/0.16 | T9a pty conformance | three existing pty tests re-examined against frames, plus a new bar assertion |
| `atlas-docs` 0.03/0.06 | T9b | atlas, README |
| `milestone-review` 0.0/0.30 | the boundary: run + manual verification | |
| `milestone-review` 0.0/0.32 | the boundary: remediation | top of the range, on `#7`'s eight-round evidence |

**THE SPECIFIC RISK, named rather than smoothed into a multiplier.** `#30` built
the machinery this issue adopts and measured **10.91h against 3.19h — 3.4×**,
the worst row in the ledger. That was greenfield terminal work and this is its
second consumer, so it should be cheaper by construction; but terminal work is
where this repo's estimates have been worst, and the reasons are visible in the
plan: a Critical the gate caught (the alternate screen surviving a reveal), a
padding behaviour that must not leak into the REPL, and three pty tests that
assert over a byte stream this issue changes. If any row here is going to be
wrong, it is T3 and T5.

**The calibration picture, unchanged and now three rows wide.** `#30` 3.4×, `#7`
1.70×, `#39` 0.54×. That is a WIDE SPREAD rather than a consistent bias — my
`#39` block predicted ~4.5-5h against a 2.94 estimate and the truth was 1.59, so
the prediction was further off than the model. No private correction is applied
here for the same reason as before: a per-issue fudge corrupts the ledger that
exists to measure the spread. `#127` should read the variance.

## Plan

Single-pass: plain checkboxes, ONE boundary (AGENTS.md §3). Full detail, the
decisions and the twelve Done-when rows live in
`workshop/plans/000041-play-tui-plan.md`.

- [x] Design via `sdlc start-plan` — plan doc written, cleared plan-quality in 4 rounds.
- [x] **T1** — `Paint`'s `menu` becomes `footer`, with `fitMenu`; one concept, two consumers (D2).
- [x] **T2** — `sittingBar` in `cmd/define/playbar.go`, sharing `finish`'s wording (D8).
- [x] **T3** — `--play` builds a `console`, and the reveal's `restore`/`enterRaw` pair is DELETED (D1, D5a).
- [x] **T4** — the question is written to the buffer once, not per keystroke (D4).
- [x] **T5** — the live edge: `newPinnedScreen` and paint-time padding (D3, D3a).
- [x] **T6** — figures in memory; `todaysQuestions` returns its work and `finish` stops re-reading (D7).
- [x] **T7** — paging intercepted in the LOOP; `toInput` untouched (D6).
- [x] **T8** — SIGWINCH repaints through the console.
- [x] **T9** — pty conformance (three existing tests re-examined) and docs.

## Log

### 2026-08-31

Filed from a design conversation. The operator's framing was "full TUI with a
bottom bar of today's coverage and progress"; the enabling change is narrower
than that, because `#30` already built the screen abstraction for the editor and
this is its second consumer.

## Log

### 2026-08-31 — T1 and T2 landed; resume at T3

**Done, both mutation-verified.** T1 renamed `Paint`'s `menu` to `footer`
(`fitMenu` → `fitFooter` with it) and rewrote the doc prose, since the concept
generalised rather than the spelling changing. T2 added `sittingBar` /
`sittingSummary` / `costPhrase` in `cmd/define/playbar.go`, and `finish` now
formats through the shared `sittingSummary` — giving them separate bodies turns
`TestTheBarAndTheSummaryAgree` red.

**Resume at T3, which is the risky one.** It is coupled to T4 and T5 and they
should land together, because none of them builds alone:

- **T3** — `runPlay` builds a `console` the way `replraw.go:38-70` does
  (`enterAlt`, `enterMouse`, a screen, `watchResize`, `onceHandBack`), and
  `playSession` takes it instead of `stdout`/`stderr`/`rawTerm`. **DELETE the
  reveal's `restore`/`enterRaw` pair and its error branch** (`play_loop.go:177`
  and `:185`) — that is D5a, and skipping it is what makes the alternate screen
  vanish on the first reveal.
- **T4** — track the written question index; write `Prompt()` on transition and
  `Reveal()` on `OutcomeReveal`, not per keystroke.
- **T5** — `newPinnedScreen`, and pad the buffer region at PAINT time.

Every `playSession` call site in the tests moves with T3's signature.

**One test is red ON PURPOSE and clears as T3–T6 land:**
`TestPlanTableStatusMatchesTheChangeWindow` names `draw`, `todaysQuestions` and
`playSession` as claimed-modified-but-untouched. Nothing else fails.

Branch: `000041-play-tui`.

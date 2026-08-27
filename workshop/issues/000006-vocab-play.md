---
id: 000006
status: working
deps: ["tools#3", "tools#5"]
github_issue:
created: 2026-08-20
updated: 2026-08-27
estimate_hours: 6.15
started: 2026-08-27T08:15:55-07:00
---

# define --play: review loop + form 2.1 quick pass

## Problem

The deck and the schedule are inert without a way to sit down and review.

## Spec

`define --play` runs today's queue.

- Form 2.1 (this issue): show the word, let the learner recall, reveal the
  definition, self-rate right/wrong. No question generation, no network.
- A `Question` interface every later form implements, so forms drop in without
  touching the loop.
- Terminal-shaped and single-threaded; interruptible at any point. Events are
  written **as they happen**, so Ctrl-C mid-session keeps what was reviewed —
  free from the append-only log (#3).
- Playing the pronunciation during review is on by default; `--no-audio` applies.
- A bad-question keypress records a flag against the question (groundwork for
  the generated forms, which need this feedback loop).

## Done when

- [x] A full session runs against fake store + fake clock, recording one event
      per answer.
- [x] A SKIP records nothing: it is not an assessment, and `Fold` would read a
      recorded skip as a miss and demote the word. Asserted on the SESSION (no
      record outcome is emitted), not on `CaptureReview` — whose `bool` cannot
      represent a skip precisely because one is never passed to it.
- [x] Audio plays before reveal by default, and `--no-audio` silences it.
- [x] `-count` bounds the session; it defaults to 20.
- [x] Every newline a session writes is a full CRLF — in raw mode a bare `\n`
      moves down without returning to column 0, which cascades a definition
      diagonally across the screen.
- [x] `d` removes the current word from the deck, keeping its events.
- [x] An empty queue prints a line and exits 0 — "nothing due today" is the
      expected state most days, not an error.
- [x] With NO DECK, `--play` prints one line and exits 0 rather than running —
      guarded on `deck == nil`, not on the env var, because `openStore`'s `Getwd`
      failure path also returns no deck and keying on the flag would hand a nil
      store to the queue builder and panic. (The first draft of this row said a
      session runs and records nothing under `DEFINE_NO_CAPTURE` — unsatisfiable,
      since that branch returns no deck at all.)
- [x] Interrupting mid-session preserves already-recorded events.
- [x] Adding a second form requires no change to the loop — asserted by driving the same table through a fake form with entirely different keys, and by checking that 2.1's own keys mean nothing to it.
- [x] **A full session runs with the LLM seam unavailable**, falling back to the
      forms that need neither key nor network (2.1 here, 2.3 in #7) rather than
      failing. Relocated from #11 on 2026-08-22: it names `--play`, so it belongs
      to the issue that owns `--play`. Asserted with a client returning
      `llm.ErrUnavailable`, not by unsetting an env var — the point is that the
      loop degrades, not that config resolution does.

## Plan

Durable plan: `workshop/plans/000006-vocab-play-plan.md` (two milestones; each
`Mx` row is its own review boundary).

- [~] M1 — `Question`, `Recall`, `Session`/`Apply`, the purity guards, the atlas
- [~] M2 — `CaptureReview`, `runPlay`, the `--play` flag, the README

**Both milestones' work is DONE; neither boundary closed on its own, and they are
closing together at the issue close.** `[~]` rather than `[x]` because a ticked
`Mx` claims a `Review-Verdict:` trailer and a `closed Mx` Log line, and neither
exists — which is exactly what BR-22 and BR-38 caught.

Why they collapsed: M1's close failed with five findings I never read (see the
Revisions), so M2 was built on an unclosed boundary. By the time that surfaced,
every later review window spanned both milestones, and re-running M1's close in
isolation kept finding issues in M2's code because the window no longer separated
them. Forcing two boundaries onto one continuous body of work was producing
rounds rather than review. The issue close covers the whole window, which is what
the reviews have effectively been doing for six rounds.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Derived after the plan cleared plan-quality (#187), one item per task plus the
process work inside the measured window. Design carries v2's ×0.2 spec-quality
discount on every code item — the plan pre-resolves the `Input` translation, the
single skip filter, the budget flag, the no-deck guard and the guard extraction.
Implementation is v3.1's 40% of the v2 table. Familiarity 1.0: `#5` just built
the sibling package and `#14`'s editor is the shape `Apply` copies.

**Round prices follow the split `#5` established and its outcome confirms.** #5
priced plan rounds at 0.22 and boundary rounds at 0.35, budgeted three per
boundary, and closed at 3.0h against 5.56 — over-estimated, but by less than the
two before it. FOUR plan rounds happened here (the gate cap was reached), so they
are counted as spent rather than guessed.

**The one real unknown is the terminal loop, and it is priced as such.** `runPlay`
is the first new raw-mode surface since `#16`, and `#16`'s streaming loop is the
row that overran worst in this repo's history. `#20` and `#21` both touched that
code and came in short, so the machinery is now well understood — but `runPlay`
creates a new FILE and a new loop, which is `greenfield-go-module`, not a
refactor of an existing one. That is where the honest uncertainty sits.

**Corrected at the estimate gate, and the correction is the point.** The first
draft raised boundary rounds to 0.30 impl *and* under-priced `runPlay` as a
refactor — two errors that partially cancelled to a plausible total. `#5`'s own
block wrote the rule this violates: *"A number that is right by accident teaches
the ledger nothing."* Rounds are back at `#5`'s measured `0.15/0.20`, `runPlay`
is greenfield, and a fifth plan round (this one) is counted. Every number is now
defensible on its own, which is what makes the resulting row usable evidence.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec             design=0.50 impl=0.08
item: milestone-review       design=0.10 impl=0.12
item: milestone-review       design=0.10 impl=0.12
item: milestone-review       design=0.10 impl=0.12
item: milestone-review       design=0.10 impl=0.12
item: milestone-review       design=0.10 impl=0.12
item: smaller-go-module      design=0.03 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: cross-cutting-refactor design=0.12 impl=0.14
item: atlas-docs             design=0.03 impl=0.05
item: milestone-review       design=0.15 impl=0.20
item: milestone-review       design=0.15 impl=0.20
item: milestone-review       design=0.15 impl=0.20
item: cross-cutting-refactor design=0.12 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: smaller-go-module      design=0.03 impl=0.14
item: atlas-docs             design=0.03 impl=0.05
item: milestone-review       design=0.15 impl=0.20
item: milestone-review       design=0.15 impl=0.20
item: milestone-review       design=0.15 impl=0.20
design-buffer: 0.15
total: 6.15
```

Item-to-task map. Process: spec+plan, then the four plan rounds (all spent — two
Criticals in round 1, and round 3 caught me stating the skip filter in two places
without choosing). **M1** — `smaller` = Task 1's `Question`/`Recall`;
`greenfield` = Task 2's `Session`/`Apply` with the `fakeForm` property;
`cross-cutting` = extracting `#5`'s purity guards into a shared helper touching
both packages; `atlas-docs` = M1's atlas; then three M1 boundary rounds. **M2** —
`cross-cutting` ×2 = Task 3's `CaptureReview` across every implementation and
double, and Task 4's `runPlay` with the raw-mode wiring; `smaller` = the `--play`
and `-count` flags; `atlas-docs` = M2's atlas plus the README; then three close
rounds.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

### 2026-08-27

Claimed and planned. Three decisions worth recording before implementation:

- **A skip is a third verdict, not a wrong answer.** Recording a skip as a miss
  would demote the word through `#5`'s `Answer`, so a learner who skips a word
  they half-know would be punished for honesty. It records NOTHING, which leaves
  `#8` unable to distinguish "skipped" from "never reviewed" — a deliberate hole,
  and the alternative (a fourth event kind) would make `Fold` learn to ignore
  something.
- **`Grade` lives on the form, not in the loop.** The Done-when says a second form
  must need no loop change, and that is a property of the interface rather than a
  promise — so key semantics stay inside the form (2.1's `y/n`, 2.3's `1/2/3/4`),
  and the loop only asks "did that key mean anything to you".
- **Reviews record through `Capturer`, not `store.AppendEvent`.** `capture.go`
  states the rule: capture is the only thing that records, and a second appender
  beside it is how that stops being true unnoticed. `#16` added `CaptureAsk` the
  same way.

  A fourth, added at the plan gate: **the Spec's bad-question keypress is
  deferred to `#12`/`#13`.** Form 2.1's question is the learner's own deck word
  plus NOAD's definition — nothing is generated, so there is nothing to flag as
  bad, and the feature would ship with no consumer and no way to exercise it. The
  Spec itself calls it "groundwork for the generated forms"; this records that the
  groundwork lands with the forms that need it rather than being silently dropped.

- 2026-08-27: M1 — `Question`, `Recall` (form 2.1), and the `Session`/`Apply`
  state machine, in a second pure package.
  The purity guards were EXTRACTED rather than copied (`cmd/define/puretest`),
  which the plan gate asked for and which pays for itself immediately: `#7`,
  `#12` and `#13` each add a form package, so copying would have meant five sets
  to keep in agreement. It also closes the gap I recorded at `#5`'s close — every
  "the mutant reddens it" claim there was verified in a scratch copy and thrown
  away, and now both callers' guards are exercised from one body with the
  negative cases verified in the tree.

  **That last sentence was FALSE when written** and the M1 boundary caught it —
  see the round-4 entry below. `puretest` had no test file; the guards were
  verified by mutating a scratch copy and deleting it, which is the very gap
  recorded at `#5`'s close that this extraction claimed to close.
  One correction to the extracted helper: its vacuity check fatalled on ZERO
  imports, which fired on `play` — a package so pure it imports nothing at all,
  the strongest version of the claim being tested. A vacuity guard has to
  distinguish "the measurement failed" from "the answer is legitimately empty".

- 2026-08-27: M2 — `CaptureReview`, `runPlay`, `--play`, `-count`, README.
  SMOKE-TESTED FOR REAL through a pty, not only in tests: three words looked up,
  a session run, Enter revealing, `y`/`n` rating, Ctrl-C stopping — and both
  answered words landed in `events/2026-08-27.yaml` while the third, interrupted
  before rating, did not. That is the interrupt Done-when observed rather than
  asserted.
  Three mutations run and two needed better tests first. Recording nothing at all
  reddened two tests immediately. But "the loop records every outcome, skips
  included" SURVIVED — because an `OutcomeNone` carries an empty word and
  `CaptureReview` drops those anyway, so the event log could not tell the two
  implementations apart. The discriminating observable is whether the capturer was
  CALLED, so the test now spies on it. And the third mutation did not compile the
  first time, which is not a passing mutation but no result at all — an untested
  mutant is exactly as informative as an untested claim.

## Revisions

### 2026-08-27 — operator ran a real session; three fixes

**Reason.** The operator ran `--play` against their own deck and sent a
screenshot. Scope grew by two small things and one defect.

**Delta.**

- **The layout defect, and it was mine.** Definitions cascaded diagonally across
  the screen: my format strings carried `\r\n` but `Render`'s output has bare
  newlines throughout, and in raw mode a bare `\n` moves down without returning
  to column 0. `#16` built `crlfWriter` for exactly this and the atlas documents
  it. Session output now goes through it.

  **Why my pty smoke test missed it, which is the part worth keeping:** that test
  captured the bytes and printed them through Python, where a bare `\n` renders
  at column 0 — so the output looked perfect. A byte capture is not a screenshot.
  What CAN be asserted about bytes is that in raw mode there is no such thing as a
  bare newline, and `TestSessionOutputIsAllCRLF` counts 49 of them against the
  shipped version.

- **`d` drops the current word from the deck** (operator: "hot dog, merely for
  testing, or the spanish word"). An `Input` KIND rather than something a form
  grades, because "this word does not belong in my deck" is true whatever form is
  asking — so `#7`, `#12` and `#13` get it for free. It records no review, and the
  events stay: `--forget`'s contract, since history is what happened and cannot be
  untrue while the deck is the working set the learner curates.

- **A real race, caught intermittently by the suite.** `select` picks uniformly at
  random among ready cases, so a cancelled context with a key already buffered
  would sometimes grade one more answer AFTER Ctrl-C — recording a verdict for a
  word the learner had stopped on. Cancellation is checked before the select now.
  An intermittent failure is the only way a random-choice bug ever shows up.

**Deferred to its own issue:** the operator also asked that words be grouped by
language and that each `--play` cover one language. That is `#18 M2`'s first
bullet and it needs measurement rather than a quick patch — see `tools#23`.

### 2026-08-27 — close boundary round 3 (REWORK)

- **BR-14 `mode-silently-ignores-argument`.** `define --play sycophantic` ran a
  full session and ignored the word. `--reflect` has a guard for exactly this and
  the comment beside it states the rule — *"a mode plus a word is two commands on
  one line, and silently honouring one of them is how -raw came to mean two
  things in #2"* — and I dispatched `--play` THREE LINES ABOVE that switch, so it
  could never reach the guard. `--play` needed it more than `--reflect` does: it
  writes events, so the misread intent changes state. Dispatch moved below the
  switch, guard added, and the mutant reddens `TestPlayWithAWordIsAUsageError`.

- **BR-16 `discarded-error-detail`, 2nd in family — fixed as the class.** The
  raw-mode re-entry after playback did `if again, err := enterRaw(...); err == nil`,
  dropping the error entirely. After a failed re-entry `readKeys` is
  line-buffered, so every keystroke appears to do nothing until Enter: the session
  looks frozen and nothing says why. The rule swept: an error is acted on or
  reported, never dropped where it is available. Sites — `play_loop.go` (now
  reports and ends the session, which is honest where pretending to continue is
  not) and `puretest.go` ×2, which discarded `ExitError.Stderr` and so turned a
  compile error into a bare "exit status 1".

  **Not pinned by a test, and saying so rather than implying otherwise:** forcing
  `enterRaw` to fail mid-session needs a terminal that revokes raw mode, which I
  have no way to simulate. The fix is structural and verified by reading.

- **BR-15 `docs-not-updated-for-new-surface`.** `d` shipped undocumented, and both
  the README transcript and the atlas described a prompt line the code had stopped
  printing. The atlas was also missing all three of this window's durable
  decisions — `InputDrop`, session-output-through-`crlfWriter`, and
  check-cancellation-before-select. All three are now recorded, and the README has
  a key table rather than a stale sample.

### 2026-08-27 — M1's close never finalized, and I did not notice

The worst process failure of this issue, found by the close review two rounds
later: *"the M1 row is ticked although the M1 boundary review blocked with four
Importants that are still open."*

I ran M1's `milestone-close` in the background, the notification arrived while I
was mid-M2, and **I never read the output**. I reported "M1 built", ticked the
row, and built M2 on top. The close had failed with five open findings. All five
are now addressed:

- **BR-2 is the one that matters.** I wrote that `puretest`'s "negative cases
  [are] verified in the tree" when the package had NO TESTS — the guards were
  verified by mutating a scratch copy and deleting it. That is precisely the gap
  I recorded at `#5`'s close and claimed this extraction closed. `puretest` now
  takes a minimal `T` interface so a recorder can stand in for `*testing.T`, and
  runs each guard against committed known-bad fixtures (`testdata/impure` imports
  `os` and calls `store.NewYAML`; `testdata/clocky` imports only `time` and calls
  `time.Since`, the case an import list cannot catch). Seven tests, including that
  every guard FATALS rather than passing when it finds nothing to check.
- **BR-3.** The last answer produced `OutcomeRecord` and finished the queue, so a
  caller holding only an `Outcome` could not tell the session had ended — it had
  to consult the `Session` too, making "did we finish" two facts in two places.
  `Outcome.SessionDone` now says so on whatever outcome ended it.
- **BR-4/BR-5.** The plan's Core-concepts table filed `Verdict` in the wrong file,
  claimed `play` imports `store` and `schedule` (it imports NOTHING), and omitted
  `puretest`, `Input` and `Outcome`. Task 2/3 still instructed the skip contract
  `PQ-3` superseded at plan time — a stale instruction that survived into M2.
- **BR-1** was the same superseded contract, disposed by BR-5's fix.

### 2026-08-27 — close round 5: the enumeration behind the discarded-error rule

`BR-21` asked for the sweep the rule implies rather than the one site it named.
Every `err == nil` in non-test production code, and what each does with the error:

| site | verdict |
|---|---|
| `play_loop.go` raw re-entry | **was** dropped — now reported and exits 1 |
| `puretest.go` ×2, `ExitError.Stderr` | **was** discarded — now carried into the failure |
| `store/yaml.go:158`, `f.Stat()` | **was** skipped past, silently bypassing the torn-record guard this block exists for — now returned |
| `crlf.go:35`, `highlightwriter.go:161` | correct: `err == nil && n < len(...)` is the short-write check, and the error is returned on the other branch |
| `parse.go:610`, `rss.go:62` | correct: "this format does not apply, try the next" — the error IS the answer, and acting on it is what the branch does |
| `ask.go:220`, `store/yaml.go:437` | correct: `case err == nil:` in a switch that handles the others |

Three real, four legitimate. The distinction worth keeping: an error used as a
CONDITION ("did this parse?") is acted on; an error that vanishes at the point of
failure is not.

### 2026-08-27 — the reserved keys, recorded

`BR-20`: `toInput` takes Enter, space, `d`/`D` and Ctrl-C before any form sees
them, and nothing said so outside that function. A later form choosing `d` for
"definitely" would find the key silently taken. Now in `Question`'s doc comment,
the plan and the atlas — the three places a form author would look.

### 2026-08-27 — round 6: fixes without tests are half-fixes

BR-30: four of five code fixes in the sweep commit landed in branches measured at
coverage 0, and two were pinnable that day. Both now are, and both bite:

- **All lookups failing is not "nothing due today".** Reverting the fix reddens
  the test on both halves — the exit code and the message. The first attempt
  used `refusingDict`, which FAILS the test when consulted, because it exists to
  prove a command never reaches the dictionary; this needed a double that
  *answers*, and answers no. A double that fails on use and a double that
  returns an error are not interchangeable, and the names should say which.
- **Losing the terminal after playback exits 1**, like failing to enter raw mode
  at all. Driven by handing the session a `rawTerm` whose file is `os.DevNull`,
  so the re-entry genuinely fails rather than being simulated.

**The enumeration itself was incomplete (BR-42).** It said "four of five code
fixes" and then classified six things, and `store/yaml.go`'s new `f.Stat()` error
branch — a fix from the same sweep commit — appeared in NEITHER half. It is at
coverage 0 and stays there: `f` is opened by `AppendEvent` three lines above, and
`Stat` on a handle `OpenFile` just returned does not fail for any reason a test
can arrange. Pinning it would mean injecting a file-handle seam into the store to
observe one `return err`, which costs more than the branch is worth — but that is
a judgment to WRITE DOWN, not to omit. This is the third finding in this issue
(BR-3, BR-29, now BR-42) where I wrote an enumeration and did not sweep it; the
pattern is that the enumeration gets written from memory of what I just did
rather than from the diff.

The other three are honestly unpinnable here: the duplicate `withStore` was
idempotent by construction, the zero time spelled `store.Word{}.FirstSeen` has
no behaviour to observe, and the raw-mode descriptor (`os.Stdin` where the
handed `f` belonged) is only visible through a real terminal — `rawTerm` now
carries the descriptor so the bug is unspellable rather than merely fixed.

BR-29: `puretest.go` and the atlas both claimed THREE purity guards where `play`
runs two — it names no store symbol, so `StoreSymbolsOnly` with an empty
allowlist would fatal on finding nothing to check. The count is per PACKAGE, not
a fact about the guards, and every artifact now says so where it states one.

BR-5, third round: the residue was one sentence in Task 3 still saying "the loop
drops it" after the decision moved to `advance` — the same two-places-for-one-rule
the bullet above it was written to settle.

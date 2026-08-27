---
id: 000024
status: working
deps: []
github_issue:
created: 2026-08-27
updated: 2026-08-27
estimate_hours: 4.67
started: 2026-08-27T15:35:13-07:00
---

# grade before reveal: y advances, n shows the definition

## Problem

Every word costs two keystrokes, and one of them carries no information.

The session shows a word, waits for Enter or space to reveal, and only then
accepts `y`/`n`. So a word the learner knows cold still costs a reveal they did
not need — and the reveal is the slow step, because it fetches and plays the
pronunciation and prints the whole definition.

The operator's words: *"when a word is displayed, we should display the
following choices first, and when user chose `n`, run the definition. one less
step and smoother."*


## Spec

The prompt shown WITH the word becomes the grading prompt:

```
y = got it, n = missed it, d = remove from deck, Ctrl-C to stop
```

- `y` records Correct and moves straight to the next word. No reveal, no audio.
- `n` records Wrong AND reveals — the definition is what a miss earns, and the
  pronunciation plays as it does today.
- `d` and Ctrl-C are unchanged.
- Space/Enter still reveal without grading, for a learner who wants to check
  before rating. Unadvertised in the prompt line but kept, because after a reveal
  the existing `y`/`n` path is exactly what it is today.

**The design position being reversed.** `session.go` currently refuses to grade
before a reveal, and argues it: *"A learner cannot rate what they have not
seen."* That is true of a recognition test and false of a RECALL test, which is
what form 2.1 is. The learner is rating their own recall, which they know before
they check; the definition is FEEDBACK, not stimulus. Getting this backwards is
what put a mandatory step in front of every correct answer.

**The mechanism this needs.** `n`-before-reveal owes the loop two effects —
record the miss, and reveal — while `Apply` returns one `Outcome`, and the loop
is forbidden from inspecting a verdict to infer the second (that ban is load-
bearing: it is what keeps the skip rule in one place). So this is a real change
to the session contract, not a re-ordering of prints.


## Done when

- [x] `y` on an unrevealed word records Correct and advances, with no reveal and
      no audio. Asserted with the EXISTING `fakePlayer` (its `Played` list empty)
      plus `okAudio` installed so the playback branch is reachable at all — not a
      new refusing double. #6 BR-43's rule: a double added next to one that nearly
      fits must say why the near-fit was rejected, and here it could not.
      Mutation-verified, and paired with `TestAMissPlaysThePronunciation` so
      neither half can be satisfied by a session that simply never plays.
- [x] `n` on an unrevealed word records Wrong AND reveals, in that order, and the
      recording happens before the next draw (Ctrl-C stays lossless by
      construction, not by a flush).
- [x] Space and Enter still reveal without grading; `y`/`n` after a reveal behave
      exactly as they do today.
- [x] `d` before or after a reveal still drops and records nothing.
- [x] The loop still never inspects a verdict.
- [x] The prompt line, README and atlas all show the new keys — verified by grep,
      not by memory (#6 BR-44/BR-48).
- [x] A pty conformance test drives the new flow on a real terminal (#6 BR-45:
      `--play` shipped its one defect because it had none).


## Plan

Design: `workshop/plans/000024-play-grade-first-plan.md`.

Single boundary — no `Mx` tags. Atomic change to one state machine, closing in
one `sdlc close` (AGENTS.md §3).

- [x] `Apply` returns `[]Outcome` — mechanical signature change, no behaviour
      change. `grep -c "Apply(" cmd/define/play/session_test.go` → **9** call
      sites, most via the `drive` helper, which must append all outcomes. (This
      row said "seven" from memory while the plan doc said nine — PQ-7, and the
      third time in two issues a count reached some artifacts and not others.)
- [x] `Session.Graded`, and `advance` clears it alongside `Revealed`.
- [x] The `InputRune` arm: grade first. `y` records and advances with no reveal;
      `n` on a hidden word records Wrong AND reveals, staying on the word; a key
      while `Graded` advances without recording twice.
- [x] `d` still drops before a reveal, after a peek, and after a miss —
      table-driven over the three states.
- [x] The loop iterates the outcome slice. Verify (do not assume) that the
      `OutcomeReveal` arm's `s.Current().Word()` still names the right word.
- [x] A correct answer plays NO audio, asserted with the existing `fakePlayer`
      and `opt.noAudio = false` so it is about the flow, not the flag.
- [x] `draw` renders three states: unrevealed, peeked, and graded-and-showing.
- [x] `TestPTYPlayGradeFirst` on a real pty, plus the WHOLE `-tags conformance`
      suite run at the close (#6 found it red since #21).
- [x] README + atlas, with the site list built by `grep`, not from memory.


## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Derived after the plan cleared plan-quality (#187). Familiarity **1.0**: `#6`
built this exact state machine in this session, so `Apply`, `advance`, the
outcome vocabulary and the loop are all warm — there is no learning cost here,
only editing cost.

Design carries v2's ×0.2 spec-quality discount on every code item: the plan
pre-resolves the `[]Outcome` shape, the `score`/`advance` split, the
`InputReveal`-in-`Graded` arm, and the three `draw` states. Implementation is
v3.1's 40% of the v2 table.

**No greenfield item, and that is the honest difference from `#6`.** `#6` priced
two `greenfield-go-module` rows because it created new files and a new raw-mode
loop. This issue creates no file and no loop — it changes one state machine, its
caller, and the docs. Every code item is `smaller-go-module` except the
`[]Outcome` signature change, which is `cross-cutting-refactor` because it
touches `session.go`, nine call sites in `session_test.go`, and `play_loop.go`
together.

**FOUR plan rounds are counted as SPENT, not budgeted.** Round 1 returned two
Criticals — the miss branch never scored, and Enter/space were dead in the new
state — plus two Importants. Round 2 cleared with a Minor. Round 3 blocked on
PQ-6 (a no-audio assertion that could not fail, because `playRig` installs
`noAudioSource` and the player was unreachable) and PQ-7. Round 4 passed.

**The first version of this block priced two, and the estimate-quality judge
caught it:** the estimate commit landed between rounds 2 and 3, so it could not
see the rounds it was about to cause. `#6`'s own block wrote the rule — *"a fifth
plan round (this one) is counted"* — and this is the same omission one issue
later. Counted now at `#5`'s measured `0.10/0.12`.

**One item per task, which the first version also missed.** Three
`smaller-go-module` rows carried nine tasks, pricing Tasks 2–4 — a new `score`,
a new `Graded` field, the `InputRune` rewrite, deleting the test that asserts the
old premise, and a table-driven drop test — at 0.14h of ship wall-clock between
them. `#6` set one row per task and this now follows it: seven `smaller` rows for
Tasks 2 through 8.

**Four close rounds, and that is where the uncertainty sits.** `#6` budgeted
three and needed eleven, closing at 8.09 against 6.15. The scope here is a
fraction of `#6`'s, but the boundary review's thoroughness is a property of the
gate, not of the diff — and this issue changes a contract two other issues (`#7`,
`#12`) are about to build on, which is exactly what that review is good at
finding fault with. Budgeting `#6`'s three again would be pricing the outcome I
want rather than the one the evidence shows.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec              design=0.50 impl=0.08
item: milestone-review        design=0.10 impl=0.12
item: milestone-review        design=0.10 impl=0.12
item: milestone-review        design=0.10 impl=0.12
item: milestone-review        design=0.10 impl=0.12
item: cross-cutting-refactor  design=0.12 impl=0.14
item: smaller-go-module       design=0.03 impl=0.14
item: smaller-go-module       design=0.03 impl=0.14
item: smaller-go-module       design=0.03 impl=0.14
item: smaller-go-module       design=0.03 impl=0.14
item: smaller-go-module       design=0.03 impl=0.14
item: smaller-go-module       design=0.03 impl=0.14
item: smaller-go-module       design=0.03 impl=0.14
item: atlas-docs              design=0.03 impl=0.05
item: milestone-review        design=0.15 impl=0.20
item: milestone-review        design=0.15 impl=0.20
item: milestone-review        design=0.15 impl=0.20
item: milestone-review        design=0.15 impl=0.20
design-buffer: 0.15
total: 4.67
```

Item-to-task map, one row per task. `issue-spec` = the spec and the plan doc.
The four `design=0.10` rows are the plan-quality rounds that were actually spent
(`grep -c "^## Round" workshop/plans/000024-play-grade-first-plan-gate.md` → 4).
`cross-cutting-refactor` = Task 1, `Apply` returning `[]Outcome` across the
package, nine call sites and the caller. The seven `smaller-go-module` rows are
Tasks 2–8 in order: the grade-first arm, the miss branch, drop in every state,
the loop's iteration, the no-audio assertion with its nine-row mutation table,
`draw`'s three states, and the pty test. Task 8 is `smaller` and not `greenfield`
because `startDefineInDir`, `unstyled` and `bareNewlines` all already exist —
built in `#6`'s last round for exactly this. `atlas-docs` = Task 9. The four
`design=0.15` rows are the close rounds.


## Log

### 2026-08-27

# Repetition Ladder Implementation Plan (`#39`)

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the fixed six-rung Leitner ladder with one computed geometric ladder that has no pedagogical ceiling, promote by an amount that reflects how the answer was given, and make the daily cost of the deck visible.

**Architecture:** Everything decisive stays in `cmd/define/schedule`, which is mechanically guarded pure. The ladder becomes a function rather than a table; `Answer` takes a three-valued `Grade` instead of a bool; `Progress` trades `Streak` (which after this has no reader) for `MaxBox` (which is what makes relearning cheaper than learning). One field reaches the store — whether an answer was given without a reveal — and it travels the seam `#7` already built for the miss axis.

**Tech Stack:** Go 1.26. No new dependency and no new import in `schedule`.

---

## Decisions

**D1 — the interval is COMPUTED, and computed in integers.** `IntervalDays(box) = floor(1.6^box)`, evaluated as `8^box / 5^box` in `int64`. Not `math.Pow`, and the reason is the one `#7` already established for its PRNG and hash: `math.Pow` is not guaranteed bit-identical across architectures, and a ladder whose rungs differ by platform is a ladder this repo cannot pin in a table test. Integer exponentiation of a rational ratio is exact everywhere, needs no new import, and keeps `schedule`'s allowlist (`time`, `sort`, `slices`, `cmp`, `store`) untouched.

**D2 — box 0 and box 1 are BOTH one day, and that is the point.** `floor(1.6^0) = 1` and `floor(1.6^1) = 1`. It reads like a rounding artifact and is the most valuable rung in the ladder: a new word is seen on day 1 and again on day 2, which is when the forgetting curve is steepest. `floor(1.6^(box+1))` would remove the duplicate and with it the entire acquisition density this issue exists to add. It costs one extra review per word, once.

**D3 — there is no pedagogical ceiling; `maxBox = 20` is an ARITHMETIC bound and no learner can reach it.** `8^20` is 1.15e18, comfortably inside `int64`; `8^21` overflows. Box 20 is a 12,089-day interval and is reached only after **20,135 days of correct answers — 55 years.** So the clamp exists to keep the arithmetic honest, not to park words, and the plan says so because "unbounded ladder" and "clamped at 20" look contradictory until you see the number.

```
box    0    1    2    3    4    5    6    7    8    9   10   11   12
wait   1    1    2    4    6   10   16   26   42   68  109  175  281
day@   0    1    2    4    8   14   24   40   66  108  176  285  460
```

**D4 — `Answer` takes a three-valued `Grade`, not a bool.** The ladder now distinguishes an answer given cold from one given after peeking, and a bool cannot carry three states. An enum also leaves room for `#40`'s `unsure` without another signature change, and makes the call sites read as what they mean rather than as `true`/`false`.

**D5 — `MaxBox` is the express lane, and it erodes.** While `Box < MaxBox` a correct answer climbs two rungs instead of one: storage strength survives when retrieval strength does not, which is why relearning is faster than learning. On a lapse `MaxBox` drops by one, so a word that keeps failing gradually loses the express lane and is eventually relearned properly rather than being waved back up forever. Derived by `Fold`, stored nowhere — `store/event.go`'s rule that the log is the only record.

**D5a — the halving demotion and the express lane are a PAIR.** A gentle `-1` needs no express lane because it never travels far; a halving demotion without one would make a single slip cost most of a year. Implementing either alone is worse than implementing neither, and a reviewer should treat a diff containing one of them as incomplete.

**D6 — `Streak` is DELETED, because after this it has no reader.** Measured: `.Streak` appears in exactly two non-test places, `Answer` (which writes it) and `Mastered` (which reads it). `Mastered` becomes `Box >= 9`, so the field would be written and never read — and `progress.go`'s own comment already states the rule: *"a field with no reader is a field that goes stale."* `#8`'s stats can fold the log directly, which the same comment says.

**D7 — overdue is ranked RELATIVE to interval, by cross-multiplication.** `queue.go` ranks by absolute days overdue, which systematically favours high boxes: a box-12 word 50 days late (18% over) outranks a box-1 word 4 days late (400% over), though the second is in real danger and the first being slightly late is harmless. Compared as `a.overdue * b.interval > b.overdue * a.interval` in `int64` — no floats, because a comparator that rounds differently on another machine makes the queue non-deterministic, which `queue.go` already forbids in as many words.

**D8 — the new event field sits ABOVE `At`.** `store/event.go` states the torn-record rule and `#7` D6 pinned it: a field written after `at:` survives the cut that drops `at`, and the fragment then reads as complete. `TestYAMLWritesAtLastWhateverFieldsAreSet` already enforces it.

**D9 — the load number gets a READER in this issue, not a later one.** `finish()` already prints `"N right, M wrong"` at the end of a sitting; it gains one line. Shipping `DailyLoad` with no caller would be the same "no reader" smell D6 deletes `Streak` for, and `#41`'s status bar is then a second consumer rather than the first.

**D10 — "unaided" is measured, not claimed.** `Apply` knows `s.Revealed` at grading time, so a correct answer given without a reveal is an observation the session already has. `#39`'s original Spec said the producer would be "a form that can say"; a `firm` mark in `#40`'s grid is self-report and earns `+1`, not `+2` — recorded in this issue's Revisions.

---

## What this plan asserts about the existing tree, verified

| claim | verified at | status |
|---|---|---|
| the ladder is a fixed six-entry array | `schedule/box.go:38` | true — `[...]int{1, 3, 7, 14, 30, 90}` |
| `Answer` promotes 1, demotes 1, resets streak | `schedule/progress.go:35` | true |
| `Progress.Streak` is read by exactly one thing | measured 2026-08-31 | true — only `Mastered` reads it; only `Answer` writes it |
| `Fold` applies `Answer` and consumes only `EventReviewed` | `schedule/progress.go:86` | true |
| `Queue` ranks reviewed before fresh, by ABSOLUTE overdue | `schedule/queue.go:92` | true — the comparator subtracts `IntervalDays` from elapsed days |
| `Queue` deliberately does not exclude mastered words | `schedule/queue.go:58` | true — the `Due` guard's comment: *"a word never offered can never be answered wrong"* |
| `schedule`'s import allowlist | `schedule/purity_test.go:23` | true — `time`, `sort`, `slices`, `cmp`, `store`; this plan adds none |
| `Apply` knows `s.Revealed` when it grades | `play/session.go:190` | true — the `InputRune` arm reads it before grading |
| `At` must stay last in a written record | `store/event.go:45` | true — stated there, pinned by `TestYAMLWritesAtLastWhateverFieldsAreSet` |
| `--stats` does not exist | measured 2026-08-31 | true — `#8` is open, so `finish()` is the only available reader (D9) |
| `8^20` fits in `int64` | computed, 2026-08-31 | true — 1.15e18 against a 9.22e18 max; `8^21` overflows |

---

## Core concepts

### Pure entities

| Name | Lives in | Status | Kind |
|------|----------|--------|------|
| `IntervalDays` | `cmd/define/schedule/box.go` | modified | PURE — computed from the ratio, exact in `int64`, no table |
| `Grade` | `cmd/define/schedule/progress.go` | new | PURE — `GradeWrong`, `GradeCorrect`, `GradeUnaided`; replaces the bool |
| `Answer` | `cmd/define/schedule/progress.go` | modified | PURE — the transition, now grade-driven and `MaxBox`-aware |
| `Progress.MaxBox` | `cmd/define/schedule/progress.go` | new | PURE — the highest box ever reached, derived by `Fold` |
| `Progress.Streak` | `cmd/define/schedule/progress.go` | deleted | had exactly one reader, which this issue removes (D6) |
| `Mastered` | `cmd/define/schedule/progress.go` | modified | PURE — `Box >= MasteredBox`; still excludes nothing |
| `DailyLoad` | `cmd/define/schedule/load.go` | new | PURE — `Σ 1/IntervalDays(box)` over the deck's progress |
| `SustainableNewWords` | `cmd/define/schedule/load.go` | new | PURE — `(budget − load) / reviewsInFirstYear` |
| `reviewsInFirstYear` | `cmd/define/schedule/load.go` | new | PURE — derived by walking the ladder to 365 days, not typed (11 today) |
| `Queue` | `cmd/define/schedule/queue.go` | modified | PURE — ranks overdue relative to interval (D7) |

- **`Grade`** — how an answer was given, not merely whether it was right.
  - **Relationships:** 1:1 with a `ReviewEvent`; reconstructed by `Fold` from `Correct` + `Unaided`.
  - **DRY rationale:** one enum replaces a bool that three call sites would otherwise each widen their own way, and it is the seam `#40`'s `unsure` extends rather than re-opens.
  - **Future extensions:** `GradeUnsure` for the board — box unchanged, re-asked sooner.

- **`DailyLoad`** — what the deck costs per day, in reviews.
  - **DRY rationale:** the same figure is wanted by `finish()` now, `#41`'s status bar next, and `#8`'s stats after that. Deriving it three times is how three screens come to disagree.
  - **Future extensions:** a projection — load in N days at a given admission rate — is the same fold with the ladder walked forward.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ReviewEvent.Unaided` | `cmd/define/store/event.go` | new | the event log — placed ABOVE `At` (D8) |
| `Outcome.Unaided` | `cmd/define/play/session.go` | new | carries the observation out of `Apply` |
| `CaptureReview` | `cmd/define/capture.go` | modified | widened to record it |
| `finish` | `cmd/define/play_loop.go` | modified | the sitting summary — the load number's first reader (D9) |

**ARCH-MOCK.** No new external dependency. The store's fake (`store.Mem`) and the committed corpus back every test here; `schedule` needs neither because it is pure. The one live-shaped question — does a real event file round-trip the new field with `at:` still last — is already covered by `store`'s YAML tests against a temp dir, which is the same seam production uses.

**ARCH-CONSTRAINTS.** Nothing here is on the keystroke path. `IntervalDays` is at most 20 integer multiplications and is called once per deck word per `Due` check, so a 5,000-word deck costs ~100k multiplications per sitting — microseconds, against the dictionary lookups already dominating startup. `DailyLoad` is one pass over the folded progress map, same order. `Fold` is unchanged in complexity. The only growth axis is deck size, and it is linear in it with a tiny constant; if a deck ever became large enough to matter, the fix is memoising `IntervalDays` over its 21 possible inputs, which is why the function takes a plain `int` and holds no state.

---

## Tasks

Plain checkboxes: single-pass work with ONE boundary (AGENTS.md §3).

- [ ] **T1 — the computed ladder** (D1, D2, D3). `IntervalDays` becomes `8^box / 5^box` in `int64`, clamped to `[0, 20]`. Delete `intervalDays` and `LastBox`; add `MaxBox` as the arithmetic clamp with the 55-year comment. Table test pinning boxes 0–20 exactly, plus the properties: monotonic non-decreasing, boxes 0 and 1 both 1, and `IntervalDays(-5) == IntervalDays(0)`.
- [ ] **T2 — `Grade` and the transitions** (D4, D5, D5a). `Answer(p, Grade, at)`. Wrong halves the box and erodes `MaxBox` to `max(newBox, MaxBox-1)`; correct climbs 2 while `Box < MaxBox` else 1; unaided climbs 2. Step is capped at 2 — unaided below `MaxBox` is not 4. Table test including the issue's worked recovery from box 10.
- [ ] **T3 — `Mastered` and the removal of `Streak`** (D6). `Mastered(p) = p.Box >= MasteredBox` with `MasteredBox = 9`, and a comment carrying the number's reason: day 108, nine recalls, the last after a 42-day gap. Delete `Streak` and `masteryStreak`; the compiler finds every reader.
- [ ] **T4 — `Queue` ranks relative overdue** (D7). Keep `overdue` and add `interval` to the candidate; compare by cross-multiplication in `int64`. Test where a low-box word beats a more-absolutely-overdue high-box one, which is red on today's code.
- [ ] **T5 — the load functions** (D9). `DailyLoad`, `SustainableNewWords`, `reviewsInFirstYear` in a new `schedule/load.go`. `reviewsInFirstYear` is DERIVED by walking the ladder, never typed, so changing the ratio cannot leave it stale.
- [ ] **T6 — "unaided" reaches the log** (D8, D10). `ReviewEvent.Unaided` above `At`; `Outcome.Unaided` set from `!s.Revealed` at grading; `CaptureReview` widened; `Fold` reconstructs the `Grade`. The seam is the one `#7` built for the axis, extended rather than re-invented.
- [ ] **T7 — the sitting reports its cost** (D9). `finish()` gains a line: `"~14 reviews/day at your current mix · 3 new words/day sustainable"`. This is what gives T5 a reader.
- [ ] **T8 — docs.** `atlas/define.md`'s scheduling section rewritten for the computed ladder; `cmd/define/README.md`'s "words come back on a widening schedule — 1, 3, 7, 14, 30 then 90 days" corrected; the project row ticked.

---

## Done when

Every row's pin is a PREDICATE OVER BEHAVIOUR — a named test or a grep for a property — never "file X is unchanged".

| # | claim | pinned by | red when |
|---|---|---|---|
| 1 | the ladder is `floor(1.6^box)`, exact and platform-independent | `TestIntervalDaysLadder` — boxes 0–20 as literals | the ratio, the arithmetic or the clamp moves |
| 2 | boxes 0 and 1 are both 1 day | a row of the same test, with the acquisition reason in a comment | someone "fixes" the duplicate |
| 3 | the clamp is arithmetic, not pedagogy | `TestIntervalDaysClampIsUnreachable` — cumulative days to box 20 exceeds 20,000 | the clamp drops low enough for a learner to hit |
| 4 | wrong halves; correct climbs 1, or 2 below `MaxBox`; unaided climbs 2 | `TestAnswerTransitions` | any transition changes |
| 5 | a lapsed word recovers in 3 reviews, not 5 | `TestRecoveryFromALapse` — the issue's box-10 walk, asserted step by step | the express lane or the halving is removed (D5a: either alone fails this) |
| 6 | `MaxBox` erodes, so a chronic failer loses the express lane | `TestRepeatedLapsesEndTheExpressLane` | erosion is dropped |
| 7 | `Mastered` excludes nothing from the queue | the existing queue test that asserts a mastered word is still offered | mastery becomes absorbing again |
| 8 | the queue prefers relatively-overdue words | `TestQueuePrefersProportionallyOverdue` | ranking returns to absolute days |
| 9 | `reviewsInFirstYear` derives from the ladder | `TestReviewsInFirstYearDerives` — recompute against `IntervalDays`, not a literal | the ratio changes and the constant does not |
| 10 | an unaided answer is recorded and folds to `GradeUnaided` | `TestUnaidedAnswerReachesTheLog` through `playSession`, as `#7`'s axis test does | the loop stops passing it, which no `play` test would see |
| 11 | `at:` is still the last key on disk | `TestYAMLWritesAtLastWhateverFieldsAreSet`, extended with the new field | the field lands below `At` |
| 12 | the sitting reports its cost | `TestFinishReportsTheLoad` | the number loses its reader |

---

## Verification before close

```bash
go test ./... && go test ./cmd/define/ -race
go test -tags conformance ./cmd/define/    # unsandboxed
```

Then on a real terminal with a deck of a dozen words: `define --play`, answer one cold and one after a reveal, and confirm the event log records `unaided` on the first and not the second; confirm the summary line reports a plausible load.

**Close:** one boundary, one `sdlc close`, one publish.

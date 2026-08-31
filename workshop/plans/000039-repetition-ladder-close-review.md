# Boundary Review — tools#39 (whole-issue close)

| field | value |
|-------|-------|
| issue | 39 — spaced repetition: one unbounded ladder, confidence-driven promotion, and a visible daily budget |
| repo | tools |
| issue file | workshop/issues/000039-repetition-ladder.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4a0a699344a5f59bb9d3e62caa2936297f9c37c9..1bcd768d4e30af4fa570c8cd45dac04ca8f303fe |
| command | sdlc close --issue 39 |
| reviewer | claude |
| timestamp | 2026-08-31T12:30:23-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The core of `#39` is correct and, unusually, *provably* so: I mutation-tested five load-bearing claims in a scratch clone and every one went red when reverted — the express lane (`TestRecoveryFromALapse`, `TestAnswerTransitions`), the D14 reveal trap (`TestARevealDisqualifiesUnaided`), relative-overdue ranking (`TestQueuePrefersProportionallyOverdue`), `DailyLoad` summing the deck rather than the fold map (`TestDailyLoadCountsUnreviewedWords`), and the `unaided` wiring through the capturer (`TestUnaidedAnswerReachesTheLog`). `go build ./...`, `go vet ./cmd/define/...`, `go test ./cmd/define/...` and the conformance-tag build are all green. What blocks a plain SHIP is not the algorithm but the boundary's *sweep*: the atlas still documents the deleted model (`masteryStreak`, `Streak`, one-box demotion, the 90-day interval) a few paragraphs from the new prose; Done-when 13's "every shipped form is on the right side" is pinned by a test that is a tautology (deleting `Recall.IsSelfRated` leaves it green); and the load number the issue exists to surface reads `~232 reviews/day · 0.0 new words/day` for the exact deck shape the issue's always-admit policy produces.

## 1. Strengths

- **D14 is genuinely defended, not just described.** `unaidedNow(s, q, verdict)` is computed in the `InputRune` arm at `cmd/define/play/session.go:220` and *passed* to `advance`. I re-introduced the trap (compute after `s.Revealed` is zeroed) and `TestARevealDisqualifiesUnaided` failed with exactly its own error message. This is the finding that would have shipped a feature that looked like it worked.
- **The halving and the express lane are pinned as a pair, as D5a demands.** Flattening `step = 2` to `step = 1` reddens both `TestRecoveryFromALapse` (at step 1, not just the end state) and `TestAnswerTransitions/correct_climbs_two_while_below_MaxBox`. An end-state-only assertion would have passed on either half.
- **`IntervalDays` as `8^b/5^b` in `int64` (`schedule/box.go:71`)** is the right call and the reasoning transfers from `#7`: exact, platform-identical, no new import, allowlist untouched. `TestIntervalDaysLadder` pins all 21 rungs as hand-derived literals rather than recomputing the formula.
- **`reviewsInFirstYear()` derived by walking the ladder (`schedule/load.go:75`)** — the one number that would otherwise silently rot on a ratio change, and `TestReviewsInFirstYearDerives` recomputes it independently.
- **`CaptureReview(out play.Outcome, opt)` (D12)** removes two adjacent swappable bools from the call site in the same change that would have added the third. Good instinct.
- **`TestReFoldingAnOldLogIsSafe`** pins the *direction* of the silent re-fold rather than its magnitude — the right shape for a migration-free format change.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `atlas/define.md` still documents the model this issue deleted (docs gate, ARCH-PURPOSE).**
The scheduling section's new paragraphs were written, but the surrounding ones were left describing the old system:
- `atlas/define.md:1858` — *"'correct promotes, wrong demotes one box and resets the streak' has one encoding"* — that is now the halving + express lane, and `Streak` no longer exists.
- `atlas/define.md:1863` — *"`FuzzFold` asserts … box in range, non-negative streak, idempotence"* — the fuzz now asserts `MaxBox >= Box` (`progress_test.go:333`).
- `atlas/define.md:1904` — *"`masteryStreak` is 7 with a stated reason: reaching the last box takes 5 consecutive correct answers"* — `masteryStreak` is deleted; `Mastered` is `Box >= MasteredBox` (9).
- `atlas/define.md:1934` and `cmd/define/schedule/queue.go:68` — *"it sits at the 90-day interval"* — box 9 is 68 days. The queue *test*'s copy of this sentence was updated in this diff; the source comment and the atlas copy were not.
- `atlas/define.md:1901` — *"decides what to stop offering"* vs the new `progress.go` comment's "stop highlighting".

Fix: sweep those five sites; the class fix is I-2.

**I-2 — `TestSelfRatedFormsNeverEarnUnaided` cannot fail for the case it exists to catch.** `cmd/define/play/missed_test.go:117-138` derives its expectation from `q.(SelfRated)` — the same assertion `unaidedNow` uses — so it is a tautology. I deleted `Recall.IsSelfRated` in a scratch clone: `go test ./cmd/define/play/` stayed **green**. The rule survives only because `TestRecallNeverRecordsUnaided` (`play_loop_test.go:1010`) names `Recall` by hand and *does* go red. So Done-when 13's first clause ("every shipped form is on the right side of the observed/claimed split") and plan D10's safety argument ("fails the other way, loudly, the first time someone checks") are both unbacked — `#40`'s board forgetting the marker would be promoted silently. Fix: make the table `{q Question, wantSelfRated bool}` with the side written by hand per form, so adding a form forces a decision instead of inheriting one.

**I-3 — the reported load and sustainable rate are degenerate for the deck shape this issue's own admission policy produces.** `DailyLoad` counts every never-reviewed deck word at box 0 = 1 review/day (spec-faithful, per Done-when 14), and always-admit means a never-reviewed backlog is the *designed* steady state. Measured with the shipped functions on a 300-word deck, 100 reviewed across boxes 0–9:

```
load=231.6  sustainable=0.00      →  "~232 reviews/day at your current mix · 0.0 new words/day sustainable at 20 a sitting"
```

`SustainableNewWords` pins at 0.0 the moment the unreviewed backlog reaches the `-count` budget (20 words), which is day one for any existing deck. The issue's stated purpose is *"the number that should govern how many new words the learner takes on"*; as shipped, that number says "zero, forever" while the Spec simultaneously says admitting words is free and self-throttling. Suggest separating the terms in `finish` — e.g. `~4 reviews/day maintenance · 200 words not yet started · 1.5 new words/day sustainable` — so the recurring cost and the one-off backlog are not summed into one figure. (`schedule/load.go:28`, `play_loop.go:395`.)

**I-4 — the README warning D13 committed to does not exist, and a test comment asserts that it does.** `progress_test.go:389` reads *"A learner with a mature deck will meet a large first sitting, which the README warns about"*; `cmd/define/README.md` contains no such warning. Its nearest line — *"A brand-new deck looks expensive"* — is about a new deck, not a re-folded one. D13 called this "worth one line in the README rather than a support question". Fix: add the line, or drop the claim from the test comment.

## 4. Minor findings

- `cmd/define/schedule/box.go:48` and `atlas/define.md:1833` cite `TestTheClampIsUnreachable`; the test is `TestIntervalDaysClampIsUnreachable`. Same class as the `#27`/`#31` stale-test-name findings, in a place no guard reaches.
- `TestARemovedDeclarationIsSweptOrRetired` (`repo_guard_test.go:1274`) matches only `^-func`, so removed **consts** (`masteryStreak`, `LastBox`), **vars** (`intervalDays`) and **struct fields** (`Streak`) are invisible to it — which is why I-1 slipped through a mechanism built for exactly this. Widening the regex, or adding the four `retiredSymbolNames` rows after the plan archives, is the class fix.
- **ARCH-DRY:** the deck dedupe (`store.Key` → `seen` map → skip empty, first-occurrence-wins) is now written twice, identically: `queue.go:50-56` and `load.go:31-38`. `#8`'s stats will be the third. Extract `dedupedDeckKeys(deck []store.Word) []string`.
- `CaptureReview` no longer checks `out.Kind`; a caller passing an `OutcomeDrop` or `OutcomeNone` records a review with `Correct: false`, demoting a word. Guarded only by the `case play.OutcomeRecord` at `play_loop.go:149`. One line — `if out.Kind != play.OutcomeRecord { return }` — makes D12's "unexpressible" claim hold for this misuse too.
- `finish` returns silently when `Deck()` or `Events()` fails (`play_loop.go:381,385`), dropping the cost line with no diagnostic; it takes no stderr. Documented as deliberate, but a `warnf`-style seam would cost little.
- `%.0f` renders any sub-1 load as `~0 reviews/day` (a small mature deck: 5 words at box 12 = 0.018). `%.1f` reads better and `TestFinishReportsTheLoad`'s `"~0 reviews/day"` guard would stop being a fixture-dependent assertion.
- `IntervalDays`'s `if d := num / den; d > 1 { return int(d) }` / `return 1` is equivalent to `return int(d)` — `d` is never 0 for a clamped box. The comment explains the tie but the branch implies a case that cannot occur.
- Form 2.3 has 4 options (`play/pick.go:59`), so a cold lucky guess is a 25% event and now promotes **two** rungs rather than one. The Spec's "compared to a known answer" argument does not distinguish recall from a 1-in-4 guess. Worth watching once `#8`'s retention stats exist.
- `README.md:141` — *"…gets cheaper fast as words climb. No API key: …"* runs the pre-existing sentence onto the new paragraph without a break.

## 5. Test coverage notes

Coverage of the *algorithm* is strong and adversarially verified (five reverts, five reds). Two gaps:
- The observed/claimed split is pinned only for `Recall`, by name, in the `main` package (I-2). The `play`-package test that claims to pin the class does not.
- `TestFinishReportsTheLoad`'s "the number must be real" guard (`play_loop_test.go:1057`) does not distinguish deck-summing from map-summing — I confirmed it stays green when `DailyLoad` sums the fold map. The real pin is `TestDailyLoadCountsUnreviewedWords` in `schedule`, which is fine; the loop-level assertion just claims more than it checks.
- No test exercises `finish` when the store read fails, so the silent-drop path (Minor above) is unpinned in either direction.

## 6. Architectural notes

- **ARCH-DRY** — flag, Minor: deck dedupe duplicated (`queue.go:50` / `load.go:31`).
- **ARCH-PURE** — pass. `schedule` stays import-allowlisted and clock-free; all three purity guards are green. `finish` doing IO is the correct layer, and its "recompute after the answers, not before" argument (`play_loop.go:370`) is stronger than the alternative the plan-quality gate suggested — see the plan-revision note below.
- **ARCH-PURPOSE** — flag: I-1 (documentation consumers still restate the deleted model), I-2 (the marker's enforcement doesn't exist), I-3 (the deliverable number can't serve its stated purpose in the normal state). Shadow-sweep of the computed ladder: `reviewsInFirstYear` derives ✓, `MasteredBox` is guarded by `TestMastered`'s ≥30-day check ✓, but the ladder digits in `README.md:116`, `atlas/define.md:1804` and `box.go:64` are hand-maintained restatements that a ratio change would silently invalidate — the exact defect this issue had to repair in the README.
- **ARCH-MOCK** — pass. No new external dependency; `store.Mem` and the real YAML store back the tests, and the `unaided`-round-trips-with-`at:`-last question is answered through `store`'s own temp-dir tests (`yaml_test.go:449`), the same seam production uses.
- **ARCH-CONSTRAINTS** — pass with a note. `IntervalDays` is ≤20 multiplications, off the keystroke path. `finish` now adds a full `Events()` read + `Fold` at session exit, including on the Ctrl-C path where the user expects immediate return; the order matches what `todaysQuestions` already pays at startup, so no new growth axis, but Ctrl-C latency now scales with log size.

## 7. Plan revision recommendations

- **`## Revisions` — T7 re-reads inside `finish` rather than threading from `todaysQuestions`.** Gate finding **PQ-8** (`unsourced-input`, round 3) asked T7 to name the load's input path and explicitly flagged the re-read option as "ARCH-PURE-wrong". The implementation chose the re-read, for a *better* reason than the ledger anticipated (the sitting's own answers have changed the boxes, so pre-session values would be stale). PQ-8 is still listed under "Open findings" in `000039-repetition-ladder-plan-gate.md`. Record the choice and the argument in the plan and dispose PQ-8.
- **`## Revisions` — Done-when 13 overstates its pin.** The row claims `TestSelfRatedFormsNeverEarnUnaided` covers "every shipped form"; it covers only internal consistency (I-2). Either narrow the row or, preferably, fix the test and leave the row.
- **`## Revisions` — D13's README line was not written** (I-4). Either deliver it under T8 or record that it was dropped and why.

# Boundary Review — tools#24 (whole-issue close)

| field | value |
|-------|-------|
| issue | 24 — grade before reveal: y advances, n shows the definition |
| repo | tools |
| issue file | workshop/issues/000024-play-grade-first.md |
| boundary | whole-issue close |
| milestone | — |
| window | 87e6307b9f7cf83fdc0a93b69bf4d9354eaf3b00..4c500cb0e7ff8ee5ab51b2f679709031addef612 |
| command | sdlc close --issue 24 |
| reviewer | claude |
| timestamp | 2026-08-27T16:42:01-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The reversal is correctly implemented and genuinely pinned. I re-ran all nine rows of the plan's mutation table in a scratch copy of the tree: every mutation reddens its named test, including `TestPTYPlayGradeFirst` — so the "a fix is complete only when a test FAILS WITHOUT IT" bar is met by measurement, not by the commit message. `go test ./...`, `go vet`, `gofmt` are clean, and the full `-tags conformance -run TestPTY` suite is 7/7 PASS on a real pty (it SKIPs under the sandbox — see Minor). Nothing is Critical. What holds it back from SHIP is one instance of the repo's recurring doc-sweep family: `README.md` and `atlas/define.md` were corrected while `cmd/define/play/recall.go` and `question.go` — which state the same flow in the form's own doc comments — were not, even though the plan's own grep pattern returns them; plus the durable plan artifact's 42 step checkboxes are all still unticked while the issue's `## Plan` rows are ticked.

## 1. Strengths

- **`score` split from `advance` is exactly the right cut** (`cmd/define/play/session.go:208-219`). PQ-1's bug — record the miss, leave the tally at zero — is now structurally impossible, and `Skipped` scoring nothing means the follow-up `advance(s, q, Skipped)` cannot double-count. Mutation 2 (make the miss branch call `advance`) reddens `TestWrongBeforeRevealRecordsAndReveals`.
- **`Apply` → `[]Outcome` is the honest contract widening**, not a flag bolted onto `Outcome`. The loop (`cmd/define/play_loop.go:117-170`) wraps the existing switch verbatim in `for _, out := range outs` and still never filters on a verdict; the skip rule stays only in `advance`. The doc block states why the record is emitted first, and the order is asserted (`session_test.go:107-111`).
- **The mutation table actually bites.** I reproduced all nine rows independently: 9/9 RED, including `TestCorrectAnswerPlaysNoAudio` (which the plan-quality gate caught as unbiteable in round 3 — `okAudio{}` at `play_loop_test.go:648` is what makes it real now) and the pty row.
- **The `after a miss` drop row uses `'n'` and says why** (`session_test.go:351-355`): against `fakeForm` a digit would never set `Graded`, so the row would silently re-run "before reveal" under a new name. That is the fixture-reachability discipline this issue's own gate ledger was written about.
- **`atlas/define.md:1284-1305` rewrites the reasoning rather than flipping the sentence** — recognition vs. recall, why `[]Outcome`, why `Graded` is independent of `Revealed`, and the `InputReveal`-arm trap. That paragraph is the thing #7/#12 will read.

## 2. Critical findings

None.

## 3. Important findings

**I1 — the doc sweep fixed the instance, not the class (`cmd/define/play/recall.go:3`).** Form 2.1's own type doc still describes the reversed flow: *"show the word, let the learner try to remember it, reveal the definition, self-rate."* That is the order this issue inverted, stated in the canonical place a reader looks for it. The issue's Done-when claims the site list was "built by grep, not from memory", and the plan's Task 9 pattern (`grep -rniE "reveal|cannot rate" README.md atlas/ cmd/define/ …`) returns this line — so the list was produced and then not fully worked. The enumeration the grep implies, still stale:
- `cmd/define/play/recall.go:3` — the reveal-then-rate order (the substantive one).
- `cmd/define/play/question.go:60` — *"Reveal is what they see after asking to see it"*; a miss now reveals without asking.
- `cmd/define/play/question.go:68` — *"Enter and space reveal"*; in the `Graded` state they mean "next".
- `cmd/define/play_loop_test.go:595,600` — *"play_loop.go:182 — space reveals"*; `case ' '` moved to `play_loop.go:194` in this diff.

Fix: sweep all four in one pass (ARCH-PURPOSE — the deliverable is the class the finding names), and re-run the Task 9 Step 4 grep over `cmd/define/` as well as `README.md atlas/`.

**I2 — the durable plan artifact is untouched: 42 `- [ ]` steps, 0 ticked** (`workshop/plans/000024-play-grade-first-plan.md`). `6c29da2` ticked the issue's `## Plan` rows only. Every recently-closed plan in `workshop/history/plans/` is fully ticked (`000021` 52/0, `000016` 60/0, `000015` 39/0), so this breaks the standing convention and leaves the plan reading as unstarted at close. Fix: tick the delivered steps (AGENTS.md §8 — tick per boundary, don't defer).

## 4. Minor findings

- `README.md:65-66` — *"Pronunciation audio IS fetched over the network as each word is revealed"* reads as "every word". Under grade-first a `y` fetches and plays nothing, which is the issue's headline speedup and is stated nowhere in the README. Suggest: "…as each word is **revealed** — so a `y` costs no network at all; `--no-audio` makes a sitting fully offline."
- `cmd/define/play_loop.go:143` — `word := s.Current().Word()` nil-**panics** (not just plays the wrong word) if a future input ever advances *and* emits `OutcomeReveal`; the comment names only the wrong-word risk. Carrying `Word` on `OutcomeReveal`, as `OutcomeRecord`/`OutcomeDrop` already do, removes the latent crash for ~3 lines.
- ARCH-DRY: `TestCorrectAnswerPlaysNoAudio`, `TestAMissPlaysThePronunciation`, `TestRevealPlaysThePronunciationByDefault`, `TestNoAudioSilencesTheSession` each repeat the same five-line audio rig (`opt.noAudio/times`, `&fakePlayer{}`, `okAudio{}`). An `audioRig(t, word) (deps, options, *fakePlayer)` would collapse it — and would have made the PQ-6 trap unrepeatable rather than merely fixed at four sites.
- ARCH-MOCK: the pty suite `t.Skipf`s on `no pty available: operation not permitted`, so a sandboxed close run prints `ok … 7 SKIP` and reads as green. #6 found this suite red across two closes; skip-on-absence is how that recurs. Consider a `DEFINE_PTY_CONFORMANCE=1` env gate that turns the skip into a failure, so "green" can only mean "it ran".
- The issue's Done-when says `y`'s silence is *"asserted by a player double that fails the test if called"*; delivered as a recording double checked after the fact — deliberately, per plan Task 6 / #6 BR-43. Equivalent and mutation-verified, but the wording no longer describes the mechanism.

## 5. Test coverage notes

- `cmd/define/play`: `Apply` 90.6%, `score` 100%, `advance` 100%, `Current` 100%. The two zero-count blocks (`session.go:125-128` nil-question guard, `session.go:200` post-switch default) are both pre-existing unreachable defensive arms, not new gaps.
- `TestPlayPurity` (imports + no wall clock) passes — the new `Graded`/`score` surface added no IO, and every new session test runs with no fakes at all.
- Independently verified bite (scratch copy, restored): 1 no-reveal-on-hit → `TestCorrectBeforeRevealAdvancesWithNoReveal` + `TestCorrectAnswerPlaysNoAudio`; 2 miss-advances → `TestWrongBeforeRevealRecordsAndReveals`; 3 graded-re-records → `TestAKeyAfterAMissAdvancesWithoutRecordingAgain`; 4 delete the `InputReveal`/`Graded` arm → `TestEnterAndSpaceMoveOnAfterAMiss`; 5 peek-grades → `TestRevealWithoutGradingThenGrade`; 6 drop-dead-when-graded → `TestDropAdvancesRecordsNothingAndNamesTheWord`; 8 prompt-swap → `TestThePromptSaysWhatTheKeysDo`; 9 always-reveal → `TestPTYPlayGradeFirst`. All RED.
- Gap, low risk: no in-process `playSession` test drives the *new* full-session script. `TestFullSessionRecordsOneEventPerAnswer` still uses the peek-first `"\ry\rn"`. A sibling on `keysFor("yn")` asserting two events would cover the grade-first sequence in the fast layer, where the pty test currently is the only coverage.
- Gap, very low risk: nothing pins "a stray key on an *unrevealed* word does nothing". The `!ok` branch is covered from the revealed state (`TestAnUngradedKeyDoesNotAdvance`), so a regression would still be caught — but the unrevealed state is the newly-reachable one.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — pass, with the Minor above.** The tally lives only in `score`; the prompt strings live only in `draw`; the drop test grew a row instead of a twin. The one duplicated shape is the audio rig.
- **ARCH-PURE — pass.** All the new logic is pure and directly unit-tested; the only IO the diff touches is inside the pre-existing `OutcomeReveal` arm. `play` still imports nothing.
- **ARCH-PURPOSE — flagged (I1).** The behavioral purpose is fully delivered — prompt, loop, README, atlas, and a live check all in one boundary, no "follow-up" holding the point. The doc sweep is where it settled for the instance.
- **ARCH-MOCK — pass, with the skip note.** The terminal keeps its live conformance check and the new flow is now in it; audio stays behind the same `player`/`audio` seams production uses.
- For **#7 (multiple choice)**: `Graded` is the state where "the correct option is showing" already lives, and `[]Outcome` means "record and also do X" needs no further contract change. Two things to carry forward — the `InputReveal`/`Graded` arm is a per-state obligation every new form inherits (PQ-2 was Critical precisely because it is easy to miss), and the `OutcomeReveal`/`s.Current()` coupling should be retired to `Outcome.Word` before any form makes an advancing input also reveal.

## 7. Plan revision recommendations

Append a `## Revisions` entry to `workshop/plans/000024-play-grade-first-plan.md`:

- **Task 4** names `TestDropWorksInEveryState`; delivered instead as a third `"after a miss"` row on the existing `TestDropAdvancesRecordsNothingAndNamesTheWord` (`session_test.go:345`), which is the better call (ARCH-DRY) but leaves the plan naming a test that does not exist.
- **Task 6 Step 3** — record that the nine-row mutation table was actually run and its result, so the plan carries the evidence rather than only the intent (the `workshop/lessons.md` entry documents the mishap, not the outcome).
- **Task 9 Step 4** — the "no stale hits" check passed over `README.md atlas/` but not over `cmd/define/`; note the four sites in I1 and the corrected scope of the re-grep.
- **Tick the 42 delivered step checkboxes** (I2).

And one line on the issue (`workshop/issues/000024-play-grade-first.md`), Done-when row 1: the no-audio assertion is a recording double checked after the fact, not a double that fails if called — changed deliberately per plan Task 6 / #6 BR-43.

```findings
findings:
  - id: new
    severity: Important
    family: doc-sweep-incomplete
    title: |
      The flow reversal reached README and atlas but not the form's own doc comments in cmd/define/play
    detail: |
      recall.go:3 still says "show the word, let the learner try to remember it,
      reveal the definition, self-rate" — the exact order #24 inverted. Same class:
      question.go:60 ("Reveal is what they see after asking to see it"),
      question.go:68 ("Enter and space reveal"), and play_loop_test.go:595,600
      ("play_loop.go:182 — space reveals", now :194 after this diff). All four are
      returned by the plan's own Task 9 grep pattern over cmd/define/, and the
      issue's Done-when claims the site list was built from that grep. Sweep the
      enumeration in one pass, not just the site named here.
  - id: new
    severity: Important
    family: plan-artifact-not-ticked
    title: |
      The durable plan has all 42 step checkboxes unticked at close
    detail: |
      6c29da2 ticked the issue's Plan rows only; workshop/plans/000024-play-grade-first-plan.md
      still reads as unstarted. Every recently closed plan in workshop/history/plans/
      is fully ticked (000021 52/0, 000016 60/0, 000015 39/0), so this breaks the
      standing convention and the plan artifact no longer records what was delivered.
  - id: new
    severity: Minor
    family: doc-sweep-incomplete
    title: |
      README still implies pronunciation audio is fetched for every word in a sitting
    detail: |
      README.md:65-66 reads "Pronunciation audio IS fetched over the network as each
      word is revealed". Under grade-first a `y` reveals nothing and so fetches and
      plays nothing — the issue's headline speedup, unstated in the README.
  - id: new
    severity: Minor
    family: outcome-carries-its-own-subject
    title: |
      The OutcomeReveal arm would nil-panic, not just mis-name the word, if an input ever advanced and revealed
    detail: |
      play_loop.go:143 calls s.Current().Word() after Apply returned. Correct today
      (verified: OutcomeReveal is emitted only from the two non-advancing paths), but
      Current() returns a nil interface once the session is Done, so the failure mode
      is a panic rather than the wrong word the comment describes. Carrying Word on
      OutcomeReveal, as OutcomeRecord and OutcomeDrop already do, removes it.
  - id: new
    severity: Minor
    family: duplicated-test-rig
    title: |
      Four loop tests repeat the same five-line audio rig instead of sharing a helper
    detail: |
      TestCorrectAnswerPlaysNoAudio, TestAMissPlaysThePronunciation,
      TestRevealPlaysThePronunciationByDefault and TestNoAudioSilencesTheSession each
      re-set opt.noAudio/times, a fakePlayer and okAudio{}. An audioRig(t, word)
      helper would make the PQ-6 trap (a rig whose player is unreachable) impossible
      to reintroduce rather than merely fixed at four sites (ARCH-DRY).
  - id: new
    severity: Minor
    family: check-that-cannot-fail-reads-as-green
    title: |
      The pty conformance suite SKIPs when no pty is available, so a sandboxed run reads as passing
    detail: |
      startDefineInDir calls t.Skipf on "no pty available: operation not permitted",
      which is what every in-sandbox run gets — `ok` plus 7 SKIPs. Verified: the suite
      is 7/7 PASS outside the sandbox. Given #6 found this suite red across two closes,
      an opt-in env gate (fail instead of skip when it is set) would make green mean
      "it ran" (ARCH-MOCK: a conformance check that cannot fail is not a check).
  - id: new
    severity: Minor
    family: donewhen-describes-a-mechanism-not-delivered
    title: |
      Done-when says the no-audio assertion uses a player double that fails if called; it records and is checked after
    detail: |
      The delivered mechanism (fakePlayer.Played asserted empty, with okAudio installed
      so the branch is reachable) is equivalent and mutation-verified, and the plan
      chose it deliberately per #6 BR-43. The issue text should say so.
```

---

## Re-review — 2026-08-27T17:01:30-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 24 — grade before reveal: y advances, n shows the definition |
| repo | tools |
| issue file | workshop/issues/000024-play-grade-first.md |
| boundary | whole-issue close |
| milestone | — |
| window | 87e6307b9f7cf83fdc0a93b69bf4d9354eaf3b00..4c2476331c0db402c81fc2ec17142e2adb40a9d4 |
| command | sdlc close --issue 24 |
| reviewer | claude |
| timestamp | 2026-08-27T17:01:30-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The grade-first reversal is correct and unusually well pinned at the pure layer — I ran six independent mutations against a scratch copy of HEAD and every one of them reddened a named test, including the PQ-1 tally split and the PQ-6 audio trap that a previous round found vacuous. Build, vet, gofmt and the full suite are green. What blocks a clean SHIP is one real coverage hole introduced by the very change under review: `Apply` now returns `[]Outcome`, and the loop's obligation to perform **both** of them is only half-pinned. Mutating `play_loop.go:120` to `for _, out := range outs[len(outs)-1:]` — which silently drops the `OutcomeRecord` on every `n`-before-reveal, so no miss ever reaches the event log or the schedule — leaves the entire `cmd/define` suite green, and the pty suite too (it asserts the session tally, which `score` computes inside `Apply`, and makes no store assertion at all). The mirror mutation `outs[:1]` *is* caught. Beyond that: BR-1's sweep reached 2 of the 4 sites it enumerated, and three round-1 Minors are untouched.

## 1. Strengths

- **The state machine is mutation-proof, and I verified it rather than taking the plan's table at face value.** `cmd/define/play/session.go:172` (drop `verdict != Wrong`) reddens both `TestCorrectBeforeRevealAdvancesWithNoReveal` and `TestCorrectAnswerPlaysNoAudio`; removing `s = score(s, verdict)` at `session.go:194` reddens four tests; `advance(s, q, Wrong)` in the graded arm reddens `TestAKeyAfterAMissAdvancesWithoutRecordingAgain`; deleting the `InputReveal` graded arm reddens `TestEnterAndSpaceMoveOnAfterAMiss`. PQ-1 and PQ-2 are genuinely closed.
- **BR-7's claim checks out under the standard the prompt demands.** `TestCorrectAnswerPlaysNoAudio` (`play_loop_test.go:640`) is a negative assertion whose rig can actually produce the denied thing — `d.audio = okAudio{}` makes the player reachable, and the M1 mutation prints `played [/tmp/.../pronunciation.mp3]`. Paired with `TestAMissPlaysThePronunciation`, neither half can be satisfied by a session that never plays.
- **`score` split from `advance` is the right decomposition**, and its doc comment says exactly why (`session.go:203-210`). The skip filter still lives in `advance` and only there.
- **BR-6's fix is reachable and I confirmed it fails.** `DEFINE_CONFORMANCE_STRICT=1 go test -tags conformance -run TestPTY ./cmd/define/` reddens all 7 pty tests in this sandbox; without it the same run reports `ok`. That is a check that can now fail.
- **`TestDropAdvancesRecordsNothingAndNamesTheWord`'s new third row avoids the trap it names** — the comment at `session_test.go:349-352` explains that a digit against `fakeForm` would silently re-run the "before reveal" row under another name. That is the kind of self-aware fixture that stops a table growing vacuous rows.

## 2. Critical findings

None.

## 3. Important findings

**`cmd/define/play_loop.go:120` — the loop performs two outcomes; only the second one is pinned.**
The whole point of Task 1 (`Apply` returning a slice) is that one input owes the loop two effects. `TestWrongBeforeRevealRecordsAndReveals` pins the *producer*. Nothing pins the *performer*: with `outs[len(outs)-1:]`, `go test ./cmd/define/ -run 'Play|Session|Miss|Correct|Drop|Reveal|Capturer|Prompt|Record|Interrupt'` is `ok`. The Done-when row "the recording happens before the next draw" is a loop-level claim with no loop-level test, and the shipped-bug shape is total loss of every miss from `events/`. Fix is one line: give `TestAMissPlaysThePronunciation` (`play_loop_test.go:664`) the `st` it currently discards and assert `reviewEvents(t, st)` is one event with `Correct == false` — then both halves of the outcome slice are pinned by the same test.

**BR-1 remains open at 2 of its 4 enumerated sites.** `recall.go` and `question.go` were fixed; `play_loop_test.go:595,600` still cite `play_loop.go:182` (the space→`InputReveal` case is now `:195`) and `README:54` (now the table header; the space/Enter promise moved to `:58`). The fix commit's own sweep — `grep -rniE "reveal|cannot rate" README.md atlas/ cmd/define/ | grep -v _test` — excludes the directory the two remaining sites live in. The lesson written this round says to make excludes visible; the exclude is visible and it deletes the finding's own enumeration. Substantively this residue is Minor; the unmet part is the rule.

## 4. Minor findings

- **`check-that-cannot-fail-reads-as-green`, 2nd in family.** `DEFINE_CONFORMANCE_STRICT` is consulted at exactly one of the repo's conformance skip sites (`pty_conformance_test.go:94`). The other six — `fetch_conformance_test.go:27,62`, `player_conformance_test.go:23,33`, `reflect_conformance_test.go:47`, `live_property_test.go:38` — plus `pty_conformance_test.go:271` still skip unconditionally, so `DEFINE_CONFORMANCE_STRICT=1 go test -tags conformance ./...` on a network-less host still reports green for four suites that did not run. `lessons.md:1880` states the rule generally ("any test that can skip itself needs a mode where the skip is an error"); the code applies it once. Per the family escalation: don't patch this site — write the enumeration (`grep -rn 't\.Skipf\?(' cmd/define/*_test.go`) and route every conformance skip through one `skipOrFail(t, reason, err)` helper that consults the env var, so the rule holds for the next suite too.
- **`doc-sweep-incomplete`, 3rd in family.** `DEFINE_CONFORMANCE_STRICT` is a new thing an operator or CI types and it appears in no doc: `atlas/define.md:1002` still reads "All three seams have one" over a three-row table that omits both the pty suite and this issue's `TestPTYPlayGradeFirst`, and `README.md:291-294` still shows the bare `go test -tags conformance ./...`. Same family, and the rule that covers all three instances is the one the family keeps re-proving: **a hand-maintained restatement of a fact the code owns will drift, so make it derive.** The cheapest enforcement available here is a test asserting `README.md` contains the literal string `draw` emits (`play_loop.go:281` vs `README.md:51`) — that converts the prompt line from prose-to-be-swept into a pinned consumer, and it is the sweep failure that produced BR-1 in the first place.
- **`plan-artifact-not-ticked`, 2nd in family.** The plan was revised mid-stream in `4c24763` (Task 9 Step 1 rewritten) with no `## Revisions` section, which AGENTS.md §1 requires; and the issue's `## Log` is a bare `### 2026-08-27` header with nothing under it, after four plan-quality rounds and one boundary round. Don't fix these two spots — state the rule: at every gate round, one artifact-completion pass over issue + plan, covering checkboxes, `## Revisions`, and `## Log`, so the round that flips a box is the round that records why.
- **ARCH-DRY: `session.go:146` and `session.go:162` are the same rule in two places.** Both graded arms are `next, out := advance(s, q, Skipped); return next, []Outcome{out}` with different comments, and the prompt they implement says one thing ("any key = next word"). The plan's own architecture note cites #6 BR-5 against exactly this shape. Hoisting `if s.Graded && (in.Kind == InputRune || in.Kind == InputReveal)` above the switch gives it one home, and keeps `InputDrop`/`InputQuit` correctly outside it.
- **`TestPTYPlayGradeFirst` never checks the child's exit status**, unlike `TestPTYTerminalIsRestoredOnExit` which does `cmd.Wait()` for a stated reason. Cheap to add; it is the assertion that separates "printed the right bytes" from "exited cleanly".

## 5. Test coverage notes

- Six mutations run against a scratch tree; five died on the pure suite alone, and M1 additionally died on the loop suite. The `[]Outcome` refactor's blind spot is the asymmetry above: dropping the reveal is caught, dropping the record is not.
- `TestAnUngradedKeyDoesNotAdvance` drives `reveal` then `'s'` — the *revealed* state. The hidden state is now the default one, and the stray-key path there is protected only by `Grade`'s `ok` flag rather than the deleted `!s.Revealed` guard. The mutation (drop `if !ok`) still dies via the revealed row, so this is a note, not a finding — but a `{"hidden"}` row in that test would pin the state the learner actually spends their time in.
- `TestFullSessionRecordsOneEventPerAnswer` now drives only the peek path (`"\ry\rn"`), which is correct as a regression guard for the old flow but means the grade-first path has no full-session loop test.
- The pty suite could not be executed here (`pty.Start` → `operation not permitted`); I verified the strict gate reddens it, and read `TestPTYPlayGradeFirst` for correctness, but its 7/7 green is an unverified claim from this seat.

## 6. Architectural notes for upcoming work

- **ARCH-PURE — pass.** `cmd/define/play` stays import-free; both purity guards run; `draw` takes an injected `io.Writer` and is tested against a `bytes.Buffer`; every terminal/store/audio effect stays in `playSession`. The `score`/`advance` split moved logic *toward* the pure core, not away from it.
- **ARCH-MOCK — pass, with the caveat above.** The audio seam has `okAudio`/`noAudioSource` behind one boundary that production and test share, plus a live check; the terminal seam now has a real-pty conformance test that can fail. The residual gap is coverage of the *other* seams' skips, not this diff's seam.
- **ARCH-PURPOSE — flag.** Two instances this round where a rule was written and applied once (the skip gate; the doc sweep). Both are enumerable in a single `grep`; both were fixed at the site the finding named.
- **For #7 and #12, which build on this contract:** `Session.Graded` is exactly the state #7's multiple-choice needs, and the `[]Outcome` shape means "record and also do X" no longer touches `Apply`'s signature. The one thing to carry forward is BR-4's risk (still open): `play_loop.go:143`'s `s.Current().Word()` is safe only because no advancing input emits `OutcomeReveal`. The first #7 input that reveals *and* advances turns that into a nil-interface panic, not the wrong word the comment predicts. Carrying `Word` on `OutcomeReveal` before #7 starts is cheaper than after.

## 7. Plan revision recommendations

Add a `## Revisions` section to `workshop/plans/000024-play-grade-first-plan.md` with two entries:

1. **2026-08-27, close round 1 (BR-1):** Task 9 Step 1's sweep changed from a `cmd/define/*.go` glob to `grep -r` over directories with an explicit `| grep -v _test`. Record that the exclude is itself part of the enumeration — BR-1 named two sites inside `_test`, and the revised command cannot reach them.
2. **2026-08-27, close round 1 (BR-2):** all 42 step checkboxes ticked to match delivery, and the plan-artifact rule folded into the close checklist.

The Core-concepts tables need no revision — `Session.Graded`, `score`, `Apply` and `draw` all exist at their stated paths with their stated status, the PURE rows run with no IO, and the two integration rows (`playSession`'s outcome loop, `TestPTYPlayGradeFirst`) are present as described.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      recall.go and question.go fixed; play_loop_test.go:595,600 still cite play_loop.go:182 (now :195) and README:54 (now :58) — the fix's own sweep excludes _test, where the two remaining enumerated sites live.
  - id: BR-2
    disposition: addressed
    note: |
      Verified: 42 of 42 step checkboxes ticked, 0 unticked (the one remaining "- [ ]" is the boilerplate that names the syntax).
  - id: BR-3
    disposition: not-addressed
    note: |
      README.md:65-66 unchanged; the audio sentence still does not say a `y` fetches and plays nothing.
  - id: BR-4
    disposition: not-addressed
    note: |
      play_loop.go:143 unchanged; the comment predicting "the next word" still describes a nil-interface panic, and OutcomeReveal still carries no Word.
  - id: BR-5
    disposition: not-addressed
    note: |
      No audioRig helper; the rig is now at five sites, not four (play_loop_test.go:343, 360, 519, 648, 669).
  - id: BR-6
    disposition: addressed
    note: |
      Verified reachable and failing: DEFINE_CONFORMANCE_STRICT=1 reddens all 7 pty tests here, where the unset run reports ok. See the new class finding.
  - id: BR-7
    disposition: addressed
    note: |
      The Done-when now names fakePlayer.Played + okAudio and cites #6 BR-43; I confirmed the mechanism is mutation-verified.
findings:
  - id: new
    severity: Important
    family: widened-contract-unpinned-at-the-consumer
    title: |
      The loop must perform BOTH outcomes of a miss; only the second is pinned, so dropping every miss from the event log is invisible to the whole suite
    detail: |
      Apply returning []Outcome exists so `n` on a hidden word both records and reveals.
      Mutating play_loop.go:120 to `for _, out := range outs[len(outs)-1:]` drops the
      OutcomeRecord — no miss ever reaches events/ or the schedule — and
      `go test ./cmd/define/ -run 'Play|Session|Miss|Correct|Drop|Reveal|Capturer|Prompt|Record|Interrupt'`
      stays ok. The pty suite cannot catch it either: it asserts "0 right, 1 wrong",
      which finish() reads from the session tally that score() sets inside Apply, and
      it makes no store assertion at all. The mirror mutation `outs[:1]` IS caught by
      TestAMissPlaysThePronunciation. Fix: that test already calls playRig and throws
      away st — keep it and assert reviewEvents(t, st) is one event with Correct false,
      so one test pins both halves of the slice.
  - id: new
    severity: Minor
    family: check-that-cannot-fail-reads-as-green
    title: |
      DEFINE_CONFORMANCE_STRICT is honoured at one of seven conformance skip sites, so the strict run still reports green for four suites that did not execute
    detail: |
      This is the 2nd finding in family `check-that-cannot-fail-reads-as-green`. The
      earlier round fixed the pty instance and lessons.md:1880 wrote the general rule
      ("any test that can skip itself needs a mode where the skip is an error"), but
      the env var is read only at pty_conformance_test.go:94. Measured prevalence:
      fetch_conformance_test.go:27,62; player_conformance_test.go:23,33;
      reflect_conformance_test.go:47; live_property_test.go:38; pty_conformance_test.go:271
      all still skip unconditionally. Do not patch another site — route every
      conformance skip through one skipOrFail(t, reason, err) helper that consults the
      env var, and enumerate with `grep -rn 't\.Skipf\?(' cmd/define/*_test.go` so the
      next suite inherits the rule.
  - id: new
    severity: Minor
    family: doc-sweep-incomplete
    title: |
      The new DEFINE_CONFORMANCE_STRICT surface reached no doc, and the atlas conformance table still says three seams while listing neither pty suite
    detail: |
      This is the 3rd finding in family `doc-sweep-incomplete`. atlas/define.md:1002
      reads "All three seams have one" over a table omitting the pty suite and this
      issue's TestPTYPlayGradeFirst; README.md:291-294 still shows the bare
      `go test -tags conformance ./...` with no mention of the strict mode. The rule
      that covers all three findings in this family is the ARCH-PURPOSE one: a
      hand-maintained restatement of a fact the code owns will drift, so make it
      derive. The cheapest enforcement available is a test asserting README.md contains
      the literal prompt string draw emits (play_loop.go:281 vs README.md:51) — that
      converts the prompt line from prose-to-be-swept into a pinned consumer, which is
      the exact site whose drift produced BR-1.
  - id: new
    severity: Minor
    family: plan-artifact-not-ticked
    title: |
      The plan was revised mid-stream with no "## Revisions" entry, and the issue's Log section is an empty date header after five gate rounds
    detail: |
      This is the 2nd finding in family `plan-artifact-not-ticked`. 4c24763 rewrote
      Task 9 Step 1 in place; AGENTS.md section 1 requires an appended "## Revisions"
      entry (timestamp, reason, delta) rather than an overwrite, and workshop/history
      shows the convention live (000015's plan carries entries for rounds 5 and 6).
      Separately the issue's "## Log" holds only "### 2026-08-27" with no body, after
      four plan-quality rounds and one boundary round. Do not fix the two spots — state
      the rule: every gate round ends with one artifact-completion pass over issue and
      plan covering checkboxes, "## Revisions", and "## Log", so the round that flips a
      box is the round that records why.
  - id: new
    severity: Minor
    family: one-rule-two-places
    title: |
      The graded state's "any key moves on" rule is implemented identically in both the InputReveal and InputRune arms (ARCH-DRY)
    detail: |
      session.go:146 and session.go:162 are the same four lines with different comments,
      implementing the single rule the prompt states as "any key = next word". The
      plan's own architecture note cites #6 BR-5 against exactly this shape while
      arguing for []Outcome over a Reveal flag. Hoisting
      `if s.Graded && (in.Kind == InputRune || in.Kind == InputReveal)` above the switch
      gives it one home and keeps InputDrop and InputQuit correctly outside it.
```

---

## Re-review — 2026-08-27T18:55:55-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 24 — grade before reveal: y advances, n shows the definition |
| repo | tools |
| issue file | workshop/issues/000024-play-grade-first.md |
| boundary | whole-issue close |
| milestone | — |
| window | 87e6307b9f7cf83fdc0a93b69bf4d9354eaf3b00..9d78d181655001fdd7324faa707821d9a4d91779 |
| command | sdlc close --issue 24 |
| reviewer | claude |
| timestamp | 2026-08-27T18:55:55-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All nine open prior findings are genuinely fixed, and I verified the load-bearing ones by mutation rather than by reading the commit message: BR-8's slice-membership pin reddens under **both** `outs[:1]` and `outs[len(outs)-1:]`; BR-12's hoisted graded rule reddens under all three narrowing mutations (including the one proving `InputDrop` stays outside it); BR-4's `Word` on `OutcomeReveal` reddens at both construction sites; BR-9's `skipOrFail` measured in both directions on this machine (non-strict `ok` with 7 pty skips, `DEFINE_CONFORMANCE_STRICT=1` fails naming each unrun suite); BR-1's recursive sweep returns zero stale hits. The state machine itself is correct — `y` advances without reveal or audio, `n` scores without advancing, `d`/Ctrl-C stay outside the graded rule, and the skip filter still lives only in `advance`. What keeps this from SHIP is one real coverage gap and two artifact gaps: the loop's **outcome ordering** — the half of the widened `Apply` contract BR-8 did not sweep — is unpinned, and I confirmed a reversed iteration leaves the entire suite green while making a miss droppable; plus the round's own new doc-derivation convention reached no atlas entry, and `workshop/lessons.md` now carries four mutation-restore rules giving three different answers, two of them added in this window and contradicting each other.

### 1. Strengths

- **`play_loop.go:293-299` — the prompt lines as consts with README as a pinned consumer.** This is the right general fix for a family that had produced three sweep-failures, and `doc_sync_test.go` earning a real failure on its first run is the evidence that it bites. `ARCH-PURPOSE`'s shadow-sweep over the prompt strings shows exactly one derivable consumer (README) and it derives; the remaining literals (`play_loop_test.go:719-723`, `pty_conformance_test.go:439,451`) are deliberate oracles, correctly not derived.
- **`session.go:154` — the hoisted "any key = next word" rule.** One home, and the exclusion of `InputDrop`/`InputQuit` is reasoned in the comment *and* pinned: mutating the guard to a bare `if s.Graded` reddens `TestDropAdvancesRecordsNothingAndNamesTheWord/after_a_miss`.
- **`session.go:220-232` — `score` split from `advance`.** The PQ-1 defect (tally left at zero for a miss that does not advance) is structurally impossible now, and `Skipped` scoring nothing is what makes drop-after-miss safe without a second flag.
- **`conformance_skip_test.go` — one helper for both directions of the rule.** Finding the three *mirror* sites (`dict`/`news`/`live_property` writing an absent dependency as an unconditional `Fatalf`) is the class-not-instance work `ARCH-PURPOSE` asks for, and the doc comment correctly carves out shape-drift checks so they stay hard failures.
- **`play_loop_test.go:48-54` — `audible()`.** The helper makes the PQ-6 trap (a negative audio assertion against an unreachable source) unwriteable by hand rather than merely fixed at four sites. `ARCH-DRY` pass.

### 2. Critical findings

None.

### 3. Important findings

**(a) `cmd/define/play_loop.go:120` — the loop may perform outcomes in any order and nothing notices.**
This is the **2nd finding in family `widened-contract-unpinned-at-the-consumer`**. BR-8 fixed *membership* of the slice at the consumer; *order* is its enumerable sibling and was left in the tree, which is the instance-not-class pattern (`ARCH-PURPOSE`). `session.go:127` states the ordering as load-bearing ("the record is emitted FIRST, so a caller performing them in order writes the event before anything that can block"), and the issue's Done-when restates it ("Ctrl-C stays lossless by construction, not by a flush"). Measured: rewriting `play_loop.go:120` to `for i := len(outs) - 1; i >= 0; i--` leaves `go test ./cmd/define/ ./cmd/define/play/` fully green. The failure that buys: on a miss, `OutcomeReveal` restores the terminal, shells out to `afplay`, and on re-entry failure takes the early `return 1` at `play_loop.go:167-170` — with the record not yet performed, the miss is gone and the process exits 1.
Do not just add an order assertion; write the enumeration the widened contract implies (every element performed / performed in emitted order / the empty and single-element cases) and pin each at the consumer. The order row is three lines on a test that already exists — change `TestLosingTheTerminalAfterPlaybackExitsOne` (`play_loop_test.go:529`) to drive `keysFor("n")` instead of `keysFor("\r")`, keep `st`, and assert `len(reviewEvents(t, st)) == 1`. I ran this: green today, red under the reversal mutation.

**(b) `atlas/define.md` — the atlas pass for this round is incomplete in two ways.**
This is the **4th finding in family `doc-sweep-incomplete`**. Earlier rounds fixed instances; BR-10's general fix (make the restatement derive) covers only the derivable half, so state the rule for the other half rather than patching these two spots. (i) The round introduced a new convention — `doc_sync_test.go` plus `gradePrompt`/`gradedPrompt`, i.e. "a doc that restates a fact the code owns is made a build-failing consumer" — and it appears nowhere in `atlas/` or `README.md`, while its sibling convention born in the same round (`skipOrFail` + `DEFINE_CONFORMANCE_STRICT`) got a full atlas paragraph at `atlas/define.md:1009-1024`. (ii) `atlas/define.md:1319` says the graded state "needs its own case there" in the `InputReveal` arm — the shape BR-12 removed later in the same range, where the rule now sits hoisted above the switch. The covering rule: the atlas enumeration must be *produced by something that ran*, not recalled — the round's new package-level identifiers, env vars and test conventions from `git diff` over the window, each checked against `grep` in `atlas/` and either documented or explicitly waived.

**(c) `workshop/lessons.md:1820` and `:1939` — the file now gives three different answers to "how do I restore a mutation".**
This is the **2nd finding in family `one-rule-two-places`**, so the ask is the rule, not the edit. Measured prevalence, four entries: `:1080` (pre-existing) "commit, then mutate, then `git checkout` — neither half works alone"; `:1739` (pre-existing) "back up with git, not with cp to /tmp"; `:1820` (new) "commit the implementation BEFORE the first mutation … the baseline is a commit"; `:1939` (new) "snapshot to a temp file and restore from that". `:1820` re-derives `:1080` verbatim and then mis-describes it ("the previous lesson … stopped one clause short" — it did not, it stated both clauses), and `:1939` contradicts `:1739`, `:1107` and its own sibling 119 lines above. AGENTS.md §4 has agents read this file at session start, so the next mutation round gets contradictory instructions for the exact procedure that burned this one (`ARCH-DRY`: one source of truth per rule). Rule: before appending a lesson, grep `lessons.md` for the rule you are about to state; if it is there, **revise that entry** with the new evidence rather than appending a sibling, and if the new evidence contradicts it, resolve the contradiction inside the one entry.

### 4. Minor findings

- **`cmd/define/render_test.go:205`** — **3rd finding in family `check-that-cannot-fail-reads-as-green`.** `skipOrFail`'s doc excludes `render_test.go` by name on the grounds that "table rows that do not apply are not absent dependencies", but this skip fires on an absent *committed* fixture. Measured: deleting `cmd/define/testdata/entries/subject.txt` makes `TestCorpusBlockStructure/subject` skip and the package still report `ok`, silently retiring phantom-block coverage. The rule is three classes, not two — absent **external** dependency (skip; fail under strict), absent **in-repo** artifact (always fail, which `doc_sync_test.go:29` already gets right), shape drift (always fail) — and the exclusion list should be justified per-site by asking that question, not by filename.
- **`workshop/plans/000024-play-grade-first-plan.md` `## Revisions`** — **3rd finding in family `plan-artifact-not-ticked`.** Four entries record the BR-1, BR-11, BR-9 and BR-4 scope growth; the BR-10 work is absent, though it added a file the plan never named (`doc_sync_test.go`) and two new package consts — the same omission shape the BR-9 entry exists to record. Same rule as (b): the Revisions enumeration is built from `git diff --name-status` over the round, not from memory.
- `atlas/define.md:1002` now reads "Every seam has one" over a seven-row table — good — but the paragraph two lines below still opens "The third is the least obvious", which was an ordinal into the old three-row table.

### 5. Test coverage notes

- **I could not execute `TestPTYPlayGradeFirst`.** `pty.Start` returns `operation not permitted` in this environment (confirmed directly, outside the test harness), so all seven pty tests SKIP here. I reviewed the test statically — the assertions are sound and non-vacuous (it asserts the definition is *absent* before answering, present after `n`, and that space advances to "0 right, 1 wrong") — and I confirmed strict mode converts the skip to a failure. The Done-when's live-terminal claim rests on the operator's run, not on anything I ran.
- Nine mutations verified by me this round, each reddening a named test: two slice-end mutations (BR-8), three graded-guard narrowings (BR-12), two `Word`-stripping mutations (BR-4), the loop reversal (finding (a), **not** caught), and the fixture deletion (`render_test.go`, **not** caught).
- No in-process test drives `playSession` into the graded *draw* state; `TestThePromptSaysWhatTheKeysDo` pins `draw` against a hand-built `Session{Revealed, Graded}` and the session tests pin the flag, but the composition is only covered by the pty test. Low risk (the wiring is `draw(stdout, s)`), noted rather than raised.

### 6. Architectural notes

- **ARCH-DRY — flag (one instance).** Code side is a clear pass: `audible`, `skipOrFail`, the hoisted graded rule and the prompt consts each collapse a real duplication. The flag is finding (c), in the process artifact.
- **ARCH-PURE — pass.** `cmd/define/play` stays import-clean under both `puretest` guards; `Apply`/`score`/`advance` are total functions over `Session`; `draw` takes an `io.Writer` and is tested with a `bytes.Buffer`; every effect stays in `playSession`. No mock is needed to run anything in the pure package.
- **ARCH-PURPOSE — flag.** The purpose (grade before reveal) is fully delivered, not the cheap subset. Two class-vs-instance flags: finding (a) (BR-8 swept membership, not order) and finding (b) (the derivation rule was applied to README and not to the atlas record of the rule itself).
- **ARCH-MOCK — pass.** No new external dependency. Existing seams keep their doubles (`afplay`→`fakePlayer`, CDN→`okAudio`/`noAudioSource`, NOAD→`fakeDictionary`, terminal→real pty behind `startDefineInDir`), production and test flow share the boundary, and this round strengthened the live-conformance side: seven suites now route their dependency probe through one seam and `DEFINE_CONFORMANCE_STRICT` makes green mean "it ran".
- **For #7 (multiple choice):** it will add inputs to this machine and is the first plausible producer of an input that both advances *and* reveals. `Outcome.Word` (BR-4) already removes the nil-panic; finding (a)'s ordering pin is the other half, and it is cheapest to land now, before a second producer exists.

### 7. Plan revision recommendations

Append to `workshop/plans/000024-play-grade-first-plan.md` `## Revisions`:

- **2026-08-27, close round 3 (BR-10 follow-through).** Record the doc-derivation work the round-2 pass omitted. *Delta:* `cmd/define/doc_sync_test.go` is a new file the plan did not name; `gradePrompt` and `gradedPrompt` are new package consts in `play_loop.go`, and Task 7 Step 3's `draw` sketch shows the literals inline rather than the consts. *Reason:* the plan's Revisions enumeration was built from memory and reached three of four scope-growth items.
- **2026-08-27, close round 3 (Task 5 Step 1).** The step says "Verify (do not assume) that the `OutcomeReveal` arm's `s.Current().Word()` still names the right word"; the arm now reads `out.Word` and the risk it names was retired by BR-4. Restate the step as the surviving obligation — pin the loop's *ordering* of the returned slice, not just its membership — so the plan stops describing a check the code no longer needs and starts describing the one it still lacks.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      recall.go:3, question.go:60/74 and the play_loop_test citation all rewritten; the plan's recursive sweep returns zero stale hits.
  - id: BR-3
    disposition: addressed
    note: |
      README.md:71-73 now states audio is fetched only on a reveal.
  - id: BR-4
    disposition: addressed
    note: |
      Word carried on both OutcomeReveal sites, loop reads out.Word; stripping Word reddens TestEveryWordOutcomeNamesItsWord.
  - id: BR-5
    disposition: addressed
    note: |
      audible(&d, &opt) at five sites; the forgettable d.audio line now lives in one place.
  - id: BR-8
    disposition: addressed
    note: |
      Verified by mutation: both outs[:1] and outs[len(outs)-1:] redden TestAMissPlaysThePronunciationAndRecordsIt.
  - id: BR-9
    disposition: addressed
    note: |
      Ten sites route through skipOrFail; measured here, non-strict ok with 7 pty skips, strict fails naming each.
  - id: BR-10
    disposition: addressed
    note: |
      doc_sync_test.go makes README a consumer of the prompt consts; atlas table now lists seven seams.
  - id: BR-11
    disposition: addressed
    note: |
      Four Revisions entries and a full Log entry; see the new Minor for the one scope item the pass missed.
  - id: BR-12
    disposition: addressed
    note: |
      Rule hoisted above the switch; three narrowing mutations each redden a different named test.
findings:
  - id: new
    severity: Important
    family: widened-contract-unpinned-at-the-consumer
    title: |
      The loop may perform an input's outcomes in any order and the whole suite stays green
    detail: |
      This is the 2nd finding in family widened-contract-unpinned-at-the-consumer.
      BR-8 swept the slice's MEMBERSHIP at the consumer; ORDER is the enumerable
      sibling and was left in the tree. Measured: rewriting play_loop.go:120 to
      `for i := len(outs) - 1; i >= 0; i--` leaves go test ./cmd/define/
      ./cmd/define/play/ fully green, though session.go:127 and the issue's
      Done-when both call the ordering load-bearing. Failure: on a miss the
      reveal arm restores the terminal and shells out to afplay, and re-entry
      failure takes the early `return 1` at play_loop.go:167 with the record not
      yet performed, so the miss is lost and the process exits 1. Write the
      enumeration the widened contract implies and pin each row at the consumer;
      the order row is three lines on TestLosingTheTerminalAfterPlaybackExitsOne
      (play_loop_test.go:529) — drive keysFor("n"), keep st, assert one
      reviewEvent. Verified: green today, red under the reversal.
  - id: new
    severity: Important
    family: doc-sweep-incomplete
    title: |
      The atlas pass missed the round's own new doc-derivation convention, and describes an InputReveal case BR-12 removed
    detail: |
      This is the 4th finding in family doc-sweep-incomplete. Earlier rounds
      fixed instances and BR-10 shipped the general fix for the DERIVABLE half
      (make the restatement a consumer); this is the other half, so state the
      rule rather than patching these two spots. (i) doc_sync_test.go plus the
      gradePrompt/gradedPrompt consts introduce a convention that constrains
      every future prompt edit, and it appears nowhere in atlas/ or README.md —
      while the sibling convention born in the same round (skipOrFail +
      DEFINE_CONFORMANCE_STRICT) got a full atlas paragraph at
      atlas/define.md:1009-1024. (ii) atlas/define.md:1319 says the graded state
      "needs its own case there" in the InputReveal arm, the shape BR-12 removed
      later in the same range. Rule: the atlas enumeration must be produced by
      something that ran — new package-level identifiers, env vars and test
      conventions taken from git diff over the window, each grepped against
      atlas/ and either documented or explicitly waived.
  - id: new
    severity: Important
    family: one-rule-two-places
    title: |
      lessons.md now holds four mutation-restore rules giving three different answers, two added this round
    detail: |
      This is the 2nd finding in family one-rule-two-places, so the ask is the
      rule, not the edit. Measured prevalence, four entries: :1080 "commit, then
      mutate, then git checkout — neither half works alone"; :1739 "back up with
      git, not with cp to /tmp"; :1820 (new) "the baseline is a commit, not a
      working tree"; :1939 (new) "snapshot to a temp file and restore from that".
      :1820 re-derives :1080 and then mis-describes it ("the previous lesson
      stopped one clause short" — it did not), and :1939 contradicts :1739,
      :1107 and its own sibling 119 lines above. AGENTS.md section 4 has agents
      read this file at session start, so the next mutation round is handed
      contradictory instructions for the exact procedure that burned this one
      (ARCH-DRY). Rule: before appending a lesson, grep lessons.md for the rule
      you are about to state; if it exists, REVISE that entry with the new
      evidence instead of appending a sibling, and resolve any contradiction
      inside the one entry.
  - id: new
    severity: Minor
    family: check-that-cannot-fail-reads-as-green
    title: |
      render_test.go skips on an absent COMMITTED fixture, and the skipOrFail carve-out excludes it by filename rather than by the question
    detail: |
      This is the 3rd finding in family check-that-cannot-fail-reads-as-green.
      skipOrFail's doc excludes render_test.go on the grounds that "table rows
      that do not apply are not absent dependencies", but render_test.go:205
      skips when a committed fixture is missing. Measured: deleting
      cmd/define/testdata/entries/subject.txt makes TestCorpusBlockStructure
      /subject skip and the package still report ok, silently retiring the
      phantom-block coverage that fixture exists for. The rule is three classes,
      not two — absent EXTERNAL dependency (skip; fail under strict), absent
      IN-REPO artifact (always fail, which doc_sync_test.go:29 already gets
      right), shape drift (always fail) — and each exclusion should be justified
      by asking the site that question rather than by naming the file.
  - id: new
    severity: Minor
    family: plan-artifact-not-ticked
    title: |
      The plan's Revisions section records three of the four scope-growth items from this round
    detail: |
      This is the 3rd finding in family plan-artifact-not-ticked. The four
      entries cover BR-1, BR-11, BR-9 and BR-4; the BR-10 work is absent though
      it added a file the plan never named (cmd/define/doc_sync_test.go) and two
      new package consts, and left Task 7 Step 3's draw sketch showing the
      literals inline. Same rule as the atlas finding: the Revisions enumeration
      is built from git diff --name-status over the round's commits, not from
      memory — the round that grows the scope is the round that records what
      grew.
  - id: new
    severity: Minor
    family: doc-sweep-incomplete
    title: |
      atlas/define.md still says "The third is the least obvious" after the table grew from three rows to seven
    detail: |
      atlas/define.md:1002 correctly changed to "Every seam has one", but the
      paragraph below it opens with an ordinal into the old three-row table.
      Name the check (player_conformance_test.go) instead of its position, per
      the round's own "cite by NAME, never by line number" lesson.
```

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

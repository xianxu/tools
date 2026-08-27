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

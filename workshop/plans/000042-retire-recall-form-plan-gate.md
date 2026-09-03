---
gate: plan-quality
issue: 42
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-02T17:33:07-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: 'packBoards applies shrink-or-skip to every triage word, deleting #40 D15''s form-2.3 fallback for mature words'
          detail: |-
            Task 1 Step 5 justifies replacing D15's fallback with shrink-then-skip because "2.3 is
            unavailable for exactly the words newly arriving here" — true only of box <= 2 words with no
            distractors, but applied to box >= 3 words too. boardFitsIn (play_loop.go:608) refuses any
            board when termCols < minWrapWidth (20, playbar.go:183), so on a 19-column or sub-5-row
            terminal every triage word is skipped; on a mature deck that is every due word, and
            todaysQuestions then prints "N words are due but none could be looked up" (play_loop.go:999),
            which is false. Today the same terminal runs a full sitting on singles
            (TestAShortTerminalGetsMeaningChoiceNotAClippedBoard, play_loop_test.go:2507). Skip only a
            word that can be neither drawn nor built into a 2.3.
          family: widening-outruns-its-population
          round: 1
        - id: PQ-2
          severity: Important
          title: Spec claims store.Forget has one caller and that a learner "could not delete a typo'd capture at all" — both false
          detail: |-
            main.go:425 registers the -forget flag and main.go:1032 calls d.deck.Forget via forgetWord;
            play_loop.go:452's own comment names it "--forget's path". So Forget has two production
            callers and a CLI drop path already exists. The whole Task 3 widening was put to the operator
            on this premise. Correct the Spec, state what is actually lost (leaving the sitting), and let
            the operator re-decide.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-3
          severity: Important
          title: A third mode state on the board's keys row has two columns of headroom, and that width feeds fitsABoard
          detail: |-
            gradePrompt(board) is Keys() + ", " + quitKey = 62 + 2 + 14 = 78 columns at 80, and board.go:458
            pins all mode spellings to one width. boardFitsIn charges displayRows(gradePrompt(q), termCols)
            into fitsABoard, so a wider three-state row wraps at 80 and raises the minimum terminal height
            for every board — on the path this issue routes untestable young words onto, where the fallback
            is now a skip. TestTheRefusalRowIsNoWiderThanTheKeysRow will not catch it.
          family: prompt-row-is-a-budget
          round: 1
        - id: PQ-4
          severity: Important
          title: The loop already prints a per-drop transcript line, so the planned board `dropped:` line is a second owner
          detail: |-
            play_loop.go:460 writes "\nremoved %q from the deck\n" to stdout on every OutcomeDrop. Task 3
            Step 5 adds a `dropped:` line at board close as relearnLine's sibling. That is two statements of
            one fact, and the existing one is a buffer write landing mid-board — exactly what relearnLine's
            "WRITTEN AS THE BOARD CLOSES" reasoning exists to avoid. Say which survives on a board.
          family: two-owners-of-one-fact
          round: 1
        - id: PQ-5
          severity: Important
          title: 'Deleting boardsFor removes four #40 pins with no named replacement for two of them'
          detail: |-
            boardsFor is called at play_loop_test.go:2446, 2459, 2475, 2517, 2536 across
            TestTheBoxPicksTheForm (:2439), TestBoardsArePackedToTheLabelAlphabet (:2466) and
            TestAShortTerminalGetsMeaningChoiceNotAClippedBoard (:2507) — #40's Done-when 3 and 10.
            TestUntestableWordsReachABoardAtEveryBox replaces only the box-threshold half. Say that the
            sixteen-word packing pin and the short-terminal pin are re-pointed at packBoards/formFor.
          family: deleted-function-drops-its-pins
          round: 1
        - id: PQ-6
          severity: Minor
          title: The plan does not say where triaged young words land in the queue, or whether they share a board with mature ones
          detail: |-
            play_loop.go:825 records singles-first-then-boards as a deliberate choice ("retrieval gets the
            learner's freshest attention"). Under the new rule an untestable young word moves from the
            retrieval-first half to the sweep tail, possibly onto a board of box >= 3 words. State the
            decision.
          family: unstated-ordering-decision
          round: 1
        - id: PQ-7
          severity: Minor
          title: Task 2's per-file counts are wrong and four production files naming Recall are in no task's file list
          detail: |-
            session_test.go has 20 Recall references (plan says 16), play_loop_test.go 14 (plan says 8),
            atlas/define.md 6 (plan says ~10). More usefully: choice.go:16 (which cites recall.go:19 by
            file:line), board.go:121, play_loop.go:226 and optionpool.go:14 carry production comments
            naming Recall and appear in no task's Files list. Task 4 Step 3's tree-wide grep is the right
            instrument; point it at these too rather than keeping counts the compiler will supersede.
          family: stale-inventory-in-plan
          round: 1
        - id: PQ-8
          severity: Minor
          title: The ARCH-CONSTRAINTS block names the Render saved but not the Render newly paid on every board word
          detail: |-
            todaysQuestions' board loop (play_loop.go:971-990) does one Lookup + ParseEntry +
            targetCandidate and no Render at all. A unified loop that renders every due word pays a Render
            and a click-region map entry per mature board word. Bounded by opt.count so it is small — but
            the cost table should say so rather than only naming the saving.
          family: one-directional-cost-claim
          round: 1
        - id: PQ-9
          severity: Minor
          title: Done-when 5's "no longer than it was" has no in-tree baseline once Recall is deleted
          detail: |-
            TestRetiringRecallDoesNotLengthenAYoungSitting asserts "strictly fewer or equal" without
            naming what to. Name the baseline explicitly — one screen per word, which is what form 2.1
            gave — so the assertion survives the deletion.
          family: undefined-acceptance-baseline
          round: 1
        - id: PQ-10
          severity: Minor
          title: Where Dropping is consulted decides whether a drop can be emitted twice
          detail: |-
            The plan says Apply asks Dropped() "after Grade/Mark" but cites missedAxis, which lives in
            advance. Asked from the outer Apply, a board whose last mark was Dropped re-emits OutcomeDrop
            on the next Tab or refused click, breaking the "once — no outcome is performed twice"
            obligation the loop enumerates at play_loop.go:~470. Say it is asked on the successful
            Grade/Mark path, or that Dropped() is one-shot.
          family: capability-consumption-unstated
          round: 1
      blocked: true
---

# Gate ledger — tools#42 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-02T17:33:07-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `widening-outruns-its-population` packBoards applies shrink-or-skip to every triage word, deleting #40 D15's form-2.3 fallback for mature words
  Task 1 Step 5 justifies replacing D15's fallback with shrink-then-skip because "2.3 is
  unavailable for exactly the words newly arriving here" — true only of box <= 2 words with no
  distractors, but applied to box >= 3 words too. boardFitsIn (play_loop.go:608) refuses any
  board when termCols < minWrapWidth (20, playbar.go:183), so on a 19-column or sub-5-row
  terminal every triage word is skipped; on a mature deck that is every due word, and
  todaysQuestions then prints "N words are due but none could be looked up" (play_loop.go:999),
  which is false. Today the same terminal runs a full sitting on singles
  (TestAShortTerminalGetsMeaningChoiceNotAClippedBoard, play_loop_test.go:2507). Skip only a
  word that can be neither drawn nor built into a 2.3.
- **PQ-2** [Important] `unbacked-existing-behavior-claim` Spec claims store.Forget has one caller and that a learner "could not delete a typo'd capture at all" — both false
  main.go:425 registers the -forget flag and main.go:1032 calls d.deck.Forget via forgetWord;
  play_loop.go:452's own comment names it "--forget's path". So Forget has two production
  callers and a CLI drop path already exists. The whole Task 3 widening was put to the operator
  on this premise. Correct the Spec, state what is actually lost (leaving the sitting), and let
  the operator re-decide.
- **PQ-3** [Important] `prompt-row-is-a-budget` A third mode state on the board's keys row has two columns of headroom, and that width feeds fitsABoard
  gradePrompt(board) is Keys() + ", " + quitKey = 62 + 2 + 14 = 78 columns at 80, and board.go:458
  pins all mode spellings to one width. boardFitsIn charges displayRows(gradePrompt(q), termCols)
  into fitsABoard, so a wider three-state row wraps at 80 and raises the minimum terminal height
  for every board — on the path this issue routes untestable young words onto, where the fallback
  is now a skip. TestTheRefusalRowIsNoWiderThanTheKeysRow will not catch it.
- **PQ-4** [Important] `two-owners-of-one-fact` The loop already prints a per-drop transcript line, so the planned board `dropped:` line is a second owner
  play_loop.go:460 writes "\nremoved %q from the deck\n" to stdout on every OutcomeDrop. Task 3
  Step 5 adds a `dropped:` line at board close as relearnLine's sibling. That is two statements of
  one fact, and the existing one is a buffer write landing mid-board — exactly what relearnLine's
  "WRITTEN AS THE BOARD CLOSES" reasoning exists to avoid. Say which survives on a board.
- **PQ-5** [Important] `deleted-function-drops-its-pins` Deleting boardsFor removes four #40 pins with no named replacement for two of them
  boardsFor is called at play_loop_test.go:2446, 2459, 2475, 2517, 2536 across
  TestTheBoxPicksTheForm (:2439), TestBoardsArePackedToTheLabelAlphabet (:2466) and
  TestAShortTerminalGetsMeaningChoiceNotAClippedBoard (:2507) — #40's Done-when 3 and 10.
  TestUntestableWordsReachABoardAtEveryBox replaces only the box-threshold half. Say that the
  sixteen-word packing pin and the short-terminal pin are re-pointed at packBoards/formFor.
- **PQ-6** [Minor] `unstated-ordering-decision` The plan does not say where triaged young words land in the queue, or whether they share a board with mature ones
  play_loop.go:825 records singles-first-then-boards as a deliberate choice ("retrieval gets the
  learner's freshest attention"). Under the new rule an untestable young word moves from the
  retrieval-first half to the sweep tail, possibly onto a board of box >= 3 words. State the
  decision.
- **PQ-7** [Minor] `stale-inventory-in-plan` Task 2's per-file counts are wrong and four production files naming Recall are in no task's file list
  session_test.go has 20 Recall references (plan says 16), play_loop_test.go 14 (plan says 8),
  atlas/define.md 6 (plan says ~10). More usefully: choice.go:16 (which cites recall.go:19 by
  file:line), board.go:121, play_loop.go:226 and optionpool.go:14 carry production comments
  naming Recall and appear in no task's Files list. Task 4 Step 3's tree-wide grep is the right
  instrument; point it at these too rather than keeping counts the compiler will supersede.
- **PQ-8** [Minor] `one-directional-cost-claim` The ARCH-CONSTRAINTS block names the Render saved but not the Render newly paid on every board word
  todaysQuestions' board loop (play_loop.go:971-990) does one Lookup + ParseEntry +
  targetCandidate and no Render at all. A unified loop that renders every due word pays a Render
  and a click-region map entry per mature board word. Bounded by opt.count so it is small — but
  the cost table should say so rather than only naming the saving.
- **PQ-9** [Minor] `undefined-acceptance-baseline` Done-when 5's "no longer than it was" has no in-tree baseline once Recall is deleted
  TestRetiringRecallDoesNotLengthenAYoungSitting asserts "strictly fewer or equal" without
  naming what to. Name the baseline explicitly — one screen per word, which is what form 2.1
  gave — so the assertion survives the deletion.
- **PQ-10** [Minor] `capability-consumption-unstated` Where Dropping is consulted decides whether a drop can be emitted twice
  The plan says Apply asks Dropped() "after Grade/Mark" but cites missedAxis, which lives in
  advance. Asked from the outer Apply, a board whose last mark was Dropped re-emits OutcomeDrop
  on the next Tab or refused click, breaking the "once — no outcome is performed twice"
  obligation the loop enumerates at play_loop.go:~470. Say it is asked on the successful
  Grade/Mark path, or that Dropped() is one-shot.

## Open findings

- **PQ-1** [Critical] `widening-outruns-its-population` packBoards applies shrink-or-skip to every triage word, deleting #40 D15's form-2.3 fallback for mature words
- **PQ-2** [Important] `unbacked-existing-behavior-claim` Spec claims store.Forget has one caller and that a learner "could not delete a typo'd capture at all" — both false
- **PQ-3** [Important] `prompt-row-is-a-budget` A third mode state on the board's keys row has two columns of headroom, and that width feeds fitsABoard
- **PQ-4** [Important] `two-owners-of-one-fact` The loop already prints a per-drop transcript line, so the planned board `dropped:` line is a second owner
- **PQ-5** [Important] `deleted-function-drops-its-pins` Deleting boardsFor removes four #40 pins with no named replacement for two of them
- **PQ-6** [Minor] `unstated-ordering-decision` The plan does not say where triaged young words land in the queue, or whether they share a board with mature ones
- **PQ-7** [Minor] `stale-inventory-in-plan` Task 2's per-file counts are wrong and four production files naming Recall are in no task's file list
- **PQ-8** [Minor] `one-directional-cost-claim` The ARCH-CONSTRAINTS block names the Render saved but not the Render newly paid on every board word
- **PQ-9** [Minor] `undefined-acceptance-baseline` Done-when 5's "no longer than it was" has no in-tree baseline once Recall is deleted
- **PQ-10** [Minor] `capability-consumption-unstated` Where Dropping is consulted decides whether a drop can be emitted twice

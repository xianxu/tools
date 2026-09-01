---
gate: plan-quality
issue: 40
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-01T09:58:48-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: D5 makes a click record a verdict, reversing a stated loop invariant, and the region machinery cannot reach the live edge anyway
          detail: 'play_loop.go:258 states "A CLICK ACTS AND NEVER ANSWERS ... it stops before toInput, so play.Apply never learns that a mouse exists"; D5 inverts that without saying so. It is also unbuildable as drawn: RegionAtRow resolves through screen.LineAt (screen.go:196, :663-673), which returns false for prompt and footer rows, so a grid on D6''s live edge is unclickable, while a grid in the buffer cannot show marks landing. Board cells would additionally collide with the RegionHeadword audio target at play_loop.go:178, :268-274.'
          family: click-never-answers-invariant
          round: 1
        - id: PQ-2
          severity: Critical
          title: Gating advance on Spent() is not enough — three other Apply paths and livePrompt assume one word per question
          detail: A No mark grades Wrong on an unrevealed board, hitting session.go:204/230, which sets s.Graded = true. The next mark then trips the hoist at session.go:169 and calls advance(..., Skipped), abandoning the rest of the board. Enter hits session.go:192 and returns OutcomeNone instead of spending Rest(Wrong), and livePrompt (play_loop.go:639-643) switches to "any key = next word". T1 as scoped does not make a board usable.
          family: apply-assumes-one-word-per-question
          round: 1
        - id: PQ-3
          severity: Critical
          title: Core concepts, Integration points, Tasks and Done-when still specify the unsure mark, the cursor, and a Mastered-based threshold that D4, D5 and D7 delete
          detail: T3 builds EventUnsure, CaptureReview writing it and Fold ignoring it; T4 routes "a word with a recent unsure" to 2.3; Done-when row 4 pins TestUnsureDoesNotMoveTheBoxAndForcesARealTest — all deleted by D7. T2 and the Board entry render "with the cursor" and Word() returns "the CURSOR's word", deleted by D5. boardsFor partitions "by Mastered" (box >= 9, progress.go:39/126) against D4's box >= 3. Every task's parenthetical decision reference is also off by the deleted decisions (T3 cites D4, T4 cites D6, T6 cites D7, T8 cites D8). An implementer executing the Tasks section builds the superseded design.
          family: plan-body-stale-after-revision
          round: 1
        - id: PQ-4
          severity: Important
          title: D4a's ReviewEvent.Form — operator-confirmed and the instrument that makes the deferred remedies safe — has no task and no Done-when row
          detail: 'ARCH-PURPOSE: the plan defers both promotion remedies on the grounds that D4a will let them be chosen from evidence, then omits D4a from the executable half; T3 spends that slot on the deleted EventUnsure. Add the field, the capture-site line, and a pinned row. Note the field must be declared BEFORE At (store/event.go:56-60): a field written after "at:" survives the cut that drops the timestamp, so a torn record would look whole.'
          family: confirmed-purpose-has-no-task
          round: 1
        - id: PQ-5
          severity: Important
          title: T5 claims the loop needs no code, but three decisions require it — the grid's redraw path, the footer rows, and Tab
          detail: show() writes q.Prompt() into the buffer once per index (play_loop.go:176-183, guarded by written != s.Index), so the Architecture header's "the grid is a Prompt() string like any other" would draw marks that never update; the footer is hardcoded []string{sittingBar(fig)} at :186, so D6's toggle and definition panel need a second change; and Tab decodes to KeyTab (key.go:23,92) with no case in toInput (play_loop.go:407-424), so it is dropped before play sees it. Name the render path explicitly and give T5 real scope.
          family: loop-changes-unacknowledged
          round: 1
        - id: PQ-6
          severity: Minor
          title: No Done-when row pins D3's Enter-commits / Ctrl-C-cancels split, nor D5's labelled keys skipping d
          detail: D3 calls the Enter sweep "the only expensive-to-undo action on this surface" and rests on Ctrl-C leaving unmarked words with no event; neither half is pinned. D5's mouse-less path (labels 0-9 then a b c e f g, d reserved by toInput at play_loop.go:418) is likewise unpinned, and it is exactly the path that would silently degrade to "everything is No".
          family: done-when-omits-decided-behaviour
          round: 1
        - id: PQ-7
          severity: Minor
          title: T9's doc sweep misses the in-tree forward references to this issue that D7 falsifies
          detail: schedule/progress.go:13-14 says this issue's board "extends [the Grade seam] with GradeUnsure", which D7 deletes; play/session.go:305 describes the mark as "firm". T9 names README.md, atlas/define.md and the --help key table only.
          family: doc-sweep-incomplete
          round: 1
      blocked: true
---

# Gate ledger — tools#40 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-01T09:58:48-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `click-never-answers-invariant` D5 makes a click record a verdict, reversing a stated loop invariant, and the region machinery cannot reach the live edge anyway
  play_loop.go:258 states "A CLICK ACTS AND NEVER ANSWERS ... it stops before toInput, so play.Apply never learns that a mouse exists"; D5 inverts that without saying so. It is also unbuildable as drawn: RegionAtRow resolves through screen.LineAt (screen.go:196, :663-673), which returns false for prompt and footer rows, so a grid on D6's live edge is unclickable, while a grid in the buffer cannot show marks landing. Board cells would additionally collide with the RegionHeadword audio target at play_loop.go:178, :268-274.
- **PQ-2** [Critical] `apply-assumes-one-word-per-question` Gating advance on Spent() is not enough — three other Apply paths and livePrompt assume one word per question
  A No mark grades Wrong on an unrevealed board, hitting session.go:204/230, which sets s.Graded = true. The next mark then trips the hoist at session.go:169 and calls advance(..., Skipped), abandoning the rest of the board. Enter hits session.go:192 and returns OutcomeNone instead of spending Rest(Wrong), and livePrompt (play_loop.go:639-643) switches to "any key = next word". T1 as scoped does not make a board usable.
- **PQ-3** [Critical] `plan-body-stale-after-revision` Core concepts, Integration points, Tasks and Done-when still specify the unsure mark, the cursor, and a Mastered-based threshold that D4, D5 and D7 delete
  T3 builds EventUnsure, CaptureReview writing it and Fold ignoring it; T4 routes "a word with a recent unsure" to 2.3; Done-when row 4 pins TestUnsureDoesNotMoveTheBoxAndForcesARealTest — all deleted by D7. T2 and the Board entry render "with the cursor" and Word() returns "the CURSOR's word", deleted by D5. boardsFor partitions "by Mastered" (box >= 9, progress.go:39/126) against D4's box >= 3. Every task's parenthetical decision reference is also off by the deleted decisions (T3 cites D4, T4 cites D6, T6 cites D7, T8 cites D8). An implementer executing the Tasks section builds the superseded design.
- **PQ-4** [Important] `confirmed-purpose-has-no-task` D4a's ReviewEvent.Form — operator-confirmed and the instrument that makes the deferred remedies safe — has no task and no Done-when row
  ARCH-PURPOSE: the plan defers both promotion remedies on the grounds that D4a will let them be chosen from evidence, then omits D4a from the executable half; T3 spends that slot on the deleted EventUnsure. Add the field, the capture-site line, and a pinned row. Note the field must be declared BEFORE At (store/event.go:56-60): a field written after "at:" survives the cut that drops the timestamp, so a torn record would look whole.
- **PQ-5** [Important] `loop-changes-unacknowledged` T5 claims the loop needs no code, but three decisions require it — the grid's redraw path, the footer rows, and Tab
  show() writes q.Prompt() into the buffer once per index (play_loop.go:176-183, guarded by written != s.Index), so the Architecture header's "the grid is a Prompt() string like any other" would draw marks that never update; the footer is hardcoded []string{sittingBar(fig)} at :186, so D6's toggle and definition panel need a second change; and Tab decodes to KeyTab (key.go:23,92) with no case in toInput (play_loop.go:407-424), so it is dropped before play sees it. Name the render path explicitly and give T5 real scope.
- **PQ-6** [Minor] `done-when-omits-decided-behaviour` No Done-when row pins D3's Enter-commits / Ctrl-C-cancels split, nor D5's labelled keys skipping d
  D3 calls the Enter sweep "the only expensive-to-undo action on this surface" and rests on Ctrl-C leaving unmarked words with no event; neither half is pinned. D5's mouse-less path (labels 0-9 then a b c e f g, d reserved by toInput at play_loop.go:418) is likewise unpinned, and it is exactly the path that would silently degrade to "everything is No".
- **PQ-7** [Minor] `doc-sweep-incomplete` T9's doc sweep misses the in-tree forward references to this issue that D7 falsifies
  schedule/progress.go:13-14 says this issue's board "extends [the Grade seam] with GradeUnsure", which D7 deletes; play/session.go:305 describes the mark as "firm". T9 names README.md, atlas/define.md and the --help key table only.

## Open findings

- **PQ-1** [Critical] `click-never-answers-invariant` D5 makes a click record a verdict, reversing a stated loop invariant, and the region machinery cannot reach the live edge anyway
- **PQ-2** [Critical] `apply-assumes-one-word-per-question` Gating advance on Spent() is not enough — three other Apply paths and livePrompt assume one word per question
- **PQ-3** [Critical] `plan-body-stale-after-revision` Core concepts, Integration points, Tasks and Done-when still specify the unsure mark, the cursor, and a Mastered-based threshold that D4, D5 and D7 delete
- **PQ-4** [Important] `confirmed-purpose-has-no-task` D4a's ReviewEvent.Form — operator-confirmed and the instrument that makes the deferred remedies safe — has no task and no Done-when row
- **PQ-5** [Important] `loop-changes-unacknowledged` T5 claims the loop needs no code, but three decisions require it — the grid's redraw path, the footer rows, and Tab
- **PQ-6** [Minor] `done-when-omits-decided-behaviour` No Done-when row pins D3's Enter-commits / Ctrl-C-cancels split, nor D5's labelled keys skipping d
- **PQ-7** [Minor] `doc-sweep-incomplete` T9's doc sweep misses the in-tree forward references to this issue that D7 falsifies

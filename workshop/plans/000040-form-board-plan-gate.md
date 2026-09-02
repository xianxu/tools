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
    - "n": 2
      timestamp: "2026-09-01T10:19:29-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: 'D10 moves the grid to the footer and adds FooterRowAt; D11 restates the invariant and keeps #38''s test untouched.'
          round: 2
        - id: PQ-2
          disposition: addressed
          note: D12 names the miss-on-hidden branch, InputDrop, livePrompt and Enter as the four Batch consultation points, and T1 scopes all four.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Core concepts, Integration points, Tasks and Done-when now carry two marks, no cursor, box >= 3, and correct decision references.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: T7 builds ReviewEvent.Form and Done-when row 9 pins it over all three forms; event.go's own comment guards the before-At ordering.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: T3, T4, T5 and T6 give the loop real scope for Tab, the footer origin, the click offer and the footer floor.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Done-when rows 3, 4 and 6 pin Enter-commits, Ctrl-C-cancels and the mouse-less path including d.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: T12 names both in-tree forward references D7 falsifies.
          round: 2
      findings:
        - id: PQ-8
          severity: Minor
          title: Three of the issue's seven Done-when rows are contradicted or unserved by the revised plan
          detail: |-
            This is the 2nd finding in family confirmed-purpose-has-no-task, so the deliverable is the
            rule, not the site: after a revision, sweep every artifact that restates a deleted decision —
            the plan body AND the issue's Spec and Done-when — and make every issue Done-when row either
            served by a task, restated via a Revisions entry, or named a non-goal with its reason.
            Measured here: row 3 says box >= 8 against D4's >= 3, row 4 pins unsure against D7, and row 5
            (/board forces the form) has no task, no pin and no D9 entry. Row 3's "a deck spanning both"
            also has no plan row testing the below-threshold side going to 2.3.
          family: confirmed-purpose-has-no-task
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-01T10:25:53-07:00"
      agent: claude
      dispose:
        - id: PQ-8
          disposition: not-addressed
          note: Issue rows 3/4/5 swept and the rule stated in Revisions; only the both-sides pin for box >= 3 remains open.
          round: 3
      findings:
        - id: PQ-9
          severity: Important
          title: Space is InputReveal too, so it spends the board — the plan decides only Enter
          detail: |-
            This is the 2nd finding in family apply-assumes-one-word-per-question (prevalence 2:
            PQ-2 at the state machine, this at the input mapping). Do not fix the instance. The rule
            is: a form that reinterprets an existing Input kind must enumerate EVERY key toInput can
            turn into that kind, not the one key the decision was written about. toInput maps
            KeyEnter (play_loop.go:411) and rune ' ' (play_loop.go:415) to the same InputReveal, and
            a board is never Graded (D12) so session.go:169 does not intercept — space therefore runs
            Rest(Wrong) over every unmarked word, the action D3 calls the only expensive-to-undo one
            on this surface. Write the enumeration into the plan as a table (toInput's five cases plus
            the gestures viewportGesture consumes at play_loop.go:254, one row each for what a board
            does) and sweep it in this round.
          family: apply-assumes-one-word-per-question
          round: 3
        - id: PQ-10
          severity: Important
          title: fitFooter's new floor breaks the budget invariant Paint documents, with no stated behaviour at the limit
          detail: |-
            T6/D10 give fitFooter a floor so grid rows are never dropped. fitFooter (screen.go:740-750)
            today guarantees footerRows <= avail, which is what makes s.rows (screen.go:419) and the
            cursor walk-back (screen.go:450) sound; Paint's own comment (screen.go:380-414) states that
            a footer taller than the terminal scrolls it and "a click at viewport row R stops meaning
            buffer line R+offset" — landing on FooterRowAt, the seam this issue adds. State the
            replacement contract and the over-budget behaviour (clip to whole cell-rows and carry the
            rest, refuse the board below a minimum height, or give the buffer zero rows and accept that
            T9's relearn line is invisible), and add a Done-when row whose mutation is "the floor lets
            footerRows exceed termRows - promptRows".
          family: contract-override-omits-limit-case
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-01T10:28:12-07:00"
      agent: claude
      dispose:
        - id: PQ-8
          disposition: addressed
          note: Threshold, unsure and /board swept in the issue; the revision states the sweeping rule itself.
          round: 4
        - id: PQ-9
          disposition: addressed
          note: D14 splits InputFinish from InputReveal; toInput's five cases each decided, two Done-when rows pin it.
          round: 4
        - id: PQ-10
          disposition: addressed
          note: D15 drops the floor entirely — fitFooter unchanged, a board that cannot be drawn whole is not offered.
          round: 4
      blocked: false
content_hash: 3d7b0ec1ec89baf89a6a070cb8cb42bc85ea64d15a619f4d80b44fe5f89c4c1d
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

## Round 2 — 2026-09-01T10:19:29-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — D10 moves the grid to the footer and adds FooterRowAt; D11 restates the invariant and keeps #38's test untouched.
- PQ-2 — addressed — D12 names the miss-on-hidden branch, InputDrop, livePrompt and Enter as the four Batch consultation points, and T1 scopes all four.
- PQ-3 — addressed — Core concepts, Integration points, Tasks and Done-when now carry two marks, no cursor, box >= 3, and correct decision references.
- PQ-4 — addressed — T7 builds ReviewEvent.Form and Done-when row 9 pins it over all three forms; event.go's own comment guards the before-At ordering.
- PQ-5 — addressed — T3, T4, T5 and T6 give the loop real scope for Tab, the footer origin, the click offer and the footer floor.
- PQ-6 — addressed — Done-when rows 3, 4 and 6 pin Enter-commits, Ctrl-C-cancels and the mouse-less path including d.
- PQ-7 — addressed — T12 names both in-tree forward references D7 falsifies.

### Raised

- **PQ-8** [Minor] `confirmed-purpose-has-no-task` Three of the issue's seven Done-when rows are contradicted or unserved by the revised plan
  This is the 2nd finding in family confirmed-purpose-has-no-task, so the deliverable is the
  rule, not the site: after a revision, sweep every artifact that restates a deleted decision —
  the plan body AND the issue's Spec and Done-when — and make every issue Done-when row either
  served by a task, restated via a Revisions entry, or named a non-goal with its reason.
  Measured here: row 3 says box >= 8 against D4's >= 3, row 4 pins unsure against D7, and row 5
  (/board forces the form) has no task, no pin and no D9 entry. Row 3's "a deck spanning both"
  also has no plan row testing the below-threshold side going to 2.3.

## Round 3 — 2026-09-01T10:25:53-07:00 (claude) — BLOCKED

### Disposed

- PQ-8 — not-addressed — Issue rows 3/4/5 swept and the rule stated in Revisions; only the both-sides pin for box >= 3 remains open.

### Raised

- **PQ-9** [Important] `apply-assumes-one-word-per-question` Space is InputReveal too, so it spends the board — the plan decides only Enter
  This is the 2nd finding in family apply-assumes-one-word-per-question (prevalence 2:
  PQ-2 at the state machine, this at the input mapping). Do not fix the instance. The rule
  is: a form that reinterprets an existing Input kind must enumerate EVERY key toInput can
  turn into that kind, not the one key the decision was written about. toInput maps
  KeyEnter (play_loop.go:411) and rune ' ' (play_loop.go:415) to the same InputReveal, and
  a board is never Graded (D12) so session.go:169 does not intercept — space therefore runs
  Rest(Wrong) over every unmarked word, the action D3 calls the only expensive-to-undo one
  on this surface. Write the enumeration into the plan as a table (toInput's five cases plus
  the gestures viewportGesture consumes at play_loop.go:254, one row each for what a board
  does) and sweep it in this round.
- **PQ-10** [Important] `contract-override-omits-limit-case` fitFooter's new floor breaks the budget invariant Paint documents, with no stated behaviour at the limit
  T6/D10 give fitFooter a floor so grid rows are never dropped. fitFooter (screen.go:740-750)
  today guarantees footerRows <= avail, which is what makes s.rows (screen.go:419) and the
  cursor walk-back (screen.go:450) sound; Paint's own comment (screen.go:380-414) states that
  a footer taller than the terminal scrolls it and "a click at viewport row R stops meaning
  buffer line R+offset" — landing on FooterRowAt, the seam this issue adds. State the
  replacement contract and the over-budget behaviour (clip to whole cell-rows and carry the
  rest, refuse the board below a minimum height, or give the buffer zero rows and accept that
  T9's relearn line is invisible), and add a Done-when row whose mutation is "the floor lets
  footerRows exceed termRows - promptRows".

## Round 4 — 2026-09-01T10:28:12-07:00 (claude) — passed

### Disposed

- PQ-8 — addressed — Threshold, unsure and /board swept in the issue; the revision states the sweeping rule itself.
- PQ-9 — addressed — D14 splits InputFinish from InputReveal; toInput's five cases each decided, two Done-when rows pin it.
- PQ-10 — addressed — D15 drops the floor entirely — fitFooter unchanged, a board that cannot be drawn whole is not offered.

## Open findings

(none — every finding has been disposed)

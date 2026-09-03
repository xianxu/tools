---
gate: boundary-review
issue: 42
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-03T10:54:32-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: 'The board''s drop colour is unpinned: both the Palette.Drop entry and paint''s Dropped arm can be deleted with the full suite green'
          detail: |-
            Verified by reverting at HEAD in a scratch worktree. Deleting `Drop: "\x1b[2;9m"` from boardPalette
            (play_loop.go:636) leaves `go test ./cmd/define/...` green; separately deleting `case Dropped: return
            b.pal.Drop` from paint (play/board.go:427) also leaves both packages green. So a dropped cell could paint
            identically to an unmarked one with nothing noticing. board_test.go:365 asserts Yes/No only and the pty row
            (pty_conformance_test.go:1077) asserts Yes only and never enters drop mode, while README.md:141 promises
            "struck-out for a drop". Plan Task 3 Step 8 names the palette entry as one of three required mutation checks
            and is ticked; the issue Log lists four verified pieces and the palette is not among them. Fix: assert
            pal.Drop around a dropped cell in board_test.go, and add a drop row to the pty suite so the gesture also
            gets a live-terminal check.
          family: pin-that-cannot-fail
          round: 1
        - id: BR-2
          severity: Important
          title: todaysQuestions reports "none could be looked up" for words whose lookup succeeded but which were unaskable
          detail: |-
            Reproduced: a one-word deck (`bases`) at opt.width=12 prints the correct per-word skip line naming both
            causes, then `define: 1 words are due but none could be looked up` (play_loop.go:1054) and returns 1. The
            dictionary answered fine; the window was too narrow for a board and the deck could supply no distractors.
            This is PQ-1's false-summary concern narrowed rather than removed, and the surviving population is exactly
            this issue's subject — a young deck on a narrow terminal, which ran a full sitting before #42. Fix: track
            skips that were not lookup failures and phrase the summary accordingly ("none could be asked in this
            window"), pinned by the reproduction above.
          family: message-names-a-cause-the-code-did-not-establish
          round: 1
        - id: BR-3
          severity: Important
          title: Four production doc comments still route words to form 2.1, and the atlas still lists `recall` as a live form stamp
          detail: |-
            optionpool.go:112, :141, :225 and play/pick.go:155 all state as CURRENT behaviour that a word "falls back to
            form 2.1"; atlas/define.md:2235 still enumerates ReviewEvent.Form as "recall, meaning, board". All are
            currentTruthFiles. PQ-7 named four different files (choice.go:16, board.go:121, play_loop.go:226,
            optionpool.go:14) and exactly those four were fixed — the instance, not the class (ARCH-PURPOSE). Plan Task 4
            Step 3 ticks a tree-wide grep and cites lessons.md's "a retraction is not done until git grep over the TREE
            is clean". Fix: sweep the five sites, then add a retiredPhrases row (repo_guard_test.go:1095) keyed on the
            present-tense phrase "fall(s) back to form 2.1" rather than the bare form name, since the plan deliberately
            keeps ~8 historical mentions a bare row would redden.
          family: retraction-not-swept-over-the-tree
          round: 1
        - id: BR-4
          severity: Minor
          title: The Lookup / skip-message / ParseEntry block is copy-pasted three times inside todaysQuestions
          detail: |-
            play_loop.go:977, :1005 and :1033 carry the same four lines; the last two are byte-identical. An
            `entryFor(key) (Entry, bool)` closure beside `ask`, memoising into the existing `parsed` map, collapses all
            three and removes the chance that one of them drifts on the error path (ARCH-DRY).
          family: duplicated-block-should-be-a-helper
          round: 1
        - id: BR-5
          severity: Minor
          title: Board.Dropped() reads b.cells[b.last], so its correctness rests on b.last not having moved since the arming Mark
          detail: |-
            play/board.go:592-598. `Rest` also writes `b.last` and does not clear `dropping`, so the two fields are only
            consistent because every Mark path calls advance immediately. Capturing the word at arm time
            (`b.dropWord = b.cells[i].Word` in Mark) makes the invariant local instead of a property of the caller.
          family: derived-state-depends-on-call-order
          round: 1
        - id: BR-6
          severity: Minor
          title: The plan's ARCH-CONSTRAINTS block claims a Render newly paid on every mature board word; the code pays none
          detail: |-
            The plan's todaysQuestions bullet says the unified loop newly pays "a Render plus a click-region map entry on
            every MATURE board word". The implemented loop appends mature words to `triage` before any Lookup
            (play_loop.go:971-974), so they are never rendered and lookups stay at one per due word. The cost table
            should describe the code; a `## Revisions` entry is the right repair.
          family: plan-claim-outruns-the-code
          round: 1
      blocked: true
---

# Gate ledger — tools#42 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-03T10:54:32-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `pin-that-cannot-fail` The board's drop colour is unpinned: both the Palette.Drop entry and paint's Dropped arm can be deleted with the full suite green
  Verified by reverting at HEAD in a scratch worktree. Deleting `Drop: "\x1b[2;9m"` from boardPalette
  (play_loop.go:636) leaves `go test ./cmd/define/...` green; separately deleting `case Dropped: return
  b.pal.Drop` from paint (play/board.go:427) also leaves both packages green. So a dropped cell could paint
  identically to an unmarked one with nothing noticing. board_test.go:365 asserts Yes/No only and the pty row
  (pty_conformance_test.go:1077) asserts Yes only and never enters drop mode, while README.md:141 promises
  "struck-out for a drop". Plan Task 3 Step 8 names the palette entry as one of three required mutation checks
  and is ticked; the issue Log lists four verified pieces and the palette is not among them. Fix: assert
  pal.Drop around a dropped cell in board_test.go, and add a drop row to the pty suite so the gesture also
  gets a live-terminal check.
- **BR-2** [Important] `message-names-a-cause-the-code-did-not-establish` todaysQuestions reports "none could be looked up" for words whose lookup succeeded but which were unaskable
  Reproduced: a one-word deck (`bases`) at opt.width=12 prints the correct per-word skip line naming both
  causes, then `define: 1 words are due but none could be looked up` (play_loop.go:1054) and returns 1. The
  dictionary answered fine; the window was too narrow for a board and the deck could supply no distractors.
  This is PQ-1's false-summary concern narrowed rather than removed, and the surviving population is exactly
  this issue's subject — a young deck on a narrow terminal, which ran a full sitting before #42. Fix: track
  skips that were not lookup failures and phrase the summary accordingly ("none could be asked in this
  window"), pinned by the reproduction above.
- **BR-3** [Important] `retraction-not-swept-over-the-tree` Four production doc comments still route words to form 2.1, and the atlas still lists `recall` as a live form stamp
  optionpool.go:112, :141, :225 and play/pick.go:155 all state as CURRENT behaviour that a word "falls back to
  form 2.1"; atlas/define.md:2235 still enumerates ReviewEvent.Form as "recall, meaning, board". All are
  currentTruthFiles. PQ-7 named four different files (choice.go:16, board.go:121, play_loop.go:226,
  optionpool.go:14) and exactly those four were fixed — the instance, not the class (ARCH-PURPOSE). Plan Task 4
  Step 3 ticks a tree-wide grep and cites lessons.md's "a retraction is not done until git grep over the TREE
  is clean". Fix: sweep the five sites, then add a retiredPhrases row (repo_guard_test.go:1095) keyed on the
  present-tense phrase "fall(s) back to form 2.1" rather than the bare form name, since the plan deliberately
  keeps ~8 historical mentions a bare row would redden.
- **BR-4** [Minor] `duplicated-block-should-be-a-helper` The Lookup / skip-message / ParseEntry block is copy-pasted three times inside todaysQuestions
  play_loop.go:977, :1005 and :1033 carry the same four lines; the last two are byte-identical. An
  `entryFor(key) (Entry, bool)` closure beside `ask`, memoising into the existing `parsed` map, collapses all
  three and removes the chance that one of them drifts on the error path (ARCH-DRY).
- **BR-5** [Minor] `derived-state-depends-on-call-order` Board.Dropped() reads b.cells[b.last], so its correctness rests on b.last not having moved since the arming Mark
  play/board.go:592-598. `Rest` also writes `b.last` and does not clear `dropping`, so the two fields are only
  consistent because every Mark path calls advance immediately. Capturing the word at arm time
  (`b.dropWord = b.cells[i].Word` in Mark) makes the invariant local instead of a property of the caller.
- **BR-6** [Minor] `plan-claim-outruns-the-code` The plan's ARCH-CONSTRAINTS block claims a Render newly paid on every mature board word; the code pays none
  The plan's todaysQuestions bullet says the unified loop newly pays "a Render plus a click-region map entry on
  every MATURE board word". The implemented loop appends mature words to `triage` before any Lookup
  (play_loop.go:971-974), so they are never rendered and lookups stay at one per due word. The cost table
  should describe the code; a `## Revisions` entry is the right repair.

## Open findings

- **BR-1** [Important] `pin-that-cannot-fail` The board's drop colour is unpinned: both the Palette.Drop entry and paint's Dropped arm can be deleted with the full suite green
- **BR-2** [Important] `message-names-a-cause-the-code-did-not-establish` todaysQuestions reports "none could be looked up" for words whose lookup succeeded but which were unaskable
- **BR-3** [Important] `retraction-not-swept-over-the-tree` Four production doc comments still route words to form 2.1, and the atlas still lists `recall` as a live form stamp
- **BR-4** [Minor] `duplicated-block-should-be-a-helper` The Lookup / skip-message / ParseEntry block is copy-pasted three times inside todaysQuestions
- **BR-5** [Minor] `derived-state-depends-on-call-order` Board.Dropped() reads b.cells[b.last], so its correctness rests on b.last not having moved since the arming Mark
- **BR-6** [Minor] `plan-claim-outruns-the-code` The plan's ARCH-CONSTRAINTS block claims a Render newly paid on every mature board word; the code pays none

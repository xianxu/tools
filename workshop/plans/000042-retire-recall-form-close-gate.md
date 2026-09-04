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
    - "n": 2
      timestamp: "2026-09-03T11:33:49-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Fixed at the class via play.Marks() + Palette.For; both mutations now redden named tests — I re-verified each by reverting at HEAD.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Branch added and pinned in both directions by TestAnEmptySittingNamesWhyItIsEmpty; reverting the branch reddens it. See the new Minor for the surviving mixed case.
          round: 2
        - id: BR-3
          disposition: addressed
          note: All five named sites swept and retiredPhrases rows added; I confirmed the guard fires on a reintroduced claim. The class is re-raised as a new finding under the same family for boardsFor.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: play_loop.go:981, :1009, :1038 still carry the same Lookup/skip/ParseEntry block. Minor, non-blocking.
          round: 2
        - id: BR-5
          disposition: not-addressed
          note: Board.Dropped() still reads b.cells[b.last]; no dropWord field. Minor, non-blocking — no path currently interleaves Rest with an armed drop.
          round: 2
        - id: BR-6
          disposition: addressed
          note: A 2026-09-03 "## Revisions" entry corrects the cost claim and the bullet is marked "(corrected at close)", which is the repo's append-don't-overwrite convention.
          round: 2
      findings:
        - id: BR-7
          severity: Important
          title: boardsFor was deleted by this window and is still the current account of selection in six current-truth sites, including the atlas's Form 2.5 section
          detail: |-
            2ND FINDING IN THIS FAMILY — do not fix the six sites; fix the rule. Round 1 swept the instance it was
            given (form 2.1 routing claims) and added retiredPhrases rows for that phrase, but the enumeration the
            class implies — every top-level declaration this window removed — was never written, and `boardsFor` was
            deleted in the same window as `Recall`. Sites: atlas/define.md:2100 ("Selection was a capability question
            until this: form 2.3 … 2.1 when it cannot. `boardsFor` partitions the day's keys at box >= 3"),
            atlas/define.md:2138, play_loop.go:297, play_loop.go:654, play/board.go:216, play/board.go:354 — all
            present tense, all currentTruthFiles, and the atlas now holds two contradictory accounts of the selection
            rule this issue exists to change. THE RULE: a window that removes a top-level declaration owes a tree-wide
            sweep of that name regardless of export status, enforced by the guard. TestARemovedDeclarationIsSweptOrRetired
            (repo_guard_test.go:1497) already implements the sweep but gates on isCitableName (:1628), which requires an
            exported or Test* name. Proven: adding `if name == "boardsFor" { return true }` to isCitableName turns the
            guard red on atlas/define.md, play/board.go, play_loop.go and the plan. Widen the filter to removed camelCase
            identifiers with an interior capital (which keeps the `ids`/"for-bids" case out), and note the fix must
            handle workshop/plans/…-plan.md, which legitimately names boardsFor in its "deleted" row and is itself a
            currentTruthFile.
          family: retraction-not-swept-over-the-tree
          round: 2
        - id: BR-8
          severity: Important
          title: Five current-truth comments restate an extent this window changed, including Mark's own doc three lines above the const that falsifies it
          detail: |-
            play/board.go:28 says "TWO marks and an ABSENCE, which is not a third mark" while the const block directly
            below now declares Dropped, added by this diff; :33 says "a cell has three states" (four); :472 says "Both
            spellings are the SAME WIDTH" while Keys() beneath it returns three and its test is named
            TestEveryModeSpellingIsTheSameWidthAndFitsEighty; play/question.go:30 says "NO SHIPPED FORM PRODUCES Skipped
            today" and cites form 2.1, but Mark.Verdict() returns Skipped for Dropped so Board.Mark now returns
            (Skipped, true); atlas/define.md:2183 says the palette is "green for yes, red for no" while README.md:141
            promises a third, struck-out sequence. New family, not the retraction one: nothing was retracted here, a
            count was restated. THE RULE: a comment must not restate the cardinality or extent of a set the code
            enumerates. This window already built the owner — play.Marks() is the extent — so these comments should
            defer to it or drop the count, exactly as Keys()'s own comment declines to enumerate the label set. The #42
            Spec took "a comment describing the opposite of the code, on the key this issue changes" into scope;
            board.go:28 is that comment on the type this issue extended.
          family: comment-restates-a-count-the-code-owns
          round: 2
        - id: BR-9
          severity: Important
          title: play.Marks() and Palette.For are new exported surface with no atlas entry and no Core-concepts row
          detail: |-
            play/board.go:760-786. Both were added in the close round as the single-source mechanism for the mark set.
            Marks()' own doc positions it beside BoardLabels and numRegionKinds — and the atlas documents numRegionKinds
            at :461 and :2290 precisely because it is that kind of mechanism — but atlas/define.md gained no entry for
            either, and the palette paragraph it belongs in (:2183) is the same one finding (b) leaves stale. The plan's
            Core-concepts table (plan.md:20-29) also has no rows for them; TestPlanTablesNameEntitiesThatExist only
            checks table -> tree, so the reverse direction is unguarded and the table now under-describes the diff — the
            mirror of the correction its own Revision 2 made.
          family: new-surface-undocumented
          round: 2
        - id: BR-10
          severity: Minor
          title: The empty-sitting summary still misattributes when the deck mixes lookup failures with unaskable words
          detail: |-
            2ND FINDING IN THIS FAMILY — do not fix the branch condition. Reproduced at HEAD with
            playRig(t, "bases", "rizz") and opt.width=12: `rizz` fails lookup, `bases` is unaskable, so
            `unaskable(1) != len(keys)(2)` at play_loop.go:1066 and the summary prints "2 words are due but none could
            be looked up" over a word the dictionary answered fine. THE RULE: an aggregate summary must be derived from
            a tally of the per-word outcomes, never from one counter compared against the total. Every skip path already
            prints its own cause, so either count each reason and phrase from the tally, or have the summary stop naming
            a cause when the tally is mixed ("none could be asked; see the reasons above").
          family: message-names-a-cause-the-code-did-not-establish
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-03T22:51:04-07:00"
      agent: claude
      blocked: true
      protocol_error: no valid findings block
    - "n": 4
      timestamp: "2026-09-03T23:11:35-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: not-addressed
          note: Still three copies at play_loop.go:981, :1009, :1038; Minor and explicitly declined with a stated reason.
          round: 4
        - id: BR-5
          disposition: not-addressed
          note: play/board.go:594-599 still reads b.cells[b.last]; no dropWord field. No path interleaves Rest with an armed drop today.
          round: 4
        - id: BR-7
          disposition: addressed
          note: Six sites swept (tree grep clean) and isCitableName widened — but the guard half is unpinned; raised as new under pin-that-cannot-fail.
          round: 4
        - id: BR-8
          disposition: addressed
          note: All five named sites corrected; the residual restatements the fix itself introduced are raised as the 2nd in that family.
          round: 4
        - id: BR-9
          disposition: addressed
          note: atlas/define.md:2468 documents both, plan Core-concepts gains Marks and Palette.For rows, and a Revisions entry explains the late addition.
          round: 4
        - id: BR-10
          disposition: not-addressed
          note: Reproduced at HEAD with playRig("bases","rizz") at width 12 — stderr still prints "2 words are due but none could be looked up".
          round: 4
      findings:
        - id: BR-11
          severity: Important
          title: BR-7's rule fix — the isCitableName widening — can be reverted with the entire suite green
          detail: |-
            2ND FINDING IN THIS FAMILY — do not add one assertion; fix the rule. Verified in a pinned scratch
            worktree at HEAD: replacing the interior-capital clause at repo_guard_test.go:1677 with `return false`
            leaves `go test ./...` green in every package, and separately deleting the `"boardsFor"` row at :1059
            is also green because currentTruthOnly's new exemption already strips the plan's mentions. So both
            halves of the round-2 fix are individually revertible with nothing noticing. The rule is already
            written at repo_guard_test.go:1477 ("A finding-fix with no test that reddens without it is not
            addressed") and in lessons.md, and the mechanism is eight lines below it:
            TestPlanStatusNormalisesToTheVocabulary is a fixture table for planStatus precisely because no plan in
            the tree exercises its branches. Give isCitableName the same fixture table (boardsFor/choiceFor true,
            ids/binds/paint false, Test*/Fuzz*/exported true, "" false).
          family: pin-that-cannot-fail
          round: 4
        - id: BR-12
          severity: Important
          title: Seven production doc comments still predicate on form 2.1 in the present tense, three lines from the block round 2 rewrote
          detail: |-
            3RD FINDING IN THIS FAMILY — do not fix the seven sites; fix the rule. `git grep -n 'form 2\.1|2\.1 or'`
            over non-test Go returns nine hits, seven present tense: play/choice.go:8 ("where form 2.1 is a recall
            test"), :217 ("a concept form 2.1 has no answer for"), play/question.go:53 ("form 2.1's y/n and form
            2.3's 1/2/3/4 are the same shape") on the central Question interface, play/session.go:139 ("would make
            form 2.1 answer a question it cannot") and :380 ("which is what form 2.1 is"), store/event.go:34 ("form
            2.3 sets it; form 2.1 cannot") on a persisted field, and play/board.go:469 ("Tab reaches nothing at all
            on 2.1 or 2.3") — three lines above the comment BR-8's fix rewrote. Atlas and README are clean, so the
            residue is entirely Go doc comments. THE RULE: the sweep's input set is the retired CONCEPT, not only
            the removed SYMBOL, and the discriminator round 1 needed is TENSE rather than the bare name. Round 1
            declined a bare ban because ~8 historical mentions would redden; retiredPhrases already matches phrases
            case-insensitively over currentTruthOnly across production Go, so tense-keyed rows ("form 2.1 is",
            "form 2.1 has", "form 2.1 cannot", "form 2.1's", "2.1 or 2.3") close the class while leaving
            "was"/"used to"/"before #42" alone by construction.
          family: retraction-not-swept-over-the-tree
          round: 4
        - id: BR-13
          severity: Important
          title: The count class was closed by re-wording, and the new wording restates the count — including the comment that declares the rule, and a false claim about the test that pins it
          detail: |-
            2ND FINDING IN THIS FAMILY — do not re-word again. play/board.go:28 reads "THREE marks and an ABSENCE,
            and `Marks()` below is the EXTENT — a count spelled in prose is a second owner of it", spelling the
            count in the sentence that forbids it; :512 repeats "THREE now"; play_loop.go:614 still gives the
            palette as "GREEN for yes, RED for no" for a three-sequence palette. The sharp one is board.go:480-481,
            which claims TestEveryModeSpellingIsTheSameWidthAndFitsEighty "derives its loop from the cycle rather
            than counting the spellings here" while play/board_test.go:950 is `for range 3`. Proven by mutation: I
            added an Unsure mark to the const block, Marks() and Toggle — TestEveryBoardMarkHasAPaintedSequence went
            red ("mark 4 has no sequence") and the spelling test stayed green, with Keys()' switch silently
            returning the [yes] default for the unwritten mode. So this window built the extent owner and wired it
            to the palette only; the prompt row, which R11 says is the last thing a short window gives up, has none.
            Derive the spelling loop from play.Marks() and give Keys() a loud default, then let the comments defer
            to that instead of asserting a property the test does not have.
          family: comment-restates-a-count-the-code-owns
          round: 4
        - id: BR-14
          severity: Minor
          title: The `| deleted |` exemption strips the symbol from the whole document, not just the row, and this plan already relies on it
          detail: |-
            repo_guard_test.go:658-662 ReplaceAllString's the row's symbol out of the entire artifact, so a document
            carrying one deleted-row gets a blind spot for that name everywhere in it. The plan already uses it:
            Step 8a still says "`boardsFor` is called from five places" and the PQ-6 block "`boardsFor` records
            singles-first-then-boards", both present tense and both now invisible to
            TestNoArtifactNamesARetiredSymbol. The comment's "self-limiting in two directions" claim is true across
            documents and symbols but not within a document. Also: the match is unanchored (no `\b`), and the
            HasSuffix("| deleted |") test only fires on three-column tables, so a deleted row in the
            four-column Integration-points table gets no exemption at all.
          family: guard-exemption-wider-than-its-warrant
          round: 4
      blocked: false
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

## Round 2 — 2026-09-03T11:33:49-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Fixed at the class via play.Marks() + Palette.For; both mutations now redden named tests — I re-verified each by reverting at HEAD.
- BR-2 — addressed — Branch added and pinned in both directions by TestAnEmptySittingNamesWhyItIsEmpty; reverting the branch reddens it. See the new Minor for the surviving mixed case.
- BR-3 — addressed — All five named sites swept and retiredPhrases rows added; I confirmed the guard fires on a reintroduced claim. The class is re-raised as a new finding under the same family for boardsFor.
- BR-4 — not-addressed — play_loop.go:981, :1009, :1038 still carry the same Lookup/skip/ParseEntry block. Minor, non-blocking.
- BR-5 — not-addressed — Board.Dropped() still reads b.cells[b.last]; no dropWord field. Minor, non-blocking — no path currently interleaves Rest with an armed drop.
- BR-6 — addressed — A 2026-09-03 "## Revisions" entry corrects the cost claim and the bullet is marked "(corrected at close)", which is the repo's append-don't-overwrite convention.

### Raised

- **BR-7** [Important] `retraction-not-swept-over-the-tree` boardsFor was deleted by this window and is still the current account of selection in six current-truth sites, including the atlas's Form 2.5 section
  2ND FINDING IN THIS FAMILY — do not fix the six sites; fix the rule. Round 1 swept the instance it was
  given (form 2.1 routing claims) and added retiredPhrases rows for that phrase, but the enumeration the
  class implies — every top-level declaration this window removed — was never written, and `boardsFor` was
  deleted in the same window as `Recall`. Sites: atlas/define.md:2100 ("Selection was a capability question
  until this: form 2.3 … 2.1 when it cannot. `boardsFor` partitions the day's keys at box >= 3"),
  atlas/define.md:2138, play_loop.go:297, play_loop.go:654, play/board.go:216, play/board.go:354 — all
  present tense, all currentTruthFiles, and the atlas now holds two contradictory accounts of the selection
  rule this issue exists to change. THE RULE: a window that removes a top-level declaration owes a tree-wide
  sweep of that name regardless of export status, enforced by the guard. TestARemovedDeclarationIsSweptOrRetired
  (repo_guard_test.go:1497) already implements the sweep but gates on isCitableName (:1628), which requires an
  exported or Test* name. Proven: adding `if name == "boardsFor" { return true }` to isCitableName turns the
  guard red on atlas/define.md, play/board.go, play_loop.go and the plan. Widen the filter to removed camelCase
  identifiers with an interior capital (which keeps the `ids`/"for-bids" case out), and note the fix must
  handle workshop/plans/…-plan.md, which legitimately names boardsFor in its "deleted" row and is itself a
  currentTruthFile.
- **BR-8** [Important] `comment-restates-a-count-the-code-owns` Five current-truth comments restate an extent this window changed, including Mark's own doc three lines above the const that falsifies it
  play/board.go:28 says "TWO marks and an ABSENCE, which is not a third mark" while the const block directly
  below now declares Dropped, added by this diff; :33 says "a cell has three states" (four); :472 says "Both
  spellings are the SAME WIDTH" while Keys() beneath it returns three and its test is named
  TestEveryModeSpellingIsTheSameWidthAndFitsEighty; play/question.go:30 says "NO SHIPPED FORM PRODUCES Skipped
  today" and cites form 2.1, but Mark.Verdict() returns Skipped for Dropped so Board.Mark now returns
  (Skipped, true); atlas/define.md:2183 says the palette is "green for yes, red for no" while README.md:141
  promises a third, struck-out sequence. New family, not the retraction one: nothing was retracted here, a
  count was restated. THE RULE: a comment must not restate the cardinality or extent of a set the code
  enumerates. This window already built the owner — play.Marks() is the extent — so these comments should
  defer to it or drop the count, exactly as Keys()'s own comment declines to enumerate the label set. The #42
  Spec took "a comment describing the opposite of the code, on the key this issue changes" into scope;
  board.go:28 is that comment on the type this issue extended.
- **BR-9** [Important] `new-surface-undocumented` play.Marks() and Palette.For are new exported surface with no atlas entry and no Core-concepts row
  play/board.go:760-786. Both were added in the close round as the single-source mechanism for the mark set.
  Marks()' own doc positions it beside BoardLabels and numRegionKinds — and the atlas documents numRegionKinds
  at :461 and :2290 precisely because it is that kind of mechanism — but atlas/define.md gained no entry for
  either, and the palette paragraph it belongs in (:2183) is the same one finding (b) leaves stale. The plan's
  Core-concepts table (plan.md:20-29) also has no rows for them; TestPlanTablesNameEntitiesThatExist only
  checks table -> tree, so the reverse direction is unguarded and the table now under-describes the diff — the
  mirror of the correction its own Revision 2 made.
- **BR-10** [Minor] `message-names-a-cause-the-code-did-not-establish` The empty-sitting summary still misattributes when the deck mixes lookup failures with unaskable words
  2ND FINDING IN THIS FAMILY — do not fix the branch condition. Reproduced at HEAD with
  playRig(t, "bases", "rizz") and opt.width=12: `rizz` fails lookup, `bases` is unaskable, so
  `unaskable(1) != len(keys)(2)` at play_loop.go:1066 and the summary prints "2 words are due but none could
  be looked up" over a word the dictionary answered fine. THE RULE: an aggregate summary must be derived from
  a tally of the per-word outcomes, never from one counter compared against the total. Every skip path already
  prints its own cause, so either count each reason and phrase from the tally, or have the summary stop naming
  a cause when the tally is mixed ("none could be asked; see the reasons above").

## Round 3 — 2026-09-03T22:51:04-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 4 — 2026-09-03T23:11:35-07:00 (claude) — passed

### Disposed

- BR-4 — not-addressed — Still three copies at play_loop.go:981, :1009, :1038; Minor and explicitly declined with a stated reason.
- BR-5 — not-addressed — play/board.go:594-599 still reads b.cells[b.last]; no dropWord field. No path interleaves Rest with an armed drop today.
- BR-7 — addressed — Six sites swept (tree grep clean) and isCitableName widened — but the guard half is unpinned; raised as new under pin-that-cannot-fail.
- BR-8 — addressed — All five named sites corrected; the residual restatements the fix itself introduced are raised as the 2nd in that family.
- BR-9 — addressed — atlas/define.md:2468 documents both, plan Core-concepts gains Marks and Palette.For rows, and a Revisions entry explains the late addition.
- BR-10 — not-addressed — Reproduced at HEAD with playRig("bases","rizz") at width 12 — stderr still prints "2 words are due but none could be looked up".

### Raised

- **BR-11** [Important] `pin-that-cannot-fail` BR-7's rule fix — the isCitableName widening — can be reverted with the entire suite green
  2ND FINDING IN THIS FAMILY — do not add one assertion; fix the rule. Verified in a pinned scratch
  worktree at HEAD: replacing the interior-capital clause at repo_guard_test.go:1677 with `return false`
  leaves `go test ./...` green in every package, and separately deleting the `"boardsFor"` row at :1059
  is also green because currentTruthOnly's new exemption already strips the plan's mentions. So both
  halves of the round-2 fix are individually revertible with nothing noticing. The rule is already
  written at repo_guard_test.go:1477 ("A finding-fix with no test that reddens without it is not
  addressed") and in lessons.md, and the mechanism is eight lines below it:
  TestPlanStatusNormalisesToTheVocabulary is a fixture table for planStatus precisely because no plan in
  the tree exercises its branches. Give isCitableName the same fixture table (boardsFor/choiceFor true,
  ids/binds/paint false, Test*/Fuzz*/exported true, "" false).
- **BR-12** [Important] `retraction-not-swept-over-the-tree` Seven production doc comments still predicate on form 2.1 in the present tense, three lines from the block round 2 rewrote
  3RD FINDING IN THIS FAMILY — do not fix the seven sites; fix the rule. `git grep -n 'form 2\.1|2\.1 or'`
  over non-test Go returns nine hits, seven present tense: play/choice.go:8 ("where form 2.1 is a recall
  test"), :217 ("a concept form 2.1 has no answer for"), play/question.go:53 ("form 2.1's y/n and form
  2.3's 1/2/3/4 are the same shape") on the central Question interface, play/session.go:139 ("would make
  form 2.1 answer a question it cannot") and :380 ("which is what form 2.1 is"), store/event.go:34 ("form
  2.3 sets it; form 2.1 cannot") on a persisted field, and play/board.go:469 ("Tab reaches nothing at all
  on 2.1 or 2.3") — three lines above the comment BR-8's fix rewrote. Atlas and README are clean, so the
  residue is entirely Go doc comments. THE RULE: the sweep's input set is the retired CONCEPT, not only
  the removed SYMBOL, and the discriminator round 1 needed is TENSE rather than the bare name. Round 1
  declined a bare ban because ~8 historical mentions would redden; retiredPhrases already matches phrases
  case-insensitively over currentTruthOnly across production Go, so tense-keyed rows ("form 2.1 is",
  "form 2.1 has", "form 2.1 cannot", "form 2.1's", "2.1 or 2.3") close the class while leaving
  "was"/"used to"/"before #42" alone by construction.
- **BR-13** [Important] `comment-restates-a-count-the-code-owns` The count class was closed by re-wording, and the new wording restates the count — including the comment that declares the rule, and a false claim about the test that pins it
  2ND FINDING IN THIS FAMILY — do not re-word again. play/board.go:28 reads "THREE marks and an ABSENCE,
  and `Marks()` below is the EXTENT — a count spelled in prose is a second owner of it", spelling the
  count in the sentence that forbids it; :512 repeats "THREE now"; play_loop.go:614 still gives the
  palette as "GREEN for yes, RED for no" for a three-sequence palette. The sharp one is board.go:480-481,
  which claims TestEveryModeSpellingIsTheSameWidthAndFitsEighty "derives its loop from the cycle rather
  than counting the spellings here" while play/board_test.go:950 is `for range 3`. Proven by mutation: I
  added an Unsure mark to the const block, Marks() and Toggle — TestEveryBoardMarkHasAPaintedSequence went
  red ("mark 4 has no sequence") and the spelling test stayed green, with Keys()' switch silently
  returning the [yes] default for the unwritten mode. So this window built the extent owner and wired it
  to the palette only; the prompt row, which R11 says is the last thing a short window gives up, has none.
  Derive the spelling loop from play.Marks() and give Keys() a loud default, then let the comments defer
  to that instead of asserting a property the test does not have.
- **BR-14** [Minor] `guard-exemption-wider-than-its-warrant` The `| deleted |` exemption strips the symbol from the whole document, not just the row, and this plan already relies on it
  repo_guard_test.go:658-662 ReplaceAllString's the row's symbol out of the entire artifact, so a document
  carrying one deleted-row gets a blind spot for that name everywhere in it. The plan already uses it:
  Step 8a still says "`boardsFor` is called from five places" and the PQ-6 block "`boardsFor` records
  singles-first-then-boards", both present tense and both now invisible to
  TestNoArtifactNamesARetiredSymbol. The comment's "self-limiting in two directions" claim is true across
  documents and symbols but not within a document. Also: the match is unanchored (no `\b`), and the
  HasSuffix("| deleted |") test only fires on three-column tables, so a deleted row in the
  four-column Integration-points table gets no exemption at all.

## Open findings

- **BR-4** [Minor] `duplicated-block-should-be-a-helper` The Lookup / skip-message / ParseEntry block is copy-pasted three times inside todaysQuestions
- **BR-5** [Minor] `derived-state-depends-on-call-order` Board.Dropped() reads b.cells[b.last], so its correctness rests on b.last not having moved since the arming Mark
- **BR-10** [Minor] `message-names-a-cause-the-code-did-not-establish` The empty-sitting summary still misattributes when the deck mixes lookup failures with unaskable words
- **BR-11** [Important] `pin-that-cannot-fail` BR-7's rule fix — the isCitableName widening — can be reverted with the entire suite green
- **BR-12** [Important] `retraction-not-swept-over-the-tree` Seven production doc comments still predicate on form 2.1 in the present tense, three lines from the block round 2 rewrote
- **BR-13** [Important] `comment-restates-a-count-the-code-owns` The count class was closed by re-wording, and the new wording restates the count — including the comment that declares the rule, and a false claim about the test that pins it
- **BR-14** [Minor] `guard-exemption-wider-than-its-warrant` The `| deleted |` exemption strips the symbol from the whole document, not just the row, and this plan already relies on it

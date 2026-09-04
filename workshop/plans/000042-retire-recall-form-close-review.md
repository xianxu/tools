# Boundary Review — tools#42 (whole-issue close)

| field | value |
|-------|-------|
| issue | 42 — retire form 2.1: the board is the fallback when a real test cannot be built |
| repo | tools |
| issue file | workshop/issues/000042-retire-recall-form.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6c5f1eb9d826a422af53a6940a1e53728ddcf019..16880c26bb7992264846558254ce2d21fcd9fde5 |
| command | sdlc close --issue 42 |
| reviewer | claude |
| timestamp | 2026-09-03T10:54:32-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The deletion is done properly: `play.Recall` and `boardsFor` are gone, selection genuinely moved after the lookup without paying a second dictionary call (mature words are triaged before any `Lookup`, young words carry their parsed entry forward via `parsed`), and the drop landed as a capability rather than a type switch. I built, vetted, gofmt'd and ran the full suite green in a pinned worktree at HEAD, plus the pty conformance subset, and I mutation-checked four of the pieces the issue's Log claims — `advance`'s `droppedBy` call, the one-shot `dropping` reset, and `Dropped.Verdict()` all redden a named test. Three things block a clean SHIP, all cheap: the drop's *colour* is entirely unpinned (I deleted both `boardPalette`'s `Drop` entry and `paint`'s `case Dropped` arm, separately, and the whole suite stayed green — while plan Task 3 Step 8, which demanded exactly that mutation, is ticked); `todaysQuestions` now prints "none could be looked up" for words whose lookup succeeded but which were unaskable, which is the *narrowed* survivor of PQ-1's false-message concern and I reproduced it; and four production doc comments still say a word "falls back to form 2.1", so the tree-wide sweep Task 4 Step 3 ticks is not clean.

## 1. Strengths

- **`ask` is the right seam.** `play_loop.go:951-969` puts `optionsFor` *before* the `Render`, so a triaged word is never wrapped, coloured and click-mapped for a form it will not take — and one closure owns both the question and the click map, so they cannot be built from different strings. This is the split the plan promised, delivered.
- **The drop is pinned where it actually lives.** `TestDroppingAWordOnABoardRemovesItFromTheDeck` drives `playSession` end-to-end by key *and* by click; deleting the `droppedBy` block from `advance` leaves `./cmd/define/play` entirely green and reddens only the main-package test — exactly the failure mode the test's own comment predicts. `TestABoardNamesTheWordItDropped`'s one-shot row reddens on removing `b.dropping = false`. Verified by reverting.
- **PQ-1's fix is real and pinned in both directions.** `packBoards` hands leftovers back rather than skipping them (`play_loop.go:846-858`), and `TestANarrowTerminalFallsBackToChoiceBeforeSkipping` asserts the mature-deck regression the gate caught.
- **`gradeKey` was made honest rather than patched** (`play_loop_test.go:161-181`). Finding that a dozen tests had silently stopped grading, and turning the helper into a probe with an explicit `Grid` arm, is the kind of repair that pays forward.
- **The keys row was re-cut inside its budget**, and `TestEveryModeSpellingIsTheSameWidthAndFitsEighty` charges the 80-column check that `TestTheRefusalRowIsNoWiderThanTheKeysRow` structurally cannot.

## 2. Critical findings

None.

## 3. Important findings

**(a) `cmd/define/play_loop.go:636` + `cmd/define/play/board.go:427` — the drop's colour is unpinned end to end.**
In a worktree at HEAD I deleted `Drop: "\x1b[2;9m"` from `boardPalette` and ran `go test ./cmd/define/...` → green. Separately I deleted `case Dropped: return b.pal.Drop` from `paint` and ran both packages → green. So a dropped cell could render identically to an unmarked one and nothing notices. `board_test.go:365`'s palette row asserts `pal.Yes`/`pal.No` only; the pty row at `pty_conformance_test.go:1077` asserts `pal.Yes` only and never enters drop mode. README.md:141 promises "struck-out for a drop" with nothing defending it, and plan Task 3 Step 8 — "Revert each of: the `Dropped` verdict mapping, `Apply`'s `Dropping` call, **the palette entry**. Each must redden a named test" — is ticked `[x]` while the issue's own Log quietly lists four verified pieces, none of them the palette.
*Fix:* extend `board_test.go`'s painted-cell row to a `Palette{Yes:…, No:…, Drop:…}` and assert `pal.Drop+"[1] mesa"+pal.Off` after a drop mark; add a `boardPalette(options{color:true}).Drop` assertion to the pty row (which would also give the drop its first live-terminal exercise).

**(b) `cmd/define/play_loop.go:1054` — the empty-sitting message names a cause the code did not establish.**
Reproduced: a one-word deck (`bases`) at `opt.width=12` yields stderr
`define: skipping "bases": this window is too short to draw a board and no multiple choice can be built for it` followed by `define: 1 words are due but none could be looked up`, and exit code 1. The lookup succeeded. This is PQ-1's false-message concern narrowed rather than removed — the population is now "a deck where no word can build a 2.3, on a terminal under `minWrapWidth`", i.e. exactly the young deck this issue is about, on a narrow window. Before #42 that learner got a working sitting.
*Fix:* count the non-lookup skips in the loop and branch the summary — "N words are due; none could be asked in this window" vs. the existing lookup wording. Pin with the reproduction above.

**(c) `cmd/define/optionpool.go:112,141,225` and `cmd/define/play/pick.go:155` — four production comments still route words to a form that no longer exists.**
All four are present-tense claims about current behaviour ("so the caller falls back to form 2.1", "a redirect falls back to form 2.1 — the same route `bases` takes", "The caller falls back to form 2.1, which is invisible to the learner…", "what lets the caller fall back to form 2.1 (D9)"). These are `currentTruthFiles`. PQ-7 enumerated four *other* files (`choice.go:16`, `board.go:121`, `play_loop.go:226`, `optionpool.go:14`) and exactly those four were fixed — the instance the finding named, not the class it implied (ARCH-PURPOSE). `atlas/define.md:2235` is a fifth site: it still enumerates `ReviewEvent.Form` as "`recall`, `meaning`, `board`", a set that is now two. Plan Task 4 Step 3 ticks a tree-wide grep and cites `workshop/lessons.md` ("a retraction is not done until `git grep` over the TREE is clean"); it is not clean.
*Fix:* sweep the four comments to say "the word is triaged onto a board" and drop `recall` from the atlas enumeration. Then make it mechanical the way this repo already does: the deletion of a whole *drawn* form is precisely what `retiredPhrases` (`repo_guard_test.go:1095`) exists for — but key it on the present-tense phrase (`"falls back to form 2.1"`, `"fall back to form 2.1"`), not the bare form name, because the plan deliberately keeps ~8 historical mentions (`choice.go:8,217`, `question.go:51,82`, `session.go:139,380`, `atlas:2087,2460,2597`) that a bare row would redden.

## 4. Minor findings

- `play_loop.go:977`, `:1005`, `:1033` — the same `Lookup` → skip-message → `ParseEntry` block appears three times in one function (was twice). A `entryFor(key) (Entry, bool)` closure beside `ask`, memoising into `parsed`, collapses all three (ARCH-DRY).
- `play/board.go:597` — `Dropped()` returns `b.cells[b.last].Word`, so correctness rests on `b.last` not having moved between the arming `Mark` and the read. `Rest` also writes `b.last` and does not touch `dropping`; today no path interleaves them, but storing the word at arm time (`b.dropWord = b.cells[i].Word`) removes the coupling entirely.
- `\x1b[2;9m` — SGR 9 (strikethrough) is not universal across terminals; SGR 2 (dim) carries it where 9 is ignored, which is fine, but worth a line in `boardPalette`'s comment since the README states the visual as a promise.
- `question.go:82` still illustrates `Keys()` with `"y = got it, n = missed it"`, and `doc_sync_test.go:325`'s comment says "falls back to form 2.1" — both cosmetic, both in the same family as (c).

## 5. Test coverage notes

- Mutation-verified by me at HEAD: `droppedBy` in `advance` → reddens `TestDroppingAWordOnABoardRemovesItFromTheDeck` (both subtests) and `TestADropOnABoardIsReportedOnce`; `b.dropping = false` → reddens `TestABoardNamesTheWordItDropped`; `Dropped.Verdict()` → reddens `TestADroppedCellRecordsNothing`. Those three claims hold.
- Not verified and not pinned: the palette (finding a).
- `TestADropOnABoardIsReportedOnce` is a good pin — it drives a Tab *and* a refused click after the drop, which is the specific double-emit PQ-10 was about.
- `TestUntestableWordsReachABoardAtEveryBox` has the third assertion that makes it falsifiable (a young testable word must still get 2.3). Its title says "at every box" while it covers box 0 untestable and box 5 mature-testable; an untestable word at box 5 takes the same branch, so this is fine, but the title over-promises slightly.
- The pty conformance suite has no drop row. Given the drop is irreversible and reaches `store.Forget`, a live-terminal row (Tab ×2, click a cell, assert the deck file) is the natural extension and would subsume finding (a)'s second half (ARCH-MOCK: the fake/live seam exists, but the new behaviour we depend on has no conformance check).

## 6. Architectural notes

- **ARCH-DRY** — flag, Minor: the triple lookup block above. Otherwise good: `packBoards` reuses `boardFits`, `choiceFor` reuses `optionsFor`, and the drop reuses `OutcomeDrop` rather than inventing a kind.
- **ARCH-PURE** — pass with a note. `optionsFor` and `packBoards` are pure and unit-tested; `play` stays import-allowlisted. The selection *rule* now lives inside the IO function `todaysQuestions`, so `TestTheBoxPicksTheForm` had to be re-pointed from a direct pure call to a whole-loop drive through fakes. The plan's Revision 3 argues this correctly (a pure `formFor(entry)` would force a lookup on every mature word), and the fake dict/store make it cheap — but it is a real reduction in how directly the rule is testable, and worth remembering if a fourth form arrives.
- **ARCH-PURPOSE** — flag: finding (c). The purpose was "form 2.1 does not exist and nothing says it does"; the code half is complete, the documentation half stopped at the sites a prior finding enumerated.
- **ARCH-MOCK** — pass with a note: dictionary, store, player and terminal all sit behind fakes, and `TestDroppingAWordOnABoardRemovesItFromTheDeck` runs the real production path against `store.Mem`. The gap is conformance coverage for the new gesture (above).
- **ARCH-CONSTRAINTS** — pass, and better than the plan claimed. The plan's PQ-8 answer says a Render is "newly paid on every MATURE board word"; the code does not do that — mature words are appended to `triage` before `Lookup` (`play_loop.go:971-974`), so lookups stay at one per due word and no new Render is paid. `packBoards` is ≤16 `boardFits` probes per board. One tiny waste: a young untestable word on a too-narrow terminal runs `optionsFor` twice (once in loop 1, once in the undrawable loop); bounded, pure, and only on that path.
- **ARCH-SECURE** — N/A. No new untrusted input, no credentials, no new external surface. `Dropped()`'s index is guaranteed by the arming path (see the Minor note for the latent coupling), and the drop routes through the pre-existing `store.Forget`, which keeps events — so the destructive action is recoverable by lookup, as the README states.

## 7. Plan revision recommendations

Three `## Revisions` entries for `workshop/plans/000042-retire-recall-form-plan.md`:

1. **"Task 3 Step 8's palette mutation was not performed."** The step is ticked and the issue Log names four verified pieces, none of them the palette; deleting `boardPalette`'s `Drop` entry or `paint`'s `Dropped` arm leaves the full suite green. Either add the pin and re-tick, or record the tick as unsupported.
2. **"The ARCH-CONSTRAINTS cost table describes a design that was not built."** The `todaysQuestions` block claims a Render and a click-region entry newly paid per mature board word; the implemented loop triages mature words before the lookup, so nothing new is paid. The plan should say what the code does — the claim is currently wrong in the conservative direction, which is the direction that stops a future reader from trusting the table.
3. **"Task 4 Step 3's tree-wide grep was not clean."** Four production comments (`optionpool.go:112,141,225`, `play/pick.go:155`) and `atlas/define.md:2235` still state form 2.1 / the `recall` form stamp as current, and no `retiredPhrases` row was added — so the next commit is not swept either.

```findings
findings:
  - id: new
    severity: Important
    family: pin-that-cannot-fail
    title: |
      The board's drop colour is unpinned: both the Palette.Drop entry and paint's Dropped arm can be deleted with the full suite green
    detail: |
      Verified by reverting at HEAD in a scratch worktree. Deleting `Drop: "\x1b[2;9m"` from boardPalette
      (play_loop.go:636) leaves `go test ./cmd/define/...` green; separately deleting `case Dropped: return
      b.pal.Drop` from paint (play/board.go:427) also leaves both packages green. So a dropped cell could paint
      identically to an unmarked one with nothing noticing. board_test.go:365 asserts Yes/No only and the pty row
      (pty_conformance_test.go:1077) asserts Yes only and never enters drop mode, while README.md:141 promises
      "struck-out for a drop". Plan Task 3 Step 8 names the palette entry as one of three required mutation checks
      and is ticked; the issue Log lists four verified pieces and the palette is not among them. Fix: assert
      pal.Drop around a dropped cell in board_test.go, and add a drop row to the pty suite so the gesture also
      gets a live-terminal check.
  - id: new
    severity: Important
    family: message-names-a-cause-the-code-did-not-establish
    title: |
      todaysQuestions reports "none could be looked up" for words whose lookup succeeded but which were unaskable
    detail: |
      Reproduced: a one-word deck (`bases`) at opt.width=12 prints the correct per-word skip line naming both
      causes, then `define: 1 words are due but none could be looked up` (play_loop.go:1054) and returns 1. The
      dictionary answered fine; the window was too narrow for a board and the deck could supply no distractors.
      This is PQ-1's false-summary concern narrowed rather than removed, and the surviving population is exactly
      this issue's subject — a young deck on a narrow terminal, which ran a full sitting before #42. Fix: track
      skips that were not lookup failures and phrase the summary accordingly ("none could be asked in this
      window"), pinned by the reproduction above.
  - id: new
    severity: Important
    family: retraction-not-swept-over-the-tree
    title: |
      Four production doc comments still route words to form 2.1, and the atlas still lists `recall` as a live form stamp
    detail: |
      optionpool.go:112, :141, :225 and play/pick.go:155 all state as CURRENT behaviour that a word "falls back to
      form 2.1"; atlas/define.md:2235 still enumerates ReviewEvent.Form as "recall, meaning, board". All are
      currentTruthFiles. PQ-7 named four different files (choice.go:16, board.go:121, play_loop.go:226,
      optionpool.go:14) and exactly those four were fixed — the instance, not the class (ARCH-PURPOSE). Plan Task 4
      Step 3 ticks a tree-wide grep and cites lessons.md's "a retraction is not done until git grep over the TREE
      is clean". Fix: sweep the five sites, then add a retiredPhrases row (repo_guard_test.go:1095) keyed on the
      present-tense phrase "fall(s) back to form 2.1" rather than the bare form name, since the plan deliberately
      keeps ~8 historical mentions a bare row would redden.
  - id: new
    severity: Minor
    family: duplicated-block-should-be-a-helper
    title: |
      The Lookup / skip-message / ParseEntry block is copy-pasted three times inside todaysQuestions
    detail: |
      play_loop.go:977, :1005 and :1033 carry the same four lines; the last two are byte-identical. An
      `entryFor(key) (Entry, bool)` closure beside `ask`, memoising into the existing `parsed` map, collapses all
      three and removes the chance that one of them drifts on the error path (ARCH-DRY).
  - id: new
    severity: Minor
    family: derived-state-depends-on-call-order
    title: |
      Board.Dropped() reads b.cells[b.last], so its correctness rests on b.last not having moved since the arming Mark
    detail: |
      play/board.go:592-598. `Rest` also writes `b.last` and does not clear `dropping`, so the two fields are only
      consistent because every Mark path calls advance immediately. Capturing the word at arm time
      (`b.dropWord = b.cells[i].Word` in Mark) makes the invariant local instead of a property of the caller.
  - id: new
    severity: Minor
    family: plan-claim-outruns-the-code
    title: |
      The plan's ARCH-CONSTRAINTS block claims a Render newly paid on every mature board word; the code pays none
    detail: |
      The plan's todaysQuestions bullet says the unified loop newly pays "a Render plus a click-region map entry on
      every MATURE board word". The implemented loop appends mature words to `triage` before any Lookup
      (play_loop.go:971-974), so they are never rendered and lookups stay at one per due word. The cost table
      should describe the code; a `## Revisions` entry is the right repair.
```

---

## Re-review — 2026-09-03T11:33:49-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 42 — retire form 2.1: the board is the fallback when a real test cannot be built |
| repo | tools |
| issue file | workshop/issues/000042-retire-recall-form.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6c5f1eb9d826a422af53a6940a1e53728ddcf019..e0238725958eaf3421bcd19ef826cc9f4192324f |
| command | sdlc close --issue 42 |
| reviewer | claude |
| timestamp | 2026-09-03T11:33:49-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

I ran the full window in a pinned scratch worktree at `e023872`: `go build`, `go vet`, `gofmt -l`, `go test ./...` and `go test -tags conformance ./cmd/define` all green, and I mutation-verified every claimed fix by reverting it. All three blocking findings from round 1 are genuinely addressed and pinned — deleting `Drop:` from `boardPalette` reddens `TestEveryBoardMarkHasAPaintedSequence`, deleting `case Dropped` from `Palette.For` reddens both that and `TestAMarkedCellIsPaintedAndKeepsItsKey`, reverting the empty-sitting branch reddens `TestAnEmptySittingNamesWhyItIsEmpty`, and the new `retiredPhrases` rows fire when a stale routing claim is reintroduced. The behaviour is correct: I checked the edge the drop opens (dropping the last unmarked cell) and the session advances and ends cleanly, and `packBoards`' all-or-nothing claim holds because a one-word board's height is word-independent. What blocks a clean SHIP is that BR-3's class was drawn too narrowly a second time: the round swept the symbol it was told about (`Recall`) and the phrase it was told about ("falls back to form 2.1"), but this window deleted a *second* symbol — `boardsFor` — and six current-truth sites still describe selection through it, including `atlas/define.md`'s Form 2.5 section, which is the atlas's own account of the rule this issue exists to change. The repo's guard for exactly this cannot see it, because `isCitableName` filters unexported names. Two smaller documentation gaps and a narrowed survivor of BR-2 follow the same shape.

## 1. Strengths

- **BR-1 was fixed at the class, and the class fix is real.** `play.Marks()` as the extent plus `Palette.For` as the one owner (`play/board.go:760-786`) means a fourth mark with no colour fails the day it is declared, and `TestEveryBoardMarkHasAPaintedSequence` derives its loop from `Marks()` rather than listing fields. I proved both halves by reverting; this is strictly better than the one assertion the review asked for.
- **BR-2's fix is pinned in both directions.** `TestAnEmptySittingNamesWhyItIsEmpty` has a second row asserting the *original* cause still reports as a lookup failure, so the fix cannot pass by simply renaming the message.
- **The drop is pinned where it lives, not where it is convenient.** `TestDroppingAWordOnABoardRemovesItFromTheDeck` drives `playSession` end to end by key *and* by click, asserts the neighbours survived, and asserts no review event was written — the three ways this could have gone wrong.
- **`advance` is the right seam for `Dropping`.** `s.Index++`/`s.Done` run *before* the `droppedBy` return and `SessionDone` is carried (`play/session.go:438-458`), so dropping the last unmarked cell does not strand the sitting — I verified this with a scratch driver.
- **`ask` puts `optionsFor` before the `Render`** (`play_loop.go:951-969`), so a triaged word is never wrapped, coloured and click-mapped for a form it will not take, and one closure owns both the question and the click map.

## 2. Critical findings

None.

## 3. Important findings

**(a) `atlas/define.md:2100`, `atlas/define.md:2138`, `cmd/define/play_loop.go:297`, `:654`, `cmd/define/play/board.go:216`, `:354` — `boardsFor` was deleted by this window and is still the current account of selection in six places.**
This is the **2nd finding in family `retraction-not-swept-over-the-tree`.** Round 1 named the instance (stale "falls back to form 2.1" claims) and the fix swept that phrase and added `retiredPhrases` rows for it. But the enumeration the class implies — *every symbol and every claim this window retired* — was never written, and `boardsFor` was deleted in the same window as `Recall` and entered no sweep. The worst site is `atlas/define.md:2098-2100`: "*Selection was a capability question until this: form 2.3 when the deck can supply distractors, 2.1 when it cannot. `boardsFor` partitions the day's keys at box ≥ 3…*" — the atlas's primary description of form selection, still describing the pre-`#42` rule with form 2.1 as live, while the atlas's *other* account (2445-2470) correctly describes `#42`. Two contradictory accounts of the same rule in one file.

Do not fix these six sites. **The rule is: a window that removes a top-level declaration owes a tree-wide sweep of that name regardless of its export status, and the guard must enforce it.** `TestARemovedDeclarationIsSweptOrRetired` (`repo_guard_test.go:1497`) already implements the sweep but gates it on `isCitableName` (`:1628`), which requires an exported or `Test*` name. I proved the hole: adding a one-line `if name == "boardsFor" { return true }` to `isCitableName` turns the guard red on `atlas/define.md`, `cmd/define/play/board.go`, `cmd/define/play_loop.go` and the plan file. The `isCitableName` filter exists for a good reason (`ids` vs "for-bids"), but its comment justifies it as "names that cannot plausibly BE prose" — a camelCase identifier like `boardsFor` is no more prose-like than an exported one. Widening the filter to *removed names ≥ 2 words in camelCase*, or to *any removed name containing an interior capital*, closes it without reintroducing the `ids` problem. Note the fix must also handle the plan file: `workshop/plans/000042-retire-recall-form-plan.md` legitimately names `boardsFor` in its "deleted" row, and it is a `currentTruthFile`, so either the row goes in `retiredSymbolNames` with the plan's deleted-status rows excluded, or the sweep runs before the plan lands.

**(b) `cmd/define/play/board.go:28`, `:33`, `:472`, `cmd/define/play/question.go:30`, `atlas/define.md:2183` — five current-truth comments restate an extent this window changed, and the code now owns that extent.**
- `board.go:28`: "*TWO marks and an ABSENCE, which is not a third mark*" — the const block **three lines below it** now declares `Dropped`, added by this diff.
- `board.go:33`: "*a cell has three states*" — four.
- `board.go:472`: "*Both spellings are the SAME WIDTH*" — `Keys()` directly beneath returns three, and its own test is named `TestEveryModeSpellingIsTheSameWidthAndFitsEighty`.
- `question.go:30`: "*NO SHIPPED FORM PRODUCES Skipped today… a verdict with no producer looks like dead code*" — `Mark.Verdict()` returns `Skipped` for `Dropped`, so `Board.Mark` now returns `(Skipped, true)`; the example it gives is a form that no longer exists.
- `atlas/define.md:2183`: "*The sequences come from `main` through `play.Palette` — green for yes, red for no*" — three sequences now, and the README promises the third ("struck-out for a drop", `README.md:141`).

This is a different rule from (a) — nothing was retracted; a count was restated. **The rule: a comment must not restate the cardinality or extent of a set the code enumerates.** The fix this window already built is the answer: `play.Marks()` *is* the extent, so these comments should defer to it ("every mark `Marks()` lists…") or drop the count entirely, exactly as `Keys()`'s own comment says the label set "is NOT enumerated" for the same reason. The `#42` Spec explicitly took "a comment describing the opposite of the code, on the key this issue changes" into scope; `board.go:28` is that comment, on the type this issue extended.

**(c) `cmd/define/play/board.go:760-786` — `play.Marks()` and `Palette.For` are new exported surface in the pure package with no atlas entry and no Core-concepts row.**
Both were added in the close round. `Marks()`' own doc comment positions it as a sibling of `BoardLabels` and `numRegionKinds` — and the atlas documents `numRegionKinds` twice (`:461`, `:2290`) precisely because it is that kind of mechanism. `atlas/define.md`'s palette paragraph (`:2183`) is the natural home and is the same paragraph finding (b) touches. The plan's Core-concepts table (`plan.md:20-29`) also has no rows for them; `TestPlanTablesNameEntitiesThatExist` only checks table → tree, so the reverse direction is unguarded and the table now under-describes the diff — the mirror of the correction its own Revision 2 made.

## 4. Minor findings

- **`cmd/define/play_loop.go:1066`** — the empty-sitting summary still misattributes on a *mixed* deck. Reproduced with `playRig(t, "bases", "rizz")` at `opt.width=12`: `rizz` fails lookup, `bases` is unaskable, `unaskable(1) != len(keys)(2)`, so the summary prints "*2 words are due but none could be looked up*" over a word the dictionary answered. This is the **2nd finding in family `message-names-a-cause-the-code-did-not-establish`**, so do not fix the branch condition. **The rule: an aggregate summary must be derived from a tally of the per-word outcomes, never from one counter compared against the total.** Every skip path already prints its own cause; either count each reason and phrase from the tally, or have the summary stop naming a cause at all ("none could be asked; see above") when the tally is mixed.
- `cmd/define/doc_sync_test.go:325` — "*The README must name EVERY reason a word falls back to form 2.1*". Test files are exempt from `currentTruthFiles` by design, so no guard will ever reach it; same class as (a)/(b).
- `\x1b[2;9m` (`play_loop.go:636`) — SGR 9 is not universal; SGR 2 carries it where 9 is dropped, which is why the pair works, but the README states the strikethrough as a promise and the palette comment does not record the fallback.

## 5. Test coverage notes

- Mutation-verified by me at `e023872`: `boardPalette`'s `Drop` entry → reddens `TestEveryBoardMarkHasAPaintedSequence`; `Palette.For`'s `case Dropped` → reddens that *and* `TestAMarkedCellIsPaintedAndKeepsItsKey`; the `unaskable` branch → reddens `TestAnEmptySittingNamesWhyItIsEmpty`; the `retiredPhrases` rows → fire on a reintroduced "falls back to form 2.1". Round 1's three blockers are all pinned by tests that fail without the fix.
- Every test the issue's Done-when names exists in the tree (checked all twelve by name).
- The pty conformance suite still has no drop row (`pty_conformance_test.go:1015-1090` enters neither drop mode nor a second Tab). The click path *is* covered end-to-end through `playSession` against `store.Mem`, so this is a live-terminal gap rather than an untested path — but the drop is the one irreversible action on the surface, and the pty row is where `#40` found its wrap Critical.
- Not covered anywhere: a board whose *last unmarked* cell is dropped. It works — I verified with a scratch driver that the sitting ends rather than stalling — but nothing pins it, and `advance`'s ordering (`s.Index++` before the `droppedBy` return) is what makes it work.

## 6. Architectural notes

- **ARCH-DRY** — flag, Minor: BR-4's triple `Lookup`/skip/`ParseEntry` block is unchanged at `play_loop.go:981`, `:1009`, `:1038`. Otherwise this round *improved* DRY meaningfully: `Palette.For` collapsed two owners of mark→sequence into one, and `packBoards`/`choiceFor` reuse `boardFits`/`optionsFor`.
- **ARCH-PURE** — pass. `optionsFor`, `packBoards`, `Marks`, `Palette.For` are pure and unit-tested without IO; the purity guard was updated (`purity_test.go:56`) rather than left scanning for a deleted type. The selection *rule* living inside `todaysQuestions` remains a real reduction in direct testability, correctly argued in plan Revision 3 — worth remembering if a fourth form arrives.
- **ARCH-PURPOSE** — flag: finding (a). The purpose was "form 2.1 does not exist and nothing says it does, and selection is one rule". The code delivers both; the documentation delivered the first and, at `atlas/define.md:2100`, still contradicts the second. A gate family that repeats across rounds is the ledger reporting the enumeration was never written, and here the enumeration is trivially mechanical — `git diff --name-status` plus `^-func` — which is why the fix belongs in the guard.
- **ARCH-MOCK** — pass with a note. Dictionary, store, player and terminal all sit behind fakes; the drop runs the production path against `store.Mem`. The seam is sound; the missing piece is a conformance row for the new gesture (§5).
- **ARCH-CONSTRAINTS** — pass, and better than the plan originally claimed. I confirmed mature words are appended to `triage` before any `Lookup` (`play_loop.go:971-974`), so lookups stay at one per due word and no new `Render` is paid; the plan's Revision now says so. `packBoards` is ≤16 `boardFits` probes per board. One bounded waste survives: a young untestable word on a too-narrow terminal runs `optionsFor` twice.
- **ARCH-SECURE** — N/A for new input or credentials. Worth recording that the drop is destructive and irreversible in-sitting, and that the blast radius is correctly bounded: it routes through the pre-existing `store.Forget`, which keeps events, so a lookup restores the word — and the README states that.
- Pre-existing, not this window's: `fallbackReasons` is single-sourced into the README by `TestREADMENamesEveryFallbackReason`, but `atlas/define.md:2447-2460` restates the same three reasons by hand with no pin. The atlas's own paragraph says the list was declared *because* it had drifted across code, README and atlas — and the atlas is still the undeferred consumer.

## 7. Plan revision recommendations

1. **"The tree-wide sweep enumerated one deleted symbol, not the window's removals."** Task 4 Step 3 is ticked and its grep was `'form 2\.1\|Recall'`. `boardsFor` was deleted by the same window and survives in six current-truth sites including the atlas's Form 2.5 selection paragraph. Record that the sweep's input set is "every top-level declaration this window removed", and that `TestARemovedDeclarationIsSweptOrRetired` could not catch it because `isCitableName` filters unexported names.
2. **"The Core-concepts table does not list the entities the close round added."** `play.Marks()` and `Palette.For` are new exported surface in the pure package and appear in no row. Plan Revision 2 already established that a table row has to describe what the diff did; this is the same obligation in the other direction, and nothing checks it.
3. *(Optional)* **"Task 3 Step 8's palette mutation is now performed."** The step was ticked before the check was run (recorded in `lessons.md`); it is now genuinely supported — I re-ran both halves. A one-line note closes the loop on the false tick inside the plan, where the tick lives.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Fixed at the class via play.Marks() + Palette.For; both mutations now redden named tests — I re-verified each by reverting at HEAD.
  - id: BR-2
    disposition: addressed
    note: |
      Branch added and pinned in both directions by TestAnEmptySittingNamesWhyItIsEmpty; reverting the branch reddens it. See the new Minor for the surviving mixed case.
  - id: BR-3
    disposition: addressed
    note: |
      All five named sites swept and retiredPhrases rows added; I confirmed the guard fires on a reintroduced claim. The class is re-raised as a new finding under the same family for boardsFor.
  - id: BR-4
    disposition: not-addressed
    note: |
      play_loop.go:981, :1009, :1038 still carry the same Lookup/skip/ParseEntry block. Minor, non-blocking.
  - id: BR-5
    disposition: not-addressed
    note: |
      Board.Dropped() still reads b.cells[b.last]; no dropWord field. Minor, non-blocking — no path currently interleaves Rest with an armed drop.
  - id: BR-6
    disposition: addressed
    note: |
      A 2026-09-03 "## Revisions" entry corrects the cost claim and the bullet is marked "(corrected at close)", which is the repo's append-don't-overwrite convention.
findings:
  - id: new
    severity: Important
    family: retraction-not-swept-over-the-tree
    title: |
      boardsFor was deleted by this window and is still the current account of selection in six current-truth sites, including the atlas's Form 2.5 section
    detail: |
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
  - id: new
    severity: Important
    family: comment-restates-a-count-the-code-owns
    title: |
      Five current-truth comments restate an extent this window changed, including Mark's own doc three lines above the const that falsifies it
    detail: |
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
  - id: new
    severity: Important
    family: new-surface-undocumented
    title: |
      play.Marks() and Palette.For are new exported surface with no atlas entry and no Core-concepts row
    detail: |
      play/board.go:760-786. Both were added in the close round as the single-source mechanism for the mark set.
      Marks()' own doc positions it beside BoardLabels and numRegionKinds — and the atlas documents numRegionKinds
      at :461 and :2290 precisely because it is that kind of mechanism — but atlas/define.md gained no entry for
      either, and the palette paragraph it belongs in (:2183) is the same one finding (b) leaves stale. The plan's
      Core-concepts table (plan.md:20-29) also has no rows for them; TestPlanTablesNameEntitiesThatExist only
      checks table -> tree, so the reverse direction is unguarded and the table now under-describes the diff — the
      mirror of the correction its own Revision 2 made.
  - id: new
    severity: Minor
    family: message-names-a-cause-the-code-did-not-establish
    title: |
      The empty-sitting summary still misattributes when the deck mixes lookup failures with unaskable words
    detail: |
      2ND FINDING IN THIS FAMILY — do not fix the branch condition. Reproduced at HEAD with
      playRig(t, "bases", "rizz") and opt.width=12: `rizz` fails lookup, `bases` is unaskable, so
      `unaskable(1) != len(keys)(2)` at play_loop.go:1066 and the summary prints "2 words are due but none could
      be looked up" over a word the dictionary answered fine. THE RULE: an aggregate summary must be derived from
      a tally of the per-word outcomes, never from one counter compared against the total. Every skip path already
      prints its own cause, so either count each reason and phrase from the tally, or have the summary stop naming
      a cause when the tally is mixed ("none could be asked; see the reasons above").
```

---

## Re-review — 2026-09-03T22:51:04-07:00 (unknown)

| field | value |
|-------|-------|
| issue | 42 — retire form 2.1: the board is the fallback when a real test cannot be built |
| repo | tools |
| issue file | workshop/issues/000042-retire-recall-form.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6c5f1eb9d826a422af53a6940a1e53728ddcf019..294c6c97e626279d0e22019819382d848b9e7c7f |
| command | sdlc close --issue 42 |
| reviewer | claude |
| timestamp | 2026-09-03T22:51:04-07:00 |
| verdict | unknown |

## Review

Failed to authenticate. API Error: 401 OAuth access token has been revoked.

---

## Re-review — 2026-09-03T23:11:35-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 42 — retire form 2.1: the board is the fallback when a real test cannot be built |
| repo | tools |
| issue file | workshop/issues/000042-retire-recall-form.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6c5f1eb9d826a422af53a6940a1e53728ddcf019..ecefdc0c945e44981358910951d8957956203847 |
| command | sdlc close --issue 42 |
| reviewer | claude |
| timestamp | 2026-09-03T23:11:35-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

All inspection commands ran clean. Full suite, `-tags conformance`, `go vet` and `gofmt -l` are green at HEAD; I re-ran each myself and mutation-verified the claimed class fixes in a pinned scratch worktree.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The behavioural work is sound and I found no correctness bug: `play.Recall` and `boardsFor` are gone, selection genuinely moved after the lookup without paying a second dictionary call (mature words reach `triage` before any `Lookup`; young words carry their parsed entry forward in `parsed`), `packBoards`' all-or-nothing claim holds because a one-cell board's `Rows()` is word-independent, and the drop landed as an injected capability asked at the one point both mark paths meet, one-shot and pinned end-to-end through `playSession` against the store fake. BR-1's class fix is real — I added a fourth `Mark` and `TestEveryBoardMarkHasAPaintedSequence` went red, which is what an extent-derived pin is supposed to do. What blocks a clean SHIP is that the *same* mutation showed the mark set now has an owner for colour and none for the prompt row, and that two of the three class-fixes this round shipped are hand-edits with no guard behind them: reverting the `isCitableName` widening — BR-7's entire declared rule fix — leaves `go test ./...` fully green, and the retraction sweep has now missed seven present-tense production comments for the third consecutive round.

## 1. Strengths

- **`TestEveryBoardMarkHasAPaintedSequence` (`cmd/define/play_loop_test.go:4223`) is a genuine extent pin.** I added an `Unsure` mark to the const block, `Marks()` and `Toggle`; the test failed with `mark 4 has no sequence`. `paint` delegating to `Palette.For` (`play/board.go:427`) removes the second owner that made BR-1 possible.
- **The `| deleted |` exemption in `currentTruthOnly` (`repo_guard_test.go:640-663`) is load-bearing and correctly placed.** Removing it reddens `TestNoArtifactNamesARetiredSymbol` on the plan file. Settling a two-guard conflict where "is this a record?" is already answered is the right move, and the exemption's self-limiting argument is honestly stated.
- **`TestDroppingAWordOnABoardRemovesItFromTheDeck` (`play_loop_test.go:1476`)** drives the key path *and* the click path through the real loop, reads the click column off what `Prompt()` drew rather than guessing it, and asserts the neighbours survived and no review event was written. That last assertion is the one that would catch a drop recorded as a miss.
- **`Dropping`'s placement in `advance` (`play/session.go:451`)** is right, and the one-shot contract is pinned in both directions — `TestABoardNamesTheWordItDropped` for the form, `TestADropOnABoardIsReportedOnce` for the loop (Tab + refused click after the drop).
- **`optionsFor`/`choiceFor` split (`optionpool.go:229`)** is a clean DRY seam: selection can ask "could this be a 2.3?" without paying a `Render`, and all three `fallbackReasons` stay on one side of the split so the declared list still describes one function.

## 2. Critical findings

None.

## 3. Important findings

**(a) `cmd/define/repo_guard_test.go:1677` — the `isCitableName` widening is BR-7's entire rule fix and no test fails without it.**
**This is the 2nd finding in family `pin-that-cannot-fail`.** Do not fix this instance by adding one assertion — state the rule. Verified by reverting in a scratch worktree at HEAD: replacing the interior-capital clause with `return false` leaves `go test ./...` green in every package (I ran the full 108s `cmd/define` suite, not just the guards). Separately, deleting `"boardsFor"` from `retiredSymbolNames` (`:1059`) is *also* green, because `currentTruthOnly`'s new exemption already strips the plan's mentions. So both halves of the round-2 fix are individually revertible with nothing noticing. The rule is already written down twice — `repo_guard_test.go:1477` ("A finding-fix with no test that reddens without it is not addressed, however plausible the diff") and `workshop/lessons.md` — and the mechanism already exists eight lines below that comment: `TestPlanStatusNormalisesToTheVocabulary` is a fixture table for `planStatus` precisely *because* "no plan in the tree writes one today, which is exactly why they must be fixtures". Fix: give `isCitableName` the same fixture table (`boardsFor`/`choiceFor` → true, `ids`/`binds`/`paint` → false, `Test*`/`Fuzz*`/exported → true, `""` → false), so the branch that BR-7 exists to add is exercised by something other than a window that happens to contain it.

**(b) `play/board.go:469`, `play/choice.go:8` and `:217`, `play/question.go:53`, `play/session.go:139` and `:380`, `store/event.go:34` — seven production comments still predicate on form 2.1 in the present tense.**
**This is the 3rd finding in family `retraction-not-swept-over-the-tree`.** Do not fix the seven sites; fix the rule. Measured prevalence: `git grep -n 'form 2\.1\|2\.1 or'` over non-test Go returns nine hits, seven of them present tense — `"where form 2.1 **is** a recall test"`, `"a concept form 2.1 **has** no answer for"`, `"form 2.1's y/n and form 2.3's 1/2/3/4 **are** the same shape"`, `"would make form 2.1 answer a question it **cannot**"`, `"which **is** what form 2.1 is"`, `"form 2.3 sets it; form 2.1 **cannot**"`, `"Tab reaches nothing at all on **2.1 or 2.3**"`. The atlas and README are clean, so this residue is entirely in Go doc comments — including the doc on the central `Question` interface and the doc on a persisted event field. `board.go:469` is three lines above the comment round 2 rewrote, which is the same shape BR-8 itself named. THE RULE: the sweep's input set is the retired **concept**, not only the removed **symbol** — and the discriminator round 1 needed but never wrote down is TENSE, not the bare name. Round 1 declined a `form 2.1` ban because ~8 historical mentions would redden; `retiredPhrases` already matches phrases case-insensitively over `currentTruthOnly` across production Go, so the enumeration belongs there as tense-keyed rows (`form 2.1 is`, `form 2.1 has`, `form 2.1 cannot`, `form 2.1's`, `2.1 or 2.3`), which leaves `"was"`/`"used to"`/`"before #42"` alone by construction.

**(c) `play/board.go:28`, `:480-481` and `cmd/define/play_loop.go:614` — the count class was closed by re-wording, and the wording still restates the count, including in the comment that declares the rule.**
**This is the 2nd finding in family `comment-restates-a-count-the-code-owns`.** Do not fix the wording again. `board.go:28` reads *"THREE marks and an ABSENCE, and `Marks()` below is the EXTENT — a count spelled in prose is a second owner of it"* — it spells the count in the same sentence that forbids it, and `:512` repeats "THREE now". `play_loop.go:614` still gives the palette as *"GREEN for yes, RED for no"* for a three-sequence palette. The sharp one is `:480-481`, which claims `TestEveryModeSpellingIsTheSameWidthAndFitsEighty` *"derives its loop from the cycle rather than counting the spellings here"* — `play/board_test.go:950` is `for range 3`. I proved the gap with the same fourth-mark mutation: the palette test went red, the spelling test stayed green, and `Keys()`' `switch` silently returned the `[yes]` default for the unwritten mode. So this window built the extent owner (`Marks()`) and wired it to the palette only; the prompt row — the row `#40` R11 says is the last thing a short window gives up — has no owner. Fix: derive the spelling loop from `play.Marks()` (or `len(Marks())-1`) and give `Keys()` a loud default, then let the comments defer to that instead of asserting a property the test does not have.

## 4. Minor findings

- `repo_guard_test.go:658-662` — the `| deleted |` exemption `ReplaceAllString`s the symbol out of the **whole document**, not just the row, so a plan carrying a deleted-row keeps a blind spot for that name everywhere in it. This plan already uses it: `plan.md` Step 8a still says *"`boardsFor` **is** called from five places"* and the PQ-6 block still says *"`boardsFor` **records** singles-first-then-boards"*, both present tense, both now invisible to the guard. Family: `guard-exemption-wider-than-its-warrant`. Also note the match is unanchored (no `\b`) and only fires on three-column rows, so a `deleted` row in the four-column Integration-points table gets no exemption at all.
- `cmd/define/pty_conformance_test.go:1015` — *"`d` MUST NOT BE OFFERED: Apply refuses the drop on a form holding many words"*. Apply no longer refuses; `session.go:261` hands `d` to a grid as an ordinary cell key. The assertion below it is still correct, only its stated reason is false. Pre-existing (the file is untouched by this window) and a test file, so out of the guard's `currentTruthFiles` — recorded as prevalence for finding (b) rather than as its own item.

## 5. Test coverage notes

- Every Done-when row has a named test and I confirmed each exists and runs; the plan-table guards enforce table↔tree in both the delete and the add direction after BR-9.
- **The drop gesture has no live-terminal coverage.** `pty_conformance_test.go` was not touched by this window and never enters drop mode, so the only evidence that `\x1b[2;9m` renders as a strikethrough on a real terminal is the unit assertion on the escape string. The issue's own `--verified` caveat ("I cannot press keys") covers the operator sitting; this is the mechanical half of the same gap, and BR-1's original fix sketch asked for it. Worth a follow-up row rather than blocking here.
- The mixed-cause empty-sitting path (BR-10) is the one uncovered branch I could reproduce as user-visible wrong output — see the disposition below for the captured stderr.

## 6. Architectural notes

- **ARCH-DRY — flag.** BR-4's three `Lookup`/skip/`ParseEntry` copies stand (`play_loop.go:981`, `:1009`, `:1038`); `paint`→`Palette.For` is a good consolidation; the mode-spelling extent is finding (c).
- **ARCH-PURE — pass, with a note.** `formFor` was correctly abandoned (Revision 3's argument holds: the box half must precede the lookup or every mature word pays a dictionary call), and the two halves that *are* pure — `optionsFor`, `packBoards` — were extracted and are unit-tested without IO. The residue is that the ordering rule itself now lives inside an IO loop and is only reachable through a dictionary fake. Acceptable given the fake is stateful and injected, but the next form added here will want the rule extracted rather than a fourth arm in that loop.
- **ARCH-PURPOSE — flag.** Findings (b) and (c) are both "answered the instance, not the class", at the third and second round respectively. The pattern across this gate is consistent: every fix that became *mechanism* (`Marks()`/`Palette.For`, the `currentTruthOnly` exemption) held under mutation; every fix that stayed *prose or an unpinned predicate* did not.
- **ARCH-MOCK — pass.** Dictionary and store fakes sit behind the same seam production uses, `playRig` boots the whole sitting from them, and a real-pty conformance suite exists for the live-terminal half (see the coverage note for what it doesn't yet cover).
- **ARCH-CONSTRAINTS — pass.** Lookups stay at one `Lookup` per due word — I read the loop rather than the cost table, and mature words are appended to `triage` before any fetch. `packBoards` costs at most 16 pure `boardFits` probes per chunk. The keys-row width budget is enforced for the three shipped modes; finding (c) is why a fourth would escape it.
- **ARCH-SECURE — pass.** The one new destructive path (`store.Forget`) takes a word the deck itself supplied, is reachable only from an explicitly-labelled mode two Tabs from the default, and keeps the event history, so the action is recoverable by re-lookup. No new untrusted input and no credentials.

## 7. Plan revision recommendations

- **`## Revisions` — "the mode spelling has no extent owner."** Record that `Marks()` was wired to the palette only, that `Keys()` and `TestEveryModeSpellingIsTheSameWidthAndFitsEighty` still enumerate three by hand, and that `play/board.go:480-481`'s claim the test "derives its loop from the cycle" was false when written.
- **`## Revisions` — "the deleted-row exemption is document-wide."** Note that `boardsFor` mentions anywhere in this plan are now invisible to `TestNoArtifactNamesARetiredSymbol`, and move Step 8a's *"is called from five places"* and the PQ-6 block's *"records singles-first-then-boards"* to past tense so the prose does not rely on the exemption to be tolerable.

```findings
dispose:
  - id: BR-4
    disposition: not-addressed
    note: |
      Still three copies at play_loop.go:981, :1009, :1038; Minor and explicitly declined with a stated reason.
  - id: BR-5
    disposition: not-addressed
    note: |
      play/board.go:594-599 still reads b.cells[b.last]; no dropWord field. No path interleaves Rest with an armed drop today.
  - id: BR-7
    disposition: addressed
    note: |
      Six sites swept (tree grep clean) and isCitableName widened — but the guard half is unpinned; raised as new under pin-that-cannot-fail.
  - id: BR-8
    disposition: addressed
    note: |
      All five named sites corrected; the residual restatements the fix itself introduced are raised as the 2nd in that family.
  - id: BR-9
    disposition: addressed
    note: |
      atlas/define.md:2468 documents both, plan Core-concepts gains Marks and Palette.For rows, and a Revisions entry explains the late addition.
  - id: BR-10
    disposition: not-addressed
    note: |
      Reproduced at HEAD with playRig("bases","rizz") at width 12 — stderr still prints "2 words are due but none could be looked up".
findings:
  - id: new
    severity: Important
    family: pin-that-cannot-fail
    title: |
      BR-7's rule fix — the isCitableName widening — can be reverted with the entire suite green
    detail: |
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
  - id: new
    severity: Important
    family: retraction-not-swept-over-the-tree
    title: |
      Seven production doc comments still predicate on form 2.1 in the present tense, three lines from the block round 2 rewrote
    detail: |
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
  - id: new
    severity: Important
    family: comment-restates-a-count-the-code-owns
    title: |
      The count class was closed by re-wording, and the new wording restates the count — including the comment that declares the rule, and a false claim about the test that pins it
    detail: |
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
  - id: new
    severity: Minor
    family: guard-exemption-wider-than-its-warrant
    title: |
      The `| deleted |` exemption strips the symbol from the whole document, not just the row, and this plan already relies on it
    detail: |
      repo_guard_test.go:658-662 ReplaceAllString's the row's symbol out of the entire artifact, so a document
      carrying one deleted-row gets a blind spot for that name everywhere in it. The plan already uses it:
      Step 8a still says "`boardsFor` is called from five places" and the PQ-6 block "`boardsFor` records
      singles-first-then-boards", both present tense and both now invisible to
      TestNoArtifactNamesARetiredSymbol. The comment's "self-limiting in two directions" claim is true across
      documents and symbols but not within a document. Also: the match is unanchored (no `\b`), and the
      HasSuffix("| deleted |") test only fires on three-column tables, so a deleted row in the
      four-column Integration-points table gets no exemption at all.
```

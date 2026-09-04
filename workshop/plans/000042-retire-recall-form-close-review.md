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

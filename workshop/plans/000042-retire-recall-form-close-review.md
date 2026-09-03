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

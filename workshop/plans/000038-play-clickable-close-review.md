# Boundary Review — tools#38 (whole-issue close)

| field | value |
|-------|-------|
| issue | 38 — the review loop's words are not clickable, because --play draws its own frames |
| repo | tools |
| issue file | workshop/issues/000038-play-clickable.md |
| boundary | whole-issue close |
| milestone | — |
| window | f56b34e5f54d9ba050246d3fde25e8036636f4ab..70b18a5e4cf62a461b6e6d3863dcfbbe313e192a |
| command | sdlc close --issue 38 |
| reviewer | claude |
| timestamp | 2026-08-31T20:24:15-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The issue's headline is delivered and the interesting parts are done well: `playRegion` really is one registry (there is exactly one `case RegionHeadword:` action switch left in non-test code, at `replraw.go:593`, and both loops reach it), `playsAudio()` really did absorb all four hand-copies, and the operator's "nothing seems clickable" regression is fixed *and* mutation-pinned — I reverted the per-line rule to all-or-nothing and `TestTheAskedWordIsClickableOnAMultipleChoiceQuestion` and `TestAWrapMovesTheClickMapRatherThanDroppingIt` both went red; I also reverted the shift, the reveal's `marksIn` call and the `KeyClick` branch, and each has a test that reddens. What blocks the gate is that the wrap the click map has to survive is now computed **twice from two different widths**: `writeClickable` is handed `opt.width` (fixed at startup) while the pinned screen wraps at `l.cols` (updated by `Resize`). After a mid-sitting narrowing resize the two disagree and the map is *misplaced rather than dropped* — the exact failure `wrapMovedRegions`' own doc comment, plan Done-when 3a, the atlas paragraph and the README paragraph all promise cannot happen. I reproduced it in the real loop against a real screen. Secondarily, the last commit changed the rule but left the atlas, the README and the plan's Done-when table describing the rule it replaced.

## 1. Strengths

- **`playRegion` is a real lift, not a copy** (`replraw.go:579`). The editor's `clicked` closure collapsed to one call, the indicator is the only parameter that differs, and `TestEveryRegionKindIsActionableThroughTheSharedRegistry` derives its loop from `numRegionKinds`, so a third kind is exercised the moment it is declared. `TestAnUnknownRegionKindPlaysNothing` pins refusal rather than a plausible fallback — that is the right shape.
- **T0's sweep is complete.** `grep noAudio`/`times > 0` over non-test code returns only the predicate's own definition and comments; all four sites derive. `TestPlayAnnouncedFetchesNothingWithAudioOff` asserts on the CDN fake's `Requested()`, not just the player, so "fetched but didn't play" cannot pass it (ARCH-PURPOSE, ARCH-DRY).
- **`marksIn` locates the render inside the reveal instead of counting from a formula** (`play_loop.go:573`). That is the right call — the form owns its layout — and `TestPlayARevealedDefinitionCarriesItsRegions` drives the loop against a real `newPinnedScreen` and asserts a *click resolves*, so the loop's offset arithmetic and the screen's rebasing are pinned at their joint. Reverting the regions to `nil` reddens it.
- **`wrapMovedRegions` asks `wrapWritten` itself** rather than re-deriving the erase-gesture exemption and the sub-20 policy (`playbar.go:200`). Given the design chose to compute the wrap at the caller, this is the correct way to do it.
- **`lessons.md` records the class, not the instance** — "every sentinel-valued default in a rig is a state production may not have" — and the rig was actually changed to `defaultCols`, which is what makes the lesson load-bearing.

## 2. Critical findings

**`cmd/define/play_loop.go:179` and `:358` — `writeClickable` is given `opt.width`, a second and stale answer to a question the screen owns.**

`opt.width` is set once from `terminalWidth(stdout)` (`main.go:528`) and never updated; the loop's own resize case deliberately does not re-derive it, with the comment *"Setting a second width here would be a second answer to the same question"* (`play_loop.go:222`). But `writeClickable(..., opt.width)` is exactly that second answer, while `liveScreen.writeBuffer` wraps at `l.cols`, which `Resize` does update. After a narrowing resize the caller believes nothing broke, the screen breaks lines anyway, and `addRegions` files every region at an unshifted buffer line.

Reproduced against the real loop and a real `newPinnedScreen` (start 80 cols, `Resize(400, 40)`, reveal each question), asserting that each recorded region underlines its own text:

```
MISPLACED: region "concrete"  (kind 0) on line 18 col  0 underlines ""; line=""
MISPLACED: region "con·crete" (kind 0) on line 18 col 10 underlines ""; line=""
MISPLACED: region "French"    (kind 1) on line 54 col 58 underlines ""; line="        “the post is concreted into the"
start=80 then=40: 3 regions, 3 misplaced   (start=80 then=80: 0 misplaced)
```

So the underline is painted over unrelated text (or nothing), the word you can see is inert, and a click at a shifted coordinate plays a word you did not point at. Silent, and it contradicts four written promises: `wrapMovedRegions`' *"Dropped rather than placed by guess"*, plan Done-when 3a, `atlas/define.md:2055`, `README.md:159`.

Fix sketch — give the width one owner. Cleanest is to move the region-moving *into* `liveScreen.WriteRegions` (`screen.go:637`): it already holds `l.cols` and already routes through `writeBuffer`, its own comment already says the wrap lives there *"so that is true of every path rather than of the one anybody thought about"*, and its BR-25 note explicitly hands this obligation to the first consumer. Then `writeClickable`'s `width` parameter disappears, `#40`'s board (named in that comment as the next caller) inherits the guarantee for free, and the wrap stops running twice per write. A cheaper stopgap is a `Cols() int` on `display` so the loop asks the screen. Either way, add a loop-level row: resize narrower before a reveal, then assert every region in the map still underlines its own `Text` — that is the test this diff is missing, and it is what would have caught this.

## 3. Important findings

**`atlas/define.md:2055-2062` and `cmd/define/README.md:159-160` describe the rule commit `70b18a5` replaced.** Both still say the map is passed along *"only if it changed nothing"* / that "the underlines stop appearing until the next question". The rule is now per line — a region on an unbroken line is *moved*, and only regions on broken lines are dropped — which is precisely why the operator's multiple-choice case works now. `70b18a5` touched `playbar.go`, `play_loop_test.go`, `lessons.md` and the plan, and left both docs behind. The README sentence is also the user-visible one, and it is currently wrong in the common case (at a sitting's own width the glosses *do* wrap and the underline *does* survive). This is the docs half of the Critical: whichever way that is fixed, both paragraphs need rewriting to the rule the code has.

**`workshop/plans/000038-play-clickable-plan.md:151` — Done-when row 3a names `TestClickMapIsDroppedRatherThanMisplacedByAWrap`, which does not exist in the tree.** The test is `TestAWrapMovesTheClickMapRatherThanDroppingIt`, and the row's *claim* ("DROPPED rather than misplaced when a wrap would move it") no longer states the rule either. This is the `plan-table-vs-tree` family the plan itself flags, in the same document whose 2026-08-31 revision is titled "the Done-when names the tests that exist" — the fix commit updated the Revisions prose and left the table.

## 4. Minor findings

- `replraw.go:554-563` — `replayInPlace`'s doc comment now runs straight into `playRegion`'s with no blank line, so godoc attributes *"replayInPlace speaks the current word again…"* to `playRegion`, and `replayInPlace` (`:603`) is left undocumented. One blank line and moving the paragraph down.
- `play_loop.go:447/465/500` — `held.marks` is initialised to an empty map, a second local `marks` is built, and the local is assigned over it at the end. One map would do; build into `held.marks` directly.
- `play_loop.go:270` — a sitting's click always passes `entry: ""`, so a `RegionOriginLang` click degrades to `#29`'s headword fallback (`utteranceFor` → `SourceSpellings(word, ParseEntry(""))`) and says the English spelling in the origin voice. `todaysQuestions` has the raw `text` in hand at `:475` and discards it; one more field on `clickable` would close it. Worth noting because the README claims the reveal is clickable "exactly as in the interactive session", where the current entry's source spellings *are* used.
- `replraw.go:584` — `playRegion` writes `nothingToReplay` with `Fprintln` (`\n`) while `replayInPlace:612` writes the same constant with `\r\n`. Invisible today (both stream to the screen, which normalises), but `#32`'s whole point is that line endings have one owner.
- `workshop/issues/000038-play-clickable.md:3` — still `status: punt` with every Plan row ticked; the resume never moved it off.

## 5. Test coverage notes

- Mutation-verified by me, all red as claimed: all-or-nothing rule, keeping regions on broken lines, dropping the `r.Line = start[r.Line]` shift, `nil` regions on the reveal, and an inert `KeyClick` branch. The claimed fix in `70b18a5` is genuinely pinned.
- The gap is the one the Critical lives in: **no test drives the loop across a resize with regions.** `TestPlayRepaintsOnResize` asserts a repaint and nothing about the map; `TestAWrapMovesTheClickMapRatherThanDroppingIt` is pure and is handed a single consistent width, so it cannot see two widths disagreeing. Note also that `r.Line = start[r.Line]` never moves anything in production at a consistent width (the only region above a broken line is the prompt's, on line 1) — the shift's *only* live exercise is the resize case, where it currently runs on the wrong width.
- `TestPlayClickIsNotAnAnswer` pins the negative correctly but would also pass with the `KeyClick` branch deleted (`toInput` rejects `KeyClick` anyway); the positive is carried by `TestPlayClickOnThePromptWordPlaysIt`, which is fine — worth knowing which row is doing which job.
- Every `TestPTYPlay*` row skips here (`no pty available: operation not permitted`), including Done-when 5, 6b, 7 and 8. That is `#37`'s environment limitation, not this diff's, but it means the close's manual terminal pass is load-bearing — and given the finding above, **that pass should include resizing the window mid-sitting and then clicking**, not only clicking and paging.

## 6. Architectural notes

- **ARCH-DRY — flag.** Consolidation of the audio predicate and the click registry: pass, verified by sweep. But the wrap policy is now computed in two places from two sources (`wrapMovedRegions` at the caller, `writeBuffer` inside the screen), and the wrap literally runs twice per write. That duplication *is* the Critical; the consolidation target already exists at `screen.go:637`.
- **ARCH-PURE — pass.** `wrapMovedRegions` and `marksIn` are pure and testable without a terminal; `writeRendered`'s interface dispatch is untouched; `playRegion` is honestly labelled INTEGRATION and tested through the repo's fakes rather than being called "pure" and mocked.
- **ARCH-PURPOSE — pass, with one remaining instance.** The response to the operator's report answered the class, not the site: per-line rule + the rig's sentinel default fixed + the generalisation in `lessons.md`. The class *"a region's coordinates are only valid against the width they were computed at"* is, however, closed for the wrap and left open for the **width source** — the enumeration has one more row, and it is the Critical.
- **ARCH-MOCK — pass.** The player and CDN stateful fakes are the seam for every new audio row (`rig.cdn.Requested()`, `rig.player.count()`), the screen is exercised as a real object in the loop tests, and the pty rows cover what only a terminal answers.
- **ARCH-CONSTRAINTS — pass, noted.** No new budget claimed and none needed: `wrapMovedRegions` is O(lines-of-one-entry) per transition write, not per frame. Two observations for later: the wrap runs twice per write (removed by the consolidation above), and a click blocks the loop for the fetch plus `times` playbacks — consistent with the reveal's existing playback, and Ctrl-C still reaches the reader through `ctx`, so this is a note rather than a finding.

## 7. Plan revision recommendations

Add a `## Revisions` entry covering:
- **Done-when row 3a** — rename the pin to `TestAWrapMovesTheClickMapRatherThanDroppingIt` and restate the claim as the rule the code has ("a region on a line the wrap does not break is MOVED to where that line lands; only regions on broken lines are dropped"), with the red-when updated to match.
- **A new row for the width source** — "the click map is computed against the width the screen will actually wrap at", red when `opt.width` and the screen's `cols` can disagree, pinned by a loop-level resize-then-reveal test. Without this row the plan's table claims a guarantee the tree does not provide.
- **T5's resolution paragraph** (the 2026-08-31 entry) — its closing claim, "after a NARROWING resize the underlines stop until the next question is written", is falsified above; correct it to what actually happens today and to what the fix will make happen.

```findings
findings:
  - id: new
    severity: Critical
    family: wrap-width-single-owner
    title: |
      writeClickable is handed opt.width while the screen wraps at its own cols, so a resize MISPLACES the click map instead of dropping it
    detail: |
      play_loop.go:179 and :358 pass opt.width, fixed at startup by terminalWidth
      and never updated; liveScreen.writeBuffer wraps at l.cols, which Resize does
      update. Reproduced in the real loop (start 80 cols, Resize to 40, reveal):
      all three regions land on lines that do not contain their text - a headword
      region on a blank line, the ORIGIN "French" region on a quotation line. That
      is the wrong-click failure wrapMovedRegions' own comment, plan Done-when 3a,
      atlas and README all promise cannot happen, and the loop's own resize comment
      says a second width here "would be a second answer to the same question".
      Fix by moving the region-moving into liveScreen.WriteRegions, which already
      holds l.cols and already routes through writeBuffer, and add a loop-level
      resize-then-reveal test asserting every region still underlines its own text.
  - id: new
    severity: Important
    family: docs-lag-behavior-change
    title: |
      atlas and README still describe the all-or-nothing wrap rule that commit 70b18a5 replaced
    detail: |
      atlas/define.md:2055-2062 and cmd/define/README.md:159-160 both say the map is
      passed along "only if it changed nothing" and that the underlines stop after a
      narrowing resize. The rule is now per line, and the resize claim is false as
      written. 70b18a5 updated playbar.go, the tests, lessons.md and the plan prose
      but neither doc.
  - id: new
    severity: Important
    family: plan-table-vs-tree
    title: |
      Done-when row 3a names TestClickMapIsDroppedRatherThanMisplacedByAWrap, which does not exist
    detail: |
      The test in the tree is TestAWrapMovesTheClickMapRatherThanDroppingIt, and the
      row's claim ("DROPPED rather than misplaced") no longer states the rule either.
      Same family the plan flags against itself, in the document whose latest revision
      is titled "the Done-when names the tests that exist".
  - id: new
    severity: Minor
    family: doc-comment-attachment
    title: |
      replayInPlace's doc comment now heads playRegion, leaving replayInPlace undocumented
    detail: |
      replraw.go:554-563 runs straight into playRegion's comment with no blank line,
      so godoc reads "replayInPlace speaks the current word again..." as playRegion's
      doc.
  - id: new
    severity: Minor
    family: redundant-duplicate-state
    title: |
      todaysQuestions builds two marks maps where one would do
    detail: |
      play_loop.go:447 initialises held.marks, :465 builds a second local map, :500
      assigns it over the first. Build into held.marks directly.
  - id: new
    severity: Minor
    family: available-context-discarded
    title: |
      a sitting's ORIGIN-language click always passes an empty entry, degrading to the headword fallback
    detail: |
      play_loop.go:270 passes entry "", so utteranceFor gets ParseEntry("") and plays
      the English spelling in the origin voice. todaysQuestions has the raw text at
      :475 and discards it; one field on clickable would close it. The README claims
      the reveal is clickable "exactly as in the interactive session", where the
      current entry's source spellings are used.
  - id: new
    severity: Minor
    family: line-ending-single-owner
    title: |
      playRegion writes nothingToReplay with \n while replayInPlace writes the same constant with \r\n
    detail: |
      replraw.go:584 vs :612. Invisible today because both stream to the screen,
      which normalises, but the constant now has two spellings of its terminator.
  - id: new
    severity: Minor
    family: tracker-state-stale
    title: |
      the issue is still status: punt with every Plan row ticked
    detail: |
      workshop/issues/000038-play-clickable.md:3 - the resume never moved the status
      off punt.
```

---

## Re-review — 2026-08-31T22:34:51-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 38 — the review loop's words are not clickable, because --play draws its own frames |
| repo | tools |
| issue file | workshop/issues/000038-play-clickable.md |
| boundary | whole-issue close |
| milestone | — |
| window | f56b34e5f54d9ba050246d3fde25e8036636f4ab..5b764c082d97f29fccdb3680e6abe3528575b72a |
| command | sdlc close --issue 38 |
| reviewer | claude |
| timestamp | 2026-08-31T22:34:51-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The code this round is genuinely good and the round-1 Critical is really fixed: `wrapMovedRegions` now runs inside `liveScreen.WriteRegions` against `l.cols`, and I mutation-verified it twice in a scratch copy — deleting the call, and substituting a hardcoded 80 for `l.cols`, each reddens `TestAResizeDoesNotMisplaceTheClickMap` and `TestWriteRegionsMovesTheMapByTheScreensOwnWidth`. `Line: 1` for the prompt region reddens two tests when reverted to `0`. What blocks the gate is the plan artifact, and it is not bookkeeping: the Core-concepts table names two PURE entities (`playRegions`, `promptRegionFor`) that the tree does not declare, and `## Tasks` still shows T0/T1/T4/T5/T6/T7/T8 unticked. The unticked boxes are exactly what suppresses `TestPlanTablesNameEntitiesThatExist`'s check — tick them (which `sdlc close`'s plan-unchecked gate requires) and the suite goes **red** on those two names. I reproduced that. Separately, the fix commit's message asserts "All three are rows in T5 now" for BR-1 and the T5/T4 rows are byte-identical to before; BR-2, BR-6, BR-7 and BR-8 are untouched.

## 1. Strengths

- **BR-3's fix is the right shape and is pinned by tests that fail without it.** `screen.go:655` moves the arithmetic to the one holder of `l.cols`, and the `width` parameter is gone from every caller, so the second ruler is unexpressible rather than merely unused. `wrapMovedRegions` measures rows by calling `wrapWritten` itself (`playbar.go:190`), so the erase-gesture exemption and the sub-20 policy cannot differ between the two sides — the wrap and the map are literally the same function.
- **The underline and the hit test share one source.** `Paint` marks clickable spans from `s.regions[top+i]` (`screen.go:430`) and `RegionAt` reads the same map, so a moved region moves its underline too. There is no second path that could disagree.
- **T0's sweep is complete, not partial.** `grep noAudio` over non-test code returns only the flag, the field, the constructor and the predicate's own definition — all four hand-copies derive. `TestPlayAnnouncedFetchesNothingWithAudioOff` asserts on the CDN fake's `Requested()`, so "fetched but didn't play" cannot pass it (ARCH-DRY, ARCH-MOCK).
- **`playRegion` is one registry, verified rather than claimed.** Exactly one `case RegionHeadword:` action switch exists outside tests (`replraw.go:592`), both loops reach it, `TestEveryRegionKindIsActionableThroughTheSharedRegistry` derives its loop from `numRegionKinds`, and `TestAnUnknownRegionKindPlaysNothing` pins refusal instead of a plausible fallback.
- **`marksIn` locates the render inside the reveal** (`play_loop.go:581`) rather than counting from a formula the form owns, and `TestPlayARevealedDefinitionCarriesItsRegions` drives the loop against a real `newPinnedScreen` and asserts a *click resolves* — the joint, not the pieces.

## 2. Critical findings

**`workshop/plans/000038-play-clickable-plan.md:88-89` — the Core-concepts PURE table names `playRegions` and `promptRegionFor`; neither exists anywhere in `cmd/`.** What was delivered is `clickable` (`play_loop.go:519`) and `(*sittingDeck).marksIn` (`play_loop.go:581`); the prompt region is built inline in `show()` and has no named function at all. Reproduced blast radius, in a scratch copy: with the plan's task boxes as they are, `TestPlanTablesNameEntitiesThatExist` PASSES (the `inProgress` exemption honours a `new` row while the plan is unfinished); tick the boxes and it fails with both names. So the close gate's plan-unchecked requirement and the repo's own guard are in direct conflict right now. Fix: rewrite the two rows to `clickable` and `marksIn` with their real kinds, then tick.

## 3. Important findings

**`workshop/plans/000038-play-clickable-plan.md:120-136` — the plan's `## Tasks` still show T0, T1, T4, T5, T6, T7, T8 as `- [ ]` while the issue's `## Plan` ticks all nine and the code is in the tree (T6 even carries "LANDED BY #41" in the issue and is unticked here).** This is the 2nd finding in family `tracker-state-stale` — do not fix this instance alone. The rule that covers both: **at a boundary, an issue's completion state lives in four places and they are swept as one enumeration, not one per finding** — issue frontmatter `status:`, the issue's `## Plan` boxes, the plan's `## Tasks` boxes, and the referencing project row. BR-10 fixed slot 1; slot 3 was left, and slot 4 (`workshop/projects/define-learn.md` has no `[tools#38]` row at all, unlike `[tools#41]`) has never been checked. Measured prevalence this issue: 2 of 4 slots wrong at HEAD after a round that named one of them.

## 4. Minor findings

- `cmd/define/play/choice.go:113-114` and `cmd/define/README.md:159` — two prose claims falsified by this window's own commits: `Choice.Prompt`'s doc says `#38` "computes its region as line 0, column 0" and that "#38 is PARKED" (it is line 1, and the issue is `working`); the README says "Narrow the window and the links follow the text as it re-wraps", but `Resize` only updates `l.rows/l.cols` — no buffer line is re-wrapped and no existing region moves, only text written *after* the resize. 2nd in family `docs-lag-behavior-change`; the rule: **a behaviour change sweeps every prose site that names the behaviour, found by grepping the changed symbol and the issue number, not by remembering which doc mentioned it.** `choice.go`'s comment exists specifically to protect this invariant, so its being stale defeats its only purpose.
- `cmd/define/replraw.go:584`, `:616`, `cmd/define/repl.go:313` — three identical copies of `if !opt.playsAudio() { Fprintln(stderr, nothingToReplay) }`. D11 justified leaving the message with the callers because "callers keep their own MESSAGES"; all three messages are the same constant, so the justification does not hold for these. ARCH-DRY.

## 5. Test coverage notes

- Everything green: `go test ./...` passes (`cmd/define` 108s). Mutation-verified in a scratch copy: removing the `wrapMovedRegions` call, substituting a fixed width for `l.cols`, and reverting `Line: 1`→`0` each redden at least one test.
- **One gap.** Deleting `Word: key` from the `Render` call (`play_loop.go`) leaves the entire suite green — I ran the full `cmd/define` package with that mutation and the only failures were the git-dependent repo guards (scratch has no `.git`). `regionsIn` falls back to `e.Headword()`, so the `jalapeno`/`jalapeño` divergence the fix commit calls "a live defect — a click on a normalised deck word would have fetched the wrong recording" has no pin. A row asserting that a deck key differing from the entry's headword yields `Region.Word == key` would close it.
- `TestPlanNamedTestsExist` currently **skips** on this plan ("no finished unit of work names a test") for the same reason as the Critical, so BR-5's rename is correct but unguarded until the boxes are ticked. Once ticked it passes — I verified.

## 6. Architectural notes for upcoming work

- **ARCH-DRY** — pass, with the `nothingToReplay` triple noted above. The `playsAudio()` and `playRegion` consolidations are the real thing: both were swept to zero remaining copies, not merely given an owner.
- **ARCH-PURE** — pass. `wrapMovedRegions` and `marksIn` are pure and the IO sits at `WriteRegions`/`playRegion`; the pure rule is unit-tested with no screen. (The plan's PURE *table* is wrong, but that is the Critical, not a purity violation.)
- **ARCH-PURPOSE** — shadow-sweep on "one registry": one action switch outside tests, both loops derive; `wrapMovedRegions` has exactly one caller. Flag: BR-8 is the deferred *purpose* rather than an extension — README promises the reveal is clickable "exactly as in the interactive session", and a sitting's ORIGIN click always passes `entry ""`, so `#29`'s source spellings are never used even though `todaysQuestions` holds the raw text and throws it away. One field on `clickable` closes it.
- **ARCH-MOCK** — pass. Player and CDN fakes behind the same seam; the tests drive the real `newPinnedScreen` into a `syncBuf`, so production and test share the boundary.
- **ARCH-CONSTRAINTS** — pass, one note for `#40`'s board: `WriteRegions` now wraps twice per call (`wrapMovedRegions` per line to measure, `writeBuffer` on the whole text to apply). Harmless at a sitting's cadence — once per question/reveal, bounded by `-count` — but the close-review's fix sketch predicted the move would *stop* the double wrap and it did not. If the board writes regions per frame, return the wrapped text from the measuring pass.

## 7. Plan revision recommendations

- A `## Revisions` entry rewriting Core concepts: `playRegions` → `clickable` (struct, `play_loop.go`, new, PURE) and `promptRegionFor` → `(*sittingDeck).marksIn` (`play_loop.go`, new, PURE), recording that the prompt region is built inline in `show()` and has no named entity — the row was written before `#41` deleted `draw`.
- The same entry should tick T0/T1/T4/T5/T7/T8 and mark T6 `[x]` as landed by `#41`, and state the four-slot completion enumeration from the Important above.
- T4's row still reads "`draw` is append-only, so the word lands on the line about to be written… at column 0" — `draw` does not exist and the region is line 1. T5 still reads "carries its regions, through `writeRendered`" with none of the three obligations BR-1 asked to be made checkable. These are the rows the fix commit claimed to have changed.
- The "What this plan asserts about the existing tree, verified" table needs re-anchoring by symbol: I checked ten anchors and eight point at unrelated lines at HEAD (`play_loop.go` 283/64/262/174/178, `replraw.go` 264/170/366); only `main.go:801` and `play/recall.go:29` still land. Two rows are also false in substance (`replraw.go:264` "the click registry closes over its session" — it is `playRegion` now; `replraw.go:366-380` "the viewport keys — `--play` needs its own cases" — `viewportGesture` is already shared).

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      code side fixed (Line 1 and the screen-width ruler are both mutation-pinned; Word:key is present but unpinned), but T4/T5 in `## Tasks` are byte-identical to before — the commit's "All three are rows in T5 now" did not happen.
  - id: BR-2
    disposition: not-addressed
    note: |
      the current-truth table was not touched; 8 of 10 anchors I re-measured at HEAD point at unrelated lines, and two rows are false in substance.
  - id: BR-3
    disposition: addressed
    note: |
      mutation-verified twice in a scratch copy — deleting the call and substituting a fixed 80 for l.cols each redden two tests.
  - id: BR-4
    disposition: addressed
    note: |
      atlas/define.md and README.md both rewritten to the per-line rule and the one-ruler rule; one new inaccuracy in the README raised separately.
  - id: BR-5
    disposition: addressed
    note: |
      row 3a renamed and restated, row 3b added; TestPlanNamedTestsExist passes on the plan once the task boxes are ticked.
  - id: BR-6
    disposition: not-addressed
    note: |
      replraw.go:554-563 still runs into playRegion's comment with no blank line; replayInPlace's own doc at :613 does not name it.
  - id: BR-7
    disposition: not-addressed
    note: |
      play_loop.go still initialises held.marks at the constructor, builds a second local map, and assigns it over the first.
  - id: BR-8
    disposition: not-addressed
    note: |
      play_loop.go:272 still passes entry "" — and this is the issue's stated parity promise, not an extension (ARCH-PURPOSE).
  - id: BR-9
    disposition: addressed
    note: |
      both sites now use fmt.Fprintln; the \r\n spelling is gone from this path.
  - id: BR-10
    disposition: addressed
    note: |
      issue frontmatter is status: working; the wider sweep it was one slot of is raised as a new finding.
findings:
  - id: new
    severity: Critical
    family: plan-table-vs-tree
    title: |
      the plan's Core-concepts PURE table names playRegions and promptRegionFor, neither of which the tree declares
    detail: |
      This is the 2nd finding in family `plan-table-vs-tree`. Earlier rounds fixed
      instances (BR-5 renamed one Done-when cell). Do NOT fix this instance alone —
      the rule is that EVERY identifier a plan states as current truth is checked
      against the tree in one sweep at the boundary, and the repo already owns the
      enumerator: TestPlanTablesNameEntitiesThatExist for Core-concepts cells,
      TestPlanNamedTestsExist for backticked test names. Both are currently
      suppressed on this plan by its unticked task boxes.
      Measured: the delivered entities are `clickable` (play_loop.go:519) and
      `(*sittingDeck).marksIn` (play_loop.go:581); the prompt region is built inline
      in `show()` and has no named function. Reproduced in a scratch copy — with the
      boxes as they are the guard PASSES; tick them (which `sdlc close`'s
      plan-unchecked gate requires) and it fails on both names. So the close cannot
      be recorded without either a red suite or a corrected table.
  - id: new
    severity: Important
    family: tracker-state-stale
    title: |
      the plan's Tasks still show T0/T1/T4/T5/T6/T7/T8 unticked while the issue ticks all nine and the code has landed
    detail: |
      This is the 2nd finding in family `tracker-state-stale`. BR-10 fixed one
      instance (issue frontmatter status). Do NOT fix this instance alone — the rule:
      an issue's completion state lives in FOUR slots and is swept as one enumeration
      at the boundary: issue frontmatter `status:`, the issue's `## Plan` boxes, the
      plan's `## Tasks` boxes, and the referencing project row. Measured at HEAD: slot
      1 fixed by BR-10; slot 3 wrong (plan lines 120-136, including T6 which the issue
      marks landed by `#41`); slot 4 never checked — workshop/projects/define-learn.md
      carries a `[tools#41]` row and no `[tools#38]` row at all. 2 of 4 wrong after a
      round that named one of them.
      This is also what hides the Critical above: TestPlanTablesNameEntitiesThatExist
      exempts `new` rows while a plan has unticked steps, and TestPlanNamedTestsExist
      skips the document entirely ("no finished unit of work names a test").
  - id: new
    severity: Minor
    family: docs-lag-behavior-change
    title: |
      Choice.Prompt's doc still says the region is line 0 and that 38 is PARKED, and the README says links follow the text as it re-wraps
    detail: |
      This is the 2nd finding in family `docs-lag-behavior-change`. BR-4 fixed the
      atlas and README paragraphs for the wrap rule. Do NOT fix these two instances
      alone — the rule: a behaviour change sweeps every prose site that NAMES the
      behaviour, located by grepping the changed symbol and the issue number, not by
      recalling which doc mentioned it. Measured this round: play/choice.go:113-114
      states "computes its region as line 0, column 0" and "#38 is PARKED" (it is line
      1 and the issue is working) — a comment written specifically to protect this
      invariant, so a stale one defeats its only purpose; and README.md:159 "Narrow
      the window and the links follow the text as it re-wraps" is false, since Resize
      only updates l.rows/l.cols and neither existing buffer lines nor existing
      regions move — only writes made after the resize are affected.
  - id: new
    severity: Minor
    family: duplicated-guard-and-message
    title: |
      three copies of the audio-off guard plus its identical message, where D11 justified the copies by claiming callers keep their own
    detail: |
      replraw.go:584 (playRegion), replraw.go:616 (replayInPlace) and repl.go:313
      (replayPiped) each spell `if !opt.playsAudio() { Fprintln(stderr,
      nothingToReplay) }`. D11 left the message with the callers on the grounds that
      "callers keep their own MESSAGES; only the condition is shared" — but all three
      messages are the same constant, so the premise does not hold for these three.
      This window added the third. ARCH-DRY.
  - id: new
    severity: Minor
    family: mechanism-adopted-without-its-obligations
    title: |
      the RenderOpts.Word obligation is fixed in code but no test fails without it
    detail: |
      Deleting `Word: key` from the Render call in todaysQuestions leaves the whole
      cmd/define suite green — I ran it with that mutation and the only failures were
      the git-dependent repo guards (the scratch copy has no .git). regionsIn falls
      back to e.Headword(), so the divergence the fix commit calls "a live defect — a
      click on a normalised deck word would have fetched the wrong recording" is
      unpinned. A row asserting Region.Word == the deck key when key and headword
      differ (jalapeno / jalapeño) would close it. Same family as BR-1 because it is
      the same obligation, one step further on: adopted, but not made falsifiable.
```

---

## Re-review — 2026-08-31T23:02:27-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 38 — the review loop's words are not clickable, because --play draws its own frames |
| repo | tools |
| issue file | workshop/issues/000038-play-clickable.md |
| boundary | whole-issue close |
| milestone | — |
| window | f56b34e5f54d9ba050246d3fde25e8036636f4ab..fd2972cbd07f5f84141709321f4de18a6975b3e0 |
| command | sdlc close --issue 38 |
| reviewer | claude |
| timestamp | 2026-08-31T23:02:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The two Criticals from earlier rounds are genuinely closed and now mechanically enforced: I reverted the one-ruler fix in a scratch worktree (`wrapMovedRegions(text, rs, l.cols)` → `, 80`) and both `TestAResizeDoesNotMisplaceTheClickMap` and `TestWriteRegionsMovesTheMapByTheScreensOwnWidth` went red; I deleted `Word: key` and `TestARegionAnswersForTheDeckKeyNotTheHeadword` went red; I renamed a Core-concepts cell back to `playRegions` and `TestPlanTablesNameEntitiesThatExist` failed with the exact message the family needed. The full suite is green (`go test ./...`), conformance is green apart from `TestReflectAgainstTheLiveService`, which is a live-LLM row unrelated to this window, and every `TestPTY*` row SKIPs here ("no pty available") as the estimate predicted. What blocks SHIP is that the same round that wrote *"a claim is discharged by something that can fail"* into `lessons.md` left the issue's own D8 — the click must never answer — undischarged: I disabled the entire `KeyClick` branch **and** routed `KeyClick` into `play.Apply` as an `InputReveal`, which is verbatim the Done-when row's `red when` cell, and `TestPlayClickIsNotAnAnswer` stayed green. Alongside that, the four-slot tracker enumeration BR-12 asked for was written down one slot short and three slots are still wrong at HEAD.

## 1. Strengths

- **`liveScreen.WriteRegions` now owns both rulers** (`cmd/define/screen.go:655`). Moving `wrapMovedRegions` inside the screen makes the second width *unexpressible* rather than merely unused — the caller has no width to pass. Mutation-confirmed above; this is the right shape and matches `#41` BR-20's resolution.
- **`wrapMovedRegions` asks `wrapWritten` itself how many rows a line becomes** (`cmd/define/playbar.go:186`) instead of re-deriving the rule. The erase-gesture exemption and the sub-20 policy therefore cannot differ between the wrap and the map — exactly the failure mode a second implementation would have. ARCH-DRY pass.
- **`marksIn` LOCATES the render inside the reveal** (`cmd/define/play_loop.go:581`) rather than counting from a formula. I checked this against `Choice.Reveal()` (`play/choice.go:156`), which emits a pre-wrapped option gloss that can itself be multi-line — a formula would have been wrong there, and `strings.Count` is exactly right.
- **`playRegion`'s `default: return`** (`cmd/define/replraw.go:596`) refuses an unknown kind instead of falling back to the headword, and `TestEveryRegionKindIsActionableThroughTheSharedRegistry` derives its loop from `numRegionKinds`. Deleting the `RegionOriginLang` case reddens both the editor's row and the shared one.
- **`playAnnounced` applies the audio-off predicate itself** (`cmd/define/main.go:850`). Disabling that guard makes `TestPlayAnnouncedFetchesNothingWithAudioOff` fetch from the CDN in all three subcases — the row genuinely covers what no caller-level test could reach.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `TestPlayClickIsNotAnAnswer` cannot fail; Done-when row 2 and T7 ship unpinned.** `cmd/define/play_loop.go:266`, plan Done-when row 2.

This is the **3rd finding in family `mechanism-adopted-without-its-obligations`.** BR-1 and BR-15 fixed instances (T5's three obligations; the `Word: key` pin). Do NOT fix this instance alone — the rule is that **every Done-when row's `red when:` cell is executed at the boundary, as a sweep of the whole table, and a row whose named test survives its own stated mutation is not a pin.** I ran that sweep on the 9 in-process rows this window owns; 8 discriminate and 1 does not:

| row | mutation applied | result |
|---|---|---|
| 1 | prompt write given `nil` regions | red |
| 1 | `Line: 1` → `Line: 0` | red (3 tests) |
| **2** | **`KeyClick` branch disabled** | **green** |
| **2** | **`KeyClick` routed into `play.Apply` as `InputReveal` — the literal `red when`** | **green** |
| 3 | reveal written with `nil` regions | red |
| 3 | `off := 0` | red |
| 4 | `case RegionOriginLang` removed | red |
| 4b | `playAnnounced` guard disabled | red |
| 3a/3b | `l.cols` → `80` | red |
| 3c | `Word: key` deleted | red |

Row 2 is green because `toInput` (`play_loop.go:407`) already returns `false` for `KeyClick` — so the property is delivered by an unrelated default, and the `continue` that T7 exists to add is dead protection. It only reddens if a click is made to *grade* (I confirmed `InputRune 'y'` fails it), which is a strictly narrower claim than the row makes. Fix: pin the branch itself — assert the click PLAYED and did not advance in the same test (`s.Index` unchanged, `player.count() > 0`), and drive it against a real `newPinnedScreen` rather than `paintInto`'s hand-`offer`ed region so the guard is on the path. Then re-run the table sweep and record it, as `#41`'s 4c row already does with "mutation-verified".

**I-2 — the completion-state enumeration is one slot short, and 3 of 6 are wrong at HEAD.**

This is the **3rd finding in family `tracker-state-stale`.** BR-10 fixed slot 1, BR-12 fixed slot 3 of a four-slot list. Do NOT fix these instances — the four-slot list was itself the defect. Measured at HEAD:

| slot | state | enforced by |
|---|---|---|
| issue frontmatter `status:` | ✓ `working` | `sdlc close` |
| issue `## Plan` boxes | ✓ all `[x]` | `sdlc close` plan-unchecked |
| plan `## Tasks` boxes | ✓ all `[x]` | (BR-12) |
| **issue `## Done when` boxes** | **✗ all 5 unticked** | **nothing** |
| **issue `## Log` boundary entry** | **✗ absent** (last entry: "resumed; `#41` landed three of the nine tasks"; three review rounds unlogged, contra AGENTS.md §3) | **nothing** |
| **referencing project row** | **✗ `workshop/projects/define-learn.md` has no `[tools#38]` row at all** | **nothing** |

The family recurs because the three enforced slots are exactly the three that keep coming back green. `sdlc close`'s plan-unchecked gate reads `## Plan` only (`ariadne/cmd/sdlc/close.go:569`), so nothing can refuse a close whose Done-when says nothing is done — and the repo convention is that they ARE ticked (`workshop/history/issues/000041-play-tui.md`, `000039-repetition-ladder.md` both close fully ticked). The durable fix belongs where the other two artifact guards already live: a `repo_guard_test.go` row asserting that an active issue with a fully-ticked `## Plan` has a fully-ticked `## Done when`. That converts slot 4 into slot 2's enforcement class and is the same move `TestPlanTablesNameEntitiesThatExist` made for the symbol family.

Also inside slot 4: Done-when row 3 still reads *"The click reaches `replayInPlace`"*, which plan PQ-4/D4 superseded — the click reaches `playRegion` → `playAnnounced`, and `replayInPlace` is no longer on the click path at all. That row is false as current truth, not merely unticked.

**I-3 — the doc-comment attachment defect is a class of 5, mechanically enumerable.** `cmd/define/replraw.go:554-563`.

This is the **2nd finding in family `doc-comment-attachment`** (BR-6). Do NOT fix the one instance — I wrote a ~50-line `go/ast` walk and it finds every sibling in one pass:

- `replraw.go:579` — `playRegion` carries `replayInPlace`'s doc (no blank line at :562/:563); godoc reads "replayInPlace speaks the current word again…" as `playRegion`'s. **In-window, introduced here.**
- `replraw.go:607` — `replayInPlace` now carries an unnamed "Plain \n, not \r\n" note. **In-window.**
- `screen.go:149` — `addRegions`'s doc opens "regionsAt records…"; stale name.
- `play/recall.go:38` — `Keys()`'s doc opens "Grade reads the self-rating…"; the prose belongs to `Grade`.
- `playbar.go:143-152` — `isOptionLine`'s doc block runs into `minWrapWidth`'s with no blank line, so `minWrapWidth` carries `isOptionLine`'s prose and `isOptionLine` (:154) is undocumented.

`go vet` does not catch this and the exported-comment linters do not reach unexported funcs. A `FuncDecl`-only guard (first word of the doc == the declared name, with an allowlist for the deliberate build-tag continuation at `dict_stub.go:27`) is ~40 lines in `repo_guard_test.go` and closes the family permanently.

## 4. Minor findings

- **BR-2 not addressed** — 6 of the 12 rows in "What this plan asserts about the existing tree" still cite line anchors that miss: `play_loop.go:262-265` now lands in the click comment, `:174-198` in the prompt comment, `:178-181` on the region literal, `replraw.go:264` on `voc := vocabularyFor(...)`, `:170` mid-comment, `:366-380` on the exit comment. Two rows are also false in substance (entries' regions are no longer discarded; the viewport keys are shared via `viewportGesture`).
- **BR-6 not addressed** — see I-3.
- **BR-7 not addressed** — `play_loop.go:448` initialises `held.marks`, `:466` builds a second local map, `:508` assigns over the first.
- **BR-8 not addressed** — `play_loop.go:274` passes `entry: ""`, so `utteranceFor` gets `ParseEntry("")` and an ORIGIN click plays the English spelling in the foreign voice. The raw text is in hand at `:468` and discarded. README.md:157 now claims the reveal is clickable *"exactly as in the interactive session"*, which makes this a documented-behaviour divergence rather than a silent one.
- **BR-13 not addressed** — `play/choice.go:113-114` still says "computes its region as line 0, column 0" (it is line 1) and "#38 is PARKED" (the issue is `working`); `README.md:159` "Narrow the window and the links follow the text as it re-wraps" is false — `Resize` (`screen.go:694`) only sets `l.rows`/`l.cols`, so neither existing buffer lines nor existing regions move, and it contradicts the accurate sentence two paragraphs above it ("wraps what comes after it to the new width").
- **BR-14 not addressed** — `replraw.go:584`, `replraw.go:616`, `repl.go:313` still each spell `if !opt.playsAudio() { Fprintln(stderr, nothingToReplay) }` with the same constant.
- `play_loop.go:270-273` calls `defaultIndicator(opt)` "the record-shaped indicator, not the editor's erasable one" — with `opt.tty` true (which `--play` requires) it returns `erase: eraseLine`, i.e. it *is* erasable; only `before: "\n"` differs.

## 5. Test coverage notes

- The mutation sweep in I-1 is the main result: 8 of 9 in-process Done-when rows discriminate, row 2 does not.
- `marksIn`'s `at < 0` branch (render not present in the reveal — "a form that reworded its reveal") has no test. Cheap to add as a pure row, and it is the branch that silently returns no regions.
- `TestPlayClickOnThePromptWordPlaysIt` asserts only `player.count() > 0`, not *which* word played; `TestTheAskedWordIsClickableOnAMultipleChoiceQuestion` covers the identity separately, so the pair is adequate.
- The `playRig` fix (`width: 0` → `defaultCols`) is the right generalisation of `#41` BR-24 and `lessons.md` carries the rule rather than the instance.
- Environmental, not a finding: every `TestPTY*` SKIPs here, so Done-when 5, 6b, 7 and 8 rest entirely on the manual terminal pass the plan's "Verification before close" specifies. No `## Log` entry records that pass yet (see I-2, slot 5) — `sdlc close --verified` will need it in the evidence string.

## 6. Architectural notes

- **ARCH-DRY — flag.** One registry lifted (`playRegion`), one predicate lifted (`playsAudio`), one wrap owner — all good. Flagged: BR-14's three identical guard-and-message copies (D11 justified leaving messages with callers on the grounds that they differ; all three are the same constant, so the premise does not hold for these three), and BR-7's duplicate `marks` map.
- **ARCH-PURE — pass.** `clickable`, `marksIn` and `wrapMovedRegions` are pure and `TestAWrapMovesTheClickMapRatherThanDroppingIt` drives the last with no screen at all. `playRegion` is correctly labelled INTEGRATION and injected into both loops with the indicator as its only parameter. No business logic moved into IO.
- **ARCH-PURPOSE — flag.** Shadow-sweep of the click registry: the editor consumer derives (`replraw.go:329`), the sitting consumer derives (`play_loop.go:274`), and `numRegionKinds` drives the test loop so a third kind is exercised on declaration. The one consumer that does *not* fully derive is the sitting's ORIGIN path (BR-8): it drops the entry text the editor supplies, so the same registry produces a degraded utterance in one loop — and the README asserts the two are identical.
- **ARCH-MOCK — pass.** `playAnnounced` reaches the CDN and player through the existing stateful fakes (`rig.cdn.Requested()`, `fakePlayer.count()`); `playSession` is drivable with a scripted key channel and a real `newPinnedScreen`, so production flow and test flow share the boundary. Live conformance exists for the pty surface, environmentally skipped here.
- **ARCH-CONSTRAINTS — pass.** `wrapMovedRegions` runs once per `WriteRegions`, i.e. per question and per reveal, not per keystroke; `held.marks` retains one already-allocated rendered string per question, bounded by `-count`. The text is wrapped twice per `WriteRegions` (once to measure, once to write) — bounded and off the keystroke path, and the redundancy buys the single-owner guarantee, so it is the right trade. The plan's "no new budget claimed" holds.

## 7. Plan revision recommendations

- **A `## Revisions` entry for the Done-when sweep.** Record that every row's `red when:` was executed as a mutation at the boundary, with the result per row, and correct row 2's cell — either to what the test actually discriminates ("the click GRADES") or, better, strengthen the test and keep the cell. `#41`'s rows 4c and 6 already carry "mutation-verified"; this plan should carry it for all of its own.
- **A `## Revisions` entry retiring the tree-claims table's line anchors** in favour of symbol/test names, per BR-2's rule, and correcting the two rows now false in substance (regions kept, not discarded; `viewportGesture` shared).
- **The issue** (not the plan) needs: Done-when row 3 rewritten to `playRegion`/`playAnnounced` per D4, all five boxes ticked, and a `## Log` entry for the three boundary rounds.
- **`workshop/projects/define-learn.md`** needs a `[tools#38]` row beside the `[tools#41]` one, or an explicit note that this thread does not track it.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      T4 now specifies line 1, and T5 carries all three HEAD-documented obligations as rows.
  - id: BR-2
    disposition: not-addressed
    note: |
      6 of 12 tree-claim rows still cite wrong lines; two rows are false in substance.
  - id: BR-6
    disposition: not-addressed
    note: |
      replraw.go:554-563 unchanged; see the new finding for the enumerated class of 5.
  - id: BR-7
    disposition: not-addressed
    note: |
      play_loop.go:448, :466, :508 still build and overwrite two maps.
  - id: BR-8
    disposition: not-addressed
    note: |
      play_loop.go:274 still passes entry ""; README.md:157 now claims exactness with the editor.
  - id: BR-11
    disposition: addressed
    note: |
      Table names clickable / sittingDeck.marksIn / wrapMovedRegions; guard PASSES and I confirmed it reddens on a reverted cell.
  - id: BR-12
    disposition: not-addressed
    note: |
      Plan Tasks fixed, but the enumeration was one slot short: Done-when, Log and the project row are all still wrong at HEAD.
  - id: BR-13
    disposition: not-addressed
    note: |
      choice.go:113-114 and README.md:159 both unchanged and both false at HEAD.
  - id: BR-14
    disposition: not-addressed
    note: |
      replraw.go:584, replraw.go:616 and repl.go:313 still each spell the guard and the same constant.
  - id: BR-15
    disposition: addressed
    note: |
      Verified by reverting Word: key in a scratch worktree — TestARegionAnswersForTheDeckKeyNotTheHeadword goes red on all three regions.
findings:
  - id: new
    severity: Important
    family: mechanism-adopted-without-its-obligations
    title: |
      Done-when row 2 survives its own stated mutation, so T7's click guard ships with no failing test
    detail: |
      3rd in this family, so the deliverable is the rule, not the instance: every Done-when
      row's `red when:` cell is EXECUTED as a mutation at the boundary, swept as one table.
      I ran that sweep on the 9 in-process rows; 8 redden and row 2 does not. Disabling the
      whole `KeyClick` branch (play_loop.go:266) leaves TestPlayClickIsNotAnAnswer green, and
      so does the literal `red when` — I routed KeyClick into play.Apply as an InputReveal and
      it still passed. The property is delivered by toInput's default (play_loop.go:421
      returns false for KeyClick), not by T7's guard, and the test only reddens if a click is
      made to GRADE. It also drives paintInto with a hand-`offer`ed region rather than a real
      pinned screen, so the guard is not on the path under test. Fix the rule: assert the
      click PLAYED and did not advance (s.Index unchanged, player.count() > 0) against a real
      newPinnedScreen, then record the per-row sweep result in the plan the way #41's rows 4c
      and 6 already do.
  - id: new
    severity: Important
    family: tracker-state-stale
    title: |
      the completion-state enumeration is one slot short, and 3 of 6 slots are wrong at HEAD
    detail: |
      3rd in this family, so the rule rather than the instances: an issue's completion state
      is SIX slots, and the four-slot list BR-12 wrote was itself the defect. Measured at
      HEAD: frontmatter status OK; issue `## Plan` OK; plan `## Tasks` OK; issue `## Done
      when` all five boxes UNTICKED; issue `## Log` carries no entry for any of the three
      boundary rounds (AGENTS.md 3 requires it); workshop/projects/define-learn.md has no
      `[tools#38]` row at all beside its `[tools#41]` one. The family recurs because exactly
      the three enforced slots are the three that stay green — `sdlc close`'s plan-unchecked
      gate reads `## Plan` only (ariadne/cmd/sdlc/close.go:569), so nothing can refuse a close
      whose Done-when says nothing is done, while the repo convention is that they ARE ticked
      (history/issues/000041 and 000039 both close fully ticked). Durable fix: a
      repo_guard_test.go row asserting that an active issue with a fully-ticked `## Plan` has
      a fully-ticked `## Done when`, which puts that slot in the same enforcement class as the
      two that stopped recurring. Separately, Done-when row 3 still reads "The click reaches
      `replayInPlace`", which D4/PQ-4 superseded — it is false as current truth, not merely
      unticked.
  - id: new
    severity: Important
    family: doc-comment-attachment
    title: |
      the mis-attached doc comment is a class of five, enumerable in one go/ast pass
    detail: |
      2nd in this family (BR-6), so the rule: a doc comment's first word must be the name of
      the declaration it sits on, and the class is checked mechanically rather than by
      re-reading the file. A ~50-line go/ast walk over cmd/define finds every sibling at once:
      replraw.go:579 (playRegion carries replayInPlace's doc — in-window), replraw.go:607
      (replayInPlace carries an unnamed line-ending note — in-window), screen.go:149
      (addRegions' doc opens "regionsAt records…"), play/recall.go:38 (Keys() carries Grade's
      prose), playbar.go:143-152 (isOptionLine's block runs into minWrapWidth's, leaving
      isOptionLine undocumented). go vet does not catch this and the exported-comment linters
      do not reach unexported funcs, so a FuncDecl-only guard in repo_guard_test.go — with an
      allowlist for the deliberate build-tag continuation at dict_stub.go:27 — is what closes
      the family.
```

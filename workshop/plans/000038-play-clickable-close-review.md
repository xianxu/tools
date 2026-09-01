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

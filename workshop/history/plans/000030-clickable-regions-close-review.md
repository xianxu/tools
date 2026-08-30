# Boundary Review — tools#30 (whole-issue close)

| field | value |
|-------|-------|
| issue | 30 — clickable regions in the terminal: click ORIGIN French to hear it, click the IPA to replay |
| repo | tools |
| issue file | workshop/issues/000030-clickable-regions.md |
| boundary | whole-issue close |
| milestone | — |
| window | 168b1c9f3ed7367f122af4795002ad336fd41e02..92e490db92cec4e58b47b4fc993d07f330644f2f |
| command | sdlc close --issue 30 |
| reviewer | claude |
| timestamp | 2026-08-30T15:31:21-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Fourteen rounds in, the code is in good shape: `go test ./...`, `go test -race ./cmd/define/` and `go vet -tags conformance` are all green at HEAD (verified, not reported), and I ran four fuzz targets at 40s each (`FuzzRenderDoesNotPanic`, `FuzzDecodeMouseIsBounded`, `FuzzRenderLosesNothing`, `FuzzScreenWriteDoesNotPanic`) — all clean, so the class that produced BR-51 is not hiding another. I verified both Criticals by reversion in a scratch copy: reverting `findVisible`'s empty-span guard reproduces the original panic at `render.go:462` inside `TestRenderSurvivesADegenerateEntry`, and replacing `word := key` with `word := e.Headword()` reddens `TestAClickAsksForExactlyWhatEnterAsksFor` and `TestTheClickableSpanIsTheWholeHeadword` with the recorded `hot dog`/`a priori`/`bargainer` messages. I also spot-checked the M2 mutation table (`writeRendered`) and it produced its predicted message verbatim, so that enumeration is honest rather than aspirational. What blocks a clean SHIP is one disposition: **BR-55 is not-addressed** — the `RenderOpts` half of the guard is fixed and now fires, but the sibling the same finding named in the same file (`TestAtlasDescribesEveryRegionKind`'s bare `k.String()`) is untouched, and I measured it: deleting the entire `## Clickable regions` section from the atlas leaves the `headword` kind green. That is the instance, not the class (ARCH-PURPOSE), and it is a one-line fix.

## 1. Strengths

- **Both Criticals are pinned by tests that genuinely fail without them** — verified by reversion, not by reading the commit message. `render.go:436`'s "a span with no visible extent is not a span" is measured in cells, and the NUL crasher is committed to `testdata/fuzz/FuzzRenderDoesNotPanic/`.
- **`visible()` returning the frame and its top line as one answer** (`screen.go:205`) is the right shape for BR-42, not a hoisted call: `Frame`, `LineAt` and `Paint` cannot answer from different states because there is only one state to answer from.
- **The mentions producer delivers the issue's founding insight.** I probed `piano` and `ballet` directly: both French *and* Italian come back as separate regions at correct columns (line 7, cols 16 and 58 for `piano`), and `read`'s Dutch cognate is correctly not offered. The README's `concrete`→French and `jalapeño`→Spanish claims both check out against the committed fixtures.
- **`escapeLen` (`render.go:527`) really is the one owner now** — `visibleCells`, `clipVisible`, `visibleIndex` and `markClickable` all skip through it, and `RenderLine`'s cursor park counts cells. BR-35 and BR-26 are both genuinely closed.
- **`TestPlanNamedTestsExist` (`repo_guard_test.go:749`) works** — I renamed row 1b's test in a scratch copy and it reddened with the intended message. Per-milestone scoping is the right granularity.

## 2. Critical findings

None.

## 3. Important findings

**BR-55 remains open — `cmd/define/doc_sync_test.go:233`.** `TestAtlasDescribesEveryRegionKind` still searches for the bare `k.String()`; `"headword"` occurs 19 times in `atlas/define.md` for unrelated reasons. Measured: with the whole `## Clickable regions` section deleted, `RenderOpts.Word`/`Color`/`Width` all fire (the round-13 fix works) and `"ORIGIN language"` fires, but `RegionHeadword` stays green. Fix: put the qualified Go identifiers (`RegionHeadword`, `RegionOriginLang`) in the atlas section and search for those, or use the `<!-- … -->` anchor convention this file already uses at `TestDocsQuoteThePronCommandHelp`.

**BR-29 remains open — the tree→table direction still has no guard.** Measured at HEAD, six top-level declarations added by this window sit in files the Core-concepts tables name and have no row: `newLiveScreen`, `liveScreen.throttledPaint`, `wheelLines` (screen.go/key.go), `digits` (key.go), `submitLine` (replraw.go), `headingLine` (render.go). `repo_guard_test.go` already has `changedLines()` and `repoRoot()` to build it with. (Filed separately as `#33`, per `workshop/issues/000033-plan-table-both-directions.md` — if the intent is to defer it there, say so explicitly at the close rather than leaving it implicit, the same disposition BR-56 got.)

## 4. Minor findings

- `isClickButton` (`key.go:256`) accepts extended buttons 8–11 as a left press: `ESC[<128;5;3M` decodes to `KeyClick` at row 2 col 4, and so does the X10 form. The comment promises "LEFT only".
- `markClickable` (`screen.go:306`) abandons every later span on a line when one span's `Col` is not a reachable cell boundary or overlaps its predecessor — `next` never advances. Measured on `"日本語 abc"`: a region at col 1 (inside a wide rune) silently drops the mark for the region at col 7.
- The issue's `## Done when` (`workshop/issues/000030-clickable-regions.md:202-214`) is entirely unticked at the boundary that closes it; every closed issue in `workshop/history/issues/` ticks them.
- Still open and re-measured unchanged: BR-1, BR-41, BR-44/BR-53 (all five stale comments, incl. `internal/llm/config.go:153` claiming a four-character key against `defaultLocalKey = "parley-local"`), BR-45, BR-48, BR-49, BR-57.

## 5. Test coverage notes

All 12 `TestPTY*` rows SKIP in this environment ("no pty available"), so Done-when M2 rows 6 and 8 are certified here only by their in-process counterparts — correctly recorded in the Log and filed as `#37`. The fuzz corpus is exercised by `go test ./...` at seed level only; my 160s of `-fuzz` found nothing new, which raises confidence but does not change #37's point. `TestAClickAtAPaintedCellPlaysWhatIsUnderIt` is the strongest test in the diff — it drives a real `liveScreen` as both view and stdout, decodes a real SGR report, and reads the underlined cell out of the painted frame rather than inventing a coordinate.

## 6. Architectural notes for upcoming work

- **ARCH-DRY — flag (existing BR-49).** Four traversals of styled text by display cell still each re-spell the loop. They share `escapeLen`/`cellWidth`, so the grammar is fine; it is the walk that is duplicated four ways.
- **ARCH-PURE — pass, with BR-41.** `screen` is unit-tested with no terminal and `liveScreen` is the only IO; `display` is injected. The `PURE` label on `LineAt`/`Frame` is inaccurate (both write `s.offset` through `clamp`), which is a labelling defect, not a layering one.
- **ARCH-PURPOSE — flag (BR-55).** The one place this round under-delivers: a finding that named two sites got one fixed.
- **ARCH-MOCK — pass, with #37.** The terminal's seam is real (`rawSession.control` is an `io.Writer`, `display` is an interface, production and test share the boundary). The live conformance check exists but runs nowhere automatic — filed.
- **ARCH-CONSTRAINTS — pass, with a note.** The declared envelope (16 ms `paintInterval` with trailing flush; deliberately uncapped `screen.lines`) names only the line buffer. M2 added a second unbounded per-session structure, `screen.regions`, which the envelope statement does not mention. It is small (~2–3 regions per lookup) and I am not raising it, but the next edit to that paragraph should name it.

## 7. Plan revision recommendations

- **`## Revisions` — "M2 close: BR-55 was the instance, not the class."** Record that `TestAtlasDescribesEveryRenderOpt` was fixed and verified by mutation while `TestAtlasDescribesEveryRegionKind` was left with the same defect the same finding named, and state the rule the pair implies: a derived docs guard anchors on a token that exists ONLY in the documentation it defends.
- **M1 Core-concepts tables** — add rows (or an explicit deferral to `#33`) for `newLiveScreen`, `liveScreen.throttledPaint`, `wheelLines`, `digits`, `submitLine`, `headingLine`.
- **The ARCH-CONSTRAINTS block in "M1 boundary review → 5"** — extend the Buffer bullet to cover `screen.regions`, since the envelope now has two unbounded members and names one.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan line 132 still enumerates the four cases verbatim; screen_test.go has only FuzzScreenWriteDoesNotPanic, which asserts no panic, not chunk-independence.
  - id: BR-26
    disposition: addressed
    note: |
      All six enumerated sites measure cells at HEAD — editor.go:232 visibleCells, command.go:308 truncate delegates to clipVisible — and the combining-mark case is pinned at render_test.go:482.
  - id: BR-29
    disposition: not-addressed
    note: |
      No tree-to-table guard exists; measured six new decls with no row — newLiveScreen, liveScreen.throttledPaint, wheelLines, digits, submitLine, headingLine.
  - id: BR-34
    disposition: addressed
    note: |
      Verified by mutation — renaming TestPaintFitsTheTerminalAndParksTheCursor in the plan reddens TestPlanNamedTestsExist with the intended message.
  - id: BR-35
    disposition: addressed
    note: |
      escapeLen (render.go:527) wraps scanEscape and visibleCells, clipVisible, visibleIndex and markClickable all skip through it.
  - id: BR-36
    disposition: addressed
    note: |
      screen_test.go:465 asserts the cursor row exactly, derived by replaying the prompt through readFrame rather than restating Paint's formula.
  - id: BR-41
    disposition: not-addressed
    note: |
      LineAt (screen.go:174) reaches clamp through visible(), which writes s.offset; screen.go:186 and the plan's M2 row both still say PURE.
  - id: BR-44
    disposition: not-addressed
    note: |
      key.go:186, :198 and :344 unchanged and decodeWheel is still the name; subsumed by BR-53.
  - id: BR-45
    disposition: not-addressed
    note: |
      screen.go:143 still decrements base without shifting Col, and screen_test.go:739 still supplies Col 12 pre-offset by hand.
  - id: BR-48
    disposition: not-addressed
    note: |
      The "the command menu closing" subtest at screen_test.go:906 is unchanged.
  - id: BR-49
    disposition: not-addressed
    note: |
      No forEachCell exists; visibleCells (render.go:510), visibleIndex (render.go:471), clipVisible (screen.go:667) and markClickable (screen.go:282) each still spell the traversal.
  - id: BR-53
    disposition: not-addressed
    note: |
      All five sites unchanged; internal/llm/config.go:153 still claims a four-character key against defaultLocalKey = "parley-local", twelve characters.
  - id: BR-55
    disposition: not-addressed
    note: |
      The RenderOpts half is fixed and fires; the sibling the same finding named, doc_sync_test.go:233, still searches the bare k.String() and stays green with the whole atlas section deleted.
  - id: BR-56
    disposition: addressed
    note: |
      Filed as tools#37 and stated explicitly in the close commit and the issue Log; re-measured, all 12 pty rows still skip here.
  - id: BR-57
    disposition: not-addressed
    note: |
      atlas/define.md:401 still says an empty RenderOpts.Word means no click map; render.go:316 still falls back to e.Headword() and returns regions.
findings:
  - id: new
    severity: Minor
    family: predicate-narrower-than-its-encoding
    title: |
      isClickButton accepts extended mouse buttons 8-11 as a left press, against its own "LEFT only" comment
    detail: |
      key.go:256 rejects only bits 64 (wheel) and 32 (motion) before testing
      b&3 == 0, so button 8 (b=128) satisfies both. Measured: decodeKey("\x1b[<128;5;3M")
      returns KeyClick at row 2 col 4, identical to a left press, and the X10
      form {0x1b,'[','M',160,33,33} does the same. A five-button mouse's back
      button over an underlined headword therefore plays it. The rule: a
      predicate over an external wire encoding is written against the
      encoding's whole defined range, not the values the fixtures happen to
      carry — the same shape as M2.6's mode rule, one level down from the mode
      to the button field.
  - id: new
    severity: Minor
    family: helper-precondition-unguarded
    title: |
      markClickable abandons every later span on a line when one span's Col is unreachable or overlapping
    detail: |
      This is the 2nd finding in family helper-precondition-unguarded (BR-51 is
      its sibling). Do NOT just guard the one call — state the rule: a helper
      consuming a region list either enforces its preconditions at the owner
      (regionsIn) or degrades PER REGION, never by abandoning the rest of the
      line. screen.go:306 advances `next` only on an exact `col == spans[next].Col`
      match, so a Col that no cell boundary can equal parks the cursor forever.
      Measured on "日本語 abc": regions at {Col 1, W 2} and {Col 7, W 3} produce
      NO marks at all, and the overlapping triple {0,6},{2,2},{7,3} marks only
      the first. Unreachable from regionsIn today — findVisible returns real
      boundaries and the corpus was probed for overlap at M2 close — which is
      exactly why it is worth stating now: the registry's whole premise is that
      "a third consumer is a row", and a third kind whose span overlaps the
      headword would silently unmark the rest of the line with a green suite.
  - id: new
    severity: Minor
    family: unsettled-claim-at-boundary
    title: |
      The issue's Done-when checklist is entirely unticked at the boundary that closes it
    detail: |
      workshop/issues/000030-clickable-regions.md:202-214 — all seven acceptance
      rows are still "- [ ]" while the ## Plan rows above them are all ticked
      and the close commit is next. Every closed issue in workshop/history/issues/
      ticks them (checked #29 and #35). The rows are substantively delivered —
      I verified the ORIGIN click, the multi-language case on piano/ballet, the
      underline mark, and the Option/scrollback documentation in README — so
      this is a bookkeeping gap, not a delivery one. The rule: the boundary that
      closes a claim settles it in writing, either ticked or explicitly recorded
      as not delivered with the reason.
```

---

## Re-review — 2026-08-30T15:48:07-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 30 — clickable regions in the terminal: click ORIGIN French to hear it, click the IPA to replay |
| repo | tools |
| issue file | workshop/issues/000030-clickable-regions.md |
| boundary | whole-issue close |
| milestone | — |
| window | 168b1c9f3ed7367f122af4795002ad336fd41e02..92e490db92cec4e58b47b4fc993d07f330644f2f |
| command | sdlc close --issue 30 |
| reviewer | claude |
| timestamp | 2026-08-30T15:48:07-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 15 disposes the two blocking Importants: `TestAtlasDescribesEveryRegionKind` now searches `RegionKind.identifier()` — mutation-verified, deleting the atlas's two region bullets reddens it with both names, where the old `k.String()` form stayed green with the whole `## Clickable regions` section gone — and `isClickButton` now rejects bit 7, mutation-verified (reverting the guard reddens two of three subtests, and the accepted `Cb` set over 0..255 is now exactly `{0,4,8,12,16,20,24,28}`, left button with any modifier combination). BR-29 is disposed by explicit deferral to `#33`, which exists, predates this issue, and owns the tree→table direction as its whole subject — a legitimate deferral, not the deferred point of `#30`. `go build`, `go vet`, `go test ./...` and `go test -race` on the new screen surface are all green in the working tree. Two things keep this from SHIP, neither blocking: **the three fixes are uncommitted**, so they sit outside the pinned window `168b1c9..92e490d` and a reader checking out the named head would find both defects intact; and the PURE relabel swept the one site BR-41 named while six enumerable siblings in the same table and the same file still carry the label — the instance, not the class. Nine Minors remain open (BR-1, 44, 45, 48, 49, 53, 57, 59, 60), none blocking.

## 1. Strengths

- **The registry genuinely single-sources.** `numRegionKinds` (render.go:253) feeds `String`, the new `identifier`, `TestEveryRegionKindIsActionable` (editorloop_test.go:868) and `TestAtlasDescribesEveryRegionKind`, and all four fall through to `RegionKind(%d)` for an unhandled kind — so a third kind reddens the actionable guard, the naming guard and the atlas guard independently. Shadow-swept: no hand-maintained restatement of the kind set survives in the tree.
- **`TestAClickAtAPaintedCellPlaysWhatIsUnderIt`** (editorloop_test.go:901) drives real bytes → `decodeKey` → `Key.Row` → `LineAt` → region → replay with a real `liveScreen` as both view and stdout, and reads the underlined span out of the painted frame rather than inventing a coordinate. That is the joint the first eight rounds kept missing.
- **`writeRendered`/`regionWriter`** (main.go:797) — the seam fills itself by type assertion, so a pipe gets exactly today's bytes and the screen gets the map, from one call site. `TestRenderOutputMatchesTheCorpusGolden` against a pre-signature-change golden makes "the bytes are identical" a property rather than a promise.
- **`Transcript` joins raw buffer lines** (screen.go:715), so the underline never reaches scrollback — the affordance is live-only, which is the honest answer and matches what README now says.
- **`isClickButton`'s new comment states the modifier bits it deliberately ignores**, so the next reader can tell the blacklist from an oversight.

## 2. Critical findings

None.

## 3. Important findings

None open. BR-55 and BR-29 both disposed above.

## 4. Minor findings

- **`Frame`, `Paint`, `Scroll`, `Page` and `Write` still say PURE while every one writes the receiver** — screen.go:192 and plan lines 80–85, 104. Measured: with a 5-line buffer scrolled back 2 and `rows` grown to 10, `Frame()` moves `offset` 2 → 0. `Frame`'s "PURE" sits two lines above `LineAt`'s new "NOT pure" comment describing the identical `visible()` call.
- **The round's fixes are uncommitted**, so BR-55's and BR-58's "addressed" rest on a tree the pinned window does not contain.
- Carried, unchanged and verified still open at the working tree: BR-1 (plan:132), BR-44/BR-53 (key.go:186, :198, :344; render.go:304; internal/llm/config.go:153), BR-45 (screen.go:143), BR-48 (screen_test.go:906), BR-49 (render.go:488/527, screen.go:288/673), BR-57 (atlas/define.md:407), BR-59 (screen.go:313), BR-60 (issue:202-213).

## 5. Test coverage notes

Two mutation checks run this round, both red as intended. `TestExtendedMouseButtonsAreNotClicks`'s "button 9 (forward)" subtest does *not* redden under the mutation (Cb 129 has a low bit set, so the old mask rejected it anyway) — that is a legitimate regression guard for adjacent behavior, not a vacuous pin, and I am not raising it. Outside the byte range, `Cb` 256/512/1024 still decode to `KeyClick`: no terminal emits those, but the blacklist form defaults an undefined value to "click" where `b&^28 == 0` would default to "not a click". One line, folded into BR-58's note rather than raised.

## 6. Architectural notes

- **ARCH-DRY — flag (BR-49, open).** `escapeLen`, `clickAt`, `historicalStages` and `replayInPlace` are each the one owner of their fact; the four cell-walks are the exception and are already recorded.
- **ARCH-PURE — flag (new finding).** The `screen`/`liveScreen` split is right and the arithmetic is unit-tested with no pty; the label vocabulary is what has come apart.
- **ARCH-PURPOSE — pass with residue.** All seven Done-when rows are substantively delivered; BR-57 (a stated guarantee nothing enforces) and BR-60 (unticked) are the paperwork.
- **ARCH-MOCK — pass.** The pty harness is the terminal's stateful double and `readFrame` interprets frames as a terminal does; the 12 rows skipping without a pty is filed as `#37`.
- **ARCH-CONSTRAINTS — pass.** The throttle bounds staleness at one interval with an unconditional trailing flush and unconditional paints on every burst-ending gesture; the frame is budgeted in display rows with a stated order of sacrifice; the uncapped buffer is a recorded decision.

## 7. Plan revision recommendations

One entry: the Core-concepts status column needs a single definition. If PURE means "no IO, unit-testable without a terminal" (ARCH-PURE's sense), restore `LineAt`/`visible` to PURE and say in the legend that these methods re-establish the viewport clamp; if it means "does not mutate the receiver", then rows 80–85 and the prose at line 104 must change with it, along with screen.go:192. Leaving one row relabelled gives the column two meanings.

```findings
dispose:
  - id: BR-55
    disposition: addressed
    note: |
      Mutation-verified: deleting the atlas's RegionHeadword/RegionOriginLang bullets reddens TestAtlasDescribesEveryRegionKind for both, where the old k.String() form stayed green with the whole section deleted.
  - id: BR-58
    disposition: addressed
    note: |
      Mutation-verified: reverting the b&128 guard reddens two of three subtests; accepted Cb over 0..255 is now exactly {0,4,8,12,16,20,24,28}. Residue, not re-raised — Cb 256/512/1024 still decode to KeyClick because the guard is a blacklist; b&^28 == 0 would close it in one line.
  - id: BR-41
    disposition: addressed
    note: |
      LineAt's comment and the plan's LineAt/visible row now say NOT pure with the BR-42 reason. The class it belongs to is raised separately this round.
  - id: BR-29
    disposition: addressed
    note: |
      Deferred to tools#33 explicitly, in the plan's Revisions, with the class named and six declarations recorded as evidence; #33 exists, predates this issue, and owns the tree-to-table direction as its whole subject. Seventh piece of evidence for it — RegionKind.identifier, added this round, is named in the Revisions prose and has no Core-concepts row.
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan line 132 still enumerates the four cases verbatim; no chunk-boundary property test for Write exists.
  - id: BR-44
    disposition: not-addressed
    note: |
      key.go:186, :198 and :344 unchanged, decodeWheel still the name; subsumed by BR-53.
  - id: BR-45
    disposition: not-addressed
    note: |
      screen.go:143 still decrements base without shifting Col; screen_test.go still supplies Col pre-offset by hand.
  - id: BR-48
    disposition: not-addressed
    note: |
      screen_test.go:906 unchanged.
  - id: BR-49
    disposition: not-addressed
    note: |
      No forEachCell; render.go:488, render.go:527, screen.go:288 and screen.go:673 each still spell the traversal.
  - id: BR-53
    disposition: not-addressed
    note: |
      All five sites verified unchanged, including internal/llm/config.go:153 claiming a four-character key against defaultLocalKey "parley-local".
  - id: BR-57
    disposition: not-addressed
    note: |
      atlas/define.md:407 still says empty RenderOpts.Word means no click map; render.go:333 still falls back to e.Headword() and returns regions.
  - id: BR-59
    disposition: not-addressed
    note: |
      screen.go:313 still advances next only on an exact col == spans[next].Col match.
  - id: BR-60
    disposition: not-addressed
    note: |
      workshop/issues/000030-clickable-regions.md:202-213 all still "- [ ]" while every Plan row is ticked.
findings:
  - id: new
    severity: Minor
    family: pure-label-hides-mutation
    title: |
      The PURE relabel swept the one site BR-41 named; six siblings in the same table and the same file still claim PURE while writing the receiver
    detail: |
      This is the 2nd finding in family pure-label-hides-mutation. Do NOT relabel
      the six sites one at a time — state the rule. Measured prevalence: seven
      sites, one fixed. screen.go:192 says "Frame is the rows to paint, oldest
      first. PURE" two lines above LineAt's new "NOT pure" comment, and both
      reach the same visible() call; measured, Frame moved s.offset from 2 to 0
      on a 5-line buffer scrolled back 2 with rows grown to 10. The plan's table
      likewise still says PURE for screen.Write (81), screen.Frame (82),
      screen.Scroll/clamp (83), screen.Page (84) and screen.Paint (85, "PURE,
      given the writer" — it writes s.cols, s.rows and s.offset), plus the prose
      at line 104. The rule: the status column carries ONE definition of PURE.
      Under ARCH-PURE's sense (no IO, unit-testable with no terminal) all seven
      are PURE and LineAt/visible should go back; under "does not mutate the
      receiver" none of them are. Relabelling one row of seven leaves a reader
      with two meanings of the same word in one table, which is worse than the
      label BR-41 objected to.
  - id: new
    severity: Minor
    family: verification-claim-unreproduced
    title: |
      This round's three fixes are uncommitted, so the gate's evidence is not reproducible from the commit the window names
    detail: |
      This is the 2nd finding in family verification-claim-unreproduced (BR-23 is
      its sibling — a suite recorded green that was red at HEAD). Do NOT just
      commit these three. The rule: a gate round's evidence must be reproducible
      from the commit the round names. The window is 168b1c9..92e490d, and the
      fixes for BR-55, BR-58 and BR-41 exist only in the working tree — at
      92e490d, TestAtlasDescribesEveryRegionKind still searches k.String() and
      isClickButton still accepts Cb 128. I verified the fixes against the
      working tree by mutation and they hold, so the ledger's "addressed" is
      substantively true and procedurally unreproducible. Satisfy it by
      committing before re-running the gate, or by having the round record the
      tree it measured.
```

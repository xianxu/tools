# Boundary Review — tools#30 (milestone M2)

| field | value |
|-------|-------|
| issue | 30 — clickable regions in the terminal: click ORIGIN French to hear it, click the IPA to replay |
| repo | tools |
| issue file | workshop/issues/000030-clickable-regions.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | a6584e243f2a032d209f8f4f1f44c50ed801f0ad..91e4fcebbad518289cf3e325f1912fb3310ade40 |
| command | sdlc milestone-close --issue 30 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-08-30T11:42:27-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M2 delivers what it claims: `Render` emits a region map without moving a byte (I verified this independently rather than trusting the golden — regenerating the corpus from the base commit in a scratch worktree produces a file byte-identical to the committed 315,434-byte golden), the hit test maps viewport rows to buffer lines correctly through a single `topLine` owner, both click actions route through `replayInPlace` so they cannot drift from `/pron`, and the underline is spliced by the screen and provably never reaches a pipe. `go test ./...`, `-race` and `go vet` are green here; a 45s / 2.24M-exec fuzz run of `FuzzDecodeMouseIsBounded` found nothing. Nothing blocks on correctness. What holds it back from SHIP is three Importants, all recurrences of families this issue has already paid for: the one object that joins Render's regions to the click (`liveScreen.WriteRegions`/`RegionAtRow`) is pinned by nothing — I confirmed by mutation that swapping `addRegions` and `Write` leaves the entire suite green; the "one registry" guard restates the enum's extent instead of deriving it, so Done-when 7's stated failure mode cannot actually fire; and neither `README.md` nor `atlas/define.md` says a word about the feature the issue is named for.

## 1. Strengths

- **`cmd/define/render_test.go:509` — byte-identity is real evidence, not a self-comparison.** I rebuilt the corpus dump from `a6584e2` in a throwaway worktree: `cmp` says identical. The claim "generated from the commit BEFORE the change" survives an adversarial check, and the shape that makes it cheap (`regionsIn` reads the finished string, `Render`'s body untouched) is the right one.
- **`cmd/define/screen.go:212` — `topLine` is a falsifiable single owner.** Mutating it by `+1` reddens `TestScreenResolvesAClickToWhatWasRenderedThere` (3 subtests), `TestClicksFollowTheTextWhenScrolled` and `TestScreenMarksClickableSpans` — the click path and the paint path both, which is exactly the disagreement the one-owner argument predicts.
- **`cmd/define/origin.go:167` — the mentions producer, with `OriginLanguage` reduced to `mentions[0]`.** One cut-and-mask, two consumers, and `maskOut` preserves length so offsets survive. `TestOriginMentionOffsetsIndexTheSourceText` corrected the plan's own exemplar in the process (`concrete` does not shift; `ballet` shifts 9 columns) — measurement beating an inherited claim.
- **`cmd/define/key_test.go:389 — `TestEveryEnabledMouseModeIsDecoded` reads the modes off `mouseOn`.** This is the mechanical form of "a mode enabled is a grammar accepted": adding `1005` to the constant reddens the suite. It is the model the other registry guard in this window should have copied (see Important 2).
- **`cmd/define/render_test.go:583` — `TestRegionsAddressTheRenderedOutput` is a corpus property, not an example.** Every region over every entry at two widths must sit at the line and *display cell* it claims, with `Width == visibleCells(Text)`. That is the assertion shape that catches an entry nobody thought of.

## 2. Critical findings

None.

## 3. Important findings

### I-1 · The production join between Render's regions and the click is pinned by nothing — `cmd/define/screen.go:551`

**This is the 6th finding in family `unfalsifiable-test-pin`.** Earlier rounds fixed instances (BR-13's nil-file restore pin, BR-24's pty-only exit sequence, BR-36's `>=` cursor assertion). Do **not** fix this instance alone.

Measured, not inferred. I swapped the two statements in `liveScreen.WriteRegions`:

```go
l.s.Write([]byte(text))   // was: addRegions first
l.s.addRegions(rs)
```

`go test ./cmd/define/` → **`ok`, 106s, green.** In production that mutation shifts every region forward by the entry's own line count: every click plays the wrong word or nothing. `liveScreen.WriteRegions` and `liveScreen.RegionAtRow` appear in no test file (`grep` finds only the `recordDisplay` double's methods). The loop tests script `RegionAtRow`'s answer; the screen tests call `addRegions` directly; nothing runs the two halves through the object production uses. `writeRendered`'s `w.(regionWriter)` assertion has the same shape — a wrapper introduced on the interactive stdout silently disables clicking with a green suite.

The rule the family keeps asking for, stated so it covers all six: **a Done-when's "pinned by" is satisfied only when mutating the production code that implements the claim reddens a named test — and a test double may not stand in for the object that *joins* two separately-pinned halves.** The enumeration that rule implies, and that should be written this round rather than the next: for every row of the plan's Core-concepts *Integration points* table, name the test that runs that object; `liveScreen.WriteRegions`, `liveScreen.RegionAtRow` and `writeRendered`'s seam are the three rows currently empty. A single in-process test — write a real `Render` output plus its regions into a real `liveScreen`, then click the coordinates the headword actually landed on — closes all three and is the test the swap above would redden. (The wiring is *correct* today; I confirmed a real `liveScreen` resolves row 1 col 3 to the headword. It is only undefended.)

### I-2 · `TestEveryRegionKindIsActionable` restates the enum's extent, so Done-when 7 cannot fire — `cmd/define/editorloop_test.go:851`

**This is the 4th finding in family `one-owner-per-invariant`** (after `escapeLen`, the rune-counted cursor move, and `topLine` — the last of which the implementor caught themselves). Do **not** fix this instance alone.

```go
for kind := RegionHeadword; kind <= RegionOriginLang; kind++ {
```

The plan's M2 Done-when row 7 says this is red "when a kind is added with no action". It is not: `RegionKind` has no count sentinel, so adding `RegionIPA` after `RegionOriginLang` leaves the loop's upper bound at the old last member and the new kind is never exercised — while `clicked`'s `switch` in `replraw.go:270` has no `default`, so it silently does nothing. The registry promise ("a third consumer is a row rather than a new feature") breaks exactly the way the issue's Done-when 2 exists to prevent, with a green suite.

**A second instance of the same rule lives in this window**, which is why the class is worth writing down rather than patching: `originLineRange` (`render.go:377`) re-derives the section boundary by a heuristic over rendered text — non-empty, equal to its own `ToUpper`, no `" .,‘’"` — when `e.Sections` already owns that structure. A wrapped ORIGIN body line that happens to be a single caps-or-digits token (`"1960:"` contains no excluded character and equals its own uppercase) ends the range early and drops every later ORIGIN region.

The rule: **the extent and the structure of a declared set have exactly one owner — the declaration — and every guard, loop bound and boundary test derives from it rather than restating it.** `TestEveryEnabledMouseModeIsDecoded` in this same milestone is the pattern; it reads its enumeration off `mouseOn`. The enumeration this rule implies, to be swept in the same round: every `for … <= LastMember` and every hand-written heading/boundary matcher in `cmd/define`. Concretely, `RegionKind` gains a `regionKinds` sentinel the test iterates to, and `originLineRange` counts `e.Sections` headings by name.

Cite: **ARCH-PURPOSE** — the very commit that made the mouse-mode rule mechanical (`209a8ff`, "the mode rule made mechanical") left its sibling registry rule restated. That is the instance fixed and the class left standing.

### I-3 · Neither README.md nor atlas/define.md documents the feature the issue is named for

**This is the 2nd finding in family `docs-lag-new-surface`.** Do **not** fix this instance alone.

`git diff --name-status` over the window touches `atlas/define.md` for 11 lines — all of it the `console` side-quest from `124c55c` — and does not touch `README.md` at all. Absent from both: that clicking a headword plays it, that clicking the language after `ORIGIN` plays it in that language, that an underlined span is what "clickable" looks like, and the degraded case (`Region.Word` replays a scrolled-past entry through `#29`'s fallback). `README.md:82-98` has the key table and the drag-select note from M1 and stops there; `atlas/define.md`'s "The screen" section still refers to clicks in the future tense (`:283` "M2's click is a method on `display`", `:326` "M2's click map", `:350` "M2.5 splices an underline"). The atlas also carries no record of the region registry, `regionsIn`'s read-the-finished-output decision, the occurrence-index carry-over, or `writeRendered`'s seam.

The cause is structural, and it is the rule: **M1.6 made the docs sweep a *task* ("a docs sweep follows ANY change to this surface"), and M2's task list M2.1–M2.6 has no docs row at all.** A sweep that lives in one milestone's task list is not swept at the next boundary. AGENTS.md §8 already states the obligation per-milestone; the fix that makes it stick is mechanical, and `repo_guard_test.go` already reads the `base..HEAD` window: a guard that fails when the window adds entities named in a milestone's Core-concepts table while `atlas/define.md` is unchanged in that same window. Failing that, every milestone's task list carries a docs row by construction. (Round 3 of the M1 review already wrote this rule — "the atlas is one of the sites, not a follow-up" — and it recurred anyway, which is the ledger reporting that the enumeration was never written.)

## 4. Minor findings

- `cmd/define/screen.go:284` — `markClickable` re-tests its column trigger after every byte step, so a span starting at an escape emits the attribute twice: the real headword case yields `"\x1b[4m\x1b[1;36m\x1b[4mpotassium\x1b[24m\x1b[0m …"` (measured). Idempotent, so cosmetic only; fire the "on" branch once per column. Family: `column-trigger-fires-per-byte`.
- `cmd/define/screen.go:171` — `LineAt` is tabled as PURE but `Frame()` clamps and writes back `s.offset`, so the hit test mutates the viewport. Safe (the caller holds `mu`), but the label overstates it.
- `cmd/define/key.go:270` (`digits`) and `screen.go:499` (`throttledPaint`) are new declarations absent from the plan's Core-concepts tables. Trivial private helpers; noting only so the next table edit is deliberate.

## 5. Test coverage notes

- **All 12 PTY conformance rows SKIP in this environment** — `pty_conformance_test.go:59: no pty available: operation not permitted`. So `TestPTYWithoutMouseBehavesAsBefore`, the named pin for M2 Done-when 6, could not be executed by this review. Row 6 does not rest on it alone (`TestNoColorTakesTheLineLoopAndEmitsNoEscapes` and `TestEveryEnabledMouseModeIsDecoded` both run in process), which is BR-24's rule being honoured. No click is exercised through a real pty anywhere; combined with I-1, nothing at all runs a click through production objects.
- Ran here: `go build ./...`, `go test ./...` green, `go test -race ./cmd/define/` green (118s), `go vet ./cmd/define/` clean, `go test -fuzz FuzzDecodeMouseIsBounded -fuzztime 45s` → 2,242,117 execs, 0 new interesting, PASS.
- Falsifiability spot-checks I ran: `topLine +1` → 3 tests red across 2 files (good); `WriteRegions` statement swap → suite green (I-1).
- Gap worth a row beyond I-1: `regionsIn`'s occurrence-index carry-over is checked against the corpus as it exists, but no case exercises a language name occurring **before** a source mention in a cognate clause. The arithmetic is right (the count is taken over the unmasked source prefix, so both sides see the same occurrences), but nothing pins it.

## 6. Architectural notes

- **ARCH-DRY — pass, with the I-2 exception.** `escapeLen` is genuinely the one escape grammar and all four walkers use it; `OriginLanguage` derives from the mentions producer rather than re-spelling the rule. Flagged under I-2: `originLineRange` re-derives section structure the `Entry` owns.
- **ARCH-PURE — pass.** Region collection, the hit test, and the mark are pure functions; IO is confined to `liveScreen`. `regionsIn` takes exactly what it reads. The one wrinkle is the PURE-labelled `LineAt` clamping (Minor).
- **ARCH-PURPOSE — flag.** Both click consumers ship, the registry exists, degrade is handled, and the shadow-sweep on the single-source change (one producer, two consumers of ORIGIN mentions) passes. The flag is the class-vs-instance axis: the mode rule was made mechanical in this window while its sibling registry rule was left restated (I-2), and the docs sweep was fixed as one milestone's task rather than as a boundary obligation (I-3).
- **ARCH-MOCK — flag.** The terminal's stateful double (the `creack/pty` harness) exists and gained a row, but production flow and test flow do **not** share the boundary for the click: the loop's tests inject a double that answers `RegionAtRow` from a scripted map, and the real `liveScreen` is on no test's path. That is I-1 restated in this principle's terms.
- **ARCH-CONSTRAINTS — pass.** The 16 ms repaint throttle with trailing flush is untouched; `markClickable` adds O(cells) per painted row per frame, bounded by the terminal, and returns immediately for the common no-regions line. The regions map is sparse and grows with the deliberately-uncapped buffer (already disposed as BR-2). `OriginLanguageMentions` compiles 22 regexes per call but runs once per lookup, not per keystroke.

## 7. Plan revision recommendations

1. **`## Revisions` — "Done-when 7's stated failure mode is not the one the test has."** Record that `TestEveryRegionKindIsActionable` restates `RegionKind`'s extent, so "red when a kind is added with no action" was false as written; record the sentinel-derived form that replaces it and the sibling instance (`originLineRange`), as the 4th and 5th entries in `one-owner-per-invariant`.
2. **`## Revisions` — "M2 shipped with no docs row, and that is the rule not the omission."** Record that M1.6 scoped the docs sweep to a task, that M2.1–M2.6 therefore carried none, and what makes it a per-boundary obligation (a `repo_guard_test.go` window guard, or a docs row in every milestone's task list by construction).
3. **`## Revisions` — the Integration-points rows with no pin.** Add `liveScreen.WriteRegions`, `liveScreen.RegionAtRow` and `writeRendered` to M2's Done-when with the test that runs them, and record the mutation (`addRegions`/`Write` swapped → suite green) as the evidence that the rows were empty.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      M1.1's row still enumerates the four cases verbatim and no chunk-boundary property test for screen.Write exists (screen_test.go has only FuzzScreenWriteDoesNotPanic); M1 is closed, so this is now a carried Minor.
findings:
  - id: new
    severity: Important
    family: unfalsifiable-test-pin
    title: |
      liveScreen.WriteRegions/RegionAtRow — the production click join is pinned by nothing (verified by mutation)
    detail: |
      Swapping `l.s.addRegions(rs)` and `l.s.Write([]byte(text))` in screen.go:551
      leaves `go test ./cmd/define/` green, though in production it shifts every
      region forward by the entry's line count. 6th in the family: state the rule
      — a "pinned by" claim holds only when mutating the implementing code reddens
      a named test, and a double may not stand in for the object joining two
      separately-pinned halves — then write the enumeration (every Integration-points
      row names the test that runs it) and sweep it this round.
  - id: new
    severity: Important
    family: one-owner-per-invariant
    title: |
      The region-registry guard restates RegionKind's extent, so Done-when 7 cannot fire
    detail: |
      editorloop_test.go:851 loops `kind <= RegionOriginLang`; RegionKind has no
      count sentinel, so a third kind is never exercised and `clicked`'s switch has
      no default. Second instance in the same window: originLineRange (render.go:377)
      re-derives the section boundary by an all-caps heuristic that `e.Sections`
      already owns. 4th/5th in the family: state the rule — the extent and structure
      of a declared set have one owner and every guard derives from it, as
      TestEveryEnabledMouseModeIsDecoded already does with mouseOn — and sweep both.
  - id: new
    severity: Important
    family: docs-lag-new-surface
    title: |
      README.md and atlas/define.md document none of M2's delivered surface
    detail: |
      README.md is untouched in the window; atlas/define.md changed 11 lines, all
      the console side-quest, and still refers to clicks in the future tense
      (:283, :326, :350). Missing: click-to-play, ORIGIN-language click, the
      underline mark, the scrolled-past degrade, the region registry, writeRendered's
      seam. 2nd in the family: the cause is that M1.6 made the sweep a TASK and
      M2.1-M2.6 carries no docs row — fix that, ideally as a repo_guard_test.go
      window guard, not this instance.
  - id: new
    severity: Minor
    family: column-trigger-fires-per-byte
    title: |
      markClickable emits the underline twice when a span begins at an escape
    detail: |
      Measured: markClickable("\x1b[1;36mpotassium\x1b[0m is a metal", {Col:0,Width:9})
      yields "\x1b[4m\x1b[1;36m\x1b[4mpotassium...". The column trigger is re-tested
      after each byte step. Idempotent, so cosmetic; fire it once per column.
  - id: new
    severity: Minor
    family: pure-label-hides-mutation
    title: |
      screen.LineAt is tabled PURE but clamps and writes back s.offset via Frame()
```

---

## Re-review — 2026-08-30T12:15:57-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 30 — clickable regions in the terminal: click ORIGIN French to hear it, click the IPA to replay |
| repo | tools |
| issue file | workshop/issues/000030-clickable-regions.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | a6584e243f2a032d209f8f4f1f44c50ed801f0ad..8f6a458a06ec9efe0ef93c8eeb1e16abdf751c97 |
| command | sdlc milestone-close --issue 30 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-08-30T12:15:57-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2 delivers both click consumers, the registry, the mark and the docs, and the three findings from round 6 are genuinely fixed — I confirmed each by mutation rather than by the commit message (the `addRegions`/`Write` swap now reddens `TestLiveScreenJoinsRegionsToTheLinesTheyWereRenderedFor`; a third `RegionKind` reddens all three registry guards; the atlas and README carry the surface). I also re-verified byte-identity independently: regenerating the corpus dump from `a6584e2` in a scratch worktree produces a file `cmp`-identical to the committed golden. `go test ./...`, `-race`, `go vet` and a 1.7M-exec fuzz run are green here. What blocks the boundary is one Critical I found on the production object: `screen.Frame`'s fast path returns without clamping, so whenever the viewport grows while the reader is scrolled back — maximise the window, close the suggestion menu, un-wrap a long prompt — `topLine` goes negative and the click map silently detaches from the text. Reproduced end to end on a real `liveScreen`: after a resize, `RegionAtRow(0,0)` on the row that actually shows the headword answers *nothing*, and the underline is painted nine rows below the word it belongs to. That is the exact property D1 and the whole alternate-screen decision exist to guarantee.

## 1. Strengths

- **`cmd/define/screen_test.go:832` — the BR-37 fix is falsifiable, and I checked rather than assumed.** Swapping `l.s.addRegions(rs)` / `l.s.Write(...)` in `screen.go:545` now reddens both subtests *and* the frame-mark assertion. The test drives the real object end to end (write → viewport row → region), which is the shape the family kept asking for.
- **`cmd/define/render.go:239` — `numRegionKinds` is a real owner, verified.** Adding a third kind reddens `TestEveryRegionKindIsActionable`, `TestEveryRegionKindIsNamed` and `TestAtlasDescribesEveryRegionKind` simultaneously. Deriving the *docs* guard from the same sentinel is the strongest part of the round: it converts BR-39's structural cause into a mechanism rather than a promise.
- **`cmd/define/render.go:363` — `originLineRange` now reads `e.Sections`.** The all-caps heuristic is gone and the section boundary comes from the parser that owns it. Right fix, right reason.
- **Byte-identity survives an adversarial check.** Independently regenerated from the base commit: identical to the 315,434-byte golden. The shape that makes it cheap (`regionsIn` reads the finished string; `Render`'s body untouched) is what earns it.
- **`cmd/define/screen.go:305` — `clipVisible` closes what `markClickable` opened.** I probed the interaction: a clip that cuts mid-span emits `\x1b[0m`, so a marked span truncated at the right edge cannot leak its underline into the rows below. Confirmed-good ground.

## 2. Critical findings

### C-1 · `Frame`'s fast path skips `clamp`, so the click map detaches from the text after the viewport grows — `cmd/define/screen.go:197-206`

**This is the 5th finding in family `one-owner-per-invariant`.** Earlier rounds fixed instances (`escapeLen`, the rune-counted cursor move, `topLine`, `numRegionKinds`). Do **not** fix only this instance — state the rule and sweep it.

```go
func (s *screen) Frame() []string {
	if s.rows <= 0 { return nil }
	if len(s.lines) <= s.rows { return s.lines }   // ← returns WITHOUT clamping
	s.clamp()
	...
}
```

`clamp` is documented as "the ONE place the viewport's limits are spelled" (`screen.go:221`). The early return is a second, implicit answer to the same question — "it all fits, so the offset does not matter" — and it is wrong, because `topLine` reads the offset unconditionally (`topLine = len(lines) - offset - len(frame)`).

Measured on the production object:

```
newLiveScreen(&tty, 12, 80); WriteRegions(20 lines, headword region on line 0)
Page(10)                       → offset 9   (correct while rows = 11)
Resize(30, 80); Draw("> ", nil) → rows 29, len(lines) 20 → fast path, offset STILL 9
                                 topLine = -9
RegionAtRow(0,0) → nothing      (row 0 is where the headword is painted)
```

Every mark is painted `offset` rows away from its own text (`Paint` looks up `s.regions[top+i]` with a negative `top`), and a click on the word itself is dead. Three routine triggers, none of them exotic: resize taller while scrolled back; close the suggestion menu while scrolled back (`s.rows` grows by `menuRows`); kill a wrapped prompt with Ctrl-U (`s.rows` grows by `promptRows-1`). `screen.write` resets `offset` to 0, so the state only survives until the next write — but a click *is* the next input in exactly this scenario.

**Fix (verified):** hoist `s.clamp()` above the early return. I applied it, the repro resolves (`offset 0`, `topLine 0`, row 0 → `"potassium"`), and `go test ./cmd/define/` stays green.

**The rule the family keeps asking for, stated to cover all five:** *a derived invariant is re-established on every path that reads it, not only on the path that happens to call its owner.* The enumeration that implies, to be swept in this round: every early return and every fast path in `screen.go` that returns before `clamp`, and every reader of `s.offset`/`s.rows` outside `Paint`. Cite **ARCH-PURE** as well — `Frame` is tabled PURE while writing back `s.offset`, and that unstated write-on-read is precisely what makes the missing clamp invisible at the call site (`LineAt` looks like a query).

## 3. Important findings

### I-1 · The enumeration BR-37 demanded was not written, and two fixes in this same commit shipped unpinned — `workshop/plans/000030-clickable-regions-plan.md:223`

**This is the 7th finding in family `unfalsifiable-test-pin`.** Do **not** fix this instance — the rule is the deliverable.

The rule was *stated* (commit body, plan Revisions) and the named instance was *fixed and pinned*. The enumeration was not:

- **Done-when row 1's `pinned by` cell is unchanged.** It still names `TestClickOnHeadwordReplays`, `TestScreenResolvesAClickToWhatWasRenderedThere`, `TestClicksFollowTheTextWhenScrolled`, `TestRegionsLandOnTheLinesTheirRenderWroteTo` — the exact four that the `addRegions`/`Write` swap left green last round. The test that actually defends the claim, `TestLiveScreenJoinsRegionsToTheLinesTheyWereRenderedFor`, is named nowhere in the plan. Round 6's own plan-revision recommendation #3 asked for this and it was not done.
- **`originLineRange`'s fix is unpinned.** I reverted it to the exact all-caps heuristic BR-38 named and ran the full package: `ok github.com/xianxu/tools/cmd/define 106.9s`. A structural fix nothing can fail is the same shape as the finding that produced it.
- **C-1 above is green with and without its fix**, which is the third empty row.

For the record, one row of the enumeration *is* filled and I verified it rather than assuming: disabling `writeRendered`'s `w.(regionWriter)` branch reddens `TestALookupHandsItsRegionsToTheScreen`. Family prevalence: 7 findings across 7 rounds on this issue.

**The enumeration to write, this round:** for every M2 Core-concepts row and every M2 Done-when `pinned by` cell, name the test that reddens when the *implementing* code is mutated, and record the mutation used. Currently empty: `originLineRange`'s section derivation, `Frame`'s clamp, and Done-when 1's production join.

## 4. Minor findings

- `cmd/define/key.go:186,196,344` — `decodeWheel`'s doc says it "answers only the WHEEL" and that "a click therefore stays `KeyUnknown` — consumed whole and inert"; `decodeX10Mouse`'s says "Only the wheel is answered, matching `decodeWheel`". Both are false at HEAD — the function returns `KeyClick` — and the name `decodeWheel` is now a misnomer. 2nd in `stale-rationale`: the rule is that a comment stating what a function does *not* do is swept in the commit that makes it do it; both sites, plus the name.
- `cmd/define/screen.go:143` — `addRegions` decrements `base` for a partial line but leaves `Col` untouched, so a render starting mid-line lands its regions in the wrong columns. `screen_test.go:738` looks like it covers the branch but passes `Col: 12` already offset by hand, so it asserts the caller's arithmetic rather than the code's. Either enforce the documented precondition or shift `Col` by the open line's `visibleCells`.
- `cmd/define/render.go:396` (`headingLine`) is a new declaration absent from the plan's M2 Core-concepts table, alongside `digits` and `throttledPaint` noted last round.
- M2's Core-concepts table declares no PURE/INTEGRATION kind for four rows (`liveScreen.WriteRegions`/`RegionAtRow`, `regionWriter`/`writeRendered`, `Region.Word`), unlike M1's split tables — so the cross-check has nothing to read for exactly the IO rows.

## 5. Test coverage notes

- Ran here: `go build ./...`; `go test ./...` green (`cmd/define` 107s); `go test -race ./cmd/define/` green (118s); `go vet ./cmd/define/` clean; `FuzzDecodeMouseIsBounded` 30s → 1,709,828 execs, 0 new interesting, PASS.
- Mutations run: `addRegions`/`Write` swap → **red** (BR-37 confirmed); third `RegionKind` → **red** ×3 (BR-38 confirmed); `writeRendered` seam disabled → **red**; `originLineRange` reverted to the heuristic → **green** (I-1); `Frame` clamp hoisted → **green** either way (C-1).
- **All 12 PTY conformance rows SKIP here** (`no pty available: operation not permitted`), including `TestPTYWithoutMouseBehavesAsBefore`, the named pin for Done-when 6. Unchanged from last round; row 6 does not rest on it alone.
- Gap: nothing exercises the hit test with a *stale non-zero offset over a buffer that fits* — the state C-1 lives in. `TestScreenResolvesAClickToWhatWasRenderedThere` uses `rows 10 / 5 lines / offset 0`, and `TestClicksFollowTheTextWhenScrolled` uses `31 lines / rows 5`, so both sit on opposite sides of the untested case.
- Still unpinned from round 6's note: no case exercises a language name occurring in a cognate clause *before* a source mention.

## 6. Architectural notes

- **ARCH-DRY — pass.** `escapeLen` remains the one escape grammar (`visibleIndex`, `markClickable`, `clipVisible`); `clickAt`/`wheelFromButton` own the wire conversion for both encodings; `OriginLanguage` derives from the mentions producer. No new duplication in this round's diff.
- **ARCH-PURE — flag.** `regionsIn`, `markClickable` and `RegionAt` are genuinely pure. `Frame`/`LineAt` are tabled PURE and write back `s.offset` (BR-41, still open) — and C-1 is that label's cost, not a cosmetic mislabel: a reader of `LineAt` has no signal that a viewport-limit invariant depends on it having been called.
- **ARCH-PURPOSE — flag.** Both consumers, the registry, degrade and docs ship; the shadow-sweep on the single-source change (one cut-and-mask, two consumers) passes. The flag is the class-vs-instance axis again: three instances fixed, the enumeration BR-37 asked for not written, and two of this commit's own fixes landing with nothing that could fail without them (I-1).
- **ARCH-MOCK — flag.** The `creack/pty` double gained `TestPTYWithoutMouseBehavesAsBefore`, and the in-process `liveScreen` test closes the worst of last round's "production flow and test flow do not share the boundary". But no click is exercised through a real pty anywhere, and every pty row skips in this environment, so the terminal double is again unexecuted at the gate.
- **ARCH-CONSTRAINTS — pass, with a note.** `markClickable` is O(cells) per painted row inside the 16 ms throttle and returns immediately for an unmarked line; the regions map is sparse. `Render` now runs `regionsIn` (22 regex compiles via `OriginLanguageMentions`, plus `anyLanguageIn`) on *every* path including those that discard the map — `--play`'s `todaysQuestions` loop, the one-shot, pipes. Bounded and once-per-entry, not a keystroke path, so not a finding; compiling the 22 patterns once at package scope would be the free win if it ever shows up.

## 7. Plan revision recommendations

1. **`## Revisions` — "the viewport's limits had a second owner, and it was an early return."** Record C-1: `Frame`'s fast path returned without `clamp`, so a viewport that grows while scrolled back left `topLine` negative and detached the click map from the text; record the three triggers (resize taller, menu closes, wrapped prompt killed), the measured repro on `liveScreen`, and the rule as the 5th entry in `one-owner-per-invariant`.
2. **Done-when row 1 (`plan.md:223`) — add the test that actually defends it.** Name `TestLiveScreenJoinsRegionsToTheLinesTheyWereRenderedFor` in the `pinned by` cell with its `red when` (the `addRegions`/`Write` swap), and add a row for the clamp property once C-1 is pinned. Record that the four tests currently named all stayed green under the swap.
3. **`## Revisions` — the enumeration, written rather than promised.** For each M2 Core-concepts row, the test that runs it and the mutation that reddens it; explicitly mark `originLineRange` as fixed-but-unpinned (reverting to the heuristic → suite green, measured) so the gap is on the record rather than in a reviewer's head.
4. **M2 Core-concepts table — declare a Kind for the four IO rows** (`liveScreen.WriteRegions`/`RegionAtRow`, `regionWriter`/`writeRendered`, `Region.Word`), add `headingLine`, and correct `screen.LineAt`'s `PURE` to note the write-back (BR-41).

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      M1.1's row still enumerates the four cases and no chunk-boundary property exists for screen.Write; M1 is closed, so this stays a carried Minor.
  - id: BR-37
    disposition: addressed
    note: |
      Verified by mutation: swapping addRegions/Write reddens TestLiveScreenJoinsRegionsToTheLinesTheyWereRenderedFor; writeRendered's seam also reddens when disabled. The class enumeration it demanded is raised separately as the 7th in the family.
  - id: BR-38
    disposition: addressed
    note: |
      numRegionKinds verified by mutation (a third kind reddens all three guards); originLineRange now derives from e.Sections, though reverting it leaves the suite green.
  - id: BR-39
    disposition: addressed
    note: |
      README gains the clickable section, atlas gains "## Clickable regions", and TestAtlasDescribesEveryRegionKind derives from numRegionKinds - verified red for an undescribed third kind.
  - id: BR-40
    disposition: not-addressed
    note: |
      Re-measured at HEAD: markClickable("\x1b[1;36mpotassium\x1b[0m is a metal", {Col:0,Width:9}) still yields "\x1b[4m\x1b[1;36m\x1b[4mpotassium...".
  - id: BR-41
    disposition: not-addressed
    note: |
      The plan's M2 row still tables screen.LineAt as PURE; Frame still clamps and writes back s.offset. C-1 is the cost of that unstated write.
findings:
  - id: new
    severity: Critical
    family: one-owner-per-invariant
    title: |
      screen.Frame's fast path returns without clamping, so the click map detaches from the text when the viewport grows
    detail: |
      5th in the family, so the rule is the deliverable: a derived invariant is
      re-established on every path that reads it, not only on the path that calls
      its owner. Measured on the real liveScreen — 20 lines at 12 rows, Page(10)
      then Resize(30,80): offset stays 9, topLine = -9, and RegionAtRow(0,0) on
      the row that actually shows the headword answers nothing while its underline
      is painted 9 rows lower. Three routine triggers: resize taller while scrolled
      back, the suggestion menu closing, a wrapped prompt killed with Ctrl-U.
      Hoisting s.clamp() above the early return fixes the repro and keeps the suite
      green — which is also the second half of the finding, since nothing pins it.
      Sweep: every early return in screen.go that precedes clamp, and every reader
      of s.offset/s.rows outside Paint. ARCH-PURE: Frame/LineAt are tabled PURE
      while writing back s.offset, which is what hides the missing clamp.
  - id: new
    severity: Important
    family: unfalsifiable-test-pin
    title: |
      BR-37's enumeration was never written, and two fixes in the same commit ship with nothing that fails without them
    detail: |
      7th in the family, across 7 rounds on this issue. The rule was stated and the
      named instance pinned, but: Done-when row 1 still names only the four tests
      that stayed green under last round's mutation, and never names
      TestLiveScreenJoinsRegionsToTheLinesTheyWereRenderedFor; reverting
      originLineRange to the exact all-caps heuristic BR-38 named leaves
      `go test ./cmd/define/` green (measured, 107s); and C-1's clamp is green with
      and without its fix. One row IS filled and I checked it: disabling
      writeRendered's regionWriter branch reddens TestALookupHandsItsRegionsToTheScreen.
      Deliverable: for every M2 Core-concepts row and every Done-when "pinned by"
      cell, name the test and record the mutation that reddens it.
  - id: new
    severity: Minor
    family: stale-rationale
    title: |
      decodeWheel's and decodeX10Mouse's comments still say a click stays KeyUnknown, which stopped being true in this window
    detail: |
      key.go:186 and :196 say the decoder "answers only the WHEEL" and that "a click
      therefore stays KeyUnknown — consumed whole and inert"; key.go:344 repeats it
      for the X10 path. Both functions now return KeyClick, and the name decodeWheel
      is a misnomer for a decoder that also decodes presses. 2nd in the family: the
      rule is that a comment stating what a function does NOT do is swept in the
      commit that makes it do it — sweep both sites and the name.
  - id: new
    severity: Minor
    family: vacuous-pin
    title: |
      addRegions shifts base for a partial line but not Col, and the test that looks like it covers this passes Col pre-offset
    detail: |
      screen.go:143 decrements base when s.partial, so a render starting mid-line
      lands on the right LINE — but Col is left relative to the render, not to the
      buffer line, so the regions sit in the wrong columns. screen_test.go:738 seems
      to cover the branch and instead supplies Col: 12 already offset by hand, so it
      asserts the caller's arithmetic. Not reachable today (the loop writes "\r\n"
      before an entry), but the comment documents one contract and the test another.
      Either enforce the precondition or shift Col by visibleCells of the open line.
```

---

## Re-review — 2026-08-30T12:49:38-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 30 — clickable regions in the terminal: click ORIGIN French to hear it, click the IPA to replay |
| repo | tools |
| issue file | workshop/issues/000030-clickable-regions.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | a6584e243f2a032d209f8f4f1f44c50ed801f0ad..946947452090ea93f52c9007991c685d5e7e8ab6 |
| command | sdlc milestone-close --issue 30 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-08-30T12:49:38-07:00 |
| verdict | REWORK |

## Review

I've completed the review. Full mutation-based verification of the prior round's claims, plus independent regeneration of the corpus golden, plus end-to-end probes of the click path against the real `liveScreen`.

```verdict
verdict: REWORK
confidence: high
```

M2 delivers the shape the issue was filed for — one registry, guards that derive from `numRegionKinds`, regions read out of the finished output so `Render`'s bytes are provably unmoved, and a hit test whose two ends I confirmed agree in production. The prior round's two open blockers are genuinely fixed and genuinely pinned: I reverted `screen.go` to `8f6a458` and `TestClickMapSurvivesTheViewportGrowing` fails with the exact message the plan records, and seven of the enumeration table's "mutation that reddens it" rows redden with their recorded messages when I apply the named mutation. What blocks SHIP is a correctness defect the corpus property test is structurally blind to: **on a multi-word entry the clickable headword is only the first field, and clicking it plays a different word.** Measured on the committed `hot dog` fixture — a bare Enter asks the CDN for `hot_dog_en_us_1.mp3`, the click on the same entry's underlined headword asks for `hot_en_us_1.mp3`, `hot_en_us_2.mp3` and the `hot--` fallbacks. `a priori` is the same: the region is the single letter `a`.

## 1. Strengths

- **`cmd/define/testdata/golden/render-corpus.golden` — the byte-identity claim survives an adversarial check I ran myself.** I exported `a6584e2` to a scratch tree, wrote a generator using the *pre-change* single-return `Render`, and `cmp` reports the output identical to the committed 315,434-byte golden. Done-when 3 is real evidence, not a self-comparison.
- **`cmd/define/screen.go:205` — BR-42's fix is structural and falsifiable.** `visible()` returning frame-and-top as one answer is the right shape, and reverting `screen.go` to the previous commit reddens `TestClickMapSurvivesTheViewportGrowing` with "the word shows on row 20 and offers nothing". Note that hoisting *only* the clamp (leaving the old `topLine`) leaves it green — the one-fact refactor is what earns the pin, which is exactly the argument the comment makes.
- **`workshop/plans/000030-clickable-regions-plan.md:230` — the BR-43 enumeration holds up under sampling.** I mutated seven rows (`addRegions` partial decrement, `markClickable`'s `underlineOff`, `findVisible`'s column +1, `writeRendered`'s `regionWriter` branch, `clicked`'s `r.Lang`, `WriteRegions`' statement order, `maskOut` → length-changing `ReplaceAll`) and every one reddened with the message the table records.
- **`cmd/define/editorloop_test.go:863` — `TestEveryRegionKindIsActionable` is derived *and* end-to-end.** It loops to `numRegionKinds` and drives a real `runEditor` with a `KeyClick` per kind, asserting on play count. That is the registry promise checked rather than restated.
- **`cmd/define/repo_guard_test.go:92` — `declaredInBlock`/`declaredAsField` fix the guard without loosening it.** The comments explain precisely why an unqualified field match was refused; a guard that a struct field of the wrong type could satisfy would be the same class of lie it exists to catch.

## 2. Critical findings

### C-1 · A multi-word headword's click plays a different word — `cmd/define/render.go:307`

`regionsIn` builds the headword region from individual `HeadWord`/`HeadSyllables` tokens and stamps `Word: e.Headword()`. `parseHead` (`parse.go:436`) makes only `fields[0]` a `HeadWord`; the rest of a multi-word head is `HeadOther`. So for `hot dog` the entry produces exactly one region — `{Text:"hot", Word:"hot", Col:0, Width:3}` — and `clicked` (`replraw.go:262`) builds `session{current:"hot"}`, which `utteranceFor` turns into `hot_*` URLs.

Measured, both gestures on the same entry:

```
bare-Enter CDN: [.../hot_dog_en_us_1.mp3]
click CDN:      [.../hot_dog_en_us_1.mp3   ← the lookup
                 .../hot_en_us_1.mp3 .../hot_en_us_2.mp3
                 /sounds/oxford/hot--_us_1.mp3 /sounds/oxford/hot--_us_2.mp3]
```

`a priori` is the same shape: the only region is `{Text:"a", Word:"a", Width:1}`, so the underline marks a single letter in `a priori  a pri·o·ri` and the click asks for `a_*`. Both fixtures are committed, and multi-word heads are ordinary NOAD (`ad hoc`, `de facto`, `ice cream`). `RegionOriginLang` inherits it — `Word` is the same `e.Headword()`, so an ORIGIN click on `hot dog` would replay `hot` in French.

The mark is wrong in the same breath: a reader sees `hot dog` with only `hot` underlined, which claims the wrong extent (Done-when 4).

Fix sketch: derive the region's playable identity from the same source the gesture it shortcuts uses. `lookupAndRender` already holds the lookup key — the thing a bare Enter replays — so stamp it onto the regions at the `writeRendered` seam (or pass it into `regionsIn`), rather than re-deriving from `Headword()`, whose "first field" contract exists for a different purpose. Separately, widen the span so the underline covers the whole head phrase, and add the corpus property that is missing: for every headword region, the first audio candidate for `r.Word` must equal the first candidate a bare-Enter replay of that entry produces. `hot dog` and `a priori` make it red today.

Family: `shortcut-rederives-its-target` (new — a gesture advertised as a shortcut must resolve its target from the same source as the gesture it shortcuts, not re-derive it).

## 3. Important findings

### I-1 · The click's coordinate joint — terminal row → viewport row — is pinned by nothing

**This is the 8th finding in family `unfalsifiable-test-pin`.** Earlier rounds fixed instances (BR-37 pinned `liveScreen.WriteRegions`; BR-43 wrote the enumeration). Do not fix this instance in isolation — the rule the family keeps producing is: **the enumeration must be of JOINTS, not of entities.** Every row in the M2 table names a *function*; what has now failed three times is the place where two separately-pinned layers exchange a value across a boundary (render coordinates ↔ buffer lines, wire coordinates ↔ screen cells, terminal rows ↔ viewport rows). A joint gets a row only if a test drives *both real objects*, and the enumeration should be regenerated with that as its unit.

The concrete gap: `runEditor` is never driven against a real `liveScreen` (grep — `newLiveScreen` appears only in `screen_test.go` and `replraw_test.go`). `TestClickOnHeadwordReplays` and `TestClickOnOriginLanguagePlaysIt` script `RegionAtRow`'s answer on `recordDisplay`, choosing row/col on both sides; `TestLiveScreenJoins…` calls `RegionAtRow` directly with hand-written regions. Nothing asserts that the row `decodeKey` reports is the row `screen.LineAt` indexes — an invariant that holds only because `Paint` starts at `cursorHome` and `clipVisible` guarantees one terminal row per frame line. I verified by probe that it *is* correct today (real `liveScreen` as view and stdout, real `concrete` lookup, click at the frame cell where `French` actually renders → `concrete_fr_*` requested), so this is coverage, not a defect. The cheap pin is that probe, promoted to a test.

Note also there is no pty row that sends a real mouse report and asserts playback (ARCH-MOCK): `TestPTYWithoutMouseBehavesAsBefore` covers only the mouse-*less* case. The in-process end-to-end is the better ask — pty rows skip wherever no pty exists, including in this review (all 12 rows `SKIP: no pty available: operation not permitted`).

## 4. Minor findings

- **`TestClickMapSurvivesTheViewportGrowing/the command menu closing` passes against the pre-fix code.** 3rd in `vacuous-pin` — state the rule rather than patching the case: a subtest earns its row only if it reddens under the mutation the table names. Measured: with `screen.go` reverted to `8f6a458`, `a resize taller` fails and `the command menu closing` passes, because at 21 lines / 11 rows it never reaches the un-clamped early return.
- **ARCH-DRY: four hand-rolled "walk styled text by cell" loops** — `visibleCells` (render.go:466), `visibleIndex` (render.go:~700), `clipVisible` (screen.go:658), `markClickable` (screen.go:288). They correctly share `escapeLen` and `cellWidth`, but each re-spells the traversal skeleton. 6th in `one-owner-per-invariant`: the rule is already written, so the durable fix is structural — one `forEachCell(line, func(byteRange, col))` iterator the four callers consume — not a fifth careful copy when the next site arrives.
- **The issue's `## Log` carries no M2 entry at all.** M1 logged eight; every M2 discovery (`Region.Word`, `writeRendered`'s seam, the `24`-not-`0` decision, the degrade-by-routing rule) lives only in the plan's Revisions. AGENTS.md §2 puts discoveries in the issue Log, and a reader of the tracker currently sees M1 finish and nothing after.

## 5. Test coverage notes

`go test ./...`, `go test -race ./cmd/define/` and `go vet ./cmd/define/` are green at HEAD (107s / 118s). The 12 PTY conformance rows all SKIP here (`no pty available: operation not permitted`), so I could not confirm the pty claims — `conformance.SkipOrFail` means CI still enforces them.

The corpus-wide tests are the right shape and I checked their reach: `TestRegionsAddressTheRenderedOutput` asserts a region's *address* over 34 entries × 2 widths and reddens on a one-column shift; `TestRenderNeverMarksSpansItself` guards the mark's absence with a non-vacuous count. Neither asserts a region's *meaning* — that the span covers the headword and that `Word` is what the entry replays as — which is exactly the hole C-1 fell through. I also probed for overlapping regions across the corpus at three widths: none, so `markClickable`'s monotonic `next` walk is safe today.

## 6. Architectural notes

- **ARCH-DRY** — flag, Minor above (the four cell-walks).
- **ARCH-PURE** — pass. `screen` stays free of IO, `liveScreen` is the only part touching a terminal, `regionsIn`/`markClickable`/`clickAt` are pure and unit-tested with no pty. The `visible()` clamp still writes back `s.offset` under a PURE table row (BR-41, open) — harmless, but the label is doing less work than it claims.
- **ARCH-PURPOSE** — flag, via C-1. The shadow-sweep on the registry passes: `numRegionKinds` is the extent and all three guards (`Actionable`, `Named`, `AtlasDescribes`) derive from it, so a third kind is a row. But the issue's stated purpose is "the playback target is the HEADWORD, so it exists in every entry and every language" — and for a multi-word headword the target delivered is the first field, which is a different word. That is the purpose, not a follow-up.
- **ARCH-MOCK** — pass with a note. The terminal's stateful double (the `creack/pty` harness) gained a row for the mouse-less case; the click gesture itself has no conformance row, noted in I-1.
- **ARCH-CONSTRAINTS** — pass. `markClickable` early-returns on a region-free line, so the added per-paint cost is bounded by the ≤2 lines that carry regions; the 16 ms throttle and trailing flush are unchanged. `s.regions` grows with the session under the deliberately-uncapped buffer decision, which is consistent with D3.

## 7. Plan revision recommendations

- **A `## Revisions` entry for C-1**, recording that `Region.Word` and the headword span were derived from `e.Headword()` / individual `HeadWord` tokens, that this is only the first field of a multi-word head, and what they derive from instead. Add a row to the *M2 — what runs each row* table: `Region.Word` | *the new property test* | "look up `hot dog` and click the headword → asks for `hot_*` rather than `hot_dog_*`".
- **Regenerate the enumeration table by JOINT rather than by entity** (I-1). At minimum add the row `Key.Row → screen.LineAt` | *the promoted end-to-end test* | "offset the frame by a header row → the click resolves one line off", and mark the `screen.visible clamp` row's second subtest as not falsifiable.
- **Done-when row 1** should name the object that joins the loop to the real screen, the same way it now names `TestLiveScreenJoinsRegionsToTheLinesTheyWereRenderedFor` — the current entry stops one layer short.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Still open: no chunk-boundary property over screen.Write exists — TestScreenWriteBuildsLines is still the four enumerated cases.
  - id: BR-40
    disposition: not-addressed
    note: |
      Reproduced at HEAD: markClickable("\x1b[1;36mpotassium\x1b[0m is a metal", {Col:0,Width:9}) = "\x1b[4m\x1b[1;36m\x1b[4mpotassium\x1b[24m\x1b[0m is a metal".
  - id: BR-41
    disposition: not-addressed
    note: |
      LineAt now reaches clamp through visible() rather than Frame(), so the write-back under a PURE table row remains.
  - id: BR-42
    disposition: addressed
    note: |
      Verified by revert: screen.go at 8f6a458 reddens TestClickMapSurvivesTheViewportGrowing with the exact recorded message.
  - id: BR-43
    disposition: addressed
    note: |
      Enumeration exists and holds: seven sampled rows each reddened under the mutation the table names.
  - id: BR-44
    disposition: not-addressed
    note: |
      key.go:186/196 still say the decoder answers only the wheel and a click stays KeyUnknown; key.go:344 repeats it; decodeWheel is still the name.
  - id: BR-45
    disposition: not-addressed
    note: |
      Reproduced: a mid-line render with Col relative to the render resolves at buffer column 0, not at the open line's width.
findings:
  - id: new
    severity: Critical
    family: shortcut-rederives-its-target
    title: |
      A multi-word entry's clickable headword is only its first field, so the click plays a different word
    detail: |
      regionsIn (render.go:307) builds the headword region from individual HeadWord
      tokens and stamps Word: e.Headword(), which parseHead (parse.go:436) fills
      from fields[0] alone. Measured on the committed `hot dog` fixture: a bare
      Enter asks the CDN for hot_dog_en_us_1.mp3, while a click on the same
      entry's underlined headword asks for hot_en_us_1.mp3, hot_en_us_2.mp3 and
      the hot-- fallbacks. `a priori` produces a single region for the letter "a".
      RegionOriginLang carries the same Word, so a French replay on such an entry
      is also the wrong word. The mark is wrong with it: only "hot" is underlined
      in a head line reading "hot dog". Derive the target from the same source the
      gesture it shortcuts uses — lookupAndRender holds the lookup key — and widen
      the span to the whole head phrase. The missing property: for every headword
      region, the first audio candidate for r.Word equals the first candidate a
      bare-Enter replay of that entry produces.
  - id: new
    severity: Important
    family: unfalsifiable-test-pin
    title: |
      The terminal-row to viewport-row joint is pinned by nothing, and runEditor never runs against a real liveScreen
    detail: |
      8th in this family. Do NOT fix only this instance — the rule the family keeps
      producing is that the enumeration must be of JOINTS, not entities: every place
      two separately-pinned layers exchange a value across a coordinate or unit
      boundary gets a row, and a row earns it only when a test drives both real
      objects. Concretely: newLiveScreen appears in no editor-loop test, both click
      action tests script recordDisplay.RegionAtRow's answer with row/col of the
      test's own choosing, and TestLiveScreenJoins… calls RegionAtRow directly with
      hand-written regions. Nothing asserts Key.Row is the row screen.LineAt indexes.
      I verified it is correct today by probe (real liveScreen as view and stdout, real
      `concrete` lookup, click at the frame cell where French renders → concrete_fr_*),
      so this is coverage, not a defect. Also no pty row sends a real mouse report.
  - id: new
    severity: Minor
    family: vacuous-pin
    title: |
      TestClickMapSurvivesTheViewportGrowing's "the command menu closing" subtest is green against the pre-fix code
    detail: |
      3rd in this family, so the deliverable is the rule: a subtest earns its row in
      the mutation table only if it reddens under the mutation that row names.
      Measured — with screen.go reverted to 8f6a458, "a resize taller" fails and
      "the command menu closing" passes, because at 21 lines into 11 rows the
      un-clamped early return is never taken.
  - id: new
    severity: Minor
    family: one-owner-per-invariant
    title: |
      Four hand-rolled walks of styled text by display cell
    detail: |
      6th in this family, so state the rule rather than patch a site: visibleCells
      (render.go:466), visibleIndex (render.go), clipVisible (screen.go:658) and
      markClickable (screen.go:288) correctly share escapeLen and cellWidth but each
      re-spells the traversal. The structural fix is one forEachCell(line, fn)
      iterator the four consume, before M2.5's splice adds a fifth reading.
  - id: new
    severity: Minor
    family: docs-lag-new-surface
    title: |
      The issue's Log carries no M2 entry; every M2 discovery lives only in the plan's Revisions
    detail: |
      3rd in this family, so the rule: the issue Log is one of the sites a fact has
      to reach, not a follow-up. M1 logged eight entries; M2 logged none, so a reader
      of workshop/issues/000030-clickable-regions.md sees M1 close and nothing after,
      while Region.Word, the writeRendered seam, the 24-not-0 decision and the
      degrade-by-routing rule are recorded only in the plan.
```

---

## Re-review — 2026-08-30T14:35:48-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 30 — clickable regions in the terminal: click ORIGIN French to hear it, click the IPA to replay |
| repo | tools |
| issue file | workshop/issues/000030-clickable-regions.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | a6584e243f2a032d209f8f4f1f44c50ed801f0ad..eb85b10e3cec8f10bc4aa6ba71c2533eaed9d4e4 |
| command | sdlc milestone-close --issue 30 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-08-30T14:35:48-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The BR-46 and BR-47 fixes are real and I verified both by mutation rather than by reading the commit message: reverting `regionsIn` to `e.Headword()` reddens `TestAClickAsksForExactlyWhatEnterAsksFor`, `TestTheClickableSpanIsTheWholeHeadword` and `TestASpanNotOnTheLineFallsBackToTheHeadword` with the exact messages the plan records (`hot dog`, `a priori`, `bargainer`), and the new joint test reddens for an off-by-one on *either* side of the boundary (`clickAt`'s 1-based conversion, and `visible()`'s top line) — it is the first test in this issue that crosses from terminal bytes to a played word through real objects, and it is stable at `-count=25 -race`. What blocks SHIP is that the same commit introduced a **panic**: `regionsIn` now hands the lookup key (and, on the `--play` path, an empty headword) to `findVisible`, which indexes `cols[0]` without checking, so `Render(ParseEntry(" "), …)` crashes. This is not a hypothetical — the repo's own `FuzzRenderLosesNothing` finds it in 0.18 s and minimizes the crasher to a single space, while `go test ./...` stays green because it only runs the seed corpus. Everything else in the window is green here (`go test ./...`, `go test -race ./cmd/define/ ./internal/...`, `go vet ./...`); the 12 PTY rows skip (`no pty available: operation not permitted`).

## 1. Strengths

- **`cmd/define/render.go:332` — the fix is the class, not the site.** Taking the target from `RenderOpts.Word` also corrected `RegionOriginLang` (which carried the same wrong word) and the *mark's* extent in the same move, and the fallback to the headword token when the key is not on the line (`define jalapeno` → `jalapeño`) keeps the span honest without letting the played word drift. I probed the whole corpus at three widths: every region's `Word` is the key, no zero-width region, no two regions on one line overlap.
- **`cmd/define/editorloop_test.go:906` — `TestAClickAtAPaintedCellPlaysWhatIsUnderIt` is the joint test the family kept asking for.** No coordinate is invented by the test; it reads the frame the screen is *currently* showing under the screen's own lock, builds the SGR bytes a terminal would send, and decodes them through `decodeKey`. Both recorded mutations redden it. The marker-keystroke idle signal is the right answer to the flake — a frame count would not have been.
- **`cmd/define/screen.go:289` — `markClickable`'s rewrite is a genuine improvement beyond BR-40.** Stepping escapes first also fixed a second latent bug: a span whose `Col` lies past the line's width used to emit a stray `\x1b[24m` at end of line (`next < len(spans)`); the `open` flag makes the close conditional on an open span. Measured at HEAD: `"\x1b[1;36m\x1b[4mpotassium\x1b[24m\x1b[0m is a metal"`.
- **`internal/llm/config_test.go:170` — the side-quest's negative cases are the interesting ones and they are all there.** "another local port" and "a remote provider" both decline, an explicit key wins in both directions, and the same endpoint spelled explicitly still handshakes. `TestResolveIsPureOverTheLookup` was rewritten to assert the property rather than the instrument it had been using, which is the right response to a test whose observable disappeared.
- **`cmd/define/repo_guard_test.go:86` — `declaredInBlock`/`declaredAsField` widen the plan guard without loosening it.** The comments explain exactly why an unqualified field match was refused; a guard a struct field of the wrong type could satisfy would be the class of lie the guard exists to catch.

## 2. Critical findings

### C-1 · `Render` panics on a one-space entry — `cmd/define/render.go:445`, reached from `:333` and `:340`

`regionsIn` now passes a *span string* to `findVisible`, and two of them can be empty: `e.Headword()` when the key is not on the head line (`:334`), and `word` itself on the `--play` path, which passes no `Word` (`play_loop.go:262`). `findVisible` does `strings.Index(plain[from:], needle)`, which returns `0` for an empty needle, then `return cols[i]` — and `cols` is empty when the first rendered line is empty. Measured at HEAD:

```
Render(ParseEntry(""),   RenderOpts{Width: 80, Word: "somekey"})  → panic: index out of range [0] with length 0
Render(ParseEntry(" "),  RenderOpts{})                            → same
Render(ParseEntry("| notation |"), RenderOpts{})                  → same
```

The pre-fix `regionsIn` (walking `e.Head` tokens) returned zero regions for all three — I ran the same probe against `9469474:render.go` and it passes — so this is introduced by `3b60e1a`, the commit that closed the previous Critical. It is reachable on the ordinary lookup path: `selectedDictionary.Lookup` (`dict_darwin.go:230`) returns `(C.GoString(res), nil)` and does not require non-empty text, so a dictionary that answers with whitespace crashes `define <word>` rather than printing it.

The repo's own fuzz target already covers this surface and finds it immediately:

```
$ go test ./cmd/define/ -fuzz FuzzRenderLosesNothing -fuzztime 45s
--- FAIL: FuzzRenderLosesNothing (0.18s)   panic: index out of range [0] with length 0
    findVisible → regionsIn:333 → Render:236
crasher, minimized:  string(" ")
```

Fix sketch, at the owner rather than the caller: `findVisible` should refuse an empty needle (`if needle == "" { return 0, 0, false }`) — an empty span is not a thing on screen, which is the same argument that makes `Width` a display-cell count. That also removes the bogus zero-width `RegionHeadword` a non-empty first line currently produces on the `--play` path. Then commit the minimized crasher to `cmd/define/testdata/fuzz/FuzzRenderLosesNothing/`, which is the convention the target's own doc comment states and which this window already followed twice for `FuzzDecodeMouseIsBounded`.

The rule the family names: **a fix that gives an existing helper a new class of input inherits that helper's unstated preconditions, so the fix's own commit re-runs the property tests that cover it — not just the ones the finding named.** `go test ./...` cannot see this class; the fuzz target can, and it was not run.

## 3. Important findings

### I-1 · The atlas's "Clickable regions" section was not swept when the Critical fix changed what a region carries — `atlas/define.md:383`, `:409`

**This is the 4th finding in family `docs-lag-new-surface`** (after BR-25's M1 atlas lag, BR-39's whole-feature absence, and BR-50's missing issue Log). Earlier rounds fixed instances, and one produced a guard. Do **not** fix this instance alone.

Two concrete gaps, both introduced by `3b60e1a`, which touched no `atlas/` file:

- `atlas/define.md:383` still reads `regionsIn     (Entry, rendered) -> []Region`. The signature is `regionsIn(e Entry, rendered, key string)`.
- `:409` still explains `Region.Word` as "the entry it belongs to". The round's durable decision — *the click's target is the LOOKUP KEY, carried on `RenderOpts.Word`, owned by the caller, because a shortcut must not re-derive its target* — is recorded in the plan's Revisions and the issue Log and appears nowhere in the atlas, though `RenderOpts` gained a field and `Region.Word` changed meaning.

The interesting part is *why the guard did not catch it*. `TestAtlasDescribesEveryRegionKind` (added last round for BR-39) derives from `numRegionKinds` and therefore defends exactly one axis: the set of kinds. This boundary changed a different axis — a field's meaning and a function's signature — and no guard covers that, so the obligation fell back to a human sweep and was missed. The rule: **a derived docs guard defends only the axis it derives from; every other axis a boundary changes is still an owed sweep, and the enumerable form of that sweep is "for every Core-concepts row whose cell this window edited, re-read the atlas line that names the same entity."** Two rows changed in this window (`regionsIn`, `Region.Word`/`RenderOpts.Word`) and both have a stale atlas line — a 2/2 hit rate, which is why the enumeration is cheap and worth writing rather than repeating the sweep by memory. `README.md` is fine: the user-visible sentence ("Click the headword to hear it again") survives the change.

## 4. Minor findings

- **Stale comments, enumerated — 3rd in `stale-rationale`, and the enumeration is the deliverable.** BR-44 named two sites and is still open; this window added three more, so the class is measurable at five: `key.go:186` ("answers only the WHEEL") and `:198` ("a click therefore stays `KeyUnknown`") and `:344` ("Only the wheel is answered") all describe a decoder that now returns `KeyClick`, and `decodeWheel` is a misnomer; `render.go:287` says `Region.Word` is "the ENTRY this region belongs to … the word to play is still the headword", which is what `3b60e1a` stopped being true; `internal/llm/config.go:153` says "the parley proxy's key is four characters, so it is the common case here" in the commit that made that key `"parley-local"` (12 characters, so it takes the *other* branch). The rule: **a comment that states a fact about a value, or about what a function does not do, is part of that fact's blast radius — the commit that changes the fact sweeps every site stating it, and the sweep is a grep, not a recollection.**
- `cmd/define/editorloop_test.go:1004` — `keysOf` is added in this window and called from nowhere (`grep` finds only its own declaration). Dead scaffolding from an earlier draft of the joint test; Go will not complain about an unused function, so nothing else will catch it.
- `cmd/define/editorloop_test.go:982` — `frameCell`'s doc comment sits above `livePromptOf`, so both helpers are documented by the wrong paragraph.
- `cmd/define/editorloop_test.go:971` — when the joint test fails it says "timed out waiting for the stream", which names the instrument rather than the claim. Both mutations I applied produced that message; a reader would not learn that the click resolved to the wrong line. Worth reporting what was clicked and what it resolved to.

## 5. Test coverage notes

- Ran here at `eb85b10`: `go build ./...` and `go vet ./...` clean; `go test ./...` green; `go test -race ./cmd/define/ ./internal/...` green (118 s + 27 s); `go test -race -count=25` over the three click tests green, so `3b60e1a`'s flake fix holds. All 12 PTY rows skip in this environment.
- Mutations I ran, with results: `regionsIn`'s `word := key` → `e.Headword()` reddens three tests with the recorded messages ✓; `clickAt`'s `wireRow - 1` → `wireRow` reddens the joint test and `TestClickCarriesItsPosition` ✓; `visible()`'s `end - s.rows` → `+1` reddens the joint test ✓; **`screen.go` reverted to `9469474` (undoing the `markClickable` rewrite) leaves `go test ./cmd/define/` green** — the existing assertions all use `underlineOn+text`, which the doubled-emission form also satisfied, so the corrected placement is undefended.
- The gap C-1 fell through is the one the previous round already named in a different shape: the corpus property tests assert over *entries that exist*, and every new input class this milestone introduced (an empty key, an empty headword, an empty first line) is outside that corpus. `FuzzRenderLosesNothing` is the instrument that covers it and is not part of the close's verification recipe — `## Verification before close` lists `go test ./...` and the conformance tag only.

## 6. Architectural notes

- **ARCH-DRY — flag, carried.** BR-49 is unchanged: `visibleCells` (`render.go:466`), `visibleIndex` (`render.go:454`), `clipVisible` (`screen.go`) and `markClickable` (`screen.go:289`) still each re-spell the "step escapes, measure cells" traversal, and `markClickable`'s rewrite in this window re-spelled it a fourth time rather than consuming a shared iterator. Separately, `defaultLocalKey` is a constant duplicated across two repos (tools and parley) with no single source and no conformance check; the commit reasons about that explicitly and accepts a 401 as the failure mode, and `askrun_test.go:606` does pin that a 401 reaches the user carrying its cause — so this is a recorded trade, not a gap.
- **ARCH-PURE — flag, via C-1.** `regionsIn` and `findVisible` are pure and take exactly what they read, which is right; a pure function that panics on an input class is still a defect, and this one is now reachable from the production lookup path. BR-41's smaller wrinkle stands: `LineAt` → `visible()` → `clamp()` writes back `s.offset` under a table row that says PURE.
- **ARCH-PURPOSE — flag.** BR-46 was answered as a class rather than an instance (the ORIGIN region and the mark's extent were swept with the headword), which is the right shape. BR-47 was not: the rule "the enumeration must be of JOINTS, not entities" is now written in the plan's Revisions and *one* joint row was added, but the table is still organised by entity and BR-45 names a joint — `addRegions`' base/`Col` exchange across the partial-line boundary — that is still both unpinned and unenumerated. That is the instance fixed and the enumeration deferred, one more time.
- **ARCH-MOCK — pass with a note.** The terminal's stateful double gained `TestPTYWithoutMouseBehavesAsBefore`, and the in-process joint test now runs the click through production objects, which is the better of the two asks. Still no pty row sends a real mouse report and asserts playback, so the *encoding* half of the click is confirmed only in process.
- **ARCH-CONSTRAINTS — pass.** `regionsIn` adds at most two extra `findVisible` scans of line 0 per lookup, not per keystroke; `markClickable` still early-returns on a region-free line; the 16 ms throttle and trailing flush are untouched. The side-quest adds no work to any path — `Resolve` is still a pure function over a lookup and still does no startup probe, which the atlas re-states.

## 7. Plan revision recommendations

1. **`## Revisions` — "the empty span the key made possible."** Record that carrying the lookup key into `regionsIn` gave `findVisible` a caller that can supply an empty needle, that `Render` therefore panicked on a one-space entry, and that the guard belongs to `findVisible` (an empty span is not a thing on screen). Add a row to *M2 — what runs each row*: `findVisible`'s empty needle | `FuzzRenderLosesNothing` + the committed crasher | "revert the guard → `go test -fuzz` red in under a second on `\" \"`". Add `go test -fuzz FuzzRenderLosesNothing` to `## Verification before close`, since `go test ./...` cannot reach this class.
2. **`## Revisions` — the atlas axis the kind-guard does not cover** (I-1). Record that `TestAtlasDescribesEveryRegionKind` defends the set of kinds only, that `regionsIn`'s signature and `Region.Word`'s meaning changed in this window with no atlas edit, and the enumerable sweep that replaces the memory-based one.
3. **M2 Done-when row 1** still stops one layer short — it names `TestLiveScreenJoinsRegionsToTheLinesTheyWereRenderedFor` but not `TestAClickAtAPaintedCellPlaysWhatIsUnderIt`, which is now the row's strongest pin. (Round 10 recommended this and it landed in the enumeration table instead.)
4. **The `Region` bullet** under M2's Core concepts still describes the struct as `{Kind, Text, Lang, Line, Col, Width}`, omitting `Word` — the field the whole round turned on.
5. **`markClickable`'s enumeration row** records only "drop `underlineOff` → the headword is not marked as clickable". That mutation does not cover the placement property the code now claims; add the one that does, or record that the placement is currently unpinned (measured: reverting `screen.go` to `9469474` leaves the suite green).

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      M1.1 still enumerates the four cases verbatim and screen_test.go still has only FuzzScreenWriteDoesNotPanic — no chunk-boundary property over screen.Write.
  - id: BR-40
    disposition: not-addressed
    note: |
      Behaviour is corrected (measured at HEAD: one \x1b[4m, placed after the palette escape) but reverting screen.go to 9469474 leaves go test ./cmd/define/ green, so nothing defends it.
  - id: BR-41
    disposition: not-addressed
    note: |
      LineAt still reaches clamp through visible(), which writes back s.offset, while the plan's M2 table still labels the group PURE.
  - id: BR-44
    disposition: not-addressed
    note: |
      key.go:186, :198 and :344 unchanged, decodeWheel still the name; three more sites of the same class arrived in this window (see the stale-rationale finding).
  - id: BR-45
    disposition: not-addressed
    note: |
      addRegions still decrements base for a partial line and leaves Col relative to the render; screen_test.go:738 still supplies Col pre-offset by hand.
  - id: BR-46
    disposition: addressed
    note: |
      Verified by mutation: word := e.Headword() reddens three tests with the recorded hot dog / a priori / bargainer messages. Corpus probe at three widths shows every region's Word is the key, none zero-width, none overlapping.
  - id: BR-47
    disposition: addressed
    note: |
      Verified by two mutations on opposite sides of the joint (clickAt's 1-based conversion; visible()'s top line) — both redden the new test, which drives a real liveScreen as view and stdout. The by-joint regeneration of the table is only partial: BR-45's joint is still unenumerated.
  - id: BR-48
    disposition: not-addressed
    note: |
      The "the command menu closing" subtest is unchanged at screen_test.go:906.
  - id: BR-49
    disposition: not-addressed
    note: |
      No forEachCell exists; markClickable's rewrite in this window re-spelled the traversal a fourth time rather than consuming a shared iterator.
  - id: BR-50
    disposition: addressed
    note: |
      The issue now carries "2026-08-30 — M2: the clicks, and what the boundary found" with Region.Word, the writeRendered seam, the 24-not-0 decision and degrade-by-routing.
findings:
  - id: new
    severity: Critical
    family: helper-precondition-unguarded
    title: |
      Render panics on a one-space entry — regionsIn hands findVisible an empty needle and it indexes cols[0]
    detail: |
      Introduced by 3b60e1a. regionsIn now passes span STRINGS to findVisible, and
      two can be empty: e.Headword() when the key is not on the head line
      (render.go:334) and word itself on the --play path, which passes no Word
      (play_loop.go:262). strings.Index returns 0 for an empty needle, so
      findVisible (render.go:445) returns cols[0] on an empty cols slice.
      Measured: Render(ParseEntry(" "), RenderOpts{}) and
      Render(ParseEntry(""), RenderOpts{Word: "somekey"}) both panic; the same
      probe against 9469474:render.go passes, so it is new in this window. The
      repo's own FuzzRenderLosesNothing finds it in 0.18s and minimizes the
      crasher to a single space, while go test ./... stays green because it runs
      only the seed corpus. Reachable on the ordinary lookup path:
      selectedDictionary.Lookup returns (text, nil) without requiring non-empty
      text. Fix at the owner — findVisible refuses an empty needle, since an
      empty span is not a thing on screen — and commit the minimized crasher to
      testdata/fuzz/FuzzRenderLosesNothing/ per the target's own convention. The
      rule: a fix that gives an existing helper a new class of input inherits
      that helper's unstated preconditions, so its own commit re-runs the
      property tests covering it, not only the ones the finding named.
  - id: new
    severity: Important
    family: docs-lag-new-surface
    title: |
      The atlas's clickable-regions section was not swept when the Critical fix changed what a region carries
    detail: |
      4th in this family, so the deliverable is the rule, not the two lines.
      3b60e1a touched no atlas file: atlas/define.md:383 still prints
      "regionsIn (Entry, rendered) -> []Region" against a function that now takes
      the key, and :409 still explains Region.Word as "the entry it belongs to"
      while the round's durable decision — the target is the LOOKUP KEY carried
      on RenderOpts.Word, because a shortcut must not re-derive its target — lives
      only in the plan and the issue Log though RenderOpts gained a field. Last
      round's guard did not fire because TestAtlasDescribesEveryRegionKind derives
      from numRegionKinds and therefore defends the set of KINDS only; this
      boundary changed a different axis. The rule: a derived docs guard defends
      only the axis it derives from, and every other axis a boundary changes is
      still an owed sweep whose enumerable form is "for every Core-concepts row
      this window edited, re-read the atlas line naming the same entity". Two rows
      changed here and both have a stale atlas line.
  - id: new
    severity: Minor
    family: stale-rationale
    title: |
      Five comments now state facts their own commits made false, and the enumeration is the fix
    detail: |
      3rd in this family; BR-44 named two sites and this window added three, so
      prevalence is measurable at five. key.go:186 ("answers only the WHEEL"),
      :198 ("a click therefore stays KeyUnknown") and :344 ("Only the wheel is
      answered") describe a decoder that returns KeyClick, and decodeWheel is a
      misnomer. render.go:287 says Region.Word is "the ENTRY this region belongs
      to … the word to play is still the headword", which 3b60e1a stopped being
      true. internal/llm/config.go:153 says "the parley proxy's key is four
      characters, so it is the common case here" in the very commit that made
      that key "parley-local" — twelve characters, which takes the other branch.
      The rule: a comment stating a fact about a value, or about what a function
      does NOT do, is part of that fact's blast radius; the commit that changes
      the fact sweeps every site stating it, by grep rather than by recollection.
  - id: new
    severity: Minor
    family: dead-test-scaffolding
    title: |
      keysOf is added in this window and called from nowhere, and frameCell's doc comment sits above livePromptOf
    detail: |
      editorloop_test.go:1004 declares keysOf(map[int][]Region) []int; grep finds
      no caller. Left over from an earlier draft of the joint test — Go does not
      complain about an unused function, so nothing else will catch it. At :982
      the paragraph describing frameCell sits above livePromptOf, so both helpers
      are documented by the wrong comment.
```

---

## Re-review — 2026-08-30T15:00:49-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 30 — clickable regions in the terminal: click ORIGIN French to hear it, click the IPA to replay |
| repo | tools |
| issue file | workshop/issues/000030-clickable-regions.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | a6584e243f2a032d209f8f4f1f44c50ed801f0ad..b3ca133f94fce04d1b8e95bd11007828a0337967 |
| command | sdlc milestone-close --issue 30 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-08-30T15:00:49-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

Round 11's Critical is genuinely fixed and I verified it the hard way: reverting the `findVisible` guard in a scratch copy of `b3ca133` reddens `TestRenderSurvivesADegenerateEntry` with the exact original panic (`index out of range [0] with length 0` at `render.go:445`). I also re-verified round 10's two fixes by mutation (deriving `Word` from `e.Headword()` reddens `TestAClickAsksForExactlyWhatEnterAsksFor` on `hot dog`/`bargainer`/`a priori`; shifting `clickAt`'s 1-based conversion reddens both `TestClickCarriesItsPosition` and the joint test). Full suite green (107s), `-race` green over 3 runs of the four concurrent click tests, and 45s of fresh `FuzzRenderDoesNotPanic` found nothing. What holds SHIP back is that **round 11's *other* deliverable does not work**: `TestAtlasDescribesEveryRenderOpt` — the guard written specifically because `RenderOpts.Word` went undocumented — is measurably unable to fire for `RenderOpts.Word`, because its `` `Name` `` fallback disjunct is satisfied by an unrelated sentence about `Question` at `atlas/define.md:1916`. I deleted the atlas row, then the entire `## Clickable regions` section, and the guard stayed green for that field both times. That is the fourth `vacuous-pin` on this issue. Separately: eight prior Minors are still open and untouched, and the mechanism that would have caught BR-51 before a reviewer did (fuzzing) is not wired into anything the suite runs.

## 1. Strengths

- **`findVisible` is the right owner for the fix, and measuring in cells rather than bytes was the half the report missed.** `render.go:436` catches the NUL headword (width 0 for the same reason a combining mark's is) that a `needle == ""` check would have let through — and the committed crasher `testdata/fuzz/FuzzRenderDoesNotPanic/390ca22614d4ce1a` is that input.
- **The occurrence-index carry-across is correct on a case nothing explicitly tests.** I probed `ORIGIN … from Old French concret, later from French concret`: counting `nth` in the *unmasked* text (`render.go:361`) is what makes the region land on column 61 — the modern "French" — rather than on the one inside "Old French" at column 34. Subtle, and right.
- **`clipVisible` closes a clipped style with `sgrOff` (`screen.go:689`)**, so a `markClickable` underline cut by the width budget cannot bleed into the rows painted after it. The mark/clip ordering in `Paint` (`screen.go:399`) is safe because of it.
- **`visible()` returning frame and top line as one answer** (`screen.go:205`) is a structural fix rather than a hoisted call, and `Paint` and `RegionAtRow` provably cannot answer from different states.
- **`TestAClickAtAPaintedCellPlaysWhatIsUnderIt`** (`editorloop_test.go:901`) drives real objects end to end and reddens on either side of the joint — verified.

## 2. Critical findings

None.

## 3. Important findings

**(a) `TestAtlasDescribesEveryRenderOpt` cannot fire for the one field it was written for** — `cmd/define/doc_sync_test.go:273`.

The condition is `!strings.Contains(atlas, "RenderOpts."+name) && !strings.Contains(atlas, "`"+name+"`")`. Measured twice: delete `atlas/define.md:401` (the `RenderOpts.Word` row) → green; delete the whole `## Clickable regions` section → still green for `Word` (and for `Vocab`), failing only on `Color` and `Width`. The satisfying match is `atlas/define.md:1916`, a sentence about `Question`. `TestAtlasDescribesEveryRegionKind` (`:224`) has the same defect from the other direction: it searches for `k.String()`, and "headword" occurs 10+ times elsewhere in the atlas, so deleting the section left that kind green too.

**This is the 4th finding in family `vacuous-pin`.** Do not patch the two disjuncts. The rule: *a derived docs guard must search for a token that exists ONLY in the documentation it defends* — a qualified anchor (`RenderOpts.Word`, or a per-kind marker), never a bare name that ordinary prose can supply. Applied here that means dropping the `` `Name` `` fallback outright (all four fields already carry `RenderOpts.X`, so the suite stays green) and giving `RegionKind` a qualified anchor rather than its `String()`. Prevalence is now 3 derived docs guards, 2 of which I measured as unable to fire for their motivating case.

**(b) 15 fuzz targets and 12 pty rows execute in no automated run, and that is why BR-51 shipped.**

`go test ./...` runs fuzz targets against the seed corpus only; `grep -rn fuzz` finds nothing in `Makefile`, `Makefile.local`, `Makefile.workflow`, `scripts/`, or `.github/workflows/merge-check.yml`, and `scripts/merge-checks.d/` does not exist. BR-51 was found by a reviewer typing `-fuzz`, in 0.18s — and round 11's answer was to add a 15th target with the same property. The pty rows are the same rule from another angle: all 12 report `no pty available: operation not permitted` here, so `TestPTYWithoutMouseBehavesAsBefore` and `TestPTYMouseTrackingIsAskedForAndGivenBack` (Done-when 6 and 8) certified nothing in this review; only their in-process counterparts did, and those pass.

Family `unrun-test-surface`, new: *a target that runs only when a human remembers to invoke it is not part of the suite*. M1's review already applied half this rule (`handBack`/`onceHandBack` got in-process pins beside the pty rows) — the other half, a `make fuzz` with a bounded `-fuzztime` that CI or the close gate invokes, was never written. Cheapest honest disposition at this gate is a follow-up issue rather than work inside `#30`; say so explicitly rather than letting it sit.

## 4. Minor findings

- `atlas/define.md:401` claims `RenderOpts.Word` empty means *"no click map wanted"* — measured false: `Render(…, RenderOpts{Width: 80})` returns 2 regions with `Word` falling back to `e.Headword()`, which is exactly the shape BR-46 was filed against. `play_loop.go:262` is the only caller that relies on discarding them. Family `doc-overclaim` (2nd): the rule is that a doc stating a *guarantee* about a field names the code that enforces it, or states the actual behaviour.
- `editorloop_test.go:930–950` carries two successive drafts of the same "one run in five" paragraph — the frame-vs-stream explanation and the marker-keystroke explanation, both live. Folded into the BR-53 disposition below rather than raised separately.
- `regionsIn` calls `originText(e)` once per mention (`render.go:361`), and `OriginLanguageMentions`/`anyLanguageIn` each `regexp.MustCompile` per language per call (`origin.go:202`, `:250`). Per-lookup, not per-keystroke, so ARCH-CONSTRAINTS is unaffected — noting it only because `anyLanguageIn`'s comment claims the two "cannot disagree" while spelling the pattern twice.

## 5. Test coverage notes

- Mutation-verified this round: BR-51 (revert → panic returns), BR-46 (both `Word` and span mutations), BR-47 (`clickAt` off-by-one → joint test and unit test both red), BR-48 (pre-fix `screen.go` → `a resize taller` red, `the command menu closing` **green**).
- `screen.Write` still has no chunk-independence property — only the three hand-picked splits at `screen_test.go:32`. The repo already owns the shape (`FuzzHighlightWriterIsChunkIndependent`), which is what makes BR-1's suggestion concrete rather than stylistic.
- `TestAClickAsksForExactlyWhatEnterAsksFor`'s second assertion uses `strings.HasPrefix`, so it stayed green when I mutated the span to `e.Headword()` ("hot" is a prefix of "hot dog"). The word-identity half of that test is strong; the span half is not, and `TestTheClickableSpanIsTheWholeHeadword` is what actually catches it.

## 6. Architectural notes

- **ARCH-DRY — flag.** BR-49 stands: `visibleCells`, `visibleIndex`, `clipVisible`, `markClickable` are four spellings of one cell walk, and M2.5's splice made a fifth reading of it. Nothing was consolidated this window.
- **ARCH-PURE — flag.** BR-41 stands and I measured it: `s.offset = 999` survives `RegionAt` but becomes 15 after `Frame()` or `LineAt()`. Three sites label this PURE (`screen.go:186`, `:158`, and the plan's Core-concepts row). The mutation is deliberate post-BR-42; the label is what is wrong.
- **ARCH-PURPOSE — pass on the issue, flag on the round.** Every Done-when has a delivered mechanism and both consumers route through `#29`'s `replayInPlace`. The flag is the class axis: BR-52's guard is the instance-shaped answer wearing a rule's clothes.
- **ARCH-MOCK — pass.** Stateful fakes for the CDN, dictionary and player; the production `liveScreen` is driven in-process by the joint test; `dict_conformance_test.go` is the live drift check. The pty rows are the right seam, merely unreachable here.
- **ARCH-CONSTRAINTS — pass.** 16ms throttle with trailing flush, every frame component budgeted, click resolution is a map lookup, buffer uncapped with the reason recorded. Nothing in M2 adds fan-out or blocks the keystroke path.

## 7. Plan revision recommendations

1. **The Core-concepts table lists `numRegionKinds` / `RegionKind.String` twice** — plan lines 183 and 189, same entities, same file, two prose descriptions. Collapse to one row.
2. **The `screen.RegionAt / LineAt / visible / addRegions` row is labeled `PURE`** (plan line 185) while `visible` clamps and writes back `s.offset`. Re-label it, or split `visible` from the genuinely pure three, and record why the mutation is correct.
3. **`liveScreen.WriteRegions / RegionAtRow` and `regionWriter / writeRendered`** carry no PURE/INTEGRATION marker at all, only prose. Give them `INTEGRATION` so the table reads uniformly.
4. **Add a `## Revisions` entry for round 12** recording (a) that `TestAtlasDescribesEveryRenderOpt` was measured unable to fire for `RenderOpts.Word` and what replaced it, and (b) the disposition of the eight carried Minors — swept in the close commit, or moved to a follow-up issue by number. Eleven rounds with a growing untouched Minor tail is the ledger saying the enumeration is being deferred rather than written.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan line 132 still enumerates the four cases verbatim, and screen.Write still has no chunk-independence property (only three hand-picked splits at screen_test.go:32).
  - id: BR-40
    disposition: addressed
    note: |
      Measured: markClickable now emits "\x1b[1;36m\x1b[4mpotassium\x1b[24m…" — one underline, escapes stepped before the column trigger.
  - id: BR-41
    disposition: not-addressed
    note: |
      Measured: offset 999 becomes 15 after Frame() and after LineAt(); screen.go:186, :158 and the plan row all still say PURE.
  - id: BR-44
    disposition: not-addressed
    note: |
      key.go untouched since 251a85a; :186, :198 and :344 all still stale. Subsumed by BR-53.
  - id: BR-45
    disposition: not-addressed
    note: |
      screen.go:143 still decrements base without shifting Col; screen_test.go:739 still supplies Col: 12 pre-offset by hand.
  - id: BR-48
    disposition: not-addressed
    note: |
      Re-measured against 8f6a458:screen.go — "a resize taller" fails, "the command menu closing" passes. The row still does not earn its place.
  - id: BR-49
    disposition: not-addressed
    note: |
      No forEachCell exists; visibleCells, visibleIndex, clipVisible and markClickable still each spell the traversal.
  - id: BR-51
    disposition: addressed
    note: |
      Verified by revert: removing the findVisible guard reddens TestRenderSurvivesADegenerateEntry with the original panic at render.go:445. Cell-based rather than byte-based, and the NUL crasher is committed.
  - id: BR-52
    disposition: addressed
    note: |
      atlas:383 and :401 both swept and a derived guard added — but the guard cannot fire for RenderOpts.Word; raised separately as a vacuous-pin finding rather than re-raised here.
  - id: BR-53
    disposition: not-addressed
    note: |
      All five sites still stale (key.go:186/:198/:344, render.go:287, internal/llm/config.go:153), and a sixth: editorloop_test.go:930-950 carries two successive drafts of the same paragraph.
  - id: BR-54
    disposition: addressed
    note: |
      keysOf removed and the frameCell comment moved above frameCell.
findings:
  - id: new
    severity: Important
    family: vacuous-pin
    title: |
      TestAtlasDescribesEveryRenderOpt cannot fire for RenderOpts.Word, the field it was written for
    detail: |
      doc_sync_test.go:273 accepts a bare "`Word`" anywhere in the atlas, which
      atlas/define.md:1916 supplies in a sentence about Question. Measured twice:
      deleting the RenderOpts.Word row leaves it green, and deleting the whole
      "## Clickable regions" section leaves it green for Word and Vocab, failing
      only on Color and Width. TestAtlasDescribesEveryRegionKind (:224) has the
      same defect for "headword", which occurs 10+ times elsewhere in the atlas.
      4th in this family, so the deliverable is the rule: a derived docs guard
      must search for a token that exists ONLY in the documentation it defends —
      a qualified anchor, never a bare name ordinary prose can supply. Dropping
      the backtick fallback keeps the suite green, since all four fields already
      carry a qualified RenderOpts.X line.
  - id: new
    severity: Important
    family: unrun-test-surface
    title: |
      15 fuzz targets and 12 pty rows run in nothing automated, which is why BR-51 shipped
    detail: |
      go test ./... exercises fuzz targets against the seed corpus only, and
      there is no -fuzz invocation in Makefile, Makefile.local, Makefile.workflow,
      scripts/, or .github/workflows/merge-check.yml (scripts/merge-checks.d/
      does not exist). BR-51 was a reachable Critical panic that the repo's own
      fuzzer finds in under a second, found instead by a reviewer typing the
      flag — and round 11's answer added a 15th target with the same property.
      The 12 pty rows are the same rule from another angle: all report "no pty
      available: operation not permitted" here, so Done-when 6 and 8 were
      certified this round only by their in-process counterparts. The rule: a
      target that runs only when a human remembers to invoke it is not part of
      the suite. M1 already applied half of it by pinning handBack in process;
      the other half is a bounded `make fuzz` the close gate or CI invokes.
      Reasonably disposed as a follow-up issue rather than work inside #30 — but
      say which, rather than leaving it implicit.
  - id: new
    severity: Minor
    family: doc-overclaim
    title: |
      The atlas says an empty RenderOpts.Word means "no click map wanted"; measured false
    detail: |
      atlas/define.md:401. Render(ParseEntry(entry), RenderOpts{Width: 80}) returns
      2 regions, with Word falling back to e.Headword() at render.go:314 — which
      is precisely the shape BR-46 was filed against. Nothing enforces the stated
      guarantee; play_loop.go:262 is the only caller and it happens to discard the
      regions. Either enforce it (empty key => no regions) or state the actual
      fallback behaviour.
```

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

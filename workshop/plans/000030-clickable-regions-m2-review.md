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

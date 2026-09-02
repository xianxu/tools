# Boundary Review — tools#44 (whole-issue close)

| field | value |
|-------|-------|
| issue | 44 — the play frame's chrome: separate it, colour it, and stop it growing a line per click |
| repo | tools |
| issue file | workshop/issues/000044-play-chrome.md |
| boundary | whole-issue close |
| milestone | — |
| window | 33321b152e51730cd1ed92e06385b8f6290c8102..9f143d98a464b27931ab763c615d3eb8340c4c8b |
| command | sdlc close --issue 44 |
| reviewer | claude |
| timestamp | 2026-09-02T14:10:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All three operator complaints are genuinely fixed, and the frame arithmetic — the part where a wrong row means a permanent mark on the wrong word — is correct and independently verified: I re-derived `fitsABoard`'s no-charge equivalence proof by hand, confirmed `fitFooter` gets its budget before the gap so a whole board can never lose a row to it, and ran `go test ./...` (green), `go vet` (clean), `gofmt -l` (clean) and `go test -tags conformance ./cmd/define` (green, 135s). I mutation-tested every pin the issue claims: all four `asChrome` sites, the `grantedGap` sacrifice, the `Paint` gap emission, the reveal indicator, and the AST guard each go red when their code is removed. Three do not. Nothing here is a correctness bug — the blockers to a clean SHIP are that the production wiring `newPinnedScreen → gap: chromeGap` is pinned by **no test at all** (deleting the line leaves the entire suite *and* the conformance suite green), that the new footer-click test is tautological (it passes for any value of `footerTop`), and that `Paint`'s 35-line doc comment was silently reparented onto `const chromeGap`.

## 1. Strengths

- **`grantedGap` as one owner with a proved-equivalent second consumer** (`cmd/define/screen.go:466`, `cmd/define/play_loop.go:720`). Discovering mid-implementation that charging the gap in `fitsABoard` is a no-op, *not writing it*, and replacing the code with a proof plus `TestTheChromeGapNeverChangesWhetherABoardFits` (exhausting 0–40 × 1–30 × 1–5) is the right call. I re-derived it: `T−P−F ≥ 2 ⟹ F+P+1 ≤ T−1`, so the two predicates coincide at every shape. The `{T:8,F:7,P:1}` counterexample in the doc comment is real.
- **Deleting `playRegion`'s `ind` parameter instead of guarding it** (`cmd/define/replraw.go:583`). A click can only arrive in raw mode, so the parameter offered a choice with one right answer. Two supplying sites removed structurally beats two sites swept.
- **The AST guard is not decorative.** I reverted `play_loop.go:511` to `defaultIndicator(opt)` and it fired with the file:line. It also correctly fatals rather than skips when the file list or the match count is empty.
- **The dim test is anchored to the text, not the row.** `strings.Contains(line, "\x1b[2m"+text)` after `unstyled` is what makes it survive `Paint` reprinting the prompt onto the same `\n`-split line as the bar. I deleted each of the four styling sites in turn; each produced a distinct failure.
- **`TestSittingPlaybackCommitsNothingToTheBuffer` measures a difference**, running the same deck audible and silent so content is held fixed. Stable over `-count=5`. It failed at 547 vs 542 on revert.
- I checked the one regression risk the sweep introduces — `submitLine` lost `before: "\r\n"`, so an *open* partial line would be eaten whole by `eraseOpenLine`. Empirically `s.partial == false` after `lookupAndRender`, and the reveal path writes `"\n"+Reveal()+"\n"`. Not a defect.

## 2. Critical findings

None.

## 3. Important findings

**(a) `newPinnedScreen`'s gap is set at zero *tested* call sites — `cmd/define/screen.go:650`.**
Every gap test builds the screen by hand (`screen{pinned: true, gap: chromeGap}`). I deleted `l.s.gap = chromeGap` and ran the full package suite *and* `go test -tags conformance`: both green (only the four `repo_guard` tests fail, environmentally, because the scratch tree has no `main` branch). So the issue's headline Done-when — the operator's actual complaint, a blank row in a real `--play` frame — has no pin on the path production takes. Fix: have `TestAFullBufferStillLeavesARowAboveThePrompt` construct through `newPinnedScreen(&tty, 10, 40)`, or add a one-line assertion that `newPinnedScreen(...).s.gap == chromeGap`.

**(b) `TestAFooterClickIsUnmovedByTheChromeGap` is tautological — `cmd/define/screen_test.go:1394`.**
It queries `row := sc.footerTop + want` and asserts `FooterRowAt(row) == want`, but `FooterRowAt` is *defined* as `row - s.footerTop` indexed into `s.footer`. I set `footerTop` to `bufRows+promptRows` (the gap dropped) and to `bufRows+gap+promptRows+3` (nonsense): the test passed both times. The second assertion (`FooterRowAt(footerTop-1)` is not ok) is likewise unconditionally true. The property *is* covered — the pre-existing `TestFooterRowAtNamesTheEntryUnderAClick` goes red on the same mutation, in absolute terminal rows — but the test the plan designated as the pin for "the row of arithmetic that must not be wrong" pins nothing. Fix: assert against an absolute row, e.g. `sc.footerTop == termRows - len(footer)` for a pinned screen, or read the emitted frame with `readFrame` and check the first footer entry's text lands on `footerTop`.

**(c) `Paint`'s doc comment now documents `const chromeGap` — `cmd/define/screen.go:412`/`:447`.**
The `chromeGap` comment was inserted immediately after the last line of `Paint`'s doc block with no blank line, so Go attaches the whole 35-line block ("A frame is budgeted in DISPLAY ROWS, not in lines, and that distinction is the whole guarantee…") to the constant, and `Paint` — the function this issue changes — is left undocumented. Verified with `go doc -all -u`: `func (s *screen) Paint(...)` prints with no prose, and `const chromeGap = 1` prints "Paint draws one whole frame…". Fix: blank line before `// chromeGap is the rows…`, or move `chromeGap`/`grantedGap` above `Paint`'s doc block entirely.

**(d) `screenHostedFiles` is an opt-in enumeration, so a new screen-hosted file escapes the rule silently — `cmd/define/indicator_guard_test.go:18` (ARCH-PURPOSE).**
The list is complete today (I checked every `playAnnounced` call site: `main.go:720` is the one-shot path and `repl.go:316` is the piped loop, both correctly outside). But the issue's own thesis is that a sweep is not a fix, and the guard's *scope* is a sweep — a sixth screen-hosted file added in `editor.go` is invisible to it. The comment cites this package's purity guard as the precedent, and that guard is an **allowlist**: everything is in scope unless exempted. Inverting this one the same way (`nonScreenFiles = {"main.go", "repl.go"}`, skip `_test.go`, check every other file in package `main`) makes a new file default to the rule instead of out of it.

## 4. Minor findings

- `workshop/issues/000044-play-chrome.md:179` overstates the guard: after `playRegion`'s parameter was deleted it takes no `indicator`, so the guard checks exactly the four `playAnnounced` arguments — which a callee-name walk would also have reached. The by-type predicate is still the better rule (it generalises to a future forwarder), but "a callee-name walk would have missed it" describes the pre-deletion tree, not this one.
- `cmd/define/README.md:148` — "The bottom two rows are chrome" is true of an ordinary sitting but not of a board, and the paragraph sits in the board section, directly under a code fence that shows the prompt row *above* the grid. On a board the dimmed rows are the first and last of the live edge with the grid between them.
- `cmd/define/play_loop_test.go:3648` hardcodes `"\x1b[2m"` rather than reading `newPalette(true).dim` — a second speller of a sequence the palette owns (ARCH-DRY, cosmetic here since it fails loudly).
- `cmd/define/play_loop.go:222` — `written = s.Index` is now unconditional in the `Grid` arm. Correct (idempotent), just worth noting the guard that used to wrap it is gone.

## 5. Test coverage notes

Mutation results, all run against a scratch checkout of `9f143d9`:

| mutation | caught by |
|---|---|
| `gap := 0` in `Paint` | `TestAFullBufferStillLeavesARowAboveThePrompt` ✓ |
| `grantedGap` returns `want` unconditionally | `TestTheChromeGapIsGivenUpBeforeTheFrameOverflows` (17 shapes) + `TestTheChromeGapNeverChangesWhetherABoardFits` ✓ |
| `footerTop` drops `+gap` | `TestFooterRowAtNamesTheEntryUnderAClick` ✓ (**not** the new test) |
| reveal takes `defaultIndicator(opt)` | behaviour test + AST guard ✓ |
| each of the 4 `asChrome` sites removed | `TestTheChromeBandIsDimmedTogether` ✓ (4/4, distinct subtests) |
| `newPinnedScreen` drops `gap` | **nothing** ✗ — full suite *and* conformance suite green |

`TestTheEditorsFrameReservesNoGap` is genuinely non-vacuous (a gap of 1 on `screen{}` would shift `line 39` off row 8). `TestALongRevealPagesRatherThanScrollingTheWordAway` moving 8→9 rows is a premise repair, not a papering-over — the test's own premise assertion still holds. New tests are stable over `-count=5`.

## 6. Architectural notes

- **ARCH-DRY — pass.** `chromeGap`, `grantedGap`, `screenIndicator`, `asChrome` each replace a spelled-out rule with one owner, and `playRegion`'s deleted parameter removes two of the five drifted sites structurally. The one duplication (`asChrome(sittingBar(fig), pal)` at both `Draw` seams) is forced by keeping `sittingBar` plain for the README pin, and is correct.
- **ARCH-PURE — pass.** `grantedGap`, `asChrome`, `chromeGap`, `Paint` are pure; every test runs against `strings.Builder`/`bytes.Buffer` with no mocks. The palette resolves once per sitting at the `main`-owned seam rather than per frame.
- **ARCH-PURPOSE — flag, finding (d).** The class (the indicator rule) is enforced rather than swept, which is the right instinct — but the guard's *file scope* is itself a hand-maintained enumeration, which is the same shape one level up.
- **ARCH-MOCK — pass.** Playback goes through the `fakePlayer` seam; the pty conformance suite is the live conformance check and it is green, including `TestAClickOnABoardMarksIt` and `TestANarrowingResizeKeepsTheBoardsClickMapHonest`.
- **ARCH-CONSTRAINTS — pass.** Keystroke path: gap arithmetic is O(1) per frame, escapes are skipped by every measuring helper so no budget moves, and the frame provably never exceeds `termRows` (swept over 1–12 rows × 2 prompt widths × 3 footer shapes). The AST guard parses ~50 files per run at ~60 ms.

For upcoming work: `#42` reworks this same fit arithmetic, and the one thing it must inherit is that `fitFooter` is budgeted *before* `grantedGap` — that ordering is the whole reason the gap can never cost a board a row. It is stated in `fitsABoard`'s comment and in the atlas but is not pinned by a test; if `#42` reorders those two lines nothing goes red.

## 7. Plan revision recommendations

The plan already carries the `fitsABoard` revision and `TestPlanTableStatusMatchesTheChangeWindow` is green, so the Core-concepts table matches the code — every row verified present at its stated path (`chromeGap`, `screen.gap`, `grantedGap` at `screen.go`; `asChrome` at `playbar.go`; `newPinnedScreen`/`playRegion` modified as described; the guard test present). Two entries to add:

- **Task 2 Step 2** — record that the click-map test as written is tautological and that the property is carried by the pre-existing `TestFooterRowAtNamesTheEntryUnderAClick`; state the replacement assertion so the plan stops claiming a pin it does not have.
- **Task 2 Step 4** — the plan says "In `newPinnedScreen`: `l.s.gap = chromeGap`" but never names a test that drives the gap through the constructor. Add the wiring pin to the step, since that is the line the operator's complaint actually depends on.

```findings
findings:
  - id: new
    severity: Important
    family: pin-must-fail-without-the-code
    title: |
      newPinnedScreen's `gap: chromeGap` is pinned by no test — deleting it leaves the full suite and the conformance suite green
    detail: |
      Every gap test builds `screen{pinned: true, gap: chromeGap}` by hand, so the
      production wiring at cmd/define/screen.go:650 is unreachable from any
      assertion. Verified by deleting the line in a scratch checkout: `go test
      ./cmd/define` and `go test -tags conformance ./cmd/define` both pass. Have
      TestAFullBufferStillLeavesARowAboveThePrompt construct through
      newPinnedScreen, or assert newPinnedScreen(...).s.gap == chromeGap.
  - id: new
    severity: Important
    family: pin-must-fail-without-the-code
    title: |
      TestAFooterClickIsUnmovedByTheChromeGap is tautological — it passes for any value of footerTop
    detail: |
      cmd/define/screen_test.go:1394 queries `sc.footerTop + want` and asserts
      FooterRowAt decodes it, which is FooterRowAt's definition (`row -
      s.footerTop`). Mutating footerTop to `bufRows+promptRows` and to
      `bufRows+gap+promptRows+3` left it green both times. The property is covered
      by the pre-existing TestFooterRowAtNamesTheEntryUnderAClick, which does go
      red — but the designated pin for "the arithmetic that must not be wrong"
      pins nothing. Assert an absolute row instead.
  - id: new
    severity: Important
    family: doc-attaches-to-the-wrong-decl
    title: |
      Paint's 35-line doc comment was reparented onto `const chromeGap`, leaving Paint undocumented
    detail: |
      The chromeGap comment was inserted at cmd/define/screen.go:447 with no blank
      line after Paint's doc block, so Go attaches the whole block ("A frame is
      budgeted in DISPLAY ROWS…") to the constant. Confirmed with `go doc -all
      -u`: Paint prints with no prose. Insert a blank line, or move
      chromeGap/grantedGap above Paint's doc block.
  - id: new
    severity: Important
    family: sweep-every-site-of-the-rule
    title: |
      the indicator guard's screenHostedFiles is an opt-in list, so a new screen-hosted file escapes the rule silently
    detail: |
      cmd/define/indicator_guard_test.go:18 enumerates the files the rule applies
      to. The list is complete today (main.go is the one-shot path, repl.go the
      piped loop, both correctly exempt), but the guard's own comment cites this
      package's purity guard as precedent and that one is an allowlist: in scope
      unless exempted. Invert to `nonScreenFiles = {"main.go", "repl.go"}` over
      package main's non-test files so a sixth site defaults into the class.
  - id: new
    severity: Minor
    family: cite-the-code-you-claim
    title: |
      the issue's guard Done-when overstates what the by-type predicate reaches
    detail: |
      workshop/issues/000044-play-chrome.md:179 says the guard "also reaches the
      two sites that call playAnnounced through playRegion — which a callee-name
      walk would have missed". After playRegion's parameter was deleted it takes
      no indicator, so the guard checks exactly the four playAnnounced arguments,
      all of which a callee-name walk would also have reached. The by-type rule is
      still the better one; the sentence describes the pre-deletion tree.
  - id: new
    severity: Minor
    family: cite-the-code-you-claim
    title: |
      README's "the bottom two rows are chrome" is placed in the board section, where it is not true
    detail: |
      cmd/define/README.md:148 sits under the board paragraphs, but on a board the
      dimmed rows are the prompt (drawn above the grid, as the fence at line 106
      shows) and the bar below it — not the bottom two. Accurate for an ordinary
      sitting; reword or relocate.
  - id: new
    severity: Minor
    family: one-owner-per-quantity
    title: |
      the chrome-band test hardcodes the dim escape rather than reading it from the palette
    detail: |
      cmd/define/play_loop_test.go:3648 spells "\x1b[2m" where newPalette(true).dim
      is the owner. Cosmetic — it fails loudly rather than silently — but it is a
      second speller of a sequence the palette exists to own (ARCH-DRY).
```

---

## Re-review — 2026-09-02T14:45:42-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 44 — the play frame's chrome: separate it, colour it, and stop it growing a line per click |
| repo | tools |
| issue file | workshop/issues/000044-play-chrome.md |
| boundary | whole-issue close |
| milestone | — |
| window | 33321b152e51730cd1ed92e06385b8f6290c8102..7b4383c611ffcb1662a228bceeb07257f16e155b |
| command | sdlc close --issue 44 |
| reviewer | claude |
| timestamp | 2026-09-02T14:45:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The frame arithmetic is correct and I verified it independently rather than reading it: `bufRows + gap + promptRows + footerRows == termRows` holds at every reachable shape (the `s.rows < 0` clamp cannot fire while `gap > 0`, because `grantedGap` requires two rows of slack), `LineAt` already refuses the gap row so a click on it maps to nothing, and `clipVisible` already hands the style back at a cut so the dim cannot leak past a clipped prompt. All nine prior findings are genuinely addressed, and I confirmed the four Important ones by mutation rather than by reading the commit message: deleting `l.s.gap = chromeGap` reddens the full-buffer test (BR-3), dropping `+gap` from `footerTop` reddens the click test (BR-4), `go doc` now prints `Paint` with its own prose and the widened guard catches a const-owner reparent (BR-5), and a new screen-hosted file with a bad indicator reddens the guard (BR-6). Deleting each of the four `asChrome` sites in turn reddens `TestTheChromeBandIsDimmedTogether`, and restoring `before: "\n"` reddens the leak test at 547 vs 542 lines. `go test ./...`, `go vet ./...` and `go test -tags conformance ./cmd/define` (137s) are all green. What stops this being SHIP is a second round of the same two families: the BR-5 guard was widened on the owner side but not the neighbour side and I reproduced the escape; two documented behaviours in this window survive mutation with the whole suite green; `grantedGap`'s doc still describes the design the plan's own `## Revisions` retracted; and the durable plan is 0-of-28 ticked at the close boundary, which silently exempts two repo guards from checking it.

## 1. Strengths

- **The gap is a frame row, and the sacrifice order is right.** `screen.go:504` takes the gap *after* `fitFooter` has had its full `termRows-promptRows` budget, so a border can never cost a board a grid row — and `TestTheFooterIsBudgetedBeforeTheGap` (screen_test.go:1466) pins that ordering unprompted, which is the single most valuable thing here for `#42`.
- **The `fitsABoard` revision is the right call and was made for the right reason.** Charging the gap there is provably a no-op, and writing it would have *looked* like a reconciliation. `TestTheChromeGapNeverChangesWhetherABoardFits` turns the proof into a pin over 41×30×5 shapes.
- **Deleting `playRegion`'s `ind` parameter instead of guarding it.** Two of the five drifting sites became structurally unable to be wrong. That is the correct answer to a "five sites disagree three ways" defect, and the by-type guard covers the rest.
- **`readFrame` was widened to carry row *content*.** Placement assertions now read the frame the way a terminal does instead of substring-searching it; `frameGeometry.row` (screen_test.go:339) is what makes "the row above the prompt is empty" answerable at all.
- **The chrome-band test anchors the escape to the text** (`strings.Contains(line, dim+text)`, play_loop_test.go:3650) rather than to the row — which is what stops `Paint`'s prompt reprint from making the board's case vacuous.

## 2. Critical findings

None.

## 3. Important findings

**I1 — `cmd/define/repo_guard_test.go:1736` — the BR-5 guard was widened on the owner side only, and the escape is reproducible.**
This is the 2nd finding in family `doc-attaches-to-the-wrong-decl`. `documentedDecl` (repo_guard_test.go:1713) now accepts single-spec const/var as a doc *owner*, but `declaredIn` still collects only `FuncDecl` names, so a block whose first word is a **const or var** name is invisible. Verified in a scratch checkout: inserting `func inserted() {}` between `// boardRefusal is the prompt row…` and `const boardRefusal` leaves `go test ./cmd/define -run TestADocCommentNamesWhatItSitsOn` green, while `go doc -all -u` prints `func inserted` wearing boardRefusal's prose and `boardRefusal` with none — exactly the BR-5 failure, mirrored.
**Do not fix the instance; fix the rule:** the owner set and the neighbour set must be the same set of declaration shapes. Derive `declaredIn` from `documentedDecl` (walk `f.Decls`, take `name, _ := documentedDecl(d)`, ignore the doc) so widening one widens both by construction.

**I2 — `cmd/define/play_loop.go:220`, `cmd/define/screen.go:436` — two claims in this window survive mutation with the whole suite green.**
This is the 3rd finding in family `pin-must-fail-without-the-code` (BR-3, BR-4 were the first two, and both were fixed as instances). Measured prevalence, 2 live:
- Done-when 6 ("the board's blank-buffer-line special case is gone"): re-adding `if written != s.Index { written = s.Index; fmt.Fprintln(stdout) }` to the grid arm leaves `go test ./cmd/define` green (only the four git-dependent guards fail, and only because the scratch repo has no history). The regression it would ship — a blank in the exit transcript plus two blank rows above a grid on a full buffer — is caught by nothing.
- `grantedGap`'s documented rule "granted only when a buffer row survives beside it": relaxing `>= want+1` to `>= want` leaves the entire suite green. The exhaustion test cannot see it (the fit-equivalence holds for *any* threshold ≥ `want`) and the overflow test cannot either (the frame still totals `termRows`, just with zero content rows).

**State the rule rather than patching these two:** a Done-when row is not tickable, and a threshold constant is not landable, until the reverting mutation has been *shown* red. The enumeration is finite and already written down — the issue's eight Done-when rows plus every constant/threshold the window introduces (`chromeGap`, `grantedGap`'s `+1`). Sweep that enumeration this round and record the mutation used per row in `## Log`, as the dim row already does.

**I3 — `cmd/define/main.go:847` — `screenIndicator()`'s contract now depends on an invariant stated nowhere and pinned by nothing.**
`eraseOpenLine` takes back the whole **open** line. With no `before`, an indicator written while the buffer is mid-line joins that line, and the erase deletes the caller's content with it. Demonstrated directly against `screen`: `Write("a\nb-no-newline")`, then `Write("  ♫ playing 3×")`, then `Write(eraseLine)` leaves `Lines() == ["a"]` — the definition's last line gone. `submitLine` previously passed `before: "\r\n"`, which closed the line first; that defence is now removed. Not reachable today (I checked all four sites: `Render` always ends in `"\n"`, the reveal writes `"\n"+…+"\n"`, and `replraw.go:545` writes `"\r\n"` before `submitLine`) — but the guard now *forces* every future screen-hosted site into this shape, and nothing tells the next author about the precondition. Fix: state it in `screenIndicator`'s doc ("the caller must not be mid-line; `eraseOpenLine` takes the whole open line") and pin it with a `screen`-level unit test on the swallow, so the day a caller does leave a partial line the failure has a name.

**I4 — `cmd/define/screen.go:426` — `grantedGap`'s doc describes the design the plan's own `## Revisions` retracted.**
This is the 4th finding in family `cite-the-code-you-claim`. The block says it is "ONE owner because **TWO consumers** ask: Paint when it draws, and fitsABoard when it decides whether Enter may spend a board" — but `git grep grantedGap` shows exactly one production consumer (`screen.go:504`), and `fitsABoard`'s own doc at `play_loop.go:699` now says the opposite ("THE CHROME GAP IS NOT A TERM HERE"). Two doc blocks in the same window contradict each other about this issue's central design decision, and the stale one reads as an instruction to add the term back. Two more measured instances of the same class in this window: `indicator_guard_test.go:31` justifies exempting `repl.go` as "the piped loop keeps the indicator as a record", but `repl.go:317` passes `indicator{}` (`show:false`) so *nothing* is written — the record form is `defaultIndicator` on a non-tty, which is main.go's site; and `play_loop.go:220`'s "`written` still tracks the board so the non-grid arm below can tell a new question from a redraw" is false, since the non-grid arm sets `written` itself and deleting the line reddens nothing.
**Do not fix the three sites; fix the rule:** a plan `## Revisions` entry is not complete until `git grep <entity>` is clean of comments still stating the superseded design. The enumeration *is* that grep — three entities were revised here (`fitsABoard`, `grantedGap`, `playRegion`), and one grep per entity would have caught all three instances above.

**I5 — `workshop/plans/000044-play-chrome-plan.md` — 0 of 28 checkboxes ticked at the close boundary, which switches two repo guards off.**
22 of the 25 archived plans in `workshop/history/plans/` are fully ticked; this one is fully unticked. That is not only a record about to be archived claiming no step was done — `TestPlanTablesNameEntitiesThatExist` skips every `new` row while `inProgress` (repo_guard_test.go:772) and `TestPlanNamedTestsExist` skips the whole document (repo_guard_test.go:917), so this plan's Core-concepts table and its named tests were never checked. I ticked them in a scratch copy and both guards pass, so **nothing substantive is hidden** — but the guards were nonetheless off for the window they exist to cover. Tick the plan, and strike Task 2 Step 5 ("Charge it in the board's fit") rather than ticking it, since the `## Revisions` entry retracted it.

## 4. Minor findings

- **M1** — 2nd finding in family `sweep-every-site-of-the-rule`: the indicator class got a source-level guard; the dim class did not. `asChrome` is applied by hand at four sites (`play_loop.go:220`, `:248` ×2, `:679`), and a fifth `view.Draw` in the sitting would ship undimmed chrome silently. Prevalence is 2 `Draw` sites today, both pinned by mutation. The rule, if it is worth stating: every string handed to the sitting's `Draw` as prompt or footer-bar is `asChrome`'d — checkable the same way the indicator guard is (argument position of `view.Draw` within `playSession`). Note for `#42`, which adds draw paths.
- **M2** — `play_loop.go:220`: `written = s.Index` in the grid arm is dead (deleting it reddens nothing, and indices only move forward). Harmless as an invariant, but its comment claims necessity — see I4.
- **M3** — the plan's Core-concepts table lists `screenIndicator` under **Integration points**; it is a pure constructor with no IO. Cosmetic classification only.

## 5. Test coverage notes

Coverage is strong where it was attacked last round and thin exactly where it was not. Confirmed red by mutation: the gap wiring, `footerTop`'s `+gap`, all four dim sites, the indicator's `before`, and a new screen-hosted file joining the guard. Confirmed green (i.e. unpinned) by mutation: the board's blank-line deletion, `grantedGap`'s `want+1` threshold, and the grid arm's `written`. `TestTheChromeGapIsGivenUpBeforeTheFrameOverflows` still builds `screen{pinned: true, gap: chromeGap}` by hand — acceptable, since it is about overflow rather than wiring and two sibling tests go through `newPinnedScreen`, but worth a one-line comment saying so, since BR-3's rule is now written down. `TestSittingPlaybackCommitsNothingToTheBuffer` measuring audible-vs-silent as a *difference* is the right shape and is why it can be sharp. The conformance suite is the only end-to-end proof `footerTop` still maps a real terminal's click, and it passes — `#37` remains the open risk that nothing runs it automatically.

## 6. Architectural notes

- **ARCH-DRY — pass, with I4.** `asChrome` and `grantedGap` are single owners; the dim escape now reads from `newPalette`; the five-way indicator drift is collapsed to one constructor plus a deleted parameter. The failure is documentary, not structural: `grantedGap`'s doc advertises a second consumer it does not have (I4).
- **ARCH-PURE — pass.** `chromeGap`, `grantedGap`, `asChrome`, `screenIndicator` are pure; every new test runs against `screen`/`Paint` with a `strings.Builder` or `bytes.Buffer` and no mocks. `Paint` remains the thin seam and `liveScreen` the only tty-touching part.
- **ARCH-PURPOSE — pass on the three operator complaints; flag on one class.** All three are delivered end to end, and the indicator's class got a guard rather than a sweep, which is the issue's own thesis honoured. The dim's class did not (M1), and the `## Revisions` sweep was not carried to the comments (I4) — instance fixed, class left.
- **ARCH-MOCK — pass.** The audio binary sits behind `fakePlayer` at the same seam production uses (`audible(&d, &opt)`), the dictionary behind the store fake, and the real-terminal behaviour behind the pty conformance suite, which I ran and which passes. Production flow and test flow share the `playAnnounced`/`screen` boundary.
- **ARCH-CONSTRAINTS — pass.** `Paint` is the keystroke path; `grantedGap` adds one comparison and the gap loop is O(1) at `chromeGap = 1`. No unbounded fan-out. The exhaustion test is 6,150 pure iterations and the overflow sweep 72 paints — the package suite is 107s, dominated by pre-existing work, and the conformance suite 137s. The one envelope claim the code makes and does not enforce is `grantedGap`'s "a buffer row survives beside it" (I2).

## 7. Plan revision recommendations

- **`## Revisions` — "grantedGap has one consumer, not two".** The 2026-09-02 entry retracted `fitsABoard`'s charge but the plan's Core-concepts bullet still says `grantedGap` is "**Relationships:** 1:2 — `Paint` asks it when drawing, `fitsABoard` when deciding whether Enter may spend a board", which the code no longer does. Append a delta correcting the relationship to 1:1 and noting that the DRY rationale now rests on the *equivalence proof* rather than on a shared call, so the row stops claiming what `screen.go:504` alone delivers. Same entry should record the `git grep grantedGap` sweep (I4).
- **Task 2 Step 5 — mark struck, not done.** It reads as an unticked instruction to write the charge that `## Revisions` retracted; strike it in place (`- [x] ~~Step 5: Charge it in the board's fit~~ — see ## Revisions`) when the plan is ticked (I5), so the archived artifact does not preserve a live instruction to reintroduce the term.

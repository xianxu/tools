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

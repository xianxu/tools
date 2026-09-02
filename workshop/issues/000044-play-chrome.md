---
id: 000044
status: codecomplete
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours: 2.16
started: 2026-09-02T12:50:12-07:00
actual_hours: 4.60
---

# the play frame's chrome: separate it, colour it, and stop it growing a line per click

## Problem

Three findings from one operator sitting, 2026-09-02, all on the same surface:
the band at the bottom of a `--play` frame where the record stops and the
controls begin.

**1. Nothing separates the action row from the form.** `Paint` stacks buffer
rows, then the prompt, then the footer, with no gap — so
`1-4 = pick the definition, d = remove from deck, Ctrl-C to stop` butts directly
against the last line of the definition it belongs under. On a short entry the
pinned buffer's padding fakes a gap; on an entry that fills the screen there is
none, and the two read as one block. Operator: *"there should be a blank line
between the action footer and forms."*

**2. The action row is the same weight as the text above it.** Both are plain,
unstyled ASCII, so nothing marks where the thing you READ ends and the thing you
PRESS begins. Operator: *"'1-4 = pick the definition ...' should be colorized to
make the border clear."* Every other surface in this program is already coloured
— the definition, the highlighted deck words, the board's marks — which makes
the chrome the one unstyled thing on screen.

**3. Playback grows the buffer by one blank line, every time.** Operator:
*"after clicking on pronunciation in the daily play, one additional line's
inserted"*, with a screenshot of a sitting whose frame had drifted up.

**A click is where the operator SAW it, not where it lives.** The plan gate
(PQ-1/PQ-2) found the same defect on the sitting's ordinary reveal playback
(`play_loop.go:507`) and in the editor's own submit path (`replraw.go:653`,
`before: "\r\n"`, which `screen.Write` normalises to `"\n"` and then commits). So
it is one blank line per PLAYBACK — per answered question in a sitting, per
looked-up word in the editor — and a click only made it repeatable on one word.
The first draft of this issue said "one per click" and claimed the editor
"already knows this"; both were generalisations from the one site that had been
looked at.

The mechanism, traced:

- `defaultIndicator` (`main.go:824`) is `{show: true, before: "\n", erase: eraseLine}`.
- `playAnnounced` writes `ind.before`, then `  ♫ playing 3×` with NO trailing
  newline, then `ind.erase`.
- `screen.Write` splits on `eraseLine` and calls `eraseOpenLine`, which drops the
  **open** line only — deliberately, since "a completed line is scrollback".
- So the erase takes back the indicator's own partial line, and the `"\n"` that
  preceded it stays as a committed blank line. One per playback, forever.

`before: "\n"` is cursor positioning, which is what it meant on a cooked
terminal. **Inside a screen a newline is CONTENT**, and the buffer is
append-only.

**Five call sites write playback into a screen, and they disagree three ways**
— which is the actual defect, the blank lines being its symptom:

| site | passes | |
|---|---|---|
| `play_loop.go:356` — click | `defaultIndicator(opt)` | commits a blank |
| `play_loop.go:507` — reveal | `defaultIndicator(opt)` | commits a blank |
| `replraw.go:653` — submit | literal, `before: "\r\n"` | commits a blank |
| `replraw.go:343` — click | literal, no `before` | correct |
| `replraw.go:633` — `/pron` | literal, no `before` | correct |

Three copies of a literal and two of a constructor meant for a different
surface, with nothing naming the rule they are all instances of (ARCH-DRY). The
comment at `play_loop.go:356` even states the wrong one — `defaultIndicator` is
"what every other playback on this path already uses" — which is true of the
one-shot path and false of every screen.

## Spec

**One owner for "the chrome is separated from the record above it, and looks
like chrome."**

### The gap is a RESERVED ROW in the frame, not a newline in a string

The obvious move — prefix `"\n"` to the prompt — is wrong twice over, and both
are the frame arithmetic this program has already been burned by:
`Paint` measures the prompt with `displayRows(prompt, cols)`, which counts
VISIBLE CELLS and knows nothing about an embedded newline, so a two-line prompt
would be charged one row and the frame would be one row too tall — the terminal
scrolls, and every row the app believes it placed moves. And `Paint` writes the
prompt with a bare `WriteString`, where in raw mode a `\n` moves down without
returning the carriage.

So the gap is a row the frame RESERVES, in the one accounting that already
budgets every component:

- `s.rows` (the buffer's share) gives the row up, exactly as it already gives
  rows to the prompt and the footer;
- the gap is emitted after the buffer rows and before the prompt;
- **`s.footerTop` includes it**, or every footer click lands one row out — this
  is the part that must not be got wrong, because on a board a mis-mapped click
  marks the WRONG WORD and the mark is irreversible;
- `fitsABoard` charges it, so a board is still only offered when it can be drawn
  whole.

The cursor walk-back (`footerRows+promptRows-1`) is unaffected: the gap is above
the prompt, and the walk-back only climbs from the footer to the prompt's first
row.

**It belongs to the SITTING's screen, not to every screen.** `newPinnedScreen`
already exists for exactly this distinction — "the two surfaces want opposite
things and the difference should be visible where the screen is BUILT" — so the
gap is set there, beside `pinned`, and named for what it is rather than folded
into `pinned`'s meaning. The editor's frame is untouched: its prompt is the line
you are TYPING, a continuation of what is above it, where the play frame's prompt
is a legend of what you can press. Different things, and only one of them wants a
border. (If the operator wants the editor to adopt it too, it is one line at the
other constructor.)

**This deletes the board's special-case blank.** `show()` writes one blank buffer
line the first time a board is drawn (`play_loop.go`, guarded by `written !=
s.Index`) because "a board writes nothing else to the buffer, so without it the
grid begins immediately under the previous question's last line". That is this
same gap, discovered once for one form and paid for out of the append-only
buffer. A reserved frame row makes it every form's, and makes it a row that
cannot end up in the exit transcript.

### The style goes on the strings; the plain text is untouched

`gradePrompt(q)` and `sittingBar(f)` stay PLAIN — `README.md` quotes the prompt
lines verbatim and `TestREADMEQuotesThePromptsTheLoopActuallyPrints` pins that,
so styling them in place would either break the pin or push escape sequences into
the README. The dim is applied where the strings are handed to `Draw`.

Escapes cost no columns and every measuring helper here already skips them
(`visibleCells`, `clipVisible`, `displayRows`), so styling changes no arithmetic.
The one pin that must be re-read is
`TestTheRefusalRowIsNoWiderThanTheKeysRow`, which compares two prompt strings and
must keep comparing VISIBLE width.

### The chrome is the prompt row AND the bar

Dimmed together, as one band. Dimming only the action row would leave the figure
line below it brighter than the controls above it, which inverts their
importance — the bar is a number you glance at, the action row is what you press.
Recorded as a decision the operator can reverse in a sitting: if the bar should
stay bright, it is a one-line change.

Through `newPalette`'s existing `dim`, from `main`, because `main` owns the
terminal's colours — the same seam `boardPalette` sits on. Derived from
`opt.color` rather than assumed: `--play` refuses `-no-color` (BR-3), so this is
belt, but a rig that runs colourless is exactly how #40's wrap Critical stayed
invisible, so the styled path is what the tests must drive.

### The indicator inside a screen has no `before` — and a guard says so

All five sites take one named constructor, `screenIndicator()`. The screen is
where the rule "a newline is content, not cursor movement" is true, so that is
what the name says. `defaultIndicator` keeps `before: "\n"` for the one-shot and
piped paths, where the cursor genuinely does have to move.

**A sweep is not a fix — the rule gets a guard.** Five sites drifted three ways
precisely because nothing enforced them; correcting three of them by hand leaves
the sixth free to be written wrong (ARCH-PURPOSE: the deliverable is the CLASS,
not the instances). So a source-level test enumerates every `playAnnounced` call
in the screen-hosted files and fails on one that does not pass
`screenIndicator()` — the same shape as this package's existing purity guard (an
import allowlist plus a wall-clock grep) and its doc-sync tests, and the reason
those keep working where hand-maintained enumerations did not.

## Done when

- [x] A frame whose buffer fills the screen still shows one blank line between the last content row and the action row, pinned by a frame test that reads the placement rather than searching for a substring. — `TestAFullBufferStillLeavesARowAboveThePrompt`, over an overfull buffer so the pinned padding cannot fake the gap; `readFrame` was widened to carry row CONTENT so it reads placement. `TestTheEditorsFrameReservesNoGap` is the other half.
- [x] The action row and the bar carry the dim style in a coloured sitting — at BOTH bar sites, the board's and the common one — and neither carries an escape sequence when the palette is off. — `TestTheChromeBandIsDimmedTogether`, three rows. **Verified by deleting each styling site in turn and watching it fail**, which is how two holes were found: the board's bar was uncovered, and the predicate "this line contains a dim" passed on the PROMPT's dim, because `Paint` reprints the prompt on the same `\n`-split line as the bar.
- [x] ~~The gap is CHARGED where a board is offered~~ and SACRIFICED FIRST where a frame is drawn, pinned at both boundary heights. — **Revised: it is not charged, and the proof is the deliverable.** `grantedGap` hands the row out only when there were two rows spare, so `T-P-F >= 2` gives `F+P+1 <= T-1`: charging `fitsABoard` alters no answer while looking like a reconciliation, and at `{T:8, F:7, P:1}` a naive charge refuses a board `Paint` draws whole. `TestTheChromeGapNeverChangesWhetherABoardFits` exhausts 0–40 × 1–30 × 1–5. Sacrifice: `TestTheChromeGapIsGivenUpBeforeTheFrameOverflows` sweeps heights 1–12 × two prompt widths × three footer shapes. See the plan's `## Revisions`.
- [x] A sitting that plays audio N times leaves the buffer the height it was — driven through the REVEAL path as well as the click path. — `TestSittingPlaybackCommitsNothingToTheBuffer` runs the same sitting audible and silent and compares buffer heights, so content is held fixed and playback is the only variable; it failed at 547 vs 542 lines over five questions. **Stated precisely: the behavioural test drives the REVEAL only**; the click path is covered by the guard below and by `playRegion` no longer taking an indicator at all.
- [x] No `playAnnounced` call in a screen-hosted file passes anything but `screenIndicator()`, enforced by a test rather than by having swept the five that exist today. — `TestEveryScreenPlaybackTakesTheScreenIndicator`. It matches on the ARGUMENT'S TYPE and its file scope is an ALLOWLIST (`nonScreenFiles`), so a new screen-hosted file defaults into the rule rather than out of it (BR-6). **Stated precisely:** with `playRegion`'s parameter deleted the tree now has only direct `playAnnounced` arguments, which a callee-name walk would also have reached — the by-type predicate is the better rule because it survives the next forwarder, not because it catches something today. The earlier claim that a callee walk "would have missed the reported site" described the PRE-deletion tree.
- [x] The board's own blank-buffer-line special case is gone, and a board still reads as separated from the question above it. — deleted from `show()`; the separation is now the frame's, for every form.
- [x] `README.md` still quotes the prompt lines verbatim and the doc pin still passes — the plain text is unchanged. — `TestREADMEQuotesThePromptsTheLoopActuallyPrints` green; `gradePrompt`/`sittingBar` untouched.
- [x] `go test -tags conformance ./cmd/define` passes: the pty suite's SGR-1006 click is the only end-to-end proof that `footerTop` still maps a real terminal's click to the intended cell. — green, 133s.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Design is light throughout because the plan doc resolved the decisions — the
`design-buffer` is 0.15 rather than 0.30 for exactly that reason. The two
expensive rows are the frame budget (an arithmetic change on the surface where a
wrong row means a permanent mark on the wrong word) and the test churn behind it:
`newPinnedScreen` has ~30 test call sites and the gap moves every placement
expectation among them.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: cross-cutting-refactor   design=0.15 impl=0.14
item: smaller-go-module        design=0.1  impl=0.2
item: tui-screen               design=0.4  impl=0.36
item: cross-cutting-refactor   design=0.05 impl=0.16
item: smaller-go-module        design=0.05 impl=0.12
item: atlas-docs               design=0.1  impl=0.06
item: milestone-review         design=0.0  impl=0.14
design-buffer: 0.15
total: 2.16
```

| row | what it is |
|---|---|
| `cross-cutting-refactor` #1 | the five-site indicator sweep + deleting `playRegion`'s parameter |
| `smaller-go-module` #1 | the AST guard, matching on argument type |
| `tui-screen` | `chromeGap`: `Paint`'s budget, the sacrifice order, `footerTop`, `fitsABoard` |
| `cross-cutting-refactor` #2 | the shifted placement expectations across the pinned-screen tests |
| `smaller-go-module` #2 | `asChrome` at its four sites |
| `atlas-docs` | README (derived block regenerated) + the atlas's screen section |
| `milestone-review` | the one boundary review at close |

## Plan

Durable design: `workshop/plans/000044-play-chrome-plan.md`.

Single-pass: one review boundary, so plain checkboxes rather than `Mx` tags.

- [x] The indicator inside a screen has no `before` — `screenIndicator()`, shared by the sitting's click path and the editor's, and a test that clicks TWICE.
- [x] The frame reserves `chromeGap` between the record and the live edge — `Paint`'s budget, its `footerTop`, and `newPinnedScreen`; pinned by a placement test on a FULL buffer and a footer-click test.
- [x] ~~`fitsABoard` charges the gap~~ — proved equivalent to not charging it, and pinned by exhaustion instead (see the plan's `## Revisions`).
- [x] The board's blank-buffer-line special case is deleted, its reasoning now the frame's.
- [x] `asChrome` dims the action row and the bar together, leaving `gradePrompt`/`sittingBar` plain so the README pin still holds.
- [x] README + atlas follow; the derived board block is regenerated rather than hand-edited.

## Log

### 2026-09-02
- 2026-09-02: closed — go test ./... green; go vet + gofmt clean; go test -tags conformance ./cmd/define green (131s) — one live-model row (TestReflectAgainstTheLiveService) flaked once and passed on re-run in isolation and in full; non-deterministic by construction, unrelated to this issue paths, noted in Log. Rounds 1-3 all findings fixed at the RULE and mutation-verified (code reverted, named test watched red). Round 3: BR-10 the revision sweep now runs over the TREE not one directory (it had left atlas/define.md stating both retracted designs, incl. playRegions deleted indicator parameter described as a deliberate difference between the loops); BR-11 the chrome guard inverted from a hardcoded filename to nonSittingDrawFiles, the same allowlist shape BR-6 required of its sibling.; review verdict: FIX-THEN-SHIP

Filed mid-planning on `#42`, from three operator messages during one sitting.
Sequenced AHEAD of `#42` on the operator's call: the blank line changes the row
budget (`fitsABoard` charges the prompt's measured height) and `#42` reworks
exactly that arithmetic, so doing chrome first means the fit math is written once
against the final chrome. `#42` is parked `blocked` on this.

### 2026-09-02 — what the work turned out to be

Three operator observations, one of which was a much larger bug than it looked
and one of which was smaller.

**The stray line was the big one.** Filed as "one per click"; it is one per
PLAYBACK — every answered question in a sitting and every lookup in the editor —
across five screen-hosted sites that disagreed three ways. The plan gate found
that (PQ-1, PQ-2) before any code was written. What closed it was not the sweep
but two structural moves: `playRegion`'s indicator parameter is DELETED (a click
can only happen inside a screen, so the parameter offered a choice with one right
answer), and the guard matches on the ARGUMENT'S TYPE rather than a callee name —
the callee-name version, which the plan proposed first, was blind to exactly the
site the operator reported (PQ-7).

**`fitsABoard` did not need changing.** The plan had it charging `chromeGap`;
working the arithmetic through, that is provably a no-op, and writing it would
have looked like a reconciliation while doing nothing. The proof plus an
exhaustive pin replaced the code. Recorded in the plan's `## Revisions`, and
caught as a stale row by `TestPlanTableStatusMatchesTheChangeWindow`.

**Two vacuous tests, both found by breaking the code rather than by reading.**
The chrome-band test covered one of the two bar sites, and its predicate — "this
line carries a dim escape" — was satisfied by the PROMPT's dim, because `Paint`
walks the cursor back and reprints the prompt on the same `\n`-split line as the
bar. Deleting each styling site in turn is what surfaced both. The lesson is the
one this repo already knows in another form: a guard has to be shown failing.

**One test moved rather than broke.** `TestALongRevealPagesRatherThanScrollingTheWordAway`
ran an 8-row terminal; the gap costs the buffer a row, so the entry's tail no
longer reached `DERIVATIVES`. Nine rows restores the premise the test is about
("several screenfuls") rather than papering over it.

Estimate 2.16h. The estimate-quality judge flagged (INFO, non-blocking) that no
`ux-rename-iteration` row was booked on a change whose whole subject is how the
thing LOOKS — fair, and the issue's own provenance is an operator iteration
round. Left as filed so the close-time ledger scores the real miss.

### 2026-09-02 — boundary review round 1: four blockers, all about the TESTS

Verdict FIX-THEN-SHIP. No correctness finding — the frame arithmetic was
independently re-derived and held. Every blocker was a pin that did not pin,
which is the more useful kind of finding here.

- **BR-3 — `newPinnedScreen`'s `gap: chromeGap` was pinned by nothing.** Every
  gap test built `screen{pinned: true, gap: chromeGap}` by hand, so deleting the
  production wiring left the full suite *and* the conformance suite green — on
  the issue's headline Done-when. The class: **a test that constructs the object
  by hand does not test the wiring that constructs it in production.** Both gap
  tests now go through `newPinnedScreen`, and the mutation goes red.
- **BR-4 — the footer-click test was a tautology.** It asked
  `FooterRowAt(sc.footerTop + i) == i`, and `FooterRowAt` *is* `row - footerTop`;
  it passed with the gap dropped and with a nonsense `footerTop`. It now asserts
  an ABSOLUTE row (`termRows - len(footer)`, from D3a's bottom-edge rule) and
  reads the drawn frame back, so it can fail.
- **BR-5 — `Paint`'s 35-line doc block was reparented onto `const chromeGap`.**
  `go doc` printed the constant with "Paint draws one whole frame…" and left
  `Paint` undocumented. Fixed by moving the constants above the block — and the
  CLASS is that `TestADocCommentNamesWhatItSitsOn` only walked `FuncDecl`, so a
  block reparented onto a const was invisible to the guard written for exactly
  this failure. Widened to single-spec const/var, and **it immediately found two
  pre-existing instances** (`unstyled`/`sgr`, `builtBinary`/`builtBinaryOnce`),
  both fixed.
- **BR-6 — the indicator guard's file scope was itself a swept enumeration.**
  Naming the two screen-hosted files meant a third would silently escape the
  rule — this issue's own thesis, one level up. Inverted to an allowlist
  (`nonScreenFiles`), so a new file defaults into the rule and each exemption
  carries its reason; the guard also fails if an exemption stops naming a real
  file.

Minors also taken: the dim escape now reads from `newPalette`, the README
sentence moved out of the board section (where "the bottom two rows" is false),
and the issue's own claim about the by-type guard was corrected — post-deletion,
a callee-name walk would reach the same four arguments, so the predicate is
better because it survives the NEXT forwarder, not because it catches one today.

Also pinned unprompted, because the review named it as the one thing `#42` must
inherit and nothing went red if it were broken: `TestTheFooterIsBudgetedBeforeTheGap`
holds that `fitFooter` gets its budget before `grantedGap` does — the ordering
that is the whole reason a border can never cost a board a grid row.

### 2026-09-02 — boundary review round 2: the same two families, one level up

Round 2 confirmed all four round-1 blockers fixed, by its own mutation runs
rather than by reading the commit. It then found five more, and the useful thing
about them is that four are the SAME TWO FAMILIES as round 1 — the fix had been
applied to the instance and not to the rule.

- **I1 — `doc-attaches-to-the-wrong-decl`, 2nd.** The doc guard was widened on the
  OWNER side (`documentedDecl` accepts consts) and not the NEIGHBOUR side
  (`declaredIn` still collected only functions), so a block whose first word is a
  CONSTANT's name stayed invisible — the same failure mirrored. Worse, my first
  attempt derived `declaredIn` from `documentedDecl`, which requires a doc — and
  the declaration being caught is precisely one that has just LOST its doc. Split
  `declName` (shape only) from `documentedDecl` (shape + doc), so the two sides
  widen together by construction. Verified by inserting a function between a
  const's doc and the const.
- **I2 — `pin-must-fail-without-the-code`, 3rd.** Two claims survived mutation with
  the suite green: Done-when 6 (the board's deleted blank line) and `grantedGap`'s
  `>= want+1` threshold. Both now pinned —
  `TestABoardWritesNothingToTheBuffer` (driven LIVE, measuring while the board is
  on screen, because reading after the sitting counts `finish`'s summary) and
  `TestTheChromeGapNeverTakesTheLastContentRow` (pure, over `grantedGap` itself,
  because reading it back out of a frame also measures `s.rows`'s clamp).
- **I3 — an unstated precondition.** `screenIndicator` carries no `before`, so the
  caller must not be mid-line: `eraseOpenLine` takes the whole open line and would
  delete the caller's text with the indicator. No shipped site violates it, but
  the guard now FORCES every future screen site into this shape, so the
  precondition came with it and was written nowhere. Stated on the constructor and
  pinned by `TestAnIndicatorAfterAPartialLineSwallowsIt`.
- **I4 — `cite-the-code-you-claim`, 4th.** `grantedGap`'s doc still advertised the
  second consumer the `## Revisions` entry had retracted — which reads as an
  instruction to add the term back. Two more of the same class beside it. The RULE
  is now recorded in the plan: a revision is not complete until `git grep
  <entity>` is clean of comments stating the superseded design, one grep per
  revised entity. Running it found all three.
- **I5 — the plan was 0-of-28 ticked**, which silently switched off two repo
  guards that skip in-progress plans. Ticked; Task 2 Step 5 STRUCK rather than
  ticked, so the archived artifact does not preserve a live instruction to
  reintroduce the retracted term.
- **M1 —** the indicator class got a guard and the dim class did not, so a fifth
  `view.Draw` would ship undimmed chrome silently — and `#42` adds draw paths.
  `TestEverySittingDrawPassesChromeThroughAsChrome` closes it.
- **M2 —** the grid arm's `written = s.Index` was dead, with a comment claiming
  necessity. Deleted.

Every fix above was mutation-verified: the code was reverted and the named test
watched to go red.

### 2026-09-02 — boundary review round 3: both new rules failed on first use

Nine findings disposed; two new, and both are the rules written in round 2
failing the first time they were applied. That is the useful shape of this round.

- **BR-10 — the revision sweep ran over `cmd/`, not the tree.** The rule I had
  just recorded says "a `## Revisions` entry is not done until `git grep <entity>`
  is clean"; I ran it over the directory I was editing. `atlas/define.md` was
  still stating BOTH retracted designs — that `grantedGap` has two consumers, and
  that `playRegion`'s indicator is "its one parameter, because the editor's is
  erasable and a sitting's is the record-shaped `defaultIndicator`". That second
  one is the bug written down as a design note, in the document whose whole job is
  telling the next reader how this works. Rule amended in the plan: **over the
  tree**, and the amendment says why.
- **BR-11 — the chrome guard hardcoded one filename**, which is the inclusion-list
  shape BR-6 had made me invert for its sibling — written one screen away from the
  comment explaining why that shape is wrong. Inverted to `nonSittingDrawFiles`,
  so a new file with a `Draw` call defaults into the rule; `replraw.go` is the one
  exemption, because the editor's prompt is content being typed rather than chrome.

Both mutation-verified. The pattern across three rounds is one lesson, not three:
**a fix applied at the site of the finding is not the fix** — and that applies to
rules about rules, which is where rounds 2 and 3 both landed.

### 2026-09-02 — a flaky conformance row, noted not fixed

`TestReflectAgainstTheLiveService` failed once during the round-3 verification and
passed on re-run, in isolation and in the full suite. It calls the real model and
asserts over generated prose, so it is non-deterministic by construction — nothing
to do with this issue's paths (the frame, the indicator, the chrome). Recorded
because a conformance suite that fails at random is a gate that will eventually be
ignored, which is `#19`'s shape one seam over. Not filed: it is pre-existing and
outside this issue's scope.

### 2026-09-02 — close: the two demoted findings, fixed before committing

The gate converged at round 5 and demoted two Importants past the round cap with
the warning that no later gate picks them up. Both were fixed before the close
commit, per the FIX-THEN-SHIP protocol (#174).

- **BR-17 — the two AST guards were a hand-copied pair, and the copy had already
  lost a Fatal.** ~35 verbatim shared lines, and where the original fatals with
  "package main parsed to no files; this guard would certify nothing", the copy did
  `pkgs["main"].Files` and nil-dereferenced — a diagnosis turned into a panic.
  ARCH-DRY, on the very artifact this issue spent five rounds arguing for.
  Extracted `scanPackageMain(t, exempt, visit)`; both call it, and `#42` wants a
  third. Both guards re-verified by mutation afterwards.
- **BR-18 — five rounds produced six families and two new rules, and
  `workshop/lessons.md` had nothing.** The rules lived only in the plan's
  `## Revisions`, which is archived at close and which `AGENTS.md` §2 tells the
  next agent not to read — so their entire value, which is to the NEXT issue, was
  being thrown away. Three entries added, and the third records that fact as its
  own lesson.

---
id: 000080
status: working
deps: []
github_issue:
created: 2026-09-20
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T16:48:56-07:00
flow: {kind: quick, provenance: inferred, spec: "1f554c7f", done: "37b5d6fb"}
---

# define: click a cloze option's number to answer it

## Problem

Operator, 2026-09-20:

> create a task to support click to select from the cloze answer choices, by
> clicking on the [1] [2] numbering. the words themselves have underscore so
> clicking means to pronounce.

A cloze question is answered from the keyboard only: press `1`–`4`. The mouse is
already live on that same prompt — each option word is underlined and clicking it
**pronounces** it — so a learner with a hand on the mouse has no way to *answer*
with it, and the one thing the pointer can do on those rows is the thing that must
not answer.

So the gesture is split by target: **the number answers; the word speaks.** ("Underscore"
in the request is read as the underline `markClickable` splices under a deck word.)

What exists today, so the change is measured against it:

- The option rows are printed `1  keel` — `optionLine` (`play/choice.go:195`) is the
  digit and two spaces, **no brackets**. `[1]` in the request is taken as shorthand
  for "the number", not a glyph to add; whether to draw brackets is a design
  question below.
- The option *words* are `RegionWord` regions from `deckSpans` (`deckwords.go`),
  found by walking the same text that decides colour. A click on one plays the word
  through `playRegion`; `TestClozeOptionsAreClickableButNotColoured` pins that they
  are clickable and uncoloured.
- The digit is not in any region, so a click on it is *ordinary text* — nothing,
  by design ("A click on ordinary text is NOTHING: no beep, no message").
- Answering by pointer exists for exactly one form. `formCell` (`play_loop.go:819`)
  offers a click to the question first and only a `play.Grid` (the board) takes it,
  becoming `InputMark`. Everything else declines and falls through to `playRegion`.
  The atlas states the invariant: *a click never answers a form that did not ask
  for it* (D8).

## Spec

Not designed yet — the task as filed, with the hazards the code makes visible. A
brainstorm comes before a plan.

### The behaviour

Clicking the number (and the gap after it, see below) on an option row of an
**unanswered** cloze question answers it exactly as pressing that digit does: same
verdict, same record, same reveal. One act, two ways in — the loop's own words for
the board's mark.

Clicking the option **word** is unchanged: it pronounces and never answers.

### Hazards, the reason this needs a design rather than a one-line region

1. **A stale digit must never answer the current question.** Click regions are
   never pruned from the buffer: an earlier question's rows are still on screen and
   still in the click map (`screen.go:59`). A digit-region that means "answer the
   current question with option 2" would let a click on last question's `2` answer
   this one. The region has to carry the identity of the question it belongs to,
   and the loop has to check it against the *current, unanswered* one.
2. **A click after the answer must not advance the sitting.** `Apply` treats *any*
   `InputRune` on a graded question as "any key = next word" (`session.go:271`). If
   the click is forged as the digit key, a click on **any** number still on screen
   after the answer — the question's own option list above the reveal, and, after a
   wrong pick, the reveal's own `you chose` / `2  word` line
   (`RevealPresentation`) — skips the reveal the learner was reading. Post-answer, a
   click on a number must do nothing, like a click on ordinary text.
3. **The pointer is less precise than a key, and an answer is not undoable.** A
   1-cell target on a digit is easy to miss by a column and hit the word (which
   plays audio, harmlessly) — or, with an over-wide target, to hit the *wrong row's*
   number and record a real Wrong against the schedule
   (`schedule.Progress` is per word). Acceptance: the target is the digit **plus the
   two-space gap** (columns 0–2), which never overlaps the word's span; it never
   extends onto the word. Whether that is enough, or the digit should be drawn more
   clickably (brackets, like the request's `[1]`), is a decision for the brainstorm.
4. **Wrapping moves the map.** A stem that wraps changes which screen row each
   option lands on; `liveScreen.WriteRegions` re-points regions against its own
   column count and drops a region whose line wrapped. Option rows are one short
   line each, so the digit region survives a wrap of the *stem* — but this needs a
   test at a width that wraps the stem, not an assumption.

### Design fork

- **(A) A new region kind** (`RegionOption`-shaped) produced beside `RegionWord`,
  carrying the option index and the question's identity, resolved by the same
  `RegionAt`. Fits the existing registry, its guards (`numRegionKinds`,
  `TestEveryRegionKindIsActionable`, `TestEveryRegionKindIsNamed`) and coordinates
  in buffer terms — which is where a cloze prompt lives, unlike the board's footer
  cells.
- **(B) Widen the `formCell` seam** to a smaller capability than `Grid` (say, "which
  option is at this line and column"), answered by the question itself, which is
  the only thing that knows where it drew its numbers.

The tension: (A) reuses the click map but has to *smuggle question identity* through
a region; (B) keeps identity structural (the loop asks the *current* question, so a
stale digit cannot exist) but the cloze prompt is written through `writePrompt`
(`play_loop.go:1318`) into the buffer, while `formCell` resolves against footer
entries — the board is the footer's tenant — so its footer arithmetic does not
apply as-is. Recommendation to test first
in the brainstorm: **(B)'s ownership with (A)'s coordinates** — the form answers "is
this line/column my number, and which", from buffer-relative coordinates. That
makes hazard 1 unrepresentable rather than checked.

Either way the click reaches `Apply` as a distinct input, **not** a forged `InputRune`
— the atlas's own argument for `InputMark` (a machine that cannot tell a key from a
pointer, on the one surface where the difference matters) and hazard 2's rule.

### Open questions

1. **Cloze only, or every numbered form?** Multiple-choice (form 2.3, `Choice`)
   shares `optionSet`, `optionLine` and `Grade`; its options are glosses, so its
   *words* are not clickable regions at all, and a click on its number is even less
   contested. The request names cloze. Recommend: cloze in scope; `Choice` included
   only if the capability is on `optionSet` and costs no extra code — otherwise a
   follow-up issue, not a quiet widening.
2. **Does the prompt say so?** The board's prompt row states what a click does
   ("click or key marks"). A cloze's hint line may need "click a number or press
   1–4" — and if so it is subject to the width tests on the prompt row.
3. **Interaction with #75.** #75 makes a click on a *stem* word **mark** it
   (unknown-word gesture). With this issue a cloze prompt would mean three things by
   position: stem word → mark, option word → speak, number → answer. Each is
   distinct by target, but they should be designed as one map, not discovered as three
   PRs. Sequence with #75 or land it with the region kinds already disjoint.

### Out of scope

- Any change to what the option *word* click does.
- Drag selection over options; a drag never becomes a click.
- Keyboard behaviour: `1`–`4` stays exactly as is, and stays the only way to answer
  in `-raw` or a pipe, where there is no mouse.

## Done when

Each row names the test that pins it.

- Clicking the digit of option *k* on an unanswered cloze answers it with option
  *k* — same verdict and same recorded review as pressing that digit, asserted
  through the pty conformance shape the board's click uses
  (`TestPTYPlayBoardIsDrawnAndClickable`), not by a unit test of a handler alone.
- Clicking the option **word** still plays it and does not answer — the existing
  `RegionWord` rows stay green untouched.
- A click on the gap between the digit and the word answers; a click on the word's
  first column pronounces. The boundary is asserted, one column each side.
- A click on a previous question's number answers **nothing** and does not move the
  current question — with a fixture where two questions are on screen and the old
  `2` is under the pointer.
- A click on any number after the question is answered — the option list, or the
  reveal's `you chose` line after a wrong pick — does nothing and does not advance.
- A stem that wraps at the test width leaves every option's number clickable, or
  drops its region (never mis-points it) — asserted at a width that forces the wrap.
- The click survives colour off, matching `TestClicksSurviveNoColour`, and no
  mouse mode is requested where none was before (`-raw` and a pipe stay
  keyboard-only).
- Every region kind (or seam) this adds is named and actionable
  (`TestEveryRegionKindIsNamed`, `TestEveryRegionKindIsActionable`, or the
  capability's equivalent), and `play` still imports no terminal notion
  (`purity_test.go`).
- `atlas/define.md` restates the invariant *a click never answers a form that did
  not ask for it* with cloze as the second asker, and records the number-answers /
  word-speaks split and why the target excludes the word.

### Design (brainstorm, 2026-09-23)

Operator decisions: **brackets** — options draw `[1] keel`, the number not
underlined (the underline stays "click = pronounce"); **multiple-choice
included** — it shares `optionSet`/`optionLine`, so it is the same code, not a
widening; **a hint** — the keys line says `1-4 or click = pick …`.

The fork resolves to #67's precedent, which already solved hazard 1 for passages:
**a region for the coordinates, an absolute buffer range for identity.**

- **Producer (pure, `play`).** `promptBuilder.option` records each option's
  number span (`[k] `, byte offsets + index) as it writes it, the way `help`
  records help lines — one producer of "where the numbers are" (ARCH-DRY), so
  no consumer re-parses the prompt. Carried out on `Presentation` beside
  `Spans`. `optionLine` becomes `[k] ` and `OptionIndent` 4; `isOptionLine`
  (`playbar.go`) derives its prefix test from `optionLine` rather than
  restating the shape.
- **Region.** New `RegionOption` kind, `Region.Option` = index (set on that
  kind only, like `Lang`). `writePrompt` turns the presentation's option spans
  into regions and passes them with the rest to `writePracticePresentation`, so
  `renderPracticeOutput`'s existing wrap re-pointing covers them (hazard 4).
  `regionUnderlines(RegionOption)` = false. `regionPlaysAudio` generalises to a
  declared action per kind (audio / mark / answer) so
  `TestEveryRegionKindIsActionable` keeps deriving its set from
  `numRegionKinds` and checks each kind in the loop that owns its action.
- **Identity (loop).** The sitting records the buffer range its current prompt
  was written into (`BufferLines()` before/after `writePrompt`, reset on each
  new `written`). `formCell` stays the one "does this click answer the form"
  decision and learns the second shape: a `RegionOption` hit whose absolute
  `hit.line` is inside the current range → `InputMark{Cell: option}`; any other
  hit (an older question, the reveal's `you chose [2] …`) is ordinary text.
- **Session (ARCH-ORDER).** `Apply(InputMark)` on a `Graded` question →
  `OutcomeNone` (hazard 2 — refused in the machine, not only by the loop's
  range). The case asks a new `Marker{ Mark(i) (Verdict, bool) }` capability,
  which `Grid` embeds; `optionSet.Mark(i)` is the pick `Grade` already makes
  for digit i (Grade calls it), so key and click cannot disagree on verdict,
  `chosen`, or `unaidedNow`. State between events is exactly `Session`
  (`Index`, `Revealed`, `Graded`) plus the loop's prompt range, which is keyed
  to `written` and replaced whenever a new prompt is written.
- N/A: ARCH-MOCK (no external dependency), ARCH-SECURE (input is the local
  terminal's mouse report, already parsed by `pointer.resolve`), ARCH-FUNERAL
  (regions already live for the buffer's life; nothing new persists),
  ARCH-CONSTRAINTS (one keystroke-path lookup, O(regions on a line)).

**#75** has no plan yet. Its gesture targets stem words; this targets
`[k] ` on option rows — disjoint lines, so the three-way map (stem → mark,
option word → speak, number → answer) holds by target. Noted on #75 at close.

**Risk to measure, not assume:** `1-4 or click = pick the word, ? = bad
question, d = remove from deck, Ctrl-C to stop` is 84 columns — it wraps at 80.
The prompt row is measured (`displayRows`), so it is correct either way; if the
wrap is judged ugly, shorten the reserved tail, not the hint, and say so in the
Log.

## Plan

- [x] brainstorm — (A)/(B) fork, open questions 1–3 (above)
- [x] check #75's plan for the cloze click map — none yet; disjoint by target
- [x] tests first (red): stale digit on screen answers nothing; post-answer
      click on the option list and on the reveal's `you chose` line does
      nothing and does not advance; boundary pair (`[k] ` last column answers,
      word's first column pronounces); wrapped stem at a narrow width; colour off;
      key-vs-click same verdict + same recorded review (pty,
      `TestPTYPlayBoardIsDrawnAndClickable` shape) for cloze and multiple-choice
- [x] `play`: `optionLine` → `[k] `, `OptionIndent` 4, option spans on
      `Presentation`, `Marker` + `optionSet.Mark`, `Apply` Graded refusal;
      fix `isOptionLine`, option-wrap tests, README option blocks
- [x] loop: `RegionOption` (+ String/identifier/actions registry), regions
      from `writePrompt`, prompt range, `formCell` second shape
- [x] keys hint `1-4 or click = pick …`; README quotes; width check
- [x] atlas `define.md`: second asker of "a click never answers a form that did
      not ask for it", number-answers / word-speaks, why the target excludes the
      word; then `sdlc close`

## Log

### 2026-09-20

Filed from an operator request the same session as #79. Read before writing:
`play/cloze.go`, `play/choice.go` (`optionLine`), `play/presentation.go`,
`play/session.go` (`InputMark`, the any-key-advances rule), `play_loop.go`
(`formCell`, the click branch), `deckwords.go` (`wordRegions`), the atlas's
*sitting words are clickable* and *a click never answers* sections, and
`TestClozeOptionsAreClickableButNotColoured`.

The two hazards were found by reading, not by running — a stale region answering
the wrong question, and a post-answer click advancing the sitting — and neither is
tested yet. They are the reason the Done-when carries explicit rows for both.

Not claimed and not started — the request was to file the task.

### 2026-09-23

Claimed. Brainstorm settled with the operator (brackets, multiple-choice in,
hint on the keys line); design recorded under `### Design`. The deciding read
was #67's `passageSpanAt(hit.line, …)` — the absolute-buffer-line identity check
is already the house answer to "regions are never pruned", so the fork's (A)/(B)
tension dissolves: region for coordinates, loop-held range for identity,
`Graded` refusal in `Apply` for post-answer clicks.

Implemented. What the tests pin, and what each mutation proved:

- `play`: `TestAPickedOptionIsGradedExactlyAsItsDigit` (both forms, every
  option, cold and after a peek), `TestAClickOnAnAnsweredQuestionNeitherAnswersNorAdvances`,
  `TestAPickPastTheOptionsIsNothing`, `TestPromptOptionsLocateTheNumbersAndTheRevealHasNone`.
  Mutations: dropping the Graded refusal → 12 failures; routing a pick through
  the board's `advance` instead of `graded` → 4.
- loop: `TestAClickOnAnOptionNumberAnswersOnlyTheQuestionBeingAsked` walks the
  word column (speaks, no review), the gap column (answers), post-answer clicks
  on the list and on `you chose`, and the previous question's stale `[2]`;
  `TestAPickedNumberRecordsWhatItsDigitRecords`;
  `TestOptionNumbersSurviveAWrappedStemAndNoColour` (24 columns, colour on/off).
  Mutations caught: range check removed, range widened to the buffer top,
  option regions not written, number span extended onto the word.
- pty: `TestPTYPlayClickingAnOptionNumberAnswers` — real SGR click on a real
  form 2.3; red with the option regions removed.

Found on the way: the sitting row hung rather than failed under the range
mutation (a wrongly-ended sitting stops reading keys) — its sends are now
bounded. **side-quest:** `seedDeckN` never passed `--here`, so every `--play`
pty row read an empty deck; fixing it exposed that the board's pty click sent a
press without a release, which #67's gesture machine never treats as a click.
Both fixed; the two rows are green. 11 other `TestPTY*` rows still fail on main
for the first-run deck prompt — separate issue.

The keys line is 84 columns (`1-4 or click = pick the word, ? = bad question,
d = remove from deck, Ctrl-C to stop`) and wraps to two rows at 80. The prompt
row is measured (`displayRows`), so it is drawn correctly; left as is and
flagged to the operator rather than trimming shared reserved text.

## Revisions

- 2026-09-23 — implementation: the capability is `play.Picker{Pick(i)}`, not a
  `Marker` embedded in `Grid`. A picked option goes through `graded` — the tail
  the digit uses, where a miss reveals without advancing — while a board mark
  advances, so the two are different capabilities. The prompt range is a
  plain `lineRange` (`asking`) with no question index: `show()` rewrites it for
  every new prompt before the next key, and a board (which writes none) is not
  a Picker, so an index check could never fire. `regionAnswers` declares the
  answering kind beside `regionPlaysAudio`; the editor's actionable guard
  defers it to the sitting row by name.


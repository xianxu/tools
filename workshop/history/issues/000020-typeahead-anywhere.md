---
id: 000020
status: done
deps: []
github_issue:
created: 2026-08-26
updated: 2026-08-26
estimate_hours: 5.24
started: 2026-08-26T11:03:49-07:00
actual_hours: 1.35
---

# typeahead beyond the first word: complete deck words anywhere in the line

## Problem

The grey inline suggestion only fires on the first word of a line.

`Suggestion` (editor.go:145) byte-prefix-matches candidates against the **whole
typed line**, and `completionsFor` (command.go:41) returns whole past lines. So
typing `syco` completes to `sycophantic` — a past line that starts with `syco` —
but typing `what's the difference to obseq` needs a past line starting with that
entire text. A sentence you have never typed before matches nothing, so the grey
tail silently stops existing the moment you type a space.

That is exactly backwards for the words this tool exists to teach: the long,
hard-to-spell ones (`obsequious`, `certiorari`, `defenestrate`) are hardest to
type precisely when you are asking a free-form question ABOUT them, which is the
one place completion currently never fires.

## Spec

Generalise the match from "the whole line" to "any trailing word-boundary
segment of the line", longest first.

A candidate completes the line if it extends some trailing segment of it:

    line:     what's the difference to obseq
    segments: [what's the difference to obseq]   <- today's only case
              [the difference to obseq]
              [difference to obseq]
              [to obseq]
              [obseq]                            <- matches `obsequious`
    render:   what's the difference to obseq▓uious

The first segment with a match wins, so today's whole-line recall keeps
precedence over any inner match — a real past line is always a better guess
than a word glued onto a head.

**Two lists, because they stopped being one.** Up-arrow recalls lines you
actually submitted; the grey tail composes a line you have *not*. Those were the
same set until now, so `Apply` took one `matches` slice and routed it to both
`walk` and `acceptSuggestion` (editor.go:39, :79-82). Glue synthesized entries
into that one slice and Up-arrow starts offering — and Enter starts submitting —
lines the user never typed. So `Apply` takes a `candidates{recall, complete}`
pair instead:

    recall   = hist.Prefix(base)                  // only ever really-submitted lines
    complete = completionsFor(base, hist, cmds)   // what the line could become

This also retires an existing wart: today a command line feeds the command menu
to `walk` too, so Up inside `/his` browses `/history` rather than recalling. With
the split, Up always recalls — commands included, since `recallLine` puts
submitted `/history 7` into `hist`. `Suggestion`, `acceptSuggestion` and
`RenderLine` are otherwise untouched: candidates still arrive glued to the head
(`what's the difference to obsequious`), so they still match the whole line.
That is the pattern command mode established (command.go:30-40: "command mode is
a different match SOURCE rather than a different editor"). ARCH-DRY, ARCH-PURE:
word-boundary logic is one pure function; the editor stays a state machine over
plain data.

**The segment loop lives in the history arm only.** `completionsFor` keeps its
single `parseCommandLine(base)` test on the WHOLE line, exactly as today, and
only the else-branch expands segments:

    completionsFor(base, hist, cmds):
        if parseCommandLine(base) -> commandCompletions(...)   // unchanged, no loop
        else                      -> historyCompletions(base, hist)

So `/history 7` still resolves in the command namespace and still offers nothing
(the completion is shorter than the line), and no trailing segment can re-enter
the command branch and complete a `/command` mid-line. `historyCompletions` and
`trailingSegments` are both pure and both unit-tested directly.

**The candidate source stays `hist`.** The durable half of the recall log is
`EventLookedUp` only (history_store.go:51) — i.e. words you looked up, not text
scraped out of your sentences. So "complete from your deck" is already what the
existing seam yields, with no second store read, no new interface, and no
in-session staleness (a word looked up two minutes ago is completable
immediately, because `hist.Add` already recorded it). Operator chose this source
over "every word I have typed" and over a NOAD headword index.

Two knowing consequences:

- **Failed lookups are in the namespace.** `Load` takes every `EventLookedUp`
  regardless of `Found`, deliberately, so typos stay recallable. A typo is
  therefore completable too — but `prefixMatch` is newest-first and a typo is
  normally followed by its correction, so the correction wins. If this proves
  noisy the fix is a `Found` filter in `storeHistory.Load`, not here.
- **Multi-word headwords work.** `hot dog` is a legal lookup; typing `... hot d`
  matches it at the second-to-last segment, which the whole-line-only match
  could never do.
- **A recall line is not always a word.** `hist.Add` stores `cmd.recallLine()`
  (repl.go:131-144), the canonical *re-submittable* form — so an in-session
  force-literal lookup is stored `\word`, asks as `?…`, commands as `/…`. The
  completion namespace wants the WORD, so `matchesFor` unwraps the marker before
  matching and after gluing. **`?` is unwrapped too** — see Revisions, this
  changed during implementation: the `?` on an asked line is added by the SYSTEM
  (`readsAsQuestion` classifies a bare sentence, `recallLine` stores `?…`), so
  requiring the user to type one to complete a question they asked without one
  would make past questions uncompletable — the opposite of what this issue is
  for. `/` stays alone: commands are a real separate namespace with their own
  completion path, not a marker on a word.

**Inner segments need three runes.** A one- or two-rune tail is far more likely
to be a preposition than a word being reached for: without a floor, typing
`to` suggests `torpid` and every second word grows a grey tail. The floor
applies only to segments after the first — segment 0 is the whole line, where
there is no ambiguity about what is being completed, so `ob` still completes
there exactly as it does today. No regression to single-word typeahead.

Out of scope: a NOAD headword index (would complete words you have no
relationship to); completing at a cursor that is not at end of line
(`Suggestion` already declines that case).

## Done when

- [x] Typing a partial deck word anywhere in a multi-word line shows the grey tail.
      `TestGreyTailCompletesADeckWordMidSentence`, and on a real terminal in
      `TestPTYSuggestionAndAcceptance`.
- [x] Tab / Right / End accept it, producing the whole line with the word
      completed. Tab asserted through a pty.
- [x] A past line still wins over an inner segment when both match.
      `TestWholeLineBeatsAnInnerSegment`.
- [x] Single-word typeahead still completes at 2 runes.
      `TestGreyTailStillCompletesASingleWordAtTwoRunes`. **Amended:** this row
      first read "byte-identical to today". It is not, and the difference is
      wanted — unwrapping `?` means a past question can now complete at segment 0
      where before only a lookup could. What is preserved is that every line that
      completed before still completes to the same thing; what is added is lines
      that could not complete at all.
- [x] `/command` completion is untouched. `TestCommandCompletionIsUnchanged`.
- [x] A line ending in a space suggests nothing (no empty-prefix flood).
      `TestTrailingSpaceSuggestsNothing`, and structurally: a segment starts only
      where a word starts.
- [x] `head + segment == line` for every segment. `FuzzTrailingSegments`, 864k
      execs clean.
- [x] Up/Down walk ONLY really-submitted lines — never a glued candidate.
      `TestUpRecallsSubmittedLinesNotTheCommandMenu`,
      `TestUpOffersNothingWhenNoSubmittedLineMatches`.
- [x] `/history 7` completes exactly as today; no segment re-enters the command
      branch. `TestACommandArgumentDoesNotDrawFromHistory`,
      `TestASlashSegmentMidLineDoesNotCompleteACommand`.
- [x] An in-session `\word` lookup completes mid-line as `word`.
      `TestForcedLiteralLookupCompletesAsThePlainWord`.
- [x] A past question completes when retyped bare.
      `TestPastQuestionCompletesWhenRetypedBare`. Added during implementation.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Derived after the plan cleared plan-quality (#187), one item per Plan row plus
the process work inside the measured window. No separate plan doc: per AGENTS.md
§1 this is single-file work, so the issue's Spec + Plan *is* the plan artifact —
and the gate certified it executable as-written, which is what the "thorough plan
doc" buffer condition proxies for. Hence +15%, not +30%.

Design carries v2's ×0.2 spec-quality discount on every primitive the plan
pre-resolves — which is all of the code items: the segment rule, the precedence
order, the floor, the namespace split and the `\`-strip are all decided above,
so what is left at design time is reading. No discount on spec authoring or on
review rounds; those *are* the design work rather than beneficiaries of it.
Implementation is v3.1's 40% of the v2 table.

`issue-spec` is priced at the **bottom** of its 0.5–1.5 band rather than the mid.
The band assumes a brainstorm; this spec had exactly one open fork (the mid-line
candidate source), the operator resolved it in a single exchange, and everything
else fell out of reading four existing functions.

**Review rounds are priced from the only round cost anyone has actually
measured: #16's ≈0.9h.** An earlier draft of this block priced them *below* that,
citing #17 as having come in under estimate with three boundary rounds included.
That citation was wrong, and #17's own log says so
(`workshop/issues/000017-user-model.md:308-312`): its review cost "is still
unmeasured for this issue", the 3.8h figure covers the feature work only, and it
was hand-recorded as a wall-clock upper bound after `sdlc actual` returned an
impossible 15.42h on a 3h49m window — not a measurement at all. So the rounds
are written `design=0.30 impl=0.60` (= 0.90, #16's observed rate), and **three**
are budgeted, not two.

Three because the Plan carries no `Mx` tags: per AGENTS.md §3 this is single-pass
atomic work with exactly ONE mandatory boundary review, at `sdlc close`. The
other two rows are the fix-then-re-review cycles that verdict reliably produces —
#16 needed 3–4 per milestone and #17 M1 needed 3. Budgeting one round would be
budgeting for a CLEAN first pass this repo has not yet produced.

**Disclosure: the drift in this repo runs the other way, and nothing here
corrects for it.** `calibration-ledger.tsv`, newest-per-issue with
`window_trusted=yes`: tools#1 0.59, #3 0.96, #4 0.20, #11 0.64, #14 2.83, #15
0.27, #16 0.37 — six of seven under 1.0. The nearest analogue is the closest one
possible: **#15 (0.27×) is the command-mode typeahead work in these exact
functions.** If that ratio held, this lands near 10h rather than 5.24. The items
are *not* inflated to meet that prediction — hand-correcting a derivation to hit
a guessed actual is the back-fitting the estimate gate exists to catch, and it
would destroy this row's value as evidence. Familiarity stays 1.0 (same package,
same files) even though #15 shows familiarity did not help there. If this closes
near 10h, that is the v3.1 model drifting (ariadne#127) and this row is data for
the recalibration, not a mistake to have hidden.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec             design=0.50 impl=0.08
item: milestone-review       design=0.10 impl=0.14
item: milestone-review       design=0.10 impl=0.14
item: cross-cutting-refactor design=0.12 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: smaller-go-module      design=0.03 impl=0.14
item: atlas-docs             design=0.03 impl=0.05
item: milestone-review       design=0.30 impl=0.60
item: milestone-review       design=0.30 impl=0.60
item: milestone-review       design=0.30 impl=0.60
design-buffer: 0.15
total: 5.24
```

Item-to-Plan-row map, one row each and none unmapped: `issue-spec` = the Spec
(spent); `milestone-review` ×2 = plan-quality round 1 (spent) and round 2 +
estimate-quality (spent); `cross-cutting-refactor` = Plan row 1, the `candidates`
split; `smaller-go-module` ×5 = rows 2, 3, 4+5, 6, 7 — rows 4 and 5 are one item
because "write the failing test" and "add `historyCompletions`" are one TDD
cycle, not two; `atlas-docs` = row 8; `milestone-review` ×3 = the close boundary.

Two mappings are deliberately *not* the obvious ones. `historyCompletions` is
`smaller-go-module`, not `greenfield-go-module`: it is explicitly the else-branch
of an existing function, which is the table's own "well-specced; mirror or
extend" definition, and pricing it as greenfield alongside `trailingSegments` as
smaller would have been two prices for two comparable pure functions. The
mutation check is `smaller-go-module`, not `milestone-review`: mutating a
constant, an ordering and a struct split and re-running the suite each time is
implementation work, not review overhead.

## Plan

- [x] Split the candidate list: `candidates{recall, complete}`, `Apply` takes it,
      `walk` gets `recall`, `acceptSuggestion` gets `complete`. Two production
      call sites, not one: `Apply` at replraw.go:180 takes the pair, and `draw`
      at replraw.go:126 already feeds `Suggestion` the complete list (correct
      as-is). Plus ~5 test sites (editor_test.go:15,157,239,243,247). Assert Up
      never surfaces a glued line, and that Up in command mode now recalls.
- [x] Write `trailingSegments` + table test (pure, no IO): boundaries, unicode,
      runs of spaces, trailing space, empty line, single word.
- [x] Add `FuzzTrailingSegments` for the `head+text == line` invariant — the
      repo idiom (invariant_test.go, live_property_test.go); a table is blind to
      the malformed-input class by construction.
- [x] Write the failing completion test: multi-word line, deck word tail.
- [x] Add `historyCompletions` (segment loop + glue + `\` strip); leave
      `completionsFor`'s command branch untouched. Unit-test it directly.
- [x] Regression tests: whole-line precedence beats an inner segment, 2-rune
      first segment still completes, `/history 7`, trailing space, multi-word
      headword, in-session `\word`.
- [x] Mutation-check that the floor constant, the precedence order, and the
      recall/complete split each actually bite.
- [x] Sweep atlas/define.md for the "matches the whole typed line" claim.

## Log

### 2026-08-26
- 2026-08-26: closed — go test ./... + go vet green; go test -race ./cmd/define green. Live pty conformance (TestPTYSuggestionAndAcceptance, darwin+conformance tag) asserts the grey tail mid-sentence on a real terminal and Tab accepting it. FuzzTrailingSegments 864k execs clean on the head+text==line invariant. Eight mutations run, each killing exactly its intended test: floor 3->1, floor applied to segment 0, each submission marker dropped, segments reversed, empty trailing segment emitted, namespace order swapped, parseCommandLine per segment; plus a pre-#20 revert of completionsFor, which reproduced the operator-reported bug in the pty frame (every keystroke to "what is a syc" with no grey tail). Two vacuous tests found and fixed by that sweep: typeKeys resolved candidates with h.Prefix instead of the production candidatesFor, so two split tests passed before the split existed; and TestCommandCompletionIsUnchanged used history that both namespace orderings answered identically. Both recorded in lessons.md.; review verdict: FIX-THEN-SHIP

Opened from an operator report: "type syco would allow grayed auto completion,
but auto complete not showing up in longer sentence I type."

Design fork resolved with the operator: the mid-line candidate source is the
deck (words looked up), not every word ever typed, and not a dictionary index.

## Revisions

### 2026-08-26 — plan-quality round 1

Five findings, all accepted; the two blocking ones were real design holes.

- **PQ-1 (Critical), `shared-seam-second-consumer`** — addressed. I had noticed
  that `matches` feeds both `walk` and `acceptSuggestion` and talked myself into
  accepting synthesized entries in the Up-arrow stream. The gate was right that
  this is a design decision, not a footnote: Up would replace your line with a
  sentence you never typed and Enter would submit it. Fixed as the class, not the
  site — the two namespaces get two fields (`candidates{recall, complete}`)
  rather than the glue being special-cased at one call. Retires the pre-existing
  command-mode variant of the same bug for free.
- **PQ-2 (Important), `command-namespace-boundary`** — addressed. Missed
  entirely. The segment loop now lives inside `historyCompletions`; the
  `parseCommandLine` test stays once, on the whole line. Named as a directly
  unit-tested function, as asked.
- **PQ-3 (Minor), `recall-line-not-raw-word`** — addressed rather than
  documented. `historyCompletions` strips the `\` force marker so the spec's
  claim is true instead of caveated; `?`/`/` deliberately left alone.
- **PQ-4 (Minor), `invariant-needs-property-not-table`** — addressed. Added
  `FuzzTrailingSegments` to the plan.
- **PQ-5 (Minor), `in-flight-branch-ownership`** — surfaced to the operator.
  #17 is `working` on the current branch with M2 blocked on #6, so the branch
  point for #20 is a merge-ordering question (stacking on #17 means merging #20
  would merge #17's M1 too), not a purely technical one.

### 2026-08-26 — estimate-quality round 1

Verdict INFO (non-blocking), eight findings, and the two Importants were right in
a way worth recording: **I cited #17's log for the opposite of what it says.**

- **F1 (Important)** — addressed. The block justified pricing review rounds
  *below* #17's figure by claiming #17 "came in under, three boundary rounds
  included, measured ~3.8h". #17's log (`:308-312`) says the review cost "is
  still unmeasured for this issue", that M1 had not been through even one round
  at the time of writing, and that 3.8h was **hand-recorded as a wall-clock upper
  bound** because `sdlc actual` returned an impossible 15.42h. I wrote a sentence
  contradicting an entry I made the day before. Rounds are now priced at #16's
  ≈0.9h — the only round cost ever measured — and three are budgeted, not two.
  Estimate 3.87 → 5.24.
- **F2 (Important)** — addressed by disclosure rather than by inflation. The
  ledger's trusted tools rows run 0.20–0.96 (six of seven under 1.0), and #15 —
  this exact `completionsFor` code — ran 0.27×. Stated in the block, with the
  reason the items were not adjusted to meet it: back-fitting a derivation to a
  predicted actual is what the gate exists to catch, and it would make this row
  worthless as recalibration evidence.
- **F3 (Minor)** — addressed. `historyCompletions` re-priced `greenfield-go-module`
  → `smaller-go-module`; it is the else-branch of an existing function, i.e. the
  table's "mirror or extend".
- **F4 (Minor)** — addressed. Plan row 4 had no line item while the prose claimed
  one item per row. Rows 4+5 are now explicitly one TDD item.
- **F5 (Minor)** — addressed. The mutation check moved `milestone-review` →
  `smaller-go-module`; it is implementation, not review overhead.
- **F6 (Minor)** — addressed. The block now states the boundary structure it is
  pricing: no `Mx` tags, one mandatory review at close, two fix-then-re-review
  cycles.
- **F7 (Nit)** — addressed; the second process item is named as plan round 2 plus
  estimate-quality.
- **F8 (Nit)** — addressed in the Plan itself: two production call sites plus
  ~5 test sites, not "the one production call site".

### 2026-08-26 — implementation: one spec decision reversed

The Spec said `?` would be left alone, on the reasoning that it "marks a
genuinely different namespace". Writing the tests made that wrong: unlike `\`,
which the user types, the `?` on an asked line is added by the SYSTEM —
`readsAsQuestion` classifies a bare sentence as a question and `recallLine`
stores the canonical `?…` form. So a question asked the ordinary way (typed bare)
is stored with a marker the user never typed, and leaving `?` alone would mean
past questions can only be completed by typing a character nobody types.

Since the operator's request was explicitly "including free form questions", that
is the purpose rather than an extension of it (ARCH-PURPOSE). `matchesFor`
unwraps both markers. `/` still stands alone — commands have their own completion
path, so unwrapping it would mean two namespaces answering one line.

This widens segment-0 behaviour, so the "byte-identical to today" Done-when row
was amended rather than quietly ticked.

### 2026-08-26 — close review: FIX-THEN-SHIP, and one finding is this round's own lesson

Verdict FIX-THEN-SHIP, no Criticals, four Importants. The reviewer independently
re-ran the mutation sweep and killed the same axes I had — and found the one I
had not.

- **I1 (Important) — segment precedence was NOT pinned, though Plan row 7 claimed
  it.** My M5 mutant reversed `trailingSegments`' OUTPUT, which reddens the table
  test on ordering; the sharper mutant reverses the ITERATION in
  `historyCompletions`, leaving the table green. Then the whole suite passes,
  because `TestWholeLineBeatsAnInnerSegment`'s fixture (`hist("island", …)`
  against `hot dog`) has no inner-segment match at all — the same defect as the
  `sevenfold` fixture I had already found and written a lesson about, on a second
  axis of the same commit. Fixture is now `hist("dog house", "hot dog and
  fries")`: longest-first gives `" and fries"`, shortest-first `" house"`.
  Verified red under the iteration mutant, green on HEAD.
- **I2 (Important) — the atlas's PRIMARY editor description was still pre-#20.**
  My sweep matched one phrase ("matches the whole typed line") and missed
  `Apply(Editor, Key, matches)` and "hands the same slice to both" 200 lines
  above the new prose. Both fixed, plus the `:444-446` sibling.
- **I3 (Important) — four of the five test sites Plan row 1 enumerated were left
  standing.** All four were `Suggestion(e, h.Prefix(...))`, the exact anti-pattern
  this round's own lessons.md entry names, in the same file, in the same round.
  Instance vs class again (ARCH-PURPOSE). All four now resolve via
  `completionsFor`.
- **I4 (Important) — `?` stayed a bare literal at three sites while `\` got a
  constant, in the same commit whose comment says "one constant because three
  places have to agree".** `submissionMarkers` was literally the class variable
  with one member a constant and one a literal. Added `forceAsk` beside
  `forceLiteral`.
- **Minors** — the `forceLiteral` const had been inserted between `recallLine`'s
  doc comment and its signature (godoc attached the BR-12 rule to a
  one-character constant); `candidates`' "newest-first" claim was false for
  `complete`, which is grouped-by-marker; the floor counts a trailing space, and
  "first match" is not "first useful match" — both kept and documented with the
  reason rather than changed, since each is the behaviour I want; `draw`'s use of
  `completionsFor` over `candidatesFor` now says why.
- **Coverage gaps closed** — Done-when row 2 rested entirely on a pty test behind
  `//go:build darwin && conformance`, which SKIPS wherever a pty cannot be
  allocated (it does in the sandbox here). `TestTabAcceptsAGluedCandidate` now
  covers glued acceptance in the default suite; `TestMatchesForDedupesAcrossMarkers`
  covers cross-marker dedup. Both mutation-verified.

### 2026-08-26 — the measured actual was wrong, and the cause is nameable

`sdlc close` adopted **0.33h** (ratio 15.9×, recorded trusted). It measured only
`e058be39 → HEAD` — 11:45 to 12:02, the two #20 code commits — while its own
warnings say it saw and discarded the rest:

    unattributed 58.4m/100% unattributed fallback without issue commit boundary
                                          (2026-08-26 11:03 → 2026-08-26 12:02)

Unlike #17's defect (an impossible 15.42h on a 3h49m window), this one has a
clear cause: **`sdlc claim` does not plant the anchor AGENTS.md §2 promises it
plants.** §2 says the claim commit "anchors the active-time window at the claim
commit, so design attention is measured". But claim-on-main commits with the
generic message `issue-sync: update issues`, which carries no issue reference —
so the attribution engine cannot tie it to #20 and falls back to the first
`#20`-referencing commit, discarding every minute of design, planning, both plan
gate rounds and the estimate gate. Here the first claim additionally failed to
commit at all (`could not find a worktree on branch 'main'`), but that is
incidental: the 11:35 claim DID commit and still could not anchor.

Recorded **1.35h** instead — wall clock from the first claim attempt (11:03) to
this close commit, therefore an upper bound on active time; the session was
continuous, and v3.1 counts AI execution spans as elapsed wall time. Against the
5.24 estimate that is 3.9×, i.e. the estimate was HIGH — worth noting because the
whole estimate rationale argued the repo's drift runs the other way. One round of
boundary review instead of the three budgeted is most of the difference.

The calibration ledger row was corrected from 0.33 to 1.35 **and flipped to
`window_trusted: no`**: a hand-measured value should not enter the scale fit as
clean evidence, and leaving a trusted 15.9× row would have been the worse
pollution. This is an ariadne bug — `sdlc claim` should reference the issue in
its commit subject — and it is the second measurement defect in two issues.

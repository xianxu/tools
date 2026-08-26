---
id: 000020
status: working
deps: []
github_issue:
created: 2026-08-26
updated: 2026-08-26
estimate_hours: 5.24
started: 2026-08-26T11:03:49-07:00
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
  completion namespace wants the WORD, so `historyCompletions` strips a leading
  `\` from candidates before matching and after gluing. `?` and `/` are left
  alone: those mark genuinely different namespaces, whereas `\` exists only to
  disambiguate submission and the thing behind it is exactly a deck word.
  (Durable entries carry no marker — `Load` reads `e.Word` — so this is an
  in-session-only gap, but it is one line to close.)

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

- [ ] Typing a partial deck word anywhere in a multi-word line shows the grey tail.
- [ ] Tab / Right / End accept it, producing the whole line with the word completed.
- [ ] A past line still wins over an inner segment when both match.
- [ ] Single-word typeahead is byte-identical to today, including 2-rune prefixes.
- [ ] `/command` completion is untouched.
- [ ] A line ending in a space suggests nothing (no empty-prefix flood).
- [ ] `head + segment == line` for every segment, so gluing cannot corrupt the line.
- [ ] Up/Down walk ONLY really-submitted lines — never a glued candidate.
- [ ] `/history 7` completes exactly as today; no segment re-enters the command branch.
- [ ] An in-session `\word` lookup completes mid-line as `word`.

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

- [ ] Split the candidate list: `candidates{recall, complete}`, `Apply` takes it,
      `walk` gets `recall`, `acceptSuggestion` gets `complete`. Two production
      call sites, not one: `Apply` at replraw.go:180 takes the pair, and `draw`
      at replraw.go:126 already feeds `Suggestion` the complete list (correct
      as-is). Plus ~5 test sites (editor_test.go:15,157,239,243,247). Assert Up
      never surfaces a glued line, and that Up in command mode now recalls.
- [ ] Write `trailingSegments` + table test (pure, no IO): boundaries, unicode,
      runs of spaces, trailing space, empty line, single word.
- [ ] Add `FuzzTrailingSegments` for the `head+text == line` invariant — the
      repo idiom (invariant_test.go, live_property_test.go); a table is blind to
      the malformed-input class by construction.
- [ ] Write the failing completion test: multi-word line, deck word tail.
- [ ] Add `historyCompletions` (segment loop + glue + `\` strip); leave
      `completionsFor`'s command branch untouched. Unit-test it directly.
- [ ] Regression tests: whole-line precedence beats an inner segment, 2-rune
      first segment still completes, `/history 7`, trailing space, multi-word
      headword, in-session `\word`.
- [ ] Mutation-check that the floor constant, the precedence order, and the
      recall/complete split each actually bite.
- [ ] Sweep atlas/define.md for the "matches the whole typed line" claim.

## Log

### 2026-08-26

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

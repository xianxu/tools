---
id: 000021
status: working
deps: []
github_issue:
created: 2026-08-26
updated: 2026-08-26
estimate_hours: 9.28
started: 2026-08-26T12:40:00-07:00
---

# highlight the words you are learning wherever they appear

## Problem

Nothing on screen distinguishes a word the learner has studied from a word they
have never met. A definition of `obsequious` may use `sycophantic` in its own
gloss — the exact connection worth noticing — and it reads the same as every
other word on the line. An LLM answer comparing two words the learner looked up
last week gives no sign that those two are already theirs.

Operator's framing:

> use different color ... for the words user checked ... this makes them easier
> to spot ... such highlighting should appear in definition and LLM responses as
> well, this help reinforce words user are learning.

The reinforcement only works if it is everywhere text appears. Highlighting only
the prompt would mark words at the moment the learner already knows they are
typing them — the least informative moment.

## Spec

**One predicate, three surfaces.** Every highlight decision goes through
`Vocabulary.Has(word)`. Today that set is the deck; #22 will narrow it to the
words still being learned, and nothing downstream should change when it does.
The seam exists now precisely so that swap is one place later (ARCH-PURPOSE:
build the seam the stated future needs, not the future itself).

**The source is the deck, not history.** `words/<slug>.yaml` holds found
lookups, canonical and correct. The event log holds every attempt including
typos, which is right for recall (#20 completes from it deliberately) and wrong
here — highlighting a misspelling as a word you know is the opposite of
reinforcement. Operator was explicit: *"not history, but correct form of the
words"*.

**Matching is exact, on `store.Key` normalisation** — case-insensitive,
whitespace-collapsed, so `Obsequious` matches `obsequious`. Multi-word headwords
(`hot dog`, `a priori`) match as phrases via longest-match at each token
boundary. Inflections do NOT match: `obsequiousness` stays plain until it is
itself looked up. Operator chose this over a suffix list, which buys
`obsequiously` at the cost of wrongly lighting `rationing` for a deck holding
`ration`, and still misses stem changes like `run`/`running`.

**Bold green**, chosen over underline and amber. It reads as "known", is
instantly separable from the grey suggestion and the plain bold of ordinary
input, and on the prompt line it collides with nothing (amber is the
part-of-speech label inside definition bodies).

### The three surfaces, and why two of them are one mechanism

| surface | how text reaches the screen | mechanism |
|---|---|---|
| typed line | `RenderLine` builds one frame string | pure matcher, directly |
| definitions | `Render(e, opt) string`, fully built then printed | `highlightWriter` |
| LLM answers | `Stream(ctx, req, func(delta string){...})`, in chunks | `highlightWriter` |

Definitions and answers share a mechanism because they share the hard part:
**the text is already carrying ANSI codes, and a word can straddle a boundary.**
`Render` emits `\x1b[3;32m` around examples, `\x1b[1;33m` around parts of speech;
ANSI has no nesting, so injecting green inside a styled run must restore the
enclosing style afterwards. And a stream delivers `obseq` in one chunk and
`uious` in the next.

`highlightWriter` is a stateful `io.Writer` that (1) passes escape sequences
through untouched while tracking the active SGR, (2) holds back a trailing
partial token across writes, (3) rewrites matched tokens as
`green + word + off + enclosingSGR`. Feeding it a complete string and flushing
is the definition case; wrapping the stream's `out` is the answer case. This is
the `crlfWriter` shape already in the tree (`crlf.go`), which carries `lastWasCR`
across writes for exactly the same class of reason.

The typed line does not use it: `RenderLine` builds a whole frame with one
enclosing style and no streaming, so the pure matcher serves it directly. The
matcher is the shared core — one implementation of "which spans of this text are
words you know", two renderers over it (ARCH-DRY, ARCH-PURE).

### Deliberately out of scope

- **Narrowing to actively-learned words** — that is #22, and it is why the seam
  is a predicate rather than a deck read.
- **Inflection/stemming** — decided against above.
- **Re-styling the headword line.** The head is already bold cyan; highlighting
  is for body text, answers, and input. The headword's own occurrences *inside*
  body text do highlight, uniformly with every other deck word — one rule, and
  #22 is the operator's own answer to any resulting noise.
- **`-raw` output**, which is by contract the unparsed entry.
- **Highlighting the grey suggestion**, which is grey by definition.

## Done when

- [ ] A deck word in the line you are typing renders bold green; the rest is unchanged.
- [ ] A deck word appearing in a definition body renders bold green, and the
      enclosing style resumes after it.
- [ ] A deck word in a streamed LLM answer renders bold green even when it
      arrives split across chunks.
- [ ] Matching is case-insensitive and covers multi-word headwords.
- [ ] `-no-color` and piped output emit no highlight codes at all.
- [ ] A word looked up during the session highlights immediately, without restart.
- [ ] `Render`'s no-data-loss invariant still holds with highlighting on.
- [ ] Every highlight decision routes through `Vocabulary.Has`, so #22 changes
      one place.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

Derived after the plan cleared plan-quality (#187), one item per task in
`workshop/plans/000021-highlight-learned-plan.md` plus the process work inside
the measured window. Design carries v2's ×0.2 spec-quality discount on every code
item — the plan pre-resolves the matcher, the phrase rule, the writer's four
contract rules and the wiring points, so what is left at design time is reading.
No discount on spec/plan authoring or review rounds; those *are* the design work.
Implementation is v3.1's 40% of the v2 table. Familiarity 1.0 — same package,
and `crlfWriter` is a working precedent for the hardest piece.

`issue-spec` is priced at the **top** of its 0.5–1.5 band, once rather than
twice. It covers the spec *and* a 368-line durable plan with a normative contract
section, and the scope changed mid-specification (prompt line → all three
surfaces), which meant re-deriving the architecture rather than extending it.

**Review rounds are priced from #20's measurement — the first round cost this
repo has actually observed end to end.** #20's single boundary review took ~7
minutes of gate time plus ~25 minutes of fixes, ≈0.5h, so a round is written
`design=0.20 impl=0.30`. Four are budgeted: this plan has three `Mx` boundaries,
each of which owes a `milestone-close`, plus one fix-then-re-review cycle — #20
needed exactly one and its verdict was FIX-THEN-SHIP.

**Two UX iteration rounds are line items, not hope.** The entire deliverable is a
colour choice: the operator has already changed scope once mid-spec (prompt line
→ all three surfaces, recorded in the Log), the palette decision is contested in
the Spec, and the plan's own Risks section says the green-inside-italic-green
case must be looked at on a real terminal rather than reasoned about. v2.1's
known-limitations names this directly — 3–5 iteration rounds are typical for TUI
features, not 1. Two is the conservative read given the colour is already chosen.

**M3 is two `cross-cutting` items, not one.** Task 8 carries three flush exit
paths to enumerate and test, the `highlightWriter`/`crlfWriter` nesting order to
determine *and* pin, and a capture-derived test helper. Pricing that as one item
made M3 cost a third of M1 while containing the riskiest integration in the plan.

**Disclosure: the two populations of evidence disagree, and averaging them would
be a fudge.** The trusted ledger rows say this repo under-estimates: tools#1 0.59,
#3 0.96, #4 0.20, #11 0.64, #14 2.83, #15 0.27, #16 0.37 — six of seven below
1.0. But both hand-measured closes say the opposite: #17 M1 est 8.39 against ~3.8h
recorded, and #20 est 5.24 against 1.35h. Those two are `window_trusted: no`
precisely because `sdlc actual` failed on them in opposite directions, so they are
not comparable to the trusted rows and must not be pooled with them. I am
deriving from the primitive table and reporting the tension rather than applying a
correction factor to land somewhere between — a back-fitted total is what the
estimate gate exists to catch, and #20 is already in the ledger as evidence for
whichever way this resolves.

The estimate-quality judge offered a sharper hypothesis than "the populations
disagree", and it is worth naming because this issue can test it: the
over-pricing may be in **design/process** while the under-pricing is in
**implementation**. That reconciles both populations — #16 (est 6.49 → 17.63) and
#11 (est 7.98 → 12.38) ran long on implementation-heavy work, while #17 and #20,
whose estimates were dominated by process and review rounds, ran short. This
block is 54% design after buffering; if it closes short again, that is the
hypothesis confirming, and the fix is to the design side of the primitive table
rather than to a global factor.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec             design=1.50 impl=0.08
item: milestone-review       design=0.10 impl=0.14
item: milestone-review       design=0.10 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: smaller-go-module      design=0.03 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: cross-cutting-refactor design=0.12 impl=0.14
item: milestone-review       design=0.20 impl=0.30
item: smaller-go-module      design=0.03 impl=0.14
item: greenfield-go-module   design=0.25 impl=0.22
item: smaller-go-module      design=0.03 impl=0.14
item: cross-cutting-refactor design=0.12 impl=0.14
item: milestone-review       design=0.20 impl=0.30
item: smaller-go-module      design=0.03 impl=0.14
item: cross-cutting-refactor design=0.12 impl=0.14
item: cross-cutting-refactor design=0.12 impl=0.14
item: ux-rename-iteration    design=0.55 impl=0.08
item: ux-rename-iteration    design=0.55 impl=0.08
item: atlas-docs             design=0.03 impl=0.05
item: milestone-review       design=0.20 impl=0.30
item: milestone-review       design=0.20 impl=0.30
design-buffer: 0.15
total: 9.28
```

Item-to-task map. Process: spec+plan, then plan rounds 1 and 2 (both spent).
**M1** — `greenfield` = Task 1 the `Vocabulary` seam; `smaller` = Task 2 the
tokenizer + its fuzz; `greenfield` = Task 3 `highlightSpans` + its fuzz;
`cross-cutting` = Task 4, which touches `editor.go`, `replraw.go`, `main.go` and
`capture.go`; then the M1 boundary. **M2** — `smaller` = Task 5 `sgrState`;
`greenfield` = Task 6 `highlightWriter`; `smaller` = Task 6's downstream-contract
tests and fuzz, itemised separately because contract rule 4 is a second piece of
work rather than a case of the first; `cross-cutting` = Task 7 definitions plus
the no-data-loss invariant; then the M2 boundary. **M3** — `smaller` = Task 8's
capture-reading helper; `cross-cutting` = the stream wiring, the flush paths and
the `crlfWriter` ordering; `atlas-docs` = Task 8 Step 7; then the close boundary
and one fix-then-re-review.

## Plan

Durable plan: `workshop/plans/000021-highlight-learned-plan.md` (three
milestones; each `Mx` row below is its own review boundary).

- [x] M1 — `Vocabulary` seam + pure matcher + typed line
- [x] M2 — `highlightWriter` + definitions
- [ ] M3 — LLM answers, streamed

## Log

### 2026-08-26
- 2026-08-26: closed M1 — go test ./... + go vet + gofmt clean. FuzzWordRuns re-fuzzed 2.24M execs clean after its property was corrected to the trimmed contract (it was RED at HEAD; go test runs seeds only). FuzzHighlightSpans 792k execs on the span-join invariant. Round 2 findings addressed as rules, not instances: (a) the test-completeness enumeration is now over the production dependency chain rather than over comments — hop 2, withStore merging sd.vocab into deps.vocab, carries no comment and was invisible to the round-1 sweep while its deletion kills the feature with the suite green; now pinned by TestWithStoreCarriesTheHighlightSetThrough using the deps{newStore: openStore}.withStore pattern, and MUT-G dies. (b) doc prose at a boundary describes only what that milestone shipped — the round-1 fix commit had written a fresh over-claim into README while fixing the atlas one; README now covers the typed line only and Task 8 Step 7 records that its job is to widen it. Twelve mutations die across M1. ACTUAL 2.9h is wall clock 12:40-15:35; sdlc actual reports 0.84h while its own warning says it discarded 117.6m as unattributed — the sdlc claim anchoring defect from #20.; review verdict: SHIP

Opened from the operator's request. Scope grew mid-specification: the first ask
was the prompt line only, then "such highlighting should appear in definition and
LLM responses as well". The follow-on concept (words graduating out of the
highlight set) is filed as #22 rather than deferred inside this issue, and shaped
this spec's central decision — the predicate seam.

- 2026-08-26: M1 — `Vocabulary` seam (`vocab.go`), tokenizer + `highlightSpans`
  (`highlight.go`), typed-line rendering, and in-session growth through the one
  capturer that already knows a lookup earned a deck entry. Six mutations run;
  five died first time. The sixth — aliasing `knownOn` to `inputOn`, which
  removes every visible highlight — SURVIVED, because the assertion was
  `Contains(got, knownOn+"obsequious")`, the constant compared against itself.
  That is the #16 lesson in a second shape, on the one feature whose entire
  deliverable is a colour. Now asserted as literal escape bytes plus
  `knownOn != inputOn`, and the mutant dies. lessons.md extended rather than
  given a new entry: same family, and a reader hitting the first shape should
  find the second beside it.

- 2026-08-26: M1 boundary round 1 — FIX-THEN-SHIP, 8 findings, 2 blocking, all
  addressed. Both blocking findings were one rule the reviewer stated as a
  sweepable enumeration: for each behaviour the diff asserts in a comment, does a
  mutation that falsifies it redden a named test? The set→screen link — the M1
  Done-when itself — was pinned by nothing, because every test called RenderLine
  directly and none drove the loop; passing nil at both call sites and deleting
  voc.Load() each survived the whole suite. That is #20's typeKeys class again,
  and it was also a plan step (Task 4 Step 7) the milestone had quietly skipped.
  The BR-2 fix was then vacuous on the first attempt for a third reason —
  failingStore fails AppendEvent, so Capture returned before the deck write and
  the mutant lived regardless. Five mutations now die; lessons.md gains the
  isolating-double rule.

- 2026-08-26: M1 boundary round 2 — REWORK. One Critical of my own making: the
  round-1 joiner-trim changed the wordRuns contract and FuzzWordRuns still
  asserted the old one, so the target was RED at HEAD and I did not know — `go
  test` runs a fuzz target against its seed corpus only. Property rewritten,
  seeds gained the shape I had just changed, re-fuzzed 2.24M execs clean. Two
  Importants, both explicitly "fix the rule, not the instance": (a) round 1's
  enumeration was over COMMENTS, and the missing hop — withStore's vocab merge —
  carries no comment, so the sweep was structurally blind to it; deleting that
  line kills the feature in production with the whole suite green. Replaced with
  the production-chain enumeration (one test per hop, crossing through production
  code) and hop 2 is now pinned. (b) The commit that fixed round 1's
  atlas-claims-unbuilt-surface finding wrote a FRESH instance of it into README
  in the same commit. Both rules recorded in lessons.md.

- 2026-08-26: M1 boundary round 3 — SHIP. 8 findings disposed, 4 advisory
  recorded. Took all four now rather than at the close review, since M2 builds
  directly on this code: the "define: " warning prefix was written out in three
  seams (one `warnTo`, each seam keeping its own policy — capture's warn-once,
  history's session-only suffix); the Vocabulary seam had TWO absence
  representations (nil and an empty stand-in) with four guards, now nil only,
  interpreted in `highlightSpans` and guarded elsewhere only where nil would
  panic; `voc.Load()` read the whole deck under `-no-color`, where the render
  cannot show a highlight, now skipped; and a deck word sharing a command name
  rendered green inside `/history 7`, which leaked the vocabulary feature across
  the namespace boundary #20 decides exactly once — `highlightSetFor` withholds
  the set on a command line.

- 2026-08-26: M2 — `sgrState` + `scanEscape` (`sgr.go`), `highlightWriter`
  (`highlightwriter.go`), and definitions wired at the one `Render` print site.
  The design fault worth recording is rule 3 of the writer's contract, which the
  plan did not anticipate: holding the last `MaxPhraseWords` tokens is correct
  for text that may still grow, but that hold point lands INSIDE a completed
  phrase — with `hot dog` in the deck, "one hot dog please" emitted `hot` alone
  and lost the match permanently, because the held remainder is re-analysed
  without it. A known span straddling the hold point now drags it back to that
  span's start. Found by the phrase tests, not by reasoning.
  Chain enumerated per M1's rule: lookupAndRender reads d.vocab → highlightText →
  writer → stdout, pinned end-to-end by TestDefinitionBodyHighlightsADeckWord
  (driven through lookupAndRender, not through the helper); the colour gate by
  TestDefinitionHighlightingIsOffWithoutColour. Seven mutations, all die.
  FuzzHighlightWriterIsChunkIndependent 803k execs on byte-identity across split
  points plus visible-text preservation.

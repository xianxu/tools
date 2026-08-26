---
id: 000020
status: working
deps: []
github_issue:
created: 2026-08-26
updated: 2026-08-26
estimate_hours:
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

## Plan

- [ ] Split the candidate list: `candidates{recall, complete}`, `Apply` takes it,
      `walk` gets `recall`, `acceptSuggestion` gets `complete`. Update the one
      production call site (replraw.go:180) and the editor tests. Assert Up never
      surfaces a glued line, and that Up in command mode now recalls.
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

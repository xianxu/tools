---
id: 000068
status: working
deps: [tools#67]
github_issue:
created: 2026-09-16
updated: 2026-09-17
estimate_hours:
started: 2026-09-17T10:56:31-07:00
---

# define: use real sentences as cloze material — finish the `usage/` thread

## Problem

`define` caches real sentences for a word and never reads them.

`usage/<slug>.yaml` holds `store.NewsItem` records from Google News RSS, and
`entryUsages` supplies NOAD's own examples; `bothSources` (`usage.go:153`) joins
them behind one seam and is constructed into `deps` at `main.go:280` and `:389`.
**`Usages` has no caller** (`usage.go:161` — the only references are the interface,
the implementation and its own helper). The atlas says plainly: *"Nothing
user-facing consumes it yet, deliberately. #10's authoring step is the consumer."*
That consumer was never built.

Meanwhile `--harvest` pays a model call to AUTHOR a sentence for every deck word
(`renderAuthorPrompt`, `harvest_item.go:370`), then pays another to entailment-judge
it (`harvest.go:369`), for words that may already have real sentences sitting in
`usage/`.

#67 surfaced a third source: a passage the learner actually read and marked a word
in. It was deliberately kept out of #67 so this issue can serve all three sources
with ONE consumer rather than accumulating a second stalled producer.

## Spec

Needs a brainstorm. The shape is a consumer for `Usages`, not a new producer.

**A real sentence replaces the AUTHORING step, not the pipeline.** What it does not
supply, and what must still run:

- **Distractors.** `pickDistractors` (`harvest.go:390`) draws from the deck's BANDED
  words — *"a word cannot supply a distractor until"* it has been banded — and the
  veto then filters them. A real sentence changes none of that.
- **Quality judgement, and MORE of it than an authored sentence needs.** This is the
  finding that makes the issue non-trivial. `renderAuthorPrompt` states four
  requirements, two of which wild prose routinely fails:
  1. the sentence must POINT AT the word without defining it;
  2. it must **never gloss** it — *"not as an appositive, not as a relative clause,
     not as a contrast"*, a ban added because half of the first authored batch came
     back as *"the alewife, the small silver herring Alosa pseudoharengus"* — a
     reading test rather than a vocabulary test.

  Expository prose — the genre a learner is reading when they meet a hard word —
  **defines terms in place**. *"The slow precession of the equinox, the 26,000-year
  wobble of Earth's axis, …"* is exactly the banned appositive, and blanking
  `precession` there puts the answer beside the blank. So the judge that authored
  sentences pass is the judge real ones most need.

**Therefore the core question for the brainstorm:** is this a SELECTION problem
(judge candidate real sentences, keep the ones that work as stems, fall back to
authoring) or a REPAIR problem (trim the gloss out of a real sentence)? Selection is
simpler and honest about failure; repair salvages more material but re-introduces a
model call and a way to produce a sentence nobody wrote.

**Three sources, one shape.** `Usage` already exists as that shape, tagged by
provenance (`news`, `noad`). A passage sentence is a third tag. The consumer should
not know which it is beyond preferring by provenance.

**What "better" would mean, measured.** The claim to test is that a real sentence
produces a better item than an authored one. `harvest` already has the judges to
score both; this issue should say how it will know, not assert it.

## Done when

- `Usages` has a production caller, and a word with cached real sentences can reach
  a practice item without an authoring model call.
- A real sentence that glosses its own word is REJECTED as a stem, with a test whose
  fixture is an appositive of the shape the author prompt bans.
- Distractor selection and the veto are unchanged and still run — asserted, not
  assumed.
- A word with no usable real sentence falls back to authoring, and the fallback is
  covered.
- The model-call saving is measured rather than claimed.
- Provenance preference is stated and tested across `news`, `noad` and passage
  sentences.

## Plan

Awaiting brainstorm.

- [ ] brainstorm: selection vs repair
- [ ] `sdlc start-plan`, then a durable plan

## Log

### 2026-09-16

Split out of #67 during its planning. #67 originally carried a droppable task to
write a marked word's sentence straight into `store.Item{Form: FormCloze}` — the
`Stem` IS the example sentence there (`harvest.go:445`), so the datatype fit. Two
things killed it as a #67 task: distractors are not supplied by a sentence, and the
gloss problem above means a wild sentence needs the judge more than an authored one
does. Investigating that turned up the unused `usage/` cache, which is the same
problem one layer down — hence one issue, not two.

Corrects a note still visible in #67's history: *"an authentic sentence is better
material than an authored one."* It is more REAL, not automatically better as a
stem, and the difference is exactly the appositive ban.

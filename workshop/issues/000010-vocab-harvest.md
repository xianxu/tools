---
id: 000010
status: working
deps: ["tools#3", "tools#11", "tools#17"]
github_issue:
created: 2026-08-20
updated: 2026-09-04
estimate_hours:
started: 2026-09-04T09:38:50-07:00
---

# authored practice items: level-tagged words, and stems the model writes offline

## Problem

A multiple-choice question is only as good as its wrong answers, and asking a
model to invent them is how a distractor ends up being *also* correct. The fix is
to stop GENERATING distractors and start SELECTING them from real words at a
known level.

That needs two things this program does not have. **Words have no level**, so
"at the learner's band, or one below" is not yet arithmetic on anything — `#17`
assigns the LEARNER a band and nothing assigns one to a word. And **there are no
items to select into**: a raw sentence is not a question. Measured against the
live feed for `sycophantic` (2026-08-22, 100 items): blanking *"Sycophantic AI
decreases prosocial intentions"* produces a question its own sentence does not
entail, and 10 of the first 14 matching headlines were about AI chatbots, so the
learner would acquire the collocation rather than the word.

So the material has to be AUTHORED — read across usages, write a stem that
entails its answer — and authored **ahead of time**, because a review sitting
must stay instant, free and offline.

## Spec

Two deliverables and a store, all behind `define --harvest`, none of it on the
path a learner waits on.

**1. A band and a domain for every word, cached forever.**

- **CEFR is the single scale**, because `#17` already puts the LEARNER on it —
  "the learner's band, or one below" is then arithmetic on one 6-point scale
  rather than a mapping nothing reconciles.
- **The model is the source at both ends**, and there is no external asset. A
  downloadable CEFR or frequency list is DEFERRED, not rejected; the trigger for
  reaching for one is model-assigned bands proving too inconsistent to use.
- **Cached per word FOREVER**, which is what makes run-to-run fuzziness
  acceptable: assign once, stable after. A word's band is a fact about the word,
  not about the run.
- **No frequency signal.** The original Spec named one as the cheap level proxy
  and the project's own PRD separately argued distractors should come from the
  learner's register *"not from a generic frequency band"*. The two disagreed;
  register wins.

**2. Authored items, written offline and stored finished.**

- **The model writes the stem**, reading across usages from its own knowledge, so
  that the stem ENTAILS its answer. That is the job the measurement above says is
  real.
- **Distractors are SELECTED, never invented**: same domain (or general
  vocabulary), at the learner's band or one below. One band below rather than
  above, because a distractor the learner does not know is unrejectable — they
  eliminate it by ignorance rather than by knowing it does not fit.
- **Name real people, places and institutions.** A model asked for a natural
  sentence drifts to the neutral and unnamed, and an unnamed subject gives the
  learner no referent to attach the word to. Checkable: an LLM judge can score
  "would a person say this in ordinary conversation", and topic spread across a
  batch is measurable with no model at all.
- **Learner-aware**: reads `#17`'s `user-model.md` so items are pitched at the
  right level and drawn from the domains the learner actually reads in. An absent
  file means generic authoring, not an error.

**3. The store**, in the working directory beside the deck (`#3`). Items are
prunable and inspectable; growth is bounded and pruning is deterministic.

**`#9` (the news seam) is RETAINED and OPTIONAL.** Its specificity is a stronger
forcing function for named, current sentences than an instruction alone, so it
stays available as enrichment. It is no longer the spine, and nothing depends on
it.

**What is LOST, recorded rather than discovered: provenance.** The original Spec
wanted source URL, fetch date and sentence *"because a bad pool is the failure
mode"*. A model-authored item has no external artifact to inspect when the pool
goes bad — the item itself is the only evidence. That raises the bar on the
verification below rather than lowering it.

**This is the project's material-quality checkpoint.** Stop here and read
generated items before building the forms that consume them (`#12`, `#13`).

### Two decisions, 2026-09-04

**1. Banding is measured by STABILITY, not against hand labels.** The Done-when
below names both and calls the hand-labelled sample "the honest option"; the
operator declined the ~100 words of labelling, which the row explicitly provides
for ("or, if that is refused"). So this is inside the Spec rather than a
departure from it.

**What that buys:** the property the cache actually depends on. A band is assigned
once and reused forever, so what must hold is that the model gives the same word
the same band — and stability is exactly that, measurable with no human input.

**What it does NOT buy, recorded rather than discovered later:** stability cannot
detect a model that is confidently and CONSISTENTLY wrong. Every downstream use of
a band — "the learner's band, or one below", which is the whole distractor rule —
rests on the scale being right, and a uniformly skewed scale passes a stability
check perfectly. The measure will therefore be reported as what it is: agreement
across repeated assignment, with no claim about correctness. If distractors later
read as mispitched, this is the first thing to suspect and the hand-labelled
sample is the thing to build.

**2. Two milestones, and the stop between them is real.**

- **M1** — bands and domains per word, cached forever, plus the store. Closes on
  its own.
- **M2** — authored stems and the distractor veto. **STOP HERE** and read real
  generated items before `#12` and `#13` consume them.

The stop is the project's own instruction ("stop here and read the output before
building the forms that consume it"), and it is why this is milestoned rather than
single-pass: a lone boundary would put the first review over seven Done-when rows
of greenfield code with an LLM judge in it, and would let the checkpoint pass as a
formality.

## Done when

- [ ] `define --harvest` produces finished items with no review session running,
      and a sitting never waits on it.
- [ ] Every word carries a CEFR band and a domain, assigned once and cached; a
      second run over the same word re-reads rather than re-asks.
- [ ] **The banding is measured, not asserted** — and the measure is named,
      because the model is its own source. Either a hand-labelled held-out sample
      (the honest option, and small: ~100 words settles it), or, if that is
      refused, a STABILITY measure over repeated assignment, which is what the
      cache actually depends on. A row that says "checked" without naming the
      reference is the one shape this row may not take.
- [ ] A distractor is never the answer: an LLM judge vetoes a candidate that
      would also fit the stem, and the veto is exercised by a committed
      known-bad case.
- [ ] Authored stems ENTAIL their answers, and name real subjects — judged on a
      batch, with the topic-spread half measured without a model.
- [ ] A model outage leaves the existing store usable and untouched; harvesting
      is the only thing that stops.
- [ ] Store growth is bounded; pruning is deterministic and tested.

## Plan

Durable design: `workshop/plans/000010-vocab-harvest-plan.md`.

**Two milestones, and the stop between them is the point** (see the decisions
above). Each `Mx` row closes with its own `sdlc milestone-close`.

- [ ] M1 — `WordFacts` (band + domain) and its store surface, on the `Store`
      interface with `storetest` rows so `Mem` and `YAML` cannot diverge; `facts`
      joins `RuntimeDirs`.
- [ ] M1 — `bandTask`, one call per word for both facts, with a golden and a live
      conformance row.
- [ ] M1 — the STABILITY measure (`agreement` over N assignments), reported by
      `--harvest` rather than buried, and documented as agreement-not-correctness.
- [ ] M1 — `--harvest` dispatched as a MODE beside `--forget`; a sitting never
      waits on it (model seam PANICS in the test), a second run makes zero calls,
      and an outage leaves the store untouched.
- [ ] M2 — authored stems: entailment and named subjects as prompt REQUIREMENTS,
      learner-aware via `#17`'s model, absent file meaning generic not error.
- [ ] M2 — `topicSpread` measured with NO model, and a batch entailment judge with
      a committed known-bad stem.
- [ ] M2 — distractors SELECTED at band-or-one-below, vetoed by a judge exercised
      on a committed known-bad case; and a recorded answer on whether
      `play.PickOptions` is the same rule (ARCH-DRY).
- [ ] M2 — `prune`: bounded, deterministic, tested by pruning twice.
- [ ] M2 — **generate a real batch and READ it** before `#12`/`#13`. The
      checkpoint, and the one row no test replaces.

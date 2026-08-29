---
id: 000010
status: open
deps: ["tools#3", "tools#9"]
github_issue:
created: 2026-08-20
updated: 2026-08-20
estimate_hours:
---

# news word harvester: async level-aware vocabulary pool from the news

## Problem

A multiple-choice question is only as good as its wrong answers, and asking a
model to invent them is how a distractor ends up being *also* correct.

The fix is to stop generating distractors and start **selecting** them from a
curated pool of real words. That pool has to come from somewhere, it should be
current, and it should sit at the learner's level — which means harvesting it
from the news ahead of time rather than at question time.

## Spec

An asynchronous harvester that maintains a pool of candidate words.

- Pulls news via the seam (#9) on a schedule, independent of any review session,
  so a session never waits on the network.
- Extracts candidate words from headlines and descriptions; keeps the sentence
  each came from, since #12 needs a real sentence anyway.
- **Level-aware.** A distractor far above or below the learner's level is not
  plausible and teaches nothing. Level signal, cheapest first: word frequency
  (a frequency list is a static asset, no model needed), length/morphology, and
  whether neighbouring deck words are known. Calibrate against the learner's own
  deck — the words they look up *are* their level, which is a signal this tool
  has and a generic vocabulary app does not.
- Pool stored in the brain (#3) with provenance: source URL, fetch date,
  sentence. Prunable and inspectable, because a bad pool is the failure mode.
- Runs via `define --harvest` and, later, on the nous service rhythm.

## Done when

- [ ] Harvest populates a pool with sentence + provenance, without a review
      session running.
- [ ] Words are bucketed by level, and the bucketing is checked against a held-out
      sample rather than asserted.
- [ ] A network outage leaves the existing pool usable and untouched.
- [ ] Pool growth is bounded; pruning is deterministic and tested.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

## Revisions

### 2026-08-22 — the harvester authors finished items, and reads the learner

**Reason.** Measured against the live feed for `sycophantic` the same day (100
items, 14+ containing the word): the feed **collapses thematically** — 10 of the
first 14 matching headlines were about AI chatbots — and headlines do not work as
stems. Blanking *"Sycophantic AI decreases prosocial intentions"* produces a
question its own sentence does not entail. A pool of words plus a raw sentence is
not enough raw material to make a question worth answering.

**Delta.**

- **Output is finished, verified practice items**, not a word pool: authored stem,
  selected options, provenance (source URL, date, the usages it was written from),
  and the verdicts of whatever filters ran. Prunable and inspectable, as the pool
  was.
- **The model authors the stem** by reading *across* usages and writing one that
  entails its answer — which is the job the measurement above says is real.
  Distractor sourcing is unchanged (see #12's revision).
- **Authoring happens here, offline and ahead of time**, never at question time.
  This is what keeps a review session instant, free and network-independent, and it
  is why the whole pipeline sits behind `--harvest`.
- **Learner-aware.** Reads `user-model.md` (#17) so items are pitched at the right
  level and drawn from the domains the learner actually reads in. Absent file →
  generic authoring, not an error.
- **Level classification gains a model step.** Frequency (a static list) as the
  coarse prefilter; the model labels register and domain, cached per word
  indefinitely; the learner's own deck anchors the band. The Done-when below
  already demands the bucketing be checked against a held-out sample rather than
  asserted — that stands, and now covers the model's labels too.

**Unchanged.** Asynchronous and independent of any review session; provenance
required; a network outage leaves the existing store usable and untouched; growth
bounded and pruning deterministic.

**This is the project's material-quality checkpoint.** Stop here and read generated
items before building the forms that consume them.

### 2026-08-28 — the pool is a static asset; the feed becomes optional

**Reason.** Operator, after `#23` shipped and the news feed turned out to be
unconsumed plumbing:

> we can for now use model to propose sentences, not constrained by google news

and, on distractors:

> distractor should just be selected randomly from ... vocabularies at same
> "grade level" or "general domain" ... we select words at same or one level
> below (to focus on learning those words, not the distractors)

Recorded in full in the project's `## Decisions`, 2026-08-28.

**Delta.**

- **The candidate pool no longer comes from news.** It is a static
  level-and-domain-tagged vocabulary plus the learner's own deck. That deletes
  the harvest spine — the scheduled pull, the extraction, the provenance record
  — from this issue's critical path.
- **The Spec's frequency signal is dropped.** *"word frequency (a frequency list
  is a static asset, no model needed)"* contradicted the project's own PRD, which
  argued distractors should come from the learner's register *"not from a generic
  frequency band"*. **CEFR** is the single scale; `#17` already assigns the
  learner a band, so word bands make the two ends commensurable. Assigned by the
  model, cached per word forever, which is what makes run-to-run fuzziness
  acceptable.
- **What this issue still owns:** assigning each word a CEFR band and a domain,
  cached forever; and the authoring step, which is unchanged in kind but now
  reads from the model's own knowledge rather than requiring a news pull.
- **Authoring gains an explicit constraint:** name real people, places and
  institutions. A model asked for a natural sentence drifts to the neutral and
  unnamed, and an unnamed subject gives the learner no referent to attach the
  word to. This is checkable — an LLM judge can score "would a person say this in
  ordinary conversation", and the topic spread across a batch is measurable with
  no model at all.
- **`#9` is retained and OPTIONAL.** Its specificity is a stronger forcing
  function for named, current sentences than an instruction alone, so it stays
  available as enrichment rather than as the spine. Nothing is deleted.
- **What is LOST, recorded rather than discovered:** provenance. This Spec wanted
  source URL, fetch date and sentence *"because a bad pool is the failure mode"*.
  A model-authored item has no external artifact to inspect when the pool goes
  bad — the item itself is the only evidence.

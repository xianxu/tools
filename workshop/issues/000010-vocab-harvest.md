---
id: 000010
status: open
deps: ["tools#3", "tools#11", "tools#17"]
github_issue:
created: 2026-08-20
updated: 2026-08-31
estimate_hours:
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

- [ ] Design via `sdlc start-plan` before implementing. Two separable
      deliverables — the word→(band, domain) index, and the authoring pipeline —
      which the plan should decide to split or not. The index is independently
      useful (`#8`'s stats, `#22`'s active learning set), which is the argument
      for two milestones; they share one store and one model seam, which is the
      argument against.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

## Revisions

### 2026-08-31 — the Spec is rewritten to what two revisions had already decided

**Reason.** Operator: *"can you clean up this issue as we have changed it
significantly. or you think we should make a new issue and punt old one?"*

**Cleaned up IN PLACE rather than re-filed, and the reasoning is worth keeping.**
It is the same job — produce practice material offline, ahead of time, level- and
learner-aware — with a different source (news → the model's own knowledge) and a
different level signal (frequency → CEFR). A punt would read as "we decided not
to do this", which is false. The ID is also load-bearing in five places: `#12`'s
and `#18`'s `deps`, plus prose in `#12`, `#13` and `#18`, plus the project's
`mvp_scope` and Breakdown. And nothing is orphaned — this issue is `open`, never
claimed, with no branch, plan or estimate, so the append-only rule (which exists
to protect decisions made DURING work) protects nothing here.

**Delta, all of it applying decisions already taken on 2026-08-22 and
2026-08-28 rather than making new ones:**

- **Title** was "news word harvester: async level-aware vocabulary pool from the
  news". The news is neither the source nor required.
- **Spec** described the scheduled pull, the extraction, provenance and a
  frequency list — every one of which the 2026-08-28 revision removed from the
  critical path. Rewritten as the two deliverables that survived.
- **Done-when row 1** demanded "a pool with sentence + provenance", which the
  same revision records as LOST. Replaced.
- **Done-when banding row** said the bucketing is "checked against a held-out
  sample rather than asserted" and named no reference to check against — and with
  the model as its own source there is none by default. That was the one real
  hole in the design, and the row now forces the choice: a hand-labelled sample,
  or an honest stability measure. **A row that says "checked" without naming the
  reference is unfalsifiable**, which is the shape this repo's guards exist to
  refuse.
- **`deps`** were `["tools#3", "tools#9"]`. `#9` is now optional by decision, so
  declaring it is wrong even though it is done — a dependency records what
  BLOCKS. Now `#3` (the store), `#11` (the LLM harness, which the authoring step
  cannot run without) and `#17` (the learner model it reads).

**Both earlier revisions are kept below**, and the decisions themselves live in
the project's `## Decisions` for 2026-08-22 and 2026-08-28, which is where the
2026-08-28 entry already points.

**Not decided here, deliberately:** whether the word index and the authoring
pipeline are one milestone or two. That is a plan-time call and the Plan row says
so.


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

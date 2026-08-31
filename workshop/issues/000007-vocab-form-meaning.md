---
id: 000007
status: working
deps: ["tools#6"]
github_issue:
created: 2026-08-20
updated: 2026-08-30
estimate_hours: 2.81
started: 2026-08-30T17:13:15-07:00
---

# review form 2.3: meaning multiple choice from the local deck

## Problem

Recognition needs testing, not just recall. And the whole review loop should
work with no API key and no network.

## Spec

Form 2.3: given a word, choose its definition from four options.

- **Distractors come from the learner's own deck**, not from a model: offline,
  free, deterministic, and pedagogically better — the wrong answers are words
  they are actually confusing right now.
- Pick distractors that are *far* in meaning from the answer (see #12's
  selection rule), so the question has one defensible answer.
- Pure question construction: deck + target + seed → question. Same seed, same
  question, which is what makes it testable.
- This form plus #6 is a complete trainer needing neither network nor key. That
  is the M1 boundary.

**Record the CHOSEN distractor, not just right/wrong.** This is #17 M2's error
taxonomy arriving for free, and it is the reason M2 was not filed as its own
issue when #17 closed at M1.

The learner model (`user-model.md`, shipped) carries Level and Domains, and #17
reserved a third section — Weaknesses — for misses classified by KIND
(near-synonym collapse, connotation, register, domain). That was blocked on
review events, which now exist; but `store.ReviewEvent` records `Correct bool`,
a binary verdict that cannot carry a kind. Classifying post-hoc would mean a
model call per miss.

In THIS form the classification is already in the learner's hand: the distractor
they picked IS the error kind, since distractors are drawn from their own deck
and selected by semantic distance. Recording which one was chosen costs a field
and no inference. Whatever this form does with the answer, keep the identity of
the chosen option rather than collapsing it to a boolean — the weakness section
is downstream of that one decision.

## Done when

- [ ] Four options, exactly one correct, drawn from the local deck.
- [ ] The CHOSEN option is recorded, not just correctness — see the note above;
      collapsing it to a boolean is what makes the weakness taxonomy expensive
      later.
- [ ] Deterministic under a fixed seed.
- [ ] Degrades sensibly when the deck has fewer than four words.
- [ ] Works with the network off.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.45 impl=0.1
item: greenfield-go-module     design=0.05 impl=0.24
item: smaller-go-module        design=0.03 impl=0.2
item: greenfield-go-module     design=0.06 impl=0.32
item: smaller-go-module        design=0.02 impl=0.12
item: cross-cutting-refactor   design=0.03 impl=0.14
item: greenfield-go-module     design=0.05 impl=0.24
item: smaller-go-module        design=0.02 impl=0.16
item: atlas-docs               design=0.03 impl=0.06
item: milestone-review         design=0.0  impl=0.16
item: milestone-review         design=0.0  impl=0.12
item: milestone-review         design=0.0  impl=0.1
design-buffer: 0.15
total: 2.81
```

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.* The calibration doc is tagged **stale** by
`sdlc estimate-source` (the ledger is newer; recalibration is `#127`), so the
per-primitive hours are provisional — recorded because a close-time comparison
against a provisional baseline should know it was provisional.

| item | task | why this primitive |
|---|---|---|
| `issue-spec` 0.45/0.1 | the design carrier | spec, plan doc, three plan-quality rounds. **Measured, not allocated** — see below |
| `greenfield-go-module` 0.05/0.24 | T1 `senseLabel`, `noadLabels`, `excludeCrossReferenced` | new single-concern parsing over dictionary prose |
| `smaller-go-module` 0.03/0.2 | T2 `Choice`, `Option`, `Axis` | MIRRORS `Recall`; impl at the top of the range for D5a's import-free constraint |
| `greenfield-go-module` 0.06/0.32 | T3 `pickOptions`, `shuffle` | axis priority, determinism, hand-rolled PRNG, small-deck degradation |
| `smaller-go-module` 0.02/0.12 | T4a `ReviewEvent` field + the `At` ordering | the store half, pinned by the torn-record test |
| `cross-cutting-refactor` 0.03/0.14 | T4b `Outcome`, `CaptureReview`, `Apply` | the widening across the session seam |
| `greenfield-go-module` 0.05/0.24 | T5a the pool + seeded sampling under a cap | **not an extension**: ARCH-CONSTRAINTS requires a sampling mechanism that does not exist yet |
| `smaller-go-module` 0.02/0.16 | T5b `TestASittingFallsBackToRecall`, `TestSittingWithNoModelAndNoNetwork` | sitting-level integration, not unit |
| `atlas-docs` 0.03/0.06 | T6 | README section, atlas, project row |
| `milestone-review` 0.0/0.16 | the boundary, run | |
| `milestone-review` 0.0/0.12 | the boundary, remediate | a boundary rarely clears in one round; `#30` booked both lines per boundary and still overran |
| `milestone-review` 0.0/0.1 | the manual verification pass | "Verification before close" is a real terminal, a deliberate wrong answer, event-log inspection, a network-off sitting, `-race` and `-tags conformance`. `#30` gave exactly this its own line |

**Design is MEASURED here, not allocated.** `sdlc actual --issue 7` reports
**0.18h** attributed to `#7` over the claim-to-estimate window
(`3c098707` 17:13 → 17:45, 0.53h elapsed and shared with `#30` and `#38`). The
first version of this block booked **1.35h** of design — smeared as round
`0.3`/`0.2`/`0.5` allocations across the task lines, against a window that
cannot contain it. It is now one measured `issue-spec` carrier at 0.45, with the
per-task design lines at `0.02`–`0.06`, because the plan doc has already
pre-resolved the decisions those lines would have paid for: D6/D7/D8 settle T4
entirely, D1a/D4/D5a settle T2, D2/D3 settle T1. Design that is already spent is
not design that remains.

**Why the total barely moved, and why that is the interesting part.** 2.85 →
2.81. But design fell 1.35 → 0.74 and impl rose 1.30 → 1.96, and the
redistribution is the whole point. `#30` estimated 3.19h and measured **10.91h
(3.4×)** across 13 boundary rounds; the open question that overrun poses for
`#127` is whether v3.1's *implementation* scale — impl written at 40% of the v2
table — is too aggressive for `define`-sized work. A row can only answer that if
impl is the line carrying the estimate. The first version raised **design**, the
line that is already spent and shrinkable, and left impl *below* `#38`'s 1.40 —
so however it landed, it would not have isolated the variable. This version sits
above `#38` on impl and below it on design, which makes it a clean same-unit
test.

Still no private correction factor, and deliberately: applying one would corrupt
the ledger that exists to detect the bias. If this lands near 3× again, that is
two consecutive rows against the impl scale, and the pair is worth more to `#127`
than either row alone.

## Plan

Single-pass: plain checkboxes, ONE boundary (AGENTS.md §3). The full task detail,
the ten decisions and the eight Done-when rows live in
`workshop/plans/000007-vocab-form-meaning-plan.md`.

- [x] Design via `sdlc start-plan` — plan doc written, cleared plan-quality in 3 rounds.
- [ ] **T1** — `senseLabel`, `noadLabels`, `excludeCrossReferenced` in `cmd/define/glosslabel.go` (D2, D3, D3a).
- [ ] **T2** — `Option`, `Axis`, `Choice` in `cmd/define/play/choice.go`, import-free (D4, D5a).
- [ ] **T3** — `pickOptions` + `shuffle`: axis priority, seeded determinism, small decks (D1, D2a, D4a).
- [ ] **T4a** — `ReviewEvent` gains the axis field ABOVE `At`; torn-record test pins the order (D6).
- [ ] **T4b** — `Outcome`, `CaptureReview` and `Apply` carry the choice out (D7, D8).
- [ ] **T5a** — `todaysQuestions` builds the pool, sampled under the seed within the cap (ARCH-CONSTRAINTS).
- [ ] **T5b** — sitting-level tests: fallback to `Recall` on a small deck, and a network-off sitting.
- [ ] **T6** — `cmd/define/README.md`, `atlas/define.md`, the project row.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

### 2026-08-27

- #17 closed at M1. Its M2 (weaknesses, from an error taxonomy) was NOT filed as
  a separate issue: the data it needs does not exist yet, and this form produces
  it for free. Recorded above as a Done-when row rather than a tracked issue, so
  it is met at the moment it is cheap instead of becoming a backlog item whose
  first design question is "wait for #7".


## Revisions

### 2026-08-28 — the same rule, and a tension this form has to settle

**Reason.** The project's 2026-08-28 decision changes the distractor rule to
**same domain (or general vocabulary), at the learner's CEFR band or one below**.
This Spec says *"pick distractors that are far in meaning from the answer (see
#12's selection rule)"*, so it inherits by reference.

**Delta.**

- **Inherits the new rule.** Domain and band replace semantic distance as the
  primary filter. The pool for this form is unchanged — the learner's own deck —
  so nothing here depends on `#10`'s harvest.
- **This form stays no-LLM, and that is now a positive result rather than an
  assumption.** `#12` can afford plausible distractors because it has a model
  veto. This form has none by design. It is safe anyway, because the options here
  are DEFINITIONS: two different words rarely share one, so a same-domain
  distractor does not create a second correct answer the way it does in a cloze
  blank. Domain-matching here only removes the giveaway where three options are
  obviously medical and one is legal.

**An OPEN TENSION this issue must settle when it is built — do not resolve it by
inheriting.** Two claims in this Spec pull against each other, and the new rule
sharpens the conflict rather than causing it:

1. *"Pick distractors far in meaning"* — which the near-synonym guard requires.
2. *"The distractor they picked IS the error kind"*, with the kinds named as
   near-synonym collapse, connotation, register, domain.

If every distractor is far in meaning, choosing one says only *"did not know
it"*, never which KIND of confusion — so the error taxonomy `#17 M2` was folded
into this issue for cannot be read off the choice. Making the option set
diagnostic needs distractors that VARY along those axes, which means deliberately
including a near-synonym — and a near-synonym's definition may plausibly define
the target, which is the ambiguity the guard exists to prevent.

Same-domain selection moves this form toward diagnosis and toward ambiguity at
the same time. Whether the taxonomy survives, or `#17 M2` needs its own path
after all, is a question for this issue's design — it should be answered
deliberately rather than discovered at a review boundary.

### 2026-08-30 — the open tension is SETTLED: safe axes only

**Reason.** The 2026-08-28 revision left this for the build and said to answer it
deliberately. Answered, by the operator, on a measurement.

**The measurement.** The four error kinds `#17` named are not equally available
offline, and not equally risky. NOAD labels two of them ITSELF, inline in the
sense text — counted over the committed corpus: 51 `informal`, plus `formal`,
`archaic`, `dated`, `dialect`, `rare`, `humorous`; and domain labels `Law`,
`Grammar`, `Nautical`, `Music`, `Military`, `Computing`.

| kind | derivable with no model? | ambiguity risk |
|---|---|---|
| domain | YES — NOAD's own label | none: a `Law` sense defines something else entirely |
| register | YES — NOAD's own label | none |
| connotation | no — needs semantics | moderate |
| near-synonym collapse | no — needs semantics | HIGH — its definition may genuinely fit the target |

The two axes that are free are exactly the two that are safe, and the two that
need a model are exactly the two that create the second-defensible-answer problem
the far-in-meaning guard exists to prevent. That is what dissolves the tension
rather than trading one horn for the other.

**DECIDED.**

- **The far-in-meaning guard STAYS.** No distractor is a near-synonym, so every
  question keeps exactly one defensible answer — which this form needs, because
  unlike `#12` it has no model veto by design.
- **The three distractors are CHOSEN to vary along the labelled axes**: one from
  another domain, one from another register, one general. A miss therefore still
  carries information — "picked the `Law` one" is not the same event as "picked
  the `archaic` one" — without any option plausibly defining the target.
- **`#17 M2`'s taxonomy arrives REDUCED, and that is now a stated outcome rather
  than a discovery.** This form can produce *domain confusion*, *register
  confusion* and *did not know it*. It CANNOT produce near-synonym collapse or
  connotation, and no amount of care here will make it: those need semantic
  closeness, which needs a model. They belong to `#12` (cloze, which has a veto)
  or `#13` (free sentence, graded). The Revisions asked whether the taxonomy
  survives or `#17 M2` needs its own path — the answer is BOTH, split by kind.
- **Recording the chosen option is unchanged and still the load-bearing bit.**
  The reduced taxonomy is readable only because the event keeps which option was
  picked; collapsing to a boolean forecloses it exactly as `#17`'s close warned.

---
id: 000017
status: open
deps: [tools#3, tools#11]
github_issue:
created: 2026-08-22
updated: 2026-08-22
estimate_hours:
---

# learner model: batch analysis into a durable user-model.md

## Problem

Nothing in the trainer knows who it is teaching. `--stats` (#8) counts answers;
it does not know that this learner reads judicial opinions for pleasure, sits at
C1–C2, and consistently mistakes contemptuous praise words for neutral ones.

Without that, every generated item is generic — and generic is exactly what a
personal tool has no excuse to be. The words a person looks up *are* a signal a
vocabulary app cannot buy.

## Spec

One durable artifact, `user-model.md`, in the same working directory as `words/`
and `events/`. Batch-generated, human-correctable, and read by every authoring
prompt.

### Why a document, not a table

It is meant to be read and argued with. A learner who disagrees ("I read these for
pleasure, not for the bar exam") must be able to say so and have it stick — which
is also the escape hatch for a batch analysis that draws the wrong conclusion.

### Shape

```markdown
---
type: user-model
learner: <name>
updated: <ISO date>
window: <from>..<to>          # N lookups, M reviews
generated_by: define --reflect (<model>)
---

## Level
Working band, anchored on evidence from the deck rather than asserted.

## Domains they read in
Inferred from what gets looked up — never asked. Table of domain, share, and the
words that are the evidence. Carries the authoring directive that follows from it.

## Weaknesses
Error kinds with counts, each naming the events it was derived from.

## Corrections          ← human-owned
```

### Rules

- **Batch, never per-answer.** `define --reflect` folds the event log and the deck
  into this file on demand. No model call ever sits inside the review loop or the
  lookup path; review stays instant and works offline.
- **`## Corrections` is never rewritten.** Regeneration replaces everything above
  it and appends nothing to it. Corrections are authoritative over anything
  inferred, and the authoring prompt is told so explicitly.
- **Every claim names its evidence.** "34% of lookups are legal — `certiorari`,
  `dicta`, `estoppel`" is checkable; "you like law" is not. A claim that cannot
  name the events behind it does not go in the file (`lessons.md`: a claim must not
  outrun the width it was measured at).
- **Regeneration is idempotent under a fixed clock, fake seam and fixed store.**
  Same inputs, same file — which is what makes it testable at all.
- **Absent file is normal.** Authoring works without it, just generically; nothing
  blocks on it existing.

### Milestones

- **M1 — from lookups.** Level and domain need only the deck and the lookup events,
  both of which exist today (#3, #4). Lands before any review events do, so the very
  first authored item is already learner-aware.
- **M2 — weaknesses.** The error taxonomy needs review events from #6, and the
  misses classified by kind (near-synonym collapse, connotation, register, domain,
  right-meaning-wrong-usage). Feeds back into which words surface and what the next
  item is authored to probe.

## Done when

M1:
- [ ] `--reflect` writes a `user-model.md` whose level and domain claims each name
      the deck words they were derived from.
- [ ] Regeneration is idempotent against a fake seam, fixed clock and fixed store.
- [ ] A hand-written `## Corrections` section survives regeneration byte-for-byte,
      asserted by a test that fails when the preservation is removed.
- [ ] Authoring (#10) reads the file, and its absence degrades to generic authoring
      rather than an error.
- [ ] Domain inference is checked against a held-out sample of deck words, not
      asserted — the same bar #10 sets for level bucketing.

M2:
- [ ] Misses classify into a fixed, enumerated error taxonomy; an unrecognised
      classification is dropped with a warning, not admitted as a new kind.
- [ ] A weakness claim names the events behind it, and the count is reproducible
      from the log.
- [ ] The model demonstrably changes what is authored: same deck, two different
      `user-model.md` files, different distractor domains — asserted, not asserted-ish.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-22

Created from the operator conversation that broadened `define-learn` to an adaptive
program: *"keep one user model of where user is, what area/words they seem to know
more… if the words appear more in supreme court cases, then the comparables and
wrong options would be drawn from those areas more. the whole thing should be
adaptive."*

Batch analysis over accumulated errors was specified in preference to diagnosing
each miss as it happens.

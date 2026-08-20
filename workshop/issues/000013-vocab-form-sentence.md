---
id: 000013
status: open
deps: ["tools#6", "tools#11"]
github_issue:
created: 2026-08-20
updated: 2026-08-20
estimate_hours:
---

# review form 2.4: free sentence graded by the model

## Problem

Recognition is not production. The last step is using the word yourself.

## Spec

Form 2.4: the learner writes a sentence; the model grades it.

- Prompt asks for a **structured verdict plus a one-line reason** the learner can
  read and disagree with. A bare pass/fail is not reviewable and not debuggable.
- Optionally seeded with a current event from the pool (#10) so the sentence has
  something to be about.
- Grading is a judgment call, so treat disagreement as data: a "that was wrong"
  keypress records the sentence, verdict and reason for later prompt work.
- Unavailable seam → this form is skipped, not failed (#11).

## Done when

- [ ] A graded round trip runs against the fake, with the verdict parsed
      structurally.
- [ ] A malformed or refused response skips the question without losing the
      session.
- [ ] Disagreement is recorded with enough context to improve the prompt.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

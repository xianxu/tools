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

> **OUT of the define-learn MVP as of 2026-09-07** (operator decision; see that
> project's scope event). The project's done-when never referenced this form, so
> its removal changed nothing there. It stays open and filed — but see the
> revision below: the shape the operator wants is not the one specified here.


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
- [ ] **An unavailable seam skips this form entirely**, and the session continues
      with the local forms. Distinct from the row above and not covered by it:
      malformed/refused means the model answered badly, unavailable means it was
      never reached, and only the second one is the ordinary offline state.
      Relocated from #11 on 2026-08-22.
- [ ] Disagreement is recorded with enough context to improve the prompt.

## Plan

- [ ] Design via `sdlc start-plan` before implementing.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

## Revisions

### 2026-09-07 — out of MVP, and the shape is superseded

**Reason.** Operator, choosing what remained: *"let's move #13 out of mvp."* The
`define-learn` done-when never referenced form 2.4, so this costs the project
nothing it had promised.

**The bigger reason is that the Spec above is not what is wanted.** From the
operator, 2026-09-06:

> *"often it's tricky for people to know when to use it, or to use it in abstract
> is not very useful. rather, maybe use LLM to describe something or some
> scenario where those words can be easily used. not sure if that's possible? so
> not just write a free form sentence, but rather construct some context/chat/
> essay that may make use of such words, either as LLM 'speaks' using those
> words, or as user replying/commenting using those words."*

and, extending it:

> *"given those set of words, and the events, construct some context/chat/essay
> that may make use of such words, either as LLM 'speaks' using those works, or
> as user replying/commenting using those words."*

**Delta.** The unit of practice is a SET of words in a constructed situation, not
one word in an isolated sentence. Two directions rather than one — the model
writes using the words (input flooding: the learner reads them working), or the
learner replies using them (production, with something concrete to react to).
Grounded in the news pool `#10` already fills, which is what makes the situation
current rather than invented.

**Not re-specced here.** That is a brainstorm, then a plan, and it is bigger than
this issue as filed — which is exactly why it is out of the MVP rather than
blocking it. What is preserved is the requirement that survives either shape: a
structured verdict plus a one-line reason the learner can read and DISAGREE with,
and the disagreement recorded as data.

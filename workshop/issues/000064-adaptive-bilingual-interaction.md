---
id: 000064
status: open
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# define: adaptive bilingual interaction level, learned or instructed, remembered per deck

## Problem

define's model features assume one learner profile: an advanced English reader
learning English words. The console-question prompt says "You are helping someone
build their English vocabulary" and carries no language or level. The learner
model grades on A2–C2 and never names the deck's language. With a Spanish deck
that is wrong for the user, a beginner, and for their daughter at Spanish 3. They
need very different amounts of English, and the right amount changes as they
progress.

#61 adds a per-deck `/bilingual on|off` switch and English help while practising.
That is a binary version of what is needed. This issue generalizes it.

## Spec

Draft from the user's request (2026-09-15); needs a brainstorm before a plan.

**Interaction stages for a non-English deck**, as the user described them:

- a. Ask in English; replies in English with target-language words sprinkled in
  (typical request: "how do you say this in Spanish, just the words").
- b. Ask in English; replies in simple target language, with sentences growing
  longer over time.
- c. Ask in the target language; replies in the target language with English
  annotation.
- d. Target language to target language only.

**Per deck, per language.** Each deck directory records its own stage and
preference, so two learners with two decks get two profiles. A switch with
`/lang` reads that language's record.

**Learned and instructed.**

- Inferred: the model estimates the stage from how the learner asks, including
  question language, request shape, and lookups, and records it with its
  evidence. The stage moves as the evidence moves.
- Stated: a natural-language instruction such as "I'm in Spanish 3" or "answer
  only in Spanish" is recorded as the learner's own statement. It wins over
  inference until the learner says otherwise, following the learner model's
  existing human-owned Corrections convention.
- The learner can always see what is recorded and change it.

**Consumers derive from the one record** (ARCH-PURPOSE):

- console answers follow the stage's reply-language policy;
- practice English help (#61) follows it, for example full help at stage a and
  none at stage d;
- authored practice sentences are pitched at the recorded level, in the deck's
  language.

**Spanish generation quality** is part of the scope. Authoring, entailment and
veto prompts and the learner model must work in the deck's language, at
beginner levels (A1 is missing today). A live conformance check should cover
Spanish output.

Open questions for the brainstorm:

- whether `/bilingual` becomes a view of the stage or stays a separate override;
- where the record lives (the learner model file or a sibling setting);
- how an instruction is detected, from a model side channel or a command;
- how quickly inference may move a stage, and how the move is announced.
- **whether an answer should be tinted at all** — inherited from #72, 2026-09-17.
  `renderAskPrompt` names a study language but never requests a REPLY language,
  and `/bilingual` does not reach the ask path: the request is byte-identical
  with it on and off. So the answer tint colours a language the model
  self-reports, unlike the definition tint, which comes from verified
  `dictionarySourceLanguage` metadata known before a byte is painted. Once a
  stage sets the reply language, the annotation describes something that was
  ASKED for and the tint becomes a fact rather than a claim. Decide it here,
  with the stage model in front of you, rather than twice.

## Done when

- A stated level or reply preference persists per deck and language across restarts, and wins over inference.
- The inferred stage updates from the learner's interactions and is visible to the learner.
- Console replies follow the recorded stage for a non-English deck; English decks behave as today.
- Practice help and authored material derive from the same record.
- Spanish authoring and answering pass a live conformance check at beginner and intermediate levels.

## Plan

- [ ] Brainstorm the stage model, record location, instruction detection and consumer policies; write the durable plan.

## Log

### 2026-09-15

- Filed from the user's request during #61. The user is a Spanish beginner; their
  daughter is at Spanish 3. The user noted that the bilingual behaviour should be
  learned from interaction or set by natural-language instruction, remembered per
  deck.
- Code facts at filing: `askSystem` hard-codes English vocabulary and
  `renderAskPrompt` carries no language. `reflectSystem` bands A2–C2 with no
  language named. `renderAuthorPrompt` states the language only in a header line;
  #61 adds an explicit requirement.

### 2026-09-17

#72 measured two facts this issue's brainstorm should start from.

- **`/bilingual` never reaches the ask path.** Verified by grep: no reference in
  `ask.go`, `askctx.go` or `passageprompt.go`, and `askContext` has no such
  field. The binary switch this issue generalizes does not currently apply to
  console answers at all, so stage work there is greenfield rather than a
  widening of #61.
- **Reply language is emergent, not requested.** The model infers it from the
  question's language and the context blocks. With English selected it often
  leaves its English prose untagged and annotates only a foreign fragment — one
  recording produced a longest span of three bytes — so even the annotation we
  DO ask for varies run to run. Any stage policy has to state the reply language
  in the prompt rather than assume the current behaviour is a floor.

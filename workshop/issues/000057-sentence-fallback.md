---
id: 000057
status: working
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-13
estimate_hours: 0.68
started: 2026-09-13T13:17:22-07:00
---

# define: route four-word dictionary misses to the LLM

## Problem

`so lickspittle is similar to sycophantic` reports no dictionary entry because
the fallback recognizes question/request openers but deliberately ignores length.

## Spec

After a dictionary miss, route input containing four or more whitespace-separated
words to the LLM. Existing question rules still admit shorter questions.
Successful dictionary phrases win; explicit `?` forces asking and `\` suppresses
fallback. Raw mode continues to suppress LLM requests. The operator approved
this precise heuristic in the conversation; no change to session context.

## Done when

- The reported sentence and a four-word miss route to the LLM; three-word misses
  remain misses unless existing question rules match.
- Tests preserve dictionary-first behavior and explicit/raw overrides.

## Estimate

Produced via the previously read estimate-logic-v3.1 calibration (provisional),
Method A. Clear approved spec: issue-spec design 0.5 × 0.2, impl 0.1 × 0.4;
smaller-Go-module design 0.3 × 0.2, impl 0.5 × 0.4; atlas design 0.05 × 0.2,
impl 0.1 × 0.4; one review impl 0.5 × 0.4. Existing predicate and test harness
cover the mechanism; no library discovery is needed. Familiarity 1, buffer 15%.

```estimate
model: estimate-logic-v3.1
familiarity: 1
item: issue-spec design=0.10 impl=0.04
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.01 impl=0.04
item: milestone-review design=0 impl=0.20
design-buffer: 0.15
total: 0.68
```

## Plan

- [ ] Add boundary and reported-sentence regressions to question/route tests;
  observe failures, then extend the shared readsAsQuestion predicate.
- [ ] Update comments, CLI help and user/atlas descriptions of the heuristic.
- [ ] Run focused and define-package verification; close through the review
  gate, publish and rebuild the local binary preserving installed features.

## Log

### 2026-09-13

- Created and claimed after the operator approved the four-word fallback.
  ARCH-DRY/PURE: change only the existing pure predicate, reached after the
  dictionary misses; preserve the route's shared override guards. No new IO,
  state, dependencies, or durable runtime artifacts. Linear existing whitespace
  tokenization handles the same bounded console input as before.
- The plan is a small, fully specified predicate extension, so use change-code's
  documented --no-judge path for trivial changes; the close review still runs.

---
id: 000054
status: working
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours:
started: 2026-09-12T17:16:34-07:00
---

# the TUI keeps practice material current in the background, so new words get cloze questions without a command

## Problem

Cloze questions exist only for words `define --harvest` has written a practice
sentence for, and nothing runs `--harvest` for you: `runHarvest` is reachable only
from its flag (`main.go:860`). The learner model is the same, written only by
`define --reflect` (`main.go:845`). A user who never learns those commands exist,
which is the likely case, sees only multiple choice and boards forever. A user who
does learn them has to remember to run them after every batch of lookups.
Operator, 2026-09-12: "user won't remember this."

## Spec

The interactive TUI, and only it, runs harvest in the background as the deck
grows, and writes or refreshes the learner model rarely. Not `-raw`, not a
one-shot `define word`, not piped input, and not the `--play`, `--harvest` or
`--reflect` modes themselves.

**Harvest: every 10 new deck words.** 10 is the operator's starting number, to be
tuned. A "new" word is one `--harvest` would still do work for: no band yet
(`facts.Harvested()`) or no practice item. The trigger derives that from the
store rather than keeping its own counter. A counter would be a second answer that
drifts when the user runs `--harvest` by hand, `--forget`s a word, or looks words
up from the command line. Counting reads a file per word, so how to count cheaply
(once at startup, then per new word in the session) is a plan decision.

**Learner model: rarely**, because a learner's level is stable (operator). Two
triggers:
- none exists and the deck has crossed `--reflect`'s 12-word floor. The first
  model matters most, since authoring without one is generic;
- the deck has grown a lot since the model was written. The threshold is a plan
  decision; the model's frontmatter already records `updated:` and the lookup
  window it was built from (`usermodel.go`).

When both are due, reflect runs first, because authoring reads the model at
harvest time (`readLearner`).

**Never in the way.** The existing rule is that no lookup or review waits on a
model, and it still holds:
- the work runs off the input loop, and a lookup, question or `/play` sitting
  never waits on it;
- one background job at a time. A trigger while one is running is dropped; harvest
  is incremental, so the next trigger picks the words up;
- quitting cancels it. Each word is saved as it's answered (`runHarvest`:
  "Everything banded before this point is already durable"), so a quit loses at
  most the word in flight;
- a sitting already under way doesn't see items written during it
  (`todaysQuestions` reads once); the next sitting does.

**Never writes where it shouldn't (#50).** It runs only when the deck question is
already decided and allowed. Otherwise `gatedStore` routes writes to `store.Mem`,
spending model calls on material thrown away at exit, or the background job
becomes the thing that asks "make this directory a deck?". It must never ask.

**Reports on screen, never on stderr.** The TUI owns a raw terminal, where stray
output breaks the display (the hazard #32 records). A batch that lands gets a
short line on screen, for example "10 new practice questions ready"; its shape is
a plan decision. With no model configured (no proxy, no key) it skips quietly,
with at most one notice per session. Lookups must not start nagging.

**Budget and an off switch.** Harvest costs about six model calls per new word:
band, author, entail, and one veto per wrong answer (`harvest.go:165`, `:280`,
`:303`, `:334`). A background run takes a small budget rather than `--limit`'s
default of 200 (`harvestLimit`), so a big backlog (an existing deck never
harvested) drains over several triggers instead of one long burst. Background
calls cost money for a user on an API key, so there is a documented way to turn
this off. With #52's local Apple model they would be free.

**Two processes, one deck.** Two TUIs in one directory, or a TUI plus a manual
`define --harvest`, could author the same word twice. Either a deck-level lock,
or a proof that duplicates are harmless (`Items()` is newest-first and prune keeps
the newest). Plan decision.

**Docs.** The README and atlas say "batch and on demand" and that nothing calls a
model while you look a word up. They need to describe the background work and
restate the rule as what still holds: nothing WAITS on a model. The operator has
an uncommitted README rewrite in progress (2026-09-12); do the README part after
it lands.

## Done when

- [ ] In the TUI, 10 new deck words trigger a background harvest that labels them
      and writes their sentences, and a `/play` sitting afterwards asks cloze
      questions for them. Asserted end to end against the fake model, not a
      stubbed trigger.
- [ ] The trigger counts from the store, so words looked up from the command line,
      harvested by hand, or forgotten are counted right. Asserted for each.
- [ ] No lookup, question or sitting waits on it: a test holds the fake model's
      reply and shows the TUI still answers input.
- [ ] Quitting mid-run cancels it and keeps every word already written.
- [ ] It never runs in an undecided or declined directory, and never asks the deck
      question.
- [ ] A learner model is written in the background once the deck crosses 12 words
      with none; refreshed only past the growth threshold; reflect runs before
      harvest when both are due.
- [ ] Nothing reaches stderr while the TUI owns the terminal; a batch that lands
      is reported on screen; a missing model gives at most one notice per session.
- [ ] A per-run budget and an off switch, both documented.
- [ ] Two processes in one deck don't author the same word twice, or the plan
      shows why that's harmless.
- [ ] The README and atlas describe the background work and the restated rule.

## Plan

Detailed plan: `workshop/plans/000054-background-harvest-plan.md`. Two review
boundaries. Code branches after PR #38 (#53) merges, because M1's docs edit the
README rewrite it carries.

- [ ] M1 — background harvest: one lock for every dictionary call; the harvest
      core returns a typed outcome (the CLI's output unchanged); the state
      machine, the job and its runner; the session wiring, notices and off
      switch; tests; docs.
- [ ] M2 — background reflect: the reflect core returns a typed outcome; when a
      model is due (none at 12 words, or the deck's lookups doubled); the job
      reflects before it harvests; tests; docs.

## Log

### 2026-09-12

Operator: users won't remember `--harvest` and `--reflect`. Have the TUI track new
words and harvest in the background, about every 10 words. The learner model can
refresh rarely, since a learner's level is stable.

Findings that shaped the spec:
- `runHarvest` and `runReflect` are reachable only from their flags
  (`main.go:845`, `:860`). `--harvest` is incremental (it skips labelled words and
  words with an item) and capped at 200 calls per run.
- Practice items are frozen when written (`store.Item` keeps the sentence and the
  wrong answers), and harvest never rewrites a word that has one, so a later
  learner model doesn't improve old items.
- Multiple choice is built each sitting from the live deck, so new words already
  appear there without any harvest.
- The TUI has one goroutine today (`scanLines` in `repl.go`, the stdin reader) and
  no surface for background jobs.

Related: #52 (a local model would make these calls free), #53.

Planning, after `sdlc claim` and `sdlc start-plan`. A read-only exploration
mapped the integration points; the facts that shaped the plan:

- The editor loop is one `select` over the context, resizes and keys in
  `runEditor`. Lookups, answers, playback and `/play` all run inside the keys
  case, and nothing hands an async result back yet, so a buffered result channel
  as a new case is the seam.
- Every screen write must follow `view.Draw("", nil)`
  (`TestNothingIsWrittenWhileAPromptIsShown`), so the job never writes; the loop
  prints its notice.
- `runHarvest` and `runReflect` return only exit codes and print prose, so the
  job cannot tell a missing model from other failures without typed cores.
- `llm.Resolve` succeeds on the default local proxy with nothing running; the
  first call is what reports `ErrUnavailable`.
- The dictionary is cgo DictionaryServices with no lock, and harvesting looks
  words up, so two goroutines would call it at once.
- The `llmtest` fake cannot hold a non-streamed reply, so the "never waits" test
  needs a blocking stub client through `d.newLLM`.
- No cross-process lock exists, and `SetItems` replaces a word's file, so
  duplicate authoring wastes calls rather than doubling material.
- A `/play` sitting reads items once (`todaysQuestions`), so items written during
  a sitting appear in the next.

Correction to the first Log entry: the raw editor already runs three
goroutines (the key reader and resize watcher in `rawterm.go`, the signal relay
in `interrupt.go`); `scanLines` belongs to the line loop. The conclusion stands:
nothing hands a job's result back to the loop yet.

Plan-quality round 2 cleared the plan with one advisory, PQ-5, the second
finding in `unbacked-claims-about-existing-code`: prose citations get no guard.
Fixed: Task 1.6's comment list, the change-window guard's wording, and a citation
pass over the plan's prose, now named in its guards section.

## Revisions

### 2026-09-12 — planning

- **The trigger is a check, not a counter of new words.** The loop asks for a
  check at session start and every 10 successful lookups; the job counts the
  store's unbanded words and harvests when there are at least 10. The loop never
  decides whether a word is new, so the store stays the only count, as the Spec
  asked.
- **No cross-process lock.** `SetItems` replaces a word's file, so two processes
  authoring one word leave one item, and the waste is bounded by each process's
  budget.
- **Dictionary calls are serialized process-wide**, a need the Spec did not name:
  DictionaryServices has no lock of its own, and the job looks words up while the
  loop does.
- **Harvest and reflect get typed cores**, because the job must tell a missing
  model from other failures, and both entry points return only exit codes.

### 2026-09-12 — plan-quality round 1

- **The count is the Spec's again.** Counting only unbanded words, while harvest
  bands the whole deck before authoring any, stalls a backlog: a small budget
  goes to banding, and once nothing is unbanded the count stops firing while most
  words still have no item. Pending now means no band or no item, and a job bands
  and authors one batch, the newest ten pending words.
- **Failed authoring is retried at most once per session**, through the
  session's `tried` set, so counting the no-item half cannot turn into a retry
  loop on words the model cannot write. A persistent backoff is decision 7 for
  the operator.
- **Two interleavings written down:** a word forgotten mid-job (ignored, and
  why), and which context the job derives from (the session's).

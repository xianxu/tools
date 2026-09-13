---
id: 000054
status: working
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours: 2.75
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

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module    design=0.2 impl=0.32
item: cross-cutting-refactor  design=0.12 impl=0.2
item: smaller-go-module       design=0.09 impl=0.4
item: tui-screen              design=0.2 impl=0.4
item: atlas-docs              design=0.04 impl=0.12
item: milestone-review        design=0.0 impl=0.56
design-buffer: 0.15
total: 2.75
```

Design takes the v2 range's middle × 0.2 throughout, because the plan settles
every decision; implementation is the v2 range × 0.4.

- greenfield-go-module: `background.go`, the state machine, `pendingWords`, the
  job and its runner. Design 1.0 × 0.2; impl at the top of 0.3–0.8, because it
  carries seven tests and a mutation pass: 0.8 × 0.4 = 0.32.
- cross-cutting-refactor: the harvest core takes a batch and returns a typed
  outcome, with every `--harvest` line unchanged. Design 0.6 × 0.2; impl 0.5 × 0.4.
- smaller-go-module, three of them: the dictionary lock (0.03 / 0.1), the reflect
  core's split (0.03 / 0.14), and `reflectDue` with `modelLookups` and their fuzz
  (0.03 / 0.16).
- tui-screen: the session wiring, the notices, and seven loop tests including a
  blocking stub client. Design 1.0 × 0.2; impl at the top of 0.3–1: 1.0 × 0.4.
- atlas-docs: two passes, M1 and M2, at 0.02 / 0.06 each.
- milestone-review: two boundaries (M1's milestone-close and the final close) at
  two rounds each, because #53's close needed three: 4 × 0.35 × 0.4 = 0.56.
- Design buffer +15% for a thorough plan: 0.65 × 1.15 + 2.0 = 2.75.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.*

## Plan

Detailed plan: `workshop/plans/000054-background-harvest-plan.md`. Two review
boundaries. Code branches after PR #38 (#53) merges, because M1's docs edit the
README rewrite it carries.

- [x] M1 — background harvest: one lock for every dictionary call; the harvest
      core returns a typed outcome (the CLI's output unchanged); the state
      machine, the job and its runner; the session wiring, notices and off
      switch; tests; docs.
- [ ] M2 — background reflect: the reflect core returns a typed outcome; when a
      model is due (none at 12 words, or the deck's lookups doubled); the job
      reflects before it harvests; tests; docs.

## Log

- 2026-09-13: closed M1 — M1 review round 1 (FIX-THEN-SHIP) fixed in 631e6e3. BR-1: every word a pass could not finish (a refused band, a model stop or store error on that word, authoring that kept nothing) joins failed and so the runner tried set; a budget cut is not a failure. Pins: TestABandRefusalIsRetriedOncePerSession, TestHarvestDeckCountsAStoppedWordAsUnfinished. BR-2: the job reads the store through quietStore (store.YAML.Quiet, through the deck gate), so no store warning reaches the terminal from its goroutine. Pin: TestTheJobWritesNothingToTheTerminal, bare and gated, with a control read of the same store that must warn. BR-3: plan table moves pendingWords to Integration points, adds quietStore, and appends a Revisions entry. Minors: notice now says "the model did not answer" (ErrUnavailable includes 429/5xx); assertNoJob counts model clients built and asserts after end() instead of a 700ms poll; hasModelSeam is the shared guard; one Deck read per job (pendingWords and runAuthoring take the deck); 130-col comment rewrapped. Mutations in a throwaway worktree at 631e6e3, each anchor matched once: 8/8 red (refused band dropped, stop dropped, job reads the loud store, gated quietStore loud, Quiet keeps warn, permission check dropped, off switch ignored, runner drops failed); unmutated control green. Round-1 mutations 12/13 (the budget row is unobservable by construction, recorded in plan and test). gofmt clean, go vet clean, go test ./... green after the commit (cmd/define 117.6s). Operator TUI smoke test (plan Task 1.7 Step 3) still pending.; review verdict: SHIP

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

`sdlc change-code`: plan-quality cleared in three rounds. Estimate-quality: INFO,
expecting nearer 4–5 h than 2.75, chiefly for the unbudgeted mutation runs (18
full-package runs at about 110 s each), a flaky suite, the loop tests, and a third
review round per boundary. The flaky full-package run has now been seen twice
(once during #53, once by that judge); a hunt runs in a separate worktree.

The flake hunt ran the whole package four times (`-count=4`, a separate worktree
at the plan commit) and nothing failed, so the failure is rare or depends on
load. Every full run now saves its output, so a recurrence names its test.

M1 tasks 1.1–1.6 landed (`8965908`, `41fe96d`, `4ae2328`, `27afcb3`, `c8c77b4`,
and the docs), each red before its code and green after, with the whole suite run
after every commit. Two small departures from the plan's text, recorded in the
plan itself: the "missing model" tests script a 500 (the harvest tests' existing
pattern) rather than a closed server, and the backlog tests use twelve words,
because the rig's word list holds fourteen. The README's learner-model line said
nothing calls a model during a lookup; M1 made that false, so the docs task fixed
it early instead of leaving it for M2.

M1 verification, in a throwaway worktree at `5d4eb83`. 12 of 13 mutations turned
their guard red, and the unmutated control was green:

| mutation | went red |
|---|---|
| the lock is skipped | TestLockedDictionarySerializesAcrossInstances |
| production builds a bare dictionary | TestProductionDictionariesAreLocked |
| a lookup while running starts a job | TestStepBackgroundTransitions |
| pending drops the no-item half | TestPendingWordsFollowsTheSpec |
| the job ignores the threshold | TestRunBackgroundJobHarvestsOnlyPastTheThreshold |
| the job bands the whole backlog | TestABacklogDrainsOnBothHalves |
| the runner drops `failed` | TestAWordThatFailsIsRetriedOncePerSession |
| the send ignores cancellation | TestTheRunnerNeverBlocksAfterStop |
| the permission check is dropped | TestNoJobWhereTheDeckWasNotAgreedTo |
| the off switch is ignored | TestTheOffSwitchStopsIt |
| `noModel` is never set | TestNoModelIsOneNoticeThenQuiet |
| the notice skips clearing the frame | TestABackgroundNoticeIsWrittenBetweenPrompts |

Not observable: the job's budget replaced by the CLI's 200. Ten words cost at most
60 calls (band, author, entail, three vetoes each), which is the budget itself, so
the cap cannot bind; the batch size is the real bound today, and the budget stays
as a backstop for an item that costs more. Recorded in the plan's table and the
test's comment rather than claimed as a pin. gofmt and `go vet ./...` clean; every
other package green; the whole `cmd/define` suite green after every commit. The
operator's TUI smoke test is pending.

**M1 boundary review, round 1: FIX-THEN-SHIP, three Important.** All three are
fixed in one commit, with the cheap Minors:

- A refused band never reached `tried`, so a word the model mis-bands would take
  a batch slot at every check. `harvestOutcome.failed` now holds every word a pass
  ran for and could not finish: a refused band, authoring that kept nothing, or a
  model stop or store error on that word. A budget cut is not a failure. Pinned
  by TestABandRefusalIsRetriedOncePerSession and
  TestHarvestDeckCountsAStoppedWordAsUnfinished; the store-error joins have no
  pin, because no rig here fails one word's read.
- A job's store reads could print a warning into the frame from its goroutine,
  because the session's stores warn to the process stderr. The job reads through
  `quietStore` (`store.YAML.Quiet`, through the deck gate). Pinned by
  TestTheJobWritesNothingToTheTerminal, bare and gated, with a control read that
  must warn. The loop's own reads still warn to stderr as they did before #54.
- The plan listed `pendingWords` as pure; it moved to the integration table.
- Minors: the notice says "the model did not answer", since a 429 or a 5xx is
  `ErrUnavailable` too; `assertNoJob` counts the clients built after `end()`
  instead of polling for 700 ms; `hasModelSeam` is the one guard; a job reads the
  deck once; a 130-column comment in `harvest.go` is rewrapped.


**M2 built.** The reflect core returns what it did (`reflectDeck`); `reflectDue`
and `modelLookups` decide when the session writes the learner model; the job
writes it before it harvests. With them, the M1 review's round-2 Minors, fixed as
rules: `markUnfinished` and `stopMeans` decide in one place what a pass gives up
on and what a stop means, a deck whose files fail is said once and stops
background work, a reflect that writes nothing is not retried this session, a
miss not counting toward a check is pinned, and the permission comment names the
job's reads. The review's budget-cut finding does not hold against the code:
`errBudget` breaks before the return that would mark the word.

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

### 2026-09-12 — M1 boundary review, round 1

- **Any word a job could not finish is tried at most once a session**, not only
  one whose authoring kept nothing: a refused band, a model stop on that word, or
  a store error on it counts too, so the retry bound covers the banding half.
- **The job writes nothing to the terminal**, store warnings included: it reads
  through a quiet view of the same store.
- **The notice reads "the model did not answer"**: a rate limit or a 5xx also
  turns the session's background work off, and "no model answered" overstated it.

### 2026-09-13 — M2, and the M1 review's second round

- **A deck whose files cannot be read or written is said once**, and the
  session's background work stops, as it does for a model that does not answer:
  otherwise it would fail the same way at every check and never say why.
- **A learner model the job could not write is not asked for again this
  session**, by the same rule as a word it could not finish.

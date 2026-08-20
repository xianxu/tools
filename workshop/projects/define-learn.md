---
type: project
name: "define-learn"
goal: "Turn define from a lookup tool into a retention tool: words looked up enter a per-user deck, define --play reviews what is due with several question forms, define --stats shows whether it is working."
done_when: "A real day of review runs end to end: words captured by ordinary define use, scheduled by spaced repetition, reviewed in at least two forms, persisted to the brain, and visible in --stats."
status: ideation
created: 2026-08-20
updated: 2026-08-20
---

# define-learn

## PRD

## Estimate

## Breakdown

- [ ]

## Log

## Sequencing

**M1 is a complete trainer with no network and no API key** — deliberate: it
delivers daily value alone and de-risks everything after it.

| phase | issues | delivers |
|---|---|---|
| **M0 — standalone** | `#2` REPL | independently shippable; useful before any deck exists |
| **M1 — offline trainer** | `#3` store → `#4` capture, `#5` schedule → `#6` play → `#7` meaning form, `#8` stats | a working daily review loop |
| **M2 — grounded questions** | `#9` news → `#10` harvest, `#11` LLM → `#12` cloze, `#13` sentence | cloze from current news + AI grading |

## Tasks

- [ ] `#2` define REPL — bare invocation reads, defines, speaks; bare return replays
- [ ] `#3` vocabulary store — Store seam, YAML in a brain, clock injected
- [ ] `#4` capture on lookup — successful lookups build the deck
- [ ] `#5` scheduling engine — Leitner, pure
- [ ] `#6` `--play` loop + form 2.1
- [ ] `#7` form 2.3 — meaning multiple choice, deck distractors, no LLM
- [ ] `#8` `--stats` — all derived from the event log
- [ ] `#9` news seam — Google News RSS (not the SERP)
- [ ] `#10` news harvester — async, level-aware candidate pool
- [ ] `#11` LLM seam — narrow tasks, stateful fake, offline degradation
- [ ] `#12` form 2.2 — cloze from news, distractors **selected not invented**
- [ ] `#13` form 2.4 — free sentence, graded

## Design decisions taken up front

- **Storage shape is chosen for git.** One file per word + an append-only event
  log per day; a brain syncs across machines, so merge conflicts are the failure
  mode. `nous push` supplies sync, so no sync code is written.
- **Distractors are selected, never invented** (operator, 2026-08-20). The pool is
  news-harvested words at the learner's level plus the learner's own deck,
  filtered for substantial semantic difference. The model only *vetoes* a
  candidate that would also fit — a yes/no check on a concrete pair, not open
  generation. This removes the "the LLM's wrong answer is also right" failure mode.
- **Google News RSS, not the SERP.** Measured: 41–100 items per word. The SERP
  was measured too — a 91 KB JS shell with zero usable content.
- **Clock injected everywhere.** Spaced repetition is date-driven; "due today" is
  untestable against a wall clock.
- **Every LLM/network feature degrades.** No key, no network → `--play` falls
  back to the local forms rather than failing.

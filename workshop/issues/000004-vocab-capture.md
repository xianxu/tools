---
id: 000004
status: working
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-08-21
estimate_hours: 1.22
started: 2026-08-21T10:02:22-07:00
---

# capture looked-up words into the deck

## Problem

A vocabulary deck nobody has to curate is the only one that gets used. Every
`define` lookup is already a signal of interest — it should build the deck.

## Spec

`define <word>` records the lookup into the deck.

- **Capture only on a successful lookup.** Typos exit 1 and never enter, so the
  dictionary is its own spam filter and no validation layer is needed.
- Record lookup count and timestamps: a word looked up three times is a stronger
  signal than one looked up once, and `--play` can order by that.
- `define --forget <word>` removes the word from the deck. It does **not** delete
  events: the deck is a working set, the log is history, and rewriting the past
  would corrupt every statistic `#8` derives.
- `DEFINE_NO_CAPTURE=1` means **write nothing in this directory** — not "deck
  only". Persisted history is the event log (`#3`), so the flag also drops
  history to session-only. That cost is documented beside the flag rather than
  discovered.
- Capture must never break a lookup: a store failure warns on stderr and leaves
  the exit code alone, exactly as audio failure does today.
- Applies to REPL lookups (#2) too.

## Done when

- [ ] A successful lookup appears in the deck; a failed one does not.
- [ ] Repeat lookups increment the count rather than duplicating the word.
- [ ] `--forget` removes the word and leaves `events/` untouched, asserted by
      comparing the directory before and after.
- [ ] `--forget` cannot delete outside `words/` — asserted, not inherited from
      `Slug`.
- [ ] `DEFINE_NO_CAPTURE=1` writes nothing at all, and history falls back to
      session-only rather than half-persisting.
- [ ] A failing store degrades to a warning, never a failed lookup.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.15 impl=0.04
item: smaller-go-module        design=0.15 impl=0.2
item: smaller-go-module        design=0.1  impl=0.16
item: milestone-review         design=0.0  impl=0.12
item: milestone-review         design=0.0  impl=0.12
item: atlas-docs               design=0.05 impl=0.06
design-buffer: 0.15
total: 1.22
```

Derivation notes:

- **Two `smaller-go-module`s, no greenfield.** Nothing here is new: `#3` already
  writes on the interactive path, so this extracts that decision behind a seam
  and widens it to two more call sites. The first covers the extraction plus
  widening, the second `--forget` and the opt-out.
- Design discounted ×0.2 on both — the plan fixes the seam, the policy inputs and
  the three call sites; what is left is typing.
- **Two `milestone-review`s.** `#3` needed four rounds on one function, and this
  diff touches the same store. Budgeting one would repeat the mistake the ledger
  has now recorded three times.
- `familiarity: 1.0` — same package, same store, same test posture.
- Library-availability check: nothing external is involved; no halving applies.

Σdesign 0.45 × 1.15 = 0.5175; Σimpl 0.70; total **1.22**.

## Plan

See `workshop/plans/000004-vocab-capture-plan.md`.

- [ ] Extract the capture policy behind a `Capturer` seam; `storeHistory` uses it.
- [ ] Widen to the one-shot and line paths.
- [ ] `--forget` (word only, never events) and `DEFINE_NO_CAPTURE=1`.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

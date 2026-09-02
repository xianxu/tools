---
type: pensive
topic: the boundary review is where the hours go, and the estimate model does not price it
created: 2026-09-01
---

# The boundary review is the estimate

Two issues closed on consecutive days against the same machinery, and both
overran by half — entirely at the boundary rather than in the building:

| issue | est | actual | ratio | close rounds | findings |
|---|---|---|---|---|---|
| `#41` `--play` paints frames | 3.31 | 4.77 | 1.44× | 5 | 16 |
| `#38` clickable words | 2.83 | 4.24 | 1.50× | 4 | 18 |

Both priced the `milestone-review` pair at **0.30 / 0.32 = 0.62h**. Both spent
roughly **half the issue** there. v3.1's own scaled ceiling for those rows is
0.20 each, so the model is not merely mis-tuned here — the primitive table is
describing a different activity than the one this repo actually runs.

## What the rounds bought, which is the part that stops this being waste

They were not ceremony. Across the two closes the reviews found **three
Criticals**, and every one was a real defect that would have shipped:

- `#41` BR-23 — a guard I added in round 3 skipped every line carrying an escape,
  and since `--play` only runs with colour on, that exempted the entire class the
  guard existed to serve.
- `#38` BR-3 — the click map was moved by the loop's width while the screen
  wrapped at its own, so a resize put a headword region on a blank line.
- `#38` BR-11 — the plan's own table named two entities the tree did not declare.

And two of the three were introduced **by a previous round's fix**. That is the
signature worth naming: the cost is not "reviewing takes N rounds", it is
**"fixing at the wrong altitude generates the next finding."** `#38` BR-16 is the
cleanest instance — a `red when` cell that survived its own mutation twice, first
because `toInput` already delivered the property and then because a one-word
fixture could not express the failure.

## The estimating consequence

An issue's cost here is roughly `build + (rounds × remediation)`, and `rounds` is
driven by **how many surfaces a change touches**, not by how much code it is.
`#38` was ~300 production lines and took four rounds, because it reversed an
invariant, widened a seam two loops share, and edited an append-only buffer's
contract.

So the estimating question is not "how big is this" but **"how many contracts
does this touch, and how many of them are documented somewhere I have not read
yet?"** `#38`'s BR-1 was exactly that: `WriteRegions` and `RenderOpts` document
three obligations between them, the plan carried none, and two were live defects.

`#40` prices its boundary at 0.40/0.45 on this evidence, and predicts 5–6h
against a 4.46 estimate. **If that lands, the ledger has three consecutive rows
saying the same thing and the primitive table should move** — which is `#127`'s
job, and this note is the input.

## The counter-consideration I do not want to lose

The obvious reading — "raise the review primitives" — may be treating a symptom.
The rounds are finding defects that a *better plan* would have prevented: two of
the three Criticals were mine, introduced while fixing something else. It is
possible the right adjustment is not a bigger boundary budget but a **plan-time
enumeration step** — read the doc comments of every mechanism the plan adopts,
and turn each documented obligation into a row — which `#38`'s lessons entry now
prescribes. If that works, the boundary cost should fall rather than the estimate
rising to meet it.

Both readings fit the current three rows. `#40` is the first issue planned with
that enumeration done up front, so it is the discriminating case.

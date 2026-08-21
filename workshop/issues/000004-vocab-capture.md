---
id: 000004
status: working
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-08-21
estimate_hours: 1.31
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

- [x] A successful lookup appears in the deck; a failed one does not.
- [x] Repeat lookups increment the count rather than duplicating the word.
- [x] `--forget` removes the word and leaves `events/` untouched, asserted by
      counting events before and after in the shared conformance suite. (The
      original wording said "comparing the directory"; the assertion counts
      events, which covers the same risk — corrected rather than left claiming
      something else.)
- [x] `--forget` cannot delete outside `words/`. The guard is now a pure
      `wordFileName(slug)` tested directly with hostile names — behind `Slug` it
      was unreachable in principle, so the earlier tick claimed an assertion that
      did not exist. `Slug` remains the first line (fuzzed); this is the net
      under a future regression in it.
- [x] `DEFINE_NO_CAPTURE=1` writes nothing at all, and history falls back to
      session-only rather than half-persisting.
- [x] A failing store degrades to a warning, never a failed lookup.

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
total: 1.31
```

Derivation notes:

- **Two `smaller-go-module`s, no greenfield.** Nothing here is new: `#3` already
  writes on the interactive path, so this extracts that decision behind a seam
  and widens it to two more call sites. The first covers the extraction plus
  widening, the second `--forget` and the opt-out.
- Design discounted **×0.5, not ×0.2** — correcting the first draft's note, which
  claimed a figure it had not applied. Half rather than a fifth because the plan
  needed four gate rounds to settle: the capture site was named wrongly at first
  (`defineOnce` is not on the raw path), and the opt-out's collision with `#3`'s
  event-backed history was found at the gate, not in the draft.
- **Two `milestone-review`s.** `#3` needed four rounds on one function, and this
  diff touches the same store. Budgeting one would repeat the mistake the ledger
  has now recorded three times.
- `familiarity: 1.0` — same package, same store, same test posture.
- Library-availability check: nothing external is involved; no halving applies.

Σdesign 0.50 × 1.15 = 0.5750; Σimpl 0.74; total **1.31**.

## Plan

See `workshop/plans/000004-vocab-capture-plan.md`.

- [x] Extract the capture policy behind a `Capturer` seam; `storeHistory` uses it.
- [x] Widen to the one-shot and line paths.
- [x] `--forget` (word only, never events) and `DEFINE_NO_CAPTURE=1`.

## Log

### 2026-08-20

Created as part of the `define-learn` project.

### 2026-08-21 — implementation notes

Capture landed at `lookupAndRender`, the one function every entry path shares —
**not** `defineOnce`, which the plan named first and which the raw editor
bypasses entirely. Verified against the call graph rather than argued.

**Three review rounds, and the third escalated by family rather than by
instance** — the mechanism filed as `ariadne#195`, met in the wild:

- `unpinned-invariant` (4th): three fixes had shipped with no test that fails
  without them. The rule now applied to every fix in the round — *delete the line
  you added and run the suite; if it stays green you have documentation, not a
  pin.* All three now go RED on removal. The traversal guard forced the honest
  choice the review named: behind `Slug` it was untestable in principle, so it
  became a pure `wordFileName(slug)` exercisable with hostile input.
- `prose-contradicts-code` (4th): `storeHistory`'s type comment still called
  itself the writer, two rounds after it stopped being one. Fixed by deleting the
  restatement rather than correcting it — one normative home per design fact.
- `undocumented-work-log` (2nd): a Done-when box was ticked for an assertion that
  did not exist, and another described a check different from the one written.
  Both corrected above; this Log is the rest of it.

**A false reading I nearly reported:** one mutation check printed GREEN because
the substitution silently failed to apply, not because the test was blind. The
tell was an assertion error in the same output. A mutation that does not apply is
not a passing result — verify the mutation landed before believing what it says.

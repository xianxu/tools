---
id: 000004
status: codecomplete
deps: ["tools#3"]
github_issue:
created: 2026-08-20
updated: 2026-08-21
estimate_hours: 1.31
started: 2026-08-21T10:02:22-07:00
actual_hours: 3.85
---

# capture looked-up words into the deck

## Problem

A vocabulary deck nobody has to curate is the only one that gets used. Every
`define` lookup is already a signal of interest — it should build the deck.

## Spec

`define <word>` records the lookup into the deck.

- **Only a successful lookup enters the deck.** A failed one is still recorded
  as an event, because up-arrow recall must reach a word you typed and got
  wrong. So the dictionary is the deck's spam filter — typos are history, never
  vocabulary — and no validation layer is needed. `decideCapture` is the single
  place that draws this line; see `capture.go`.
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
- [x] Repeat lookups increment the count rather than duplicating the word —
      pinned **through the capture path** (`TestRepeatLookupsIncrementThroughCapture`),
      not only at the `Store` level. Capture supplies `Lookups: 1` on every call,
      so accumulation depends on both halves agreeing; breaking the merge
      reddens it.
- [x] `--forget` removes the word and leaves `events/` untouched, asserted by
      counting events before and after in the shared conformance suite. (The
      original wording said "comparing the directory"; the assertion counts
      events, which covers the same risk — corrected rather than left claiming
      something else.)
- [x] `--forget` cannot delete outside `words/` — stated precisely, because two
      earlier versions of this box were false:
      - **`Slug` is the effective guarantee**, and it is fuzzed (6.9M execs) for
        "always exactly one safe path element".
      - **`wordFileName(slug)` is a second net** under a future `Slug`
        regression. It is a pure function tested directly with hostile names, and
        removing it turns that test RED.
      - **`Upsert` and `Forget` both derive their filename from it** — verified by
        grep, not by test, and honestly so: because `Slug` sanitises first, no
        test at the `Store` API can distinguish routing through the guard from
        bypassing it. Measured, not assumed: bypassing it on the `--forget` path
        leaves the suite GREEN. The value is ARCH-DRY (one definition of a safe
        word filename), not a behavioural pin.
- [x] `DEFINE_NO_CAPTURE=1` writes nothing at all, and history falls back to
      session-only rather than half-persisting.
- [x] A failing store degrades to a warning, never a failed lookup. Both halves
      pinned: warn-once at the capturer, and the half a user actually feels —
      the definition still prints and the exit code stays 0
      (`TestFailingStoreStillDefinesAndExitsZero`).

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
- 2026-08-21: closed — The residual was entirely artifact-side and is now closed. Round 5 measured code 10/10 and plan 0/7; all seven plan sites are corrected (one call site not three; storeHistory does nothing rather than delegates; lookupAndRender not defineOnce; openStore not openHistory; the three-argument Capturer; the three deps fields the plan never mentioned) and the plan now carries the AGENTS.md §1 Revisions section that three consecutive boundaries recommended. Also swept the two stale comments this round CREATED in files it touched: Forget doc comment asserted a security property the same commit retracted and named a filepath.Base call deleted in the same hunk, and inserting type storeDeps orphaned openStore doc comment onto the struct -- both verified re-attached to the code they describe. Code side unchanged and re-verified: full suite and -race green across both packages, go vet green, GOOS=linux CGO_ENABLED=0 green.; review verdict: FIX-THEN-SHIP
- 2026-08-21: closed — Round 4 addressed instance-by-instance, which round 3 was not. Of the 10 instances the three families enumerated, 3 were closed in round 3 (all titles) and the remaining 7 are closed now: BR-6 Forget was still on an inline copy while wordFileName had exactly one caller (Upsert) -- my own previous fix introduced the duplication it was meant to remove, and the copy had already drifted by not rejecting a leading dot; the atlas entry-modes table claimed run dispatches into a single shared defineOnce, which the raw editor bypasses; README exit codes omitted that 1 now also means --forget found nothing; deps.forgetter(), the newStore triple with three nil-merges, and newStoreHistorys dead Clock param are deleted. The BR-6 Done-when is now precise and measured rather than claimed: Slug is the effective guarantee (fuzzed), wordFileName is a second net tested directly with hostile names (RED on removal), and both Upsert and Forget derive from it -- verified by grep, because bypassing it on the --forget path leaves the suite GREEN, so the value is ARCH-DRY rather than a behavioural pin. Two false mutation readings were caught and recorded: one that failed to apply printed GREEN, one that failed to compile printed RED. go test and -race green across both packages, go vet green, GOOS=linux CGO_ENABLED=0 green.; review verdict: FIX-THEN-SHIP

The review preamble that recurs in every round of `close-review.md` (the harness
trust-dialog notice, now at six occurrences) is **ariadne#201**, filed 2026-08-21.
It is a defect in how the review artifact captures the reviewer's stderr, so it
cannot converge in this repo; the issue also raises whether the protocol should
let a finding be dispositioned "external, tracked at <ref>" once instead of
re-costing a slot each boundary.

**This branch's history was rewritten on 2026-08-21.** A 9.6 MB `cmd/define/define`
was committed in `42cc96d`; round 7 deleted the file in a *follow-up* commit, which
left the blob reachable and every clone paying for it — measured at 5.9 MB against
`main`'s 604 KB. `git filter-branch --index-filter` over `main..HEAD` rewrote the
adding commit. Re-measured after: **712 KB, identical to `main`**, blob absent from
a fresh clone. Anyone holding an old copy of this branch must re-fetch rather than
merge. The trees are byte-identical, so only the object history changed.

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

### 2026-08-21 — round 4: the family rule, applied properly

Round 3 escalated three families; I closed **3 of the 10 instances they
enumerated, and all three were the ones in the titles** — the exact substitution
(fix the named thing, leave the class) the escalation exists to prevent. Round 4
caught that as its own finding, which is the mechanism working.

Instance by instance this round:

| family | instance | disposition |
|---|---|---|
| `unpinned-invariant` | raw capture row | fixed round 3 |
| | `openStore` deck path | fixed round 3 |
| | BR-6 `Forget` guard | **fixed now** — `Forget` was still using an inline copy; `wordFileName` had one caller and it was `Upsert` |
| `prose-contradicts-code` | `history_store.go` comment | fixed round 3 |
| | atlas entry-modes table | **fixed now** — it claimed `run` dispatches into "a single shared `defineOnce`", which the raw path bypasses |
| | `--help` | fixed round 3 |
| | README exit codes | **fixed now** — `1` now also means `--forget` found nothing; this issue invalidated the list and did not update it |
| `needless-indirection` | `deps.forgetter()` | **fixed now** — deleted; `d.deck == nil` says it directly |
| | the `newStore` triple | **fixed now** — one `storeDeps` value and one `withStore`, replacing three returns and three nil-merges |
| | dead `Clock` param | **fixed now** — `newStoreHistory` stopped needing it when it stopped writing |

**My BR-6 fix had introduced the duplication it was meant to remove.** I added
`wordFileName` and wired only `Upsert` to it, leaving `Forget` — the entire
subject of the finding — on an inline copy that had already drifted (it did not
reject a leading dot). One line of wiring closed the duplication, the dead
branches and the false Done-when together.

**Two false readings caught while verifying**, both worth recording: a mutation
that failed to compile reported RED (a build break is not a test failure), and
before that one that failed to apply reported GREEN. Neither is a result. The
working form: confirm the mutation compiled *and* landed, then read the suite.

### 2026-08-21 — round 6: audited every box rather than the named one

The 5th `unpinned-invariant` named Done-when #2. Rather than pin that one, I
audited all six boxes for the symbol each claims and found **two** unpinned: the
named repeat-lookup claim, and "never a failed lookup", whose warn-once half was
pinned at the capturer while the half a user feels — the lookup still succeeding
— was pinned nowhere. Both now redden under mutations verified to have applied
*and* compiled.

That is the family rule finally applied the way five rounds of findings asked
for: enumerate the class, close the class, and say which instances were found by
audit rather than by being named.

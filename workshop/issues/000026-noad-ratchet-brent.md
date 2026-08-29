---
id: 000026
status: working
deps: [tools#27]
github_issue:
created: 2026-08-27
updated: 2026-08-28
estimate_hours: 0.65
started: 2026-08-28T17:57:39-07:00
---

# NOAD ratchet regressed 27 to 32: a British-English pronunciation block renders raw

## Problem

`TestRenderLosesNothingOverLiveEntries` carries a ratchet — a measured count of
live entries that still render unconverted NOAD notation, held at
`knownRawNotationEntries = 27`. Run against the live dictionary it now reports
**32**, and fails:

```
live_property_test.go:127: 32/73502 live entries rendered unconverted NOAD
    notation, up from the known 27 — a regression
```

Two of the three named entries are a shape the ratchet's own comment does not
cover:

```
Brent: unconverted NOAD notation survived, near
  "brent\n\n    | AmE brɛnt, BrE brɛnt | noun (British English) "
brent: (identical)
charge: "…“he charged me”\n    2. euros for the postcard | the restaurant
  charged $15 for dinner | [no obje…"
```

`charge` is the documented cause — *"a prose numeral that happens to continue a
sense sequence is accepted as a sense number"*. `Brent`/`brent` are not: they are
a **dual-pronunciation block**, `| AmE … , BrE … |`, which the renderer leaves in
raw pipe notation.

## Spec

**Triaged 2026-08-28, and the triage answered both questions by removing them.**

`#23 M2` made English select `com.apple.dictionary.NOAD` plus
`com.apple.dictionary.AppleDictionary` **by identifier**, where before it passed
NULL and searched every ACTIVE dictionary on the host. `Brent` was a
British-dictionary entry reachable only through that search. Verified: `define
brent` now reports **no dictionary entry** at all.

So the dual-locale block is not fixed — **its input is gone**. Both Spec
questions (converter vs splitter; render both pronunciations or the deck's) are
moot, and so is `deps: [tools#27]`: they existed only to decide how to render a
shape that no longer arrives.

**It is DORMANT, not dead, and that is the thing to write down.** `curated["en"]`
is `{NOAD, AppleDictionary}`. Adding Oxford Dictionary of English — a plausible
future entry, it is strictly monolingual `en` and passes every metadata filter —
brings `| AmE … , BrE … |` straight back, along with this defect. The resurrection
condition belongs in the record so the next person adding a curated dictionary
meets it.

### What the re-measurement actually found

Row 3 is the real work, and it was larger than "re-count". The ratchet samples
only the first three survivors, so 23 of them had never been looked at. Enumerated
in full against the live dictionary, 2026-08-28 — **26 entries, four causes, not
one**:

| cause | n | entries |
|---|---|---|
| a prose numeral continuing a sense sequence — *the documented cause* | 7 | `charge`, `chargee`, `charging`, `depth`, `just`, `justness`, `shortness` |
| a pronunciation glued to the headword — `(aˈhəndrədzˈhəndrəd/)` | 7 | `hundred`, `hundredfold`, `million`, `millionfold`, `millions`, `thousand`, `thousandfold` |
| a phrase-block pronunciation run into prose — `lick someoneˌoud əv ˈSHāp/` | 8 | `shape`, `shapable`, `Shape`, `shaper`, `shaping`, `short`, `shorter`, `shortish` |
| **a literal `|` that IS the content** — *"• the symbol \|."* | 4 | `pipe`, `piped`, `pipeful`, `pipeless` |

Only 7 of 26 are the cause the ratchet's own comment claims covers all of them,
and `atlas/define.md` repeats that attribution.

**The `pipe` family is a false positive, and the oracle stays broad anyway.** The
check is `IndexByte(out, '|') >= 0`; those four entries define the pipe
character, so their content legitimately contains one. Narrowing the oracle to
exclude them is exactly the move `workshop/lessons.md` records as having already
failed here — the narrow oracle read 0% while 2.2% of entries showed raw pipes,
and that false 0% was published in the atlas. So the count stays 26 and the four
are ATTRIBUTED rather than excluded, which is what "attributable rather than
absorbed" asks for.

## Done when

- [ ] The `| AmE … , BrE … |` shape is DISPOSITIONED rather than ticked: recorded
      as unreachable on the curated path, with the condition that resurrects it
      (a British dictionary entering `curated["en"]`) written where the next
      person adding one will meet it.
- [ ] The ratchet is re-measured and `knownRawNotationEntries` lowered to the new
      true count — not raised to 32 to make the test pass.
- [ ] The remaining raw-notation entries are re-enumerated BY CAUSE, so the next
      drift is attributable rather than absorbed — and the ratchet's own comment
      stops claiming they are all one cause.
- [ ] `atlas/define.md`'s Limits entry stops attributing the whole count to the
      prose numeral, and stops naming entries the sweep no longer reaches.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.15 impl=0.05
item: smaller-go-module        design=0.05 impl=0.10
item: atlas-docs               design=0.04 impl=0.06
item: milestone-review         design=0.00 impl=0.16
design-buffer: 0.15
total: 0.65
```

Derivation notes.

- **`issue-spec` at 0.15, well under the undiscounted band**, and deliberately:
  the triage was already done while answering "what's left in #26" — the live
  enumeration, the four causes and the `brent` check are spent, not forecast.
  What remains under this row is writing them down.
- **One `smaller-go-module`, not a one-liner.** The constant is one character;
  the ratchet's comment is the deliverable, because "all of one known cause" is
  the false claim this issue exists to retire.
- **`atlas-docs` is not optional here** — the atlas repeats the same false
  attribution and names three entries the narrowed sweep no longer reaches. A
  Done-when row covers it.
- **One `milestone-review`** for the close boundary. At the band ceiling would be
  0.20; 0.16 because the diff is a constant, two comments and an issue — the
  smallest surface this repo has reviewed.
- Library-availability check: nothing external; no halving applies.

Σdesign 0.24 × 1.15 = 0.276; Σimpl 0.37; total **0.65**.

## Plan

- [ ] Lower `knownRawNotationEntries` to 26 and rewrite the ratchet comment with
      the four causes, so a future movement names which group moved.
- [ ] Correct `atlas/define.md`'s Limits entry: the count, the attribution, and
      the entry list (`ratio`, `glop`, `logarithmic` are no longer reachable).
- [ ] Record the dormant-not-dead resurrection condition beside the curated list.
- [ ] Re-run the live ratchet unsandboxed to confirm green in both directions.

## Log

### 2026-08-27

- **Found by running the conformance suite UNSANDBOXED**, which is the point of
  the suite and had not happened in the sessions around #24. Sandboxed, this test
  reaches 0 live entries and skips (or, before #24, hard-failed as "sandboxed?"),
  so the ratchet had effectively never run.
- **Confirmed PRE-EXISTING, not introduced by #24.** Ran the same test in a
  worktree at `87e6307b` (the #24 branch point): identical `32/73502`, same three
  named entries. #24 touched no rendering code.
- The 5-entry gap (27 → 32) is only partly enumerated: three entries are named in
  the failure output because the test reports "a handful, not thousands". A full
  list needs the report cap raised for one run.
- Reproduce:
  ```sh
  go test -tags conformance ./cmd/define/ -run TestRenderLosesNothingOverLiveEntries -v
  ```
  Must run unsandboxed; takes ~51s over 73502 live entries.

### 2026-08-28

- **Blocked on [tools#27]** (pronunciation locale as a parameter), split out of
  #18 the same day for this reason among others. Deliberately NOT started: the
  cheap fix here is to widen the notation converter to swallow
  `| AmE …, BrE … |`, and that would bake in "show both" as a decision by
  accident, at the exact moment the codebase is about to gain a real locale.
- The ratchet stays red until then, which is correct — it is reporting a true
  regression, and the number must not be raised to 32 to silence it.

### 2026-08-28 — re-measured after `#23 M2`; the cause is gone and the pin is now too high

`#23 M2` made English select `com.apple.dictionary.NOAD` plus
`com.apple.dictionary.AppleDictionary` **by identifier**, instead of passing NULL
to `DCSCopyTextDefinition` and searching every ACTIVE dictionary on the host.

That removes this bug's cause rather than fixing its symptom. `Brent`'s
`| AmE brɛnt, BrE brɛnt |` is a British-dictionary shape — it was reachable only
because the NULL search consulted whatever the host had enabled. Re-measured
against the live dictionary, 2026-08-28:

```
checked 70886 live entries: 0 lost content, 26 kept raw notation;
0 non-Latin (other active dictionaries), 165090 absent
```

- **26, not 32** — and below the pinned 27, so the ratchet now fails in the GOOD
  direction: *"only 26 … below the pinned 27 — lower knownRawNotationEntries to
  lock the improvement in"*.
- **`0 non-Latin`**, where the previous run had a class of them. The three sampled
  survivors are now all `charge`/`chargee`/`charging` — the prose-numeral cause
  the ratchet's own comment already documents, not a new shape.
- **The sweep narrowed**, and that qualification belongs here rather than being
  discovered later: 70,886 entries checked against 73,502, with 165,090 absent,
  because two dictionaries are asked rather than all of them. Some of the drop is
  "we stopped looking at entries we never serve." It is still better
  proportionally — 0.037% against 0.044% — so the improvement is real, but it is
  not purely a parsing gain.

**What is left of this issue:** lower `knownRawNotationEntries` from 27 to 26 and
close it. The uncovered-shape half of the Problem no longer reproduces.

**Its `deps: [tools#27]` is now moot** — the dependency was on `#27` owning
locale as a parameter, and the British-English entry that motivated it is
unreachable on the curated path.


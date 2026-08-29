---
id: 000026
status: working
deps: []
github_issue:
created: 2026-08-27
updated: 2026-08-28
estimate_hours: 1.22
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

**Positive control, because a negative probe alone proves nothing** (PQ-6). In
the same session and the same build: `define charge` returns an entry with no
`no known en dictionary is installed` warning, so the curated path IS being
exercised and `brent`'s absence is a fact about the dictionaries rather than
about a broken lookup context.

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

`ratio`, `glop` and `logarithmic` — which `atlas/define.md` names as the same
shape — are **no longer in the list, and they are NOT unreachable**. Verified:
all three still return entries. They render CLEAN now. A first draft of this Plan
said "unreachable", which is a different claim and a wrong one (PQ-1).

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
      as unreachable on the curated path — with its positive control — and with
      the condition that resurrects it (a British dictionary entering
      `curated["en"]`) written where the next person adding one will meet it.
- [ ] The ratchet is re-measured and `knownRawNotationEntries` lowered to the new
      true count — not raised to make the test pass.
- [ ] **The run itself names which group moved.** Each survivor is classified by
      cause and the per-cause counts are reported, so a future 27 does not print
      one number and three samples and send the next person back to raising the
      diagnostic cap by hand — a cost this issue already paid once.
- [ ] The classifier is testable WITHOUT the live dictionary: one real captured
      exemplar per cause, so the taxonomy is not conformance-only (ARCH-MOCK).
- [ ] `atlas/define.md`'s count DERIVES from `knownRawNotationEntries` rather
      than restating it — the number has drifted three times, which is the
      repo's own trigger for making a doc a consumer.
- [ ] Every measured claim the re-measurement invalidated is corrected, from a
      grep that ran rather than a hand list: the sweep width, the non-Latin
      count, the fixture count, the attribution, and the live test's own comment
      claiming there is no API to select a dictionary.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec               design=0.15 impl=0.05
item: smaller-go-module        design=0.05 impl=0.16
item: smaller-go-module        design=0.05 impl=0.12
item: smaller-go-module        design=0.05 impl=0.10
item: cross-cutting-refactor   design=0.05 impl=0.12
item: atlas-docs               design=0.04 impl=0.06
item: milestone-review         design=0.00 impl=0.16
design-buffer: 0.15
total: 1.22
```

Derivation notes.

- **This was 0.65 and is 1.22, and the gate is why.** The first block priced
  "lower a constant and correct a comment". PQ-4 and PQ-5 pointed out that
  Done-when row 3 asks the next drift to be ATTRIBUTABLE, and a comment cannot
  enforce that — so the deliverable is a classifier with per-cause reporting plus
  offline exemplars, not a number. Three `smaller-go-module`s: the classifier and
  its reporting, the exemplar corpus and its unit test, and the atlas doc-sync
  test.
- **`cross-cutting-refactor` for the doc sweep**, six sites across two files,
  enumerated by a grep that ran (PQ-2) rather than by hand. Small, but it is the
  shape `#24`'s nine call sites were priced as, not a one-liner.
- **`issue-spec` stays 0.15** — the triage, the live enumeration and the positive
  control were spent before this block existed.
- **One `milestone-review`** at 0.16 rather than the 0.20 ceiling: the diff is a
  test, a fixture directory and doc corrections.
- Library-availability check: nothing external; no halving applies.

Σdesign 0.39 × 1.15 = 0.4485; Σimpl 0.77; total **1.22**.

## Plan

- [ ] **M1 — the ratchet describes itself.** Classify each survivor by cause
      (prose-numeral, headword-glued pronunciation, phrase-block pronunciation,
      literal-pipe-as-content) and report per-cause counts. Capture one real
      exemplar per cause into `testdata/rawnotation/` — deliberately NOT under
      `testdata/entries/<lang>/`, which `capturedLanguages` walks and over which
      `TestNoRawPronunciationNotationSurvives` asserts a hard zero. Unit-test the
      classifier against those exemplars offline.
- [ ] **M2 — the number and the claims.** Lower `knownRawNotationEntries` to the
      measured count; make `atlas/define.md` a CONSUMER of it on the
      `TestREADMEQuotesThePromptsTheLoopActuallyPrints` precedent; sweep the
      claims the grep found — `atlas/define.md` fixture count (29 → 33), sweep
      width (70,897 → 70,886), non-Latin (530 → 0, with the reason), the
      attribution, and `live_property_test.go`'s own comment asserting "there is
      no public API to select one", untrue since `#23 M2` and sitting in the file
      this issue edits.
- [ ] Re-run the live ratchet unsandboxed and confirm green in BOTH directions.

**Non-goal, stated rather than left implicit:** the `pipe` family is not excluded
from the count. Those four entries define the pipe character, so their content
legitimately contains one — but narrowing the oracle is the move `lessons.md`
records as already having failed here (the narrow oracle read 0% while 2.2% of
entries showed raw pipes, and that false 0% was published in the atlas). They are
ATTRIBUTED instead, which is what "attributable rather than absorbed" asks for.

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


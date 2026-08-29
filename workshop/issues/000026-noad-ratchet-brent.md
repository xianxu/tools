---
id: 000026
status: working
deps: [tools#27]
github_issue:
created: 2026-08-27
updated: 2026-08-28
estimate_hours:
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

Not yet designed. The shape to handle is a pronunciation block carrying more than
one locale (`AmE`/`BrE`), which the current pipe-notation conversion does not
recognise. Two questions to settle before coding:

1. Is the fix in the notation converter (recognise the multi-locale block) or in
   the block splitter (this is a pronunciation field being mistaken for prose)?
2. What does the rendered output *want* to be — both pronunciations, or the one
   matching the deck's locale? **This is why the issue depends on [tools#27].**
   The question is unanswerable while "locale" is a literal inside a URL builder
   (`audiourl.go:41`); #27 makes language and locale parameters, and only then is
   "render the one matching the deck's locale" a thing the code can express.
   Triage AFTER #27 lands, not before.

## Done when

- [ ] `Brent` and `brent` render without raw `| … |` notation.
- [ ] The ratchet is re-measured and `knownRawNotationEntries` lowered to the new
      true count — not raised to 32 to make the test pass.
- [ ] The remaining raw-notation entries are re-enumerated, so the next drift is
      attributable rather than absorbed.

## Plan

- [ ] Design after triage — see the two Spec questions.

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


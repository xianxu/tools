---
id: 000026
status: open
deps: [tools#27]
github_issue:
created: 2026-08-27
updated: 2026-08-27
estimate_hours:
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


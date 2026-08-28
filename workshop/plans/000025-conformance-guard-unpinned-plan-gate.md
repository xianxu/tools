---
gate: plan-quality
issue: 25
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-27T20:40:03-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Plan commits to pinning the strict failure message, but a substitute *testing.T exposes no reader for it
          detail: |-
            Row 3's "the message naming the variable" cannot be asserted through the
            fake-T mechanism the rest of the plan uses: testing.T exposes only
            Skipped() and Failed(); the text passed to t.Fatalf at
            internal/conformance/conformance.go:78 lands in the unexported
            common.output, and Go 1.25's T.Output() is a writer, not a reader. The row
            will be dropped or become a vacuous assertion. ARCH-PURE: extract a pure
            message(reason string, err error, strict bool) string and assert on its
            return, leaving SkipOrFail as thin Fatalf/Skip glue.
          family: assertion-cannot-observe-target
          round: 1
        - id: PQ-2
          severity: Important
          title: Tree-wide Done-when is backed only by a one-time two-mode suite read, and the plan never states the enumeration it swept
          detail: |-
            Done-when row 1 asserts a standing invariant ("no test changes verdict on
            the variable alone") but the only backing is "Verify unsandboxed in both
            env states" — the read-the-output method the Problem section indicts #24
            for. ARCH-PURPOSE: name the enumerable class in the plan. There are 8
            substitute-*testing.T sites (reachable_test.go:26,47,70;
            golden_test.go:28,35,49; skiporfail_test.go:37,78) and only the
            reachable_test.go ones sit behind a conformance-routed helper — golden.go
            has no conformance reference. State that sweep, and declare mechanical
            cross-mode enforcement an explicit non-goal with its reason, rather than
            leaving a one-shot measurement reading as a guarded invariant.
          family: instance-not-class
          round: 1
        - id: PQ-3
          severity: Minor
          title: Plan adds three more copies of the goroutine + substitute-T + done-channel idiom without naming a shared helper
          detail: |-
            ARCH-DRY: the shape documented at golden_test.go:42-45 reaches eight sites
            across three files after this plan lands. A shared helper would need a new
            test-support package importable by both conformance_test and llmtest,
            which may not be worth it — but the plan should acknowledge the
            duplication and say so, rather than accrete it silently.
          family: repeated-idiom-no-helper
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-27T20:47:59-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: message(reason, err, strict) split out of SkipOrFail; asserted directly in an internal test.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Class enumerated in Done-when with a measured grep; cross-mode enforcement declared an explicit non-goal with its reason.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Duplication acknowledged with the coupling-vs-duplication reason for not extracting.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-08-27T20:51:05-07:00"
      agent: claude
      blocked: false
content_hash: 0fa37951023d86649c53e8fe15ae8a47ea276cd165932d0bc620ae9132c70ddd
---

# Gate ledger — tools#25 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-27T20:40:03-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `assertion-cannot-observe-target` Plan commits to pinning the strict failure message, but a substitute *testing.T exposes no reader for it
  Row 3's "the message naming the variable" cannot be asserted through the
  fake-T mechanism the rest of the plan uses: testing.T exposes only
  Skipped() and Failed(); the text passed to t.Fatalf at
  internal/conformance/conformance.go:78 lands in the unexported
  common.output, and Go 1.25's T.Output() is a writer, not a reader. The row
  will be dropped or become a vacuous assertion. ARCH-PURE: extract a pure
  message(reason string, err error, strict bool) string and assert on its
  return, leaving SkipOrFail as thin Fatalf/Skip glue.
- **PQ-2** [Important] `instance-not-class` Tree-wide Done-when is backed only by a one-time two-mode suite read, and the plan never states the enumeration it swept
  Done-when row 1 asserts a standing invariant ("no test changes verdict on
  the variable alone") but the only backing is "Verify unsandboxed in both
  env states" — the read-the-output method the Problem section indicts #24
  for. ARCH-PURPOSE: name the enumerable class in the plan. There are 8
  substitute-*testing.T sites (reachable_test.go:26,47,70;
  golden_test.go:28,35,49; skiporfail_test.go:37,78) and only the
  reachable_test.go ones sit behind a conformance-routed helper — golden.go
  has no conformance reference. State that sweep, and declare mechanical
  cross-mode enforcement an explicit non-goal with its reason, rather than
  leaving a one-shot measurement reading as a guarded invariant.
- **PQ-3** [Minor] `repeated-idiom-no-helper` Plan adds three more copies of the goroutine + substitute-T + done-channel idiom without naming a shared helper
  ARCH-DRY: the shape documented at golden_test.go:42-45 reaches eight sites
  across three files after this plan lands. A shared helper would need a new
  test-support package importable by both conformance_test and llmtest,
  which may not be worth it — but the plan should acknowledge the
  duplication and say so, rather than accrete it silently.

## Round 2 — 2026-08-27T20:47:59-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — message(reason, err, strict) split out of SkipOrFail; asserted directly in an internal test.
- PQ-2 — addressed — Class enumerated in Done-when with a measured grep; cross-mode enforcement declared an explicit non-goal with its reason.
- PQ-3 — addressed — Duplication acknowledged with the coupling-vs-duplication reason for not extracting.

## Round 3 — 2026-08-27T20:51:05-07:00 (claude) — passed

## Open findings

(none — every finding has been disposed)

---
gate: plan-quality
issue: 8
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-07T19:10:50-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Plan asserts Mastered reads MaxBox; it reads Box, and the pin derived from that cannot be written
          detail: |-
            schedule/progress.go:130-131 is `return p.Box >= MasteredBox`, and Mastered/MasteredBox
            have no non-test callers (play_loop.go:982 uses an unrelated boardBox), so "would disagree
            with the sitting's own display" is also false. Task 1 Step 5 asks to pin a difference
            between two identical expressions; the implementer either writes a vacuous test or changes
            the exported predicate to read MaxBox, breaking schedule/progress_test.go:158-166 and
            altering semantics for a future --play consumer. Keep the reuse, fix the rationale and the pin.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-2
          severity: Important
          title: No-deck path names two incompatible contracts and points away from the existing helper
          detail: |-
            The plan says runStats mirrors runReflect but prints "the same sentence --play prints",
            naming DEFINE_NO_CAPTURE as a cause. play_loop.go:29-32 prints an inline literal to stdout
            with exit 0 and ignores DEFINE_NO_CAPTURE; reflect.go:339-341 uses noDeckMessage
            (main.go:1194-1199) on errOut with exit 1, which is the helper that distinguishes it.
            Name the helper, the stream, and the exit code (ARCH-DRY).
          family: reuse-existing-helper
          round: 1
        - id: PQ-3
          severity: Important
          title: Mode registration names neither the modes slice nor the NArg guard, and no test would catch the miss
          detail: |-
            "Register -stats beside -reflect and -play" omits the modes slice at main.go:598-604 and the
            `*reflect && fs.NArg() != 0` refusal at main.go:654. TestModeCollision (harvest_test.go:627-631)
            hand-lists the five modes despite claiming derivation, so a -stats missing from modes reddens
            nothing and `define -stats -play` is silently swallowed — the exact bug main.go:589-593 records.
          family: unguarded-enumeration
          round: 1
        - id: PQ-4
          severity: Important
          title: No ARCH-SECURE line for a hand-editable event log whose bad timestamps silently poison the figures
          detail: |-
            Summarise folds a persisted artifact that store.YAML.Events (store/yaml.go:323-355) does not
            sanitise; sanitisation covers facts and items only (store/item.go:109,145,160). Existing schedule
            code guards zero timestamps explicitly (progress.go:112, queue.go:75). A zero or absurd At puts
            StartOfDay in year 1 and corrupts longest-streak and active-days without any visible failure.
            State what a zero/garbage At does and whether a log-sourced Form string reaches the terminal verbatim.
          family: untrusted-persisted-input
          round: 1
        - id: PQ-5
          severity: Important
          title: Added-per-day leaves unstated the deck-vs-log choice the plan argues at length for Known
          detail: |-
            store.Word.FirstSeen (store/word.go:22) and EventLookedUp in the log give different answers for
            forgotten words — the very asymmetry the plan builds Summarise's signature around
            (main.go:441, "events are kept"). One line naming the source.
          family: deck-vs-log-source
          round: 1
        - id: PQ-6
          severity: Minor
          title: ARCH-ORDER says the log is not guaranteed sorted; the contract and both implementations say it is
          detail: |-
            store/store.go:18 documents chronological order, store/mem.go:105 and store/yaml.go:352 both sort,
            and schedule/progress.go:145-148 states Fold depends on that ordering. The min/max mitigation is
            harmless; the stated basis contradicts the package the plan reuses.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-7
          severity: Minor
          title: The reuse table credits schedule.Fold with a lapse count that Progress does not carry
          detail: |-
            Progress is Box, MaxBox, LastReviewed (schedule/progress.go:52-63); the doc there notes Streak was
            deleted for having no reader. Nothing in the plan depends on lapses, but the table asserts it.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-8
          severity: Minor
          title: Plan deliberately deviates from two Done-when rows without amending the issue
          detail: |-
            Known comes from the deck, not events ("every figure derived from events"), and the clock is a
            parameter rather than a fake clock. Both deviations are improvements; record them as a
            `## Revisions` note on the issue so the close gate does not read a plan contradicting its criteria.
          family: done-when-drift
          round: 1
      blocked: true
---

# Gate ledger — tools#8 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-07T19:10:50-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `unbacked-existing-behavior-claim` Plan asserts Mastered reads MaxBox; it reads Box, and the pin derived from that cannot be written
  schedule/progress.go:130-131 is `return p.Box >= MasteredBox`, and Mastered/MasteredBox
  have no non-test callers (play_loop.go:982 uses an unrelated boardBox), so "would disagree
  with the sitting's own display" is also false. Task 1 Step 5 asks to pin a difference
  between two identical expressions; the implementer either writes a vacuous test or changes
  the exported predicate to read MaxBox, breaking schedule/progress_test.go:158-166 and
  altering semantics for a future --play consumer. Keep the reuse, fix the rationale and the pin.
- **PQ-2** [Important] `reuse-existing-helper` No-deck path names two incompatible contracts and points away from the existing helper
  The plan says runStats mirrors runReflect but prints "the same sentence --play prints",
  naming DEFINE_NO_CAPTURE as a cause. play_loop.go:29-32 prints an inline literal to stdout
  with exit 0 and ignores DEFINE_NO_CAPTURE; reflect.go:339-341 uses noDeckMessage
  (main.go:1194-1199) on errOut with exit 1, which is the helper that distinguishes it.
  Name the helper, the stream, and the exit code (ARCH-DRY).
- **PQ-3** [Important] `unguarded-enumeration` Mode registration names neither the modes slice nor the NArg guard, and no test would catch the miss
  "Register -stats beside -reflect and -play" omits the modes slice at main.go:598-604 and the
  `*reflect && fs.NArg() != 0` refusal at main.go:654. TestModeCollision (harvest_test.go:627-631)
  hand-lists the five modes despite claiming derivation, so a -stats missing from modes reddens
  nothing and `define -stats -play` is silently swallowed — the exact bug main.go:589-593 records.
- **PQ-4** [Important] `untrusted-persisted-input` No ARCH-SECURE line for a hand-editable event log whose bad timestamps silently poison the figures
  Summarise folds a persisted artifact that store.YAML.Events (store/yaml.go:323-355) does not
  sanitise; sanitisation covers facts and items only (store/item.go:109,145,160). Existing schedule
  code guards zero timestamps explicitly (progress.go:112, queue.go:75). A zero or absurd At puts
  StartOfDay in year 1 and corrupts longest-streak and active-days without any visible failure.
  State what a zero/garbage At does and whether a log-sourced Form string reaches the terminal verbatim.
- **PQ-5** [Important] `deck-vs-log-source` Added-per-day leaves unstated the deck-vs-log choice the plan argues at length for Known
  store.Word.FirstSeen (store/word.go:22) and EventLookedUp in the log give different answers for
  forgotten words — the very asymmetry the plan builds Summarise's signature around
  (main.go:441, "events are kept"). One line naming the source.
- **PQ-6** [Minor] `unbacked-existing-behavior-claim` ARCH-ORDER says the log is not guaranteed sorted; the contract and both implementations say it is
  store/store.go:18 documents chronological order, store/mem.go:105 and store/yaml.go:352 both sort,
  and schedule/progress.go:145-148 states Fold depends on that ordering. The min/max mitigation is
  harmless; the stated basis contradicts the package the plan reuses.
- **PQ-7** [Minor] `unbacked-existing-behavior-claim` The reuse table credits schedule.Fold with a lapse count that Progress does not carry
  Progress is Box, MaxBox, LastReviewed (schedule/progress.go:52-63); the doc there notes Streak was
  deleted for having no reader. Nothing in the plan depends on lapses, but the table asserts it.
- **PQ-8** [Minor] `done-when-drift` Plan deliberately deviates from two Done-when rows without amending the issue
  Known comes from the deck, not events ("every figure derived from events"), and the clock is a
  parameter rather than a fake clock. Both deviations are improvements; record them as a
  `## Revisions` note on the issue so the close gate does not read a plan contradicting its criteria.

## Open findings

- **PQ-1** [Critical] `unbacked-existing-behavior-claim` Plan asserts Mastered reads MaxBox; it reads Box, and the pin derived from that cannot be written
- **PQ-2** [Important] `reuse-existing-helper` No-deck path names two incompatible contracts and points away from the existing helper
- **PQ-3** [Important] `unguarded-enumeration` Mode registration names neither the modes slice nor the NArg guard, and no test would catch the miss
- **PQ-4** [Important] `untrusted-persisted-input` No ARCH-SECURE line for a hand-editable event log whose bad timestamps silently poison the figures
- **PQ-5** [Important] `deck-vs-log-source` Added-per-day leaves unstated the deck-vs-log choice the plan argues at length for Known
- **PQ-6** [Minor] `unbacked-existing-behavior-claim` ARCH-ORDER says the log is not guaranteed sorted; the contract and both implementations say it is
- **PQ-7** [Minor] `unbacked-existing-behavior-claim` The reuse table credits schedule.Fold with a lapse count that Progress does not carry
- **PQ-8** [Minor] `done-when-drift` Plan deliberately deviates from two Done-when rows without amending the issue

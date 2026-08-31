---
gate: plan-quality
issue: 39
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-31T11:13:02-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: D10 grants +2 to form 2.1 self-report, contradicting the issue's Revision
          detail: |-
            The Revision restricts +2 to "a correct answer in form 2.3 given WITHOUT a
            reveal" and says self-report earns +1. D10 sets Unaided from !s.Revealed in
            Apply, which cannot tell the forms apart, and play_loop.go:283-286 runs both
            in one sitting. A `y` in Recall (recall.go:41-46) is ungraded self-report and
            would earn +2. Gate Unaided on form capability the way Axis already is
            (session.go:88) — the value is written into an append-only log and cannot be
            reinterpreted later.
          family: form-capability-gate
          round: 1
        - id: PQ-2
          severity: Important
          title: T7 prints a sustainable new-word rate with no named source for `budget`
          detail: |-
            SustainableNewWords needs a daily review budget. The only candidate is
            opt.count (main.go:419, default 20), documented as a per-session cap, not a
            daily budget — so the printed advice changes meaning under `-count 5`. Name
            the input and its basis.
          family: unsourced-input
          round: 1
        - id: PQ-3
          severity: Important
          title: DailyLoad's deck-vs-log scope is unstated and unpinned
          detail: |-
            T5 names no signature and no Done-when row covers it. Fold returns progress
            for every word ever reviewed; queue.go:26-28 documents that --forget keeps
            events after deck removal, so folding the log would inflate the reported
            cost by every forgotten word.
          family: deck-is-the-roster
          round: 1
        - id: PQ-4
          severity: Important
          title: '`MaxBox` is planned as both the arithmetic clamp and a Progress field'
          detail: |-
            T1 adds MaxBox = 20 as the clamp; the concepts table adds Progress.MaxBox as
            the per-word high-water mark. Inside Answer, `p.Box < MaxBox` and
            `p.Box < p.MaxBox` are both valid Go with opposite meanings — the first
            grants every word a permanent express lane. Rename the constant.
          family: name-one-meaning
          round: 1
        - id: PQ-5
          severity: Minor
          title: T6 adds a second adjacent bool to CaptureReview right after D4 argues against bools
          detail: |-
            capture.go:128 already takes `correct bool`; adding `unaided bool` beside it
            makes play_loop.go:152 read `CaptureReview(out.Word, out.Verdict ==
            play.Correct, out.Unaided, out.Axis, opt)` — two swappable bools at the call
            site, in the same change that introduces Grade to remove one.
          family: bool-param-over-enum
          round: 1
        - id: PQ-6
          severity: Minor
          title: The plan says nothing about what re-folding existing logs does to live schedules
          detail: |-
            Fold replays the whole log under the new transition, so every word's box is
            silently re-derived. It is benign here — a word needs ten correct answers
            (>=455 days under the old 1/3/7/14/30/90 ladder) before the new interval
            exceeds the old 90-day cap — but that takes deriving and belongs in the plan.
          family: fold-replay-compat
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-31T11:16:45-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: D10 adds the SelfRated optional interface with Recall on the claimed side and Done-when 13 pinning the split.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: D11 names opt.count (main.go:419, default 20) and states the once-a-day assumption in the printed line.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: DailyLoad(deck []store.Word, prog map[string]Progress) with the missing-entry rule and Done-when 14.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: The clamp is ladderLimit; D3 gives the one-letter-apart collision as the reason.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: D12 changes the seam to CaptureReview(out play.Outcome, opt options).
          round: 2
        - id: PQ-6
          disposition: addressed
          note: D13 derives the re-fold direction table and Done-when 15 pins direction, not magnitude.
          round: 2
      findings:
        - id: PQ-7
          severity: Minor
          title: Unaided is derived in advance(), which zeroes s.Revealed before it builds the outcome
          detail: |-
            play/session.go:251 sets `s.Revealed, s.Graded = false, false` and only then
            constructs the Record outcome at :262 — the sole site that can carry a
            Correct verdict. The plan's claim row cites session.go:190 (Apply's
            InputRune arm), which is a true statement about a different line than the
            one T6 lands on. Computing Unaided from s.Revealed inside advance yields
            true for every correct answer, including one given after a reveal, and no
            Done-when row covers that negative case: 13 drives forms through an
            unrevealed correct answer and 10 pins the positive path, so both stay green.
            Capture the flag before the reset, and add the reveal-then-correct case to
            Done-when 13 rather than leaving it to the manual verification block.
          family: observe-before-state-reset
          round: 2
      blocked: false
content_hash: 37b9e398a1a5a9086274b8d573ff48f51b4eed089dc0853d4e8d0a087f6ba945
---

# Gate ledger — tools#39 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-31T11:13:02-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `form-capability-gate` D10 grants +2 to form 2.1 self-report, contradicting the issue's Revision
  The Revision restricts +2 to "a correct answer in form 2.3 given WITHOUT a
  reveal" and says self-report earns +1. D10 sets Unaided from !s.Revealed in
  Apply, which cannot tell the forms apart, and play_loop.go:283-286 runs both
  in one sitting. A `y` in Recall (recall.go:41-46) is ungraded self-report and
  would earn +2. Gate Unaided on form capability the way Axis already is
  (session.go:88) — the value is written into an append-only log and cannot be
  reinterpreted later.
- **PQ-2** [Important] `unsourced-input` T7 prints a sustainable new-word rate with no named source for `budget`
  SustainableNewWords needs a daily review budget. The only candidate is
  opt.count (main.go:419, default 20), documented as a per-session cap, not a
  daily budget — so the printed advice changes meaning under `-count 5`. Name
  the input and its basis.
- **PQ-3** [Important] `deck-is-the-roster` DailyLoad's deck-vs-log scope is unstated and unpinned
  T5 names no signature and no Done-when row covers it. Fold returns progress
  for every word ever reviewed; queue.go:26-28 documents that --forget keeps
  events after deck removal, so folding the log would inflate the reported
  cost by every forgotten word.
- **PQ-4** [Important] `name-one-meaning` `MaxBox` is planned as both the arithmetic clamp and a Progress field
  T1 adds MaxBox = 20 as the clamp; the concepts table adds Progress.MaxBox as
  the per-word high-water mark. Inside Answer, `p.Box < MaxBox` and
  `p.Box < p.MaxBox` are both valid Go with opposite meanings — the first
  grants every word a permanent express lane. Rename the constant.
- **PQ-5** [Minor] `bool-param-over-enum` T6 adds a second adjacent bool to CaptureReview right after D4 argues against bools
  capture.go:128 already takes `correct bool`; adding `unaided bool` beside it
  makes play_loop.go:152 read `CaptureReview(out.Word, out.Verdict ==
  play.Correct, out.Unaided, out.Axis, opt)` — two swappable bools at the call
  site, in the same change that introduces Grade to remove one.
- **PQ-6** [Minor] `fold-replay-compat` The plan says nothing about what re-folding existing logs does to live schedules
  Fold replays the whole log under the new transition, so every word's box is
  silently re-derived. It is benign here — a word needs ten correct answers
  (>=455 days under the old 1/3/7/14/30/90 ladder) before the new interval
  exceeds the old 90-day cap — but that takes deriving and belongs in the plan.

## Round 2 — 2026-08-31T11:16:45-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — D10 adds the SelfRated optional interface with Recall on the claimed side and Done-when 13 pinning the split.
- PQ-2 — addressed — D11 names opt.count (main.go:419, default 20) and states the once-a-day assumption in the printed line.
- PQ-3 — addressed — DailyLoad(deck []store.Word, prog map[string]Progress) with the missing-entry rule and Done-when 14.
- PQ-4 — addressed — The clamp is ladderLimit; D3 gives the one-letter-apart collision as the reason.
- PQ-5 — addressed — D12 changes the seam to CaptureReview(out play.Outcome, opt options).
- PQ-6 — addressed — D13 derives the re-fold direction table and Done-when 15 pins direction, not magnitude.

### Raised

- **PQ-7** [Minor] `observe-before-state-reset` Unaided is derived in advance(), which zeroes s.Revealed before it builds the outcome
  play/session.go:251 sets `s.Revealed, s.Graded = false, false` and only then
  constructs the Record outcome at :262 — the sole site that can carry a
  Correct verdict. The plan's claim row cites session.go:190 (Apply's
  InputRune arm), which is a true statement about a different line than the
  one T6 lands on. Computing Unaided from s.Revealed inside advance yields
  true for every correct answer, including one given after a reveal, and no
  Done-when row covers that negative case: 13 drives forms through an
  unrevealed correct answer and 10 pins the positive path, so both stay green.
  Capture the flag before the reset, and add the reveal-then-correct case to
  Done-when 13 rather than leaving it to the manual verification block.

## Open findings

- **PQ-7** [Minor] `observe-before-state-reset` Unaided is derived in advance(), which zeroes s.Revealed before it builds the outcome

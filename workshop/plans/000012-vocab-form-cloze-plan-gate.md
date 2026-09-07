---
gate: plan-quality
issue: 12
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-06T23:12:46-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: 'No task builds the item-to-question step: option order, the seed, or which item is chosen'
          detail: |-
            The Done-when row "Deterministic under a fixed seed" has no plan step. Nothing
            names the function that turns a store.Item (one Answer, N Distractors) into
            Cloze's ordered option set, names a seed, or says where the answer sits among
            the distractors — so as written it is always position 1. #7 solved this at
            play_loop.go:957 with seedFor(key, day) and pinned it with
            TestPickOptionsMovesTheAnswerAround (pick_test.go:72). Same gap covers which
            item is picked when a word holds two FormCloze items (ItemCap is 4).
          family: donewhen-without-plan-step
          round: 1
        - id: PQ-2
          severity: Important
          title: The flag is specified at both endpoints and nowhere in between; the obvious EventKind demotes the word
          detail: |-
            Outcome is the only channel out of Apply and none of its kinds carries a flag
            (advance returns OutcomeNone for Skipped). CaptureReview hardcodes
            Kind: store.EventReviewed (capture.go:143), and Fold folds every reviewed event
            (schedule/progress.go:155) with GradeOf(correct=false) mapping to GradeWrong
            (progress.go:180) — so a flag written as a reviewed event demotes the word,
            exactly what Task 6 Step 4 forbids. Name the OutcomeKind, the EventKind, and
            the capture verb; the event log is append-only, so this is not cheaply reversed.
          family: signal-path-unnamed
          round: 1
        - id: PQ-3
          severity: Important
          title: Task 3 checks that `?` does not collide, not which session states accept it
          detail: |-
            toInput passes `?` through as InputRune (play_loop.go:548-559), so collision is
            not the risk. session.go:242 is: once s.Graded, every rune means "next word", so
            `?` after answering silently advances — which is the moment a learner discovers
            a question is broken, having read the reveal. The plan also never says what a
            flag does to the question itself (advance, record nothing, score nothing).
          family: state-event-unenumerated
          round: 1
        - id: PQ-4
          severity: Important
          title: blankOut should delegate to blankStem; the plan's reason for keeping both misstates what blankOut costs
          detail: |-
            blankOut (harvest_judge.go:184) blanks only the FIRST occurrence and uses the
            answer's own length, so a stem using the word twice shows the veto judge the
            answer verbatim and `keels` becomes `___s`. That is not "some context" — the
            veto prompt asks whether a candidate ALSO fits the blank, and #10's revisions
            make that veto load-bearing. The committed golden
            (testdata/golden/veto-prompt.txt) has one uninflected occurrence, so delegating
            moves nothing.
          family: dry-parallel-implementations
          round: 1
        - id: PQ-5
          severity: Minor
          title: Cloze's Reveal drops the definition, against choice.go's recorded rationale
          detail: |-
            choice.go:82 argues at length that a recognition form revealing less than form
            2.1 "would have taught less than the easier form did". Cloze restores the
            sentence only. Settle it at plan time because it decides whether Task 5 still
            calls d.dict.Lookup for a word taking cloze.
          family: reveal-content-unweighed
          round: 1
        - id: PQ-6
          severity: Minor
          title: The unusable-item check names one instance; enumerate the class a hand-edited item can break
          detail: |-
            harvest.go:368 writes an item on ONE surviving distractor, and sanitiseItem
            neutralises without re-validating. Besides "stem no longer contains its answer",
            a file on disk can carry zero distractors, an empty answer, or a distractor
            equal to the answer (two correct options, one marked). Enumerate the fields and
            state the floor, rather than the single row.
          family: untrusted-artifact-unvalidated
          round: 1
        - id: PQ-7
          severity: Minor
          title: blankStem's guard is a fixed table plus a fixed corpus; it is the one function here that earns a fuzz
          detail: |-
            Model-authored text, folding runes, and a sibling that already shipped a
            slice-bounds panic on `Ⱥ`/`İ`. Seed a fuzz target from the leak table with the
            invariant: valid UTF-8 out, never panics, and no word-boundary occurrence of the
            answer survives. The repo already has nine fuzz targets for exactly this class.
          family: leak-guard-not-adversarial
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-06T23:16:46-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: clozeFor now owns item choice, seedFor(key, day), and the shuffle, with the answer-moves pin in Task 5.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Five-row path table names OutcomeFlag/CaptureFlag/EventFlagged; Fold-is-safe-by-construction verified at progress.go:152.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Three session states enumerated against session.go:242, routed by the InputDrop/InputQuit precedent; scores nothing.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: blankOut delegates to blankStem; the first-occurrence-only defect confirmed at harvest_judge.go:188-192.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Reveal keeps the rendered entry, with the Task 5 consequence that cloze words still need d.dict.Lookup.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: usableItem enumerates answer, stem-contains-answer, distractor count, and distractor-equals-answer.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: FuzzBlankStem seeded from the leak table with no-panic, valid-UTF-8, and no-surviving-occurrence.
          round: 2
      blocked: false
content_hash: a3ae65cae98070f52f579b7aad6d9d5576c42edf9cf68b0d99497db2ff5e5eb6
---

# Gate ledger — tools#12 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-06T23:12:46-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `donewhen-without-plan-step` No task builds the item-to-question step: option order, the seed, or which item is chosen
  The Done-when row "Deterministic under a fixed seed" has no plan step. Nothing
  names the function that turns a store.Item (one Answer, N Distractors) into
  Cloze's ordered option set, names a seed, or says where the answer sits among
  the distractors — so as written it is always position 1. #7 solved this at
  play_loop.go:957 with seedFor(key, day) and pinned it with
  TestPickOptionsMovesTheAnswerAround (pick_test.go:72). Same gap covers which
  item is picked when a word holds two FormCloze items (ItemCap is 4).
- **PQ-2** [Important] `signal-path-unnamed` The flag is specified at both endpoints and nowhere in between; the obvious EventKind demotes the word
  Outcome is the only channel out of Apply and none of its kinds carries a flag
  (advance returns OutcomeNone for Skipped). CaptureReview hardcodes
  Kind: store.EventReviewed (capture.go:143), and Fold folds every reviewed event
  (schedule/progress.go:155) with GradeOf(correct=false) mapping to GradeWrong
  (progress.go:180) — so a flag written as a reviewed event demotes the word,
  exactly what Task 6 Step 4 forbids. Name the OutcomeKind, the EventKind, and
  the capture verb; the event log is append-only, so this is not cheaply reversed.
- **PQ-3** [Important] `state-event-unenumerated` Task 3 checks that `?` does not collide, not which session states accept it
  toInput passes `?` through as InputRune (play_loop.go:548-559), so collision is
  not the risk. session.go:242 is: once s.Graded, every rune means "next word", so
  `?` after answering silently advances — which is the moment a learner discovers
  a question is broken, having read the reveal. The plan also never says what a
  flag does to the question itself (advance, record nothing, score nothing).
- **PQ-4** [Important] `dry-parallel-implementations` blankOut should delegate to blankStem; the plan's reason for keeping both misstates what blankOut costs
  blankOut (harvest_judge.go:184) blanks only the FIRST occurrence and uses the
  answer's own length, so a stem using the word twice shows the veto judge the
  answer verbatim and `keels` becomes `___s`. That is not "some context" — the
  veto prompt asks whether a candidate ALSO fits the blank, and #10's revisions
  make that veto load-bearing. The committed golden
  (testdata/golden/veto-prompt.txt) has one uninflected occurrence, so delegating
  moves nothing.
- **PQ-5** [Minor] `reveal-content-unweighed` Cloze's Reveal drops the definition, against choice.go's recorded rationale
  choice.go:82 argues at length that a recognition form revealing less than form
  2.1 "would have taught less than the easier form did". Cloze restores the
  sentence only. Settle it at plan time because it decides whether Task 5 still
  calls d.dict.Lookup for a word taking cloze.
- **PQ-6** [Minor] `untrusted-artifact-unvalidated` The unusable-item check names one instance; enumerate the class a hand-edited item can break
  harvest.go:368 writes an item on ONE surviving distractor, and sanitiseItem
  neutralises without re-validating. Besides "stem no longer contains its answer",
  a file on disk can carry zero distractors, an empty answer, or a distractor
  equal to the answer (two correct options, one marked). Enumerate the fields and
  state the floor, rather than the single row.
- **PQ-7** [Minor] `leak-guard-not-adversarial` blankStem's guard is a fixed table plus a fixed corpus; it is the one function here that earns a fuzz
  Model-authored text, folding runes, and a sibling that already shipped a
  slice-bounds panic on `Ⱥ`/`İ`. Seed a fuzz target from the leak table with the
  invariant: valid UTF-8 out, never panics, and no word-boundary occurrence of the
  answer survives. The repo already has nine fuzz targets for exactly this class.

## Round 2 — 2026-09-06T23:16:46-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — clozeFor now owns item choice, seedFor(key, day), and the shuffle, with the answer-moves pin in Task 5.
- PQ-2 — addressed — Five-row path table names OutcomeFlag/CaptureFlag/EventFlagged; Fold-is-safe-by-construction verified at progress.go:152.
- PQ-3 — addressed — Three session states enumerated against session.go:242, routed by the InputDrop/InputQuit precedent; scores nothing.
- PQ-4 — addressed — blankOut delegates to blankStem; the first-occurrence-only defect confirmed at harvest_judge.go:188-192.
- PQ-5 — addressed — Reveal keeps the rendered entry, with the Task 5 consequence that cloze words still need d.dict.Lookup.
- PQ-6 — addressed — usableItem enumerates answer, stem-contains-answer, distractor count, and distractor-equals-answer.
- PQ-7 — addressed — FuzzBlankStem seeded from the leak table with no-panic, valid-UTF-8, and no-surviving-occurrence.

## Open findings

(none — every finding has been disposed)

---
gate: plan-quality
issue: 3
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-20T17:57:15-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: The merge-conflict rationale driving the on-disk layout fails for the same word or same day on two machines, and Done-when 2 and 4 have no steps
          detail: |-
            "One file per word and an append-only day log never conflict" holds only across
            distinct words: two machines looking up the same word both write
            words/<slug>.yaml with different Lookups/LastSeen, and two machines used on the
            same day both append to events/YYYY-MM-DD.yaml, which git conflicts at the tail
            without a merge=union driver. Nothing in Tasks 1-4 produces evidence for
            Done-when 2, and the plan never says whether the event file is O_APPEND
            (never truncated, not atomic) or write-temp-then-rename (atomic, rewrites the
            day) though Done-when 4 turns on exactly that.
          round: 1
        - id: PQ-2
          severity: Important
          title: storeHistory.Add is specified to Upsert unconditionally, so failed lookups become deck entries
          detail: |-
            history.go:11 records every submitted line with a found flag on purpose, and
            replraw.go:168 passes found=code==0. Mapping Add to Upsert without conditioning
            on found makes every typo a durable words/<slug>.yaml, contradicting issue 4's
            "capture only on a successful lookup". The plan never defines what Deck()
            returns relative to Word.Found, though issues 5, 6 and 8 all consume it.
          round: 1
        - id: PQ-3
          severity: Important
          title: History.Prefix runs per keystroke and cannot report errors; no caching or write-failure strategy is stated
          detail: |-
            replraw.go:84 calls hist.Prefix inside the per-key loop (also 70, 111, 130), and
            replraw.go:65 already flags the cost. A Deck()-backed Prefix means a readdir plus
            a YAML parse of every word file on each keystroke. Neither Add nor Prefix
            (history.go:16,19) returns an error, so a mid-session write failure has nowhere
            to go; the plan covers only failure-to-open.
          round: 1
        - id: PQ-4
          severity: Important
          title: Overlap with issue 4 (capture) is undeclared, and the one-shot and replLines paths stay uncaptured without saying so
          detail: |-
            Issue 4 (deps tools#3) owns lookup counts, timestamps, REPL capture and
            degrade-on-store-failure; Task 4 delivers those for the raw path and Word carries
            Lookups. repl.go's replLines and defineOnce never touch History, so piped and
            one-shot lookups persist nothing. Name the split in Non-goals so both issues
            don't write competing Upsert call sites.
          round: 1
        - id: PQ-5
          severity: Important
          title: Slug's collision rule is left for the implementer to choose, though it fixes filenames the plan calls expensive to change
          detail: |-
            Task 1 Step 1 says "pick one rule and assert it" for hot dog vs hot-dog. The
            testdata corpus already contains "hot dog", "a priori", iPad, iPhone, MacBook and
            Amazon, so the lowercasing rule also folds Amazon/amazon and March/march. Decide
            the normalisation in the plan; keep the fuzz target for the safety property.
          round: 1
        - id: PQ-6
          severity: Minor
          title: Task 4 does not say where History is injected or what the nil default is
          detail: |-
            A deps field touches realDeps (main.go:31) and the rigs at main_test.go:14,41; a
            runEditor parameter touches twelve call sites in editorloop_test.go. replraw.go:61
            currently guarantees non-nil, so the deps route nil-panics every existing rig
            unless a default is stated.
          round: 1
        - id: PQ-7
          severity: Minor
          title: Task 1 Step 1 and Task 3 Step 2 enumerate test cases in prose; compress to one strategy line per risky function
          detail: |-
            Both lists are lossy pre-images of code that will exist within the hour. Task 2's
            suite obligations are the seam contract and should stay.
          round: 1
        - id: PQ-8
          severity: Minor
          title: go.yaml.in/yaml/v3 is not a dependency of this module and no step adds it
          detail: |-
            go.mod declares github.com/xianxu/tools with only creack/pty and golang.org/x/term,
            so the estimate's "already an ariadne dependency" does not transfer. Pin @v3.0.4,
            which is present in the shared module cache, since an @latest query needs a proxy
            this environment may not reach.
          round: 1
        - id: PQ-9
          severity: Minor
          title: The define-learn project file still records the retired brain/nous-push storage decision
          detail: |-
            workshop/projects/define-learn.md lists "Storage shape is chosen for git... nous
            push supplies sync" under design decisions taken up front, which the narrowed spec
            replaced with cwd-only, no git of any kind.
          round: 1
      blocked: true
---

# Gate ledger — tools#3 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-20T17:57:15-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] The merge-conflict rationale driving the on-disk layout fails for the same word or same day on two machines, and Done-when 2 and 4 have no steps
  "One file per word and an append-only day log never conflict" holds only across
  distinct words: two machines looking up the same word both write
  words/<slug>.yaml with different Lookups/LastSeen, and two machines used on the
  same day both append to events/YYYY-MM-DD.yaml, which git conflicts at the tail
  without a merge=union driver. Nothing in Tasks 1-4 produces evidence for
  Done-when 2, and the plan never says whether the event file is O_APPEND
  (never truncated, not atomic) or write-temp-then-rename (atomic, rewrites the
  day) though Done-when 4 turns on exactly that.
- **PQ-2** [Important] storeHistory.Add is specified to Upsert unconditionally, so failed lookups become deck entries
  history.go:11 records every submitted line with a found flag on purpose, and
  replraw.go:168 passes found=code==0. Mapping Add to Upsert without conditioning
  on found makes every typo a durable words/<slug>.yaml, contradicting issue 4's
  "capture only on a successful lookup". The plan never defines what Deck()
  returns relative to Word.Found, though issues 5, 6 and 8 all consume it.
- **PQ-3** [Important] History.Prefix runs per keystroke and cannot report errors; no caching or write-failure strategy is stated
  replraw.go:84 calls hist.Prefix inside the per-key loop (also 70, 111, 130), and
  replraw.go:65 already flags the cost. A Deck()-backed Prefix means a readdir plus
  a YAML parse of every word file on each keystroke. Neither Add nor Prefix
  (history.go:16,19) returns an error, so a mid-session write failure has nowhere
  to go; the plan covers only failure-to-open.
- **PQ-4** [Important] Overlap with issue 4 (capture) is undeclared, and the one-shot and replLines paths stay uncaptured without saying so
  Issue 4 (deps tools#3) owns lookup counts, timestamps, REPL capture and
  degrade-on-store-failure; Task 4 delivers those for the raw path and Word carries
  Lookups. repl.go's replLines and defineOnce never touch History, so piped and
  one-shot lookups persist nothing. Name the split in Non-goals so both issues
  don't write competing Upsert call sites.
- **PQ-5** [Important] Slug's collision rule is left for the implementer to choose, though it fixes filenames the plan calls expensive to change
  Task 1 Step 1 says "pick one rule and assert it" for hot dog vs hot-dog. The
  testdata corpus already contains "hot dog", "a priori", iPad, iPhone, MacBook and
  Amazon, so the lowercasing rule also folds Amazon/amazon and March/march. Decide
  the normalisation in the plan; keep the fuzz target for the safety property.
- **PQ-6** [Minor] Task 4 does not say where History is injected or what the nil default is
  A deps field touches realDeps (main.go:31) and the rigs at main_test.go:14,41; a
  runEditor parameter touches twelve call sites in editorloop_test.go. replraw.go:61
  currently guarantees non-nil, so the deps route nil-panics every existing rig
  unless a default is stated.
- **PQ-7** [Minor] Task 1 Step 1 and Task 3 Step 2 enumerate test cases in prose; compress to one strategy line per risky function
  Both lists are lossy pre-images of code that will exist within the hour. Task 2's
  suite obligations are the seam contract and should stay.
- **PQ-8** [Minor] go.yaml.in/yaml/v3 is not a dependency of this module and no step adds it
  go.mod declares github.com/xianxu/tools with only creack/pty and golang.org/x/term,
  so the estimate's "already an ariadne dependency" does not transfer. Pin @v3.0.4,
  which is present in the shared module cache, since an @latest query needs a proxy
  this environment may not reach.
- **PQ-9** [Minor] The define-learn project file still records the retired brain/nous-push storage decision
  workshop/projects/define-learn.md lists "Storage shape is chosen for git... nous
  push supplies sync" under design decisions taken up front, which the narrowed spec
  replaced with cwd-only, no git of any kind.

## Open findings

- **PQ-1** [Critical] The merge-conflict rationale driving the on-disk layout fails for the same word or same day on two machines, and Done-when 2 and 4 have no steps
- **PQ-2** [Important] storeHistory.Add is specified to Upsert unconditionally, so failed lookups become deck entries
- **PQ-3** [Important] History.Prefix runs per keystroke and cannot report errors; no caching or write-failure strategy is stated
- **PQ-4** [Important] Overlap with issue 4 (capture) is undeclared, and the one-shot and replLines paths stay uncaptured without saying so
- **PQ-5** [Important] Slug's collision rule is left for the implementer to choose, though it fixes filenames the plan calls expensive to change
- **PQ-6** [Minor] Task 4 does not say where History is injected or what the nil default is
- **PQ-7** [Minor] Task 1 Step 1 and Task 3 Step 2 enumerate test cases in prose; compress to one strategy line per risky function
- **PQ-8** [Minor] go.yaml.in/yaml/v3 is not a dependency of this module and no step adds it
- **PQ-9** [Minor] The define-learn project file still records the retired brain/nous-push storage decision

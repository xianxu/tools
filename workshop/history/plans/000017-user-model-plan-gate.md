---
gate: plan-quality
issue: 17
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-25T16:54:33-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: foldLookups re-implements the existing summariseLookups fold without naming it
          detail: |-
            cmd/define/history_cmd.go:105-144 already folds []store.ReviewEvent into
            per-word rows with the same EventLookedUp-and-Found filter, the same
            store.Key keying, and the same FirstAt/LastAt/Lookups accumulation;
            historyRow (history_cmd.go:78-86) is the plan's wordRow. The plan's DRY
            rationale claims to be creating "the one place" that decides this, and
            that place exists (ARCH-DRY). Name it and either reuse/generalise it or
            state why a second fold is warranted.
          family: reuse-existing-helper
          round: 1
        - id: PQ-2
          severity: Important
          title: D5 sites --reflect where d.deck and d.clock are still nil
          detail: |-
            --llm-check dispatches at main.go:307-309 with os.Getenv/llm.New because it
            needs no directory; d = d.withStore(opt, stderr) is main.go:339. D4 relies
            on d.clock and the store being injectable, so dispatching "before the
            argument count is judged" gets nil for both. The store-needing mode's
            pattern is --forget: arity checked in the switch at main.go:317-329,
            dispatched at main.go:341-343 after withStore. Also state what --reflect
            does when d.deck is nil under DEFINE_NO_CAPTURE (noDeckMessage, main.go:598).
          family: entrypoint-dependency-availability
          round: 1
        - id: PQ-3
          severity: Important
          title: spliceCorrections needs a property over malformed input, not five examples
          detail: |-
            It is a line scanner over human-edited text whose failure mode is silently
            discarding the learner's own writing, and byte-for-byte survival is a
            Done-when row. The five table cases miss the malformed class by
            construction: tilde fences, unterminated fences, indented fences, a marker
            with trailing whitespace, and CRLF (this repo already carries
            cmd/define/crlf.go). One line: a fuzz or property asserting everything from
            the first out-of-fence marker is preserved verbatim.
          family: malformed-input-property
          round: 1
        - id: PQ-4
          severity: Minor
          title: deckEvidence carries per-word lookups from the deck and the total from events
          detail: |-
            The Task 1 test asserts Words[0].Lookups == 4 (from store.Word.Lookups) and
            Lookups == 1 (counted from events), so the counts the prompt carries do not
            sum to the total stated beside them. D6 should say which source is
            authoritative for "how many lookups".
          family: single-source-of-truth
          round: 1
        - id: PQ-5
          severity: Minor
          title: assertGoldenFile is new surface that appears in no entity or integration table
          detail: |-
            llmtest.AssertGolden (llmtest/golden.go:30) takes an llm.Request, not a
            string, so Task 3's helper is genuinely new. The milestone-review judge
            greps the tables against the diff. Add the row, and say it reuses llmtest's
            -update flag rather than registering a second one.
          family: entity-table-completeness
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-25T16:59:17-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: D6 now calls summariseLookups (verified history_cmd.go:105-144); wordRow deleted, historyRow in the entity table as reused.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: D5 + Task 6 both dispatch after main.go:339 withStore, beside --forget; nil deck refuses via noDeckMessage (main.go:598).
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Fuzz property added asserting byte-identical survival below the first out-of-fence marker, seeded with the malformed classes.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: 'D7 states the rule: the deck decides which words are evidence, the log decides how many times and when.'
          round: 2
        - id: PQ-5
          disposition: addressed
          note: assertGoldenFile has an entity row and reads llmtest.Updating() rather than registering a second -update flag.
          round: 2
      findings:
        - id: PQ-6
          severity: Minor
          title: Task 1's test sketch restates field origins that D7 already decides, and now contradicts it
          detail: |-
            Second finding in this family. The sketch asserts got.Words[0].Text (historyRow's field
            is Word, history_cmd.go:82) and Lookups == 4 (store.Word's count; the log holds one
            found lookup, so D6+D7 yield 1) — passing it as written requires the second fold PQ-1
            removed. Rule, not instance: D7 is the one statement of where each deckEvidence field
            comes from, so the test sketches must derive from it, not restate it. Replace the
            field-by-field assertions with the property D7 claims — a --forget-removed word stops
            being evidence while its log history still counts — and compress Tasks 2-4's test
            bodies to one strategy line per risky function, as Task 4's fuzz property already does.
          family: single-source-of-truth
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-08-25T17:02:45-07:00"
      agent: claude
      dispose:
        - id: PQ-6
          disposition: addressed
          note: Task 1 is now four named properties with one strategy line each; the D7 contradiction is gone.
          round: 3
      findings:
        - id: PQ-7
          severity: Minor
          title: D1 says checkEvidence drops a claim citing an absent word; Task 2's test keeps it with evidence pruned
          detail: |-
            Third instance in this family (PQ-1 duplicated fold, PQ-4 two count sources, PQ-6 test restating D7),
            so the deliverable is the rule, not the site: a `## Decisions` entry is the sole statement of its fact
            and task bodies cite it (D1/D7) rather than restate it. D1's sentence needs the prune-then-drop-if-empty
            semantics that `TestCheckEvidenceDropsALevelClaimWithNoSupport` and the `"mixed"` case actually specify,
            then the task bodies can cite D1 instead of paraphrasing it.
          family: single-source-of-truth
          round: 3
      blocked: false
content_hash: 728d2016634861e024a7ee9421402fcb457c73cb2988fa59405c7e62ba25315e
---

# Gate ledger — tools#17 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-25T16:54:33-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `reuse-existing-helper` foldLookups re-implements the existing summariseLookups fold without naming it
  cmd/define/history_cmd.go:105-144 already folds []store.ReviewEvent into
  per-word rows with the same EventLookedUp-and-Found filter, the same
  store.Key keying, and the same FirstAt/LastAt/Lookups accumulation;
  historyRow (history_cmd.go:78-86) is the plan's wordRow. The plan's DRY
  rationale claims to be creating "the one place" that decides this, and
  that place exists (ARCH-DRY). Name it and either reuse/generalise it or
  state why a second fold is warranted.
- **PQ-2** [Important] `entrypoint-dependency-availability` D5 sites --reflect where d.deck and d.clock are still nil
  --llm-check dispatches at main.go:307-309 with os.Getenv/llm.New because it
  needs no directory; d = d.withStore(opt, stderr) is main.go:339. D4 relies
  on d.clock and the store being injectable, so dispatching "before the
  argument count is judged" gets nil for both. The store-needing mode's
  pattern is --forget: arity checked in the switch at main.go:317-329,
  dispatched at main.go:341-343 after withStore. Also state what --reflect
  does when d.deck is nil under DEFINE_NO_CAPTURE (noDeckMessage, main.go:598).
- **PQ-3** [Important] `malformed-input-property` spliceCorrections needs a property over malformed input, not five examples
  It is a line scanner over human-edited text whose failure mode is silently
  discarding the learner's own writing, and byte-for-byte survival is a
  Done-when row. The five table cases miss the malformed class by
  construction: tilde fences, unterminated fences, indented fences, a marker
  with trailing whitespace, and CRLF (this repo already carries
  cmd/define/crlf.go). One line: a fuzz or property asserting everything from
  the first out-of-fence marker is preserved verbatim.
- **PQ-4** [Minor] `single-source-of-truth` deckEvidence carries per-word lookups from the deck and the total from events
  The Task 1 test asserts Words[0].Lookups == 4 (from store.Word.Lookups) and
  Lookups == 1 (counted from events), so the counts the prompt carries do not
  sum to the total stated beside them. D6 should say which source is
  authoritative for "how many lookups".
- **PQ-5** [Minor] `entity-table-completeness` assertGoldenFile is new surface that appears in no entity or integration table
  llmtest.AssertGolden (llmtest/golden.go:30) takes an llm.Request, not a
  string, so Task 3's helper is genuinely new. The milestone-review judge
  greps the tables against the diff. Add the row, and say it reuses llmtest's
  -update flag rather than registering a second one.

## Round 2 — 2026-08-25T16:59:17-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — D6 now calls summariseLookups (verified history_cmd.go:105-144); wordRow deleted, historyRow in the entity table as reused.
- PQ-2 — addressed — D5 + Task 6 both dispatch after main.go:339 withStore, beside --forget; nil deck refuses via noDeckMessage (main.go:598).
- PQ-3 — addressed — Fuzz property added asserting byte-identical survival below the first out-of-fence marker, seeded with the malformed classes.
- PQ-4 — addressed — D7 states the rule: the deck decides which words are evidence, the log decides how many times and when.
- PQ-5 — addressed — assertGoldenFile has an entity row and reads llmtest.Updating() rather than registering a second -update flag.

### Raised

- **PQ-6** [Minor] `single-source-of-truth` Task 1's test sketch restates field origins that D7 already decides, and now contradicts it
  Second finding in this family. The sketch asserts got.Words[0].Text (historyRow's field
  is Word, history_cmd.go:82) and Lookups == 4 (store.Word's count; the log holds one
  found lookup, so D6+D7 yield 1) — passing it as written requires the second fold PQ-1
  removed. Rule, not instance: D7 is the one statement of where each deckEvidence field
  comes from, so the test sketches must derive from it, not restate it. Replace the
  field-by-field assertions with the property D7 claims — a --forget-removed word stops
  being evidence while its log history still counts — and compress Tasks 2-4's test
  bodies to one strategy line per risky function, as Task 4's fuzz property already does.

## Round 3 — 2026-08-25T17:02:45-07:00 (claude) — passed

### Disposed

- PQ-6 — addressed — Task 1 is now four named properties with one strategy line each; the D7 contradiction is gone.

### Raised

- **PQ-7** [Minor] `single-source-of-truth` D1 says checkEvidence drops a claim citing an absent word; Task 2's test keeps it with evidence pruned
  Third instance in this family (PQ-1 duplicated fold, PQ-4 two count sources, PQ-6 test restating D7),
  so the deliverable is the rule, not the site: a `## Decisions` entry is the sole statement of its fact
  and task bodies cite it (D1/D7) rather than restate it. D1's sentence needs the prune-then-drop-if-empty
  semantics that `TestCheckEvidenceDropsALevelClaimWithNoSupport` and the `"mixed"` case actually specify,
  then the task bodies can cite D1 instead of paraphrasing it.

## Open findings

- **PQ-7** [Minor] `single-source-of-truth` D1 says checkEvidence drops a claim citing an absent word; Task 2's test keeps it with evidence pruned

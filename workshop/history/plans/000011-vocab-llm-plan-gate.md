---
gate: plan-quality
issue: 11
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-22T18:09:18-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: The 2026-08-22 revision's five "Task list changes" never landed in any task body
          detail: |-
            Task 4 still scripts invented literals and specifies no Cassette at all (no
            file, no key derivation, no -record registration, no miss behaviour); Task 5's
            tests still assert Reply{Text: "flattering, servile"}; Task 8 Step 2 still says
            "blocked" though Delta 1 retracts it; Measured fact 2 and the whole "Open
            question" section still assert the retracted proxy facts unmarked; and the
            preamble assertions Delta 4 adds to Tasks 12/13 have no field to read - Usage
            carries only InputTokens/OutputTokens/Duration, no cache_creation/cache_read.
            Enumerate all five bullets and sweep them into the bodies in this round.
          family: revision-not-propagated
          round: 1
        - id: PQ-2
          severity: Important
          title: Task 8 says "commit the captured SSE sample" but no such artifact exists on disk
          detail: |-
            Delta 3 demotes Task 8 from recording to committing, citing a capture whose only
            surviving record is the prose event list in the revision. A find over the repo
            returns no .sse file and git status is clean, so producing it means hand-writing
            the frames - which the plan itself forbids. Keep a real recording step in Task 8.
          family: unpersisted-measurement
          round: 1
        - id: PQ-3
          severity: Important
          title: Fake envelope is single-text-block, but opus-5 with adaptive thinking always returns a thinking block first
          detail: |-
            The plan leaves thinking unset on claude-opus-5 deliberately, so live content
            arrays lead with a thinking block (SDK message.go:3066-3074, messageutil.go:91),
            yet the fake serves content:[{type:"text"}] and the plan never states how
            Response.Text is derived from a multi-block array. Response.Text = Content[0].Text
            passes the whole suite against the fake and returns "" live, breaking decode.
          family: fake-models-unobserved-shape
          round: 1
        - id: PQ-4
          severity: Important
          title: Golden and Cassette independently render the same Request tuple (ARCH-DRY)
          detail: |-
            Task 11's AssertGolden renders model/effort/system/prompt/schema to text; Delta 5's
            Cassette hashes a rendering of the identical tuple. Two renderers drift, and then a
            golden update leaves a stale cassette still matching. Name one canonical
            renderRequest that Golden prints and Cassette hashes.
          family: single-request-rendering
          round: 1
        - id: PQ-5
          severity: Important
          title: Task 10 enumerates seven decode cases in prose instead of naming a strategy
          detail: |-
            decode parses arbitrary model output and is what Done-when 3 rests on. Compress the
            seven bullets to one strategy line and add a fuzz target seeded with the fenced,
            truncated and array forms, with the property that it never panics and never
            partially populates.
          family: test-enumeration-in-prose
          round: 1
        - id: PQ-6
          severity: Important
          title: The relocated degradation Done-when landed in no destination issue
          detail: |-
            The revision says the obligation moves to #6, #12 and #13. Verified:
            000012-vocab-form-cloze.md:56 already carried it beforehand, #6's three Done-when
            rows contain nothing about the seam being unavailable, #13 covers malformed-or-refused
            rather than unavailable, and #11's own Done-when checkbox 1 still stands unstruck.
          family: revision-not-propagated
          round: 1
        - id: PQ-7
          severity: Minor
          title: Fake.next iterates a Go map, so overlapping scripted keys serve a random reply
          detail: |-
            f.replies is ranged over with substring matching; when two scripted keys both match
            one prompt the served reply is nondeterministic and the test flakes. Use an ordered
            matcher slice, first match wins.
          family: nondeterministic-fake-matching
          round: 1
        - id: PQ-8
          severity: Minor
          title: Task 1 Step 1 runs go mod tidy before anything imports the SDK
          detail: |-
            Nothing imports either module at that point, so tidy removes both requires and the
            stated expectation that go.mod gains them as direct requires will not hold. Move the
            tidy after the first import.
          family: tidy-before-import
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-22T18:22:04-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Four of five task bullets landed; the retracted-facts sweep did not — plan :35, :246 and :1525 still assert the retracted proxy blocker unmarked.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: All three captures now tracked in git, recorded by the committed scripts/llm-probe.sh.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Fake emits thinking-then-text, Response.Text joins text blocks, TestTextSkipsThinkingBlock serves the real capture.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Task 11 names one renderRequest, printed by Golden and hashed by Cassette, with a test that a prompt edit moves both.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: One strategy line plus a seeded fuzz target with a stated invariant.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: 'Verified in all three destinations: 000006:38, 000012:56, 000013:52; #11''s own row struck in place.'
          round: 2
        - id: PQ-7
          disposition: addressed
          note: Ordered matchers, first-match-wins, with the rationale recorded; leftover replies refs at :946/:960 folded into PQ-1.
          round: 2
        - id: PQ-8
          disposition: addressed
          note: Task 1 Step 1 now forbids tidy there and defers it to Tasks 5 and 9.
          round: 2
      findings:
        - id: PQ-9
          severity: Important
          title: The plan's response model is narrower than its own committed captures, in block order and in stop_reason
          detail: |-
            Second in this family, so the fix is the rule, not the site: every field of every
            committed capture must be reconciled against the response model and the assertions
            built on it. Measured prevalence, two dimensions from two captures — message-thinking
            is [thinking, text] but message-schema is [thinking, text, thinking], so Task 8 Step 1
            and Task 4 Step 4 (plan :1069) assert a live block sequence the plan's own artifact
            contradicts; and message-schema carries stop_reason max_tokens, which the taxonomy
            never maps (only "refusal" is special-cased). That capture's payload is
            {"verdict":"yes", "reason":": Ā"} at output_tokens 512 against max_tokens 512 —
            syntactically valid, every required field present, semantically truncated. decode
            returns it with a nil error, so Task 10's "never partially populates" invariant holds
            while Done-when 3 silently fails. The taxonomy is what all five consumers branch on,
            so this is cheap now and a contract break later (ARCH-MOCK).
          family: fake-models-unobserved-shape
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-08-22T18:29:47-07:00"
      agent: claude
      dispose:
        - id: PQ-9
          disposition: addressed
          note: |-
            Verified field-by-field against all three JSON captures on disk; the block-order,
            stop_reason, thinking-key, stop_details and split-cache findings all landed.
          round: 3
        - id: PQ-1
          disposition: not-addressed
          note: |-
            4 of 5 bullets landed; Measured fact 2 (plan :34-38) and the whole Open question
            section (plan :1665-1680) still assert the proxy facts Delta 1 retracts.
          round: 3
      findings:
        - id: PQ-10
          severity: Important
          title: The committed SSE capture has no thinking or signature frames, so the streaming path's block preservation is untestable
          detail: |-
            This is the 3rd finding in family fake-models-unobserved-shape. Do not fix this
            instance; the rule is: a capture is evidence only for the shape its recording
            conditions elicit, so every probe must be recorded under conditions that produce
            the shape the fake models, and those conditions live in the probe script beside
            the capture. Measured prevalence 3, all in scripts/llm-probe.sh: probe_blocks
            (trivial prompt returned ["text"] alone - PQ-3), probe_schema (max_tokens 512 ate
            the budget - PQ-9), probe_stream (still "Say: one two three" at max_tokens 128).
            stream-sample.sse opens content_block_start index 0 on a text block; the SDK
            shows thinking_delta/signature_delta at message.go:7093,7095,8214. So a Stream
            that drops thinking blocks, or that feeds thinking deltas to onDelta, passes the
            whole fake-backed suite while breaking the byte-for-byte echo that #16 is named
            as depending on (ARCH-MOCK).
          family: fake-models-unobserved-shape
          round: 3
        - id: PQ-11
          severity: Minor
          title: Two illustrative code blocks do not compile as written
          detail: |-
            Task 1 Step 2's Block struct uses json.RawMessage but llm.go's import block lists
            only context and time, and Step 3 asserts `go build ./internal/...` exits 0. Task 4
            Step 1's NewFake initializes `replies: map[string][]Reply{}` and Script writes
            f.replies[match], but the struct field became `matchers []matcher` in the PQ-7 fix
            and `matcher` is never defined. Plan code blocks get pasted verbatim, so a half-applied
            refactor in one propagates.
          family: plan-code-not-buildable
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-08-22T18:38:14-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: All five task-body changes landed; measured fact 2 and the Open question are corrected in place, and Usage now has the preamble/thinking fields Tasks 12/13 read.
          round: 4
        - id: PQ-10
          disposition: addressed
          note: llm-probe.sh verify declares each capture's required shape and record stages-verifies-promotes; stream-sample.sse now carries thinking_delta and signature_delta.
          round: 4
        - id: PQ-11
          disposition: addressed
          note: 'Both blocks extracted from the plan and built: contract block compiles with encoding/json; fake block compiles and vets with matcher defined and matchers used throughout.'
          round: 4
      blocked: false
    - "n": 5
      timestamp: "2026-08-22T18:42:22-07:00"
      agent: claude
      blocked: false
content_hash: c356ecb762568be73e368186284b0f64ff8e41cd47f496352cfd96e84473399c
---

# Gate ledger — tools#11 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-22T18:09:18-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `revision-not-propagated` The 2026-08-22 revision's five "Task list changes" never landed in any task body
  Task 4 still scripts invented literals and specifies no Cassette at all (no
  file, no key derivation, no -record registration, no miss behaviour); Task 5's
  tests still assert Reply{Text: "flattering, servile"}; Task 8 Step 2 still says
  "blocked" though Delta 1 retracts it; Measured fact 2 and the whole "Open
  question" section still assert the retracted proxy facts unmarked; and the
  preamble assertions Delta 4 adds to Tasks 12/13 have no field to read - Usage
  carries only InputTokens/OutputTokens/Duration, no cache_creation/cache_read.
  Enumerate all five bullets and sweep them into the bodies in this round.
- **PQ-2** [Important] `unpersisted-measurement` Task 8 says "commit the captured SSE sample" but no such artifact exists on disk
  Delta 3 demotes Task 8 from recording to committing, citing a capture whose only
  surviving record is the prose event list in the revision. A find over the repo
  returns no .sse file and git status is clean, so producing it means hand-writing
  the frames - which the plan itself forbids. Keep a real recording step in Task 8.
- **PQ-3** [Important] `fake-models-unobserved-shape` Fake envelope is single-text-block, but opus-5 with adaptive thinking always returns a thinking block first
  The plan leaves thinking unset on claude-opus-5 deliberately, so live content
  arrays lead with a thinking block (SDK message.go:3066-3074, messageutil.go:91),
  yet the fake serves content:[{type:"text"}] and the plan never states how
  Response.Text is derived from a multi-block array. Response.Text = Content[0].Text
  passes the whole suite against the fake and returns "" live, breaking decode.
- **PQ-4** [Important] `single-request-rendering` Golden and Cassette independently render the same Request tuple (ARCH-DRY)
  Task 11's AssertGolden renders model/effort/system/prompt/schema to text; Delta 5's
  Cassette hashes a rendering of the identical tuple. Two renderers drift, and then a
  golden update leaves a stale cassette still matching. Name one canonical
  renderRequest that Golden prints and Cassette hashes.
- **PQ-5** [Important] `test-enumeration-in-prose` Task 10 enumerates seven decode cases in prose instead of naming a strategy
  decode parses arbitrary model output and is what Done-when 3 rests on. Compress the
  seven bullets to one strategy line and add a fuzz target seeded with the fenced,
  truncated and array forms, with the property that it never panics and never
  partially populates.
- **PQ-6** [Important] `revision-not-propagated` The relocated degradation Done-when landed in no destination issue
  The revision says the obligation moves to #6, #12 and #13. Verified:
  000012-vocab-form-cloze.md:56 already carried it beforehand, #6's three Done-when
  rows contain nothing about the seam being unavailable, #13 covers malformed-or-refused
  rather than unavailable, and #11's own Done-when checkbox 1 still stands unstruck.
- **PQ-7** [Minor] `nondeterministic-fake-matching` Fake.next iterates a Go map, so overlapping scripted keys serve a random reply
  f.replies is ranged over with substring matching; when two scripted keys both match
  one prompt the served reply is nondeterministic and the test flakes. Use an ordered
  matcher slice, first match wins.
- **PQ-8** [Minor] `tidy-before-import` Task 1 Step 1 runs go mod tidy before anything imports the SDK
  Nothing imports either module at that point, so tidy removes both requires and the
  stated expectation that go.mod gains them as direct requires will not hold. Move the
  tidy after the first import.

## Round 2 — 2026-08-22T18:22:04-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — not-addressed — Four of five task bullets landed; the retracted-facts sweep did not — plan :35, :246 and :1525 still assert the retracted proxy blocker unmarked.
- PQ-2 — addressed — All three captures now tracked in git, recorded by the committed scripts/llm-probe.sh.
- PQ-3 — addressed — Fake emits thinking-then-text, Response.Text joins text blocks, TestTextSkipsThinkingBlock serves the real capture.
- PQ-4 — addressed — Task 11 names one renderRequest, printed by Golden and hashed by Cassette, with a test that a prompt edit moves both.
- PQ-5 — addressed — One strategy line plus a seeded fuzz target with a stated invariant.
- PQ-6 — addressed — Verified in all three destinations: 000006:38, 000012:56, 000013:52; #11's own row struck in place.
- PQ-7 — addressed — Ordered matchers, first-match-wins, with the rationale recorded; leftover replies refs at :946/:960 folded into PQ-1.
- PQ-8 — addressed — Task 1 Step 1 now forbids tidy there and defers it to Tasks 5 and 9.

### Raised

- **PQ-9** [Important] `fake-models-unobserved-shape` The plan's response model is narrower than its own committed captures, in block order and in stop_reason
  Second in this family, so the fix is the rule, not the site: every field of every
  committed capture must be reconciled against the response model and the assertions
  built on it. Measured prevalence, two dimensions from two captures — message-thinking
  is [thinking, text] but message-schema is [thinking, text, thinking], so Task 8 Step 1
  and Task 4 Step 4 (plan :1069) assert a live block sequence the plan's own artifact
  contradicts; and message-schema carries stop_reason max_tokens, which the taxonomy
  never maps (only "refusal" is special-cased). That capture's payload is
  {"verdict":"yes", "reason":": Ā"} at output_tokens 512 against max_tokens 512 —
  syntactically valid, every required field present, semantically truncated. decode
  returns it with a nil error, so Task 10's "never partially populates" invariant holds
  while Done-when 3 silently fails. The taxonomy is what all five consumers branch on,
  so this is cheap now and a contract break later (ARCH-MOCK).

## Round 3 — 2026-08-22T18:29:47-07:00 (claude) — BLOCKED

### Disposed

- PQ-9 — addressed — Verified field-by-field against all three JSON captures on disk; the block-order,
stop_reason, thinking-key, stop_details and split-cache findings all landed.
- PQ-1 — not-addressed — 4 of 5 bullets landed; Measured fact 2 (plan :34-38) and the whole Open question
section (plan :1665-1680) still assert the proxy facts Delta 1 retracts.

### Raised

- **PQ-10** [Important] `fake-models-unobserved-shape` The committed SSE capture has no thinking or signature frames, so the streaming path's block preservation is untestable
  This is the 3rd finding in family fake-models-unobserved-shape. Do not fix this
  instance; the rule is: a capture is evidence only for the shape its recording
  conditions elicit, so every probe must be recorded under conditions that produce
  the shape the fake models, and those conditions live in the probe script beside
  the capture. Measured prevalence 3, all in scripts/llm-probe.sh: probe_blocks
  (trivial prompt returned ["text"] alone - PQ-3), probe_schema (max_tokens 512 ate
  the budget - PQ-9), probe_stream (still "Say: one two three" at max_tokens 128).
  stream-sample.sse opens content_block_start index 0 on a text block; the SDK
  shows thinking_delta/signature_delta at message.go:7093,7095,8214. So a Stream
  that drops thinking blocks, or that feeds thinking deltas to onDelta, passes the
  whole fake-backed suite while breaking the byte-for-byte echo that #16 is named
  as depending on (ARCH-MOCK).
- **PQ-11** [Minor] `plan-code-not-buildable` Two illustrative code blocks do not compile as written
  Task 1 Step 2's Block struct uses json.RawMessage but llm.go's import block lists
  only context and time, and Step 3 asserts `go build ./internal/...` exits 0. Task 4
  Step 1's NewFake initializes `replies: map[string][]Reply{}` and Script writes
  f.replies[match], but the struct field became `matchers []matcher` in the PQ-7 fix
  and `matcher` is never defined. Plan code blocks get pasted verbatim, so a half-applied
  refactor in one propagates.

## Round 4 — 2026-08-22T18:38:14-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — All five task-body changes landed; measured fact 2 and the Open question are corrected in place, and Usage now has the preamble/thinking fields Tasks 12/13 read.
- PQ-10 — addressed — llm-probe.sh verify declares each capture's required shape and record stages-verifies-promotes; stream-sample.sse now carries thinking_delta and signature_delta.
- PQ-11 — addressed — Both blocks extracted from the plan and built: contract block compiles with encoding/json; fake block compiles and vets with matcher defined and matchers used throughout.

## Round 5 — 2026-08-22T18:42:22-07:00 (claude) — passed

## Open findings

(none — every finding has been disposed)

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

## Open findings

- **PQ-1** [Important] `revision-not-propagated` The 2026-08-22 revision's five "Task list changes" never landed in any task body
- **PQ-2** [Important] `unpersisted-measurement` Task 8 says "commit the captured SSE sample" but no such artifact exists on disk
- **PQ-3** [Important] `fake-models-unobserved-shape` Fake envelope is single-text-block, but opus-5 with adaptive thinking always returns a thinking block first
- **PQ-4** [Important] `single-request-rendering` Golden and Cassette independently render the same Request tuple (ARCH-DRY)
- **PQ-5** [Important] `test-enumeration-in-prose` Task 10 enumerates seven decode cases in prose instead of naming a strategy
- **PQ-6** [Important] `revision-not-propagated` The relocated degradation Done-when landed in no destination issue
- **PQ-7** [Minor] `nondeterministic-fake-matching` Fake.next iterates a Go map, so overlapping scripted keys serve a random reply
- **PQ-8** [Minor] `tidy-before-import` Task 1 Step 1 runs go mod tidy before anything imports the SDK

---
gate: plan-quality
issue: 83
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-10-08T13:38:16-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Timing properties (c) and (e) are false for windows that run past the end of the recording
          detail: 'build_segments stops at times[-1] and warp adds nothing past the last segment (cut.py:57-76). The past-end guard only checks the stamp ct (cut.py:101-103), so a caption stamped near the end gets an output window shorter than its hold, or zero length when prev_end > times[-1]. The plan must decide: clip both properties to [0, times[-1]], or change the model (hold the last frame, or reject windows that run past the end).'
          family: property-contradicts-model
          round: 1
        - id: PQ-2
          severity: Important
          title: warp ported as written is O(N^2) per cut, and no workload envelope is declared
          detail: cut.py calls warp once per event and walks every segment from the start each time (cut.py:69-76,107). Declare the batch workload (event count and size, target runtime), specify one monotone pass over the sorted times, and include a large cast in the property tests (ARCH-CONSTRAINTS).
          family: operating-envelope-undeclared
          round: 1
        - id: PQ-3
          severity: Important
          title: The asciinema fake has no live conformance check against the real binary
          detail: Add a test built with //go:build conformance that runs real asciinema rec --headless --return -f asciicast-v3 --window-size 95x36 --command 'printf hi; exit 3'. Assert exit code 3 and that the output parses with parseCast; call conformance.SkipOrFail when asciinema is absent (ARCH-MOCK).
          family: fake-without-live-conformance
          round: 1
        - id: PQ-4
          severity: Minor
          title: Goal, Architecture, Tech Stack, entity table and file tree still describe byte-identity and pyjson
          detail: The supersedes revision overrides them correctly. Add a one-line pointer under the Goal so a top-down reader is not misled.
          family: superseded-text-unmarked
          round: 1
        - id: PQ-5
          severity: Minor
          title: M1 Tasks 1-2 and M2 Tasks 5-6 list test cases instead of one strategy line per risky function
          detail: 'Replace the lists with: parseCast and parseCaptions fuzzed, seeded with malformed lines; Task 3''s property approach is already the right shape.'
          family: test-prose-enumeration
          round: 1
        - id: PQ-6
          severity: Minor
          title: Overlapping debounced PUT /notes requests can finish out of order and overwrite newer notes
          detail: State that the viewer chains PUTs (one at a time, latest body wins) and that a second tab is last-writer-wins.
          family: ordering-unstated
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-10-08T13:38:50-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: not-addressed
          note: Plan unchanged since round 1 (HEAD e080bc5 predates the round); cut.py:53-76,101-103 still confirm the end-of-recording truncation.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: No envelope, no monotone-pass warp, no large-cast property test; warp at cut.py:69-76 and inside at cut.py:55 are both linear per call.
          round: 2
        - id: PQ-3
          disposition: not-addressed
          note: No conformance-tagged test against the real asciinema binary in Task 5 or the revisions.
          round: 2
        - id: PQ-4
          disposition: not-addressed
          note: Goal/Architecture/Tech Stack/file tree still describe byte-identity and pyjson with no pointer to the superseding revision.
          round: 2
        - id: PQ-5
          disposition: not-addressed
          note: Revision Tasks 1-2 and M2 Tasks 5-6 still enumerate cases; no fuzz strategy for parseCast/parseCaptions.
          round: 2
        - id: PQ-6
          disposition: not-addressed
          note: Debounced PUT ordering and multi-tab policy still unstated in Task 6.
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-10-08T13:39:49-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: The last frame is held through any window that runs past the last event, so (c) and (e) always hold and (a) allows one trailing hold event.
          round: 3
        - id: PQ-2
          disposition: addressed
          note: Workload limits declared, cursor-based warp and window passes, and a 2x10^5-event property test plus a test against a simple reference warp.
          round: 3
        - id: PQ-3
          disposition: addressed
          note: record_conformance_test.go runs the real asciinema (exit 3, parseCast, 95 cols) and calls SkipOrFail when it is missing; the fake shares the same checks.
          round: 3
        - id: PQ-4
          disposition: addressed
          note: A pointer to the Revisions section now sits under the Goal.
          round: 3
        - id: PQ-5
          disposition: addressed
          note: Case lists replaced by fuzz targets seeded with malformed input, plus the properties and mutation-checked guard rows.
          round: 3
        - id: PQ-6
          disposition: addressed
          note: At most one PUT in flight, latest text wins via a dirty flag, a second tab is last-writer-wins, and this is documented in --help.
          round: 3
      blocked: false
    - "n": 4
      timestamp: "2026-10-08T13:40:54-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Trailing hold event extends the timeline; (a), (c), (e) restated to hold unconditionally.
          round: 4
        - id: PQ-2
          disposition: addressed
          note: Envelope declared; cursor-based linear warp with naive-reference cross-check and wall-clock bound.
          round: 4
        - id: PQ-3
          disposition: addressed
          note: record_conformance_test.go runs real asciinema; fake and real share one assertion table.
          round: 4
        - id: PQ-4
          disposition: addressed
          note: Head-of-file pointer marks byte-identity/pyjson/oracle as superseded by the Revisions.
          round: 4
        - id: PQ-5
          disposition: addressed
          note: One strategy line per risky function (fuzz seeds + invariants, mutation checks).
          round: 4
        - id: PQ-6
          disposition: addressed
          note: Single in-flight PUT with dirty flag, atomic server write, multi-tab semantics stated.
          round: 4
      blocked: false
content_hash: c74620481b706cab3c46c720f265138a5290e1c7a9c13855a9d0b593ab63d5f1
---

# Gate ledger — tools#83 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-08T13:38:16-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `property-contradicts-model` Timing properties (c) and (e) are false for windows that run past the end of the recording
  build_segments stops at times[-1] and warp adds nothing past the last segment (cut.py:57-76). The past-end guard only checks the stamp ct (cut.py:101-103), so a caption stamped near the end gets an output window shorter than its hold, or zero length when prev_end > times[-1]. The plan must decide: clip both properties to [0, times[-1]], or change the model (hold the last frame, or reject windows that run past the end).
- **PQ-2** [Important] `operating-envelope-undeclared` warp ported as written is O(N^2) per cut, and no workload envelope is declared
  cut.py calls warp once per event and walks every segment from the start each time (cut.py:69-76,107). Declare the batch workload (event count and size, target runtime), specify one monotone pass over the sorted times, and include a large cast in the property tests (ARCH-CONSTRAINTS).
- **PQ-3** [Important] `fake-without-live-conformance` The asciinema fake has no live conformance check against the real binary
  Add a test built with //go:build conformance that runs real asciinema rec --headless --return -f asciicast-v3 --window-size 95x36 --command 'printf hi; exit 3'. Assert exit code 3 and that the output parses with parseCast; call conformance.SkipOrFail when asciinema is absent (ARCH-MOCK).
- **PQ-4** [Minor] `superseded-text-unmarked` Goal, Architecture, Tech Stack, entity table and file tree still describe byte-identity and pyjson
  The supersedes revision overrides them correctly. Add a one-line pointer under the Goal so a top-down reader is not misled.
- **PQ-5** [Minor] `test-prose-enumeration` M1 Tasks 1-2 and M2 Tasks 5-6 list test cases instead of one strategy line per risky function
  Replace the lists with: parseCast and parseCaptions fuzzed, seeded with malformed lines; Task 3's property approach is already the right shape.
- **PQ-6** [Minor] `ordering-unstated` Overlapping debounced PUT /notes requests can finish out of order and overwrite newer notes
  State that the viewer chains PUTs (one at a time, latest body wins) and that a second tab is last-writer-wins.

## Round 2 — 2026-10-08T13:38:50-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — not-addressed — Plan unchanged since round 1 (HEAD e080bc5 predates the round); cut.py:53-76,101-103 still confirm the end-of-recording truncation.
- PQ-2 — not-addressed — No envelope, no monotone-pass warp, no large-cast property test; warp at cut.py:69-76 and inside at cut.py:55 are both linear per call.
- PQ-3 — not-addressed — No conformance-tagged test against the real asciinema binary in Task 5 or the revisions.
- PQ-4 — not-addressed — Goal/Architecture/Tech Stack/file tree still describe byte-identity and pyjson with no pointer to the superseding revision.
- PQ-5 — not-addressed — Revision Tasks 1-2 and M2 Tasks 5-6 still enumerate cases; no fuzz strategy for parseCast/parseCaptions.
- PQ-6 — not-addressed — Debounced PUT ordering and multi-tab policy still unstated in Task 6.

## Round 3 — 2026-10-08T13:39:49-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — The last frame is held through any window that runs past the last event, so (c) and (e) always hold and (a) allows one trailing hold event.
- PQ-2 — addressed — Workload limits declared, cursor-based warp and window passes, and a 2x10^5-event property test plus a test against a simple reference warp.
- PQ-3 — addressed — record_conformance_test.go runs the real asciinema (exit 3, parseCast, 95 cols) and calls SkipOrFail when it is missing; the fake shares the same checks.
- PQ-4 — addressed — A pointer to the Revisions section now sits under the Goal.
- PQ-5 — addressed — Case lists replaced by fuzz targets seeded with malformed input, plus the properties and mutation-checked guard rows.
- PQ-6 — addressed — At most one PUT in flight, latest text wins via a dirty flag, a second tab is last-writer-wins, and this is documented in --help.

## Round 4 — 2026-10-08T13:40:54-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Trailing hold event extends the timeline; (a), (c), (e) restated to hold unconditionally.
- PQ-2 — addressed — Envelope declared; cursor-based linear warp with naive-reference cross-check and wall-clock bound.
- PQ-3 — addressed — record_conformance_test.go runs real asciinema; fake and real share one assertion table.
- PQ-4 — addressed — Head-of-file pointer marks byte-identity/pyjson/oracle as superseded by the Revisions.
- PQ-5 — addressed — One strategy line per risky function (fuzz seeds + invariants, mutation checks).
- PQ-6 — addressed — Single in-flight PUT with dirty flag, atomic server write, multi-tab semantics stated.

## Open findings

(none — every finding has been disposed)

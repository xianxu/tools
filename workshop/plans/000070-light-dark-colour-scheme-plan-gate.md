---
gate: plan-quality
issue: 70
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-17T21:33:51-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: ClearScheme's os.Remove(dir) deletes a symlinked config dir even when its target holds other files
          detail: 'The plan''s comment says os.Remove(dir) fails harmlessly when something else lives there. That is false for a symlink: os.Remove unlinks it and returns nil (checked here: a link to a non-empty dir was deleted and the target''s file survived). A stow or chezmoi-managed ~/.config/define loses its link on /scheme auto. Remove only a real empty directory (syscall.Rmdir gives ENOTDIR on a symlink, or Lstat and require IsDir), and add a store test with a symlinked dir.'
          family: cleanup-removes-only-owned-residue
          round: 1
        - id: PQ-2
          severity: Minor
          title: Task 16 test 5 (dropped reply is silent) asserts no notice while y's own drop notice may already be posted
          detail: The y write returning only proves the reply chunk was handled. readInput then drops y and posts "input full" at the same time as the no-notice assertion, so the test can fail when the code is correct. Use a zero-length io.Pipe write as the barrier (it is delivered as a Read), assert, then write y and wait for the notice.
          family: test-barrier-orders-observation
          round: 1
        - id: PQ-3
          severity: Minor
          title: An aborted swallow after ESC ] 11 ; re-decodes the rest of the reply as typing, digits included
          detail: An 8-bit ST (0x9C) or a reply over the 64-byte cap aborts and leaks 11;rgb:... as runes, and in a sitting those digits are answers. 0x9C can never occur inside a legal payload, so accepting it as a terminator is safe. Otherwise, narrow the spec's claim that no reply format can leak.
          family: reply-swallow-completeness
          round: 1
        - id: PQ-4
          severity: Minor
          title: 'The plan restates the diff: full implementations, test cases listed in prose, call sites by line number'
          detail: The gate asks for named functions plus one strategy line per risky function. It does not block this plan, which can be carried out and whose facts were checked; compress in future plans rather than rewriting this one.
          family: plan-restates-code
          round: 1
        - id: PQ-5
          severity: Minor
          title: schemeHolder.Store lets any code in package main replace the state without a named transition
          detail: 'ARCH-ORDER structural enforcement: expose transition methods (detect, choose, forget, each reporting whether the painted shade changed) and have applyScheme and tests use them, so the single-writer rule has one entry point.'
          family: state-writes-bypass-transitions
          round: 1
        - id: PQ-6
          severity: Minor
          title: The only check against real terminals is one manual pass at Task 18, with no record location or re-run trigger
          detail: ARCH-MOCK wants a stated cadence. Record the terminal/appearance matrix (atlas or Log) and say when it is re-run, e.g. when a terminal is added or a detection bug is reported.
          family: live-conformance-cadence
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-17T21:37:36-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: ClearScheme checks Lstat and IsDir before removing the directory. Task 7 adds a symlinked-directory test and a mutation that drops the guard.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: A zero-length io.Pipe write is the barrier. io/pipe.go:85 delivers it, and readInput reads again only after decoding the previous chunk (selection_input.go:160-225).
          round: 2
        - id: PQ-3
          disposition: addressed
          note: 0x9C is now a terminator. The decodeOSC comment names the over-64-byte case; the Spec sentence should be corrected in the Log at M3.
          round: 2
        - id: PQ-4
          disposition: not-addressed
          note: Left as is on the finding's own advice not to rewrite this plan. Carried to the close review as a Minor.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: choose, forget and detect are the only callers of the shared set. applyScheme and the loop handlers use them.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Task 18 records a dated terminal and appearance table in the atlas, with named triggers for re-running it.
          round: 2
      blocked: false
content_hash: aa4c5615dcc7111c89beb6e086ab2d2e52a251841608bc30f4bbe3ceaab0f9f7
---

# Gate ledger — tools#70 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-17T21:33:51-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `cleanup-removes-only-owned-residue` ClearScheme's os.Remove(dir) deletes a symlinked config dir even when its target holds other files
  The plan's comment says os.Remove(dir) fails harmlessly when something else lives there. That is false for a symlink: os.Remove unlinks it and returns nil (checked here: a link to a non-empty dir was deleted and the target's file survived). A stow or chezmoi-managed ~/.config/define loses its link on /scheme auto. Remove only a real empty directory (syscall.Rmdir gives ENOTDIR on a symlink, or Lstat and require IsDir), and add a store test with a symlinked dir.
- **PQ-2** [Minor] `test-barrier-orders-observation` Task 16 test 5 (dropped reply is silent) asserts no notice while y's own drop notice may already be posted
  The y write returning only proves the reply chunk was handled. readInput then drops y and posts "input full" at the same time as the no-notice assertion, so the test can fail when the code is correct. Use a zero-length io.Pipe write as the barrier (it is delivered as a Read), assert, then write y and wait for the notice.
- **PQ-3** [Minor] `reply-swallow-completeness` An aborted swallow after ESC ] 11 ; re-decodes the rest of the reply as typing, digits included
  An 8-bit ST (0x9C) or a reply over the 64-byte cap aborts and leaks 11;rgb:... as runes, and in a sitting those digits are answers. 0x9C can never occur inside a legal payload, so accepting it as a terminator is safe. Otherwise, narrow the spec's claim that no reply format can leak.
- **PQ-4** [Minor] `plan-restates-code` The plan restates the diff: full implementations, test cases listed in prose, call sites by line number
  The gate asks for named functions plus one strategy line per risky function. It does not block this plan, which can be carried out and whose facts were checked; compress in future plans rather than rewriting this one.
- **PQ-5** [Minor] `state-writes-bypass-transitions` schemeHolder.Store lets any code in package main replace the state without a named transition
  ARCH-ORDER structural enforcement: expose transition methods (detect, choose, forget, each reporting whether the painted shade changed) and have applyScheme and tests use them, so the single-writer rule has one entry point.
- **PQ-6** [Minor] `live-conformance-cadence` The only check against real terminals is one manual pass at Task 18, with no record location or re-run trigger
  ARCH-MOCK wants a stated cadence. Record the terminal/appearance matrix (atlas or Log) and say when it is re-run, e.g. when a terminal is added or a detection bug is reported.

## Round 2 — 2026-09-17T21:37:36-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — ClearScheme checks Lstat and IsDir before removing the directory. Task 7 adds a symlinked-directory test and a mutation that drops the guard.
- PQ-2 — addressed — A zero-length io.Pipe write is the barrier. io/pipe.go:85 delivers it, and readInput reads again only after decoding the previous chunk (selection_input.go:160-225).
- PQ-3 — addressed — 0x9C is now a terminator. The decodeOSC comment names the over-64-byte case; the Spec sentence should be corrected in the Log at M3.
- PQ-4 — not-addressed — Left as is on the finding's own advice not to rewrite this plan. Carried to the close review as a Minor.
- PQ-5 — addressed — choose, forget and detect are the only callers of the shared set. applyScheme and the loop handlers use them.
- PQ-6 — addressed — Task 18 records a dated terminal and appearance table in the atlas, with named triggers for re-running it.

## Open findings

- **PQ-4** [Minor] `plan-restates-code` The plan restates the diff: full implementations, test cases listed in prose, call sites by line number

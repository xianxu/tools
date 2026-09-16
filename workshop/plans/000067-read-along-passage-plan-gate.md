---
gate: plan-quality
issue: 67
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-16T14:10:25-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: The paste scanner's drain state is unreachable through the decodeKey hook Task 1.2 specifies
          detail: |-
            Task 1.2 Step 3 hooks the scanner on a 0x1b byte only, but pasteScanner.draining is
            entered when the buffer head is ordinary text mid-paste, so decodeKey never consults it
            again and the rest of an oversize paste arrives as KeyRune — the exact failure the drain
            exists to prevent. All three Task 1.2 tests start at an ESC, so nothing catches it. State
            the rule (the scanner takes every byte while draining, 0x1b otherwise) and add one test
            driving an oversize paste through decodeKey across two reads.
          family: paste-scanner-hook-contract
          round: 1
        - id: PQ-2
          severity: Important
          title: The paste boundary's degenerate-input class is unhandled — no closer, embedded closer, escapes in the body
          detail: |-
            ARCH-SECURE names the prompt and store boundaries and stops at the frame. With no closer
            ever, scan returns 0 forever so readInput (selection_input.go:174) never advances buf and
            the editor goes deaf; draining has no exit but a closer. An embedded ESC[201~ in the
            payload ends the paste early and delivers the remainder as live keys including \r. Escape
            sequences in the body reach the footer, which passes producer SGR through by construction
            (clipVisible; selection_frame.go:236-245). Name where the body becomes a typed value, and
            make the guard a fuzz target driving ONE scanner across calls — the plan's fresh-decoder-
            per-iteration rule leaves the stateful half unfuzzed.
          family: paste-input-untrusted
          round: 1
        - id: PQ-3
          severity: Important
          title: The passage line / footer entry / frame row mapping is unstated and the plan assumes both answers
          detail: |-
            Task 2.1's wrapped-row test (col+offset*cols) implies logical-line footer entries wrapped
            by the screen, matching FooterRowAt (screen.go:247). Task 2.2's phraseGap note assumes the
            opposite: phraseGap rejects a gap containing a newline (highlight.go:95-105), so under
            logical-line entries it does not guard a display wrap at all and the claim is wrong. The
            inverse of wordAtCell — mark byte offsets to per-frame-row cellRanges for the widened
            highlightRow — is named nowhere and every Task 3.2/3.3 test uses a single-row frame.
          family: passage-coordinate-mapping
          round: 1
        - id: PQ-4
          severity: Important
          title: The ask's non-success outcomes are unstated for mark-clearing and deck admission
          detail: |-
            runAsk (ask.go:146-221) distinguishes no-seam-wired (returns before sending), Ctrl-C
            mid-stream, ErrUnavailable after sending, ErrTruncated, and a default error exit. The plan
            states only that marks clear after the ask. Say which outcomes clear the mark set and which
            run CaptureMarked: a Ctrl-C that silently empties the marks, or an unconfigured model that
            turns words green, is uncertainty collapsed into success. askScoped (ask.go:56) is the seam.
          family: external-outcome-enumeration
          round: 1
        - id: PQ-5
          severity: Minor
          title: The parseREPLLine call-site enumeration omits repl_test.go, which is also the better home for the new rows
          detail: |-
            repl_test.go:58 calls parseREPLLine(tc.line, tc.hasCurrent) and is not in Task 4.2's list,
            so the signature change breaks the test build. It is also the table that field-compares
            note, where the mark-me nudge row belongs; TestConsoleDecisionTable (route_test.go:16)
            reduces to one of four outcome strings through the real dictionary.
          family: callsite-enumeration-incomplete
          round: 1
        - id: PQ-6
          severity: Minor
          title: The issue's M3 row still promises the generalised actionability guard that Chunk 3 removed
          detail: |-
            Chunk 3 correctly declines a new RegionKind, so TestEveryRegionKindIsActionable
            (editorloop_test.go:953) is untouched. The issue's Plan M3 checkbox still lists it, which a
            milestone-close reviewer will read as unfinished work.
          family: issue-plan-stale-row
          round: 1
      blocked: true
---

# Gate ledger — tools#67 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-16T14:10:25-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `paste-scanner-hook-contract` The paste scanner's drain state is unreachable through the decodeKey hook Task 1.2 specifies
  Task 1.2 Step 3 hooks the scanner on a 0x1b byte only, but pasteScanner.draining is
  entered when the buffer head is ordinary text mid-paste, so decodeKey never consults it
  again and the rest of an oversize paste arrives as KeyRune — the exact failure the drain
  exists to prevent. All three Task 1.2 tests start at an ESC, so nothing catches it. State
  the rule (the scanner takes every byte while draining, 0x1b otherwise) and add one test
  driving an oversize paste through decodeKey across two reads.
- **PQ-2** [Important] `paste-input-untrusted` The paste boundary's degenerate-input class is unhandled — no closer, embedded closer, escapes in the body
  ARCH-SECURE names the prompt and store boundaries and stops at the frame. With no closer
  ever, scan returns 0 forever so readInput (selection_input.go:174) never advances buf and
  the editor goes deaf; draining has no exit but a closer. An embedded ESC[201~ in the
  payload ends the paste early and delivers the remainder as live keys including \r. Escape
  sequences in the body reach the footer, which passes producer SGR through by construction
  (clipVisible; selection_frame.go:236-245). Name where the body becomes a typed value, and
  make the guard a fuzz target driving ONE scanner across calls — the plan's fresh-decoder-
  per-iteration rule leaves the stateful half unfuzzed.
- **PQ-3** [Important] `passage-coordinate-mapping` The passage line / footer entry / frame row mapping is unstated and the plan assumes both answers
  Task 2.1's wrapped-row test (col+offset*cols) implies logical-line footer entries wrapped
  by the screen, matching FooterRowAt (screen.go:247). Task 2.2's phraseGap note assumes the
  opposite: phraseGap rejects a gap containing a newline (highlight.go:95-105), so under
  logical-line entries it does not guard a display wrap at all and the claim is wrong. The
  inverse of wordAtCell — mark byte offsets to per-frame-row cellRanges for the widened
  highlightRow — is named nowhere and every Task 3.2/3.3 test uses a single-row frame.
- **PQ-4** [Important] `external-outcome-enumeration` The ask's non-success outcomes are unstated for mark-clearing and deck admission
  runAsk (ask.go:146-221) distinguishes no-seam-wired (returns before sending), Ctrl-C
  mid-stream, ErrUnavailable after sending, ErrTruncated, and a default error exit. The plan
  states only that marks clear after the ask. Say which outcomes clear the mark set and which
  run CaptureMarked: a Ctrl-C that silently empties the marks, or an unconfigured model that
  turns words green, is uncertainty collapsed into success. askScoped (ask.go:56) is the seam.
- **PQ-5** [Minor] `callsite-enumeration-incomplete` The parseREPLLine call-site enumeration omits repl_test.go, which is also the better home for the new rows
  repl_test.go:58 calls parseREPLLine(tc.line, tc.hasCurrent) and is not in Task 4.2's list,
  so the signature change breaks the test build. It is also the table that field-compares
  note, where the mark-me nudge row belongs; TestConsoleDecisionTable (route_test.go:16)
  reduces to one of four outcome strings through the real dictionary.
- **PQ-6** [Minor] `issue-plan-stale-row` The issue's M3 row still promises the generalised actionability guard that Chunk 3 removed
  Chunk 3 correctly declines a new RegionKind, so TestEveryRegionKindIsActionable
  (editorloop_test.go:953) is untouched. The issue's Plan M3 checkbox still lists it, which a
  milestone-close reviewer will read as unfinished work.

## Open findings

- **PQ-1** [Important] `paste-scanner-hook-contract` The paste scanner's drain state is unreachable through the decodeKey hook Task 1.2 specifies
- **PQ-2** [Important] `paste-input-untrusted` The paste boundary's degenerate-input class is unhandled — no closer, embedded closer, escapes in the body
- **PQ-3** [Important] `passage-coordinate-mapping` The passage line / footer entry / frame row mapping is unstated and the plan assumes both answers
- **PQ-4** [Important] `external-outcome-enumeration` The ask's non-success outcomes are unstated for mark-clearing and deck admission
- **PQ-5** [Minor] `callsite-enumeration-incomplete` The parseREPLLine call-site enumeration omits repl_test.go, which is also the better home for the new rows
- **PQ-6** [Minor] `issue-plan-stale-row` The issue's M3 row still promises the generalised actionability guard that Chunk 3 removed
